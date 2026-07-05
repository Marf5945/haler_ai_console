package main

// app_popout.go — 釘選對話框（popout）多視窗架構，主行程端。
//
// Wails v2 只支援單一原生視窗，因此「真正的 OS 獨立視窗」由同一顆執行檔
// 以 --popout 模式再起的子行程實現（Mac / Windows 行為一致）。
// 子行程透過 127.0.0.1 loopback HTTP + 一次性 token 與主行程溝通：
//
//	主行程（本檔）＝記憶與模型的唯一管家：
//	  - 所有對話讀寫都走 conversationRootForAgent(agentID)，
//	    popout 對話視為 SUB，各自有獨立 talk_full.md，互不污染；
//	    只有 MAIN 保有總覽/調度所有記憶的能力。
//	  - LLM 呼叫（SendCLIMessage / SendAPIMessage）只在主行程執行，
//	    子行程永遠不直接碰模型與檔案。
//
//	子行程（popout_child.go）＝純顯示殼：小視窗 + 輕量聊天 UI。
//
// SEC 考量：server 只綁 127.0.0.1、隨機 port、所有請求驗 X-Popout-Token；
// token 只經由子行程命令列傳遞（本機同使用者可見，等級與現有 sidecar 相同）。

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PopoutInfo 是一個釘選視窗的描述，pin 當下由前端帶入「當前主人格與模型（adapter）」。
type PopoutInfo struct {
	AgentID     string `json:"agent_id"`
	Name        string `json:"name"`
	AdapterID   string `json:"adapter_id"`
	IsAPI       bool   `json:"is_api"`
	PersonaID   string `json:"persona_id"`
	PersonaName string `json:"persona_name"`
	Model       string `json:"model"`
	Locale      string `json:"locale"`
}

type popoutConfigRequest struct {
	Agent     string `json:"agent"`
	PersonaID string `json:"persona_id"`
	AdapterID string `json:"adapter_id"`
	Model     string `json:"model"`
}

type popoutChild struct {
	info   PopoutInfo
	cmd    *exec.Cmd
	sendMu sync.Mutex // 每個 popout 對話各自序列化送訊，彼此可並行
}

type popoutHub struct {
	mu       sync.Mutex
	app      *App
	server   *http.Server
	port     int
	token    string
	children map[string]*popoutChild
}

var popouts = &popoutHub{children: map[string]*popoutChild{}}

// ---------------------------------------------------------------------------
// Wails bindings（主視窗前端呼叫）
// ---------------------------------------------------------------------------

// PinOutConversation 把一個 sub 對話釘選成獨立 OS 視窗。
// main（主haㄌer）不可釘出：只有 MAIN 能調動所有對話記憶。
func (a *App) PinOutConversation(info PopoutInfo) error {
	agentID := strings.TrimSpace(info.AgentID)
	if agentID == "" || agentID == "main" || agentID == "主haㄌer" {
		return fmt.Errorf("main conversation cannot be pinned out")
	}
	// 驗證 agent 存在且有獨立 conversation root（記憶隔離的前提）。
	if _, err := conversationRootForAgent(agentID); err != nil {
		return err
	}

	popouts.mu.Lock()
	defer popouts.mu.Unlock()
	popouts.app = a
	if _, exists := popouts.children[agentID]; exists {
		return fmt.Errorf("conversation already pinned: %s", agentID)
	}
	if err := popouts.ensureServerLocked(); err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	cmd := exec.Command(exe,
		"--popout",
		"--popout-agent="+agentID,
		"--popout-name="+info.Name,
		fmt.Sprintf("--popout-port=%d", popouts.port),
		"--popout-token="+popouts.token,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch popout window: %w", err)
	}

	child := &popoutChild{info: info, cmd: cmd}
	popouts.children[agentID] = child

	// 子行程結束（不論解除釘選、使用者關窗或崩潰）→ 移除並通知主前端回彈。
	go func() {
		_ = cmd.Wait()
		popouts.mu.Lock()
		if popouts.children[agentID] == child {
			delete(popouts.children, agentID)
		}
		popouts.emitChangedLocked()
		popouts.mu.Unlock()
	}()

	popouts.emitChangedLocked()
	return nil
}

// UnpinConversation 由主視窗收回釘選（點擊已釘選的分頁）。
func (a *App) UnpinConversation(agentID string) error {
	popouts.mu.Lock()
	child, ok := popouts.children[strings.TrimSpace(agentID)]
	popouts.mu.Unlock()
	if !ok {
		return nil
	}
	// 直接終止子行程；Wait watcher 會清理狀態並發事件。
	if child.cmd != nil && child.cmd.Process != nil {
		_ = child.cmd.Process.Kill()
	}
	return nil
}

// ListPinnedPopouts 回傳目前所有釘選中的對話（主前端啟動/重整時同步用）。
func (a *App) ListPinnedPopouts() []PopoutInfo {
	popouts.mu.Lock()
	defer popouts.mu.Unlock()
	out := make([]PopoutInfo, 0, len(popouts.children))
	for _, child := range popouts.children {
		out = append(out, child.info)
	}
	return out
}

// shutdownPopouts 主程式關閉時終止所有子視窗，避免孤兒行程。
func (h *popoutHub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, child := range h.children {
		if child.cmd != nil && child.cmd.Process != nil {
			_ = child.cmd.Process.Kill()
		}
		delete(h.children, id)
	}
	if h.server != nil {
		_ = h.server.Close()
		h.server = nil
	}
}

// ---------------------------------------------------------------------------
// loopback HTTP server（子行程 → 主行程）
// ---------------------------------------------------------------------------

func (h *popoutHub) ensureServerLocked() error {
	if h.server != nil {
		return nil
	}
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("popout token: %w", err)
	}
	h.token = hex.EncodeToString(tokenBytes)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("popout listen: %w", err)
	}
	h.port = listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", h.withAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.HandleFunc("/api/state", h.withAuth(h.handleState))
	mux.HandleFunc("/api/config", h.withAuth(h.handleConfig))
	mux.HandleFunc("/api/send", h.withAuth(h.handleSend))
	mux.HandleFunc("/api/unpin", h.withAuth(h.handleUnpin))

	h.server = &http.Server{Handler: mux}
	go func() { _ = h.server.Serve(listener) }()
	return nil
}

func (h *popoutHub) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		token := h.token
		h.mu.Unlock()
		got := r.Header.Get("X-Popout-Token")
		if token == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (h *popoutHub) lookup(agentID string) (*popoutChild, *App) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.children[agentID], h.app
}

func (h *popoutHub) snapshot(agentID string) (*popoutChild, *App, PopoutInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()
	child := h.children[agentID]
	if child == nil {
		return nil, h.app, PopoutInfo{}
	}
	return child, h.app, child.info
}

func writePopoutJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func popoutContainsString(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func popoutStatePayload(app *App, info PopoutInfo, messages []string) map[string]interface{} {
	if app == nil {
		return map[string]interface{}{"messages": messages}
	}

	personas := []map[string]string{}
	if app.settingsService != nil {
		for _, persona := range app.settingsService.State().Personas {
			personas = append(personas, map[string]string{
				"id":       persona.ID,
				"name":     persona.Name,
				"identity": persona.Identity,
			})
			if info.PersonaID == "" && persona.Name == info.PersonaName {
				info.PersonaID = persona.ID
			}
		}
	}

	adapters := []map[string]interface{}{}
	choices := map[string]string{}
	if app.settingsService != nil {
		choices = app.settingsService.AdapterModelChoices()
	}
	if app.adapterRegistry != nil {
		for _, adapter := range app.adapterRegistry.ListAvailable() {
			if !shouldExposeAdapter(adapter) {
				continue
			}
			kind := strings.TrimSpace(adapter.Kind)
			if kind == "" {
				kind = "cli"
			}
			models := app.ListAdapterModelOptions(adapter.ID)
			model := strings.TrimSpace(choices[adapter.ID])
			if model == "" {
				model = strings.TrimSpace(adapter.Model)
			}
			if model != "" && !popoutContainsString(models, model) {
				models = append([]string{model}, models...)
			}
			adapters = append(adapters, map[string]interface{}{
				"id":     adapter.ID,
				"name":   adapter.Name,
				"kind":   kind,
				"status": string(adapter.Status),
				"model":  model,
				"models": models,
			})
		}
	}

	return map[string]interface{}{
		"agent_id":     info.AgentID,
		"name":         info.Name,
		"adapter_id":   info.AdapterID,
		"is_api":       info.IsAPI,
		"persona_id":   info.PersonaID,
		"persona_name": info.PersonaName,
		"model":        info.Model,
		"locale":       info.Locale,
		"personas":     personas,
		"adapters":     adapters,
		"messages":     messages,
	}
}

// GET /api/state?agent=… → 視窗初始資料（含該 agent 自己的對話歷史）。
func (h *popoutHub) handleState(w http.ResponseWriter, r *http.Request) {
	agentID := strings.TrimSpace(r.URL.Query().Get("agent"))
	child, app, info := h.snapshot(agentID)
	if child == nil || app == nil {
		http.Error(w, "not pinned", http.StatusNotFound)
		return
	}
	messages, err := app.GetTalkMessagesForAgent(agentID)
	if err != nil {
		messages = []string{}
	}
	writePopoutJSON(w, popoutStatePayload(app, info, messages))
}

// POST /api/config {agent, persona_id, adapter_id, model} → 更新此 popout 的人格／模型選擇。
func (h *popoutHub) handleConfig(w http.ResponseWriter, r *http.Request) {
	var req popoutConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	agentID := strings.TrimSpace(req.Agent)
	h.mu.Lock()
	child := h.children[agentID]
	app := h.app
	if child == nil || app == nil {
		h.mu.Unlock()
		http.Error(w, "not pinned", http.StatusNotFound)
		return
	}
	next := child.info
	if personaID := strings.TrimSpace(req.PersonaID); personaID != "" && app.settingsService != nil {
		for _, persona := range app.settingsService.State().Personas {
			if persona.ID == personaID {
				next.PersonaID = persona.ID
				next.PersonaName = persona.Name
				break
			}
		}
	}
	if adapterID := strings.TrimSpace(req.AdapterID); adapterID != "" && app.adapterRegistry != nil {
		for _, adapter := range app.adapterRegistry.ListAvailable() {
			if adapter.ID == adapterID && shouldExposeAdapter(adapter) {
				next.AdapterID = adapter.ID
				next.IsAPI = app.isAPIOrLocalAdapter(adapter.ID)
				if strings.TrimSpace(req.Model) == "" {
					next.Model = strings.TrimSpace(adapter.Model)
				}
				break
			}
		}
	}
	if model := strings.TrimSpace(req.Model); model != "" {
		next.Model = model
	}
	child.info = next
	h.emitChangedLocked()
	h.mu.Unlock()

	messages, err := app.GetTalkMessagesForAgent(agentID)
	if err != nil {
		messages = []string{}
	}
	writePopoutJSON(w, popoutStatePayload(app, next, messages))
}

// POST /api/send {agent, text} → 寫入該 agent 專屬記憶 → 走完整對話路由 → 寫回覆 → 回傳。
// 記憶隔離：只碰 conversationRootForAgent(agent) 下的 talk_full.md。
//
// 路由對齊主視窗：這裡走 ExecuteSkillMessage（＝主視窗 composer 一般對話所用的同一個
// 入口），因此獨立視窗也能吃到 skill 判斷 / tool 路由 / 本機＋網路搜尋 / 確認閘，
// 不再是「直通 adapter」的純聊天。ExecuteSkillMessage 內部自行依 adapter 種類
// 選 CLI 或 API 送出，故不需要在這裡分 IsAPI。
//
// 先天分工（照架構）：DAG 多步驟任務仍只在 MAIN 主視窗跑——它需要進度 UI 與
// 全域單一任務鎖（activeTaskRunID），且是事件驅動、事件只發往主視窗；popout 是
// 同步請求／回覆的顯示殼，硬跑 DAG 會與單一任務模型衝突。故 popout 走「單輪完整
// 路由」，這已涵蓋 skill/tool/搜尋；需要多步驟拆解時請於主視窗發起。
//
// renderPopoutDecision 把 ExecuteSkillMessage 的三態決策收斂成 (replyText, replyErr)。
func renderPopoutDecision(dec *SkillExecutionDecision, err error) (replyText, replyErr string) {
	if err != nil {
		return "", err.Error()
	}
	if dec == nil {
		return "", "empty response"
	}
	// auto_execute / no_skill：已實際送出，Response 帶模型回覆或錯誤。
	if dec.Response != nil {
		return dec.Response.Text, dec.Response.Error
	}
	// need_confirm / candidate / review：popout 無確認 UI，把提示原文回給使用者，
	// 讓他知道這步需要回主視窗確認才會執行，而不是靜默什麼都沒發生。
	if msg := strings.TrimSpace(dec.Message); msg != "" {
		return msg, ""
	}
	return "", "empty response"
}

func (h *popoutHub) handleSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agent string `json:"agent"`
		Text  string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	agentID := strings.TrimSpace(req.Agent)
	text := strings.TrimSpace(req.Text)
	child, app := h.lookup(agentID)
	if child == nil || app == nil {
		http.Error(w, "not pinned", http.StatusNotFound)
		return
	}
	if text == "" {
		http.Error(w, "empty text", http.StatusBadRequest)
		return
	}

	child.sendMu.Lock()
	defer child.sendMu.Unlock()
	h.mu.Lock()
	info := child.info
	h.mu.Unlock()
	if info.PersonaID == "" && app.settingsService != nil {
		for _, persona := range app.settingsService.State().Personas {
			if persona.Name == info.PersonaName {
				info.PersonaID = persona.ID
				break
			}
		}
	}

	if err := app.AppendTalkEntryForAgent(agentID, "user", text); err != nil {
		writePopoutJSON(w, map[string]string{"error": err.Error()})
		return
	}

	// traceID 必須是「非 task 前綴」，否則 isTaskProgressTraceID 會判成內部任務而
	// 跳過整套路由；"popout-" 前綴安全。sessionID 各 agent 獨立，記憶／續接互不污染。
	traceID := fmt.Sprintf("popout-%s-%d", agentID, time.Now().UnixNano())
	sessionID := "popout-" + agentID

	// 走與主視窗一般對話相同的入口：skill 判斷 → tool 路由 → 搜尋 → 確認閘 →
	// 依 adapter 種類自動走 CLI 或 API。
	dec, err := app.executeSkillMessageWithOverrides(info.AdapterID, sessionID, text, traceID, info.PersonaID, info.Model)
	replyText, replyErr := renderPopoutDecision(dec, err)

	if replyText != "" {
		_ = app.AppendTalkEntryForAgent(agentID, "assistant", replyText)
	}
	writePopoutJSON(w, map[string]string{"reply": replyText, "error": replyErr})
}

// POST /api/unpin {agent} → 子視窗按了圖釘或關窗：先移除記錄再通知主前端回彈。
// 子行程送出此請求後會自行結束；這裡不 Kill，避免競態。
func (h *popoutHub) handleUnpin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agent string `json:"agent"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	agentID := strings.TrimSpace(req.Agent)
	h.mu.Lock()
	delete(h.children, agentID)
	h.emitChangedLocked()
	h.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// emitChangedLocked 通知主前端釘選清單變了（回彈分頁、重新載入該 agent 記憶）。
// 呼叫方必須已持有 h.mu。
func (h *popoutHub) emitChangedLocked() {
	if h.app == nil || h.app.ctx == nil {
		return
	}
	list := make([]PopoutInfo, 0, len(h.children))
	for _, child := range h.children {
		list = append(list, child.info)
	}
	runtime.EventsEmit(h.app.ctx, "popout:changed", list)
}
