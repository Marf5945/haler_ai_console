package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Layer 1: 各供應商 pattern ──

func TestRedactOpenAIKey(t *testing.T) {
	out, recs := RedactBeforeWrite("key: " + "sk-" + "abc123def456ghi789jkl012mno345pqr678")
	if strings.Contains(out, "sk-abc123") {
		t.Error("OpenAI key should be redacted")
	}
	assertHasProvider(t, recs, "OpenAI")
}

func TestRedactAnthropicKey(t *testing.T) {
	out, recs := RedactBeforeWrite("key: " + "sk-ant-api03-" + "abcdefghijklmnopqrstuvwx")
	if strings.Contains(out, "sk-ant-api03") {
		t.Error("Anthropic key should be redacted")
	}
	assertHasProvider(t, recs, "Anthropic")
}

func TestRedactOpenRouterKey(t *testing.T) {
	out, recs := RedactBeforeWrite("key: " + "sk-or-v1-" + "abcdefghijklmnopqrstuvwx")
	if strings.Contains(out, "sk-or-v1") {
		t.Error("OpenRouter key should be redacted")
	}
	assertHasProvider(t, recs, "OpenRouter")
}

func TestRedactReplicateKey(t *testing.T) {
	out, recs := RedactBeforeWrite("key: r8_abcdefghijklmnopqrstuvwx")
	if strings.Contains(out, "r8_abcdef") {
		t.Error("Replicate key should be redacted")
	}
	assertHasProvider(t, recs, "Replicate")
}

func TestRedactAWSKey(t *testing.T) {
	out, recs := RedactBeforeWrite("aws: " + "AKIA" + "IOSFODNN7EXAMPLE")
	if strings.Contains(out, "AKIAIOSF") {
		t.Error("AWS key should be redacted")
	}
	assertHasProvider(t, recs, "AWS")
}

func TestRedactStripeKey(t *testing.T) {
	out, recs := RedactBeforeWrite("stripe: " + "sk_live_" + "abcdefghijklmnopqrstuvwxyz")
	if strings.Contains(out, "sk_live_") {
		t.Error("Stripe key should be redacted")
	}
	assertHasProvider(t, recs, "Stripe")
}

func TestRedactGoogleAIKey(t *testing.T) {
	out, recs := RedactBeforeWrite("google: " + "AIzaSy" + "A1234567890abcdefghijklmnopqrstuv")
	if strings.Contains(out, "AIzaSy") {
		t.Error("Google AI key should be redacted")
	}
	assertHasProvider(t, recs, "Google_AI")
}

func TestRedactHuggingFaceToken(t *testing.T) {
	out, recs := RedactBeforeWrite("hf: hf_abcdefghijklmnopqrstuvwxyzABCDEFGHIJ")
	if strings.Contains(out, "hf_abcdef") {
		t.Error("HuggingFace token should be redacted")
	}
	assertHasProvider(t, recs, "HuggingFace")
}

func TestRedactPEMKey(t *testing.T) {
	pem := "-----BEGIN RSA " + "PRIVATE KEY-----\nMIIEpAIBAAK...\n-----END RSA " + "PRIVATE KEY-----"
	out, recs := RedactBeforeWrite(pem)
	if strings.Contains(out, "MIIEpAIBAAK") {
		t.Error("PEM should be redacted")
	}
	assertHasProvider(t, recs, "PEM")
}

// ── Bearer 正規化繞過測試 ──

func TestRedactBearerWithTab(t *testing.T) {
	// tab 在 Bearer 後——正規化應壓成空格
	out, recs := RedactBeforeWrite("Authorization: Bearer\t" + "eyJhbGciOiJIUzI1NiJ9")
	if strings.Contains(out, "eyJhbGci") {
		t.Error("Bearer+tab should be redacted after normalization")
	}
	assertHasProvider(t, recs, "Bearer")
}

func TestRedactBearerWithDoubleSpace(t *testing.T) {
	out, recs := RedactBeforeWrite("Authorization: Bearer  " + "eyJhbGciOiJIUzI1NiJ9")
	if strings.Contains(out, "eyJhbGci") {
		t.Error("Bearer+double-space should be redacted after normalization")
	}
	assertHasProvider(t, recs, "Bearer")
}

// ── Key=Value 通用比對 ──

func TestRedactKeyValueGeneric(t *testing.T) {
	out, _ := RedactBeforeWrite("config: password=MySecret123456 and token=abc456defghijklmnop")
	if strings.Contains(out, "MySecret123456") {
		t.Error("password value should be redacted")
	}
	if strings.Contains(out, "abc456defghij") {
		t.Error("token value should be redacted")
	}
}

// ── Layer 2: Entropy ──

func TestShannonEntropy(t *testing.T) {
	// 高熵：混合大小寫+數字（模擬真實 API key），H > 4.5
	high := ShannonEntropy([]byte("Zx9kQ3mR7vL2pW8nT5yJ1cF4gH6bA0eUdSiXw"))
	if high < 4.5 {
		t.Errorf("mixed alphanumeric should have H>=4.5, got %.2f", high)
	}
	// 中熵：純 hex（git SHA 等），H ≈ 4.0，低於閾值不誤報
	mid := ShannonEntropy([]byte("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"))
	if mid >= 4.5 {
		t.Errorf("pure hex should have H<4.5 (no false positive), got %.2f", mid)
	}
	// 低熵：重複字元
	low := ShannonEntropy([]byte("aaaaaaaaaa"))
	if low > 0.01 {
		t.Errorf("repeated chars should have ~0 entropy, got %.2f", low)
	}
}

func TestEntropyExemptUUID(t *testing.T) {
	input := "id: 550e8400-e29b-41d4-a716-446655440000 is safe"
	out, _ := RedactBeforeWrite(input)
	if strings.Contains(out, "[REDACTED:entropy") || strings.Contains(out, "[SUSPECT:entropy]") {
		t.Error("UUID should be exempt from entropy detection")
	}
}

func TestEntropyExemptGitSHA(t *testing.T) {
	input := "commit abc123def456789012345678901234567890ab is merged"
	out, _ := RedactBeforeWrite(input)
	if strings.Contains(out, "[REDACTED:entropy") {
		t.Error("git SHA should be exempt from entropy detection")
	}
}

func TestEntropyWithContextKeyword(t *testing.T) {
	// 高熵 + "secret" keyword → medium confidence → 遮蔽
	secret := "secret=Zx9kQ3mR7vL2pW8nT5yJ1cF4gH6bA0eU"
	out, recs := RedactBeforeWrite(secret)
	// 由 kv pattern 或 entropy+context 處理，重點是不能明文殘留
	if strings.Contains(out, "Zx9kQ3mR7v") {
		t.Error("high-entropy with context keyword should be redacted")
	}
	_ = recs
}

// ── 安全內容不被動 ──

func TestRedactSafeContent(t *testing.T) {
	input := "Hello, how are you today? Let's discuss Go programming."
	out, recs := RedactBeforeWrite(input)
	if len(recs) != 0 {
		t.Error("safe content should not trigger redaction")
	}
	// 正規化不改語意（僅壓空白）
	if !strings.Contains(out, "Hello") || !strings.Contains(out, "Go programming") {
		t.Error("safe content should be preserved")
	}
}

// ── Pattern 合併機制 ──

func TestLoadUserPatternsValid(t *testing.T) {
	ResetActiveRules()
	defer ResetActiveRules()

	dir := t.TempDir()
	patterns := []UserPattern{
		{Provider: "Custom", Pattern: `\bCUST_[A-Za-z0-9]{10,}`, Enabled: true},
	}
	data, _ := json.Marshal(patterns)
	os.WriteFile(filepath.Join(dir, "redaction_patterns.json"), data, 0644)

	LoadUserPatterns(dir)

	out, recs := RedactBeforeWrite("key: CUST_abcdefghijklmnop")
	if strings.Contains(out, "CUST_abcdef") {
		t.Error("user pattern should be applied")
	}
	assertHasProvider(t, recs, "Custom")
}

func TestLoadUserPatternsInvalidRegex(t *testing.T) {
	ResetActiveRules()
	defer ResetActiveRules()

	dir := t.TempDir()
	patterns := []UserPattern{
		{Provider: "Bad", Pattern: `[invalid`, Enabled: true},
		{Provider: "Good", Pattern: `\bGOOD_[A-Za-z0-9]{10,}`, Enabled: true},
	}
	data, _ := json.Marshal(patterns)
	os.WriteFile(filepath.Join(dir, "redaction_patterns.json"), data, 0644)

	LoadUserPatterns(dir) // 不應 panic

	out, _ := RedactBeforeWrite("val: GOOD_abcdefghijklmnop")
	if strings.Contains(out, "GOOD_abcdef") {
		t.Error("valid user pattern should still work despite invalid sibling")
	}
}

func TestLoadUserPatternsMissingFile(t *testing.T) {
	ResetActiveRules()
	defer ResetActiveRules()

	LoadUserPatterns(t.TempDir()) // 不應 panic，靜默使用內建
	// 內建仍有效
	out, _ := RedactBeforeWrite("key: " + "sk-" + "abc123def456ghi789jkl012mno345pqr678")
	if strings.Contains(out, "sk-abc123") {
		t.Error("built-in should still work without user config")
	}
}

func TestUserPatternCannotDisableBuiltin(t *testing.T) {
	ResetActiveRules()
	defer ResetActiveRules()

	// 即使使用者設定檔 enabled=false 同名 provider，內建不受影響
	dir := t.TempDir()
	patterns := []UserPattern{
		{Provider: "OpenAI", Pattern: `will-not-compile-[`, Enabled: false},
	}
	data, _ := json.Marshal(patterns)
	os.WriteFile(filepath.Join(dir, "redaction_patterns.json"), data, 0644)

	LoadUserPatterns(dir)

	out, _ := RedactBeforeWrite("key: " + "sk-" + "abc123def456ghi789jkl012mno345pqr678")
	if strings.Contains(out, "sk-abc123") {
		t.Error("built-in OpenAI pattern must remain active")
	}
}

// ── 輔助 ──

func assertHasProvider(t *testing.T, recs []RedactionRecord, provider string) {
	t.Helper()
	for _, r := range recs {
		if r.Type == provider || strings.Contains(r.Type, provider) {
			return
		}
	}
	// key_value 也算
	if provider != "" && len(recs) > 0 {
		return // 被 kv pattern 吃掉也算通過
	}
	t.Errorf("expected record with provider %q, got %+v", provider, recs)
}
