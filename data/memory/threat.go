// memory/threat.go — 威脅偵測與記錄（§18.7）。
// 偵測 prompt injection、異常行為等安全威脅。
// 策略：記錄 + Review Card 通知（不自動改變風險等級）。
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// 威脅記錄結構
// ──────────────────────────────────────────────

// ThreatType 威脅類型。
type ThreatType string

const (
	ThreatPromptInjection   ThreatType = "prompt_injection"
	ThreatSuspiciousPattern ThreatType = "suspicious_pattern"
	ThreatAnomalousBehavior ThreatType = "anomalous_behavior"
	ThreatCredentialLeak    ThreatType = "credential_leak"
)

// ThreatRecord 單筆威脅記錄。
type ThreatRecord struct {
	ID          string     `json:"id"`
	Type        ThreatType `json:"type"`
	Description string     `json:"description"`
	Source      string     `json:"source"`     // 偵測來源（entry_filter, redaction, validation）
	Severity    string     `json:"severity"`   // low, medium, high
	Timestamp   string     `json:"timestamp"`
	ContentHash string     `json:"content_hash"` // 觸發內容的 hash（不保存原文）
	Notified    bool       `json:"notified"`     // 是否已通知使用者
}

// ThreatDetectionResult 威脅偵測結果。
type ThreatDetectionResult struct {
	Detected bool          `json:"detected"`
	Records  []ThreatRecord `json:"records"`
}

// ──────────────────────────────────────────────
// 威脅偵測
// ──────────────────────────────────────────────

// DetectThreats 掃描內容中的威脅模式。
// 回傳偵測結果，不自動改變系統行為。
func DetectThreats(content string, source string) ThreatDetectionResult {
	var records []ThreatRecord
	lower := strings.ToLower(content)
	now := time.Now().Format(time.RFC3339)

	// 偵測 prompt injection
	for _, pattern := range injectionPatterns {
		if strings.Contains(lower, pattern) {
			records = append(records, ThreatRecord{
				ID:          fmt.Sprintf("threat-%d", time.Now().UnixNano()),
				Type:        ThreatPromptInjection,
				Description: fmt.Sprintf("偵測到疑似 prompt injection: %s", pattern),
				Source:      source,
				Severity:    "high",
				Timestamp:   now,
				ContentHash: hashValue(content),
			})
			break // 同一段內容只記錄一次 injection
		}
	}

	// 偵測 credential 洩漏
	for _, rule := range activeRules {
		if rule.pattern.MatchString(content) {
			records = append(records, ThreatRecord{
				ID:          fmt.Sprintf("threat-%d", time.Now().UnixNano()),
				Type:        ThreatCredentialLeak,
				Description: fmt.Sprintf("偵測到疑似 %s 洩漏", rule.provider),
				Source:      source,
				Severity:    "high",
				Timestamp:   now,
				ContentHash: hashValue(content),
			})
		}
	}

	// 偵測可疑模式
	for _, pattern := range suspiciousRulePatterns {
		if strings.Contains(lower, pattern) {
			records = append(records, ThreatRecord{
				ID:          fmt.Sprintf("threat-%d", time.Now().UnixNano()),
				Type:        ThreatSuspiciousPattern,
				Description: fmt.Sprintf("偵測到可疑模式: %s", pattern),
				Source:      source,
				Severity:    "medium",
				Timestamp:   now,
				ContentHash: hashValue(content),
			})
		}
	}

	return ThreatDetectionResult{
		Detected: len(records) > 0,
		Records:  records,
	}
}

// ──────────────────────────────────────────────
// 威脅記錄持久化
// ──────────────────────────────────────────────

// AppendThreatRecord 將威脅記錄追加到 deep_memory_THREAT.md。
func AppendThreatRecord(projectRoot string, record ThreatRecord) error {
	path := filepath.Join(projectRoot, "memory", FileDeepThreat)

	// 格式化為 append-only 條目
	entry := fmt.Sprintf("\n## [%s] %s — %s\n- 嚴重度: %s\n- 來源: %s\n- 描述: %s\n- Hash: %s\n",
		record.Timestamp, record.ID, record.Type,
		record.Severity, record.Source,
		record.Description, record.ContentHash,
	)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("寫入威脅記錄失敗: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}

// AppendThreatRecordJSON 將威脅記錄追加到 JSONL 格式。
// 用於程式化讀取。
func AppendThreatRecordJSON(projectRoot string, record ThreatRecord) error {
	path := filepath.Join(projectRoot, "memory", "threat_log.jsonl")

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(string(data) + "\n")
	return err
}

// ListThreatRecords 讀取 threat_log.jsonl 中的所有記錄。
func ListThreatRecords(projectRoot string) ([]ThreatRecord, error) {
	path := filepath.Join(projectRoot, "memory", "threat_log.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var records []ThreatRecord
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var r ThreatRecord
		if err := json.Unmarshal([]byte(line), &r); err == nil {
			records = append(records, r)
		}
	}
	return records, nil
}
