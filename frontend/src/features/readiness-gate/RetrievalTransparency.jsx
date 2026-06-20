/**
 * RetrievalTransparency — 掃描動畫 + 來源標籤
 *
 * §12A.6: 當 RETRIEVE_MORE 被觸發時，在對話訊息流中 inline 顯示。
 * 預設模式僅顯示粗分類來源類別（project docs、uploaded files 等）。
 * 標籤必須通過 Safe Display Filter，禁止顯示完整路徑或敏感檔名。
 *
 * Props:
 *   isScanning — 是否正在掃描中
 *   sources    — 來源標籤陣列 ['project docs', 'uploaded files']
 */
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
