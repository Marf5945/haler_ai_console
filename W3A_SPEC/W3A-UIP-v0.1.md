# W3A-UIP — W3A UI Plan

**Spec 版本：** v0.1 (draft)
**配套：** [`W3A-SPEC-v0.1.md`](./W3A-SPEC-v0.1.md)
**一句定義：** UI plan 是 `w3a compile` 的唯一輸出——一份**語言中立、平台中立、無安全意義**的顯示中介格式。真正要跨 agent 的不是 markdown 原文、也不是某個視窗，而是這份 IR。

---

## 0. 角色定位（最重要）

```
W3ACTIONS.md + W3ALAYOUT.md  →  w3a compile  →  UI Plan  →  renderer  →  介面
```

- UI plan **不攜帶機密**，可自由傳遞（記錄、快取、跨進程傳送）。
- UI plan **不繼承信任**：renderer 拿到它不代表可以略過任何安全控制。
- 所有安全決策（打哪個 origin、要不要確認、能不能當 HTML）一律由 core 回查**已驗章的** `W3ACTIONS.md`（見 W3A-SPEC §6 S1–S8）。UI plan 只描述「長相與互動意圖」。

> 因此 UI plan 故意**不含** `backend_origin`、`target`、`method`、`mutates`、token 等安全相關欄位。它只給 `action_id`。

---

## 1. UI Plan 結構

```json
{
  "ui_plan_version": "0.1",
  "app_id": "com.example.board",
  "app_version": 1,
  "source_digest": "sha256:...",
  "blocks": [
    {
      "id": "main_board",
      "type": "message_board",
      "entity": "message",
      "entity_fields": ["id", "author", "text", "created_at"],
      "data_action": "read_messages",
      "actions": [
        { "action_id": "submit_message", "verb": "submit", "mutates": true,  "confirm": true,  "input": { "text": "string" } },
        { "action_id": "comment_message","verb": "comment","mutates": true,  "confirm": true,  "input": { "text": "string" } },
        { "action_id": "search_messages","verb": "search", "mutates": false, "confirm": false, "input": { "q": "string" } }
      ],
      "fallback": ["list", "text"],
      "pagination": { "limit": 50, "cursor": true },
      "sort": "created_at desc"
    }
  ],
  "presentation": {
    "theme_color": "blue",
    "font_scale": 2,
    "density": "comfortable"
  },
  "applied": {
    "readonly": false,
    "hidden_actions": ["comment_message"],
    "hidden_blocks": []
  }
}
```

### 1.1 欄位語意

| 欄位 | 意義 | 備註 |
|------|------|------|
| `ui_plan_version` | UIP 規格版本 | renderer 用來判相容性 |
| `app_id` / `app_version` | 對應來源契約身分 | 確認「同一個 app」 |
| `source_digest` | 來源契約的正規化雜湊 | 供稽核追溯，**非**信任憑證 |
| `blocks[]` | 要顯示的區塊（偏好套用後的結果） | 順序即建議顯示順序 |
| `block.type` | 核心或 `x-` 區塊型別 | renderer 不支援時走 `fallback` |
| `block.entity` / `entity_fields` | 此區塊呈現的資料形狀 | 讓不同 renderer 渲染一致欄位 |
| `block.data_action` | 取資料用的 `read`/`search` action id | renderer 透過 core 呼叫 |
| `block.actions[]` | 此區塊可觸發的動作 | 只給 `action_id` + 顯示所需最小資訊 |
| `action.mutates` / `confirm` | **僅供 UI 提示**（例如顯示確認樣式） | **不具強制力**；真正強制在 core（S2） |
| `block.fallback` | 降級鏈 | 見 §3 |
| `presentation` | 純視覺偏好套用結果 | theme/font/density |
| `applied` | 偏好/權限套用後的狀態快照 | readonly / 隱藏項 |

> `mutates`/`confirm` 出現在 UI plan 只是讓 renderer 能畫出「這顆按鈕會改資料、會跳確認」的提示。即使被竄改成 `confirm:false`，core 在 operate 時仍回查簽章源強制確認（S1+S2），所以無法藉此繞過。

---

## 2. Renderer Capability 描述

renderer 在接 UI plan 前，向 core 宣告自己支援什麼：

```json
{
  "renderer": "renderer-text",
  "supports": {
    "blocks": ["list", "search_box", "composer"],
    "widgets": ["button", "textarea", "list", "text", "modal_confirm"]
  }
}
```

- core / compile 依此把不支援的 block 型別**事先降級**，或 renderer 在本地依 `fallback` 鏈降級。
- 若 renderer 連 `modal_confirm` 都不支援，**不得**渲染任何 `mutates:true` 動作（core 也會拒絕對應 operate），只能呈現唯讀版。

---

## 3. 降級演算法（fallback）

每個 block 帶一條 `fallback` 鏈，例：`message_board → list → text`。

```
function resolveBlock(block, caps):
    candidates = [block.type] + block.fallback
    for t in candidates:
        if t in caps.supports.blocks:
            return renderAs(block, t)
    return renderAs(block, "text")   # 最終保底，所有 renderer 必支援 "text"
```

規則：
1. `text` 是**強制保底**型別，任何 renderer 都必須支援。
2. 降級只能往「能力更低」走，不得往上臆造。
3. 降級不得犧牲安全：降成 `list`/`text` 後，mutating 動作仍受 S2 確認約束；無 `modal_confirm` 能力時這些動作呈現為唯讀。

> 這讓 Codex、瀏覽器、Wails、終端機、聊天列都能接**同一份** UI plan，而不是每個平台重新發明一次。

---

## 4. Operate 綁定（執行期）

`compile` 是靜態的；操作是執行期。renderer 把 UI 事件接到 core 的 `operate`：

```
renderer → core.operate({
    app_id: "com.example.board",
    action_id: "submit_message",
    input: { text: "..." },
    confirmation_token?: "..."   // 由 core 發出的確認流程取得
})
```

core 收到後：
1. 以 `app_id` + `action_id` 回查**已驗章**的 `W3ACTIONS.md`。
2. 取得真實 `target` / `method` / `mutates` / `confirm` / `requires`。
3. 套用 S2–S4：確認強制、origin pinning、token 注入。
4. 執行並回傳 `output`（依 entity schema），renderer 純文字渲染（S5）。

renderer **永遠不**自行組 HTTP 請求、不持有 token、不決定是否確認。

---

## 5. 渲染安全要求（renderer 必遵）

| # | 要求 | 對應 SPEC |
|---|------|-----------|
| R1 | entity 欄位值一律當**不可信純文字**渲染，不得當 HTML / markup / script | S5 |
| R2 | 凡 `mutates:true` 動作，互動時必須觸發 core 的確認流程；不得自行宣稱已確認 | S2 |
| R3 | 不得自行向任何 URL 發請求；所有後端互動經 `core.operate` | S1, S3 |
| R4 | 不得讀取或快取 token / 憑證 | S4 |
| R5 | 不支援 `modal_confirm` 時，不得渲染 mutating 動作（呈現唯讀） | S2 |
| R6 | `applied.readonly` / `hidden_actions` / `hidden_blocks` 必須被尊重 | §1 |

> renderer 不可信，因此 R1–R6 是「renderer 行為測試」的必過項；core 同時在 operate 端獨立把關，即使 renderer 違規也擋得住。

---

## 6. Renderer Conformance（行為測試）

給定固定 UI plan + 固定互動序列，驗證行為等價，而非像素一致：

1. **渲染完整性：** 所有 `blocks` 與 `actions` 皆呈現（除非被 `applied` 隱藏）。
2. **降級正確：** 不支援的 block 依 `fallback` 鏈降到正確型別，最終保底 `text`。
3. **確認觸發：** 觸發任一 `mutates:true` 動作必經確認流程（R2）。
4. **純文字渲染：** 注入 `<script>` 的 entity 值不得被當成 markup（R1）。
5. **唯讀尊重：** `applied.readonly=true` 時所有 mutating 動作不可達（R6）。
6. **無越權請求：** renderer 不對非 `core.operate` 的端點發任何請求（R3）。

v0.1 三個 reference renderer（`text` / `web` / `wails-go`）皆須通過上述全部。

---

## 7. 版本相容

- `ui_plan_version` 為 minor 演進；renderer 遇到不認得的**較新** minor 應盡力渲染已知欄位、忽略未知欄位，但不得忽略安全相關行為（R1–R6 永遠生效）。
- 遇到**較新 major** 或無法安全渲染 → 拒絕並回報，不得猜測。

---

*W3A-UIP v0.1 draft — UI plan 只負責長相；安全永遠回源。*
