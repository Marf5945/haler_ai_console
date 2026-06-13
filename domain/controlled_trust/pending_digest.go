package controlled_trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DigestGroupBackend is the backend 5-way classification of pending items.
type DigestGroupBackend string

const (
	DigestKeepSuggestion     DigestGroupBackend = "keep_suggestion"
	DigestArchiveSuggestion  DigestGroupBackend = "archive_suggestion"
	DigestDuplicateGroup     DigestGroupBackend = "duplicate_group"
	DigestHighValueCandidate DigestGroupBackend = "high_value_candidate"
	DigestRiskyCandidate     DigestGroupBackend = "risky_candidate"
)

// DigestGroupUI is the 3-way UI classification shown to users.
// Maps from backend groups per spec §0 v3.3.0.
type DigestGroupUI string

const (
	// UIGroupDecide = risky_candidate + high_value_candidate (default: expanded)
	UIGroupDecide DigestGroupUI = "needs_your_decision"
	// UIGroupLater = keep_suggestion (default: collapsed)
	UIGroupLater DigestGroupUI = "can_wait"
	// UIGroupArchive = archive_suggestion + duplicate_group (default: collapsed)
	UIGroupArchive DigestGroupUI = "suggested_archive"
)

// DigestItem is one pending item in the digest.
type DigestItem struct {
	ID           string             `json:"id"`
	SourceType   string             `json:"source_type"` // pending source type
	BackendGroup DigestGroupBackend `json:"backend_group"`
	UIGroup      DigestGroupUI      `json:"ui_group"`
	Summary      string             `json:"summary"`
	Confidence   float64            `json:"confidence"`
	AgeDays      int                `json:"age_days"`
}

// DigestAction is the user's explicit action on a pending item.
type DigestAction string

const (
	DigestActionKeep               DigestAction = "keep"
	DigestActionReviewNow          DigestAction = "review_now"
	DigestActionArchiveSingle      DigestAction = "archive_single"
	DigestActionBatchArchiveLow    DigestAction = "batch_archive_low_value"
	DigestActionDeleteSingle       DigestAction = "delete_single"
	DigestActionBatchDelete        DigestAction = "batch_delete"
	DigestActionDeleteHighCritical DigestAction = "delete_high_or_critical_candidate"
)

// DigestActionRiskLevel is the review level required before an action runs.
type DigestActionRiskLevel string

const (
	DigestActionRiskLow            DigestActionRiskLevel = "low"
	DigestActionRiskMedium         DigestActionRiskLevel = "medium"
	DigestActionRiskHigh           DigestActionRiskLevel = "high"
	DigestActionRiskBlockingReview DigestActionRiskLevel = "blocking_review"
)

// PendingDigest is the result of one digest computation run.
// Computation is LOCAL ONLY — no LLM calls during digest generation.
// Must NOT auto-enable, auto-delete, or auto-promote any candidates.
// Corresponds to schema #54 in TASKS_1_2.md.
type PendingDigest struct {
	ID           string         `json:"id"`
	GeneratedAt  time.Time      `json:"generated_at"`
	SourceCounts map[string]int `json:"source_counts"`
	Items        []DigestItem   `json:"items"`

	// UI group counts (shown at top of Pending Digest UI).
	DecideCount  int `json:"decide_count"`  // needs_your_decision
	LaterCount   int `json:"later_count"`   // can_wait
	ArchiveCount int `json:"archive_count"` // suggested_archive
}

// PendingDigestService generates and manages pending digests.
type PendingDigestService struct {
	mu        sync.Mutex
	digestDir string
}

func NewPendingDigestService(trustDir string) *PendingDigestService {
	return &PendingDigestService{
		digestDir: filepath.Join(trustDir, "pending_digest"),
	}
}

// LoadLatest reads the last generated weekly digest. If there is no digest yet,
// callers get an empty digest and may choose to run Generate with current
// pending items. This keeps UI startup quiet while still supporting on-demand
// computation.
func (s *PendingDigestService) LoadLatest() (*PendingDigest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.digestDir, "weekly_digest.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &PendingDigest{
			ID:           "digest-empty",
			GeneratedAt:  time.Now(),
			SourceCounts: map[string]int{},
			Items:        []DigestItem{},
		}, nil
	}
	if err != nil {
		return nil, err
	}
	var digest PendingDigest
	if err := json.Unmarshal(data, &digest); err != nil {
		return nil, err
	}
	return &digest, nil
}

// Generate runs a local-only digest computation over the given pending items.
// This is called on the weekly schedule or on demand — never automatically on startup.
func (s *PendingDigestService) Generate(items []DigestItem) (*PendingDigest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Classify items into backend groups and UI groups.
	classified := make([]DigestItem, len(items))
	sourceCounts := make(map[string]int)
	for i, item := range items {
		item.UIGroup = backendToUI(item.BackendGroup)
		classified[i] = item
		sourceCounts[item.SourceType]++
	}

	digest := &PendingDigest{
		ID:           fmt.Sprintf("digest-%d", time.Now().UnixNano()),
		GeneratedAt:  time.Now(),
		SourceCounts: sourceCounts,
		Items:        classified,
	}

	// Count UI groups for top-of-page display.
	for _, item := range classified {
		switch item.UIGroup {
		case UIGroupDecide:
			digest.DecideCount++
		case UIGroupLater:
			digest.LaterCount++
		case UIGroupArchive:
			digest.ArchiveCount++
		}
	}

	return digest, s.saveLocked(digest)
}

// AcknowledgeItem records a user's explicit action on a pending item.
// None of the actions are automatic — they all require explicit user confirmation.
func (s *PendingDigestService) AcknowledgeItem(itemID string, action DigestAction) error {
	return s.AcknowledgeItemWithConfirmation(itemID, action, "")
}

// AcknowledgeItemWithConfirmation records a user's explicit action and enforces
// typed confirmation for high/critical deletion. Archive actions remain
// recoverable; delete actions write a tombstone for audit and de-duplication.
func (s *PendingDigestService) AcknowledgeItemWithConfirmation(itemID string, action DigestAction, confirmation string) error {
	switch action {
	case DigestActionKeep, DigestActionReviewNow, DigestActionArchiveSingle,
		DigestActionBatchArchiveLow, DigestActionDeleteSingle, DigestActionBatchDelete:
		if err := s.recordActionLocked(itemID, string(action), string(ActionRiskLevel(action))); err != nil {
			return err
		}
		if action == DigestActionDeleteSingle || action == DigestActionBatchDelete {
			return s.writeTombstoneLocked(itemID, action, "user_explicit_delete")
		}
		return nil
	case DigestActionDeleteHighCritical:
		expected := "DELETE " + itemID
		if confirmation != expected {
			return fmt.Errorf("pending_digest: typed confirmation required, expected %q", expected)
		}
		if err := s.recordActionLocked(itemID, string(action), string(DigestActionRiskBlockingReview)); err != nil {
			return err
		}
		return s.writeTombstoneLocked(itemID, action, "typed_confirmation")
	default:
		return fmt.Errorf("pending_digest: unknown action %q", action)
	}
}

// ActionRiskLevel maps digest actions to the fixed v3.3.2 review levels.
func ActionRiskLevel(action DigestAction) DigestActionRiskLevel {
	switch action {
	case DigestActionKeep, DigestActionReviewNow, DigestActionArchiveSingle:
		return DigestActionRiskLow
	case DigestActionBatchArchiveLow, DigestActionDeleteSingle:
		return DigestActionRiskMedium
	case DigestActionBatchDelete:
		return DigestActionRiskHigh
	case DigestActionDeleteHighCritical:
		return DigestActionRiskBlockingReview
	default:
		return DigestActionRiskBlockingReview
	}
}

// DigestAutoArchiveLimit 是 Digest Items 的上限。超過此數量時，
// 最舊的 suggested_archive 類別項目會被自動封存以釋放空間。
const DigestAutoArchiveLimit = 300

// AutoArchiveResult 記錄一次自動封存的結果，用於通知前端。
type AutoArchiveResult struct {
	ArchivedCount int    `json:"archived_count"`
	Message       string `json:"message"`
}

// AutoArchiveIfOverLimit 檢查 digest 是否超過 300 筆上限。
// 超過時，優先封存 suggested_archive 類別的最舊項目，
// 將被封存的 item 寫入 tombstone.jsonl 留下審計軌跡，
// 並回傳被封存的筆數以供 EventBus 通知前端。
func (s *PendingDigestService) AutoArchiveIfOverLimit(digest *PendingDigest) (*AutoArchiveResult, error) {
	if len(digest.Items) <= DigestAutoArchiveLimit {
		return nil, nil // 未超限，無需動作
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 將 suggested_archive 類別的項目收集出來（最舊的排前面）
	var archiveCandidates []int // 原始 index
	for i, item := range digest.Items {
		if item.UIGroup == UIGroupArchive {
			archiveCandidates = append(archiveCandidates, i)
		}
	}

	// 計算需要封存多少筆才能降到上限以下
	excess := len(digest.Items) - DigestAutoArchiveLimit
	toArchive := excess
	if toArchive > len(archiveCandidates) {
		toArchive = len(archiveCandidates) // 最多只能封存 archive 類別的數量
	}
	if toArchive == 0 {
		return nil, nil
	}

	// 寫入 tombstone 留下審計紀錄，然後從 digest 移除
	removeSet := make(map[int]bool, toArchive)
	for i := 0; i < toArchive; i++ {
		idx := archiveCandidates[i]
		item := digest.Items[idx]
		_ = s.writeTombstoneLocked(item.ID, DigestActionBatchArchiveLow, "auto_archive_300_limit")
		removeSet[idx] = true
	}

	// 重建 Items 切片，移除已封存的項目
	kept := make([]DigestItem, 0, len(digest.Items)-toArchive)
	for i, item := range digest.Items {
		if !removeSet[i] {
			kept = append(kept, item)
		}
	}
	digest.Items = kept

	// 重新計算 UI group 計數
	digest.DecideCount = 0
	digest.LaterCount = 0
	digest.ArchiveCount = 0
	for _, item := range digest.Items {
		switch item.UIGroup {
		case UIGroupDecide:
			digest.DecideCount++
		case UIGroupLater:
			digest.LaterCount++
		case UIGroupArchive:
			digest.ArchiveCount++
		}
	}

	// 儲存更新後的 digest
	_ = s.saveLocked(digest)

	return &AutoArchiveResult{
		ArchivedCount: toArchive,
		Message:       fmt.Sprintf("自動封存 %d 筆過期候選項來釋放空間", toArchive),
	}, nil
}

func backendToUI(group DigestGroupBackend) DigestGroupUI {
	switch group {
	case DigestRiskyCandidate, DigestHighValueCandidate:
		return UIGroupDecide
	case DigestKeepSuggestion:
		return UIGroupLater
	case DigestArchiveSuggestion, DigestDuplicateGroup:
		return UIGroupArchive
	default:
		return UIGroupDecide // default to visible
	}
}

func (s *PendingDigestService) saveLocked(digest *PendingDigest) error {
	if err := os.MkdirAll(s.digestDir, 0o700); err != nil {
		return err
	}
	// Save JSON digest.
	jsonPath := filepath.Join(s.digestDir, "weekly_digest.json")
	data, err := json.MarshalIndent(digest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, data, 0o600); err != nil {
		return err
	}
	// Save human-readable MD summary.
	md := fmt.Sprintf("# Pending Digest — %s\n\n"+
		"**需要你決定（%d）** · **可以稍後（%d）** · **建議封存（%d）**\n\n"+
		"Generated at: %s\n",
		digest.ID, digest.DecideCount, digest.LaterCount, digest.ArchiveCount,
		digest.GeneratedAt.Format(time.RFC3339))
	return os.WriteFile(filepath.Join(s.digestDir, "weekly_digest.md"), []byte(md), 0o600)
}

func (s *PendingDigestService) recordActionLocked(itemID, action, riskLevel string) error {
	if err := os.MkdirAll(s.digestDir, 0o700); err != nil {
		return err
	}
	logPath := filepath.Join(s.digestDir, "action_log.jsonl")
	entry := map[string]interface{}{
		"item_id":     itemID,
		"action":      action,
		"risk_level":  riskLevel,
		"recorded_at": time.Now(),
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

func (s *PendingDigestService) writeTombstoneLocked(itemID string, action DigestAction, reason string) error {
	if err := os.MkdirAll(s.digestDir, 0o700); err != nil {
		return err
	}
	entry := map[string]interface{}{
		"item_id":    itemID,
		"action":     action,
		"reason":     reason,
		"deleted_at": time.Now(),
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(s.digestDir, "tombstone.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}
