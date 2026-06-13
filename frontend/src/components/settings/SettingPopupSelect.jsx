import {useEffect, useRef, useState} from 'react';

export default function SettingPopupSelect({icon, label, value, options, onSelect}) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef(null);
  useEffect(() => {
    if (!open) return undefined;
    const onDocClick = (e) => { if (rootRef.current && !rootRef.current.contains(e.target)) setOpen(false); };
    const onKey = (e) => { if (e.key === 'Escape') setOpen(false); };
    document.addEventListener('mousedown', onDocClick);
    document.addEventListener('keydown', onKey);
    return () => { document.removeEventListener('mousedown', onDocClick); document.removeEventListener('keydown', onKey); };
  }, [open]);
  // 顏色一律交給 style.css 的 .settings-popup-list 依主題決定；這裡只留版面配置。
  const popupStyle = {
    position: 'absolute', top: 'calc(100% + 4px)', right: 0, zIndex: 60,
    minWidth: 132, padding: 4, borderRadius: 10,
    display: 'flex', flexDirection: 'column', gap: 2,
  };
  return (
    <label className="settings-select-row" ref={rootRef} style={{position: 'relative'}}>
      <span className="settings-select-label"><i>{icon}</i>{label}</span>
      <button type="button" onClick={() => setOpen((o) => !o)} aria-haspopup="listbox" aria-expanded={open}>
        <span className="settings-select-value">{value}</span>
        <span className="settings-select-caret">⌄</span>
      </button>
      {open && (
        <div role="listbox" className="settings-popup-list" style={popupStyle}>
          {options.map((opt) => (
            <button
              key={opt}
              type="button"
              role="option"
              aria-selected={opt === value}
              className={`settings-popup-option${opt === value ? ' active' : ''}`}
              onClick={() => { setOpen(false); if (opt !== value) onSelect(opt); }}
            >
              {opt}
            </button>
          ))}
        </div>
      )}
    </label>
  );
}
