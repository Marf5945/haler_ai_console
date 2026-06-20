import useI18n from '../../locales/useI18n';

export default function PackageInstallDecisionDialog({candidate, onConfirm, onCancel}) {
  const {t} = useI18n();
  const typeLabel = candidate?.typeLabel || candidate?.type || 'package';
  const name = candidate?.name || t('package.unnamedItem');
  const preview = candidate?.preview || {};
  const detail = candidate?.type === 'subagent'
    ? t('package.systemCodeDesc', { code: preview.source_system_code || 'unknown', toolCount: preview.tool_count || 0 })
    : candidate?.type === 'skill'
      ? t('package.skillIdDesc', { skillId: preview.SkillID || preview.skill_id || 'unknown', resourceCount: (preview.Resources || preview.resources || []).length })
      : candidate?.type === 'goprogram'
        ? `小程式 ${preview.program_id || preview.program_name || 'unknown'}　→ 工具 > 自動流程`
        : t('package.sourceFrom', { path: candidate?.path || 'unknown' });
  // i18n: package
  return (
    <div className="export-dialog-overlay">
      <div className="export-dialog sub-import-dialog">
        <p>{t('package.installQuestion', { type: typeLabel })}</p>
        <strong>{name}</strong>
        <small>{detail}</small>
        <div className="export-dialog-actions">
          <button type="button" onClick={onConfirm}>{t('package.installBtn')}</button>
          <button type="button" onClick={onCancel}>{t('common.cancel')}</button>
        </div>
      </div>
    </div>
  );
}
