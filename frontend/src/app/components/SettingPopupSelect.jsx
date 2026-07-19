import {useEffect, useId, useLayoutEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';

const POPUP_GAP = 4;
const POPUP_MARGIN = 12;
const OPTION_HEIGHT = 38;
const POPUP_THEME_PALETTES = Object.freeze({
  onanegiku: {list: '#140d07', option: '#21170f', active: '#51321a', text: '#fff8ef', border: '#5a4635'},
  white: {list: '#f3f3f3', option: '#ffffff', active: '#d8d8d8', text: '#202020', border: '#c8c8c8'},
  'pink-black': {list: '#080408', option: '#171015', active: '#551039', text: '#fff4fb', border: '#7a2355'},
  green: {list: '#fff4d2', option: '#f2e2ad', active: '#cdbb7d', text: '#193015', border: '#8e7c46'},
  blue: {list: '#10223c', option: '#173452', active: '#286b83', text: '#dff8ff', border: '#3f7894'},
});

// Viewport-aware popup picker. The list is portalled out of the scrollable
// settings rail so the last languages stay reachable in compact windows.
export default function SettingPopupSelect({icon, label, value, options, onSelect}) {
  const [open, setOpen] = useState(false);
  const [popupPosition, setPopupPosition] = useState(null);
  const rootRef = useRef(null);
  const triggerRef = useRef(null);
  const listRef = useRef(null);
  const optionRefs = useRef([]);
  const pendingFocusRef = useRef(null);
  const listboxId = useId();

  function updatePopupPosition() {
    const trigger = triggerRef.current;
    if (!trigger) return;
    const theme = trigger.closest('.console-shell')?.dataset?.theme || 'onanegiku';
    const rect = trigger.getBoundingClientRect();
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;
    const desiredHeight = Math.min(options.length * OPTION_HEIGHT + 8, 320);
    const availableBelow = Math.max(0, viewportHeight - rect.bottom - POPUP_GAP - POPUP_MARGIN);
    const availableAbove = Math.max(0, rect.top - POPUP_GAP - POPUP_MARGIN);
    const openAbove = availableBelow < desiredHeight && availableAbove > availableBelow;
    const maxHeight = Math.max(96, Math.min(desiredHeight, openAbove ? availableAbove : availableBelow));
    const popupHeight = Math.min(desiredHeight, maxHeight);
    const width = Math.max(96, Math.min(Math.max(176, rect.width), viewportWidth - (POPUP_MARGIN * 2)));
    const rtl = document.documentElement.dir === 'rtl';
    const preferredLeft = rtl ? rect.left : rect.right - width;
    const left = Math.min(
      Math.max(POPUP_MARGIN, preferredLeft),
      Math.max(POPUP_MARGIN, viewportWidth - width - POPUP_MARGIN),
    );
    const top = openAbove ? rect.top - popupHeight - POPUP_GAP : rect.bottom + POPUP_GAP;
    setPopupPosition({left, top: Math.max(POPUP_MARGIN, top), width, maxHeight, theme});
  }

  function openWithFocus(index) {
    pendingFocusRef.current = index;
    setOpen(true);
  }

  useLayoutEffect(() => {
    if (!open) return undefined;
    updatePopupPosition();
    const onViewportChange = () => updatePopupPosition();
    window.addEventListener('resize', onViewportChange);
    window.addEventListener('scroll', onViewportChange, true);
    const frame = window.requestAnimationFrame(() => {
      const requested = pendingFocusRef.current;
      if (requested == null) return;
      const selected = Math.max(0, options.indexOf(value));
      const index = requested === 'selected' ? selected : requested;
      optionRefs.current[index]?.focus();
      pendingFocusRef.current = null;
    });
    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener('resize', onViewportChange);
      window.removeEventListener('scroll', onViewportChange, true);
    };
  }, [open, options, value]);

  useEffect(() => {
    if (!open) return undefined;
    const onDocClick = (event) => {
      if (rootRef.current?.contains(event.target) || listRef.current?.contains(event.target)) return;
      setOpen(false);
    };
    const onKey = (event) => {
      if (event.key !== 'Escape') return;
      setOpen(false);
      triggerRef.current?.focus();
    };
    document.addEventListener('mousedown', onDocClick);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDocClick);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  function handleListKeyDown(event) {
    const current = optionRefs.current.indexOf(document.activeElement);
    let next = current;
    if (event.key === 'ArrowDown') next = Math.min(options.length - 1, current + 1);
    else if (event.key === 'ArrowUp') next = Math.max(0, current - 1);
    else if (event.key === 'Home') next = 0;
    else if (event.key === 'End') next = options.length - 1;
    else return;
    event.preventDefault();
    optionRefs.current[next]?.focus();
  }

  const popupPalette = POPUP_THEME_PALETTES[popupPosition?.theme] || POPUP_THEME_PALETTES.onanegiku;

  const popup = open && popupPosition && (
    <div
      id={listboxId}
      ref={listRef}
      role="listbox"
      className="settings-popup-list settings-popup-list-portal"
      data-theme={popupPosition.theme}
      onKeyDown={handleListKeyDown}
      style={{
        position: 'fixed',
        left: popupPosition.left,
        top: popupPosition.top,
        width: popupPosition.width,
        maxHeight: popupPosition.maxHeight,
        overflowY: 'auto',
        overscrollBehavior: 'contain',
        zIndex: 1000,
        padding: 4,
        borderRadius: 10,
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
        background: popupPalette.list,
        backgroundImage: 'none',
        color: popupPalette.text,
        border: `1px solid ${popupPalette.border}`,
        boxShadow: '0 18px 48px rgba(0, 0, 0, 0.72)',
        isolation: 'isolate',
        opacity: 1,
        backdropFilter: 'none',
        WebkitBackdropFilter: 'none',
      }}
    >
      {options.map((option, index) => (
        <button
          key={option}
          ref={(node) => { optionRefs.current[index] = node; }}
          type="button"
          role="option"
          aria-selected={option === value}
          className={`settings-popup-option${option === value ? ' active' : ''}`}
          style={{
            minHeight: OPTION_HEIGHT,
            flexShrink: 0,
            width: '100%',
            padding: '7px 10px',
            overflow: 'hidden',
            whiteSpace: 'nowrap',
            textOverflow: 'ellipsis',
            textAlign: 'start',
            background: option === value ? popupPalette.active : popupPalette.option,
            backgroundImage: 'none',
            color: popupPalette.text,
            opacity: 1,
          }}
          onClick={() => {
            setOpen(false);
            triggerRef.current?.focus();
            if (option !== value) onSelect(option);
          }}
        >
          {option}
        </button>
      ))}
    </div>
  );

  return (
    <div className="settings-select-row" ref={rootRef} style={{position: 'relative'}}>
      <span className="settings-select-label"><i>{icon}</i>{label}</span>
      <button
        ref={triggerRef}
        type="button"
        aria-label={`${label}: ${value}`}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? listboxId : undefined}
        onClick={() => {
          if (open) {
            setOpen(false);
          } else {
            openWithFocus('selected');
          }
        }}
        onKeyDown={(event) => {
          if (event.key === 'ArrowDown') {
            event.preventDefault();
            openWithFocus('selected');
          } else if (event.key === 'ArrowUp') {
            event.preventDefault();
            openWithFocus(options.length - 1);
          }
        }}
      >
        <span className="settings-select-value">{value}</span>
        <span className="settings-select-caret">⌄</span>
      </button>
      {popup ? createPortal(popup, document.body) : null}
    </div>
  );
}
