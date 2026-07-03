// gguf_import_binding.go — GGUF → Ollama 本地模型匯入。
//
// 路線（走既有「引用連結」入口，UI 不另開 modal）：
//
//	使用者貼上 .gguf 本地路徑或 https://...gguf URL
//	→ PreviewExternalLink 判為 adapter_candidate（reason 含 GGUF 字樣，前端據此顯示）
//	→ RegisterExternalLink 呼叫 StartGGUFImport 起背景 job：
//	     (URL 來源) 先下載（gguf:import_progress phase=download）
//	     → 產生暫存 Modelfile → `ollama create <name> -f`（phase=create）
//	     → 成功後自動 RegisterLocal + adapter:list_changed
//	→ 之後 ollama list / ScanLocalModels 都看得到，掃描端零改動。
//
// 設計沿用 embedding_binding.go 的 pull job 模式：goroutine + eventbus、
// 同來源去重、fail-soft 進度解析；splitOnCRorLF / parsePullPercent 直接共用。
package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/adapter_registry"
	"ui_console/internal/urlsafe"
	"ui_console/shared/eventbus"
	"ui_console/shared/executil"
)

// ─────────────────────────────────────
// 偵測（Preview / Register 共用）
// ─────────────────────────────────────

// isGGUFLocalFile：本地存在的一般檔案且副檔名為 .gguf。
func isGGUFLocalFile(path string) bool {
	path = expandUserPath(path)
	if !strings.EqualFold(filepath.Ext(path), ".gguf") {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// isGGUFRemoteURL：https URL 且 path 以 .gguf 結尾（query 忽略，兼容 HuggingFace ?download=true）。
// 僅 https：下載走 PolicyCloudAPI，本來就擋 http，這裡先擋掉給使用者清楚訊息。
func isGGUFRemoteURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(filepath.Ext(parsed.Path), ".gguf")
}

// isGGUFImportSource：本地 .gguf 檔或遠端 .gguf URL 皆可作匯入來源。
func isGGUFImportSource(raw string) bool {
	return isGGUFLocalFile(raw) || isGGUFRemoteURL(raw)
}

// ggufModelName 從來源檔名推導 ollama 模型名，正規化為 [a-z0-9._-]。
// 例 "Qwen2.5-7B-Instruct-Q4_K_M.gguf" → "qwen2.5-7b-instruct-q4_k_m"。
func ggufModelName(source string) string {
	base := source
	if isGGUFRemoteURL(source) {
		if parsed, err := url.Parse(strings.TrimSpace(source)); err == nil {
			base = parsed.Path
		}
	} else {
		base = expandUserPath(source)
	}
	base = filepath.ToSlash(base)
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-.")
	if name == "" {
		name = fmt.Sprintf("gguf-import-%d", time.Now().Unix())
	}
	return name
}

// verifyGGUFMagic：GGUF 檔頭前 4 bytes 為 "GGUF"；擋下載壞掉/貼錯檔的情況。
func verifyGGUFMagic(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return fmt.Errorf("gguf: 檔案過小，非有效 GGUF：%w", err)
	}
	if string(magic) != "GGUF" {
		return fmt.Errorf("gguf: 檔頭不是 GGUF magic（讀到 %q），可能下載不完整或不是 GGUF 檔", string(magic))
	}
	return nil
}

// ─────────────────────────────────────
// 匯入 job
// ─────────────────────────────────────

// GGUFImportJob — StartGGUFImport 立即回的票根；事件 payload 以 jobId 對應。
type GGUFImportJob struct {
	JobID     string `json:"jobId"`
	Source    string `json:"source"`
	ModelName string `json:"modelName"`
}

// 同來源去重（比照 embedding pull job）。key = lowercase source。
var (
	ggufJobMu     sync.Mutex
	ggufJobActive = map[string]struct{}{}
)

// maxGGUFDownloadBytes — 下載上限 sanity cap（120 GiB）。
const maxGGUFDownloadBytes = int64(120) << 30

const ggufLlamaCPPFallbackEndpoint = "http://localhost:1234/v1"

// StartGGUFImport：背景匯入 GGUF 成 Ollama 本地模型；立刻回 jobID。
// 進度走 eventbus gguf:import_*。同來源已在跑 → error。
func (a *App) StartGGUFImport(source string) (GGUFImportJob, error) {
	source = strings.TrimSpace(source)
	if !isGGUFImportSource(source) {
		return GGUFImportJob{}, fmt.Errorf("gguf: %q 不是本地 .gguf 檔或 https .gguf 連結", source)
	}
	key := strings.ToLower(source)
	ggufJobMu.Lock()
	if _, busy := ggufJobActive[key]; busy {
		ggufJobMu.Unlock()
		return GGUFImportJob{}, fmt.Errorf("gguf: 此來源已在匯入中")
	}
	ggufJobActive[key] = struct{}{}
	ggufJobMu.Unlock()

	job := GGUFImportJob{
		JobID:     fmt.Sprintf("gguf-%d", time.Now().UnixNano()),
		Source:    source,
		ModelName: ggufModelName(source),
	}
	go a.runGGUFImport(job, key)
	return job, nil
}

// runGGUFImport — 背景 goroutine：下載（如需）→ 驗 magic → 喚醒 daemon
// → ollama create → 自動註冊 adapter → emit done。
func (a *App) runGGUFImport(job GGUFImportJob, dedupeKey string) {
	defer func() {
		ggufJobMu.Lock()
		delete(ggufJobActive, dedupeKey)
		ggufJobMu.Unlock()
	}()

	a.emitGGUFEvent(eventbus.EventGGUFImportStarted, map[string]any{
		"jobId":  job.JobID,
		"source": job.Source,
		"model":  job.ModelName,
	})

	ggufPath := expandUserPath(job.Source)
	downloaded := false
	if isGGUFRemoteURL(job.Source) {
		path, err := a.downloadGGUF(job)
		if err != nil {
			a.emitGGUFFailed(job, "下載失敗："+err.Error())
			return
		}
		ggufPath = path
		downloaded = true
	}

	cleanupDownload := func() {
		if downloaded {
			_ = os.Remove(ggufPath)
		}
	}

	if err := verifyGGUFMagic(ggufPath); err != nil {
		cleanupDownload()
		a.emitGGUFFailed(job, err.Error())
		return
	}

	// ollama create 走 client-server，daemon 得先在線。Ollama 不接受時，改註冊成
	// llama.cpp/OpenAI-compatible endpoint，讓有效 GGUF 不會卡死在「引用文件」流程。
	if err := wakeOllamaDaemon("", ""); err != nil {
		a.emitGGUFFallbackOrFailure(job, ggufPath, "Ollama 服務未就緒："+err.Error(), cleanupDownload)
		return
	}

	if err := a.runOllamaCreate(job, ggufPath); err != nil {
		a.emitGGUFFallbackOrFailure(job, ggufPath, err.Error(), cleanupDownload)
		return
	}
	// create 成功後 blob 已複製進 ollama 模型庫，下載的暫存檔可刪。
	cleanupDownload()

	modelID := job.ModelName + ":latest"
	note := ""
	if isOllamaGenerativeModelID(modelID) {
		adapterID := "local-ollama-" + sanitizeAdapterID(modelID)
		if err := a.adapterRegistry.RegisterLocal(adapterID, "Ollama - "+modelID, "◉", "http://localhost:11434/v1", modelID); err != nil {
			note = "模型已建立，但自動註冊 adapter 失敗：" + err.Error()
		} else if a.eventBus != nil {
			a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
				"adapter_id": adapterID,
				"kind":       "local",
			})
		}
	} else {
		note = "模型名稱含 embedding 關鍵字，未自動加入聊天模型清單"
	}

	payload := map[string]any{
		"jobId": job.JobID,
		"model": modelID,
	}
	if note != "" {
		payload["note"] = note
	}
	a.emitGGUFEvent(eventbus.EventGGUFImportDone, payload)
}

func (a *App) emitGGUFFallbackOrFailure(job GGUFImportJob, ggufPath, ollamaReason string, cleanupDownload func()) {
	note, ok := a.registerGGUFLlamaCPPAdapter(job, ggufPath, ollamaReason)
	if !ok {
		if cleanupDownload != nil {
			cleanupDownload()
		}
		a.emitGGUFFailed(job, note)
		return
	}
	a.emitGGUFEvent(eventbus.EventGGUFImportDone, map[string]any{
		"jobId":    job.JobID,
		"model":    job.ModelName,
		"note":     note,
		"fallback": "llamacpp",
	})
}

func (a *App) registerGGUFLlamaCPPAdapter(job GGUFImportJob, ggufPath, ollamaReason string) (string, bool) {
	if a == nil || a.adapterRegistry == nil {
		return "Ollama 匯入失敗：" + ollamaReason + "；adapter registry 尚未初始化，無法改註冊 llama.cpp Adapter", false
	}
	modelID := strings.TrimSpace(job.ModelName)
	if modelID == "" {
		modelID = ggufModelName(ggufPath)
	}
	adapterID := "local-llamacpp-" + sanitizeAdapterID(modelID)
	endpoint := ggufLlamaCPPFallbackEndpoint
	if err := a.adapterRegistry.RegisterLocal(adapterID, "llama.cpp - "+modelID, "G", endpoint, modelID, ggufPath); err != nil {
		return "Ollama 匯入失敗：" + ollamaReason + "；llama.cpp Adapter 註冊也失敗：" + err.Error(), false
	}
	status := adapter_registry.StatusDegraded
	statusNote := "已先加入 llama.cpp Adapter；請啟動 run-breeze-server.cmd，讓 " + endpoint + " 可連線"
	if pingOpenAIModelsEndpoint(endpoint, 800*time.Millisecond) {
		status = adapter_registry.StatusOnline
		statusNote = "已改用正在執行的 llama.cpp endpoint：" + endpoint
	}
	_ = a.adapterRegistry.SetStatus(adapterID, status)
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"adapter_id": adapterID,
			"kind":       "local",
			"fallback":   "llamacpp",
		})
	}
	return "Ollama 匯入失敗：" + ollamaReason + "；" + statusNote, true
}

// downloadGGUF — 串流下載到 appDataRoot/data/models/gguf_downloads/<name>.gguf。
// 先寫 .partial 再 rename；進度每 1% 或每 2 秒（無 Content-Length 時）emit 一次。
// 走 PolicyCloudAPI SafeClient（https only、擋 private IP / metadata）。
func (a *App) downloadGGUF(job GGUFImportJob) (string, error) {
	if err := urlsafe.ValidateURL(job.Source, urlsafe.PolicyCloudAPI); err != nil {
		return "", err
	}
	destDir := filepath.Join(appDataRoot(), "data", "models", "gguf_downloads")
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", err
	}
	dest := filepath.Join(destDir, job.ModelName+".gguf")
	partial := dest + ".partial"

	// timeout 0 = 不設整體 timeout；大檔下載以分鐘計，靠 dialer timeout 顧連線。
	client := urlsafe.NewSafeClient(urlsafe.PolicyCloudAPI, "gguf_download", 0)
	resp, err := client.Get(job.Source)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.OpenFile(partial, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	fail := func(reason error) (string, error) {
		out.Close()
		_ = os.Remove(partial)
		return "", reason
	}

	total := resp.ContentLength
	var written int64
	lastPercent := -1
	var lastEmit time.Time
	buf := make([]byte, 1<<20)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return fail(werr)
			}
			written += int64(n)
			if written > maxGGUFDownloadBytes {
				return fail(fmt.Errorf("檔案超過 %d GB 上限", maxGGUFDownloadBytes>>30))
			}
			if total > 0 {
				if percent := int(written * 100 / total); percent != lastPercent {
					lastPercent = percent
					a.emitGGUFProgress(job, "download", fmt.Sprintf("已下載 %d / %d MB", written>>20, total>>20), percent)
				}
			} else if time.Since(lastEmit) > 2*time.Second {
				lastEmit = time.Now()
				a.emitGGUFProgress(job, "download", fmt.Sprintf("已下載 %d MB", written>>20), -1)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fail(rerr)
		}
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(partial)
		return "", err
	}
	if err := os.Rename(partial, dest); err != nil {
		_ = os.Remove(partial)
		return "", err
	}
	return dest, nil
}

// runOllamaCreate — 產生暫存 Modelfile 後跑 `ollama create <name> -f`，
// stdout/stderr 都掛 parser 發 phase=create 進度。
func (a *App) runOllamaCreate(job GGUFImportJob, ggufPath string) error {
	ollamaPath := resolveOllamaExecutable()
	if ollamaPath == "" {
		return fmt.Errorf("找不到 Ollama 執行檔")
	}

	mf, err := os.CreateTemp("", "haler-modelfile-*.txt")
	if err != nil {
		return fmt.Errorf("建立暫存 Modelfile 失敗：%w", err)
	}
	// Modelfile 的 FROM 用引號包路徑，處理含空白的路徑；ToSlash 讓 Windows 路徑不被當跳脫。
	if _, err := fmt.Fprintf(mf, "FROM %q\n", filepath.ToSlash(ggufPath)); err != nil {
		mf.Close()
		_ = os.Remove(mf.Name())
		return err
	}
	if err := mf.Close(); err != nil {
		_ = os.Remove(mf.Name())
		return err
	}
	defer os.Remove(mf.Name())

	cmd := executil.Command(ollamaPath, "create", job.ModelName, "-f", mf.Name())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("啟動 ollama create 失敗：%w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a.parseCreateStream(job, stdout) }()
	go func() { defer wg.Done(); a.parseCreateStream(job, stderr) }()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ollama create 失敗：%w", err)
	}
	return nil
}

// parseCreateStream — 讀 ollama create 輸出（"transferring model data 45%"、
// "creating new layer"、"success" 等），percent 或狀態行變化時 emit。
// 共用 embedding_binding.go 的 splitOnCRorLF / parsePullPercent。
func (a *App) parseCreateStream(job GGUFImportJob, r io.Reader) {
	scanner := bufio.NewScanner(splitOnCRorLF(r))
	scanner.Buffer(make([]byte, 0, 4096), 64*1024)
	lastLine := ""
	lastPercent := -1
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		percent := parsePullPercent(line)
		// 去抖：percent 沒變且行文相同就不重發（\r 進度行很吵）。
		if percent >= 0 {
			if percent == lastPercent {
				continue
			}
			lastPercent = percent
		} else if line == lastLine {
			continue
		}
		lastLine = line
		a.emitGGUFProgress(job, "create", line, percent)
	}
}

// ─────────────────────────────────────
// 附近搜尋（引用連結建議清單用）
// ─────────────────────────────────────

// GGUFSuggestion — SuggestGGUFFiles 回給前端建議清單的型別。
type GGUFSuggestion struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// SuggestGGUFFiles：以貼上的路徑為中心，往上一層與往下一層資料夾找 .gguf。
// 貼錯檔名 / 貼到資料夾都能撈到附近的正確檔；不存在的路徑會退到最近存在的上層。
// 只給前端建議清單用（preview 失敗後），不影響任何註冊判斷。
func (a *App) SuggestGGUFFiles(raw string) []GGUFSuggestion {
	base := expandUserPath(strings.TrimSpace(raw))
	if base == "" {
		return nil
	}
	// 定位起點資料夾：檔案 → 所在資料夾；不存在 → 往上找最近存在的資料夾（最多 4 層）。
	dir := base
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for i := 0; i < 4; i++ {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
	// 範圍：起點、起點的子資料夾（下一層）、上一層、上一層的子資料夾（兄弟）。
	dirs := []string{dir}
	dirs = append(dirs, listSubdirs(dir, 40)...)
	parent := filepath.Dir(dir)
	if parent != dir {
		dirs = append(dirs, parent)
		dirs = append(dirs, listSubdirs(parent, 40)...)
	}
	return collectGGUF(dirs, 20)
}

// listGGUFWithin — 只搜 dir 本層＋直接子資料夾（folder preview 用，不往上跨層，
// 免得使用者貼一般文件資料夾時被隔壁的模型資料夾誤攔）。
func listGGUFWithin(dir string) []GGUFSuggestion {
	dirs := append([]string{dir}, listSubdirs(dir, 40)...)
	return collectGGUF(dirs, 20)
}

// collectGGUF — 掃各資料夾「本層」的 .gguf 檔（不遞迴），去重、封頂。
func collectGGUF(dirs []string, limit int) []GGUFSuggestion {
	seenDir := map[string]struct{}{}
	seenFile := map[string]struct{}{}
	var out []GGUFSuggestion
	for _, d := range dirs {
		key := strings.ToLower(filepath.Clean(d))
		if _, dup := seenDir[key]; dup {
			continue
		}
		seenDir[key] = struct{}{}
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".gguf") {
				continue
			}
			full := filepath.Join(d, entry.Name())
			fkey := strings.ToLower(full)
			if _, dup := seenFile[fkey]; dup {
				continue
			}
			seenFile[fkey] = struct{}{}
			out = append(out, GGUFSuggestion{Name: entry.Name(), Path: full})
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

// listSubdirs — 列 dir 的直接子資料夾（絕對路徑），略過隱藏資料夾，上限 limit。
func listSubdirs(dir string, limit int) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		out = append(out, filepath.Join(dir, entry.Name()))
		if len(out) >= limit {
			break
		}
	}
	return out
}

// ─────────────────────────────────────
// event helpers（比照 emitEmbeddingEvent 系列）
// ─────────────────────────────────────

func (a *App) emitGGUFEvent(name string, payload map[string]any) {
	if a == nil || a.eventBus == nil {
		return
	}
	a.eventBus.Emit(name, payload)
}

func (a *App) emitGGUFProgress(job GGUFImportJob, phase, status string, percent int) {
	payload := map[string]any{
		"jobId":  job.JobID,
		"model":  job.ModelName,
		"phase":  phase,
		"status": status,
	}
	if percent >= 0 {
		payload["percent"] = percent
	}
	a.emitGGUFEvent(eventbus.EventGGUFImportProgress, payload)
}

func (a *App) emitGGUFFailed(job GGUFImportJob, reason string) {
	a.emitGGUFEvent(eventbus.EventGGUFImportFailed, map[string]any{
		"jobId": job.JobID,
		"model": job.ModelName,
		"error": reason,
	})
}
