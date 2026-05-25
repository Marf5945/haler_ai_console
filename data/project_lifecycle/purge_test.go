package project_lifecycle

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 測試自動清除安全類目錄
func TestPurgeAutoSafe(t *testing.T) {
	tmpDir := t.TempDir()
	// 建立安全類目錄
	for _, dir := range autoSafeDirs {
		os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
		os.WriteFile(filepath.Join(tmpDir, dir, "test.tmp"), []byte("data"), 0644)
	}

	svc := NewService(tmpDir)
	result, err := svc.Purge("test-project", TriggerUserArchives)
	if err != nil {
		t.Fatalf("purge failed: %v", err)
	}
	if len(result.AutoCleaned) != len(autoSafeDirs) {
		t.Errorf("expected %d auto cleaned, got %d", len(autoSafeDirs), len(result.AutoCleaned))
	}
}

func TestBackupAutoSafeDirsOnlyCopiesPurgeTargets(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "runtime", "temp_sessions", "session.tmp")
	actionFile := filepath.Join(tmpDir, "runtime", "action_results", "result.json")
	memoryFile := filepath.Join(tmpDir, "memory", "main_memory.md")
	os.MkdirAll(filepath.Dir(sessionFile), 0755)
	os.MkdirAll(filepath.Dir(actionFile), 0755)
	os.MkdirAll(filepath.Dir(memoryFile), 0755)
	os.WriteFile(sessionFile, []byte("session"), 0644)
	os.WriteFile(actionFile, []byte("result"), 0644)
	os.WriteFile(memoryFile, []byte("important"), 0644)

	svc := NewService(tmpDir)
	manifest, err := svc.BackupAutoSafeDirs("test-project")
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}
	if len(manifest.Entries) != 2 {
		t.Fatalf("expected 2 backed up dirs, got %d", len(manifest.Entries))
	}
	if _, err := os.Stat(filepath.Join(manifest.Root, "runtime", "temp_sessions", "session.tmp")); err != nil {
		t.Error("temp_sessions file should be backed up")
	}
	if _, err := os.Stat(filepath.Join(manifest.Root, "runtime", "action_results", "result.json")); err != nil {
		t.Error("action_results file should be backed up")
	}
	if _, err := os.Stat(filepath.Join(manifest.Root, "memory", "main_memory.md")); !os.IsNotExist(err) {
		t.Error("forbidden memory file must not be backed up by purge backup")
	}
}

// 測試禁止類不被清除
func TestPurgeForbidden(t *testing.T) {
	tmpDir := t.TempDir()
	// 建立禁止類檔案
	forbiddenPath := filepath.Join(tmpDir, "memory", "main_memory.md")
	os.MkdirAll(filepath.Dir(forbiddenPath), 0755)
	os.WriteFile(forbiddenPath, []byte("important"), 0644)

	svc := NewService(tmpDir)
	result, _ := svc.Purge("test", TriggerUserDeletes)

	// 檢查檔案仍然存在
	if _, err := os.Stat(forbiddenPath); os.IsNotExist(err) {
		t.Error("forbidden file should not be deleted")
	}
	if len(result.Skipped) == 0 {
		t.Error("should report skipped items")
	}
}

// 測試邊界類標記為 need_review
func TestPurgeBoundary(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "controlled_trust", "draft_sandbox_runs"), 0755)

	svc := NewService(tmpDir)
	result, _ := svc.Purge("test", TriggerUserArchives)

	if len(result.NeedReview) == 0 {
		t.Error("boundary dirs should be marked for review")
	}
}

// 測試 PurgeBoundaryDir 防護
func TestPurgeBoundaryDirProtection(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// 不允許清除非邊界路徑
	err := svc.PurgeBoundaryDir("memory/main_memory.md")
	if err == nil {
		t.Error("should reject non-boundary path")
	}
}

// 測試過期掃描清理
func TestScanAndCleanExpired(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// 建立過期檔案（修改時間為 60 天前）
	dir := filepath.Join(tmpDir, "runtime", "temp_sessions")
	os.MkdirAll(dir, 0755)
	oldFile := filepath.Join(dir, "old_session.tmp")
	os.WriteFile(oldFile, []byte("old data"), 0644)
	// 設定修改時間為 60 天前
	oldTime := time.Now().AddDate(0, 0, -60)
	os.Chtimes(oldFile, oldTime, oldTime)

	// 建立新檔案
	newFile := filepath.Join(dir, "new_session.tmp")
	os.WriteFile(newFile, []byte("new data"), 0644)

	result, err := svc.ScanAndCleanExpired(30)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// 過期檔案應被清除
	if len(result.Cleaned) == 0 {
		t.Error("expired files should be cleaned")
	}
	// 新檔案應保留
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("new file should be preserved")
	}
}

// 測試 Manifest 生成與儲存
func TestPurgeManifest(t *testing.T) {
	tmpDir := t.TempDir()
	entries := []PurgeEntry{
		{Path: "runtime/temp", Category: "auto_safe", Size: 1024, Action: "removed"},
		{Path: "memory/main.md", Category: "forbidden", Action: "preserved"},
	}

	manifest := NewPurgeManifest("test", TriggerUserArchives, entries)
	if manifest.Summary.TotalRemoved != 1 {
		t.Errorf("expected 1 removed, got %d", manifest.Summary.TotalRemoved)
	}
	if manifest.Summary.TotalPreserved != 1 {
		t.Errorf("expected 1 preserved, got %d", manifest.Summary.TotalPreserved)
	}

	err := manifest.Save(tmpDir)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// 載入驗證
	manifests, err := ListManifests(tmpDir)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(manifests) != 1 {
		t.Errorf("expected 1 manifest, got %d", len(manifests))
	}
}

// 測試路徑分類
func TestClassifyPath(t *testing.T) {
	cases := []struct {
		path string
		want PurgeCategory
	}{
		{"runtime/temp_sessions/abc", CategoryAutoSafe},
		{"runtime/crash_recovery/x", CategoryAutoSafe},
		{"memory/main_memory.md", CategoryForbidden},
		{"controlled_trust/draft_sandbox_runs/r1", CategoryBoundary},
		{"unknown/path", CategoryForbidden}, // 未知預設禁止
	}
	for _, c := range cases {
		got := ClassifyPath(c.path)
		if got != c.want {
			t.Errorf("ClassifyPath(%s) = %s, want %s", c.path, got, c.want)
		}
	}
}
