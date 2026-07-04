import useI18n from '../../locales/useI18n';
import {openExternal} from '../../lib/openExternal';

// 各 provider 的 API Key 前綴範例，幫使用者確認「有沒有貼對那一串」。
const API_KEY_PLACEHOLDER = {
  'nvidia-nim': 'nvapi-...',
  anthropic: 'sk-ant-...',
  gemini: 'AIza...',
  cohere: 'co-...',
  huggingface: 'hf_...',
};

// 各 provider 的完整 model id 範例。NVIDIA / OpenRouter 等需要「廠牌/型號」格式，
// 少一個斜線就會被退回 404，所以這裡直接給可複製的正確樣板。
const MODEL_EXAMPLE = {
  'nvidia-nim': 'meta/llama-3.1-8b-instruct',
  openai: 'gpt-4.1-mini',
  deepseek: 'deepseek-chat',
  gemini: 'gemini-2.5-flash',
  xai: 'grok-4',
  groq: 'openai/gpt-oss-20b',
  openrouter: 'openai/gpt-4.1-mini',
  mistral: 'mistral-small-latest',
  together: 'meta-llama/Llama-3.3-70B-Instruct-Turbo',
  perplexity: 'sonar',
  huggingface: 'openai/gpt-oss-120b:cerebras',
};

export default function LLMAPISetupModal({setup, guideStep, onGuideStepChange, onChange, onCancel, onTest, onSubmit}) {
  const {t} = useI18n();
  const update = (field, value) => onChange({...setup, [field]: value});
  const providerId = setup.providerId || '';
  const apiKeyPlaceholder = API_KEY_PLACEHOLDER[providerId] || 'sk-...';
  const modelExample = MODEL_EXAMPLE[providerId] || setup.model || 'model id';
  const guide = [
    {
      title: t('llmSetup.guide0Title'),
      body: t('llmSetup.guide0Body', { providerName: setup.providerName || 'Provider' }),
      action: t('llmSetup.guide0Action'),
      url: setup.apiKeyURL,
    },
    {
      title: t('llmSetup.guide1Title'),
      body: t('llmSetup.guide1Body'),
      action: t('llmSetup.guide1Action'),
      url: setup.docsURL,
    },
    {
      title: t('llmSetup.guide2Title'),
      body: t('llmSetup.guide2Body'),
      action: t('llmSetup.guide2Action'),
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
          <input type="password" value={setup.apiKey || ''} onChange={(event) => update('apiKey', event.target.value)} placeholder={apiKeyPlaceholder} />
          <small className="reference-link-hint">{t('llmSetup.apiKeyHint', { example: apiKeyPlaceholder })}</small>
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
          <input type="text" value={setup.model || ''} onChange={(event) => update('model', event.target.value)} placeholder={modelExample} />
          <small className="reference-link-hint">{t('llmSetup.modelFormatHint', { example: modelExample })}</small>
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
