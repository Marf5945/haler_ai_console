import React, {useRef, useState} from 'react';
import useI18n from '../locales/useI18n';
import {
  fileBaseName,
  fileExtLabel,
  referenceFileKey,
  referenceFileStatusLabel,
  sharedSourceKey,
  shouldShowReferenceFileDetail,
  twoLineFileName,
} from './referenceFiles';

export default function RightRail({
  isLearningEnabled,
  isRecordingEnabled,
  isToolPopupOpen,
  learningDigestReady,
  sourceTrustHint,
  referenceFiles,
  activeCodeFileName = '',
  sharedLinks = [],
  sharedListings = [],
  onLearningToggle,
  onRecordingToggle,
  onReferenceFileDrop,
  onReferenceFileDragOut,
  onReferenceFileReorder,
  onReferenceInternalDrag,
  onReferenceLinkOpen,
  onReferenceCardDoubleClick,
  onReferenceFailedRemove,
  onSharedSourceDragOut,
  onScheduleOpen,
  onToolFavorite,
  onToolPopupToggle,
}) {
  const t = useI18n(s => s.t);
  const [draggedReferenceKey, setDraggedReferenceKey] = useState('');
  const draggedReferenceKeyRef = useRef('');
  const [draggedSharedSourceKey, setDraggedSharedSourceKey] = useState('');
  const draggedSharedSourceKeyRef = useRef('');
  // 共用資料夾 → 第一層檔案清單（後端只讀該層，不遞迴）
  const listingByPath = {};
  (Array.isArray(sharedListings) ? sharedListings : []).forEach((listing) => {
    if (listing?.path) listingByPath[listing.path] = listing;
  });

  function handleReferenceDragStart(event, file) {
    const fileKey = referenceFileKey(file);
    if (!fileKey) {
      event.preventDefault();
      return;
    }
    // §M3+ 失敗 entry：三層保險阻止 OS 拿到任何 drag payload
    const isFailed = file?.status === 'error' || file?.source !== 'library';
    if (isFailed) {
      try {
        // 1. 清空 dataTransfer 內容（避免 text/plain 變 .textClipping）
        event.dataTransfer?.clearData?.();
        if (event.dataTransfer) {
          event.dataTransfer.effectAllowed = 'none';
          event.dataTransfer.setData('text/plain', '');
        }
      } catch (_) { /* old browsers */ }
      // 2. 阻止預設 drag 行為
      event.preventDefault();
      event.stopPropagation();
      // 3. 立即從 state 移除（即使 drag 真的被啟動，state 也已乾淨）
      onReferenceFailedRemove?.(fileKey);
      return false;
    }
    draggedReferenceKeyRef.current = fileKey;
    onReferenceInternalDrag?.(true);
    setDraggedReferenceKey(fileKey);
    event.dataTransfer.effectAllowed = 'copyMove';
    // §M3+ 不用 text/plain（macOS 會把 path-like text 升級成 file drag、誤觸 import）。
    // 改用 custom MIME；OS 不認得，drop 在外面也不會建桌面假檔；reorder 只看 draggedReferenceKeyRef。
    try { event.dataTransfer.setData('application/x-ai-console-ref-key', fileKey); } catch (_) {}
    void onReferenceFileDragOut?.(file);
  }

  function handleReferenceDragOver(event, file) {
    const draggedKey = draggedReferenceKeyRef.current;
    const targetKey = referenceFileKey(file);
    if (!draggedKey || !targetKey || draggedKey === targetKey) return;
    event.preventDefault();
    event.stopPropagation();
    event.dataTransfer.dropEffect = 'move';
    const rect = event.currentTarget.getBoundingClientRect();
    const placement = event.clientY > rect.top + (rect.height / 2) ? 'after' : 'before';
    onReferenceFileReorder?.(draggedKey, targetKey, placement);
  }

  function finishReferenceDrag(event) {
    event?.stopPropagation?.();
    draggedReferenceKeyRef.current = '';
    onReferenceInternalDrag?.(false);
    setDraggedReferenceKey('');
  }

  function handleSharedSourceDragStart(event, link) {
    const key = sharedSourceKey(link);
    if (!key) {
      event.preventDefault();
      return;
    }
    draggedSharedSourceKeyRef.current = key;
    setDraggedSharedSourceKey(key);
    event.dataTransfer.effectAllowed = 'move';
    try { event.dataTransfer.clearData(); } catch (_) {}
    try { event.dataTransfer.setData('application/x-ai-console-shared-source-key', key); } catch (_) {}
  }

  function finishSharedSourceDrag(event, link) {
    const key = draggedSharedSourceKeyRef.current;
    draggedSharedSourceKeyRef.current = '';
    setDraggedSharedSourceKey('');
    if (!key) return;
    const leftWindow =
      event.clientX <= 0 ||
      event.clientY <= 0 ||
      event.clientX >= window.innerWidth ||
      event.clientY >= window.innerHeight;
    const droppedOutside = leftWindow || (event.clientX === 0 && event.clientY === 0);
    if (droppedOutside) {
      onSharedSourceDragOut?.(link);
    }
  }

  function handleReferenceDrop(event) {
    // 只把內部 reference 排序拖曳吃掉；外部檔案仍要進圖片/引用分流。
    if (draggedReferenceKeyRef.current) {
      event?.preventDefault?.();
      finishReferenceDrag(event);
      return;
    }
    if (event?.dataTransfer?.files?.length) {
      event.preventDefault();
      event.stopPropagation();
      onReferenceFileDrop?.(Array.from(event.dataTransfer.files));
    }
  }

  return (
    <aside
      className="right-panel"
      onDragOver={(event) => {
        if (
          event.dataTransfer.files?.length
          || Array.from(event.dataTransfer.types).includes('Files')
          || Array.from(event.dataTransfer.types).includes('application/x-ai-console-tool-id')
        ) {
          event.preventDefault();
          event.dataTransfer.dropEffect = event.dataTransfer.files?.length || Array.from(event.dataTransfer.types).includes('Files') ? 'copy' : 'move';
        }
      }}
      onDrop={(event) => {
        // §M3+ 內部 reorder drag 不應觸發 import；以 draggedReferenceKeyRef 為憑。
        if (draggedReferenceKeyRef.current) {
          event.preventDefault();
          finishReferenceDrag(event);
          return;
        }
        if (event.dataTransfer.files?.length) {
          event.preventDefault();
          event.stopPropagation();
          onReferenceFileDrop(Array.from(event.dataTransfer.files));
          return;
        }
        const toolId = event.dataTransfer.getData('application/x-ai-console-tool-id');
        if (!toolId) return;
        event.preventDefault();
        onToolFavorite(toolId);
      }}
    >
      <button className="tool-card tool-amber reference-link-card" type="button" onClick={onReferenceLinkOpen}>
        <span>▤</span>
        <span>{t('rightRail.citeLink')}</span>
      </button>
      <button className="tool-card tool-amber tool-schedule-card" type="button" onClick={onScheduleOpen}>
        <span>◴</span>
        <span>{t('rightRail.schedule')}</span>
      </button>
      {sourceTrustHint && (
        <div
          className="source-trust-chip source-trust-rail-chip"
          data-level={sourceTrustHint.level}
          title={sourceTrustHint.text}
        >
          <span className="source-trust-icon">{sourceTrustHint.level === 'trusted' ? '✓' : sourceTrustHint.level === 'blocked' ? '✕' : '⚠'}</span>
          <span className="source-trust-label">{sourceTrustHint.text}</span>
        </div>
      )}
      <div className="rail-mode-row">
        <button
          className={`rail-mode-btn rail-mode-learning ${isLearningEnabled ? 'rail-mode-active' : ''} ${learningDigestReady ? 'rail-mode-notify' : ''}`}
          type="button"
          onClick={onLearningToggle}
          title={t('rightRail.learningTooltip')}
        >
          <span>{t('rightRail.learning')}</span>
          <small>{learningDigestReady ? t('rightRail.hasUpdate') : isLearningEnabled ? t('rightRail.editing') : t('rightRail.close')}</small>
        </button>
        <button
          className={`rail-mode-btn rail-mode-record ${isRecordingEnabled ? 'rail-mode-active rail-mode-recording' : ''}`}
          type="button"
          onClick={onRecordingToggle}
          title={t('rightRail.recordTooltip')}
        >
          <span>{t('rightRail.record')}</span>
          <small>{isRecordingEnabled ? t('rightRail.recording') : t('rightRail.close')}</small>
        </button>
      </div>
      <div
        className="tool-card shared-link-card"
        aria-label={`${t('rightRail.sharedLink')} - ${t('rightRail.sharedLinkHint')}`}
      >
        <span>◎</span>
        <span>{t('rightRail.sharedLink')}</span>
        <div className="rail-hint-popover" role="tooltip">
          <strong>{t('rightRail.sharedLink')}</strong>
          <small>{t('rightRail.sharedLinkHint')}</small>
        </div>
      </div>
      {Array.isArray(sharedLinks) && sharedLinks.length > 0 && (
        <div className="shared-source-list" aria-label={t('rightRail.sharedLink')}>
          {sharedLinks.map((link, index) => {
            const sourcePath = link?.url || link?.URL || '';
            const label = link?.label || link?.Label || fileBaseName(sourcePath) || t('rightRail.sharedLink');
            const sourceKey = sharedSourceKey(link) || `${sourcePath}-${index}`;
            const isDragging = draggedSharedSourceKey === sourceKey;
            const listing = listingByPath[sourcePath];
            const listingFiles = listing?.is_dir ? (listing.files || []) : [];
            const showListing = Boolean(listing && (listing.is_dir || listing.error));
            return (
              <React.Fragment key={sourceKey}>
                <div
                  className={`shared-source-name${isDragging ? ' shared-source-dragging' : ''}`}
                  data-draggable="true"
                  draggable
                  title={sourcePath}
                  onDragStart={(event) => handleSharedSourceDragStart(event, link)}
                  onDragEnd={(event) => finishSharedSourceDrag(event, link)}
                >
                  <span className="reference-file-title">
                    {twoLineFileName(label, t('rightRail.sharedLink')).map((line, lineIndex) => <span key={lineIndex}>{line}</span>)}
                  </span>
                  {sourcePath && <small className="reference-file-detail">{sourcePath}</small>}
                </div>
                {showListing && (
                  <div className="shared-source-files" aria-label={`${label} - ${t('rightRail.sharedFolderTopOnly')}`}>
                    {listing.is_dir && <small className="shared-source-scope-hint">{t('rightRail.sharedFolderTopOnly')}</small>}
                    {listing.error ? (
                      <small className="shared-source-file-empty">{t('rightRail.sharedFolderUnreadable')}</small>
                    ) : listingFiles.length === 0 ? (
                      <small className="shared-source-file-empty">{t('rightRail.sharedFolderEmpty')}</small>
                    ) : (
                      listingFiles.map((file) => (
                        <div className="shared-source-file-row" key={file.path} title={file.path}>
                          <span className="shared-source-file-name">{file.name}</span>
                          {file.ext && <span className="reference-file-ext-badge">{file.ext}</span>}
                        </div>
                      ))
                    )}
                    {listing.truncated && (
                      <small className="shared-source-file-empty">{t('rightRail.sharedFolderMore', {count: listingFiles.length})}</small>
                    )}
                  </div>
                )}
              </React.Fragment>
            );
          })}
        </div>
      )}
      <div
        className="tool-card reference-file-card"
        aria-label={`${t('rightRail.citeFile')} - ${t('rightRail.citeFileHint')}`}
        onDragOver={(event) => {
          event.preventDefault();
          event.dataTransfer.dropEffect = 'copy';
        }}
        onDrop={(event) => {
          handleReferenceDrop(event);
        }}
      >
        <span>▤</span>
        <span>{t('rightRail.citeFile')}</span>
        <div className="rail-hint-popover" role="tooltip">
          <strong>{t('rightRail.citeFile')}</strong>
          <small>{t('rightRail.citeFileHint')}</small>
        </div>
      </div>
      <div
        className="reference-file-list"
        onDragOver={(event) => {
          if (!draggedReferenceKeyRef.current) return;
          event.preventDefault();
          event.stopPropagation();
          event.dataTransfer.dropEffect = 'move';
        }}
        onDrop={handleReferenceDrop}
      >
        {referenceFiles.map((file, index) => {
          const fileKey = referenceFileKey(file) || `${file.path}-${index}`;
          const isDragging = draggedReferenceKey === fileKey;
          const isActiveCode = Boolean(activeCodeFileName) && file.name === activeCodeFileName;
          return (
            <div
              className={`reference-file-name${isDragging ? ' reference-file-dragging' : ''}${isActiveCode ? ' reference-file-code-active' : ''}`}
              data-status={file.status || 'ready'}
              data-draggable="true"
              draggable
              key={fileKey}
              title={file.detail || file.path}
              onDragStart={(event) => handleReferenceDragStart(event, file)}
              onDragOver={(event) => handleReferenceDragOver(event, file)}
              onDrop={handleReferenceDrop}
              onDragEnd={(event) => {
                // §M3+ 失敗 entry 拖到 window 外 → 移除（同 ToolPopup 的 leftWindow pattern）
                const leftWindow =
                  event.clientX <= 0 ||
                  event.clientY <= 0 ||
                  event.clientX >= window.innerWidth ||
                  event.clientY >= window.innerHeight;
                const droppedOutside = leftWindow || (event.clientX === 0 && event.clientY === 0);
                const isFailed = file?.status === 'error' || file?.source !== 'library';
                finishReferenceDrag(event);
                if (droppedOutside && isFailed) {
                  onReferenceFailedRemove?.(referenceFileKey(file));
                }
              }}
              onDoubleClick={(event) => {
                event.preventDefault();
                const rect = event.currentTarget.getBoundingClientRect();
                onReferenceCardDoubleClick?.(file, rect);
              }}
            >
              <div className="reference-file-main">
                <span className="reference-file-title">
                  {twoLineFileName(file.name, t('rightRail.unnamedFile')).map((line, lineIndex) => <span key={lineIndex}>{line}</span>)}
                </span>
                <small className="reference-file-status">{referenceFileStatusLabel(file.status, t)}</small>
                {file.addedVia === 'floating_avatar' && (
                  <small className="reference-file-source-badge">{t('floatingAvatar.addedViaFloating')}</small>
                )}
              </div>
              {shouldShowReferenceFileDetail(file) && <small className="reference-file-detail">{file.detail}</small>}
              {fileExtLabel(file.name) && <span className="reference-file-ext-badge">{fileExtLabel(file.name)}</span>}
            </div>
          );
        })}
      </div>
      <button className="tool-card tool-use-bottom" type="button" onClick={onToolPopupToggle}>
        <span>{isToolPopupOpen ? '×' : '⌕'}</span>
        <span>{isToolPopupOpen ? t('rightRail.close') : t('rightRail.useTools')}</span>
      </button>
    </aside>
  );
}

// §3.1.11 影片副檔名判斷：拖入時用來分流到 data/videos
