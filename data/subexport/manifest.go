package subexport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type InstallManifest struct {
	FormatVersion       string              `json:"format_version"`
	ExportType          string              `json:"export_type"`
	ExportedAt          string              `json:"exported_at"`
	SourceSystemCode    string              `json:"source_system_code"`
	DisplayName         string              `json:"display_name"`
	Portable            PortablePolicy      `json:"portable"`
	DependencySelection DependencySelection `json:"dependency_selection"`
	Files               ManifestFiles       `json:"files"`
	InstallInstructions InstallInstructions `json:"install_instructions"`
}

type ManifestFiles struct {
	Memory      string         `json:"memory"`
	DAG         string         `json:"dag"`
	ToolHistory string         `json:"tool_history"`
	Tools       []ManifestTool `json:"tools"`
}

type ManifestTool struct {
	Type       string `json:"type"`
	Path       string `json:"path"`
	OriginalID string `json:"original_id"`
}

type PortablePolicy struct {
	CrossPlatform        bool     `json:"cross_platform"`
	IncludesConversation bool     `json:"includes_conversation"`
	IncludesWorkflowDeps bool     `json:"includes_workflow_deps"`
	Excludes             []string `json:"excludes"`
}

type DependencySelection struct {
	Mode  string   `json:"mode"`
	Notes []string `json:"notes"`
}

type InstallInstructions struct {
	MemoryTarget       string `json:"memory_target"`
	DAGTarget          string `json:"dag_target"`
	ToolHistoryTarget  string `json:"tool_history_target"`
	ToolConflictPolicy string `json:"tool_conflict_policy"`
}

func NewInstallManifest(displayName, sourceCode string, tools []ManifestTool) *InstallManifest {
	return &InstallManifest{
		FormatVersion:    "1.0",
		ExportType:       "sub_handler",
		ExportedAt:       time.Now().Format(time.RFC3339),
		SourceSystemCode: sourceCode,
		DisplayName:      displayName,
		Portable: PortablePolicy{
			CrossPlatform:        true,
			IncludesConversation: true,
			IncludesWorkflowDeps: len(tools) > 0,
			Excludes: []string{
				"api keys",
				"credentials",
				"secret tokens",
				"uploaded file payloads",
				"local-only caches",
			},
		},
		DependencySelection: DependencySelection{
			Mode: "auto_detected_from_sub_history",
			Notes: []string{
				"The package is intended to move between macOS and Windows.",
				"Workflow dependencies are copied when referenced by this sub package or explicitly supplied by the UI.",
				"Credentials, API keys, uploaded file payloads, and local-only caches are skipped during export.",
			},
		},
		Files: ManifestFiles{
			Memory:      "memory/",
			DAG:         "dag/",
			ToolHistory: "tool_history/",
			Tools:       tools,
		},
		InstallInstructions: InstallInstructions{
			MemoryTarget:       "subagents/callable/[NEW_CODE]/memory/",
			DAGTarget:          "subagents/callable/[NEW_CODE]/dag/",
			ToolHistoryTarget:  "subagents/callable/[NEW_CODE]/tool_history/",
			ToolConflictPolicy: "ask_user",
		},
	}
}

func SaveManifest(dir string, m *InstallManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	path := filepath.Join(dir, "install_manifest.json")
	return os.WriteFile(path, data, 0o600)
}

func LoadManifest(dir string) (*InstallManifest, error) {
	path := filepath.Join(dir, "install_manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m InstallManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

func GenerateReadme(m *InstallManifest) string {
	var b strings.Builder

	b.WriteString("# Sub Handler Install Guide\n\n")
	b.WriteString(fmt.Sprintf("- **Display Name**: %s\n", m.DisplayName))
	b.WriteString(fmt.Sprintf("- **Source System Code**: `%s`\n", m.SourceSystemCode))
	b.WriteString(fmt.Sprintf("- **Export Date**: %s\n", m.ExportedAt))
	b.WriteString(fmt.Sprintf("- **Format Version**: %s\n\n", m.FormatVersion))

	b.WriteString("## Portable Policy\n\n")
	b.WriteString("- Cross-platform package: yes\n")
	b.WriteString("- Includes conversation memory/history needed to rebuild the sub workflow.\n")
	b.WriteString("- Includes referenced workflow dependencies when detected or explicitly selected.\n")
	b.WriteString("- Excludes API keys, credentials, tokens, uploaded file payloads, and local-only caches.\n\n")

	b.WriteString("## Package Layout\n\n")
	b.WriteString("```\n")
	b.WriteString(fmt.Sprintf("%s/\n", m.DisplayName))
	b.WriteString("  memory/              conversation memory and summaries\n")
	b.WriteString("  dag/                 workflow DAG state\n")
	b.WriteString("  tool_history/        tool-use history\n")
	b.WriteString("  tools/               referenced skills, MCPs, and apps\n")
	for _, t := range m.Files.Tools {
		b.WriteString(fmt.Sprintf("    %s\n", t.Path))
	}
	b.WriteString("  install_manifest.json\n")
	b.WriteString("  README_INSTALL.md\n")
	b.WriteString("```\n\n")

	b.WriteString("## 手動安裝步驟\n\n")
	b.WriteString("1. Create a new sub folder: `subagents/callable/[NEW_CODE]/`.\n")
	b.WriteString("2. Copy `memory/` to `subagents/callable/[NEW_CODE]/memory/`.\n")
	b.WriteString("3. Copy `dag/` to `subagents/callable/[NEW_CODE]/dag/`.\n")
	b.WriteString("4. Copy `tool_history/` to `subagents/callable/[NEW_CODE]/tool_history/`.\n")
	b.WriteString("5. Install referenced workflow dependencies:\n\n")

	if len(m.Files.Tools) == 0 {
		b.WriteString("   - No referenced workflow dependency was packaged.\n\n")
	} else {
		for _, t := range m.Files.Tools {
			b.WriteString(fmt.Sprintf("   - `%s` (type: %s, id: %s)\n", t.Path, t.Type, t.OriginalID))
		}
		b.WriteString("\n")
	}

	b.WriteString("6. Update `sub_registry_snapshot.json` with the new system code.\n")
	b.WriteString("7. Update `tab_order.json` or `sub_order` with the new system code.\n")

	return b.String()
}

func SaveReadme(dir string, m *InstallManifest) error {
	content := GenerateReadme(m)
	path := filepath.Join(dir, "README_INSTALL.md")
	return os.WriteFile(path, []byte(content), 0o600)
}
