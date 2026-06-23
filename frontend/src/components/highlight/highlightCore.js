// highlightCore.js — 路線 B：零依賴文字高亮核心（v3 互斥版）
//
// 設計原則：
//  - 顏色只是「同一群」的視覺記號，不帶語意。
//  - 一個詞只屬一個群組：上色前先清掉選取範圍內所有既有標記（互斥），
//    再重新包一層，杜絕巢狀殘留。改群＝先取消舊標、再上新色。
//  - 反白 → 送本地模型評分 → 預選最高分群組色 → 點色採用 / 連點換色 → 移開 commit。
//  - 前端只算 {messageId, startOffset, endOffset, quote, groupId, colorSlot}，
//    語意 / 摘要 / 權重全在 Go 後台。

export const COLOR_SLOTS = [
  { slot: 0, name: 'beige',     bg: '#F5E6C5', fg: '#1A1A1A', border: 'rgba(0,0,0,.18)' }, // 米黃 / 黑
  { slot: 1, name: 'lightblue', bg: '#BBD9F2', fg: '#1A1A1A', border: 'rgba(0,0,0,.18)' }, // 淺藍 / 黑
  { slot: 2, name: 'navy',      bg: '#214A7C', fg: '#FFFFFF', border: 'rgba(0,0,0,.18)' }, // 深藍 / 白
  { slot: 3, name: 'black',     bg: '#1C1C1E', fg: '#FFFFFF', border: 'rgba(0,0,0,.18)' }, // 深黑 / 白
  { slot: 4, name: 'lightred',  bg: '#F2C2C2', fg: '#1A1A1A', border: 'rgba(0,0,0,.18)' }, // 淺紅 / 黑
  { slot: 5, name: 'whitered',  bg: '#FFFFFF', fg: '#C0392B', border: '#C0392B' },         // 白底 / 紅字（紅框）
  { slot: 6, name: 'lime',      bg: '#B6F36B', fg: '#1A1A1A', border: 'rgba(0,0,0,.18)' }, // 嫩綠 / 黑
  { slot: 7, name: 'olive',     bg: '#33402E', fg: '#C7E86B', border: 'rgba(0,0,0,.18)' }, // 橄欖綠 / 黃綠字
];
export const MAX_GROUPS = COLOR_SLOTS.length; // 8

export const HL_ATTR = 'data-hl-id';
export const HL_GROUP_ATTR = 'data-hl-group';
export const HL_SLOT_ATTR = 'data-hl-slot';

// ---- offset 工具 ----
function textNodesUnder(root) {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null);
  const nodes = [];
  let n;
  while ((n = walker.nextNode())) nodes.push(n);
  return nodes;
}
function globalOffset(root, node, nodeOffset) {
  const nodes = textNodesUnder(root);
  let acc = 0;
  for (const tn of nodes) {
    if (tn === node) return acc + nodeOffset;
    acc += tn.textContent.length;
  }
  return acc;
}
function locateOffset(root, target) {
  const nodes = textNodesUnder(root);
  let acc = 0;
  for (const tn of nodes) {
    const len = tn.textContent.length;
    if (target <= acc + len) return { node: tn, offset: target - acc };
    acc += len;
  }
  const last = nodes[nodes.length - 1];
  return last ? { node: last, offset: last.textContent.length } : null;
}
function findMessageRoot(node, rootSelector) {
  let el = node.nodeType === Node.TEXT_NODE ? node.parentElement : node;
  while (el && el !== document.body) {
    if (el.matches?.(rootSelector)) return el;
    el = el.parentElement;
  }
  return null;
}

// ---- span 上色 / 換色 / 拆除 ----
function applySpanStyle(span, slot) {
  span.className = `hl hl-slot-${slot}`;
  span.setAttribute(HL_SLOT_ATTR, String(slot));
}
function wrapRange(range, { hlId, groupId, slot }) {
  const spans = [];
  const make = () => {
    const s = document.createElement('span');
    applySpanStyle(s, slot);
    s.setAttribute(HL_ATTR, hlId);
    s.setAttribute(HL_GROUP_ATTR, groupId);
    return s;
  };
  const root = range.commonAncestorContainer;
  const walker = document.createTreeWalker(
    root.nodeType === Node.TEXT_NODE ? root.parentNode : root,
    NodeFilter.SHOW_TEXT,
    null,
  );
  const targets = [];
  let n;
  while ((n = walker.nextNode())) {
    if (range.intersectsNode(n)) targets.push(n);
  }
  if (targets.length === 0 && root.nodeType === Node.TEXT_NODE) targets.push(root);
  targets.forEach((tn) => {
    const start = tn === range.startContainer ? range.startOffset : 0;
    const end = tn === range.endContainer ? range.endOffset : tn.textContent.length;
    if (end <= start) return;
    const sub = document.createRange();
    sub.setStart(tn, start);
    sub.setEnd(tn, end);
    const span = make();
    try {
      sub.surroundContents(span);
      spans.push(span);
    } catch (_) {
      const frag = sub.extractContents();
      span.appendChild(frag);
      sub.insertNode(span);
      spans.push(span);
    }
  });
  return spans;
}
function recolorHighlight(hlId, slot, container = document) {
  container.querySelectorAll(`[${HL_ATTR}="${cssEsc(hlId)}"]`).forEach((span) => {
    applySpanStyle(span, slot);
  });
}
export function removeHighlightDomById(hlId, container = document) {
  const spans = container.querySelectorAll(`[${HL_ATTR}="${cssEsc(hlId)}"]`);
  spans.forEach((span) => {
    const parent = span.parentNode;
    if (!parent) return;
    while (span.firstChild) parent.insertBefore(span.firstChild, span);
    parent.removeChild(span);
    parent.normalize?.();
  });
  return spans.length > 0;
}
// 跳回原文：捲動到該標記並閃一下（重點清單 ↔ 對話 互相對照）
export function flashHighlightDomById(hlId, container = document) {
  const spans = container.querySelectorAll(`[${HL_ATTR}="${cssEsc(hlId)}"]`);
  if (spans.length === 0) return false;
  spans[0].scrollIntoView({ behavior: 'smooth', block: 'center' });
  spans.forEach((span) => {
    span.classList.remove('hl-flash');
    void span.offsetWidth; // 強制 reflow，讓動畫可重複觸發
    span.classList.add('hl-flash');
    setTimeout(() => span.classList.remove('hl-flash'), 1200);
  });
  return true;
}
// 互斥核心：找出與 range 相交的所有既有標記 id
function highlightIdsInRange(range, container) {
  const ids = new Set();
  container.querySelectorAll(`[${HL_ATTR}]`).forEach((span) => {
    try {
      if (range.intersectsNode(span)) ids.add(span.getAttribute(HL_ATTR));
    } catch (_) {}
  });
  return [...ids];
}
function cssEsc(s) {
  return String(s).replace(/["\\]/g, '\\$&');
}
let _seq = 0;
function newId(prefix) {
  _seq += 1;
  return `${prefix}_${Date.now().toString(36)}_${_seq}`;
}

// ---- createHighlightLayer ----
export function createHighlightLayer(opts) {
  const {
    rootSelector = '.message-text',
    messageIdOf = (el) => el.getAttribute('data-message-id') || '',
    onCommit = () => {},
    onRemove = () => {},
    scoreFor = null,
    lastSlotRef = { current: null },
    allocateGroupForSlot = (slot) => `grp_slot_${slot}`,
    container = document,
  } = opts || {};

  let toolbar = null;
  let swatchEls = [];
  let pendingRange = null;
  let pendingRoot = null;
  let draft = null; // { hlId, groupId, slot, messageId, startOffset, endOffset, quote }

  function buildToolbar() {
    const bar = document.createElement('div');
    bar.className = 'hl-toolbar';
    bar.setAttribute('role', 'toolbar');
    swatchEls = COLOR_SLOTS.map(({ slot, bg, fg, border }) => {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'hl-swatch';
      btn.setAttribute(HL_SLOT_ATTR, String(slot));
      btn.style.background = bg;
      btn.style.color = fg;
      btn.style.boxShadow = `0 0 0 1px ${border}`;
      btn.title = `標記到群組 ${slot + 1}`;
      btn.textContent = 'A';
      btn.addEventListener('mousedown', (e) => {
        e.preventDefault();
        chooseSlot(slot);
      });
      bar.appendChild(btn);
      return btn;
    });
    const rm = document.createElement('button');
    rm.type = 'button';
    rm.className = 'hl-remove';
    rm.title = '移除標記';
    rm.setAttribute('aria-label', '移除標記');
    rm.textContent = '✕';
    rm.addEventListener('mousedown', (e) => {
      e.preventDefault();
      removeCurrent();
    });
    bar.appendChild(rm);
    document.body.appendChild(bar);
    return bar;
  }

  function markPreselect(slot) {
    swatchEls.forEach((el) => {
      const s = Number(el.getAttribute(HL_SLOT_ATTR));
      el.classList.toggle('hl-swatch-suggested', s === slot);
    });
  }
  function showToolbar(rect) {
    if (!toolbar) toolbar = buildToolbar();
    toolbar.style.display = 'flex';
    const top = window.scrollY + rect.top - toolbar.offsetHeight - 8;
    const left = window.scrollX + rect.left + rect.width / 2 - toolbar.offsetWidth / 2;
    toolbar.style.top = `${Math.max(8, top)}px`;
    toolbar.style.left = `${Math.max(8, left)}px`;
  }
  function hideToolbar() {
    if (toolbar) toolbar.style.display = 'none';
    swatchEls.forEach((el) => el.classList.remove('hl-swatch-suggested'));
  }

  async function onMouseUp() {
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed || sel.rangeCount === 0) return commitDraft();
    const range = sel.getRangeAt(0);
    const root = findMessageRoot(range.startContainer, rootSelector);
    if (!root || root !== findMessageRoot(range.endContainer, rootSelector)) return commitDraft();
    const quote = range.toString().trim();
    if (!quote) return commitDraft();

    commitDraft(); // 移到別處先 commit 上一個
    pendingRange = range.cloneRange();
    pendingRoot = root;
    showToolbar(range.getBoundingClientRect());

    // 預選：先沿用上一次顏色保底，評分回來再覆蓋
    let preslot = lastSlotRef.current != null ? lastSlotRef.current : 0;
    markPreselect(preslot);
    if (scoreFor) {
      try {
        const ranked = await scoreFor(quote);
        if (Array.isArray(ranked) && ranked.length > 0 && pendingRange) {
          markPreselect(ranked[0].colorSlot);
        }
      } catch (_) {}
    }
  }

  // 點色塊：draft 已存在＝換色（in-place，無殘留）；第一次＝互斥清除後建立
  function chooseSlot(slot) {
    if (draft) {
      draft.slot = slot;
      recolorHighlight(draft.hlId, slot, container);
      lastSlotRef.current = slot;
      markPreselect(slot);
      return;
    }
    if (!pendingRange || !pendingRoot) return;

    const messageId = messageIdOf(pendingRoot);
    const startOffset = globalOffset(pendingRoot, pendingRange.startContainer, pendingRange.startOffset);
    const endOffset = globalOffset(pendingRoot, pendingRange.endContainer, pendingRange.endOffset);

    // ★ 互斥：先清掉選取範圍內所有既有標記（一個詞只屬一組）
    const removed = highlightIdsInRange(pendingRange, container);
    removed.forEach((id) => removeHighlightDomById(id, container));
    removed.forEach((id) => { try { onRemove(id); } catch (_) {} });

    // DOM 已變，用 offset 重建乾淨的 range，再包一層（不會巢狀）
    const s = locateOffset(pendingRoot, startOffset);
    const e = locateOffset(pendingRoot, endOffset);
    if (!s || !e) { resetPending(); return; }
    const r = document.createRange();
    try {
      r.setStart(s.node, s.offset);
      r.setEnd(e.node, e.offset);
    } catch (_) { resetPending(); return; }
    const quote = r.toString();
    const groupId = allocateGroupForSlot(slot);
    const hlId = newId('hl');
    wrapRange(r, { hlId, groupId, slot });

    draft = { hlId, groupId, slot, messageId, startOffset, endOffset, quote };
    lastSlotRef.current = slot;
    markPreselect(slot);
    resetPending();
  }

  function resetPending() {
    window.getSelection()?.removeAllRanges();
    pendingRange = null;
  }

  function commitDraft() {
    hideToolbar();
    pendingRange = null;
    pendingRoot = null;
    if (!draft) return;
    const annotation = {
      id: draft.hlId,
      messageId: draft.messageId,
      groupId: draft.groupId,
      colorSlot: draft.slot,
      startOffset: draft.startOffset,
      endOffset: draft.endOffset,
      quote: draft.quote,
      createdAt: new Date().toISOString(),
    };
    draft = null;
    try { onCommit(annotation); } catch (_) {}
  }

  // ✕ 移除：有 draft 移 draft；否則移選取範圍內的既有標記
  function removeCurrent() {
    if (draft) {
      const id = draft.hlId;
      removeHighlightDomById(id, container);
      draft = null;
      hideToolbar();
      try { onRemove(id); } catch (_) {}
      return;
    }
    if (pendingRange) {
      const ids = highlightIdsInRange(pendingRange, container);
      ids.forEach((id) => removeHighlightDomById(id, container));
      hideToolbar();
      ids.forEach((id) => { try { onRemove(id); } catch (_) {} });
      pendingRange = null;
      return;
    }
    hideToolbar();
  }

  function restore(list) {
    (list || []).forEach((a) => {
      const root = [...container.querySelectorAll(rootSelector)]
        .find((el) => messageIdOf(el) === a.messageId);
      if (!root) return;
      const start = locateOffset(root, a.startOffset);
      const end = locateOffset(root, a.endOffset);
      if (!start || !end) return;
      const range = document.createRange();
      try {
        range.setStart(start.node, start.offset);
        range.setEnd(end.node, end.offset);
        wrapRange(range, { hlId: a.id, groupId: a.groupId, slot: a.colorSlot });
      } catch (_) {}
    });
  }

  function onDocPointer(e) {
    if (toolbar && toolbar.contains(e.target)) return;
    if (window.getSelection()?.isCollapsed) commitDraft();
  }

  function attach() {
    container.addEventListener('mouseup', onMouseUp);
    document.addEventListener('mousedown', onDocPointer);
  }
  function detach() {
    commitDraft();
    container.removeEventListener('mouseup', onMouseUp);
    document.removeEventListener('mousedown', onDocPointer);
    if (toolbar) { toolbar.remove(); toolbar = null; swatchEls = []; }
  }

  return { attach, detach, restore, commitDraft, removeCurrent, COLOR_SLOTS, MAX_GROUPS };
}
