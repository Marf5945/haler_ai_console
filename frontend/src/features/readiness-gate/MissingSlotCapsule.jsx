import useI18n from '../../locales/useI18n';

/**
 * MissingSlotCapsule — 缺欄位膠囊提示
 *
 * §12A.3: 當必填欄位仍缺時，以精簡膠囊顯示缺少項目。
 * 每次使用者送出訊息後更新，所有欄位補齊後消失。
 * 高風險任務的膠囊帶有警告色（.slot-warning）。
 *
 * Props:
 *   missingSlots — 缺少欄位名稱陣列 ['資料夾', '檔案類型', '時間條件']
 *   isHighRisk   — 是否為高風險任務（影響膠囊顏色）
 */
export default function MissingSlotCapsule({missingSlots = [], isHighRisk = false}) {
  const {t} = useI18n();
  if (!missingSlots.length) return null;
  return (
    <span className={`missing-slot-capsule ${isHighRisk ? 'slot-warning' : ''}`}>
      <span className="slot-label">{t('slot.missing')}</span>
      <span className="slot-items">{missingSlots.join('、')}</span>
    </span>
  );
}
