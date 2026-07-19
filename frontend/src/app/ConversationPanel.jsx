import React, {useEffect, useRef, useState} from 'react';
import MessageRow from '../components/chat/MessageRow';
import {messageDomId} from '../components/highlight/messageId';
import FloatingCandidateActions from '../features/readiness-gate/FloatingCandidateActions';
import MissingSlotCapsule from '../features/readiness-gate/MissingSlotCapsule';
import RetrievalTransparency from '../features/readiness-gate/RetrievalTransparency';
import ConfirmationTier from '../features/readiness-gate/ConfirmationTier';
import ComposerConfirmBubble from '../features/readiness-gate/ComposerConfirmBubble';
import useI18n from '../locales/useI18n';
import {voiceStatusLabel} from '../lib/panelSettings';
import {localizeChatSystemMessage, stripComposerPendingMarker} from './conversationMessages';

export const fallbackReadinessGate = {
  risk_tier: 'none',
  missing_slots: [],
  floating_candidates: [],
  clarification_count: 0,
  max_clarifications: 2,
  retrieval_scanning: false,
  retrieval_sources: [],
  impact_explanation: '',
  low_confidence_output: false,
  assumption_used: false,
  auto_output_allowed: false,
};

function MicIcon() {
  return (
    <svg className="mic-icon" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 3.5c-1.7 0-3 1.3-3 3v5c0 1.7 1.3 3 3 3s3-1.3 3-3v-5c0-1.7-1.3-3-3-3Z" />
      <path d="M6.5 10.5c0 3 2.4 5.5 5.5 5.5s5.5-2.5 5.5-5.5M12 16v3.5M9 20.5h6" />
    </svg>
  );
}

export default function ConversationPanel({
  messages, personaName, draft, onDraftChange, onSend, onDelete, onSummarizeSearch, onExportSearchSummary,
  onInjectText, activeConversationId, chatLocale = 'zh-TW',
  readinessGate = fallbackReadinessGate,
  selectedFloatingCandidateIDs = [],
  longPressProgress = 0, gachaPhase = null, riskImpactExpanded = false,
  onSelectCandidate, onNormalConfirm, onNormalReject, onHighRiskYes,
  onLongPressStart, onLongPressEnd,
  dispatchStatus = {},
  voiceState = null, voiceRecording = false, voiceBusy = false, voiceStatus = '', voiceError = '',
  onVoicePressStart, onVoicePressEnd, onVoiceCancel,
  voiceChatEnabled = false, voiceChatHint = false,
  voiceSyncActive = false, voiceSyncPhase = 'idle',
  onToggleVoiceChat, onDismissVoiceChatHint,
  taskActive = false, onCancelTask,
  pendingTaskReview = null, taskReviewDetailsOpen = false,
  onConfirmTaskReview, onCancelTaskReview, onShowTaskReviewDetails,
  composerConfirmAction = null, onComposerConfirm, onComposerCancel,
}) {
  const t = useI18n(s => s.t);
  const [activeMessage, setActiveMessage] = useState(null);
  const [imagePreviews, setImagePreviews] = useState([]);
  const [imageError, setImageError] = useState('');
  const MAX_IMAGE_BYTES = 8 * 1024 * 1024;
  const composerComposingRef = useRef(false);
  const taskReviewCardRef = useRef(null);
  const imageInputRef = useRef(null);
  const voiceReady = voiceState?.status === 'ready';
  const micAvailable = typeof navigator !== 'undefined' && !!navigator.mediaDevices?.getUserMedia;
  const voiceDisabled = voiceBusy || !voiceReady || !micAvailable;
  const voiceTitle = !voiceReady
    ? voiceStatusLabel(voiceState?.status)
    : voiceChatEnabled
      ? (voiceSyncActive ? t('composer.voiceSyncStop') : t('composer.voiceSyncTap'))
      : t('composer.voiceHold');
  const hasSelectedFloatingCandidates = selectedFloatingCandidateIDs.length > 0;
  const composerReady = !!draft.trim() || hasSelectedFloatingCandidates;
  const displayMessage = (message) => localizeChatSystemMessage(stripComposerPendingMarker(message), chatLocale)
    .replace(/^Ai:/, personaName + ':')
    .replace(/^輸入:/, '');

  function addImageFiles(fileList) {
    const files = Array.from(fileList || []).filter((file) => file?.type?.startsWith('image/'));
    if (files.length === 0) return;
    files.forEach((file) => {
      if (file.size > MAX_IMAGE_BYTES) {
        setImageError(t('composer.imageTooLarge', {max: '8MB'}));
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
          <MessageRow
            key={`${message}-${index}`}
            message={message}
            displayText={displayMessage(message)}
            kind={message.startsWith('Ai:') ? 'ai' : 'user'}
            isActive={activeMessage === index}
            index={index}
            domMessageId={messageDomId(message, index, messages)}
            summaryLabel={t('composer.summary')}
            deleteLabel={t('composer.delete')}
            onActivate={() => setActiveMessage(index)}
            onDelete={(rowIndex) => {
              onDelete(rowIndex);
              setActiveMessage(null);
            }}
            onSummarizeSearch={onSummarizeSearch}
            onExportSearchSummary={onExportSearchSummary}
            onInjectText={onInjectText}
            sessionId={activeConversationId}
            previousMessage={messages[index - 1] || ''}
            chatLocale={chatLocale}
          />
        ))}
        {pendingTaskReview && (
          <article className="task-review-inline-card" ref={taskReviewCardRef}>
            <header><strong>需要你確認這一步</strong><span>待確認</span></header>
            <div className="task-review-inline-grid">
              <div><small>這一步要做什麼</small><p>{pendingTaskReview.title}</p></div>
              <div><small>為什麼需要確認</small><p>{pendingTaskReview.reason}</p></div>
            </div>
            <p className="task-review-inline-impact">會影響：{pendingTaskReview.impact}</p>
            <p className="task-review-inline-impact">使用的工具/模型：{pendingTaskReview.tool}</p>
            <footer>
              <button type="button" className="task-review-detail-btn" onClick={onShowTaskReviewDetails}>{taskReviewDetailsOpen ? '關閉' : '查看內容'}</button>
              <button type="button" className="task-review-cancel-btn" onClick={onCancelTaskReview}>取消</button>
              <button type="button" className="task-review-confirm-btn" onClick={() => onConfirmTaskReview?.(pendingTaskReview.id)}>確認執行</button>
            </footer>
          </article>
        )}
      </div>

      {(readinessGate.retrieval_scanning
        || readinessGate.floating_candidates?.length > 0
        || readinessGate.missing_slots?.length > 0
        || (readinessGate.risk_tier && readinessGate.risk_tier !== 'none')) && (
        <div className="readiness-above-composer">
          <RetrievalTransparency isScanning={readinessGate.retrieval_scanning} sources={readinessGate.retrieval_sources || []} />
          <FloatingCandidateActions candidates={readinessGate.floating_candidates || []} selectedIDs={selectedFloatingCandidateIDs} onSelect={onSelectCandidate} />
          <MissingSlotCapsule missingSlots={readinessGate.missing_slots || []} isHighRisk={readinessGate.risk_tier === 'high'} />
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

      <ComposerConfirmBubble action={composerConfirmAction} onConfirm={onComposerConfirm} onCancel={onComposerCancel} />

      {voiceChatHint && !voiceChatEnabled && (
        <div className="voice-chat-hint-bubble" role="status">
          <button type="button" className="voice-chat-hint-btn" onClick={() => onToggleVoiceChat?.(true)}><span aria-hidden="true">🔊</span> {t('voice.enableChatMode')}</button>
          <button type="button" className="voice-chat-hint-close" aria-label={t('voice.dismissHint')} onClick={() => onDismissVoiceChatHint?.()}>×</button>
        </div>
      )}

      <form
        className="composer"
        onSubmit={(event) => {
          onSend(event, imagePreviews);
          if (draft.trim() || hasSelectedFloatingCandidates) {
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
                <button type="button" onClick={() => setImagePreviews((prev) => prev.filter((item) => item.id !== img.id))} aria-label={t('composer.removeImage')}>×</button>
              </div>
            ))}
          </div>
        )}
        {imageError && <div className="composer-image-error">{imageError}</div>}
        <button
          className={`attach-btn task-stop-btn ${taskActive ? 'task-stop-active' : ''}`}
          type="button"
          title={taskActive ? t('composer.stopTask') : t('composer.noActiveTask')}
          aria-label={taskActive ? t('composer.stopTask') : t('composer.noActiveTask')}
          disabled={!taskActive}
          onClick={() => onCancelTask?.()}
        ><span aria-hidden="true">■</span></button>
        <button
          className={`voice-btn ${voiceRecording ? 'voice-btn-recording' : ''} ${voiceSyncActive ? `voice-btn-sync voice-btn-sync-${voiceSyncPhase}` : ''}`}
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
        ><MicIcon /></button>
        <button className="image-insert-btn" type="button" title={t('composer.insertImage')} aria-label={t('composer.insertImage')} onClick={() => imageInputRef.current?.click()}><span aria-hidden="true">▧</span></button>
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
        <div className="input-wrap" onDrop={handleComposerDrop} onDragOver={(event) => event.preventDefault()}>
          <textarea
            value={draft}
            rows={1}
            placeholder={t('composer.placeholder')}
            onChange={(event) => onDraftChange(event.target.value)}
            onPaste={handleComposerPaste}
            onCompositionStart={() => { composerComposingRef.current = true; }}
            onCompositionEnd={() => { composerComposingRef.current = false; }}
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
          <button className={`send-btn ${composerReady ? 'send-btn-enabled' : ''} ${hasSelectedFloatingCandidates ? 'send-btn-ready' : ''}`} type="submit"><span>◢</span>{t('composer.send')}</button>
          {voiceChatEnabled && (
            <button className="voice-chat-toggle-btn" type="button" title={t('voice.chatModeOn')} aria-label={t('voice.chatModeOn')} onClick={() => onToggleVoiceChat?.(false)}><span aria-hidden="true">🔊</span></button>
          )}
        </div>
        {(voiceStatus || voiceError || voiceState?.status !== 'ready') && (
          <div className={`voice-status ${voiceError ? 'voice-status-error' : ''}`}>{voiceError || voiceStatus || voiceStatusLabel(voiceState?.status)}</div>
        )}
        <div className="composer-ai-disclaimer">{t('composer.aiDisclaimer')}</div>
        {Object.values(dispatchStatus).length > 0 && (
          <div className="dispatch-status-area">
            {Object.values(dispatchStatus).map((ds) => {
              if (!ds) return null;
              if (ds.done && ds.overall === 'success') return <span key={ds.dispatchId} className="dispatch-status dispatch-success">{t('composer.sent')}</span>;
              if (ds.done && ds.overall === 'partial_fail') {
                const failedSegs = (ds.segments || []).filter(s => s.error);
                return <span key={ds.dispatchId} className="dispatch-status dispatch-partial-fail">{t('composer.sentPartialFail', {success: `${(ds.segments || []).length - failedSegs.length}/${(ds.segments || []).length}`, failed: failedSegs.map(s => s.part_index + 1).join(',')})}<button className="dispatch-retry-btn" type="button" onClick={() => {}}>{t('composer.retry')}</button></span>;
              }
              if (ds.done && ds.overall === 'failed') return <span key={ds.dispatchId} className="dispatch-status dispatch-fail">{t('composer.sendFailed')}<button className="dispatch-retry-btn" type="button" onClick={() => {}}>{t('composer.retry')}</button></span>;
              return <span key={ds.dispatchId} className="dispatch-status dispatch-sending">{t('composer.sending', {current: ds.partIndex + 1, total: ds.totalParts})}</span>;
            })}
          </div>
        )}
      </form>
    </section>
  );
}
