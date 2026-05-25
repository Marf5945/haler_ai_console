package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchReferenceFilesForPromptUsesFilenameAndSummary(t *testing.T) {
	root := t.TempDir()
	referenceDir := filepath.Join(root, "data", "references", "files")
	if err := os.MkdirAll(referenceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(referenceDir, "試試看.txt"), []byte("可以試試看的檔案"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(referenceDir, "不相關.txt"), []byte("完全不同的內容"), 0o600); err != nil {
		t.Fatal(err)
	}

	matches, err := matchReferenceFilesForPrompt("你有找到試試看的檔案嗎", root)
	if err != nil {
		t.Fatalf("matchReferenceFilesForPrompt: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one filename match, got %#v", matches)
	}
	if matches[0].Name != "試試看.txt" {
		t.Fatalf("expected 試試看.txt, got %#v", matches[0])
	}
	if !strings.Contains(matches[0].Summary, "可以試試看") {
		t.Fatalf("summary not loaded: %#v", matches[0])
	}
	if !matches[0].Exists {
		t.Fatalf("matched file should be marked as currently existing: %#v", matches[0])
	}
}

func TestFormatReferencePromptContextIncludesRules(t *testing.T) {
	out := formatReferencePromptContext([]string{"試試看"}, []referencePromptMatch{{
		Name:    "試試看.txt",
		Summary: "可以試試看的檔案",
		Score:   100,
		Exists:  true,
	}})
	if !strings.Contains(out, "檔名=試試看.txt") || !strings.Contains(out, "狀態=目前存在於引用文件庫") || !strings.Contains(out, "摘要（400字）=可以試試看的檔案") {
		t.Fatalf("context missing reference facts: %s", out)
	}
	if !strings.Contains(out, "檔名可省略副檔名") {
		t.Fatalf("context missing matching rule: %s", out)
	}
	if !strings.Contains(out, "不可用H中的舊結果回答找得到") {
		t.Fatalf("context should make fresh scan override stale history: %s", out)
	}
}

func TestReferencePromptContextRescansPreviousTarget(t *testing.T) {
	root := t.TempDir()
	referenceDir := filepath.Join(root, "data", "references", "files")
	if err := os.MkdirAll(referenceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(referenceDir, "試試看.txt")
	if err := os.WriteFile(path, []byte("可以試試看的檔案"), 0o600); err != nil {
		t.Fatal(err)
	}

	first, err := buildReferencePromptContextFromRoot("幫我找試試看的文檔", root, nil, referenceSearchPlan{Search: true, Keywords: []string{"試試看"}})
	if err != nil {
		t.Fatalf("first context: %v", err)
	}
	if len(first.Targets) != 1 || first.Targets[0] != "試試看.txt" {
		t.Fatalf("first turn should remember matched target: %#v", first.Targets)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	second, err := buildReferencePromptContextFromRoot("那你還找得到嗎？", root, first.Targets, referenceSearchPlan{Search: true, Keywords: []string{"試試看.txt"}})
	if err != nil {
		t.Fatalf("second context: %v", err)
	}
	if !strings.Contains(second.Context, "檔名=試試看.txt") || !strings.Contains(second.Context, "狀態=目前不存在於引用文件庫") {
		t.Fatalf("follow-up should rescan and report missing target: %s", second.Context)
	}
}

func TestParseReferenceSearchPlan(t *testing.T) {
	plan := parseReferenceSearchPlan("```json\n{\"search\":true,\"keywords\":[\"甜點\",\"食譜\"]}\n```")
	if !plan.Search || len(plan.Keywords) != 2 || plan.Keywords[0] != "甜點" || plan.Keywords[1] != "食譜" {
		t.Fatalf("unexpected parsed plan: %#v", plan)
	}
	plan = parseReferenceSearchPlan("{\"search\":false,\"keywords\":[\"試試看\"]}")
	if plan.Search || len(plan.Keywords) != 0 {
		t.Fatalf("false search should discard keywords: %#v", plan)
	}
}
