import {createPortal} from 'react-dom';
import {t as _t} from '../../locales/useI18n';

export default function ConsequenceMenuOverlay({cards, result, onExecute, onRecreate}) {
  const card = (cards || []).find((item) => item.risk_class === 'user_owned_asset_destructive' && item.operation === 'purge_project');
  if (!card) return null;
  const retryCount = Number(result?.retry_count || 0);
  // i18n: confirm
  return createPortal(
    <div className="project-manage-overlay" role="dialog" aria-modal="true" aria-label={_t('confirm.destructiveTitle')}>
      <div className="project-manage-popup consequence-menu">
        <div className="project-manage-header">
          <h3>{_t('confirm.destructiveTitle')}</h3>
        </div>
        <div className="consequence-table">
          <div><span>operation</span><strong>{card.operation}</strong></div>
          <div><span>target</span><strong>{card.target}</strong></div>
          <div><span>affected scope</span><strong>{card.accept_effect}</strong></div>
          <div><span>accept effect</span><strong>{_t('confirm.acceptEffect')}</strong></div>
          <div><span>reject effect</span><strong>{card.reject_effect || _t('confirm.rejectEffect')}</strong></div>
          <div><span>rollback</span><strong>{card.rollback_available ? 'available' : 'not available'}</strong></div>
          <div><span>backup</span><strong>{card.backup_available ? 'available' : 'not available'}</strong></div>
          <div><span>log</span><strong>{card.log_location || 'review/review_decision_log.jsonl'}</strong></div>
        </div>
        {result?.message && <p className="consequence-status">{result.message}</p>}
        <div className="project-manage-actions consequence-actions">
          <button type="button" className="project-manage-btn" disabled={!card.backup_available} onClick={() => onExecute(card, 'backup_then_execute')}>
            <span>{_t('confirm.backupThenExecute')}</span>
          </button>
          <button type="button" className="project-manage-btn project-manage-btn-danger" onClick={() => onExecute(card, 'direct_execute')}>
            <span>{_t('confirm.directExecute')}</span>
          </button>
          <button type="button" className="project-manage-btn" onClick={() => onExecute(card, 'cancel_keep_pending')}>
            <span>{_t('confirm.cancel')}</span>
          </button>
          {retryCount === 1 && (
            <button type="button" className="project-manage-btn" onClick={() => onRecreate(card)}>
              <span>{_t('confirm.recreateReviewCard')}</span>
            </button>
          )}
        </div>
      </div>
    </div>,
    document.body,
  );
}
