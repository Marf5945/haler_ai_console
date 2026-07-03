import React, {useEffect, useRef, useState} from 'react';
import './popout.css';

// PopoutChat — 釘選對話框（popout）子視窗的輕量聊天 UI。
//
// 這個視窗是純顯示殼：所有記憶讀寫與模型呼叫都在主行程完成，
// 透過子行程的 PopoutApp binding（window.go.main.PopoutApp）轉發。
// 記憶隔離：這裡只看得到自己 agent 的 talk_full.md，碰不到其他對話。

function popoutBridge() {
  return window?.go?.main?.PopoutApp || null;
}

function callPopout(method, ...args) {
  const bridge = popoutBridge();
  const fn = bridge?.[method];
  if (typeof fn !== 'function') {
    return Promise.reject(new Error('popout bridge unavailable'));
  }
  return fn(...args);
}

function messageKind(text) {
  const value = String(text || '');
  if (value.startsWith('Ai:')) return 'assistant';
  if (value.startsWith('[')) return 'system';
  return 'user';
}

function messageBody(text) {
  const value = String(text || '');
  return value.startsWith('Ai:') ? value.slice(3) : value;
}

export default function PopoutChat({boot = {}}) {
  const [state, setState] = useState(null);
  const [messages, setMessages] = useState([]);
  const [draft, setDraft] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [loadFailed, setLoadFailed] = useState(false);
  const listRef = useRef(null);
  const inputRef = useRef(null);

  // 初始載入：binding 注入可能比 React mount 晚，帶重試。
  useEffect(() => {
    let cancelled = false;
    let tries = 0;
    function load() {
      callPopout('GetState')
        .then((snapshot) => {
          if (cancelled) return;
          setState(snapshot || {});
          setMessages(Array.isArray(snapshot?.messages) ? snapshot.messages : []);
        })
        .catch(() => {
          if (cancelled) return;
          tries += 1;
          if (tries < 40) {
            setTimeout(load, 250);
          } else {
            setLoadFailed(true);
          }
        });
    }
    load();
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    const node = listRef.current;
    if (node) node.scrollTop = node.scrollHeight;
  }, [messages, busy]);

  function send(event) {
    event?.preventDefault?.();
    const text = String(draft || '').trim();
    if (!text || busy) return;
    setDraft('');
    setError('');
    setBusy(true);
    setMessages((prev) => [...prev, text]);
    callPopout('Send', text)
      .then((resp) => {
        const reply = String(resp?.reply || '');
        const respError = String(resp?.error || '');
        if (reply) setMessages((prev) => [...prev, `Ai:${reply}`]);
        if (respError) setError(respError);
      })
      .catch((err) => setError(err?.message || String(err)))
      .finally(() => {
        setBusy(false);
        inputRef.current?.focus?.();
      });
  }

  function unpin() {
    callPopout('Unpin').catch(() => {});
  }

  const title = state?.name || boot?.name || '';
  const personaName = state?.persona_name || '';
  const adapterId = state?.adapter_id || '';

  return (
    <div className="popout-shell">
      {/* --wails-draggable：除了 OS 標題列外，抓著頂欄也能拖動整個視窗 */}
      <header className="popout-header" style={{'--wails-draggable': 'drag'}}>
        <button
          className="popout-pin popout-pin-active"
          type="button"
          title="解除釘選，收回主視窗"
          onClick={unpin}
        >
          <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
            <path
              fill="currentColor"
              d="M14.6 2.6a1 1 0 0 1 1.4 0l5.4 5.4a1 1 0 0 1 0 1.4l-1.2 1.2a1 1 0 0 1-1 .25l-.9-.27-3.8 3.8.3 2.9a1 1 0 0 1-.29.82l-.9.9a1 1 0 0 1-1.41 0l-3.3-3.3-5 5a1 1 0 0 1-1.41-1.41l5-5-3.3-3.3a1 1 0 0 1 0-1.41l.9-.9a1 1 0 0 1 .83-.29l2.9.3 3.8-3.8-.28-.9a1 1 0 0 1 .25-1z"
            />
          </svg>
        </button>
        <div className="popout-title">
          <strong>{title}</strong>
          <span>{[personaName, adapterId].filter(Boolean).join(' · ')}</span>
        </div>
        {busy && <span className="popout-busy" aria-label="thinking">…</span>}
      </header>

      <main className="popout-messages" ref={listRef}>
        {loadFailed && (
          <div className="popout-banner">無法連上主程式，請關閉此視窗後重新釘選。</div>
        )}
        {messages.map((message, index) => {
          const kind = messageKind(message);
          return (
            <div key={`${index}-${String(message).slice(0, 24)}`} className={`popout-row popout-row-${kind}`}>
              <div className="popout-bubble">{messageBody(message)}</div>
            </div>
          );
        })}
        {busy && (
          <div className="popout-row popout-row-assistant">
            <div className="popout-bubble popout-bubble-pending">思考中…</div>
          </div>
        )}
      </main>

      {error && <div className="popout-error">{error}</div>}

      <form className="popout-composer" onSubmit={send}>
        <input
          ref={inputRef}
          value={draft}
          disabled={busy || loadFailed}
          placeholder={busy ? '等待回覆中…' : '輸入訊息…'}
          onChange={(event) => setDraft(event.target.value)}
          autoFocus
        />
        <button type="submit" disabled={busy || loadFailed || !String(draft || '').trim()}>
          送出
        </button>
      </form>
    </div>
  );
}
