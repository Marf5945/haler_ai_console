import {useState} from 'react';
import useI18n from '../../locales/useI18n';

const LANGUAGE_OPTIONS = [
  {code: 'zh-TW', label: '中文'},
  {code: 'en', label: 'English'},
  {code: 'ja', label: '日本語'},
  {code: 'pt-PT', label: 'Português'},
  {code: 'es', label: 'Español'},
  {code: 'th', label: 'ไทย'},
  {code: 'ko', label: '한국어'},
];

const API_PROVIDER_OPTIONS = [
  {id: 'openai', name: 'OpenAI', baseURL: 'https://api.openai.com/v1', model: 'gpt-4.1-mini'},
  {id: 'deepseek', name: 'DeepSeek', baseURL: 'https://api.deepseek.com', model: 'deepseek-chat'},
  {id: 'openrouter', name: 'OpenRouter', baseURL: 'https://openrouter.ai/api/v1', model: 'openai/gpt-4.1-mini'},
  {id: 'groq', name: 'Groq', baseURL: 'https://api.groq.com/openai/v1', model: 'openai/gpt-oss-20b'},
];

function defaultAPISetup() {
  const provider = API_PROVIDER_OPTIONS[0];
  return {
    providerId: provider.id,
    providerName: provider.name,
    baseURL: provider.baseURL,
    model: provider.model,
    apiKey: '',
  };
}

function sanitizeAdapterToken(value) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 48) || 'model';
}

function OnboardingToolDemo() {
  const t = useI18n(s => s.t);
  return (
    <div className="onboarding-tool-demo" aria-label={t('onboarding.toolDemoAriaLabel')}>
      <div className="tool-demo-stage">
        <aside className="tool-demo-rail" aria-hidden="true">
          <span className="tool-demo-rail-label">{t('onboarding.leftRail')}</span>
          <button className="tool-demo-rail-btn" type="button">
            <span>⊕</span>
            <span>{t('onboarding.newAgent')}</span>
          </button>
          <button className="tool-demo-rail-btn tool-demo-trigger" type="button">
            <span>⌕</span>
            <span>{t('onboarding.tools')}</span>
          </button>
          <button className="tool-demo-rail-btn" type="button">
            <span>⚙</span>
            <span>{t('onboarding.settings')}</span>
          </button>
        </aside>

        <section className="tool-demo-panel" aria-hidden="true">
          <header className="tool-demo-tabs">
            <span className="active">{t('onboarding.tabExternalLinks')}</span>
            <span>{t('onboarding.tabAutomation')}</span>
            <span>{t('onboarding.tabPackage')}</span>
          </header>
          <div className="tool-demo-list">
            <div className="tool-demo-item tool-demo-item-primary">
              <span>▤</span>
              <strong>{t('onboarding.refLink')}</strong>
              <small>{t('onboarding.refLinkHint')}</small>
            </div>
            <div className="tool-demo-item">
              <span>▤</span>
              <strong>{t('onboarding.refDoc')}</strong>
              <small>{t('onboarding.refDocHint')}</small>
            </div>
            <div className="tool-demo-item">
              <span>✉</span>
              <strong>Gmail</strong>
              <small>{t('onboarding.gmailHint')}</small>
            </div>
          </div>
          <div className="tool-demo-drag-chip">
            <span>▤</span>
            <strong>{t('onboarding.refLink')}</strong>
          </div>
        </section>

        <section className="tool-demo-favorites" aria-hidden="true">
          <span className="tool-demo-drop-label">{t("rightRail.useTools")}</span>
          <strong>{t('onboarding.toolFavorites')}</strong>
          <p>{t('onboarding.toolDragHint')}</p>
          <div className="tool-demo-drop-slot">
            <span>{t('onboarding.dropHere')}</span>
          </div>
        </section>
      </div>

      <ol className="tool-demo-captions">
        <li><span>1</span>{t('onboarding.toolStep1')}</li>
        <li><span>2</span>{t('onboarding.toolStep2')}</li>
        <li><span>3</span>{t('onboarding.toolStep3')}</li>
      </ol>

      <div className="onboarding-search-demo" aria-hidden="true">
        <div className="onboarding-search-card">
          <span className="onboarding-search-icon">⌕</span>
          <span>
            <strong>{t('onboarding.searchDemoTitle')}</strong>
            <small>{t('onboarding.searchDemoKeyword')}</small>
          </span>
          <i>{t('onboarding.searchDemoBadge')}</i>
        </div>
        <div className="onboarding-search-preview">
          <strong>{t('onboarding.searchDemoFile')}</strong>
          <p># {t('onboarding.searchDemoTitle')}</p>
        </div>
        <div className="onboarding-search-drag">
          <span>{t('onboarding.searchDemoFile')}</span>
        </div>
      </div>
    </div>
  );
}

export default function OnboardingOverlay({state, onCompleteStep, onFinish, onGoBack, onDetectCLI, onEnableCLI, onScanLocalModels, onEnableLocalModel, onRegisterCustomCLI, onRegisterLLMAPI, onConfirmRegisterLLMAPI, onOpenSettings, onCloseSettings}) {
  const t = useI18n(s => s.t);
  const language = useI18n(s => s.language);
  const setLanguage = useI18n(s => s.setLanguage);
  const [detectResults, setDetectResults] = useState(null);
  const [detecting, setDetecting] = useState(false);
  const [enabledIds, setEnabledIds] = useState(new Set());
  const [connectionMode, setConnectionMode] = useState('api');
  const [showManual, setShowManual] = useState(true);
  const [manualName, setManualName] = useState('');
  const [manualPath, setManualPath] = useState('');
  const [manualError, setManualError] = useState('');
  const [localModelResults, setLocalModelResults] = useState([]);
  const [apiSetup, setApiSetup] = useState(defaultAPISetup);
  const [apiError, setApiError] = useState('');
  const [apiStatus, setApiStatus] = useState('');
  const [registeringAPI, setRegisteringAPI] = useState(false);
  const [manualLocalName, setManualLocalName] = useState('');
  const [manualLocalEndpoint, setManualLocalEndpoint] = useState('http://localhost:11434/v1');
  const [manualLocalModel, setManualLocalModel] = useState('');
  const [manualLocalProvider, setManualLocalProvider] = useState('ollama');
  const [manualLocalError, setManualLocalError] = useState('');
  if (!state || !state.is_first_run) return null;

  const steps = state.steps || [];
  const currentIdx = state.current_step ?? 0;
  const current = steps[currentIdx];
  const isFirstStep = currentIdx === 0;
  const isLastStep = currentIdx >= steps.length - 1;
  const isPersonaStep = current?.id === 'persona';
  const isToolStep = current?.id === 'tool';
  const isAdapterStep = current?.id === 'adapter';

  async function runDetect() {
    setDetecting(true);
    try {
      const [cliResults, localResults] = await Promise.all([
        onDetectCLI().catch(() => []),
        onScanLocalModels?.().catch(() => []),
      ]);
      setDetectResults(cliResults || []);
      setLocalModelResults(localResults || []);
    } catch {
      setDetectResults([]);
      setLocalModelResults([]);
    }
    setDetecting(false);
  }

  async function handleEnable(adapterID) {
    try {
      await onEnableCLI(adapterID);
      setEnabledIds((prev) => new Set([...prev, adapterID]));
    } catch { /* ignore */ }
  }

  async function handleEnableLocal(result) {
    try {
      await onEnableLocalModel(result);
      setEnabledIds((prev) => new Set([...prev, result.adapter_id]));
    } catch { /* ignore */ }
  }

  async function handleManualRegister() {
    setManualError('');
    if (!manualName.trim() || !manualPath.trim()) {
      setManualError(t('onboarding.fillNamePath'));
      return;
    }
    try {
      const result = await onRegisterCustomCLI(manualName.trim(), manualPath.trim());
      if (result && result.found) {
        setDetectResults((prev) => [...(prev || []), result]);
        setEnabledIds((prev) => new Set([...prev, result.adapter_id]));
        setManualName('');
        setManualPath('');
        setShowManual(false);
      }
    } catch (e) {
      setManualError(e?.message || t('onboarding.registerFail'));
    }
  }

  function handleNext() {
    if (isPersonaStep) onCloseSettings?.();
    if (current) onCompleteStep(current.id);
    if (isLastStep) onFinish();
    const nextStep = steps[currentIdx + 1];
    if (nextStep?.id === 'persona') onOpenSettings?.();
  }

  function handlePrev() {
    if (isPersonaStep) onCloseSettings?.();
    if (isFirstStep) return;
    const prevStep = steps[currentIdx - 1];
    if (prevStep) {
      onGoBack?.();
      if (prevStep.id === 'persona') onOpenSettings?.();
    }
  }

  function handleSkip() {
    if (isPersonaStep) onCloseSettings?.();
    if (!window.confirm(t('onboarding.skipWarning'))) return;
    onFinish();
  }

  const supportedDetectResults = detectResults ? detectResults.filter((r) => r.found && r.supported !== false) : [];
  const unsupportedDetectResults = detectResults ? detectResults.filter((r) => r.found && r.supported === false) : [];
  const foundCount = supportedDetectResults.length;
  const isSpotlightMode = isPersonaStep;
  const currentTitle = current ? t(`onboarding.steps.${current.id}.title`) : t('onboarding.completedTitle');
  const currentDescription = current ? t(`onboarding.steps.${current.id}.description`) : '';

  function updateAPIProvider(providerId) {
    const provider = API_PROVIDER_OPTIONS.find((item) => item.id === providerId) || API_PROVIDER_OPTIONS[0];
    setApiSetup((prev) => ({
      ...prev,
      providerId: provider.id,
      providerName: provider.name,
      baseURL: provider.baseURL,
      model: provider.model || prev.model,
    }));
    setApiError('');
    setApiStatus('');
  }

  async function handleAPIRegister() {
    setApiError('');
    setApiStatus('');
    if (!apiSetup.apiKey.trim() || !apiSetup.baseURL.trim() || !apiSetup.model.trim()) {
      setApiError(t('onboarding.fillAPIFields'));
      return;
    }
    if (!onRegisterLLMAPI) {
      setApiError(t('onboarding.registerFail'));
      return;
    }
    setRegisteringAPI(true);
    try {
      let result = await onRegisterLLMAPI(apiSetup);
      if (result?.need_confirm && result?.confirm_type === 'private_network') {
        const ok = window.confirm(t('onboarding.privateNetworkConfirm', {host: result.hostname || result.original_url || apiSetup.baseURL}));
        if (!ok) {
          setApiStatus(t('onboarding.privateNetworkCancel'));
          return;
        }
        result = await onConfirmRegisterLLMAPI?.(apiSetup);
      }
      if (result?.adapter_id) {
        setEnabledIds((prev) => new Set([...prev, result.adapter_id]));
        setApiStatus(t('onboarding.apiRegistered', {name: result.name || apiSetup.providerName || 'LLM API'}));
      } else {
        setApiStatus(t('onboarding.apiRegistered', {name: apiSetup.providerName || 'LLM API'}));
      }
    } catch (e) {
      setApiError(e?.message || t('onboarding.apiRegisterFail'));
    } finally {
      setRegisteringAPI(false);
    }
  }

  async function handleManualLocalRegister() {
    setManualLocalError('');
    if (!manualLocalEndpoint.trim() || !manualLocalModel.trim()) {
      setManualLocalError(t('onboarding.fillLocalFields'));
      return;
    }
    const provider = manualLocalProvider || 'ollama';
    const model = manualLocalModel.trim();
    const endpoint = manualLocalEndpoint.trim();
    const name = manualLocalName.trim() || `${provider === 'lmstudio' ? 'LM Studio' : provider === 'ollama' ? 'Ollama' : 'Local'} - ${model}`;
    const adapterID = `local-${sanitizeAdapterToken(provider)}-${sanitizeAdapterToken(model)}-${Date.now()}`;
    try {
      await onEnableLocalModel?.({
        adapter_id: adapterID,
        name,
        model_id: model,
        provider,
        endpoint,
      });
      setEnabledIds((prev) => new Set([...prev, adapterID]));
      setManualLocalError('');
    } catch (e) {
      setManualLocalError(e?.message || t('onboarding.localRegisterFail'));
    }
  }

  return (
    <div className={`onboarding-overlay${isSpotlightMode ? ' spotlight-mode' : ''}`}>
      <div className={`onboarding-card${isSpotlightMode ? ' spotlight-card' : ''}${isToolStep ? ' tool-demo-card' : ''}${isAdapterStep ? ' adapter-card' : ''}`}>
        <div className="onboarding-progress">
          {steps.map((s, i) => (
            <div key={s.id} className={`onboarding-dot${i === currentIdx ? ' active' : ''}${s.completed ? ' done' : ''}`} />
          ))}
        </div>

        <h2 className="onboarding-title">{currentTitle}</h2>
        <p className="onboarding-desc">{currentDescription}</p>

        {isPersonaStep && (
          <div className="onboarding-spotlight-hint">
            <div className="spotlight-arrow">←</div>
            <p>{t('onboarding.spotlightHint')}</p>
          </div>
        )}

        {isToolStep && <OnboardingToolDemo />}

        {current?.id === 'adapter' && (
          <div className="onboarding-adapter-section">
            <div className="onboarding-connection-tabs" role="tablist" aria-label={t('onboarding.connectionChoice')}>
              {['api', 'cli', 'local'].map((mode) => (
                <button
                  key={mode}
                  type="button"
                  className={connectionMode === mode ? 'active' : ''}
                  onClick={() => setConnectionMode(mode)}
                >
                  <strong>{t(`onboarding.connection.${mode}.title`)}</strong>
                  <small>{t(`onboarding.connection.${mode}.hint`)}</small>
                </button>
              ))}
            </div>

            {connectionMode === 'api' && (
              <div className="onboarding-connection-panel">
                <label className="onboarding-manual-field">
                  <span>{t('onboarding.apiProvider')}</span>
                  <select
                    className="onboarding-manual-input"
                    value={apiSetup.providerId}
                    onChange={(e) => updateAPIProvider(e.target.value)}
                  >
                    {API_PROVIDER_OPTIONS.map((provider) => (
                      <option key={provider.id} value={provider.id}>{provider.name}</option>
                    ))}
                  </select>
                </label>
                <label className="onboarding-manual-field">
                  <span>API Key</span>
                  <input type="password" className="onboarding-manual-input" placeholder="sk-..."
                    value={apiSetup.apiKey} onChange={(e) => setApiSetup((prev) => ({...prev, apiKey: e.target.value}))} />
                </label>
                <label className="onboarding-manual-field">
                  <span>Base URL</span>
                  <input type="text" className="onboarding-manual-input" placeholder="https://api.example.com/v1"
                    value={apiSetup.baseURL} onChange={(e) => setApiSetup((prev) => ({...prev, baseURL: e.target.value}))} />
                </label>
                <label className="onboarding-manual-field">
                  <span>Model</span>
                  <input type="text" className="onboarding-manual-input" placeholder="model id"
                    value={apiSetup.model} onChange={(e) => setApiSetup((prev) => ({...prev, model: e.target.value}))} />
                </label>
                <button className="onboarding-detect-btn primary" type="button" onClick={handleAPIRegister} disabled={registeringAPI}>
                  {registeringAPI ? t('onboarding.registering') : t('onboarding.connectAPI')}
                </button>
                {apiError && <p className="onboarding-manual-error">{apiError}</p>}
                {apiStatus && <p className="onboarding-detect-summary">{apiStatus}</p>}
              </div>
            )}

            {connectionMode === 'cli' && (
              <div className="onboarding-connection-panel">
                {!detecting && (
                  <button className="onboarding-detect-btn" type="button" onClick={runDetect}>
                    {detectResults ? t('onboarding.rescan') : t('onboarding.scanCLI')}
                  </button>
                )}
                <p className="onboarding-mini-note">{t('onboarding.cliManualHint')}</p>
              </div>
            )}

            {connectionMode === 'local' && (
              <div className="onboarding-connection-panel">
                {!detecting && (
                  <button className="onboarding-detect-btn" type="button" onClick={runDetect}>
                    {detectResults ? t('onboarding.rescanLocal') : t('onboarding.scanLocal')}
                  </button>
                )}
                <div className="onboarding-manual-form">
                  <label className="onboarding-manual-field">
                    <span>{t('onboarding.localProvider')}</span>
                    <select className="onboarding-manual-input" value={manualLocalProvider}
                      onChange={(e) => {
                        const provider = e.target.value;
                        setManualLocalProvider(provider);
                        if (provider === 'ollama') setManualLocalEndpoint('http://localhost:11434/v1');
                        if (provider === 'lmstudio') setManualLocalEndpoint('http://localhost:1234/v1');
                      }}>
                      <option value="ollama">Ollama</option>
                      <option value="lmstudio">LM Studio</option>
                    </select>
                  </label>
                  <label className="onboarding-manual-field">
                    <span>{t('onboarding.localName')}</span>
                    <input type="text" className="onboarding-manual-input" placeholder={t('onboarding.localNamePlaceholder')}
                      value={manualLocalName} onChange={(e) => setManualLocalName(e.target.value)} />
                  </label>
                  <label className="onboarding-manual-field">
                    <span>{t('onboarding.localEndpoint')}</span>
                    <input type="text" className="onboarding-manual-input" placeholder="http://localhost:11434/v1"
                      value={manualLocalEndpoint} onChange={(e) => setManualLocalEndpoint(e.target.value)} />
                  </label>
                  <label className="onboarding-manual-field">
                    <span>{t('onboarding.localModel')}</span>
                    <input type="text" className="onboarding-manual-input" placeholder="qwen2.5:14b"
                      value={manualLocalModel} onChange={(e) => setManualLocalModel(e.target.value)} />
                  </label>
                  {manualLocalError && <p className="onboarding-manual-error">{manualLocalError}</p>}
                  <div className="onboarding-manual-actions">
                    <button type="button" onClick={handleManualLocalRegister}>{t('onboarding.confirmAdd')}</button>
                  </div>
                </div>
              </div>
            )}

            {detecting && <p className="onboarding-scanning">{t('onboarding.scanning')}</p>}

            {connectionMode === 'cli' && detectResults && foundCount > 0 && (
              <ul className="onboarding-detect-list">
                {supportedDetectResults.map((r) => {
                  const isEnabled = enabledIds.has(r.adapter_id);
                  return (
                    <li key={r.adapter_id} className={isEnabled ? 'found enabled' : 'found'}>
                      <span className="detect-name">{r.name}</span>
                      <span className="detect-path">{r.path}</span>
                      {isEnabled ? (
                        <span className="detect-enabled-badge">{t('onboarding.enabled')}</span>
                      ) : (
                        <button className="detect-enable-btn" type="button" onClick={() => handleEnable(r.adapter_id)}>{t('onboarding.enable')}</button>
                      )}
                    </li>
                  );
                })}
              </ul>
            )}

            {connectionMode === 'cli' && unsupportedDetectResults.length > 0 && (
              <ul className="onboarding-detect-list">
                {unsupportedDetectResults.map((r) => (
                  <li key={r.adapter_id} className="found disabled">
                    <span className="detect-name">{r.name}</span>
                    <span className="detect-path">{r.path}</span>
                    <span className="detect-path">{r.reason || t('onboarding.registerFail')}</span>
                  </li>
                ))}
              </ul>
            )}

            {connectionMode === 'cli' && detectResults && foundCount === 0 && (
              <p className="onboarding-install-hint">
                {t('onboarding.noCLI')}
                <br />
                <code>npm install -g @anthropic-ai/claude-code</code>
              </p>
            )}

            {connectionMode === 'cli' && detectResults && foundCount > 0 && (
              <p className="onboarding-detect-summary">
                {t('onboarding.cliFound', { foundCount, enabledCount: enabledIds.size })}
              </p>
            )}

            {connectionMode === 'local' && localModelResults && localModelResults.length > 0 && (
              <>
                <p className="onboarding-detect-summary" style={{marginTop: '12px'}}>
                  {t('onboarding.localModelFound', { count: localModelResults.length })}
                </p>
                <ul className="onboarding-detect-list">
                  {localModelResults.map((r) => {
                    const isEnabled = enabledIds.has(r.adapter_id);
                    return (
                      <li key={r.adapter_id} className={isEnabled ? 'found enabled local-model' : 'found local-model'}>
                        <span className="detect-name">{r.name}</span>
                        <span className="detect-path">{r.provider}</span>
                        {isEnabled ? (
                          <span className="detect-enabled-badge">{t("onboarding.enabled")}</span>
                        ) : (
                          <button className="detect-enable-btn" type="button" onClick={() => handleEnableLocal(r)}>{t("onboarding.enable")}</button>
                        )}
                      </li>
                    );
                  })}
                </ul>
              </>
            )}

            {connectionMode === 'cli' && (
              <div className="onboarding-manual-section">
                {!showManual ? (
                  <button className="onboarding-manual-toggle" type="button" onClick={() => setShowManual(true)}>
                    {t('onboarding.addCLIManual')}
                  </button>
                ) : (
                  <div className="onboarding-manual-form">
                    <input type="text" className="onboarding-manual-input" placeholder={t('onboarding.namePlaceholder')}
                      value={manualName} onChange={(e) => setManualName(e.target.value)} />
                    <input type="text" className="onboarding-manual-input" placeholder={t('onboarding.pathPlaceholder')}
                      value={manualPath} onChange={(e) => setManualPath(e.target.value)} />
                    {manualError && <p className="onboarding-manual-error">{manualError}</p>}
                    <div className="onboarding-manual-actions">
                      <button type="button" onClick={handleManualRegister}>{t('onboarding.confirmAdd')}</button>
                      <button type="button" className="onboarding-manual-cancel" onClick={() => { setShowManual(false); setManualError(''); }}>{t('onboarding.cancel')}</button>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        <div className="onboarding-language-strip" aria-label={t('onboarding.languageLabel')}>
          <span>{t('onboarding.languageLabel')}</span>
          <div className="onboarding-language-options">
            {LANGUAGE_OPTIONS.map((option) => (
              <button
                key={option.code}
                type="button"
                className={language === option.code ? 'active' : ''}
                onClick={() => setLanguage(option.code)}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>

        <div className="onboarding-actions">
          {!isFirstStep && (
            <button className="onboarding-prev" type="button" onClick={handlePrev} title={t('onboarding.prevStep')}>
              ←
            </button>
          )}
          <button className="onboarding-skip" type="button" onClick={handleSkip}>{t('onboarding.skip')}</button>
          <button className="onboarding-next" type="button" onClick={handleNext}>
            {isLastStep ? t('onboarding.getStarted') : '→'}
          </button>
        </div>
      </div>
    </div>
  );
}
