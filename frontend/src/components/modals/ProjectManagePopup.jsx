import {useState} from 'react';
import {ExportProjectBackupHandler, SelectProjectBackupExportDirectory} from '../../../wailsjs/go/main/App';
import useI18n from '../../locales/useI18n';
import {callWails} from '../../lib/appShared';

export default function ProjectManagePopup({ view, manifests, confirmStep, onPurgeProject, onPurgeBoundary, onViewManifests, onBackToMenu, onClose }) {
  const {t} = useI18n();
  const [backupBusy, setBackupBusy] = useState(false);
  const [backupNotice, setBackupNotice] = useState('');

  async function handleBackupExport() {
    setBackupBusy(true);
    setBackupNotice('');
    try {
      // 與 SettingsMenu 的匯出流程一致：先選存放位置，再呼叫 handler。
      // destDir 不可為空——原版 handler 會以「請先選擇存放位置」拒絕空值。
      const destDir = await callWails(SelectProjectBackupExportDirectory);
      if (!destDir) { setBackupBusy(false); return; } // 使用者取消選擇
      const res = await callWails(() => ExportProjectBackupHandler('default', destDir, '', false, false));
      setBackupNotice(res?.message || '備份匯出已送出');
    } catch (err) {
      setBackupNotice(String(err?.message || err));
    } finally {
      setBackupBusy(false);
    }
  }

  // i18n: project
  return (
    <div className="project-manage-overlay" role="dialog" aria-modal="true" aria-label={t('project.manageTitle')}>
      <div className="project-manage-popup">
        <div className="project-manage-header">
          <h3>{view === 'manifests' ? t('project.manifestsTitle') : t('project.manageTitle')}</h3>
          <button type="button" className="project-manage-close" onClick={onClose} aria-label={t('common.close')}>×</button>
        </div>
        {view === 'menu' ? (
          <div className="project-manage-actions">
            <button type="button" className="project-manage-btn" disabled={backupBusy} onClick={handleBackupExport}>
              <span className="project-manage-icon">⇪</span>
              <span>{backupBusy ? '正在檢查備份功能' : '匯出對話備份'}</span>
            </button>
            {backupNotice && <p className="project-manage-empty">{backupNotice}</p>}
            <button
              type="button"
              className={`project-manage-btn ${confirmStep === 'project' ? 'project-manage-btn-danger' : ''}`}
              onClick={onPurgeProject}
            >
              <span className="project-manage-icon">⚠</span>
              <span>{confirmStep === 'project' ? t('project.confirmPurge') : t('project.purgeProjectData')}</span>
            </button>
            <button
              type="button"
              className={`project-manage-btn ${confirmStep === 'boundary' ? 'project-manage-btn-danger' : ''}`}
              onClick={onPurgeBoundary}
            >
              <span className="project-manage-icon">⚠</span>
              <span>{confirmStep === 'boundary' ? t('project.confirmPurge') : t('project.purgeBoundaryData')}</span>
            </button>
            <button type="button" className="project-manage-btn" onClick={onViewManifests}>
              <span className="project-manage-icon">☰</span>
              <span>{t('project.viewManifests')}</span>
            </button>
          </div>
        ) : (
          <div className="project-manage-manifests">
            <button type="button" className="project-manage-back" onClick={onBackToMenu}>← {t('common.back')}</button>
            {manifests.length === 0 ? (
              <p className="project-manage-empty">{t('project.noManifests')}</p>
            ) : (
              <ul className="project-manage-list">
                {manifests.map((m, i) => (
                  <li key={m.id || i} className="project-manage-list-item">
                    <span className="manifest-time">{m.timestamp || m.created_at || '—'}</span>
                    <span className="manifest-type">{m.trigger || m.purge_type || t('project.manualPurge')}</span>
                    <span className="manifest-count">{t('project.fileCount', { count: m.file_count || m.files?.length || 0 })}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
