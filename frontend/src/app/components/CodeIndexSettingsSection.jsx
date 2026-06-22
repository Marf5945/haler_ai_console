import {useState} from 'react';

export default function CodeIndexSettingsSection({
  onRebuild = async () => null,
  onSearch = async () => [],
  onBuildContext = async () => null,
}) {
  const [query, setQuery] = useState('BuildLLMContext');
  const [busy, setBusy] = useState('');
  const [status, setStatus] = useState('');
  const [matches, setMatches] = useState([]);
  const [contextResult, setContextResult] = useState(null);

  async function rebuild() {
    setBusy('rebuild');
    setStatus('');
    try {
      const result = await onRebuild();
      setStatus(`indexed=${result?.indexed_files || 0}, reused=${result?.reused_files || 0}, parse_errors=${result?.parse_errors || 0}`);
    } catch (err) {
      setStatus(err?.message || String(err));
    } finally {
      setBusy('');
    }
  }

  async function search() {
    const nextQuery = query.trim();
    if (!nextQuery) return;
    setBusy('search');
    setStatus('');
    setContextResult(null);
    try {
      const result = await onSearch(nextQuery, 8);
      setMatches(Array.isArray(result) ? result : []);
    } catch (err) {
      setStatus(err?.message || String(err));
    } finally {
      setBusy('');
    }
  }

  async function buildContext() {
    const nextQuery = query.trim();
    if (!nextQuery) return;
    setBusy('context');
    setStatus('');
    try {
      const result = await onBuildContext(nextQuery, false);
      setContextResult(result || null);
      setMatches(Array.isArray(result?.sections) ? result.sections : []);
    } catch (err) {
      setStatus(err?.message || String(err));
    } finally {
      setBusy('');
    }
  }

  const sectionCount = contextResult?.sections?.length || 0;
  const blockCount = contextResult?.payload?.content_blocks?.length || 0;

  return (
    <div className="skill-settings-section code-index-settings-section">
      <h4>Code Section Index</h4>
      <div className="code-index-toolbar">
        <input
          aria-label="Code index query"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') search();
          }}
          placeholder="symbol, tag, path, summary"
        />
        <button type="button" onClick={search} disabled={!!busy}>{busy === 'search' ? 'Searching' : 'Search'}</button>
        <button type="button" onClick={buildContext} disabled={!!busy}>{busy === 'context' ? 'Building' : 'Build context'}</button>
        <button type="button" onClick={rebuild} disabled={!!busy}>{busy === 'rebuild' ? 'Rebuilding' : 'Rebuild'}</button>
      </div>
      {status && <small className="code-index-status">{status}</small>}
      {contextResult && (
        <small className="code-index-status">context sections={sectionCount}, blocks={blockCount}</small>
      )}
      {matches.length > 0 && (
        <ul className="code-index-results">
          {matches.slice(0, 8).map((match) => {
            const section = match.section || {};
            const tags = match.tags || [];
            return (
              <li key={`${section.id || section.symbol_name}-${match.score}`}>
                <strong>{section.symbol_name || 'unknown'}</strong>
                <span>{section.file_path}:{section.start_line}</span>
                <small>{tags.join(', ') || section.summary || 'no tags'}</small>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
