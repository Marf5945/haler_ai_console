// DecisionCard.jsx — 通用行內決策卡片（同意/不同意/自行輸入）。
import { useState, useCallback } from 'react';
import useI18n from '../locales/useI18n';

// Props: cardId, title, description, preview,
//        agreeLabel, disagreeLabel, agreeReason, disagreeConsequence, onDecision
export default function DecisionCard({
  cardId,
  title,
  description,
  preview,
  agreeLabel = '',
  disagreeLabel = '',
  agreeReason,
  disagreeConsequence,
  onDecision,
}) {
  const t = useI18n(s => s.t);
  const [decided, setDecided] = useState(null);
  const [showCustom, setShowCustom] = useState(false);
  const [customText, setCustomText] = useState('');
  const [previewExpanded, setPreviewExpanded] = useState(false);

  const handleAgree = useCallback(() => {
    setDecided('agree');
    onDecision?.(cardId, 'agree');
  }, [cardId, onDecision]);

  const handleDisagree = useCallback(() => {
    setDecided('disagree');
    onDecision?.(cardId, 'disagree');
  }, [cardId, onDecision]);

  const handleCustomSubmit = useCallback(() => {
    if (!customText.trim()) return;
    setDecided('custom');
    onDecision?.(cardId, 'custom', customText.trim());
  }, [cardId, customText, onDecision]);

  // 已決策 → 顯示結果摘要
  if (decided) {
    const labels = { agree: agreeLabel, disagree: disagreeLabel, custom: t('review.customLabel') };
    return (
      <div className="decision-card decided" data-testid="decision-card-decided">
        <span className="decision-result">✓ {title}：{labels[decided]}</span>
        {decided === 'custom' && <span className="decision-custom-echo">「{customText}」</span>}
      </div>
    );
  }

  return (
    <div className="decision-card" data-testid="decision-card">
      <div className="decision-header">
        <strong>{title}</strong>
      </div>

      {description && <p className="decision-desc">{description}</p>}

      {preview && (
        <div className="decision-preview">
          <button
            className="btn-preview-toggle"
            onClick={() => setPreviewExpanded(v => !v)}
          >
            {previewExpanded ? t('review.previewCollapse') : t('review.previewExpand')}
          </button>
          {previewExpanded && (
            <pre className="decision-preview-content">{preview}</pre>
          )}
        </div>
      )}

      <div className="decision-actions">
        <button className="btn-agree" data-testid="decision-agree" onClick={handleAgree}>
          {agreeLabel}
          {agreeReason && <span className="action-hint">（{agreeReason}）</span>}
        </button>

        <button className="btn-disagree" data-testid="decision-disagree" onClick={handleDisagree}>
          {disagreeLabel}
          {disagreeConsequence && <span className="action-hint">（{disagreeConsequence}）</span>}
        </button>

        <button
          className="btn-custom"
          data-testid="decision-custom-toggle"
          onClick={() => setShowCustom(v => !v)}
        >
          {t('review.customInput')}
        </button>
      </div>

      {showCustom && (
        <div className="decision-custom-input" data-testid="decision-custom-input">
          <textarea
            value={customText}
            onChange={e => setCustomText(e.target.value)}
            placeholder={t('review.customPlaceholder')}
            rows={3}
          />
          <button
            className="btn-custom-submit"
            onClick={handleCustomSubmit}
            disabled={!customText.trim()}
          >
            {t('review.submit')}
          </button>
        </div>
      )}
    </div>
  );
}
