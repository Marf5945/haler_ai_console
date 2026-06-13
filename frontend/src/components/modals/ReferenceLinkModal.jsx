import useI18n from '../../locales/useI18n';

export default function ReferenceLinkModal({value, linkPreview, error, suggestions, onCancel, onChange, onUseSuggestion, onPreview, onConfirm}) {
  const {t} = useI18n();
  const isOllamaModelLibraryPreview = linkPreview?.link_type === 'adapter_candidate' &&
    /ollama|模型庫|本機模型|blobs|manifests|\.ollama[\\/]+models/i.test(`${value || ''} ${linkPreview?.url || ''} ${linkPreview?.reason || ''}`);
  const getLinkTypeLabel = () => ({
    external_service: t('link.externalService'),
    adapter_candidate: isOllamaModelLibraryPreview ? '本地模型 Adapter' : 'CLI Adapter',
    llm_provider_candidate: t('link.llmApiInterface'),
    documentation: t('link.docLinkShort'),
    unsupported: t('link.unsupported'),
  });
  const linkTypeLabel = getLinkTypeLabel();
  // i18n: link
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={t('link.addRefLinkLabel')}>
      <section className="reference-link-modal">
        <label>
          <span>{t('link.refLinkSpan')}</span>
          <div>
            <input
              autoFocus
              placeholder={t('link.inputPlaceholder')}
              value={value}
              onChange={(event) => onChange(event.target.value)}
            />
            {!linkPreview ? (
              <button type="button" onClick={onPreview}>{t('link.preview')}</button>
            ) : (
              <button type="button" onClick={onConfirm}>{t('link.confirm')}</button>
            )}
          </div>
          <small className="reference-link-hint">
            {t('link.inputHint')}
          </small>
        </label>
        {linkPreview?.valid && linkPreview.type === 'remote_bridge' && (
          <p className="reference-link-preview reference-link-bridge">
            {t('link.createCommsQuestion')} <strong>{linkPreview.hint_label}</strong>
            <small>{linkPreview.setup_required ? t('link.setupRequired') : t('link.testAndName')}</small>
          </p>
        )}
        {linkPreview?.valid && linkPreview.type !== 'remote_bridge' && (
          <p className="reference-link-preview">
            {t('link.typeLabel')}<strong>{linkTypeLabel[linkPreview.link_type] || linkPreview.link_type}</strong>
            {linkPreview.link_type === 'llm_provider_candidate' && (
              <small>
                {t('link.detectedProvider', { name: linkPreview.provider_name || 'LLM/API Provider', baseUrl: linkPreview.base_url ? t('link.defaultApi', { url: linkPreview.base_url }) : '' })}
              </small>
            )}
            {linkPreview.link_type === 'documentation' && <small>{t('link.docOnlyRef')}</small>}
            {linkPreview.link_type === 'adapter_candidate' && linkPreview.reason && <small>（{linkPreview.reason}）</small>}
          </p>
        )}
        {error && <p className="reference-link-error">{error}</p>}
        {suggestions?.length > 0 && (
          <div className="reference-link-suggestions">
            <strong>{t('link.suggestionsTitle')}</strong>
            {suggestions.map((item) => (
              <button
                type="button"
                key={`${item.name}-${item.path}`}
                onClick={() => onUseSuggestion(item.path)}
                title={item.path}
              >
                <span>{item.detected ? t('link.foundItem', { name: item.name }) : t('link.exampleItem', { name: item.name })}</span>
                <code>{item.path}</code>
              </button>
            ))}
          </div>
        )}
        <button className="reference-link-cancel" type="button" onClick={onCancel}>{t('common.cancel')}</button>
      </section>
    </div>
  );
}
