package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

func TestRunReportsUTF8BOMAndRewritesIt(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "README.md")
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello\n")...)

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := run([]string{"-root", root}, &out, &errOut); code != 1 {
		t.Fatalf("expected exit code 1, got %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "README.md (utf-8-bom)") {
		t.Fatalf("expected BOM report, got %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run([]string{"-root", root, "-write"}, &out, &errOut); code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, errOut.String())
	}

	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten file: %v", err)
	}
	if hasUTF8BOM(rewritten) {
		t.Fatal("expected BOM to be removed")
	}
	if !utf8.Valid(rewritten) {
		t.Fatal("expected rewritten content to be valid UTF-8")
	}
}

func TestRunConvertsBig5ToUTF8(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.txt")
	want := "中文註解\n"

	raw, err := encodeText("big5", want)
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := run([]string{"-root", root, "-write"}, &out, &errOut); code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "note.txt (big5)") {
		t.Fatalf("expected Big5 report, got %q", out.String())
	}

	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten file: %v", err)
	}
	if string(rewritten) != want {
		t.Fatalf("unexpected rewritten content: got %q want %q", string(rewritten), want)
	}
	if !utf8.Valid(rewritten) {
		t.Fatal("expected rewritten content to be valid UTF-8")
	}

	out.Reset()
	errOut.Reset()
	if code := run([]string{"-root", root}, &out, &errOut); code != 0 {
		t.Fatalf("expected clean tree after rewrite, got %d, stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "already UTF-8 without BOM") {
		t.Fatalf("expected clean summary, got %q", out.String())
	}
}

func encodeText(encodingName, text string) ([]byte, error) {
	enc, err := htmlindex.Get(encodingName)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer := transform.NewWriter(&buf, enc.NewEncoder())
	if _, err := writer.Write([]byte(text)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
