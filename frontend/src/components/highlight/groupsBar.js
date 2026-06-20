// groupsBar.js — 重點字詞入口與「拖出彈窗移除」UI
//
// 掛在左側「專案管理」下方。列出目前對話非空群組；點開後在獨立彈窗
// 顯示所有已標記字詞。字詞從彈窗拖出去即移除該 highlight。

import { COLOR_SLOTS, removeHighlightDomById, flashHighlightDomById } from './highlightCore';

const LEFT_PANEL_ANCHOR = '[data-highlight-groups-anchor]';

export function createGroupsBar({ onRemove }) {
  let bar = null;
  let modal = null;
  let currentList = [];
  let activeGroupId = null;
  let draggingId = null;
  let lastX = 0;
  let lastY = 0;

  // 後端清單 → 安全的前端視圖：濾掉系統暗標 / 缺欄位，標準化 slot 與 quote
  function normalizeList(list) {
    return (Array.isArray(list) ? list : [])
      .filter((h) => h && !h.system && h.id && h.groupId)
      .map((h) => ({
        ...h,
        colorSlot: Number.isFinite(Number(h.colorSlot)) ? Number(h.colorSlot) : 0,
        quote: String(h.quote || '').trim(),
      }))
      .filter((h) => h.quote);
  }

  function groupsFrom(list) {
    const map = new Map(); // groupId -> { slot, items }
    list.forEach((h) => {
      const g = map.get(h.groupId) || { slot: h.colorSlot, items: [] };
      g.items.push(h);
      map.set(h.groupId, g);
    });
    return map;
  }

  function mountBar(el) {
    const anchor = document.querySelector(LEFT_PANEL_ANCHOR);
    if (anchor) {
      anchor.replaceChildren(el);
      el.classList.remove('hl-groups-bar-floating');
      return;
    }
    document.body.appendChild(el);
    el.classList.add('hl-groups-bar-floating');
  }

  function ensureBar() {
    const anchor = document.querySelector(LEFT_PANEL_ANCHOR);
    if (!bar) {
      bar = document.createElement('section');
      bar.className = 'hl-groups-bar';
      bar.setAttribute('aria-label', '重點字詞');
    }
    // anchor 被 React 重建、或 bar 脫離畫面 → 重新掛回去（避免入口憑空消失）
    const detached = !bar.isConnected || (anchor && bar.parentNode !== anchor);
    if (detached) mountBar(bar);
    return bar;
  }

  function ensureActiveGroup(groups) {
    if (activeGroupId && groups.has(activeGroupId)) return;
    activeGroupId = groups.keys().next().value || null;
  }

  // 重畫左欄入口：群組為空就收掉，否則畫「重點字詞 N」+ 各群組 pill
  function renderBar() {
    const groups = groupsFrom(currentList);
    if (groups.size === 0) {
      destroyDom();
      return;
    }

    ensureActiveGroup(groups);
    const el = ensureBar();
    el.replaceChildren();

    const trigger = document.createElement('button');
    trigger.type = 'button';
    trigger.className = 'hl-groups-trigger';
    trigger.title = '查看已標記字詞';
    trigger.addEventListener('click', () => openModal());

    const label = document.createElement('span');
    label.className = 'hl-groups-label';
    label.textContent = '重點字詞';
    trigger.appendChild(label);

    const total = document.createElement('span');
    total.className = 'hl-groups-total';
    total.textContent = String(currentList.length);
    trigger.appendChild(total);
    el.appendChild(trigger);

    const pills = document.createElement('div');
    pills.className = 'hl-groups-pills';
    for (const [groupId, { slot, items }] of groups) {
      const c = COLOR_SLOTS[slot] || COLOR_SLOTS[0];
      const pill = document.createElement('button');
      pill.type = 'button';
      pill.className = 'hl-group-pill';
      pill.style.background = c.bg;
      pill.style.color = c.fg;
      pill.style.boxShadow = `0 0 0 1px ${c.border}`;
      pill.textContent = `群組 ${slot + 1}・${items.length}`;
      pill.title = `查看群組 ${slot + 1} 的字詞`;
      pill.addEventListener('click', () => {
        activeGroupId = groupId;
        openModal();
      });
      pills.appendChild(pill);
    }
    el.appendChild(pills);
  }

  function groupTitle(groupId, group) {
    const slot = group?.slot ?? 0;
    return `群組 ${slot + 1}・${group?.items?.length || 0}`;
  }

  // 開彈窗（已開則只重畫）：點外關閉、Esc 關閉
  function openModal() {
    if (modal) {
      renderModal();
      return;
    }
    modal = document.createElement('div');
    modal.className = 'hl-words-overlay';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-label', '已標記字詞');
    modal.addEventListener('mousedown', (event) => {
      if (event.target === modal) closeModal();
    });
    document.addEventListener('keydown', onKeyDown);
    document.body.appendChild(modal);
    renderModal();
  }

  function closeModal() {
    if (!modal) return;
    modal.remove();
    modal = null;
    draggingId = null;
    document.removeEventListener('keydown', onKeyDown);
  }

  function onKeyDown(event) {
    if (event.key === 'Escape') closeModal();
  }

  // 重畫彈窗：群組分頁 tabs + 當前群組的字詞 chip 清單
  function renderModal() {
    if (!modal) return;
    const groups = groupsFrom(currentList);
    if (groups.size === 0) {
      closeModal();
      destroyDom();
      return;
    }
    ensureActiveGroup(groups);
    const activeGroup = groups.get(activeGroupId) || groups.values().next().value;

    modal.replaceChildren();
    const panel = document.createElement('section');
    panel.className = 'hl-words-popup';

    const header = document.createElement('header');
    header.className = 'hl-words-header';
    const titleWrap = document.createElement('div');
    const title = document.createElement('h3');
    title.textContent = '已標記字詞';
    const subtitle = document.createElement('small');
    subtitle.textContent = `${currentList.length} 個字詞`;
    titleWrap.append(title, subtitle);

    const close = document.createElement('button');
    close.type = 'button';
    close.className = 'hl-words-close';
    close.setAttribute('aria-label', '關閉');
    close.textContent = '×';
    close.addEventListener('click', closeModal);
    header.append(titleWrap, close);
    panel.appendChild(header);

    const tabs = document.createElement('div');
    tabs.className = 'hl-words-tabs';
    for (const [groupId, group] of groups) {
      const c = COLOR_SLOTS[group.slot] || COLOR_SLOTS[0];
      const tab = document.createElement('button');
      tab.type = 'button';
      tab.className = `hl-words-tab${groupId === activeGroupId ? ' active' : ''}`;
      tab.style.setProperty('--hl-tab-color', c.bg);
      tab.textContent = groupTitle(groupId, group);
      tab.addEventListener('click', () => {
        activeGroupId = groupId;
        renderModal();
      });
      tabs.appendChild(tab);
    }
    panel.appendChild(tabs);

    const list = document.createElement('div');
    list.className = 'hl-words-list';
    activeGroup.items.forEach((item) => {
      const c = COLOR_SLOTS[item.colorSlot] || COLOR_SLOTS[0];
      const chip = document.createElement('div');
      chip.className = 'hl-word-chip';
      chip.draggable = true;
      chip.style.background = c.bg;
      chip.style.color = c.fg;
      chip.style.boxShadow = `0 0 0 1px ${c.border}`;
      chip.title = '點擊跳到原文，拖出或按 × 移除';
      const chipText = document.createElement('span');
      chipText.className = 'hl-word-chip-text';
      chipText.textContent = item.quote;
      chip.appendChild(chipText);
      const chipDel = document.createElement('button');
      chipDel.type = 'button';
      chipDel.className = 'hl-word-chip-remove';
      chipDel.setAttribute('aria-label', `移除「${item.quote}」`);
      chipDel.textContent = '×';
      chipDel.draggable = false;
      chipDel.addEventListener('mousedown', (event) => event.stopPropagation());
      chipDel.addEventListener('click', (event) => {
        event.stopPropagation();
        removeWord(item.id);
      });
      chip.appendChild(chipDel);
      // 點 chip（非拖曳、非 ×）→ 關閉彈窗並跳回原文閃一下
      chip.addEventListener('click', () => {
        closeModal();
        flashHighlightDomById(item.id);
      });
      chip.addEventListener('dragstart', (event) => {
        draggingId = item.id;
        lastX = event.clientX;
        lastY = event.clientY;
        chip.classList.add('dragging');
        event.dataTransfer?.setData('text/plain', item.quote);
        event.dataTransfer?.setDragImage?.(chip, Math.min(24, chip.offsetWidth / 2), 14);
      });
      // drag 事件持續回報座標；dragend 常回 0,0，改用最後已知座標判斷
      chip.addEventListener('drag', (event) => {
        if (event.clientX || event.clientY) {
          lastX = event.clientX;
          lastY = event.clientY;
        }
      });
      chip.addEventListener('dragend', (event) => {
        chip.classList.remove('dragging');
        const panelRect = panel.getBoundingClientRect();
        const x = (event.clientX || event.clientY) ? event.clientX : lastX;
        const y = (event.clientX || event.clientY) ? event.clientY : lastY;
        const id = draggingId;
        draggingId = null;
        // 指標從未移出 popup（含 Esc 取消）→ 不刪；只有確實拖到 popup 外才刪
        if (id && (x || y) && !pointInRect(x, y, panelRect)) removeWord(id);
      });
      list.appendChild(chip);
    });
    panel.appendChild(list);

    // 底部操作提示：三種互動一眼可見
    const hint = document.createElement('p');
    hint.className = 'hl-words-hint';
    hint.textContent = '點字詞跳回原文 · 拖出彈窗或按 × 移除';
    panel.appendChild(hint);

    modal.appendChild(panel);
  }

  function pointInRect(x, y, rect) {
    return x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom;
  }

  // 移除字詞：樂觀更新（先拆 DOM、重畫），後端失敗則 rollback
  async function removeWord(id) {
    const removed = currentList.find((h) => h.id === id);
    if (!removed) return;
    currentList = currentList.filter((h) => h.id !== id);
    removeHighlightDomById(id);
    renderBar();
    renderModal();
    try {
      await onRemove?.(id);
    } catch (_) {
      currentList = [...currentList, removed];
      renderBar();
      renderModal();
    }
  }

  // 外部（useHighlight）餵入最新清單時重畫入口與彈窗
  async function refresh(list) {
    currentList = normalizeList(list);
    renderBar();
    if (modal) renderModal();
  }

  function destroyDom() {
    closeModal();
    if (bar) {
      bar.remove();
      bar = null;
    }
  }

  // 卸載：移除 DOM 與彈窗、清空狀態（切換對話 / unmount 時呼叫）
  function destroy() {
    destroyDom();
    currentList = [];
    activeGroupId = null;
  }

  return { refresh, destroy };
}

export default createGroupsBar;
