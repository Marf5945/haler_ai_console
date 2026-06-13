

export default function DigestAutoArchiveToast({message, onDismiss}) {
  return (
    <div className="digest-toast" role="status" aria-live="polite">
      <span className="digest-toast-icon">📦</span>
      <span className="digest-toast-msg">{message}</span>
      <button type="button" className="digest-toast-close" onClick={onDismiss}>×</button>
    </div>
  );
}
