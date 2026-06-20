import {isExclusiveCandidateSet} from '../../lib/appHelpers';

/**
 * FloatingCandidateActions — 浮動意圖候選按鈕
 *
 * §12A.1: 當 Readiness Gate 偵測到模糊使用者意圖時，
 * 在輸入框正上方顯示最多 3 個候選按鈕。
 * 使用者點擊後填入 draft 並觸發重新評估，送出新訊息後全部消失。
 *
 * Props:
 *   candidates — 候選陣列 [{id, label, draft}]
 *   selectedIDs — 已選候選 id
 *   onSelect    — 點擊候選回呼 (candidateID) => void
 */
export default function FloatingCandidateActions({candidates = [], selectedIDs = [], onSelect}) {
  if (!candidates.length) return null;
  const selectedSet = new Set(selectedIDs);
  const selectionMode = isExclusiveCandidateSet(candidates) ? 'single' : 'multiple';
  return (
    <div className="floating-candidates" data-selection-mode={selectionMode}>
      {candidates.slice(0, 3).map((candidate) => (
        <button
          key={candidate.id}
          type="button"
          className={`floating-candidate-btn ${selectedSet.has(candidate.id) ? 'is-selected' : ''}`}
          aria-pressed={selectedSet.has(candidate.id)}
          onClick={() => onSelect(candidate.id)}
        >
          {candidate.label}
        </button>
      ))}
    </div>
  );
}
