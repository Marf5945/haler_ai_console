import {createPortal} from 'react-dom';
import useI18n from '../../locales/useI18n';

export default function TrustedSessionExpiredDialog({onChoice}) {
  const t = useI18n(s => s.t);
  return createPortal(
    <div className="sandbox-stop-overlay" role="dialog" aria-label={t("trust.expiredLabel")}>
      <article className="sandbox-stop-card">
        <header>
          <span>{t("trust.expiredTitle")}</span>
        </header>
        <p>{t("trust.expiredHint")}</p>
        <div className="sandbox-stop-actions">
          <button type="button" onClick={() => onChoice('reenable')}>
            {t("trust.reauthOS")}
          </button>
          <button type="button" onClick={() => onChoice('allow_once')}>
            {t("trust.allowOnce")}
          </button>
          <button type="button" className="sandbox-stop-discard" onClick={() => onChoice('cancel')}>
            {t('common.cancel')}
          </button>
        </div>
      </article>
    </div>,
    document.body,
  );
}
