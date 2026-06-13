import {useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import {ExportSubHandler, ReorderAdapters, ReorderTabs, SelectSubExportDirectory} from '../../../wailsjs/go/main/App';
import useI18n from '../../locales/useI18n';
import {adapterKey, adapterMeta, callWails, normalizeAdapterDTO} from '../../lib/appShared';
import SettingsMenu from '../settings/SettingsMenu';

export default function Sidebar({
  adapters, adapterList, activeAdapterId,
  adapterModelChoices, adapterModelOptions, onAdapterModelPick, onAdapterModelRefresh,
  onAdapterSelect, onLocalAdapterWake, adapterCandidateLinks, activePanel,
  isToolPopupOpen, panelSettings, onPanelChange, voiceState, voiceInstallBusy,
  onVoiceSettingsChange, onVoiceSettingsRefresh, onVoiceModelInstall, onVoiceModelRemove, onRestoreDefaults, onTogglePanel,
  onToolPopupToggle, onProjectManageOpen, onCreateSubagent, onAdaptersReordered, onAdaptersChanged, onAdapterRemove,
}) {
  const t = useI18n(s => s.t);
  const settingsOpen = activePanel === 'settings';
  // §31: 拖曳狀態
  const [draggedAdapter, setDraggedAdapter] = useState(null);
  const draggedAdapterRef = useRef(null);
  const adapterPointerDragRef = useRef(null);
  const adapterPointerSelectRef = useRef({key: '', time: 0});
  const [exportDialog, setExportDialog] = useState(null); // {name, key} 或 null
  // §M-1 model picker 彈窗：{adapterID, rect, label} 或 null
  const [modelPicker, setModelPicker] = useState(null);
  useEffect(() => {
    if (!modelPicker) return;
    const close = (ev) => {
      if (ev.target && ev.target.closest && ev.target.closest('.adapter-model-picker')) return;
      setModelPicker(null);
    };
    const esc = (ev) => { if (ev.key === 'Escape') setModelPicker(null); };
    window.addEventListener('mousedown', close);
    window.addEventListener('keydown', esc);
    return () => {
      window.removeEventListener('mousedown', close);
      window.removeEventListener('keydown', esc);
    };
  }, [modelPicker]);

  // 優先使用 adapterList（含狀態的完整物件），fallback 用純名稱陣列
  const adapterItems = (adapterList && adapterList.length > 0)
    ? adapterList.map((raw) => {
        const a = normalizeAdapterDTO(raw);
        const id = adapterKey(a);
        const rawKind = String(a.kind || '').toLowerCase();
        const inferredKind = rawKind || (String(id || '').startsWith('llm-api-') ? 'api' : ((a.name || id || '').toLowerCase() === 'main' ? 'main' : 'cli'));
        const rawName = a.name || a.id;
        const localProviderName = String(rawName || id || '').toLowerCase().includes('lm studio') ? 'LM Studio' : 'Ollama';
        return {
          key: id,
          name: rawName,
          displayName: inferredKind === 'local' ? localProviderName : rawName,
          icon: a.icon,
          status: a.status || 'offline',
          kind: inferredKind,
          model: a.model || '',
          endpoint: a.endpoint || '',
          isMain: inferredKind === 'main' || (a.name || a.id || '').toLowerCase() === 'main',
        };
      })
    : (adapters || []).map((name) => ({
        key: name,
        name,
        displayName: name,
        icon: (adapterMeta[name] || adapterMeta.Claude).icon,
        status: 'offline',
        isMain: name.toLowerCase() === 'main',
        kind: name.toLowerCase() === 'main' ? 'main' : 'cli',
      }));
  const isRegistryAdapter = (item) => item && (item.kind === 'cli' || item.kind === 'api' || item.kind === 'local');
  /* i18n: adapter status */ const statusLabel = {online: t('adapter.statusOnline'), offline: t('adapter.statusOffline'), degraded: t('adapter.statusDegraded')};

  const selectAdapterItem = (item, source = 'click') => {
    if (source === 'click') {
      const last = adapterPointerSelectRef.current;
      if (last.key === item.key && Date.now() - last.time < 500) return;
    }
    if (source === 'pointer') {
      adapterPointerSelectRef.current = {key: item.key, time: Date.now()};
    }
    onAdapterSelect?.(item.key);
    const disconnected = item.status === 'offline' || item.status === 'error' || item.status === 'degraded';
    if (item.kind === 'local' && disconnected) onLocalAdapterWake?.(item);
  };

  const moveAdapterItem = (draggedKey, targetKey) => {
    const draggedItem = adapterItems.find((a) => a.key === draggedKey);
    const targetItem = adapterItems.find((a) => a.key === targetKey);
    if (!draggedItem) return;
    if (!targetItem) return;
    const bothSub = draggedItem.kind === 'sub' && targetItem.kind === 'sub';
    const bothAdapters = isRegistryAdapter(draggedItem) && isRegistryAdapter(targetItem);
    if (!bothSub && !bothAdapters) return;

    const orderedItems = adapterItems.filter((a) => bothSub ? a.kind === 'sub' : isRegistryAdapter(a));
    const fromIdx = orderedItems.findIndex((a) => a.key === draggedKey);
    const toIdx = orderedItems.findIndex((a) => a.key === targetItem.key);
    if (fromIdx >= 0 && toIdx >= 0 && fromIdx !== toIdx) {
      const newOrder = orderedItems.map((a) => a.key);
      const [moved] = newOrder.splice(fromIdx, 1);
      newOrder.splice(toIdx, 0, moved);
      onAdaptersReordered?.(newOrder, bothSub ? 'sub' : 'adapter');
      const reorderCall = bothSub ? ReorderTabs : ReorderAdapters;
      callWails(() => reorderCall(JSON.stringify(newOrder)))
        .then(() => onAdaptersChanged?.())
        .catch(console.error);
    }
  };

  const startAdapterPointerDrag = (event, item) => {
    selectAdapterItem(item, 'pointer');
    if (!item || item.isMain) return;
    adapterPointerDragRef.current = {
      key: item.key,
      startX: event.clientX,
      startY: event.clientY,
      moved: false,
      pointerId: event.pointerId,
    };
    draggedAdapterRef.current = item.key;
    event.currentTarget.setPointerCapture?.(event.pointerId);
  };

  const handleAdapterPointerMove = (event) => {
    const drag = adapterPointerDragRef.current;
    if (!drag) return;
    const distance = Math.hypot(event.clientX - drag.startX, event.clientY - drag.startY);
    if (distance < 8 && !drag.moved) return;
    drag.moved = true;
    event.preventDefault();
    setDraggedAdapter(drag.key);
    const target = document.elementFromPoint?.(event.clientX, event.clientY)?.closest?.('[data-adapter-key]');
    const targetKey = target?.dataset?.adapterKey;
    if (targetKey && targetKey !== drag.key) {
      moveAdapterItem(drag.key, targetKey);
    }
  };

  const finishAdapterPointerDrag = (event) => {
    const drag = adapterPointerDragRef.current;
    if (!drag) return;
    event.currentTarget.releasePointerCapture?.(drag.pointerId ?? event.pointerId);
    const leftWindow =
      event.clientX <= 0 ||
      event.clientY <= 0 ||
      event.clientX >= window.innerWidth ||
      event.clientY >= window.innerHeight;
    const droppedOutside = leftWindow || (event.clientX === 0 && event.clientY === 0);
    if (drag.moved && droppedOutside) {
      const item = adapterItems.find((a) => a.key === drag.key);
      if (item && !item.isMain) {
        setExportDialog({name: item.name, key: item.key, kind: item.kind});
      }
    }
    draggedAdapterRef.current = null;
    setDraggedAdapter(null);
    setTimeout(() => {
      if (adapterPointerDragRef.current === drag) {
        adapterPointerDragRef.current = null;
      }
    }, 0);
  };

  // §31.3: 匯出對話框操作
  const handleExportAction = async (action) => {
    if (!exportDialog) return;
    const {name, key} = exportDialog;
    setExportDialog(null);
    if (action === 'cancel') return;
    if (isRegistryAdapter(exportDialog)) {
      if (action === 'remove') {
        await onAdapterRemove?.(key);
        await onAdaptersChanged?.();
      }
      return;
    }
    try {
      const mode = action === 'remove' ? 'export_remove' : 'export_copy';
      const destDir = await callWails(SelectSubExportDirectory);
      if (!destDir) return;
      await callWails(() => ExportSubHandler(key, name, mode, destDir, '[]'));
      await onAdaptersChanged?.();
    } catch (err) {
      console.error('[EXPORT]', err);
    }
  };

  return (
    <aside className="left-panel">
      {/* #I-806: adapter 清單由 Sidecar IPC→adapter_registry 白名單審查後動態更新 */}
      {/* 「可選 Adapter」為提示標籤，非按鈕；下方列出已連接的 CLI adapter */}
      <nav className="adapter-stack">
        <span className="adapter-hint">{t('adapter.selectHint')}</span>
        {adapterItems.map((item) => {
          const meta = adapterMeta[item.name] || adapterMeta.Claude;
          const isActive = item.name === activeAdapterId || item.key === activeAdapterId;
          const sourceBadge = item.kind === 'api' ? 'api' : (item.kind === 'local' ? 'loc' : '');
          const disconnected = item.status === 'offline' || item.status === 'error' || item.status === 'degraded';
          const sourceClass = disconnected ? 'adapter-disconnected' : (item.kind === 'local' ? 'adapter-local-model' : (item.kind === 'api' ? 'adapter-api' : 'adapter-cli'));
          const titleName = item.kind === 'local' && item.model ? `${item.displayName} — ${item.model}` : item.name;
          return (
            <button
              className={`adapter-btn ${sourceClass}${isActive ? ' adapter-active' : ''}${draggedAdapter === item.key ? ' adapter-dragging' : ''}`}
              type="button"
              key={item.key}
              data-adapter-key={item.key}
              draggable={false}
              onPointerDown={(event) => startAdapterPointerDrag(event, item)}
              onPointerMove={handleAdapterPointerMove}
              onPointerUp={finishAdapterPointerDrag}
              onPointerCancel={finishAdapterPointerDrag}
              onDragStart={(event) => event.preventDefault()}
              onClick={(event) => {
                if (adapterPointerDragRef.current?.moved) {
                  event.preventDefault();
                  return;
                }
                selectAdapterItem(item);
              }}
              onContextMenu={(event) => {
                if (item.isMain) return;
                event.preventDefault();
                setExportDialog({name: item.name, key: item.key, kind: item.kind});
              }}
              onDoubleClick={(e) => {
                e.preventDefault();
                // §M-1：有候選清單就開 model picker；無候選保留原 local wake 行為
                const opts = adapterModelOptions?.[item.key];
                if (Array.isArray(opts) && opts.length > 0) {
                  const rect = e.currentTarget.getBoundingClientRect();
                  setModelPicker({adapterID: item.key, rect, label: item.displayName || item.name});
                  return;
                }
                if (item.kind === 'local') onLocalAdapterWake?.(item);
              }}
              title={`${titleName} — ${statusLabel[item.status] || item.status}${item.isMain ? '' : t('adapter.dragToUnlink')}`}
            >
              <span className="adapter-icon">{item.icon || meta.icon}</span>
              <span className="adapter-name">{item.displayName || item.name}</span>
              {sourceBadge && <span className="adapter-source-badge">{sourceBadge}</span>}
              <span className={`adapter-status-dot status-${item.status}`} />
            </button>
          );
        })}
        {/* I-5: adapter_candidate 外部連結 — 從 ListExternalLinksByType("adapter_candidate") */}
        {(adapterCandidateLinks || []).map((link) => (
          <button className="adapter-btn adapter-cli" type="button" key={link.id} title={link.url}>
            <span className="adapter-icon">↗</span>
            <span>{link.label || link.url}</span>
          </button>
        ))}
      </nav>

      {/* §31.3: 匯出對話框 */}
      {exportDialog && (
        <div className="export-dialog-overlay">
          <div className="export-dialog">
            <p>{t('adapter.exportQuestion', { name: exportDialog.name })}</p>
            <div className="export-dialog-actions">
              <button type="button" onClick={() => handleExportAction('remove')}>{t('adapter.remove')}</button>
              <button
                type="button"
                disabled={isRegistryAdapter(exportDialog)}
                className={isRegistryAdapter(exportDialog) ? 'export-action-disabled' : ''}
                onClick={() => handleExportAction('copy')}
              >
                {t('adapter.copyAction')}
              </button>
              <button type="button" onClick={() => handleExportAction('cancel')}>{t('common.cancel')}</button>
            </div>
            {exportDialog.kind === 'cli' && <small>{t('adapter.cliExportNote')}</small>}
            {exportDialog.kind === 'api' && <small>{t('adapter.apiExportNote')}</small>}
          </div>
        </div>
      )}
      <nav className="command-stack">
        <button className="command-btn command-primary" type="button" onClick={onCreateSubagent}><span>⊕</span><span>{t('system.newAgent')}</span></button>
        <button className="command-btn command-tool-trigger" type="button" onClick={onToolPopupToggle}>
          <span>{isToolPopupOpen ? '×' : '⌕'}</span>
          <span>{isToolPopupOpen ? t('common.close') : t('onboarding.tools')}</span>
        </button>
        <button className="command-btn" type="button" onClick={() => onTogglePanel('settings')}>
          <span>{settingsOpen ? '‹' : '⚙'}</span>
          <span>{settingsOpen ? t('common.close') : t('onboarding.settings')}</span>
        </button>
        <button className="command-btn" type="button" onClick={onProjectManageOpen}>
          <span>⛁</span>
          <span>{t('project.manageTitle')}</span>
        </button>
      </nav>
      {settingsOpen && (
        <SettingsMenu
          panel={panelSettings}
          onPanelChange={onPanelChange}
          voiceState={voiceState}
          voiceInstallBusy={voiceInstallBusy}
          onVoiceSettingsChange={onVoiceSettingsChange}
          onVoiceSettingsRefresh={onVoiceSettingsRefresh}
          onVoiceModelInstall={onVoiceModelInstall}
          onVoiceModelRemove={onVoiceModelRemove}
          onRestoreDefaults={onRestoreDefaults}
        />
      )}
      {/* §M-1 model picker 彈窗 — 雙擊 adapter 卡片時顯示，貼在卡片右側 */}
      {modelPicker && typeof document !== 'undefined' && createPortal(
        <div
          className="adapter-model-picker"
          style={{
            left: Math.max(12, Math.min(modelPicker.rect.right + 10, (typeof window !== 'undefined' ? window.innerWidth : 9999) - 300)),
            top: Math.max(12, Math.min(modelPicker.rect.top, (typeof window !== 'undefined' ? window.innerHeight : 9999) - 360)),
          }}
        >
          <header className="adapter-model-picker-head">
            <div>
              <div className="adapter-model-picker-title">{t('adapter.pickModel') || '選擇 model'}</div>
              <small>{modelPicker.label || modelPicker.adapterID}</small>
            </div>
            <div className="adapter-model-picker-actions">
              <button
                type="button"
                title={t('adapter.refreshModels') || '重新整理模型'}
                aria-label={t('adapter.refreshModels') || '重新整理模型'}
                onClick={() => onAdapterModelRefresh?.(modelPicker.adapterID)}
              >
                ↻
              </button>
              <button type="button" aria-label={t('common.close')} onClick={() => setModelPicker(null)}>×</button>
            </div>
          </header>
          {(adapterModelOptions?.[modelPicker.adapterID] || []).map((m) => {
            const isActive = (adapterModelChoices?.[modelPicker.adapterID] || (adapterModelOptions[modelPicker.adapterID] || [])[0]) === m;
            return (
              <button
                key={m}
                type="button"
                className={`adapter-model-picker-item${isActive ? ' active' : ''}`}
                onClick={() => {
                  onAdapterModelPick?.(modelPicker.adapterID, m);
                  setModelPicker(null);
                }}
              >
                <span>{m}</span>
                {isActive && <b>{t('adapter.currentModel') || '目前'}</b>}
              </button>
            );
          })}
        </div>,
        document.body
      )}
    </aside>
  );
}
