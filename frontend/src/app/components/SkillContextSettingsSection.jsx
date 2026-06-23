import useI18n from '../../locales/useI18n';

// #I-207: Settings 頁 Skill Context 靜態說明區塊
export default function SkillContextSettingsSection() {
  const t = useI18n(s => s.t);
  return (
    <div className="skill-settings-section">
      <h4>Skill Context Orchestration</h4>
      <p>{t('skill.settingsDesc')}</p>
      <ul>
        <li>{t("skill.autoInjectHint")}</li>
        <li>{t("skill.multiCandidateHint")}</li>
        <li>{t("skill.highRiskHint")}</li>
        <li>{t("skill.auditHint")}</li>
      </ul>
    </div>
  );
}
