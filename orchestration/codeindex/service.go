package codeindex

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	datastore "ui_console/data/codeindex"
	domain "ui_console/domain/codeindex"
	"ui_console/domain/llm_context"
	"ui_console/shared/controlseal"
)

var splitTokenRe = regexp.MustCompile(`[A-Za-z0-9_]+`)

type BuildContextResult struct {
	Query    string                      `json:"query"`
	Sections []domain.ContextSection     `json:"sections"`
	Payload  *llm_context.ContextPayload `json:"payload"`
}

type Service struct {
	workspaceRoot string
	store         *datastore.Store

	mu       sync.RWMutex
	snapshot domain.Snapshot
}

func NewService(workspaceRoot, projectRoot string) *Service {
	return &Service{
		workspaceRoot: workspaceRoot,
		store:         datastore.NewStore(projectRoot),
	}
}

func (s *Service) Rebuild() (domain.RebuildResult, error) {
	outcome, err := s.rebuildSnapshot()
	if err != nil {
		return domain.RebuildResult{}, err
	}
	if err := s.store.Save(outcome.Snapshot); err != nil {
		return domain.RebuildResult{}, err
	}
	s.mu.Lock()
	s.snapshot = outcome.Snapshot
	s.mu.Unlock()
	return outcome.Result, nil
}

func (s *Service) Search(query string, opts domain.QueryOptions) ([]domain.Match, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("codeindex: empty query")
	}
	snapshot, err := s.ensureSnapshot()
	if err != nil {
		return nil, err
	}
	opts = normalizeOptions(opts)
	matches := scoreMatches(snapshot, query)
	if opts.IncludeRelated {
		matches = appendRelated(snapshot, matches, opts.Limit)
	}
	sortMatches(matches)
	if len(matches) > opts.Limit {
		matches = matches[:opts.Limit]
	}
	return matches, nil
}

func (s *Service) BuildContext(query string, opts domain.QueryOptions, isHighImpact bool) (*BuildContextResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("codeindex: empty query")
	}
	snapshot, err := s.ensureSnapshot()
	if err != nil {
		return nil, err
	}
	opts = normalizeOptions(opts)
	matches := scoreMatches(snapshot, query)
	if opts.IncludeRelated {
		matches = appendRelated(snapshot, matches, opts.Limit)
	}
	sortMatches(matches)
	if len(matches) > opts.Limit {
		matches = matches[:opts.Limit]
	}

	var (
		sections []domain.ContextSection
		blocks   []llm_context.ContentBlock
		sources  []llm_context.SourceToken
	)
	for i, match := range matches {
		content, truncated, err := s.readSectionContent(match.Section, opts)
		if err != nil {
			return nil, err
		}
		ctxSection := domain.ContextSection{
			Section:   match.Section,
			Score:     match.Score,
			Tags:      match.Tags,
			Reasons:   match.Reasons,
			Related:   match.Related,
			Content:   content,
			Truncated: truncated,
		}
		sections = append(sections, ctxSection)

		rendered := renderContextSection(ctxSection)
		safe := controlseal.SanitizeForLLM(controlseal.SourceDocument, rendered).LLMText
		blocks = append(blocks, llm_context.ContentBlock{
			Source:  fmt.Sprintf("code:%s#L%d-L%d", match.Section.FilePath, match.Section.StartLine, match.Section.EndLine),
			Content: safe,
			Role:    "reference",
		})
		sources = append(sources, llm_context.SourceToken{
			Hostname: fmt.Sprintf("local-code:%s", filepath.Base(match.Section.FilePath)),
			Rank:     len(matches) - i,
			AuthOK:   true,
		})
	}
	payload, err := llm_context.BuildContextPayload(blocks, sources, isHighImpact)
	if err != nil {
		return nil, err
	}
	return &BuildContextResult{
		Query:    query,
		Sections: sections,
		Payload:  payload,
	}, nil
}

type rebuildOutcome struct {
	Snapshot domain.Snapshot
	Result   domain.RebuildResult
}

func (s *Service) ensureSnapshot() (domain.Snapshot, error) {
	s.mu.RLock()
	if len(s.snapshot.Sections) > 0 || len(s.snapshot.Files) > 0 {
		snapshot := s.snapshot
		s.mu.RUnlock()
		return snapshot, nil
	}
	s.mu.RUnlock()

	loaded, err := s.store.Load()
	if err != nil {
		return domain.Snapshot{}, err
	}
	if len(loaded.Sections) == 0 && len(loaded.Files) == 0 {
		outcome, err := s.rebuildSnapshot()
		if err != nil {
			return domain.Snapshot{}, err
		}
		loaded = outcome.Snapshot
		if err := s.store.Save(loaded); err != nil {
			return domain.Snapshot{}, err
		}
	}
	s.mu.Lock()
	s.snapshot = loaded
	s.mu.Unlock()
	return loaded, nil
}

func (s *Service) rebuildSnapshot() (rebuildOutcome, error) {
	previous, err := s.store.Load()
	if err != nil {
		return rebuildOutcome{}, err
	}
	fileRecords := map[string]domain.FileRecord{}
	sectionPathByID := map[string]string{}
	sectionsByFile := map[string][]domain.Section{}
	tagsByFile := map[string][]domain.Tag{}
	edgesByFile := map[string][]domain.Edge{}
	for _, record := range previous.Files {
		fileRecords[record.FilePath] = record
	}
	for _, section := range previous.Sections {
		sectionsByFile[section.FilePath] = append(sectionsByFile[section.FilePath], section)
		sectionPathByID[section.ID] = section.FilePath
	}
	for _, tag := range previous.Tags {
		if path := sectionPathByID[tag.SectionID]; path != "" {
			tagsByFile[path] = append(tagsByFile[path], tag)
		}
	}
	for _, edge := range previous.Edges {
		if path := sectionPathByID[edge.FromSectionID]; path != "" {
			edgesByFile[path] = append(edgesByFile[path], edge)
		}
	}

	paths, err := collectGoFiles(s.workspaceRoot)
	if err != nil {
		return rebuildOutcome{}, err
	}
	seen := make(map[string]struct{}, len(paths))
	var (
		nextSections []domain.Section
		nextTags     []domain.Tag
		nextEdges    []domain.Edge
		nextFiles    []domain.FileRecord
		result       domain.RebuildResult
	)
	now := time.Now().UTC()
	for _, relPath := range paths {
		seen[relPath] = struct{}{}
		fullPath := filepath.Join(s.workspaceRoot, filepath.FromSlash(relPath))
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return rebuildOutcome{}, err
		}
		sum := checksum(content)
		if prev, ok := fileRecords[relPath]; ok && prev.SHA256 == sum && prev.ParseError == "" {
			nextFiles = append(nextFiles, prev)
			nextSections = append(nextSections, sectionsByFile[relPath]...)
			nextTags = append(nextTags, tagsByFile[relPath]...)
			nextEdges = append(nextEdges, edgesByFile[relPath]...)
			result.ReusedFiles++
			continue
		}
		fileSections, fileTags, fileEdges, record := indexGoFile(relPath, content, now)
		nextFiles = append(nextFiles, record)
		nextSections = append(nextSections, fileSections...)
		nextTags = append(nextTags, fileTags...)
		nextEdges = append(nextEdges, fileEdges...)
		result.IndexedFiles++
		if record.ParseError != "" {
			result.ParseErrors++
		}
	}
	for path := range fileRecords {
		if _, ok := seen[path]; !ok {
			result.DeletedFiles++
		}
	}

	sectionsByID := map[string]domain.Section{}
	for _, section := range nextSections {
		sectionsByID[section.ID] = section
	}
	for i := range nextEdges {
		if toID := resolveEdgeTarget(nextEdges[i].ToSymbol, sectionsByID); toID != "" {
			nextEdges[i].ToSectionID = toID
		}
	}

	snapshot := domain.Snapshot{
		Version:       1,
		WorkspaceRoot: s.workspaceRoot,
		IndexedAt:     now,
		Sections:      nextSections,
		Tags:          nextTags,
		Edges:         nextEdges,
		Files:         nextFiles,
	}
	result.UpdatedAt = now
	return rebuildOutcome{Snapshot: snapshot, Result: result}, nil
}

func collectGoFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() {
			switch name {
			case ".git", "node_modules", "vendor", "build", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func indexGoFile(relPath string, content []byte, now time.Time) ([]domain.Section, []domain.Tag, []domain.Edge, domain.FileRecord) {
	record := domain.FileRecord{
		FilePath:  relPath,
		SHA256:    checksum(content),
		IndexedAt: now,
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, content, parser.ParseComments)
	if err != nil {
		record.ParseError = err.Error()
		return nil, nil, nil, record
	}
	record.Package = file.Name.Name
	source := string(content)

	var (
		sections []domain.Section
		tags     []domain.Tag
		edges    []domain.Edge
	)
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			section := buildFuncSection(fset, relPath, file.Name.Name, source, d, now)
			funcTags := inferSectionTags(section, d, now)
			section.RiskLevel = inferRiskLevel(section.Summary, tagNames(funcTags))
			sections = append(sections, section)
			tags = append(tags, funcTags...)
			edges = append(edges, collectFuncEdges(section.ID, d, now)...)
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					section := buildTypeSection(fset, relPath, file.Name.Name, source, d, typeSpec, now)
					typeTags := inferTypeTags(section, now)
					section.RiskLevel = inferRiskLevel(section.Summary, tagNames(typeTags))
					sections = append(sections, section)
					tags = append(tags, typeTags...)
					edges = append(edges, collectTypeEdges(section.ID, typeSpec, now)...)
				}
			case token.CONST, token.VAR:
				section := buildValueSection(fset, relPath, file.Name.Name, source, d, now)
				valueTags := inferValueTags(section, now)
				section.RiskLevel = inferRiskLevel(section.Summary, tagNames(valueTags))
				sections = append(sections, section)
				tags = append(tags, valueTags...)
			}
		}
	}
	return sections, tags, edges, record
}

func buildFuncSection(fset *token.FileSet, relPath, pkg, source string, fn *ast.FuncDecl, now time.Time) domain.Section {
	start := fset.Position(fn.Pos()).Line
	end := fset.Position(fn.End()).Line
	doc, docStart, docEnd := docInfo(fset, fn.Doc)
	receiver := receiverName(fn)
	kind := domain.KindFunction
	if receiver != "" {
		kind = domain.KindMethod
	}
	return domain.Section{
		ID:           stableSectionID(relPath, receiver, fn.Name.Name, kind, start, end),
		FilePath:     relPath,
		Package:      pkg,
		SymbolName:   fn.Name.Name,
		Receiver:     receiver,
		Kind:         kind,
		StartLine:    start,
		EndLine:      end,
		DocStartLine: docStart,
		DocEndLine:   docEnd,
		Doc:          doc,
		Summary:      summarizeSymbol(fn.Name.Name, kind, pkg, doc),
		RiskLevel:    domain.RiskLow,
		Hash:         checksum([]byte(sectionSlice(source, start, end))),
		UpdatedAt:    now,
	}
}

func buildTypeSection(fset *token.FileSet, relPath, pkg, source string, decl *ast.GenDecl, spec *ast.TypeSpec, now time.Time) domain.Section {
	start := fset.Position(spec.Pos()).Line
	end := fset.Position(spec.End()).Line
	docGroup := spec.Doc
	if docGroup == nil {
		docGroup = decl.Doc
	}
	doc, docStart, docEnd := docInfo(fset, docGroup)
	kind := domain.KindType
	if spec.Assign.IsValid() {
		kind = domain.KindTypeAlias
	} else {
		switch spec.Type.(type) {
		case *ast.StructType:
			kind = domain.KindStruct
		case *ast.InterfaceType:
			kind = domain.KindInterface
		}
	}
	return domain.Section{
		ID:           stableSectionID(relPath, "", spec.Name.Name, kind, start, end),
		FilePath:     relPath,
		Package:      pkg,
		SymbolName:   spec.Name.Name,
		Kind:         kind,
		StartLine:    start,
		EndLine:      end,
		DocStartLine: docStart,
		DocEndLine:   docEnd,
		Doc:          doc,
		Summary:      summarizeSymbol(spec.Name.Name, kind, pkg, doc),
		RiskLevel:    domain.RiskLow,
		Hash:         checksum([]byte(sectionSlice(source, start, end))),
		UpdatedAt:    now,
	}
}

func buildValueSection(fset *token.FileSet, relPath, pkg, source string, decl *ast.GenDecl, now time.Time) domain.Section {
	start := fset.Position(decl.Pos()).Line
	end := fset.Position(decl.End()).Line
	doc, docStart, docEnd := docInfo(fset, decl.Doc)
	names := strings.Join(collectValueNames(decl), ", ")
	kind := domain.KindVar
	if decl.Tok == token.CONST {
		kind = domain.KindConst
	}
	return domain.Section{
		ID:           stableSectionID(relPath, "", names, kind, start, end),
		FilePath:     relPath,
		Package:      pkg,
		SymbolName:   names,
		Kind:         kind,
		StartLine:    start,
		EndLine:      end,
		DocStartLine: docStart,
		DocEndLine:   docEnd,
		Doc:          doc,
		Summary:      summarizeSymbol(names, kind, pkg, doc),
		RiskLevel:    domain.RiskLow,
		Hash:         checksum([]byte(sectionSlice(source, start, end))),
		UpdatedAt:    now,
	}
}

func inferSectionTags(section domain.Section, fn *ast.FuncDecl, now time.Time) []domain.Tag {
	tagSet := map[string]float64{}
	add := func(tag string, weight float64) {
		if current, ok := tagSet[tag]; !ok || weight > current {
			tagSet[tag] = weight
		}
	}
	lowerSymbol := strings.ToLower(section.SymbolName)
	lowerDoc := strings.ToLower(section.Doc + " " + section.Summary)
	if strings.Contains(lowerSymbol, "prompt") || strings.Contains(lowerDoc, "prompt") || strings.Contains(lowerSymbol, "context") {
		add("prompt_boundary", 1.0)
	}
	if strings.Contains(lowerSymbol, "llm") || strings.Contains(lowerDoc, "llm") || strings.Contains(lowerSymbol, "chat") {
		add("llm_egress", 0.9)
	}
	if strings.Contains(lowerSymbol, "handle") || strings.Contains(lowerSymbol, "command") {
		add("command_handler", 0.7)
	}
	if strings.Contains(lowerSymbol, "route") || strings.Contains(lowerSymbol, "register") {
		add("route_handler", 0.6)
	}
	if hasParamLike(fn, "user", "input", "query", "prompt", "text") {
		add("user_input", 0.6)
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			callName := renderCallName(node.Fun)
			switch {
			case strings.Contains(callName, "SanitizeForLLM"):
				add("sanitize_required", 1.0)
				add("untrusted_text", 0.8)
			case strings.Contains(callName, "BuildLLMContext"), strings.Contains(callName, "callAPI"), strings.Contains(callName, "SendAPIMessage"):
				add("llm_egress", 1.0)
			case strings.HasPrefix(callName, "http."), strings.Contains(callName, "Client.Do"), strings.Contains(callName, ".Get"), strings.Contains(callName, ".Post"), strings.Contains(callName, "urlsafe.NewSafeClient"):
				add("network_call", 0.8)
			case strings.Contains(callName, "os.WriteFile"), strings.Contains(callName, "os.Remove"), strings.Contains(callName, "os.Rename"), strings.Contains(callName, "os.MkdirAll"), strings.Contains(callName, "OpenFile"):
				add("filesystem_write", 0.8)
			case strings.Contains(callName, "os.ReadFile"), strings.Contains(callName, "os.Open"), strings.Contains(callName, "os.Stat"), strings.Contains(callName, "filepath.Walk"), strings.Contains(callName, "filepath.WalkDir"):
				add("filesystem_read", 0.6)
			case strings.Contains(callName, "exec.Command"), strings.Contains(callName, "executil.Command"):
				add("external_process", 1.0)
			}
		case *ast.Ident:
			name := strings.ToLower(node.Name)
			if name == "apikey" || strings.Contains(name, "secret") || strings.Contains(name, "token") || strings.Contains(name, "credential") {
				add("secret_touch", 0.8)
			}
		}
		return true
	})
	return buildTagList(section.ID, tagSet, now)
}

func inferTypeTags(section domain.Section, now time.Time) []domain.Tag {
	tagSet := map[string]float64{}
	lower := strings.ToLower(section.SymbolName + " " + section.Doc + " " + section.Summary)
	if strings.Contains(lower, "context") {
		tagSet["prompt_boundary"] = 0.5
	}
	if strings.Contains(lower, "credential") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") {
		tagSet["secret_touch"] = 0.7
	}
	return buildTagList(section.ID, tagSet, now)
}

func inferValueTags(section domain.Section, now time.Time) []domain.Tag {
	tagSet := map[string]float64{}
	lower := strings.ToLower(section.SymbolName + " " + section.Doc + " " + section.Summary)
	if strings.Contains(lower, "prompt") {
		tagSet["prompt_boundary"] = 0.5
	}
	if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "credential") {
		tagSet["secret_touch"] = 0.7
	}
	return buildTagList(section.ID, tagSet, now)
}

func collectFuncEdges(sectionID string, fn *ast.FuncDecl, now time.Time) []domain.Edge {
	if fn.Body == nil {
		return nil
	}
	var edges []domain.Edge
	seen := map[string]struct{}{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		target := lastSegment(renderCallName(call.Fun))
		if target == "" || isBuiltinType(target) {
			return true
		}
		if _, ok := seen[target]; ok {
			return true
		}
		seen[target] = struct{}{}
		edges = append(edges, domain.Edge{
			FromSectionID: sectionID,
			ToSymbol:      target,
			Kind:          domain.EdgeCalls,
			Weight:        1.0,
			UpdatedAt:     now,
		})
		return true
	})
	return edges
}

func collectTypeEdges(sectionID string, spec *ast.TypeSpec, now time.Time) []domain.Edge {
	var edges []domain.Edge
	seen := map[string]struct{}{}
	ast.Inspect(spec.Type, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok || ident.Name == "" || isBuiltinType(ident.Name) {
			return true
		}
		if _, ok := seen[ident.Name]; ok {
			return true
		}
		seen[ident.Name] = struct{}{}
		edges = append(edges, domain.Edge{
			FromSectionID: sectionID,
			ToSymbol:      ident.Name,
			Kind:          domain.EdgeUsesType,
			Weight:        0.6,
			UpdatedAt:     now,
		})
		return true
	})
	return edges
}

func scoreMatches(snapshot domain.Snapshot, query string) []domain.Match {
	queryLower := strings.ToLower(strings.TrimSpace(query))
	queryTokens := tokenize(queryLower)
	tagsBySection := map[string][]string{}
	for _, tag := range snapshot.Tags {
		tagsBySection[tag.SectionID] = append(tagsBySection[tag.SectionID], tag.Tag)
	}
	var matches []domain.Match
	for _, section := range snapshot.Sections {
		score, reasons := scoreSection(section, tagsBySection[section.ID], queryLower, queryTokens)
		if score == 0 {
			continue
		}
		matches = append(matches, domain.Match{
			Section: section,
			Score:   score,
			Tags:    dedupStrings(tagsBySection[section.ID]),
			Reasons: reasons,
		})
	}
	return matches
}

func scoreSection(section domain.Section, tags []string, query string, queryTokens []string) (int, []string) {
	var (
		score   int
		reasons []string
	)
	symbolLower := strings.ToLower(section.SymbolName)
	pathLower := strings.ToLower(section.FilePath)
	metaLower := strings.ToLower(section.Doc + " " + section.Summary + " " + section.Package)

	if symbolLower == query {
		score += 100
		reasons = append(reasons, "exact_symbol")
	}
	for _, tag := range tags {
		if strings.EqualFold(tag, query) {
			score += 80
			reasons = append(reasons, "exact_tag")
			break
		}
	}
	if symbolLower != query && strings.Contains(symbolLower, query) {
		score += 70
		reasons = append(reasons, "partial_symbol")
	}
	for _, token := range queryTokens {
		if strings.Contains(metaLower, token) {
			score += 18
			reasons = append(reasons, "summary_doc")
		}
		if strings.Contains(pathLower, token) {
			score += 10
			reasons = append(reasons, "path_package")
		}
		for _, tag := range tags {
			if strings.EqualFold(tag, token) {
				score += 24
				reasons = append(reasons, "tag_token")
				break
			}
		}
	}
	return score, dedupStrings(reasons)
}

func appendRelated(snapshot domain.Snapshot, matches []domain.Match, limit int) []domain.Match {
	if len(matches) == 0 {
		return matches
	}
	sortMatches(matches)
	if limit > len(matches) {
		limit = len(matches)
	}
	base := matches[:limit]
	sectionsByID := map[string]domain.Section{}
	tagsBySection := map[string][]string{}
	for _, section := range snapshot.Sections {
		sectionsByID[section.ID] = section
	}
	for _, tag := range snapshot.Tags {
		tagsBySection[tag.SectionID] = append(tagsBySection[tag.SectionID], tag.Tag)
	}
	existing := map[string]struct{}{}
	for _, match := range matches {
		existing[match.Section.ID] = struct{}{}
	}
	for _, match := range base {
		for _, edge := range snapshot.Edges {
			if edge.FromSectionID != match.Section.ID {
				continue
			}
			targetID := edge.ToSectionID
			if targetID == "" {
				targetID = resolveEdgeTarget(edge.ToSymbol, sectionsByID)
			}
			if targetID == "" {
				continue
			}
			if _, ok := existing[targetID]; ok {
				continue
			}
			target := sectionsByID[targetID]
			matches = append(matches, domain.Match{
				Section: target,
				Score:   max(match.Score/3, 20),
				Tags:    dedupStrings(tagsBySection[targetID]),
				Reasons: []string{"related_" + string(edge.Kind)},
				Related: true,
			})
			existing[targetID] = struct{}{}
		}
	}
	return matches
}

func (s *Service) readSectionContent(section domain.Section, opts domain.QueryOptions) (string, bool, error) {
	fullPath := filepath.Join(s.workspaceRoot, filepath.FromSlash(section.FilePath))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", false, err
	}
	lines := strings.Split(string(data), "\n")
	start := section.StartLine
	if section.DocStartLine > 0 {
		start = section.DocStartLine
	}
	start = max(1, start-opts.ContextBefore)
	end := min(len(lines), section.EndLine+opts.ContextAfter)
	selected := lines[start-1 : end]
	truncated := false
	if opts.MaxContextLines > 0 && len(selected) > opts.MaxContextLines {
		selected = selected[:opts.MaxContextLines]
		truncated = true
	}
	content := strings.Join(selected, "\n")
	if opts.MaxContextBytes > 0 && len(content) > opts.MaxContextBytes {
		content = truncateBytes(content, opts.MaxContextBytes)
		truncated = true
	}
	return content, truncated, nil
}

func renderContextSection(section domain.ContextSection) string {
	var b strings.Builder
	fmt.Fprintf(&b, "file=%s\n", section.Section.FilePath)
	fmt.Fprintf(&b, "package=%s\n", section.Section.Package)
	fmt.Fprintf(&b, "symbol=%s\n", section.Section.SymbolName)
	if section.Section.Receiver != "" {
		fmt.Fprintf(&b, "receiver=%s\n", section.Section.Receiver)
	}
	fmt.Fprintf(&b, "kind=%s\n", section.Section.Kind)
	fmt.Fprintf(&b, "lines=%d-%d\n", section.Section.StartLine, section.Section.EndLine)
	fmt.Fprintf(&b, "risk=%s\n", section.Section.RiskLevel)
	if len(section.Tags) > 0 {
		fmt.Fprintf(&b, "tags=%s\n", strings.Join(section.Tags, ", "))
	}
	if section.Section.Summary != "" {
		fmt.Fprintf(&b, "summary=%s\n", section.Section.Summary)
	}
	if len(section.Reasons) > 0 {
		fmt.Fprintf(&b, "reasons=%s\n", strings.Join(section.Reasons, ", "))
	}
	if section.Truncated {
		b.WriteString("truncated=true\n")
	}
	b.WriteString("\n")
	b.WriteString(section.Content)
	return b.String()
}

func normalizeOptions(opts domain.QueryOptions) domain.QueryOptions {
	if opts.Limit <= 0 {
		opts.Limit = 5
	}
	if opts.ContextBefore == 0 {
		opts.ContextBefore = 3
	}
	if opts.ContextAfter == 0 {
		opts.ContextAfter = 5
	}
	if opts.MaxContextLines <= 0 {
		opts.MaxContextLines = 140
	}
	if opts.MaxContextBytes <= 0 {
		opts.MaxContextBytes = 6 * 1024
	}
	return opts
}

func sortMatches(matches []domain.Match) {
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			if matches[i].Related != matches[j].Related {
				return !matches[i].Related
			}
			if matches[i].Section.FilePath == matches[j].Section.FilePath {
				return matches[i].Section.StartLine < matches[j].Section.StartLine
			}
			return matches[i].Section.FilePath < matches[j].Section.FilePath
		}
		return matches[i].Score > matches[j].Score
	})
}

func docInfo(fset *token.FileSet, group *ast.CommentGroup) (string, int, int) {
	if group == nil {
		return "", 0, 0
	}
	return strings.TrimSpace(group.Text()), fset.Position(group.Pos()).Line, fset.Position(group.End()).Line
}

func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	switch expr := fn.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		if ident, ok := expr.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func summarizeSymbol(symbol string, kind domain.SectionKind, pkg, doc string) string {
	if doc != "" {
		line := strings.TrimSpace(strings.Split(doc, "\n")[0])
		if line != "" {
			return line
		}
	}
	phrase := strings.Join(splitCamel(symbol), " ")
	if phrase == "" {
		phrase = symbol
	}
	switch kind {
	case domain.KindFunction:
		return fmt.Sprintf("Function %s in package %s.", phrase, pkg)
	case domain.KindMethod:
		return fmt.Sprintf("Method %s in package %s.", phrase, pkg)
	case domain.KindStruct:
		return fmt.Sprintf("Struct %s in package %s.", phrase, pkg)
	case domain.KindInterface:
		return fmt.Sprintf("Interface %s in package %s.", phrase, pkg)
	case domain.KindConst:
		return fmt.Sprintf("Const block %s in package %s.", phrase, pkg)
	case domain.KindVar:
		return fmt.Sprintf("Var block %s in package %s.", phrase, pkg)
	default:
		return fmt.Sprintf("%s %s in package %s.", strings.Title(strings.ReplaceAll(string(kind), "_", " ")), phrase, pkg)
	}
}

func buildTagList(sectionID string, tags map[string]float64, now time.Time) []domain.Tag {
	keys := make([]string, 0, len(tags))
	for tag := range tags {
		keys = append(keys, tag)
	}
	sort.Strings(keys)
	out := make([]domain.Tag, 0, len(keys))
	for _, tag := range keys {
		out = append(out, domain.Tag{
			SectionID: sectionID,
			Tag:       tag,
			Weight:    tags[tag],
			Source:    domain.TagSourceSystem,
			UpdatedAt: now,
		})
	}
	return out
}

func collectValueNames(decl *ast.GenDecl) []string {
	var names []string
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range valueSpec.Names {
			names = append(names, name.Name)
		}
	}
	return names
}

func hasParamLike(fn *ast.FuncDecl, terms ...string) bool {
	if fn.Type.Params == nil {
		return false
	}
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			lower := strings.ToLower(name.Name)
			for _, term := range terms {
				if strings.Contains(lower, term) {
					return true
				}
			}
		}
	}
	return false
}

func renderCallName(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		left := renderCallName(node.X)
		if left == "" {
			return node.Sel.Name
		}
		return left + "." + node.Sel.Name
	case *ast.IndexExpr:
		return renderCallName(node.X)
	case *ast.IndexListExpr:
		return renderCallName(node.X)
	case *ast.StarExpr:
		return renderCallName(node.X)
	default:
		return ""
	}
}

func stableSectionID(relPath, receiver, symbol string, kind domain.SectionKind, start, end int) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%d|%d", relPath, receiver, symbol, kind, start, end)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:8])
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sectionSlice(source string, start, end int) string {
	lines := strings.Split(source, "\n")
	if start < 1 || start > len(lines) {
		return ""
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start-1:end], "\n")
}

func splitCamel(value string) []string {
	var (
		out     []string
		current []rune
	)
	for i, r := range value {
		if r == '_' || r == '-' || r == ' ' || r == ',' {
			if len(current) > 0 {
				out = append(out, strings.ToLower(string(current)))
				current = current[:0]
			}
			continue
		}
		if i > 0 && r >= 'A' && r <= 'Z' && len(current) > 0 {
			out = append(out, strings.ToLower(string(current)))
			current = current[:0]
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		out = append(out, strings.ToLower(string(current)))
	}
	return out
}

func inferRiskLevel(summary string, tags []string) domain.RiskLevel {
	lower := strings.ToLower(summary + " " + strings.Join(tags, " "))
	switch {
	case strings.Contains(lower, "secret_touch"), strings.Contains(lower, "credential"), strings.Contains(lower, "external_process"), strings.Contains(lower, "token"):
		return domain.RiskHigh
	case strings.Contains(lower, "llm"), strings.Contains(lower, "prompt"), strings.Contains(lower, "network"), strings.Contains(lower, "filesystem"):
		return domain.RiskMedium
	default:
		return domain.RiskLow
	}
}

func tagNames(tags []domain.Tag) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, tag.Tag)
	}
	return out
}

func tokenize(value string) []string {
	return dedupStrings(splitTokenRe.FindAllString(strings.ToLower(value), -1))
}

func dedupStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func resolveEdgeTarget(symbol string, sections map[string]domain.Section) string {
	target := lastSegment(symbol)
	for _, section := range sections {
		if section.SymbolName == target {
			return section.ID
		}
	}
	return ""
}

func lastSegment(value string) string {
	if idx := strings.LastIndex(value, "."); idx >= 0 {
		return value[idx+1:]
	}
	return value
}

func isBuiltinType(name string) bool {
	switch name {
	case "string", "bool", "error", "int", "int64", "int32", "int16", "int8", "uint", "uint64", "uint32", "uint16", "uint8", "byte", "rune", "float32", "float64", "complex64", "complex128", "map", "chan", "any":
		return true
	default:
		return false
	}
}

func truncateBytes(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	cut := maxBytes
	for cut > 0 && cut < len(value) && (value[cut]&0xC0) == 0x80 {
		cut--
	}
	return value[:cut]
}
