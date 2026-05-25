package review

// ┌─────────────────────────────────────────────────────────────────────┐
// │ persistence_test.go — TASKS_1_6_3 持久化驗證                       │
// │                                                                     │
// │ 驗證 review_inbox.json / review_archive.json 的完整生命週期：      │
// │  1. AddCard 後寫入 review_inbox.json                               │
// │  2. 模擬 app 重啟後 open cards 還在                                │
// │  3. Resolve 後從 inbox 移除、寫入 archive + decision log           │
// │  4. 再重啟一次，resolved card 不會回到 open list                   │
// └─────────────────────────────────────────────────────────────────────┘

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ui_console/audit_log"
	"ui_console/domain/risk"
)

// tmpPersistRoot 建立獨立的 temp dir（避免與 decision_log_test 的 tmpDataRoot 衝突）。
func tmpPersistRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "review_persist_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// ──────────────────────────────────────────────
// 測試 1：AddCard → 重啟 → ListOpen 仍有該 card
// ──────────────────────────────────────────────

// TestReviewServicePersistsOpenCardsAcrossRestart 驗證：
//   - 建立一張 SourceType="visual_learning" 的 Review Card
//   - 確認 review_inbox.json 已寫入
//   - 模擬 app 重啟（重新 NewServiceWithDataRoot）
//   - ListOpen() 仍可看到那張 card
func TestReviewServicePersistsOpenCardsAcrossRestart(t *testing.T) {
	dataRoot := tmpPersistRoot(t)

	// ── 第一次啟動：建立 card ──
	svc := NewServiceWithDataRoot(dataRoot)
	card := svc.AddCard(CardParams{
		RiskClass:      risk.Medium,
		Operation:      "review_visual_learning_candidate",
		Target:         "visual_candidate:abc",
		Reason:         "needs user review",
		AcceptLabel:    "Approve",
		RejectLabel:    "Reject",
		AcceptEffect:   "候選項進入正式知識庫",
		RejectEffect:   "候選項被丟棄",
		SourceType:     "visual_learning",
		SourceID:       "abc",
		EngineerReason: "confidence=0.72, below threshold",
	})

	if card.ID == "" {
		t.Fatal("card ID should not be empty")
	}

	// ── 驗證 review_inbox.json 已寫入 ──
	inboxPath := filepath.Join(dataRoot, "review", "review_inbox.json")
	inboxData, err := os.ReadFile(inboxPath)
	if err != nil {
		t.Fatalf("review_inbox.json should exist after AddCard: %v", err)
	}
	var inboxCards []Card
	if err := json.Unmarshal(inboxData, &inboxCards); err != nil {
		t.Fatalf("review_inbox.json should be valid JSON: %v", err)
	}
	if len(inboxCards) != 1 {
		t.Fatalf("review_inbox.json should have 1 card, got %d", len(inboxCards))
	}
	if inboxCards[0].ID != card.ID {
		t.Errorf("inbox card ID mismatch: want %s, got %s", card.ID, inboxCards[0].ID)
	}

	// ── 模擬 app 重啟：重新建立 service ──
	reloaded := NewServiceWithDataRoot(dataRoot)
	open := reloaded.ListOpen()

	if len(open) != 1 {
		t.Fatalf("after restart, ListOpen should have 1 card, got %d", len(open))
	}
	if open[0].ID != card.ID {
		t.Errorf("after restart, card ID mismatch: want %s, got %s", card.ID, open[0].ID)
	}
	if open[0].SourceType != "visual_learning" {
		t.Errorf("after restart, SourceType should be visual_learning, got %s", open[0].SourceType)
	}
	if open[0].RiskClass != risk.Medium {
		t.Errorf("after restart, RiskClass should be medium, got %s", open[0].RiskClass)
	}
}

// ──────────────────────────────────────────────
// 測試 2：Resolve → 重啟 → resolved card 不在 open list
// ──────────────────────────────────────────────

// TestReviewServiceMovesResolvedCardToArchiveAcrossRestart 驗證：
//   - 建立 card → Resolve
//   - ListOpen() 不再看到它
//   - review_inbox.json 不再有它
//   - review_archive.json 有它
//   - review_decision_log.jsonl 有 resolve 紀錄（僅 high+ 會記錄）
//   - 再重啟一次，resolved card 不會回到 open list
func TestReviewServiceMovesResolvedCardToArchiveAcrossRestart(t *testing.T) {
	dataRoot := tmpPersistRoot(t)

	// ── 建立 card（用 HighNonDestructive 確保 decision log 會記錄）──
	svc := NewServiceWithDataRoot(dataRoot)
	card := svc.AddCard(CardParams{
		RiskClass:      risk.HighNonDestructive,
		Operation:      "review_visual_learning_candidate",
		Target:         "visual_candidate:xyz",
		Reason:         "high risk visual action",
		AcceptLabel:    "Approve",
		RejectLabel:    "Reject",
		AcceptEffect:   "候選項進入正式知識庫",
		RejectEffect:   "候選項被丟棄",
		SourceType:     "visual_learning",
		SourceID:       "xyz",
		EngineerReason: "high risk action detected",
	})

	// ── Resolve ──
	if err := svc.Resolve(card.ID); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// ── 驗證 ListOpen() 不再看到它 ──
	if len(svc.ListOpen()) != 0 {
		t.Error("ListOpen should be empty after Resolve")
	}

	// ── 驗證 review_inbox.json 不再有它 ──
	inboxPath := filepath.Join(dataRoot, "review", "review_inbox.json")
	inboxData, err := os.ReadFile(inboxPath)
	if err != nil {
		t.Fatalf("review_inbox.json should exist: %v", err)
	}
	var inboxCards []Card
	_ = json.Unmarshal(inboxData, &inboxCards)
	for _, c := range inboxCards {
		if c.ID == card.ID {
			t.Error("resolved card should not be in review_inbox.json")
		}
	}

	// ── 驗證 review_archive.json 有它 ──
	archivePath := filepath.Join(dataRoot, "review", "review_archive.json")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("review_archive.json should exist after Resolve: %v", err)
	}
	var archivedCards []ArchivedCard
	if err := json.Unmarshal(archiveData, &archivedCards); err != nil {
		t.Fatalf("review_archive.json should be valid JSON: %v", err)
	}
	found := false
	for _, ac := range archivedCards {
		if ac.ID == card.ID {
			found = true
			if ac.Status != "resolved" {
				t.Errorf("archived card status should be 'resolved', got %s", ac.Status)
			}
			if ac.SourceType != "visual_learning" {
				t.Errorf("archived card SourceType should be 'visual_learning', got %s", ac.SourceType)
			}
		}
	}
	if !found {
		t.Error("resolved card should be in review_archive.json")
	}

	// ── 驗證 review_decision_log.jsonl 有 resolve 紀錄 ──
	logPath := filepath.Join(dataRoot, "review", "review_decision_log.jsonl")
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("review_decision_log.jsonl should exist for high-risk resolve: %v", err)
	}
	lines := audit_log.SplitLines(logData)
	logFound := false
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry decisionLogEntry
		if err := json.Unmarshal(line, &entry); err == nil && entry.ReviewID == card.ID {
			logFound = true
			if entry.Decision != "accepted" {
				t.Errorf("decision log entry should be 'accepted', got %s", entry.Decision)
			}
		}
	}
	if !logFound {
		t.Error("resolved card should have an entry in review_decision_log.jsonl")
	}

	// ── 再重啟一次：resolved card 不應回到 open list ──
	reloaded := NewServiceWithDataRoot(dataRoot)
	open := reloaded.ListOpen()
	if len(open) != 0 {
		t.Errorf("after restart, resolved card should not reappear in ListOpen, got %d cards", len(open))
	}
}

// ──────────────────────────────────────────────
// 測試 3：多張 card 混合場景
// ──────────────────────────────────────────────

// TestReviewServiceMixedCardsAcrossRestart 驗證多張 card 的混合持久化：
//   - 建立 3 張 card（不同 SourceType）
//   - Resolve 其中 1 張
//   - 重啟 → 只剩 2 張 open
func TestReviewServiceMixedCardsAcrossRestart(t *testing.T) {
	dataRoot := tmpPersistRoot(t)

	svc := NewServiceWithDataRoot(dataRoot)

	// randomSuffix 已改為 atomic counter，同秒建立多張 card 不再碰撞。
	card1 := svc.AddCard(CardParams{
		RiskClass: risk.Medium, Operation: "review_vl_label",
		Target: "label:a", Reason: "test", AcceptLabel: "ok", RejectLabel: "no",
		SourceType: "visual_learning", SourceID: "a",
	})
	card2 := svc.AddCard(CardParams{
		RiskClass: risk.Low, Operation: "package_import",
		Target: "pkg:b", Reason: "test", AcceptLabel: "ok", RejectLabel: "no",
		SourceType: "package_import", SourceID: "b",
	})
	card3 := svc.AddCard(CardParams{
		RiskClass: risk.Medium, Operation: "review_vl_action",
		Target: "action:c", Reason: "test", AcceptLabel: "ok", RejectLabel: "no",
		SourceType: "visual_learning", SourceID: "c",
	})

	// 確認三張 card ID 不同
	if card1.ID == card2.ID || card2.ID == card3.ID || card1.ID == card3.ID {
		t.Fatalf("card IDs should be unique: %s, %s, %s", card1.ID, card2.ID, card3.ID)
	}

	// Resolve card1
	if err := svc.Resolve(card1.ID); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// 重啟
	reloaded := NewServiceWithDataRoot(dataRoot)
	open := reloaded.ListOpen()

	if len(open) != 2 {
		t.Fatalf("after restart, should have 2 open cards, got %d", len(open))
	}

	// 確認 card1 不在（用 SourceID 辨識）
	for _, c := range open {
		if c.SourceID == "a" && c.SourceType == "visual_learning" && c.Operation == "review_vl_label" {
			t.Error("resolved card1 (SourceID=a) should not be in open list after restart")
		}
	}

	// 確認 card2 和 card3 存在
	foundB, foundC := false, false
	for _, c := range open {
		if c.SourceID == "b" {
			foundB = true
		}
		if c.SourceID == "c" {
			foundC = true
		}
	}
	if !foundB {
		t.Error("card2 (SourceID=b) should be in open list after restart")
	}
	if !foundC {
		t.Error("card3 (SourceID=c) should be in open list after restart")
	}
}

// ──────────────────────────────────────────────
// 測試 4：generateID 同秒內不碰撞
// ──────────────────────────────────────────────

// TestGenerateIDUniqueWithinSameSecond 驗證在同一秒內快速產生 1000 個 ID，
// 每個都不重複。確認 randomSuffix 的 atomic counter 正確運作。
func TestGenerateIDUniqueWithinSameSecond(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := generateID("rev")
		if seen[id] {
			t.Fatalf("duplicate id at iteration %d: %s", i, id)
		}
		seen[id] = true
	}
}
