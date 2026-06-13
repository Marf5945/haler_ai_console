import {useState} from 'react';
import {t as _t} from '../../locales/useI18n';

export default function ControlledTrustSettings({
  trustDomClickEnabled, activeOverrides, trustedSessionActive, deviceProfile,
  onToggleTrustDomClick, onEnableOverride, onEnableWorkflowTrust,
  onDisableOverride, onEnableTrustedSession, onLoadDeviceProfile,
}) {
  const [overrideScope, setOverrideScope] = useState('');
  const [overrideRisk, setOverrideRisk] = useState('low');
  const [workflowHours, setWorkflowHours] = useState(1);

  return (
    <div className="trust-settings-section">
      <h4>{_t('controlledTrust.title')}</h4>

      {/* #I-304 Trust DOM & Click */}
      <div className="trust-toggle-row">
        <span>{_t('controlledTrust.trustDomClick')}</span>
        <button
          type="button"
          className={trustDomClickEnabled ? 'trust-active' : ''}
          onClick={() => onToggleTrustDomClick(!trustDomClickEnabled)}
        >
          {trustDomClickEnabled ? _t('controlledTrust.enabled') : _t('controlledTrust.disabled')}
        </button>
      </div>

      {/* #I-302 Trusted Session Scope */}
      <div className="trust-toggle-row">
        <span>Trusted Session Scope</span>
        <button
          type="button"
          className={trustedSessionActive ? 'trust-active' : ''}
          onClick={() => onEnableTrustedSession('', '')}
          disabled={trustedSessionActive}
        >
          {trustedSessionActive ? _t('controlledTrust.enabled') : _t('controlledTrust.enable')}
        </button>
      </div>
      {trustedSessionActive && (
        <small style={{display: 'block', fontSize: '11px', opacity: 0.6, padding: '2px 0 6px'}}>
          {_t('controlledTrust.mergeHint')}
        </small>
      )}

      {/* #I-301 Contextual Risk Override */}
      <div className="trust-toggle-row" style={{flexWrap: 'wrap', gap: '6px'}}>
        <span>{_t('controlledTrust.contextualOverride')}</span>
        <div style={{display: 'flex', gap: '4px', alignItems: 'center', flexWrap: 'wrap'}}>
          <input
            type="text"
            placeholder="scope JSON"
            value={overrideScope}
            onChange={(e) => setOverrideScope(e.target.value)}
            style={{width: '140px', fontSize: '11px', padding: '3px 6px', borderRadius: '4px', border: '1px solid var(--surface-2)', background: 'var(--bg)', color: 'var(--text)'}}
          />
          <select
            value={overrideRisk}
            onChange={(e) => setOverrideRisk(e.target.value)}
            style={{fontSize: '11px', padding: '3px 6px', borderRadius: '4px', border: '1px solid var(--surface-2)', background: 'var(--bg)', color: 'var(--text)'}}
          >
            <option value="low">low</option>
            <option value="medium">medium</option>
            <option value="high_non_destructive">high_non_destructive</option>
          </select>
          <button type="button" onClick={() => { if (overrideScope) onEnableOverride(overrideScope, overrideRisk); }}>{_t('controlledTrust.enable')}</button>
        </div>
      </div>

      {/* Workflow Trust shortcut */}
      <div className="trust-toggle-row" style={{flexWrap: 'wrap', gap: '6px'}}>
        <span>{_t('controlledTrust.workflowTrust')}</span>
        <div style={{display: 'flex', gap: '4px', alignItems: 'center'}}>
          <input
            type="number"
            min="1"
            max="999"
            value={workflowHours}
            onChange={(e) => setWorkflowHours(Math.max(1, Math.min(999, Number(e.target.value) || 1)))}
            style={{width: '50px', fontSize: '11px', padding: '3px 6px', borderRadius: '4px', border: '1px solid var(--surface-2)', background: 'var(--bg)', color: 'var(--text)'}}
          />
          <span style={{fontSize: '11px'}}>{_t('controlledTrust.hours')}</span>
          <button type="button" onClick={() => { if (overrideScope) onEnableWorkflowTrust(overrideScope, overrideRisk, workflowHours); }}>{_t('controlledTrust.enable')}</button>
        </div>
      </div>

      {/* Active overrides list */}
      {activeOverrides.length > 0 && (
        <div className="trust-override-list">
          {activeOverrides.map((o) => (
            <div className="trust-override-item" key={o.id}>
              <span>{o.allowedRisk}{o.hours ? ` · ${o.hours}h` : ''}</span>
              <button type="button" onClick={() => onDisableOverride(o.id)}>{_t('controlledTrust.disabled')}</button>
            </div>
          ))}
        </div>
      )}

      {/* #I-305 Device Trust Profile */}
      <div className="trust-toggle-row" style={{marginTop: '8px'}}>
        <span>{_t('controlledTrust.deviceTrust')}</span>
        <button type="button" onClick={() => onLoadDeviceProfile('default')}>
          {deviceProfile ? _t('controlledTrust.reload') : _t('controlledTrust.load')}
        </button>
      </div>
      {deviceProfile && (
        <div style={{fontSize: '11px', opacity: 0.7, padding: '4px 0'}}>
          <span>Profile: {deviceProfile.id || 'default'}</span>
          {deviceProfile.devicePenalty != null && <span> · penalty: {deviceProfile.devicePenalty}</span>}
        </div>
      )}
    </div>
  );
}
