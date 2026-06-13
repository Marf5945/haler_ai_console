package hookgene

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// emitBloatedInvocation 送一個「肥大」完整 invocation（>16 個 ㄅ → oversized）。
func emitBloatedInvocation(r *Recorder, skillID, invID string, at time.Time) {
	r.Emit(Signal{SkillID: skillID, InvocationID: invID, Type: SignalDataEntered, At: at})
	for i := 0; i < 20; i++ { // 20 個內部處理 → oversized
		r.Emit(Signal{SkillID: skillID, InvocationID: invID, Type: SignalDataProcessed, At: at})
	}
	r.Emit(Signal{SkillID: skillID, InvocationID: invID, Type: SignalDataLeft, CrossedBoundary: true, At: at})
	r.Emit(Signal{SkillID: skillID, InvocationID: invID, Type: SignalCompleted, At: at})
}

func TestRecorderFormsGeneAndPromptsReview(t *testing.T) {
	dir := t.TempDir()
	r, err := NewRecorder(dir)
	if err != nil {
		t.Fatal(err)
	}
	r.Start()
	now := time.Now()
	for i := 0; i < 7; i++ { // 7 次完整肥大樣本 → 達門檻
		emitBloatedInvocation(r, "skill-A", "inv-"+itoa(i), now)
	}
	r.Stop()

	st, ok := r.Stats("skill-A")
	if !ok {
		t.Fatal("expected stats for skill-A")
	}
	if st.CompleteCount() != 7 {
		t.Fatalf("complete samples = %d, want 7", st.CompleteCount())
	}
	if st.BloatRatio() != 1.0 {
		t.Fatalf("bloat ratio = %v, want 1.0", st.BloatRatio())
	}
	if !st.ShouldPromptReview() {
		t.Fatal("should prompt review at 7 samples / 100% bloat")
	}

	// events 檔存在，且 hash chain 串接正確。
	assertHashChain(t, filepath.Join(dir, eventsFileName))
}

func TestRecorderReplayRebuildsState(t *testing.T) {
	dir := t.TempDir()
	r, _ := NewRecorder(dir)
	r.Start()
	now := time.Now()
	for i := 0; i < 7; i++ {
		emitBloatedInvocation(r, "skill-B", "b-"+itoa(i), now)
	}
	r.Stop()
	before, _ := r.Stats("skill-B")

	// 刪除 state.json，模擬壞檔/遺失 → 由 events replay 重建。
	if err := os.Remove(filepath.Join(dir, stateFileName)); err != nil {
		t.Fatal(err)
	}
	r2, err := NewRecorder(dir)
	if err != nil {
		t.Fatal(err)
	}
	after, ok := r2.Stats("skill-B")
	if !ok {
		t.Fatal("replay should rebuild skill-B")
	}
	if after.CompleteCount() != before.CompleteCount() || after.BloatRatio() != before.BloatRatio() {
		t.Fatalf("replay mismatch: before(%d,%v) after(%d,%v)",
			before.CompleteCount(), before.BloatRatio(), after.CompleteCount(), after.BloatRatio())
	}
}

func TestRecorderDiscardsTornLastLine(t *testing.T) {
	dir := t.TempDir()
	r, _ := NewRecorder(dir)
	r.Start()
	now := time.Now()
	emitBloatedInvocation(r, "skill-C", "c-0", now)
	r.Stop()

	// 在 events 檔尾端附加一段崩潰殘行（半行 JSON）。
	f, _ := os.OpenFile(filepath.Join(dir, eventsFileName), os.O_WRONLY|os.O_APPEND, 0o600)
	_, _ = f.WriteString(`{"event_id":"torn","skill_id":"skill-C"`) // 無換行、無收尾
	_ = f.Close()
	_ = os.Remove(filepath.Join(dir, stateFileName))

	r2, err := NewRecorder(dir) // replay 應丟棄殘行而不報錯
	if err != nil {
		t.Fatalf("replay should not fail on torn line: %v", err)
	}
	st, ok := r2.Stats("skill-C")
	if !ok || st.CompleteCount() != 1 {
		t.Fatalf("expected 1 complete sample after discarding torn line, got %v / %d", ok, st.CompleteCount())
	}
}

// ── helpers ─────────────────────────────────────────────────────────

func assertHashChain(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("events file missing: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	prev := ""
	count := 0
	for sc.Scan() {
		var ev RecorderEvent
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			t.Fatalf("bad event line: %v", err)
		}
		if ev.PrevHash != prev {
			t.Fatalf("hash chain break at line %d: prev_hash=%s want %s", count, ev.PrevHash, prev)
		}
		if ev.Hash != ev.computeHash() {
			t.Fatalf("hash mismatch at line %d", count)
		}
		prev = ev.Hash
		count++
	}
	if count == 0 {
		t.Fatal("no events written")
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func TestEvictStalePending(t *testing.T) {
	// 直接驗證孤兒驅逐邏輯（缺 completed 的 invocation 不永久殘留）。
	r := &Recorder{pending: map[string]*pendingInvocation{}}
	now := time.Now()
	r.pending["old"] = &pendingInvocation{firstSeen: now.Add(-time.Hour)}     // > 30m → 驅逐
	r.pending["fresh"] = &pendingInvocation{firstSeen: now.Add(-time.Minute)} // 新 → 保留
	r.evictStalePending(now)
	if _, ok := r.pending["old"]; ok {
		t.Fatal("stale pending should be evicted")
	}
	if _, ok := r.pending["fresh"]; !ok {
		t.Fatal("fresh pending should remain")
	}
	if r.EvictedPending() != 1 {
		t.Fatalf("evicted count = %d, want 1", r.EvictedPending())
	}
}
