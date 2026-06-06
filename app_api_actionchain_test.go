package main

import (
	"strings"
	"testing"
	"time"

	"ui_console/shared/settings"
)

func TestBuildAPIActionChainPromptIncludesControllerRules(t *testing.T) {
	app := &App{}
	basePrompt := app.buildMainComposerPrompt(settings.Persona{Name: "測試"})
	prompt := buildAPIActionChainPrompt(basePrompt, []string{"查詢"}, "現在幾點")

	// API adapters do not have shell access, so the prompt must carry current time.
	if !strings.Contains(prompt, "now=") {
		t.Fatal("API action-chain prompt should include now= time injection")
	}
	if !strings.Contains(prompt, "本輪命令前綴") {
		t.Fatal("API action-chain prompt should include control seal rules")
	}
	if !strings.Contains(prompt, "動作ㄌ目標ㄌ下一步") {
		t.Fatal("API action-chain prompt should include action-chain output format")
	}
	if !strings.Contains(prompt, "選項ㄌㄤ選項一ㄤ選項二ㄌ待命") {
		t.Fatal("API action-chain prompt should include option-card format")
	}
	if strings.Contains(prompt, "#候選") {
		t.Fatal("API action-chain prompt should not teach legacy # candidate format")
	}
	if strings.Contains(prompt, "候選動作：查詢") {
		t.Fatal("API action-chain prompt should expose compact canonical action tags")
	}
	if !strings.Contains(prompt, "已知答案用「輸出」") {
		t.Fatal("API action-chain prompt should define output as the action for known answers")
	}
	if strings.Contains(prompt, "不分析") {
		t.Fatal("main composer prompt should allow history/status analysis for continuity")
	}
	if !strings.Contains(prompt, "H可用於續聊、解析代名詞/檔名指代") {
		t.Fatal("main composer prompt should explain how H may be used")
	}
	if !strings.Contains(prompt, "H不複述") {
		t.Fatal("main composer prompt should keep history concise and non-repetitive")
	}
}

func TestFormatPromptNowIncludesUTCOffset(t *testing.T) {
	text := formatPromptNow(time.Date(2026, 5, 23, 16, 10, 0, 0, time.FixedZone("TST", 8*60*60)))
	if !strings.Contains(text, "2026-05-23 16:10") || !strings.Contains(text, "TST") || !strings.Contains(text, "UTC+08:00") {
		t.Fatalf("prompt time should include wall time, zone, and offset, got %q", text)
	}
}

func TestToolRoutingDecisionAcceptsLocalModelChatSeparatorTypo(t *testing.T) {
	decision := parseToolRoutingDecision(" 閒聊ㄎ👋 hello")
	if decision.Kind != toolRoutingDecisionChat || decision.Text != "👋 hello" {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestInspectorPromptAndReplyHideInternalLaneMarkers(t *testing.T) {
	app := &App{}
	prompt := app.buildInspectorPrompt(settings.Persona{Name: "測試"}, "你好")
	if strings.Contains(prompt, "lane=top") || strings.Contains(prompt, "只用top") {
		t.Fatalf("inspector prompt should not expose copyable lane markers: %q", prompt)
	}
	if got := cleanInspectorReply("嘿，有什麼能幫到你的嗎？ _top"); got != "嘿，有什麼能幫到你的嗎？" {
		t.Fatalf("reply cleaner should strip trailing top marker, got %q", got)
	}
	if got := cleanInspectorReply("我在 lane=top"); got != "我在" {
		t.Fatalf("reply cleaner should strip trailing lane marker, got %q", got)
	}
}

func TestResolveActionChainResponseForAPIBuiltIn(t *testing.T) {
	app := &App{}
	resp := app.resolveActionChainResponse("輸出ㄌ2026-05-23 14:27 CSTㄌ待命", nil, "trace-test", "session-test")

	// Built-in output strips protocol markers before rendering the chat bubble.
	if resp.Text != "2026-05-23 14:27 CST" {
		t.Fatalf("display text mismatch: %q", resp.Text)
	}
	if resp.Action != "輸出" || resp.Target != "2026-05-23 14:27 CST" || resp.Next != "待命" {
		t.Fatalf("parsed action-chain mismatch: %+v", resp)
	}
}

func TestQuestionPayloadPrefersNextCandidatePayload(t *testing.T) {
	payload := questionPayload("搜尋文件論文", "請提供關鍵字或主題#論文搜尋=input:搜尋 論文 主題：#文件搜尋=input:搜尋 文件 關鍵字：#取消=不用了")
	if !strings.HasPrefix(payload, "請提供關鍵字或主題#") {
		t.Fatalf("question payload should come from next field, got %q", payload)
	}
	question, candidates := floatingCandidatesFromQuestionTarget(payload)
	if question != "請提供關鍵字或主題" || len(candidates) != 3 {
		t.Fatalf("unexpected question card: question=%q candidates=%#v", question, candidates)
	}
}

func TestResolveActionChainResponseForOptions(t *testing.T) {
	app := &App{}
	resp := app.resolveActionChainResponse("選項ㄌㄤ台北ㄤ本地ㄤ台中ㄌ等待", nil, "trace-test", "session-test")
	if resp.Action != "選項" || resp.Next != "待命" {
		t.Fatalf("parsed action-chain mismatch: %+v", resp)
	}
	if resp.Text != "ㄤ台北ㄤ本地ㄤ台中" {
		t.Fatalf("resolve should preserve target for App post-processing, got %q", resp.Text)
	}
}

func TestBuildLocalModelPromptDiscouragesFieldLabelsAndGreetingQuestions(t *testing.T) {
	prompt := buildLocalModelPrompt("", []string{"\u8f38\u51fa", "\u64cd\u4f5c"}, "\u4f60\u597d")
	for _, want := range []string{
		"\u4e0d\u8981\u5beb \u52d5\u4f5c:",
		"\u4e0d\u8981\u52a0 \u5167\u5bb9:",
		"\u4e00\u822c\u804a\u5929",
		"\u64cd\u4f5c=\u53ea\u4ee3\u8868\u57f7\u884c\u6216\u91cd\u73fe\u5df2\u4fdd\u5b58\u7684\u87a2\u5e55 replay \u64cd\u4f5c",
		"\u4e0d\u8981\u9078 \u64cd\u4f5c",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("local model prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestAssembleLocalResponseStripsFieldLabels(t *testing.T) {
	chain := assembleLocalResponse("\u52d5\u4f5c:\u8f38\u51fa\n\u5167\u5bb9:\uff1a\u4f60\u662f\u5426\u9700\u8981\u67d0\u500b\u7279\u5b9a\u7684\u8cc7\u8a0a\u6216\u5e6b\u52a9\uff1f\n\u4e0b\u4e00\u6b65:\u5f85\u547d")
	if chain.Action != "\u8f38\u51fa" {
		t.Fatalf("Action = %q", chain.Action)
	}
	if chain.Target != "\u4f60\u662f\u5426\u9700\u8981\u67d0\u500b\u7279\u5b9a\u7684\u8cc7\u8a0a\u6216\u5e6b\u52a9\uff1f" {
		t.Fatalf("Target = %q", chain.Target)
	}
	if chain.Next != "\u5f85\u547d" {
		t.Fatalf("Next = %q", chain.Next)
	}
}

func TestToolRoutingPromptUsesCompactRoutingContract(t *testing.T) {
	prompt := buildToolRoutingDecisionPrompt("", "\u4f60\u597d", "")
	for _, want := range []string{
		"\u64cd\u4f5c\u310c<\u5019\u9078tag/\u540d\u7a31/\u95dc\u9375\u8a5e>\u310c\u5f85\u547d",
		"\u7db2\u8def\u310c<\u641c\u5c0b\u95dc\u9375\u5b57>\u310c\u5f85\u547d",
		"\u9700\u8981\u5de5\u5177\uff1a\u9700\u8981\u5176\u4ed6\u5de5\u5177\uff0c\u6216\u5019\u9078\u4e0d\u8db3\u4f46\u4e0d\u50cf\u9592\u804a\u3002",
		"\u7db2\u8def\u8def\u7531\uff1a\u51e1\u9700\u7db2\u8def\u641c\u5c0b\u624d\u80fd\u5224\u65b7\u7684\u8b8a\u52d5\u8cc7\u6599",
		"\u53ea\u6709 recent_operations \u4e0d\u7b97\u660e\u78ba",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("tool routing prompt missing %q:\n%s", want, prompt)
		}
	}
}
