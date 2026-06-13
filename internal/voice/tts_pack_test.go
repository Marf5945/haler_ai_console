package voice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTTSPackStartsUnconfigured(t *testing.T) {
	svc := NewService(t.TempDir(), "/tmp/work", "/tmp/build/bin/ai-console.app", "")
	status := svc.TTSPackStatus()
	if status.Configured {
		t.Fatalf("tts pack should not be configured until a pinned native artifact exists: %#v", status)
	}
	if status.RequiresPayment {
		t.Fatal("built-in TTS pack must not require payment")
	}
	if status.Status != "not_configured" {
		t.Fatalf("status = %q, want not_configured", status.Status)
	}
}

func TestInstallTTSPackRejectsUnpinnedPack(t *testing.T) {
	svc := NewService(t.TempDir(), "/tmp/work", "/tmp/build/bin/ai-console.app", "")
	status, err := svc.InstallTTSPack(context.Background())
	if err == nil {
		t.Fatal("InstallTTSPack should reject unpinned pack")
	}
	if status.Status != "not_configured" {
		t.Fatalf("status = %q, want not_configured", status.Status)
	}
}

func TestValidateManagedTTSPackURLAllowlist(t *testing.T) {
	good := []string{
		"https://huggingface.co/hexgrad/Kokoro-82M-v1.1-zh/resolve/main/model.voicepack",
		"https://github.com/example/releases/download/v1/model.voicepack",
	}
	for _, raw := range good {
		if err := validateManagedTTSPackURL(raw); err != nil {
			t.Fatalf("valid URL rejected %q: %v", raw, err)
		}
	}
	bad := []string{
		"http://huggingface.co/hexgrad/model.voicepack",
		"https://example.com/model.voicepack",
		"https://huggingface.co/hexgrad/../model.voicepack",
	}
	for _, raw := range bad {
		if err := validateManagedTTSPackURL(raw); err == nil {
			t.Fatalf("invalid URL accepted %q", raw)
		}
	}
}

func TestVerifyTTSPackFilePinsSHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pack.voicepack")
	body := []byte("small test pack")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write pack: %v", err)
	}
	sum := sha256.Sum256(body)
	spec := TTSPackSpec{
		ID:        TTSPackKokoroZH,
		SourceURL: "https://huggingface.co/hexgrad/Kokoro-82M-v1.1-zh/resolve/main/model.voicepack",
		SHA256:    hex.EncodeToString(sum[:]),
	}
	if err := verifyTTSPackFile(path, spec); err != nil {
		t.Fatalf("verifyTTSPackFile: %v", err)
	}
	spec.SHA256 = strings.Repeat("0", 64)
	if err := verifyTTSPackFile(path, spec); err == nil {
		t.Fatal("verifyTTSPackFile should reject checksum mismatch")
	}
}

func TestEnsureDiskForDownloadDoesNotFailWhenSmall(t *testing.T) {
	if err := ensureDiskForDownload(t.TempDir(), 1024); err != nil {
		t.Fatalf("small download should pass disk guard: %v", err)
	}
}
