export const visualLearningInteractiveSelector = [
  'button', 'a[href]', 'input', 'select', 'textarea',
  '[role="button"]', '[role="link"]', '[data-vl-target]',
].join(',');

export const learningReplayBlockedSelector = [
  '.rail-mode-record', '.sandbox-stop-overlay', '.reference-embed-popup',
].join(',');

export const learningRecordingBlockedSelector = learningReplayBlockedSelector;
export const visualReplayLastDemoDirective = '[[控制:回放剛剛示範]]';
export const legacyVisualReplayLastDemoDirective = '[[visual_replay:last_demo]]';
export const visualReplayTaggedDirectivePattern = /\[\[控制:回放示範\s+tag=([a-zA-Z0-9_.:-]+)\]\]/;
export const learningReplayStepDelayMs = 950;
export const learningSensitiveTextPattern = /(password|passwd|token|api[_ -]?key|secret|bearer|sk-[a-z0-9_-]{16,}|xox[baprs]-|gh[pousr]_[a-z0-9_]{20,})/i;

export function compactLearningText(value, fallback = '') {
  const text = String(value || '').replace(/\s+/g, ' ').trim();
  return text ? text.slice(0, 120) : fallback;
}

export function normalizeLearningKey(value) {
  const key = String(value || '').trim();
  if (!key) return '';
  if (key === ' ') return 'Space';
  if (/^esc$/i.test(key)) return 'Escape';
  if (/^return$/i.test(key)) return 'Enter';
  if (/^arrow(left|right|up|down)$/i.test(key)) return key.charAt(0).toUpperCase() + key.slice(1);
  if (key.length === 1) return key.toUpperCase();
  return key.charAt(0).toUpperCase() + key.slice(1);
}

function cssEscapeIdent(value) {
  if (typeof CSS !== 'undefined' && CSS.escape) return CSS.escape(value);
  return String(value || '').replace(/[^a-zA-Z0-9_-]/g, '\\$&');
}

export function buildLearningSelector(element) {
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
    if (node.classList?.length) part += `.${Array.from(node.classList).slice(0, 2).map(cssEscapeIdent).join('.')}`;
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

export function describeLearningTarget(rawTarget) {
  const element = rawTarget?.nodeType === 1 ? rawTarget : rawTarget?.parentElement;
  const interactive = element?.closest?.(visualLearningInteractiveSelector) || element;
  const rect = interactive?.getBoundingClientRect?.();
  const tag = interactive?.tagName?.toLowerCase?.() || '';
  const role = interactive?.getAttribute?.('role') || (tag === 'a' ? 'link' : tag);
  const label = compactLearningText(
    interactive?.getAttribute?.('aria-label') || interactive?.getAttribute?.('title') ||
    interactive?.innerText || interactive?.value || interactive?.placeholder || tag,
    tag || 'element',
  );
  return {
    element: interactive, label, role, tag, selector: buildLearningSelector(interactive),
    rect: rect ? {
      x: Number(rect.x.toFixed(2)), y: Number(rect.y.toFixed(2)),
      width: Number(rect.width.toFixed(2)), height: Number(rect.height.toFixed(2)),
    } : null,
  };
}

export function clampReplayCoordinate(value, min, max) {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, value));
}

export function clampLearningBBox(box, width, height) {
  const w = clampReplayCoordinate(Math.max(1, Math.round(Number(box.w || 1))), 1, width);
  const h = clampReplayCoordinate(Math.max(1, Math.round(Number(box.h || 1))), 1, height);
  const x = clampReplayCoordinate(Math.round(Number(box.x || 0)), 0, Math.max(0, width - w));
  const y = clampReplayCoordinate(Math.round(Number(box.y || 0)), 0, Math.max(0, height - h));
  return {x, y, w, h};
}

export function buildLearningWindowsAnchor(clickX, clickY, rect, viewport) {
  const width = Math.max(1, Math.round(Number(viewport?.width || window.innerWidth || 1)));
  const height = Math.max(1, Math.round(Number(viewport?.height || window.innerHeight || 1)));
  const click = {
    x: clampReplayCoordinate(Math.round(Number(clickX || 0)), 0, width - 1),
    y: clampReplayCoordinate(Math.round(Number(clickY || 0)), 0, height - 1),
  };
  const hasRect = rect && Number(rect.width) > 0 && Number(rect.height) > 0;
  const box = clampLearningBBox(hasRect ? {
    x: Math.round(Number(rect.x || 0)), y: Math.round(Number(rect.y || 0)),
    w: Math.round(Number(rect.width || 0)), h: Math.round(Number(rect.height || 0)),
  } : {x: click.x - 14, y: click.y - 14, w: 28, h: 28}, width, height);
  return {
    platform: 'windows', ok: true,
    mode: hasRect ? 'dom_rect_anchor' : 'manual_click_box',
    reason: hasRect
      ? 'in-app DOM element rect captured during learning; YOLO/OpenCV screenshot resolver was not required'
      : 'no element rectangle was available during learning; preserving the click as a small manual anchor',
    click,
    execution_point: hasRect ? {x: Math.round(box.x + box.w / 2), y: Math.round(box.y + box.h / 2)} : click,
    execution_hint: hasRect ? 'click_bbox_center' : 'fast_click_original_point',
    anchor_bbox: box, crop_bbox: box, ocr_status: 'not_used',
    ocr_note: hasRect
      ? 'OCR is optional and not used for this recorded DOM anchor.'
      : 'OCR is optional and not used for this recorded manual anchor.',
    detector_backend: 'dom', detector_degraded: !hasRect, needs_review: !hasRect,
  };
}

export function isLearningReplayRequest(text) {
  const normalized = String(text || '').trim().toLowerCase();
  return /按照.*剛剛.*示範|照.*剛剛.*示範|回放.*剛剛.*(示範|步驟|操作)|重播.*剛剛.*(示範|步驟|操作)|再.*(示範|執行|跑).*剛剛|剛剛.*步驟/.test(normalized)
    || /replay.*(last.*demo|previous.*demo|demo|steps)|follow.*demo/.test(normalized);
}

export function normalizeLearningOperationQuery(text) {
  const raw = String(text || '').trim();
  if (!raw) return '';
  return raw.replace(/操作|操做|operation/gi, ' ')
    .replace(/相關|關於|有關|已保存|已儲存|保存|儲存|錄影紀錄|錄製紀錄|示範紀錄|示範|流程|畫面|tag/gi, ' ')
    .replace(/幫我|請|查詢|搜尋|查找|尋找|找|列出|查看|看看|知道|執行|回放|重播|開啟|打開|有哪些|什麼|甚麼|樣|的/g, ' ')
    .replace(/[，。！？、,.!?;:()[\]{}"'`<>|\\/]/g, ' ').replace(/\s+/g, ' ').trim();
}

export function parseLearningOperationCatalogRequest(text) {
  const raw = String(text || '').trim();
  if (!raw) return null;
  const mentionsSavedCatalog = /已保存|已儲存|保存|儲存|錄影|錄製|示範|紀錄|記錄|catalog/i.test(raw);
  const asksSavedTag = /tag/i.test(raw) && /儲存|保存|已保存|已儲存|畫面|錄影|錄製|示範|紀錄|記錄|操作|操做/.test(raw);
  const asksCatalogList = /有哪些|列出|清單|查看|看看|知道/.test(raw)
    && (mentionsSavedCatalog || /tag/i.test(raw)) && /操作|操做|錄影|錄製|示範|tag|catalog/i.test(raw);
  if (!asksSavedTag && !asksCatalogList) return null;
  const query = normalizeLearningOperationQuery(raw);
  return query ? {mode: 'search', query} : {mode: 'list', query: ''};
}

export function operationIntentFromCLIResponse(resp) {
  const action = String(resp?.action || '').trim().toLowerCase();
  const target = String(resp?.target || '').trim();
  const next = String(resp?.next || '').trim().toLowerCase();
  if (!target) return null;
  if (['查詢', '搜尋', 'search', 'query'].includes(action) && ['操作', '操做'].includes(next)) {
    return {mode: 'query', query: normalizeLearningOperationQuery(target) || target};
  }
  if (action === '操作' || action === '操做') {
    return {mode: 'execute', query: normalizeLearningOperationQuery(target) || target, raw: target};
  }
  return null;
}

export function isLastLearningOperationReference(intent) {
  const text = `${intent?.raw || ''} ${intent?.query || ''}`.toLowerCase();
  return Boolean(text.trim()) && (/上次|上一個|上一筆|剛剛|剛才|再一次|再執行一次|重跑一次|重新.*一次/.test(text)
    || /last|previous|again|one more time|rerun/.test(text));
}

export function resolveLearningOperationMatch(matches = []) {
  if (!matches.length) return null;
  const firstScore = Number(matches[0]?.score || 0);
  const secondScore = Number(matches[1]?.score || 0);
  if (firstScore < 1.5 || (matches[1] && secondScore > 0 && firstScore - secondScore < 0.75)) return null;
  return matches[0];
}

export function isLearningReplayRelatedText(text) {
  const normalized = String(text || '').trim().toLowerCase();
  return Boolean(normalized) && (/示範|回放|重播|剛剛.*步驟|剛剛.*操作|照.*剛剛|按照.*剛剛/.test(normalized)
    || /replay|demo|last\s+steps|previous\s+steps/.test(normalized));
}

export function extractVisualReplayDirective(text) {
  const raw = String(text || '');
  const taggedMatch = raw.match(visualReplayTaggedDirectivePattern);
  if (taggedMatch) return {shouldReplay: true, tag: taggedMatch[1], text: raw.replace(visualReplayTaggedDirectivePattern, '').trim()};
  const directive = raw.includes(visualReplayLastDemoDirective) ? visualReplayLastDemoDirective
    : raw.includes(legacyVisualReplayLastDemoDirective) ? legacyVisualReplayLastDemoDirective : '';
  return directive
    ? {shouldReplay: true, tag: '', text: raw.split(directive).join('').trim()}
    : {shouldReplay: false, tag: '', text: raw};
}
