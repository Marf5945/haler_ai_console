# Security Fix Decisions — v3.0

審查日期：2026-05-22
決策者：marf
狀態：✅ 全部實作完成，待編譯驗證

---

## 修復優先順序

1. SEC-03、SEC-05、SEC-14（High — 立即修）
2. SEC-13、SEC-16、SEC-15（Medium — 接續修）
3. SEC-19/20（Medium — 一步到位）
4. SEC-17/18（移除 legacy）
5. SEC-21/22（Low — 最後補）

---

## SEC-03 [High] LLM API baseURL SSRF

**決策：混合 C+A**

- ollama / lmstudio 的 providerID 自動允許 `localhost` (http)
- 其他 provider 偵測到 RFC-1918 / link-local / localhost 時，彈前端確認框讓使用者手動放行
- 非私有 IP 一律強制 HTTPS scheme

**實作位置：**
- 新增 `ValidateLLMBaseURL(providerID, baseURL string) error`（純標準庫）
- 在 `RegisterLLMAPIAdapter()` (app.go:2363) 註冊前呼叫
- 利用現有 `llmProviderWhitelist` 比對已知 provider 的合法 BaseURL

**注意事項：**
- app.go:1664 的 ollama/lmstudio 自動填入 localhost URL 路徑不能被擋
- 確認框文案建議：「此 API 端點位於私有網路 (192.168.x.x)，確定要連線？」
- DNS rebinding 暫不處理（需額外 DNS 解析，成本過高）

---

## SEC-05 [High] CLI auth_url 未驗證即開瀏覽器

**決策：分級策略**

- `KnownAuthDomains` 內的 4 個 adapter（gemini/claude/codex/copilot）→ trusted，自動開瀏覽器
- 其他 adapter（含 CustomCLI）→ untrusted，只 emit 事件給前端，前端顯示確認框「即將開啟 xxx.com，是否繼續？」
- CustomCLI 不提供自訂信任 domain 功能（auth 只觸發一次，每次確認的 UX 代價很低）

**實作位置：**
- app.go:1437 的 `if resp.AuthURL != ""` 區塊內加入 `ValidateAuthURL()` 呼叫
- trusted → `go openBrowser(resp.AuthURL)`（維持現有行為）
- untrusted → 只 emit `EventCLIAuthRequired`，前端確認後再呼叫 `openBrowser`

**注意事項：**
- ValidateAuthURL() 已存在於 internal/urlsafe/auth_domains.go，只需接上
- scheme 必須是 https（已在 ValidateAuthURL 中實作）
- 空 AuthURL 不開瀏覽器（已有 if 檢查）

---

## SEC-14 [High] subID 路徑穿越

**決策：binding 層 regex + data 層 ValidatePath（兩層驗證）**

- binding 層 (sub_export_binding.go) 用 regex `^[a-zA-Z0-9_-]+$` 快速拒絕非法 subID
- data 層 (data/subexport/export.go) 用 `storage.ValidatePath()` 做完整邊界檢查
- 兩層職責不同：binding 擋格式、data 擋路徑

**實作位置：**
- sub_export_binding.go:48 ExportSubHandler 入口加 regex 檢查
- export.go:84 PackExport 和 export.go:140 RemoveSubFromSystem 加 ValidatePath
- RemoveSubFromSystem 的 os.RemoveAll 前必須確認路徑在 projectRoot/subagents/callable/ 下

**注意事項：**
- 確認 subID 是 system code（如 sub-1716000000）而非 display name
- 若未來 subID 允許中文，regex 需調整為 filepath.Base() + 禁止 ".."

---

## SEC-13 [High/Latent] docID 路徑穿越

**決策：blobPath() 內部加一行 filepath.Base()**

- 在 document_store.go 的 `blobPath()` 方法內加 `docID = filepath.Base(docID)`
- 一次修復覆蓋 Save / Load / Delete / ExportToTemp 全部路徑
- 不需在每個 Wails binding 入口各自驗證

**實作位置：**
- document_store.go:121 blobPath 方法

**實作程式碼：**
```go
func (s *Store) blobPath(docID string) string {
    docID = filepath.Base(docID) // SEC-13: 防止路徑穿越
    return filepath.Join(s.docsDir, docID+".json")
}
```

**注意事項：**
- 現有 docID 全是 `doc-{UnixNano}` 格式，不含路徑分隔符
- filepath.Base() 對現有資料完全透明，不會破壞向下相容性
- documentStore 目前是延遲初始化，前端利用路徑未完全確認，但應防禦性修復

---

## SEC-16 [Medium] Zip Bomb

**決策：單一 zip entry 上限 500MB**

- zipReadFile 和 docx_read.go 的 io.ReadAll(rc) 改為先檢查 UncompressedSize64，再用 io.LimitReader
- 上限 500MB（使用者指定，涵蓋極大文件場景）
- 不需新套件，純標準庫

**實作位置：**
- builtin/zipxml_util.go:31 zipReadFile
- builtin/docx_read.go:45 ExtractDocxText

**實作範例（zipReadFile）：**
```go
const maxZipEntrySize = 500 * 1024 * 1024 // 500MB

func zipReadFile(zipPath, entryName string) ([]byte, error) {
    r, err := zip.OpenReader(zipPath)
    if err != nil {
        return nil, err
    }
    defer r.Close()
    for _, f := range r.File {
        if f.Name == entryName {
            if f.UncompressedSize64 > maxZipEntrySize {
                return nil, fmt.Errorf("zip entry %s too large: %d bytes", entryName, f.UncompressedSize64)
            }
            rc, err := f.Open()
            if err != nil {
                return nil, err
            }
            defer rc.Close()
            return io.ReadAll(io.LimitReader(rc, maxZipEntrySize))
        }
    }
    return nil, nil
}
```

**注意事項：**
- zipReadFile 是 docx/xlsx/pptx/odt/epub 的共用工具，改一處全覆蓋
- 呼叫端都有 error 檢查，超限時會拋出錯誤給前端

---

## SEC-15 [Medium] 匯出暫存目錄 0755

**決策：加 const 統一管理**

- export.go 頂部加 `const exportDirPerm = 0o700`
- 一次改完所有 6 處 0755
- sub_export_binding.go:140 的 tempRoot 也一起改

**實作位置：**
- data/subexport/export.go:78, 106, 151, 206（4 處）
- sub_export_binding.go:139-140（tempRoot）

**注意事項：**
- macOS Finder 複製時會用目的地的 umask 重設權限，0700 只影響暫存期間
- delegation_log.jsonl 的父目錄（export.go:151）也改 0700

---

## SEC-17/18 [Medium] legacy HTTP 無限制 body

**決策：移除 main.go**

- 確認 legacyhttp build tag 已無使用者後，直接移除 main.go
- 降低維護成本，消除整類安全問題
- Wails 主程式 (wails_main.go) 已有完整的 API response LimitReader

**實作位置：**
- 刪除 main.go（//go:build legacyhttp）

**注意事項：**
- 移除前確認 CI/CD 中沒有使用 `-tags legacyhttp` 的建置流程
- 若有需要保留的 utility function，先遷移到其他檔案

---

## SEC-19/20 [Medium] exec.Command 無 timeout + goroutine 洩漏

**決策：一步到位**

- resolveNpmPrefix() 改用 `exec.CommandContext(ctx, 5s)`
- AutoDetect() 改為接受 `context.Context` 參數
- app.go startup 中使用 `context.WithTimeout` + `sync.WaitGroup` 追蹤 goroutine
- shutdown 時可以 cancel 並等待完成

**實作位置：**
- adapter/adapter_registry/registry.go:436 resolveNpmPrefix
- adapter/adapter_registry/registry.go AutoDetect 方法簽名
- app.go:490 啟動呼叫處
- app.go shutdown/beforeClose 加入 WaitGroup.Wait()

**實作範例（resolveNpmPrefix）：**
```go
func resolveNpmPrefix() string {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    // ... existing npm lookup ...
    out, err := exec.CommandContext(ctx, npmBin, "prefix", "-g").Output()
    if err != nil {
        return ""
    }
    return strings.TrimSpace(string(out))
}
```

**注意事項：**
- timeout 後 AutoDetect 結果為空，前端顯示「偵測未完成，可手動註冊」
- debug log 記錄超時原因
- 需在 App struct 加入 `shutdownWg sync.WaitGroup`

---

## SEC-21/22 [Low] 語音 base64 + 暫存檔名

**決策：標準庫一行修復**

SEC-21：先用 base64 長度估算拒絕超限 payload
```go
maxBase64Len := maxAudioBytes*4/3 + 4
if len(audioBase64) > maxBase64Len {
    return TranscriptResult{}, fmt.Errorf("voice: audio payload too large")
}
```

SEC-22：改用 os.CreateTemp
```go
wavFile, err := os.CreateTemp(workDir, "voice-*.wav")
// 取代 time.Now().Format(...) 命名
```

**實作位置：**
- internal/voice/transcriber.go:40（SEC-21，在 DecodeString 前加檢查）
- internal/voice/transcriber.go:60（SEC-22，改 os.CreateTemp）

**注意事項：**
- os.CreateTemp pattern "voice-*.wav" 保持 voice- prefix，pruneDebugFiles 的 ModTime 邏輯不受影響
- debug JSON 裡已記錄時間，檔名不再需要可讀時間戳

---

# 實作結果 — 2026-05-22

## 修改檔案清單

| 檔案 | SEC | 修改摘要 |
|------|-----|---------|
| `builtin/document_store.go` | SEC-13 | `blobPath()` 加 `filepath.Base(docID)` |
| `internal/urlsafe/validate.go` | SEC-03 | 新增 `ValidateLLMBaseURL()` 函式 |
| `app.go` | SEC-03, SEC-05, SEC-20 | import urlsafe、RegisterLLMAPIAdapter 加驗證、auth_url 加 ValidateAuthURL、startupWg 追蹤 goroutine |
| `internal/urlsafe/auth_domains.go` | — | 無修改（已存在 ValidateAuthURL） |
| `sub_export_binding.go` | SEC-14, SEC-15 | 新增 `validSubID` regex、ExportSubHandler 入口檢查、tempRoot 改 0o700 |
| `data/subexport/export.go` | SEC-14, SEC-15 | PackExport/RemoveSubFromSystem 加 filepath.Base + Rel 驗證、所有 0755→exportDirPerm(0o700) |
| `builtin/zipxml_util.go` | SEC-16 | 新增 `maxZipEntrySize` 常數、zipReadFile 加 UncompressedSize64 檢查 + LimitReader |
| `builtin/docx_read.go` | SEC-16 | ExtractDocxText 加 UncompressedSize64 檢查 + LimitReader |
| `adapter/adapter_registry/registry.go` | SEC-19 | resolveNpmPrefix 改用 `exec.CommandContext` 5s timeout |
| `main.go` → `main.go.removed_sec17_18` | SEC-17/18 | 重新命名移除（legacyhttp build tag 無其他引用） |
| `internal/voice/transcriber.go` | SEC-21, SEC-22 | base64 長度預檢、改用 `os.CreateTemp` |

## 各項修復狀態

| SEC | 嚴重度 | 狀態 | 驗證方式 |
|-----|--------|------|---------|
| SEC-13 | High/Latent | ✅ 完成 | `filepath.Base("../../etc/passwd")` → `"passwd"` 無法穿越 |
| SEC-03 | High | ✅ 完成 | `ValidateLLMBaseURL("generic-api", "http://192.168.1.1")` → needConfirm=true |
| SEC-05 | High | ✅ 完成 | `ValidateAuthURL("custom-cli", "https://evil.com")` → trusted=false |
| SEC-14 | High | ✅ 完成 | `validSubID.MatchString("../../../etc")` → false；data 層 filepath.Base + Rel 雙重驗證 |
| SEC-16 | Medium | ✅ 完成 | UncompressedSize64 > 500MB → 拒絕；LimitReader 雙重防護 |
| SEC-15 | Medium | ✅ 完成 | 6 處 0755 → exportDirPerm(0o700)；tempRoot 也改 0o700 |
| SEC-19 | Medium | ✅ 完成 | `exec.CommandContext` 5 秒 timeout，超時 log 記錄 |
| SEC-20 | Medium | ✅ 完成 | `startupWg.Add(1)` / `defer Done()` 追蹤 AutoDetect goroutine |
| SEC-17/18 | Medium | ✅ 完成 | main.go 重新命名移除，無其他 legacyhttp 引用 |
| SEC-21 | Low | ✅ 完成 | base64 長度 > maxAudioBytes*4/3+4 → 直接拒絕（不分配記憶體） |
| SEC-22 | Low | ✅ 完成 | `os.CreateTemp(workDir, "voice-*.wav")` → 不可預測檔名 |

## 靜態驗證結果

- ✅ 所有檔案括號平衡（braces=0, parens=0）
- ✅ 新增 import 均有實際使用（urlsafe, context, log, regexp, fmt）
- ⏳ 待使用者本機 `go build` / `go vet` 完整編譯驗證

## 下一步

1. 執行 `go build ./...` 確認編譯通過
2. 執行 `go vet ./...` 確認無靜態分析警告
3. 確認 `main.go.removed_sec17_18` 不影響 build（若 CI 正常可直接刪除）
4. 前端需配合 SEC-03 的 `need_confirm` DTO 新增確認框 UI
5. 前端需配合 SEC-05 的 untrusted auth URL 新增確認框 UI

---

# 前端修復步驟 — 2026-05-22 (Phase 2)

## SEC-03 前端：private network 確認框

**問題：** `RegisterLLMAPIAdapter` 現在對 private IP 的 baseURL 回傳 `{need_confirm: true, confirm_type: "private_network", hostname, original_url}`，前端需處理此 DTO。

**修復步驟：**
1. `submitLLMAPISetup()` 收到 result 後檢查 `result.need_confirm`
2. 若為 true，顯示確認對話框：「此 API 端點位於私有網路 (hostname)，確定要連線？」
3. 使用者確認後，附帶 `confirm_private=true` 再次呼叫 `RegisterLLMAPIAdapter`
4. 使用者取消則直接關閉，不註冊

**實作位置：** `frontend/src/App.jsx` 的 `submitLLMAPISetup()` 函式（約 L3771）

**驗證方法：**
- 在 LLM API 設定面板輸入 `http://192.168.1.100:8080` 作為 baseURL
- 應跳出確認框而非直接註冊
- 確認後應成功建立 adapter

---

## SEC-05 前端：untrusted auth URL 確認框

**問題：** Go 端對 untrusted adapter 不再自動開瀏覽器，只 emit `cli:auth_required` 事件。前端需區分 trusted（瀏覽器已開）和 untrusted（需使用者確認開啟）。

**修復步驟：**
1. `cli:auth_required` 事件的 payload 新增 `trusted` 欄位（Go 端已帶）
2. 前端收到事件時：
   - trusted：維持現有行為（顯示「已開啟瀏覽器」對話框）
   - untrusted：顯示「即將開啟 {hostname}，是否繼續？」確認框
3. 使用者確認後用 `BrowserOpenURL(auth_url)` 開瀏覽器
4. 取消則只關閉對話框

**實作位置：**
- Go 端：`app.go` 的 `EventCLIAuthRequired` emit 需帶 `trusted` 欄位
- 前端：`App.jsx` L1108 的 `EventsOn('cli:auth_required')` handler + L4665 的 UI 渲染

**驗證方法：**
- 使用已知 adapter（gemini-cli）觸發授權 → 應自動開瀏覽器
- 使用 CustomCLI adapter 觸發授權 → 應先跳確認框，確認後才開瀏覽器

---

## 清理殘留檔案

1. 刪除 `run_sec_tests.sh`（驗證腳本）
2. 刪除 `main.go.removed_sec17_18`（已確認無引用）
3. 確認 `orchestration/cli_manager/adapter.go:27` 的 `actionchain` import 正常（非本次引入，有使用）

---

---

# Phase 2 實作結果 — 2026-05-23

## 修改檔案清單 (Phase 2)

| 檔案 | SEC | 修改摘要 |
|------|-----|---------|
| `app.go` | SEC-03 | 重構為 `registerLLMAPIAdapterInternal`，新增 `ConfirmRegisterLLMAPIAdapter` binding，event payload 加 `auth_trusted`/`auth_hostname` |
| `frontend/wailsjs/go/main/App.js` | SEC-03 | 新增 `ConfirmRegisterLLMAPIAdapter` JS binding |
| `frontend/wailsjs/go/main/App.d.ts` | SEC-03 | 新增 TypeScript 宣告 |
| `frontend/src/App.jsx` | SEC-03, SEC-05 | `submitLLMAPISetup` 處理 `need_confirm` DTO + 確認框；auth 事件區分 trusted/untrusted，按鈕動態切換「開啟並授權」/「已完成授權」 |

## 清理檔案

| 動作 | 檔案 |
|------|------|
| 已刪除 | `run_sec_tests.sh` |
| 已刪除 | `main.go.removed_sec17_18` |

## Phase 2 驗證結果

- ✅ 所有 Go 檔案括號平衡
- ✅ 前端 JSX 括號平衡
- ⏳ 待使用者 `go build ./...` + `go vet ./...` 編譯驗證
- ⏳ 待使用者 `cd frontend && npm run build` 前端編譯驗證

## SEC-03 前端確認流程

```
使用者輸入 private IP baseURL
    → RegisterLLMAPIAdapter 回傳 {need_confirm: true, hostname}
    → 前端彈 window.confirm("此 API 端點位於私有網路...")
    → 確認 → ConfirmRegisterLLMAPIAdapter（跳過 SSRF 檢查）
    → 取消 → 顯示「已取消私有網路連線」
```

## SEC-05 前端確認流程

```
CLI 觸發 auth
    → Go 端 ValidateAuthURL 判斷 trusted/untrusted
    → trusted: Go 自動開瀏覽器 + emit {auth_trusted: "true"}
       → 前端顯示「已開啟授權頁面」+ 按鈕「已完成授權」
    → untrusted: Go 不開瀏覽器 + emit {auth_trusted: "false", auth_hostname}
       → 前端顯示警告色提示 + 按鈕「開啟並授權」
       → 使用者點擊 → BrowserOpenURL(auth_url) → 按鈕變「已完成授權」
       → 再次點擊 → 重試 CLI 訊息
```
