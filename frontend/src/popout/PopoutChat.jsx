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
  const [configOpen, setConfigOpen] = useState(false);
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

  function updateConfig(patch) {
    const next = {
      persona_id: state?.persona_id || '',
      adapter_id: state?.adapter_id || '',
      model: state?.model || '',
      ...patch,
    };
    setError('');
    callPopout('SetConfig', next)
      .then((snapshot) => {
        setState((prev) => ({...(prev || {}), ...(snapshot || {})}));
        if (Array.isArray(snapshot?.messages)) {
          setMessages(snapshot.messages);
        }
      })
      .catch((err) => setError(err?.message || String(err)));
  }

  const title = state?.name || boot?.name || '';
  const personaName = state?.persona_name || '';
  const adapterId = state?.adapter_id || '';
  const modelName = state?.model || '';
  const personas = Array.isArray(state?.personas) ? state.personas : [];
  const adapters = Array.isArray(state?.adapters) ? state.adapters : [];
  const activeAdapter = adapters.find((adapter) => adapter?.id === adapterId) || adapters[0] || null;
  const modelOptions = Array.isArray(activeAdapter?.models) ? activeAdapter.models : [];
  const adapterLabel = activeAdapter?.name || adapterId || '';
  const contextLabel = [personaName, modelName || adapterLabel || adapterId].filter(Boolean).join(' · ');

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
          <button
            className="popout-context-trigger"
            type="button"
            aria-expanded={configOpen}
            onClick={() => setConfigOpen((open) => !open)}
            title="更改人格與使用模型"
            style={{'--wails-draggable': 'no-drag'}}
          >
            <span>{contextLabel || '選擇人格與模型'}</span>
            <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true">
              <path fill="currentColor" d="M4 6l4 4 4-4z" />
            </svg>
          </button>
          {configOpen && (
            <div className="popout-config-menu" style={{'--wails-draggable': 'no-drag'}}>
              <label>
                <span>人格</span>
                <select
                  value={state?.persona_id || ''}
                  onChange={(event) => updateConfig({persona_id: event.target.value})}
                >
                  {personas.map((persona) => (
                    <option key={persona.id} value={persona.id}>
                      {persona.name || persona.id}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                <span>模型來源</span>
                <select
                  value={adapterId}
                  onChange={(event) => {
                    const nextAdapter = adapters.find((adapter) => adapter?.id === event.target.value);
                    const models = Array.isArray(nextAdapter?.models) ? nextAdapter.models : [];
                    updateConfig({
                      adapter_id: event.target.value,
                      model: nextAdapter?.model || models[0] || '',
                    });
                  }}
                >
                  {adapters.map((adapter) => (
                    <option key={adapter.id} value={adapter.id}>
                      {[adapter.name || adapter.id, adapter.kind].filter(Boolean).join(' / ')}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                <span>模型</span>
                <select
                  value={modelName}
                  disabled={modelOptions.length === 0}
                  onChange={(event) => updateConfig({model: event.target.value})}
                >
                  {modelOptions.length === 0 ? (
                    <option value="">使用預設模型</option>
                  ) : (
                    modelOptions.map((model) => <option key={model} value={model}>{model}</option>)
                  )}
                </select>
              </label>
            </div>
          )}
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
