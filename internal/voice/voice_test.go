package voice

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWhisperLanguageFromPanel(t *testing.T) {
	cases := map[string]string{
		"繁中":      "zh",
		"簡中":      "zh",
		"英文":      "en",
		"日文":      "ja",
		"unknown": "auto",
	}
	for input, want := range cases {
		if got := WhisperLanguageFromPanel(input); got != want {
			t.Fatalf("WhisperLanguageFromPanel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestManagedModelPathLivesNextToAppBundle(t *testing.T) {
	svc := NewService(t.TempDir(), "/tmp/work", "/tmp/build/bin/ai-console.app", "/tmp/build/bin/ai-console.app/Contents/Resources")
	want := filepath.Join("/tmp/build/bin", "voice", "models", ManagedModelFile)
	if got := svc.ManagedModelPath(); got != want {
		t.Fatalf("ManagedModelPath() = %q, want %q", got, want)
	}
}

func TestManagedRunnerPathLivesNextToAppBundle(t *testing.T) {
	svc := NewService(t.TempDir(), "/tmp/work", "/tmp/build/bin/ai-console.app", "/tmp/build/bin/ai-console.app/Contents/Resources")
	want := filepath.Join("/tmp/build/bin", "voice", ManagedRunnerFile)
	if got := svc.ManagedRunnerPath(); got != want {
		t.Fatalf("ManagedRunnerPath() = %q, want %q", got, want)
	}
}

func TestRouteCommandRequiresEnabledMode(t *testing.T) {
	route := RouteCommand("停止", false)
	if route.Matched || route.Reason != "command_mode_disabled" {
		t.Fatalf("disabled route = %#v", route)
	}
}

func TestRouteCommandAllowlist(t *testing.T) {
	cases := map[string]string{
		"停止":       "stop_active_job",
		"continue": "resume_active_job",
		"不要改檔":     "append_readonly_constraint",
	}
	for input, want := range cases {
		route := RouteCommand(input, true)
		if !route.Matched || route.Action != want {
			t.Fatalf("RouteCommand(%q) = %#v, want action %q", input, route, want)
		}
	}
}

func TestRouteCommandRejectsFreeText(t *testing.T) {
	route := RouteCommand("幫我刪掉專案", true)
	if route.Matched || route.Action != "" {
		t.Fatalf("free text should not match: %#v", route)
	}
}

func TestRunWhisperReadsTranscriptFile(t *testing.T) {
	bin := fakeWhisper(t, `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-of" ]; then
    shift
    out="$1"
  fi
  shift
done
printf 'whisper_model_load: loading model\n'
printf '你好世界\n' > "$out.txt"
`)
	dir := t.TempDir()
	got, err := runWhisper(context.Background(), bin, "model.bin", "audio.wav", filepath.Join(dir, "voice-out"), "zh")
	if err != nil {
		t.Fatalf("runWhisper: %v", err)
	}
	if got != "你好世界" {
		t.Fatalf("transcript = %q", got)
	}
}

func TestRunWhisperDoesNotReturnDiagnosticOutput(t *testing.T) {
	bin := fakeWhisper(t, `#!/bin/sh
printf 'whisper_model_load: loading model\n'
printf 'whisper_full_with_state: input is too short - 80 ms < 100 ms\n'
printf 'output_txt: saving output to file\n'
`)
	dir := t.TempDir()
	got, err := runWhisper(context.Background(), bin, "model.bin", "audio.wav", filepath.Join(dir, "voice-out"), "zh")
	if err == nil {
		t.Fatalf("expected no-speech error, got transcript %q", got)
	}
	if got != "" {
		t.Fatalf("diagnostic output must not be returned as transcript: %q", got)
	}
	if !strings.Contains(err.Error(), "no speech recognized") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunWhisperPassesAutoLanguage(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	bin := fakeWhisper(t, `#!/bin/sh
args_path="$FAKE_WHISPER_ARGS"
out=""
printf '%s\n' "$*" > "$args_path"
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-of" ]; then
    shift
    out="$1"
  fi
  shift
done
printf 'hello\n' > "$out.txt"
`)
	t.Setenv("FAKE_WHISPER_ARGS", argsPath)
	_, err := runWhisper(context.Background(), bin, "model.bin", "audio.wav", filepath.Join(dir, "voice-out"), "auto")
	if err != nil {
		t.Fatalf("runWhisper: %v", err)
	}
	raw, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	args := string(raw)
	if !strings.Contains(args, "-l auto") {
		t.Fatalf("expected -l auto in args, got %q", args)
	}
}

func fakeWhisper(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake whisper is POSIX-only")
	}
	path := filepath.Join(t.TempDir(), "whisper-fake")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake whisper: %v", err)
	}
	return path
}
