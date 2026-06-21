package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildVisionOrganRequestBody(t *testing.T) {
	imgs := []composerImage{
		{MIME: "image/png", DataB64: "AAAA"},
		{MIME: "image/jpeg", DataB64: "BBBB"},
	}
	body, err := buildVisionOrganRequestBody("qwen2.5vl", "這張圖在說什麼？", imgs)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	var req voOllamaReq
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if req.Model != "qwen2.5vl" {
		t.Errorf("model = %q", req.Model)
	}
	if req.Stream {
		t.Errorf("stream should be false")
	}
	if len(req.Images) != 2 || req.Images[0] != "AAAA" || req.Images[1] != "BBBB" {
		t.Errorf("images = %v", req.Images)
	}
	if !strings.Contains(req.Prompt, "這張圖在說什麼？") {
		t.Errorf("prompt should embed user question, got: %s", req.Prompt)
	}
}

func TestBuildVisionOrganRequestBody_NoImages(t *testing.T) {
	if _, err := buildVisionOrganRequestBody("qwen2.5vl", "x", nil); err == nil {
		t.Fatal("expected error for no images")
	}
}

func TestBuildVisionOrganPrompt_NoQuestion(t *testing.T) {
	p := buildVisionOrganPrompt("   ")
	if strings.Contains(p, "使用者針對這張圖的問題") {
		t.Errorf("blank question should not produce focused clause: %s", p)
	}
	if !strings.Contains(p, "沒有附加問題") {
		t.Errorf("expected generic-description clause: %s", p)
	}
}

func TestBrainIsMultimodal(t *testing.T) {
	cases := []struct {
		adapterID, model string
		want             bool
	}{
		{"claude-cli", "", true},
		{"gemini-cli", "", true},
		{"codex-cli", "gpt-5.5", true},
		{"ollama-cli", "qwen2.5-vl:7b", true},
		{"ollama-cli", "llava:13b", true},
		{"ollama-cli", "moondream", true},
		{"some-api", "gpt-4o", true},
		{"ollama-cli", "qwen2.5:14b", false}, // 純文字
		{"ollama-cli", "llama3.2:1b", false},
		{"some-api", "deepseek-chat", false},
		{"", "", false},
	}
	for _, c := range cases {
		if got := brainIsMultimodal(c.adapterID, c.model); got != c.want {
			t.Errorf("brainIsMultimodal(%q,%q) = %v, want %v", c.adapterID, c.model, got, c.want)
		}
	}
}

func TestIsLoopbackEndpoint(t *testing.T) {
	ok := []string{
		"http://127.0.0.1:11434/api/generate",
		"http://localhost:11434/api/generate",
		"http://[::1]:11434/api/generate",
	}
	for _, e := range ok {
		if !isLoopbackEndpoint(e) {
			t.Errorf("%s should be loopback", e)
		}
	}
	bad := []string{
		"http://192.168.1.10:11434/api/generate",
		"https://api.openai.com/v1",
		"http://example.com",
	}
	for _, e := range bad {
		if isLoopbackEndpoint(e) {
			t.Errorf("%s should NOT be loopback", e)
		}
	}
}

func TestVisionOrganInjection(t *testing.T) {
	if got := visionOrganInjection("  "); got != "" {
		t.Errorf("blank desc should yield empty injection, got %q", got)
	}
	got := visionOrganInjection("圖中是一張發票")
	if !strings.Contains(got, "視覺器官轉述") || !strings.Contains(got, "圖中是一張發票") {
		t.Errorf("injection malformed: %s", got)
	}
}
