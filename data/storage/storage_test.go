package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// 測試專案目錄結構建立（§17.2）
func TestEnsureProjectLayout(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureProjectLayout(tmpDir, "test-project")
	if err != nil {
		t.Fatalf("EnsureProjectLayout failed: %v", err)
	}

	// 驗證關鍵目錄存在
	expectedDirs := []string{
		"memory",
		"dag_runs",
		"runtime/temp_sessions",
		"runtime/action_results",
		"runtime/crash_recovery",
		"controlled_trust",
		"controlled_trust/draft_sandbox_runs",
		"execution_hook_runs",
		"visual_learning/learning_runs",
		"subagents/callable",
		"subagents/candidates",
		"source_trust",
		"review",
	}
	root := ProjectRoot(tmpDir, "test-project")
	for _, dir := range expectedDirs {
		fullPath := filepath.Join(root, dir)
		if info, err := os.Stat(fullPath); err != nil || !info.IsDir() {
			t.Errorf("目錄 %s 應該存在", dir)
		}
	}

	// 驗證關鍵檔案存在
	expectedFiles := []string{
		"runtime/purge_manifest.json",
		"controlled_trust/trusted_session_scope.json",
		"memory/memory_manifest.json",
		"source_trust/project_source_allowlist.json",
		"review/review_inbox.json",
		"review/review_decision_log.jsonl",
	}
	for _, file := range expectedFiles {
		fullPath := filepath.Join(root, file)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("檔案 %s 應該存在", file)
		}
	}
}

// 測試重複呼叫不覆寫現有檔案
func TestEnsureProjectLayoutIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// 第一次建立
	if err := EnsureProjectLayout(tmpDir, "idempotent"); err != nil {
		t.Fatal(err)
	}

	// 寫入自定內容
	inboxPath := filepath.Join(ProjectRoot(tmpDir, "idempotent"), "review/review_inbox.json")
	os.WriteFile(inboxPath, []byte(`[{"id":"existing"}]`), 0644)

	// 第二次建立不應覆寫
	if err := EnsureProjectLayout(tmpDir, "idempotent"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(inboxPath)
	if string(data) != `[{"id":"existing"}]` {
		t.Error("EnsureProjectLayout 不應覆寫現有檔案")
	}
}

// 測試 Persona 目錄結構建立（§17.3）
func TestEnsurePersonaLayout(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsurePersonaLayout(tmpDir, "main-persona")
	if err != nil {
		t.Fatalf("EnsurePersonaLayout failed: %v", err)
	}

	avatarDir := filepath.Join(PersonaRoot(tmpDir, "main-persona"), "avatar")
	if info, err := os.Stat(avatarDir); err != nil || !info.IsDir() {
		t.Error("avatar 目錄應該存在")
	}
}

// 測試路徑根函式
func TestRootPaths(t *testing.T) {
	base := "/app"
	if got := ProjectRoot(base, "demo"); got != filepath.Join("/app", "data", "projects", "demo") {
		t.Errorf("ProjectRoot = %s", got)
	}
	if got := PersonaRoot(base, "p1"); got != filepath.Join("/app", "data", "personas", "p1") {
		t.Errorf("PersonaRoot = %s", got)
	}
}

// 測試路徑驗證：拒絕路徑穿越
func TestValidatePathTraversal(t *testing.T) {
	if err := ValidatePath("../etc/passwd", ""); err == nil {
		t.Error("should reject path traversal")
	}
	if err := ValidatePath("foo/../../bar", ""); err == nil {
		t.Error("should reject embedded traversal")
	}
}

// 測試路徑驗證：拒絕空路徑
func TestValidatePathEmpty(t *testing.T) {
	if err := ValidatePath("", ""); err == nil {
		t.Error("should reject empty path")
	}
}

// 測試路徑驗證：正常路徑應通過
func TestValidatePathNormal(t *testing.T) {
	if err := ValidatePath("my-project", ""); err != nil {
		t.Errorf("normal path should pass: %v", err)
	}
	if err := ValidatePath("project_123", ""); err != nil {
		t.Errorf("normal path should pass: %v", err)
	}
}

// 測試邊界檢查
func TestValidatePathBoundary(t *testing.T) {
	tmpDir := t.TempDir()
	innerDir := filepath.Join(tmpDir, "inner")
	os.MkdirAll(innerDir, 0755)

	// 內部路徑應通過
	if err := ValidatePath(filepath.Join(tmpDir, "inner", "file.txt"), tmpDir); err != nil {
		t.Errorf("inner path should pass: %v", err)
	}
}

// 測試 Zip-slip 保護
func TestValidateZipEntry(t *testing.T) {
	dest := "/tmp/extract"

	// 正常 entry 應通過
	path, err := ValidateZipEntry(dest, "readme.txt")
	if err != nil {
		t.Errorf("normal entry should pass: %v", err)
	}
	if path != filepath.Join(dest, "readme.txt") {
		t.Errorf("unexpected path: %s", path)
	}

	// 路徑穿越 entry 應拒絕
	_, err = ValidateZipEntry(dest, "../../../etc/passwd")
	if err == nil {
		t.Error("should reject zip-slip traversal")
	}

	// 絕對路徑 entry 應拒絕
	_, err = ValidateZipEntry(dest, "/etc/passwd")
	if err == nil {
		t.Error("should reject absolute zip entry")
	}
}

// 測試 AtomicWriteFile
func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "atomic_test.json")

	data := []byte(`{"key":"value"}`)
	if err := AtomicWriteFile(target, data, 0644); err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}

	read, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(read) != string(data) {
		t.Errorf("content mismatch: got %s", string(read))
	}
}

// 測試 AtomicAppendLine
func TestAtomicAppendLine(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.jsonl")

	lines := []string{
		`{"event":"a"}`,
		`{"event":"b"}`,
	}
	for _, line := range lines {
		if err := AtomicAppendLine(logFile, []byte(line)); err != nil {
			t.Fatalf("AtomicAppendLine failed: %v", err)
		}
	}

	data, _ := os.ReadFile(logFile)
	content := string(data)
	if content != "{\"event\":\"a\"}\n{\"event\":\"b\"}\n" {
		t.Errorf("unexpected log content: %q", content)
	}
}

// 測試 ProjectExists / PersonaExists
func TestExistsChecks(t *testing.T) {
	tmpDir := t.TempDir()

	if ProjectExists(tmpDir, "nonexistent") {
		t.Error("nonexistent project should not exist")
	}

	EnsureProjectLayout(tmpDir, "exists-test")
	if !ProjectExists(tmpDir, "exists-test") {
		t.Error("created project should exist")
	}

	if PersonaExists(tmpDir, "nonexistent") {
		t.Error("nonexistent persona should not exist")
	}

	EnsurePersonaLayout(tmpDir, "exists-test")
	if !PersonaExists(tmpDir, "exists-test") {
		t.Error("created persona should exist")
	}
}

// 測試拒絕路徑穿越的專案 ID
func TestEnsureProjectLayoutRejectsTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	err := EnsureProjectLayout(tmpDir, "../escape")
	if err == nil {
		t.Error("should reject project ID with path traversal")
	}
}
