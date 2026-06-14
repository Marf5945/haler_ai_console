import React from 'react';
import {RegisterURLOccurrence} from '../../../wailsjs/go/main/App';

// SEC-06: 訊息內 URL 偵測（與後端 urlInTextRe 對齊）。
const MESSAGE_URL_RE = /https?:\/\/[^\s'"<>）)】\]]+/g;
const MEMORY_TAG_RE = /\[?([SD]-\d+)(?::[^\]]*)?\]?/g;
const MEMORY_TAG_ONLY_RE = /^\[?([SD]-\d+)(?::[^\]]*)?\]?$/;

// MessageText 把訊息文字 linkify：URL 旁加來源 chip + 讀取按鈕，記憶 tag 旁加展開入口。
// AI 訊息內的 URL 在渲染時記為 llm_extracted（閉合洗白防護）。
export default function MessageText({text, kind, onInjectText, sessionId}) {
  const urls = React.useMemo(() => {
    const found = text.match(MESSAGE_URL_RE);
    return found ? Array.from(new Set(found)) : [];
  }, [text]);
  const memoryTags = React.useMemo(() => {
    const tags = [];
    for (const match of text.matchAll(MEMORY_TAG_RE)) {
      if (match[1] && !tags.includes(match[1])) tags.push(match[1]);
    }
    return tags;
  }, [text]);

  React.useEffect(() => {
    if (kind !== 'ai' || urls.length === 0) return;
    urls.forEach((url) => {
      try {
        RegisterURLOccurrence(url, 'llm_extracted', sessionId || '', '').catch(() => {});
      } catch {
        // binding 缺失（測試環境）忽略
      }
    });
  }, [kind, urls, sessionId]);

  if (urls.length === 0 && memoryTags.length === 0) return <>{text}</>;

  const tokenRe = new RegExp(`${MESSAGE_URL_RE.source}|${MEMORY_TAG_RE.source}`, 'g');
  const nodes = [];
  let lastIndex = 0;
  for (const match of text.matchAll(tokenRe)) {
    if (match.index > lastIndex) nodes.push(text.slice(lastIndex, match.index));
    const raw = match[0];
    const tag = raw.match(MEMORY_TAG_ONLY_RE)?.[1];
    if (tag) {
      nodes.push(
        <span className="memory-tag-token" key={`mem-${match.index}-${tag}`}>
          <code>{raw}</code>
          <button
            type="button"
            className="memory-expand-btn"
            onClick={(event) => {
              event.stopPropagation();
              onInjectText?.(`展開 ${tag} 的細節`);
            }}
          >展開</button>
        </span>
      );
    } else {
      nodes.push(
        <span className="url-token" key={`url-${match.index}-${raw}`}>
          <a
            href={raw}
            onClick={(event) => {
              event.preventDefault();
              onInjectText?.(raw);
            }}
            title="點擊選擇讀取或開啟"
          >{raw}</a>
          {kind === 'ai' && (
            <span className="url-source-chip url-source-llm" title="此網址由 LLM 擷取，內容未經信任">
              LLM 擷取
            </span>
          )}
          <button
            type="button"
            className="url-read-btn"
            onClick={(event) => {
              event.stopPropagation();
              onInjectText?.(`讀取 ${raw} 的內容`);
            }}
          >讀取內容</button>
        </span>
      );
    }
    lastIndex = match.index + raw.length;
  }
  if (lastIndex < text.length) nodes.push(text.slice(lastIndex));

  return (
    <>
      {nodes.map((node, index) => <React.Fragment key={index}>{node}</React.Fragment>)}
    </>
  );
}
