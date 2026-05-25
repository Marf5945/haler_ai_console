// conversation/conversation_test.go — 對話套件單元測試。
package conversation

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ──────────────────────────────────────────────
// TestAddInput — 新增輸入句，驗證 ID 為 [I-001]
// ──────────────────────────────────────────────

func TestAddInput(t *testing.T) {
	store := NewSentenceStore()
	sent := store.AddInput("你好")
	if sent.ID != "[I-001]" {
		t.Errorf("AddInput: 期望 [I-001]，實際 %s", sent.ID)
	}
	if sent.Role != "user" {
		t.Errorf("AddInput: 期望 role=user，實際 %s", sent.Role)
	}
}

// ──────────────────────────────────────────────
// TestAddOutput — 新增輸出句，驗證 ID 為 [O-001]
// ──────────────────────────────────────────────

func TestAddOutput(t *testing.T) {
	store := NewSentenceStore()
	sent := store.AddOutput("天氣晴朗")
	if sent.ID != "[O-001]" {
		t.Errorf("AddOutput: 期望 [O-001]，實際 %s", sent.ID)
	}
	if sent.Role != "assistant" {
		t.Errorf("AddOutput: 期望 role=assistant，實際 %s", sent.Role)
	}
}

// ──────────────────────────────────────────────
// TestAddToolAction — 驗證 tool-action ID 格式
// ──────────────────────────────────────────────

func TestAddToolAction(t *testing.T) {
	store := NewSentenceStore()
	// 新增輸入句以獲得 I-001
	_ = store.AddInput("查天氣")
	sent := store.AddToolAction("[I-001]", "weather", "28°C")

	expected := "[tool-action: I-001 weather → 28°C]"
	if sent.ID != expected {
		t.Errorf("AddToolAction ID: 期望 %q，實際 %q", expected, sent.ID)
	}
	if sent.Role != "tool-action" {
		t.Errorf("AddToolAction Role: 期望 tool-action，實際 %s", sent.Role)
	}
}

// ──────────────────────────────────────────────
// TestSequentialIDs — 連續新增三筆輸入，驗證 ID 遞增
// ──────────────────────────────────────────────

func TestSequentialIDs(t *testing.T) {
	store := NewSentenceStore()
	ids := []string{"[I-001]", "[I-002]", "[I-003]"}
	for i, expected := range ids {
		sent := store.AddInput("內容")
		if sent.ID != expected {
			t.Errorf("第 %d 筆: 期望 %s，實際 %s", i+1, expected, sent.ID)
		}
	}
}

// ──────────────────────────────────────────────
// TestDelete — 新增三筆後刪除 [I-002]，確認剩兩筆
// ──────────────────────────────────────────────

func TestDelete(t *testing.T) {
	store := NewSentenceStore()
	store.AddInput("第一句")
	store.AddInput("第二句")
	store.AddInput("第三句")

	ok := store.Delete("[I-002]")
	if !ok {
		t.Error("Delete: 應成功刪除 [I-002]，卻回傳 false")
	}
	all := store.GetAll()
	if len(all) != 2 {
		t.Errorf("Delete: 期望剩 2 筆，實際 %d 筆", len(all))
	}
	// 確認 I-002 已不存在
	for _, s := range all {
		if s.ID == "[I-002]" {
			t.Error("Delete: [I-002] 仍存在於 store")
		}
	}
}

// ──────────────────────────────────────────────
// TestMove — 新增三筆，將 [I-003] 移到 position 0，驗證順序
// ──────────────────────────────────────────────

func TestMove(t *testing.T) {
	store := NewSentenceStore()
	store.AddInput("第一句")
	store.AddInput("第二句")
	store.AddInput("第三句")

	ok := store.Move("[I-003]", 0)
	if !ok {
		t.Error("Move: 應成功，卻回傳 false")
	}
	all := store.GetAll()
	if len(all) != 3 {
		t.Fatalf("Move: 期望 3 筆，實際 %d 筆", len(all))
	}
	// position 0 應為 I-003
	if all[0].ID != "[I-003]" {
		t.Errorf("Move: 位置 0 期望 [I-003]，實際 %s", all[0].ID)
	}
	// 原第一句應移到 position 1
	if all[1].ID != "[I-001]" {
		t.Errorf("Move: 位置 1 期望 [I-001]，實際 %s", all[1].ID)
	}
}

// ──────────────────────────────────────────────
// TestCharCount — 混合輸入/輸出/工具動作，驗證字元計數（使用中文）
// ──────────────────────────────────────────────

func TestCharCount(t *testing.T) {
	store := NewSentenceStore()
	// 中文 rune 計數：「你好」= 2，「世界」= 2，工具動作不計
	store.AddInput("你好")                        // 2 runes
	store.AddOutput("世界")                       // 2 runes
	store.AddToolAction("[I-001]", "calc", "4") // 不計入

	count := store.CharCount()
	if count != 4 {
		t.Errorf("CharCount: 期望 4，實際 %d", count)
	}
}

// ──────────────────────────────────────────────
// TestCheckSummaryIntegrity — 連續群組通過，刪除中間元素後失敗
// ──────────────────────────────────────────────

func TestCheckSummaryIntegrity(t *testing.T) {
	store := NewSentenceStore()
	store.AddInput("A")
	store.AddInput("B")
	store.AddInput("C")
	store.AddInput("D")

	// 連續群組 [I-001]~[I-004] 應通過
	ids := []string{"[I-001]", "[I-002]", "[I-003]", "[I-004]"}
	if !store.CheckSummaryIntegrity(ids) {
		t.Error("CheckSummaryIntegrity: 連續群組應回傳 true")
	}

	// 刪除 [I-002] 使群組不連續，應失敗
	store.Delete("[I-002]")
	if store.CheckSummaryIntegrity(ids) {
		t.Error("CheckSummaryIntegrity: 缺少 [I-002] 後應回傳 false")
	}
}

// ──────────────────────────────────────────────
// TestFormatForTalkFull — 格式化句子，驗證輸出含 ID 與 role
// ──────────────────────────────────────────────

func TestFormatForTalkFull(t *testing.T) {
	store := NewSentenceStore()
	store.AddInput("你好世界")
	store.AddOutput("哈囉")
	sents := store.GetAll()

	output := FormatForTalkFull(sents)
	// 驗證包含 I-001 區段標題
	if !strings.Contains(output, "[I-001]") {
		t.Error("FormatForTalkFull: 缺少 [I-001]")
	}
	// 驗證包含 O-001 區段標題
	if !strings.Contains(output, "[O-001]") {
		t.Error("FormatForTalkFull: 缺少 [O-001]")
	}
	// 驗證包含 user 角色標記
	if !strings.Contains(output, "user") {
		t.Error("FormatForTalkFull: 缺少 user role")
	}
	// 驗證包含句子內容
	if !strings.Contains(output, "你好世界") {
		t.Error("FormatForTalkFull: 缺少句子內容")
	}
}

// ──────────────────────────────────────────────
// TestParseTalkFull — 格式化再解析，驗證內容一致（round-trip）
// ──────────────────────────────────────────────

func TestParseTalkFull(t *testing.T) {
	store := NewSentenceStore()
	store.AddInput("你好世界")
	store.AddOutput("哈囉")
	original := store.GetAll()

	// 格式化後解析
	formatted := FormatForTalkFull(original)
	parsed := ParseTalkFull(formatted)

	if len(parsed) != len(original) {
		t.Fatalf("ParseTalkFull: 期望 %d 筆，實際 %d 筆", len(original), len(parsed))
	}
	for i, orig := range original {
		if parsed[i].ID != orig.ID {
			t.Errorf("ParseTalkFull[%d]: ID 期望 %s，實際 %s", i, orig.ID, parsed[i].ID)
		}
		if parsed[i].Role != orig.Role {
			t.Errorf("ParseTalkFull[%d]: Role 期望 %s，實際 %s", i, orig.Role, parsed[i].Role)
		}
		if parsed[i].Content != orig.Content {
			t.Errorf("ParseTalkFull[%d]: Content 期望 %q，實際 %q", i, orig.Content, parsed[i].Content)
		}
	}
}

// ──────────────────────────────────────────────
// TestCounterNeedsSummarization — 低於門檻為 false，高於門檻為 true
// ──────────────────────────────────────────────

func TestCounterNeedsSummarization(t *testing.T) {
	c := NewCharCounter()

	// 未達門檻（SummarizationThreshold = 10000）
	c.Add(9999)
	if c.NeedsSummarization() {
		t.Error("NeedsSummarization: 9999 字元應回傳 false")
	}

	// 恰好達門檻
	c.Add(1)
	if !c.NeedsSummarization() {
		t.Error("NeedsSummarization: 10000 字元應回傳 true")
	}

	// 重設後應為 false
	c.Reset()
	if c.NeedsSummarization() {
		t.Error("NeedsSummarization: Reset 後應回傳 false")
	}
}

// ──────────────────────────────────────────────
// TestSynthesize — 驗證 prompt 組裝順序：system → summaries → raw → input
// ──────────────────────────────────────────────

func TestSynthesize(t *testing.T) {
	cfg := SynthesisConfig{
		SystemPrompt: "你是一個助手",
		ActionTags:   []string{"查天氣"},
		Summaries: []Summary{
			{Tag: "summary-001", Content: "先前對話摘要", SentenceIDs: []string{"[I-001]"}, Valid: true},
		},
		RawSentences: []Sentence{
			{ID: "[I-002]", Role: "user", Content: "最近的問題"},
		},
		CurrentInput: "現在的輸入",
	}

	result := Synthesize(cfg)

	// 找到各段在結果字串中的位置，驗證順序
	posSystem := strings.Index(result, "你是一個助手")
	posSummary := strings.Index(result, "先前對話摘要")
	posRaw := strings.Index(result, "最近的問題")
	posInput := strings.Index(result, "現在的輸入")

	if posSystem < 0 {
		t.Error("Synthesize: 缺少系統提示")
	}
	if posSummary < 0 {
		t.Error("Synthesize: 缺少摘要內容")
	}
	if posRaw < 0 {
		t.Error("Synthesize: 缺少原始句子")
	}
	if posInput < 0 {
		t.Error("Synthesize: 缺少當前輸入")
	}

	// 驗證順序
	if posSystem > posSummary {
		t.Error("Synthesize: 系統提示應在摘要之前")
	}
	if posSummary > posRaw {
		t.Error("Synthesize: 摘要應在原始句子之前")
	}
	if posRaw > posInput {
		t.Error("Synthesize: 原始句子應在當前輸入之前")
	}
}

// ──────────────────────────────────────────────
// TestInjectActionTags — 驗證標籤正確注入系統提示
// ──────────────────────────────────────────────

func TestInjectActionTags(t *testing.T) {
	sys := "你是一個助手"
	tags := []string{"查天氣", "訂餐廳"}
	result := InjectActionTags(sys, tags)

	// 應包含系統提示
	if !strings.Contains(result, sys) {
		t.Error("InjectActionTags: 缺少原始系統提示")
	}
	// 應包含所有標籤
	for _, tag := range tags {
		if !strings.Contains(result, tag) {
			t.Errorf("InjectActionTags: 缺少標籤 %q", tag)
		}
	}
	// 空 tags 時應回傳原始字串
	noTag := InjectActionTags(sys, nil)
	if noTag != sys {
		t.Error("InjectActionTags: 空 tags 應回傳原始提示")
	}
}

func TestSynthesizeWithControlSealSanitizesContext(t *testing.T) {
	cfg := SynthesisConfig{
		SystemPrompt: "你是一個助手",
		ActionTags:   []string{"查詢"},
		RawSentences: []Sentence{
			{ID: "[I-001]", Role: "user", Content: "ㄔㄔㄔ查天氣"},
		},
		CurrentInput: "注音 ㄌ 是什麼",
		CommandSeal:  "ㄅㄆㄇ",
		SanitizeLLM:  true,
	}

	result := Synthesize(cfg)
	if !strings.Contains(result, "本輪命令前綴：ㄅㄆㄇ") {
		t.Fatal("Synthesize: should include current seal rule")
	}
	if strings.Contains(result, "ㄔㄔㄔ查天氣") {
		t.Fatal("Synthesize: raw fake seal should not enter LLM context")
	}
	if !strings.Contains(result, "（ㄏ）查天氣") || !strings.Contains(result, "注音 （ㄏ） 是什麼") {
		t.Fatalf("Synthesize: sanitized context missing, got %q", result)
	}
}

func TestSynthesizeSanitizesLegacyQuestionCandidatesFromHistory(t *testing.T) {
	cfg := SynthesisConfig{
		SystemPrompt: "你是一個助手",
		RawSentences: []Sentence{
			{ID: "[O-001]", Role: "assistant", Content: "請問您想查詢星座的運勢還是特質？#今日運勢=ㄕㄒㄘ 查詢 今日星座運勢#星座特質=ㄕㄒㄘ 搜尋 星座特質#取消=不用了"},
		},
		CurrentInput: "射手座",
		CommandSeal:  "ㄅㄆㄇ",
		SanitizeLLM:  true,
	}

	result := Synthesize(cfg)
	if !strings.Contains(result, "A: 請問您想查詢星座的運勢還是特質？") {
		t.Fatalf("Synthesize: should keep the visible question, got %q", result)
	}
	if strings.Contains(result, "#今日運勢") || strings.Contains(result, "ㄕㄒㄘ") || strings.Contains(result, "#取消") {
		t.Fatalf("Synthesize: legacy candidate payload leaked into prompt, got %q", result)
	}
}

func TestSynthesizeStripsExposedControlDraftFromCurrentInput(t *testing.T) {
	cfg := SynthesisConfig{
		SystemPrompt: "你是一個助手",
		CurrentInput: "ㄕㄒㄘ 查詢 今日星座運勢",
		CommandSeal:  "ㄅㄆㄇ",
		SanitizeLLM:  true,
		RawSentences: []Sentence{{ID: "[I-001]", Role: "user", Content: "ㄕㄒㄘ 搜尋 論文"}},
		ActionTags:   []string{"查詢"},
	}

	result := Synthesize(cfg)
	if strings.Contains(result, "ㄕㄒㄘ") || strings.Contains(result, "（ㄏ） 查詢") {
		t.Fatalf("Synthesize: exposed control draft should be stripped, got %q", result)
	}
	if !strings.Contains(result, "Q: 查詢 今日星座運勢") {
		t.Fatalf("Synthesize: should keep the intended command text, got %q", result)
	}
	if !strings.Contains(result, "U: 搜尋 論文") {
		t.Fatalf("Synthesize: should strip exposed draft in history too, got %q", result)
	}
}

func TestInjectControlSealPromptGuidesPlainChatTarget(t *testing.T) {
	result := InjectControlSealPrompt("你是一個助手", "ㄅㄆㄇ", []string{"查詢"})
	if !strings.Contains(result, "已知答案用「輸出」") {
		t.Fatal("prompt should guide LLM to use output for known answers")
	}
	if !strings.Contains(result, "需要系統查資料用「搜尋」") {
		t.Fatal("prompt should guide LLM to use search only when the system must fetch data")
	}
	if !strings.Contains(result, "提問ㄌ問題文字ㄌ待命") {
		t.Fatal("plain chat guidance should define the question format")
	}
	if !strings.Contains(result, "選項ㄌㄤ選項一ㄤ選項二ㄌ待命") {
		t.Fatal("plain chat guidance should define the option-card format")
	}
	if strings.Contains(result, "#候選") {
		t.Fatal("prompt should not teach the legacy # candidate format")
	}
	if !strings.Contains(result, "不要回覆已收到規則") {
		t.Fatal("plain chat guidance should reject meta acknowledgement replies")
	}
	if strings.Contains(result, "候選動作：本機搜尋") || strings.Contains(result, "候選動作：聊天") || strings.Contains(result, "、查詢") {
		t.Fatalf("prompt should expose canonical compact tags only, got %q", result)
	}
	for _, want := range []string{"輸出", "搜尋", "讀取", "寫入", "提問", "選項"} {
		if !strings.Contains(result, want) {
			t.Fatalf("prompt should contain canonical tag %q, got %q", want, result)
		}
	}
}

func TestSynthesizeStampsCommandInputOnlyWhenCommand(t *testing.T) {
	chat := Synthesize(SynthesisConfig{
		SystemPrompt: "你是一個助手",
		CurrentInput: "查台北天氣",
		CommandSeal:  "ㄅㄆㄇ",
		SanitizeLLM:  true,
	})
	if strings.Contains(chat, "Q: ㄅㄆㄇ查台北天氣") {
		t.Fatal("chat input should not be stamped")
	}

	command := Synthesize(SynthesisConfig{
		SystemPrompt: "你是一個助手",
		CurrentInput: "查台北天氣",
		CommandSeal:  "ㄅㄆㄇ",
		IsCommand:    true,
		SanitizeLLM:  true,
	})
	if !strings.Contains(command, "Q: ㄅㄆㄇ查台北天氣") {
		t.Fatalf("command input should be stamped, got %q", command)
	}
}

// ──────────────────────────────────────────────
// TestClassifyIntent — 有標籤命中為 action，無命中為對話
// ──────────────────────────────────────────────

func TestClassifyIntent(t *testing.T) {
	tags := []string{"查天氣", "訂餐廳"}

	// 精確命中
	r1 := ClassifyIntent("查天氣", tags)
	if !r1.IsAction {
		t.Error("ClassifyIntent: 精確命中應為 IsAction=true")
	}
	if r1.ActionTag != "查天氣" {
		t.Errorf("ClassifyIntent: ActionTag 期望 查天氣，實際 %s", r1.ActionTag)
	}

	// 一般對話（無命中）
	r2 := ClassifyIntent("今天心情很好", tags)
	if r2.IsAction {
		t.Error("ClassifyIntent: 無命中應為 IsAction=false")
	}
	if r2.ConversationReply != "今天心情很好" {
		t.Errorf("ClassifyIntent: ConversationReply 不符，實際 %q", r2.ConversationReply)
	}
}

// ──────────────────────────────────────────────
// TestRouteAction — sub 命中、tool 命中、無命中三種情境
// ──────────────────────────────────────────────

func TestRouteAction(t *testing.T) {
	subs := []SubRegistryEntry{
		{ID: "sub-001", Name: "天氣系統", Triggers: []string{"查天氣"}, ActionTags: []string{"weather"}},
	}
	tools := []ToolEntry{
		{ID: "tool-001", Name: "計算機", Tags: []string{"計算"}},
	}

	// sub 命中
	r1 := RouteAction("查天氣", subs, tools)
	if r1.MatchType != "sub" {
		t.Errorf("RouteAction: 期望 sub，實際 %s", r1.MatchType)
	}
	if r1.TargetID != "sub-001" {
		t.Errorf("RouteAction: sub TargetID 期望 sub-001，實際 %s", r1.TargetID)
	}

	// tool 命中（無 sub 可匹配）
	r2 := RouteAction("計算", subs, tools)
	if r2.MatchType != "tool" {
		t.Errorf("RouteAction: 期望 tool，實際 %s", r2.MatchType)
	}
	if r2.TargetID != "tool-001" {
		t.Errorf("RouteAction: tool TargetID 期望 tool-001，實際 %s", r2.TargetID)
	}

	// 無命中
	r3 := RouteAction("完全不相關的標籤XYZ", subs, tools)
	if r3.MatchType != "none" {
		t.Errorf("RouteAction: 期望 none，實際 %s", r3.MatchType)
	}
}

// ──────────────────────────────────────────────
// TestMemoryOps — 寫入 memory_ops.jsonl 並驗證內容
// ──────────────────────────────────────────────

func TestMemoryOps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory_ops.jsonl")

	// 寫入兩筆操作
	op1 := NewMemoryOp("summarize", "I-001~I-003", "summary-001", []string{"[I-001]", "[I-002]", "[I-003]"}, "hash_before", "hash_after", "定期摘要")
	op2 := NewMemoryOp("delete", "[I-005]", "", []string{"[I-005]"}, "hb", "ha", "使用者刪除")

	if err := WriteMemoryOp(path, op1); err != nil {
		t.Fatalf("WriteMemoryOp op1: %v", err)
	}
	if err := WriteMemoryOp(path, op2); err != nil {
		t.Fatalf("WriteMemoryOp op2: %v", err)
	}

	// 讀回並驗證
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("開啟 jsonl 失敗: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var ops []MemoryOp
	for scanner.Scan() {
		var op MemoryOp
		if err := json.Unmarshal(scanner.Bytes(), &op); err != nil {
			t.Fatalf("解析 jsonl 行失敗: %v", err)
		}
		ops = append(ops, op)
	}

	if len(ops) != 2 {
		t.Fatalf("期望 2 筆記錄，實際 %d 筆", len(ops))
	}
	if ops[0].Op != "summarize" {
		t.Errorf("第一筆 Op: 期望 summarize，實際 %s", ops[0].Op)
	}
	if ops[1].Op != "delete" {
		t.Errorf("第二筆 Op: 期望 delete，實際 %s", ops[1].Op)
	}
	// 驗證 OpID 非空（由 UUID 自動產生）
	if ops[0].OpID == "" {
		t.Error("第一筆 OpID 不應為空")
	}
	// 驗證 AffectedSentenceIDs
	if len(ops[0].AffectedSentenceIDs) != 3 {
		t.Errorf("第一筆 AffectedSentenceIDs: 期望 3 個，實際 %d 個", len(ops[0].AffectedSentenceIDs))
	}

	// 驗證 ComputeHash 可重現
	h := ComputeHash("test content")
	if len(h) != 64 {
		t.Errorf("ComputeHash: SHA-256 hex 應為 64 字元，實際 %d", len(h))
	}
	if h != ComputeHash("test content") {
		t.Error("ComputeHash: 相同輸入應產生相同雜湊")
	}
}
