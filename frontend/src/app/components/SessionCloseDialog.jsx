import {useState} from 'react';
import useI18n from '../../locales/useI18n';

export default function SessionCloseDialog({analysis, onSave, onDiscard, onCancel}) {
  const t = useI18n(s => s.t);
  const [subName, setSubName] = useState(analysis?.suggested_name || '');
  if (analysis?.mode === 'active_task') {
    return (
      <div className="session-close-overlay" role="dialog" aria-modal="true" aria-label={t("session.closeConfirmLabel")}>
        <div className="session-close-dialog">
          <h3>{t('session.closeTaskTitle')}</h3>
          <p className="session-close-hint">
            {t('session.closeTaskHint')}
          </p>
          <div className="session-close-actions">
            <button type="button" className="session-close-btn session-close-save" onClick={onDiscard}>
              {t('session.closeTaskConfirm')}
            </button>
            <button type="button" className="session-close-btn session-close-discard" onClick={onCancel}>
              {t('session.goBack')}
            </button>
          </div>
        </div>
      </div>
    );
  }
  if (analysis?.mode === 'active_sub') {
    return (
      <div className="session-close-overlay" role="dialog" aria-modal="true" aria-label={t("session.closeConfirmLabel")}>
        <div className="session-close-dialog">
          <h3>{t("session.closeSubTitle")}</h3>
          <p className="session-close-hint">
            {t('session.closeSubHint')}
          </p>
          <div className="session-close-actions">
            <button type="button" className="session-close-btn session-close-save" onClick={onDiscard}>
              {t('session.close')}
            </button>
            <button type="button" className="session-close-btn session-close-discard" onClick={onCancel}>
              {t('session.goBack')}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="session-close-overlay" role="dialog" aria-modal="true" aria-label={t("session.closePreConfirmLabel")}>
      <div className="session-close-dialog">
        <h3>{t("session.createWorkflowTitle")}</h3>
        <p className="session-close-stats">
          {t("session.workflowStats", { total: analysis.total_actions, direct: analysis.main_direct_actions, ratio: Math.round(analysis.direct_ratio * 100) })}
        </p>
        <p className="session-close-hint">
          {t("session.workflowSaveHint")}
        </p>
        <label className="session-close-name-label">
          {t("session.workflowName")}
          <input
            type="text"
            value={subName}
            onChange={(e) => setSubName(e.target.value)}
            placeholder={t("session.workflowNamePlaceholder")}
            autoFocus
          />
        </label>
        <div className="session-close-actions">
          <button type="button" className="session-close-btn session-close-save" onClick={() => onSave(subName)}>
            {t('session.yes')}
          </button>
          <button type="button" className="session-close-btn session-close-discard" onClick={onDiscard}>
            {t('session.no')}
          </button>
        </div>
      </div>
    </div>
  );
}
