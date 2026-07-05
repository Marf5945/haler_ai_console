# 憂樂傻酷 Pixi Motion Pack

這個資料夾是 PixiJS 角色動圖素材包。`manifest.json` 是唯一入口，圖片可分狀態放在多個子資料夾：

- `fullbody/*.png`：每個狀態的整體 fallback 圖。
- `parts/<angle>/<state>/*.png`：可選的分層部件，例如 `front/idle/head.png`、`front/idle/pelvis_vcut.png`、`front/idle/mouth_open_talk.png`。
- `sequences/<action>/frame_000.png`：可選的逐格動作序列。

目前 manifest 先宣告狀態、動畫參數與 `partCatalog`；如果 active layer 還沒有啟用，系統會回退到既有全身圖。

## Front Idle v1

已輸出的前視角待機元件包含：

- 身體核心：頭、頸、胸、上腹、下腹、V 型骨盆。
- 腿部：左右大腿髖部版、小腿、腳踝、腳掌。
- 手臂：上臂、小臂、手腕、手腕毛套、張掌、握拳。
- 臉部：吻部鼻子、中性嘴、微笑嘴、說話張嘴、睜眼、半眯眼、眉額毛。
- 尾巴：尾根、尾中段、尾尖。

髖部不要和大腿合併成同一張圖。`pelvis_vcut.png` 是骨盆遮罩/髖臼視覺件，左右 `*_thigh_hip.png` 是獨立旋轉件；動畫時讓大腿掛在骨盆底下，靠重疊毛邊藏縫，這樣大腿才能抬到接近 90 度而不穿幫。

臉部與手部先採「替換件」策略：嘴型、眼皮、手型用狀態切換，等互動需求真的需要細節時，再把手指或瞳孔拆成更小的骨架件。

## Walk Forward v1

已補上前進走路的 8 張標準 walk cycle，放在 `sequences/walk_forward/frame_000.png` 到 `frame_007.png`：

- `frame_000`：left contact。
- `frame_001`：left down。
- `frame_002`：left passing。
- `frame_003`：left up。
- `frame_004`：right contact。
- `frame_005`：right down。
- `frame_006`：right passing。
- `frame_007`：right up。

這組是 8 格 forward walk sequence，統一切成 200x360 frame。往前走的深度感由幀本身提供：接地腳掌朝前放大、後腳縮小、passing/up 抬膝、down 壓低身體、up 抬高身體；runtime 可以直接播 PNG frame，也可以在 Pixi 裡再疊一點 y bob、x 微縮放或陰影變化。

## Front Happy Tail Wag v1

已補上正面開心大幅左右搖尾巴素材，放在 `sequences/tail_wag_happy_front/frame_000.png` 到 `frame_007.png`。這 8 格是透明 200x360 對位尾巴 overlay，不是整張全身替換圖；裁切後的尾巴素材另放在 `parts/front/idle/happy_tail_wag/tail_frame_000.png` 到 `tail_frame_007.png`。

尾巴 overlay 依 200x360 畫布對位，並遮掉前景手、身體與腿，避免把右手一起甩進尾巴素材。

## Back Waist v2

已輸出的背面手扶腰元件放在 `parts/back/waist/`。

- 身體核心：轉頭背面、頸背、上背、下背、骨盆/臀部。
- 腿部：觀眾視角左右大腿、小腿、腳掌。
- 手臂：觀眾視角左右上臂、小臂、扶在髖部的手掌。
- 臉部：側面吻部、悲傷嘴、側眼、眉、左右耳。
- 尾巴：尾根、尾中段、尾尖。

這組是手扶腰轉身用的背面候選切件，已組成 `states.back_waist`，fallback 整體圖是 `fullbody/back_waist.png`。目前 `back_waist` 採絕對座標排版，和來源圖比例一致；頭、耳、尾巴有極小幅 motion，身體主件先保持穩定，避免剛切完就出現縫線跳動。

## Front 30 v5

已重新輸出的正面 30 度過渡元件放在 `parts/front/left30/` 與 `parts/front/right30/`。`front_right30` 是由 `front_left30` 直接水平鏡像而來，避免左右角度或臉型漂移。

- 身體核心：頭、胸腹、骨盆。
- 腿部：觀眾視角左右大腿、小腿、腳掌。
- 手臂：觀眾視角左右上臂、小臂、手掌。
- 臉部備件：吻部/鼻、嘴、眼、耳。
- 尾巴：尾根、尾中段、尾尖。

這組使用 `states.front_left30` / `states.front_right30` 作為轉身橋接影格，並已升級成和新增角度一致的 22 個 canonical slots。`states.front_left30_arms_up` / `states.front_right30_arms_up` 也改指向同一套新切件，避免混到舊版 30 度骨架。

## Side Turn v1

已補上側面中間幀 `states.side_left` / `states.side_right`，fallback 整體圖放在 `fullbody/side_left.png` 與 `fullbody/side_right.png`；這兩張是 75 到 90 度側身角度，不是把正面圖旋轉或鏡射硬接。

側面也已拆成同一套 22 個 Pixi canonical slots，放在 `parts/side/left/` 與 `parts/side/right/`。拆圖直接使用既有 200x360 fullbody 圖，不縮放、不調整比例；`turnRigContract.angles` 裡用 `front_left75` / `front_right75` 作為別名，表示這兩張在轉身 sequence 裡扮演接近 75 度的側面橋接幀。

`fullbody/back_waist.png` 也已重生為肉球修正版；背面白色腳掌現在包含可讀的黑色 toe beans 與中央肉球，避免從 `side_*` 接到 `back_waist` 時腳掌細節突然消失。

尾巴上下搖晃先不另外補整張 PNG。側面已拆成 `tail_base` / `tail_mid` / `tail_tip`，可沿用 manifest 既有的局部 rotate motion；這比新增尾巴上/下兩張整圖更不容易讓身體和尾根跳位。左右側面尾巴已在 `manifest.json` 以三段不同 `pivotPx`、`phase`、`rotate.amount` 做左右搖晃，尾根小幅、尾中段延遲、尾尖最大幅。

若要做轉背面，先用狀態切換：前面 `idle` → 中間側面/過渡影格 → `back_waist`。不要直接把正面件旋轉 180 度，因為臉、胸腹、尾根和手臂遮擋關係都不同。

側面動畫建議用套圖，而不是完整 3D：

- 短轉身：用 5 到 7 張套圖最穩，例：front、front_3q、side、back_3q、back_waist，再用 crossfade、x-scale、局部頭/尾 tween 接起來。
- 可互動長時間轉向：再考慮 Live2D/Spine 類 2.5D 骨架，仍然需要側面與背面素材。
- 完整 3D：只有在未來要任意角度、換鏡頭或大量姿勢時才值得；成本是重建模型、貼圖、材質與表情，且很容易失去現在 2D 毛色手感。

目前最適合的路線是「2D 套圖 + Pixi 分層骨架」：先補側面和 3/4 過渡圖，再在 Pixi 裡切 state 或 sequence。

## Turn Angle Assets v1

已補齊這些轉身角度，讓側面轉背面時不用瞬間切成全背：

- 正面 45 度：`front_left45`、`front_right45`。
- 背面 30 度 / 背面 3/4：`back_left30`、`back_right30`，別名可記作 `back_3q_left`、`back_3q_right`。
- 中性背面 3/4 hold：`back_3q`。
- 中性完整背面：`back_full`，不要拿 `back_waist` 直接硬接側面。

比例暫時不要調整。所有新增角度維持 `manifest.size` 的 200x360 設計畫布與目前身高、頭身、胸腹、骨盆位置，只做同畫布內的透明切件與對位；等所有角度都能接起來再評估比例細修。

每個新增或重生角度都拆同一套 Pixi 可動槽位：

- 核心：頭、胸腹、骨盆。
- 手臂：觀眾視角左/右上臂、前臂、手。
- 腿：觀眾視角左/右大腿、小腿、腳。
- 尾巴：尾根、尾中段、尾尖。
- 臉部簡單表情件：吻部/鼻、嘴、眼、耳。

`manifest.json` 的 `turnRigContract` 是這份規格的機器可讀入口。完成的角度都已升成 state，且 layer id 維持 canonical slot，不會因為檔名有 `_back`、`_left45` 就改掉 Pixi 的骨架名稱。背面完整圖沒有可見臉部，因此 `back_full` 的吻部、嘴、眼是透明 placeholder，保留槽位給 Pixi runtime。

驗證指令：

```bash
cd frontend
node ./scripts/validate-yulesaku-motion-pack.js
```
