import { t as _t } from '../../locales/useI18n';

function getStateTokenDescriptions() {
  return [
    {color: 'var(--amber, #f5a623)', label: _t('stateToken.highRiskLabel'), desc: _t('stateToken.highRiskDesc')},
    {color: 'var(--msg, #7f2d20)', label: _t('stateToken.interruptedLabel'), desc: _t('stateToken.interruptedDesc')},
    {color: '#4caf50', label: _t('stateToken.normalLabel'), desc: _t('stateToken.normalDesc')},
    {color: 'var(--muted, #bbb5a5)', label: _t('stateToken.idleLabel'), desc: _t('stateToken.idleDesc')},
  ];
}

export default function StateTokenLegend() {
  const descriptions = getStateTokenDescriptions();
  return (
    <div className="state-token-legend">
      <h4>{_t('stateToken.legendTitle')}</h4>
      <div className="state-token-list">
        {descriptions.map((item) => (
          <div className="state-token-row" key={item.label}>
            <span className="state-token-dot" style={{background: item.color}} />
            <div className="state-token-info">
              <span className="state-token-label">{item.label}</span>
              <span className="state-token-desc">{item.desc}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
