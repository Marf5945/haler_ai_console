package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testPassword = "correct-horse-9"

// buildTestProject 建立含對話、設定與 runtime 暫存的假專案。
func buildTestProject(t *testing.T, baseDir, projectID string) string {
	t.Helper()
	root := filepath.Join(baseDir, "data", "projects", projectID)
	mustWrite := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("memory/talk_full.md", "user: 幫我查訂單\nassistant: 好的\napi key 是 "+testBackupOpenAIKey()+"\n")
	mustWrite("memory/memory_manifest.json", `{"version":1}`)
	mustWrite("dag_runs/run1.json", `{"status":"done"}`)
	mustWrite("runtime/temp_sessions/tmp.json", `{"junk":true}`)
	return root
}

func exportToTemp(t *testing.T, baseDir string, redact bool) string {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "p.aicbak")
	res, err := ExportProject(baseDir, "default", dest, testPassword, redact)
	if err != nil {
		t.Fatalf("ExportProject: %v", err)
	}
	if res.BundlePath != dest {
		t.Fatalf("bundle path = %q", res.BundlePath)
	}
	return dest
}

// ── 基本 roundtrip ───────────────────────────────

func TestExportImportRoundtrip(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	bundle := exportToTemp(t, src, false)

	dst := t.TempDir()
	res, err := ImportProject(dst, bundle, testPassword, ModeFailIfExists)
	if err != nil {
		t.Fatalf("ImportProject: %v", err)
	}
	if res.RestoredAs != "default" {
		t.Fatalf("restored as %q", res.RestoredAs)
	}
	talk, err := os.ReadFile(filepath.Join(dst, "data", "projects", "default", "memory", "talk_full.md"))
	if err != nil {
		t.Fatalf("talk_full 未還原: %v", err)
	}
	if !strings.Contains(string(talk), "幫我查訂單") {
		t.Fatal("對話內容遺失")
	}
	// runtime/ 暫存不應進備份
	if _, err := os.Stat(filepath.Join(dst, "data", "projects", "default", "runtime", "temp_sessions", "tmp.json")); err == nil {
		t.Fatal("runtime 暫存不應被還原")
	}
}

// ── 密碼與竄改 ───────────────────────────────────

func TestWrongPasswordRejected(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	bundle := exportToTemp(t, src, false)

	if _, err := ImportProject(t.TempDir(), bundle, "wrong-password-1", ModeFailIfExists); err != ErrDecryptFailed {
		t.Fatalf("want ErrDecryptFailed, got %v", err)
	}
}

func TestShortPasswordRejected(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	dest := filepath.Join(t.TempDir(), "p.aicbak")
	if _, err := ExportProject(src, "default", dest, "short", false); err != ErrPasswordTooShort {
		t.Fatalf("want ErrPasswordTooShort, got %v", err)
	}
}

func TestTamperedBundleRejected(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	bundle := exportToTemp(t, src, false)

	raw, err := os.ReadFile(bundle)
	if err != nil {
		t.Fatal(err)
	}
	// 翻轉密文中段一個 bit
	raw[len(raw)/2] ^= 0x01
	tampered := filepath.Join(t.TempDir(), "tampered.aicbak")
	if err := os.WriteFile(tampered, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportProject(t.TempDir(), tampered, testPassword, ModeFailIfExists); err != ErrDecryptFailed {
		t.Fatalf("want ErrDecryptFailed, got %v", err)
	}
}

func TestHeaderSwapRejected(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	bundle := exportToTemp(t, src, false)
	raw, _ := os.ReadFile(bundle)
	// 改 salt（AAD 綁定檔頭，應拒開）
	raw[len(magicHeader)] ^= 0xFF
	swapped := filepath.Join(t.TempDir(), "swapped.aicbak")
	_ = os.WriteFile(swapped, raw, 0o600)
	if _, err := ImportProject(t.TempDir(), swapped, testPassword, ModeFailIfExists); err != ErrDecryptFailed {
		t.Fatalf("want ErrDecryptFailed, got %v", err)
	}
}

func TestGarbageFileRejected(t *testing.T) {
	garbage := filepath.Join(t.TempDir(), "x.aicbak")
	_ = os.WriteFile(garbage, []byte("not a backup at all"), 0o600)
	if _, err := ImportProject(t.TempDir(), garbage, testPassword, ModeFailIfExists); err != ErrBadFormat {
		t.Fatalf("want ErrBadFormat, got %v", err)
	}
}

// ── 路徑穿越（SEC）─────────────────────────────

// TestMaliciousEntryRejected 手工構造含 ../ entry 的備份，驗證匯入被拒。
func TestMaliciousEntryRejected(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	manifest := []byte(`{"format_version":1,"project_id":"evil","created_at":"2026-01-01T00:00:00Z","redacted":false,"file_count":1}`)
	_ = tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: "manifest.json", Mode: 0o600, Size: int64(len(manifest)), ModTime: time.Now()})
	_, _ = tw.Write(manifest)

	payload := []byte("pwned")
	_ = tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: "project/../../escape.txt", Mode: 0o600, Size: int64(len(payload)), ModTime: time.Now()})
	_, _ = tw.Write(payload)
	_ = tw.Close()
	_ = gz.Close()

	sealed, err := seal(buf.Bytes(), testPassword)
	if err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(t.TempDir(), "evil.aicbak")
	_ = os.WriteFile(bundle, sealed, 0o600)

	dst := t.TempDir()
	if _, err := ImportProject(dst, bundle, testPassword, ModeFailIfExists); err == nil {
		t.Fatal("路徑穿越 entry 應被拒絕")
	}
	if _, err := os.Stat(filepath.Join(dst, "escape.txt")); err == nil {
		t.Fatal("穿越檔案被寫出，zip-slip 防護失效")
	}
}

// ── 衝突模式 ─────────────────────────────────────

func TestImportConflictModes(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	bundle := exportToTemp(t, src, false)

	dst := t.TempDir()
	if _, err := ImportProject(dst, bundle, testPassword, ModeFailIfExists); err != nil {
		t.Fatal(err)
	}
	// 第二次：預設模式應回報衝突
	if _, err := ImportProject(dst, bundle, testPassword, ModeFailIfExists); err != ErrProjectExists {
		t.Fatalf("want ErrProjectExists, got %v", err)
	}
	// copy 模式：另存新 ID
	res, err := ImportProject(dst, bundle, testPassword, ModeCopy)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.RestoredAs, "default-restored-") {
		t.Fatalf("copy 模式還原 ID = %q", res.RestoredAs)
	}
	// overwrite 模式：覆蓋成功且內容存在
	if _, err := ImportProject(dst, bundle, testPassword, ModeOverwrite); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "data", "projects", "default", "memory", "talk_full.md")); err != nil {
		t.Fatal("overwrite 後內容遺失")
	}
}

// ── 遮蔽選項 ─────────────────────────────────────

func TestRedactOptionMasksSecrets(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	bundle := exportToTemp(t, src, true)

	dst := t.TempDir()
	res, err := ImportProject(dst, bundle, testPassword, ModeFailIfExists)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Redacted {
		t.Fatal("manifest 應標記 redacted")
	}
	talk, _ := os.ReadFile(filepath.Join(dst, "data", "projects", "default", "memory", "talk_full.md"))
	if strings.Contains(string(talk), testBackupOpenAIKey()) {
		t.Fatal("API key 未被遮蔽")
	}
	if !strings.Contains(string(talk), "[REDACTED:") {
		t.Fatal("應出現遮蔽標記")
	}
	if !strings.Contains(string(talk), "幫我查訂單") {
		t.Fatal("一般對話內容不應被改動")
	}
	// 結構化檔案不應被遮蔽流程破壞
	manifestJSON, _ := os.ReadFile(filepath.Join(dst, "data", "projects", "default", "memory", "memory_manifest.json"))
	if string(manifestJSON) != `{"version":1}` {
		t.Fatalf("結構化檔案被改動: %s", manifestJSON)
	}
}

func testBackupOpenAIKey() string {
	return "sk-" + "abcdefghijklmnopqrstuvwxyz123456"
}

// ── 預覽 ─────────────────────────────────────────

func TestInspectBundle(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	bundle := exportToTemp(t, src, false)

	m, err := InspectBundle(bundle, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if m.ProjectID != "default" || m.FormatVersion != FormatVersion {
		t.Fatalf("manifest = %+v", m)
	}
}

// ── 明文模式（「加密備份檔」勾選框未勾）──────────

func TestPlainExportImportRoundtrip(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	dest := filepath.Join(t.TempDir(), "plain.aicbak")
	res, err := ExportProject(src, "default", dest, "", false) // 空密碼 = 明文
	if err != nil {
		t.Fatalf("明文匯出失敗: %v", err)
	}
	if res.Encrypted {
		t.Fatal("空密碼應產出明文備份")
	}
	// 匯入不需要密碼
	dst := t.TempDir()
	ires, err := ImportProject(dst, dest, "", ModeFailIfExists)
	if err != nil {
		t.Fatalf("明文匯入失敗: %v", err)
	}
	if ires.RestoredAs != "default" {
		t.Fatalf("restored as %q", ires.RestoredAs)
	}
	talk, err := os.ReadFile(filepath.Join(dst, "data", "projects", "default", "memory", "talk_full.md"))
	if err != nil || !strings.Contains(string(talk), "幫我查訂單") {
		t.Fatal("明文備份內容未正確還原")
	}
}

func TestIsBundleEncrypted(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")

	plain := filepath.Join(t.TempDir(), "p.aicbak")
	if _, err := ExportProject(src, "default", plain, "", false); err != nil {
		t.Fatal(err)
	}
	enc := filepath.Join(t.TempDir(), "e.aicbak")
	if _, err := ExportProject(src, "default", enc, testPassword, false); err != nil {
		t.Fatal(err)
	}

	if got, err := IsBundleEncrypted(plain); err != nil || got {
		t.Fatalf("明文檔判斷錯誤: got=%v err=%v", got, err)
	}
	if got, err := IsBundleEncrypted(enc); err != nil || !got {
		t.Fatalf("加密檔判斷錯誤: got=%v err=%v", got, err)
	}
	garbage := filepath.Join(t.TempDir(), "g.aicbak")
	_ = os.WriteFile(garbage, []byte("garbage data here"), 0o600)
	if _, err := IsBundleEncrypted(garbage); err == nil {
		t.Fatal("垃圾檔應回報格式錯誤")
	}
}

func TestEncryptedStillRequiresPassword(t *testing.T) {
	// 明文模式存在後，加密檔仍必須拿密碼才能開（防止繞過）。
	src := t.TempDir()
	buildTestProject(t, src, "default")
	enc := filepath.Join(t.TempDir(), "e.aicbak")
	if _, err := ExportProject(src, "default", enc, testPassword, false); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportProject(t.TempDir(), enc, "", ModeFailIfExists); err == nil {
		t.Fatal("加密檔不給密碼竟然開成功")
	}
}

// ── 輸入驗證 ─────────────────────────────────────

func TestExportRejectsBadProjectID(t *testing.T) {
	src := t.TempDir()
	buildTestProject(t, src, "default")
	dest := filepath.Join(t.TempDir(), "p.aicbak")
	for _, id := range []string{"../etc", "a/b", "a\\b", "a b", ""} {
		if _, err := ExportProject(src, id, dest, testPassword, false); err == nil {
			t.Fatalf("專案 ID %q 應被拒絕", id)
		}
	}
}
