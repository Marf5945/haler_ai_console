export default function ComposerConfirmBubble({action, onConfirm, onCancel}) {
  if (!action) return null;
  return (
    <div className="composer-confirm-bubble" role="region" aria-label={action.title || '確認操作'}>
      <div className="composer-confirm-copy">
        <strong>{action.title}</strong>
        {action.lines?.length > 0 && (
          <div className="composer-confirm-lines">
            {action.lines.map((line, index) => (
              <span key={`${line}-${index}`}>{line}</span>
            ))}
          </div>
        )}
      </div>
      <div className="composer-confirm-actions">
        {action.summary && (
          <button type="button" className="composer-confirm-summary" onClick={() => window.alert(action.summary)}>
            摘要
          </button>
        )}
        <button type="button" className="composer-confirm-cancel" onClick={onCancel}>
          {action.cancelLabel || '取消'}
        </button>
        <button type="button" className="composer-confirm-primary" onClick={onConfirm} disabled={action.busy}>
          {action.primaryLabel || '確定'}
        </button>
      </div>
    </div>
  );
}
