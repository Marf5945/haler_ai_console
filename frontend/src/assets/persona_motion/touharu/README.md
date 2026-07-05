# 東春 Pixi Motion Pack

這個資料夾參考 `yulesaku` 的素材包結構整理：

- `fullbody/idle.png`：由 `fullbody_idle.png` 等比例縮放到 200x360 的待機 fallback。
- `fullbody/reach.png`：由 `fullbody.png` 等比例縮放到 200x360 的伸手/驚訝 fallback。
- `parts/front/idle/*.png`：由待機圖切出的透明小部件。
目前 `manifest.json` 保守設定 `renderRig: false`，所以 UI 會先顯示穩定的全身 fallback。部件、bbox、layer 草稿已放進 `partCatalog` 與 `states.idle.layers`，等之後要啟用 Pixi 骨架時，再校準 pivot、parentId 與遮擋順序後把 `renderRig` 改成 `true`。

東春的長裙遮住腿部關節，抱胸姿勢也遮住手肘/手腕，所以這一版沒有假造看不見的大腿、小腿骨架；改輸出實際可見的裙片、袖片、足袋與木屐部件。
