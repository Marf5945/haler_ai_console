import {useState} from 'react';
import useI18n from '../../locales/useI18n';
import SettingPopupSelect from './SettingPopupSelect';
import {
  _fontPresetLabelMap,
  _panelLangLabelMap,
  _roleLangLabelMap,
  cycleValue,
  fontScaleOptions,
  loadStyleOptionOrder,
  localizeBackendLabel,
  normalizePanelStyle,
  reorderItems,
  saveStyleOptionOrder,
  styleLabel,
  voiceLanguageModeLabel,
  voiceStatusLabel,
} from '../../lib/panelSettings';

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
          options={[t('settings.fontDefault'), t('settings.fontNormal'), t('settings.fontHand'), t('settings.fontCalli'), t('settings.fontRound'), t('settings.fontMono')]}
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
        <button className="settings-restore-btn" type="button" onClick={onRestoreDefaults}>
          <span>↺</span>
          <span>{t('settings.resetDefault')}</span>
        </button>
        <small className="settings-restore-note">{t('settings.resetHint')}</small>
      </div>
    </section>
  );
}
