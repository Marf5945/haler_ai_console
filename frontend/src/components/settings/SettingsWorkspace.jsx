import useI18n from '../../locales/useI18n';
import BrowserSettingsSection from './BrowserSettingsSection';
import ControlledTrustSettings from './ControlledTrustSettings';
import PersonaSettingsDrawer from './PersonaSettingsDrawer';
import SkillContextSettingsSection from './SkillContextSettingsSection';
import StateTokenLegend from './StateTokenLegend';

export default function SettingsWorkspace(props) {
  const t = useI18n(s => s.t);
  const {
    trustDomClickEnabled, activeOverrides, trustedSessionActive, deviceProfile,
    onToggleTrustDomClick, onEnableOverride, onEnableWorkflowTrust,
    onDisableOverride, onEnableTrustedSession, onLoadDeviceProfile,
    browserPref, panelSettings, adapterList, summaryModelSettings, summaryModelScan,
    onSummaryModelChange, onSummaryModelRescan, voiceState, voiceInstallBusy,
    onVoiceSettingsChange, onVoiceSettingsRefresh, onVoiceModelInstall, onVoiceModelRemove,
    onVoiceDebugClear, onSaveBrowserPref, onPreviewStyleDiff,
    ...personaProps
  } = props;

  return (
    <main className="settings-workspace" aria-label={t('settings.settingsWorkspace')}>
      <PersonaSettingsDrawer {...personaProps}/>
      <ControlledTrustSettings
        trustDomClickEnabled={trustDomClickEnabled}
        activeOverrides={activeOverrides}
        trustedSessionActive={trustedSessionActive}
        deviceProfile={deviceProfile}
        onToggleTrustDomClick={onToggleTrustDomClick}
        onEnableOverride={onEnableOverride}
        onEnableWorkflowTrust={onEnableWorkflowTrust}
        onDisableOverride={onDisableOverride}
        onEnableTrustedSession={onEnableTrustedSession}
        onLoadDeviceProfile={onLoadDeviceProfile}
      />
      <BrowserSettingsSection
        browserPref={browserPref}
        panelSettings={panelSettings}
        adapterList={adapterList}
        summaryModelSettings={summaryModelSettings}
        summaryModelScan={summaryModelScan}
        onSummaryModelChange={onSummaryModelChange}
        onSummaryModelRescan={onSummaryModelRescan}
        voiceState={voiceState}
        voiceInstallBusy={voiceInstallBusy}
        onVoiceSettingsChange={onVoiceSettingsChange}
        onVoiceSettingsRefresh={onVoiceSettingsRefresh}
        onVoiceModelInstall={onVoiceModelInstall}
        onVoiceModelRemove={onVoiceModelRemove}
        onVoiceDebugClear={onVoiceDebugClear}
        onSaveBrowserPref={onSaveBrowserPref}
        onPreviewStyleDiff={onPreviewStyleDiff}
      />
      <StateTokenLegend />
      {/* #I-207: Skill Context Orchestration 靜態說明（Settings 頁常駐） */}
      <SkillContextSettingsSection />
    </main>
  );
}
