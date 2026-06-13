package skill_step

import (
	"reflect"
	"testing"
)

func TestParseSkillFrontmatter_Basic(t *testing.T) {
	src := []byte("---\nname: pdf-processor\ndescription: 從 PDF 抽取文字與表格\nallowed-tools: Read, Bash\n---\n\n# PDF Processor\n本文……\n")
	fm, ok := ParseSkillFrontmatter(src)
	if !ok {
		t.Fatalf("expected frontmatter to be found")
	}
	if fm.Name != "pdf-processor" {
		t.Errorf("Name = %q, want pdf-processor", fm.Name)
	}
	if fm.Description != "從 PDF 抽取文字與表格" {
		t.Errorf("Description = %q", fm.Description)
	}
	if !reflect.DeepEqual(fm.AllowedTools, []string{"Read", "Bash"}) {
		t.Errorf("AllowedTools = %#v", fm.AllowedTools)
	}
}

func TestParseSkillFrontmatter_InlineArray(t *testing.T) {
	src := []byte("---\nname: x\nallowed-tools: [Read, \"Bash\", Edit]\n---\nbody\n")
	fm, ok := ParseSkillFrontmatter(src)
	if !ok {
		t.Fatal("expected ok")
	}
	if !reflect.DeepEqual(fm.AllowedTools, []string{"Read", "Bash", "Edit"}) {
		t.Errorf("AllowedTools = %#v", fm.AllowedTools)
	}
}

func TestParseSkillFrontmatter_BlockList(t *testing.T) {
	src := []byte("---\nname: x\nallowed-tools:\n  - Read\n  - Bash\n  - Edit\ndescription: hi\n---\nbody\n")
	fm, ok := ParseSkillFrontmatter(src)
	if !ok {
		t.Fatal("expected ok")
	}
	if !reflect.DeepEqual(fm.AllowedTools, []string{"Read", "Bash", "Edit"}) {
		t.Errorf("AllowedTools = %#v", fm.AllowedTools)
	}
	if fm.Description != "hi" {
		t.Errorf("Description = %q (block list should not swallow following key)", fm.Description)
	}
}

func TestParseSkillFrontmatter_QuotedAndCRLF(t *testing.T) {
	src := []byte("\ufeff---\r\nname: \"My Skill\"\r\ndescription: 'single quoted'\r\n---\r\nbody\r\n")
	fm, ok := ParseSkillFrontmatter(src)
	if !ok {
		t.Fatal("expected ok (BOM + CRLF)")
	}
	if fm.Name != "My Skill" {
		t.Errorf("Name = %q", fm.Name)
	}
	if fm.Description != "single quoted" {
		t.Errorf("Description = %q", fm.Description)
	}
}

func TestParseSkillFrontmatter_LeadingBlankLines(t *testing.T) {
	src := []byte("\n\n---\nname: y\n---\nbody")
	fm, ok := ParseSkillFrontmatter(src)
	if !ok {
		t.Fatal("expected ok with leading blank lines")
	}
	if fm.Name != "y" {
		t.Errorf("Name = %q", fm.Name)
	}
}

func TestParseSkillFrontmatter_NoFrontmatter(t *testing.T) {
	cases := map[string][]byte{
		"plain markdown":   []byte("# Title\njust a doc\n"),
		"opening only":     []byte("---\nname: x\nno closing fence\n"),
		"empty":            []byte(""),
		"delimiter midway": []byte("intro\n---\nname: x\n---\n"),
	}
	for label, src := range cases {
		if _, ok := ParseSkillFrontmatter(src); ok {
			t.Errorf("%s: expected ok=false", label)
		}
	}
}

func TestParseSkillFrontmatter_CommentsAndUnknownKeys(t *testing.T) {
	src := []byte("---\n# a comment\nname: z\nlicense: MIT\nversion: 1.2.3\n---\nbody")
	fm, ok := ParseSkillFrontmatter(src)
	if !ok {
		t.Fatal("expected ok")
	}
	if fm.Name != "z" {
		t.Errorf("Name = %q", fm.Name)
	}
	if fm.Fields["license"] != "MIT" || fm.Fields["version"] != "1.2.3" {
		t.Errorf("unknown keys not captured: %#v", fm.Fields)
	}
}

func TestAppendUniqueFM(t *testing.T) {
	out := appendUniqueFM([]string{"a", "b"}, "a")
	if !reflect.DeepEqual(out, []string{"a", "b"}) {
		t.Errorf("dup should be skipped: %#v", out)
	}
	out = appendUniqueFM([]string{"a"}, "c")
	if !reflect.DeepEqual(out, []string{"a", "c"}) {
		t.Errorf("new value should append: %#v", out)
	}
}
