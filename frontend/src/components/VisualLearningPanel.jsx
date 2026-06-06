// VisualLearningPanel.jsx — Visual Capture Replay Learning Mode 前端面板。
//
// 規範依據：AI_Console_Spec_v4_3.md §14 + AI_Console_UI_v2_4.md §14.3
//
// 此元件是獨立區塊，由 App.jsx 引用。
// 整合所有 22 個 Visual Learning Wails bindings。
//
// 區段：
//   1. 引擎狀態（YOLO / OpenCV）
//   2. 學習模式控制
//   3. 待審候選項目
//   4. VL Review Cards
//   5. Canonical Labels
//   6. 元素辭典 / 動作辭典
//   7. Dry-Run 信心分
//   8. 安全匯出

import React, { useState, useEffect, useCallback, useRef } from 'react';
import useI18n from '../locales/useI18n';
import {
  // Engine status
  GetYOLOStatus,
  GetYOLODetailedStatus,
  GetOpenCVStatus,
  // Learning mode
  StartLearningMode,
  StopLearningMode,
  IsLearningModeActive,
  GetActiveLearningRun,
  // Detection
  DetectRegions,
  ProposeRegions,
  ResolveWindowsClickAnchor,
  // Canonical labels
  MapCanonicalLabel,
  GetCanonicalLabelSchema,
  // Element dictionary
  AddElementDictionaryEntry,
  ApproveElementDictionaryEntry,
  ListElementDictionary,
  // Action dictionary
  AddActionCandidate,
  ApproveActionCandidate,
  ListFormalActions,
  ListPendingActions,
  // Pending candidates
  ListPendingCandidates,
  GetPendingCandidateCount,
  RefreshPendingCandidateAges,
  // Safe export
  ExportVisualLearning,
  // VL Review cards
  AddVLReviewCard,
  ResolveVLReviewCard,
  ListOpenVLReviewCards,
  HasBlockingVLReview,
  // Dry-run confidence
  ComputeDryRunConfidence,
} from '../../wailsjs/go/main/App';

// ─────────────────────────────────────────
// callWails — 與 App.jsx 相同的 wrapper
// ─────────────────────────────────────────
function callWails(fn) {
  try {
    return Promise.resolve(fn());
  } catch (error) {
    return Promise.reject(error);
  }
}

// ─────────────────────────────────────────
// 常數
// ─────────────────────────────────────────
const POLL_INTERVAL_MS = 10000; // 10 秒輪詢
const PANEL_POSITION_STORAGE_KEY = 'visual_learning_monitor_position';
const PANEL_VIEWPORT_MARGIN = 18;
const PANEL_DEFAULT_WIDTH = 560;
const PANEL_DEFAULT_HEIGHT = 420;

function clampPositionToViewport(x, y, width = PANEL_DEFAULT_WIDTH, height = PANEL_DEFAULT_HEIGHT) {
  if (typeof window === 'undefined') return { x, y };
  const maxX = Math.max(PANEL_VIEWPORT_MARGIN, window.innerWidth - width - PANEL_VIEWPORT_MARGIN);
  const maxY = Math.max(PANEL_VIEWPORT_MARGIN, window.innerHeight - height - PANEL_VIEWPORT_MARGIN);
  return {
    x: Math.min(Math.max(PANEL_VIEWPORT_MARGIN, x), maxX),
    y: Math.min(Math.max(PANEL_VIEWPORT_MARGIN, y), maxY),
  };
}

function normalizeCandidate(candidate) {
  const bbox = candidate?.bbox_relative || candidate?.BBoxRelative || candidate?.bboxRelative || {};
  return {
    id: candidate?.region_id || candidate?.RegionID || candidate?.regionID || 'region',
    source: candidate?.source || candidate?.Source || 'opencv',
    elementTypeGuess: candidate?.element_type_guess || candidate?.ElementTypeGuess || '',
    confidence: candidate?.confidence ?? candidate?.Confidence ?? 0,
    bbox: {
      x: Number(bbox.x ?? bbox.X ?? 0),
      y: Number(bbox.y ?? bbox.Y ?? 0),
      w: Number(bbox.w ?? bbox.W ?? 0),
      h: Number(bbox.h ?? bbox.H ?? 0),
    },
  };
}

function isDrawableCandidate(candidate) {
  const bbox = candidate?.bbox || {};
  const isIcon = candidate?.source?.includes('icon');
  const area = (bbox.w || 0) * (bbox.h || 0);
  if (isIcon) return true;
  if ((bbox.w > 0.58 && bbox.h < 0.22) || (bbox.h > 0.62 && bbox.w < 0.14)) return false;
  return area <= 0.24;
}

function findCandidateAt(candidates, nx, ny) {
  const normalized = (candidates || []).map(normalizeCandidate).filter(isDrawableCandidate);
  const hits = normalized
    .filter(({ bbox }) => nx >= bbox.x && nx <= bbox.x + bbox.w && ny >= bbox.y && ny <= bbox.y + bbox.h)
    .sort((a, b) => (a.bbox.w * a.bbox.h) - (b.bbox.w * b.bbox.h));
  if (hits[0]) return hits[0];

  // 沒有精準框時，用最近的小候選輔助 demo 點擊對位。
  return normalized
    .map((candidate) => {
      const cx = candidate.bbox.x + candidate.bbox.w / 2;
      const cy = candidate.bbox.y + candidate.bbox.h / 2;
      const dx = nx - cx;
      const dy = ny - cy;
      return { candidate, distance: Math.sqrt(dx * dx + dy * dy) };
    })
    .filter(({ distance }) => distance <= 0.045)
    .sort((a, b) => a.distance - b.distance)[0]?.candidate || null;
}

function pixelBBoxStyle(bbox, width, height) {
  if (!bbox || !width || !height) return null;
  const x = Number(bbox.x ?? bbox.X ?? 0);
  const y = Number(bbox.y ?? bbox.Y ?? 0);
  const w = Number(bbox.w ?? bbox.W ?? 0);
  const h = Number(bbox.h ?? bbox.H ?? 0);
  if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(w) || !Number.isFinite(h) || w <= 0 || h <= 0) {
    return null;
  }
  return {
    left: `${(x / width) * 100}%`,
    top: `${(y / height) * 100}%`,
    width: `${(w / width) * 100}%`,
    height: `${(h / height) * 100}%`,
  };
}

function stopMediaStream(stream) {
  stream?.getTracks?.().forEach((track) => track.stop());
}

// ─────────────────────────────────────────
// VisualLearningPanel — 主元件
// ─────────────────────────────────────────

export default function VisualLearningPanel({
  // 從 App.jsx 傳入的外部狀態（可選，面板也能自行管理）
  learningActive: externalLearningActive,
  onLearningToggle: externalToggle,
  pendingCount: externalPendingCount,
  hasBlocking: externalHasBlocking,
  recentEvents = [],
  onClose,
  // 控制面板展開的區段
  defaultSection = 'status',
}) {
  const t = useI18n((s) => s.t);

  // ─── 引擎狀態 ─────────────────────────
  const [yoloReady, setYoloReady] = useState(false);
  const [yoloStatus, setYoloStatus] = useState(null);
  const [opencvReady, setOpencvReady] = useState(false);
  const [engineLoading, setEngineLoading] = useState(true);

  // ─── 學習模式 ─────────────────────────
  const [learningActive, setLearningActive] = useState(externalLearningActive ?? false);
  const [activeLearningRun, setActiveLearningRun] = useState(null);

  // ─── 待審候選 ─────────────────────────
  const [pendingCandidates, setPendingCandidates] = useState([]);
  const [pendingActions, setPendingActions] = useState([]);
  const [pendingCount, setPendingCount] = useState(externalPendingCount ?? 0);

  // ─── VL Review Cards ──────────────────
  const [vlReviewCards, setVlReviewCards] = useState([]);
  const [hasBlocking, setHasBlocking] = useState(externalHasBlocking ?? false);

  // ─── Canonical Labels ─────────────────
  const [labelSchema, setLabelSchema] = useState([]);

  // ─── 元素辭典 ─────────────────────────
  const [elemDict, setElemDict] = useState([]);
  const [elemDictFilter, setElemDictFilter] = useState('all'); // all | pending | approved

  // ─── 動作辭典 ─────────────────────────
  const [formalActions, setFormalActions] = useState([]);

  // ─── Dry-Run 信心分 ───────────────────
  const [dryRunScore, setDryRunScore] = useState(null);

  // ─── UI 狀態 ──────────────────────────
  const [activeSection, setActiveSection] = useState(defaultSection);
  const [exportBusy, setExportBusy] = useState(false);
  const [exportResult, setExportResult] = useState(null);
  const [panelPosition, setPanelPosition] = useState(null);
  const [minimized, setMinimized] = useState(false);
  const [screenCapture, setScreenCapture] = useState({
    busy: false,
    error: '',
    preview: '',
    width: 0,
    height: 0,
    candidates: [],
    click: null,
    matched: null,
    degraded: false,
    reason: '',
  });
  const panelRef = useRef(null);
  const dragRef = useRef(null);
  const pollRef = useRef(null);
  const screenImageDataRef = useRef(null);

  useEffect(() => {
    try {
      const saved = window.localStorage?.getItem(PANEL_POSITION_STORAGE_KEY);
      if (!saved) return;
      const parsed = JSON.parse(saved);
      if (Number.isFinite(parsed?.x) && Number.isFinite(parsed?.y)) {
        setPanelPosition(clampPositionToViewport(parsed.x, parsed.y));
      }
    } catch {
      /* monitor position is optional */
    }
  }, []);

  const clampPanelPosition = useCallback((x, y) => {
    const rect = panelRef.current?.getBoundingClientRect();
    return clampPositionToViewport(x, y, rect?.width, rect?.height);
  }, []);

  const savePanelPosition = useCallback((position) => {
    try {
      window.localStorage?.setItem(PANEL_POSITION_STORAGE_KEY, JSON.stringify(position));
    } catch {
      /* best-effort monitor preference */
    }
  }, []);

  useEffect(() => {
    if (!panelPosition) return undefined;
    const keepPanelVisible = () => {
      const rect = panelRef.current?.getBoundingClientRect();
      const next = clampPositionToViewport(panelPosition.x, panelPosition.y, rect?.width, rect?.height);
      if (next.x !== panelPosition.x || next.y !== panelPosition.y) {
        setPanelPosition(next);
        savePanelPosition(next);
      }
    };
    keepPanelVisible();
    window.addEventListener('resize', keepPanelVisible);
    return () => window.removeEventListener('resize', keepPanelVisible);
  }, [panelPosition, savePanelPosition]);

  const handlePanelDragStart = useCallback((event) => {
    if (event.button !== 0 || event.target?.closest?.('button, input, textarea, select, a')) return;
    const rect = panelRef.current?.getBoundingClientRect();
    if (!rect) return;
    dragRef.current = {
      offsetX: event.clientX - rect.left,
      offsetY: event.clientY - rect.top,
    };
    event.currentTarget.setPointerCapture?.(event.pointerId);
    event.preventDefault();
  }, []);

  const handlePanelDragMove = useCallback((event) => {
    if (!dragRef.current) return;
    const next = clampPanelPosition(
      event.clientX - dragRef.current.offsetX,
      event.clientY - dragRef.current.offsetY
    );
    setPanelPosition(next);
  }, [clampPanelPosition]);

  const handlePanelDragEnd = useCallback((event) => {
    if (!dragRef.current) return;
    const next = clampPanelPosition(
      event.clientX - dragRef.current.offsetX,
      event.clientY - dragRef.current.offsetY
    );
    dragRef.current = null;
    setPanelPosition(next);
    savePanelPosition(next);
  }, [clampPanelPosition, savePanelPosition]);

  // ════════════════════════════════════════
  // 資料載入
  // ════════════════════════════════════════

  const refreshEngineStatus = useCallback(async () => {
    setEngineLoading(true);
    try {
      const [yoloDetail, opencv] = await Promise.all([
        callWails(GetYOLODetailedStatus)
          .catch(() => callWails(GetYOLOStatus).then((available) => ({ available })).catch(() => ({ available: false }))),
        callWails(GetOpenCVStatus).catch(() => false),
      ]);
      setYoloStatus(yoloDetail || null);
      setYoloReady(!!yoloDetail?.available);
      setOpencvReady(opencv);
    } finally {
      setEngineLoading(false);
    }
  }, []);

  const refreshLearningState = useCallback(async () => {
    const [active, run, count, blocking] = await Promise.all([
      callWails(IsLearningModeActive).catch(() => false),
      callWails(GetActiveLearningRun).catch(() => null),
      callWails(GetPendingCandidateCount).catch(() => 0),
      callWails(HasBlockingVLReview).catch(() => false),
    ]);
    setLearningActive(!!active);
    setActiveLearningRun(run);
    setPendingCount(count || 0);
    setHasBlocking(!!blocking);
  }, []);

  const refreshPendingCandidates = useCallback(async () => {
    const [candidates, actions] = await Promise.all([
      callWails(ListPendingCandidates).catch(() => []),
      callWails(ListPendingActions).catch(() => []),
    ]);
    setPendingCandidates(Array.isArray(candidates) ? candidates : []);
    setPendingActions(Array.isArray(actions) ? actions : []);
  }, []);

  const refreshVLReviewCards = useCallback(async () => {
    const cards = await callWails(ListOpenVLReviewCards).catch(() => []);
    setVlReviewCards(Array.isArray(cards) ? cards : []);
  }, []);

  const refreshLabelSchema = useCallback(async () => {
    const schema = await callWails(GetCanonicalLabelSchema).catch(() => []);
    setLabelSchema(Array.isArray(schema) ? schema : []);
  }, []);

  const refreshElementDictionary = useCallback(async () => {
    const status = elemDictFilter === 'all' ? '' : elemDictFilter;
    const entries = await callWails(() => ListElementDictionary(status)).catch(() => []);
    setElemDict(Array.isArray(entries) ? entries : []);
  }, [elemDictFilter]);

  const refreshFormalActions = useCallback(async () => {
    const actions = await callWails(ListFormalActions).catch(() => []);
    setFormalActions(Array.isArray(actions) ? actions : []);
  }, []);

  // ════════════════════════════════════════
  // 初始載入 + 輪詢
  // ════════════════════════════════════════

  useEffect(() => {
    refreshEngineStatus();
    refreshLearningState();
    refreshLabelSchema();
  }, [refreshEngineStatus, refreshLearningState, refreshLabelSchema]);

  // 輪詢學習狀態（每 10 秒）
  useEffect(() => {
    pollRef.current = setInterval(() => {
      refreshLearningState();
    }, POLL_INTERVAL_MS);
    return () => clearInterval(pollRef.current);
  }, [refreshLearningState]);

  // 切換到特定區段時載入對應資料
  useEffect(() => {
    if (activeSection === 'candidates') {
      refreshPendingCandidates();
    } else if (activeSection === 'reviews') {
      refreshVLReviewCards();
    } else if (activeSection === 'elements') {
      refreshElementDictionary();
    } else if (activeSection === 'actions') {
      refreshFormalActions();
    }
  }, [activeSection, refreshPendingCandidates, refreshVLReviewCards, refreshElementDictionary, refreshFormalActions]);

  // elemDictFilter 變更時重載
  useEffect(() => {
    if (activeSection === 'elements') {
      refreshElementDictionary();
    }
  }, [elemDictFilter, activeSection, refreshElementDictionary]);

  // 同步外部 props
  useEffect(() => {
    if (externalLearningActive !== undefined) setLearningActive(externalLearningActive);
  }, [externalLearningActive]);
  useEffect(() => {
    if (externalPendingCount !== undefined) setPendingCount(externalPendingCount);
  }, [externalPendingCount]);
  useEffect(() => {
    if (externalHasBlocking !== undefined) setHasBlocking(externalHasBlocking);
  }, [externalHasBlocking]);

  // ════════════════════════════════════════
  // 動作 handlers
  // ════════════════════════════════════════

  const handleToggleLearning = useCallback(async () => {
    if (externalToggle) {
      externalToggle();
      return;
    }
    if (learningActive) {
      await callWails(StopLearningMode).catch(() => {});
      setLearningActive(false);
      setActiveLearningRun(null);
    } else {
      try {
        const run = await callWails(() => StartLearningMode('window-' + Date.now()));
        setActiveLearningRun(run);
        setLearningActive(true);
      } catch { /* fallback */ }
    }
  }, [learningActive, externalToggle]);

  const handleApproveElement = useCallback(async (id) => {
    await callWails(() => ApproveElementDictionaryEntry(id)).catch(() => {});
    refreshElementDictionary();
  }, [refreshElementDictionary]);

  const handleApproveAction = useCallback(async (id) => {
    await callWails(() => ApproveActionCandidate(id)).catch(() => {});
    refreshPendingCandidates();
    refreshFormalActions();
  }, [refreshPendingCandidates, refreshFormalActions]);

  const handleResolveVLReview = useCallback(async (cardId) => {
    await callWails(() => ResolveVLReviewCard(cardId)).catch(() => {});
    refreshVLReviewCards();
    refreshLearningState();
  }, [refreshVLReviewCards, refreshLearningState]);

  const handleRefreshAges = useCallback(async () => {
    await callWails(RefreshPendingCandidateAges).catch(() => {});
    refreshPendingCandidates();
  }, [refreshPendingCandidates]);

  const handleComputeDryRun = useCallback(async (base = 1.0, devicePenalty = 0.0, runtimeEvidence = 0.0) => {
    const score = await callWails(() => ComputeDryRunConfidence(base, devicePenalty, runtimeEvidence)).catch(() => null);
    setDryRunScore(score);
    return score;
  }, []);

  const handleCaptureScreen = useCallback(async () => {
    setScreenCapture((current) => ({ ...current, busy: true, error: '', matched: null, click: null }));
    let stream = null;
    try {
      if (!navigator.mediaDevices?.getDisplayMedia) {
        throw new Error('目前 WebView 不支援螢幕擷取 API');
      }
      stream = await navigator.mediaDevices.getDisplayMedia({
        video: { cursor: 'always' },
        audio: false,
      });
      const video = document.createElement('video');
      video.srcObject = stream;
      video.muted = true;
      video.playsInline = true;
      await video.play();
      if (!video.videoWidth || !video.videoHeight) {
        await new Promise((resolve) => {
          video.onloadedmetadata = resolve;
          setTimeout(resolve, 600);
        });
      }
      const sourceWidth = video.videoWidth || 1280;
      const sourceHeight = video.videoHeight || 720;
      // 保留小 icon 細節，同時限制送進 Go 的像素量。
      const scale = Math.min(1, 1280 / sourceWidth, 720 / sourceHeight);
      const width = Math.max(1, Math.round(sourceWidth * scale));
      const height = Math.max(1, Math.round(sourceHeight * scale));
      const canvas = document.createElement('canvas');
      canvas.width = width;
      canvas.height = height;
      const ctx = canvas.getContext('2d', { willReadFrequently: true });
      ctx.drawImage(video, 0, 0, width, height);
      const image = ctx.getImageData(0, 0, width, height);
      const imageData = Array.from(image.data);
      screenImageDataRef.current = imageData;
      const result = await callWails(() => ProposeRegions(imageData, width, height));
      const candidates = Array.isArray(result?.candidates) ? result.candidates.map(normalizeCandidate) : [];
      setScreenCapture({
        busy: false,
        error: '',
        preview: canvas.toDataURL('image/png'),
        width,
        height,
        candidates,
        click: null,
        matched: null,
        anchor: null,
        anchorError: '',
        degraded: !!result?.degraded,
        reason: result?.reason || '',
      });
    } catch (error) {
      setScreenCapture((current) => ({
        ...current,
        busy: false,
        error: error?.message || String(error),
      }));
    } finally {
      stopMediaStream(stream);
    }
  }, []);

  const handleScreenPreviewClick = useCallback((event) => {
    const rect = event.currentTarget.getBoundingClientRect();
    if (!rect.width || !rect.height || !screenCapture.width || !screenCapture.height) return;
    const x = ((event.clientX - rect.left) / rect.width) * screenCapture.width;
    const y = ((event.clientY - rect.top) / rect.height) * screenCapture.height;
    const nx = x / screenCapture.width;
    const ny = y / screenCapture.height;
    const matched = findCandidateAt(screenCapture.candidates, nx, ny);
    setScreenCapture((current) => ({
      ...current,
      click: { x, y, nx, ny },
      matched,
      anchor: null,
      anchorError: '',
    }));
    const imageData = screenImageDataRef.current;
    if (!Array.isArray(imageData) || imageData.length === 0) return;
    callWails(() => ResolveWindowsClickAnchor(
      imageData,
      screenCapture.width,
      screenCapture.height,
      Math.round(x),
      Math.round(y),
      JSON.stringify({ near_miss_px: 12, shape_fallback_radius: 48, manual_box_size: 28, crop_padding: 12 })
    ))
      .then((anchor) => {
        setScreenCapture((current) => ({
          ...current,
          anchor,
          anchorError: '',
        }));
      })
      .catch((error) => {
        setScreenCapture((current) => ({
          ...current,
          anchor: null,
          anchorError: error?.message || String(error),
        }));
      });
  }, [screenCapture]);

  const handleExport = useCallback(async (sections = ['elements', 'actions', 'labels']) => {
    setExportBusy(true);
    setExportResult(null);
    try {
      const result = await callWails(() => ExportVisualLearning(JSON.stringify(sections)));
      setExportResult(result);
    } catch (err) {
      setExportResult({ error: String(err) });
    } finally {
      setExportBusy(false);
    }
  }, []);

  // ════════════════════════════════════════
  // Section 導航列
  // ════════════════════════════════════════

  const sections = [
    { id: 'status',     label: t('vl.sectionStatus')     || '引擎狀態' },
    { id: 'candidates', label: t('vl.sectionCandidates') || '待審項目', badge: pendingCount },
    { id: 'reviews',    label: t('vl.sectionReviews')    || '審查卡片', badge: hasBlocking ? '!' : null },
    { id: 'elements',   label: t('vl.sectionElements')   || '元素辭典' },
    { id: 'actions',    label: t('vl.sectionActions')     || '動作辭典' },
    { id: 'dryrun',     label: t('vl.sectionDryRun')     || 'Dry-Run' },
    { id: 'export',     label: t('vl.sectionExport')     || '匯出' },
  ];

  // ════════════════════════════════════════
  // 渲染
  // ════════════════════════════════════════

  return (
    <div
      className={`vl-panel ${minimized ? 'vl-panel-minimized' : ''}`}
      ref={panelRef}
      style={panelPosition ? { left: `${panelPosition.x}px`, top: `${panelPosition.y}px`, right: 'auto' } : undefined}
    >
      {/* ── 標題列 ── */}
      <div
        className="vl-panel-header"
        onPointerDown={handlePanelDragStart}
        onPointerMove={handlePanelDragMove}
        onPointerUp={handlePanelDragEnd}
        onPointerCancel={handlePanelDragEnd}
      >
        <h3 className="vl-panel-title">
          {t('vl.panelTitle') || 'Visual Learning'}
        </h3>
        <div className="vl-panel-actions">
          <button
            className={`vl-learning-toggle ${learningActive ? 'vl-learning-active' : ''}`}
            onClick={handleToggleLearning}
            type="button"
          >
            {learningActive
              ? (t('vl.learningStop') || '停止示範')
              : (t('vl.learningStart') || '開始示範')}
          </button>
          <button
            className="vl-panel-icon-btn"
            onClick={() => setMinimized((value) => !value)}
            type="button"
            aria-label={minimized ? '展開監視面板' : '縮小監視面板'}
            title={minimized ? '展開' : '縮小'}
          >
            {minimized ? '+' : '-'}
          </button>
          <button
            className="vl-panel-icon-btn"
            onClick={onClose}
            type="button"
            aria-label="關閉監視面板"
            title="關閉"
          >
            x
          </button>
        </div>
      </div>

      {!minimized && (
        <>
      {/* ── 快捷狀態列 ── */}
      <div className="vl-status-bar">
        <span className={`vl-indicator ${yoloReady ? 'vl-ok' : 'vl-degraded'}`}>
          YOLO {yoloReady ? 'ON' : 'OFF'}
        </span>
        <span className={`vl-indicator ${opencvReady ? 'vl-ok' : 'vl-degraded'}`}>
          OpenCV {opencvReady ? 'ON' : 'OFF'}
        </span>
        {learningActive && activeLearningRun && (
          <span className="vl-indicator vl-active">
            {t('vl.runActive') || '示範中'}
          </span>
        )}
        {hasBlocking && (
          <span className="vl-indicator vl-blocking">
            {t('vl.hasBlocking') || '有阻擋審查'}
          </span>
        )}
      </div>

      {/* ── 區段導航 ── */}
      <nav className="vl-section-nav">
        {sections.map((sec) => (
          <button
            key={sec.id}
            className={`vl-nav-btn ${activeSection === sec.id ? 'vl-nav-active' : ''}`}
            onClick={() => setActiveSection(sec.id)}
            type="button"
          >
            {sec.label}
            {sec.badge != null && <span className="vl-badge">{sec.badge}</span>}
          </button>
        ))}
      </nav>

      {/* ── 區段內容 ── */}
      <div className="vl-section-content">
        {activeSection === 'status' && (
          <SectionEngineStatus
            t={t}
            yoloReady={yoloReady}
            opencvReady={opencvReady}
            yoloStatus={yoloStatus}
            loading={engineLoading}
            onRefresh={refreshEngineStatus}
            learningActive={learningActive}
            activeLearningRun={activeLearningRun}
            recentEvents={recentEvents}
            screenCapture={screenCapture}
            onCaptureScreen={handleCaptureScreen}
            onScreenPreviewClick={handleScreenPreviewClick}
            labelSchema={labelSchema}
          />
        )}

        {activeSection === 'candidates' && (
          <SectionPendingCandidates
            t={t}
            candidates={pendingCandidates}
            actions={pendingActions}
            pendingCount={pendingCount}
            onApproveAction={handleApproveAction}
            onRefreshAges={handleRefreshAges}
            onRefresh={refreshPendingCandidates}
          />
        )}

        {activeSection === 'reviews' && (
          <SectionVLReviews
            t={t}
            cards={vlReviewCards}
            hasBlocking={hasBlocking}
            onResolve={handleResolveVLReview}
            onRefresh={refreshVLReviewCards}
          />
        )}

        {activeSection === 'elements' && (
          <SectionElementDictionary
            t={t}
            entries={elemDict}
            filter={elemDictFilter}
            onFilterChange={setElemDictFilter}
            onApprove={handleApproveElement}
            onRefresh={refreshElementDictionary}
          />
        )}

        {activeSection === 'actions' && (
          <SectionActionDictionary
            t={t}
            formalActions={formalActions}
            onRefresh={refreshFormalActions}
          />
        )}

        {activeSection === 'dryrun' && (
          <SectionDryRun
            t={t}
            score={dryRunScore}
            onCompute={handleComputeDryRun}
          />
        )}

        {activeSection === 'export' && (
          <SectionExport
            t={t}
            busy={exportBusy}
            result={exportResult}
            onExport={handleExport}
          />
        )}
      </div>
        </>
      )}
    </div>
  );
}

// ════════════════════════════════════════════
// Sub-sections
// ════════════════════════════════════════════

// ── 1. 引擎狀態 ──

function SectionEngineStatus({ t, yoloReady, opencvReady, yoloStatus, loading, onRefresh, learningActive, activeLearningRun, recentEvents, screenCapture, onCaptureScreen, onScreenPreviewClick, labelSchema }) {
  return (
    <div className="vl-section">
      <div className="vl-section-row">
        <h4>{t('vl.engineTitle') || '推論引擎'}</h4>
        <button className="vl-btn-sm" onClick={onRefresh} type="button">
          {t('vl.refresh') || '重新整理'}
        </button>
      </div>

      {loading ? (
        <p className="vl-muted">{t('vl.engineLoading') || '載入中...'}</p>
      ) : (
        <div className="vl-engine-grid">
          <div className={`vl-engine-card ${yoloReady ? 'vl-ok' : 'vl-degraded'}`}>
            <strong>YOLOX-S Button</strong>
            <span>{yoloReady
              ? (t('vl.engineReady') || '就緒')
              : (t('vl.engineDegraded') || '降級（OpenCV-only）')}</span>
            {yoloStatus?.backend && <small className="vl-engine-detail">backend: {yoloStatus.backend}</small>}
            {yoloStatus?.reason && <small className="vl-engine-detail">{yoloStatus.reason}</small>}
          </div>
          <div className={`vl-engine-card ${opencvReady ? 'vl-ok' : 'vl-degraded'}`}>
            <strong>OpenCV</strong>
            <span>{opencvReady
              ? (t('vl.engineReady') || '就緒')
              : (t('vl.engineUnavailable') || '不可用')}</span>
          </div>
        </div>
      )}

      {learningActive && activeLearningRun && (
        <div className="vl-run-info">
          <h4>{t('vl.activeRun') || '目前示範回合'}</h4>
          <pre className="vl-code">{JSON.stringify(activeLearningRun, null, 2)}</pre>
        </div>
      )}

      {learningActive && (
        <div className="vl-run-info">
          <h4>{t('vl.recentClicks') || '最近示範點擊'}</h4>
          {recentEvents.length === 0 ? (
            <p className="vl-muted">{t('vl.noRecentClicks') || '開始示範後，點擊 App 內的按鈕即可記錄。'}</p>
          ) : (
            <div className="vl-click-list">
              {recentEvents.map((event) => (
                <div className="vl-click-item" key={event.id}>
                  <strong>{event.label || event.role || 'element'}</strong>
                  <span>{event.role || 'target'} · x:{event.x} y:{event.y}</span>
                  {event.selector && <small>{event.selector}</small>}
                  <em className={event.error ? 'vl-click-error' : event.saved ? 'vl-click-saved' : ''}>
                    {event.error ? (t('vl.clickSaveFailed') || '寫入失敗') : event.saved ? (t('vl.clickSaved') || '已寫入 trace') : (t('vl.clickSaving') || '寫入中')}
                  </em>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="vl-run-info">
        <div className="vl-section-row">
          <h4>{t('vl.screenProbe') || '截圖元素偵測'}</h4>
          <button className="vl-btn-sm" onClick={onCaptureScreen} type="button" disabled={screenCapture.busy}>
            {screenCapture.busy ? (t('vl.capturingScreen') || '擷取中...') : (t('vl.captureScreen') || '擷取畫面')}
          </button>
        </div>
        {screenCapture.error && <p className="vl-error-text">{screenCapture.error}</p>}
        {screenCapture.preview && (
          <>
            <div
              className="vl-screen-preview"
              onClick={onScreenPreviewClick}
              role="button"
              tabIndex={0}
              title={t('vl.screenPreviewHint') || '點預覽圖上的按鈕位置，對應 OpenCV 候選區域'}
            >
              <img src={screenCapture.preview} alt={t('vl.screenPreviewAlt') || '畫面截圖'} />
              {screenCapture.candidates.filter(isDrawableCandidate).map((candidate) => (
                <span
                  className={`vl-detection-box${candidate.source?.includes('icon') ? ' vl-detection-box-icon' : ''}${screenCapture.matched?.id === candidate.id ? ' vl-detection-box-hit' : ''}`}
                  key={candidate.id}
                  style={{
                    left: `${candidate.bbox.x * 100}%`,
                    top: `${candidate.bbox.y * 100}%`,
                    width: `${candidate.bbox.w * 100}%`,
                    height: `${candidate.bbox.h * 100}%`,
                  }}
                />
              ))}
              {pixelBBoxStyle(screenCapture.anchor?.anchor_bbox, screenCapture.width, screenCapture.height) && (
                <span
                  className="vl-anchor-box"
                  style={pixelBBoxStyle(screenCapture.anchor?.anchor_bbox, screenCapture.width, screenCapture.height)}
                />
              )}
              {screenCapture.click && (
                <span
                  className="vl-click-marker"
                  style={{
                    left: `${screenCapture.click.nx * 100}%`,
                    top: `${screenCapture.click.ny * 100}%`,
                  }}
                />
              )}
            </div>
            <p className="vl-muted">
              {(t('vl.detectedRegions') || '偵測區域')}: {screenCapture.candidates.filter(isDrawableCandidate).length}
              {screenCapture.matched && ` · ${(t('vl.matchedRegion') || '命中')}: ${screenCapture.matched.id}`}
              {screenCapture.degraded && ` · ${screenCapture.reason || 'OpenCV-only'}`}
            </p>
            {screenCapture.anchor && (
              <pre className="vl-code vl-anchor-result">{JSON.stringify({
                mode: screenCapture.anchor.mode,
                ok: screenCapture.anchor.ok,
                execution_hint: screenCapture.anchor.execution_hint,
                anchor_bbox: screenCapture.anchor.anchor_bbox,
                reason: screenCapture.anchor.reason || '',
                needs_review: screenCapture.anchor.needs_review,
              }, null, 2)}</pre>
            )}
            {screenCapture.anchorError && <p className="vl-error-text">{screenCapture.anchorError}</p>}
          </>
        )}
      </div>

      {labelSchema.length > 0 && (
        <div className="vl-label-schema">
          <h4>{t('vl.canonicalLabels') || 'Canonical Label 架構'}</h4>
          <div className="vl-tag-list">
            {labelSchema.map((label, i) => (
              <span key={i} className="vl-tag">
                {label.element_type}/{label.action_semantic}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// ── 2. 待審候選 ──

function SectionPendingCandidates({ t, candidates, actions, pendingCount, onApproveAction, onRefreshAges, onRefresh }) {
  return (
    <div className="vl-section">
      <div className="vl-section-row">
        <h4>
          {t('vl.pendingTitle') || '待審項目'}
          {pendingCount > 0 && <span className="vl-badge">{pendingCount}</span>}
        </h4>
        <div className="vl-btn-group">
          <button className="vl-btn-sm" onClick={onRefreshAges} type="button">
            {t('vl.refreshAges') || '更新時效'}
          </button>
          <button className="vl-btn-sm" onClick={onRefresh} type="button">
            {t('vl.refresh') || '重新整理'}
          </button>
        </div>
      </div>

      {candidates.length === 0 && actions.length === 0 ? (
        <p className="vl-muted">{t('vl.noPending') || '目前沒有待審項目'}</p>
      ) : (
        <>
          {candidates.length > 0 && (
            <div className="vl-list">
              <h5>{t('vl.elementCandidates') || '元素候選'}</h5>
              {candidates.map((c, i) => (
                <div key={c.id || i} className="vl-list-item">
                  <span className="vl-item-id">{c.id || `#${i + 1}`}</span>
                  <span className="vl-item-label">
                    {c.proposed_label?.element_type || c.fingerprint_id || '—'}
                  </span>
                  <span className="vl-item-score">
                    {typeof c.confidence === 'number' ? `${(c.confidence * 100).toFixed(0)}%` : ''}
                  </span>
                </div>
              ))}
            </div>
          )}

          {actions.length > 0 && (
            <div className="vl-list">
              <h5>{t('vl.actionCandidates') || '動作候選'}</h5>
              {actions.map((a, i) => (
                <div key={a.id || i} className="vl-list-item">
                  <span className="vl-item-id">{a.id || `#${i + 1}`}</span>
                  <span className="vl-item-label">
                    {a.risk_level || '—'} / {(a.steps || []).length} steps
                  </span>
                  <button
                    className="vl-btn-xs vl-btn-approve"
                    onClick={() => onApproveAction(a.id)}
                    type="button"
                  >
                    {t('vl.approve') || '核准'}
                  </button>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ── 3. VL Review Cards ──

function SectionVLReviews({ t, cards, hasBlocking, onResolve, onRefresh }) {
  return (
    <div className="vl-section">
      <div className="vl-section-row">
        <h4>
          {t('vl.reviewTitle') || '審查卡片'}
          {hasBlocking && <span className="vl-badge vl-badge-warn">!</span>}
        </h4>
        <button className="vl-btn-sm" onClick={onRefresh} type="button">
          {t('vl.refresh') || '重新整理'}
        </button>
      </div>

      {cards.length === 0 ? (
        <p className="vl-muted">{t('vl.noReviews') || '沒有待處理審查'}</p>
      ) : (
        <div className="vl-list">
          {cards.map((card, i) => (
            <div key={card.id || i} className={`vl-review-card ${card.level === 'blocking_review' ? 'vl-review-blocking' : ''}`}>
              <div className="vl-review-header">
                <span className="vl-review-level">{card.level || 'unknown'}</span>
                <span className="vl-review-source">{card.source_type || ''}</span>
              </div>
              <p className="vl-review-reason">{card.plain_reason || card.engineer_reason || '—'}</p>
              <button
                className="vl-btn-xs"
                onClick={() => onResolve(card.id)}
                type="button"
              >
                {t('vl.resolve') || '解決'}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ── 4. 元素辭典 ──

function SectionElementDictionary({ t, entries, filter, onFilterChange, onApprove, onRefresh }) {
  const filters = [
    { value: 'all',      label: t('vl.filterAll')      || '全部' },
    { value: 'pending',  label: t('vl.filterPending')  || '待審' },
    { value: 'approved', label: t('vl.filterApproved') || '已核准' },
  ];

  return (
    <div className="vl-section">
      <div className="vl-section-row">
        <h4>{t('vl.elemDictTitle') || '元素辭典'}</h4>
        <div className="vl-btn-group">
          {filters.map((f) => (
            <button
              key={f.value}
              className={`vl-btn-sm ${filter === f.value ? 'vl-btn-active' : ''}`}
              onClick={() => onFilterChange(f.value)}
              type="button"
            >
              {f.label}
            </button>
          ))}
          <button className="vl-btn-sm" onClick={onRefresh} type="button">
            {t('vl.refresh') || '重新整理'}
          </button>
        </div>
      </div>

      {entries.length === 0 ? (
        <p className="vl-muted">{t('vl.noElements') || '辭典目前為空'}</p>
      ) : (
        <div className="vl-list">
          {entries.map((entry, i) => (
            <div key={entry.id || i} className="vl-list-item">
              <span className="vl-item-id">{entry.fingerprint_id || entry.id || `#${i + 1}`}</span>
              <span className="vl-item-label">
                {entry.canonical_label?.element_type || '—'}
                {entry.canonical_label?.action_semantic ? ` / ${entry.canonical_label.action_semantic}` : ''}
              </span>
              <span className={`vl-item-status ${entry.status === 'approved' ? 'vl-ok' : 'vl-pending'}`}>
                {entry.status || 'pending'}
              </span>
              {entry.status !== 'approved' && (
                <button
                  className="vl-btn-xs vl-btn-approve"
                  onClick={() => onApprove(entry.id)}
                  type="button"
                >
                  {t('vl.approve') || '核准'}
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ── 5. 動作辭典 ──

function SectionActionDictionary({ t, formalActions, onRefresh }) {
  return (
    <div className="vl-section">
      <div className="vl-section-row">
        <h4>{t('vl.actionDictTitle') || '動作辭典（已核准）'}</h4>
        <button className="vl-btn-sm" onClick={onRefresh} type="button">
          {t('vl.refresh') || '重新整理'}
        </button>
      </div>

      {formalActions.length === 0 ? (
        <p className="vl-muted">{t('vl.noActions') || '目前沒有已核准的動作'}</p>
      ) : (
        <div className="vl-list">
          {formalActions.map((action, i) => (
            <div key={action.id || i} className="vl-action-card">
              <div className="vl-action-header">
                <span className="vl-item-id">{action.id || `#${i + 1}`}</span>
                <span className={`vl-risk vl-risk-${(action.risk_level || 'low').toLowerCase()}`}>
                  {action.risk_level || 'low'}
                </span>
              </div>
              {Array.isArray(action.steps) && (
                <ol className="vl-steps">
                  {action.steps.map((step, si) => (
                    <li key={si}>
                      <span className="vl-step-elem">{step.element_id || '?'}</span>
                      <span className="vl-step-label">
                        {step.canonical_label?.action_semantic || '—'}
                      </span>
                      {step.wait_after_ms > 0 && (
                        <span className="vl-step-wait">+{step.wait_after_ms}ms</span>
                      )}
                    </li>
                  ))}
                </ol>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ── 6. Dry-Run 信心分 ──

function SectionDryRun({ t, score, onCompute }) {
  const [base, setBase] = useState('1');
  const [devicePenalty, setDevicePenalty] = useState('0');
  const [runtimeEvidence, setRuntimeEvidence] = useState('0');

  const toNumber = (value, fallback = 0) => {
    const parsed = Number.parseFloat(value);
    return Number.isFinite(parsed) ? parsed : fallback;
  };

  const handleCompute = () => onCompute(
    toNumber(base, 1),
    toNumber(devicePenalty, 0),
    toNumber(runtimeEvidence, 0)
  );

  return (
    <div className="vl-section">
      <h4>{t('vl.dryRunTitle') || 'Dry-Run 信心分計算'}</h4>
      <p className="vl-muted">
        {t('vl.dryRunDesc') || '§14.5：新裝置 replay 必須先通過 dry-run。信心分 < 閾值時需人工審查。'}
      </p>

      <div className="vl-form-grid">
        <label>
          {t('vl.dryRunBase') || '基礎分 (base)'}
          <input type="number" step="0.1" min="0" max="1" inputMode="decimal"
            value={base} onChange={(e) => setBase(e.target.value)} />
        </label>
        <label>
          {t('vl.dryRunPenalty') || '裝置差異扣分'}
          <input type="number" step="0.1" min="0" max="1" inputMode="decimal"
            value={devicePenalty} onChange={(e) => setDevicePenalty(e.target.value)} />
        </label>
        <label>
          {t('vl.dryRunEvidence') || '運行時證據加分'}
          <input type="number" step="0.1" min="0" max="1" inputMode="decimal"
            value={runtimeEvidence} onChange={(e) => setRuntimeEvidence(e.target.value)} />
        </label>
      </div>

      <button className="vl-btn" onClick={handleCompute} type="button">
        {t('vl.computeScore') || '計算信心分'}
      </button>

      {score !== null && (
        <div className={`vl-score-result ${score >= 0.7 ? 'vl-ok' : score >= 0.4 ? 'vl-warn' : 'vl-degraded'}`}>
          <strong>{t('vl.scoreResult') || '信心分'}: </strong>
          <span>{(score * 100).toFixed(1)}%</span>
          <small>
            {score >= 0.7
              ? (t('vl.scorePass') || '通過 — 可自動 replay')
              : score >= 0.4
              ? (t('vl.scoreReview') || '需人工審查')
              : (t('vl.scoreFail') || '不通過 — 禁止 replay')}
          </small>
        </div>
      )}
    </div>
  );
}

// ── 7. 安全匯出 ──

function SectionExport({ t, busy, result, onExport }) {
  const [selectedSections, setSelectedSections] = useState(['elements', 'actions', 'labels']);

  const exportOptions = [
    { id: 'elements', label: t('vl.exportElements') || '元素辭典' },
    { id: 'actions',  label: t('vl.exportActions')  || '動作辭典' },
    { id: 'labels',   label: t('vl.exportLabels')   || 'Canonical Labels' },
  ];

  const toggle = (id) => {
    setSelectedSections((prev) =>
      prev.includes(id) ? prev.filter((s) => s !== id) : [...prev, id]
    );
  };

  return (
    <div className="vl-section">
      <h4>{t('vl.exportTitle') || '安全匯出（§14.4）'}</h4>
      <p className="vl-muted">
        {t('vl.exportDesc') || '匯出學習資料。僅包含已核准項目，不含 raw screenshots 或 PII。'}
      </p>

      <div className="vl-checkbox-group">
        {exportOptions.map((opt) => (
          <label key={opt.id} className="vl-checkbox">
            <input
              type="checkbox"
              checked={selectedSections.includes(opt.id)}
              onChange={() => toggle(opt.id)}
            />
            {opt.label}
          </label>
        ))}
      </div>

      <button
        className="vl-btn"
        onClick={() => onExport(selectedSections)}
        disabled={busy || selectedSections.length === 0}
        type="button"
      >
        {busy
          ? (t('vl.exporting') || '匯出中...')
          : (t('vl.exportBtn') || '匯出')}
      </button>

      {result && (
        <div className={`vl-export-result ${result.error ? 'vl-degraded' : 'vl-ok'}`}>
          {result.error
            ? <p>{t('vl.exportError') || '匯出失敗'}: {result.error}</p>
            : <pre className="vl-code">{JSON.stringify(result, null, 2)}</pre>}
        </div>
      )}
    </div>
  );
}
