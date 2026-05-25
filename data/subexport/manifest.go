// subexport/manifest.go — install_manifest.json + README_INSTALL.md 產生（§31.4）。
// 定義匯出清單結構、序列化與 README 範本。
package subexport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// Manifest 資料結構
// ──────────────────────────────────────────────

// InstallManifest 匯出清單（install_manifest.json）。
type InstallManifest struct {
	FormatVersion     string            `json:"format_version"`
	ExportType        string            `json:"export_type"`
	ExportedAt        string            `json:"exported_at"`
	SourceSystemCode  string            `json:"source_system_code"`
	DisplayName       string            `json:"display_name"`
	Files             ManifestFiles     `json:"files"`
	InstallInstructions InstallInstructions `json:"install_instructions"`
}

// ManifestFiles 匯出檔案清單。
type ManifestFiles struct {
	Memory      string         `json:"memory"`
	DAG         string         `json:"dag"`
	ToolHistory string         `json:"tool_history"`
	Tools       []ManifestTool `json:"tools"`
}

// ManifestTool 匯出的工具記錄。
type ManifestTool struct {
	Type       string `json:"type"`        // skill, mcp, app
	Path       string `json:"path"`        // 匯出資料夾內相對路徑
	OriginalID string `json:"original_id"` // 系統內原始 ID
}

// InstallInstructions 安裝指引。
type InstallInstructions struct {
	MemoryTarget       string `json:"memory_target"`
	DAGTarget          string `json:"dag_target"`
	ToolHistoryTarget  string `json:"tool_history_target"`
	ToolConflictPolicy string `json:"tool_conflict_policy"`
}

// ──────────────────────────────────────────────
// Manifest 建立與讀取
// ──────────────────────────────────────────────

// NewInstallManifest 建立新的匯出清單。
func NewInstallManifest(displayName, sourceCode string, tools []ManifestTool) *InstallManifest {
	return &InstallManifest{
		FormatVersion:    "1.0",
		ExportType:       "sub_handler",
		ExportedAt:       time.Now().Format(time.RFC3339),
		SourceSystemCode: sourceCode,
		DisplayName:      displayName,
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

// SaveManifest 將 manifest 寫入指定目錄的 install_manifest.json。
func SaveManifest(dir string, m *InstallManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 manifest 失敗: %w", err)
	}
	path := filepath.Join(dir, "install_manifest.json")
	return os.WriteFile(path, data, 0o600)
}

// LoadManifest 從匯出資料夾讀取 install_manifest.json。
func LoadManifest(dir string) (*InstallManifest, error) {
	path := filepath.Join(dir, "install_manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("讀取 manifest 失敗: %w", err)
	}
	var m InstallManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("解析 manifest 失敗: %w", err)
	}
	return &m, nil
}

// ──────────────────────────────────────────────
// README 產生
// ──────────────────────────────────────────────

// GenerateReadme 產生人類可讀的安裝步驟 README_INSTALL.md。
func GenerateReadme(m *InstallManifest) string {
	var b strings.Builder

	b.WriteString("# Sub Handler 安裝說明\n\n")
	b.WriteString(fmt.Sprintf("- **Display Name**: %s\n", m.DisplayName))
	b.WriteString(fmt.Sprintf("- **Source System Code**: `%s`\n", m.SourceSystemCode))
	b.WriteString(fmt.Sprintf("- **Export Date**: %s\n", m.ExportedAt))
	b.WriteString(fmt.Sprintf("- **Format Version**: %s\n\n", m.FormatVersion))

	b.WriteString("## 檔案結構\n\n")
	b.WriteString("```\n")
	b.WriteString(fmt.Sprintf("%s/\n", m.DisplayName))
	b.WriteString("├── memory/              — 對話記憶（talk_full, summaries, deep_memory, index, manifest, memory_ops）\n")
	b.WriteString("├── dag/                 — DAG 狀態\n")
	b.WriteString("├── tool_history/        — 工具使用記錄\n")
	b.WriteString("├── tools/               — 連接的工具定義\n")
	for _, t := range m.Files.Tools {
		b.WriteString(fmt.Sprintf("│   └── %s\n", t.Path))
	}
	b.WriteString("├── install_manifest.json\n")
	b.WriteString("└── README_INSTALL.md\n")
	b.WriteString("```\n\n")

	b.WriteString("## 手動安裝步驟\n\n")
	b.WriteString("1. 在目標系統建立新的 sub 目錄: `subagents/callable/[NEW_CODE]/`\n")
	b.WriteString("2. 複製 `memory/` 到 `subagents/callable/[NEW_CODE]/memory/`\n")
	b.WriteString("3. 複製 `dag/` 到 `subagents/callable/[NEW_CODE]/dag/`\n")
	b.WriteString("4. 複製 `tool_history/` 到 `subagents/callable/[NEW_CODE]/tool_history/`\n")
	b.WriteString("5. 檢查工具衝突:\n\n")

	if len(m.Files.Tools) == 0 {
		b.WriteString("   （無連接工具）\n\n")
	} else {
		for _, t := range m.Files.Tools {
			b.WriteString(fmt.Sprintf("   - `%s` (type: %s, id: %s): 若系統已存在同名工具，選擇覆蓋或保留現有\n", t.Path, t.Type, t.OriginalID))
		}
		b.WriteString("\n")
	}

	b.WriteString("6. 在 `sub_registry_snapshot.json` 中新增 sub 記錄（使用新的 system code）\n")
	b.WriteString("7. 在 `tab_order.json` 的 `sub_order` 末尾新增 system code\n")

	return b.String()
}

// SaveReadme 將 README_INSTALL.md 寫入指定目錄。
func SaveReadme(dir string, m *InstallManifest) error {
	content := GenerateReadme(m)
	path := filepath.Join(dir, "README_INSTALL.md")
	return os.WriteFile(path, []byte(content), 0o600)
}
