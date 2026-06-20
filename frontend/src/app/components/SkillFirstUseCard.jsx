import useI18n from '../../locales/useI18n';

// #I-207: 初次使用說明卡 — 第一次 skill 注入成功後一次性顯示
/* i18n: skill first use card */
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
