import {useState} from 'react';
import useI18n from '../../locales/useI18n';
import {callWails} from '../../lib/callWails';
import {
  SelectProjectBackupExportDirectory,
  ExportProjectBackupHandler,
  SelectProjectBackupFile,
  IsProjectBackupEncryptedHandler,
  ImportProjectBackupHandler,
} from '../../../wailsjs/go/main/App';

export default function ProjectManagePopup({ view, manifests, confirmStep, onPurgeProject, onPurgeBoundary, onViewManifests, onBackToMenu, onClose }) {
  const {t} = useI18n();
  // i18n: project
  // 對話備份／還原（SEC：加密為「可選」，勾了才問密碼）
  const [backupDialog, setBackupDialog] = useState(null); // {kind:'export'|'import-password'|'import-conflict', ...}
  const [backupEncrypt, setBackupEncrypt] = useState(false);
  const [backupRedact, setBackupRedact] = useState(false);
  const [backupPassword, setBackupPassword] = useState('');
  const [backupPassword2, setBackupPassword2] = useState('');
  const [backupBusy, setBackupBusy] = useState(false);
  const [backupNotice, setBackupNotice] = useState('');

  function resetBackupForm() {
    setBackupEncrypt(false);
    setBackupRedact(false);
    setBackupPassword('');
    setBackupPassword2('');
  }

  async function runBackupExport() {
    if (backupEncrypt) {
      if ((backupPassword || '').length < 8) { setBackupNotice('密碼至少 8 個字元'); return; }
      if (backupPassword !== backupPassword2) { setBackupNotice('兩次密碼不一致'); return; }
    }
    setBackupBusy(true);
    setBackupNotice('');
    try {
      const destDir = await callWails(SelectProjectBackupExportDirectory);
      if (!destDir) { setBackupBusy(false); return; }
      const res = await callWails(() => ExportProjectBackupHandler(
        'default', destDir, backupEncrypt ? backupPassword : '', backupRedact, backupEncrypt));
      if (res?.status === 'ok') {
        setBackupDialog(null);
        resetBackupForm();
        setBackupNotice(`已匯出：${res.bundle_path}`);
      } else {
        setBackupNotice(res?.message || '匯出失敗');
      }
    } catch (err) {
      setBackupNotice(String(err?.message || err));
    }
    setBackupBusy(false);
  }

  async function startBackupImport() {
    setBackupNotice('');
    setBackupBusy(true);
    try {
      const path = await callWails(SelectProjectBackupFile);
      if (!path) { setBackupNotice(t('project.backupImportCancelled')); setBackupBusy(false); return; }
      const probe = await callWails(() => IsProjectBackupEncryptedHandler(path));
      if (probe?.status !== 'ok') { setBackupNotice(probe?.message || t('project.backupReadFail')); setBackupBusy(false); return; }
      if (probe.encrypted) {
        setBackupPassword('');
        setBackupBusy(false);
        setBackupDialog({kind: 'import-password', path});
        return;
      }
      setBackupBusy(false);
      await finishBackupImport(path, '', 'fail_if_exists');
    } catch (err) {
      setBackupNotice(String(err?.message || err));
      setBackupBusy(false);
    }
  }

  async function finishBackupImport(path, password, mode) {
    setBackupBusy(true);
    try {
      const res = await callWails(() => ImportProjectBackupHandler(path, password, mode));
      if (res?.status === 'ok') {
        setBackupDialog(null);
        resetBackupForm();
        setBackupNotice(`已還原專案：${res.restored_as}（重新整理或重啟 App 後生效）`);
      } else if (res?.status === 'conflict') {
        setBackupDialog({kind: 'import-conflict', path, password});
      } else {
        setBackupNotice(res?.message || '匯入失敗');
        if (res?.status !== 'bad_password') setBackupDialog(null);
      }
    } catch (err) {
      setBackupNotice(String(err?.message || err));
      setBackupDialog(null);
    }
    setBackupBusy(false);
  }
  return (
    <>
    <div className="project-manage-overlay" role="dialog" aria-modal="true" aria-label={t('project.manageTitle')}>
      <div className="project-manage-popup">
        <div className="project-manage-header">
          <h3>{view === 'manifests' ? t('project.manifestsTitle') : t('project.manageTitle')}</h3>
          <button type="button" className="project-manage-close" onClick={onClose} aria-label={t('common.close')}>×</button>
        </div>
        {view === 'menu' ? (
          <div className="project-manage-actions">
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
            <button type="button" className="project-manage-btn" onClick={() => { setBackupNotice(''); resetBackupForm(); setBackupDialog({kind: 'export'}); }}>
              <span className="project-manage-icon">⇪</span>
              <span>{t('project.exportWholeProject')}</span>
            </button>
            <button type="button" className="project-manage-btn" disabled={backupBusy} onClick={startBackupImport}>
              <span className="project-manage-icon">⇩</span>
              <span>{t('project.importWholeProject')}</span>
            </button>
            {backupNotice && <small className="project-manage-note">{backupNotice}</small>}
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
      {backupDialog?.kind === 'export' && (
        <div className="export-dialog-overlay">
          <div className="export-dialog">
            <p>{t('project.backupExportTitle')}</p>
            <label style={{display: 'flex', gap: '6px', alignItems: 'center'}}>
              <input type="checkbox" checked={backupEncrypt}
                onChange={(e) => { setBackupEncrypt(e.target.checked); setBackupNotice(''); }} />
              <span>{t('project.backupEncryptFile')}</span>
            </label>
            {backupEncrypt ? (
              <>
                <input type="password" placeholder={t('project.backupPwdSet')} value={backupPassword}
                  onChange={(e) => setBackupPassword(e.target.value)} />
                <input type="password" placeholder={t('project.backupPwdRepeat')} value={backupPassword2}
                  onChange={(e) => setBackupPassword2(e.target.value)} />
                <small>{t('project.backupPwdWarn')}</small>
              </>
            ) : (
              <small>{t('project.backupPlainWarn')}</small>
            )}
            <label style={{display: 'flex', gap: '6px', alignItems: 'center'}}>
              <input type="checkbox" checked={backupRedact}
                onChange={(e) => setBackupRedact(e.target.checked)} />
              <span>{t('project.backupRedact')}</span>
            </label>
            {backupNotice && <small>{backupNotice}</small>}
            <div className="export-dialog-actions">
              <button type="button" onClick={() => { setBackupDialog(null); setBackupNotice(''); }}>{t('common.cancel')}</button>
              <button type="button" disabled={backupBusy} onClick={runBackupExport}>{t('project.backupChooseExport')}</button>
            </div>
          </div>
        </div>
      )}
      {backupDialog?.kind === 'import-password' && (
        <div className="export-dialog-overlay">
          <div className="export-dialog">
            <p>{t('project.backupEncryptedPrompt')}</p>
            <input type="password" placeholder={t('project.backupPwdInput')} value={backupPassword}
              onChange={(e) => setBackupPassword(e.target.value)} />
            {backupNotice && <small>{backupNotice}</small>}
            <div className="export-dialog-actions">
              <button type="button" onClick={() => { setBackupDialog(null); setBackupNotice(''); }}>{t('common.cancel')}</button>
              <button type="button" disabled={backupBusy}
                onClick={() => finishBackupImport(backupDialog.path, backupPassword, 'fail_if_exists')}>{t('project.backupRestore')}</button>
            </div>
          </div>
        </div>
      )}
      {backupDialog?.kind === 'import-conflict' && (
        <div className="export-dialog-overlay">
          <div className="export-dialog">
            <p>{t('project.backupConflictPrompt')}</p>
            <div className="export-dialog-actions">
              <button type="button" onClick={() => { setBackupDialog(null); setBackupNotice(''); }}>{t('common.cancel')}</button>
              <button type="button" disabled={backupBusy}
                onClick={() => finishBackupImport(backupDialog.path, backupDialog.password, 'copy')}>{t('project.backupSaveCopy')}</button>
              <button type="button" disabled={backupBusy}
                onClick={() => finishBackupImport(backupDialog.path, backupDialog.password, 'overwrite')}>{t('project.backupOverwrite')}</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
