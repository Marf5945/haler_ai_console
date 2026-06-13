// web_chain_test.go — Web Chain Phase 3 純邏輯 + 持久化測試。
package skill_eval

import (
	"path/filepath"
	"testing"
)

func TestWebChainSignature(t *testing.T) {
	if got := WebChainSignature([]string{"網路", " ", "網路"}); got != "網路>網路" {
		t.Fatalf("簽章錯誤，got %q", got)
	}
	if got := WebChainSignature(nil); got != "" {
		t.Fatalf("空序列應回空字串，got %q", got)
	}
}

func TestSummarizeAndCandidate(t *testing.T) {
	runs := []WebChainRun{
		{Signature: "網路>網路", DriftCount: 0},
		{Signature: "網路>網路", DriftCount: 0},
		{Signature: "網路>網路", DriftCount: 0},
		{Signature: "網路>網路>網路", DriftCount: 0},
		{Signature: "網路>網路>網路", DriftCount: 2}, // 有 drift → 整簽章不可升格
		{Signature: "網路>網路>網路", DriftCount: 0},
		{Signature: "網路>網路>網路", DriftCount: 0},
	}
	stats := SummarizeWebChainRuns(runs)
	if len(stats) != 2 {
		t.Fatalf("應彙整成 2 個簽章，got %d", len(stats))
	}
	// 排序穩定：較短簽章字典序在前。
	a := stats[0]
	if a.Signature != "網路>網路" || a.TotalRuns != 3 || a.CleanRuns != 3 || a.DriftRuns != 0 {
		t.Fatalf("網路>網路 統計錯誤：%+v", a)
	}
	b := stats[1]
	if b.TotalRuns != 4 || b.CleanRuns != 3 || b.DriftRuns != 1 || b.TotalDrifts != 2 {
		t.Fatalf("網路>網路>網路 統計錯誤：%+v", b)
	}

	cands := WebChainSkillCandidates(stats)
	if len(cands) != 1 || cands[0].Signature != "網路>網路" {
		t.Fatalf("只有零 drift 且 >=3 乾淨跑可升格，got %+v", cands)
	}
}

func TestIsSkillCandidateThreshold(t *testing.T) {
	// 乾淨但次數不足 → 不升格。
	if (WebChainSigStats{CleanRuns: 2, DriftRuns: 0}).IsSkillCandidate() {
		t.Fatal("2 次乾淨未達門檻不應升格")
	}
	// 達門檻但曾 drift → 不升格。
	if (WebChainSigStats{CleanRuns: 5, DriftRuns: 1}).IsSkillCandidate() {
		t.Fatal("曾 drift 不應升格")
	}
	if !(WebChainSigStats{CleanRuns: 3, DriftRuns: 0}).IsSkillCandidate() {
		t.Fatal("3 次乾淨且零 drift 應升格")
	}
}

func TestWebChainRunRoundTrip(t *testing.T) {
	store := NewStore(t.TempDir(), "default")
	for i := 0; i < 3; i++ {
		if err := store.AppendWebChainRun(WebChainRun{Signature: "網路>網路", DriftCount: 0}); err != nil {
			t.Fatalf("append 失敗：%v", err)
		}
	}
	runs, err := store.LoadWebChainRuns()
	if err != nil {
		t.Fatalf("load 失敗：%v", err)
	}
	if len(runs) != 3 || runs[0].Schema != SchemaWebChainRunV1 {
		t.Fatalf("round-trip 失敗：%+v", runs)
	}
	// 寫在獨立檔，不污染 events.jsonl。
	if _, err := store.LoadWebChainRuns(); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(store.webChainRunPath()) != "web_chain_runs.jsonl" {
		t.Fatalf("檔名錯誤：%s", store.webChainRunPath())
	}
}

func TestLoadWebChainRunsMissing(t *testing.T) {
	store := NewStore(t.TempDir(), "default")
	runs, err := store.LoadWebChainRuns()
	if err != nil || runs != nil {
		t.Fatalf("不存在應回 nil,nil；got %v,%v", runs, err)
	}
}
