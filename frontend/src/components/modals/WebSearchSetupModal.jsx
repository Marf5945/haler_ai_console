import {defaultWebSearchProviderOptions, openExternal} from '../../lib/appShared';

export default function WebSearchSetupModal({setup, config, error, onChange, onCancel, onClear, onSubmit}) {
  const options = Array.isArray(setup?.options) && setup.options.length ? setup.options : defaultWebSearchProviderOptions();
  const selected = options.find((option) => option.id === setup.providerId) || options[0];
  const fields = Array.isArray(selected?.fields) ? selected.fields : ['api_key'];
  const update = (key, value) => onChange({...setup, [key]: value});
  const chooseProvider = (providerId) => onChange({
    ...setup,
    providerId,
    apiKey: '',
    cx: '',
  });
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label="Web search setup">
      <section className="reference-link-modal web-search-setup-modal">
        <h4>內建網路搜尋</h4>
        <p className="reference-link-hint">
          選擇搜尋供應商並輸入 API 欄位。金鑰會寫入 Windows DPAPI 加密 credential store，不會交給 Agent 或顯示在提示詞中。
        </p>
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
        <label>
          <span>API Key</span>
          <input
            type="password"
            value={setup.apiKey || ''}
            onChange={(event) => update('apiKey', event.target.value)}
            placeholder={selected?.id === 'tavily' ? 'tvly-...' : 'API key'}
          />
        </label>
        {fields.includes('cx') && (
          <label>
            <span>Search Engine ID (cx)</span>
            <input
              type="text"
              value={setup.cx || ''}
              onChange={(event) => update('cx', event.target.value)}
              placeholder="Programmable Search Engine ID"
            />
          </label>
        )}
        {config?.configured && (
          <small className="reference-link-hint">
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
