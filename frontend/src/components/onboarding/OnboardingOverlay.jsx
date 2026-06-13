import {useState} from 'react';
import useI18n from '../../locales/useI18n';
import OnboardingToolDemo from './OnboardingToolDemo';

export default function OnboardingOverlay({state, onCompleteStep, onFinish, onGoBack, onDetectCLI, onEnableCLI, onScanLocalModels, onEnableLocalModel, onRegisterCustomCLI, onOpenSettings, onCloseSettings}) {
  const t = useI18n(s => s.t);
  const [detectResults, setDetectResults] = useState(null);
  const [detecting, setDetecting] = useState(false);
  const [enabledIds, setEnabledIds] = useState(new Set());
  const [showManual, setShowManual] = useState(false);
  const [manualName, setManualName] = useState('');
  const [manualPath, setManualPath] = useState('');
  const [manualError, setManualError] = useState('');
  const [localModelResults, setLocalModelResults] = useState([]);
  if (!state || !state.is_first_run) return null;

  const steps = state.steps || [];
  const currentIdx = state.current_step ?? 0;
  const current = steps[currentIdx];
  const isFirstStep = currentIdx === 0;
  const isLastStep = currentIdx >= steps.length - 1;
  const isPersonaStep = current?.id === 'persona';
  const isToolStep = current?.id === 'tool';

  // CLI + Local model 偵測 handler（只偵測，不註冊）
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

  // 使用者點「啟用」→ 註冊 CLI 到 registry
  async function handleEnable(adapterID) {
    try {
      await onEnableCLI(adapterID);
      setEnabledIds((prev) => new Set([...prev, adapterID]));
    } catch { /* ignore */ }
  }

  // 使用者點「啟用」→ 註冊 Local model 到 registry
  async function handleEnableLocal(result) {
    try {
      await onEnableLocalModel(result);
      setEnabledIds((prev) => new Set([...prev, result.adapter_id]));
    } catch { /* ignore */ }
  }

  // 手動註冊 CLI
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

  // 下一步
  function handleNext() {
    // 離開人格步驟 → 關閉設定頁
    if (isPersonaStep) onCloseSettings?.();
    if (current) onCompleteStep(current.id);
    if (isLastStep) onFinish();
    // 進入人格步驟 → 開啟設定頁
    const nextStep = steps[currentIdx + 1];
    if (nextStep?.id === 'persona') onOpenSettings?.();
  }

  // 上一步
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
    onFinish();
  }

  const foundCount = detectResults ? detectResults.filter((r) => r.found).length : 0;

  // 人格步驟 → spotlight 模式（半透明背景，讓底層 UI 可見）
  const isSpotlightMode = isPersonaStep;

  return (
    <div className={`onboarding-overlay${isSpotlightMode ? ' spotlight-mode' : ''}`}>
      <div className={`onboarding-card${isSpotlightMode ? ' spotlight-card' : ''}${isToolStep ? ' tool-demo-card' : ''}`}>
        {/* 進度指示器 */}
        <div className="onboarding-progress">
          {steps.map((s, i) => (
            <div key={s.id} className={`onboarding-dot${i === currentIdx ? ' active' : ''}${s.completed ? ' done' : ''}`} />
          ))}
        </div>

        <h2 className="onboarding-title">{current?.title || t('onboarding.completedTitle')}</h2>
        <p className="onboarding-desc">{current?.description || ''}</p>

        {/* 人格步驟：spotlight 指引 */}
        {isPersonaStep && (
          <div className="onboarding-spotlight-hint">
            <div className="spotlight-arrow">←</div>
            <p>{t('onboarding.spotlightHint')}</p>
          </div>
        )}

        {/* 試用工具步驟：卡片內完整示範 */}
        {isToolStep && <OnboardingToolDemo />}

        {/* Adapter 偵測 + 啟用 */}
        {current?.id === 'adapter' && (
          <div className="onboarding-adapter-section">
            {!detecting && (
              <button className="onboarding-detect-btn" type="button" onClick={runDetect}>
                {detectResults ? t('onboarding.rescan') : t('onboarding.scanCLI')}
              </button>
            )}
            {detecting && <p className="onboarding-scanning">{t('onboarding.scanning')}</p>}

            {detectResults && foundCount > 0 && (
              <ul className="onboarding-detect-list">
                {detectResults.filter((r) => r.found).map((r) => {
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

            {detectResults && foundCount === 0 && (
              <p className="onboarding-install-hint">
                {t('onboarding.noCLI')}
                <br />
                <code>npm install -g @anthropic-ai/claude-code</code>
              </p>
            )}

            {detectResults && foundCount > 0 && (
              <p className="onboarding-detect-summary">
                {t('onboarding.cliFound', { foundCount, enabledCount: enabledIds.size })}
              </p>
            )}

            {/* Local model detection results */}
            {localModelResults && localModelResults.length > 0 && (
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

            {detectResults && (
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

        {/* 導航按鈕列 */}
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
