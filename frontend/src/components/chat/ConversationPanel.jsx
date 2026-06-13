import {useEffect, useRef, useState} from 'react';
import useI18n from '../../locales/useI18n';
import {fallbackReadinessGate, stripComposerPendingMarker, voiceStatusLabel} from '../../lib/appShared';
import ConfirmationTier from './ConfirmationTier';
import FloatingCandidateActions from './FloatingCandidateActions';
import MessageText from './MessageText';
import MicIcon from './MicIcon';
import MissingSlotCapsule from './MissingSlotCapsule';
import RetrievalTransparency from './RetrievalTransparency';

export default function ConversationPanel({
  messages, personaName, draft, onDraftChange, onSend, onDelete, onSummarizeSearch,
  onInjectText, activeConversationId,
  // v3.6.4 Readiness Gate props
  readinessGate = fallbackReadinessGate,
  longPressProgress = 0, gachaPhase = null, riskImpactExpanded = false,
  onSelectCandidate, onNormalConfirm, onNormalReject, onHighRiskYes,
  onLongPressStart, onLongPressEnd,
  // §12A.5B Dispatch 狀態
  dispatchStatus = {},
  voiceState = null, voiceRecording = false, voiceBusy = false, voiceStatus = '', voiceError = '',
  onVoicePressStart, onVoicePressEnd, onVoiceCancel,
  taskActive = false, onCancelTask,
  pendingTaskReview = null, taskReviewDetailsOpen = false,
  onConfirmTaskReview, onCancelTaskReview, onShowTaskReviewDetails,
}) {
  const t = useI18n(s => s.t);
  const [activeMessage, setActiveMessage] = useState(null);
  const [imagePreviews, setImagePreviews] = useState([]);
  const [imageError, setImageError] = useState('');
  const MAX_IMAGE_BYTES = 8 * 1024 * 1024; // 8MB：擋住整張截圖把記憶體灌爆
  const composerComposingRef = useRef(false);
  const taskReviewCardRef = useRef(null);
  const imageInputRef = useRef(null);
  const voiceReady = voiceState?.status === 'ready';
  const micAvailable = typeof navigator !== 'undefined' && !!navigator.mediaDevices?.getUserMedia;
  const voiceDisabled = voiceBusy || !voiceReady || !micAvailable;
  const voiceTitle = voiceReady ? t('composer.voiceHold') : voiceStatusLabel(voiceState?.status);
  const displayMessage = (message) => stripComposerPendingMarker(message)
    .replace(/^Ai:/, personaName + ':')
    .replace(/^輸入:/, '');

  const messageKind = (message) => {
    if (message.startsWith('Ai:')) return 'ai';
    return 'user';
  };
  const isLocalSearchMessage = (message) => message.startsWith('Ai:本機搜尋');

  function addImageFiles(fileList) {
    const files = Array.from(fileList || []).filter((file) => file?.type?.startsWith('image/'));
    if (files.length === 0) return;
    files.forEach((file) => {
      if (file.size > MAX_IMAGE_BYTES) {
        // 大圖直接擋下：避免 base64 在 state 與 DOM 各塞一份把記憶體灌爆、UI 卡頓。
        setImageError(t('composer.imageTooLarge', { max: '8MB' }));
        return;
      }
      const reader = new FileReader();
      reader.onload = (event) => {
        setImagePreviews((prev) => [...prev, {
          id: `img-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
          src: event.target?.result || '',
          name: file.name || 'image',
          type: file.type,
        }]);
        setImageError('');
      };
      reader.readAsDataURL(file);
    });
  }

  function removeImage(id) {
    setImagePreviews((prev) => prev.filter((img) => img.id !== id));
  }

  function handleComposerPaste(event) {
    const files = Array.from(event.clipboardData?.files || []).filter((file) => file?.type?.startsWith('image/'));
    if (files.length === 0) return;
    event.preventDefault();
    addImageFiles(files);
  }

  function handleComposerDrop(event) {
    const files = Array.from(event.dataTransfer?.files || []).filter((file) => file?.type?.startsWith('image/'));
    if (files.length === 0) return;
    event.preventDefault();
    addImageFiles(files);
  }

  useEffect(() => {
    if (!pendingTaskReview?.id) return;
    window.requestAnimationFrame(() => {
      taskReviewCardRef.current?.scrollIntoView({block: 'center', behavior: 'smooth'});
    });
  }, [pendingTaskReview?.id]);

  // SEC-06 UX: 新訊息進來（含「讀取內容」按鈕注入）時自動捲到底，
  // 讓使用者立刻看到最新狀態，不會以為點了沒反應。只在訊息變多時捲。
  const messageListRef = useRef(null);
  const prevMessageCountRef = useRef(messages.length);
  useEffect(() => {
    if (messages.length > prevMessageCountRef.current) {
      window.requestAnimationFrame(() => {
        const el = messageListRef.current;
        if (el) el.scrollTo({top: el.scrollHeight, behavior: 'smooth'});
      });
    }
    prevMessageCountRef.current = messages.length;
  }, [messages.length]);

  return (
    <section
      className="conversation-panel"
      onDragOver={(event) => { if (event.dataTransfer?.types?.includes('Files')) event.preventDefault(); }}
      onDrop={(event) => { if (event.dataTransfer?.types?.includes('Files')) event.preventDefault(); }}
    >
      <div className="message-list" ref={messageListRef}>
        {messages.map((message, index) => (
          <article
            className={`message-row message-${messageKind(message)} ${activeMessage === index ? 'message-row-active' : ''}`}
            key={`${message}-${index}`}
            onClick={() => setActiveMessage(index)}
          >
            <div className="message-text">
              <MessageText
                text={displayMessage(message)}
                kind={messageKind(message)}
                onInjectText={onInjectText}
                sessionId={activeConversationId}
              />
            </div>
            {isLocalSearchMessage(message) && (
              <button
                type="button"
                className="message-action"
                onClick={(event) => {
                  event.stopPropagation();
                  onSummarizeSearch?.(message.replace(/^Ai:/, ''));
                }}
              >
                {t('composer.summary')}
              </button>
            )}
            <button
              type="button"
              className="message-delete-action"
              onClick={(event) => {
                event.stopPropagation();
                onDelete(index);
                setActiveMessage(null);
              }}
            >
              {t('composer.delete')}
            </button>
          </article>
        ))}
        {pendingTaskReview && (
          <article className="task-review-inline-card" ref={taskReviewCardRef}>
            <header>
              <strong>需要你確認這一步</strong>
              <span>待確認</span>
            </header>
            <div className="task-review-inline-grid">
              <div>
                <small>這一步要做什麼</small>
                <p>{pendingTaskReview.title}</p>
              </div>
              <div>
                <small>為什麼需要確認</small>
                <p>{pendingTaskReview.reason}</p>
              </div>
            </div>
            <p className="task-review-inline-impact">會影響：{pendingTaskReview.impact}</p>
            <p className="task-review-inline-impact">使用的工具/模型：{pendingTaskReview.tool}</p>
            <footer>
              <button type="button" className="task-review-detail-btn" onClick={onShowTaskReviewDetails}>
                {taskReviewDetailsOpen ? '關閉' : '查看內容'}
              </button>
              <button type="button" className="task-review-cancel-btn" onClick={onCancelTaskReview}>取消</button>
              <button
                type="button"
                className="task-review-confirm-btn"
                onClick={() => onConfirmTaskReview?.(pendingTaskReview.id)}
              >
                確認執行
              </button>
            </footer>
          </article>
        )}
      </div>

      {/* ── v3.6.4 Readiness Gate 輸入框上方區域 ──
          §12A 佈局：所有 Readiness Gate 暫態 UI 元素掛在 composer 上方。
          由上而下順序：RetrievalTransparency → FloatingCandidateActions → MissingSlotCapsule → ConfirmationTier
          操作完成後消失刷新，不佔用固定空間。 */}
      {(readinessGate.retrieval_scanning
        || readinessGate.floating_candidates?.length > 0
        || readinessGate.missing_slots?.length > 0
        || (readinessGate.risk_tier && readinessGate.risk_tier !== 'none')
      ) && (
        <div className="readiness-above-composer">
          {/* §12A.6: Retrieval Transparency — 掃描動畫在對話訊息流 inline 顯示 */}
          <RetrievalTransparency
            isScanning={readinessGate.retrieval_scanning}
            sources={readinessGate.retrieval_sources || []}
          />
          {/* §12A.1: Floating Candidate Actions — 最多 3 個意圖候選 */}
          <FloatingCandidateActions
            candidates={readinessGate.floating_candidates || []}
            onSelect={onSelectCandidate}
          />
          {/* §12A.3: Missing Slot Capsule — 缺欄位膠囊 */}
          <MissingSlotCapsule
            missingSlots={readinessGate.missing_slots || []}
            isHighRisk={readinessGate.risk_tier === 'high'}
          />
          {/* §11.2: 三層確認機制 — 依風險等級切換 */}
          <ConfirmationTier
            riskTier={readinessGate.risk_tier}
            impactExplanation={readinessGate.impact_explanation}
            impactExpanded={riskImpactExpanded}
            longPressProgress={longPressProgress}
            gachaPhase={gachaPhase}
            onNormalConfirm={onNormalConfirm}
            onNormalReject={onNormalReject}
            onHighRiskYes={onHighRiskYes}
            onPressStart={onLongPressStart}
            onPressEnd={onLongPressEnd}
          />
        </div>
      )}

      {/* ── 原有 Composer（聊天輸入區）── */}
      <form
        className="composer"
        onSubmit={(event) => {
          // submitComposerText 對空字串會 return；只有真的有文字送出時才連帶清縮圖，
          // 純貼圖尚未送出時保留，避免誤刪使用者剛貼上的圖。
          onSend(event, imagePreviews);
          if (draft.trim()) {
            setImagePreviews([]);
            setImageError('');
          }
        }}
      >
        {imagePreviews.length > 0 && (
          <div className="composer-image-strip">
            {imagePreviews.map((img) => (
              <div className="composer-image-thumb" key={img.id}>
                <img src={img.src} alt={img.name || t('composer.imagePreview')} />
                <button type="button" onClick={() => removeImage(img.id)} aria-label={t('composer.removeImage')}>×</button>
              </div>
            ))}
          </div>
        )}
        {imageError && <div className="composer-image-error">{imageError}</div>}
        <button
          className={`attach-btn task-stop-btn ${taskActive ? 'task-stop-active' : ''}`}
          type="button"
          title={taskActive ? '停止目前任務' : '目前沒有執行中的任務'}
          aria-label={taskActive ? '停止目前任務' : '目前沒有執行中的任務'}
          disabled={!taskActive}
          onClick={() => onCancelTask?.()}
        >
          <span aria-hidden="true">■</span>
        </button>
        <button
          className={`voice-btn ${voiceRecording ? 'voice-btn-recording' : ''}`}
          type="button"
          title={voiceTitle}
          aria-label={voiceTitle}
          disabled={voiceDisabled}
          onPointerDown={(event) => {
            event.preventDefault();
            event.currentTarget.setPointerCapture?.(event.pointerId);
            onVoicePressStart?.();
          }}
          onPointerUp={(event) => {
            event.preventDefault();
            event.currentTarget.releasePointerCapture?.(event.pointerId);
            onVoicePressEnd?.();
          }}
          onPointerCancel={() => onVoiceCancel?.()}
        >
          <MicIcon />
        </button>
        <button
          className="image-insert-btn"
          type="button"
          title={t('composer.insertImage')}
          aria-label={t('composer.insertImage')}
          onClick={() => imageInputRef.current?.click()}
        >
          <span aria-hidden="true">▧</span>
        </button>
        <input
          ref={imageInputRef}
          className="composer-image-file"
          type="file"
          accept="image/*"
          multiple
          onChange={(event) => {
            addImageFiles(event.target.files);
            event.target.value = '';
          }}
        />
        <div
          className="input-wrap"
          onDrop={handleComposerDrop}
          onDragOver={(event) => event.preventDefault()}
        >
          <textarea
            value={draft}
            rows={1}
            placeholder={t('composer.placeholder')}
            onChange={(event) => onDraftChange(event.target.value)}
            onPaste={handleComposerPaste}
            onCompositionStart={() => {
              // 中文/日文 IME 組字期間，Enter 先保留給選字。
              composerComposingRef.current = true;
            }}
            onCompositionEnd={() => {
              composerComposingRef.current = false;
            }}
            onKeyDown={(event) => {
              const composing = composerComposingRef.current
                || event.isComposing
                || event.nativeEvent?.isComposing
                || event.nativeEvent?.keyCode === 229;
              if (event.key === 'Enter' && !event.shiftKey && !composing) {
                event.preventDefault();
                event.currentTarget.form.requestSubmit();
              }
            }}
          />
          <button className="send-btn" type="submit"><span>◢</span>{t('composer.send')}</button>
        </div>
        {(voiceStatus || voiceError || voiceState?.status !== 'ready') && (
          <div className={`voice-status ${voiceError ? 'voice-status-error' : ''}`}>
            {voiceError || voiceStatus || voiceStatusLabel(voiceState?.status)}
          </div>
        )}
        {/* §12A.5B Dispatch 狀態提示 */}
        {Object.values(dispatchStatus).length > 0 && (
          <div className="dispatch-status-area">
            {Object.values(dispatchStatus).map((ds) => {
              if (!ds) return null;
              if (ds.done && ds.overall === 'success') {
                return <span key={ds.dispatchId} className="dispatch-status dispatch-success">{t('composer.sent')}</span>;
              }
              if (ds.done && ds.overall === 'partial_fail') {
                const failedSegs = (ds.segments || []).filter(s => s.error);
                return (
                  <span key={ds.dispatchId} className="dispatch-status dispatch-partial-fail">
                    {t('composer.sentPartialFail', { success: `${(ds.segments || []).length - failedSegs.length}/${(ds.segments || []).length}`, failed: failedSegs.map(s => s.part_index + 1).join(',') })}
                    <button className="dispatch-retry-btn" type="button" onClick={() => {}}>{t('composer.retry')}</button>
                  </span>
                );
              }
              if (ds.done && ds.overall === 'failed') {
                return (
                  <span key={ds.dispatchId} className="dispatch-status dispatch-fail">
                    {t('composer.sendFailed')}
                    <button className="dispatch-retry-btn" type="button" onClick={() => {}}>{t('composer.retry')}</button>
                  </span>
                );
              }
              return <span key={ds.dispatchId} className="dispatch-status dispatch-sending">{t('composer.sending', { current: ds.partIndex + 1, total: ds.totalParts })}</span>;
            })}
          </div>
        )}
      </form>
    </section>
  );
}
