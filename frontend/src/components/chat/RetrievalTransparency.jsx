

export default function RetrievalTransparency({isScanning = false, sources = []}) {
  if (!isScanning) return null;
  return (
    <div className="retrieval-scanning">
      <span className="scan-dot" />
      <span className="scan-label">scanning:</span>
      {sources.map((source) => (
        <span key={source} className="scan-source">{source}</span>
      ))}
    </div>
  );
}
