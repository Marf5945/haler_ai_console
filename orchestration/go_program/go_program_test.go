package go_program

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateBuildExecuteStdlibProgram(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go binary not available")
	}
	src := t.TempDir()
	writeTestFile(t, filepath.Join(src, "main.go"), `package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	var in map[string]interface{}
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		panic(err)
	}
	out := map[string]interface{}{"recommendation": fmt.Sprintf("ok:%v", in["weather"])}
	_ = json.NewEncoder(os.Stdout).Encode(out)
}
`)
	manifest := Manifest{
		ProgramID:    "weather-clothes",
		DisplayName:  "穿衣服比較",
		SourceDir:    src,
		OutputSchema: ObjectSchema{Required: []string{"recommendation"}},
	}
	toolchain := Toolchain{GoBinary: goBin, Version: "test"}
	vr, err := Validate(manifest, toolchain)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if vr.HasIssues() {
		t.Fatalf("unexpected validation issues: %+v", vr.Issues)
	}
	buildDir := filepath.Join(t.TempDir(), "build")
	br, err := Build(context.Background(), manifest, toolchain, buildDir, Limits{BuildTimeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("Build: %v stderr=%s", err, br.Stderr)
	}
	input := []byte(`{"weather":"rain"}`)
	if err := ValidateJSONInput(ObjectSchema{Required: []string{"weather"}}, input); err != nil {
		t.Fatalf("ValidateJSONInput: %v", err)
	}
	ex, err := Execute(context.Background(), br.BinaryPath, input, Limits{ExecuteTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, string(ex.Stderr))
	}
	if err := ValidateJSONOutput(manifest.OutputSchema, ex.Stdout); err != nil {
		t.Fatalf("ValidateJSONOutput: %v stdout=%s", err, string(ex.Stdout))
	}
}

func TestValidateThirdPartyImportRequestsReview(t *testing.T) {
	src := t.TempDir()
	writeTestFile(t, filepath.Join(src, "main.go"), `package main

import "github.com/example/notallowed"

func main() { _ = notallowed.X }
`)
	vr, err := Validate(Manifest{SourceDir: src}, Toolchain{Version: "test"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(vr.ReviewRequests) != 1 || vr.ReviewRequests[0].Kind != ReviewUnauthorizedPackage {
		t.Fatalf("expected unauthorized package review, got %+v", vr.ReviewRequests)
	}
}

func TestValidateNetworkAndShellNeedReview(t *testing.T) {
	src := t.TempDir()
	writeTestFile(t, filepath.Join(src, "main.go"), `package main

import (
	"net/http"
	"os/exec"
)

func main() {
	_, _ = http.Get("https://example.com")
	_ = exec.Command("echo", "x").Run()
}
`)
	vr, err := Validate(Manifest{SourceDir: src}, Toolchain{Version: "test"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	kinds := map[ReviewKind]bool{}
	for _, req := range vr.ReviewRequests {
		kinds[req.Kind] = true
	}
	if !kinds[ReviewNetworkRequired] || !kinds[ReviewShellRequired] {
		t.Fatalf("expected network and shell review, got %+v", vr.ReviewRequests)
	}
}

func TestAttemptStorePreservesVersionsAndDetectsNoProgress(t *testing.T) {
	root := t.TempDir()
	store := NewAttemptStore(root)
	manifest := Manifest{ProgramID: "x"}
	toolchain := Toolchain{Version: "test"}
	files := map[string]string{"main.go": "package main\nfunc main(){}\n"}
	rec1, err := store.SaveAttempt(1, manifest, toolchain, files, "same-error")
	if err != nil {
		t.Fatalf("SaveAttempt 1: %v", err)
	}
	rec2, err := store.SaveAttempt(2, manifest, toolchain, files, "same-error")
	if err != nil {
		t.Fatalf("SaveAttempt 2: %v", err)
	}
	if rec1.Hash == "" || rec2.Hash == "" {
		t.Fatal("attempt hash should be populated")
	}
	if !RepeatedProgress([]AttemptRecord{*rec1, *rec2}) {
		t.Fatal("expected repeated progress detection")
	}
	if _, err := os.Stat(filepath.Join(root, "attempt-1", "main.go")); err != nil {
		t.Fatalf("attempt-1 code missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "attempt-2", "main.go")); err != nil {
		t.Fatalf("attempt-2 code missing: %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
