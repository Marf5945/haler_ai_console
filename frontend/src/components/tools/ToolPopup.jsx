import {useState} from 'react';
import useI18n from '../../locales/useI18n';
import {dagStatusLabel, getToolTabOptions, openExternal, toolTabFor} from '../../lib/appShared';
import GoProgramAuthoringCatalogList from './GoProgramAuthoringCatalogList';
import RecordingCatalogList from './RecordingCatalogList';
import ToolFlowSectionLabel from './ToolFlowSectionLabel';

export default function ToolPopup({side, tools, activeTab, favoriteToolIds, hiddenToolIds, toolResult, externalServiceLinks, documentationLinks, dagRun, onTabChange, onToolActivate, onToolDragStart, onToolDragOut, onSkillNativeDragStart, onReorder}) {
  const t = useI18n(s => s.t);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const normalizedQuery = searchQuery.trim().toLowerCase();
  const visibleTools = tools.filter((tool) => (
    tool.id !== 'tool-entrance' &&
    (side !== 'right' || favoriteToolIds.includes(tool.id)) &&
    !hiddenToolIds.includes(tool.id) &&
    toolTabFor(tool) === activeTab
  )).filter((tool) => {
    if (!normalizedQuery) return true;
    return [tool.title, tool.detail, tool.id]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(normalizedQuery));
  });
  const visibleExternalServiceLinks = (externalServiceLinks || []).filter((link) => !hiddenToolIds.includes(link.id));
  const visibleDocumentationLinks = (documentationLinks || []).filter((link) => !hiddenToolIds.includes(link.id));

  function externalLinkTool(link, icon, detail) {
    return {
      id: link.id,
      icon,
      title: link.label || link.url,
      detail: link.url || detail,
      kind: 'external',
      target: link.url,
      enabled: true,
      available: true,
    };
  }

  function startToolDrag(event, tool) {
    event.dataTransfer.effectAllowed = 'copyMove';
    event.dataTransfer.setData('application/x-ai-console-tool-id', tool.id);
    onToolDragStart({tool, source: side});
    if (tool.kind === 'skill') {
      void onSkillNativeDragStart?.(tool, side);
    }
  }

  function finishToolDrag(event, tool) {
    if (tool.kind === 'skill') {
      onToolDragStart(null);
      return;
    }
    const leftWindow =
      event.clientX <= 0 ||
      event.clientY <= 0 ||
      event.clientX >= window.innerWidth ||
      event.clientY >= window.innerHeight;
    const droppedOutside = leftWindow || (event.clientX === 0 && event.clientY === 0);
    if (droppedOutside && event.dataTransfer.dropEffect !== 'move') {
      onToolDragOut({tool, source: side});
    }
    onToolDragStart(null);
  }

  return (
    <section className={`tools-popup tools-popup-${side}`} aria-label={t('tool.panelLabel')}>
      <header className="tools-popup-tabs">
        {getToolTabOptions().map((tab) => (
          <button
            className={`tools-popup-tab ${activeTab === tab.id ? 'tools-popup-tab-active' : ''}`}
            type="button"
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
          >
            {tab.label}
          </button>
        ))}
        <button
          className={`tools-popup-search ${searchOpen ? 'tools-popup-search-active' : ''}`}
          type="button"
          aria-expanded={searchOpen}
          aria-label={t('tool.searchLabel')}
          onClick={() => setSearchOpen((open) => !open)}
        >
          <svg className="tools-popup-search-icon" viewBox="0 0 20 20" aria-hidden="true">
            <circle cx="8.4" cy="8.4" r="4.8" />
            <path d="M12 12l4.2 4.2" />
          </svg>
        </button>
      </header>
      {searchOpen && (
        <div className="tools-search-popover" role="search">
          <input
            autoFocus
            aria-label={t('tool.searchNameLabel')}
            placeholder={t('tool.searchPlaceholder')}
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Escape') {
                setSearchQuery('');
                setSearchOpen(false);
              }
            }}
          />
          {searchQuery && (
            <button type="button" aria-label={t('tool.clearSearch')} onClick={() => setSearchQuery('')}>×</button>
          )}
        </div>
      )}
        {/* I-6 (#I-601): tool-menu-list 同時作為 intra-list 拖曳排序的 drop container。
           拖入同列表另一項時，根據 drop 位置計算 newRank 並呼叫 onReorder。 */}
      <div
        className="tool-menu-list"
        onDragOver={(event) => {
          if (event.dataTransfer.types.includes('application/x-ai-console-tool-id')) {
            event.preventDefault();
            event.dataTransfer.dropEffect = 'move';
          }
        }}
        onDrop={(event) => {
          const droppedId = event.dataTransfer.getData('application/x-ai-console-tool-id');
          if (!droppedId || !onReorder) return;
          event.preventDefault();
          // 根據滑鼠 Y 位置找到最近的 tool-menu-item 計算 newRank
          const items = Array.from(event.currentTarget.querySelectorAll('[data-tool-id]'));
          let targetIndex = items.length; // 預設放最後
          for (let i = 0; i < items.length; i++) {
            const rect = items[i].getBoundingClientRect();
            if (event.clientY < rect.top + rect.height / 2) {
              targetIndex = i;
              break;
            }
          }
          onReorder(droppedId, targetIndex);
        }}
      >
        {/* I-7: 自動流程 tab mirrors the active DAG Hook Run and node summaries. */}
        {activeTab === 'flow' && dagRun && (
          <button
            type="button"
            className="tool-menu-item flow-asset-card"
            title={`${dagRun.title} · ${dagStatusLabel(dagRun.status)}`}
          >
            <span className="tool-menu-icon">◇</span>
            <strong>{dagRun.title}</strong>
            <small>{dagStatusLabel(dagRun.status)} · {dagRun.nodes?.length || 0}</small>
          </button>
        )}
        {/* Recording assets live here; DAG run history stays out of this list. */}
        {activeTab === 'flow' && (
          <RecordingCatalogList />
        )}
        {activeTab === 'flow' && visibleTools.length > 0 && (
          <ToolFlowSectionLabel>{t('tool.flowCategorySkill')}</ToolFlowSectionLabel>
        )}
        {activeTab === 'flow' && (
          <GoProgramAuthoringCatalogList showLabel={visibleTools.length === 0} />
        )}
        {/* ── #44 工具清單：可用工具在前、斷線工具在後 ── */}
        {/* unavailable 工具：icon 加黑線打叉覆蓋，保持可見，不灰化 */}
        {visibleTools.map((tool) => {
          const isUnavailable = tool.available === false || tool.needs_reauth || !tool.enabled;
          return (
            <button
              className={`tool-menu-item ${isUnavailable ? 'tool-menu-item-unavailable' : ''}`}
              draggable
              type="button"
              key={tool.id || tool.title}
              data-tool-id={tool.id}
              onClick={() => onToolActivate(tool.id)}
              onDragEnd={(event) => finishToolDrag(event, tool)}
              onDragStart={(event) => startToolDrag(event, tool)}
              title={isUnavailable
                ? `${tool.title} — ${tool.unavailable_reason || t('tool.unavailableReason')}`
                : tool.title}
            >
              <span className="tool-menu-icon" style={{position: 'relative'}}>
                {tool.icon}
                {/* #44: 黑線打叉覆蓋（取代舊的 lightning-fracture） */}
                {isUnavailable && (
                  <span className="tool-unavailable-x" aria-label={t('tool.unavailableAriaLabel')} style={{
                    position: 'absolute', top: 0, left: 0, right: 0, bottom: 0,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: '1.4em', color: '#1a1a1a', fontWeight: 'bold',
                    pointerEvents: 'none', lineHeight: 1,
                  }}>✕</span>
                )}
              </span>
              <strong>{tool.title}</strong>
              <small>{isUnavailable ? (tool.unavailable_reason || t('tool.disconnected')) : tool.detail}</small>
            </button>
          );
        })}
        {/* I-5: external_service 連結 — 來自 ListExternalLinksByType("external_service")，顯示於「外部連結」tab */}
        {activeTab === 'external' && visibleExternalServiceLinks.map((link) => {
          const linkTool = externalLinkTool(link, '↗', t('link.externalService'));
          return (
            <button
              className="tool-menu-item tool-menu-item-link"
              key={link.id}
              draggable
              type="button"
              data-tool-id={link.id}
              title={link.url}
              onClick={() => openExternal(link.url)}
              onDragEnd={(event) => finishToolDrag(event, linkTool)}
              onDragStart={(event) => startToolDrag(event, linkTool)}
            >
              <span className="tool-menu-icon">↗</span>
              <strong>{link.label || link.url}</strong>
              <small>{t('link.externalService')}</small>
            </button>
          );
        })}
        {/* I-5: documentation 連結 — 純參考，不進工具執行區。
             必須用 Wails BrowserOpenURL 在 OS 預設瀏覽器外部開啟，
             不得在 AI Console 內部 WebView 開啟（#47 合約要求）。 */}
        {activeTab === 'external' && visibleDocumentationLinks.map((link) => {
          const linkTool = externalLinkTool(link, '▤', t('link.docLinkLabel'));
          return (
            <button
              className="tool-menu-item tool-menu-item-link tool-menu-item-doc"
              key={link.id}
              draggable
              type="button"
              data-tool-id={link.id}
              title={t('link.docLinkTitle', { url: link.url })}
              onClick={() => openExternal(link.url)}
              onDragEnd={(event) => finishToolDrag(event, linkTool)}
              onDragStart={(event) => startToolDrag(event, linkTool)}
            >
              <span className="tool-menu-icon">▤</span>
              <strong>{link.label || link.url}</strong>
              <small>{t('link.docLinkLabel')}</small>
            </button>
          );
        })}
        {!visibleTools.length && activeTab !== 'flow' && !(activeTab === 'external' && (visibleExternalServiceLinks.length || visibleDocumentationLinks.length)) && (
          <div className="tool-search-empty">
            <strong>{t('tool.noMatch')}</strong>
            <span>{t('tool.noMatchHint')}</span>
          </div>
        )}
      </div>
      {toolResult?.message && (
        <p className={toolResult.ok ? 'tool-action-note tool-action-note-ok' : 'tool-action-note'}>
          {toolResult.message}
        </p>
      )}
    </section>
  );
}
