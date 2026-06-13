import {createPortal} from 'react-dom';
import useI18n from '../../locales/useI18n';

export default function DraftSandboxStopDialog({onPromote, onDismiss}) {
  const t = useI18n(s => s.t);
  return createPortal(
    <div className="sandbox-stop-overlay" role="dialog" aria-label={t('recording.optionsLabel')}>
      <article className="sandbox-stop-card">
        <header>
          <span>{t('recording.stopped')}</span>
        </header>
        <p>{t('recording.processHint')}</p>
        <div className="sandbox-stop-actions">
          <button type="button" onClick={() => onPromote('formal_review')}>
            {t('recording.startReview')}
          </button>
          <button type="button" onClick={() => onPromote('pending_candidate')}>
            {t('recording.saveCandidate')}
          </button>
          <button type="button" className="sandbox-stop-discard" onClick={onDismiss}>
            {t('recording.discard')}
          </button>
        </div>
      </article>
    </div>,
    document.body,
  );
}
