import useI18n from '../../locales/useI18n';
import {openExternal} from '../../lib/appShared';

export default function LLMAPISetupModal({setup, guideStep, onGuideStepChange, onChange, onCancel, onTest, onSubmit}) {
  const {t} = useI18n();
  const update = (field, value) => onChange({...setup, [field]: value});
  const guide = [
    {
      title: t('llmSetup.getApiKeyTitle'),
      body: t('llmSetup.getApiKeyBody', { providerName: setup.providerName || 'Provider' }),
      action: t('llmSetup.getApiKeyAction'),
      url: setup.apiKeyURL,
    },
    {
      title: t('llmSetup.confirmBaseUrlTitle'),
      body: t('llmSetup.confirmBaseUrlBody'),
      action: t('llmSetup.confirmBaseUrlAction'),
      url: setup.docsURL,
    },
    {
      title: t('llmSetup.fillModelTitle'),
      body: t('llmSetup.fillModelBody'),
      action: t('llmSetup.fillModelAction'),
      url: setup.docsURL,
    },
  ];
  const currentGuide = guide[Math.min(guideStep || 0, guide.length - 1)];
  const openHelp = (field) => {
    if (field === 'apiKey' && setup.apiKeyURL) return openExternal(setup.apiKeyURL);
    if (setup.docsURL) return openExternal(setup.docsURL);
    openExternal(`https://www.google.com/search?q=${encodeURIComponent(`${setup.providerName || 'LLM API'} ${field} official docs`)}`);
  };
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('llmSetup.setupTitle')}>
      <section className="reference-link-modal llm-api-setup-modal">
        <header className="llm-api-setup-head">
          <span>{t('llmSetup.apiSetupTitle')}</span>
          <strong>{setup.providerName || 'LLM API'}</strong>
        </header>
        <div className="rb-guide-card llm-api-guide-card">
          <span className="rb-guide-count">{t('remoteBridgeSetup.stepCount', { current: (guideStep || 0) + 1, total: guide.length })}</span>
          <strong>{currentGuide.title}</strong>
          <p>{currentGuide.body}</p>
          <div className="rb-guide-actions">
            {currentGuide.url && (
              <button type="button" onClick={() => openExternal(currentGuide.url)}>
                {currentGuide.action}
              </button>
            )}
            {guideStep < guide.length - 1 ? (
              <button type="button" onClick={() => onGuideStepChange(Math.min((guideStep || 0) + 1, guide.length - 1))}>{t('remoteBridgeSetup.next')}</button>
            ) : (
              <button type="button" onClick={() => onGuideStepChange(0)}>{t('remoteBridgeSetup.restartGuide')}</button>
            )}
          </div>
        </div>
        <label className="rb-field">
          <span className="rb-field-title">
            <span>API Key</span>
            <button className="rb-help-btn" type="button" onClick={() => openHelp('apiKey')}>?</button>
          </span>
          <input type="password" value={setup.apiKey || ''} onChange={(event) => update('apiKey', event.target.value)} placeholder="sk-..." />
        </label>
        <label className="rb-field">
          <span className="rb-field-title">
            <span>Base URL</span>
            <button className="rb-help-btn" type="button" onClick={() => openHelp('baseURL')}>?</button>
          </span>
          <input type="text" value={setup.baseURL || ''} onChange={(event) => update('baseURL', event.target.value)} placeholder="https://api.example.com/v1" />
        </label>
        <label className="rb-field">
          <span className="rb-field-title">
            <span>Model</span>
            <button className="rb-help-btn" type="button" onClick={() => openHelp('model')}>?</button>
          </span>
          <input type="text" value={setup.model || ''} onChange={(event) => update('model', event.target.value)} placeholder="model id" />
        </label>
        <small className="reference-link-hint">{t('llmSetup.nameHint')}</small>
        <div className="rb-actions">
          <button type="button" onClick={onTest}>{t('llmSetup.testConnection')}</button>
          <button type="button" onClick={onSubmit}>{t('llmSetup.save')}</button>
          <button type="button" className="rb-cancel-btn" onClick={onCancel}>{t('common.cancel')}</button>
        </div>
      </section>
    </div>
  );
}
