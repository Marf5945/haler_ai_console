import React, {useEffect, useMemo, useRef, useState} from 'react';

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

// WKWebView（macOS 後台浮窗）已知相容性問題：透明視窗一旦出現 WebGL 加速內容，
// 合成器可能把整片 webview 畫成不透明黑。對 WebKit 引擎改用 premultipliedAlpha:false
// 並關閉 MSAA；Windows（WebView2/Chromium）維持預設不受影響。
const IS_WEBKIT_ENGINE = typeof navigator !== 'undefined'
  && /AppleWebKit/i.test(navigator.userAgent || '')
  && !/Chrome|Chromium|Edg\//i.test(navigator.userAgent || '');

// 搔癢感應半徑（畫布像素）
const TICKLE_REACH = 58;
// 撫摸感應半徑：游標貼著手/腕慢慢滑 → 抬手
const PET_REACH = 34;
// 游標多久沒動視為「無聊」，開始輪播行為（走路、換姿勢）
const BORED_AFTER_MS = 10000;
// 轉向階梯進入閾值，單位=「身位」（頭像框寬度）：
// 游標離中心 1 個身位轉 30°、2 個身位轉 45°、3.2 個身位轉側面。
// 依驗收規格：游標離頭像中心 0.7 個身位 → 30°；1.4 個身位（選單那一帶）→ 45°；2.6 → 完全側面
const TURN_ENTER = [0, 0.7, 1.4, 2.6];
// 退出遲滯（身位）：往回走要少掉這麼多距離才會轉回來，防抖。
const TURN_EXIT_GAP = 0.25;

function numberValue(value, fallback) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
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

function targetFromPoint(element, clientX, clientY) {
  const rect = element?.getBoundingClientRect?.();
  if (!rect || rect.width <= 0 || rect.height <= 0) return {x: 0, y: 0, turnX: 0};
  const centerX = rect.left + rect.width / 2;
  const centerY = rect.top + rect.height * 0.22;
  return {
    x: clamp((clientX - centerX) / Math.max(42, rect.width * 0.44), -1, 1),
    y: clamp((clientY - centerY) / Math.max(58, rect.height * 0.34), -1, 1),
    // 轉向用：游標離中心幾個「身位」（不截斷，給距離階梯判斷）
    turnX: (clientX - centerX) / Math.max(1, rect.width),
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

// 關節軸心：旋轉/縮放繞「關節」轉（肩、髖、頸），不是部件圖幾何中心。
// layer.pivotPx 可明訂；沒訂時用啟發式——子部件的 positionPx 是它相對父件中心的向量，
// 關節約在兩中心連線中點，故預設 pivot = -positionPx/2（夾限在部件範圍內）。
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

  // pivot 平移補償：靜止姿勢與原本逐像素相同，只有旋轉/縮放時繞關節轉。
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
}) {
  const hostRef = useRef(null);
  const lookRef = useRef({x: 0, y: 0});
  const [failed, setFailed] = useState(false);
  const [ready, setReady] = useState(false);
  const onReadyRef = useRef(onReady);
  const stateKey = manifest?.state || expression || 'idle';
  // manifest 每次 render 都是新物件參考；序列化成穩定字串當依賴，避免不斷重掛 Pixi。
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

    // ===== 行為控制器（游標活動、轉向階梯、無聊輪播、撫摸） =====
    const ctl = {
      lastActive: performance.now(),
      nextBehaviorAt: performance.now() + BORED_AFTER_MS,
      behavior: null,
      behaviorStarted: 0,
      behaviorUntil: 0,
      behaviorIndex: 0,
      turnValue: 0,
      turnSwitched: 0,
    };
    const pet = {lift: 0, arm: 'right'};

    async function start() {
      try {
        const rigEnabled = manifest?.renderRig !== false && manifest?.layers?.length > 0;
        const mainDef = {
          layers: rigEnabled ? manifest.layers : [],
          baseSrc: manifest?.baseSrc || fallbackSrc,
          motion: manifest?.motion || {},
        };
        if (!mainDef.layers.length && !mainDef.baseSrc) {
          throw new Error('Pixi avatar has no renderable layers');
        }

        const {Application, Assets, Container, Sprite, Point, loadTextures} = await import('pixi.js');
        // 嚴格 CSP（無 unsafe-eval）：載入官方相容模組，改用非 eval polyfill。
        await import('pixi.js/unsafe-eval');
        // CSP 三道相容（都不動 CSP）：不開 blob worker；用 <img> 而非 fetch 載圖。
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
            /* 尚未完全初始化的 app 銷毀失敗可安全忽略 */
          }
          return;
        }
        app = instance;
        initialized = true;

        app.canvas.className = 'pixi-avatar-canvas';
        app.canvas.setAttribute('aria-hidden', alt ? 'false' : 'true');
        host.replaceChildren(app.canvas);

        const designSize = manifest?.size || {width, height};

        // ===== 群組（rig / 平面立繪 / 走路序列），全部掛在 stage 上交叉淡化 =====
        const groups = new Map();
        const groupDefs = new Map();
        groupDefs.set('turn:0', mainDef);
        const turnEnabled = stateKey === 'idle'
          && Array.isArray(manifest?.turnStates) && manifest.turnStates.length > 0;
        if (turnEnabled) {
          manifest.turnStates.forEach((ts) => {
            groupDefs.set(`turn:${ts.value}`, {layers: ts.layers || [], baseSrc: ts.baseSrc, motion: ts.motion || {}});
          });
          (manifest.poseStates || []).forEach((ps) => {
            groupDefs.set(`pose:${ps.id}`, {layers: ps.layers || [], baseSrc: ps.baseSrc, motion: ps.motion || {}});
          });
          if (manifest.walkFrames?.length) {
            groupDefs.set('walk', {kind: 'walk', frames: manifest.walkFrames});
          }
        }
        const behaviorPlaylist = [];
        if (turnEnabled) {
          if (groupDefs.has('walk')) behaviorPlaylist.push({key: 'walk', duration: 7000});
          if (groupDefs.has('pose:front_left30_arms_up')) behaviorPlaylist.push({key: 'pose:front_left30_arms_up', duration: 4500});
          if (groupDefs.has('walk')) behaviorPlaylist.push({key: 'walk', duration: 5500});
          if (groupDefs.has('pose:back_waist')) behaviorPlaylist.push({key: 'pose:back_waist', duration: 5000});
          if (groupDefs.has('pose:front_right30_arms_up')) behaviorPlaylist.push({key: 'pose:front_right30_arms_up', duration: 4500});
        }

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
              // 單一部件載入失敗不炸整個骨架，跳過該層即可。
              let texture = null;
              try {
                texture = await Assets.load(layer.src);
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
              node.addChild(sprite);
              applyRigLayout(node, layer);
              group.nodes.push(node);
              nodeMap.set(layer.id || `layer-${group.nodes.length}`, node);
            }
            group.nodes.forEach((node) => {
              const parent = nodeMap.get(node.__layer.parentId);
              if (node.__layer.parentId && !parent) {
                // 父件沒載成功：座標系失效，隱藏勝於錯位。
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
              const sprite = new Sprite(texture);
              sprite.width = numberValue(designSize.width, width);
              sprite.height = numberValue(designSize.height, height);
              group.root.addChild(sprite);
              group.__flatSprite = sprite;
              group.kind = 'flat';
              group.ready = true;
            } catch (err) {
              console.warn('[PixiAvatarStage] flat pose load failed', err);
            }
          }
          layoutGroup(group);
          return group;
        }

        // 主姿勢必須先就緒
        const mainGroup = await buildGroup('turn:0', mainDef);
        if (cancelled) return;
        if (!mainGroup.ready) throw new Error('Pixi avatar main pose failed to load');
        mainGroup.fade = 1;
        mainGroup.root.alpha = 1;
        mainGroup.root.visible = true;

        // 其餘姿勢背景預載（不擋首繪）
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

        try {
          const hr = host.getBoundingClientRect();
          console.log('[PIXI-DIAG] mounted',
            '| engine=', IS_WEBKIT_ENGINE ? 'webkit' : 'chromium',
            '| canvas=', app.canvas.width + 'x' + app.canvas.height,
            '| host=', Math.round(hr.width) + 'x' + Math.round(hr.height),
            '| groups=', groupDefs.size,
            '| mainNodes=', mainGroup.nodes.length,
            '| turn=', turnEnabled);
        } catch (diagErr) {
          console.warn('[PIXI-DIAG] log failed', diagErr);
        }

        tickerHandler = (ticker) => {
          const time = performance.now() / 1000;
          const now = performance.now();
          const delta = ticker.deltaTime / 60;
          const look = lookRef.current;
          const pointer = pointerRef.current;
          const tickleOn = tickleModeRef.current;
          pointer.speed *= 0.82;
          pointer.poke *= 0.86;
          if (!tickleOn) {
            pointer.poke = 0;
            pointer.speed *= 0.4;
          }
          const speedNorm = clamp(pointer.speed / 1.4, 0, 1);

          // ===== 1) 行為決策 =====
          // 互動（搔癢）模式：凍結無聊行為與轉向切換，專心被搔，避免姿勢中途跳掉。
          if (tickleOn && ctl.behavior) {
            ctl.behavior = null;
            ctl.nextBehaviorAt = now + BORED_AFTER_MS;
          }
          if (turnEnabled && !tickleOn) {
            // 無聊輪播：游標動了立刻收工；時間到換下一個行為
            if (ctl.behavior && ctl.lastActive > ctl.behaviorStarted) {
              ctl.behavior = null;
              ctl.nextBehaviorAt = now + BORED_AFTER_MS;
            } else if (ctl.behavior && now > ctl.behaviorUntil) {
              ctl.behavior = null;
              ctl.nextBehaviorAt = now + 6000 + Math.random() * 6000;
            }
            if (!ctl.behavior && behaviorPlaylist.length
              && now - ctl.lastActive > 9500 && now >= ctl.nextBehaviorAt) {
              const next = behaviorPlaylist[ctl.behaviorIndex % behaviorPlaylist.length];
              ctl.behaviorIndex += 1;
              const g = groups.get(next.key);
              if (g?.ready) {
                ctl.behavior = next;
                ctl.behaviorStarted = now;
                ctl.behaviorUntil = now + next.duration;
              }
            }
            // 轉向階梯：一步一步走（0 ↔ 30° ↔ 45° ↔ 側面），距離以「身位」計，帶遲滯防抖
            const lx = look.turnX || 0;
            const mag = Math.abs(lx);
            const sgn = lx < 0 ? -1 : 1;
            let desired = 0;
            if (mag >= TURN_ENTER[3]) desired = 3;
            else if (mag >= TURN_ENTER[2]) desired = 2;
            else if (mag >= TURN_ENTER[1]) desired = 1;
            desired *= sgn;
            const cur = ctl.turnValue;
            if (desired !== cur && now - ctl.turnSwitched > 220) {
              const target = cur + (desired > cur ? 1 : -1);
              const growing = Math.abs(target) > Math.abs(cur);
              const need = growing
                ? TURN_ENTER[Math.abs(target)]
                : TURN_ENTER[Math.abs(cur)] - TURN_EXIT_GAP;
              const ok = growing ? mag >= need : mag < need;
              if (ok && groupDefs.has(`turn:${target}`)) {
                ctl.turnValue = target;
                ctl.turnSwitched = now;
              }
            }
          }

          // ===== 2) 撫摸偵測（貼著手慢慢滑）=====
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

          // ===== 3) 決定活動群組 =====
          let key = 'turn:0';
          if (ctl.behavior) key = ctl.behavior.key;
          else if (ctl.turnValue !== 0) key = `turn:${ctl.turnValue}`;
          if (!groups.get(key)?.ready) {
            if (!groups.has(key) && groupDefs.has(key)) {
              buildGroup(key, groupDefs.get(key)).catch(() => {});
            }
            key = 'turn:0';
          }
          ctl.__activeKey = key;

          // ===== 4) 姿勢切換：先讓舊姿勢快速退場，清場後新姿勢才進場 =====
          // （兩張差異大的姿勢若同時半透明疊著，看起來像破圖；序列式切換乾淨俐落）
          let stageBusy = false;
          groups.forEach((group) => {
            if (group.key !== key && group.fade > 0.06) stageBusy = true;
          });
          groups.forEach((group) => {
            const target = (group.key === key && !stageBusy) ? 1 : 0;
            const rate = target === 1 ? 10 : 13;
            group.fade += (target - group.fade) * Math.min(1, delta * rate);
            if (group.fade < 0.012 && target === 0) group.fade = 0;
            group.root.alpha = Math.min(1, group.fade);
            group.root.visible = group.fade > 0.01;
          });

          // ===== 5) 各群組動畫 =====
          groups.forEach((group) => {
            if (!group.root.visible || !group.ready) return;
            if (group.kind === 'walk') {
              const textures = group.__walkTextures;
              const frame = Math.floor(time * 9) % textures.length;
              if (group.__walkSprite.texture !== textures[frame]) {
                group.__walkSprite.texture = textures[frame];
              }
              // 無聊踱步：左右緩慢漂移
              group.root.x = group.__rootBase.x
                + Math.sin(time * 0.5) * 16 * group.root.scale.x;
              return;
            }
            if (group.kind === 'flat') {
              // 平面立繪（側面）：輕輕呼吸浮動
              group.__flatSprite.y = Math.sin(time * 1.05) * 2.4;
              return;
            }
            const isActive = group.key === key;
            group.nodes.forEach((node, index) => {
              const base = node.__base;
              const motion = node.__motion;
              const layer = node.__layer;
              const phase = numberValue(layer.phase, index * 0.37);
              const breathe = motion.breathe || {};
              const sway = motion.sway || {};
              const lookMotion = motion.look || {};
              const blink = motion.blink || {};
              const mouth = motion.mouth || {};
              const rotate = motion.rotate || {};
              const breatheWave = Math.sin(time * numberValue(breathe.speed, 1.1) + phase);
              const swayWave = Math.sin(time * numberValue(sway.speed, 0.72) + phase);
              const blinkWindow = Math.sin(time * numberValue(blink.speed, 0.33) + phase) > 0.985;
              const mouthWave = stateKey === 'working'
                ? Math.max(0, Math.sin(time * numberValue(mouth.speed, 5.4) + phase))
                : 0;
              const rotateWave = Math.sin(time * numberValue(rotate.speed, 1) + phase);

              node.x = base.x
                + swayWave * numberValue(sway.x, 0)
                + look.x * numberValue(lookMotion.x, 0);
              node.y = base.y
                - breatheWave * numberValue(breathe.y, 0)
                + look.y * numberValue(lookMotion.y, 0)
                - mouthWave * numberValue(mouth.y, 0);
              node.rotation = base.rotation
                + swayWave * numberValue(sway.rotate, 0)
                + look.x * numberValue(lookMotion.rotate, 0)
                + rotateWave * numberValue(rotate.amount, 0);
              node.scale.x = base.scale * (1 + breatheWave * numberValue(breathe.scaleX, 0));
              node.scale.y = base.scale * (1 + breatheWave * numberValue(breathe.scaleY, 0));
              if (blink.enabled) {
                node.alpha = blink.mode === 'show'
                  ? (blinkWindow ? base.alpha : numberValue(blink.restAlpha, 0))
                  : (blinkWindow ? numberValue(blink.alpha, 0.18) : base.alpha);
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

              // 撫摸抬手：被摸的那隻手臂鏈輕輕抬起＋開心晃動
              // （幅度收在肉邊縫份內，不會露背景）
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

              // 搔癢（僅活動群組）
              let tickleTarget = 0;
              let dirX = 0;
              let dirY = 0;
              if (isActive && tickleOn && tmpPointRef && (pointer.inside || pointer.poke > 0.02)) {
                const gp = node.getGlobalPosition(tmpPointRef);
                const dx = gp.x - pointer.x;
                const dy = gp.y - pointer.y;
                const dist = Math.hypot(dx, dy) || 1;
                const proximity = clamp(1 - dist / TICKLE_REACH, 0, 1);
                tickleTarget = proximity * (pointer.inside ? 1 : 0) + proximity * pointer.poke;
                dirX = dx / dist;
                dirY = dy / dist;
              }
              const prevTickle = node.__tickle || 0;
              const nextTickle = tickleTarget > prevTickle
                ? prevTickle + (tickleTarget - prevTickle) * 0.5
                : prevTickle * 0.84;
              node.__tickle = nextTickle < 0.001 ? 0 : nextTickle;
              if (node.__tickle > 0.001) {
                const power = Math.min(1.2, node.__tickle + pointer.poke * 0.6);
                const giggleW = speedNorm * power;
                const dodgeW = (1 - speedNorm) * power;
                node.x += dirX * 7.5 * dodgeW;
                node.y += dirY * 5.5 * dodgeW;
                node.rotation += (dirX >= 0 ? 1 : -1) * 0.16 * dodgeW;
                const jitter = time * 34 + phase;
                node.x += Math.sin(jitter) * 3.6 * giggleW;
                node.y += Math.cos(jitter * 0.92) * 3.0 * giggleW;
                node.scale.x *= 1 + Math.sin(jitter) * 0.06 * giggleW;
                node.scale.y *= 1 - Math.sin(jitter) * 0.06 * giggleW;
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

    const updateLook = (event) => {
      const prev = lookRef.current;
      const next = targetFromPoint(host, event.clientX, event.clientY);
      // 游標活動偵測：位移夠大才算「有人在」，餵給無聊計時器
      if (Math.abs(next.x - prev.x) + Math.abs(next.y - prev.y) > 0.012) {
        ctl.lastActive = performance.now();
      }
      lookRef.current = next;
      updateTicklePointer(event);
    };
    const settleLook = () => {
      lookRef.current = {
        x: lookRef.current.x * 0.42,
        y: lookRef.current.y * 0.42,
        turnX: lookRef.current.turnX || 0,
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
    };
  }, [alt, fallbackSrc, height, manifestKey, stateKey, width]);

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
