import useI18n from '../../locales/useI18n';

export default function ToolCopyConfirmModal({tool, onCancel, onConfirm}) {
  const t = useI18n(s => s.t);
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('tool.confirmInstallLabel')}>
      <section className="tool-copy-modal">
        <p>{t('tool.confirmInstallQuestion', { title: tool.title, id: tool.id })}</p>
        <div>
          <button type="button" onClick={onConfirm}>{t('common.yes')}</button>
          <button type="button" onClick={onCancel}>{t('common.no')}</button>
        </div>
      </section>
    </div>
  );
}
