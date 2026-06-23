import {useRef} from 'react';
import {isNativeReplayStep, basenameForDisplay} from '../../lib/appHelpers';

export default function LearningReplayStartConfirmCard({pending, onConfirm, onCancel}) {
  const plan = pending?.plan || {};
  const steps = Array.isArray(plan.steps) ? plan.steps : [];
  const nativeSteps = steps.filter(isNativeReplayStep);
  const domSteps = steps.length - nativeSteps.length;
  const windows = Array.from(new Set(nativeSteps.map((step) => (
    basenameForDisplay(step.window_process || step.tag || step.window_title || 'unknown')
  )))).filter(Boolean).slice(0, 3);
  const first = steps[0] || {};
  const target = first.window_title || first.label || first.role || first.css_selector || first.tag || `座標 (${first.x || 0}, ${first.y || 0})`;
  const actionInvokedRef = useRef(false);
  const invokeOnce = (handler) => (event) => {
    event.preventDefault();
    event.stopPropagation();
    if (actionInvokedRef.current) return;
    actionInvokedRef.current = true;
    handler?.();
    window.setTimeout(() => {
      actionInvokedRef.current = false;
    }, 0);
  };
  const confirm = invokeOnce(onConfirm);
  const cancel = invokeOnce(onCancel);
  const keyActivate = (handler) => (event) => {
    if (event.key === 'Enter' || event.key === ' ') {
      handler(event);
    }
  };
  return (
    <div className="learning-replay-confirm" role="dialog" aria-label="Replay start confirmation">
      <header>
        <strong>確認 replay plan</strong>
        <span>尚未執行</span>
      </header>
      <p>我已讀到示範並產生安全 plan。確認下面內容正確後，才會真的移動滑鼠或點擊。</p>
      <div className="learning-replay-confirm-grid">
        <span>Run</span><strong>{plan.run_id || 'unknown'}</strong>
        <span>Title</span><strong>{plan.title || plan.run_name || 'untitled'}</strong>
        <span>Steps</span><strong>{steps.length}（本視窗 {domSteps}，外部 {nativeSteps.length}）</strong>
        <span>First</span><strong>{target}</strong>
        {windows.length > 0 && <><span>Windows</span><strong>{windows.join(', ')}</strong></>}
      </div>
      <footer>
        <button type="button" onPointerUp={cancel} onKeyDown={keyActivate(cancel)} onClick={cancel}>先不要</button>
        <button type="button" className="learning-replay-confirm-primary" onPointerUp={confirm} onKeyDown={keyActivate(confirm)} onClick={confirm}>執行 replay</button>
      </footer>
    </div>
  );
}
