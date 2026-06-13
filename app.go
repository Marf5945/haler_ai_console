package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
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
	"ui_console/shared/executil"
	"ui_console/shared/health"
	"ui_console/shared/hookgene"
	"ui_console/shared/localsearch"
	"ui_console/shared/onboarding"
	"ui_console/shared/package_import"
	"ui_console/shared/preference"
	"ui_console/shared/riskgrant"
	"ui_console/shared/scheduler"
	"ui_console/shared/settings"
	"ui_console/shared/statusrail"
	"ui_console/shared/tools"
	"ui_console/shared/websearch"
)

type schedulerSkillExecutor struct {
	app *App
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
	skillGrants     *riskgrant.Store // TASK 31：skill「允許一次」帶 TTL 授權

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
	nativeInput         *visual_learning.NativeInput
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
	toolReadinessMu        sync.Mutex
	pendingToolQuestions   map[string]pendingToolQuestion
	toolBackgroundContexts map[string][]toolBackgroundAnswer

	// #I-801: Node Sidecar 生命週期管理器
	sidecar *cli_manager.SidecarManager
	// #I-803: CLIAdapter 透過 Sidecar IPC 與 Node 通訊
	cliAdapter      skill_step.CLIAdapter
	adapterHealthMu sync.Mutex
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

	// H-12 / §3.1.5.18.7: main app owns the single hook gene recorder.
	hookGeneMu       sync.Mutex
	hookGeneRecorder *hookgene.Recorder
	hookGeneStarted  bool
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
	yoloModelBasePath := visualLearningModelBasePath(cwd)
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
		skillGrants:            riskgrant.NewStore(),
		previewCache:           make(map[string]*skill_step.ScanPreview),
		resolveCache:           make(map[string]*skill_step.ResolveResult),
		globalSessionID:        fmt.Sprintf("session-%d", time.Now().UnixNano()),
		referencePromptTargets: make(map[string][]string),
		pendingToolQuestions:   make(map[string]pendingToolQuestion),
		toolBackgroundContexts: make(map[string][]toolBackgroundAnswer),

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
		yoloDetector:        visual_learning.NewYOLODetector(yoloModelBasePath, visual_learning.NewOpenCVPipeline()),
		nativeInput:         visual_learning.NewNativeInput(),
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
	// UI 雙擊 adapter 卡片時會經 SetAdapterModelChoice 寫入 settings；此 closure 在
	// 每次 IPC sendMessage 前才查詢，所以使用者一改就生效，不用重啟 sidecar。
	sidecarAdapter.SetModelProvider(func(adapterID string) string {
		return a.settingsService.AdapterModelChoices()[adapterID]
	})
	// §29.3：注入事件匯流排，否則 checkAndEmitSummarizationNeeded 因 eventBus==nil 早退，
	// 摘要 banner 永遠不會出現（這是先前 ④ 沉睡的根因）。
	sidecarAdapter.SetEventBus(a.eventBus)
	a.cliAdapter = sidecarAdapter

	// #I-804: DAG 節流閥，避免高頻事件卡住前端
	a.dagThrottler = cli_manager.NewDAGThrottler(a.eventBus)

	// §24: 註冊文件內建能力（優先於外部 skill）

	// §27: 初始化排程引擎（啟動在 startup 中）
	a.schedulerService = scheduler.NewService(scheduler.ServiceConfig{
		DataRoot:  hookRoot,
		EventBus:  a.eventBus,
		SkillExec: schedulerSkillExecutor{app: a},
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
// SEC-05 2b（2026-06-06）：前端直呼處已收口到 OpenExternalURL binding。
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
		cmd = executil.Command("open", rawURL)
	case "linux":
		cmd = executil.Command("xdg-open", rawURL)
	case "windows":
		cmd = executil.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		log.Printf("openBrowser: unsupported OS %s", runtime.GOOS)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("openBrowser: failed to open %s: %v", rawURL, err)
	}
}

// OpenExternalURL SEC-05 2b: 前端開外部連結的唯一入口（取代直呼 BrowserOpenURL）。
// 規則：僅 http/https；metadata 主機名與無條件危險 IP 字面值拒絕；
// loopback/private 放行——開瀏覽器看本機頁面（debug trace monitor、
// GetMonitorLinks 連結）是合法用途，風險模型與 App 代抓內容不同。
func (a *App) OpenExternalURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("無法解析 URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("僅允許 http/https 連結（收到 %q）", scheme)
	}
	if err := urlsafe.ScreenExternalOpenTarget(u.Hostname()); err != nil {
		log.Printf("event=safe_http_blocked operation=open_external policy=browser host=%s decision=blocked reason=%v", u.Hostname(), err)
		return err
	}
	openBrowser(rawURL)
	return nil
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
	a.startHookGeneRecorder()
	// DEBUG_TRACE_REMOVE: Temporary local trace viewer for UI -> CLI diagnostics.
	// Keep while cleaning dead code: the monitor can look offline when the app is
	// stopped, but previous trace logs depend on this UI -> Go -> sidecar path.
	debugtrace.Start(debugtrace.DefaultAddr)
	debugtrace.Record("go.startup", "", map[string]interface{}{
		"trace_link": debugtrace.Snapshot(),
	})
	writeMonitorLinkSnapshot(debugtrace.Snapshot())
	// SEC-06: 清掉上次未正常停止的 ephemeral browser profile（冪等）。
	_ = a.CleanupEphemeralProfiles()
	// #7: Inject Wails context into event bus so it can emit to frontend.
	a.eventBus.SetContext(ctx)
	a.interruptStaleTaskRuns("App 重新啟動")
	// 啟動時把舊 skill 還空著的 Description（用法欄位）補上，讓「怎麼用」有內容可讀。
	a.backfillSkillDescriptions("startup")
	// #44 修補：把已歸檔且可見的 skill 同步進工具列，讓重開 App 後仍看得到。
	// 先 backfill 再同步，工具列同一次啟動就能吃到補完的 Description。
	a.syncArchivedSkillsToToolbar()

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
		a.refreshAdapterRuntimeHealth()
	}()

	// v3.6.1: 啟動時掃描 allowlist 到期狀態（§9.9）
	go func() {
		expiring, err := a.allowlistStore.CheckExpiries()
		if err == nil && len(expiring) > 0 {
			a.eventBus.Emit("source_trust:allowlist_expiring", expiring)
		}
	}()

	// TASK 31: 啟動時為引用文件建立缺少的向量索引
	go a.ensureReferenceVectorIndexes()

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

// DeleteTalkMessageForAgent 刪除指定 agent 的 talk_full.md 中「內容相符」的第一筆對話。
// 為何用內容比對而非索引：前端 messages 是混合陣列（talk_full 歷史 + session 即時訊息
// + 系統橫幅），index 與檔案條目不是 1:1，按 index 刪會刪錯。內容比對最壞只是 no-op
// （即時/系統訊息在檔案找不到），絕不刪錯條目。走 pipeline 記 delete_sentences 維持 hash 鏈。
// 回傳是否真的從檔案刪到一筆。
func (a *App) DeleteTalkMessageForAgent(agentID, display string) (bool, error) {
	target := strings.TrimSpace(display)
	if target == "" {
		return false, nil
	}
	root, err := conversationRootForAgent(agentID)
	if err != nil {
		return false, err
	}
	raw, err := memory.ReadTalkFull(root)
	if err != nil || strings.TrimSpace(raw) == "" {
		return false, err
	}
	// 以與 parseTalkMessages 相同的切塊規則切，逐塊還原顯示字串比對。
	parts := strings.Split(raw, "\n## [")
	deleted := false
	var b strings.Builder
	b.WriteString(parts[0]) // 開頭（# Talk Full 標頭區）原樣保留
	for _, block := range parts[1:] {
		if !deleted {
			if disp, ok := talkBlockDisplay(block); ok && disp == target {
				deleted = true
				continue // 跳過此塊 = 刪除
			}
		}
		b.WriteString("\n## [")
		b.WriteString(block)
	}
	if !deleted {
		return false, nil // 檔案沒有對應條目（前端即時/系統訊息）→ 不動檔案
	}
	if err := memory.NewPipeline(root).RewriteTalkFullForDelete(b.String()); err != nil {
		return false, err
	}
	return true, nil
}

// talkBlockDisplay 把 talk_full 一個區塊（"時間] 角色\n內文…"）還原成
// 與 parseTalkMessages 相同的顯示字串，供刪除比對。
func talkBlockDisplay(block string) (string, bool) {
	lines := strings.SplitN(block, "\n", 2)
	if len(lines) < 2 {
		return "", false
	}
	role := strings.TrimSpace(lines[0])
	if idx := strings.LastIndex(role, "]"); idx >= 0 {
		role = strings.TrimSpace(role[idx+1:])
	}
	body := strings.TrimSpace(lines[1])
	if body == "" {
		return "", false
	}
	switch role {
	case "assistant", "ai":
		return "Ai:" + body, true
	case "user":
		return body, true
	default:
		return "[" + role + "] " + body, true
	}
}

// cliSummarizer 是 adapter 摘要能力的窄介面（避免 app 直接依賴 cli_manager 具體型別）。
type cliSummarizer interface {
	RunSummarizationNow(adapterID, cliPath, model string) (conversation.Summary, error)
}

// RunSummarizationNow 由前端 banner「整理」觸發：呼叫 adapter 產摘要、寫 summaries.md、回摘要文字。
// 失敗直接回 error（Rule 11：不靜默續用舊 context）。
func (a *App) RunSummarizationNow(adapterID string) (string, error) {
	summer, ok := a.cliAdapter.(cliSummarizer)
	if !ok {
		return "", fmt.Errorf("此 adapter 不支援摘要")
	}
	// cliPath 與 sendCLIMessage 相同方式解析；model 留空交給 adapter 的 modelProvider。
	cliPath := ""
	if adapterID != "" && a.adapterRegistry != nil {
		if resolved, err := a.adapterRegistry.ResolveExecutable(adapterID); err == nil {
			cliPath = resolved
		}
	}
	sum, err := summer.RunSummarizationNow(adapterID, cliPath, "")
	if err != nil {
		return "", err
	}
	// Rule 15：寫 summaries.md（非 talk_full）；Rule 8：AppendSummary 內部做 redaction。
	root := storage.ProjectRoot(appDataRoot(), "default")
	pipeline := memory.NewPipeline(root)
	if _, err := pipeline.AppendSummary(sum.Tag, sum.Content); err != nil {
		return "", fmt.Errorf("寫入 summaries.md 失敗: %w", err)
	}
	// v3.1.7：摘要的原文細節落 deep_memory.md + index 對照，之後可用 展開ㄌtagㄌ待命 撈回。
	// 失敗不擋摘要主流程（細節層是輔助），但記 trace 供查。
	if strings.TrimSpace(sum.OriginalContent) != "" {
		deepTag := "D-" + strings.TrimPrefix(sum.Tag, "S-")
		if _, derr := pipeline.AppendDeepMemory(deepTag, sum.OriginalContent); derr == nil {
			_ = pipeline.AppendIndexEntry(memory.MemoryIndexEntry{
				SummaryTag:  sum.Tag,
				DeepTag:     deepTag,
				SentenceIDs: sum.SentenceIDs,
				CreatedAt:   time.Now().Format(time.RFC3339),
			})
		} else {
			debugtrace.Record("go.RunSummarizationNow.deep_memory_failed", "", map[string]interface{}{"error": derr.Error()})
		}
	}
	return sum.Content, nil
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

// recentConversationSentences 讀目前對話 agent 的 talk_full.md（UI 刪除對話會清空它，
// 故天然刪除感知；內容只含 user/assistant，不含系統提示），回傳最後 n 句作為路由子呼叫的歷史 H。
func (a *App) recentConversationSentences(n int) []conversation.Sentence {
	root, err := conversationRootForAgent(a.activeAgentID)
	if err != nil {
		return nil
	}
	text, err := memory.ReadTalkFull(root)
	if err != nil || strings.TrimSpace(text) == "" {
		return nil
	}
	sentences := conversation.ParseTalkFull(text)
	if n > 0 && len(sentences) > n {
		sentences = sentences[len(sentences)-n:]
	}
	// 截斷每句內容，避免把整串網頁/工具結果原文塞進每次 routing 子呼叫造成 token 暴增。
	// 路由與指代解析只需要「上一輪講了什麼」的梗概，不需要完整結果。
	out := make([]conversation.Sentence, len(sentences))
	for i, sent := range sentences {
		if r := []rune(sent.Content); len(r) > recentHistoryMaxRunesPerTurn {
			sent.Content = string(r[:recentHistoryMaxRunesPerTurn]) + "…"
		}
		out[i] = sent
	}
	return out
}

func compactRoutingHistory(recent []conversation.Sentence, current string, limit int) []conversation.Sentence {
	if limit <= 0 || len(recent) == 0 {
		return nil
	}
	currentKey := comparablePromptText(current)
	out := make([]conversation.Sentence, 0, limit)
	for i := len(recent) - 1; i >= 0; i-- {
		sent := recent[i]
		if comparablePromptText(sent.Content) == "" || comparablePromptText(sent.Content) == currentKey {
			continue
		}
		if r := []rune(sent.Content); len(r) > 90 {
			sent.Content = string(r[:90]) + "..."
		}
		out = append(out, sent)
		if len(out) >= limit {
			break
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func comparablePromptText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func compactPromptField(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.Join(strings.Fields(text), " ")
}

func formatCompactRoutingHistory(recent []conversation.Sentence, current string, limit int) string {
	recent = compactRoutingHistory(recent, current, limit)
	if len(recent) == 0 {
		return ""
	}
	parts := make([]string, 0, len(recent))
	for _, sent := range recent {
		role := "A"
		switch strings.TrimSpace(sent.Role) {
		case "user":
			role = "U"
		case "assistant":
			role = "A"
		case "tool-action":
			role = "T"
		}
		parts = append(parts, role+":"+compactPromptField(sent.Content))
	}
	return "H=" + strings.Join(parts, " / ")
}

// recentHistoryMaxRunesPerTurn 是注入路由子呼叫的每句歷史長度上限（rune）。
const recentHistoryMaxRunesPerTurn = 200

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

// pushActionStatus 把一個動作合成人類狀態句推到 status rail（只給人看，不外洩 ㄌ wire format）。
// 動作執行前呼叫，讓使用者看到「正在用網路搜尋「…」…」這類即時狀態。
func (a *App) pushActionStatus(action, target string) {
	if a == nil || a.statusRail == nil {
		return
	}
	view := a.statusRail.SetText(actionchain.HumanStatus(action, target, actionchain.PhaseRunning))
	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventStatusRailUpdated, view)
	}
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

func (a *App) GetTTSPackStatus() voice.TTSPackStatus {
	return a.voiceService.TTSPackStatus()
}

func (a *App) InstallTTSPack() (voice.TTSPackStatus, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	downloadCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	return a.voiceService.InstallTTSPack(downloadCtx)
}

func (a *App) RemoveTTSPack() (voice.TTSPackStatus, error) {
	return a.voiceService.RemoveTTSPack()
}

func (a *App) SavePersona(persona settings.Persona) settings.State {
	_ = a.settingsService.SavePersona(persona)
	return a.GetSettingsState()
}

func (a *App) ReorderPersonas(orderIDs []string) settings.State {
	_ = a.settingsService.ReorderPersonas(orderIDs)
	return a.GetSettingsState()
}

// ---------------------------------------------------------------------------
// #44 修補：已安裝的 skill / MCP 註冊進工具列（toolsService）。
// 先前 AddTool 從未被呼叫，導致歸檔成功的 skill 與確認安裝的 MCP 都不會
// 出現在「工具」面板、也不會被列為可候選工具。以下集中補上註冊路徑。
// ---------------------------------------------------------------------------

type cliActionTagSetter interface {
	SetActionTags(tags []string)
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
	// Visual Learning operation replay is app-owned; LLMs may select it via action-chain.
	add([]string{"操作", "程式", "流程"})
	sort.Strings(tags)
	return tags
}

type pendingLocalSearchWebFallback struct {
	Query     string
	Limit     int
	ExpiresAt time.Time
}

var (
	pendingLocalSearchWebFallbackMu sync.Mutex
	pendingLocalSearchWebFallbacks  = map[string]pendingLocalSearchWebFallback{}
)

func (a *App) recentReferenceFilesForRouting(limit int) []routingReferenceFile {
	if limit <= 0 {
		limit = 6
	}
	referenceDir := filepath.Join(appDataRoot(), "data", "references", "files")
	entries, err := os.ReadDir(referenceDir)
	if err != nil {
		return nil
	}
	var refs []routingReferenceFile
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := entry.Name()
		refs = append(refs, routingReferenceFile{
			Name:       name,
			Path:       filepath.Join(referenceDir, name),
			Ext:        strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), "."),
			ModifiedAt: info.ModTime(),
		})
	}
	sort.SliceStable(refs, func(i, j int) bool {
		return refs[i].ModifiedAt.After(refs[j].ModifiedAt)
	})
	if len(refs) > limit {
		refs = refs[:limit]
	}
	return refs
}

func compactReferenceFilesForTrace(refs []routingReferenceFile) []map[string]string {
	if len(refs) == 0 {
		return nil
	}
	out := make([]map[string]string, 0, len(refs))
	for i, ref := range refs {
		if i >= 6 {
			break
		}
		out = append(out, map[string]string{
			"name":  ref.Name,
			"ext":   ref.Ext,
			"mtime": ref.ModifiedAt.Format(time.RFC3339),
		})
	}
	return out
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
	resp, err := a.sendCLIMessage(adapterID, sessionID, userText, traceID, "")
	// 配額／限流即時偵測：模型額度用盡時，UI 不該看到原始錯誤，改提示切換模型。
	if notice, hit := quotaSwitchModelNotice(adapterID, resp, err); hit {
		debugtrace.Record("go.SendCLIMessage.quota_exhausted", traceID, map[string]interface{}{
			"adapter_id": adapterID,
			"resp_error": cliResponseErrorString(resp),
			"err":        errorString(err),
		})
		return &skill_step.CLIResponse{Text: notice}, nil
	}
	return resp, err
}

// routeUserIntentOnce 是「一句使用者意圖」走完整對話路由的唯一入口：直接重用
// SendCLIMessage 的捷徑→判斷器→dispatch（含 用法/list/搜尋/resourceGate/toolReadiness/流程）。
// 一般聊天與 DAG 的 chat_route 節點都經由它，確保路由真相只有一套（不另立第二套路由器）。
// 注意：traceID 必須是「非 task 前綴」，否則會被 isTaskProgressTraceID 當成內部任務而跳過路由。
func (a *App) routeUserIntentOnce(adapterID, sessionID, userText, traceID string) (*skill_step.CLIResponse, error) {
	return a.SendCLIMessage(adapterID, sessionID, userText, traceID)
}

// quotaMarkers 是各家模型「配額／容量用盡、限流」錯誤的特徵字串（小寫比對）。
// 這些字串只會出現在錯誤訊息，不會出現在正常的 skill／文件內容，掃 resp.Text
// 也不易誤判。
var quotaMarkers = []string{
	"exhausted your capacity", // gemini: You have exhausted your capacity on this model
	"retryablequotaerror",
	"quota will reset",
	"quota exceeded",
	"insufficient_quota",
	"model_capacity_exhausted",
	"no capacity available for model",
	"resource_exhausted",
	"ratelimitexceeded",
	"too many requests",
	"status 429",
	"容量不足",
	"伺服器容量不足",
	"限流",
}

// cliResponseErrorString 安全取出 resp.Error。
func cliResponseErrorString(resp *skill_step.CLIResponse) string {
	if resp == nil {
		return ""
	}
	return resp.Error
}

func (a *App) sendCLIMessage(adapterID string, sessionID string, userText string, traceID string, modelOverride string) (*skill_step.CLIResponse, error) {
	// === 診斷 log：記錄每次 CLI 呼叫的關鍵參數，方便追蹤問題 ===
	log.Printf("SendCLIMessage: adapter=%q session=%q text_len=%d", adapterID, sessionID, len(userText))
	// DEBUG_TRACE_REMOVE: Captures the Wails binding input with full dev-mode text.
	sendTrace := map[string]interface{}{
		"adapter_id": adapterID,
		"session_id": sessionID,
		"text_len":   len(userText),
	}
	addDebugTraceUserText(sendTrace, traceID, userText)
	debugtrace.Record("go.SendCLIMessage.enter", traceID, sendTrace)

	// 本回合附圖（CLI 路徑）：先消費暫存圖片，避免外洩到下一則。
	stagedImages := takeSessionImages(sessionID)

	var inj *skill_step.Injection
	if a.skillInjections != nil {
		inj = a.skillInjections.Get(sessionID)
	}
	actionTags := a.syncActionTagsToCLIAdapter(traceID)
	isTaskProgressInternal := isTaskProgressTraceID(traceID)
	if !isTaskProgressInternal {
		if resp, handled := a.maybeHandleResourceGate(userText, sessionID, traceID); handled {
			debugtrace.Record("go.SendCLIMessage.resource_gate.direct", traceID, map[string]interface{}{
				"text":   resp.Text,
				"action": resp.Action,
				"target": resp.Target,
				"next":   resp.Next,
			})
			return resp, nil
		}
		if decision, handled := a.consumePendingToolAnswer(sessionID, userText, traceID); handled {
			if routed, routedResp := a.responseFromToolRoutingDecision(decision, sessionID, traceID, decision.Raw); routed {
				return &routedResp, nil
			}
		}
		if resp, handled := a.maybeHandleWebSearch(userText, sessionID, traceID); handled {
			debugtrace.Record("go.SendCLIMessage.web_search.direct", traceID, map[string]interface{}{
				"text":          resp.Text,
				"error":         resp.Error,
				"auth_required": false,
				"auth_url":      "",
			})
			return resp, nil
		}
		if resp, handled := a.maybeHandleLocalSearch(userText, sessionID, traceID); handled {
			debugtrace.Record("go.SendAPIMessage.local_search.direct", traceID, map[string]interface{}{
				"text":          resp.Text,
				"error":         resp.Error,
				"auth_required": false,
				"auth_url":      "",
			})
			return resp, nil
		}
	}

	// CLI inline 讀圖是各 CLI 自家介面；先附上誠實提示讓圖片不被靜默吞掉，
	// 待接該 CLI 的 image flag 時改 cliImageNotice / 此處編碼即可。
	if notice := cliImageNotice(stagedImages); notice != "" {
		userText += notice
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
	if isOllamaPromptCLI(adapterID, cliPath) {
		message := "Ollama 不能用 CLI adapter 方式聊天；請在引用連結貼 %%USERPROFILE%%\\.ollama\\models 註冊成本地模型。"
		debugtrace.Record("go.SendCLIMessage.ollama_cli_blocked", traceID, map[string]interface{}{
			"adapter_id": adapterID,
			"cli_path":   cliPath,
			"message":    message,
		})
		return &skill_step.CLIResponse{Error: message}, nil
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

	personaPrompt := a.buildMainComposerPrompt(a.getActivePersona())
	if bg := a.formatToolBackgroundContext(sessionID); bg != "" {
		personaPrompt += "\n\n" + bg + "\n回答或組織工具結果時，只能使用已補充背景與工具結果，不可新增未確認的地點、時間、數字或來源。"
	}
	routingContextForToolPrompt := ""
	if !isTaskProgressInternal {
		// Problem 2：上一輪本機找不到並記了 pending 網路 fallback，這輪是肯定回覆
		// → 直接走網路、跳過 keyword/judge（修迴圈、省 token）。
		if resp, ok := a.maybeHandlePendingLocalSearchWebFallback(userText, sessionID, traceID); ok {
			return resp, nil
		}
		recentHistory := compactRoutingHistory(a.recentConversationSentences(6), userText, 3)
		keywordResp, keywordErr := a.cliAdapter.SendMessage(skill_step.CLIMessageOptions{
			AdapterID:      adapterID,
			CLIPath:        cliPath,
			SessionID:      sessionID,
			UserText:       buildSearchTermExtractionPrompt("", userText, recentHistory),
			Model:          strings.TrimSpace(modelOverride),
			SystemPrompt:   "",
			ContinuityKey:  conversationContinuityKey("tool-keywords", sessionID),
			TraceID:        traceID,
			SkipContinuity: true,
		})
		debugtrace.Record("go.searchTerms.extract", traceID, map[string]interface{}{
			"text":  keywordResp.Text,
			"error": keywordResp.Error,
			"err":   errorString(keywordErr),
		})
		if keywordErr != nil || keywordResp.Error != "" || keywordResp.AuthRequired {
			return &keywordResp, keywordErr
		}
		terms := parseSearchTerms(keywordResp.Text, userText)
		routingLookup := a.lookupToolRoutingContext(terms, userText, traceID)
		// SEC-06: 若上一輪系統發過確認問句，judge prompt 帶 pending 摘要，
		// 讓「我要取消」這類回應不被誤判成操作/查詢。
		routingContextForToolPrompt = formatToolRoutingLookupContext(routingLookup) + pendingConfirmPromptContext(sessionID)
		judgeResp, judgeErr := a.cliAdapter.SendMessage(skill_step.CLIMessageOptions{
			AdapterID:      adapterID,
			CLIPath:        cliPath,
			SessionID:      sessionID,
			UserText:       buildToolRoutingDecisionPrompt("", userText, routingContextForToolPrompt, recentHistory),
			Model:          strings.TrimSpace(modelOverride),
			SystemPrompt:   "",
			ContinuityKey:  conversationContinuityKey("tool-judge", sessionID),
			TraceID:        traceID,
			SkipContinuity: true,
		})
		if judgeResp.AuthRequired {
			return &judgeResp, judgeErr
		}
		if judgeErr != nil || judgeResp.Error != "" {
			// judge 失敗：先用本機候選做保底路由，撈不到才把錯誤丟回。
			if fb, ok := fallbackDecisionFromLookup(routingLookup); ok {
				debugtrace.Record("go.toolRouting.judge_fallback", traceID, map[string]interface{}{
					"judge_error": errorString(judgeErr),
					"resp_error":  judgeResp.Error,
					"action":      fb.Action,
					"target":      fb.Target,
					"local_hits":  len(routingLookup.LocalMatches),
				})
				if handled, routedResp := a.responseFromToolRoutingDecision(fb, sessionID, traceID, userText); handled {
					return &routedResp, nil
				}
			}
			return &judgeResp, judgeErr
		}
		decision := normalizeToolRoutingDecision(parseToolRoutingDecision(judgeResp.Text), userText, routingLookup)
		if shouldRepairToolRoutingDecision(userText, decision) {
			repairResp, repairErr := a.cliAdapter.SendMessage(skill_step.CLIMessageOptions{
				AdapterID:      adapterID,
				CLIPath:        cliPath,
				SessionID:      sessionID,
				UserText:       buildToolRoutingRepairPrompt(buildToolRoutingDecisionPrompt("", userText, routingContextForToolPrompt, recentHistory), judgeResp.Text, userText),
				Model:          strings.TrimSpace(modelOverride),
				SystemPrompt:   "",
				ContinuityKey:  conversationContinuityKey("tool-judge-repair", sessionID),
				TraceID:        traceID + "-repair",
				SkipContinuity: true,
			})
			debugtrace.Record("go.toolRouting.judge_repair", traceID, map[string]interface{}{
				"text":        repairResp.Text,
				"error":       repairResp.Error,
				"judge_error": errorString(repairErr),
			})
			if repairErr == nil && repairResp.Error == "" && !repairResp.AuthRequired {
				repaired := normalizeToolRoutingDecision(parseToolRoutingDecision(repairResp.Text), userText, routingLookup)
				if repaired.Kind == toolRoutingDecisionAction || repaired.Kind == toolRoutingDecisionNeedTool {
					decision = repaired
				}
			}
		}
		debugtrace.Record("go.toolRouting.judge", traceID, map[string]interface{}{
			"text":           judgeResp.Text,
			"error":          judgeResp.Error,
			"decision":       decision.Kind,
			"action":         decision.Action,
			"target":         decision.Target,
			"next":           decision.Next,
			"need_tool":      decision.Kind == toolRoutingDecisionNeedTool,
			"lookup_query":   routingLookup.Query,
			"operation_hits": len(routingLookup.Operations),
			"recent_ops":     len(routingLookup.RecentOperations),
			"local_hits":     len(routingLookup.LocalMatches),
			"judge_error":    errorString(judgeErr),
		})
		if handled, routedResp := a.responseFromToolRoutingDecision(decision, sessionID, traceID, userText); handled {
			return &routedResp, nil
		}
	}

	var referencePlan referenceSearchPlan
	if isTaskProgressInternal {
		referencePlan = planTaskProgressReferenceSearch(traceID, userText)
	} else if isLearningOperationCatalogText(userText) {
		referencePlan = referenceSearchPlan{}
	} else {
		referencePlan = a.planReferenceSearchWithCLI(adapterID, cliPath, sessionID, userText, traceID)
	}
	systemPrompt := personaPrompt + webSearchRoutingPrompt()
	if routingContextForToolPrompt != "" {
		systemPrompt += routingContextForToolPrompt
	}
	docSearchContext := a.buildDocSearchContext(sessionID, userText, adapterID, traceID, referencePlan)
	if docSearchContext != "" {
		systemPrompt += docSearchContext
	}
	if !isTaskProgressInternal {
		systemPrompt += a.buildLearningReplayPromptContext(traceID)
		systemPrompt += a.buildLearningOperationPromptContext(userText, traceID)
	}
	continuityKey := conversationContinuityKey("composer", sessionID)
	if isTaskProgressInternal {
		continuityKey = conversationContinuityKey("task-progress", sessionID)
	}
	opts := skill_step.CLIMessageOptions{
		AdapterID:      adapterID,
		CLIPath:        cliPath,
		SessionID:      sessionID,
		UserText:       userText,
		Model:          strings.TrimSpace(modelOverride),
		SkillInjection: inj,
		SystemPrompt:   systemPrompt,
		ContinuityKey:  continuityKey,
		TraceID:        traceID,
		SkipContinuity: isTaskProgressInternal,
	}
	// DEBUG_TRACE_REMOVE: Captures the final Go options before the sidecar adapter.
	optionsTrace := map[string]interface{}{
		"adapter_id":          opts.AdapterID,
		"cli_path":            opts.CLIPath,
		"session_id":          opts.SessionID,
		"model":               opts.Model,
		"continuity_key":      opts.ContinuityKey,
		"system_prompt_len":   len([]rune(opts.SystemPrompt)),
		"reference_plan":      referencePlan,
		"has_doc_search_ctx":  docSearchContext != "",
		"has_skill_injection": opts.SkillInjection != nil,
		"action_tags":         actionTags,
	}
	addDebugTraceUserText(optionsTrace, traceID, opts.UserText)
	debugtrace.Record("go.CLIMessageOptions", traceID, optionsTrace)
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
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: resp.Action, Target: resp.Target, Next: resp.Next}
	if handled, readinessResp := a.maybeAskForToolReadiness(sessionID, decision, userText, traceID); handled {
		return &readinessResp, nil
	}
	if handled, flowResp := a.maybeHandleSkillFlow(decision, sessionID, traceID, userText); handled {
		return &flowResp, nil
	}
	if handled, programResp := a.maybeHandleGoProgramAuthoring(decision, sessionID, traceID, userText); handled {
		return &programResp, nil
	}
	target := resp.Target
	if resp.Action == "網路" {
		target = a.targetWithBackground(sessionID, target)
	}
	if req, ok := websearch.RequestFromAction(resp.Action, target); ok {
		webResp := a.executeWebSearch(req, traceID)
		return &webResp, nil
	}
	if req, ok := localsearch.RequestFromAction(resp.Action, resp.Target); ok {
		// CLI/local model only chooses the builtin action; App owns the actual local scan.
		localResp := a.executeLocalSearch(req, sessionID, traceID)
		return &localResp, nil
	}
	// 展開：撈回 deep_memory 細節（v3.1.7），與 搜尋 同模式。
	if handled, memResp := a.maybeExpandMemory(resp.Action, resp.Target, traceID); handled {
		return &memResp, nil
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
	resp, err := a.sendAPIMessageImpl(adapterID, sessionID, userText, traceID)
	// 配額／限流即時偵測（同 CLI 路徑）。task-progress 內部呼叫交給 planner 自己
	// 的 transient 處理，避免把錯誤改寫成文字打亂其重試／澄清流程。
	if !isTaskProgressTraceID(traceID) {
		if notice, hit := quotaSwitchModelNotice(adapterID, resp, err); hit {
			debugtrace.Record("go.SendAPIMessage.quota_exhausted", traceID, map[string]interface{}{
				"adapter_id": adapterID,
				"resp_error": cliResponseErrorString(resp),
				"err":        errorString(err),
			})
			return &skill_step.CLIResponse{Text: notice}, nil
		}
	}
	return resp, err
}

func (a *App) sendAPIMessageImpl(adapterID string, sessionID string, userText string, traceID string) (*skill_step.CLIResponse, error) {
	debugtrace.Record("go.SendAPIMessage.enter", traceID, map[string]interface{}{
		"adapter_id": adapterID,
		"session_id": sessionID,
		"text_len":   len(userText),
	})
	actionTags := a.syncActionTagsToCLIAdapter(traceID)
	isTaskProgressInternal := isTaskProgressTraceID(traceID)
	if !isTaskProgressInternal {
		if resp, handled := a.maybeHandleResourceGate(userText, sessionID, traceID); handled {
			debugtrace.Record("go.SendAPIMessage.resource_gate.direct", traceID, map[string]interface{}{
				"text":   resp.Text,
				"action": resp.Action,
				"target": resp.Target,
				"next":   resp.Next,
			})
			return resp, nil
		}
		if decision, handled := a.consumePendingToolAnswer(sessionID, userText, traceID); handled {
			if routed, routedResp := a.responseFromToolRoutingDecision(decision, sessionID, traceID); routed {
				return &routedResp, nil
			}
		}
		if resp, handled := a.maybeHandleWebSearch(userText, sessionID, traceID); handled {
			debugtrace.Record("go.SendAPIMessage.web_search.direct", traceID, map[string]interface{}{
				"text":          resp.Text,
				"error":         resp.Error,
				"auth_required": false,
				"auth_url":      "",
			})
			return resp, nil
		}
		// 與 CLI 路徑對齊：API 路徑同樣在進判斷前處理本機搜尋／列出 skill，
		// 否則「列出你有的 skill」會被判為閒聊、交給模型自答。
		if resp, handled := a.maybeHandleLocalSearch(userText, sessionID, traceID); handled {
			debugtrace.Record("go.SendAPIMessage.local_search.direct", traceID, map[string]interface{}{
				"text":  resp.Text,
				"error": resp.Error,
			})
			return resp, nil
		}
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
	// SEC-05 2a: Safe Client，policy 由 PolicyForLLMEndpoint 集中決定。
	client := urlsafe.NewSafeClient(urlsafe.PolicyForLLMEndpoint(cfg.ProviderID, cfg.BaseURL), "llm_chat", 45*time.Second)
	// 本回合若有附圖（API 路徑）：消費暫存圖片，於 buildOpenAIRequestBody 內以 base64 內嵌，不落地。
	stagedImages := takeSessionImages(sessionID)
	doRequest := func(prompt string) (*http.Response, error) {
		body, err := buildOpenAIRequestBody(model, prompt, stagedImages)
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

	personaPrompt := a.buildMainComposerPrompt(a.getActivePersona())
	if bg := a.formatToolBackgroundContext(sessionID); bg != "" {
		personaPrompt += "\n\n" + bg + "\n回答或組織工具結果時，只能使用已補充背景與工具結果，不可新增未確認的地點、時間、數字或來源。"
	}
	routingContextForToolPrompt := ""
	if !isTaskProgressInternal && !isLocalAdapter {
		// Problem 2（API 路徑同 CLI）：pending 網路 fallback + 肯定回覆 → 直接走網路。
		if resp, ok := a.maybeHandlePendingLocalSearchWebFallback(userText, sessionID, traceID); ok {
			return resp, nil
		}
		recentHistory := compactRoutingHistory(a.recentConversationSentences(6), userText, 3)
		keywordText, keywordErr := callAPI(buildSearchTermExtractionPrompt("", userText, recentHistory))
		debugtrace.Record("go.searchTerms.extract", traceID, map[string]interface{}{
			"text":         keywordText,
			"error":        errorString(keywordErr),
			"adapter_kind": "api",
		})
		if keywordErr != nil {
			return &skill_step.CLIResponse{Error: keywordErr.Error()}, nil
		}
		terms := parseSearchTerms(keywordText, userText)
		routingLookup := a.lookupToolRoutingContext(terms, userText, traceID)
		// SEC-06: judge prompt 帶 pending 確認摘要（同 CLI 路徑）。
		routingContextForToolPrompt = formatToolRoutingLookupContext(routingLookup) + pendingConfirmPromptContext(sessionID)
		judgeText, judgeErr := callAPI(buildToolRoutingDecisionPrompt("", userText, routingContextForToolPrompt, recentHistory))
		if judgeErr != nil {
			// judge 失敗：先用本機候選做保底路由，撈不到才把錯誤丟回。
			if fb, ok := fallbackDecisionFromLookup(routingLookup); ok {
				debugtrace.Record("go.toolRouting.judge_fallback", traceID, map[string]interface{}{
					"judge_error":  judgeErr.Error(),
					"action":       fb.Action,
					"target":       fb.Target,
					"local_hits":   len(routingLookup.LocalMatches),
					"adapter_kind": "api",
				})
				if handled, routedResp := a.responseFromToolRoutingDecision(fb, sessionID, traceID, userText); handled {
					return &routedResp, nil
				}
			}
			return &skill_step.CLIResponse{Error: judgeErr.Error()}, nil
		}
		decision := normalizeToolRoutingDecision(parseToolRoutingDecision(judgeText), userText, routingLookup)
		if shouldRepairToolRoutingDecision(userText, decision) {
			repairText, repairErr := callAPI(buildToolRoutingRepairPrompt(buildToolRoutingDecisionPrompt("", userText, routingContextForToolPrompt, recentHistory), judgeText, userText))
			debugtrace.Record("go.toolRouting.judge_repair", traceID, map[string]interface{}{
				"text":         repairText,
				"error":        errorString(repairErr),
				"adapter_kind": "api",
			})
			if repairErr == nil {
				repaired := normalizeToolRoutingDecision(parseToolRoutingDecision(repairText), userText, routingLookup)
				if repaired.Kind == toolRoutingDecisionAction || repaired.Kind == toolRoutingDecisionNeedTool {
					decision = repaired
				}
			}
		}
		debugtrace.Record("go.toolRouting.judge", traceID, map[string]interface{}{
			"text":           judgeText,
			"error":          errorString(judgeErr),
			"decision":       decision.Kind,
			"action":         decision.Action,
			"target":         decision.Target,
			"next":           decision.Next,
			"need_tool":      decision.Kind == toolRoutingDecisionNeedTool,
			"lookup_query":   routingLookup.Query,
			"operation_hits": len(routingLookup.Operations),
			"recent_ops":     len(routingLookup.RecentOperations),
			"local_hits":     len(routingLookup.LocalMatches),
			"adapter_kind":   "api",
		})
		if handled, routedResp := a.responseFromToolRoutingDecision(decision, sessionID, traceID, userText); handled {
			return &routedResp, nil
		}
	}

	var referencePlan referenceSearchPlan
	if isTaskProgressInternal {
		referencePlan = planTaskProgressReferenceSearch(traceID, userText)
	} else if isLocalAdapter {
		referencePlan = referenceSearchPlan{}
	} else if isLearningOperationCatalogText(userText) {
		referencePlan = referenceSearchPlan{}
	} else {
		referencePlan = a.planReferenceSearchWithAPI(sessionID, userText, traceID, callAPI)
	}
	basePrompt := personaPrompt + webSearchRoutingPrompt()
	if routingContextForToolPrompt != "" {
		basePrompt += routingContextForToolPrompt
	}
	docSearchContext := a.buildDocSearchContext(sessionID, userText, adapterID, traceID, referencePlan)
	if docSearchContext != "" {
		basePrompt += docSearchContext
	}
	if !isTaskProgressInternal {
		basePrompt += a.buildLearningReplayPromptContext(traceID)
		basePrompt += a.buildLearningOperationPromptContext(userText, traceID)
	}
	var synthesizedPrompt string
	if isLocalAdapter {
		synthesizedPrompt = buildLocalModelPrompt(basePrompt, actionTags, userText)
	} else {
		synthesizedPrompt = buildAPIActionChainPrompt(basePrompt, actionTags, userText)
	}
	debugtrace.Record("go.APIMessageOptions", traceID, map[string]interface{}{
		"adapter_id":         adapterID,
		"session_id":         sessionID,
		"user_text":          userText,
		"system_prompt_len":  len([]rune(basePrompt)),
		"synth_prompt_len":   len([]rune(synthesizedPrompt)),
		"reference_plan":     referencePlan,
		"has_doc_search_ctx": docSearchContext != "",
		"has_now":            strings.Contains(synthesizedPrompt, "now="),
		"has_action_chain":   strings.Contains(synthesizedPrompt, "動作ㄌ目標ㄌ下一步"),
		"has_control_seal":   strings.Contains(synthesizedPrompt, "本輪命令前綴"),
		"action_tags":        actionTags,
		"action_tags_count":  len(actionTags),
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

const visualLearningReplayDirective = "[[控制:回放剛剛示範]]"
const learningPromptMaxSteps = 8

type learningMetadataLLMResponse struct {
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Keywords     []string `json:"keywords"`
	OperationTag string   `json:"operation_tag"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func buildAPIActionChainPrompt(systemPrompt string, actionTags []string, userText string) string {
	return conversation.Synthesize(conversation.SynthesisConfig{
		SystemPrompt: systemPrompt + webSearchRoutingPrompt(),
		ActionTags:   append([]string(nil), actionTags...),
		CurrentInput: userText,
		CommandSeal:  controlseal.CurrentSeal(),
		SanitizeLLM:  true,
	})
}

type toolRoutingLookupContext struct {
	Query            string
	Terms            []string
	Operations       []visual_learning.OperationSearchResult
	RecentOperations []visual_learning.LearningRunCatalogItem
	LocalMatches     []localsearch.SearchResult
	RecentReferences []routingReferenceFile
}

type routingReferenceFile struct {
	Name       string
	Path       string
	Ext        string
	ModifiedAt time.Time
}

const (
	toolRoutingDecisionNeedTool = "need_tool"
	toolRoutingDecisionChat     = "chat"
	toolRoutingDecisionAction   = "action"
)

type toolRoutingDecision struct {
	Kind   string
	Text   string
	Action string
	Target string
	Next   string
	Raw    string
}

// fallbackDecisionFromLookup 在 judge 呼叫失敗（配額耗盡、CLI 逾時、Gemini 嘗試
// 呼叫不存在的工具等）時，用 lookup 已取得的本機候選推導保底決策：本機若已命中
// （skill／工具／文件），就視為本機搜尋把使用者導向本機資料，而不是把 CLI 錯誤
// 直接丟回。沒有任何候選時回 false，維持原本的錯誤回傳行為。
func fallbackDecisionFromLookup(lookup toolRoutingLookupContext) (toolRoutingDecision, bool) {
	if len(lookup.LocalMatches) == 0 {
		return toolRoutingDecision{}, false
	}
	target := strings.TrimSpace(lookup.Query)
	if target == "" {
		target = strings.TrimSpace(lookup.LocalMatches[0].Title)
	}
	if target == "" {
		return toolRoutingDecision{}, false
	}
	return toolRoutingDecision{
		Kind:   toolRoutingDecisionAction,
		Action: "搜尋",
		Target: target,
		Next:   actionchain.StandbyNext,
	}, true
}

func isLoadedReferenceVisibilityQuestion(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	mentionsFile := containsAny(trimmed, []string{"檔案", "文件", "引用", "引用文件", "拉進來", "拖進來", "剛剛", "最近", "已載入", "匯入"}) ||
		containsAny(lower, []string{"file", "files", "upload", "uploaded", "drag", "drop", "reference"})
	if !mentionsFile {
		return false
	}
	asksVisibility := containsAny(trimmed, []string{"有看到", "看到", "看得到", "知道", "有沒有", "是不是有", "列出", "有哪些", "什麼檔"}) ||
		containsAny(lower, []string{"see", "seen", "loaded", "list", "what file", "which file"})
	if !asksVisibility {
		return false
	}
	return !containsAny(trimmed, []string{"產生", "生成", "製作", "做", "分析", "比較", "轉換", "匯出", "處理"})
}

func formatRecentReferenceFilesAnswer(refs []routingReferenceFile) string {
	if len(refs) == 0 {
		return "我這邊目前沒有看到已載入的引用檔。"
	}
	var b strings.Builder
	b.WriteString("有，已載入的引用檔：")
	for i, ref := range refs {
		if i >= 6 {
			break
		}
		fmt.Fprintf(&b, "\n%d. %s（%s，%s）", i+1, ref.Name, strings.ToUpper(ref.Ext), ref.ModifiedAt.Format("2006-01-02 15:04:05"))
	}
	return b.String()
}

func inferGoProgramAuthoringRequest(userText string) (string, bool) {
	text := strings.TrimSpace(userText)
	if text == "" {
		return "", false
	}
	lower := strings.ToLower(text)
	if !containsAny(lower, []string{"skill", "小程式", "程式", "program"}) {
		return "", false
	}
	if !containsAny(text, []string{"做", "建立", "製作", "產生", "生成", "新增", "幫我做", "幫我建立", "幫我製作"}) {
		return "", false
	}
	if !containsAny(lower, []string{"json", "表格", "csv", "xlsx", "資料", "data", "輸出", "輸入", "建議", "判斷", "比較", "轉換", "計算", "處理", "table", "report"}) &&
		!containsAny(text, []string{"表", "料表", "欄位", "欄", "格式", "清單", "報表"}) {
		return "", false
	}
	name := extractGoProgramName(text)
	if name == "" {
		name = "資料處理程式"
	}
	return name, true
}

func extractGoProgramName(text string) string {
	cleaned := strings.TrimSpace(text)
	replacer := strings.NewReplacer("「", "", "」", "", "『", "", "』", "", "\"", "", "'", "")
	cleaned = replacer.Replace(cleaned)
	for _, prefix := range []string{"請幫我", "幫我", "請", "我要", "我想要"} {
		cleaned = strings.TrimPrefix(strings.TrimSpace(cleaned), prefix)
	}
	for _, verb := range []string{"做一個", "建立一個", "製作一個", "產生一個", "生成一個", "新增一個", "做", "建立", "製作", "產生", "生成", "新增"} {
		if strings.HasPrefix(cleaned, verb) {
			cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, verb))
			break
		}
	}
	for {
		before := cleaned
		for _, prefix := range []string{"skill", "Skill", "小程式", "程式", "program", "Program", "輸入"} {
			cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, prefix))
		}
		cleaned = strings.Trim(cleaned, " :-_／/")
		if cleaned == before {
			break
		}
	}
	cutAt := len(cleaned)
	for _, marker := range []string{"，", ",", "。", "\n", "依照", "根據", "用", "來", "可以", "能", "會", "輸出"} {
		if idx := strings.Index(cleaned, marker); idx >= 0 && idx < cutAt {
			cutAt = idx
		}
	}
	cleaned = strings.TrimSpace(cleaned[:cutAt])
	for _, suffix := range []string{"skill", "Skill", "小程式", "程式", "program", "Program"} {
		cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, suffix))
	}
	cleaned = strings.Trim(cleaned, " -_／/")
	// 攔截「用途／用法／做法／作法」這類問法/說明詞，避免被當成 skill 名稱的一部分
	// （例如「產出電料Bom的用法」→「產出電料Bom」）。連同緊鄰的「的」一起反覆去除，
	// 直到名稱穩定。
	for {
		before := cleaned
		for _, w := range []string{"用途", "用法", "做法", "作法"} {
			cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, w))
		}
		cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, "的"))
		cleaned = strings.Trim(cleaned, " -_／/")
		if cleaned == before {
			break
		}
	}
	if len([]rune(cleaned)) > 24 {
		r := []rune(cleaned)
		cleaned = string(r[:24])
	}
	return strings.TrimSpace(cleaned)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// assembleLocalResponse parses a local model's 3-line output into an ActionChain.
// Falls back to treating the whole output as chat if parsing fails.
func assembleLocalResponse(raw string) actionchain.ActionChain {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	// Filter out empty lines.
	var nonEmpty []string
	for _, l := range lines {
		l = stripLocalModelFieldLabel(l)
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
	cmd := executil.Command("git", "status", "--short")
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
	decision := toolRoutingDecision{Kind: toolRoutingDecisionAction, Action: chain.Action, Target: chain.Target, Next: chain.Next}
	// 此路徑無原始輸入變數，用 rawText（含 ㄌ 鏈，可能帶地點）做地點線索檢查。
	if handled, resp := a.maybeAskForToolReadiness(sessionID, decision, rawText, traceID); handled {
		return resp
	}
	if handled, resp := a.maybeHandleGoProgramAuthoring(decision, sessionID, traceID, rawText); handled {
		return resp
	}
	target := chain.Target
	if chain.Action == "網路" {
		target = a.targetWithBackground(sessionID, target)
	}
	if req, ok := websearch.RequestFromAction(chain.Action, target); ok {
		resp := a.executeWebSearch(req, traceID)
		resp.Action = chain.Action
		resp.Target = target
		resp.Next = chain.Next
		return resp
	}
	if req, ok := localsearch.RequestFromAction(chain.Action, chain.Target); ok {
		// LLM only routes intent with ㄌ syntax; search results never go back to the model.
		resp := a.executeLocalSearch(req, sessionID, traceID)
		resp.Action = chain.Action
		resp.Target = chain.Target
		resp.Next = chain.Next
		return resp
	}
	// 展開：撈回 deep_memory 細節（v3.1.7）。
	if handled, memResp := a.maybeExpandMemory(chain.Action, chain.Target, traceID); handled {
		memResp.Next = chain.Next
		return memResp
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
		// v3.1.8：loop 來源不出工具卡（loop 自己管 pending/risk），消除雙重提示。
		if a.eventBus != nil && !eventbus.IsTaskLoopTrace(traceID) {
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
	if isOllamaPromptCLI(adapterID, cliPath) {
		message := "Ollama 不能用 CLI adapter 方式聊天；請在引用連結貼 %%USERPROFILE%%\\.ollama\\models 註冊成本地模型。"
		debugtrace.Record("go.SendInspectorMessage.ollama_cli_blocked", traceID, map[string]interface{}{
			"adapter_id": adapterID,
			"cli_path":   cliPath,
			"message":    message,
		})
		return &skill_step.CLIResponse{Error: message}, nil
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
		a.finishInspectorReply(adapterID, userText, &resp)
	}

	return &resp, nil
}

// SendTopInteractionMessage keeps the frontend on one top-lane contract.
// Backend drivers still differ: CLI uses sidecar; API/local uses HTTP.
func (a *App) SendTopInteractionMessage(adapterID, sessionID, userText, traceID string) (*skill_step.CLIResponse, error) {
	if a.isAPIOrLocalAdapter(adapterID) {
		return a.SendInspectorAPIMessage(adapterID, sessionID, userText, traceID)
	}
	return a.SendInspectorMessage(adapterID, sessionID, userText, traceID)
}

// SendInspectorAPIMessage is the top interaction lane for API/local adapters.
// It intentionally avoids the main composer action-chain so the top and bottom
// chat lanes stay separate even when both use the same local model.
func (a *App) SendInspectorAPIMessage(adapterID, sessionID, userText, traceID string) (*skill_step.CLIResponse, error) {
	log.Printf("SendInspectorAPIMessage: adapter=%q text_len=%d", adapterID, len(userText))
	debugtrace.Record("go.SendInspectorAPIMessage.enter", traceID, map[string]interface{}{
		"adapter_id": adapterID,
		"session_id": sessionID,
		"text_len":   len(userText),
	})

	cfg, err := a.loadLLMAPIAdapterConfig(adapterID)
	if err != nil {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return nil, err
	}

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
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return nil, fmt.Errorf("API model is missing for adapter %s", adapterID)
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// SEC-05 2a: Safe Client，policy 由 PolicyForLLMEndpoint 集中決定。
	client := urlsafe.NewSafeClient(urlsafe.PolicyForLLMEndpoint(cfg.ProviderID, cfg.BaseURL), "llm_inspector", 45*time.Second)
	prompt := a.buildInspectorPrompt(a.getActivePersona(), userText)
	reqBody := openAIChatRequest{
		Model: model,
		Messages: []openAIChatMessage{
			{Role: "user", Content: prompt},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	doRequest := func() (*http.Response, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, openAIChatCompletionsURL(cfg.BaseURL), bytes.NewReader(body))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
		req.Header.Set("Content-Type", "application/json")
		return client.Do(req)
	}

	res, err := doRequest()
	if err != nil && isLocalAdapter && strings.Contains(localAdapterInfo.Endpoint, ":11434") {
		debugtrace.Record("go.SendInspectorAPIMessage.local.wake_retry", traceID, map[string]interface{}{
			"adapter_id": adapterID,
			"error":      err.Error(),
		})
		if _, wakeErr := a.wakeOllamaAdapter(localAdapterInfo); wakeErr == nil {
			res, err = doRequest()
		} else {
			debugtrace.Record("go.SendInspectorAPIMessage.local.wake_error", traceID, map[string]interface{}{
				"adapter_id": adapterID,
				"error":      wakeErr.Error(),
			})
		}
	}
	if err != nil {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		debugtrace.Record("go.SendInspectorAPIMessage.http.error", traceID, map[string]interface{}{"error": err.Error()})
		if isLocalAdapter {
			return &skill_step.CLIResponse{Error: localAdapterReconnectHint(localAdapterInfo, err.Error())}, nil
		}
		return &skill_step.CLIResponse{Error: err.Error()}, nil
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024))
	if err != nil {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return &skill_step.CLIResponse{Error: err.Error()}, nil
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
			msg = localAdapterReconnectHint(localAdapterInfo, fmt.Sprintf("API HTTP %d: %s", res.StatusCode, msg))
		} else {
			msg = fmt.Sprintf("API HTTP %d: %s", res.StatusCode, msg)
		}
		return &skill_step.CLIResponse{Error: msg}, nil
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return &skill_step.CLIResponse{Error: parsed.Error.Message}, nil
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusDegraded)
		return &skill_step.CLIResponse{Error: "API response did not include assistant content"}, nil
	}

	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	resp := &skill_step.CLIResponse{Text: text, AdapterID: adapterID}
	a.finishInspectorReply(adapterID, userText, resp)
	a.setAdapterRuntimeStatus(adapterID, adapter_registry.StatusOnline)
	debugtrace.Record("go.SendInspectorAPIMessage.response", traceID, map[string]interface{}{
		"text":        resp.Text,
		"local_model": isLocalAdapter,
	})
	return resp, nil
}

func (a *App) finishInspectorReply(adapterID, userText string, resp *skill_step.CLIResponse) {
	if resp == nil {
		return
	}
	resp.Text = cleanInspectorReply(resp.Text)
	if resp.AdapterID == "" {
		resp.AdapterID = adapterID
	}
	if resp.Text == "" {
		return
	}
	a.appendInspectorHistory("user", userText)
	a.appendInspectorHistory("assistant", resp.Text)
	// Top replies only update the status rail; main conversation storage stays untouched.
	if a.statusRail != nil {
		view := a.statusRail.SetText(resp.Text)
		if a.eventBus != nil {
			a.eventBus.Emit(eventbus.EventStatusRailUpdated, view)
		}
	}
}

func cleanInspectorReply(text string) string {
	cleaned := strings.TrimSpace(text)
	for {
		next := stripInspectorInternalSuffix(cleaned)
		if next == cleaned {
			return cleaned
		}
		cleaned = next
	}
}

func stripInspectorInternalSuffix(text string) string {
	cleaned := strings.TrimSpace(text)
	lower := strings.ToLower(cleaned)
	for _, suffix := range []string{"_top", " lane=top", " lane: top", " [top]", " (top)"} {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimSpace(cleaned[:len(cleaned)-len(suffix)])
		}
	}
	return cleaned
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

// replyLanguageField turns the user's Role Language setting into the compact
// "語言=..." directive injected into the persona system prompt. Picking a
// specific language (e.g. Japanese) instructs the model to always reply in it;
// Auto / unknown follows whatever language the user wrote in.
func (a *App) replyLanguageField() string {
	role := ""
	if a.uiSettingsService != nil {
		role = strings.TrimSpace(a.uiSettingsService.Get().RoleLanguage)
	}
	switch role {
	case "中", "中文", "繁中", "繁體中文", "Traditional Chinese", "Chinese", "zh-TW", "zh":
		return "語言=繁體中文（請一律以繁體中文回覆）"
	case "en", "英文", "English":
		return "語言=English（always reply in English）"
	case "ja", "日文", "日本語", "Japanese":
		return "語言=日本語（必ず日本語で返信してください）"
	case "pt", "pt-PT", "Português", "葡萄牙文":
		return "語言=Português de Portugal（responda sempre em português）"
	case "es", "Español", "西班牙文":
		return "語言=Español（responde siempre en español）"
	case "th", "ไทย", "泰文":
		return "語言=ไทย（ตอบกลับเป็นภาษาไทยเสมอ）"
	default:
		// Auto / 自動 / unrecognised: mirror the user's input language.
		return "語言=與使用者輸入語言一致"
	}
}

// buildSharedPersonaPrompt keeps persona data compact; user-written persona
// fields such as "請簡短回答" remain global rules for both lanes.
func (a *App) buildSharedPersonaPrompt(persona settings.Persona) string {
	name := strings.TrimSpace(persona.Name)
	if name == "" {
		name = "憂樂傻酷"
	}

	fields := []string{"角色=" + name, a.replyLanguageField()}
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
	sb.WriteString("這是短互動回覆；只使用本區短歷史；直接回答，不輸出系統欄位、通道名稱或標記；三句內。\n")

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
	if adapter_registry.IsOllamaExecutablePath(expandUserPath(url)) {
		return frontendDTO(external_link.PreviewResult{
			URL:      expandUserPath(url),
			LinkType: external_link.LinkUnsupported,
			Valid:    false,
			Reason:   "Ollama 程式本體不能當 CLI adapter；請貼包含 blobs / manifests 的模型庫資料夾，例如 %%%USERPROFILE%%%\\.ollama\\models。",
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
	if adapter_registry.IsOllamaExecutablePath(expandUserPath(url)) {
		return nil, fmt.Errorf("ollama.exe 不能當 CLI adapter；請貼 Ollama 模型庫資料夾，例如 %%USERPROFILE%%\\.ollama\\models")
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
		})
	}
	return files, nil
}

// maybeEmitConfigMissingOnStartup 在 startup 末尾呼叫；只在「有 reference 檔 + 未設定 + 未跳過」三條件全成立時 emit 一次。
// 給 ensureReferenceVectorIndexes 一點時間先跑（等 1.5 秒），讓 disk → JSON → meta 流程穩定後再彈 modal。
func (a *App) maybeEmitConfigMissingOnStartup() {
	time.Sleep(1500 * time.Millisecond)
	if a == nil || a.settingsService == nil || a.eventBus == nil {
		return
	}
	cfg := a.settingsService.EmbeddingConfig()
	if cfg.ProviderID != "" || cfg.PickerDismissed {
		return
	}
	refDir := filepath.Join(appDataRoot(), "data", "references", "files")
	entries, err := os.ReadDir(refDir)
	if err != nil || len(entries) == 0 {
		return
	}
	nonDot := 0
	for _, e := range entries {
		if e.IsDir() || len(e.Name()) == 0 || e.Name()[0] == '.' {
			continue
		}
		nonDot++
	}
	if nonDot == 0 {
		return
	}
	a.eventBus.Emit("embedding:config_missing", map[string]any{
		"displayName": fmt.Sprintf("startup (%d 個歷史引用)", nonDot),
	})
}

func (a *App) ImportReferenceFile(sourcePath string) (ReferenceFile, error) {
	referenceDir := filepath.Join(appDataRoot(), "data", "references", "files")
	ref, err := importReferenceFileToDir(sourcePath, referenceDir)
	if err != nil {
		return ReferenceFile{}, err
	}
	if err := a.indexReferenceFileIfNeeded(ref.Path, referenceVectorsDir(), a.currentVectorizer()); err != nil {
		debugtrace.Record("reference_import.index_error", "", map[string]interface{}{
			"name":  ref.Name,
			"error": err.Error(),
		})
	}
	a.maybeEmitConfigMissing(ref.Name)
	return ref, nil
}

func importReferenceFileToDir(sourcePath, referenceDir string) (ReferenceFile, error) {
	if sourcePath == "" {
		return ReferenceFile{}, fmt.Errorf("reference: source path is empty")
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return ReferenceFile{}, err
	}
	if info.IsDir() {
		return ReferenceFile{}, fmt.Errorf("reference: folders are not supported")
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return ReferenceFile{}, err
	}
	defer source.Close()

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
		_ = target.Close()
		_ = os.Remove(targetPath)
		return ReferenceFile{}, err
	}
	return ReferenceFile{Name: baseName, Path: targetPath, Source: "library", Status: "ready"}, nil
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
	pending, err := a.packageService.ConfirmInstall(importID)
	if err != nil {
		return err
	}
	a.registerPackageTool(pending)
	a.eventBus.Emit("tools:list_changed", nil)
	return nil
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

// ---------------------------------------------------------------------------
// #55 — Skill Context Orchestration Wails Bindings
// ---------------------------------------------------------------------------

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

func runCLIHealthProbe(ctx context.Context, cliPath string) error {
	for _, args := range [][]string{{"--version"}, {"--help"}} {
		cmd := executil.CommandContext(ctx, cliPath, args...)
		out, err := cmd.CombinedOutput()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err == nil {
			return nil
		}
		if len(out) > 0 && strings.Contains(strings.ToLower(string(out)), "usage:") {
			return nil
		}
	}
	return fmt.Errorf("CLI did not respond to --version/--help")
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

// adjustNativeStepToWindowRect 將「錄製當下相對視窗的點」按比例映射到視窗
// 目前的 rect（兩者皆為螢幕 point 座標空間）。
func adjustNativeStepToWindowRect(step visual_learning.LearningReplayStep, current visual_learning.PixelBBox) visual_learning.LearningReplayStep {
	recorded := step.WindowRect
	if recorded.W <= 0 || recorded.H <= 0 || current.W <= 0 || current.H <= 0 {
		return step
	}
	if recorded == current {
		return step
	}
	relX := float64(step.X-recorded.X) / float64(recorded.W)
	relY := float64(step.Y-recorded.Y) / float64(recorded.H)
	if relX < 0 || relX > 1 || relY < 0 || relY > 1 {
		return step
	}
	step.X = current.X + int(math.Round(relX*float64(current.W)))
	step.Y = current.Y + int(math.Round(relY*float64(current.H)))
	return step
}

func debugInfoPathForImage(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return path + ".json"
	}
	return strings.TrimSuffix(path, ext) + ".json"
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

// GetYOLODetailedStatus 回傳 YOLO detector 的詳細診斷狀態。
func (a *App) GetYOLODetailedStatus() visual_learning.InferenceStatus {
	return a.yoloDetector.Status()
}

// ProposeRegions 執行 OpenCV 區域提案（degraded 模式回傳空列表）。
func (a *App) ProposeRegions(imageData []byte, width, height int) interface{} {
	return frontendDTO(a.opencvPipeline.Propose(imageData, width, height))
}

// DetectRegions 執行 YOLO 區域偵測（degraded 模式回傳空列表）。
func (a *App) DetectRegions(imageData []byte, width, height int) visual_learning.DetectorResult {
	return a.yoloDetector.Detect(imageData, width, height)
}

// GetNativeOCRStatus reports the platform OCR capability used only as an
// optional visual-anchor hint. macOS uses Apple Vision; Windows is disabled
// until a package-identity/MSIX path is introduced for Windows.Media.Ocr.
func (a *App) GetNativeOCRStatus() visual_learning.OCRStatus {
	return visual_learning.NewNativeOCRProvider().Status()
}

// RecognizeNativeOCR runs platform-native OCR on a small cropped UI image.
// The returned text must not be used as the sole authority for click replay.
func (a *App) RecognizeNativeOCR(imageData []byte) ([]visual_learning.OCRResult, error) {
	return visual_learning.NewNativeOCRProvider().Recognize(imageData)
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
