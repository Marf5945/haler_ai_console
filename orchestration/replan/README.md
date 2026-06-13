# orchestration/replan — Bounded Replan v1

讓 agent 在執行中觀察工具結果，**只在 low-risk 且不改變目標的前提下**自動改寫尚未開始的後續節點（tail），連續無進展達上限即停下交人。核心原則：

> **LLM 只「提案」新 tail；Go 才「裁決」silent / review / stop。**
> 硬天花板編譯在 Go：read-only allowlist + classifier 否決 + GoalContract 不變。

零第三方依賴：僅 stdlib + 內部 `dag` / `domain/risk` / `audit_log`。

## 檔案地圖

| 檔 | 責任 |
|---|---|
| `types.go` | 列舉（Decision/Stage/FailureCategory/Intent）、read-only allowlist、全域 deny、提案型別 |
| `contract.go` | `GoalContract` 行為（同目標 / 輸出驗證）；資料型別在 `dag.GoalContract` |
| `failure.go` | `ClassifyFailure`：執行層失敗 → 結構化 `FailureCategory` |
| `policy.go` | `Gate`：硬規則裁決（唯一裁決入口） |
| `patch.go` | `TailPatch` + 三重 CAS + tail hash + 震盪 signature + `ApplyTailPatch` |
| `counter.go` | 連續無進展 + 總閘 + 震盪加速 |
| `coordinator.go` | `Coordinator`：提案→Gate→CAS 套用→audit；`Proposer` / `Critic` 介面 |
| `ux.go` | `ActivitySummary`：status rail 一句活動摘要 |
| `audit.go` | `ReplanAuditEntry` + `SafeSummary`（遮蔽路徑/token） |
| `hash.go` | sha256 helper |

main 端接線：`replan_binding.go`（真 `plannerProposer` + flag + `tryReplanOnFailure`）、`task_progress_binding.go`（`executeTaskNode` 觸發）。

## 安全不變量（違反任一即 bug）

1. silent 路徑永遠只執行全 read-only-eligible 的 tail。
2. 新節點風險由 `risk.ClassifyOperation` 裁定，不信模型自報。
3. 同目標換路可 silent；改變目標/產出一律 review。
4. 只動 `planned`/`ready` tail；不碰 succeeded/running/waiting_review/blocked/failed。
5. silent 對使用者，不 silent 對 log（每次 replan 都寫 `audit_log/replan.jsonl`）。
6. Critic 只能收緊（silent→review），不能放寬。
7. proposer 錯誤 / 逾時 / CAS 衝突 → 一律 fail-safe 進 review，不套用。

## 啟用（預設關）

活線觸發由環境變數控制，**預設關閉，行為與接線前完全一致**：

```bash
UICONSOLE_BOUNDED_REPLAN=1   # 啟用；未設 / 其他值 = 關
```

開啟後：只有 low-risk read-only 步驟失敗會嘗試自動換路；medium+ / scope 變更 / 連續無進展達上限一律 review/stop。第一次啟用務必盯 `audit_log/replan.jsonl`。

## 驗收

```bash
env GOCACHE=/private/tmp/go-build-cache go build ./...
env GOCACHE=/private/tmp/go-build-cache go test ./orchestration/replan/... ./orchestration/dag/... -v
```

計數常數（`counter.go`）：`MaxConsecutiveNoProgress=5`、`MaxRunTotal=8`、`OscillationPenalty=2`。

## 尚未完成（非阻塞）

- 前端讀 `DAGRun.ReplanActivity`（隨 `task:progress_replanned` event）顯示活動摘要。
- 真 LLM Critic adapter（可選；`Coordinator.Critic` 預設 nil）。
- **活線（flag 開）的實機 runtime 驗證**：build 綠只代表編譯過，活線行為需跑真任務確認。
- `GoalContract.OutputPredicate` / `Scope` 的更精準推導（目前 v1 從 plan 只抓 goal summary）。

後續工作以程式碼註解、測試與 issue/PR 追蹤為準。
