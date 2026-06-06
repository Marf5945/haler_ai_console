# YOLO Adjust

這是一個半自動標註小工具，目標是先把截圖、DOM 候選框、預覽圖和 YOLO label 管線跑通。

## 啟動

```bash
cd ~/Desktop/yolo_adjust
python3 yolo_adjust_app.py
```

或直接雙擊 `run_yolo_adjust.command`。

## 使用流程

1. 先打開你要收資料的網頁。
2. 回到 YOLO Adjust，選瀏覽器，按「1. 調整視窗並截圖新編號」。
3. 按「2. 只複製 DOM 擷取腳本（不截圖）」。
4. 到瀏覽器 DevTools Console 貼上執行。
5. 把 Console 印出的 JSON 貼回 YOLO Adjust 的 DOM 小視窗。
6. 按「確定 / 產生預覽圖」。
7. 用右側列表檢查框；需要時可以接受/取消、修改、刪除或新增框。
8. 確認後按「輸出 YOLO labels」。

重要：DevTools 不要停靠在右側或下方，否則截圖會把 DevTools 一起截進去。
建議做法是把 DevTools 改成獨立視窗，或先執行 DOM 腳本、複製 JSON 後關掉 DevTools，再截圖。

## 輸出位置

```text
~/Desktop/yolo_adjust/dataset/
  images/train/page_001.png
  labels/train/page_001.txt
  meta/page_001.dom_candidates.json
  meta/page_001.review.json
  previews/page_001.preview.png
```

## 注意

第一次使用 macOS 可能會要求：

- Screen Recording 權限，給 Terminal 或 Python。
- Automation 權限，允許控制瀏覽器調整視窗。

如果截圖高度上下有偏差，請調整「上偏移」。Chrome 常見值大約是 80 到 100。
