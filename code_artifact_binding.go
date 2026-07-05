package main

// code_artifact_binding.go — 產出程式碼 → 資料區（引用文件）
//
// 需求脈絡（3.1.x）：
//   使用者請模型「產生程式碼」時，程式碼不再整坨貼在聊天室，而是：
//   1) 存成引用文件（data/references/files/<name>.<ext>），出現在右欄資料區卡片。
//   2) 附 tag ＋摘要（像一般檔案），卡片有「展開」→ 彈窗檢視器（可複製 / 匯出 / 標色）。
//   3) 聊天室只回一句「已產生並收進資料區」＋泡泡選項（是否編譯）。
//   4) 預設語言 Go；使用者指定（如 C++）則切換輸出。
//   5) 模型可用「4 種 LLM 專用標示」在程式碼上標重點 / 修改處（見 code marks 協定），
//      與使用者的 8 色標色系統（highlight_binding.go）完全獨立、不混用。
//
// 中繼資料存 data/references/files/.code_artifacts.json（點字首檔，
// ListReferenceFiles 天生跳過，不會變成幽靈卡片）。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/orchestration/skill_step"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ───────────────────────── 資料結構 ─────────────────────────

// CodeArtifactMark — LLM 專用標示（與使用者 8 色 slot 無關）。
// Slot 1..4，樣式在前端以粗體 / 斜體系呈現（刻意跟 8 色底色區隔）：
//
//	1 核心重點   → 粗體
//	2 本次修改   → 斜體
//	3 風險注意   → 粗體＋斜體
//	4 可改選項   → 半粗體＋點狀底線
type CodeArtifactMark struct {
	Slot        int    `json:"slot"` // 1..4
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
	Quote       string `json:"quote"`
}

// CodeArtifactMeta — 資料區程式碼卡片的中繼資料（tag、摘要、語言、標示）。
type CodeArtifactMeta struct {
	FileName      string             `json:"file_name"` // 引用庫內的實際檔名（唯一鍵）
	DisplayName   string             `json:"display_name"`
	Kind          string             `json:"kind"`     // 程式碼種類，例：小計算機程式
	Language      string             `json:"language"` // go | cpp | python | ...
	LanguageLabel string             `json:"language_label"`
	Tags          []string           `json:"tags"`
	Summary       string             `json:"summary"`
	CreatedAt     string             `json:"created_at"`
	Marks         []CodeArtifactMark `json:"marks,omitempty"`
	CompileStatus string             `json:"compile_status,omitempty"` // "" | success | failed | unsupported | tool_missing
	CompileDetail string             `json:"compile_detail,omitempty"`
	ContentSHA    string             `json:"content_sha,omitempty"` // 拖出再拉回時靠內容雜湊認親
	BinaryPath    string             `json:"binary_path,omitempty"` // 編譯成功後的執行檔（拖出時打包成資料夾）
}

// CodeArtifactDetail — 彈窗檢視器一次拿齊 meta ＋內容。
type CodeArtifactDetail struct {
	Meta    CodeArtifactMeta `json:"meta"`
	Content string           `json:"content"`
	Path    string           `json:"path"`
}

// CodeCompileResult — 編譯結果回報。
type CodeCompileResult struct {
	Status  string `json:"status"` // success | failed | unsupported | tool_missing
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

// CodeArtifactActivity — 編譯／自動修復期間給前端 pending 泡泡看的可觀測進度。
type CodeArtifactActivity struct {
	TraceID   string `json:"trace_id"`
	SessionID string `json:"session_id,omitempty"`
	FileName  string `json:"file_name,omitempty"`
	Phase     string `json:"phase"`
	Title     string `json:"title"`
	Detail    string `json:"detail,omitempty"`
	Status    string `json:"status"` // running | success | failed | warning
	Attempt   int    `json:"attempt,omitempty"`
	At        string `json:"at"`
}

const codeArtifactIndexFilename = ".code_artifacts.json"

var (
	codeArtifactMu         sync.Mutex
	lastCodeArtifactMu     sync.Mutex
	lastCodeArtifactBySess = map[string]string{} // sessionID → fileName（供「編譯剛產生的程式」）
	lastCodeLanguageBySess = map[string]string{} // sessionID → 語言 ID（語言黏著確認用）
)

func codeArtifactSHA(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func codeArtifactDir() string {
	return filepath.Join(appDataRoot(), "data", "references", "files")
}

func codeArtifactIndexPath() string {
	return filepath.Join(codeArtifactDir(), codeArtifactIndexFilename)
}

func loadCodeArtifactIndex() map[string]CodeArtifactMeta {
	index := map[string]CodeArtifactMeta{}
	raw, err := os.ReadFile(codeArtifactIndexPath())
	if err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &index) // 壞檔回空，不擋流程
	}
	return index
}

func saveCodeArtifactIndex(index map[string]CodeArtifactMeta) error {
	if err := os.MkdirAll(codeArtifactDir(), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(codeArtifactIndexPath(), raw, 0o600)
}

// ───────────────────────── 語言偵測（預設 Go） ─────────────────────────

type codeLanguageSpec struct {
	ID       string
	Label    string
	Ext      string
	Keywords []string // 出現在使用者文字 → 指定該語言
}

var codeLanguageSpecs = []codeLanguageSpec{
	{"cpp", "C++", "cpp", []string{"c++", "cpp", "c艹"}},
	{"csharp", "C#", "cs", []string{"c#", "csharp"}},
	{"python", "Python", "py", []string{"python", "py檔", "派森"}},
	{"typescript", "TypeScript", "ts", []string{"typescript", " ts "}},
	{"javascript", "JavaScript", "js", []string{"javascript", " js ", "node"}},
	{"html", "HTML", "html", []string{"html", "網頁版", "單一 html"}},
	{"rust", "Rust", "rs", []string{"rust"}},
	{"java", "Java", "java", []string{"java ", "java程式", "java 程式"}},
	{"swift", "Swift", "swift", []string{"swift"}},
	{"kotlin", "Kotlin", "kt", []string{"kotlin"}},
	{"shell", "Shell", "sh", []string{"bash", "shell", "腳本 sh"}},
	{"c", "C", "c", []string{"c語言", "c 語言", "用c寫", "用 c 寫"}},
	{"go", "Go", "go", []string{"golang", "go語言", "go 語言", "用go", "用 go"}},
}

var codeLanguageByID = func() map[string]codeLanguageSpec {
	m := map[string]codeLanguageSpec{}
	for _, spec := range codeLanguageSpecs {
		m[spec.ID] = spec
	}
	// fence tag 別名
	m["golang"] = m["go"]
	m["c++"] = m["cpp"]
	m["cs"] = m["csharp"]
	m["py"] = m["python"]
	m["ts"] = m["typescript"]
	m["js"] = m["javascript"]
	m["rs"] = m["rust"]
	m["bash"] = m["shell"]
	m["sh"] = m["shell"]
	return m
}()

// requestedCodeLanguage — 使用者有指定就回傳該語言與 true；沒指定回傳預設 Go 與 false。
func requestedCodeLanguage(userText string) (codeLanguageSpec, bool) {
	lower := " " + strings.ToLower(userText) + " "
	for _, spec := range codeLanguageSpecs {
		for _, kw := range spec.Keywords {
			if strings.Contains(lower, kw) {
				return spec, true
			}
		}
	}
	return codeLanguageByID["go"], false
}

// detectRequestedCodeLanguage — 使用者有指定就切換，沒指定回傳預設 Go。
func detectRequestedCodeLanguage(userText string) codeLanguageSpec {
	spec, _ := requestedCodeLanguage(userText)
	return spec
}

func languageSpecForFence(tag string, fallback codeLanguageSpec) codeLanguageSpec {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if spec, ok := codeLanguageByID[tag]; ok && tag != "" {
		return spec
	}
	return fallback
}

func codeArtifactLanguageForResponse(userText, fenceTag string) codeLanguageSpec {
	requested, explicit := requestedCodeLanguage(userText)
	if explicit {
		return requested
	}
	// 未指定語言時，產品規則是預設 Go；不要讓模型自行加上的 fence tag 改寫語言。
	return codeLanguageByID["go"]
}

// ───────────────────────── code fence 抽取 ─────────────────────────

var fencedCodeRe = regexp.MustCompile("(?s)```([A-Za-z0-9+#_-]*)[ \t]*\r?\n(.*?)```")

// extractLargestFencedCode — 取回覆中最大的一塊 fenced code（多塊時取最長）。
func extractLargestFencedCode(text string) (fenceTag, code, prose string, ok bool) {
	matches := fencedCodeRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return "", "", "", false
	}
	best := matches[0]
	for _, m := range matches[1:] {
		if (m[5] - m[4]) > (best[5] - best[4]) {
			best = m
		}
	}
	fenceTag = text[best[2]:best[3]]
	code = strings.Trim(text[best[4]:best[5]], "\n")
	prose = strings.TrimSpace(text[:best[0]])
	if len([]rune(prose)) > 160 {
		prose = string([]rune(prose)[:160]) + "…"
	}
	return fenceTag, code, prose, code != ""
}

// ───────────────────────── LLM 4 種標示協定 ─────────────────────────
//
// 模型在程式碼內用全形書名號標記（不會撞到任何主流語言語法）：
//   《標1》…《/標1》 核心重點
//   《標2》…《/標2》 本次修改
//   《標3》…《/標3》 風險注意
//   《標4》…《/標4》 可改選項
// 存檔前全部剝除（檔案保持可編譯），區段轉成 offset 存 meta。

var codeMarkOpenRe = regexp.MustCompile(`《標([1-4])》`)
var codeMarkCloseRe = regexp.MustCompile(`《/標[1-4]》`)

// CodeMarkSlotLegend — 給前端畫圖例、給提示詞組協定說明。
var codeMarkSlotLegend = []string{"核心重點", "本次修改", "風險注意", "可改選項"}

func parseCodeArtifactMarks(raw string) (clean string, marks []CodeArtifactMark) {
	var b strings.Builder
	b.Grow(len(raw))
	type openMark struct {
		slot  int
		start int
	}
	var stack []openMark
	rest := raw
	for len(rest) > 0 {
		openLoc := codeMarkOpenRe.FindStringSubmatchIndex(rest)
		closeLoc := codeMarkCloseRe.FindStringIndex(rest)
		// 取最先出現的 token
		nextOpen := -1
		if openLoc != nil {
			nextOpen = openLoc[0]
		}
		nextClose := -1
		if closeLoc != nil {
			nextClose = closeLoc[0]
		}
		switch {
		case nextOpen == -1 && nextClose == -1:
			b.WriteString(rest)
			rest = ""
		case nextClose == -1 || (nextOpen != -1 && nextOpen < nextClose):
			b.WriteString(rest[:nextOpen])
			slot := int(rest[openLoc[2]] - '0')
			stack = append(stack, openMark{slot: slot, start: b.Len()})
			rest = rest[openLoc[1]:]
		default:
			b.WriteString(rest[:nextClose])
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				end := b.Len()
				if end > top.start {
					quote := b.String()[top.start:end]
					if len([]rune(quote)) > 400 {
						quote = string([]rune(quote)[:400])
					}
					marks = append(marks, CodeArtifactMark{
						Slot:        top.slot,
						StartOffset: top.start,
						EndOffset:   end,
						Quote:       quote,
					})
				}
			}
			rest = rest[closeLoc[1]:]
		}
	}
	sort.Slice(marks, func(i, j int) bool { return marks[i].StartOffset < marks[j].StartOffset })
	return b.String(), marks
}

// codeMarkProtocolPrompt — 塞進 direct code answer 提示詞，讓模型主動標重點。
func codeMarkProtocolPrompt() string {
	return strings.Join([]string{
		"程式碼標示協定（選用，最多各 3 處、總計 8 處以內）：",
		"你可以在程式碼中用下列成對記號標出值得注意的區段，系統會剝除記號、轉成獨立標示顯示，不影響程式執行：",
		"《標1》核心重點《/標1》、《標2》本次修改處《/標2》、《標3》風險或需注意《/標3》、《標4》可改選項《/標4》。",
		"記號必須成對、不可巢狀，只包程式碼原文，不要包說明文字。",
	}, "\n")
}

// ───────────────────────── 命名 / tag / 摘要 ─────────────────────────

func deriveCodeArtifactKind(userText string, lang codeLanguageSpec) string {
	name := extractGoProgramName(userText)
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 24 {
		name = "程式"
	}
	if !strings.HasSuffix(name, "程式") {
		name += "程式"
	}
	return name
}

func deriveCodeArtifactTags(userText, kind string, lang codeLanguageSpec) []string {
	tags := []string{lang.Label, strings.TrimSuffix(kind, "程式")}
	pairs := []struct{ kw, tag string }{
		{"計算", "計算"}, {"檔案", "檔案處理"}, {"轉換", "轉換"}, {"排程", "排程"},
		{"搜尋", "搜尋"}, {"網頁", "網頁"}, {"測試", "測試"}, {"表格", "表格"}, {"報表", "報表"},
	}
	for _, p := range pairs {
		if strings.Contains(userText, p.kw) && !containsString(tags, p.tag) {
			tags = append(tags, p.tag)
		}
	}
	tags = append(tags, "AI產生")
	if len(tags) > 5 {
		tags = tags[:5]
	}
	return tags
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

// deriveCodeArtifactSummary — 先抓程式碼開頭註解，抓不到就濃縮使用者需求。
func deriveCodeArtifactSummary(userText, code string, lang codeLanguageSpec) string {
	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		comment := ""
		switch {
		case strings.HasPrefix(line, "//"):
			comment = strings.TrimSpace(strings.TrimPrefix(line, "//"))
		case strings.HasPrefix(line, "#") && lang.ID != "csharp":
			comment = strings.TrimSpace(strings.TrimLeft(line, "#!"))
		case strings.HasPrefix(line, "<!--"):
			comment = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "<!--"), "-->"))
		}
		if comment != "" && !strings.HasPrefix(comment, "go:") {
			if runes := []rune(comment); len(runes) > 60 {
				comment = string(runes[:60]) + "…"
			}
			return comment
		}
		break // 第一個非空行不是註解 → 放棄註解路線
	}
	need := strings.TrimSpace(userText)
	if runes := []rune(need); len(runes) > 60 {
		need = string(runes[:60]) + "…"
	}
	return "依需求產生：" + need
}

func sanitizeArtifactFileStem(kind string) string {
	stem := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r >= 0x4E00 && r <= 0x9FFF: // CJK 保留
			return r
		default:
			return '_'
		}
	}, strings.TrimSuffix(kind, "程式"))
	stem = strings.Trim(stem, "_")
	if stem == "" {
		stem = "program"
	}
	return stem
}

// ───────────────────────── 存檔（進資料區） ─────────────────────────

func (a *App) saveCodeArtifact(userText, fenceTag, rawCode, sessionID, traceID string) (*CodeArtifactMeta, error) {
	lang := codeArtifactLanguageForResponse(userText, fenceTag)
	clean, marks := parseCodeArtifactMarks(rawCode)
	kind := deriveCodeArtifactKind(userText, lang)

	if err := os.MkdirAll(codeArtifactDir(), 0o700); err != nil {
		return nil, err
	}
	stem := sanitizeArtifactFileStem(kind)
	fileName := fmt.Sprintf("%s.%s", stem, lang.Ext)
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(codeArtifactDir(), fileName)); os.IsNotExist(err) {
			break
		}
		fileName = fmt.Sprintf("%s_%d.%s", stem, i, lang.Ext)
	}
	if err := os.WriteFile(filepath.Join(codeArtifactDir(), fileName), []byte(clean), 0o600); err != nil {
		return nil, err
	}

	meta := CodeArtifactMeta{
		FileName:      fileName,
		DisplayName:   kind,
		Kind:          kind,
		Language:      lang.ID,
		LanguageLabel: lang.Label,
		Tags:          deriveCodeArtifactTags(userText, kind, lang),
		Summary:       deriveCodeArtifactSummary(userText, clean, lang),
		CreatedAt:     time.Now().Format(time.RFC3339),
		Marks:         marks,
		ContentSHA:    codeArtifactSHA([]byte(clean)),
	}

	codeArtifactMu.Lock()
	index := loadCodeArtifactIndex()
	index[fileName] = meta
	err := saveCodeArtifactIndex(index)
	codeArtifactMu.Unlock()
	if err != nil {
		return nil, err
	}

	lastCodeArtifactMu.Lock()
	lastCodeArtifactBySess[sessionID] = fileName
	lastCodeLanguageBySess[sessionID] = lang.ID
	lastCodeArtifactMu.Unlock()

	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "code_artifact:created", meta)
	}
	debugtrace.Record("go.codeArtifact.saved", traceID, map[string]interface{}{
		"file_name": fileName,
		"language":  lang.ID,
		"marks":     len(marks),
	})
	return &meta, nil
}

// captureCodeArtifactFromDirectAnswer — direct code answer 回覆攔截點。
// 有 code fence → 存進資料區、掛編譯泡泡，聊天室改回一句話；沒有 → 原文照走。
func (a *App) captureCodeArtifactFromDirectAnswer(userText, responseText, sessionID, traceID string) (string, bool) {
	fenceTag, code, prose, ok := extractLargestFencedCode(responseText)
	if !ok || len(strings.Split(code, "\n")) < 3 {
		return "", false
	}
	meta, err := a.saveCodeArtifact(userText, fenceTag, code, sessionID, traceID)
	if err != nil {
		debugtrace.Record("go.codeArtifact.save_error", traceID, map[string]interface{}{"error": err.Error()})
		return "", false
	}

	// 是否編譯泡泡（對話框上方）；不可編譯語言就不掛。
	if codeArtifactCompileSupported(meta.Language) {
		setCustomFloatingCandidates("要不要順手編譯這支程式？", []FloatingCandidate{
			{ID: "code-compile-yes", Label: "編譯程式碼", Draft: "編譯資料區程式：" + meta.FileName},
			{ID: "code-compile-no", Label: "先不編譯", Draft: "先不編譯資料區程式"},
		}, traceID)
	}

	// [[code-artifact:檔名]] 由前端 MessageText 渲染成「展開」按鈕（按了開彈窗、卡片亮起）。
	lines := []string{
		fmt.Sprintf("已產生並檢查 %s 的「%s」，收在資料區（引用文件）。 [[code-artifact:%s]]", meta.LanguageLabel, meta.Kind, meta.FileName),
		"摘要：" + meta.Summary,
	}
	if len(meta.Marks) > 0 {
		lines = append(lines, fmt.Sprintf("模型自己標了 %d 處重點／修改，展開後看得到。", len(meta.Marks)))
	}
	lines = append(lines, "tag："+strings.Join(meta.Tags, "、"))
	if prose != "" {
		lines = append([]string{prose}, lines...)
	}
	return strings.Join(lines, "\n"), true
}

// setCustomFloatingCandidates — 自訂泡泡（沿用 readiness gate 浮動候選管線）。
func setCustomFloatingCandidates(question string, candidates []FloatingCandidate, traceID string) {
	readinessMu.Lock()
	currentGateState.FloatingCandidates = candidates
	currentGateState.MissingSlots = nil
	currentGateState.RiskTier = "none"
	readinessMu.Unlock()
	debugtrace.Record("readiness.custom_candidates", traceID, map[string]interface{}{
		"question":        question,
		"candidate_count": len(candidates),
	})
}

// codeIntentAmbiguous — 分不出「純程式碼」還是「新 skill」時回 true（由授權流程掛泡泡）。
// 走到這裡時 isDirectCodeAnswerRequest 已為 false；只要再確認沒有 skill 傾向詞即可視為模糊。
func codeIntentAmbiguous(userText string) bool {
	text := strings.TrimSpace(userText)
	if text == "" || strings.Contains(text, "使用者補充:") {
		return false
	}
	if strings.HasPrefix(text, "我想要純程式碼") || strings.HasPrefix(text, "我想要新的skill") {
		return false
	}
	lower := strings.ToLower(text)
	mentionsProgram := containsAny(lower, []string{"code", "program"}) ||
		containsAny(text, []string{"程式", "小程式", "計算機", "工具程式"})
	if !mentionsProgram {
		return false
	}
	wantsSkill := containsAny(lower, []string{"skill"}) ||
		containsAny(text, []string{"流程", "安裝", "保存", "加入工具", "建立工具", "做成工具", "排程", "定期"})
	return !wantsSkill
}

// ───────────────────────── 編譯 ─────────────────────────

func codeArtifactCompileSupported(langID string) bool {
	switch langID {
	case "go", "c", "cpp", "rust":
		return true
	}
	return false
}

// maybeHandleCodeArtifactControl — 編譯泡泡點下來的訊息在進路由前攔截。
func (a *App) maybeHandleCodeArtifactControl(adapterID, userText, sessionID, traceID string) (*skill_step.CLIResponse, bool) {
	text := strings.TrimSpace(userText)
	if text == "" {
		return nil, false
	}
	if strings.HasPrefix(text, "先不編譯資料區程式") {
		return &skill_step.CLIResponse{Text: "好，程式先放資料區，要編譯時再叫一聲。"}, true
	}
	// 只有泡泡的精確指令直接編譯；口語「幫我編譯」屬意圖不明 → 掛確認泡泡。
	if after, found := strings.CutPrefix(text, "編譯資料區程式："); found {
		// 泡泡點選後可能再帶使用者打的字（換行相接），檔名只取第一行。
		target := strings.TrimSpace(strings.SplitN(after, "\n", 2)[0])
		if target == "" {
			return nil, false
		}
		// 編譯＋失敗自動回修（最多 2 輪，見 code_artifact_repair.go）
		return &skill_step.CLIResponse{Text: a.compileCodeArtifactWithAutoRepair(adapterID, sessionID, target, traceID)}, true
	}
	// 口語編譯（短句、帶指涉詞）→ 找出對象後掛泡泡確認，不直接動手。
	if strings.Contains(text, "編譯") && len([]rune(text)) <= 40 &&
		containsAny(text, []string{"幫我", "一下", "剛", "程式", "檔", "那個", "這個", "資料區"}) {
		lastCodeArtifactMu.Lock()
		target := lastCodeArtifactBySess[sessionID]
		lastCodeArtifactMu.Unlock()
		if target == "" {
			// session 沒記錄 → 若資料區只有一支可編譯的就當對象
			if list, err := a.ListCodeArtifacts(); err == nil {
				compilable := []CodeArtifactMeta{}
				for _, m := range list {
					if codeArtifactCompileSupported(m.Language) {
						compilable = append(compilable, m)
					}
				}
				if len(compilable) == 1 {
					target = compilable[0].FileName
				}
			}
		}
		if target == "" {
			return &skill_step.CLIResponse{Text: "資料區裡沒找到可編譯的程式；先請我產一支，或講明檔名。"}, true
		}
		meta, _ := codeArtifactMetaByFileName(target)
		if !codeArtifactCompileSupported(meta.Language) {
			return &skill_step.CLIResponse{Text: fmt.Sprintf("「%s」是 %s，直譯／腳本類不用編譯，直接執行即可。", meta.Kind, meta.LanguageLabel)}, true
		}
		question := fmt.Sprintf("要我編譯「%s」（%s）嗎？點泡泡確認。", meta.Kind, meta.FileName)
		setCustomFloatingCandidates(question, []FloatingCandidate{
			{ID: "code-compile-confirm", Label: "編譯程式碼", Draft: "編譯資料區程式：" + meta.FileName},
			{ID: "code-compile-skip", Label: "先不編譯", Draft: "先不編譯資料區程式"},
		}, traceID)
		return &skill_step.CLIResponse{Text: question}, true
	}
	return nil, false
}

// maybeConfirmCodeArtifactLanguage — 語言黏著確認：
// 本輪沒指定語言、但這個 session 上次產碼用的不是預設 Go → 泡泡問清楚再動工。
func (a *App) maybeConfirmCodeArtifactLanguage(userText, sessionID, traceID string) (string, bool) {
	if _, explicit := requestedCodeLanguage(userText); explicit {
		return "", false
	}
	lastCodeArtifactMu.Lock()
	lastLangID := lastCodeLanguageBySess[sessionID]
	lastCodeArtifactMu.Unlock()
	if lastLangID == "" || lastLangID == "go" {
		return "", false // 沒歷史或本來就預設 → 直接用 Go，不囉嗦
	}
	lastSpec, ok := codeLanguageByID[lastLangID]
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(userText)
	question := fmt.Sprintf("這次沒指定語言——還是用上次的 %s？還是回預設 Go？", lastSpec.Label)
	setCustomFloatingCandidates(question, []FloatingCandidate{
		{ID: "code-lang-last", Label: "繼續用 " + lastSpec.Label, Draft: fmt.Sprintf("我想要純程式碼，用 %s 寫：%s", strings.ToLower(lastSpec.Label), trimmed)},
		{ID: "code-lang-go", Label: "改用預設 Go", Draft: "我想要純程式碼，用 go 寫：" + trimmed},
	}, traceID)
	debugtrace.Record("go.codeArtifact.language_confirm", traceID, map[string]interface{}{
		"last_language": lastLangID,
	})
	return question, true
}

// CompileCodeArtifact — 在系統暫存目錄編譯資料區的程式碼檔（不污染引用庫）。
func (a *App) CompileCodeArtifact(fileName string) (*CodeCompileResult, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName)) // 防路徑跳脫
	codeArtifactMu.Lock()
	index := loadCodeArtifactIndex()
	meta, ok := index[fileName]
	codeArtifactMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("資料區找不到程式碼中繼資料：%s", fileName)
	}
	srcPath := filepath.Join(codeArtifactDir(), fileName)
	if _, err := os.Stat(srcPath); err != nil {
		return nil, fmt.Errorf("資料區檔案不見了：%s", fileName)
	}

	result := &CodeCompileResult{}
	if !codeArtifactCompileSupported(meta.Language) {
		result.Status = "unsupported"
		result.Message = fmt.Sprintf("%s 屬直譯／腳本類，不用編譯，直接執行即可。", meta.LanguageLabel)
		a.updateCodeArtifactCompileStatus(fileName, result.Status, result.Message)
		return result, nil
	}

	workDir, err := os.MkdirTemp("", "code-artifact-build-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workDir)
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}
	buildSrc := filepath.Join(workDir, fileName)
	if err := os.WriteFile(buildSrc, raw, 0o600); err != nil {
		return nil, err
	}

	var cmd *exec.Cmd
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	switch meta.Language {
	case "go":
		if _, lookErr := exec.LookPath("go"); lookErr != nil {
			result.Status = "tool_missing"
			result.Message = "找不到 go 工具鏈；裝好 Go 之後再來編譯。"
			a.updateCodeArtifactCompileStatus(fileName, result.Status, result.Message)
			return result, nil
		}
		modCmd := exec.CommandContext(ctx, "go", "mod", "init", "code_artifact_build")
		modCmd.Dir = workDir
		_ = modCmd.Run()
		cmd = exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(workDir, "artifact_bin"), ".")
	case "c", "cpp":
		compiler := "cc"
		if meta.Language == "cpp" {
			compiler = "c++"
		}
		if _, lookErr := exec.LookPath(compiler); lookErr != nil {
			result.Status = "tool_missing"
			result.Message = fmt.Sprintf("找不到 %s 編譯器。", compiler)
			a.updateCodeArtifactCompileStatus(fileName, result.Status, result.Message)
			return result, nil
		}
		cmd = exec.CommandContext(ctx, compiler, buildSrc, "-o", filepath.Join(workDir, "artifact_bin"))
	case "rust":
		if _, lookErr := exec.LookPath("rustc"); lookErr != nil {
			result.Status = "tool_missing"
			result.Message = "找不到 rustc。"
			a.updateCodeArtifactCompileStatus(fileName, result.Status, result.Message)
			return result, nil
		}
		cmd = exec.CommandContext(ctx, "rustc", buildSrc, "-o", filepath.Join(workDir, "artifact_bin"))
	}
	cmd.Dir = workDir
	out, buildErr := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if runes := []rune(output); len(runes) > 1200 {
		output = string(runes[:1200]) + "…"
	}
	if buildErr != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("「%s」編譯失敗，錯誤訊息附上——要我修的話把它展開貼回來。\n%s", meta.Kind, output)
		result.Output = output
	} else {
		result.Status = "success"
		result.Message = fmt.Sprintf("「%s」編譯通過。執行檔已留存，拖出時會連原始碼一起打包成資料夾。", meta.Kind)
		result.Output = output
	}
	binaryPath := ""
	if result.Status == "success" {
		// 執行檔留存到 .code_builds/<fileName>/（點字首目錄，不會變卡片）；
		// 之後拖出時與原始碼一起包成資料夾。
		binName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		if goruntime.GOOS == "windows" {
			binName += ".exe"
		}
		buildKeepDir := filepath.Join(codeArtifactDir(), ".code_builds", fileName)
		if err := os.MkdirAll(buildKeepDir, 0o700); err == nil {
			if binRaw, readErr := os.ReadFile(filepath.Join(workDir, "artifact_bin")); readErr == nil {
				keepPath := filepath.Join(buildKeepDir, binName)
				if writeErr := os.WriteFile(keepPath, binRaw, 0o755); writeErr == nil {
					binaryPath = keepPath
				}
			}
		}
	}
	a.updateCodeArtifactCompileStatus(fileName, result.Status, firstNonEmpty(result.Output, result.Message))
	a.updateCodeArtifactBinaryPath(fileName, binaryPath)
	return result, nil
}

func (a *App) updateCodeArtifactBinaryPath(fileName, binaryPath string) {
	codeArtifactMu.Lock()
	index := loadCodeArtifactIndex()
	meta, ok := index[fileName]
	if !ok {
		codeArtifactMu.Unlock()
		return
	}
	meta.BinaryPath = binaryPath
	index[fileName] = meta
	_ = saveCodeArtifactIndex(index)
	codeArtifactMu.Unlock()
	a.emitCodeArtifactUpdated(meta)
}

// maybeReattachCodeArtifactMeta — 拖出去再拉回來的程式檔，靠內容雜湊認親、
// 自動補回 meta（tag／摘要／標示），讓顯示狀態跟移出前一致。
func (a *App) maybeReattachCodeArtifactMeta(newFileName string) {
	newFileName = filepath.Base(strings.TrimSpace(newFileName))
	if newFileName == "" {
		return
	}
	raw, err := os.ReadFile(filepath.Join(codeArtifactDir(), newFileName))
	if err != nil {
		return
	}
	sha := codeArtifactSHA(raw)
	codeArtifactMu.Lock()
	defer codeArtifactMu.Unlock()
	index := loadCodeArtifactIndex()
	if _, exists := index[newFileName]; exists {
		return // 已有 meta，不動
	}
	for _, meta := range index {
		if meta.ContentSHA != "" && meta.ContentSHA == sha {
			clone := meta
			clone.FileName = newFileName
			index[newFileName] = clone
			_ = saveCodeArtifactIndex(index)
			return
		}
	}
}

// CopyCodeArtifactToClipboard — 走 Go 端剪貼簿，前端不用選取文字（不反黑）。
func (a *App) CopyCodeArtifactToClipboard(fileName string) error {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	raw, err := os.ReadFile(filepath.Join(codeArtifactDir(), fileName))
	if err != nil {
		return fmt.Errorf("資料區檔案讀不到：%w", err)
	}
	return wailsruntime.ClipboardSetText(a.ctx, string(raw))
}

// NativeDragExportCodeArtifactBundle — 有編譯產物時，拖出成「資料夾」
// （內含原始碼＋執行檔）；沒有編譯產物就照舊拖單檔。
func (a *App) NativeDragExportCodeArtifactBundle(fileName string) (*NativeReferenceFileDragResult, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	srcPath := filepath.Join(codeArtifactDir(), fileName)
	if _, err := os.Stat(srcPath); err != nil {
		return nil, fmt.Errorf("資料區檔案不見了：%s", fileName)
	}
	meta, ok := codeArtifactMetaByFileName(fileName)
	if !ok || meta.BinaryPath == "" {
		return a.NativeDragExportReferenceFile(srcPath)
	}
	if _, err := os.Stat(meta.BinaryPath); err != nil {
		return a.NativeDragExportReferenceFile(srcPath) // 執行檔遺失 → 退回單檔
	}
	stem := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	tmpParent, err := os.MkdirTemp("", "code-artifact-bundle-")
	if err != nil {
		return nil, err
	}
	bundleDir := filepath.Join(tmpParent, stem)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, err
	}
	srcRaw, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(bundleDir, fileName), srcRaw, 0o644); err != nil {
		return nil, err
	}
	binRaw, err := os.ReadFile(meta.BinaryPath)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(bundleDir, filepath.Base(meta.BinaryPath)), binRaw, 0o755); err != nil {
		return nil, err
	}

	dragResult := startNativeFileDrag(bundleDir)
	out := &NativeReferenceFileDragResult{
		Status:           dragResult.Status,
		SourcePath:       srcPath,
		LandedPath:       dragResult.LandedPath,
		Platform:         goruntime.GOOS,
		FallbackRequired: dragResult.FallbackRequired,
		Message:          dragResult.Message,
		DisplayName:      stem + "（原始碼＋執行檔）",
		DropTargetKind:   dragResult.DropTargetKind,
		DropTargetDir:    dragResult.DropTargetDir,
	}
	if dragResult.Status == nativeDragStatusSuccess && a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "reference:native_completed", out)
	}
	return out, nil
}

func (a *App) updateCodeArtifactCompileStatus(fileName, status, detail string) {
	codeArtifactMu.Lock()
	index := loadCodeArtifactIndex()
	meta, ok := index[fileName]
	if !ok {
		codeArtifactMu.Unlock()
		return
	}
	meta.CompileStatus = status
	if runes := []rune(detail); len(runes) > 600 {
		detail = string(runes[:600]) + "…"
	}
	meta.CompileDetail = detail
	index[fileName] = meta
	_ = saveCodeArtifactIndex(index)
	codeArtifactMu.Unlock()
	a.emitCodeArtifactUpdated(meta)
}

func (a *App) emitCodeArtifactUpdated(meta CodeArtifactMeta) {
	if a != nil && a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "code_artifact:updated", meta)
	}
}

func (a *App) emitCodeArtifactActivity(traceID, sessionID, fileName, phase, title, detail, status string, attempt int) {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return
	}
	if status == "" {
		status = "running"
	}
	detail = codeArtifactActivityDetail(detail)
	payload := CodeArtifactActivity{
		TraceID:   traceID,
		SessionID: strings.TrimSpace(sessionID),
		FileName:  filepath.Base(strings.TrimSpace(fileName)),
		Phase:     strings.TrimSpace(phase),
		Title:     strings.TrimSpace(title),
		Detail:    detail,
		Status:    strings.TrimSpace(status),
		Attempt:   attempt,
		At:        time.Now().Format(time.RFC3339),
	}
	if payload.Title == "" {
		payload.Title = "資料區程式處理中"
	}
	if a != nil && a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "code_artifact:activity", payload)
	}
	debugtrace.Record("go.codeArtifact.activity", traceID, map[string]interface{}{
		"file_name": payload.FileName,
		"phase":     payload.Phase,
		"title":     payload.Title,
		"status":    payload.Status,
		"attempt":   payload.Attempt,
	})
}

func codeArtifactActivityDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return ""
	}
	detail = strings.Join(strings.Fields(detail), " ")
	if runes := []rune(detail); len(runes) > 320 {
		return string(runes[:320]) + "…"
	}
	return detail
}

// lastCodeArtifactForSession 取回該 session 最後產生的資料區程式碼（meta＋全文），
// 供 direct code 在「追加需求／修改上一版」時把完整舊碼帶進 prompt。
func lastCodeArtifactForSession(sessionID string) (CodeArtifactMeta, string, bool) {
	lastCodeArtifactMu.Lock()
	fileName := strings.TrimSpace(lastCodeArtifactBySess[strings.TrimSpace(sessionID)])
	lastCodeArtifactMu.Unlock()
	if fileName == "" {
		return CodeArtifactMeta{}, "", false
	}
	meta, ok := codeArtifactMetaByFileName(fileName)
	if !ok {
		return CodeArtifactMeta{}, "", false
	}
	raw, err := os.ReadFile(filepath.Join(codeArtifactDir(), filepath.Base(fileName)))
	if err != nil {
		return CodeArtifactMeta{}, "", false
	}
	return meta, string(raw), true
}

// codeArtifactMetaByFileName — 提供其他模組（如 D: 區塊）查 meta。
func codeArtifactMetaByFileName(fileName string) (CodeArtifactMeta, bool) {
	codeArtifactMu.Lock()
	defer codeArtifactMu.Unlock()
	meta, ok := loadCodeArtifactIndex()[filepath.Base(strings.TrimSpace(fileName))]
	return meta, ok
}

// ───────────────────────── Wails bindings（前端呼叫） ─────────────────────────

// ListCodeArtifacts — 資料區卡片要顯示 tag／摘要／展開鈕用。
func (a *App) ListCodeArtifacts() ([]CodeArtifactMeta, error) {
	codeArtifactMu.Lock()
	index := loadCodeArtifactIndex()
	codeArtifactMu.Unlock()
	list := make([]CodeArtifactMeta, 0, len(index))
	for fileName, meta := range index {
		if _, err := os.Stat(filepath.Join(codeArtifactDir(), fileName)); err != nil {
			continue // 檔被使用者移走 → 卡片自然消失，不留幽靈 meta
		}
		list = append(list, meta)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt > list[j].CreatedAt })
	return list, nil
}

// GetCodeArtifactDetail — 展開彈窗一次拿 meta＋全文。
func (a *App) GetCodeArtifactDetail(fileName string) (*CodeArtifactDetail, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	codeArtifactMu.Lock()
	index := loadCodeArtifactIndex()
	meta, ok := index[fileName]
	codeArtifactMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("資料區沒有這支程式的中繼資料：%s", fileName)
	}
	path := filepath.Join(codeArtifactDir(), fileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &CodeArtifactDetail{Meta: meta, Content: string(raw), Path: path}, nil
}

// ExportCodeArtifactToFolder — 匯出選單：跳資料夾選擇框，複製檔案過去。
func (a *App) ExportCodeArtifactToFolder(fileName string) (string, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	srcPath := filepath.Join(codeArtifactDir(), fileName)
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("資料區檔案讀不到：%w", err)
	}
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "選擇要放程式碼的資料夾",
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(dir) == "" {
		return "", nil // 使用者取消
	}
	destPath := filepath.Join(dir, fileName)
	for i := 2; ; i++ {
		if _, statErr := os.Stat(destPath); os.IsNotExist(statErr) {
			break
		}
		ext := filepath.Ext(fileName)
		destPath = filepath.Join(dir, fmt.Sprintf("%s_%d%s", strings.TrimSuffix(fileName, ext), i, ext))
	}
	if err := os.WriteFile(destPath, raw, 0o644); err != nil {
		return "", err
	}
	return destPath, nil
}
