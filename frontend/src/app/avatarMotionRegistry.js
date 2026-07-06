const manifestModules = import.meta.glob('../assets/persona_motion/*/manifest.json', {
  eager: true,
  import: 'default',
});

const motionAssetUrls = import.meta.glob([
  '../assets/persona_motion/**/*.png',
  '!../assets/persona_motion/**/_source/*.png',
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

function assetURL(folder, relativePath) {
  if (!folder || !relativePath) return '';
  return motionAssetUrls[`../assets/persona_motion/${folder}/${relativePath}`] || '';
}

function hydrateSequence(folder, sequence) {
  if (!sequence?.frames?.length) return sequence;
  return {
    ...sequence,
    frames: sequence.frames
      .map((frame) => assetURL(folder, frame))
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

function inheritedLayerDefs(manifest, stateManifest = {}) {
  const parentStateName = stateManifest.extends;
  if (!parentStateName) return null;

  const parentLayers = manifest.states?.[parentStateName]?.layers || [];
  const omit = new Set(stateManifest.omitLayers || []);
  const overrides = new Map((stateManifest.layerOverrides || [])
    .filter((layer) => layer?.id)
    .map((layer) => [layer.id, layer]));
  const seen = new Set();

  const layers = parentLayers
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
        motion: {...(layer.motion || {}), ...(override.motion || {})},
      };
    });

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

// Cursor turn steps, ordered from back-left through back-right.
const TURN_LADDER = [
  {id: 'back_left30', value: -6},
  {id: 'back_left60', value: -5},
  {id: 'side_left', value: -4},
  {id: 'front_left60', value: -3},
  {id: 'front_left45', value: -2},
  {id: 'front_left30', value: -1},
  {id: 'front_right30', value: 1},
  {id: 'front_right45', value: 2},
  {id: 'front_right60', value: 3},
  {id: 'side_right', value: 4},
  {id: 'back_right60', value: 5},
  {id: 'back_right30', value: 6},
];

const POSE_STATES = ['front_left30_arms_up', 'front_right30_arms_up', 'back_waist', 'working_reaction', 'happy_tail_wag_front'];
const SHY_SEQUENCE = ['back_full', 'back_3q'];

function resolveSubState(entry, stateName) {
  const stateManifest = entry.manifest?.states?.[stateName];
  if (!stateManifest) return null;
  const baseSrc = assetURL(entry.folder, stateManifest.base);
  const stateSequence = hydrateSequence(entry.folder, stateManifest.sequence);
  const layers = (stateManifest.layers || [])
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

export function resolveAvatarMotionManifest(pack, state, fallbackSrc = '') {
  const entry = manifestByAlias.get(normalizeID(pack));
  if (!entry?.manifest) return proceduralManifest(pack, state, fallbackSrc);

  const manifest = entry.manifest;
  const activeState = state && manifest.states?.[state] ? state : 'idle';
  const stateManifest = manifest.states?.[activeState] || manifest.states?.idle || {};
  const stateBaseSrc = assetURL(entry.folder, stateManifest.base);
  const stateSequence = hydrateSequence(entry.folder, stateManifest.sequence);
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
  const sharedRigInteractive = stateManifest.interactive !== false
    && !usesStateSequence
    && Boolean(manifest.turnRigContract?.transitionOrder?.length);
  const interactive = stateManifest.interactive === true
    || activeState === 'idle'
    || sharedRigInteractive;
  const turnStates = interactive
    ? TURN_LADDER
      .map((t) => {
        const sub = resolveSubState(entry, t.id);
        return sub ? {...sub, value: t.value} : null;
      })
      .filter(Boolean)
    : [];
  const poseStates = interactive
    ? POSE_STATES.map((id) => resolveSubState(entry, id)).filter(Boolean)
    : [];
  const shyStates = interactive
    ? SHY_SEQUENCE.map((id) => resolveSubState(entry, id)).filter(Boolean)
    : [];
  const walkFrames = interactive ? walkFrameURLs(entry.folder) : [];
  const walkStartFrames = interactive ? sequenceFrameURLs(entry.folder, 'walk_start') : [];
  const walkStopFrames = interactive ? sequenceFrameURLs(entry.folder, 'walk_stop') : [];

  return {
    ...manifest,
    state: activeState,
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
