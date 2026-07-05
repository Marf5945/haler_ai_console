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
  const src = assetURL(folder, layer.src);
  return {
    ...layer,
    src,
  };
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

// 游標轉向階梯：value 絕對值越大轉越多，負=面向畫面左，正=面向畫面右。
const TURN_LADDER = [
  {id: 'side_left', value: -3},
  {id: 'front_left45', value: -2},
  {id: 'front_left30', value: -1},
  {id: 'front_right30', value: 1},
  {id: 'front_right45', value: 2},
  {id: 'side_right', value: 3},
];

// 無聊行為姿勢庫：待機久了輪播；arms_up 也做「撫摸抬手」的目標姿勢。
const POSE_STATES = ['front_left30_arms_up', 'front_right30_arms_up', 'back_waist'];

function resolveSubState(entry, stateName) {
  const stateManifest = entry.manifest?.states?.[stateName];
  if (!stateManifest) return null;
  const baseSrc = assetURL(entry.folder, stateManifest.base);
  const layers = (stateManifest.layers || [])
    .map((layer) => hydrateLayer(entry.folder, layer))
    .filter((layer) => layer.src);
  if (!layers.length && !baseSrc) return null;
  return {
    id: stateName,
    baseSrc,
    layers,
    motion: {
      ...(entry.manifest.motion || {}),
      ...(stateManifest.motion || {}),
    },
  };
}

function walkFrameURLs(folder) {
  const prefix = `../assets/persona_motion/${folder}/sequences/walk_forward/`;
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
  const sourceLayers = stateManifest.layers?.length
    ? stateManifest.layers
    : (stateBaseSrc ? [] : (manifest.states?.idle?.layers || []));
  const layers = sourceLayers
    .map((layer) => hydrateLayer(entry.folder, layer))
    .filter((layer) => layer.src);

  // 只有 idle（站姿待機）掛上轉向／行為資源；其他表情維持單一姿勢。
  const interactive = activeState === 'idle';
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
  const walkFrames = interactive ? walkFrameURLs(entry.folder) : [];

  return {
    ...manifest,
    state: activeState,
    folder: entry.folder,
    baseSrc: stateBaseSrc || fallbackSrc || '',
    layers,
    turnStates,
    poseStates,
    walkFrames,
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
