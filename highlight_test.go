package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui_console/data/storage"
)

func TestGrounded(t *testing.T) {
	src := "機車需要定期保養，才不會半路損壞"
	if !grounded("機車", src) {
		t.Fatalf("機車 應該接地成功")
	}
	if !grounded("定期保養", src) {
		t.Fatalf("定期保養 應該接地成功")
	}
	if grounded("汽車", src) {
		t.Fatalf("汽車 不在原文，應丟棄")
	}
	if grounded("[REDACTED_EMAIL]", src+" [REDACTED_EMAIL]") {
		t.Fatalf("含遮蔽標記片段不得收")
	}
	if grounded("  ", src) {
		t.Fatalf("空白不得收")
	}
}

func TestRedactPII(t *testing.T) {
	in := "聯絡 a.b@example.com 或 0912-345-678，卡號 1234567890123"
	out := redactPII(in, "user_text")
	for _, leak := range []string{"a.b@example.com", "0912-345-678", "1234567890123"} {
		if strings.Contains(out, leak) {
			t.Fatalf("PII 未遮蔽: %q 仍在 %q", leak, out)
		}
	}
	if !strings.Contains(out, "[REDACTED_EMAIL]") {
		t.Fatalf("email 應被遮蔽，得到 %q", out)
	}
}

func TestDefaultWeightForSlot(t *testing.T) {
	if got := defaultWeightForSlot(0); got != 0.5 {
		t.Fatalf("slot0 weight=%v 期望 0.5", got)
	}
	if defaultWeightForSlot(1) <= defaultWeightForSlot(0) {
		t.Fatalf("權重應隨 slot 遞增")
	}
	if got := defaultWeightForSlot(100); got > 1 {
		t.Fatalf("權重不得超過 1，得到 %v", got)
	}
}

func TestExtractJSONArrayAndParse(t *testing.T) {
	if got := extractJSONArray("噪音前 [\"a\",\"b\"] 噪音後"); got != "[\"a\",\"b\"]" {
		t.Fatalf("extractJSONArray=%q", got)
	}
	arr := parseStringArray("結果：[\"機車\", \" 保養 \", \"\"]")
	if len(arr) != 2 || arr[0] != "機車" || arr[1] != "保養" {
		t.Fatalf("parseStringArray=%v", arr)
	}
}

func TestGcSystemMarks(t *testing.T) {
	st := &highlightStore{}
	for i := 0; i < systemMarkStoreCap+50; i++ {
		conf := 0.1
		if i < 10 {
			conf = 0.9 // 前 10 筆高信心，應被保留
		}
		st.Highlights = append(st.Highlights, Highlight{
			ID:         newSystemMarkID(),
			System:     true,
			Confidence: conf,
			CreatedAt:  "2026-06-20T00:00:00Z",
		})
	}
	gcSystemMarks(st)
	if len(st.Highlights) != systemMarkStoreCap {
		t.Fatalf("gc 後筆數=%d 期望 %d", len(st.Highlights), systemMarkStoreCap)
	}
	high := 0
	for _, h := range st.Highlights {
		if h.Confidence == 0.9 {
			high++
		}
	}
	if high != 10 {
		t.Fatalf("高信心筆數應全保留，得到 %d", high)
	}
}

func TestLexicalScoresPrefersOverlap(t *testing.T) {
	st := &highlightStore{
		Groups: []HighlightGroup{
			{GroupID: "g_vehicle", ColorSlot: 1},
			{GroupID: "g_food", ColorSlot: 2},
		},
		Highlights: []Highlight{
			{GroupID: "g_vehicle", ColorSlot: 1, Quote: "機車定期保養"},
			{GroupID: "g_food", ColorSlot: 2, Quote: "晚餐吃牛肉麵"},
		},
	}
	groups := st.nonEmptyGroups()
	scores := lexicalScores("機車保養", st, groups)
	var vehicle, food float64
	for _, s := range scores {
		if s.GroupID == "g_vehicle" {
			vehicle = s.Score
		}
		if s.GroupID == "g_food" {
			food = s.Score
		}
	}
	if vehicle <= food {
		t.Fatalf("機車保養 應較貼近 vehicle 組：vehicle=%v food=%v", vehicle, food)
	}
}

func TestColorSlotBounds(t *testing.T) {
	if highlightMaxGroups != 8 {
		t.Fatalf("色盤應為 8，得到 %d", highlightMaxGroups)
	}
}

func TestClearMainTalkPurgesConversationMarks(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AI_CONSOLE_DATA_ROOT", root)
	if err := storage.EnsureProjectLayout(root, "default"); err != nil {
		t.Fatalf("EnsureProjectLayout error: %v", err)
	}
	projectRoot := storage.ProjectRoot(root, "default")
	talkPath := filepath.Join(projectRoot, "memory", "talk_full.md")
	if err := os.WriteFile(talkPath, []byte("user: hello"), 0o600); err != nil {
		t.Fatalf("write talk_full: %v", err)
	}
	if err := saveHighlightStore("main", &highlightStore{Highlights: []Highlight{{ID: "h1", MessageID: "m1", Quote: "hello"}}}); err != nil {
		t.Fatalf("save user highlights: %v", err)
	}
	sysPath, err := systemMarksPath("main")
	if err != nil {
		t.Fatalf("systemMarksPath error: %v", err)
	}
	if err := saveHighlightStoreAt(sysPath, &highlightStore{Highlights: []Highlight{{ID: "s1", MessageID: "m1", Quote: "hello", System: true}}}); err != nil {
		t.Fatalf("save system marks: %v", err)
	}
	if err := saveSystemPending("main", &systemPendingQueue{Items: []SystemMarkPending{{MessageID: "m1", Text: "hello", Source: "user_text"}}}); err != nil {
		t.Fatalf("save pending marks: %v", err)
	}
	statsPath, err := highlightStatsPath("main")
	if err != nil {
		t.Fatalf("highlightStatsPath error: %v", err)
	}
	if err := os.WriteFile(statsPath, []byte(`{"suggestedTotal":1,"accepted":1}`), 0o600); err != nil {
		t.Fatalf("write stats: %v", err)
	}

	if err := (&App{}).ClearMainTalk(); err != nil {
		t.Fatalf("ClearMainTalk error: %v", err)
	}
	if raw, err := os.ReadFile(talkPath); err != nil || len(raw) != 0 {
		t.Fatalf("talk_full not empty: len=%d err=%v", len(raw), err)
	}
	userStore, err := loadHighlightStore("main")
	if err != nil {
		t.Fatalf("load user highlights: %v", err)
	}
	if len(userStore.Highlights) != 0 {
		t.Fatalf("user highlights not purged: %d", len(userStore.Highlights))
	}
	sysStore, err := loadHighlightStoreAt(sysPath)
	if err != nil {
		t.Fatalf("load system marks: %v", err)
	}
	if len(sysStore.Highlights) != 0 {
		t.Fatalf("system marks not purged: %d", len(sysStore.Highlights))
	}
	pending, err := loadSystemPending("main")
	if err != nil {
		t.Fatalf("load pending marks: %v", err)
	}
	if len(pending.Items) != 0 {
		t.Fatalf("pending marks not purged: %d", len(pending.Items))
	}
	if _, err := os.Stat(statsPath); !os.IsNotExist(err) {
		t.Fatalf("highlight_stats should be removed, err=%v", err)
	}
}

func TestPurgeConversationMarksReportsError(t *testing.T) {
	// 不存在 / 不合法的 subagent → 路徑解析失敗 → 應回傳非 nil 錯誤，
	// 讓 ClearMainTalk 那層能記 log（而非靜默吞掉）。
	if err := (&App{}).PurgeConversationMarks("__no_such_sub_zzz__"); err == nil {
		t.Fatalf("對不存在的 sub 應回傳錯誤，供呼叫端記 log")
	}
}
