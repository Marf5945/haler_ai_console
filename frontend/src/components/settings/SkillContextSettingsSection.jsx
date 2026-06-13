import useI18n from '../../locales/useI18n';

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
