import {useState} from 'react';
import useI18n from '../../locales/useI18n';

export default function PackageConfirmModal({pending, packageData, error, onConfirm, onReject}) {
  const t = useI18n(s => s.t);
  const [detailsExpanded, setDetailsExpanded] = useState(false);
  const manifest = pending?.manifest || {};
  const name = manifest.name || packageData?.name || t('package.unknownPackage');
  const riskTag = manifest.risk_tag || 'unknown';
  const riskLabelMap = {low: t('package.riskLow'), medium: t('package.riskMedium'), high: t('package.riskHigh'), unknown: t('package.riskUnknown')};
  const riskColorClass = {low: 'pkg-risk-low', medium: 'pkg-risk-medium', high: 'pkg-risk-high', unknown: 'pkg-risk-unknown'}[riskTag] || 'pkg-risk-unknown';

  // 權限宣告：優先用新欄位 declared_permissions，向下相容 required_perms
  const perms = manifest.declared_permissions?.length ? manifest.declared_permissions
    : manifest.required_perms?.length ? manifest.required_perms : [];

  // 預計寫入位置
  const writeTargets = manifest.write_targets || [];

  return (
    <div className="pkg-modal-overlay" role="dialog" aria-modal="true" aria-label={t("package.installConfirmLabel")}>
      <div className="pkg-modal">
        <div className="pkg-modal-header">
          <span aria-hidden="true">📦</span>
          <span>{t("package.installConfirmTitle")}</span>
        </div>

        {/* ── 上半區：始終可見的四項關鍵資訊 ── */}
        <div className="pkg-modal-body">
          <div className="pkg-modal-name">{name}</div>
          <div className={`pkg-modal-risk ${riskColorClass}`}>
            {t('package.riskLevel', { level: riskLabelMap[riskTag] || riskTag })}
          </div>
          {perms.length > 0 && (
            <div className="pkg-modal-perms">
              <span>{t("package.permissions")}</span>
              {perms.join('、')}
            </div>
          )}
          {writeTargets.length > 0 && (
            <div className="pkg-modal-write-targets">
              <span>{t("package.writeLocation")}</span>
              {writeTargets.join('、')}
            </div>
          )}

          <p className="pkg-modal-notice">
            {t("package.quarantineHint")}
          </p>

          {/* ── 下半區：展開工程細節（收摺） ── */}
          <button
            className="pkg-modal-details-toggle"
            type="button"
            onClick={() => setDetailsExpanded(!detailsExpanded)}
            aria-expanded={detailsExpanded}
          >
            {detailsExpanded ? t('package.collapseDetails') : t('package.expandDetails')}
          </button>

          {detailsExpanded && (
            <div className="pkg-modal-details">
              {manifest.source_metadata && (
                <div className="pkg-detail-row">
                  <span className="pkg-detail-label">{t("package.source")}</span>
                  <span className="pkg-detail-value">{manifest.source_metadata || manifest.source_path}</span>
                </div>
              )}
              {manifest.package_hash && (
                <div className="pkg-detail-row">
                  <span className="pkg-detail-label">Package Hash：</span>
                  <span className="pkg-detail-value pkg-hash">{manifest.package_hash}</span>
                </div>
              )}
              {manifest.manifest_hash && (
                <div className="pkg-detail-row">
                  <span className="pkg-detail-label">Manifest Hash：</span>
                  <span className="pkg-detail-value pkg-hash">{manifest.manifest_hash}</span>
                </div>
              )}
              <div className="pkg-detail-row">
                <span className="pkg-detail-label">{t("package.newItems")}</span>
                <span className="pkg-detail-value">
                  {[
                    manifest.adds_tools && t('package.addTools'),
                    manifest.adds_skills && 'Skill',
                    manifest.adds_mcp_server && 'MCP Server',
                    manifest.adds_adapter && 'Adapter',
                  ].filter(Boolean).join(', ') || t('package.none')}
                </span>
              </div>
              <div className="pkg-detail-row">
                <span className="pkg-detail-label">{t("package.affectsRouting")}</span>
                <span className="pkg-detail-value">{manifest.affects_routing ? t('common.yes') : t('common.no')}</span>
              </div>
              {manifest.declared_entry_points?.length > 0 && (
                <div className="pkg-detail-row">
                  <span className="pkg-detail-label">Entry Points：</span>
                  <span className="pkg-detail-value">{manifest.declared_entry_points.join('、')}</span>
                </div>
              )}
            </div>
          )}

          {pending?.validation_issues?.length > 0 && (
            <div className="pkg-modal-review">
              {pending.validation_issues.map((issue) => <span key={issue}>{issue}</span>)}
            </div>
          )}
          {error && <div className="pkg-modal-error">{error}</div>}
        </div>
        <div className="pkg-modal-actions">
          <button className="pkg-modal-cancel" type="button" onClick={onReject}>{t('common.cancel')}</button>
          <button className="pkg-modal-confirm" type="button" onClick={onConfirm}>{t('package.confirmInstall')}</button>
        </div>
      </div>
    </div>
  );
}
