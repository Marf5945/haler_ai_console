import React from 'react';
import {RegisterURLOccurrence} from '../../../wailsjs/go/main/App';
import { tForLanguage } from '../../locales/useI18n';

// SEC-06: 訊息內 URL 偵測（與後端 urlInTextRe 對齊）。
const MESSAGE_URL_RE = /https?:\/\/[^\s'"<>）)】\]]+/g;
const MEMORY_TAG_RE = /\[?([SD]-\d+)(?::[^\]]*)?\]?/g;
const MEMORY_TAG_ONLY_RE = /^\[?([SD]-\d+)(?::[^\]]*)?\]?$/;
const COLLAPSE_LINE_LIMIT = 3;
const COLLAPSE_CHAR_LIMIT = 520;

// MessageText 把訊息文字 linkify：URL 旁加來源 chip + 讀取按鈕，記憶 tag 旁加展開入口。
// AI 訊息內的 URL 在渲染時記為 llm_extracted（閉合洗白防護）。
export default function MessageText({text, kind, onInjectText, sessionId, chatLocale}) {
  const tChat = React.useCallback((key, params) => (
    tForLanguage(chatLocale || 'zh-TW', key, params)
  ), [chatLocale]);
  const [expanded, setExpanded] = React.useState(false);
  const collapseInfo = React.useMemo(() => buildCollapseInfo(text), [text]);
  const renderText = collapseInfo.shouldCollapse && !expanded ? collapseInfo.preview : text;
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

  const body = renderRichText(renderText, {
    urls,
    memoryTags,
    kind,
    sessionId,
    onInjectText,
    tChat,
  });

  return (
    <>
      <span className={collapseInfo.shouldCollapse && !expanded ? 'message-text-preview' : undefined}>
        {body}
      </span>
      {collapseInfo.shouldCollapse && (
        <button
          type="button"
          className="message-expand-toggle"
          onClick={(event) => {
            event.stopPropagation();
            setExpanded((value) => !value);
          }}
        >
          {expanded ? tChat('chatSystem.collapseText') : tChat('chatSystem.expandText')}
        </button>
      )}
    </>
  );
}

function renderRichText(text, {kind, onInjectText, tChat}) {
  if (!new RegExp(MESSAGE_URL_RE.source).test(text) && !new RegExp(MEMORY_TAG_RE.source).test(text)) return <>{text}</>;
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
              onInjectText?.(tChat('chatSystem.expandMemoryPrompt', { tag }));
            }}
          >{tChat('chatSystem.expandText')}</button>
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
            title={tChat('chatSystem.urlSelectTitle')}
          >{raw}</a>
          {kind === 'ai' && (
            <span className="url-source-chip url-source-llm" title={tChat('chatSystem.urlLLMSourceTitle')}>
              {tChat('chatSystem.urlLLMSource')}
            </span>
          )}
          <button
            type="button"
            className="url-read-btn"
            onClick={(event) => {
              event.stopPropagation();
              onInjectText?.(tChat('chatSystem.readURLPrompt', { url: raw }));
            }}
          >{tChat('chatSystem.readContent')}</button>
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

function buildCollapseInfo(text) {
  const raw = String(text || '');
  const lines = raw.split(/\r?\n/);
  const shouldCollapse = lines.length > COLLAPSE_LINE_LIMIT || raw.length > COLLAPSE_CHAR_LIMIT;
  if (!shouldCollapse) return { shouldCollapse: false, preview: raw };
  const previewLines = lines.slice(0, COLLAPSE_LINE_LIMIT);
  let preview = previewLines.join('\n').trimEnd();
  if (preview.length > COLLAPSE_CHAR_LIMIT) {
    preview = `${preview.slice(0, COLLAPSE_CHAR_LIMIT).trimEnd()}...`;
  }
  return { shouldCollapse: true, preview };
}
