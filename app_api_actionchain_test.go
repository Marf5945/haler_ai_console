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
