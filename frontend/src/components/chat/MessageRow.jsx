import React from 'react';
import MessageText from './MessageText';

export default function MessageRow({
  message,
  displayText,
  kind,
  isActive,
  index,
  summaryLabel,
  deleteLabel,
  onActivate,
  onDelete,
  onSummarizeSearch,
  onInjectText,
  sessionId,
}) {
  const isLocalSearch = message.startsWith('Ai:本機搜尋');
  return (
    <article
      className={`message-row message-${kind} ${isActive ? 'message-row-active' : ''}`}
      onClick={onActivate}
    >
      <div className="message-text">
        <MessageText
          text={displayText}
          kind={kind}
          onInjectText={onInjectText}
          sessionId={sessionId}
        />
      </div>
      {isLocalSearch && (
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
    </article>
  );
}
