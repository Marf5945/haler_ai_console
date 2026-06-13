import {useState} from 'react';
import useI18n from '../../locales/useI18n';
import {styleLabel, voiceStatusLabel} from '../../lib/appShared';

export default function BrowserSettingsSection({
  browserPref, panelSettings, adapterList, summaryModelSettings, summaryModelScan,
  onSummaryModelChange, onSummaryModelRescan, voiceState, voiceInstallBusy,
  onVoiceSettingsChange, onVoiceSettingsRefresh, onVoiceModelInstall, onVoiceModelRemove,
  onVoiceDebugClear, onSaveBrowserPref, onPreviewStyleDiff,
}) {
  const t = useI18n(s => s.t);
  const browserOptions = ['chrome', 'safari', 'firefox', 'edge', 'arc'];
  const [diffInput, setDiffInput] = useState('');
  const summarySettings = summaryModelSettings || {source: 'cli_adapter', modelId: 'current', endpoint: '', alwaysUse: false};
  const voiceSettings = voiceState?.settings || {languageMode: 'auto', manualLanguage: '', debugMode: false, commandMode: false, whisperBinPath: '', modelPath: ''};
  const localOptions = summaryModelScan?.options || [];

  return (
    <div className="browser-settings-section">
      <h4>{t('settings.browserUISettings')}</h4>

      {/* Browser Preference Selector */}
      <div className="browser-pref-row">
        <label htmlFor="browser-select">{t('settings.defaultBrowser')}</label>
        <select
          id="browser-select"
          value={browserPref?.id || ''}
          onChange={(e) => onSaveBrowserPref(e.target.value)}
        >
          <option value="" disabled>{t('settings.selectBrowser')}</option>
          {browserOptions.map((b) => (
            <option key={b} value={b}>{b.charAt(0).toUpperCase() + b.slice(1)}</option>
          ))}
        </select>
      </div>

      <div className="ui-settings-info">
        <span className="ui-settings-label">{t('settings.currentPanel')}</span>
        <span className="ui-settings-value">{styleLabel(panelSettings?.panelStyle)}</span>
      </div>

      <section className="summary-model-settings">
        <h4>{t('settings.summaryModelSettings')}</h4>
        <div className="browser-pref-row">
          <label htmlFor="summary-source">{t('settings.modelSource')}</label>
          <select
            id="summary-source"
            value={summarySettings.source}
            onChange={(e) => onSummaryModelChange?.({source: e.target.value, modelId: e.target.value === 'cli_adapter' ? 'current' : ''})}
          >
            <option value="cli_adapter">{t('settings.currentCLIAdapter')}</option>
            <option value="local_model">{t('settings.localModel')}</option>
            <option value="manual">{t('settings.manualInput')}</option>
          </select>
        </div>
        {summarySettings.source === 'cli_adapter' && (
          <div className="browser-pref-row">
            <label htmlFor="summary-adapter">CLI adapter</label>
            <select
              id="summary-adapter"
              value={summarySettings.modelId || 'current'}
              onChange={(e) => onSummaryModelChange?.({modelId: e.target.value})}
            >
              <option value="current">{t('settings.followCurrentAdapter')}</option>
              {(adapterList || []).map((adapter) => (
                <option key={adapter.id || adapter.name} value={adapter.id || adapter.name}>{adapter.name || adapter.id}</option>
              ))}
            </select>
          </div>
        )}
        {summarySettings.source === 'local_model' && (
          <>
            <div className="browser-pref-row">
              <label htmlFor="summary-local">{t('settings.localModel')}</label>
              <select
                id="summary-local"
                value={summarySettings.modelId || ''}
                onChange={(e) => {
                  const picked = localOptions.find((option) => option.id === e.target.value);
                  onSummaryModelChange?.({modelId: e.target.value, endpoint: picked?.endpoint || ''});
                }}
              >
                <option value="">{summaryModelScan?.message || t('settings.selectLocalModel')}</option>
                {localOptions.map((option) => (
                  <option key={`${option.provider}:${option.id}`} value={option.id}>{option.label}</option>
                ))}
              </select>
            </div>
            <button className="style-diff-preview-btn" type="button" onClick={onSummaryModelRescan}>{t('settings.rescanLocalModel')}</button>
          </>
        )}
        {(summarySettings.source === 'manual' || (summarySettings.source === 'local_model' && localOptions.length === 0)) && (
          <div className="summary-manual-grid">
            <input
              type="text"
              placeholder={t('settings.modelNamePlaceholder')}
              value={summarySettings.manualModel || ''}
              onChange={(e) => onSummaryModelChange?.({manualModel: e.target.value, modelId: e.target.value})}
            />
            <input
              type="text"
              placeholder={t('settings.endpointPlaceholder')}
              value={summarySettings.manualEndpoint || summarySettings.endpoint || ''}
              onChange={(e) => onSummaryModelChange?.({manualEndpoint: e.target.value, endpoint: e.target.value})}
            />
          </div>
        )}
        <label className="summary-model-checkbox">
          <input
            type="checkbox"
            checked={!!summarySettings.alwaysUse}
            onChange={(e) => onSummaryModelChange?.({alwaysUse: e.target.checked})}
          />
          <span>{t('settings.alwaysUseSetting')}</span>
        </label>
      </section>

      <section className="voice-settings">
        <header className="voice-settings-header">
          <h4>{t('settings.localVoice')}</h4>
          <button type="button" className="style-diff-preview-btn" onClick={onVoiceSettingsRefresh}>{t('settings.voiceRecheck')}</button>
        </header>
        <div className="voice-health-row">
          <span className={`voice-health-dot ${voiceState?.status === 'ready' ? 'voice-health-ready' : ''}`} />
          <span>{voiceStatusLabel(voiceState?.status)}</span>
          <small>{voiceState?.language ? `language: ${voiceState.language}` : ''}</small>
        </div>
        <div className="voice-install-row">
          <button
            type="button"
            className="style-diff-preview-btn"
            disabled={voiceInstallBusy}
            onClick={voiceState?.modelAvailable ? onVoiceModelRemove : onVoiceModelInstall}
          >
            {voiceInstallBusy ? t('settings.voiceProcessing') : voiceState?.modelAvailable ? t('settings.voiceRemoveBase') : t('settings.voiceInstallBase')}
          </button>
          <code>{voiceState?.managedModelPath || 'voice/models/ggml-base.bin'}</code>
        </div>
        <div className="browser-pref-row">
          <label htmlFor="voice-language-mode">{t('settings.voiceLanguage')}</label>
          <select
            id="voice-language-mode"
            value={voiceSettings.languageMode || 'auto'}
            onChange={(e) => onVoiceSettingsChange?.({languageMode: e.target.value})}
          >
            <option value="follow_app">{t('settings.followAppLang')}</option>
            <option value="auto">{t('settings.autoDetect')}</option>
            <option value="manual">{t('settings.manualSpecify')}</option>
          </select>
        </div>
        {voiceSettings.languageMode === 'manual' && (
          <div className="browser-pref-row">
            <label htmlFor="voice-language-manual">{t('settings.specifyLanguage')}</label>
            <select
              id="voice-language-manual"
              value={voiceSettings.manualLanguage || 'zh'}
              onChange={(e) => onVoiceSettingsChange?.({manualLanguage: e.target.value})}
            >
              <option value="zh">{t('settings.chinese')}</option>
              <option value="en">English</option>
              <option value="ja">{t("settings.langJa")}</option>
              <option value="ko">한국어</option>
            </select>
          </div>
        )}
        <div className="voice-install-row">
          <code>{voiceState?.whisperBinPath || t('settings.voiceNotIncluded')}</code>
        </div>
        <label className="summary-model-checkbox">
          <input
            type="checkbox"
            checked={!!voiceSettings.debugMode}
            onChange={(e) => onVoiceSettingsChange?.({debugMode: e.target.checked})}
          />
          <span>{t('settings.voiceDebugMode')}</span>
          <button type="button" className="style-diff-preview-btn" onClick={onVoiceDebugClear}>{t('settings.voiceDebugClear')}</button>
        </label>
        <label className="summary-model-checkbox">
          <input
            type="checkbox"
            checked={!!voiceSettings.commandMode}
            onChange={(e) => onVoiceSettingsChange?.({commandMode: e.target.checked})}
          />
          <span>{t('settings.voiceCommandMode')}</span>
        </label>
      </section>

      {/* Style Diff Preview trigger */}
      <div className="style-diff-input-row">
        <label htmlFor="style-diff-input">Style Diff JSON</label>
        <textarea
          id="style-diff-input"
          className="style-diff-textarea"
          placeholder='{"--bg": "#111", "--surface": "#222"}'
          value={diffInput}
          onChange={(e) => setDiffInput(e.target.value)}
          rows={3}
        />
        <button
          type="button"
          className="style-diff-preview-btn"
          disabled={!diffInput.trim()}
          onClick={() => onPreviewStyleDiff(diffInput.trim())}
        >
          {t('styleDiff.previewLabel')}
        </button>
      </div>
    </div>
  );
}
