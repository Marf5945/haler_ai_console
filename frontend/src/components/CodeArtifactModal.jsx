// CodeArtifactModal.jsx — 資料區程式碼「展開」彈窗檢視器
//
// 規格（3.1.x）：
//  - 預設寬約半個螢幕，可拖曳移動、右下角可調整大小，內容區有上下左右滑桿。
//  - 顯示 tag／摘要／語言／編譯狀態；可一鍵複製、匯出到資料夾（選單）。
//  - 支援兩套標示，互不干擾：
//    1) 使用者 8 色標色：內容區掛 .message-text + data-message-id，
//       由既有 useHighlight 圈選上色管線接手。
//    2) LLM 4 種專用標示（meta.marks，offset 為 UTF-8 byte）：
//       粗體 / 斜體 / 粗斜體 / 半粗＋點狀底線，刻意不用底色，跟 8 色系統區隔。

import React, {useMemo, useRef, useState, useCallback, useEffect} from 'react';
import {createPortal} from 'react-dom';

const MARK_LEGEND = [
  {slot: 1, label: '核心重點', className: 'code-llm-mark-1'},
  {slot: 2, label: '本次修改', className: 'code-llm-mark-2'},
  {slot: 3, label: '風險注意', className: 'code-llm-mark-3'},
  {slot: 4, label: '可改選項', className: 'code-llm-mark-4'},
];

// Go 端 offset 是 UTF-8 byte；用 byte 切再 decode，中文註解才不會位移。
function segmentsFromMarks(content, marks) {
  const encoder = new TextEncoder();
  const decoder = new TextDecoder();
  const bytes = encoder.encode(String(content || ''));
  const sorted = [...(marks || [])]
    .filter((m) => m && m.slot >= 1 && m.slot <= 4 && m.endOffset > m.startOffset)
    .sort((a, b) => a.startOffset - b.startOffset);
  const segments = [];
  let cursor = 0;
  for (const mark of sorted) {
    const start = Math.max(cursor, Math.min(mark.startOffset, bytes.length));
    const end = Math.min(Math.max(mark.endOffset, start), bytes.length);
    if (end <= start) continue; // 重疊或越界 → 略過
    if (start > cursor) segments.push({text: decoder.decode(bytes.subarray(cursor, start)), slot: 0});
    segments.push({text: decoder.decode(bytes.subarray(start, end)), slot: mark.slot});
    cursor = end;
  }
  if (cursor < bytes.length) segments.push({text: decoder.decode(bytes.subarray(cursor)), slot: 0});
  if (!segments.length) segments.push({text: String(content || ''), slot: 0});
  return segments;
}

export default function CodeArtifactModal({detail, onClose, onExport, onCopyToast}) {
  const meta = detail?.meta || {};
  const content = detail?.content || '';
  const panelRef = useRef(null);
  const dragStateRef = useRef(null);
  const [copied, setCopied] = useState(false);
  // 預設約半個螢幕寬、置中偏上。
  const [pos, setPos] = useState(() => ({
    x: Math.round(window.innerWidth * 0.25),
    y: Math.round(window.innerHeight * 0.09),
  }));

  const segments = useMemo(() => segmentsFromMarks(content, meta.marks), [content, meta.marks]);
  const usedSlots = useMemo(() => new Set((meta.marks || []).map((m) => m.slot)), [meta.marks]);

  const handleHeaderPointerDown = useCallback((event) => {
    if (event.target.closest('button')) return;
    dragStateRef.current = {startX: event.clientX, startY: event.clientY, baseX: pos.x, baseY: pos.y};
    event.currentTarget.setPointerCapture?.(event.pointerId);
  }, [pos]);

  const handleHeaderPointerMove = useCallback((event) => {
    const drag = dragStateRef.current;
    if (!drag) return;
    setPos({
      x: Math.min(Math.max(drag.baseX + event.clientX - drag.startX, -200), window.innerWidth - 120),
      y: Math.min(Math.max(drag.baseY + event.clientY - drag.startY, 0), window.innerHeight - 60),
    });
  }, []);

  const handleHeaderPointerUp = useCallback(() => {
    dragStateRef.current = null;
  }, []);

  useEffect(() => {
    const onKey = (event) => {
      if (event.key === 'Escape') onClose?.();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  async function copyAll() {
    // 優先走 Go 端剪貼簿（不碰網頁選取範圍 → 不會整片反黑）
    try {
      await window.go?.main?.App?.CopyCodeArtifactToClipboard?.(meta.file_name);
    } catch (_) {
      try { await navigator.clipboard.writeText(content); } catch (_) { /* 兩路都失敗就算了 */ }
    }
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
    onCopyToast?.();
  }

  const compileBadge = (() => {
    switch (meta.compile_status) {
      case 'success': return {text: '編譯通過', tone: 'ok'};
      case 'failed': return {text: '編譯失敗', tone: 'bad'};
      case 'tool_missing': return {text: '缺編譯工具', tone: 'warn'};
      case 'unsupported': return {text: '免編譯', tone: 'dim'};
      default: return null;
    }
  })();

  return createPortal(
    <div className="code-artifact-modal" ref={panelRef} style={{left: pos.x, top: pos.y}} role="dialog" aria-label={`程式碼檢視：${meta.display_name || meta.file_name}`}>
      <header
        className="code-artifact-modal-header"
        onPointerDown={handleHeaderPointerDown}
        onPointerMove={handleHeaderPointerMove}
        onPointerUp={handleHeaderPointerUp}
      >
        <div className="code-artifact-modal-title">
          <strong>{meta.display_name || meta.file_name}</strong>
          <span className="code-artifact-lang-badge">{meta.language_label || meta.language}</span>
          {compileBadge && <span className={`code-artifact-compile-badge tone-${compileBadge.tone}`}>{compileBadge.text}</span>}
        </div>
        <div className="code-artifact-modal-actions">
          <button type="button" onClick={copyAll}>{copied ? '已複製 ✓' : '複製'}</button>
          <button type="button" onClick={() => onExport?.(meta.file_name)}>匯出到資料夾…</button>
          <button type="button" onClick={onClose} aria-label="關閉">×</button>
        </div>
      </header>
      <div className="code-artifact-modal-meta">
        {meta.summary && <p className="code-artifact-summary">{meta.summary}</p>}
        {usedSlots.size > 0 && (
          <div className="code-artifact-mark-legend">
            <small>LLM 標示：</small>
            {MARK_LEGEND.filter((item) => usedSlots.has(item.slot)).map((item) => (
              <small className={item.className} key={item.slot}>{item.label}</small>
            ))}
          </div>
        )}
      </div>
      {/* .message-text + data-message-id → 既有 8 色標色管線可直接圈選上色 */}
      <div className="code-artifact-modal-body">
        <pre className="message-text code-artifact-pre" data-message-id={`code-artifact:${meta.file_name}`}>
          {segments.map((segment, index) => segment.slot === 0
            ? <span key={index}>{segment.text}</span>
            : <span key={index} className={`code-llm-mark-${segment.slot}`} title={MARK_LEGEND[segment.slot - 1]?.label}>{segment.text}</span>)}
        </pre>
      </div>
      <footer className="code-artifact-modal-footer">
        <div className="code-artifact-tags">
          {(meta.tags || []).map((tag) => <span className="code-artifact-tag" key={tag}>{tag}</span>)}
          <span className="code-artifact-file-chip" title={detail?.path || ''}>{meta.file_name}</span>
        </div>
      </footer>
      <div className="code-artifact-resize-hint" aria-hidden="true">⇲</div>
    </div>,
    document.body,
  );
}
