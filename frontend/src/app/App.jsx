import React, {useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import '../tailwind.css';
import MessageRow from '../components/chat/MessageRow';
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
  // Keep: debug trace monitor bridge. Used by postDebugTrace() even when the
  // local monitor port is not currently listening.
  GetMonitorLinks,
  // SEC-05 2b: 開外部連結唯一入口（Go 端做 scheme/metadata 檢查，loopback 放行）
  GetNewSubagentCandidates,
  GetPendingDigest,
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
  PreviewExternalLink,
  RegisterExternalLink,
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
  UpdateScheduledJob,
  GetRemoteBridgeInboundEndpoint,
  SaveRemoteBridgeInboundSecret,
  ListRemoteBridgeInboundAdapters,
  // ── v3.6 雜項接線 ──
  FinalizeNativeSearchSummaryExport,
  FinalizeNativeReferenceFileExport,
  ImportReferenceFile,
  ListReferenceFiles,
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
  EnterFloatingAvatarNative,
  ExitFloatingAvatarNative,
} from '../../wailsjs/go/main/App';
import {
  OnFileDrop,
  OnFileDropOff,
  EventsOn,
  BrowserOpenURL,
  Quit,
  ClipboardGetText,
  ClipboardSetText,
  WindowGetPosition,
  WindowGetSize,
  WindowSetAlwaysOnTop,
  WindowSetBackgroundColour,
  WindowSetMinSize,
  WindowSetPosition,
  WindowSetSize,
  WindowShow,
  WindowUnminimise,
} from '../../wailsjs/runtime/runtime';

import DocumentReviewCard from '../components/DocumentReviewCard';
import VisualLearningPanel from '../components/VisualLearningPanel';
import EmbeddingPickerModal from '../components/EmbeddingPickerModal';
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
import PackageConfirmModal from './components/PackageConfirmModal';
import SessionCloseDialog from './components/SessionCloseDialog';
// small leaf components extracted to app/components
import SkillActivityCard from './components/SkillActivityCard';
import SkillFirstUseCard from './components/SkillFirstUseCard';
import SkillContextSettingsSection from './components/SkillContextSettingsSection';
import DraftSandboxStopDialog from './components/DraftSandboxStopDialog';
import TrustedSessionExpiredDialog from './components/TrustedSessionExpiredDialog';
import SettingsMenu from './components/SettingsMenu';
import SettingsWorkspace from './components/SettingsWorkspace';
import {isNativeReplayStep, basenameForDisplay, isExclusiveCandidateSet, defaultWebSearchProviderOptions, toolTabFor, dagStatusLabel} from '../lib/appHelpers';
import {
  _fontPresetLabelMap,
  _panelLangLabelMap,
  _roleLangLabelMap,
  defaultPanelStyle,
  localizeBackendLabel,
  normalizePanelStyle,
  styleKeyOf,
  voiceStatusLabel,
} from '../lib/panelSettings';
// Tier-1 dialogs extracted to app/components
import PackageInstallDecisionDialog from './components/PackageInstallDecisionDialog';
import SubToolConflictDialog from './components/SubToolConflictDialog';
import DragActionModal from './components/DragActionModal';
import LearningReplayStartConfirmCard from './components/LearningReplayStartConfirmCard';
import LearningReplayConfirmCard from './components/LearningReplayConfirmCard';
import LLMAPISetupModal from './components/LLMAPISetupModal';
import WebSearchSetupModal from './components/WebSearchSetupModal';
import StyleDiffPreviewModal from './components/StyleDiffPreviewModal';
// readiness-gate feature group
import LongPressConfirmButton from '../features/readiness-gate/LongPressConfirmButton';
import FloatingCandidateActions from '../features/readiness-gate/FloatingCandidateActions';
import MissingSlotCapsule from '../features/readiness-gate/MissingSlotCapsule';
import RetrievalTransparency from '../features/readiness-gate/RetrievalTransparency';
import ConfirmationTier from '../features/readiness-gate/ConfirmationTier';
import ComposerConfirmBubble from '../features/readiness-gate/ComposerConfirmBubble';
import useI18n, { t as _t, tForLanguage, getAllFemaleKeywords, getAllBeastKeywords, buildGreetingTextKeyMap } from '../locales/useI18n';

const taskProgressDebugEnabled = typeof window !== 'undefined'
  && (new URLSearchParams(window.location.search).has('taskDebug')
    || window.localStorage?.getItem('task_progress_debug') === '1');

const MAIN_WINDOW_MIN_SIZE = {width: 1180, height: 560};
const FLOATING_AVATAR_SIZE = 80;
const FLOATING_AVATAR_INSET = 16;
const FLOATING_AVATAR_WINDOW_SIZE = FLOATING_AVATAR_SIZE + FLOATING_AVATAR_INSET * 2;
// 後台頭像單擊展開迷你框時，需要把頭像浮窗放大到能容納面板的尺寸。
const FLOATING_AVATAR_PANEL_W = 720;
const FLOATING_AVATAR_PANEL_H = 540;

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
  const haystack = [persona.name, persona.identity, persona.personality, persona.scenario]
    .filter(Boolean)
    .join(' ')
    .toLowerCase();
  if (!haystack) return 'male';
  if (matchesAnyKeyword(haystack, GREETING_FEMALE_KEYWORDS)) return 'fem';
  if (matchesAnyKeyword(haystack, GREETING_BEAST_KEYWORDS)) return 'wild';
  return 'male';
}

function pickRotatingGreeting(currentText, variant = 'male') {
  const options = getGreetingRotationOptions(variant);
  const regular = options.filter((item) => !item.rare);
  const rare = options.filter((item) => item.rare);
  const basePool = rare.length > 0 && Math.random() < 0.05 ? rare : regular;
  let pool = basePool.filter((item) => item.text !== currentText);
  if (pool.length === 0) {
    pool = regular.filter((item) => item.text !== currentText);
  }
  if (pool.length === 0) {
    pool = getGreetingRotationOptions(variant);
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

const STYLE_KEY_THEME = {
  default: 'onanegiku',
  passiveWhite: 'white',
  pinkBetrayal: 'pink-black',
  forgiveMeGreen: 'green',
  defeatBlue: 'blue',
};
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
function candidateReplyText(candidate) {
  const draft = stripInternalControlDraft(candidate?.draft || candidate?.label || '');
  return draft.startsWith('input:') ? draft.slice('input:'.length).trim() : draft.trim();
}

const GACHA_PULSE_MS = 500;
const GACHA_PARTICLE_MS = 600;
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
    {id: 'persona-d', name: _t('persona.defaultNameD'), icon: '⚖', avatarUrl: '', identity: _t('persona.defaultIdentityD'), replyStrategy: '', roleStrength: '20%', personality: _t('persona.defaultPersonalityD'), scenario: '', description: ''},
    {id: 'persona-e', name: _t('persona.defaultNameE'), icon: '☯', avatarUrl: '', identity: _t('persona.defaultIdentityE'), replyStrategy: '', roleStrength: '20%', personality: _t('persona.defaultPersonalityE'), scenario: '', description: ''},
  ],
  removedDefaultPersonaIds: [],
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
const getChatTonePresets = () => [
  {id: 'deadpan_weary', label: _t('persona.toneDeadpanWeary')},
  {id: 'stoic_tough', label: _t('persona.toneStoicTough')},
  {id: 'calm_rational', label: _t('persona.toneCalmRational')},
  {id: 'warm_gentle', label: _t('persona.toneWarmGentle')},
  {id: 'upbeat_energetic', label: _t('persona.toneUpbeatEnergetic')},
  {id: 'witty_playful', label: _t('persona.toneWittyPlayful')},
  {id: 'tsundere', label: _t('persona.toneTsundere')},
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

const adapterMeta = {
  Claude: {className: 'adapter-lime', icon: 'C'},
  Codex: {className: 'adapter-pink', icon: '◎'},
  Gemini: {className: 'adapter-lime', icon: '✦'},
};

const maxPersonas = 16;

const personaAvatarUrls = {
  'persona-a': new URL('../assets/persona_avatars/persona-a.svg', import.meta.url).href,
  'persona-b': new URL('../assets/persona_avatars/persona-b.svg', import.meta.url).href,
  'persona-c': new URL('../assets/persona_avatars/persona-c.svg', import.meta.url).href,
  'persona-d': new URL('../assets/persona_avatars/persona-d.svg', import.meta.url).href,
  'persona-e': new URL('../assets/persona_avatars/touharu/idle.png', import.meta.url).href,
};

const wolfdogAvatarUrls = {
  idle: new URL('../assets/persona_avatars/wolfdog/idle.png', import.meta.url).href,
  thinking: new URL('../assets/persona_avatars/wolfdog/thinking.png', import.meta.url).href,
  working: new URL('../assets/persona_avatars/wolfdog/working.png', import.meta.url).href,
  happy: new URL('../assets/persona_avatars/wolfdog/happy.png', import.meta.url).href,
  warning: new URL('../assets/persona_avatars/wolfdog/warning.png', import.meta.url).href,
  blocked: new URL('../assets/persona_avatars/wolfdog/blocked.png', import.meta.url).href,
  sleepy: new URL('../assets/persona_avatars/wolfdog/sleepy.png', import.meta.url).href,
  sad: new URL('../assets/persona_avatars/wolfdog/sad.png', import.meta.url).href,
  speechless: new URL('../assets/persona_avatars/wolfdog/speechless.png', import.meta.url).href,
};

const uncleBustAvatarUrls = {
  idle: new URL('../assets/persona_avatars/uncle_bust/idle.png', import.meta.url).href,
  thinking: new URL('../assets/persona_avatars/uncle_bust/thinking.png', import.meta.url).href,
  working: new URL('../assets/persona_avatars/uncle_bust/working.png', import.meta.url).href,
  happy: new URL('../assets/persona_avatars/uncle_bust/happy.png', import.meta.url).href,
  warning: new URL('../assets/persona_avatars/uncle_bust/warning.png', import.meta.url).href,
  blocked: new URL('../assets/persona_avatars/uncle_bust/blocked.png', import.meta.url).href,
  sleepy: new URL('../assets/persona_avatars/uncle_bust/sleepy.png', import.meta.url).href,
  sad: new URL('../assets/persona_avatars/uncle_bust/sad.png', import.meta.url).href,
  speechless: new URL('../assets/persona_avatars/uncle_bust/speechless.png', import.meta.url).href,
};

const secretaryAvatarUrls = {
  idle: new URL('../assets/persona_avatars/secretary/idle.png', import.meta.url).href,
  thinking: new URL('../assets/persona_avatars/secretary/thinking.png', import.meta.url).href,
  working: new URL('../assets/persona_avatars/secretary/working.png', import.meta.url).href,
  happy: new URL('../assets/persona_avatars/secretary/happy.png', import.meta.url).href,
  warning: new URL('../assets/persona_avatars/secretary/warning.png', import.meta.url).href,
  blocked: new URL('../assets/persona_avatars/secretary/blocked.png', import.meta.url).href,
  sleepy: new URL('../assets/persona_avatars/secretary/sleepy.png', import.meta.url).href,
  sad: new URL('../assets/persona_avatars/secretary/sad.png', import.meta.url).href,
  speechless: new URL('../assets/persona_avatars/secretary/speechless.png', import.meta.url).href,
};

const policeAvatarUrls = {
  idle: new URL('../assets/persona_avatars/police/idle.png', import.meta.url).href,
  thinking: new URL('../assets/persona_avatars/police/thinking.png', import.meta.url).href,
  working: new URL('../assets/persona_avatars/police/working.png', import.meta.url).href,
  happy: new URL('../assets/persona_avatars/police/happy.png', import.meta.url).href,
  warning: new URL('../assets/persona_avatars/police/warning.png', import.meta.url).href,
  blocked: new URL('../assets/persona_avatars/police/blocked.png', import.meta.url).href,
  sleepy: new URL('../assets/persona_avatars/police/sleepy.png', import.meta.url).href,
  sad: new URL('../assets/persona_avatars/police/sad.png', import.meta.url).href,
  speechless: new URL('../assets/persona_avatars/police/speechless.png', import.meta.url).href,
};

const touharuAvatarUrls = {
  idle: new URL('../assets/persona_avatars/touharu/idle.png', import.meta.url).href,
  thinking: new URL('../assets/persona_avatars/touharu/thinking.png', import.meta.url).href,
  working: new URL('../assets/persona_avatars/touharu/working.png', import.meta.url).href,
  happy: new URL('../assets/persona_avatars/touharu/happy.png', import.meta.url).href,
  warning: new URL('../assets/persona_avatars/touharu/warning.png', import.meta.url).href,
  blocked: new URL('../assets/persona_avatars/touharu/blocked.png', import.meta.url).href,
  sleepy: new URL('../assets/persona_avatars/touharu/sleepy.png', import.meta.url).href,
  sad: new URL('../assets/persona_avatars/touharu/sad.png', import.meta.url).href,
  speechless: new URL('../assets/persona_avatars/touharu/speechless.png', import.meta.url).href,
};

const pixelAvatarRenderSize = 128;
const pixelAvatarRenderVersion = '2026-06-21-transparent-touharu-raster-v1';
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
  let text = String(message || '');
  const markerIndex = text.indexOf(composerPendingMarker);
  if (markerIndex >= 0) {
    text = text.slice(0, markerIndex);
  }
  // Defensive cleanup for any pending trace suffix that accidentally reaches UI.
  return text.replace(/\s*pending:[A-Za-z0-9_-]+$/g, '');
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
const learningSensitiveTextPattern = /(password|passwd|token|api[_ -]?key|secret|bearer|sk-[a-z0-9_-]{16,}|xox[baprs]-|gh[pousr]_[a-z0-9_]{20,})/i;

function compactLearningText(value, fallback = '') {
  const text = String(value || '').replace(/\s+/g, ' ').trim();
  return text ? text.slice(0, 120) : fallback;
}

function normalizeLearningKey(value) {
  const key = String(value || '').trim();
  if (!key) return '';
  if (key === ' ') return 'Space';
  if (/^esc$/i.test(key)) return 'Escape';
  if (/^return$/i.test(key)) return 'Enter';
  if (/^arrow(left|right|up|down)$/i.test(key)) {
    return key.charAt(0).toUpperCase() + key.slice(1);
  }
  if (key.length === 1) return key.toUpperCase();
  return key.charAt(0).toUpperCase() + key.slice(1);
}

function formatVisualLearningPermissionStatus(status, requested = false) {
  const missing = Array.isArray(status?.missing) ? status.missing : [];
  if (!missing.length && status?.accessibility && status?.input_monitoring) {
    return requested
      ? '[系統] 已呼叫 macOS 權限請求；Visual Learning 必要權限看起來已允許。若剛剛才開啟，請重啟 ai-console 後再錄一次。'
      : '[系統] Visual Learning 權限看起來已允許。';
  }
  const list = missing.length ? missing.join('、') : '輔助使用 / 輸入監控';
  return `[系統] 已呼叫 macOS 權限請求。仍缺：${list}。請在系統對話框或 系統設定 → 隱私權與安全性 開啟 ai-console，然後重啟 app 再錄。`;
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
  const [skillExportDialog, setSkillExportDialog] = useState(null);
  const [referenceFiles, setReferenceFiles] = useState([]);
  const [referenceExportDialog, setReferenceExportDialog] = useState(null);
  const [searchSummaryExportDialog, setSearchSummaryExportDialog] = useState(null);
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
  useHighlight({enabled: true, conversationId: activeConversationId});
  const activeConversationIdRef = useRef('main');
  const loadedConversationIdsRef = useRef(new Set(['main']));
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
  const [floatingAvatarDrafts, setFloatingAvatarDrafts] = useState({});
  const [floatingReminderPause, setFloatingReminderPause] = useState({mode: '', until: 0});
  const floatingAvatarWindowRef = useRef({restore: null, compactPosition: null});
  const floatingAvatarDragWindowRef = useRef(null);
  const [floatingAvatarPosition, setFloatingAvatarPosition] = useState(() => {
    try {
      const saved = JSON.parse(window.localStorage.getItem('floating_avatar_position') || 'null');
      if (saved && Number.isFinite(saved.x) && Number.isFinite(saved.y)) return saved;
    } catch {}
    return {
      x: Math.max(12, window.innerWidth - FLOATING_AVATAR_WINDOW_SIZE),
      y: Math.max(12, window.innerHeight - FLOATING_AVATAR_WINDOW_SIZE - 16),
    };
  });
  const floatingAvatarModeRef = useRef(floatingAvatarMode);
  const floatingAvatarCompactWindowRef = useRef(floatingAvatarCompactWindow);
  const floatingAvatarPositionRef = useRef(floatingAvatarPosition);
  const floatingAvatarTransitionRef = useRef(false);
  floatingAvatarModeRef.current = floatingAvatarMode;
  floatingAvatarCompactWindowRef.current = floatingAvatarCompactWindow;
  floatingAvatarPositionRef.current = floatingAvatarPosition;

  useEffect(() => {
    if (floatingAvatarCompactWindow) return;
    try {
      window.localStorage.setItem('floating_avatar_position', JSON.stringify(floatingAvatarPosition));
    } catch {}
  }, [floatingAvatarCompactWindow, floatingAvatarPosition]);

  useEffect(() => {
    document.documentElement.classList.toggle('floating-avatar-window-active', floatingAvatarMode);
    document.body.classList.toggle('floating-avatar-window-active', floatingAvatarMode);
    return () => {
      document.documentElement.classList.remove('floating-avatar-window-active');
      document.body.classList.remove('floating-avatar-window-active');
    };
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
      return `${label} 的模型 ${model} 已失效，已清除固定選項。你現在可以改選目前可用的模型；建議先試 ${fallbackModel}。`;
    }
    if (reason && reason.includes('fallback list')) {
      return `${label} 的模型 ${model} 已失效，已清除固定選項。這次先退回 app 的備援清單。`;
    }
    return `${label} 的模型 ${model} 已失效，已清除固定選項。`;
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

  async function refreshAdapterModelOptionsForAdapters(adapters = []) {
    const out = {};
    for (const adapter of adapters) {
      const id = adapter?.id || adapter?.name;
      if (!id) continue;
      const opts = await callWails(() => ListAdapterModelOptions(id)).catch(() => null);
      if (Array.isArray(opts) && opts.length > 0) out[id] = opts;
    }
    setAdapterModelOptions(out);
    await reconcileAdapterModelChoicesAgainstOptions(adapterModelChoicesRef.current, out);
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
        const out = {};
        for (const a of adapterList) {
          const id = a.id || a.name;
          if (!id) continue;
          const opts = await callWails(() => ListAdapterModelOptions(id)).catch(() => null);
          if (Array.isArray(opts) && opts.length > 0) out[id] = opts;
        }
        if (cancelled) return;
        setAdapterModelOptions(out);
        await reconcileAdapterModelChoicesAgainstOptions(choices || {}, out);
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
      setW3aToastMsg(t('system.imported', { name: data?.display_name || t('w3a.defaultDocName') }));
      setTimeout(() => setW3aToastMsg(null), 4000);
    });

    // skill 產出落位完成 → 立即刷新右側引用面板（5 秒輪詢之外的即時路徑）。
    const offReferenceImported = EventsOn('reference:imported', (data) => {
      refreshReferenceFiles().catch(() => {});
      if (data?.name) {
        setW3aToastMsg(t('system.imported', { name: data.name }));
        setTimeout(() => setW3aToastMsg(null), 4000);
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
      const title = String(payload?.title || '').trim() || '排程任務';
      const summary = String(payload?.summary || payload?.action || '').trim();
      const notice = `Ai:排程提醒：「${title}」時間到，系統要開始處理${summary ? `：${summary}` : ''}。`;
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
            formatVisualLearningPermissionStatus(status, true) + (opened ? '\n已打開第一個缺少的權限設定頁。' : ''),
          ]);
        })
        .catch((error) => {
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            `[系統] 無法呼叫 macOS 權限請求。${error?.message || String(error || '')}`,
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
            allowed ? '[系統] 已保存這段文字輸入到示範。' : '[系統] 未保存文字內容；示範只留下敏感輸入佔位。',
          ]);
        })
        .catch((error) => {
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            `[系統] 文字輸入確認失敗。${error?.message || String(error || '')}`,
          ]);
        });
    });

    return () => {
      offAdapterChanged();
      offAdapterStatus();
      offAdapterModelCleared();
      offToolsChanged();
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
      offSchedulerRequested();
      offSchedulerReminder();
      offW3AVerified();
      offW3APollution();
      offW3AExported();
      offW3AImported();
      offW3ATrust();
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

  async function finishComposerExecution({resp, payload, apiAdapter, traceId, conversationId, clearPendingTimers}) {
    // 使用者已按停止鈕中斷此 trace：丟棄遲到的結果，不覆蓋「已中斷」訊息。
    if (cancelledChatTracesRef.current.has(traceId)) {
      cancelledChatTracesRef.current.delete(traceId);
      clearPendingTimers?.();
      return;
    }
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
      const message = `Ai:我找不到符合「${operationIntent.query}」的已保存操作：${err?.message || String(err)}`;
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
        const permissionStatus = await callWails(RequestVisualLearningPermissions).catch(() => null);
        if (permissionStatus?.missing?.length) {
          const opened = await openFirstMissingVisualLearningPermission(permissionStatus);
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            formatVisualLearningPermissionStatus(permissionStatus, true) + (opened ? '\n已打開第一個缺少的權限設定頁。' : ''),
          ]);
        }
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
              `[系統] 示範結束，但後端停止記錄時回報錯誤。${detail ? ` ${detail}` : ''}`,
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
    const greetingVariant = personaGreetingVariant(findActivePersona(settingsState));
    const next = pickRotatingGreeting(state.greeting, greetingVariant);
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

  // 停止鈕統一入口：優先中斷正在跑的一般對話／skill（非 DAG），否則回到 DAG 任務取消。
  async function cancelActiveExecution(reason = 'user_stop') {
    const chatTrace = activeChatTraceRef.current;
    if (chatTrace) {
      cancelledChatTracesRef.current.add(chatTrace);
      activeChatTraceRef.current = null;
      setActiveChatTrace(null);
      const conversationId = activeConversationIdRef.current || 'main';
      setConversationMessages(conversationId, (prev) => replaceComposerPendingMessage(prev, chatTrace, 'Ai:已中斷本次回覆。'));
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
    // DEBUG_TRACE_REMOVE: Correlates every debug trace event for this composer send.
    const traceId = makeDebugTraceID('chat');
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
    const sessionId = appSessionId || '';
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
    scheduleReadinessGateBurstRefresh();
    // 重置高風險確認流程狀態
    setRiskImpactExpanded(false);
    setGachaPhase(null);
    setLongPressProgress(0);
  }

  // ── DEV-ONLY 語系驗收矩陣鉤子（import.meta.env.DEV 閘控，不進 production）──
  // 為什麼：Computer Use 模擬鍵盤對 CJK 會掉字，原生日文/韓文題目打不進 UI。
  // 此鉤子把 CJK 題目寫死在 JS，逐題切角色語言並送入「真 UI」，讓操作者目視
  // 選項卡等渲染（選項卡人工檢核）。自動語系判定在 Go 端
  // （cjklang.go + lang_matrix_integration_test.go）；此處只負責驅動真 UI。
  // 用法：DevTools console 執行  await window.__runLangMatrix()
  useEffect(() => {
    if (!import.meta.env.DEV) return undefined;
    const LANG_MATRIX = [
      { label: '日本語', category: '一般對答', text: 'こんにちは、今日も手伝ってくれますか？' },
      { label: '日本語', category: '網路搜尋', text: 'ネットで夜行性の動物を調べて' },
      { label: '日本語', category: '非顏色選項', text: '6月21日、6月22日、6月23日から一つ選んで' },
      { label: '日本語', category: '本機搜尋', text: 'ローカルで動物のレポートを探して' },
      { label: '日本語', category: '程式流程', text: 'CSVをグラフに変換するプログラムを作って' },
      { label: '한국어', category: '一般對答', text: '안녕하세요, 오늘도 도와줄 수 있나요?' },
      { label: '한국어', category: '網路搜尋', text: '인터넷에서 야행성 동물을 검색해줘' },
      { label: '한국어', category: '非顏色選項', text: '6월 21일, 6월 22일, 6월 23일 중에서 하나 골라줘' },
      { label: '한국어', category: '本機搜尋', text: '로컬에서 동물 보고서를 찾아줘' },
      { label: '한국어', category: '程式流程', text: 'CSV를 그래프로 변환하는 프로그램을 만들어줘' },
      { label: 'English', category: '一般對答', text: 'Hello, can you help me today?' },
      { label: 'English', category: '網路搜尋', text: 'Search the web for nocturnal animal species' },
      { label: 'English', category: '非顏色選項', text: 'Choose one date: June 21, June 22, June 23' },
      { label: 'English', category: '本機搜尋', text: 'Find local animal reports' },
      { label: 'English', category: '程式流程', text: 'Make a program that charts animal counts from CSV' },
      { label: '中文', category: '一般對答', text: '你好，今天可以幫我嗎？' },
    ];
    const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
    window.__runLangMatrix = async (opts = {}) => {
      const waitMs = opts.waitMs ?? 9000;
      const rounds = opts.rounds ?? 2;
      console.log('%c[langMatrix] 開始；請先在模型選單選好本地 Ollama 模型', 'color:#3b82f6');
      console.log('[langMatrix] 預設跑兩輪；選項題用日期，搜尋題用動物。');
      console.log('[langMatrix] 人工目視：選項卡 label 語系正確、可點選；韓文不得吐日文片假名，也不得用英文句子混過。');
      for (let round = 0; round < rounds; round += 1) {
        console.group(`[langMatrix] Round ${round + 1}/${rounds}`);
        for (let i = 0; i < LANG_MATRIX.length; i += 1) {
          const c = LANG_MATRIX[i];
          try {
            await callWails(() => ApplyUIStyleDiff(JSON.stringify({ panel_language: '繁中', role_language: c.label })));
          } catch (e) {
            console.warn('[langMatrix] 切語言失敗', c.label, e);
          }
          await sleep(400);
          console.group(`[langMatrix] ${i + 1}/${LANG_MATRIX.length} ${c.label} / ${c.category}`);
          console.log('送出：', c.text);
          try {
            await submitComposerText(c.text);
          } catch (e) {
            console.error('[langMatrix] 送出失敗', e);
          }
          console.log(`等待 ${waitMs}ms 讓模型回覆與 UI 渲染…`);
          console.groupEnd();
          await sleep(waitMs);
        }
        console.groupEnd();
      }
      try {
        await callWails(() => ApplyUIStyleDiff(JSON.stringify({ panel_language: '繁中', role_language: '中文' })));
      } catch (e) { /* ignore */ }
      console.log('%c[langMatrix] 完成；角色語言已切回中文。請人工覆核選項卡與 debug trace。', 'color:#16a34a');
    };
    return () => {
      try { delete window.__runLangMatrix; } catch (e) { /* ignore */ }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
    const map = {
      [_t('settings.langZhTW')]: 'zh-TW',
      [_t('settings.langEn')]: 'en',
      [_t('settings.langJa')]: 'ja',
      [_t('settings.langPt')]: 'pt-PT',
      [_t('settings.langEs')]: 'es',
      [_t('settings.langTh')]: 'th',
      [_t('settings.langKo')]: 'ko',
      // fallback hardcoded labels
      '繁中': 'zh-TW', '中文': 'zh-TW', '英文': 'en', '日文': 'ja',
      'Traditional Chinese': 'zh-TW', 'Chinese': 'zh-TW', 'English': 'en', 'Japanese': 'ja',
      '中': 'zh-TW', 'en': 'en', 'ja': 'ja',
      'pt': 'pt-PT', 'pt-PT': 'pt-PT', 'es': 'es', 'th': 'th', 'ko': 'ko',
      'Português': 'pt-PT', '葡萄牙文': 'pt-PT', 'Español': 'es', '西班牙文': 'es', 'ไทย': 'th', '泰文': 'th', '한국어': 'ko', '韓文': 'ko', '韓語': 'ko', 'Korean': 'ko',
      // self-heal: raw i18n keys leaked by an older build
      'settings.langPt': 'pt-PT', 'settings.langEs': 'es', 'settings.langTh': 'th', 'settings.langKo': 'ko',
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
        sendSchedulerReply(`Ai:已寫入修改：「${normalized.name}」。你可以點右側「排程」查看摘要。`);
      } else {
        await callWails(() => CreateScheduledJob(
          normalized.name,
          normalized.cronExpr,
          'event',
          schedulerDefaultPayload(normalized.name, normalized.actionText || normalized.name, normalized.summary),
        ));
        sendSchedulerReply(`Ai:已建立排程：「${normalized.name}」。你可以點右側「排程」確認是否寫入正確。`);
      }
      resetSchedulerComposerState('排程已寫入');
      await Promise.all([refreshSchedulerClock(), refreshSchedulerJobs()]);
    } catch (error) {
      setSchedulerError(error?.message || String(error));
      sendSchedulerReply(`Ai:排程寫入失敗：${error?.message || String(error)}`);
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
    const message = 'Ai:好，我先取消這次排程確認。';
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
      sendSchedulerReply('Ai:好，我先取消這次排程設定。');
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
    setToolResult({toolId: 'scheduler', ok: false, message: SCHEDULER_DEGRADED_NOTICE_SHORT});
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
        sendSchedulerReply('Ai:我先打開排程清單，請確認要看或修改哪一筆。');
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
        sendSchedulerReply('Ai:我找不到你想修改的那一筆排程，先打開清單讓你確認編號。');
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
        out = out.replace(/^Ai:/, `Ai:${SCHEDULER_DEGRADED_NOTICE}\n\n`);
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
      sendWithNotice(`Ai:${intent.message || '我先打開排程清單，請確認要修改哪一筆。'}`);
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
    const [files, videos, images] = await Promise.all([
      callWails(ListReferenceFiles).catch(() => []),
      callWails(ListVideoFiles).catch(() => []),
      callWails(ListReferenceImages).catch(() => []),
    ]);
    const loadedFiles = [
      ...(Array.isArray(files) ? files : []),
      ...(Array.isArray(videos) ? videos : []),
      ...(Array.isArray(images) ? images : []),
    ];
    setReferenceFiles((current) => mergeReferenceLibraryFiles(current, loadedFiles));
    return loadedFiles;
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
        let referencePathForStatus = path;
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
            addedVia,
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
  }

  async function refreshLinkPreviewSuggestions(shouldSuggest) {
    if (!shouldSuggest) {
      setLinkPreviewSuggestions([]);
      return;
    }
    try {
      const detected = await callWails(AutoDetectCLI);
      const detectedSuggestions = (detected || [])
        .filter((item) => item?.found && item?.path && item?.supported !== false)
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
      groq: 'openai/gpt-oss-20b',
      together: 'meta-llama/Llama-3.3-70B-Instruct-Turbo',
      perplexity: 'sonar',
      cohere: 'command-a-03-2025',
      fireworks: 'accounts/fireworks/models/llama-v3p1-70b-instruct',
      cerebras: 'llama3.1-8b',
      huggingface: 'openai/gpt-oss-120b:cerebras',
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

  // v3.6: 學習模式切換 — 對接後端 LearningService
  async function toggleLearning() {
    const conversationId = activeConversationIdRef.current || 'main';
    const backendActive = await callWails(IsLearningModeActive).catch(() => learningEnabled);
    const wasEnabled = learningEnabled || !!backendActive;
    if (wasEnabled) {
      // 關閉學習模式
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
            `[系統] 示範結束，但後端停止記錄時回報錯誤。${detail ? ` ${detail}` : ''}`,
          ]);
        }
      }
      setLearningEnabled(false);
      setVlLearningActive(false);
      setVlActiveLearningRun(null);
      setLearningDigestReady(false);
      learningDigestReadyRef.current = false;
      try { window.localStorage.removeItem(learningDigestStorageKey); } catch { /* */ }
    } else {
      // 開啟學習模式（需要 activeWindowHash，目前用 session ID）
      try {
        const permissionStatus = await callWails(RequestVisualLearningPermissions).catch(() => null);
        if (permissionStatus?.missing?.length) {
          const opened = await openFirstMissingVisualLearningPermission(permissionStatus);
          setConversationMessages(conversationId, (prev) => [
            ...prev,
            formatVisualLearningPermissionStatus(permissionStatus, true) + (opened ? '\n已打開第一個缺少的權限設定頁。' : ''),
          ]);
        }
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
  const floatingPersona = activePersona || mainPersona;
  const floatingAvatarConfig = avatarConfigs[floatingPersona.id] || null;
  const floatingRenderedPixelAvatar = renderedPixelAvatars[floatingPersona.id] || '';
  const floatingAvatarSrc = resolvePersonaAvatarSrc(floatingPersona, floatingAvatarConfig, staticAvatarPreviews, floatingRenderedPixelAvatar);
  const activeFloatingAgentId = activeConversationId === 'main'
    ? floatingPersona.id || 'main'
    : activeConversationId || floatingPersona.id || 'main';
  const floatingAvatarDraft = floatingAvatarDrafts[activeFloatingAgentId] || '';
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
    floatingAvatarTransitionRef.current = true;
    try {
      const [windowSize, windowPosition] = await Promise.all([
        WindowGetSize(),
        WindowGetPosition(),
      ]);
      const avatarPosition = floatingAvatarPositionRef.current || floatingAvatarPosition || {
        x: Math.max(12, window.innerWidth - FLOATING_AVATAR_WINDOW_SIZE),
        y: Math.max(12, window.innerHeight - FLOATING_AVATAR_WINDOW_SIZE - 16),
      };
      const avatarX = Number(avatarPosition?.x ?? Math.max(12, window.innerWidth - FLOATING_AVATAR_WINDOW_SIZE));
      const avatarY = Number(avatarPosition?.y ?? Math.max(12, window.innerHeight - FLOATING_AVATAR_WINDOW_SIZE - 16));
      const compactX = Math.round((windowPosition?.x ?? 0) + avatarX - FLOATING_AVATAR_INSET);
      const compactY = Math.round((windowPosition?.y ?? 0) + avatarY - FLOATING_AVATAR_INSET);
      floatingAvatarWindowRef.current = {
        restore: {
          size: windowSize || {w: window.innerWidth, h: window.innerHeight},
          position: windowPosition || {x: 0, y: 0},
          avatarPosition: {x: avatarX, y: avatarY},
        },
        compactPosition: {x: compactX, y: compactY},
      };
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
    } catch (error) {
      console.warn('floating avatar compact window failed', error);
      floatingAvatarCompactWindowRef.current = false;
      setFloatingAvatarCompactWindow(false);
    } finally {
      floatingAvatarTransitionRef.current = false;
    }
    floatingAvatarModeRef.current = true;
    setFloatingAvatarMode(true);
    setSchedulerBgPrompt(false);
    setManualAvatarState('happy');
    setToolResult({toolId: 'scheduler', ok: true, message: t('floatingAvatar.entered')});
  }

  async function restoreFloatingAvatarWindow() {
    if (floatingAvatarTransitionRef.current) return;
    const restore = floatingAvatarWindowRef.current?.restore;
    const wasCompact = floatingAvatarCompactWindowRef.current || Boolean(restore);
    if (!wasCompact) return;
    floatingAvatarTransitionRef.current = true;
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
    const restoredAvatarPosition = restore?.avatarPosition
      || {
        x: Math.max(12, window.innerWidth - FLOATING_AVATAR_WINDOW_SIZE),
        y: Math.max(12, window.innerHeight - FLOATING_AVATAR_WINDOW_SIZE - 16),
      };
    floatingAvatarPositionRef.current = restoredAvatarPosition;
    setFloatingAvatarPosition(restoredAvatarPosition);
  }

  async function restoreFromFloatingAvatar(target = 'auto') {
    await restoreFloatingAvatarWindow();
    floatingAvatarModeRef.current = false;
    setFloatingAvatarMode(false);
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
    await restoreFloatingAvatarWindow();
    floatingAvatarModeRef.current = false;
    setFloatingAvatarMode(false);
    setManualAvatarState('');
    try {
      await callWails(() => ResolveSchedulerBackgroundPrompt(false));
    } catch {
      // 停用背景喚醒失敗時仍嘗試完整關閉，避免留下殘留浮窗/圖示。
    }
    await callWails(() => ConfirmClose(false, ''));
  }

  // 單擊頭像展開迷你框時，把浮窗放大到面板尺寸；關閉時縮回頭像尺寸。
  // 視窗會貼著頭像展開，並自動避開螢幕右/下邊緣，頭像螢幕位置維持不變。
  async function setCompactAvatarExpanded(open) {
    if (!floatingAvatarCompactWindowRef.current) return;
    const ref = floatingAvatarWindowRef.current;
    if (!ref) return;
    try {
      if (open) {
        if (ref.expanded) return;
        // compact 模式可由原生 draggable 移動，展開前先同步真實視窗位置。
        const livePosition = await WindowGetPosition().catch(() => null);
        const origin = Number.isFinite(livePosition?.x) && Number.isFinite(livePosition?.y)
          ? {x: Math.round(livePosition.x), y: Math.round(livePosition.y)}
          : ref.compactPosition || {x: 0, y: 0};
        ref.compactPosition = origin;
        const avatarScreenX = origin.x + FLOATING_AVATAR_INSET;
        const avatarScreenY = origin.y + FLOATING_AVATAR_INSET;
        const sw = window.screen?.availWidth || window.innerWidth || 1440;
        const sh = window.screen?.availHeight || window.innerHeight || 900;
        // 視窗展開後保留上方回覆泡泡、左側提示、右側迷你框與下方選項，
        // 並讓頭像螢幕位置在展開前後維持不變（不會跳）。
        const LEFT_ROOM = 236;
        const TOP_ROOM = 118;
        const expandedW = Math.min(FLOATING_AVATAR_PANEL_W, sw);
        const expandedH = Math.min(FLOATING_AVATAR_PANEL_H, sh);
        let newX = avatarScreenX - LEFT_ROOM;
        let newY = avatarScreenY - TOP_ROOM;
        if (newX < 0) newX = 0;
        if (newX + expandedW > sw) newX = Math.max(0, sw - expandedW);
        if (newY + expandedH > sh) newY = Math.max(0, sh - expandedH);
        if (newY < 0) newY = 0;
        const avatarLocalX = avatarScreenX - newX;
        const avatarLocalY = avatarScreenY - newY;
        ref.expanded = {avatarScreenX, avatarScreenY};
        WindowSetSize(expandedW, expandedH);
        WindowSetPosition(Math.round(newX), Math.round(newY));
        ref.compactPosition = {x: Math.round(newX), y: Math.round(newY)};
        const expandedAvatarPosition = {x: Math.round(avatarLocalX), y: Math.round(avatarLocalY)};
        floatingAvatarPositionRef.current = expandedAvatarPosition;
        setFloatingAvatarPosition(expandedAvatarPosition);
      } else {
        if (!ref.expanded) return;
        const {avatarScreenX, avatarScreenY} = ref.expanded;
        const newX = Math.round(avatarScreenX - FLOATING_AVATAR_INSET);
        const newY = Math.round(avatarScreenY - FLOATING_AVATAR_INSET);
        WindowSetSize(FLOATING_AVATAR_WINDOW_SIZE, FLOATING_AVATAR_WINDOW_SIZE);
        WindowSetPosition(newX, newY);
        ref.compactPosition = {x: newX, y: newY};
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

  function beginCompactFloatingAvatarDrag() {
    floatingAvatarDragWindowRef.current = floatingAvatarWindowRef.current?.compactPosition || null;
  }

  function moveCompactFloatingAvatarWindow(dx, dy) {
    const start = floatingAvatarDragWindowRef.current;
    if (!start) return;
    const next = {x: Math.round(start.x + dx), y: Math.round(start.y + dy)};
    floatingAvatarWindowRef.current = {
      ...floatingAvatarWindowRef.current,
      compactPosition: next,
    };
    WindowSetPosition(next.x, next.y);
  }

  function updateFloatingAvatarDraft(value) {
    setFloatingAvatarDrafts((prev) => ({...prev, [activeFloatingAgentId]: value}));
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

  async function switchFloatingAvatarAgent(personaId) {
    const persona = settingsState.personas.find((item) => item.id === personaId);
    if (!persona) return;
    await savePersonaPatch(persona.id, persona);
    await loadCurrentAvatar(persona.id);
    setToolResult({toolId: persona.id, ok: true, message: t('floatingAvatar.agentSwitched', {name: persona.name || t('floatingAvatar.agentFallback')})});
  }

  async function submitFloatingAvatarText(rawText) {
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
  const floatingStatusText = stripComposerPendingMarker(pendingTaskReview?.title
    || schedulerConfirm?.title
    || toolResult?.message
    || state.statusRail?.text
    || state.greeting
    || '');
  const floatingLatestText = stripComposerPendingMarker(messages[messages.length - 1] || state.greeting || '');
  // 只在「需要主動提醒」時才浮出上方泡泡（待確認/排程確認）。
  // 進後台、一般狀態、問候語都不浮泡泡，維持乾淨頭像；人格名稱/回覆改由點開迷你框顯示。
  const floatingBubbleText = stripComposerPendingMarker(pendingTaskReview?.reason
    || pendingTaskReview?.title
    || schedulerConfirm?.reason
    || schedulerConfirm?.title
    || '');

  return (
    <div
      className={`console-shell ${activePanel === 'settings' ? 'settings-open' : ''} ${floatingAvatarMode ? 'floating-avatar-shell-active' : ''}`}
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
        avatarExpression={!floatingReminderPaused && (pendingTaskReview || schedulerConfirm) ? 'warning' : avatarExpression}
        persona={floatingPersona}
        personas={floatingAvatarPersonas}
        adapterLabel={floatingAdapter?.name || floatingAdapter?.Name || activeAdapterId || ''}
        position={floatingAvatarPosition}
        avatarSize={FLOATING_AVATAR_SIZE}
        onPositionChange={updateFloatingAvatarPosition}
        compactWindowMode={floatingAvatarCompactWindow}
        onCompactDragStart={beginCompactFloatingAvatarDrag}
        onCompactDrag={moveCompactFloatingAvatarWindow}
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
        shakeDialogueOptions={getGreetingRotationOptions(personaGreetingVariant(floatingPersona))}
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
                  import('../../wailsjs/go/main/App').then((m) => m.PurgeMessageMarks?.(convId, messageDomId(target, index, messages))).catch(() => {});
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
              onVoicePressStart={startVoiceRecording}
              onVoicePressEnd={stopVoiceRecording}
              onVoiceCancel={cancelVoiceRecording}
              taskActive={isTaskProgressActive(dagRun) || !!activeChatTrace}
              onCancelTask={cancelActiveExecution}
              pendingTaskReview={pendingTaskReview}
              taskReviewDetailsOpen={reviewPopup === 'risk'}
              onConfirmTaskReview={confirmSkillBuild}
              onCancelTaskReview={() => cancelActiveTaskProgress('review_cancel')}
              onShowTaskReviewDetails={() => setReviewPopup((current) => current === 'risk' ? null : 'risk')}
              composerConfirmAction={buildSchedulerComposerConfirmAction(schedulerConversation, schedulerBusy)}
              onComposerConfirm={confirmComposerAction}
              onComposerCancel={cancelComposerAction}
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
                  try {
                    await closeAfterBackgroundAvatarExit();
                  } catch (error) {
                    setToolResult({toolId: 'session-close', ok: false, message: t('subagent.closeFail', { error: error?.message || error })});
                  } finally {
                    setSchedulerBgBusy(false);
                    setSchedulerBgPrompt(false);
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
    return {mode: 'execute', query: normalizeLearningOperationQuery(target) || target, raw: target};
  }
  return null;
}

function isLastLearningOperationReference(operationIntent) {
  const haystack = `${operationIntent?.raw || ''} ${operationIntent?.query || ''}`.toLowerCase();
  if (!haystack.trim()) return false;
  return /上次|上一個|上一筆|剛剛|剛才|再一次|再執行一次|重跑一次|重新.*一次/.test(haystack)
    || /last|previous|again|one more time|rerun/.test(haystack);
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
  const stepCount = Number(run?.step_count ?? run?.StepCount ?? 0);
  if (stepCount <= 0) {
    return 'Ai:這次示範已停止，但沒有錄到任何可回放步驟。請確認「輸入監控」仍允許 ai-console，重新啟動 app，開始示範後先點一下或輸入一段文字再停止。';
  }
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
    const inputInfo = step.action === 'text'
      ? `，文字: ${step.sensitive ? '敏感佔位' : `${String(step.text || '').length} 字`}`
      : step.action === 'shortcut'
        ? `，快捷鍵: ${(step.modifiers || []).join('+')}${step.modifiers?.length ? '+' : ''}${step.key || ''}`
        : '';
    return `${index + 1}. ${step.action || 'click'} ${target}，${coordLabel} (${step.x}, ${step.y})${inputInfo}${nativeInfo}${selector}${anchor}`;
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
  const textSteps = steps.filter((step) => step.action === 'text' || step.action === 'sensitive_text');
  const windows = Array.from(new Map(nativeSteps.map((step) => {
    const label = `${basenameForDisplay(step.window_process || step.tag || 'unknown')} - ${step.window_title || 'unknown window'}`;
    return [label, label];
  })).values()).slice(0, 4);
  const lines = [
    `即將依照剛剛的示範執行 ${steps.length} 個步驟。`,
    `本視窗 selector ${domSteps} 步，外部 native click ${nativeSteps.length} 步。`,
  ];
  if (textSteps.length) {
    lines.push(`包含文字/敏感輸入 ${textSteps.length} 步；需確認的文字會先停下來。`);
  }
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
    `Ai:Replay executor 執行完成：${okCount}/${steps.length} 步成功。`,
    `selector 命中 ${selectorCount} 步，native 座標 ${nativeOK}/${nativeTotal} 步${nativeTotal ? `（前景確認 ${nativeForegroundOK}/${nativeTotal}）` : ''}，座標 fallback ${coordinateCount} 步。`,
    `文字輸入成功 ${textOK} 步，快捷鍵/按鍵成功 ${keyOK} 步。`,
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
      error: '敏感文字未保存；需透過授權的密碼/憑證 App 填入',
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
      error: `尚未支援 ${action}`,
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
        error: '座標落在系統控制項，已略過',
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
      error: '文字輸入需要確認後才回放',
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
      error: '找不到可輸入文字的元素',
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

function clampReplayCoordinate(value, min, max) {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, value));
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

function referenceFileStatusLabel(status, translate = _t) {
  return {
    importing: translate('rightRail.refStatusImporting'),
    checking: translate('rightRail.refStatusChecking'),
    ready: translate('rightRail.refStatusReady'),
    error: translate('rightRail.refStatusError'),
  }[status] || translate('rightRail.refStatusAdded');
}

function shouldShowReferenceFileDetail(file) {
  if (!file?.detail) return false;
  return !(file.source === 'library' && file.status === 'ready');
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
  if (pack === 'wolf') return wolfdogAvatarUrls[state] || wolfdogAvatarUrls.idle;
  if (pack === 'uncle') return uncleBustAvatarUrls[state] || uncleBustAvatarUrls.idle;
  if (pack === 'secretary') return secretaryAvatarUrls[state] || secretaryAvatarUrls.idle;
  if (pack === 'police') return policeAvatarUrls[state] || policeAvatarUrls.idle;
  if (pack === 'touharu') return touharuAvatarUrls[state] || touharuAvatarUrls.idle;
  return '';
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
  if (monitorLinkCache?.url) {
    return Promise.resolve(monitorLinkCache.url);
  }
  if (!monitorLinkPending) {
    monitorLinkPending = Promise.resolve()
      .then(() => GetMonitorLinks?.())
      .then((link) => {
        monitorLinkCache = link || {url: ''};
        return monitorLinkCache.url;
      })
      .catch(() => {
        monitorLinkCache = {url: ''};
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


/* i18n: map backend-persisted Chinese labels to current locale display labels.
   The Go backend stores display strings as-is (always Chinese on first save). */
// 版型 → 字型堆疊（family 名稱需與 fontFaces.js 一致）。fontDefault 不覆寫，沿用語言預設字。
const FONT_PRESET_STACKS = {
  'settings.fontNormal': "'Inter','Noto Sans TC','Noto Sans JP',sans-serif",
  'settings.fontHand':   "'Caveat','Klee One','LXGW WenKai','Noto Sans TC','Noto Sans JP',cursive",
  'settings.fontCalli':  "'Dancing Script','LXGW WenKai','Yuji Syuku','Noto Serif TC','Noto Sans JP',serif",
  'settings.fontRound':  "'Fredoka','jf-openhuninn','Noto Sans JP','Noto Sans TC',sans-serif",
  'settings.fontMono':   "'JetBrains Mono','Noto Sans TC','Noto Sans JP',monospace",
};

// 把已儲存的版型值（可能是任一語言的顯示字串）正規化為穩定 key。
function fontPresetKey(value) {
  if (!value) return 'settings.fontDefault';
  if (_fontPresetLabelMap[value]) return _fontPresetLabelMap[value];
  for (const k of new Set(Object.values(_fontPresetLabelMap))) {
    if (_t(k) === value) return k;
  }
  return 'settings.fontDefault';
}

// 版型 → 套在根元素的 CSS 變數（覆寫字體 family）。fontDefault 回傳空物件。
function fontPresetVars(value) {
  const stack = FONT_PRESET_STACKS[fontPresetKey(value)];
  if (!stack) return {};
  return { '--font-console': stack, '--i18n-font': stack };
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
  const personas = normalizeLockedPersonas(
    merged.personas || fallbackSettings.personas,
    merged.removedDefaultPersonaIds || [],
  );
  const activePersonaId = personas.some((persona) => persona.id === merged.activePersonaId)
    ? merged.activePersonaId
    : lockedPersonaId;
  return {...merged, activePersonaId, personas, panel: normalizePanelSettings(merged.panel)};
}

function normalizeLockedPersonas(personas = [], removedDefaultPersonaIds = []) {
  // "憂樂傻酷" is a reserved persona identity, not a reserved slot. Keep its
  // name stable while preserving whatever ordering the user chose, because the
  // app treats the first card as the current main persona.
  const normalized = personas.map((persona) => {
    if (persona.id === lockedPersonaId) {
      return {...persona, name: lockedPersonaName, identity: persona.identity || _t('persona.defaultIdentityA')};
    }
    return normalizeBuiltInPersonaCopy(persona);
  });
  const lockedIndex = normalized.findIndex((persona) => persona.id === lockedPersonaId);
  if (lockedIndex < 0) {
    return appendMissingDefaultPersonas([fallbackSettings.personas[0], ...normalized], removedDefaultPersonaIds);
  }
  return appendMissingDefaultPersonas(normalized, removedDefaultPersonaIds);
}

const defaultPersonaCopy = {
  'persona-d': {
    name: () => _t('persona.defaultNameD'),
    identity: () => _t('persona.defaultIdentityD'),
    personality: () => _t('persona.defaultPersonalityD'),
    legacyNames: ['人格 D', '規則警察', '警察桂澤', 'Rule Police', 'Officer Reggie Law', 'Agente Reglaz', 'Agente Regraldo', '木曽久巡査', '규식 순경', 'ผู้หมวดกฎเก่ง'],
    legacyIdentities: [
      '循規蹈矩、嚴格守序、看到違規就會說教的警察助手',
      '循規蹈矩的警察助手；「桂澤」聽起來像規則，看到流程被跳過就會立刻吹哨。',
      'A strict rule-following police assistant who lectures when rules are bent',
      'Rules-first police assistant who lectures when rules are bent',
    ],
    legacyPersonalities: [
      '重視規則、流程與安全，回答會先提醒限制與責任，語氣像警察一樣嚴肅但可靠。',
      '把規則、流程與安全放第一，回答會先提醒限制、責任與風險；嚴肅但可靠，說教時也有一點冷面幽默。',
      'Prioritizes rules, process, and safety. Replies first with constraints and responsibility, stern like a police officer but reliable.',
    ],
  },
  'persona-e': {
    name: () => _t('persona.defaultNameE'),
    identity: () => _t('persona.defaultIdentityE'),
    personality: () => _t('persona.defaultPersonalityE'),
    legacyNames: ['東春巫女', 'Touharu Miko', 'Miko Touharu', '東春の巫女', '동춘 무녀', 'มิโกะโทฮารุ'],
    legacyIdentities: [
      '白髮馬尾、棕眼曬黑、直覺敏銳的巫女助手',
      '白髮馬尾、棕眼曬黑、直覺敏銳的東春巫女助手',
      'A tan white-ponytailed miko assistant with sharp intuition',
      'Tan-skinned white-ponytailed miko assistant with sharp intuition',
    ],
    legacyPersonalities: [
      '傲嬌、淘氣又直覺敏銳，嘴上不饒人，但能很快察覺問題不對勁。',
      '傲嬌又淘氣，嘴上不饒人，但能很快察覺問題不對勁，像春雷一樣先提醒你。',
      'Tsundere and mischievous, with quick instincts and a habit of noticing when something feels off.',
      'Tsundere and mischievous, but highly intuitive and quick to sense when something is wrong.',
    ],
  },
};

function isLegacyDefaultCopy(value, legacyValues = []) {
  const text = String(value || '').trim();
  return text === '' || legacyValues.includes(text);
}

function normalizeBuiltInPersonaCopy(persona = {}) {
  const copy = defaultPersonaCopy[persona.id];
  if (!copy) return persona;
  return {
    ...persona,
    name: isLegacyDefaultCopy(persona.name, copy.legacyNames) ? copy.name() : persona.name,
    identity: isLegacyDefaultCopy(persona.identity, copy.legacyIdentities) ? copy.identity() : persona.identity,
    personality: isLegacyDefaultCopy(persona.personality, copy.legacyPersonalities) ? copy.personality() : persona.personality,
  };
}

function appendMissingDefaultPersonas(personas = [], removedDefaultPersonaIds = []) {
  const known = new Set(personas.map((persona) => persona.id));
  const removed = new Set(removedDefaultPersonaIds || []);
  const seeds = fallbackSettings.personas.filter((persona) => (
    persona.id !== lockedPersonaId && !known.has(persona.id) && !removed.has(persona.id)
  ));
  return seeds.length > 0 ? [...personas, ...seeds] : personas;
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
  if (personaID === 'persona-d') return 'police';
  if (personaID === 'persona-e') return 'touharu';
  return 'wolf';
}

function pixelPackForPersona(persona, config) {
  const pack = config?.pixel_pack || config?.PixelPack || '';
  if (['wolf', 'uncle', 'secretary', 'police', 'touharu'].includes(pack)) return pack;
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
      </nav>
      <div className="highlight-groups-anchor" data-highlight-groups-anchor />
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
  const activeAvatarConfig = avatarConfigs[activePersona.id] || null;
  const activeAvatarProvider = resolveAvatarProvider(activeAvatarConfig);
  const activeRenderedPixelAvatar = renderedPixelAvatars[activePersona.id] || '';
  const activeAvatarSrc = resolvePersonaAvatarSrc(activePersona, activeAvatarConfig, staticAvatarPreviews, activeRenderedPixelAvatar);
  const activeReplyStrategy = activePersona.replyStrategy || '';
  const activeReplyStrategyIsPreset = !activeReplyStrategy || Boolean(replyStrategyPresetFor(activeReplyStrategy));
  const activeChatTone = activePersona.personality || '';
  const activeChatToneIsPreset = Boolean(chatTonePresetFor(activeChatTone));
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

function TopConsole({
  activePersona, activeAvatarConfig, avatarExpression, avatarModeNotice, avatarProvider, avatarSrc,
  browserPref, greeting, haoras, subagentTabs, personaJob, personaName,
  dagRun, reviewState = fallbackReviewState, reviewPopup, skillInjections = [], snoozeHours,
  systemStatusHistory = [], reviewArchive = [],
  showSkillFirstUseCard, onDismissSkillFirstUse,
  w3aImportPopup, w3aDetail, w3aPollutionResult, w3aTransferGuidance, w3aTrustList = [],
  w3aActionBusy = '', w3aActionError = '', w3aStatusConfig = {}, w3aToastMsg,
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
  panelTheme = 'onanegiku', taskLoopRounds = {}, taskLoopReply = {}, onTaskLoopReplyChange = () => {},
  subExportCapabilities,
  activeHaoraId, onHaoraSelect, onRenameHaora, onHaorasReordered, onSubagentsChanged,
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

function MicIcon() {
  return (
    <svg className="mic-icon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 3.5c-1.7 0-3 1.3-3 3v5c0 1.7 1.3 3 3 3s3-1.3 3-3v-5c0-1.7-1.3-3-3-3Z" />
      <path d="M6.5 10.5c0 3 2.4 5.5 5.5 5.5s5.5-2.5 5.5-5.5M12 16v3.5M9 20.5h6" />
    </svg>
  );
}

function localizeChatSystemMessage(message, chatLocale = 'zh-TW') {
  const raw = String(message || '');
  const body = raw.replace(/^Ai:/, '').trim();
  if (raw.startsWith('Ai:') && body === '請選擇：') {
    return `Ai:${tForLanguage(chatLocale, 'chatSystem.choose')}`;
  }
  return raw;
}

function ConversationPanel({
  messages, personaName, draft, onDraftChange, onSend, onDelete, onSummarizeSearch, onExportSearchSummary,
  onInjectText, activeConversationId, chatLocale = 'zh-TW',
  // v3.6.4 Readiness Gate props
  readinessGate = fallbackReadinessGate,
  selectedFloatingCandidateIDs = [],
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
  composerConfirmAction = null, onComposerConfirm, onComposerCancel,
}) {
  const t = useI18n(s => s.t);
  const [activeMessage, setActiveMessage] = useState(null);
  const [imagePreviews, setImagePreviews] = useState([]);
  const [imageError, setImageError] = useState('');
  const MAX_IMAGE_BYTES = 8 * 1024 * 1024; // 8MB：擋住整張截圖把記憶體灌爆
  const composerComposingRef = useRef(false);
  const taskReviewCardRef = useRef(null);
  const imageInputRef = useRef(null);
  const voiceReady = voiceState?.status === 'ready';
  const micAvailable = typeof navigator !== 'undefined' && !!navigator.mediaDevices?.getUserMedia;
  const voiceDisabled = voiceBusy || !voiceReady || !micAvailable;
  const voiceTitle = voiceReady ? t('composer.voiceHold') : voiceStatusLabel(voiceState?.status);
  const hasSelectedFloatingCandidates = selectedFloatingCandidateIDs.length > 0;
  const composerReady = !!draft.trim() || hasSelectedFloatingCandidates;
  const displayMessage = (message) => localizeChatSystemMessage(stripComposerPendingMarker(message), chatLocale)
    .replace(/^Ai:/, personaName + ':')
    .replace(/^輸入:/, '');

  const messageKind = (message) => {
    if (message.startsWith('Ai:')) return 'ai';
    return 'user';
  };

  function addImageFiles(fileList) {
    const files = Array.from(fileList || []).filter((file) => file?.type?.startsWith('image/'));
    if (files.length === 0) return;
    files.forEach((file) => {
      if (file.size > MAX_IMAGE_BYTES) {
        // 大圖直接擋下：避免 base64 在 state 與 DOM 各塞一份把記憶體灌爆、UI 卡頓。
        setImageError(t('composer.imageTooLarge', { max: '8MB' }));
        return;
      }
      const reader = new FileReader();
      reader.onload = (event) => {
        setImagePreviews((prev) => [...prev, {
          id: `img-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
          src: event.target?.result || '',
          name: file.name || 'image',
          type: file.type,
        }]);
        setImageError('');
      };
      reader.readAsDataURL(file);
    });
  }

  function removeImage(id) {
    setImagePreviews((prev) => prev.filter((img) => img.id !== id));
  }

  function handleComposerPaste(event) {
    const files = Array.from(event.clipboardData?.files || []).filter((file) => file?.type?.startsWith('image/'));
    if (files.length === 0) return;
    event.preventDefault();
    addImageFiles(files);
  }

  function handleComposerDrop(event) {
    const files = Array.from(event.dataTransfer?.files || []).filter((file) => file?.type?.startsWith('image/'));
    if (files.length === 0) return;
    event.preventDefault();
    addImageFiles(files);
  }

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
    <section
      className="conversation-panel"
      onDragOver={(event) => { if (event.dataTransfer?.types?.includes('Files')) event.preventDefault(); }}
      onDrop={(event) => { if (event.dataTransfer?.types?.includes('Files')) event.preventDefault(); }}
    >
      <div className="message-list" ref={messageListRef}>
        {messages.map((message, index) => (
          <MessageRow
            key={`${message}-${index}`}
            message={message}
            displayText={displayMessage(message)}
            kind={messageKind(message)}
            isActive={activeMessage === index}
            index={index}
            domMessageId={messageDomId(message, index, messages)}
            summaryLabel={t('composer.summary')}
            deleteLabel={t('composer.delete')}
            onActivate={() => setActiveMessage(index)}
            onDelete={(rowIndex) => {
              onDelete(rowIndex);
              setActiveMessage(null);
            }}
            onSummarizeSearch={onSummarizeSearch}
            onExportSearchSummary={onExportSearchSummary}
            onInjectText={onInjectText}
            sessionId={activeConversationId}
            previousMessage={messages[index - 1] || ''}
            chatLocale={chatLocale}
          />
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
            selectedIDs={selectedFloatingCandidateIDs}
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

      <ComposerConfirmBubble
        action={composerConfirmAction}
        onConfirm={onComposerConfirm}
        onCancel={onComposerCancel}
      />

      {/* ── 原有 Composer（聊天輸入區）── */}
      <form
        className="composer"
        onSubmit={(event) => {
          // submitComposerText 對空字串會 return；只有真的有文字送出時才連帶清縮圖，
          // 純貼圖尚未送出時保留，避免誤刪使用者剛貼上的圖。
          onSend(event, imagePreviews);
          if (draft.trim() || hasSelectedFloatingCandidates) {
            setImagePreviews([]);
            setImageError('');
          }
        }}
      >
        {imagePreviews.length > 0 && (
          <div className="composer-image-strip">
            {imagePreviews.map((img) => (
              <div className="composer-image-thumb" key={img.id}>
                <img src={img.src} alt={img.name || t('composer.imagePreview')} />
                <button type="button" onClick={() => removeImage(img.id)} aria-label={t('composer.removeImage')}>×</button>
              </div>
            ))}
          </div>
        )}
        {imageError && <div className="composer-image-error">{imageError}</div>}
        <button
          className={`attach-btn task-stop-btn ${taskActive ? 'task-stop-active' : ''}`}
          type="button"
          title={taskActive ? t('composer.stopTask') : t('composer.noActiveTask')}
          aria-label={taskActive ? t('composer.stopTask') : t('composer.noActiveTask')}
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
        <button
          className="image-insert-btn"
          type="button"
          title={t('composer.insertImage')}
          aria-label={t('composer.insertImage')}
          onClick={() => imageInputRef.current?.click()}
        >
          <span aria-hidden="true">▧</span>
        </button>
        <input
          ref={imageInputRef}
          className="composer-image-file"
          type="file"
          accept="image/*"
          multiple
          onChange={(event) => {
            addImageFiles(event.target.files);
            event.target.value = '';
          }}
        />
        <div
          className="input-wrap"
          onDrop={handleComposerDrop}
          onDragOver={(event) => event.preventDefault()}
        >
          <textarea
            value={draft}
            rows={1}
            placeholder={t('composer.placeholder')}
            onChange={(event) => onDraftChange(event.target.value)}
            onPaste={handleComposerPaste}
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
          <button className={`send-btn ${composerReady ? 'send-btn-enabled' : ''} ${hasSelectedFloatingCandidates ? 'send-btn-ready' : ''}`} type="submit"><span>◢</span>{t('composer.send')}</button>
        </div>
        {(voiceStatus || voiceError || voiceState?.status !== 'ready') && (
          <div className={`voice-status ${voiceError ? 'voice-status-error' : ''}`}>
            {voiceError || voiceStatus || voiceStatusLabel(voiceState?.status)}
          </div>
        )}
        <div className="composer-ai-disclaimer">
          {t('composer.aiDisclaimer')}
        </div>
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
                          {`第 ${taskLoopRounds[node.id].iteration} 輪：${taskLoopRounds[node.id].action} ${taskLoopRounds[node.id].target}`}
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
// 模型失效退回本機 regex 時，給使用者的明確提示，避免誤以為模型變笨。
const SCHEDULER_DEGRADED_NOTICE = '（提示：模型暫時無法使用，已改用本機規則解析排程，結果可能較簡略；模型恢復後會自動回到智慧判斷。）';
const SCHEDULER_DEGRADED_NOTICE_SHORT = '模型暫時無法使用，已改用本機規則';

const schedulerWeekdayMap = [
  ['日', 0], ['天', 0], ['一', 1], ['二', 2], ['三', 3], ['四', 4], ['五', 5], ['六', 6],
];
const schedulerSlotLabels = {name: '標題', time: '時間'};
const schedulerChineseNumberMap = {
  零: 0,
  〇: 0,
  一: 1,
  二: 2,
  兩: 2,
  三: 3,
  四: 4,
  五: 5,
  六: 6,
  七: 7,
  八: 8,
  九: 9,
};

function isSchedulerAffirmation(text) {
  return /^(可以|可以了|好|好的|確認|確定|沒問題|ok|okay|yes|建立|寫入|送出|對|對的)$/i.test(String(text || '').trim());
}

function isSchedulerCancellation(text) {
  return /^(取消|不要了|先不用|算了|停止)$/i.test(String(text || '').trim());
}

function schedulerDefaultPayload(title, actionText = '', summary = '') {
  const cleanTitle = title || actionText || '提醒';
  const cleanAction = actionText || cleanTitle;
  const cleanSummary = summary || `在排程時間執行：${cleanAction}`;
  return JSON.stringify({
    event_name: 'scheduler:reminder',
    data: {
      title: cleanTitle,
      action: cleanAction,
      summary: cleanSummary,
    },
  });
}

function parseSchedulerChineseNumber(text) {
  const raw = String(text || '').trim();
  if (!raw) return null;
  if (/^\d+$/.test(raw)) return Number(raw);
  if (Object.prototype.hasOwnProperty.call(schedulerChineseNumberMap, raw)) return schedulerChineseNumberMap[raw];
  if (raw === '十') return 10;
  const tenIndex = raw.indexOf('十');
  if (tenIndex < 0) return null;
  const left = raw.slice(0, tenIndex);
  const right = raw.slice(tenIndex + 1);
  const tens = left ? schedulerChineseNumberMap[left] : 1;
  const ones = right ? schedulerChineseNumberMap[right] : 0;
  if (tens == null || ones == null) return null;
  return tens * 10 + ones;
}

function normalizeSchedulerTimeNumerals(text) {
  return String(text || '').replace(
    /([零〇一二兩三四五六七八九十]{1,3})(?=\s*(?:點|:|：|分|日|號))/g,
    (match) => {
      const parsed = parseSchedulerChineseNumber(match);
      return parsed == null ? match : String(parsed);
    },
  );
}

function hasSchedulerClockText(text) {
  return /(?:上午|早上|中午|下午|晚上)?\s*\d{1,2}(?:[:：]\d{2}|點(?:半|\d{1,2}分?)?)/.test(normalizeSchedulerTimeNumerals(text));
}

function hasSchedulerTimeText(text) {
  const raw = normalizeSchedulerTimeNumerals(text);
  const hasClock = hasSchedulerClockText(raw);
  const hasWeekly = /(?:每\s*)?(?:週|周|星期|禮拜)[日天一二三四五六]/.test(raw);
  const hasMonthly = /每\s*月\s*\d{1,2}\s*(?:日|號)?/.test(raw);
  return /@(?:hourly|daily|weekly|monthly|yearly)/i.test(raw)
    || /\d{1,2}\s+\d{1,2}\s+\*|\*\s+\*\s+\*/.test(raw)
    || /每\s*(小時|一小時|個小時)|hourly/i.test(raw)
    || ((/每天|每日|每\s*(天|日)/.test(raw) || hasWeekly || hasMonthly) && hasClock);
}

function stripSchedulerTimeText(text) {
  return normalizeSchedulerTimeNumerals(text)
    .replace(/@(?:hourly|daily|weekly|monthly|yearly)/ig, ' ')
    .replace(/(?:上午|早上|中午|下午|晚上)?\s*\d{1,2}(?:[:：]\d{2}|點(?:半|\d{1,2}分?)?)?/g, ' ')
    .replace(/每\s*(小時|一小時|個小時)|每天|每日|每週|每周|每月|星期[日天一二三四五六]|禮拜[日天一二三四五六]|週[日天一二三四五六]|周[日天一二三四五六]/g, ' ')
    .replace(/每\s*月\s*\d{1,2}\s*(?:日|號)?/g, ' ');
}

function extractSchedulerDeliveryText(text) {
  const raw = String(text || '').trim();
  const match = raw.match(/(?:幫我|請|再|並|然後)?\s*(?:做成|整理成|產出|生成|製作成)\s*(?:一個|一份|一張|一套)?\s*([^，,。；;\n]+)/);
  if (!match?.[1]) return '';
  return `整理成${match[1].trim()}`;
}

function removeSchedulerDeliveryClauses(text) {
  return String(text || '')
    .replace(/(?:，|,|。|；|;)?\s*(?:幫我|請|再|並|然後)?\s*(?:做成|整理成|產出|生成|製作成)\s*(?:一個|一份|一張|一套)?\s*[^，,。；;\n]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function parseSchedulerTimeText(text, fallback = '') {
  const raw = normalizeSchedulerTimeNumerals(text);
  const shortcut = raw.match(/@(hourly|daily|weekly|monthly|yearly)/i);
  if (shortcut) return `@${shortcut[1].toLowerCase()}`;
  const cron = raw.match(/(?:^|\s)(\S+\s+\S+\s+\S+\s+\S+\s+\S+)(?:\s|$)/);
  if (cron && /^[\d*,\-\/]+\s+[\d*,\-\/]+\s+[\d*,\-\/]+\s+[\d*,\-\/]+\s+[\d*,\-\/]+$/.test(cron[1])) return cron[1];
  const hasClock = hasSchedulerClockText(raw);
  if (!hasSchedulerTimeText(raw) && !hasClock) return fallback;
  const timeMatch = raw.match(/(?:上午|早上|中午|下午|晚上)?\s*(\d{1,2})(?:[:：](\d{2})|點(?:(半)|(\d{1,2})分?)?)/);
  let hour = 9;
  let minute = 0;
  if (timeMatch) {
    hour = Math.min(23, Math.max(0, Number(timeMatch[1])));
    minute = timeMatch[2]
      ? Math.min(59, Math.max(0, Number(timeMatch[2])))
      : (timeMatch[3] ? 30 : (timeMatch[4] ? Math.min(59, Math.max(0, Number(timeMatch[4]))) : 0));
    if ((raw.includes('下午') || raw.includes('晚上')) && hour < 12) hour += 12;
    if (raw.includes('中午') && hour < 11) hour += 12;
  }
  if (/每\s*(小時|一小時|個小時)|hourly/i.test(raw)) return '@hourly';
  const weekdayToken = schedulerWeekdayMap.find(([label]) => new RegExp(`(?:每\\s*)?(?:週|周|星期|禮拜)${label}`).test(raw));
  if (weekdayToken) return `${minute} ${hour} * * ${weekdayToken[1]}`;
  const monthMatch = raw.match(/每\s*月\s*(\d{1,2})\s*(?:日|號)?/);
  if (monthMatch) return `${minute} ${hour} ${Math.min(31, Math.max(1, Number(monthMatch[1])))} * *`;
  if (/每天|每日|daily/i.test(raw)) return `${minute} ${hour} * * *`;
  if (hasClock) return `${minute} ${hour} * * *`;
  return fallback;
}

function cleanSchedulerNameText(text) {
  const raw = removeSchedulerDeliveryClauses(stripSchedulerTimeText(text));
  const cleaned = String(raw || '')
    .replace(/我想要|想要|我要|我想|請|幫我|幫忙|新增|建立|規劃|排程|排定|安排|任務|提醒/g, ' ')
    .replace(/一個|一項|一筆|一下/g, ' ')
    .replace(/時間(改成|改為|為|是).*/g, ' ')
    .replace(/動作(改成|改為|為|是).*/g, ' ')
    .replace(/的$/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  return cleaned.slice(0, 32);
}

function parseSchedulerActionText(text) {
  const raw = removeSchedulerDeliveryClauses(String(text || '').trim());
  const channelMatch = raw.match(/(?:要)?(?:用|以)\s*(.+?)\s*(?:提醒我|通知我|叫我)$/);
  if (channelMatch?.[1]) return `${channelMatch[1].trim()}提醒`;
  const remindMeSuffix = raw.match(/(.+?)\s*(?:提醒我|通知我|叫我)$/);
  if (remindMeSuffix?.[1]) return remindMeSuffix[1].trim();
  const notifyMatch = raw.match(/(?:提醒我|叫我|通知我)\s*(.+)$/);
  if (notifyMatch?.[1]) return notifyMatch[1].trim();
  const runMatch = raw.match(/(?:執行|做|跑)\s*(.+)$/);
  if (runMatch?.[1]) return runMatch[1].trim();
  const actionMatch = raw.match(/動作(?:改成|改為|為|是|:|：)\s*(.+)$/);
  if (actionMatch?.[1]) return actionMatch[1].trim();
  const quoted = raw.match(/[「『"]([^」』"]+)[」』"]/);
  if (quoted?.[1]) return quoted[1].trim();
  return cleanSchedulerNameText(raw);
}

function formatSchedulerSummary(draft) {
  const delivery = draft?.deliveryText || extractSchedulerDeliveryText(draft?.sourceText || '');
  const task = String(draft?.actionText || draft?.name || '提醒').trim();
  const output = delivery ? `，${delivery}` : '';
  return `在設定時間執行「${task}」${output}。`;
}

function normalizeSchedulerDraft(draft) {
  const actionText = String(draft.actionText || '').trim();
  const name = draft.name || actionText;
  const summary = String(draft.summary || '').trim();
  const finalSummary = summary || formatSchedulerSummary({...draft, name, actionText});
  return {
    name: String(name || '').trim(),
    cronExpr: String(draft.cronExpr || '').trim(),
    actionText,
    summary: finalSummary,
    deliveryText: String(draft.deliveryText || '').trim(),
    sourceText: String(draft.sourceText || '').trim(),
    actionPayload: draft.actionPayload || (name || actionText ? schedulerDefaultPayload(name || '提醒', actionText || name, finalSummary) : ''),
  };
}

function schedulerMissingSlots(draft) {
  const normalized = normalizeSchedulerDraft(draft || {});
  return ['name', 'time'].filter((slot) => {
    if (slot === 'name') return !normalized.name;
    if (slot === 'time') return !normalized.cronExpr;
    return false;
  });
}

function schedulerQuestionForMissing(missing) {
  const labels = missing.map((slot) => schedulerSlotLabels[slot]).filter(Boolean);
  return `Ai:目前缺少「${labels.join('、')}」，請幫我補齊。`;
}

function schedulerConfirmationMessage(draft, mode = 'create', job = null) {
  const normalized = normalizeSchedulerDraft(draft);
  const title = mode === 'update' ? `準備修改排程「${job?.name || normalized.name}」` : '準備寫入排程';
  const displayTime = formatSchedulerCronNextTime(normalized.cronExpr);
  return [
    `Ai:${title}，請確認：`,
    `標題：${normalized.name}`,
    `時間：${displayTime}`,
    `摘要：${normalized.summary}`,
    '如果正確，按下方「確定」或回覆「可以」我就寫入；要改的話直接說「改時間為…」或「改標題為…」。',
  ].join('\n');
}

function buildSchedulerComposerConfirmAction(conversation, busy = false) {
  if (!conversation || conversation.phase !== 'confirm') return null;
  const normalized = normalizeSchedulerDraft(conversation.draft);
  return {
    type: 'scheduler',
    title: conversation.mode === 'update' ? '確認修改排程' : '確認寫入排程',
    primaryLabel: busy ? '寫入中' : '確定',
    cancelLabel: '取消',
    busy,
    lines: [
      `標題：${normalized.name}`,
      `時間：${formatSchedulerCronNextTime(normalized.cronExpr)}`,
      `摘要：${normalized.summary}`,
    ],
    summary: normalized.summary,
  };
}

function mergeSchedulerSlotsFromText(currentDraft, text, preferredSlot = '') {
  const raw = String(text || '').trim();
  const next = {...(currentDraft || {})};
  const actionText = parseSchedulerActionText(raw);
  const cleanedName = cleanSchedulerNameText(raw);
  const deliveryText = extractSchedulerDeliveryText(raw);
  const explicitAction = /提醒|提醒我|叫我|通知我|動作|執行|做|跑/.test(raw);

  if (preferredSlot === 'name' && raw) next.name = raw.slice(0, 32);
  if (preferredSlot === 'action' && raw) next.actionText = raw.slice(0, 80);
  if (preferredSlot === 'time' && (hasSchedulerTimeText(raw) || hasSchedulerClockText(raw))) next.cronExpr = parseSchedulerTimeText(raw, next.cronExpr || '');

  if (hasSchedulerTimeText(raw)) next.cronExpr = parseSchedulerTimeText(raw, next.cronExpr || '');
  if (/名稱|標題/.test(raw) && cleanedName) next.name = cleanedName;
  if (explicitAction && actionText) next.actionText = actionText.slice(0, 80);
  if (!next.name && (cleanedName || actionText)) next.name = (actionText || cleanedName).slice(0, 32);
  if (!next.actionText && (cleanedName || actionText)) next.actionText = (actionText || cleanedName).slice(0, 80);
  if (deliveryText) next.deliveryText = deliveryText;
  if (raw) next.sourceText = [next.sourceText, raw].filter(Boolean).join('\n').slice(-500);

  const normalized = normalizeSchedulerDraft(next);
  normalized.actionPayload = normalized.actionText
    ? schedulerDefaultPayload(normalized.name || '提醒', normalized.actionText, normalized.summary)
    : '';
  return normalized;
}

function findSchedulerJobFromText(text, jobs = []) {
  const raw = String(text || '');
  const indexWords = [
    ['第一', 0], ['第1', 0], ['1', 0],
    ['第二', 1], ['第2', 1], ['2', 1],
    ['第三', 2], ['第3', 2], ['3', 2],
    ['第四', 3], ['第4', 3], ['4', 3],
    ['第五', 4], ['第5', 4], ['5', 4],
  ];
  const indexed = indexWords.find(([word]) => raw.includes(`${word}個任務`) || raw.includes(`${word}個排程`) || raw.includes(`${word}任務`) || raw.includes(`${word}排程`));
  if (indexed) {
    const targetNo = indexed[1] + 1;
    return jobs.find((job, index) => schedulerJobNo(job, index) === targetNo) || null;
  }
  const hashNo = raw.match(/#\s*(\d+)|編號\s*(\d+)/);
  if (hashNo) {
    const targetNo = Number(hashNo[1] || hashNo[2]);
    return jobs.find((job, index) => schedulerJobNo(job, index) === targetNo) || null;
  }
  return jobs.find((job) => job?.name && raw.includes(job.name)) || null;
}

function parseSchedulerConversationIntent(text, jobs = []) {
  const raw = String(text || '').trim();
  if (!/(排程|排定|定時|提醒|規劃|每小時|每天|每日|每週|每周|禮拜|星期)/.test(raw)) return null;
  const isUpdate = /(修改|更改|改成|改為|變更|調整)/.test(raw);
  if (isUpdate) {
    const job = findSchedulerJobFromText(raw, jobs);
    if (!job) return {type: 'open', message: '我找不到你想修改的排程，先打開清單讓你確認編號。'};
    const nextName = /名稱/.test(raw) ? (cleanSchedulerNameText(raw) || job.name) : job.name;
    const nextCron = hasSchedulerTimeText(raw)
      ? parseSchedulerTimeText(raw, job.cron_expr)
      : job.cron_expr;
    const nextActionText = /動作/.test(raw) ? parseSchedulerActionText(raw) : '';
    const nextPayload = nextActionText ? schedulerDefaultPayload(nextName, nextActionText) : job.action_payload;
    return {
      type: 'update',
      job,
      patch: {
        name: nextName,
        cronExpr: nextCron,
        actionType: job.action_type || 'event',
        actionPayload: nextPayload,
      },
    };
  }
  const draft = mergeSchedulerSlotsFromText({}, raw);
  return {
    type: 'create',
    draft,
  };
}

function formatSchedulerTime(value) {
  if (!value) return '尚未排定';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return formatSchedulerDateKey(date);
}

function formatSchedulerDateKey(value) {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return String(value || '');
  const pad = (number) => String(number).padStart(2, '0');
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
    pad(date.getHours()),
    pad(date.getMinutes()),
  ].join('-');
}

function schedulerCronPartNumber(part) {
  const raw = String(part || '').trim();
  return /^\d{1,2}$/.test(raw) ? Number(raw) : null;
}

function schedulerCronPartMatches(part, value) {
  const raw = String(part || '').trim();
  if (raw === '*' || raw === '') return true;
  const number = schedulerCronPartNumber(raw);
  return number == null ? false : number === value;
}

function nextSchedulerFireDate(cronExpr, baseDate = new Date()) {
  const raw = String(cronExpr || '').trim();
  if (!raw) return null;
  const base = new Date(baseDate);
  if (Number.isNaN(base.getTime())) return null;
  if (/^@hourly$/i.test(raw)) {
    const next = new Date(base);
    next.setMinutes(0, 0, 0);
    next.setHours(next.getHours() + 1);
    return next;
  }
  if (/^@daily$/i.test(raw)) return nextSchedulerFireDate('0 9 * * *', base);
  if (/^@weekly$/i.test(raw)) return nextSchedulerFireDate('0 9 * * 1', base);
  if (/^@monthly$/i.test(raw)) return nextSchedulerFireDate('0 9 1 * *', base);
  const parts = raw.split(/\s+/);
  if (parts.length !== 5) return null;
  const minute = schedulerCronPartNumber(parts[0]);
  const hour = schedulerCronPartNumber(parts[1]);
  if (minute == null || hour == null) return null;
  const start = new Date(base);
  start.setSeconds(0, 0);
  for (let offset = 0; offset <= 370; offset += 1) {
    const candidate = new Date(start);
    candidate.setDate(start.getDate() + offset);
    candidate.setHours(hour, minute, 0, 0);
    if (candidate <= start) continue;
    const dayOfMonth = candidate.getDate();
    const month = candidate.getMonth() + 1;
    const weekday = candidate.getDay();
    if (!schedulerCronPartMatches(parts[2], dayOfMonth)) continue;
    if (!schedulerCronPartMatches(parts[3], month)) continue;
    if (!schedulerCronPartMatches(parts[4], weekday)) continue;
    return candidate;
  }
  return null;
}

function formatSchedulerCronNextTime(cronExpr, fallback = '') {
  const next = nextSchedulerFireDate(cronExpr);
  return next ? formatSchedulerDateKey(next) : (fallback || cronExpr || '尚未排定');
}

function formatSchedulerJobNextTime(job) {
  const nextFire = job?.next_fire || job?.nextFire || '';
  if (nextFire) return formatSchedulerTime(nextFire);
  return formatSchedulerCronNextTime(job?.cron_expr || job?.cronExpr || '');
}

// 排程清單／摘要只顯示「下次會啟動的時間」：停用或無 next_fire 一律留空，
// 讓使用者一眼看出沒有排程；每日/每週/每月等 cron 規則不顯示給使用者。
function schedulerActiveNextLabel(job) {
  if (!job || job.enabled === false) return '';
  const nextFire = job.next_fire || job.nextFire || '';
  return nextFire ? formatSchedulerTime(nextFire) : '';
}

function formatSchedulerTimeLong(value) {
  if (!value) return '尚未排定';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-TW', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatSchedulerPayload(payload, fallback = '提醒') {
  const parsed = parseSchedulerPayload(payload);
  return parsed.action || parsed.title || parsed.event || payload || fallback;
}

function formatSchedulerPayloadSummary(payload, fallback = '尚無摘要') {
  const parsed = parseSchedulerPayload(payload);
  return parsed.summary || parsed.action || parsed.title || payload || fallback;
}

function parseSchedulerPayload(payload) {
  try {
    const parsed = JSON.parse(payload || '{}');
    return {
      title: parsed?.data?.title || '',
      action: parsed?.data?.action || '',
      summary: parsed?.data?.summary || '',
      event: parsed?.event_name || '',
    };
  } catch {
    return {title: '', action: String(payload || ''), summary: '', event: ''};
  }
}

function schedulerJobNo(job, fallbackIndex = 0) {
  const no = Number(job?.schedule_no ?? job?.scheduleNo ?? 0);
  return no > 0 ? no : fallbackIndex + 1;
}

function SchedulerPanel({
  clock,
  jobs,
  draft,
  busy,
  error,
  onDraftChange,
  onRefresh,
  onCreate,
  onJobAction,
  onBootstrapSkill,
  onClose,
}) {
  const t = useI18n(s => s.t);
  const cronPresets = [
    {label: t('scheduler.presetHourly'), value: '@hourly'},
    {label: t('scheduler.presetDaily0900'), value: '0 9 * * *'},
    {label: t('scheduler.presetWeeklyMon0900'), value: '0 9 * * 1'},
  ];
  const hasDraft = Boolean(draft.name || draft.actionPayload);
  const [summaryJob, setSummaryJob] = useState(null);
  const summaryText = summaryJob ? formatSchedulerPayloadSummary(summaryJob.action_payload, summaryJob.name || t('scheduler.noSummary')) : '';
  const updateDraft = (patch) => onDraftChange((current) => ({...current, ...patch}));

  return createPortal(
    <div className="scheduler-overlay" role="dialog" aria-modal="true" aria-label={t('scheduler.dialogLabel')}>
      <section className="scheduler-panel">
        <header className="scheduler-header">
          <div>
            <span className="scheduler-kicker">Scheduler</span>
            <h3>{t('scheduler.title')}</h3>
          </div>
          <button className="scheduler-close" type="button" onClick={onClose} aria-label={t('common.close')}>×</button>
        </header>

        <div className="scheduler-clock-band">
          <div>
            <span>{t('scheduler.systemTime')}</span>
            <strong>{clock?.local_time || t('scheduler.syncing')}</strong>
          </div>
          <div>
            <span>{t('scheduler.timezone')}</span>
            <strong>{[clock?.timezone, clock?.utc_offset].filter(Boolean).join(' ') || t('scheduler.local')}</strong>
          </div>
          <button type="button" onClick={onRefresh} disabled={busy}>{t('scheduler.sync')}</button>
        </div>

        {hasDraft && (
          <div className="scheduler-draft-strip">
            <div className="scheduler-draft-fields">
              <label>
                <span>{t('scheduler.draftName')}</span>
                <input
                  type="text"
                  value={draft.name}
                  onChange={(event) => updateDraft({name: event.target.value})}
                  placeholder={t('scheduler.draftNamePlaceholder')}
                />
              </label>
              <label>
                <span>{t('scheduler.rule')}</span>
                <input
                  type="text"
                  value={draft.cronExpr}
                  onChange={(event) => updateDraft({cronExpr: event.target.value})}
                  placeholder="0 9 * * *"
                />
              </label>
              <label>
                <span>{t('scheduler.summary')}</span>
                <textarea
                  value={formatSchedulerPayloadSummary(draft.actionPayload, '')}
                  onChange={(event) => updateDraft({actionPayload: schedulerDefaultPayload(draft.name || t('scheduler.defaultReminder'), draft.name || t('scheduler.defaultReminder'), event.target.value)})}
                  placeholder={t('scheduler.summaryPlaceholder')}
                  rows={2}
                />
              </label>
            </div>
            <div className="scheduler-preset-row">
              {cronPresets.map((preset) => (
                <button type="button" key={preset.value} onClick={() => updateDraft({cronExpr: preset.value})}>
                  {preset.label}
                </button>
              ))}
              <button className="scheduler-create-btn" type="button" onClick={onCreate} disabled={busy}>
                {busy ? t('scheduler.busy') : t('scheduler.create')}
              </button>
            </div>
          </div>
        )}

        {error && <p className="scheduler-error">{error}</p>}

        <div className="scheduler-table">
          <div className="scheduler-table-scroll">
            <div className="scheduler-table-head">
              <span>#</span>
              <span>{t('scheduler.tableTitle')}</span>
              <span>{t('scheduler.tableTime')}</span>
              <span>{t('scheduler.tableStatus')}</span>
              <span></span>
            </div>
            {jobs.length === 0 ? (
              <div className="scheduler-empty">{t('scheduler.empty')}</div>
            ) : jobs.map((job, index) => (
              <article className="scheduler-job" key={job.id}>
                <span className="scheduler-job-index">{schedulerJobNo(job, index)}</span>
                <div className="scheduler-job-name">
                  <strong>{job.name}</strong>
                </div>
                <span>{schedulerActiveNextLabel(job)}</span>
                <small>{job.enabled ? t('scheduler.enabled') : t('scheduler.paused')}</small>
                <div className="scheduler-job-actions">
                  <button type="button" disabled={busy} onClick={() => setSummaryJob(job)}>{t('scheduler.summary')}</button>
                  {job.enabled ? (
                    <button type="button" disabled={busy} onClick={() => onJobAction(job.id, 'pause')}>{t('scheduler.pause')}</button>
                  ) : (
                    <button type="button" disabled={busy} onClick={() => onJobAction(job.id, 'resume')}>{t('scheduler.resume')}</button>
                  )}
                  <button type="button" disabled={busy} onClick={() => onJobAction(job.id, 'delete')}>{t('scheduler.delete')}</button>
                </div>
              </article>
            ))}
          </div>
        </div>

        {summaryJob && (
          <div className="scheduler-summary-modal" role="dialog" aria-label={t('scheduler.summary')}>
            <section className="scheduler-summary-card">
              <header>
                <div>
                  <span>{t('scheduler.summary')}</span>
                  <h4>{summaryJob.name || t('scheduler.unnamed')}</h4>
                </div>
                <button type="button" onClick={() => setSummaryJob(null)} aria-label={t('scheduler.closeSummary')}>×</button>
	              </header>
	              <p>{summaryText}</p>
	              <dl>
	                <div>
	                  <dt>{t('scheduler.tableTime')}</dt>
	                  <dd>{schedulerActiveNextLabel(summaryJob)}</dd>
	                </div>
	                <div>
	                  <dt>{t('scheduler.tableStatus')}</dt>
	                  <dd>{summaryJob.enabled ? t('scheduler.enabled') : t('scheduler.paused')}</dd>
	                </div>
	                <div>
	                  <dt>Skill</dt>
	                  <dd>{summaryJob.skill_id || t('scheduler.notCreated')}</dd>
	                </div>
	                <div>
	                  <dt>Sub</dt>
	                  <dd>{summaryJob.source_sub_id || ''}</dd>
	                </div>
	              </dl>
	              {!summaryJob.skill_id && (
	                <button
	                  className="scheduler-create-btn"
	                  type="button"
	                  disabled={busy}
	                  onClick={async () => {
	                    await onBootstrapSkill?.(summaryJob);
	                    setSummaryJob(null);
	                  }}
	                >
	                  {busy ? t('scheduler.busy') : t('scheduler.createAutoSkill')}
	                </button>
	              )}
	            </section>
	          </div>
	        )}
      </section>
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
  onScheduleOpen,
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
          event.stopPropagation();
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
      <button className="tool-card tool-amber tool-schedule-card" type="button" onClick={onScheduleOpen}>
        <span>◴</span>
        <span>{t('rightRail.schedule')}</span>
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
              <small className="reference-file-status">{referenceFileStatusLabel(file.status, t)}</small>
              {file.addedVia === 'floating_avatar' && (
                <small className="reference-file-source-badge">{t('floatingAvatar.addedViaFloating')}</small>
              )}
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

export default App;
