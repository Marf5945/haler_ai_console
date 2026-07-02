import React, {useEffect, useMemo, useRef, useState} from 'react';
import AnimatedFullBodyAvatar from './AnimatedFullBodyAvatar.jsx';

const defaultAvatarSize = 64;
const edgeGap = 12;
const compactTipGap = 8;
const compactTipMinWidth = 80;
const compactTipMaxWidth = 224;
const shakePreviewDelayMs = 1200;
const shakePreviewDistance = 80;
const shakeVomitDelayMs = 3000;
const shakeVomitDistance = 500;
const fullBodyAvatarWidth = 200;
const fullBodyAvatarHeight = 360;

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

function candidateText(candidate) {
  return String(candidate?.label || candidate?.draft || candidate?.id || '').replace(/^Ai:/, '').trim();
}

function installSourceLabel(candidate) {
  const src = candidate?.preview?.source_system_code || candidate?.path || '';
  return String(src).split(/[\\/]/).pop() || String(src);
}

function installRiskSummary(candidate, t) {
  return candidate?.type === 'learningrun'
    ? t('floatingAvatar.installRiskLow')
    : t('floatingAvatar.installRiskExec');
}

function randomShakePreview(options = []) {
  const pool = (options || []).filter((item) => String(item?.text || '').trim());
  if (pool.length === 0) return {text: '', expression: 'sad'};
  return pool[Math.floor(Math.random() * pool.length)];
}

// 後台迷你框：自然語言排程的輕量偵測，送出前在泡泡裡先確認一次。
const SCHEDULE_HINT_RE = /(每天|每日|每週|每周|每月|明天|後天|今晚|稍後|提醒我|排程|定時|幾點|\d點|\d\s*分鐘後|\d\s*小時後|\d{1,2}:\d{2}|daily|weekly|every\s+day|tomorrow|remind\s+me|schedule|at\s+\d)/i;
function looksLikeSchedule(text) {
  return SCHEDULE_HINT_RE.test(String(text || ''));
}

export default function FloatingAvatarMode({
  active,
  t,
  avatarSrc,
  fullBodyAvatarSrc = '',
  fullBodyAvatarKey = '',
  avatarExpression,
  persona,
  personas = [],
  adapterLabel = '',
  position,
  onPositionChange,
  onRestore,
  onQuit,
  onOpenSettings,
  onDropFiles,
  onSubmit,
  onChatModeChange,
  onChatModeToggle,
  onSwitchAgent,
  onSetReminderMode,
  compactWindowMode = false,
  panelOpenSignal = 0,
  bodyMode = 'head',
  onBodyModeChange,
  onCompactDragStart,
  onCompactDrag,
  onCompactDragEnd,
  onCompactExpandChange,
  flyingBack = false,
  chatMode = false,
  activePersonaId = '',
  reminderPaused = false,
  reminderLabel = '',
  candidates = [],
  selectedCandidateIDs = [],
  exclusiveCandidates = false,
  onSelectCandidate,
  statusTitle = '',
  statusText = '',
  latestText = '',
  bubbleText = '',
  pendingConfirm = null,
  installCandidate = null,
  onConfirmInstall,
  onCancelInstall,
  draft,
  onDraftChange,
  shakeDialogueOptions = [],
  onShakePreview,
  onShakeTooLong,
  avatarSize = defaultAvatarSize,
}) {
  const [panelOpen, setPanelOpen] = useState(false);
  const [contextOpen, setContextOpen] = useState(false);
  const [agentPickerOpen, setAgentPickerOpen] = useState(false);
  const [reminderPickerOpen, setReminderPickerOpen] = useState(false);
  const [dropActive, setDropActive] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const [guideVisible, setGuideVisible] = useState(true);
  const [shakeState, setShakeState] = useState(null);
  const [shakeMessageVisible, setShakeMessageVisible] = useState(false);
  const [shakeDialogue, setShakeDialogue] = useState('');
  const [bubbleDismissed, setBubbleDismissed] = useState(false);
  const [installArmed, setInstallArmed] = useState(false);
  const [scheduleConfirmPending, setScheduleConfirmPending] = useState(false);
  const setBodyMode = onBodyModeChange || (() => {});
  const showBody = bodyMode !== 'head';
  const avatarFrameWidth = showBody ? fullBodyAvatarWidth : avatarSize;
  const avatarFrameHeight = showBody ? fullBodyAvatarHeight : avatarSize;
  const rootRef = useRef(null);
  const dragRef = useRef(null);
  const clickTimerRef = useRef(null);
  const shakeResetRef = useRef(null);
  const shakeRef = useRef({startedAt: 0, lastPoint: null, distance: 0, previewFired: false, vomitFired: false});

  const clampedPosition = useMemo(() => {
    const maxX = Math.max(edgeGap, window.innerWidth - avatarFrameWidth - edgeGap);
    const maxY = Math.max(edgeGap, window.innerHeight - avatarFrameHeight - edgeGap);
    return {
      x: clamp(Number(position?.x ?? maxX), edgeGap, maxX),
      y: clamp(Number(position?.y ?? maxY), edgeGap, maxY),
    };
  }, [avatarFrameHeight, avatarFrameWidth, position?.x, position?.y]);

  useEffect(() => {
    if (!active) return;
    const rawX = Number(position?.x ?? clampedPosition.x);
    const rawY = Number(position?.y ?? clampedPosition.y);
    if (rawX !== clampedPosition.x || rawY !== clampedPosition.y) {
      onPositionChange?.(clampedPosition);
    }
  }, [active, clampedPosition, onPositionChange, position?.x, position?.y]);

  useEffect(() => {
    if (!active) return undefined;
    const onResize = () => onPositionChange?.(clampedPosition);
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, [active, clampedPosition, onPositionChange]);

  useEffect(() => {
    if (!contextOpen) return undefined;
    const close = (event) => {
      if (rootRef.current?.contains(event.target)) return;
      setContextOpen(false);
      setAgentPickerOpen(false);
      setReminderPickerOpen(false);
    };
    document.addEventListener('pointerdown', close);
    return () => document.removeEventListener('pointerdown', close);
  }, [contextOpen]);

  useEffect(() => {
    if (!compactWindowMode) return undefined;
    const hasTopBubble = (Boolean(String(bubbleText || '').trim()) && !bubbleDismissed) || shakeMessageVisible;
    onCompactExpandChange?.(panelOpen || contextOpen || hasTopBubble || Boolean(installCandidate) || showBody);
    return undefined;
  }, [compactWindowMode, panelOpen, contextOpen, bubbleText, bubbleDismissed, shakeMessageVisible, installCandidate, showBody, bodyMode]);

  useEffect(() => () => {
    if (clickTimerRef.current) window.clearTimeout(clickTimerRef.current);
    if (shakeResetRef.current) window.clearTimeout(shakeResetRef.current);
  }, []);

  useEffect(() => {
    if (bubbleText) setBubbleDismissed(false);
  }, [bubbleText]);

  useEffect(() => {
    setInstallArmed(false);
  }, [installCandidate?.path, installCandidate?.type]);

  useEffect(() => {
    if (!panelOpen) setScheduleConfirmPending(false);
  }, [panelOpen]);

  useEffect(() => {
    if (!active || !panelOpenSignal) return;
    setContextOpen(false);
    setAgentPickerOpen(false);
    setReminderPickerOpen(false);
    setPanelOpen(true);
  }, [active, panelOpenSignal]);

  if (!active) return null;

  const selectedSet = new Set(selectedCandidateIDs || []);
  const visibleCandidates = (candidates || []).slice(0, 5);
  const panelLeft = clampedPosition.x < window.innerWidth - 390
    ? clampedPosition.x + avatarFrameWidth + 14
    : clampedPosition.x - 342;
  const panelTop = clamp(clampedPosition.y - 18, edgeGap, Math.max(edgeGap, window.innerHeight - 410));
  const stackLeft = clamp(clampedPosition.x - 44, edgeGap, Math.max(edgeGap, window.innerWidth - 240));
  const stackTop = clamp(clampedPosition.y + avatarFrameHeight + 10, edgeGap, Math.max(edgeGap, window.innerHeight - 230));
  const leftTipLeft = clampedPosition.x > 250 ? clampedPosition.x - 236 : clampedPosition.x + avatarFrameWidth + 16;
  const rightTipLeft = clampedPosition.x < window.innerWidth - 360 ? clampedPosition.x + avatarFrameWidth + 16 : clampedPosition.x - 304;
  // compact 浮窗模式：面板要落在放大後的視窗內（非整個螢幕座標）。
  const compactPanelTop = clamp(clampedPosition.y - 10, edgeGap, Math.max(edgeGap, window.innerHeight - 430));
  const compactPanelLeft = clamp(clampedPosition.x + avatarFrameWidth + 18, edgeGap, Math.max(edgeGap, window.innerWidth - 340));
  const compactPanelStyle = {
    left: compactPanelLeft,
    top: compactPanelTop,
    width: Math.min(328, Math.max(220, window.innerWidth - compactPanelLeft - edgeGap)),
    maxHeight: Math.max(180, window.innerHeight - compactPanelTop - edgeGap),
  };
  const compactStackTop = clampedPosition.y + avatarFrameHeight + 12;
  const compactStackStyle = {
    left: clamp(clampedPosition.x - 82, edgeGap, Math.max(edgeGap, window.innerWidth - 230)),
    top: compactStackTop,
  };
  // compact 模式提示優先放頭像左側；若左側空間不足，留在視窗內改放頭像下方。
  const compactTipLeftSpace = clampedPosition.x - edgeGap - compactTipGap;
  const compactTipFitsLeft = compactTipLeftSpace >= compactTipMinWidth;
  const compactTipWidth = compactTipFitsLeft
    ? clamp(compactTipLeftSpace, compactTipMinWidth, compactTipMaxWidth)
    : clamp(window.innerWidth - edgeGap * 2, compactTipMinWidth, compactTipMaxWidth);
  const compactTipStyle = compactTipFitsLeft
    ? {left: -(compactTipWidth + compactTipGap), top: 0, width: compactTipWidth}
    : {left: edgeGap - clampedPosition.x, top: avatarFrameHeight + compactTipGap, width: compactTipWidth};
  const displayStatus = pendingConfirm?.title || statusTitle || t('floatingAvatar.statusIdle');
  const displayLatest = pendingConfirm?.reason || latestText || statusText || t('floatingAvatar.latestIdle');
  const speakerName = persona?.name || t('floatingAvatar.agentFallback');
  const cleanLatest = String(displayLatest || '').replace(/^Ai[:：]\s*/, '');
  const shakeMessage = shakeDialogue || (shakeState === 'speechless' ? t('floatingAvatar.shakeTooLong') : '');
  const cleanBubbleText = String(shakeMessage || bubbleText || '').replace(/^Ai[:：]\s*/, '').trim();
  const topBubbleVisible = compactWindowMode && cleanBubbleText && (!bubbleDismissed || shakeMessageVisible);
  const topBubbleStyle = {
    left: clamp(clampedPosition.x - 124, edgeGap, Math.max(edgeGap, window.innerWidth - 318)),
    top: clamp(clampedPosition.y - 92, edgeGap, Math.max(edgeGap, window.innerHeight - 92)),
  };
  const expressionClass = shakeState
    ? `floating-avatar-${shakeState}`
    : (!reminderPaused && pendingConfirm ? 'floating-avatar-alert' : `floating-avatar-${avatarExpression || 'idle'}`);
  const reminderOptions = [
    ['30m', t('floatingAvatar.mute30m')],
    ['1h', t('floatingAvatar.mute1h')],
    ['tomorrow', t('floatingAvatar.muteTomorrow')],
    ['manual', t('floatingAvatar.muteManual')],
  ];
  const menuCheck = (active, label) => active ? `✓ ${label}` : label;

  function noteAction() {
    if (guideVisible) setGuideVisible(false);
    if (bubbleDismissed === false) setBubbleDismissed(true);
  }

  function updateDragShake(clientX, clientY) {
    const now = Date.now();
    const shake = shakeRef.current;
    if (!shake.startedAt) {
      shake.startedAt = now;
      shake.lastPoint = {x: clientX, y: clientY};
      shake.distance = 0;
      shake.previewFired = false;
      shake.vomitFired = false;
      return;
    }
    const last = shake.lastPoint || {x: clientX, y: clientY};
    shake.distance += Math.abs(clientX - last.x) + Math.abs(clientY - last.y);
    shake.lastPoint = {x: clientX, y: clientY};
    const elapsed = now - shake.startedAt;
    // 搖晃中即時改表情：先暈（sad），1.2 秒後抽問候池，3 秒後進嘔吐彩蛋。
    if (!shake.previewFired && shake.distance > 240) {
      setShakeState((prev) => (prev === 'speechless' ? prev : 'sad'));
    }
    if (!shake.previewFired && elapsed >= shakePreviewDelayMs && shake.distance > shakePreviewDistance) {
      const preview = randomShakePreview(shakeDialogueOptions);
      shake.previewFired = true;
      setShakeState(preview.expression || 'sad');
      setShakeDialogue(preview.text || t('floatingAvatar.shakeTooLong'));
      setBubbleDismissed(false);
      setShakeMessageVisible(true);
      onShakePreview?.(preview);
    }
    if (!shake.vomitFired && elapsed >= shakeVomitDelayMs && shake.distance > shakeVomitDistance) {
      const dialogue = t('floatingAvatar.shakeTooLong');
      shake.vomitFired = true;
      setShakeState('speechless');
      setShakeDialogue(dialogue);
      setBubbleDismissed(false);
      setShakeMessageVisible(true);
      onShakeTooLong?.(dialogue);
    }
  }

  function startDrag(event) {
    if (event.button !== 0) return;
    if (guideVisible) setGuideVisible(false);
    onCompactDragStart?.();
    if (shakeResetRef.current) {
      window.clearTimeout(shakeResetRef.current);
      shakeResetRef.current = null;
    }
    setShakeDialogue('');
    dragRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      startScreenX: event.screenX,
      startScreenY: event.screenY,
      originX: clampedPosition.x,
      originY: clampedPosition.y,
      moved: false,
    };
    const shakePoint = compactWindowMode ? {x: event.screenX, y: event.screenY} : {x: event.clientX, y: event.clientY};
    shakeRef.current = {startedAt: Date.now(), lastPoint: shakePoint, distance: 0, previewFired: false, vomitFired: false};
    event.currentTarget.setPointerCapture?.(event.pointerId);
  }

  function moveDrag(event) {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    const pointerX = compactWindowMode ? event.screenX : event.clientX;
    const pointerY = compactWindowMode ? event.screenY : event.clientY;
    const startX = compactWindowMode ? drag.startScreenX : drag.startX;
    const startY = compactWindowMode ? drag.startScreenY : drag.startY;
    const dx = pointerX - startX;
    const dy = pointerY - startY;
    if (Math.abs(dx) + Math.abs(dy) > 4) drag.moved = true;
    if (compactWindowMode) {
      onCompactDrag?.(dx, dy);
      updateDragShake(pointerX, pointerY);
      return;
    }
    const maxX = Math.max(edgeGap, window.innerWidth - avatarFrameWidth - edgeGap);
    const maxY = Math.max(edgeGap, window.innerHeight - avatarFrameHeight - edgeGap);
    onPositionChange?.({
      x: clamp(drag.originX + dx, edgeGap, maxX),
      y: clamp(drag.originY + dy, edgeGap, maxY),
    });
    updateDragShake(pointerX, pointerY);
  }

  function endDrag(event) {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    event.currentTarget.releasePointerCapture?.(event.pointerId);
    window.setTimeout(() => { dragRef.current = null; }, 0);
    if (compactWindowMode && drag.moved) {
      onCompactDragEnd?.();
    }
    if (shakeResetRef.current) window.clearTimeout(shakeResetRef.current);
    shakeResetRef.current = window.setTimeout(() => {
      setShakeState(null);
      setShakeMessageVisible(false);
      setShakeDialogue('');
      shakeRef.current = {startedAt: 0, lastPoint: null, distance: 0, previewFired: false, vomitFired: false};
      shakeResetRef.current = null;
    }, 1200);
  }

  // 單擊：顯示左側提示 + 開/關迷你框。雙擊：恢復主系統。
  // 用 220ms 計時器分辨單擊與雙擊，雙擊時取消尚未觸發的單擊動作。
  function performSingleClick() {
    if (dragRef.current?.moved) return;
    setBubbleDismissed(true);
    setGuideVisible(true);
    setPanelOpen((open) => !open);
    setContextOpen(false);
  }

  function handleAvatarClick() {
    if (dragRef.current?.moved) return;
    if (clickTimerRef.current) return;
    clickTimerRef.current = window.setTimeout(() => {
      clickTimerRef.current = null;
      performSingleClick();
    }, 220);
  }

  function handleAvatarDoubleClick(event) {
    event.preventDefault();
    if (clickTimerRef.current) {
      window.clearTimeout(clickTimerRef.current);
      clickTimerRef.current = null;
    }
    setPanelOpen(false);
    onRestore?.();
  }

  function handleDrop(event) {
    event.preventDefault();
    event.stopPropagation();
    noteAction();
    setDropActive(false);
    const files = Array.from(event.dataTransfer?.files || []);
    onDropFiles?.(files);
  }

  return (
    <div className="floating-avatar-stage" aria-label={t('floatingAvatar.stageLabel')}>
      {topBubbleVisible && (
        <aside className="floating-avatar-top-bubble" style={topBubbleStyle}>
          <strong>{speakerName}</strong>
          <span>{expanded ? cleanBubbleText : truncateStatus(cleanBubbleText, 72)}</span>
          {cleanBubbleText.length > 72 && (
            <button type="button" onClick={() => setExpanded((value) => !value)}>
              {expanded ? t('floatingAvatar.less') : t('floatingAvatar.more')}
            </button>
          )}
        </aside>
      )}
      <div
        ref={rootRef}
        className={`floating-avatar-root ${compactWindowMode ? 'floating-avatar-compact-root' : ''} ${flyingBack ? 'floating-avatar-flyback' : ''} ${showBody ? `floating-avatar-body-active floating-avatar-body-${bodyMode}` : ''}`}
        style={{
          left: clampedPosition.x,
          top: clampedPosition.y,
          width: avatarFrameWidth,
          height: avatarFrameHeight,
          '--floating-avatar-size': `${avatarSize}px`,
          '--floating-avatar-frame-width': `${avatarFrameWidth}px`,
          '--floating-avatar-frame-height': `${avatarFrameHeight}px`,
        }}
        onContextMenu={(event) => {
          event.preventDefault();
          noteAction();
          setContextOpen((open) => !open);
          setPanelOpen(false);
          setReminderPickerOpen(false);
          setAgentPickerOpen(false);
        }}
      >
        {((!compactWindowMode && guideVisible) || (compactWindowMode && panelOpen)) && (
          <aside
            className="floating-avatar-left-tip"
            style={compactWindowMode ? compactTipStyle : {left: leftTipLeft - clampedPosition.x, top: 4}}
          >
            {t('floatingAvatar.leftHint')}
          </aside>
        )}
        <button
          type="button"
          className={`floating-avatar-button ${expressionClass} ${dropActive ? 'floating-avatar-drop-active' : ''}`}
          title={t('floatingAvatar.doubleClickRestore')}
          aria-label={t('floatingAvatar.avatarAria', {name: persona?.name || t('floatingAvatar.agentFallback')})}
          onPointerDown={startDrag}
          onPointerMove={moveDrag}
          onPointerUp={endDrag}
          onPointerCancel={endDrag}
          onClick={handleAvatarClick}
          onDoubleClick={handleAvatarDoubleClick}
          onDragOver={(event) => {
            event.preventDefault();
            event.dataTransfer.dropEffect = 'copy';
            setDropActive(true);
          }}
          onDragLeave={() => setDropActive(false)}
          onDrop={handleDrop}
        >
          {showBody ? (
            <AnimatedFullBodyAvatar
              src={fullBodyAvatarSrc || avatarSrc}
              fallbackSrc={avatarSrc}
              pack={fullBodyAvatarKey}
              mode={bodyMode}
            />
          ) : (
            <img className="floating-avatar-img" src={avatarSrc} alt="" draggable={false} />
          )}
        </button>
        {dropActive && <span className="floating-avatar-drop-label">{t('floatingAvatar.releaseToAdd')}</span>}
        {!compactWindowMode && pendingConfirm && !reminderPaused && (
        <aside className="floating-avatar-right-status" style={{left: rightTipLeft - clampedPosition.x, top: 1}}>
          <strong>{displayStatus}</strong>
          <span>{expanded ? displayLatest : truncateStatus(displayLatest)}</span>
          {String(displayLatest || '').length > 42 && (
            <button type="button" onClick={() => setExpanded((value) => !value)}>
              {expanded ? t('floatingAvatar.less') : t('floatingAvatar.more')}
            </button>
          )}
        </aside>
        )}
        {contextOpen && (
          <nav className="floating-avatar-menu" aria-label={t('floatingAvatar.menuLabel')}>
            <button type="button" onClick={onRestore}>{t('floatingAvatar.restore')}</button>
            <button type="button" onClick={onOpenSettings}>{t('floatingAvatar.settings')}</button>
            <button
              type="button"
              className={chatMode ? 'floating-avatar-body-active-btn' : ''}
              aria-pressed={chatMode}
              onClick={() => {
                if (onChatModeToggle) {
                  onChatModeToggle(!chatMode);
                } else {
                  onChatModeChange?.(!chatMode);
                }
                setReminderPickerOpen(false);
                setAgentPickerOpen(false);
              }}
            >
              {menuCheck(chatMode, chatMode ? t('floatingAvatar.chatModeOff') : t('floatingAvatar.chatModeOn'))}
            </button>
            <div className="floating-avatar-body-switch" role="group" aria-label={t('floatingAvatar.bodySwitchLabel')}>
              {[['head', 'bodyHead'], ['full', 'bodyFull']].map(([mode, key]) => (
                <button
                  key={mode}
                  type="button"
                  className={bodyMode === mode ? 'floating-avatar-body-active-btn' : ''}
                  aria-pressed={bodyMode === mode}
                  onClick={() => setBodyMode(mode)}
                >
                  {menuCheck(bodyMode === mode, t(`floatingAvatar.${key}`))}
                </button>
              ))}
            </div>
            <button
              type="button"
              onClick={() => {
                setReminderPickerOpen((open) => !open);
                setAgentPickerOpen(false);
              }}
            >
              {reminderPaused ? t('floatingAvatar.reminderPaused') : t('floatingAvatar.reminderSettings')}
            </button>
            <button
              type="button"
              onClick={() => {
                setAgentPickerOpen((open) => !open);
                setReminderPickerOpen(false);
              }}
            >
              {t('floatingAvatar.switchAgent')}
            </button>
            <button type="button" className="floating-avatar-danger" onClick={onQuit}>{t('floatingAvatar.quit')}</button>
            {reminderPickerOpen && (
              <div className="floating-avatar-reminder-picker">
                {reminderPaused && <small>{reminderLabel || t('floatingAvatar.reminderPaused')}</small>}
                {reminderOptions.map(([mode, label]) => (
                  <button
                    key={mode}
                    type="button"
                    onClick={() => {
                      onSetReminderMode?.(mode);
                      setReminderPickerOpen(false);
                      setContextOpen(false);
                    }}
                  >
                    {label}
                  </button>
                ))}
                {reminderPaused && (
                  <button
                    type="button"
                    className="floating-avatar-reminder-resume"
                    onClick={() => {
                      onSetReminderMode?.('resume');
                      setReminderPickerOpen(false);
                      setContextOpen(false);
                    }}
                  >
                    {t('floatingAvatar.reminderResume')}
                  </button>
                )}
              </div>
            )}
            {agentPickerOpen && (
              <div className="floating-avatar-agent-picker">
                {(personas || []).map((item) => (
                  <button
                    key={item.id || item.name}
                    type="button"
                    className={item.id === activePersonaId ? 'floating-avatar-agent-active' : ''}
                    onClick={() => {
                      onSwitchAgent?.(item.id);
                      setAgentPickerOpen(false);
                      setContextOpen(false);
                    }}
                  >
                    <img src={item.avatarSrc || avatarSrc} alt="" />
                    <div>
                      <strong>{item.name || t('floatingAvatar.agentFallback')}</strong>
                      <span>{adapterLabel || t('floatingAvatar.adapterFallback')}</span>
                      <small>{[item.identity, item.personality].filter(Boolean).join(' / ') || t('floatingAvatar.agentSummaryFallback')}</small>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </nav>
        )}
      </div>

      {panelOpen && visibleCandidates.length > 0 && (
        <div className="floating-avatar-candidate-stack" style={compactWindowMode ? compactStackStyle : {left: stackLeft, top: stackTop}}>
          <small>{exclusiveCandidates ? t('floatingAvatar.singleSelect') : t('floatingAvatar.multiSelect')}</small>
          {visibleCandidates.map((candidate) => (
            <button
              key={candidate.id}
              type="button"
              className={selectedSet.has(candidate.id) ? 'floating-avatar-candidate-active' : ''}
              onClick={() => {
                noteAction();
                onSelectCandidate?.(candidate.id);
              }}
            >
              {candidateText(candidate)}
            </button>
          ))}
        </div>
      )}

      {compactWindowMode && installCandidate && (
        <section
          className="floating-avatar-install-bubble"
          style={compactPanelStyle}
          aria-label={t('floatingAvatar.installTitle')}
        >
          <header><strong>{t('floatingAvatar.installTitle')}</strong></header>
          <div className="floating-avatar-install-steps">
            {installArmed ? t('floatingAvatar.installStep2') : t('floatingAvatar.installStep1')}
          </div>
          {installArmed && (
            <button
              type="button"
              className="floating-avatar-install-send"
              onClick={() => { setInstallArmed(false); onConfirmInstall?.(); }}
            >
              {t('floatingAvatar.installSend')}
            </button>
          )}
          <div className="floating-avatar-install-body">
            <strong>{installCandidate.name}</strong>
            <span>{t('floatingAvatar.installType')}：{installCandidate.typeLabel}</span>
            <span>{t('floatingAvatar.installSource')}：{installSourceLabel(installCandidate)}</span>
            <small>{installRiskSummary(installCandidate, t)}</small>
          </div>
          <div className="floating-avatar-install-actions">
            <button type="button" onClick={() => setInstallArmed(true)} disabled={installArmed}>
              {t('floatingAvatar.installConfirm')}
            </button>
            <button
              type="button"
              className="floating-avatar-danger"
              onClick={() => { setInstallArmed(false); onCancelInstall?.(); }}
            >
              {t('floatingAvatar.installCancel')}
            </button>
          </div>
        </section>
      )}

      {panelOpen && (
        <section className="floating-avatar-mini-panel" style={compactWindowMode ? compactPanelStyle : {left: panelLeft, top: panelTop}} aria-label={t('floatingAvatar.panelLabel')}>
          <header>
            <div>
              <span>{chatMode ? t('floatingAvatar.chatPanelKicker') : t('floatingAvatar.panelKicker')}</span>
              <strong>{persona?.name || t('floatingAvatar.agentFallback')}</strong>
            </div>
            <button type="button" onClick={() => setPanelOpen(false)} aria-label={t('common.close')}>×</button>
          </header>
          <div className="floating-avatar-mini-status">
            {pendingConfirm && <strong>{displayStatus}</strong>}
            <p><strong>{speakerName}：</strong>{expanded ? cleanLatest : truncateStatus(cleanLatest, 86)}</p>
          </div>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              noteAction();
              if (!chatMode && !scheduleConfirmPending && looksLikeSchedule(draft)) {
                setScheduleConfirmPending(true);
                return;
              }
              setScheduleConfirmPending(false);
              onSubmit?.(draft);
            }}
          >
            <textarea
              value={draft}
              rows={3}
              placeholder={chatMode ? t('floatingAvatar.chatInputPlaceholder') : t('floatingAvatar.inputPlaceholder')}
              onChange={(event) => {
                noteAction();
                setScheduleConfirmPending(false);
                onDraftChange?.(event.target.value);
              }}
            />
            {scheduleConfirmPending ? (
              <div className="floating-avatar-schedule-confirm">
                <small>{t('floatingAvatar.scheduleConfirmHint')}</small>
                <div className="floating-avatar-schedule-actions">
                  <button type="submit">{t('floatingAvatar.scheduleConfirm')}</button>
                  <button
                    type="button"
                    onClick={() => { setScheduleConfirmPending(false); onSubmit?.(draft); }}
                  >
                    {t('floatingAvatar.scheduleCancel')}
                  </button>
                </div>
              </div>
            ) : (
              <button type="submit">{t('floatingAvatar.send')}</button>
            )}
          </form>
        </section>
      )}
    </div>
  );
}

function truncateStatus(text, limit = 42) {
  const source = String(text || '');
  return source.length > limit ? `${source.slice(0, limit)}...` : source;
}
