package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConsoleHaorasFromProjectReadsSavedSubagents(t *testing.T) {
	projectRoot := t.TempDir()
	callableDir := filepath.Join(projectRoot, "subagents", "callable")
	if err := os.MkdirAll(filepath.Join(callableDir, "sub-20260516-120000"), 0o755); err != nil {
		t.Fatalf("mkdir first sub: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(callableDir, "sub-20260516-120500"), 0o755); err != nil {
		t.Fatalf("mkdir second sub: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(callableDir, "sub-20260516-120000", "sub_meta.json"),
		[]byte(`{"id":"sub-20260516-120000","name":"天氣查詢流程","created_at":"2026-05-16T12:00:00+08:00"}`),
		0o644,
	); err != nil {
		t.Fatalf("write first metadata: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(callableDir, "sub-20260516-120500", "sub_meta.json"),
		[]byte(`{"id":"sub-20260516-120500","name":"資料整理流程","created_at":"2026-05-16T12:05:00+08:00"}`),
		0o644,
	); err != nil {
		t.Fatalf("write second metadata: %v", err)
	}

	got := consoleHaorasFromProject(projectRoot)
	want := []string{"主haㄌer", "天氣查詢流程", "資料整理流程"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("consoleHaorasFromProject() = %#v, want %#v", got, want)
	}
}

func TestConsoleHaorasFromProjectFallsBackToMainOnly(t *testing.T) {
	got := consoleHaorasFromProject(t.TempDir())
	want := []string{"主haㄌer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("consoleHaorasFromProject() = %#v, want %#v", got, want)
	}
}

func TestCreateSubagentCreatesCallableMemoryAndTabOrder(t *testing.T) {
	projectRoot := t.TempDir()
	created, err := createSubagentInProject(projectRoot, "手動新流程")
	if err != nil {
		t.Fatalf("CreateSubagent: %v", err)
	}

	if created.ID == "" || created.SubDir == "" || created.MemoryDir == "" {
		t.Fatalf("created subagent has empty fields: %#v", created)
	}
	for _, path := range []string{
		created.SubDir,
		created.MemoryDir,
		filepath.Join(created.SubDir, "dag"),
		filepath.Join(created.SubDir, "tool_history"),
		filepath.Join(created.MemoryDir, "talk_full.md"),
		filepath.Join(created.SubDir, "sub_meta.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected path %s to exist: %v", path, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(created.SubDir, "sub_meta.json"))
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	var meta struct {
		Name        string `json:"name"`
		CreatedFrom string `json:"created_from"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	if meta.Name != "手動新流程" || meta.CreatedFrom != "manual_create" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}

	got := consoleHaorasFromProject(projectRoot)
	want := []string{"主haㄌer", "手動新流程"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("consoleHaorasFromProject() = %#v, want %#v", got, want)
	}
}

func TestRenameSubagentUpdatesMetadata(t *testing.T) {
	projectRoot := t.TempDir()
	if _, err := createSubagentInProject(projectRoot, "原本名稱"); err != nil {
		t.Fatalf("createSubagentInProject: %v", err)
	}

	renamed, err := renameSubagentInProject(projectRoot, "原本名稱", "新名稱")
	if err != nil {
		t.Fatalf("renameSubagentInProject: %v", err)
	}
	if renamed.Name != "新名稱" {
		t.Fatalf("renamed.Name = %q, want 新名稱", renamed.Name)
	}

	got := consoleHaorasFromProject(projectRoot)
	want := []string{"主haㄌer", "新名稱"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("consoleHaorasFromProject() = %#v, want %#v", got, want)
	}
}
