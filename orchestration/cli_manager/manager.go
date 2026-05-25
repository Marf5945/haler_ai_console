// Package cli_manager 管理 Node.js Sidecar 程序的生命週期與 IPC 通訊。
//
// 對應 v3.4.0 §2 Mode B Sidecar 架構：
//   - Go Controller 負責 spawn / 監控 / 關閉 Node 行程
//   - stdin/stdout JSON-RPC 作為 IPC 通道
//   - 5MB 防爆緩衝區防止 CLI 輸出 OOM
//
// #I-801: Sidecar 生命週期管理
// #I-802: stdio Pipe 與防爆緩衝區
package cli_manager

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/debugtrace"
)

// maxScanSize 是 bufio.Scanner 的最大 token 大小（5MB）。
// 超過此限制的 CLI 輸出會被截斷，防止 OOM。
const maxScanSize = 5 * 1024 * 1024

// shutdownTimeout 是 SIGTERM 後等待優雅關閉的時限，逾時則 SIGKILL。
const shutdownTimeout = 5 * time.Second

// SidecarState 描述 Sidecar 行程的當前狀態。
type SidecarState string

const (
	StateIdle     SidecarState = "idle"     // 未啟動
	StateStarting SidecarState = "starting" // 正在啟動
	StateRunning  SidecarState = "running"  // 正常運行中
	StateStopping SidecarState = "stopping" // 正在關閉
	StateCrashed  SidecarState = "crashed"  // 非預期退出
)

// IPCRequest 是 Go→Node 的 JSON-RPC 請求格式。
type IPCRequest struct {
	ID     string      `json:"id"`     // 請求 ID，用於匹配回應
	Method string      `json:"method"` // 呼叫方法名稱
	Params interface{} `json:"params"` // 方法參數（JSON-serializable）
}

// IPCResponse 是 Node→Go 的 JSON-RPC 回應格式。
type IPCResponse struct {
	ID     string          `json:"id"`               // 對應 IPCRequest.ID
	Result json.RawMessage `json:"result,omitempty"` // 成功結果
	Error  string          `json:"error,omitempty"`  // 錯誤描述
}

// CrashHandler 在 Sidecar 非預期退出時被呼叫。
// exitCode 為程序退出碼，err 為 cmd.Wait() 回傳的錯誤。
type CrashHandler func(exitCode int, err error)

type pendingCallMeta struct {
	method  string
	traceID string
}

type pendingCallSnapshot struct {
	ID      string `json:"id"`
	Method  string `json:"method"`
	TraceID string `json:"trace_id"`
}

// SidecarManager 管理單一 Node.js Sidecar 程序。
// 並發安全：所有公開方法持有 mu 鎖。
type SidecarManager struct {
	mu           sync.Mutex
	state        SidecarState
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	cancel       context.CancelFunc          // 呼叫後觸發 context 取消 → 行程終止
	pendingCalls map[string]chan IPCResponse // id → 等待中的回應 channel
	pendingMeta  map[string]pendingCallMeta  // id → trace/method diagnostics
	onCrash      CrashHandler                // 可選的崩潰回呼
	scriptPath   string                      // Node 腳本路徑
}

// NewSidecarManager 建立管理器。scriptPath 為 Node.js 入口腳本。
func NewSidecarManager(scriptPath string) *SidecarManager {
	return &SidecarManager{
		state:        StateIdle,
		pendingCalls: make(map[string]chan IPCResponse),
		pendingMeta:  make(map[string]pendingCallMeta),
		scriptPath:   scriptPath,
	}
}

// SetCrashHandler 設定崩潰回呼。必須在 Start 前呼叫。
func (m *SidecarManager) SetCrashHandler(handler CrashHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onCrash = handler
}

// State 回傳目前的 Sidecar 狀態。
func (m *SidecarManager) State() SidecarState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Start 背景啟動 Node.js Sidecar。
// 使用 context.WithCancel 包裹，確保 Stop/app 關閉時可中止。
func (m *SidecarManager) Start(parentCtx context.Context) error {
	m.mu.Lock()
	if m.state == StateRunning || m.state == StateStarting {
		m.mu.Unlock()
		return fmt.Errorf("cli_manager: sidecar already %s", m.state)
	}
	m.state = StateStarting
	m.mu.Unlock()

	ctx, cancel := context.WithCancel(parentCtx)

	// #I-801: 以 exec.CommandContext 啟動 Node，context 取消時自動發 signal。
	// macOS .app 打包後 PATH 通常只有 /usr/bin:/bin，需要主動搜尋 node 的完整路徑。
	nodeBin := findNodeBinary()
	log.Printf("cli_manager: resolved node binary: %s", nodeBin)
	log.Printf("cli_manager: sidecar script: %s", m.scriptPath)
	cmd := exec.CommandContext(ctx, nodeBin, m.scriptPath)

	// 將 node 所在目錄與常用 CLI 路徑注入 child process 的 PATH，
	// 確保 sidecar 內部 spawn CLI 時也能找到正確的執行檔。
	cmd.Env = buildSidecarEnv(nodeBin)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		m.setStateLocked(StateIdle)
		return fmt.Errorf("cli_manager: stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		m.setStateLocked(StateIdle)
		return fmt.Errorf("cli_manager: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		m.setStateLocked(StateIdle)
		log.Printf("cli_manager: ERROR — failed to start sidecar: %v", err)
		return fmt.Errorf("cli_manager: start: %w", err)
	}

	log.Printf("cli_manager: sidecar started successfully (pid=%d)", cmd.Process.Pid)

	m.mu.Lock()
	m.cmd = cmd
	m.stdin = stdinPipe
	m.cancel = cancel
	m.state = StateRunning
	m.mu.Unlock()

	// #I-802: 獨立 goroutine 讀取 stdout，bufio.Scanner 限制 5MB
	go m.readLoop(stdoutPipe)

	// #I-801/#I-805: 監聽行程退出，處理正常關閉與崩潰
	go m.waitLoop(cmd)

	return nil
}

func findNodeBinary() string {
	if path, err := exec.LookPath("node"); err == nil {
		return path
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		"/opt/homebrew/bin/node",
		"/usr/local/bin/node",
		"/usr/bin/node",
	}
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "node"),
			filepath.Join(home, ".nvm", "current", "bin", "node"),
		)
		if entries, err := os.ReadDir(filepath.Join(home, ".nvm", "versions", "node")); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					candidates = append(candidates, filepath.Join(home, ".nvm", "versions", "node", entry.Name(), "bin", "node"))
				}
			}
		}
		if entries, err := os.ReadDir(filepath.Join(home, ".local", "share", "fnm", "node-versions")); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					candidates = append(candidates, filepath.Join(home, ".local", "share", "fnm", "node-versions", entry.Name(), "installation", "bin", "node"))
				}
			}
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			log.Printf("cli_manager: found node at candidate path: %s", candidate)
			return candidate
		}
	}
	// 最後 fallback：回傳 "node"，讓 exec 試 PATH（幾乎一定會失敗）
	log.Printf("cli_manager: WARNING — node not found in any known path, falling back to bare \"node\"")
	return "node"
}

// buildSidecarEnv 為 sidecar child process 建構環境變數。
// 核心目的：把 node 所在目錄、Homebrew、npm global 等常見路徑注入 PATH，
// 讓 sidecar 內部 spawn CLI（claude, gemini 等）時不會因為 PATH 太短而失敗。
func buildSidecarEnv(nodeBin string) []string {
	env := os.Environ()

	// 收集要補進 PATH 的額外目錄
	extraDirs := []string{
		filepath.Dir(nodeBin), // node 本身所在目錄
	}

	// macOS .app 啟動時 PATH 非常短，需要主動補上常見的 CLI 安裝路徑
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		extraDirs = append(extraDirs,
			"/opt/homebrew/bin", // Apple Silicon Homebrew
			"/usr/local/bin",    // Intel Homebrew / 手動安裝
		)
		if home, _ := os.UserHomeDir(); home != "" {
			extraDirs = append(extraDirs,
				filepath.Join(home, ".local", "bin"),      // pipx, uv 等
				filepath.Join(home, ".cargo", "bin"),      // Rust CLI
				filepath.Join(home, ".bun", "bin"),        // bun global
				filepath.Join(home, "bin"),                // 使用者自訂
				filepath.Join(home, ".npm-global", "bin"), // npm config prefix
			)
		}
	}

	// 從現有 env 中找到 PATH 並合併
	pathKey := "PATH"
	merged := false
	for i, kv := range env {
		if strings.HasPrefix(kv, pathKey+"=") {
			existing := strings.TrimPrefix(kv, pathKey+"=")
			// 把額外目錄加到 PATH 前面，優先使用
			env[i] = pathKey + "=" + strings.Join(extraDirs, string(os.PathListSeparator)) +
				string(os.PathListSeparator) + existing
			merged = true
			break
		}
	}
	if !merged {
		env = append(env, pathKey+"="+strings.Join(extraDirs, string(os.PathListSeparator)))
	}
	return env
}

// Stop 優雅關閉 Sidecar：cancel context → SIGTERM → 等 5s → SIGKILL。
func (m *SidecarManager) Stop() error {
	m.mu.Lock()
	if m.state != StateRunning {
		m.mu.Unlock()
		return nil
	}
	m.state = StateStopping
	cancelFn := m.cancel
	cmd := m.cmd
	m.mu.Unlock()

	// 觸發 context 取消（CommandContext 會發 SIGKILL，但我們先嘗試優雅關閉）
	if cancelFn != nil {
		cancelFn()
	}

	// 等待行程結束，超時則強制殺
	done := make(chan struct{})
	go func() {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Wait()
		}
		close(done)
	}()

	select {
	case <-done:
		// 正常結束
	case <-time.After(shutdownTimeout):
		// 超時：SIGKILL 拔管
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}

	m.setStateLocked(StateIdle)
	return nil
}

// Call 發送 JSON-RPC 請求並等待回應（阻塞式）。
// timeout 為等待回應的上限，超時回傳 error。
func (m *SidecarManager) Call(method string, params interface{}, timeout time.Duration) (*IPCResponse, error) {
	traceID := traceIDFromParams(params)
	m.mu.Lock()
	if m.state != StateRunning {
		m.mu.Unlock()
		// DEBUG_TRACE_REMOVE: Sidecar manager rejected the IPC call.
		debugtrace.Record("go.SidecarManager.call.notRunning", traceID, map[string]interface{}{
			"method": method,
			"state":  m.state,
		})
		return nil, fmt.Errorf("cli_manager: sidecar not running (state=%s)", m.state)
	}

	// 產生唯一 ID
	id := fmt.Sprintf("rpc-%d", time.Now().UnixNano())
	ch := make(chan IPCResponse, 1)
	m.pendingCalls[id] = ch
	m.pendingMeta[id] = pendingCallMeta{method: method, traceID: traceID}
	stdinWriter := m.stdin
	m.mu.Unlock()

	// 將 request 寫入 stdin（JSON + newline）
	req := IPCRequest{ID: id, Method: method, Params: params}
	data, err := json.Marshal(req)
	if err != nil {
		m.removePending(id)
		return nil, fmt.Errorf("cli_manager: marshal request: %w", err)
	}
	data = append(data, '\n')
	// DEBUG_TRACE_REMOVE: Shows the newline-delimited JSON sent into Node stdin.
	debugtrace.Record("go.SidecarManager.stdin.write", traceID, map[string]interface{}{
		"rpc_id":  id,
		"method":  method,
		"payload": string(data),
	})

	if _, err := stdinWriter.Write(data); err != nil {
		m.removePending(id)
		// DEBUG_TRACE_REMOVE: Failed to write the IPC request to Node.
		debugtrace.Record("go.SidecarManager.stdin.error", traceID, map[string]interface{}{
			"rpc_id": id,
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("cli_manager: write stdin: %w", err)
	}

	// 等待回應或超時
	select {
	case resp := <-ch:
		// DEBUG_TRACE_REMOVE: Raw JSON-RPC response received from Node stdout.
		debugtrace.Record("go.SidecarManager.response", traceID, map[string]interface{}{
			"rpc_id": id,
			"error":  resp.Error,
			"result": string(resp.Result),
		})
		return &resp, nil
	case <-time.After(timeout):
		pending := m.pendingSnapshot()
		m.removePending(id)
		// DEBUG_TRACE_REMOVE: IPC timeout.
		debugtrace.Record("go.SidecarManager.timeout", traceID, map[string]interface{}{
			"rpc_id":         id,
			"method":         method,
			"timeout":        timeout.String(),
			"pending_before": pending,
		})
		return nil, fmt.Errorf("cli_manager: call %q timed out after %s", method, timeout)
	}
}

// DEBUG_TRACE_REMOVE: Pulls trace_id from generic IPC params without coupling the
// manager to a concrete request struct. Remove with the rest of debug tracing.
func traceIDFromParams(params interface{}) string {
	data, err := json.Marshal(params)
	if err != nil {
		return ""
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return ""
	}
	if traceID, ok := decoded["trace_id"].(string); ok {
		return traceID
	}
	return ""
}

// --- 內部 goroutine ---

// readLoop 在獨立 goroutine 中讀取 Node stdout。
// #I-802: MaxScanTokenSize = 5MB，超過的行會被截斷。
func (m *SidecarManager) readLoop(r io.Reader) {
	debugtrace.Record("go.SidecarManager.readLoop.started", "", map[string]interface{}{})
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)  // 初始 64KB
	scanner.Buffer(buf, maxScanSize) // 最大 5MB

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		rawLine := string(line)
		debugtrace.Record("go.SidecarManager.readLoop.line", "", map[string]interface{}{
			"bytes":        len(line),
			"preview":      previewString(rawLine, 500),
			"pending_now":  m.pendingSnapshot(),
			"scanner_size": maxScanSize,
		})

		var resp IPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			debugtrace.Record("go.SidecarManager.readLoop.nonJSON", "", map[string]interface{}{
				"error":       err.Error(),
				"bytes":       len(line),
				"line":        rawLine,
				"pending_now": m.pendingSnapshot(),
			})
			continue // 非 JSON 行靜默跳過（可能是 stderr 混入）
		}

		// 匹配 pending call
		m.mu.Lock()
		ch, ok := m.pendingCalls[resp.ID]
		meta := m.pendingMeta[resp.ID]
		lineTraceID := meta.traceID
		if ok {
			delete(m.pendingCalls, resp.ID)
			delete(m.pendingMeta, resp.ID)
		}
		pendingAfter := m.pendingSnapshotLocked()
		m.mu.Unlock()

		if lineTraceID != "" {
			debugtrace.Record("go.SidecarManager.readLoop.responseLine", lineTraceID, map[string]interface{}{
				"rpc_id": resp.ID,
				"bytes":  len(line),
				"error":  resp.Error,
				"result": string(resp.Result),
			})
		}

		if ok {
			debugtrace.Record("go.SidecarManager.readLoop.matched", meta.traceID, map[string]interface{}{
				"rpc_id":        resp.ID,
				"method":        meta.method,
				"error":         resp.Error,
				"result":        string(resp.Result),
				"pending_after": pendingAfter,
			})
			ch <- resp
		} else {
			debugtrace.Record("go.SidecarManager.readLoop.unmatched", "", map[string]interface{}{
				"rpc_id":        resp.ID,
				"error":         resp.Error,
				"result":        string(resp.Result),
				"pending_after": pendingAfter,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		debugtrace.Record("go.SidecarManager.readLoop.error", "", map[string]interface{}{
			"error":       err.Error(),
			"pending_now": m.pendingSnapshot(),
		})
	} else {
		debugtrace.Record("go.SidecarManager.readLoop.closed", "", map[string]interface{}{
			"pending_now": m.pendingSnapshot(),
		})
	}
}

// waitLoop 監聽行程退出。正常關閉（StateStopping）不觸發 crash。
// #I-805: 非預期退出觸發 CrashHandler。
func (m *SidecarManager) waitLoop(cmd *exec.Cmd) {
	waitErr := cmd.Wait()

	m.mu.Lock()
	prevState := m.state
	handler := m.onCrash
	m.mu.Unlock()

	if prevState == StateStopping || prevState == StateIdle {
		// 正常關閉流程，不觸發 crash
		m.setStateLocked(StateIdle)
		return
	}

	// 非預期退出 → 標記為 crashed
	m.setStateLocked(StateCrashed)

	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	if handler != nil {
		handler(exitCode, waitErr)
	}
}

// --- helpers ---

func (m *SidecarManager) setStateLocked(s SidecarState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = s
}

func (m *SidecarManager) removePending(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pendingCalls, id)
	delete(m.pendingMeta, id)
}

func (m *SidecarManager) pendingSnapshot() []pendingCallSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pendingSnapshotLocked()
}

func (m *SidecarManager) pendingSnapshotLocked() []pendingCallSnapshot {
	pending := make([]pendingCallSnapshot, 0, len(m.pendingCalls))
	for id := range m.pendingCalls {
		meta := m.pendingMeta[id]
		pending = append(pending, pendingCallSnapshot{
			ID:      id,
			Method:  meta.method,
			TraceID: meta.traceID,
		})
	}
	return pending
}

func previewString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}
