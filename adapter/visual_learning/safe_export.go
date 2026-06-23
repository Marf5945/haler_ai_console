package visual_learning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SafeExportManifest records what was (and was not) included in an export.
// Corresponds to schema #53 / spec #25 in TASKS_1_2.md.
//
// ALLOWED exports: Element Dictionary, Action Dictionary, canonical labels,
// DAG structure, multi-scale hash, relative bbox, risk rules, metadata, pending branches.
//
// FORBIDDEN exports: full screenshots, readable text patches, form content,
// accounts, email, tokens, API keys, passwords, payment info, any sensitive screen content.
type SafeExportManifest struct {
	ExportedAt       time.Time `json:"exported_at"`
	IncludedSections []string  `json:"included_sections"`
	ExcludedSections []string  `json:"excluded_sections"`
	RedactionSummary string    `json:"redaction_summary"`
	ManifestHash     string    `json:"manifest_hash"`
}

// allowedSections lists what Safe Export may include.
var allowedSections = []string{
	"element_dictionary",
	"action_dictionary",
	"canonical_label_schema",
	"dag_structure",
	"multi_scale_hash",
	"relative_bbox",
	"risk_rules",
	"metadata",
	"pending_branches",
}

// forbiddenSections lists what Safe Export must NEVER include.
var forbiddenSections = []string{
	"full_screenshots",
	"readable_text_patches",
	"form_content",
	"account_data",
	"email_addresses",
	"api_keys",
	"tokens",
	"passwords",
	"payment_info",
	"credential_data",
	"sensitive_screen_content",
}

// SafeExporter performs the export filter and writes the manifest.
type SafeExporter struct {
	exportDir string
}

func NewSafeExporter(learnDir string) *SafeExporter {
	return &SafeExporter{
		exportDir: filepath.Join(learnDir, "export"),
	}
}

// Export validates the requested sections against the allowlist/denylist and
// writes a SafeExportManifest. Returns an error if any forbidden section is requested.
func (e *SafeExporter) Export(requestedSections []string) (*SafeExportManifest, error) {
	// Reject any forbidden section immediately.
	for _, req := range requestedSections {
		for _, forbidden := range forbiddenSections {
			if req == forbidden {
				return nil, fmt.Errorf("safe_export: section %q is forbidden and must not be exported", req)
			}
		}
	}

	// Only include sections that are on the allowlist.
	var included []string
	for _, req := range requestedSections {
		if isAllowed(req) {
			included = append(included, req)
		}
	}

	manifest := &SafeExportManifest{
		ExportedAt:       time.Now(),
		IncludedSections: included,
		ExcludedSections: forbiddenSections,
		RedactionSummary: "All screenshots, readable text patches, credentials, tokens, emails, API keys, form content, and payment information have been excluded.",
	}
	manifest.ManifestHash = computeManifestHash(manifest)

	if err := os.MkdirAll(e.exportDir, 0o700); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(e.exportDir, "safe_export_manifest.json")
	if err := os.WriteFile(manifestPath, data, 0o600); err != nil {
		return nil, err
	}
	return manifest, nil
}

func isAllowed(section string) bool {
	for _, a := range allowedSections {
		if section == a {
			return true
		}
	}
	return false
}

func computeManifestHash(m *SafeExportManifest) string {
	raw := fmt.Sprintf("%s|%v|%s", m.ExportedAt.UTC().Format(time.RFC3339), m.IncludedSections, m.RedactionSummary)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
