package main

import (
	"os"
	"testing"

	"ui_console/orchestration/skill_step"
)

// 拆解：單一意圖只輸出一步（對應「單句不進多步」的不過度拆解期望）。
func TestParseDecomposeSingleStep(t *testing.T) {
	_, nodes, err := parseDecomposePlan(`{"title":"t","steps":[{"id":"s1","text":"用產出電料Bom處理載入的檔案"}]}`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ExecutorType != "chat_route" || nodes[0].Target == "" {
		t.Fatalf("want 1 chat_route node with target, got %+v", nodes)
	}
}

// 兩個獨立意圖 → 兩個 chat_route 節點（這正是吞整句 bug 要解的情境）。
func TestParseDecomposeTwoIntents(t *testing.T) {
	_, nodes, err := parseDecomposePlan(`{"title":"t","steps":[
		{"id":"s1","text":"你有看到我放入的檔案嗎"},
		{"id":"s2","text":"用產出電料Bom產出料表","depends_on":["s1"]}]}`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("want 2 nodes, got %d", len(nodes))
	}
	if len(nodes[1].Dependencies) != 1 || nodes[1].Dependencies[0] != "s1" {
		t.Fatalf("dep not preserved: %+v", nodes[1].Dependencies)
	}
}

// 空步驟要被丟掉。
func TestParseDecomposeDropEmpty(t *testing.T) {
	_, nodes, err := parseDecomposePlan(`{"steps":[{"id":"s1","text":"做A"},{"id":"s2","text":"  "}]}`)
	if err != nil || len(nodes) != 1 {
		t.Fatalf("empty step should be dropped, got nodes=%d err=%v", len(nodes), err)
	}
}

// 相依成環 → 整批退回 error（寧可退回單句流程）。
func TestParseDecomposeRejectsCycle(t *testing.T) {
	_, _, err := parseDecomposePlan(`{"steps":[
		{"id":"a","text":"A","depends_on":["b"]},
		{"id":"b","text":"B","depends_on":["a"]}]}`)
	if err == nil {
		t.Fatal("cyclic dependency should be rejected")
	}
}

// 指向不存在 id 的相依要被清掉。
func TestParseDecomposeDropsDanglingDep(t *testing.T) {
	_, nodes, err := parseDecomposePlan(`{"steps":[{"id":"s1","text":"A","depends_on":["nope"]}]}`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(nodes[0].Dependencies) != 0 {
		t.Fatalf("dangling dep should be dropped, got %+v", nodes[0].Dependencies)
	}
}

// 超過上限要被截到 decomposeMaxSteps。
func TestParseDecomposeCapsSteps(t *testing.T) {
	raw := `{"steps":[`
	for i := 0; i < decomposeMaxSteps+5; i++ {
		if i > 0 {
			raw += ","
		}
		raw += `{"id":"s` + itoaTest(i) + `","text":"step"}`
	}
	raw += `]}`
	_, nodes, err := parseDecomposePlan(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(nodes) != decomposeMaxSteps {
		t.Fatalf("want capped at %d, got %d", decomposeMaxSteps, len(nodes))
	}
}

// chatRouteNeedsUser：NeedsUser 或 提問 視為等使用者，其餘否。
func TestChatRouteNeedsUser(t *testing.T) {
	if !chatRouteNeedsUser(&skill_step.CLIResponse{NeedsUser: true}) {
		t.Fatal("NeedsUser=true should need user")
	}
	if !chatRouteNeedsUser(&skill_step.CLIResponse{Action: "提問"}) {
		t.Fatal("action=提問 should need user")
	}
	if chatRouteNeedsUser(&skill_step.CLIResponse{Text: "結果"}) {
		t.Fatal("plain result should not need user")
	}
}

// feature flag 預設關閉 → 行為與舊版相同（不自動進拆解）。
func TestDecomposeDisabledByDefault(t *testing.T) {
	os.Unsetenv("AI_CONSOLE_DECOMPOSE")
	if decomposeEnabled() {
		t.Fatal("decompose should be off by default")
	}
	os.Setenv("AI_CONSOLE_DECOMPOSE", "1")
	defer os.Unsetenv("AI_CONSOLE_DECOMPOSE")
	if !decomposeEnabled() {
		t.Fatal("decompose should be on when flag=1")
	}
}

func TestChatRouteTraceAvoidsTaskGuard(t *testing.T) {
	if isTaskProgressTraceID("chatroute-run1-s1") {
		t.Fatal("chat_route trace must not be treated as internal task progress trace")
	}
	if !isTaskProgressTraceID("task-node-s1") {
		t.Fatal("task-node trace should still be treated as internal task progress trace")
	}
}

func itoaTest(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
