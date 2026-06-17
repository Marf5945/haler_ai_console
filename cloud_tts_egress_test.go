package main

import (
	"strings"
	"testing"
)

func TestPrepareCloudTTSTextRedactsSecrets(t *testing.T) {
	secret := "sk-" + "abcdefghijklmnopqrstuvwxyz123456"
	preview := prepareCloudTTSEgressPreview("請朗讀這段 key: " + secret)
	if preview.Allowed {
		t.Fatal("cloud TTS text with secrets must not be allowed directly")
	}
	if !preview.RequiresConfirmation {
		t.Fatal("cloud TTS secret hit should require confirmation")
	}
	if preview.HitCount == 0 {
		t.Fatal("expected at least one redaction hit")
	}
	if strings.Contains(preview.MaskedText, secret) {
		t.Fatal("masked cloud TTS text must not contain the raw secret")
	}
}

func TestPrepareCloudTTSTextAllowsPlainText(t *testing.T) {
	preview := prepareCloudTTSEgressPreview("Task finished. The summary is ready.")
	if !preview.Allowed || preview.RequiresConfirmation || preview.HitCount != 0 {
		t.Fatalf("plain text should pass without confirmation: %#v", preview)
	}
}
