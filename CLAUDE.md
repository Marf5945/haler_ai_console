# CLAUDE.md

AI Console — Wails(Go)+ React 桌面 AI 助手。本檔供 AI 助手快速理解專案,修改程式前先讀這裡,不要全 repo 探索。

## 架構地圖

- `app.go` — Wails App 主體(~8k 行,**用 grep 定位函式名再讀區段,不要整檔讀**);其餘 `app_*.go` 與 `*_binding.go` 為同 package 拆檔
- `orchestration/` — 任務編排:cli_manager(LLM adapter)、dag(任務排程)、spec_patch_checker
- `adapter/` — 外部介接:visual_learning、adapter_registry、remote_bridge
- `domain/` — 業務規則:controlled_trust、review
- `data/` — 持久層:conversation(對話歷史/摘要)、memory、storage
- `shared/`、`internal/`、`builtin/`、`tools/` — 共用工具與內建功能
- `frontend/src/App.jsx` — React 主元件(6k 行,持續拆細中)
- `frontend/src/lib/appShared.jsx` — App 拆出的共用 helpers/常數
- `frontend/src/components/{layout,chat,tools,settings,onboarding,modals}/` — 拆出的 UI 元件
- `frontend/wailsjs/` — Wails 自動生成 binding,**不要手改**

## 常用指令

- Go 建置/測試:`go build ./...`、`go test ./...`(Windows 上用 workspace `.gocache` 避開 AppData 權限問題)
- 前端:`cd frontend && npm run build`(含 tailwind 重建)、`npm test`(vitest)
- 整體驗證:`./verify.sh`;Windows 打包:`build.cmd`

## 專案約定

- app.go 拆檔走同 package 多檔模式(`app_<域>.go`),不需改 import
- 前端元件:一檔一元件、default export;共用邏輯進 `src/lib/`
- 測試 `frontend/src/test/i906-build-binding.test.js` 會串接 App.jsx + lib + components 檢查 Wails binding import,拆檔不需改該測試
- 外部連結一律走 `openExternal()`(Go 端 `OpenExternalURL` 檢查),不用 `<a target="_blank">`
- 註解與 commit 訊息使用繁體中文

## 地雷警告

- **不要讀** `TASKS_2_6_1.md`(324KB 歷史任務紀錄)
- **不要掃** `build/`、`frontend/dist/`、`assets/runtimes/`、`.gocache*`、`reference/`(合計 ~158MB 二進位/產物)
- `frontend/src/tailwind.css`(296KB)是生成物;改樣式去 `style.css` 或元件
- token 優化現況與已知熱點見 `TOKEN_OPTIMIZATION_REPORT.md`(v2,經兩輪稽核)
