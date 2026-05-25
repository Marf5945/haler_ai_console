package main

import (
	"context"
	"slices"
	"strings"
	"testing"

	"ui_console/data/conversation"
	"ui_console/orchestration/skill_step"
	"ui_console/shared/tools"
)

type recordingActionTagAdapter struct {
	tags []string
}

func (r *recordingActionTagAdapter) SetActionTags(tags []string) {
	r.tags = append([]string(nil), tags...)
}

func (r *recordingActionTagAdapter) SendMessage(options skill_step.CLIMessageOptions) (skill_step.CLIResponse, error) {
	return skill_step.CLIResponse{Text: "ok"}, nil
}

func TestCollectActionTagsIncludesBuiltinSkillsAndTools(t *testing.T) {
	archive := skill_step.NewArchiveService(t.TempDir())
	router := skill_step.NewRouter(archive)
	skill_step.RegisterDocumentBuiltins(router)
	toolSvc := tools.NewService()
	toolSvc.AddTool(tools.Tool{
		ID:         "line",
		Title:      "LINE",
		Kind:       "connector",
		ActionTags: []string{"傳送", "聊天"},
	})
	app := &App{toolsService: toolSvc, skillRouter: router}

	tags := app.collectActionTags()
	for _, want := range []string{"讀取", "匯入", "建立", "傳送", "搜尋"} {
		if !slices.Contains(tags, want) {
			t.Fatalf("missing action tag %q in %#v", want, tags)
		}
	}
	if slices.Contains(tags, "聊天") {
		t.Fatalf("reserved tag should not be exposed as dynamic tool tag: %#v", tags)
	}

	prompt := conversation.Synthesize(conversation.SynthesisConfig{
		SystemPrompt: "P",
		ActionTags:   tags,
		CurrentInput: "讀取文件",
		CommandSeal:  "ㄅㄆㄇ",
		IsCommand:    true,
		SanitizeLLM:  true,
		RawSentences: nil,
		Summaries:    nil,
	})
	if !strings.Contains(prompt, "候選動作") || !strings.Contains(prompt, "讀取") || !strings.Contains(prompt, "傳送") {
		t.Fatalf("prompt did not include dynamic action tags: %s", prompt)
	}
}

func TestSyncActionTagsToCLIAdapter(t *testing.T) {
	archive := skill_step.NewArchiveService(t.TempDir())
	router := skill_step.NewRouter(archive)
	skill_step.RegisterDocumentBuiltins(router)
	adapter := &recordingActionTagAdapter{}
	app := &App{skillRouter: router, toolsService: tools.NewService(), cliAdapter: adapter}
	app.ctx = context.Background()

	tags := app.syncActionTagsToCLIAdapter("test-trace")
	if len(tags) == 0 {
		t.Fatal("expected action tags")
	}
	if !slices.Equal(tags, adapter.tags) {
		t.Fatalf("adapter tags = %#v, want %#v", adapter.tags, tags)
	}
}
