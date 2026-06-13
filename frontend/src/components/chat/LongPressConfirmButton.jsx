import {t as _t} from '../../locales/useI18n';
import {PARTICLE_ANGLES} from '../../lib/appShared';

export default function LongPressConfirmButton({label, danger = false, progress = 0, gachaPhase = null, onPressStart, onPressEnd}) {
  // 轉蛋動畫完成 → 顯示「執行中」膠囊
  if (gachaPhase === 'reveal') {
    return (
      <span className="gacha-reveal">
        <span className="gacha-spinner" />
        {_t('confirm.execute')}
      </span>
    );
  }

  // 轉蛋動畫中間階段 → 顯示特效容器
  if (gachaPhase === 'collapse' || gachaPhase === 'pulse' || gachaPhase === 'particles') {
    return (
      <span style={{position: 'relative', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', minWidth: 120, minHeight: 44}}>
        {/* collapse 階段：按鈕縮小 */}
        {gachaPhase === 'collapse' && (
          <button type="button" className={`long-press-btn lp-completing ${danger ? 'lp-danger' : ''}`}>
            {label}
          </button>
        )}
        {/* pulse 階段：脈衝環向外擴散 */}
        {gachaPhase === 'pulse' && <span className="gacha-pulse" />}
        {/* particles 階段：8 個粒子向外散射 */}
        {gachaPhase === 'particles' && (
          <span className="gacha-particles">
            {PARTICLE_ANGLES.map((angle) => (
              <span
                key={angle}
                className="gacha-particle"
                style={{
                  transform: `translate(${Math.cos(angle * Math.PI / 180) * 30}px, ${Math.sin(angle * Math.PI / 180) * 30}px)`,
                }}
              />
            ))}
          </span>
        )}
      </span>
    );
  }

  // 正常狀態 → 可長按的確認按鈕
  return (
    <button
      type="button"
      className={`long-press-btn ${danger ? 'lp-danger' : ''}`}
      onMouseDown={onPressStart}
      onMouseUp={onPressEnd}
      onMouseLeave={onPressEnd}
      onTouchStart={onPressStart}
      onTouchEnd={onPressEnd}
      onTouchCancel={onPressEnd}
    >
      {label}
      {/* 長按進度條：從左到右填滿 */}
      <span className="lp-progress" style={{width: `${progress}%`}} />
    </button>
  );
}
