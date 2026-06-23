package hookgene

import "path/filepath"

// 獨立資料夾內的固定檔名。
const (
	eventsFileName   = "recorder_events.jsonl"
	stateFileName    = "recorder_state.json"
	manifestFileName = "recorder_rotation_manifest.json"
)

// HookGeneDir 回傳本獨立系統的根資料夾：data/projects/[project]/hook_gene/。
// baseDir 為應用程式根目錄（與 data/storage.ProjectRoot 對齊，但本套件不 import 它以維持 leaf）。
func HookGeneDir(baseDir, projectID string) string {
	return filepath.Join(baseDir, "data", "projects", projectID, "hook_gene")
}
