# AI Console UI Decisions v2.0

狀態：v2.0 Review / DAG 高風險確認、低風險可改選、設定 workspace 深色化、面板風格排序  
適用範圍：`ui_console_wails_v_1.2/frontend/src/App.jsx`、`frontend/src/style.css`、`frontend/src/tailwind.css`  
對應規格：`AI_Console_UI_v1_9.md`  
來源：承接 `UI_DECISIONS_v1_12.md`，固化 Review Panel、設定 workspace、人格列與面板風格排序決策

參考圖：

- `reference/home-preview-input.png`
- `reference/settings-preview.png`

---

## 0. v2.0 更新重點

- 設定頁已改成真正的 settings workspace：左側控制欄保留，中央 + 右側合併，首頁中央內容與右側工具欄在設定開啟時不 render。
- `App` 根節點在設定開啟時加上 `settings-open` class，`.console-shell.settings-open` 使用兩欄布局。
- 設定 workspace 主背景由大面積橘色收斂成深色工作台，只保留暖橘光暈、邊框、active glow 等 accent。
- `原諒青` 設定側欄字體需維持白色；不得因面板背景變深而改成黑字。
- 人格頭像列縮小 25%，使用 `.persona-card-row { width: 75%; }` 控制，仍維持水平 carousel。
- 人格頭像列下方橫桿固定在頭像列與表單之間的 grid row，不再用 `translateY(clamp(...vw...))` 或負 margin 隨視窗寬度飄移。
- 角色名稱欄限制 100 字，使用 `maxLength={100}` 與 `Array.from(...).slice(0,100)` 雙層保護，避免中文被半字截斷。
- 面板風格選項改為每行 3 個、按鈕寬度隨文字變長，文字不得換行；風格按鈕可拖拉改變排序，排序存入 localStorage。
- Review Panel 預設只顯示一行摘要：`高風險確認` 與 `可改選選項`，不得用大卡片壓縮對話空間。
- 高風險狀態為 pending 時，`高風險確認` 摘要按鈕需變亮，提示使用者需要處理。
- 點擊 `高風險確認` 才顯示彈出式詳細窗；詳細窗包含 `permission summary`、`target paths`、預測 `diff`、`skill id`、`summary hash`、逾時與後果。
- 點擊 `可改選選項` 才顯示低風險多 skill 匹配彈窗；彈窗包含候選 skill、match reason、score、risk 與「不再打擾」小時輸入。
- 高風險 pending 時，底部 composer 切換為 `[是] [否] ▸ 自行輸入`；確認或拒絕後回到一般聊天 composer。
- Review 詳細窗為浮動 popup，疊在工作區上方，不改變 conversation grid 高度，不把訊息列表往下擠。
- Review 詳細窗開啟時必須高於 conversation panel；對話訊息層不得蓋過高風險或可改選 popup。
- Review 詳細窗需掛在全域 viewport 層，不能留在 conversation/top-console 的局部 stacking context 裡。
- 左側 adapter icon 節奏、top-console 比例、右側工具卡片質感、訊息泡泡內縮感已調整，保持工具型工作台密度。
- 面板設定列右側選擇按鈕需固定在標籤右方中段位置，不得壓住文字，也不得貼出外框；字體大小選項為 `50% / 80% / 90% / 100% / 110% / 120%` 並需即時影響主要 UI 文字。
- 純 Vite 預覽缺少 Wails runtime 時，前端需以 `callWails()` 捕捉同步 throw 並回退 fallback data，避免黑畫面。
- 角色包拖入需用 `readAsText()` 讀取 JSON，不得用 Data URL 造成 JSON parse 失敗。
- 首頁人格頭像預設只顯示頭像；點擊頭像才展開名稱與職業，再點一次收起。名稱與職業仍可在展開後編輯。
- 首頁 top-console 的 haㄌer / sub 列只顯示 sub 名稱，方塊高度縮半，使用隱藏式水平捲動，不顯示額外拉桿。
- `system prompt` 與 Review 摘要功能列排列在 sub 列下方的獨立區域，不得覆蓋 haㄌer / sub 卡片。
- sub 列外框不得保留大面積空白下緣；只包住 sub 按鈕高度與必要 padding。
- 工具彈窗內容採 app launcher icon 形式，不再使用重複的大型工具按鈕；tab 為 `外部連結`、`自動流程`、`工具包`，且可實際切換。
- 工具彈窗右上搜尋 icon 必須展開可輸入搜尋框，並即時篩選目前分類內的工具。
- 右側常駐工具欄固定為 `引用連結` 與 `引用文件` 兩格，不顯示 `＋` 佔位按鈕。`引用連結` 點擊後彈出輸入框，支援網址或本地位置；`引用文件` 接受檔案拖入。
- 右側常駐工具欄需提供 `學習` 與 `錄製` 模式按鈕；學習開啟後 20 分鐘未操作時整理 CLI token / sub 流程 / skill，錄製開啟時閃爍提示正在教系統螢幕操作。
- 外部檔案拖入 `引用文件` 後，需複製到 app 可讀資料夾（目前為 app data 的 `data/references/files`），並在 `引用文件` 下方顯示檔名；檔名最多兩行，每行最多 20 字。
- 工具 icon 拖到右側使用工具區代表加入最愛；從右側「使用工具」彈窗拖出後的 `移除最愛` 只移除最愛，不從總工具選單解除。從一般工具彈窗拖出後的 `解除` 才代表完全移除工具欄位。
- 工具 icon 拖到 app 外不得自動產生桌面文件；只有使用者點擊 `複製` 後，才進入「是否安裝 xxskill」確認流程。`外部連結` 類工具的 `複製` 需反灰不可用。

---

## 0. v1.12 更新重點

- Wails 視窗不得鎖死過高：預設高度改為 `860`，最小高度改為 `560`；前端 `.console-shell` 最小高度同步為 `560px`。
- 視窗高度必須允許使用者縮小，避免 macOS menu bar / Dock 擋住右下「使用工具」按鈕。
- 右側「使用工具」按鈕不得貼齊視窗底部，需保留 `36px` 到 `72px` 的下方距離。
- 左右側欄外框需貼近視窗左右內緣，不保留大段外側空白。
- 左右側欄按鈕框高度不得因字體放大而變高；按鈕維持緊湊高度，文字與圖形用 `clamp()` 自適應縮放，完整顯示且不得截斷。
- 側欄按鈕文字不得使用省略號截斷；若空間不足，需縮小字級以符合按鈕框。
- 左側「工具」與右側「使用工具」入口顏色跟隨目前面板風格，不使用固定亮綠色。
- 入口按鈕只比同主題一般按鈕更明顯：`喔黏菊`、`敗北藍`、`粉切黑`可偏亮；`原諒青`、`消極白`需偏暗、穩定、可讀。

## 0.1. 2026-05-11 UI 修正摘要

- Review 詳細 popup 改用 viewport portal，避免 conversation panel 再次蓋住高風險、可改選、Pending Digest 詳細窗。
- top-console 的 sub 列空白已縮減，system prompt / Review 功能列改排在 sub 列下方，不覆蓋 sub 卡片。
- 右側工具欄加入 `學習` / `錄製` 模式按鈕；學習狀態文案使用 `修改中`，錄製開啟時以閃爍顯示。
- 工具彈窗右上搜尋 icon 已改為可展開輸入框，支援即時篩選目前 tab 的工具。
- 左側面板設定列按鈕固定在標籤右方中段，避開標籤與 scrollbar；字體大小選項擴為 `50% / 80% / 90% / 100% / 110% / 120%`，並透過 `--ui-font-scale` 影響主要 UI 文字。
- 本輪新增與調整的程式碼區塊已補上段落註解，範圍包含右欄學習/錄製、工具搜尋、Review viewport popup、設定列定位與字體縮放。
- 右上互動按鈕改為固定 icon-only 欄位，瀏覽器狀態 chip 限寬顯示，避免白色主題下撐破 top-console。
- 設定列選項按鈕拆成 value / caret，value 超出時以省略號收斂，避免擋住「繁中 / 自動 / 預設 / 100%」文字。
- I-7 Execution Hook / DAG 前端入口設在 Review 摘要列（黃框區）內；點擊後以 popup 顯示 DAG 執行程序、目前正在執行的 node、每個步驟的開始/完成時間與耗時。
- DAG run 由送出特定任務訊息後系統自動建立，不要求使用者手動先建 run；DAG run 一建立就必須同步呼叫 `StartHookRun(dagRunID, outlineID)`。
- Hook 永遠記錄，不依賴「學習」按鈕；觀察範圍包含 CLI / subagent 執行步驟與輸出擷取、使用者 UI 點擊、工具使用、螢幕錄製學習資料。
- 每個 DAG node 完成時都要呼叫 `RecordStepTrace` 並產生一次 node summary；node summary 顯示於 Review Panel 的 Pending Digest 與工具彈窗的 `自動流程` tab。
- 高風險 DAG node 必須卡住，不得繼續跑相依 node；卡住狀態需同時顯示在 DAG popup / 自動流程，以及 Review Panel 的高風險確認。
- `學習` 模式不負責啟動 Hook；它只負責在 20 分鐘未操作後整理 Hook 記錄、CLI token、sub 流程與 skill 候選。
- 「工具」與「使用工具」彈窗可同時開啟，互不關閉。
- 左側「工具」彈窗需位於中央上半區，右側「使用工具」彈窗需位於中央下半區；色塊標註只代表位置與形狀，不代表要更改背景色。
- 工具彈窗背景色由主題決定，不因位置調整改色。
- 開啟設定時，左側設定框背景需與同主題工具彈窗背景一致；不得改工具彈窗背景來遷就設定框。
- 工具彈窗 tab 不顯示獨立「工」「具」字樣；此 v1.12 舊決策原為「套件包」，v2.0 起已覆蓋為「工具包」。
- 人格設定頭像卡只顯示頭像，不在卡片底部覆蓋「人格 A / B / C」名稱。
- 人格頭像列下方的橫桿需上移到頭像列下方的黃框區域，不能掉到大段空白下方。
- 人格設定表單需上移並貼近橫桿下方，避免中間留出無用大空白。
- 角色強度仍維持 `10%` 到 `100%`，代表語氣提示詞加入 system prompt 的擲骰機率。
- 每次 UI 修改若要提供 `build/bin/ai-console.app` 給使用者測試，必須先跑 `npm run build`，再跑 `/Users/tester/go/bin/wails build`。

---

## 0. v1.11 更新重點

- 新增人格頭像資料夾：`frontend/src/assets/persona_avatars/`。
- 預設頭像檔案：`persona-a.svg`、`persona-b.svg`、`persona-c.svg`、`persona-d.svg`。
- `App.jsx` 以 `new URL('./assets/persona_avatars/...', import.meta.url).href` 連結頭像，讓 Vite / Wails build 可一起打包。
- 人格 A / B / C / D 的卡片視為頭像卡，不再是文字資訊卡。
- 頭像卡必須為正方形，以卡片寬度作為正方形邊長。
- 卡片必須 `overflow: hidden`，圖片 `object-fit: cover`，避免圖像或框線越界。
- 卡片圓角、內距、底部名稱標籤需跟隨正方形尺寸收斂，不得撐高卡片。
- 設定頁頭像列下方的滑桿不得遮擋頭像，需與頭像列保留明確垂直間距。
- 角色身份、回答策略、角色強度等下方表單也必須跟著滑桿下移。
- `.settings-persona-drawer` 必須允許垂直捲動，避免正方形頭像變高後裁切底部表單。
- Wails 若使用 embedded `frontend/dist`，前端 build 後仍需重開 Wails；若使用已編譯 app，需重新 Wails build。
- 預設人格只保留 A/B/C；第四格不是人格 D，而是「＋ 新增人格 / 拖入角色包」入口。
- 人格總數最多 16 個，前端與後端都必須限制。
- 舊版預設 `persona-d` 若已被存入 settings，且沒有任何自訂內容，啟動時應自動移除並還原第四格新增入口。
- 正確開發入口為 `/Users/tester/go/bin/wails dev` 啟動的新 Wails dev 視窗；`build/bin` 內的舊 app / executable 可能是舊 bundle，不作為目前修 UI 的驗證來源。
- 人格頭像列必須是水平滑動 carousel，不得換行成 4x4 或多列網格。
- 每次可視區最多顯示 4 格。
- 超過 4 格時，使用者透過底部拉桿、水平捲動或左右箭頭左右移動。
- 左右箭頭每次推動一個可視頁寬。
- 不得額外顯示人格列自己的瀏覽器 scrollbar；下方截圖中既有的白色長條才是唯一可見拉桿。
- `.persona-card-row` 可保留水平 scroll 行為，但 `scrollbar-width: none` 且 `::-webkit-scrollbar { display: none; }`。
- 已重新執行 `/Users/tester/go/bin/wails build`，更新 `build/bin/ai-console.app`。
- 面板風格選項更新為 5 個可直接切換的配置：`喔黏菊`（目前預設橘黑畫面）、`消極白`、`粉切黑`、`原諒青`、`敗北藍`；舊資料中的 `預設` / `喔黏橘` 視為 `喔黏菊`。

---

## 1. 現況掃描

目前 UI 已從舊版內嵌 HTML / CSS / JS 轉為 Wails + React + Tailwind 實作。

主要檔案：

- `frontend/src/App.jsx`：主要 React component 與互動狀態
- `frontend/src/style.css`：Tailwind v4 source CSS 與主要元件樣式
- `frontend/src/tailwind.css`：由 `style.css` build 產生，不應手修

目前畫面仍是三欄工作台：

```txt
+---------------------------------------------------------------+
| 左側控制欄 | 中央工作區                                  | 右側工具欄 |
+---------------------------------------------------------------+
```

目前狀態：

- 左側已是垂直 adapter 卡片與 command stack。
- 中央已分為 `top-console` 與 `conversation-panel`。
- 右側已有工具卡片欄。
- 底部輸入已是聊天 composer。
- `haㄌer` 已是橫向滑列，且目前沒有顯示 `subagent 介面` 字串。
- 設定相關 component 已升級為 `SettingsWorkspace`：設定開啟時左側控制欄保留，中央 + 右側合併為獨立設定 workspace，右側工具欄不 render。

---

## 2. v1.6 核心決策

### 2.1 底部輸入區

平常狀態固定為聊天 composer：

```txt
[ + ] [ 輸入訊息...                                      ] [ 送出 ]
```

互動規則：

- Enter 送出。
- Shift + Enter 換行。
- 送出後即時 append 到對話列表。
- 不得在平常狀態常駐顯示 `是 / 否 / 自行輸入`。

高風險確認狀態才切換為：

```txt
[是] [否]
▸ 自行輸入
```

觸發條件：

- Review / DAG / action 需要使用者明確同意。
- high / critical / destructive 或其他高風險操作。

未確認後果：

- 選 `否`：取消該 action，DAG node 進入 blocked / cancelled，後續相依步驟不得繼續。
- 逾時未選：Review Card 過期或 action 暫停，必須重新產生 Review Card 或重新確認。
- 關閉視窗：不得視為同意。
- UI 必須在 Review Panel 或 runtime notice 顯示停止、逾時、作廢或需重新確認的原因。

進階風險狀態處理（新增自 v2.0 規格）：
- **高風險 Skill 注入**：UI 必須額外展示 `permission summary`、`target paths`、預計資源使用、預測 `diff`、`skill id` 與 `summary hash`。
- **低風險歧義（Ambiguous low-risk）**：若多個低風險 skill 匹配，需顯示非阻塞「可改選」卡片，並必須包含「不再打擾」的時數輸入設定（單位：小時）。
- v2.0 起，Review Panel 預設不得展開成大卡片；只顯示一行摘要列，包含 `高風險確認` 與 `可改選選項`。
- v2.0 起，Review 詳細資訊需使用彈出式視窗；彈窗疊在工作區上方，不得壓縮對話空間。
- `高風險確認` pending 時摘要字體與邊框需變亮；使用者點擊後才顯示完整高風險詳細窗。
- `可改選選項` 點擊後才顯示低風險多 skill 候選詳細窗。


### 2.1.1 I-7 Execution Hook / DAG 前端觸發點

目的：

- 補齊目前只有 Execution Hook 後端、沒有 DAG 前端觸發點的缺口。
- 讓 Hook Run 能從使用者送出任務後自動建立，並在 UI 中可見、可追蹤、可卡住高風險 node。

入口位置：

- DAG 前端入口放在 Review 摘要列，也就是 `Review Panel · 高風險確認 · Skill Activity · Pending Digest` 所在的黃框區。
- 入口需是一個明確可點擊的 DAG / 自動流程狀態按鈕，不得只藏在工具彈窗內。
- 點擊入口後開啟浮動 popup，不改變 conversation grid 高度。

popup 內容：

- 顯示目前 DAG run 名稱或任務摘要。
- 顯示每個 node 正在執行什麼、狀態、開始時間、完成時間、耗時。
- 顯示 Hook Run id / DAG Run id / outline id，方便追蹤證據鏈。
- 高風險 node 卡住時，popup 內該 node 必須顯示等待確認狀態。

建立與啟動：

- 使用者送出某些任務訊息後，系統自動建立 DAG run。
- DAG run 一建立，就必須同步呼叫 `StartHookRun(dagRunID, outlineID)`。
- 不要求使用者先手動按「建立 DAG」。

Hook 觀察範圍：

- CLI / subagent 執行步驟。
- CLI 輸出、step trace、工具結果與必要的擷取紀錄。
- 使用者 UI 點擊。
- 工具使用事件。
- 螢幕錄製學習資料。

node summary：

- 每個 DAG node 完成時都要呼叫 `RecordStepTrace`。
- 每個 node 完成時產生一次 summary，而不是等整個 DAG 完成才整理。
- node summary 需出現在 Review Panel 的 Pending Digest。
- node summary 也需出現在工具彈窗的 `自動流程` tab。

高風險卡住規則：

- 高風險 DAG node 必須暫停，不得繼續執行相依 node。
- 卡住狀態需同時顯示在 DAG popup / 自動流程，以及 Review Panel 的高風險確認。
- 使用者確認前，不得把卡住視為同意。

學習模式關係：

- DAG Hook 永遠記錄，不依賴 `學習` 按鈕是否開啟。
- `學習` 模式只負責在 20 分鐘未操作後整理 Hook 記錄、CLI token、sub 流程與 skill 候選。
- `錄製` 模式提供螢幕操作資料來源；錄製資料可被 Hook summary / 學習整理引用，但不取代 Hook Run。

### 2.2 haㄌer 滑列

決策：

- 不顯示 `subagent 介面` 文字。
- 一次顯示 5 個角色方塊。
- 多出的角色用水平滑動查看。
- 下方保留水平拉桿或滑動提示。
- 主 agent 用呼吸燈或小狀態點標示。

目前實作符合：

- `TopConsole` 內的 `.haora-band` / `.haora-scroll` / `.haora-card` 已是橫向滑列。
- 目前畫面文字只顯示 `主haㄌer`、`haㄌer1` 等角色名稱，沒有舊字串。

Review Rail / Panel 擴展：
- 必須支援非阻塞、廣播式的 **Skill Activity Card**。
- 每次顯示最新 2 筆 Skill 注入紀錄（包含 `skill_id`、`match reason`、`summary hash`），支援展開查看詳細 score、risk 與清除狀態。

下一輪修正方向：

- 微調 top deck 比例，讓人格卡、Status Rail、haㄌer 滑列更接近 reference。
- 保持橫向滑列，不回到文字型入口。

### 2.3 設定 Workspace

v1.6 起採 v1.9 的設定 workspace 決策：

```txt
+---------------------------------------------------------------+
| 左側控制欄 |             設定 Workspace（中央 + 右側）          |
+---------------------------------------------------------------+
```

規則：

- 左側控制欄保留，不被覆蓋。
- 中央主頁內容隱藏。
- 右側工具欄隱藏。
- 設定 workspace 佔滿原本中央 + 右側。
- 關閉設定後完整回到首頁。
- 不使用半透明 overlay 假裝 workspace。
- 不把人格設定退回右側窄欄。

### 2.3.1 人格頭像卡（v1.7）

人格設定中的 `人格 A / B / C / D` 卡片即為頭像卡。

資料夾：

```txt
frontend/src/assets/persona_avatars/
├── persona-a.svg
├── persona-b.svg
├── persona-c.svg
└── persona-d.svg
```

程式連結：

```js
new URL('./assets/persona_avatars/persona-a.svg', import.meta.url).href
```

排版規則：

- 每張 `.settings-persona-card` 必須是正方形。
- 正方形邊長以目前卡片寬度決定，使用 `aspect-ratio: 1 / 1`。
- `.persona-card-row` 必須是 `grid-flow-col` 水平列。
- `.persona-card-row` 的 `grid-auto-columns` 以 `(100% - 3 gaps) / 4` 計算，確保同一視窗最多 4 格。
- `.persona-card-row` 不得 wrap；超出的卡片進入水平捲動。
- `.persona-card-row` 不得顯示第二條 scrollbar。
- 卡片圖片使用 `.settings-persona-avatar`，必須 `position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover;`。
- 卡片必須 `overflow: hidden`，避免頭像、光暈或框線超出方框。
- 名稱標籤固定在頭像底部，不得增加卡片高度。
- v1.12 起，設定 workspace 的人格頭像卡不再覆蓋顯示人格名稱；名稱改由下方表單欄位管理，避免遮擋頭像。
- 圓角建議不超過 12px，與目前工作台卡片語言一致。
- `.settings-persona-drawer` 不得 `overflow: hidden` 裁掉下方表單；應使用 `overflow-y: auto`。
- `.settings-card-track` 需位於頭像列下方，不得與頭像卡重疊。
- v1.12 起，`.settings-card-track` 需上移到頭像列下方的短橫桿區域，不得掉到大段空白下方。
- v1.12 起，`.persona-form` 需上移貼近滑桿下方，不得保留大段無用空白。

目前實作符合：

- `fallbackSettings.personas` 已保留 `avatarUrl` 欄位。
- 預設頭像由 `personaAvatarUrls` map 提供。
- `getPersonaAvatar(persona)` 會優先使用使用者頭像，否則回退到預設 SVG。
- `.settings-persona-card` 已改為正方形，`.settings-persona-avatar` 已填滿卡片。
- `.settings-persona-drawer` 已改成可垂直捲動。
- `.settings-card-track` 已加高並與頭像列拉開。
- `.persona-form` 已增加上方與欄位間距。
- `.persona-card-row` 已改為水平 carousel，每頁最多顯示 4 格。
- `.settings-card-track` 左右箭頭會水平捲動人格列。
- `.persona-card-row` 原生 scrollbar 已隱藏，視覺上只保留下方既有拉桿。
- v1.12 起，設定頭像卡片不再 render `<strong>{persona.name}</strong>` 名稱覆蓋。
- v1.12 起，滑桿與下方設定表單使用視覺位移上移，以符合目前標註位置。

### 2.3.2 新增人格入口（v1.9）

規則：

- 預設只顯示 `人格 A`、`人格 B`、`人格 C`。
- 第四格顯示 `＋ 新增人格`，不是 `人格 D`。
- 點擊第四格會建立新人格。
- 使用者可把角色包檔案拖到第四格安裝。
- 角色包 JSON 可包含：
  - `schema`
  - `id`
  - `name`
  - `icon`
  - `avatarUrl`
  - `identity`
  - `replyStrategy`
  - `roleStrength`
  - `personality`
  - `scenario`
  - `description`
- 從人格卡拖出時，匯出為 `ai-console.persona.v1` JSON 角色包，必須包含名稱與人格設定欄位，不得只匯出 persona id。
- 若角色包無法解析，使用檔名作為人格名稱並建立「拖入的角色包」。
- 點擊或按住人格頭像卡前，必須先提交目前人格設定表單的 DOM 值，避免最後一筆尚未 blur 的文字未進入 state。
- `roleStrength` 以 `10%` 到 `100%` 儲存，只代表「語氣提示詞」被加入 system prompt 的擲骰機率，不代表任務權限或角色能力。
- system prompt 組裝時以 `roleStrength` 擲骰；命中時加入由角色個性、使用場景、其餘描述組成的語氣提示詞。
- 人格上限為 16；達上限後不顯示新增卡。

後端規則：

- `settings.MaxPersonas = 16`。
- `SavePersona` 新增人格前必須檢查上限。
- `defaultState()` 不得再建立 `persona-d`。
- `removeLegacyDefaultPersonaD()` 只移除舊版空白預設 D，不移除使用者自訂角色。

目前狀態：

- `App` 依 `activePanel === 'settings'` 加上 `settings-open` class。
- settings 開啟時不 render `TopConsole`、`ConversationPanel`、`RightRail`。
- settings 開啟時 render 跨中央 + 右側的 `SettingsWorkspace`。
- `SettingsMenu` 留在左欄作為設定導覽。
- `PersonaSettingsDrawer` 仍作為人格設定內容 component，但外層語意由 `SettingsWorkspace` 承接，不再是只覆蓋中央欄的 drawer。

下一輪修正方向：

- 後續可再把 `PersonaSettingsDrawer` 正式改名成 `PersonaSettingsWorkspaceContent` 或等效名稱，避免 component 名稱殘留 drawer 語意。

### 2.4 工具面板

目前工具面板分兩處：

- 左側 `ToolsMenu`：點擊左側工具後展開。
- 右側 `RightRail`：常駐工具卡片。

決策：

- 首頁保留右側工具欄，但右下另有「使用工具」入口。
- 左側「工具」與右側「使用工具」可以同時打開彈窗。
- 左側「工具」彈窗位於中央上半區；右側「使用工具」彈窗位於中央下半區。
- 彈窗位置標註色塊只代表幾何位置和形狀，不代表背景顏色。
- 工具彈窗背景色必須跟隨面板風格，不得固定為綠色。
- 工具彈窗的座標需基於變數計算（嚴格遵守以下幾何）：
  ```css
  .tools-popup {
    left: calc(var(--shell-pad-left) + var(--left-rail-width) + var(--shell-gap));
    right: calc(var(--shell-pad-right) + var(--right-rail-width) + var(--shell-gap));
  }
  .tools-popup-left {
    top: 28px;
  }
  .tools-popup-right {
    bottom: 28px;
  }
  ```
- 左右工具入口按鈕也必須跟隨面板風格，只比一般按鈕更容易辨識。
- `喔黏菊`、`敗北藍`、`粉切黑` 的工具入口可比一般按鈕亮。
- `原諒青`、`消極白` 的工具入口需較暗，避免淺色背景下刺眼。
- 工具彈窗 tab 不顯示單獨的「工」「具」字樣。
- 第三個 tab 使用「工具包」，不得再使用「套件包」。
- 右側「使用工具」按鈕不得貼底；需保留底部距離避免被 Dock 或視窗底邊遮住。
- 右側工具卡應維持卡片感，不回到整條亮橘色。
- 右側常駐欄固定顯示 `引用連結` 與 `引用文件`，不得顯示 `＋` 佔位。`引用連結` 是輸入網址或本地位置的彈框入口；`引用文件` 是檔案 drop zone。
- `引用文件` 收到外部檔案拖入時，需將檔案複製到 app data 可讀位置，再在 `引用文件` 下方顯示檔名。檔名顯示最多兩行，每行最多 20 字。
- 左右工具彈窗內容應為 app launcher icon grid，不得回到整列按鈕清單。左側工具彈窗每欄三行，右側「使用工具」彈窗每欄兩行，icon 與間距維持緊湊。
- 工具 icon 可拖曳：拖到右側使用工具區為加入最愛；從右側「使用工具」彈窗拖出後，操作彈窗顯示 `移除最愛 / 複製 / 取消`，其中 `移除最愛` 只移除最愛，不完全解除工具。
- 從一般工具彈窗拖出後，操作彈窗顯示 `解除 / 複製 / 取消`；這裡的 `解除` 才是完全從工具欄位移除。
- 工具 icon 拖到 app 外不得自動產生桌面文件或文字 payload；只有點擊 `複製` 後才進入「是否安裝 xxskill（檔案名稱）」確認。`外部連結` 類工具的 `複製` 需反灰不可用。

下一輪修正方向：

- 工具抽屜與右側工具欄的關係之後再和 v3.3 工具入口契約對齊。

### 2.5 側欄與視窗幾何（v1.12）

Wails 視窗：

- `wails_main.go` 預設 `Height` 為 `860`。
- `wails_main.go` 最小 `MinHeight` 為 `560`。
- 前端 `.console-shell` 最小高度同步為 `560px`。
- 不得用前端 `min-height` 或 Wails `MinHeight` 把視窗鎖到過高，導致使用者無法調整高度。

三欄版面：

- `.console-shell` 左右外側 padding 為 `0px`，讓視窗內容外框貼近左右側欄。
- 左右側欄寬度使用 CSS 變數：
  - `--left-rail-width`
  - `--right-rail-width`
  - `--shell-gap`
- 三欄變數具體實作如下（不採用舊版固定 30:75:15 比例）：
  ```css
  .console-shell {
    --shell-pad-left: 0px;
    --shell-pad-right: 0px;
    --shell-gap: 16px;
    --left-rail-width: clamp(220px, 17.2vw, 352px);
    --right-rail-width: clamp(210px, 16vw, 328px);
    grid-template-columns: var(--left-rail-width) minmax(580px, 1fr) var(--right-rail-width);
    min-height: 560px;
  }
  ```
- 工具彈窗位置需以這些變數計算，避免手寫固定座標導致不同視窗寬度失準。

側欄按鈕：

- 按鈕框高度維持緊湊，不因字體放大而增加高度。
- 文字與 icon 可放大，但必須用 `clamp()` 限制在按鈕框內。
- 按鈕文字不可 `truncate` 或顯示省略號；需完整顯示，空間不足時縮小字級。
- `工具`、`使用工具` 入口同樣遵守按鈕框尺寸限制。

### 2.6 設定框背景（v1.12）

- 開啟設定時，左側 `settings-side-panel` 背景要與目前主題的工具彈窗背景一致。
- 不要反向修改工具彈窗背景；只調整設定框背景去對齊工具彈窗。
- 每個面板風格都需有對應覆蓋：
  - `喔黏菊`：橘棕工具彈窗背景
  - `消極白`：灰白工具彈窗背景
  - `粉切黑`：粉黑工具彈窗背景
  - `原諒青`：深綠工具彈窗背景
  - `敗北藍`：藍青玻璃感工具彈窗背景

---

## 3. 配色決策

目前實作已是黑底暖色工作台，但有偏橘過重的風險。

保留方向：

```css
--bg: #050403;
--surface: #151412;
--surface-2: #1f1d1a;
--amber: #c56f08;
--amber-soft: #e18a18;
--brown: #6f3708;
--msg: #7f2d20;
--msg-2: #a8422c;
--pink: #f23a9a;
--text: #fff8ef;
--muted: rgba(255,248,239,.62);
```

禁用：

- 大面積亮青背景。
- 大面積亮綠按鈕。
- 亮藍 system prompt。
- 亮紅滿版訊息條。
- 設定頁整條高飽和橘色。
- 單一色相壓滿整個畫面。

下一輪修正方向：

- 設定 workspace 已收斂成深色 workspace + 暖色 accent；後續只需微調對比，不得回到大面積橘色。
- `haora-band` 可保留橘色，但要避免和設定頁一起形成滿版橘色。
- pink 只作 Codex、選取狀態、送出按鈕、高權限提示 accent。

---

## 4. Component 對照

目前 React component 對照：

| Component | 角色 | 狀態 |
| --- | --- | --- |
| `App` | 全域狀態與版面組裝 | 已有 `settings-open` class 與 settings layout 分支 |
| `Sidebar` | 左側 adapter / command / settings / tools | 可保留，settings 開啟時作左側設定導覽 |
| `SettingsMenu` | 面板設定 | 已可用，面板風格可拖拉排序 |
| `ToolsMenu` | 左側工具抽屜 | 可保留 |
| `TopConsole` | 人格卡 / Status Rail / haㄌer | 已可用，v2.0 已微調比例 |
| `ConversationPanel` | system prompt / review / 對話 / composer | 已有 Review 摘要列、彈出式詳細窗與高風險 composer |
| `ReviewPanel` | 高風險確認 / 低風險可改選 | v2.0 新增；預設一行摘要，詳細內容用 popup |
| `PersonaSettingsDrawer` | 人格設定內容 | 已放入 `SettingsWorkspace`，仍可後續改名消除 drawer 語意 |
| `RightRail` | 右側工具欄 | 首頁保留，settings 開啟時隱藏 |

---

## 5. 已完成

- React / Wails UI 已存在。
- 主頁已是黑底暖色卡片工作台。
- 底部輸入已是聊天 composer。
- Enter / Shift + Enter 行為已實作。
- `haㄌer` 已是橫向滑列。
- 首頁 haㄌer / sub 列已改為隱藏式水平捲動，只顯示 sub 名稱，不再顯示雙拉桿或額外 icon。
- 主畫面未顯示 `subagent 介面` 文字。
- 左側已有 adapter stack、工具、設定入口。
- 設定資料可透過 Wails binding 儲存。
- 工具列表可透過 Wails binding 讀取並觸發。
- 已建立人格頭像資料夾與 A/B/C/D 預設 SVG 頭像。
- 人格卡已改為正方形頭像卡，並以寬度決定高度。
- 拉桿與下方表單已下移，避免遮擋正方形頭像卡。
- 第四格已改為新增人格 / 拖入角色包入口。
- 後端已限制最多 16 個人格，且移除舊版空白預設 D。
- 人格列已改為水平滑動，每次最多顯示 4 格。
- `build/bin/ai-console.app` 已重新打包為最新版。
- Wails 視窗高度限制已降低，避免視窗被鎖到超出可用螢幕高度。
- `.console-shell` 最小高度已與 Wails `MinHeight` 對齊為 `560px`。
- 右側「使用工具」按鈕已上移，避免貼底或被 Dock 遮住。
- 左右側欄外側大留白已移除。
- 側欄按鈕文字不再截斷，改用自適應字級完整顯示。
- 工具入口按鈕與彈窗顏色已改為跟隨面板風格。
- 左右工具彈窗已可同時打開，並分別位於中央上半區與下半區。
- 人格頭像卡已移除名稱覆蓋。
- 人格設定橫桿與下方表單已上移到目前標註位置。
- 設定框背景已調整為跟同主題工具彈窗一致。
- 設定頁已改成真正 workspace，settings 開啟時中央與右側合併，首頁中央內容與 `RightRail` 不 render。
- 設定 workspace 背景已由大面積橘色收斂成深色工作台 + 暖色 accent。
- `原諒青` 設定側欄字體維持白色。
- 人格頭像列已縮小 25%，橫桿固定在頭像列下方 grid row，不再隨視窗寬度飄移。
- 角色名稱欄已限制 100 字並支援中文安全截斷。
- 面板風格按鈕已改為每行 3 個、按鈕寬度跟文字走，且可拖拉排序。
- Review Panel 預設已收斂成一行摘要：`高風險確認` 與 `可改選選項`。
- 高風險 pending 時，`高風險確認` 摘要按鈕會變亮提示處理。
- 高風險詳細資料與低風險可改選資料已改為彈出式視窗，不壓縮對話空間。
- `system prompt` 與 Review 摘要功能列已移到 sub 列下方的獨立區域，不覆蓋 sub 卡片。
- 高風險 composer 已支援 `[是] [否] ▸ 自行輸入`，確認或拒絕後回到一般聊天 composer。
- 高風險詳細窗已顯示 permission summary、target paths、diff、skill id、summary hash、逾時與後果說明。
- 低風險可改選彈窗已顯示多 skill 候選、match reason、score、risk 與「不再打擾」時數設定。
- 左側 adapter icon 節奏、top-console 比例、右側工具卡片質感、訊息泡泡內縮感已調整。
- 純 Vite 預覽缺少 Wails runtime 時可回退 fallback data，不再黑畫面。
- 角色包拖入已改用 `readAsText()` 讀取 JSON。
- 首頁人格頭像已改為點擊展開 / 收合名稱與職業。
- 右側常駐欄已改為 `引用連結` 與 `引用文件` 固定入口，並移除 `＋` 佔位。
- `引用連結` 已支援點擊彈出輸入框；`引用文件` 已支援檔案拖入後複製到 app data 並在下方顯示檔名。
- 工具彈窗已改為 icon grid；右側「使用工具」彈窗只顯示最愛工具，`移除最愛` 後 icon 會從該彈窗消失。

---

## 6. 待修正

P0：

- 無開放 P0。

P1：

- 將目前前端 fallback review state 串接到實際 Review / DAG / skill router 後端資料。
- 高風險 Review Card 逾時倒數需由 runtime 或 DAG 狀態驅動，目前為前端示意資料。
- 後續若支援上傳頭像，必須寫入 persona `avatarUrl` 或等效欄位，且輸出圖像仍需裁切為正方形。
- 角色包安裝之後可再補檔案安全檢查與 manifest schema，目前先支援 JSON 拖入建立人格。

P2：

- 後續可依 reference 再細修 top-console、右側工具卡與訊息泡泡視覺，但不得回到滿版訊息長條或細長工具欄。

---

## 7. 驗證方式

前端 build：

```bash
cd "/Users/tester/Desktop/AI攜帶型助手/實作/ui_console/ui_console_wails_v_1.2/frontend"
npm run build
```

Wails / Go 測試：

```bash
cd "/Users/tester/Desktop/AI攜帶型助手/實作/ui_console/ui_console_wails_v_1.2"
go test ./...
```

開發預覽：

```bash
cd "/Users/tester/Desktop/AI攜帶型助手/實作/ui_console/ui_console_wails_v_1.2/frontend"
npm run dev
```

---

## 8. 接手注意

- 不要手修 `frontend/src/tailwind.css`，它是 build 產物。
- UI 修改主要改 `frontend/src/App.jsx` 與 `frontend/src/style.css`。
- Wails 視窗尺寸在 `wails_main.go`；不要再把 `MinHeight` 或 `.console-shell min-height` 拉回過高。
- 若調整視窗高度或底部工具入口，必須實測 macOS menu bar + Dock 情境，確認右下「使用工具」不被遮住。
- 不要把設定頁退回右側窄欄。
- 不要把聊天 composer 退回常駐三選項按鈕。
- 高風險 composer 只能在 Review / DAG 高風險 pending 時出現；一般聊天不得顯示 `[是] [否] ▸ 自行輸入`。
- Review Panel 預設只能是一行摘要列，不得讓高風險與可改選大卡片常駐壓縮對話空間。
- `高風險確認` 與 `可改選選項` 詳細內容必須是彈出式視窗。
- 高風險 pending 時摘要文字與邊框需變亮，讓使用者知道需要確認。
- 不要讓 `subagent 介面` 字串回到主畫面。
- 不要把設定頁做成半透明 modal overlay。
- 不要把設定 workspace 改回大面積橘色；需維持深色工作台 + 暖色 accent。
- 工作台 UI 要穩定、可掃描、可長時間使用；不要做 landing page 或裝飾性 hero。
- 不要把人格 A/B/C/D 頭像卡改回扁長文字卡。
- 不要在設定人格頭像卡底部重新覆蓋人格名稱。
- 調整頭像尺寸後必須同步檢查滑桿與下方表單是否下移，避免重疊。
- 若畫面仍顯示舊 icon / 舊長方卡，優先確認 Wails 是否重開或重新 build。
- 不要再把第四格命名為人格 D；第四格在預設狀態下是新增入口。
- 不要把 5-16 個人格換行排成多列；必須維持單列水平滑動。
- 不要再新增第二條可見拉桿；下方白色長條是唯一可見拉桿。
- 不要用 `truncate` 截斷側欄按鈕文字；需完整顯示並以字級縮放塞進按鈕框。
- 不要把工具入口按鈕固定成亮綠色；它們必須跟著目前面板風格換色。
- 不要更改工具彈窗背景來修設定框；設定框背景應追隨工具彈窗。
- 面板風格按鈕每行 3 個，文字不得換行；按鈕框寬度需跟著命名長度變長。
- 角色名稱欄不得超過 100 字，且要以中文安全方式截斷。
