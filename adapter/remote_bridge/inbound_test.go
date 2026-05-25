package remote_bridge

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

func lineSignature(secret string, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestNormalizeInboundCommand(t *testing.T) {
	cases := map[string]string{
		"同意":       InboundCommandApprove,
		" 取消 ":     InboundCommandCancel,
		"狀態":       InboundCommandStatus,
		"停止":       InboundCommandStop,
		"whatever": InboundCommandUnknown,
	}
	for text, want := range cases {
		if got := NormalizeInboundCommand(text); got != want {
			t.Fatalf("NormalizeInboundCommand(%q)=%q want %q", text, got, want)
		}
	}
}

func TestParseLineInboundVerifiesSignatureAndMapsCommands(t *testing.T) {
	secret := "line-channel-secret"
	body := `{"events":[{"type":"message","source":{"type":"user","userId":"U123"},"message":{"type":"text","text":"同意"}}]}`

	result, err := ParseLineInbound("rb_line_1", []byte(body), lineSignature(secret, body), secret)
	if err != nil {
		t.Fatalf("ParseLineInbound error: %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events len=%d want 1", len(result.Events))
	}
	event := result.Events[0]
	if event.Command != InboundCommandApprove || event.SourceID != "U123" || event.Channel != string(ChannelLINE) {
		t.Fatalf("unexpected event: %+v", event)
	}

	if _, err := ParseLineInbound("rb_line_1", []byte(body), "bad-signature", secret); err == nil {
		t.Fatal("expected signature failure")
	}
}

func TestHandleInboundHTTPRequestLoadsEncryptedLineSecret(t *testing.T) {
	store := &mockSecretStore{data: make(map[string]string)}
	service := NewService(t.TempDir(), store)
	service.bindings = []ChannelBinding{{ID: "rb_line_1", Channel: ChannelLINE, TestPassed: true}}
	service.loaded = true
	if err := store.Store("remote_bridge:rb_line_1:channel_secret", "secret"); err != nil {
		t.Fatalf("store secret: %v", err)
	}
	body := `{"events":[{"type":"message","source":{"type":"group","groupId":"C123"},"message":{"type":"text","text":"狀態"}}]}`
	headers := http.Header{}
	headers.Set("X-Line-Signature", lineSignature("secret", body))

	result, err := service.HandleInboundHTTPRequest("rb_line_1", headers, []byte(body))
	if err != nil {
		t.Fatalf("HandleInboundHTTPRequest error: %v", err)
	}
	if result.Events[0].Command != InboundCommandStatus || !strings.HasPrefix(result.Events[0].SourceID, "C") {
		t.Fatalf("unexpected result: %+v", result.Events[0])
	}
}
