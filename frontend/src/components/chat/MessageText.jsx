import React from 'react';
import {RegisterURLOccurrence} from '../../../wailsjs/go/main/App';
import {MESSAGE_URL_RE} from '../../lib/appShared';

export default function MessageText({ text, kind, onInjectText, sessionId }) {
  const urls = React.useMemo(() => {
    const found = text.match(MESSAGE_URL_RE);
    return found ? Array.from(new Set(found)) : [];
  }, [text]);

  // 閉合洗白防護：AI 回答裡的 URL 記成 llm_extracted。
  React.useEffect(() => {
    if (kind !== 'ai' || urls.length === 0) return;
    urls.forEach((u) => {
      try {
        RegisterURLOccurrence(u, 'llm_extracted', sessionId || '', '').catch(() => {});
      } catch { /* binding 缺失（測試環境）忽略 */ }
    });
  }, [kind, urls, sessionId]);

  if (urls.length === 0) return <>{text}</>;

  // 以 URL 切段，URL 段渲染成連結 + chip + 讀取按鈕。
  const parts = text.split(MESSAGE_URL_RE);
  const matches = text.match(MESSAGE_URL_RE) || [];
  return (
    <>
      {parts.map((seg, i) => (
        <React.Fragment key={i}>
          {seg}
          {i < matches.length && (
            <span className="url-token">
              <a
                href={matches[i]}
                onClick={(e) => { e.preventDefault(); onInjectText?.(matches[i]); }}
                title="點擊選擇讀取或開啟"
              >{matches[i]}</a>
              {kind === 'ai' && <span className="url-source-chip url-source-llm" title="此網址由 LLM 擷取，內容未經信任">LLM 擷取</span>}
              <button
                type="button"
                className="url-read-btn"
                onClick={(e) => { e.stopPropagation(); onInjectText?.(`讀取 ${matches[i]} 的內容`); }}
              >讀取內容</button>
            </span>
          )}
        </React.Fragment>
      ))}
    </>
  );
}
