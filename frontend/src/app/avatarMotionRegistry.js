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

export function resolveAvatarMotionManifest(pack, state, fallbackSrc = '') {
  const entry = manifestByAlias.get(normalizeID(pack));
  if (!entry?.manifest) return proceduralManifest(pack, state, fallbackSrc);

  const manifest = entry.manifest;
  const activeState = state && manifest.states?.[state] ? state : 'idle';
  const stateManifest = manifest.states?.[activeState] || manifest.states?.idle || {};
  const sourceLayers = stateManifest.layers?.length
    ? stateManifest.layers
    : (manifest.states?.idle?.layers || []);
  const layers = sourceLayers
    .map((layer) => hydrateLayer(entry.folder, layer))
    .filter((layer) => layer.src);

  return {
    ...manifest,
    state: activeState,
    folder: entry.folder,
    baseSrc: assetURL(entry.folder, stateManifest.base) || fallbackSrc || '',
    layers,
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
