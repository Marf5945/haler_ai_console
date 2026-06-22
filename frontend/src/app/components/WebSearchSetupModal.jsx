import {openExternal} from '../../lib/openExternal';
import {defaultWebSearchProviderOptions} from '../../lib/appHelpers';

export default function WebSearchSetupModal({setup, config, error, onChange, onCancel, onClear, onSubmit}) {
  const options = Array.isArray(setup?.options) && setup.options.length ? setup.options : defaultWebSearchProviderOptions();
  const selected = options.find((option) => option.id === setup.providerId) || options[0];
  const update = (key, value) => onChange({...setup, [key]: value});
  const setupGuide = Array.isArray(selected?.setup_guide) ? selected.setup_guide : [];
  const chooseProvider = (providerId) => onChange({
    ...setup,
    providerId,
    apiKey: '',
  });
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label="Web search setup">
      <section className="reference-link-modal web-search-setup-modal">
        <header className="web-search-setup-header">
          <small className="web-search-setup-eyebrow">Secure Provider Setup</small>
          <h4>內建網路搜尋</h4>
          <p className="reference-link-hint web-search-setup-subtitle">
            選擇搜尋來源並貼上 API key 後即可啟用。金鑰只會寫入本機 Windows DPAPI 憑證庫，不會交給 Agent，也不會出現在提示詞中。
          </p>
        </header>
        <div className="web-search-provider-grid">
          {options.map((option) => (
            <button
              key={option.id}
              type="button"
              className={option.id === setup.providerId ? 'web-search-provider-card web-search-provider-active' : 'web-search-provider-card'}
              onClick={() => chooseProvider(option.id)}
            >
              <strong>{option.name}</strong>
              <small>{option.free_tier_hint || option.id}</small>
            </button>
          ))}
        </div>
        <section className="web-search-provider-detail" aria-label="provider details">
          <div className="web-search-provider-detail-head">
            <div>
              <strong>{selected?.name || 'Provider'}</strong>
              <small>{selected?.free_tier_hint || ''}</small>
            </div>
            {selected?.best_for && <span className="web-search-provider-bestfor">適合：{selected.best_for}</span>}
          </div>
        </section>
        <label>
          <span>API Key / Token</span>
          <input
            type="password"
            value={setup.apiKey || ''}
            onChange={(event) => update('apiKey', event.target.value)}
            placeholder={selected?.api_key_placeholder || 'API key'}
          />
        </label>
        {setupGuide.length > 0 && (
          <div className="reference-link-hint web-search-setup-guide">
            <strong>設定方式</strong>
            <ol>
              {setupGuide.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
            {selected?.usage_hint && <div className="web-search-setup-example">{selected.usage_hint}</div>}
          </div>
        )}
        {config?.configured && (
          <small className="reference-link-hint web-search-setup-status">
            目前已設定：{config.provider || config.provider_id}
          </small>
        )}
        {error && <div className="rb-error">{error}</div>}
        <div className="rb-actions">
          {selected?.docs_url && (
            <button type="button" onClick={() => openExternal(selected.docs_url)}>官方文件</button>
          )}
          <button type="button" onClick={onSubmit}>儲存</button>
          <button type="button" onClick={onClear}>清除設定</button>
          <button type="button" className="rb-cancel-btn" onClick={onCancel}>取消</button>
        </div>
      </section>
    </div>
  );
}
