package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/adapter_registry"
	"ui_console/adapter/debugtrace"
	"ui_console/adapter/external_link"
	"ui_console/adapter/persona_avatar"
	"ui_console/adapter/remote_bridge"
	"ui_console/adapter/visual_learning"
	"ui_console/adapter/w3a_media"
	"ui_console/builtin"
	"ui_console/data/conversation"
	"ui_console/data/memory"
	"ui_console/data/project_lifecycle"
	"ui_console/data/storage"
	"ui_console/domain/controlled_trust"
	"ui_console/domain/credential"
	"ui_console/domain/degraded"
	"ui_console/domain/execution_hook"
	"ui_console/domain/llm_context"
	"ui_console/domain/review"
	"ui_console/domain/risk"
	"ui_console/domain/source_trust"
	"ui_console/internal/urlsafe"
	"ui_console/internal/voice"
	"ui_console/orchestration/cli_manager"
	"ui_console/orchestration/dag"
	"ui_console/orchestration/skill_step"
	"ui_console/orchestration/stop_recovery"
	"ui_console/shared/actionchain"
	"ui_console/shared/browser_pref"
	"ui_console/shared/controlseal"
	"ui_console/shared/eventbus"
	"ui_console/shared/health"
	"ui_console/shared/localsearch"
	"ui_console/shared/onboarding"
	"ui_console/shared/package_import"
	"ui_console/shared/preference"
	"ui_console/shared/scheduler"
	"ui_console/shared/settings"
	"ui_console/shared/statusrail"
	"ui_console/shared/taborder"
	"ui_console/shared/tools"
)

type schedulerSkillExecutor struct {
	router *skill_step.Router
}

func (e schedulerSkillExecutor) ExecuteSkill(ctx context.Context, actionTarget string, sessionID string) error {
	if e.router == nil {
		return fmt.Errorf("scheduler skill executor: skill router is nil")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	at, err := skill_step.ParseActionTarget(actionTarget)
	if err != nil {
		return err
	}
	_, err = e.router.Resolve(at, sessionID)
	return err
}

type llmAPIAdapterConfig struct {
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
	BaseURL    string `json:"base_url"`
	Model      string `json:"model"`
}

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type App struct {
	ctx             context.Context
	startupWg       sync.WaitGroup // SEC-20: 追蹤背景 goroutine
	statusRail      *statusrail.Service
	settingsService *settings.Service
	toolsService    *tools.Service
	voiceService    *voice.Service
	greetings       []string

	// v3.0.0 Execution Hook
	hookService          *execution_hook.Service
	reinforcementService *execution_hook.ReinforcementService
	candidateService     *execution_hook.CandidateService

	// v3.2.0 Controlled Trust
	trustLog        *controlled_trust.TrustLog
	overrideService *controlled_trust.ContextualOverrideService
	sessionService  *controlled_trust.TrustedSessionService
	deviceService   *controlled_trust.DeviceProfileService
	domClickService *controlled_trust.TrustDomClickService
	sandboxService  *controlled_trust.DraftSandboxService
	digestService   *controlled_trust.PendingDigestService

	// v3.3.2 P0 services
	packageService    *package_import.Service
	prefStore         *preference.PreferenceStore
	browserService    *browser_pref.Service
	uiSettingsService *settings.UISettingsService
	linkService       *external_link.Service

	// Skill Context Orchestration
	skillArchive    *skill_step.ArchiveService
	skillRouter     *skill_step.Router
	skillInjections *skill_step.InjectionStore
	skillAudit      *skill_step.AuditLog

	// 遺留待重建能力 — Wails 基礎架構遷移 (#1–#7)
	adapterRegistry *adapter_registry.Service // #1: 動態 adapter 列表
	reviewService   *review.Service           // #2: 統一 Review Card（取代 REST）
	eventBus        *eventbus.Bus             // #7: 正式事件匯流排（取代 WebSocket）
	healthService   *health.Service           // #4: Memory health / config public（read-only）
	degradedService *degraded.Service         // #5: Degraded mode controller
	onboardService  *onboarding.Service       // #6: First-run onboarding / read-only mode

	// #I-1002: Review Archive — rejected Review Card 持久化
	reviewArchive       *review.ArchiveService
	destructiveFailures map[string]int

	// v3.6.1 GatewaySentinel / Source Trust（§9）
	allowlistStore *source_trust.AllowlistStore

	// v3.6.1 Persona Avatar（§10）— 三層 provider 頭像系統
	avatarService *persona_avatar.Service
	// TASKS_1_6_3 Step 4：統一加密憑證儲存（取代 persona_avatar.CredentialStore）
	secretStore     credential.SecretStore
	credentialStore *credential.Store

	// v3.6.1 LLM Context Governance（§11）— 雙層掃描
	// 無需 service 實例，全部為純函式（llm_context 套件）

	// v3.6.1 Memory Pipeline（§18）— 三級記憶管線
	memoryPipeline *memory.Pipeline

	// v3.6.1 DAG Scheduler（§19）— 僅關鍵節點持久化
	dagScheduler    *dag.Scheduler
	taskMu          sync.Mutex
	activeTaskRunID string
	taskReviewIndex map[string]taskReviewRef

	// v3.6.1 Project Lifecycle（§7.5）— 專案清除 + manifest
	lifecycleService *project_lifecycle.Service

	// v3.6.1 Stop Recovery（§21）— 恢復卡片流程
	stopRecoveryService *stop_recovery.Service

	// v3.6.1 Visual Learning（§12）— 完整接入
	visualLearning      *visual_learning.Service
	learningService     *visual_learning.LearningService
	opencvPipeline      *visual_learning.OpenCVPipeline
	yoloDetector        *visual_learning.YOLODetector
	canonicalLabelSvc   *visual_learning.CanonicalLabelService
	elementDictionary   *visual_learning.ElementDictionary
	actionDictionary    *visual_learning.ActionDictionary
	pendingCandidateMgr *visual_learning.PendingCandidateManager
	vlSafeExporter      *visual_learning.SafeExporter

	// v3.6.2 W3A Media Provenance（§9A）— 媒體原件來源追蹤
	// 7 種驗證狀態、雙層指紋、操作指紋簽章、模型污染偵測、sidecar 管理。
	w3aMedia *w3a_media.Service

	// v3.6.3 Remote Bridge Communication（§12A）— 遠端橋接通訊
	// 管理 Telegram / Discord / LINE 三內建通道的註冊、啟用、模式切換、憑證與稽核。
	// 初始化於 NewApp()，hookRoot 作為持久化根目錄。
	remoteBridge        *remote_bridge.Service
	remoteBridgeInbound *remote_bridge.InboundServer
	remoteBridgeDiscord *remote_bridge.DiscordGatewayManager

	// v3.6.4 Readiness Gate UI Interaction Layer
	// 狀態由 readinessMu + currentGateState 全域管理（輕量級，不需 service 實例）

	// #I-801: Node Sidecar 生命週期管理器
	sidecar *cli_manager.SidecarManager
	// #I-803: CLIAdapter 透過 Sidecar IPC 與 Node 通訊
	cliAdapter skill_step.CLIAdapter
	// #I-804: DAG 事件節流閥（150ms）
	dagThrottler *cli_manager.DAGThrottler

	cacheMu         sync.Mutex
	previewCache    map[string]*skill_step.ScanPreview
	resolveCache    map[string]*skill_step.ResolveResult
	globalSessionID string
	closeMu         sync.Mutex
	allowClose      bool
	activeAgentID   string

	// 上方互動歷史只存記憶體，不進下方主聊天 SentenceStore。
	inspectorMu      sync.Mutex
	inspectorHistory []string // 格式："user: xxx" / "assistant: xxx"

	referencePromptMu      sync.Mutex
	referencePromptTargets map[string][]string

	// v3.6.5 Document Service（§24）— 文件匯入/匯出
	documentStore *builtin.Store

	// v4.0 Scheduler 時間排程系統（§27）— 定時執行 Event/Skill/Callback
	schedulerService *scheduler.Service
	pathGuard        *builtin.PathGuard
	docOnce          sync.Once
}

const inspectorHistoryLimit = 30

func NewApp() *App {
	greetings := []string{
		"主人今天好嗎？",
		"本人會默默陪主人完成事情",
		"來吧，讓本人幫主人一把！",
		"嘿嘿，被抓到了。",
		"今天是棒棒的一天...",
		"今天依舊忙碌的一天...",
		"嚕嚕.....拉拉，本人......嘟嘟，耶嘿！",
		"先休息一下吧！",
		"勇於認錯，態度依舊。",
		"哈哈，本人做的喔。",
	}
	root := appDataRoot()

	// v3.6.1: 雙層儲存架構 — 確保專案與 persona 目錄結構存在（§17）
	if err := storage.EnsureProjectLayout(root, "default"); err != nil {
		fmt.Fprintf(os.Stderr, "warning: EnsureProjectLayout failed: %v\n", err)
	}
	if err := storage.EnsurePersonaLayout(root, "main-persona"); err != nil {
		fmt.Fprintf(os.Stderr, "warning: EnsurePersonaLayout failed: %v\n", err)
	}

	hookRoot := storage.ProjectRoot(root, "default")
	trustDir := filepath.Join(hookRoot, "controlled_trust")

	trustLog := controlled_trust.NewTrustLog(trustDir)

	// Build the skill archive service once so the router shares the same instance.
	archiveSvc := skill_step.NewArchiveService(root)

	// TASKS_1_6_3 Step 4：共用 SecretStore，供 App 與 remote_bridge 共同使用。
	sharedSecretStore := credential.NewStore(root)

	cwd, _ := os.Getwd()
	a := &App{
		greetings:       greetings,
		statusRail:      statusrail.NewService(root, greetings),
		settingsService: settings.NewService(root),
		toolsService:    tools.NewService(),
		voiceService:    voice.NewService(root, cwd, appProgramRoot(cwd), appResourceRoot()),

		// v3.0.0 Execution Hook services
		hookService:          execution_hook.NewService(hookRoot),
		reinforcementService: execution_hook.NewReinforcementService(hookRoot),
		candidateService:     execution_hook.NewCandidateService(hookRoot),

		// v3.2.0 Controlled Trust services
		trustLog:        trustLog,
		overrideService: controlled_trust.NewContextualOverrideService(trustDir, trustLog),
		sessionService:  controlled_trust.NewTrustedSessionService(trustDir, trustLog),
		deviceService:   controlled_trust.NewDeviceProfileService(trustDir, trustLog),
		domClickService: controlled_trust.NewTrustDomClickService(trustDir, trustLog),
		sandboxService:  controlled_trust.NewDraftSandboxService(trustDir, trustLog),
		digestService:   controlled_trust.NewPendingDigestService(trustDir),

		// v3.3.2 P0 services
		packageService:    package_import.NewService(hookRoot),
		prefStore:         preference.NewPreferenceStore(root),
		browserService:    browser_pref.NewService(root),
		uiSettingsService: settings.NewUISettingsService(root),
		linkService:       external_link.NewService(root, false), // false = release mode

		// Skill Context Orchestration — archiveSvc is shared so that preview
		// cache and disk reads stay consistent across ScanFolder/ConfirmArchive.
		skillArchive:           archiveSvc,
		skillRouter:            skill_step.NewRouter(archiveSvc),
		skillInjections:        skill_step.NewInjectionStore(),
		skillAudit:             skill_step.NewAuditLog(root),
		previewCache:           make(map[string]*skill_step.ScanPreview),
		resolveCache:           make(map[string]*skill_step.ResolveResult),
		globalSessionID:        fmt.Sprintf("session-%d", time.Now().UnixNano()),
		referencePromptTargets: make(map[string][]string),

		// #I-1002: Review Archive — rejected card 持久化
		reviewArchive: review.NewArchiveService(hookRoot),

		// v3.6.1 GatewaySentinel
		allowlistStore: source_trust.NewAllowlistStore(hookRoot),

		// v3.6.1 Persona Avatar（§10）
		avatarService: persona_avatar.NewService(root),
		// TASKS_1_6_3 Step 4：統一 SecretStore（AES-256-GCM）
		secretStore:     sharedSecretStore,
		credentialStore: sharedSecretStore,

		// v3.6.1 Memory Pipeline（§18）
		memoryPipeline: memory.NewPipeline(hookRoot),

		// v3.6.1 DAG Scheduler（§19）
		dagScheduler:    dag.NewScheduler(hookRoot),
		taskReviewIndex: make(map[string]taskReviewRef),

		// v3.6.1 Project Lifecycle（§7.5）
		lifecycleService: project_lifecycle.NewService(hookRoot),

		// v3.6.1 Stop Recovery（§21）
		stopRecoveryService: stop_recovery.NewService(),

		// v3.6.1 Visual Learning（§12）— 完整接入
		visualLearning:      visual_learning.NewService(hookRoot),
		learningService:     visual_learning.NewLearningService(hookRoot),
		opencvPipeline:      visual_learning.NewOpenCVPipeline(),
		yoloDetector:        visual_learning.NewYOLODetector(filepath.Join(root, "assets", "models", "yolo_nano"), visual_learning.NewOpenCVPipeline()),
		canonicalLabelSvc:   visual_learning.NewCanonicalLabelService(filepath.Join(hookRoot, "data", "visual_learning")),
		elementDictionary:   visual_learning.NewElementDictionary(filepath.Join(hookRoot, "data", "visual_learning")),
		actionDictionary:    visual_learning.NewActionDictionary(filepath.Join(hookRoot, "data", "visual_learning")),
		pendingCandidateMgr: visual_learning.NewPendingCandidateManager(filepath.Join(hookRoot, "data", "visual_learning")),
		vlSafeExporter:      visual_learning.NewSafeExporter(filepath.Join(hookRoot, "data", "visual_learning")),

		// v3.6.2 W3A Media Provenance（§9A）
		w3aMedia: w3a_media.NewService(hookRoot),

		// v3.6.3 Remote Bridge Communication（§12A）— 注入共用 SecretStore
		remoteBridge: remote_bridge.NewService(hookRoot, sharedSecretStore),

		// 遺留待重建能力 (#1–#7)
		adapterRegistry:     adapter_registry.NewService(root),
		reviewService:       review.NewServiceWithDataRoot(hookRoot),
		eventBus:            eventbus.New(nil), // ctx injected in startup()
		healthService:       health.NewService(root, false),
		degradedService:     degraded.NewService(),
		onboardService:      onboarding.NewService(root),
		destructiveFailures: make(map[string]int),
	}

	// #I-801: 建立 Sidecar 管理器（啟動時將內建腳本落地到 app data）
	sidecarScriptPath, sidecarErr := ensureSidecarScript(root)
	if sidecarErr != nil {
		sidecarScriptPath = filepath.Join(root, "sidecar", "index.js")
	}
	sidecarMgr := cli_manager.NewSidecarManager(sidecarScriptPath)

	// #I-803: 以 Sidecar 建立 CLIAdapter（記憶體注入 SkillInjection）
	a.sidecar = sidecarMgr
	sidecarAdapter := cli_manager.NewSidecarCLIAdapter(sidecarMgr, filepath.Join(root, "cli-workspaces"))
	sidecarAdapter.SetControlSealSettings(a.settingsService.State().ControlSeal)
	a.cliAdapter = sidecarAdapter

	// #I-804: DAG 節流閥，避免高頻事件卡住前端
	a.dagThrottler = cli_manager.NewDAGThrottler(a.eventBus)

	// §24: 註冊文件內建能力（優先於外部 skill）

	// §27: 初始化排程引擎（啟動在 startup 中）
	a.schedulerService = scheduler.NewService(scheduler.ServiceConfig{
		DataRoot:  hookRoot,
		EventBus:  a.eventBus,
		SkillExec: schedulerSkillExecutor{router: a.skillRouter},
	})
	skill_step.RegisterDocumentBuiltins(a.skillRouter)
	return a
}

func appProgramRoot(fallback string) string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return fallback
	}
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "MacOS" && filepath.Base(filepath.Dir(dir)) == "Contents" {
		return filepath.Dir(filepath.Dir(filepath.Dir(dir)))
	}
	return dir
}

func appResourceRoot() string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return ""
	}
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "MacOS" && filepath.Base(filepath.Dir(dir)) == "Contents" {
		return filepath.Join(filepath.Dir(dir), "Resources")
	}
	return ""
}

// openBrowser 用系統預設瀏覽器開啟 URL。
// macOS 用 "open"，Linux 用 "xdg-open"，Windows 用 "rundll32"。
// 這是 fire-and-forget 操作，錯誤只 log 不回傳。
// openBrowser 在 OS 預設瀏覽器外部開啟 URL。
// macOS 用 "open"，Linux 用 "xdg-open"，Windows 用 "rundll32"。
// 這是 fire-and-forget 操作，錯誤只 log 不回傳。
//
// SEC-W06 Phase 1（2026-05-24）：只允許 http / https。
// 拒絕 file:// (Quick Look 開檔)、smb:// / afp:// (NTLM relay 攻擊)、
// javascript: / data: / mailto: 等非 web scheme。
// Phase 2 待辦：前端 BrowserOpenURL 直呼處（App.jsx 9 處）需獨立 wrapper。
func openBrowser(rawURL string) {
	if rawURL == "" {
		return
	}
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		log.Printf("openBrowser: refusing unparsable URL: %v", err)
		return
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		// allowed
	default:
		log.Printf("openBrowser: refusing non-http(s) scheme %q in URL %s", u.Scheme, rawURL)
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		log.Printf("openBrowser: unsupported OS %s", runtime.GOOS)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("openBrowser: failed to open %s: %v", rawURL, err)
	}
}

func appDataRoot() string {
	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return filepath.Join(configDir, "ai-console")
	}
	root, _ := os.Getwd()
	return root
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// DEBUG_TRACE_REMOVE: Temporary local trace viewer for UI -> CLI diagnostics.
	// Keep while cleaning dead code: the monitor can look offline when the app is
	// stopped, but previous trace logs depend on this UI -> Go -> sidecar path.
	debugtrace.Start(debugtrace.DefaultAddr)
	debugtrace.Record("go.startup", "", map[string]interface{}{
		"trace_link": debugtrace.Snapshot(),
	})
	writeMonitorLinkSnapshot(debugtrace.Snapshot())
	// #7: Inject Wails context into event bus so it can emit to frontend.
	a.eventBus.SetContext(ctx)
	a.interruptStaleTaskRuns("App 重新啟動")

	if a.remoteBridgeInbound == nil {
		a.remoteBridgeInbound = remote_bridge.NewInboundServer(a.remoteBridge, a.eventBus)
	}
	if err := a.remoteBridgeInbound.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: remote bridge inbound server failed: %v\n", err)
	}
	if a.remoteBridgeDiscord == nil {
		a.remoteBridgeDiscord = remote_bridge.NewDiscordGatewayManager(a.remoteBridge, a.eventBus)
	}
	a.remoteBridgeDiscord.Sync(ctx)

	// #I-805: 設定 Sidecar 崩潰回呼——標記 DAG 節點失敗並通知前端
	if a.sidecar != nil {
		a.sidecar.SetCrashHandler(func(exitCode int, err error) {
			payload := map[string]interface{}{
				"exit_code": exitCode,
				"reason":    "sidecar_crashed",
			}
			if err != nil {
				payload["error"] = err.Error()
			}
			a.eventBus.Emit(eventbus.EventExecutionInterrupted, payload)
			a.eventBus.Emit(eventbus.EventSidecarStateChanged, "crashed")

			// v3.6.1: 自動建立 Stop Recovery Card（§21）
			signal := fmt.Sprintf("exit code %d", exitCode)
			card := a.stopRecoveryService.CreateCard(stop_recovery.ReasonSidecarCrash, signal)
			a.eventBus.Emit("stop_recovery:card_created", map[string]string{
				"card_id": card.ID, "reason": string(card.StopReason),
			})
		})
		// #I-801: 背景啟動 Sidecar（不阻塞 UI 啟動）
		go func() {
			if startErr := a.sidecar.Start(ctx); startErr != nil {
				// 啟動失敗不致命——前端仍可運作，SendCLIMessage 會回傳 stub
				a.eventBus.Emit(eventbus.EventSidecarStateChanged, "start_failed")
			}
		}()
	}

	// v2.4: 啟動時背景偵測本機 CLI（只偵測不註冊，由使用者選擇啟用）
	// SEC-20: 用 WaitGroup 追蹤，防止 goroutine 洩漏
	a.startupWg.Add(1)
	go func() {
		defer a.startupWg.Done()
		a.adapterRegistry.AutoDetect() // 結果快取在記憶體，前端呼叫 AutoDetectCLI 時回傳
	}()

	// v3.6.1: 啟動時掃描 allowlist 到期狀態（§9.9）
	go func() {
		expiring, err := a.allowlistStore.CheckExpiries()
		if err == nil && len(expiring) > 0 {
			a.eventBus.Emit("source_trust:allowlist_expiring", expiring)
		}
	}()

	// v3.6.1: 啟動時掃描過期暫存檔（§7.5 ScanAndCleanExpired，30 天）
	go func() {
		if _, err := a.lifecycleService.ScanAndCleanExpired(30); err != nil {
			fmt.Fprintf(os.Stderr, "warning: startup expired scan failed: %v\n", err)
		}
	}()

	// v3.6.1: 啟動時產生 pixel avatar 快取（§10.9）
	go func() {
		pixelDir := filepath.Join(appDataRoot(), "data", "cache", "pixel_avatars")
		if err := persona_avatar.GenerateAllPixelAvatars(pixelDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: pixel avatar generation failed: %v\n", err)
		}
	}()

	// §27: 背景啟動排程引擎（載入持久化 Job + 補執行 + 啟動 Ticker）
	go func() {
		if err := a.schedulerService.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "warning: scheduler start failed: %v\n", err)
		}
	}()
}

// GetMonitorLinks returns the current local monitor/debug links.
// Keep: App.jsx uses this to find the active trace port before POSTing events.
func (a *App) GetMonitorLinks() debugtrace.LinkSnapshot {
	return debugtrace.Snapshot()
}

func writeMonitorLinkSnapshot(snapshot debugtrace.LinkSnapshot) {
	debugDir := filepath.Join(appDataRoot(), "debug")
	if err := os.MkdirAll(debugDir, 0o700); err != nil {
		log.Printf("monitor_link: create debug dir failed: %v", err)
		return
	}
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		log.Printf("monitor_link: marshal failed: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(debugDir, "monitor_link.json"), payload, 0o600); err != nil {
		log.Printf("monitor_link: write json failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(debugDir, "monitor_link.txt"), []byte(snapshot.URL+"\n"), 0o600); err != nil {
		log.Printf("monitor_link: write txt failed: %v", err)
	}
}

type ConsoleState struct {
	Greeting   string          `json:"greeting"`
	StatusRail statusrail.View `json:"statusRail"`
	Adapters   []string        `json:"adapters"`
	Haoras     []string        `json:"haoras"`
	Messages   []string        `json:"messages"`
}

type ReferenceFile struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
	Status string `json:"status,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// ---------------------------------------------------------------------------
// Console boot state hydration
// ---------------------------------------------------------------------------
// The frontend calls GetConsoleState immediately after Wails starts. This must
// return durable user-facing state, not just fresh in-memory defaults:
//   - adapter names come from the adapter registry;
//   - haㄌer labels stay on the subagent quick-send lane, separate from personas;
//   - recent chat bubbles are reconstructed from memory/talk_full.md.
//
// Chat persistence remains append-only in the memory pipeline. This hydration
// layer only reads the latest entries and maps them back to the UI message
// format ("Ai:" prefix for assistant bubbles), so reopening the app continues
// the visible conversation instead of starting from a blank panel.
func (a *App) GetConsoleState() ConsoleState {
	view := a.statusRail.View()
	// #1: 從 adapter_registry 動態取得 adapter 列表（取代硬編碼）
	adapters := a.adapterRegistry.ListAvailable()
	adapterNames := make([]string, len(adapters))
	for i, ad := range adapters {
		adapterNames[i] = ad.Name
	}
	return ConsoleState{
		Greeting:   view.Text,
		StatusRail: view,
		Adapters:   adapterNames,
		Haoras:     a.consoleHaoras(),
		Messages:   a.recentTalkMessages(80),
	}
}

func (a *App) consoleHaoras() []string {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	return consoleHaorasFromProject(projectRoot)
}

type subagentHaoraMeta struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type savedSubagent struct {
	id        string
	label     string
	createdAt time.Time
}

func consoleHaorasFromProject(projectRoot string) []string {
	haoras := []string{"主haㄌer"}
	for _, sub := range listSubagentTabs(projectRoot) {
		haoras = append(haoras, sub.label)
	}
	return haoras
}

func listSubagentTabs(projectRoot string) []savedSubagent {
	callableDir := filepath.Join(projectRoot, "subagents", "callable")
	entries, err := os.ReadDir(callableDir)
	if err != nil {
		return nil
	}

	var saved []savedSubagent
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		label := id
		var createdAt time.Time
		data, err := os.ReadFile(filepath.Join(callableDir, id, "sub_meta.json"))
		if err == nil {
			var meta subagentHaoraMeta
			if json.Unmarshal(data, &meta) == nil {
				if strings.TrimSpace(meta.Name) != "" {
					label = strings.TrimSpace(meta.Name)
				}
				if !meta.CreatedAt.IsZero() {
					createdAt = meta.CreatedAt
				}
			}
		}
		saved = append(saved, savedSubagent{id: id, label: label, createdAt: createdAt})
	}

	sort.SliceStable(saved, func(i, j int) bool {
		if !saved[i].createdAt.IsZero() && !saved[j].createdAt.IsZero() {
			return saved[i].createdAt.Before(saved[j].createdAt)
		}
		return saved[i].id < saved[j].id
	})

	return orderSavedSubagents(projectRoot, saved)
}

func orderSavedSubagents(projectRoot string, saved []savedSubagent) []savedSubagent {
	byID := make(map[string]savedSubagent, len(saved))
	for _, sub := range saved {
		byID[sub.id] = sub
	}

	seen := make(map[string]bool, len(saved))
	var ordered []savedSubagent
	orderMgr := taborder.NewManager(projectRoot)
	for _, id := range orderMgr.GetOrder().SubOrder {
		sub, ok := byID[id]
		if !ok || seen[id] {
			continue
		}
		ordered = append(ordered, sub)
		seen[id] = true
	}
	for _, sub := range saved {
		if !seen[sub.id] {
			ordered = append(ordered, sub)
			seen[sub.id] = true
		}
	}

	ids := make([]string, 0, len(ordered))
	for _, sub := range ordered {
		ids = append(ids, sub.id)
	}
	_ = orderMgr.Reorder(ids)
	return ordered
}

func resolveSubagentOrder(projectRoot string, requested []string) []string {
	callableDir := filepath.Join(projectRoot, "subagents", "callable")
	entries, err := os.ReadDir(callableDir)
	if err != nil {
		return requested
	}

	ids := make(map[string]bool)
	labels := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		ids[id] = true
		label := id
		data, err := os.ReadFile(filepath.Join(callableDir, id, "sub_meta.json"))
		if err == nil {
			var meta subagentHaoraMeta
			if json.Unmarshal(data, &meta) == nil && strings.TrimSpace(meta.Name) != "" {
				label = strings.TrimSpace(meta.Name)
			}
		}
		labels[label] = id
	}

	resolved := make([]string, 0, len(requested))
	seen := make(map[string]bool, len(requested))
	for _, item := range requested {
		id := item
		if !ids[id] {
			if mapped, ok := labels[item]; ok {
				id = mapped
			}
		}
		if ids[id] && !seen[id] {
			resolved = append(resolved, id)
			seen[id] = true
		}
	}
	return resolved
}

func (a *App) recentTalkMessages(limit int) []string {
	if limit <= 0 {
		limit = 80
	}
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	talk, err := memory.ReadTalkFull(projectRoot)
	if err != nil || strings.TrimSpace(talk) == "" {
		return []string{}
	}
	return parseTalkMessages(talk, limit)
}

// GetTalkMessagesForAgent reads the selected agent's own talk_full.md.
// main -> data/projects/default/memory/talk_full.md
// sub  -> data/projects/default/subagents/callable/[sub-id]/memory/talk_full.md
func (a *App) GetTalkMessagesForAgent(agentID string) ([]string, error) {
	root, err := conversationRootForAgent(agentID)
	if err != nil {
		return nil, err
	}
	talk, err := memory.ReadTalkFull(root)
	if err != nil || strings.TrimSpace(talk) == "" {
		return []string{}, err
	}
	return parseTalkMessages(talk, 80), nil
}

// AppendTalkEntryForAgent writes only to the selected agent's talk_full.md.
// Main and subs never share the same conversation file.
func (a *App) AppendTalkEntryForAgent(agentID, role, text string) error {
	root, err := conversationRootForAgent(agentID)
	if err != nil {
		return err
	}
	_, err = memory.NewPipeline(root).AppendTalkEntry(role, text)
	return err
}

// validAgentID SEC-W03 第二刀（2026-05-24）：agentID 只允許英數、底線、連字號。
// 例外 white-list：空字串 / "main" / "主haㄌer" 由上方 if 早退。
// 與 sub_export_binding.go 的 validSubID 風格一致。
var validAgentID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func conversationRootForAgent(agentID string) (string, error) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	id := strings.TrimSpace(agentID)
	if id == "" || id == "main" || id == "主haㄌer" {
		return projectRoot, nil
	}
	// SEC-W03: regex 拒絕 .. / . / 路徑分隔符 / 控制字元，比舊版 ContainsAny 嚴格。
	if !validAgentID.MatchString(id) {
		return "", fmt.Errorf("invalid agent id: %s", id)
	}
	subRoot := filepath.Join(projectRoot, "subagents", "callable", id)
	info, err := os.Stat(subRoot)
	if err != nil {
		return "", fmt.Errorf("subagent not found: %s", id)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("subagent path is not a directory: %s", id)
	}
	return subRoot, nil
}

func parseTalkMessages(talk string, limit int) []string {
	sections := strings.Split(talk, "\n## [")
	messages := make([]string, 0, limit)
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" || strings.HasPrefix(section, "# Talk Full") {
			continue
		}

		lines := strings.SplitN(section, "\n", 2)
		if len(lines) < 2 {
			continue
		}
		header := strings.TrimSpace(lines[0])
		body := strings.TrimSpace(lines[1])
		if body == "" {
			continue
		}

		role := strings.TrimSpace(header)
		if idx := strings.LastIndex(role, "]"); idx >= 0 {
			role = strings.TrimSpace(role[idx+1:])
		}
		switch role {
		case "assistant", "ai":
			messages = append(messages, "Ai:"+body)
		case "user":
			messages = append(messages, body)
		default:
			messages = append(messages, "["+role+"] "+body)
		}
	}
	if len(messages) > limit {
		return messages[len(messages)-limit:]
	}
	return messages
}

func (a *App) NextGreeting(current string) string {
	return a.statusRail.NextGreeting(current)
}

func (a *App) PollStatusRail() statusrail.View {
	return a.statusRail.View()
}

func (a *App) RecordMainInteraction(role string, text string) statusrail.View {
	a.statusRail.AddSnapshot(role, text)
	return a.statusRail.RecordMainInteraction()
}

func (a *App) NotifyStatusRail(source string, template string, subject string, priority string) statusrail.View {
	return a.statusRail.AddNotice(source, toNoticeTemplate(template), subject, toNoticePriority(priority))
}

func (a *App) AcknowledgeStatusRail() statusrail.View {
	return a.statusRail.AcknowledgeNotices()
}

func (a *App) GetSettingsState() settings.State {
	state := a.settingsService.State()
	// Panel 設定由 UISettingsService 管理，組合進 State DTO 供前端使用
	ui := a.uiSettingsService.Get()
	state.Panel = settings.PanelSettings{
		PanelLanguage: ui.PanelLanguage,
		RoleLanguage:  ui.RoleLanguage,
		FontPreset:    ui.FontPreset,
		FontScale:     ui.FontScale,
		PanelStyle:    ui.PanelStyle,
	}
	return state
}

func (a *App) SavePanelSettings(panel settings.PanelSettings) settings.State {
	// 面板設定委派給 UISettingsService（唯一來源）
	diffMap := map[string]interface{}{
		"panel_language": panel.PanelLanguage,
		"role_language":  panel.RoleLanguage,
		"font_preset":    panel.FontPreset,
		"font_scale":     panel.FontScale,
		"panel_style":    panel.PanelStyle,
	}
	diffJSON, _ := json.Marshal(diffMap)
	_, _ = a.uiSettingsService.ApplyStyleDiff(string(diffJSON))
	return a.GetSettingsState()
}

func (a *App) currentPanelLanguage() string {
	if a.uiSettingsService == nil {
		return "繁中"
	}
	return a.uiSettingsService.Get().PanelLanguage
}

func (a *App) GetVoiceSettings() voice.State {
	return a.voiceService.Get(a.currentPanelLanguage())
}

func (a *App) SaveVoiceSettings(next voice.Settings) (voice.State, error) {
	return a.voiceService.Save(next, a.currentPanelLanguage())
}

func (a *App) InstallVoiceBaseModel() (voice.State, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// Model download is user-triggered and can take a while on slower networks.
	downloadCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	return a.voiceService.InstallBaseModel(downloadCtx, a.currentPanelLanguage())
}

func (a *App) RemoveVoiceBaseModel() (voice.State, error) {
	return a.voiceService.RemoveManagedModel(a.currentPanelLanguage())
}

func (a *App) ClearVoiceDebug() (voice.State, error) {
	return a.voiceService.ClearDebug(a.currentPanelLanguage())
}

func (a *App) TranscribeVoiceWAV(audioBase64, mimeType string) (*voice.TranscriptResult, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// ASR runs out-of-process, so bound it to keep the UI from waiting forever.
	transcribeCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	result, err := a.voiceService.TranscribeWAVBase64(transcribeCtx, audioBase64, mimeType, a.currentPanelLanguage())
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *App) RouteVoiceCommand(text string) voice.CommandRoute {
	state := a.voiceService.Get(a.currentPanelLanguage())
	return voice.RouteCommand(text, state.Settings.CommandMode)
}

func (a *App) SavePersona(persona settings.Persona) settings.State {
	_ = a.settingsService.SavePersona(persona)
	return a.GetSettingsState()
}

func (a *App) ReorderPersonas(orderIDs []string) settings.State {
	_ = a.settingsService.ReorderPersonas(orderIDs)
	return a.GetSettingsState()
}

func (a *App) ListTools() []tools.Tool {
	return a.toolsService.List()
}

func (a *App) ActivateTool(id string) tools.ActionResult {
	return a.toolsService.Activate(id)
}

type cliActionTagSetter interface {
	SetActionTags(tags []string)
}

func (a *App) syncActionTagsToCLIAdapter(traceID string) []string {
	tags := a.collectActionTags()
	if setter, ok := a.cliAdapter.(cliActionTagSetter); ok {
		// Keep prompt candidates fresh; tools and builtin skills can change at runtime.
		setter.SetActionTags(tags)
	}
	debugtrace.Record("go.actionTags.sync", traceID, map[string]interface{}{
		"count": len(tags),
		"tags":  tags,
	})
	return tags
}

func (a *App) collectActionTags() []string {
	seen := make(map[string]bool)
	var tags []string
	add := func(values []string) {
		for _, tag := range values {
			tag = strings.TrimSpace(tag)
			// Reserved tags remain Controller-owned unless a later explicit opt-in path routes them.
			if tag == "" || actionchain.IsReservedTag(tag) || seen[tag] {
				continue
			}
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	if a.toolsService != nil {
		add(a.toolsService.ActionTags())
	}
	if a.skillRouter != nil {
		if routerTags, err := a.skillRouter.ActionTags(); err == nil {
			add(routerTags)
		} else {
			add(a.skillRouter.BuiltinActionTags())
		}
	}
	sort.Strings(tags)
	return tags
}

func (a *App) maybeHandleLocalSearch(userText, traceID string) (*skill_step.CLIResponse, bool) {
	req, ok := localsearch.ParseUserQuery(userText)
	if !ok {
		return nil, false
	}
	resp := a.executeLocalSearch(req, traceID)
	return &resp, true
}

func (a *App) executeLocalSearch(req localsearch.SearchRequest, traceID string) skill_step.CLIResponse {
	debugtrace.Record("local_search.enter", traceID, map[string]interface{}{
		"query": req.Query,
		"scope": req.Scope,
		"limit": req.Limit,
	})
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	// Bound local scans so a large managed data folder cannot freeze chat.
	ctx, cancel := context.WithTimeout(baseCtx, 2*time.Second)
	defer cancel()

	service := localsearch.NewService(a.localSearchRoots(), a.localSearchItems())
	outcome, err := service.SearchWithContext(ctx, req)
	if err != nil {
		if errors.Is(err, localsearch.ErrEmptyQuery) {
			return skill_step.CLIResponse{Text: localsearch.EmptyQueryMessage()}
		}
		debugtrace.Record("local_search.error", traceID, map[string]interface{}{
			"error": err.Error(),
		})
		return skill_step.CLIResponse{Error: err.Error()}
	}
	debugtrace.Record("local_search.results", traceID, map[string]interface{}{
		"count":         len(outcome.Results),
		"incomplete":    outcome.Incomplete,
		"reason":        outcome.Reason,
		"files_scanned": outcome.FilesScanned,
		"bytes_scanned": outcome.BytesScanned,
	})
	return skill_step.CLIResponse{Text: localsearch.FormatSearchOutcome(req, outcome)}
}

func (a *App) localSearchRoots() []localsearch.Root {
	root := appDataRoot()
	projectRoot := storage.ProjectRoot(root, "default")
	return []localsearch.Root{
		{Path: filepath.Join(projectRoot, "memory"), Source: "memory"},
		{Path: filepath.Join(projectRoot, "runtime"), Source: "trace"},
		{Path: filepath.Join(root, "documents"), Source: "document"},
		{Path: filepath.Join(root, "data", "documents"), Source: "document"},
		{Path: filepath.Join(root, "data", "references", "files"), Source: "document"},
		{Path: filepath.Join(root, "data", "skills"), Source: "skill"},
		{Path: filepath.Join(root, "debug"), Source: "trace"},
	}
}

func (a *App) localSearchItems() []localsearch.Item {
	var items []localsearch.Item
	if a.toolsService != nil {
		for _, tool := range a.toolsService.List() {
			items = append(items, localsearch.Item{
				Source: "tool",
				Title:  "工具: " + tool.Title,
				Path:   tool.ID,
				Content: strings.Join([]string{
					tool.ID,
					tool.Title,
					tool.Detail,
					tool.Kind,
					tool.Target,
					strings.Join(tool.ActionTags, " "),
				}, "\n"),
			})
		}
	}
	if a.skillRouter != nil {
		if tags, err := a.skillRouter.ActionTags(); err == nil {
			items = append(items, localsearch.Item{
				Source:  "skill",
				Title:   "技能 action tags",
				Content: strings.Join(tags, " "),
			})
		}
	}
	for _, event := range debugtrace.EventsSnapshot() {
		payload, _ := json.Marshal(event.Data)
		items = append(items, localsearch.Item{
			Source: "trace",
			Title:  fmt.Sprintf("trace #%d %s", event.ID, event.Node),
			Path:   event.TraceID,
			Content: strings.Join([]string{
				event.Time,
				event.TraceID,
				event.Node,
				string(payload),
			}, "\n"),
		})
	}
	return items
}

func (a *App) GetAppSessionID() string {
	return a.globalSessionID
}

// ---------------------------------------------------------------------------
// #I-801/#I-805 — Sidecar Wails Bindings
// ---------------------------------------------------------------------------

// GetSidecarState 回傳目前 Sidecar 狀態，供前端 UI 顯示。
func (a *App) GetSidecarState() string {
	if a.sidecar == nil {
		return string(cli_manager.StateIdle)
	}
	return string(a.sidecar.State())
}

// RestartSidecar 重新啟動 Node Sidecar（供前端 Stop Recovery Card 的重啟按鈕使用）。
func (a *App) RestartSidecar() error {
	if a.sidecar == nil {
		return fmt.Errorf("sidecar not configured")
	}
	_ = a.sidecar.Stop()
	if a.ctx == nil {
		return fmt.Errorf("app context not ready")
	}
	return a.sidecar.Start(a.ctx)
}

// StopSidecar 優雅關閉 Node Sidecar。
func (a *App) StopSidecar() error {
	if a.sidecar == nil {
		return nil
	}
	return a.sidecar.Stop()
}

// ---------------------------------------------------------------------------
// #12 — Execution Hook Wails Bindings (v3.0.0)
// ---------------------------------------------------------------------------

// StartHookRun begins a new hook observation run for a DAG run + outline pair.
func (a *App) StartHookRun(dagRunID, outlineID string) (string, error) {
	run, err := a.hookService.StartRun(dagRunID, outlineID)
	if err != nil {
		return "", err
	}
	return run.ID, nil
}

// RecordStepTrace records an actual step trace within an active hook run.
// tracePayload must be a JSON-encoded execution_hook.StepTrace.
func (a *App) RecordStepTrace(hookRunID string, tracePayload string) error {
	var trace execution_hook.StepTrace
	if err := json.Unmarshal([]byte(tracePayload), &trace); err != nil {
		return fmt.Errorf("RecordStepTrace: invalid trace payload: %w", err)
	}
	return a.hookService.RecordTrace(hookRunID, trace)
}

// GetHookSummary completes an active hook run and returns its summary.
func (a *App) GetHookSummary(hookRunID string) (interface{}, error) {
	summary, err := a.hookService.CompleteRun(hookRunID)
	return frontendDTO(summary), err
}

// GetPendingTagPatches returns all tag patches awaiting human review.
func (a *App) GetPendingTagPatches() (interface{}, error) {
	patches, err := a.reinforcementService.GetPendingTagPatches()
	return frontendDTO(patches), err
}

// GetToolRegistryPatchProposals returns tool registry patch proposals generated
// by hook evidence analysis.
func (a *App) GetToolRegistryPatchProposals() (interface{}, error) {
	proposals, err := a.reinforcementService.GetRegistryProposals()
	return frontendDTO(proposals), err
}

// GetNewSubagentCandidates returns new subagent candidates created by hook runs.
// Candidates are read-only proposals — they do NOT replace or disable existing subagents.
func (a *App) GetNewSubagentCandidates() (interface{}, error) {
	candidates, err := a.candidateService.ListCandidates()
	return frontendDTO(candidates), err
}

// ---------------------------------------------------------------------------
// #38 — Controlled Trust Wails Bindings (v3.2.0)
// ---------------------------------------------------------------------------

// EnableContextualRiskOverride creates a new scoped risk override.
// scopeJSON must encode a controlled_trust.OverrideScope; allowedRisk is "low", "medium", or "high_non_destructive".
// This ONLY reduces repeated review frequency — it does NOT lower final_risk.
func (a *App) EnableContextualRiskOverride(scopeJSON string, allowedRisk string) (string, error) {
	var scope controlled_trust.OverrideScope
	if err := json.Unmarshal([]byte(scopeJSON), &scope); err != nil {
		return "", fmt.Errorf("EnableContextualRiskOverride: invalid scope JSON: %w", err)
	}
	override, err := a.overrideService.Enable(scope, controlled_trust.AllowedRiskLevel(allowedRisk))
	if err != nil {
		return "", err
	}
	return override.ID, nil
}

// EnableWorkflowTrustForHours backs the Review Card choices:
// "trust this workflow for 30 minutes" and user-entered 1..999 hours.
// It only reduces repeated review prompts; final risk and hard rules are intact.
func (a *App) EnableWorkflowTrustForHours(scopeJSON string, allowedRisk string, hours int) (string, error) {
	if hours < 1 || hours > 999 {
		return "", fmt.Errorf("EnableWorkflowTrustForHours: hours must be between 1 and 999")
	}
	var scope controlled_trust.OverrideScope
	if err := json.Unmarshal([]byte(scopeJSON), &scope); err != nil {
		return "", fmt.Errorf("EnableWorkflowTrustForHours: invalid scope JSON: %w", err)
	}
	scope.Expiry = time.Now().Add(time.Duration(hours) * time.Hour)
	override, err := a.overrideService.Enable(scope, controlled_trust.AllowedRiskLevel(allowedRisk))
	if err != nil {
		return "", err
	}
	return override.ID, nil
}

// DisableContextualRiskOverride deactivates an override by ID.
func (a *App) DisableContextualRiskOverride(overrideID string) error {
	return a.overrideService.Disable(overrideID)
}

// EnableTrustedSessionScope activates a short-lived trusted session scope.
// Requires OS-level authentication in production (caller must enforce OS auth before calling).
// max_risk_covered: medium — critical/destructive operations remain unaffected.
func (a *App) EnableTrustedSessionScope(appSessionID, workspaceID, dagRunID, activeWindowHash string) error {
	_, err := a.sessionService.Enable(appSessionID, workspaceID, dagRunID, activeWindowHash)
	return err
}

// GetDeviceTrustProfile returns the trust profile for the given device, or nil if unknown.
func (a *App) GetDeviceTrustProfile(profileID string) interface{} {
	profile, _ := a.deviceService.GetProfile(profileID)
	return frontendDTO(profile)
}

// SetTrustDomAndClick enables or disables the advanced Trust DOM & Click preference.
// Default is OFF (advanced_preference: true, default: false per spec §0.6).
func (a *App) SetTrustDomAndClick(enabled bool) error {
	return a.domClickService.Set(enabled)
}

// StartDraftSandbox begins a draft sandbox session for the given active window.
// The sandbox is strictly limited to the current app session and active window.
func (a *App) StartDraftSandbox(activeWindowHash string) (string, error) {
	run, err := a.sandboxService.Start(activeWindowHash)
	if err != nil {
		return "", err
	}
	return run.ID, nil
}

// StopDraftSandbox halts an active sandbox. After stopping, the UI MUST present
// three continuation options: formal_review, pending_candidate, or discard.
func (a *App) StopDraftSandbox(sandboxID, reason string) error {
	_, err := a.sandboxService.Stop(sandboxID, reason)
	return err
}

// PromoteDraftToPending upgrades a stopped sandbox trace to a pending candidate.
// promotion must be "formal_review", "pending_candidate", or "discard".
// All three paths still respect Review and risk gate requirements.
//
// 方案 B 組合層：app.go 負責根據 promotion type 呼叫對應 service，
// DraftSandboxService.Promote 只做自身狀態轉換，
// 跨服務 side effect（Review Card / pending candidate store）由本層組合。
func (a *App) PromoteDraftToPending(sandboxID, promotion string) (string, error) {
	promo := controlled_trust.SandboxPromotion(promotion)

	// Step 1: DraftSandboxService 自身狀態轉換
	msg, err := a.sandboxService.Promote(sandboxID, promo)
	if err != nil {
		return "", err
	}

	// Step 2: 根據 promotion type 呼叫對應 service 產生 side effect
	switch promo {
	case controlled_trust.PromoteFormalReview:
		// 建立 Review Card，將 sandbox trace 送入正式審查流程
		card := a.reviewService.AddCard(review.CardParams{
			RiskClass:      "medium",
			Operation:      "draft_sandbox_promote",
			Target:         sandboxID,
			Reason:         "Draft Sandbox 錄製結果送交正式審查",
			AcceptLabel:    "確認採用",
			RejectLabel:    "捨棄",
			AcceptEffect:   "將錄製步驟加入正式流程",
			RejectEffect:   "不執行任何操作，錄製結果保留於歷史紀錄",
			RollbackAvail:  true,
			BackupAvail:    false,
			SourceType:     "draft_sandbox",
			SourceID:       sandboxID,
			EngineerReason: fmt.Sprintf("sandbox %s promoted to formal_review", sandboxID),
		})
		a.eventBus.Emit("review:card_added", card.ID)
		msg = fmt.Sprintf("sandbox %s → Review Card %s 已建立", sandboxID, card.ID)

	case controlled_trust.PromotePendingCandidate:
		// 存入 pending candidate store，供稍後審查
		record, addErr := a.pendingCandidateMgr.Add("draft_sandbox", sandboxID)
		if addErr != nil {
			return "", fmt.Errorf("PromoteDraftToPending: pending candidate 寫入失敗: %w", addErr)
		}
		a.eventBus.Emit("pending_candidate:added", record.ID)
		msg = fmt.Sprintf("sandbox %s → pending candidate %s 已儲存", sandboxID, record.ID)

	case controlled_trust.PromoteDiscard:
		// discard 路徑已由 DraftSandboxService.Promote 處理（清除 trace）
		a.eventBus.Emit("draft_sandbox:discarded", sandboxID)
	}

	return msg, nil
}

// GetPendingDigest returns the most recently generated pending digest.
// Digest is local-only computation — no LLM calls are made.
// #I-1001: 載入後檢查 300 筆上限，超過時自動封存並透過 EventBus 通知前端。
func (a *App) GetPendingDigest() (interface{}, error) {
	digest, err := a.digestService.LoadLatest()
	if err != nil {
		return nil, err
	}
	if digest.ID == "digest-empty" {
		// If no weekly digest exists yet, compute an empty digest immediately.
		digest, err = a.digestService.Generate([]controlled_trust.DigestItem{})
		if err != nil {
			return nil, err
		}
	}
	// #I-1001: 300 筆上限自動封存
	result, archiveErr := a.digestService.AutoArchiveIfOverLimit(digest)
	if archiveErr == nil && result != nil && result.ArchivedCount > 0 {
		a.eventBus.Emit(eventbus.EventDigestAutoArchived, result)
	}
	return frontendDTO(digest), nil
}

// AcknowledgePendingItem records a user's explicit action on a pending item.
// action must be one of: keep, archive, delete, review_now, batch_archive_low_value.
// No action is automatic — all require explicit user confirmation.
func (a *App) AcknowledgePendingItem(itemID, action string) error {
	return a.digestService.AcknowledgeItem(itemID, controlled_trust.DigestAction(action))
}

func (a *App) AcknowledgePendingItemWithConfirmation(itemID, action, confirmation string) error {
	return a.digestService.AcknowledgeItemWithConfirmation(itemID, controlled_trust.DigestAction(action), confirmation)
}

// ---------------------------------------------------------------------------
// #49 — v3.3.2 P0 Wails Bindings
// ---------------------------------------------------------------------------

// GetBrowserPreference returns the current browser selection. Read-only chip in UI.
func (a *App) GetBrowserPreference() interface{} {
	return frontendDTO(a.browserService.Get())
}

// SetBrowserPreference updates the user's browser selection.
// Triggers a one-time Safari notice if Safari is newly selected.
func (a *App) SetBrowserPreference(browser, profilePath string) (browser_pref.RuntimeNoticeResult, error) {
	return a.browserService.Set(browser_pref.BrowserKind(browser), profilePath)
}

// GetSafariRuntimeNotice returns the per-run Safari profile-reuse notice.
func (a *App) GetSafariRuntimeNotice() browser_pref.RuntimeNoticeResult {
	return a.browserService.SafariRuntimeNotice()
}

// GetUISettings returns current UI panel settings (isolated from memory/DAG/tools).
func (a *App) GetUISettings() interface{} {
	return frontendDTO(a.uiSettingsService.Get())
}

// RestoreUIDefaults resets UI settings to factory defaults.
// ONLY resets UI preferences — memory, DAG, installed tools, history are untouched.
func (a *App) RestoreUIDefaults() (interface{}, error) {
	next, err := a.uiSettingsService.RestoreDefaults()
	return frontendDTO(next), err
}

// ApplyUIStyleDiff applies an agent-supplied custom style after user confirmation.
func (a *App) ApplyUIStyleDiff(diffJSON string) (interface{}, error) {
	next, err := a.uiSettingsService.ApplyStyleDiff(diffJSON)
	return frontendDTO(next), err
}

// MarkSkillFirstUseExplained 持久化「初次使用說明卡已顯示」旗標。
// 前端在使用者關閉說明卡後呼叫，下次啟動不再顯示。
func (a *App) MarkSkillFirstUseExplained() (interface{}, error) {
	next, err := a.uiSettingsService.MarkSkillFirstUseExplained()
	return frontendDTO(next), err
}

// ──────────────────────────────────────────────
// #46 Live Preview 600 秒自動回滾 — Wails binding
// ──────────────────────────────────────────────

// StartLivePreview 啟動 Live Preview：套用新樣式並開始 600 秒倒數。
func (a *App) StartLivePreview(diffJSON string) (interface{}, error) {
	next, err := a.uiSettingsService.StartLivePreview(diffJSON)
	return frontendDTO(next), err
}

// CommitPreview 確認保留 Live Preview 樣式，停止計時器。
func (a *App) CommitPreview() (interface{}, error) {
	next, err := a.uiSettingsService.CommitPreview()
	return frontendDTO(next), err
}

// CancelPreview 取消 Live Preview，回滾到前一版。
func (a *App) CancelPreview() (interface{}, error) {
	next, err := a.uiSettingsService.CancelPreview()
	return frontendDTO(next), err
}

// IsLivePreviewActive 回傳 Live Preview 是否正在進行。
func (a *App) IsLivePreviewActive() bool {
	return a.uiSettingsService.IsLivePreviewActive()
}

// ──────────────────────────────────────────────
// #42.1 安全檢查 + #42.2 孤兒掃描 — Wails binding
// ──────────────────────────────────────────────

// RunPackageSecurityCheck 對 quarantine 中的 package 執行完整安全檢查。
func (a *App) RunPackageSecurityCheck(importID string) (interface{}, error) {
	result, err := a.packageService.RunSecurityCheck(importID)
	return frontendDTO(result), err
}

// ScanOrphanQuarantine 啟動時掃描孤兒 quarantine 項目。
func (a *App) ScanOrphanQuarantine() (interface{}, error) {
	results, err := a.packageService.ScanOrphanQuarantine()
	return frontendDTO(results), err
}

// CleanOrphanQuarantine 清除安全檢查失敗的孤兒項目。
func (a *App) CleanOrphanQuarantine(orphanID string) error {
	return a.packageService.CleanOrphan(orphanID)
}

// ──────────────────────────────────────────────
// #44 工具斷線 / 恢復 — Wails binding
// ──────────────────────────────────────────────

// MarkToolUnavailable 標記工具為斷線狀態。
func (a *App) MarkToolUnavailable(toolID, reason string) {
	a.toolsService.MarkUnavailable(toolID, reason)
}

// MarkToolAvailable 標記工具恢復連線。
func (a *App) MarkToolAvailable(toolID string) {
	a.toolsService.MarkAvailable(toolID)
}

func (a *App) ensureSidecarRunning() error {
	if a.sidecar == nil {
		return nil
	}
	state := a.sidecar.State()
	if state == cli_manager.StateRunning {
		return nil
	}
	for i := 0; state == cli_manager.StateStarting && i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		state = a.sidecar.State()
		if state == cli_manager.StateRunning {
			return nil
		}
	}
	if a.ctx == nil {
		return fmt.Errorf("cli_manager: sidecar context not ready")
	}
	log.Printf("ensureSidecarRunning: sidecar state=%s, attempting restart", state)
	if err := a.sidecar.Start(a.ctx); err != nil {
		state = a.sidecar.State()
		if state == cli_manager.StateRunning {
			return nil
		}
		return err
	}
	for i := 0; i < 20; i++ {
		state = a.sidecar.State()
		if state == cli_manager.StateRunning {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("cli_manager: sidecar failed to start (state=%s)", a.sidecar.State())
}

// SendCLIMessage 透過 CLIAdapter 發送訊息給選中的 CLI。
// 自動從 InjectionStore 取得當前 session 的 SkillInjection 並附加。
// 若 CLIAdapter 尚未設定（cliAdapter == nil），回傳 stub 回應。
func (a *App) SendCLIMessage(adapterID string, sessionID string, userText string, traceID string) (*skill_step.CLIResponse, error) {
	// === 診斷 log：記錄每次 CLI 呼叫的關鍵參數，方便追蹤問題 ===
	log.Printf("SendCLIMessage: adapter=%q session=%q text_len=%d", adapterID, sessionID, len(userText))
	// DEBUG_TRACE_REMOVE: Captures the Wails binding input with full dev-mode text.
	debugtrace.Record("go.SendCLIMessage.enter", traceID, map[string]interface{}{
		"adapter_id": adapterID,
		"session_id": sessionID,
		"user_text":  userText,
		"text_len":   len(userText),
	})

	var inj *skill_step.Injection
	if a.skillInjections != nil {
		inj = a.skillInjections.Get(sessionID)
	}
	actionTags := a.syncActionTagsToCLIAdapter(traceID)
	if resp, handled := a.maybeHandleLocalSearch(userText, traceID); handled {
		debugtrace.Record("go.cliAdapter.response", traceID, map[string]interface{}{
			"text":          resp.Text,
			"error":         resp.Error,
			"auth_required": false,
			"auth_url":      "",
		})
		return resp, nil
	}

	// 從 adapter_registry 解析 CLI 的實際執行檔路徑。
	// 若 registry 是空的（使用者未走過 onboarding），這裡會失敗並留下空的 cliPath。
	cliPath := ""
	if adapterID != "" && a.adapterRegistry != nil {
		if resolvedPath, err := a.adapterRegistry.ResolveExecutable(adapterID); err == nil {
			cliPath = resolvedPath
			log.Printf("SendCLIMessage: resolved CLI path: %s", cliPath)
			// DEBUG_TRACE_REMOVE: Shows which executable will receive the prompt.
			debugtrace.Record("go.adapter.resolve.ok", traceID, map[string]interface{}{
				"adapter_id": adapterID,
				"cli_path":   cliPath,
			})
		} else {
			// 關鍵診斷：adapter 註冊了但找不到執行檔，或根本沒註冊
			log.Printf("SendCLIMessage: WARNING — failed to resolve executable for adapter %q: %v", adapterID, err)
			// DEBUG_TRACE_REMOVE: Keeps adapter resolution failures visible in the trace page.
			debugtrace.Record("go.adapter.resolve.error", traceID, map[string]interface{}{
				"adapter_id": adapterID,
				"error":      err.Error(),
			})
		}
	} else {
		log.Printf("SendCLIMessage: WARNING — adapterID is empty or registry is nil")
	}

	// 記錄 sidecar 目前狀態，方便判斷是 sidecar 沒跑還是 CLI 找不到
	if a.sidecar != nil {
		log.Printf("SendCLIMessage: sidecar state=%s", a.sidecar.State())
	} else {
		log.Printf("SendCLIMessage: WARNING — sidecar is nil")
	}

	if a.cliAdapter == nil {
		// Stub：CLIAdapter 尚未接入時回傳本地回應，讓 UI 流程可測試。
		log.Printf("SendCLIMessage: cliAdapter is nil, returning stub response")
		// DEBUG_TRACE_REMOVE: Stub response path.
		debugtrace.Record("go.SendCLIMessage.stub", traceID, map[string]interface{}{
			"adapter_id": adapterID,
			"user_text":  userText,
		})
		return &skill_step.CLIResponse{
			Text: fmt.Sprintf("[stub:%s] received: %s (skill=%v)", adapterID, userText, inj != nil),
		}, nil
	}
	if err := a.ensureSidecarRunning(); err != nil {
		log.Printf("SendCLIMessage: ERROR — ensureSidecarRunning failed: %v", err)
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		// DEBUG_TRACE_REMOVE: Sidecar startup/restart failure.
		debugtrace.Record("go.sidecar.ensure.error", traceID, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	referencePlan := a.planReferenceSearchWithCLI(adapterID, cliPath, sessionID, userText, traceID)
	systemPrompt := a.buildMainComposerPrompt(a.getActivePersona())
	referenceContext := a.buildReferencePromptContextFromPlan(sessionID, userText, traceID, referencePlan)
	if referenceContext != "" {
		systemPrompt += referenceContext
	}
	opts := skill_step.CLIMessageOptions{
		AdapterID:      adapterID,
		CLIPath:        cliPath,
		SessionID:      sessionID,
		UserText:       userText,
		SkillInjection: inj,
		SystemPrompt:   systemPrompt,
		ContinuityKey:  conversationContinuityKey("composer", sessionID),
		TraceID:        traceID,
	}
	// DEBUG_TRACE_REMOVE: Captures the final Go options before the sidecar adapter.
	debugtrace.Record("go.CLIMessageOptions", traceID, map[string]interface{}{
		"adapter_id":          opts.AdapterID,
		"cli_path":            opts.CLIPath,
		"session_id":          opts.SessionID,
		"user_text":           opts.UserText,
		"continuity_key":      opts.ContinuityKey,
		"system_prompt_len":   len([]rune(opts.SystemPrompt)),
		"reference_plan":      referencePlan,
		"has_reference_ctx":   referenceContext != "",
		"has_skill_injection": opts.SkillInjection != nil,
		"action_tags":         actionTags,
	})
	resp, err := a.cliAdapter.SendMessage(opts)
	if err != nil {
		log.Printf("SendCLIMessage: ERROR — cliAdapter.SendMessage failed: %v", err)
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		// DEBUG_TRACE_REMOVE: Adapter call failure.
		debugtrace.Record("go.cliAdapter.error", traceID, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}
	if resp.Action == "提問" {
		// 內建「提問」只更新回問狀態，不直接執行工具。
		resp.Text = setQuestionFloatingCandidates(questionPayload(resp.Target, resp.Next), traceID)
	}
	if resp.Action == "選項" {
		resp.Text = setOptionFloatingCandidates(resp.Target, traceID)
	}
	if resp.Action == "版控" {
		resp.Text = executeGitStatusShort()
	}
	if req, ok := localsearch.RequestFromAction(resp.Action, resp.Target); ok {
		// CLI/local model only chooses the builtin action; App owns the actual local scan.
		localResp := a.executeLocalSearch(req, traceID)
		return &localResp, nil
	}
	// DEBUG_TRACE_REMOVE: Response returned from sidecar/CLI to Go.
	debugtrace.Record("go.cliAdapter.response", traceID, map[string]interface{}{
		"text":          resp.Text,
		"error":         resp.Error,
		"auth_required": resp.AuthRequired,
		"auth_url":      resp.AuthURL,
		"action":        resp.Action,
		"target":        resp.Target,
		"next":          resp.Next,
	})
	log.Printf("[CLI_MONITOR] Go App.SendCLIMessage return trace=%s text_len=%d error=%q auth_required=%v text=%s", traceID, len(resp.Text), resp.Error, resp.AuthRequired, resp.Text)

	// === 授權攔截：CLI 需要瀏覽器 OAuth 授權 ===
	// 當 sidecar 偵測到 CLI 輸出授權提示（如 Gemini 首次執行），
	// 回應中會帶 AuthRequired=true + AuthURL。
	// 此處自動用系統瀏覽器開啟 OAuth URL，並 emit 事件通知前端顯示授權對話框。
	if resp.AuthRequired {
		log.Printf("SendCLIMessage: CLI %q requires browser authorization (url=%s)", adapterID, resp.AuthURL)
		// SEC-05: 驗證 auth URL，trusted 自動開瀏覽器，untrusted 需前端確認
		authTrusted := false
		authHostname := ""
		if resp.AuthURL != "" {
			authTrusted, authHostname = urlsafe.ValidateAuthURL(adapterID, resp.AuthURL)
			if authTrusted {
				go openBrowser(resp.AuthURL)
			} else {
				log.Printf("SendCLIMessage: untrusted auth URL %q (host=%s), requesting user confirmation", resp.AuthURL, authHostname)
			}
		}
		// 通知前端顯示授權對話框
		// SEC-05: 帶 trusted 欄位讓前端決定是否顯示確認框
		a.eventBus.Emit(eventbus.EventCLIAuthRequired, map[string]string{
			"adapter_id":    adapterID,
			"auth_url":      resp.AuthURL,
			"message":       resp.Text,
			"user_text":     userText,
			"session_id":    sessionID,
			"trace_id":      traceID,
			"auth_trusted":  fmt.Sprintf("%v", authTrusted),
			"auth_hostname": authHostname,
		})
		return &resp, nil
	}
	if resp.Error != "" {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
	} else {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusOnline)
	}

	log.Printf("SendCLIMessage: success, response_len=%d", len(resp.Text))

	return &resp, nil
}

// SendAPIMessage sends a composer message through an LLM API adapter.
func (a *App) SendAPIMessage(adapterID string, sessionID string, userText string, traceID string) (*skill_step.CLIResponse, error) {
	debugtrace.Record("go.SendAPIMessage.enter", traceID, map[string]interface{}{
		"adapter_id": adapterID,
		"session_id": sessionID,
		"text_len":   len(userText),
	})
	actionTags := a.syncActionTagsToCLIAdapter(traceID)
	if resp, handled := a.maybeHandleLocalSearch(userText, traceID); handled {
		debugtrace.Record("go.apiAdapter.response", traceID, map[string]interface{}{
			"text":          resp.Text,
			"error":         resp.Error,
			"auth_required": false,
			"auth_url":      "",
		})
		return resp, nil
	}
	cfg, err := a.loadLLMAPIAdapterConfig(adapterID)
	if err != nil {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return nil, err
	}
	// Local adapters (Ollama, LM Studio) don't need an API key.
	isLocalAdapter := false
	var localAdapterInfo adapter_registry.Adapter
	if a.adapterRegistry != nil {
		if adapterInfo, adapterErr := a.adapterRegistry.GetStatus(adapterID); adapterErr == nil && adapterInfo.Kind == "local" {
			isLocalAdapter = true
			localAdapterInfo = adapterInfo
		}
	}
	apiKey := ""
	if !isLocalAdapter {
		var keyErr error
		apiKey, keyErr = a.secretStore.Load("llm_provider:" + adapterID + ":api_key")
		if keyErr != nil || strings.TrimSpace(apiKey) == "" {
			a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
			return nil, fmt.Errorf("API key not found for adapter %s", adapterID)
		}
	}
	if !isOpenAICompatibleProvider(cfg.ProviderID) {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return &skill_step.CLIResponse{
			Error: fmt.Sprintf("%s 目前尚未接上直接 API 協議；請先使用 DeepSeek/OpenAI-compatible provider。", cfg.Name),
		}, nil
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, fmt.Errorf("API model is missing for adapter %s", adapterID)
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	client := &http.Client{Timeout: 45 * time.Second}
	doRequest := func(prompt string) (*http.Response, error) {
		reqBody := openAIChatRequest{
			Model: model,
			Messages: []openAIChatMessage{
				// API adapters receive the same synthesized controller prompt as CLI adapters.
				{Role: "user", Content: prompt},
			},
		}
		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIChatCompletionsURL(cfg.BaseURL), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
		req.Header.Set("Content-Type", "application/json")
		return client.Do(req)
	}
	callAPI := func(prompt string) (string, error) {
		res, err := doRequest(prompt)
		if err != nil && isLocalAdapter && strings.Contains(localAdapterInfo.Endpoint, ":11434") {
			debugtrace.Record("go.SendAPIMessage.local.wake_retry", traceID, map[string]interface{}{
				"adapter_id": adapterID,
				"error":      err.Error(),
			})
			if _, wakeErr := a.wakeOllamaAdapter(localAdapterInfo); wakeErr == nil {
				res, err = doRequest(prompt)
			} else {
				debugtrace.Record("go.SendAPIMessage.local.wake_error", traceID, map[string]interface{}{
					"adapter_id": adapterID,
					"error":      wakeErr.Error(),
				})
			}
		}
		if err != nil {
			a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
			debugtrace.Record("go.SendAPIMessage.http.error", traceID, map[string]interface{}{"error": err.Error()})
			if isLocalAdapter {
				return "", errors.New(localAdapterReconnectHint(localAdapterInfo, err.Error()))
			}
			return "", err
		}
		defer res.Body.Close()
		raw, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024))
		if err != nil {
			a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
			return "", err
		}
		var parsed openAIChatResponse
		_ = json.Unmarshal(raw, &parsed)
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
			msg := strings.TrimSpace(parsedErrorMessage(parsed))
			if msg == "" {
				msg = strings.TrimSpace(string(raw))
			}
			if isLocalAdapter && res.StatusCode >= 500 {
				return "", errors.New(localAdapterReconnectHint(localAdapterInfo, fmt.Sprintf("API HTTP %d: %s", res.StatusCode, msg)))
			}
			return "", fmt.Errorf("API HTTP %d: %s", res.StatusCode, msg)
		}
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
			return "", errors.New(parsed.Error.Message)
		}
		if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
			a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
			return "", errors.New("API response did not include assistant content")
		}
		return parsed.Choices[0].Message.Content, nil
	}

	referencePlan := a.planReferenceSearchWithAPI(sessionID, userText, traceID, callAPI)
	basePrompt := a.buildMainComposerPrompt(a.getActivePersona())
	referenceContext := a.buildReferencePromptContextFromPlan(sessionID, userText, traceID, referencePlan)
	if referenceContext != "" {
		basePrompt += referenceContext
	}
	var synthesizedPrompt string
	if isLocalAdapter {
		synthesizedPrompt = buildLocalModelPrompt(basePrompt, actionTags, userText)
	} else {
		synthesizedPrompt = buildAPIActionChainPrompt(basePrompt, actionTags, userText)
	}
	debugtrace.Record("go.APIMessageOptions", traceID, map[string]interface{}{
		"adapter_id":        adapterID,
		"session_id":        sessionID,
		"user_text":         userText,
		"system_prompt_len": len([]rune(basePrompt)),
		"synth_prompt_len":  len([]rune(synthesizedPrompt)),
		"reference_plan":    referencePlan,
		"has_reference_ctx": referenceContext != "",
		"has_now":           strings.Contains(synthesizedPrompt, "now="),
		"has_action_chain":  strings.Contains(synthesizedPrompt, "動作ㄌ目標ㄌ下一步"),
		"has_control_seal":  strings.Contains(synthesizedPrompt, "本輪命令前綴"),
		"action_tags":       actionTags,
		"action_tags_count": len(actionTags),
	})

	var resp skill_step.CLIResponse
	if isLocalAdapter {
		// Local models: single call, no retry; assembler handles any format.
		rawText, err := callAPI(synthesizedPrompt)
		if err != nil {
			return &skill_step.CLIResponse{Error: err.Error()}, nil
		}
		debugtrace.Record("go.SendAPIMessage.rawResult", traceID, map[string]interface{}{
			"result":        rawText,
			"retry_attempt": 0,
			"local_model":   true,
		})
		chain := assembleLocalResponse(rawText)
		debugtrace.Record("go.APIMessage.localAssembled", traceID, map[string]interface{}{
			"action": chain.Action,
			"target": chain.Target,
			"next":   chain.Next,
		})
		resp = a.resolveActionChainResponse(
			chain.Action+actionchain.Separator+chain.Target+actionchain.Separator+chain.Next,
			actionTags, traceID, sessionID,
		)
		// If resolve still fails, fallback to raw text as chat.
		if resp.Error != "" {
			resp = skill_step.CLIResponse{Text: chain.Target}
		}
	} else {
		for attempt := 0; attempt <= 2; attempt++ {
			prompt := synthesizedPrompt
			if attempt > 0 {
				prompt = synthesizedPrompt + "\n\n" + actionchain.RetryPrompt()
			}
			rawText, err := callAPI(prompt)
			if err != nil {
				return &skill_step.CLIResponse{Error: err.Error()}, nil
			}
			debugtrace.Record("go.SendAPIMessage.rawResult", traceID, map[string]interface{}{
				"result":        rawText,
				"retry_attempt": attempt,
			})
			resp = a.resolveActionChainResponse(rawText, actionTags, traceID, sessionID)
			if resp.Error == "" {
				break
			}
			if attempt == 2 {
				debugtrace.Record("go.APIMessage.actionChain.fallback", traceID, map[string]interface{}{
					"raw_text": rawText,
					"reason":   "protocol retries exhausted",
				})
				return &skill_step.CLIResponse{Text: rawText}, nil
			}
		}
	}
	if resp.Action == "提問" {
		// 內建「提問」只更新回問狀態，不直接執行工具。
		resp.Text = setQuestionFloatingCandidates(questionPayload(resp.Target, resp.Next), traceID)
	}
	if resp.Action == "選項" {
		resp.Text = setOptionFloatingCandidates(resp.Target, traceID)
	}
	if resp.Action == "版控" {
		resp.Text = executeGitStatusShort()
	}
	a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusOnline)
	debugtrace.Record("go.apiAdapter.response", traceID, map[string]interface{}{
		"text":   resp.Text,
		"error":  resp.Error,
		"action": resp.Action,
		"target": resp.Target,
		"next":   resp.Next,
	})
	return &resp, nil
}

func buildAPIActionChainPrompt(systemPrompt string, actionTags []string, userText string) string {
	return conversation.Synthesize(conversation.SynthesisConfig{
		SystemPrompt: systemPrompt,
		ActionTags:   append([]string(nil), actionTags...),
		CurrentInput: userText,
		CommandSeal:  controlseal.CurrentSeal(),
		SanitizeLLM:  true,
	})
}

// buildLocalModelPrompt produces a dead-simple prompt for small local models.
// Instead of the ㄌ protocol, it asks for three short lines.
func buildLocalModelPrompt(systemPrompt string, actionTags []string, userText string) string {
	tagList := strings.Join(conversation.PromptActionTags(actionTags), "、")
	return fmt.Sprintf(
		"%s\n回答規則：用三行回答，每行一個欄位。\n第一行寫動作（從候選中選：%s）\n動作定義：已知答案用 輸出；需要系統查資料用 搜尋；讀取=取得內容並回報；開啟=用外部應用程式呈現；寫入=新增或修改檔案；匯出=產生檔案；提問=補問必要資訊；選項=顯示選項卡。\n第二行寫內容；若第一行是選項且沒有問題文字，必須用 ㄤ 開頭，例如 ㄤ紅色ㄤ綠色ㄤ藍色；若有問題文字，寫 問題ㄤ選項一ㄤ選項二。\n第三行只寫 待命、輸出、選項 其中之一；通常寫 待命。\n範例：\n輸出\n你好啊\n待命\n\nQ: %s\n",
		systemPrompt, tagList, userText,
	)
}

// assembleLocalResponse parses a local model's 3-line output into an ActionChain.
// Falls back to treating the whole output as chat if parsing fails.
func assembleLocalResponse(raw string) actionchain.ActionChain {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	// Filter out empty lines.
	var nonEmpty []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}
	if len(nonEmpty) >= 3 {
		return actionchain.ActionChain{
			Action: strings.TrimSpace(nonEmpty[0]),
			Target: strings.TrimSpace(nonEmpty[1]),
			Next:   actionchain.NormalizeNext(strings.TrimSpace(nonEmpty[2])),
			Raw:    raw,
		}
	}
	if len(nonEmpty) == 2 {
		return actionchain.ActionChain{
			Action: strings.TrimSpace(nonEmpty[0]),
			Target: strings.TrimSpace(nonEmpty[1]),
			Next:   actionchain.StandbyNext,
			Raw:    raw,
		}
	}
	// Can't parse — treat as plain chat.
	return actionchain.ActionChain{
		Action: "聊天",
		Target: strings.TrimSpace(raw),
		Next:   actionchain.StandbyNext,
		Raw:    raw,
	}
}

// executeGitStatusShort 在專案目錄執行 git status --short，回傳結果。
// 僅唯讀操作，cmd.Dir 鎖定 cwd 不允許任意路徑。
func executeGitStatusShort() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "無法取得工作目錄：" + err.Error()
	}
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = cwd
	// SEC-W17（2026-05-24）：鎖住 git 不讀外部 config，避免 CVE-2022-39253 系
	// （core.fsmonitor / core.attributesfile 等 config 注入路徑）。cwd 保留原語意。
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if strings.Contains(outStr, "not a git repository") {
			return "目前資料夾不是 git 儲存庫。"
		}
		if errors.Is(err, exec.ErrNotFound) {
			return "未安裝 git。macOS 可在終端機輸入 git --version 觸發自動安裝。"
		}
		return "git 執行失敗：" + err.Error()
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "工作區乾淨，沒有任何變更。"
	}
	return result
}
func (a *App) resolveActionChainResponse(rawText string, actionTags []string, traceID string, sessionID string) skill_step.CLIResponse {
	chain, parseErr := actionchain.Parse(rawText)
	if parseErr != nil {
		debugtrace.Record("go.APIMessage.actionChain.structure_error", traceID, map[string]interface{}{
			"error": parseErr.Error(),
		})
		return skill_step.CLIResponse{Error: actionchain.RetryPrompt()}
	}

	displayText := chain.Target
	registry := actionchain.NewStaticRegistry(actionTags...)
	validation := actionchain.ValidateActionTag(chain.Action, registry)
	if validation.Code == actionchain.ValidationUnknown {
		debugtrace.Record("go.APIMessage.actionChain.unknown", traceID, map[string]interface{}{
			"action": chain.Action,
			"target": chain.Target,
			"next":   chain.Next,
		})
	}
	if req, ok := localsearch.RequestFromAction(chain.Action, chain.Target); ok {
		// LLM only routes intent with ㄌ syntax; search results never go back to the model.
		resp := a.executeLocalSearch(req, traceID)
		resp.Action = chain.Action
		resp.Target = chain.Target
		resp.Next = chain.Next
		return resp
	}
	if decision := actionchain.ResolveBuiltIn(chain); decision.Handled {
		displayText = decision.DisplayText
		debugtrace.Record("go.APIMessage.actionChain.builtin", traceID, map[string]interface{}{
			"action":       decision.Chain.Action,
			"target":       decision.Chain.Target,
			"next":         decision.Chain.Next,
			"terminal":     decision.Terminal,
			"display_text": decision.DisplayText,
		})
	} else if validation.Code == actionchain.ValidationOK {
		// Non-builtin valid actions become review/tool-card requests, same as CLI path.
		if a.eventBus != nil {
			a.eventBus.Emit(eventbus.EventSchedulerActionRequested, map[string]string{
				"action":     chain.Action,
				"target":     chain.Target,
				"next":       chain.Next,
				"trace_id":   traceID,
				"session_id": sessionID,
				"raw":        chain.Raw,
			})
		}
		debugtrace.Record("go.APIMessage.actionChain.routed", traceID, map[string]interface{}{
			"action":       chain.Action,
			"target":       chain.Target,
			"next":         chain.Next,
			"display_text": displayText,
		})
	}
	return skill_step.CLIResponse{
		Text:   displayText,
		Action: chain.Action,
		Target: chain.Target,
		Next:   chain.Next,
	}
}

func localAdapterReconnectHint(adapter adapter_registry.Adapter, cause string) string {
	name := strings.TrimSpace(adapter.Name)
	if name == "" {
		name = "本機模型"
	}
	cause = strings.TrimSpace(cause)
	if cause != "" {
		cause = "\n原始錯誤：" + cause
	}
	return fmt.Sprintf("本機模型「%s」連線受阻。系統已嘗試自動喚醒一次；如果仍然失敗，請點一下或雙擊左側的本機模型卡片來喚醒連線，確認狀態燈變綠後再送出訊息。%s", name, cause)
}

func (a *App) loadLLMAPIAdapterConfig(adapterID string) (llmAPIAdapterConfig, error) {
	ref := "llm_provider:" + adapterID + ":config"
	if a.secretStore != nil {
		if raw, err := a.secretStore.Load(ref); err == nil && strings.TrimSpace(raw) != "" {
			var cfg llmAPIAdapterConfig
			if err := json.Unmarshal([]byte(raw), &cfg); err == nil {
				cfg.ProviderID = strings.TrimSpace(cfg.ProviderID)
				cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
				cfg.Model = strings.TrimSpace(cfg.Model)
				cfg.Name = strings.TrimSpace(cfg.Name)
				return fillLLMAPIConfigDefaults(adapterID, cfg), nil
			}
		}
	}
	// For local adapters, pull endpoint + model from the adapter registry as fallback.
	if a.adapterRegistry != nil {
		if adapterInfo, adapterErr := a.adapterRegistry.GetStatus(adapterID); adapterErr == nil && adapterInfo.Kind == "local" && adapterInfo.Endpoint != "" {
			providerID := "ollama"
			if strings.Contains(adapterInfo.Endpoint, ":1234") {
				providerID = "lmstudio"
			}
			cfg := fillLLMAPIConfigDefaults(adapterID, llmAPIAdapterConfig{
				ProviderID: providerID,
				Name:       adapterInfo.Name,
				BaseURL:    adapterInfo.Endpoint,
				Model:      adapterInfo.Model, // e.g. "qwen2.5:14b"
			})
			return cfg, nil
		}
	}
	cfg := fillLLMAPIConfigDefaults(adapterID, llmAPIAdapterConfig{})
	if cfg.ProviderID == "" || cfg.BaseURL == "" {
		return cfg, fmt.Errorf("API adapter config not found for %s", adapterID)
	}
	return cfg, nil
}

func fillLLMAPIConfigDefaults(adapterID string, cfg llmAPIAdapterConfig) llmAPIAdapterConfig {
	if cfg.ProviderID == "" {
		cfg.ProviderID = inferLLMProviderID(adapterID)
	}
	switch cfg.ProviderID {
	case "deepseek":
		if cfg.Name == "" {
			cfg.Name = "DeepSeek"
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.deepseek.com"
		}
		if cfg.Model == "" {
			cfg.Model = "deepseek-chat"
		}
	case "openai":
		if cfg.Name == "" {
			cfg.Name = "OpenAI"
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.openai.com/v1"
		}
	case "xai":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.x.ai/v1"
		}
	case "openrouter":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://openrouter.ai/api/v1"
		}
	case "groq":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.groq.com/openai/v1"
		}
	case "ollama":
		if cfg.Name == "" {
			cfg.Name = "Ollama"
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "http://localhost:11434/v1"
		}
	case "lmstudio":
		if cfg.Name == "" {
			cfg.Name = "LM Studio"
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "http://localhost:1234/v1"
		}
	}
	return cfg
}

func inferLLMProviderID(adapterID string) string {
	rest := strings.TrimPrefix(adapterID, "llm-api-")
	if idx := strings.LastIndex(rest, "-"); idx > 0 {
		if _, err := time.ParseDuration(rest[idx+1:] + "ms"); err == nil {
			return rest[:idx]
		}
	}
	return rest
}

func isOpenAICompatibleProvider(providerID string) bool {
	switch providerID {
	case "deepseek", "openai", "xai", "openrouter", "mistral", "groq", "together", "perplexity", "fireworks", "huggingface", "cerebras", "nvidia-nim", "qwen", "kimi", "zhipu", "ollama", "lmstudio":
		return true
	default:
		return false
	}
}

func openAIChatCompletionsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

func parsedErrorMessage(resp openAIChatResponse) string {
	if resp.Error == nil {
		return ""
	}
	return resp.Error.Message
}

// ══════════════════════════════════════════════════════════════════════
// 上方互動路徑 — SendInspectorMessage
// 共用人格 + 三句限制 + 記憶體短歷史（不進下方 SentenceStore）
// ══════════════════════════════════════════════════════════════════════

// SendInspectorMessage 是上方互動專用 binding。
func (a *App) SendInspectorMessage(adapterID, sessionID, userText, traceID string) (*skill_step.CLIResponse, error) {
	log.Printf("SendInspectorMessage: adapter=%q text_len=%d", adapterID, len(userText))

	// ── 取得當前人格資料 ──
	persona := a.getActivePersona()

	// ── 組裝閒聊 prompt（人格 + 歷史 + 限制 + 輸入）──
	prompt := a.buildInspectorPrompt(persona, userText)

	// ── 解析 CLI 路徑 ──
	cliPath := ""
	if adapterID != "" && a.adapterRegistry != nil {
		if resolved, err := a.adapterRegistry.ResolveExecutable(adapterID); err == nil {
			cliPath = resolved
		}
	}

	// ── 確保 sidecar 已啟動 ──
	if err := a.ensureSidecarRunning(); err != nil {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return nil, err
	}
	if a.cliAdapter == nil {
		return &skill_step.CLIResponse{Text: "[stub] " + userText}, nil
	}

	// 上方互動自行帶短歷史，跳過下方主聊天的 SentenceStore。
	opts := skill_step.CLIMessageOptions{
		AdapterID:      adapterID,
		CLIPath:        cliPath,
		SessionID:      sessionID,
		UserText:       prompt,
		SystemPrompt:   a.buildSharedPersonaPrompt(persona),
		ContinuityKey:  conversationContinuityKey("inspector", sessionID),
		TraceID:        traceID,
		SkipContinuity: true,
	}
	resp, err := a.cliAdapter.SendMessage(opts)
	if err != nil {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return nil, err
	}
	if resp.Error != "" {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
	} else {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusOnline)
	}

	// 上方短歷史只在記憶體內，程式關閉後自然清空。
	if resp.Text != "" {
		a.appendInspectorHistory("user", userText)
		a.appendInspectorHistory("assistant", resp.Text)

		// 上方輸出只同步 statusRail/greeting，不進下方 messages。
		view := a.statusRail.SetText(resp.Text)
		a.eventBus.Emit(eventbus.EventStatusRailUpdated, view)
	}

	return &resp, nil
}

// ClearInspectorHistory 清除閒聊歷史。前端關閉互動視窗時呼叫。
func (a *App) ClearInspectorHistory() {
	a.inspectorMu.Lock()
	defer a.inspectorMu.Unlock()
	a.inspectorHistory = nil
	log.Printf("ClearInspectorHistory: cleared")
}

// getActivePersona 從 settings 取得目前啟用的人格。
func (a *App) getActivePersona() settings.Persona {
	state := a.settingsService.State()
	for _, p := range state.Personas {
		if p.ID == state.ActivePersonaID {
			return p
		}
	}
	// fallback：回傳第一個，或空值
	if len(state.Personas) > 0 {
		return state.Personas[0]
	}
	return settings.Persona{Name: "助手"}
}

func conversationContinuityKey(lane, sessionID string) string {
	// lane 將上下對話拆成不同歷史桶，例如 composer / inspector。
	lane = strings.TrimSpace(lane)
	if lane == "" {
		lane = "default"
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = "default"
	}
	return lane + ":" + sessionID
}

// buildSharedPersonaPrompt keeps persona data compact; user-written persona
// fields such as "請簡短回答" remain global rules for both lanes.
func (a *App) buildSharedPersonaPrompt(persona settings.Persona) string {
	name := strings.TrimSpace(persona.Name)
	if name == "" {
		name = "憂樂傻酷"
	}

	fields := []string{"角色=" + name, "語言=繁中"}
	if value := strings.TrimSpace(persona.Identity); value != "" {
		fields = append(fields, "身份="+value)
	}
	if value := strings.TrimSpace(persona.Personality); value != "" {
		fields = append(fields, "個性="+value)
	}
	if value := strings.TrimSpace(persona.Scenario); value != "" {
		fields = append(fields, "情境="+value)
	}
	if value := strings.TrimSpace(persona.Description); value != "" {
		fields = append(fields, "提示="+value)
	}
	if value := strings.TrimSpace(persona.ReplyStrategy); value != "" {
		fields = append(fields, "回答策略="+expandReplyStrategyPrompt(value))
	}
	return "P: " + strings.Join(fields, "；") + "\n"
}

func expandReplyStrategyPrompt(value string) string {
	switch strings.TrimSpace(value) {
	case "concise", "節省模式":
		return "節省模式：以極短、機械、肯定的方式回答。只保留重要關鍵句，不寒暄，不說「好、收到、沒錯、我來」等回應詞。除非必要，不解釋背景、不補充延伸、不列替代方案。若使用者要求操作，直接給最短可執行資訊。"
	case "reflective_question", "反問模式":
		return "反問模式：回答前固定提出1到3句反問，幫使用者確認風險、代價、方向或需求是否真的成立。反問要具體，不空泛。可以指出取捨，例如「這樣會增加實作難度，你確定要用這方式嗎？」反問後可給簡短建議，但不要急著完整執行。"
	case "suggestive", "建議模式":
		return "建議模式：優先提供2到3個可選方案，允許有創意但必須可落地。每個選項用短句說明特色、適合情境與代價。最後可以標出推薦選項，但不要替使用者完全決定，保留選擇感。"
	case "teacher_question", "提問模式":
		return "提問模式：採老師式教學。用清楚、有順序的問題引導使用者理解概念、釐清條件、推導答案。每次最多提出1到3個問題。必要時先給一小段概念提示，再問下一步。避免直接把完整答案一次塞滿。"
	case "confirm_before_action", "執行模式":
		return "執行模式：做任何事情前，先列出預計步驟，並詢問使用者「這樣對嗎？」或等價確認。取得確認後才開始執行。步驟要具體、可檢查、避免過長。若涉及檔案修改、命令執行、設定變更或不可逆操作，更必須先確認。"
	case "decision_tree", "分析模式":
		return "分析模式：結論先行。先用1到3句說出判斷結果，再列出決策樹。決策樹應呈現條件、分支、建議行動與風險。適合複雜選擇、架構判斷、問題診斷與規格討論。避免只做散亂條列。"
	case "companion", "陪伴模式":
		return "陪伴模式：像自然可靠的朋友一樣回應，可以多說一點，語氣放鬆、有溫度、有陪伴感。允許輕微閒聊、共感與補充觀察。不要過度機械，也不要急著把每句話都變成任務。仍需在重要資訊上保持清楚可靠。"
	case "creative", "創作模式":
		return "創作模式：優先產生多個有風格差異的創意方向，而不是只給單一保守答案。可以使用比喻、命名、情境、語氣變體、視覺風格與故事感來擴展想法。輸出時通常給3到5個候選版本，並簡短說明每個版本的氣質與適用情境。若使用者已給明確限制，創意必須在限制內發揮，不要偏離任務。"
	default:
		return strings.TrimSpace(value)
	}
}

// buildInspectorPrompt 組裝上方互動專用 prompt。
// 結構：共用人格 → 上方互動歷史 → 上方輸出限制 → 當前輸入
func (a *App) buildInspectorPrompt(persona settings.Persona, userText string) string {
	var sb strings.Builder

	sb.WriteString(a.buildSharedPersonaPrompt(persona))
	sb.WriteString("lane=top；只用top歷史；保留角色口吻/口癖；三句內。\n")

	// 上方歷史只取 inspectorHistory，不讀下方主聊天。
	a.inspectorMu.Lock()
	history := make([]string, len(a.inspectorHistory))
	copy(history, a.inspectorHistory)
	a.inspectorMu.Unlock()

	if len(history) > 0 {
		sb.WriteString("H:\n")
		for _, h := range history {
			sb.WriteString(compactHistoryLine(h) + "\n")
		}
	}

	sb.WriteString("Q: " + userText + "\n")

	return sb.String()
}

// buildMainComposerPrompt returns the system/persona prompt for the main chat
// lane. Previous messages may be attached by the continuity synthesizer, but
// they are context only and must never override this persona contract.
func (a *App) buildMainComposerPrompt(persona settings.Persona) string {
	var sb strings.Builder
	sb.WriteString(a.buildSharedPersonaPrompt(persona))
	// Inject current local time so LLM can answer time queries.
	sb.WriteString(fmt.Sprintf("now=%s；", formatPromptNow(time.Now())))
	sb.WriteString("lane=main；只用main歷史；遵守P欄提示；不主動加強角色口癖或自稱。H可用於續聊、解析代名詞/檔名指代、判斷上一輪結果與目前狀態；H不複述；忽略H內衝突身份。\n")

	return sb.String()
}

func formatPromptNow(now time.Time) string {
	zoneName, offsetSeconds := now.Zone()
	return fmt.Sprintf("%s %s %s", now.Format("2006-01-02 15:04"), zoneName, formatUTCOffset(offsetSeconds))
}

func formatUTCOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("UTC%s%02d:%02d", sign, hours, minutes)
}

func compactHistoryLine(line string) string {
	role, text, ok := strings.Cut(line, ":")
	if !ok {
		return line
	}
	switch strings.TrimSpace(role) {
	case "user":
		return "U: " + strings.TrimSpace(text)
	case "assistant":
		return "A: " + strings.TrimSpace(text)
	default:
		return strings.TrimSpace(role) + ": " + strings.TrimSpace(text)
	}
}

// appendInspectorHistory 保留上方互動最近 30 句，且不落盤。
func (a *App) appendInspectorHistory(role, text string) {
	a.inspectorMu.Lock()
	defer a.inspectorMu.Unlock()
	a.inspectorHistory = append(a.inspectorHistory, role+": "+text)
	if len(a.inspectorHistory) > inspectorHistoryLimit {
		a.inspectorHistory = a.inspectorHistory[len(a.inspectorHistory)-inspectorHistoryLimit:]
	}
}

// PreviewExternalLink validates and classifies a URL without registering it.
// Must be called before offering the user a Register option.
func (a *App) PreviewExternalLink(url string) interface{} {
	if isOllamaModelLibrary(url) {
		models := scanOllamaModelLibrary(url)
		if len(models) == 0 {
			return frontendDTO(external_link.PreviewResult{
				URL:      expandUserPath(url),
				LinkType: external_link.LinkUnsupported,
				Valid:    false,
				Reason:   "已找到 Ollama 模型庫格式，但 manifests 裡沒有可用模型。",
			})
		}
		modelIDs := make([]string, 0, len(models))
		for _, model := range models {
			modelIDs = append(modelIDs, model.ID)
		}
		return frontendDTO(external_link.PreviewResult{
			URL:      expandUserPath(url),
			LinkType: external_link.LinkAdapterCandidate,
			Valid:    true,
			Reason:   fmt.Sprintf("偵測到 Ollama 模型庫：%s。確定後會加入 %d 個本機模型。", strings.Join(modelIDs, "、"), len(modelIDs)),
		})
	}
	if cli, err := adapter_registry.ResolveCustomCLI("", url); err == nil && cli.Found {
		return frontendDTO(external_link.PreviewResult{
			URL:      cli.Path,
			LinkType: external_link.LinkAdapterCandidate,
			Valid:    true,
			Reason:   fmt.Sprintf("偵測到本機 CLI：%s", cli.Name),
		})
	}
	return frontendDTO(a.linkService.Preview(url))
}

// ListLLMProviderWhitelist 回傳引用連結用的 LLM/API Provider 白名單。
func (a *App) ListLLMProviderWhitelist() interface{} {
	return frontendDTO(external_link.ListLLMProviderWhitelist())
}

// RegisterExternalLink registers an external link after user confirmation.
// documentation links are accepted but flagged reference-only (never enter routing/execution area).
// adapter 類型的 URL 由 adapter_registry 管理，不再經由 external_link。
func (a *App) RegisterExternalLink(url, label string) (interface{}, error) {
	if isOllamaModelLibrary(url) {
		models := scanOllamaModelLibrary(url)
		if len(models) == 0 {
			return nil, fmt.Errorf("ollama: model library %q has no manifest models", url)
		}
		for _, model := range models {
			if strings.TrimSpace(model.ID) == "" {
				continue
			}
			adapterID := "local-ollama-" + sanitizeAdapterID(model.ID)
			if err := a.adapterRegistry.RegisterLocal(adapterID, "Ollama - "+model.ID, "O", "http://localhost:11434/v1", model.ID, expandUserPath(url)); err != nil {
				return nil, err
			}
			if a.eventBus != nil {
				a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
					"adapter_id": adapterID,
					"kind":       "local",
				})
			}
		}
		return frontendDTO(&external_link.ExternalLink{
			ID:       "ollama-model-library",
			URL:      expandUserPath(url),
			LinkType: external_link.LinkAdapterCandidate,
			Label:    fmt.Sprintf("Ollama 模型庫（%d 個模型）", len(models)),
		}), nil
	}
	// 先檢查是否為 CLI adapter 路徑——如果是，只走 adapter_registry
	if cli, err := a.adapterRegistry.RegisterCustomCLI("", url); err == nil && cli.Found {
		if a.eventBus != nil {
			a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
				"adapter_id": cli.AdapterID,
				"path":       cli.Path,
			})
		}
		// 回傳一個 ExternalLink DTO 供前端顯示，但不實際存入 external_link registry
		return frontendDTO(&external_link.ExternalLink{
			ID:       cli.AdapterID,
			URL:      cli.Path,
			LinkType: external_link.LinkAdapterCandidate,
			Label:    cli.Name,
		}), nil
	}
	link, err := a.linkService.Register(url, label)
	return frontendDTO(link), err
}

// ListExternalLinksByType returns registered links of a given type.
// Callers MUST NOT pass documentation links to routing or execution areas.
func (a *App) ListExternalLinksByType(linkType string) interface{} {
	return frontendDTO(a.linkService.ListByType(external_link.LinkType(linkType)))
}

func (a *App) ListReferenceFiles() ([]ReferenceFile, error) {
	referenceDir := filepath.Join(appDataRoot(), "data", "references", "files")
	if err := os.MkdirAll(referenceDir, 0o700); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(referenceDir)
	if err != nil {
		return nil, err
	}
	files := make([]ReferenceFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		files = append(files, ReferenceFile{
			Name:   name,
			Path:   filepath.Join(referenceDir, name),
			Source: "library",
			Status: "ready",
			Detail: "已從引用庫載入",
		})
	}
	return files, nil
}

func (a *App) ImportReferenceFile(sourcePath string) (ReferenceFile, error) {
	if sourcePath == "" {
		return ReferenceFile{}, fmt.Errorf("reference: source path is empty")
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return ReferenceFile{}, err
	}
	defer source.Close()

	referenceDir := filepath.Join(appDataRoot(), "data", "references", "files")
	if err := os.MkdirAll(referenceDir, 0o700); err != nil {
		return ReferenceFile{}, err
	}

	baseName := filepath.Base(sourcePath)
	targetPath := filepath.Join(referenceDir, baseName)
	if _, err := os.Stat(targetPath); err == nil {
		ext := filepath.Ext(baseName)
		stem := baseName[:len(baseName)-len(ext)]
		targetPath = filepath.Join(referenceDir, fmt.Sprintf("%s-%d%s", stem, time.Now().UnixNano(), ext))
		baseName = filepath.Base(targetPath)
	}

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return ReferenceFile{}, err
	}
	defer target.Close()
	if _, err := io.Copy(target, source); err != nil {
		return ReferenceFile{}, err
	}
	return ReferenceFile{Name: baseName, Path: targetPath, Source: "library", Status: "ready", Detail: "已複製到引用庫"}, nil
}

// PreparePackageInstall quarantines a dropped package for review.
// The package is NOT installed and NOT added to tool_registry at this stage.
// manifestJSON is a JSON-encoded package_import.PackageManifest (best-effort; falls back to minimal defaults).
func (a *App) PreparePackageInstall(sourcePath, manifestJSON string) (interface{}, error) {
	var manifest package_import.PackageManifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		manifest = package_import.PackageManifest{
			Name:       sourcePath,
			Version:    "0.0.1",
			SourcePath: sourcePath,
			RiskTag:    "unknown",
		}
	}
	manifest.SourcePath = sourcePath
	pending, err := a.packageService.PrepareInstall(sourcePath, manifest)
	return frontendDTO(pending), err
}

func (a *App) PreparePackageInstallPayload(sourceName, fileContent, manifestJSON string) (interface{}, error) {
	var manifest package_import.PackageManifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		manifest = package_import.PackageManifest{
			Name:       sourceName,
			Version:    "0.0.1",
			SourcePath: sourceName,
			RiskTag:    "unknown",
		}
	}
	manifest.SourcePath = sourceName
	pending, err := a.packageService.PrepareInstallFromBytes(sourceName, []byte(fileContent), manifest)
	return frontendDTO(pending), err
}

// ConfirmPackageInstall confirms a quarantined import after user review.
// After this returns, the caller may write to tool_registry.
func (a *App) ConfirmPackageInstall(importID string) error {
	_, err := a.packageService.ConfirmInstall(importID)
	return err
}

// RejectPackageInstall rejects a quarantined import; removes quarantine package.
// reason should be "rejected" or "cancelled".
// #I-1002: 同時生成 rejected Review Card 寫入 review_archive.json，
// 並在 reviewService 中新增一張即時卡片供前端 Review Panel 顯示。
func (a *App) RejectPackageInstall(importID, reason string) error {
	summary, err := a.packageService.RejectInstall(importID, reason)
	if err != nil {
		return err
	}
	// 寫入 review_archive.json 留下持久化軌跡
	plainReason := fmt.Sprintf("套件「%s」安裝被拒絕：%s", summary.Name, reason)
	engineerReason := fmt.Sprintf("import_id=%s risk=%s reason=%s", summary.ImportID, summary.RiskTag, reason)
	_, _ = a.reviewArchive.AppendRejected("package_import", importID, plainReason, engineerReason, reason)
	// 同時加入 reviewService 讓前端即時看到
	a.reviewService.AddLegacyCard(review.LevelBackground, "package_import", importID, plainReason, engineerReason)
	// 通知前端 Review Panel 刷新
	a.eventBus.Emit(eventbus.EventReviewCardAdded, map[string]string{"source": "package_import", "id": importID})
	return nil
}

// ListPendingPackages returns all quarantined (unconfirmed) package imports.
func (a *App) ListPendingPackages() interface{} {
	return frontendDTO(a.packageService.ListPending())
}

// ListReviewArchive 回傳 review_archive.json 中的所有卡片（含 rejected 軌跡）。
// #I-1002: 前端 Review Panel 歷史紀錄區塊用此拉取 rejected 卡片。
func (a *App) ListReviewArchive() ([]review.ArchivedCard, error) {
	return a.reviewArchive.ListArchived()
}

// GetToolVisibility returns the rendered visibility state (rank only) for all installed tools.
// 工具可用性（Available / NeedsReauth）由 tools / adapter_registry 套件管理。
func (a *App) GetToolVisibility() interface{} {
	toolList := a.toolsService.List()
	toolIDs := make([]string, len(toolList))
	for i, t := range toolList {
		toolIDs[i] = t.ID
	}
	return frontendDTO(a.prefStore.BuildVisibilityList(toolIDs, ""))
}

// SetToolPreference records a user's explicit drag-position preference for a tool.
// scope: "sub" | "main" | "global" | "routing". dagRevision required when scope="main".
func (a *App) SetToolPreference(toolID, scope, dagRevision string, rank int) error {
	return a.prefStore.SetPreference(preference.ToolPreferenceEntry{
		ToolID:      toolID,
		Rank:        preference.PreferenceRank(rank),
		Scope:       preference.PreferenceScope(scope),
		DAGRevision: dagRevision,
	})
}

// ---------------------------------------------------------------------------
// #55 — Skill Context Orchestration Wails Bindings
// ---------------------------------------------------------------------------

// ScanSkillFolder scans a folder at path, caches the preview, and returns it.
func (a *App) ScanSkillFolder(path string) (interface{}, error) {
	preview, err := a.skillArchive.ScanFolder(path)
	if err != nil {
		return nil, err
	}
	a.cacheMu.Lock()
	a.previewCache[preview.PreviewID] = preview
	a.cacheMu.Unlock()
	return frontendDTO(preview), nil
}

// ConfirmSkillArchive confirms a pending scan preview and archives the skill.
func (a *App) ConfirmSkillArchive(previewID string) (interface{}, error) {
	a.cacheMu.Lock()
	preview, ok := a.previewCache[previewID]
	a.cacheMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("app: ConfirmSkillArchive: preview %q not found", previewID)
	}
	manifest, err := a.skillArchive.ConfirmArchive(preview)
	return frontendDTO(manifest), err
}

// ListArchivedSkills returns all skill manifests currently in the archive.
func (a *App) ListArchivedSkills() (interface{}, error) {
	manifests, err := a.skillArchive.ListArchived()
	return frontendDTO(manifests), err
}

// ResolveSkillForAction parses actionTarget (format: "動作ㄌ目標"), resolves it
// against the archive, caches the result, and returns it.
func (a *App) ResolveSkillForAction(actionTarget string, sessionID string) (*skill_step.ResolveResult, error) {
	at, err := skill_step.ParseActionTarget(actionTarget)
	if err != nil {
		return nil, err
	}
	result, err := a.skillRouter.Resolve(at, sessionID)
	if err != nil {
		return nil, err
	}
	a.cacheMu.Lock()
	a.resolveCache[result.ResolveID] = result
	a.cacheMu.Unlock()
	return result, nil
}

// BuildSkillContext looks up a cached ResolveResult, builds the Injection,
// stores it in the InjectionStore, and records it in the audit log.
func (a *App) BuildSkillContext(resolveID string, sessionID string) (*skill_step.Injection, error) {
	a.cacheMu.Lock()
	result, ok := a.resolveCache[resolveID]
	a.cacheMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("app: BuildSkillContext: resolve result %q not found", resolveID)
	}
	if result.SelectedSkillID == "" || len(result.Candidates) == 0 {
		return nil, fmt.Errorf("app: BuildSkillContext: no selected skill in resolve result %q", resolveID)
	}
	if sessionID == "" {
		sessionID = result.SessionID
	}
	if sessionID == "" {
		sessionID = a.globalSessionID
	}

	// Find the selected candidate.
	var selected skill_step.Candidate
	for _, c := range result.Candidates {
		if c.SkillID == result.SelectedSkillID {
			selected = c
			break
		}
	}

	// Load manifest for resource refs.
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return nil, err
	}
	var manifest skill_step.SkillManifest
	for _, m := range manifests {
		if m.SkillID == result.SelectedSkillID {
			manifest = m
			break
		}
	}

	inj := skill_step.BuildInjection(sessionID, selected, manifest)
	a.skillInjections.Set(sessionID, inj)

	_ = a.skillAudit.Record(skill_step.InjectionAudit{
		Timestamp:     time.Now(),
		SessionID:     sessionID,
		SkillID:       inj.SkillID,
		ResolveStatus: string(result.Status),
		Reason:        inj.Reason,
		SummaryHash:   inj.SummaryHash,
		Risk:          inj.Risk,
	})

	return inj, nil
}

// ClearSkillContext clears the active skill injection for sessionID and records
// the clear event in the audit log.
func (a *App) ClearSkillContext(sessionID string, reason string) error {
	inj := a.skillInjections.Get(sessionID)
	a.skillInjections.Clear(sessionID)

	skillID := ""
	summaryHash := ""
	risk := ""
	if inj != nil {
		skillID = inj.SkillID
		summaryHash = inj.SummaryHash
		risk = inj.Risk
	}

	now := time.Now()
	return a.skillAudit.Record(skill_step.InjectionAudit{
		Timestamp:   now,
		SessionID:   sessionID,
		SkillID:     skillID,
		SummaryHash: summaryHash,
		Risk:        risk,
		ClearedAt:   &now,
		ClearReason: reason,
	})
}

// GetRecentSkillInjections returns the 2 most recent audit entries for sessionID.
func (a *App) GetRecentSkillInjections(sessionID string) (interface{}, error) {
	injections, err := a.skillAudit.Recent(sessionID, 2)
	return frontendDTO(injections), err
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func toNoticeTemplate(template string) statusrail.NoticeTemplate {
	switch template {
	case string(statusrail.NoticeNeedsConfirm):
		return statusrail.NoticeNeedsConfirm
	case string(statusrail.NoticeCompleted):
		return statusrail.NoticeCompleted
	case string(statusrail.NoticeError):
		return statusrail.NoticeError
	case string(statusrail.NoticeReviewPending):
		return statusrail.NoticeReviewPending
	case string(statusrail.NoticeAssetValidated):
		return statusrail.NoticeAssetValidated
	default:
		return statusrail.NoticeNeedsConfirm
	}
}

func toNoticePriority(priority string) statusrail.NoticePriority {
	switch priority {
	case string(statusrail.PriorityCritical):
		return statusrail.PriorityCritical
	case string(statusrail.PriorityDestructive):
		return statusrail.PriorityDestructive
	default:
		return statusrail.PriorityNormal
	}
}

// =============================================================================
// 遺留待重建能力 — Wails Bindings (#1–#7)
// =============================================================================

// --- #1 Adapter Registry ---

// AutoDetectCLI 掃描本機 PATH + 常見安裝路徑偵測已安裝的 CLI。
// 只偵測不註冊——回傳偵測結果供 UI 顯示，由使用者選擇要啟用哪些。
func (a *App) AutoDetectCLI() interface{} {
	return frontendDTO(a.adapterRegistry.AutoDetect())
}

// EnableDetectedCLI 使用者在 UI 上選擇要啟用的 CLI 後呼叫，將偵測到的 CLI 註冊到 registry。
func (a *App) EnableDetectedCLI(adapterID string) error {
	if err := a.adapterRegistry.EnableDetectedCLI(adapterID); err != nil {
		return err
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"adapter_id": adapterID,
		})
	}
	return nil
}

// RegisterCustomCLI 讓使用者手動註冊一個自訂 CLI adapter。
// name: 顯示名稱, path: 可執行檔完整路徑。
func (a *App) RegisterCustomCLI(name, path string) (interface{}, error) {
	result, err := a.adapterRegistry.RegisterCustomCLI(name, path)
	return frontendDTO(result), err
}

// RegisterLLMAPIAdapter 第一版只建立 sidebar API adapter 入口；API key/baseURL/model
// 的實際呼叫流程會接在下一階段，名稱修改沿用現有 rename 彈窗。
func (a *App) RegisterLLMAPIAdapter(providerID, providerName, baseURL, model, apiKey string) (interface{}, error) {
	providerID = strings.TrimSpace(providerID)
	providerName = strings.TrimSpace(providerName)
	if providerID == "" {
		providerID = "generic-api"
	}
	if providerName == "" {
		providerName = "LLM API"
	}
	// SEC-03: 驗證 baseURL 防止 SSRF
	needConfirm, host, urlErr := urlsafe.ValidateLLMBaseURL(providerID, baseURL)
	if urlErr != nil {
		return nil, fmt.Errorf("baseURL 驗證失敗 (%s): %w", baseURL, urlErr)
	}
	if needConfirm {
		// 私有 IP 需前端確認，回傳特殊 DTO 讓前端彈確認框
		return frontendDTO(map[string]interface{}{
			"need_confirm":  true,
			"confirm_type":  "private_network",
			"hostname":      host,
			"original_url":  strings.TrimSpace(baseURL),
			"provider_id":   providerID,
			"provider_name": providerName,
			"model":         strings.TrimSpace(model),
		}), nil
	}

	return a.registerLLMAPIAdapterInternal(providerID, providerName, baseURL, model, apiKey)
}

// ConfirmRegisterLLMAPIAdapter SEC-03: 使用者已確認私有網路連線後呼叫。
func (a *App) ConfirmRegisterLLMAPIAdapter(providerID, providerName, baseURL, model, apiKey string) (interface{}, error) {
	providerID = strings.TrimSpace(providerID)
	providerName = strings.TrimSpace(providerName)
	if providerID == "" {
		providerID = "generic-api"
	}
	if providerName == "" {
		providerName = "LLM API"
	}
	log.Printf("ConfirmRegisterLLMAPIAdapter: user confirmed private network %s", baseURL)
	return a.registerLLMAPIAdapterInternal(providerID, providerName, baseURL, model, apiKey)
}

// registerLLMAPIAdapterInternal 實際建立 adapter 的內部函式。
func (a *App) registerLLMAPIAdapterInternal(providerID, providerName, baseURL, model, apiKey string) (interface{}, error) {
	adapterID := fmt.Sprintf("llm-api-%s-%d", providerID, time.Now().UnixMilli())
	secretRef := "llm_provider:" + adapterID + ":api_key"
	if strings.TrimSpace(apiKey) != "" {
		if err := a.secretStore.Store(secretRef, strings.TrimSpace(apiKey)); err != nil {
			return nil, err
		}
	}
	configRef := "llm_provider:" + adapterID + ":config"
	config := llmAPIAdapterConfig{
		ProviderID: providerID,
		Name:       providerName,
		BaseURL:    strings.TrimSpace(baseURL),
		Model:      strings.TrimSpace(model),
	}
	if configRaw, err := json.Marshal(config); err == nil {
		if err := a.secretStore.Store(configRef, string(configRaw)); err != nil {
			return nil, err
		}
	}
	icon := "A"
	if runes := []rune(providerName); len(runes) > 0 {
		icon = strings.ToUpper(string(runes[0]))
	}
	if err := a.adapterRegistry.RegisterAPI(adapterID, providerName, icon); err != nil {
		return nil, err
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"adapter_id": adapterID,
			"kind":       "api",
		})
	}
	return frontendDTO(map[string]string{
		"adapter_id":  adapterID,
		"name":        providerName,
		"kind":        "api",
		"base_url":    strings.TrimSpace(baseURL),
		"model":       strings.TrimSpace(model),
		"api_key_ref": secretRef,
		"config_ref":  configRef,
	}), nil
}

// RenameAdapter 更新左側 Adapter 顯示名稱。API adapter 註冊後也走這個命名流程。
func (a *App) RenameAdapter(adapterID, displayName string) (interface{}, error) {
	adapter, err := a.adapterRegistry.Rename(adapterID, displayName)
	if err == nil && a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"adapter_id": adapterID,
		})
	}
	return frontendDTO(adapter), err
}

// UnregisterAdapter 移除一個已註冊的 adapter。
func (a *App) UnregisterAdapter(adapterID string) error {
	return a.adapterRegistry.Unregister(adapterID)
}

// ── Local Model (Ollama / LM Studio) ────────────────────────────────

// LocalModelDetectResult describes a locally installed model found by ScanLocalModels.
type LocalModelDetectResult struct {
	AdapterID string `json:"adapter_id"` // e.g. "local-ollama-qwen2.5:14b"
	Name      string `json:"name"`       // display name, e.g. "Ollama - qwen2.5:14b"
	ModelID   string `json:"model_id"`   // raw model ID from ollama list
	Provider  string `json:"provider"`   // "ollama" or "lmstudio"
	Endpoint  string `json:"endpoint"`   // e.g. "http://localhost:11434/v1"
	Found     bool   `json:"found"`
}

// ScanLocalModels detects locally running Ollama / LM Studio models.
// Returns detection results without auto-registering — the user picks which to enable.
func (a *App) ScanLocalModels() interface{} {
	var results []LocalModelDetectResult

	// Scan Ollama
	ollamaModels := scanOllamaModels()
	for _, m := range ollamaModels {
		results = append(results, LocalModelDetectResult{
			AdapterID: "local-ollama-" + sanitizeAdapterID(m.ID),
			Name:      "Ollama - " + m.ID,
			ModelID:   m.ID,
			Provider:  "ollama",
			Endpoint:  "http://localhost:11434/v1",
			Found:     true,
		})
	}

	// Scan LM Studio
	lmsModels := scanLMStudioModels()
	for _, m := range lmsModels {
		results = append(results, LocalModelDetectResult{
			AdapterID: "local-lmstudio-" + sanitizeAdapterID(m.ID),
			Name:      "LM Studio - " + m.ID,
			ModelID:   m.ID,
			Provider:  "lmstudio",
			Endpoint:  "http://localhost:1234/v1",
			Found:     true,
		})
	}

	return frontendDTO(results)
}

// EnableLocalModel registers a detected local model into the adapter list.
func (a *App) EnableLocalModel(adapterID, name, modelID, provider, endpoint string) error {
	icon := "◉"
	if provider == "lmstudio" {
		icon = "◈"
	}
	if err := a.adapterRegistry.RegisterLocal(adapterID, name, icon, endpoint, modelID); err != nil {
		return err
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"adapter_id": adapterID,
			"kind":       "local",
		})
	}
	return nil
}

// WakeLocalAdapter is a user-triggered local-model wake-up action. It only
// starts known local runtimes from fixed discovery paths; arbitrary adapter
// paths are never executed.
func (a *App) WakeLocalAdapter(adapterID string) (interface{}, error) {
	if a.adapterRegistry == nil {
		return nil, fmt.Errorf("adapter registry is not available")
	}
	adapter, err := a.adapterRegistry.GetStatus(adapterID)
	if err != nil {
		return nil, err
	}
	if adapter.Kind != "local" {
		return nil, fmt.Errorf("adapter %s is not a local model", adapterID)
	}
	if strings.Contains(adapter.Endpoint, ":11434") || strings.Contains(strings.ToLower(adapter.ID+" "+adapter.Name), "ollama") {
		result, err := a.wakeOllamaAdapter(adapter)
		return frontendDTO(result), err
	}
	if strings.Contains(adapter.Endpoint, ":1234") {
		if pingOpenAIModelsEndpoint(adapter.Endpoint, 800*time.Millisecond) {
			a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusOnline)
			return frontendDTO(map[string]string{"status": "online", "message": "LM Studio 已在線"}), nil
		}
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return nil, fmt.Errorf("LM Studio 尚未啟動；請先在 LM Studio 啟動 local server")
	}
	return nil, fmt.Errorf("unknown local adapter endpoint: %s", adapter.Endpoint)
}

func (a *App) wakeOllamaAdapter(adapter adapter_registry.Adapter) (map[string]string, error) {
	baseURL := ollamaBaseURL(adapter.Endpoint)
	if pingOllamaTags(baseURL, 800*time.Millisecond) {
		a.setAdapterRuntimeStatus(adapter.ID, adapter_registry.StatusOnline)
		return map[string]string{"status": "online", "message": "Ollama 已在線"}, nil
	}

	ollamaPath := resolveOllamaExecutable()
	if ollamaPath == "" {
		a.setAdapterRuntimeStatus(adapter.ID, adapter_registry.StatusDegraded)
		return nil, fmt.Errorf("找不到 Ollama CLI，請安裝 Ollama 或加入 /opt/homebrew/bin/ollama")
	}

	cmd := exec.Command(ollamaPath, "serve")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Env = os.Environ()
	if modelDir := resolveOllamaModelDir(adapter.Path); modelDir != "" {
		cmd.Env = append(cmd.Env, "OLLAMA_MODELS="+modelDir)
	}
	if err := cmd.Start(); err != nil {
		a.setAdapterRuntimeStatus(adapter.ID, adapter_registry.StatusDegraded)
		return nil, err
	}
	go func() { _ = cmd.Wait() }()

	for i := 0; i < 30; i++ {
		if pingOllamaTags(baseURL, 300*time.Millisecond) {
			a.setAdapterRuntimeStatus(adapter.ID, adapter_registry.StatusOnline)
			return map[string]string{"status": "online", "message": "Ollama 已喚醒"}, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	a.setAdapterRuntimeStatus(adapter.ID, adapter_registry.StatusDegraded)
	return nil, fmt.Errorf("Ollama 已嘗試啟動，但 API 尚未回應")
}

func resolveOllamaModelDir(adapterPath string) string {
	for _, candidate := range []string{
		adapterPath,
		os.Getenv("OLLAMA_MODELS"),
		userOllamaModelDir(),
		defaultOllamaModelDir(),
	} {
		candidate = expandUserPath(candidate)
		if isOllamaModelLibrary(candidate) {
			return candidate
		}
	}
	return ""
}

func userOllamaModelDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "ollama")
}

func defaultOllamaModelDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".ollama", "models")
}

func ollamaBaseURL(endpoint string) string {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasSuffix(base, "/v1") {
		base = strings.TrimSuffix(base, "/v1")
	}
	if base == "" {
		base = "http://localhost:11434"
	}
	return base
}

func pingOllamaTags(baseURL string, timeout time.Duration) bool {
	client := http.Client{Timeout: timeout}
	resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func pingOpenAIModelsEndpoint(endpoint string, timeout time.Duration) bool {
	client := http.Client{Timeout: timeout}
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasSuffix(base, "/v1") {
		base += "/models"
	} else {
		base += "/v1/models"
	}
	resp, err := client.Get(base)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// sanitizeAdapterID replaces characters that are invalid in adapter IDs.
func sanitizeAdapterID(s string) string {
	r := strings.NewReplacer(":", "-", "/", "-", " ", "-")
	return strings.ToLower(r.Replace(s))
}

// ReorderAdapters updates the persisted sidebar order for CLI/API adapters.
func (a *App) ReorderAdapters(orderJSON string) error {
	var orderIDs []string
	if err := json.Unmarshal([]byte(orderJSON), &orderIDs); err != nil {
		return err
	}
	if err := a.adapterRegistry.Reorder(orderIDs); err != nil {
		return err
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterListChanged, map[string]string{
			"reason": "reordered",
		})
	}
	return nil
}

// ListAvailableAdapters returns CLI adapters plus callable sub tabs for the sidebar.
func (a *App) ListAvailableAdapters() interface{} {
	items := make([]map[string]interface{}, 0)
	for _, adapter := range a.adapterRegistry.ListAvailable() {
		kind := strings.TrimSpace(adapter.Kind)
		if kind == "" {
			kind = "cli"
		}
		items = append(items, map[string]interface{}{
			"id":         adapter.ID,
			"name":       adapter.Name,
			"icon":       adapter.Icon,
			"path":       adapter.Path,
			"endpoint":   adapter.Endpoint,
			"model":      adapter.Model,
			"status":     adapter.Status,
			"last_check": adapter.LastCheck,
			"kind":       kind,
		})
	}

	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	for _, sub := range listSubagentTabs(projectRoot) {
		items = append(items, map[string]interface{}{
			"id":     sub.id,
			"name":   sub.label,
			"icon":   "⊕",
			"status": "offline",
			"kind":   "sub",
		})
	}
	return frontendDTO(items)
}

// GetAdapterStatus returns the status of a specific adapter by ID.
func (a *App) GetAdapterStatus(adapterID string) (interface{}, error) {
	status, err := a.adapterRegistry.GetStatus(adapterID)
	return frontendDTO(status), err
}

// SetAdapterStatus updates the connectivity status of an adapter.
func (a *App) SetAdapterStatus(adapterID string, status string) error {
	err := a.adapterRegistry.SetStatus(adapterID, adapter_registry.Status(status))
	if err == nil {
		a.eventBus.Emit(eventbus.EventAdapterStatusChanged, map[string]string{
			"adapter_id": adapterID, "status": status,
		})
	}
	return err
}

func (a *App) setAdapterRuntimeStatus(adapterID string, status adapter_registry.Status) {
	if adapterID == "" || a.adapterRegistry == nil {
		return
	}
	if err := a.adapterRegistry.SetStatus(adapterID, status); err != nil {
		return
	}
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventAdapterStatusChanged, map[string]string{
			"adapter_id": adapterID,
			"status":     string(status),
		})
	}
}

// --- v3.6.1 GatewaySentinel / Source Trust (§9) ---

// ClassifySource 分類外部來源的信任等級。
func (a *App) ClassifySource(url, contentSnippet string, visualFlags []string) source_trust.SourceTrustEvidence {
	vf := make([]source_trust.VisualFlag, len(visualFlags))
	for i, f := range visualFlags {
		vf[i] = source_trust.VisualFlag(f)
	}
	evidence := source_trust.Classify(url, contentSnippet, vf)
	// 填入 allowlist 狀態
	evidence.AllowlistStatus = a.allowlistStore.CheckStatus(evidence.CanonicalHostname)
	// 計算 AUTH_OK
	source_trust.EnrichAuthOK(&evidence)
	return evidence
}

// GetProjectAllowlist 回傳專案白名單（含已過期）。
func (a *App) GetProjectAllowlist() (interface{}, error) {
	entries, err := a.allowlistStore.List()
	return frontendDTO(entries), err
}

// AddSourceToAllowlist 新增來源到白名單（呼叫端須先通過 Review Card）。
func (a *App) AddSourceToAllowlist(entryJSON string) error {
	var entry source_trust.AllowlistEntry
	if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
		return fmt.Errorf("invalid allowlist entry: %w", err)
	}
	return a.allowlistStore.Add(entry)
}

// RenewAllowlistEntry 續期白名單記錄。
func (a *App) RenewAllowlistEntry(entryID string) error {
	return a.allowlistStore.Renew(entryID)
}

// RemoveAllowlistEntry 移除白名單記錄。
func (a *App) RemoveAllowlistEntry(entryID string) error {
	return a.allowlistStore.Remove(entryID)
}

// GetHighImpactDomains 回傳有效的高影響領域清單。
func (a *App) GetHighImpactDomains() []string {
	return source_trust.EffectiveHighImpactDomains(nil, nil)
}

// --- v3.6.1 LLM Context Governance (§11) ---

// BuildLLMContext 組合 LLM context payload（雙層掃描）。
func (a *App) BuildLLMContext(blocksJSON string, sourcesJSON string, isHighImpact bool) (*llm_context.ContextPayload, error) {
	var blocks []llm_context.ContentBlock
	if err := json.Unmarshal([]byte(blocksJSON), &blocks); err != nil {
		return nil, fmt.Errorf("invalid blocks: %w", err)
	}
	var sources []llm_context.SourceToken
	if err := json.Unmarshal([]byte(sourcesJSON), &sources); err != nil {
		return nil, fmt.Errorf("invalid sources: %w", err)
	}
	return llm_context.BuildContextPayload(blocks, sources, isHighImpact)
}

// EscapeExternalTokens 逃脫外部內容中的系統 token。
func (a *App) EscapeExternalTokens(content string) string {
	return llm_context.EscapeExternalTokens(content)
}

// --- v3.6.1 Memory Pipeline (§18) ---

// GetMemoryPipelineState 回傳記憶管線即時狀態。
func (a *App) GetMemoryPipelineState() memory.PipelineState {
	return a.memoryPipeline.GetState()
}

// AppendTalkEntry 寫入對話記錄（含 redaction + 威脅偵測 + Review Card）。
func (a *App) AppendTalkEntry(role, text string) error {
	// 寫入（含自動 redaction）
	_, err := a.memoryPipeline.AppendTalkEntry(role, text)
	if err != nil {
		return err
	}

	// 威脅偵測
	result := memory.DetectThreats(text, "talk_entry")
	if result.Detected {
		hookRoot := storage.ProjectRoot(appDataRoot(), "default")
		for _, record := range result.Records {
			memory.AppendThreatRecordJSON(hookRoot, record)
			// 產生 Review Card 通知使用者（不自動改變風險）
			a.reviewService.AddLegacyCard(
				review.LevelBlocking,
				"threat_detection",
				record.ID,
				fmt.Sprintf("偵測到安全威脅：%s", record.Description),
				fmt.Sprintf("type=%s severity=%s source=%s", record.Type, record.Severity, record.Source),
			)
			a.eventBus.Emit(eventbus.EventReviewCardAdded, map[string]string{
				"source": "threat_detection", "id": record.ID,
			})
		}
	}

	// 檢查是否需要輪轉
	if a.memoryPipeline.CheckRotation() == memory.RotationRequired {
		a.eventBus.Emit("memory:rotation_required", nil)
	}

	return nil
}

// RotateTalkFull 執行 talk_full.md 輪轉歸檔。
func (a *App) RotateTalkFull() (string, error) {
	return a.memoryPipeline.Rotate()
}

// ValidateMemoryItem 驗證記憶項目格式與安全性。
func (a *App) ValidateMemoryItem(content string) memory.ValidationResult {
	return memory.ValidateMemoryItem(content)
}

// ListThreatRecords 列出所有威脅偵測記錄。
func (a *App) ListThreatRecords() (interface{}, error) {
	hookRoot := storage.ProjectRoot(appDataRoot(), "default")
	records, err := memory.ListThreatRecords(hookRoot)
	return frontendDTO(records), err
}

// --- v3.6.1 DAG Scheduler (§19) ---

// CreateDAGRun 建立新的 DAG 執行。
// 修正 Bug（TASK 17.1）：前端傳入 (runID, nodesJSON) 兩個參數。
// runID 用於日誌追蹤，實際 DAG Run ID 由後端產生。
func (a *App) CreateDAGRun(frontendRunID string, nodesJSON string) (*dag.DAGRun, error) {
	var nodes []dag.DAGNode
	if err := json.Unmarshal([]byte(nodesJSON), &nodes); err != nil {
		return nil, fmt.Errorf("invalid nodes: %w", err)
	}
	hookRoot := storage.ProjectRoot(appDataRoot(), "default")
	hashes := dag.ComputeCurrentHashes(hookRoot)
	run, err := a.dagScheduler.CreateRun(nodes, hashes.CombinedHash())
	if err != nil {
		return nil, err
	}
	// 持久化建立時的 guard hashes
	dag.SaveGuardHashes(hookRoot, run.ID, hashes)

	// §19.4: 更新 dag_runs_index.json
	dag.AppendRunIndex(hookRoot, dag.DAGRunSummary{
		RunID:     run.ID,
		Status:    run.Status,
		StartedAt: run.CreatedAt,
		NodeCount: len(run.Nodes),
	})

	return run, nil
}

// GetDAGRun 取得指定 DAGRun。
func (a *App) GetDAGRun(runID string) (*dag.DAGRun, error) {
	return a.dagScheduler.GetRun(runID)
}

// ListDAGRuns 列出所有 DAGRun（記憶體中的完整 run）。
func (a *App) ListDAGRuns() []*dag.DAGRun {
	return a.dagScheduler.ListRuns()
}

// §19.4 ListDAGRunsFromIndex 從 index 讀取歷史（輕量版，用於 UI 列表）。
func (a *App) ListDAGRunsFromIndex(limit int, statusFilter string) []dag.DAGRunSummary {
	hookRoot := storage.ProjectRoot(appDataRoot(), "default")
	return dag.ListDAGRuns(hookRoot, limit, statusFilter)
}

// UpdateDAGNodeStatus 更新 DAG 節點狀態。
func (a *App) UpdateDAGNodeStatus(runID, nodeID, status string) error {
	return a.dagScheduler.UpdateNodeStatus(runID, nodeID, dag.NodeStatus(status))
}

// ResumeDAGRun 恢復 DAGRun（含 Resume Guard 檢查）。
func (a *App) ResumeDAGRun(runID string) (*dag.GuardCheckResult, error) {
	hookRoot := storage.ProjectRoot(appDataRoot(), "default")
	run, err := a.dagScheduler.GetRun(runID)
	if err != nil {
		return nil, err
	}

	// Resume Guard 檢查
	current := dag.ComputeCurrentHashes(hookRoot)
	result := dag.CheckResumeGuard(run, current)
	if !result.Safe {
		// 阻擋所有非終端節點
		for _, node := range run.Nodes {
			if !dag.IsTerminal(node.Status) {
				a.dagScheduler.SetNodeBlocked(runID, node.ID, result.BlockReason)
			}
		}
		return &result, nil
	}
	return &result, nil
}

// --- v3.6.1 Persona Avatar (§10) ---

// GetCurrentAvatar 回傳指定 persona 的頭像設定。
func (a *App) GetCurrentAvatar(personaID string) persona_avatar.AvatarConfig {
	return a.avatarService.GetCurrentAvatar(personaID)
}

// SetAvatarProvider 設定頭像 provider（built_in_pixel / static_image / image_api）。
func (a *App) SetAvatarProvider(personaID string, provider string) error {
	return a.avatarService.SetProvider(personaID, persona_avatar.AvatarProvider(provider))
}

// SaveStaticAvatar 上傳並儲存靜態頭像（§10.7）。
// imageData 為 base64 解碼後的圖片資料，mimeType 須為 PNG/JPEG/WebP。
func (a *App) SaveStaticAvatar(personaID string, imageData []byte, mimeType string, cropJSON string) error {
	var crop *persona_avatar.CropRect
	if cropJSON != "" {
		crop = &persona_avatar.CropRect{}
		if err := json.Unmarshal([]byte(cropJSON), crop); err != nil {
			return fmt.Errorf("invalid crop rect: %w", err)
		}
	}
	return a.avatarService.SaveStaticAvatar(personaID, mimeType, imageData, crop, 256)
}

// DeleteStaticAvatar 刪除靜態頭像，自動 fallback 到 pixel（§10.8）。
func (a *App) DeleteStaticAvatar(personaID string) error {
	return a.avatarService.DeleteStaticAvatar(personaID)
}

// GetAvatarStateTrigger 根據任務狀態和風險等級回傳頭像狀態觸發器。
func (a *App) GetAvatarStateTrigger(taskStatus, riskLevel string) string {
	return string(persona_avatar.GetStateTrigger(taskStatus, riskLevel))
}

// RenderPixelAvatar 渲染指定狀態的 pixel art 頭像 PNG。
// 根據目前選擇的 pixel pack（wolf / uncle / secretary）渲染對應的頭像。
func (a *App) RenderPixelAvatar(state string, size int) ([]byte, error) {
	pack := a.GetPixelAvatarPack()
	return a.renderPixelAvatarPack(pack, state, size)
}

// RenderPixelAvatarPreview 渲染指定 pack 的預覽，不改變目前使用者選擇。
func (a *App) RenderPixelAvatarPreview(pack string, state string, size int) ([]byte, error) {
	if pack != "wolf" && pack != "uncle" && pack != "secretary" {
		return nil, fmt.Errorf("unknown pixel pack: %s", pack)
	}
	return a.renderPixelAvatarPack(pack, state, size)
}

func (a *App) renderPixelAvatarPack(pack string, state string, size int) ([]byte, error) {
	trigger := persona_avatar.AvatarStateTrigger(state)
	pack = persona_avatar.NormalizePixelPack("", pack)
	switch pack {
	case "uncle":
		return persona_avatar.RenderUnclePixelAvatar(trigger, size)
	case "secretary":
		return persona_avatar.RenderSecretaryPixelAvatar(trigger, size)
	default:
		return persona_avatar.RenderPixelAvatar(trigger, size)
	}
}

// GetPixelAvatarPack 取得目前選擇的 pixel avatar 套件（wolf / uncle / secretary）。
func (a *App) GetPixelAvatarPack() string {
	packFile := filepath.Join(appDataRoot(), "data", "cache", "pixel_pack.txt")
	data, err := os.ReadFile(packFile)
	if err != nil {
		return "wolf" // 預設
	}
	pack := strings.TrimSpace(string(data))
	switch pack {
	case "uncle", "secretary":
		return pack
	default:
		return "wolf"
	}
}

// SetPixelAvatarPack 切換 pixel avatar 套件（wolf / uncle / secretary）。
func (a *App) SetPixelAvatarPack(pack string) error {
	if pack != "wolf" && pack != "uncle" && pack != "secretary" {
		return fmt.Errorf("unknown pixel pack: %s", pack)
	}
	cacheDir := filepath.Join(appDataRoot(), "data", "cache")
	os.MkdirAll(cacheDir, 0755)
	packFile := filepath.Join(cacheDir, "pixel_pack.txt")
	return os.WriteFile(packFile, []byte(pack), 0o600)
}

// SetPersonaPixelAvatarPack assigns a built-in expression pack to one persona.
// This preserves the wolfdog brand while letting B/C keep uncle/secretary packs.
func (a *App) SetPersonaPixelAvatarPack(personaID string, pack string) error {
	return a.avatarService.SetPixelPack(personaID, pack)
}

// ListPixelAvatarPacks 列出所有可用的 pixel avatar 套件。
func (a *App) ListPixelAvatarPacks() []map[string]string {
	return []map[string]string{
		{"id": "wolf", "name": "狼犬獸人", "desc": "銀白毛色 · 琥珀金瞳 · 尖耳"},
		{"id": "uncle", "name": "帥氣大叔", "desc": "黑短髮 · 淺藍灰瞳 · 金耳環"},
		{"id": "secretary", "name": "可愛秘書", "desc": "巧克力棕髮 · 藍框眼鏡 · 紅蝴蝶結"},
	}
}

// PrepareAvatarGenerateRequest 組合圖像生成 API 請求（§10.4）。
func (a *App) PrepareAvatarGenerateRequest(personaID string, state string) (*persona_avatar.GenerateAvatarRequest, error) {
	return a.avatarService.PrepareGenerateRequest(personaID, persona_avatar.AvatarStateTrigger(state))
}

// ListAvatarPresets 列出所有可用的風格模板。
func (a *App) ListAvatarPresets() []persona_avatar.StylePreset {
	return persona_avatar.ListPresets()
}

// StoreAvatarCredential 加密儲存 API 金鑰（§10.5）。
// TASKS_1_6_3 Step 4：轉接統一 SecretStore，加 persona_avatar: namespace。
func (a *App) StoreAvatarCredential(ref, value string) error {
	return a.secretStore.Store("persona_avatar:"+ref, value)
}

// HasAvatarCredential 檢查指定 credential 是否存在。
func (a *App) HasAvatarCredential(ref string) bool {
	return a.secretStore.Has("persona_avatar:" + ref)
}

// DeleteAvatarCredential 刪除指定 credential。
func (a *App) DeleteAvatarCredential(ref string) error {
	return a.secretStore.Delete("persona_avatar:" + ref)
}

// GetCredentialMigrationStatus lets the UI ask before any legacy key is used.
func (a *App) GetCredentialMigrationStatus() credential.MigrationStatus {
	if a.credentialStore == nil {
		return credential.MigrationStatus{Error: "credential store unavailable"}
	}
	return a.credentialStore.MigrationStatus()
}

// ConfirmCredentialMigration upgrades v1 credentials only after user consent.
func (a *App) ConfirmCredentialMigration() credential.MigrationStatus {
	if a.credentialStore == nil {
		return credential.MigrationStatus{Error: "credential store unavailable"}
	}
	if err := a.credentialStore.ConfirmMigration(); err != nil {
		status := a.credentialStore.MigrationStatus()
		status.Error = err.Error()
		return status
	}
	return a.credentialStore.MigrationStatus()
}

// DisableCredentialMigration blocks credential use until the user reconfigures secrets.
func (a *App) DisableCredentialMigration() credential.MigrationStatus {
	if a.credentialStore == nil {
		return credential.MigrationStatus{Error: "credential store unavailable"}
	}
	if err := a.credentialStore.DisableMigration(); err != nil {
		status := a.credentialStore.MigrationStatus()
		status.Error = err.Error()
		return status
	}
	return a.credentialStore.MigrationStatus()
}

// GetPixelAvatarPath 取得 pixel avatar 快取檔案路徑。
func (a *App) GetPixelAvatarPath(state string, size int) string {
	pixelDir := filepath.Join(appDataRoot(), "data", "cache", "pixel_avatars")
	return persona_avatar.GetPixelAvatarPath(pixelDir, persona_avatar.AvatarStateTrigger(state), size)
}

// --- #2 Review Card ---

// ListOpenReviewCards returns all unresolved review cards.
func (a *App) ListOpenReviewCards() []review.Card {
	return a.reviewService.ListOpen()
}

// ResolveReviewCard marks a review card as resolved.
func (a *App) ResolveReviewCard(cardID string) error {
	// §4.6：Step 2 在後端重算 hash 並比對 Step 1 快照
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	err := a.reviewService.Resolve(cardID, projectRoot)
	if err == nil {
		a.eventBus.Emit(eventbus.EventReviewCardResolved, map[string]string{"card_id": cardID})
	}
	// hash 不一致時 emit invalidation event 通知前端重新產生 card
	if err == review.ErrReviewContextChanged {
		a.eventBus.Emit(eventbus.EventReviewCardInvalidated, map[string]string{"card_id": cardID, "reason": "review_context_changed"})
	}
	return err
}

// HasBlockingReviewCard returns true if any blocking card exists (gates DAG resume).
func (a *App) HasBlockingReviewCard() bool {
	return a.reviewService.HasBlocking()
}

// CreateDestructiveReviewCard 建立 user_owned_asset_destructive Review Card。
// TASKS_1_7：統一 project purge / memory clear / subagent delete 進 Consequence Menu。
func (a *App) CreateDestructiveReviewCard(operation, target, reason, affectedScope string) (review.Card, error) {
	card := a.reviewService.AddCard(review.CardParams{
		RiskClass:     risk.UserOwnedAssetDestructive,
		Operation:     operation,
		Target:        target,
		Reason:        reason,
		AcceptLabel:   "直接執行",
		RejectLabel:   "取消",
		AcceptEffect:  affectedScope,
		RejectEffect:  "不執行任何操作",
		RollbackAvail: false,
		BackupAvail:   true,
		LogLocation:   "data/projects/default/runtime/purge_manifest.json",
	})
	a.eventBus.Emit(eventbus.EventReviewCardAdded, map[string]string{"card_id": card.ID})
	return card, nil
}

// DestructiveReviewExecutionResult 回傳破壞性操作執行結果。
type DestructiveReviewExecutionResult struct {
	ReviewID    string                                 `json:"review_id"`
	Mode        string                                 `json:"mode"`
	Operation   string                                 `json:"operation"`
	RetryCount  int                                    `json:"retry_count"`
	CardPending bool                                   `json:"card_pending"`
	Message     string                                 `json:"message"`
	Backup      *project_lifecycle.PurgeBackupManifest `json:"backup,omitempty"`
	Purge       interface{}                            `json:"purge,omitempty"`
}

// ResolveAndExecuteDestructiveReviewCard 驗證卡片、執行操作，成功後才封存卡片。
func (a *App) ResolveAndExecuteDestructiveReviewCard(cardID, mode string) (*DestructiveReviewExecutionResult, error) {
	result := &DestructiveReviewExecutionResult{ReviewID: cardID, Mode: mode, CardPending: true}
	if mode == "cancel_keep_pending" {
		result.Message = "已取消，Review Card 保留待處理"
		return result, nil
	}

	card, err := a.reviewService.GetCard(cardID)
	if err != nil {
		return result, err
	}
	result.Operation = card.Operation
	if card.RiskClass != risk.UserOwnedAssetDestructive {
		return result, fmt.Errorf("review card 不是破壞性操作: %s", card.RiskClass)
	}
	if card.Operation != "purge_project" || card.Target != "default" {
		return result, fmt.Errorf("不支援的破壞性操作: %s/%s", card.Operation, card.Target)
	}

	if mode == "backup_then_execute" {
		backup, err := a.lifecycleService.BackupAutoSafeDirs("default")
		if err != nil {
			return a.recordDestructiveFailure(cardID, result, err)
		}
		result.Backup = backup
	} else if mode != "direct_execute" {
		return result, fmt.Errorf("未知的執行模式: %s", mode)
	}

	purgeResult, err := a.lifecycleService.Purge("default", project_lifecycle.TriggerUserDeletes)
	if err != nil {
		return a.recordDestructiveFailure(cardID, result, err)
	}
	result.Purge = frontendDTO(purgeResult)

	if err := a.reviewService.Resolve(cardID, storage.ProjectRoot(appDataRoot(), "default")); err != nil {
		return result, err
	}
	delete(a.destructiveFailures, cardID)
	result.CardPending = false
	result.Message = "破壞性操作已完成"
	a.eventBus.Emit(eventbus.EventReviewCardResolved, map[string]string{"card_id": cardID})
	return result, nil
}

func (a *App) recordDestructiveFailure(cardID string, result *DestructiveReviewExecutionResult, cause error) (*DestructiveReviewExecutionResult, error) {
	a.destructiveFailures[cardID]++
	result.RetryCount = a.destructiveFailures[cardID]
	result.CardPending = true
	if result.RetryCount == 1 {
		result.Message = "執行失敗，可重試或重新產生 Review Card"
	} else {
		result.Message = "再次失敗，保留原 Review Card"
	}
	result.Message = fmt.Sprintf("%s：%v", result.Message, cause)
	return result, nil
}

// GetReviewExecutionContext 取得 Review Card 當前 execution context（前端顯示/除錯��）。
// 真正的 hash snapshot 由 DualStepConfirmStep1 後端自行計算。
func (a *App) GetReviewExecutionContext(cardID string) (*review.ReviewExecutionContext, error) {
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	return a.reviewService.ComputeExecutionContext(cardID, projectRoot)
}

// DualStepConfirmStep1 記錄雙步驟確認的 Step 1（§4.6 v3.6.1）。
// 後端自行即時計算 hash snapshot，前端傳入的 hash 參數被忽略（相容性保留簽名）。
func (a *App) DualStepConfirmStep1(cardID, scopeHash, riskPolicyHash, toolRegistryHash string) error {
	// 忽略前端 hash 參數，後端自行計算確保安全
	projectRoot := storage.ProjectRoot(appDataRoot(), "default")
	ctx, err := a.reviewService.ComputeExecutionContext(cardID, projectRoot)
	if err != nil {
		return err
	}
	return a.reviewService.DualStepConfirmStep1(cardID, ctx.ScopeHash, ctx.RiskPolicyHash, ctx.ToolRegistryHash)
}

// InvalidateDualStep 使雙步驟��認失效（hash 變更時）。
func (a *App) InvalidateDualStep(cardID string) error {
	return a.reviewService.InvalidateDualStep(cardID)
}

// ListOpenLightweightCards 回傳所有未解決的 Lightweight Review Card。
func (a *App) ListOpenLightweightCards() []review.LightweightCard {
	return a.reviewService.ListOpenLightweight()
}

// ResolveLightweightCard 解決 Lightweight Review Card。
func (a *App) ResolveLightweightCard(reviewID string) error {
	return a.reviewService.ResolveLightweight(reviewID)
}

// --- #4 Memory Health / Config Public (read-only) ---

// GetMemoryHealth returns current memory/runtime metrics.
func (a *App) GetMemoryHealth() health.MemoryHealth {
	return a.healthService.GetMemoryHealth()
}

// GetConfigPublic returns non-sensitive app configuration.
func (a *App) GetConfigPublic() health.ConfigPublic {
	return a.healthService.GetConfigPublic()
}

// --- #5 Degraded Mode ---

// GetDegradedState returns the current degraded mode state.
func (a *App) GetDegradedState() interface{} {
	return frontendDTO(a.degradedService.GetState())
}

// EnterDegradedMode activates degraded mode with the given reason.
// Invalidates all open review cards and emits event to frontend.
func (a *App) EnterDegradedMode(reason string) interface{} {
	state := a.degradedService.Enter(degraded.Reason(reason))
	// 作廢 all open review cards
	a.reviewService.InvalidateAll()
	a.eventBus.Emit(eventbus.EventDegradedModeEntered, state)
	return frontendDTO(state)
}

// ExitDegradedMode deactivates degraded mode and emits event.
func (a *App) ExitDegradedMode() interface{} {
	state := a.degradedService.Exit()
	a.eventBus.Emit(eventbus.EventDegradedModeExited, state)
	return frontendDTO(state)
}

// IsDegradedBlocked checks whether a specific operation is blocked by degraded mode.
func (a *App) IsDegradedBlocked(opCategory string) bool {
	return a.degradedService.IsBlocked(opCategory)
}

// --- #6 Onboarding / Read-only Mode ---

// GetOnboardingState returns the current onboarding state.
func (a *App) GetOnboardingState() onboarding.State {
	return a.onboardService.GetState()
}

// IsReadOnlyMode returns whether the app is in read-only mode.
func (a *App) IsReadOnlyMode() bool {
	return a.onboardService.IsReadOnly()
}

// CompleteOnboardingStep marks an onboarding step as complete.
func (a *App) CompleteOnboardingStep(stepID string) onboarding.State {
	return a.onboardService.CompleteStep(stepID)
}

// GoBackOnboarding moves onboarding to the previous step.
func (a *App) GoBackOnboarding() onboarding.State {
	return a.onboardService.GoBack()
}

// FinishOnboarding completes the onboarding flow, exits read-only mode.
func (a *App) FinishOnboarding() error {
	return a.onboardService.MarkComplete()
}

// ---------------------------------------------------------------------------
// v3.6.1 Project Lifecycle（§7.5）Wails Bindings
// ---------------------------------------------------------------------------

// PurgeProject 執行專案清除（分類自動 + 禁止跳過 + 邊界待審）。
func (a *App) PurgeProject(projectID, trigger string) (interface{}, error) {
	result, err := a.lifecycleService.Purge(projectID, project_lifecycle.PurgeTrigger(trigger))
	return frontendDTO(result), err
}

// PurgeBoundaryDir 使用者確認後清除邊界類目錄。
func (a *App) PurgeBoundaryDir(relativePath string) error {
	return a.lifecycleService.PurgeBoundaryDir(relativePath)
}

// ListPurgeManifests 列出所有清除 manifest。
func (a *App) ListPurgeManifests() (interface{}, error) {
	hookRoot := storage.ProjectRoot(appDataRoot(), "default")
	manifests, err := project_lifecycle.ListManifests(hookRoot)
	return frontendDTO(manifests), err
}

// ---------------------------------------------------------------------------
// v3.6.1 Stop Recovery（§21）Wails Bindings
// ---------------------------------------------------------------------------

// CreateStopRecoveryCard 建立恢復卡片（通常由系統自動觸發）。
func (a *App) CreateStopRecoveryCard(reason, signal string) *stop_recovery.StopRecoveryCard {
	card := a.stopRecoveryService.CreateCard(stop_recovery.StopReason(reason), signal)
	a.eventBus.Emit("stop_recovery:card_created", map[string]string{
		"card_id": card.ID, "reason": string(card.StopReason),
	})
	return card
}

// ResolveStopRecoveryCard 解決恢復卡片（使用者選擇動作）。
func (a *App) ResolveStopRecoveryCard(cardID, action string) error {
	err := a.stopRecoveryService.ResolveCard(cardID, stop_recovery.RecoveryAction(action))
	if err == nil {
		a.eventBus.Emit("stop_recovery:card_resolved", map[string]string{
			"card_id": cardID, "action": action,
		})
	}
	return err
}

// ListOpenStopRecoveryCards 列出所有未解決的恢復卡片。
func (a *App) ListOpenStopRecoveryCards() []*stop_recovery.StopRecoveryCard {
	return a.stopRecoveryService.ListOpen()
}

// HasOpenStopRecoveryCard 是否有未解決的恢復卡片（阻擋 DAG 繼續）。
func (a *App) HasOpenStopRecoveryCard() bool {
	return a.stopRecoveryService.HasOpen()
}

// GetStopRecoveryCard 取得指定恢復卡片。
func (a *App) GetStopRecoveryCard(cardID string) (*stop_recovery.StopRecoveryCard, error) {
	return a.stopRecoveryService.GetCard(cardID)
}

// ---------------------------------------------------------------------------
// v3.6.1 Visual Learning（§14）Wails Bindings — 完整接入
// ---------------------------------------------------------------------------

// --- Learning Mode 錄製控制 ---

// StartLearningMode 啟動視覺學習錄製模式（使用者必須明確啟動，禁止背景錄製）。
func (a *App) StartLearningMode(activeWindowHash string) (interface{}, error) {
	run, err := a.learningService.StartDemonstration(activeWindowHash)
	if err != nil {
		return nil, err
	}
	a.eventBus.Emit("visual_learning:recording_started", map[string]string{"run_id": run.ID})
	return frontendDTO(run), nil
}

// StopLearningMode 停止錄製模式。
func (a *App) StopLearningMode() (interface{}, error) {
	run, err := a.learningService.StopDemonstration()
	if err != nil {
		return nil, err
	}
	a.eventBus.Emit("visual_learning:recording_stopped", map[string]string{"run_id": run.ID})
	return frontendDTO(run), nil
}

// IsLearningModeActive 是否正在錄製。
func (a *App) IsLearningModeActive() bool {
	return a.learningService.IsRecording()
}

// GetActiveLearningRun 回傳目前錄製中的 run（無錄製回傳 nil）。
func (a *App) GetActiveLearningRun() interface{} {
	return frontendDTO(a.learningService.ActiveRun())
}

// --- OpenCV + YOLO 推論狀態 ---

// GetOpenCVStatus 回傳 OpenCV pipeline 可用狀態。
func (a *App) GetOpenCVStatus() bool {
	return a.opencvPipeline.IsAvailable()
}

// GetYOLOStatus 回傳 YOLO detector 可用狀態。
func (a *App) GetYOLOStatus() bool {
	return a.yoloDetector.IsAvailable()
}

// ProposeRegions 執行 OpenCV 區域提案（degraded 模式回傳空列表）。
func (a *App) ProposeRegions(imageData []byte, width, height int) interface{} {
	return frontendDTO(a.opencvPipeline.Propose(imageData, width, height))
}

// DetectRegions 執行 YOLO 區域偵測（degraded 模式回傳空列表）。
func (a *App) DetectRegions(imageData []byte, width, height int) visual_learning.DetectorResult {
	return a.yoloDetector.Detect(imageData, width, height)
}

// --- Canonical Label 標籤對映 ---

// MapCanonicalLabel 將 LLM 描述對映到正規標籤（無法對映時建立 pending candidate）。
func (a *App) MapCanonicalLabel(llmDescription, sourceRegionID string, confidence float64) (visual_learning.CanonicalLabel, bool) {
	return a.canonicalLabelSvc.MapLabel(llmDescription, sourceRegionID, confidence)
}

// GetCanonicalLabelSchema 回傳目前正規標籤 schema。
func (a *App) GetCanonicalLabelSchema() []visual_learning.CanonicalLabel {
	return a.canonicalLabelSvc.GetSchema()
}

// --- Element Dictionary 元素辭典 ---

// AddElementDictionaryEntry 新增元素辭典條目（status=pending，需 Review 通過）。
func (a *App) AddElementDictionaryEntry(fingerprintID string, labelJSON string, confidence float64) (interface{}, error) {
	var label visual_learning.CanonicalLabel
	if err := json.Unmarshal([]byte(labelJSON), &label); err != nil {
		return nil, fmt.Errorf("invalid label: %w", err)
	}
	entry, err := a.elementDictionary.Add(fingerprintID, label, confidence)
	return frontendDTO(entry), err
}

// ApproveElementDictionaryEntry 核准元素辭典條目（Review 通過後可用於自動化）。
func (a *App) ApproveElementDictionaryEntry(id string) error {
	return a.elementDictionary.Approve(id)
}

// ListElementDictionary 列出元素辭典條目（可按 status 過濾）。
func (a *App) ListElementDictionary(status string) interface{} {
	return frontendDTO(a.elementDictionary.List(visual_learning.DictEntryStatus(status)))
}

// --- Action Dictionary 動作辭典 ---

// AddActionCandidate 新增動作候選（pending_action_candidate，不可執行）。
func (a *App) AddActionCandidate(stepsJSON string, riskLevel string, learningRunID string) (interface{}, error) {
	var steps []visual_learning.ActionStep
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		return nil, fmt.Errorf("invalid steps: %w", err)
	}
	entry, err := a.actionDictionary.AddPendingCandidate(steps, visual_learning.ActionRiskLevel(riskLevel), learningRunID)
	return frontendDTO(entry), err
}

// ApproveActionCandidate 核准動作候選（Review 通過後可執行）。
func (a *App) ApproveActionCandidate(id string) error {
	return a.actionDictionary.Approve(id)
}

// ListFormalActions 列出已核准的動作辭典條目。
func (a *App) ListFormalActions() interface{} {
	return frontendDTO(a.actionDictionary.ListFormal())
}

// ListPendingActions 列出待審的動作候選。
func (a *App) ListPendingActions() interface{} {
	return frontendDTO(a.actionDictionary.ListPending())
}

// --- Pending Candidate 生命週期管理 ---

// ListPendingCandidates 列出非已歸檔的 pending candidates。
func (a *App) ListPendingCandidates() interface{} {
	return frontendDTO(a.pendingCandidateMgr.List())
}

// GetPendingCandidateCount 回傳 active pending candidate 數量。
func (a *App) GetPendingCandidateCount() int {
	return a.pendingCandidateMgr.ActiveCount()
}

// RefreshPendingCandidateAges 更新所有 pending candidate 的年齡狀態。
func (a *App) RefreshPendingCandidateAges() {
	a.pendingCandidateMgr.RefreshAges()
}

// --- Safe Export 安全匯出 ---

// ExportVisualLearning 執行安全匯出（僅允許項目通過，禁止項目被阻擋）。
func (a *App) ExportVisualLearning(sectionsJSON string) (interface{}, error) {
	var sections []string
	if err := json.Unmarshal([]byte(sectionsJSON), &sections); err != nil {
		return nil, fmt.Errorf("invalid sections: %w", err)
	}
	manifest, err := a.vlSafeExporter.Export(sections)
	return frontendDTO(manifest), err
}

// ──────────────────────────────────────────────────────────────────────────
// Visual Learning Review — 轉接正式 reviewService（TASKS_1_6_3）
//
// 原先 VL 有自己的 ReviewInbox（adapter/visual_learning/review_card.go），
// 使用 3 級 ReviewLevel + 自有 inbox/archive JSON。
// 現已統一到 domain/review.Service，使用 7 級 RiskClass + 共用持久化。
//
// Wails binding 簽名不變，前端 JS 呼叫不需改動。
// DTO 透過 frontendDTO 轉換，Card.LegacyLevel 提供向下相容。
// ──────────────────────────────────────────────────────────────────────────

// mapVLLevelToRiskClass 將舊版 VL 三級 ReviewLevel 映射到正式七級 RiskClass。
// 與 domain/review.legacyLevelToRisk 保持一致。
func mapVLLevelToRiskClass(level string) risk.RiskClass {
	switch level {
	case "blocking_review":
		return risk.HighNonDestructive
	case "pending_review":
		return risk.Medium
	default: // "background_review"
		return risk.Low
	}
}

// AddVLReviewCard 新增視覺學習 Review Card（轉接 reviewService）。
func (a *App) AddVLReviewCard(level, sourceType, sourceID, plainReason, engineerReason string) (interface{}, error) {
	riskClass := mapVLLevelToRiskClass(level)
	card := a.reviewService.AddCard(review.CardParams{
		RiskClass:      riskClass,
		Operation:      "review_visual_learning_candidate",
		Target:         sourceID,
		Reason:         plainReason,
		AcceptLabel:    "批准",
		RejectLabel:    "拒絕",
		AcceptEffect:   "候選項進入正式知識庫",
		RejectEffect:   "候選項被丟棄",
		SourceType:     "visual_learning",
		SourceID:       sourceID,
		EngineerReason: engineerReason,
	})
	a.eventBus.Emit("visual_learning:review_card_added", map[string]string{
		"card_id": card.ID, "level": level,
	})
	return frontendDTO(card), nil
}

// ResolveVLReviewCard 解決視覺學習 Review Card（轉接 reviewService）。
func (a *App) ResolveVLReviewCard(cardID string) error {
	err := a.reviewService.Resolve(cardID)
	if err == nil {
		a.eventBus.Emit("visual_learning:review_card_resolved", map[string]string{"card_id": cardID})
	}
	return err
}

// ListOpenVLReviewCards 列出所有開放的視覺學習 Review Card。
// 從 reviewService.ListOpen() 過濾 SourceType=="visual_learning"。
func (a *App) ListOpenVLReviewCards() interface{} {
	all := a.reviewService.ListOpen()
	var vlCards []review.Card
	for _, c := range all {
		if c.SourceType == "visual_learning" {
			vlCards = append(vlCards, c)
		}
	}
	return frontendDTO(vlCards)
}

// HasBlockingVLReview 是否有阻擋型視覺學習 Review Card。
// 從 reviewService.ListOpen() 過濾 SourceType + blocking 等級。
func (a *App) HasBlockingVLReview() bool {
	all := a.reviewService.ListOpen()
	for _, c := range all {
		if c.SourceType == "visual_learning" && risk.IsHigherOrEqual(c.RiskClass, risk.HighNonDestructive) {
			return true
		}
	}
	return false
}

// --- Dry Run 定位驗證 ---

// ComputeDryRunConfidence 計算 dry-run 四階段信心分數。
func (a *App) ComputeDryRunConfidence(base, devicePenalty, runtimeEvidence float64) float64 {
	return visual_learning.ComputeConfidence(base, devicePenalty, runtimeEvidence)
}

// ──────────────────────────────────────────────────────────────
// §12A Remote Bridge Communication（v3.6.3）
// ──────────────────────────────────────────────────────────────
//
// 以下方法是 remote_bridge 套件對前端的 Wails binding 代理層。
// 每個方法都：
//   1. 委派給 a.remoteBridge（remote_bridge.Service）執行業務邏輯
//   2. 成功時透過 a.eventBus.Emit 發送事件，前端 EventsOn 監聽後
//      自動呼叫 refreshRemoteBridgeChannels() 刷新 UI
//
// 前端 UX 流程：
//   引用連結貼 URL
//     → DetectRemoteBridgeChannel（辨識平台）
//     → TestRemoteBridgeConnection（連線測試）
//     → RegisterRemoteBridgeChannel（註冊通道，icon 出現）
//     → ActivateRemoteBridgeChannel（點擊 icon 啟用，notification_only）
//     → SwitchRemoteBridgeMode（長按/右鍵切換模式）
//
// 事件清單：
//   remote_bridge:channel_registered
//   remote_bridge:channel_activated
//   remote_bridge:channel_deactivated
//   remote_bridge:mode_switched
//   remote_bridge:channel_removed
// ──────────────────────────────────────────────────────────────

// DetectRemoteBridgeChannel 從 URL 偵測通訊軟體類型。
func (a *App) DetectRemoteBridgeChannel(rawURL string) remote_bridge.DetectResult {
	return a.remoteBridge.DetectChannelFromURL(rawURL)
}

// TestRemoteBridgeConnection 測試指定 URL 的連線。
func (a *App) TestRemoteBridgeConnection(rawURL string) interface{} {
	return frontendDTO(a.remoteBridge.TestChannelConnection(rawURL))
}

func (a *App) syncDiscordGateways() {
	if a.ctx == nil || a.remoteBridge == nil || a.eventBus == nil {
		return
	}
	if a.remoteBridgeDiscord == nil {
		a.remoteBridgeDiscord = remote_bridge.NewDiscordGatewayManager(a.remoteBridge, a.eventBus)
	}
	a.remoteBridgeDiscord.Sync(a.ctx)
}

// RegisterRemoteBridgeChannel 註冊通道（URL 已通過連線測試）。
func (a *App) RegisterRemoteBridgeChannel(rawURL string) (interface{}, error) {
	binding, err := a.remoteBridge.RegisterChannel(rawURL)
	if err == nil {
		a.eventBus.Emit("remote_bridge:channel_registered", map[string]string{
			"channel_id": binding.ID,
			"channel":    string(binding.Channel),
		})
		a.syncDiscordGateways()
	}
	return frontendDTO(binding), err
}

// RegisterRemoteBridgeChannelWithMode 以指定模式註冊通道（§12A.2 使用者分流）。
func (a *App) RegisterRemoteBridgeChannelWithMode(setupMode, presetID, fieldsJSON, customConfigJSON string) (interface{}, error) {
	// 解析 fields
	fields := make(map[string]string)
	if fieldsJSON != "" {
		if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
			return nil, fmt.Errorf("invalid fields JSON: %w", err)
		}
	}

	// 解析 custom config（developer mode）
	var customConfig *remote_bridge.WebhookRequest
	if customConfigJSON != "" {
		customConfig = &remote_bridge.WebhookRequest{}
		if err := json.Unmarshal([]byte(customConfigJSON), customConfig); err != nil {
			return nil, fmt.Errorf("invalid custom config JSON: %w", err)
		}
	}

	binding, err := a.remoteBridge.RegisterChannelWithMode(setupMode, presetID, fields, customConfig)
	if err == nil {
		a.eventBus.Emit("remote_bridge:channel_registered", map[string]string{
			"channel_id": binding.ID,
			"channel":    string(binding.Channel),
			"setup_mode": setupMode,
		})
		a.syncDiscordGateways()
	}
	return frontendDTO(binding), err
}

// ListRemoteBridgePresets 列出所有可用平台 preset。
func (a *App) ListRemoteBridgePresets() interface{} {
	ids := remote_bridge.ListPresets()
	result := make([]map[string]interface{}, 0, len(ids))
	for _, id := range ids {
		preset, _ := remote_bridge.GetPreset(id)
		result = append(result, map[string]interface{}{
			"platform_id":        preset.PlatformID,
			"max_length":         preset.MaxLength,
			"required_fields":    preset.RequiredFields,
			"auto_detect_fields": preset.AutoDetectFields,
		})
	}
	return result
}

// ActivateRemoteBridgeChannel 啟用通道（自動停用其他通道）。
func (a *App) ActivateRemoteBridgeChannel(channelID string) (interface{}, error) {
	binding, err := a.remoteBridge.ActivateChannel(channelID)
	if err == nil {
		a.eventBus.Emit("remote_bridge:channel_activated", map[string]string{
			"channel_id": binding.ID,
			"channel":    string(binding.Channel),
			"mode":       string(binding.Mode),
		})
		a.syncDiscordGateways()
	}
	return frontendDTO(binding), err
}

// SetRemoteBridgePrimaryChannel 設定唯一主頻道；主頻道權限/優先權高於一般啟用通道。
func (a *App) SetRemoteBridgePrimaryChannel(channelID string) (interface{}, error) {
	binding, err := a.remoteBridge.SetPrimaryChannel(channelID)
	if err == nil {
		a.eventBus.Emit("remote_bridge:primary_changed", map[string]string{
			"channel_id": binding.ID,
			"channel":    string(binding.Channel),
			"mode":       string(binding.Mode),
		})
		a.syncDiscordGateways()
	}
	return frontendDTO(binding), err
}

// DeactivateRemoteBridgeChannel 停用通道。
func (a *App) DeactivateRemoteBridgeChannel(channelID string) error {
	err := a.remoteBridge.DeactivateChannel(channelID)
	if err == nil {
		a.eventBus.Emit("remote_bridge:channel_deactivated", map[string]string{
			"channel_id": channelID,
		})
		a.syncDiscordGateways()
	}
	return err
}

// SwitchRemoteBridgeMode 切換通道權限模式。
func (a *App) SwitchRemoteBridgeMode(channelID string, mode string) (interface{}, error) {
	binding, err := a.remoteBridge.SwitchMode(channelID, remote_bridge.ChannelMode(mode))
	if err == nil {
		a.eventBus.Emit("remote_bridge:mode_switched", map[string]string{
			"channel_id": binding.ID,
			"channel":    string(binding.Channel),
			"mode":       string(binding.Mode),
		})
	}
	return frontendDTO(binding), err
}

// ListRemoteBridgeChannels 列出所有已註冊通道。
func (a *App) ListRemoteBridgeChannels() interface{} {
	return frontendDTO(a.remoteBridge.ListChannels())
}

// RenameRemoteBridgeChannel 更新通道顯示名稱。
func (a *App) RenameRemoteBridgeChannel(channelID, displayName string) (interface{}, error) {
	binding, err := a.remoteBridge.RenameChannel(channelID, displayName)
	if err == nil {
		a.eventBus.Emit("remote_bridge:channel_renamed", map[string]string{
			"channel_id": binding.ID,
			"name":       binding.DisplayName,
		})
	}
	return frontendDTO(binding), err
}

// RemoveRemoteBridgeChannel 移除通道。
func (a *App) RemoveRemoteBridgeChannel(channelID string) error {
	err := a.remoteBridge.RemoveChannel(channelID)
	if err == nil {
		a.eventBus.Emit("remote_bridge:channel_removed", map[string]string{
			"channel_id": channelID,
		})
		a.syncDiscordGateways()
	}
	return err
}

// GetRemoteBridgeAudit 取得最近稽核記錄。
func (a *App) GetRemoteBridgeAudit(n int) interface{} {
	return frontendDTO(a.remoteBridge.GetRecentAudit(n))
}

// GetRemoteBridgeInboundEndpoint 回傳本機接收 URL。若要給 LINE 使用，需再由
// 使用者選擇 tunnel/relay 對外公開，程式碼本身不綁定特定雲服務。
func (a *App) GetRemoteBridgeInboundEndpoint(channelID string) interface{} {
	if a.remoteBridgeInbound == nil {
		a.remoteBridgeInbound = remote_bridge.NewInboundServer(a.remoteBridge, a.eventBus)
		_ = a.remoteBridgeInbound.Start()
	}
	return map[string]interface{}{
		"channel_id": channelID,
		"local_url":  a.remoteBridgeInbound.URLForChannel(channelID),
		"addr":       a.remoteBridgeInbound.Addr(),
		"note":       "LINE Webhook 需要可被 LINE 伺服器連到的 HTTPS URL；請用 Cloudflare Tunnel、ngrok 或自己的 relay 指到這個本機 URL。",
	}
}

// SaveRemoteBridgeInboundSecret 儲存 LINE Channel secret，供 webhook 驗簽。
func (a *App) SaveRemoteBridgeInboundSecret(channelID, channelSecret string) error {
	err := a.remoteBridge.StoreInboundSecret(channelID, channelSecret)
	if err == nil {
		a.eventBus.Emit("remote_bridge:inbound_secret_saved", map[string]string{
			"channel_id": channelID,
		})
	}
	return err
}

// ListRemoteBridgeInboundAdapters 讓前端/測試知道哪些平台已有半雙向接口。
func (a *App) ListRemoteBridgeInboundAdapters() interface{} {
	return frontendDTO(remote_bridge.ListInboundAdapterStatuses())
}

// §12A.2 Telegram Onboarding — 自動偵測 chat_id
// StartTelegramOnboarding 驗證 bot token 有效性。
func (a *App) StartTelegramOnboarding(botToken string) interface{} {
	session, _ := remote_bridge.StartTelegramOnboarding(botToken)
	return frontendDTO(session)
}

// PollTelegramChatID 輪詢 getUpdates 尋找含 /start 的 chat。
func (a *App) PollTelegramChatID(botToken string) interface{} {
	candidates, _ := remote_bridge.PollTelegramChatID(botToken)
	return frontendDTO(candidates)
}

// ConfirmTelegramChatID 確認 chat_id 並存入 device-local secret。
func (a *App) ConfirmTelegramChatID(channelID, chatID string) error {
	return remote_bridge.ConfirmTelegramChatID(a.secretStore, channelID, chatID)
}

// §12A.5B 非同步分發 — 不阻塞 UI
// DispatchRemoteBridgeAsync 非同步送出訊息，立即回傳 dispatch_id。
func (a *App) DispatchRemoteBridgeAsync(channelID, content string) string {
	dispatcher := remote_bridge.NewAsyncDispatcher(a.remoteBridge, a.eventBus)
	return dispatcher.DispatchAsync(remote_bridge.AsyncDispatchRequest{
		ChannelID: channelID,
		Content:   content,
	})
}

// RetryRemoteBridgeSegment 手動重試指定段。
func (a *App) RetryRemoteBridgeSegment(dispatchID string, partIndex int, content string) error {
	dispatcher := remote_bridge.NewAsyncDispatcher(a.remoteBridge, a.eventBus)
	return dispatcher.RetrySegment(dispatchID, partIndex, content)
}

// ──────────────────────────────────────────────────────────────
// §9A W3A Media Provenance Wails Bindings（8 個）
// ──────────────────────────────────────────────────────────────
// 所有方法委派給 a.w3aMedia（w3a_media.Service）執行。
// 成功操作透過 eventBus 通知前端更新 UI。
// ──────────────────────────────────────────────────────────────

// VerifyMediaFile 對媒體檔案執行完整 W3A 驗證。
func (a *App) VerifyMediaFile(path string) (interface{}, error) {
	info, err := a.w3aMedia.VerifyMediaFile(path)
	if err == nil {
		a.eventBus.Emit("w3a:verified", map[string]interface{}{"path": path, "status": string(info.Status)})
	}
	return frontendDTO(info), err
}

// GetMediaW3AInfo 取得媒體的 W3A 資訊。
func (a *App) GetMediaW3AInfo(path string) (interface{}, error) {
	info, err := a.w3aMedia.GetMediaW3AInfo(path)
	return frontendDTO(info), err
}

// DetectModelPollution 對媒體執行模型污染偵測。
func (a *App) DetectModelPollution(path string) (*w3a_media.PollutionReport, error) {
	report, err := a.w3aMedia.DetectMediaPollution(path)
	if err == nil && report.IsPollutionRisk {
		a.eventBus.Emit("w3a:pollution_detected", map[string]interface{}{"path": path, "score": report.WeightedTotal})
	}
	return report, err
}

// ExportMediaWithSidecar 匯出媒體檔案，同時複製 sidecar。
func (a *App) ExportMediaWithSidecar(srcPath, destPath string) error {
	err := a.w3aMedia.ExportMedia(srcPath, destPath)
	if err == nil {
		a.eventBus.Emit("w3a:exported", map[string]interface{}{"src": srcPath, "dest": destPath})
	}
	return err
}

// ImportMediaVerify 匯入媒體並驗證，回傳含 UX 提示的結果。
func (a *App) ImportMediaVerify(path string) (interface{}, error) {
	result, err := a.w3aMedia.ImportAndVerify(path)
	if err == nil {
		a.eventBus.Emit("w3a:imported", map[string]interface{}{"path": path, "has_sidecar": result.HasSidecar, "status": string(result.Info.Status)})
	}
	return frontendDTO(result), err
}

// GetW3ATransferGuidance 取得原檔傳輸引導建議。
func (a *App) GetW3ATransferGuidance() w3a_media.TransferGuidance {
	return a.w3aMedia.GetGuidance()
}

// ListW3ATrustedDevelopers 列出所有信任的開發者。
func (a *App) ListW3ATrustedDevelopers() interface{} {
	return frontendDTO(a.w3aMedia.ListTrustedDevelopers())
}

// AddW3ATrustedDeveloper 新增信任的開發者。
func (a *App) AddW3ATrustedDeveloper(appID, pubKey, displayName string) error {
	err := a.w3aMedia.AddTrustedDeveloper(appID, pubKey, displayName)
	if err == nil {
		a.eventBus.Emit("w3a:trust_updated", map[string]interface{}{"action": "add", "app_id": appID})
	}
	return err
}

// ──────────────────────────────────────────────────────────────
// v3.6.4 Readiness Gate UI Interaction Layer Wails Bindings
// ──────────────────────────────────────────────────────────────
//
// 本區塊提供 Readiness Gate 前端互動層所需的後端資料結構與 Wails 綁定。
// 前端透過這些綁定取得：
//   - 浮動意圖候選（Floating Candidate Actions）
//   - 缺欄位膠囊（Missing Slot Capsule）
//   - Readiness Trace（runtime 追蹤記錄）
//   - 風險等級判定（三層確認機制的觸發依據）
//
// 設計原則：
//   「UI 可以像高級咖啡店一樣簡潔，後端像機場海關一樣硬。」
//   前端 Clean UI 可隱藏低信心標籤，但 readiness_trace 不能省。

// ReadinessTrace 是每次 Readiness Gate 評估的最小追蹤結構。
// 前端不直接顯示此結構，但後端必須記錄，用於稽核與除錯。
type ReadinessTrace struct {
	TaskID             string   `json:"task_id"`
	Decision           string   `json:"decision"` // ACT | ASK | RETRIEVE_MORE | REVIEW | STOP
	MissingSlots       []string `json:"missing_slots"`
	AssumptionUsed     bool     `json:"assumption_used"`
	LowConfidence      bool     `json:"low_confidence_output"`
	MemoryWrite        string   `json:"memory_write"`  // allowed | blocked | restricted
	ContextReuse       string   `json:"context_reuse"` // allowed | restricted
	RiskLevel          string   `json:"risk_level"`    // none | low | medium | high | critical
	FinalActionAllowed bool     `json:"final_action_allowed"`
}

// FloatingCandidate 代表一個浮動意圖候選按鈕的資料。
// 當 Readiness Gate 偵測到模糊使用者意圖時，最多回傳三個候選。
type FloatingCandidate struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Draft string `json:"draft"` // 點擊後填入 structured_task.draft
}

func floatingCandidatesFromQuestionTarget(target string) (string, []FloatingCandidate) {
	parts := strings.Split(target, "#")
	question := stripInternalControlPrefix(parts[0])
	var candidates []FloatingCandidate
	for _, raw := range parts[1:] {
		if len(candidates) >= 3 {
			break
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		label, draft, hasDraft := strings.Cut(raw, "=")
		label = stripInternalControlPrefix(label)
		draft = stripCandidateDraft(draft)
		if label == "" {
			continue
		}
		if !hasDraft || draft == "" {
			draft = label
		}
		candidates = append(candidates, FloatingCandidate{
			ID:    fmt.Sprintf("intent-question-%d", len(candidates)+1),
			Label: label,
			Draft: draft,
		})
	}
	return question, candidates
}

func floatingCandidatesFromOptionTarget(target string) (string, []FloatingCandidate) {
	trimmed := stripInternalControlPrefix(target)
	parts := strings.Split(trimmed, "ㄤ")
	question := "請選擇："
	start := 0
	if len(parts) > 0 && looksLikeOptionQuestion(parts[0]) {
		question = stripInternalControlPrefix(parts[0])
		start = 1
	}
	var candidates []FloatingCandidate
	for _, raw := range parts[start:] {
		if len(candidates) >= 3 {
			break
		}
		label := stripInternalControlPrefix(raw)
		if label == "" {
			continue
		}
		candidates = append(candidates, FloatingCandidate{
			ID:    fmt.Sprintf("intent-option-%d", len(candidates)+1),
			Label: label,
			Draft: label,
		})
	}
	return question, candidates
}

func looksLikeOptionQuestion(text string) bool {
	text = stripInternalControlPrefix(text)
	if text == "" {
		return false
	}
	for _, marker := range []string{"？", "?", "請", "想", "哪", "選", "選擇", "提供", "輸入"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func stripCandidateDraft(draft string) string {
	draft = strings.TrimSpace(draft)
	if after, ok := strings.CutPrefix(draft, "input:"); ok {
		return "input:" + stripInternalControlPrefix(after)
	}
	return stripInternalControlPrefix(draft)
}

func stripInternalControlPrefix(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "ㄌㄤㄤ")
	runes := []rune(text)
	if len(runes) >= controlseal.SealLength && looksLikeBopomofoControlPrefix(runes[:controlseal.SealLength]) {
		return strings.TrimSpace(string(runes[controlseal.SealLength:]))
	}
	return text
}

func looksLikeBopomofoControlPrefix(runes []rune) bool {
	if len(runes) != controlseal.SealLength {
		return false
	}
	for _, r := range runes {
		if r < 'ㄅ' || r > 'ㄩ' {
			return false
		}
	}
	return true
}

func questionPayload(target, next string) string {
	target = strings.TrimSpace(target)
	next = strings.TrimSpace(next)
	if next != "" && next != actionchain.StandbyNext && strings.Contains(next, "#") {
		// Some models put the question-card payload in 下一步; prefer it when it has candidates.
		return next
	}
	if target != "" {
		return target
	}
	return next
}

func setQuestionFloatingCandidates(target, traceID string) string {
	question, candidates := floatingCandidatesFromQuestionTarget(target)
	readinessMu.Lock()
	currentGateState.FloatingCandidates = candidates
	currentGateState.MissingSlots = nil
	currentGateState.RiskTier = "none"
	currentGateState.ClarificationCount++
	if currentGateState.MaxClarifications == 0 {
		currentGateState.MaxClarifications = 2
	}
	readinessMu.Unlock()
	debugtrace.Record("readiness.intent_question", traceID, map[string]interface{}{
		"question":         question,
		"candidate_count":  len(candidates),
		"candidate_labels": candidateLabels(candidates),
	})
	return question
}

func setOptionFloatingCandidates(target, traceID string) string {
	question, candidates := floatingCandidatesFromOptionTarget(target)
	readinessMu.Lock()
	currentGateState.FloatingCandidates = candidates
	currentGateState.MissingSlots = nil
	currentGateState.RiskTier = "none"
	currentGateState.ClarificationCount++
	if currentGateState.MaxClarifications == 0 {
		currentGateState.MaxClarifications = 2
	}
	readinessMu.Unlock()
	debugtrace.Record("readiness.intent_options", traceID, map[string]interface{}{
		"question":         question,
		"candidate_count":  len(candidates),
		"candidate_labels": candidateLabels(candidates),
	})
	return question
}

func candidateLabels(candidates []FloatingCandidate) []string {
	labels := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		labels = append(labels, candidate.Label)
	}
	return labels
}

// ReadinessGateState 是前端輪詢用的完整 Readiness Gate 狀態快照。
// 包含風險等級、缺欄位、浮動候選、以及掃描中的來源標籤。
type ReadinessGateState struct {
	RiskTier           string              `json:"risk_tier"` // none | normal | medium | high
	MissingSlots       []string            `json:"missing_slots"`
	FloatingCandidates []FloatingCandidate `json:"floating_candidates"`
	ClarificationCount int                 `json:"clarification_count"`
	MaxClarifications  int                 `json:"max_clarifications"`
	RetrievalScanning  bool                `json:"retrieval_scanning"`
	RetrievalSources   []string            `json:"retrieval_sources"`  // e.g. ["project docs", "uploaded files"]
	ImpactExplanation  string              `json:"impact_explanation"` // 高風險時的影響說明文字
	LowConfidence      bool                `json:"low_confidence_output"`
	AssumptionUsed     bool                `json:"assumption_used"`
	AutoOutputAllowed  bool                `json:"auto_output_allowed"` // Progressive Ready State
}

// currentGateState 保存目前的 Readiness Gate 狀態（非持久化，重啟後清空）。
// SEC-W05 第一刀（2026-05-24）：移除 readinessTraces slice 與 trace write/read methods
// （EmitDagEvent / RecordLearningEvent / UpdateReadinessGateState /
// RecordReadinessTrace / ListReadinessTraces 全部移除，無 caller）。
var (
	readinessMu      sync.Mutex
	currentGateState = ReadinessGateState{
		RiskTier:          "none",
		MaxClarifications: 2,
	}
)

// GetReadinessGateState 回傳目前的 Readiness Gate 狀態快照。
// 前端定期輪詢此方法以更新 Floating Candidate Actions、Missing Slot Capsule 等 UI 元素。
func (a *App) GetReadinessGateState() ReadinessGateState {
	readinessMu.Lock()
	defer readinessMu.Unlock()
	return currentGateState
}

// SelectFloatingCandidate 處理使用者點擊 Floating Candidate Action。
// 將選中的候選填入 structured_task.draft 並觸發重新評估。
// 回傳更新後的 ReadinessGateState。
func (a *App) SelectFloatingCandidate(candidateID string) ReadinessGateState {
	readinessMu.Lock()
	defer readinessMu.Unlock()
	// 清除已選擇的候選（前端會消失刷新）
	currentGateState.FloatingCandidates = nil
	return currentGateState
}

// DismissFloatingCandidates 清除所有浮動候選（使用者送出新訊息時呼叫）。
func (a *App) DismissFloatingCandidates() {
	readinessMu.Lock()
	defer readinessMu.Unlock()
	currentGateState.FloatingCandidates = nil
}
