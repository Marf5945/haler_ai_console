package main

import (
	"strings"
	"testing"

	"ui_console/orchestration/dag"
)

// 落盤 → 讀回 roundtrip；壞行跳過。
func TestTaskExperienceAppendLoad(t *testing.T) {
	root := t.TempDir()
	for _, exp := range []TaskExperience{
		{RunID: "r1", Title: "產出電料BOM", Status: "failed", FailedNodeTitle: "寫入xlsx", FailureCategory: "tool_error", CreatedAt: "t1"},
		{RunID: "r2", Title: "整理會議紀錄", Status: "completed", NodeCount: 3, CreatedAt: "t2"},
	} {
		if err := appendTaskExperience(root, exp); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	got := loadRecentTaskExperiences(root, 10)
	if len(got) != 2 || got[0].RunID != "r1" || got[1].Status != "completed" {
		t.Fatalf("roundtrip wrong: %+v", got)
	}
}

// 相似度比對：2-gram 重疊、失敗加權、不相干不選。
func TestMatchTaskExperiences(t *testing.T) {
	exps := []TaskExperience{
		{Title: "產出電料BOM", Status: "failed", FailedNodeTitle: "寫入xlsx"},
		{Title: "產出電料清單", Status: "completed"},
		{Title: "翻譯日文信件", Status: "completed"},
	}
	got := matchTaskExperiences("幫我產出電料BOM表", exps, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got))
	}
	if got[0].Status != "failed" { // 失敗經驗加權應排前
		t.Fatalf("failed experience should rank first: %+v", got)
	}
	for _, exp := range got {
		if exp.Title == "翻譯日文信件" {
			t.Fatal("unrelated experience must not match")
		}
	}
}

// 注入 flag：關 → 空字串；開 → 含避雷提示。
func TestTaskExperienceDigestFlag(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	app := &App{}
	t.Setenv("AI_CONSOLE_TASK_EXPERIENCE", "0")
	if app.taskExperienceDigest("產出電料BOM") != "" {
		t.Fatal("flag off must return empty")
	}
	t.Setenv("AI_CONSOLE_TASK_EXPERIENCE", "1")
	if app.taskExperienceDigest("產出電料BOM") != "" {
		t.Fatal("no experiences yet must return empty")
	}
}

// 終局萃取：failed run 帶出失敗節點與分類；取消不記。
func TestRecordTaskExperienceExtractsFailure(t *testing.T) {
	root := t.TempDir()
	run := &dag.DAGRun{ID: "r9", Title: "測試任務", Status: "failed", Nodes: []dag.DAGNode{
		{ID: "n1", Title: "搜尋", Status: dag.StatusSucceeded},
		{ID: "n2", Title: "寫入報表", Status: dag.StatusFailed, FailureCategory: "tool_error"},
	}}
	recordTaskExperience(root, run)
	got := loadRecentTaskExperiences(root, 5)
	if len(got) != 1 || got[0].FailedNodeTitle != "寫入報表" || got[0].FailureCategory != "tool_error" {
		t.Fatalf("extract wrong: %+v", got)
	}
	run.Status = "cancelled"
	recordTaskExperience(root, run)
	if len(loadRecentTaskExperiences(root, 5)) != 1 {
		t.Fatal("cancelled must not be recorded")
	}
}

// 依賴 context 摘要式降階：超限時舊依賴變 digest 行、最後依賴保留全文。
func TestBuildTaskDependencyContextDigest(t *testing.T) {
	big := strings.Repeat("資料內容", 4096) // 單筆 48KB 級
	run := &dag.DAGRun{Nodes: []dag.DAGNode{
		{ID: "d1", Title: "舊步驟", Status: dag.StatusSucceeded, ResultSummary: big},
		{ID: "d2", Title: "新步驟", Status: dag.StatusSucceeded, ResultSummary: "最新結果重點"},
		{ID: "n", Dependencies: []string{"d1", "d2"}},
	}}
	got := buildTaskDependencyContext(run, run.Nodes[2])
	if !strings.Contains(got, "result(摘)") || !strings.Contains(got, "[older dependency results digested]") {
		t.Fatal("old dep should be digested")
	}
	if !strings.Contains(got, "最新結果重點") {
		t.Fatal("last dep must keep full result")
	}
	if len(got) > taskDependencyContextLimit+1024 {
		t.Fatalf("still over limit: %d", len(got))
	}
	// 未超限 → 全文照舊
	small := &dag.DAGRun{Nodes: []dag.DAGNode{
		{ID: "d1", Title: "步驟", Status: dag.StatusSucceeded, ResultSummary: "簡短結果"},
		{ID: "n", Dependencies: []string{"d1"}},
	}}
	if got := buildTaskDependencyContext(small, small.Nodes[1]); strings.Contains(got, "(摘)") {
		t.Fatal("under limit should keep full chunks")
	}
}
