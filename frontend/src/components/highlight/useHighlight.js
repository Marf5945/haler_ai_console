// useHighlight.js — 把自動歸類高亮接到 React 對話畫面
//
// 整合（3 步）：
//  1) 訊息容器加 data-message-id（用穩定 content hash，見 messageId.js）。
//  2) App.jsx：import './components/highlight/highlight.css'; useHighlight({...});
//  3) Go：Bind 整個 app（Wails v2 自動曝露全部方法）。
//     後端未就緒時自動退化：只前端上色、不評分、不落地、不顯示群組列。
//
// 行為：反白 → 評分預選 → 點色採用 / 連點換色 → 移開 commit + debounce 摘要。
// 另外 body-mount 一條「重點群組」列，可把同色群組拖出成文檔。

import { useEffect, useRef, useCallback } from 'react';
import * as AppBindings from '../../../wailsjs/go/main/App';
import { createHighlightLayer, COLOR_SLOTS } from './highlightCore';
import { createGroupsBar } from './groupsBar';

async function bindings() {
  return AppBindings;
}

const SUMMARY_DEBOUNCE_MS = 4000;

export function useHighlight({
  enabled = true,
  rootSelector = '.message-text',
  conversationId = 'main',
} = {}) {
  const layerRef = useRef(null);
  const barRef = useRef(null);
  const lastSlotRef = useRef(null);
  const summaryTimersRef = useRef(new Map());

  const scheduleSummary = useCallback((groupId) => {
    const timers = summaryTimersRef.current;
    if (timers.has(groupId)) clearTimeout(timers.get(groupId));
    const t = setTimeout(async () => {
      timers.delete(groupId);
      const api = await bindings();
      try { await api.RebuildGroupSummary?.(conversationId, groupId); } catch (_) {}
    }, SUMMARY_DEBOUNCE_MS);
    timers.set(groupId, t);
  }, [conversationId]);

  const refreshBar = useCallback(async () => {
    const api = await bindings();
    if (!barRef.current) return;
    try {
      const raw = await api.ListHighlights?.(conversationId);
      const list = typeof raw === 'string' ? JSON.parse(raw || '[]') : (raw || []);
      barRef.current.refresh(list);
    } catch (_) {
      barRef.current.refresh([]);
    }
  }, [conversationId]);

  const deleteHighlight = useCallback(async (hlId) => {
    const api = await bindings();
    await api.DeleteHighlight?.(conversationId, hlId);
    COLOR_SLOTS.forEach((c) => scheduleSummary(`grp_${conversationId}_slot_${c.slot}`));
    refreshBar();
  }, [conversationId, scheduleSummary, refreshBar]);

  // 把建議色 vs 使用者實選回報後端（acceptance 指標）
  const recordOutcome = useCallback(async (suggestedSlot, chosenSlot) => {
    const api = await bindings();
    try { await api.RecordSuggestionOutcome?.(conversationId, suggestedSlot, chosenSlot); } catch (_) {}
  }, [conversationId]);

  useEffect(() => {
    if (!enabled) return undefined;
    let alive = true;
    const timers = summaryTimersRef.current;
    let lastSuggested = null;

    const bar = createGroupsBar({
      onRemove: deleteHighlight,
    });
    barRef.current = bar;

    const layer = createHighlightLayer({
      rootSelector,
      messageIdOf: (el) => el.getAttribute('data-message-id') || '',
      lastSlotRef,
      allocateGroupForSlot: (slot) => `grp_${conversationId}_slot_${slot}`,

      scoreFor: async (quote) => {
        const api = await bindings();
        if (!api.ScoreHighlightGroups) return [];
        try {
          const raw = await api.ScoreHighlightGroups(conversationId, quote);
          const ranked = typeof raw === 'string' ? JSON.parse(raw) : raw;
          lastSuggested = Array.isArray(ranked) && ranked.length > 0 ? ranked[0].colorSlot : null;
          return Array.isArray(ranked) ? ranked : [];
        } catch (_) {
          return [];
        }
      },

      onCommit: async (annotation) => {
        const api = await bindings();
        try {
          await api.SaveHighlight?.(JSON.stringify({ ...annotation, conversationId }));
          scheduleSummary(annotation.groupId);
          recordOutcome(lastSuggested != null ? lastSuggested : -1, annotation.colorSlot);
          lastSuggested = null;
          refreshBar();
        } catch (_) {}
      },

      onRemove: async (hlId) => {
        deleteHighlight(hlId).catch(() => {});
      },
    });
    layerRef.current = layer;
    layer.attach();

    (async () => {
      const api = await bindings();
      try {
        const raw = await api.ListHighlights?.(conversationId);
        if (!alive || !raw) return;
        const list = typeof raw === 'string' ? JSON.parse(raw) : raw;
        requestAnimationFrame(() => {
          layer.restore(list);
          bar.refresh(list);
        });
      } catch (_) {}
    })();

    return () => {
      alive = false;
      layer.detach();
      layerRef.current = null;
      bar.destroy();
      barRef.current = null;
      timers.forEach((t) => clearTimeout(t));
      timers.clear();
    };
  }, [enabled, rootSelector, conversationId, scheduleSummary, refreshBar, deleteHighlight, recordOutcome]);

  const exportGroup = useCallback(async (groupId) => {
    const api = await bindings();
    return api.ExportGroup?.(conversationId, groupId);
  }, [conversationId]);

  return { exportGroup, COLOR_SLOTS };
}

export default useHighlight;
