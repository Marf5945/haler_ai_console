import useI18n from '../../locales/useI18n';

export default function AdapterRenameModal({target, draft, onDraftChange, onCancel, onSave}) {
  const t = useI18n(s => s.t);
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('llmSetup.renameAdapterLabel')}>
      <section className="reference-link-modal adapter-rename-modal">
        <h4>{t('llmSetup.renameAdapterTitle')}</h4>
        <input
          className="remote-bridge-name-input"
          autoFocus
          value={draft}
          onChange={(event) => onDraftChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') onSave();
            if (event.key === 'Escape') onCancel();
          }}
          placeholder={target?.name || 'LLM API'}
        />
        <div className="rb-actions">
          <button type="button" onClick={onSave}>{t('llmSetup.confirmRename')}</button>
          <button type="button" className="rb-cancel-btn" onClick={onCancel}>{t('llmSetup.renameLater')}</button>
        </div>
      </section>
    </div>
  );
}
