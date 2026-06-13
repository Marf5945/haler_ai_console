import './tailwind.css';
import React, {useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import {AcknowledgePendingItem, AcknowledgePendingItemWithConfirmation, AcknowledgeStatusRail, ActivateRemoteBridgeChannel, ActivateTool, AddSourceToAllowlist, AddW3ATrustedDeveloper, AppendTalkEntryForAgent, ApplyUIStyleDiff, ApproveTaskStep, AutoDetectCLI, BuildLLMContext, BuildSkillContext, CancelActiveTaskProgress, CancelChatMessage, ClassifySource, ClearSkillContext, ClearVoiceDebug, ClearWebSearchConfig, CompleteOnboardingStep, ConfirmAndExecuteSkillExecution, ConfirmClose, ConfirmCredentialMigration, ConfirmPackageInstall, ConfirmRegisterLLMAPIAdapter, ConfirmSkillArchive, CreateDestructiveReviewCard, CreateSubagent, DeactivateRemoteBridgeChannel, DeleteStaticAvatar, DeleteTalkMessageForAgent, DetectModelPollution, DetectRemoteBridgeChannel, DisableContextualRiskOverride, DisableCredentialMigration, DismissFloatingCandidates, DispatchRemoteBridgeAsync, DualStepConfirmStep1, EnableContextualRiskOverride, EnableDetectedCLI, EnableLocalModel, EnableTrustedSessionScope, EnableWorkflowTrustForHours, EnterDegradedMode, EscapeExternalTokens, ExecuteSkillMessage, ExitDegradedMode, ExportMediaWithSidecar, FinalizeNativePersonaExport, FinalizeNativeReferenceFileExport, FinalizeNativeSkillExport, FinishOnboarding, GenerateLearningRunMetadata, GetActiveLearningRun, GetAdapterModelChoices, GetAppSessionID, GetBrowserPreference, GetConfigPublic, GetConsoleState, GetCredentialMigrationStatus, GetCurrentAvatar, GetDegradedState, GetDeviceTrustProfile, GetEmbeddingConfig, GetHighImpactDomains, GetHookSummary, GetLastLearningReplayPlan, GetLearningReplayPlan, GetMediaW3AInfo, GetMemoryHealth, GetMemoryPipelineState, GetNewSubagentCandidates, GetOnboardingState, GetPendingCandidateCount, GetPendingDigest, GetPendingTagPatches, GetProjectAllowlist, GetReadinessGateState, GetRecentSkillInjections, GetRemoteBridgeInboundEndpoint, GetSafariRuntimeNotice, GetSettingsState, GetSubExportCapabilities, GetSummaryModelSettings, GetTabOrder, GetTalkMessagesForAgent, GetToolRegistryPatchProposals, GetToolVisibility, GetUISettings, GetVoiceSettings, GetW3ATransferGuidance, GetWebSearchConfig, GoBackOnboarding, HasBlockingReviewCard, HasBlockingVLReview, HasOpenStopRecoveryCard, ImportGoProgramExport, ImportLearningRunExport, ImportMediaVerify, ImportReferenceFile, ImportSubHandler, ImportVideoFile, InstallVoiceBaseModel, InvalidateDualStep, IsDegradedBlocked, IsLearningModeActive, IsReadOnlyMode, ListAdapterModelOptions, ListArchivedSkills, ListAvailableAdapters, ListAvatarPresets, ListExternalLinksByType, ListLearningReplayCatalog, ListOpenLightweightCards, ListOpenReviewCards, ListOpenStopRecoveryCards, ListPendingPackages, ListPurgeManifests, ListReferenceFiles, ListReferenceImages, ListRemoteBridgeChannels, ListRemoteBridgeInboundAdapters, ListReviewArchive, ListThreatRecords, ListTools, ListVideoFiles, ListW3ATrustedDevelopers, MarkSkillFirstUseExplained, NativeDragExportPersonaHandler, NativeDragExportReferenceFile, NativeDragExportSkill, PollStatusRail, PreparePackageInstallPayload, PreviewExternalLink, PreviewGoProgramExport, PreviewLearningRunExport, PreviewSubPackage, PromoteDraftToPending, PurgeBoundaryDir, RecordLearningMouseEvent, RecordStepTrace, RegisterCustomCLI, RegisterExternalLink, RegisterLLMAPIAdapter, RegisterRemoteBridgeChannel, RegisterRemoteBridgeChannelWithMode, RejectPackageInstall, RemoveAllowlistEntry, RemoveRemoteBridgeChannel, RemoveVoiceBaseModel, RenameAdapter, RenameRemoteBridgeChannel, RenameSubagent, RenewAllowlistEntry, ReorderPersonas, ReorderTabs, ResolveAndExecuteDestructiveReviewCard, ResolveImportToolConflicts, ResolveLightweightCard, ResolveReviewCard, ResolveSkillForAction, ResolveStopRecoveryCard, RestartSidecar, RestoreUIDefaults, RouteVoiceCommand, RunSummarizationNow, SavePanelSettings, SavePersona, SaveRemoteBridgeInboundSecret, SaveStaticAvatar, SaveSummaryModelSettings, SaveVoiceSettings, SaveWebSearchConfig, ScanLocalModels, ScanLocalSummaryModels, ScanSkillFolder, SearchLearningOperations, SelectFloatingCandidate, SendAPIMessage, SendCLIMessage, SendTopInteractionMessage, SetActiveConversationAgent, SetAdapterModelChoice, SetAdapterStatus, SetAvatarProvider, SetBrowserPreference, SetRemoteBridgePrimaryChannel, SetToolPreference, SetTrustDomAndClick, StartDraftSandbox, StartLearningMode, StartTaskProgress, StopDraftSandbox, StopLearningMode, StopSidecar, SwitchRemoteBridgeMode, TestRemoteBridgeConnection, TranscribeVoiceWAV, UnregisterAdapter, ValidateMemoryItem, WakeLocalAdapter} from '../wailsjs/go/main/App';
import {ClipboardGetText, ClipboardSetText, EventsOn, OnFileDrop, OnFileDropOff, Quit} from '../wailsjs/runtime/runtime';
import VisualLearningPanel from './components/VisualLearningPanel';
import EmbeddingPickerModal from './components/EmbeddingPickerModal';
import useI18n, {t as _t} from './locales/useI18n';
import {GACHA_COLLAPSE_MS, GACHA_PARTICLE_MS, GACHA_PULSE_MS, LONG_PRESS_DURATION_MS, MAX_VOICE_RECORDING_MS, MIN_VOICE_RECORDING_MS, VOICE_BG_THRESHOLD_MS, adapterField, adapterKey, appendUniqueReferenceFile, autoAvatarOverrideStates, avatarStateOptions, blobToBase64, buildLearningWindowsAnchor, bytesToDataUrl, callWails, composerPendingSlowText, composerPendingVerySlowText, dagHighRiskPattern, defaultPanelStyle, defaultPixelPackForPersona, defaultWebSearchProviderOptions, delayLearningReplay, deriveAvatarExpression, describeLearningTarget, encodeWav, executeLearningReplayPlan, executeLearningReplayStep, extractVisualReplayDirective, fallbackReadinessGate, fallbackReviewState, fallbackSettings, fallbackState, findActivePersona, fontScaleValue, formatLearningOperationCatalog, formatLearningOperationLearned, formatLearningOperationSearchResults, formatLearningReplayExecutionResult, formatLearningReplayPlan, getFallbackTools, getPersonaAvatar, getPixelAvatarDataUrl, isInvalidReferencePlaceholder, isLearningReplayRequest, isTaskProgressActive, isVideoPath, learningDigestStorageKey, learningIdleDelayMs, learningRecordingBlockedSelector, lockedPersonaId, lockedPersonaName, looksLikeReferenceLocalPath, makeComposerPendingMessage, makeDebugTraceID, mapBackendTaskRun, maxPersonas, mergeDraftText, mergeReferenceLibraryFiles, normalizeAdapterDTO, normalizeLockedPersonas, normalizePanelSettings, normalizeReferenceImportPaths, normalizeSettingsState, normalizeToolList, openExternal, operationIntentFromCLIResponse, panelFromUISettings, panelStyleTheme, parseLearningOperationCatalogRequest, pickRotatingGreeting, pixelAvatarRenderSize, pixelPackForPersona, plannerClarificationFromError, postDebugTrace, referenceFileKey, reorderItemsByKeys, reorderReferenceFiles, replaceComposerPendingMessage, resolveAvatarProvider, resolveLearningOperationMatch, resolvePersonaAvatarSrc, resolveStaticAvatarPath, reviewCardID, shouldCreateDagRun, shouldHandleLearningShortcutBeforeLLM, shouldKeepNewerTaskRun, splitAvailableAdapters, stripInternalControlDraft, stripInternalControlPrefix, toolTabFor, updateReferenceFileStatus, waitForVoiceChunks} from './lib/appShared';
import AdapterRenameModal from './components/modals/AdapterRenameModal';
import AvatarUploadModal from './components/settings/AvatarUploadModal';
import ConsequenceMenuOverlay from './components/tools/ConsequenceMenuOverlay';
import ConversationPanel from './components/chat/ConversationPanel';
import DigestAutoArchiveToast from './components/modals/DigestAutoArchiveToast';
import DraftSandboxStopDialog from './components/modals/DraftSandboxStopDialog';
import DragActionModal from './components/tools/DragActionModal';
import LLMAPISetupModal from './components/modals/LLMAPISetupModal';
import LearningReplayConfirmCard from './components/modals/LearningReplayConfirmCard';
import LearningReplayStartConfirmCard from './components/modals/LearningReplayStartConfirmCard';
import OnboardingOverlay from './components/onboarding/OnboardingOverlay';
import PackageConfirmModal from './components/modals/PackageConfirmModal';
import PackageErrorToast from './components/modals/PackageErrorToast';
import PackageInstallDecisionDialog from './components/modals/PackageInstallDecisionDialog';
import ProjectManagePopup from './components/modals/ProjectManagePopup';
import ReauthInterceptDialog from './components/modals/ReauthInterceptDialog';
import ReferenceLinkModal from './components/modals/ReferenceLinkModal';
import RightRail from './components/layout/RightRail';
import SafariNoticeModal from './components/modals/SafariNoticeModal';
import SessionCloseDialog from './components/modals/SessionCloseDialog';
import SettingsWorkspace from './components/settings/SettingsWorkspace';
import Sidebar from './components/layout/Sidebar';
import StopRecoveryCard from './components/modals/StopRecoveryCard';
import StyleDiffPreviewModal from './components/modals/StyleDiffPreviewModal';
import SubToolConflictDialog from './components/modals/SubToolConflictDialog';
import ToolCopyConfirmModal from './components/tools/ToolCopyConfirmModal';
import ToolPopup from './components/tools/ToolPopup';
import TopConsole from './components/layout/TopConsole';
import TrustedSessionExpiredDialog from './components/modals/TrustedSessionExpiredDialog';
import WebSearchSetupModal from './components/modals/WebSearchSetupModal';

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
  const [referenceLinkOpen, setReferenceLinkOpen] = useState(false);
  const [referenceLinkValue, setReferenceLinkValue] = useState('');
  const referenceInternalDragRef = useRef(false);
  const referenceDropSuppressUntilRef = useRef(0);
  const referenceImportInFlightRef = useRef(new Set());
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
    // 使用者已按停止鈕中斷此 trace：吞掉被 kill 的子程序錯誤，不覆蓋「已中斷」訊息。
    if (cancelledChatTracesRef.current.has(traceId)) {
      cancelledChatTracesRef.current.delete(traceId);
      clearPendingTimers?.();
      return;
    }
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

  async function sendMessage(event, images = []) {
    event.preventDefault();
    submitComposerText(draft, images);
  }

  async function submitComposerText(rawText, images = []) {
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
      // 此回合對話送出已收尾 → 釋放「執行中」狀態，停止鈕回到 disabled。
      if (activeChatTraceRef.current === traceId) {
        activeChatTraceRef.current = null;
        setActiveChatTrace(null);
      }
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
      '繁中': 'zh-TW', '中文': 'zh-TW', '英文': 'en', '日文': 'ja',
      'Traditional Chinese': 'zh-TW', 'Chinese': 'zh-TW', 'English': 'en', 'Japanese': 'ja',
      '中': 'zh-TW', 'en': 'en', 'ja': 'ja',
      'pt': 'pt-PT', 'pt-PT': 'pt-PT', 'es': 'es', 'th': 'th',
      'Português': 'pt-PT', '葡萄牙文': 'pt-PT', 'Español': 'es', '西班牙文': 'es', 'ไทย': 'th', '泰文': 'th',
      // self-heal: raw i18n keys leaked by an older build
      'settings.langPt': 'pt-PT', 'settings.langEs': 'es', 'settings.langTh': 'th',
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
              taskActive={isTaskProgressActive(dagRun) || !!activeChatTrace}
              onCancelTask={cancelActiveExecution}
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

export default App;
