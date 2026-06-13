import useI18n from '../../locales/useI18n';

export default function ReauthInterceptDialog({tool, onRetry, onCancel}) {
  const t = useI18n(s => s.t);
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('reauth.title')}>
      <section className="reauth-dialog">
        <header>
          <span className="reauth-dialog-icon">⚡</span>
          <strong>{tool.title}</strong>
        </header>
        <p>{t('reauth.disconnected')}</p>
        <p className="reauth-dialog-hint">{t('reauth.hint')}</p>
        <div className="reauth-dialog-actions">
          <button type="button" onClick={onRetry}>{t('reauth.retry')}</button>
          <button type="button" onClick={onCancel}>{t('common.cancel')}</button>
        </div>
      </section>
    </div>
  );
}
