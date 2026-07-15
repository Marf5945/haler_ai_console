import React, {useEffect, useMemo, useRef, useState} from 'react';
import {InochiAvatar} from 'inochi-avatar';
import {createInochiParamDriver} from '../inochiParamDriver.js';

// WKWebView（macOS 透明浮窗）合成器問題：WebGL canvas 用預設 premultipliedAlpha
// 會被整片吃掉（黑塊或直接消失）——Pixi 線上同樣的病、同樣的藥。
// inox2d 在 wasm 內部自建 context，管不到參數，只好攔截 getContext 代為注入。
const IS_WEBKIT_ENGINE = typeof navigator !== 'undefined'
  && /AppleWebKit/i.test(navigator.userAgent || '')
  && !/Chrome|Chromium|Edg\//i.test(navigator.userAgent || '');

if (typeof HTMLCanvasElement !== 'undefined' && IS_WEBKIT_ENGINE
  && !window.__inochiCtxPatched) {
  window.__inochiCtxPatched = true;
  const originalGetContext = HTMLCanvasElement.prototype.getContext;
  HTMLCanvasElement.prototype.getContext = function patchedGetContext(type, attrs) {
    if ((type === 'webgl2' || type === 'webgl')
      && this.classList?.contains('floating-avatar-inochi-canvas')) {
      return originalGetContext.call(this, type, {
        ...(attrs || {}),
        alpha: true,
        premultipliedAlpha: false,
        antialias: false,
      });
    }
    return originalGetContext.call(this, type, attrs);
  };
}

function numberValue(value, fallback) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function anchorFromPointer(element, clientX) {
  const rect = element?.getBoundingClientRect?.();
  if (!rect || rect.width <= 0) return 'front_000';
  const dx = clientX - (rect.left + rect.width / 2);
  const threshold = Math.max(28, rect.width * 0.18);
  if (dx < -threshold) return 'left_030';
  if (dx > threshold) return 'right_030';
  return 'front_000';
}

function normalizedPointer(element, clientX, clientY) {
  const rect = element?.getBoundingClientRect?.();
  if (!rect || rect.width <= 0 || rect.height <= 0) return {x: 0, y: 0};
  const x = ((clientX - rect.left) / rect.width - 0.5) * 2;
  const y = ((clientY - rect.top) / rect.height - 0.5) * 2;
  return {
    x: Math.min(1, Math.max(-1, x)),
    y: Math.min(1, Math.max(-1, y)),
  };
}

export function transitionFrames(config, from, to) {
  if (from === to) return [];
  const key = `${from}_to_${to}`;
  const direct = config?.transitions?.[key];
  if (direct?.length) return direct;
  if (to === 'front_000') {
    const reverse = config?.transitions?.[`front_000_to_${from}`] || [];
    return [...reverse].reverse();
  }
  if (from !== 'front_000' && to !== 'front_000') {
    return [
      ...transitionFrames(config, from, 'front_000'),
      ...transitionFrames(config, 'front_000', to),
    ];
  }
  return [];
}

function transitionSource(frame) {
  return typeof frame === 'string' ? frame : frame?.src || '';
}

function transitionStyle(config, frame) {
  if (!frame || typeof frame === 'string' || !Array.isArray(frame.hip)) return undefined;
  const [canvasWidth, canvasHeight] = config?.transitionCanvas || [200, 360];
  const [targetX, targetY] = config?.transitionHip || [100, 200];
  const sourceX = numberValue(frame.hip[0], targetX);
  const sourceY = numberValue(frame.hip[1], targetY);
  const dx = numberValue(targetX, 100) - sourceX;
  const dy = numberValue(targetY, 200) - sourceY;
  const targetColoredHeight = Math.max(1, numberValue(config?.targetColoredHeight, 352));
  const sourceColoredHeight = Math.max(1, numberValue(frame.coloredHeight, targetColoredHeight));
  const scale = targetColoredHeight / sourceColoredHeight;
  return {
    // Scale the actual painted silhouette uniformly around the anatomical hip,
    // then move that hip to the shared stage anchor. Transparent canvas size
    // never participates in the character proportion calculation.
    transform: `translate(${dx / canvasWidth * 100}%, ${dy / canvasHeight * 100}%) scale(${scale})`,
    transformOrigin: `${sourceX / canvasWidth * 100}% ${sourceY / canvasHeight * 100}%`,
  };
}

export default function InochiAvatarStage({
  config,
  fallbackSrc = '',
  width = 200,
  height = 360,
  expression = '',
  tickle = false,
  motionMode = 'rig',
  frameSequences = null,
  onReady,
  onPetComplete,
}) {
  const hostRef = useRef(null);
  const onReadyRef = useRef(onReady);
  const onPetCompleteRef = useRef(onPetComplete);
  const avatarsRef = useRef(new Map());
  const driversRef = useRef(new Map());
  const expressionRef = useRef(expression);
  const activeAnchorRef = useRef('front_000');
  const transitionTimerRef = useRef(null);
  const frameTimersRef = useRef([]);
  const petTimerRef = useRef(null);
  const randomTimerRef = useRef(null);
  const returnTimerRef = useRef(null);
  // 「PNG 逐格」動作模式用：目前逐格模式、可用序列、播放器狀態、當前顯示影格
  const motionModeRef = useRef(motionMode);
  const frameSeqRef = useRef(frameSequences);
  const seqPlayerRef = useRef(null);
  // 注意力狀態：engaged=會跟著滑鼠轉身；散掉後只有「轉到另一邊」或點擊才會回神
  const attentionRef = useRef({engaged: true, lastSide: null});
  const interactionRef = useRef({
    lastX: 0,
    lastY: 0,
    lastAt: 0,
    petZone: false,
    petTriggered: false,
  });
  const canvasIds = useMemo(() => {
    const prefix = `inochi-${Math.random().toString(36).slice(2)}`;
    return {
      front_000: `${prefix}-front-000`,
      left_030: `${prefix}-left-030`,
      right_030: `${prefix}-right-030`,
    };
  }, []);
  const [ready, setReady] = useState(false);
  const [activeAnchor, setActiveAnchor] = useState('front_000');
  const [transitionFrame, setTransitionFrame] = useState({current: null, key: 0});
  const [seqFrame, setSeqFrame] = useState(null);
  const [loadError, setLoadError] = useState(null);
  const anchors = config?.anchors || {};
  const runtimeUrl = config?.runtimeUrl || '/inochi2d-runtime/inochi_fox_demo.js';
  const cameraZoom = numberValue(config?.camera?.zoom, 0.006);
  const cameraX = numberValue(config?.camera?.x, 0);
  const cameraY = numberValue(config?.camera?.y, 0);
  const transitionDurationMs = Math.max(300, numberValue(config?.transitionDurationMs, 1500));
  const attentionCfg = config?.attention || {};
  const returnDelayMs = Math.max(1000, numberValue(attentionCfg.returnDelayMs, 8000));
  const returnDurationMs = Math.max(300, numberValue(attentionCfg.returnDurationMs, 3000));
  const clickReaction = attentionCfg.clickReaction || {};
  const maxTurnAngle = Math.max(
    1,
    ...Object.values(anchors).map((anchor) => Math.abs(numberValue(anchor?.angleDeg, 0))),
  );
  const randomAnimationKey = JSON.stringify({
    choices: config?.randomAnimations || [],
    interval: config?.randomAnimationIntervalMs || [],
    allowed: config?.randomAnimationAllowedExpressions || [],
  });
  const anchorLoadKey = useMemo(() => JSON.stringify({
    runtimeUrl,
    cameraZoom,
    cameraX,
    cameraY,
    paramMotion: config?.paramMotion || null,
    paramChannels: config?.paramChannels || null,
    expressions: config?.expressions || null,
    randomAnimationMotions: config?.randomAnimationMotions || null,
    anchors: Object.entries(anchors).map(([name, anchor]) => [
      name,
      anchor?.modelSrc || '',
      anchor?.mode || 'idle',
      anchor?.mouseTracking !== false,
      anchor?.camera || null,
    ]),
  }), [anchors, cameraX, cameraY, cameraZoom, config, runtimeUrl]);

  useEffect(() => {
    onReadyRef.current = onReady;
  }, [onReady]);

  useEffect(() => {
    onPetCompleteRef.current = onPetComplete;
  }, [onPetComplete]);

  // 逐格播放器：停止當前序列並清掉顯示影格
  function stopFrameSequence() {
    const player = seqPlayerRef.current;
    if (player?.timer) window.clearTimeout(player.timer);
    if (player?.endTimer) window.clearTimeout(player.endTimer);
    seqPlayerRef.current = null;
    setSeqFrame(null);
  }

  // 逐格播放器：依 fps 逐格播 frameSequences[id]。durationMs>0 時到時自動停。
  // 只在 motionMode==='frames' 生效；成功接手回傳 true（呼叫端可略過 puppet 擺動）。
  function playFrameSequence(id, durationMs = 0) {
    if (motionModeRef.current !== 'frames') return false;
    const seq = frameSeqRef.current?.[id];
    if (!seq?.frames?.length) return false;
    stopFrameSequence();
    const frames = seq.frames;
    const interval = 1000 / (Number(seq.fps) > 0 ? Number(seq.fps) : 8);
    const loop = seq.loop !== false;
    const player = {timer: null, endTimer: null};
    seqPlayerRef.current = player;
    let idx = 0;
    const tick = () => {
      setSeqFrame(frames[idx]);
      idx += 1;
      if (idx >= frames.length) {
        if (!loop) return; // 非循環：停在最後一格
        idx = 0;
      }
      player.timer = window.setTimeout(tick, interval);
    };
    tick();
    if (durationMs > 0) {
      player.endTimer = window.setTimeout(stopFrameSequence, durationMs);
    }
    return true;
  }

  useEffect(() => {
    frameSeqRef.current = frameSequences;
  }, [frameSequences]);

  // 切換動作模式：進 frames 且目前表情有對應序列就開播；回 rig 立刻收掉疊層
  useEffect(() => {
    motionModeRef.current = motionMode;
    if (motionMode === 'frames') {
      if (frameSeqRef.current?.[expressionRef.current]) {
        playFrameSequence(expressionRef.current, 0);
      }
    } else {
      stopFrameSequence();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [motionMode]);

  useEffect(() => {
    expressionRef.current = expression;
    driversRef.current.forEach((driver) => driver.setExpression(expression));
    // 逐格模式下，表情若有對應逐格序列（如 working_reaction）就循環播到表情改變為止
    if (motionModeRef.current === 'frames') {
      if (frameSeqRef.current?.[expression]) {
        playFrameSequence(expression, 0);
      } else {
        stopFrameSequence();
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expression]);

  useEffect(() => {
    let cancelled = false;
    let loadedAny = false;
    const loadedAvatars = new Map();
    const loadedDrivers = new Map();
    setLoadError(null);

    async function loadAnchors() {
      try {
        const entries = Object.entries(anchors).filter(([, anchor]) => anchor?.modelSrc);
        if (!entries.length) {
          throw new Error('No Inochi2D .inp model URLs were resolved.');
        }
        for (const [anchorName, anchor] of entries) {
          const canvasId = canvasIds[anchorName];
          if (!canvasId) continue;
          const avatar = new InochiAvatar(canvasId, anchor.modelSrc, {
            autoPlay: false,
            defaultMode: anchor.mode || 'idle',
            enableMouseTracking: anchor.mouseTracking !== false,
          });
          await avatar.load(runtimeUrl);
          if (cancelled) {
            avatar.stop();
            return;
          }
          avatar.setCameraZoom(numberValue(anchor.camera?.zoom, cameraZoom));
          avatar.setCameraPosition(
            numberValue(anchor.camera?.x, cameraX),
            numberValue(anchor.camera?.y, cameraY),
          );
          let paramNames = [];
          try {
            paramNames = avatar.model?.get_parameter_names?.() || [];
          } catch (error) {
            paramNames = [];
          }
          if (!paramNames.length && avatar.model) {
            // 這顆 demo wasm 的 get_parameter_names 有 bug：參數明明存在
            // （set_parameter 對不存在的名字會丟 NoParameterNamed，存在則成功），
            // 清單卻永遠回空。改用 set_parameter 反向探測候選參數名。
            const candidates = new Set([
              'Blink', 'HeadYaw', 'HeadNod', 'EyeTracking', 'MouthOpen',
              'Breath', 'BodySway', 'TailSway', 'ArmNearSway', 'EarWiggle',
              'WalkPhase', 'ArmRaise', 'PetResponse', 'Tickle',
              ...Object.keys(config?.paramMotion || {}),
              ...Object.values(config?.paramChannels || {}),
              ...Object.values(config?.expressions || {})
                .flatMap((spec) => Object.keys(spec?.params || {})),
            ]);
            try { avatar.model.begin_frame?.(); } catch (_) { /* 容錯 */ }
            for (const name of candidates) {
              if (typeof name !== 'string' || !name) continue;
              try {
                avatar.model.set_parameter(name, 0);
                paramNames.push(name);
              } catch (_) {
                try {
                  avatar.model.set_parameter_2d(name, 0, 0);
                  paramNames.push(name);
                } catch (__) { /* 此參數確實不存在，略過 */ }
              }
            }
            try { avatar.model.end_frame?.(0); } catch (_) { /* 容錯 */ }
            if (paramNames.length) {
              console.info(
                `[InochiAvatarStage] ${anchorName} get_parameter_names 回空（runtime bug），探測到參數:`,
                paramNames,
              );
            }
          }
          const driver = createInochiParamDriver({
            paramNames,
            paramMotion: config?.paramMotion,
            channels: config?.paramChannels,
            expressions: config?.expressions,
            ambientMotions: config?.randomAnimationMotions,
          });
          if (driver.active) {
            avatar.onCustomAnimate = (deltaTime) => driver.update(avatar, deltaTime);
            driver.setExpression(expressionRef.current);
            avatar.play('custom');
            console.info(
              `[InochiAvatarStage] ${anchorName} puppet params:`,
              paramNames,
              '| driving:',
              driver.matched,
            );
          } else {
            avatar.play(anchor.mode || 'idle');
            console.warn(
              `[InochiAvatarStage] ${anchorName} has no driveable params, falling back to '${anchor.mode || 'idle'}' mode.`,
              paramNames,
            );
          }
          loadedAvatars.set(anchorName, avatar);
          loadedDrivers.set(anchorName, driver);
          loadedAny = true;
        }
        if (!cancelled && loadedAny) {
          avatarsRef.current = loadedAvatars;
          driversRef.current = loadedDrivers;
          setReady(true);
          onReadyRef.current?.();
        }
      } catch (error) {
        if (!cancelled) {
          console.warn('[InochiAvatarStage] failed to load Inochi2D puppet', error);
          setLoadError(error);
        }
      }
    }

    loadAnchors();
    return () => {
      cancelled = true;
      if (transitionTimerRef.current) {
        window.clearTimeout(transitionTimerRef.current);
        transitionTimerRef.current = null;
      }
      frameTimersRef.current.forEach((timer) => window.clearTimeout(timer));
      frameTimersRef.current = [];
      stopFrameSequence();
      if (petTimerRef.current) {
        window.clearTimeout(petTimerRef.current);
        petTimerRef.current = null;
      }
      if (returnTimerRef.current) {
        window.clearTimeout(returnTimerRef.current);
        returnTimerRef.current = null;
      }
      loadedAvatars.forEach((avatar) => avatar.stop());
      avatarsRef.current.forEach((avatar) => avatar.stop());
      avatarsRef.current = new Map();
      driversRef.current = new Map();
    };
  }, [anchorLoadKey, cameraX, cameraY, cameraZoom, canvasIds, runtimeUrl]);

  useEffect(() => {
    const choices = Array.isArray(config?.randomAnimations)
      ? config.randomAnimations.filter((item) => item?.id && numberValue(item?.durationMs, 0) > 0)
      : [];
    if (!ready || !choices.length) return undefined;

    let cancelled = false;
    const interval = Array.isArray(config?.randomAnimationIntervalMs)
      ? config.randomAnimationIntervalMs
      : [22000, 28000];
    const allowedExpressions = new Set(
      Array.isArray(config?.randomAnimationAllowedExpressions)
        ? config.randomAnimationAllowedExpressions
        : ['idle'],
    );
    const RETRY_MS = 800;
    const randomDelay = () => {
      const lo = Math.max(1000, numberValue(interval[0], 20000));
      const hi = Math.max(lo, numberValue(interval[1], 30000));
      return lo + Math.random() * (hi - lo);
    };
    const scheduleAt = (delayMs) => {
      if (cancelled) return;
      if (randomTimerRef.current) window.clearTimeout(randomTimerRef.current);
      randomTimerRef.current = window.setTimeout(runRandomAnimation, delayMs);
    };
    const runRandomAnimation = () => {
      randomTimerRef.current = null;
      const currentExpression = expressionRef.current || 'idle';
      if (transitionTimerRef.current || interactionRef.current.petZone
        || !allowedExpressions.has(currentExpression)) {
        // 轉向/互動優先。時間已到但被擋 -> 短間隔重試，
        // 不重抽整段 20~30 秒（轉向不會把計時歸零）。
        scheduleAt(RETRY_MS);
        return;
      }
      const choice = choices[Math.floor(Math.random() * choices.length)] || choices[0];
      const durationMs = Math.max(250, numberValue(choice.durationMs, 2500));
      let triggeredCount = 0;
      const framePlayed = motionModeRef.current === 'frames'
        && playFrameSequence(choice.id, durationMs);
      if (!framePlayed) {
        driversRef.current.forEach((driver) => {
          if (driver.triggerAmbient?.(choice.id, durationMs / 1000)) triggeredCount += 1;
        });
      }
      console.info(
        `[InochiAvatarStage] random animation '${choice.id}' triggered on ${triggeredCount}/${driversRef.current.size} driver(s) for ${durationMs}ms`,
      );
      scheduleAt(durationMs + randomDelay());
    };

    scheduleAt(randomDelay());
    return () => {
      cancelled = true;
      if (randomTimerRef.current) {
        window.clearTimeout(randomTimerRef.current);
        randomTimerRef.current = null;
      }
      driversRef.current.forEach((driver) => driver.cancelAmbient?.());
    };
  }, [randomAnimationKey, ready]);

  const switchAnchor = (nextAnchor, durationOverrideMs) => {
    if (!avatarsRef.current.has(nextAnchor)) nextAnchor = 'front_000';
    if (!avatarsRef.current.has(nextAnchor)) return;
    const current = activeAnchorRef.current;
    if (current === nextAnchor) return;
    // 轉向動畫優先：正在播的隨機動圖立刻讓路
    driversRef.current.forEach((driver) => driver.cancelAmbient?.());
    const duration = Math.max(300, numberValue(durationOverrideMs, transitionDurationMs));
    const frames = transitionFrames(config, current, nextAnchor);
    activeAnchorRef.current = nextAnchor;
    if (transitionTimerRef.current) window.clearTimeout(transitionTimerRef.current);
    if (!frames.length) {
      setTransitionFrame({current: null, key: 0});
      setActiveAnchor(nextAnchor);
      return;
    }
    const showFrame = (frame) => setTransitionFrame((previous) => ({
      current: frame,
      key: previous.key + 1,
    }));
    showFrame(frames[0]);
    const frameDuration = duration / frames.length;
    frameTimersRef.current.forEach((timer) => window.clearTimeout(timer));
    frameTimersRef.current = [];
    frames.slice(1).forEach((frame, index) => {
      const timer = window.setTimeout(
        () => showFrame(frame), frameDuration * (index + 1),
      );
      frameTimersRef.current.push(timer);
    });
    transitionTimerRef.current = window.setTimeout(() => {
      setActiveAnchor(nextAnchor);
      setTransitionFrame({current: null, key: 0});
      transitionTimerRef.current = null;
    }, duration);
  };

  // 轉向 8 秒後自動回正；回正時長依角度佔比縮放（30 度 = 滿 3 秒）
  const returnToFront = () => {
    returnTimerRef.current = null;
    const current = activeAnchorRef.current;
    if (current === 'front_000') return;
    const angle = Math.abs(numberValue(anchors[current]?.angleDeg, maxTurnAngle));
    const duration = Math.max(300, returnDurationMs * Math.min(1, angle / maxTurnAngle));
    attentionRef.current.engaged = false;
    attentionRef.current.lastSide = current;
    console.info(
      `[InochiAvatarStage] attention timeout, returning to front over ${Math.round(duration)}ms (angle=${angle})`,
    );
    switchAnchor('front_000', duration);
  };

  const turnTo = (nextAnchor) => {
    if (returnTimerRef.current) {
      window.clearTimeout(returnTimerRef.current);
      returnTimerRef.current = null;
    }
    switchAnchor(nextAnchor);
    if (nextAnchor !== 'front_000') {
      returnTimerRef.current = window.setTimeout(returnToFront, returnDelayMs);
    }
  };

  const handlePointerMove = (event) => {
    const desired = anchorFromPointer(hostRef.current, event.clientX);
    const att = attentionRef.current;
    if (desired !== activeAnchorRef.current) {
      if (att.engaged) {
        // 注意力在線：跟著滑鼠轉。同一邊內移動不會走到這裡，
        // 8 秒回正計時只在「真的換了方向」時重算。
        turnTo(desired);
      } else {
        // 注意力已散：只有滑鼠跑到「另一邊」才重新理你
        const opposite = att.lastSide === 'left_030' ? 'right_030'
          : att.lastSide === 'right_030' ? 'left_030' : null;
        if (desired === opposite) {
          att.engaged = true;
          turnTo(desired);
        }
      }
    }
    const pointer = normalizedPointer(hostRef.current, event.clientX, event.clientY);
    driversRef.current.forEach((driver) => driver.setPointer(pointer.x, pointer.y));

    const now = performance.now();
    const interaction = interactionRef.current;
    const elapsed = interaction.lastAt ? Math.max(1, now - interaction.lastAt) : 0;
    const distance = Math.hypot(
      event.clientX - interaction.lastX,
      event.clientY - interaction.lastY,
    );
    const speed = elapsed > 0 && elapsed < 240 ? distance / elapsed : 0;
    const petZone = Math.abs(pointer.x) < 0.78 && pointer.y > -0.92 && pointer.y < -0.08;
    const slowPet = petZone && speed < 0.34;
    const tickleValue = tickle && speed > 0.62
      ? (event.clientX >= interaction.lastX ? 1 : -1)
      : 0;

    interaction.lastX = event.clientX;
    interaction.lastY = event.clientY;
    interaction.lastAt = now;
    interaction.petZone = slowPet;

    if (slowPet) {
      driversRef.current.forEach((driver) => driver.setInteraction({pet: 0.38, tickle: 0}));
      if (!petTimerRef.current) {
        petTimerRef.current = window.setTimeout(() => {
          petTimerRef.current = null;
          if (!interactionRef.current.petZone) return;
          driversRef.current.forEach((driver) => driver.setInteraction({pet: 1, tickle: 0}));
          if (!interactionRef.current.petTriggered) {
            interactionRef.current.petTriggered = true;
            onPetCompleteRef.current?.();
          }
        }, 620);
      }
    } else {
      if (petTimerRef.current) {
        window.clearTimeout(petTimerRef.current);
        petTimerRef.current = null;
      }
      if (!petZone) interaction.petTriggered = false;
      driversRef.current.forEach((driver) => driver.setInteraction({pet: 0, tickle: tickleValue}));
    }
  };

  const handlePointerDown = (event) => {
    const att = attentionRef.current;
    att.engaged = true;
    // 點擊把關注力抓回來：立即轉向鼠標方向，並重啟 8 秒回正計時
    turnTo(anchorFromPointer(hostRef.current, event.clientX));
    const pointer = normalizedPointer(hostRef.current, event.clientX, event.clientY);
    // 越靠近角色（畫面中線）反應越大
    const proximity = 1 - Math.min(1, Math.abs(pointer.x));
    const minIntensity = numberValue(clickReaction.minIntensity, 0.8);
    const maxIntensity = Math.max(minIntensity, numberValue(clickReaction.maxIntensity, 1.6));
    const intensity = minIntensity + (maxIntensity - minIntensity) * proximity;
    const reactionId = clickReaction.id || 'working_reaction';
    const durationMs = Math.max(250, numberValue(clickReaction.durationMs, 2500));
    const framePlayed = motionModeRef.current === 'frames'
      && playFrameSequence(reactionId, durationMs);
    if (!framePlayed) {
      driversRef.current.forEach((driver) => {
        driver.triggerAmbient?.(reactionId, durationMs / 1000, intensity);
      });
    }
    console.info(
      `[InochiAvatarStage] click reaction '${reactionId}' intensity=${intensity.toFixed(2)} (proximity=${proximity.toFixed(2)})`,
    );
  };

  const handlePointerLeave = () => {
    if (petTimerRef.current) {
      window.clearTimeout(petTimerRef.current);
      petTimerRef.current = null;
    }
    interactionRef.current = {
      lastX: 0,
      lastY: 0,
      lastAt: 0,
      petZone: false,
      petTriggered: false,
    };
    driversRef.current.forEach((driver) => {
      driver.setPointer(0, 0);
      driver.setInteraction({pet: 0, tickle: 0});
    });
  };

  if (!Object.values(anchors).some((anchor) => anchor?.modelSrc)) {
    throw new Error('Inochi2D renderer selected but no .inp models were found.');
  }
  if (loadError) throw loadError;

  return (
    <span
      ref={hostRef}
      className="floating-avatar-inochi-stage"
      style={{width, height}}
      onPointerMove={handlePointerMove}
      onPointerDown={handlePointerDown}
      onPointerLeave={handlePointerLeave}
      aria-hidden="true"
    >
      {!ready && fallbackSrc && (
        <img className="floating-avatar-img floating-avatar-pixi-backstop" src={fallbackSrc} alt="" draggable={false} />
      )}
      {Object.keys(canvasIds).map((anchorName) => (
        <canvas
          key={anchorName}
          id={canvasIds[anchorName]}
          className="floating-avatar-inochi-canvas"
          width={width}
          height={height}
          style={{
            ...transitionStyle(config, anchors[anchorName]),
            opacity: ready && !transitionFrame.current && !seqFrame && activeAnchor === anchorName ? 1 : 0,
          }}
        />
      ))}
      {transitionFrame.current && (
        <img
          key={transitionFrame.key}
          className="floating-avatar-inochi-transition floating-avatar-inochi-transition-current"
          src={transitionSource(transitionFrame.current)}
          style={transitionStyle(config, transitionFrame.current)}
          alt=""
          draggable={false}
        />
      )}
      {seqFrame && (
        <img
          className="floating-avatar-inochi-transition floating-avatar-inochi-sequence"
          src={seqFrame}
          style={{position: 'absolute', left: 0, top: 0, width, height, objectFit: 'contain'}}
          alt=""
          draggable={false}
        />
      )}
    </span>
  );
}
