package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClassifyOutputKind(t *testing.T) {
	cases := map[string]outputKind{
		"chart.png":            outputKindImage,
		"photo.JPG":            outputKindImage,
		"icon.svg":             outputKindImage,
		"clip.mp4":             outputKindVideo,
		"demo.MOV":             outputKindVideo,
		"電料BOM_SLM003.xlsx":    outputKindDocument,
		"notes.md":             outputKindDocument,
		"data.csv":             outputKindDocument,
		"archive.zip":          outputKindDocument,
		"noext":                outputKindDocument,
	}
	for name, want := range cases {
		got, dir := classifyOutputKind(name)
		if got != want {
			t.Errorf("classifyOutputKind(%q) kind = %q, want %q", name, got, want)
		}
		if strings.TrimSpace(dir) == "" {
			t.Errorf("classifyOutputKind(%q) returned empty dir", name)
		}
	}
}

func TestIsMeaninglessOutputName(t *testing.T) {
	meaningless := []string{
		"output.xlsx", "result.png", "out.csv", "tmp_3.csv",
		"result-2.png", "12345.png", "image.png", "untitled.docx", "report.docx",
	}
	for _, n := range meaningless {
		if !isMeaninglessOutputName(n) {
			t.Errorf("isMeaninglessOutputName(%q) = false, want true", n)
		}
	}
	meaningful := []string{
		"電料BOM_SLM003_20260327-150405.xlsx",
		"weather_report.md",
		"sales-summary-q1.xlsx",
		"員工名冊.csv",
	}
	for _, n := range meaningful {
		if isMeaninglessOutputName(n) {
			t.Errorf("isMeaninglessOutputName(%q) = true, want false", n)
		}
	}
}

func TestDeriveSkillOutputBaseName(t *testing.T) {
	when := time.Date(2026, 3, 27, 15, 4, 5, 0, time.UTC)

	// 有意義 → 沿用
	got := deriveSkillOutputBaseName("電料BOM_SLM003.xlsx", "產出電料Bom", when)
	if got != "電料BOM_SLM003.xlsx" {
		t.Errorf("meaningful name not preserved: got %q", got)
	}

	// 代號 → 補 skill 名稱 + 時間
	got = deriveSkillOutputBaseName("output.xlsx", "產出電料Bom", when)
	want := "產出電料Bom_20260327-150405.xlsx"
	if got != want {
		t.Errorf("meaningless rename: got %q, want %q", got, want)
	}

	// skill 名稱含空白/非法字元 → 清成安全片段
	got = deriveSkillOutputBaseName("result.png", "My Skill: v2", when)
	if !strings.HasSuffix(got, ".png") || strings.ContainsAny(got, `<>:"/\|?* `) {
		t.Errorf("unsafe derived name: %q", got)
	}
}

func TestSafeOutputFileComponent(t *testing.T) {
	got := safeOutputFileComponent("a/b c:d")
	if strings.ContainsAny(got, `/\: `) {
		t.Errorf("illegal chars remain: %q", got)
	}
	if safeOutputFileComponent("產出電料Bom") != "產出電料Bom" {
		t.Errorf("Chinese should be preserved: %q", safeOutputFileComponent("產出電料Bom"))
	}
}

func TestMoveIntoDirCollisionSafe(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "library")

	src := filepath.Join(srcDir, "電料BOM.xlsx")
	if err := os.WriteFile(src, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	final, err := moveIntoDirCollisionSafe(src, dstDir, "電料BOM.xlsx")
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if filepath.Base(final) != "電料BOM.xlsx" {
		t.Errorf("name = %q", filepath.Base(final))
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source should be moved away, stat err = %v", err)
	}
	if b, _ := os.ReadFile(final); string(b) != "hello" {
		t.Errorf("content mismatch: %q", string(b))
	}

	// 撞名 → 補時間戳，不覆蓋
	src2 := filepath.Join(srcDir, "dup.xlsx")
	_ = os.WriteFile(src2, []byte("world"), 0o600)
	final2, err := moveIntoDirCollisionSafe(src2, dstDir, "電料BOM.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if final2 == final {
		t.Errorf("collision not avoided: %q", final2)
	}
	if b, _ := os.ReadFile(final); string(b) != "hello" {
		t.Errorf("original overwritten: %q", string(b))
	}
}
