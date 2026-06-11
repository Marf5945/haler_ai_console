package voice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ui_console/audit_log"
)

const (
	TTSPackKokoroZH      = "kokoro_zh"
	ManagedTTSPackFile   = "kokoro-zh.voicepack"
	ttsPackDownloadSlack = 64 * 1024 * 1024
)

type TTSPackSpec struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Engine        string `json:"engine"`
	License       string `json:"license"`
	SourceURL     string `json:"source_url,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
	ExpectedBytes int64  `json:"expected_bytes,omitempty"`
	MaxBytes      int64  `json:"max_bytes,omitempty"`
}

type TTSPackStatus struct {
	PackID          string `json:"packId"`
	Label           string `json:"label"`
	Engine          string `json:"engine"`
	License         string `json:"license"`
	ManagedPackPath string `json:"managedPackPath"`
	Available       bool   `json:"available"`
	Configured      bool   `json:"configured"`
	RequiresPayment bool   `json:"requiresPayment"`
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
	ExpectedBytes   int64  `json:"expectedBytes,omitempty"`
}

type TTSAuditEntry struct {
	CreatedAt     time.Time `json:"created_at"`
	Event         string    `json:"event"`
	PackID        string    `json:"pack_id"`
	Outcome       string    `json:"outcome"`
	BytesExpected int64     `json:"bytes_expected,omitempty"`
	BytesWritten  int64     `json:"bytes_written,omitempty"`
	Error         string    `json:"error,omitempty"`
}

// Keep this unconfigured until a native, non-pickle Kokoro zh artifact is pinned.
var ttsPackSpecs = map[string]TTSPackSpec{
	TTSPackKokoroZH: {
		ID:            TTSPackKokoroZH,
		Label:         "Kokoro Chinese local voice pack",
		Engine:        "kokoro_zh_native",
		License:       "Apache-2.0 model; runtime must be permissive",
		ExpectedBytes: 90 * 1024 * 1024,
		MaxBytes:      350 * 1024 * 1024,
	},
}

func (s *Service) TTSPackStatus() TTSPackStatus {
	return s.ttsPackStatus(ttsPackSpecs[TTSPackKokoroZH], "")
}

func (s *Service) ManagedTTSPackPath() string {
	return filepath.Join(s.managedBase(), "voice", "tts", ManagedTTSPackFile)
}

func (s *Service) InstallTTSPack(ctx context.Context) (TTSPackStatus, error) {
	spec := ttsPackSpecs[TTSPackKokoroZH]
	status := s.ttsPackStatus(spec, "")
	if err := validateTTSPackSpec(spec); err != nil {
		status.Status = "not_configured"
		status.Reason = err.Error()
		_ = s.appendTTSAudit(TTSAuditEntry{Event: "install_tts_pack", PackID: spec.ID, Outcome: "blocked", Error: err.Error(), BytesExpected: spec.ExpectedBytes})
		return status, err
	}
	target := s.ManagedTTSPackPath()
	if err := ensureDiskForDownload(filepath.Dir(target), spec.ExpectedBytes); err != nil {
		status.Status = "insufficient_space"
		status.Reason = err.Error()
		_ = s.appendTTSAudit(TTSAuditEntry{Event: "install_tts_pack", PackID: spec.ID, Outcome: "blocked", Error: err.Error(), BytesExpected: spec.ExpectedBytes})
		return status, err
	}
	written, err := downloadPinnedTTSPack(ctx, spec, target)
	if err != nil {
		status.Status = "install_failed"
		status.Reason = err.Error()
		_ = s.appendTTSAudit(TTSAuditEntry{Event: "install_tts_pack", PackID: spec.ID, Outcome: "failed", Error: err.Error(), BytesExpected: spec.ExpectedBytes})
		return status, err
	}
	_ = s.appendTTSAudit(TTSAuditEntry{Event: "install_tts_pack", PackID: spec.ID, Outcome: "ok", BytesExpected: spec.ExpectedBytes, BytesWritten: written})
	return s.ttsPackStatus(spec, ""), nil
}

func (s *Service) RemoveTTSPack() (TTSPackStatus, error) {
	spec := ttsPackSpecs[TTSPackKokoroZH]
	target := s.ManagedTTSPackPath()
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		_ = s.appendTTSAudit(TTSAuditEntry{Event: "remove_tts_pack", PackID: spec.ID, Outcome: "failed", Error: err.Error()})
		return s.ttsPackStatus(spec, err.Error()), err
	}
	_ = os.Remove(target + ".download")
	_ = s.appendTTSAudit(TTSAuditEntry{Event: "remove_tts_pack", PackID: spec.ID, Outcome: "ok"})
	return s.ttsPackStatus(spec, ""), nil
}

func (s *Service) ttsPackStatus(spec TTSPackSpec, reason string) TTSPackStatus {
	path := s.ManagedTTSPackPath()
	configured := strings.TrimSpace(spec.SourceURL) != "" && strings.TrimSpace(spec.SHA256) != ""
	status := TTSPackStatus{
		PackID:          spec.ID,
		Label:           spec.Label,
		Engine:          spec.Engine,
		License:         spec.License,
		ManagedPackPath: path,
		Configured:      configured,
		RequiresPayment: false,
		Status:          "missing_pack",
		Reason:          reason,
		ExpectedBytes:   spec.ExpectedBytes,
	}
	if !configured {
		status.Status = "not_configured"
		if status.Reason == "" {
			status.Reason = "native Kokoro zh voice pack artifact is not pinned yet"
		}
		return status
	}
	if err := verifyTTSPackFile(path, spec); err != nil {
		status.Reason = err.Error()
		if os.IsNotExist(err) {
			status.Status = "missing_pack"
		} else {
			status.Status = "invalid_pack"
		}
		return status
	}
	status.Available = true
	status.Status = "ready"
	status.Reason = ""
	return status
}

func validateTTSPackSpec(spec TTSPackSpec) error {
	if strings.TrimSpace(spec.ID) == "" {
		return fmt.Errorf("tts pack id is empty")
	}
	if strings.TrimSpace(spec.SourceURL) == "" || strings.TrimSpace(spec.SHA256) == "" {
		return fmt.Errorf("tts pack %s is not pinned", spec.ID)
	}
	if err := validateManagedTTSPackURL(spec.SourceURL); err != nil {
		return err
	}
	if _, err := hex.DecodeString(strings.TrimSpace(spec.SHA256)); err != nil {
		return fmt.Errorf("tts pack %s sha256 is invalid: %w", spec.ID, err)
	}
	return nil
}

func validateManagedTTSPackURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("voice tts: invalid pack URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("voice tts: pack URL must use https")
	}
	if parsed.Host != "huggingface.co" && parsed.Host != "github.com" {
		return fmt.Errorf("voice tts: pack URL host %q is not allowed", parsed.Host)
	}
	if strings.Contains(parsed.Path, "..") {
		return fmt.Errorf("voice tts: pack URL path is not allowed")
	}
	return nil
}

func ensureDiskForDownload(dir string, expected int64) error {
	if expected <= 0 {
		return nil
	}
	available, ok := availableDiskBytes(dir)
	if !ok {
		return nil
	}
	need := uint64(expected + ttsPackDownloadSlack)
	if available < need {
		return fmt.Errorf("voice tts: insufficient disk space; need %d bytes, available %d", need, available)
	}
	return nil
}

func downloadPinnedTTSPack(ctx context.Context, spec TTSPackSpec, target string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return 0, fmt.Errorf("voice tts: create pack dir: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.SourceURL, nil)
	if err != nil {
		return 0, fmt.Errorf("voice tts: create pack request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("voice tts: download pack: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("voice tts: download pack HTTP %d", resp.StatusCode)
	}
	if spec.MaxBytes > 0 && resp.ContentLength > spec.MaxBytes {
		return 0, fmt.Errorf("voice tts: pack too large (%d bytes)", resp.ContentLength)
	}
	tmp := target + ".download"
	_ = os.Remove(tmp)
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, fmt.Errorf("voice tts: create pack file: %w", err)
	}
	hasher := sha256.New()
	limited := io.Reader(resp.Body)
	if spec.MaxBytes > 0 {
		limited = io.LimitReader(resp.Body, spec.MaxBytes+1)
	}
	written, copyErr := io.Copy(io.MultiWriter(file, hasher), limited)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return written, fmt.Errorf("voice tts: write pack: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return written, fmt.Errorf("voice tts: close pack file: %w", closeErr)
	}
	if spec.MaxBytes > 0 && written > spec.MaxBytes {
		_ = os.Remove(tmp)
		return written, fmt.Errorf("voice tts: pack exceeds max size")
	}
	if sum := hex.EncodeToString(hasher.Sum(nil)); !strings.EqualFold(sum, spec.SHA256) {
		_ = os.Remove(tmp)
		return written, fmt.Errorf("voice tts: pack checksum mismatch")
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return written, fmt.Errorf("voice tts: install pack file: %w", err)
	}
	return written, nil
}

func verifyTTSPackFile(path string, spec TTSPackSpec) error {
	if err := validateTTSPackSpec(spec); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	if sum := hex.EncodeToString(hasher.Sum(nil)); !strings.EqualFold(sum, spec.SHA256) {
		return fmt.Errorf("voice tts: pack checksum mismatch")
	}
	return nil
}

func (s *Service) appendTTSAudit(entry TTSAuditEntry) error {
	log := audit_log.New[TTSAuditEntry](
		filepath.Join(s.root, "audit_log", "voice_tts.jsonl"),
		audit_log.WithSkipCorruptLines[TTSAuditEntry](),
		audit_log.WithBeforeAppend[TTSAuditEntry](func(e *TTSAuditEntry) error {
			if e.CreatedAt.IsZero() {
				e.CreatedAt = time.Now()
			}
			return nil
		}),
	)
	return log.Append(entry)
}

type CloudTTSEgressPreview struct {
	Allowed              bool     `json:"allowed"`
	RequiresConfirmation bool     `json:"requiresConfirmation"`
	MaskedText           string   `json:"maskedText"`
	HitCount             int      `json:"hitCount"`
	HitTypes             []string `json:"hitTypes"`
}
