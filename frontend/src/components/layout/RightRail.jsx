import {useRef, useState} from 'react';
import useI18n from '../../locales/useI18n';
import {fileExtLabel, referenceFileKey, referenceFileStatusLabel, shouldShowReferenceFileDetail, twoLineFileName} from '../../lib/appShared';

export default function RightRail({
  isLearningEnabled,
  isRecordingEnabled,
  isToolPopupOpen,
  learningDigestReady,
  sourceTrustHint,
  referenceFiles,
  onLearningToggle,
  onRecordingToggle,
  onReferenceFileDrop,
  onReferenceFileDragOut,
  onReferenceFileReorder,
  onReferenceInternalDrag,
  onReferenceLinkOpen,
  onReferenceCardDoubleClick,
  onReferenceFailedRemove,
  onToolFavorite,
  onToolPopupToggle,
}) {
  const t = useI18n(s => s.t);
  const [draggedReferenceKey, setDraggedReferenceKey] = useState('');
  const draggedReferenceKeyRef = useRef('');

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
      <button className="tool-card tool-amber tool-schedule-card" type="button">
        <span>◴</span>
        <span>排程</span>
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
	        className="tool-card reference-file-card"
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
          return (
          <div
            className={`reference-file-name${isDragging ? ' reference-file-dragging' : ''}`}
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
              <small className="reference-file-status">{referenceFileStatusLabel(file.status)}</small>
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
