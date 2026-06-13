

export default function LearningReplayConfirmCard({pending, onConfirm, onCancel}) {
  const step = pending?.result?.steps?.[pending?.stoppedStepOffset] || {};
  const label = step.label || step.window_title || step.window_process || 'native-window';
  return (
    <div className="learning-replay-confirm" role="dialog" aria-label="Replay confirmation">
      <header>
        <strong>確認這一步</strong>
        <span>OpenCV fallback</span>
      </header>
      <p>已把游標移到候選位置。確認後會點擊這一步並繼續剩餘 replay。</p>
      <div className="learning-replay-confirm-grid">
        <span>Step</span><strong>{step.index || pending?.stoppedStepOffset + 1}</strong>
        <span>Target</span><strong>{label}</strong>
        <span>Point</span><strong>{Math.round(Number(step.x || 0))}, {Math.round(Number(step.y || 0))}</strong>
      </div>
      {step.error && <small>{step.error}</small>}
      <footer>
        <button type="button" onClick={onCancel}>取消</button>
        <button type="button" className="learning-replay-confirm-primary" onClick={onConfirm}>確認繼續</button>
      </footer>
    </div>
  );
}
