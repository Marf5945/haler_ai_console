import {t as _t} from '../../locales/useI18n';
import {getStateTokenDescriptions} from '../../lib/appShared';

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
