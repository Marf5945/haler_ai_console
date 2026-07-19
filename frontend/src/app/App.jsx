import React, {useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import '../tailwind.css';
import CodeArtifactModal from '../components/CodeArtifactModal';
import useHighlight from '../components/highlight/useHighlight';
import '../components/highlight/highlight.css';
import useSystemMarks from '../components/highlight/useSystemMarks';
import {messageDomId} from '../components/highlight/messageId';
import {
  AcknowledgePendingItem,
  AcknowledgePendingItemWithConfirmation,
  ActivateTool,
  ApplyUIStyleDiff,
  BuildSkillContext,
  BootstrapScheduledSkill,
  ConfirmAndExecuteSkillExecution,
  ClearSkillContext,
  ClearWebSearchConfig,
  ConfirmPackageInstall,
  ConfirmSkillArchive,
  CreateScheduledJob,
  DeleteScheduledJob,
  DisableContextualRiskOverride,
  EnableContextualRiskOverride,
  EnableTrustedSessionScope,
  EnableWorkflowTrustForHours,
  GetAppSessionID,
  GetBrowserPreference,
  GetConsoleState,
  GetDeviceTrustProfile,
  GetHookSummary,
  // SEC-05 2b: 開外部連結唯一入口（Go 端做 scheme/metadata 檢查，loopback 放行）
  GetNewSubagentCandidates,
  GetPendingDigest,
  PurgeMessageMarks,
  GetPendingTagPatches,
  GetRecentSkillInjections,
  GetSafariRuntimeNotice,
  GetSchedulerClock,
  GetSettingsState,
  GetToolRegistryPatchProposals,
  GetToolVisibility,
  GetUISettings,
  ListArchivedSkills,
  ListPendingPackages,
  ListScheduledJobs,
  NormalizeSchedulerDraft,
  ResolveSchedulerBackgroundPrompt,
  ConfirmScheduledRun,
  // #I-1002: Review Archive binding
  ListReviewArchive,
  ListTools,
  PollStatusRail,
  PrepareLearningDigest,
  PreparePackageInstall,
  PreparePackageInstallPayload,
  PromoteDraftToPending,
  ListExternalLinksByType,
  ListSharedSourceFiles,
  PreviewExternalLink,
  RegisterExternalLink,
  StartGGUFImport,
  SuggestGGUFFiles,
  RemoveExternalLink,
  RejectPackageInstall,
  ReorderPersonas,
  ResumeScheduledJob,
  ResolveSkillForAction,
  OpenVisualLearningPermissionSettings,
  ExecuteSkillMessage,
  RestoreUIDefaults,
  SavePanelSettings,
  SavePersona,
  FinalizeNativeGoProgramAuthoringExport,
  FinalizeNativePersonaExport,
  FinalizeNativeSkillExport,
  GetGoProgramAuthoringDetail,
  ListGoProgramAuthoringCatalog,
  NativeDragExportPersonaHandler,
  ExportProjectBackupHandler,
  ImportProjectBackupHandler,
  IsProjectBackupEncryptedHandler,
  SelectProjectBackupExportDirectory,
  SelectProjectBackupFile,
  NativeDragExportGoProgramAuthoring,
  NativeDragExportSkill,
  ScanSkillFolder,
  SetBrowserPreference,
  SetToolPreference,
  SetTrustDomAndClick,
  ExecuteNativeLearningReplayStep,
  HideNativeLearningReplayCursor,
  ConfirmLearningTextEvent,
  RecordLearningMouseEvent,
  RecordLearningKeyboardEvent,
  RecordStepTrace,
  RequestVisualLearningPermissions,
  StartDraftSandbox,
  StartHookRun,
  StopDraftSandbox,
  MarkSkillFirstUseExplained,
  SendCLIMessage,
  SendAPIMessage,
  SendTopInteractionMessage,
  SetActiveConversationAgent,
  ClearInspectorHistory,
  ConfirmCommemorativePhoto,
  GetKeepsakeConfig,
  SaveKeepsakeConfig,
  GenerateAvatarViaImageGen,
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
  TestLLMAPIConnection,
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
  RebuildCodeIndex,
  SearchCodeSections,
  BuildCodeContext,
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
  CancelChatMessage,
  ApproveTaskStep,
  GetDAGRunDebug,
  SubmitTaskLoopInput,
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
  PauseScheduledJob,
  // ── v3.6.2 WA3 Media Provenance（§9A）接線 ──
  // 對應 Go app.go 的 WA3 Wails binding 方法，
  // 涵蓋媒體驗證→污染偵測→匯入匯出→傳輸引導→信任清單管理。
  CreateDocumentWA3,
  GetMediaWA3Info,
  DetectModelPollution,
  ExportMediaWithSidecar,
  ImportMediaVerify,
  GetWA3TransferGuidance,
  ListWA3TrustedDevelopers,
  AddWA3TrustedDeveloper,
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
  UpdateScheduledJob,
  GetRemoteBridgeInboundEndpoint,
  SaveRemoteBridgeInboundSecret,
  ListRemoteBridgeInboundAdapters,
  // ── v3.6 雜項接線 ──
  FinalizeNativeSearchSummaryExport,
  FinalizeNativeReferenceFileExport,
  ImportReferenceFile,
  ListReferenceFiles,
  ListCodeArtifacts,
  GetCodeArtifactDetail,
  ExportCodeArtifactToFolder,
  CopyCodeArtifactToClipboard,
  NativeDragExportCodeArtifactBundle,
  ImportVideoFile,
  ListVideoFiles,
  ListReferenceImages,
  NativeDragExportSearchSummary,
  NativeDragExportReferenceFile,
  StopSidecar,
  // ── v3.6.4 Readiness Gate UI Interaction Layer 接線 ──
  // 對應 Go app.go 的 6 個 Wails binding 方法，
  // 涵蓋 Gate 狀態查詢→更新→追蹤記錄→候選選擇→候選清除完整生命週期。
  GetReadinessGateState,
  // UpdateReadinessGateState — 後端 pipeline 內部呼叫，前端 import-only 預留
  // RecordReadinessTrace — 後端自動記錄，前端 import-only 預留
  // ListReadinessTraces — 預留給未來開發者除錯面板
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
  PreviewGoProgramExport,
  ImportGoProgramExport,
  NativeDragExportLearningRun,
  FinalizeNativeLearningRunExport,
  PreviewLearningRunExport,
  ImportLearningRunExport,
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
  SpeakVoiceLine,
  PreviewVoiceProfileText,
  VoiceProfiles,
  SetVoiceOutputEnabled,
  StopVoiceOutput,
  EnterFloatingAvatarNative,
  ExitFloatingAvatarNative,
  EnterFloatingAvatarOverlayImage,
  ExitFloatingAvatarOverlay,
  GetFloatingAvatarOverlayPosition,
  SetFloatingAvatarOverlayChatMode,
  SetFloatingAvatarOverlayMetadata,
} from '../../wailsjs/go/main/App';
import {
  OnFileDrop,
  OnFileDropOff,
  BrowserOpenURL,
  Quit,
  ClipboardGetText,
  ClipboardSetText,
  ScreenGetAll,
  WindowGetPosition,
  WindowGetSize,
  WindowSetAlwaysOnTop,
  WindowSetBackgroundColour,
  WindowSetMinSize,
  WindowSetPosition,
  WindowSetSize,
  WindowHide,
  WindowShow,
  WindowUnminimise,
} from '../../wailsjs/runtime/runtime';
import {EventsOn} from '../lib/wailsRuntime';

import DocumentReviewCard from '../components/DocumentReviewCard';
import ConsequenceMenuOverlay from './components/ConsequenceMenuOverlay';
import ToolCopyConfirmModal from './components/ToolCopyConfirmModal';
import ReferenceLinkModal from './components/ReferenceLinkModal';
import AdapterRenameModal from './components/AdapterRenameModal';
import ReauthInterceptDialog from './components/ReauthInterceptDialog';
import StopRecoveryCard from './components/StopRecoveryCard';
import SafariNoticeModal from './components/SafariNoticeModal';
import DigestAutoArchiveToast from './components/DigestAutoArchiveToast';
import PackageErrorToast from './components/PackageErrorToast';
import OnboardingOverlay from '../features/onboarding/OnboardingOverlay';
// shared helpers extracted from App.jsx
import {openExternal} from '../lib/openExternal';
import {callWails} from '../lib/callWails';
// group-1 components extracted to app/components
import ProjectManagePopup from './components/ProjectManagePopup';
import ToolFlowSectionLabel from './components/ToolFlowSectionLabel';
import RecordingCatalogList from './components/RecordingCatalogList';
import GoProgramAuthoringCatalogList from './components/GoProgramAuthoringCatalogList';
import GoCodePreviewDock from './components/GoCodePreviewDock';
import ToolPopup from './components/ToolPopup';
import AvatarUploadModal from './components/AvatarUploadModal';
import FloatingAvatarMode from './components/FloatingAvatarMode';
import {resolveAvatarMotionManifest} from './avatarMotionRegistry';
import PackageConfirmModal from './components/PackageConfirmModal';
import SessionCloseDialog from './components/SessionCloseDialog';
// small leaf components extracted to app/components
import SkillActivityCard from './components/SkillActivityCard';
import SkillFirstUseCard from './components/SkillFirstUseCard';
import SkillContextSettingsSection from './components/SkillContextSettingsSection';
import DraftSandboxStopDialog from './components/DraftSandboxStopDialog';
import TrustedSessionExpiredDialog from './components/TrustedSessionExpiredDialog';
import {isNativeReplayStep, basenameForDisplay, isExclusiveCandidateSet, defaultWebSearchProviderOptions, toolTabFor, dagStatusLabel} from '../lib/appHelpers';
import {defaultPanelStyle} from '../lib/panelSettings';
// Tier-1 dialogs extracted to app/components
import PackageInstallDecisionDialog from './components/PackageInstallDecisionDialog';
import SubToolConflictDialog from './components/SubToolConflictDialog';
import DragActionModal from './components/DragActionModal';
import LearningReplayStartConfirmCard from './components/LearningReplayStartConfirmCard';
import LearningReplayConfirmCard from './components/LearningReplayConfirmCard';
import LLMAPISetupModal from './components/LLMAPISetupModal';
import WebSearchSetupModal from './components/WebSearchSetupModal';
import StyleDiffPreviewModal from './components/StyleDiffPreviewModal';
import SettingPopupSelect from './components/SettingPopupSelect';
// readiness-gate feature group
import LongPressConfirmButton from '../features/readiness-gate/LongPressConfirmButton';
import useI18n, {t as _t, getAllFemaleKeywords, getAllBeastKeywords, buildGreetingTextKeyMap} from '../locales/useI18n';
import {advanceVoiceSyncSession, createVoiceSyncSession} from './voiceSyncState';
import ConversationPanel, {fallbackReadinessGate} from './ConversationPanel';
import RightRail from './RightRail';
import {
  composerPendingSlowText,
  composerPendingVerySlowText,
  makeComposerPendingMessage,
  replaceComposerPendingMessage,
  stripComposerPendingMarker,
  updateComposerPendingMessage,
} from './conversationMessages';
import {
  buildSchedulerComposerConfirmAction,
  formatSchedulerPayload,
  hasSchedulerClockText,
  hasSchedulerTimeText,
  isSchedulerAffirmation,
  isSchedulerCancellation,
  mergeSchedulerSlotsFromText,
  normalizeSchedulerDraft,
  parseSchedulerActionText,
  parseSchedulerConversationIntent,
  parseSchedulerTimeText,
  schedulerConfirmationMessage,
  schedulerDefaultPayload,
  schedulerJobNo,
  schedulerMissingSlots,
  schedulerQuestionForMissing,
} from './scheduler';
import {fileBaseName, isVideoPath, referenceFileKey} from './referenceFiles';
import {
  DEFAULT_PERSONA_DISPLAY_NAMES,
  LOCKED_PERSONA_ID,
  PATROL_DIALOGUE_ROLE_VARIANTS,
  PERSONA_AVATAR_URLS,
  PERSONA_FULL_BODY_URLS,
  PERSONA_STATE_AVATAR_URLS,
  defaultPixelPackForPersona,
} from './personas/registry';
import {
  fallbackSettings,
  fontPresetVars,
  fontScaleValue,
  normalizeLockedPersonas,
  normalizePanelSettings,
  normalizeSettingsState,
  panelFromUISettings,
  panelLanguageLabelToLocale,
  panelStyleTheme,
  personaLocaleFromPanel,
} from './settingsController';
import {
  buildLearningWindowsAnchor,
  clampReplayCoordinate,
  compactLearningText,
  describeLearningTarget,
  extractVisualReplayDirective,
  isLastLearningOperationReference,
  isLearningReplayRelatedText,
  isLearningReplayRequest,
  learningRecordingBlockedSelector,
  learningReplayBlockedSelector,
  learningReplayStepDelayMs,
  learningSensitiveTextPattern,
  normalizeLearningKey,
  normalizeLearningOperationQuery,
  operationIntentFromCLIResponse,
  parseLearningOperationCatalogRequest,
  resolveLearningOperationMatch,
  visualLearningInteractiveSelector,
} from './learningReplayController';

const SchedulerPanel = React.lazy(() => import('./SchedulerPanel'));
const EmbeddingPickerModal = React.lazy(() => import('../components/EmbeddingPickerModal'));
const SettingsMenu = React.lazy(() => import('./components/SettingsMenu'));
const SettingsWorkspace = React.lazy(() => import('./components/SettingsWorkspace'));
const VisualLearningPanel = React.lazy(() => import('../components/VisualLearningPanel'));

const taskProgressDebugEnabled = typeof window !== 'undefined'
  && (new URLSearchParams(window.location.search).has('taskDebug')
    || window.localStorage?.getItem('task_progress_debug') === '1');

const MAIN_WINDOW_MIN_SIZE = {width: 1180, height: 560};
const FLOATING_AVATAR_SIZE = 96;
const FLOATING_AVATAR_INSET = 16;
const FLOATING_AVATAR_WINDOW_SIZE = FLOATING_AVATAR_SIZE + FLOATING_AVATAR_INSET * 2;
// 後台頭像單擊展開迷你框時，需要把頭像浮窗放大到能容納面板的尺寸。
const FLOATING_AVATAR_PANEL_W = 720;
const FLOATING_AVATAR_PANEL_H = 540;
const FLOATING_AVATAR_LEFT_TIP_ROOM = 260;
const FLOATING_AVATAR_BODY_W = 200;
const FLOATING_AVATAR_BODY_H = 360;
const FLOATING_AVATAR_BODY_WINDOW_W = 720;
const FLOATING_AVATAR_BODY_WINDOW_H = 560;
const FLOATING_AVATAR_BODY_LOCAL_X = Math.round((FLOATING_AVATAR_BODY_WINDOW_W - FLOATING_AVATAR_BODY_W) / 2);
const FLOATING_AVATAR_BODY_LOCAL_Y = 180;
const IS_WINDOWS_RUNTIME = typeof window !== 'undefined' && /\bWindows\b/i.test(window.navigator?.userAgent || '');
// v2.7：Windows 改走 mac 接法——主視窗直接變形成無框透明置頂浮窗，
// React 前端（含 Pixi 動態全身像）繼續在同一個 webview 裡渲染。
// 原生端：macOS 走 floating_avatar_darwin.go（cgo/AppKit），
// Windows 走 floating_avatar_windows.go（Win32 樣式切換）＋ wails_main.go 的
// Windows 建窗選項（WebviewIsTransparent/WindowIsTranslucent）。舊的靜態 PNG
// overlay 路徑（floating_avatar_overlay_windows.go）保留為死碼，改回
// !IS_WINDOWS_RUNTIME 即可退回。
const ENABLE_NATIVE_FLOATING_AVATAR_WINDOW = true;

function firstFiniteNumber(...values) {
  for (const value of values) {
    if (Number.isFinite(value)) return Number(value);
  }
  return null;
}

function clampNumber(value, min, max) {
  const lower = Math.min(min, max);
  const upper = Math.max(min, max);
  return Math.max(lower, Math.min(upper, value));
}

function floatingAvatarCenterPosition(width = FLOATING_AVATAR_WINDOW_SIZE, height = FLOATING_AVATAR_WINDOW_SIZE) {
  const viewportW = firstFiniteNumber(window.innerWidth, width) || width;
  const viewportH = firstFiniteNumber(window.innerHeight, height) || height;
  const maxX = Math.max(12, viewportW - width - 12);
  const maxY = Math.max(12, viewportH - height - 12);
  return {
    x: Math.round(clampNumber((viewportW - width) / 2, 12, maxX)),
    y: Math.round(clampNumber((viewportH - height) / 2, 12, maxY)),
  };
}

function normalizeFloatingAvatarViewportPosition(position, width = FLOATING_AVATAR_WINDOW_SIZE, height = FLOATING_AVATAR_WINDOW_SIZE) {
  if (!Number.isFinite(position?.x) || !Number.isFinite(position?.y)) {
    return floatingAvatarCenterPosition(width, height);
  }
  const viewportW = firstFiniteNumber(window.innerWidth, width) || width;
  const viewportH = firstFiniteNumber(window.innerHeight, height) || height;
  const maxX = Math.max(12, viewportW - width - 12);
  const maxY = Math.max(12, viewportH - height - 12);
  return {
    x: Math.round(clampNumber(Number(position.x), 12, maxX)),
    y: Math.round(clampNumber(Number(position.y), 12, maxY)),
  };
}

function clampFloatingAvatarScreenPosition(position, width = FLOATING_AVATAR_WINDOW_SIZE, height = FLOATING_AVATAR_WINDOW_SIZE) {
  const screenLeft = firstFiniteNumber(window.screen?.availLeft, window.screen?.left, 0) || 0;
  const screenTop = firstFiniteNumber(window.screen?.availTop, window.screen?.top, 0) || 0;
  const screenW = firstFiniteNumber(window.screen?.availWidth, window.screen?.width, window.innerWidth, width) || width;
  const screenH = firstFiniteNumber(window.screen?.availHeight, window.screen?.height, window.innerHeight, height) || height;
  const maxX = screenLeft + Math.max(0, screenW - width);
  const maxY = screenTop + Math.max(0, screenH - height);
  return {
    x: Math.round(clampNumber(Number(position?.x ?? screenLeft), screenLeft, maxX)),
    y: Math.round(clampNumber(Number(position?.y ?? screenTop), screenTop, maxY)),
  };
}

function currentFloatingAvatarScreenBounds(width = FLOATING_AVATAR_WINDOW_SIZE, height = FLOATING_AVATAR_WINDOW_SIZE) {
  const left = firstFiniteNumber(window.screen?.availLeft, window.screen?.left, 0);
  const top = firstFiniteNumber(window.screen?.availTop, window.screen?.top, 0);
  const screenWidth = firstFiniteNumber(window.screen?.availWidth, window.screen?.width, window.innerWidth, width);
  const screenHeight = firstFiniteNumber(window.screen?.availHeight, window.screen?.height, window.innerHeight, height);
  if (!Number.isFinite(left) || !Number.isFinite(top) || !Number.isFinite(screenWidth) || !Number.isFinite(screenHeight)) {
    return null;
  }
  return {left, top, right: left + screenWidth, bottom: top + screenHeight, width: screenWidth, height: screenHeight};
}

function pointInsideFloatingAvatarScreen(point, bounds) {
  if (!bounds || !Number.isFinite(point?.x) || !Number.isFinite(point?.y)) return false;
  return point.x >= bounds.left && point.x <= bounds.right && point.y >= bounds.top && point.y <= bounds.bottom;
}

function roundFloatingAvatarScreenPosition(position, fallback = {x: 0, y: 0}) {
  return {
    x: Math.round(Number.isFinite(position?.x) ? Number(position.x) : Number(fallback?.x || 0)),
    y: Math.round(Number.isFinite(position?.y) ? Number(position.y) : Number(fallback?.y || 0)),
  };
}

function clampFloatingAvatarAnchorPosition(anchor, width = FLOATING_AVATAR_SIZE, height = FLOATING_AVATAR_SIZE) {
  const raw = roundFloatingAvatarScreenPosition(anchor);
  const bounds = currentFloatingAvatarScreenBounds(width, height);
  if (!pointInsideFloatingAvatarScreen(raw, bounds)) {
    return raw;
  }
  return {
    x: Math.round(clampNumber(raw.x, bounds.left, bounds.right - width)),
    y: Math.round(clampNumber(raw.y, bounds.top, bounds.bottom - height)),
  };
}

function compactWindowPositionFromAvatarAnchor(anchor) {
  const safeAnchor = clampFloatingAvatarAnchorPosition(anchor, FLOATING_AVATAR_SIZE, FLOATING_AVATAR_SIZE);
  return {
    x: Math.round(safeAnchor.x - FLOATING_AVATAR_INSET),
    y: Math.round(safeAnchor.y - FLOATING_AVATAR_INSET),
  };
}

const fallbackState = {
  /* i18n: fallbackState */ greeting: _t('greeting.hello'),
  statusRail: {text: _t('greeting.hello'), layer: 'L1', degraded: true, lockedCount: 0},
  // #I-806: adapter 清單由 Sidecar IPC 傳入，經 adapter_registry 白名單審查後更新
  // 預設空陣列：未接 CLI 時不顯示任何 adapter 按鈕
  adapters: [],
  haoras: ['主haㄌer'],
  messages: [],
};

/* i18n: greeting pool — expression metadata shared by default & feminine pools */
const GREETING_POOL_EXPRESSIONS = [
  {expression: 'idle'},
  {expression: 'idle'},
  {expression: 'idle'},
  {expression: 'speechless'},
  {expression: 'idle', rare: true},
  {expression: 'speechless'},
  {expression: 'happy'},
  {expression: 'sleepy'},
  {expression: 'idle'},
  {expression: 'idle'},
];

// variant: 'male' (default) | 'fem' (feminine secretary) | 'wild' (獸人/野性, 本汪)
const GREETING_VARIANT_POOL = {fem: 'greeting.poolFem', wild: 'greeting.poolWild', male: 'greeting.pool'};
const getGreetingRotationOptions = (variant = 'male') => {
  const poolKey = GREETING_VARIANT_POOL[variant] || GREETING_VARIANT_POOL.male;
  return GREETING_POOL_EXPRESSIONS.map((meta, i) => ({...meta, text: _t(`${poolKey}.${i}`)}));
};

// 關鍵字庫（所有語系聯集）：女性稱呼 / 獸人野性
const GREETING_FEMALE_KEYWORDS = getAllFemaleKeywords();
const GREETING_BEAST_KEYWORDS = getAllBeastKeywords();

function matchesAnyKeyword(haystack, keywords) {
  return keywords.some((kw) => {
    if (!kw) return false;
    // ASCII 關鍵字用詞界比對，避免 "sis"/"fox" 命中無關長字之類誤判
    if (/^[a-z0-9 ]+$/.test(kw)) {
      const escaped = kw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      return new RegExp(`(^|[^a-z0-9])${escaped}([^a-z0-9]|$)`).test(haystack);
    }
    return haystack.includes(kw);
  });
}

// 回傳問候語氣：'fem' | 'wild' | 'male'
// 優先序：女性稱呼 → 獸人野性 → 男性(預設)，以保護女性偵測不被野性蓋過
function personaGreetingVariant(persona) {
  if (!persona) return 'male';
  const haystack = [persona.name, persona.identity, persona.personality, persona.scenario, persona.patrolDialogue]
    .filter(Boolean)
    .join(' ')
    .toLowerCase();
  if (!haystack) return 'male';
  if (matchesAnyKeyword(haystack, GREETING_FEMALE_KEYWORDS)) return 'fem';
  if (matchesAnyKeyword(haystack, GREETING_BEAST_KEYWORDS)) return 'wild';
  return 'male';
}

const PATROL_EXPRESSION_ALIASES = {
  '等待': 'idle',
  '待機': 'idle',
  '休息': 'sleepy',
  '睡眠': 'sleepy',
  '行動': 'working',
  '工作': 'working',
  '開心': 'happy',
  '快樂': 'happy',
  '思索': 'thinking',
  '思考': 'thinking',
  '悲傷': 'sad',
  '難過': 'sad',
  '禁止': 'blocked',
  '阻擋': 'blocked',
  '警告': 'warning',
  '注意': 'warning',
  '無言': 'speechless',
  '傻眼': 'speechless',
  idle: 'idle',
  waiting: 'idle',
  sleepy: 'sleepy',
  rest: 'sleepy',
  working: 'working',
  action: 'working',
  happy: 'happy',
  thinking: 'thinking',
  sad: 'sad',
  blocked: 'blocked',
  warning: 'warning',
  speechless: 'speechless',
};
const PATROL_DEFAULT_EXPRESSION = 'idle';

function patrolDialogueOption(text, expression = PATROL_DEFAULT_EXPRESSION, extra = {}) {
  return {
    text,
    expression: PATROL_EXPRESSION_ALIASES[String(expression || '').trim()] || PATROL_DEFAULT_EXPRESSION,
    ...extra,
  };
}

function normalizePatrolDialogueSpec(value) {
  return String(value || '').replace(/\r\n?/g, '\n').trim();
}

function patrolDialogueEntryFromRoleName(spec) {
  const key = normalizePatrolDialogueSpec(spec).toLowerCase();
  if (!key || key.includes('\n')) return null;
  return PATROL_DIALOGUE_ROLE_VARIANTS.find((entry) => entry.names.some((name) => key === String(name).toLowerCase())) || null;
}

function patrolDialogueEntryFromPersona(persona) {
  const spec = normalizePatrolDialogueSpec(persona?.patrolDialogue || persona?.patrol_dialogue || '');
  const specEntry = patrolDialogueEntryFromRoleName(spec);
  if (specEntry) return specEntry;
  if (spec) return null;
  const keys = [persona?.id, persona?.name]
    .filter(Boolean)
    .map((value) => String(value).toLowerCase());
  return PATROL_DIALOGUE_ROLE_VARIANTS.find((entry) => (
    entry.names.some((name) => keys.includes(String(name).toLowerCase()))
  )) || null;
}

function patrolDialogueVariantFromRoleName(spec) {
  return patrolDialogueEntryFromRoleName(spec)?.variant || '';
}

function patrolDialogueOptionsFromEntry(entry) {
  if (!entry) return [];
  return entry.options || getGreetingRotationOptions(entry.variant);
}

function stripPatrolDialogueLineMeta(line) {
  let text = String(line || '').trim();
  const meta = {};
  if (/\(\s*10%\s*機率\s*\)\s*$/i.test(text)) {
    meta.rare = true;
    meta.rareChance = 0.1;
    text = text.replace(/\(\s*10%\s*機率\s*\)\s*$/i, '').trim();
  }
  const idleMatch = text.match(/\(\s*不動\s*(\d+)\s*分鐘\s*\)\s*$/);
  if (idleMatch) {
    meta.idleAfterMinutes = Number.parseInt(idleMatch[1], 10);
    text = text.replace(/\(\s*不動\s*\d+\s*分鐘\s*\)\s*$/, '').trim();
  }
  if (/\(\s*一開始的時候\s*\)\s*$/.test(text)) {
    meta.initial = true;
    text = text.replace(/\(\s*一開始的時候\s*\)\s*$/, '').trim();
  }
  return {text, meta};
}

function trimPatrolDialogueQuotes(value) {
  return String(value || '').trim().replace(/^["“”]+|["“”]+$/g, '').trim();
}

function parsePatrolDialogueLine(line) {
  const {text: withoutMeta, meta} = stripPatrolDialogueLineMeta(line);
  const labels = Object.keys(PATROL_EXPRESSION_ALIASES).sort((a, b) => b.length - a.length);
  for (const label of labels) {
    const markers = [`""${label}"`, `""${label}`, `"${label}"`, `"${label}`];
    const marker = markers.find((candidate) => withoutMeta.endsWith(candidate));
    if (marker) {
      return patrolDialogueOption(
        trimPatrolDialogueQuotes(withoutMeta.slice(0, -marker.length)),
        label,
        meta,
      );
    }
    const separatorMatch = withoutMeta.match(new RegExp(`^(.*?)[|｜,，:：]\\s*${label}$`, 'i'));
    if (separatorMatch) {
      return patrolDialogueOption(trimPatrolDialogueQuotes(separatorMatch[1]), label, meta);
    }
  }
  return patrolDialogueOption(trimPatrolDialogueQuotes(withoutMeta), PATROL_DEFAULT_EXPRESSION, meta);
}

function manualPatrolDialogueOptions(spec) {
  const roleVariant = patrolDialogueVariantFromRoleName(spec);
  if (roleVariant) return [];
  return normalizePatrolDialogueSpec(spec)
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map(parsePatrolDialogueLine)
    .filter((option) => option.text);
}

function defaultPatrolDialogueVariant(persona) {
  return personaGreetingVariant(persona) === 'fem' ? 'fem' : 'wild';
}

function getPersonaPatrolDialogueOptions(persona) {
  const spec = normalizePatrolDialogueSpec(persona?.patrolDialogue || persona?.patrol_dialogue || '');
  const roleOptions = patrolDialogueOptionsFromEntry(patrolDialogueEntryFromPersona(persona));
  if (roleOptions.length > 0) return roleOptions;
  const manualOptions = manualPatrolDialogueOptions(spec);
  if (manualOptions.length > 0) return manualOptions;
  return getGreetingRotationOptions(defaultPatrolDialogueVariant(persona));
}

function personaPatrolDialogueBadge(persona) {
  const spec = normalizePatrolDialogueSpec(persona?.patrolDialogue || persona?.patrol_dialogue || '');
  const entry = patrolDialogueEntryFromPersona(persona);
  if (entry?.label) return entry.label;
  if (!spec) return defaultPatrolDialogueVariant(persona) === 'fem' ? 'AssiStand' : 'YuRoSaKu';
  return manualPatrolDialogueOptions(spec).length > 0 ? 'Manual' : '';
}

function personaNameDisplayUnits(name = '') {
  return Array.from(String(name || '').trim()).reduce((total, char) => {
    if (/\s/.test(char)) return total + 0.45;
    if (/[\u3040-\u30ff\u3400-\u9fff\uf900-\ufaff]/u.test(char)) return total + 1.35;
    if (/[\u1100-\u11ff\u3130-\u318f\uac00-\ud7af]/u.test(char)) return total + 1.12;
    if (/[\u0e00-\u0e7f]/u.test(char)) return total + 0.95;
    return total + 1;
  }, 0);
}

function personaNameFontSize(name = '') {
  const units = personaNameDisplayUnits(name);
  if (units <= 8) return 17;
  if (units <= 12) return 15;
  if (units <= 15.5) return 13.5;
  if (units <= 19) return 12;
  return 10.5;
}

function personaCardDisplayName(persona = {}, t = _t) {
  const display = DEFAULT_PERSONA_DISPLAY_NAMES[persona.id];
  if (!display) return persona.name || '';
  const name = String(persona.name || '').trim();
  const localizedName = t(display.key);
  if (persona.id === LOCKED_PERSONA_ID || !name || display.legacyNames.includes(name)) {
    return localizedName;
  }
  return persona.name;
}

function pickRotatingGreeting(currentText, source = 'male') {
  const options = Array.isArray(source) ? source : getGreetingRotationOptions(source);
  const regular = options.filter((item) => !item.rare);
  const rare = options.filter((item) => item.rare);
  const rareChance = rare.reduce((max, item) => Math.max(max, Number(item.rareChance) || 0.05), 0.05);
  const basePool = rare.length > 0 && Math.random() < rareChance ? rare : regular;
  let pool = basePool.filter((item) => item.text !== currentText);
  if (pool.length === 0) {
    pool = regular.filter((item) => item.text !== currentText);
  }
  if (pool.length === 0) {
    pool = Array.isArray(source) ? source : getGreetingRotationOptions(source);
  }
  return pool[Math.floor(Math.random() * pool.length)];
}

// 由所有語系問候字串自動產生（含 poolFem），避免手動維護；外加少數歷史別名
const LEGACY_GREETING_ALIASES = {
  'Hi 主人，今天天氣不錯！': 'greeting.pool.0',
  '你好，主人。': 'greeting.pool.0',
  '先喘口氣也可以。': 'greeting.pool.7',
  '需要時叫我一聲。': 'greeting.pool.1',
};
const statusRailTextI18nKeys = {...buildGreetingTextKeyMap(), ...LEGACY_GREETING_ALIASES};

function localizeStatusRailText(text) {
  const normalized = String(text || '').trim();
  if (!normalized) return text;
  const key = statusRailTextI18nKeys[normalized];
  return key ? _t(key) : text;
}

function localizeStatusRailView(view) {
  if (!view) return view;
  const rawText = view.rawText || view.text;
  return {...view, rawText, text: localizeStatusRailText(rawText)};
}

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
    cliVersion: adapter?.cliVersion ?? adapter?.cli_version ?? '',
    modelOptionSource: adapter?.modelOptionSource ?? adapter?.model_option_source ?? '',
    modelOptionNote: adapter?.modelOptionNote ?? adapter?.model_option_note ?? '',
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

function applyAdapterStatusUpdate(items, payload) {
  const adapterID = String(payload?.adapter_id || payload?.adapterId || '').trim();
  const status = String(payload?.status || '').trim().toLowerCase();
  if (!adapterID || !status || !Array.isArray(items)) return items;
  return items.map((item) => (
    adapterKey(item) === adapterID ? {...item, status} : item
  ));
}

function sanitizeDisplayedCLIError(error) {
  const raw = String(error || '').trim();
  if (!raw) return raw;
  return raw
    .replace(/^cli rpc error:\s*/i, '')
    .replace(/^Error:\s*/i, '')
    .trim();
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
const dagTaskRequestPattern = /(幫我|幫忙|請你|請幫我|麻煩|替我|我要|我想要|我需要|幫我把|幫我處理)/;
const dagStrongTaskActionPattern = /(搜尋|查詢|整理|建立|新增|執行|打開|開啟|分析|寄送|寄出|下載|安裝|錄製|學習|製作|產生|產出|生成|修改|修正|更新|檢查|測試|驗證|比較|轉換|匯出|匯入|讀取|刪除|計算)/;
const dagWeakTaskWithObjectPattern = /(查|寫|做).{0,12}(檔案|文件|報告|摘要|表格|簡報|程式|程式碼|網頁|網站|工具|小工具|計算機|清單|圖表|測試|驗證|天氣|地圖|資料)/;
const dagDomainTaskPattern = /(搜尋|查詢|查|打開|開啟|導航|規劃).{0,12}(天氣|地圖|路線|地址|位置)/;
const dagQuestionIntentPattern = /(請問|想問|問一下|你覺得|你認為|為什麼|什麼是|怎麼|如何|怎樣|可不可以解釋|可以解釋|告訴我.*(怎麼|如何|為什麼|什麼))/;
const dagLeadingQuestionPattern = /^(請問|想問|問一下|你覺得|你認為|為什麼|什麼是|怎麼看|如何理解|可以解釋|你好|嗨|早安|晚安|謝謝)/;
const dagAmbiguousTaskSignalPattern = /(看|查|寫|做|找|弄|處理|研究|幫我|幫忙|請你|我要|我想要|我需要|天氣|地圖|路線|資料|檔案|文件|報告|程式|工具|小工具|計算機)/;
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
function candidateReplyText(candidate) {
  const draft = stripInternalControlDraft(candidate?.draft || candidate?.label || '');
  return draft.startsWith('input:') ? draft.slice('input:'.length).trim() : draft.trim();
}

const GACHA_PULSE_MS = 500;
const GACHA_PARTICLE_MS = 600;
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
const getChatTonePresets = () => [
  {id: 'deadpan_weary', label: _t('persona.toneDeadpanWeary')},
  {id: 'stoic_tough', label: _t('persona.toneStoicTough')},
  {id: 'calm_rational', label: _t('persona.toneCalmRational')},
  {id: 'warm_gentle', label: _t('persona.toneWarmGentle')},
  {id: 'upbeat_energetic', label: _t('persona.toneUpbeatEnergetic')},
  {id: 'witty_playful', label: _t('persona.toneWittyPlayful')},
  {id: 'tsundere', label: _t('persona.toneTsundere')},
];
const voiceProfileLabelKeys = {
  lively_male: 'settings.voiceProfileLivelyMale',
  low_uncle_male: 'settings.voiceProfileLowUncleMale',
  professional_female: 'settings.voiceProfileProfessionalFemale',
  excited_clear_male: 'settings.voiceProfileExcitedClearMale',
  bright_girl: 'settings.voiceProfileBrightGirl',
};
const defaultVoiceIdByPersonaId = {
  'persona-a': 'lively_male',
  'persona-b': 'low_uncle_male',
  'persona-c': 'professional_female',
  'persona-d': 'excited_clear_male',
  'persona-e': 'bright_girl',
};
function localizedVoiceProfileName(profile, t) {
  const key = voiceProfileLabelKeys[profile?.voiceId];
  return key ? t(key) : profile?.displayName || profile?.voiceId || '';
}
const avatarStateOptions = ['idle', 'thinking', 'working', 'working_reaction', 'happy', 'warning', 'blocked', 'sleepy', 'sad', 'speechless'];
/* i18n: avatar state labels */
const getAvatarStateLabels = () => ({
  idle: _t('avatar.idle'),
  thinking: _t('avatar.thinking'),
  working: _t('avatar.working'),
  working_reaction: '工作反應',
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

function normalizeReviewState(value) {
  const source = value && typeof value === 'object' ? value : {};
  const sourceHighRisk = source.highRisk && typeof source.highRisk === 'object' ? source.highRisk : {};
  const sourceLowRisk = source.lowRiskAmbiguity && typeof source.lowRiskAmbiguity === 'object' ? source.lowRiskAmbiguity : {};
  const sourceHookCandidates = source.hookCandidates && typeof source.hookCandidates === 'object' ? source.hookCandidates : {};
  return {
    ...fallbackReviewState,
    ...source,
    highRisk: {
      ...fallbackReviewState.highRisk,
      ...sourceHighRisk,
    },
    lowRiskAmbiguity: {
      ...fallbackReviewState.lowRiskAmbiguity,
      ...sourceLowRisk,
      candidates: Array.isArray(sourceLowRisk.candidates) ? sourceLowRisk.candidates : [],
    },
    hookCandidates: {
      ...fallbackReviewState.hookCandidates,
      ...sourceHookCandidates,
      tagPatches: Array.isArray(sourceHookCandidates.tagPatches) ? sourceHookCandidates.tagPatches : [],
      subagentCandidates: Array.isArray(sourceHookCandidates.subagentCandidates) ? sourceHookCandidates.subagentCandidates : [],
      registryProposals: Array.isArray(sourceHookCandidates.registryProposals) ? sourceHookCandidates.registryProposals : [],
    },
    pendingPackages: Array.isArray(source.pendingPackages) ? source.pendingPackages : [],
  };
}

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
  if (/^\/dag\b/i.test(raw) || /^#dag\b/i.test(raw)) return true;
  // 自然語言只做高信心自動任務判定：先排除問話，再要求「請系統做事」
  // 且命中明確動詞；查/寫/做 這類弱動詞必須帶明確產物或領域。
  if (dagQuestionIntentPattern.test(raw)) return false;
  if (raw.length < 6) return false;
  if (!dagTaskRequestPattern.test(raw)) return false;
  return dagStrongTaskActionPattern.test(raw)
    || dagWeakTaskWithObjectPattern.test(raw)
    || dagDomainTaskPattern.test(raw);
}

function shouldSuggestDagRun(text) {
  const raw = String(text || '').trim();
  if (!raw) return false;
  if (/^\/dag\b/i.test(raw) || /^#dag\b/i.test(raw)) return false;
  if (shouldCreateDagRun(raw)) return false;
  if (dagLeadingQuestionPattern.test(raw)) return false;
  if (raw.length < 4) return false;
  if (!dagTaskRequestPattern.test(raw)) return false;
  return dagQuestionIntentPattern.test(raw) || dagAmbiguousTaskSignalPattern.test(raw);
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

const adapterMeta = {
  Claude: {className: 'adapter-lime', icon: 'C'},
  Codex: {className: 'adapter-pink', icon: '◎'},
  Gemini: {className: 'adapter-lime', icon: '✦'},
};

const maxPersonas = 16;
const lockedPersonaId = LOCKED_PERSONA_ID;

const pixelAvatarRenderSize = 128;
const pixelAvatarRenderVersion = '2026-06-21-transparent-touharu-raster-v1';
const pixelAvatarRenderCache = new Map();
const autoAvatarOverrideStates = new Set(['blocked', 'warning', 'working', 'working_reaction']);

function formatVisualLearningPermissionStatus(status, requested = false) {
  const missing = Array.isArray(status?.missing) ? status.missing : [];
  if (!missing.length && status?.accessibility && status?.input_monitoring) {
    return requested
      ? _t('chatSystem.learning.permissionsAllowedAfterRequest')
      : _t('chatSystem.learning.permissionsAllowed');
  }
  const list = missing.length ? missing.join(_t('chatSystem.listSeparator')) : _t('chatSystem.learning.defaultPermissions');
  return _t('chatSystem.learning.permissionsMissing', {list});
}

function isNoActiveLearningRecordingError(detail) {
  return /no active recording/i.test(String(detail || ''));
}

function openFirstMissingVisualLearningPermission(status) {
  const missingKeys = Array.isArray(status?.missing_keys) ? status.missing_keys : [];
  const first = missingKeys[0];
  if (!first) return Promise.resolve(false);
  return callWails(() => OpenVisualLearningPermissionSettings(first))
    .then(() => true)
    .catch(() => false);
}

function App() {
  /* i18n: component hook */
  const t = useI18n(s => s.t);
  const [state, setState] = useState(fallbackState);
  const [messagesByAgent, setMessagesByAgent] = useState({main: fallbackState.messages});
  const [draft, setDraft] = useState('');
  const [dismissedDagSuggestionText, setDismissedDagSuggestionText] = useState('');
  const [dagClarificationPending, setDagClarificationPending] = useState(false);
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
  const [skillExportDialog, setSkillExportDialog] = useState(null);
  const [referenceFiles, setReferenceFiles] = useState([]);
  // 資料區程式碼卡片 meta（file_name → meta）＋展開彈窗
  const [codeArtifacts, setCodeArtifacts] = useState({});
  const [codeArtifactModal, setCodeArtifactModal] = useState(null);
  const codeArtifactActivityLogsRef = useRef({});
  const [referenceExportDialog, setReferenceExportDialog] = useState(null);
  const [searchSummaryExportDialog, setSearchSummaryExportDialog] = useState(null);
  const [sharedSourceActionDialog, setSharedSourceActionDialog] = useState(null);
  const [referenceLinkOpen, setReferenceLinkOpen] = useState(false);
  const [referenceLinkValue, setReferenceLinkValue] = useState('');
  const referenceInternalDragRef = useRef(false);
  const referenceDropSuppressUntilRef = useRef(0);
  const referenceImportInFlightRef = useRef(new Set());
  const [settingsState, setSettingsState] = useState(fallbackSettings);
  const [tools, setTools] = useState(getFallbackTools());
  const [toolResult, setToolResult] = useState(null);
  const [schedulerPanelOpen, setSchedulerPanelOpen] = useState(false);
  const [schedulerClock, setSchedulerClock] = useState(null);
  const [schedulerJobs, setSchedulerJobs] = useState([]);
  const [schedulerBusy, setSchedulerBusy] = useState(false);
  const [schedulerError, setSchedulerError] = useState('');
  const [schedulerDraft, setSchedulerDraft] = useState({
    name: '',
    cronExpr: '@daily',
    actionPayload: '',
  });
  const [schedulerConversation, setSchedulerConversation] = useState(null);

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
  const learningDigestReadyRef = useRef(false);
  const learningDigestPreparingRef = useRef(false);

  useEffect(() => {
    refreshSchedulerClock();
    refreshSchedulerJobs();
  }, []);

  // ── v3.6.2 WA3 Media Provenance（§9A）state ──
  // wa3ImportPopup   : 匯入選單資料（ImportResult 結構，null = 關閉）
  // wa3ToastMsg      : 傳輸引導 toast 訊息（null = 隱藏）
  // wa3TrustList     : 信任清單陣列
  const [wa3ImportPopup, setWa3ImportPopup] = useState(null);
  const [wa3ToastMsg, setWa3ToastMsg] = useState(null);
  const [wa3TrustList, setWa3TrustList] = useState([]);
  const [wa3Detail, setWa3Detail] = useState(null);
  const [wa3PollutionResult, setWa3PollutionResult] = useState(null);
  const [wa3TransferGuidance, setWa3TransferGuidance] = useState(null);
  const [wa3ActionBusy, setWa3ActionBusy] = useState('');
  const [wa3ActionError, setWa3ActionError] = useState('');

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
  const [extSharedLinks, setExtSharedLinks] = useState([]);
  // 共用資料夾第一層檔案清單（key = 資料夾路徑；只讀該層，不遞迴）
  const [sharedSourceListings, setSharedSourceListings] = useState([]);
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
  const adapterModelChoicesRef = useRef({});
  adapterModelChoicesRef.current = adapterModelChoices;
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
  // 語音聊天模式：AI 回覆自動以當前角色聲線朗讀（spec §34.5/§34.6 表現層）。
  const [voiceChatEnabled, setVoiceChatEnabled] = useState(false);
  const [voiceChatHint, setVoiceChatHint] = useState(false);
  const voiceChatEnabledRef = useRef(false);
  useEffect(() => {
    const enabled = Boolean(voiceState?.settings?.ttsEnabled);
    setVoiceChatEnabled(enabled);
    voiceChatEnabledRef.current = enabled;
  }, [voiceState]);
  const [voiceRecording, setVoiceRecording] = useState(false);
  // voice_sync_session（spec §34.1）：語音聊天模式下點一下進入同步聆聽。
  const [voiceSyncActive, setVoiceSyncActive] = useState(false);
  const [voiceSyncPhase, setVoiceSyncPhase] = useState('idle');
  const voiceSyncRef = useRef({active: false, phase: 'idle', lastVoiceAt: 0, spokeMs: 0, committing: false, startedAt: 0});
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
  useHighlight({enabled: true, conversationId: activeConversationId});
  const activeConversationIdRef = useRef('main');
  const loadedConversationIdsRef = useRef(new Set(['main']));
  // 釘選對話框（popout）：被釘成獨立 OS 視窗的 sub 對話清單。
  // 釘選期間該分頁在主視窗鎖定（記憶由子視窗獨佔），點擊分頁＝收回。
  const [pinnedPopouts, setPinnedPopouts] = useState([]);
  const pinnedPopoutsRef = useRef([]);
  pinnedPopoutsRef.current = pinnedPopouts || [];
  useEffect(() => {
    // 啟動時與後端同步一次（開發熱重載時清單不會遺失）。
    callWails(() => window.go?.main?.App?.ListPinnedPopouts?.())
      .then((list) => setPinnedPopouts(Array.isArray(list) ? list : []))
      .catch(() => {});
    const offPopoutChanged = EventsOn('popout:changed', (list) => {
      const next = Array.isArray(list) ? list : [];
      const nextIds = new Set(next.map((item) => item?.agent_id));
      // 回彈的對話：popout 期間記憶檔已被子視窗更新，強制從 talk_full.md 重讀。
      (pinnedPopoutsRef.current || []).forEach((item) => {
        if (item?.agent_id && !nextIds.has(item.agent_id)) {
          loadConversationMessages(item.agent_id, {force: true});
        }
      });
      setPinnedPopouts(next);
    });
    return () => { offPopoutChanged?.(); };
  }, []);

  function pinOutConversation(haora) {
    if (!haora || haora.isMain) return; // 只有 MAIN 能調動所有記憶，主haㄌer不可釘出
    const agentId = haora.id || haora.name;
    const adapter = resolveAdapterFromRefs();
    const adapterID = adapter?.id || activeAdapterIdRef.current || '';
    const payload = {
      agent_id: agentId,
      name: haora.name || agentId,
      // 釘選當下「吃當前主人格和模型」：綁定現用 adapter/model 與主人格名。
      adapter_id: adapterID,
      is_api: isAPIAdapter(adapter),
      persona_id: mainPersona?.id || '',
      persona_name: mainPersona?.name || '',
      model: adapterModelChoicesRef.current?.[adapterID] || adapter?.model || '',
      locale: useI18n.getState().language || 'zh-TW',
    };
    // 若釘出的分頁正在使用，先切回主haㄌer，避免主/子視窗同時寫同一份記憶。
    if (activeHaoraId === agentId || activeHaoraId === haora.name) setActiveHaoraId(null);
    callWails(() => window.go?.main?.App?.PinOutConversation?.(payload))
      .catch((err) => {
        setToolResult({toolId: 'subagent', ok: false, message: String(err?.message || err)});
      });
  }

  function unpinConversation(haora) {
    const agentId = haora?.id || haora?.name;
    if (!agentId) return;
    callWails(() => window.go?.main?.App?.UnpinConversation?.(agentId)).catch(() => {});
  }
  const messages = messagesByAgent[activeConversationId] || [];
  useSystemMarks({messages, conversationId: activeConversationId});
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
    const valueSnapshot = new WeakMap();
    const baseKeyboardPayload = (event, target, described) => ({
      timestamp: new Date().toISOString(),
      x: Math.round(event.clientX || 0),
      y: Math.round(event.clientY || 0),
      source: 'webview',
      coordinate_space: 'viewport',
      target_region_id: `vl-key-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
      target_label: described.label,
      target_role: described.role,
      target_tag: described.tag,
      css_selector: described.selector,
      target_rect: described.rect,
      viewport: {
        width: window.innerWidth,
        height: window.innerHeight,
        device_scale: window.devicePixelRatio || 1,
      },
    });
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
    const handleLearningInput = (event) => {
      if (learningReplayExecutingRef.current) return;
      const target = event.target;
      if (!target || target?.closest?.(learningRecordingBlockedSelector)) return;
      const isTextControl = target instanceof HTMLInputElement
        || target instanceof HTMLTextAreaElement
        || target?.isContentEditable;
      if (!isTextControl) return;
      const described = describeLearningTarget(target);
      const rawValue = target?.isContentEditable ? (target.innerText || target.textContent || '') : String(target.value || '');
      const previous = valueSnapshot.get(target) || '';
      valueSnapshot.set(target, rawValue);
      const sensitive = target instanceof HTMLInputElement && String(target.type || '').toLowerCase() === 'password';
      let inserted = typeof event.data === 'string' ? event.data : '';
      if (!inserted && rawValue.length > previous.length && rawValue.startsWith(previous)) {
        inserted = rawValue.slice(previous.length);
      }
      if (!inserted && !sensitive) return;
      const highRisk = sensitive || learningSensitiveTextPattern.test(`${inserted} ${described.label} ${described.selector}`);
      const payload = {
        ...baseKeyboardPayload(event, target, described),
        event_type: sensitive ? 'sensitive_text' : 'text',
        text: sensitive ? '' : inserted,
        key: '',
        modifiers: [],
        playback: 'paste_safe',
        sensitive,
        requires_confirmation: highRisk && !sensitive,
      };
      callWails(() => RecordLearningKeyboardEvent(JSON.stringify(payload))).catch(() => {});
    };
    const handleLearningKeyDown = (event) => {
      if (learningReplayExecutingRef.current || event.isComposing) return;
      const target = event.target;
      if (!target || target?.closest?.(learningRecordingBlockedSelector)) return;
      const described = describeLearningTarget(target);
      const modifiers = [
        event.metaKey ? 'cmd' : '',
        event.ctrlKey ? 'ctrl' : '',
        event.altKey ? 'option' : '',
        event.shiftKey ? 'shift' : '',
      ].filter(Boolean);
      const key = normalizeLearningKey(event.key);
      const isShortcut = modifiers.length > 0 && key.length > 0;
      const isControlKey = ['Enter', 'Tab', 'Escape'].includes(key);
      if (!isShortcut && !isControlKey) return;
      const payload = {
        ...baseKeyboardPayload(event, target, described),
        event_type: isShortcut ? 'shortcut' : 'key',
        text: '',
        key,
        modifiers,
        playback: 'type',
        sensitive: false,
        requires_confirmation: false,
      };
      callWails(() => RecordLearningKeyboardEvent(JSON.stringify(payload))).catch(() => {});
    };
    window.addEventListener('click', handleLearningClick, true);
    window.addEventListener('input', handleLearningInput, true);
    window.addEventListener('keydown', handleLearningKeyDown, true);
    return () => {
      window.removeEventListener('click', handleLearningClick, true);
      window.removeEventListener('input', handleLearningInput, true);
      window.removeEventListener('keydown', handleLearningKeyDown, true);
    };
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

  const openSharedSourceActionDialog = (link) => {
    if (!link) return;
    const sourcePath = String(link?.url || link?.URL || '');
    const sourceID = String(link?.id || link?.ID || '');
    const label = String(link?.label || link?.Label || fileBaseName(sourcePath) || t('rightRail.sharedLink'));
    setSharedSourceActionDialog({
      id: sourceID,
      url: sourcePath,
      name: label,
      detail: sourcePath,
    });
  };

  const handleSharedSourceAction = async (action) => {
    if (!sharedSourceActionDialog) return;
    const target = sharedSourceActionDialog;
    setSharedSourceActionDialog(null);
    if (action !== 'remove') return;
    const key = target.id || target.url;
    if (!key) return;
    try {
      await callWails(() => RemoveExternalLink('shared_source', key));
      setExtSharedLinks((current) => (current || []).filter((link) => {
        const id = String(link?.id || link?.ID || '');
        const url = String(link?.url || link?.URL || '');
        return id !== key && url !== key;
      }));
      await refreshExternalLinks();
      setToolResult({toolId: 'reference-link', ok: true, message: '共用來源已移除'});
    } catch (error) {
      setToolResult({toolId: 'reference-link', ok: false, message: error?.message || String(error)});
      refreshExternalLinks();
    }
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
  // v3.1.6：task:loop_round 事件 → {nodeId: {iteration, action, target}}；節點卡片顯示「第 N 輪」。
  const [taskLoopRounds, setTaskLoopRounds] = useState({});
  const [taskLoopReply, setTaskLoopReply] = useState({});
  const dagRunRef = useRef(null);
  // 一般對話／skill 送出中的 trace（非 DAG）。讓停止鈕在閒聊/skill 執行時也能中斷。
  const [activeChatTrace, setActiveChatTrace] = useState(null);
  const activeChatTraceRef = useRef(null);
  const cancelledChatTracesRef = useRef(new Set());
  const [pendingTaskReview, setPendingTaskReview] = useState(null);
  // #I-805: Sidecar 狀態追蹤 — 已移至上方（502 行），統一用字串格式。
  // 舊版用 {status, error} object，現在統一為 'unknown' | 'idle' | 'running' | 'crashed' | 'start_failed'。

  // #I-1001: 自動封存 Toast 提示 + 系統狀態歷史
  const [autoArchiveToast, setAutoArchiveToast] = useState(null);
  const [adapterModelRepairToast, setAdapterModelRepairToast] = useState(null);
  const [systemStatusHistory, setSystemStatusHistory] = useState([]);

  // §30: 關閉視窗 → 存成 sub 對話框
  const [sessionClosePrompt, setSessionClosePrompt] = useState(null);
  const [schedulerBgPrompt, setSchedulerBgPrompt] = useState(false);
  const [schedulerBgBusy, setSchedulerBgBusy] = useState(false);
  const [schedulerConfirm, setSchedulerConfirm] = useState(null);
  const [schedulerConfirmBusy, setSchedulerConfirmBusy] = useState(false);
  const [floatingAvatarMode, setFloatingAvatarMode] = useState(false);
  const [floatingAvatarFlyingBack, setFloatingAvatarFlyingBack] = useState(false);
  const [floatingAvatarCompactWindow, setFloatingAvatarCompactWindow] = useState(false);
  const [floatingAvatarSurfaceOnly, setFloatingAvatarSurfaceOnly] = useState(false);
  const [floatingAvatarBodyMode, setFloatingAvatarBodyMode] = useState('head');
  const [floatingAvatarDynamicImage, setFloatingAvatarDynamicImage] = useState(false);
  const [floatingAvatarMotionMode, setFloatingAvatarMotionMode] = useState(() => {
    try {
      const saved = window.localStorage.getItem('floating_avatar_motion_mode');
      if (saved === 'frames' || saved === 'rig') return saved;
    } catch {}
    return 'rig';
  });
  const [floatingAvatarChatMode, setFloatingAvatarChatMode] = useState(false);
  const [floatingAvatarPanelOpenSignal, setFloatingAvatarPanelOpenSignal] = useState(0);
  const [floatingAvatarReplyBubble, setFloatingAvatarReplyBubble] = useState('');
  const [floatingPhotoBusy, setFloatingPhotoBusy] = useState(false);
  const [floatingAvatarDrafts, setFloatingAvatarDrafts] = useState({});
  const [floatingReminderPause, setFloatingReminderPause] = useState({mode: '', until: 0});
  const floatingAvatarWindowRef = useRef({restore: null, compactPosition: null});
  const floatingAvatarDragWindowRef = useRef(null);
  const floatingAvatarDragFrameRef = useRef(0);
  const floatingAvatarPendingWindowPositionRef = useRef(null);
  const floatingAvatarLastCompactPositionRef = useRef(null);
  const [floatingAvatarPosition, setFloatingAvatarPosition] = useState(() => {
    try {
      const saved = JSON.parse(window.localStorage.getItem('floating_avatar_position') || 'null');
      if (saved && Number.isFinite(saved.x) && Number.isFinite(saved.y)) {
        return normalizeFloatingAvatarViewportPosition(saved);
      }
    } catch {}
    return floatingAvatarCenterPosition();
  });
  const floatingAvatarModeRef = useRef(floatingAvatarMode);
  const floatingAvatarCompactWindowRef = useRef(floatingAvatarCompactWindow);
  const floatingAvatarBodyModeRef = useRef(floatingAvatarBodyMode);
  const floatingAvatarPositionRef = useRef(floatingAvatarPosition);
  const floatingAvatarTransitionRef = useRef(false);
  const floatingAvatarChatModeRef = useRef(floatingAvatarChatMode);
  const floatingAvatarChatHistoryRef = useRef([]);
  const floatingAvatarChatRequestSeqRef = useRef(0);
  floatingAvatarModeRef.current = floatingAvatarMode;
  floatingAvatarCompactWindowRef.current = floatingAvatarCompactWindow;
  floatingAvatarBodyModeRef.current = floatingAvatarBodyMode;
  floatingAvatarPositionRef.current = floatingAvatarPosition;
  floatingAvatarChatModeRef.current = floatingAvatarChatMode;

  useEffect(() => {
    if (floatingAvatarMode || floatingAvatarCompactWindow) return;
    try {
      window.localStorage.setItem('floating_avatar_position', JSON.stringify(floatingAvatarPosition));
    } catch {}
  }, [floatingAvatarCompactWindow, floatingAvatarMode, floatingAvatarPosition]);

  useEffect(() => {
    const nativeWindowActive = floatingAvatarMode && (ENABLE_NATIVE_FLOATING_AVATAR_WINDOW || floatingAvatarCompactWindow || floatingAvatarSurfaceOnly);
    document.documentElement.classList.toggle('floating-avatar-window-active', nativeWindowActive);
    document.body.classList.toggle('floating-avatar-window-active', nativeWindowActive);
    // v2.7：Windows 已走 mac 接法（真透明原生浮窗），不再套 keyed 不透明底色。
    // keyed class 與其 CSS 保留給日後 fallback（此處恆為 false）。
    const keyedFallback = false && IS_WINDOWS_RUNTIME;
    document.documentElement.classList.toggle('floating-avatar-window-keyed', nativeWindowActive && keyedFallback);
    document.body.classList.toggle('floating-avatar-window-keyed', nativeWindowActive && keyedFallback);
    return () => {
      document.documentElement.classList.remove('floating-avatar-window-active');
      document.documentElement.classList.remove('floating-avatar-window-keyed');
      document.body.classList.remove('floating-avatar-window-active');
      document.body.classList.remove('floating-avatar-window-keyed');
    };
  }, [floatingAvatarMode, floatingAvatarCompactWindow, floatingAvatarSurfaceOnly]);

  // v2.7 fix：浮窗模式下，臨時情緒表情（happy/sad/speechless）數秒後自動歸還 idle。
  // 原因：manifest 只有 idle（含轉身/走路）拆了骨架部件，表情狀態是單張平面立繪
  // （layers=[]，by design）。進浮窗的問候 happy 若不收回，manualAvatarState 會
  // 永遠霸佔表情，Pixi 只能一直畫平面 fallback——跟滑鼠轉頭、轉向階梯、發呆走路
  // 全部無法啟動。工作中/警告/封鎖等「任務態」不在此列，由任務結束自行復原。
  useEffect(() => {
    if (!floatingAvatarMode) return undefined;
    if (!['happy', 'sad', 'speechless'].includes(manualAvatarState)) return undefined;
    const timer = window.setTimeout(() => {
      setManualAvatarState('');
    }, 5000);
    return () => window.clearTimeout(timer);
  }, [floatingAvatarMode, manualAvatarState]);

  useEffect(() => {
    if (!floatingAvatarMode) return undefined;
    const handleKeyDown = (event) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      restoreFromFloatingAvatar('auto');
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [floatingAvatarMode]);

  useEffect(() => {
    const until = Number(floatingReminderPause.until || 0);
    if (floatingReminderPause.mode === 'manual' || !until || until <= Date.now()) return undefined;
    const timer = window.setTimeout(() => setFloatingReminderPause({mode: '', until: 0}), Math.max(1000, until - Date.now()));
    return () => window.clearTimeout(timer);
  }, [floatingReminderPause.mode, floatingReminderPause.until]);

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
  const [selectedFloatingCandidateIDs, setSelectedFloatingCandidateIDs] = useState([]);
  const [longPressActive, setLongPressActive] = useState(false);
  const [longPressProgress, setLongPressProgress] = useState(0);
  const [gachaPhase, setGachaPhase] = useState(null);
  const [riskImpactExpanded, setRiskImpactExpanded] = useState(false);
  const floatingCandidateKey = (readinessGate.floating_candidates || [])
    .map((candidate) => String(candidate?.id || ''))
    .join('|');
  const longPressTimerRef = useRef(null);
  const longPressStartRef = useRef(null);
  const readinessBurstTimerRef = useRef(null);
  const manualGreetingLockedRef = useRef(false);

  useEffect(() => () => {
    if (readinessBurstTimerRef.current) {
      window.clearInterval(readinessBurstTimerRef.current);
      readinessBurstTimerRef.current = null;
    }
  }, []);

  useEffect(() => {
    const liveIDs = new Set((readinessGate.floating_candidates || []).map((candidate) => candidate?.id));
    setSelectedFloatingCandidateIDs((prev) => prev.filter((id) => liveIDs.has(id)));
  }, [floatingCandidateKey]);

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

  function buildAdapterModelRepairMessage({adapterID, model, fallbackModel, reason}) {
    const adapter = adapterList.find((item) => (item.id || item.name) === adapterID);
    const label = adapter?.displayName || adapter?.name || adapterID || 'adapter';
    if (fallbackModel) {
      return t('adapter.modelRepairWithFallback', {label, model, fallbackModel});
    }
    if (reason && reason.includes('fallback list')) {
      return t('adapter.modelRepairFallbackList', {label, model});
    }
    return t('adapter.modelRepairCleared', {label, model});
  }

  async function reconcileAdapterModelChoicesAgainstOptions(choices, optionMap) {
    const nextChoices = {...(choices || {})};
    const repaired = [];
    for (const [adapterID, model] of Object.entries(choices || {})) {
      const opts = optionMap?.[adapterID];
      if (!model || !Array.isArray(opts) || opts.length === 0 || opts.includes(model)) continue;
      delete nextChoices[adapterID];
      repaired.push({adapterID, model, fallbackModel: opts[0] || '', reason: 'options_mismatch'});
      try {
        await callWails(() => SetAdapterModelChoice(adapterID, ''));
      } catch (_) {}
    }
    setAdapterModelChoices(nextChoices);
    if (repaired.length > 0) {
      setAdapterModelRepairToast(buildAdapterModelRepairMessage(repaired[0]));
    }
    return nextChoices;
  }

  async function refreshAdapterModelOptionsForAdapters(adapters = [], choices = adapterModelChoicesRef.current) {
    const out = {};
    for (const adapter of adapters) {
      const id = adapter?.id || adapter?.name;
      if (!id) continue;
      const opts = await callWails(() => ListAdapterModelOptions(id)).catch(() => null);
      if (Array.isArray(opts) && opts.length > 0) out[id] = opts;
    }
    setAdapterModelOptions(out);
    await reconcileAdapterModelChoicesAgainstOptions(choices, out);
    return out;
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
        await refreshAdapterModelOptionsForAdapters(adapterList, choices || {});
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
      const nextOptions = {...adapterModelOptions, [adapterID]: opts};
      setAdapterModelOptions(nextOptions);
      await reconcileAdapterModelChoicesAgainstOptions(adapterModelChoicesRef.current, nextOptions);
      refreshAvailableAdapters().catch(() => {});
      return opts;
    }
    setAdapterModelOptions((prev) => {
      const next = {...prev};
      delete next[adapterID];
      return next;
    });
    refreshAvailableAdapters().catch(() => {});
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
        const localizedStatusRail = localizeStatusRailView(hydrated.statusRail) || fallbackState.statusRail;
        setState({
          ...hydrated,
          greeting: localizeStatusRailText(hydrated.greeting || localizedStatusRail?.text || fallbackState.greeting),
          statusRail: localizedStatusRail,
        });
        loadedConversationIdsRef.current.add('main');
        setConversationMessages('main', hydrated.messages);
      })
      .catch(() => {});
    callWails(GetSettingsState)
      .then((next) => {
        const normalized = normalizeSettingsState(next);
        setSettingsState(normalized);
        const startupLocale = panelLangToLocale(normalized.panel?.panelLanguage);
        if (startupLocale && startupLocale !== useI18n.getState().language) {
          useI18n.getState().setLanguage(startupLocale);
        }
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
    const offAdapterStatus = EventsOn('adapter:status_changed', (payload) => {
      setAdapterList((prev) => applyAdapterStatusUpdate(prev, payload));
      refreshAvailableAdapters().catch(() => {});
    });
    const offAdapterModelCleared = EventsOn('adapter:model_choice_cleared', (payload) => {
      const adapterID = payload?.adapter_id || '';
      const clearedModel = payload?.cleared_model || '';
      const fallbackModel = payload?.fallback_model || '';
      if (adapterID) {
        setAdapterModelChoices((prev) => {
          const next = {...prev};
          delete next[adapterID];
          adapterModelChoicesRef.current = next;
          return next;
        });
      }
      if (clearedModel) {
        setAdapterModelRepairToast(buildAdapterModelRepairMessage({
          adapterID,
          model: clearedModel,
          fallbackModel,
          reason: payload?.reason || '',
        }));
      }
      if (adapterID) {
        handleAdapterModelRefresh(adapterID).catch(() => {});
      } else {
        refreshAvailableAdapters().catch(() => {});
      }
    });
    // #44 修補：skill / MCP 安裝後後端會 emit，前端重新拉取工具列。
    const offToolsChanged = EventsOn('tools:list_changed', () => {
      callWails(ListTools)
        .then((nextTools) => setTools(normalizeToolList(nextTools)))
        .catch(() => {});
    });

    // GGUF 匯入（引用連結貼 .gguf → 下載 → ollama create）進度 / 完成 / 失敗
    const offGGUFProgress = EventsOn('gguf:import_progress', (payload) => {
      const phase = payload?.phase === 'download' ? 'GGUF 下載中' : 'GGUF 建立模型中';
      const pct = typeof payload?.percent === 'number' ? ` ${payload.percent}%` : '';
      setToolResult({toolId: 'reference-link', ok: true, message: `${phase}${pct}：${payload?.model || ''}`});
    });
    const offGGUFDone = EventsOn('gguf:import_done', (payload) => {
      const note = payload?.note ? `（${payload.note}）` : '';
      setToolResult({toolId: 'reference-link', ok: true, message: `GGUF 模型已建立：${payload?.model || ''}${note}`});
      refreshAvailableAdapters().catch(() => {});
    });
    const offGGUFFailed = EventsOn('gguf:import_failed', (payload) => {
      setToolResult({toolId: 'reference-link', ok: false, message: `GGUF 匯入失敗：${payload?.error || ''}`});
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
    const offTaskCompleted = EventsOn('task:progress_completed', (payload) => {
      syncTaskEvent(payload);
      setTaskLoopRounds({});
    });
    const offTaskFailed = EventsOn('task:progress_failed', (payload) => {
      syncTaskEvent(payload);
      setTaskLoopRounds({});
    });
    const offTaskCancelled = EventsOn('task:progress_cancelled', (payload) => {
      syncTaskEvent(payload);
      setTaskLoopRounds({});
    });
    // v3.1.6：節點內 loop 每輪進度
    const offTaskLoopRound = EventsOn('task:loop_round', (payload) => {
      const nodeId = payload?.node_id;
      if (!nodeId) return;
      setTaskLoopRounds((prev) => ({...prev, [nodeId]: {
        iteration: payload?.iteration || 0,
        action: payload?.action || '',
        target: payload?.target || '',
        message: payload?.message || '',
      }}));
    });
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
      setWa3ToastMsg(t('system.imported', { name: data?.display_name || t('wa3.defaultDocName') }));
      setTimeout(() => setWa3ToastMsg(null), 4000);
    });

    // skill 產出落位完成 → 立即刷新右側引用面板（5 秒輪詢之外的即時路徑）。
    const offReferenceImported = EventsOn('reference:imported', (data) => {
      refreshReferenceFiles().catch(() => {});
      if (data?.name) {
        setWa3ToastMsg(t('system.imported', { name: data.name }));
        setTimeout(() => setWa3ToastMsg(null), 4000);
      }
    });

    // 上方互動輸出：statusRail 事件只同步 greeting，不寫入下方 messages。
    const offStatusRail = EventsOn('statusrail:updated', (payload) => {
      if (!payload) return;
      unlockManualGreeting();
      const localizedPayload = localizeStatusRailView(payload);
      setState((prev) => ({
        ...prev,
        greeting: localizedPayload.text || prev.greeting,
        statusRail: localizedPayload,
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
    const offDigestUpdated = EventsOn('digest:updated', () => {
      loadReviewPanelData();
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

    const offCodeArtifactActivity = EventsOn('code_artifact:activity', (payload) => {
      const entry = normalizeCodeArtifactActivity(payload);
      if (!entry?.traceId) return;
      const currentEntries = codeArtifactActivityLogsRef.current[entry.traceId] || [];
      const nextEntries = [...currentEntries, entry].slice(-40);
      codeArtifactActivityLogsRef.current[entry.traceId] = nextEntries;
      const conversationId = activeConversationIdRef.current || 'main';
      const progressText = formatCodeArtifactActivityProgress(nextEntries);
      setConversationMessages(conversationId, (prev) => updateComposerPendingMessage(
        prev,
        entry.traceId,
        makeComposerPendingMessage(entry.traceId, progressText),
      ));
    });

    const offSchedulerRequested = EventsOn('scheduler:action_requested', (payload) => {
      const rawText = String(payload?.raw || '').trim();
      const fallbackText = [payload?.action, payload?.target, payload?.next].filter(Boolean).join(' ');
      const text = [rawText, fallbackText].filter(Boolean).join(' ');
      const intent = parseSchedulerConversationIntent(rawText || text, schedulerJobs)
        || parseSchedulerConversationIntent(text, schedulerJobs);
      const sourceText = rawText || text;
      const modelTitle = String(payload?.title || '').trim();
      const modelSummary = String(payload?.summary || '').trim();
      const fallbackName = modelTitle || cleanSchedulerNameText(sourceText) || payload?.action || '新的排程任務';
      const fallbackAction = parseSchedulerActionText(sourceText) || fallbackName;
      const draft = intent?.type === 'create'
        ? {...intent.draft, name: modelTitle || intent.draft.name, summary: modelSummary || intent.draft.summary}
        : {
            name: fallbackName,
            cronExpr: hasSchedulerTimeText(text) ? parseSchedulerTimeText(text, '') : '',
            actionText: fallbackAction,
            summary: modelSummary,
            actionPayload: schedulerDefaultPayload(fallbackName, fallbackAction, modelSummary),
          };
      const normalized = normalizeSchedulerDraft(draft);
      const missing = schedulerMissingSlots(normalized);
      setSchedulerConversation({mode: 'create', phase: missing.length ? 'collecting' : 'confirm', draft: normalized, missing, job: null});
      setSchedulerDraft({
        name: normalized.name,
        cronExpr: normalized.cronExpr || '@daily',
        actionPayload: schedulerDefaultPayload(normalized.name || '提醒', normalized.actionText || normalized.name, normalized.summary),
      });
      Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()]).catch(() => {});
      setToolResult({toolId: 'scheduler', ok: true, message: missing.length ? '排程資訊待補齊' : '排程資訊等待確認'});
    });

    // Phase D：排程到點時後端 emit 'scheduler:reminder'，這裡在 app 內顯示提醒，
    // 讓使用者知道系統要開始做事（remote_bridge 那條由後端 NotifyJobFired 另外送）。
    const offSchedulerReminder = EventsOn('scheduler:reminder', (payload) => {
      const title = String(payload?.title || '').trim() || t('chatSystem.scheduler.defaultTask');
      const summary = String(payload?.summary || payload?.action || '').trim();
      const notice = `Ai:${t(summary ? 'chatSystem.scheduler.reminderWithSummary' : 'chatSystem.scheduler.reminder', {title, summary})}`;
      const conversationId = activeConversationIdRef.current || appSessionId || 'main';
      setConversationMessages(conversationId, (prev) => [...prev, notice]);
      setToolResult({toolId: 'scheduler', ok: true, message: `排程提醒：${title}`});
      postDebugTrace('ui.scheduler.reminder', '', {title, summary});
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

    // ── v3.6.2 WA3 Media Provenance（§9A）事件監聯 ──
    const offWA3Verified = EventsOn('wa3:verified', (data) => {
      // 驗證完成 → 同步更新目前檢視中媒體的狀態（若路徑相符）
      if (data?.path) {
        setWa3Detail((current) => (
          current && current.file_path === data.path
            ? {...current, status: data.status}
            : current
        ));
      }
    });
    const offWA3Pollution = EventsOn('wa3:pollution_detected', (data) => {
      if (data) setWa3ToastMsg(t('wa3.pollutionWarning'));
      setTimeout(() => setWa3ToastMsg(null), 6000);
    });
    const offWA3Exported = EventsOn('wa3:exported', () => {
      setWa3ToastMsg(t('wa3.transmitHint'));
      setTimeout(() => setWa3ToastMsg(null), 5000);
    });
    const offWA3Imported = EventsOn('wa3:imported', (data) => {
      // 由 importMediaWA3 函式處理彈窗
    });
    const offWA3DocCreated = EventsOn('wa3:doc_created', () => {
      setWa3ToastMsg(t('wa3.docCreated'));
      setTimeout(() => setWa3ToastMsg(null), 4000);
    });
    const offWA3Trust = EventsOn('wa3:trust_updated', () => refreshWA3TrustList());
    // 啟動時載入一次信任清單，避免匯入選單首次顯示信任數為 0
    refreshWA3TrustList();

    // §30: 關閉視窗 → 前端收到後端 session:close_prompt 事件，顯示對話框
    const offSessionClose = EventsOn('session:close_prompt', (payload) => {
      if (payload?.analysis) {
        setSessionClosePrompt(payload.analysis);
      }
    });
    // Phase G：關閉前後端 emit 此事件 → 顯示後台頭像選擇。
    const offSchedulerBgPrompt = EventsOn('scheduler:background_prompt', async () => {
      if (floatingAvatarCompactWindowRef.current || floatingAvatarWindowRef.current?.restore) {
        await restoreFloatingAvatarWindow();
        floatingAvatarModeRef.current = false;
        setFloatingAvatarMode(false);
        setManualAvatarState('');
      }
      setSchedulerBgPrompt(true);
    });
    // H/I：高風險/付費 API 需確認 → 後端 emit，前端跳確認卡。
    const offSchedulerConfirm = EventsOn('scheduler:confirm_needed', (payload) => {
      if (payload?.job_id) setSchedulerConfirm(payload);
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
      callWails(RequestVisualLearningPermissions)
        .then(async (status) => {
          const opened = await openFirstMissingVisualLearningPermission(status);
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            formatVisualLearningPermissionStatus(status, true) + (opened ? `\n${t('chatSystem.learning.permissionSettingsOpened')}` : ''),
          ]);
        })
        .catch((error) => {
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            t('chatSystem.learning.permissionRequestFailed', {error: error?.message || String(error || '')}),
          ]);
        });
    });
    const offLearningTextConfirm = EventsOn('visual_learning:text_confirmation_requested', (payload) => {
      const conversationId = activeConversationIdRef.current || 'main';
      const id = String(payload?.id || '');
      if (!id) return;
      const source = payload?.source || 'unknown';
      const target = payload?.window_title || payload?.label || 'unknown target';
      const preview = payload?.preview ? `\n\nPreview: ${payload.preview}` : '';
      const allowed = window.confirm(`是否把這段 ${source} 文字輸入保存到示範？\n目標：${target}\n長度：${payload?.length || 0}${preview}`);
      callWails(() => ConfirmLearningTextEvent(id, allowed ? 'save' : 'redact'))
        .then(() => {
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            t(allowed ? 'chatSystem.learning.textSaved' : 'chatSystem.learning.textRedacted'),
          ]);
        })
        .catch((error) => {
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            t('chatSystem.learning.textConfirmFailed', {error: error?.message || String(error || '')}),
          ]);
        });
    });

    return () => {
      offAdapterChanged();
      offAdapterStatus();
      offAdapterModelCleared();
      offToolsChanged();
      offGGUFProgress();
      offGGUFDone();
      offGGUFFailed();
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
      offTaskLoopRound();
      offTaskSystemMessage();
      offReviewAdded();
      offReviewResolved();
      offDocImported();
      offReferenceImported();
      offStatusRail();
      offDegradedEnter();
      offDegradedExit();
      offMemory();
      offExecInterrupted();
      offDigestUpdated();
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
      offCodeArtifactActivity();
      offSchedulerRequested();
      offSchedulerReminder();
      offWA3Verified();
      offWA3Pollution();
      offWA3Exported();
      offWA3Imported();
      offWA3DocCreated();
      offWA3Trust();
      offSessionClose();
      offSessionCloseConfirmed();
      offSchedulerBgPrompt();
      offSchedulerConfirm();
      offLearningTextConfirm();
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

  function normalizeCodeArtifactActivity(payload) {
    const traceId = String(payload?.trace_id || payload?.traceId || '').trim();
    if (!traceId) return null;
    const status = String(payload?.status || 'running').trim() || 'running';
    const title = String(payload?.title || payload?.phase || '資料區程式處理中').trim();
    const detail = String(payload?.detail || '').trim();
    const fileName = String(payload?.file_name || payload?.fileName || '').trim();
    const attempt = Number(payload?.attempt || 0);
    const at = String(payload?.at || new Date().toISOString()).trim();
    return {traceId, status, title, detail, fileName, attempt, at};
  }

  function formatCodeArtifactActivityLine(entry, index) {
    const prefix = entry.status === 'success' ? '完成' : entry.status === 'failed' ? '失敗' : entry.status === 'warning' ? '注意' : '進行中';
    const attemptText = entry.attempt > 0 ? `（第 ${entry.attempt} 輪）` : '';
    const detailText = entry.detail ? `：${entry.detail}` : '';
    return `${index + 1}. ${prefix} ${entry.title}${attemptText}${detailText}`;
  }

  function formatCodeArtifactActivityProgress(entries) {
    const visibleEntries = entries.slice(-5);
    const latest = visibleEntries[visibleEntries.length - 1];
    const title = latest?.title || '資料區程式處理中';
    return [
      `資料區程式正在處理：${title}`,
      ...visibleEntries.map((entry, index) => formatCodeArtifactActivityLine(entry, Math.max(0, entries.length - visibleEntries.length) + index)),
    ].join('\n');
  }

  function formatCodeArtifactActivityTranscript(entries) {
    if (!entries.length) return '';
    return [
      '',
      '',
      '處理過程（完成後可展開檢查）：',
      ...entries.map((entry, index) => formatCodeArtifactActivityLine(entry, index)),
    ].join('\n');
  }

  function appendCodeArtifactActivityTranscript(traceId, text) {
    const entries = codeArtifactActivityLogsRef.current?.[traceId] || [];
    if (!entries.length) return text;
    delete codeArtifactActivityLogsRef.current[traceId];
    return `${text}${formatCodeArtifactActivityTranscript(entries)}`;
  }

  async function finishComposerExecution({resp, payload, apiAdapter, traceId, conversationId, clearPendingTimers}) {
    // 使用者已按停止鈕中斷此 trace：丟棄遲到的結果，不覆蓋「已中斷」訊息。
    if (cancelledChatTracesRef.current.has(traceId)) {
      cancelledChatTracesRef.current.delete(traceId);
      delete codeArtifactActivityLogsRef.current[traceId];
      clearPendingTimers?.();
      return;
    }
    clearPendingTimers?.();
    const cliResp = await applyComposerBuiltInSideEffects(normalizeCLIResponse(resp));
    postDebugTrace(apiAdapter ? 'ui.composer.after.SendAPIMessage' : 'ui.composer.after.SendCLIMessage', traceId, {response: cliResp || null});
    const cliAction = String(cliResp?.action || '').trim();
    const cliNext = String(cliResp?.next || '').trim();
    setDagClarificationPending(cliAction === '提問' || cliNext === '提問');
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
        const errorMessage = `Ai:${t('chatSystem.learning.replayDirectiveReadFailed', {error: err?.message || String(err)})}`;
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
    if (!cliResp?.auth_required && !cliResp?.error && cliAction === '任務') {
      const taskText = String(cliResp?.target || payload?.userText || '').trim();
      if (taskText) {
        setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(
          prev,
          traceId,
          makeComposerPendingMessage(traceId, '任務規劃中，請稍等。'),
        ));
        postDebugTrace('ui.composer.dag_intent_confirmed', traceId, {user_text: taskText});
        setDagClarificationPending(false);
        startDagForMessage(taskText, {conversationId, traceId});
        return;
      }
    }
    if (cliResp?.auth_required) {
      const messageText = appendCodeArtifactActivityTranscript(traceId, cliResp.text || t('system.authRequired'));
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, `Ai:${messageText}`));
    } else if (cliResp?.text) {
      const messageText = appendCodeArtifactActivityTranscript(traceId, cliResp.text);
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, `Ai:${messageText}`));
      persistConversationEntry(conversationId, 'assistant', messageText, traceId).catch(() => {});
    } else if (cliResp?.error) {
      const messageText = appendCodeArtifactActivityTranscript(traceId, cliResp.error);
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, `[${t('system.sysLabel')}] ${messageText}`));
    }
  }

  async function executeLearningOperationIntent(operationIntent, {conversationId, traceId}) {
    try {
      if (operationIntent.mode === 'execute' && isLastLearningOperationReference(operationIntent)) {
        postDebugTrace('ui.composer.learning_operation_last_replay', traceId, {
          query: operationIntent.query,
          raw: operationIntent.raw || '',
        });
        const plan = await callWails(GetLastLearningReplayPlan);
        const message = formatLearningReplayPlan(plan);
        setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
        persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, ''), traceId).catch(() => {});
        await executeLearningReplayWithChat(plan, conversationId, traceId);
        return;
      }
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
      const message = `Ai:${t('chatSystem.learning.operationLookupFailed', {query: operationIntent.query, error: err?.message || String(err)})}`;
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
    }
  }

  async function enrichStoppedLearningRun(run, traceId) {
    const runId = run?.id || run?.run_id || '';
    if (!runId) return run;
    const stepCount = Number(run?.step_count ?? run?.StepCount ?? 0);
    if (stepCount <= 0) {
      postDebugTrace('ui.learning_metadata.generate.skip_empty', traceId, {
        run_id: runId,
        reason: 'no_recorded_steps',
      });
      return run;
    }
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
    // 使用者已按停止鈕中斷此 trace：吞掉被 kill 的子程序錯誤，不覆蓋「已中斷」訊息。
    if (cancelledChatTracesRef.current.has(traceId)) {
      cancelledChatTracesRef.current.delete(traceId);
      delete codeArtifactActivityLogsRef.current[traceId];
      clearPendingTimers?.();
      return;
    }
    clearPendingTimers?.();
    postDebugTrace(apiAdapter ? 'ui.composer.SendAPIMessage.error' : 'ui.composer.SendCLIMessage.error', traceId, {error: err?.message || String(err)});
    const rawErrorMsg = sanitizeDisplayedCLIError(err?.message || String(err));
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
    const messageText = appendCodeArtifactActivityTranscript(traceId, t('system.sendFail', { error: errorMsg }));
    setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, messageText));
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
        const permissionStatus = await callWails(RequestVisualLearningPermissions).catch(() => null);
        if (permissionStatus?.missing?.length) {
          const opened = await openFirstMissingVisualLearningPermission(permissionStatus);
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            formatVisualLearningPermissionStatus(permissionStatus, true) + (opened ? `\n${t('chatSystem.learning.permissionSettingsOpened')}` : ''),
          ]);
        }
        const run = await callWails(() => StartLearningMode(windowHash));
        setVlActiveLearningRun(run);
        setVlLearningActive(true);
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          t('chatSystem.learning.recordingStartedProcess', {run: run?.id ? `Run: ${run.id}. ` : ''}),
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
            t('chatSystem.learning.recordingResumedStop', {run: activeRun?.id ? `Run: ${activeRun.id}. ` : ''}),
          ]);
        } else {
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            t('chatSystem.learning.recordingBackendFailed', {error: detail}),
          ]);
        }
      }
      setRecordingEnabled(true);
      setSandboxStopOptions(null);
    } catch (error) {
      const detail = error?.message || String(error || '');
      setConversationMessages(conversationId, (prev) => [
        ...prev,
        t('chatSystem.learning.recordingStartFailed', {error: detail}),
      ]);
    }
  }

  async function stopDraftSandbox(reason) {
    if (!activeSandboxId) return;
    const conversationId = activeConversationIdRef.current || 'main';
    try {
      const backendActive = await callWails(IsLearningModeActive).catch(() => vlLearningActive);
      if (vlLearningActive || backendActive) {
        try {
          if (backendActive) {
            const traceId = makeDebugTraceID('learning-metadata');
            const run = await callWails(StopLearningMode);
            const namedRun = await enrichStoppedLearningRun(run, traceId);
            setConversationMessages(conversationId, (prev) => [
              ...prev,
              formatLearningOperationLearned(namedRun),
            ]);
          }
        } catch (error) {
          const detail = error?.message || String(error || '');
          if (!isNoActiveLearningRecordingError(detail)) {
            setConversationMessages(conversationId, (prev) => [
              ...prev,
              t('chatSystem.learning.recordingStopFailed', {error: detail}),
            ]);
          }
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
          `Ai:${t('chatSystem.learning.replayReadFailed', {error: error?.message || String(error)})}`,
        ]);
      }
      setSandboxStopOptions(null);
      setActiveSandboxId(null);
      if (plan) {
        await executeLearningReplayWithChat(plan, conversationId);
      }
      return;
    } else if (promotion === 'pending_candidate') {
      setConversationMessages(conversationId, (prev) => [...prev, t('chatSystem.learning.candidateSaved')]);
    }
    setSandboxStopOptions(null);
    setActiveSandboxId(null);
  }

  function dismissSandboxOptions() {
    const conversationId = activeConversationIdRef.current || 'main';
    setConversationMessages(conversationId, (prev) => [...prev, t('chatSystem.learning.demoDiscarded')]);
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
    const cancelMessage = `Ai:${t('chatSystem.learning.replayCancelled')}`;
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
    callWails(() => HideNativeLearningReplayCursor()).catch(() => {});
    learningReplayExecutingRef.current = true;
    try {
      const confirmedStep = {
        ...originalStep,
        x: Number.isFinite(Number(previewResult.x)) ? Number(previewResult.x) : originalStep.x,
        y: Number.isFinite(Number(previewResult.y)) ? Number(previewResult.y) : originalStep.y,
        replay_confirmed: true,
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
    callWails(() => HideNativeLearningReplayCursor()).catch(() => {});
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
          const localizedStatusRail = localizeStatusRailView(statusRail);
          // 上方互動輸出快照：輪詢只刷新 greeting/statusRail。
          setState((prev) => ({
            ...prev,
            greeting: manualGreetingLockedRef.current ? prev.greeting : (localizedStatusRail.text || prev.greeting),
            statusRail: manualGreetingLockedRef.current ? {...localizedStatusRail, text: prev.greeting} : localizedStatusRail,
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
        refreshAdapterModelOptionsForAdapters(adapterList).catch(() => {});
        refreshAvailableAdapters().catch(() => {});
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
  }, [adapterList]);

  useEffect(() => {
    if (!learningEnabled) return undefined;

    let idleTimer = window.setTimeout(() => markLearningDigestReady('idle_20m'), learningIdleDelayMs);
    const resetIdleTimer = () => {
      window.clearTimeout(idleTimer);
      idleTimer = window.setTimeout(() => markLearningDigestReady('idle_20m'), learningIdleDelayMs);
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
    learningDigestReadyRef.current = learningDigestReady;
  }, [learningDigestReady]);

  useEffect(() => {
    try {
      OnFileDrop((x, y, paths) => {
        if (referenceInternalDragRef.current || Date.now() < referenceDropSuppressUntilRef.current) return;
        const nativePaths = normalizeReferenceImportPaths(paths);
        if (!nativePaths.length) return;
        // 先分類再分流：若是可辨識的安裝包（subagent / skill / go-program 匯出），
        // 走安裝流程且「不」複製進引用文件；否則才當一般引用文件匯入。
        (async () => {
          if (shouldProbeDroppedInstallPackage(nativePaths)) {
            const recognized = await detectDroppedInstallPackage(nativePaths[0]);
            if (recognized) return;
          }
          importReferencePaths(nativePaths);
        })();
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

  // 回傳 true 表示已辨識為某種安裝包（已設定 installCandidate）；false 表示不是安裝包。
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
        return true;
      }
    } catch {
      // 不是 sub 包時繼續試其他安裝類型。
    }
    try {
      const preview = await callWails(() => PreviewGoProgramExport(path));
      if (preview?.export_type === 'go_program_authoring') {
        setInstallCandidate({
          type: 'goprogram',
          typeLabel: 'go-program',
          name: preview.program_name || preview.program_id || t('package.unnamedItem'),
          path,
          preview,
        });
        return true;
      }
    } catch {
      // 不是 go-program 匯出時繼續試 skill。
    }
    try {
      const preview = await callWails(() => PreviewLearningRunExport(path));
      if (preview?.export_type === 'learning_run') {
        setInstallCandidate({
          type: 'learningrun',
          typeLabel: '錄製',
          name: preview.title || preview.tag || preview.run_id || t('package.unnamedItem'),
          path,
          preview,
        });
        return true;
      }
    } catch {
      // 不是錄製匯出時繼續試 skill。
    }
    try {
      const name = basenameForDisplay(path) || t('package.unnamedItem');
      const manifestJSON = buildPersonaPackageManifest(name, path);
      const pending = await callWails(() => PreparePackageInstall(path, manifestJSON));
      if (pending?.persona) {
        setPendingImport(pending);
        setPendingPackageData(pending.persona);
        setPackageInstallError('');
        return true;
      }
    } catch {
      // 不是 persona 匯出時繼續試 skill。
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
        return true;
      }
      setToolResult({toolId: 'package-detect', ok: false, message: t('package.unrecognized')});
      return false;
    } catch {
      setToolResult({toolId: 'package-detect', ok: false, message: t('package.unrecognized')});
      return false;
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
        // v3.1.8：索引快取驗證發現竄改/損毀 → 明確提醒使用者（報告已回寫進該 sub）。
        const integ = result?.memory_integrity;
        if (integ && (integ.dropped > 0 || integ.terms_dropped > 0)) {
          const parts = [];
          if (integ.dropped > 0) parts.push(`${integ.dropped} 筆索引對不上記憶內容，已丟棄`);
          if (integ.terms_dropped > 0) parts.push(`${integ.terms_dropped} 個關鍵詞接地失敗，已剔除`);
          setToolResult({
            toolId: 'subagent',
            ok: false,
            message: `匯入的 sub 索引快取有問題（疑似篡改或損毀）：${parts.join('；')}。` +
              `詳細報告已寫入該 sub。建議切到此 sub 用「整理」重新建立摘要與索引。`,
          });
        }
        await refreshAvailableAdapters();
        return;
      }
      if (installCandidate.type === 'goprogram') {
        const item = await callWails(() => ImportGoProgramExport(installCandidate.path));
        setInstallCandidate(null);
        setToolResult({
          toolId: 'package-install',
          ok: true,
          message: `已安裝小程式：${item?.program_name || item?.program_id || installCandidate.name}（工具 > 自動流程）`,
        });
        return;
      }
      if (installCandidate.type === 'learningrun') {
        const item = await callWails(() => ImportLearningRunExport(installCandidate.path));
        setInstallCandidate(null);
        setToolResult({
          toolId: 'package-install',
          ok: true,
          message: `已安裝錄製：${item?.title || item?.tag || item?.run_id || installCandidate.name}`,
        });
        return;
      }
      if (installCandidate.type === 'skill') {
        const previewId = installCandidate.preview?.PreviewID || installCandidate.preview?.preview_id;
        if (!previewId) return;
        await callWails(() => ConfirmSkillArchive(previewId));
        setInstallCandidate(null);
        const skills = await callWails(ListArchivedSkills);
        setArchivedSkills(skills || []);
        // 安裝完成後不等後端事件，前端直接重拉工具列，讓 ✦ skill 立即出現
        // （tools:list_changed 事件仍保留，這裡是雙保險）。
        callWails(ListTools)
          .then((nextTools) => setTools(normalizeToolList(nextTools)))
          .catch(() => {});
        setToolResult({
          toolId: 'package-install',
          ok: true,
          message: `已安裝 skill：${installCandidate.name}（工具 > 自動流程）`,
        });
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

  // ── Voice Sync R1：session 控制（spec §34.1 狀態機的前端簡化版）──
  async function startVoiceSyncSession() {
    if (voiceRecording || voiceBusy) return;
    // barge-in：使用者開口，AI 先閉嘴（spec §34.6 spoken output 不得疊使用者語音）。
    callWails(StopVoiceOutput).catch(() => {});
    voiceSyncRef.current = createVoiceSyncSession();
    setVoiceSyncActive(true);
    setVoiceSyncPhase('listening');
    await startVoiceRecording();
    if (!voiceRecorderRef.current) {
      voiceSyncRef.current = {...voiceSyncRef.current, active: false, phase: 'idle'};
      setVoiceSyncActive(false);
      setVoiceSyncPhase('idle');
      return;
    }
    setVoiceStatus(t('voice.syncListening'));
  }

  async function abortVoiceSyncSession() {
    voiceSyncRef.current = {...voiceSyncRef.current, active: false, phase: 'idle'};
    setVoiceSyncActive(false);
    setVoiceSyncPhase('idle');
    await cancelVoiceRecording();
  }

  function commitVoiceSyncTurn() {
    const sync = voiceSyncRef.current;
    if (!sync.active || sync.committing) return;
    voiceSyncRef.current = {...sync, active: false, committing: true, phase: 'committing'};
    setVoiceSyncPhase('committing');
    Promise.resolve(stopVoiceRecording()).finally(() => {
      setVoiceSyncActive(false);
      setVoiceSyncPhase('idle');
    });
  }

  function handleVoicePressStart() {
    if (voiceChatEnabledRef.current) {
      if (voiceSyncRef.current.active) {
        abortVoiceSyncSession();
      } else {
        startVoiceSyncSession();
      }
      return;
    }
    startVoiceRecording();
  }

  function handleVoicePressEnd() {
    if (voiceChatEnabledRef.current) return; // sync 模式：放開不動作，靜音才 commit
    stopVoiceRecording();
  }

  function handleVoicePointerCancel() {
    if (voiceChatEnabledRef.current) return; // sync 模式：pointercancel 不中斷 session
    cancelVoiceRecording();
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
        const data = event.inputBuffer.getChannelData(0);
        chunks.push(new Float32Array(data));
        const sync = voiceSyncRef.current;
        if (!sync.active) return;
        // 能量偵測（RMS，取樣間隔 8 降低成本）——spec §34.2 自寫、零依賴。
        let sum = 0;
        let count = 0;
        for (let i = 0; i < data.length; i += 8) { sum += data[i] * data[i]; count += 1; }
        const rms = Math.sqrt(sum / Math.max(count, 1));
        const now = Date.now();
        const blockMs = (data.length / audioContext.sampleRate) * 1000;
        const transition = advanceVoiceSyncSession(sync, {rms, blockMs, now});
        voiceSyncRef.current = transition.session;
        if (transition.action === 'commit') {
          commitVoiceSyncTurn();
          return;
        }
        if (transition.action === 'abort') {
          abortVoiceSyncSession();
          return;
        }
        if (transition.phaseChanged) {
          setVoiceSyncPhase(transition.session.phase);
          if (transition.session.phase === 'short_pause') {
            setVoiceStatus(t('voice.syncThinkingPause'));
          } else {
            setVoiceStatus(t('voice.syncListening'));
          }
        }
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
    // 120 秒自動停止等外部路徑也要收掉 sync session，避免狀態卡在 listening。
    if (voiceSyncRef.current.active) {
      voiceSyncRef.current = {...voiceSyncRef.current, active: false, phase: 'idle'};
      setVoiceSyncActive(false);
      setVoiceSyncPhase('idle');
    }
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

  function cleanTextForSpeech(text) {
    return String(text || '')
      .replace(/^Ai:/, '')
      .replace(/```[\s\S]*?```/g, ' ')
      .replace(/`[^`]*`/g, ' ')
      .replace(/https?:\/\/\S+/g, ' ')
      .replace(/[#*_>|]{1,}/g, ' ')
      .replace(/\s+/g, ' ')
      .trim()
      .slice(0, 1200);
  }

  function speakAssistantReply(text) {
    const clean = cleanTextForSpeech(text);
    if (!clean) return;
    callWails(() => SpeakVoiceLine(clean, 'readout')).catch(() => {});
  }

  async function toggleVoiceChatMode(next) {
    const target = typeof next === 'boolean' ? next : !voiceChatEnabledRef.current;
    setVoiceChatHint(false);
    try {
      const state = await callWails(() => SetVoiceOutputEnabled(target));
      setVoiceState(state || null);
      if (!target) callWails(StopVoiceOutput).catch(() => {});
    } catch (err) {
      setVoiceError(err?.message || String(err));
    }
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
    if (!voiceChatEnabledRef.current) setVoiceChatHint(true);
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
    const next = pickRotatingGreeting(state.greeting, getPersonaPatrolDialogueOptions(findActivePersona(settingsState)));
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
          `Ai:${t('chatSystem.taskStarted', {title: run?.title || t('chatSystem.taskProgress')})}`,
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

  // 停止鈕統一入口：優先中斷正在跑的一般對話／skill（非 DAG），否則回到 DAG 任務取消。
  async function cancelActiveExecution(reason = 'user_stop') {
    callWails(StopVoiceOutput).catch(() => {});
    if (voiceSyncRef.current.active || voiceRecorderRef.current) {
      await abortVoiceSyncSession();
      setVoiceStatus(t('voice.stopRecording'));
      return;
    }
    const chatTrace = activeChatTraceRef.current;
    if (chatTrace) {
      cancelledChatTracesRef.current.add(chatTrace);
      activeChatTraceRef.current = null;
      setActiveChatTrace(null);
      const conversationId = activeConversationIdRef.current || 'main';
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, chatTrace, `Ai:${t('chatSystem.replyInterrupted')}`));
      try {
        await callWails(() => CancelChatMessage(chatTrace));
      } catch (error) {
        // best-effort：子程序可能已自行結束。
      }
      return;
    }
    return cancelActiveTaskProgress(reason);
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
      error: sanitizeDisplayedCLIError(resp.error ?? resp.Error ?? ''),
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
    // 語音聊天模式開啟時，AI 回覆同步進入朗讀佇列（voice_id 於建立當下快照）。
    if (role === 'assistant' && voiceChatEnabledRef.current) {
      speakAssistantReply(text);
    }
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
      postDebugTrace('ui.inspector.after.SendTopInteractionMessage', traceId, {response: cliResp || null});
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

  async function sendMessage(event, images = []) {
    event.preventDefault();
    const selectedText = selectedFloatingCandidateIDs
      .map((candidateID) => (readinessGate.floating_candidates || []).find((candidate) => candidate.id === candidateID))
      .map(candidateReplyText)
      .filter(Boolean)
      .join('\n');
    const typedText = String(draft || '').trim();
    submitComposerText([selectedText, typedText].filter(Boolean).join('\n'), images);
  }

  function acceptDagTaskSuggestion() {
    const text = String(draft || '').trim();
    if (!text) return;
    setDismissedDagSuggestionText('');
    submitComposerText(text, [], {forceDag: true});
  }

  function dismissDagTaskSuggestion() {
    setDismissedDagSuggestionText(String(draft || '').trim());
  }

  function summarizeSearchResultText(text) {
    const sourceText = String(text || '').trim();
    if (!sourceText) return;
    submitComposerText(
      t('system.summarizeSearch', { text: sourceText }),
      [],
      {displayText: t('system.summarizeSearchDisplay')},
    );
  }

  async function submitComposerText(rawText, images = [], options = {}) {
    const text = stripInternalControlPrefix(rawText);
    if (!text) return;
    const displayText = String(options.displayText || '').trim() || text;
    unlockManualGreeting();
    const conversationId = activeConversationIdRef.current || 'main';
    const traceId = makeDebugTraceID('chat');
    const sessionId = appSessionId || '';
    const pendingMessage = makeComposerPendingMessage(traceId);
    postDebugTrace('ui.composer.submit.raw', traceId, {user_text: text});
    setConversationMessages(conversationId, (prev) => [...prev, displayText, pendingMessage]);
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
      // 此回合對話送出已收尾 → 釋放「執行中」狀態，停止鈕回到 disabled。
      if (activeChatTraceRef.current === traceId) {
        activeChatTraceRef.current = null;
        setActiveChatTrace(null);
      }
    };
    setDraft('');
    setSelectedFloatingCandidateIDs([]);
    try {
      await persistConversationEntry(conversationId, 'user', displayText, traceId);
    } catch (err) {
      setConversationMessages(conversationId, (prev) => [...prev, t('system.conversationSaveFail', { error: err?.message || err })]);
    }
    if (await handleSchedulerComposerIntent(text, conversationId, traceId, clearPendingTimers)) {
      return;
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
            `Ai:${t('chatSystem.learning.replayReadFailed', {error: err?.message || String(err)})}`
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
        const message = `Ai:${t('chatSystem.learning.catalogReadFailed', {error: err?.message || String(err)})}`;
        setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
      }
      return;
    }
    if (!dagClarificationPending && !options.forceDag && shouldSuggestDagRun(text)) {
      try {
        const clarifyResp = await callWails(() => window.go?.main?.App?.ClarifyDagIntent?.(sessionId, text, traceId));
        const cliResp = normalizeCLIResponse(clarifyResp);
        if (cliResp?.action === '提問') {
          clearPendingTimers();
          setDagClarificationPending(true);
          setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(
            prev,
            traceId,
            `Ai:${cliResp.text || t('chatSystem.taskClarification')}`,
          ));
          persistConversationEntry(conversationId, 'assistant', cliResp.text || t('chatSystem.taskClarification'), traceId).catch(() => {});
          refreshReadinessGateState();
          scheduleReadinessGateBurstRefresh();
          return;
        }
      } catch (error) {
        postDebugTrace('ui.composer.dag_intent_clarify.error', traceId, {error: error?.message || String(error)});
      }
    }
    const explicitDagRun = /^\/dag\b/i.test(text) || /^#dag\b/i.test(text);
    if (options.forceDag || explicitDagRun || (!dagClarificationPending && shouldCreateDagRun(text))) {
      clearPendingTimers();
      setDagClarificationPending(false);
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(
        prev,
        traceId,
        makeComposerPendingMessage(traceId, '任務規劃中，請稍等。'),
      ));
      postDebugTrace('ui.composer.task_progress_only', traceId, {user_text: text, forced: !!options.forceDag});
      classifySourceInText(text);
      startDagForMessage(text, {conversationId, traceId});
      return;
    }
    // 一般對話／skill 路徑：標記此 trace 為「執行中」，讓停止鈕可中斷正在跑的 CLI 子程序。
    activeChatTraceRef.current = traceId;
    setActiveChatTrace(traceId);
    // v3.6: Source Trust — 若訊息含 URL，自動分類來源並顯示提示
    classifySourceInText(text);
    // v3.6: 跳脫外部 token 標記（防止使用者輸入干擾 LLM context 邊界）
    // 結果用於送出，不影響 UI 顯示的原始訊息
    const safeText = callWails(() => EscapeExternalTokens(text))
      .catch(() => text);
    // 透過 CLIAdapter 送出（含 SkillInjection），同時記錄互動
    // 送出前把本回合附圖暫存到後端（StageSessionImages）。傳空陣列也會清掉前一則殘留，
    // 避免圖片外洩到下一則。用 window.go 取用，避開 wails 綁定重新生成前的前端 build 破壞。
    const stagedImageURLs = (images || []).map((img) => img.src).filter(Boolean);
    await callWails(() => window.go?.main?.App?.StageSessionImages?.(sessionId, stagedImageURLs)).catch(() => {});
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
    setDismissedDagSuggestionText('');
    scheduleReadinessGateBurstRefresh();
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
    try {
      const state = await callWails(GetReadinessGateState);
      if (state) setReadinessGate(state);
      return state;
    } catch {
      return null;
    }
  }

  function scheduleReadinessGateBurstRefresh({durationMs = 12000, intervalMs = 700} = {}) {
    if (readinessBurstTimerRef.current) {
      window.clearInterval(readinessBurstTimerRef.current);
      readinessBurstTimerRef.current = null;
    }
    const startedAt = Date.now();
    const tick = async () => {
      const state = await refreshReadinessGateState();
      if ((state?.floating_candidates || []).length > 0 || Date.now() - startedAt >= durationMs) {
        window.clearInterval(readinessBurstTimerRef.current);
        readinessBurstTimerRef.current = null;
      }
    };
    readinessBurstTimerRef.current = window.setInterval(tick, intervalMs);
    tick();
  }

  async function handleSelectCandidate(candidateID) {
    const candidates = readinessGate.floating_candidates || [];
    if (!candidates.some((candidate) => candidate.id === candidateID)) return;
    const exclusive = isExclusiveCandidateSet(candidates);
    setSelectedFloatingCandidateIDs((prev) => {
      if (prev.includes(candidateID)) {
        return prev.filter((id) => id !== candidateID);
      }
      return exclusive ? [candidateID] : [...prev, candidateID];
    });
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
    return panelLanguageLabelToLocale(displayLabel);
  }

  async function savePanelPatch(patch) {
    try {
      const panel = normalizePanelSettings({...settingsState.panel, ...patch});
      const next = await callWails(() => SavePanelSettings(panel));
      const normalized = normalizeSettingsState(next, {...settingsState, panel});
      setSettingsState(normalized);
      const activePersona = findActivePersona(normalized);
      if (activePersona?.name) setPersonaName(activePersona.name);
      if (activePersona?.identity) setPersonaJob(activePersona.identity);
      callWails(GetVoiceSettings).then((settings) => setVoiceState(settings || null)).catch(() => {});

      /* i18n: if panel language changed, sync i18n store and reload */
      if (patch.panelLanguage) {
        const locale = panelLangToLocale(patch.panelLanguage);
        if (locale && locale !== useI18n.getState().language) {
          useI18n.getState().setLanguage(locale); // saves to localStorage and updates in place
        }
      }
    } catch {
      const normalized = normalizeSettingsState({
        ...settingsState,
        panel: normalizePanelSettings({...settingsState.panel, ...patch}),
      });
      setSettingsState(normalized);
      const activePersona = findActivePersona(normalized);
      if (activePersona?.name) setPersonaName(activePersona.name);
      if (activePersona?.identity) setPersonaJob(activePersona.identity);
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

  // ── v3.6.2 WA3 Media Provenance（§9A）──
  //
  // 函式群組說明：
  //  refreshWA3TrustList  — 從後端拉取開發者信任清單
  //  importMediaWA3       — 匯入媒體 → 驗證 → 顯示選單解釋 WA3 功能
  //  dismissWA3ImportPopup — 關閉匯入選單

  async function refreshWA3TrustList() {
    callWails(ListWA3TrustedDevelopers)
      .then((list) => setWa3TrustList(list || []))
      .catch(() => {});
  }

  function isWA3MediaPath(filePath) {
    return /\.(png|jpe?g|webp|gif|bmp|tiff?|wav|mp3|m4a|aac|flac|ogg|mp4|mov|m4v|webm)$/i.test(String(filePath || ''));
  }

  // WA3 文檔來源證明適用的文字檔格式（與 media 互斥）。
  function isWA3DocPath(filePath) {
    return /\.(txt|md|markdown|rtf|csv|html?|pdf|docx?|odt)$/i.test(String(filePath || ''));
  }

  function shouldProbeDroppedInstallPackage(paths = []) {
    if (paths.length !== 1 || isWA3MediaPath(paths[0])) return false;
    const name = String(paths[0] || '').split(/[\\/]/).pop() || '';
    return !name.includes('.') || /\.(zip|skill|subagent)$/i.test(name);
  }

  function resolveWA3MediaPath() {
    return wa3ImportPopup?.source_path
      || wa3ImportPopup?.info?.file_path
      || wa3Detail?.file_path
      || '';
  }

  function makeWA3CopyPath(filePath) {
    const rawPath = String(filePath || '');
    const splitAt = Math.max(rawPath.lastIndexOf('/'), rawPath.lastIndexOf('\\'));
    const dir = splitAt >= 0 ? rawPath.slice(0, splitAt + 1) : '';
    const name = splitAt >= 0 ? rawPath.slice(splitAt + 1) : rawPath;
    const extAt = name.lastIndexOf('.');
    return extAt > 0
      ? `${dir}${name.slice(0, extAt)}.wa3-copy${name.slice(extAt)}`
      : `${dir}${name}.wa3-copy`;
  }

  async function importMediaWA3(filePath) {
    try {
      setWa3ActionError('');
      const result = await callWails(() => ImportMediaVerify(filePath));
      if (result) {
        const popup = {...result, source_path: filePath};
        setWa3ImportPopup(popup);
        setWa3Detail(popup.info || null);
        setWa3PollutionResult(popup.info?.pollution || null);
      }
      return result;
    } catch (error) {
      const message = error?.message || String(error);
      setWa3ActionError(message);
      throw error;
    }
  }

  async function loadWA3MediaInfo() {
    const filePath = resolveWA3MediaPath();
    if (!filePath) {
      setWa3ActionError(t('wa3.noMediaPath'));
      return;
    }
    setWa3ActionBusy('info');
    setWa3ActionError('');
    try {
      const info = await callWails(() => GetMediaWA3Info(filePath));
      setWa3Detail(info || null);
      setWa3ImportPopup((current) => current ? {...current, info: info || current.info, source_path: filePath} : current);
      setWa3ToastMsg(t('wa3.infoLoaded'));
    } catch (error) {
      setWa3ActionError(error?.message || String(error));
    } finally {
      setWa3ActionBusy('');
    }
  }

  async function detectWA3Pollution() {
    const filePath = resolveWA3MediaPath();
    if (!filePath) {
      setWa3ActionError(t('wa3.noMediaPath'));
      return;
    }
    setWa3ActionBusy('pollution');
    setWa3ActionError('');
    try {
      const report = await callWails(() => DetectModelPollution(filePath));
      setWa3PollutionResult(report || null);
      setWa3Detail((current) => current ? {...current, pollution: report || null} : current);
      setWa3ToastMsg(report?.is_pollution_risk ? t('wa3.pollutionRiskDetected') : t('wa3.pollutionSafe'));
    } catch (error) {
      setWa3ActionError(error?.message || String(error));
    } finally {
      setWa3ActionBusy('');
    }
  }

  async function showWA3TransferGuidance() {
    setWa3ActionBusy('guidance');
    setWa3ActionError('');
    try {
      const guidance = await callWails(GetWA3TransferGuidance);
      setWa3TransferGuidance(guidance || null);
      setWa3ToastMsg(guidance?.ui_message || t('wa3.exportHint'));
    } catch (error) {
      setWa3ActionError(error?.message || String(error));
    } finally {
      setWa3ActionBusy('');
    }
  }

  async function trustWA3Developer() {
    const signature = wa3Detail?.developer_signature || wa3ImportPopup?.info?.developer_signature;
    if (!signature?.app_id || !signature?.public_key) {
      setWa3ActionError(t('wa3.noDeveloperSignature'));
      return;
    }
    setWa3ActionBusy('trust');
    setWa3ActionError('');
    try {
      await callWails(() => AddWA3TrustedDeveloper(signature.app_id, signature.public_key, signature.app_id));
      await refreshWA3TrustList();
      setWa3ToastMsg(t('wa3.trustAdded'));
    } catch (error) {
      setWa3ActionError(error?.message || String(error));
    } finally {
      setWa3ActionBusy('');
    }
  }

  async function exportWA3WithSidecarCopy() {
    const filePath = resolveWA3MediaPath();
    if (!filePath) {
      setWa3ActionError(t('wa3.noMediaPath'));
      return;
    }
    const destPath = makeWA3CopyPath(filePath);
    setWa3ActionBusy('export');
    setWa3ActionError('');
    try {
      await callWails(() => ExportMediaWithSidecar(filePath, destPath));
      setWa3ToastMsg(t('wa3.exportCopied', { path: destPath }));
    } catch (error) {
      setWa3ActionError(error?.message || String(error));
    } finally {
      setWa3ActionBusy('');
    }
  }

  function dismissWA3ImportPopup() {
    setWa3ImportPopup(null);
    setWa3ActionError('');
  }

  // WA3 驗證狀態對應的 icon 與顏色
  const wa3StatusConfig = {
    exact_original:        { icon: '✅', color: '#2ecc40', label: t('wa3.originalFile') },
    wa3_app_processed:     { icon: '🔏', color: '#3498db', label: t('wa3.appProcessed') },
    platform_processed_copy: { icon: '📋', color: '#f39c12', label: t('wa3.platformProcessed') },
    unauthorized_copy:     { icon: '🚫', color: '#e74c3c', label: t('wa3.unauthorizedCopy') },
    content_modified:      { icon: '✏️', color: '#e67e22', label: t('wa3.contentModified') },
    model_pollution_risk:  { icon: '☣️', color: '#e74c3c', label: t('wa3.pollutionRisk') },
    unverified:            { icon: '❓', color: '#95a5a6', label: t('wa3.unverified') },
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
      // 統一接口：頭像直接走「紀念照產圖設定」(ComfyUI / 雲端)，不再另設 API。
      setAvatarModeNotice('正在用產圖設定生成頭像…（首次較久）');
      try {
        await callWails(() => GenerateAvatarViaImageGen(targetPersonaID, 'idle'));
        await loadCurrentAvatar(targetPersonaID);
        setAvatarModeNotice('已用產圖設定生成頭像 ✓');
        setAvatarPickerOpen(false);
      } catch (e) {
        setAvatarModeNotice('頭像產圖失敗：' + String(e && e.message ? e.message : e) + '（請先在 CLI 互動的「📸 拍照 → ⚙ 設定」設好 ComfyUI 或雲端）');
      }
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
    if (persona.id === lockedPersonaId) persona.name = t('persona.lockedName');
    try {
      const next = await callWails(() => SavePersona(persona));
      const normalized = normalizeSettingsState(next, settingsState);
      setSettingsState(normalized);
      if (persona.id === normalized.activePersonaId) {
        const activePersona = findActivePersona(normalized);
        if (activePersona?.name) setPersonaName(activePersona.name);
        if (activePersona?.identity) setPersonaJob(activePersona.identity);
      }
    } catch {
      setSettingsState((prev) => {
        const normalizedPersonas = normalizeLockedPersonas(
          prev.personas.map((item) => (item.id === persona.id ? persona : item)),
          prev.removedDefaultPersonaIds || [],
          personaLocaleFromPanel(prev.panel),
        );
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
      patrolDialogue: packageData.patrolDialogue || packageData.patrol_dialogue || '',
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
    const manifestJSON = buildPersonaPackageManifest(
      parsedData.name || fileName.replace(/\.[^.]+$/, ''),
      fileName,
      parsedData,
    );
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

  async function refreshSchedulerClock() {
    try {
      const clock = await callWails(GetSchedulerClock);
      setSchedulerClock(clock || null);
      return clock;
    } catch (error) {
      setSchedulerError(error?.message || String(error));
      return null;
    }
  }

  async function refreshSchedulerJobs() {
    try {
      const jobs = await callWails(ListScheduledJobs);
      setSchedulerJobs(Array.isArray(jobs) ? jobs : []);
      return jobs;
    } catch (error) {
      setSchedulerError(error?.message || String(error));
      return [];
    }
  }

  async function openSchedulerPanel() {
    setToolPopupsOpen({left: false, right: false});
    setSchedulerPanelOpen(true);
    setSchedulerError('');
    await Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()]);
  }

  async function createSchedulerJob() {
    const name = schedulerDraft.name.trim();
    const cronExpr = schedulerDraft.cronExpr.trim();
    if (!name || !cronExpr) {
      setSchedulerError('請填寫名稱與 cron 時間');
      return;
    }
    const payload = schedulerDraft.actionPayload.trim() || JSON.stringify({
      event_name: 'scheduler:reminder',
      data: {title: name},
    });
    setSchedulerBusy(true);
    setSchedulerError('');
    try {
      await callWails(() => CreateScheduledJob(name, cronExpr, 'event', payload));
      setSchedulerDraft({name: '', cronExpr, actionPayload: ''});
      await Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()]);
      setToolResult({toolId: 'scheduler', ok: true, message: '排程已建立'});
    } catch (error) {
      setSchedulerError(error?.message || String(error));
    } finally {
      setSchedulerBusy(false);
    }
  }

  async function applySchedulerJobPatch(job, patch) {
    if (!job?.id || !patch) return null;
    const name = patch.name || job.name;
    const cronExpr = patch.cronExpr || job.cron_expr;
    const actionType = patch.actionType || job.action_type || 'event';
    const actionPayload = patch.actionPayload || job.action_payload || schedulerDefaultPayload(name);
    return callWails(() => UpdateScheduledJob(job.id, name, cronExpr, actionType, actionPayload));
  }

  async function updateSchedulerJob(id, action) {
    if (!id) return;
    setSchedulerBusy(true);
    setSchedulerError('');
    try {
      if (action === 'pause') await callWails(() => PauseScheduledJob(id));
      if (action === 'resume') await callWails(() => ResumeScheduledJob(id));
      if (action === 'delete') await callWails(() => DeleteScheduledJob(id));
      await Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()]);
    } catch (error) {
      setSchedulerError(error?.message || String(error));
    } finally {
      setSchedulerBusy(false);
    }
  }

  async function bootstrapSchedulerSkill(job) {
    if (!job?.id) return;
    setSchedulerBusy(true);
    setSchedulerError('');
    try {
      const result = await callWails(() => BootstrapScheduledSkill(job.id));
      await Promise.all([refreshSchedulerClock(), refreshSchedulerJobs(), refreshAvailableAdapters()]);
      if (result?.source_sub_id) {
        setActiveHaoraId(result.source_sub_id);
        await callWails(() => SetActiveConversationAgent(result.source_sub_id));
      }
      setToolResult({
        toolId: 'scheduler',
        ok: true,
        message: `已建立排程 skill：${result?.skill_id || job.name}`,
      });
    } catch (error) {
      setSchedulerError(error?.message || String(error));
    } finally {
      setSchedulerBusy(false);
    }
  }

  function resetSchedulerComposerState(message = '') {
    setSchedulerConversation(null);
    setSchedulerDraft({name: '', cronExpr: '@daily', actionPayload: ''});
    if (message) setToolResult({toolId: 'scheduler', ok: true, message});
  }

  async function commitSchedulerConversation(sendSchedulerReply) {
    const pending = schedulerConversation;
    if (!pending || pending.phase !== 'confirm') return false;

    const normalized = normalizeSchedulerDraft(pending.draft);
    const missing = schedulerMissingSlots(normalized);
    if (missing.length) {
      setSchedulerConversation({...pending, phase: 'collecting', draft: normalized, missing});
      sendSchedulerReply(schedulerQuestionForMissing(missing));
      return true;
    }

    setSchedulerBusy(true);
    try {
      if (pending.mode === 'update' && pending.job) {
        await applySchedulerJobPatch(pending.job, {
          name: normalized.name,
          cronExpr: normalized.cronExpr,
          actionType: pending.job.action_type || 'event',
          actionPayload: schedulerDefaultPayload(normalized.name, normalized.actionText || normalized.name, normalized.summary),
        });
        sendSchedulerReply(`Ai:${t('chatSystem.scheduler.updated', {name: normalized.name})}`);
      } else {
        await callWails(() => CreateScheduledJob(
          normalized.name,
          normalized.cronExpr,
          'event',
          schedulerDefaultPayload(normalized.name, normalized.actionText || normalized.name, normalized.summary),
        ));
        sendSchedulerReply(`Ai:${t('chatSystem.scheduler.created', {name: normalized.name})}`);
      }
      resetSchedulerComposerState('排程已寫入');
      await Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()]);
    } catch (error) {
      setSchedulerError(error?.message || String(error));
      sendSchedulerReply(`Ai:${t('chatSystem.scheduler.writeFailed', {error: error?.message || String(error)})}`);
    } finally {
      setSchedulerBusy(false);
    }
    return true;
  }

  async function confirmComposerAction() {
    if (schedulerConversation?.phase !== 'confirm') return;
    const conversationId = activeConversationIdRef.current || 'main';
    await commitSchedulerConversation((message) => {
      setConversationMessages(conversationId, (prev) => [...prev, message]);
      persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, '')).catch(() => {});
    });
  }

  function cancelComposerAction() {
    if (schedulerConversation?.phase !== 'confirm') return;
    const conversationId = activeConversationIdRef.current || 'main';
    const message = `Ai:${t('chatSystem.scheduler.confirmationCancelled')}`;
    setConversationMessages(conversationId, (prev) => [...prev, message]);
    persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, '')).catch(() => {});
    resetSchedulerComposerState('已取消排程確認');
  }

  async function handleSchedulerComposerIntent(text, conversationId, traceId, clearPendingTimers) {
    const sendSchedulerReply = (message) => {
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, traceId, message));
      persistConversationEntry(conversationId, 'assistant', message.replace(/^Ai:/, ''), traceId).catch(() => {});
    };
    const pending = schedulerConversation;
    // 輕量 regex 只當「這句是否與排程有關」的前置過濾，避免攔截一般聊天；
    // 真正的意圖判斷、時間換算、名稱/動作抽取一律交給模型（見下方 model-first）。
    const startsScheduler = parseSchedulerConversationIntent(text, schedulerJobs);
    if (!pending && !startsScheduler) return false;

    clearPendingTimers();
    setSchedulerError('');

    // 直接可判定、不需動用模型的快捷詞：取消這次設定 / 在 confirm 階段的肯定確認。
    if (pending && isSchedulerCancellation(text)) {
      resetSchedulerComposerState();
      sendSchedulerReply(`Ai:${t('chatSystem.scheduler.setupCancelled')}`);
      return true;
    }
    if (pending && pending.phase === 'confirm' && isSchedulerAffirmation(text)) {
      return commitSchedulerConversation(sendSchedulerReply);
    }

    // 取得既有排程清單，供模型判斷「要改哪一筆」。
    let jobs = schedulerJobs;
    try {
      const refreshed = await refreshSchedulerJobs();
      if (Array.isArray(refreshed)) jobs = refreshed;
    } catch (error) {
      postDebugTrace('ui.scheduler.jobs_refresh.error', traceId, {error: error?.message || String(error)});
    }

    // ---- 模型優先：由模型合成意圖與所有欄位（時間→cron、名稱、動作、追問）。 ----
    let model = null;
    try {
      model = await synthesizeSchedulerWithModel({pending, text, jobs, traceId});
    } catch (error) {
      postDebugTrace('ui.scheduler.model_fallback', traceId, {error: error?.message || String(error)});
    }
    if (model) {
      const handled = applySchedulerModelResult({model, pending, jobs, sendSchedulerReply});
      if (handled !== null) return handled;
      // handled === null：模型判定與排程無關（intent=none 且無進行中草稿）→ 交回一般路由。
      postDebugTrace('ui.scheduler.model_intent_none', traceId, {user_text: text});
      return false;
    }

    // ---- 後援：模型失效時退回本機 regex，並明確告知使用者已降級。 ----
    postDebugTrace('ui.scheduler.degraded', traceId, {user_text: text});
    setToolResult({toolId: 'scheduler', ok: false, message: t('chatSystem.scheduler.degradedNoticeShort')});
    return handleSchedulerComposerIntentFallback({pending, text, jobs, sendSchedulerReply});
  }

  // synthesizeSchedulerWithModel 把目前情境（草稿/階段/既有清單）打包丟給模型，
  // 取回完整的排程規劃結果。失敗時 throw，由呼叫端退回後援。
  async function synthesizeSchedulerWithModel({pending, text, jobs, traceId}) {
    const adapterId = activeAdapterIdRef.current || activeAdapterId || '';
    const sessionId = appSessionId || activeConversationIdRef.current || 'main';
    const base = normalizeSchedulerDraft(pending?.draft || {});
    const input = {
      current_draft: base,
      phase: pending?.phase || 'start',
      mode: pending?.mode || '',
      target_job: pending?.job
        ? {no: schedulerJobNo(pending.job), name: pending.job.name, cron_expr: pending.job.cron_expr}
        : null,
      jobs: (Array.isArray(jobs) ? jobs : []).map((job, index) => ({
        no: schedulerJobNo(job, index),
        name: job.name,
        cron_expr: job.cron_expr,
        action: formatSchedulerPayload(job.action_payload),
      })),
    };
    const normalized = await callWails(() => NormalizeSchedulerDraft(
      adapterId,
      sessionId,
      JSON.stringify(input),
      text,
      `${traceId}-scheduler-synth`,
    ));
    postDebugTrace('ui.scheduler.synth.after', traceId, {request: text, response: normalized || null});
    if (!normalized) throw new Error('模型未回傳排程結果');
    return normalized;
  }

  // applySchedulerModelResult 把模型結果套成 UI 草稿與對話狀態。
  // 回傳 true=已處理；null=與排程無關（交回一般路由）。
  function applySchedulerModelResult({model, pending, jobs, sendSchedulerReply}) {
    const intent = String(model.intent || '').toLowerCase();
    const jobList = Array.isArray(jobs) ? jobs : [];

    // 沒有進行中的草稿時，先依模型意圖分流。
    if (!pending) {
      if (intent === 'none') return null;
      if (intent === 'open') {
        setSchedulerPanelOpen(true);
        Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()]).catch(() => {});
        sendSchedulerReply(`Ai:${t('chatSystem.scheduler.openList')}`);
        return true;
      }
    }

    // 決定模式與目標 job（沿用 pending，或依模型 target_job_no 對應既有清單）。
    let mode = pending?.mode || (intent === 'update' ? 'update' : 'create');
    let job = pending?.job || null;
    if (!job && intent === 'update' && Number(model.target_job_no) > 0) {
      job = jobList.find((item, index) => schedulerJobNo(item, index) === Number(model.target_job_no)) || null;
      if (!job) {
        setSchedulerPanelOpen(true);
        Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()]).catch(() => {});
        sendSchedulerReply(`Ai:${t('chatSystem.scheduler.updateNotFound')}`);
        return true;
      }
    }

    // 以 pending 草稿或既有 job 為基底，疊上模型欄位。
    const baseDraft = pending?.draft
      ? {...pending.draft}
      : (job
          ? {name: job.name || '', cronExpr: job.cron_expr || '', actionText: formatSchedulerPayload(job.action_payload), actionPayload: job.action_payload || ''}
          : {});
    const next = normalizeSchedulerDraft(baseDraft);
    const title = String(model.title || '').trim();
    const action = String(model.action_text || '').trim();
    const summary = String(model.summary || '').trim();
    const cronExpr = String(model.cron_expr || '').trim();
    const timeText = String(model.time_text || '').trim();
    if (title) {
      next.name = title.slice(0, 32);
      if (!action) next.actionText = title.slice(0, 80);
    }
    if (action) next.actionText = action.slice(0, 80);
    if (summary) next.summary = summary.slice(0, 180);
    if (cronExpr) {
      next.cronExpr = cronExpr;
    } else if (timeText && (hasSchedulerTimeText(timeText) || hasSchedulerClockText(timeText))) {
      next.cronExpr = parseSchedulerTimeText(timeText, next.cronExpr || '');
    }

    const normalized = normalizeSchedulerDraft(next);
    normalized.actionPayload = schedulerDefaultPayload(
      normalized.name || '提醒',
      normalized.actionText || normalized.name,
      normalized.summary,
    );
    const missing = schedulerMissingSlots(normalized);
    const phase = missing.length ? 'collecting' : 'confirm';
    setSchedulerConversation({mode, phase, draft: normalized, missing, job});
    setSchedulerDraft({
      name: normalized.name,
      cronExpr: normalized.cronExpr || '@daily',
      actionPayload: schedulerDefaultPayload(normalized.name || '提醒', normalized.actionText || normalized.name, normalized.summary),
    });
    const question = String(model.question || '').trim();
    sendSchedulerReply(missing.length
      ? (question ? `Ai:${question}` : schedulerQuestionForMissing(missing))
      : schedulerConfirmationMessage(normalized, mode, job));
    setToolResult({toolId: 'scheduler', ok: true, message: missing.length ? '排程資訊待補齊' : '排程資訊等待確認'});
    return true;
  }

  // handleSchedulerComposerIntentFallback 為模型失效時的本機 regex 後援，
  // 行為等同舊版硬編碼流程，但會在回覆前明確標示「已降級為本機規則」。
  function handleSchedulerComposerIntentFallback({pending, text, jobs, sendSchedulerReply}) {
    let noticeShown = false;
    const sendWithNotice = (message) => {
      let out = message;
      if (!noticeShown) {
        out = out.replace(/^Ai:/, `Ai:${t('chatSystem.scheduler.degradedNotice')}\n\n`);
        noticeShown = true;
      }
      sendSchedulerReply(out);
    };

    if (pending) {
      const preferredSlot = pending.missing?.[0] || '';
      const nextDraft = mergeSchedulerSlotsFromText(pending.draft, text, preferredSlot);
      const missing = schedulerMissingSlots(nextDraft);
      const phase = missing.length ? 'collecting' : 'confirm';
      setSchedulerConversation({...pending, draft: nextDraft, missing, phase});
      setSchedulerDraft({
        name: nextDraft.name,
        cronExpr: nextDraft.cronExpr || '@daily',
        actionPayload: schedulerDefaultPayload(nextDraft.name || '提醒', nextDraft.actionText || nextDraft.name, nextDraft.summary),
      });
      sendWithNotice(missing.length
        ? schedulerQuestionForMissing(missing)
        : schedulerConfirmationMessage(nextDraft, pending.mode, pending.job));
      return true;
    }

    const intent = parseSchedulerConversationIntent(text, Array.isArray(jobs) ? jobs : schedulerJobs);
    if (!intent) return false;
    if (intent.type === 'open') {
      setSchedulerPanelOpen(true);
      Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()]).catch(() => {});
      sendWithNotice(`Ai:${intent.message || t('chatSystem.scheduler.openListForUpdate')}`);
      return true;
    }

    const mode = intent.type === 'update' ? 'update' : 'create';
    const baseDraft = intent.type === 'update'
      ? {
          name: intent.patch?.name || intent.job?.name || '',
          cronExpr: intent.patch?.cronExpr || intent.job?.cron_expr || '',
          actionText: formatSchedulerPayload(intent.patch?.actionPayload || intent.job?.action_payload),
          actionPayload: intent.patch?.actionPayload || intent.job?.action_payload || '',
        }
      : intent.draft;
    const normalized = mergeSchedulerSlotsFromText(baseDraft, text);
    const missing = schedulerMissingSlots(normalized);
    const phase = missing.length ? 'collecting' : 'confirm';
    setSchedulerConversation({mode, phase, draft: normalized, missing, job: intent.job || null});
    setSchedulerDraft({
      name: normalized.name,
      cronExpr: normalized.cronExpr || '@daily',
      actionPayload: schedulerDefaultPayload(normalized.name || '提醒', normalized.actionText || normalized.name, normalized.summary),
    });
    sendWithNotice(missing.length
      ? schedulerQuestionForMissing(missing)
      : schedulerConfirmationMessage(normalized, mode, intent.job));
    setToolResult({toolId: 'scheduler', ok: true, message: '排程資訊等待確認（本機模式）'});
    return true;
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

  function formatSkillNativeDropDetail(result) {
    const landed = result?.landed_path || result?.message || 'Skill 已拖出';
    if (!result?.drop_target_kind || !result?.drop_target_dir) return landed;
    return `${landed}\n${result.drop_target_kind}: ${result.drop_target_dir}`;
  }

  async function startNativeSkillExport(tool, source = 'right') {
    if (!tool?.target) {
      setToolResult({toolId: tool?.id || 'skill', ok: false, message: '找不到 skill_id，無法拖出'});
      return null;
    }
    try {
      const result = await callWails(() => NativeDragExportSkill(tool.target));
      if (result?.status === 'success' && result?.landed_path) {
        setSkillExportDialog({
          skillID: result.skill_id || tool.target,
          toolID: tool.id,
          name: result.display_name || tool.title || tool.target,
          tempExportDir: result.export_dir,
          landedPath: result.landed_path,
          landedDetail: formatSkillNativeDropDetail(result),
        });
      } else if (result?.status !== 'cancelled') {
        setToolResult({toolId: tool.id, ok: false, message: result?.message || 'Skill 拖曳失敗'});
        // 原生拖曳失敗（如 beginDraggingSession returned nil）時退回操作卡，
        // 讓「解除／複製／取消」功能不因原生層失效而消失。
        setDragActionTool((current) => current || {tool, source});
      }
      return result;
    } catch (error) {
      setToolResult({toolId: tool.id, ok: false, message: error?.message || String(error)});
      setDragActionTool((current) => current || {tool, source});
      return null;
    }
  }

  async function finalizeSkillExport(action) {
    if (!skillExportDialog) return;
    const target = skillExportDialog;
    setSkillExportDialog(null);
    try {
      await callWails(() => FinalizeNativeSkillExport(
        action,
        target.skillID || '',
        target.tempExportDir || '',
        target.landedPath || '',
      ));
      if (action === 'remove') {
        setFavoriteToolIds((current) => current.filter((id) => id !== target.toolID));
        setHiddenToolIds((current) => current.filter((id) => id !== target.toolID));
        callWails(ListTools)
          .then((nextTools) => setTools(normalizeToolList(nextTools)))
          .catch(() => {});
      }
      setToolResult({
        toolId: target.toolID || `skill:${target.skillID}`,
        ok: true,
        message: action === 'cancel'
          ? '已刪除剛剛拖出的 skill 資料夾'
          : action === 'remove'
            ? 'Skill 已複製，並已從本機索引移除'
            : 'Skill 已複製',
      });
    } catch (error) {
      setToolResult({toolId: target.toolID || `skill:${target.skillID}`, ok: false, message: error?.message || String(error)});
    }
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
    const [files, videos, images, codeMetaList] = await Promise.all([
      callWails(ListReferenceFiles).catch(() => []),
      callWails(ListVideoFiles).catch(() => []),
      callWails(ListReferenceImages).catch(() => []),
      callWails(ListCodeArtifacts).catch(() => []),
    ]);
    const codeMap = {};
    (Array.isArray(codeMetaList) ? codeMetaList : []).forEach((meta) => {
      if (meta?.file_name) codeMap[meta.file_name] = meta;
    });
    setCodeArtifacts(codeMap);
    const loadedFiles = [
      ...(Array.isArray(files) ? files : []),
      ...(Array.isArray(videos) ? videos : []),
      ...(Array.isArray(images) ? images : []),
    ];
    setReferenceFiles((current) => mergeReferenceLibraryFiles(current, loadedFiles));
    return loadedFiles;
  }

  useEffect(() => {
    const onExpand = (event) => {
      const fileName = event?.detail?.fileName;
      if (fileName) openCodeArtifactModal(fileName);
    };
    window.addEventListener('code-artifact:expand', onExpand);
    return () => window.removeEventListener('code-artifact:expand', onExpand);
  }, []);

  // 自動回修覆寫檔案後（code_artifact:updated）：刷新卡片；彈窗開著同一支就重載內容。
  useEffect(() => {
    let off = null;
    try {
      off = EventsOn('code_artifact:updated', (meta) => {
        refreshReferenceFiles().catch(() => {});
        const fileName = meta?.file_name;
        if (!fileName) return;
        setCodeArtifactModal((current) => {
          if (current?.meta?.file_name === fileName) {
            callWails(() => GetCodeArtifactDetail(fileName))
              .then((detail) => { if (detail) setCodeArtifactModal(detail); })
              .catch(() => {});
          }
          return current;
        });
      });
    } catch (_) { /* binding 未就緒（測試環境）忽略 */ }
    return () => { try { off?.(); } catch (_) {} };
  }, []);

  async function openCodeArtifactModal(fileName) {
    try {
      const detail = await callWails(() => GetCodeArtifactDetail(fileName));
      if (detail) setCodeArtifactModal(detail);
    } catch (err) {
      setToolResult({toolId: 'code-artifact', ok: false, message: String(err?.message || err)});
    }
  }

  async function exportCodeArtifactToFolder(fileName) {
    try {
      const landed = await callWails(() => ExportCodeArtifactToFolder(fileName));
      if (landed) setToolResult({toolId: 'code-artifact', ok: true, message: '已匯出程式碼：' + landed});
    } catch (err) {
      setToolResult({toolId: 'code-artifact', ok: false, message: '匯出失敗：' + String(err?.message || err)});
    }
  }

  function formatReferenceNativeDropDetail(result) {
    const landed = result?.landed_path || result?.message || '引用文件已拖出';
    if (!result?.drop_target_kind || !result?.drop_target_dir) return landed;
    return `${landed}\n${result.drop_target_kind}: ${result.drop_target_dir}`;
  }

  function formatNativeSearchSummaryDropDetail(result) {
    const landed = result?.landed_path || result?.message || '搜尋摘要已拖出';
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

  function showNativeSearchSummaryExportDialog(result) {
    if (result?.status !== 'success' || !result?.landed_path) return;
    setSearchSummaryExportDialog({
      name: result.display_name || result.filename || '搜尋摘要.md',
      tempPath: result.temp_path || '',
      landedPath: result.landed_path || '',
      checksum: result.checksum || '',
      landedDetail: formatNativeSearchSummaryDropDetail(result),
    });
  }

  useEffect(() => {
    const offNativeReferenceExport = EventsOn('reference:native_completed', (result) => {
      handleReferenceInternalDrag(false);
      showNativeReferenceExportDialog(result);
    });
    return () => offNativeReferenceExport();
  }, []);

  useEffect(() => {
    const offNativeSearchSummaryExport = EventsOn('searchsummary:native_completed', (result) => {
      showNativeSearchSummaryExportDialog(result);
    });
    return () => offNativeSearchSummaryExport();
  }, []);

  async function startNativeSearchSummaryExport(card) {
    // card 由 MessageRow 算好（與 Popover 預覽同一份），匯出即所見，避免漂移。
    const markdown = String(card?.markdown || '');
    if (!markdown.trim()) {
      setToolResult({toolId: 'doc-entrance', ok: false, message: '搜尋摘要內容是空的'});
      return null;
    }
    try {
      const result = await callWails(() => NativeDragExportSearchSummary(
        card?.title || '搜尋摘要',
        card?.filename || '',
        markdown,
        `search-summary-${Date.now()}`,
      ));
      // 彈窗統一由 searchsummary:native_completed 事件開（與 reference 一致），這裡只回報失敗。
      if (result && result.status !== 'success' && result.status !== 'cancelled') {
        setToolResult({toolId: 'doc-entrance', ok: false, message: result?.message || '搜尋摘要拖曳失敗'});
      }
      return result;
    } catch (error) {
      setToolResult({toolId: 'doc-entrance', ok: false, message: error?.message || String(error)});
      return null;
    }
  }

  async function startNativeReferenceExport(file) {
    if (!file?.path || file.source !== 'library') {
      handleReferenceInternalDrag(false);
      setToolResult({toolId: 'doc-entrance', ok: false, message: '只有引用庫內的本機檔案可以拖出複製'});
      return null;
    }
    try {
      const codeMeta = codeArtifacts[file.name];
      const isBundle = Boolean(codeMeta?.binary_path);
      const result = isBundle
        ? await callWails(() => NativeDragExportCodeArtifactBundle(file.name))
        : await callWails(() => NativeDragExportReferenceFile(file.path));
      if (result?.status === 'success' && result?.landed_path) {
        if (isBundle) {
          // 資料夾（原始碼＋執行檔）落地：不進單檔的移除/複製選單
          setToolResult({toolId: 'code-artifact', ok: true, message: '已拖出資料夾（原始碼＋執行檔）：' + result.landed_path});
        } else {
          showNativeReferenceExportDialog(result, file);
        }
      } else if (result?.status !== 'cancelled') {
        setToolResult({toolId: 'doc-entrance', ok: false, message: result?.message || '引用文件拖曳失敗'});
        await refreshReferenceFiles();
      }
      return result;
    } catch (error) {
      setToolResult({toolId: 'doc-entrance', ok: false, message: error?.message || String(error)});
      await refreshReferenceFiles();
      return null;
    } finally {
      handleReferenceInternalDrag(false);
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

  async function handleSearchSummaryExportAction(action) {
    if (!searchSummaryExportDialog) return;
    const target = searchSummaryExportDialog;
    setSearchSummaryExportDialog(null);
    try {
      await callWails(() => FinalizeNativeSearchSummaryExport(
        action,
        target.tempPath || '',
        target.landedPath || '',
        target.checksum || '',
      ));
      if (action === 'cancel') {
        setToolResult({toolId: 'doc-entrance', ok: true, message: '已刪除剛剛拖出的搜尋摘要'});
      } else {
        setToolResult({toolId: 'doc-entrance', ok: true, message: '搜尋摘要已複製'});
      }
    } catch (error) {
      setToolResult({toolId: 'doc-entrance', ok: false, message: error?.message || String(error)});
    }
  }

  async function importReferencePaths(paths = [], options = {}) {
    const addedVia = options.addedVia || '';
    const nativePaths = normalizeReferenceImportPaths(paths);
    const importPaths = [];
    const inFlight = referenceImportInFlightRef.current;
    for (const path of nativePaths) {
      if (inFlight.has(path)) continue;
      inFlight.add(path);
      importPaths.push(path);
    }
    for (const path of importPaths) {
      try {
        const name = String(path || '').split(/[\/]/).pop() || t('system.unnamedFile');
        // GGUF 是模型不是文件：改走 Ollama 匯入（ollama create），完成後進 Adapter 清單。
        // 不複製進引用庫——7GB 的模型檔複製一份太傷，匯入後 blob 會進 Ollama 自己的模型庫。
        if (/\.gguf$/i.test(name)) {
          try {
            const job = await callWails(() => StartGGUFImport(path));
            setToolResult({toolId: 'doc-entrance', ok: true, message: `GGUF 模型匯入已開始：${job?.modelName || name}（背景執行，完成後自動加入 Adapter）`});
          } catch (error) {
            setToolResult({toolId: 'doc-entrance', ok: false, message: error?.message || 'GGUF 模型匯入失敗'});
          }
          continue;
        }
        let referencePathForStatus = path;
        let referenceImportReady = false;
        setReferenceFiles((current) => appendUniqueReferenceFile(current, {
          name,
          path,
          source: 'pending',
          addedVia,
          status: 'importing',
          detail: '正在複製到引用庫',
        }));
        try {
          // §3.1.11 影片落獨立資料夾 data/videos，其餘維持引用庫；UI 仍同一份清單
          const imported = await callWails(() => (isVideoPath(path) ? ImportVideoFile(path) : ImportReferenceFile(path)));
          const importedFile = {
            ...imported,
            addedVia: addedVia || imported?.addedVia || '',
            status: isWA3MediaPath(path) ? 'checking' : 'ready',
            detail: isWA3MediaPath(path) ? '正在檢查媒體來源' : '',
          };
          referencePathForStatus = importedFile.path || path;
          referenceImportReady = true;
          setReferenceFiles((current) => appendUniqueReferenceFile(
            current.filter((file) => file.path !== path),
            importedFile,
          ));
          setToolResult({
            toolId: 'doc-entrance',
            ok: true,
            message: `${isWA3MediaPath(path) ? '已加入引用媒體' : '已加入引用文件'}：${importedFile.name || name}`,
          });
        } catch (error) {
          const message = error?.message || '無法複製，已保留原路徑';
          setReferenceFiles((current) => appendUniqueReferenceFile(current.filter((file) => file.path !== path), {
            name,
            path,
            source: 'memory',
            addedVia,
            status: 'error',
            detail: message,
          }));
          setToolResult({toolId: 'doc-entrance', ok: false, message});
        }
        if (isWA3MediaPath(path)) {
          try {
            await importMediaWA3(path);
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
        else if (referenceImportReady && isWA3DocPath(path)) {
          try {
            await CreateDocumentWA3(referencePathForStatus);
            setReferenceFiles((current) => updateReferenceFileStatus(current, referencePathForStatus, {
              status: 'ready',
              detail: '文件來源證明已建立',
            }));
          } catch (error) {
            // best-effort：建 .wa3.json 失敗不影響文件已匯入
            setReferenceFiles((current) => updateReferenceFileStatus(current, referencePathForStatus, {
              status: 'ready',
              detail: '已匯入（來源證明稍後重試）',
            }));
          }
        }
      } finally {
        referenceImportInFlightRef.current.delete(path);
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
    callWails(() => ListExternalLinksByType('shared_source'))
      .then((links) => setExtSharedLinks(links || []))
      .catch(() => {});
    callWails(ListSharedSourceFiles)
      .then((listings) => setSharedSourceListings(listings || []))
      .catch(() => {});
  }

  async function refreshLinkPreviewSuggestions(shouldSuggest) {
    if (!shouldSuggest) {
      setLinkPreviewSuggestions([]);
      return;
    }
    try {
      // GGUF：以貼上的路徑為中心，上下一層資料夾找 .gguf（貼錯檔名 / 貼到資料夾都能救）。
      const value = referenceLinkValue.trim();
      const [detected, ggufNearby] = await Promise.all([
        callWails(AutoDetectCLI).catch(() => []),
        value ? callWails(() => SuggestGGUFFiles(value)).catch(() => []) : Promise.resolve([]),
      ]);
      const ggufSuggestions = (ggufNearby || [])
        .filter((item) => item?.path)
        .map((item) => ({
          name: `GGUF 模型：${item.name || item.path}`,
          path: item.path,
          detected: true,
        }));
      const detectedSuggestions = (detected || [])
        .filter((item) => item?.found && item?.path && item?.supported !== false)
        .map((item) => ({
          name: item.name || item.adapter_id || 'CLI',
          path: item.path,
          detected: true,
        }));
      setLinkPreviewSuggestions([...ggufSuggestions, ...detectedSuggestions].slice(0, 6));
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
      groq: 'openai/gpt-oss-20b',
      together: 'meta-llama/Llama-3.3-70B-Instruct-Turbo',
      perplexity: 'sonar',
      cohere: 'command-a-03-2025',
      fireworks: 'accounts/fireworks/models/llama-v3p1-70b-instruct',
      cerebras: 'llama3.1-8b',
      huggingface: 'openai/gpt-oss-120b:cerebras',
      'nvidia-nim': 'meta/llama-3.1-8b-instruct',
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

  async function previewLLMAPIConnection() {
    if (!llmAPISetup) return;
    const missing = [];
    if (!llmAPISetup.apiKey?.trim()) missing.push('API Key');
    if (!llmAPISetup.baseURL?.trim()) missing.push('Base URL');
    if (!llmAPISetup.model?.trim()) missing.push('Model');
    if (missing.length) {
      setToolResult({
        toolId: 'reference-link',
        ok: false,
        message: t('link.missingFields', { fields: missing.join('、') }),
      });
      return;
    }
    // 真的打一發：確認連得上、金鑰對、模型存在。
    setToolResult({ toolId: 'reference-link', ok: true, message: t('llmSetup.testing') });
    try {
      const result = await callWails(() => TestLLMAPIConnection(
        llmAPISetup.providerId || 'generic-api',
        llmAPISetup.baseURL || '',
        llmAPISetup.model || '',
        llmAPISetup.apiKey || '',
      ));
      setToolResult({
        toolId: 'reference-link',
        ok: !!result?.ok,
        message: result?.message || (result?.ok ? t('llmSetup.testOk') : t('llmSetup.testFail')),
      });
    } catch (error) {
      setToolResult({
        toolId: 'reference-link',
        ok: false,
        message: t('llmSetup.testFail') + '：' + (error?.message || String(error)),
      });
    }
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
      const isGGUFImport = isAdapterCandidate && /gguf/i.test(`${link?.label || ''} ${link?.url || ''} ${linkPreview?.reason || ''}`);
      const isOllamaLibrary = !isGGUFImport && isAdapterCandidate && /ollama/i.test(`${link?.label || ''} ${link?.url || ''} ${linkPreview?.reason || ''}`);
      const isSharedSource = link?.link_type === 'shared_source' || linkPreview?.link_type === 'shared_source';
      setToolResult({toolId: 'reference-link', ok: true, message: isSharedSource ? t('link.addedSharedSource') : isAdapterCandidate ? (isGGUFImport ? 'GGUF 匯入已開始（背景執行，完成後自動加入本機模型）' : isOllamaLibrary ? t('link.addedOllama') : t('link.addedCLI')) : t('link.addedLink')});
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

  // Idle learning prepares a real Review Panel digest before showing the hint.
  async function markLearningDigestReady(reason = 'idle') {
    if (learningDigestReadyRef.current || learningDigestPreparingRef.current) return;
    learningDigestPreparingRef.current = true;
    try {
      const result = await callWails(() => PrepareLearningDigest(reason)).catch(() => null);
      if (result?.has_updates === false) return;
      learningDigestReadyRef.current = true;
      setLearningDigestReady(true);
      loadReviewPanelData();
      try {
        window.localStorage.setItem(learningDigestStorageKey, 'true');
      } catch {
        /* local notification hint only */
      }
    } finally {
      learningDigestPreparingRef.current = false;
    }
  }

  function confirmLearningBackground(event) {
    event.preventDefault();
    event.returnValue = t('adapter.learningModeHint');
    return event.returnValue;
  }

  // v3.7: 學習按鈕 = 純生態系整理模式（不再啟動螢幕錄製）。
  // 開啟後閒置 20 分鐘會觸發 PrepareLearningDigest 整理 CLI token、sub 流程與 skill。
  // 螢幕操作錄製請用「示範」按鈕（Draft Sandbox）或 VL 面板的 toggleVisualLearningRecording。
  function toggleLearning() {
    const conversationId = activeConversationIdRef.current || 'main';
    if (learningEnabled) {
      setLearningEnabled(false);
      setLearningDigestReady(false);
      learningDigestReadyRef.current = false;
      try { window.localStorage.removeItem(learningDigestStorageKey); } catch { /* */ }
      setConversationMessages(conversationId, (prev) => [
        ...prev,
        t('chatSystem.learning.modeOff'),
      ]);
    } else {
      setLearningEnabled(true);
      setConversationMessages(conversationId, (prev) => [
        ...prev,
        t('chatSystem.learning.modeOn', {minutes: Math.round(learningIdleDelayMs / 60000)}),
      ]);
    }
  }

  // v3.7: VL 面板的示範錄製切換（原 v3.6 toggleLearning 的錄製部分，已與學習模式脫鉤）
  async function toggleVisualLearningRecording() {
    const conversationId = activeConversationIdRef.current || 'main';
    const backendActive = await callWails(IsLearningModeActive).catch(() => vlLearningActive);
    if (vlLearningActive || backendActive) {
      // 停止示範錄製
      try {
        if (backendActive) {
          const traceId = makeDebugTraceID('learning-metadata');
          const run = await callWails(StopLearningMode);
          const namedRun = await enrichStoppedLearningRun(run, traceId);
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            formatLearningOperationLearned(namedRun),
          ]);
        }
      } catch (error) {
        const detail = error?.message || String(error || '');
        if (!isNoActiveLearningRecordingError(detail)) {
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            t('chatSystem.learning.recordingStopFailed', {error: detail}),
          ]);
        }
      }
      setVlLearningActive(false);
      setVlActiveLearningRun(null);
    } else {
      // 開啟示範錄製（需要 activeWindowHash，目前用 session ID）
      try {
        const permissionStatus = await callWails(RequestVisualLearningPermissions).catch(() => null);
        if (permissionStatus?.missing?.length) {
          const opened = await openFirstMissingVisualLearningPermission(permissionStatus);
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            formatVisualLearningPermissionStatus(permissionStatus, true) + (opened ? `\n${t('chatSystem.learning.permissionSettingsOpened')}` : ''),
          ]);
        }
        const run = await callWails(() => StartLearningMode('window-' + Date.now()));
        setVlActiveLearningRun(run);
        setVlLearningActive(true);
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          t('chatSystem.learning.recordingStartedTarget', {run: run?.id ? `Run: ${run.id}. ` : ''}),
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
            t('chatSystem.learning.recordingResumedToggle', {run: activeRun?.id ? `Run: ${activeRun.id}. ` : ''}),
          ]);
          return;
        }
        setConversationMessages(conversationId, (prev) => [
          ...prev,
          t('chatSystem.learning.recordingBackendInactive', {error: detail}),
        ]);
      }
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
  const floatingPersona = activePersona || mainPersona;
  const floatingAvatarConfig = avatarConfigs[floatingPersona.id] || null;
  const floatingRenderedPixelAvatar = renderedPixelAvatars[floatingPersona.id] || '';
  const floatingAvatarSrc = resolvePersonaAvatarSrc(floatingPersona, floatingAvatarConfig, staticAvatarPreviews, floatingRenderedPixelAvatar);
  const activeFloatingAgentId = activeConversationId === 'main'
    ? floatingPersona.id || 'main'
    : activeConversationId || floatingPersona.id || 'main';
  const floatingAvatarDraft = floatingAvatarDrafts[activeFloatingAgentId] || '';
  const floatingAvatarChatOwnerRef = useRef(activeFloatingAgentId);
  useEffect(() => {
    if (floatingAvatarChatOwnerRef.current && floatingAvatarChatOwnerRef.current !== activeFloatingAgentId) {
      setFloatingAvatarChatModeEnabled(false, {clearHistory: true, silent: true});
    }
    floatingAvatarChatOwnerRef.current = activeFloatingAgentId;
  }, [activeFloatingAgentId]);
  const floatingReminderPaused = floatingReminderPause.mode === 'manual'
    || Number(floatingReminderPause.until || 0) > (avatarClock || Date.now());
  const floatingReminderLabel = floatingReminderPause.mode === 'manual'
    ? t('floatingAvatar.reminderManualActive')
    : floatingReminderPaused
      ? t('floatingAvatar.reminderUntil', {time: new Date(Number(floatingReminderPause.until)).toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'})})
      : '';
  const floatingAvatarPersonas = settingsState.personas.map((persona) => ({
    ...persona,
    avatarSrc: resolvePersonaAvatarSrc(
      persona,
      avatarConfigs[persona.id],
      staticAvatarPreviews,
      renderedPixelAvatars[persona.id] || '',
    ),
  }));

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

  const panelTheme = panelStyleTheme(settingsState.panel.panelStyle);

  /* i18n: apply language + direction attributes to root shell */
  const _i18nLang = useI18n(s => s.language);
  const _i18nDir  = useI18n(s => s.getDirection)();

  useEffect(() => {
    setState((prev) => {
      if (manualGreetingLockedRef.current) return prev;
      const localizedStatusRail = localizeStatusRailView(prev.statusRail);
      return {
        ...prev,
        greeting: localizedStatusRail?.text || localizeStatusRailText(prev.greeting),
        statusRail: localizedStatusRail || prev.statusRail,
      };
    });
    setSettingsState((prev) => normalizeSettingsState(prev));
  }, [_i18nLang]);

  useEffect(() => {
    if (activePersona?.name) setPersonaName(activePersona.name);
    if (activePersona?.identity) setPersonaJob(activePersona.identity);
  }, [activePersona?.id, activePersona?.name, activePersona?.identity]);

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

  async function enterFloatingAvatarMode() {
    if (floatingAvatarTransitionRef.current) return;
    if (!ENABLE_NATIVE_FLOATING_AVATAR_WINDOW) {
      floatingAvatarTransitionRef.current = true;
      try {
        const [windowSize, windowPosition] = await Promise.all([
          WindowGetSize().catch(() => ({w: window.innerWidth, h: window.innerHeight})),
          WindowGetPosition().catch(() => ({x: 0, y: 0})),
        ]);
        const avatarPosition = floatingAvatarCenterPosition();
        const overlayX = Math.round((windowPosition?.x ?? 0) + Number(avatarPosition?.x || 0));
        const overlayY = Math.round((windowPosition?.y ?? 0) + Number(avatarPosition?.y || 0));
        floatingAvatarPositionRef.current = avatarPosition;
        setFloatingAvatarPosition(avatarPosition);
        setFloatingAvatarBodyMode('head');
        await showFloatingAvatarOverlayAt('head', overlayX, overlayY);
        floatingAvatarCompactWindowRef.current = false;
        setFloatingAvatarCompactWindow(false);
        setFloatingAvatarSurfaceOnly(false);
        setFloatingAvatarFlyingBack(false);
        floatingAvatarDragWindowRef.current = null;
        floatingAvatarWindowRef.current = {
          restore: {
            size: windowSize || {w: window.innerWidth, h: window.innerHeight},
            position: windowPosition || {x: 0, y: 0},
            avatarPosition,
          },
          compactPosition: null,
        };
        floatingAvatarModeRef.current = true;
        setFloatingAvatarMode(true);
        setSchedulerBgPrompt(false);
        setManualAvatarState('happy');
        WindowHide();
        setToolResult({toolId: 'scheduler', ok: true, message: t('floatingAvatar.entered')});
      } catch (error) {
        console.warn('floating avatar overlay failed', error);
        setToolResult({toolId: 'scheduler', ok: false, message: t('floatingAvatar.backgroundFail', {error: error?.message || error})});
      } finally {
        floatingAvatarTransitionRef.current = false;
      }
      return;
    }
    floatingAvatarTransitionRef.current = true;
    try {
      const [windowSize, windowPosition] = await Promise.all([
        WindowGetSize(),
        WindowGetPosition(),
      ]);
      const avatarPosition = floatingAvatarCenterPosition();
      const avatarX = Number(avatarPosition.x);
      const avatarY = Number(avatarPosition.y);
      const compactPosition = compactWindowPositionFromAvatarAnchor({
        x: Math.round((windowPosition?.x ?? 0) + avatarX),
        y: Math.round((windowPosition?.y ?? 0) + avatarY),
      });
      const compactX = compactPosition.x;
      const compactY = compactPosition.y;
      floatingAvatarWindowRef.current = {
        restore: {
          size: windowSize || {w: window.innerWidth, h: window.innerHeight},
          position: windowPosition || {x: 0, y: 0},
          avatarPosition: {x: avatarX, y: avatarY},
        },
        compactPosition: {x: compactX, y: compactY},
      };
      rememberFloatingAvatarCompactPosition({x: compactX, y: compactY});
      WindowUnminimise();
      WindowShow();
      WindowSetMinSize(FLOATING_AVATAR_WINDOW_SIZE, FLOATING_AVATAR_WINDOW_SIZE);
      WindowSetBackgroundColour(0, 0, 0, 0);
      WindowSetAlwaysOnTop(true);
      WindowSetSize(FLOATING_AVATAR_WINDOW_SIZE, FLOATING_AVATAR_WINDOW_SIZE);
      WindowSetPosition(compactX, compactY);
      // 切成原生無框透明置頂浮窗（macOS 真實作；其他平台 no-op）。
      await callWails(() => EnterFloatingAvatarNative()).catch((e) => console.warn('EnterFloatingAvatarNative failed', e));
      const compactAvatarPosition = {x: FLOATING_AVATAR_INSET, y: FLOATING_AVATAR_INSET};
      floatingAvatarPositionRef.current = compactAvatarPosition;
      setFloatingAvatarPosition(compactAvatarPosition);
      floatingAvatarCompactWindowRef.current = true;
      setFloatingAvatarCompactWindow(true);
      setFloatingAvatarSurfaceOnly(false);
    } catch (error) {
      console.warn('floating avatar compact window failed', error);
      floatingAvatarCompactWindowRef.current = false;
      setFloatingAvatarCompactWindow(false);
      setFloatingAvatarSurfaceOnly(false);
    } finally {
      floatingAvatarTransitionRef.current = false;
    }
    floatingAvatarModeRef.current = true;
    setFloatingAvatarMode(true);
    setSchedulerBgPrompt(false);
    setManualAvatarState('happy');
    setToolResult({toolId: 'scheduler', ok: true, message: t('floatingAvatar.entered')});
  }

  async function openFloatingAvatarPanelFromOverlay() {
    if (floatingAvatarTransitionRef.current) return;
    floatingAvatarTransitionRef.current = true;
    try {
      const overlayPosition = await callWails(() => GetFloatingAvatarOverlayPosition()).catch(() => null);
      const overlayFallback = clampFloatingAvatarScreenPosition({
        x: (firstFiniteNumber(window.screen?.availLeft, window.screen?.left, 0) || 0) + ((firstFiniteNumber(window.screen?.availWidth, window.screen?.width, window.innerWidth) || FLOATING_AVATAR_WINDOW_SIZE) - FLOATING_AVATAR_WINDOW_SIZE) / 2,
        y: (firstFiniteNumber(window.screen?.availTop, window.screen?.top, 0) || 0) + ((firstFiniteNumber(window.screen?.availHeight, window.screen?.height, window.innerHeight) || FLOATING_AVATAR_WINDOW_SIZE) - FLOATING_AVATAR_WINDOW_SIZE) / 2,
      });
      const avatarScreenX = Number.isFinite(overlayPosition?.x) ? Math.round(overlayPosition.x) : overlayFallback.x;
      const avatarScreenY = Number.isFinite(overlayPosition?.y) ? Math.round(overlayPosition.y) : overlayFallback.y;
      await callWails(() => ExitFloatingAvatarOverlay()).catch((e) => console.warn('ExitFloatingAvatarOverlay failed', e));

      const screens = await ScreenGetAll().catch(() => []);
      const currentScreen = (screens || []).find((item) => item?.isCurrent) || (screens || []).find((item) => item?.isPrimary) || null;
      const screenWidth = firstFiniteNumber(window.screen?.availWidth, currentScreen?.width, window.innerWidth) || FLOATING_AVATAR_PANEL_W;
      const screenHeight = firstFiniteNumber(window.screen?.availHeight, currentScreen?.height, window.innerHeight) || FLOATING_AVATAR_PANEL_H;
      const panelW = Math.min(FLOATING_AVATAR_PANEL_W, screenWidth);
      const panelH = Math.min(FLOATING_AVATAR_PANEL_H, screenHeight);
      let panelX = avatarScreenX - FLOATING_AVATAR_LEFT_TIP_ROOM;
      let panelY = avatarScreenY - 118;
      const knownBounds = currentFloatingAvatarScreenBounds(panelW, panelH);
      if (pointInsideFloatingAvatarScreen({x: avatarScreenX, y: avatarScreenY}, knownBounds)) {
        const minX = knownBounds.left;
        const maxX = knownBounds.right - panelW;
        const minY = knownBounds.top;
        const maxY = knownBounds.bottom - panelH;
        panelX = Math.max(Math.min(panelX, Math.max(minX, maxX)), Math.min(minX, maxX));
        panelY = Math.max(Math.min(panelY, Math.max(minY, maxY)), Math.min(minY, maxY));
      }

      const avatarPosition = {x: Math.round(avatarScreenX - panelX), y: Math.round(avatarScreenY - panelY)};
      floatingAvatarWindowRef.current = {
        ...floatingAvatarWindowRef.current,
        compactPosition: {x: Math.round(panelX), y: Math.round(panelY)},
        expanded: {avatarScreenX, avatarScreenY},
      };
      floatingAvatarPositionRef.current = avatarPosition;
      setFloatingAvatarPosition(avatarPosition);
      floatingAvatarCompactWindowRef.current = true;
      setFloatingAvatarCompactWindow(true);
      setFloatingAvatarSurfaceOnly(true);
      WindowSetMinSize(320, 320);
      WindowSetSize(panelW, panelH);
      WindowSetPosition(Math.round(panelX), Math.round(panelY));
      WindowSetAlwaysOnTop(true);
      WindowSetBackgroundColour(0, 0, 0, 0);
      WindowUnminimise();
      WindowShow();
      floatingAvatarModeRef.current = true;
      setFloatingAvatarMode(true);
      setFloatingAvatarPanelOpenSignal((value) => value + 1);
    } catch (error) {
      console.warn('floating avatar panel open failed', error);
      setToolResult({toolId: 'floating-avatar', ok: false, message: t('floatingAvatar.backgroundFail', {error: error?.message || error})});
    } finally {
      floatingAvatarTransitionRef.current = false;
    }
  }

  async function restoreFloatingAvatarWindow() {
    if (floatingAvatarTransitionRef.current) return;
    if (!ENABLE_NATIVE_FLOATING_AVATAR_WINDOW && floatingAvatarModeRef.current) {
      const restore = floatingAvatarWindowRef.current?.restore;
      await callWails(() => ExitFloatingAvatarOverlay()).catch((e) => console.warn('ExitFloatingAvatarOverlay failed', e));
      WindowUnminimise();
      WindowShow();
      WindowSetAlwaysOnTop(false);
      WindowSetMinSize(MAIN_WINDOW_MIN_SIZE.width, MAIN_WINDOW_MIN_SIZE.height);
      WindowSetBackgroundColour(5, 5, 5, 255);
      if (restore?.size) {
        WindowSetSize(restore.size.w || restore.size.width || 1536, restore.size.h || restore.size.height || 860);
      }
      if (restore?.position) {
        WindowSetPosition(restore.position.x || 0, restore.position.y || 0);
      }
      floatingAvatarCompactWindowRef.current = false;
      setFloatingAvatarCompactWindow(false);
      setFloatingAvatarSurfaceOnly(false);
      floatingAvatarWindowRef.current = {restore: null, compactPosition: null};
      return;
    }
    const restore = floatingAvatarWindowRef.current?.restore;
    const wasCompact = floatingAvatarCompactWindowRef.current || Boolean(restore);
    if (!wasCompact) return;
    floatingAvatarTransitionRef.current = true;
    const expandedState = floatingAvatarWindowRef.current?.expanded;
    const compactLivePosition = await WindowGetPosition().catch(() => null);
    if (!expandedState && Number.isFinite(compactLivePosition?.x) && Number.isFinite(compactLivePosition?.y)) {
      const compactSafePosition = compactWindowPositionFromAvatarAnchor({
        x: Number(compactLivePosition.x) + FLOATING_AVATAR_INSET,
        y: Number(compactLivePosition.y) + FLOATING_AVATAR_INSET,
      });
      rememberFloatingAvatarCompactPosition(compactSafePosition);
      floatingAvatarWindowRef.current = {
        ...floatingAvatarWindowRef.current,
        compactPosition: compactSafePosition,
      };
    } else if (Number.isFinite(floatingAvatarWindowRef.current?.compactPosition?.x) && Number.isFinite(floatingAvatarWindowRef.current?.compactPosition?.y)) {
      rememberFloatingAvatarCompactPosition(floatingAvatarWindowRef.current.compactPosition);
    }
    // 先在頭像浮窗內播放「飛回」動畫，再真正放大視窗，避免縮放當下硬切。
    setFloatingAvatarFlyingBack(true);
    await new Promise((resolve) => window.setTimeout(resolve, 240));
    try {
      // 還原一般有框、不透明視窗。
      await callWails(() => ExitFloatingAvatarNative()).catch((e) => console.warn('ExitFloatingAvatarNative failed', e));
      WindowUnminimise();
      WindowShow();
      WindowSetAlwaysOnTop(false);
      WindowSetBackgroundColour(5, 5, 5, 255);
      WindowSetMinSize(MAIN_WINDOW_MIN_SIZE.width, MAIN_WINDOW_MIN_SIZE.height);
      if (restore?.size) {
        WindowSetSize(restore.size.w || restore.size.width || 1536, restore.size.h || restore.size.height || 860);
      }
      if (restore?.position) {
        WindowSetPosition(restore.position.x || 0, restore.position.y || 0);
      }
    } catch (error) {
      console.warn('floating avatar restore window failed', error);
    } finally {
      floatingAvatarTransitionRef.current = false;
    }
    floatingAvatarCompactWindowRef.current = false;
    setFloatingAvatarCompactWindow(false);
    setFloatingAvatarFlyingBack(false);
    floatingAvatarDragWindowRef.current = null;
    floatingAvatarWindowRef.current = {restore: null, compactPosition: null};
    const restoredAvatarPosition = normalizeFloatingAvatarViewportPosition(
      restore?.avatarPosition || floatingAvatarCenterPosition()
    );
    floatingAvatarPositionRef.current = restoredAvatarPosition;
    setFloatingAvatarPosition(restoredAvatarPosition);
  }

  async function restoreFromFloatingAvatar(target = 'auto') {
    await restoreFloatingAvatarWindow();
    floatingAvatarModeRef.current = false;
    setFloatingAvatarMode(false);
    setFloatingAvatarSurfaceOnly(false);
    setManualAvatarState('');
    if (target === 'settings') {
      setActivePanel('settings');
    } else {
      // 自動定位：待確認優先 → 否則最近任務。
      // 先回到主視圖（關掉設定等面板），待確認卡片本身有 effect 會 scrollIntoView，
      // 沒有待確認時則確保最近任務/對話在主視圖可見。
      setActivePanel(null);
      if (pendingTaskReview?.id) {
        setReviewPopup('risk');
      }
    }
    setToolResult({
      toolId: 'scheduler',
      ok: true,
      message: pendingTaskReview?.id ? t('floatingAvatar.restoredPending') : t('floatingAvatar.restoredLatest'),
    });
  }

  async function closeAfterBackgroundAvatarExit() {
    await setFloatingAvatarChatModeEnabled(false, {clearHistory: true, silent: true});
    await callWails(() => ExitFloatingAvatarOverlay()).catch((error) => {
      console.warn('ExitFloatingAvatarOverlay failed', error);
    });
    floatingAvatarModeRef.current = false;
    setFloatingAvatarMode(false);
    setManualAvatarState('');
    try {
      await callWails(() => ResolveSchedulerBackgroundPrompt(false));
    } catch {
      // 停用背景喚醒失敗時仍嘗試完整關閉，避免留下殘留浮窗/圖示。
    }
    await callWails(() => ConfirmClose(false, ''));
    Quit();
  }

  // 單擊頭像展開迷你框時，把浮窗放大到面板尺寸；關閉時縮回頭像尺寸。
  // 視窗會貼著頭像展開，並自動避開螢幕右/下邊緣，頭像螢幕位置維持不變。
  async function setCompactAvatarExpanded(open, detail = {}) {
    if (!floatingAvatarCompactWindowRef.current) return;
    const ref = floatingAvatarWindowRef.current;
    if (!ref) return;
    if (!ENABLE_NATIVE_FLOATING_AVATAR_WINDOW && !open) {
      if (!ref.expanded) return;
      const {avatarScreenX, avatarScreenY} = ref.expanded;
      ref.expanded = null;
      ref.compactPosition = null;
      floatingAvatarCompactWindowRef.current = false;
      setFloatingAvatarCompactWindow(false);
      setFloatingAvatarSurfaceOnly(false);
      WindowHide();
      await showFloatingAvatarOverlayAt(floatingAvatarBodyMode, avatarScreenX, avatarScreenY).catch((error) => {
        console.warn('restore floating avatar overlay failed', error);
      });
      return;
    }
    try {
      if (open) {
        const bodyViewport = (detail?.body || floatingAvatarBodyMode === 'full') && !detail?.panel;
        let avatarScreenX = ref.expanded?.avatarScreenX;
        let avatarScreenY = ref.expanded?.avatarScreenY;
        if (!Number.isFinite(avatarScreenX) || !Number.isFinite(avatarScreenY)) {
          // compact 模式可由原生 draggable 移動，展開前先同步真實視窗位置。
          const livePosition = await WindowGetPosition().catch(() => null);
          const origin = roundFloatingAvatarScreenPosition(
            Number.isFinite(livePosition?.x) && Number.isFinite(livePosition?.y)
              ? livePosition
              : ref.compactPosition || {x: 0, y: 0}
          );
          ref.compactPosition = origin;
          rememberFloatingAvatarCompactPosition(origin);
          avatarScreenX = origin.x + FLOATING_AVATAR_INSET;
          avatarScreenY = origin.y + FLOATING_AVATAR_INSET;
        }
        const screens = await ScreenGetAll().catch(() => []);
        const currentScreen = (screens || []).find((item) => item?.isCurrent) || (screens || []).find((item) => item?.isPrimary) || null;
        // 視窗展開後保留上方回覆泡泡、左側提示、右側迷你框與下方選項，
        // 並讓頭像螢幕位置在展開前後維持不變（不會跳）。
        const TOP_ROOM = 118;
        const screenWidth = firstFiniteNumber(window.screen?.availWidth, currentScreen?.width, window.innerWidth) || FLOATING_AVATAR_PANEL_W;
        const screenHeight = firstFiniteNumber(window.screen?.availHeight, currentScreen?.height, window.innerHeight) || FLOATING_AVATAR_PANEL_H;
        const expandedW = Math.min(bodyViewport ? FLOATING_AVATAR_BODY_WINDOW_W : FLOATING_AVATAR_PANEL_W, screenWidth);
        const expandedH = Math.min(bodyViewport ? FLOATING_AVATAR_BODY_WINDOW_H : FLOATING_AVATAR_PANEL_H, screenHeight);
        const preferredLocalX = bodyViewport ? FLOATING_AVATAR_BODY_LOCAL_X : FLOATING_AVATAR_LEFT_TIP_ROOM;
        const preferredLocalY = bodyViewport ? FLOATING_AVATAR_BODY_LOCAL_Y : TOP_ROOM;
        let newX = avatarScreenX - preferredLocalX;
        let newY = avatarScreenY - preferredLocalY;
        const knownBounds = currentFloatingAvatarScreenBounds(expandedW, expandedH);
        const anchorPoint = {x: avatarScreenX, y: avatarScreenY};
        if (pointInsideFloatingAvatarScreen(anchorPoint, knownBounds)) {
          const minX = knownBounds.left;
          const maxX = knownBounds.right - expandedW;
          const minY = knownBounds.top;
          const maxY = knownBounds.bottom - expandedH;
          newX = Math.max(Math.min(newX, Math.max(minX, maxX)), Math.min(minX, maxX));
          newY = Math.max(Math.min(newY, Math.max(minY, maxY)), Math.min(minY, maxY));
        }
        const avatarLocalX = bodyViewport ? FLOATING_AVATAR_BODY_LOCAL_X : avatarScreenX - newX;
        const avatarLocalY = bodyViewport ? FLOATING_AVATAR_BODY_LOCAL_Y : avatarScreenY - newY;
        ref.expanded = {
          avatarScreenX: Math.round(newX + avatarLocalX),
          avatarScreenY: Math.round(newY + avatarLocalY),
        };
        WindowSetSize(expandedW, expandedH);
        WindowSetPosition(Math.round(newX), Math.round(newY));
        const expandedAvatarPosition = {x: Math.round(avatarLocalX), y: Math.round(avatarLocalY)};
        floatingAvatarPositionRef.current = expandedAvatarPosition;
        setFloatingAvatarPosition(expandedAvatarPosition);
      } else {
        if (!ref.expanded) return;
        const {avatarScreenX, avatarScreenY} = ref.expanded;
        const nextCompact = compactWindowPositionFromAvatarAnchor({x: avatarScreenX, y: avatarScreenY});
        WindowSetSize(FLOATING_AVATAR_WINDOW_SIZE, FLOATING_AVATAR_WINDOW_SIZE);
        WindowSetPosition(nextCompact.x, nextCompact.y);
        ref.compactPosition = nextCompact;
        ref.expanded = null;
        const compactAvatarPosition = {x: FLOATING_AVATAR_INSET, y: FLOATING_AVATAR_INSET};
        floatingAvatarPositionRef.current = compactAvatarPosition;
        setFloatingAvatarPosition(compactAvatarPosition);
      }
    } catch (e) {
      console.warn('setCompactAvatarExpanded failed', e);
    }
  }

  function updateFloatingAvatarPosition(nextPosition) {
    floatingAvatarPositionRef.current = nextPosition;
    setFloatingAvatarPosition(nextPosition);
  }

  function rememberFloatingAvatarCompactPosition(nextPosition) {
    if (!Number.isFinite(nextPosition?.x) || !Number.isFinite(nextPosition?.y)) return;
    floatingAvatarLastCompactPositionRef.current = {
      x: Math.round(nextPosition.x),
      y: Math.round(nextPosition.y),
    };
  }

  function beginCompactFloatingAvatarDrag() {
    if (floatingAvatarDragFrameRef.current) {
      window.cancelAnimationFrame(floatingAvatarDragFrameRef.current);
      floatingAvatarDragFrameRef.current = 0;
    }
    floatingAvatarPendingWindowPositionRef.current = null;
    const ref = floatingAvatarWindowRef.current || {};
    const avatarLocal = floatingAvatarPositionRef.current || {};
    if (ref.expanded && Number.isFinite(ref.expanded.avatarScreenX) && Number.isFinite(ref.expanded.avatarScreenY)) {
      const avatarLocalX = Number.isFinite(avatarLocal.x) ? Number(avatarLocal.x) : FLOATING_AVATAR_INSET;
      const avatarLocalY = Number.isFinite(avatarLocal.y) ? Number(avatarLocal.y) : FLOATING_AVATAR_INSET;
      floatingAvatarDragWindowRef.current = {
        x: Math.round(ref.expanded.avatarScreenX - avatarLocalX),
        y: Math.round(ref.expanded.avatarScreenY - avatarLocalY),
        expanded: true,
        avatarLocalX,
        avatarLocalY,
      };
      return;
    }
    floatingAvatarDragWindowRef.current = roundFloatingAvatarScreenPosition(ref.compactPosition || null);
  }

  function flushCompactFloatingAvatarDragPosition() {
    const pending = floatingAvatarPendingWindowPositionRef.current;
    if (!pending) return;
    floatingAvatarPendingWindowPositionRef.current = null;
    WindowSetPosition(pending.x, pending.y);
  }

  function moveCompactFloatingAvatarWindow(dx, dy) {
    const start = floatingAvatarDragWindowRef.current;
    if (!start) return;
    const rawWindow = {
      x: Math.round(start.x + dx),
      y: Math.round(start.y + dy),
    };
    let next = rawWindow;
    if (start.expanded) {
      const bodyMode = floatingAvatarBodyModeRef.current === 'full';
      const avatarW = bodyMode ? FLOATING_AVATAR_BODY_W : FLOATING_AVATAR_SIZE;
      const avatarH = bodyMode ? FLOATING_AVATAR_BODY_H : FLOATING_AVATAR_SIZE;
      const anchor = clampFloatingAvatarAnchorPosition({
        x: rawWindow.x + start.avatarLocalX,
        y: rawWindow.y + start.avatarLocalY,
      }, avatarW, avatarH);
      next = {
        x: Math.round(anchor.x - start.avatarLocalX),
        y: Math.round(anchor.y - start.avatarLocalY),
      };
      floatingAvatarWindowRef.current = {
        ...floatingAvatarWindowRef.current,
        expanded: {
          avatarScreenX: Math.round(next.x + start.avatarLocalX),
          avatarScreenY: Math.round(next.y + start.avatarLocalY),
        },
      };
    } else {
      next = compactWindowPositionFromAvatarAnchor({
        x: rawWindow.x + FLOATING_AVATAR_INSET,
        y: rawWindow.y + FLOATING_AVATAR_INSET,
      });
      floatingAvatarWindowRef.current = {
        ...floatingAvatarWindowRef.current,
        compactPosition: next,
      };
      rememberFloatingAvatarCompactPosition(next);
    }
    floatingAvatarPendingWindowPositionRef.current = next;
    if (!floatingAvatarDragFrameRef.current) {
      floatingAvatarDragFrameRef.current = window.requestAnimationFrame(() => {
        floatingAvatarDragFrameRef.current = 0;
        flushCompactFloatingAvatarDragPosition();
      });
    }
  }

  async function syncCompactFloatingAvatarWindowPosition() {
    if (floatingAvatarDragFrameRef.current) {
      window.cancelAnimationFrame(floatingAvatarDragFrameRef.current);
      floatingAvatarDragFrameRef.current = 0;
    }
    flushCompactFloatingAvatarDragPosition();
    const livePosition = await WindowGetPosition().catch(() => null);
    if (!Number.isFinite(livePosition?.x) || !Number.isFinite(livePosition?.y)) return;
    const ref = floatingAvatarWindowRef.current || {};
    if (ref.expanded) {
      const avatarLocal = floatingAvatarPositionRef.current || {};
      const avatarLocalX = Number.isFinite(avatarLocal.x) ? Number(avatarLocal.x) : FLOATING_AVATAR_INSET;
      const avatarLocalY = Number.isFinite(avatarLocal.y) ? Number(avatarLocal.y) : FLOATING_AVATAR_INSET;
      floatingAvatarWindowRef.current = {
        ...ref,
        expanded: {
          avatarScreenX: Math.round(livePosition.x + avatarLocalX),
          avatarScreenY: Math.round(livePosition.y + avatarLocalY),
        },
      };
      return;
    }
    const next = compactWindowPositionFromAvatarAnchor({
      x: Number(livePosition.x) + FLOATING_AVATAR_INSET,
      y: Number(livePosition.y) + FLOATING_AVATAR_INSET,
    });
    floatingAvatarWindowRef.current = {...ref, compactPosition: next};
    rememberFloatingAvatarCompactPosition(next);
  }

  function updateFloatingAvatarDraft(value) {
    setFloatingAvatarDrafts((prev) => ({...prev, [activeFloatingAgentId]: value}));
  }

  function syncFloatingAvatarOverlayMetadata(replyText = floatingAvatarReplyBubble) {
    const personaName = floatingPersona?.name || t('floatingAvatar.agentFallback');
    const placeholder = t('floatingAvatar.chatInputPlaceholder');
    callWails(() => SetFloatingAvatarOverlayMetadata(personaName, replyText || '', placeholder)).catch(() => {});
  }

  function syncFloatingAvatarOverlayChatMode(enabled) {
    callWails(() => SetFloatingAvatarOverlayChatMode(Boolean(enabled))).catch(() => {});
    syncFloatingAvatarOverlayMetadata(enabled ? floatingAvatarReplyBubble : '');
  }

  function clearFloatingAvatarChatMemory() {
    floatingAvatarChatHistoryRef.current = [];
    floatingAvatarChatRequestSeqRef.current += 1;
  }

  async function setFloatingAvatarChatModeEnabled(enabled, options = {}) {
    const next = Boolean(enabled);
    floatingAvatarChatModeRef.current = next;
    setFloatingAvatarChatMode(next);
    syncFloatingAvatarOverlayChatMode(next);
    if (!next || options.clearHistory) {
      clearFloatingAvatarChatMemory();
      setFloatingAvatarReplyBubble('');
      syncFloatingAvatarOverlayMetadata('');
    }
    if (!options.silent) {
      setToolResult({
        toolId: 'floating-avatar-chat',
        ok: true,
        message: next ? t('floatingAvatar.chatStarted') : t('floatingAvatar.chatClosed'),
      });
    }
  }

  async function toggleFloatingAvatarChatMode(enabled, options = {}) {
    if (!enabled) {
      await setFloatingAvatarChatModeEnabled(false, {clearHistory: true});
      return;
    }
    if (options.prompt === false) {
      await setFloatingAvatarChatModeEnabled(true);
      return;
    }
    await promptFloatingAvatarChatInput();
  }

  function floatingAvatarThinkingMessageForPersona(persona) {
    const id = String(persona?.id || '').toLowerCase();
    const name = String(persona?.name || '').trim().toLowerCase();
    const identity = String(persona?.identity || '').trim().toLowerCase();
    const haystack = [id, name, identity].join(' ');
    if (id === 'persona-a' || haystack.includes('憂樂傻酷') || haystack.includes('yurosaku')) {
      return '本犬要動一下腦袋，稍等！';
    }
    if (id === 'persona-b' || haystack.includes('厭世叔') || haystack.includes('厭世大叔') || haystack.includes('grumpy uncle')) {
      return '啊啊，我想一下，先別繼續問.....。';
    }
    if (id === 'persona-c' || haystack.includes('秘書小妹') || haystack.includes('秘書小姐') || haystack.includes('assistant') || haystack.includes('secretary')) {
      return '好的，請給我點時間，馬上處理中。';
    }
    if (id === 'persona-d' || haystack.includes('警察') || haystack.includes('rossfork') || haystack.includes('police')) {
      return '身為警察，任何事都會盡心盡力服務的，請稍等。';
    }
    if (id === 'persona-e' || haystack.includes('東春') || haystack.includes('touharu') || haystack.includes('miko')) {
      return '真是拿你沒辦法，竟然還要我說？';
    }
    return '想一下…';
  }

  async function submitFloatingAvatarChatText(rawText) {
    const text = String(rawText || '').trim();
    if (!text) return;
    updateFloatingAvatarDraft('');
    setSelectedFloatingCandidateIDs([]);
    await setFloatingAvatarChatModeEnabled(true, {silent: true});
    const requestSeq = floatingAvatarChatRequestSeqRef.current + 1;
    floatingAvatarChatRequestSeqRef.current = requestSeq;
    const waitingText = floatingAvatarThinkingMessageForPersona(floatingPersona);
    setFloatingAvatarReplyBubble(waitingText);
    syncFloatingAvatarOverlayMetadata(waitingText);
    const resp = await sendFloatingAvatarQuickChat(text);
    if (requestSeq !== floatingAvatarChatRequestSeqRef.current || !floatingAvatarChatModeRef.current) {
      return;
    }
    if (resp?.error) {
      setToolResult({toolId: 'floating-avatar-chat', ok: false, message: resp.error});
      setManualAvatarState('sad');
    } else if (resp?.text) {
      floatingAvatarChatHistoryRef.current = [
        ...floatingAvatarChatHistoryRef.current,
        {role: 'user', text},
        {role: 'assistant', text: resp.text},
      ].slice(-30);
      setFloatingAvatarReplyBubble(resp.text);
      syncFloatingAvatarOverlayMetadata(resp.text);
      setManualAvatarState('happy');
    }
  }

  function buildFloatingAvatarQuickChatPrompt(userText, history) {
    const personaName = floatingPersona?.name || t('floatingAvatar.agentFallback');
    const personality = String(floatingPersona?.personality || '').trim();
    // 提速：prompt 只帶最近 8 條（暫存仍留 30 條），token 少、回覆快。
    const recent = (Array.isArray(history) ? history : [])
      .slice(-8)
      .map((entry) => `${entry.role === 'assistant' ? personaName : '主人'}：${entry.text}`)
      .join('\n');
    return [
      // 後端看到這個 sentinel 會直達模型，跳過意圖路由（否則慢、且會被誤判成搜尋）。
      '[[AI_CONSOLE_QUICK_CHAT]]',
      // RAW 行讓後端判斷唯一的工具特例：明講「網路搜尋/上網查 …」才觸發網搜。
      `[[RAW]]${String(userText).replace(/\s+/g, ' ').trim()}`,
      `你現在是「${personaName}」的閒聊模式。`,
      personality ? `人格語氣：${personality}` : '',
      '只做輕量閒聊回覆，不規劃任務、不呼叫工具、不要求使用者等待主控台流程。',
      '回覆請簡短自然，最多 3 句；若適合可用繁體中文。',
      '一律純文字聊天，不要使用 Markdown、標題、清單、表格、JSON 或程式碼區塊。',
      recent ? `最近暫存對話：\n${recent}` : '',
      `主人：${userText}`,
      `${personaName}：`,
    ].filter(Boolean).join('\n');
  }

  async function sendFloatingAvatarQuickChat(userText) {
    const adapter = resolveAdapterFromRefs();
    const adapterID = adapter?.id || adapter?.adapter_id || activeAdapterIdRef.current || '';
    if (!adapterID) {
      return {error: '找不到可用 adapter。'};
    }
    const sessionId = appSessionIdRef.current || '';
    const traceId = makeDebugTraceID('avatar-chat');
    const prompt = buildFloatingAvatarQuickChatPrompt(userText, floatingAvatarChatHistoryRef.current);
    const escaped = await callWails(() => EscapeExternalTokens(prompt)).catch(() => prompt);
    try {
      const resp = await callWails(() => (
        isAPIAdapter(adapter)
          ? SendAPIMessage(adapterID, sessionId, escaped || prompt, traceId)
          : SendCLIMessage(adapterID, sessionId, escaped || prompt, traceId)
      ));
      const cliResp = normalizeCLIResponse(resp);
      return cliResp;
    } catch (error) {
      return {error: sanitizeDisplayedCLIError(error?.message || String(error))};
    }
  }

  async function promptFloatingAvatarChatInput() {
    await setFloatingAvatarChatModeEnabled(true, {silent: true});
    const value = window.prompt(t('floatingAvatar.chatPrompt'), '');
    if (value === null) {
      await setFloatingAvatarChatModeEnabled(false, {clearHistory: true});
      return;
    }
    await submitFloatingAvatarChatText(value);
  }

  function setFloatingReminderMode(mode) {
    const now = Date.now();
    if (mode === 'resume') {
      setFloatingReminderPause({mode: '', until: 0});
      setToolResult({toolId: 'floating-avatar', ok: true, message: t('floatingAvatar.reminderResumed')});
      return;
    }
    let until = 0;
    if (mode === '30m') until = now + 30 * 60 * 1000;
    if (mode === '1h') until = now + 60 * 60 * 1000;
    if (mode === 'tomorrow') {
      const tomorrow = new Date(now);
      tomorrow.setDate(tomorrow.getDate() + 1);
      tomorrow.setHours(8, 0, 0, 0);
      until = tomorrow.getTime();
    }
    const next = mode === 'manual' ? {mode, until: 0} : {mode, until};
    setFloatingReminderPause(next);
    setToolResult({toolId: 'floating-avatar', ok: true, message: t('floatingAvatar.reminderPausedNotice')});
  }

  async function showFloatingAvatarOverlayAt(mode, x, y) {
    const overlayMode = mode === 'full' ? 'full' : 'head';
    const src = overlayMode === 'full'
      ? floatingFullBodyAvatarSrc
      : floatingAvatarSrc;
    if (!src) return;
    const imageData = await imageSrcToBytes(src);
    const overlayModePayload = overlayMode === 'full'
      ? `${overlayMode}:${floatingFullBodyAvatarKey || 'wolf'}`
      : overlayMode;
    await callWails(() => EnterFloatingAvatarOverlayImage(imageData, overlayModePayload, Math.round(x), Math.round(y)));
    syncFloatingAvatarOverlayMetadata();
  }

  async function repaintFloatingAvatarOverlay(mode = floatingAvatarBodyMode) {
    const position = await callWails(() => GetFloatingAvatarOverlayPosition()).catch(() => null);
    const x = Number.isFinite(position?.x) ? position.x : 0;
    const y = Number.isFinite(position?.y) ? position.y : 0;
    await showFloatingAvatarOverlayAt(mode, x, y);
  }

  async function changeFloatingAvatarBodyMode(mode) {
    const nextMode = mode === 'full' ? 'full' : 'head';
    setFloatingAvatarBodyMode(nextMode);
    if (nextMode === 'full') {
      // 全身像預設維持靜態（不自動勾動態）；使用者要動態自行勾選。
      // 仍需把 macOS 原生浮窗展開到足夠容納 200×360 立繪，否則高瘦身體會被 128px 方窗裁掉看似消失。
      if (ENABLE_NATIVE_FLOATING_AVATAR_WINDOW && floatingAvatarModeRef.current) {
        await setCompactAvatarExpanded(true, {body: true}).catch((error) => {
          console.warn('expand floating avatar for full body failed', error);
        });
        // 視窗尺寸變更後重套一次透明置頂參數（幂等），避免 WKWebView 合成殘留。
        await callWails(() => EnterFloatingAvatarNative()).catch(() => {});
        // WindowSetSize 與 WKWebView 重建 backing store 有時間差，
        // 立即重套可能落在 resize 完成前；晚一拍再重套一次（同樣幂等）。
        window.setTimeout(() => {
          if (floatingAvatarModeRef.current) {
            callWails(() => EnterFloatingAvatarNative()).catch(() => {});
          }
        }, 180);
      }
    }
    if (!ENABLE_NATIVE_FLOATING_AVATAR_WINDOW && floatingAvatarModeRef.current) {
      await repaintFloatingAvatarOverlay(nextMode).catch((error) => {
        console.warn('repaint floating avatar overlay failed', error);
      });
    }
  }

  async function changeFloatingAvatarDynamicImage(enabled) {
    const next = Boolean(enabled);
    setFloatingAvatarDynamicImage(next);
    if (next && floatingAvatarBodyMode !== 'full') {
      await changeFloatingAvatarBodyMode('full');
    }
    // macOS 原生浮窗：勾選動態後 Pixi(WebGL) 掛載會讓 WKWebView 重走合成路徑，
    // 重套一次透明視窗參數（幂等），避免整片視窗變成不透明黑塊。
    if (next && ENABLE_NATIVE_FLOATING_AVATAR_WINDOW && floatingAvatarModeRef.current) {
      await callWails(() => EnterFloatingAvatarNative()).catch(() => {});
    }
  }

  function changeFloatingAvatarMotionMode(mode) {
    const next = mode === 'frames' ? 'frames' : 'rig';
    setFloatingAvatarMotionMode(next);
    try { window.localStorage.setItem('floating_avatar_motion_mode', next); } catch {}
  }

  async function switchFloatingAvatarAgent(personaId) {
    const persona = settingsState.personas.find((item) => item.id === personaId);
    if (!persona) return;
    if (persona.id !== floatingPersona.id) {
      await setFloatingAvatarChatModeEnabled(false, {clearHistory: true, silent: true});
    }
    await savePersonaPatch(persona.id, persona);
    await loadCurrentAvatar(persona.id);
    setToolResult({toolId: persona.id, ok: true, message: t('floatingAvatar.agentSwitched', {name: persona.name || t('floatingAvatar.agentFallback')})});
  }

  async function submitFloatingAvatarText(rawText) {
    if (floatingAvatarChatModeRef.current) {
      await submitFloatingAvatarChatText(rawText);
      return;
    }
    const selectedText = selectedFloatingCandidateIDs
      .map((candidateID) => (readinessGate.floating_candidates || []).find((candidate) => candidate.id === candidateID))
      .map(candidateReplyText)
      .filter(Boolean)
      .join('\n');
    const text = [selectedText, String(rawText || '').trim()].filter(Boolean).join('\n');
    if (!text) return;
    updateFloatingAvatarDraft('');
    await submitComposerText(text);
  }

  function handleFloatingAvatarDrop(files = []) {
    const fileList = Array.from(files || []);
    const nativePaths = normalizeReferenceImportPaths(fileList.map((file) => file.path));
    if (nativePaths.length) {
      if (shouldProbeDroppedInstallPackage(nativePaths)) {
        detectDroppedInstallPackage(nativePaths[0]).then((recognized) => {
          if (!recognized) importReferencePaths(nativePaths, {addedVia: 'floating_avatar'});
        });
        return;
      }
      importReferencePaths(nativePaths, {addedVia: 'floating_avatar'});
      return;
    }
    importReferenceFileObjects(fileList);
  }

  const floatingAdapter = resolveActiveAdapter();
  const floatingStatusText = pendingTaskReview?.title
    || schedulerConfirm?.title
    || toolResult?.message
    || state.statusRail?.text
    || state.greeting
    || '';
  // 後台迷你框顯示前先剝 composer pending 內部標記（\u2063pending:traceId），避免 traceId 亂碼外洩。
  const floatingLatestText = stripComposerPendingMarker(messages[messages.length - 1] || state.greeting || '');
  // 只在「需要主動提醒」時才浮出上方泡泡（待確認/排程確認）。
  // 進後台、一般狀態、問候語都不浮泡泡，維持乾淨頭像；人格名稱/回覆改由點開迷你框顯示。
  // 全身像來源：persona 自填的 fullBodyAvatarUrl 優先，否則依角色 pack/state 取內建立繪。
  const floatingExpression = !floatingReminderPaused && (pendingTaskReview || schedulerConfirm) ? 'warning' : avatarExpression;
  const floatingFullBodyAvatarKey = pixelPackForPersona(floatingPersona, floatingAvatarConfig);
  const floatingFullBodyAvatarSrc = resolvePersonaFullBodySrc(floatingPersona, floatingAvatarConfig, floatingExpression);
  const floatingFullBodyMotionManifest = resolveAvatarMotionManifest(
    floatingFullBodyAvatarKey,
    floatingExpression,
    floatingFullBodyAvatarSrc,
  );
  // 紀念照「拍照」共用流程：Windows 原生浮窗選單與 mac 迷你面板的拍照鈕都走這裡，
  // 讓兩個平台的後台頭像介面行為一致。
  async function takeFloatingKeepsakePhoto() {
    if (floatingPhotoBusy) return;
    const scene = [
      floatingAvatarDraft,
      floatingAvatarReplyBubble,
      floatingLatestText,
      state.greeting,
    ].map((item) => String(item || '').trim()).find(Boolean) || '';
    setFloatingPhotoBusy(true);
    setToolResult({toolId: 'floating-avatar-photo', ok: true, message: '正在拍照…（首次產圖較久）'});
    try {
      const photo = await callWails(() => ConfirmCommemorativePhoto(scene, ''));
      const message = '已拍下並存入相冊：' + (photo?.scene || '合照');
      setFloatingAvatarReplyBubble(message);
      syncFloatingAvatarOverlayMetadata(message);
      setManualAvatarState('happy');
      setToolResult({toolId: 'floating-avatar-photo', ok: true, message});
    } catch (error) {
      const message = '產圖失敗：' + String(error?.message || error);
      setFloatingAvatarReplyBubble(message);
      syncFloatingAvatarOverlayMetadata(message);
      setManualAvatarState('sad');
      setToolResult({toolId: 'floating-avatar-photo', ok: false, message});
    } finally {
      setFloatingPhotoBusy(false);
    }
  }
  useEffect(() => {
    const off = EventsOn('floating_avatar:menu_action', async (payload) => {
      const action = payload?.action || 'restore';
      if (action === 'restore') {
        await restoreFromFloatingAvatar('auto');
        return;
      }
      if (action === 'open_panel') {
        await setFloatingAvatarChatModeEnabled(true, {silent: true});
        return;
      }
      if (action === 'quit') {
        await closeAfterBackgroundAvatarExit();
        return;
      }
      if (action === 'chat_on') {
        await setFloatingAvatarChatModeEnabled(true, {silent: true});
        return;
      }
      if (action === 'chat_submit') {
        await submitFloatingAvatarChatText(payload?.text || '');
        return;
      }
      if (action === 'photo') {
        await takeFloatingKeepsakePhoto();
        return;
      }
      if (action === 'chat') {
        await toggleFloatingAvatarChatMode(!floatingAvatarChatModeRef.current, {prompt: false});
        return;
      }
      if (action === 'head' || action === 'full') {
        await changeFloatingAvatarBodyMode(action);
      }
    });
    return () => off?.();
  }, [floatingAvatarSrc, floatingFullBodyAvatarSrc, floatingAvatarDraft, floatingAvatarReplyBubble, floatingLatestText, state.greeting]);
  const floatingBubbleText = pendingTaskReview?.reason
    || pendingTaskReview?.title
    || schedulerConfirm?.reason
    || schedulerConfirm?.title
    || floatingAvatarReplyBubble
    || '';
  const dagSuggestionText = String(draft || '').trim();
  const dagTaskSuggestionAction = dagSuggestionText
    && dismissedDagSuggestionText !== dagSuggestionText
    && shouldSuggestDagRun(dagSuggestionText)
    ? {
        title: '要轉成任務嗎？',
        lines: ['這句可能需要多步驟處理；轉成任務後會顯示進度，並啟用自我修正。'],
        cancelLabel: '留在聊天',
        primaryLabel: '轉成任務',
      }
    : null;
  const schedulerComposerConfirmAction = buildSchedulerComposerConfirmAction(schedulerConversation, schedulerBusy);
  const activeComposerConfirmAction = schedulerComposerConfirmAction || dagTaskSuggestionAction;
  const activeComposerConfirmHandler = schedulerComposerConfirmAction ? confirmComposerAction : acceptDagTaskSuggestion;
  const activeComposerCancelHandler = schedulerComposerConfirmAction ? cancelComposerAction : dismissDagTaskSuggestion;

  return (
    <div
      className={`console-shell ${activePanel === 'settings' ? 'settings-open' : ''} ${floatingAvatarMode && (ENABLE_NATIVE_FLOATING_AVATAR_WINDOW || floatingAvatarCompactWindow || floatingAvatarSurfaceOnly) ? 'floating-avatar-shell-active' : ''}`}
      data-theme={panelTheme}
      data-lang={_i18nLang}
      dir={_i18nDir}
      onDragOver={handleGlobalFileDragOver}
      onDrop={handleGlobalFileDrop}
      style={{'--ui-font-scale': fontScaleValue(settingsState.panel.fontScale), ...fontPresetVars(settingsState.panel.fontPreset)}}
    >
      <FloatingAvatarMode
        active={floatingAvatarMode}
        t={t}
        avatarSrc={floatingAvatarSrc}
        fullBodyAvatarSrc={floatingFullBodyAvatarSrc}
        fullBodyAvatarKey={floatingFullBodyAvatarKey}
        fullBodyMotionManifest={floatingFullBodyMotionManifest}
        avatarExpression={floatingExpression}
        persona={floatingPersona}
        personas={floatingAvatarPersonas}
        adapterLabel={floatingAdapter?.name || floatingAdapter?.Name || activeAdapterId || ''}
        position={floatingAvatarPosition}
        avatarSize={FLOATING_AVATAR_SIZE}
        onPositionChange={updateFloatingAvatarPosition}
        compactWindowMode={floatingAvatarCompactWindow}
        panelOpenSignal={floatingAvatarPanelOpenSignal}
        bodyMode={floatingAvatarBodyMode}
        dynamicImageEnabled={floatingAvatarDynamicImage}
        motionMode={floatingAvatarMotionMode}
        chatMode={floatingAvatarChatMode}
        onBodyModeChange={changeFloatingAvatarBodyMode}
        onDynamicImageChange={changeFloatingAvatarDynamicImage}
        onMotionModeChange={changeFloatingAvatarMotionMode}
        onChatModeChange={(enabled) => setFloatingAvatarChatModeEnabled(enabled, {clearHistory: !enabled})}
        onChatModeToggle={toggleFloatingAvatarChatMode}
        onCompactDragStart={beginCompactFloatingAvatarDrag}
        onCompactDrag={moveCompactFloatingAvatarWindow}
        onCompactDragEnd={syncCompactFloatingAvatarWindowPosition}
        onCompactExpandChange={setCompactAvatarExpanded}
        flyingBack={floatingAvatarFlyingBack}
        onRestore={() => restoreFromFloatingAvatar('auto')}
        onQuit={async () => {
          try {
            await closeAfterBackgroundAvatarExit();
          } catch (error) {
            setToolResult({toolId: 'scheduler', ok: false, message: t('subagent.closeFail', { error: error?.message || error })});
          }
        }}
        onOpenSettings={() => restoreFromFloatingAvatar('settings')}
        onDropFiles={handleFloatingAvatarDrop}
        onSubmit={submitFloatingAvatarText}
        onPhoto={takeFloatingKeepsakePhoto}
        photoBusy={floatingPhotoBusy}
        voiceState={voiceState}
        voiceRecording={voiceRecording}
        voiceBusy={voiceBusy}
        voiceChatEnabled={voiceChatEnabled}
        voiceSyncActive={voiceSyncActive}
        voiceSyncPhase={voiceSyncPhase}
        onVoicePressStart={handleVoicePressStart}
        onVoicePressEnd={handleVoicePressEnd}
        onVoiceCancel={handleVoicePointerCancel}
        onSwitchAgent={switchFloatingAvatarAgent}
        onSetReminderMode={setFloatingReminderMode}
        activePersonaId={floatingPersona.id}
        reminderPaused={floatingReminderPaused}
        reminderLabel={floatingReminderLabel}
        candidates={readinessGate.floating_candidates || []}
        selectedCandidateIDs={selectedFloatingCandidateIDs}
        exclusiveCandidates={isExclusiveCandidateSet(readinessGate.floating_candidates || [])}
        onSelectCandidate={handleSelectCandidate}
        statusTitle={pendingTaskReview?.id ? t('floatingAvatar.pendingConfirm') : t('floatingAvatar.statusTitle')}
        statusText={floatingStatusText}
        latestText={floatingLatestText}
        bubbleText={floatingBubbleText}
        pendingConfirm={pendingTaskReview || schedulerConfirm}
        installCandidate={installCandidate}
        onConfirmInstall={confirmInstallCandidate}
        onCancelInstall={() => setInstallCandidate(null)}
        draft={floatingAvatarDraft}
        onDraftChange={updateFloatingAvatarDraft}
        shakeDialogueOptions={getPersonaPatrolDialogueOptions(floatingPersona)}
        onShakePreview={(preview) => {
          if (preview?.expression) setManualAvatarState(preview.expression);
        }}
        onShakeTooLong={(dialogue) => {
          setManualAvatarState('speechless');
          setToolResult({toolId: 'scheduler', ok: false, message: dialogue || t('floatingAvatar.shakeTooLong')});
        }}
      />
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
        onRegisterLLMAPI={async (setup) => {
          const result = await callWails(() => RegisterLLMAPIAdapter(
            setup.providerId || 'openai',
            setup.providerName || 'LLM API',
            setup.baseURL || '',
            setup.model || '',
            setup.apiKey || '',
          ));
          if (!result?.need_confirm) await refreshAvailableAdapters();
          return result;
        }}
        onConfirmRegisterLLMAPI={async (setup) => {
          const result = await callWails(() => ConfirmRegisterLLMAPIAdapter(
            setup.providerId || 'openai',
            setup.providerName || 'LLM API',
            setup.baseURL || '',
            setup.model || '',
            setup.apiKey || '',
          ));
          await refreshAvailableAdapters();
          return result;
        }}
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
        voiceChatEnabled={voiceChatEnabled}
        voiceInstallBusy={voiceInstallBusy}
        onToggleVoiceChat={toggleVoiceChatMode}
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
      {installCandidate && !floatingAvatarMode && (
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
      {schedulerPanelOpen && (
        <React.Suspense fallback={null}>
          <SchedulerPanel
            clock={schedulerClock}
            jobs={schedulerJobs}
            draft={schedulerDraft}
            busy={schedulerBusy}
            error={schedulerError}
            onDraftChange={setSchedulerDraft}
            onRefresh={() => Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()])}
            onCreate={createSchedulerJob}
            onJobAction={updateSchedulerJob}
            onBootstrapSkill={bootstrapSchedulerSkill}
            onClose={() => setSchedulerPanelOpen(false)}
          />
        </React.Suspense>
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
          <React.Suspense fallback={null}>
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
            onCodeIndexRebuild={() => callWails(RebuildCodeIndex)}
            onCodeIndexSearch={(query, limit) => callWails(() => SearchCodeSections(query, limit))}
            onCodeIndexBuildContext={(query, highImpact) => callWails(() => BuildCodeContext(query, highImpact))}
            avatarConfigs={avatarConfigs}
            avatarExpression={avatarExpression}
            avatarModeNotice={avatarModeNotice}
            renderedPixelAvatars={renderedPixelAvatars}
            staticAvatarPreviews={staticAvatarPreviews}
            onAvatarProviderSelect={setAvatarProviderMode}
            onAvatarStateSelect={setManualAvatarState}
            onAvatarLoad={loadCurrentAvatar}
            />
          </React.Suspense>
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
              pinnedPopouts={pinnedPopouts}
              onHaoraPinOut={pinOutConversation}
              onHaoraUnpin={unpinConversation}
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
              wa3ImportPopup={wa3ImportPopup}
              wa3Detail={wa3Detail}
              wa3PollutionResult={wa3PollutionResult}
              wa3TransferGuidance={wa3TransferGuidance}
              wa3TrustList={wa3TrustList}
              wa3ActionBusy={wa3ActionBusy}
              wa3ActionError={wa3ActionError}
              wa3StatusConfig={wa3StatusConfig}
              wa3ToastMsg={wa3ToastMsg}
              onLoadWA3Info={loadWA3MediaInfo}
              onDetectWA3Pollution={detectWA3Pollution}
              onShowWA3Guidance={showWA3TransferGuidance}
              onTrustWA3Developer={trustWA3Developer}
              onExportWA3Copy={exportWA3WithSidecarCopy}
              onDismissWA3ImportPopup={dismissWA3ImportPopup}
              onShowWA3Toast={(msg) => {
                setWa3ToastMsg(msg);
                setTimeout(() => setWa3ToastMsg(null), 4000);
              }}
              onDismissWA3Toast={() => setWa3ToastMsg(null)}
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
              panelTheme={panelTheme}
              taskLoopRounds={taskLoopRounds}
              taskLoopReply={taskLoopReply}
              onTaskLoopReplyChange={setTaskLoopReply}
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
                  // 級聯清除該訊息的使用者標註與系統暗標（動態載入，避免綁定尚未產生時 build 失敗）
                  Promise.resolve(PurgeMessageMarks?.(convId, messageDomId(target, index, messages))).catch(() => {});
                }
              }}
              onSummarizeSearch={summarizeSearchResultText}
              onExportSearchSummary={startNativeSearchSummaryExport}
              // SEC-06: 讀取網址按鈕注入文字，複用後端確認流程
              onInjectText={(text) => submitComposerText(text)}
              activeConversationId={activeConversationIdRef.current || 'main'}
              chatLocale={panelLangToLocale(settingsState.panel?.roleLanguage) || panelLangToLocale(settingsState.panel?.panelLanguage) || _i18nLang || 'zh-TW'}
              // v3.6.4 Readiness Gate UI Interaction Layer props
              readinessGate={readinessGate}
              selectedFloatingCandidateIDs={selectedFloatingCandidateIDs}
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
              onVoicePressStart={handleVoicePressStart}
              onVoicePressEnd={handleVoicePressEnd}
              onVoiceCancel={handleVoicePointerCancel}
              voiceSyncActive={voiceSyncActive}
              voiceSyncPhase={voiceSyncPhase}
              voiceChatEnabled={voiceChatEnabled}
              voiceChatHint={voiceChatHint}
              onToggleVoiceChat={toggleVoiceChatMode}
              onDismissVoiceChatHint={() => setVoiceChatHint(false)}
              taskActive={isTaskProgressActive(dagRun) || !!activeChatTrace || voiceSyncActive || voiceRecording}
              onCancelTask={cancelActiveExecution}
              pendingTaskReview={pendingTaskReview}
              taskReviewDetailsOpen={reviewPopup === 'risk'}
              onConfirmTaskReview={confirmSkillBuild}
              onCancelTaskReview={() => cancelActiveTaskProgress('review_cancel')}
              onShowTaskReviewDetails={() => setReviewPopup((current) => current === 'risk' ? null : 'risk')}
              composerConfirmAction={activeComposerConfirmAction}
              onComposerConfirm={activeComposerConfirmHandler}
              onComposerCancel={activeComposerCancelHandler}
            />
          </main>
          <RightRail
          tools={tools}
          toolVisibility={toolVisibility}
          onToolActivate={activateTool}
          isToolPopupOpen={toolPopupsOpen.right}
          referenceFiles={referenceFiles}
          activeCodeFileName={codeArtifactModal?.meta?.file_name || ''}
          sharedLinks={extSharedLinks}
          sharedListings={sharedSourceListings}
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
          onSharedSourceDragOut={openSharedSourceActionDialog}
          onScheduleOpen={openSchedulerPanel}
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
          className={[
            'vl-monitor-launcher',
            vlMonitorOpen ? 'vl-monitor-launcher-active' : '',
            floatingAvatarMode ? 'vl-monitor-launcher-avatar-mode' : '',
            floatingAvatarMode && (vlHasBlocking || vlPendingCount > 0) ? 'vl-monitor-launcher-attention' : '',
          ].filter(Boolean).join(' ')}
          data-vl-target="visual-learning-monitor-launcher"
          onClick={() => setVlMonitorOpen((v) => !v)}
          type="button"
          aria-label={vlMonitorOpen ? t('vl.hideMonitor') : t('vl.openMonitor')}
          title={vlMonitorOpen ? t('vl.hidePanel') : t('vl.panelTitle')}
        >
          <span>VL</span>
          {vlPendingCount > 0 && <span className="vl-monitor-launcher-badge">{vlPendingCount}</span>}
        </button>,
        document.body
      )}
      {typeof document !== 'undefined' && vlMonitorOpen && createPortal(
        <React.Suspense fallback={null}>
	          <VisualLearningPanel
	            learningActive={vlLearningActive}
	            onLearningToggle={toggleVisualLearningRecording}
	            pendingCount={vlPendingCount}
	            hasBlocking={vlHasBlocking}
	            recentEvents={vlRecentLearningEvents}
	            onClose={() => setVlMonitorOpen(false)}
	          />
        </React.Suspense>,
        document.body
      )}
      {/* §M3 Embedding picker modal：first-drop 才開 */}
      {embeddingPickerTarget && typeof document !== 'undefined' && createPortal(
        <React.Suspense fallback={null}>
          <EmbeddingPickerModal
            displayName={embeddingPickerTarget.displayName}
            onClose={() => setEmbeddingPickerTarget(null)}
          />
        </React.Suspense>,
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
      {adapterModelRepairToast && (
        <DigestAutoArchiveToast message={adapterModelRepairToast} onDismiss={() => setAdapterModelRepairToast(null)} />
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
          onSkillNativeDragStart={startNativeSkillExport}
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
          onSkillNativeDragStart={startNativeSkillExport}
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
      {skillExportDialog && (
        <DragActionModal
          ariaLabel="Skill export"
          icon="✦"
          title={skillExportDialog.name}
          detail={skillExportDialog.landedDetail || skillExportDialog.landedPath}
          actions={[
            {label: t('adapter.remove'), onClick: () => finalizeSkillExport('remove')},
            {label: t('adapter.copyAction'), onClick: () => finalizeSkillExport('copy')},
            {label: t('common.cancel'), onClick: () => finalizeSkillExport('cancel')},
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
      {codeArtifactModal && (
        <CodeArtifactModal
          detail={codeArtifactModal}
          onClose={() => setCodeArtifactModal(null)}
          onExport={exportCodeArtifactToFolder}
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
      {searchSummaryExportDialog && (
        <DragActionModal
          ariaLabel="搜尋摘要拖曳操作"
          icon="☰"
          title={searchSummaryExportDialog.name}
          detail={searchSummaryExportDialog.landedDetail || searchSummaryExportDialog.landedPath}
          actions={[
            // 對話來源不可「移除」原文，故鎖住此鍵；保留三鍵外觀與共用視窗一致。
            {label: t('adapter.remove'), disabled: true, onClick: () => {}},
            {label: t('adapter.copyAction'), onClick: () => handleSearchSummaryExportAction('copy')},
            {label: t('common.cancel'), onClick: () => handleSearchSummaryExportAction('cancel')},
          ]}
        />
      )}
      {sharedSourceActionDialog && (
        <DragActionModal
          ariaLabel="共用來源拖曳操作"
          icon="◎"
          title={sharedSourceActionDialog.name}
          detail={sharedSourceActionDialog.detail || '共用來源'}
          actions={[
            {label: t('adapter.remove'), onClick: () => handleSharedSourceAction('remove')},
            {label: t('adapter.copyAction'), disabled: true, onClick: () => {}},
            {label: t('common.cancel'), onClick: () => handleSharedSourceAction('cancel')},
          ]}
        />
      )}
      <GoCodePreviewDock />
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

      {/* H/I：排程需確認執行卡 */}
      {schedulerConfirm && (
        <div className="session-close-overlay" role="dialog" aria-modal="true" aria-label="排程確認執行">
          <div className="session-close-dialog">
            <h3>排程需要你確認：{schedulerConfirm.title || '排程任務'}</h3>
            <p className="session-close-hint">
              {schedulerConfirm.reason === 'paid_api'
                ? '只剩付費雲端 API 可用，確認後才會用 API 執行（可能產生費用）。'
                : '此排程風險等級較高，確認後才會執行。'}
              {schedulerConfirm.summary ? ` 內容：${schedulerConfirm.summary}` : ''}
            </p>
            <div className="session-close-actions">
              <button
                type="button"
                className="session-close-btn session-close-save"
                disabled={schedulerConfirmBusy}
                onClick={async () => {
                  setSchedulerConfirmBusy(true);
                  try {
                    await callWails(() => ConfirmScheduledRun(schedulerConfirm.job_id));
                    setToolResult({toolId: 'scheduler', ok: true, message: `已確認執行：${schedulerConfirm.title || ''}`});
                    setSchedulerConfirm(null);
                  } catch (error) {
                    setToolResult({toolId: 'scheduler', ok: false, message: `執行失敗：${error?.message || error}`});
                  } finally {
                    setSchedulerConfirmBusy(false);
                  }
                }}
              >確認執行</button>
              <button
                type="button"
                className="session-close-btn session-close-discard"
                disabled={schedulerConfirmBusy}
                onClick={() => setSchedulerConfirm(null)}
              >略過</button>
            </div>
          </div>
        </div>
      )}

      {/* Phase G：關閉前的後台頭像選擇 */}
      {schedulerBgPrompt && (
        <div className="session-close-overlay" role="dialog" aria-modal="true" aria-label={t('floatingAvatar.closeDialogLabel')}>
          <div className="session-close-dialog">
            <h3>{t('floatingAvatar.closeDialogTitle')}</h3>
            <p className="session-close-hint">
              {t('floatingAvatar.closeDialogHintBefore')}
              <strong>{t('floatingAvatar.closeDialogNoShutdown')}</strong>
              {t('floatingAvatar.closeDialogHintAfter')}
            </p>
            <div className="session-close-actions">
              <button
                type="button"
                className="session-close-btn session-close-save"
                disabled={schedulerBgBusy}
                onClick={async () => {
                  setSchedulerBgBusy(true);
                  try {
                    await callWails(() => ResolveSchedulerBackgroundPrompt(true));
                    await enterFloatingAvatarMode();
                  } catch (error) {
                    setToolResult({toolId: 'scheduler', ok: false, message: t('floatingAvatar.backgroundFail', {error: error?.message || error})});
                  } finally {
                    setSchedulerBgBusy(false);
                  }
                }}
              >{t('floatingAvatar.enterBackground')}</button>
              <button
                type="button"
                className="session-close-btn session-close-discard"
                disabled={schedulerBgBusy}
                onClick={async () => {
                  setSchedulerBgBusy(true);
                  try { await callWails(() => ResolveSchedulerBackgroundPrompt(false)); } catch { /* 停用 best-effort */ } finally {
                    setSchedulerBgBusy(false);
                    setSchedulerBgPrompt(false);
                    Quit();
                  }
                }}
              >{t('floatingAvatar.closeDirectly')}</button>
            </div>
          </div>
        </div>
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

function formatLearningOperationSearchResults(query, matches, forExecution = false) {
  if (!matches.length) {
    return `Ai:${_t('chatSystem.learning.operationNoMatches', {query})}`;
  }
  const lines = matches.slice(0, 6).map((item, index) => {
    const risk = item?.risk?.level ? _t('chatSystem.learning.risk', {value: item.risk.level}) : '';
    const keywords = Array.isArray(item?.keywords) && item.keywords.length
      ? _t('chatSystem.learning.keywords', {value: item.keywords.slice(0, 6).join(_t('chatSystem.listSeparator'))})
      : '';
    const opTag = item?.operation_tag ? _t('chatSystem.learning.operationType', {value: item.operation_tag}) : '';
    const score = Number.isFinite(Number(item?.score)) ? _t('chatSystem.learning.matchScore', {value: Number(item.score).toFixed(2)}) : '';
    const meta = [opTag, keywords, risk, score].filter(Boolean).join(_t('chatSystem.metaSeparator'));
    return _t('chatSystem.learning.catalogItem', {
      index: index + 1,
      title: item.title || item.operation_tag || item.tag || item.run_id || _t('chatSystem.learning.unnamedOperation'),
      summary: item.summary || _t('chatSystem.learning.noSummary'),
      meta: meta ? `\n   ${meta}` : '',
    });
  });
  const prefix = forExecution
    ? `Ai:${_t('chatSystem.learning.operationAmbiguous', {query})}`
    : `Ai:${_t('chatSystem.learning.operationMatches', {query})}`;
  return [prefix, ...lines, _t('chatSystem.learning.specifyOperation')].join('\n');
}

function formatLearningOperationCatalog(items) {
  if (!items.length) {
    return `Ai:${_t('chatSystem.learning.catalogEmpty')}`;
  }
  const lines = items.slice(0, 10).map((item, index) => {
    const risk = item?.risk?.level ? _t('chatSystem.learning.risk', {value: item.risk.level}) : '';
    const keywords = Array.isArray(item?.keywords) && item.keywords.length
      ? _t('chatSystem.learning.keywords', {value: item.keywords.slice(0, 6).join(_t('chatSystem.listSeparator'))})
      : '';
    const opTag = item?.operation_tag ? _t('chatSystem.learning.operationType', {value: item.operation_tag}) : '';
    const internal = item?.tag || item?.run_id ? _t('chatSystem.learning.internalId', {value: item.tag || item.run_id}) : '';
    const meta = [opTag, keywords, risk, _t('chatSystem.learning.stepCount', {count: item.step_count || 0}), internal].filter(Boolean).join(_t('chatSystem.metaSeparator'));
    return _t('chatSystem.learning.catalogItemWithMeta', {
      index: index + 1,
      title: item.title || item.operation_tag || _t('chatSystem.learning.unnamedOperation'),
      summary: item.summary || _t('chatSystem.learning.noSummary'),
      meta,
    });
  });
  return [`Ai:${_t('chatSystem.learning.catalogHeader')}`, ...lines].join('\n');
}

function formatLearningOperationLearned(run) {
  const stepCount = Number(run?.step_count ?? run?.StepCount ?? 0);
  if (stepCount <= 0) {
    return `Ai:${_t('chatSystem.learning.recordingEmpty')}`;
  }
  const title = run?.title || run?.name || run?.operation_tag || _t('chatSystem.learning.unnamedOperation');
  const summary = run?.summary || _t('chatSystem.learning.demoSaved');
  const keywords = Array.isArray(run?.keywords) && run.keywords.length
    ? `\n${_t('chatSystem.learning.keywords', {value: run.keywords.slice(0, 8).join(_t('chatSystem.listSeparator'))})}`
    : '';
  const opTag = run?.operation_tag ? `\n${_t('chatSystem.learning.operationType', {value: run.operation_tag})}` : '';
  const risk = run?.risk?.level ? `\n${_t('chatSystem.learning.risk', {value: run.risk.level})}` : '';
  return `Ai:${_t('chatSystem.learning.operationSaved', {title, summary, operationType: opTag, keywords, risk})}`;
}

function formatLearningReplayPlan(plan) {
  const steps = Array.isArray(plan?.steps) ? plan.steps : [];
  if (!steps.length) {
    return `Ai:${_t('chatSystem.learning.noReplaySteps')}`;
  }
  const lines = steps.map((step, index) => {
    const target = step.window_title || step.label || step.role || step.css_selector || step.tag || _t('chatSystem.learning.coordinates', {x: step.x, y: step.y});
    const selector = step.css_selector ? `，selector: ${step.css_selector}` : '';
    const anchor = step.windows_anchor?.ok ? `，anchor: ${step.windows_anchor.mode || 'available'}` : '，anchor: none';
    const nativeInfo = isNativeReplayStep(step)
      ? _t('chatSystem.learning.nativeWindowInfo', {window: step.window_title || 'unknown', process: basenameForDisplay(step.window_process || step.tag || '')})
      : '';
    const coordLabel = _t(isNativeReplayStep(step) ? 'chatSystem.learning.screenCoordinatesLabel' : 'chatSystem.learning.coordinatesLabel');
    const inputInfo = step.action === 'text'
      ? _t('chatSystem.learning.textInfo', {value: step.sensitive ? _t('chatSystem.learning.sensitivePlaceholder') : _t('chatSystem.learning.characterCount', {count: String(step.text || '').length})})
      : step.action === 'shortcut'
        ? _t('chatSystem.learning.shortcutInfo', {value: `${(step.modifiers || []).join('+')}${step.modifiers?.length ? '+' : ''}${step.key || ''}`})
        : '';
    return _t('chatSystem.learning.planStep', {index: index + 1, action: step.action || 'click', target, coordinateLabel: coordLabel, x: step.x, y: step.y, inputInfo, nativeInfo, selector, anchor});
  });
  const anchorCount = steps.filter((step) => step.windows_anchor?.ok).length;
  return [
    `Ai:${_t('chatSystem.learning.planIntro')}`,
    `Tag: ${plan?.tag || 'demo-last'}，Title: ${plan?.title || plan?.run_name || 'untitled'}。`,
    plan?.run_summary ? `Summary: ${plan.run_summary}` : '',
    _t('chatSystem.learning.planRunMeta', {run: plan?.run_id || 'unknown', steps: steps.length, anchors: anchorCount}),
    ...lines,
    _t('chatSystem.learning.planExecutionNote'),
  ].filter(Boolean).join('\n');
}

function formatLearningReplayConfirmation(plan) {
  const steps = Array.isArray(plan?.steps) ? plan.steps : [];
  const nativeSteps = steps.filter(isNativeReplayStep);
  const domSteps = steps.length - nativeSteps.length;
  const textSteps = steps.filter((step) => step.action === 'text' || step.action === 'sensitive_text');
  const windows = Array.from(new Map(nativeSteps.map((step) => {
    const label = `${basenameForDisplay(step.window_process || step.tag || 'unknown')} - ${step.window_title || 'unknown window'}`;
    return [label, label];
  })).values()).slice(0, 4);
  const lines = [
    _t('chatSystem.learning.confirmStepCount', {count: steps.length}),
    _t('chatSystem.learning.confirmMethodCounts', {dom: domSteps, native: nativeSteps.length}),
  ];
  if (textSteps.length) {
    lines.push(_t('chatSystem.learning.confirmTextSteps', {count: textSteps.length}));
  }
  if (windows.length) {
    lines.push('', _t('chatSystem.learning.externalWindows'), ...windows.map((item) => `- ${item}`));
  }
  lines.push('', _t('chatSystem.learning.confirmNativeWarning'), _t('chatSystem.learning.confirmQuestion'));
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
  const textOK = steps.filter((step) => step.ok && (step.method === 'text' || step.method === 'native_paste_safe' || step.method === 'native_type_text')).length;
  const keyOK = steps.filter((step) => step.ok && (step.method === 'key' || step.method === 'native_key')).length;
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
    `Ai:${_t('chatSystem.learning.executionComplete', {ok: okCount, total: steps.length})}`,
    _t('chatSystem.learning.executionMethods', {
      selector: selectorCount,
      nativeOk: nativeOK,
      nativeTotal,
      foreground: nativeTotal ? _t('chatSystem.learning.foregroundCount', {ok: nativeForegroundOK, total: nativeTotal}) : '',
      fallback: coordinateCount,
    }),
    _t('chatSystem.learning.executionInputs', {text: textOK, keys: keyOK}),
    _t('chatSystem.learning.executionVisual', {
      anchors: anchorTotal,
      total: steps.length,
      review: reviewAnchors,
      visualOk: visualOK,
      visualTotal,
      confirm: visualConfirm ? _t('chatSystem.learning.pendingReviewCount', {count: visualConfirm}) : '',
    }),
    _t('chatSystem.learning.executionIssues', {skipped: skipped.length, failed: failed.length, warned: warned.length}),
  ];
  const details = steps.filter((step) => step.warning || step.skipped || (!step.ok && !step.skipped) || isVisualRelocation(step)).map((step) => {
    const target = step.label || step.selector || _t('chatSystem.learning.coordinates', {x: step.x, y: step.y});
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
      error: _t('chatSystem.learning.errorSystemControl'),
    };
  }
  if (action === 'sensitive_text') {
    return {
      ok: false,
      skipped: true,
      method: 'sensitive_text',
      index: step?.index,
      selector,
      label,
      x: step?.x,
      y: step?.y,
      sensitive: true,
      error: _t('chatSystem.learning.errorSensitiveText'),
    };
  }
  if (action === 'text') {
    return executeWebviewTextReplayStep(step, selector, label);
  }
  if (action === 'shortcut' || action === 'key') {
    return executeWebviewKeyReplayStep(step, selector, label);
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
      error: _t('chatSystem.learning.errorUnsupportedAction', {action}),
    };
  }

  const bySelector = findReplayElementBySelector(selector);
  if (bySelector) {
    await showLearningReplayWebCursor(bySelector.x, bySelector.y);
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
        error: _t('chatSystem.learning.errorBlockedCoordinate'),
      };
    }
    await showLearningReplayWebCursor(point.x, point.y);
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
    error: _t('chatSystem.learning.errorNoClickableElement'),
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

function findReplayInputTarget(selector, step) {
  const bySelector = findReplayElementBySelector(selector);
  if (bySelector?.element) return bySelector;
  const point = normalizeReplayPoint(step);
  const byPoint = document.elementFromPoint(point.x, point.y);
  const element = byPoint?.closest?.('input, textarea, [contenteditable="true"]') || byPoint;
  if (!element) return null;
  return {element, x: point.x, y: point.y};
}

function executeWebviewTextReplayStep(step, selector, label) {
  if (step?.requires_confirmation && !step?.replay_confirmed) {
    return {
      ok: false,
      skipped: true,
      needs_confirmation: true,
      method: 'text_confirmation',
      index: step?.index,
      selector,
      label,
      x: step?.x,
      y: step?.y,
      text: step?.text || '',
      error: _t('chatSystem.learning.errorTextNeedsConfirmation'),
    };
  }
  const target = findReplayInputTarget(selector, step);
  if (!target?.element) {
    return {
      ok: false,
      method: 'text',
      index: step?.index,
      selector,
      label,
      x: step?.x,
      y: step?.y,
      error: _t('chatSystem.learning.errorNoTextElement'),
    };
  }
  const text = String(step?.text || '');
  insertReplayText(target.element, text);
  return {
    ok: true,
    method: 'text',
    index: step?.index,
    selector,
    label,
    x: target.x,
    y: target.y,
    text,
  };
}

function executeWebviewKeyReplayStep(step, selector, label) {
  const target = findReplayInputTarget(selector, step) || {element: document.activeElement || document.body, x: step?.x, y: step?.y};
  const element = target.element || document.body;
  const key = step?.key || '';
  const init = {
    key,
    code: key.length === 1 ? `Key${key.toUpperCase()}` : key,
    bubbles: true,
    cancelable: true,
    metaKey: (step?.modifiers || []).includes('cmd'),
    ctrlKey: (step?.modifiers || []).includes('ctrl'),
    altKey: (step?.modifiers || []).includes('option'),
    shiftKey: (step?.modifiers || []).includes('shift'),
  };
  element.dispatchEvent(new KeyboardEvent('keydown', init));
  element.dispatchEvent(new KeyboardEvent('keyup', init));
  return {
    ok: true,
    method: 'key',
    index: step?.index,
    selector,
    label,
    x: target.x,
    y: target.y,
    key,
    modifiers: step?.modifiers || [],
  };
}

function insertReplayText(element, text) {
  if (!element) return;
  element.focus?.();
  if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement) {
    const start = Number.isFinite(element.selectionStart) ? element.selectionStart : element.value.length;
    const end = Number.isFinite(element.selectionEnd) ? element.selectionEnd : start;
    element.value = `${element.value.slice(0, start)}${text}${element.value.slice(end)}`;
    const next = start + text.length;
    element.setSelectionRange?.(next, next);
  } else if (element.isContentEditable) {
    document.execCommand?.('insertText', false, text);
  }
  element.dispatchEvent(new InputEvent('input', {bubbles: true, cancelable: true, data: text, inputType: 'insertText'}));
  element.dispatchEvent(new Event('change', {bubbles: true}));
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
      text: result?.text || step?.text || '',
      key: result?.key || step?.key || '',
      modifiers: Array.isArray(result?.modifiers) ? result.modifiers : (step?.modifiers || []),
      sensitive: Boolean(result?.sensitive || step?.sensitive),
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

function buildPersonaPackageManifest(name, sourcePath, packageData = {}) {
  return JSON.stringify({
    name: name || basenameForDisplay(sourcePath) || 'persona',
    version: packageData.version || '0.0.1',
    package_type: 'persona',
    source_path: sourcePath || '',
    risk_tag: packageData.risk_tag || 'unknown',
    required_perms: packageData.required_perms || [],
    write_targets: [],
  });
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

const learningReplayWebCursorState = {
  cursor: null,
  trail: [],
  x: null,
  y: null,
  hideTimer: null,
};

function ensureLearningReplayWebCursor() {
  if (typeof document === 'undefined') return null;
  if (learningReplayWebCursorState.cursor?.isConnected) {
    return learningReplayWebCursorState.cursor;
  }
  const cursor = document.createElement('div');
  cursor.setAttribute('aria-hidden', 'true');
  cursor.style.position = 'fixed';
  cursor.style.left = '0px';
  cursor.style.top = '0px';
  cursor.style.width = '24px';
  cursor.style.height = '24px';
  cursor.style.zIndex = '2147483600';
  cursor.style.pointerEvents = 'none';
  cursor.style.transform = 'translate(-50%, -50%)';
  cursor.style.filter = 'drop-shadow(0 2px 5px rgba(0, 0, 0, 0.34))';
  cursor.style.transition = 'opacity 160ms ease';
  const svgNS = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(svgNS, 'svg');
  svg.setAttribute('viewBox', '0 0 32 32');
  svg.setAttribute('width', '24');
  svg.setAttribute('height', '24');
  svg.setAttribute('focusable', 'false');
  svg.setAttribute('aria-hidden', 'true');
  const paths = [
    {
      d: 'M6.5 12.2 10.5 4.8 14 10.2c1.3-.6 2.8-.6 4 0l3.5-5.4 4 7.4c1 5.8-2.1 10.3-9.5 15.2-7.4-4.9-10.5-9.4-9.5-15.2Z',
      fill: 'rgba(238,238,232,0.96)',
      stroke: 'rgba(24,24,22,0.92)',
      'stroke-width': '1.7',
      'stroke-linejoin': 'round',
    },
    {
      d: 'M11.2 18.4 16 25.8l4.8-7.4c-1.6 1.1-3.2 1.5-4.8 1.5s-3.2-.4-4.8-1.5Z',
      fill: 'rgba(160,160,154,0.38)',
    },
    {
      d: 'M11.6 14.2h2.4M18 14.2h2.4',
      stroke: 'rgba(20,20,18,0.95)',
      'stroke-width': '1.8',
      'stroke-linecap': 'round',
    },
    {
      d: 'M14.1 19.4c.5-.7 3.3-.7 3.8 0-.3 1.1-.9 1.7-1.9 1.7s-1.6-.6-1.9-1.7Z',
      fill: 'rgba(20,20,18,0.95)',
    },
  ];
  paths.forEach((attrs) => {
    const path = document.createElementNS(svgNS, 'path');
    Object.entries(attrs).forEach(([name, value]) => path.setAttribute(name, value));
    svg.appendChild(path);
  });
  const focusRing = document.createElementNS(svgNS, 'circle');
  focusRing.setAttribute('cx', '16');
  focusRing.setAttribute('cy', '16');
  focusRing.setAttribute('r', '3.1');
  focusRing.setAttribute('fill', 'none');
  focusRing.setAttribute('stroke', 'rgba(20,20,18,0.78)');
  focusRing.setAttribute('stroke-width', '1.2');
  svg.appendChild(focusRing);
  cursor.appendChild(svg);
  document.body.appendChild(cursor);
  learningReplayWebCursorState.cursor = cursor;
  return cursor;
}

function addLearningReplayWebCursorTrailDot(x, y) {
  if (typeof document === 'undefined') return;
  const dot = document.createElement('div');
  dot.setAttribute('aria-hidden', 'true');
  dot.style.position = 'fixed';
  dot.style.left = `${Math.round(x)}px`;
  dot.style.top = `${Math.round(y)}px`;
  dot.style.width = '4px';
  dot.style.height = '4px';
  dot.style.borderRadius = '999px';
  dot.style.background = 'rgba(238, 238, 232, 0.72)';
  dot.style.boxShadow = '0 0 0 1px rgba(20, 20, 18, 0.22)';
  dot.style.zIndex = '2147483599';
  dot.style.pointerEvents = 'none';
  dot.style.transform = 'translate(-50%, -50%)';
  dot.style.transition = 'opacity 420ms ease';
  document.body.appendChild(dot);
  learningReplayWebCursorState.trail.push(dot);
  while (learningReplayWebCursorState.trail.length > 10) {
    learningReplayWebCursorState.trail.shift()?.remove();
  }
  window.setTimeout(() => {
    dot.style.opacity = '0';
    window.setTimeout(() => dot.remove(), 460);
  }, 130);
}

async function showLearningReplayWebCursor(x, y) {
  if (typeof window === 'undefined' || typeof document === 'undefined') return;
  const targetX = Number(x);
  const targetY = Number(y);
  if (!Number.isFinite(targetX) || !Number.isFinite(targetY)) return;
  const cursor = ensureLearningReplayWebCursor();
  if (!cursor) return;
  if (learningReplayWebCursorState.hideTimer) {
    window.clearTimeout(learningReplayWebCursorState.hideTimer);
    learningReplayWebCursorState.hideTimer = null;
  }
  cursor.style.opacity = '1';
  const hasPrevious = Number.isFinite(learningReplayWebCursorState.x) && Number.isFinite(learningReplayWebCursorState.y);
  const startX = hasPrevious ? learningReplayWebCursorState.x : targetX - 28;
  const startY = hasPrevious ? learningReplayWebCursorState.y : targetY + 18;
  const frames = 9;
  for (let index = 1; index <= frames; index += 1) {
    const ratio = index / frames;
    const eased = 1 - Math.pow(1 - ratio, 3);
    const currentX = startX + (targetX - startX) * eased;
    const currentY = startY + (targetY - startY) * eased;
    cursor.style.left = `${Math.round(currentX)}px`;
    cursor.style.top = `${Math.round(currentY)}px`;
    addLearningReplayWebCursorTrailDot(currentX, currentY);
    await delayLearningReplay(18);
  }
  learningReplayWebCursorState.x = targetX;
  learningReplayWebCursorState.y = targetY;
  learningReplayWebCursorState.hideTimer = window.setTimeout(() => {
    cursor.style.opacity = '0';
  }, 900);
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
  if (/\.gguf$/i.test(path) || /\.gguf$/i.test(name)) return true;
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

function avatarStateLabel(state) {
  return getAvatarStateLabels()[state] || state;
}

function getPixelAvatarDataUrl(pack, state, size = pixelAvatarRenderSize) {
  const bundledAvatar = getBundledAvatarPackUrl(pack, state);
  if (bundledAvatar) return Promise.resolve(bundledAvatar);
  const cacheKey = `${pixelAvatarRenderVersion}:${pack}:${state}:${size}`;
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

function getBundledAvatarPackUrl(pack, state = 'idle') {
  const urls = PERSONA_STATE_AVATAR_URLS[pack];
  return urls?.[state] || urls?.idle || '';
}

let monitorLinkCache = null;
let monitorLinkPending = null;

function postDebugTrace(node, traceId, data) {
  resolveMonitorTraceURL()
    .then((url) => {
      if (!url) return null;
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
  monitorLinkCache = {url: ''};
  monitorLinkPending = null;
  return Promise.resolve('');
}

function makeDebugTraceID(scope) {
  return `${scope}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}


function normalizeToolList(toolList = []) {
  const merged = new Map(getFallbackTools().map((tool) => [tool.id, tool]));
  toolList.forEach((tool) => {
    if (!tool?.id) return;
    merged.set(tool.id, {...(merged.get(tool.id) || {}), ...tool});
  });
  return Array.from(merged.values());
}

function getPersonaAvatar(persona) {
  return persona?.avatarUrl || PERSONA_AVATAR_URLS[persona?.id] || PERSONA_AVATAR_URLS[LOCKED_PERSONA_ID];
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

async function imageSrcToBytes(src) {
  if (!src) return [];
  const response = await fetch(src);
  if (!response.ok) {
    throw new Error(`avatar image fetch failed: ${response.status}`);
  }
  return Array.from(new Uint8Array(await response.arrayBuffer()));
}

function resolveAvatarProvider(config) {
  return config?.avatar_provider || config?.AvatarProvider || 'built_in_pixel';
}

function resolveStaticAvatarPath(config) {
  return config?.static_avatar_path || config?.StaticAvatarPath || '';
}

function pixelPackForPersona(persona, config) {
  const pack = config?.pixel_pack || config?.PixelPack || '';
  if (['wolf', 'uncle', 'secretary', 'police', 'touharu'].includes(pack)) return pack;
  return defaultPixelPackForPersona(persona?.id);
}

// 全身像來源：persona 自填的 fullBodyAvatarUrl 優先，否則依角色 pack/state 取內建立繪。
// 若特定狀態沒有獨立立繪，blocked 仍走禁止姿勢，其餘回到等待姿勢。
function resolvePersonaFullBodySrc(persona, config, state) {
  const explicit = persona?.fullBodyAvatarUrl || persona?.full_body_avatar_url || '';
  if (explicit) return explicit;
  const pack = PERSONA_FULL_BODY_URLS[pixelPackForPersona(persona, config)];
  if (!pack) return '';
  if (state && pack[state]) return pack[state];
  return state === 'blocked' ? pack.blocked : pack.idle;
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
  if (dagRun?.status === 'running' || dagRun?.status === 'starting') return 'working_reaction';
  if (dagRun?.status === 'blocked') return 'warning';
  if (dagRun?.status === 'failed') return 'sad';
  if (dagRun?.status === 'completed') return 'working_reaction';
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

function replyStrategyPresetFor(value) {
  const key = String(value || '').trim();
  if (!key) return null;
  return getReplyStrategyPresets().find((preset) => preset.id === key || preset.label === key) || null;
}

function chatTonePresetFor(value) {
  const key = String(value || '').trim();
  if (!key) return null;
  return getChatTonePresets().find((preset) => preset.id === key || preset.label === key) || null;
}

function Sidebar({
  adapters, adapterList, activeAdapterId,
  adapterModelChoices, adapterModelOptions, onAdapterModelPick, onAdapterModelRefresh,
  onAdapterSelect, onLocalAdapterWake, adapterCandidateLinks, activePanel,
  isToolPopupOpen, panelSettings, onPanelChange, voiceState, voiceChatEnabled, voiceInstallBusy,
  onToggleVoiceChat,
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
        const localIdentity = String(`${rawName || ''} ${id || ''} ${a.endpoint || ''}`).toLowerCase();
        const localProviderName = localIdentity.includes('llamacpp') || localIdentity.includes('llama.cpp')
          ? 'llama.cpp'
          : localIdentity.includes('lm studio')
            ? 'LM Studio'
            : 'Ollama';
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
                  setModelPicker({
                    adapterID: item.key,
                    rect,
                    label: item.displayName || item.name,
                    cliVersion: item.cliVersion || '',
                    modelOptionSource: item.modelOptionSource || '',
                    modelOptionNote: item.modelOptionNote || '',
                  });
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
        <button
          className={`command-btn command-voice-chat${voiceChatEnabled ? ' command-voice-chat-active' : ''}`}
          type="button"
          aria-pressed={voiceChatEnabled}
          onClick={() => onToggleVoiceChat?.()}
        >
          <span>♬</span>
          <span>{t('settings.voiceChatMode')}</span>
          <i className="command-voice-chat-status" aria-hidden="true" />
        </button>
      </nav>
      <div className="highlight-groups-anchor" data-highlight-groups-anchor />
      {settingsOpen && (
        <React.Suspense fallback={null}>
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
        </React.Suspense>
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
              {(modelPicker.cliVersion || modelPicker.modelOptionSource) && (
                <small>
                  {[modelPicker.cliVersion ? `CLI ${modelPicker.cliVersion}` : '', modelPicker.modelOptionSource || ''].filter(Boolean).join(' · ')}
                </small>
              )}
              {modelPicker.modelOptionNote && <small>{modelPicker.modelOptionNote}</small>}
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

export function PersonaSettingsDrawer({
  settingsState, onPersonaAdd, onPersonaDrop, onPersonaChange, onPersonaNativeDrag, onPersonaNativeExportAction, onPersonaReorder,
  avatarConfigs = {}, avatarExpression, avatarModeNotice, renderedPixelAvatars = {},
  staticAvatarPreviews = {}, onAvatarProviderSelect, onAvatarStateSelect, onAvatarLoad,
}) {
  const t = useI18n(s => s.t);
  const personas = settingsState.personas;
  const activePersona = findActivePersona(settingsState) || fallbackSettings.personas[0];
  const activeReplyStrategy = activePersona.replyStrategy || '';
  const activeReplyStrategyIsPreset = !activeReplyStrategy || Boolean(replyStrategyPresetFor(activeReplyStrategy));
  const activeChatTone = activePersona.personality || '';
  const activeChatToneIsPreset = Boolean(chatTonePresetFor(activeChatTone));
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
  const [avatarMenuPersonaId, setAvatarMenuPersonaId] = useState(null);
  const [strengthDraft, setStrengthDraft] = useState(parseToneChance(activePersona.roleStrength));
  const [voiceProfiles, setVoiceProfiles] = useState([]);
  const [voicePreviewBusy, setVoicePreviewBusy] = useState(false);
  const activeVoiceId = activePersona.voiceId || defaultVoiceIdByPersonaId[activePersona.id] || 'os_default';
  const activeVoiceProfile = voiceProfiles.find((profile) => profile.voiceId === activeVoiceId) || voiceProfiles[0] || null;

  useEffect(() => {
    setStrengthDraft(parseToneChance(activePersona.roleStrength));
    setStrengthPickerOpen(false);
    onAvatarLoad?.(activePersona.id);
  }, [activePersona.id]);

  useEffect(() => {
    VoiceProfiles().then((list) => setVoiceProfiles(Array.isArray(list) ? list : [])).catch(() => {});
  }, []);

  function collectPersonaFormPatch() {
    const form = personaFormRef.current;
    if (!form) return {};
    return {
      name: limitChineseText(form.elements.personaName?.value || '', 100),
      identity: form.elements.identity?.value || '',
      replyStrategy: form.elements.replyStrategy?.value || '',
      roleStrength: `${strengthDraft}%`,
      personality: form.elements.personality?.value || '',
      voiceId: activeVoiceId,
      scenario: form.elements.scenario?.value || '',
      description: form.elements.description?.value || '',
      patrolDialogue: form.elements.patrolDialogue?.value || '',
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

  function selectPersonaVoice(label) {
    const profile = voiceProfiles.find((item) => localizedVoiceProfileName(item, t) === label);
    if (!profile) return;
    onPersonaChange(activePersona.id, {voiceId: profile.voiceId});
  }

  async function previewPersonaVoice() {
    if (!activeVoiceId || voicePreviewBusy) return;
    const name = activePersona.name || t('persona.fallbackName');
    setVoicePreviewBusy(true);
    try {
      await PreviewVoiceProfileText(activeVoiceId, t('persona.voicePreviewLine', { name }));
    } catch { /* 引擎不可用時靜默 */ }
    setVoicePreviewBusy(false);
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
        {personas.map((persona) => {
          const displayName = personaCardDisplayName(persona, t);
          return (
          <div className="settings-persona-card-wrap" key={persona.id}>
            <button
              className={`settings-persona-card ${persona.id === activePersona.id ? 'settings-persona-card-active' : ''} ${persona.id === draggedPersonaId ? 'settings-persona-card-dragging' : ''} ${persona.id === lockedPersonaId ? 'settings-persona-card-locked' : ''}`}
              type="button"
              data-persona-id={persona.id}
              draggable={false}
              onDragStart={(event) => event.preventDefault()}
              onClick={(event) => {
                if (suppressPersonaClickRef.current) {
                  event.preventDefault();
                  event.stopPropagation();
                  return;
                }
                setAvatarMenuPersonaId(null);
                commitActivePersonaForm();
                onPersonaChange(persona.id, {});
              }}
              onPointerDown={(event) => startPersonaPointerDrag(event, persona)}
            >
              <img
                className="settings-persona-avatar"
                draggable={false}
                onDragStart={(event) => event.preventDefault()}
                onDoubleClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  setAvatarMenuPersonaId((current) => (
                    persona.id === lockedPersonaId ? null : (current === persona.id ? null : persona.id)
                  ));
                }}
                src={resolvePersonaAvatarSrc(persona, avatarConfigs[persona.id], staticAvatarPreviews, renderedPixelAvatars[persona.id] || '')}
                alt={t('persona.avatarAlt', { name: displayName })}
              />
              <strong
                title={displayName}
                style={{'--persona-name-font-size': `${personaNameFontSize(displayName)}px`}}
              >
                {displayName}
              </strong>
              {personaPatrolDialogueBadge(persona) && (
                <small className="settings-persona-patrol-badge">{personaPatrolDialogueBadge(persona)}</small>
              )}
              {persona.id === lockedPersonaId && <small className="settings-persona-lock">{t('persona.lockedName')}</small>}
            </button>
            {avatarMenuPersonaId === persona.id && persona.id !== lockedPersonaId && (
              <div className="settings-persona-avatar-menu" role="menu" aria-label={t('persona.changeAvatarTitle')}>
                <button
                  type="button"
                  role="menuitem"
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    setAvatarMenuPersonaId(null);
                    onAvatarProviderSelect?.('static_image', persona.id);
                  }}
                >
                  {t('persona.changeAvatar')}
                </button>
              </div>
            )}
          </div>
          );
        })}
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
        <input
          aria-label={t('persona.nameAriaLabel')}
          defaultValue={activePersona.id === lockedPersonaId ? t('persona.lockedName') : activePersona.name}
          disabled={activePersona.id === lockedPersonaId}
          key={`${activePersona.id}-name-${activePersona.name}-${t('persona.lockedName')}`}
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
          <div className="persona-strategy-select persona-tone-field">
            <input name="personality" type="hidden" value={activePersona.personality || ''} readOnly />
            <select
              aria-label={t('persona.chatToneAriaLabel')}
              value={activeChatToneIsPreset ? activeChatTone : ''}
              onChange={(event) => onPersonaChange(activePersona.id, {personality: event.target.value})}
            >
              <option value="">{t('persona.chatToneNone')}</option>
              {getChatTonePresets().map((preset) => (
                <option key={preset.id} value={preset.id}>{preset.label}</option>
              ))}
            </select>
            {!activeChatToneIsPreset && (
              <input
                className="persona-tone-custom"
                aria-label={t('persona.chatToneCustomAriaLabel')}
                defaultValue={activeChatTone}
                key={`${activePersona.id}-tone-custom`}
                placeholder={t('persona.chatToneCustomPlaceholder')}
                maxLength={60}
                onBlur={(event) => onPersonaChange(activePersona.id, {personality: event.target.value})}
              />
            )}
          </div>
          <div className="persona-scenario-stack">
            <input
              aria-label={t('persona.scenarioAriaLabel')}
              defaultValue={activePersona.scenario}
              key={`${activePersona.id}-scenario`}
              name="scenario"
              placeholder={t('persona.scenePlaceholder')}
              onBlur={(event) => onPersonaChange(activePersona.id, {scenario: event.target.value})}
            />
            <div className="persona-voice-row">
              <SettingPopupSelect
                icon="♪"
                label={t('persona.voiceSelectLabel')}
                value={activeVoiceProfile ? localizedVoiceProfileName(activeVoiceProfile, t) : t('persona.voiceSelectLabel')}
                options={voiceProfiles.map((profile) => localizedVoiceProfileName(profile, t))}
                onSelect={selectPersonaVoice}
              />
              <button
                type="button"
                className="persona-voice-preview-btn"
                disabled={!activeVoiceProfile || voicePreviewBusy}
                onClick={previewPersonaVoice}
              >
                {voicePreviewBusy ? '…' : t('settings.voicePreviewPlay')}
              </button>
            </div>
          </div>
        </div>
        <textarea
          aria-label={t('persona.descriptionAriaLabel')}
          defaultValue={activePersona.description}
          key={`${activePersona.id}-description`}
          name="description"
          placeholder={t('persona.extraPlaceholder')}
          onBlur={(event) => onPersonaChange(activePersona.id, {description: event.target.value})}
        />
        <textarea
          aria-label={t('persona.patrolDialogueAriaLabel')}
          className="persona-patrol-dialogue"
          defaultValue={activePersona.patrolDialogue || ''}
          key={`${activePersona.id}-patrolDialogue`}
          name="patrolDialogue"
          placeholder={t('persona.patrolDialoguePlaceholder')}
          onBlur={(event) => onPersonaChange(activePersona.id, {patrolDialogue: event.target.value})}
        />
      </form>
      <button className="settings-bottom-action" type="button">↓</button>
    </main>
  );
}

function KeepsakeField({ label, value, onChange, placeholder, password }) {
  return (
    <label style={{ display: 'block', marginBottom: 8 }}>
      <div style={{ fontSize: 12, opacity: 0.7, marginBottom: 3 }}>{label}</div>
      <input
        type={password ? 'password' : 'text'}
        value={value || ''}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
        style={{ width: '100%', padding: '6px 8px', borderRadius: 6, border: '1px solid #555', background: 'transparent', color: 'inherit', boxSizing: 'border-box' }}
      />
    </label>
  );
}

function TopConsole({
  activePersona, activeAvatarConfig, avatarExpression, avatarModeNotice, avatarProvider, avatarSrc,
  browserPref, greeting, haoras, subagentTabs, personaJob, personaName,
  dagRun, reviewState = fallbackReviewState, reviewPopup, skillInjections = [], snoozeHours,
  systemStatusHistory = [], reviewArchive = [],
  showSkillFirstUseCard, onDismissSkillFirstUse,
  wa3ImportPopup, wa3Detail, wa3PollutionResult, wa3TransferGuidance, wa3TrustList = [],
  wa3ActionBusy = '', wa3ActionError = '', wa3StatusConfig = {}, wa3ToastMsg,
  onLoadWA3Info, onDetectWA3Pollution, onShowWA3Guidance, onTrustWA3Developer, onExportWA3Copy,
  onDismissWA3ImportPopup, onShowWA3Toast, onDismissWA3Toast,
  onAvatarProviderSelect, onAvatarLoad, onAvatarStateSelect, onPersonaJobChange, onPersonaNameChange,
  onReviewPopupChange, onRotateGreeting, onSkillSelect,
  onSnooze, onSnoozeHoursChange, onAcknowledgeDigestItem, onConfirmSkillBuild,
  activeAdapter, adapterOptions = [], activeAdapterId, onAdapterSelect, cliInspectorBusy, cliInspectorLog, onCLIInspectSend,
  // v3.6.3 Remote Bridge props
  remoteBridgeChannels, remoteBridgeModePopup, remoteBridgeInboundInfo, remoteBridgeInboundAdapters,
  onToggleRemoteBridge, onOpenRemoteBridgeMode, onSwitchRemoteBridgeMode,
  onRemoveRemoteBridge, onCloseRemoteBridgeModePopup,
  onOpenRemoteBridgeSetup, onRemoteBridgeRename, onTemporaryRemoteBridgeTest, onSaveRemoteBridgeInboundSecret, onMakeRemoteBridgePrimary,
  panelTheme = 'onanegiku', taskLoopRounds = {}, taskLoopReply = {}, onTaskLoopReplyChange = () => {},
  subExportCapabilities,
  activeHaoraId, onHaoraSelect, onRenameHaora, onHaorasReordered, onSubagentsChanged,
  pinnedPopouts = [], onHaoraPinOut, onHaoraUnpin,
}) {
  const {t} = useI18n();
  const safeReviewState = normalizeReviewState(reviewState);
  const safeSkillInjections = Array.isArray(skillInjections) ? skillInjections : [];
  const safeSystemStatusHistory = Array.isArray(systemStatusHistory) ? systemStatusHistory : [];
  const [editingName, setEditingName] = useState(false);
  const [editingJob, setEditingJob] = useState(false);
  const [personaInfoOpen, setPersonaInfoOpen] = useState(false);
  const [interactiveOpen, setInteractiveOpen] = useState(false);
  const [interactiveText, setInteractiveText] = useState('');
  // 紀念照：手動拍照的確認泡泡狀態
  const [keepsakeAsk, setKeepsakeAsk] = useState(false);
  const [keepsakeBusy, setKeepsakeBusy] = useState(false);
  const [keepsakeMsg, setKeepsakeMsg] = useState('');
  // 紀念照產圖設定（ComfyUI / 雲端）
  const [keepsakeSetOpen, setKeepsakeSetOpen] = useState(false);
  const [keepsakeCfg, setKeepsakeCfg] = useState(null);
  const openKeepsakeSettings = () => {
    setKeepsakeSetOpen(true);
    callWails(() => GetKeepsakeConfig()).then(setKeepsakeCfg).catch(() => setKeepsakeCfg({ mode: 'comfyui' }));
  };
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
  const localizedLockedPersonaName = t('persona.lockedName');

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
          const bundledPreview = getBundledAvatarPackUrl(p.id, 'idle');
          if (bundledPreview) {
            previews[p.id] = bundledPreview;
            continue;
          }
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
      setNameDraft(localizedLockedPersonaName);
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
                {personaNameLocked ? localizedLockedPersonaName : personaName}
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
            <div className="interactive-header-actions">
              <button
                type="button"
                className="interactive-photo-action"
                title="拍一張紀念照"
                onClick={() => { setKeepsakeAsk(true); setKeepsakeMsg(''); }}
              >
                <span aria-hidden="true">📸</span>
                <span>拍照</span>
              </button>
              <button
                type="button"
                className="interactive-photo-action"
                title="紀念照產圖設定"
                onClick={openKeepsakeSettings}
              >
                <span aria-hidden="true">⚙</span>
                <span>設定</span>
              </button>
            </div>
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
          {keepsakeAsk && (
            <div
              className="keepsake-bubble"
              style={{ position: 'relative', alignSelf: 'flex-start', margin: '4px 0 8px', maxWidth: 340, background: 'rgba(120,90,40,0.18)', border: '1px solid rgba(214,160,74,0.5)', borderRadius: 14, padding: '12px 14px' }}
            >
              <div style={{ marginBottom: 10 }}>
                要拍一張紀念照嗎？{interactiveText.trim() ? `場景：「${interactiveText.trim()}」` : '（沒填場景就拍一張合照）'}
              </div>
              {keepsakeMsg && <div style={{ fontSize: 12, opacity: 0.9, marginBottom: 8 }}>{keepsakeMsg}</div>}
              <div style={{ display: 'flex', gap: 8 }}>
                <button
                  type="button"
                  disabled={keepsakeBusy}
                  onClick={async () => {
                    setKeepsakeBusy(true);
                    setKeepsakeMsg('正在拍照…（首次產圖較久）');
                    try {
                      const photo = await callWails(() => ConfirmCommemorativePhoto(interactiveText.trim(), ''));
                      setKeepsakeMsg('已拍下並存入相冊：' + (photo?.scene || '合照'));
                    } catch (e) {
                      setKeepsakeMsg('產圖失敗：' + String(e && e.message ? e.message : e));
                    } finally {
                      setKeepsakeBusy(false);
                    }
                  }}
                >
                  {keepsakeBusy ? '拍照中…' : '拍'}
                </button>
                <button type="button" disabled={keepsakeBusy} onClick={() => { setKeepsakeAsk(false); setKeepsakeMsg(''); }}>關閉</button>
              </div>
            </div>
          )}
          {keepsakeSetOpen && keepsakeCfg && (
            <div
              className="keepsake-settings"
              style={{ margin: '4px 0 8px', background: 'rgba(40,40,48,0.55)', border: '1px solid rgba(214,160,74,0.4)', borderRadius: 12, padding: '12px 14px' }}
            >
              <div style={{ fontWeight: 600, marginBottom: 10 }}>紀念照產圖設定</div>
              <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
                <button
                  type="button"
                  onClick={() => setKeepsakeCfg({ ...keepsakeCfg, mode: 'comfyui' })}
                  style={{ flex: 1, padding: '6px', borderRadius: 8, cursor: 'pointer', color: 'inherit', background: 'transparent', border: (keepsakeCfg.mode || 'comfyui') === 'comfyui' ? '2px solid #d6a04a' : '1px solid #555' }}
                >
                  本機 ComfyUI
                </button>
                <button
                  type="button"
                  onClick={() => setKeepsakeCfg({ ...keepsakeCfg, mode: 'cloud' })}
                  style={{ flex: 1, padding: '6px', borderRadius: 8, cursor: 'pointer', color: 'inherit', background: 'transparent', border: keepsakeCfg.mode === 'cloud' ? '2px solid #d6a04a' : '1px solid #555' }}
                >
                  綁定雲端
                </button>
              </div>
              {(keepsakeCfg.mode || 'comfyui') === 'comfyui' ? (
                <>
                  <KeepsakeField label="ComfyUI 位址" value={keepsakeCfg.comfyui_url} onChange={(x) => setKeepsakeCfg({ ...keepsakeCfg, comfyui_url: x })} placeholder="http://127.0.0.1:8188" />
                  <KeepsakeField label="動漫模型 checkpoint（必填）" value={keepsakeCfg.checkpoint} onChange={(x) => setKeepsakeCfg({ ...keepsakeCfg, checkpoint: x })} placeholder="anything-v5.safetensors" />
                </>
              ) : (
                <>
                  <KeepsakeField label="雲端端點" value={keepsakeCfg.cloud_endpoint} onChange={(x) => setKeepsakeCfg({ ...keepsakeCfg, cloud_endpoint: x })} placeholder="https://api.openai.com/v1/images/generations" />
                  <KeepsakeField label="API 金鑰" value={keepsakeCfg.cloud_api_key} onChange={(x) => setKeepsakeCfg({ ...keepsakeCfg, cloud_api_key: x })} placeholder="sk-..." password />
                  <KeepsakeField label="模型" value={keepsakeCfg.cloud_model} onChange={(x) => setKeepsakeCfg({ ...keepsakeCfg, cloud_model: x })} placeholder="dall-e-3" />
                </>
              )}
              <KeepsakeField label="畫風前綴（可空，預設動漫風）" value={keepsakeCfg.style_preset} onChange={(x) => setKeepsakeCfg({ ...keepsakeCfg, style_preset: x })} placeholder="anime, 2D illustration, cel shading" />
              <div style={{ display: 'flex', gap: 8, marginTop: 6 }}>
                <button
                  type="button"
                  onClick={() => { callWails(() => SaveKeepsakeConfig(keepsakeCfg)).catch(() => {}); setKeepsakeSetOpen(false); }}
                  style={{ background: '#d6a04a', color: '#1b1b1f', border: 'none', borderRadius: 8, padding: '6px 16px', cursor: 'pointer', fontWeight: 600 }}
                >
                  儲存
                </button>
                <button type="button" onClick={() => setKeepsakeSetOpen(false)} style={{ background: 'transparent', color: 'inherit', border: '1px solid #555', borderRadius: 8, padding: '6px 16px', cursor: 'pointer' }}>關閉</button>
              </div>
            </div>
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
            // 釘選中的分頁：記憶由獨立視窗獨佔，主視窗鎖定；點擊＝收回。
            const isPinnedOut = !haora.isMain && (pinnedPopouts || []).some((item) => (
              item?.agent_id === haora.id || item?.agent_id === haora.name
            ));
            if (isPinnedOut) {
              return (
                <button
                  className="haora-card haora-card-pinned"
                  type="button"
                  key={haora.key}
                  data-haora-key={haora.key}
                  draggable={false}
                  title={t('subagent.pinnedTitle')}
                  onClick={() => onHaoraUnpin?.(haora)}
                >
                  <span className="haora-pin haora-pin-active" aria-hidden="true">
                    <svg viewBox="0 0 24 24" width="12" height="12">
                      <path fill="currentColor" d="M14.6 2.6a1 1 0 0 1 1.4 0l5.4 5.4a1 1 0 0 1 0 1.4l-1.2 1.2a1 1 0 0 1-1 .25l-.9-.27-3.8 3.8.3 2.9a1 1 0 0 1-.29.82l-.9.9a1 1 0 0 1-1.41 0l-3.3-3.3-5 5a1 1 0 0 1-1.41-1.41l5-5-3.3-3.3a1 1 0 0 1 0-1.41l.9-.9a1 1 0 0 1 .83-.29l2.9.3 3.8-3.8-.28-.9a1 1 0 0 1 .25-1z"/>
                    </svg>
                  </span>
                  <span className="haora-name">{haora.name}</span>
                </button>
              );
            }
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
                <span className="haora-name">{haora.name}</span>
                {/* 亮點只跟著目前 active 的 haㄌer/sub。 */}
                {isActiveHaora && <i aria-hidden="true"/>}
                {/* 圖釘：把這個對話釘成獨立 OS 視窗（吃當前主人格與模型，記憶獨立）。 */}
                {!haora.isMain && (
                  <span
                    className="haora-pin"
                    role="button"
                    tabIndex={-1}
                    aria-label={t('subagent.pinOut')}
                    title={t('subagent.pinOut')}
                    onClick={(event) => {
                      event.stopPropagation();
                      event.preventDefault();
                      onHaoraPinOut?.(haora);
                    }}
                    onPointerDown={(event) => event.stopPropagation()}
                    onPointerUp={(event) => event.stopPropagation()}
                    onDoubleClick={(event) => event.stopPropagation()}
                  >
                    <svg viewBox="0 0 24 24" width="12" height="12" aria-hidden="true">
                      <path fill="currentColor" d="M14.6 2.6a1 1 0 0 1 1.4 0l5.4 5.4a1 1 0 0 1 0 1.4l-1.2 1.2a1 1 0 0 1-1 .25l-.9-.27-3.8 3.8.3 2.9a1 1 0 0 1-.29.82l-.9.9a1 1 0 0 1-1.41 0l-3.3-3.3-5 5a1 1 0 0 1-1.41-1.41l5-5-3.3-3.3a1 1 0 0 1 0-1.41l.9-.9a1 1 0 0 1 .83-.29l2.9.3 3.8-3.8-.28-.9a1 1 0 0 1 .25-1z"/>
                    </svg>
                  </span>
                )}
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
        {safeSystemStatusHistory.length > 0 && (
          <div className="system-status-history">
            <span className="system-status-history-label">{t('persona.systemStatus')}</span>
            {safeSystemStatusHistory.slice(-5).map((entry, i) => (
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
          <span className={`review-strip-status review-status-${safeReviewState.highRisk.status}`}>
            {reviewStatusLabel(safeReviewState.highRisk.status)}
          </span>
        </div>
        <ReviewPanel
          activePopup={reviewPopup}
          dagRun={dagRun}
          onPopupChange={onReviewPopupChange}
          onSkillSelect={onSkillSelect}
          onSnooze={onSnooze}
          onSnoozeHoursChange={onSnoozeHoursChange}
          reviewState={safeReviewState}
          snoozeHours={snoozeHours}
          onAcknowledgeDigestItem={onAcknowledgeDigestItem}
          onConfirmSkillBuild={onConfirmSkillBuild}
          reviewArchive={reviewArchive}
          wa3ImportPopup={wa3ImportPopup}
          wa3Detail={wa3Detail}
          wa3PollutionResult={wa3PollutionResult}
          wa3TransferGuidance={wa3TransferGuidance}
          wa3TrustList={wa3TrustList}
          wa3ActionBusy={wa3ActionBusy}
          wa3ActionError={wa3ActionError}
          wa3StatusConfig={wa3StatusConfig}
          wa3ToastMsg={wa3ToastMsg}
          onLoadWA3Info={onLoadWA3Info}
          onDetectWA3Pollution={onDetectWA3Pollution}
          onShowWA3Guidance={onShowWA3Guidance}
          onTrustWA3Developer={onTrustWA3Developer}
          onExportWA3Copy={onExportWA3Copy}
          onDismissWA3ImportPopup={onDismissWA3ImportPopup}
          onShowWA3Toast={onShowWA3Toast}
          onDismissWA3Toast={onDismissWA3Toast}
          panelTheme={panelTheme}
          taskLoopRounds={taskLoopRounds}
          taskLoopReply={taskLoopReply}
          onTaskLoopReplyChange={onTaskLoopReplyChange}
        />
        {safeSkillInjections.length > 0 && (
          <SkillActivityCard injections={safeSkillInjections} />
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

function ReviewPanel({
  activePopup, dagRun, onPopupChange, reviewState, onSkillSelect, onSnooze,
  onSnoozeHoursChange, snoozeHours, onAcknowledgeDigestItem, onConfirmSkillBuild,
  reviewArchive, wa3ImportPopup, wa3Detail, wa3PollutionResult, wa3TransferGuidance, wa3TrustList = [],
  wa3ActionBusy = '', wa3ActionError = '', wa3StatusConfig = {}, wa3ToastMsg,
  onLoadWA3Info = () => {}, onDetectWA3Pollution = () => {}, onShowWA3Guidance = () => {},
  onTrustWA3Developer = () => {}, onExportWA3Copy = () => {},
  onDismissWA3ImportPopup = () => {}, onShowWA3Toast = () => {}, onDismissWA3Toast = () => {},
  panelTheme = 'onanegiku',
  taskLoopRounds = {}, taskLoopReply = {}, onTaskLoopReplyChange = () => {},
}) {
  const t = useI18n(s => s.t);
  const normalizedReviewState = normalizeReviewState(reviewState);
  const {highRisk, lowRiskAmbiguity, hookCandidates, pendingDigest, pendingPackages} = normalizedReviewState;
  const ambiguityHidden = Boolean(lowRiskAmbiguity.dismissedUntil);
  const panelRef = useRef(null);
  const [popupFrame, setPopupFrame] = useState(null);
  const [digestExpanded, setDigestExpanded] = useState({urgent: true, later: false, archive: false});
  const [taskDebugDump, setTaskDebugDump] = useState(null);
  const activeWA3Info = wa3Detail || wa3ImportPopup?.info || {};
  const activeWA3Training = activeWA3Info.training || {};
  const activeWA3Pollution = wa3PollutionResult || activeWA3Info.pollution;
  const activeWA3Path = wa3ImportPopup?.source_path || activeWA3Info.file_path || '';
  const activeWA3Sidecar = wa3ImportPopup?.sidecar_path || '';

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
          <span>{t('dag.taskProgress')}</span>
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
          <strong>{ambiguityHidden ? t('dag.dismissLabel', { time: lowRiskAmbiguity.dismissedUntil }) : t('dag.lowRisk')}</strong>
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
          data-theme={panelTheme}
          style={{left: `${popupFrame.left}px`, top: `${popupFrame.top}px`, width: `${popupFrame.width}px`}}
        >
          <header>
            <span>{dagRun?.title || t('dag.noDagRun')}</span>
            <strong>{dagRun ? dagStatusLabel(dagRun.status) : t('dag.idle')}</strong>
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
                      {node.status === 'running' && taskLoopRounds[node.id] && (
                        <small className="dag-loop-round">
                          {taskLoopRounds[node.id].message
                            ? `第 ${taskLoopRounds[node.id].iteration} 輪：${taskLoopRounds[node.id].message}`
                            : `第 ${taskLoopRounds[node.id].iteration} 輪：${taskLoopRounds[node.id].action} ${taskLoopRounds[node.id].target}`}
                        </small>
                      )}
                      {node.status === 'waiting_user' && (
                        <span className="dag-loop-reply">
                          <input
                            type="text"
                            value={taskLoopReply[node.id] || ''}
                            placeholder="輸入補充資訊後送出"
                            onChange={(e) => {
                              const value = e.target.value;
                              onTaskLoopReplyChange((prev) => ({...prev, [node.id]: value}));
                            }}
                            onKeyDown={(e) => {
                              if (e.key !== 'Enter') return;
                              const reply = (taskLoopReply[node.id] || '').trim();
                              if (!reply) return;
                              callWails(() => SubmitTaskLoopInput(dagRun.id, node.id, reply))
                                .then(() => onTaskLoopReplyChange((prev) => ({...prev, [node.id]: ''})))
                                .catch((error) => onTaskLoopReplyChange((prev) => ({...prev, [node.id]: reply, [`${node.id}:error`]: error?.message || String(error)})));
                            }}
                          />
                          {taskLoopReply[`${node.id}:error`] && <small className="dag-loop-error">{taskLoopReply[`${node.id}:error`]}</small>}
                        </span>
                      )}
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
          data-theme={panelTheme}
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
          data-theme={panelTheme}
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
          data-theme={panelTheme}
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
                      <small>{item.category || item.ui_group || item.backend_group} · {item.risk || item.risk_level || 'unknown'}</small>
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
                      <small>{item.category || item.ui_group || item.backend_group}</small>
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
                      <small>{item.category || item.ui_group || item.backend_group}</small>
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

      {/* ── v3.6.2 WA3 Media Provenance（§9A）UI ── */}

      {/* WA3 匯入選單：偵測到 sidecar 後顯示功能說明 */}
      {wa3ImportPopup && (
        <div className="wa3-import-overlay" onClick={onDismissWA3ImportPopup}>
          <div className="wa3-import-popup" onClick={(e) => e.stopPropagation()}>
            <div className="wa3-import-header">
              <span className="wa3-import-icon">
                {wa3StatusConfig[wa3ImportPopup.info?.status]?.icon || '❓'}
              </span>
              <span className="wa3-import-title">{t("wa3.title")}</span>
            </div>
            <div className="wa3-import-status" style={{color: wa3StatusConfig[activeWA3Info.status]?.color || '#95a5a6'}}>
              {wa3StatusConfig[activeWA3Info.status]?.label || t('wa3.unknownStatus')}
            </div>
            <div className="wa3-import-recommendation">{wa3ImportPopup.recommendation}</div>
            {wa3ImportPopup.has_sidecar && (
              <div className="wa3-import-sidecar-badge">{t("wa3.sidecarDetected")}</div>
            )}
            <div className="wa3-detail-grid">
              {activeWA3Path && (
                <div className="wa3-detail-row">
                  <span>{t('wa3.sourcePath')}</span>
                  <code>{activeWA3Path}</code>
                </div>
              )}
              {activeWA3Sidecar && (
                <div className="wa3-detail-row">
                  <span>{t('wa3.sidecarPath')}</span>
                  <code>{activeWA3Sidecar}</code>
                </div>
              )}
              {activeWA3Info.media_scope && (
                <div className="wa3-detail-row">
                  <span>{t('wa3.mediaScope')}</span>
                  <strong>{activeWA3Info.media_scope}</strong>
                </div>
              )}
              {activeWA3Info.training && (
                <>
                  <div className="wa3-detail-row">
                    <span>{t('wa3.trainingSafe')}</span>
                    <strong>{activeWA3Training.training_safe ? t('wa3.safeYes') : t('wa3.safeNo')}</strong>
                  </div>
                  <div className="wa3-detail-row">
                    <span>{t('wa3.filterRequired')}</span>
                    <strong>{activeWA3Training.filter_required ? t('common.yes') : t('common.no')}</strong>
                  </div>
                </>
              )}
              <div className="wa3-detail-row">
                <span>{t('wa3.trustCount')}</span>
                <strong>{wa3TrustList.length}</strong>
              </div>
            </div>
            {activeWA3Pollution && (
              <div className={`wa3-pollution-card ${activeWA3Pollution.is_pollution_risk ? 'wa3-pollution-risk' : ''}`}>
                <span>{t('wa3.weightedTotal')}</span>
                <strong>{typeof activeWA3Pollution.weighted_total === 'number' ? activeWA3Pollution.weighted_total.toFixed(2) : activeWA3Pollution.weighted_total}</strong>
                {activeWA3Pollution.details && <small>{activeWA3Pollution.details}</small>}
              </div>
            )}
            {wa3TransferGuidance && (
              <div className="wa3-guidance-card">
                <strong>{wa3TransferGuidance.ui_message}</strong>
                <div className="wa3-guidance-list">
                  <span>{t('wa3.recommended')}</span>
                  {(wa3TransferGuidance.recommended || []).map((item) => <small key={item}>{item}</small>)}
                </div>
                <div className="wa3-guidance-list">
                  <span>{t('wa3.notRecommended')}</span>
                  {(wa3TransferGuidance.not_recommended || []).map((item) => <small key={item}>{item}</small>)}
                </div>
              </div>
            )}
            <div className="wa3-import-capabilities">
              <span className="wa3-import-cap-title">{t("wa3.capTitle")}</span>
              {(wa3ImportPopup.capabilities || []).map((cap, i) => (
                <div className="wa3-import-cap-item" key={i}>▸ {cap}</div>
              ))}
            </div>
            {wa3ActionError && <div className="wa3-import-error">{wa3ActionError}</div>}
            <div className="wa3-import-actions">
              <button className="wa3-import-btn wa3-import-btn-secondary" type="button" disabled={!!wa3ActionBusy} onClick={onLoadWA3Info}>{wa3ActionBusy === 'info' ? t('wa3.actionBusy') : t('wa3.infoAction')}</button>
              <button className="wa3-import-btn wa3-import-btn-secondary" type="button" disabled={!!wa3ActionBusy} onClick={onDetectWA3Pollution}>{wa3ActionBusy === 'pollution' ? t('wa3.actionBusy') : t('wa3.pollutionAction')}</button>
              <button className="wa3-import-btn wa3-import-btn-secondary" type="button" disabled={!!wa3ActionBusy} onClick={onShowWA3Guidance}>{wa3ActionBusy === 'guidance' ? t('wa3.actionBusy') : t('wa3.guidanceAction')}</button>
              <button className="wa3-import-btn wa3-import-btn-secondary" type="button" disabled={!!wa3ActionBusy} onClick={onTrustWA3Developer}>{wa3ActionBusy === 'trust' ? t('wa3.actionBusy') : t('wa3.trustAction')}</button>
              <button className="wa3-import-btn wa3-import-btn-warning" type="button" disabled={!!wa3ActionBusy} onClick={onExportWA3Copy}>{wa3ActionBusy === 'export' ? t('wa3.actionBusy') : t('wa3.exportCopyAction')}</button>
              <button className="wa3-import-btn wa3-import-btn-primary" type="button" onClick={onDismissWA3ImportPopup}>{t('wa3.confirm')}</button>
            </div>
          </div>
        </div>
      )}

      {/* §24: 文件寫入確認卡片 */}
      <DocumentReviewCard onToast={onShowWA3Toast} />

      {/* WA3 傳輸引導 toast（軟性提示） */}
      {wa3ToastMsg && (
        <div className="wa3-toast" onClick={onDismissWA3Toast}>
          <span className="wa3-toast-icon">🔏</span>
          <span className="wa3-toast-text">{wa3ToastMsg}</span>
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
    const cat = item.category || item.backend_group || item.ui_group || '';
    if (cat === 'risky' || cat === 'high_value' || cat === 'risky_candidate' || cat === 'high_value_candidate' || cat === 'needs_decision' || cat === 'needs_your_decision') {
      blocks.urgent.push(item);
    } else if (cat === 'keep_suggestion' || cat === 'can_wait') {
      blocks.later.push(item);
    } else if (cat === 'archive_suggestion' || cat === 'duplicate_group' || cat === 'suggest_archive' || cat === 'suggested_archive') {
      blocks.archive.push(item);
    } else {
      blocks.later.push(item);
    }
  }
  return blocks;
}

// ---------------------------------------------------------------------------
// I-4: Safari Runtime Notice Modal
// ---------------------------------------------------------------------------
export default App;
