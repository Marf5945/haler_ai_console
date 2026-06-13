import useI18n from '../../locales/useI18n';

export default function StopRecoveryCard({error, onRestart}) {
  const t = useI18n(s => s.t);
  return (
    <div className="stop-recovery-overlay" role="alert" aria-label={t('system.sidecarInterrupt')}>
      <section className="stop-recovery-card">
        <header className="stop-recovery-header">
          <span className="stop-recovery-icon">⚠</span>
          <strong>{t('system.sidecarInterrupt')}</strong>
        </header>
        <p className="stop-recovery-msg">{error || t('system.sidecarInterruptDetail')}</p>
        <div className="stop-recovery-actions">
          <button type="button" className="stop-recovery-restart" onClick={onRestart}>{t('system.sidecarRestart')}</button>
        </div>
      </section>
    </div>
  );
}
