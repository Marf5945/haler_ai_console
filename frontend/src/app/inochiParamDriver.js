/**
 * inochiParamDriver.js
 * Semantic idle/expression driver for Inochi2D puppets built by the
 * split-and-pack pipeline (wolfdog rig naming: HeadYaw, TailSway, ...).
 *
 * Unlike the fox-demo naming baked into the inochi-avatar package's
 * idle mode, this driver reads the puppet's real parameter list and
 * only drives parameters that actually exist. Configure per persona in
 * manifest.json under inochi2d.paramMotion / paramChannels / expressions.
 */

function lerp(a, b, t) {
  return a + (b - a) * t;
}

function clamp(v, lo, hi) {
  return Math.min(hi, Math.max(lo, v));
}

function randomIn(range, fallbackLo = 3, fallbackHi = 8) {
  const [lo, hi] = Array.isArray(range) && range.length === 2
    ? range
    : [fallbackLo, fallbackHi];
  return lo + Math.random() * Math.max(0, hi - lo);
}

// Semantic channels -> actual parameter names. Override via
// manifest inochi2d.paramChannels when a rig uses different names.
const DEFAULT_CHANNELS = {
  blink: 'Blink',
  headYaw: 'HeadYaw',
  headNod: 'HeadNod',
  eyeTracking: 'EyeTracking',
  mouthOpen: 'MouthOpen',
  walkPhase: 'WalkPhase',
  armRaise: 'ArmRaise',
  petResponse: 'PetResponse',
  tickle: 'Tickle',
};

// Ambient waves for the wolfdog rig. Only applied when the parameter
// exists on the loaded puppet. Override/extend via inochi2d.paramMotion.
const DEFAULT_PARAM_MOTION = {
  Breath: {wave: 'sine', speed: 1.6, min: 0, max: 1},
  BodySway: {wave: 'sine', speed: 0.5, min: -0.35, max: 0.35},
  TailSway: {wave: 'sine', speed: 2.4, min: -0.6, max: 0.6},
  ArmNearSway: {wave: 'sine', speed: 0.7, min: -0.25, max: 0.25},
  HandSway: {wave: 'sine', speed: 0.8, min: -0.35, max: 0.35},
  LegNearSway: {wave: 'sine', speed: 1.05, min: -0.45, max: 0.45},
  LegFarSway: {wave: 'sine', speed: 1.05, phase: Math.PI, min: -0.45, max: 0.45},
  HeadTilt: {wave: 'sine', speed: 0.35, min: -0.18, max: 0.18},
  EarWiggle: {wave: 'twitch', interval: [4, 9], duration: 0.55, cycles: 8, min: -1, max: 1},
  EarWiggleOther: {wave: 'twitch', interval: [4, 9], duration: 0.5, cycles: 8, min: -1, max: 1},
};

const DEFAULT_WALK_MOTION = {
  Breath: {wave: 'sine', speed: 2.6, min: 0, max: 1},
  BodySway: {wave: 'sine', speed: 1.35, min: -0.5, max: 0.5},
  LegNearSway: {wave: 'sine', speed: 3.2, min: -0.9, max: 0.9},
  LegFarSway: {wave: 'sine', speed: 3.2, phase: Math.PI, min: -0.9, max: 0.9},
  ArmNearSway: {wave: 'sine', speed: 3.2, phase: Math.PI, min: -0.55, max: 0.55},
  HandSway: {wave: 'sine', speed: 3.2, min: -0.5, max: 0.5},
  TailSway: {wave: 'sine', speed: 2.8, min: -0.75, max: 0.75},
};

function mergeMotion(base, override, weight) {
  if (!override) return base;
  const out = {...(base || {}), ...override};
  if (base) {
    for (const key of ['speed', 'min', 'max']) {
      const a = Number(base[key]);
      const b = Number(override[key]);
      if (Number.isFinite(a) && Number.isFinite(b)) out[key] = lerp(a, b, weight);
    }
  }
  return out;
}

export function createInochiParamDriver({
  paramNames = [],
  paramMotion = {},
  channels = {},
  expressions = {},
  ambientMotions = {},
} = {}) {
  const paramSet = new Set(paramNames);
  const motionTable = {...DEFAULT_PARAM_MOTION, ...(paramMotion || {})};
  const chan = {...DEFAULT_CHANNELS, ...(channels || {})};

  const drivenWaves = Object.keys(motionTable).filter((name) => paramSet.has(name));
  // Every param mentioned by any expression gets a neutral baseline (0),
  // so dropping an expression eases the param back instead of freezing it.
  const expressionParams = new Set(
    Object.values(expressions || {}).flatMap((spec) => Object.keys(spec?.params || {})),
  );
  const hasHead = paramSet.has(chan.headYaw) || paramSet.has(chan.headNod);
  const hasBlink = paramSet.has(chan.blink);
  const hasEyes = paramSet.has(chan.eyeTracking);
  const hasWalk = paramSet.has(chan.walkPhase);
  const hasArmRaise = paramSet.has(chan.armRaise);
  const hasPetResponse = paramSet.has(chan.petResponse);
  const hasTickle = paramSet.has(chan.tickle);
  const active = drivenWaves.length > 0 || hasHead || hasBlink || hasEyes
    || hasWalk || hasArmRaise || hasPetResponse || hasTickle;

  const state = {
    time: 0,
    pointer: {x: 0, y: 0},
    headYaw: 0,
    headNod: 0,
    eyeX: 0,
    eyeY: 0,
    blink: 0,
    blinkTimer: 0,
    nextBlink: 2 + Math.random() * 3,
    twitch: {},
    smooth: {},
    interaction: {pet: 0, tickle: 0},
    walk: {
      phase: 0,
    },
    ambient: {name: '', until: 0, intensity: 1},
    expression: {name: '', params: {}, motion: {}, weight: 1},
  };

  function setExpression(name) {
    const key = typeof name === 'string' ? name : '';
    if (key === state.expression.name) return;
    const spec = (expressions && expressions[key]) || {};
    state.expression = {
      name: key,
      params: spec.params || {},
      motion: spec.motion || {},
      weight: 0,
    };
  }

  function setPointer(x, y) {
    state.pointer.x = clamp(Number(x) || 0, -1, 1);
    state.pointer.y = clamp(Number(y) || 0, -1, 1);
  }

  function setInteraction({pet = 0, tickle = 0} = {}) {
    state.interaction.pet = clamp(Number(pet) || 0, 0, 1);
    state.interaction.tickle = clamp(Number(tickle) || 0, -1, 1);
  }

  function triggerAmbient(name, durationSeconds = 0, intensity = 1) {
    const key = typeof name === 'string' ? name : '';
    const duration = Math.max(0, Number(durationSeconds) || 0);
    if (!key || duration <= 0) return false;
    const level = clamp(Number(intensity) || 1, 0.2, 3);
    state.ambient = {name: key, until: state.time + duration, intensity: level};
    if (key === 'walk') state.walk.phase = 0;
    return true;
  }

  function cancelAmbient() {
    state.ambient = {name: '', until: 0, intensity: 1};
    state.walk.phase = 0;
  }

  // 依 intensity 圍繞中點放大/縮小波形振幅
  function scaleMotionAmp(cfg, level) {
    if (!cfg || !Number.isFinite(level) || level === 1) return cfg;
    const min = Number.isFinite(cfg.min) ? cfg.min : -1;
    const max = Number.isFinite(cfg.max) ? cfg.max : 1;
    const mid = (min + max) / 2;
    const amp = (max - min) / 2;
    return {...cfg, min: mid - amp * level, max: mid + amp * level};
  }

  function waveValue(name, cfg, dt) {
    const min = Number.isFinite(cfg.min) ? cfg.min : -1;
    const max = Number.isFinite(cfg.max) ? cfg.max : 1;
    if (cfg.wave === 'twitch') {
      const tw = state.twitch[name] || (state.twitch[name] = {
        timer: 0,
        next: randomIn(cfg.interval),
        progress: -1,
      });
      const duration = Number(cfg.duration) || 0.5;
      if (tw.progress >= 0) {
        tw.progress += dt / duration;
        if (tw.progress < 1) {
          const p = tw.progress;
          // cycles 可由 paramMotion 覆寫：值越大左右擺越快（預設 6 = 約 3 次來回）。
          const cycles = Number.isFinite(cfg.cycles) ? cfg.cycles : 6;
          const wiggle = Math.sin(Math.PI * p) * Math.sin(p * Math.PI * cycles);
          return clamp(max * wiggle, min, max);
        }
        tw.progress = -1;
        tw.timer = 0;
        tw.next = randomIn(cfg.interval);
      } else {
        tw.timer += dt;
        if (tw.timer >= tw.next) tw.progress = 0;
      }
      return 0;
    }
    const speed = Number.isFinite(cfg.speed) ? cfg.speed : 1;
    const phase = Number(cfg.phase) || 0;
    const mid = (min + max) / 2;
    const amp = (max - min) / 2;
    return mid + amp * Math.sin(state.time * speed + phase);
  }

  function update(avatar, dt) {
    if (!avatar) return;
    const step = Number.isFinite(dt) && dt > 0 ? Math.min(dt, 0.1) : 0.016;
    state.time += step;

    const expr = state.expression;
    expr.weight = Math.min(1, expr.weight + step * 4);
    const w = expr.weight;

    const values = {};
    const values2d = {};

    const interactionBusy = state.interaction.pet > 0.05
      || Math.abs(state.interaction.tickle) > 0.05;
    if (state.ambient.name && (interactionBusy || state.time >= state.ambient.until)) {
      cancelAmbient();
    }
    const ambientName = state.ambient.name;
    const ambientLevel = Number.isFinite(state.ambient.intensity) ? state.ambient.intensity : 1;
    const ambientSpec = ambientName && ambientName !== 'walk'
      ? ((expressions && expressions[ambientName]) || {})
      : {};
    const ambientParams = {};
    for (const [name, raw] of Object.entries(ambientSpec.params || {})) {
      if (Array.isArray(raw)) {
        ambientParams[name] = raw.map((v) => clamp((Number(v) || 0) * ambientLevel, -1, 1));
      } else {
        ambientParams[name] = clamp((Number(raw) || 0) * ambientLevel, -1, 1);
      }
    }
    const effectiveParams = {...(expr.params || {}), ...ambientParams};

    for (const name of drivenWaves) {
      let cfg = mergeMotion(motionTable[name], expr.motion?.[name], w);
      if (ambientSpec.motion?.[name]) cfg = mergeMotion(cfg, ambientSpec.motion[name], 1);
      if (ambientName === 'walk') {
        const walkMotion = ambientMotions?.walk?.[name] || DEFAULT_WALK_MOTION[name];
        if (walkMotion) cfg = mergeMotion(cfg, walkMotion, 1);
      }
      if (ambientName) cfg = scaleMotionAmp(cfg, ambientLevel);
      values[name] = waveValue(name, cfg, step);
    }

    if (hasBlink) {
      state.blinkTimer += step;
      if (state.blinkTimer >= state.nextBlink) {
        state.blink = 1;
        state.blinkTimer = 0;
        state.nextBlink = Math.random() < 0.2 ? 0.25 : 2 + Math.random() * 4;
      }
      state.blink = Math.max(0, state.blink - step * 12);
      values[chan.blink] = state.blink;
    }

    if (hasHead) {
      const wanderYaw = Math.sin(state.time * 0.35) * 0.12;
      const wanderNod = Math.sin(state.time * 0.23) * 0.06;
      const targetYaw = clamp(state.pointer.x * 0.85 + wanderYaw, -1, 1);
      const targetNod = clamp(-state.pointer.y * 0.6 + wanderNod, -1, 1);
      state.headYaw += (targetYaw - state.headYaw) * Math.min(1, step * 4);
      state.headNod += (targetNod - state.headNod) * Math.min(1, step * 4);
      if (paramSet.has(chan.headYaw)) values[chan.headYaw] = state.headYaw;
      if (paramSet.has(chan.headNod)) values[chan.headNod] = state.headNod;
    }

    if (hasEyes) {
      state.eyeX += (state.pointer.x - state.eyeX) * Math.min(1, step * 8);
      state.eyeY += (-state.pointer.y - state.eyeY) * Math.min(1, step * 8);
      values2d[chan.eyeTracking] = [state.eyeX, state.eyeY];
    }

    if (hasWalk) {
      if (ambientName === 'walk' && !interactionBusy) {
        // One full step cycle takes about 3.6s. The old 1.2s cycle made the
        // whole-leg cutouts pump rapidly and read as hovering.
        state.walk.phase = (state.walk.phase + step * 0.28) % 1;
        values[chan.walkPhase] = state.walk.phase;
      } else {
        state.walk.phase = 0;
        values[chan.walkPhase] = 0;
      }
    }

    const interactionEase = Math.min(1, step * 8);
    if (hasArmRaise) {
      const prev = Number.isFinite(state.smooth[chan.armRaise]) ? state.smooth[chan.armRaise] : 0;
      state.smooth[chan.armRaise] = prev
        + (state.interaction.pet - prev) * interactionEase;
      values[chan.armRaise] = state.smooth[chan.armRaise];
    }
    if (hasPetResponse) {
      const prev = Number.isFinite(state.smooth[chan.petResponse]) ? state.smooth[chan.petResponse] : 0;
      state.smooth[chan.petResponse] = prev
        + (state.interaction.pet - prev) * interactionEase;
      values[chan.petResponse] = state.smooth[chan.petResponse];
    }
    if (hasTickle) {
      const target = state.interaction.tickle
        ? state.interaction.tickle * Math.sin(state.time * 16)
        : 0;
      const prev = Number.isFinite(state.smooth[chan.tickle]) ? state.smooth[chan.tickle] : 0;
      state.smooth[chan.tickle] = prev + (target - prev) * interactionEase;
      values[chan.tickle] = state.smooth[chan.tickle];
    }

    const ease = Math.min(1, step * 6);
    for (const name of expressionParams) {
      if (!paramSet.has(name)) continue;
      const target = effectiveParams[name];
      const targetWeight = Object.prototype.hasOwnProperty.call(ambientParams, name) ? 1 : w;
      if (Array.isArray(target) || Array.isArray(values2d[name]) || Array.isArray(state.smooth[name])) {
        const base = values2d[name] || [0, 0];
        const goal = Array.isArray(target)
          ? [lerp(base[0], Number(target[0]) || 0, targetWeight), lerp(base[1], Number(target[1]) || 0, targetWeight)]
          : base;
        const prev = Array.isArray(state.smooth[name]) ? state.smooth[name] : goal;
        state.smooth[name] = [
          prev[0] + (goal[0] - prev[0]) * ease,
          prev[1] + (goal[1] - prev[1]) * ease,
        ];
        values2d[name] = state.smooth[name];
      } else {
        const base = Number.isFinite(values[name]) ? values[name] : 0;
        const goal = target === undefined ? base : lerp(base, Number(target) || 0, targetWeight);
        const prev = Number.isFinite(state.smooth[name]) ? state.smooth[name] : goal;
        state.smooth[name] = prev + (goal - prev) * ease;
        values[name] = state.smooth[name];
      }
    }

    for (const [name, value] of Object.entries(values)) {
      avatar.setParameter(name, value);
    }
    for (const [name, pair] of Object.entries(values2d)) {
      avatar.setParameter2D(name, pair[0], pair[1]);
    }
  }

  return {
    active,
    matched: [
      ...drivenWaves,
      ...(hasBlink ? [chan.blink] : []),
      ...(paramSet.has(chan.headYaw) ? [chan.headYaw] : []),
      ...(paramSet.has(chan.headNod) ? [chan.headNod] : []),
      ...(hasEyes ? [chan.eyeTracking] : []),
      ...(hasWalk ? [chan.walkPhase] : []),
      ...(hasArmRaise ? [chan.armRaise] : []),
      ...(hasPetResponse ? [chan.petResponse] : []),
      ...(hasTickle ? [chan.tickle] : []),
    ],
    setPointer,
    setInteraction,
    setExpression,
    triggerAmbient,
    cancelAmbient,
    update,
  };
}
