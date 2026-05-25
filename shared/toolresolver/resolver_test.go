package toolresolver

import "testing"

func TestResolveToolUsesRecentLineContext(t *testing.T) {
	tools := []Tool{
		{ID: "line", Name: "LINE", Kind: "connector", ActionTags: []string{"傳送"}},
		{ID: "email", Name: "Email", Kind: "connector", ActionTags: []string{"傳送"}},
	}
	result := ResolveTool("傳送", "小明", Context{RecentChannel: "line"}, tools)
	if result.Status != StatusSelected || result.Tool.ID != "line" {
		t.Fatalf("resolution = %#v", result)
	}
}

func TestResolveToolAsksWhenMultipleToolsMatch(t *testing.T) {
	tools := []Tool{
		{ID: "line", Name: "LINE", ActionTags: []string{"傳送"}},
		{ID: "slack", Name: "Slack", ActionTags: []string{"傳送"}},
	}
	result := ResolveTool("傳送", "小明", Context{}, tools)
	if result.Status != StatusAmbiguous || len(result.Candidates) != 2 {
		t.Fatalf("resolution = %#v", result)
	}
}
