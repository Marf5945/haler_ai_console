package dag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 存取 roundtrip：欄位完整、revision 遞增。
func TestLoopStateSaveLoadRoundtrip(t *testing.T) {
	root := t.TempDir()
	state := &LoopState{RunID: "run1", NodeID: "n1", Iteration: 3}
	state.RecordSignature("搜尋|設定檔")
	state.Observations = append(state.Observations, ObservationRecord{Kind: "tool", Action: "搜尋", SanitizedText: "found"})
	if err := SaveLoopStateLocked(root, state); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded := LoadLoopState(root, "run1", "n1")
	if loaded.Iteration != 3 || loaded.LoopRevision != 1 || len(loaded.Observations) != 1 {
		t.Fatalf("roundtrip mismatch: %+v", loaded)
	}
	if loaded.SeenSignatures["搜尋|設定檔"] != 1 {
		t.Fatalf("signatures lost")
	}
}

// recovery policy：run/node 不符或 JSON 壞掉 → 回全新狀態，不沿用舊內容。
func TestLoopStateLoadMismatchReturnsFresh(t *testing.T) {
	root := t.TempDir()
	state := &LoopState{RunID: "run1", NodeID: "n1", Iteration: 5}
	if err := SaveLoopStateLocked(root, state); err != nil {
		t.Fatalf("save: %v", err)
	}
	// 把 run1.n1 的檔案內容偽裝成別的 node → 視為不一致
	path := filepath.Join(root, "dag_runs", "run1.n1.loop.json")
	data, _ := os.ReadFile(path)
	_ = os.WriteFile(path, []byte(strings.Replace(string(data), `"node_id": "n1"`, `"node_id": "other"`, 1)), 0o600)
	loaded := LoadLoopState(root, "run1", "n1")
	if loaded.Iteration != 0 {
		t.Fatalf("mismatch should reset, got iteration=%d", loaded.Iteration)
	}
}

// 雙上限之一：超預算丟最舊內文，但保留 Hash 與摘要線索。
func TestLoopStateTrimToBudgetKeepsHash(t *testing.T) {
	state := &LoopState{RunID: "r", NodeID: "n"}
	big := strings.Repeat("甲", 200)
	for i := 0; i < 5; i++ {
		state.Observations = append(state.Observations, ObservationRecord{
			Kind: "tool", SanitizedText: big, Hash: "h",
		})
	}
	state.TrimToBudget(1200)
	if state.SanitizedBytes() > 1200+200 { // 修剪後保留 80 bytes 線索的餘量
		t.Fatalf("budget not enforced: %d", state.SanitizedBytes())
	}
	if !state.Observations[0].Truncated || state.Observations[0].Hash != "h" {
		t.Fatalf("oldest should be trimmed but keep hash: %+v", state.Observations[0])
	}
	// 修剪後字串必須仍是合法 UTF-8 開頭（不可切在 rune 中間）
	first := state.Observations[0].SanitizedText
	if strings.Contains(first, "�") {
		t.Fatalf("trim produced invalid utf8: %q", first)
	}
}

// run cleanup：loop sidecar 一併刪除，不留孤兒檔。
func TestDeleteLoopStatesForRun(t *testing.T) {
	root := t.TempDir()
	for _, node := range []string{"n1", "n2"} {
		_ = SaveLoopStateLocked(root, &LoopState{RunID: "run1", NodeID: node})
	}
	_ = SaveLoopStateLocked(root, &LoopState{RunID: "run2", NodeID: "n1"})
	DeleteLoopStatesForRun(root, "run1")
	entries, _ := os.ReadDir(filepath.Join(root, "dag_runs"))
	var left []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), suffixLoop) {
			left = append(left, e.Name())
		}
	}
	if len(left) != 1 || !strings.HasPrefix(left[0], "run2.") {
		t.Fatalf("expected only run2 sidecar left, got %v", left)
	}
}

// v3.1.8 摘要式壓縮：最舊一批合併成單筆 digest，新觀察保留全文。
func TestLoopStateCompressToBudget(t *testing.T) {
	state := &LoopState{RunID: "r", NodeID: "n"}
	for i := 0; i < 6; i++ {
		state.Observations = append(state.Observations, ObservationRecord{
			Kind: "tool", Action: "搜尋", Target: "目標", SanitizedText: strings.Repeat("甲", 300),
		})
	}
	state.Observations[5].SanitizedText = "最新觀察重點"
	state.CompressToBudget(1500)
	if state.SanitizedBytes() > 1500 {
		t.Fatalf("over budget: %d", state.SanitizedBytes())
	}
	if state.Observations[0].Kind != "digest" || !strings.Contains(state.Observations[0].SanitizedText, "搜尋") {
		t.Fatalf("oldest should merge into digest: %+v", state.Observations[0])
	}
	last := state.Observations[len(state.Observations)-1]
	if last.SanitizedText != "最新觀察重點" {
		t.Fatalf("newest must keep full text: %+v", last)
	}
	// 未超限 → 不動
	small := &LoopState{Observations: []ObservationRecord{{SanitizedText: "ok"}, {SanitizedText: "ok2"}, {SanitizedText: "ok3"}}}
	small.CompressToBudget(1024)
	if len(small.Observations) != 3 {
		t.Fatal("under budget must not compress")
	}
}
