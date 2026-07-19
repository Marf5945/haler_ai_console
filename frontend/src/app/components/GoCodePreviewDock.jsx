import {useEffect, useMemo, useRef, useState} from 'react';
import {GetGoProgramAuthoringSourcePreview} from '../../../wailsjs/go/main/App';
import {callWails} from '../../lib/callWails';

export const GO_PROGRAM_SOURCE_EVENT = 'ai-console:open-go-program-source';

const goKeywords = new Set([
  'break', 'case', 'chan', 'const', 'continue', 'default', 'defer', 'else',
  'fallthrough', 'for', 'func', 'go', 'goto', 'if', 'import', 'interface',
  'map', 'package', 'range', 'return', 'select', 'struct', 'switch', 'type', 'var',
]);

export function openGoProgramSourcePreview(runID) {
  const id = String(runID || '').trim();
  if (!id) return;
  window.dispatchEvent(new CustomEvent(GO_PROGRAM_SOURCE_EVENT, {detail: {runID: id}}));
}

export default function GoCodePreviewDock() {
  const [browser, setBrowser] = useState(null);
  const [windows, setWindows] = useState([]);
  const dragRef = useRef(null);
  const zRef = useRef(70);

  useEffect(() => {
    const onOpen = (event) => {
      const runID = String(event.detail?.runID || '').trim();
      if (runID) void loadPreview(runID);
    };
    window.addEventListener(GO_PROGRAM_SOURCE_EVENT, onOpen);
    return () => window.removeEventListener(GO_PROGRAM_SOURCE_EVENT, onOpen);
  }, []);

  useEffect(() => {
    const onPointerMove = (event) => {
      const drag = dragRef.current;
      if (!drag) return;
      setWindows((current) => current.map((win) => (
        win.key === drag.key
          ? {...win, x: event.clientX - drag.offsetX, y: event.clientY - drag.offsetY}
          : win
      )));
    };
    const onPointerUp = () => {
      dragRef.current = null;
    };
    document.addEventListener('pointermove', onPointerMove);
    document.addEventListener('pointerup', onPointerUp);
    return () => {
      document.removeEventListener('pointermove', onPointerMove);
      document.removeEventListener('pointerup', onPointerUp);
    };
  }, []);

  async function loadPreview(runID) {
    setBrowser({runID, loading: true, error: '', preview: null});
    try {
      if (typeof GetGoProgramAuthoringSourcePreview !== 'function') {
        throw new Error('Wails binding GetGoProgramAuthoringSourcePreview 尚未產生');
      }
      const preview = await callWails(() => GetGoProgramAuthoringSourcePreview(runID));
      setBrowser({runID, loading: false, error: '', preview});
    } catch (error) {
      setBrowser({runID, loading: false, error: error?.message || String(error), preview: null});
    }
  }

  function openFileWindow(preview, file) {
    if (!preview || !file) return;
    const key = `${preview.run_id}:${preview.attempt}:${file.path}`;
    const existing = windows.find((win) => win.key === key);
    if (existing) {
      bringToFront(key);
      return;
    }
    const offset = windows.length * 22;
    setWindows((current) => [...current, {
      key,
      preview,
      file,
      x: 96 + offset,
      y: 74 + offset,
      z: ++zRef.current,
    }]);
  }

  function bringToFront(key) {
    setWindows((current) => current.map((win) => (
      win.key === key ? {...win, z: ++zRef.current} : win
    )));
  }

  function closeWindow(key) {
    setWindows((current) => current.filter((win) => win.key !== key));
  }

  function startDrag(event, win) {
    if (event.target?.closest?.('button')) return;
    event.currentTarget.setPointerCapture?.(event.pointerId);
    dragRef.current = {
      key: win.key,
      offsetX: event.clientX - win.x,
      offsetY: event.clientY - win.y,
    };
    bringToFront(win.key);
  }

  return (
    <>
      {browser && (
        <section className="go-code-browser" role="dialog" aria-label="Go program source list">
          <header>
            <div>
              <strong>{browser.preview?.program_name || browser.preview?.program_id || 'Go program'}</strong>
              <small>
                {browser.preview?.attempt ? `attempt ${browser.preview.attempt}` : 'source preview'}
                {browser.preview?.attempt_hash ? ` · ${String(browser.preview.attempt_hash).slice(0, 12)}` : ''}
              </small>
            </div>
            <button type="button" onClick={() => setBrowser(null)} aria-label="關閉">×</button>
          </header>
          {browser.loading && <p className="go-code-browser-note">讀取 Go 程式碼...</p>}
          {browser.error && <p className="go-code-browser-error">{browser.error}</p>}
          {!browser.loading && !browser.error && (
            <div className="go-code-file-list">
              {(browser.preview?.files || []).length === 0 && (
                <p className="go-code-browser-note">尚無可預覽的 .go 檔。</p>
              )}
              {(browser.preview?.files || []).map((file) => (
                <button
                  key={file.path}
                  type="button"
                  className="go-code-file-row"
                  onClick={() => openFileWindow(browser.preview, file)}
                >
                  <span>{file.path}</span>
                  <small>{file.error || `${Math.max(1, Math.round((file.size_bytes || file.content?.length || 0) / 1024))} KB`}</small>
                </button>
              ))}
            </div>
          )}
        </section>
      )}
      {windows.map((win) => (
        <GoCodeWindow
          key={win.key}
          win={win}
          onClose={() => closeWindow(win.key)}
          onBringToFront={() => bringToFront(win.key)}
          onDragStart={(event) => startDrag(event, win)}
        />
      ))}
    </>
  );
}

function GoCodeWindow({win, onClose, onBringToFront, onDragStart}) {
  const tokens = useMemo(() => tokenizeGo(win.file?.content || ''), [win.file?.content]);
  const title = `${win.preview?.program_name || win.preview?.program_id || 'Go program'} / ${win.file?.path || 'source.go'}`;

  function copyCode() {
    navigator.clipboard?.writeText(win.file?.content || '').catch(() => {});
  }

  function handleDragStart(event) {
    const selected = String(window.getSelection?.().toString() || '').trim();
    event.dataTransfer.effectAllowed = 'copy';
    event.dataTransfer.setData('text/plain', selected || win.file?.content || '');
  }

  return (
    <section
      className="go-code-window"
      role="dialog"
      aria-label={title}
      style={{left: `${win.x}px`, top: `${win.y}px`, zIndex: win.z}}
      onPointerDown={onBringToFront}
    >
      <header className="go-code-window-titlebar" onPointerDown={onDragStart}>
        <div>
          <strong>{win.file?.path || 'source.go'}</strong>
          <small>{win.preview?.program_name || win.preview?.program_id || win.preview?.run_id}</small>
        </div>
        <button type="button" onClick={onClose} aria-label="關閉">×</button>
      </header>
      <div className="go-code-window-actions">
        <span>{win.preview?.attempt ? `attempt ${win.preview.attempt}` : 'source'}{win.preview?.attempt_hash ? ` · ${String(win.preview.attempt_hash).slice(0, 12)}` : ''}</span>
        <button type="button" onClick={copyCode}>複製</button>
      </div>
      {win.file?.error ? (
        <p className="go-code-browser-error">{win.file.error}</p>
      ) : (
        <pre className="go-code-preview" draggable onDragStart={handleDragStart}>
          <code>
            {tokens.map((token, index) => (
              token.className
                ? <span key={index} className={token.className}>{token.text}</span>
                : <span key={index}>{token.text}</span>
            ))}
          </code>
        </pre>
      )}
    </section>
  );
}

function tokenizeGo(code) {
  const tokens = [];
  let i = 0;
  let expectName = '';
  const push = (text, className = '') => {
    if (text) tokens.push({text, className});
  };

  while (i < code.length) {
    const ch = code[i];
    const next = code[i + 1];

    if (ch === '/' && next === '/') {
      const end = code.indexOf('\n', i);
      const stop = end < 0 ? code.length : end;
      push(code.slice(i, stop), 'tok-comment');
      i = stop;
      continue;
    }
    if (ch === '/' && next === '*') {
      const end = code.indexOf('*/', i + 2);
      const stop = end < 0 ? code.length : end + 2;
      push(code.slice(i, stop), 'tok-comment');
      i = stop;
      continue;
    }
    if (ch === '`') {
      const end = code.indexOf('`', i + 1);
      const stop = end < 0 ? code.length : end + 1;
      push(code.slice(i, stop), 'tok-string');
      i = stop;
      continue;
    }
    if (ch === '"' || ch === "'") {
      const quote = ch;
      let stop = i + 1;
      while (stop < code.length) {
        if (code[stop] === '\\') {
          stop += 2;
          continue;
        }
        if (code[stop] === quote) {
          stop += 1;
          break;
        }
        stop += 1;
      }
      push(code.slice(i, stop), 'tok-string');
      i = stop;
      continue;
    }
    if (/[0-9]/.test(ch)) {
      let stop = i + 1;
      while (stop < code.length && /[0-9A-Fa-f_xXoObB.]/.test(code[stop])) stop += 1;
      push(code.slice(i, stop), 'tok-number');
      i = stop;
      continue;
    }
    if (/[A-Za-z_]/.test(ch)) {
      let stop = i + 1;
      while (stop < code.length && /[A-Za-z0-9_]/.test(code[stop])) stop += 1;
      const word = code.slice(i, stop);
      if (goKeywords.has(word)) {
        push(word, 'tok-keyword');
        expectName = word === 'func' || word === 'type' ? word : '';
      } else if (expectName === 'func') {
        push(word, 'tok-fn');
        expectName = '';
      } else if (expectName === 'type') {
        push(word, 'tok-type');
        expectName = '';
      } else {
        push(word);
      }
      i = stop;
      continue;
    }
    push(ch);
    if (!/\s/.test(ch)) expectName = '';
    i += 1;
  }
  return tokens;
}
