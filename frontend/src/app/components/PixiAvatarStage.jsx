import React, {useEffect, useMemo, useRef, useState} from 'react';

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

// WKWebView can render transparent WebGL views as black; keep WebKit on the
// conservative alpha path while Chromium/WebView2 uses the default path.
const IS_WEBKIT_ENGINE = typeof navigator !== 'undefined'
  && /AppleWebKit/i.test(navigator.userAgent || '')
  && !/Chrome|Chromium|Edg\//i.test(navigator.userAgent || '');

const TICKLE_REACH = 58;
const PET_REACH = 34;
const TICKLE_POKE_SCALE = 0.58;
const TICKLE_MAX_POWER = 0.72;
const WALK_SPEED_SCALE = 5 / 8;
const LOOK_EASE_RATE = 8;
const TURN_EASE_RATE = 9.5;
const TURN_FAST_STEP_DELAY_MS = 42;
const TURN_SLOW_STEP_DELAY_MS = 68;
const TURN_CENTER_STEP_DELAY_MS = 32;
const ATTENTION_TURN_STEP_DELAY_MS = 72;
const POSE_FADE_IN_RATE = 9;
const POSE_FADE_OUT_RATE = 8;
const TURN_FADE_IN_RATE = 8;
const TURN_FADE_OUT_RATE = 10;
const TURN_FADE_CUTOFF = 0.08;
const BORED_AFTER_MS = 10000;
const LOOK_RETURN_DELAY_MS = 5000;
const ATTENTION_TRIGGER_DELAY_MS = 3500;
const ATTENTION_TURN_OUT_MS = 3000;
const ATTENTION_TURN_TOTAL_MS = 6000;
const ATTENTION_CENTER_WIDTH = 200;
const TURN_ENTER = [0, 0.035, 0.075, 0.12, 0.17, 0.225, 0.285, 0.35, 0.42, 0.5, 0.58, 0.66, 0.74, 0.82, 0.9, 0.97];
const TURN_SIDE_STEP = 9;
const TURN_CURSOR_START_STEP = 2;
const TURN_CURSOR_45PX = 92;
const TURN_CURSOR_SIDE_PX = 292;
const TURN_EXIT_GAP = 0.06;
const ATTENTION_FINISH_COOLDOWN_MS = 3000;
const PET_LOVE_HOLD_MS = 2000;
const PET_LOVE_COOLDOWN_MS = 12000;
const PIXI_IMAGE_SOURCE_WARN = 'ImageSource: Image element passed, converting to canvas. Use CanvasSource instead.';

function numberValue(value, fallback) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function installPixiImageSourceWarnFilter() {
  if (typeof console === 'undefined' || typeof console.warn !== 'function') return () => {};
  const originalWarn = console.warn;
  const filteredWarn = (...args) => {
    if (String(args?.[0] || '').includes(PIXI_IMAGE_SOURCE_WARN)) return;
    originalWarn(...args);
  };
  console.warn = filteredWarn;
  return () => {
    if (console.warn === filteredWarn) console.warn = originalWarn;
  };
}

function layerMotion(layer, manifestMotion) {
  return {
    breathe: {...(manifestMotion?.breathe || {}), ...(layer.motion?.breathe || {})},
    sway: {...(manifestMotion?.sway || {}), ...(layer.motion?.sway || {})},
    look: {...(manifestMotion?.look || {}), ...(layer.motion?.look || {})},
    blink: {...(manifestMotion?.blink || {}), ...(layer.motion?.blink || {})},
    mouth: {...(manifestMotion?.mouth || {}), ...(layer.motion?.mouth || {})},
    rotate: {...(manifestMotion?.rotate || {}), ...(layer.motion?.rotate || {})},
  };
}

function tickleLayerWeight(layerID = '') {
  const id = String(layerID || '').replace(/^viewer_/, '');
  if (/tail_(mid|tip)|ear|muzzle|mouth|eye|brow|hand|wrist/.test(id)) return 1;
  if (/head|forearm|upper_arm/.test(id)) return 0.42;
  if (/tail_base/.test(id)) return 0.28;
  return 0;
}

function frameSequenceDurationMs(def) {
  const frames = Array.isArray(def?.frames) ? def.frames.length : 0;
  if (!frames) return 0;
  return (frames / Math.max(1, numberValue(def?.fps, 8))) * 1000;
}

function targetFromPoint(element, clientX, clientY) {
  const rect = element?.getBoundingClientRect?.();
  if (!rect || rect.width <= 0 || rect.height <= 0) return {x: 0, y: 0, turnX: 0};
  const centerX = rect.left + rect.width / 2;
  const centerY = rect.top + rect.height * 0.22;
  const dx = clientX - centerX;
  const turnAbsPx = Math.abs(dx);
  const turnSign = dx < 0 ? -1 : 1;
  const sideProgress = clamp(
    (turnAbsPx - TURN_CURSOR_45PX) / Math.max(1, TURN_CURSOR_SIDE_PX - TURN_CURSOR_45PX),
    0,
    1,
  );
  const turnX = turnAbsPx < TURN_CURSOR_45PX
    ? 0
    : turnSign * (TURN_ENTER[TURN_CURSOR_START_STEP] + (TURN_ENTER[TURN_SIDE_STEP] - TURN_ENTER[TURN_CURSOR_START_STEP]) * sideProgress);
  return {
    x: clamp((clientX - centerX) / Math.max(42, rect.width * 0.44), -1, 1),
    y: clamp((clientY - centerY) / Math.max(58, rect.height * 0.34), -1, 1),
    turnX,
  };
}

function fitScale(texture, size, fallback = 1) {
  if (!size?.width && !size?.height) return fallback;
  const widthScale = size.width ? numberValue(size.width, texture.width) / Math.max(1, texture.width) : Infinity;
  const heightScale = size.height ? numberValue(size.height, texture.height) / Math.max(1, texture.height) : Infinity;
  return Math.min(widthScale, heightScale);
}

function layerPosition(layer, fallbackX, fallbackY) {
  const positionPx = layer.positionPx || {};
  return {
    x: numberValue(positionPx.x, fallbackX),
    y: numberValue(positionPx.y, fallbackY),
  };
}

function applyRigRoot(root, width, height, designSize) {
  const designWidth = numberValue(designSize?.width, width);
  const designHeight = numberValue(designSize?.height, height);
  const scale = Math.min(width / Math.max(1, designWidth), height / Math.max(1, designHeight));
  root.scale.set(scale);
  root.x = (width - designWidth * scale) / 2;
  root.y = (height - designHeight * scale) / 2;
}

// Pivot around a rough joint position instead of the sprite center.
function layerPivot(layer, texture, visualScale) {
  const explicit = layer.pivotPx;
  if (explicit) {
    return {x: numberValue(explicit.x, 0), y: numberValue(explicit.y, 0)};
  }
  if (!layer.parentId) return {x: 0, y: 0};
  const position = layerPosition(layer, 0, 0);
  const halfW = Math.max(4, (texture?.width || 0) * Math.abs(visualScale) / 2) * 1.1;
  const halfH = Math.max(4, (texture?.height || 0) * Math.abs(visualScale) / 2) * 1.1;
  return {
    x: clamp(-position.x / 2, -halfW, halfW),
    y: clamp(-position.y / 2, -halfH, halfH),
  };
}

function applyRigLayout(node, layer) {
  const sprite = node.__sprite;
  const anchor = layer.anchor || {};
  const position = layerPosition(layer, 0, 0);
  const visualScale = fitScale(sprite.texture, layer.size, 1) * numberValue(layer.scale, 1);
  const scaleX = visualScale * (layer.flipX ? -1 : 1);
  const scaleY = visualScale * (layer.flipY ? -1 : 1);

  sprite.anchor.set(numberValue(anchor.x, 0.5), numberValue(anchor.y, 0.5));
  sprite.x = numberValue(layer.offsetPx?.x, 0);
  sprite.y = numberValue(layer.offsetPx?.y, 0);
  sprite.scale.set(scaleX, scaleY);
  node.__spriteBase = {
    x: sprite.x,
    y: sprite.y,
    scaleX,
    scaleY,
  };

  const pivot = layerPivot(layer, sprite.texture, visualScale);
  node.pivot.set(pivot.x, pivot.y);

  node.__base = {
    x: position.x + pivot.x,
    y: position.y + pivot.y,
    scale: numberValue(layer.containerScale, 1),
    rotation: numberValue(layer.rotation, 0),
    alpha: numberValue(layer.alpha, 1),
  };

  node.x = node.__base.x;
  node.y = node.__base.y;
  node.scale.set(node.__base.scale);
  node.rotation = node.__base.rotation;
  node.alpha = node.__base.alpha;
}

export default function PixiAvatarStage({
  manifest,
  fallbackSrc = '',
  expression = 'idle',
  width = 200,
  height = 360,
  alt = '',
  tickle = false,
  onReady,
  onAttentionPauseChange,
  onPetComplete,
}) {
  const hostRef = useRef(null);
  const lookRef = useRef({x: 0, y: 0});
  const targetLookRef = useRef({x: 0, y: 0, turnX: 0});
  const [failed, setFailed] = useState(false);
  const [ready, setReady] = useState(false);
  const onReadyRef = useRef(onReady);
  const onAttentionPauseChangeRef = useRef(onAttentionPauseChange);
  const onPetCompleteRef = useRef(onPetComplete);
  const stateKey = manifest?.state || expression || 'idle';
  const manifestKey = useMemo(() => {
    try {
      return JSON.stringify(manifest);
    } catch (_) {
      return String(manifest?.id || '');
    }
  }, [manifest]);
  const pointerRef = useRef({x: 0, y: 0, speed: 0, inside: false, poke: 0, lastX: 0, lastY: 0, lastT: 0});
  const tickleModeRef = useRef(tickle);
  useEffect(() => {
    onReadyRef.current = onReady;
  }, [onReady]);

  useEffect(() => {
    onAttentionPauseChangeRef.current = onAttentionPauseChange;
  }, [onAttentionPauseChange]);

  useEffect(() => {
    onPetCompleteRef.current = onPetComplete;
  }, [onPetComplete]);

  useEffect(() => {
    tickleModeRef.current = tickle;
    if (!tickle) {
      const p = pointerRef.current;
      p.inside = false;
      p.speed = 0;
      p.poke = 0;
    }
  }, [tickle]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return undefined;

    setReady(false);
    setFailed(false);
    let cancelled = false;
    let app = null;
    let initialized = false;
    let resizeObserver = null;
    let tickerHandler = null;
    let tmpPointRef = null;
    const restorePixiWarnFilter = installPixiImageSourceWarnFilter();

    const ctl = {
      lastActive: performance.now(),
      nextBehaviorAt: performance.now() + (manifest?.id === 'yulesaku' ? 60000 : BORED_AFTER_MS),
      behavior: null,
      behaviorStarted: 0,
      behaviorUntil: 0,
      behaviorIndex: 0,
      attentionBehaviorIndex: 0,
      turnValue: 0,
      turnSwitched: 0,
      turnTransitionKey: '',
      turnTransitionUntil: 0,
    };
    const pet = {lift: 0, arm: 'right'};
    const petLove = {start: 0, cooldownUntil: 0};
    const shy = {petStart: 0, active: false, phaseIndex: -1, phaseUntil: 0, threshold: 0, cooldownUntil: 0};
    const attention = {paused: false, side: null, timer: null, cooldownUntil: 0, startedAt: 0};
    let nextAttentionFinishBehavior = () => null;

    const clearAttentionTimer = () => {
      if (attention.timer) {
        window.clearTimeout(attention.timer);
        attention.timer = null;
      }
    };

    const setAttentionPaused = (paused, now = performance.now(), finishBehavior = null) => {
      if (attention.paused === paused) return;
      attention.paused = paused;
      attention.startedAt = paused ? now : 0;
      if (paused) {
        targetLookRef.current = {x: 0, y: 0, turnX: 0};
        ctl.behavior = finishBehavior;
        ctl.nextBehaviorAt = now + (manifest?.id === 'yulesaku' ? 60000 : BORED_AFTER_MS);
        if (finishBehavior) {
          ctl.behaviorStarted = now;
          ctl.behaviorUntil = now + Math.max(
            numberValue(finishBehavior.duration, 0),
            ATTENTION_TURN_TOTAL_MS + ATTENTION_TURN_STEP_DELAY_MS * 3,
          );
        }
      } else {
        if (ctl.behavior?.source === 'attention') ctl.behavior = null;
        attention.cooldownUntil = now + ATTENTION_FINISH_COOLDOWN_MS;
      }
      onAttentionPauseChangeRef.current?.(paused);
    };
    async function start() {
      try {
        const rigEnabled = manifest?.renderRig !== false && manifest?.layers?.length > 0;
        const mainDef = {
          layers: rigEnabled ? manifest.layers : [],
          baseSrc: manifest?.baseSrc || fallbackSrc,
          sequence: manifest?.sequence || null,
          motion: manifest?.motion || {},
        };
        if (!mainDef.layers.length && !mainDef.baseSrc) {
          throw new Error('Pixi avatar has no renderable layers');
        }

        const {Application, Assets, Container, Sprite, Point, loadTextures} = await import('pixi.js');
        await import('pixi.js/unsafe-eval');
        if (loadTextures?.config) {
          loadTextures.config.preferWorkers = false;
          loadTextures.config.preferCreateImageBitmap = false;
        }
        if (cancelled) return;
        const tmpPoint = new Point();
        tmpPointRef = tmpPoint;
        const instance = new Application();
        await instance.init({
          width,
          height,
          backgroundAlpha: 0,
          antialias: !IS_WEBKIT_ENGINE,
          premultipliedAlpha: !IS_WEBKIT_ENGINE,
          autoDensity: true,
          resolution: Math.min(window.devicePixelRatio || 1, 2),
          preference: 'webgl',
        });
        if (cancelled) {
          try {
            instance.destroy(true, {children: true, texture: false, textureSource: false});
          } catch (_) {
            /* Ignore destroy errors from a partially initialized Pixi app. */
          }
          return;
        }
        app = instance;
        initialized = true;

        app.canvas.className = 'pixi-avatar-canvas';
        app.canvas.setAttribute('aria-hidden', alt ? 'false' : 'true');
        host.replaceChildren(app.canvas);

        const designSize = manifest?.size || {width, height};

        const groups = new Map();
        const groupDefs = new Map();
        groupDefs.set('turn:0', mainDef);
        const turnEnabled = Array.isArray(manifest?.turnStates) && manifest.turnStates.length > 0;
        if (turnEnabled) {
          manifest.turnStates.forEach((ts) => {
            groupDefs.set(`turn:${ts.value}`, {layers: ts.layers || [], baseSrc: ts.baseSrc, motion: ts.motion || {}});
          });
          Object.entries(manifest.turnTransitions || {}).forEach(([id, transition]) => {
            const from = numberValue(transition?.from, NaN);
            const to = numberValue(transition?.to, NaN);
            if (!Number.isFinite(from) || !Number.isFinite(to) || !transition?.frames?.length) return;
            groupDefs.set(`turn-transition:${from}:${to}`, {
              baseSrc: transition.frames[0],
              sequence: transition,
              motion: manifest.motion || {},
              transitionId: id,
            });
          });
          (manifest.poseStates || []).forEach((ps) => {
            groupDefs.set(`pose:${ps.id}`, {layers: ps.layers || [], baseSrc: ps.baseSrc, sequence: ps.sequence || null, motion: ps.motion || {}});
          });
          (manifest.shyStates || []).forEach((ps) => {
            if (!groupDefs.has(`pose:${ps.id}`)) {
              groupDefs.set(`pose:${ps.id}`, {layers: ps.layers || [], baseSrc: ps.baseSrc, motion: ps.motion || {}});
            }
          });
          if (manifest.walkFrames?.length) {
            if (manifest.walkStartFrames?.length) {
              groupDefs.set('walk_start', {
                kind: 'walk',
                frames: manifest.walkStartFrames,
                fps: numberValue(manifest.sequences?.walk_start?.fps, 8),
                loop: false,
              });
            }
            groupDefs.set('walk', {
              kind: 'walk',
              frames: manifest.walkFrames,
              fps: numberValue(manifest.sequences?.walk_forward?.fps, 9) * WALK_SPEED_SCALE,
              loop: true,
            });
            if (manifest.walkStopFrames?.length) {
              groupDefs.set('walk_stop', {
                kind: 'walk',
                frames: manifest.walkStopFrames,
                fps: numberValue(manifest.sequences?.walk_stop?.fps, 8),
                loop: false,
              });
            }
          }
        }
        const walkStartDurationMs = frameSequenceDurationMs(groupDefs.get('walk_start'));
        const walkStopDurationMs = frameSequenceDurationMs(groupDefs.get('walk_stop'));
        const behaviorPlaylist = [];
        const attentionFinishPlaylist = [];
        if (turnEnabled) {
          const happyTailPose = groupDefs.has('pose:happy_tail_wag_front')
            ? {key: 'pose:happy_tail_wag_front', duration: stateKey === 'happy' ? 3600 : 2600}
            : null;
          if (happyTailPose) attentionFinishPlaylist.push({...happyTailPose, source: 'attention'});
          if (groupDefs.has('walk')) attentionFinishPlaylist.push({key: 'walk', duration: 5000, source: 'attention'});
          if (stateKey === 'happy' && happyTailPose) {
            behaviorPlaylist.push(happyTailPose, happyTailPose, happyTailPose);
          }
          if (groupDefs.has('walk')) behaviorPlaylist.push({key: 'walk', duration: 7000});
          if (groupDefs.has('pose:working_reaction')) behaviorPlaylist.push({key: 'pose:working_reaction', duration: 2500});
          if (happyTailPose) behaviorPlaylist.push(happyTailPose);
          if (groupDefs.has('pose:front_left30_arms_up')) behaviorPlaylist.push({key: 'pose:front_left30_arms_up', duration: 4500});
          if (groupDefs.has('walk')) behaviorPlaylist.push({key: 'walk', duration: 5500});
          if (groupDefs.has('pose:back_waist')) behaviorPlaylist.push({key: 'pose:back_waist', duration: 5000});
          if (stateKey === 'happy' && happyTailPose) behaviorPlaylist.push(happyTailPose, happyTailPose);
          if (groupDefs.has('pose:working_reaction')) behaviorPlaylist.push({key: 'pose:working_reaction', duration: 2500});
          if (groupDefs.has('pose:front_right30_arms_up')) behaviorPlaylist.push({key: 'pose:front_right30_arms_up', duration: 4500});
        }
        nextAttentionFinishBehavior = () => {
          if (!attentionFinishPlaylist.length) {
            return groupDefs.has('pose:working_reaction')
              ? {key: 'pose:working_reaction', duration: ATTENTION_TURN_TOTAL_MS, source: 'attention'}
              : null;
          }
          const next = attentionFinishPlaylist[ctl.attentionBehaviorIndex % attentionFinishPlaylist.length];
          ctl.attentionBehaviorIndex += 1;
          return {...next, duration: ATTENTION_TURN_TOTAL_MS};
        };
        const shySteps = (manifest.shyStates || [])
          .map((ps) => `pose:${ps.id}`)
          .filter((k) => groupDefs.has(k));
        const shyDurations = [1600, 2400];
        const shyEnabled = turnEnabled && shySteps.length > 0;

        function layoutGroup(group) {
          applyRigRoot(group.root, app.screen.width, app.screen.height, designSize);
          group.__rootBase = {x: group.root.x, y: group.root.y};
          if (group.kind === 'rig') {
            group.nodes.forEach((node) => applyRigLayout(node, node.__layer));
          }
        }

        async function buildGroup(key, def) {
          if (groups.has(key)) return groups.get(key);
          const group = {key, def, root: new Container(), nodes: [], fade: 0, ready: false, kind: 'pending'};
          groups.set(key, group);
          group.root.label = `avatar-${key}`;
          group.root.alpha = 0;
          group.root.visible = false;
          app.stage.addChild(group.root);

          if (def.kind === 'walk') {
            const textures = [];
            for (const src of def.frames) {
              try {
                const texture = await Assets.load(src);
                textures.push(texture);
              } catch (err) {
                console.warn('[PixiAvatarStage] walk frame load failed', err);
              }
              if (cancelled) return group;
            }
            if (textures.length) {
              const sprite = new Sprite(textures[0]);
              sprite.width = numberValue(designSize.width, width);
              sprite.height = numberValue(designSize.height, height);
              group.root.addChild(sprite);
              group.__walkSprite = sprite;
              group.__walkTextures = textures;
              group.kind = 'walk';
              group.ready = true;
            }
            layoutGroup(group);
            return group;
          }

          const layers = (def.layers || []).filter((layer) => layer.src);
          if (layers.length) {
            const nodeMap = new Map();
            for (const layer of layers) {
              let texture = null;
              let sequenceTextures = [];
              try {
                texture = await Assets.load(layer.src);
                if (layer.sequence?.frames?.length) {
                  sequenceTextures = (await Promise.all(layer.sequence.frames.map((src) => Assets.load(src))))
                    .filter(Boolean);
                  if (sequenceTextures[0]) texture = sequenceTextures[0];
                }
              } catch (err) {
                console.warn('[PixiAvatarStage] layer load failed:', layer.id, err);
                continue;
              }
              if (cancelled) return group;
              const sprite = new Sprite(texture);
              sprite.label = layer.id || 'layer';
              const node = new Container();
              node.label = layer.id || 'layer';
              node.__sprite = sprite;
              node.__layer = layer;
              node.__motion = layerMotion(layer, def.motion || {});
              node.__sequence = layer.sequence || null;
              node.__sequenceTextures = sequenceTextures;
              node.addChild(sprite);
              applyRigLayout(node, layer);
              group.nodes.push(node);
              nodeMap.set(layer.id || `layer-${group.nodes.length}`, node);
            }
            group.nodes.forEach((node) => {
              const parent = nodeMap.get(node.__layer.parentId);
              if (node.__layer.parentId && !parent) {
                node.visible = false;
              }
              (parent || group.root).addChild(node);
            });
            group.kind = 'rig';
            group.ready = group.nodes.length > 0;
          } else if (def.baseSrc) {
            try {
              const texture = await Assets.load(def.baseSrc);
              if (cancelled) return group;
              const sequenceTextures = def.sequence?.frames?.length
                ? (await Promise.all(def.sequence.frames.map((src) => Assets.load(src)))).filter(Boolean)
                : [];
              if (cancelled) return group;
              const sprite = new Sprite(texture);
              if (sequenceTextures[0]) sprite.texture = sequenceTextures[0];
              sprite.width = numberValue(designSize.width, width);
              sprite.height = numberValue(designSize.height, height);
              group.root.addChild(sprite);
              group.__flatSprite = sprite;
              group.__flatSequence = def.sequence || null;
              group.__flatTextures = sequenceTextures;
              group.kind = 'flat';
              group.ready = true;
            } catch (err) {
              console.warn('[PixiAvatarStage] flat pose load failed', err);
            }
          }
          layoutGroup(group);
          return group;
        }

        const mainGroup = await buildGroup('turn:0', mainDef);
        if (cancelled) return;
        if (!mainGroup.ready) throw new Error('Pixi avatar main pose failed to load');
        mainGroup.fade = 1;
        mainGroup.root.alpha = 1;
        mainGroup.root.visible = true;

        if (groupDefs.size > 1) {
          (async () => {
            for (const [key, def] of groupDefs) {
              if (cancelled) return;
              if (!groups.has(key)) {
                try {
                  await buildGroup(key, def);
                } catch (err) {
                  console.warn('[PixiAvatarStage] pose preload failed', key, err);
                }
              }
            }
          })();
        }

        const resize = () => {
          const rect = host.getBoundingClientRect();
          const nextWidth = Math.max(1, Math.round(rect.width || width));
          const nextHeight = Math.max(1, Math.round(rect.height || height));
          app.renderer.resize(nextWidth, nextHeight);
          groups.forEach((group) => layoutGroup(group));
        };
        resizeObserver = new ResizeObserver(resize);
        resizeObserver.observe(host);
        resize();

        tickerHandler = (ticker) => {
          const time = performance.now() / 1000;
          const now = performance.now();
          const delta = ticker.deltaTime / 60;
          const look = lookRef.current;
          const lookTarget = targetLookRef.current;
          const lookAlpha = Math.min(1, delta * LOOK_EASE_RATE);
          const turnAlpha = Math.min(1, delta * TURN_EASE_RATE);
          look.x += (lookTarget.x - look.x) * lookAlpha;
          look.y += (lookTarget.y - look.y) * lookAlpha;
          look.turnX = numberValue(look.turnX, 0)
            + (numberValue(lookTarget.turnX, 0) - numberValue(look.turnX, 0)) * turnAlpha;
          const pointer = pointerRef.current;
          const tickleOn = tickleModeRef.current;
          pointer.speed *= 0.82;
          pointer.poke *= 0.86;
          if (!tickleOn) {
            pointer.poke = 0;
            pointer.speed *= 0.4;
          }
          const speedNorm = clamp(pointer.speed / 1.4, 0, 1);
          if (!attention.paused && now - ctl.lastActive >= LOOK_RETURN_DELAY_MS) {
            targetLookRef.current = {x: 0, y: 0, turnX: 0};
          }
          if (attention.paused) {
            const attentionElapsed = now - attention.startedAt;
            const sideSign = attention.side === 'left' ? -1 : 1;
            const turnTarget = attentionElapsed < ATTENTION_TURN_OUT_MS
              ? sideSign * TURN_ENTER[TURN_ENTER.length - 1]
              : 0;
            targetLookRef.current = {x: 0, y: 0, turnX: turnTarget};
            ctl.lastActive = now;
            ctl.nextBehaviorAt = now + (manifest?.id === 'yulesaku' ? 60000 : BORED_AFTER_MS);
            if (attentionElapsed >= ATTENTION_TURN_TOTAL_MS && ctl.turnValue === 0) {
              setAttentionPaused(false, now);
            }
          }

          if (tickleOn && ctl.behavior) {
            ctl.behavior = null;
            ctl.nextBehaviorAt = now + (manifest?.id === 'yulesaku' ? 60000 : BORED_AFTER_MS);
          }
          if (turnEnabled) {
            if (ctl.behavior && ctl.behavior.source !== 'attention' && ctl.lastActive > ctl.behaviorStarted) {
              ctl.behavior = null;
              ctl.nextBehaviorAt = now + (manifest?.id === 'yulesaku' ? 60000 : BORED_AFTER_MS);
            } else if (ctl.behavior && now > ctl.behaviorUntil) {
              const wasAttentionBehavior = ctl.behavior.source === 'attention';
              ctl.behavior = null;
              if (wasAttentionBehavior && attention.paused) {
                setAttentionPaused(false, now);
                ctl.lastActive = now;
                ctl.nextBehaviorAt = now + (manifest?.id === 'yulesaku' ? 60000 : BORED_AFTER_MS);
              } else {
                ctl.nextBehaviorAt = manifest?.id === 'yulesaku'
                  ? now + 60000
                  : stateKey === 'happy'
                  ? now + 900 + Math.random() * 1800
                  : now + 6000 + Math.random() * 6000;
              }
            }
            const behaviorIdleDelay = manifest?.id === 'yulesaku'
              ? 60000
              : stateKey === 'happy'
                ? 1200
                : 9500;
            if (!tickleOn && !ctl.behavior && behaviorPlaylist.length
              && now - ctl.lastActive > behaviorIdleDelay && now >= ctl.nextBehaviorAt) {
              const next = behaviorPlaylist[Math.floor(Math.random() * behaviorPlaylist.length)];
              ctl.behaviorIndex += 1;
              const g = groups.get(next.key);
              if (g?.ready) {
                ctl.behavior = next;
                ctl.behaviorStarted = now;
                ctl.behaviorUntil = now + next.duration;
              }
            }
            const lx = look.turnX || 0;
            const mag = Math.abs(lx);
            const sgn = lx < 0 ? -1 : 1;
            let desired = 0;
            for (let i = TURN_ENTER.length - 1; i >= 1; i -= 1) {
              if (mag >= TURN_ENTER[i]) {
                desired = i;
                break;
              }
            }
            desired *= sgn;
            const cur = ctl.turnValue;
            const crossesCenter = cur !== 0 && desired !== 0 && Math.sign(cur) !== Math.sign(desired);
            const turnDelay = attention.paused
              ? ATTENTION_TURN_STEP_DELAY_MS
              : crossesCenter
              ? TURN_CENTER_STEP_DELAY_MS
              : (speedNorm > 0.45 ? TURN_FAST_STEP_DELAY_MS : TURN_SLOW_STEP_DELAY_MS);
            if (desired !== cur && now - ctl.turnSwitched > turnDelay) {
              let target = crossesCenter ? 0 : cur + (desired > cur ? 1 : -1);
              if (desired === 0 && Math.abs(cur) <= 2) target = 0;
              const growing = Math.abs(target) > Math.abs(cur);
              const need = growing
                ? TURN_ENTER[Math.abs(target)]
                : TURN_ENTER[Math.abs(cur)] - TURN_EXIT_GAP;
              const ok = target === 0 || (growing ? mag >= need : mag < need);
              if (ok && groupDefs.has(`turn:${target}`)) {
                const transitionKey = `turn-transition:${cur}:${target}`;
                const transitionDef = groupDefs.get(transitionKey);
                if (transitionDef?.sequence?.frames?.length) {
                  ctl.turnTransitionKey = transitionKey;
                  ctl.turnTransitionUntil = now + frameSequenceDurationMs(transitionDef.sequence);
                } else {
                  ctl.turnTransitionKey = '';
                  ctl.turnTransitionUntil = 0;
                }
                ctl.turnValue = target;
                ctl.turnSwitched = now;
              }
            }
          }

          let petTarget = 0;
          const probeGroup = groups.get(ctl.__activeKey || 'turn:0');
          if (probeGroup?.kind === 'rig' && pointer.inside
            && pointer.speed > 0.015 && pointer.speed < 0.6) {
            for (const node of probeGroup.nodes) {
              if (!node.visible) continue;
              const id = (node.__layer?.id || '').replace('viewer_', '');
              if (!/hand|wrist/.test(id)) continue;
              const gp = node.getGlobalPosition(tmpPoint);
              const dist = Math.hypot(gp.x - pointer.x, gp.y - pointer.y);
              if (dist < PET_REACH) {
                petTarget = 1;
                pet.arm = /right/.test(id) ? 'right' : 'left';
                break;
              }
            }
          }
          pet.lift += (petTarget - pet.lift) * Math.min(1, delta * (petTarget ? 2.4 : 1.1));
          if (pet.lift < 0.004) pet.lift = 0;

          if (pet.lift > 0.5 && now > petLove.cooldownUntil) {
            if (!petLove.start) {
              petLove.start = now;
            } else if (now - petLove.start >= PET_LOVE_HOLD_MS) {
              petLove.start = 0;
              petLove.cooldownUntil = now + PET_LOVE_COOLDOWN_MS;
              onPetCompleteRef.current?.();
            }
          } else if (pet.lift < 0.15 && petLove.start) {
            petLove.start = 0;
          }

          if (shyEnabled) {
            if (!shy.active) {
              if (pet.lift > 0.5 && now > shy.cooldownUntil) {
                if (!shy.petStart) {
                  shy.petStart = now;
                  shy.threshold = 2000 + Math.random() * 3000;
                } else if (now - shy.petStart > shy.threshold) {
                  shy.active = true;
                  shy.phaseIndex = 0;
                  shy.phaseUntil = now + (shyDurations[0] || 1600);
                }
              } else if (pet.lift < 0.15 && shy.petStart) {
                shy.petStart = 0;
              }
            } else if (now > shy.phaseUntil) {
              shy.phaseIndex += 1;
              if (shy.phaseIndex >= shySteps.length) {
                shy.active = false;
                shy.phaseIndex = -1;
                shy.petStart = 0;
                shy.cooldownUntil = now + 10000;
                pet.lift = 0;
              } else {
                shy.phaseUntil = now + (shyDurations[shy.phaseIndex] || 2400);
              }
            }
          }

          let key = 'turn:0';
          if (shy.active && shySteps[shy.phaseIndex]) key = shySteps[shy.phaseIndex];
          else if (attention.paused && ctl.turnValue === 0 && groupDefs.has('pose:working_reaction')) key = 'pose:working_reaction';
          else if (ctl.behavior?.key === 'walk') {
            const behaviorElapsed = now - ctl.behaviorStarted;
            const behaviorRemaining = ctl.behaviorUntil - now;
            if (walkStartDurationMs > 0 && behaviorElapsed < walkStartDurationMs) {
              key = 'walk_start';
            } else if (walkStopDurationMs > 0 && behaviorRemaining < walkStopDurationMs) {
              key = 'walk_stop';
            } else {
              key = 'walk';
            }
          } else if (ctl.behavior) key = ctl.behavior.key;
          else if (ctl.turnValue !== 0) key = `turn:${ctl.turnValue}`;
          if (!groups.get(key)?.ready) {
            if (!groups.has(key) && groupDefs.has(key)) {
              buildGroup(key, groupDefs.get(key)).catch(() => {});
            }
            key = 'turn:0';
          }
          const previousActiveKey = ctl.__activeKey;
          ctl.__activeKey = key;
          if (previousActiveKey !== key) {
            const activeGroup = groups.get(key);
            if (activeGroup) activeGroup.__playStartedAt = now;
          }

          const activeIsTurn = key.startsWith('turn:');
          groups.forEach((group) => {
            const target = group.key === key ? 1 : 0;
            const groupIsTurn = group.key.startsWith('turn:');
            const rate = activeIsTurn && groupIsTurn
              ? (target === 1 ? TURN_FADE_IN_RATE : TURN_FADE_OUT_RATE)
              : (target === 1 ? POSE_FADE_IN_RATE : POSE_FADE_OUT_RATE);
            group.fade += (target - group.fade) * Math.min(1, delta * rate);
            if (activeIsTurn && groupIsTurn && target === 0 && group.fade < TURN_FADE_CUTOFF) group.fade = 0;
            if (group.fade < 0.012 && target === 0) group.fade = 0;
            group.root.alpha = Math.min(1, group.fade);
            group.root.visible = group.fade > 0.01;
          });

          groups.forEach((group) => {
            if (!group.root.visible || !group.ready) return;
            if (group.kind === 'walk') {
              const textures = group.__walkTextures;
              const walkFps = Math.max(1, numberValue(group.def?.fps, 9));
              const elapsed = Math.max(0, (now - (group.__playStartedAt || now)) / 1000);
              const frame = group.def?.loop === false
                ? Math.min(textures.length - 1, Math.floor(elapsed * walkFps))
                : Math.floor(time * walkFps) % textures.length;
              if (group.__walkSprite.texture !== textures[frame]) {
                group.__walkSprite.texture = textures[frame];
              }
              group.root.x = group.__rootBase.x
                + Math.sin(time * 0.5) * 16 * group.root.scale.x;
              return;
            }
            if (group.kind === 'flat') {
              if (group.__flatTextures?.length > 1) {
                const sequenceFps = Math.max(1, numberValue(group.__flatSequence?.fps, 8));
                const elapsed = Math.max(0, (now - (group.__playStartedAt || now)) / 1000);
                const rawFrame = group.__flatSequence?.loop === false
                  ? Math.floor(elapsed * sequenceFps)
                  : Math.floor(time * sequenceFps);
                const frame = group.__flatSequence?.loop === false
                  ? Math.min(group.__flatTextures.length - 1, rawFrame)
                  : rawFrame % group.__flatTextures.length;
                if (group.__flatSprite.texture !== group.__flatTextures[frame]) {
                  group.__flatSprite.texture = group.__flatTextures[frame];
                }
              }
              return;
            }
            const isActive = group.key === key;
            group.nodes.forEach((node, index) => {
              const base = node.__base;
              const motion = node.__motion;
              const layer = node.__layer;
              if (node.__sequenceTextures?.length > 1) {
                const sequenceFps = Math.max(1, numberValue(node.__sequence?.fps, 8));
                const rawFrame = Math.floor(time * sequenceFps);
                const frame = node.__sequence?.loop === false
                  ? Math.min(node.__sequenceTextures.length - 1, rawFrame)
                  : rawFrame % node.__sequenceTextures.length;
                if (node.__sprite.texture !== node.__sequenceTextures[frame]) {
                  node.__sprite.texture = node.__sequenceTextures[frame];
                }
              }
              const phase = numberValue(layer.phase, index * 0.37);
              const breathe = motion.breathe || {};
              const sway = motion.sway || {};
              const lookMotion = motion.look || {};
              const blink = motion.blink || {};
              const mouth = motion.mouth || {};
              const rotate = motion.rotate || {};
              const spriteBase = node.__spriteBase;
              if (spriteBase) {
                node.__sprite.x = spriteBase.x;
                node.__sprite.y = spriteBase.y;
                node.__sprite.scale.x = spriteBase.scaleX;
                node.__sprite.scale.y = spriteBase.scaleY;
              }
              const breatheWave = Math.sin(time * numberValue(breathe.speed, 1.1) + phase);
              const swayWave = Math.sin(time * numberValue(sway.speed, 0.72) + phase);
              const blinkWindow = Math.sin(time * numberValue(blink.speed, 0.33) + phase) > 0.985;
              const mouthWave = stateKey === 'working'
                ? Math.max(0, Math.sin(time * numberValue(mouth.speed, 5.4) + phase))
                : 0;
              const rotateWave = Math.sin(time * numberValue(rotate.speed, 1) + phase);
              const localOnlyBreathe = breathe.localOnly === true;

              node.x = base.x
                + swayWave * numberValue(sway.x, 0)
                + look.x * numberValue(lookMotion.x, 0);
              node.y = base.y
                - (localOnlyBreathe ? 0 : breatheWave * numberValue(breathe.y, 0))
                + look.y * numberValue(lookMotion.y, 0)
                - mouthWave * numberValue(mouth.y, 0);
              node.rotation = base.rotation
                + swayWave * numberValue(sway.rotate, 0)
                + look.x * numberValue(lookMotion.rotate, 0)
                + rotateWave * numberValue(rotate.amount, 0);
              node.scale.x = base.scale * (1 + breatheWave * (localOnlyBreathe ? 0 : numberValue(breathe.scaleX, 0)));
              node.scale.y = base.scale * (1 + breatheWave * (localOnlyBreathe ? 0 : numberValue(breathe.scaleY, 0)));
              if (localOnlyBreathe && spriteBase) {
                node.__sprite.y = spriteBase.y - breatheWave * numberValue(breathe.y, 0);
                node.__sprite.scale.x = spriteBase.scaleX * (1 + breatheWave * numberValue(breathe.scaleX, 0));
                node.__sprite.scale.y = spriteBase.scaleY * (1 + breatheWave * numberValue(breathe.scaleY, 0));
              }
              if (blink.enabled) {
                node.alpha = blink.mode === 'show'
                  ? (blinkWindow ? base.alpha : numberValue(blink.restAlpha, 0))
                  : (blinkWindow ? numberValue(blink.alpha, 0.18) : base.alpha);
              } else if (mouth.mode === 'idle') {
                node.alpha = stateKey === 'working'
                  ? numberValue(mouth.workingAlpha, 0)
                  : base.alpha;
              } else if (mouth.mode === 'talk') {
                node.alpha = stateKey === 'working'
                  ? Math.max(numberValue(mouth.minAlpha, 0), mouthWave * base.alpha)
                  : numberValue(mouth.restAlpha, 0);
              } else {
                node.alpha = base.alpha;
              }
              if (layer.id?.startsWith('tail_')) {
                node.rotation += Math.sin(time * 1.8 + phase) * 0.018 * delta;
              }

              if (isActive && pet.lift > 0.01) {
                const id = (layer.id || '').replace('viewer_', '');
                if (pet.arm === 'left') {
                  if (id === 'left_upper_arm') {
                    node.rotation += -0.10 * pet.lift + Math.sin(time * 7) * 0.05 * pet.lift;
                  } else if (id === 'left_hand') {
                    node.rotation += Math.sin(time * 8) * 0.10 * pet.lift;
                  }
                } else if (id === 'right_upper_arm') {
                  node.rotation += -0.07 * pet.lift;
                } else if (id === 'right_forearm') {
                  node.rotation += -0.05 * pet.lift + Math.sin(time * 7) * 0.04 * pet.lift;
                } else if (id === 'right_hand') {
                  node.rotation += Math.sin(time * 8) * 0.08 * pet.lift;
                }
              }

              let tickleTarget = 0;
              let dirX = 0;
              let dirY = 0;
              const tickleWeight = tickleLayerWeight(layer.id);
              if (isActive && tickleOn && tmpPointRef && tickleWeight > 0 && pointer.poke > 0.02) {
                const gp = node.getGlobalPosition(tmpPointRef);
                const dx = gp.x - pointer.x;
                const dy = gp.y - pointer.y;
                const dist = Math.hypot(dx, dy) || 1;
                const proximity = clamp(1 - dist / TICKLE_REACH, 0, 1);
                tickleTarget = proximity * pointer.poke * tickleWeight * TICKLE_POKE_SCALE;
                dirX = dx / dist;
                dirY = dy / dist;
              }
              const prevTickle = node.__tickle || 0;
              const nextTickle = tickleTarget > prevTickle
                ? prevTickle + (tickleTarget - prevTickle) * 0.5
                : prevTickle * 0.84;
              node.__tickle = nextTickle < 0.001 ? 0 : nextTickle;
              if (node.__tickle > 0.001) {
                const power = Math.min(TICKLE_MAX_POWER, node.__tickle + pointer.poke * 0.18);
                const giggleW = speedNorm * power;
                const dodgeW = (1 - speedNorm) * power;
                node.x += dirX * 3.2 * dodgeW;
                node.y += dirY * 2.4 * dodgeW;
                node.rotation += (dirX >= 0 ? 1 : -1) * 0.055 * dodgeW;
                const jitter = time * 34 + phase;
                node.x += Math.sin(jitter) * 1.4 * giggleW;
                node.y += Math.cos(jitter * 0.92) * 1.1 * giggleW;
                node.scale.x *= 1 + Math.sin(jitter) * 0.02 * giggleW;
                node.scale.y *= 1 - Math.sin(jitter) * 0.02 * giggleW;
              }
            });
          });
        };
        app.ticker.add(tickerHandler);
        window.requestAnimationFrame(() => {
          if (cancelled) return;
          setReady(true);
          onReadyRef.current?.();
        });
      } catch (error) {
        console.warn('[PixiAvatarStage]', error);
        if (!cancelled) {
          setReady(false);
          setFailed(true);
        }
      }
    }

    const updateTicklePointer = (event) => {
      const rect = host.getBoundingClientRect();
      if (!rect || rect.width <= 0 || rect.height <= 0) return false;
      const x = event.clientX - rect.left;
      const y = event.clientY - rect.top;
      const now = performance.now();
      const p = pointerRef.current;
      const dt = Math.max(1, now - (p.lastT || now));
      const moved = Math.hypot(x - p.lastX, y - p.lastY);
      p.speed = p.speed * 0.6 + (moved / dt) * 0.4;
      const margin = 28;
      p.inside = x >= -margin && y >= -margin
        && x <= rect.width + margin && y <= rect.height + margin;
      p.x = x;
      p.y = y;
      p.lastX = x;
      p.lastY = y;
      p.lastT = now;
      return true;
    };

    const updateAttentionLock = (event) => {
      const rect = host.getBoundingClientRect();
      if (!rect || rect.width <= 0 || rect.height <= 0) return false;
      const now = performance.now();
      if (now < attention.cooldownUntil) return false;
      const centerX = rect.left + rect.width / 2;
      const halfWidth = ATTENTION_CENTER_WIDTH / 2;
      const side = event.clientX < centerX - halfWidth
        ? 'left'
        : event.clientX > centerX + halfWidth
          ? 'right'
          : null;
      if (!side) {
        clearAttentionTimer();
        attention.side = null;
        setAttentionPaused(false);
        return false;
      }
      if (attention.paused) return true;
      if (attention.side !== side) {
        clearAttentionTimer();
        attention.side = side;
          attention.timer = window.setTimeout(() => {
            attention.timer = null;
            setAttentionPaused(true, performance.now(), nextAttentionFinishBehavior());
        }, ATTENTION_TRIGGER_DELAY_MS);
      }
      return false;
    };

    const updateLook = (event) => {
      if (updateAttentionLock(event)) {
        targetLookRef.current = {x: 0, y: 0, turnX: 0};
        updateTicklePointer(event);
        return;
      }
      const prev = targetLookRef.current;
      const next = targetFromPoint(host, event.clientX, event.clientY);
      if (Math.abs(next.x - prev.x) + Math.abs(next.y - prev.y) > 0.012) {
        ctl.lastActive = performance.now();
      }
      targetLookRef.current = next;
      updateTicklePointer(event);
    };
    const settleLook = () => {
      if (attention.paused) {
        targetLookRef.current = {x: 0, y: 0, turnX: 0};
        return;
      }
      targetLookRef.current = {
        x: targetLookRef.current.x * 0.42,
        y: targetLookRef.current.y * 0.42,
        turnX: (targetLookRef.current.turnX || 0) * 0.42,
      };
    };
    const pokeDown = (event) => {
      if (!tickleModeRef.current || event.button !== 0) return;
      updateTicklePointer(event);
      if (pointerRef.current.inside) pointerRef.current.poke = 1;
      ctl.lastActive = performance.now();
    };
    window.addEventListener('pointermove', updateLook);
    window.addEventListener('pointerup', settleLook);
    window.addEventListener('pointerdown', pokeDown);
    start();

    return () => {
      cancelled = true;
      window.removeEventListener('pointermove', updateLook);
      window.removeEventListener('pointerup', settleLook);
      window.removeEventListener('pointerdown', pokeDown);
      clearAttentionTimer();
      setAttentionPaused(false);
      resizeObserver?.disconnect();
      try {
        if (tickerHandler) app?.ticker?.remove?.(tickerHandler);
        app?.ticker?.stop?.();
      } catch (error) {
        console.warn('[PixiAvatarStage] ticker cleanup failed', error);
      }
      if (app && initialized) {
        try {
          app.destroy(true, {children: true, texture: false, textureSource: false});
        } catch (error) {
          console.warn('[PixiAvatarStage] destroy failed', error);
        }
      }
      app = null;
      initialized = false;
      try {
        if (host.isConnected) host.replaceChildren();
      } catch (error) {
        console.warn('[PixiAvatarStage] host cleanup failed', error);
      }
      restorePixiWarnFilter();
    };
  }, [
    alt,
    fallbackSrc,
    height,
    manifestKey,
    stateKey,
    width,
  ]);

  if (failed) {
    const fallbackImageSrc = manifest?.baseSrc || fallbackSrc;
    return (
      <img
        className="floating-avatar-img pixi-avatar-fallback pixi-avatar-fallback-animated"
        src={fallbackImageSrc}
        alt={alt}
        draggable={false}
        onError={(event) => {
          if (fallbackSrc && event.currentTarget.src !== fallbackSrc) {
            event.currentTarget.src = fallbackSrc;
          }
        }}
      />
    );
  }

  return (
    <span
      ref={hostRef}
      className="pixi-avatar-stage"
      aria-hidden={alt ? undefined : true}
      aria-label={alt || undefined}
      data-avatar-state={stateKey}
      data-avatar-motion={manifest?.id || 'procedural'}
      data-avatar-tickle={tickle ? 'on' : undefined}
      data-avatar-ready={ready ? 'true' : 'false'}
    />
  );
}
