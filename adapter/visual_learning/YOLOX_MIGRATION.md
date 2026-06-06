# 視覺學習模組：YOLOX 遷移說明（授權合規）

## 為什麼換

原本使用 **YOLOv5 nano（Ultralytics）**。Ultralytics 的 YOLOv5/v8/11 程式碼**與其訓練出的權重**皆為 **AGPL-3.0**，與本專案的 **Apache-2.0** 授權不相容（AGPL 為強 copyleft，且含網路使用揭露義務）。

改用 **YOLOX**（Megvii，**Apache-2.0**，anchor-free）即可維持 Apache-2.0 相容。

> 來源：<https://github.com/Megvii-BaseDetection/YOLOX>

## 已完成的程式碼變更

- `yolo_postprocess.go`：改為 **anchor-free** 解碼（`cx=(x+gx)*s`、`cy=(y+gy)*s`、`w=exp(w)*s`、`h=exp(h)*s`），移除 anchor 表。
- `YOLOConfig` 移除 `Anchors` 欄位；新增 `DefaultYOLOXNanoConfig`（輸入 416、stride 8/16/32、COCO 80 類）。
- `yolo_detector.go` / `app.go`：模型基底路徑改為 `assets/models/yolox_nano`。
- `model_hashes.json`：清空 `models`，待你填入自行匯出模型的 SHA256。
- 測試 fixture 改為 `testdata/yolox_nano_output.json`（anchor-free）。
- 標註工具 `tools/yolo_adjust`：移除 `ultralytics` / `yolo` CLI 探測，輸出仍為通用 YOLO txt 格式。
- 全 repo 內的 `yolo_nano.mlmodelc` 權重已清為 0 位元組（AGPL 內容已移除）。

## 你還需要做的兩件事

### 1) 手動刪除殘留空資料夾（沙箱無法 unlink）

權重內容已清空，但空目錄需你在 Finder 手動刪除：

```
ui_console/ui_console_wails_v_3.0/assets/models/yolo_nano.mlmodelc/
ui_console/ui_console_wails_v_3.0/assets/models/__dummy_test__
ui_console/ui_console_wails_v_3.0/build/bin/.../assets/models/yolo_nano.mlmodelc/
old/ui_console_wails_v_2.4/.../yolo_nano.mlmodelc/
old/ui_console_wails_v_2.5/.../yolo_nano.mlmodelc/
adapter/visual_learning/testdata/yolo_nano_output.json   (舊 fixture，已清空)
```

### 2) 自行訓練 / 匯出 YOLOX 模型並接入

用 YOLOX 官方 repo（Apache-2.0）訓練你的 UI 偵測模型後匯出：

- macOS：匯出 `.mlmodelc`，放到 `assets/models/yolox_nano.mlmodelc`
- Windows：匯出 `.onnx`，放到 `assets/models/yolox_nano.onnx`

然後把檔案 SHA256 填入 `model_hashes.json`（格式 `sha256:<hex>`）。
未填或 hash 不符時，偵測器會自動降級到純 Go 的 OpenCV pipeline，不會載入未驗證模型。

### 解碼假設（如匯出設定不同需調整）

`DecodeYOLOOutput` 假設輸出 tensor 為 **未在模型內 decode** 的原始 logits：
`entry = [tx, ty, tw, th, obj_logit, cls_logit...]`，形狀 `[1, cells, 5+NumClasses]`，
flatten 順序為各 stride level 由小到大、每 level row-major（gy 外、gx 內）。
若你的匯出已在模型內完成 decode（box 為像素座標、obj/cls 已過 sigmoid），請對應調整 `yolo_postprocess.go`。
