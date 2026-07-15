// useSystemMarks.js — 系統側暗標（輔助系統）的前端接線
//
// 兩件事：
//  1) 訊息定稿時呼叫 EnqueueSystemMark（便宜，只 redact + 入佇列，後端做）。
//     最後一則可能還在串流，用 debounce 補；其餘視為已定稿立即 enqueue。
//  2) idle 觸發 OrganizeSystemMarks（重，模型抽取）。盡量做電量 gating；
//     WKWebView 無 navigator.getBattery 時交給後端 scheduler 做真正 gating。
//
// 不影響 §22 使用者標註；後端未綁定時自動 no-op。

import { useEffect, useRef } from 'react';
import { messageDomId } from './messageId';

async function bindings() {
  try {
    return await import('../../../wailsjs/go/main/App');
  } catch (_) {
    return {};
  }
}

const FINALIZE_DEBOUNCE_MS = 3000;  // 最後一則訊息靜止幾秒視為定稿
const IDLE_ORGANIZE_MS = 300000;    // 無操作 5 分鐘 → 觸發整理（後端另有佇列門檻，攢一批才真的跑）
const BATTERY_THRESHOLD = 0.4;      // 電量 > 門檻才跑重整理

function sourceOf(msg) {
  return /^Ai:/.test(String(msg || '')) ? 'llm_output' : 'user_text';
}
function cleanText(msg) {
  return String(msg || '').replace(/^Ai:/, '').trim();
}

async function batteryAllows() {
  try {
    if (navigator.getBattery) {
      const b = await navigator.getBattery();
      return b.charging || b.level > BATTERY_THRESHOLD;
    }
  } catch (_) {}
  return true; // 無 API（WKWebView）→ 後端 scheduler 負責真正 gating
}

export function useSystemMarks({ messages = [], conversationId = 'main', enabled = true } = {}) {
  const enqueuedRef = useRef(new Set());
  const lastTimerRef = useRef(null);

  // 切換對話：重置已 enqueue 紀錄
  useEffect(() => {
    enqueuedRef.current = new Set();
  }, [conversationId]);

  // 訊息定稿 → enqueue
  useEffect(() => {
    if (!enabled) return undefined;
    let cancelled = false;
    (async () => {
      const api = await bindings();
      if (cancelled || !api.EnqueueSystemMark) return;
      const enqueue = (idx) => {
        const key = messageDomId(messages[idx], idx, messages);
        if (enqueuedRef.current.has(key)) return;
        const text = cleanText(messages[idx]);
        if (!text) return;
        enqueuedRef.current.add(key);
        try {
          api.EnqueueSystemMark(
            conversationId,
            JSON.stringify({ messageId: key, text, source: sourceOf(messages[idx]) }),
          );
        } catch (_) {}
      };
      for (let i = 0; i < messages.length - 1; i++) enqueue(i); // 除最後一則外視為定稿
      if (lastTimerRef.current) clearTimeout(lastTimerRef.current);
      const last = messages.length - 1;
      if (last >= 0) lastTimerRef.current = setTimeout(() => enqueue(last), FINALIZE_DEBOUNCE_MS);
    })();
    return () => { cancelled = true; };
  }, [messages, conversationId, enabled]);

  // idle → 整理（讓位使用者：任何操作重置計時）
  useEffect(() => {
    if (!enabled) return undefined;
    let timer = null;
    const schedule = () => {
      if (timer) clearTimeout(timer);
      timer = setTimeout(async () => {
        if (!document.hidden && (await batteryAllows())) {
          const api = await bindings();
          try { await api.OrganizeSystemMarks?.(conversationId); } catch (_) {}
        }
        schedule();
      }, IDLE_ORGANIZE_MS);
    };
    const reset = () => schedule();
    schedule();
    window.addEventListener('mousemove', reset, { passive: true });
    window.addEventListener('keydown', reset);
    window.addEventListener('scroll', reset, { passive: true });
    return () => {
      if (timer) clearTimeout(timer);
      window.removeEventListener('mousemove', reset);
      window.removeEventListener('keydown', reset);
      window.removeEventListener('scroll', reset);
    };
  }, [conversationId, enabled]);
}

export default useSystemMarks;
