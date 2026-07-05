import React, {useEffect, useMemo, useRef, useState} from 'react';

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

// WKWebView（macOS 後台浮窗）已知相容性問題：透明視窗一旦出現 WebGL 加速內容，
// 合成器可能把整片 webview 畫成不透明黑（全身像＋動態圖像切換時的大黑塊）。
// 對 WebKit 引擎改用 premultipliedAlpha:false 並關閉 MSAA，讓 GL canvas 以直通 alpha 合成；
// Windows（WebView2/Chromium）維持預設不受影響。
const IS_WEBKIT_ENGINE = typeof navigator !== 'undefined'
  && /AppleWebKit/i.test(navigator.userAgent || '')
  && !/Chrome|Chromium|Edg\//i.test(navigator.userAgent || '');

// 搔癢感應半徑（畫布像素）：游標距離某個身體節點在此範圍內才會被搔到。
const TICKLE_REACH = 58;

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

function defaultLayer(src) {
  return {
    id: 'base',
    src,
    anchor: {x: 0.5, y: 1},
    position: {x: 0.5, y: 0.985},
    scale: 1,
    motion: {
      breathe: {speed: 1.12, y: 4, scaleX: 0.006, scaleY: 0.005},
      sway: {speed: 0.72, x: 1.6, rotate: 0.01},
      look: {x: 5, y: 3, rotate: 0.012},
    },
  };
}

function targetFromPoint(element, clientX, clientY) {
  const rect = element?.getBoundingClientRect?.();
  if (!rect || rect.width <= 0 || rect.height <= 0) return {x: 0, y: 0};
  const centerX = rect.left + rect.width / 2;
  const centerY = rect.top + rect.height * 0.22;
  return {
    x: clamp((clientX - centerX) / Math.max(42, rect.width * 0.44), -1, 1),
    y: clamp((clientY - centerY) / Math.max(58, rect.height * 0.34), -1, 1),
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

function applySpriteLayout(sprite, layer, width, height) {
  const anchor = layer.anchor || {};
  const position = layer.position || {};
  sprite.anchor.set(numberValue(anchor.x, 0.5), numberValue(anchor.y, 1));

  const fitW = numberValue(layer.fitWidth, sprite.texture.width);
  const fitH = numberValue(layer.fitHeight, sprite.texture.height);
  const baseScale = Math.min(width / Math.max(1, fitW), height / Math.max(1, fitH));
  const scale = baseScale * numberValue(layer.scale, 1);

  sprite.__base = {
    x: numberValue(position.x, 0.5) * width,
    y: numberValue(position.y, 0.985) * height,
    scale,
    rotation: numberValue(layer.rotation, 0),
    alpha: numberValue(layer.alpha, 1),
  };

  sprite.x = sprite.__base.x;
  sprite.y = sprite.__base.y;
  sprite.scale.set(scale);
  sprite.rotation = sprite.__base.rotation;
  sprite.alpha = sprite.__base.alpha;
}

function applyRigRoot(root, width, height, designSize) {
  const designWidth = numberValue(designSize?.width, width);
  const designHeight = numberValue(designSize?.height, height);
  const scale = Math.min(width / Math.max(1, designWidth), height / Math.max(1, designHeight));
  root.scale.set(scale);
  root.x = (width - designWidth * scale) / 2;
  root.y = (height - designHeight * scale) / 2;
}

function applyRigLayout(node, layer) {
  const sprite = node.__sprite;
  const anchor = layer.anchor || {};
  const position = layerPosition(layer, 0, 0);
  const visualScale = fitScale(sprite.texture, layer.size, 1) * numberValue(layer.scale, 1);
  const scaleX = visualScale * numberValue(layer.flipX ? -1 : 1, 1);
  const scaleY = visualScale * numberValue(layer.flipY ? -1 : 1, 1);

  sprite.anchor.set(numberValue(anchor.x, 0.5), numberValue(anchor.y, 0.5));
  sprite.x = numberValue(layer.offsetPx?.x, 0);
  sprite.y = numberValue(layer.offsetPx?.y, 0);
  sprite.scale.set(scaleX, scaleY);

  node.__base = {
    x: position.x,
    y: position.y,
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
  // manifest 每次 render 都是新物件參考；序列化成穩定字串當依賴，
  // 避免 useEffect 每次 render 都重掛 Pixi（不斷建立/銷毀 WebGL context）。
  const manifestKey = useMemo(() => {
    try {
      return JSON.stringify(manifest);
    } catch (_) {
      return String(manifest?.id || '');
    }
  }, [manifest]);
  // 搔癢用的指標狀態：host 內座標、滑動力道(speed)、是否在感應範圍、左鍵poke衝量。
  // 用 ref 保存，切換 tickle 模式時不會重掛 Pixi。
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
    const sprites = [];
    const rigNodes = [];
    const rigNodeMap = new Map();
    let rigRoot = null;
    let tickerHandler = null;
    let tmpPointRef = null;
    const sourceLayers = manifest?.layers?.length
      ? manifest.layers
      : [defaultLayer(manifest?.baseSrc || fallbackSrc)];
    const isRig = sourceLayers.some((layer) => layer.positionPx || layer.parentId || layer.size);

    async function start() {
      try {
        const usableLayers = sourceLayers.filter((layer) => layer.src);
        if (!usableLayers.length) throw new Error('Pixi avatar has no renderable layers');

        const {Application, Assets, Container, Sprite, Point} = await import('pixi.js');
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
          // 元件在 init 完成前就被卸載（StrictMode 會故意重掛一次）：
          // 自己把剛建好的實例銷毀掉，別讓半初始化的 app 流回 cleanup。
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

        if (isRig) {
          rigRoot = new Container();
          rigRoot.label = `${manifest?.id || 'avatar'}-rig-root`;
          app.stage.addChild(rigRoot);
        }

        for (const layer of usableLayers) {
          const texture = await Assets.load(layer.src);
          if (cancelled) return;
          const sprite = new Sprite(texture);
          sprite.label = layer.id || 'layer';
          if (isRig) {
            const node = new Container();
            node.label = layer.id || 'layer';
            node.__sprite = sprite;
            node.__layer = layer;
            node.__motion = layerMotion(layer, manifest?.motion || {});
            node.addChild(sprite);
            applyRigLayout(node, layer);
            rigNodes.push(node);
            rigNodeMap.set(layer.id || `layer-${rigNodes.length}`, node);
          } else {
            sprite.__layer = layer;
            sprite.__motion = layerMotion(layer, manifest?.motion || {});
            applySpriteLayout(sprite, layer, app.screen.width, app.screen.height);
            sprites.push(sprite);
            app.stage.addChild(sprite);
          }
        }

        if (isRig) {
          rigNodes.forEach((node) => {
            const parent = rigNodeMap.get(node.__layer.parentId);
            (parent || rigRoot).addChild(node);
          });
        }

        const resize = () => {
          const rect = host.getBoundingClientRect();
          const nextWidth = Math.max(1, Math.round(rect.width || width));
          const nextHeight = Math.max(1, Math.round(rect.height || height));
          app.renderer.resize(nextWidth, nextHeight);
          if (isRig) {
            applyRigRoot(rigRoot, nextWidth, nextHeight, manifest?.size || {width, height});
            rigNodes.forEach((node) => applyRigLayout(node, node.__layer));
          } else {
            sprites.forEach((sprite) => applySpriteLayout(sprite, sprite.__layer, nextWidth, nextHeight));
          }
        };
        resizeObserver = new ResizeObserver(resize);
        resizeObserver.observe(host);
        resize();

        tickerHandler = (ticker) => {
          const time = performance.now() / 1000;
          const delta = ticker.deltaTime / 60;
          const look = lookRef.current;
          // 搔癢力道每幀衰減：手停/離開後慢慢平復；poke 衝量衰減更快（點一下彈一下）。
          const pointer = pointerRef.current;
          const tickleOn = tickleModeRef.current;
          pointer.speed *= 0.82;
          pointer.poke *= 0.86;
          if (!tickleOn) {
            pointer.poke = 0;
            pointer.speed *= 0.4;
          }
          // 力道正規化：慢滑→偏閃躲扭身，快滑→偏咯咯笑抖動。
          const speedNorm = clamp(pointer.speed / 1.4, 0, 1);
          const animatedNodes = isRig ? rigNodes : sprites;
          animatedNodes.forEach((node, index) => {
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

            // === 搔癢：感應游標下最近的部位，力道混合（輕搔=閃躲扭身，快滑=咯咯笑抖動）===
            let tickleTarget = 0;
            let dirX = 0;
            let dirY = 0;
            if (tickleOn && tmpPointRef && (pointer.inside || pointer.poke > 0.02)) {
              const gp = node.getGlobalPosition(tmpPointRef);
              const dx = gp.x - pointer.x;
              const dy = gp.y - pointer.y;
              const dist = Math.hypot(dx, dy) || 1;
              const proximity = clamp(1 - dist / TICKLE_REACH, 0, 1);
              // 越靠近該部位越癢；poke（左鍵）額外加一記。
              tickleTarget = proximity * (pointer.inside ? 1 : 0) + proximity * pointer.poke;
              dirX = dx / dist;
              dirY = dy / dist;
            }
            const prevTickle = node.__tickle || 0;
            // 被搔到瞬間彈起(0.5)、離開後緩慢平復(0.84)，做出「怕癢一縮」的手感。
            const nextTickle = tickleTarget > prevTickle
              ? prevTickle + (tickleTarget - prevTickle) * 0.5
              : prevTickle * 0.84;
            node.__tickle = nextTickle < 0.001 ? 0 : nextTickle;
            if (node.__tickle > 0.001) {
              const power = Math.min(1.2, node.__tickle + pointer.poke * 0.6);
              const giggleW = speedNorm * power;        // 快速大幅滑動 → 咯咯笑
              const dodgeW = (1 - speedNorm) * power;    // 輕輕慢搔 → 閃躲扭身
              // 閃躲：往游標反方向縮一下，順勢歪身。
              node.x += dirX * 7.5 * dodgeW;
              node.y += dirY * 5.5 * dodgeW;
              node.rotation += (dirX >= 0 ? 1 : -1) * 0.16 * dodgeW;
              // 咯咯笑：高頻抖動 + 擠壓回彈。
              const jitter = time * 34 + phase;
              node.x += Math.sin(jitter) * 3.6 * giggleW;
              node.y += Math.cos(jitter * 0.92) * 3.0 * giggleW;
              node.scale.x *= 1 + Math.sin(jitter) * 0.06 * giggleW;
              node.scale.y *= 1 - Math.sin(jitter) * 0.06 * giggleW;
            }
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

    // 更新搔癢指標：換算成 host 內座標，並用滑動位移/時間估「力道」(px/ms)。
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
      lookRef.current = targetFromPoint(host, event.clientX, event.clientY);
      if (tickleModeRef.current) updateTicklePointer(event);
    };
    const settleLook = () => {
      lookRef.current = {
        x: lookRef.current.x * 0.42,
        y: lookRef.current.y * 0.42,
      };
    };
    // 左鍵點一下 = 一記搔癢（poke 衝量），落在游標下最近的部位。
    const pokeDown = (event) => {
      if (!tickleModeRef.current || event.button !== 0) return;
      updateTicklePointer(event);
      if (pointerRef.current.inside) pointerRef.current.poke = 1;
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
      // 只銷毀「已完成初始化」的 app，並吞掉任何錯誤——
      // 銷毀時再拋例外會冒泡到 React commit 階段，直接打爆整個畫面（黑屏）。
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
    return <img className="floating-avatar-img pixi-avatar-fallback pixi-avatar-fallback-animated" src={manifest?.baseSrc || fallbackSrc} alt={alt} draggable={false} />;
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
