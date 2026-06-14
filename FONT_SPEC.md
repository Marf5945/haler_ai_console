# 字體版型規範（Font Preset Spec）

本文件定義面板「字體版型」（fontPreset）的字型系統：版型如何對應到字型、字型授權與來源、檔案放置與接線方式，以及如何新增版型。

## 1. 設計總則

- **全部使用 SIL Open Font License 1.1（OFL）** 開源字型。OFL 允許免費內嵌、隨應用程式一起散布，僅需於散布物中保留授權文字、且不得單獨販售字型檔。
- **每個版型 = 一組「拉丁字型 + CJK 字型」字型堆疊（font stack）**。葡萄牙文／西班牙文／英文同屬拉丁字母，共用同一支拉丁字型即可。瀏覽器會依每個字元自動挑選字型堆疊中第一個能顯示該字的字型。
- **字型檔不進 git**（檔案大）。由 `scripts/fetch_fonts.sh` 於本機下載到 `frontend/public/fonts/`。
- **`預設` 版型不覆寫任何字型**，沿用既有依面板語言切換的系統字（`--i18n-font`）。

## 2. 版型對應表

| 版型 | 拉丁（英/葡/西） | CJK（中/日） | 風格 |
|------|------------------|--------------|------|
| 預設 Default | 系統字 | 系統字（PingFang／Noto） | 不覆寫，跟隨語言 |
| 普通 Normal | Inter | Noto Sans TC + Noto Sans JP | 中性無襯線 |
| 手寫 Handwriting | Caveat | Klee One → LXGW 文楷 | 鉛筆手寫感 |
| 書法 Calligraphy | Dancing Script | LXGW 文楷 + Yuji Syuku + Noto Serif TC | 楷書／毛筆 |
| 圓潤 Rounded | Fredoka | jf open 粉圓 + Noto Sans JP | 可愛、普普 |
| 等寬 Monospace | JetBrains Mono | Noto Sans TC/JP（後備） | 等寬程式碼風 |

> CJK 覆蓋度說明：日系字型（Klee One、Yuji Syuku）以日文漢字（JIS）為主，常用繁中多半涵蓋，少數繁中專用字會回退到字型堆疊後段的 Noto Sans TC／LXGW 文楷。需要完整繁中覆蓋的版型，皆已在堆疊中加入 LXGW 文楷或 Noto Sans TC 作為後備。

## 3. 字型來源與授權

| 字型 | 授權 | 來源 |
|------|------|------|
| Inter | OFL-1.1 | Google Fonts |
| Noto Sans TC / JP、Noto Serif TC | OFL-1.1 | Google（Noto）|
| Caveat、Dancing Script、Fredoka | OFL-1.1 | Google Fonts |
| JetBrains Mono | OFL-1.1 | JetBrains |
| Klee One、Yuji Syuku | OFL-1.1 | Fontworks（Google Fonts）|
| LXGW WenKai 霞鶩文楷 | OFL-1.1 | github.com/lxgw/LxgwWenKai |
| jf open 粉圓（jf-openhuninn） | OFL-1.1 | github.com/justfont/open-huninn-font |

精確下載網址見 `scripts/fetch_fonts.sh`。

> 關於「MIT」：完整可用的 CJK 字型在開源界幾乎都是 OFL，而非 MIT；本專案因此採用 OFL。對「內嵌進桌面 App 並隨之散布」這個用途，OFL 與 MIT 同樣自由，只是合規動作不同（保留 OFL 文字、不單售字檔）。

## 4. 安裝與建置流程

```bash
# 1) 下載字型（只需在 build 前跑一次；新環境或字型更新時重跑）
bash scripts/fetch_fonts.sh          # 全部
bash scripts/fetch_fonts.sh --core   # 只下載 普通／等寬 需要的基礎字型

# 2) 一般建置流程（vite build / wails build）會把 public/fonts 複製進 dist
```

字型放在 `frontend/public/fonts/`（Vite 的 public 目錄），建置時原樣複製到 `dist/fonts/`，Wails 再內嵌。

## 5. 接線架構（程式碼契約）

字型大小（fontScale）早已透過根元素 `--ui-font-scale` 接好；本次補上 fontPreset 的對應鏈：

1. **`frontend/src/fontFaces.js`** — 啟動時以純文字 `<style>` 注入 `@font-face`，src 指向 `fonts/<檔名>.ttf`。
   - 採執行期注入而非寫進 CSS，是為了**避免缺字型檔時 Vite build 失敗**；缺檔時瀏覽器只會回退後備字型。
   - `family` 名稱、`<檔名>.ttf` 必須與 `fetch_fonts.sh`、`FONT_PRESET_STACKS` 三方一致。
2. **`frontend/src/main.jsx`** — 在 render 前呼叫 `injectFontFaces()`。
3. **`frontend/src/App.jsx`**
   - `FONT_PRESET_STACKS`：版型 key → font-family 堆疊字串。
   - `fontPresetKey(value)`：把已儲存的（任一語言）顯示字串正規化為穩定 key。
   - `fontPresetVars(value)`：回傳 `{ '--font-console', '--i18n-font' }`；`預設` 回傳空物件。
   - 根元素 `style` 併入 `...fontPresetVars(panel.fontPreset)`，覆寫既有 `--font-console` / `--i18n-font`，全 UI 即時套用。
4. **`frontend/src/locales/*.json`** — `settings.fontNormal/fontHand/fontCalli` 等標籤（6 語系）。

CSS 端原本就以 `var(--font-console)` 與 `var(--i18n-font, var(--font-console))` 取用字型，故覆寫這兩個變數即全域生效。

## 6. 如何新增一個版型

1. `fetch_fonts.sh`：在 `ALL` 陣列加一行 `輸出檔名|網址|授權|顯示名稱`（OFL 字型）。
2. `fontFaces.js`：在 `FONT_FACES` 加上對應 `family` 與 `file`（檔名一致）。
3. `App.jsx`：在 `FONT_PRESET_STACKS` 加 `'settings.fontX': "..."`；在 `_fontPresetLabelMap` 加各語言顯示字串 → `'settings.fontX'`；在版型選單 `options` 陣列加 `t('settings.fontX')`。
4. `locales/*.json`：6 個語系各加 `"fontX": "..."`。
5. `THIRD_PARTY_NOTICES.md`：補上字型授權。

## 7. 注意事項

- **檔案大小**：CJK 可變字型單檔可達數 MB～20 MB（LXGW 文楷最大）。全部下載約數十 MB。在意體積者可只保留實際會用到的版型字型。
- **Thai 等其他語系**：未指定專屬字型，會回退系統字。如需內建可比照新增版型流程加入 Noto Sans Thai（OFL）。
- 字型檔已加入 `.gitignore`，不進版控；CI／新機器需先跑 `fetch_fonts.sh`。
