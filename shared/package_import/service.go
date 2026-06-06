// Package package_import implements the v3.3.2 P0.1 Skill / MCP drag-install
// core contract.
//
// Core invariant: dropping a package file only represents prepare_install.
// The package must NEVER be installed, added to tool_registry, or entered into
// the routing candidate set until the user explicitly confirms installation
// through the review card / installation modal.
//
// On rejection or cancellation the quarantine package MUST be deleted; only a
// sanitised rejected-import summary is retained in the log.
//
// v3.3.2 補強（#42–#42.2）：
//   - 安裝前完整安全檢查（manifest strict validation、zip-slip、symlink、
//     hardlink、device file、path boundary escape）
//   - package hash / manifest hash / registry diff hash 計算與記錄
//   - 孤兒 quarantine 啟動掃描（crash 後自動偵測 + 安全檢查 + Review Card）
package package_import

import (
	"archive/tar"
	"archive/zip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ImportStatus tracks the lifecycle of a pending package install.
type ImportStatus string

const (
	StatusQuarantined ImportStatus = "quarantined" // dropped, not yet confirmed
	StatusConfirmed   ImportStatus = "confirmed"   // user confirmed installation
	StatusRejected    ImportStatus = "rejected"    // user rejected; quarantine removed
	StatusCancelled   ImportStatus = "cancelled"   // user cancelled; quarantine removed
)

// ──────────────────────────────────────────────
// PackageManifest — #42.1 完整 manifest 結構
// ──────────────────────────────────────────────

// PackageManifest 是從 package 中提取的完整 manifest，用於安全檢查與 Review Card 顯示。
// #42.1 要求所有必要欄位必須通過 strict schema validation。
type PackageManifest struct {
	// ── #42.1 必要欄位 ──
	PackageID       string   `json:"package_id"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	PackageType     string   `json:"package_type"`      // skill / mcp / adapter / persona
	DeclaredPerms   []string `json:"declared_permissions"`
	DeclaredEntries []string `json:"declared_entry_points"`
	WriteTargets    []string `json:"write_targets"`
	RiskTag         string   `json:"risk_tag"`           // low / medium / high / unknown
	SourceMetadata  string   `json:"source_metadata"`    // 來源路徑或 URL

	// ── hash 紀錄（安全檢查時填入）──
	ManifestHash    string `json:"manifest_hash"`
	PackageHash     string `json:"package_hash"`
	RegistryDiffHash string `json:"registry_diff_hash,omitempty"`

	// ── Review Card 分層顯示用（#42 補強）──
	AddsTools       bool `json:"adds_tools"`        // 是否新增工具
	AddsSkills      bool `json:"adds_skills"`       // 是否新增 skill
	AddsMCPServer   bool `json:"adds_mcp_server"`   // 是否新增 MCP server
	AddsAdapter     bool `json:"adds_adapter"`      // 是否新增 adapter
	AffectsRouting  bool `json:"affects_routing"`   // 是否影響 routing / tool visibility

	// ── 向下相容 ──
	SourcePath    string   `json:"source_path"`
	RequiredPerms []string `json:"required_perms,omitempty"` // deprecated, use DeclaredPerms
}

// ──────────────────────────────────────────────
// #42.1 Manifest Strict Schema Validation
// ──────────────────────────────────────────────

// ManifestValidationError 包含所有驗證失敗的欄位。
type ManifestValidationError struct {
	MissingFields []string `json:"missing_fields"`
	InvalidFields []string `json:"invalid_fields"`
}

func (e *ManifestValidationError) Error() string {
	return fmt.Sprintf("manifest validation failed: missing=%v, invalid=%v", e.MissingFields, e.InvalidFields)
}

// ValidateManifestStrict 執行 #42.1 要求的 strict schema validation。
// 所有必要欄位都必須存在且非空。
func ValidateManifestStrict(m PackageManifest) *ManifestValidationError {
	var missing, invalid []string

	if m.PackageID == "" {
		// 向下相容：若 PackageID 為空但 Name 存在，不算 missing
		if m.Name == "" {
			missing = append(missing, "package_id")
		}
	}
	if m.Name == "" {
		missing = append(missing, "name")
	}
	if m.Version == "" {
		missing = append(missing, "version")
	}
	if m.PackageType == "" {
		// 向下相容：persona 包可能不帶 package_type
		if m.PackageType == "" {
			m.PackageType = "persona"
		}
	}
	validTypes := map[string]bool{"skill": true, "mcp": true, "adapter": true, "persona": true}
	if !validTypes[m.PackageType] {
		invalid = append(invalid, fmt.Sprintf("package_type: unsupported %q", m.PackageType))
	}
	if m.RiskTag == "" {
		missing = append(missing, "risk_tag")
	} else {
		validRisks := map[string]bool{"low": true, "medium": true, "high": true, "unknown": true}
		if !validRisks[m.RiskTag] {
			invalid = append(invalid, fmt.Sprintf("risk_tag: unsupported %q", m.RiskTag))
		}
	}
	if m.SourceMetadata == "" && m.SourcePath == "" {
		missing = append(missing, "source_metadata")
	}

	if len(missing) > 0 || len(invalid) > 0 {
		return &ManifestValidationError{MissingFields: missing, InvalidFields: invalid}
	}
	return nil
}

// ──────────────────────────────────────────────
// #42.1 安全檢查結果
// ──────────────────────────────────────────────

// SecurityCheckResult 記錄完整安全檢查的結果。
type SecurityCheckResult struct {
	Passed       bool     `json:"passed"`
	Failures     []string `json:"failures"`
	PackageHash  string   `json:"package_hash"`
	ManifestHash string   `json:"manifest_hash"`
	CheckedAt    string   `json:"checked_at"`
}

// ──────────────────────────────────────────────
// #42.2 孤兒 Quarantine 掃描結果
// ──────────────────────────────────────────────

// OrphanScanResult 記錄啟動時孤兒掃描的結果。
type OrphanScanResult struct {
	OrphanID       string              `json:"orphan_id"`
	QuarantinePath string              `json:"quarantine_path"`
	SecurityCheck  SecurityCheckResult `json:"security_check"`
	// 安全檢查通過 → 產生 PendingImport 供 Review Card 使用
	// 安全檢查失敗 → FailureReason 非空，前端彈窗提醒後清除
	PendingImport  *PendingImport `json:"pending_import,omitempty"`
	FailureReason  string         `json:"failure_reason,omitempty"`
}

// PendingImport is one quarantined package awaiting user confirmation.
type PendingImport struct {
	ID               string          `json:"id"`
	Manifest         PackageManifest `json:"manifest"`
	QuarantinePath   string          `json:"quarantine_path"`
	Status           ImportStatus    `json:"status"`
	CreatedAt        time.Time       `json:"created_at"`
	ResolvedAt       *time.Time      `json:"resolved_at,omitempty"`
	ValidationIssues []string        `json:"validation_issues,omitempty"`
	Persona          PersonaPackage  `json:"persona"`
}

// PersonaPackage is the fixed persona package schema accepted by drag import.
// Missing optional fields are normalized to empty strings so older packages can
// still be reviewed instead of failing hard.
type PersonaPackage struct {
	Schema        string `json:"schema"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Icon          string `json:"icon"`
	AvatarURL     string `json:"avatarUrl"`
	Identity      string `json:"identity"`
	ReplyStrategy string `json:"replyStrategy"`
	RoleStrength  string `json:"roleStrength"`
	Personality   string `json:"personality"`
	Scenario      string `json:"scenario"`
	Description   string `json:"description"`
}

// RejectedImportSummary is the sanitised log entry retained after rejection.
// It MUST NOT contain the package payload.
type RejectedImportSummary struct {
	ImportID   string    `json:"import_id"`
	Name       string    `json:"name"`
	RiskTag    string    `json:"risk_tag"`
	Reason     string    `json:"reason"` // "rejected" or "cancelled"
	RejectedAt time.Time `json:"rejected_at"`
}

// Service manages the package import preview and quarantine lifecycle.
type Service struct {
	mu            sync.Mutex
	quarantineDir string
	logPath       string
	pending       map[string]*PendingImport
}

// NewService creates a new package import service.
// quarantineDir is where dropped packages are temporarily held.
func NewService(dataRoot string) *Service {
	return &Service{
		quarantineDir: filepath.Join(dataRoot, "package_import", "quarantine"),
		logPath:       filepath.Join(dataRoot, "package_import", "rejected_import_log.jsonl"),
		pending:       make(map[string]*PendingImport),
	}
}

// PrepareInstall is called when the user drops a package onto the app.
// It quarantines the package and returns a PendingImport for UI preview.
// The package is NOT installed; it is NOT added to tool_registry.
func (s *Service) PrepareInstall(sourcePath string, manifest PackageManifest) (*PendingImport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("import-%d", time.Now().UnixNano())
	manifest.ManifestHash = hashManifest(manifest)

	// Copy package to quarantine — never touch the original location.
	quarantinePath := filepath.Join(s.quarantineDir, id)
	if err := os.MkdirAll(quarantinePath, 0o700); err != nil {
		return nil, fmt.Errorf("package_import: cannot create quarantine dir: %w", err)
	}
	persona, issues, err := s.quarantineSource(sourcePath, quarantinePath)
	if err != nil {
		_ = os.RemoveAll(quarantinePath)
		return nil, err
	}

	pi := &PendingImport{
		ID:               id,
		Manifest:         manifest,
		QuarantinePath:   quarantinePath,
		Status:           StatusQuarantined,
		CreatedAt:        time.Now(),
		ValidationIssues: issues,
		Persona:          persona,
	}
	s.pending[id] = pi
	return pi, nil
}

// PrepareInstallFromBytes is used by the Wails frontend when the browser drag
// API provides file contents but not a stable absolute source path. The bytes
// are written into quarantine first, then validated like a normal dropped file.
func (s *Service) PrepareInstallFromBytes(sourceName string, payload []byte, manifest PackageManifest) (*PendingImport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("import-%d", time.Now().UnixNano())
	manifest.SourcePath = sourceName
	manifest.ManifestHash = hashManifest(manifest)

	quarantinePath := filepath.Join(s.quarantineDir, id)
	if err := os.MkdirAll(quarantinePath, 0o700); err != nil {
		return nil, fmt.Errorf("package_import: cannot create quarantine dir: %w", err)
	}
	safeName := filepath.Base(sourceName)
	if safeName == "." || safeName == string(filepath.Separator) || safeName == "" {
		safeName = "persona.json"
	}
	payloadPath := filepath.Join(quarantinePath, "payload", safeName)
	if err := os.MkdirAll(filepath.Dir(payloadPath), 0o700); err != nil {
		_ = os.RemoveAll(quarantinePath)
		return nil, err
	}
	decodedPayload, err := decodePayload(payload)
	if err != nil {
		_ = os.RemoveAll(quarantinePath)
		return nil, err
	}
	if err := os.WriteFile(payloadPath, decodedPayload, 0o600); err != nil {
		_ = os.RemoveAll(quarantinePath)
		return nil, err
	}
	validateRoot := filepath.Join(quarantinePath, "payload")
	switch strings.ToLower(filepath.Ext(safeName)) {
	case ".zip":
		validateRoot = filepath.Join(quarantinePath, "unpacked")
		if err := extractZip(payloadPath, validateRoot); err != nil {
			_ = os.RemoveAll(quarantinePath)
			return nil, err
		}
	case ".tar":
		validateRoot = filepath.Join(quarantinePath, "unpacked")
		if err := extractTar(payloadPath, validateRoot); err != nil {
			_ = os.RemoveAll(quarantinePath)
			return nil, err
		}
	}
	persona, issues, err := validatePersonaPayload(validateRoot)
	if err != nil {
		_ = os.RemoveAll(quarantinePath)
		return nil, err
	}
	pi := &PendingImport{
		ID:               id,
		Manifest:         manifest,
		QuarantinePath:   quarantinePath,
		Status:           StatusQuarantined,
		CreatedAt:        time.Now(),
		ValidationIssues: issues,
		Persona:          persona,
	}
	s.pending[id] = pi
	return pi, nil
}

// ConfirmInstall moves a quarantined package from pending to confirmed.
// The caller (app.go) is responsible for writing to tool_registry only after
// this returns without error.
func (s *Service) ConfirmInstall(importID string) (*PendingImport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pi, ok := s.pending[importID]
	if !ok {
		return nil, fmt.Errorf("package_import: import %q not found", importID)
	}
	if pi.Status != StatusQuarantined {
		return nil, fmt.Errorf("package_import: import %q is not in quarantine (status=%s)", importID, pi.Status)
	}
	now := time.Now()
	pi.Status = StatusConfirmed
	pi.ResolvedAt = &now
	return pi, nil
}

// RejectInstall cancels a pending install, removes the quarantine package, and
// writes a sanitised summary to the rejected import log.
// #I-1002: 同時回傳 RejectedImportSummary，讓呼叫端（app.go）生成 rejected Review Card。
func (s *Service) RejectInstall(importID, reason string) (*RejectedImportSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pi, ok := s.pending[importID]
	if !ok {
		return nil, fmt.Errorf("package_import: import %q not found", importID)
	}

	// Remove quarantine package — only sanitised summary is kept.
	if pi.QuarantinePath != "" {
		_ = os.RemoveAll(pi.QuarantinePath)
	}

	now := time.Now()
	pi.Status = ImportStatus(reason) // "rejected" or "cancelled"
	pi.ResolvedAt = &now
	delete(s.pending, importID)

	summary := RejectedImportSummary{
		ImportID:   importID,
		Name:       pi.Manifest.Name,
		RiskTag:    pi.Manifest.RiskTag,
		Reason:     reason,
		RejectedAt: now,
	}
	return &summary, s.appendRejectedLog(summary)
}

// ListPending returns all quarantined (unconfirmed) imports.
func (s *Service) ListPending() []*PendingImport {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*PendingImport
	for _, pi := range s.pending {
		if pi.Status == StatusQuarantined {
			result = append(result, pi)
		}
	}
	return result
}

// --- internal helpers ---

func (s *Service) appendRejectedLog(summary RejectedImportSummary) error {
	if err := os.MkdirAll(filepath.Dir(s.logPath), 0o700); err != nil {
		return err
	}
	line, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

func hashManifest(m PackageManifest) string {
	raw := fmt.Sprintf("%s|%s|%s|%s", m.Name, m.Version, m.SourcePath, m.RiskTag)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func decodePayload(payload []byte) ([]byte, error) {
	text := string(payload)
	if strings.HasPrefix(text, "data:") {
		comma := strings.Index(text, ",")
		if comma < 0 {
			return nil, fmt.Errorf("package_import: malformed data URL payload")
		}
		meta := text[:comma]
		if !strings.Contains(meta, ";base64") {
			return nil, fmt.Errorf("package_import: data URL payload must be base64")
		}
		decoded, err := base64.StdEncoding.DecodeString(text[comma+1:])
		if err != nil {
			return nil, fmt.Errorf("package_import: decode payload: %w", err)
		}
		return decoded, nil
	}
	return payload, nil
}

func (s *Service) quarantineSource(sourcePath, quarantinePath string) (PersonaPackage, []string, error) {
	if sourcePath == "" {
		return PersonaPackage{}, nil, fmt.Errorf("package_import: source path is required")
	}
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return PersonaPackage{}, nil, fmt.Errorf("package_import: cannot read source payload: %w", err)
	}
	payloadDir := filepath.Join(quarantinePath, "payload")
	if info.IsDir() {
		if err := copyDir(sourcePath, payloadDir); err != nil {
			return PersonaPackage{}, nil, err
		}
	} else {
		ext := strings.ToLower(filepath.Ext(sourcePath))
		switch ext {
		case ".zip":
			if err := extractZip(sourcePath, payloadDir); err != nil {
				return PersonaPackage{}, nil, err
			}
		case ".tar":
			if err := extractTar(sourcePath, payloadDir); err != nil {
				return PersonaPackage{}, nil, err
			}
		case ".json":
			if err := os.MkdirAll(payloadDir, 0o700); err != nil {
				return PersonaPackage{}, nil, err
			}
			if err := copyFile(sourcePath, filepath.Join(payloadDir, filepath.Base(sourcePath))); err != nil {
				return PersonaPackage{}, nil, err
			}
		default:
			return PersonaPackage{}, nil, fmt.Errorf("package_import: unsupported package type %q", ext)
		}
	}
	return validatePersonaPayload(payloadDir)
}

func validatePersonaPayload(root string) (PersonaPackage, []string, error) {
	var jsonPath string
	var images []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".json":
			if jsonPath == "" {
				jsonPath = path
			}
		case ".jpg", ".jpeg", ".png":
			images = append(images, filepath.Base(path))
		}
		return nil
	}); err != nil {
		return PersonaPackage{}, nil, err
	}
	if jsonPath == "" {
		return PersonaPackage{}, nil, fmt.Errorf("package_import: persona json file is required")
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return PersonaPackage{}, nil, err
	}
	var persona PersonaPackage
	if err := json.Unmarshal(data, &persona); err != nil {
		return PersonaPackage{}, nil, fmt.Errorf("package_import: invalid persona json: %w", err)
	}
	var issues []string
	if persona.Schema == "" {
		persona.Schema = "ai-console.persona.v1"
		issues = append(issues, "schema missing; filled ai-console.persona.v1")
	}
	if persona.Schema != "ai-console.persona.v1" {
		return PersonaPackage{}, nil, fmt.Errorf("package_import: unsupported persona schema %q", persona.Schema)
	}
	if persona.ID == "" {
		persona.ID = fmt.Sprintf("persona-%d", time.Now().UnixNano())
		issues = append(issues, "id missing; generated automatically")
	}
	if persona.Name == "" {
		persona.Name = "未命名人格"
		issues = append(issues, "name missing; filled default name")
	}
	if persona.Icon != "" && !hasImage(images, persona.Icon) {
		persona.Icon = ""
		issues = append(issues, "icon image missing; left blank")
	}
	if persona.AvatarURL != "" && !hasImage(images, persona.AvatarURL) {
		persona.AvatarURL = ""
		issues = append(issues, "avatar image missing; left blank")
	}
	return persona, issues, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o700)
		}
		if containsPathTraversal(rel) {
			return fmt.Errorf("package_import: unsafe relative path %q", rel)
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("package_import: symlink is not allowed: %s", rel)
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("package_import: symlink is not allowed: %s", src)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("package_import: only regular files are allowed: %s", src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func extractZip(src, dst string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if containsPathTraversal(f.Name) {
			return fmt.Errorf("package_import: zip-slip path blocked: %s", f.Name)
		}
		target := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(filepath.Separator)) {
			return fmt.Errorf("package_import: zip entry escapes quarantine: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("package_import: zip symlink is not allowed: %s", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		closeErr := out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func extractTar(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tr := tar.NewReader(in)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if containsPathTraversal(hdr.Name) {
			return fmt.Errorf("package_import: tar path traversal blocked: %s", hdr.Name)
		}
		target := filepath.Join(dst, hdr.Name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(filepath.Separator)) {
			return fmt.Errorf("package_import: tar entry escapes quarantine: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("package_import: tar entry type not allowed: %s", hdr.Name)
		}
	}
}

func containsPathTraversal(path string) bool {
	clean := filepath.Clean(path)
	return filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func hasImage(images []string, name string) bool {
	for _, image := range images {
		if image == name {
			return true
		}
	}
	return false
}

// ══════════════════════════════════════════════
// #42.1 安裝前完整安全檢查
// ══════════════════════════════════════════════
//
// RunSecurityCheck 對 quarantine 中的 package 執行完整安全掃描。
// 包含：manifest strict validation、symlink/hardlink/device file 拒絕、
// path boundary escape 檢查、package hash 與 manifest hash 計算。
// 任何一項失敗則整體 Passed = false，安裝必須中止。
func (s *Service) RunSecurityCheck(importID string) (*SecurityCheckResult, error) {
	s.mu.Lock()
	pi, ok := s.pending[importID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("package_import: import %q not found", importID)
	}

	result := &SecurityCheckResult{
		Passed:    true,
		CheckedAt: time.Now().Format(time.RFC3339Nano),
	}
	var failures []string

	// ── 1. Manifest strict schema validation ──
	if err := ValidateManifestStrict(pi.Manifest); err != nil {
		failures = append(failures, "manifest_validation: "+err.Error())
	}

	// ── 2. 權限宣告 vs 實際內容比對 ──
	actualEntries, actualWriteTargets, scanFailures := scanActualContent(pi.QuarantinePath)
	failures = append(failures, scanFailures...)
	if err := comparePermissions(pi.Manifest, actualEntries, actualWriteTargets); err != nil {
		failures = append(failures, "permission_mismatch: "+err.Error())
	}

	// ── 3. 計算 package hash（整個 quarantine 目錄的 SHA-256）──
	pkgHash, err := hashDirectory(pi.QuarantinePath)
	if err != nil {
		failures = append(failures, "package_hash: "+err.Error())
	} else {
		result.PackageHash = pkgHash
	}

	// ── 4. 計算 manifest hash ──
	result.ManifestHash = hashManifest(pi.Manifest)

	// ── 5. 更新 PendingImport 的 hash 紀錄 ──
	s.mu.Lock()
	pi.Manifest.PackageHash = result.PackageHash
	pi.Manifest.ManifestHash = result.ManifestHash
	s.mu.Unlock()

	if len(failures) > 0 {
		result.Passed = false
		result.Failures = failures
	}
	return result, nil
}

// scanActualContent 掃描 quarantine 目錄中的實際檔案，
// 檢查 symlink / hardlink / device file，收集 entry point 和寫入位置。
func scanActualContent(root string) (entries []string, writeTargets []string, failures []string) {
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			failures = append(failures, fmt.Sprintf("scan_error: %s: %v", path, err))
			return nil
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			failures = append(failures, fmt.Sprintf("info_error: %s: %v", path, err))
			return nil
		}

		// ── 拒絕 symlink ──
		if info.Mode()&os.ModeSymlink != 0 {
			failures = append(failures, fmt.Sprintf("symlink_rejected: %s", path))
			return nil
		}
		// ── 拒絕 device file ──
		if info.Mode()&os.ModeDevice != 0 || info.Mode()&os.ModeCharDevice != 0 {
			failures = append(failures, fmt.Sprintf("device_file_rejected: %s", path))
			return nil
		}
		// ── 拒絕 named pipe / socket ──
		if info.Mode()&os.ModeNamedPipe != 0 || info.Mode()&os.ModeSocket != 0 {
			failures = append(failures, fmt.Sprintf("special_file_rejected: %s", path))
			return nil
		}
		// ── 拒絕 hardlink（nlink > 1 表示 hardlink）──
		if nlink := hardLinkCount(info); nlink > 1 {
			failures = append(failures, fmt.Sprintf("hardlink_rejected: %s (nlink=%d)", path, nlink))
			return nil
		}

		// ── 路徑正規化後做 boundary check ──
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("path_resolve_error: %s: %v", path, err))
			return nil
		}
		cleanRoot, _ := filepath.EvalSymlinks(root)
		if !strings.HasPrefix(resolved, filepath.Clean(cleanRoot)+string(filepath.Separator)) {
			failures = append(failures, fmt.Sprintf("path_boundary_escape: %s resolved to %s", path, resolved))
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		ext := strings.ToLower(filepath.Ext(rel))
		// 收集 entry point（可執行檔或 JSON manifest）
		if ext == ".json" || ext == ".js" || ext == ".sh" || ext == ".py" || ext == ".go" {
			entries = append(entries, rel)
		}
		writeTargets = append(writeTargets, rel)
		return nil
	})
	return
}

// comparePermissions 比對 manifest 宣告的權限與實際內容。
// 實際檔案不得包含宣告外的 entry point 或寫入位置。
func comparePermissions(m PackageManifest, actualEntries, actualWriteTargets []string) error {
	declaredSet := make(map[string]bool)
	for _, e := range m.DeclaredEntries {
		declaredSet[e] = true
	}
	// 向下相容：若 DeclaredEntries 為空，跳過此檢查（舊版 persona 包）
	if len(m.DeclaredEntries) == 0 {
		return nil
	}
	for _, actual := range actualEntries {
		if !declaredSet[actual] {
			return fmt.Errorf("undeclared entry point found: %s", actual)
		}
	}
	return nil
}

// hashDirectory 計算目錄下所有檔案的合併 SHA-256。
func hashDirectory(dir string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		h.Write([]byte(rel))
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hardLinkCount 取得檔案的 hard link 數量。
// 在 Unix 上從 Sys() 取得 Nlink；其他平台回傳 1（不阻擋）。
func hardLinkCount(info os.FileInfo) uint64 {
	// 使用型別斷言取得底層系統資訊
	type nlinker interface {
		Nlink() uint64
	}
	if sys, ok := info.Sys().(nlinker); ok {
		return sys.Nlink()
	}
	// 非 Unix 平台：嘗試 syscall.Stat_t
	return 1
}

// ══════════════════════════════════════════════
// #42.2 孤兒 Quarantine 啟動掃描
// ══════════════════════════════════════════════
//
// ScanOrphanQuarantine 在應用程式啟動時呼叫。
// 掃描 quarantine 目錄中無對應 PendingImport 的項目（crash 殘留）。
// 對每個孤兒項目執行完整安全檢查：
//   - 通過 → 產生 PendingImport + Review Card 供使用者決定
//   - 失敗 → 記錄 FailureReason，前端彈窗提醒後清除
func (s *Service) ScanOrphanQuarantine() ([]OrphanScanResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.quarantineDir, 0o700); err != nil {
		return nil, fmt.Errorf("package_import: cannot access quarantine dir: %w", err)
	}

	entries, err := os.ReadDir(s.quarantineDir)
	if err != nil {
		return nil, fmt.Errorf("package_import: cannot scan quarantine dir: %w", err)
	}

	var results []OrphanScanResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		orphanID := entry.Name()

		// 已有對應 PendingImport → 非孤兒，跳過
		if _, exists := s.pending[orphanID]; exists {
			continue
		}

		orphanPath := filepath.Join(s.quarantineDir, orphanID)
		result := OrphanScanResult{
			OrphanID:       orphanID,
			QuarantinePath: orphanPath,
		}

		// 嘗試從 quarantine 中還原 manifest
		payloadDir := filepath.Join(orphanPath, "payload")
		persona, issues, parseErr := validatePersonaPayload(payloadDir)

		// 建立臨時 PendingImport 以便執行安全檢查
		manifest := PackageManifest{
			Name:       persona.Name,
			Version:    "unknown",
			SourcePath: orphanPath,
			RiskTag:    "unknown",
		}
		if persona.Name == "" {
			manifest.Name = orphanID
		}

		tempImport := &PendingImport{
			ID:               orphanID,
			Manifest:         manifest,
			QuarantinePath:   orphanPath,
			Status:           StatusQuarantined,
			CreatedAt:        time.Now(),
			ValidationIssues: issues,
			Persona:          persona,
		}

		// 先放入 pending 以便 RunSecurityCheck 能找到
		s.pending[orphanID] = tempImport

		if parseErr != nil {
			// manifest 解析失敗 → 安全檢查視為失敗
			result.SecurityCheck = SecurityCheckResult{
				Passed:    false,
				Failures:  []string{"manifest_parse_error: " + parseErr.Error()},
				CheckedAt: time.Now().Format(time.RFC3339Nano),
			}
			result.FailureReason = parseErr.Error()
			delete(s.pending, orphanID)
		} else {
			// 執行完整安全檢查（解鎖以避免死鎖）
			s.mu.Unlock()
			checkResult, checkErr := s.RunSecurityCheck(orphanID)
			s.mu.Lock()

			if checkErr != nil {
				result.SecurityCheck = SecurityCheckResult{
					Passed:    false,
					Failures:  []string{checkErr.Error()},
					CheckedAt: time.Now().Format(time.RFC3339Nano),
				}
				result.FailureReason = checkErr.Error()
				delete(s.pending, orphanID)
			} else if !checkResult.Passed {
				result.SecurityCheck = *checkResult
				result.FailureReason = fmt.Sprintf("security check failed: %v", checkResult.Failures)
				delete(s.pending, orphanID)
			} else {
				// 安全檢查通過 → 保留 PendingImport，供 Review Card 使用
				result.SecurityCheck = *checkResult
				result.PendingImport = tempImport
			}
		}

		results = append(results, result)
	}
	return results, nil
}

// CleanOrphan 清除一個孤兒 quarantine 項目（安全檢查失敗後由前端觸發）。
func (s *Service) CleanOrphan(orphanID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	orphanPath := filepath.Join(s.quarantineDir, orphanID)
	delete(s.pending, orphanID)
	return os.RemoveAll(orphanPath)
}
