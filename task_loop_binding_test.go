package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui_console/orchestration/dag"
)

// schema gate：未知欄位、缺必填、數量上限都要擋。
func TestValidateXlsxWriteArgs(t *testing.T) {
	if err := validateXlsxWriteArgs(json.RawMessage(`{"file_name":"a.xlsx","rows":[["x"]]}`)); err != nil {
		t.Fatalf("valid args rejected: %v", err)
	}
	if err := validateXlsxWriteArgs(json.RawMessage(`{"file_name":"a.xlsx","rows":[["x"]],"evil":"1"}`)); err == nil {
		t.Fatal("unknown field must be rejected")
	}
	if err := validateXlsxWriteArgs(json.RawMessage(`{"file_name":"a.xlsx"}`)); err == nil || !strings.Contains(err.Error(), "cells 或 rows") {
		t.Fatalf("missing rows/cells should report: %v", err)
	}
	if err := validateXlsxWriteArgs(json.RawMessage(`{"rows":[["x"]]}`)); err == nil || !strings.Contains(err.Error(), "file_name") {
		t.Fatalf("missing file_name should report: %v", err)
	}
}

// 淺層合併：新值覆蓋、舊值保留。
func TestMergePendingArgs(t *testing.T) {
	merged, err := mergePendingArgs(json.RawMessage(`{"file_name":"a.xlsx","sheet":"S1"}`), json.RawMessage(`{"sheet":"S2","rows":[["x"]]}`))
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	var m map[string]interface{}
	_ = json.Unmarshal(merged, &m)
	if m["file_name"] != "a.xlsx" || m["sheet"] != "S2" || m["rows"] == nil {
		t.Fatalf("merge wrong: %v", m)
	}
	if _, err := mergePendingArgs(nil, json.RawMessage(`not json`)); err == nil {
		t.Fatal("bad patch must fail")
	}
}

// deterministic compact：超限取 head+tail，且不切壞 UTF-8。
func TestCompactObservationText(t *testing.T) {
	short, truncated := compactObservationText("  hello  ")
	if short != "hello" || truncated {
		t.Fatalf("short text should pass through: %q %v", short, truncated)
	}
	long := strings.Repeat("中文資料", 400) // 4800 bytes > cap
	compact, truncated2 := compactObservationText(long)
	if !truncated2 || len(compact) > taskLoopPerObservationCap+64 {
		t.Fatalf("compact failed: len=%d truncated=%v", len(compact), truncated2)
	}
	if strings.Contains(compact, "�") {
		t.Fatalf("compact produced invalid utf8")
	}
}

// 簽名分級：read-only 二元組；write 類帶 hash（同 target 同簽名、異 target 異簽名）。
func TestLoopSignatureTiers(t *testing.T) {
	if loopSignature("搜尋", "設定檔") != "搜尋|設定檔" {
		t.Fatalf("read-only signature changed: %s", loopSignature("搜尋", "設定檔"))
	}
	w1 := loopSignature("寫入", "/a/b.txt")
	w2 := loopSignature("寫入", "/a/b.txt")
	w3 := loopSignature("寫入", "/a/c.txt")
	if w1 != w2 || w1 == w3 || !strings.HasPrefix(w1, "寫入|") {
		t.Fatalf("write signature wrong: %s %s %s", w1, w2, w3)
	}
}

// 防打轉：同簽名累計次數。
func TestLoopStateRecordSignature(t *testing.T) {
	state := &dag.LoopState{}
	if state.RecordSignature("搜尋|x") != 1 || state.RecordSignature("搜尋|x") != 2 {
		t.Fatal("signature count wrong")
	}
}

// PendingInput 建立條件：只攔 寫入/匯出 + Excel 線索，不攔一般檔案寫入。
func TestWantsXlsxWrite(t *testing.T) {
	for _, c := range []struct {
		action, target string
		want           bool
	}{
		{"寫入", "report.xlsx 三欄成本表", true},
		{"匯出", "幫我匯出 Excel", true},
		{"寫入", "做一張試算表", true},
		{"寫入", "notes.txt", false},
		{"搜尋", "report.xlsx", false},
	} {
		if got := wantsXlsxWrite(c.action, c.target); got != c.want {
			t.Fatalf("wantsXlsxWrite(%s,%s)=%v want %v", c.action, c.target, got, c.want)
		}
	}
}

// 端到端：建立 pending → 輸入ㄌ{含ㄌ的JSON}ㄌ待命 → gate → 真的產出 xlsx 檔。
func TestApplyLoopPendingInputEndToEnd(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)            // macOS: UserConfigDir 用 HOME
	t.Setenv("XDG_CONFIG_HOME", tmp) // Linux: 優先用 XDG
	app := &App{}
	state := &dag.LoopState{RunID: "r", NodeID: "n",
		PendingInput: &dag.PendingInput{Tool: "xlsx_write", SchemaID: "xlsx_write.v1"}}

	// 第一包缺 rows → 報缺欄位、pending 保留且記住已給的部分
	if _, err := app.applyLoopPendingInput(state, `輸入ㄌ{"file_name":"e2e.xlsx"}ㄌ待命`); err == nil || !strings.Contains(err.Error(), "cells 或 rows") {
		t.Fatalf("missing rows should report: %v", err)
	}
	if state.PendingInput == nil || !strings.Contains(string(state.PendingInput.PartialArgs), "e2e.xlsx") {
		t.Fatalf("partial args not kept: %+v", state.PendingInput)
	}

	// 第二包補 rows（值內含 ㄌ，驗證 first/last parser 端到端）→ 合併、過 gate、實際產檔
	obs, err := app.applyLoopPendingInput(state, `輸入ㄌ{"rows":[["甲ㄌ乙","丙"]]}ㄌ待命`)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if state.PendingInput != nil {
		t.Fatal("pending should clear after success")
	}
	if obs.Kind != "tool" || !strings.Contains(obs.CompactText, "e2e.xlsx") {
		t.Fatalf("observation wrong: %+v", obs)
	}
	matches, _ := filepath.Glob(filepath.Join(tmp, "**", "outputs", "e2e.xlsx"))
	if len(matches) == 0 {
		// 路徑層級依 storage.ProjectRoot 而異，退而求其次全樹找
		found := false
		_ = filepath.WalkDir(tmp, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && d.Name() == "e2e.xlsx" {
				found = true
			}
			return nil
		})
		if !found {
			t.Fatal("e2e.xlsx not generated")
		}
	}
}

// prompt 組裝：含目標、觀察與控制規則；user_input 顯示為使用者補充。
func TestBuildTaskLoopPrompt(t *testing.T) {
	state := &dag.LoopState{Observations: []dag.ObservationRecord{
		{Kind: "tool", Action: "搜尋", Target: "設定檔", SanitizedText: "找到 config.json"},
		{Kind: "user_input", SanitizedText: "用 out.xlsx"},
	}}
	prompt := buildTaskLoopPrompt("整理設定", state)
	for _, want := range []string{"整理設定", "找到 config.json", "使用者補充", "輸出ㄌ"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}
