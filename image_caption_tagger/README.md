# Image Caption Tagger

把圖片資料夾轉成 Stable Diffusion / LoRA 訓練常用的同名 `.txt` 標註檔。

這個版本不下載模型，也不會真的辨識「人物、衣服、物件」。它會先根據檔名、圖片尺寸、方向、亮度、色彩、透明度產生基本 tags，然後讓你人工補上真正重要的內容描述。

如果加上 `--wd14`，它會使用 WD14 anime tagger 產生更接近 Danbooru / Stable Diffusion 訓練用的 tags。預設模型是 Hugging Face 上的 `SmilingWolf/wd-v1-4-swinv2-tagger-v2`，模型頁標示 License: Apache-2.0。

## 快速使用

替資料夾內所有圖片產生同名 `.txt`：

```bash
python3 image_caption_tagger/caption_tagger.py /path/to/images --token my_style_token
```

互動模式，逐張確認並補 tags：

```bash
python3 image_caption_tagger/caption_tagger.py /path/to/images --token my_style_token --review
```

包含子資料夾：

```bash
python3 image_caption_tagger/caption_tagger.py /path/to/images --token my_style_token --recursive --review
```

加入固定 tags：

```bash
python3 image_caption_tagger/caption_tagger.py /path/to/images \
  --token my_style_token \
  --base-tags "anime style, clean lineart, soft shading" \
  --review
```

## WD14 自動標註

第一次使用 WD14 前先建立本地 env 並安裝額外依賴：

```bash
bash image_caption_tagger/setup_wd14_env.sh
source image_caption_tagger/.venv/bin/activate
```

使用 WD14 產生基本 tags，再進入人工 review：

```bash
image_caption_tagger/.venv/bin/python image_caption_tagger/caption_tagger.py /path/to/images \
  --token my_style_token \
  --wd14 \
  --review
```

常用可調參數：

```bash
image_caption_tagger/.venv/bin/python image_caption_tagger/caption_tagger.py /path/to/images \
  --token my_style_token \
  --wd14 \
  --wd14-threshold 0.35 \
  --wd14-character-threshold 0.85 \
  --review
```

說明：

- `--wd14-threshold` 越低，tags 越多也越容易混入錯誤；越高則更保守
- `--wd14-character-threshold` 控制角色名稱 tags，預設比較高，避免亂猜角色
- 預設不輸出 rating tags；需要時可加 `--wd14-include-ratings`
- 預設會把 `long_hair` 轉成 `long hair`；想保留 Danbooru 底線可加 `--wd14-keep-underscores`
- 第一次使用會透過 Hugging Face 下載 `model.onnx` 和 `selected_tags.csv`
- 之後會走 Hugging Face 快取，不需要每次重新下載

安全注意：

- WD14 依賴已釘在 `requirements-wd14.txt`
- 安裝腳本會使用 `--only-binary=:all:`，避免安裝時執行來源碼 build
- 安裝後會跑 `pip-audit` 檢查已知 Python 套件漏洞
- 這不能保證模型檔或供應鏈永遠安全，只能降低常見套件依賴風險

## 簡單介面

已提供一個 Tkinter 小介面，可以選圖片資料夾、填觸發詞、勾選 WD14、產生同名 `.txt`。

用命令啟動：

```bash
image_caption_tagger/.venv/bin/python image_caption_tagger/tagger_gui.py
```

macOS 也可以雙擊：

```text
image_caption_tagger/open_tagger_gui.command
```

介面版會直接產生標註檔；如果要逐張補細節，可以在產生後打開每張同名 `.txt` 手動編輯。

## 輸出格式

如果資料夾裡有：

```text
001.png
002.jpg
```

會產生：

```text
001.txt
002.txt
```

每個 `.txt` 裡是一行逗號分隔 caption，例如：

```text
my_style_token, portrait, high_resolution, warm_tone, bright_image
```

## 互動模式

`--review` 會逐張顯示建議 caption：

- 直接按 Enter：接受建議
- 輸入 `red dress, beach background`：補到建議後面
- 輸入 `!edit full caption here`：整行改成你輸入的 caption
- 輸入 `!skip`：跳過這張
- 輸入 `!quit`：停止

## 覆蓋規則

預設不覆蓋已存在的 `.txt`。

覆蓋既有標註：

```bash
python3 image_caption_tagger/caption_tagger.py /path/to/images --overwrite
```

把新 tags 追加到既有標註後面：

```bash
python3 image_caption_tagger/caption_tagger.py /path/to/images --append
```

## LoRA 建議

訓練風格 LoRA 時，建議：

- `--token` 使用一個不常見的觸發詞，例如 `marf_style_v1`
- 每張圖再人工補上主體、構圖、背景、衣服、情緒、媒材等 tags
- 不要只放觸發詞，否則模型比較容易把整張圖的所有內容都綁到觸發詞上
