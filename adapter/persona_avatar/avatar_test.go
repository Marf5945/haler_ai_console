package persona_avatar

import (
	"os"
	"path/filepath"
	"testing"
)

// 測試預設 config 回傳 pixel fallback
func TestDefaultConfig(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	config := svc.GetCurrentAvatar("test-persona")
	if config.AvatarProvider != ProviderPixel {
		t.Errorf("default provider = %s, want built_in_pixel", config.AvatarProvider)
	}
	if config.OutputSize != 128 {
		t.Errorf("default size = %d, want 128", config.OutputSize)
	}
}

// 測試 ResolveProvider fallback 順序
func TestResolveProviderFallback(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// 無任何設定 → pixel
	provider := svc.ResolveProvider("p1")
	if provider != ProviderPixel {
		t.Errorf("empty config should resolve to pixel, got %s", provider)
	}
}

// 測試狀態觸發器映射
func TestGetStateTrigger(t *testing.T) {
	cases := []struct {
		task, risk string
		want       AvatarStateTrigger
	}{
		{"running", "", StateWorking},
		{"thinking", "", StateThinking},
		{"completed", "", StateHappy},
		{"idle", "", StateIdle},
		{"sleeping", "", StateSleepy},
		{"", "critical_runtime_action", StateBlocked},
		{"", "security_boundary_rewrite", StateBlocked},
		{"running", "high_non_destructive", StateWarning}, // 風險優先
	}
	for _, c := range cases {
		got := GetStateTrigger(c.task, c.risk)
		if got != c.want {
			t.Errorf("GetStateTrigger(%q, %q) = %s, want %s", c.task, c.risk, got, c.want)
		}
	}
}

// 測試 Prompt 組合
func TestComposePrompt(t *testing.T) {
	preset, err := GetPresetByID("cyberpunk_helper")
	if err != nil {
		t.Fatalf("preset not found: %v", err)
	}

	prompt, err := ComposePrompt(*preset, StateWorking)
	if err != nil {
		t.Fatalf("compose failed: %v", err)
	}

	if prompt == "" {
		t.Error("prompt should not be empty")
	}
	// 不應包含 {state_prompt} 佔位符
	if contains(prompt, "{state_prompt}") {
		t.Error("prompt should not contain placeholder")
	}
}

// 測試未知 preset ID
func TestGetPresetByIDNotFound(t *testing.T) {
	_, err := GetPresetByID("nonexistent")
	if err == nil {
		t.Error("should return error for unknown preset")
	}
}

// 測試 Pixel Avatar 渲染
func TestRenderPixelAvatar(t *testing.T) {
	for _, state := range AllStateTriggers {
		data, err := RenderPixelAvatar(state, 128)
		if err != nil {
			t.Errorf("RenderPixelAvatar(%s, 128) failed: %v", state, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("RenderPixelAvatar(%s, 128) returned empty data", state)
		}
		// 驗證 PNG magic bytes
		if len(data) < 4 || data[0] != 0x89 || data[1] != 0x50 || data[2] != 0x4E || data[3] != 0x47 {
			t.Errorf("RenderPixelAvatar(%s) output is not valid PNG", state)
		}
	}
}

// 測試非支援尺寸會 fallback 到省資源的 128x128 渲染
func TestRenderPixelAvatarUnsupportedSizeFallsBack(t *testing.T) {
	data, err := RenderPixelAvatar(StateIdle, 64)
	if err != nil {
		t.Fatalf("render fallback failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("empty PNG")
	}
}

// 測試批次產生所有 pixel avatar
func TestGenerateAllPixelAvatars(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "pixels")

	err := GenerateAllPixelAvatars(outputDir)
	if err != nil {
		t.Fatalf("GenerateAllPixelAvatars failed: %v", err)
	}

	// 驗證每個支援狀態都有一張 128x128 PNG
	entries, _ := os.ReadDir(outputDir)
	expected := len(AllStateTriggers)
	if len(entries) != expected {
		t.Errorf("expected %d PNG files, got %d", expected, len(entries))
	}
}

// TestCredentialStoreRoundTrip 已搬移至 domain/credential/store_test.go
// （TASKS_1_6_3 Step 5：統一 CredentialStore 後移除舊測試）

// 測試 SetProvider + 讀回
func TestSetProvider(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// 確保目錄存在
	os.MkdirAll(filepath.Join(tmpDir, "data", "personas", "p1", "avatar"), 0755)

	err := svc.SetProvider("p1", ProviderStaticImage)
	if err != nil {
		t.Fatalf("SetProvider failed: %v", err)
	}

	config := svc.GetCurrentAvatar("p1")
	if config.AvatarProvider != ProviderStaticImage {
		t.Errorf("provider = %s, want static_image", config.AvatarProvider)
	}
}

func TestLockedPersonaAvatarCannotChange(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	if err := svc.SetProvider("persona-a", ProviderStaticImage); err == nil {
		t.Fatal("locked persona should reject provider changes")
	}
	if err := svc.SetPixelPack("persona-a", "uncle"); err == nil {
		t.Fatal("locked persona should reject pixel pack changes")
	}
	if err := svc.DeleteStaticAvatar("persona-a"); err == nil {
		t.Fatal("locked persona should reject static avatar deletion")
	}

	config := svc.GetCurrentAvatar("persona-a")
	if config.AvatarProvider != ProviderPixel || config.PixelPack != "wolf" {
		t.Fatalf("locked persona should stay on wolf pixel avatar, got %#v", config)
	}
}

// 測試 DeleteStaticAvatar fallback 到 pixel
func TestDeleteStaticAvatarFallback(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	personaID := "del-test"
	avatarDir := filepath.Join(tmpDir, "data", "personas", personaID, "avatar")
	os.MkdirAll(avatarDir, 0755)

	// 先設定為 static
	svc.SetProvider(personaID, ProviderStaticImage)

	// 刪除
	err := svc.DeleteStaticAvatar(personaID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	config := svc.GetCurrentAvatar(personaID)
	if config.AvatarProvider != ProviderPixel {
		t.Errorf("after delete should be pixel, got %s", config.AvatarProvider)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
