import React from 'react';
import MessageText from './MessageText';
import { tForLanguage } from '../../locales/useI18n';

export default function MessageRow({
  message,
  domMessageId,
  displayText,
  kind,
  isActive,
  index,
  summaryLabel,
  deleteLabel,
  onActivate,
  onDelete,
  onSummarizeSearch,
  onExportSearchSummary,
  onInjectText,
  sessionId,
  previousMessage,
  chatLocale,
}) {
  const isLocalSearch = isLocalSearchMessage(message);
  const isWebSearch = isWebSearchMessage(message, previousMessage);
  const isSearchSummary = isLocalSearch || isWebSearch;
  const tChat = React.useCallback((key, params) => (
    tForLanguage(chatLocale || 'zh-TW', key, params)
  ), [chatLocale]);
  const searchSummary = React.useMemo(
    () => buildSearchSummaryCard(message, previousMessage, tChat),
    [message, previousMessage, tChat],
  );

  // 詳情 popover：點卡片浮出「網址＋擷取內容」，Esc 或點外關閉。
  const [searchSummaryOpen, setSearchSummaryOpen] = React.useState(false);
  const searchSummaryRef = React.useRef(null);

  React.useEffect(() => {
    if (!searchSummaryOpen) return undefined;
    const onKeyDown = (event) => {
      if (event.key === 'Escape') setSearchSummaryOpen(false);
    };
    const onPointerDown = (event) => {
      if (!searchSummaryRef.current?.contains(event.target)) setSearchSummaryOpen(false);
    };
    document.addEventListener('keydown', onKeyDown);
    document.addEventListener('pointerdown', onPointerDown);
    return () => {
      document.removeEventListener('keydown', onKeyDown);
      document.removeEventListener('pointerdown', onPointerDown);
    };
  }, [searchSummaryOpen]);

  function startSearchSummaryExport(event) {
    event?.stopPropagation?.();
    if (event?.dataTransfer) {
      event.dataTransfer.effectAllowed = 'copy';
      try { event.dataTransfer.setData('application/x-ai-console-search-summary', String(index)); } catch (_) {}
    }
    // 把已算好的 card 往上傳：匯出的就是 Popover 預覽的同一份，避免兩邊內容漂移。
    void onExportSearchSummary?.(searchSummary);
  }

  return (
    <article
      className={`message-row message-${kind} ${isSearchSummary ? 'message-search-summary-row' : ''} ${isActive ? 'message-row-active' : ''}`}
      onClick={onActivate}
    >
      {isSearchSummary ? (
        <div className="search-summary-shell" ref={searchSummaryRef}>
          {/* 用 div 而非 button：WebKit 從 button 起原生 file-promise 拖曳不可靠，Finder 常收不到 drop；div 比照可運作的 reference 卡片。
              點卡片＝啟用該列（摘要/刪除按鈕浮出）＋開詳情 popover（網址＋擷取內容）。 */}
          <div
            className="search-summary-card"
            role="button"
            tabIndex={0}
            draggable
            onClick={(event) => {
              event.stopPropagation();
              onActivate?.(event);
              setSearchSummaryOpen((open) => !open);
            }}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                event.stopPropagation();
                onActivate?.(event);
                setSearchSummaryOpen((open) => !open);
              }
            }}
            onDragStart={startSearchSummaryExport}
          >
            <span className="search-summary-icon">⌕</span>
            <span className="search-summary-copy">
              <strong className="search-summary-title">
                {renderSearchSummaryTitle(searchSummary.title, searchSummary.query)}
              </strong>
              <small>{searchSummary.keywordText}</small>
            </span>
            <span className="search-summary-count">{searchSummary.countLabel}</span>
          </div>
          {searchSummaryOpen && (
            <div className="search-summary-popover" role="dialog" aria-label={tChat('chatSystem.searchDetails')}>
              <strong className="search-summary-popover-title">{searchSummary.filename}</strong>
              <pre className="search-summary-md-preview">{searchSummary.markdown}</pre>
            </div>
          )}
        </div>
      ) : (
        <div className="message-text" data-message-id={domMessageId || `m_${index}`}>
          <MessageText
            text={displayText}
            kind={kind}
            onInjectText={onInjectText}
            sessionId={sessionId}
            chatLocale={chatLocale}
          />
        </div>
      )}
      {isSearchSummary && (
        <button
          type="button"
          className="message-action"
          onClick={(event) => {
            event.stopPropagation();
            onSummarizeSearch?.(message.replace(/^Ai:/, ''));
          }}
        >
          {summaryLabel}
        </button>
      )}
      {isSearchSummary && (
        <button
          type="button"
          className="message-delete-action"
          onClick={(event) => {
            event.stopPropagation();
            onDelete(index);
          }}
        >
          {deleteLabel}
        </button>
      )}
      {!isSearchSummary && <button
        type="button"
        className="message-delete-action"
        onClick={(event) => {
          event.stopPropagation();
          onDelete(index);
        }}
      >
        {deleteLabel}
      </button>}
    </article>
  );
}

function buildSearchSummaryCard(text, previousMessage = '', tChat = tForLanguage.bind(null, 'zh-TW')) {
  const raw = String(text || '').replace(/^Ai:/, '').trim();
  if (isWebSearchMessage(text, previousMessage)) {
    const purpose = extractSearchPurpose(previousMessage) || extractWebSearchQuery(raw) || tChat('chatSystem.searchResults');
    const title = tChat('chatSystem.webSearchCompleted', { query: purpose });
    const sourceCount = countWebSearchSources(raw);
    return {
      title,
      query: purpose,
      keywordText: tChat('chatSystem.searchPurpose', { query: purpose }),
      countLabel: sourceCount > 0 ? tChat('chatSystem.sourceCount', { count: sourceCount }) : tChat('chatSystem.summary'),
      filename: `${sanitizeSearchSummaryFilename(tChat('chatSystem.webSearchSummaryFile', { query: purpose }))}.md`,
      markdown: raw.startsWith('#') ? `${raw}\n` : `# ${title}\n\n${raw}\n`,
    };
  }
  const lines = raw.split('\n').map((line) => line.trim()).filter(Boolean);
  const firstLine = lines[0] || '本機搜尋結果';
  const quoted = firstLine.match(/[「"“](.*?)[」"”]/)?.[1];
  const keywordLine = lines.find((line) => /^關鍵字[:：]/.test(line));
  const query = quoted || keywordLine?.replace(/^關鍵字[:：]\s*/, '').trim() || '';
  const count = lines.filter((line) => /^\d+[.、)]\s*/.test(line)).length;
  const title = firstLine.replace(/^本機搜尋[:：]?\s*/, '').slice(0, 64) || '本機搜尋摘要';
  const exportTitle = firstLine.replace(/^本機搜尋[:：]?\s*/, '本機搜尋摘要：').slice(0, 80) || '本機搜尋摘要';
  const filename = `${sanitizeSearchSummaryFilename(exportTitle)}.md`;
  const markdown = raw.startsWith('#') ? `${raw}\n` : `# ${exportTitle}\n\n${raw}\n`;
  return {
    title,
    query,
    keywordText: query ? `關鍵字：${query}` : (lines[1] || '搜尋細節'),
    countLabel: count > 0 ? `${count} 筆` : '摘要',
    filename,
    markdown,
  };
}

function isLocalSearchMessage(text) {
  const raw = String(text || '');
  return raw.startsWith('Ai:本機搜尋')
    || raw.startsWith('Ai:本機資料裡找不到')
    || raw.startsWith('Ai:本機資料沒有找到')
    || raw.startsWith('Ai:Local search found')
    || raw.startsWith("Ai:I couldn't find")
    || raw.startsWith('Ai:A pesquisa local')
    || raw.startsWith('Ai:Não encontrei');
}

function isWebSearchMessage(text, previousMessage = '') {
  const raw = String(text || '');
  if (!raw.startsWith('Ai:')) return false;
  const body = raw.replace(/^Ai:/, '').trim();
  if (!body) return false;
  if (body.includes('Web search is not configured') || body.includes('Please provide a web search query')) return false;
  const previousPurpose = extractSearchPurpose(previousMessage);
  if (!previousPurpose) return false;
  return /^Web search \(/i.test(body)
    || /\n(?:來源：|Sources:|Fontes:)\s*(?:\n|$)/i.test(body)
    || /^\[\d+\]\s+/m.test(body);
}

function extractSearchPurpose(text) {
  const raw = String(text || '').replace(/^輸入:/, '').trim();
  if (!raw || raw.startsWith('Ai:')) return '';
  if (/^(好|好的|可以|要|是|yes|y|ok|okay)$/i.test(raw)) return '';
  const withoutQuoted = raw.match(/[「"“](.*?)[」"”]/)?.[1];
  if (withoutQuoted) return withoutQuoted.trim();
  return raw
    .replace(/^(?:請幫我|請你幫我|可以幫我|幫我|幫忙|麻煩你?|請)?\s*/i, '')
    .replace(/^(?:用)?(?:網路|本機|internet|online|web)?\s*(?:搜尋|查詢|查找|查|找|搜索|search(?:\s+for)?|look\s+up)\s*/i, '')
    .replace(/^(?:一下|一下子|看看)\s*/i, '')
    .replace(/(?:的)?(?:搜尋結果|資料|資訊|消息)?\s*(?:是什麼|如何|怎麼樣|一下)?\s*$/i, '')
    .replace(/[「」"“”]/g, '')
    .replace(/[。！？.!?]\s*$/g, '')
    .trim();
}

function extractWebSearchQuery(text) {
  const body = String(text || '').trim();
  const found = body.match(/Web search \([^)]+\) found \d+ result\(s\) for ["“](.*?)["”]/i)
    || body.match(/found no usable results for ["“](.*?)["”]/i);
  return found?.[1]?.trim() || '';
}

function countWebSearchSources(text) {
  const raw = String(text || '');
  const formattedCount = raw.match(/found (\d+) result\(s\)/i)?.[1];
  if (formattedCount) return Number(formattedCount) || 0;
  const matches = raw.match(/^\[\d+\]\s+/gm);
  return matches?.length || 0;
}

function sanitizeSearchSummaryFilename(name) {
  return String(name || tForLanguage('zh-TW', 'chatSystem.summary')).replace(/[\\/:*?"<>|]+/g, '_').slice(0, 96);
}

function renderSearchSummaryTitle(title, query) {
  const plainTitle = String(title || tForLanguage('zh-TW', 'chatSystem.localSearchSummary'));
  const plainQuery = String(query || '').trim();
  if (!plainQuery) return plainTitle;
  const index = plainTitle.indexOf(plainQuery);
  if (index < 0) return plainTitle;
  return (
    <>
      {plainTitle.slice(0, index)}
      <span className="search-summary-query">{plainQuery}</span>
      {plainTitle.slice(index + plainQuery.length)}
    </>
  );
}
