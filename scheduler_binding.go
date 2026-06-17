// scheduler_binding.go — Wails 前端綁定方法，暴露排程器 CRUD 給前端。
//
// 所有方法為 App 的 exported method，Wails 會自動產生 TypeScript 綁定檔。
// 前端透過 wailsjs/go/main/App.XXX() 呼叫這些方法。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/scheduler"
)

// SchedulerClockSnapshot 描述排程器目前使用的本機系統時間。
type SchedulerClockSnapshot struct {
	Now           string `json:"now"`
	LocalTime     string `json:"local_time"`
	UnixMillis    int64  `json:"unix_millis"`
	Timezone      string `json:"timezone"`
	UTCOffset     string `json:"utc_offset"`
	UTCOffsetMins int    `json:"utc_offset_minutes"`
}

// SchedulerDraftNormalization 是模型對排程的完整規劃／判斷結果。
// 空欄位（或 0）表示保持原值或無資訊；前端只套用非空欄位。
//
// 此結構同時用於：
//   - json.Unmarshal 解析「模型輸出的 JSON」（欄位 tag 即為模型須輸出的 key）。
//   - Wails 回傳給前端（前端依同樣的 key 讀取）。
type SchedulerDraftNormalization struct {
	// Intent 是模型對使用者意圖的判斷：create / update / open / none。
	Intent string `json:"intent"`
	// Title 為排程短標題（動作名，已去除指令鷹架字）。
	Title string `json:"title"`
	// Summary 為一句話說明。
	Summary string `json:"summary"`
	// TimeText 為使用者講的時間自然語言（模型不確定 cron 時改填此欄，由前端換算）。
	TimeText string `json:"time_text"`
	// CronExpr 為模型直接推得的標準五欄位 cron 或 @daily 等快捷字；
	// 後端會用 scheduler.ParseCron 驗證，不合法時清空改走 TimeText 後援。
	CronExpr string `json:"cron_expr"`
	// ActionText 為排程實際要執行的動作描述。
	ActionText string `json:"action_text"`
	// Question 為仍缺關鍵資訊時的追問句。
	Question string `json:"question"`
	// TargetJobNo 為 intent=update 時，對應既有排程清單的 no（1-based）；否則為 0。
	TargetJobNo int `json:"target_job_no"`
	// Confidence 為模型自評信心：high / medium / low。
	Confidence string `json:"confidence"`
	Raw        string `json:"raw,omitempty"`
}

// GetSchedulerClock 回傳後端排程器所依據的本機系統時間。
func (a *App) GetSchedulerClock() SchedulerClockSnapshot {
	now := time.Now()
	zoneName, offsetSeconds := now.Zone()
	return SchedulerClockSnapshot{
		Now:           now.Format(time.RFC3339),
		LocalTime:     now.Format("2006-01-02 15:04:05"),
		UnixMillis:    now.UnixMilli(),
		Timezone:      zoneName,
		UTCOffset:     formatUTCOffset(offsetSeconds),
		UTCOffsetMins: offsetSeconds / 60,
	}
}

// NormalizeSchedulerDraft 用目前模型把「排程草稿 + 使用者補答/修正」正規化成 UI 欄位。
// 這是排程專用的窄語意入口：不走工具路由、不寫記憶、不執行任何動作。
func (a *App) NormalizeSchedulerDraft(adapterID, sessionID, draftJSON, userText, traceID string) (*SchedulerDraftNormalization, error) {
	if a == nil || a.cliAdapter == nil {
		return nil, fmt.Errorf("模型 adapter 尚未初始化")
	}
	adapterID = strings.TrimSpace(adapterID)
	if adapterID == "" {
		adapterID = a.defaultSkillExecutionAdapterID()
	}
	cliPath := ""
	if adapterID != "" && a.adapterRegistry != nil {
		if resolved, err := a.adapterRegistry.ResolveExecutable(adapterID); err == nil {
			cliPath = resolved
		}
	}
	if isOllamaPromptCLI(adapterID, cliPath) {
		return nil, fmt.Errorf("Ollama 不能用 CLI adapter 方式解析排程草稿")
	}
	if err := a.ensureSidecarRunning(); err != nil {
		return nil, err
	}
	prompt := buildSchedulerDraftNormalizePrompt(draftJSON, userText)
	recordPromptSynthesisTrace("go.scheduler.normalize_synthesis", traceID, prompt, map[string]interface{}{
		"adapter_id":         adapterID,
		"session_id":         sessionID,
		"draft_json_len":     len([]rune(draftJSON)),
		"draft_json_preview": truncateRunes(draftJSON, 4000),
		"user_reply_len":     len([]rune(userText)),
		"user_reply_preview": truncateRunes(userText, 1200),
		"continuity_key":     conversationContinuityKey("scheduler-normalize", sessionID),
	})
	resp, err := a.cliAdapter.SendMessage(skill_step.CLIMessageOptions{
		AdapterID:      adapterID,
		CLIPath:        cliPath,
		SessionID:      sessionID,
		UserText:       prompt,
		ContinuityKey:  conversationContinuityKey("scheduler-normalize", sessionID),
		TraceID:        traceID,
		SkipContinuity: true,
	})
	debugtrace.Record("go.scheduler.normalize", traceID, map[string]interface{}{
		"text":  resp.Text,
		"error": resp.Error,
		"err":   errorString(err),
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}
	out := parseSchedulerDraftNormalization(resp.Text)
	out.Raw = resp.Text
	if out.Intent == "" && out.Title == "" && out.Summary == "" &&
		out.TimeText == "" && out.CronExpr == "" && out.ActionText == "" && out.Question == "" {
		return nil, fmt.Errorf("模型沒有回傳可用的排程欄位")
	}
	return out, nil
}

func buildSchedulerDraftNormalizePrompt(draftJSON, userText string) string {
	return strings.Join([]string{
		"你是排程規劃器。只輸出一個 JSON object，不要 markdown，不要任何解釋文字。",
		"context 是一段 JSON，可能含：current_draft（目前草稿）、phase（start/collecting/confirm）、mode、jobs（既有排程清單，每筆有 no/name/cron_expr/action）、target_job、history（先前對話）。",
		"任務：讀懂 user_reply，判斷使用者意圖，並把程式需要的欄位都填好（程式會直接用這些欄位建立或修改排程）。",
		"必須輸出的欄位：",
		"  intent：create=要新建排程；update=要修改既有排程；open=只是想看排程清單；none=與排程無關。",
		"  title：排程短標題（用動作命名，不要含「幫我、設個、提醒我、標題變成、改成」這類指令鷹架字）。",
		"  summary：一句話說明這個排程會做什麼。",
		"  cron_expr：標準五欄位 cron「分 時 日 月 週」（週日=0、週一=1…週六=6）或快捷字 @daily/@hourly/@weekly/@monthly。能從 user_reply 算出時間就填；不確定就留空字串。",
		"  time_text：若你不確定 cron 寫法，改用此欄填使用者講的時間自然語言（例如「每天早上六點」），交由程式換算。",
		"  action_text：排程實際要執行的動作。",
		"  target_job_no：intent=update 時，填 jobs 裡對應那筆的 no（整數）；其餘情況填 0。",
		"  question：仍缺關鍵資訊（時間或動作）才填一句追問；否則留空字串。",
		"  confidence：你對本次判斷的信心，只能是 high / medium / low。",
		"規則：",
		"  - 空字串／0 代表沿用原值或目前無資訊；不要捏造時間。",
		"  - user_reply 是修正句（例如「標題變成發貼文」）時，只更新對應欄位，並讓 summary 同步成自然描述。",
		"  - 已在 collecting/confirm 階段時，沿用 current_draft 已有資訊，只補上 user_reply 帶來的新資訊。",
		"輸出範例：{\"intent\":\"create\",\"title\":\"發貼文\",\"summary\":\"每天早上六點提醒發一篇貼文。\",\"cron_expr\":\"0 6 * * *\",\"time_text\":\"\",\"action_text\":\"發一篇貼文\",\"target_job_no\":0,\"question\":\"\",\"confidence\":\"high\"}",
		"context=" + compactPromptField(draftJSON),
		"user_reply=" + compactPromptField(userText),
	}, "\n")
}

func parseSchedulerDraftNormalization(text string) *SchedulerDraftNormalization {
	raw := strings.TrimSpace(text)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end >= start {
		raw = raw[start : end+1]
	}
	var out SchedulerDraftNormalization
	_ = json.Unmarshal([]byte(raw), &out)
	out.Intent = sanitizeSchedulerIntent(out.Intent)
	out.Title = sanitizeSchedulerModelField(out.Title, 48)
	out.Summary = sanitizeSchedulerModelField(out.Summary, 180)
	out.TimeText = sanitizeSchedulerModelField(out.TimeText, 80)
	out.ActionText = sanitizeSchedulerModelField(out.ActionText, 80)
	out.Question = sanitizeSchedulerModelField(out.Question, 120)
	out.Confidence = strings.ToLower(sanitizeSchedulerModelField(out.Confidence, 12))
	out.CronExpr = sanitizeSchedulerCron(out.CronExpr)
	if out.TargetJobNo < 0 {
		out.TargetJobNo = 0
	}
	return &out
}

// sanitizeSchedulerIntent 只接受白名單意圖，其餘一律視為未判定（空字串）。
func sanitizeSchedulerIntent(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "create", "update", "open", "none":
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return ""
	}
}

// sanitizeSchedulerCron 驗證模型給的 cron 是否合法；
// 不合法時回空字串，讓前端改用 time_text 後援換算，避免把壞 cron 寫進排程。
func sanitizeSchedulerCron(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`\"'「」")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if _, err := scheduler.ParseCron(s); err != nil {
		return ""
	}
	return s
}

func sanitizeSchedulerModelField(s string, maxRunes int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	s = strings.Trim(s, ` "'「」『』`)
	if maxRunes > 0 {
		r := []rune(s)
		if len(r) > maxRunes {
			s = strings.TrimSpace(string(r[:maxRunes]))
		}
	}
	return s
}

// --------------------------------------------------------------------------
// CreateScheduledJob — 建立排程任務
// --------------------------------------------------------------------------

// CreateScheduledJob 建立一個新的排程任務。
//
// 參數：
//   - name:          任務顯示名稱
//   - cronExpr:      Cron 表達式（五欄位格式或 @hourly 等快捷字）
//   - actionType:    動作類型（"event" / "skill" / "callback"）
//   - actionPayload: 動作的 JSON 酬載
//
// 回傳新建立的 Job 或驗證錯誤（如 cron 表達式不合法）。
func (a *App) CreateScheduledJob(name, cronExpr, actionType, actionPayload string) (*scheduler.Job, error) {
	if a.schedulerService == nil {
		return nil, fmt.Errorf("scheduler service 尚未初始化")
	}
	// TODO: riskClass 待接上 risk.ClassifyOperation；projectID 待前端傳入
	return a.schedulerService.CreateJob(name, cronExpr, scheduler.ActionType(actionType), actionPayload, "medium", "")
}

// --------------------------------------------------------------------------
// CancelScheduledJob — 取消正在執行的排程任務
// --------------------------------------------------------------------------

// CancelScheduledJob 取消正在執行中的指定排程任務。
func (a *App) CancelScheduledJob(id string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.CancelJob(id)
}

// --------------------------------------------------------------------------
// ListScheduledJobs — 列出所有排程任務
// --------------------------------------------------------------------------

// ListScheduledJobs 回傳所有排程任務的副本清單。
func (a *App) ListScheduledJobs() []scheduler.Job {
	if a.schedulerService == nil {
		return []scheduler.Job{}
	}
	return a.schedulerService.ListJobs()
}

// --------------------------------------------------------------------------
// DeleteScheduledJob — 刪除排程任務
// --------------------------------------------------------------------------

// DeleteScheduledJob 刪除指定 ID 的排程任務。
func (a *App) DeleteScheduledJob(id string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.DeleteJob(id)
}

// --------------------------------------------------------------------------
// PauseScheduledJob — 暫停排程任務
// --------------------------------------------------------------------------

// PauseScheduledJob 暫停指定 ID 的排程任務（Enabled = false）。
func (a *App) PauseScheduledJob(id string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.PauseJob(id)
}

// --------------------------------------------------------------------------
// ResumeScheduledJob — 恢復排程任務
// --------------------------------------------------------------------------

// ResumeScheduledJob 恢復指定 ID 的排程任務（Enabled = true），並重新計算 NextFire。
func (a *App) ResumeScheduledJob(id string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.ResumeJob(id)
}

// UpdateScheduledJob 更新指定排程任務。
func (a *App) UpdateScheduledJob(id, name, cronExpr, actionType, actionPayload string) (*scheduler.Job, error) {
	if a.schedulerService == nil {
		return nil, fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.UpdateJob(id, name, cronExpr, scheduler.ActionType(actionType), actionPayload)
}

// --------------------------------------------------------------------------
// GetJobExecutionHistory — 查詢執行歷史
// --------------------------------------------------------------------------

// GetJobExecutionHistory 查詢指定任務的執行歷史紀錄。
// limit 為回傳的最大筆數，依時間由新到舊排列。
func (a *App) GetJobExecutionHistory(jobID string, limit int) ([]scheduler.JobExecution, error) {
	if a.schedulerService == nil {
		return nil, fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.schedulerService.GetJobHistory(jobID, limit)
}

// --------------------------------------------------------------------------
// RegisterSchedulerCallback — 註冊 Go 側 callback placeholder
// --------------------------------------------------------------------------

// RegisterSchedulerCallback 註冊一個具名 callback placeholder。
// 實際產品功能可由 Go 側模組用同名 callback 覆蓋此 placeholder。
func (a *App) RegisterSchedulerCallback(name string) error {
	if a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	if name == "" {
		return fmt.Errorf("callback name 不可為空")
	}
	a.schedulerService.Callbacks().Register(name, func(ctx context.Context, args string) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})
	return nil
}

// --------------------------------------------------------------------------
// 背景喚醒（Phase G）— 前端關閉時詢問是否進低耗能背景並安裝喚醒鬧鐘
// --------------------------------------------------------------------------

// SchedulerHasActiveJobs 回報是否有啟用中的排程，前端關閉時用來決定要不要問「背景喚醒」。
func (a *App) SchedulerHasActiveJobs() bool {
	if a == nil || a.schedulerService == nil {
		return false
	}
	for _, j := range a.schedulerService.ListJobs() {
		if j.Enabled {
			return true
		}
	}
	return false
}

// SetSchedulerBackgroundWake 啟用/關閉「低耗能背景喚醒」。
// macOS：寫入/移除 user LaunchAgent（StartCalendarInterval，可從睡眠喚醒，免 sudo）。
// 其他平台：回傳不支援錯誤。注意：關機無法喚醒，UI 須提示「請勿關機」。
func (a *App) SetSchedulerBackgroundWake(enable bool) error {
	if a == nil || a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	return a.setSchedulerWake(enable)
}

// ResolveSchedulerBackgroundPrompt 前端在「關閉時背景選擇」對話框做出選擇後呼叫。
// enable=true → 安裝背景喚醒；false → 確保移除。決定後標記 bgPromptResolved，前端再 Quit()
// 即正常關閉（仍會走既有 save-as-sub 提示，不被本提示吃掉）。
func (a *App) ResolveSchedulerBackgroundPrompt(enable bool) error {
	werr := a.setSchedulerWake(enable)
	a.closeMu.Lock()
	a.bgPromptResolved = true
	a.closeMu.Unlock()
	if enable {
		return werr // 啟用失敗要讓前端知道；停用則 best-effort
	}
	return nil
}

// ConfirmScheduledRun 使用者在 app 內按「確認執行」後呼叫：繞過高風險/付費 API 閘門，
// 跑這一次（同日去重仍生效）。
func (a *App) ConfirmScheduledRun(jobID string) error {
	if a == nil || a.schedulerService == nil {
		return fmt.Errorf("scheduler service 尚未初始化")
	}
	job, ok := a.schedulerService.GetJobByID(jobID)
	if !ok {
		return fmt.Errorf("找不到排程 %q", jobID)
	}
	return a.runScheduledJobOrchestrationOpt(context.Background(), job, true)
}
