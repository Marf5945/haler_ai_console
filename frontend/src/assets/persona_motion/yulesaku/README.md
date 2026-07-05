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
