// subexport/subexport_test.go — subexport 套件測試。
package subexport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ──────────────────────────────────────────────
// codegen 測試
// ──────────────────────────────────────────────

func TestGenerateRandomCode_Length(t *testing.T) {
	code, err := GenerateRandomCode()
	if err != nil {
		t.Fatalf("GenerateRandomCode 錯誤: %v", err)
	}
	if len(code) != 12 {
		t.Errorf("碼長度應為 12，得到 %d", len(code))
	}
}

func TestGenerateRandomCode_Charset(t *testing.T) {
	code, err := GenerateRandomCode()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range code {
		if !strings.ContainsRune(charset, c) {
			t.Errorf("包含非法字元: %c", c)
		}
		if c == 'I' || c == 'O' {
			t.Errorf("包含排除字元: %c", c)
		}
	}
}

func TestGenerateRandomCode_Unique(t *testing.T) {
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := GenerateRandomCode()
		if err != nil {
			t.Fatal(err)
		}
		if codes[code] {
			t.Errorf("產生重複碼: %s", code)
		}
		codes[code] = true
	}
}

func TestGenerateSystemCode_Format(t *testing.T) {
	code, err := GenerateSystemCode("handler1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(code, "handler1_SUB_") {
		t.Errorf("應以 handler1_SUB_ 開頭，得到: %s", code)
	}
	if !strings.Contains(code, "_SUB_") {
		t.Errorf("應包含 _SUB_，得到: %s", code)
	}
	// 驗證格式: handler1_SUB_12碼
	parts := strings.Split(code, "_SUB_")
	if len(parts) != 2 {
		t.Errorf("應有 _SUB_ 分隔為兩段，得到 %d 段", len(parts))
	}
	if len(parts[1]) != 12 {
		t.Errorf("隨機碼應為 12 碼，得到 %d", len(parts[1]))
	}
}

func TestGenerateSystemCode_SanitizesPathSeparators(t *testing.T) {
	code, err := GenerateSystemCode("新haㄌer 19/45")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(code, "/") || strings.Contains(code, "\\") {
		t.Errorf("系統碼不應包含路徑分隔符: %s", code)
	}
	if !strings.HasPrefix(code, "新haㄌer 19／45_SUB_") {
		t.Errorf("應保留可讀名稱並替換 slash，得到: %s", code)
	}
}

// ──────────────────────────────────────────────
// manifest 測試
// ──────────────────────────────────────────────

func TestManifest_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	tools := []ManifestTool{
		{Type: "skill", Path: "tools/skills/weather/", OriginalID: "weather"},
	}
	m := NewInstallManifest("test_handler", "test_20260516_1430_SUB_ABCDEFGHJKLM", tools)

	if err := SaveManifest(dir, m); err != nil {
		t.Fatalf("SaveManifest 失敗: %v", err)
	}

	loaded, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest 失敗: %v", err)
	}

	if loaded.DisplayName != "test_handler" {
		t.Errorf("DisplayName 不符: %s", loaded.DisplayName)
	}
	if loaded.FormatVersion != "1.0" {
		t.Errorf("FormatVersion 不符: %s", loaded.FormatVersion)
	}
	if len(loaded.Files.Tools) != 1 {
		t.Errorf("Tools 數量不符: %d", len(loaded.Files.Tools))
	}
	if loaded.InstallInstructions.ToolConflictPolicy != "ask_user" {
		t.Errorf("ToolConflictPolicy 不符: %s", loaded.InstallInstructions.ToolConflictPolicy)
	}
}

func TestGenerateReadme_ContainsInfo(t *testing.T) {
	tools := []ManifestTool{
		{Type: "mcp", Path: "tools/mcps/server1/", OriginalID: "server1"},
	}
	m := NewInstallManifest("myhandler", "myhandler_20260516_1430_SUB_123456789ABC", tools)

	readme := GenerateReadme(m)

	checks := []string{"myhandler", "myhandler_20260516_1430_SUB_123456789ABC", "tools/mcps/server1/", "手動安裝步驟"}
	for _, check := range checks {
		if !strings.Contains(readme, check) {
			t.Errorf("README 應包含 %q", check)
		}
	}
}

// ──────────────────────────────────────────────
// export 測試
// ──────────────────────────────────────────────

func TestPackExport_CreatesStructure(t *testing.T) {
	// 準備假的 project 結構
	projectRoot := t.TempDir()
	subID := "test_sub_code"
	subBase := filepath.Join(projectRoot, "subagents", "callable", subID)

	// 建立 sub 目錄與假資料
	for _, d := range []string{"memory", "dag", "tool_history"} {
		os.MkdirAll(filepath.Join(subBase, d), 0755)
	}
	os.WriteFile(filepath.Join(subBase, "memory", "talk_full.md"), []byte("# Talk"), 0644)
	os.WriteFile(filepath.Join(subBase, "dag", "state.json"), []byte("{}"), 0644)

	destDir := t.TempDir()

	result, err := PackExport(ExportOptions{
		ProjectRoot:    projectRoot,
		SubID:          subID,
		DisplayName:    "handler1",
		Mode:           ExportCopy,
		DestDir:        destDir,
		ConnectedTools: nil,
	})
	if err != nil {
		t.Fatalf("PackExport 失敗: %v", err)
	}

	// 驗證匯出資料夾存在
	if _, err := os.Stat(result.ExportDir); os.IsNotExist(err) {
		t.Errorf("匯出資料夾不存在: %s", result.ExportDir)
	}

	// 驗證 manifest 存在
	if _, err := os.Stat(filepath.Join(result.ExportDir, "install_manifest.json")); os.IsNotExist(err) {
		t.Error("install_manifest.json 不存在")
	}

	// 驗證 README 存在
	if _, err := os.Stat(filepath.Join(result.ExportDir, "README_INSTALL.md")); os.IsNotExist(err) {
		t.Error("README_INSTALL.md 不存在")
	}

	// 驗證 memory 內容已複製
	data, err := os.ReadFile(filepath.Join(result.ExportDir, "memory", "talk_full.md"))
	if err != nil || string(data) != "# Talk" {
		t.Error("memory/talk_full.md 複製失敗")
	}

	// 複製模式：資料夾名仍維持 sub名稱_SUB_代號，不額外加 _copy。
	baseName := filepath.Base(result.ExportDir)
	if !strings.HasPrefix(baseName, "handler1_SUB_") {
		t.Errorf("匯出資料夾應使用 sub名稱_SUB_代號: %s", result.ExportDir)
	}
	if strings.HasSuffix(result.ExportDir, "_copy") {
		t.Errorf("匯出資料夾不應以 _copy 結尾: %s", result.ExportDir)
	}
}

func TestRemoveSubFromSystem(t *testing.T) {
	projectRoot := t.TempDir()
	subID := "to_remove"
	subDir := filepath.Join(projectRoot, "subagents", "callable", subID, "memory")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "talk_full.md"), []byte("x"), 0644)

	if err := RemoveSubFromSystem(projectRoot, subID); err != nil {
		t.Fatalf("RemoveSubFromSystem 失敗: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, "subagents", "callable", subID)); !os.IsNotExist(err) {
		t.Error("sub 目錄應已被刪除")
	}
}

// ──────────────────────────────────────────────
// import 測試
// ──────────────────────────────────────────────

func TestImportSub_BasicFlow(t *testing.T) {
	// 準備匯出資料夾
	exportDir := t.TempDir()
	os.MkdirAll(filepath.Join(exportDir, "memory"), 0755)
	os.MkdirAll(filepath.Join(exportDir, "dag"), 0755)
	os.MkdirAll(filepath.Join(exportDir, "tool_history"), 0755)
	os.WriteFile(filepath.Join(exportDir, "memory", "talk_full.md"), []byte("# History"), 0644)

	// 建立 manifest
	manifest := NewInstallManifest("imported_handler", "old_code_SUB_XXXX", nil)
	SaveManifest(exportDir, manifest)

	// 匯入到新 project
	projectRoot := t.TempDir()
	os.MkdirAll(filepath.Join(projectRoot, "main"), 0755) // delegation_log 需要

	result, err := ImportSub(exportDir, projectRoot)
	if err != nil {
		t.Fatalf("ImportSub 失敗: %v", err)
	}

	// 新碼應不同於舊碼
	if result.NewSystemCode == "old_code_SUB_XXXX" {
		t.Error("匯入應產生新碼")
	}

	// 驗證 memory 已複製
	talkPath := filepath.Join(result.SubDir, "memory", "talk_full.md")
	data, err := os.ReadFile(talkPath)
	if err != nil || string(data) != "# History" {
		t.Error("memory 複製失敗")
	}
}

func TestImportSub_ToolConflict(t *testing.T) {
	// 準備匯出資料夾（含工具）
	exportDir := t.TempDir()
	os.MkdirAll(filepath.Join(exportDir, "memory"), 0755)
	os.MkdirAll(filepath.Join(exportDir, "dag"), 0755)
	os.MkdirAll(filepath.Join(exportDir, "tool_history"), 0755)
	toolDir := filepath.Join(exportDir, "tools", "skills", "weather")
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(toolDir, "config.json"), []byte(`{"version":"new"}`), 0644)

	tools := []ManifestTool{
		{Type: "skill", Path: "tools/skills/weather/", OriginalID: "weather"},
	}
	manifest := NewInstallManifest("handler_with_tools", "x_SUB_Y", tools)
	SaveManifest(exportDir, manifest)

	// project 中已存在同名工具
	projectRoot := t.TempDir()
	os.MkdirAll(filepath.Join(projectRoot, "main"), 0755)
	existingTool := filepath.Join(projectRoot, "tools", "skills", "weather")
	os.MkdirAll(existingTool, 0755)
	os.WriteFile(filepath.Join(existingTool, "config.json"), []byte(`{"version":"old"}`), 0644)

	result, err := ImportSub(exportDir, projectRoot)
	if err != nil {
		t.Fatalf("ImportSub 失敗: %v", err)
	}

	// 應回報衝突
	if len(result.ToolConflicts) != 1 {
		t.Fatalf("應有 1 個衝突，得到 %d", len(result.ToolConflicts))
	}
	if result.ToolConflicts[0].OriginalID != "weather" {
		t.Errorf("衝突工具 ID 不符: %s", result.ToolConflicts[0].OriginalID)
	}

	// 現有工具不應被覆蓋
	data, _ := os.ReadFile(filepath.Join(existingTool, "config.json"))
	var obj map[string]string
	json.Unmarshal(data, &obj)
	if obj["version"] != "old" {
		t.Error("衝突時不應自動覆蓋")
	}
}

func TestResolveToolConflict(t *testing.T) {
	// 準備匯出版本
	exportToolDir := t.TempDir()
	os.WriteFile(filepath.Join(exportToolDir, "config.json"), []byte(`{"v":"new"}`), 0644)

	// 準備系統版本
	systemToolDir := t.TempDir()
	os.WriteFile(filepath.Join(systemToolDir, "config.json"), []byte(`{"v":"old"}`), 0644)

	conflict := ToolConflict{
		OriginalID: "test_tool",
		Type:       "skill",
		ExportPath: exportToolDir,
		SystemPath: systemToolDir,
	}

	if err := ResolveToolConflict(conflict); err != nil {
		t.Fatalf("ResolveToolConflict 失敗: %v", err)
	}

	// 驗證已覆蓋
	data, _ := os.ReadFile(filepath.Join(systemToolDir, "config.json"))
	var obj map[string]string
	json.Unmarshal(data, &obj)
	if obj["v"] != "new" {
		t.Error("覆蓋後應為新版本")
	}
}

// ──────────────────────────────────────────────
// delegation_log 測試
// ──────────────────────────────────────────────

func TestWriteDelegationLog(t *testing.T) {
	projectRoot := t.TempDir()
	os.MkdirAll(filepath.Join(projectRoot, "main"), 0755)

	if err := WriteDelegationLog(projectRoot, "export_remove", "sub1", "new_code"); err != nil {
		t.Fatalf("WriteDelegationLog 失敗: %v", err)
	}

	logPath := filepath.Join(projectRoot, "main", "delegation_log.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("讀取 log 失敗: %v", err)
	}
	if !strings.Contains(string(data), "export_remove") {
		t.Error("log 應包含 export_remove")
	}
}
