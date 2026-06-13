import {useState} from 'react';
import {ExportProjectBackupHandler, ImportProjectBackupHandler, IsProjectBackupEncryptedHandler, SelectProjectBackupExportDirectory, SelectProjectBackupFile} from '../../../wailsjs/go/main/App';
import useI18n from '../../locales/useI18n';
import {_fontPresetLabelMap, _panelLangLabelMap, _roleLangLabelMap, callWails, cycleValue, fontScaleOptions, loadStyleOptionOrder, localizeBackendLabel, normalizePanelStyle, reorderItems, saveStyleOptionOrder, styleLabel, voiceLanguageModeLabel, voiceStatusLabel} from '../../lib/appShared';
import SettingPopupSelect from './SettingPopupSelect';

export default function SettingsMenu({
  panel, onPanelChange, voiceState, voiceInstallBusy,
  onVoiceSettingsChange, onVoiceSettingsRefresh, onVoiceModelInstall, onVoiceModelRemove, onRestoreDefaults,
}) {
  const t = useI18n(s => s.t);
  const activePanelStyle = normalizePanelStyle(panel.panelStyle);
  const panelLanguageValue = localizeBackendLabel(panel.panelLanguage, _panelLangLabelMap);
  const roleLanguageValue = localizeBackendLabel(panel.roleLanguage, _roleLangLabelMap);
  const fontPresetValue = localizeBackendLabel(panel.fontPreset, _fontPresetLabelMap);
  const [orderedStyleOptions, setOrderedStyleOptions] = useState(loadStyleOptionOrder);
  const [draggedStyleOption, setDraggedStyleOption] = useState('');

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
    try {
      const path = await callWails(SelectProjectBackupFile);
      if (!path) return;
      const probe = await callWails(() => IsProjectBackupEncryptedHandler(path));
      if (probe?.status !== 'ok') { setBackupNotice(probe?.message || '無法讀取備份檔'); return; }
      if (probe.encrypted) {
        setBackupPassword('');
        setBackupDialog({kind: 'import-password', path});
        return;
      }
      await finishBackupImport(path, '', 'fail_if_exists');
    } catch (err) {
      setBackupNotice(String(err?.message || err));
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
  const voiceSettings = voiceState?.settings || {languageMode: 'auto', manualLanguage: '', debugMode: false, commandMode: false};

  function moveStyleOption(targetOption) {
    if (!draggedStyleOption || draggedStyleOption === targetOption) return;
    setOrderedStyleOptions((current) => {
      const next = reorderItems(current, draggedStyleOption, targetOption);
      saveStyleOptionOrder(next);
      return next;
    });
    setDraggedStyleOption('');
  }

  return (
    <section className="settings-side-panel" aria-label={t('settings.panelSettings')}>
      <h2>{t('settings.panelSettings')}</h2>
      <div className="settings-menu-list">
        <SettingPopupSelect
          icon="◎"
          label={t('settings.panelLanguage')}
          value={panelLanguageValue}
          options={[t('settings.langZhTW'), t('settings.langEn'), t('settings.langJa'), t('settings.langPt'), t('settings.langEs'), t('settings.langTh')]}
          onSelect={(lang) => onPanelChange({panelLanguage: lang})}
        />
        <SettingPopupSelect
          icon="♙"
          label={t('settings.roleLanguage')}
          value={roleLanguageValue}
          options={[t('settings.roleLangAuto'), t('settings.langZhTW'), t('settings.langEn'), t('settings.langJa'), t('settings.langPt'), t('settings.langEs'), t('settings.langTh')]}
          onSelect={(lang) => onPanelChange({roleLanguage: lang})}
        />
        <SettingPopupSelect
          icon="Aa"
          label={t('settings.fontPreset')}
          value={fontPresetValue}
          options={[t('settings.fontDefault'), t('settings.fontRound'), t('settings.fontMono')]}
          onSelect={(preset) => onPanelChange({fontPreset: preset})}
        />
        <SettingPopupSelect
          icon="Tt"
          label={t('settings.fontSize')}
          value={panel.fontScale}
          options={fontScaleOptions}
          onSelect={(scale) => onPanelChange({fontScale: scale})}
        />
      </div>
      <div className="settings-style-block">
        <div className="settings-style-label"><span>◌</span><span>{t('settings.panelStyle')}</span></div>
        <div className="settings-segments">
          {orderedStyleOptions.map((option) => (
            <button
              className={activePanelStyle === option ? 'segment-active' : ''}
              draggable
              type="button"
              key={option}
              onDragEnd={() => setDraggedStyleOption('')}
              onDragOver={(event) => event.preventDefault()}
              onDragStart={() => setDraggedStyleOption(option)}
              onDrop={() => moveStyleOption(option)}
              onClick={() => onPanelChange({panelStyle: option})}
            >
              {styleLabel(option)}
            </button>
          ))}
        </div>
      </div>
      <div className="settings-voice-block">
        <div className="settings-voice-head">
          <span>{t('settings.localVoice')}</span>
          <button type="button" onClick={onVoiceSettingsRefresh}>{t('settings.voiceCheck')}</button>
        </div>
        <div className="settings-voice-state">
          <span className={`settings-voice-dot ${voiceState?.status === 'ready' ? 'settings-voice-ready' : ''}`} />
          <strong>{voiceStatusLabel(voiceState?.status)}</strong>
          <small>{voiceState?.language ? t('settings.voiceLang', { language: voiceState.language }) : t('settings.voiceFollowApp')}</small>
        </div>
        <div className="settings-voice-controls">
          <button
            type="button"
            className={voiceSettings.languageMode === 'auto' ? 'settings-voice-toggle settings-voice-toggle-on' : 'settings-voice-toggle'}
            onClick={() => onVoiceSettingsChange?.({languageMode: cycleValue(['auto', 'follow_app', 'manual'], voiceSettings.languageMode || 'auto')})}
          >
            {voiceLanguageModeLabel(voiceSettings.languageMode)}
          </button>
          <button
            type="button"
            className="settings-voice-toggle"
            disabled={voiceInstallBusy}
            onClick={voiceState?.modelAvailable ? onVoiceModelRemove : onVoiceModelInstall}
          >
            {voiceInstallBusy ? t('settings.voiceProcessing') : voiceState?.modelAvailable ? t('settings.voiceRemoveModel') : t('settings.voiceInstallModel')}
          </button>
          <button
            type="button"
            className={voiceSettings.debugMode ? 'settings-voice-toggle settings-voice-toggle-on' : 'settings-voice-toggle'}
            onClick={() => onVoiceSettingsChange?.({debugMode: !voiceSettings.debugMode})}
          >
            debug {voiceSettings.debugMode ? 'on' : 'off'}
          </button>
          <button
            type="button"
            className={voiceSettings.commandMode ? 'settings-voice-toggle settings-voice-toggle-on' : 'settings-voice-toggle'}
            onClick={() => onVoiceSettingsChange?.({commandMode: !voiceSettings.commandMode})}
          >
            {t('settings.command')} {voiceSettings.commandMode ? 'on' : 'off'}
          </button>
        </div>
      </div>
      <div className="settings-restore-block">
        <button className="settings-restore-btn" type="button"
          onClick={() => { setBackupNotice(''); resetBackupForm(); setBackupDialog({kind: 'export'}); }}>
          <span>⇪</span>
          <span>匯出對話備份</span>
        </button>
        <button className="settings-restore-btn" type="button" disabled={backupBusy} onClick={startBackupImport}>
          <span>⇩</span>
          <span>匯入對話備份</span>
        </button>
        {backupNotice && <small className="settings-restore-note">{backupNotice}</small>}
      </div>
      {backupDialog?.kind === 'export' && (
        <div className="export-dialog-overlay">
          <div className="export-dialog">
            <p>匯出對話備份</p>
            <label style={{display: 'flex', gap: '6px', alignItems: 'center'}}>
              <input type="checkbox" checked={backupEncrypt}
                onChange={(e) => { setBackupEncrypt(e.target.checked); setBackupNotice(''); }} />
              <span>加密備份檔</span>
            </label>
            {backupEncrypt ? (
              <>
                <input type="password" placeholder="設定密碼（至少 8 字元）" value={backupPassword}
                  onChange={(e) => setBackupPassword(e.target.value)} />
                <input type="password" placeholder="再輸入一次密碼" value={backupPassword2}
                  onChange={(e) => setBackupPassword2(e.target.value)} />
                <small>忘記密碼將無法開啟備份，請妥善保管。</small>
              </>
            ) : (
              <small>未加密：任何拿到檔案的人都能讀取，請勿放公開位置。</small>
            )}
            <label style={{display: 'flex', gap: '6px', alignItems: 'center'}}>
              <input type="checkbox" checked={backupRedact}
                onChange={(e) => setBackupRedact(e.target.checked)} />
              <span>遮蔽敏感資料（金鑰等）</span>
            </label>
            {backupNotice && <small>{backupNotice}</small>}
            <div className="export-dialog-actions">
              <button type="button" onClick={() => { setBackupDialog(null); setBackupNotice(''); }}>取消</button>
              <button type="button" disabled={backupBusy} onClick={runBackupExport}>選擇位置並匯出</button>
            </div>
          </div>
        </div>
      )}
      {backupDialog?.kind === 'import-password' && (
        <div className="export-dialog-overlay">
          <div className="export-dialog">
            <p>這是加密備份檔，請輸入密碼</p>
            <input type="password" placeholder="備份密碼" value={backupPassword}
              onChange={(e) => setBackupPassword(e.target.value)} />
            {backupNotice && <small>{backupNotice}</small>}
            <div className="export-dialog-actions">
              <button type="button" onClick={() => { setBackupDialog(null); setBackupNotice(''); }}>取消</button>
              <button type="button" disabled={backupBusy}
                onClick={() => finishBackupImport(backupDialog.path, backupPassword, 'fail_if_exists')}>還原</button>
            </div>
          </div>
        </div>
      )}
      {backupDialog?.kind === 'import-conflict' && (
        <div className="export-dialog-overlay">
          <div className="export-dialog">
            <p>同名專案已存在，要怎麼處理？</p>
            <div className="export-dialog-actions">
              <button type="button" onClick={() => { setBackupDialog(null); setBackupNotice(''); }}>取消</button>
              <button type="button" disabled={backupBusy}
                onClick={() => finishBackupImport(backupDialog.path, backupDialog.password, 'copy')}>另存副本</button>
              <button type="button" disabled={backupBusy}
                onClick={() => finishBackupImport(backupDialog.path, backupDialog.password, 'overwrite')}>覆蓋現有</button>
            </div>
          </div>
        </div>
      )}
      <div className="settings-restore-block">
        <button className="settings-restore-btn" type="button" onClick={onRestoreDefaults}>
          <span>↺</span>
          <span>{t('settings.resetDefault')}</span>
        </button>
        <small className="settings-restore-note">{t('settings.resetHint')}</small>
      </div>
    </section>
  );
}
