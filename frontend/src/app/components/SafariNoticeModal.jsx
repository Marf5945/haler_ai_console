import {createPortal} from 'react-dom';
import useI18n from '../../locales/useI18n';

export default function SafariNoticeModal({notice, onDismiss}) {
  const t = useI18n(s => s.t);
  return createPortal(
    <div className="sandbox-stop-overlay" role="dialog" aria-label={t("safari.noticeLabel")}>
      <article className="safari-notice-modal">
        <header>
          <span>{t("safari.noticeTitle")}</span>
        </header>
        <p className="safari-notice-body">{notice.message || notice}</p>
        {notice.details && <p className="safari-notice-details">{notice.details}</p>}
        <div className="safari-notice-actions">
          <button type="button" onClick={onDismiss}>{t("safari.acknowledge")}</button>
        </div>
      </article>
    </div>,
    document.body,
  );
}
