import React, {useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import './tailwind.css';
import {
  AcknowledgePendingItem,
  AcknowledgePendingItemWithConfirmation,
  ActivateTool,
  ApplyUIStyleDiff,
  BuildSkillContext,
  ConfirmAndExecuteSkillExecution,
  ClearSkillContext,
  ClearWebSearchConfig,
  ConfirmPackageInstall,
  ConfirmSkillArchive,
  DisableContextualRiskOverride,
  EnableContextualRiskOverride,
  EnableTrustedSessionScope,
  EnableWorkflowTrustForHours,
  GetAppSessionID,
  GetBrowserPreference,
  GetConsoleState,
  GetDeviceTrustProfile,
  GetHookSummary,
  // Keep: debug trace monitor bridge. Used by postDebugTrace() even when the
  // local monitor port is not currently listening.
  GetMonitorLinks,
  // SEC-05 2b: 開外部連結唯一入口（Go 端做 scheme/metadata 檢查，loopback 放行）
  OpenExternalURL,
  // SEC-06: URL provenance — 記錄 LLM 擷取的 URL（閉合洗白防護）
  RegisterURLOccurrence,
  GetNewSubagentCandidates,
  GetPendingDigest,
  GetPendingTagPatches,
  GetRecentSkillInjections,
  GetSafariRuntimeNotice,
  GetSettingsState,
  GetToolRegistryPatchProposals,
  GetToolVisibility,
  GetUISettings,
  ListArchivedSkills,
  ListPendingPackages,
  // #I-1002: Review Archive binding
  ListReviewArchive,
  ListTools,
  PollStatusRail,
  PreparePackageInstallPayload,
  PromoteDraftToPending,
  ListExternalLinksByType,
  PreviewExternalLink,
  RegisterExternalLink,
  RejectPackageInstall,
  ReorderPersonas,
  ResolveSkillForAction,
  ExecuteSkillMessage,
  RestoreUIDefaults,
  SavePanelSettings,
  SavePersona,
  FinalizeNativeGoProgramAuthoringExport,
  FinalizeNativePersonaExport,
  GetGoProgramAuthoringDetail,
  ListGoProgramAuthoringCatalog,
  NativeDragExportPersonaHandler,
  NativeDragExportGoProgramAuthoring,
  ScanSkillFolder,
  SetBrowserPreference,
  SetToolPreference,
  SetTrustDomAndClick,
  ExecuteNativeLearningReplayStep,
  RecordLearningMouseEvent,
  RecordStepTrace,
  StartDraftSandbox,
  StartHookRun,
  StopDraftSandbox,
  MarkSkillFirstUseExplained,
  SendCLIMessage,
  SendAPIMessage,
  SendTopInteractionMessage,
  SetActiveConversationAgent,
  ClearInspectorHistory,
  RestartSidecar,
  // 遺留能力 #1–#8: Wails 架構遷移 bindings
  ListAvailableAdapters,
  SetAdapterStatus,
  ListOpenReviewCards,
  ResolveReviewCard,
  HasBlockingReviewCard,
  CreateDestructiveReviewCard,
  ResolveAndExecuteDestructiveReviewCard,
  GetMemoryHealth,
  GetConfigPublic,
  GetDegradedState,
  EnterDegradedMode,
  ExitDegradedMode,
  IsDegradedBlocked,
  GetOnboardingState,
  IsReadOnlyMode,
  CompleteOnboardingStep,
  FinishOnboarding,
  GoBackOnboarding,
  AutoDetectCLI,
  RegisterCustomCLI,
  RegisterLLMAPIAdapter,
  ConfirmRegisterLLMAPIAdapter,
  RenameAdapter,
  UnregisterAdapter,
  EnableDetectedCLI,
  ScanLocalModels,
  EnableLocalModel,
  WakeLocalAdapter,
  GetEmbeddingConfig,
  GetAdapterModelChoices,
  SetAdapterModelChoice,
  ListAdapterModelOptions,
  // ── v3.6 Review / Risk 接線 ──
  DualStepConfirmStep1,
  InvalidateDualStep,
  ListOpenLightweightCards,
  ResolveLightweightCard,
  AcknowledgeStatusRail,
  // ── v3.6 Source Trust 接線 ──
  ClassifySource,
  AddSourceToAllowlist,
  RemoveAllowlistEntry,
  RenewAllowlistEntry,
  GetProjectAllowlist,
  GetHighImpactDomains,
  ListThreatRecords,
  // ── v3.6 Avatar 接線 ──
  GetCurrentAvatar,
  ListAvatarPresets,
  SaveStaticAvatar,
  DeleteStaticAvatar,
  SetAvatarProvider,
  GetCredentialMigrationStatus,
  ConfirmCredentialMigration,
  DisableCredentialMigration,
  RenderPixelAvatarPreview,
  // ── v3.6 LLM Context + Memory 接線 ──
  BuildLLMContext,
  EscapeExternalTokens,
  ValidateMemoryItem,
  GetMemoryPipelineState,
  AppendTalkEntryForAgent,
  GetTalkMessagesForAgent,
  DeleteTalkMessageForAgent,
  // ── 任務進度 / DAG Runtime v1 接線 ──
  StartTaskProgress,
  CancelActiveTaskProgress,
  ApproveTaskStep,
  GetDAGRunDebug,
  // ── v3.6 Visual Learning 接線 ──
  StartLearningMode,
  StopLearningMode,
  GenerateLearningRunMetadata,
  IsLearningModeActive,
  GetActiveLearningRun,
  GetLastLearningReplayPlan,
  GetLearningReplayPlan,
  ListLearningReplayCatalog,
  SearchLearningOperations,
  GetPendingCandidateCount,
  HasBlockingVLReview,
  // ── v3.6 Stop Recovery 接線 ──
  ResolveStopRecoveryCard,
  ListOpenStopRecoveryCards,
  HasOpenStopRecoveryCard,
  // ── v3.6 Project Lifecycle 接線 ──
  PurgeProject,
  PurgeBoundaryDir,
  ListPurgeManifests,
  // ── v3.6.2 W3A Media Provenance（§9A）接線 ──
  // 對應 Go app.go 的 W3A Wails binding 方法，
  // 涵蓋媒體驗證→污染偵測→匯入匯出→傳輸引導→信任清單管理。
  GetMediaW3AInfo,
  DetectModelPollution,
  ExportMediaWithSidecar,
  ImportMediaVerify,
  GetW3ATransferGuidance,
  ListW3ATrustedDevelopers,
  AddW3ATrustedDeveloper,
  // ── v3.6.3 Remote Bridge Communication（§12A）接線 ──
  // 對應 Go app.go 的 9 個 Wails binding 方法，
  // 涵蓋通道偵測→測試→註冊→啟用→模式切換→移除→稽核查詢完整生命週期。
  DetectRemoteBridgeChannel,
  TestRemoteBridgeConnection,
  RegisterRemoteBridgeChannel,
  RegisterRemoteBridgeChannelWithMode,
  ActivateRemoteBridgeChannel,
  SetRemoteBridgePrimaryChannel,
  RunSummarizationNow,
  DispatchRemoteBridgeAsync,
  DeactivateRemoteBridgeChannel,
  SwitchRemoteBridgeMode,
  ListRemoteBridgeChannels,
  RenameRemoteBridgeChannel,
  RemoveRemoteBridgeChannel,
  GetRemoteBridgeInboundEndpoint,
  SaveRemoteBridgeInboundSecret,
  ListRemoteBridgeInboundAdapters,
  // ── v3.6 雜項接線 ──
  FinalizeNativeReferenceFileExport,
  ImportReferenceFile,
  ListReferenceFiles,
  ImportVideoFile,
  ListVideoFiles,
  NativeDragExportReferenceFile,
  StopSidecar,
  // ── v3.6.4 Readiness Gate UI Interaction Layer 接線 ──
  // 對應 Go app.go 的 6 個 Wails binding 方法，
  // 涵蓋 Gate 狀態查詢→更新→追蹤記錄→候選選擇→候選清除完整生命週期。
  GetReadinessGateState,
  // UpdateReadinessGateState — 後端 pipeline 內部呼叫，前端 import-only 預留
  // RecordReadinessTrace — 後端自動記錄，前端 import-only 預留
  // ListReadinessTraces — 預留給未來開發者除錯面板
  SelectFloatingCandidate,
  DismissFloatingCandidates,
  // §30: 關閉視窗流程
  ConfirmClose,
  CreateSubagent,
  RenameSubagent,
  // ── Pixel Avatar Pack 切換 ──
  SetPersonaPixelAvatarPack,
  ListPixelAvatarPacks,
  // §31: Sub Export/Import + Tab Order
  ExportSubHandler,
  FinalizeNativeSubExport,
  NativeDragExportSubHandler,
  ImportSubHandler,
  PreviewSubPackage,
  ResolveImportToolConflicts,
  GetTabOrder,
  GetSubExportCapabilities,
  ReorderAdapters,
  ReorderTabs,
  SelectSubExportDirectory,
  GetSummaryModelSettings,
  SaveSummaryModelSettings,
  ScanLocalSummaryModels,
  GetVoiceSettings,
  GetWebSearchConfig,
  SaveVoiceSettings,
  SaveWebSearchConfig,
  InstallVoiceBaseModel,
  RemoveVoiceBaseModel,
  ClearVoiceDebug,
  TranscribeVoiceWAV,
  RouteVoiceCommand,
} from '../wailsjs/go/main/App';
import {OnFileDrop, OnFileDropOff, EventsOn, BrowserOpenURL, Quit, ClipboardGetText, ClipboardSetText} from '../wailsjs/runtime/runtime';

// SEC-05 2b: 所有外部連結一律走 Go 端 OpenExternalURL 檢查
// （僅 http/https、擋 metadata、loopback 放行，monitor 頁不受影響）。
// catch 兩層：binding 缺失時（測試環境）退回 BrowserOpenURL；正式環境 binding 必存在。
const openExternal = (url) => {
  try {
    OpenExternalURL(url).catch((err) => console.error('openExternal blocked:', err));
  } catch {
    BrowserOpenURL(url);
  }
};
import DocumentReviewCard from './components/DocumentReviewCard';
import VisualLearningPanel from './components/VisualLearningPanel';
import EmbeddingPickerModal from './components/EmbeddingPickerModal';
import useI18n, { t as _t } from './locales/useI18n';

const taskProgressDebugEnabled = typeof window !== 'undefined'
  && (new URLSearchParams(window.location.search).has('taskDebug')
    || window.localStorage?.getItem('task_progress_debug') === '1');

const fallbackState = {
  /* i18n: fallbackState */ greeting: _t('greeting.hello'),
  statusRail: {text: _t('greeting.hello'), layer: 'L1', degraded: true, lockedCount: 0},
  // #I-806: adapter 清單由 Sidecar IPC 傳入，經 adapter_registry 白名單審查後更新
  // 預設空陣列：未接 CLI 時不顯示任何 adapter 按鈕
  adapters: [],
  haoras: ['主haㄌer'],
  messages: [],
};

/* i18n: greeting pool */
const getGreetingRotationOptions = () => [
  {text: _t('greeting.pool.0'), expression: 'idle'},
  {text: _t('greeting.pool.1'), expression: 'idle'},
  {text: _t('greeting.pool.2'), expression: 'idle'},
  {text: _t('greeting.pool.3'), expression: 'speechless'},
  {text: _t('greeting.pool.4'), expression: 'idle', rare: true},
  {text: _t('greeting.pool.5'), expression: 'speechless'},
  {text: _t('greeting.pool.6'), expression: 'happy'},
  {text: _t('greeting.pool.7'), expression: 'sleepy'},
  {text: _t('greeting.pool.8'), expression: 'idle'},
  {text: _t('greeting.pool.9'), expression: 'idle'},
];

function pickRotatingGreeting(currentText) {
  const options = getGreetingRotationOptions();
  const regular = options.filter((item) => !item.rare);
  const rare = options.filter((item) => item.rare);
  const basePool = rare.length > 0 && Math.random() < 0.05 ? rare : regular;
  let pool = basePool.filter((item) => item.text !== currentText);
  if (pool.length === 0) {
    pool = regular.filter((item) => item.text !== currentText);
  }
  if (pool.length === 0) {
    pool = getGreetingRotationOptions();
  }
  return pool[Math.floor(Math.random() * pool.length)];
}

// Panel-style identity is a STABLE key (language-independent). Labels are
// resolved live via _t()/styleLabel() so switching language re-translates the
// buttons (and never resets the active selection).
const STYLE_KEYS = ['default', 'passiveWhite', 'pinkBetrayal', 'forgiveMeGreen', 'defeatBlue'];
const STYLE_KEY_I18N = {
  default: 'style.default',
  passiveWhite: 'style.options.passiveWhite',
  pinkBetrayal: 'style.options.pinkBetrayal',
  forgiveMeGreen: 'style.options.forgiveMeGreen',
  defeatBlue: 'style.options.defeatBlue',
};
const STYLE_KEY_THEME = {
  default: 'onanegiku',
  passiveWhite: 'white',
  pinkBetrayal: 'pink-black',
  forgiveMeGreen: 'green',
  defeatBlue: 'blue',
};
const defaultPanelStyle = 'default';
const styleOptions = STYLE_KEYS;
// Panel font scale is a UI-wide preference; CSS consumes it through --ui-font-scale.
const fontScaleOptions = ['50%', '80%', '90%', '100%', '110%', '120%'];
const styleOptionOrderStorageKey = 'ai-console.style-option-order';

function floatTo16BitPCM(view, offset, input) {
  for (let i = 0; i < input.length; i += 1, offset += 2) {
    const sample = Math.max(-1, Math.min(1, input[i]));
    view.setInt16(offset, sample < 0 ? sample * 0x8000 : sample * 0x7fff, true);
  }
}

function writeWavString(view, offset, value) {
  for (let i = 0; i < value.length; i += 1) {
    view.setUint8(offset + i, value.charCodeAt(i));
  }
}

function encodeWav(chunks, sampleRate) {
  const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
  const samples = new Float32Array(totalLength);
  let offset = 0;
  chunks.forEach((chunk) => {
    samples.set(chunk, offset);
    offset += chunk.length;
  });
  const buffer = new ArrayBuffer(44 + samples.length * 2);
  const view = new DataView(buffer);
  writeWavString(view, 0, 'RIFF');
  view.setUint32(4, 36 + samples.length * 2, true);
  writeWavString(view, 8, 'WAVE');
  writeWavString(view, 12, 'fmt ');
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true);
  view.setUint16(22, 1, true);
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * 2, true);
  view.setUint16(32, 2, true);
  view.setUint16(34, 16, true);
  writeWavString(view, 36, 'data');
  view.setUint32(40, samples.length * 2, true);
  floatTo16BitPCM(view, 44, samples);
  return new Blob([view], {type: 'audio/wav'});
}

function blobToBase64(blob) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onloadend = () => resolve(String(reader.result || '').split(',')[1] || '');
    reader.onerror = reject;
    reader.readAsDataURL(blob);
  });
}

function waitForVoiceChunks(recorder, timeoutMs = 700) {
  if (recorder.chunks.length > 0) return Promise.resolve();
  return new Promise((resolve) => {
    const startedAt = Date.now();
    const timer = setInterval(() => {
      if (recorder.chunks.length > 0 || Date.now() - startedAt >= timeoutMs) {
        clearInterval(timer);
        resolve();
      }
    }, 50);
  });
}

const MIN_VOICE_RECORDING_MS = 300;
const MAX_VOICE_RECORDING_MS = 120000; // 120 秒錄音上限
const VOICE_BG_THRESHOLD_MS = 30000;   // >30 秒走背景轉錄

function mergeDraftText(current, addition) {
  const left = String(current || '').trimEnd();
  const right = String(addition || '').trim();
  if (!left) return right;
  if (!right) return left;
  return `${left}\n${right}`;
}

function adapterField(adapter, camel, pascal, fallback = '') {
  if (!adapter) return fallback;
  const value = adapter[camel] ?? adapter[pascal];
  return value == null ? fallback : value;
}

function adapterKey(adapter) {
  return adapterField(adapter, 'id', 'ID', '') || adapterField(adapter, 'name', 'Name', '');
}

function normalizeAdapterDTO(adapter) {
  if (!adapter) return adapter;
  return {
    ...adapter,
    id: adapterField(adapter, 'id', 'ID', ''),
    name: adapterField(adapter, 'name', 'Name', ''),
    icon: adapterField(adapter, 'icon', 'Icon', ''),
    path: adapterField(adapter, 'path', 'Path', ''),
    endpoint: adapterField(adapter, 'endpoint', 'Endpoint', ''),
    model: adapterField(adapter, 'model', 'Model', ''),
    kind: adapterField(adapter, 'kind', 'Kind', ''),
    status: adapterField(adapter, 'status', 'Status', 'offline'),
  };
}

function isSubagentAdapter(adapter) {
  return String(adapterField(adapter, 'kind', 'Kind', '')).toLowerCase() === 'sub';
}

function orderSubagentTabsByTabOrder(tabs, tabOrder) {
  if (!Array.isArray(tabs)) return [];
  const rank = new Map((tabOrder?.sub_order || []).map((id, index) => [id, index]));
  return [...tabs].sort((a, b) => {
    const aKey = adapterKey(a);
    const bKey = adapterKey(b);
    const aRank = rank.has(aKey) ? rank.get(aKey) : Number.MAX_SAFE_INTEGER;
    const bRank = rank.has(bKey) ? rank.get(bKey) : Number.MAX_SAFE_INTEGER;
    if (aRank !== bRank) return aRank - bRank;
    return String(adapterField(a, 'name', 'Name', '') || adapterKey(a)).localeCompare(String(adapterField(b, 'name', 'Name', '') || adapterKey(b)), 'zh-Hant');
  });
}

function splitAvailableAdapters(adapters, tabOrder) {
  const items = Array.isArray(adapters) ? adapters.map(normalizeAdapterDTO) : [];
  return {
    cliAdapters: items.filter((item) => !isSubagentAdapter(item)),
    subagentTabs: orderSubagentTabsByTabOrder(items.filter(isSubagentAdapter), tabOrder),
  };
}

function reorderItemsByKeys(items, orderKeys) {
  const source = Array.isArray(items) ? items : [];
  const order = Array.isArray(orderKeys) ? orderKeys : [];
  const byKey = new Map(source.map((item) => [adapterKey(item), item]));
  const used = new Set();
  const next = [];
  for (const key of order) {
    if (byKey.has(key) && !used.has(key)) {
      next.push(byKey.get(key));
      used.add(key);
    }
  }
  for (const item of source) {
    const key = adapterKey(item);
    if (!used.has(key)) next.push(item);
  }
  return next;
}

const learningDigestStorageKey = 'ai-console.learning-digest-ready';
const learningIdleDelayMs = 20 * 60 * 1000;
const dagTaskPattern = /(幫我|請|查|搜尋|整理|建立|執行|打開|開啟|分析|寫|寄|下載|安裝|錄製|學習|流程|天氣|地圖)/;
const dagHighRiskPattern = /(刪除|安裝|寄|付款|修改系統|開啟地圖|地圖|錄製|螢幕)/;
const internalControlPrefixPattern = /^[ㄅ-ㄩ]{3}\s*/;

function stripInternalControlPrefix(text) {
  return String(text || '').trim().replace(/^ㄌㄤㄤ/, '').replace(internalControlPrefixPattern, '').trim();
}

function stripInternalControlDraft(draft) {
  const raw = String(draft || '').trim();
  if (raw.startsWith('input:')) {
    return `input:${stripInternalControlPrefix(raw.slice('input:'.length))}`;
  }
  return stripInternalControlPrefix(raw);
}

// ── v3.6.4 Readiness Gate UI Interaction Layer 常數與 fallback ──
// 長按確認所需時間（毫秒），對應 §11.3 轉蛋動畫觸發門檻
const LONG_PRESS_DURATION_MS = 800;
// 轉蛋動畫各階段時間（毫秒）
const GACHA_COLLAPSE_MS = 300;
const GACHA_PULSE_MS = 500;
const GACHA_PARTICLE_MS = 600;
// 粒子爆發方向角度（8 個粒子均分 360 度）
const PARTICLE_ANGLES = Array.from({length: 8}, (_, i) => i * 45);
// 預設 Readiness Gate 狀態（前端 fallback，對應 Go ReadinessGateState 結構）
const fallbackReadinessGate = {
  risk_tier: 'none',        // none | normal | medium | high
  missing_slots: [],
  floating_candidates: [],
  clarification_count: 0,
  max_clarifications: 2,
  retrieval_scanning: false,
  retrieval_sources: [],
  impact_explanation: '',
  low_confidence_output: false,
  assumption_used: false,
  auto_output_allowed: false,
};

const fallbackSettings = {
  panel: {
    /* i18n: settings defaults */ panelLanguage: _t('settings.langZhTW'),
    roleLanguage: _t('settings.roleLangAuto'),
    fontPreset: _t('settings.fontDefault'),
    fontScale: '100%',
    panelStyle: defaultPanelStyle,
  },
  activePersonaId: 'persona-a',
  personas: [
    {id: 'persona-a', name: _t('persona.defaultNameA'), icon: '♙', avatarUrl: '', identity: _t('persona.defaultIdentityA'), replyStrategy: '', roleStrength: '20%', personality: '', scenario: '', description: ''},
    {id: 'persona-b', name: _t('persona.defaultNameB'), icon: '♚', avatarUrl: '', identity: _t('persona.defaultIdentityB'), replyStrategy: '', roleStrength: '20%', personality: '', scenario: '', description: ''},
    {id: 'persona-c', name: _t('persona.defaultNameC'), icon: '★', avatarUrl: '', identity: _t('persona.defaultIdentityC'), replyStrategy: '', roleStrength: '20%', personality: '', scenario: '', description: ''},
  ],
};

const lockedPersonaId = 'persona-a';
/* i18n: persona locked */ const lockedPersonaName = _t('persona.lockedName');
/* i18n: reply strategy presets */
const getReplyStrategyPresets = () => [
  {id: 'concise', label: _t('strategy.save.label'), prompt: _t('strategy.save.prompt')},
  {id: 'reflective_question', label: _t('strategy.counter.label'), prompt: _t('strategy.counter.prompt')},
  {id: 'suggestive', label: _t('strategy.suggest.label'), prompt: _t('strategy.suggest.prompt')},
  {id: 'teacher_question', label: _t('strategy.question.label'), prompt: _t('strategy.question.prompt')},
  {id: 'confirm_before_action', label: _t('strategy.execute.label'), prompt: _t('strategy.execute.prompt')},
  {id: 'decision_tree', label: _t('strategy.analyze.label'), prompt: _t('strategy.analyze.prompt')},
  {id: 'companion', label: _t('strategy.companion.label'), prompt: _t('strategy.companion.prompt')},
  {id: 'creative', label: _t('strategy.creative.label'), prompt: _t('strategy.creative.prompt')},
];
const avatarStateOptions = ['idle', 'thinking', 'working', 'happy', 'warning', 'blocked', 'sleepy', 'sad', 'speechless'];
/* i18n: avatar state labels */
const getAvatarStateLabels = () => ({
  idle: _t('avatar.idle'),
  thinking: _t('avatar.thinking'),
  working: _t('avatar.working'),
  happy: _t('avatar.happy'),
  warning: _t('avatar.warning'),
  blocked: _t('avatar.blocked'),
  sleepy: _t('avatar.sleepy'),
  sad: _t('avatar.sad'),
  speechless: _t('avatar.speechless'),
});

/* i18n: tool list */
const getFallbackTools = () => [
  {id: 'tool-entrance', icon: '⌕', title: _t('tool.useTools.title'), detail: _t('tool.useTools.detail'), enabled: true},
  {id: 'reference-link', icon: '▤', title: _t('tool.citeLink.title'), detail: _t('tool.citeLink.detail'), enabled: true},
  {id: 'doc-entrance', icon: '▤', title: _t('tool.citeFile.title'), detail: _t('tool.citeFile.detail'), enabled: true},
  {id: 'external-link', icon: '↗', title: _t('tool.openExternal.title'), detail: _t('tool.openExternal.detail'), enabled: false},
  {id: 'gmail', icon: '✉', title: _t('tool.gmail.title'), detail: _t('tool.gmail.detail'), enabled: false},
  {id: 'flow-mail-digest', icon: '◇', title: _t('tool.summarizeMail.title'), detail: _t('tool.summarizeMail.detail'), enabled: true},
  {id: 'package-import', icon: '□', title: _t('tool.importPack.title'), detail: _t('tool.importPack.detail'), enabled: true},
];

/* i18n: tool tab options */
const getToolTabOptions = () => [
  {id: 'external', label: _t('tool.tabExternal')},
  {id: 'flow', label: _t('tool.tabAutoflow')},
  {id: 'package', label: _t('tool.tabToolkit')},
];

const toolTabById = {
  'external-link': 'external',
  'reference-link': 'external',
  gmail: 'external',
  'doc-entrance': 'external',
  'flow-mail-digest': 'flow',
  'package-import': 'package',
};

function toolTabFor(tool = {}) {
  if (toolTabById[tool.id]) return toolTabById[tool.id];
  const fields = [tool.id, tool.kind, tool.target, tool.title, tool.detail]
    .map((value) => String(value || '').toLowerCase());
  const haystack = fields.join(' ');
  const kind = fields[1];
  if (kind === 'skill' || haystack.includes('skill') || haystack.includes('recording') || haystack.includes('replay')) {
    return 'flow';
  }
  if (kind === 'mcp' || kind === 'program' || kind === 'package' || kind === 'cli' || haystack.includes('mcp') || haystack.includes('program')) {
    return 'package';
  }
  if (kind === 'connector' || kind === 'external' || kind === 'api' || kind === 'local' || kind === 'website' || haystack.includes('http')) {
    return 'external';
  }
  return 'package';
}

const fallbackReviewState = {
  highRisk: {
    id: '',
    status: 'none',
    /* i18n: review */ title: _t('dag.highRiskConfirm'),
    action: '',
    skillId: '',
    summaryHash: '',
    permissionSummary: '',
    targetPaths: [],
    diff: [],
    expiresIn: '',
    source: '',
  },
  lowRiskAmbiguity: {
    id: '',
    dismissedUntil: '',
    selectedSkillId: '',
    candidates: [],
  },
  // I-1: Hook candidates (tag patches, subagent candidates, tool registry proposals)
  hookCandidates: {
    tagPatches: [],
    subagentCandidates: [],
    registryProposals: [],
  },
  // I-1: Pending Digest — 五分類映射為三區塊
  pendingDigest: null,
  // I-1: Package install queue
  pendingPackages: [],
  // I-2: Skill activity (最新 2 筆注入紀錄)
  skillInjections: [],
};

function createDagRunFromMessage(text) {
  const now = Date.now();
  const hasHighRiskStep = dagHighRiskPattern.test(text);
  const nodes = [
    {
      id: `dag-node-${now}-1`,
      /* i18n: DAG nodes */ title: _t('dag.nodeParseTask'),
      action: _t('dag.actionParse'),
      tool: 'main-agent',
      risk: 'low',
      status: 'queued',
    },
    {
      id: `dag-node-${now}-2`,
      title: _t('dag.nodeSelectTool'),
      action: _t('dag.actionSelect'),
      tool: 'skill-router',
      risk: 'medium',
      status: 'queued',
    },
    {
      id: `dag-node-${now}-3`,
      title: hasHighRiskStep ? _t('dag.nodeHighRisk') : _t('dag.nodeOutput'),
      action: hasHighRiskStep ? _t('dag.actionHighRisk') : _t('dag.actionOutput'),
      tool: hasHighRiskStep ? 'review-gate' : 'main-agent',
      risk: hasHighRiskStep ? 'high' : 'low',
      status: 'queued',
    },
  ];

  return {
    id: `dag-${now}`,
    outlineId: `outline-${now}`,
    hookRunId: '',
    title: text.length > 28 ? `${text.slice(0, 28)}...` : text,
    status: 'starting',
    createdAt: new Date(now).toISOString(),
    nodes,
    summaries: [],
    currentNodeId: nodes[0].id,
    summaryHash: '',
  };
}

function shouldCreateDagRun(text) {
  const raw = String(text || '').trim();
  if (!raw) return false;
  // Auto-DAG runs only after an explicit internal/LLM route. Natural language
  // must reach the LLM first so it can choose chat, search, saved operation, or DAG.
  return /^\/dag\b/i.test(raw) || /^#dag\b/i.test(raw);
}

function shouldHandleLearningShortcutBeforeLLM() {
  // Keep replay/catalog as LLM-routed tools. The frontend may execute the model's
  // directive later, but it should not classify natural language before the LLM.
  return false;
}

function getField(source, camelName, snakeName, fallback = '') {
  if (!source) return fallback;
  if (source[camelName] !== undefined && source[camelName] !== null) return source[camelName];
  if (source[snakeName] !== undefined && source[snakeName] !== null) return source[snakeName];
  const pascalName = `${camelName.slice(0, 1).toUpperCase()}${camelName.slice(1)}`;
  if (source[pascalName] !== undefined && source[pascalName] !== null) return source[pascalName];
  return fallback;
}

function normalizeTaskStatus(status) {
  if (status === 'succeeded') return 'completed';
  if (status === 'waiting_review') return 'waiting_review';
  return status || 'queued';
}

function isTaskProgressActive(run) {
  return ['starting', 'planning', 'running', 'waiting_review', 'blocked'].includes(run?.status);
}

function shouldKeepNewerTaskRun(current, incoming) {
  if (!current || !incoming || current.id !== incoming.id) return false;
  const incomingEarly = ['planning', 'running'].includes(incoming.status);
  const currentAdvanced = ['waiting_review', 'completed', 'cancelled', 'failed', 'interrupted'].includes(current.status);
  return incomingEarly && currentAdvanced;
}

function plannerClarificationFromError(error) {
  const raw = String(error?.message || error || '').replace(/^Error:\s*/i, '').trim();
  if (!raw) return '';
  const candidate = raw.replace(/^任務規劃失敗：\s*/, '').trim();
  const lower = candidate.toLowerCase();
  const hasQuestion = candidate.includes('?') || candidate.includes('？');
  const hasAsk = [
    '請提供', '請告訴', '請確認', '我需要知道', '需要知道', '以下幾點',
    '哪個檔案', '哪個文件', '什麼格式', '什麼內容', '更多資訊',
  ].some((needle) => candidate.includes(needle)) || [
    'please provide', 'please clarify', 'need to know', 'which file',
    'what format', 'more information', 'additional information',
  ].some((needle) => lower.includes(needle));
  const hasBlocker = [
    '無法', '不能', '資訊不足', '資料不足', '不清楚', '太模糊', '缺少',
    '沒有更多資訊', '沒有足夠資訊',
  ].some((needle) => candidate.includes(needle)) || [
    'cannot', "can't", 'unable', 'ambiguous', 'lack', 'not enough information',
  ].some((needle) => lower.includes(needle));
  return (hasAsk && hasBlocker) || (hasQuestion && hasAsk) ? candidate : '';
}

function mapBackendTaskRun(rawRun) {
  if (!rawRun) return null;
  const nodes = (rawRun.nodes || rawRun.Nodes || []).map((node) => {
    const status = normalizeTaskStatus(getField(node, 'status', 'status', 'queued'));
    const risk = getField(node, 'riskClass', 'risk_class', getField(node, 'risk', 'risk', 'low'));
    const modelRisk = getField(node, 'modelRiskClass', 'model_risk_class', risk);
    return {
      id: getField(node, 'id', 'id'),
      title: getField(node, 'title', 'title', getField(node, 'operation', 'operation', '任務步驟')),
      action: getField(node, 'action', 'action', getField(node, 'operation', 'operation', '')),
      actionCode: getField(node, 'actionCode', 'action_code', ''),
      tool: getField(node, 'executorType', 'executor_type', getField(node, 'tool', 'tool', '')),
      target: getField(node, 'target', 'target', ''),
      risk,
      modelRisk,
      status,
      reviewId: getField(node, 'reviewId', 'review_id', ''),
      startedAt: getField(node, 'startedAt', 'started_at', ''),
      endedAt: getField(node, 'completedAt', 'completed_at', getField(node, 'endedAt', 'ended_at', '')),
      resultSummary: getField(node, 'resultSummary', 'result_summary', ''),
      traceHash: getField(node, 'traceHash', 'trace_hash', ''),
      outputRef: getField(node, 'outputRef', 'output_ref', ''),
    };
  });
  const status = normalizeTaskStatus(getField(rawRun, 'status', 'status', 'starting'));
  return {
    id: getField(rawRun, 'id', 'id', ''),
    outlineId: getField(rawRun, 'outlineId', 'outline_id', ''),
    hookRunId: getField(rawRun, 'hookRunId', 'hook_run_id', ''),
    title: getField(rawRun, 'title', 'title', getField(rawRun, 'prompt', 'prompt', '任務進度')),
    status,
    createdAt: getField(rawRun, 'createdAt', 'created_at', ''),
    nodes,
    summaries: nodes
      .filter((node) => node.resultSummary)
      .map((node) => ({
        nodeId: node.id,
        title: node.title,
        status: node.status,
        risk: node.risk,
        text: node.resultSummary,
      })),
    currentNodeId: getField(rawRun, 'activeNodeId', 'active_node_id', ''),
    summaryHash: getField(rawRun, 'summaryHash', 'summary_hash', ''),
    interruptReason: getField(rawRun, 'interruptReason', 'interrupt_reason', ''),
  };
}

function formatNodeRiskLabel(node) {
  const finalRisk = node?.risk || 'unknown';
  const modelRisk = node?.modelRisk || finalRisk;
  if (modelRisk && modelRisk !== finalRisk) {
    return `系統判定 ${finalRisk} / 模型 ${modelRisk}`;
  }
  return finalRisk;
}

function formatDagTime(value) {
  if (!value) return '--:--:--';
  return new Date(value).toLocaleTimeString('zh-TW', {hour12: false});
}

/* i18n: DAG status labels */
function dagStatusLabel(status) {
  const labels = {
    starting: _t('dag.starting'),
    planning: '規劃中',
    running: _t('dag.running'),
    queued: _t('dag.queued'),
    completed: _t('dag.completed'),
    blocked: _t('dag.blocked'),
    waiting_review: '待確認',
    cancelled: '已取消',
    interrupted: '已中斷',
    skipped: '已略過',
    failed: _t('dag.failed'),
  };
  return labels[status] || status || _t('dag.standby');
}

const adapterMeta = {
  Claude: {className: 'adapter-lime', icon: 'C'},
  Codex: {className: 'adapter-pink', icon: '◎'},
  Gemini: {className: 'adapter-lime', icon: '✦'},
};

const maxPersonas = 16;

const personaAvatarUrls = {
  'persona-a': new URL('./assets/persona_avatars/persona-a.svg', import.meta.url).href,
  'persona-b': new URL('./assets/persona_avatars/persona-b.svg', import.meta.url).href,
  'persona-c': new URL('./assets/persona_avatars/persona-c.svg', import.meta.url).href,
  'persona-d': new URL('./assets/persona_avatars/persona-d.svg', import.meta.url).href,
};

const pixelAvatarRenderSize = 128;
const pixelAvatarRenderCache = new Map();
const autoAvatarOverrideStates = new Set(['blocked', 'warning', 'working']);
/* i18n: composer pending */ const composerPendingInitialText = _t('composer.pendingInitial');
const composerPendingSlowText = _t('composer.pendingSlow');
const composerPendingVerySlowText = _t('composer.pendingVerySlow');
const composerPendingMarker = '\u2063pending:';

function makeComposerPendingMessage(traceId, text = composerPendingInitialText) {
  return `Ai:${text}${composerPendingMarker}${traceId}`;
}

function stripComposerPendingMarker(message) {
  const markerIndex = String(message || '').indexOf(composerPendingMarker);
  return markerIndex >= 0 ? message.slice(0, markerIndex) : message;
}

function replaceComposerPendingMessage(messages, traceId, replacement) {
  const needle = `${composerPendingMarker}${traceId}`;
  const index = messages.findIndex((message) => String(message || '').includes(needle));
  if (index < 0) return [...messages, replacement];
  const next = [...messages];
  next[index] = replacement;
  return next;
}

const visualLearningInteractiveSelector = [
  'button',
  'a[href]',
  'input',
  'select',
  'textarea',
  '[role="button"]',
  '[role="link"]',
  '[data-vl-target]',
].join(',');

const learningReplayBlockedSelector = [
  '.rail-mode-record',
  '.sandbox-stop-overlay',
  '.reference-embed-popup',
].join(',');

const learningRecordingBlockedSelector = [
  '.rail-mode-record',
  '.sandbox-stop-overlay',
  '.reference-embed-popup',
].join(',');

const visualReplayLastDemoDirective = '[[控制:回放剛剛示範]]';
const legacyVisualReplayLastDemoDirective = '[[visual_replay:last_demo]]';
const visualReplayTaggedDirectivePattern = /\[\[控制:回放示範\s+tag=([a-zA-Z0-9_.:-]+)\]\]/;
const learningReplayStepDelayMs = 950;

function compactLearningText(value, fallback = '') {
  const text = String(value || '').replace(/\s+/g, ' ').trim();
  return text ? text.slice(0, 120) : fallback;
}

function cssEscapeIdent(value) {
  if (typeof CSS !== 'undefined' && CSS.escape) return CSS.escape(value);
  return String(value || '').replace(/[^a-zA-Z0-9_-]/g, '\\$&');
}

function buildLearningSelector(element) {
  if (!element || element.nodeType !== 1) return '';
  if (element.id) return `#${cssEscapeIdent(element.id)}`;
  const parts = [];
  let node = element;
  while (node && node.nodeType === 1 && parts.length < 4) {
    const tag = node.tagName.toLowerCase();
    let part = tag;
    const dataAttr = node.getAttribute('data-testid') ? 'data-testid' : node.getAttribute('data-vl-target') ? 'data-vl-target' : '';
    const testId = dataAttr ? node.getAttribute(dataAttr) : '';
    if (testId) {
      part += `[${dataAttr}="${String(testId).replace(/"/g, '\\"')}"]`;
      parts.unshift(part);
      break;
    }
    if (node.classList?.length) {
      part += `.${Array.from(node.classList).slice(0, 2).map(cssEscapeIdent).join('.')}`;
    }
    const parent = node.parentElement;
    if (parent) {
      const siblings = Array.from(parent.children).filter((child) => child.tagName === node.tagName);
      if (siblings.length > 1) part += `:nth-of-type(${siblings.indexOf(node) + 1})`;
    }
    parts.unshift(part);
    node = parent;
  }
  return parts.join(' > ');
}

function describeLearningTarget(rawTarget) {
  const element = rawTarget?.nodeType === 1 ? rawTarget : rawTarget?.parentElement;
  const interactive = element?.closest?.(visualLearningInteractiveSelector) || element;
  const rect = interactive?.getBoundingClientRect?.();
  const tag = interactive?.tagName?.toLowerCase?.() || '';
  const role = interactive?.getAttribute?.('role') || (tag === 'a' ? 'link' : tag);
  const label = compactLearningText(
    interactive?.getAttribute?.('aria-label') ||
    interactive?.getAttribute?.('title') ||
    interactive?.innerText ||
    interactive?.value ||
    interactive?.placeholder ||
    tag,
    tag || 'element',
  );
  return {
    element: interactive,
    label,
    role,
    tag,
    selector: buildLearningSelector(interactive),
    rect: rect ? {
      x: Number(rect.x.toFixed(2)),
      y: Number(rect.y.toFixed(2)),
      width: Number(rect.width.toFixed(2)),
      height: Number(rect.height.toFixed(2)),
    } : null,
  };
}

function buildLearningWindowsAnchor(clickX, clickY, rect, viewport) {
  const width = Math.max(1, Math.round(Number(viewport?.width || window.innerWidth || 1)));
  const height = Math.max(1, Math.round(Number(viewport?.height || window.innerHeight || 1)));
  const click = {
    x: clampReplayCoordinate(Math.round(Number(clickX || 0)), 0, width - 1),
    y: clampReplayCoordinate(Math.round(Number(clickY || 0)), 0, height - 1),
  };
  if (rect && Number(rect.width) > 0 && Number(rect.height) > 0) {
    const box = clampLearningBBox({
      x: Math.round(Number(rect.x || 0)),
      y: Math.round(Number(rect.y || 0)),
      w: Math.round(Number(rect.width || 0)),
      h: Math.round(Number(rect.height || 0)),
    }, width, height);
    return {
      platform: 'windows',
      ok: true,
      mode: 'dom_rect_anchor',
      reason: 'in-app DOM element rect captured during learning; YOLO/OpenCV screenshot resolver was not required',
      click,
      execution_point: {
        x: Math.round(box.x + box.w / 2),
        y: Math.round(box.y + box.h / 2),
      },
      execution_hint: 'click_bbox_center',
      anchor_bbox: box,
      crop_bbox: box,
      ocr_status: 'not_used',
      ocr_note: 'OCR is optional and not used for this recorded DOM anchor.',
      detector_backend: 'dom',
      detector_degraded: false,
      needs_review: false,
    };
  }
  const box = clampLearningBBox({
    x: click.x - 14,
    y: click.y - 14,
    w: 28,
    h: 28,
  }, width, height);
  return {
    platform: 'windows',
    ok: true,
    mode: 'manual_click_box',
    reason: 'no element rectangle was available during learning; preserving the click as a small manual anchor',
    click,
    execution_point: click,
    execution_hint: 'fast_click_original_point',
    anchor_bbox: box,
    crop_bbox: box,
    ocr_status: 'not_used',
    ocr_note: 'OCR is optional and not used for this recorded manual anchor.',
    detector_backend: 'dom',
    detector_degraded: true,
    needs_review: true,
  };
}

function clampLearningBBox(box, width, height) {
  const w = clampReplayCoordinate(Math.max(1, Math.round(Number(box.w || 1))), 1, width);
  const h = clampReplayCoordinate(Math.max(1, Math.round(Number(box.h || 1))), 1, height);
  const x = clampReplayCoordinate(Math.round(Number(box.x || 0)), 0, Math.max(0, width - w));
  const y = clampReplayCoordinate(Math.round(Number(box.y || 0)), 0, Math.max(0, height - h));
  return { x, y, w, h };
}

function App() {
  /* i18n: component hook */
  const t = useI18n(s => s.t);
  const [state, setState] = useState(fallbackState);
  const [messagesByAgent, setMessagesByAgent] = useState({main: fallbackState.messages});
  const [draft, setDraft] = useState('');
  /* i18n: persona fallback */ const [personaName, setPersonaName] = useState(_t('persona.fallbackName'));
  const [personaJob, setPersonaJob] = useState(_t('persona.fallbackJob'));
  const [activePanel, setActivePanel] = useState(null);
  const [toolPopupsOpen, setToolPopupsOpen] = useState({left: false, right: false});
  const [activeToolTabs, setActiveToolTabs] = useState({left: 'external', right: 'external'});
  const [favoriteToolIds, setFavoriteToolIds] = useState(['doc-entrance', 'gmail']);
  const [hiddenToolIds, setHiddenToolIds] = useState([]);
  const [draggedTool, setDraggedTool] = useState(null);
  const [dragActionTool, setDragActionTool] = useState(null);
  const [copyConfirmTool, setCopyConfirmTool] = useState(null);
  const [referenceFiles, setReferenceFiles] = useState([]);
  const [referenceExportDialog, setReferenceExportDialog] = useState(null);
  const [referenceLinkOpen, setReferenceLinkOpen] = useState(false);
  const [referenceLinkValue, setReferenceLinkValue] = useState('');
  const referenceInternalDragRef = useRef(false);
  const referenceDropSuppressUntilRef = useRef(0);
  const [settingsState, setSettingsState] = useState(fallbackSettings);
  const [tools, setTools] = useState(getFallbackTools());
  const [toolResult, setToolResult] = useState(null);

  // Right rail learning/recording mode state. Backend capture and background work can attach here.
  const [learningEnabled, setLearningEnabled] = useState(false);
  const [recordingEnabled, setRecordingEnabled] = useState(false);
  const [learningDigestReady, setLearningDigestReady] = useState(() => {
    try {
      return window.localStorage.getItem(learningDigestStorageKey) === 'true';
    } catch {
      return false;
    }
  });

  // ── v3.6.2 W3A Media Provenance（§9A）state ──
  // w3aImportPopup   : 匯入選單資料（ImportResult 結構，null = 關閉）
  // w3aToastMsg      : 傳輸引導 toast 訊息（null = 隱藏）
  // w3aTrustList     : 信任清單陣列
  const [w3aImportPopup, setW3aImportPopup] = useState(null);
  const [w3aToastMsg, setW3aToastMsg] = useState(null);
  const [w3aTrustList, setW3aTrustList] = useState([]);
  const [w3aDetail, setW3aDetail] = useState(null);
  const [w3aPollutionResult, setW3aPollutionResult] = useState(null);
  const [w3aTransferGuidance, setW3aTransferGuidance] = useState(null);
  const [w3aActionBusy, setW3aActionBusy] = useState('');
  const [w3aActionError, setW3aActionError] = useState('');

  // ── v3.6.3 Remote Bridge Communication（§12A）state ──
  // remoteBridgeChannels : 已註冊通道陣列，結構同 Go ChannelBinding（id, channel, mode, active…）
  //   → 由 refreshRemoteBridgeChannels() 從後端拉取，驅動黃框區域 icon 渲染
  // remoteBridgeDetecting : 正在偵測/測試連線中（用於 loading indicator）
  // remoteBridgeModePopup : 目前開啟模式切換彈窗的 channelID（null = 關閉）
  const [remoteBridgeChannels, setRemoteBridgeChannels] = useState([]);
  const [remoteBridgeDetecting, setRemoteBridgeDetecting] = useState(false);
  const [remoteBridgeModePopup, setRemoteBridgeModePopup] = useState(null);

  // §12A.2 使用者分流模式 — Quick Mode / Developer Mode
  // remoteBridgeSetupMode: null（未選）| 'quick' | 'developer'
  // remoteBridgeSetupStep: 'mode_select' | 'quick_platform' | 'quick_fields' | 'developer_form'
  const [remoteBridgeSetupMode, setRemoteBridgeSetupMode] = useState(null);
  const [remoteBridgeSetupStep, setRemoteBridgeSetupStep] = useState('mode_select');
  const [remoteBridgeSetupGuideStep, setRemoteBridgeSetupGuideStep] = useState(0);
  const [remoteBridgeSetupPlatform, setRemoteBridgeSetupPlatform] = useState(null);
  const [remoteBridgeSetupFields, setRemoteBridgeSetupFields] = useState({});
  const [remoteBridgeSetupOpen, setRemoteBridgeSetupOpen] = useState(false);
  const [remoteBridgeRenameTarget, setRemoteBridgeRenameTarget] = useState(null);
  const [remoteBridgeRenameDraft, setRemoteBridgeRenameDraft] = useState('');
  const [remoteBridgeInboundInfo, setRemoteBridgeInboundInfo] = useState(null);
  const [remoteBridgeInboundAdapters, setRemoteBridgeInboundAdapters] = useState([]);
  // §12A.5B Dispatch 狀態追蹤（key: dispatch_id）
  const [dispatchStatus, setDispatchStatus] = useState({});
  // v3.3.2 P0 state
  const [toolVisibility, setToolVisibility] = useState([]);
  const [pendingImport, setPendingImport] = useState(null);
  const [pendingPackageData, setPendingPackageData] = useState(null);
  const [packageInstallError, setPackageInstallError] = useState('');

  // I-5: External Link 三路分流 — 後端真實資料取代靜態假資料
  // external_service → 工具彈窗「外部連結」tab
  // adapter_candidate → 左側 CLI Adapter 列表
  // documentation → 純參考連結，不進工具執行區
  const [extServiceLinks, setExtServiceLinks] = useState([]);
  const [extAdapterLinks, setExtAdapterLinks] = useState([]);
  const [extDocLinks, setExtDocLinks] = useState([]);
  // I-5: PreviewExternalLink 預覽結果，用戶確認後才 Register
  const [linkPreview, setLinkPreview] = useState(null);
  const [linkPreviewError, setLinkPreviewError] = useState('');
  const [linkPreviewSuggestions, setLinkPreviewSuggestions] = useState([]);
  const [llmAPISetup, setLlmAPISetup] = useState(null);
  const [webSearchConfig, setWebSearchConfig] = useState(null);
  const [webSearchSetup, setWebSearchSetup] = useState(null);
  const [webSearchSetupError, setWebSearchSetupError] = useState('');
  const [adapterRenameTarget, setAdapterRenameTarget] = useState(null);
  const [adapterRenameDraft, setAdapterRenameDraft] = useState('');
  const [llmAPISetupGuideStep, setLlmAPISetupGuideStep] = useState(0);
  // I-6 (#I-602): Reauth Intercept — unavailable 工具執行前攔截
  // 存放被攔截的 tool 物件，dialog 顯示後由用戶決定重試或取消
  const [reauthTool, setReauthTool] = useState(null);

  // --- 遺留能力 #1–#8 state ---
  // #1 Adapter Registry: 從 Go service 取得的即時 adapter 列表與狀態
  const [adapterList, setAdapterList] = useState([]);
  // §M-1 adapter model 雙擊彈窗：choices = {adapterID: model}, options = {adapterID: [model...]}
  const [adapterModelChoices, setAdapterModelChoices] = useState({});
  const [adapterModelOptions, setAdapterModelOptions] = useState({});
  const [subagentTabs, setSubagentTabs] = useState([]);
  const [subExportCapabilities, setSubExportCapabilities] = useState({
    platform: '',
    native_drag_supported: false,
    native_drag_strategy: 'fallback_directory',
    fallback_supported: true,
    message: '',
  });
  const [summaryModelSettings, setSummaryModelSettings] = useState(null);
  const [voiceState, setVoiceState] = useState(null);
  const [voiceRecording, setVoiceRecording] = useState(false);
  const [voiceBusy, setVoiceBusy] = useState(false);
  const [voiceInstallBusy, setVoiceInstallBusy] = useState(false);
  const [voiceStatus, setVoiceStatus] = useState('');
  const [voiceError, setVoiceError] = useState('');
  const [voiceBackgroundResult, setVoiceBackgroundResult] = useState(null);
  const voiceRecorderRef = useRef(null);
  const [credentialMigrationStatus, setCredentialMigrationStatus] = useState(null);
  const [credentialMigrationBusy, setCredentialMigrationBusy] = useState(false);
  const [summaryModelScan, setSummaryModelScan] = useState({options: [], message: ''});
  const [installCandidate, setInstallCandidate] = useState(null);
  const [subImportResult, setSubImportResult] = useState(null);
  // 目前使用者選中的 adapter（點擊按鈕亮框）
  const [activeAdapterId, setActiveAdapterId] = useState(null);
  const [activeHaoraId, setActiveHaoraId] = useState(null);
  const activeAdapterIdRef = useRef(null);
  const adapterListRef = useRef([]);
  const subagentTabsRef = useRef([]);
  const summaryInFlightRef = useRef(false);
  activeAdapterIdRef.current = activeAdapterId;
  adapterListRef.current = adapterList || [];
  subagentTabsRef.current = subagentTabs || [];
  // 只有選到 sub tab 時才切對話；選左側 CLI adapter 仍留在 main。
  const activeSubagentForConversation = (subagentTabs || []).find((tab) => (
    activeHaoraId && (tab.id === activeHaoraId || tab.name === activeHaoraId)
  ));
  const activeConversationId = activeSubagentForConversation?.id || 'main';
  const activeConversationIdRef = useRef('main');
  const loadedConversationIdsRef = useRef(new Set(['main']));
  const messages = messagesByAgent[activeConversationId] || [];
  const [cliInspectorLog, setCliInspectorLog] = useState(null);
  const [, setChatCliLog] = useState(null);
  const [cliInspectorBusy, setCliInspectorBusy] = useState(false);
  // Sidecar 狀態：追蹤 Node sidecar 是否正常運行，
  // 用於在 UI 上顯示連線狀態提示（running / start_failed / crashed）
  const [sidecarState, setSidecarState] = useState('unknown');
  // CLI 授權對話框狀態：當 CLI 需要瀏覽器 OAuth 時，存放授權請求的資訊。
  // 結構：{adapter_id, auth_url, message, user_text, session_id} 或 null（無授權請求）
  const [cliAuthRequest, setCliAuthRequest] = useState(null);
  // #2 Review Card: 統一 review card inbox（blocking + pending + background）
  const [reviewCards, setReviewCards] = useState([]);
  const [hasBlocking, setHasBlocking] = useState(false);
  // #I-1002: Review Archive — rejected 卡片歷史
  const [reviewArchive, setReviewArchive] = useState([]);
  // v3.6: Lightweight Review Cards
  const [lightweightCards, setLightweightCards] = useState([]);
  // v3.6: Stop Recovery Cards（取代簡易 sidecar crash 偵測）
  const [stopRecoveryCards, setStopRecoveryCards] = useState([]);
  const [hasOpenStopRecovery, setHasOpenStopRecovery] = useState(false);
  // v3.6: Visual Learning 後端狀態
  const [vlLearningActive, setVlLearningActive] = useState(false);
  const [vlActiveLearningRun, setVlActiveLearningRun] = useState(null);
  const [vlPendingCount, setVlPendingCount] = useState(0);
  const [vlHasBlocking, setVlHasBlocking] = useState(false);
  const [vlMonitorOpen, setVlMonitorOpen] = useState(false);
  const [vlRecentLearningEvents, setVlRecentLearningEvents] = useState([]);
  // §M3 Embedding picker：拖入第一份文件時若沒設定 embed model，後端會發 embedding:config_missing
  const [embeddingPickerTarget, setEmbeddingPickerTarget] = useState(null); // {displayName} 或 null
  // §M3+ 雙擊引用文件 → 顯示 embedding 狀態 popup
  const [refEmbedPopup, setRefEmbedPopup] = useState(null); // {file, rect, config} 或 null
  useEffect(() => {
    const off = EventsOn('embedding:config_missing', (payload) => {
      setEmbeddingPickerTarget({displayName: payload?.displayName || ''});
    });
    return () => { off && off(); };
  }, []);
  useEffect(() => {
    if (!refEmbedPopup) return;
    const close = (ev) => {
      if (ev.target?.closest?.('.reference-embed-popup')) return;
      setRefEmbedPopup(null);
    };
    const esc = (ev) => { if (ev.key === 'Escape') setRefEmbedPopup(null); };
    window.addEventListener('mousedown', close);
    window.addEventListener('keydown', esc);
    return () => {
      window.removeEventListener('mousedown', close);
      window.removeEventListener('keydown', esc);
    };
  }, [refEmbedPopup]);
  useEffect(() => {
    if (!vlLearningActive) return undefined;
    const handleLearningClick = (event) => {
      if (learningReplayExecutingRef.current) return;
      const target = event.target;
      if (target?.closest?.(learningRecordingBlockedSelector)) return;

      const described = describeLearningTarget(target);
      const eventId = `vl-click-${Date.now()}-${Math.round(event.clientX)}-${Math.round(event.clientY)}`;
      const viewport = {
        width: window.innerWidth,
        height: window.innerHeight,
        device_scale: window.devicePixelRatio || 1,
      };
      const payload = {
        timestamp: new Date().toISOString(),
        event_type: 'click',
        x: Math.round(event.clientX),
        y: Math.round(event.clientY),
        button: event.button === 2 ? 'right' : event.button === 1 ? 'middle' : 'left',
        target_region_id: eventId,
        target_label: described.label,
        target_role: described.role,
        target_tag: described.tag,
        css_selector: described.selector,
        target_rect: described.rect,
        viewport,
        windows_anchor: buildLearningWindowsAnchor(
          Math.round(event.clientX),
          Math.round(event.clientY),
          described.rect,
          viewport
        ),
      };
      const recent = {
        id: eventId,
        label: payload.target_label,
        role: payload.target_role,
        selector: payload.css_selector,
        x: payload.x,
        y: payload.y,
        saved: false,
        error: '',
      };
      setVlRecentLearningEvents((items) => [recent, ...items].slice(0, 8));
      callWails(() => RecordLearningMouseEvent(JSON.stringify(payload)))
        .then(() => {
          setVlRecentLearningEvents((items) => items.map((item) => (
            item.id === eventId ? {...item, saved: true} : item
          )));
        })
        .catch((error) => {
          setVlRecentLearningEvents((items) => items.map((item) => (
            item.id === eventId ? {...item, error: error?.message || String(error)} : item
          )));
        });
    };
    window.addEventListener('click', handleLearningClick, true);
    return () => window.removeEventListener('click', handleLearningClick, true);
  }, [vlLearningActive]);
  const handleReferenceCardDoubleClick = async (file, rect) => {
    try {
      const cfg = await GetEmbeddingConfig();
      setRefEmbedPopup({file, rect, config: cfg || {}});
    } catch (_) {
      setRefEmbedPopup({file, rect, config: {}});
    }
  };
  // §M3+ 失敗 reference entry：拖到 panel 外面 → 從前端 state 移除（後端沒有實檔可刪）
  const handleReferenceFailedRemove = (key) => {
    if (!key) return;
    setReferenceFiles((current) => current.filter((f) => referenceFileKey(f) !== key));
  };
  // v3.6: Avatar 後端狀態
  const [currentAvatar, setCurrentAvatar] = useState(null);
  const [avatarConfigs, setAvatarConfigs] = useState({});
  const [avatarPresets, setAvatarPresets] = useState([]);
  const [avatarModeNotice, setAvatarModeNotice] = useState('');
  const [avatarUploadTargetId, setAvatarUploadTargetId] = useState(null);
  const [manualAvatarState, setManualAvatarState] = useState('');
  const [avatarClock, setAvatarClock] = useState(Date.now());
  const [windowInactive, setWindowInactive] = useState(() => typeof document !== 'undefined' ? document.hidden : false);
  const appStartedAtRef = useRef(Date.now());
  const learningReplayExecutingRef = useRef(false);
  const [pendingLearningReplayStartConfirm, setPendingLearningReplayStartConfirm] = useState(null);
  const [pendingLearningReplayConfirm, setPendingLearningReplayConfirm] = useState(null);
  const [renderedPixelAvatars, setRenderedPixelAvatars] = useState({});
  const [staticAvatarPreviews, setStaticAvatarPreviews] = useState({});
  // v3.6: Memory Pipeline 狀態
  const [memoryPipelineState, setMemoryPipelineState] = useState(null);
  // v3.6: Source Trust 來源信任提示（後端自動分類，前端顯示中文提示）
  const [sourceTrustHint, setSourceTrustHint] = useState(null);
  // v3.6: Project Lifecycle — 專案管理彈窗狀態
  const [projectManageOpen, setProjectManageOpen] = useState(false);
  const [projectManageView, setProjectManageView] = useState('menu'); // 'menu' | 'manifests'
  const [purgeManifests, setPurgeManifests] = useState([]);
  const [purgeConfirmStep, setPurgeConfirmStep] = useState(null); // null | 'project' | 'boundary'
  const [dismissedDestructiveCards, setDismissedDestructiveCards] = useState([]);
  const [destructiveReviewResult, setDestructiveReviewResult] = useState(null);
  // #4 Memory Health / Config Public: read-only 健康指標與公開設定
  const [memoryHealth, setMemoryHealth] = useState(null);
  const [configPublic, setConfigPublic] = useState(null);
  // #5 Degraded Mode: 高風險能力停用狀態
  const [degradedState, setDegradedState] = useState({active: false, blocked_ops: []});
  // #6 Onboarding / Read-only mode
  const [onboardingState, setOnboardingState] = useState(null);
  const [readOnlyMode, setReadOnlyMode] = useState(false);

  // I-1: Review Panel real data
  const [reviewState, setReviewState] = useState(fallbackReviewState);
  const [reviewPopup, setReviewPopup] = useState(null);
  const [snoozeHours, setSnoozeHours] = useState(4);
  // I-7: DAG run front-end mirror; backend Hook Run is attached once a task is created.
  const [dagRun, setDagRun] = useState(null);
  const dagRunRef = useRef(null);
  const [pendingTaskReview, setPendingTaskReview] = useState(null);
  // #I-805: Sidecar 狀態追蹤 — 已移至上方（502 行），統一用字串格式。
  // 舊版用 {status, error} object，現在統一為 'unknown' | 'idle' | 'running' | 'crashed' | 'start_failed'。

  // #I-1001: 自動封存 Toast 提示 + 系統狀態歷史
  const [autoArchiveToast, setAutoArchiveToast] = useState(null);
  const [systemStatusHistory, setSystemStatusHistory] = useState([]);

  // §30: 關閉視窗 → 存成 sub 對話框
  const [sessionClosePrompt, setSessionClosePrompt] = useState(null);

  // I-2: Skill Context Orchestration
  const [appSessionId, setAppSessionId] = useState('');
  const appSessionIdRef = useRef('');
  appSessionIdRef.current = appSessionId || '';
  const [skillInjections, setSkillInjections] = useState([]);
  const [archivedSkills, setArchivedSkills] = useState([]);
  const [skillScanPreview, setSkillScanPreview] = useState(null);
  // #I-207: 初次使用說明卡——後端 settings 持久化旗標
  const [skillFirstUseExplained, setSkillFirstUseExplained] = useState(true); // 預設 true 避免閃爍
  const [showSkillFirstUseCard, setShowSkillFirstUseCard] = useState(false);
  const [skillExecutionConfirm, setSkillExecutionConfirm] = useState(null);

  // I-3: Controlled Trust
  const [activeSandboxId, setActiveSandboxId] = useState(null);
  const [sandboxStopOptions, setSandboxStopOptions] = useState(null); // {sandboxId, reason}
  const [activeOverrides, setActiveOverrides] = useState([]); // [{id, scope, allowedRisk, expiry}]
  const [trustedSessionActive, setTrustedSessionActive] = useState(false);
  const [trustedSessionExpired, setTrustedSessionExpired] = useState(false);
  const [trustDomClickEnabled, setTrustDomClickEnabled] = useState(false);
  const [deviceProfile, setDeviceProfile] = useState(null);

  // I-4: Browser Preference & UI Settings
  const [browserPref, setBrowserPref] = useState(null); // {browser, profilePath, ...}
  const [safariNotice, setSafariNotice] = useState(null); // {message, ...}
  const [showSafariNotice, setShowSafariNotice] = useState(false);
  const [styleDiffPreview, setStyleDiffPreview] = useState(null); // {diffJson, preview}
  const [styleDiffError, setStyleDiffError] = useState('');

  // ── v3.6.4 Readiness Gate UI Interaction Layer state ──
  // readinessGate       : 後端 ReadinessGateState 快照（定期輪詢同步）
  // longPressActive     : 長按確認按鈕是否正在按住中
  // longPressProgress   : 長按進度百分比（0–100），驅動進度條寬度
  // gachaPhase          : 轉蛋動畫階段（null | 'collapse' | 'pulse' | 'particles' | 'reveal'）
  // riskImpactExpanded  : 高風險影響說明面板是否已展開（第三層確認流程）
  const [readinessGate, setReadinessGate] = useState(fallbackReadinessGate);
  const [longPressActive, setLongPressActive] = useState(false);
  const [longPressProgress, setLongPressProgress] = useState(0);
  const [gachaPhase, setGachaPhase] = useState(null);
  const [riskImpactExpanded, setRiskImpactExpanded] = useState(false);
  const longPressTimerRef = useRef(null);
  const longPressStartRef = useRef(null);
  const manualGreetingLockedRef = useRef(false);

  function unlockManualGreeting() {
    manualGreetingLockedRef.current = false;
  }

  // 對話依 agent id 分桶，避免切 sub 時共用同一份 messages。
  function setConversationMessages(conversationId, updater) {
    const id = conversationId || 'main';
    setMessagesByAgent((prev) => {
      const current = Array.isArray(prev[id]) ? prev[id] : [];
      const next = typeof updater === 'function' ? updater(current) : updater;
      return {...prev, [id]: Array.isArray(next) ? next : []};
    });
  }

  function setMessages(updater) {
    setConversationMessages(activeConversationIdRef.current, updater);
  }

  // 第一次切到某個 agent 時，從它自己的 talk_full.md 補回對話。
  function loadConversationMessages(conversationId, options = {}) {
    const id = conversationId || 'main';
    if (!options.force && loadedConversationIdsRef.current.has(id)) return;
    loadedConversationIdsRef.current.add(id);
    callWails(() => GetTalkMessagesForAgent(id))
      .then((next) => {
        setConversationMessages(id, Array.isArray(next) ? next : []);
      })
      .catch((err) => {
        loadedConversationIdsRef.current.delete(id);
        setConversationMessages(id, (prev) => (
          prev.length > 0 ? prev : [t('system.conversationLoadFailed', { id: id === 'main' ? t('system.mainAgent') : id, error: err?.message || err })]
        ));
      });
  }

  useEffect(() => {
    activeConversationIdRef.current = activeConversationId;
    callWails(() => SetActiveConversationAgent(activeConversationId)).catch(() => {});
    loadConversationMessages(activeConversationId);
  }, [activeConversationId]);

  async function refreshAvailableAdapters() {
    const adapters = await callWails(ListAvailableAdapters).catch(() => []);
    const order = await callWails(GetTabOrder).catch(() => null);
    const {cliAdapters, subagentTabs: nextSubagentTabs} = splitAvailableAdapters(adapters || [], order);
    setAdapterList(cliAdapters);
    setSubagentTabs(nextSubagentTabs);
    setState((prev) => ({
      ...prev,
      adapters: cliAdapters.map((adapter) => adapter.name || adapter.id),
      haoras: [t('system.mainAgent'), ...nextSubagentTabs.map((tab) => tab.name || tab.id)],
    }));
    return {cliAdapters, subagentTabs: nextSubagentTabs};
  }

  useEffect(() => {
    let cancelled = false;
    callWails(GetSubExportCapabilities)
      .then((capabilities) => {
        if (!cancelled && capabilities) setSubExportCapabilities(capabilities);
      })
      .catch(() => {});
    return () => { cancelled = true; };
  }, []);

  // §M-1 載入 adapter model 偏好 + 候選清單。adapterList 變動時重新查一次。
  useEffect(() => {
    if (!adapterList || adapterList.length === 0) return;
    let cancelled = false;
    (async () => {
      try {
        const choices = await callWails(GetAdapterModelChoices).catch(() => ({}));
        if (cancelled) return;
        setAdapterModelChoices(choices || {});
        const out = {};
        for (const a of adapterList) {
          const id = a.id || a.name;
          if (!id) continue;
          const opts = await callWails(() => ListAdapterModelOptions(id)).catch(() => null);
          if (Array.isArray(opts) && opts.length > 0) out[id] = opts;
        }
        if (!cancelled) setAdapterModelOptions(out);
      } catch (_) { /* silent — badge 只是輔助，失敗不要中斷 UI */ }
    })();
    return () => { cancelled = true; };
  }, [adapterList]);

  // §M-1 套用 adapter 的 model 選擇（picker 點按時呼叫）。
  const handleAdapterModelPick = async (adapterID, model) => {
    if (!adapterID || !model) return;
    setAdapterModelChoices((prev) => ({...prev, [adapterID]: model}));
    try { await callWails(() => SetAdapterModelChoice(adapterID, model)); } catch (_) {}
  };

  const handleAdapterModelRefresh = async (adapterID) => {
    if (!adapterID) return [];
    const opts = await callWails(() => ListAdapterModelOptions(adapterID)).catch(() => null);
    if (Array.isArray(opts) && opts.length > 0) {
      setAdapterModelOptions((prev) => ({...prev, [adapterID]: opts}));
      return opts;
    }
    setAdapterModelOptions((prev) => {
      const next = {...prev};
      delete next[adapterID];
      return next;
    });
    return [];
  };

  // Learning mode waits for an idle window before surfacing a digest-ready hint.
  useEffect(() => {
    callWails(GetConsoleState)
      .then((next) => {
        const hydrated = {
          ...fallbackState,
          ...(next || {}),
          adapters: Array.isArray(next?.adapters) ? next.adapters : fallbackState.adapters,
          haoras: Array.isArray(next?.haoras) ? next.haoras : fallbackState.haoras,
          messages: Array.isArray(next?.messages) ? next.messages : [],
        };
        setState(hydrated);
        loadedConversationIdsRef.current.add('main');
        setConversationMessages('main', hydrated.messages);
      })
      .catch(() => {});
    callWails(GetSettingsState)
      .then((next) => {
        const normalized = normalizeSettingsState(next);
        setSettingsState(normalized);
        const activePersona = findActivePersona(normalized);
        if (activePersona?.name) setPersonaName(activePersona.name);
        if (activePersona?.identity) setPersonaJob(activePersona.identity);
      })
      .catch(() => {});
    callWails(ListTools)
      .then((nextTools) => setTools(normalizeToolList(nextTools)))
      .catch(() => {});
    callWails(GetToolVisibility)
      .then(setToolVisibility)
      .catch(() => {});
    refreshReferenceFiles().catch(() => {});

    // I-2: Get app session ID on startup
    callWails(GetAppSessionID)
      .then((sid) => setAppSessionId(sid || ''))
      .catch(() => {});

    // I-1: Load Review Panel real data
    loadReviewPanelData();

    // I-2: Load archived skills for 工具包 tab
    callWails(ListArchivedSkills)
      .then((skills) => setArchivedSkills(skills || []))
      .catch(() => {});

    // I-4: Load browser preference on startup
    callWails(GetBrowserPreference)
      .then((pref) => setBrowserPref(pref || null))
      .catch(() => {});

    // I-4: Load UI Settings on startup（含 #I-207 初次使用旗標）
    callWails(GetUISettings)
      .then((settings) => {
        if (settings && !settings.skill_first_use_explained) {
          setSkillFirstUseExplained(false);
        }
      })
      .catch(() => {});
    callWails(GetSummaryModelSettings)
      .then((settings) => setSummaryModelSettings(settings || null))
      .catch(() => {});
    callWails(GetVoiceSettings)
      .then((settings) => setVoiceState(settings || null))
      .catch(() => setVoiceState(null));
    callWails(GetWebSearchConfig)
      .then((config) => setWebSearchConfig(config || null))
      .catch(() => setWebSearchConfig({configured: false, options: defaultWebSearchProviderOptions()}));
    callWails(ScanLocalSummaryModels)
      .then((result) => setSummaryModelScan(result || {options: [], message: ''}))
      .catch(() => setSummaryModelScan({options: [], message: t('settings.noLocalModelDetected')}));

    // I-5: Load external links by type — 三路分流初始化
    refreshExternalLinks();

    // --- 遺留能力 #1–#8 初始化 ---

    // #1 Adapter List: CLI adapter 與 subagent tabs 分流，避免左側 Adapter 混入黃色 haㄌer。
    refreshAvailableAdapters().catch(() => {});

    // #2 Review Card: 載入 open cards + blocking 狀態
    refreshReviewCards();

    // #4 Memory Health + Config Public
    callWails(GetMemoryHealth)
      .then((h) => setMemoryHealth(h || null))
      .catch(() => {});
    callWails(GetConfigPublic)
      .then((c) => setConfigPublic(c || null))
      .catch(() => {});

    // #5 Degraded Mode 初始狀態
    callWails(GetDegradedState)
      .then((s) => setDegradedState(s || {active: false, blocked_ops: []}))
      .catch(() => {});

    // #6 Onboarding: 檢查是否為首次啟動
    callWails(GetOnboardingState)
      .then((s) => {
        setOnboardingState(s || null);
        setReadOnlyMode(s?.read_only_mode || false);
      })
      .catch(() => {});
    callWails(IsReadOnlyMode)
      .then((ro) => setReadOnlyMode(ro || false))
      .catch(() => {});

    // ── v3.6 啟動載入 ──

    // Stop Recovery Cards
    refreshStopRecoveryCards();
    // Visual Learning 狀態同步
    refreshVisualLearningState();
    // Avatar 預設列表
    loadAvatarPresets();
    loadCurrentAvatar();
    callWails(GetCredentialMigrationStatus)
      .then((status) => setCredentialMigrationStatus(status || null))
      .catch((err) => setCredentialMigrationStatus({error: err?.message || String(err)}));
    // v3.6: Source Trust — 啟動時載入信任清單 + 高影響領域
    refreshSourceTrustState();
    // Memory Pipeline 狀態
    callWails(GetMemoryPipelineState)
      .then((s) => setMemoryPipelineState(s || null))
      .catch(() => {});

    // v3.6.3 Remote Bridge — 載入已註冊通道
    refreshRemoteBridgeChannels();
    callWails(ListRemoteBridgeInboundAdapters)
      .then((adapters) => setRemoteBridgeInboundAdapters(adapters || []))
      .catch(() => setRemoteBridgeInboundAdapters([]));

    // v3.6.4 Readiness Gate — 啟動時載入初始狀態
    refreshReadinessGateState();
  }, []);

  // #7 事件匯流排: 訂閱 Wails runtime events（取代 WebSocket event log）
  useEffect(() => {
    // Adapter 狀態變更 → 重新拉取列表
    const offAdapterChanged = EventsOn('adapter:list_changed', () => {
      refreshAvailableAdapters().catch(() => {});
    });
    const offAdapterStatus = EventsOn('adapter:status_changed', () => {
      refreshAvailableAdapters().catch(() => {});
    });

    // Sidecar 狀態事件：監聽 Node sidecar 的生命週期變化。
    // 當 sidecar 啟動失敗或崩潰時，在主聊天區顯示錯誤提示，
    // 讓使用者立即知道 CLI 通道斷了，而非送出訊息後一片空白。
    const offSidecarState = EventsOn('sidecar:state_changed', (newState) => {
      setSidecarState(newState || 'unknown');
      if (newState === 'start_failed') {
        setMessages((prev) => [
          ...prev,
          t('system.sidecarFail'),
        ]);
      } else if (newState === 'crashed') {
        setMessages((prev) => [
          ...prev,
          t('system.sidecarCrash'),
        ]);
      }
    });

    // CLI 授權事件：CLI（如 Gemini）需要瀏覽器 OAuth 授權時，
    // Go 端已自動用系統瀏覽器開啟 OAuth URL，此處顯示授權對話框讓使用者確認。
    const offCLIAuth = EventsOn('cli:auth_required', (payload) => {
      if (!payload) return;
      const conversationId = activeConversationIdRef.current || 'main';
      // SEC-05: 帶入 trusted 狀態，前端依此決定 UI 行為
      const isTrusted = payload.auth_trusted === 'true';
      setCliAuthRequest({...payload, conversation_id: conversationId, isTrusted});
      if (isTrusted) {
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          t('system.authBrowserOpen', { adapter: payload.adapter_id || 'CLI' }),
        ]);
      } else {
        // untrusted：尚未開瀏覽器，需使用者確認
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          t('system.authBrowserConfirm', { adapter: payload.adapter_id || 'CLI', url: payload.auth_hostname || payload.auth_url || t('common.unknown') }),
        ]);
      }
    });

    const offWebSearchConfigRequired = EventsOn('web_search:config_required', (payload) => {
      const config = payload || {configured: false, options: defaultWebSearchProviderOptions()};
      setWebSearchConfig(config);
      openWebSearchSetup(config);
    });

    // DAG 事件 → 更新 dagRun 狀態
    const offDagStarted = EventsOn('dag:run_started', (payload) => {
      if (payload?.run_id) {
        setDagRun((prev) => prev ? {...prev, status: 'running'} : prev);
      }
    });
    const offDagNodeDone = EventsOn('dag:node_completed', (payload) => {
      if (payload?.node_id) {
        setDagRun((prev) => {
          if (!prev) return prev;
          return {
            ...prev,
            nodes: prev.nodes.map((n) =>
              n.id === payload.node_id ? {...n, status: 'completed'} : n
            ),
          };
        });
      }
    });
    const offDagCompleted = EventsOn('dag:run_completed', () => {
      setDagRun((prev) => prev ? {...prev, status: 'completed'} : prev);
    });
    const offDagFailed = EventsOn('dag:run_failed', () => {
      setDagRun((prev) => prev ? {...prev, status: 'failed'} : prev);
    });

    // 任務進度事件：後端保存完整 DAG，前端只顯示使用者可理解的進度。
    const syncTaskEvent = (payload) => {
      const run = syncTaskProgressRun(payload);
      if (run) syncTaskReviewState(run);
      if (run?.status === 'completed') {
        window.setTimeout(() => {
          setReviewPopup((current) => current === 'dag' ? null : current);
        }, 5000);
      }
    };
    const offTaskStarted = EventsOn('task:progress_started', syncTaskEvent);
    const offTaskUpdated = EventsOn('task:progress_updated', syncTaskEvent);
    const offTaskWaiting = EventsOn('task:progress_waiting_review', (payload) => {
      syncTaskEvent(payload);
      refreshReviewCards();
    });
    const offTaskCompleted = EventsOn('task:progress_completed', syncTaskEvent);
    const offTaskFailed = EventsOn('task:progress_failed', syncTaskEvent);
    const offTaskCancelled = EventsOn('task:progress_cancelled', syncTaskEvent);
    const offTaskSystemMessage = EventsOn('task:system_message', (payload) => {
      const message = payload?.text;
      if (!message) return;
      const conversationId = activeConversationIdRef.current || 'main';
      setConversationMessages(conversationId, (prev) => [...prev, `Ai:${message}`]);
    });

    // Review Card 事件 → 即時更新
    const offReviewAdded = EventsOn('review:card_added', () => refreshReviewCards());
    const offReviewResolved = EventsOn('review:card_resolved', () => refreshReviewCards());

    // §24: 文件匯入完成 toast
    const offDocImported = EventsOn('document:imported', (data) => {
      setW3aToastMsg(t('system.imported', { name: data?.display_name || t('w3a.defaultDocName') }));
      setTimeout(() => setW3aToastMsg(null), 4000);
    });

    // 上方互動輸出：statusRail 事件只同步 greeting，不寫入下方 messages。
    const offStatusRail = EventsOn('statusrail:updated', (payload) => {
      if (!payload) return;
      unlockManualGreeting();
      setState((prev) => ({
        ...prev,
        greeting: payload.text || prev.greeting,
        statusRail: payload,
      }));
    });

    // Degraded Mode 事件
    const offDegradedEnter = EventsOn('degraded:entered', (payload) => {
      setDegradedState(payload || {active: true, blocked_ops: []});
    });
    const offDegradedExit = EventsOn('degraded:exited', () => {
      setDegradedState({active: false, blocked_ops: []});
    });

    // Memory Health 事件
    const offMemory = EventsOn('memory:health_changed', (payload) => {
      if (payload) setMemoryHealth(payload);
    });

    // #I-1001: Digest 300 筆自動封存通知 → Toast + 系統狀態歷史
    const offDigestArchived = EventsOn('digest:auto_archived', (payload) => {
      if (payload?.archived_count > 0) {
        const msg = payload.message || t('system.autoArchive', { count: payload.archived_count });
        setAutoArchiveToast(msg);
        setSystemStatusHistory((prev) => [...prev, {time: new Date().toISOString(), text: msg}]);
        // Toast 5 秒後自動消失
        setTimeout(() => setAutoArchiveToast(null), 5000);
      }
    });

    // #I-805: Sidecar 崩潰 / 狀態變更事件
    const offExecInterrupted = EventsOn('dag:execution_interrupted', (payload) => {
      // 統一使用字串格式的 sidecarState（與上方 sidecar:state_changed 一致）
      setSidecarState('crashed');
      setMessages((prev) => [
        ...prev,
        t('system.executionInterrupt', { reason: payload?.reason || t('stopRecovery.sidecarAbnormal') }),
      ]);
    });
    // 注意：sidecar:state_changed 已在上方統一訂閱（含錯誤提示），
    // 此處不再重複訂閱，避免同名變數衝突與狀態格式不一致。

    // v3.6.3 Remote Bridge 事件
    // Go 後端 app.go 透過 eventBus.Emit 發送以下事件，
    // 前端收到後統一呼叫 refreshRemoteBridgeChannels() 重拉通道清單。
    const offRBRegistered = EventsOn('remote_bridge:channel_registered', () => refreshRemoteBridgeChannels());
    const offRBActivated = EventsOn('remote_bridge:channel_activated', () => refreshRemoteBridgeChannels());
    const offRBDeactivated = EventsOn('remote_bridge:channel_deactivated', () => refreshRemoteBridgeChannels());
    const offRBModeSwitched = EventsOn('remote_bridge:mode_switched', () => refreshRemoteBridgeChannels());
    const offRBRemoved = EventsOn('remote_bridge:channel_removed', () => refreshRemoteBridgeChannels());
    const offRBPrimaryChanged = EventsOn('remote_bridge:primary_changed', () => refreshRemoteBridgeChannels());
    const offRBDiscordStatus = EventsOn('remote_bridge:discord_gateway_status', (payload) => {
      if (!payload) return;
      if (payload.status === 'connected') {
        setToolResult({toolId: 'reference-link', ok: true, message: t('remote.discordConnected')});
      } else if (payload.error) {
        setToolResult({toolId: 'reference-link', ok: false, message: `Discord Gateway：${payload.error}`});
      }
    });
    const offRBInbound = EventsOn('remote_bridge:inbound_command', (payload) => {
      if (!payload) return;
      const isChatMessage = payload.command === 'unknown' && payload.text && payload.channel === 'discord';
      const commandLabels = {approve: t('remote.cmdApprove'), cancel: t('remote.cmdCancel'), status: t('remote.cmdStatus'), stop: t('remote.cmdStop')};
      const label = commandLabels[payload.command] || payload.command || t('remote.cmdDefault');
      const message = isChatMessage ? t('remote.discordMessage') : t('remote.remoteReply', { channel: payload.channel || 'remote', label });
      setToolResult({toolId: 'reference-link', ok: true, message});
      if (!isChatMessage) {
        setMessages((prev) => [...prev, `[${t('remote.remoteComms')}] ${message}`]);
      }
      if (payload.command === 'unknown' && payload.text && payload.channel === 'discord') {
        sendRemoteInboundText(payload);
      }
    });

    // §29.3 Summarization 觸發事件：背景整理上下文，不顯示手動摘要 banner。
    const offSummarizationNeeded = EventsOn('summarization:needed', (payload) => {
      if (!payload || summaryInFlightRef.current) return;
      summaryInFlightRef.current = true;
      callWails(() => RunSummarizationNow(activeAdapterIdRef.current || ''))
        .catch((err) => { console.error('background summarize failed', err); })
        .finally(() => { summaryInFlightRef.current = false; });
    });

    // §12A.5B Dispatch 進度 / 結果事件 — 非同步送出狀態提示
    const offDispatchProgress = EventsOn('remote_bridge:dispatch_progress', (payload) => {
      if (!payload) return;
      setDispatchStatus((prev) => ({
        ...prev,
        [payload.dispatch_id]: {
          ...prev[payload.dispatch_id],
          dispatchId: payload.dispatch_id,
          partIndex: payload.part_index,
          totalParts: payload.total_parts,
          lastStatus: payload.status,
        },
      }));
    });
    const offDispatchResult = EventsOn('remote_bridge:dispatch_result', (payload) => {
      if (!payload) return;
      setDispatchStatus((prev) => ({
        ...prev,
        [payload.dispatch_id]: {
          ...prev[payload.dispatch_id],
          dispatchId: payload.dispatch_id,
          overall: payload.overall_status,
          segments: payload.segment_results,
          done: true,
        },
      }));
      // 成功時 3 秒後自動清除
      if (payload.overall_status === 'success') {
        setTimeout(() => {
          setDispatchStatus((prev) => {
            const copy = {...prev};
            delete copy[payload.dispatch_id];
            return copy;
          });
        }, 3000);
      }
    });

    // ── v3.6.2 W3A Media Provenance（§9A）事件監聯 ──
    const offW3AVerified = EventsOn('w3a:verified', (data) => {
      if (data) setW3aVerifyResult(data);
    });
    const offW3APollution = EventsOn('w3a:pollution_detected', (data) => {
      if (data) setW3aToastMsg(t('w3a.pollutionWarning'));
      setTimeout(() => setW3aToastMsg(null), 6000);
    });
    const offW3AExported = EventsOn('w3a:exported', () => {
      setW3aToastMsg(t('w3a.transmitHint'));
      setTimeout(() => setW3aToastMsg(null), 5000);
    });
    const offW3AImported = EventsOn('w3a:imported', (data) => {
      // 由 importMediaW3A 函式處理彈窗
    });
    const offW3ATrust = EventsOn('w3a:trust_updated', () => refreshW3ATrustList());

    // §30: 關閉視窗 → 前端收到後端 session:close_prompt 事件，顯示對話框
    const offSessionClose = EventsOn('session:close_prompt', (payload) => {
      if (payload?.analysis) {
        setSessionClosePrompt(payload.analysis);
      }
    });
    const offSessionCloseConfirmed = EventsOn('session:close_confirmed', () => {
      // 後端已完成清除，下一次關閉會由 beforeClose 放行。
      Quit();
    });

    // macOS 原生錄製器降級（多半是缺 Accessibility 授權）→ 明確告知使用者
    const offNativeRecorderDegraded = EventsOn('visual_learning:native_recorder_degraded', (payload) => {
      const conversationId = activeConversationIdRef.current || 'main';
      const detail = (payload && payload.error) ? String(payload.error) : '';
      setConversationMessages(conversationId, (prev) => [
        ...prev,
        t('system.nativeRecorderDegraded') + (detail ? ` (${detail})` : ''),
      ]);
    });

    return () => {
      offAdapterChanged();
      offAdapterStatus();
      offDagStarted();
      offDagNodeDone();
      offDagCompleted();
      offDagFailed();
      offTaskStarted();
      offTaskUpdated();
      offTaskWaiting();
      offTaskCompleted();
      offTaskFailed();
      offTaskCancelled();
      offTaskSystemMessage();
      offReviewAdded();
      offReviewResolved();
      offDocImported();
      offStatusRail();
      offDegradedEnter();
      offDegradedExit();
      offMemory();
      offExecInterrupted();
      offSidecarState();
      offCLIAuth();
      offNativeRecorderDegraded();
      offWebSearchConfigRequired();
      offDigestArchived();
      offRBRegistered();
      offRBActivated();
      offRBDeactivated();
      offRBModeSwitched();
      offRBRemoved();
      offRBPrimaryChanged();
      offRBDiscordStatus();
      offRBInbound();
      offW3AVerified();
      offW3APollution();
      offW3AExported();
      offW3AImported();
      offW3ATrust();
      offSessionClose();
      offSessionCloseConfirmed();
    };
  }, []);

  // I-1: Periodic refresh of review panel data (every 10s)
  // v3.6: 同時定期刷新 Stop Recovery Cards + Visual Learning 狀態
  useEffect(() => {
    const reviewTimer = window.setInterval(() => {
      loadReviewPanelData();
      refreshStopRecoveryCards();
      refreshVisualLearningState();
      // v3.6.4: 定期同步 Readiness Gate 狀態（Floating Candidates / Missing Slots / scanning）
      refreshReadinessGateState();
    }, 10000);
    return () => window.clearInterval(reviewTimer);
  }, []);

  // I-2: Periodic refresh of skill injections (every 5s)
  useEffect(() => {
    if (!appSessionId) return undefined;
    const skillTimer = window.setInterval(() => {
      callWails(() => GetRecentSkillInjections(appSessionId))
        .then((injections) => setSkillInjections(injections || []))
        .catch(() => {});
    }, 5000);
    // Initial load
    callWails(() => GetRecentSkillInjections(appSessionId))
      .then((injections) => setSkillInjections(injections || []))
      .catch(() => {});
    return () => window.clearInterval(skillTimer);
  }, [appSessionId]);

  // I-1: Load all review panel backend data
  function loadReviewPanelData() {
    // Hook candidates: tag patches, subagent candidates, registry proposals
    Promise.all([
      callWails(GetPendingTagPatches).catch(() => []),
      callWails(GetNewSubagentCandidates).catch(() => []),
      callWails(GetToolRegistryPatchProposals).catch(() => []),
    ]).then(([tagPatches, subagentCandidates, registryProposals]) => {
      setReviewState((prev) => ({
        ...prev,
        hookCandidates: {
          tagPatches: tagPatches || [],
          subagentCandidates: subagentCandidates || [],
          registryProposals: registryProposals || [],
        },
      }));
    });

    // Pending Digest — v3.6: 載入後用 ValidateMemoryItem 驗證每筆記憶項目
    callWails(GetPendingDigest)
      .then(async (digest) => {
        if (digest && digest.items && digest.items.length > 0) {
          const validated = await Promise.all(
            digest.items.map(async (item) => {
              try {
                const result = await callWails(() => ValidateMemoryItem(item));
                // 若驗證失敗，附加警告提示（不暴露工程函式名）
                if (result && !result.valid) {
                  return { ...item, _validationWarning: result.reason || t('review.validationWarning') };
                }
              } catch {
                /* 驗證失敗時不阻擋顯示 */
              }
              return item;
            })
          );
          setReviewState((prev) => ({...prev, pendingDigest: { ...digest, items: validated }}));
        } else {
          setReviewState((prev) => ({...prev, pendingDigest: digest || null}));
        }
      })
      .catch(() => {});

    // Package install queue
    callWails(ListPendingPackages)
      .then((packages) => {
        setReviewState((prev) => ({...prev, pendingPackages: packages || []}));
      })
      .catch(() => {});
  }

  // I-1: Acknowledge a pending digest item
  async function acknowledgePendingItem(itemId, action) {
    try {
      await callWails(() => AcknowledgePendingItem(itemId, action));
      loadReviewPanelData();
    } catch {
      /* best-effort */
    }
  }

  // I-1: Acknowledge with confirmation (high-risk delete)
  async function acknowledgePendingItemConfirm(itemId, action, confirmation) {
    try {
      await callWails(() => AcknowledgePendingItemWithConfirmation(itemId, action, confirmation));
      loadReviewPanelData();
    } catch {
      /* best-effort */
    }
  }

  // I-2: Resolve skill for an action target
  async function resolveSkillAction(actionTarget) {
    if (!appSessionId) return null;
    try {
      const result = await callWails(() => ResolveSkillForAction(actionTarget, appSessionId));
      if (!result) return null;
      if (result.status === 'needs_user_review') {
        // High-risk: populate review state with skill data
        setReviewState((prev) => ({
          ...prev,
          highRisk: {
            id: result.resolveID || result.resolve_id || '',
            status: 'pending',
            title: t('review.highRiskSkillTitle'),
            action: actionTarget,
            skillId: result.selectedSkillID || result.selected_skill_id || '',
            summaryHash: result.summaryHash || result.summary_hash || '',
            permissionSummary: result.permissionSummary || result.permission_summary || '',
            targetPaths: result.targetPaths || result.target_paths || [],
            diff: result.predictedDiff || result.predicted_diff || [],
            expiresIn: result.expiresIn || '05:00',
            source: 'skill',
          },
        }));
      } else if (result.status === 'needs_cli_candidate') {
        // Low-risk ambiguous: populate candidates
        setReviewState((prev) => ({
          ...prev,
          lowRiskAmbiguity: {
            id: result.resolveID || result.resolve_id || '',
            dismissedUntil: '',
            selectedSkillId: result.selectedSkillID || result.selected_skill_id || '',
            candidates: (result.candidates || []).map((c) => ({
              id: c.skillID || c.skill_id || c.id || '',
              reason: c.matchReason || c.match_reason || c.reason || '',
              score: c.score || 0,
              risk: c.risk || 'low',
            })),
          },
        }));
      } else if (result.status === 'auto_selected') {
        // #I-207: auto-selected 也算首次注入，觸發初次說明卡
        if (!skillFirstUseExplained) {
          setShowSkillFirstUseCard(true);
        }
      }
      return result;
    } catch {
      return null;
    }
  }

  // I-2: Build skill context after user confirms high-risk
  async function confirmSkillBuild(resolveId) {
    if (await approveDagNode(resolveId)) return;
    if (!appSessionId) return;
    try {
      await callWails(() => BuildSkillContext(resolveId, appSessionId));
      // Refresh injections
      const injections = await callWails(() => GetRecentSkillInjections(appSessionId));
      setSkillInjections(injections || []);
      // #I-207: 第一次注入成功時顯示初次使用說明卡
      if (!skillFirstUseExplained) {
        setShowSkillFirstUseCard(true);
      }
      // Clear high-risk state
      setReviewState((prev) => ({
        ...prev,
        highRisk: {...prev.highRisk, status: 'approved'},
      }));
      setPendingTaskReview(null);
    } catch {
      /* best-effort */
    }
  }

  async function finishComposerExecution({resp, payload, apiAdapter, traceId, conversationId, clearPendingTimers}) {
    clearPendingTimers?.();
    const cliResp = await applyComposerBuiltInSideEffects(normalizeCLIResponse(resp));
    console.log('[CLI_MONITOR] frontend raw resp -> normalized', {traceId, resp, cliResp});
    postDebugTrace(apiAdapter ? 'ui.composer.after.SendAPIMessage' : 'ui.composer.after.SendCLIMessage', traceId, {response: cliResp || null});
    refreshReadinessGateState();
    setChatCliLog((prev) => ({
      ...(prev || {payload}),
      status: cliResp?.error ? 'error' : 'done',
      response: cliResp || null,
      error: cliResp?.error || null,
      finished_at: new Date().toISOString(),
    }));
    const visualReplayDirective = extractVisualReplayDirective(cliResp?.text);
    if (!cliResp?.auth_required && !cliResp?.error && visualReplayDirective.shouldReplay) {
      const modelText = visualReplayDirective.text;
      if (modelText) {
        setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, `Ai:${modelText}`));
        persistConversationEntry(conversationId, 'assistant', modelText, traceId).catch(() => {});
      }
      try {
        const plan = visualReplayDirective.tag
          ? await callWails(() => GetLearningReplayPlan(visualReplayDirective.tag))
          : await callWails(GetLastLearningReplayPlan);
        const message = formatLearningReplayPlan(plan);
        setConversationMessages(conversationId, (prev) => (
          modelText ? [...prev, message] : replaceComposerPendingMessage(prev, traceId, message)
        ));
        persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, ''), traceId).catch(() => {});
        await executeLearningReplayWithChat(plan, conversationId, traceId);
      } catch (err) {
        const errorMessage = `Ai:我收到 replay 指令，但讀不到上一段示範：${err?.message || String(err)}`;
        setConversationMessages(conversationId, (prev) => (
          modelText ? [...prev, errorMessage] : replaceComposerPendingMessage(prev, traceId, errorMessage)
        ));
      }
      return;
    }
    const operationIntent = operationIntentFromCLIResponse(cliResp);
    if (!cliResp?.auth_required && !cliResp?.error && operationIntent) {
      postDebugTrace('ui.composer.learning_operation_intent', traceId, {
        ...operationIntent,
        source: 'cli_response',
      });
      await executeLearningOperationIntent(operationIntent, {conversationId, traceId});
      return;
    }
    if (cliResp?.auth_required) {
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, `Ai:${cliResp.text || t('system.authRequired')}`));
    } else if (cliResp?.text) {
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, `Ai:${cliResp.text}`));
      persistConversationEntry(conversationId, 'assistant', cliResp.text, traceId).catch(() => {});
    } else if (cliResp?.error) {
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, `[${t('system.sysLabel')}] ${cliResp.error}`));
    }
  }

  async function executeLearningOperationIntent(operationIntent, {conversationId, traceId}) {
    try {
      const items = await callWails(() => SearchLearningOperations(
        operationIntent.query,
        operationIntent.mode === 'query' ? 8 : 5,
      ));
      const matches = Array.isArray(items) ? items : [];
      if (operationIntent.mode === 'query') {
        const message = formatLearningOperationSearchResults(operationIntent.query, matches);
        setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
        persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, ''), traceId).catch(() => {});
        return;
      }
      const resolved = resolveLearningOperationMatch(matches);
      if (!resolved) {
        const message = formatLearningOperationSearchResults(operationIntent.query, matches, true);
        setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
        persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, ''), traceId).catch(() => {});
        return;
      }
      const plan = await callWails(() => GetLearningReplayPlan(resolved.tag || resolved.run_id));
      const message = formatLearningReplayPlan(plan);
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
      persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, ''), traceId).catch(() => {});
      await executeLearningReplayWithChat(plan, conversationId, traceId);
    } catch (err) {
      const message = `Ai:我找不到符合「${operationIntent.query}」的已保存操作：${err?.message || String(err)}`;
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
    }
  }

  async function enrichStoppedLearningRun(run, traceId) {
    const runId = run?.id || run?.run_id || '';
    if (!runId) return run;
    const adapter = resolveActiveAdapter();
    const adapterId = adapter?.id || activeAdapterId || '';
    const sessionId = appSessionId || '';
    try {
      postDebugTrace('ui.learning_metadata.generate.request', traceId, {
        run_id: runId,
        adapter_id: adapterId,
      });
      const enriched = await callWails(() => GenerateLearningRunMetadata(adapterId, sessionId, runId, traceId));
      postDebugTrace('ui.learning_metadata.generate.ok', traceId, {
        run_id: enriched?.id || enriched?.run_id || runId,
        operation_tag: enriched?.operation_tag || '',
        title: enriched?.title || enriched?.name || '',
      });
      return enriched || run;
    } catch (err) {
      postDebugTrace('ui.learning_metadata.generate.error', traceId, {
        run_id: runId,
        error: err?.message || String(err),
      });
      return run;
    }
  }

  function failComposerExecution({err, payload, apiAdapter, adapter, traceId, conversationId, clearPendingTimers}) {
    clearPendingTimers?.();
    postDebugTrace(apiAdapter ? 'ui.composer.SendAPIMessage.error' : 'ui.composer.SendCLIMessage.error', traceId, {error: err?.message || String(err)});
    const rawErrorMsg = err?.message || String(err);
    const errorMsg = apiAdapter && String(adapter?.kind || '').toLowerCase() === 'local'
      ? t('system.localModelBlocked', { name: adapter?.name || t('settings.localModelDefault'), error: rawErrorMsg })
      : rawErrorMsg;
    setChatCliLog((prev) => ({
      ...(prev || {payload}),
      status: 'error',
      response: null,
      error: errorMsg,
      finished_at: new Date().toISOString(),
    }));
    setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, t('system.sendFail', { error: errorMsg })));
  }

  async function confirmSkillExecutionChoice(choice) {
    const pending = skillExecutionConfirm;
    if (!pending) return;
    if (choice === 'cancel') {
      pending.clearPendingTimers?.();
      setConversationMessages(pending.conversationId, (prev) => replaceComposerPendingMessage(prev, pending.traceId, t('common.cancel')));
      setSkillExecutionConfirm(null);
      return;
    }
    setSkillExecutionConfirm(null);
    try {
      const decision = await callWails(() => ConfirmAndExecuteSkillExecution(
        pending.resolveId,
        pending.sessionId,
        choice,
        pending.adapterId,
        pending.userText,
        pending.traceId,
      ));
      if (decision?.response) {
        await finishComposerExecution({...pending, resp: decision.response});
      } else if (decision?.message) {
        pending.clearPendingTimers?.();
        setConversationMessages(pending.conversationId, (prev) => replaceComposerPendingMessage(prev, pending.traceId, decision.message));
      }
    } catch (err) {
      failComposerExecution({...pending, err});
    }
  }

  // #I-207: 關閉初次使用說明卡並持久化
  async function dismissSkillFirstUseCard() {
    setShowSkillFirstUseCard(false);
    setSkillFirstUseExplained(true);
    try {
      await callWails(MarkSkillFirstUseExplained);
    } catch {
      /* best-effort：即使後端失敗也不影響當次體驗 */
    }
  }

  // I-2: Clear skill context
  async function clearSkillContext(reason) {
    if (!appSessionId) return;
    try {
      await callWails(() => ClearSkillContext(appSessionId, reason));
      const injections = await callWails(() => GetRecentSkillInjections(appSessionId));
      setSkillInjections(injections || []);
    } catch {
      /* best-effort */
    }
  }

  // I-2: Scan skill folder for install preview
  async function scanSkillForInstall(path) {
    try {
      const preview = await callWails(() => ScanSkillFolder(path));
      setSkillScanPreview(preview || null);
      return preview;
    } catch {
      return null;
    }
  }

  // I-2: Confirm skill archive after preview
  async function confirmSkillArchiveInstall(previewId) {
    try {
      await callWails(() => ConfirmSkillArchive(previewId));
      setSkillScanPreview(null);
      // Refresh archived skills
      const skills = await callWails(ListArchivedSkills);
      setArchivedSkills(skills || []);
    } catch {
      /* best-effort */
    }
  }

  // -------------------------------------------------------------------------
  // I-3: Controlled Trust handlers
  // -------------------------------------------------------------------------

  // #I-303 Draft Sandbox — 錄製按鈕啟動 / 停止 / 三選項升格
  async function startDraftSandbox() {
    const conversationId = activeConversationIdRef.current || 'main';
    try {
      const windowHash = `window-${Date.now()}`;
      const sandboxId = await callWails(() => StartDraftSandbox(windowHash));
      setActiveSandboxId(sandboxId);
      try {
        const run = await callWails(() => StartLearningMode(windowHash));
        setVlActiveLearningRun(run);
        setVlLearningActive(true);
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          `[系統] 示範開始。${run?.id ? `Run: ${run.id}。` : ''}請操作一次你要教我的流程。`,
        ]);
      } catch (error) {
        const detail = error?.message || String(error || '');
        const activeRun = detail.includes('already recording')
          ? await callWails(GetActiveLearningRun).catch(() => null)
          : null;
        if (activeRun) {
          setVlActiveLearningRun(activeRun);
          setVlLearningActive(true);
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            `[系統] 已接回仍在進行中的示範。${activeRun?.id ? `Run: ${activeRun.id}。` : ''}請先停止目前示範。`,
          ]);
        } else {
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            `[系統] 示範開始，但 Visual Learning 後端沒有成功啟動記錄。${detail ? ` ${detail}` : ''}`,
          ]);
        }
      }
      setRecordingEnabled(true);
      setSandboxStopOptions(null);
    } catch (error) {
      const detail = error?.message || String(error || '');
      setConversationMessages(conversationId, (prev) => [
        ...prev,
        `[系統] 示範啟動失敗。${detail ? ` ${detail}` : ''}`,
      ]);
    }
  }

  async function stopDraftSandbox(reason) {
    if (!activeSandboxId) return;
    const conversationId = activeConversationIdRef.current || 'main';
    try {
      if (vlLearningActive) {
        try {
          const traceId = makeDebugTraceID('learning-metadata');
          const run = await callWails(StopLearningMode);
          const namedRun = await enrichStoppedLearningRun(run, traceId);
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            formatLearningOperationLearned(namedRun),
          ]);
        } catch (error) {
          const detail = error?.message || String(error || '');
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            `[系統] 示範結束，但後端停止記錄時回報錯誤。${detail ? ` ${detail}` : ''}`,
          ]);
        }
        setVlLearningActive(false);
        setVlActiveLearningRun(null);
      }
      await callWails(() => StopDraftSandbox(activeSandboxId, reason || 'user_stop'));
      setSandboxStopOptions({sandboxId: activeSandboxId, reason: reason || 'user_stop'});
      setRecordingEnabled(false);
    } catch {
      setRecordingEnabled(false);
    }
  }

  async function promoteDraft(promotion) {
    if (!sandboxStopOptions) return;
    const conversationId = activeConversationIdRef.current || 'main';
    try {
      await callWails(() => PromoteDraftToPending(sandboxStopOptions.sandboxId, promotion));
    } catch {
      /* best-effort */
    }
    if (promotion === 'formal_review') {
      let plan = null;
      try {
        plan = await callWails(GetLastLearningReplayPlan);
        const message = formatLearningReplayPlan(plan);
        setConversationMessages(conversationId, (prev) => [...prev, message]);
        persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, '')).catch(() => {});
      } catch (error) {
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          `Ai:我還讀不到上一段示範：${error?.message || String(error)}`,
        ]);
      }
      setSandboxStopOptions(null);
      setActiveSandboxId(null);
      if (plan) {
        await executeLearningReplayWithChat(plan, conversationId);
      }
      return;
    } else if (promotion === 'pending_candidate') {
      setConversationMessages(conversationId, (prev) => [...prev, '[系統] 已把這次示範存成候選流程。']);
    }
    setSandboxStopOptions(null);
    setActiveSandboxId(null);
  }

  function dismissSandboxOptions() {
    const conversationId = activeConversationIdRef.current || 'main';
    setConversationMessages(conversationId, (prev) => [...prev, '[系統] 已放棄這次示範。']);
    setSandboxStopOptions(null);
    setActiveSandboxId(null);
  }

  async function executeLearningReplayWithChat(plan, conversationId, traceId = '') {
    const steps = Array.isArray(plan?.steps) ? plan.steps : [];
    if (!steps.length) return;
    setPendingLearningReplayStartConfirm({plan, conversationId, traceId, createdAt: Date.now()});
  }

  async function startConfirmedLearningReplay() {
    const pending = pendingLearningReplayStartConfirm;
    if (!pending || learningReplayExecutingRef.current) return;
    setPendingLearningReplayStartConfirm(null);
    await runLearningReplayPlanWithChat(pending.plan, pending.conversationId, pending.traceId);
  }

  function cancelLearningReplayStartConfirm() {
    const pending = pendingLearningReplayStartConfirm;
    setPendingLearningReplayStartConfirm(null);
    if (!pending) return;
    const cancelMessage = 'Ai:Replay 尚未執行；我已保留上面的安全 plan，確認內容正確後可以再輸入「請按照我剛剛的示範」。';
    setConversationMessages(pending.conversationId, (prev) => [...prev, cancelMessage]);
    persistConversationEntry(pending.conversationId, 'assistant', cancelMessage.replace(/^Ai:/, ''), pending.traceId).catch(() => {});
  }

  async function runLearningReplayPlanWithChat(plan, conversationId, traceId = '') {
    await delayLearningReplay(350);
    learningReplayExecutingRef.current = true;
    try {
      const result = await executeLearningReplayPlan(plan);
      const resultMessage = formatLearningReplayExecutionResult(result);
      setConversationMessages(conversationId, (prev) => [...prev, resultMessage]);
      persistConversationEntry(conversationId, 'assistant', resultMessage.replace(/^Ai:/, ''), traceId).catch(() => {});
      if (result?.pending_confirmation) {
        setPendingLearningReplayConfirm({
          plan,
          conversationId,
          traceId,
          result,
          stoppedStepOffset: Math.max(0, result.steps.length - 1),
          createdAt: Date.now(),
        });
      }
    } finally {
      learningReplayExecutingRef.current = false;
    }
  }

  async function continueLearningReplayAfterConfirm() {
    const pending = pendingLearningReplayConfirm;
    if (!pending || learningReplayExecutingRef.current) return;
    const steps = Array.isArray(pending.plan?.steps) ? pending.plan.steps : [];
    const previewResult = pending.result?.steps?.[pending.stoppedStepOffset];
    const originalStep = steps[pending.stoppedStepOffset];
    if (!originalStep || !previewResult) {
      setPendingLearningReplayConfirm(null);
      return;
    }
    setPendingLearningReplayConfirm(null);
    learningReplayExecutingRef.current = true;
    try {
      const confirmedStep = {
        ...originalStep,
        x: Number.isFinite(Number(previewResult.x)) ? Number(previewResult.x) : originalStep.x,
        y: Number.isFinite(Number(previewResult.y)) ? Number(previewResult.y) : originalStep.y,
        windows_anchor: null,
      };
      const confirmedResult = await executeLearningReplayStep(confirmedStep);
      const nextResult = await executeLearningReplayPlan(pending.plan, {
        startOffset: pending.stoppedStepOffset + 1,
        initialResults: [
          ...(pending.result?.steps || []).slice(0, pending.stoppedStepOffset),
          {...confirmedResult, confirmed_after_preview: true},
        ],
      });
      const message = formatLearningReplayExecutionResult(nextResult);
      setConversationMessages(pending.conversationId, (prev) => [...prev, message]);
      persistConversationEntry(pending.conversationId, 'assistant', message.replace(/^Ai:/, ''), pending.traceId).catch(() => {});
      if (nextResult?.pending_confirmation) {
        setPendingLearningReplayConfirm({
          plan: pending.plan,
          conversationId: pending.conversationId,
          traceId: pending.traceId,
          result: nextResult,
          stoppedStepOffset: Math.max(0, nextResult.steps.length - 1),
          createdAt: Date.now(),
        });
      }
    } finally {
      learningReplayExecutingRef.current = false;
    }
  }

  function cancelLearningReplayConfirm() {
    setPendingLearningReplayConfirm(null);
  }

  // #I-301 Contextual Risk Override
  async function enableContextualOverride(scopeJson, allowedRisk) {
    try {
      const overrideId = await callWails(() => EnableContextualRiskOverride(scopeJson, allowedRisk));
      setActiveOverrides((prev) => [...prev, {id: overrideId, allowedRisk, createdAt: Date.now()}]);
      return overrideId;
    } catch {
      return null;
    }
  }

  async function enableWorkflowTrust(scopeJson, allowedRisk, hours) {
    try {
      const overrideId = await callWails(() => EnableWorkflowTrustForHours(scopeJson, allowedRisk, hours));
      setActiveOverrides((prev) => [...prev, {id: overrideId, allowedRisk, hours, createdAt: Date.now()}]);
      return overrideId;
    } catch {
      return null;
    }
  }

  async function disableContextualOverride(overrideId) {
    try {
      await callWails(() => DisableContextualRiskOverride(overrideId));
      setActiveOverrides((prev) => prev.filter((o) => o.id !== overrideId));
    } catch {
      /* best-effort */
    }
  }

  // #I-302 Trusted Session Scope
  async function enableTrustedSession(workspaceId, dagRunId) {
    if (!appSessionId) return;
    const windowHash = `window-${Date.now()}`;
    try {
      await callWails(() => EnableTrustedSessionScope(appSessionId, workspaceId || '', dagRunId || '', windowHash));
      setTrustedSessionActive(true);
      setTrustedSessionExpired(false);
    } catch {
      /* best-effort */
    }
  }

  function handleTrustedSessionExpired(choice) {
    // choice: 'reenable' | 'allow_once' | 'cancel'
    if (choice === 'reenable') {
      enableTrustedSession('', '');
    } else {
      setTrustedSessionExpired(false);
      if (choice === 'cancel') {
        setTrustedSessionActive(false);
      }
    }
  }

  // #I-304 Trust DOM & Click
  async function toggleTrustDomClick(enabled) {
    try {
      await callWails(() => SetTrustDomAndClick(enabled));
      setTrustDomClickEnabled(enabled);
    } catch {
      /* best-effort */
    }
  }

  // #I-305 Device Trust Profile
  async function loadDeviceProfile(profileId) {
    try {
      const profile = await callWails(() => GetDeviceTrustProfile(profileId || 'default'));
      setDeviceProfile(profile || null);
    } catch {
      setDeviceProfile(null);
    }
  }

  // -------------------------------------------------------------------------
  // I-4: Browser Preference & UI Settings handlers
  // -------------------------------------------------------------------------

  // #I-401 Set browser preference (from Settings)
  async function saveBrowserPreference(browser, profilePath) {
    try {
      const result = await callWails(() => SetBrowserPreference(browser, profilePath || ''));
      // Reload pref
      const pref = await callWails(GetBrowserPreference);
      setBrowserPref(pref || null);
      // If Safari was selected, check for runtime notice
      if (browser === 'safari' || browser === 'Safari') {
        const notice = await callWails(GetSafariRuntimeNotice);
        if (notice && notice.message) {
          setSafariNotice(notice);
          setShowSafariNotice(true);
        }
      }
      return result;
    } catch {
      return null;
    }
  }

  // #I-402 Dismiss Safari notice
  function dismissSafariNotice() {
    setShowSafariNotice(false);
  }

  // #I-402 Check Safari notice (called before profile reuse tasks)
  async function checkSafariNotice() {
    try {
      const notice = await callWails(GetSafariRuntimeNotice);
      if (notice && notice.message) {
        setSafariNotice(notice);
        setShowSafariNotice(true);
        return notice;
      }
    } catch { /* best-effort */ }
    return null;
  }

  // #I-403 Apply UI Style Diff with preview
  async function previewStyleDiff(diffJson) {
    setStyleDiffPreview({diffJson, pending: true});
    setStyleDiffError('');
  }

  async function confirmStyleDiff() {
    if (!styleDiffPreview?.diffJson) return;
    try {
      const result = await callWails(() => ApplyUIStyleDiff(styleDiffPreview.diffJson));
      if (result) {
        setSettingsState((prev) => ({
          ...prev,
          panel: normalizePanelSettings(panelFromUISettings(result, prev.panel)),
        }));
      }
      setStyleDiffPreview(null);
      setStyleDiffError('');
    } catch (err) {
      setStyleDiffError(err?.message || t('styleDiff.applyFail'));
      setStyleDiffPreview(null);
    }
  }

  function cancelStyleDiff() {
    setStyleDiffPreview(null);
    setStyleDiffError('');
  }

  useEffect(() => {
    const timer = window.setInterval(() => {
      callWails(PollStatusRail)
        .then((statusRail) => {
          if (!statusRail) return;
          // 上方互動輸出快照：輪詢只刷新 greeting/statusRail。
          setState((prev) => ({
            ...prev,
            greeting: manualGreetingLockedRef.current ? prev.greeting : (statusRail.text || prev.greeting),
            statusRail: manualGreetingLockedRef.current ? {...statusRail, text: prev.greeting} : statusRail,
          }));
        })
        .catch(() => {});
    }, 1000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    if (!draggedTool) return undefined;
    const showDragActions = () => {
      window.setTimeout(() => {
        setDragActionTool((current) => current || draggedTool);
        setDraggedTool(null);
      }, 120);
    };
    window.addEventListener('blur', showDragActions);
    return () => window.removeEventListener('blur', showDragActions);
  }, [draggedTool]);

  useEffect(() => {
    const refreshIfVisible = () => {
      if (document.visibilityState !== 'hidden') {
        refreshReferenceFiles().catch(() => {});
      }
    };
    const timer = window.setInterval(refreshIfVisible, 5000);
    window.addEventListener('focus', refreshIfVisible);
    document.addEventListener('visibilitychange', refreshIfVisible);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener('focus', refreshIfVisible);
      document.removeEventListener('visibilitychange', refreshIfVisible);
    };
  }, []);

  useEffect(() => {
    if (!learningEnabled) return undefined;

    let idleTimer = window.setTimeout(markLearningDigestReady, learningIdleDelayMs);
    const resetIdleTimer = () => {
      window.clearTimeout(idleTimer);
      idleTimer = window.setTimeout(markLearningDigestReady, learningIdleDelayMs);
    };
    const activityEvents = ['keydown', 'pointerdown', 'mousemove', 'wheel', 'touchstart'];
    activityEvents.forEach((eventName) => window.addEventListener(eventName, resetIdleTimer, {passive: true}));
    window.addEventListener('beforeunload', confirmLearningBackground);

    return () => {
      window.clearTimeout(idleTimer);
      activityEvents.forEach((eventName) => window.removeEventListener(eventName, resetIdleTimer));
      window.removeEventListener('beforeunload', confirmLearningBackground);
    };
  }, [learningEnabled]);

  useEffect(() => {
    try {
      OnFileDrop((x, y, paths) => {
        if (referenceInternalDragRef.current || Date.now() < referenceDropSuppressUntilRef.current) return;
        const nativePaths = normalizeReferenceImportPaths(paths);
        if (!nativePaths.length) return;
        importReferencePaths(nativePaths);
        if (shouldProbeDroppedInstallPackage(nativePaths)) {
          detectDroppedInstallPackage(nativePaths[0]);
        }
      }, false);
      return () => OnFileDropOff();
    } catch {
      return undefined;
    }
  }, []);

  useEffect(() => {
    setReferenceFiles((current) => {
      const cleaned = current.filter((file) => !isInvalidReferencePlaceholder(file));
      return cleaned.length === current.length ? current : cleaned;
    });
  }, [referenceFiles]);

  async function detectDroppedInstallPackage(path) {
    try {
      const preview = await callWails(() => PreviewSubPackage(path));
      if (preview?.source_system_code) {
        setInstallCandidate({
          type: 'subagent',
          typeLabel: 'subagent',
          name: preview.display_name || preview.source_system_code || t('subagent.unnamedSubagent'),
          path,
          preview,
        });
        return;
      }
    } catch {
      // 不是 sub 包時繼續試其他安裝類型。
    }
    try {
      const preview = await callWails(() => ScanSkillFolder(path));
      const resources = preview?.Resources || preview?.resources || [];
      const hasSkillMarker = preview?.HasManifest || preview?.has_manifest ||
        resources.some((item) => ['SKILL.md', 'skill_manifest.json'].includes(item?.Name || item?.name));
      if ((preview?.PreviewID || preview?.preview_id) && hasSkillMarker) {
        setInstallCandidate({
          type: 'skill',
          typeLabel: 'skill',
          name: preview.DisplayName || preview.display_name || preview.SkillID || preview.skill_id || t('package.unnamedSkill'),
          path,
          preview,
        });
        return;
      }
      setToolResult({toolId: 'package-detect', ok: false, message: t('package.unrecognized')});
    } catch {
      setToolResult({toolId: 'package-detect', ok: false, message: t('package.unrecognized')});
    }
  }

  async function confirmInstallCandidate() {
    if (!installCandidate) return;
    try {
      if (installCandidate.type === 'subagent') {
        const exportDir = installCandidate.preview?.export_dir;
        if (!exportDir) return;
        const result = await callWails(() => ImportSubHandler(exportDir));
        setInstallCandidate(null);
        setSubImportResult(result || null);
        await refreshAvailableAdapters();
        return;
      }
      if (installCandidate.type === 'skill') {
        const previewId = installCandidate.preview?.PreviewID || installCandidate.preview?.preview_id;
        if (!previewId) return;
        await callWails(() => ConfirmSkillArchive(previewId));
        setInstallCandidate(null);
        const skills = await callWails(ListArchivedSkills);
        setArchivedSkills(skills || []);
        return;
      }
    } catch (err) {
      setToolResult({toolId: 'package-install', ok: false, message: String(err)});
    }
  }

  async function resolveSubImportConflicts(strategy) {
    const conflicts = subImportResult?.tool_conflicts || [];
    if (!conflicts.length || strategy === 'keep_all') {
      setSubImportResult(null);
      return;
    }
    try {
      await callWails(() => ResolveImportToolConflicts(JSON.stringify({strategy, conflicts})));
      setSubImportResult(null);
    } catch (err) {
      setToolResult({toolId: 'sub-import', ok: false, message: String(err)});
    }
  }

  async function saveSummaryModelPatch(patch) {
    const current = summaryModelSettings || {source: 'cli_adapter', modelId: 'current', endpoint: '', alwaysUse: false};
    const next = {...current, ...patch};
    const saved = await callWails(() => SaveSummaryModelSettings(next));
    setSummaryModelSettings(saved || next);
  }

  async function saveVoiceSettingsPatch(patch) {
    const current = voiceState?.settings || {languageMode: 'auto', manualLanguage: '', debugMode: false, commandMode: false, whisperBinPath: '', modelPath: ''};
    const next = {...current, ...patch};
    const saved = await callWails(() => SaveVoiceSettings(next));
    setVoiceState(saved || {...voiceState, settings: next});
  }

  async function installVoiceBaseModel() {
    setVoiceInstallBusy(true);
    setVoiceError('');
    setVoiceStatus(t('voice.downloadingBaseModel'));
    try {
      const next = await callWails(InstallVoiceBaseModel);
      setVoiceState(next || null);
      setVoiceStatus(t('voice.baseModelInstalled'));
    } catch (err) {
      setVoiceError(err?.message || String(err));
      setVoiceStatus('');
    } finally {
      setVoiceInstallBusy(false);
    }
  }

  async function removeVoiceBaseModel() {
    setVoiceInstallBusy(true);
    setVoiceError('');
    try {
      const next = await callWails(RemoveVoiceBaseModel);
      setVoiceState(next || null);
      setVoiceStatus(t('voice.baseModelRemoved'));
    } catch (err) {
      setVoiceError(err?.message || String(err));
    } finally {
      setVoiceInstallBusy(false);
    }
  }

  async function clearVoiceDebug() {
    setVoiceError('');
    try {
      const next = await callWails(ClearVoiceDebug);
      setVoiceState(next || null);
      setVoiceStatus(t('voice.debugCleared'));
    } catch (err) {
      setVoiceError(err?.message || String(err));
    }
  }

  async function startVoiceRecording() {
    if (voiceRecording || voiceBusy) return;
    setVoiceError('');
    setVoiceStatus(t('voice.recording'));
    try {
      const stream = await navigator.mediaDevices.getUserMedia({audio: {channelCount: 1, noiseSuppression: true, echoCancellation: true}});
      const AudioContextClass = window.AudioContext || window.webkitAudioContext;
      const audioContext = new AudioContextClass();
      const source = audioContext.createMediaStreamSource(stream);
      const processor = audioContext.createScriptProcessor(4096, 1, 1);
      const monitor = audioContext.createGain();
      const chunks = [];

      // Web Audio captures mono PCM so the backend can hand whisper.cpp a plain wav.
      processor.onaudioprocess = (event) => {
        chunks.push(new Float32Array(event.inputBuffer.getChannelData(0)));
      };
      monitor.gain.value = 0;
      source.connect(processor);
      processor.connect(monitor);
      monitor.connect(audioContext.destination);
      if (audioContext.state === 'suspended') {
        await audioContext.resume();
      }
      const autoStopTimer = setTimeout(() => {
        stopVoiceRecording();
      }, MAX_VOICE_RECORDING_MS);
      voiceRecorderRef.current = {stream, audioContext, source, processor, monitor, chunks, sampleRate: audioContext.sampleRate, startedAt: Date.now(), autoStopTimer};
      setVoiceRecording(true);
    } catch (err) {
      setVoiceRecording(false);
      setVoiceStatus('');
      setVoiceError(err?.message || t('voice.noMicPermission'));
    }
  }

  async function stopVoiceRecording() {
    const recorder = voiceRecorderRef.current;
    if (!recorder) return;
    voiceRecorderRef.current = null;
    const recordingDuration = Date.now() - recorder.startedAt;
    if (recorder.autoStopTimer) clearTimeout(recorder.autoStopTimer);
    setVoiceRecording(false);
    setVoiceBusy(true);
    setVoiceStatus(t('voice.transcribing'));
    try {
      await waitForVoiceChunks(recorder);
      recorder.processor.disconnect();
      recorder.monitor?.disconnect?.();
      recorder.source.disconnect();
      recorder.stream.getTracks().forEach((track) => track.stop());
      await recorder.audioContext.close();
      if (recordingDuration < MIN_VOICE_RECORDING_MS) {
        throw new Error(t('voice.tooShort'));
      }
      if (!recorder.chunks.length) throw new Error(t('voice.noAudio'));
      const wavBlob = encodeWav(recorder.chunks, recorder.sampleRate);
      const audioBase64 = await blobToBase64(wavBlob);
      const result = await callWails(() => TranscribeVoiceWAV(audioBase64, 'audio/wav'));
      const text = String(result?.text || '').trim();
      if (!text) throw new Error(t('voice.noText'));
      if (recordingDuration <= VOICE_BG_THRESHOLD_MS) {
        await applyVoiceTranscript(text);
      } else {
        setVoiceBackgroundResult({ text, duration: recordingDuration });
        setVoiceStatus(t('voice.transcriptReady'));
      }
      callWails(GetVoiceSettings).then((settings) => setVoiceState(settings || null)).catch(() => {});
    } catch (err) {
      setVoiceError(err?.message || String(err));
      setVoiceStatus('');
    } finally {
      setVoiceBusy(false);
    }
  }

  async function cancelVoiceRecording() {
    const recorder = voiceRecorderRef.current;
    if (!recorder) return;
    voiceRecorderRef.current = null;
    if (recorder.autoStopTimer) clearTimeout(recorder.autoStopTimer);
    recorder.processor.disconnect();
    recorder.monitor?.disconnect?.();
    recorder.source.disconnect();
    recorder.stream.getTracks().forEach((track) => track.stop());
    await recorder.audioContext.close().catch(() => {});
    setVoiceRecording(false);
    setVoiceBusy(false);
    setVoiceStatus('');
  }

  async function applyVoiceTranscript(text) {
    if (voiceState?.settings?.commandMode) {
      const route = await callWails(() => RouteVoiceCommand(text));
      if (route?.matched) {
        if (route.action === 'stop_active_job') {
          await callWails(StopSidecar);
          setVoiceStatus(t('voice.cliStopped'));
          return;
        }
        if (route.action === 'resume_active_job') {
          await callWails(RestartSidecar);
          setVoiceStatus(t('voice.cliRestarted'));
          return;
        }
        if (route.action === 'append_readonly_constraint') {
          setDraft((prev) => mergeDraftText(prev, t('voice.readonlyConstraintText')));
          setVoiceStatus(t('voice.readonlyConstraintAdded'));
          return;
        }
      }
    }
    await submitComposerText(text);
    setVoiceStatus(t('voice.sent'));
  }

  function handleVoiceBackgroundAction(action) {
    if (!voiceBackgroundResult) return;
    const text = voiceBackgroundResult.text;
    setVoiceBackgroundResult(null);
    if (action === 'send') {
      submitComposerText(text);
      setVoiceStatus(t('voice.sent'));
    } else if (action === 'draft') {
      setDraft((prev) => (prev ? prev + '\n' + text : text));
      setVoiceStatus(t('voice.drafted'));
    } else {
      setVoiceStatus(t('voice.discarded'));
    }
  }

  async function rotateGreeting() {
    const next = pickRotatingGreeting(state.greeting);
    manualGreetingLockedRef.current = true;
    if (next?.expression) {
      setManualAvatarState(next.expression);
    }
    setState((prev) => ({...prev, greeting: next.text, statusRail: {...prev.statusRail, text: next.text, layer: 'L1'}}));
  }

  function patchDagRun(runId, updater) {
    setDagRun((current) => {
      if (!current || current.id !== runId) return current;
      return updater(current);
    });
  }

  function patchDagNode(runId, nodeId, patch) {
    patchDagRun(runId, (current) => ({
      ...current,
      nodes: current.nodes.map((node) => (node.id === nodeId ? {...node, ...patch} : node)),
    }));
  }

  function addDagNodeSummary(runId, summary) {
    patchDagRun(runId, (current) => ({
      ...current,
      summaries: [...current.summaries.filter((item) => item.nodeId !== summary.nodeId), summary],
    }));
  }

  function syncTaskProgressRun(rawRun, options = {}) {
    const mappedRun = mapBackendTaskRun(rawRun);
    if (!mappedRun) return null;
    const currentRun = dagRunRef.current;
    if (options.preserveNewer && shouldKeepNewerTaskRun(currentRun, mappedRun)) {
      return currentRun;
    }
    dagRunRef.current = mappedRun;
    setDagRun(mappedRun);
    if (mappedRun.status !== 'waiting_review') {
      setPendingTaskReview(null);
      const reviewIds = new Set((mappedRun.nodes || []).map((node) => node.reviewId).filter(Boolean));
      if (reviewIds.size > 0) {
        const nextStatus = mappedRun.status === 'completed' ? 'approved' : 'rejected';
        setReviewState((prev) => {
          if (!reviewIds.has(prev.highRisk?.id)) return prev;
          return {
            ...prev,
            highRisk: {
              ...prev.highRisk,
              status: nextStatus,
              expiresIn: '',
              note: mappedRun.interruptReason || prev.highRisk.note,
            },
          };
        });
        setReviewPopup((current) => current === 'risk' ? null : current);
      } else {
        setReviewState((prev) => {
          if (prev.highRisk?.source !== 'task_progress') return prev;
          return {
            ...prev,
            highRisk: {...fallbackReviewState.highRisk},
          };
        });
        setReviewPopup((current) => current === 'risk' ? null : current);
      }
    }
    return mappedRun;
  }

  function syncTaskReviewState(rawRun) {
    const run = mapBackendTaskRun(rawRun);
    const activeNode = run?.nodes?.find((node) => node.status === 'waiting_review' || node.id === run.currentNodeId);
    if (!run || run.status !== 'waiting_review' || !activeNode?.reviewId) return;
    const nextReview = {
      id: activeNode.reviewId,
      title: activeNode.title || activeNode.action || '需要你確認這一步',
      action: activeNode.action || activeNode.title || '',
      impact: activeNode.target || activeNode.action || '目前步驟會執行外部工具或模型。',
      tool: activeNode.tool || activeNode.actionCode || '目前模型',
      reason: '執行前需要使用者確認',
    };
    setPendingTaskReview(nextReview);
    setReviewState((prev) => ({
      ...prev,
      highRisk: {
        id: nextReview.id,
        status: 'pending',
        title: '需要你確認這一步',
        action: nextReview.title,
        skillId: nextReview.tool,
        summaryHash: run.hookRunId || activeNode.traceHash || '',
        permissionSummary: nextReview.impact,
        targetPaths: [nextReview.impact].filter(Boolean),
        diff: [`風險等級：${activeNode.risk || 'high'}`, '執行前需要使用者確認'],
        expiresIn: '等待確認',
        source: 'task_progress',
      },
    }));
  }

  async function recordDagStepTrace(run, hookRunId, node, startedAt, endedAt, status = 'ok') {
    if (!hookRunId) return;
    const tracePayload = JSON.stringify({
      step_id: node.id,
      outline_step_id: `${run.outlineId}-${node.id}`,
      action: node.action,
      target: run.title,
      tool_used: node.tool,
      started_at: startedAt,
      ended_at: endedAt,
      result_status: status,
      risk_level: node.risk,
    });
    try {
      await callWails(() => RecordStepTrace(hookRunId, tracePayload));
    } catch {
      /* Hook evidence is best-effort until the real DAG runtime owns retries. */
    }
  }

  function completeDagNode(run, hookRunId, node, startedAt, endedAt) {
    const durationMs = Math.max(0, new Date(endedAt).getTime() - new Date(startedAt).getTime());
    patchDagNode(run.id, node.id, {status: 'completed', endedAt, durationMs});
    addDagNodeSummary(run.id, {
      nodeId: node.id,
      title: node.title,
      status: 'completed',
      risk: node.risk,
      generatedAt: endedAt,
      durationMs,
      text: t('dag.nodeCompleted', { title: node.title, duration: (durationMs / 1000).toFixed(1) }),
    });
    recordDagStepTrace(run, hookRunId, node, startedAt, endedAt);
  }

  function blockHighRiskDagNode(run, node, startedAt) {
    patchDagNode(run.id, node.id, {status: 'blocked', startedAt});
    patchDagRun(run.id, (current) => ({...current, status: 'blocked', currentNodeId: node.id}));
    setReviewState((prev) => ({
      ...prev,
      highRisk: {
        id: node.id,
        status: 'pending',
        title: t('dag.highRiskNodeTitle'),
        action: `${run.title} · ${node.action}`,
        skillId: node.tool,
        summaryHash: run.hookRunId || 'hook pending',
        permissionSummary: t('dag.highRiskNodePermission'),
        targetPaths: [t('dag.highRiskNodeTarget')],
        diff: ['block: dependent nodes paused', 'await: explicit user confirmation'],
        expiresIn: t('dag.waitingConfirm'),
        source: 'legacy_dag',
      },
    }));
  }

  function finishDagRun(runId, hookRunId) {
    callWails(() => GetHookSummary(hookRunId))
      .then((summary) => {
        patchDagRun(runId, (current) => ({
          ...current,
          status: 'completed',
          summaryHash: summary?.summaryHash || summary?.summary_hash || current.summaryHash,
        }));
        loadReviewPanelData();
      })
      .catch(() => {
        patchDagRun(runId, (current) => ({...current, status: 'completed'}));
      });
  }

  function runDagNodes(run, hookRunId, startIndex = 0) {
    const node = run.nodes[startIndex];
    if (!node) {
      finishDagRun(run.id, hookRunId);
      return;
    }

    const startedAt = new Date().toISOString();
    patchDagRun(run.id, (current) => ({...current, status: 'running', currentNodeId: node.id}));
    patchDagNode(run.id, node.id, {status: 'running', startedAt});

    window.setTimeout(() => {
      if (node.risk === 'high') {
        blockHighRiskDagNode({...run, hookRunId}, node, startedAt);
        return;
      }
      const endedAt = new Date().toISOString();
      completeDagNode(run, hookRunId, node, startedAt, endedAt);
      runDagNodes(run, hookRunId, startIndex + 1);
    }, 900 + startIndex * 180);
  }

  // 任務進度 v1：由後端 DAG runtime 建立、持久化並推送節點狀態。
  async function startDagForMessage(text, options = {}) {
    // #5 Degraded Mode guard: 降級時禁止啟動新 DAG run
    if (degradedState.active) {
      setToolResult({toolId: 'dag', ok: false, message: t('dag.degradedPaused')});
      return;
    }
    const adapter = resolveActiveAdapter();
    const adapterID = adapter?.id || adapter?.name || '';
    const modelID = adapterModelChoices?.[adapterID] || adapter?.model_id || adapter?.modelID || adapter?.model || '';
    const startingRun = {
      id: `task-starting-${Date.now()}`,
      outlineId: '',
      hookRunId: '',
      title: text.length > 28 ? `${text.slice(0, 28)}...` : text,
      status: 'starting',
      createdAt: new Date().toISOString(),
      nodes: [],
      summaries: [],
      currentNodeId: '',
      summaryHash: '',
    };
    dagRunRef.current = startingRun;
    setDagRun(startingRun);
    setActiveToolTabs((current) => ({...current, left: 'flow'}));
    try {
      const run = await callWails(() => StartTaskProgress(text, adapterID, modelID, appSessionId || ''));
      syncTaskProgressRun(run, {preserveNewer: true});
      if (options.conversationId && options.traceId) {
        setConversationMessages(options.conversationId, (prev) => replaceComposerPendingMessage(
          prev,
          options.traceId,
          `Ai:任務已開始：${run?.title || '任務進度'}`,
        ));
      }
    } catch (error) {
      const errorMessage = error?.message || String(error);
      const clarification = plannerClarificationFromError(error);
      setDagRun((current) => current ? {
        ...current,
        status: 'failed',
        summaries: [{
          nodeId: 'planner',
          title: '任務規劃失敗',
          status: 'failed',
          risk: 'low',
          text: errorMessage,
        }],
      } : current);
      if (!clarification) {
        setToolResult({toolId: 'dag', ok: false, message: errorMessage});
      }
      if (options.conversationId && options.traceId) {
        setConversationMessages(options.conversationId, (prev) => replaceComposerPendingMessage(
          prev,
          options.traceId,
          clarification ? `Ai:${clarification}` : `[${t('system.sysLabel')}] ${errorMessage}`,
        ));
      }
    }
  }

  async function cancelActiveTaskProgress(reason = 'user_stop') {
    try {
      const run = await callWails(() => CancelActiveTaskProgress(reason));
      if (run) syncTaskProgressRun(run);
      await refreshReviewCards();
    } catch (error) {
      setToolResult({toolId: 'dag', ok: false, message: error?.message || String(error)});
    }
  }

  // ── v3.6 Source Trust 內部接線 ──
  // 將後端 SourceTrustEvidence 轉為使用者可見的中文提示
  function sourceTrustToHint(evidence) {
    if (!evidence) return null;
    const label = evidence.source_trust_label;
    const allowlist = evidence.allowlist_status;
    // 已在信任清單
    if (allowlist === 'active') {
      return { level: 'trusted', text: t('sourceTrust.trusted') };
    }
    // 高影響任務 + 需要 review
    if (evidence.is_high_impact && evidence.review_required) {
      return { level: 'high_impact', text: t('sourceTrust.highImpact') };
    }
    // 低信任
    if (label === 'LOW_TRUST') {
      return { level: 'untrusted', text: t('sourceTrust.untrusted') };
    }
    // 未驗證
    if (label === 'UNVERIFIED' || label === 'USER_GENERATED' || label === 'PENDING_SOURCE_REVIEW') {
      return { level: 'unverified', text: t('sourceTrust.unverified') };
    }
    // 其他（已驗證等）不需要提示
    return null;
  }

  // 對輸入文字中的 URL 執行來源分類（非阻塞）
  function classifySourceInText(text) {
    // 簡易 URL 偵測
    const urlMatch = text.match(/https?:\/\/[^\s]+/);
    if (!urlMatch) return;
    callWails(() => ClassifySource(urlMatch[0], text, []))
      .then((evidence) => {
        const hint = sourceTrustToHint(evidence);
        if (hint) setSourceTrustHint(hint);
      })
      .catch(() => {});
  }

  // 啟動時載入信任清單 + 高影響領域 + 威脅紀錄（供後續分類使用）
  function refreshSourceTrustState() {
    callWails(GetProjectAllowlist).catch(() => {});
    callWails(GetHighImpactDomains).catch(() => {});
    callWails(ListThreatRecords).catch(() => {});
  }

  // 將來源加入信任清單（由 UI 提示操作觸發）
  async function addToAllowlist(url, hostname) {
    try {
      await callWails(() => AddSourceToAllowlist(JSON.stringify({ url, hostname })));
      setSourceTrustHint({ level: 'trusted', text: t('sourceTrust.trusted') });
      refreshSourceTrustState();
    } catch { /* best-effort */ }
  }

  // 從信任清單移除來源
  async function removeFromAllowlist(entryID) {
    try {
      await callWails(() => RemoveAllowlistEntry(entryID));
      setSourceTrustHint(null);
      refreshSourceTrustState();
    } catch { /* best-effort */ }
  }

  // 續期已過期的信任清單項目
  async function renewAllowlistItem(entryID) {
    try {
      await callWails(() => RenewAllowlistEntry(entryID));
      refreshSourceTrustState();
    } catch { /* best-effort */ }
  }

  // ── v3.6 Project Lifecycle 專案管理 ──
  // 開啟專案管理彈窗
  function openProjectManage() {
    setProjectManageOpen(true);
    setProjectManageView('menu');
    setPurgeConfirmStep(null);
  }

  // TASKS_1_7：清除專案資料改走 Consequence Menu（透過後端產生 ReviewCard）
  // 舊的「再點一次」雙擊確認已移除，統一進 review card 流程
  async function handlePurgeProject() {
    try {
      const card = await callWails(() => CreateDestructiveReviewCard(
        'purge_project', 'default', t('project.purgeAllDesc'), 'runtime/temp_sessions · runtime/action_results · runtime/crash_recovery'
      ));
      setDismissedDestructiveCards((prev) => prev.filter((id) => id !== reviewCardID(card)));
      await refreshReviewCards();
      setProjectManageOpen(false);
    } catch (error) {
      setDestructiveReviewResult({message: error?.message || t('project.purgeCreateFail')});
    }
  }

  // 清除隔離區暫存採較簡化確認，不進完整 Consequence Menu。
  async function handlePurgeBoundary() {
    if (purgeConfirmStep !== 'boundary') {
      setPurgeConfirmStep('boundary');
      return;
    }
    try {
      await callWails(() => PurgeBoundaryDir('controlled_trust/draft_sandbox_runs'));
      setPurgeConfirmStep(null);
      setProjectManageOpen(false);
    } catch { /* best-effort */ }
  }

  async function executeDestructiveReview(card, mode) {
    const cardID = reviewCardID(card);
    setDestructiveReviewResult(null);
    try {
      const result = await callWails(() => ResolveAndExecuteDestructiveReviewCard(cardID, mode));
      setDestructiveReviewResult(result);
      if (mode === 'cancel_keep_pending') {
        setDismissedDestructiveCards((prev) => [...new Set([...prev, cardID])]);
      }
      await refreshReviewCards();
      if (result && !result.card_pending) {
        setDismissedDestructiveCards((prev) => prev.filter((id) => id !== cardID));
      }
    } catch (error) {
      setDestructiveReviewResult({review_id: cardID, message: error?.message || t('project.purgeExecuteFail')});
    }
  }

  async function recreateDestructiveReview(card) {
    const cardID = reviewCardID(card);
    setDestructiveReviewResult(null);
    setDismissedDestructiveCards((prev) => [...new Set([...prev, cardID])]);
    await handlePurgeProject();
  }

  // 查看清除紀錄
  async function loadPurgeManifests() {
    try {
      const manifests = await callWails(ListPurgeManifests);
      setPurgeManifests(manifests || []);
      setProjectManageView('manifests');
    } catch {
      setPurgeManifests([]);
      setProjectManageView('manifests');
    }
  }

  async function approveDagNode(reviewId) {
    try {
      const run = await callWails(() => ApproveTaskStep(reviewId));
      if (!run) return false;
      syncTaskProgressRun(run);
      setReviewState((prev) => ({...prev, highRisk: {...prev.highRisk, status: 'approved'}}));
      setPendingTaskReview(null);
      await refreshReviewCards();
      return true;
    } catch {
      return false;
    }
  }

  function resolveActiveAdapter() {
    const adapters = adapterList || [];
    if (!adapters.length) return null;
    if (!activeAdapterId) return adapters[0];
    return adapters.find((adapter) => adapterKey(adapter) === activeAdapterId || adapterField(adapter, 'name', 'Name', '') === activeAdapterId) || adapters[0];
  }

  function resolveAdapterFromRefs() {
    const adapters = adapterListRef.current || [];
    const selectedID = activeAdapterIdRef.current;
    if (!adapters.length) return null;
    if (!selectedID) return adapters[0];
    return adapters.find((adapter) => adapterKey(adapter) === selectedID || adapterField(adapter, 'name', 'Name', '') === selectedID) || adapters[0];
  }

  function makeCLIInspectorPayload(adapter, sessionId, userText) {
    const normalized = normalizeAdapterDTO(adapter);
    return {
      adapter_id: adapterKey(normalized) || activeAdapterId || '',
      adapter_name: normalized?.name || activeAdapterId || '',
      cli_path: normalized?.path || 'resolved_by_backend',
      session_id: sessionId || '',
      user_text: userText,
      skill_injection: skillInjections?.length > 0 ? 'resolved_by_backend' : null,
    };
  }

  function normalizeCLIResponse(resp) {
    if (!resp) return null;
    return {
      ...resp,
      text: resp.text ?? resp.Text ?? '',
      error: resp.error ?? resp.Error ?? '',
      auth_required: resp.auth_required ?? resp.AuthRequired ?? false,
      auth_url: resp.auth_url ?? resp.AuthURL ?? '',
      adapter_id: resp.adapter_id ?? resp.AdapterID ?? '',
      action: resp.action ?? resp.Action ?? '',
      target: resp.target ?? resp.Target ?? '',
      next: resp.next ?? resp.Next ?? '',
    };
  }

  async function applyComposerBuiltInSideEffects(cliResp) {
    if (!cliResp) return cliResp;
    const action = String(cliResp.action || '').trim();
    const target = String(cliResp.target || '');
    if (action === t('system.copyAction')) {
      // 內建複製由 UI runtime 寫入剪貼簿，LLM 只提供內容。
      try {
        await ClipboardSetText(target);
        return {...cliResp, text: t('system.copied')};
      } catch {
        return {...cliResp, text: t('system.copyFail')};
      }
    }
    if (action === t('system.pasteAction')) {
      let clipText = '';
      try {
        clipText = await ClipboardGetText();
      } catch {
        return {...cliResp, text: t('system.pasteFail')};
      }
      if (!target || /composer/i.test(target) || target === t('system.composerTarget') || target === t('system.currentTarget') || target === t('system.chatTarget')) {
        setDraft(clipText || '');
        return {...cliResp, text: t('system.pasted')};
      }
      return {...cliResp, text: t('system.pasteConfirm', { target })};
    }
    return cliResp;
  }

  function isAPIAdapter(adapter) {
    const id = String(adapter?.id || adapter?.adapter_id || '').toLowerCase();
    const kind = String(adapter?.kind || adapter?.Kind || '').toLowerCase();
    return kind === 'api' || kind === 'local' || id.startsWith('llm-api-') || id.startsWith('local-');
  }

  // -----------------------------------------------------------------------
  // Composer fast path vs. local side effects
  // -----------------------------------------------------------------------
  // Keep the chat composer responsive by separating the work into two lanes.
  //
  // Fast path:
  //   1. render the user's bubble immediately;
  //   2. escape external tokens;
  //   3. call SendCLIMessage as soon as the CLI payload is ready.
  //
  // Side-effect lane:
  //   - BuildLLMContext;
  //   - AppendTalkEntry for durable local memory.
  //
  // Talk persistence is intentionally outside this background lane: each
  // main/sub has its own talk_full.md, and writes must complete before close.
  function buildComposerContextArgs(entry) {
    const blocks = [{
      source: 'composer',
      role: 'user',
      content: entry.escapedText || entry.userText,
    }];
    const sources = [];
    const isHighImpact = dagHighRiskPattern.test(entry.userText);
    return [JSON.stringify(blocks), JSON.stringify(sources), isHighImpact];
  }

  function runComposerSideEffects(entry, phase) {
    const tasks = [];
    if (phase === 'user') {
      const [blocksJSON, sourcesJSON, isHighImpact] = buildComposerContextArgs(entry);
      tasks.push(
        callWails(() => BuildLLMContext(blocksJSON, sourcesJSON, isHighImpact)),
      );
    }
    Promise.allSettled(tasks).then((results) => {
      const rejected = results.filter((result) => result.status === 'rejected');
      if (rejected.length > 0) {
        postDebugTrace('ui.composer.side_effects.error', entry.traceId, {
          phase,
          rejected: rejected.map((result) => result.reason?.message || String(result.reason)),
        });
      }
    });
  }

  function persistConversationEntry(conversationId, role, text, traceId) {
    // conversationId is the exact owner of this turn: "main" or a sub id.
    return callWails(() => AppendTalkEntryForAgent(conversationId || 'main', role, text))
      .catch((err) => {
        postDebugTrace('ui.conversation.persist.error', traceId, {
          conversation_id: conversationId || 'main',
          role,
          error: err?.message || String(err),
        });
        throw err;
      });
  }

  async function sendCLIInspectorText(text) {
    const trimmed = text.trim();
    if (!trimmed) return null;
    unlockManualGreeting();
    // DEBUG_TRACE_REMOVE: Correlates every debug trace event for this inspector send.
    const traceId = makeDebugTraceID('inspect');
    postDebugTrace('ui.inspector.submit.raw', traceId, {user_text: trimmed});
    const sessionId = appSessionId || '';
    const adapter = resolveActiveAdapter();
    const escaped = await callWails(() => EscapeExternalTokens(trimmed)).catch(() => trimmed);
    const payload = makeCLIInspectorPayload(adapter, sessionId, escaped || trimmed);
    payload.trace_id = traceId;
    postDebugTrace('ui.inspector.escaped', traceId, {escaped_text: escaped || trimmed, payload});
    setCliInspectorBusy(true);
    setCliInspectorLog({
      status: 'sending',
      payload,
      response: null,
      error: null,
      sent_at: new Date().toISOString(),
    });
    try {
      postDebugTrace('ui.inspector.before.SendTopInteractionMessage', traceId, payload);
      const resp = await callWails(() => SendTopInteractionMessage(payload.adapter_id, sessionId, escaped || trimmed, traceId));
      const cliResp = normalizeCLIResponse(resp);
      console.log('[CLI_MONITOR] frontend raw resp -> normalized', {traceId, resp, cliResp});
      postDebugTrace('ui.inspector.after.SendTopInteractionMessage', traceId, {response: cliResp || null});
      console.log('[CLI_MONITOR] frontend cliInspectorLog.response write', {traceId, response: cliResp || null});
      setCliInspectorLog((prev) => ({
        ...(prev || {payload}),
        status: cliResp?.error ? 'error' : 'done',
        response: cliResp || null,
        error: cliResp?.error || null,
        finished_at: new Date().toISOString(),
      }));
      if (cliResp?.text) {
        // 上方互動回覆只更新 greeting/statusRail，不碰下方主聊天。
        setState((prev) => ({
          ...prev,
          greeting: cliResp.text,
          statusRail: {...prev.statusRail, text: cliResp.text, layer: 'CLI'},
        }));
      }
      return cliResp;
    } catch (err) {
      const errorMsg = err?.message || String(err);
      setCliInspectorLog((prev) => ({
        ...(prev || {payload}),
        status: 'error',
        response: null,
        error: errorMsg,
        finished_at: new Date().toISOString(),
      }));
      return null;
    } finally {
      setCliInspectorBusy(false);
    }
  }

  // #61 CLIAdapter 整合：sendMessage 透過 Go 端 SendCLIMessage 發送，
  // 自動攜帶目前選中的 adapter 與當前 session 的 SkillInjection。
  // SkillInjection.BlockedUse 由 adapter 實作端驗證，前端不負責過濾。
  function sendRemoteInboundText(payload) {
    const text = String(payload?.text || '').trim();
    if (!text) return;
    unlockManualGreeting();
    const conversationId = activeConversationIdRef.current || 'main';
    const routedText = payload?.channel === 'discord'
      ? t('system.discordExternal', { text })
      : text;
    const traceId = makeDebugTraceID('remote');
    const sessionId = appSessionIdRef.current || '';
    const sourceLabel = payload.channel || 'remote';
    setConversationMessages(conversationId, (prev) => [...prev, `${sourceLabel}:${text}`]);
    callWails(() => EscapeExternalTokens(routedText))
      .catch(() => routedText)
      .then((escaped) => {
        const adapter = resolveAdapterFromRefs();
        const apiAdapter = isAPIAdapter(adapter);
        const adapterID = adapter?.id || activeAdapterIdRef.current || '';
        postDebugTrace(apiAdapter ? 'ui.remote.before.SendAPIMessage' : 'ui.remote.before.SendCLIMessage', traceId, {
          adapter_id: adapterID,
          adapter_name: adapter?.name || '',
          cli_path: adapter?.path || '',
          channel_id: payload.channel_id,
          user_text: escaped || routedText,
        });
        return callWails(() => (
          apiAdapter
            ? SendAPIMessage(adapterID, sessionId, escaped || routedText, traceId)
            : SendCLIMessage(adapterID, sessionId, escaped || routedText, traceId)
        ));
      })
      .then((resp) => {
        const cliResp = normalizeCLIResponse(resp);
        if (cliResp?.text) {
          setConversationMessages(conversationId, (prev) => [...prev, `Ai:${cliResp.text}`]);
          if (payload.channel_id) {
            callWails(() => DispatchRemoteBridgeAsync(payload.channel_id, cliResp.text)).catch(() => {});
          }
        } else if (cliResp?.error) {
          setConversationMessages(conversationId, (prev) => [...prev, t('system.remoteMessageFail', { error: cliResp.error })]);
        }
      })
      .catch((err) => {
        setConversationMessages(conversationId, (prev) => [...prev, t('system.remoteMessageFail', { error: err?.message || err })]);
      });
  }

  async function sendMessage(event) {
    event.preventDefault();
    submitComposerText(draft);
  }

  async function submitComposerText(rawText) {
    const text = stripInternalControlPrefix(rawText);
    if (!text) return;
    unlockManualGreeting();
    const conversationId = activeConversationIdRef.current || 'main';
    // DEBUG_TRACE_REMOVE: Correlates every debug trace event for this composer send.
    const traceId = makeDebugTraceID('chat');
    const pendingMessage = makeComposerPendingMessage(traceId);
    postDebugTrace('ui.composer.submit.raw', traceId, {user_text: text});
    setConversationMessages(conversationId, (prev) => [...prev, text, pendingMessage]);
    // 等待泡泡用瀏覽器 timer 控制，不交給 LLM；回覆/錯誤時會清掉。
    const pendingMediumTimer = window.setTimeout(() => {
      // 超過 30 秒仍未回覆時，維持同一顆等待泡泡，提示資料還在路上。
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(
        prev,
        traceId,
        makeComposerPendingMessage(traceId, composerPendingSlowText),
      ));
    }, 30000);
    const pendingVerySlowTimer = window.setTimeout(() => {
      // 超過 1 分鐘仍未回覆時，再更新同一顆等待泡泡，避免訊息列表洗版。
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(
        prev,
        traceId,
        makeComposerPendingMessage(traceId, composerPendingVerySlowText),
      ));
    }, 60000);
    const clearPendingTimers = () => {
      window.clearTimeout(pendingMediumTimer);
      window.clearTimeout(pendingVerySlowTimer);
    };
    setDraft('');
    try {
      await persistConversationEntry(conversationId, 'user', text, traceId);
    } catch (err) {
      setConversationMessages(conversationId, (prev) => [...prev, t('system.conversationSaveFail', { error: err?.message || err })]);
    }
    if (shouldHandleLearningShortcutBeforeLLM(text) && isLearningReplayRequest(text)) {
      clearPendingTimers();
      postDebugTrace('ui.composer.learning_replay_plan', traceId, {user_text: text});
      callWails(GetLastLearningReplayPlan)
        .then(async (plan) => {
          const message = formatLearningReplayPlan(plan);
          setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
          persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, ''), traceId).catch(() => {});
          await executeLearningReplayWithChat(plan, conversationId, traceId);
        })
        .catch((err) => {
          setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(
            prev,
            traceId,
            `Ai:我還讀不到上一段示範：${err?.message || String(err)}`
          ));
      });
      return;
    }
    const operationCatalogRequest = shouldHandleLearningShortcutBeforeLLM(text)
      ? parseLearningOperationCatalogRequest(text)
      : null;
    if (operationCatalogRequest) {
      clearPendingTimers();
      postDebugTrace('ui.composer.learning_operation_catalog', traceId, operationCatalogRequest);
      try {
        const items = operationCatalogRequest.mode === 'list'
          ? await callWails(() => ListLearningReplayCatalog(10))
          : await callWails(() => SearchLearningOperations(operationCatalogRequest.query, 8));
        const matches = Array.isArray(items) ? items : [];
        const message = operationCatalogRequest.mode === 'list'
          ? formatLearningOperationCatalog(matches)
          : formatLearningOperationSearchResults(operationCatalogRequest.query, matches);
        setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
        persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, ''), traceId).catch(() => {});
      } catch (err) {
        const message = `Ai:我讀不到已保存操作 catalog：${err?.message || String(err)}`;
        setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
      }
      return;
    }
    if (shouldCreateDagRun(text)) {
      clearPendingTimers();
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(
        prev,
        traceId,
        makeComposerPendingMessage(traceId, '任務規劃中，請稍等。'),
      ));
      postDebugTrace('ui.composer.task_progress_only', traceId, {user_text: text});
      classifySourceInText(text);
      startDagForMessage(text, {conversationId, traceId});
      return;
    }
    // v3.6: Source Trust — 若訊息含 URL，自動分類來源並顯示提示
    classifySourceInText(text);
    // v3.6: 跳脫外部 token 標記（防止使用者輸入干擾 LLM context 邊界）
    // 結果用於送出，不影響 UI 顯示的原始訊息
    const safeText = callWails(() => EscapeExternalTokens(text))
      .catch(() => text);
    // 透過 CLIAdapter 送出（含 SkillInjection），同時記錄互動
    const sessionId = appSessionId || '';
    Promise.resolve(safeText).then((escaped) => {
      const adapter = resolveActiveAdapter();
      const payload = makeCLIInspectorPayload(adapter, sessionId, escaped || text);
      payload.trace_id = traceId;
      const apiAdapter = isAPIAdapter(adapter);
      const sideEffectEntry = {
        traceId,
        sessionId,
        adapterId: payload.adapter_id,
        userText: text,
        escapedText: escaped || text,
        conversationId,
      };
      postDebugTrace('ui.composer.escaped', traceId, {escaped_text: escaped || text, payload});
      setChatCliLog({
        status: 'sending',
        payload,
        response: null,
        error: null,
        sent_at: new Date().toISOString(),
      });
      runComposerSideEffects(sideEffectEntry, 'user');
      postDebugTrace('ui.composer.before.ExecuteSkillMessage', traceId, payload);
      callWails(() => ExecuteSkillMessage(payload.adapter_id, sessionId, escaped || text, traceId))
        .then(async (decision) => {
          if (decision?.response) {
            await finishComposerExecution({resp: decision.response, payload, apiAdapter, traceId, conversationId, clearPendingTimers});
            return;
          }
          if (decision?.decision === 'need_confirm') {
            setSkillExecutionConfirm({
              resolveId: decision.resolve_id || decision.ResolveID,
              skillId: decision.skill_id || decision.SkillID,
              actionTarget: decision.action_target || decision.ActionTarget,
              message: decision.message || decision.Message,
              sessionId,
              adapterId: payload.adapter_id,
              userText: escaped || text,
              traceId,
              payload,
              apiAdapter,
              adapter,
              conversationId,
              clearPendingTimers,
            });
            setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(
              prev,
              traceId,
              makeComposerPendingMessage(traceId, decision.message || '這個 skill 需要確認後才會執行。'),
            ));
            return;
          }
          clearPendingTimers();
          setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(
            prev,
            traceId,
            decision?.message || 'Skill 需要人工確認，已暫停本次執行。',
          ));
        })
        .catch((err) => {
          failComposerExecution({err, payload, apiAdapter, adapter, traceId, conversationId, clearPendingTimers});
        });
    });
    // v3.6.4: 使用者送出新訊息 → 清除 Floating Candidate Actions
    callWails(DismissFloatingCandidates).catch(() => {});
    setReadinessGate((prev) => ({...prev, floating_candidates: []}));
    // 重置高風險確認流程狀態
    setRiskImpactExpanded(false);
    setGachaPhase(null);
    setLongPressProgress(0);
  }

  // ── v3.6.4 Readiness Gate UI Interaction Layer helper 函式 ──
  //
  // refreshReadinessGateState  — 從後端拉取最新 Gate 狀態快照
  // handleSelectCandidate      — 處理使用者點擊 Floating Candidate Action
  // handleLongPressStart       — 長按確認按鈕：開始按住
  // handleLongPressEnd         — 長按確認按鈕：放開或取消
  // triggerGachaAnimation      — 長按完成後觸發轉蛋動畫連鎖
  // handleNormalConfirm        — 普通風險：對話式 [是] 確認
  // handleNormalReject         — 普通風險：對話式 [否] 拒絕
  // handleHighRiskYes          — 高風險：點 [是] 展開影響說明面板

  async function refreshReadinessGateState() {
    callWails(GetReadinessGateState)
      .then((state) => { if (state) setReadinessGate(state); })
      .catch(() => {});
  }

  async function handleSelectCandidate(candidateID) {
    // §12A.1: 點擊 Candidate → 低風險候選直接送出；input: 候選只預填讓使用者補字。
    const candidate = readinessGate.floating_candidates?.find((c) => c.id === candidateID);
    if (candidate?.draft) {
      const draft = stripInternalControlDraft(candidate.draft);
      if (draft.startsWith('input:')) {
        setDraft(draft.slice('input:'.length));
      } else if (String(candidateID || '').startsWith('intent-option-')) {
        const conversationId = activeConversationIdRef.current || 'main';
        const traceId = makeDebugTraceID('choice');
        // 純選項卡只是替使用者做選擇，不再把選項送回模型重判。
        setConversationMessages(conversationId, (prev) => [...prev, draft]);
        persistConversationEntry(conversationId, 'user', draft, traceId).catch(() => {});
      } else {
        submitComposerText(draft);
      }
    }
    callWails(() => SelectFloatingCandidate(candidateID))
      .then((nextState) => { if (nextState) setReadinessGate(nextState); })
      .catch(() => {});
  }

  function handleLongPressStart() {
    // §11.2: 長按開始 — 啟動進度計時，觸覺回饋從 subtle 漸進增強
    setLongPressActive(true);
    setLongPressProgress(0);
    longPressStartRef.current = Date.now();
    const tick = () => {
      const elapsed = Date.now() - longPressStartRef.current;
      const progress = Math.min(100, (elapsed / LONG_PRESS_DURATION_MS) * 100);
      setLongPressProgress(progress);
      if (progress >= 100) {
        // 長按完成 → 觸發轉蛋動畫
        setLongPressActive(false);
        triggerGachaAnimation();
      } else {
        longPressTimerRef.current = requestAnimationFrame(tick);
      }
    };
    longPressTimerRef.current = requestAnimationFrame(tick);
  }

  function handleLongPressEnd() {
    // §11.2: 鬆手取消 — 清除進度，觸覺回饋 light rebound
    if (longPressTimerRef.current) {
      cancelAnimationFrame(longPressTimerRef.current);
      longPressTimerRef.current = null;
    }
    if (longPressProgress < 100) {
      setLongPressActive(false);
      setLongPressProgress(0);
    }
  }

  function triggerGachaAnimation() {
    // §11.3 轉蛋動畫連鎖：collapse → pulse → particles → reveal（「執行中」）
    setGachaPhase('collapse');
    setTimeout(() => setGachaPhase('pulse'), GACHA_COLLAPSE_MS);
    setTimeout(() => setGachaPhase('particles'), GACHA_COLLAPSE_MS + GACHA_PULSE_MS);
    setTimeout(() => {
      setGachaPhase('reveal');
      // 動畫結束後，通知後端執行確認動作
      if (reviewState.highRisk?.status === 'pending' && reviewState.highRisk?.id) {
        confirmSkillBuild(reviewState.highRisk.id);
      }
    }, GACHA_COLLAPSE_MS + GACHA_PULSE_MS + GACHA_PARTICLE_MS);
  }

  function handleNormalConfirm() {
    // §11.2 第一層：普通風險對話式確認 → 直接執行
    if (reviewState.highRisk?.status === 'pending' && reviewState.highRisk?.id) {
      confirmSkillBuild(reviewState.highRisk.id);
    }
    setReadinessGate((prev) => ({...prev, risk_tier: 'none'}));
  }

  function handleNormalReject() {
    // §11.5: 選「否」→ action 取消，DAG node 進入 blocked/cancelled
    setReadinessGate((prev) => ({...prev, risk_tier: 'none'}));
    setRiskImpactExpanded(false);
  }

  function handleHighRiskYes() {
    // §11.2 第三層：點 [是] → 展開影響說明面板，按鈕變紅
    setRiskImpactExpanded(true);
  }

  /* i18n: map panel language display label → locale code */
  function panelLangToLocale(displayLabel) {
    const map = {
      [_t('settings.langZhTW')]: 'zh-TW',
      [_t('settings.langEn')]: 'en',
      [_t('settings.langJa')]: 'ja',
      [_t('settings.langPt')]: 'pt-PT',
      [_t('settings.langEs')]: 'es',
      [_t('settings.langTh')]: 'th',
      // fallback hardcoded labels
      '繁中': 'zh-TW', '英文': 'en', '日文': 'ja',
      'Traditional Chinese': 'zh-TW', 'English': 'en', 'Japanese': 'ja',
      '中': 'zh-TW', 'en': 'en', 'ja': 'ja',
      'pt': 'pt-PT', 'pt-PT': 'pt-PT', 'es': 'es', 'th': 'th',
      'Português': 'pt-PT', 'Español': 'es', 'ไทย': 'th',
    };
    return map[displayLabel] || null;
  }

  async function savePanelPatch(patch) {
    try {
      const panel = normalizePanelSettings({...settingsState.panel, ...patch});
      const next = await callWails(() => SavePanelSettings(panel));
      setSettingsState((prev) => normalizeSettingsState(next, {...prev, panel}));
      callWails(GetVoiceSettings).then((settings) => setVoiceState(settings || null)).catch(() => {});

      /* i18n: if panel language changed, sync i18n store and reload */
      if (patch.panelLanguage) {
        const locale = panelLangToLocale(patch.panelLanguage);
        if (locale && locale !== useI18n.getState().language) {
          useI18n.getState().setLanguage(locale); // saves to localStorage and updates in place
        }
      }
    } catch {
      setSettingsState((prev) => ({...prev, panel: normalizePanelSettings({...prev.panel, ...patch})}));
    }
  }

  // #I-805: 重啟 Sidecar 並清除崩潰狀態。
  // sidecarState 統一使用字串格式（'running' | 'crashed' | 'restarting' 等）。
  async function handleRestartSidecar() {
    setSidecarState('restarting');
    try {
      await callWails(RestartSidecar);
      setSidecarState('running');
      setMessages((prev) => [...prev, t('system.sidecarRestart')]);
    } catch (err) {
      setSidecarState('crashed');
      setMessages((prev) => [...prev, `[${t('system.sysLabel')}] ${t('system.sidecarRestartFail', { error: String(err) })}`]);
    }
  }

  // v3.6: 停止 Sidecar
  async function handleStopSidecar() {
    try {
      await callWails(StopSidecar);
      setSidecarState('stopped');
    } catch { /* best-effort */ }
  }

  async function handleUniversalStop() {
    if (voiceRecording) {
      await cancelVoiceRecording();
      setVoiceStatus(t('voice.stopRecording'));
      return;
    }
    if (voiceBusy) {
      setVoiceStatus(t('voice.transcriptInProgress'));
      return;
    }
    if (sidecarState === 'running') {
      await handleStopSidecar();
      setVoiceStatus(t('voice.stopCurrentJob'));
      return;
    }
    setVoiceStatus(t('voice.noActiveJob'));
  }

  // v3.6: 拉取 Stop Recovery Cards（後端 stop_recovery service）
  async function refreshStopRecoveryCards() {
    callWails(ListOpenStopRecoveryCards)
      .then((cards) => setStopRecoveryCards(cards || []))
      .catch(() => {});
    callWails(HasOpenStopRecoveryCard)
      .then((b) => setHasOpenStopRecovery(!!b))
      .catch(() => {});
  }

  // v3.6: 解決 Stop Recovery Card（選擇恢復動作）
  async function resolveStopRecoveryCard(cardID, action) {
    try {
      await callWails(() => ResolveStopRecoveryCard(cardID, action));
      refreshStopRecoveryCards();
    } catch { /* best-effort */ }
  }

  // ── v3.6.2 W3A Media Provenance（§9A）──
  //
  // 函式群組說明：
  //  refreshW3ATrustList  — 從後端拉取開發者信任清單
  //  importMediaW3A       — 匯入媒體 → 驗證 → 顯示選單解釋 W3A 功能
  //  dismissW3AImportPopup — 關閉匯入選單

  async function refreshW3ATrustList() {
    callWails(ListW3ATrustedDevelopers)
      .then((list) => setW3aTrustList(list || []))
      .catch(() => {});
  }

  function isW3AMediaPath(filePath) {
    return /\.(png|jpe?g|webp|gif|bmp|tiff?|wav|mp3|m4a|aac|flac|ogg|mp4|mov|m4v|webm)$/i.test(String(filePath || ''));
  }

  function shouldProbeDroppedInstallPackage(paths = []) {
    if (paths.length !== 1 || isW3AMediaPath(paths[0])) return false;
    const name = String(paths[0] || '').split(/[\\/]/).pop() || '';
    return !name.includes('.') || /\.(zip|skill|subagent)$/i.test(name);
  }

  function resolveW3AMediaPath() {
    return w3aImportPopup?.source_path
      || w3aImportPopup?.info?.file_path
      || w3aDetail?.file_path
      || '';
  }

  function makeW3ACopyPath(filePath) {
    const rawPath = String(filePath || '');
    const splitAt = Math.max(rawPath.lastIndexOf('/'), rawPath.lastIndexOf('\\'));
    const dir = splitAt >= 0 ? rawPath.slice(0, splitAt + 1) : '';
    const name = splitAt >= 0 ? rawPath.slice(splitAt + 1) : rawPath;
    const extAt = name.lastIndexOf('.');
    return extAt > 0
      ? `${dir}${name.slice(0, extAt)}.w3a-copy${name.slice(extAt)}`
      : `${dir}${name}.w3a-copy`;
  }

  async function importMediaW3A(filePath) {
    try {
      setW3aActionError('');
      const result = await callWails(() => ImportMediaVerify(filePath));
      if (result) {
        const popup = {...result, source_path: filePath};
        setW3aImportPopup(popup);
        setW3aDetail(popup.info || null);
        setW3aPollutionResult(popup.info?.pollution || null);
      }
      return result;
    } catch (error) {
      const message = error?.message || String(error);
      setW3aActionError(message);
      throw error;
    }
  }

  async function loadW3AMediaInfo() {
    const filePath = resolveW3AMediaPath();
    if (!filePath) {
      setW3aActionError(t('w3a.noMediaPath'));
      return;
    }
    setW3aActionBusy('info');
    setW3aActionError('');
    try {
      const info = await callWails(() => GetMediaW3AInfo(filePath));
      setW3aDetail(info || null);
      setW3aImportPopup((current) => current ? {...current, info: info || current.info, source_path: filePath} : current);
      setW3aToastMsg(t('w3a.infoLoaded'));
    } catch (error) {
      setW3aActionError(error?.message || String(error));
    } finally {
      setW3aActionBusy('');
    }
  }

  async function detectW3APollution() {
    const filePath = resolveW3AMediaPath();
    if (!filePath) {
      setW3aActionError(t('w3a.noMediaPath'));
      return;
    }
    setW3aActionBusy('pollution');
    setW3aActionError('');
    try {
      const report = await callWails(() => DetectModelPollution(filePath));
      setW3aPollutionResult(report || null);
      setW3aDetail((current) => current ? {...current, pollution: report || null} : current);
      setW3aToastMsg(report?.is_pollution_risk ? t('w3a.pollutionRiskDetected') : t('w3a.pollutionSafe'));
    } catch (error) {
      setW3aActionError(error?.message || String(error));
    } finally {
      setW3aActionBusy('');
    }
  }

  async function showW3ATransferGuidance() {
    setW3aActionBusy('guidance');
    setW3aActionError('');
    try {
      const guidance = await callWails(GetW3ATransferGuidance);
      setW3aTransferGuidance(guidance || null);
      setW3aToastMsg(guidance?.ui_message || t('w3a.exportHint'));
    } catch (error) {
      setW3aActionError(error?.message || String(error));
    } finally {
      setW3aActionBusy('');
    }
  }

  async function trustW3ADeveloper() {
    const signature = w3aDetail?.developer_signature || w3aImportPopup?.info?.developer_signature;
    if (!signature?.app_id || !signature?.public_key) {
      setW3aActionError(t('w3a.noDeveloperSignature'));
      return;
    }
    setW3aActionBusy('trust');
    setW3aActionError('');
    try {
      await callWails(() => AddW3ATrustedDeveloper(signature.app_id, signature.public_key, signature.app_id));
      await refreshW3ATrustList();
      setW3aToastMsg(t('w3a.trustAdded'));
    } catch (error) {
      setW3aActionError(error?.message || String(error));
    } finally {
      setW3aActionBusy('');
    }
  }

  async function exportW3AWithSidecarCopy() {
    const filePath = resolveW3AMediaPath();
    if (!filePath) {
      setW3aActionError(t('w3a.noMediaPath'));
      return;
    }
    const destPath = makeW3ACopyPath(filePath);
    setW3aActionBusy('export');
    setW3aActionError('');
    try {
      await callWails(() => ExportMediaWithSidecar(filePath, destPath));
      setW3aToastMsg(t('w3a.exportCopied', { path: destPath }));
    } catch (error) {
      setW3aActionError(error?.message || String(error));
    } finally {
      setW3aActionBusy('');
    }
  }

  function dismissW3AImportPopup() {
    setW3aImportPopup(null);
    setW3aActionError('');
  }

  // W3A 驗證狀態對應的 icon 與顏色
  const w3aStatusConfig = {
    exact_original:        { icon: '✅', color: '#2ecc40', label: t('w3a.originalFile') },
    w3a_app_processed:     { icon: '🔏', color: '#3498db', label: t('w3a.appProcessed') },
    platform_processed_copy: { icon: '📋', color: '#f39c12', label: t('w3a.platformProcessed') },
    unauthorized_copy:     { icon: '🚫', color: '#e74c3c', label: t('w3a.unauthorizedCopy') },
    content_modified:      { icon: '✏️', color: '#e67e22', label: t('w3a.contentModified') },
    model_pollution_risk:  { icon: '☣️', color: '#e74c3c', label: t('w3a.pollutionRisk') },
    unverified:            { icon: '❓', color: '#95a5a6', label: t('w3a.unverified') },
  };

  // ── v3.6.3 Remote Bridge Communication（§12A）──
  //
  // 函式群組說明：
  //  refreshRemoteBridgeChannels  — 從後端拉取通道清單，更新 state → 重繪 icon
  //  detectAndRegisterRemoteBridge — 完整三步驟流程：偵測 → 連線測試 → 註冊
  //    ↳ 由 confirmReferenceLink() 在 linkPreview.type === 'remote_bridge' 時呼叫
  //  toggleRemoteBridgeChannel    — 點擊 icon：啟用（notification_only）或停用
  //  openRemoteBridgeModePopup    — 長按/右鍵：開啟模式切換彈窗
  //  switchRemoteBridgeMode       — 彈窗選擇模式後呼叫後端切換
  //  removeRemoteBridgeChannel    — 彈窗「移除通道」按鈕

  async function confirmCredentialMigration() {
    setCredentialMigrationBusy(true);
    try {
      const status = await callWails(ConfirmCredentialMigration);
      setCredentialMigrationStatus(status || null);
      if (status?.ready) {
        refreshRemoteBridgeChannels();
      }
    } catch (err) {
      setCredentialMigrationStatus((current) => ({...(current || {}), error: err?.message || String(err)}));
    } finally {
      setCredentialMigrationBusy(false);
    }
  }

  async function disableCredentialMigration() {
    setCredentialMigrationBusy(true);
    try {
      const status = await callWails(DisableCredentialMigration);
      setCredentialMigrationStatus(status || null);
    } catch (err) {
      setCredentialMigrationStatus((current) => ({...(current || {}), error: err?.message || String(err)}));
    } finally {
      setCredentialMigrationBusy(false);
    }
  }

  async function refreshRemoteBridgeChannels() {
    callWails(ListRemoteBridgeChannels)
      .then((channels) => setRemoteBridgeChannels(channels || []))
      .catch(() => {});
  }

  function remoteBridgeDefaultName(channel) {
    const channelKey = String(channel || '').toLowerCase();
    const labels = {telegram: 'Telegram', discord: 'Discord', line: 'LINE', teams: 'Teams', qq: 'QQ Bot', custom: t('remote.customWebhook')};
    return labels[channelKey] || t('remote.defaultChannelName');
  }

  function openRemoteBridgeRename(binding) {
    if (!binding?.id) return;
    const currentName = binding.display_name || binding.displayName || remoteBridgeDefaultName(binding.channel);
    setRemoteBridgeRenameTarget({...binding, display_name: currentName});
    setRemoteBridgeRenameDraft(currentName);
  }

  async function saveRemoteBridgeName() {
    if (!remoteBridgeRenameTarget?.id) return;
    const nextName = remoteBridgeRenameDraft.trim() || remoteBridgeDefaultName(remoteBridgeRenameTarget.channel);
    try {
      await callWails(() => RenameRemoteBridgeChannel(remoteBridgeRenameTarget.id, nextName));
      setRemoteBridgeRenameTarget(null);
      setRemoteBridgeRenameDraft('');
      await refreshRemoteBridgeChannels();
    } catch (err) {
      setToolResult({toolId: 'remote-bridge-rename', ok: false, message: String(err)});
    }
  }

  // 從引用連結輸入偵測 URL 並註冊通道
  async function detectAndRegisterRemoteBridge(rawURL) {
    setRemoteBridgeDetecting(true);
    try {
      const detected = await callWails(() => DetectRemoteBridgeChannel(rawURL));
      if (!detected?.matched) {
        setRemoteBridgeDetecting(false);
        return {success: false, error: t('remote.detectFail')};
      }

      // 連線測試
      const testResult = await callWails(() => TestRemoteBridgeConnection(rawURL));
      if (!testResult?.success) {
        setRemoteBridgeDetecting(false);
        return {success: false, error: testResult?.error_message || t('remote.testFail')};
      }

      // 註冊通道
      const binding = await callWails(() => RegisterRemoteBridgeChannel(rawURL));
      await refreshRemoteBridgeChannels();
      setRemoteBridgeDetecting(false);
      return {success: true, binding, detected};
    } catch (err) {
      setRemoteBridgeDetecting(false);
      return {success: false, error: String(err)};
    }
  }

  // §12A.2 模式選擇 — 開啟 / 重置設定流程
  function openRemoteBridgeSetup() {
    setRemoteBridgeSetupOpen(true);
    setRemoteBridgeSetupStep('mode_select');
    setRemoteBridgeSetupGuideStep(0);
    setRemoteBridgeSetupMode(null);
    setRemoteBridgeSetupPlatform(null);
    setRemoteBridgeSetupFields({});
  }
  function openRemoteBridgeSetupForPlatform(platformID) {
    setRemoteBridgeSetupOpen(true);
    setRemoteBridgeSetupMode('quick');
    setRemoteBridgeSetupStep('quick_fields');
    setRemoteBridgeSetupGuideStep(0);
    setRemoteBridgeSetupPlatform(platformID);
    setRemoteBridgeSetupFields({});
  }
  function closeRemoteBridgeSetup() {
    setRemoteBridgeSetupOpen(false);
    setRemoteBridgeSetupStep('mode_select');
    setRemoteBridgeSetupGuideStep(0);
    setRemoteBridgeSetupMode(null);
    setRemoteBridgeSetupPlatform(null);
    setRemoteBridgeSetupFields({});
  }

  // §12A.2 Quick Mode — 選平台後顯示必填欄位
  function selectQuickPlatform(platformID) {
    setRemoteBridgeSetupPlatform(platformID);
    setRemoteBridgeSetupStep('quick_fields');
    setRemoteBridgeSetupGuideStep(0);
    setRemoteBridgeSetupFields({});
  }

  function openRemoteBridgeFieldHelp(field) {
    const queryMap = {
      telegram: {
        bot_token: 'Telegram BotFather create bot token official guide',
      },
      discord: {
        bot_token: 'Discord Developer Portal create bot bot token official guide',
        guild_id: 'Discord enable developer mode copy server ID official guide',
        channel_id: 'Discord enable developer mode copy channel ID official guide',
      },
      teams: {
        webhook_url: 'Microsoft Teams incoming webhook workflow URL official guide',
      },
      line: {
        channel_access_token: 'LINE Messaging API channel access token official guide',
        channel_secret: 'LINE Developers Messaging API channel secret webhook signature official guide',
        recipient_id: 'LINE Messaging API userId groupId roomId recipient ID official guide',
      },
      qq: {
        bot_app_id: 'QQ Bot QQ Guild Bot AppID official guide',
        bot_token: 'QQ Bot QQ Guild Bot token official guide',
        channel_id: 'QQ Guild Bot channel ID official guide',
      },
    };
    const query = queryMap[remoteBridgeSetupPlatform]?.[field] || `${remoteBridgeSetupPlatform || 'remote bridge'} ${field} official guide`;
    openExternal(`https://www.google.com/search?q=${encodeURIComponent(query)}`);
  }

  function remoteBridgeField(label, field, options = {}) {
    const {type = 'text', placeholder = ''} = options;
    // i18n: remote
    return (
      <label className="rb-field">
        <span className="rb-field-title">
          <span>{label}</span>
          <button
            className="rb-help-btn"
            type="button"
            title={t('remote.searchHintTitle')}
            onClick={() => openRemoteBridgeFieldHelp(field)}
          >
            ?
          </button>
        </span>
        <input
          type={type}
          value={remoteBridgeSetupFields[field] || ''}
          onChange={(e) => setRemoteBridgeSetupFields(f => ({...f, [field]: e.target.value}))}
          placeholder={placeholder}
        />
      </label>
    );
  }

  /* i18n: remote */
  const lineSetupGuide = [
    {
      title: t('remote.lineGuide0Title'),
      body: t('remote.lineGuide0Body'),
      action: t('remote.lineGuide0Action'),
      url: 'https://developers.line.biz/console/',
    },
    {
      title: t('remote.lineGuide1Title'),
      body: t('remote.lineGuide1Body'),
      action: t('remote.lineGuide1Action'),
      query: 'LINE Developers create Messaging API channel access token official guide',
    },
    {
      title: t('remote.lineGuide2Title'),
      body: t('remote.lineGuide2Body'),
      action: t('remote.lineGuide2Action'),
      query: 'LINE Messaging API get userId recipient ID official guide',
    },
    {
      title: t('remote.lineGuide3Title'),
      body: t('remote.lineGuide3Body'),
      action: t('remote.lineGuide3Action'),
      query: 'LINE Developers channel secret webhook signature official guide',
    },
  ];

  const discordSetupGuide = [
    {
      title: t('remote.discordGuide0Title'),
      body: t('remote.discordGuide0Body'),
      action: t('remote.discordGuide0Action'),
      url: 'https://discord.com/developers/applications',
    },
    {
      title: t('remote.discordGuide1Title'),
      body: t('remote.discordGuide1Body'),
      action: t('remote.discordGuide1Action'),
      query: 'Discord developer mode copy server ID official guide',
    },
    {
      title: t('remote.discordGuide2Title'),
      body: t('remote.discordGuide2Body'),
      action: t('remote.discordGuide2Action'),
      query: 'Discord developer mode copy channel ID official guide',
    },
  ];

  // §12A.2 提交 Quick Mode 註冊
  async function submitQuickModeRegistration() {
    if (!remoteBridgeSetupPlatform) return;
    setRemoteBridgeDetecting(true);
    try {
      const fieldsJSON = JSON.stringify(remoteBridgeSetupFields);
      const binding = await callWails(() => RegisterRemoteBridgeChannelWithMode('quick', remoteBridgeSetupPlatform, fieldsJSON, ''));
      await refreshRemoteBridgeChannels();
      closeRemoteBridgeSetup();
      openRemoteBridgeRename(binding);
    } catch (err) {
      setToolResult({ toolId: 'remote-bridge-setup', ok: false, message: String(err) });
    }
    setRemoteBridgeDetecting(false);
  }

  // §12A.2 提交 Developer Mode 註冊
  async function submitDeveloperModeRegistration() {
    setRemoteBridgeDetecting(true);
    try {
      const customConfig = JSON.stringify({
        url: remoteBridgeSetupFields.url || '',
        method: remoteBridgeSetupFields.method || 'POST',
        headers: JSON.parse(remoteBridgeSetupFields.headers || '{}'),
        body: remoteBridgeSetupFields.body_template || '',
        timeout_seconds: 10,
      });
      const binding = await callWails(() => RegisterRemoteBridgeChannelWithMode('developer', '', '{}', customConfig));
      await refreshRemoteBridgeChannels();
      closeRemoteBridgeSetup();
      openRemoteBridgeRename(binding);
    } catch (err) {
      setToolResult({ toolId: 'remote-bridge-setup', ok: false, message: String(err) });
    }
    setRemoteBridgeDetecting(false);
  }

  // 點擊 icon — 切換啟用/停用（notification_only 模式）
  async function toggleRemoteBridgeChannel(channelID) {
    const channel = remoteBridgeChannels.find((ch) => ch.id === channelID);
    if (!channel) return;
    try {
      if (channel.active) {
        await callWails(() => DeactivateRemoteBridgeChannel(channelID));
      } else {
        await callWails(() => ActivateRemoteBridgeChannel(channelID));
      }
      await refreshRemoteBridgeChannels();
    } catch { /* best-effort */ }
  }

  async function makeRemoteBridgePrimary(channelID) {
    try {
      await callWails(() => SetRemoteBridgePrimaryChannel(channelID));
      await refreshRemoteBridgeChannels();
      setToolResult({toolId: 'reference-link', ok: true, message: t('remote.setPrimarySuccess')});
    } catch (err) {
      setToolResult({toolId: 'reference-link', ok: false, message: err?.message || t('remote.setPrimaryFail')});
    }
  }

  // 長按/右鍵 — 開啟模式切換彈窗
  function openRemoteBridgeModePopup(channelID) {
    setRemoteBridgeModePopup(channelID);
    setRemoteBridgeInboundInfo(null);
    callWails(() => GetRemoteBridgeInboundEndpoint(channelID))
      .then((info) => setRemoteBridgeInboundInfo(info || null))
      .catch(() => setRemoteBridgeInboundInfo(null));
  }

  // 切換通道權限模式
  async function switchRemoteBridgeMode(channelID, mode) {
    try {
      await callWails(() => SwitchRemoteBridgeMode(channelID, mode));
      setRemoteBridgeModePopup(null);
      await refreshRemoteBridgeChannels();
    } catch { /* best-effort */ }
  }

  // 移除通道
  async function removeRemoteBridgeChannel(channelID) {
    try {
      await callWails(() => RemoveRemoteBridgeChannel(channelID));
      setRemoteBridgeModePopup(null);
      await refreshRemoteBridgeChannels();
    } catch { /* best-effort */ }
  }

  function sendTemporaryRemoteBridgeTest(channel) {
    if (!channel?.id) return;
    const displayName = channel.display_name || channel.displayName || channel.channel || t('remote.defaultDisplay');
    const text = t('remote.testSendText', { displayName });
    Promise.resolve(callWails(() => DispatchRemoteBridgeAsync(channel.id, text)))
      .then((dispatchId) => {
        setToolResult({toolId: 'remote-bridge-test-send', ok: true, message: t('remote.testSendSuccess', { dispatchId: dispatchId || 'queued' })});
      })
      .catch((err) => {
        setToolResult({toolId: 'remote-bridge-test-send', ok: false, message: t('remote.testSendFail', { error: err?.message || err })});
      });
  }

  async function saveRemoteBridgeInboundSecret(channelID, channelSecret) {
    if (!channelID || !channelSecret?.trim()) return;
    try {
      await callWails(() => SaveRemoteBridgeInboundSecret(channelID, channelSecret));
      setToolResult({toolId: 'reference-link', ok: true, message: t('remote.saveSecretSuccess')});
    } catch (err) {
      setToolResult({toolId: 'reference-link', ok: false, message: err?.message || t('remote.saveSecretFail')});
    }
  }

  async function restoreUIDefaults() {
    try {
      const restoredUISettings = await callWails(RestoreUIDefaults);
      setSettingsState((prev) => ({
        ...prev,
        panel: panelFromUISettings(restoredUISettings, prev.panel),
      }));
    } catch {
      // Fallback: reset panel state to factory defaults in-memory
      setSettingsState((prev) => ({
        ...prev,
        panel: {
          /* i18n: settings defaults */ panelLanguage: _t('settings.langZhTW'),
          roleLanguage:  t('settings.roleLanguageAuto'),
          fontPreset:    t('settings.fontPresetDefault'),
          fontScale:     '100%',
          panelStyle:    defaultPanelStyle,
        },
      }));
    }
  }

  // v3.6: Avatar 後端接線 — 載入預設頭像 + 儲存/刪除靜態頭像
  async function loadAvatarPresets() {
    try {
      const presets = await callWails(ListAvatarPresets);
      setAvatarPresets(presets || []);
    } catch { /* best-effort */ }
  }

  async function loadCurrentAvatar(personaID = settingsState.activePersonaId) {
    try {
      const avatar = await callWails(() => GetCurrentAvatar(personaID));
      setCurrentAvatar(avatar || null);
      setAvatarConfigs((prev) => ({...prev, [personaID]: avatar || null}));
    } catch { /* best-effort */ }
  }

  async function saveStaticAvatar(personaID, file, crop = {}) {
    if (personaID === lockedPersonaId) {
      setAvatarModeNotice(t('avatar.locked'));
      return;
    }
    try {
      // Static avatar upload deliberately sends only the processed image bytes to
      // the Wails backend; provider selection and filesystem writes stay behind
      // the Avatar service boundary required by Spec §10.
      const imageData = Array.from(new Uint8Array(await file.arrayBuffer()));
      await callWails(() => SaveStaticAvatar(personaID, imageData, file.type || 'image/png', JSON.stringify(crop)));
      setStaticAvatarPreviews((prev) => ({...prev, [personaID]: bytesToDataUrl(imageData, file.type || 'image/png')}));
      await loadCurrentAvatar(personaID);
      setAvatarModeNotice(t('avatar.staticApplied'));
    } catch {
      setAvatarModeNotice(t('avatar.staticSaveFail'));
    }
  }

  async function deleteStaticAvatar(personaID) {
    if (personaID === lockedPersonaId) return;
    try {
      await callWails(() => DeleteStaticAvatar(personaID));
      loadCurrentAvatar(personaID);
    } catch { /* best-effort */ }
  }

  async function setAvatarProviderMode(provider, personaID) {
    const targetPersonaID = personaID || settingsState.personas[0]?.id || settingsState.activePersonaId;
    if (targetPersonaID === lockedPersonaId) {
      setAvatarModeNotice(t('avatar.locked'));
      setAvatarPickerOpen(false);
      return;
    }
    // Provider switching is UI-only: API generation shows a disabled state until
    // credentials exist, static image opens the local preview modal, and pixel
    // avatar is the safe built-in fallback.
    if (provider === 'user_image_api') {
      setAvatarModeNotice(t('avatar.apiKeyNotSet'));
      return;
    }
    if (provider === 'static_image') {
      setAvatarUploadTargetId(targetPersonaID);
      setAvatarModeNotice('');
      return;
    }
    try {
      await callWails(() => SetAvatarProvider(targetPersonaID, 'built_in_pixel'));
      await loadCurrentAvatar(targetPersonaID);
      setAvatarModeNotice(t('avatar.switchedToBuiltIn'));
    } catch {
      setAvatarConfigs((prev) => ({...prev, [targetPersonaID]: {avatar_provider: 'built_in_pixel', persona_id: targetPersonaID, pixel_pack: defaultPixelPackForPersona(targetPersonaID)}}));
      setAvatarModeNotice(t('avatar.switchedToBuiltInFrontend'));
    }
  }

  async function savePersonaPatch(personaId, patch) {
    const current = settingsState.personas.find((persona) => persona.id === personaId);
    if (!current && settingsState.personas.length >= maxPersonas) return;
    const persona = {...current, ...patch};
    if (!persona.id) persona.id = personaId;
    if (!persona.name) persona.name = t('persona.fallbackName', { index: settingsState.personas.length + 1 });
    if (persona.id === lockedPersonaId) persona.name = lockedPersonaName;
    try {
      const next = await callWails(() => SavePersona(persona));
      const normalized = normalizeSettingsState(next, settingsState);
      setSettingsState(normalized);
      if (persona.id === next.activePersonaId) {
        if (persona.name) setPersonaName(persona.id === lockedPersonaId ? lockedPersonaName : persona.name);
        if (persona.identity) setPersonaJob(persona.identity);
      }
    } catch {
      setSettingsState((prev) => {
        const normalizedPersonas = normalizeLockedPersonas(prev.personas.map((item) => (item.id === persona.id ? persona : item)));
        return {
          ...prev,
          activePersonaId: persona.id,
          personas: normalizedPersonas,
        };
      });
    }
  }

  function createPersonaFromPackage(packageData = {}) {
    const nextIndex = settingsState.personas.length + 1;
    const id = packageData.id || `persona-${Date.now()}`;
    return {
      id,
      name: packageData.name || t('persona.fallbackName', { index: nextIndex }),
      icon: packageData.icon || '＋',
      avatarUrl: packageData.avatarUrl || '',
      identity: packageData.identity || t('persona.defaultIdentityA'),
      replyStrategy: packageData.replyStrategy || '',
      roleStrength: packageData.roleStrength || '20%',
      personality: packageData.personality || '',
      scenario: packageData.scenario || '',
      description: packageData.description || '',
    };
  }

  async function addPersona(packageData) {
    if (settingsState.personas.length >= maxPersonas) return;
    const persona = createPersonaFromPackage(packageData);
    await savePersonaPatch(persona.id, persona);
  }

  async function reorderPersonas(orderIds) {
    try {
      const next = await callWails(() => ReorderPersonas(orderIds));
      const normalized = normalizeSettingsState(next, settingsState);
      setSettingsState(normalized);
      const activePersona = findActivePersona(normalized);
      if (activePersona?.name) setPersonaName(activePersona.name);
      if (activePersona?.identity) setPersonaJob(activePersona.identity);
    } catch {
      setSettingsState((prev) => ({
        ...prev,
        personas: orderIds
          .map((id) => prev.personas.find((persona) => persona.id === id))
          .filter(Boolean),
      }));
    }
  }

  async function startNativePersonaExport(personaId) {
    if (!personaId) return null;
    try {
      const result = await callWails(() => NativeDragExportPersonaHandler(personaId, 'export_copy'));
      if (result?.status !== 'success') {
        setToolResult({
          toolId: personaId,
          ok: false,
          message: result?.status === 'cancelled'
            ? t('persona.dragCancelled')
            : t('persona.dragExportFail', { message: result?.message || 'native drag failed' }),
        });
      }
      return result;
    } catch (error) {
      setToolResult({toolId: personaId, ok: false, message: t('persona.dragExportFail', { message: error?.message || error })});
      return null;
    }
  }

  async function finalizeNativePersonaExport(action, exportTarget) {
    if (!exportTarget?.persona_id && !exportTarget?.id) return;
    try {
      const personaId = exportTarget.persona_id || exportTarget.id;
      const result = await callWails(() => FinalizeNativePersonaExport(
        action,
        personaId,
        exportTarget.export_path || '',
        exportTarget.landed_path || '',
      ));
      if (result?.state) {
        const normalized = normalizeSettingsState(result.state, settingsState);
        setSettingsState(normalized);
        const activePersona = findActivePersona(normalized);
        if (activePersona?.name) setPersonaName(activePersona.name);
        if (activePersona?.identity) setPersonaJob(activePersona.identity);
      }
      setToolResult({
        toolId: personaId,
        ok: true,
        message: action === 'cancel'
          ? t('persona.deletedJustDragged')
          : action === 'remove'
            ? t('persona.fileCopiedAndRemoved')
            : t('persona.fileCopied'),
      });
    } catch (error) {
      const personaId = exportTarget.persona_id || exportTarget.id;
      setToolResult({toolId: personaId, ok: false, message: t('persona.exportHandleFail', { message: error?.message || error })});
    }
  }


  // v3.3.2 P0.1 — Package drag quarantine flow
  // Called from PersonaSettingsDrawer on file drop; routes through prepare_install.
  async function dropPersonaPackage(fileName, fileContent) {
    if (settingsState.personas.length >= maxPersonas) return;
    let parsedData;
    try {
      parsedData = JSON.parse(fileContent);
    } catch {
      parsedData = {name: fileName.replace(/\.[^.]+$/, ''), identity: t('persona.droppedRolepackIdentity')};
    }
    const manifestJSON = JSON.stringify({
      name: parsedData.name || fileName.replace(/\.[^.]+$/, ''),
      version: parsedData.version || '0.0.1',
      source_path: fileName,
      risk_tag: parsedData.risk_tag || 'unknown',
      required_perms: parsedData.required_perms || [],
      write_targets: [],
    });
    try {
      const pending = await callWails(() => PreparePackageInstallPayload(fileName, fileContent, manifestJSON));
      setPendingImport(pending);
      setPendingPackageData(pending?.persona || parsedData);
      setPackageInstallError('');
    } catch (error) {
      // If quarantine backend not reachable, block install (never bypass quarantine).
      setPackageInstallError(error?.message || t('persona.rolepackSecurityFail'));
    }
  }

  async function confirmPackageInstall() {
    if (!pendingImport) return;
    try {
      await callWails(() => ConfirmPackageInstall(pendingImport.id));
    } catch (error) {
      setPackageInstallError(error?.message || t('persona.rolepackInstallFail'));
      return;
    }
    await addPersona(pendingPackageData || {});
    setPendingImport(null);
    setPendingPackageData(null);
    setPackageInstallError('');
  }

  async function rejectPackageInstall() {
    if (!pendingImport) return;
    try {
      await callWails(() => RejectPackageInstall(pendingImport.id, 'cancelled'));
    } catch {
      /* best-effort reject */
    }
    setPendingImport(null);
    setPendingPackageData(null);
    setPackageInstallError('');
  }

  // I-6 (#I-602): Reauth Intercept — unavailable 工具點擊時攔截，
  // 彈出 reauth dialog 而非直接執行。lightning-fracture overlay 已有視覺提示，
  // 此處補上行為攔截，確保 unavailable 工具不會靜默呼叫後端。
  async function activateTool(toolId) {
    // #5 Degraded Mode guard: 降級時阻擋工具執行
    if (degradedState.active) {
      setToolResult({toolId, ok: false, message: t('tool.degradedPaused')});
      return;
    }
    // #6 Read-only guard: onboarding 未完成時阻擋
    if (readOnlyMode) {
      setToolResult({toolId, ok: false, message: t('tool.readonlyOnboarding')});
      return;
    }
    const tool = tools.find((t) => t.id === toolId);
    if (tool && !tool.enabled) {
      setReauthTool(tool);
      return; // 攔截：不執行，等用戶在 dialog 決定
    }
    try {
      setToolResult(await callWails(() => ActivateTool(toolId)));
    } catch {
      setToolResult({toolId, ok: false, message: t('tool.bindingNoResponse')});
    }
  }

  // I-6 (#I-602): 用戶在 reauth dialog 選擇「重試」時強制執行
  async function forceActivateTool(toolId) {
    setReauthTool(null);
    try {
      setToolResult(await callWails(() => ActivateTool(toolId)));
    } catch {
      setToolResult({toolId, ok: false, message: t('tool.reauthFail')});
    }
  }

  // I-6 (#I-601): 工具拖曳排序 — 拖曳結束後呼叫 SetToolPreference 儲存新 rank。
  // scope 預設 "global"，dagRevision 留空（非 DAG 內排序時不需要）。
  // rank 為目標位置的 0-based index。
  async function reorderTool(toolId, newRank) {
    try {
      await callWails(() => SetToolPreference(toolId, 'global', '', newRank));
    } catch {
      /* best-effort: 後端不可用時前端順序仍生效，下次啟動重載 */
    }
  }

  function toggleToolPopup(side) {
    setActivePanel(null);
    setToolPopupsOpen((current) => ({...current, [side]: !current[side]}));
  }

  function addFavoriteTool(toolId) {
    setFavoriteToolIds((current) => (current.includes(toolId) ? current : [...current, toolId]));
    setToolResult({toolId, ok: true, message: t('tool.favoriteAdded')});
    setDraggedTool(null);
  }

  function removeFavoriteOnly(toolId) {
    setFavoriteToolIds((current) => current.filter((id) => id !== toolId));
    setToolResult({toolId, ok: true, message: t('tool.favoriteRemoved')});
    setDragActionTool(null);
  }

  function removeToolCompletely(toolId) {
    setFavoriteToolIds((current) => current.filter((id) => id !== toolId));
    setHiddenToolIds((current) => (current.includes(toolId) ? current : [...current, toolId]));
    setToolResult({toolId, ok: true, message: t('tool.fullRemoved')});
    setDragActionTool(null);
  }

  function copyTool(toolAction) {
    setDragActionTool(null);
    setCopyConfirmTool(toolAction.tool);
  }

  function reorderReferenceFile(draggedKey, targetKey, placement = 'before') {
    setReferenceFiles((current) => reorderReferenceFiles(current, draggedKey, targetKey, placement));
  }

  function handleReferenceInternalDrag(active) {
    referenceInternalDragRef.current = !!active;
    if (!active) {
      referenceDropSuppressUntilRef.current = Date.now() + 350;
    }
  }

  async function confirmCopyTool() {
    if (!copyConfirmTool) return;
    setToolResult({toolId: copyConfirmTool.id, ok: true, message: t('tool.skillInstallConfirmed', { title: copyConfirmTool.title })});
    setCopyConfirmTool(null);
  }

  async function refreshReferenceFiles() {
    // §3.1.11 影片存 data/videos，與引用庫合併成同一份清單顯示
    const [files, videos] = await Promise.all([
      callWails(ListReferenceFiles).catch(() => []),
      callWails(ListVideoFiles).catch(() => []),
    ]);
    const loadedFiles = [
      ...(Array.isArray(files) ? files : []),
      ...(Array.isArray(videos) ? videos : []),
    ];
    setReferenceFiles((current) => mergeReferenceLibraryFiles(current, loadedFiles));
    return loadedFiles;
  }

  function formatReferenceNativeDropDetail(result) {
    const landed = result?.landed_path || result?.message || '引用文件已拖出';
    if (!result?.drop_target_kind || !result?.drop_target_dir) return landed;
    return `${landed}\n${result.drop_target_kind}: ${result.drop_target_dir}`;
  }

  function showNativeReferenceExportDialog(result, fallbackFile = null) {
    if (result?.status !== 'success' || !result?.landed_path) return;
    setReferenceExportDialog({
      name: result.display_name || fallbackFile?.name || '引用文件',
      sourcePath: result.source_path || fallbackFile?.path || '',
      landedPath: result.landed_path,
      landedDetail: formatReferenceNativeDropDetail(result),
    });
  }

  useEffect(() => {
    const offNativeReferenceExport = EventsOn('reference:native_completed', (result) => {
      showNativeReferenceExportDialog(result);
    });
    return () => offNativeReferenceExport();
  }, []);

  async function startNativeReferenceExport(file) {
    if (!file?.path || file.source !== 'library') {
      setToolResult({toolId: 'doc-entrance', ok: false, message: '只有引用庫內的本機檔案可以拖出複製'});
      return null;
    }
    try {
      const result = await callWails(() => NativeDragExportReferenceFile(file.path));
      if (result?.status === 'success' && result?.landed_path) {
        showNativeReferenceExportDialog(result, file);
      } else if (result?.status !== 'cancelled') {
        setToolResult({toolId: 'doc-entrance', ok: false, message: result?.message || '引用文件拖曳失敗'});
        await refreshReferenceFiles();
      }
      return result;
    } catch (error) {
      setToolResult({toolId: 'doc-entrance', ok: false, message: error?.message || String(error)});
      await refreshReferenceFiles();
      return null;
    }
  }

  async function handleReferenceExportAction(action) {
    if (!referenceExportDialog) return;
    const target = referenceExportDialog;
    setReferenceExportDialog(null);
    try {
      await callWails(() => FinalizeNativeReferenceFileExport(
        action,
        target.sourcePath || '',
        target.landedPath || '',
      ));
      if (action === 'remove') {
        setReferenceFiles((current) => current.filter((file) => file.path !== target.sourcePath));
      }
      if (action === 'cancel') {
        setToolResult({toolId: 'doc-entrance', ok: true, message: '已刪除剛剛拖出的複製檔'});
      } else if (action === 'remove') {
        setToolResult({toolId: 'doc-entrance', ok: true, message: '引用檔已複製，並已從引用庫移除'});
      } else {
        setToolResult({toolId: 'doc-entrance', ok: true, message: '引用檔已複製'});
      }
      await refreshReferenceFiles();
    } catch (error) {
      setToolResult({toolId: 'doc-entrance', ok: false, message: error?.message || String(error)});
    }
  }

  async function importReferencePaths(paths = []) {
    const nativePaths = normalizeReferenceImportPaths(paths);
    for (const path of nativePaths) {
      const name = String(path || '').split(/[\/]/).pop() || t('system.unnamedFile');
      let referencePathForStatus = path;
      setReferenceFiles((current) => appendUniqueReferenceFile(current, {
        name,
        path,
        source: 'pending',
        status: 'importing',
        detail: '正在複製到引用庫',
      }));
      try {
        // §3.1.11 影片落獨立資料夾 data/videos，其餘維持引用庫；UI 仍同一份清單
        const imported = await callWails(() => (isVideoPath(path) ? ImportVideoFile(path) : ImportReferenceFile(path)));
        const importedFile = {
          ...imported,
          status: isW3AMediaPath(path) ? 'checking' : 'ready',
          detail: isW3AMediaPath(path) ? '正在檢查媒體來源' : '',
        };
        referencePathForStatus = importedFile.path || path;
        setReferenceFiles((current) => appendUniqueReferenceFile(
          current.filter((file) => file.path !== path),
          importedFile,
        ));
        setToolResult({
          toolId: 'doc-entrance',
          ok: true,
          message: `${isW3AMediaPath(path) ? '已加入引用媒體' : '已加入引用文件'}：${importedFile.name || name}`,
        });
      } catch (error) {
        const message = error?.message || '無法複製，已保留原路徑';
        setReferenceFiles((current) => appendUniqueReferenceFile(current.filter((file) => file.path !== path), {
          name,
          path,
          source: 'memory',
          status: 'error',
          detail: message,
        }));
        setToolResult({toolId: 'doc-entrance', ok: false, message});
      }
      if (isW3AMediaPath(path)) {
        try {
          await importMediaW3A(path);
          setReferenceFiles((current) => updateReferenceFileStatus(current, referencePathForStatus, {
            status: 'ready',
            detail: '媒體來源檢查完成',
          }));
        } catch (error) {
          const message = error?.message || '媒體來源檢查失敗';
          setReferenceFiles((current) => updateReferenceFileStatus(current, referencePathForStatus, {
            status: 'error',
            detail: message,
          }));
          setToolResult({toolId: 'doc-entrance', ok: false, message});
        }
      }
    }
  }

  async function importReferenceFileObjects(files = []) {
    const nativePaths = Array.from(files).map((file) => file.path).filter(Boolean);
    if (nativePaths.length) {
      await importReferencePaths(nativePaths);
      return;
    }
    const nextFiles = Array.from(files).map((file) => ({
      name: file.name,
      path: file.name,
      source: 'memory',
      status: 'ready',
      detail: '已加入引用清單',
    }));
    setReferenceFiles((current) => nextFiles.reduce(appendUniqueReferenceFile, current));
  }

  // --- 遺留能力 #1–#8 action functions ---

  // #2 Review Card: 從 Go review service 拉取 open cards + blocking flag
  // #I-1002: 同時拉取 review_archive 以顯示 rejected 歷史
  // v3.6: 同時拉取 Lightweight Cards
  async function refreshReviewCards() {
    callWails(ListOpenReviewCards)
      .then((cards) => setReviewCards(cards || []))
      .catch(() => {});
    callWails(HasBlockingReviewCard)
      .then((b) => setHasBlocking(!!b))
      .catch(() => {});
    callWails(ListReviewArchive)
      .then((cards) => setReviewArchive(cards || []))
      .catch(() => {});
    // v3.6: Lightweight Review Cards（低風險自動通過類）
    callWails(ListOpenLightweightCards)
      .then((cards) => setLightweightCards(cards || []))
      .catch(() => {});
  }

  // #2 Review Card: 解決單張 card
  async function resolveReviewCard(cardID) {
    try {
      await callWails(() => ResolveReviewCard(cardID));
      refreshReviewCards();
    } catch { /* best-effort */ }
  }

  // v3.6: 解決 Lightweight Card
  async function resolveLightweightCard(cardID) {
    try {
      await callWails(() => ResolveLightweightCard(cardID));
      refreshReviewCards();
    } catch { /* best-effort */ }
  }

  // v3.6: 雙步驟確認 — Step1 啟動
  async function dualStepConfirm(cardID) {
    try {
      const result = await callWails(() => DualStepConfirmStep1(cardID));
      return result;
    } catch { return null; }
  }

  // v3.6: 雙步驟確認 — 取消（使 Step1 token 失效）
  async function invalidateDualStep(cardID) {
    try {
      await callWails(() => InvalidateDualStep(cardID));
    } catch { /* best-effort */ }
  }

  // v3.6: StatusRail 確認收到
  async function acknowledgeStatusRail(railID) {
    try {
      await callWails(() => AcknowledgeStatusRail(railID));
    } catch { /* best-effort */ }
  }

  // #1 Adapter: 設定 adapter 狀態（online/offline/error）
  async function updateAdapterStatus(adapterID, status) {
    try {
      await callWails(() => SetAdapterStatus(adapterID, status));
      await refreshAvailableAdapters();
    } catch { /* best-effort */ }
  }

  // #5 Degraded Mode: 進入 / 退出
  async function enterDegraded(reason) {
    try {
      const s = await callWails(() => EnterDegradedMode(reason));
      setDegradedState(s || {active: true, blocked_ops: []});
      // 進入 degraded mode 時，作廢所有 review cards（同 integration_test 驗證流程）
      refreshReviewCards();
    } catch { /* best-effort */ }
  }

  async function exitDegraded() {
    try {
      const s = await callWails(ExitDegradedMode);
      setDegradedState(s || {active: false, blocked_ops: []});
    } catch { /* best-effort */ }
  }

  // #5 Degraded Mode: 檢查操作是否被阻擋
  async function checkDegradedBlocked(opCategory) {
    try {
      return await callWails(() => IsDegradedBlocked(opCategory));
    } catch {
      return false;
    }
  }

  // #6 Onboarding: 完成單步
  async function completeOnboardStep(stepID) {
    try {
      const s = await callWails(() => CompleteOnboardingStep(stepID));
      setOnboardingState(s || null);
    } catch { /* best-effort */ }
  }

  // #6 Onboarding: 完成整個 onboarding，退出 read-only
  async function finishOnboard() {
    try {
      await callWails(FinishOnboarding);
      setReadOnlyMode(false);
      setOnboardingState((prev) => prev ? {...prev, is_first_run: false, read_only_mode: false} : prev);
    } catch { /* best-effort */ }
  }

  // I-5: 從後端拉取三種 link_type 的已註冊連結
  async function refreshExternalLinks() {
    callWails(() => ListExternalLinksByType('external_service'))
      .then((links) => setExtServiceLinks(links || []))
      .catch(() => {});
    callWails(() => ListExternalLinksByType('adapter_candidate'))
      .then((links) => setExtAdapterLinks(links || []))
      .catch(() => {});
    callWails(() => ListExternalLinksByType('documentation'))
      .then((links) => setExtDocLinks(links || []))
      .catch(() => {});
  }

  async function refreshLinkPreviewSuggestions(shouldSuggest) {
    if (!shouldSuggest) {
      setLinkPreviewSuggestions([]);
      return;
    }
    try {
      const detected = await callWails(AutoDetectCLI);
      const detectedSuggestions = (detected || [])
        .filter((item) => item?.found && item?.path)
        .map((item) => ({
          name: item.name || item.adapter_id || 'CLI',
          path: item.path,
          detected: true,
        }));
      setLinkPreviewSuggestions(detectedSuggestions.slice(0, 6));
    } catch {
      setLinkPreviewSuggestions([]);
    }
  }

  function defaultLLMModelForProvider(providerID) {
    const defaults = {
      deepseek: 'deepseek-chat',
      openai: 'gpt-4.1-mini',
      anthropic: 'claude-sonnet-4-5',
      gemini: 'gemini-2.5-flash',
      xai: 'grok-4',
      openrouter: 'openai/gpt-4.1-mini',
      mistral: 'mistral-small-latest',
      groq: 'llama-3.3-70b-versatile',
      together: 'meta-llama/Llama-3.3-70B-Instruct-Turbo',
      perplexity: 'sonar',
      cohere: 'command-a-03-2025',
      fireworks: 'accounts/fireworks/models/llama-v3p1-70b-instruct',
      cerebras: 'llama3.1-8b',
    };
    return defaults[providerID] || '';
  }

  async function submitLLMAPISetup() {
    if (!llmAPISetup) return;
    try {
      const result = await callWails(() => RegisterLLMAPIAdapter(
        llmAPISetup.providerId || 'generic-api',
        llmAPISetup.providerName || 'LLM API',
        llmAPISetup.baseURL || '',
        llmAPISetup.model || '',
        llmAPISetup.apiKey || '',
      ));
      // SEC-03: 後端回傳 need_confirm 時，彈確認框讓使用者決定
      if (result?.need_confirm && result?.confirm_type === 'private_network') {
        const ok = window.confirm(
          t('link.privateNetworkConfirm', { host: result.hostname || result.original_url })
        );
        if (!ok) {
          setToolResult({toolId: 'reference-link', ok: false, message: t('link.privateNetworkCancel')});
          return;
        }
        // 使用者確認 → 呼叫 ConfirmRegisterLLMAPIAdapter 跳過 SSRF 檢查
        const confirmed = await callWails(() => ConfirmRegisterLLMAPIAdapter(
          llmAPISetup.providerId || 'generic-api',
          llmAPISetup.providerName || 'LLM API',
          llmAPISetup.baseURL || '',
          llmAPISetup.model || '',
          llmAPISetup.apiKey || '',
        ));
        setLlmAPISetup(null);
        await refreshAvailableAdapters();
        setAdapterRenameTarget({
          id: confirmed?.adapter_id,
          name: confirmed?.name || llmAPISetup.providerName || 'LLM API',
        });
        setAdapterRenameDraft(confirmed?.name || llmAPISetup.providerName || 'LLM API');
        setToolResult({toolId: 'reference-link', ok: true, message: t('link.adapterCreatedPrivate')});
        return;
      }
      setLlmAPISetup(null);
      await refreshAvailableAdapters();
      setAdapterRenameTarget({
        id: result?.adapter_id,
        name: result?.name || llmAPISetup.providerName || 'LLM API',
      });
      setAdapterRenameDraft(result?.name || llmAPISetup.providerName || 'LLM API');
      setToolResult({toolId: 'reference-link', ok: true, message: t('link.adapterCreated')});
    } catch (err) {
      setToolResult({toolId: 'reference-link', ok: false, message: err?.message || t('link.adapterCreateFail')});
    }
  }

  function openWebSearchSetup(config = webSearchConfig) {
    const options = Array.isArray(config?.options) ? config.options : defaultWebSearchProviderOptions();
    const providerId = config?.provider_id || options[0]?.id || 'tavily';
    setWebSearchSetup({
      providerId,
      apiKey: '',
      cx: '',
      options,
    });
    setWebSearchSetupError('');
  }

  async function submitWebSearchSetup() {
    if (!webSearchSetup) return;
    setWebSearchSetupError('');
    try {
      const next = await callWails(() => SaveWebSearchConfig(
        webSearchSetup.providerId || 'tavily',
        webSearchSetup.apiKey || '',
        webSearchSetup.cx || '',
      ));
      setWebSearchConfig(next || null);
      setWebSearchSetup(null);
    } catch (err) {
      setWebSearchSetupError(err?.message || String(err));
    }
  }

  async function clearWebSearchSetup() {
    setWebSearchSetupError('');
    try {
      const next = await callWails(ClearWebSearchConfig);
      setWebSearchConfig(next || null);
      setWebSearchSetup(null);
    } catch (err) {
      setWebSearchSetupError(err?.message || String(err));
    }
  }

  function previewLLMAPIConnection() {
    if (!llmAPISetup) return;
    const missing = [];
    if (!llmAPISetup.apiKey?.trim()) missing.push('API Key');
    if (!llmAPISetup.baseURL?.trim()) missing.push('Base URL');
    if (!llmAPISetup.model?.trim()) missing.push('Model');
    setToolResult({
      toolId: 'reference-link',
      ok: missing.length === 0,
      message: missing.length
        ? t('link.missingFields', { fields: missing.join('、') })
        : t('link.fieldsComplete'),
    });
  }

  async function saveAdapterRename() {
    if (!adapterRenameTarget?.id) return;
    const nextName = adapterRenameDraft.trim() || adapterRenameTarget.name || 'LLM API';
    try {
      await callWails(() => RenameAdapter(adapterRenameTarget.id, nextName));
      setAdapterRenameTarget(null);
      setAdapterRenameDraft('');
      await refreshAvailableAdapters();
    } catch (err) {
      setToolResult({toolId: 'reference-link', ok: false, message: err?.message || t('link.adapterRenameFail')});
    }
  }

  // I-5 (#I-502): 輸入連結 → PreviewExternalLink → 顯示預覽 → 確認後 Register
  //
  // v3.6.3 新增行為：
  //  使用者貼 URL 按「預覽」時，先走 Remote Bridge 偵測（DetectRemoteBridgeChannel）。
  //  若匹配 Telegram/Discord/LINE 的 URL pattern → 設定 linkPreview.type = 'remote_bridge'，
  //  ReferenceLinkModal 會顯示「偵測到：Telegram Bot API」之類的提示。
  //  按「確定」時 confirmReferenceLink 會走 detectAndRegisterRemoteBridge 流程。
  //  若 URL 不匹配任何通訊平台 → fallback 到原本的 PreviewExternalLink 流程。
  async function previewReferenceLink() {
    const value = referenceLinkValue.trim();
    if (!value) return;
    setLinkPreviewError('');
    setLinkPreviewSuggestions([]);

    const looksLikeURL = /^[a-z][a-z0-9+.-]*:\/\//i.test(value);
    const looksLikeLocalPath = looksLikeReferenceLocalPath(value);
    if (!looksLikeURL && !looksLikeLocalPath) {
      const message = t('link.previewHint');
      setToolResult({
        toolId: 'reference-link',
        ok: false,
        message,
      });
      setLinkPreviewError(message);
      setLinkPreview(null);
      refreshLinkPreviewSuggestions(true);
      return;
    }

    // v3.6.3 §12A: 先偵測是否為通訊軟體 Bot/Webhook URL
    try {
      const detected = await callWails(() => DetectRemoteBridgeChannel(value));
      if (detected?.matched) {
        setLinkPreview({
          valid: true,
          type: 'remote_bridge',
          channel: detected.channel,
          hint_label: detected.hint_label,
          url_type: detected.url_type,
          setup_required: detected.url_type === 'platform_site',
        });
        return; // 不走一般外部連結流程
      }
    } catch { /* 偵測失敗繼續走一般流程 */ }

    try {
      const preview = await callWails(() => PreviewExternalLink(value));
      setLinkPreview(preview);
      // unsupported 或 invalid → 直接告知，不進 Register
      if (!preview?.valid) {
        const message = preview?.reason || t('link.unsupportedFormat');
        setToolResult({toolId: 'reference-link', ok: false, message});
        setLinkPreviewError(message);
        setLinkPreview(null);
        refreshLinkPreviewSuggestions(looksLikeLocalPath);
      }
    } catch (error) {
      const message = error?.message || t('link.previewFail');
      setToolResult({toolId: 'reference-link', ok: false, message});
      setLinkPreviewError(message);
      setLinkPreview(null);
      refreshLinkPreviewSuggestions(looksLikeLocalPath);
    }
  }

  // I-5 (#I-502): 用戶確認預覽結果後，才呼叫 RegisterExternalLink
  // v3.6.3: 若為 Remote Bridge 通道，走連線測試 + 註冊通道流程
  async function confirmReferenceLink() {
    const value = referenceLinkValue.trim();
    if (!value) return;
    setLinkPreviewError('');
    setLinkPreviewSuggestions([]);
    const looksLikeLocalPath = looksLikeReferenceLocalPath(value);

    // v3.6.3 §12A: Remote Bridge 通道註冊流程
    if (linkPreview?.type === 'remote_bridge') {
      if (linkPreview.setup_required || linkPreview.url_type === 'platform_site') {
        openRemoteBridgeSetupForPlatform(linkPreview.channel);
        setReferenceLinkValue('');
        setReferenceLinkOpen(false);
        setLinkPreview(null);
        return;
      }
      const result = await detectAndRegisterRemoteBridge(value);
      if (result.success) {
        setToolResult({toolId: 'reference-link', ok: true, message: t('link.channelRegistered', { label: result.detected?.hint_label || t('link.defaultChannel') })});
        openRemoteBridgeRename(result.binding);
      } else {
        setToolResult({toolId: 'reference-link', ok: false, message: result.error || t('link.channelRegisterFail')});
      }
      setReferenceLinkValue('');
      setReferenceLinkOpen(false);
      setLinkPreview(null);
      return;
    }

    if (linkPreview?.link_type === 'llm_provider_candidate') {
      setLlmAPISetup({
        providerId: linkPreview.provider_id || 'generic-api',
        providerName: linkPreview.provider_name || 'LLM API',
        baseURL: linkPreview.base_url || value,
        apiKeyURL: linkPreview.api_key_url || '',
        docsURL: linkPreview.docs_url || '',
        model: defaultLLMModelForProvider(linkPreview.provider_id),
        apiKey: '',
      });
      setLlmAPISetupGuideStep(0);
      setReferenceLinkValue('');
      setReferenceLinkOpen(false);
      setLinkPreview(null);
      return;
    }

    // v3.6: Source Trust — 註冊前先分類來源，顯示信任提示
    if (!looksLikeLocalPath) {
      callWails(() => ClassifySource(value, '', []))
        .then((evidence) => {
          const hint = sourceTrustToHint(evidence);
          if (hint) setSourceTrustHint(hint);
        })
        .catch(() => {});
    }
    try {
      const link = await callWails(() => RegisterExternalLink(value, value));
      const isAdapterCandidate = link?.link_type === 'adapter_candidate' || linkPreview?.link_type === 'adapter_candidate';
      const isOllamaLibrary = isAdapterCandidate && /ollama/i.test(`${link?.label || ''} ${link?.url || ''} ${linkPreview?.reason || ''}`);
      setToolResult({toolId: 'reference-link', ok: true, message: isAdapterCandidate ? (isOllamaLibrary ? t('link.addedOllama') : t('link.addedCLI')) : t('link.addedLink')});
      // 註冊成功後刷新三路分流資料
      refreshExternalLinks();
      if (isAdapterCandidate) {
        refreshAvailableAdapters().catch(() => {});
      }
    } catch (error) {
      const message = error?.message || t('link.linkLocalHint');
      setToolResult({toolId: 'reference-link', ok: false, message});
      setLinkPreviewError(message);
      refreshLinkPreviewSuggestions(looksLikeLocalPath);
      return;
    }
    setReferenceFiles((current) => [...current, {name: value, path: value, source: 'link', status: 'ready', detail: '外部引用連結'}]);
    setReferenceLinkValue('');
    setReferenceLinkOpen(false);
    setLinkPreview(null);
  }

  async function createSubagent() {
    try {
      const created = await callWails(() => CreateSubagent(''));
      await refreshAvailableAdapters();
      setToolResult({
        toolId: 'subagent',
        ok: true,
        message: t('subagent.created', { name: created?.name || 'haㄌer', dir: created?.memory_dir || created?.sub_dir || '' }),
      });
    } catch (error) {
      setToolResult({toolId: 'subagent', ok: false, message: t('subagent.createFail', { error: error?.message || error })});
    }
  }

  async function renameSubagent(currentName, nextName) {
    if (!currentName || currentName === t('system.mainAgent')) return;
    const trimmed = String(nextName || '').trim();
    if (!trimmed || trimmed === currentName) return;
    try {
      const renamed = await callWails(() => RenameSubagent(currentName, trimmed));
      await refreshAvailableAdapters();
      setToolResult({toolId: 'subagent', ok: true, message: t('adapter.handlerRenamed', { name: renamed?.name || trimmed })});
      return renamed;
    } catch (error) {
      setToolResult({toolId: 'subagent', ok: false, message: t('adapter.handlerRenameFail', { error: error?.message || error })});
      return null;
    }
  }

  // Stores a lightweight "new skill/sub digest" hint for the next app open.
  function markLearningDigestReady() {
    setLearningDigestReady(true);
    try {
      window.localStorage.setItem(learningDigestStorageKey, 'true');
    } catch {
      /* local notification hint only */
    }
  }

  function confirmLearningBackground(event) {
    event.preventDefault();
    event.returnValue = t('adapter.learningModeHint');
    return event.returnValue;
  }

  // v3.6: 學習模式切換 — 對接後端 LearningService
  async function toggleLearning() {
    const conversationId = activeConversationIdRef.current || 'main';
    const backendActive = await callWails(IsLearningModeActive).catch(() => learningEnabled);
    const wasEnabled = learningEnabled || !!backendActive;
    if (wasEnabled) {
      // 關閉學習模式
      try {
        const traceId = makeDebugTraceID('learning-metadata');
        const run = await callWails(StopLearningMode);
        const namedRun = await enrichStoppedLearningRun(run, traceId);
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          formatLearningOperationLearned(namedRun),
        ]);
      } catch (error) {
        const detail = error?.message || String(error || '');
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          `[系統] 示範結束，但後端停止記錄時回報錯誤。${detail ? ` ${detail}` : ''}`,
        ]);
      }
      setLearningEnabled(false);
      setVlLearningActive(false);
      setVlActiveLearningRun(null);
      setLearningDigestReady(false);
      try { window.localStorage.removeItem(learningDigestStorageKey); } catch { /* */ }
    } else {
      // 開啟學習模式（需要 activeWindowHash，目前用 session ID）
      try {
        const run = await callWails(() => StartLearningMode('window-' + Date.now()));
        setVlActiveLearningRun(run);
        setVlLearningActive(true);
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          `[系統] 示範開始。${run?.id ? `Run: ${run.id}。` : ''}請點一次你要教我的目標。`,
        ]);
      } catch (error) {
        const detail = error?.message || String(error || '');
        const activeRun = detail.includes('already recording')
          ? await callWails(GetActiveLearningRun).catch(() => null)
          : null;
        if (activeRun) {
          setVlActiveLearningRun(activeRun);
          setVlLearningActive(true);
          setLearningEnabled(true);
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            `[系統] 已接回仍在進行中的示範。${activeRun?.id ? `Run: ${activeRun.id}。` : ''}再按一次可停止記錄。`,
          ]);
          return;
        }
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          `[系統] 示範開始，但後端記錄沒有啟動。${detail ? ` ${detail}` : ''}`,
        ]);
      }
      setLearningEnabled(true);
    }
  }

  // v3.6: 定期同步 Visual Learning 後端狀態
  async function refreshVisualLearningState() {
    callWails(IsLearningModeActive)
      .then((active) => setVlLearningActive(!!active))
      .catch(() => {});
    callWails(GetPendingCandidateCount)
      .then((count) => setVlPendingCount(count || 0))
      .catch(() => {});
    callWails(HasBlockingVLReview)
      .then((b) => setVlHasBlocking(!!b))
      .catch(() => {});
  }

  const mainPersona = settingsState.personas[0] || fallbackSettings.personas[0];
  const activePersona = findActivePersona(settingsState) || mainPersona;
  useEffect(() => {
    const tick = () => setAvatarClock(Date.now());
    const onVisibilityChange = () => {
      setWindowInactive(document.hidden);
      tick();
    };
    document.addEventListener('visibilitychange', onVisibilityChange);
    const timer = window.setInterval(tick, 60 * 1000);
    return () => {
      document.removeEventListener('visibilitychange', onVisibilityChange);
      window.clearInterval(timer);
    };
  }, []);

  const autoAvatarExpression = deriveAvatarExpression({
    dagRun,
    reviewState,
    sourceTrustHint,
    hasConversation: messages.length > 0 || draft.trim().length > 0,
    lastMessageText: messages[messages.length - 1] || draft,
    windowInactive,
    idleMs: avatarClock - appStartedAtRef.current,
  });
  const avatarExpression = manualAvatarState && !autoAvatarOverrideStates.has(autoAvatarExpression)
    ? manualAvatarState
    : autoAvatarExpression;
  const mainAvatarConfig = avatarConfigs[mainPersona.id] || null;
  const mainAvatarProvider = resolveAvatarProvider(mainAvatarConfig);
  const renderedPixelAvatar = renderedPixelAvatars[mainPersona.id] || '';
  const mainAvatarSrc = resolvePersonaAvatarSrc(mainPersona, mainAvatarConfig, staticAvatarPreviews, renderedPixelAvatar);

  useEffect(() => {
    if (mainPersona?.id) loadCurrentAvatar(mainPersona.id);
  }, [mainPersona.id]);

  useEffect(() => {
    // Settings cards can edit any persona avatar, so keep lightweight avatar
    // configs warm for each visible persona instead of only the main card.
    settingsState.personas.forEach((persona) => {
      if (persona?.id && !avatarConfigs[persona.id]) loadCurrentAvatar(persona.id);
    });
  }, [settingsState.personas.map((persona) => persona.id).join('|')]);

  useEffect(() => {
    let cancelled = false;
    // Render built-in pixel avatars per persona so A/B/C can keep different
    // expression packs while sharing the same current UI expression.
    settingsState.personas.forEach((persona) => {
      const config = avatarConfigs[persona.id] || null;
      const staticAvatarPath = resolveStaticAvatarPath(config);
      if (resolveAvatarProvider(config) === 'static_image' && staticAvatarPath) return;
      const pack = pixelPackForPersona(persona, config);
      getPixelAvatarDataUrl(pack, avatarExpression, pixelAvatarRenderSize)
        .then((dataUrl) => {
          if (!cancelled && dataUrl) {
            setRenderedPixelAvatars((prev) => ({...prev, [persona.id]: dataUrl}));
          }
        })
        .catch(() => {
          if (!cancelled) setRenderedPixelAvatars((prev) => ({...prev, [persona.id]: getPersonaAvatar(persona)}));
        });
    });
    return () => {
      cancelled = true;
    };
  }, [settingsState.personas.map((persona) => persona.id).join('|'), JSON.stringify(avatarConfigs), avatarExpression]);

  useEffect(() => {
    const packs = new Set(settingsState.personas.map((persona) => {
      const config = avatarConfigs[persona.id] || null;
      return pixelPackForPersona(persona, config);
    }));
    packs.forEach((pack) => {
      avatarStateOptions.forEach((state) => {
        getPixelAvatarDataUrl(pack, state, pixelAvatarRenderSize).catch(() => {});
      });
    });
  }, [settingsState.personas.map((persona) => persona.id).join('|'), JSON.stringify(avatarConfigs)]);

  /* i18n: apply language + direction attributes to root shell */
  const _i18nLang = useI18n(s => s.language);
  const _i18nDir  = useI18n(s => s.getDirection)();

  function handleGlobalFileDragOver(event) {
    const types = Array.from(event.dataTransfer?.types || []);
    if (!types.includes('Files')) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = 'copy';
  }

  function handleGlobalFileDrop(event) {
    if (event.defaultPrevented) return;
    if (referenceInternalDragRef.current || Date.now() < referenceDropSuppressUntilRef.current) return;
    const files = Array.from(event.dataTransfer?.files || []);
    if (!files.length) return;
    event.preventDefault();
    const nativePaths = normalizeReferenceImportPaths(files.map((file) => file.path));
    if (nativePaths.length) {
      importReferencePaths(nativePaths);
      if (shouldProbeDroppedInstallPackage(nativePaths)) {
        detectDroppedInstallPackage(nativePaths[0]);
      }
      return;
    }
    importReferenceFileObjects(files);
  }

  return (
    <div
      className={`console-shell ${activePanel === 'settings' ? 'settings-open' : ''}`}
      data-theme={panelStyleTheme(settingsState.panel.panelStyle)}
      data-lang={_i18nLang}
      dir={_i18nDir}
      onDragOver={handleGlobalFileDragOver}
      onDrop={handleGlobalFileDrop}
      style={{'--ui-font-scale': fontScaleValue(settingsState.panel.fontScale)}}
    >
      {/* v2.4: 首次啟動引導覆蓋層 */}
      <OnboardingOverlay
        state={onboardingState}
        onCompleteStep={completeOnboardStep}
        onFinish={finishOnboard}
        onGoBack={() => callWails(GoBackOnboarding).then(setOnboardingState)}
        onDetectCLI={() => callWails(AutoDetectCLI)}
        onEnableCLI={(adapterID) => callWails(() => EnableDetectedCLI(adapterID))}
        onScanLocalModels={() => callWails(ScanLocalModels)}
        onEnableLocalModel={(r) => callWails(() => EnableLocalModel(r.adapter_id, r.name, r.model_id, r.provider, r.endpoint))}
        onRegisterCustomCLI={(name, path) => callWails(() => RegisterCustomCLI(name, path))}
        onOpenSettings={() => setActivePanel('settings')}
        onCloseSettings={() => setActivePanel(null)}
      />
      <Sidebar
        adapters={state.adapters}
        adapterList={adapterList}
        activeAdapterId={activeAdapterId}
        onAdapterSelect={async (name) => {
          activeAdapterIdRef.current = name;
          setActiveAdapterId(name);
          // Selection only changes routing. Runtime health is updated by real
          // send success/failure so a broken adapter is not painted green.
        }}
        adapterModelChoices={adapterModelChoices}
        adapterModelOptions={adapterModelOptions}
        onAdapterModelPick={handleAdapterModelPick}
        onAdapterModelRefresh={handleAdapterModelRefresh}
        onLocalAdapterWake={async (adapter) => {
          if (!adapter?.id) return;
          setToolResult({toolId: adapter.id, ok: true, message: t('adapter.wakingModel')});
          try {
            const result = await callWails(() => WakeLocalAdapter(adapter.id));
            await refreshAvailableAdapters();
            setToolResult({toolId: adapter.id, ok: true, message: result?.message || t('adapter.modelAwake')});
          } catch (err) {
            await refreshAvailableAdapters();
            setToolResult({toolId: adapter.id, ok: false, message: err?.message || t('adapter.modelWakeFail')});
          }
        }}
        adapterCandidateLinks={extAdapterLinks}
        activePanel={activePanel}
        isToolPopupOpen={toolPopupsOpen.left}
        panelSettings={settingsState.panel}
        onPanelChange={savePanelPatch}
        voiceState={voiceState}
        voiceInstallBusy={voiceInstallBusy}
        onVoiceSettingsChange={saveVoiceSettingsPatch}
        onVoiceSettingsRefresh={() => callWails(GetVoiceSettings).then((settings) => setVoiceState(settings || null))}
        onVoiceModelInstall={installVoiceBaseModel}
        onVoiceModelRemove={removeVoiceBaseModel}
        onRestoreDefaults={restoreUIDefaults}
        onTogglePanel={(panel) => {
          setToolPopupsOpen({left: false, right: false});
          setActivePanel((current) => (current === panel ? null : panel));
        }}
        onToolPopupToggle={() => toggleToolPopup('left')}
        onProjectManageOpen={openProjectManage}
        onCreateSubagent={createSubagent}
        onAdaptersReordered={(newOrder, lane) => {
          if (lane === 'sub') {
            setSubagentTabs((prev) => reorderItemsByKeys(prev, newOrder));
            return;
          }
          setAdapterList((prev) => reorderItemsByKeys(prev, newOrder));
        }}
        onAdaptersChanged={refreshAvailableAdapters}
        onAdapterRemove={async (adapterID) => {
          await callWails(() => UnregisterAdapter(adapterID));
          if (activeAdapterIdRef.current === adapterID) {
            activeAdapterIdRef.current = null;
            setActiveAdapterId(null);
          }
          await refreshAvailableAdapters();
          setToolResult({toolId: adapterID, ok: true, message: t('adapter.cliUnlinked')});
        }}
      />
      {installCandidate && (
        <PackageInstallDecisionDialog
          candidate={installCandidate}
          onConfirm={confirmInstallCandidate}
          onCancel={() => setInstallCandidate(null)}
        />
      )}
      {/* i18n: credential */}
      {credentialMigrationStatus && !credentialMigrationStatus.ready && (
        <div className="pkg-modal-overlay" role="dialog" aria-modal="true" aria-label={t('credential.storageUpgradeLabel')}>
          <div className="pkg-modal credential-migration-modal">
            <div className="pkg-modal-header">
              <strong>{t('credential.needsHandling')}</strong>
            </div>
            <div className="pkg-modal-body">
              {credentialMigrationStatus.required && !credentialMigrationStatus.disabled ? (
                <>
                  <div className="pkg-modal-name">{t('credential.oldVersionDetected')}</div>
                  <p className="pkg-modal-notice">
                    {t('credential.oldVersionBody')}
                  </p>
                </>
              ) : (
                <>
                  <div className="pkg-modal-name">{t('credential.unavailable')}</div>
                  <p className="pkg-modal-notice">
                    {t('credential.unavailableBody')}
                  </p>
                </>
              )}
              {credentialMigrationStatus.error && (
                <div className="pkg-modal-error">{credentialMigrationStatus.error}</div>
              )}
            </div>
            <div className="pkg-modal-actions">
              {credentialMigrationStatus.required && !credentialMigrationStatus.disabled ? (
                <>
                  <button className="pkg-modal-cancel" type="button" disabled={credentialMigrationBusy} onClick={disableCredentialMigration}>
                    {t('credential.reconfigureLater')}
                  </button>
                  <button className="pkg-modal-confirm" type="button" disabled={credentialMigrationBusy} onClick={confirmCredentialMigration}>
                    {t('credential.upgradeProtection')}
                  </button>
                </>
              ) : (
                <button className="pkg-modal-confirm" type="button" disabled={credentialMigrationBusy} onClick={() => setCredentialMigrationStatus(null)}>
                  {t('credential.gotIt')}
                </button>
              )}
            </div>
          </div>
        </div>
      )}
      {subImportResult?.tool_conflicts?.length > 0 && (
        <SubToolConflictDialog
          result={subImportResult}
          onResolve={resolveSubImportConflicts}
          onCancel={() => setSubImportResult(null)}
        />
      )}
      {/* v3.6: 專案管理彈窗 */}
      {projectManageOpen && (
        <ProjectManagePopup
          view={projectManageView}
          manifests={purgeManifests}
          confirmStep={purgeConfirmStep}
          onPurgeProject={handlePurgeProject}
          onPurgeBoundary={handlePurgeBoundary}
          onViewManifests={loadPurgeManifests}
          onBackToMenu={() => { setProjectManageView('menu'); setPurgeConfirmStep(null); }}
          onClose={() => { setProjectManageOpen(false); setPurgeConfirmStep(null); }}
        />
      )}
      <ConsequenceMenuOverlay
        cards={reviewCards.filter((card) => !dismissedDestructiveCards.includes(reviewCardID(card)))}
        result={destructiveReviewResult}
        onExecute={executeDestructiveReview}
        onRecreate={recreateDestructiveReview}
      />
      {remoteBridgeSetupOpen && (
        <div className="remote-bridge-setup-overlay">
          <div className="remote-bridge-setup-modal">
            {/* i18n: remoteBridgeSetup */}
            {remoteBridgeSetupStep === 'mode_select' && (
              <div className="rb-setup-step">
                <h4>{t('remoteBridgeSetup.title')}</h4>
                <div className="rb-mode-buttons">
                  <button type="button" className="rb-mode-btn" onClick={() => { setRemoteBridgeSetupMode('quick'); setRemoteBridgeSetupStep('quick_platform'); }}>
                    {t('remoteBridgeSetup.quickMode')}
                    <small>{t('remoteBridgeSetup.quickModeHint')}</small>
                  </button>
                  <button type="button" className="rb-mode-btn" onClick={() => { setRemoteBridgeSetupMode('developer'); setRemoteBridgeSetupStep('developer_form'); }}>
                    {t('remoteBridgeSetup.devMode')}
                    <small>{t('remoteBridgeSetup.devModeHint')}</small>
                  </button>
                </div>
                <button type="button" className="rb-cancel-btn" onClick={closeRemoteBridgeSetup}>{t('remoteBridgeSetup.cancel')}</button>
              </div>
            )}
            {remoteBridgeSetupStep === 'quick_platform' && (
              <div className="rb-setup-step">
                <h4>{t('remoteBridgeSetup.selectPlatform')}</h4>
                <div className="rb-platform-buttons">
                  <button type="button" onClick={() => selectQuickPlatform('telegram')}>Telegram</button>
                  <button type="button" onClick={() => selectQuickPlatform('discord')}>Discord</button>
                  <button type="button" onClick={() => selectQuickPlatform('line')}>LINE</button>
                  <button type="button" onClick={() => selectQuickPlatform('teams')}>Teams</button>
                  <button type="button" onClick={() => selectQuickPlatform('qq')}>QQ Bot</button>
                </div>
                <button type="button" className="rb-cancel-btn" onClick={() => setRemoteBridgeSetupStep('mode_select')}>{t('remoteBridgeSetup.back')}</button>
              </div>
            )}
            {remoteBridgeSetupStep === 'quick_fields' && (
              <div className="rb-setup-step">
                <h4>{remoteBridgeSetupPlatform?.toUpperCase()} — {t('remoteBridgeSetup.fillFields')}</h4>
                {remoteBridgeSetupPlatform === 'telegram' && (
                  remoteBridgeField('Bot Token', 'bot_token', {type: 'password', placeholder: '123456:ABC-DEF...'})
                )}
                {remoteBridgeSetupPlatform === 'discord' && (<>
                  <div className="rb-guide-card rb-discord-guide-card">
                    <span className="rb-guide-count">{t('remoteBridgeSetup.stepCount', { current: remoteBridgeSetupGuideStep + 1, total: discordSetupGuide.length })}</span>
                    <strong>{discordSetupGuide[remoteBridgeSetupGuideStep].title}</strong>
                    <p>{discordSetupGuide[remoteBridgeSetupGuideStep].body}</p>
                    <div className="rb-id-example">
                      {t('remoteBridgeSetup.channelIdHint')}
                    </div>
                    <div className="rb-guide-actions">
                      <button
                        type="button"
                        onClick={() => {
                          const step = discordSetupGuide[remoteBridgeSetupGuideStep];
                          openExternal(step.url || `https://www.google.com/search?q=${encodeURIComponent(step.query)}`);
                        }}
                      >
                        {discordSetupGuide[remoteBridgeSetupGuideStep].action}
                      </button>
                      {remoteBridgeSetupGuideStep < discordSetupGuide.length - 1 ? (
                        <button type="button" onClick={() => setRemoteBridgeSetupGuideStep((step) => Math.min(step + 1, discordSetupGuide.length - 1))}>{t('remoteBridgeSetup.next')}</button>
                      ) : (
                        <button type="button" onClick={() => setRemoteBridgeSetupGuideStep(0)}>{t('remoteBridgeSetup.restartGuide')}</button>
                      )}
                    </div>
                  </div>
                  {remoteBridgeField('Bot Token', 'bot_token', {type: 'password', placeholder: 'Discord Bot Token'})}
                  {remoteBridgeField('Server ID (Guild ID)', 'guild_id', {placeholder: t('remoteBridgeSetup.guildIdPlaceholder')})}
                  {remoteBridgeField('Channel ID', 'channel_id', {placeholder: t('remoteBridgeSetup.channelIdPlaceholder')})}
                </>)}
                {remoteBridgeSetupPlatform === 'teams' && (
                  remoteBridgeField('Webhook URL', 'webhook_url', {type: 'password', placeholder: 'https://...logic.azure.com/workflows/...'})
                )}
                {remoteBridgeSetupPlatform === 'line' && (<>
                  <div className="rb-guide-card">
                    <span className="rb-guide-count">{t('remoteBridgeSetup.stepCount', { current: remoteBridgeSetupGuideStep + 1, total: lineSetupGuide.length })}</span>
                    <strong>{lineSetupGuide[remoteBridgeSetupGuideStep].title}</strong>
                    <p>{lineSetupGuide[remoteBridgeSetupGuideStep].body}</p>
                    <div className="rb-guide-actions">
                      <button
                        type="button"
                        onClick={() => {
                          const step = lineSetupGuide[remoteBridgeSetupGuideStep];
                          openExternal(step.url || `https://www.google.com/search?q=${encodeURIComponent(step.query)}`);
                        }}
                      >
                        {lineSetupGuide[remoteBridgeSetupGuideStep].action}
                      </button>
                      {remoteBridgeSetupGuideStep < lineSetupGuide.length - 1 ? (
                        <button type="button" onClick={() => setRemoteBridgeSetupGuideStep((step) => Math.min(step + 1, lineSetupGuide.length - 1))}>{t('remoteBridgeSetup.next')}</button>
                      ) : (
                        <button type="button" onClick={() => setRemoteBridgeSetupGuideStep(0)}>{t('remoteBridgeSetup.restartGuide')}</button>
                      )}
                    </div>
                  </div>
                  {remoteBridgeField('Channel Access Token', 'channel_access_token', {type: 'password'})}
                  {remoteBridgeField(t('remoteBridgeSetup.channelSecretField'), 'channel_secret', {type: 'password', placeholder: t('remoteBridgeSetup.channelSecretPlaceholder')})}
                  {remoteBridgeField('Recipient ID (userId / groupId / roomId)', 'recipient_id', {placeholder: t('remoteBridgeSetup.recipientIdPlaceholder')})}
                </>)}
                {remoteBridgeSetupPlatform === 'qq' && (<>
                  <p className="rb-helper-text">{t('remoteBridgeSetup.qqHint')}</p>
                  {remoteBridgeField('Bot App ID', 'bot_app_id', {placeholder: t('remoteBridgeSetup.qqAppIdPlaceholder')})}
                  {remoteBridgeField('Bot Token', 'bot_token', {type: 'password', placeholder: 'Bot token / secret'})}
                  {remoteBridgeField('Channel ID', 'channel_id', {placeholder: t('remoteBridgeSetup.channelIdQQPlaceholder')})}
                </>)}
                <p className="rb-safe-hint">{t('remoteBridgeSetup.safeHint')}</p>
                <div className="rb-actions">
                  <button type="button" onClick={submitQuickModeRegistration} disabled={remoteBridgeDetecting}>
                    {remoteBridgeDetecting ? t('remoteBridgeSetup.registering') : t('remoteBridgeSetup.confirm')}
                  </button>
                  <button type="button" className="rb-cancel-btn" onClick={closeRemoteBridgeSetup}>{t('remoteBridgeSetup.cancel')}</button>
                </div>
              </div>
            )}
            {remoteBridgeSetupStep === 'developer_form' && (
              <div className="rb-setup-step">
                <h4>{t('remoteBridgeSetup.devModeTitle')}</h4>
                <label>URL<input type="text" value={remoteBridgeSetupFields.url || ''} onChange={(e) => setRemoteBridgeSetupFields(f => ({...f, url: e.target.value}))} placeholder="https://..." /></label>
                <label>Method<input type="text" value={remoteBridgeSetupFields.method || 'POST'} onChange={(e) => setRemoteBridgeSetupFields(f => ({...f, method: e.target.value}))} /></label>
                <label>Headers (JSON)<textarea value={remoteBridgeSetupFields.headers || '{}'} onChange={(e) => setRemoteBridgeSetupFields(f => ({...f, headers: e.target.value}))} rows={3} /></label>
                <label>Body Template<textarea value={remoteBridgeSetupFields.body_template || ''} onChange={(e) => setRemoteBridgeSetupFields(f => ({...f, body_template: e.target.value}))} rows={4} placeholder='{"text":"{{.Content}}"}' /></label>
                <div className="rb-actions">
                  <button type="button" onClick={submitDeveloperModeRegistration} disabled={remoteBridgeDetecting}>
                    {remoteBridgeDetecting ? t('remoteBridgeSetup.registering') : t('remoteBridgeSetup.confirm')}
                  </button>
                  <button type="button" className="rb-cancel-btn" onClick={() => setRemoteBridgeSetupStep('mode_select')}>{t('remoteBridgeSetup.back')}</button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
      {remoteBridgeRenameTarget && (
        <div className="export-dialog-overlay">
          <div className="export-dialog">
            <p>{t('remoteBridgeSetup.renameTitle')}</p>
            <input
              className="remote-bridge-name-input"
              autoFocus
              value={remoteBridgeRenameDraft}
              onChange={(event) => setRemoteBridgeRenameDraft(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') saveRemoteBridgeName();
                if (event.key === 'Escape') setRemoteBridgeRenameTarget(null);
              }}
            />
            <div className="export-dialog-actions">
              <button type="button" onClick={saveRemoteBridgeName}>{t('remoteBridgeSetup.confirmRename')}</button>
              <button type="button" onClick={() => setRemoteBridgeRenameTarget(null)}>{t('remoteBridgeSetup.renameLater')}</button>
            </div>
          </div>
        </div>
      )}
      {activePanel === 'settings' ? (
        <SettingsErrorBoundary onClose={() => setActivePanel(null)}>
          <SettingsWorkspace
            settingsState={settingsState}
            onPersonaAdd={addPersona}
            onPersonaDrop={dropPersonaPackage}
            onPersonaChange={savePersonaPatch}
            onPersonaNativeDrag={startNativePersonaExport}
            onPersonaNativeExportAction={finalizeNativePersonaExport}
            onPersonaReorder={reorderPersonas}
            trustDomClickEnabled={trustDomClickEnabled}
            activeOverrides={activeOverrides}
            trustedSessionActive={trustedSessionActive}
            deviceProfile={deviceProfile}
            onToggleTrustDomClick={toggleTrustDomClick}
            onEnableOverride={enableContextualOverride}
            onEnableWorkflowTrust={enableWorkflowTrust}
            onDisableOverride={disableContextualOverride}
            onEnableTrustedSession={enableTrustedSession}
            onLoadDeviceProfile={loadDeviceProfile}
            browserPref={browserPref}
            panelSettings={settingsState.panel}
            adapterList={adapterList}
            summaryModelSettings={summaryModelSettings}
            summaryModelScan={summaryModelScan}
            onSummaryModelChange={saveSummaryModelPatch}
            onSummaryModelRescan={() => callWails(ScanLocalSummaryModels).then((result) => setSummaryModelScan(result || {options: [], message: ''}))}
            voiceState={voiceState}
            voiceInstallBusy={voiceInstallBusy}
            onVoiceSettingsChange={saveVoiceSettingsPatch}
            onVoiceSettingsRefresh={() => callWails(GetVoiceSettings).then((settings) => setVoiceState(settings || null))}
            onVoiceModelInstall={installVoiceBaseModel}
            onVoiceModelRemove={removeVoiceBaseModel}
            onVoiceDebugClear={clearVoiceDebug}
            onSaveBrowserPref={saveBrowserPreference}
            onPreviewStyleDiff={previewStyleDiff}
            avatarConfigs={avatarConfigs}
            avatarExpression={avatarExpression}
            avatarModeNotice={avatarModeNotice}
            renderedPixelAvatars={renderedPixelAvatars}
            staticAvatarPreviews={staticAvatarPreviews}
            onAvatarProviderSelect={setAvatarProviderMode}
            onAvatarStateSelect={setManualAvatarState}
            onAvatarLoad={loadCurrentAvatar}
          />
        </SettingsErrorBoundary>
      ) : (
        <>
          <main className="workspace">
            <TopConsole
              activePersona={mainPersona}
              activeAvatarConfig={mainAvatarConfig}
              avatarExpression={avatarExpression}
              avatarModeNotice={avatarModeNotice}
              avatarProvider={mainAvatarProvider}
              avatarSrc={mainAvatarSrc}
              browserPref={browserPref}
              greeting={state.greeting}
              haoras={state.haoras}
              subagentTabs={subagentTabs}
              personaJob={mainPersona.identity || personaJob}
              personaName={mainPersona.name || personaName}
              dagRun={dagRun}
              reviewState={reviewState}
              reviewPopup={reviewPopup}
              skillInjections={skillInjections}
              showSkillFirstUseCard={showSkillFirstUseCard}
              snoozeHours={snoozeHours}
              systemStatusHistory={systemStatusHistory}
              onDismissSkillFirstUse={dismissSkillFirstUseCard}
              w3aImportPopup={w3aImportPopup}
              w3aDetail={w3aDetail}
              w3aPollutionResult={w3aPollutionResult}
              w3aTransferGuidance={w3aTransferGuidance}
              w3aTrustList={w3aTrustList}
              w3aActionBusy={w3aActionBusy}
              w3aActionError={w3aActionError}
              w3aStatusConfig={w3aStatusConfig}
              w3aToastMsg={w3aToastMsg}
              onLoadW3AInfo={loadW3AMediaInfo}
              onDetectW3APollution={detectW3APollution}
              onShowW3AGuidance={showW3ATransferGuidance}
              onTrustW3ADeveloper={trustW3ADeveloper}
              onExportW3ACopy={exportW3AWithSidecarCopy}
              onDismissW3AImportPopup={dismissW3AImportPopup}
              onShowW3AToast={(msg) => {
                setW3aToastMsg(msg);
                setTimeout(() => setW3aToastMsg(null), 4000);
              }}
              onDismissW3AToast={() => setW3aToastMsg(null)}
              onAvatarProviderSelect={(provider) => setAvatarProviderMode(provider, mainPersona.id)}
              onAvatarStateSelect={setManualAvatarState}
              onPersonaJobChange={(value) => {
                setPersonaJob(value);
                savePersonaPatch(mainPersona.id, {identity: value});
              }}
              onPersonaNameChange={(value) => {
                setPersonaName(value);
                savePersonaPatch(mainPersona.id, {name: value});
              }}
              onReviewPopupChange={setReviewPopup}
              onRotateGreeting={rotateGreeting}
              onSkillSelect={(skillId) => setReviewState((prev) => ({...prev, lowRiskAmbiguity: {...prev.lowRiskAmbiguity, selectedSkillId: skillId}}))}
              onSnooze={() => setReviewState((prev) => ({...prev, lowRiskAmbiguity: {...prev.lowRiskAmbiguity, dismissedUntil: t('dag.hours', { hours: snoozeHours })}}))}
              onSnoozeHoursChange={setSnoozeHours}
              onAcknowledgeDigestItem={acknowledgePendingItem}
              onConfirmSkillBuild={confirmSkillBuild}
              reviewArchive={reviewArchive}
              activeAdapter={resolveActiveAdapter()}
              adapterOptions={adapterList}
              activeAdapterId={activeAdapterId}
              onAdapterSelect={(name) => {
                activeAdapterIdRef.current = name;
                setActiveAdapterId(name);
              }}
              cliInspectorBusy={cliInspectorBusy}
              cliInspectorLog={cliInspectorLog}
              onCLIInspectSend={sendCLIInspectorText}
              remoteBridgeChannels={remoteBridgeChannels}
              remoteBridgeModePopup={remoteBridgeModePopup}
              remoteBridgeInboundInfo={remoteBridgeInboundInfo}
              remoteBridgeInboundAdapters={remoteBridgeInboundAdapters}
              onToggleRemoteBridge={toggleRemoteBridgeChannel}
              onOpenRemoteBridgeMode={openRemoteBridgeModePopup}
              onSwitchRemoteBridgeMode={switchRemoteBridgeMode}
              onRemoveRemoteBridge={removeRemoteBridgeChannel}
              onCloseRemoteBridgeModePopup={() => setRemoteBridgeModePopup(null)}
              onOpenRemoteBridgeSetup={openRemoteBridgeSetup}
              onRemoteBridgeRename={openRemoteBridgeRename}
              onTemporaryRemoteBridgeTest={sendTemporaryRemoteBridgeTest}
              onSaveRemoteBridgeInboundSecret={saveRemoteBridgeInboundSecret}
              onMakeRemoteBridgePrimary={makeRemoteBridgePrimary}
              subExportCapabilities={subExportCapabilities}
              onRenameHaora={renameSubagent}
              activeHaoraId={activeHaoraId}
              onHaoraSelect={(haora) => {
                setActiveHaoraId(haora?.isMain ? null : (haora?.id || haora?.name || null));
              }}
              onSubagentsChanged={refreshAvailableAdapters}
              onHaorasReordered={(nextHaoras, nextSubIds) => {
                setState((prev) => ({...prev, haoras: nextHaoras}));
                if (Array.isArray(nextSubIds)) {
                  setSubagentTabs((current) => nextSubIds
                    .map((id) => current.find((tab) => (tab.id || tab.name) === id))
                    .filter(Boolean));
                }
                callWails(() => ReorderTabs(JSON.stringify(nextSubIds || nextHaoras.slice(1)))).catch(console.error);
              }}
            />
            <ConversationPanel
              activePersona={activePersona}
              messages={messages}
              personaName={personaName}
              draft={draft}
              onDraftChange={setDraft}
              onSend={sendMessage}
              onDelete={(index) => {
                const target = messages[index];
                setMessages((prev) => prev.filter((_, i) => i !== index));
                if (target) {
                  const convId = activeConversationIdRef.current || 'main';
                  // 內容比對刪除：後端走 pipeline 記 delete_sentences 維持 hash 鏈；找不到則 no-op。
                  callWails(() => DeleteTalkMessageForAgent(convId, target)).catch(() => {});
                }
              }}
              onSummarizeSearch={(text) => submitComposerText(t('system.summarizeSearch', { text }))}
              // SEC-06: 讀取網址按鈕注入文字，複用後端確認流程
              onInjectText={(text) => submitComposerText(text)}
              activeConversationId={activeConversationIdRef.current || 'main'}
              // v3.6.4 Readiness Gate UI Interaction Layer props
              readinessGate={readinessGate}
              longPressProgress={longPressProgress}
              gachaPhase={gachaPhase}
              riskImpactExpanded={riskImpactExpanded}
              onSelectCandidate={handleSelectCandidate}
              onNormalConfirm={handleNormalConfirm}
              onNormalReject={handleNormalReject}
              onHighRiskYes={handleHighRiskYes}
              onLongPressStart={handleLongPressStart}
              onLongPressEnd={handleLongPressEnd}
              dispatchStatus={dispatchStatus}
              voiceState={voiceState}
              voiceRecording={voiceRecording}
              voiceBusy={voiceBusy}
              voiceStatus={voiceStatus}
              voiceError={voiceError}
              onVoicePressStart={startVoiceRecording}
              onVoicePressEnd={stopVoiceRecording}
              onVoiceCancel={cancelVoiceRecording}
              taskActive={isTaskProgressActive(dagRun)}
              onCancelTask={cancelActiveTaskProgress}
              pendingTaskReview={pendingTaskReview}
              taskReviewDetailsOpen={reviewPopup === 'risk'}
              onConfirmTaskReview={confirmSkillBuild}
              onCancelTaskReview={() => cancelActiveTaskProgress('review_cancel')}
              onShowTaskReviewDetails={() => setReviewPopup((current) => current === 'risk' ? null : 'risk')}
            />
          </main>
          <RightRail
          tools={tools}
          toolVisibility={toolVisibility}
          onToolActivate={activateTool}
          isToolPopupOpen={toolPopupsOpen.right}
          referenceFiles={referenceFiles}
          isLearningEnabled={learningEnabled}
          isRecordingEnabled={recordingEnabled}
          learningDigestReady={learningDigestReady}
          sourceTrustHint={sourceTrustHint}
          onLearningToggle={toggleLearning}
          onReferenceFileDrop={(pathsOrFiles) => {
            if (pathsOrFiles?.[0] instanceof File) {
              importReferenceFileObjects(pathsOrFiles);
            } else {
              importReferencePaths(pathsOrFiles);
            }
          }}
          onReferenceLinkOpen={() => setReferenceLinkOpen(true)}
          onReferenceFileDragOut={startNativeReferenceExport}
          onReferenceFileReorder={reorderReferenceFile}
          onReferenceInternalDrag={handleReferenceInternalDrag}
          onReferenceCardDoubleClick={handleReferenceCardDoubleClick}
          onReferenceFailedRemove={handleReferenceFailedRemove}
          onRecordingToggle={() => {
            if (recordingEnabled) {
              stopDraftSandbox('user_stop');
            } else {
              startDraftSandbox();
            }
          }}
          onToolFavorite={addFavoriteTool}
          onToolPopupToggle={() => toggleToolPopup('right')}
          />
        </>
      )}
      {typeof document !== 'undefined' && createPortal(
        <button
          className={`vl-monitor-launcher${vlMonitorOpen ? ' vl-monitor-launcher-active' : ''}`}
          data-vl-target="visual-learning-monitor-launcher"
          onClick={() => setVlMonitorOpen((v) => !v)}
          type="button"
          aria-label={vlMonitorOpen ? '隱藏 Visual Learning 監視' : '開啟 Visual Learning 監視'}
          title={vlMonitorOpen ? '隱藏監視面板' : 'Visual Learning 監視'}
        >
          <span>VL</span>
          {vlPendingCount > 0 && <span className="vl-monitor-launcher-badge">{vlPendingCount}</span>}
        </button>,
        document.body
      )}
      {typeof document !== 'undefined' && vlMonitorOpen && createPortal(
	        <VisualLearningPanel
	          learningActive={vlLearningActive}
	          onLearningToggle={toggleLearning}
	          pendingCount={vlPendingCount}
	          hasBlocking={vlHasBlocking}
	          recentEvents={vlRecentLearningEvents}
	          onClose={() => setVlMonitorOpen(false)}
	        />,
        document.body
      )}
      {/* §M3 Embedding picker modal：first-drop 才開 */}
      {embeddingPickerTarget && typeof document !== 'undefined' && createPortal(
        <EmbeddingPickerModal
          displayName={embeddingPickerTarget.displayName}
          onClose={() => setEmbeddingPickerTarget(null)}
        />,
        document.body
      )}
      {/* §M3+ 雙擊引用文件卡片：顯示目前 embedding 狀態 popup */}
      {refEmbedPopup && typeof document !== 'undefined' && createPortal(
        <div
          className="reference-embed-popup"
          style={{
            left: Math.min(refEmbedPopup.rect.left - 260, (typeof window !== 'undefined' ? window.innerWidth : 9999) - 280),
            top: refEmbedPopup.rect.top,
          }}
        >
          <div className="reference-embed-popup-title">{refEmbedPopup.file.name}</div>
          {refEmbedPopup.config?.providerId && refEmbedPopup.config?.modelId ? (
            <>
              <div className="reference-embed-popup-status reference-embed-on">
                ✓ 語意搜尋已啟用
              </div>
              <div className="reference-embed-popup-detail">
                Provider：{refEmbedPopup.config.providerId}
                <br />
                Model：{refEmbedPopup.config.modelId}
                {refEmbedPopup.config.dimension > 0 && <><br />Dimension：{refEmbedPopup.config.dimension}</>}
              </div>
              <div className="reference-embed-popup-actions">
                <button
                  type="button"
                  className="reference-embed-popup-btn"
                  onClick={() => {
                    setRefEmbedPopup(null);
                    setEmbeddingPickerTarget({displayName: refEmbedPopup.file.name});
                  }}
                >
                  換個 model
                </button>
                <button
                  type="button"
                  className="reference-embed-popup-btn-secondary"
                  onClick={() => setRefEmbedPopup(null)}
                >
                  取消
                </button>
              </div>
            </>
          ) : (
            <>
              <div className="reference-embed-popup-status reference-embed-off">
                ⚠ 未啟用 embedding model（目前走 TF-IDF 基本搜尋）
              </div>
              <div className="reference-embed-popup-detail">
                沒有 embedding 也能搜，但語意搜尋會更準。
                <br />
                要加上 embedding model 嗎？
              </div>
              <div className="reference-embed-popup-actions">
                <button
                  type="button"
                  className="reference-embed-popup-btn"
                  onClick={() => {
                    setRefEmbedPopup(null);
                    setEmbeddingPickerTarget({displayName: refEmbedPopup.file.name});
                  }}
                >
                  加上 model
                </button>
                <button
                  type="button"
                  className="reference-embed-popup-btn-secondary"
                  onClick={() => setRefEmbedPopup(null)}
                >
                  取消
                </button>
              </div>
            </>
          )}
        </div>,
        document.body
      )}
      {avatarUploadTargetId && (
        <AvatarUploadModal
          persona={settingsState.personas.find((persona) => persona.id === avatarUploadTargetId) || mainPersona}
          onClose={() => setAvatarUploadTargetId(null)}
          onApply={async (file) => {
            await saveStaticAvatar(avatarUploadTargetId, file);
            setAvatarUploadTargetId(null);
          }}
        />
      )}

      {/* #I-1001: 自動封存 Toast — 畫面右下角輕量提示 */}
      {autoArchiveToast && (
        <DigestAutoArchiveToast message={autoArchiveToast} onDismiss={() => setAutoArchiveToast(null)} />
      )}

      {/* #I-805: Sidecar 崩潰恢復卡片 — sidecarState 為字串（'running' | 'crashed' | 'start_failed' 等） */}
      {sidecarState === 'crashed' && (
        <StopRecoveryCard error={t('stopRecovery.sidecarAbnormalMsg')} onRestart={handleRestartSidecar} />
      )}
      {sidecarState === 'start_failed' && (
        <StopRecoveryCard error={t('stopRecovery.sidecarStartFail')} onRestart={handleRestartSidecar} />
      )}

      {/* CLI 授權對話框：CLI 需要瀏覽器 OAuth 授權時顯示。
          Go 端已自動開啟瀏覽器，此對話框讓使用者確認授權完成後重試原本的訊息。 */}
      {pendingLearningReplayStartConfirm && (
        <LearningReplayStartConfirmCard
          pending={pendingLearningReplayStartConfirm}
          onConfirm={startConfirmedLearningReplay}
          onCancel={cancelLearningReplayStartConfirm}
        />
      )}

      {pendingLearningReplayConfirm && (
        <LearningReplayConfirmCard
          pending={pendingLearningReplayConfirm}
          onConfirm={continueLearningReplayAfterConfirm}
          onCancel={cancelLearningReplayConfirm}
        />
      )}

      {cliAuthRequest && (
        <div className="dialog-overlay" role="dialog" aria-label={t('auth.cliAuthLabel')}>
          <div className="dialog-card" style={{maxWidth: 420, padding: 24}}>
            <h3 style={{margin: '0 0 12px'}}>{t('auth.authRequired')}</h3>
            <p style={{margin: '0 0 8px'}}>
              {cliAuthRequest.message || t('auth.adapterAuthRequired', { adapterId: cliAuthRequest.adapter_id })}
            </p>
            {/* SEC-05: untrusted 時顯示確認開啟提示 */}
            {!cliAuthRequest.isTrusted && cliAuthRequest.auth_url && (
              <p style={{margin: '0 0 8px', fontSize: 13, color: '#e67e22'}}>
                {t('auth.untrustedOpenHint')} <strong>{cliAuthRequest.auth_hostname || cliAuthRequest.auth_url}</strong>
              </p>
            )}
            <p style={{margin: '0 0 16px', fontSize: 13, opacity: 0.7}}>
              {cliAuthRequest.isTrusted
                ? t('auth.browserOpenedHint')
                : t('auth.clickOpenHint')}
            </p>
            <div style={{display: 'flex', gap: 8, justifyContent: 'flex-end'}}>
              <button
                type="button"
	                onClick={() => {
	                  // 取消授權：關閉對話框，不重試
	                  const conversationId = cliAuthRequest.conversation_id || activeConversationIdRef.current || 'main';
	                  setCliAuthRequest(null);
	                  setConversationMessages(conversationId, (prev) => [...prev, t('auth.cancelledMessage')]);
	                }}
              >
                {t('common.cancel')}
              </button>
              <button
                type="button"
                style={{fontWeight: 600}}
                onClick={async () => {
	                  const req = cliAuthRequest;
	                  const conversationId = req.conversation_id || activeConversationIdRef.current || 'main';
	                  // SEC-05: untrusted 時先用 BrowserOpenURL 開啟授權頁面
	                  if (!req.isTrusted && req.auth_url) {
	                    openExternal(req.auth_url);
	                    setConversationMessages(conversationId, (prev) => [...prev, t('auth.openedMessage')]);
	                    // 標記為已開啟，下次按鈕變為「已完成授權」行為
	                    setCliAuthRequest({...req, isTrusted: true});
	                    return;
	                  }
	                  // trusted（或已開啟過）→ 關閉對話框 → 重試原本的 CLI 訊息
	                  setCliAuthRequest(null);
	                  setConversationMessages(conversationId, (prev) => [...prev, t('auth.retryingMessage')]);
                  try {
                    // DEBUG_TRACE_REMOVE: Preserve auth retry as a distinct trace.
                    const traceId = req.trace_id || makeDebugTraceID('auth-retry');
                    postDebugTrace('ui.authRetry.before.SendCLIMessage', traceId, req);
                    const resp = await callWails(() =>
                      SendCLIMessage(req.adapter_id, req.session_id || '', req.user_text || '', traceId)
                    );
                    const cliResp = normalizeCLIResponse(resp);
	                    postDebugTrace('ui.authRetry.after.SendCLIMessage', traceId, {response: cliResp || null});
	                    if (cliResp?.text && !cliResp?.auth_required) {
	                      setConversationMessages(conversationId, (prev) => [...prev, `Ai:${cliResp.text}`]);
	                      callWails(() => AppendTalkEntryForAgent(conversationId, 'assistant', cliResp.text)).catch(() => {});
	                    } else if (cliResp?.auth_required) {
	                      // 還是需要授權 — 再次彈出對話框
	                      setCliAuthRequest(req);
	                      setConversationMessages(conversationId, (prev) => [...prev, t('auth.notCompletedMessage')]);
	                    } else if (cliResp?.error) {
	                      setConversationMessages(conversationId, (prev) => [...prev, t('auth.cliErrorMessage', { error: cliResp.error })]);
	                    }
	                  } catch (err) {
	                    setConversationMessages(conversationId, (prev) => [...prev, t('auth.retryFailMessage', { error: err?.message || String(err) })]);
	                  }
                }}
              >
                {cliAuthRequest?.isTrusted ? t('auth.completedAuth') : t('auth.openAndAuth')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* I-3: Draft Sandbox stop — three continuation options */}
      {sandboxStopOptions && (
        <DraftSandboxStopDialog
          onPromote={promoteDraft}
          onDismiss={dismissSandboxOptions}
        />
      )}

      {/* I-3: Trusted Session expired notification */}
      {trustedSessionExpired && (
        <TrustedSessionExpiredDialog onChoice={handleTrustedSessionExpired} />
      )}

      {/* I-4: Safari Runtime Notice Modal */}
      {showSafariNotice && safariNotice && (
        <SafariNoticeModal notice={safariNotice} onDismiss={dismissSafariNotice} />
      )}

      {/* I-4: Style Diff Preview Modal */}
      {styleDiffPreview && (
        <StyleDiffPreviewModal
          diffJson={styleDiffPreview.diffJson}
          onConfirm={confirmStyleDiff}
          onCancel={cancelStyleDiff}
        />
      )}

      {/* I-4: Style Diff Error Toast */}
      {styleDiffError && (
        <div className="pkg-error-toast" role="alert">
          <strong>{t('styleDiff.errorLabel')}</strong>
          <span>{styleDiffError}</span>
          <button type="button" onClick={() => setStyleDiffError('')}>{t('common.close')}</button>
        </div>
      )}

      {toolPopupsOpen.left && (
        <ToolPopup
          side="left"
          tools={tools}
          activeTab={activeToolTabs.left}
          favoriteToolIds={favoriteToolIds}
          hiddenToolIds={hiddenToolIds}
          toolResult={toolResult}
          externalServiceLinks={extServiceLinks}
          documentationLinks={extDocLinks}
          dagRun={dagRun}
          onTabChange={(tab) => setActiveToolTabs((current) => ({...current, left: tab}))}
          onToolActivate={activateTool}
          onToolDragStart={setDraggedTool}
          onToolDragOut={setDragActionTool}
          onReorder={reorderTool}
        />
      )}
      {toolPopupsOpen.right && (
        <ToolPopup
          side="right"
          tools={tools}
          activeTab={activeToolTabs.right}
          favoriteToolIds={favoriteToolIds}
          hiddenToolIds={hiddenToolIds}
          toolResult={toolResult}
          externalServiceLinks={extServiceLinks}
          documentationLinks={extDocLinks}
          dagRun={dagRun}
          onTabChange={(tab) => setActiveToolTabs((current) => ({...current, right: tab}))}
          onToolActivate={activateTool}
          onToolDragStart={setDraggedTool}
          onToolDragOut={setDragActionTool}
          onReorder={reorderTool}
        />
      )}
      {/* i18n: tool */}
      {dragActionTool && (
        <DragActionModal
          ariaLabel={t('tool.dragAriaLabel')}
          icon={dragActionTool.tool.icon}
          title={dragActionTool.tool.title}
          detail={dragActionTool.tool.id}
          actions={[
            {
              label: dragActionTool.source === 'right' ? t('tool.removeFavorite') : t('tool.unlink'),
              onClick: () => (
                dragActionTool.source === 'right'
                  ? removeFavoriteOnly(dragActionTool.tool.id)
                  : removeToolCompletely(dragActionTool.tool.id)
              ),
            },
            {
              label: t('tool.copy'),
              disabled: toolTabFor(dragActionTool.tool) === 'external',
              onClick: () => copyTool(dragActionTool),
            },
            {label: t('common.cancel'), onClick: () => setDragActionTool(null)},
          ]}
        />
      )}
      {copyConfirmTool && (
        <ToolCopyConfirmModal
          tool={copyConfirmTool}
          onCancel={() => setCopyConfirmTool(null)}
          onConfirm={confirmCopyTool}
        />
      )}
      {skillExecutionConfirm && (
        <DragActionModal
          ariaLabel="Skill execution confirmation"
          icon="⌁"
          title={skillExecutionConfirm.skillId || 'Skill'}
          detail={skillExecutionConfirm.actionTarget || skillExecutionConfirm.message || ''}
          actions={[
            {label: '允許一次', onClick: () => confirmSkillExecutionChoice('allow_once')},
            {label: '總是允許', onClick: () => confirmSkillExecutionChoice('always')},
            {label: t('common.cancel'), onClick: () => confirmSkillExecutionChoice('cancel')},
          ]}
        />
      )}
      {referenceExportDialog && (
        <DragActionModal
          ariaLabel="引用文件拖曳操作"
          icon="▤"
          title={referenceExportDialog.name}
          detail={referenceExportDialog.landedDetail || referenceExportDialog.landedPath}
          actions={[
            {label: t('adapter.remove'), onClick: () => handleReferenceExportAction('remove')},
            {label: t('adapter.copyAction'), onClick: () => handleReferenceExportAction('copy')},
            {label: t('common.cancel'), onClick: () => handleReferenceExportAction('cancel')},
          ]}
        />
      )}
      {referenceLinkOpen && (
        <ReferenceLinkModal
          value={referenceLinkValue}
          linkPreview={linkPreview}
          error={linkPreviewError}
          suggestions={linkPreviewSuggestions}
          onCancel={() => { setReferenceLinkOpen(false); setLinkPreview(null); setLinkPreviewError(''); setLinkPreviewSuggestions([]); }}
          onChange={(v) => { setReferenceLinkValue(v); setLinkPreview(null); setLinkPreviewError(''); setLinkPreviewSuggestions([]); }}
          onUseSuggestion={(path) => { setReferenceLinkValue(path); setLinkPreview(null); setLinkPreviewError(''); setLinkPreviewSuggestions([]); }}
          onPreview={previewReferenceLink}
          onConfirm={confirmReferenceLink}
        />
      )}
      {llmAPISetup && (
        <LLMAPISetupModal
          setup={llmAPISetup}
          guideStep={llmAPISetupGuideStep}
          onGuideStepChange={setLlmAPISetupGuideStep}
          onChange={setLlmAPISetup}
          onCancel={() => setLlmAPISetup(null)}
          onTest={previewLLMAPIConnection}
          onSubmit={submitLLMAPISetup}
        />
      )}
      {webSearchSetup && (
        <WebSearchSetupModal
          setup={webSearchSetup}
          config={webSearchConfig}
          error={webSearchSetupError}
          onChange={setWebSearchSetup}
          onCancel={() => setWebSearchSetup(null)}
          onClear={clearWebSearchSetup}
          onSubmit={submitWebSearchSetup}
        />
      )}
      {adapterRenameTarget && (
        <AdapterRenameModal
          target={adapterRenameTarget}
          draft={adapterRenameDraft}
          onDraftChange={setAdapterRenameDraft}
          onCancel={() => setAdapterRenameTarget(null)}
          onSave={saveAdapterRename}
        />
      )}
      {pendingImport && (
        <PackageConfirmModal
          pending={pendingImport}
          packageData={pendingPackageData}
          error={packageInstallError}
          onConfirm={confirmPackageInstall}
          onReject={rejectPackageInstall}
        />
      )}
      {packageInstallError && !pendingImport && (
        <PackageErrorToast message={packageInstallError} onClose={() => setPackageInstallError('')}/>
      )}

      {/* §30: 關閉視窗 → 存成 sub 對話框 */}
      {sessionClosePrompt && (
        <SessionCloseDialog
          analysis={sessionClosePrompt}
          onCancel={() => setSessionClosePrompt(null)}
          onSave={async (name) => {
            try {
              await callWails(() => ConfirmClose(true, name));
              setSessionClosePrompt(null);
            } catch (error) {
              setToolResult({toolId: 'session-close', ok: false, message: t('subagent.sessionCloseHandlerFail', { error: error?.message || error })});
            }
          }}
          onDiscard={async () => {
            try {
              await callWails(() => ConfirmClose(false, ''));
              setSessionClosePrompt(null);
            } catch (error) {
              setToolResult({toolId: 'session-close', ok: false, message: t('subagent.closeFail', { error: error?.message || error })});
            }
          }}
        />
      )}

      {/* I-6 (#I-602): Reauth Intercept Dialog — unavailable 工具執行前攔截 */}
      {reauthTool && (
        <ReauthInterceptDialog
          tool={reauthTool}
          onRetry={() => forceActivateTool(reauthTool.id)}
          onCancel={() => setReauthTool(null)}
        />
      )}
    </div>
  );
}

function findActivePersona(settingsState) {
  return settingsState.personas.find((persona) => persona.id === settingsState.activePersonaId) || settingsState.personas[0];
}

function callWails(fn) {
  try {
    return Promise.resolve(fn());
  } catch (error) {
    return Promise.reject(error);
  }
}

function isLearningReplayRequest(text) {
  const normalized = String(text || '').trim().toLowerCase();
  return /按照.*剛剛.*示範/.test(normalized)
    || /照.*剛剛.*示範/.test(normalized)
    || /回放.*剛剛.*示範/.test(normalized)
    || /回放.*剛剛.*步驟/.test(normalized)
    || /回放.*剛剛.*操作/.test(normalized)
    || /重播.*剛剛.*示範/.test(normalized)
    || /重播.*剛剛.*步驟/.test(normalized)
    || /重播.*剛剛.*操作/.test(normalized)
    || /再.*示範.*剛剛/.test(normalized)
    || /再.*執行.*剛剛/.test(normalized)
    || /再.*跑.*剛剛/.test(normalized)
    || /剛剛.*步驟/.test(normalized)
    || /replay.*last.*demo/.test(normalized)
    || /replay.*previous.*demo/.test(normalized)
    || /replay.*demo/.test(normalized)
    || /replay.*steps/.test(normalized)
    || /follow.*demo/.test(normalized);
}

function normalizeLearningOperationQuery(text) {
  const raw = String(text || '').trim();
  if (!raw) return '';
  return raw
    .replace(/操作|操做|operation/gi, ' ')
    .replace(/相關|關於|有關|已保存|已儲存|保存|儲存|錄影紀錄|錄製紀錄|示範紀錄|示範|流程|畫面|tag/gi, ' ')
    .replace(/幫我|請|查詢|搜尋|查找|尋找|找|列出|查看|看看|知道|執行|回放|重播|開啟|打開|有哪些|什麼|甚麼|樣|的/g, ' ')
    .replace(/[，。！？、,.!?;:()[\]{}"'`<>|\\/]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function parseLearningOperationCatalogRequest(text) {
  const raw = String(text || '').trim();
  if (!raw) return null;
  const lower = raw.toLowerCase();
  const mentionsSavedCatalog = /已保存|已儲存|保存|儲存|錄影|錄製|示範|紀錄|記錄|catalog/i.test(raw);
  const asksSavedTag = /tag/i.test(raw) && /儲存|保存|已保存|已儲存|畫面|錄影|錄製|示範|紀錄|記錄|操作|操做/.test(raw);
  const asksCatalogList = /有哪些|列出|清單|查看|看看|知道/.test(raw)
    && (mentionsSavedCatalog || /tag/i.test(raw))
    && /操作|操做|錄影|錄製|示範|tag|catalog/i.test(raw);
  // Only explicit catalog/list/tag questions are handled locally. Natural
  // operation requests must go to the LLM so it can choose an action-chain.
  if (!asksSavedTag && !asksCatalogList) return null;
  const catalogQuery = normalizeLearningOperationQuery(raw);
  return catalogQuery ? {mode: 'search', query: catalogQuery} : {mode: 'list', query: ''};
}

function operationIntentFromCLIResponse(resp) {
  const action = String(resp?.action || '').trim().toLowerCase();
  const target = String(resp?.target || '').trim();
  const next = String(resp?.next || '').trim().toLowerCase();
  if (!target) return null;
  // Internal CLI action-chain contract. User input stays natural language.
  if ((action === '查詢' || action === '搜尋' || action === 'search' || action === 'query') && (next === '操作' || next === '操做')) {
    return {mode: 'query', query: normalizeLearningOperationQuery(target) || target};
  }
  if (action === '操作' || action === '操做') {
    return {mode: 'execute', query: normalizeLearningOperationQuery(target) || target};
  }
  return null;
}

function resolveLearningOperationMatch(matches) {
  if (!matches.length) return null;
  const [first, second] = matches;
  const firstScore = Number(first?.score || 0);
  const secondScore = Number(second?.score || 0);
  if (firstScore < 1.5) return null;
  if (second && secondScore > 0 && (firstScore - secondScore) < 0.75) return null;
  return first;
}

function defaultWebSearchProviderOptions() {
  return [
    {
      id: 'tavily',
      name: 'Tavily',
      fields: ['api_key'],
      docs_url: 'https://docs.tavily.com/documentation/api-reference/endpoint/search',
      free_tier_hint: 'AI-agent friendly search; free credits are available from Tavily.',
    },
    {
      id: 'google_cse',
      name: 'Google Custom Search JSON API',
      fields: ['api_key', 'cx'],
      docs_url: 'https://developers.google.com/custom-search/v1/reference/rest/v1/cse/list',
      free_tier_hint: 'Requires an API key and Programmable Search Engine ID.',
    },
    {
      id: 'brave',
      name: 'Brave Search API',
      fields: ['api_key'],
      docs_url: 'https://api-dashboard.search.brave.com/documentation/guides/authentication',
      free_tier_hint: 'Requires a Brave Search API subscription token.',
    },
  ];
}

function formatLearningOperationSearchResults(query, matches, forExecution = false) {
  if (!matches.length) {
    return `Ai:我找不到符合「${query}」的已保存操作。可以重新示範一次，或換更明確的關鍵詞。`;
  }
  const lines = matches.slice(0, 6).map((item, index) => {
    const risk = item?.risk?.level ? `風險：${item.risk.level}` : '';
    const keywords = Array.isArray(item?.keywords) && item.keywords.length
      ? `關鍵詞：${item.keywords.slice(0, 6).join('、')}`
      : '';
    const opTag = item?.operation_tag ? `操作分類：${item.operation_tag}` : '';
    const score = Number.isFinite(Number(item?.score)) ? `匹配：${Number(item.score).toFixed(2)}` : '';
    const meta = [opTag, keywords, risk, score].filter(Boolean).join('；');
    return `${index + 1}. ${item.title || item.operation_tag || item.tag || item.run_id || '未命名操作'}\n   ${item.summary || '沒有摘要。'}${meta ? `\n   ${meta}` : ''}`;
  });
  const prefix = forExecution
    ? `Ai:「${query}」有多個或不夠明確的操作候選，請換更精準的關鍵詞。`
    : `Ai:我找到這些「${query}」相關操作：`;
  return [prefix, ...lines, '請用更明確的自然語言指定其中一個操作。'].join('\n');
}

function formatLearningOperationCatalog(items) {
  if (!items.length) {
    return 'Ai:目前還沒有已保存操作。請先開始示範並停止示範，系統就會產生操作名稱、摘要和關鍵詞。';
  }
  const lines = items.slice(0, 10).map((item, index) => {
    const risk = item?.risk?.level ? `風險：${item.risk.level}` : '';
    const keywords = Array.isArray(item?.keywords) && item.keywords.length
      ? `關鍵詞：${item.keywords.slice(0, 6).join('、')}`
      : '';
    const opTag = item?.operation_tag ? `操作分類：${item.operation_tag}` : '';
    const internal = item?.tag || item?.run_id ? `內部：${item.tag || item.run_id}` : '';
    const meta = [opTag, keywords, risk, `步驟：${item.step_count || 0}`, internal].filter(Boolean).join('；');
    return `${index + 1}. ${item.title || item.operation_tag || '未命名操作'}\n   ${item.summary || '沒有摘要。'}\n   ${meta}`;
  });
  return ['Ai:目前已保存的操作如下：', ...lines].join('\n');
}

function formatLearningOperationLearned(run) {
  const title = run?.title || run?.name || run?.operation_tag || '未命名操作';
  const summary = run?.summary || '已保存這次示範。';
  const keywords = Array.isArray(run?.keywords) && run.keywords.length
    ? `\n關鍵詞：${run.keywords.slice(0, 8).join('、')}`
    : '';
  const opTag = run?.operation_tag ? `\n操作分類：${run.operation_tag}` : '';
  const risk = run?.risk?.level ? `\n風險：${run.risk.level}` : '';
  return `Ai:已保存操作：「${title}」。\n${summary}${opTag}${keywords}${risk}\n之後可以用自然語言請我查找或執行這類操作。`;
}

function isLearningReplayRelatedText(text) {
  const normalized = String(text || '').trim().toLowerCase();
  if (!normalized) return false;
  return /示範|回放|重播|剛剛.*步驟|剛剛.*操作|照.*剛剛|按照.*剛剛/.test(normalized)
    || /replay|demo|last\s+steps|previous\s+steps/.test(normalized);
}

function extractVisualReplayDirective(text) {
  const raw = String(text || '');
  const taggedMatch = raw.match(visualReplayTaggedDirectivePattern);
  if (taggedMatch) {
    return {
      shouldReplay: true,
      tag: taggedMatch[1],
      text: raw.replace(visualReplayTaggedDirectivePattern, '').trim(),
    };
  }
  const matchedDirective = raw.includes(visualReplayLastDemoDirective)
    ? visualReplayLastDemoDirective
    : raw.includes(legacyVisualReplayLastDemoDirective)
      ? legacyVisualReplayLastDemoDirective
      : '';
  if (!matchedDirective) {
    return {shouldReplay: false, tag: '', text: raw};
  }
  return {
    shouldReplay: true,
    tag: '',
    text: raw.split(matchedDirective).join('').trim(),
  };
}

function formatLearningReplayPlan(plan) {
  const steps = Array.isArray(plan?.steps) ? plan.steps : [];
  if (!steps.length) {
    return 'Ai:我還沒有讀到可重放的示範步驟。請先按「開始示範」，點一次目標，然後按「停止示範」。';
  }
  const lines = steps.map((step, index) => {
    const target = step.window_title || step.label || step.role || step.css_selector || step.tag || `座標 (${step.x}, ${step.y})`;
    const selector = step.css_selector ? `，selector: ${step.css_selector}` : '';
    const anchor = step.windows_anchor?.ok ? `，anchor: ${step.windows_anchor.mode || 'available'}` : '，anchor: none';
    const nativeInfo = isNativeReplayStep(step)
      ? `，視窗: ${step.window_title || 'unknown'}，process: ${basenameForDisplay(step.window_process || step.tag || '')}`
      : '';
    const coordLabel = isNativeReplayStep(step) ? '螢幕座標' : '座標';
    return `${index + 1}. ${step.action || 'click'} ${target}，${coordLabel} (${step.x}, ${step.y})${nativeInfo}${selector}${anchor}`;
  });
  const anchorCount = steps.filter((step) => step.windows_anchor?.ok).length;
  return [
    'Ai:我讀到剛剛的示範了，已產生安全 replay plan。確認後會先嘗試操作本視窗內的元素。',
    `Tag: ${plan?.tag || 'demo-last'}，Title: ${plan?.title || plan?.run_name || 'untitled'}。`,
    plan?.run_summary ? `Summary: ${plan.run_summary}` : '',
    `Run: ${plan?.run_id || 'unknown'}，步驟 ${steps.length} 個，visual anchor 覆蓋 ${anchorCount}/${steps.length}。`,
    ...lines,
    '執行時本視窗會先走 selector，外部瀏覽器會走 native 螢幕座標；找不到本視窗元素才 fallback 到示範座標。',
  ].filter(Boolean).join('\n');
}

function formatLearningReplayConfirmation(plan) {
  const steps = Array.isArray(plan?.steps) ? plan.steps : [];
  const nativeSteps = steps.filter(isNativeReplayStep);
  const domSteps = steps.length - nativeSteps.length;
  const windows = Array.from(new Map(nativeSteps.map((step) => {
    const label = `${basenameForDisplay(step.window_process || step.tag || 'unknown')} - ${step.window_title || 'unknown window'}`;
    return [label, label];
  })).values()).slice(0, 4);
  const lines = [
    `即將依照剛剛的示範執行 ${steps.length} 個步驟。`,
    `本視窗 selector ${domSteps} 步，外部 native click ${nativeSteps.length} 步。`,
  ];
  if (windows.length) {
    lines.push('', '外部目標視窗：', ...windows.map((item) => `- ${item}`));
  }
  lines.push('', '外部 native click 會慢速移動鼠標到目標並短暫停頓；錯誤率會在執行後統整呈現，失敗步驟會跳過並繼續後面。', '確定要執行嗎？');
  return lines.join('\n');
}

function formatLearningReplayExecutionResult(result) {
  const steps = Array.isArray(result?.steps) ? result.steps : [];
  const okCount = steps.filter((step) => step.ok).length;
  const skipped = steps.filter((step) => step.skipped);
  const failed = steps.filter((step) => !step.ok && !step.skipped);
  const selectorCount = steps.filter((step) => step.ok && step.method === 'selector').length;
  const coordinateCount = steps.filter((step) => step.ok && step.method === 'coordinate').length;
  const nativeTotal = steps.filter((step) => step.method === 'native').length;
  const nativeOK = steps.filter((step) => step.ok && step.method === 'native').length;
  const nativeForegroundOK = steps.filter((step) => step.method === 'native' && step.foreground_ok).length;
  const warned = steps.filter((step) => step.warning);
  const isVisualRelocation = (step) => (
    step.method === 'visual_relocation'
    || String(step.method || '').includes('relocation')
    || Boolean(step.relocation_method)
  );
  const visualTotal = steps.filter(isVisualRelocation).length;
  const visualOK = steps.filter((step) => step.ok && isVisualRelocation(step)).length;
  const visualConfirm = steps.filter((step) => step.needs_confirmation).length;
  const anchorTotal = steps.filter((step) => step.windows_anchor?.ok).length;
  const reviewAnchors = steps.filter((step) => step.windows_anchor?.needs_review).length;
  const lines = [
    `Ai:Replay executor 執行完成：${okCount}/${steps.length} 步成功。`,
    `selector 命中 ${selectorCount} 步，native 座標 ${nativeOK}/${nativeTotal} 步${nativeTotal ? `（前景確認 ${nativeForegroundOK}/${nativeTotal}）` : ''}，座標 fallback ${coordinateCount} 步。`,
    `visual anchor 覆蓋 ${anchorTotal}/${steps.length} 步，需複核 ${reviewAnchors} 步；YOLO/OpenCV 重定位 ${visualOK}/${visualTotal} 步${visualConfirm ? `，待確認 ${visualConfirm} 步` : ''}。`,
    `略過 ${skipped.length} 步，失敗 ${failed.length} 步，警告 ${warned.length} 步。`,
  ];
  const details = steps.filter((step) => step.warning || step.skipped || (!step.ok && !step.skipped) || isVisualRelocation(step)).map((step) => {
    const target = step.label || step.selector || `座標 (${step.x}, ${step.y})`;
    const status = !step.ok && !step.skipped ? 'FAIL' : step.skipped ? 'SKIP' : 'WARN';
    const error = step.error ? `，${step.error}` : step.warning ? `，${step.warning}` : '';
    const relocation = step.relocation_method
      ? `，relocation=${step.relocation_method} ${Number(step.relocation_confidence || 0).toFixed(2)}${step.relocation_reason ? `，${step.relocation_reason}` : ''}`
      : '';
    const debug = step.debug_image_path ? `，debug=${step.debug_image_path}${step.debug_info_path ? `，info=${step.debug_info_path}` : ''}` : '';
    return `${step.index || '?'}. ${status} ${target}${error}${relocation}${debug}`;
  });
  return [...lines, ...details].join('\n');
}

async function executeLearningReplayPlan(plan, options = {}) {
  const steps = Array.isArray(plan?.steps) ? plan.steps : [];
  const startOffset = Math.max(0, Number(options.startOffset || 0));
  const results = Array.isArray(options.initialResults) ? [...options.initialResults] : [];
  let pendingConfirmation = false;
  for (const step of steps.slice(startOffset)) {
    await delayLearningReplay(learningReplayStepDelayMs);
    const result = await executeLearningReplayStep(step);
    results.push(result);
    if (result?.needs_confirmation) {
      pendingConfirmation = true;
      break;
    }
  }
  return {
    run_id: plan?.run_id || '',
    step_count: steps.length,
    steps: results,
    pending_confirmation: pendingConfirmation,
  };
}

async function executeLearningReplayStep(step) {
  const selector = String(step?.css_selector || '').trim();
  const action = String(step?.action || 'click').toLowerCase();
  const label = step?.label || step?.role || step?.tag || '';
  if (isNativeReplayStep(step)) {
    return executeNativeLearningReplayStep(step);
  }
  if (isLearningReplayBlockedStep(step)) {
    return {
      ok: false,
      skipped: true,
      method: 'blocked',
      index: step?.index,
      selector,
      label,
      x: step?.x,
      y: step?.y,
      windows_anchor: step?.windows_anchor,
      error: '系統控制項不重播，避免重新開始錄製',
    };
  }
  if (action !== 'click') {
    return {
      ok: false,
      method: 'unsupported',
      index: step?.index,
      selector,
      label,
      x: step?.x,
      y: step?.y,
      windows_anchor: step?.windows_anchor,
      error: `尚未支援 ${action}`,
    };
  }

  const bySelector = findReplayElementBySelector(selector);
  if (bySelector) {
    clickReplayElement(bySelector.element, bySelector.x, bySelector.y);
    return {
      ok: true,
      method: 'selector',
      index: step?.index,
      selector,
      label,
      x: Math.round(bySelector.x),
      y: Math.round(bySelector.y),
      windows_anchor: step?.windows_anchor,
    };
  }

  const point = normalizeReplayPoint(step);
  const byPoint = document.elementFromPoint(point.x, point.y);
  if (byPoint) {
    const interactive = byPoint.closest?.(visualLearningInteractiveSelector) || byPoint;
    if (isLearningReplayBlockedElement(interactive)) {
      return {
        ok: false,
        skipped: true,
        method: 'blocked',
        index: step?.index,
        selector,
        label,
        x: point.x,
        y: point.y,
        windows_anchor: step?.windows_anchor,
        error: '座標落在系統控制項，已略過',
      };
    }
    clickReplayElement(interactive, point.x, point.y);
    return {
      ok: true,
      method: 'coordinate',
      index: step?.index,
      selector,
      label: label || compactLearningText(interactive?.innerText || interactive?.textContent || interactive?.getAttribute?.('aria-label') || ''),
      x: point.x,
      y: point.y,
      windows_anchor: step?.windows_anchor,
    };
  }

  return {
    ok: false,
    method: 'coordinate',
    index: step?.index,
    selector,
    label,
    x: point.x,
    y: point.y,
    windows_anchor: step?.windows_anchor,
    error: '找不到可點擊元素',
  };
}

function isLearningReplayBlockedStep(step) {
  const selector = String(step?.css_selector || '').toLowerCase();
  const label = String(step?.label || step?.summary || '').trim();
  return selector.includes('rail-mode-record')
    || selector.includes('sandbox-stop-overlay')
    || /示範螢幕操作|開始示範|停止示範|demo|record/i.test(label);
}

function isLearningReplayBlockedElement(element) {
  return Boolean(element?.closest?.(learningReplayBlockedSelector));
}

function isNativeReplayStep(step) {
  return step?.source === 'native' || step?.coordinate_space === 'screen';
}

async function executeNativeLearningReplayStep(step) {
  try {
    const result = await callWails(() => ExecuteNativeLearningReplayStep(JSON.stringify(step)));
    return {
      ok: Boolean(result?.ok),
      skipped: Boolean(result?.skipped),
      needs_confirmation: Boolean(result?.needs_confirmation),
      method: result?.method || 'native',
      index: result?.index || step?.index,
      label: result?.label || step?.label || step?.window_title || '',
      selector: '',
      x: Number.isFinite(Number(result?.x)) ? Number(result.x) : step?.x,
      y: Number.isFinite(Number(result?.y)) ? Number(result.y) : step?.y,
      error: result?.error || '',
      warning: result?.warning || '',
      window_title: result?.window_title || step?.window_title || '',
      window_process: result?.window_process || step?.window_process || step?.tag || '',
      foreground_ok: Boolean(result?.foreground_ok),
      foreground_title: result?.foreground_title || '',
      foreground_process: result?.foreground_process || '',
      relocated: Boolean(result?.relocated),
      relocation_method: result?.relocation_method || '',
      relocation_confidence: Number.isFinite(Number(result?.relocation_confidence)) ? Number(result.relocation_confidence) : 0,
      relocation_reason: result?.relocation_reason || '',
      debug_image_path: result?.debug_image_path || '',
      debug_info_path: result?.debug_info_path || '',
      original_x: Number.isFinite(Number(result?.original_x)) ? Number(result.original_x) : 0,
      original_y: Number.isFinite(Number(result?.original_y)) ? Number(result.original_y) : 0,
      windows_anchor: step?.windows_anchor,
    };
  } catch (error) {
    return {
      ok: false,
      method: 'native',
      index: step?.index,
      label: step?.label || step?.window_title || '',
      x: step?.x,
      y: step?.y,
      error: error?.message || String(error),
      window_title: step?.window_title || '',
      window_process: step?.window_process || step?.tag || '',
      windows_anchor: step?.windows_anchor,
    };
  }
}

function basenameForDisplay(value) {
  const text = String(value || '').trim();
  if (!text) return '';
  const parts = text.split(/[\\/]/).filter(Boolean);
  return parts[parts.length - 1] || text;
}

function findReplayElementBySelector(selector) {
  if (!selector) return null;
  try {
    const element = document.querySelector(selector);
    if (!element) return null;
    if (isLearningReplayBlockedElement(element)) return null;
    const rect = element.getBoundingClientRect();
    const x = Math.round(rect.left + rect.width / 2);
    const y = Math.round(rect.top + rect.height / 2);
    if (!isFinite(x) || !isFinite(y)) return null;
    return {element, x, y};
  } catch {
    return null;
  }
}

function normalizeReplayPoint(step) {
  const sourceWidth = Number(step?.viewport?.width || window.innerWidth || 1);
  const sourceHeight = Number(step?.viewport?.height || window.innerHeight || 1);
  const currentWidth = window.innerWidth || sourceWidth;
  const currentHeight = window.innerHeight || sourceHeight;
  const rawX = Number(step?.x || 0);
  const rawY = Number(step?.y || 0);
  const x = sourceWidth > 0 ? Math.round((rawX / sourceWidth) * currentWidth) : Math.round(rawX);
  const y = sourceHeight > 0 ? Math.round((rawY / sourceHeight) * currentHeight) : Math.round(rawY);
  return {
    x: clampReplayCoordinate(x, 0, Math.max(0, currentWidth - 1)),
    y: clampReplayCoordinate(y, 0, Math.max(0, currentHeight - 1)),
  };
}

function clampReplayCoordinate(value, min, max) {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, value));
}

function clickReplayElement(element, x, y) {
  if (!element) return;
  const eventInit = {
    bubbles: true,
    cancelable: true,
    view: window,
    clientX: x,
    clientY: y,
    button: 0,
    buttons: 1,
  };
  if (typeof PointerEvent === 'function') {
    element.dispatchEvent(new PointerEvent('pointerdown', eventInit));
    element.dispatchEvent(new PointerEvent('pointerup', {...eventInit, buttons: 0}));
  }
  element.dispatchEvent(new MouseEvent('mousedown', eventInit));
  element.dispatchEvent(new MouseEvent('mouseup', {...eventInit, buttons: 0}));
  element.dispatchEvent(new MouseEvent('click', {...eventInit, buttons: 0}));
}

function delayLearningReplay(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

function appendUniqueReferenceFile(current, file) {
  if (!file) return current;
  if (isInvalidReferencePlaceholder(file)) return current;
  const nextPath = String(file.path || file.name || '');
  const nextName = String(file.name || nextPath);
  let found = false;
  const merged = current.map((item) => {
    const itemPath = String(item.path || item.name || '');
    const itemName = String(item.name || itemPath);
    const isMatch = itemPath === nextPath || (itemName === nextName && nextName);
    if (!isMatch) return item;
    found = true;
    return {...item, ...file};
  });
  return found ? merged : [...current, file];
}

function referenceFileKey(file) {
  return String(file?.path || file?.name || '');
}

function normalizeReferenceImportPaths(paths = []) {
  return Array.from(paths || [])
    .map((path) => (typeof path === 'string' ? path : path?.path))
    .map((path) => String(path || '').trim())
    .filter(isNativeFilePath);
}

function isNativeFilePath(path) {
  return path.startsWith('/') || /^[A-Za-z]:[\\/]/.test(path) || /^\\\\[^\\]+\\[^\\]+/.test(path);
}

function looksLikeReferenceLocalPath(path) {
  const value = String(path || '').trim();
  return value.startsWith('/') ||
    value.startsWith('~/') ||
    value.startsWith('$HOME/') ||
    value.startsWith('./') ||
    value.startsWith('../') ||
    /^[A-Za-z]:[\\/]/.test(value) ||
    /^\\\\[^\\]+\\[^\\]+/.test(value);
}

function isInvalidReferencePlaceholder(file) {
  const path = String(file?.path || '').trim();
  const name = String(file?.name || '').trim();
  return (
    file?.status === 'error'
    && !path
    && (!name || name === _t('system.unnamedFile') || name === _t('rightRail.unnamedFile'))
  );
}

function reorderReferenceFiles(current, draggedKey, targetKey, placement = 'before') {
  if (!draggedKey || !targetKey || draggedKey === targetKey) return current;
  const fromIndex = current.findIndex((file) => referenceFileKey(file) === draggedKey);
  const targetIndex = current.findIndex((file) => referenceFileKey(file) === targetKey);
  if (fromIndex < 0 || targetIndex < 0) return current;

  const next = [...current];
  const [moved] = next.splice(fromIndex, 1);
  const adjustedTargetIndex = next.findIndex((file) => referenceFileKey(file) === targetKey);
  if (adjustedTargetIndex < 0) return current;
  const insertIndex = placement === 'after' ? adjustedTargetIndex + 1 : adjustedTargetIndex;
  next.splice(insertIndex, 0, moved);
  return next;
}

function mergeReferenceLibraryFiles(current, libraryFiles) {
  const libraryPaths = new Set((libraryFiles || []).map((file) => String(file.path || file.name || '')));
  const nonStaleFiles = (current || []).filter((file) => {
    if (file?.source !== 'library') return true;
    return libraryPaths.has(String(file.path || file.name || ''));
  });
  return (libraryFiles || []).reduce(appendUniqueReferenceFile, nonStaleFiles);
}

function updateReferenceFileStatus(current, path, patch) {
  const targetPath = String(path || '');
  return current.map((item) => {
    const itemPath = String(item.path || item.name || '');
    return itemPath === targetPath ? {...item, ...patch} : item;
  });
}

function referenceFileStatusLabel(status) {
  return {
    importing: '處理中',
    checking: '檢查中',
    ready: '已載入',
    error: '失敗',
  }[status] || '已加入';
}

function shouldShowReferenceFileDetail(file) {
  if (!file?.detail) return false;
  return !(file.source === 'library' && file.status === 'ready');
}

function avatarStateLabel(state) {
  return getAvatarStateLabels()[state] || state;
}

function getPixelAvatarDataUrl(pack, state, size = pixelAvatarRenderSize) {
  const cacheKey = `${pack}:${state}:${size}`;
  const cached = pixelAvatarRenderCache.get(cacheKey);
  if (typeof cached === 'string') return Promise.resolve(cached);
  if (cached) return cached;

  const pending = callWails(() => RenderPixelAvatarPreview(pack, state, size))
    .then((bytes) => {
      const dataUrl = bytes?.length ? bytesToDataUrl(bytes) : '';
      if (dataUrl) pixelAvatarRenderCache.set(cacheKey, dataUrl);
      else pixelAvatarRenderCache.delete(cacheKey);
      return dataUrl;
    })
    .catch((error) => {
      pixelAvatarRenderCache.delete(cacheKey);
      throw error;
    });
  pixelAvatarRenderCache.set(cacheKey, pending);
  return pending;
}

let monitorLinkCache = null;
let monitorLinkPending = null;

// DEBUG_TRACE_REMOVE: Temporary browser -> local trace viewer bridge.
// Keep while cleaning dead code: this feeds the local debug trace page that
// records UI -> Wails -> Go -> sidecar -> CLI events.
// Uses the monitor-link register because the trace port may move between runs.
function postDebugTrace(node, traceId, data) {
  resolveMonitorTraceURL()
    .then((url) => {
      const endpoint = `${String(url || '').replace(/\/$/, '')}/trace`;
      return fetch(endpoint, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({node, trace_id: traceId, data}),
      });
    })
    .catch(() => {});
}

function resolveMonitorTraceURL() {
  if (monitorLinkCache?.url) {
    return Promise.resolve(monitorLinkCache.url);
  }
  if (!monitorLinkPending) {
    monitorLinkPending = Promise.resolve()
      .then(() => GetMonitorLinks?.())
      .then((link) => {
        monitorLinkCache = link || {url: 'http://127.0.0.1:48765'};
        return monitorLinkCache.url;
      })
      .catch(() => {
        monitorLinkCache = {url: 'http://127.0.0.1:48765'};
        return monitorLinkCache.url;
      })
      .finally(() => {
        monitorLinkPending = null;
      });
  }
  return monitorLinkPending;
}

// DEBUG_TRACE_REMOVE: Debug-only trace correlation ID generator.
// Keep with postDebugTrace(): trace IDs are what join UI, Go, and sidecar events.
function makeDebugTraceID(scope) {
  return `${scope}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}


/* i18n: map backend-persisted Chinese labels → current locale display labels.
   The Go backend stores display strings as-is (always Chinese on first save).
   After a locale switch we must translate them so dropdowns & style selectors match. */
function localizeBackendLabel(value, translationKeyMap) {
  if (!value) return value;
  // If value already matches a current-locale label, keep it
  const currentLabels = Object.values(translationKeyMap);
  if (currentLabels.includes(value)) return value;
  // Otherwise look up Chinese → current-locale mapping
  for (const [zhLabel, tKey] of Object.entries(translationKeyMap)) {
    if (value === zhLabel) return _t(tKey);
  }
  return value; // unknown value, pass through
}

// Mapping tables: Chinese label → translation key
const _panelLangLabelMap = {
  '繁中': 'settings.langZhTW',
  '英文': 'settings.langEn',
  '日文': 'settings.langJa',
  'Traditional Chinese': 'settings.langZhTW',
  'English': 'settings.langEn',
  'Japanese': 'settings.langJa',
  '中': 'settings.langZhTW',
  'en': 'settings.langEn',
  'ja': 'settings.langJa',
  'pt': 'settings.langPt',
  'pt-PT': 'settings.langPt',
  'es': 'settings.langEs',
  'th': 'settings.langTh',
  'Português': 'settings.langPt',
  'Español': 'settings.langEs',
  'ไทย': 'settings.langTh',
};
const _roleLangLabelMap = {
  '自動': 'settings.roleLangAuto',
  '繁中': 'settings.langZhTW',
  '英文': 'settings.langEn',
  'Auto': 'settings.roleLangAuto',
  'Traditional Chinese': 'settings.langZhTW',
  'English': 'settings.langEn',
};
const _fontPresetLabelMap = {
  '預設': 'settings.fontDefault',
  '等寬': 'settings.fontMono',
  '圓潤': 'settings.fontRound',
  'Default': 'settings.fontDefault',
  'Monospace': 'settings.fontMono',
  'Rounded': 'settings.fontRound',
};
// Any historical/localized label (zh-TW / en / ja, current + legacy) → stable key.
// Used to migrate previously-persisted display strings to language-independent keys.
const _styleLabelToKey = {
  // zh-TW
  '喔黏菊': 'default', '喔黏橘': 'default', '歐黏菊': 'default', '歐黏橘': 'default', '預設': 'default',
  '消極白': 'passiveWhite', '粉切黑': 'pinkBetrayal', '原諒青': 'forgiveMeGreen', '敗北藍': 'defeatBlue',
  // en (literal + meme-pun variants)
  'Sticky Daisy': 'default', 'Legacy Default': 'default', 'Default': 'default', 'Pretty-Please Tangerine': 'default',
  'Passive White': 'passiveWhite', 'Meh-yo White': 'passiveWhite',
  'Pink Betrayal': 'pinkBetrayal', 'Ghosted Pink': 'pinkBetrayal',
  'Forgive-me Green': 'forgiveMeGreen', 'Cuck-Green Pardon': 'forgiveMeGreen',
  'Defeat Blue': 'defeatBlue', 'L-Taken Blue': 'defeatBlue',
  // ja (literal + meme-pun variants)
  'もちもち菊': 'default', 'もちもちオレンジ': 'default', 'おねがいキク': 'default',
  '消極ホワイト': 'passiveWhite', 'どうでも白': 'passiveWhite',
  '裏切りピンク': 'pinkBetrayal', '既読スルーピンク': 'pinkBetrayal',
  '許してグリーン': 'forgiveMeGreen', '寝取られグリーン': 'forgiveMeGreen',
  '敗北ブルー': 'defeatBlue', '完敗ブルー': 'defeatBlue',
  // pt-PT
  'Laranja Faz-Favor': 'default', 'Branco Tanto-Faz': 'passiveWhite',
  'Rosa Deixado em Visto': 'pinkBetrayal', 'Verde Corno Perdoado': 'forgiveMeGreen',
  'Azul Levei um L': 'defeatBlue',
  // es
  'Naranja Porfa': 'default', 'Blanco Da Igual': 'passiveWhite',
  'Rosa Dejado en Visto': 'pinkBetrayal', 'Verde Cornudo Perdón': 'forgiveMeGreen',
  'Azul Me Dieron la L': 'defeatBlue',
  // th
  'ส้มอ้อนวอน': 'default', 'ขาวช่างมัน': 'passiveWhite',
  'ชมพูอ่านไม่ตอบ': 'pinkBetrayal', 'เขียวสวมเขา': 'forgiveMeGreen',
  'ฟ้าแพ้ราบคาบ': 'defeatBlue',
};

// Resolve any value (stable key OR localized label, any language) to a stable key.
function styleKeyOf(value) {
  if (typeof value !== 'string' || !value) return defaultPanelStyle;
  if (STYLE_KEYS.includes(value)) return value;
  return _styleLabelToKey[value] || defaultPanelStyle;
}

// Live display label for a style, in the current language.
function styleLabel(value) {
  return _t(STYLE_KEY_I18N[styleKeyOf(value)] || 'style.default');
}

// Kept name for compatibility; now returns a stable key (not a display string).
function normalizePanelStyle(style) {
  return styleKeyOf(style);
}

function normalizeStyleOptionOrder(options = []) {
  const ordered = options.map(normalizePanelStyle).filter((option) => styleOptions.includes(option));
  return [...new Set([...ordered, ...styleOptions])];
}

function loadStyleOptionOrder() {
  try {
    return normalizeStyleOptionOrder(JSON.parse(window.localStorage.getItem(styleOptionOrderStorageKey) || '[]'));
  } catch {
    return styleOptions;
  }
}

function saveStyleOptionOrder(options) {
  try {
    window.localStorage.setItem(styleOptionOrderStorageKey, JSON.stringify(normalizeStyleOptionOrder(options)));
  } catch {
    /* local UI preference only */
  }
}

function normalizePanelSettings(panel = {}) {
  return {...panel, panelStyle: normalizePanelStyle(panel.panelStyle)};
}

function panelFromUISettings(uiSettings = {}, fallbackPanel = fallbackSettings.panel) {
  return {
    panelLanguage: localizeBackendLabel(uiSettings.panel_language, _panelLangLabelMap) || fallbackPanel.panelLanguage,
    roleLanguage: localizeBackendLabel(uiSettings.role_language, _roleLangLabelMap) || fallbackPanel.roleLanguage,
    fontPreset: localizeBackendLabel(uiSettings.font_preset, _fontPresetLabelMap) || fallbackPanel.fontPreset,
    fontScale: uiSettings.font_scale || fallbackPanel.fontScale,
    panelStyle: styleKeyOf(uiSettings.panel_style || fallbackPanel.panelStyle),
  };
}

function normalizeSettingsState(settingsState = {}, fallback = fallbackSettings) {
  const merged = {
    ...fallback,
    ...settingsState,
    panel: {
      ...(fallback.panel || fallbackSettings.panel),
      ...(settingsState.panel || {}),
    },
  };
  const personas = normalizeLockedPersonas(merged.personas || fallbackSettings.personas);
  const activePersonaId = personas.some((persona) => persona.id === merged.activePersonaId)
    ? merged.activePersonaId
    : lockedPersonaId;
  return {...merged, activePersonaId, personas, panel: normalizePanelSettings(merged.panel)};
}

function normalizeLockedPersonas(personas = []) {
  // "憂樂傻酷" is a reserved persona identity, not a reserved slot. Keep its
  // name stable while preserving whatever ordering the user chose, because the
  // app treats the first card as the current main persona.
  const normalized = personas.map((persona) => (
    persona.id === lockedPersonaId
      ? {...persona, name: lockedPersonaName, identity: persona.identity || _t('persona.defaultIdentityA')}
      : persona
  ));
  const lockedIndex = normalized.findIndex((persona) => persona.id === lockedPersonaId);
  if (lockedIndex < 0) {
    return [fallbackSettings.personas[0], ...normalized];
  }
  return normalized;
}

function normalizeToolList(toolList = []) {
  const merged = new Map(getFallbackTools().map((tool) => [tool.id, tool]));
  toolList.forEach((tool) => {
    if (!tool?.id) return;
    merged.set(tool.id, {...(merged.get(tool.id) || {}), ...tool});
  });
  return Array.from(merged.values());
}

function panelStyleTheme(style) {
  return STYLE_KEY_THEME[styleKeyOf(style)] || 'onanegiku';
}

// Converts persisted values like "80%" into a bounded CSS scale number.
function fontScaleValue(fontScale) {
  const value = Number.parseInt(String(fontScale || '100%').replace('%', ''), 10);
  const normalized = Number.isNaN(value) ? 100 : value;
  return Math.min(120, Math.max(50, normalized)) / 100;
}

function getPersonaAvatar(persona) {
  return persona?.avatarUrl || personaAvatarUrls[persona?.id] || personaAvatarUrls['persona-a'];
}

function bytesToDataUrl(bytes = [], mimeType = 'image/png') {
  if (typeof bytes === 'string') return `data:${mimeType};base64,${bytes}`;
  let binary = '';
  const chunkSize = 8192;
  for (let index = 0; index < bytes.length; index += chunkSize) {
    binary += String.fromCharCode(...bytes.slice(index, index + chunkSize));
  }
  return `data:${mimeType};base64,${window.btoa(binary)}`;
}

function resolveAvatarProvider(config) {
  return config?.avatar_provider || config?.AvatarProvider || 'built_in_pixel';
}

function resolveStaticAvatarPath(config) {
  return config?.static_avatar_path || config?.StaticAvatarPath || '';
}

function defaultPixelPackForPersona(personaID) {
  if (personaID === 'persona-b') return 'uncle';
  if (personaID === 'persona-c') return 'secretary';
  return 'wolf';
}

function pixelPackForPersona(persona, config) {
  const pack = config?.pixel_pack || config?.PixelPack || '';
  if (['wolf', 'uncle', 'secretary'].includes(pack)) return pack;
  return defaultPixelPackForPersona(persona?.id);
}

function hasPendingLowRiskAmbiguity(reviewState) {
  const ambiguity = reviewState?.lowRiskAmbiguity;
  if (!ambiguity || ambiguity.dismissedUntil) return false;
  return Boolean(ambiguity.id || ambiguity.candidates?.length) && ambiguity.selectedSkillId === '';
}

function deriveAvatarExpression({dagRun, reviewState, sourceTrustHint, hasConversation = true, lastMessageText = '', windowInactive = false, idleMs = 0}) {
  if (windowInactive) return 'sleepy';
  if (reviewState?.hasBlocking || hasPendingLowRiskAmbiguity(reviewState)) return 'warning';
  if (sourceTrustHint?.risk_level === 'high' || sourceTrustHint?.risk_level === 'danger') return 'blocked';
  if (sourceTrustHint && sourceTrustHint.level !== 'trusted') return 'warning';
  if (dagRun?.status === 'running' || dagRun?.status === 'starting') return 'working';
  if (dagRun?.status === 'blocked') return 'warning';
  if (dagRun?.status === 'failed') return 'sad';
  if (dagRun?.status === 'completed') return 'happy';
  const messageExpression = deriveMessageAvatarExpression(lastMessageText);
  if (messageExpression) return messageExpression;
  if (!hasConversation) return idleMs >= 15 * 60 * 1000 ? 'idle' : 'sleepy';
  return 'idle';
}

function deriveMessageAvatarExpression(text = '') {
  const normalized = String(text || '').trim().toLowerCase();
  if (!normalized) return 'speechless';
  if (/(笑話|joke|哈哈|呵呵|好笑|冷笑話)/i.test(normalized)) return 'speechless';
  if (/(讚|太棒|做得好|漂亮|厲害|謝謝|感謝|good job|nice|great|awesome)/i.test(normalized)) return 'happy';
  if (/(罵|爛|笨|廢|糟糕|生氣|bad|stupid|useless)/i.test(normalized)) return 'sad';
  if (/(build fail|build failed|test fail|test failed|測試失敗|建置失敗|api 錯誤|llm 錯誤|錯誤|失敗)/i.test(normalized)) return 'sad';
  if (/(設定缺漏|degraded|未驗證|低風險|warning|warn)/i.test(normalized)) return 'warning';
  if (/(權限不足|credential|token 錯|discord token|高風險|需要確認|blocked|forbidden)/i.test(normalized)) return 'blocked';
  return '';
}

function staticAvatarSrc(path) {
  if (!path) return '';
  if (/^(data:|https?:|file:)/.test(path)) return path;
  return path.startsWith('/') ? path : `/${path}`;
}

function resolvePersonaAvatarSrc(persona, config, previews = {}, pixelFallback = '') {
  const provider = resolveAvatarProvider(config);
  const staticPath = resolveStaticAvatarPath(config);
  if (provider === 'static_image' && staticPath) {
    return previews[persona?.id] || staticAvatarSrc(staticPath);
  }
  return pixelFallback || getPersonaAvatar(persona);
}


function parseToneChance(roleStrength) {
  const value = Number.parseInt(String(roleStrength || '20%').replace('%', ''), 10);
  if (Number.isNaN(value)) return 20;
  return Math.min(100, Math.max(10, Math.round(value / 10) * 10));
}

function limitChineseText(text, maxLength) {
  return Array.from(String(text || '')).slice(0, maxLength).join('');
}

function reviewStatusLabel(status) {
  return {
    pending: _t('review.pendingConfirm'),
    approved: _t('review.approved'),
    rejected: _t('review.rejected'),
    custom: _t('review.customInput'),
    expired: _t('review.expired'),
  }[status] || status;
}

function reviewCardID(card) {
  return card?.review_id || card?.ID || card?.id || '';
}

function reviewConsequenceText(status, note) {
  if (note) return note;
  return {
    pending: _t('review.pendingHint'),
    approved: _t('review.approvedHint'),
    rejected: _t('review.rejectedHint'),
    custom: _t('review.customHint'),
    expired: _t('review.expiredHint'),
  }[status] || _t('review.syncHint');
}

function ConsequenceMenuOverlay({cards, result, onExecute, onRecreate}) {
  const card = (cards || []).find((item) => item.risk_class === 'user_owned_asset_destructive' && item.operation === 'purge_project');
  if (!card) return null;
  const retryCount = Number(result?.retry_count || 0);
  // i18n: confirm
  return createPortal(
    <div className="project-manage-overlay" role="dialog" aria-modal="true" aria-label={_t('confirm.destructiveTitle')}>
      <div className="project-manage-popup consequence-menu">
        <div className="project-manage-header">
          <h3>{_t('confirm.destructiveTitle')}</h3>
        </div>
        <div className="consequence-table">
          <div><span>operation</span><strong>{card.operation}</strong></div>
          <div><span>target</span><strong>{card.target}</strong></div>
          <div><span>affected scope</span><strong>{card.accept_effect}</strong></div>
          <div><span>accept effect</span><strong>{_t('confirm.acceptEffect')}</strong></div>
          <div><span>reject effect</span><strong>{card.reject_effect || _t('confirm.rejectEffect')}</strong></div>
          <div><span>rollback</span><strong>{card.rollback_available ? 'available' : 'not available'}</strong></div>
          <div><span>backup</span><strong>{card.backup_available ? 'available' : 'not available'}</strong></div>
          <div><span>log</span><strong>{card.log_location || 'review/review_decision_log.jsonl'}</strong></div>
        </div>
        {result?.message && <p className="consequence-status">{result.message}</p>}
        <div className="project-manage-actions consequence-actions">
          <button type="button" className="project-manage-btn" disabled={!card.backup_available} onClick={() => onExecute(card, 'backup_then_execute')}>
            <span>{_t('confirm.backupThenExecute')}</span>
          </button>
          <button type="button" className="project-manage-btn project-manage-btn-danger" onClick={() => onExecute(card, 'direct_execute')}>
            <span>{_t('confirm.directExecute')}</span>
          </button>
          <button type="button" className="project-manage-btn" onClick={() => onExecute(card, 'cancel_keep_pending')}>
            <span>{_t('confirm.cancel')}</span>
          </button>
          {retryCount === 1 && (
            <button type="button" className="project-manage-btn" onClick={() => onRecreate(card)}>
              <span>{_t('confirm.recreateReviewCard')}</span>
            </button>
          )}
        </div>
      </div>
    </div>,
    document.body,
  );
}

function replyStrategyPresetFor(value) {
  const key = String(value || '').trim();
  if (!key) return null;
  return getReplyStrategyPresets().find((preset) => preset.id === key || preset.label === key) || null;
}

// ── v3.6 專案管理彈窗元件 ──
// 左側欄「專案管理」按鈕開啟，包含三項操作：
//   清除專案資料 (PurgeProject)、清除隔離區暫存 (PurgeBoundaryDir)、查看清除紀錄 (ListPurgeManifests)
// 清除操作使用雙步驟按鈕確認（同 DualStepConfirm 風格）。
// 紀錄以彈窗內捲動列表顯示。
function ProjectManagePopup({ view, manifests, confirmStep, onPurgeProject, onPurgeBoundary, onViewManifests, onBackToMenu, onClose }) {
  const {t} = useI18n();
  // i18n: project
  return (
    <div className="project-manage-overlay" role="dialog" aria-modal="true" aria-label={t('project.manageTitle')}>
      <div className="project-manage-popup">
        <div className="project-manage-header">
          <h3>{view === 'manifests' ? t('project.manifestsTitle') : t('project.manageTitle')}</h3>
          <button type="button" className="project-manage-close" onClick={onClose} aria-label={t('common.close')}>×</button>
        </div>
        {view === 'menu' ? (
          <div className="project-manage-actions">
            <button
              type="button"
              className={`project-manage-btn ${confirmStep === 'project' ? 'project-manage-btn-danger' : ''}`}
              onClick={onPurgeProject}
            >
              <span className="project-manage-icon">⚠</span>
              <span>{confirmStep === 'project' ? t('project.confirmPurge') : t('project.purgeProjectData')}</span>
            </button>
            <button
              type="button"
              className={`project-manage-btn ${confirmStep === 'boundary' ? 'project-manage-btn-danger' : ''}`}
              onClick={onPurgeBoundary}
            >
              <span className="project-manage-icon">⚠</span>
              <span>{confirmStep === 'boundary' ? t('project.confirmPurge') : t('project.purgeBoundaryData')}</span>
            </button>
            <button type="button" className="project-manage-btn" onClick={onViewManifests}>
              <span className="project-manage-icon">☰</span>
              <span>{t('project.viewManifests')}</span>
            </button>
          </div>
        ) : (
          <div className="project-manage-manifests">
            <button type="button" className="project-manage-back" onClick={onBackToMenu}>← {t('common.back')}</button>
            {manifests.length === 0 ? (
              <p className="project-manage-empty">{t('project.noManifests')}</p>
            ) : (
              <ul className="project-manage-list">
                {manifests.map((m, i) => (
                  <li key={m.id || i} className="project-manage-list-item">
                    <span className="manifest-time">{m.timestamp || m.created_at || '—'}</span>
                    <span className="manifest-type">{m.trigger || m.purge_type || t('project.manualPurge')}</span>
                    <span className="manifest-count">{t('project.fileCount', { count: m.file_count || m.files?.length || 0 })}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>
    </div>
  );
}



function cycleValue(options, current) {
  const index = options.indexOf(current);
  return options[(index + 1) % options.length] || options[0];
}

function voiceStatusLabel(status) {
  switch (status) {
    case 'ready':
      return 'whisper.cpp ready';
    case 'invalid_model':
      return _t('voice.voiceModelReinstall');
    case 'missing_binary':
      return _t('settings.voiceNotIncluded');
    case 'missing_binary_and_model':
      return _t('settings.voiceNotInstalled');
    case 'missing_model':
      return _t('settings.voiceModelNotInstalled');
    default:
      return _t('settings.voiceNotInstalled');
  }
}

function voiceLanguageModeLabel(mode) {
  switch (mode) {
    case 'follow_app':
      return _t('settings.voiceFollowApp');
    case 'manual':
      return _t('settings.voiceManualSpecify');
    case 'auto':
    default:
      return _t('settings.voiceAutoDetect');
  }
}

function reorderItems(items, sourceItem, targetItem) {
  const sourceIndex = items.indexOf(sourceItem);
  const targetIndex = items.indexOf(targetItem);
  if (sourceIndex < 0 || targetIndex < 0) return items;
  const next = [...items];
  next.splice(sourceIndex, 1);
  next.splice(targetIndex, 0, sourceItem);
  return next;
}

/* i18n: onboarding tool demo */
function OnboardingToolDemo() {
  const t = useI18n(s => s.t);
  // i18n: onboarding
  return (
    <div className="onboarding-tool-demo" aria-label={t('onboarding.toolDemoAriaLabel')}>
      <div className="tool-demo-stage">
        <aside className="tool-demo-rail" aria-hidden="true">
          <span className="tool-demo-rail-label">{t('onboarding.leftRail')}</span>
          <button className="tool-demo-rail-btn" type="button">
            <span>⊕</span>
            <span>{t('onboarding.newAgent')}</span>
          </button>
          <button className="tool-demo-rail-btn tool-demo-trigger" type="button">
            <span>⌕</span>
            <span>{t('onboarding.tools')}</span>
          </button>
          <button className="tool-demo-rail-btn" type="button">
            <span>⚙</span>
            <span>{t('onboarding.settings')}</span>
          </button>
        </aside>

        <section className="tool-demo-panel" aria-hidden="true">
          <header className="tool-demo-tabs">
            <span className="active">{t('onboarding.tabExternalLinks')}</span>
            <span>{t('onboarding.tabAutomation')}</span>
            <span>{t('onboarding.tabPackage')}</span>
          </header>
          <div className="tool-demo-list">
            <div className="tool-demo-item tool-demo-item-primary">
              <span>▤</span>
              <strong>{t('onboarding.refLink')}</strong>
              <small>{t('onboarding.refLinkHint')}</small>
            </div>
            <div className="tool-demo-item">
              <span>▤</span>
              <strong>{t('onboarding.refDoc')}</strong>
              <small>{t('onboarding.refDocHint')}</small>
            </div>
            <div className="tool-demo-item">
              <span>✉</span>
              <strong>Gmail</strong>
              <small>{t('onboarding.gmailHint')}</small>
            </div>
          </div>
          <div className="tool-demo-drag-chip">
            <span>▤</span>
            <strong>{t('onboarding.refLink')}</strong>
          </div>
        </section>

        <section className="tool-demo-favorites" aria-hidden="true">
          <span className="tool-demo-drop-label">{t("rightRail.useTools")}</span>
          <strong>{t('onboarding.toolFavorites')}</strong>
          <p>{t('onboarding.toolDragHint')}</p>
          <div className="tool-demo-drop-slot">
            <span>{t('onboarding.dropHere')}</span>
          </div>
        </section>
      </div>

      <ol className="tool-demo-captions">
        <li><span>1</span>{t('onboarding.toolStep1')}</li>
        <li><span>2</span>{t('onboarding.toolStep2')}</li>
        <li><span>3</span>{t('onboarding.toolStep3')}</li>
      </ol>
    </div>
  );
}

// ── v2.4 Onboarding 引導覆蓋層 ──────────────────────────────────
// 首次啟動時全螢幕覆蓋，5 步驟引導：歡迎→偵測 CLI→設定人格→試用工具→完成。
// 「設定人格」步驟切換為 spotlight 模式：半透明背景 + 圈圈高亮設定頁人格區域。
// onboarding 完成後呼叫 onFinish() 移除覆蓋，寫入 .first_run_done marker。

/* i18n: onboarding overlay */
function OnboardingOverlay({state, onCompleteStep, onFinish, onGoBack, onDetectCLI, onEnableCLI, onScanLocalModels, onEnableLocalModel, onRegisterCustomCLI, onOpenSettings, onCloseSettings}) {
  const t = useI18n(s => s.t);
  const [detectResults, setDetectResults] = useState(null);
  const [detecting, setDetecting] = useState(false);
  const [enabledIds, setEnabledIds] = useState(new Set());
  const [showManual, setShowManual] = useState(false);
  const [manualName, setManualName] = useState('');
  const [manualPath, setManualPath] = useState('');
  const [manualError, setManualError] = useState('');
  const [localModelResults, setLocalModelResults] = useState([]);
  if (!state || !state.is_first_run) return null;

  const steps = state.steps || [];
  const currentIdx = state.current_step ?? 0;
  const current = steps[currentIdx];
  const isFirstStep = currentIdx === 0;
  const isLastStep = currentIdx >= steps.length - 1;
  const isPersonaStep = current?.id === 'persona';
  const isToolStep = current?.id === 'tool';

  // CLI + Local model 偵測 handler（只偵測，不註冊）
  async function runDetect() {
    setDetecting(true);
    try {
      const [cliResults, localResults] = await Promise.all([
        onDetectCLI().catch(() => []),
        onScanLocalModels?.().catch(() => []),
      ]);
      setDetectResults(cliResults || []);
      setLocalModelResults(localResults || []);
    } catch {
      setDetectResults([]);
      setLocalModelResults([]);
    }
    setDetecting(false);
  }

  // 使用者點「啟用」→ 註冊 CLI 到 registry
  async function handleEnable(adapterID) {
    try {
      await onEnableCLI(adapterID);
      setEnabledIds((prev) => new Set([...prev, adapterID]));
    } catch { /* ignore */ }
  }

  // 使用者點「啟用」→ 註冊 Local model 到 registry
  async function handleEnableLocal(result) {
    try {
      await onEnableLocalModel(result);
      setEnabledIds((prev) => new Set([...prev, result.adapter_id]));
    } catch { /* ignore */ }
  }

  // 手動註冊 CLI
  async function handleManualRegister() {
    setManualError('');
    if (!manualName.trim() || !manualPath.trim()) {
      setManualError(t('onboarding.fillNamePath'));
      return;
    }
    try {
      const result = await onRegisterCustomCLI(manualName.trim(), manualPath.trim());
      if (result && result.found) {
        setDetectResults((prev) => [...(prev || []), result]);
        setEnabledIds((prev) => new Set([...prev, result.adapter_id]));
        setManualName('');
        setManualPath('');
        setShowManual(false);
      }
    } catch (e) {
      setManualError(e?.message || t('onboarding.registerFail'));
    }
  }

  // 下一步
  function handleNext() {
    // 離開人格步驟 → 關閉設定頁
    if (isPersonaStep) onCloseSettings?.();
    if (current) onCompleteStep(current.id);
    if (isLastStep) onFinish();
    // 進入人格步驟 → 開啟設定頁
    const nextStep = steps[currentIdx + 1];
    if (nextStep?.id === 'persona') onOpenSettings?.();
  }

  // 上一步
  function handlePrev() {
    if (isPersonaStep) onCloseSettings?.();
    if (isFirstStep) return;
    const prevStep = steps[currentIdx - 1];
    if (prevStep) {
      onGoBack?.();
      if (prevStep.id === 'persona') onOpenSettings?.();
    }
  }

  function handleSkip() {
    if (isPersonaStep) onCloseSettings?.();
    onFinish();
  }

  const foundCount = detectResults ? detectResults.filter((r) => r.found).length : 0;

  // 人格步驟 → spotlight 模式（半透明背景，讓底層 UI 可見）
  const isSpotlightMode = isPersonaStep;

  return (
    <div className={`onboarding-overlay${isSpotlightMode ? ' spotlight-mode' : ''}`}>
      <div className={`onboarding-card${isSpotlightMode ? ' spotlight-card' : ''}${isToolStep ? ' tool-demo-card' : ''}`}>
        {/* 進度指示器 */}
        <div className="onboarding-progress">
          {steps.map((s, i) => (
            <div key={s.id} className={`onboarding-dot${i === currentIdx ? ' active' : ''}${s.completed ? ' done' : ''}`} />
          ))}
        </div>

        <h2 className="onboarding-title">{current?.title || t('onboarding.completedTitle')}</h2>
        <p className="onboarding-desc">{current?.description || ''}</p>

        {/* 人格步驟：spotlight 指引 */}
        {isPersonaStep && (
          <div className="onboarding-spotlight-hint">
            <div className="spotlight-arrow">←</div>
            <p>{t('onboarding.spotlightHint')}</p>
          </div>
        )}

        {/* 試用工具步驟：卡片內完整示範 */}
        {isToolStep && <OnboardingToolDemo />}

        {/* Adapter 偵測 + 啟用 */}
        {current?.id === 'adapter' && (
          <div className="onboarding-adapter-section">
            {!detecting && (
              <button className="onboarding-detect-btn" type="button" onClick={runDetect}>
                {detectResults ? t('onboarding.rescan') : t('onboarding.scanCLI')}
              </button>
            )}
            {detecting && <p className="onboarding-scanning">{t('onboarding.scanning')}</p>}

            {detectResults && foundCount > 0 && (
              <ul className="onboarding-detect-list">
                {detectResults.filter((r) => r.found).map((r) => {
                  const isEnabled = enabledIds.has(r.adapter_id);
                  return (
                    <li key={r.adapter_id} className={isEnabled ? 'found enabled' : 'found'}>
                      <span className="detect-name">{r.name}</span>
                      <span className="detect-path">{r.path}</span>
                      {isEnabled ? (
                        <span className="detect-enabled-badge">{t('onboarding.enabled')}</span>
                      ) : (
                        <button className="detect-enable-btn" type="button" onClick={() => handleEnable(r.adapter_id)}>{t('onboarding.enable')}</button>
                      )}
                    </li>
                  );
                })}
              </ul>
            )}

            {detectResults && foundCount === 0 && (
              <p className="onboarding-install-hint">
                {t('onboarding.noCLI')}
                <br />
                <code>npm install -g @anthropic-ai/claude-code</code>
              </p>
            )}

            {detectResults && foundCount > 0 && (
              <p className="onboarding-detect-summary">
                {t('onboarding.cliFound', { foundCount, enabledCount: enabledIds.size })}
              </p>
            )}

            {/* Local model detection results */}
            {localModelResults && localModelResults.length > 0 && (
              <>
                <p className="onboarding-detect-summary" style={{marginTop: '12px'}}>
                  {t('onboarding.localModelFound', { count: localModelResults.length })}
                </p>
                <ul className="onboarding-detect-list">
                  {localModelResults.map((r) => {
                    const isEnabled = enabledIds.has(r.adapter_id);
                    return (
                      <li key={r.adapter_id} className={isEnabled ? 'found enabled local-model' : 'found local-model'}>
                        <span className="detect-name">{r.name}</span>
                        <span className="detect-path">{r.provider}</span>
                        {isEnabled ? (
                          <span className="detect-enabled-badge">{t("onboarding.enabled")}</span>
                        ) : (
                          <button className="detect-enable-btn" type="button" onClick={() => handleEnableLocal(r)}>{t("onboarding.enable")}</button>
                        )}
                      </li>
                    );
                  })}
                </ul>
              </>
            )}

            {detectResults && (
              <div className="onboarding-manual-section">
                {!showManual ? (
                  <button className="onboarding-manual-toggle" type="button" onClick={() => setShowManual(true)}>
                    {t('onboarding.addCLIManual')}
                  </button>
                ) : (
                  <div className="onboarding-manual-form">
                    <input type="text" className="onboarding-manual-input" placeholder={t('onboarding.namePlaceholder')}
                      value={manualName} onChange={(e) => setManualName(e.target.value)} />
                    <input type="text" className="onboarding-manual-input" placeholder={t('onboarding.pathPlaceholder')}
                      value={manualPath} onChange={(e) => setManualPath(e.target.value)} />
                    {manualError && <p className="onboarding-manual-error">{manualError}</p>}
                    <div className="onboarding-manual-actions">
                      <button type="button" onClick={handleManualRegister}>{t('onboarding.confirmAdd')}</button>
                      <button type="button" className="onboarding-manual-cancel" onClick={() => { setShowManual(false); setManualError(''); }}>{t('onboarding.cancel')}</button>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* 導航按鈕列 */}
        <div className="onboarding-actions">
          {!isFirstStep && (
            <button className="onboarding-prev" type="button" onClick={handlePrev} title={t('onboarding.prevStep')}>
              ←
            </button>
          )}
          <button className="onboarding-skip" type="button" onClick={handleSkip}>{t('onboarding.skip')}</button>
          <button className="onboarding-next" type="button" onClick={handleNext}>
            {isLastStep ? t('onboarding.getStarted') : '→'}
          </button>
        </div>
      </div>
    </div>
  );
}

function PackageInstallDecisionDialog({candidate, onConfirm, onCancel}) {
  const {t} = useI18n();
  const typeLabel = candidate?.typeLabel || candidate?.type || 'package';
  const name = candidate?.name || t('package.unnamedItem');
  const preview = candidate?.preview || {};
  const detail = candidate?.type === 'subagent'
    ? t('package.systemCodeDesc', { code: preview.source_system_code || 'unknown', toolCount: preview.tool_count || 0 })
    : candidate?.type === 'skill'
      ? t('package.skillIdDesc', { skillId: preview.SkillID || preview.skill_id || 'unknown', resourceCount: (preview.Resources || preview.resources || []).length })
      : t('package.sourceFrom', { path: candidate?.path || 'unknown' });
  // i18n: package
  return (
    <div className="export-dialog-overlay">
      <div className="export-dialog sub-import-dialog">
        <p>{t('package.installQuestion', { type: typeLabel })}</p>
        <strong>{name}</strong>
        <small>{detail}</small>
        <div className="export-dialog-actions">
          <button type="button" onClick={onConfirm}>{t('package.installBtn')}</button>
          <button type="button" onClick={onCancel}>{t('common.cancel')}</button>
        </div>
      </div>
    </div>
  );
}

function SubToolConflictDialog({result, onResolve, onCancel}) {
  const {t} = useI18n();
  const conflicts = result?.tool_conflicts || [];
  return (
    <div className="export-dialog-overlay">
      <div className="export-dialog sub-import-dialog">
        <p>{t('package.conflictCount', { count: conflicts.length })}</p>
        <small>{t('package.conflictHint')}</small>
        <div className="sub-conflict-list">
          {conflicts.slice(0, 4).map((tool) => (
            <span key={`${tool.type}:${tool.original_id}`}>{tool.type} / {tool.original_id}</span>
          ))}
        </div>
        <div className="export-dialog-actions">
          <button type="button" onClick={() => onResolve('keep_all')}>{t('package.keepAll')}</button>
          <button type="button" onClick={() => onResolve('overwrite_all')}>{t('package.overwriteAll')}</button>
          <button type="button" onClick={onCancel}>{t('package.handleLater')}</button>
        </div>
      </div>
    </div>
  );
}

function Sidebar({
  adapters, adapterList, activeAdapterId,
  adapterModelChoices, adapterModelOptions, onAdapterModelPick, onAdapterModelRefresh,
  onAdapterSelect, onLocalAdapterWake, adapterCandidateLinks, activePanel,
  isToolPopupOpen, panelSettings, onPanelChange, voiceState, voiceInstallBusy,
  onVoiceSettingsChange, onVoiceSettingsRefresh, onVoiceModelInstall, onVoiceModelRemove, onRestoreDefaults, onTogglePanel,
  onToolPopupToggle, onProjectManageOpen, onCreateSubagent, onAdaptersReordered, onAdaptersChanged, onAdapterRemove,
}) {
  const t = useI18n(s => s.t);
  const settingsOpen = activePanel === 'settings';
  // §31: 拖曳狀態
  const [draggedAdapter, setDraggedAdapter] = useState(null);
  const draggedAdapterRef = useRef(null);
  const adapterPointerDragRef = useRef(null);
  const adapterPointerSelectRef = useRef({key: '', time: 0});
  const [exportDialog, setExportDialog] = useState(null); // {name, key} 或 null
  // §M-1 model picker 彈窗：{adapterID, rect, label} 或 null
  const [modelPicker, setModelPicker] = useState(null);
  useEffect(() => {
    if (!modelPicker) return;
    const close = (ev) => {
      if (ev.target && ev.target.closest && ev.target.closest('.adapter-model-picker')) return;
      setModelPicker(null);
    };
    const esc = (ev) => { if (ev.key === 'Escape') setModelPicker(null); };
    window.addEventListener('mousedown', close);
    window.addEventListener('keydown', esc);
    return () => {
      window.removeEventListener('mousedown', close);
      window.removeEventListener('keydown', esc);
    };
  }, [modelPicker]);

  // 優先使用 adapterList（含狀態的完整物件），fallback 用純名稱陣列
  const adapterItems = (adapterList && adapterList.length > 0)
    ? adapterList.map((raw) => {
        const a = normalizeAdapterDTO(raw);
        const id = adapterKey(a);
        const rawKind = String(a.kind || '').toLowerCase();
        const inferredKind = rawKind || (String(id || '').startsWith('llm-api-') ? 'api' : ((a.name || id || '').toLowerCase() === 'main' ? 'main' : 'cli'));
        const rawName = a.name || a.id;
        const localProviderName = String(rawName || id || '').toLowerCase().includes('lm studio') ? 'LM Studio' : 'Ollama';
        return {
          key: id,
          name: rawName,
          displayName: inferredKind === 'local' ? localProviderName : rawName,
          icon: a.icon,
          status: a.status || 'offline',
          kind: inferredKind,
          model: a.model || '',
          endpoint: a.endpoint || '',
          isMain: inferredKind === 'main' || (a.name || a.id || '').toLowerCase() === 'main',
        };
      })
    : (adapters || []).map((name) => ({
        key: name,
        name,
        displayName: name,
        icon: (adapterMeta[name] || adapterMeta.Claude).icon,
        status: 'offline',
        isMain: name.toLowerCase() === 'main',
        kind: name.toLowerCase() === 'main' ? 'main' : 'cli',
      }));
  const isRegistryAdapter = (item) => item && (item.kind === 'cli' || item.kind === 'api' || item.kind === 'local');
  /* i18n: adapter status */ const statusLabel = {online: t('adapter.statusOnline'), offline: t('adapter.statusOffline'), degraded: t('adapter.statusDegraded')};

  const selectAdapterItem = (item, source = 'click') => {
    if (source === 'click') {
      const last = adapterPointerSelectRef.current;
      if (last.key === item.key && Date.now() - last.time < 500) return;
    }
    if (source === 'pointer') {
      adapterPointerSelectRef.current = {key: item.key, time: Date.now()};
    }
    onAdapterSelect?.(item.key);
    const disconnected = item.status === 'offline' || item.status === 'error' || item.status === 'degraded';
    if (item.kind === 'local' && disconnected) onLocalAdapterWake?.(item);
  };

  const moveAdapterItem = (draggedKey, targetKey) => {
    const draggedItem = adapterItems.find((a) => a.key === draggedKey);
    const targetItem = adapterItems.find((a) => a.key === targetKey);
    if (!draggedItem) return;
    if (!targetItem) return;
    const bothSub = draggedItem.kind === 'sub' && targetItem.kind === 'sub';
    const bothAdapters = isRegistryAdapter(draggedItem) && isRegistryAdapter(targetItem);
    if (!bothSub && !bothAdapters) return;

    const orderedItems = adapterItems.filter((a) => bothSub ? a.kind === 'sub' : isRegistryAdapter(a));
    const fromIdx = orderedItems.findIndex((a) => a.key === draggedKey);
    const toIdx = orderedItems.findIndex((a) => a.key === targetItem.key);
    if (fromIdx >= 0 && toIdx >= 0 && fromIdx !== toIdx) {
      const newOrder = orderedItems.map((a) => a.key);
      const [moved] = newOrder.splice(fromIdx, 1);
      newOrder.splice(toIdx, 0, moved);
      onAdaptersReordered?.(newOrder, bothSub ? 'sub' : 'adapter');
      const reorderCall = bothSub ? ReorderTabs : ReorderAdapters;
      callWails(() => reorderCall(JSON.stringify(newOrder)))
        .then(() => onAdaptersChanged?.())
        .catch(console.error);
    }
  };

  const startAdapterPointerDrag = (event, item) => {
    selectAdapterItem(item, 'pointer');
    if (!item || item.isMain) return;
    adapterPointerDragRef.current = {
      key: item.key,
      startX: event.clientX,
      startY: event.clientY,
      moved: false,
      pointerId: event.pointerId,
    };
    draggedAdapterRef.current = item.key;
    event.currentTarget.setPointerCapture?.(event.pointerId);
  };

  const handleAdapterPointerMove = (event) => {
    const drag = adapterPointerDragRef.current;
    if (!drag) return;
    const distance = Math.hypot(event.clientX - drag.startX, event.clientY - drag.startY);
    if (distance < 8 && !drag.moved) return;
    drag.moved = true;
    event.preventDefault();
    setDraggedAdapter(drag.key);
    const target = document.elementFromPoint?.(event.clientX, event.clientY)?.closest?.('[data-adapter-key]');
    const targetKey = target?.dataset?.adapterKey;
    if (targetKey && targetKey !== drag.key) {
      moveAdapterItem(drag.key, targetKey);
    }
  };

  const finishAdapterPointerDrag = (event) => {
    const drag = adapterPointerDragRef.current;
    if (!drag) return;
    event.currentTarget.releasePointerCapture?.(drag.pointerId ?? event.pointerId);
    const leftWindow =
      event.clientX <= 0 ||
      event.clientY <= 0 ||
      event.clientX >= window.innerWidth ||
      event.clientY >= window.innerHeight;
    const droppedOutside = leftWindow || (event.clientX === 0 && event.clientY === 0);
    if (drag.moved && droppedOutside) {
      const item = adapterItems.find((a) => a.key === drag.key);
      if (item && !item.isMain) {
        setExportDialog({name: item.name, key: item.key, kind: item.kind});
      }
    }
    draggedAdapterRef.current = null;
    setDraggedAdapter(null);
    setTimeout(() => {
      if (adapterPointerDragRef.current === drag) {
        adapterPointerDragRef.current = null;
      }
    }, 0);
  };

  // §31.3: 匯出對話框操作
  const handleExportAction = async (action) => {
    if (!exportDialog) return;
    const {name, key} = exportDialog;
    setExportDialog(null);
    if (action === 'cancel') return;
    if (isRegistryAdapter(exportDialog)) {
      if (action === 'remove') {
        await onAdapterRemove?.(key);
        await onAdaptersChanged?.();
      }
      return;
    }
    try {
      const mode = action === 'remove' ? 'export_remove' : 'export_copy';
      const destDir = await callWails(SelectSubExportDirectory);
      if (!destDir) return;
      await callWails(() => ExportSubHandler(key, name, mode, destDir, '[]'));
      await onAdaptersChanged?.();
    } catch (err) {
      console.error('[EXPORT]', err);
    }
  };

  return (
    <aside className="left-panel">
      {/* #I-806: adapter 清單由 Sidecar IPC→adapter_registry 白名單審查後動態更新 */}
      {/* 「可選 Adapter」為提示標籤，非按鈕；下方列出已連接的 CLI adapter */}
      <nav className="adapter-stack">
        <span className="adapter-hint">{t('adapter.selectHint')}</span>
        {adapterItems.map((item) => {
          const meta = adapterMeta[item.name] || adapterMeta.Claude;
          const isActive = item.name === activeAdapterId || item.key === activeAdapterId;
          const sourceBadge = item.kind === 'api' ? 'api' : (item.kind === 'local' ? 'loc' : '');
          const disconnected = item.status === 'offline' || item.status === 'error' || item.status === 'degraded';
          const sourceClass = disconnected ? 'adapter-disconnected' : (item.kind === 'local' ? 'adapter-local-model' : (item.kind === 'api' ? 'adapter-api' : 'adapter-cli'));
          const titleName = item.kind === 'local' && item.model ? `${item.displayName} — ${item.model}` : item.name;
          return (
            <button
              className={`adapter-btn ${sourceClass}${isActive ? ' adapter-active' : ''}${draggedAdapter === item.key ? ' adapter-dragging' : ''}`}
              type="button"
              key={item.key}
              data-adapter-key={item.key}
              draggable={false}
              onPointerDown={(event) => startAdapterPointerDrag(event, item)}
              onPointerMove={handleAdapterPointerMove}
              onPointerUp={finishAdapterPointerDrag}
              onPointerCancel={finishAdapterPointerDrag}
              onDragStart={(event) => event.preventDefault()}
              onClick={(event) => {
                if (adapterPointerDragRef.current?.moved) {
                  event.preventDefault();
                  return;
                }
                selectAdapterItem(item);
              }}
              onContextMenu={(event) => {
                if (item.isMain) return;
                event.preventDefault();
                setExportDialog({name: item.name, key: item.key, kind: item.kind});
              }}
              onDoubleClick={(e) => {
                e.preventDefault();
                // §M-1：有候選清單就開 model picker；無候選保留原 local wake 行為
                const opts = adapterModelOptions?.[item.key];
                if (Array.isArray(opts) && opts.length > 0) {
                  const rect = e.currentTarget.getBoundingClientRect();
                  setModelPicker({adapterID: item.key, rect, label: item.displayName || item.name});
                  return;
                }
                if (item.kind === 'local') onLocalAdapterWake?.(item);
              }}
              title={`${titleName} — ${statusLabel[item.status] || item.status}${item.isMain ? '' : t('adapter.dragToUnlink')}`}
            >
              <span className="adapter-icon">{item.icon || meta.icon}</span>
              <span className="adapter-name">{item.displayName || item.name}</span>
              {sourceBadge && <span className="adapter-source-badge">{sourceBadge}</span>}
              <span className={`adapter-status-dot status-${item.status}`} />
            </button>
          );
        })}
        {/* I-5: adapter_candidate 外部連結 — 從 ListExternalLinksByType("adapter_candidate") */}
        {(adapterCandidateLinks || []).map((link) => (
          <button className="adapter-btn adapter-cli" type="button" key={link.id} title={link.url}>
            <span className="adapter-icon">↗</span>
            <span>{link.label || link.url}</span>
          </button>
        ))}
      </nav>

      {/* §31.3: 匯出對話框 */}
      {exportDialog && (
        <div className="export-dialog-overlay">
          <div className="export-dialog">
            <p>{t('adapter.exportQuestion', { name: exportDialog.name })}</p>
            <div className="export-dialog-actions">
              <button type="button" onClick={() => handleExportAction('remove')}>{t('adapter.remove')}</button>
              <button
                type="button"
                disabled={isRegistryAdapter(exportDialog)}
                className={isRegistryAdapter(exportDialog) ? 'export-action-disabled' : ''}
                onClick={() => handleExportAction('copy')}
              >
                {t('adapter.copyAction')}
              </button>
              <button type="button" onClick={() => handleExportAction('cancel')}>{t('common.cancel')}</button>
            </div>
            {exportDialog.kind === 'cli' && <small>{t('adapter.cliExportNote')}</small>}
            {exportDialog.kind === 'api' && <small>{t('adapter.apiExportNote')}</small>}
          </div>
        </div>
      )}
      <nav className="command-stack">
        <button className="command-btn command-primary" type="button" onClick={onCreateSubagent}><span>⊕</span><span>{t('system.newAgent')}</span></button>
        <button className="command-btn command-tool-trigger" type="button" onClick={onToolPopupToggle}>
          <span>{isToolPopupOpen ? '×' : '⌕'}</span>
          <span>{isToolPopupOpen ? t('common.close') : t('onboarding.tools')}</span>
        </button>
        <button className="command-btn" type="button" onClick={() => onTogglePanel('settings')}>
          <span>{settingsOpen ? '‹' : '⚙'}</span>
          <span>{settingsOpen ? t('common.close') : t('onboarding.settings')}</span>
        </button>
        <button className="command-btn" type="button" onClick={onProjectManageOpen}>
          <span>⛁</span>
          <span>{t('project.manageTitle')}</span>
        </button>
      </nav>
      {settingsOpen && (
        <SettingsMenu
          panel={panelSettings}
          onPanelChange={onPanelChange}
          voiceState={voiceState}
          voiceInstallBusy={voiceInstallBusy}
          onVoiceSettingsChange={onVoiceSettingsChange}
          onVoiceSettingsRefresh={onVoiceSettingsRefresh}
          onVoiceModelInstall={onVoiceModelInstall}
          onVoiceModelRemove={onVoiceModelRemove}
          onRestoreDefaults={onRestoreDefaults}
        />
      )}
      {/* §M-1 model picker 彈窗 — 雙擊 adapter 卡片時顯示，貼在卡片右側 */}
      {modelPicker && typeof document !== 'undefined' && createPortal(
        <div
          className="adapter-model-picker"
          style={{
            left: Math.max(12, Math.min(modelPicker.rect.right + 10, (typeof window !== 'undefined' ? window.innerWidth : 9999) - 300)),
            top: Math.max(12, Math.min(modelPicker.rect.top, (typeof window !== 'undefined' ? window.innerHeight : 9999) - 360)),
          }}
        >
          <header className="adapter-model-picker-head">
            <div>
              <div className="adapter-model-picker-title">{t('adapter.pickModel') || '選擇 model'}</div>
              <small>{modelPicker.label || modelPicker.adapterID}</small>
            </div>
            <div className="adapter-model-picker-actions">
              <button
                type="button"
                title={t('adapter.refreshModels') || '重新整理模型'}
                aria-label={t('adapter.refreshModels') || '重新整理模型'}
                onClick={() => onAdapterModelRefresh?.(modelPicker.adapterID)}
              >
                ↻
              </button>
              <button type="button" aria-label={t('common.close')} onClick={() => setModelPicker(null)}>×</button>
            </div>
          </header>
          {(adapterModelOptions?.[modelPicker.adapterID] || []).map((m) => {
            const isActive = (adapterModelChoices?.[modelPicker.adapterID] || (adapterModelOptions[modelPicker.adapterID] || [])[0]) === m;
            return (
              <button
                key={m}
                type="button"
                className={`adapter-model-picker-item${isActive ? ' active' : ''}`}
                onClick={() => {
                  onAdapterModelPick?.(modelPicker.adapterID, m);
                  setModelPicker(null);
                }}
              >
                <span>{m}</span>
                {isActive && <b>{t('adapter.currentModel') || '目前'}</b>}
              </button>
            );
          })}
        </div>,
        document.body
      )}
    </aside>
  );
}

// Tool launcher popup: tabbed tools, icon search, drag/favorite behavior, and unavailable overlays.
// I-5: externalServiceLinks / documentationLinks 來自 ListExternalLinksByType，取代靜態假資料。
//       documentation 僅做參考顯示，不進工具執行區。
// I-6 (#I-601): onReorder(toolId, newRank) 拖曳排序回呼 → 呼叫 SetToolPreference 儲存偏好。
// Recording catalog shown in the flow popup. DAG run history belongs in the
// review/debug surfaces, not in the user's recording asset list.
function ToolFlowSectionLabel({children}) {
  return (
    <div className="tool-flow-section-label" aria-hidden="true">
      {children}
    </div>
  );
}

function RecordingCatalogList() {
  const t = useI18n(s => s.t);
  const [recordings, setRecordings] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const loadRecordings = () => {
      setLoading(true);
      callWails(() => ListLearningReplayCatalog(20))
        .then((data) => {
          if (!cancelled) setRecordings(Array.isArray(data) ? data : []);
        })
        .catch(() => {
          if (!cancelled) setRecordings([]);
        })
        .finally(() => {
          if (!cancelled) setLoading(false);
        });
    };
    loadRecordings();
    const offStopped = EventsOn('visual_learning:recording_stopped', loadRecordings);
    return () => {
      cancelled = true;
      offStopped && offStopped();
    };
  }, []);

  if (loading || !recordings || recordings.length === 0) return null;

  return (
    <>
        <ToolFlowSectionLabel>{t('tool.flowCategoryRecording')}</ToolFlowSectionLabel>
        {recordings.map((recording) => {
          const tag = recording.tag || recording.operation_tag || recording.run_id || t('recordingCatalog.untitled');
          const title = recording.title || recording.summary || tag;
          const stoppedAt = recording.stopped_at ? new Date(recording.stopped_at).toLocaleString('zh-TW', {hour12: false}) : '';
          return (
            <button
              key={recording.run_id || tag}
              type="button"
              className="tool-menu-item flow-asset-card flow-recording-card"
              data-full-title={title}
              aria-label={`${tag} ${title}`}
              aria-disabled="true"
              onDragStart={(event) => event.preventDefault()}
            >
              {/* Keep recordings framed like tool cards so the flow popup stays visually unified. */}
              <span className="tool-menu-icon">◇</span>
              <strong>{tag}</strong>
              <small>
                {recording.step_count || 0} {t('recordingCatalog.steps')}
                {stoppedAt ? ` · ${stoppedAt}` : ''}
              </small>
            </button>
          );
        })}
    </>
  );
}

const goProgramDagStepCatalog = [
  {
    id: 'import',
    title: '接收程式草稿',
    detail: '模型只提交 JSON draft；系統建立受控工作區與 Go 原始碼檔案。',
  },
  {
    id: 'validate',
    title: '契約與權限檢查',
    detail: '檢查 package main、schema、stdlib/vendor allowlist，攔下未授權網路或子程序。',
  },
  {
    id: 'build',
    title: '受控編譯',
    detail: '使用內建 Go toolchain 編譯，禁止下載 module，套用 timeout 與輸出限制。',
  },
  {
    id: 'plan',
    title: '準備資料輸入',
    detail: '把天氣 JSON、衣服表格或 DB 查詢結果整理成 stdin JSON，寫入只走 scratch。',
  },
  {
    id: 'execute',
    title: '執行與結果審核',
    detail: '限時執行，驗證 stdout JSON contract；成功後只保存為 pending skill。',
  },
];

function goProgramStepStatus(run, index) {
  const status = String(run?.status || '').toLowerCase();
  if (status === 'completed') return 'done';
  if (status === 'needs_user_decision') return index === 0 ? 'blocked' : 'pending';
  if ((run?.attempt_count || 0) > 0) return index === 0 ? 'done' : (index === 1 ? 'active' : 'pending');
  return index === 0 ? 'active' : 'pending';
}

function goProgramStepStatusLabel(status) {
  return {done: '完成', active: '進行中', blocked: '待確認', pending: '等待'}[status] || '等待';
}

function summarizeGoProgramPermissions(permissions = {}) {
  const labels = [];
  if (permissions.read_app_data) labels.push('讀取 app data');
  if (permissions.read_outputs) labels.push('讀取 outputs');
  if (permissions.write_outputs_scratch) labels.push('寫入 scratch outputs');
  if (permissions.read_db_as_json) labels.push('讀取系統提供 DB JSON');
  if (Array.isArray(permissions.read_mounted_paths) && permissions.read_mounted_paths.length) labels.push('讀取受限掛載路徑');
  if (permissions.network) labels.push('網路需審核');
  if (permissions.shell_subprocess) labels.push('子程序需審核');
  return labels.length ? labels.join('、') : '無額外權限';
}

function summarizeGoProgramDataSources(sources = []) {
  const labels = sources.map((source) => source?.name || source?.kind).filter(Boolean);
  return labels.length ? labels.join('、') : '由系統在執行時注入 JSON；尚未綁定外部資料來源';
}

function goProgramLifecycleLabel(run = {}) {
  if (run?.pending_skill_id) return 'Pending skill';
  return {completed: '完成', ready: '待製作', authoring: '製作中', needs_user_decision: '待確認'}[run?.status] || run?.status || '待命';
}

function GoProgramAuthoringCatalogList({showLabel = true}) {
  const t = useI18n(s => s.t);
  const [runs, setRuns] = useState([]);
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(false);
  const [note, setNote] = useState('');
  const [exportDialog, setExportDialog] = useState(null);

  const loadRuns = () => {
    setLoading(true);
    callWails(() => ListGoProgramAuthoringCatalog(20))
      .then((data) => setRuns(Array.isArray(data) ? data : []))
      .catch(() => setRuns([]))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadRuns();
    const offUpdated = EventsOn('goprogram:authoring_updated', loadRuns);
    const offNative = EventsOn('goprogram:native_completed', showExportDialog);
    const timer = setInterval(loadRuns, 12000);
    return () => {
      offUpdated && offUpdated();
      offNative && offNative();
      clearInterval(timer);
    };
  }, []);

  async function openDetail(runID) {
    try {
      const data = await callWails(() => GetGoProgramAuthoringDetail(runID));
      setDetail(data);
      setNote('');
    } catch (error) {
      setNote(error?.message || String(error));
    }
  }

  function showExportDialog(result) {
    if (!result || result.status !== 'success' || !result.landed_path) {
      if (result?.message) setNote(result.message);
      return;
    }
    setExportDialog({
      runID: result.run_id,
      name: result.program_name || result.program_id || 'Go program',
      tempExportDir: result.export_dir,
      landedPath: result.landed_path,
      detail: result.drop_target_kind && result.drop_target_dir
        ? `${result.landed_path}\n${result.drop_target_kind}: ${result.drop_target_dir}`
        : result.landed_path,
    });
  }

  async function startNativeExport(event, run) {
    event?.preventDefault?.();
    event?.stopPropagation?.();
    try {
      event?.dataTransfer?.setData?.('application/x-ai-console-go-program-run-id', run?.run_id || '');
      if (event?.dataTransfer) event.dataTransfer.effectAllowed = 'copy';
    } catch (_) {}
    if (!run?.run_id) return;
    setNote('');
    try {
      const result = await callWails(() => NativeDragExportGoProgramAuthoring(run.run_id));
      showExportDialog(result);
    } catch (error) {
      setNote(error?.message || String(error));
    }
  }

  async function finalizeExport(action) {
    if (!exportDialog) return;
    const target = exportDialog;
    setExportDialog(null);
    try {
      await callWails(() => FinalizeNativeGoProgramAuthoringExport(
        action,
        target.runID || '',
        target.tempExportDir || '',
        target.landedPath || '',
      ));
      if (action === 'remove') {
        setRuns((current) => current.filter((run) => run.run_id !== target.runID));
      }
    } catch (error) {
      setNote(error?.message || String(error));
    }
  }

  if ((loading && runs.length === 0) || !runs || runs.length === 0) return null;

  const selected = detail || null;
  const selectedPermissions = selected?.manifest?.permissions || {};
  const selectedSources = selected?.manifest?.data_sources || [];

  return (
    <>
        {showLabel && <ToolFlowSectionLabel>{t('tool.flowCategorySkill')}</ToolFlowSectionLabel>}
        {runs.map((run) => {
          const updated = run.updated_at ? new Date(run.updated_at).toLocaleString('zh-TW', {hour12: false}) : '';
          const title = run.program_name || run.program_id || run.run_id;
          return (
            <button
              key={run.run_id || title}
              type="button"
              className="tool-menu-item flow-asset-card go-program-flow-card"
              draggable
              data-full-title={run.message || run.purpose || title}
              title="點按檢視，拖出資料夾"
              onClick={() => openDetail(run.run_id)}
              onDragStart={(event) => startNativeExport(event, run)}
            >
              <span className="tool-menu-icon go-program-flow-icon">⌘</span>
              <strong>{title}</strong>
              <small>{goProgramLifecycleLabel(run)} · {run.attempt_count || 0}{updated ? ` · ${updated}` : ''}</small>
            </button>
          );
        })}
      {selected && (
        <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label="Go program authoring detail">
          <section className="tool-drag-modal go-program-flow-detail-modal">
            <header>
              <span>⌘</span>
              <div className="drag-action-title">
                <strong>{selected.program_name || selected.program_id}</strong>
                <small>{goProgramLifecycleLabel(selected)} · {selected.attempt_count || 0} attempts</small>
              </div>
            </header>
            <div className="go-program-flow-detail">
              <div className="go-program-flow-meta">
                <span>Run</span><strong>{selected.run_id}</strong>
                <span>Skill</span><strong>{selected.pending_skill_id || '尚未保存'}</strong>
                <span>Attempt</span><strong>{selected.attempt_count || 0} 次；{selected.latest_attempt_hash ? `版本 ${String(selected.latest_attempt_hash).slice(0, 12)}` : '尚無版本'}</strong>
              </div>
              <div className="go-program-dag" aria-label="Go program authoring DAG">
                {goProgramDagStepCatalog.map((step, index) => {
                  const status = goProgramStepStatus(selected, index);
                  return (
                    <div className={`go-program-dag-node go-program-dag-node-${status}`} key={step.id}>
                      <i>{index + 1}</i>
                      <div>
                        <strong>{step.title}</strong>
                        <small>{step.detail}</small>
                      </div>
                      <em>{goProgramStepStatusLabel(status)}</em>
                    </div>
                  );
                })}
              </div>
              <div className="go-program-flow-meta">
                <span>權限</span><strong>{summarizeGoProgramPermissions(selectedPermissions)}</strong>
                <span>資料來源</span><strong>{summarizeGoProgramDataSources(selectedSources)}</strong>
              </div>
            </div>
            <div className="tool-drag-actions" style={{gridTemplateColumns: 'repeat(2, minmax(0, 1fr))'}}>
              <button type="button" onClick={(event) => startNativeExport(event, selected)}>拖出</button>
              <button type="button" onClick={() => setDetail(null)}>關閉</button>
            </div>
          </section>
        </div>
      )}
      {note && <p className="go-program-flow-note">{note}</p>}
      {exportDialog && (
        <DragActionModal
          ariaLabel="Go program export"
          icon="↗"
          title={exportDialog.name}
          detail={exportDialog.detail}
          actions={[
            {label: t('adapter.remove'), onClick: () => finalizeExport('remove')},
            {label: t('adapter.copyAction'), onClick: () => finalizeExport('copy')},
            {label: t('common.cancel'), onClick: () => finalizeExport('cancel')},
          ]}
        />
      )}
    </>
  );
}

/* i18n: tool popup */
function ToolPopup({side, tools, activeTab, favoriteToolIds, hiddenToolIds, toolResult, externalServiceLinks, documentationLinks, dagRun, onTabChange, onToolActivate, onToolDragStart, onToolDragOut, onReorder}) {
  const t = useI18n(s => s.t);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const normalizedQuery = searchQuery.trim().toLowerCase();
  const visibleTools = tools.filter((tool) => (
    tool.id !== 'tool-entrance' &&
    (side !== 'right' || favoriteToolIds.includes(tool.id)) &&
    !hiddenToolIds.includes(tool.id) &&
    toolTabFor(tool) === activeTab
  )).filter((tool) => {
    if (!normalizedQuery) return true;
    return [tool.title, tool.detail, tool.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalizedQuery));
  });
  const visibleExternalServiceLinks = (externalServiceLinks || []).filter((link) => !hiddenToolIds.includes(link.id));
  const visibleDocumentationLinks = (documentationLinks || []).filter((link) => !hiddenToolIds.includes(link.id));

  function externalLinkTool(link, icon, detail) {
    return {
      id: link.id,
      icon,
      title: link.label || link.url,
      detail: link.url || detail,
      kind: 'external',
      target: link.url,
      enabled: true,
      available: true,
    };
  }

  function startToolDrag(event, tool) {
    event.dataTransfer.effectAllowed = 'copyMove';
    event.dataTransfer.setData('application/x-ai-console-tool-id', tool.id);
    onToolDragStart({tool, source: side});
  }

  function finishToolDrag(event, tool) {
    const leftWindow =
      event.clientX <= 0 ||
      event.clientY <= 0 ||
      event.clientX >= window.innerWidth ||
      event.clientY >= window.innerHeight;
    const droppedOutside = leftWindow || (event.clientX === 0 && event.clientY === 0);
    if (droppedOutside && event.dataTransfer.dropEffect !== 'move') {
      onToolDragOut({tool, source: side});
    }
    onToolDragStart(null);
  }

  return (
    <section className={`tools-popup tools-popup-${side}`} aria-label={t('tool.panelLabel')}>
      <header className="tools-popup-tabs">
        {getToolTabOptions().map((tab) => (
          <button
            className={`tools-popup-tab ${activeTab === tab.id ? 'tools-popup-tab-active' : ''}`}
            type="button"
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
          >
            {tab.label}
          </button>
        ))}
        <button
          className={`tools-popup-search ${searchOpen ? 'tools-popup-search-active' : ''}`}
          type="button"
          aria-expanded={searchOpen}
          aria-label={t('tool.searchLabel')}
          onClick={() => setSearchOpen((open) => !open)}
        >
          ⌕
        </button>
      </header>
      {searchOpen && (
        <div className="tools-search-popover" role="search">
          <input
            autoFocus
            aria-label={t('tool.searchNameLabel')}
            placeholder={t('tool.searchPlaceholder')}
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Escape') {
                setSearchQuery('');
                setSearchOpen(false);
              }
            }}
          />
          {searchQuery && (
            <button type="button" aria-label={t('tool.clearSearch')} onClick={() => setSearchQuery('')}>×</button>
          )}
        </div>
      )}
        {/* I-6 (#I-601): tool-menu-list 同時作為 intra-list 拖曳排序的 drop container。
           拖入同列表另一項時，根據 drop 位置計算 newRank 並呼叫 onReorder。 */}
      <div
        className="tool-menu-list"
        onDragOver={(event) => {
          if (event.dataTransfer.types.includes('application/x-ai-console-tool-id')) {
            event.preventDefault();
            event.dataTransfer.dropEffect = 'move';
          }
        }}
        onDrop={(event) => {
          const droppedId = event.dataTransfer.getData('application/x-ai-console-tool-id');
          if (!droppedId || !onReorder) return;
          event.preventDefault();
          // 根據滑鼠 Y 位置找到最近的 tool-menu-item 計算 newRank
          const items = Array.from(event.currentTarget.querySelectorAll('[data-tool-id]'));
          let targetIndex = items.length; // 預設放最後
          for (let i = 0; i < items.length; i++) {
            const rect = items[i].getBoundingClientRect();
            if (event.clientY < rect.top + rect.height / 2) {
              targetIndex = i;
              break;
            }
          }
          onReorder(droppedId, targetIndex);
        }}
      >
        {/* I-7: 自動流程 tab mirrors the active DAG Hook Run and node summaries. */}
        {activeTab === 'flow' && dagRun && (
          <button
            type="button"
            className="tool-menu-item flow-asset-card"
            title={`${dagRun.title} · ${dagStatusLabel(dagRun.status)}`}
          >
            <span className="tool-menu-icon">◇</span>
            <strong>{dagRun.title}</strong>
            <small>{dagStatusLabel(dagRun.status)} · {dagRun.nodes?.length || 0}</small>
          </button>
        )}
        {/* Recording assets live here; DAG run history stays out of this list. */}
        {activeTab === 'flow' && (
          <RecordingCatalogList />
        )}
        {activeTab === 'flow' && visibleTools.length > 0 && (
          <ToolFlowSectionLabel>{t('tool.flowCategorySkill')}</ToolFlowSectionLabel>
        )}
        {activeTab === 'flow' && (
          <GoProgramAuthoringCatalogList showLabel={visibleTools.length === 0} />
        )}
        {/* ── #44 工具清單：可用工具在前、斷線工具在後 ── */}
        {/* unavailable 工具：icon 加黑線打叉覆蓋，保持可見，不灰化 */}
        {visibleTools.map((tool) => {
          const isUnavailable = tool.available === false || tool.needs_reauth || !tool.enabled;
          return (
            <button
              className={`tool-menu-item ${isUnavailable ? 'tool-menu-item-unavailable' : ''}`}
              draggable
              type="button"
              key={tool.id || tool.title}
              data-tool-id={tool.id}
              onClick={() => onToolActivate(tool.id)}
              onDragEnd={(event) => finishToolDrag(event, tool)}
              onDragStart={(event) => startToolDrag(event, tool)}
              title={isUnavailable
                ? `${tool.title} — ${tool.unavailable_reason || t('tool.unavailableReason')}`
                : tool.title}
            >
              <span className="tool-menu-icon" style={{position: 'relative'}}>
                {tool.icon}
                {/* #44: 黑線打叉覆蓋（取代舊的 lightning-fracture） */}
                {isUnavailable && (
                  <span className="tool-unavailable-x" aria-label={t('tool.unavailableAriaLabel')} style={{
                    position: 'absolute', top: 0, left: 0, right: 0, bottom: 0,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: '1.4em', color: '#1a1a1a', fontWeight: 'bold',
                    pointerEvents: 'none', lineHeight: 1,
                  }}>✕</span>
                )}
              </span>
              <strong>{tool.title}</strong>
              <small>{isUnavailable ? (tool.unavailable_reason || t('tool.disconnected')) : tool.detail}</small>
            </button>
          );
        })}
        {/* I-5: external_service 連結 — 來自 ListExternalLinksByType("external_service")，顯示於「外部連結」tab */}
        {activeTab === 'external' && visibleExternalServiceLinks.map((link) => {
          const linkTool = externalLinkTool(link, '↗', t('link.externalService'));
          return (
            <button
              className="tool-menu-item tool-menu-item-link"
              key={link.id}
              draggable
              type="button"
              data-tool-id={link.id}
              title={link.url}
              onClick={() => openExternal(link.url)}
              onDragEnd={(event) => finishToolDrag(event, linkTool)}
              onDragStart={(event) => startToolDrag(event, linkTool)}
            >
              <span className="tool-menu-icon">↗</span>
              <strong>{link.label || link.url}</strong>
              <small>{t('link.externalService')}</small>
            </button>
          );
        })}
        {/* I-5: documentation 連結 — 純參考，不進工具執行區。
             必須用 Wails BrowserOpenURL 在 OS 預設瀏覽器外部開啟，
             不得在 AI Console 內部 WebView 開啟（#47 合約要求）。 */}
        {activeTab === 'external' && visibleDocumentationLinks.map((link) => {
          const linkTool = externalLinkTool(link, '▤', t('link.docLinkLabel'));
          return (
            <button
              className="tool-menu-item tool-menu-item-link tool-menu-item-doc"
              key={link.id}
              draggable
              type="button"
              data-tool-id={link.id}
              title={t('link.docLinkTitle', { url: link.url })}
              onClick={() => openExternal(link.url)}
              onDragEnd={(event) => finishToolDrag(event, linkTool)}
              onDragStart={(event) => startToolDrag(event, linkTool)}
            >
              <span className="tool-menu-icon">▤</span>
              <strong>{link.label || link.url}</strong>
              <small>{t('link.docLinkLabel')}</small>
            </button>
          );
        })}
        {!visibleTools.length && activeTab !== 'flow' && !(activeTab === 'external' && (visibleExternalServiceLinks.length || visibleDocumentationLinks.length)) && (
          <div className="tool-search-empty">
            <strong>{t('tool.noMatch')}</strong>
            <span>{t('tool.noMatchHint')}</span>
          </div>
        )}
      </div>
      {toolResult?.message && (
        <p className={toolResult.ok ? 'tool-action-note tool-action-note-ok' : 'tool-action-note'}>
          {toolResult.message}
        </p>
      )}
    </section>
  );
}

function DragActionModal({ariaLabel, icon = '↗', title, detail = '', actions = []}) {
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={ariaLabel}>
      <section className="tool-drag-modal">
        <header>
          <span>{icon}</span>
          <div className="drag-action-title">
            <strong>{title}</strong>
            {detail && <small>{detail}</small>}
          </div>
        </header>
        <div
          className="tool-drag-actions"
          style={{gridTemplateColumns: `repeat(${Math.max(1, actions.length)}, minmax(0, 1fr))`}}
        >
          {actions.map((action) => (
            <button
              className={action.disabled ? 'tool-drag-action-disabled' : ''}
              type="button"
              disabled={action.disabled}
              key={action.label}
              onClick={action.onClick}
            >
              {action.label}
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}

function LearningReplayStartConfirmCard({pending, onConfirm, onCancel}) {
  const plan = pending?.plan || {};
  const steps = Array.isArray(plan.steps) ? plan.steps : [];
  const nativeSteps = steps.filter(isNativeReplayStep);
  const domSteps = steps.length - nativeSteps.length;
  const windows = Array.from(new Set(nativeSteps.map((step) => (
    basenameForDisplay(step.window_process || step.tag || step.window_title || 'unknown')
  )))).filter(Boolean).slice(0, 3);
  const first = steps[0] || {};
  const target = first.window_title || first.label || first.role || first.css_selector || first.tag || `座標 (${first.x || 0}, ${first.y || 0})`;
  return (
    <div className="learning-replay-confirm" role="dialog" aria-label="Replay start confirmation">
      <header>
        <strong>確認 replay plan</strong>
        <span>尚未執行</span>
      </header>
      <p>我已讀到示範並產生安全 plan。確認下面內容正確後，才會真的移動滑鼠或點擊。</p>
      <div className="learning-replay-confirm-grid">
        <span>Run</span><strong>{plan.run_id || 'unknown'}</strong>
        <span>Title</span><strong>{plan.title || plan.run_name || 'untitled'}</strong>
        <span>Steps</span><strong>{steps.length}（本視窗 {domSteps}，外部 {nativeSteps.length}）</strong>
        <span>First</span><strong>{target}</strong>
        {windows.length > 0 && <><span>Windows</span><strong>{windows.join(', ')}</strong></>}
      </div>
      <footer>
        <button type="button" onClick={onCancel}>先不要</button>
        <button type="button" className="learning-replay-confirm-primary" onClick={onConfirm}>執行 replay</button>
      </footer>
    </div>
  );
}

function LearningReplayConfirmCard({pending, onConfirm, onCancel}) {
  const step = pending?.result?.steps?.[pending?.stoppedStepOffset] || {};
  const label = step.label || step.window_title || step.window_process || 'native-window';
  return (
    <div className="learning-replay-confirm" role="dialog" aria-label="Replay confirmation">
      <header>
        <strong>確認這一步</strong>
        <span>OpenCV fallback</span>
      </header>
      <p>已把游標移到候選位置。確認後會點擊這一步並繼續剩餘 replay。</p>
      <div className="learning-replay-confirm-grid">
        <span>Step</span><strong>{step.index || pending?.stoppedStepOffset + 1}</strong>
        <span>Target</span><strong>{label}</strong>
        <span>Point</span><strong>{Math.round(Number(step.x || 0))}, {Math.round(Number(step.y || 0))}</strong>
      </div>
      {step.error && <small>{step.error}</small>}
      <footer>
        <button type="button" onClick={onCancel}>取消</button>
        <button type="button" className="learning-replay-confirm-primary" onClick={onConfirm}>確認繼續</button>
      </footer>
    </div>
  );
}

function ToolCopyConfirmModal({tool, onCancel, onConfirm}) {
  const t = useI18n(s => s.t);
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('tool.confirmInstallLabel')}>
      <section className="tool-copy-modal">
        <p>{t('tool.confirmInstallQuestion', { title: tool.title, id: tool.id })}</p>
        <div>
          <button type="button" onClick={onConfirm}>{t('common.yes')}</button>
          <button type="button" onClick={onCancel}>{t('common.no')}</button>
        </div>
      </section>
    </div>
  );
}

// I-5 (#I-502): 兩段式流程 — 先 Preview 顯示 link_type，確認後才 Register。
// linkPreview 為 null 表示尚未預覽；有值且 valid 時顯示確認按鈕。
function ReferenceLinkModal({value, linkPreview, error, suggestions, onCancel, onChange, onUseSuggestion, onPreview, onConfirm}) {
  const {t} = useI18n();
  const isOllamaModelLibraryPreview = linkPreview?.link_type === 'adapter_candidate' &&
    /ollama|模型庫|本機模型|blobs|manifests|\.ollama[\\/]+models/i.test(`${value || ''} ${linkPreview?.url || ''} ${linkPreview?.reason || ''}`);
  const getLinkTypeLabel = () => ({
    external_service: t('link.externalService'),
    adapter_candidate: isOllamaModelLibraryPreview ? '本地模型 Adapter' : 'CLI Adapter',
    llm_provider_candidate: t('link.llmApiInterface'),
    documentation: t('link.docLinkShort'),
    unsupported: t('link.unsupported'),
  });
  const linkTypeLabel = getLinkTypeLabel();
  // i18n: link
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('link.addRefLinkLabel')}>
      <section className="reference-link-modal">
        <label>
          <span>{t('link.refLinkSpan')}</span>
          <div>
            <input
              autoFocus
              placeholder={t('link.inputPlaceholder')}
              value={value}
              onChange={(event) => onChange(event.target.value)}
            />
            {!linkPreview ? (
              <button type="button" onClick={onPreview}>{t('link.preview')}</button>
            ) : (
              <button type="button" onClick={onConfirm}>{t('link.confirm')}</button>
            )}
          </div>
          <small className="reference-link-hint">
            {t('link.inputHint')}
          </small>
        </label>
        {linkPreview?.valid && linkPreview.type === 'remote_bridge' && (
          <p className="reference-link-preview reference-link-bridge">
            {t('link.createCommsQuestion')} <strong>{linkPreview.hint_label}</strong>
            <small>{linkPreview.setup_required ? t('link.setupRequired') : t('link.testAndName')}</small>
          </p>
        )}
        {linkPreview?.valid && linkPreview.type !== 'remote_bridge' && (
          <p className="reference-link-preview">
            {t('link.typeLabel')}<strong>{linkTypeLabel[linkPreview.link_type] || linkPreview.link_type}</strong>
            {linkPreview.link_type === 'llm_provider_candidate' && (
              <small>
                {t('link.detectedProvider', { name: linkPreview.provider_name || 'LLM/API Provider', baseUrl: linkPreview.base_url ? t('link.defaultApi', { url: linkPreview.base_url }) : '' })}
              </small>
            )}
            {linkPreview.link_type === 'documentation' && <small>{t('link.docOnlyRef')}</small>}
            {linkPreview.link_type === 'adapter_candidate' && linkPreview.reason && <small>（{linkPreview.reason}）</small>}
          </p>
        )}
        {error && <p className="reference-link-error">{error}</p>}
        {suggestions?.length > 0 && (
          <div className="reference-link-suggestions">
            <strong>{t('link.suggestionsTitle')}</strong>
            {suggestions.map((item) => (
              <button
                type="button"
                key={`${item.name}-${item.path}`}
                onClick={() => onUseSuggestion(item.path)}
                title={item.path}
              >
                <span>{item.detected ? t('link.foundItem', { name: item.name }) : t('link.exampleItem', { name: item.name })}</span>
                <code>{item.path}</code>
              </button>
            ))}
          </div>
        )}
        <button className="reference-link-cancel" type="button" onClick={onCancel}>{t('common.cancel')}</button>
      </section>
    </div>
  );
}

function LLMAPISetupModal({setup, guideStep, onGuideStepChange, onChange, onCancel, onTest, onSubmit}) {
  const {t} = useI18n();
  const update = (field, value) => onChange({...setup, [field]: value});
  const guide = [
    {
      title: t('llmSetup.getApiKeyTitle'),
      body: t('llmSetup.getApiKeyBody', { providerName: setup.providerName || 'Provider' }),
      action: t('llmSetup.getApiKeyAction'),
      url: setup.apiKeyURL,
    },
    {
      title: t('llmSetup.confirmBaseUrlTitle'),
      body: t('llmSetup.confirmBaseUrlBody'),
      action: t('llmSetup.confirmBaseUrlAction'),
      url: setup.docsURL,
    },
    {
      title: t('llmSetup.fillModelTitle'),
      body: t('llmSetup.fillModelBody'),
      action: t('llmSetup.fillModelAction'),
      url: setup.docsURL,
    },
  ];
  const currentGuide = guide[Math.min(guideStep || 0, guide.length - 1)];
  const openHelp = (field) => {
    if (field === 'apiKey' && setup.apiKeyURL) return openExternal(setup.apiKeyURL);
    if (setup.docsURL) return openExternal(setup.docsURL);
    openExternal(`https://www.google.com/search?q=${encodeURIComponent(`${setup.providerName || 'LLM API'} ${field} official docs`)}`);
  };
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('llmSetup.setupTitle')}>
      <section className="reference-link-modal llm-api-setup-modal">
        <header className="llm-api-setup-head">
          <span>{t('llmSetup.apiSetupTitle')}</span>
          <strong>{setup.providerName || 'LLM API'}</strong>
        </header>
        <div className="rb-guide-card llm-api-guide-card">
          <span className="rb-guide-count">{t('remoteBridgeSetup.stepCount', { current: (guideStep || 0) + 1, total: guide.length })}</span>
          <strong>{currentGuide.title}</strong>
          <p>{currentGuide.body}</p>
          <div className="rb-guide-actions">
            {currentGuide.url && (
              <button type="button" onClick={() => openExternal(currentGuide.url)}>
                {currentGuide.action}
              </button>
            )}
            {guideStep < guide.length - 1 ? (
              <button type="button" onClick={() => onGuideStepChange(Math.min((guideStep || 0) + 1, guide.length - 1))}>{t('remoteBridgeSetup.next')}</button>
            ) : (
              <button type="button" onClick={() => onGuideStepChange(0)}>{t('remoteBridgeSetup.restartGuide')}</button>
            )}
          </div>
        </div>
        <label className="rb-field">
          <span className="rb-field-title">
            <span>API Key</span>
            <button className="rb-help-btn" type="button" onClick={() => openHelp('apiKey')}>?</button>
          </span>
          <input type="password" value={setup.apiKey || ''} onChange={(event) => update('apiKey', event.target.value)} placeholder="sk-..." />
        </label>
        <label className="rb-field">
          <span className="rb-field-title">
            <span>Base URL</span>
            <button className="rb-help-btn" type="button" onClick={() => openHelp('baseURL')}>?</button>
          </span>
          <input type="text" value={setup.baseURL || ''} onChange={(event) => update('baseURL', event.target.value)} placeholder="https://api.example.com/v1" />
        </label>
        <label className="rb-field">
          <span className="rb-field-title">
            <span>Model</span>
            <button className="rb-help-btn" type="button" onClick={() => openHelp('model')}>?</button>
          </span>
          <input type="text" value={setup.model || ''} onChange={(event) => update('model', event.target.value)} placeholder="model id" />
        </label>
        <small className="reference-link-hint">{t('llmSetup.nameHint')}</small>
        <div className="rb-actions">
          <button type="button" onClick={onTest}>{t('llmSetup.testConnection')}</button>
          <button type="button" onClick={onSubmit}>{t('llmSetup.save')}</button>
          <button type="button" className="rb-cancel-btn" onClick={onCancel}>{t('common.cancel')}</button>
        </div>
      </section>
    </div>
  );
}

function AdapterRenameModal({target, draft, onDraftChange, onCancel, onSave}) {
  const t = useI18n(s => s.t);
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('llmSetup.renameAdapterLabel')}>
      <section className="reference-link-modal adapter-rename-modal">
        <h4>{t('llmSetup.renameAdapterTitle')}</h4>
        <input
          className="remote-bridge-name-input"
          autoFocus
          value={draft}
          onChange={(event) => onDraftChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') onSave();
            if (event.key === 'Escape') onCancel();
          }}
          placeholder={target?.name || 'LLM API'}
        />
        <div className="rb-actions">
          <button type="button" onClick={onSave}>{t('llmSetup.confirmRename')}</button>
          <button type="button" className="rb-cancel-btn" onClick={onCancel}>{t('llmSetup.renameLater')}</button>
        </div>
      </section>
    </div>
  );
}

function WebSearchSetupModal({setup, config, error, onChange, onCancel, onClear, onSubmit}) {
  const options = Array.isArray(setup?.options) && setup.options.length ? setup.options : defaultWebSearchProviderOptions();
  const selected = options.find((option) => option.id === setup.providerId) || options[0];
  const fields = Array.isArray(selected?.fields) ? selected.fields : ['api_key'];
  const update = (key, value) => onChange({...setup, [key]: value});
  const chooseProvider = (providerId) => onChange({
    ...setup,
    providerId,
    apiKey: '',
    cx: '',
  });
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label="Web search setup">
      <section className="reference-link-modal web-search-setup-modal">
        <h4>內建網路搜尋</h4>
        <p className="reference-link-hint">
          選擇搜尋供應商並輸入 API 欄位。金鑰會寫入 Windows DPAPI 加密 credential store，不會交給 Agent 或顯示在提示詞中。
        </p>
        <div className="web-search-provider-grid">
          {options.map((option) => (
            <button
              key={option.id}
              type="button"
              className={option.id === setup.providerId ? 'web-search-provider-card web-search-provider-active' : 'web-search-provider-card'}
              onClick={() => chooseProvider(option.id)}
            >
              <strong>{option.name}</strong>
              <small>{option.free_tier_hint || option.id}</small>
            </button>
          ))}
        </div>
        <label>
          <span>API Key</span>
          <input
            type="password"
            value={setup.apiKey || ''}
            onChange={(event) => update('apiKey', event.target.value)}
            placeholder={selected?.id === 'tavily' ? 'tvly-...' : 'API key'}
          />
        </label>
        {fields.includes('cx') && (
          <label>
            <span>Search Engine ID (cx)</span>
            <input
              type="text"
              value={setup.cx || ''}
              onChange={(event) => update('cx', event.target.value)}
              placeholder="Programmable Search Engine ID"
            />
          </label>
        )}
        {config?.configured && (
          <small className="reference-link-hint">
            目前已設定：{config.provider || config.provider_id}
          </small>
        )}
        {error && <div className="rb-error">{error}</div>}
        <div className="rb-actions">
          {selected?.docs_url && (
            <button type="button" onClick={() => openExternal(selected.docs_url)}>官方文件</button>
          )}
          <button type="button" onClick={onSubmit}>儲存</button>
          <button type="button" onClick={onClear}>清除設定</button>
          <button type="button" className="rb-cancel-btn" onClick={onCancel}>取消</button>
        </div>
      </section>
    </div>
  );
}

// I-6 (#I-602): Reauth Intercept Dialog
// 工具 unavailable（lightning-fracture 狀態）時，點擊執行會被攔截到此 dialog。
// 用戶可選「重試」（強制呼叫 ActivateTool）或「取消」。
// 後續可擴充為真正的 OAuth re-auth 流程入口。
function ReauthInterceptDialog({tool, onRetry, onCancel}) {
  const t = useI18n(s => s.t);
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('reauth.title')}>
      <section className="reauth-dialog">
        <header>
          <span className="reauth-dialog-icon">⚡</span>
          <strong>{tool.title}</strong>
        </header>
        <p>{t('reauth.disconnected')}</p>
        <p className="reauth-dialog-hint">{t('reauth.hint')}</p>
        <div className="reauth-dialog-actions">
          <button type="button" onClick={onRetry}>{t('reauth.retry')}</button>
          <button type="button" onClick={onCancel}>{t('common.cancel')}</button>
        </div>
      </section>
    </div>
  );
}

// #I-805: Sidecar 崩潰恢復卡片 — 顯示錯誤訊息 + 重啟按鈕
/* i18n: sidecar recovery card */
function StopRecoveryCard({error, onRestart}) {
  const t = useI18n(s => s.t);
  return (
    <div className="stop-recovery-overlay" role="alert" aria-label={t('system.sidecarInterrupt')}>
      <section className="stop-recovery-card">
        <header className="stop-recovery-header">
          <span className="stop-recovery-icon">⚠</span>
          <strong>{t('system.sidecarInterrupt')}</strong>
        </header>
        <p className="stop-recovery-msg">{error || t('system.sidecarInterruptDetail')}</p>
        <div className="stop-recovery-actions">
          <button type="button" className="stop-recovery-restart" onClick={onRestart}>{t('system.sidecarRestart')}</button>
        </div>
      </section>
    </div>
  );
}

// #I-1001: 自動封存 Toast — 右下角輕量提示，5 秒後自動消失
function DigestAutoArchiveToast({message, onDismiss}) {
  return (
    <div className="digest-toast" role="status" aria-live="polite">
      <span className="digest-toast-icon">📦</span>
      <span className="digest-toast-msg">{message}</span>
      <button type="button" className="digest-toast-close" onClick={onDismiss}>×</button>
    </div>
  );
}

function SettingsMenu({
  panel, onPanelChange, voiceState, voiceInstallBusy,
  onVoiceSettingsChange, onVoiceSettingsRefresh, onVoiceModelInstall, onVoiceModelRemove, onRestoreDefaults,
}) {
  const t = useI18n(s => s.t);
  const activePanelStyle = normalizePanelStyle(panel.panelStyle);
  const panelLanguageValue = localizeBackendLabel(panel.panelLanguage, _panelLangLabelMap);
  const roleLanguageValue = localizeBackendLabel(panel.roleLanguage, _roleLangLabelMap);
  const fontPresetValue = localizeBackendLabel(panel.fontPreset, _fontPresetLabelMap);
  const [orderedStyleOptions, setOrderedStyleOptions] = useState(loadStyleOptionOrder);
  const [draggedStyleOption, setDraggedStyleOption] = useState('');
  const voiceSettings = voiceState?.settings || {languageMode: 'auto', manualLanguage: '', debugMode: false, commandMode: false};

  function moveStyleOption(targetOption) {
    if (!draggedStyleOption || draggedStyleOption === targetOption) return;
    setOrderedStyleOptions((current) => {
      const next = reorderItems(current, draggedStyleOption, targetOption);
      saveStyleOptionOrder(next);
      return next;
    });
    setDraggedStyleOption('');
  }

  return (
    <section className="settings-side-panel" aria-label={t('settings.panelSettings')}>
      <h2>{t('settings.panelSettings')}</h2>
      <div className="settings-menu-list">
        <SettingSelect
          icon="◎"
          label={t('settings.panelLanguage')}
          value={panelLanguageValue}
          onNext={() => onPanelChange({panelLanguage: cycleValue([t('settings.langZhTW'), t('settings.langEn'), t('settings.langJa'), t('settings.langPt'), t('settings.langEs'), t('settings.langTh')], panelLanguageValue)})}
        />
        <SettingSelect
          icon="♙"
          label={t('settings.roleLanguage')}
          value={roleLanguageValue}
          onNext={() => onPanelChange({roleLanguage: cycleValue([t('settings.roleLangAuto'), t('settings.langZhTW'), t('settings.langEn')], roleLanguageValue)})}
        />
        <SettingSelect
          icon="Aa"
          label={t('settings.fontPreset')}
          value={fontPresetValue}
          onNext={() => onPanelChange({fontPreset: cycleValue([t('settings.fontDefault'), t('settings.fontRound'), t('settings.fontMono')], fontPresetValue)})}
        />
        <SettingSelect
          icon="Tt"
          label={t('settings.fontSize')}
          value={panel.fontScale}
          onNext={() => onPanelChange({fontScale: cycleValue(fontScaleOptions, panel.fontScale)})}
        />
      </div>
      <div className="settings-style-block">
        <div className="settings-style-label"><span>◌</span><span>{t('settings.panelStyle')}</span></div>
        <div className="settings-segments">
          {orderedStyleOptions.map((option) => (
            <button
              className={activePanelStyle === option ? 'segment-active' : ''}
              draggable
              type="button"
              key={option}
              onDragEnd={() => setDraggedStyleOption('')}
              onDragOver={(event) => event.preventDefault()}
              onDragStart={() => setDraggedStyleOption(option)}
              onDrop={() => moveStyleOption(option)}
              onClick={() => onPanelChange({panelStyle: option})}
            >
              {styleLabel(option)}
            </button>
          ))}
        </div>
      </div>
      <div className="settings-voice-block">
        <div className="settings-voice-head">
          <span>{t('settings.localVoice')}</span>
          <button type="button" onClick={onVoiceSettingsRefresh}>{t('settings.voiceCheck')}</button>
        </div>
        <div className="settings-voice-state">
          <span className={`settings-voice-dot ${voiceState?.status === 'ready' ? 'settings-voice-ready' : ''}`} />
          <strong>{voiceStatusLabel(voiceState?.status)}</strong>
          <small>{voiceState?.language ? t('settings.voiceLang', { language: voiceState.language }) : t('settings.voiceFollowApp')}</small>
        </div>
        <div className="settings-voice-controls">
          <button
            type="button"
            className={voiceSettings.languageMode === 'auto' ? 'settings-voice-toggle settings-voice-toggle-on' : 'settings-voice-toggle'}
            onClick={() => onVoiceSettingsChange?.({languageMode: cycleValue(['auto', 'follow_app', 'manual'], voiceSettings.languageMode || 'auto')})}
          >
            {voiceLanguageModeLabel(voiceSettings.languageMode)}
          </button>
          <button
            type="button"
            className="settings-voice-toggle"
            disabled={voiceInstallBusy}
            onClick={voiceState?.modelAvailable ? onVoiceModelRemove : onVoiceModelInstall}
          >
            {voiceInstallBusy ? t('settings.voiceProcessing') : voiceState?.modelAvailable ? t('settings.voiceRemoveModel') : t('settings.voiceInstallModel')}
          </button>
          <button
            type="button"
            className={voiceSettings.debugMode ? 'settings-voice-toggle settings-voice-toggle-on' : 'settings-voice-toggle'}
            onClick={() => onVoiceSettingsChange?.({debugMode: !voiceSettings.debugMode})}
          >
            debug {voiceSettings.debugMode ? 'on' : 'off'}
          </button>
          <button
            type="button"
            className={voiceSettings.commandMode ? 'settings-voice-toggle settings-voice-toggle-on' : 'settings-voice-toggle'}
            onClick={() => onVoiceSettingsChange?.({commandMode: !voiceSettings.commandMode})}
          >
            {t('settings.command')} {voiceSettings.commandMode ? 'on' : 'off'}
          </button>
        </div>
      </div>
      <div className="settings-restore-block">
        <button className="settings-restore-btn" type="button" onClick={onRestoreDefaults}>
          <span>↺</span>
          <span>{t('settings.resetDefault')}</span>
        </button>
        <small className="settings-restore-note">{t('settings.resetHint')}</small>
      </div>
    </section>
  );
}

function SettingsWorkspace(props) {
  const t = useI18n(s => s.t);
  const {
    trustDomClickEnabled, activeOverrides, trustedSessionActive, deviceProfile,
    onToggleTrustDomClick, onEnableOverride, onEnableWorkflowTrust,
    onDisableOverride, onEnableTrustedSession, onLoadDeviceProfile,
    browserPref, panelSettings, adapterList, summaryModelSettings, summaryModelScan,
    onSummaryModelChange, onSummaryModelRescan, voiceState, voiceInstallBusy,
    onVoiceSettingsChange, onVoiceSettingsRefresh, onVoiceModelInstall, onVoiceModelRemove,
    onVoiceDebugClear, onSaveBrowserPref, onPreviewStyleDiff,
    ...personaProps
  } = props;

  return (
    <main className="settings-workspace" aria-label={t('settings.settingsWorkspace')}>
      <PersonaSettingsDrawer {...personaProps}/>
      <ControlledTrustSettings
        trustDomClickEnabled={trustDomClickEnabled}
        activeOverrides={activeOverrides}
        trustedSessionActive={trustedSessionActive}
        deviceProfile={deviceProfile}
        onToggleTrustDomClick={onToggleTrustDomClick}
        onEnableOverride={onEnableOverride}
        onEnableWorkflowTrust={onEnableWorkflowTrust}
        onDisableOverride={onDisableOverride}
        onEnableTrustedSession={onEnableTrustedSession}
        onLoadDeviceProfile={onLoadDeviceProfile}
      />
      <BrowserSettingsSection
        browserPref={browserPref}
        panelSettings={panelSettings}
        adapterList={adapterList}
        summaryModelSettings={summaryModelSettings}
        summaryModelScan={summaryModelScan}
        onSummaryModelChange={onSummaryModelChange}
        onSummaryModelRescan={onSummaryModelRescan}
        voiceState={voiceState}
        voiceInstallBusy={voiceInstallBusy}
        onVoiceSettingsChange={onVoiceSettingsChange}
        onVoiceSettingsRefresh={onVoiceSettingsRefresh}
        onVoiceModelInstall={onVoiceModelInstall}
        onVoiceModelRemove={onVoiceModelRemove}
        onVoiceDebugClear={onVoiceDebugClear}
        onSaveBrowserPref={onSaveBrowserPref}
        onPreviewStyleDiff={onPreviewStyleDiff}
      />
      <StateTokenLegend />
      {/* #I-207: Skill Context Orchestration 靜態說明（Settings 頁常駐） */}
      <SkillContextSettingsSection />
    </main>
  );
}

class SettingsErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = {error: null};
  }

  static getDerivedStateFromError(error) {
    return {error};
  }

  componentDidCatch(error, info) {
    console.error('[Settings render]', error, info);
  }

  render() {
    if (!this.state.error) return this.props.children;
    const message = this.state.error?.stack || this.state.error?.message || String(this.state.error);
    return (
      <main className="settings-workspace settings-error-panel" aria-label="設定錯誤">
        <section className="settings-error-card">
          <h2>設定載入失敗</h2>
          <pre>{message}</pre>
          <button type="button" onClick={this.props.onClose}>關閉設定</button>
        </section>
      </main>
    );
  }
}

// ---------------------------------------------------------------------------
// I-4: Browser & UI Settings Section
// ---------------------------------------------------------------------------
function BrowserSettingsSection({
  browserPref, panelSettings, adapterList, summaryModelSettings, summaryModelScan,
  onSummaryModelChange, onSummaryModelRescan, voiceState, voiceInstallBusy,
  onVoiceSettingsChange, onVoiceSettingsRefresh, onVoiceModelInstall, onVoiceModelRemove,
  onVoiceDebugClear, onSaveBrowserPref, onPreviewStyleDiff,
}) {
  const t = useI18n(s => s.t);
  const browserOptions = ['chrome', 'safari', 'firefox', 'edge', 'arc'];
  const [diffInput, setDiffInput] = useState('');
  const summarySettings = summaryModelSettings || {source: 'cli_adapter', modelId: 'current', endpoint: '', alwaysUse: false};
  const voiceSettings = voiceState?.settings || {languageMode: 'auto', manualLanguage: '', debugMode: false, commandMode: false, whisperBinPath: '', modelPath: ''};
  const localOptions = summaryModelScan?.options || [];

  return (
    <div className="browser-settings-section">
      <h4>{t('settings.browserUISettings')}</h4>

      {/* Browser Preference Selector */}
      <div className="browser-pref-row">
        <label htmlFor="browser-select">{t('settings.defaultBrowser')}</label>
        <select
          id="browser-select"
          value={browserPref?.id || ''}
          onChange={(e) => onSaveBrowserPref(e.target.value)}
        >
          <option value="" disabled>{t('settings.selectBrowser')}</option>
          {browserOptions.map((b) => (
            <option key={b} value={b}>{b.charAt(0).toUpperCase() + b.slice(1)}</option>
          ))}
        </select>
      </div>

      <div className="ui-settings-info">
        <span className="ui-settings-label">{t('settings.currentPanel')}</span>
        <span className="ui-settings-value">{styleLabel(panelSettings?.panelStyle)}</span>
      </div>

      <section className="summary-model-settings">
        <h4>{t('settings.summaryModelSettings')}</h4>
        <div className="browser-pref-row">
          <label htmlFor="summary-source">{t('settings.modelSource')}</label>
          <select
            id="summary-source"
            value={summarySettings.source}
            onChange={(e) => onSummaryModelChange?.({source: e.target.value, modelId: e.target.value === 'cli_adapter' ? 'current' : ''})}
          >
            <option value="cli_adapter">{t('settings.currentCLIAdapter')}</option>
            <option value="local_model">{t('settings.localModel')}</option>
            <option value="manual">{t('settings.manualInput')}</option>
          </select>
        </div>
        {summarySettings.source === 'cli_adapter' && (
          <div className="browser-pref-row">
            <label htmlFor="summary-adapter">CLI adapter</label>
            <select
              id="summary-adapter"
              value={summarySettings.modelId || 'current'}
              onChange={(e) => onSummaryModelChange?.({modelId: e.target.value})}
            >
              <option value="current">{t('settings.followCurrentAdapter')}</option>
              {(adapterList || []).map((adapter) => (
                <option key={adapter.id || adapter.name} value={adapter.id || adapter.name}>{adapter.name || adapter.id}</option>
              ))}
            </select>
          </div>
        )}
        {summarySettings.source === 'local_model' && (
          <>
            <div className="browser-pref-row">
              <label htmlFor="summary-local">{t('settings.localModel')}</label>
              <select
                id="summary-local"
                value={summarySettings.modelId || ''}
                onChange={(e) => {
                  const picked = localOptions.find((option) => option.id === e.target.value);
                  onSummaryModelChange?.({modelId: e.target.value, endpoint: picked?.endpoint || ''});
                }}
              >
                <option value="">{summaryModelScan?.message || t('settings.selectLocalModel')}</option>
                {localOptions.map((option) => (
                  <option key={`${option.provider}:${option.id}`} value={option.id}>{option.label}</option>
                ))}
              </select>
            </div>
            <button className="style-diff-preview-btn" type="button" onClick={onSummaryModelRescan}>{t('settings.rescanLocalModel')}</button>
          </>
        )}
        {(summarySettings.source === 'manual' || (summarySettings.source === 'local_model' && localOptions.length === 0)) && (
          <div className="summary-manual-grid">
            <input
              type="text"
              placeholder={t('settings.modelNamePlaceholder')}
              value={summarySettings.manualModel || ''}
              onChange={(e) => onSummaryModelChange?.({manualModel: e.target.value, modelId: e.target.value})}
            />
            <input
              type="text"
              placeholder={t('settings.endpointPlaceholder')}
              value={summarySettings.manualEndpoint || summarySettings.endpoint || ''}
              onChange={(e) => onSummaryModelChange?.({manualEndpoint: e.target.value, endpoint: e.target.value})}
            />
          </div>
        )}
        <label className="summary-model-checkbox">
          <input
            type="checkbox"
            checked={!!summarySettings.alwaysUse}
            onChange={(e) => onSummaryModelChange?.({alwaysUse: e.target.checked})}
          />
          <span>{t('settings.alwaysUseSetting')}</span>
        </label>
      </section>

      <section className="voice-settings">
        <header className="voice-settings-header">
          <h4>{t('settings.localVoice')}</h4>
          <button type="button" className="style-diff-preview-btn" onClick={onVoiceSettingsRefresh}>{t('settings.voiceRecheck')}</button>
        </header>
        <div className="voice-health-row">
          <span className={`voice-health-dot ${voiceState?.status === 'ready' ? 'voice-health-ready' : ''}`} />
          <span>{voiceStatusLabel(voiceState?.status)}</span>
          <small>{voiceState?.language ? `language: ${voiceState.language}` : ''}</small>
        </div>
        <div className="voice-install-row">
          <button
            type="button"
            className="style-diff-preview-btn"
            disabled={voiceInstallBusy}
            onClick={voiceState?.modelAvailable ? onVoiceModelRemove : onVoiceModelInstall}
          >
            {voiceInstallBusy ? t('settings.voiceProcessing') : voiceState?.modelAvailable ? t('settings.voiceRemoveBase') : t('settings.voiceInstallBase')}
          </button>
          <code>{voiceState?.managedModelPath || 'voice/models/ggml-base.bin'}</code>
        </div>
        <div className="browser-pref-row">
          <label htmlFor="voice-language-mode">{t('settings.voiceLanguage')}</label>
          <select
            id="voice-language-mode"
            value={voiceSettings.languageMode || 'auto'}
            onChange={(e) => onVoiceSettingsChange?.({languageMode: e.target.value})}
          >
            <option value="follow_app">{t('settings.followAppLang')}</option>
            <option value="auto">{t('settings.autoDetect')}</option>
            <option value="manual">{t('settings.manualSpecify')}</option>
          </select>
        </div>
        {voiceSettings.languageMode === 'manual' && (
          <div className="browser-pref-row">
            <label htmlFor="voice-language-manual">{t('settings.specifyLanguage')}</label>
            <select
              id="voice-language-manual"
              value={voiceSettings.manualLanguage || 'zh'}
              onChange={(e) => onVoiceSettingsChange?.({manualLanguage: e.target.value})}
            >
              <option value="zh">{t('settings.chinese')}</option>
              <option value="en">English</option>
              <option value="ja">{t("settings.langJa")}</option>
              <option value="ko">한국어</option>
            </select>
          </div>
        )}
        <div className="voice-install-row">
          <code>{voiceState?.whisperBinPath || t('settings.voiceNotIncluded')}</code>
        </div>
        <label className="summary-model-checkbox">
          <input
            type="checkbox"
            checked={!!voiceSettings.debugMode}
            onChange={(e) => onVoiceSettingsChange?.({debugMode: e.target.checked})}
          />
          <span>{t('settings.voiceDebugMode')}</span>
          <button type="button" className="style-diff-preview-btn" onClick={onVoiceDebugClear}>{t('settings.voiceDebugClear')}</button>
        </label>
        <label className="summary-model-checkbox">
          <input
            type="checkbox"
            checked={!!voiceSettings.commandMode}
            onChange={(e) => onVoiceSettingsChange?.({commandMode: e.target.checked})}
          />
          <span>{t('settings.voiceCommandMode')}</span>
        </label>
      </section>

      {/* Style Diff Preview trigger */}
      <div className="style-diff-input-row">
        <label htmlFor="style-diff-input">Style Diff JSON</label>
        <textarea
          id="style-diff-input"
          className="style-diff-textarea"
          placeholder='{"--bg": "#111", "--surface": "#222"}'
          value={diffInput}
          onChange={(e) => setDiffInput(e.target.value)}
          rows={3}
        />
        <button
          type="button"
          className="style-diff-preview-btn"
          disabled={!diffInput.trim()}
          onClick={() => onPreviewStyleDiff(diffInput.trim())}
        >
          {t('styleDiff.previewLabel')}
        </button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// I-4: State Token Legend (static color explanation)
// ---------------------------------------------------------------------------
/* i18n: stateToken */
function getStateTokenDescriptions() {
  return [
    {color: 'var(--amber, #f5a623)', label: _t('stateToken.highRiskLabel'), desc: _t('stateToken.highRiskDesc')},
    {color: 'var(--msg, #7f2d20)', label: _t('stateToken.interruptedLabel'), desc: _t('stateToken.interruptedDesc')},
    {color: '#4caf50', label: _t('stateToken.normalLabel'), desc: _t('stateToken.normalDesc')},
    {color: 'var(--muted, #bbb5a5)', label: _t('stateToken.idleLabel'), desc: _t('stateToken.idleDesc')},
  ];
}

function StateTokenLegend() {
  const descriptions = getStateTokenDescriptions();
  return (
    <div className="state-token-legend">
      <h4>{_t('stateToken.legendTitle')}</h4>
      <div className="state-token-list">
        {descriptions.map((item) => (
          <div className="state-token-row" key={item.label}>
            <span className="state-token-dot" style={{background: item.color}} />
            <div className="state-token-info">
              <span className="state-token-label">{item.label}</span>
              <span className="state-token-desc">{item.desc}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// I-3: Controlled Trust Settings Panel
// ---------------------------------------------------------------------------
function ControlledTrustSettings({
  trustDomClickEnabled, activeOverrides, trustedSessionActive, deviceProfile,
  onToggleTrustDomClick, onEnableOverride, onEnableWorkflowTrust,
  onDisableOverride, onEnableTrustedSession, onLoadDeviceProfile,
}) {
  const [overrideScope, setOverrideScope] = useState('');
  const [overrideRisk, setOverrideRisk] = useState('low');
  const [workflowHours, setWorkflowHours] = useState(1);

  return (
    <div className="trust-settings-section">
      <h4>{_t('controlledTrust.title')}</h4>

      {/* #I-304 Trust DOM & Click */}
      <div className="trust-toggle-row">
        <span>{_t('controlledTrust.trustDomClick')}</span>
        <button
          type="button"
          className={trustDomClickEnabled ? 'trust-active' : ''}
          onClick={() => onToggleTrustDomClick(!trustDomClickEnabled)}
        >
          {trustDomClickEnabled ? _t('controlledTrust.enabled') : _t('controlledTrust.disabled')}
        </button>
      </div>

      {/* #I-302 Trusted Session Scope */}
      <div className="trust-toggle-row">
        <span>Trusted Session Scope</span>
        <button
          type="button"
          className={trustedSessionActive ? 'trust-active' : ''}
          onClick={() => onEnableTrustedSession('', '')}
          disabled={trustedSessionActive}
        >
          {trustedSessionActive ? _t('controlledTrust.enabled') : _t('controlledTrust.enable')}
        </button>
      </div>
      {trustedSessionActive && (
        <small style={{display: 'block', fontSize: '11px', opacity: 0.6, padding: '2px 0 6px'}}>
          {_t('controlledTrust.mergeHint')}
        </small>
      )}

      {/* #I-301 Contextual Risk Override */}
      <div className="trust-toggle-row" style={{flexWrap: 'wrap', gap: '6px'}}>
        <span>{_t('controlledTrust.contextualOverride')}</span>
        <div style={{display: 'flex', gap: '4px', alignItems: 'center', flexWrap: 'wrap'}}>
          <input
            type="text"
            placeholder="scope JSON"
            value={overrideScope}
            onChange={(e) => setOverrideScope(e.target.value)}
            style={{width: '140px', fontSize: '11px', padding: '3px 6px', borderRadius: '4px', border: '1px solid var(--surface-2)', background: 'var(--bg)', color: 'var(--text)'}}
          />
          <select
            value={overrideRisk}
            onChange={(e) => setOverrideRisk(e.target.value)}
            style={{fontSize: '11px', padding: '3px 6px', borderRadius: '4px', border: '1px solid var(--surface-2)', background: 'var(--bg)', color: 'var(--text)'}}
          >
            <option value="low">low</option>
            <option value="medium">medium</option>
            <option value="high_non_destructive">high_non_destructive</option>
          </select>
          <button type="button" onClick={() => { if (overrideScope) onEnableOverride(overrideScope, overrideRisk); }}>{_t('controlledTrust.enable')}</button>
        </div>
      </div>

      {/* Workflow Trust shortcut */}
      <div className="trust-toggle-row" style={{flexWrap: 'wrap', gap: '6px'}}>
        <span>{_t('controlledTrust.workflowTrust')}</span>
        <div style={{display: 'flex', gap: '4px', alignItems: 'center'}}>
          <input
            type="number"
            min="1"
            max="999"
            value={workflowHours}
            onChange={(e) => setWorkflowHours(Math.max(1, Math.min(999, Number(e.target.value) || 1)))}
            style={{width: '50px', fontSize: '11px', padding: '3px 6px', borderRadius: '4px', border: '1px solid var(--surface-2)', background: 'var(--bg)', color: 'var(--text)'}}
          />
          <span style={{fontSize: '11px'}}>{_t('controlledTrust.hours')}</span>
          <button type="button" onClick={() => { if (overrideScope) onEnableWorkflowTrust(overrideScope, overrideRisk, workflowHours); }}>{_t('controlledTrust.enable')}</button>
        </div>
      </div>

      {/* Active overrides list */}
      {activeOverrides.length > 0 && (
        <div className="trust-override-list">
          {activeOverrides.map((o) => (
            <div className="trust-override-item" key={o.id}>
              <span>{o.allowedRisk}{o.hours ? ` · ${o.hours}h` : ''}</span>
              <button type="button" onClick={() => onDisableOverride(o.id)}>{_t('controlledTrust.disabled')}</button>
            </div>
          ))}
        </div>
      )}

      {/* #I-305 Device Trust Profile */}
      <div className="trust-toggle-row" style={{marginTop: '8px'}}>
        <span>{_t('controlledTrust.deviceTrust')}</span>
        <button type="button" onClick={() => onLoadDeviceProfile('default')}>
          {deviceProfile ? _t('controlledTrust.reload') : _t('controlledTrust.load')}
        </button>
      </div>
      {deviceProfile && (
        <div style={{fontSize: '11px', opacity: 0.7, padding: '4px 0'}}>
          <span>Profile: {deviceProfile.id || 'default'}</span>
          {deviceProfile.devicePenalty != null && <span> · penalty: {deviceProfile.devicePenalty}</span>}
        </div>
      )}
    </div>
  );
}

function PersonaSettingsDrawer({
  settingsState, onPersonaAdd, onPersonaDrop, onPersonaChange, onPersonaNativeDrag, onPersonaNativeExportAction, onPersonaReorder,
  avatarConfigs = {}, avatarExpression, avatarModeNotice, renderedPixelAvatars = {},
  staticAvatarPreviews = {}, onAvatarProviderSelect, onAvatarStateSelect, onAvatarLoad,
}) {
  const t = useI18n(s => s.t);
  const personas = settingsState.personas;
  const activePersona = findActivePersona(settingsState) || fallbackSettings.personas[0];
  const activeAvatarConfig = avatarConfigs[activePersona.id] || null;
  const activeAvatarProvider = resolveAvatarProvider(activeAvatarConfig);
  const activeRenderedPixelAvatar = renderedPixelAvatars[activePersona.id] || '';
  const activeAvatarSrc = resolvePersonaAvatarSrc(activePersona, activeAvatarConfig, staticAvatarPreviews, activeRenderedPixelAvatar);
  const activeReplyStrategy = activePersona.replyStrategy || '';
  const activeReplyStrategyIsPreset = !activeReplyStrategy || Boolean(replyStrategyPresetFor(activeReplyStrategy));
  const activeAvatarLocked = activePersona.id === lockedPersonaId;
  const canAddPersona = personas.length < maxPersonas;
  const personaSlotCount = personas.length + (canAddPersona ? 1 : 0);
  const personaRowRef = useRef(null);
  const personaFormRef = useRef(null);
  const draggedPersonaIdRef = useRef(null);
  const personaPointerDragRef = useRef(null);
  const suppressPersonaClickRef = useRef(false);
  const [draggedPersonaId, setDraggedPersonaId] = useState(null);
  const [personaExportDialog, setPersonaExportDialog] = useState(null);
  const [strengthPickerOpen, setStrengthPickerOpen] = useState(false);
  const [avatarPickerOpen, setAvatarPickerOpen] = useState(false);
  const [pixelPackPopup2, setPixelPackPopup2] = useState(false);
  const [pixelPacks2, setPixelPacks2] = useState([]);
  const [currentPack2, setCurrentPack2] = useState('wolf');
  const [packPreviews2, setPackPreviews2] = useState({});
  const [strengthDraft, setStrengthDraft] = useState(parseToneChance(activePersona.roleStrength));

  useEffect(() => {
    setStrengthDraft(parseToneChance(activePersona.roleStrength));
    setStrengthPickerOpen(false);
    setAvatarPickerOpen(false);
    onAvatarLoad?.(activePersona.id);
  }, [activePersona.id]);

  // ── Pixel Pack Switcher（設定面板版）──
  useEffect(() => {
    if (!pixelPackPopup2) return;
    let cancelled = false;
    (async () => {
      try {
        const [packs] = await Promise.all([ListPixelAvatarPacks()]);
        if (cancelled) return;
        setPixelPacks2(packs);
        setCurrentPack2(pixelPackForPersona(activePersona, activeAvatarConfig));
        const previews = {};
        for (const p of packs) {
          try {
            const bytes = await RenderPixelAvatarPreview(p.id, 'idle', pixelAvatarRenderSize);
            if (bytes && bytes.length) {
              previews[p.id] = bytesToDataUrl(bytes);
            }
          } catch { /* ignore */ }
        }
        if (!cancelled) setPackPreviews2(previews);
      } catch { /* ignore */ }
    })();
    return () => { cancelled = true; };
  }, [pixelPackPopup2, activePersona.id, activeAvatarConfig]);

  async function switchPixelPack2(packId) {
    if (activeAvatarLocked) return;
    try {
      await SetPersonaPixelAvatarPack(activePersona.id, packId);
      setCurrentPack2(packId);
      setPixelPackPopup2(false);
      await onAvatarLoad?.(activePersona.id);
    } catch { /* ignore */ }
  }

  function collectPersonaFormPatch() {
    const form = personaFormRef.current;
    if (!form) return {};
    return {
      name: limitChineseText(form.elements.personaName?.value || '', 100),
      identity: form.elements.identity?.value || '',
      replyStrategy: form.elements.replyStrategy?.value || '',
      roleStrength: `${strengthDraft}%`,
      personality: form.elements.personality?.value || '',
      scenario: form.elements.scenario?.value || '',
      description: form.elements.description?.value || '',
    };
  }

  function commitActivePersonaForm() {
    if (!activePersona?.id) return activePersona;
    const patch = collectPersonaFormPatch();
    if (Object.keys(patch).length === 0) return activePersona;
    onPersonaChange(activePersona.id, patch);
    return {...activePersona, ...patch};
  }

  function scrollPersonas(direction) {
    const row = personaRowRef.current;
    if (!row) return;
    row.scrollBy({left: direction * row.clientWidth, behavior: 'smooth'});
  }

  async function handlePersonaExportAction(action) {
    if (!personaExportDialog) return;
    const target = personaExportDialog;
    setPersonaExportDialog(null);
    await onPersonaNativeExportAction?.(action, target);
  }

  function movePersonaBefore(sourcePersonaId, targetPersonaId) {
    if (!sourcePersonaId || sourcePersonaId === targetPersonaId) return;
    const orderIds = personas.map((persona) => persona.id);
    const fromIndex = orderIds.indexOf(sourcePersonaId);
    const toIndex = orderIds.indexOf(targetPersonaId);
    if (fromIndex < 0 || toIndex < 0) return;

    const nextOrderIds = [...orderIds];
    nextOrderIds.splice(fromIndex, 1);
    nextOrderIds.splice(toIndex, 0, sourcePersonaId);
    onPersonaReorder(nextOrderIds);
  }

  function personaIdAtPoint(clientX, clientY) {
    const target = document.elementFromPoint(clientX, clientY);
    return target?.closest?.('[data-persona-id]')?.dataset?.personaId || '';
  }

  function pointInsidePersonaRow(clientX, clientY) {
    const rect = personaRowRef.current?.getBoundingClientRect();
    return Boolean(rect && clientX >= rect.left && clientX <= rect.right && clientY >= rect.top && clientY <= rect.bottom);
  }

  function finishPersonaPointerDrag(event, sourcePersonaId, moved) {
    personaPointerDragRef.current = null;
    draggedPersonaIdRef.current = null;
    setDraggedPersonaId(null);
    if (!moved) return;

    const targetPersonaId = personaIdAtPoint(event.clientX, event.clientY);
    if (targetPersonaId && targetPersonaId !== sourcePersonaId) {
      movePersonaBefore(sourcePersonaId, targetPersonaId);
      return;
    }

    const isZeroCoords = event.clientX === 0 && event.clientY === 0;
    const inWindow = !isZeroCoords && event.clientX > 0 && event.clientY > 0
      && event.clientX < window.innerWidth && event.clientY < window.innerHeight;
    if (!inWindow || !pointInsidePersonaRow(event.clientX, event.clientY)) {
      return;
    }
  }

  function startPersonaPointerDrag(event, persona) {
    if (event.button != null && event.button !== 0) return;
    personaPointerDragRef.current = {
      id: persona.id,
      startX: event.clientX,
      startY: event.clientY,
      moved: false,
      nativeStarted: false,
    };

    const cleanupPointerDrag = () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', handlePointerUp);
      window.removeEventListener('pointercancel', handlePointerCancel);
    };

    const beginNativePersonaDrag = () => {
      const drag = personaPointerDragRef.current;
      if (!drag || drag.id !== persona.id || drag.nativeStarted) return;
      drag.nativeStarted = true;
      cleanupPointerDrag();
      personaPointerDragRef.current = null;
      draggedPersonaIdRef.current = null;
      setDraggedPersonaId(null);
      window.setTimeout(() => {
        suppressPersonaClickRef.current = false;
      }, 120);
      const dragPromise = onPersonaNativeDrag?.(persona.id);
      if (!dragPromise) return;
      dragPromise.then((result) => {
        if (result?.status !== 'success' || !result?.landed_path) return;
        setPersonaExportDialog({
          ...result,
          id: result.persona_id || persona.id,
          name: result.display_name || persona.name,
        });
      });
    };

    const handlePointerMove = (moveEvent) => {
      const drag = personaPointerDragRef.current;
      if (!drag || drag.id !== persona.id) return;
      const moved = Math.hypot(moveEvent.clientX - drag.startX, moveEvent.clientY - drag.startY) > 8;
      if (!moved) return;
      drag.moved = true;
      suppressPersonaClickRef.current = true;
      draggedPersonaIdRef.current = persona.id;
      setDraggedPersonaId(persona.id);
      moveEvent.preventDefault();
      if (!pointInsidePersonaRow(moveEvent.clientX, moveEvent.clientY)) {
        beginNativePersonaDrag();
      }
    };

    const handlePointerUp = (upEvent) => {
      cleanupPointerDrag();
      const moved = Boolean(personaPointerDragRef.current?.moved);
      finishPersonaPointerDrag(upEvent, persona.id, moved);
      window.setTimeout(() => {
        suppressPersonaClickRef.current = false;
      }, 0);
    };

    const handlePointerCancel = () => {
      cleanupPointerDrag();
      personaPointerDragRef.current = null;
      draggedPersonaIdRef.current = null;
      setDraggedPersonaId(null);
      window.setTimeout(() => {
        suppressPersonaClickRef.current = false;
      }, 0);
    };

    window.addEventListener('pointermove', handlePointerMove, {passive: false});
    window.addEventListener('pointerup', handlePointerUp);
    window.addEventListener('pointercancel', handlePointerCancel);
  }

  function commitRoleStrength() {
    onPersonaChange(activePersona.id, {roleStrength: `${strengthDraft}%`});
    setStrengthPickerOpen(false);
  }

  function updateRoleStrength(value) {
    const next = parseToneChance(`${value}%`);
    setStrengthDraft(next);
    onPersonaChange(activePersona.id, {roleStrength: `${next}%`});
  }

  // v3.3.2 P0.1 — file drop routes through quarantine (onPersonaDrop), never directly.
  function installPersonaPackage(event) {
    event.preventDefault();
    if (!canAddPersona) return;
    const file = event.dataTransfer.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      onPersonaDrop(file.name, String(reader.result || ''));
    };
    reader.readAsText(file);
  }

  return (
    <main className="settings-persona-drawer" aria-label={t('persona.settingsAriaLabel')}>
      <div className="persona-card-row" ref={personaRowRef}>
        {personas.map((persona) => (
          <button
            className={`settings-persona-card ${persona.id === activePersona.id ? 'settings-persona-card-active' : ''} ${persona.id === draggedPersonaId ? 'settings-persona-card-dragging' : ''} ${persona.id === lockedPersonaId ? 'settings-persona-card-locked' : ''}`}
            type="button"
            key={persona.id}
            data-persona-id={persona.id}
            draggable={false}
            onDragStart={(event) => event.preventDefault()}
            onClick={(event) => {
              if (suppressPersonaClickRef.current) {
                event.preventDefault();
                event.stopPropagation();
                return;
              }
              commitActivePersonaForm();
              onPersonaChange(persona.id, {});
            }}
            onPointerDown={(event) => startPersonaPointerDrag(event, persona)}
          >
            <img
              className="settings-persona-avatar"
              draggable={false}
              onDragStart={(event) => event.preventDefault()}
              src={resolvePersonaAvatarSrc(persona, avatarConfigs[persona.id], staticAvatarPreviews, renderedPixelAvatars[persona.id] || '')}
              alt={t('persona.avatarAlt', { name: persona.name })}
            />
            <strong>{persona.name}</strong>
            {persona.id === lockedPersonaId && <small className="settings-persona-lock">{t('persona.lockedName')}</small>}
          </button>
        ))}
        {personaExportDialog && (
          <DragActionModal
            ariaLabel={t('persona.dragAriaLabel')}
            icon="◎"
            title={personaExportDialog.name}
            detail={personaExportDialog.landed_path || (personaExportDialog.id === lockedPersonaId ? t('persona.lockedPersonaExportNote') : t('persona.fileCopied'))}
            actions={[
              {
                label: t('persona.removeLocalLabel'),
                disabled: personaExportDialog.id === lockedPersonaId,
                onClick: () => handlePersonaExportAction('remove'),
              },
              {label: t('persona.keepCopyLabel'), onClick: () => handlePersonaExportAction('copy')},
              {label: t('persona.cancelDeleteLabel'), onClick: () => handlePersonaExportAction('cancel')},
            ]}
          />
        )}
        {canAddPersona && (
          <button
            className="settings-persona-card settings-persona-add-card"
            type="button"
            onClick={() => onPersonaAdd()}
            onDragOver={(event) => event.preventDefault()}
            onDrop={installPersonaPackage}
          >
            <span className="settings-persona-add-mark">＋</span>
            <strong>{t('persona.addPersona')}</strong>
            <small>{t('persona.dropRolepack')}</small>
          </button>
        )}
      </div>
      <div className="settings-card-track">
        <button type="button" onClick={() => scrollPersonas(-1)}>‹</button>
        <span>1-{Math.min(4, personaSlotCount)} / {personaSlotCount}</span>
        <button type="button" onClick={() => scrollPersonas(1)}>›</button>
      </div>
      <form
        className="persona-form"
        ref={personaFormRef}
        onFocusCapture={(event) => {
          if (strengthPickerOpen && !event.target.closest('.role-strength-field')) {
            commitRoleStrength();
          }
        }}
      >
        <section className="settings-avatar-editor" aria-label={t('persona.avatarEditorLabel', { name: activePersona.name })}>
          <button
            className={`settings-avatar-preview ${activeAvatarLocked ? 'settings-avatar-preview-locked' : ''}`}
            type="button"
            onClick={() => {
              if (!activeAvatarLocked) setAvatarPickerOpen((open) => !open);
            }}
            title={activeAvatarLocked ? t('avatar.locked') : t('persona.changeAvatarTitle')}
            aria-disabled={activeAvatarLocked}
          >
            <img src={activeAvatarSrc} alt={t('persona.avatarAlt', { name: activePersona.name })}/>
            <span>{avatarStateLabel(avatarExpression)}</span>
          </button>
          <div className="settings-avatar-copy">
            <strong>{t('persona.avatarTitle')}</strong>
            <small>{activeAvatarProvider === 'static_image' ? t('persona.staticAvatarLabel') : activeAvatarProvider === 'user_image_api' ? t('persona.genApiLabel') : t('persona.builtinAvatarLabel')}</small>
          </div>
          <button
            className="settings-avatar-action"
            type="button"
            disabled={activeAvatarLocked}
            onClick={() => setAvatarPickerOpen((open) => !open)}
          >
            {activeAvatarLocked ? t('persona.avatarLocked') : t('persona.changeAvatar')}
          </button>
          {avatarPickerOpen && !activeAvatarLocked && (
            <div className="avatar-provider-popover settings-avatar-popover">
              {/* Settings edits target the active persona card, while the main UI
                  avatar follows settingsState.personas[0]. */}
              <button type="button" onClick={() => onAvatarProviderSelect?.('user_image_api', activePersona.id)}>
                <strong>{t('persona.genApiLabel')}</strong>
                <small>{t('persona.genApiHint')}</small>
              </button>
              <button type="button" onClick={() => onAvatarProviderSelect?.('static_image', activePersona.id)}>
                <strong>{t('persona.staticAvatarLabel')}</strong>
                <small>{t('persona.staticAvatarHint')}</small>
              </button>
              <button type="button" onClick={() => onAvatarProviderSelect?.('built_in_pixel', activePersona.id)}>
                <strong>{t('persona.builtinAvatarLabel')}</strong>
                <small>{t('persona.builtinAvatarHint')}</small>
              </button>
              <div className="avatar-state-grid" aria-label={t('persona.avatarStatePreview')}>
                {avatarStateOptions.map((state) => (
                  <button
                    className={state === avatarExpression ? 'avatar-state-active' : ''}
                    type="button"
                    key={state}
                    onClick={() => onAvatarStateSelect?.(state)}
                  >
                    {avatarStateLabel(state)}
                  </button>
                ))}
                <button type="button" onClick={() => onAvatarStateSelect?.('')}>{t('persona.autoState')}</button>
                <button
                  type="button"
                  className="pixel-pack-switch-btn"
                  onClick={() => setPixelPackPopup2((v) => !v)}
                  title={t('persona.togglePixelPack')}
                >
                  {t('persona.pixelPack')}
                </button>
              </div>
              {pixelPackPopup2 && (
                <div className="pixel-pack-popup">
                  <div className="pixel-pack-popup-title">{t('persona.selectPixelPack')}</div>
                  <div className="pixel-pack-list">
                    {pixelPacks2.map((p) => (
                      <button
                        key={p.id}
                        type="button"
                        className={`pixel-pack-card${p.id === currentPack2 ? ' pixel-pack-active' : ''}`}
                        onClick={() => switchPixelPack2(p.id)}
                      >
                        {packPreviews2[p.id] && (
                          <img
                            src={packPreviews2[p.id]}
                            alt={p.name}
                            className="pixel-pack-preview"
                          />
                        )}
                        <span className="pixel-pack-name">{p.name}</span>
                        <span className="pixel-pack-desc">{p.desc}</span>
                      </button>
                    ))}
                  </div>
                  <button
                    type="button"
                    className="pixel-pack-close"
                    onClick={() => setPixelPackPopup2(false)}
                  >
                    {t('persona.closePopup')}
                  </button>
                </div>
              )}
              <p>{avatarModeNotice || t('persona.currentProvider', { provider: activeAvatarProvider })}</p>
            </div>
          )}
        </section>
        <input
          aria-label={t('persona.nameAriaLabel')}
          defaultValue={activePersona.id === lockedPersonaId ? lockedPersonaName : activePersona.name}
          disabled={activePersona.id === lockedPersonaId}
          key={`${activePersona.id}-name`}
          name="personaName"
          placeholder={t('persona.namePlaceholder')}
          maxLength={100}
          onBlur={(event) => {
            const value = limitChineseText(event.target.value, 100);
            event.target.value = value;
            if (activePersona.id === lockedPersonaId) return;
            onPersonaChange(activePersona.id, {name: value});
          }}
        />
        <input
          aria-label={t('persona.identityAriaLabel')}
          defaultValue={activePersona.identity}
          key={`${activePersona.id}-identity`}
          name="identity"
          placeholder={t('persona.identityPlaceholder')}
          onBlur={(event) => onPersonaChange(activePersona.id, {identity: event.target.value})}
        />
        <div className="persona-form-grid">
          <label className="persona-strategy-select" aria-label={t('persona.replyStrategyAriaLabel')}>
            <select
              defaultValue={activeReplyStrategy}
              key={`${activePersona.id}-reply`}
              name="replyStrategy"
              onChange={(event) => onPersonaChange(activePersona.id, {replyStrategy: event.target.value})}
            >
              <option value="">{t('persona.replyStrategyDefault')}</option>
              {getReplyStrategyPresets().map((preset) => (
                <option key={preset.id} value={preset.id}>{preset.label}</option>
              ))}
              {!activeReplyStrategyIsPreset && (
                <option value={activeReplyStrategy}>{t('persona.customStrategy', { value: activeReplyStrategy })}</option>
              )}
            </select>
          </label>
          <div
            className="role-strength-field"
            onBlurCapture={(event) => {
              if (!event.currentTarget.contains(event.relatedTarget)) {
                commitRoleStrength();
              }
            }}
          >
            <input
              name="roleStrength"
              type="hidden"
              value={`${strengthDraft}%`}
              readOnly
            />
            <button
              aria-expanded={strengthPickerOpen}
              aria-label={t('persona.roleStrengthAriaLabel')}
              className="role-strength-trigger"
              type="button"
              onClick={() => setStrengthPickerOpen((open) => !open)}
            >
              <span>{t('persona.roleStrengthLabel')}</span>
              <strong>{strengthDraft}%</strong>
            </button>
            {strengthPickerOpen && (
              <div className="role-strength-popover">
                <input
                  aria-label={t('persona.toneProbAriaLabel')}
                  max="100"
                  min="10"
                  step="10"
                  type="range"
                  value={strengthDraft}
                  onChange={(event) => updateRoleStrength(event.target.value)}
                />
                <div className="role-strength-ticks">
                  <span>10%</span>
                  <span>50%</span>
                  <span>100%</span>
                </div>
                <small>{t('persona.tonePromptHint')}</small>
              </div>
            )}
          </div>
          <input
            aria-label={t('persona.personalityAriaLabel')}
            defaultValue={activePersona.personality}
            key={`${activePersona.id}-personality`}
            name="personality"
            placeholder={t('persona.personalityPlaceholder')}
            onBlur={(event) => onPersonaChange(activePersona.id, {personality: event.target.value})}
          />
          <input
            aria-label={t('persona.scenarioAriaLabel')}
            defaultValue={activePersona.scenario}
            key={`${activePersona.id}-scenario`}
            name="scenario"
            placeholder={t('persona.scenePlaceholder')}
            onBlur={(event) => onPersonaChange(activePersona.id, {scenario: event.target.value})}
          />
        </div>
        <textarea
          aria-label={t('persona.descriptionAriaLabel')}
          defaultValue={activePersona.description}
          key={`${activePersona.id}-description`}
          name="description"
          placeholder={t('persona.extraPlaceholder')}
          onBlur={(event) => onPersonaChange(activePersona.id, {description: event.target.value})}
        />
      </form>
      <button className="settings-bottom-action" type="button">↓</button>
    </main>
  );
}

function SettingSelect({icon, label, value, onNext}) {
  return (
    <label className="settings-select-row">
      <span className="settings-select-label"><i>{icon}</i>{label}</span>
      <button type="button" onClick={onNext}>
        <span className="settings-select-value">{value}</span>
        <span className="settings-select-caret">⌄</span>
      </button>
    </label>
  );
}

function AvatarUploadModal({persona, onApply, onClose}) {
  const t = useI18n(s => s.t);
  const fileInputRef = useRef(null);
  const [selectedFile, setSelectedFile] = useState(null);
  const [previewSrc, setPreviewSrc] = useState('');
  const [dragActive, setDragActive] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    // The modal owns temporary preview URLs so pasted or dropped images can be
    // inspected before any data is sent to the backend.
    if (!selectedFile) {
      setPreviewSrc('');
      return undefined;
    }
    const url = URL.createObjectURL(selectedFile);
    setPreviewSrc(url);
    return () => URL.revokeObjectURL(url);
  }, [selectedFile]);

  useEffect(() => {
    // Clipboard paste is scoped to this modal's lifetime; closing the dialog
    // removes the listener and avoids surprising paste behavior elsewhere.
    function handlePaste(event) {
      const file = Array.from(event.clipboardData?.files || []).find((item) => item.type?.startsWith('image/'));
      if (file) pickFile(file);
    }
    window.addEventListener('paste', handlePaste);
    return () => window.removeEventListener('paste', handlePaste);
  }, []);

  function pickFile(file) {
    if (!file?.type?.startsWith('image/')) {
      setError(t('persona.uploadImageError'));
      return;
    }
    setError('');
    setSelectedFile(file);
  }

  function handleDrop(event) {
    event.preventDefault();
    setDragActive(false);
    const file = event.dataTransfer.files?.[0];
    if (file) pickFile(file);
  }

  return createPortal(
    <div className="avatar-upload-backdrop" role="dialog" aria-modal="true" aria-label={t('persona.uploadTitle')}>
      <section className="avatar-upload-modal">
        <header>
          <span>{t('persona.uploadPersonaLabel', { name: persona?.name || t('persona.defaultName') })}</span>
          <button type="button" onClick={onClose}>×</button>
        </header>
        <button
          className={`avatar-drop-zone ${dragActive ? 'avatar-drop-zone-active' : ''}`}
          type="button"
          onClick={() => fileInputRef.current?.click()}
          onDragEnter={(event) => { event.preventDefault(); setDragActive(true); }}
          onDragOver={(event) => event.preventDefault()}
          onDragLeave={() => setDragActive(false)}
          onDrop={handleDrop}
        >
          {previewSrc ? (
            <img src={previewSrc} alt={t('persona.previewAlt')}/>
          ) : (
            <>
              <strong>{t('persona.dropImage')}</strong>
              <span>{t('persona.dropHint')}</span>
            </>
          )}
        </button>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          hidden
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) pickFile(file);
          }}
        />
        {error && <p className="avatar-upload-error">{error}</p>}
        <footer>
          <button type="button" onClick={() => fileInputRef.current?.click()}>{t('persona.selectFile')}</button>
          <button type="button" onClick={() => { setSelectedFile(null); setError(''); }}>{t('persona.rePaste')}</button>
          <button type="button" onClick={onClose}>{t('common.cancel')}</button>
          <button type="button" disabled={!selectedFile} onClick={() => selectedFile && onApply(selectedFile)}>{t('persona.apply')}</button>
        </footer>
      </section>
    </div>,
    document.body
  );
}

function TopConsole({
  activePersona, activeAvatarConfig, avatarExpression, avatarModeNotice, avatarProvider, avatarSrc,
  browserPref, greeting, haoras, subagentTabs, personaJob, personaName,
  dagRun, reviewState, reviewPopup, skillInjections, snoozeHours,
  systemStatusHistory, reviewArchive,
  showSkillFirstUseCard, onDismissSkillFirstUse,
  w3aImportPopup, w3aDetail, w3aPollutionResult, w3aTransferGuidance, w3aTrustList,
  w3aActionBusy, w3aActionError, w3aStatusConfig, w3aToastMsg,
  onLoadW3AInfo, onDetectW3APollution, onShowW3AGuidance, onTrustW3ADeveloper, onExportW3ACopy,
  onDismissW3AImportPopup, onShowW3AToast, onDismissW3AToast,
  onAvatarProviderSelect, onAvatarLoad, onAvatarStateSelect, onPersonaJobChange, onPersonaNameChange,
  onReviewPopupChange, onRotateGreeting, onSkillSelect,
  onSnooze, onSnoozeHoursChange, onAcknowledgeDigestItem, onConfirmSkillBuild,
  activeAdapter, adapterOptions = [], activeAdapterId, onAdapterSelect, cliInspectorBusy, cliInspectorLog, onCLIInspectSend,
  // v3.6.3 Remote Bridge props
  remoteBridgeChannels, remoteBridgeModePopup, remoteBridgeInboundInfo, remoteBridgeInboundAdapters,
  onToggleRemoteBridge, onOpenRemoteBridgeMode, onSwitchRemoteBridgeMode,
  onRemoveRemoteBridge, onCloseRemoteBridgeModePopup,
  onOpenRemoteBridgeSetup, onRemoteBridgeRename, onTemporaryRemoteBridgeTest, onSaveRemoteBridgeInboundSecret, onMakeRemoteBridgePrimary,
  subExportCapabilities,
  activeHaoraId, onHaoraSelect, onRenameHaora, onHaorasReordered, onSubagentsChanged,
}) {
  const {t} = useI18n();
  const [editingName, setEditingName] = useState(false);
  const [editingJob, setEditingJob] = useState(false);
  const [personaInfoOpen, setPersonaInfoOpen] = useState(false);
  const [interactiveOpen, setInteractiveOpen] = useState(false);
  const [interactiveText, setInteractiveText] = useState('');
  const [promptOpen, setPromptOpen] = useState(false);
  const [promptText, setPromptText] = useState('');
  const [avatarPickerOpen, setAvatarPickerOpen] = useState(false);
  const [pixelPackPopup, setPixelPackPopup] = useState(false);
  const [pixelPacks, setPixelPacks] = useState([]);
  const [currentPack, setCurrentPack] = useState('wolf');
  const [draggedHaoraKey, setDraggedHaoraKey] = useState(null);
  const [haoraExportDialog, setHaoraExportDialog] = useState(null);
  const [editingHaoraKey, setEditingHaoraKey] = useState(null);
  const [haoraNameDraft, setHaoraNameDraft] = useState('');
  const [haoraDragPreview, setHaoraDragPreview] = useState(null);
  const [remoteInboundSecretDraft, setRemoteInboundSecretDraft] = useState('');
  const haoraPointerDragRef = useRef(null);
  const haoraNativeExportInFlightRef = useRef(false);
  const nativeSubDragSupported = subExportCapabilities?.native_drag_supported === true;
  const haoraItems = (Array.isArray(subagentTabs) && subagentTabs.length > 0)
    ? [
        {key: 'main', id: 'main', name: haoras?.[0] || t('persona.mainHaora'), isMain: true},
        ...subagentTabs.map((tab) => ({
          key: tab.id || tab.name,
          id: tab.id || tab.name,
          name: tab.name || tab.id,
          isMain: false,
        })),
      ]
    : (haoras || []).map((name, index) => ({
        key: index === 0 ? 'main' : name,
        id: name,
        name,
        isMain: index === 0,
      }));

  function moveHaoraTab(targetKey, {keepDragging = false} = {}) {
    const sourceKey = draggedHaoraKey || haoraPointerDragRef.current?.key;
    if (!sourceKey || sourceKey === targetKey) return;
    const fromIndex = haoraItems.findIndex((item) => item.key === sourceKey);
    const toIndex = haoraItems.findIndex((item) => item.key === targetKey);
    if (fromIndex <= 0 || toIndex <= 0) return;
    const next = [...haoraItems];
    const [moved] = next.splice(fromIndex, 1);
    next.splice(toIndex, 0, moved);
    if (!keepDragging) setDraggedHaoraKey(null);
    onHaorasReordered?.(next.map((item) => item.name), next.filter((item) => !item.isMain).map((item) => item.id));
  }

  function finishHaoraDrag() {
    setDraggedHaoraKey(null);
    setHaoraDragPreview(null);
  }

  function startHaoraPointerDrag(event, haora, options = {}) {
    if (!options.skipDebug) {
      markHaoraDragDebug('pointerdown', haora, `native=${nativeSubDragSupported ? 'yes' : 'no'}`);
    }
    if (!haora || haora.isMain || editingHaoraKey) return;
    haoraPointerDragRef.current = {
      key: haora.key,
      startX: event.clientX,
      startY: event.clientY,
      moved: false,
      nativeStarted: false,
      pointerId: event.pointerId,
    };
    setDraggedHaoraKey(haora.key);
    event.currentTarget.setPointerCapture?.(event.pointerId);
  }

  function handleHaoraPointerMove(event) {
    const drag = haoraPointerDragRef.current;
    if (!drag || drag.nativeStarted) return;
    const distance = Math.hypot(event.clientX - drag.startX, event.clientY - drag.startY);
    if (distance < 8 && !drag.moved) return;
    drag.moved = true;
    event.preventDefault();
    markHaoraDragDebug('pointermove', {name: drag.key}, `distance=${Math.round(distance)}`);
    const haora = haoraItems.find((item) => item.key === drag.key);
    if (haora && !haora.isMain) {
      setHaoraDragPreview({
        key: haora.key,
        name: haora.name,
        x: event.clientX,
        y: event.clientY,
      });
    }
    const leavingWindow =
      event.clientX <= 2 ||
      event.clientY <= 2 ||
      event.clientX >= window.innerWidth - 2 ||
      event.clientY >= window.innerHeight - 2;
    if (leavingWindow && nativeSubDragSupported && haora && !haora.isMain) {
      drag.nativeStarted = true;
      // DOM previews cannot leave WebView; hand off to OS drag at the window edge.
      event.currentTarget.releasePointerCapture?.(drag.pointerId ?? event.pointerId);
      setDraggedHaoraKey(null);
      setHaoraDragPreview(null);
      haoraPointerDragRef.current = null;
      startNativeHaoraExport(haora);
      return;
    }
    const target = document.elementFromPoint?.(event.clientX, event.clientY)?.closest?.('[data-haora-key]');
    const targetKey = target?.dataset?.haoraKey;
    if (!targetKey || targetKey === 'main' || targetKey === drag.key) return;
    moveHaoraTab(targetKey, {keepDragging: true});
  }

  function finishHaoraPointerDrag(event) {
    const drag = haoraPointerDragRef.current;
    if (!drag) return;
    event.currentTarget.releasePointerCapture?.(event.pointerId);
    setDraggedHaoraKey(null);
    setHaoraDragPreview(null);
    setTimeout(() => {
      if (haoraPointerDragRef.current === drag) {
        haoraPointerDragRef.current = null;
      }
    }, 0);
  }

  function startInlineHaoraRename(event, haora) {
    if (!haora || haora.isMain) return;
    event?.preventDefault?.();
    event?.stopPropagation?.();
    setDraggedHaoraKey(null);
    setEditingHaoraKey(haora.key);
    setHaoraNameDraft(haora.name || '');
  }

  function cancelInlineHaoraRename() {
    setEditingHaoraKey(null);
    setHaoraNameDraft('');
  }

  async function commitInlineHaoraRename(haora) {
    if (!haora || editingHaoraKey !== haora.key) return;
    const next = haoraNameDraft.trim();
    setEditingHaoraKey(null);
    setHaoraNameDraft('');
    if (!next || next === haora.name) return;
    await onRenameHaora?.(haora.name, next);
  }

  function formatHaoraNativeDropDetail(result) {
    const landed = result?.landed_path || result?.message || t('subagent.dragNativeFailed');
    if (!result?.drop_target_kind || !result?.drop_target_dir) return landed;
    return `${landed}\n${result.drop_target_kind}: ${result.drop_target_dir}`;
  }

  function showNativeHaoraExportDialog(result, fallbackHaora = null) {
    if (result?.status !== 'success' || !result?.landed_path) {
      if (result?.message) {
        markHaoraDragDebug('native-no-drop', fallbackHaora, result.message);
      }
      return;
    }
    setHaoraExportDialog({
      id: result.sub_id || fallbackHaora?.id || '',
      name: result.display_name || fallbackHaora?.name || 'subagent',
      tempExportDir: result.export_dir,
      landedPath: result.landed_path,
      landedDetail: formatHaoraNativeDropDetail(result),
      newSystemCode: result.new_system_code,
      nativeFailed: false,
    });
  }

  useEffect(() => {
    const offNativeSubExport = EventsOn('subexport:native_completed', (result) => {
      setDraggedHaoraKey(null);
      showNativeHaoraExportDialog(result);
    });
    return () => offNativeSubExport();
  }, []);

  async function startNativeHaoraExport(haora) {
    if (!haora || haora.isMain) return;
    if (haoraNativeExportInFlightRef.current) {
      markHaoraDragDebug('native-skip', haora, 'already-running');
      return;
    }
    haoraNativeExportInFlightRef.current = true;
    markHaoraDragDebug('native-call', haora);
    setHaoraDragPreview(null);
    setDraggedHaoraKey(haora.key);
    try {
      const result = await callWails(() => NativeDragExportSubHandler(haora.id, haora.name, 'export_copy', '[]'));
      setDraggedHaoraKey(null);
      if (result?.status && result.status !== 'success') {
        markHaoraDragDebug('native-result', haora, result.status);
      }
      showNativeHaoraExportDialog(result, haora);
    } catch (err) {
      setDraggedHaoraKey(null);
      markHaoraDragDebug('native-error', haora, err?.message || String(err));
      console.error('[NATIVE SUB EXPORT]', err);
    } finally {
      haoraNativeExportInFlightRef.current = false;
    }
  }

  function startNativeHaoraExportDrag(event, haora) {
    event.preventDefault();
    event.stopPropagation();
    markHaoraDragDebug('dragstart', haora, `native=${nativeSubDragSupported ? 'yes' : 'no'}`);
    if (!nativeSubDragSupported || !haora || haora.isMain) return;
    if (haoraPointerDragRef.current?.key === haora.key) {
      haoraPointerDragRef.current.nativeStarted = true;
      haoraPointerDragRef.current = null;
    }
    startNativeHaoraExport(haora);
  }

  function markHaoraDragDebug(stage, haora = null, detail = '') {
    const name = haora?.name || haora?.key || '';
    const time = new Date().toLocaleTimeString('zh-TW', {hour12: false});
    const message = `${time} ${stage}${name ? ` ${name}` : ''}${detail ? ` ${detail}` : ''}`;
    console.info('[HAORA DRAG]', message);
  }

  async function handleHaoraExportAction(action) {
    if (!haoraExportDialog) return;
    const target = haoraExportDialog;
    setHaoraExportDialog(null);
    try {
      await callWails(() => FinalizeNativeSubExport(
        action,
        target.id,
        target.tempExportDir || '',
        target.landedPath || '',
        target.newSystemCode || '',
      ));
      await onSubagentsChanged?.();
    } catch (err) {
      console.error('[SUB EXPORT]', err);
    }
  }
  const [packPreviews, setPackPreviews] = useState({});
  const [nameDraft, setNameDraft] = useState(personaName);
  const [jobDraft, setJobDraft] = useState(personaJob);
  const personaNameLocked = activePersona?.id === lockedPersonaId;
  const personaAvatarLocked = activePersona?.id === lockedPersonaId;

  // ── Pixel Pack Switcher：打開彈窗時載入套件清單 + 預覽圖 ──
  useEffect(() => {
    if (!pixelPackPopup) return;
    let cancelled = false;
    (async () => {
      try {
        const [packs] = await Promise.all([ListPixelAvatarPacks()]);
        if (cancelled) return;
        setPixelPacks(packs);
        setCurrentPack(pixelPackForPersona(activePersona, activeAvatarConfig));
        const previews = {};
        for (const p of packs) {
          try {
            const bytes = await RenderPixelAvatarPreview(p.id, 'idle', pixelAvatarRenderSize);
            if (bytes && bytes.length) {
              previews[p.id] = bytesToDataUrl(bytes);
            }
          } catch { /* ignore */ }
        }
        if (!cancelled) setPackPreviews(previews);
      } catch { /* ignore */ }
    })();
    return () => { cancelled = true; };
  }, [pixelPackPopup, activePersona?.id, activeAvatarConfig]);

  async function switchPixelPack(packId) {
    if (personaAvatarLocked) return;
    try {
      await SetPersonaPixelAvatarPack(activePersona.id, packId);
      setCurrentPack(packId);
      setPixelPackPopup(false);
      await onAvatarLoad?.(activePersona.id);
    } catch { /* ignore */ }
  }

  function savePersonaName() {
    if (personaNameLocked) {
      setNameDraft(lockedPersonaName);
      setEditingName(false);
      return;
    }
    const next = nameDraft.trim();
    onPersonaNameChange(next || t('persona.fallbackName'));
    setEditingName(false);
  }

  function savePersonaJob() {
    const next = jobDraft.trim();
    onPersonaJobChange(next || t('persona.fallbackJob'));
    setEditingJob(false);
  }

  return (
    <section className="top-console">
      <div className="persona-block">
        <button
          className="persona-avatar"
          type="button"
          onClick={() => setPersonaInfoOpen((open) => !open)}
          title={t('persona.showInfo')}
        >
          <img className="persona-avatar-image" src={avatarSrc} alt={t('persona.avatarAlt', { name: activePersona?.name || personaName })}/>
          <span className={`persona-avatar-state persona-avatar-state-${avatarExpression}`}>{avatarStateLabel(avatarExpression)}</span>
        </button>
        {personaInfoOpen && (
          <div className="persona-info-popover">
            {editingName && !personaNameLocked ? (
              <input
                className="persona-name persona-name-input"
                autoFocus
                value={nameDraft}
                onChange={(event) => setNameDraft(event.target.value)}
                onBlur={savePersonaName}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') savePersonaName();
                  if (event.key === 'Escape') {
                    setNameDraft(personaName);
                    setEditingName(false);
                  }
                }}
              />
            ) : (
              <button
                className="persona-name"
                type="button"
                onDoubleClick={() => {
                  if (personaNameLocked) return;
                  setNameDraft(personaName);
                  setEditingName(true);
                }}
                title={personaNameLocked ? t('persona.lockedNameTitle') : t('persona.editNameTitle')}
              >
                {personaNameLocked ? lockedPersonaName : personaName}
              </button>
            )}
            {editingJob ? (
              <input
                className="persona-job persona-job-input"
                autoFocus
                value={jobDraft}
                onChange={(event) => setJobDraft(event.target.value)}
                onBlur={savePersonaJob}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') savePersonaJob();
                  if (event.key === 'Escape') {
                    setJobDraft(personaJob);
                    setEditingJob(false);
                  }
                }}
              />
            ) : (
              <button
                className="persona-job"
                type="button"
                onDoubleClick={() => {
                  setJobDraft(personaJob);
                  setEditingJob(true);
                }}
                title={t('persona.editJobTitle')}
              >
                {personaJob}
              </button>
            )}
            <button
              className="persona-avatar-change"
              type="button"
              disabled={personaAvatarLocked}
              onClick={() => setAvatarPickerOpen((open) => !open)}
              title={personaAvatarLocked ? t('avatar.locked') : t('persona.changeAvatarTitle')}
            >
              {personaAvatarLocked ? t('persona.avatarLocked') : t('persona.changeAvatar')}
            </button>
            {avatarPickerOpen && !personaAvatarLocked && (
              <div className="avatar-provider-popover">
                {/* Avatar expression is a display choice only. It may preview
                    states, but must not feed persona prompts, memory, routing,
                    or risk decisions. */}
                <button type="button" onClick={() => onAvatarProviderSelect('user_image_api')}>
                  <strong>{t('persona.genApiLabel')}</strong>
                  <small>{t('persona.genApiHint')}</small>
                </button>
                <button type="button" onClick={() => onAvatarProviderSelect('static_image')}>
                  <strong>{t('persona.staticAvatarLabel')}</strong>
                  <small>{t('persona.staticAvatarHint')}</small>
                </button>
                <button type="button" onClick={() => onAvatarProviderSelect('built_in_pixel')}>
                  <strong>{t('persona.builtinAvatarLabel')}</strong>
                  <small>{t('persona.builtinAvatarHint')}</small>
                </button>
                <div className="avatar-state-grid" aria-label={t('persona.avatarStatePreview')}>
                  {avatarStateOptions.map((state) => (
                    <button
                      className={state === avatarExpression ? 'avatar-state-active' : ''}
                      type="button"
                      key={state}
                      onClick={() => onAvatarStateSelect(state)}
                    >
                      {avatarStateLabel(state)}
                    </button>
                  ))}
                  <button type="button" onClick={() => onAvatarStateSelect('')}>{t('persona.autoState')}</button>
                  <button
                    type="button"
                    className="pixel-pack-switch-btn"
                    onClick={() => setPixelPackPopup((v) => !v)}
                    title={t('persona.togglePixelPack')}
                  >
                    {t('persona.pixelPack')}
                  </button>
                </div>
                {/* ── Pixel Pack Switcher Popup ── */}
                {pixelPackPopup && (
                  <div className="pixel-pack-popup">
                    <div className="pixel-pack-popup-title">{t('persona.selectPixelPack')}</div>
                    <div className="pixel-pack-list">
                      {pixelPacks.map((p) => (
                        <button
                          key={p.id}
                          type="button"
                          className={`pixel-pack-card${p.id === currentPack ? ' pixel-pack-active' : ''}`}
                          onClick={() => switchPixelPack(p.id)}
                        >
                          {packPreviews[p.id] && (
                            <img
                              src={packPreviews[p.id]}
                              alt={p.name}
                              className="pixel-pack-preview"
                            />
                          )}
                          <span className="pixel-pack-name">{p.name}</span>
                          <span className="pixel-pack-desc">{p.desc}</span>
                        </button>
                      ))}
                    </div>
                    <button
                      type="button"
                      className="pixel-pack-close"
                      onClick={() => setPixelPackPopup(false)}
                    >
                      {t('persona.closePopup')}
                    </button>
                  </div>
                )}
                <p>{avatarModeNotice || t('persona.currentProvider', { provider: avatarProvider || 'built_in_pixel' })}</p>
              </div>
            )}
          </div>
        )}
      </div>
      {/* ── v3.6.3 Remote Bridge Channel Icons（§12A） ──
        *
        * 位置：TopConsole 黃框區域，persona-block 下方、status-row 上方。
        * 每個已註冊通道渲染一個文字 icon（TG / DC / LN）：
        *   - 灰色 = 未啟用（可點擊切換為 notification_only）
        *   - 亮色 + 光暈 = 已啟用
        * 互動：
        *   - 左鍵單擊 / 右鍵 / 長按 600ms = 開啟通道選單
        *   - 選單內可啟用/停用、切模式、測試、移除
        * 模式切換彈窗：
        *   - 僅通知 / 遠端提交任務 / 遠端審查 三選一
        *   - 底部「移除通道」按鈕
        */}
      {remoteBridgeChannels && remoteBridgeChannels.length > 0 && (
        <div className="remote-bridge-strip">
          {remoteBridgeChannels.map((ch) => {
            const modeLabels = {notification_only: t('remote.notificationOnly'), remote_task_submit: t('remote.remoteTaskSubmit'), remote_review: t('remote.remoteReview')};
            const isActive = ch.active;
            const displayName = ch.display_name || ch.displayName || ch.channel?.toUpperCase?.() || ch.channel;
            let longPressTimer = null;
            return (
              <span key={ch.id} className="remote-bridge-icon-wrap" style={{position: 'relative'}}>
                <button
                  className={`remote-bridge-icon ${isActive ? 'remote-bridge-icon-active' : ''}`}
                  type="button"
                  title={t('remote.channelTooltip', { name: displayName, mode: modeLabels[ch.mode] || ch.mode, active: isActive ? t('remote.activeLabel') : '' })}
                  onClick={() => onOpenRemoteBridgeMode(ch.id)}
                  onDoubleClick={(e) => {
                    e.preventDefault();
                    onRemoteBridgeRename?.(ch);
                  }}
                  onContextMenu={(e) => {
                    e.preventDefault();
                    onOpenRemoteBridgeMode(ch.id);
                  }}
                  onMouseDown={() => {
                    longPressTimer = setTimeout(() => onOpenRemoteBridgeMode(ch.id), 600);
                  }}
                  onMouseUp={() => clearTimeout(longPressTimer)}
                  onMouseLeave={() => clearTimeout(longPressTimer)}
                >
                  {displayName}
                </button>
                {/* Mode Switcher Popup */}
                {remoteBridgeModePopup === ch.id && (
                  <div className="remote-bridge-mode-popup">
                    <span className="remote-bridge-mode-title">{t('remote.modeSwitch', { channel: ch.channel?.toUpperCase?.() })}</span>
                    <div className="remote-bridge-main-row">
                      <button
                        className={`remote-bridge-mode-btn ${ch.active ? 'remote-bridge-mode-btn-active' : ''}`}
                        type="button"
                        onClick={() => onToggleRemoteBridge(ch.id)}
                      >
                        {ch.active ? t('remote.disableChannel') : t('remote.enableChannel')}
                      </button>
                      <button
                        className={`remote-bridge-mode-btn remote-bridge-primary-btn ${ch.primary ? 'remote-bridge-primary-btn-active' : ''}`}
                        type="button"
                        title={t('remote.primaryChannelHint')}
                        onClick={() => onMakeRemoteBridgePrimary?.(ch.id)}
                      >
                        {t('remote.mainChannel')}
                      </button>
                    </div>
                    <small className="remote-bridge-primary-hint">{t('remote.mainChannelHint')}</small>
                    {ch.channel === 'line' && (
                      <div className="remote-bridge-inbound-panel">
                        <strong>{t('remote.halfDuplex')}</strong>
                        <code>{remoteBridgeInboundInfo?.channel_id === ch.id ? remoteBridgeInboundInfo.local_url : t('remote.starting')}</code>
                        <small>{t('remote.webhookHint')}</small>
                        <div className="remote-bridge-secret-row">
                          <input
                            type="password"
                            value={remoteInboundSecretDraft}
                            onChange={(event) => setRemoteInboundSecretDraft(event.target.value)}
                            placeholder="Channel secret"
                          />
                          <button
                            type="button"
                            onClick={() => {
                              onSaveRemoteBridgeInboundSecret?.(ch.id, remoteInboundSecretDraft);
                              setRemoteInboundSecretDraft('');
                            }}
                          >
                            {t('remote.save')}
                          </button>
                        </div>
                      </div>
                    )}
                    {remoteBridgeInboundAdapters?.length > 0 && (
                      <small className="remote-bridge-inbound-status">
                        {t('remote.inboundAdapters', { adapters: remoteBridgeInboundAdapters.map((adapter) => `${adapter.label}${adapter.ready ? '✓' : '·'}`).join(' / ') })}
                      </small>
                    )}
                    {['notification_only', 'remote_task_submit', 'remote_review'].map((mode) => (
                      <button
                        key={mode}
                        className={`remote-bridge-mode-btn ${ch.mode === mode ? 'remote-bridge-mode-btn-active' : ''}`}
                        type="button"
                        onClick={() => onSwitchRemoteBridgeMode(ch.id, mode)}
                      >
                        {modeLabels[mode]}
                      </button>
                    ))}
                    <button
                      className="remote-bridge-mode-btn remote-bridge-mode-temp-test"
                      type="button"
                      title={t('remote.testSendTitle')}
                      onClick={() => onTemporaryRemoteBridgeTest?.(ch)}
                    >
                      {t('remote.testSend')}
                    </button>
                    <button className="remote-bridge-mode-btn remote-bridge-mode-remove" type="button" onClick={() => onRemoveRemoteBridge(ch.id)}>
                      {t('remote.removeChannel')}
                    </button>
                    <button className="remote-bridge-mode-btn" type="button" onClick={onCloseRemoteBridgeModePopup}>
                      {t('remote.close')}
                    </button>
                  </div>
                )}
              </span>
            );
          })}
        </div>
      )}
      <div className="status-row">
        <button className="greeting-pill" type="button" onClick={onRotateGreeting}>
          <span className="sun-mark">☀</span>
          <span>{greeting}</span>
        </button>
        <button className="interactive-btn" type="button" aria-label={t('remote.openInteractive')} title={t('remote.openInteractive')} onClick={() => setInteractiveOpen(true)}>
          <span>{t('remote.openInteractive')}</span>
        </button>
      </div>
      {interactiveOpen && (
        <div className="interactive-popover" role="dialog" aria-label={t('remote.interactiveLabel')}>
          <header className="interactive-popover-header">
            <strong>{t('remote.interactiveHeader')}</strong>
            <select
              aria-label={t('remote.selectAdapter')}
              className="interactive-adapter-select"
              value={activeAdapter?.id || activeAdapterId || ''}
              onChange={(event) => onAdapterSelect?.(event.target.value || null)}
            >
              {(adapterOptions || []).length === 0 && (
                <option value="">{t('remote.noAdapterSelected')}</option>
              )}
              {(adapterOptions || []).map((adapter) => (
                <option key={adapter.id || adapter.name} value={adapter.id || adapter.name}>
                  {adapter.name || adapter.id}
                </option>
              ))}
            </select>
          </header>
          <textarea
            autoFocus
            placeholder={t('remote.interactivePlaceholder')}
            value={interactiveText}
            onChange={(event) => setInteractiveText(event.target.value)}
          />
          {cliInspectorLog && (
            <section className="cli-inspector-log" aria-label={t('remote.cliLog')}>
              <label>payload</label>
              <pre>{JSON.stringify(cliInspectorLog.payload, null, 2)}</pre>
              {cliInspectorLog.response && (
                <>
                  <label>response</label>
                  <pre>{JSON.stringify(cliInspectorLog.response, null, 2)}</pre>
                </>
              )}
              {cliInspectorLog.error && <p className="cli-inspector-error">{cliInspectorLog.error}</p>}
            </section>
          )}
          <div className="interactive-popover-actions">
            <button type="button" onClick={() => { setInteractiveOpen(false); callWails(() => ClearInspectorHistory()).catch(() => {}); }}>{t('remote.cancel')}</button>
            <button
              type="button"
              disabled={cliInspectorBusy || !interactiveText.trim()}
              onClick={async () => {
                const submitted = interactiveText.trim();
                if (!submitted) return;
                await onCLIInspectSend?.(submitted);
                setInteractiveText('');
              }}
            >
              {cliInspectorBusy ? t('remote.sending') : t('remote.send')}
            </button>
          </div>
        </div>
      )}
      <div className="haora-band">
        <div className="haora-scroll">
          {haoraItems.map((haora) => {
            const isActiveHaora = haora.isMain
              ? !haoraItems.some((item) => !item.isMain && (item.id === activeHaoraId || item.name === activeHaoraId))
              : (haora.id === activeHaoraId || haora.name === activeHaoraId);
            const isEditingHaora = editingHaoraKey === haora.key;
            if (isEditingHaora) {
              return (
                <div
                  className={`haora-card haora-card-editing ${haora.isMain ? 'haora-card-main' : ''} ${isActiveHaora ? 'haora-card-active' : ''}`}
                  key={haora.key}
                  onClick={(event) => event.stopPropagation()}
                >
                  <input
                    className="haora-name-input"
                    autoFocus
                    value={haoraNameDraft}
                    onChange={(event) => setHaoraNameDraft(event.target.value)}
                    onBlur={() => commitInlineHaoraRename(haora)}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter') {
                        event.preventDefault();
                        commitInlineHaoraRename(haora);
                      }
                      if (event.key === 'Escape') {
                        event.preventDefault();
                        cancelInlineHaoraRename();
                      }
                    }}
                  />
                  {isActiveHaora && <i aria-hidden="true"/>}
                </div>
              );
            }
            return (
              <button
                className={`haora-card ${haora.isMain ? 'haora-card-main' : ''} ${isActiveHaora ? 'haora-card-active' : ''} ${draggedHaoraKey === haora.key ? 'haora-card-dragging' : ''}`}
                type="button"
                key={haora.key}
                data-haora-key={haora.key}
                draggable={false}
                title={haora.isMain
                  ? t('subagent.switchToMain')
                  : nativeSubDragSupported
                    ? t('subagent.haoraTitle')
                    : (subExportCapabilities?.message || t('subagent.haoraTitle'))}
                onClick={(event) => {
                  if (haoraPointerDragRef.current?.moved) {
                    event.preventDefault();
                    return;
                  }
                  onHaoraSelect?.(haora);
                }}
                onDragStart={(event) => event.preventDefault()}
                onPointerDown={(event) => {
                  const draggable = !haora.isMain && nativeSubDragSupported;
                  markHaoraDragDebug('pointerdown', haora, `native=${nativeSubDragSupported ? 'yes' : 'no'} draggable=${draggable ? 'yes' : 'no'}`);
                  startHaoraPointerDrag(event, haora, {skipDebug: true});
                }}
                onPointerMove={handleHaoraPointerMove}
                onPointerUp={finishHaoraPointerDrag}
                onPointerCancel={finishHaoraPointerDrag}
                onLostPointerCapture={finishHaoraDrag}
                onDoubleClick={(event) => startInlineHaoraRename(event, haora)}
              >
                <span>{haora.name}</span>
                {/* 亮點只跟著目前 active 的 haㄌer/sub。 */}
                {isActiveHaora && <i aria-hidden="true"/>}
              </button>
            );
          })}
        </div>
      </div>
      {haoraDragPreview && (
        <div
          className="haora-drag-preview"
          style={{
            transform: `translate(${Math.round(haoraDragPreview.x)}px, ${Math.round(haoraDragPreview.y)}px) translate(12px, -50%)`,
          }}
          aria-hidden="true"
        >
          {haoraDragPreview.name}
        </div>
      )}
      {haoraExportDialog && (
        <DragActionModal
          ariaLabel={t('subagent.dragTitle')}
          icon="ha"
          title={haoraExportDialog.name}
          detail={haoraExportDialog.landedDetail || haoraExportDialog.landedPath}
          actions={haoraExportDialog.nativeFailed
            ? [{label: t('common.close'), onClick: () => setHaoraExportDialog(null)}]
            : [
                {label: t('adapter.remove'), onClick: () => handleHaoraExportAction('remove')},
                {label: t('adapter.copyAction'), onClick: () => handleHaoraExportAction('copy')},
                {label: t('common.cancel'), onClick: () => handleHaoraExportAction('cancel')},
              ]}
        />
      )}
      <div className="top-function-stack">
        <button
          className="system-strip"
          type="button"
          onClick={() => {
            setPromptText(t('persona.promptDefault'));
            setPromptOpen(true);
          }}
        >
          <span>✦</span>
          <span>{t('persona.showSystemPrompt')}</span>
        </button>
        {promptOpen && (
          <div className="prompt-popover top-prompt-popover">
            <div className="prompt-dragbar">
              <span>system prompt</span>
              <button type="button" onClick={() => setPromptOpen(false)}>{t('persona.closePrompt')}</button>
            </div>
            <textarea value={promptText} onChange={(event) => setPromptText(event.target.value)}/>
          </div>
        )}
        {/* #I-1001: 系統狀態歷史紀錄 — 自動封存等事件留存於此供後續查閱 */}
        {systemStatusHistory.length > 0 && (
          <div className="system-status-history">
            <span className="system-status-history-label">{t('persona.systemStatus')}</span>
            {systemStatusHistory.slice(-5).map((entry, i) => (
              <div key={i} className="system-status-entry">
                <span className="system-status-time">{new Date(entry.time).toLocaleTimeString()}</span>
                <span className="system-status-text">{entry.text}</span>
              </div>
            ))}
          </div>
        )}
        <div className="review-strip">
          <span className="review-strip-label">Review Panel</span>
          <span className="review-strip-hint">{t('review.highRiskTitle')} · Skill Activity · Pending Digest</span>
          <span className={`review-strip-status review-status-${reviewState.highRisk.status}`}>
            {reviewStatusLabel(reviewState.highRisk.status)}
          </span>
        </div>
        <ReviewPanel
          activePopup={reviewPopup}
          dagRun={dagRun}
          onPopupChange={onReviewPopupChange}
          onSkillSelect={onSkillSelect}
          onSnooze={onSnooze}
          onSnoozeHoursChange={onSnoozeHoursChange}
          reviewState={reviewState}
          snoozeHours={snoozeHours}
          onAcknowledgeDigestItem={onAcknowledgeDigestItem}
          onConfirmSkillBuild={onConfirmSkillBuild}
          reviewArchive={reviewArchive}
          w3aImportPopup={w3aImportPopup}
          w3aDetail={w3aDetail}
          w3aPollutionResult={w3aPollutionResult}
          w3aTransferGuidance={w3aTransferGuidance}
          w3aTrustList={w3aTrustList}
          w3aActionBusy={w3aActionBusy}
          w3aActionError={w3aActionError}
          w3aStatusConfig={w3aStatusConfig}
          w3aToastMsg={w3aToastMsg}
          onLoadW3AInfo={onLoadW3AInfo}
          onDetectW3APollution={onDetectW3APollution}
          onShowW3AGuidance={onShowW3AGuidance}
          onTrustW3ADeveloper={onTrustW3ADeveloper}
          onExportW3ACopy={onExportW3ACopy}
          onDismissW3AImportPopup={onDismissW3AImportPopup}
          onShowW3AToast={onShowW3AToast}
          onDismissW3AToast={onDismissW3AToast}
        />
        {skillInjections.length > 0 && (
          <SkillActivityCard injections={skillInjections} />
        )}
        {/* #I-207: 初次使用說明卡，注入成功且尚未顯示過時出現 */}
        {showSkillFirstUseCard && (
          <SkillFirstUseCard onDismiss={onDismissSkillFirstUse} />
        )}
      </div>
    </section>
  );
}

// ──────────────────────────────────────────────────────────────
// v3.6.4 Readiness Gate UI Interaction Layer — React 元件群
// ──────────────────────────────────────────────────────────────
//
// 本區塊包含 Readiness Gate 前端互動層的所有 React 元件：
//   - LongPressConfirmButton : 長按確認按鈕（含進度條 + 轉蛋動畫）
//   - FloatingCandidateActions : 浮動意圖候選按鈕（最多 3 個）
//   - MissingSlotCapsule : 缺欄位膠囊提示
//   - RetrievalTransparency : 掃描動畫 + 來源標籤
//   - ConfirmationTier : 三層確認機制容器
//
// 設計原則：
//   所有元素掛在輸入框正上方（conversation 底部），操作完成後消失刷新。
//   UI 像高級咖啡店一樣簡潔；動畫是煙火，不是護照。

/**
 * LongPressConfirmButton — 長按確認按鈕
 *
 * §11.2/§11.3: 中風險與高風險確認使用長按按鈕。
 * 長按過程顯示進度條，完成後觸發轉蛋動畫連鎖：
 *   button collapse → outward pulse → particle burst → capsule reveal →「執行中」
 *
 * Props:
 *   label        — 按鈕文字（預設「確認」）
 *   danger       — 是否為高風險紅色狀態
 *   progress     — 長按進度百分比（0–100）
 *   gachaPhase   — 轉蛋動畫階段（null | 'collapse' | 'pulse' | 'particles' | 'reveal'）
 *   onPressStart — 開始按住回呼
 *   onPressEnd   — 放開/取消回呼
 */
function LongPressConfirmButton({label, danger = false, progress = 0, gachaPhase = null, onPressStart, onPressEnd}) {
  // 轉蛋動畫完成 → 顯示「執行中」膠囊
  if (gachaPhase === 'reveal') {
    return (
      <span className="gacha-reveal">
        <span className="gacha-spinner" />
        {_t('confirm.execute')}
      </span>
    );
  }

  // 轉蛋動畫中間階段 → 顯示特效容器
  if (gachaPhase === 'collapse' || gachaPhase === 'pulse' || gachaPhase === 'particles') {
    return (
      <span style={{position: 'relative', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', minWidth: 120, minHeight: 44}}>
        {/* collapse 階段：按鈕縮小 */}
        {gachaPhase === 'collapse' && (
          <button type="button" className={`long-press-btn lp-completing ${danger ? 'lp-danger' : ''}`}>
            {label}
          </button>
        )}
        {/* pulse 階段：脈衝環向外擴散 */}
        {gachaPhase === 'pulse' && <span className="gacha-pulse" />}
        {/* particles 階段：8 個粒子向外散射 */}
        {gachaPhase === 'particles' && (
          <span className="gacha-particles">
            {PARTICLE_ANGLES.map((angle) => (
              <span
                key={angle}
                className="gacha-particle"
                style={{
                  transform: `translate(${Math.cos(angle * Math.PI / 180) * 30}px, ${Math.sin(angle * Math.PI / 180) * 30}px)`,
                }}
              />
            ))}
          </span>
        )}
      </span>
    );
  }

  // 正常狀態 → 可長按的確認按鈕
  return (
    <button
      type="button"
      className={`long-press-btn ${danger ? 'lp-danger' : ''}`}
      onMouseDown={onPressStart}
      onMouseUp={onPressEnd}
      onMouseLeave={onPressEnd}
      onTouchStart={onPressStart}
      onTouchEnd={onPressEnd}
      onTouchCancel={onPressEnd}
    >
      {label}
      {/* 長按進度條：從左到右填滿 */}
      <span className="lp-progress" style={{width: `${progress}%`}} />
    </button>
  );
}

/**
 * FloatingCandidateActions — 浮動意圖候選按鈕
 *
 * §12A.1: 當 Readiness Gate 偵測到模糊使用者意圖時，
 * 在輸入框正上方顯示最多 3 個候選按鈕。
 * 使用者點擊後填入 draft 並觸發重新評估，送出新訊息後全部消失。
 *
 * Props:
 *   candidates — 候選陣列 [{id, label, draft}]
 *   onSelect   — 點擊候選回呼 (candidateID) => void
 */
function FloatingCandidateActions({candidates = [], onSelect}) {
  if (!candidates.length) return null;
  return (
    <div className="floating-candidates">
      {candidates.slice(0, 3).map((candidate) => (
        <button
          key={candidate.id}
          type="button"
          className="floating-candidate-btn"
          onClick={() => onSelect(candidate.id)}
        >
          {candidate.label}
        </button>
      ))}
    </div>
  );
}

/**
 * MissingSlotCapsule — 缺欄位膠囊提示
 *
 * §12A.3: 當必填欄位仍缺時，以精簡膠囊顯示缺少項目。
 * 每次使用者送出訊息後更新，所有欄位補齊後消失。
 * 高風險任務的膠囊帶有警告色（.slot-warning）。
 *
 * Props:
 *   missingSlots — 缺少欄位名稱陣列 ['資料夾', '檔案類型', '時間條件']
 *   isHighRisk   — 是否為高風險任務（影響膠囊顏色）
 */
function MissingSlotCapsule({missingSlots = [], isHighRisk = false}) {
  const {t} = useI18n();
  if (!missingSlots.length) return null;
  return (
    <span className={`missing-slot-capsule ${isHighRisk ? 'slot-warning' : ''}`}>
      <span className="slot-label">{t('slot.missing')}</span>
      <span className="slot-items">{missingSlots.join('、')}</span>
    </span>
  );
}

/**
 * RetrievalTransparency — 掃描動畫 + 來源標籤
 *
 * §12A.6: 當 RETRIEVE_MORE 被觸發時，在對話訊息流中 inline 顯示。
 * 預設模式僅顯示粗分類來源類別（project docs、uploaded files 等）。
 * 標籤必須通過 Safe Display Filter，禁止顯示完整路徑或敏感檔名。
 *
 * Props:
 *   isScanning — 是否正在掃描中
 *   sources    — 來源標籤陣列 ['project docs', 'uploaded files']
 */
function RetrievalTransparency({isScanning = false, sources = []}) {
  if (!isScanning) return null;
  return (
    <div className="retrieval-scanning">
      <span className="scan-dot" />
      <span className="scan-label">scanning:</span>
      {sources.map((source) => (
        <span key={source} className="scan-source">{source}</span>
      ))}
    </div>
  );
}

/**
 * ConfirmationTier — 三層確認機制容器
 *
 * §11.2: 依風險等級切換不同確認 UI：
 *   第一層（普通風險）→ 對話式 [是] [否] ▸ 自行輸入
 *   第二層（中風險）  → 長按確認 + 轉蛋動畫
 *   第三層（高風險）  → 影響說明面板 → 變紅 → 長按確認 + 轉蛋動畫
 *
 * Props:
 *   riskTier          — 風險等級 'none' | 'normal' | 'medium' | 'high'
 *   impactExplanation — 高風險影響說明文字
 *   impactExpanded    — 高風險影響說明面板是否已展開
 *   longPressProgress — 長按進度百分比
 *   gachaPhase        — 轉蛋動畫階段
 *   onNormalConfirm   — 普通風險確認回呼
 *   onNormalReject    — 普通風險拒絕回呼
 *   onHighRiskYes     — 高風險點 [是] 回呼（展開說明面板）
 *   onPressStart      — 長按開始回呼
 *   onPressEnd        — 長按結束回呼
 */
function ConfirmationTier({
  riskTier = 'none', impactExplanation = '', impactExpanded = false,
  longPressProgress = 0, gachaPhase = null,
  onNormalConfirm, onNormalReject, onHighRiskYes, onPressStart, onPressEnd,
}) {
  const {t} = useI18n();
  if (riskTier === 'none') return null;

  // 第一層：普通風險 — 對話式 [是] [否] ▸ 自行輸入
  if (riskTier === 'normal') {
    return (
      <div className="confirmation-tier composer composer-risk">
        <button type="button" className="risk-confirm-btn risk-yes" onClick={onNormalConfirm}>{t('confirm.yes')}</button>
        <button type="button" className="risk-confirm-btn risk-no" onClick={onNormalReject}>{t('confirm.no')}</button>
        <div className="risk-custom-wrap">
          <span>{t('confirm.customInput')}</span>
          <input type="text" placeholder={t('confirm.customPlaceholder')} />
        </div>
      </div>
    );
  }

  // 第二層：中風險（Readiness Gate）— 長按確認
  if (riskTier === 'medium') {
    return (
      <div className="confirmation-tier">
        <LongPressConfirmButton
          label={t('confirm.longPressConfirm')}
          progress={longPressProgress}
          gachaPhase={gachaPhase}
          onPressStart={onPressStart}
          onPressEnd={onPressEnd}
        />
        <button type="button" className="risk-confirm-btn risk-no" onClick={onNormalReject}>{t('confirm.cancel')}</button>
      </div>
    );
  }

  // 第三層：高風險 — 影響說明 → 變紅 → 長按確認
  if (riskTier === 'high') {
    return (
      <div style={{display: 'flex', flexDirection: 'column', gap: 8}}>
        {/* 未展開時：顯示 [是] [否] */}
        {!impactExpanded && (
          <div className="confirmation-tier composer composer-risk">
            <button type="button" className="risk-confirm-btn risk-yes" onClick={onHighRiskYes}>{t('confirm.yes')}</button>
            <button type="button" className="risk-confirm-btn risk-no" onClick={onNormalReject}>{t('confirm.no')}</button>
            <div className="risk-custom-wrap">
              <span>{t('confirm.customInput')}</span>
              <input type="text" placeholder={t('confirm.customPlaceholder')} />
            </div>
          </div>
        )}
        {/* 展開後：顯示影響說明 + 紅色長按確認按鈕 */}
        {impactExpanded && (
          <>
            <div className="risk-impact-panel">
              <div className="impact-title">{t('confirm.willCause')}</div>
              <div className="impact-list">
                {(impactExplanation || t('confirm.highRiskFallback')).split('\n').map((line, i) => (
                  <span key={i} className="impact-item">{line}</span>
                ))}
              </div>
            </div>
            <div className="confirmation-tier">
              <LongPressConfirmButton
                label={t('confirm.longPressExecute')}
                danger
                progress={longPressProgress}
                gachaPhase={gachaPhase}
                onPressStart={onPressStart}
                onPressEnd={onPressEnd}
              />
              <button type="button" className="risk-confirm-btn risk-no" onClick={onNormalReject}>{t('confirm.cancel')}</button>
            </div>
          </>
        )}
      </div>
    );
  }

  return null;
}

// ── 原有 ConversationPanel ──

function MicIcon() {
  return (
    <svg className="mic-icon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 3.5c-1.7 0-3 1.3-3 3v5c0 1.7 1.3 3 3 3s3-1.3 3-3v-5c0-1.7-1.3-3-3-3Z" />
      <path d="M6.5 10.5c0 3 2.4 5.5 5.5 5.5s5.5-2.5 5.5-5.5M12 16v3.5M9 20.5h6" />
    </svg>
  );
}

// SEC-06: 訊息內 URL 偵測（與後端 urlInTextRe 對齊）。
const MESSAGE_URL_RE = /https?:\/\/[^\s'"<>）)】\]]+/g;

// MessageText 把訊息文字 linkify：每個 URL 旁加來源 chip + 「讀取」按鈕。
// AI 訊息內的 URL 在渲染時記為 llm_extracted（閉合洗白防護）；
// 使用者貼上的 URL 後端送出時已記 user_paste，前端不重複記。
function MessageText({ text, kind, onInjectText, sessionId }) {
  const urls = React.useMemo(() => {
    const found = text.match(MESSAGE_URL_RE);
    return found ? Array.from(new Set(found)) : [];
  }, [text]);

  // 閉合洗白防護：AI 回答裡的 URL 記成 llm_extracted。
  React.useEffect(() => {
    if (kind !== 'ai' || urls.length === 0) return;
    urls.forEach((u) => {
      try {
        RegisterURLOccurrence(u, 'llm_extracted', sessionId || '', '').catch(() => {});
      } catch { /* binding 缺失（測試環境）忽略 */ }
    });
  }, [kind, urls, sessionId]);

  if (urls.length === 0) return <>{text}</>;

  // 以 URL 切段，URL 段渲染成連結 + chip + 讀取按鈕。
  const parts = text.split(MESSAGE_URL_RE);
  const matches = text.match(MESSAGE_URL_RE) || [];
  return (
    <>
      {parts.map((seg, i) => (
        <React.Fragment key={i}>
          {seg}
          {i < matches.length && (
            <span className="url-token">
              <a
                href={matches[i]}
                onClick={(e) => { e.preventDefault(); onInjectText?.(matches[i]); }}
                title="點擊選擇讀取或開啟"
              >{matches[i]}</a>
              {kind === 'ai' && <span className="url-source-chip url-source-llm" title="此網址由 LLM 擷取，內容未經信任">LLM 擷取</span>}
              <button
                type="button"
                className="url-read-btn"
                onClick={(e) => { e.stopPropagation(); onInjectText?.(`讀取 ${matches[i]} 的內容`); }}
              >讀取內容</button>
            </span>
          )}
        </React.Fragment>
      ))}
    </>
  );
}

function ConversationPanel({
  messages, personaName, draft, onDraftChange, onSend, onDelete, onSummarizeSearch,
  onInjectText, activeConversationId,
  // v3.6.4 Readiness Gate props
  readinessGate = fallbackReadinessGate,
  longPressProgress = 0, gachaPhase = null, riskImpactExpanded = false,
  onSelectCandidate, onNormalConfirm, onNormalReject, onHighRiskYes,
  onLongPressStart, onLongPressEnd,
  // §12A.5B Dispatch 狀態
  dispatchStatus = {},
  voiceState = null, voiceRecording = false, voiceBusy = false, voiceStatus = '', voiceError = '',
  onVoicePressStart, onVoicePressEnd, onVoiceCancel,
  taskActive = false, onCancelTask,
  pendingTaskReview = null, taskReviewDetailsOpen = false,
  onConfirmTaskReview, onCancelTaskReview, onShowTaskReviewDetails,
}) {
  const t = useI18n(s => s.t);
  const [activeMessage, setActiveMessage] = useState(null);
  const composerComposingRef = useRef(false);
  const taskReviewCardRef = useRef(null);
  const voiceReady = voiceState?.status === 'ready';
  const micAvailable = typeof navigator !== 'undefined' && !!navigator.mediaDevices?.getUserMedia;
  const voiceDisabled = voiceBusy || !voiceReady || !micAvailable;
  const voiceTitle = voiceReady ? t('composer.voiceHold') : voiceStatusLabel(voiceState?.status);
  const displayMessage = (message) => stripComposerPendingMarker(message)
    .replace(/^Ai:/, personaName + ':')
    .replace(/^輸入:/, '');

  const messageKind = (message) => {
    if (message.startsWith('Ai:')) return 'ai';
    return 'user';
  };
  const isLocalSearchMessage = (message) => message.startsWith('Ai:本機搜尋');

  useEffect(() => {
    if (!pendingTaskReview?.id) return;
    window.requestAnimationFrame(() => {
      taskReviewCardRef.current?.scrollIntoView({block: 'center', behavior: 'smooth'});
    });
  }, [pendingTaskReview?.id]);

  // SEC-06 UX: 新訊息進來（含「讀取內容」按鈕注入）時自動捲到底，
  // 讓使用者立刻看到最新狀態，不會以為點了沒反應。只在訊息變多時捲。
  const messageListRef = useRef(null);
  const prevMessageCountRef = useRef(messages.length);
  useEffect(() => {
    if (messages.length > prevMessageCountRef.current) {
      window.requestAnimationFrame(() => {
        const el = messageListRef.current;
        if (el) el.scrollTo({top: el.scrollHeight, behavior: 'smooth'});
      });
    }
    prevMessageCountRef.current = messages.length;
  }, [messages.length]);

  return (
    <section className="conversation-panel">
      <div className="message-list" ref={messageListRef}>
        {messages.map((message, index) => (
          <article
            className={`message-row message-${messageKind(message)} ${activeMessage === index ? 'message-row-active' : ''}`}
            key={`${message}-${index}`}
            onClick={() => setActiveMessage(index)}
          >
            <div className="message-text">
              <MessageText
                text={displayMessage(message)}
                kind={messageKind(message)}
                onInjectText={onInjectText}
                sessionId={activeConversationId}
              />
            </div>
            {isLocalSearchMessage(message) && (
              <button
                type="button"
                className="message-action"
                onClick={(event) => {
                  event.stopPropagation();
                  onSummarizeSearch?.(message.replace(/^Ai:/, ''));
                }}
              >
                {t('composer.summary')}
              </button>
            )}
            <button
              type="button"
              onClick={(event) => {
                event.stopPropagation();
                onDelete(index);
                setActiveMessage(null);
              }}
            >
              {t('composer.delete')}
            </button>
          </article>
        ))}
        {pendingTaskReview && (
          <article className="task-review-inline-card" ref={taskReviewCardRef}>
            <header>
              <strong>需要你確認這一步</strong>
              <span>待確認</span>
            </header>
            <div className="task-review-inline-grid">
              <div>
                <small>這一步要做什麼</small>
                <p>{pendingTaskReview.title}</p>
              </div>
              <div>
                <small>為什麼需要確認</small>
                <p>{pendingTaskReview.reason}</p>
              </div>
            </div>
            <p className="task-review-inline-impact">會影響：{pendingTaskReview.impact}</p>
            <p className="task-review-inline-impact">使用的工具/模型：{pendingTaskReview.tool}</p>
            <footer>
              <button type="button" className="task-review-detail-btn" onClick={onShowTaskReviewDetails}>
                {taskReviewDetailsOpen ? '關閉' : '查看內容'}
              </button>
              <button type="button" className="task-review-cancel-btn" onClick={onCancelTaskReview}>取消</button>
              <button
                type="button"
                className="task-review-confirm-btn"
                onClick={() => onConfirmTaskReview?.(pendingTaskReview.id)}
              >
                確認執行
              </button>
            </footer>
          </article>
        )}
      </div>

      {/* ── v3.6.4 Readiness Gate 輸入框上方區域 ──
          §12A 佈局：所有 Readiness Gate 暫態 UI 元素掛在 composer 上方。
          由上而下順序：RetrievalTransparency → FloatingCandidateActions → MissingSlotCapsule → ConfirmationTier
          操作完成後消失刷新，不佔用固定空間。 */}
      {(readinessGate.retrieval_scanning
        || readinessGate.floating_candidates?.length > 0
        || readinessGate.missing_slots?.length > 0
        || (readinessGate.risk_tier && readinessGate.risk_tier !== 'none')
      ) && (
        <div className="readiness-above-composer">
          {/* §12A.6: Retrieval Transparency — 掃描動畫在對話訊息流 inline 顯示 */}
          <RetrievalTransparency
            isScanning={readinessGate.retrieval_scanning}
            sources={readinessGate.retrieval_sources || []}
          />
          {/* §12A.1: Floating Candidate Actions — 最多 3 個意圖候選 */}
          <FloatingCandidateActions
            candidates={readinessGate.floating_candidates || []}
            onSelect={onSelectCandidate}
          />
          {/* §12A.3: Missing Slot Capsule — 缺欄位膠囊 */}
          <MissingSlotCapsule
            missingSlots={readinessGate.missing_slots || []}
            isHighRisk={readinessGate.risk_tier === 'high'}
          />
          {/* §11.2: 三層確認機制 — 依風險等級切換 */}
          <ConfirmationTier
            riskTier={readinessGate.risk_tier}
            impactExplanation={readinessGate.impact_explanation}
            impactExpanded={riskImpactExpanded}
            longPressProgress={longPressProgress}
            gachaPhase={gachaPhase}
            onNormalConfirm={onNormalConfirm}
            onNormalReject={onNormalReject}
            onHighRiskYes={onHighRiskYes}
            onPressStart={onLongPressStart}
            onPressEnd={onLongPressEnd}
          />
        </div>
      )}

      {/* ── 原有 Composer（聊天輸入區）── */}
      <form className="composer" onSubmit={onSend}>
        <button
          className={`attach-btn task-stop-btn ${taskActive ? 'task-stop-active' : ''}`}
          type="button"
          title={taskActive ? '停止目前任務' : '目前沒有執行中的任務'}
          aria-label={taskActive ? '停止目前任務' : '目前沒有執行中的任務'}
          disabled={!taskActive}
          onClick={() => onCancelTask?.()}
        >
          <span aria-hidden="true">■</span>
        </button>
        <button
          className={`voice-btn ${voiceRecording ? 'voice-btn-recording' : ''}`}
          type="button"
          title={voiceTitle}
          aria-label={voiceTitle}
          disabled={voiceDisabled}
          onPointerDown={(event) => {
            event.preventDefault();
            event.currentTarget.setPointerCapture?.(event.pointerId);
            onVoicePressStart?.();
          }}
          onPointerUp={(event) => {
            event.preventDefault();
            event.currentTarget.releasePointerCapture?.(event.pointerId);
            onVoicePressEnd?.();
          }}
          onPointerCancel={() => onVoiceCancel?.()}
        >
          <MicIcon />
        </button>
        <div className="input-wrap">
          <textarea
            value={draft}
            rows={1}
            placeholder={t('composer.placeholder')}
            onChange={(event) => onDraftChange(event.target.value)}
            onCompositionStart={() => {
              // 中文/日文 IME 組字期間，Enter 先保留給選字。
              composerComposingRef.current = true;
            }}
            onCompositionEnd={() => {
              composerComposingRef.current = false;
            }}
            onKeyDown={(event) => {
              const composing = composerComposingRef.current
                || event.isComposing
                || event.nativeEvent?.isComposing
                || event.nativeEvent?.keyCode === 229;
              if (event.key === 'Enter' && !event.shiftKey && !composing) {
                event.preventDefault();
                event.currentTarget.form.requestSubmit();
              }
            }}
          />
          <button className="send-btn" type="submit"><span>◢</span>{t('composer.send')}</button>
        </div>
        {(voiceStatus || voiceError || voiceState?.status !== 'ready') && (
          <div className={`voice-status ${voiceError ? 'voice-status-error' : ''}`}>
            {voiceError || voiceStatus || voiceStatusLabel(voiceState?.status)}
          </div>
        )}
        {/* §12A.5B Dispatch 狀態提示 */}
        {Object.values(dispatchStatus).length > 0 && (
          <div className="dispatch-status-area">
            {Object.values(dispatchStatus).map((ds) => {
              if (!ds) return null;
              if (ds.done && ds.overall === 'success') {
                return <span key={ds.dispatchId} className="dispatch-status dispatch-success">{t('composer.sent')}</span>;
              }
              if (ds.done && ds.overall === 'partial_fail') {
                const failedSegs = (ds.segments || []).filter(s => s.error);
                return (
                  <span key={ds.dispatchId} className="dispatch-status dispatch-partial-fail">
                    {t('composer.sentPartialFail', { success: `${(ds.segments || []).length - failedSegs.length}/${(ds.segments || []).length}`, failed: failedSegs.map(s => s.part_index + 1).join(',') })}
                    <button className="dispatch-retry-btn" type="button" onClick={() => {}}>{t('composer.retry')}</button>
                  </span>
                );
              }
              if (ds.done && ds.overall === 'failed') {
                return (
                  <span key={ds.dispatchId} className="dispatch-status dispatch-fail">
                    {t('composer.sendFailed')}
                    <button className="dispatch-retry-btn" type="button" onClick={() => {}}>{t('composer.retry')}</button>
                  </span>
                );
              }
              return <span key={ds.dispatchId} className="dispatch-status dispatch-sending">{t('composer.sending', { current: ds.partIndex + 1, total: ds.totalParts })}</span>;
            })}
          </div>
        )}
      </form>
    </section>
  );
}

// Review details are portaled to the viewport so conversation panels never cover them.
function ReviewPanel({
  activePopup, dagRun, onPopupChange, reviewState, onSkillSelect, onSnooze,
  onSnoozeHoursChange, snoozeHours, onAcknowledgeDigestItem, onConfirmSkillBuild,
  reviewArchive, w3aImportPopup, w3aDetail, w3aPollutionResult, w3aTransferGuidance, w3aTrustList = [],
  w3aActionBusy = '', w3aActionError = '', w3aStatusConfig = {}, w3aToastMsg,
  onLoadW3AInfo = () => {}, onDetectW3APollution = () => {}, onShowW3AGuidance = () => {},
  onTrustW3ADeveloper = () => {}, onExportW3ACopy = () => {},
  onDismissW3AImportPopup = () => {}, onShowW3AToast = () => {}, onDismissW3AToast = () => {},
}) {
  const t = useI18n(s => s.t);
  const {highRisk, lowRiskAmbiguity, hookCandidates, pendingDigest, pendingPackages} = reviewState;
  const ambiguityHidden = Boolean(lowRiskAmbiguity.dismissedUntil);
  const panelRef = useRef(null);
  const [popupFrame, setPopupFrame] = useState(null);
  const [digestExpanded, setDigestExpanded] = useState({urgent: true, later: false, archive: false});
  const [taskDebugDump, setTaskDebugDump] = useState(null);
  const activeW3AInfo = w3aDetail || w3aImportPopup?.info || {};
  const activeW3ATraining = activeW3AInfo.training || {};
  const activeW3APollution = w3aPollutionResult || activeW3AInfo.pollution;
  const activeW3APath = w3aImportPopup?.source_path || activeW3AInfo.file_path || '';
  const activeW3ASidecar = w3aImportPopup?.sidecar_path || '';

  // Count total hook candidates for badge
  const hookCount = (hookCandidates?.tagPatches?.length || 0)
    + (hookCandidates?.subagentCandidates?.length || 0)
    + (hookCandidates?.registryProposals?.length || 0);
  const packageCount = pendingPackages?.length || 0;

  useEffect(() => {
    if (!activePopup) return undefined;

    const updatePopupFrame = () => {
      const rect = panelRef.current?.getBoundingClientRect();
      if (!rect) return;

      const viewportGap = 24;
      const preferredWidth = activePopup === 'skills' ? 460 : activePopup === 'dag' ? 720 : 640;
      const width = Math.min(preferredWidth, Math.max(280, rect.width - 8), window.innerWidth - viewportGap * 2);
      const left = activePopup === 'skills'
        ? Math.min(Math.max(viewportGap, rect.right - width), window.innerWidth - width - viewportGap)
        : Math.min(Math.max(viewportGap, rect.left), window.innerWidth - width - viewportGap);

      setPopupFrame({
        left,
        top: Math.min(rect.bottom + 8, window.innerHeight - 120),
        width,
      });
    };

    updatePopupFrame();
    window.addEventListener('resize', updatePopupFrame);
    window.addEventListener('scroll', updatePopupFrame, true);

    return () => {
      window.removeEventListener('resize', updatePopupFrame);
      window.removeEventListener('scroll', updatePopupFrame, true);
    };
  }, [activePopup]);

  useEffect(() => {
    setTaskDebugDump(null);
  }, [dagRun?.id]);

  // I-1: Categorize pending digest items into three UI blocks
  const digestBlocks = categorizePendingDigest(pendingDigest);
  const dagSummaryCount = dagRun?.summaries?.length || 0;
  const pendingDigestCount = (pendingDigest?.items || []).length;
  const pendingTotalCount = hookCount + packageCount + dagSummaryCount + pendingDigestCount;

  return (
    <section className="review-panel" aria-label="Review Panel" ref={panelRef}>
      <div className="review-summary-row">
        <button
          className={`review-summary-btn review-summary-dag ${['blocked', 'waiting_review'].includes(dagRun?.status) ? 'review-summary-hot' : ''}`}
          type="button"
          onClick={() => onPopupChange(activePopup === 'dag' ? null : 'dag')}
        >
          <span>任務進度</span>
          <strong>{dagRun ? dagStatusLabel(dagRun.status) : t('dag.standby')}</strong>
        </button>
        <button
          className={`review-summary-btn ${highRisk.status === 'pending' ? 'review-summary-hot' : ''}`}
          type="button"
          onClick={() => onPopupChange(activePopup === 'risk' ? null : 'risk')}
        >
          <span>{t('review.highRiskTitle')}</span>
          <strong>{reviewStatusLabel(highRisk.status)}</strong>
        </button>
        <button
          className="review-summary-btn"
          type="button"
          onClick={() => onPopupChange(activePopup === 'skills' ? null : 'skills')}
        >
          <span>{t('dag.switchableOptions')}</span>
          <strong>{ambiguityHidden ? t('dag.dismissLabel', { time: lowRiskAmbiguity.dismissedUntil }) : 'low risk'}</strong>
        </button>
        {pendingTotalCount > 0 && (
          <button
            className="review-summary-btn"
            type="button"
            onClick={() => onPopupChange(activePopup === 'digest' ? null : 'digest')}
          >
            <span>Pending</span>
            <strong>{t('dag.pendingItems', { count: pendingTotalCount })}</strong>
          </button>
        )}
      </div>

      {/* 任務進度內容：一般使用者看步驟，開發用 ID 藏在後端 debug API。 */}
      {activePopup === 'dag' && popupFrame && createPortal(
        <article
          className="review-card review-card-popup review-card-dag"
          style={{left: `${popupFrame.left}px`, top: `${popupFrame.top}px`, width: `${popupFrame.width}px`}}
        >
          <header>
            <span>{dagRun?.title || t('dag.noDagRun')}</span>
            <strong>{dagRun ? dagStatusLabel(dagRun.status) : 'idle'}</strong>
          </header>
          {dagRun ? (
            <>
              <div className="dag-node-list">
                {dagRun.nodes.map((node) => (
                  <div className={`dag-node-item dag-node-${node.status}`} key={node.id}>
                    <i className={`dag-node-dot dag-risk-${node.risk}`} />
                    <div>
                      <strong>{node.title}</strong>
                      <span>{node.action}</span>
                      <small>
                        {dagStatusLabel(node.status)}
                        {' · '}
                        {formatDagTime(node.startedAt)}
                        {' → '}
                        {formatDagTime(node.endedAt)}
                        {node.durationMs != null ? ` · ${(node.durationMs / 1000).toFixed(1)}s` : ''}
                      </small>
                      {node.resultSummary && <small>{node.resultSummary}</small>}
                    </div>
                    <em>{formatNodeRiskLabel(node)}</em>
                  </div>
                ))}
              </div>
              {taskProgressDebugEnabled && (
                <div className="task-debug-box">
                  <button
                    type="button"
                    onClick={() => callWails(() => GetDAGRunDebug(dagRun.id)).then(setTaskDebugDump).catch((error) => setTaskDebugDump({error: error?.message || String(error)}))}
                  >
                    開發工具，穩定後移除
                  </button>
                  {taskDebugDump && <pre>{JSON.stringify(taskDebugDump, null, 2)}</pre>}
                </div>
              )}
            </>
          ) : (
            <p>{t('dag.dagRunHint')}</p>
          )}
        </article>,
        document.body,
      )}

      {/* High-risk popup */}
      {activePopup === 'risk' && popupFrame && createPortal(
        <article
          className={`review-card review-card-popup review-card-risk review-card-${highRisk.status}`}
          style={{left: `${popupFrame.left}px`, top: `${popupFrame.top}px`, width: `${popupFrame.width}px`}}
        >
          <header>
            <span>{highRisk.title}</span>
            <strong>{reviewStatusLabel(highRisk.status)}</strong>
          </header>
          <div className="review-detail-row">
            <div>
              <span>這一步要做什麼</span>
              <p>{highRisk.action}</p>
            </div>
            <div>
              <span>會影響什麼</span>
              <p>{highRisk.permissionSummary}</p>
            </div>
            <div>
              <span>使用的工具/模型</span>
              <p>{highRisk.skillId || highRisk.summaryHash || '目前模型'}</p>
            </div>
            <div>
              <span>為什麼需要確認</span>
              <p>{highRisk.diff?.[1] || highRisk.diff?.[0] || '這一步風險較高，執行前需要確認。'}</p>
            </div>
          </div>
          <small>{reviewConsequenceText(highRisk.status, highRisk.note)}</small>
          {highRisk.status === 'pending' && highRisk.id && onConfirmSkillBuild && (
            <div className="review-action-row">
              <button type="button" className="review-confirm-btn" onClick={() => onConfirmSkillBuild(highRisk.id)}>{t('dag.confirmExecute')}</button>
            </div>
          )}
        </article>,
        document.body,
      )}

      {/* Low-risk ambiguity popup */}
      {activePopup === 'skills' && !ambiguityHidden && popupFrame && createPortal(
        <article
          className="review-card review-card-popup review-card-skills skill-ambiguity-card"
          style={{left: `${popupFrame.left}px`, top: `${popupFrame.top}px`, width: `${popupFrame.width}px`}}
        >
          <header>
            <span>{t('dag.switchable')}</span>
            <strong>low risk</strong>
          </header>
          <div className="skill-choice-list">
            {lowRiskAmbiguity.candidates.map((candidate) => (
              <button
                className={candidate.id === lowRiskAmbiguity.selectedSkillId ? 'skill-choice-active' : ''}
                type="button"
                key={candidate.id}
                onClick={() => onSkillSelect(candidate.id)}
              >
                <span>{candidate.id}</span>
                <small>{candidate.reason} · score {candidate.score} · {candidate.risk}</small>
              </button>
            ))}
          </div>
          <div className="skill-snooze-row">
            <label>
              {t('review.snoozeLabel')}
              <input
                max="72"
                min="1"
                type="number"
                value={snoozeHours}
                onChange={(event) => onSnoozeHoursChange(Math.max(1, Number(event.target.value) || 1))}
              />
              {t('review.hours')}
            </label>
            <button type="button" onClick={onSnooze}>{t('review.applySnooze')}</button>
          </div>
        </article>,
        document.body,
      )}

      {/* I-1: Pending Digest + Hook Candidates + Package Queue popup */}
      {activePopup === 'digest' && popupFrame && createPortal(
        <article
          className="review-card review-card-popup review-card-digest"
          style={{left: `${popupFrame.left}px`, top: `${popupFrame.top}px`, width: `${popupFrame.width}px`}}
        >
          <header>
            <span>Pending Digest</span>
            <strong>{t('review.pendingCount', { count: pendingTotalCount })}</strong>
          </header>

          {/* I-7: Node summaries surface here as Pending Digest entries. */}
          {dagSummaryCount > 0 && (
            <div className="digest-section">
              <h4>DAG Node Summary ({dagSummaryCount})</h4>
              {dagRun.summaries.map((summary) => (
                <div className="digest-item" key={summary.nodeId}>
                  <span>{summary.title}</span>
                  <small>{summary.text} · {formatDagTime(summary.generatedAt)}</small>
                </div>
              ))}
            </div>
          )}

          {/* Hook Candidates: Tag Patches */}
          {hookCandidates?.tagPatches?.length > 0 && (
            <div className="digest-section">
              <h4>{t('review.tagPatchTitle', { count: hookCandidates.tagPatches.length })}</h4>
              {hookCandidates.tagPatches.map((patch, i) => (
                <div className="digest-item" key={patch.id || i}>
                  <span>{patch.tag || patch.id}</span>
                  <small>{patch.reason || 'hook evidence'}</small>
                  <div className="digest-item-actions">
                    <button type="button" onClick={() => onAcknowledgeDigestItem(patch.id, 'keep')}>{t('review.keep')}</button>
                    <button type="button" onClick={() => onAcknowledgeDigestItem(patch.id, 'archive')}>{t('review.archive')}</button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Hook Candidates: Subagent Candidates */}
          {hookCandidates?.subagentCandidates?.length > 0 && (
            <div className="digest-section">
              <h4>{t('review.subagentTitle', { count: hookCandidates.subagentCandidates.length })}</h4>
              {hookCandidates.subagentCandidates.map((candidate, i) => (
                <div className="digest-item" key={candidate.id || i}>
                  <span>{candidate.name || candidate.id}</span>
                  <small>{candidate.source || 'hook run'}</small>
                  <div className="digest-item-actions">
                    <button type="button" onClick={() => onAcknowledgeDigestItem(candidate.id, 'review_now')}>{t('review.reviewAction')}</button>
                    <button type="button" onClick={() => onAcknowledgeDigestItem(candidate.id, 'archive')}>{t('review.archive')}</button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Hook Candidates: Registry Proposals */}
          {hookCandidates?.registryProposals?.length > 0 && (
            <div className="digest-section">
              <h4>Tool Registry Patch ({hookCandidates.registryProposals.length})</h4>
              {hookCandidates.registryProposals.map((proposal, i) => (
                <div className="digest-item" key={proposal.id || i}>
                  <span>{proposal.toolId || proposal.id}</span>
                  <small>{proposal.action || 'patch proposal'}</small>
                  <div className="digest-item-actions">
                    <button type="button" onClick={() => onAcknowledgeDigestItem(proposal.id, 'keep')}>{t('review.apply')}</button>
                    <button type="button" onClick={() => onAcknowledgeDigestItem(proposal.id, 'archive')}>{t('review.ignore')}</button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Pending Digest (weekly summary) — three blocks */}
          {pendingDigest && (
            <>
              {digestBlocks.urgent.length > 0 && (
                <div className="digest-section">
                  <button className="digest-section-toggle" type="button" onClick={() => setDigestExpanded((prev) => ({...prev, urgent: !prev.urgent}))}>
                    <h4>{t('review.urgentTitle', { count: digestBlocks.urgent.length })}</h4>
                    <span>{digestExpanded.urgent ? '▾' : '▸'}</span>
                  </button>
                  {digestExpanded.urgent && digestBlocks.urgent.map((item, i) => (
                    <div className="digest-item" key={item.id || i}>
                      <span>{item.title || item.id}</span>
                      <small>{item.category} · {item.risk || 'unknown'}</small>
                      <div className="digest-item-actions">
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'keep')}>{t('review.keep')}</button>
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'review_now')}>{t('review.reviewAction')}</button>
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'delete')}>{t('review.delete')}</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {digestBlocks.later.length > 0 && (
                <div className="digest-section">
                  <button className="digest-section-toggle" type="button" onClick={() => setDigestExpanded((prev) => ({...prev, later: !prev.later}))}>
                    <h4>{t('review.laterTitle', { count: digestBlocks.later.length })}</h4>
                    <span>{digestExpanded.later ? '▾' : '▸'}</span>
                  </button>
                  {digestExpanded.later && digestBlocks.later.map((item, i) => (
                    <div className="digest-item" key={item.id || i}>
                      <span>{item.title || item.id}</span>
                      <small>{item.category}</small>
                      <div className="digest-item-actions">
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'keep')}>{t('review.keep')}</button>
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'archive')}>{t('review.archive')}</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {digestBlocks.archive.length > 0 && (
                <div className="digest-section">
                  <button className="digest-section-toggle" type="button" onClick={() => setDigestExpanded((prev) => ({...prev, archive: !prev.archive}))}>
                    <h4>{t('review.archiveTitle', { count: digestBlocks.archive.length })}</h4>
                    <span>{digestExpanded.archive ? '▾' : '▸'}</span>
                  </button>
                  {digestExpanded.archive && digestBlocks.archive.map((item, i) => (
                    <div className="digest-item" key={item.id || i}>
                      <span>{item.title || item.id}</span>
                      <small>{item.category}</small>
                      <div className="digest-item-actions">
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'batch_archive_low_value')}>{t('review.batchArchive')}</button>
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'keep')}>{t('review.keep')}</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}

          {/* Pending Packages */}
          {pendingPackages?.length > 0 && (
            <div className="digest-section">
              <h4>{t('review.pendingPackagesTitle', { count: pendingPackages.length })}</h4>
              {pendingPackages.map((pkg, i) => (
                <div className="digest-item" key={pkg.id || i}>
                  <span>{pkg.manifest?.name || pkg.sourcePath || pkg.id}</span>
                  <small>risk: {pkg.manifest?.riskTag || 'unknown'}</small>
                </div>
              ))}
            </div>
          )}

          {/* #I-1002: Rejected Review Card 歷史 — 顯示安裝失敗 / 被拒絕的套件軌跡 */}
          {reviewArchive?.filter((c) => c.status === 'rejected').length > 0 && (
            <div className="digest-section">
              <h4>{t('review.rejectedTitle', { count: reviewArchive.filter((c) => c.status === 'rejected').length })}</h4>
              {reviewArchive.filter((c) => c.status === 'rejected').slice(-10).map((card, i) => (
                <div className="digest-item digest-item-rejected" key={card.id || i}>
                  <span>{card.plain_reason}</span>
                  <small className="rejected-reason">{t('review.rejectedReason', { reason: card.reject_reason || card.engineer_reason })}</small>
                  <small>{card.archived_at}</small>
                </div>
              ))}
            </div>
          )}
        </article>,
        document.body,
      )}

      {/* ── v3.6.2 W3A Media Provenance（§9A）UI ── */}

      {/* W3A 匯入選單：偵測到 sidecar 後顯示功能說明 */}
      {w3aImportPopup && (
        <div className="w3a-import-overlay" onClick={onDismissW3AImportPopup}>
          <div className="w3a-import-popup" onClick={(e) => e.stopPropagation()}>
            <div className="w3a-import-header">
              <span className="w3a-import-icon">
                {w3aStatusConfig[w3aImportPopup.info?.status]?.icon || '❓'}
              </span>
              <span className="w3a-import-title">{t("w3a.title")}</span>
            </div>
            <div className="w3a-import-status" style={{color: w3aStatusConfig[activeW3AInfo.status]?.color || '#95a5a6'}}>
              {w3aStatusConfig[activeW3AInfo.status]?.label || t('w3a.unknownStatus')}
            </div>
            <div className="w3a-import-recommendation">{w3aImportPopup.recommendation}</div>
            {w3aImportPopup.has_sidecar && (
              <div className="w3a-import-sidecar-badge">{t("w3a.sidecarDetected")}</div>
            )}
            <div className="w3a-detail-grid">
              {activeW3APath && (
                <div className="w3a-detail-row">
                  <span>{t('w3a.sourcePath')}</span>
                  <code>{activeW3APath}</code>
                </div>
              )}
              {activeW3ASidecar && (
                <div className="w3a-detail-row">
                  <span>{t('w3a.sidecarPath')}</span>
                  <code>{activeW3ASidecar}</code>
                </div>
              )}
              {activeW3AInfo.media_scope && (
                <div className="w3a-detail-row">
                  <span>{t('w3a.mediaScope')}</span>
                  <strong>{activeW3AInfo.media_scope}</strong>
                </div>
              )}
              {activeW3AInfo.training && (
                <>
                  <div className="w3a-detail-row">
                    <span>{t('w3a.trainingSafe')}</span>
                    <strong>{activeW3ATraining.training_safe ? t('w3a.safeYes') : t('w3a.safeNo')}</strong>
                  </div>
                  <div className="w3a-detail-row">
                    <span>{t('w3a.filterRequired')}</span>
                    <strong>{activeW3ATraining.filter_required ? t('common.yes') : t('common.no')}</strong>
                  </div>
                </>
              )}
              <div className="w3a-detail-row">
                <span>{t('w3a.trustCount')}</span>
                <strong>{w3aTrustList.length}</strong>
              </div>
            </div>
            {activeW3APollution && (
              <div className={`w3a-pollution-card ${activeW3APollution.is_pollution_risk ? 'w3a-pollution-risk' : ''}`}>
                <span>{t('w3a.weightedTotal')}</span>
                <strong>{typeof activeW3APollution.weighted_total === 'number' ? activeW3APollution.weighted_total.toFixed(2) : activeW3APollution.weighted_total}</strong>
                {activeW3APollution.details && <small>{activeW3APollution.details}</small>}
              </div>
            )}
            {w3aTransferGuidance && (
              <div className="w3a-guidance-card">
                <strong>{w3aTransferGuidance.ui_message}</strong>
                <div className="w3a-guidance-list">
                  <span>{t('w3a.recommended')}</span>
                  {(w3aTransferGuidance.recommended || []).map((item) => <small key={item}>{item}</small>)}
                </div>
                <div className="w3a-guidance-list">
                  <span>{t('w3a.notRecommended')}</span>
                  {(w3aTransferGuidance.not_recommended || []).map((item) => <small key={item}>{item}</small>)}
                </div>
              </div>
            )}
            <div className="w3a-import-capabilities">
              <span className="w3a-import-cap-title">{t("w3a.capTitle")}</span>
              {(w3aImportPopup.capabilities || []).map((cap, i) => (
                <div className="w3a-import-cap-item" key={i}>▸ {cap}</div>
              ))}
            </div>
            {w3aActionError && <div className="w3a-import-error">{w3aActionError}</div>}
            <div className="w3a-import-actions">
              <button className="w3a-import-btn w3a-import-btn-secondary" type="button" disabled={!!w3aActionBusy} onClick={onLoadW3AInfo}>{w3aActionBusy === 'info' ? t('w3a.actionBusy') : t('w3a.infoAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-secondary" type="button" disabled={!!w3aActionBusy} onClick={onDetectW3APollution}>{w3aActionBusy === 'pollution' ? t('w3a.actionBusy') : t('w3a.pollutionAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-secondary" type="button" disabled={!!w3aActionBusy} onClick={onShowW3AGuidance}>{w3aActionBusy === 'guidance' ? t('w3a.actionBusy') : t('w3a.guidanceAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-secondary" type="button" disabled={!!w3aActionBusy} onClick={onTrustW3ADeveloper}>{w3aActionBusy === 'trust' ? t('w3a.actionBusy') : t('w3a.trustAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-warning" type="button" disabled={!!w3aActionBusy} onClick={onExportW3ACopy}>{w3aActionBusy === 'export' ? t('w3a.actionBusy') : t('w3a.exportCopyAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-primary" type="button" onClick={onDismissW3AImportPopup}>{t('w3a.confirm')}</button>
            </div>
          </div>
        </div>
      )}

      {/* §24: 文件寫入確認卡片 */}
      <DocumentReviewCard onToast={onShowW3AToast} />

      {/* W3A 傳輸引導 toast（軟性提示） */}
      {w3aToastMsg && (
        <div className="w3a-toast" onClick={onDismissW3AToast}>
          <span className="w3a-toast-icon">🔏</span>
          <span className="w3a-toast-text">{w3aToastMsg}</span>
        </div>
      )}
    </section>
  );
}

// I-1: Categorize digest items into three UI blocks per spec §12
function categorizePendingDigest(digest) {
  const blocks = {urgent: [], later: [], archive: []};
  if (!digest || !digest.items) return blocks;
  for (const item of digest.items) {
    const cat = item.category || '';
    if (cat === 'risky' || cat === 'high_value') {
      blocks.urgent.push(item);
    } else if (cat === 'keep_suggestion') {
      blocks.later.push(item);
    } else if (cat === 'archive_suggestion' || cat === 'duplicate_group') {
      blocks.archive.push(item);
    } else {
      blocks.later.push(item);
    }
  }
  return blocks;
}

// I-2: Skill Activity Card — non-blocking, shows latest 2 injection records
function SkillActivityCard({injections}) {
  const t = useI18n(s => s.t);
  const [expanded, setExpanded] = useState(null);
  if (!injections || injections.length === 0) return null;

  return (
    <div className="skill-activity-card" aria-label="Skill Activity">
      <div className="skill-activity-header">
        <span>Skill Activity</span>
        <small>{t('skill.injectionCount', { count: injections.length })}</small>
      </div>
      {injections.map((inj, i) => (
        <div className="skill-activity-item" key={inj.skillID || inj.skill_id || i}>
          <button
            className="skill-activity-summary"
            type="button"
            onClick={() => setExpanded(expanded === i ? null : i)}
          >
            <span>{inj.skillID || inj.skill_id || 'unknown'}</span>
            <small>{inj.reason || inj.matchReason || inj.match_reason || ''}</small>
            <code>{inj.summaryHash || inj.summary_hash || ''}</code>
          </button>
          {expanded === i && (
            <div className="skill-activity-detail">
              <div className="review-meta-grid">
                <span>score</span><strong>{inj.score ?? '—'}</strong>
                <span>risk</span><strong>{inj.risk || '—'}</strong>
                <span>session</span><strong>{inj.sessionID || inj.session_id || '—'}</strong>
                <span>cleared</span><strong>{inj.clearedAt || inj.cleared_at ? t('skill.clearedYes') : t('skill.clearedNo')}</strong>
                {inj.clearReason || inj.clear_reason ? (<><span>clear reason</span><strong>{inj.clearReason || inj.clear_reason}</strong></>) : null}
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// #I-207: 初次使用說明卡 — 第一次 skill 注入成功後一次性顯示
// ---------------------------------------------------------------------------
/* i18n: skill first use card */
function SkillFirstUseCard({onDismiss}) {
  const t = useI18n(s => s.t);
  return (
    <div className="skill-first-use-card" role="alert" aria-label={t('skill.firstUseLabel')}>
      <div className="skill-first-use-header">
        <span>{t('skill.firstUseTitle')}</span>
        <button type="button" className="skill-first-use-close" onClick={onDismiss} aria-label={t('common.close')}>✕</button>
      </div>
      <p>{t('skill.desc1')}</p>
      <p>{t('skill.desc2')}</p>
      <p>{t('skill.desc3')}</p>
      <small>{t('skill.desc4')}</small>
    </div>
  );
}

// ---------------------------------------------------------------------------
// #I-207: Settings 頁 Skill Context 靜態說明區塊
// ---------------------------------------------------------------------------
function SkillContextSettingsSection() {
  const t = useI18n(s => s.t);
  return (
    <div className="skill-settings-section">
      <h4>Skill Context Orchestration</h4>
      <p>{t('skill.settingsDesc')}</p>
      <ul>
        <li>{t("skill.autoInjectHint")}</li>
        <li>{t("skill.multiCandidateHint")}</li>
        <li>{t("skill.highRiskHint")}</li>
        <li>{t("skill.auditHint")}</li>
      </ul>
    </div>
  );
}

// ---------------------------------------------------------------------------
// I-3: Recording stop dialog — user-facing copy for the Draft Sandbox branches.
// ---------------------------------------------------------------------------
/* i18n: recording stop dialog */
function DraftSandboxStopDialog({onPromote, onDismiss}) {
  const t = useI18n(s => s.t);
  return createPortal(
    <div className="sandbox-stop-overlay" role="dialog" aria-label={t('recording.optionsLabel')}>
      <article className="sandbox-stop-card">
        <header>
          <span>{t('recording.stopped')}</span>
        </header>
        <p>{t('recording.processHint')}</p>
        <div className="sandbox-stop-actions">
          <button type="button" onClick={() => onPromote('formal_review')}>
            {t('recording.startReview')}
          </button>
          <button type="button" onClick={() => onPromote('pending_candidate')}>
            {t('recording.saveCandidate')}
          </button>
          <button type="button" className="sandbox-stop-discard" onClick={onDismiss}>
            {t('recording.discard')}
          </button>
        </div>
      </article>
    </div>,
    document.body,
  );
}

// ---------------------------------------------------------------------------
// I-3: Trusted Session Expired Dialog
// ---------------------------------------------------------------------------
function TrustedSessionExpiredDialog({onChoice}) {
  const t = useI18n(s => s.t);
  return createPortal(
    <div className="sandbox-stop-overlay" role="dialog" aria-label={t("trust.expiredLabel")}>
      <article className="sandbox-stop-card">
        <header>
          <span>{t("trust.expiredTitle")}</span>
        </header>
        <p>{t("trust.expiredHint")}</p>
        <div className="sandbox-stop-actions">
          <button type="button" onClick={() => onChoice('reenable')}>
            {t("trust.reauthOS")}
          </button>
          <button type="button" onClick={() => onChoice('allow_once')}>
            {t("trust.allowOnce")}
          </button>
          <button type="button" className="sandbox-stop-discard" onClick={() => onChoice('cancel')}>
            {t('common.cancel')}
          </button>
        </div>
      </article>
    </div>,
    document.body,
  );
}

// ---------------------------------------------------------------------------
// I-4: Safari Runtime Notice Modal
// ---------------------------------------------------------------------------
function SafariNoticeModal({notice, onDismiss}) {
  const t = useI18n(s => s.t);
  return createPortal(
    <div className="sandbox-stop-overlay" role="dialog" aria-label={t("safari.noticeLabel")}>
      <article className="safari-notice-modal">
        <header>
          <span>{t("safari.noticeTitle")}</span>
        </header>
        <p className="safari-notice-body">{notice.message || notice}</p>
        {notice.details && <p className="safari-notice-details">{notice.details}</p>}
        <div className="safari-notice-actions">
          <button type="button" onClick={onDismiss}>{t("safari.acknowledge")}</button>
        </div>
      </article>
    </div>,
    document.body,
  );
}

// ---------------------------------------------------------------------------
// I-4: Style Diff Preview Modal
// ---------------------------------------------------------------------------
function StyleDiffPreviewModal({diffJson, onConfirm, onCancel}) {
  const t = useI18n(s => s.t);
  const entries = typeof diffJson === 'string' ? JSON.parse(diffJson) : diffJson;
  const diffLines = Object.entries(entries || {});

  return createPortal(
    <div className="sandbox-stop-overlay" role="dialog" aria-label={t("styleDiff.previewLabel")}>
      <article className="style-diff-preview">
        <header>
          <span>{t("styleDiff.previewTitle")}</span>
        </header>
        <div className="style-diff-list">
          {diffLines.map(([key, value]) => (
            <div className="style-diff-row" key={key}>
              <span className="style-diff-key">{key}</span>
              <span className="style-diff-arrow">→</span>
              <span className="style-diff-value">{typeof value === 'object' ? JSON.stringify(value) : String(value)}</span>
            </div>
          ))}
          {diffLines.length === 0 && <p className="style-diff-empty">{t("styleDiff.noDiff")}</p>}
        </div>
        <div className="style-diff-actions">
          <button type="button" onClick={onConfirm}>{t('common.apply')}</button>
          <button type="button" className="sandbox-stop-discard" onClick={onCancel}>{t('common.cancel')}</button>
        </div>
      </article>
    </div>,
    document.body,
  );
}

// v3.3.2 P0.3 — RightRail renders tool visibility state.
// Available=false → lightning-fracture overlay; never greyed-out / hidden.
/* i18n: right rail */
function RightRail({
  isLearningEnabled,
  isRecordingEnabled,
  isToolPopupOpen,
  learningDigestReady,
  sourceTrustHint,
  referenceFiles,
  onLearningToggle,
  onRecordingToggle,
  onReferenceFileDrop,
  onReferenceFileDragOut,
  onReferenceFileReorder,
  onReferenceInternalDrag,
  onReferenceLinkOpen,
  onReferenceCardDoubleClick,
  onReferenceFailedRemove,
  onToolFavorite,
  onToolPopupToggle,
}) {
  const t = useI18n(s => s.t);
  const [draggedReferenceKey, setDraggedReferenceKey] = useState('');
  const draggedReferenceKeyRef = useRef('');

  function handleReferenceDragStart(event, file) {
    const fileKey = referenceFileKey(file);
    if (!fileKey) {
      event.preventDefault();
      return;
    }
    // §M3+ 失敗 entry：三層保險阻止 OS 拿到任何 drag payload
    const isFailed = file?.status === 'error' || file?.source !== 'library';
    if (isFailed) {
      try {
        // 1. 清空 dataTransfer 內容（避免 text/plain 變 .textClipping）
        event.dataTransfer?.clearData?.();
        if (event.dataTransfer) {
          event.dataTransfer.effectAllowed = 'none';
          event.dataTransfer.setData('text/plain', '');
        }
      } catch (_) { /* old browsers */ }
      // 2. 阻止預設 drag 行為
      event.preventDefault();
      event.stopPropagation();
      // 3. 立即從 state 移除（即使 drag 真的被啟動，state 也已乾淨）
      onReferenceFailedRemove?.(fileKey);
      return false;
    }
    draggedReferenceKeyRef.current = fileKey;
    onReferenceInternalDrag?.(true);
    setDraggedReferenceKey(fileKey);
    event.dataTransfer.effectAllowed = 'copyMove';
    // §M3+ 不用 text/plain（macOS 會把 path-like text 升級成 file drag、誤觸 import）。
    // 改用 custom MIME；OS 不認得，drop 在外面也不會建桌面假檔；reorder 只看 draggedReferenceKeyRef。
    try { event.dataTransfer.setData('application/x-ai-console-ref-key', fileKey); } catch (_) {}
    void onReferenceFileDragOut?.(file);
  }

  function handleReferenceDragOver(event, file) {
    const draggedKey = draggedReferenceKeyRef.current;
    const targetKey = referenceFileKey(file);
    if (!draggedKey || !targetKey || draggedKey === targetKey) return;
    event.preventDefault();
    event.stopPropagation();
    event.dataTransfer.dropEffect = 'move';
    const rect = event.currentTarget.getBoundingClientRect();
    const placement = event.clientY > rect.top + (rect.height / 2) ? 'after' : 'before';
    onReferenceFileReorder?.(draggedKey, targetKey, placement);
  }

  function finishReferenceDrag(event) {
    event?.stopPropagation?.();
    draggedReferenceKeyRef.current = '';
    onReferenceInternalDrag?.(false);
    setDraggedReferenceKey('');
  }

  function handleReferenceDrop(event) {
    // 只把內部 reference 排序拖曳吃掉；外部檔案仍要進圖片/引用分流。
    if (draggedReferenceKeyRef.current) {
      event?.preventDefault?.();
      finishReferenceDrag(event);
      return;
    }
    if (event?.dataTransfer?.files?.length) {
      event.preventDefault();
      event.stopPropagation();
      onReferenceFileDrop?.(Array.from(event.dataTransfer.files));
    }
  }

  return (
    <aside
      className="right-panel"
      onDragOver={(event) => {
        if (
          event.dataTransfer.files?.length
          || Array.from(event.dataTransfer.types).includes('Files')
          || Array.from(event.dataTransfer.types).includes('application/x-ai-console-tool-id')
        ) {
          event.preventDefault();
          event.dataTransfer.dropEffect = event.dataTransfer.files?.length || Array.from(event.dataTransfer.types).includes('Files') ? 'copy' : 'move';
        }
      }}
      onDrop={(event) => {
        // §M3+ 內部 reorder drag 不應觸發 import；以 draggedReferenceKeyRef 為憑。
        if (draggedReferenceKeyRef.current) {
          event.preventDefault();
          finishReferenceDrag(event);
          return;
        }
        if (event.dataTransfer.files?.length) {
          event.preventDefault();
          onReferenceFileDrop(Array.from(event.dataTransfer.files));
          return;
        }
        const toolId = event.dataTransfer.getData('application/x-ai-console-tool-id');
        if (!toolId) return;
        event.preventDefault();
        onToolFavorite(toolId);
      }}
    >
      <button className="tool-card tool-amber reference-link-card" type="button" onClick={onReferenceLinkOpen}>
        <span>▤</span>
        <span>{t('rightRail.citeLink')}</span>
      </button>
      {sourceTrustHint && (
        <div
          className="source-trust-chip source-trust-rail-chip"
          data-level={sourceTrustHint.level}
          title={sourceTrustHint.text}
        >
          <span className="source-trust-icon">{sourceTrustHint.level === 'trusted' ? '✓' : sourceTrustHint.level === 'blocked' ? '✕' : '⚠'}</span>
          <span className="source-trust-label">{sourceTrustHint.text}</span>
        </div>
      )}
      <div className="rail-mode-row">
        <button
          className={`rail-mode-btn rail-mode-learning ${isLearningEnabled ? 'rail-mode-active' : ''} ${learningDigestReady ? 'rail-mode-notify' : ''}`}
          type="button"
          onClick={onLearningToggle}
          title={t('rightRail.learningTooltip')}
        >
          <span>{t('rightRail.learning')}</span>
          <small>{learningDigestReady ? t('rightRail.hasUpdate') : isLearningEnabled ? t('rightRail.editing') : t('rightRail.close')}</small>
        </button>
        <button
          className={`rail-mode-btn rail-mode-record ${isRecordingEnabled ? 'rail-mode-active rail-mode-recording' : ''}`}
          type="button"
          onClick={onRecordingToggle}
          title={t('rightRail.recordTooltip')}
        >
          <span>{t('rightRail.record')}</span>
          <small>{isRecordingEnabled ? t('rightRail.recording') : t('rightRail.close')}</small>
        </button>
      </div>
	      <div
	        className="tool-card reference-file-card"
	        onDragOver={(event) => {
	          event.preventDefault();
	          event.dataTransfer.dropEffect = 'copy';
	        }}
	        onDrop={(event) => {
	          handleReferenceDrop(event);
	        }}
	      >
        <span>▤</span>
        <span>{t('rightRail.citeFile')}</span>
      </div>
      <div
        className="reference-file-list"
	        onDragOver={(event) => {
	          if (!draggedReferenceKeyRef.current) return;
	          event.preventDefault();
	          event.stopPropagation();
	          event.dataTransfer.dropEffect = 'move';
	        }}
	        onDrop={handleReferenceDrop}
      >
        {referenceFiles.map((file, index) => {
          const fileKey = referenceFileKey(file) || `${file.path}-${index}`;
          const isDragging = draggedReferenceKey === fileKey;
          return (
          <div
            className={`reference-file-name${isDragging ? ' reference-file-dragging' : ''}`}
            data-status={file.status || 'ready'}
            data-draggable="true"
            draggable
            key={fileKey}
	            title={file.detail || file.path}
	            onDragStart={(event) => handleReferenceDragStart(event, file)}
	            onDragOver={(event) => handleReferenceDragOver(event, file)}
	            onDrop={handleReferenceDrop}
            onDragEnd={(event) => {
              // §M3+ 失敗 entry 拖到 window 外 → 移除（同 ToolPopup 的 leftWindow pattern）
              const leftWindow =
                event.clientX <= 0 ||
                event.clientY <= 0 ||
                event.clientX >= window.innerWidth ||
                event.clientY >= window.innerHeight;
              const droppedOutside = leftWindow || (event.clientX === 0 && event.clientY === 0);
              const isFailed = file?.status === 'error' || file?.source !== 'library';
              finishReferenceDrag(event);
              if (droppedOutside && isFailed) {
                onReferenceFailedRemove?.(referenceFileKey(file));
              }
            }}
            onDoubleClick={(event) => {
              event.preventDefault();
              const rect = event.currentTarget.getBoundingClientRect();
              onReferenceCardDoubleClick?.(file, rect);
            }}
          >
            <div className="reference-file-main">
              <span className="reference-file-title">
                {twoLineFileName(file.name, t('rightRail.unnamedFile')).map((line, lineIndex) => <span key={lineIndex}>{line}</span>)}
              </span>
              <small className="reference-file-status">{referenceFileStatusLabel(file.status)}</small>
            </div>
            {shouldShowReferenceFileDetail(file) && <small className="reference-file-detail">{file.detail}</small>}
            {fileExtLabel(file.name) && <span className="reference-file-ext-badge">{fileExtLabel(file.name)}</span>}
          </div>
          );
        })}
      </div>
      <button className="tool-card tool-use-bottom" type="button" onClick={onToolPopupToggle}>
        <span>{isToolPopupOpen ? '×' : '⌕'}</span>
        <span>{isToolPopupOpen ? t('rightRail.close') : t('rightRail.useTools')}</span>
      </button>
    </aside>
  );
}

// §3.1.11 影片副檔名判斷：拖入時用來分流到 data/videos
function isVideoPath(p) {
  return /\.(mp4|mov|m4v|webm|mkv|avi|wmv|flv|mpe?g|3gp|ogv)$/i.test(String(p || ''));
}

// §3.1.10 檔案方框右下角的副檔名角標（txt / md / jpeg / wmv …）
function fileExtLabel(name) {
  const m = /\.([A-Za-z0-9]+)$/.exec(String(name || ''));
  return m ? m[1].toLowerCase() : '';
}

function twoLineFileName(name, fallback = 'Unnamed File') {
  const chars = Array.from(String(name || fallback));
  const first = chars.slice(0, 20).join('');
  const second = chars.slice(20, 40).join('');
  return second ? [first, second] : [first];
}

// v3.3.2 P0.1 — Package install confirmation modal.
// Shown when a dropped package has been quarantined and awaits user decision.
// ══════════════════════════════════════════════
// #42 Review Card 分層顯示 — Package / Skill / MCP 安裝確認
// ══════════════════════════════════════════════
// 上半區（始終可見）：名稱 / risk_tag / 權限宣告 / 預計寫入位置
// 下半區（「展開工程細節」收摺）：manifest 摘要 / 來源路徑 / package hash /
//   新增項目（tools/skills/MCP/adapter）/ routing 影響
// 底層資料模型為完整標準 Review Card，分層僅為視覺呈現策略。
function PackageConfirmModal({pending, packageData, error, onConfirm, onReject}) {
  const t = useI18n(s => s.t);
  const [detailsExpanded, setDetailsExpanded] = useState(false);
  const manifest = pending?.manifest || {};
  const name = manifest.name || packageData?.name || t('package.unknownPackage');
  const riskTag = manifest.risk_tag || 'unknown';
  const riskLabelMap = {low: t('package.riskLow'), medium: t('package.riskMedium'), high: t('package.riskHigh'), unknown: t('package.riskUnknown')};
  const riskColorClass = {low: 'pkg-risk-low', medium: 'pkg-risk-medium', high: 'pkg-risk-high', unknown: 'pkg-risk-unknown'}[riskTag] || 'pkg-risk-unknown';

  // 權限宣告：優先用新欄位 declared_permissions，向下相容 required_perms
  const perms = manifest.declared_permissions?.length ? manifest.declared_permissions
    : manifest.required_perms?.length ? manifest.required_perms : [];

  // 預計寫入位置
  const writeTargets = manifest.write_targets || [];

  return (
    <div className="pkg-modal-overlay" role="dialog" aria-modal="true" aria-label={t("package.installConfirmLabel")}>
      <div className="pkg-modal">
        <div className="pkg-modal-header">
          <span aria-hidden="true">📦</span>
          <span>{t("package.installConfirmTitle")}</span>
        </div>

        {/* ── 上半區：始終可見的四項關鍵資訊 ── */}
        <div className="pkg-modal-body">
          <div className="pkg-modal-name">{name}</div>
          <div className={`pkg-modal-risk ${riskColorClass}`}>
            {t('package.riskLevel', { level: riskLabelMap[riskTag] || riskTag })}
          </div>
          {perms.length > 0 && (
            <div className="pkg-modal-perms">
              <span>{t("package.permissions")}</span>
              {perms.join('、')}
            </div>
          )}
          {writeTargets.length > 0 && (
            <div className="pkg-modal-write-targets">
              <span>{t("package.writeLocation")}</span>
              {writeTargets.join('、')}
            </div>
          )}

          <p className="pkg-modal-notice">
            {t("package.quarantineHint")}
          </p>

          {/* ── 下半區：展開工程細節（收摺） ── */}
          <button
            className="pkg-modal-details-toggle"
            type="button"
            onClick={() => setDetailsExpanded(!detailsExpanded)}
            aria-expanded={detailsExpanded}
          >
            {detailsExpanded ? t('package.collapseDetails') : t('package.expandDetails')}
          </button>

          {detailsExpanded && (
            <div className="pkg-modal-details">
              {manifest.source_metadata && (
                <div className="pkg-detail-row">
                  <span className="pkg-detail-label">{t("package.source")}</span>
                  <span className="pkg-detail-value">{manifest.source_metadata || manifest.source_path}</span>
                </div>
              )}
              {manifest.package_hash && (
                <div className="pkg-detail-row">
                  <span className="pkg-detail-label">Package Hash：</span>
                  <span className="pkg-detail-value pkg-hash">{manifest.package_hash}</span>
                </div>
              )}
              {manifest.manifest_hash && (
                <div className="pkg-detail-row">
                  <span className="pkg-detail-label">Manifest Hash：</span>
                  <span className="pkg-detail-value pkg-hash">{manifest.manifest_hash}</span>
                </div>
              )}
              <div className="pkg-detail-row">
                <span className="pkg-detail-label">{t("package.newItems")}</span>
                <span className="pkg-detail-value">
                  {[
                    manifest.adds_tools && t('package.addTools'),
                    manifest.adds_skills && 'Skill',
                    manifest.adds_mcp_server && 'MCP Server',
                    manifest.adds_adapter && 'Adapter',
                  ].filter(Boolean).join(', ') || t('package.none')}
                </span>
              </div>
              <div className="pkg-detail-row">
                <span className="pkg-detail-label">{t("package.affectsRouting")}</span>
                <span className="pkg-detail-value">{manifest.affects_routing ? t('common.yes') : t('common.no')}</span>
              </div>
              {manifest.declared_entry_points?.length > 0 && (
                <div className="pkg-detail-row">
                  <span className="pkg-detail-label">Entry Points：</span>
                  <span className="pkg-detail-value">{manifest.declared_entry_points.join('、')}</span>
                </div>
              )}
            </div>
          )}

          {pending?.validation_issues?.length > 0 && (
            <div className="pkg-modal-review">
              {pending.validation_issues.map((issue) => <span key={issue}>{issue}</span>)}
            </div>
          )}
          {error && <div className="pkg-modal-error">{error}</div>}
        </div>
        <div className="pkg-modal-actions">
          <button className="pkg-modal-cancel" type="button" onClick={onReject}>{t('common.cancel')}</button>
          <button className="pkg-modal-confirm" type="button" onClick={onConfirm}>{t('package.confirmInstall')}</button>
        </div>
      </div>
    </div>
  );
}

// ──────────────────────────────────────────────
// §30: 關閉視窗 → 存成 sub 對話框
// 使用者關閉視窗時，若 main 直接操作 ≥40%，顯示此對話框。
// ──────────────────────────────────────────────
function SessionCloseDialog({analysis, onSave, onDiscard, onCancel}) {
  const t = useI18n(s => s.t);
  const [subName, setSubName] = useState(analysis?.suggested_name || '');
  if (analysis?.mode === 'active_task') {
    return (
      <div className="session-close-overlay" role="dialog" aria-modal="true" aria-label={t("session.closeConfirmLabel")}>
        <div className="session-close-dialog">
          <h3>{t('session.closeTaskTitle')}</h3>
          <p className="session-close-hint">
            {t('session.closeTaskHint')}
          </p>
          <div className="session-close-actions">
            <button type="button" className="session-close-btn session-close-save" onClick={onDiscard}>
              {t('session.closeTaskConfirm')}
            </button>
            <button type="button" className="session-close-btn session-close-discard" onClick={onCancel}>
              {t('session.goBack')}
            </button>
          </div>
        </div>
      </div>
    );
  }
  if (analysis?.mode === 'active_sub') {
    return (
      <div className="session-close-overlay" role="dialog" aria-modal="true" aria-label={t("session.closeConfirmLabel")}>
        <div className="session-close-dialog">
          <h3>{t("session.closeSubTitle")}</h3>
          <p className="session-close-hint">
            {t('session.closeSubHint')}
          </p>
          <div className="session-close-actions">
            <button type="button" className="session-close-btn session-close-save" onClick={onDiscard}>
              {t('session.close')}
            </button>
            <button type="button" className="session-close-btn session-close-discard" onClick={onCancel}>
              {t('session.goBack')}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="session-close-overlay" role="dialog" aria-modal="true" aria-label={t("session.closePreConfirmLabel")}>
      <div className="session-close-dialog">
        <h3>{t("session.createWorkflowTitle")}</h3>
        <p className="session-close-stats">
          {t("session.workflowStats", { total: analysis.total_actions, direct: analysis.main_direct_actions, ratio: Math.round(analysis.direct_ratio * 100) })}
        </p>
        <p className="session-close-hint">
          {t("session.workflowSaveHint")}
        </p>
        <label className="session-close-name-label">
          {t("session.workflowName")}
          <input
            type="text"
            value={subName}
            onChange={(e) => setSubName(e.target.value)}
            placeholder={t("session.workflowNamePlaceholder")}
            autoFocus
          />
        </label>
        <div className="session-close-actions">
          <button type="button" className="session-close-btn session-close-save" onClick={() => onSave(subName)}>
            {t('session.yes')}
          </button>
          <button type="button" className="session-close-btn session-close-discard" onClick={onDiscard}>
            {t('session.no')}
          </button>
        </div>
      </div>
    </div>
  );
}

function PackageErrorToast({message, onClose}) {
  const t = useI18n(s => s.t);
  return (
    <div className="pkg-error-toast" role="alert">
      <strong>{t("persona.installFail")}</strong>
      <span>{message}</span>
      <button type="button" onClick={onClose}>{t('common.close')}</button>
    </div>
  );
}

export default App;
