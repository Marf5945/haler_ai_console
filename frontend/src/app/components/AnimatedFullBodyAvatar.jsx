import React, {useEffect, useMemo, useRef, useState} from 'react';

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

const defaultMotion = {
  width: 200,
  height: 360,
  sway: 5,
  eyeRangeX: 4,
  eyeRangeY: 2.5,
  pupils: '#17110d',
  pupilShadow: 'rgba(255, 255, 255, .36)',
  layers: {
    head: {clip: 'inset(0% 20% 70% 20%)', origin: '50% 16%', x: 1.6, y: 1.2, rotate: 1.1, delay: '-.2s'},
    leftHand: {clip: 'inset(24% 62% 42% 0%)', origin: '24% 40%', x: 2.8, y: 2.4, rotate: -1.4, delay: '-1.1s'},
    rightHand: {clip: 'inset(30% 0% 38% 62%)', origin: '78% 48%', x: 2.4, y: 2.1, rotate: 1.2, delay: '-.7s'},
    feet: {clip: 'inset(78% 0% 0% 0%)', origin: '50% 90%', x: 1, y: .7, rotate: .4, delay: '-1.5s'},
  },
};

export const fullBodyMotionConfigs = {
  wolf: {
    ...defaultMotion,
    sway: 6,
    eyeRangeX: 4.8,
    eyeRangeY: 2.6,
    pupils: '#15100a',
    pupilShadow: 'rgba(255, 214, 82, .38)',
    eyes: [
      {key: 'left', x: 39.8, y: 9.9, w: 4.5, h: 1.45, rotate: -5},
      {key: 'right', x: 47.4, y: 9.7, w: 4.1, h: 1.35, rotate: -4},
    ],
    layers: {
      head: {clip: 'inset(0% 28% 75% 24%)', origin: '44% 12%', x: 2, y: 1.4, rotate: 1.3, delay: '-.1s'},
      leftHand: {clip: 'inset(18% 70% 48% 0%)', origin: '15% 31%', x: 3.4, y: 2.5, rotate: -2.2, delay: '-.8s'},
      rightHand: {clip: 'inset(36% 3% 36% 58%)', origin: '72% 50%', x: 2.4, y: 1.8, rotate: 1.3, delay: '-1.3s'},
      feet: {clip: 'inset(78% 0% 0% 6%)', origin: '48% 92%', x: 1.1, y: .8, rotate: .5, delay: '-1.8s'},
    },
  },
  uncle: {
    ...defaultMotion,
    sway: 3.6,
    eyeRangeX: 3.2,
    eyeRangeY: 1.6,
    pupils: '#1a0c18',
    pupilShadow: 'rgba(221, 82, 137, .34)',
    eyes: [
      {key: 'left', x: 43.1, y: 8.6, w: 4.7, h: 1.1, rotate: 4},
      {key: 'right', x: 52.6, y: 8.6, w: 4.5, h: 1.1, rotate: -4},
    ],
    layers: {
      head: {clip: 'inset(0% 28% 78% 28%)', origin: '50% 11%', x: 1.2, y: .9, rotate: .8, delay: '-.4s'},
      leftHand: {clip: 'inset(33% 70% 40% 0%)', origin: '17% 49%', x: 1.8, y: 1.3, rotate: -.9, delay: '-1.2s'},
      rightHand: {clip: 'inset(33% 0% 40% 70%)', origin: '83% 50%', x: 1.8, y: 1.3, rotate: .9, delay: '-.8s'},
      feet: {clip: 'inset(82% 0% 0% 0%)', origin: '50% 93%', x: .7, y: .5, rotate: .25, delay: '-1.5s'},
    },
  },
  secretary: {
    ...defaultMotion,
    sway: 4.5,
    eyeRangeX: 3.8,
    eyeRangeY: 2.2,
    pupils: '#162132',
    pupilShadow: 'rgba(114, 180, 255, .38)',
    eyes: [
      {key: 'left', x: 41.8, y: 11.5, w: 5.2, h: 1.45, rotate: -7},
      {key: 'right', x: 55.3, y: 11.2, w: 5.2, h: 1.45, rotate: -7},
    ],
    layers: {
      head: {clip: 'inset(0% 8% 74% 8%)', origin: '50% 13%', x: 1.4, y: 1.1, rotate: 1, delay: '-.2s'},
      leftHand: {clip: 'inset(20% 60% 52% 0%)', origin: '27% 31%', x: 2.4, y: 1.9, rotate: -1.6, delay: '-.7s'},
      rightHand: {clip: 'inset(30% 0% 48% 48%)', origin: '62% 39%', x: 1.8, y: 1.4, rotate: 1, delay: '-1.2s'},
      feet: {clip: 'inset(83% 0% 0% 0%)', origin: '50% 94%', x: .7, y: .6, rotate: .35, delay: '-1.7s'},
    },
  },
  police: {
    ...defaultMotion,
    sway: 3.8,
    eyeRangeX: 3.5,
    eyeRangeY: 1.7,
    pupils: '#18110d',
    pupilShadow: 'rgba(255, 185, 92, .3)',
    eyes: [
      {key: 'left', x: 44.2, y: 9.7, w: 4.2, h: 1.1, rotate: -4},
      {key: 'right', x: 54.2, y: 9.7, w: 4.2, h: 1.1, rotate: 4},
    ],
    layers: {
      head: {clip: 'inset(0% 26% 77% 25%)', origin: '50% 12%', x: 1.2, y: .9, rotate: .8, delay: '-.3s'},
      leftHand: {clip: 'inset(34% 69% 38% 0%)', origin: '18% 48%', x: 1.7, y: 1.2, rotate: -.9, delay: '-1s'},
      rightHand: {clip: 'inset(31% 0% 45% 59%)', origin: '69% 38%', x: 1.9, y: 1.4, rotate: 1.1, delay: '-.6s'},
      feet: {clip: 'inset(82% 0% 0% 0%)', origin: '50% 94%', x: .6, y: .5, rotate: .25, delay: '-1.7s'},
    },
  },
  touharu: {
    ...defaultMotion,
    sway: 6.2,
    eyeRangeX: 4.6,
    eyeRangeY: 2.4,
    pupils: '#2a1606',
    pupilShadow: 'rgba(255, 218, 118, .42)',
    eyes: [
      {key: 'left', x: 43.1, y: 12.2, w: 4.2, h: 1.35, rotate: -7},
      {key: 'right', x: 53.8, y: 12.2, w: 4.2, h: 1.35, rotate: 7},
    ],
    layers: {
      head: {clip: 'inset(0% 13% 70% 12%)', origin: '50% 14%', x: 2.2, y: 1.5, rotate: 1.5, delay: '-.1s'},
      leftHand: {clip: 'inset(27% 50% 48% 4%)', origin: '32% 37%', x: 2.4, y: 1.8, rotate: -1.2, delay: '-.9s'},
      rightHand: {clip: 'inset(26% 6% 48% 47%)', origin: '62% 36%', x: 2.3, y: 1.8, rotate: 1.2, delay: '-1.3s'},
      feet: {clip: 'inset(85% 0% 0% 0%)', origin: '50% 94%', x: .8, y: .55, rotate: .3, delay: '-1.6s'},
    },
  },
};

function layerStyle(layer = {}) {
  return {
    clipPath: layer.clip,
    transformOrigin: layer.origin,
    '--part-x': `${layer.x ?? 1}px`,
    '--part-y': `${layer.y ?? 1}px`,
    '--part-rotate': `${layer.rotate ?? 1}deg`,
    animationDelay: layer.delay || '0s',
  };
}

function targetFromPoint(root, clientX, clientY) {
  const rect = root?.getBoundingClientRect?.();
  if (!rect || rect.width <= 0 || rect.height <= 0) return {x: 0, y: 0};
  const centerX = rect.left + rect.width / 2;
  const centerY = rect.top + rect.height * .18;
  return {
    x: clamp((clientX - centerX) / Math.max(48, rect.width * .42), -1, 1),
    y: clamp((clientY - centerY) / Math.max(64, rect.height * .32), -1, 1),
  };
}

export default function AnimatedFullBodyAvatar({
  src,
  fallbackSrc = '',
  pack = 'wolf',
  mode = 'full',
  alt = '',
}) {
  const rootRef = useRef(null);
  const [look, setLook] = useState({x: 0, y: 0});
  const packKey = fullBodyMotionConfigs[pack] ? pack : 'wolf';
  const config = fullBodyMotionConfigs[packKey];
  const imageSrc = src || fallbackSrc;

  const style = useMemo(() => ({
    '--look-x': look.x.toFixed(3),
    '--look-y': look.y.toFixed(3),
    '--idle-sway': `${config.sway}px`,
    '--eye-range-x': `${config.eyeRangeX}px`,
    '--eye-range-y': `${config.eyeRangeY}px`,
    '--pupil-color': config.pupils,
    '--pupil-shadow': config.pupilShadow,
  }), [config, look.x, look.y]);

  useEffect(() => {
    const updateLook = (event) => {
      setLook(targetFromPoint(rootRef.current, event.clientX, event.clientY));
    };
    const settle = () => {
      window.setTimeout(() => setLook((current) => ({
        x: current.x * .45,
        y: current.y * .45,
      })), 180);
    };
    window.addEventListener('pointermove', updateLook);
    window.addEventListener('pointerdown', updateLook);
    window.addEventListener('pointerup', settle);
    return () => {
      window.removeEventListener('pointermove', updateLook);
      window.removeEventListener('pointerdown', updateLook);
      window.removeEventListener('pointerup', settle);
    };
  }, []);

  if (!imageSrc) return null;

  return (
    <span
      ref={rootRef}
      className={`animated-fullbody-avatar animated-fullbody-avatar-${packKey} animated-fullbody-avatar-${mode}`}
      style={style}
      aria-hidden={alt ? undefined : true}
      aria-label={alt || undefined}
      data-testid="animated-fullbody-avatar"
      data-pack={packKey}
    >
      <span className="animated-fullbody-avatar-layer animated-fullbody-avatar-body">
        <img src={imageSrc} alt={alt} draggable={false} />
      </span>
      {Object.entries(config.layers).map(([name, layer]) => (
        <span
          key={name}
          className={`animated-fullbody-avatar-layer animated-fullbody-avatar-rig animated-fullbody-avatar-${name}`}
          style={layerStyle(layer)}
          data-layer={name}
        />
      ))}
      <span className="animated-fullbody-avatar-eyes" aria-hidden="true">
        {(config.eyes || []).map((eye) => (
          <span
            key={eye.key}
            className={`animated-fullbody-avatar-eye animated-fullbody-avatar-eye-${eye.key}`}
            style={{
              left: `${eye.x}%`,
              top: `${eye.y}%`,
              width: `${eye.w}%`,
              height: `${eye.h}%`,
              transform: `translate(-50%, -50%) rotate(${eye.rotate || 0}deg)`,
            }}
            data-testid={`avatar-eye-${eye.key}`}
          />
        ))}
      </span>
    </span>
  );
}
