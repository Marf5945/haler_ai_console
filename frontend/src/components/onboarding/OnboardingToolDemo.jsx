import useI18n from '../../locales/useI18n';

export default function OnboardingToolDemo() {
  const t = useI18n(s => s.t);
  // i18n: onboarding
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
    </div>
  );
}
