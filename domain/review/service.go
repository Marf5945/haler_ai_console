// Package review 提供統一的 Review Card 服務，透過 Wails 綁定暴露給前端。
// v3.6.1 重構：支援完整 Standard Card、Lightweight Card、雙步驟確認。
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ TASKS_1_6_3 持久化升級（v4.1）                                     │
// │                                                                     │
// │ 背景：原先 Service.cards 是純 in-memory slice，app restart 後      │
// │ 所有 open review cards 會消失。這對 visual learning 等需要使用者    │
// │ 待處理決策的場景不可接受 — 候選資料還在，但審核入口不見。          │
// │                                                                     │
// │ 改動摘要：                                                         │
// │  1. NewServiceWithDataRoot 啟動時從 review_inbox.json 載入         │
// │     未解決的 open cards（file-backed cold start）。                  │
// │  2. AddCard 寫入後同步 flush 到 review_inbox.json。                │
// │  3. Resolve 解決後：                                               │
// │     a. 從 review_inbox.json 移除                                   │
// │     b. 寫入 review_archive.json（透過 ArchiveService）             │
// │     c. 寫入 review_decision_log.jsonl（原有邏輯不變）              │
// │  4. InvalidateAll 同步 flush inbox。                               │
// │                                                                     │
// │ 檔案格式：review_inbox.json 是 JSON array，與 visual_learning 的   │
// │ visual_review_inbox.json 格式一致，方便 migration。                 │
// │                                                                     │
// │ 不變的部分：                                                       │
// │  - NewService()（無 dataRoot）仍為純 in-memory，供測試使用         │
// │  - Lightweight cards 暫不持久化（短命 + 低風險）                    │
// │  - 決策日誌 §5.4 邏輯完全不變                                      │
// └─────────────────────────────────────────────────────────────────────┘
//
// Spec reference: AI_Console_Spec_v3_6_1 §5
package review

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"ui_console/audit_log"
	"ui_console/domain/risk"
)

// ──────────────────────────────────────────────
// 舊版 Level 定義（向下相容）
// ──────────────────────────────────────────────

// Level 是舊版三級分類（保留供既有程式碼相容）。
type Level string

const (
	LevelBlocking   Level = "blocking_review"
	LevelPending    Level = "pending_review"
	LevelBackground Level = "background"
)

// ──────────────────────────────────────────────
// Service 主體
// ──────────────────────────────────────────────

// Service 管理所有 Review Card（Standard + Lightweight）。
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ 持久化欄位說明：                                                   │
// │  inboxPath   — review/review_inbox.json，存放所有 open cards       │
// │  archive     — ArchiveService，管理 review_archive.json            │
// │  decisionLog — review_decision_log.jsonl（append-only §5.4）       │
// │  securityLog — security_change_log.jsonl（security_boundary 專用） │
// │                                                                     │
// │ 當 dataRoot == ""（NewService() 建立）時，不做任何檔案 I/O，       │
// │ 所有持久化方法為 no-op，適合單元測試。                              │
// └─────────────────────────────────────────────────────────────────────┘
type Service struct {
	mu               sync.Mutex
	cards            []Card
	lightweightCards []LightweightCard
	dataRoot         string                                 // 專案資料根目錄，用於寫入 log
	inboxPath        string                                 // review/review_inbox.json 完整路徑
	archive          *ArchiveService                        // 管理 review_archive.json
	decisionLog      *audit_log.AppendLog[decisionLogEntry] // append-only 決策日誌
	securityLog      *audit_log.AppendLog[decisionLogEntry] // security_boundary_rewrite 額外日誌
}

// NewService 建立 review service。dataRoot 為專案資料根目錄。
func NewService() *Service {
	return &Service{
		cards:            []Card{},
		lightweightCards: []LightweightCard{},
	}
}

// NewServiceWithDataRoot 建立含資料路徑的 review service（支援 log 寫入 + 持久化）。
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ 啟動流程（TASKS_1_6_3 持久化）：                                   │
// │  1. 設定 inboxPath = {dataRoot}/review/review_inbox.json           │
// │  2. 建立 ArchiveService（管理 review_archive.json）                │
// │  3. 從 review_inbox.json 載入上次未解決的 open cards               │
// │  4. 建立 decision log / security log（原有邏輯不變）               │
// │                                                                     │
// │ 如果 inbox 檔案不存在或損毀，視為空清單（不 crash）。              │
// └─────────────────────────────────────────────────────────────────────┘
func NewServiceWithDataRoot(dataRoot string) *Service {
	// SEC-W07（2026-05-24）：拿掉 perm override，回到 audit_log framework 預設
	// （file 0o600 / dir 0o700）。review decision log 為私有稽核資料，無外部讀取需求。
	var logOpts []audit_log.Option[decisionLogEntry]

	inboxPath := filepath.Join(dataRoot, "review", "review_inbox.json")

	svc := &Service{
		cards:            []Card{},
		lightweightCards: []LightweightCard{},
		dataRoot:         dataRoot,
		inboxPath:        inboxPath,
		archive:          NewArchiveService(dataRoot),
		decisionLog:      audit_log.New[decisionLogEntry](dataRoot+"/review/review_decision_log.jsonl", logOpts...),
		securityLog:      audit_log.New[decisionLogEntry](dataRoot+"/review/security_change_log.jsonl", logOpts...),
	}

	// ── 啟動時載入 open cards（file-backed cold start）──
	svc.loadInbox()

	return svc
}

// ──────────────────────────────────────────────
// Standard Card 操作
// ──────────────────────────────────────────────

// AddCard 新增一張完整的 Review Card（v3.6.1）。
// TASKS_1_6_3：寫入後同步 flush 到 review_inbox.json，確保重啟不遺失。
func (s *Service) AddCard(params CardParams) Card {
	s.mu.Lock()
	defer s.mu.Unlock()
	card := NewCard(params)
	s.cards = append(s.cards, card)
	s.saveInboxLocked() // 持久化到磁碟
	return card
}

// AddLegacyCard 相容舊版 AddCard 介面。
// 將舊參數轉換為新 CardParams 結構。
func (s *Service) AddLegacyCard(level Level, sourceType, sourceID, plainReason, engineerReason string) Card {
	rc := legacyLevelToRisk(level)
	return s.AddCard(CardParams{
		RiskClass:      rc,
		Operation:      sourceType,
		Target:         sourceID,
		Reason:         plainReason,
		AcceptLabel:    "確認",
		RejectLabel:    "取消",
		AcceptEffect:   plainReason,
		RejectEffect:   "不執行任何操作",
		SourceType:     sourceType,
		SourceID:       sourceID,
		EngineerReason: engineerReason,
	})
}

// legacyLevelToRisk 將舊版 Level 映射到新版 RiskClass。
func legacyLevelToRisk(l Level) risk.RiskClass {
	switch l {
	case LevelBlocking:
		return risk.HighNonDestructive
	case LevelPending:
		return risk.Medium
	default:
		return risk.Low
	}
}

// ListOpen 回傳所有未解決的 Standard Review Card。
func (s *Service) ListOpen() []Card {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []Card
	for _, c := range s.cards {
		if !c.Resolved {
			result = append(result, c)
		}
	}
	return result
}

// GetCard 根據 ID 取得 Card。
func (s *Service) GetCard(cardID string) (*Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.cards {
		if s.cards[i].ID == cardID {
			return &s.cards[i], nil
		}
	}
	return nil, fmt.Errorf("card not found: %s", cardID)
}

// ──────────────────────────────────────────────
// 解決 Card（含雙步驟確認邏輯）
// ──────────────────────────────────────────────

// ErrReviewContextChanged 表示 Step 2 時環境 hash 已變更，卡片自動失效。
var ErrReviewContextChanged = fmt.Errorf("review_context_changed")

// Resolve 解決一張 Review Card。
// 對於需要雙步驟確認的卡片（security_boundary_rewrite），
// 必須先呼叫 DualStepConfirmStep1，再呼叫此方法。
// projectRoot 用於重算 hash（§4.6：Step 2 必須在後端重算比對）。
// 若 projectRoot 為空（測試場景），跳過 hash re-check。
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ TASKS_1_7 hash re-check 流程（§4.6 安全合規）：                    │
// │  1. 重算 riskPolicyHash / toolRegistryHash / targetHashSet         │
// │  2. 與 Step 1 快照比對                                             │
// │  3. 不一致 → invalidate card + 回傳 ErrReviewContextChanged        │
// │  4. 一致 → 正常 resolve                                            │
// └─────────────────────────────────────────────────────────────────────┘
func (s *Service) Resolve(cardID string, projectRoot ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.cards {
		if c.ID == cardID {
			// 雙步驟確認檢查
			if c.RequiresDualStep {
				if c.DualStepState == nil || c.DualStepState.Step1ConfirmedAt == "" {
					return fmt.Errorf("security_boundary_rewrite 需要先完成 Step 1 確認")
				}
				if c.DualStepState.Invalidated {
					return fmt.Errorf("此 Review Card 已失效，需重新產生")
				}
				// 驗證冷卻時間（≥3 秒）
				step1Time, err := time.Parse(time.RFC3339Nano, c.DualStepState.Step1ConfirmedAt)
				if err == nil {
					elapsed := time.Since(step1Time)
					if elapsed < time.Duration(c.CooldownSeconds)*time.Second {
						return fmt.Errorf("冷卻時間未到，還需等待 %v", time.Duration(c.CooldownSeconds)*time.Second-elapsed)
					}
				}

				// §4.6 Step 2 hash re-check：後端重算，與 Step 1 快照比對
				if len(projectRoot) > 0 && projectRoot[0] != "" {
					if mismatch := s.recheckHashesLocked(i, projectRoot[0]); mismatch != "" {
						s.cards[i].DualStepState.Invalidated = true
						s.saveInboxLocked()
						return ErrReviewContextChanged
					}
				}

				// 記錄 Step 2 時間
				s.cards[i].DualStepState.Step2ExecutedAt = nowRFC3339()
			}

			s.cards[i].Resolved = true
			s.cards[i].ResolvedAt = nowRFC3339()

			// 持久化：從 inbox 移除 + 寫入 archive
			s.saveInboxLocked()
			s.archiveResolvedCard(s.cards[i])

			// 寫入 append-only 決策 log（§5.4）
			s.writeDecisionLog(s.cards[i])

			return nil
		}
	}
	return nil // idempotent
}

// recheckHashesLocked 在 Step 2 時重算 hash 並與 Step 1 快照比對。
// 回傳不一致的欄位名稱（空字串表示一致）。
// 呼叫者必須持有 mu。
func (s *Service) recheckHashesLocked(cardIdx int, projectRoot string) string {
	card := s.cards[cardIdx]
	state := card.DualStepState

	// 重算當前 hash
	guardHashes := computeCurrentHashesForReview(projectRoot)
	currentTarget := ComputeTargetHashSet([]TargetEntry{{
		Operation:     card.Operation,
		Target:        card.Target,
		AffectedScope: card.AcceptEffect,
	}})

	// 比對
	if state.RiskPolicyHash != guardHashes.RiskPolicyHash {
		return "risk_policy_hash"
	}
	if state.ToolRegistryHash != guardHashes.ToolRegistryHash {
		return "tool_registry_hash"
	}
	if state.TargetHashSet != "" && state.TargetHashSet != currentTarget {
		return "target_hash_set"
	}
	return ""
}

// computeCurrentHashesForReview 計算當前 hash（讀取檔案 SHA-256）。
func computeCurrentHashesForReview(projectRoot string) struct{ RiskPolicyHash, ToolRegistryHash string } {
	rp := computeFileHashForReview(filepath.Join(projectRoot, "risk_policy", "policy.json"))
	tr := computeFileHashForReview(filepath.Join(projectRoot, "tool_registry", "registry.json"))
	return struct{ RiskPolicyHash, ToolRegistryHash string }{rp, tr}
}

// computeFileHashForReview 讀取檔案計算 SHA-256（與 dag/resume_guard.go 邏輯一致）。
func computeFileHashForReview(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "empty"
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

// DualStepConfirmStep1 記錄雙步驟確認的 Step 1（「我了解，繼續」）。
// 僅對 requires_dual_step 為 true 的卡片有效。
// hash 參數由後端（app.go）自行計算後傳入，不依賴前端 relay。
func (s *Service) DualStepConfirmStep1(cardID, scopeHash, riskPolicyHash, toolRegistryHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.cards {
		if c.ID == cardID {
			if !c.RequiresDualStep {
				return fmt.Errorf("此卡片不需要雙步驟確認")
			}
			if c.Resolved {
				return fmt.Errorf("此卡片已解決")
			}

			// 計算 target hash（從 card 欄位）
			targetHash := ComputeTargetHashSet([]TargetEntry{{
				Operation:     c.Operation,
				Target:        c.Target,
				AffectedScope: c.AcceptEffect,
			}})

			s.cards[i].DualStepState = &DualStepState{
				Step1ConfirmedAt: nowRFC3339(),
				ReviewIDAtStep1:  cardID,
				ScopeHashAtStep1: scopeHash,
				RiskPolicyHash:   riskPolicyHash,
				ToolRegistryHash: toolRegistryHash,
				TargetHashSet:    targetHash,
			}
			s.saveInboxLocked() // 持久化 dual-step 狀態
			return nil
		}
	}
	return fmt.Errorf("card not found: %s", cardID)
}

// InvalidateDualStep 使雙步驟確認失效（hash 變更時呼叫）。
func (s *Service) InvalidateDualStep(cardID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.cards {
		if c.ID == cardID && c.DualStepState != nil {
			s.cards[i].DualStepState.Invalidated = true
			return nil
		}
	}
	return fmt.Errorf("card not found or no dual-step state: %s", cardID)
}

// ──────────────────────────────────────────────
// Lightweight Card 操作
// ──────────────────────────────────────────────

// AddLightweightCard 新增 Lightweight Review Card。
// 自動驗證使用條件（§5.2.3）。
func (s *Service) AddLightweightCard(params LightweightParams, scopeMatch bool, riskClass risk.RiskClass) (*LightweightCard, error) {
	if !IsLightweightAllowed(scopeMatch, riskClass) {
		return nil, fmt.Errorf("不允許使用 Lightweight Card：scopeMatch=%v, riskClass=%s", scopeMatch, riskClass)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	card := NewLightweightCard(params)
	s.lightweightCards = append(s.lightweightCards, card)
	return &card, nil
}

// ListOpenLightweight 回傳所有未解決的 Lightweight Card。
func (s *Service) ListOpenLightweight() []LightweightCard {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []LightweightCard
	for _, c := range s.lightweightCards {
		if !c.Resolved {
			result = append(result, c)
		}
	}
	return result
}

// ResolveLightweight 解決 Lightweight Card。
func (s *Service) ResolveLightweight(reviewID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.lightweightCards {
		if c.ReviewID == reviewID {
			s.lightweightCards[i].Resolved = true
			s.lightweightCards[i].ResolvedAt = nowRFC3339()
			return nil
		}
	}
	return nil // idempotent
}

// ──────────────────────────────────────────────
// 查詢輔助
// ──────────────────────────────────────────────

// HasBlocking 判斷是否有任何未解決的 blocking 等級卡片。
// 用於 DAG resume gate 和高風險操作閘門。
func (s *Service) HasBlocking() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.cards {
		if !c.Resolved && risk.IsHigherOrEqual(c.RiskClass, risk.HighNonDestructive) {
			return true
		}
	}
	return false
}

// InvalidateAll 作廢所有未解決的卡片（降級模式使用）。
// TASKS_1_6_3：作廢後同步 flush inbox，避免重啟後殭屍卡片復活。
func (s *Service) InvalidateAll() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	now := nowRFC3339()
	for i, c := range s.cards {
		if !c.Resolved {
			s.cards[i].Resolved = true
			s.cards[i].ResolvedAt = now
			count++
		}
	}
	for i, c := range s.lightweightCards {
		if !c.Resolved {
			s.lightweightCards[i].Resolved = true
			s.lightweightCards[i].ResolvedAt = now
			count++
		}
	}
	s.saveInboxLocked() // 持久化
	return count
}

// ──────────────────────────────────────────────
// File-backed 持久化（TASKS_1_6_3）
// ──────────────────────────────────────────────
//
// ┌─────────────────────────────────────────────────────────────────────┐
// │ review_inbox.json — 儲存所有未解決的 open cards                    │
// │                                                                     │
// │ 格式：JSON array of Card（與 Card struct 完全對應）                │
// │ 只儲存 Resolved == false 的卡片。Resolve 後立即從 inbox 移除。     │
// │                                                                     │
// │ loadInbox()          — 啟動時呼叫，從磁碟載入 open cards           │
// │ saveInboxLocked()    — 每次 cards 變更後呼叫（caller 必須持有 mu） │
// │ archiveResolvedCard()— Resolve 後寫入 review_archive.json          │
// │                                                                     │
// │ 錯誤處理策略：                                                     │
// │  - loadInbox 讀取失敗 → 視為空清單，log warning，不 crash          │
// │  - saveInboxLocked 寫入失敗 → 靜默（in-memory 仍正確，下次會重試）│
// │  - archiveResolvedCard 寫入失敗 → 靜默（decision log 是主審計軌跡）│
// └─────────────────────────────────────────────────────────────────────┘

// loadInbox 從 review_inbox.json 載入未解決的 open cards。
// 僅在 NewServiceWithDataRoot 啟動時呼叫一次。
// 損毀或不存在的檔案不會 crash，視為空清單。
func (s *Service) loadInbox() {
	if s.inboxPath == "" {
		return
	}

	data, err := os.ReadFile(s.inboxPath)
	if err != nil {
		// 檔案不存在或讀取失敗 → 空清單
		return
	}

	var cards []Card
	if err := json.Unmarshal(data, &cards); err != nil {
		// 損毀的 JSON → 空清單，不 crash
		return
	}

	// 只載入未解決的卡片（防禦性過濾）
	for _, c := range cards {
		if !c.Resolved {
			s.cards = append(s.cards, c)
		}
	}
}

// saveInboxLocked 將所有未解決的 cards 寫入 review_inbox.json。
// 呼叫者必須已持有 s.mu。
// 當 dataRoot 為空（純 in-memory 模式）時為 no-op。
func (s *Service) saveInboxLocked() {
	if s.inboxPath == "" {
		return
	}

	// 收集未解決的卡片
	var open []Card
	for _, c := range s.cards {
		if !c.Resolved {
			open = append(open, c)
		}
	}

	// 確保目錄存在（0755 與 decision log 的 §5.4 目錄權限一致，
	// 避免先建立 0700 目錄導致 decision log 的 MkdirAll 跳過而權限不符）
	if err := os.MkdirAll(filepath.Dir(s.inboxPath), 0o755); err != nil {
		return // 靜默失敗
	}

	data, err := json.MarshalIndent(open, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.inboxPath, data, 0o600)
}

// archiveResolvedCard 將已解決的 card 寫入 review_archive.json。
// 透過 ArchiveService 追加，格式與既有 archive 一致。
func (s *Service) archiveResolvedCard(card Card) {
	if s.archive == nil {
		return
	}

	// 轉換為 ArchivedCard 格式
	archived := ArchivedCard{
		ID:             card.ID,
		RiskClass:      card.RiskClass,
		Level:          card.LegacyLevel,
		Status:         "resolved",
		SourceType:     card.SourceType,
		SourceID:       card.SourceID,
		PlainReason:    card.Reason,
		EngineerReason: card.EngineerReason,
		CreatedAt:      card.CreatedAt,
		ArchivedAt:     nowRFC3339(),
	}

	// 透過 ArchiveService 的內部方法追加
	// 注意：ArchiveService 有自己的 mutex，不會與 Service.mu 衝突
	s.archive.mu.Lock()
	defer s.archive.mu.Unlock()
	_ = s.archive.appendLocked(archived)
}

// ──────────────────────────────────────────────
// Append-only 決策日誌（§5.4）
// ──────────────────────────────────────────────

// decisionLogEntry 是寫入 review_decision_log.jsonl 的記錄。
type decisionLogEntry struct {
	ReviewID   string         `json:"review_id"`
	RiskClass  risk.RiskClass `json:"risk_class"`
	Operation  string         `json:"operation"`
	Target     string         `json:"target"`
	Decision   string         `json:"decision"` // "accepted" | "rejected"
	ResolvedAt string         `json:"resolved_at"`
	DualStep   bool           `json:"dual_step"`
}

// writeDecisionLog 寫入 append-only 決策日誌。
// 重構：v4.0 委託 audit_log.AppendLog 共用抽象。
func (s *Service) writeDecisionLog(card Card) {
	if s.dataRoot == "" {
		return // 無資料路徑時跳過（測試環境）
	}

	// 僅 high_non_destructive 以上的決策需要 log（§5.4）
	if !risk.IsHigherOrEqual(card.RiskClass, risk.HighNonDestructive) {
		return
	}

	entry := decisionLogEntry{
		ReviewID:   card.ID,
		RiskClass:  card.RiskClass,
		Operation:  card.Operation,
		Target:     card.Target,
		Decision:   "accepted",
		ResolvedAt: card.ResolvedAt,
		DualStep:   card.RequiresDualStep,
	}

	// 寫入主決策日誌
	if s.decisionLog != nil {
		s.decisionLog.Append(entry)
	}

	// security_boundary_rewrite 額外寫 security_change_log.jsonl（§5.4）
	if card.RiskClass == risk.SecurityBoundaryRewrite && s.securityLog != nil {
		s.securityLog.Append(entry)
	}
}

// ──────────────────────────────────────────────
// 工具函式
// ──────────────────────────────────────────────

// idCounter 是 package-level atomic counter，確保同一秒內產生的 ID 不會碰撞。
// 原先 randomSuffix() 使用 time.Now().Format("000000000") 產生 suffix，
// 但 Go 的 Format 不把 "000000000" 當作奈秒格式（需要 ".000000000"），
// 導致每次都回傳字面 "000000"，同秒建立的 card 全部 ID 相同。
// 改用 atomic counter：簡單、可測、穩定、零碰撞。
var idCounter atomic.Uint64

func generateID(prefix string) string {
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + randomSuffix()
}

// randomSuffix 回傳 6 位數字 suffix，透過 atomic counter 保證唯一。
func randomSuffix() string {
	return fmt.Sprintf("%06d", idCounter.Add(1)%1000000)
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339Nano)
}
