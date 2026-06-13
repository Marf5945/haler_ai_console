// visual_learning/service.go — Visual Learning（§12）基礎結構。
// MVP 僅建立 struct + init，不綁定 Wails，待後續 TASK 擴充。
package visual_learning

import (
	"os"
	"path/filepath"
	"sync"
)

// ──────────────────────────────────────────────
// 資源類型定義
// ──────────────────────────────────────────────

// AssetType 視覺學習資源類型。
type AssetType string

const (
	AssetScreenshot AssetType = "screenshot"   // 螢幕截圖片段
	AssetDiagram    AssetType = "diagram"      // 流程圖 / 架構圖
	AssetAnnotation AssetType = "annotation"   // 標註疊加層
	AssetComparison AssetType = "before_after" // 前後對比
)

// ──────────────────────────────────────────────
// 核心結構
// ──────────────────────────────────────────────

// Asset 單一視覺學習資源。
type Asset struct {
	ID          string    `json:"id"`
	Type        AssetType `json:"type"`
	FilePath    string    `json:"file_path"`   // 相對於 visual_learning/ 的路徑
	Description string    `json:"description"` // 中文描述
	StepIndex   int       `json:"step_index"`  // 對應 DAG 步驟索引（-1 表示獨立）
	CreatedAt   string    `json:"created_at"`
}

// Service 管理視覺學習資源。
type Service struct {
	mu      sync.Mutex
	rootDir string            // visual_learning/ 根目錄
	assets  map[string]*Asset // ID → Asset
}

// ──────────────────────────────────────────────
// 初始化
// ──────────────────────────────────────────────

// NewService 建立 Visual Learning service。
// 確保目錄結構存在。
func NewService(hookRoot string) *Service {
	root := filepath.Join(hookRoot, "visual_learning")
	// 建立子目錄
	subdirs := []string{"screenshots", "diagrams", "annotations", "processed_temp"}
	for _, sub := range subdirs {
		os.MkdirAll(filepath.Join(root, sub), 0755)
	}

	return &Service{
		rootDir: root,
		assets:  make(map[string]*Asset),
	}
}
