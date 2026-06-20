// groupsBar.js — 重點群組「拖出成文檔」UI
//
// body-mount 一條浮動群組列（不動 App.jsx 版面）。列出目前對話所有非空群組，
// 每個群組一顆可拖曳的 pill；拖出 → 後端 ExportGroup 產 Markdown → 原生拖曳到 Finder。
//
// 依賴注入 bindings()，後端未就緒時整條列不顯示。

import { COLOR_SLOTS } from './highlightCore';

export function createGroupsBar({ conversationId, getBindings, exportGroup }) {
  let bar = null;
  const pathCache = new Map(); // groupId -> exported file path

  function ensureBar() {
    if (bar) return bar;
    bar = document.createElement('div');
    bar.className = 'hl-groups-bar';
    bar.setAttribute('aria-label', '重點群組，可拖出成文檔');
    document.body.appendChild(bar);
    return bar;
  }

  function groupsFrom(list) {
    const map = new Map(); // groupId -> { slot, count }
    (list || []).forEach((h) => {
      if (h.system) return;
      const g = map.get(h.groupId) || { slot: h.colorSlot, count: 0 };
      g.count += 1;
      map.set(h.groupId, g);
    });
    return map;
  }

  async function refresh(list) {
    const api = await getBindings();
    if (!api.ExportGroup) { destroyDom(); return; } // 後端未就緒：不顯示
    const groups = groupsFrom(list);
    if (groups.size === 0) { destroyDom(); return; }

    const el = ensureBar();
    el.replaceChildren();
    const label = document.createElement('span');
    label.className = 'hl-groups-label';
    label.textContent = '重點群組（拖出存檔）';
    el.appendChild(label);

    for (const [groupId, { slot, count }] of groups) {
      const c = COLOR_SLOTS[slot] || COLOR_SLOTS[0];
      const pill = document.createElement('div');
      pill.className = 'hl-group-pill';
      pill.draggable = true;
      pill.style.background = c.bg;
      pill.style.color = c.fg;
      pill.style.boxShadow = `0 0 0 1px ${c.border}`;
      pill.textContent = `群組 ${slot + 1} · ${count}`;
      pill.title = '拖出成文檔';

      // 先匯出備好檔，拖曳時才有 path 可丟給原生拖曳
      pill.addEventListener('mouseenter', async () => {
        if (pathCache.has(groupId)) return;
        try {
          const p = await exportGroup(groupId);
          if (p) pathCache.set(groupId, p);
        } catch (_) {}
      });

      pill.addEventListener('dragstart', async (e) => {
        e.dataTransfer?.setData('text/plain', `重點群組 ${slot + 1}`);
        let p = pathCache.get(groupId);
        if (!p) {
          try { p = await exportGroup(groupId); if (p) pathCache.set(groupId, p); } catch (_) {}
        }
        if (p && api.NativeDragExportReferenceFile) {
          try { api.NativeDragExportReferenceFile(p); } catch (_) {}
        }
      });

      el.appendChild(pill);
    }
  }

  function destroyDom() {
    if (bar) { bar.remove(); bar = null; }
  }
  function destroy() {
    destroyDom();
    pathCache.clear();
  }

  return { refresh, destroy };
}

export default createGroupsBar;
