// embedding_binding.go — Phase B M2/M3：embedding 設定 + first-drop picker UX 的 Wails binding。
//
// 設計原則：
//   - 不抽 helper struct；薄薄一層 forwarding。
//   - Dimension 不曝光成 Setter binding——只能由 backend 在 ingest 時量到後寫回。
//   - PullEmbedModel 走 goroutine + eventbus：前端不 await 5 分鐘。
//   - 進度解析 fail-soft：parse 不到也照樣 emit raw status；不要把進度解析失敗變成 pull 失敗。
package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"ui_console/shared/eventbus"
	"ui_console/shared/executil"
	"ui_console/shared/settings"
)

// EmbedModelInfo — ListInstalledEmbedModels 回給前端的型別。
// 故意輕量：M3 不承諾 SizeBytes 之類欄位（scanOllamaModels 沒回，補要改 parser）；M4 再加。
type EmbedModelInfo struct {
	ID         string `json:"id"`         // 例 "nomic-embed-text:latest"
	ProviderID string `json:"providerId"` // "ollama"
	Label      string `json:"label"`      // 顯示字串（M3 用 ID 即可）
}

// EmbedPullJob — PullEmbedModel 立即回的票根。前端拿這個跟 event payload 對得起來。
type EmbedPullJob struct {
	JobID   string `json:"jobId"`
	ModelID string `json:"modelId"`
}

// ── pull job 去重 ──
// 避免使用者連點「下載」造成同一個 model 跑多個 ollama pull。
// key = modelID；存在就代表正在跑。
var (
	pullJobMu     sync.Mutex
	pullJobActive = map[string]struct{}{}
)

// ── 名稱 heuristic ──
// 不在這份清單但使用者手動輸入的也接受——modal 有「下載新 model」輸入框。
var embedModelKeywords = []string{
	"embed", "embedding", "embeddinggemma",
	"nomic", "mxbai", "bge", "minilm", "e5",
	"jina", "snowflake", "gte",
}

// ─────────────────────────────────────
// M2 既有
// ─────────────────────────────────────

// GetEmbeddingConfig 回目前的 embedding 設定。
func (a *App) GetEmbeddingConfig() settings.EmbeddingConfig {
	if a.settingsService == nil {
		return settings.EmbeddingConfig{}
	}
	return a.settingsService.EmbeddingConfig()
}

// SetEmbeddingConfig 設定 provider + model。Dimension 不可由此寫入。
// 設了 model 會清掉 PickerDismissed=false（自動讓「跳過狀態」失效）。
func (a *App) SetEmbeddingConfig(providerID, modelID string) error {
	if a.settingsService == nil {
		return nil
	}
	a.settingsService.SaveEmbeddingConfig(
		strings.TrimSpace(providerID),
		strings.TrimSpace(modelID),
	)
	return nil
}

// ─────────────────────────────────────
// M3 新增
// ─────────────────────────────────────

// DismissEmbeddingPicker：使用者在 modal 點「跳過」時呼叫。下次拖檔不再彈。
func (a *App) DismissEmbeddingPicker() error {
	if a.settingsService == nil {
		return nil
	}
	a.settingsService.DismissEmbeddingPicker()
	return nil
}

// ListInstalledEmbedModels：掃本機 ollama 已裝 model，過濾名字符合 embed heuristic 的。
// modal 用來填區塊 A。
func (a *App) ListInstalledEmbedModels() []EmbedModelInfo {
	all := scanOllamaModels()
	out := make([]EmbedModelInfo, 0, len(all))
	for _, m := range all {
		if !looksLikeEmbedModel(m.ID) {
			continue
		}
		out = append(out, EmbedModelInfo{
			ID:         m.ID,
			ProviderID: "ollama",
			Label:      m.ID,
		})
	}
	return out
}

// looksLikeEmbedModel — 名字 lowercase 後若含 keyword 就算（heuristic 一定會漏，輸入框補位）。
func looksLikeEmbedModel(id string) bool {
	low := strings.ToLower(id)
	for _, kw := range embedModelKeywords {
		if strings.Contains(low, kw) {
			return true
		}
	}
	return false
}

// PullEmbedModel：背景 spawn `ollama pull <model>`，進度走 eventbus；立刻回 jobID。
// 同一 modelID 已在跑 → 回 error 防止重複觸發。
func (a *App) PullEmbedModel(modelID string) (EmbedPullJob, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return EmbedPullJob{}, fmt.Errorf("embedding: modelID required")
	}
	pullJobMu.Lock()
	if _, busy := pullJobActive[modelID]; busy {
		pullJobMu.Unlock()
		return EmbedPullJob{}, fmt.Errorf("embedding: pull already in progress for %s", modelID)
	}
	pullJobActive[modelID] = struct{}{}
	pullJobMu.Unlock()

	job := EmbedPullJob{
		JobID:   fmt.Sprintf("pull-%d", time.Now().UnixNano()),
		ModelID: modelID,
	}

	go a.runOllamaPull(job)
	return job, nil
}

// runOllamaPull — 背景 goroutine；讀 ollama pull 的 stdout/stderr，
// best-effort 解析百分比，emit progress；結束 emit done/failed。
func (a *App) runOllamaPull(job EmbedPullJob) {
	defer func() {
		pullJobMu.Lock()
		delete(pullJobActive, job.ModelID)
		pullJobMu.Unlock()
	}()

	ollamaPath := resolveOllamaExecutable()
	if ollamaPath == "" {
		a.emitEmbeddingFailed(job, "ollama not found on PATH")
		return
	}

	a.emitEmbeddingEvent(eventbus.EventEmbeddingPullStarted, map[string]any{
		"jobId":   job.JobID,
		"modelId": job.ModelID,
	})

	cmd := executil.Command(ollamaPath, "pull", job.ModelID)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		a.emitEmbeddingFailed(job, "stdout pipe: "+err.Error())
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		a.emitEmbeddingFailed(job, "stderr pipe: "+err.Error())
		return
	}
	if err := cmd.Start(); err != nil {
		a.emitEmbeddingFailed(job, "start ollama: "+err.Error())
		return
	}

	// 並行讀 stdout / stderr，兩邊都跑 parser 發 progress。
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a.parsePullStream(job, stdout) }()
	go func() { defer wg.Done(); a.parsePullStream(job, stderr) }()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		a.emitEmbeddingFailed(job, "ollama exit: "+err.Error())
		return
	}
	a.emitEmbeddingEvent(eventbus.EventEmbeddingPullDone, map[string]any{
		"jobId":   job.JobID,
		"modelId": job.ModelID,
	})
}

// parsePullStream — 讀一條 stream，best-effort 找百分比；解析不到也照樣 emit status。
// ollama pull 的輸出兩種格式皆要兼容：
//   - "pulling abc...  25%"（換行 / \r 更新）
//   - "downloading sha256:... 100%"
//   - 普通 status line（"verifying digest"、"success" 等）
func (a *App) parsePullStream(job EmbedPullJob, r io.Reader) {
	scanner := bufio.NewScanner(splitOnCRorLF(r))
	scanner.Buffer(make([]byte, 0, 4096), 64*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		percent := parsePullPercent(line)
		payload := map[string]any{
			"jobId":   job.JobID,
			"modelId": job.ModelID,
			"status":  line,
		}
		if percent >= 0 {
			payload["percent"] = percent
		}
		a.emitEmbeddingEvent(eventbus.EventEmbeddingPullProgress, payload)
	}
}

// pullPercentPattern 找 "数字%"，例 " 12%"、"100%"。
var pullPercentPattern = regexp.MustCompile(`(\d{1,3})\s*%`)

// parsePullPercent — 找到回 0..100；找不到回 -1。Best-effort，不爆 error。
func parsePullPercent(line string) int {
	m := pullPercentPattern.FindStringSubmatch(line)
	if len(m) < 2 {
		return -1
	}
	n := 0
	for _, c := range m[1] {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	if n < 0 || n > 100 {
		return -1
	}
	return n
}

// splitOnCRorLF — 把 io.Reader 變成「以 \n 或 \r 為界」的 scanner，
// 處理 ollama 用 \r 更新進度的特殊輸出。
func splitOnCRorLF(r io.Reader) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				// 把 \r 轉成 \n 讓 scanner 認得；保留多餘空行 scanner 會跳。
				for i := 0; i < n; i++ {
					if buf[i] == '\r' {
						buf[i] = '\n'
					}
				}
				if _, werr := pw.Write(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return pr
}

// emitEmbeddingEvent — 安全發 event；eventBus 為 nil 時靜默 no-op。
func (a *App) emitEmbeddingEvent(name string, payload map[string]any) {
	if a == nil || a.eventBus == nil {
		return
	}
	a.eventBus.Emit(name, payload)
}

// emitEmbeddingFailed — 統一 failed event payload 格式。
func (a *App) emitEmbeddingFailed(job EmbedPullJob, reason string) {
	a.emitEmbeddingEvent(eventbus.EventEmbeddingPullFailed, map[string]any{
		"jobId":   job.JobID,
		"modelId": job.ModelID,
		"error":   reason,
	})
}

// ─────────────────────────────────────
// M3+ Ollama daemon 偵測 / 喚醒
// ─────────────────────────────────────

// OllamaState — modal 用來決定要顯示「沒裝 / 沒啟動 / 一切正常」哪一種提示。
type OllamaState struct {
	BinaryFound   bool   `json:"binaryFound"`
	DaemonRunning bool   `json:"daemonRunning"`
	BinaryPath    string `json:"binaryPath,omitempty"`
}

// DetectOllamaState — 給 modal 開啟時呼叫，回 binary / daemon 三態。
//
// 不阻塞太久：ping 只給 800ms timeout，跟 wakeOllamaAdapter 同個 budget。
func (a *App) DetectOllamaState() OllamaState {
	binPath := resolveOllamaExecutable()
	state := OllamaState{
		BinaryFound: binPath != "",
		BinaryPath:  binPath,
	}
	if !state.BinaryFound {
		return state
	}
	state.DaemonRunning = pingOllamaTags("http://localhost:11434", 800*time.Millisecond)
	return state
}

// WakeOllamaDaemon — modal 上點「喚醒」按鈕時呼叫。
// 直接委派給共用核心 wakeOllamaDaemon；不需要 adapter status 更新。
func (a *App) WakeOllamaDaemon() error {
	return wakeOllamaDaemon("", "")
}
