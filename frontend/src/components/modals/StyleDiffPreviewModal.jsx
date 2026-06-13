import {createPortal} from 'react-dom';
import useI18n from '../../locales/useI18n';

export default function StyleDiffPreviewModal({diffJson, onConfirm, onCancel}) {
  const t = useI18n(s => s.t);
  const entries = typeof diffJson === 'string' ? JSON.parse(diffJson) : diffJson;
  const diffLines = Object.entries(entries || {});

  return createPortal(
    <div className="sandbox-stop-overlay" role="dialog" aria-label={t("styleDiff.previewLabel")}>
      <article className="style-diff-preview">
        <header>
          <span>{t("styleDiff.previewTitle")}</span>
        </header>
        <div className="style-diff-list">
          {diffLines.map(([key, value]) => (
            <div className="style-diff-row" key={key}>
              <span className="style-diff-key">{key}</span>
              <span className="style-diff-arrow">→</span>
              <span className="style-diff-value">{typeof value === 'object' ? JSON.stringify(value) : String(value)}</span>
            </div>
          ))}
          {diffLines.length === 0 && <p className="style-diff-empty">{t("styleDiff.noDiff")}</p>}
        </div>
        <div className="style-diff-actions">
          <button type="button" onClick={onConfirm}>{t('common.apply')}</button>
          <button type="button" className="sandbox-stop-discard" onClick={onCancel}>{t('common.cancel')}</button>
        </div>
      </article>
    </div>,
    document.body,
  );
}
