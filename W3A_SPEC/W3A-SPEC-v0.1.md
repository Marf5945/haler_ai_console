# W3A — Web AI Agent Agreement

**Spec 版本：** v0.1 (draft)
**狀態：** Working Draft，行為穩定後才凍結
**一句定義：** W3A 是「同源後端的可攜式介面契約」——同一份資料、同一組功能入口、同一個 app 身分，但不同使用者可以依偏好長出不同的 UI 殼。

> 配套規格：UI plan 中介格式定義在 [`W3A-UIP-v0.1.md`](./W3A-UIP-v0.1.md)。

---

## 0. 設計原則（不可動搖）

1. **協議語言中立。** W3A 不要求任何特定程式語言。本專案提供 Go reference implementation，其他人可用 JS / Python / Rust 實作，但必須通過同一批 test vectors / golden files。統一「行為」，不統一「程式語言」。
2. **agent 自由發揮 UI 表現，但不能自由發揮資料契約、動作語意、權限與信任。**
3. **core 可信，renderer 不可信。** 所有安全控制在 core 強制，永不下放到 renderer。
4. **W3A 不是「讓高手寫 markdown」。** 它是「使用者用 wizard 填需求 → 系統產生可簽章、可驗證、可編譯的 markdown 契約」。
5. **人看起來是 markdown，agent 讀起來是半結構化契約。** 不是全自由文字，也不是難讀的 JSON。

---

## 1. 架構與責任邊界

```
W3ACTIONS.md + W3ALAYOUT.md      ← 可簽章的來源契約（信任根）
        ↓  w3a compile
UI Plan / W3A-UIP                 ← 語言中立、無安全意義的顯示中介
        ↓  renderer
Web / Wails / Codex / TUI / Chat UI
```

| 層 | 角色 | 信任 |
|----|------|------|
| **w3a-core** | parse / lint / compile / sign / verify / inspect / operate | **可信**：強制所有安全控制 |
| **UI plan (W3A-UIP)** | 顯示用中介格式 | **無安全意義**：可自由傳遞，不得攜帶機密 |
| **renderer** | 把 UI plan 畫成介面、把使用者互動回報給 core | **不可信** |

renderer 唯一能對 core 說的話是：

```
使用者觸發 action_id=submit_message，input={ text: "..." }
```

renderer **不能**決定：要 POST 到哪、這個動作要不要確認、這段內容能不能當 HTML 顯示。這些一律由 core 回查**已驗章的** `W3ACTIONS.md` 決定。

---

## 2. 六個核心概念

| 概念 | 定義 | 落在哪份檔 |
|------|------|-----------|
| **Entity** | 資料長什麼樣（欄位與型別） | W3ACTIONS.md |
| **Action** | 能做什麼（動詞、入口、輸入輸出、是否改資料、是否確認） | W3ACTIONS.md |
| **Block** | 介面上有什麼區塊（型別、綁哪個 entity / action） | W3ALAYOUT.md |
| **Policy** | 誰能做什麼（requires / fallback；伺服器端權限） | W3ACTIONS.md（動作）/ W3ALAYOUT.md（區塊） |
| **Preference** | 每個人想怎麼看（顏色、字級、密度、隱藏、唯讀） | W3ALAYOUT.md |
| **Renderer** | 怎麼把 UI plan 畫出來 | 不在協議內；由 reference renderer 示範 |

**Policy 與 Preference 必須分開：** Preference 是「使用者想怎麼看」，Policy 是「伺服器准不准」。兩者不可混用同一機制。

---

## 3. 固定詞彙表（core）

擴充允許，但核心詞彙固定，否則 agent 會亂猜。

### 3.1 動作動詞 `verb`

核心：`read` `submit` `comment` `search` `react` `update` `delete`

### 3.2 區塊型別 `type`

核心：`message_board` `feed` `search_box` `composer` `list` `detail_view` `profile_panel` `notification_bar`

### 3.3 偏好鍵 `preference`

核心：`readonly` `hide_actions` `theme_color` `font_scale` `density` `visible_blocks`

### 3.4 擴充命名空間

所有非核心項目一律加 `x-` 前綴（例：`x-vote`、`x-bookmark`），或在檔頭 `extensions:` 列出。core 名稱永遠保留給固定那組。未宣告的非核心名稱 → lint 失敗。

---

## 4. W3ACTIONS.md — 操作契約

承載：身分、信任、後端、entities、actions、input/output、權限。

```markdown
# W3ACTIONS.md

app_id: com.example.board
version: 1
backend_origin: https://api.example.com
data_scope: shared            # shared | local | drive | api

publisher:
  id: com.example.publisher
  name: Example Publisher
  public_key: ed25519:BASE64...
  signed_at: 2026-06-24T10:00:00Z
signature:
  alg: ed25519
  covers: canonical_document_sha256
  value: BASE64...

## entities
entities:
  message:
    fields:
      id: string
      author: string
      text: string
      created_at: datetime

## action: read_messages
verb: read
target: /messages
method: GET
mutates: false
output:
  type: list
  entity: message
pagination:
  limit: 50
  cursor: optional
sort: created_at desc

## action: submit_message
verb: submit
target: /messages
method: POST
mutates: true
confirm: required
input:
  text: string
output:
  entity: message
requires:
  auth: required
  role: member
fallback:
  unauthenticated: readonly
  unauthorized: hide

## action: comment_message
verb: comment
target: /messages/{id}/comments
method: POST
mutates: true
confirm: required
input:
  text: string
output:
  entity: message

## action: search_messages
verb: search
target: /messages/search
method: GET
mutates: false
input:
  q: string
output:
  type: list
  entity: message
```

### 4.1 必填欄位
- 身分：`app_id`、`version`、`backend_origin`、`data_scope`
- 信任：`publisher`（含 `public_key`、`signed_at`）、`signature`
- 每個 action：`verb`、`target`、`method`、`mutates`
- 改資料的 action（`mutates: true`）：`confirm`（預設 `required`）
- 有形狀的輸入/輸出：`input` / `output`（避免 agent 猜表單欄位）

---

## 5. W3ALAYOUT.md — 介面契約

承載：區塊、型別、綁定、偏好、renderer fallback。

```markdown
# W3ALAYOUT.md

app_id: com.example.board
version: 1

publisher: { ...同 W3ACTIONS... }
signature: { ...同 W3ACTIONS... }

## block: main_board
type: message_board
entity: message
data: read_messages
actions: submit_message, comment_message, search_messages
fallback: list, text          # renderer 不支援 message_board 時的降級鏈

## preferences
readonly:
  disables: submit_message, comment_message
hide_actions:
  allowed_values: comment_message, submit_message, search_messages
theme_color:
  allowed_values: blue, green, neutral
font_scale:
  min: 1
  max: 2
density:
  allowed_values: compact, comfortable
visible_blocks:
  allowed_values: main_board
```

### 5.1 偏好如何套用
- `readonly` → 停用所列的 mutating actions（殼層唯讀，但資料仍共用）
- `hide_actions` → 從殼上拿掉指定動作（佔位移除，不是 disable）
- `theme_color` / `font_scale` / `density` → 純視覺
- `visible_blocks` → 使用者選擇顯示哪些區塊

> 範例對應：使用者 A「不要評論、藍色、字大兩倍」＝ `hide_actions: comment_message` + `theme_color: blue` + `font_scale: 2`。使用者 B「只讀」＝ `readonly: true`。兩者皆讀寫**同一份** `data_scope: shared` 的資料。

---

## 6. 安全模型（core 強制，不可下放）

| # | 規則 | 為何 |
|---|------|------|
| S1 | **operate 永遠回查已驗章的 W3ACTIONS.md** 取得 origin / path / mutates / confirm。renderer 只回報 `action_id` + `input`，不得提供 URL 或 mutates 旗標。 | 竄改過的 UI plan 改不掉動作語意，redirect 不了後端 |
| S2 | **寫入確認由 core 強制。** `mutates:true, confirm:required` 沒有明確確認 token，core 直接拒絕執行。 | 惡意/有 bug 的 renderer 不能跳過 modal |
| S3 | **Origin pinning + path 收斂。** operate 只准打 `backend_origin`；`target` 必為相對路徑，禁止絕對 URL、禁止 `..` 逃逸，強制 TLS。 | 防 backend_origin 被換、防 SSRF / 資料外流 |
| S4 | **token 只在 core。** 認證憑證由 core 注入並保管，scope 綁 `app_id` + origin；renderer 永不取得原始 token；一份 W3A 不能拿到另一個 app/origin 的 token。 | 防憑證外洩與越權 |
| S5 | **資料顯示一律純文字。** renderer 顯示 entity 欄位值時必須當不可信純文字，不得當 HTML / markup / 可執行碼。 | 防共用寫入造成的 stored XSS |
| S6 | **擴充預設保守。** 未知 `x-` verb：不得逃 origin；若可能改資料 → 強制 confirm；core 不認得 → 標 untrusted。預設姿態：未知擴充 = 降到唯讀或拒絕（由發布者於 `extensions:` 明示）。 | 限制擴充攻擊面 |
| S7 | **資源上限。** compile 端擋「layout 炸彈」（區塊數上限）；operate 端擋超大 payload；`read` 強制 `pagination.limit`。 | 防 DoS |
| S8 | **簽章覆蓋整份正規化文件。** `signature.covers = canonical_document_sha256`。lint 第一步即驗章；驗章失敗 → 全部拒絕。**UI plan 不繼承信任**：它無安全意義，operate 永遠回源（見 S1）。 | 信任根明確；UI plan 可自由傳遞 |

> 簡記：**core = 信任 + 驗證 + 強制安全控制；UI plan = 無安全意義的顯示中介；renderer = 不可信、只負責長相。**

---

## 7. 信任與金鑰

- **演算法：** Ed25519（與本專案既有 `adapter/w3a_media/app_fingerprint.go` 的金鑰脈絡一致）。
- **覆蓋範圍：** 對兩份檔各自的正規化內容（canonical form）計算 SHA-256，再簽。
- **Bootstrap（v0.1）：** TOFU（Trust On First Use）——首次取得即 pin 住 `publisher.public_key`；之後同 publisher 換鑰必須走撤銷流程。
- **撤銷：** 預留 `revoked_at` 欄位與撤銷清單位置（`/.well-known/w3a/revocations.json`）。
- **template 不構成信任層：** template 是未簽章骨架，簽章只在發布步驟用**使用者自己的**發布者金鑰完成。

---

## 8. Discovery（如何被找到）

兩種模式並存：

1. **Well-known：** `https://{origin}/.well-known/w3a/`（含 `W3ACTIONS.md`、`W3ALAYOUT.md`、`revocations.json`）。
2. **手動可信連結：** 使用者從可信來源直接拿到檔案或連結——涵蓋「不架網站」的情境，例如把契約放在 Google Drive 檔 / Apps Script 端點。

> 「可信」由簽章 + TOFU 保證；「discovery」只負責找到檔案。兩者不同層。

---

## 9. 工具層（reference CLI：`w3a`）

| 命令 | 職責 | 自動/需確認 |
|------|------|-------------|
| `w3a init` | 互動式 wizard 建立兩份檔（8 步，見 §10） | — |
| `w3a lint` | 驗證格式、版本、schema、權限、簽章、安全規則 | 自動 |
| `w3a compile` | `W3ACTIONS.md + W3ALAYOUT.md → UI plan`（不直接畫 UI） | 自動 |
| `w3a preview` | 用 reference renderer 預覽；**走 mock / 唯讀沙盒**，絕不打真實後端 | 自動（沙盒） |
| `w3a sign` | 對文件簽章 | 需金鑰 |
| `w3a verify` | 驗證 publisher / signature | 自動 |
| `w3a inspect` | 顯示 actions / blocks / entities / permissions | 自動 |
| `w3a operate` | 經確認後呼叫後端動作 | **寫入需確認**（S2） |
| `w3a publish` | 放到 `.well-known` 或指定位置 | — |

`compile` 只輸出 UI plan。`read`/`search`/渲染可自動；`submit`/`comment`/`delete`/`publish` 等改資料動作預設需確認。

---

## 10. 建立流程（8 步 wizard，使用者不需懂底層）

使用者先看到 **template 選單**，選完才一步步問，不要一開始就面對 `input/output schema`：

```
你想建立哪種應用？
[留言板] [任務清單] [文件搜尋] [回饋表單] [問題追蹤] [名冊]
```

| 步 | 系統做的事 | 產出 |
|----|-----------|------|
| 1 | 填 App 基本資料：名稱、`app_id`、`version`、發布者；**建立或選擇發布者 Ed25519 金鑰**（決定存放與備份） | 身分 + 信任根 |
| 2 | 選資料來源：共用後端 / 本機 / Drive / API | `backend_origin`、`data_scope` |
| 3 | 定義資料實體（demo：`message` = id/author/text/created_at），並解釋給使用者 | `entities` |
| 4 | 先問要幹嘛再給建議：讀取/送出/評論/搜尋/更新/刪除（不用寫 API） | `actions` + input/output |
| 5 | 建議區塊，使用者可增減；每個 block 綁 entity + action | `blocks` |
| 6 | 設定權限：哪些需登入、哪些唯讀、哪些角色可寫（明確區分 Policy） | `requires` / `fallback` |
| 7 | 設定偏好規則：字級/顏色/密度/隱藏/唯讀（每人自己的殼） | `preferences` |
| 8 | `lint → compile → preview → sign → publish`；發布後 agent 只吃簽過的檔 | 已簽章契約 |

> 金鑰步驟（步 1）是 8 步流程的關鍵；缺它使用者會卡在步 8 無法簽章。

---

## 11. Reference renderers（示範，非協議核心）

v0.1 至少附三個，用來**證明跨平台**，不規定大家必用：

| 名稱 | 角色 |
|------|------|
| `renderer-text` | 最低能力保底：純文字清單 + 按鈕描述 + 確認提示。可攜性底線。 |
| `renderer-web` | 最易展示與測試：同一份 UI plan → 瀏覽器介面。 |
| `renderer-wails-go` | 對本專案最實用的原生視窗參考實作；**不得**成為 W3A 唯一標準。 |

降級規則由 UI plan 的 `fallback` 鏈與 renderer capability 決定（見 W3A-UIP）。例：`message_board → list → text`。

---

## 12. Conformance（一致性測試）

兩層分開驗：

1. **Compiler 層（golden files）：** 同一份 `W3ACTIONS.md + W3ALAYOUT.md` → 同一份 UI plan。任何語言的實作都必須通過同一批 test vectors。
2. **Renderer 層（行為測試）：** 同一份 UI plan + 同樣互動 → 行為等價（含 fallback 降級正確、寫入確認觸發、entity 純文字渲染）。屬行為/快照測試，非純 golden file。

安全規則（§6）屬於 compiler/operate 的**必過**測項，不是建議。

---

## 13. 套件命名

```
w3a-core            # parser / lint / compile / sign / verify / inspect / operate
w3a-uip             # UI Plan 規格（見 W3A-UIP-v0.1.md）
renderer-text       # reference renderer
renderer-web        # reference renderer
renderer-wails-go   # reference renderer
```

---

## 14. v0.1 範圍與 demo

**留言板 demo 作為 v0.1 驗收標的**，因為它剛好測到所有核心能力：共用資料、寫入、搜尋、權限、偏好、renderer 降級、簽章信任。

v0.1 凍結前必須成立：
- [ ] 兩份檔格式 + 簽章可被 `lint` / `verify` 通過
- [ ] `compile` 對 demo 產出穩定 UI plan（golden file）
- [ ] `renderer-text` 能渲染並正確降級
- [ ] 安全規則 S1–S8 全數有對應測試
- [ ] 8 步 wizard 能從 template 產出可簽章契約

---

*W3A v0.1 draft — 行為先於語言；信任先於便利。*
