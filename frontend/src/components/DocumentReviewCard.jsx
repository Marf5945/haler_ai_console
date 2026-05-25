// DocumentReviewCard.jsx — 文件寫入/匯出確認卡片。
import { useState, useEffect, useCallback } from 'react';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import DecisionCard from './DecisionCard';
import useI18n from '../locales/useI18n';

// 監聽 document:review_needed event，顯示 DecisionCard 供使用者確認。
export default function DocumentReviewCard({ onToast }) {
  const t = useI18n(s => s.t);
  const [pending, setPending] = useState([]);

  useEffect(() => {
    const off = EventsOn('document:review_needed', (data) => {
      setPending(prev => [...prev, data]);
    });
    return () => { off(); };
  }, []);

  const handleDecision = useCallback((cardId, decision) => {
    if (decision === 'agree') {
      onToast?.(t('confirm.savedConfirm', { id: cardId }));
    } else if (decision === 'disagree') {
      onToast?.(t('confirm.savedCancel'));
    }
    setPending(prev => prev.filter(p => p.doc_id !== cardId));
  }, [onToast]);

  if (pending.length === 0) return null;

  return (
    <div className="document-review-cards" data-testid="document-review-cards">
      {pending.map(doc => (
        <DecisionCard
          key={doc.doc_id}
          cardId={doc.doc_id}
          title={t('confirm.docWriteTitle', { name: doc.display_name })}
          description={t('confirm.docWriteDesc', { format: doc.format, count: doc.word_count })}
          preview={doc.preview}
          agreeLabel={t('confirm.confirmSave')}
          disagreeLabel={t('common.cancel')}
          agreeReason={t('confirm.saveReason')}
          disagreeConsequence={t('confirm.discardConsequence')}
          onDecision={handleDecision}
        />
      ))}
    </div>
  );
}
