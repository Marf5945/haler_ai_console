package skill_step

import (
	"os"
	"slices"
	"testing"
)

// TestRegisterBuiltinOverridesExternal 驗證 builtin 優先於同 ID 外部 skill。
func TestRegisterBuiltinOverridesExternal(t *testing.T) {
	dir, _ := os.MkdirTemp("", "router-test-*")
	defer os.RemoveAll(dir)

	archive := NewArchiveService(dir)
	r := NewRouter(archive)

	// 註冊 builtin
	r.RegisterBuiltin(&SkillManifest{
		SkillID:     "builtin.document.read",
		DisplayName: "文件讀取（內建）",
		Version:     "1.0.0",
		Tags: SkillTags{
			ActionTag: []string{"讀取"},
			DomainTag: []string{"文件"},
			RiskTag:   []string{"low"},
		},
	})

	m := r.builtinManifests["builtin.document.read"]
	if m == nil {
		t.Fatal("builtin manifest 未註冊")
	}
	if m.Source.SourceType != SourceBuiltin {
		t.Errorf("expected SourceBuiltin, got %s", m.Source.SourceType)
	}
}

// TestRegisterDocumentBuiltinsCount 驗證 4 個文件 builtin 全部註冊。
func TestRegisterDocumentBuiltinsCount(t *testing.T) {
	dir, _ := os.MkdirTemp("", "router-test-*")
	defer os.RemoveAll(dir)

	archive := NewArchiveService(dir)
	r := NewRouter(archive)
	RegisterDocumentBuiltins(r)

	if got := len(r.builtinManifests); got != 7 {
		t.Errorf("expected 7 builtin manifests, got %d", got)
	}

	expected := []string{
		"builtin.document.import",
		"builtin.document.read",
		"builtin.document.write",
		"builtin.document.export",
		"builtin.local.search",
		"builtin.scheduler",
		"builtin.git.status",
	}
	for _, id := range expected {
		if r.builtinManifests[id] == nil {
			t.Errorf("missing builtin: %s", id)
		}
	}
}

// TestBuiltinMergedInResolve 驗證 Resolve 時 builtin 參與評分。
func TestBuiltinMergedInResolve(t *testing.T) {
	dir, _ := os.MkdirTemp("", "router-test-*")
	defer os.RemoveAll(dir)

	archive := NewArchiveService(dir)
	r := NewRouter(archive)
	RegisterDocumentBuiltins(r)

	at := ActionTarget{Action: "讀取", Target: "文件"}
	result, err := r.Resolve(at, "test-session")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}

	// 應該有至少一個候選（builtin.document.read）
	if len(result.Candidates) == 0 {
		t.Error("expected at least 1 candidate from builtin, got 0")
	}
}

func TestActionTagsIncludesDocumentBuiltins(t *testing.T) {
	dir, _ := os.MkdirTemp("", "router-test-*")
	defer os.RemoveAll(dir)

	archive := NewArchiveService(dir)
	r := NewRouter(archive)
	RegisterDocumentBuiltins(r)

	tags, err := r.ActionTags()
	if err != nil {
		t.Fatalf("ActionTags error: %v", err)
	}
	for _, want := range []string{"匯入", "讀取", "寫入", "匯出", "搜尋", "排程", "定時", "版控", "git"} {
		if !slices.Contains(tags, want) {
			t.Fatalf("missing builtin action tag %q in %#v", want, tags)
		}
	}
}

// TestSchedulerBuiltinResolve 驗證排程 action tag 可被 Resolve 匹配。
func TestSchedulerBuiltinResolve(t *testing.T) {
	dir, _ := os.MkdirTemp("", "router-test-*")
	defer os.RemoveAll(dir)

	archive := NewArchiveService(dir)
	r := NewRouter(archive)
	RegisterDocumentBuiltins(r)

	at := ActionTarget{Action: "排程", Target: "每天早上9點"}
	result, err := r.Resolve(at, "test-session")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}

	found := false
	for _, c := range result.Candidates {
		if c.SkillID == "builtin.scheduler" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected builtin.scheduler in candidates, got %+v", result.Candidates)
	}
}

// TestGitStatusBuiltinResolve 驗證版控 action tag 可被 Resolve 匹配。
func TestGitStatusBuiltinResolve(t *testing.T) {
	dir, _ := os.MkdirTemp("", "router-test-*")
	defer os.RemoveAll(dir)

	archive := NewArchiveService(dir)
	r := NewRouter(archive)
	RegisterDocumentBuiltins(r)

	at := ActionTarget{Action: "版控", Target: "查看狀態"}
	result, err := r.Resolve(at, "test-session")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}

	found := false
	for _, c := range result.Candidates {
		if c.SkillID == "builtin.git.status" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected builtin.git.status in candidates, got %+v", result.Candidates)
	}
}
