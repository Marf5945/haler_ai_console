import React, {useEffect, useMemo, useRef, useState} from 'react';

const defaultAvatarSize = 64;
const edgeGap = 12;
const shakePreviewDelayMs = 2600;
const shakeVomitDelayMs = 4500;
const shakeAxisThreshold = 14;
const shakeReversalDistance = 18;
const shakePreviewScore = 150;
const shakeVomitScore = 320;
const shakePreviewReversals = 2;
const shakeVomitReversals = 5;

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

function sanitizeFloatingAvatarText(value) {
  let text = String(value || '');
  const pendingMarkerIndex = text.indexOf('\u2063pending:');
  if (pendingMarkerIndex >= 0) {
    text = text.slice(0, pendingMarkerIndex);
  }
  text = text.replace(/\s*pending:[A-Za-z0-9_-]+$/g, '');
  return text.replace(/^Ai[:：]\s*/, '').trim();
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

function dominantShakeVector(dx, dy) {
  const absX = Math.abs(dx);
  const absY = Math.abs(dy);
  if (absX < shakeAxisThreshold && absY < shakeAxisThreshold) return null;
  if (absX >= absY) {
    return {axis: 'x', direction: dx >= 0 ? 1 : -1, magnitude: absX};
  }
  return {axis: 'y', direction: dy >= 0 ? 1 : -1, magnitude: absY};
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
  onSwitchAgent,
  onSetReminderMode,
  compactWindowMode = false,
  onCompactDragStart,
  onCompactDrag,
  onCompactDragEnd,
  onCompactExpandChange,
  flyingBack = false,
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
  const [dragActive, setDragActive] = useState(false);
  const [shakeState, setShakeState] = useState(null);
  const [shakeMessageVisible, setShakeMessageVisible] = useState(false);
  const [shakeDialogue, setShakeDialogue] = useState('');
  const [bubbleDismissed, setBubbleDismissed] = useState(false);
  const [installArmed, setInstallArmed] = useState(false);
  const [scheduleConfirmPending, setScheduleConfirmPending] = useState(false);
  const rootRef = useRef(null);
  const dragRef = useRef(null);
  const clickTimerRef = useRef(null);
  const shakeResetRef = useRef(null);
  const shakeRef = useRef({
    startedAt: 0,
    lastPoint: null,
    score: 0,
    reversalCount: 0,
    lastVector: null,
    previewFired: false,
    vomitFired: false,
  });
  const clampedPosition = useMemo(() => {
    const maxX = Math.max(edgeGap, window.innerWidth - avatarSize - edgeGap);
    const maxY = Math.max(edgeGap, window.innerHeight - avatarSize - edgeGap);
    return {
      x: clamp(Number(position?.x ?? maxX), edgeGap, maxX),
      y: clamp(Number(position?.y ?? maxY), edgeGap, maxY),
    };
  }, [position?.x, position?.y]);

  useEffect(() => {
    if (!active || compactWindowMode) return undefined;
    const onResize = () => onPositionChange?.(clampedPosition);
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, [active, clampedPosition, compactWindowMode, onPositionChange]);

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
    onCompactExpandChange?.(!dragActive && (panelOpen || contextOpen || hasTopBubble || Boolean(installCandidate)));
    return undefined;
  }, [compactWindowMode, panelOpen, contextOpen, bubbleText, bubbleDismissed, shakeMessageVisible, installCandidate, dragActive]);

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

  if (!active) return null;

  const selectedSet = new Set(selectedCandidateIDs || []);
  const visibleCandidates = (candidates || []).slice(0, 5);
  const panelLeft = clampedPosition.x < window.innerWidth - 390
    ? clampedPosition.x + avatarSize + 14
    : clampedPosition.x - 342;
  const panelTop = clamp(clampedPosition.y - 18, edgeGap, Math.max(edgeGap, window.innerHeight - 410));
  const stackLeft = clamp(clampedPosition.x - 44, edgeGap, Math.max(edgeGap, window.innerWidth - 240));
  const stackTop = clamp(clampedPosition.y + avatarSize + 10, edgeGap, Math.max(edgeGap, window.innerHeight - 230));
  const leftTipLeft = clampedPosition.x > 250 ? clampedPosition.x - 236 : clampedPosition.x + avatarSize + 16;
  const rightTipLeft = clampedPosition.x < window.innerWidth - 360 ? clampedPosition.x + avatarSize + 16 : clampedPosition.x - 304;
  // compact 浮窗模式：面板要落在放大後的視窗內（非整個螢幕座標）。
  const compactPanelTop = clamp(clampedPosition.y - 10, edgeGap, Math.max(edgeGap, window.innerHeight - 430));
  const compactPanelLeft = clamp(clampedPosition.x + avatarSize + 18, edgeGap, Math.max(edgeGap, window.innerWidth - 340));
  const compactPanelStyle = {
    left: compactPanelLeft,
    top: compactPanelTop,
    width: Math.min(328, Math.max(220, window.innerWidth - compactPanelLeft - edgeGap)),
    maxHeight: Math.max(180, window.innerHeight - compactPanelTop - edgeGap),
  };
  const compactStackTop = clampedPosition.y + avatarSize + 12;
  const compactStackStyle = {
    left: clamp(clampedPosition.x - 82, edgeGap, Math.max(edgeGap, window.innerWidth - 230)),
    top: compactStackTop,
  };
  // compact 模式左側提示：放在頭像左邊（相對 root 為負 left），寬度依左側可用空間。
  const compactTipPreferredWidth = 220;
  const compactTipGap = 12;
  const compactTipOnLeft = clampedPosition.x >= compactTipPreferredWidth + edgeGap + compactTipGap;
  const compactTipWidth = compactTipOnLeft
    ? compactTipPreferredWidth
    : clamp(window.innerWidth - clampedPosition.x - avatarSize - compactTipGap - edgeGap, 180, 240);
  const compactTipStyle = compactTipOnLeft
    ? {left: -(compactTipWidth + compactTipGap), top: 0, width: compactTipWidth}
    : {left: avatarSize + compactTipGap, top: 0, width: compactTipWidth};
  const displayStatus = pendingConfirm?.title || statusTitle || t('floatingAvatar.statusIdle');
  const displayLatest = pendingConfirm?.reason || latestText || statusText || t('floatingAvatar.latestIdle');
  const safeDisplayLatest = sanitizeFloatingAvatarText(displayLatest);
  const safeCleanLatest = safeDisplayLatest;
  const speakerName = persona?.name || t('floatingAvatar.agentFallback');
  const shakeMessage = shakeDialogue || (shakeState === 'speechless' ? t('floatingAvatar.shakeTooLong') : '');
  const safeCleanBubbleText = sanitizeFloatingAvatarText(shakeMessage || bubbleText || '');
  const topBubbleVisible = compactWindowMode && safeCleanBubbleText && (!bubbleDismissed || shakeMessageVisible);
  const compactBubbleWidth = Math.min(306, Math.max(180, window.innerWidth - edgeGap * 2));
  const compactBubbleGap = 16;
  const compactBubbleOnRight = clampedPosition.x <= window.innerWidth - (compactBubbleWidth + avatarSize + edgeGap * 2);
  const topBubbleStyle = {
    left: clamp(
      compactBubbleOnRight
        ? clampedPosition.x + avatarSize + compactBubbleGap
        : clampedPosition.x - compactBubbleWidth - compactBubbleGap,
      edgeGap,
      Math.max(edgeGap, window.innerWidth - compactBubbleWidth - edgeGap),
    ),
    top: clamp(clampedPosition.y - 6, edgeGap, Math.max(edgeGap, window.innerHeight - 136)),
    width: compactBubbleWidth,
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
      shake.score = 0;
      shake.reversalCount = 0;
      shake.lastVector = null;
      shake.previewFired = false;
      shake.vomitFired = false;
      return;
    }
    const last = shake.lastPoint || {x: clientX, y: clientY};
    const dx = clientX - last.x;
    const dy = clientY - last.y;
    shake.lastPoint = {x: clientX, y: clientY};
    const elapsed = now - shake.startedAt;
    const vector = dominantShakeVector(dx, dy);
    if (vector) {
      const lastVector = shake.lastVector;
      if (
        lastVector
        && lastVector.axis === vector.axis
        && lastVector.direction !== vector.direction
        && lastVector.magnitude >= shakeReversalDistance
        && vector.magnitude >= shakeReversalDistance
      ) {
        shake.reversalCount += 1;
        shake.score += Math.min(lastVector.magnitude + vector.magnitude, 120);
      }
      shake.lastVector = vector;
    }
    // 搖晃中即時改表情：先暈（sad），1.2 秒後抽問候池，3 秒後進嘔吐彩蛋。
    if (!shake.previewFired && shake.reversalCount >= 1 && shake.score >= 90) {
      setShakeState((prev) => (prev === 'speechless' ? prev : 'sad'));
    }
    if (
      !shake.previewFired
      && elapsed >= shakePreviewDelayMs
      && shake.reversalCount >= shakePreviewReversals
      && shake.score >= shakePreviewScore
    ) {
      const preview = randomShakePreview(shakeDialogueOptions);
      shake.previewFired = true;
      setShakeState(preview.expression || 'sad');
      setShakeDialogue(preview.text || t('floatingAvatar.shakeTooLong'));
      setBubbleDismissed(false);
      setShakeMessageVisible(true);
      onShakePreview?.(preview);
    }
    if (
      !shake.vomitFired
      && elapsed >= shakeVomitDelayMs
      && shake.reversalCount >= shakeVomitReversals
      && shake.score >= shakeVomitScore
    ) {
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
    event.preventDefault();
    if (guideVisible) setGuideVisible(false);
    setDragActive(true);
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
    shakeRef.current = {
      startedAt: Date.now(),
      lastPoint: shakePoint,
      score: 0,
      reversalCount: 0,
      lastVector: null,
      previewFired: false,
      vomitFired: false,
    };
    event.currentTarget.setPointerCapture?.(event.pointerId);
  }

  function moveDrag(event) {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    event.preventDefault();
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
    const maxX = Math.max(edgeGap, window.innerWidth - avatarSize - edgeGap);
    const maxY = Math.max(edgeGap, window.innerHeight - avatarSize - edgeGap);
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
    setDragActive(false);
    if (compactWindowMode && drag.moved) {
      onCompactDragEnd?.();
    }
    if (shakeResetRef.current) window.clearTimeout(shakeResetRef.current);
    shakeResetRef.current = window.setTimeout(() => {
      setShakeState(null);
      setShakeMessageVisible(false);
      setShakeDialogue('');
      shakeRef.current = {
        startedAt: 0,
        lastPoint: null,
        score: 0,
        reversalCount: 0,
        lastVector: null,
        previewFired: false,
        vomitFired: false,
      };
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
        <aside className="floating-avatar-top-bubble floating-avatar-side-bubble" style={topBubbleStyle}>
          <strong>{speakerName}</strong>
          <span>{expanded ? safeCleanBubbleText : truncateStatus(safeCleanBubbleText, 72)}</span>
          {safeCleanBubbleText.length > 72 && (
            <button type="button" onClick={() => setExpanded((value) => !value)}>
              {expanded ? t('floatingAvatar.less') : t('floatingAvatar.more')}
            </button>
          )}
        </aside>
      )}
      <div
        ref={rootRef}
        className={`floating-avatar-root ${compactWindowMode ? 'floating-avatar-compact-root' : ''} ${flyingBack ? 'floating-avatar-flyback' : ''}`}
        style={{left: clampedPosition.x, top: clampedPosition.y, '--floating-avatar-size': `${avatarSize}px`}}
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
          <img src={avatarSrc} alt="" draggable={false} />
        </button>
        {dropActive && <span className="floating-avatar-drop-label">{t('floatingAvatar.releaseToAdd')}</span>}
        {!compactWindowMode && pendingConfirm && !reminderPaused && (
        <aside className="floating-avatar-right-status" style={{left: rightTipLeft - clampedPosition.x, top: 1}}>
          <strong>{displayStatus}</strong>
          <span>{expanded ? safeDisplayLatest : truncateStatus(safeDisplayLatest)}</span>
          {String(safeDisplayLatest || '').length > 42 && (
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
              <span>{t('floatingAvatar.panelKicker')}</span>
              <strong>{persona?.name || t('floatingAvatar.agentFallback')}</strong>
            </div>
            <button type="button" onClick={() => setPanelOpen(false)} aria-label={t('common.close')}>×</button>
          </header>
          <div className="floating-avatar-mini-status">
            {pendingConfirm && <strong>{displayStatus}</strong>}
            <p><strong>{speakerName}：</strong>{expanded ? safeCleanLatest : truncateStatus(safeCleanLatest, 86)}</p>
          </div>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              noteAction();
              if (!scheduleConfirmPending && looksLikeSchedule(draft)) {
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
              placeholder={t('floatingAvatar.inputPlaceholder')}
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
