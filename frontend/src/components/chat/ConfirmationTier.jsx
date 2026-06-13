import useI18n from '../../locales/useI18n';
import LongPressConfirmButton from './LongPressConfirmButton';

export default function ConfirmationTier({
  riskTier = 'none', impactExplanation = '', impactExpanded = false,
  longPressProgress = 0, gachaPhase = null,
  onNormalConfirm, onNormalReject, onHighRiskYes, onPressStart, onPressEnd,
}) {
  const {t} = useI18n();
  if (riskTier === 'none') return null;

  // 第一層：普通風險 — 對話式 [是] [否] ▸ 自行輸入
  if (riskTier === 'normal') {
    return (
      <div className="confirmation-tier composer composer-risk">
        <button type="button" className="risk-confirm-btn risk-yes" onClick={onNormalConfirm}>{t('confirm.yes')}</button>
        <button type="button" className="risk-confirm-btn risk-no" onClick={onNormalReject}>{t('confirm.no')}</button>
        <div className="risk-custom-wrap">
          <span>{t('confirm.customInput')}</span>
          <input type="text" placeholder={t('confirm.customPlaceholder')} />
        </div>
      </div>
    );
  }

  // 第二層：中風險（Readiness Gate）— 長按確認
  if (riskTier === 'medium') {
    return (
      <div className="confirmation-tier">
        <LongPressConfirmButton
          label={t('confirm.longPressConfirm')}
          progress={longPressProgress}
          gachaPhase={gachaPhase}
          onPressStart={onPressStart}
          onPressEnd={onPressEnd}
        />
        <button type="button" className="risk-confirm-btn risk-no" onClick={onNormalReject}>{t('confirm.cancel')}</button>
      </div>
    );
  }

  // 第三層：高風險 — 影響說明 → 變紅 → 長按確認
  if (riskTier === 'high') {
    return (
      <div style={{display: 'flex', flexDirection: 'column', gap: 8}}>
        {/* 未展開時：顯示 [是] [否] */}
        {!impactExpanded && (
          <div className="confirmation-tier composer composer-risk">
            <button type="button" className="risk-confirm-btn risk-yes" onClick={onHighRiskYes}>{t('confirm.yes')}</button>
            <button type="button" className="risk-confirm-btn risk-no" onClick={onNormalReject}>{t('confirm.no')}</button>
            <div className="risk-custom-wrap">
              <span>{t('confirm.customInput')}</span>
              <input type="text" placeholder={t('confirm.customPlaceholder')} />
            </div>
          </div>
        )}
        {/* 展開後：顯示影響說明 + 紅色長按確認按鈕 */}
        {impactExpanded && (
          <>
            <div className="risk-impact-panel">
              <div className="impact-title">{t('confirm.willCause')}</div>
              <div className="impact-list">
                {(impactExplanation || t('confirm.highRiskFallback')).split('\n').map((line, i) => (
                  <span key={i} className="impact-item">{line}</span>
                ))}
              </div>
            </div>
            <div className="confirmation-tier">
              <LongPressConfirmButton
                label={t('confirm.longPressExecute')}
                danger
                progress={longPressProgress}
                gachaPhase={gachaPhase}
                onPressStart={onPressStart}
                onPressEnd={onPressEnd}
              />
              <button type="button" className="risk-confirm-btn risk-no" onClick={onNormalReject}>{t('confirm.cancel')}</button>
            </div>
          </>
        )}
      </div>
    );
  }

  return null;
}
