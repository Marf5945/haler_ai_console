const manifestModules = import.meta.glob('../assets/persona_motion/*/manifest.json', {
  eager: true,
  import: 'default',
});

const motionAssetUrls = import.meta.glob([
  '../assets/persona_motion/**/*.png',
  '!../assets/persona_motion/uncle_bust/**/*.png',
  '!../assets/persona_motion/**/_source/**/*.png',
  '!../assets/persona_motion/**/old/**/*.png',
  '!../assets/persona_motion/**/generated/**/*.png',
  '!../assets/persona_motion/**/tmp/**/*.png',
  '!../assets/persona_motion/**/audit/**/*.png',
], {
  eager: true,
  query: '?url',
  import: 'default',
});

const motionModelUrls = import.meta.glob([
  '../assets/persona_motion/**/*.inp',
  '!../assets/persona_motion/**/_source/**/*.inp',
  '!../assets/persona_motion/**/old/**/*.inp',
  '!../assets/persona_motion/**/generated/**/*.inp',
  '!../assets/persona_motion/**/tmp/**/*.inp',
], {
  eager: true,
  query: '?url',
  import: 'default',
});

function normalizeID(value) {
  return String(value || '').trim().toLowerCase();
}

function folderFromManifestPath(path) {
  const match = String(path || '').match(/persona_motion\/([^/]+)\/manifest\.json$/);
  return match?.[1] || '';
}

const YULESAKU_EXPRESSION_FULLBODY_FILES = new Set([
  'blocked.png',
  'happy.png',
  'idle.png',
  'idle_fixed.png',
  'sad.png',
  'sleepy.png',
  'speechless.png',
  'thinking.png',
  'warning.png',
  'working.png',
]);

function motionAssetPathCandidates(folder, relativePath) {
  const value = String(relativePath || '').trim();
  const candidates = [value];
  if (folder !== 'yulesaku' || !value.startsWith('fullbody/')) return candidates;

  const filename = value.slice('fullbody/'.length);
  if (filename.startsWith('turn_left_')) {
    candidates.push(`fullbody/turnleft/${filename}`);
  } else if (filename.startsWith('turn_right_')) {
    candidates.push(`fullbody/turnright/${filename}`);
  } else if (YULESAKU_EXPRESSION_FULLBODY_FILES.has(filename)) {
    candidates.push(`fullbody/Expressions/${filename}`);
  }
  return [...new Set(candidates)];
}

function assetURL(folder, relativePath) {
  if (!folder || !relativePath) return '';
  if (isUnsafeMotionAssetPath(relativePath)) return '';
  for (const candidatePath of motionAssetPathCandidates(folder, relativePath)) {
    const url = motionAssetUrls[`../assets/persona_motion/${folder}/${candidatePath}`];
    if (url) return url;
  }
  return '';
}

function modelURL(folder, relativePath) {
  if (!folder || !relativePath) return '';
  if (isUnsafeMotionAssetPath(relativePath)) return '';
  const url = motionModelUrls[`../assets/persona_motion/${folder}/${relativePath}`];
  return url || '';
}

function isUnsafeMotionAssetPath(relativePath) {
  const value = String(relativePath || '').toLowerCase();
  return /(^|\/)(?:_source|tmp|old|generated|audit)\//.test(value)
    || /(?:contact[_-]?sheet|source[_-]?boxes|sprite[_-]?sheet|regen(?:erated)?|_green)(?:[./_-]|$)/.test(value);
}

function hydrateSequence(folder, sequence) {
  if (!sequence?.frames?.length) return sequence;
  return {
    ...sequence,
    frames: sequence.frames
      .map((frame) => assetURL(folder, typeof frame === 'string' ? frame : frame?.src))
      .filter(Boolean),
  };
}

const manifestEntries = Object.entries(manifestModules).map(([path, manifest]) => ({
  folder: folderFromManifestPath(path),
  manifest,
}));

const manifestByAlias = manifestEntries.reduce((map, entry) => {
  const ids = [
    entry.folder,
    entry.manifest?.id,
    ...(entry.manifest?.aliases || []),
  ].map(normalizeID).filter(Boolean);
  ids.forEach((id) => map.set(id, entry));
  return map;
}, new Map());

function hydrateLayer(folder, layer = {}) {
  const sequence = hydrateSequence(folder, layer.sequence);
  const src = assetURL(folder, layer.src) || sequence?.frames?.[0] || '';
  return {
    ...layer,
    src,
    sequence,
  };
}

function mergeLayerMotion(base = {}, override = {}) {
  const merged = {...base};
  Object.entries(override || {}).forEach(([key, value]) => {
    const baseValue = base?.[key];
    merged[key] = value && typeof value === 'object' && !Array.isArray(value)
      ? {...(baseValue || {}), ...value}
      : value;
  });
  return merged;
}

function inheritedLayerDefs(manifest, stateManifest = {}) {
  const parentStateName = stateManifest.extends;
  if (!parentStateName) return null;

  const parentLayers = manifest.states?.[parentStateName]?.layers || [];
  const omit = new Set(stateManifest.omitLayers || []);
  const overrides = new Map((stateManifest.layerOverrides || [])
    .filter((layer) => layer?.id)
    .map((layer) => [layer.id, layer]));
  const seen = new Set();

  const layers = (stateManifest.prependLayers || [])
    .filter((layer) => layer?.id && !omit.has(layer.id));
  layers.forEach((layer) => seen.add(layer.id));

  layers.push(...parentLayers
    .filter((layer) => !omit.has(layer.id))
    .map((layer) => {
      const override = overrides.get(layer.id);
      seen.add(layer.id);
      if (!override) return layer;
      return {
        ...layer,
        ...override,
        positionPx: {...(layer.positionPx || {}), ...(override.positionPx || {})},
        size: {...(layer.size || {}), ...(override.size || {})},
        anchor: {...(layer.anchor || {}), ...(override.anchor || {})},
        motion: mergeLayerMotion(layer.motion || {}, override.motion || {}),
      };
    }));

  (stateManifest.layers || []).forEach((layer) => {
    if (layer?.id && !omit.has(layer.id)) {
      layers.push(layer);
      seen.add(layer.id);
    }
  });
  (stateManifest.layerOverrides || []).forEach((layer) => {
    if (layer?.id && !seen.has(layer.id) && !omit.has(layer.id)) layers.push(layer);
  });

  return layers;
}

function proceduralManifest(pack, state, fallbackSrc) {
  return {
    id: normalizeID(pack) || 'wolf',
    renderer: 'pixi',
    state: state || 'idle',
    size: {width: 200, height: 360},
    baseSrc: fallbackSrc || '',
    layers: [],
    motion: {
      breathe: {speed: 1.18, y: 4.5, scaleX: 0.006, scaleY: 0.004},
      sway: {speed: 0.72, x: 1.8, rotate: 0.012},
      look: {x: 7, y: 4, rotate: 0.018},
    },
  };
}

// Cursor turn steps, ordered from the left turn arc through the right turn arc.
const TURN_LADDER = [
  {id: 'turn_left_010', value: -1},
  {id: 'turn_left_020', value: -2},
  {id: 'turn_left_030', value: -3},
  {id: 'turn_left_040', value: -4},
  {id: 'turn_left_050', value: -5},
  {id: 'turn_left_060', value: -6},
  {id: 'turn_left_070', value: -7},
  {id: 'turn_left_080', value: -8},
  {id: 'turn_left_090', value: -9},
  {id: 'turn_left_105', value: -10},
  {id: 'turn_left_120', value: -11},
  {id: 'turn_left_135', value: -12},
  {id: 'turn_left_150', value: -13},
  {id: 'turn_left_165', value: -14},
  {id: 'turn_left_180', value: -15},
  {id: 'turn_right_010', value: 1},
  {id: 'turn_right_020', value: 2},
  {id: 'turn_right_030', value: 3},
  {id: 'turn_right_040', value: 4},
  {id: 'turn_right_050', value: 5},
  {id: 'turn_right_060', value: 6},
  {id: 'turn_right_070', value: 7},
  {id: 'turn_right_080', value: 8},
  {id: 'turn_right_090', value: 9},
  {id: 'turn_right_105', value: 10},
  {id: 'turn_right_120', value: 11},
  {id: 'turn_right_135', value: 12},
  {id: 'turn_right_150', value: 13},
  {id: 'turn_right_165', value: 14},
  {id: 'turn_right_180', value: 15},
];

const POSE_STATES = ['happy_tail_wag_front'];
const SHY_SEQUENCE = ['back_full', 'back_3q'];
const FLAT_POSE_STATES = new Set(['back_waist']);

function resolveSubState(entry, stateName) {
  const stateManifest = entry.manifest?.states?.[stateName];
  if (!stateManifest) return null;
  const baseSrc = assetURL(entry.folder, stateManifest.base);
  const matchedSequence = Object.values(entry.manifest?.sequences || {})
    .find((sequence) => sequence?.sourceState === stateName);
  const stateSequence = hydrateSequence(entry.folder, stateManifest.sequence || matchedSequence);
  const inheritedLayers = inheritedLayerDefs(entry.manifest, stateManifest);
  const sourceLayers = FLAT_POSE_STATES.has(stateName)
    ? []
    : stateSequence?.frames?.length
    ? []
    : (inheritedLayers?.length ? inheritedLayers : (stateManifest.layers || []));
  const layers = sourceLayers
    .map((layer) => hydrateLayer(entry.folder, layer))
    .filter((layer) => layer.src);
  if (!layers.length && !baseSrc && !stateSequence?.frames?.length) return null;
  return {
    id: stateName,
    baseSrc: baseSrc || stateSequence?.frames?.[0] || '',
    sequence: stateSequence,
    layers,
    motion: {
      ...(entry.manifest.motion || {}),
      ...(stateManifest.motion || {}),
    },
  };
}

function walkFrameURLs(folder) {
  return sequenceFrameURLs(folder, 'walk_forward');
}

function sequenceFrameURLs(folder, sequenceName) {
  const prefix = `../assets/persona_motion/${folder}/sequences/${sequenceName}/`;
  return Object.keys(motionAssetUrls)
    .filter((key) => key.startsWith(prefix))
    .sort()
    .map((key) => motionAssetUrls[key]);
}

function resolveInochi2DConfig(entry, stateManifest = {}) {
  const source = {
    ...(entry.manifest?.inochi2d || {}),
    ...(stateManifest.inochi2d || {}),
  };
  if (!source.enabled) return null;
  const anchors = Object.entries(source.anchors || {}).reduce((map, [anchorName, anchor]) => {
    const modelSrc = modelURL(entry.folder, anchor.model);
    map[anchorName] = {
      ...anchor,
      modelSrc,
    };
    return map;
  }, {});
  const transitions = Object.entries(source.transitions || {}).reduce((map, [name, frames]) => {
    map[name] = (frames || [])
      .map((frame) => {
        const src = assetURL(entry.folder, typeof frame === 'string' ? frame : frame?.src);
        if (!src) return null;
        return typeof frame === 'string' ? src : {...frame, src};
      })
      .filter(Boolean);
    return map;
  }, {});
  return {
    ...source,
    runtimeUrl: source.runtimeUrl || '/inochi2d-runtime/inochi_fox_demo.js',
    anchors,
    transitions,
  };
}

export function resolveAvatarMotionManifest(pack, state, fallbackSrc = '') {
  const entry = manifestByAlias.get(normalizeID(pack));
  if (!entry?.manifest) return proceduralManifest(pack, state, fallbackSrc);

  const manifest = entry.manifest;
  const activeState = state && manifest.states?.[state] ? state : 'idle';
  const stateManifest = manifest.states?.[activeState] || manifest.states?.idle || {};
  const stateBaseSrc = assetURL(entry.folder, stateManifest.base);
  const stateSequence = hydrateSequence(entry.folder, stateManifest.sequence);
  const inochi2d = resolveInochi2DConfig(entry, stateManifest);
  const hasInochi2DModel = Boolean(inochi2d && Object.values(inochi2d.anchors || {}).some((anchor) => anchor.modelSrc));
  const inheritedLayers = inheritedLayerDefs(manifest, stateManifest);
  const sourceLayers = inheritedLayers?.length
    ? inheritedLayers
    : stateManifest.layers?.length
    ? stateManifest.layers
    : (stateBaseSrc ? [] : (manifest.states?.idle?.layers || []));
  const layers = sourceLayers
    .map((layer) => hydrateLayer(entry.folder, layer))
    .filter((layer) => layer.src);

  const usesStateSequence = Boolean(stateSequence?.frames?.length);
  const turnRigEnabled = manifest.turnRigEnabled !== false;
  const poseRigEnabled = manifest.poseRigEnabled !== false;
  const shyRigEnabled = manifest.shyRigEnabled !== false;
  const walkRigEnabled = manifest.walkRigEnabled !== false;
  const sharedRigInteractive = stateManifest.interactive !== false
    && !usesStateSequence
    && turnRigEnabled
    && Boolean(manifest.turnRigContract?.transitionOrder?.length);
  const interactive = turnRigEnabled && (stateManifest.interactive === true
    || activeState === 'idle'
    || sharedRigInteractive);
  const turnStates = interactive
    ? TURN_LADDER
      .map((t) => {
        const sub = resolveSubState(entry, t.id);
        return sub ? {...sub, value: t.value} : null;
      })
      .filter(Boolean)
    : [];
  const poseStates = interactive
    && poseRigEnabled
    ? POSE_STATES.map((id) => resolveSubState(entry, id)).filter(Boolean)
    : [];
  const shyStates = interactive
    && shyRigEnabled
    ? SHY_SEQUENCE.map((id) => resolveSubState(entry, id)).filter(Boolean)
    : [];
  const walkFrames = interactive && walkRigEnabled ? walkFrameURLs(entry.folder) : [];
  const walkStartFrames = interactive && walkRigEnabled ? sequenceFrameURLs(entry.folder, 'walk_start') : [];
  const walkStopFrames = interactive && walkRigEnabled ? sequenceFrameURLs(entry.folder, 'walk_stop') : [];
  const turnTransitions = Object.entries(manifest.turnTransitions || {}).reduce((map, [id, transition]) => {
    const sequence = hydrateSequence(entry.folder, transition);
    if (sequence?.frames?.length) {
      map[id] = {
        ...transition,
        ...sequence,
        from: transition.from,
        to: transition.to,
      };
    }
    return map;
  }, {});

  // 逐格圖序列（供「PNG 逐格」動作模式使用）。key = sequence 資料夾名，
  // 另加 randomAnimation/表情用的別名（walk -> walk_forward）。
  const frameSequences = {};
  for (const [name, seq] of Object.entries(manifest.sequences || {})) {
    const urls = sequenceFrameURLs(entry.folder, name);
    if (!urls.length) continue;
    frameSequences[name] = {
      frames: urls,
      fps: Number(seq?.fps) > 0 ? Number(seq.fps) : 8,
      loop: seq?.loop !== false,
    };
  }
  if (frameSequences.walk_forward && !frameSequences.walk) {
    frameSequences.walk = frameSequences.walk_forward;
  }

  return {
    ...manifest,
    state: activeState,
    renderer: hasInochi2DModel ? 'inochi2d' : manifest.renderer,
    inochi2d,
    interactive,
    folder: entry.folder,
    baseSrc: stateBaseSrc || stateSequence?.frames?.[0] || fallbackSrc || '',
    sequence: stateSequence,
    layers,
    turnStates,
    poseStates,
    shyStates,
    walkFrames,
    walkStartFrames,
    walkStopFrames,
    turnTransitions,
    frameSequences,
    motion: {
      ...(manifest.motion || {}),
      ...(stateManifest.motion || {}),
    },
  };
}

export function listAvatarMotionPacks() {
  return manifestEntries.map((entry) => ({
    id: entry.manifest?.id || entry.folder,
    folder: entry.folder,
    aliases: entry.manifest?.aliases || [],
    name: entry.manifest?.name || entry.folder,
  }));
}
