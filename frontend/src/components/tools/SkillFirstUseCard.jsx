import useI18n from '../../locales/useI18n';

export default function SkillFirstUseCard({onDismiss}) {
  const t = useI18n(s => s.t);
  return (
    <div className="skill-first-use-card" role="alert" aria-label={t('skill.firstUseLabel')}>
      <div className="skill-first-use-header">
        <span>{t('skill.firstUseTitle')}</span>
        <button type="button" className="skill-first-use-close" onClick={onDismiss} aria-label={t('common.close')}>✕</button>
      </div>
      <p>{t('skill.desc1')}</p>
      <p>{t('skill.desc2')}</p>
      <p>{t('skill.desc3')}</p>
      <small>{t('skill.desc4')}</small>
    </div>
  );
}
