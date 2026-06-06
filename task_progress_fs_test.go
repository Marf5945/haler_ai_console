// task_progress_fs_test.go — Phase A FS read actions table test
//
// 涵蓋：
//   1. fsCheckPath：邊界內成功 / 邊界外 reject / 敏感 component reject / symlink escape reject
//   2. fsListDirectory：合法目錄 / 截斷 / 非目錄 reject
//   3. fsReadFile：合法檔 / 截斷 metadata / .md 升上限 / 目錄 reject
//   4. fsGlob：單層、遞迴 **、brace {md,txt} 展開、截斷
//   5. fsGrepSearch：純文字檔匹配、行截斷、跳過超大 line
//
// 用 t.TempDir() 模擬 ProjectRoot；不需要真實的 appDataRoot()。
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeFSFixture 建一個假的 ProjectRoot 模擬目錄結構：
//
//	<root>/
//	├── data/references/files/
//	│   ├── tutorial.md
//	│   └── guide.txt
//	├── notes/
//	│   └── deep/nested.md
//	└── .ssh/   (sensitive, 應該被擋)
func makeFSFixture(t *testing.T) (root string, allowed []string) {
	t.Helper()
	root = t.TempDir()
	mustMkdir(t, filepath.Join(root, "data", "references", "files"))
	mustMkdir(t, filepath.Join(root, "notes", "deep"))
	mustMkdir(t, filepath.Join(root, ".ssh"))

	mustWrite(t, filepath.Join(root, "data", "references", "files", "tutorial.md"),
		"# tutorial\n本檔說明 setup\n第二行 guide content\nthird line\n")
	mustWrite(t, filepath.Join(root, "data", "references", "files", "guide.txt"),
		"教學文件範例\ntutorial sample\n")
	mustWrite(t, filepath.Join(root, "notes", "deep", "nested.md"),
		"# nested\n深層筆記\n")
	mustWrite(t, filepath.Join(root, ".ssh", "id_rsa"), "SECRET KEY")

	allowed = []string{
		root,
		filepath.Join(root, "data", "references", "files"),
	}
	return root, allowed
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o700); err != nil {
		t.Fatal(err)
	}
}
func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ──────────────────────────────────────────────────────────────────────
// fsCheckPath
// ──────────────────────────────────────────────────────────────────────

func TestFSCheckPath_BoundaryAndSensitive(t *testing.T) {
	root, allowed := makeFSFixture(t)

	cases := []struct {
		name      string
		target    string
		wantErr   bool
		wantSub   string // error 必含此 substring（wantErr=true 才檢查）
	}{
		{"absolute within root", filepath.Join(root, "data", "references", "files"), false, ""},
		{"relative path joined to root", "data/references/files", false, ""},
		{"escape via ..", "../../etc/passwd", true, "outside allowed roots"},
		{"absolute outside root", "/etc/passwd", true, "outside allowed roots"},
		{"sensitive .ssh dir", filepath.Join(root, ".ssh"), true, "sensitive"},
		{"sensitive id_rsa file", filepath.Join(root, ".ssh", "id_rsa"), true, "sensitive"},
		{"empty path", "", true, "empty"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fsCheckPath(allowed, tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantSub != "" && !strings.Contains(err.Error(), tt.wantSub) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantSub)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestFSCheckPath_SymlinkEscape(t *testing.T) {
	root, allowed := makeFSFixture(t)

	// 建一個 root 內的 symlink 指向 root 外的 /tmp（macOS 上 /tmp 通常是 /private/tmp）
	outside := t.TempDir() // 完全獨立的 tempdir，不在 root 內
	mustWrite(t, filepath.Join(outside, "secret.md"), "outside content")

	linkPath := filepath.Join(root, "evil_link")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	// 透過 symlink 嘗試讀外部 → 應該被 reject（EvalSymlinks 後不在 allowed root）
	_, err := fsCheckPath(allowed, filepath.Join(linkPath, "secret.md"))
	if err == nil {
		t.Fatalf("expected symlink escape to be rejected")
	}
	if !strings.Contains(err.Error(), "outside allowed roots") {
		t.Errorf("expected 'outside allowed roots' error, got %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────
// fsListDirectory
// ──────────────────────────────────────────────────────────────────────

func TestFSListDirectory_OKAndRejectFile(t *testing.T) {
	_, allowed := makeFSFixture(t)

	// 合法目錄
	out, err := fsListDirectory(allowed, "data/references/files")
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	var result fsListResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}

	// 對檔案呼叫 list_directory 應 reject
	_, err = fsListDirectory(allowed, "data/references/files/tutorial.md")
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' error, got %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────
// fsReadFile
// ──────────────────────────────────────────────────────────────────────

func TestFSReadFile_NormalAndDirReject(t *testing.T) {
	_, allowed := makeFSFixture(t)

	out, err := fsReadFile(allowed, "data/references/files/tutorial.md")
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	var result fsReadResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Truncated {
		t.Errorf("small file should not be truncated")
	}
	if !strings.Contains(result.Content, "tutorial") {
		t.Errorf("expected content to contain 'tutorial'")
	}
	if result.OffsetSupported {
		t.Errorf("offset_supported must be false in v1")
	}
	if result.LimitBytes != fsMaxReadTextBytes {
		t.Errorf("expected .md to use text limit %d, got %d", fsMaxReadTextBytes, result.LimitBytes)
	}

	// 對目錄呼叫 read_file 應 reject
	_, err = fsReadFile(allowed, "data/references/files")
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("expected 'is a directory' error, got %v", err)
	}
}

func TestFSReadFile_TruncatesLargeFile(t *testing.T) {
	root, allowed := makeFSFixture(t)
	// 寫一個大檔（非 .md/.txt → 預設 64 KB 上限）
	big := strings.Repeat("A", fsMaxReadDefaultBytes+5000)
	mustWrite(t, filepath.Join(root, "data", "big.bin"), big)

	out, err := fsReadFile(allowed, "data/big.bin")
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	var result fsReadResult
	_ = json.Unmarshal([]byte(out), &result)
	if !result.Truncated {
		t.Errorf("expected truncated=true")
	}
	if result.BytesRead != fsMaxReadDefaultBytes {
		t.Errorf("expected bytes_read=%d, got %d", fsMaxReadDefaultBytes, result.BytesRead)
	}
	if result.Reason != fsTruncReasonOverflow {
		t.Errorf("expected reason=%q, got %q", fsTruncReasonOverflow, result.Reason)
	}
	if result.NextOffset == 0 {
		t.Errorf("expected next_offset to be set as hint (even if not supported)")
	}
}

// ──────────────────────────────────────────────────────────────────────
// fsGlob
// ──────────────────────────────────────────────────────────────────────

func TestFSGlob_SingleLayerAndRecursive(t *testing.T) {
	_, allowed := makeFSFixture(t)

	// 單層 *.md：只找 data/references/files/ 下的 .md（注：splitGlobRoot 會把
	// 此 pattern 當成 walk root=allowed[0]，rest="*.md"——只匹配第一層）
	out, err := fsGlob(allowed, "data/references/files/*.md")
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	var r1 fsGlobResult
	_ = json.Unmarshal([]byte(out), &r1)
	if r1.Total != 1 || !strings.HasSuffix(r1.Matches[0], "tutorial.md") {
		t.Errorf("expected 1 match (tutorial.md), got %+v", r1.Matches)
	}

	// 遞迴 **/*.md：應該找到 tutorial.md 與 nested.md
	out, err = fsGlob(allowed, "**/*.md")
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	var r2 fsGlobResult
	_ = json.Unmarshal([]byte(out), &r2)
	if r2.Total < 2 {
		t.Errorf("expected at least 2 matches (recursive **), got %d: %+v", r2.Total, r2.Matches)
	}

	// brace expansion **/*.{md,txt}
	out, err = fsGlob(allowed, "**/*.{md,txt}")
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	var r3 fsGlobResult
	_ = json.Unmarshal([]byte(out), &r3)
	if r3.Total < 3 {
		t.Errorf("expected at least 3 matches (tutorial.md + guide.txt + nested.md), got %d", r3.Total)
	}
}

func TestFSGlob_SkipsSensitive(t *testing.T) {
	_, allowed := makeFSFixture(t)
	// .ssh/id_rsa 不應出現在任何結果中
	out, err := fsGlob(allowed, "**/*")
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	var r fsGlobResult
	_ = json.Unmarshal([]byte(out), &r)
	for _, m := range r.Matches {
		if strings.Contains(m, ".ssh") || strings.Contains(strings.ToLower(m), "id_rsa") {
			t.Errorf("sensitive file leaked into glob result: %s", m)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────
// fsGrepSearch
// ──────────────────────────────────────────────────────────────────────

func TestFSGrepSearch_FindsKeyword(t *testing.T) {
	_, allowed := makeFSFixture(t)
	out, err := fsGrepSearch(allowed, "教學|tutorial")
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	var r fsGrepResult
	_ = json.Unmarshal([]byte(out), &r)
	if r.Total == 0 {
		t.Errorf("expected hits for 教學|tutorial")
	}
	for _, hit := range r.Hits {
		if !strings.Contains(strings.ToLower(hit.File), "references/files") {
			t.Errorf("hit not in references/files cache: %s", hit.File)
		}
	}
}

func TestFSGrepSearch_BadPattern(t *testing.T) {
	_, allowed := makeFSFixture(t)
	_, err := fsGrepSearch(allowed, "(unclosed")
	if err == nil || !strings.Contains(err.Error(), "bad regexp") {
		t.Errorf("expected 'bad regexp' error, got %v", err)
	}
}
