import useI18n from '../../locales/useI18n';

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
