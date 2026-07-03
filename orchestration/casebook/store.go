// store.go — 案例庫節點級落盤與分群查詢。
//
// 落盤：dag_runs/casebook.jsonl（append-only，沿用 storage.AtomicAppendLine）。
// jsonl 無索引，靠「超過大小重寫保留尾段」控制檔案大小（與 task_experience.go 一致）。
// run 級摘要不在此處理 — 沿用既有 dag_runs/index.json，不重複造輪子。
package casebook

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ui_console/data/storage"
)

// appendMu 序列化同一進程內的 rotate+append，避免「讀尾段→覆寫→append」交錯
// 造成的遺失寫入。跨進程仍依賴單一 app 實例假設（與既有 jsonl log 一致）。
var appendMu sync.Mutex

const (
	caseBookFile    = "casebook.jsonl"
	caseBookMaxKeep = 500     // 讀取 / 重寫時保留最近 N 筆
	caseBookMaxFile = 1 << 20 // 1MB；超過觸發尾段重寫
	caseBookScanBuf = 1 << 20 // 單行掃描上限（截斷已保證遠小於此，這是防禦縱深）
)

// caseBookPath 回傳案例庫檔案路徑（projectRoot/dag_runs/casebook.jsonl）。
func caseBookPath(projectRoot string) string {
	return filepath.Join(projectRoot, "dag_runs", caseBookFile)
}

// Append 落一筆案例。寫前正規化 tags、補 GroupID 預設值；失敗回 error 由 caller
// 決定是否吞（建議只記 trace 不擋主流程，與 recordTaskExperience 一致）。
func Append(projectRoot string, rec CaseRecord) error {
	dir := filepath.Join(projectRoot, "dag_runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	rec.Tags = normalizeTags(rec.Tags)
	if strings.TrimSpace(rec.GroupID) == "" {
		rec.GroupID = rec.RunID
	}
	rec = rec.clampFields() // 夾住大欄位，確保單行不爆 scanner 上限

	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	appendMu.Lock()
	defer appendMu.Unlock()
	// rotate 為盡力而為的 housekeeping：失敗只代表沒收斂檔案大小，
	// 用 AtomicWriteFile 保證失敗時舊檔完好（不會半寫入）；不擋 append。
	_ = rotateIfLarge(projectRoot, caseBookPath(projectRoot))
	return storage.AtomicAppendLine(caseBookPath(projectRoot), line)
}

// rotateIfLarge 檔案過大時，原子重寫只留最近 caseBookMaxKeep 筆。
// 呼叫端須已持有 appendMu。回傳 error 供呼叫端決定是否記錄（目前盡力而為）。
func rotateIfLarge(projectRoot, path string) error {
	info, err := os.Stat(path)
	if err != nil || info.Size() <= caseBookMaxFile {
		return nil
	}
	recent := LoadRecent(projectRoot, caseBookMaxKeep)
	if len(recent) == 0 {
		return nil
	}
	var b strings.Builder
	for _, r := range recent {
		if l, err := json.Marshal(r); err == nil {
			b.Write(l)
			b.WriteByte('\n')
		}
	}
	return storage.AtomicWriteFile(path, []byte(b.String()), 0o600)
}

// LoadRecent 讀最近 limit 筆（壞行跳過）。limit<=0 視為 caseBookMaxKeep。
func LoadRecent(projectRoot string, limit int) []CaseRecord {
	if limit <= 0 {
		limit = caseBookMaxKeep
	}
	f, err := os.Open(caseBookPath(projectRoot))
	if err != nil {
		return nil
	}
	defer f.Close()

	var all []CaseRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, caseBookScanBuf), caseBookScanBuf)
	for sc.Scan() {
		var rec CaseRecord
		if json.Unmarshal(sc.Bytes(), &rec) == nil && rec.RunID != "" {
			all = append(all, rec)
		}
	}
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all
}

// Group 是一個案例群（依核心 tag 聚合）。
type Group struct {
	Tag     string       `json:"tag"`
	Success bool         `json:"success"` // 此 tag 是否屬成功群（僅 TagSuccess 為真）
	Count   int          `json:"count"`
	Records []CaseRecord `json:"records"`
}

// GroupByTag 把最近紀錄依「核心 tag」聚合成案例群，
// 自然分出成功案例群（TagSuccess）與各類失敗案例群。
// limit<=0 取 caseBookMaxKeep。
func GroupByTag(projectRoot string, limit int) map[string]*Group {
	recs := LoadRecent(projectRoot, limit)
	groups := map[string]*Group{}
	for _, r := range recs {
		tag := r.CoreTag()
		g := groups[tag]
		if g == nil {
			g = &Group{Tag: tag, Success: tag == TagSuccess}
			groups[tag] = g
		}
		g.Count++
		g.Records = append(g.Records, r)
	}
	return groups
}

// SuccessAndFailure 便捷拆分：回 (成功案例群, 失敗案例群合併)。
// 成功 = Verdict==pass；其餘（fail/partial）歸失敗群，方便直接做「成功+失敗案例群組」。
func SuccessAndFailure(projectRoot string, limit int) (success, failure []CaseRecord) {
	for _, r := range LoadRecent(projectRoot, limit) {
		if r.IsSuccess() {
			success = append(success, r)
		} else {
			failure = append(failure, r)
		}
	}
	return success, failure
}
