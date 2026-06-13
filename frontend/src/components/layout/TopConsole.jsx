import {useEffect, useRef, useState} from 'react';
import {ClearInspectorHistory, FinalizeNativeSubExport, ListPixelAvatarPacks, NativeDragExportSubHandler, RenderPixelAvatarPreview, SetPersonaPixelAvatarPack} from '../../../wailsjs/go/main/App';
import {EventsOn} from '../../../wailsjs/runtime/runtime';
import useI18n from '../../locales/useI18n';
import {avatarStateLabel, avatarStateOptions, bytesToDataUrl, callWails, lockedPersonaId, lockedPersonaName, pixelAvatarRenderSize, pixelPackForPersona, reviewStatusLabel} from '../../lib/appShared';
import DragActionModal from '../tools/DragActionModal';
import ReviewPanel from '../chat/ReviewPanel';
import SkillActivityCard from '../tools/SkillActivityCard';
import SkillFirstUseCard from '../tools/SkillFirstUseCard';

export default function TopConsole({
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
