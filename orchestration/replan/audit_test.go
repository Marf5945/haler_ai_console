package replan

import (
	"strings"
	"testing"
)

func TestSafeSummary_RedactsPathAndToken(t *testing.T) {
	in := "no match at /Users/tester/secret/creds.json token ABCDEFGHIJKLMNOPQRSTUVWXYZ012345xyz"
	out := SafeSummary(in)
	if strings.Contains(out, "secret") || strings.Contains(out, "creds.json") {
		t.Errorf("path not redacted: %q", out)
	}
	if strings.Contains(out, "ABCDEFGHIJKLMNOPQRSTUVWXYZ012345") {
		t.Errorf("token not redacted: %q", out)
	}
}

func TestSafeSummary_Truncates(t *testing.T) {
	out := SafeSummary(strings.Repeat("a ", 250)) // 帶空格避免被 token regex 整段遮蔽
	if len([]rune(out)) > 210 {                   // 200 + 省略號餘裕
		t.Errorf("summary not truncated, len=%d", len([]rune(out)))
	}
}

func TestAuditLog_AppendAndRead(t *testing.T) {
	dir := t.TempDir()
	log := NewAuditLog(dir)
	err := AppendAuditEntry(log, ReplanAuditEntry{
		RunID:         "dag-1",
		TriggerReason: "no match at /Users/tester/secret/x",
		Decision:      DecisionSilent,
		Silent:        true,
	})
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	entries, err := log.ReadAll()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Timestamp == "" {
		t.Errorf("timestamp should be auto-filled")
	}
	if strings.Contains(e.TriggerReason, "secret") {
		t.Errorf("stored trigger reason must be scrubbed: %q", e.TriggerReason)
	}
}
