package hookgene

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// queueSize 是訊號 channel 緩衝；滿了即丟棄（best-effort，§3.1.5.18.7）。
const queueSize = 1024

// RotateMaxBytes：events JSONL 超過此大小即 rotate（MVP 寫死 N MB，可調）。
const RotateMaxBytes = 8 * 1024 * 1024

// maxPendingAge / maxPendingEntries 防止「缺 completed」的 invocation 永久殘留在 pending
// （#2 防護：panic / 未收尾 / 訊號遺失的孤兒）。
const (
	maxPendingAge     = 30 * time.Minute
	maxPendingEntries = 4096
)

// Recorder 是平常 always-on 的行動基因記錄器（§3.1.5.18.3）。
//
// 併發模型（重要，維護時勿破壞）：
//   - 外部任意 goroutine 只透過 Emit 送訊號；Emit 非阻塞，queue 滿即丟棄並標 incomplete。
//   - 所有檔案寫入與 pending 聚合都在「單一 writer goroutine」(run) 內完成，無共享寫入競爭。
//   - state 另以 stateMu 保護，讓 Stats() 等讀取可安全跨 goroutine。
//   - sidecar 不直接寫檔：應把訊號送到 main app，由 main app 呼叫本 Recorder.Emit。
type Recorder struct {
	dir        string
	eventsPath string
	statePath  string

	queue chan Signal
	done  chan struct{}
	wg    sync.WaitGroup

	// 以下欄位僅由 writer goroutine（run）存取，無需鎖：
	pending    map[string]*pendingInvocation
	eventsFile *os.File

	// state 跨 goroutine 讀取 → 以 stateMu 保護。
	stateMu sync.Mutex
	state   *RecorderState

	// droppedSet：曾因 queue 滿而丟過事件的 invocation（標 incomplete 用），跨 goroutine。
	droppedSet sync.Map

	dropped        int64 // 丟棄總數（觀測用）
	evictedPending int64 // 被驅逐的未完成 invocation 數（觀測用）
}

// pendingInvocation 累積單一 invocation 在 completed 前的 hook。
type pendingInvocation struct {
	skillID   string
	hooks     []HookCode
	firstSeen time.Time // 用於 stale 驅逐：缺 completed 的孤兒不會永久殘留
}

// NewRecorder 建立 recorder 並載入既有 state + replay 補齊未套用事件（尚未啟動後台）。
func NewRecorder(dir string) (*Recorder, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	r := &Recorder{
		dir:        dir,
		eventsPath: filepath.Join(dir, eventsFileName),
		statePath:  filepath.Join(dir, stateFileName),
		queue:      make(chan Signal, queueSize),
		done:       make(chan struct{}),
		pending:    map[string]*pendingInvocation{},
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

// Start 啟動後台 writer goroutine（always-on）。
func (r *Recorder) Start() {
	r.wg.Add(1)
	go r.run()
}

// Stop 停止 recorder：排空 queue、關閉檔案。
func (r *Recorder) Stop() {
	close(r.done)
	r.wg.Wait()
	if r.eventsFile != nil {
		_ = r.eventsFile.Sync()
		_ = r.eventsFile.Close()
		r.eventsFile = nil
	}
}

// Emit 送一筆訊號。非阻塞：queue 滿即丟棄並標該 invocation incomplete，絕不卡住呼叫端
// （§3.1.5.18.7 / H-13）。
func (r *Recorder) Emit(s Signal) {
	select {
	case r.queue <- s:
	default:
		atomic.AddInt64(&r.dropped, 1)
		r.markDropped(s.InvocationID)
	}
}

// Dropped 回傳累積丟棄數（觀測用）。
func (r *Recorder) Dropped() int64 { return atomic.LoadInt64(&r.dropped) }

// Stats 回傳指定 skill 的統計快照（debug/review 用）。UI 只讀 state，不直接掃 events。
func (r *Recorder) Stats(skillID string) (SkillStats, bool) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	st, ok := r.state.Skills[skillID]
	if !ok {
		return SkillStats{}, false
	}
	cp := *st
	cp.Samples = append([]sample(nil), st.Samples...)
	return cp, true
}

// ── writer goroutine ────────────────────────────────────────────────

func (r *Recorder) run() {
	defer r.wg.Done()
	for {
		select {
		case s := <-r.queue:
			r.handle(s)
		case <-r.done:
			// 排空剩餘訊號後結束。
			for {
				select {
				case s := <-r.queue:
					r.handle(s)
				default:
					return
				}
			}
		}
	}
}

func (r *Recorder) handle(s Signal) {
	if s.Type == SignalCompleted {
		r.finalize(s)
		return
	}
	h, ok := HookFor(s)
	if !ok {
		return
	}
	p := r.pending[s.InvocationID]
	if p == nil {
		now := s.At
		if now.IsZero() {
			now = time.Now()
		}
		r.evictStalePending(now) // 防 stale：缺 completed 的孤兒不永久佔用 pending（#2）
		p = &pendingInvocation{skillID: s.SkillID, firstSeen: now}
		r.pending[s.InvocationID] = p
	}
	if p.skillID == "" {
		p.skillID = s.SkillID
	}
	p.hooks = append(p.hooks, h)
}

// finalize 在 skill_completed 時結 gene、寫事件、更新 state。
func (r *Recorder) finalize(s Signal) {
	p := r.pending[s.InvocationID]
	delete(r.pending, s.InvocationID)

	var raw []HookCode
	skillID := s.SkillID
	if p != nil {
		raw = p.hooks
		if skillID == "" {
			skillID = p.skillID
		}
	}
	gene := BuildGene(raw)

	// 完整樣本定義：未因 queue 滿丟過該 invocation 的事件（§3.1.5.18.4）。
	complete := !r.wasDropped(s.InvocationID)

	r.stateMu.Lock()
	prevHash := r.state.LastHash
	r.stateMu.Unlock()

	ev := RecorderEvent{
		EventID:      newEventID(),
		SkillID:      skillID,
		InvocationID: s.InvocationID,
		Gene:         gene.String(),
		HookCount:    len(raw),
		BCount:       gene.BCount,
		Oversized:    gene.Oversized,
		Complete:     complete,
		At:           s.At,
		PrevHash:     prevHash,
	}
	ev.Hash = ev.computeHash()

	if err := r.appendEvent(ev); err != nil {
		// best-effort：寫入失敗不影響主任務，丟棄該筆即可。
		r.clearDropped(s.InvocationID)
		return
	}
	r.applyToState(ev, gene)
	r.clearDropped(s.InvocationID)
	_ = r.saveState()
	r.maybeRotate()
}

// applyToState 把事件套進衍生 state（持有 stateMu）。
func (r *Recorder) applyToState(ev RecorderEvent, gene Gene) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	r.state.LastEventID = ev.EventID
	r.state.LastHash = ev.Hash
	r.state.UpdatedAt = ev.At

	st := r.state.Skills[ev.SkillID]
	if st == nil {
		st = &SkillStats{SkillID: ev.SkillID}
		r.state.Skills[ev.SkillID] = st
	}
	if !ev.Complete {
		st.IncompleteCount++ // 不完整 → 只計數，不入 80% 分母
		return
	}
	st.Samples = append(st.Samples, sample{At: ev.At, Bloated: gene.IsBloated()})
	st.prune(ev.At)
	st.recomputeComplexity()
}

// ── 持久化 ──────────────────────────────────────────────────────────

// appendEvent append 一筆 JSON line 並 fsync（§3.1.5.18.7）。
func (r *Recorder) appendEvent(ev RecorderEvent) error {
	if r.eventsFile == nil {
		f, err := os.OpenFile(r.eventsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return err
		}
		r.eventsFile = f
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	if _, err := r.eventsFile.Write(line); err != nil {
		return err
	}
	return r.eventsFile.Sync() // append 後立即落盤
}

// saveState 以 temp file + atomic rename 落地 state（§3.1.5.18.7）。
func (r *Recorder) saveState() error {
	r.stateMu.Lock()
	data, err := json.MarshalIndent(r.state, "", "  ")
	r.stateMu.Unlock()
	if err != nil {
		return err
	}
	tmp := r.statePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.statePath)
}

// load 載入 state（壞檔則重建），再 replay 補齊未套用事件。
func (r *Recorder) load() error {
	if data, err := os.ReadFile(r.statePath); err == nil {
		var st RecorderState
		if json.Unmarshal(data, &st) == nil && st.Skills != nil {
			r.state = &st
		}
	}
	if r.state == nil {
		r.state = &RecorderState{Skills: map[string]*SkillStats{}}
	}
	if r.state.Skills == nil {
		r.state.Skills = map[string]*SkillStats{}
	}
	return r.replayFromEvents()
}

// replayFromEvents 套用 state.last_event_id 之後的事件；崩潰殘行 / 壞行直接丟棄（§3.1.5.18.7）。
func (r *Recorder) replayFromEvents() error {
	f, err := os.Open(r.eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	seenLast := r.state.LastEventID == ""
	for sc.Scan() {
		var ev RecorderEvent
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue // 殘行 / 壞行：丟棄
		}
		if !seenLast {
			if ev.EventID == r.state.LastEventID {
				seenLast = true
			}
			continue
		}
		r.applyToState(ev, geneFromEvent(ev))
	}
	return sc.Err()
}

// ── rotation ────────────────────────────────────────────────────────

type rotationEntry struct {
	File      string    `json:"file"`
	TailHash  string    `json:"tail_hash"`
	RotatedAt time.Time `json:"rotated_at"`
}

// maybeRotate 在 events 檔超過 RotateMaxBytes 時 rotate，並把 tail hash 記入 manifest。
// state JSON 已保有 14 天窗統計，rotated raw 僅作審計/封存，可由保留策略另行 prune。
func (r *Recorder) maybeRotate() {
	if r.eventsFile == nil {
		return
	}
	info, err := r.eventsFile.Stat()
	if err != nil || info.Size() < RotateMaxBytes {
		return
	}
	_ = r.eventsFile.Sync()
	_ = r.eventsFile.Close()
	r.eventsFile = nil

	ts := time.Now().UTC().Format("20060102T150405")
	rotated := filepath.Join(r.dir, "recorder_events."+ts+".jsonl")
	if err := os.Rename(r.eventsPath, rotated); err != nil {
		return
	}
	r.stateMu.Lock()
	tail := r.state.LastHash
	r.stateMu.Unlock()
	r.appendManifest(filepath.Base(rotated), tail)
	// 新檔第一筆 PrevHash 會接上 tail（見 finalize），hash chain 不中斷。
}

func (r *Recorder) appendManifest(file, tailHash string) {
	path := filepath.Join(r.dir, manifestFileName)
	var entries []rotationEntry
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &entries)
	}
	entries = append(entries, rotationEntry{File: file, TailHash: tailHash, RotatedAt: time.Now().UTC()})
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o600) == nil {
		_ = os.Rename(tmp, path)
	}
}

// ── dropped invocation 追蹤 ─────────────────────────────────────────

func (r *Recorder) markDropped(id string) {
	if id != "" {
		r.droppedSet.Store(id, struct{}{})
	}
}
func (r *Recorder) wasDropped(id string) bool { _, ok := r.droppedSet.Load(id); return ok }
func (r *Recorder) clearDropped(id string)    { r.droppedSet.Delete(id) }

// ── stale pending 驅逐（#2 防護）────────────────────────────────────

// evictStalePending 移除超過 maxPendingAge 或超出 maxPendingEntries 的未完成 invocation。
// 這些是缺 completed 的孤兒（panic / 未收尾 / 訊號遺失），丟棄不影響已完成樣本統計。
// 僅由 writer goroutine 呼叫（單執行緒，免鎖）。
func (r *Recorder) evictStalePending(now time.Time) {
	for id, p := range r.pending {
		if now.Sub(p.firstSeen) > maxPendingAge {
			delete(r.pending, id)
			atomic.AddInt64(&r.evictedPending, 1)
		}
	}
	for len(r.pending) > maxPendingEntries {
		var oldestID string
		var oldest time.Time
		firstIter := true
		for id, p := range r.pending {
			if firstIter || p.firstSeen.Before(oldest) {
				oldestID, oldest, firstIter = id, p.firstSeen, false
			}
		}
		delete(r.pending, oldestID)
		atomic.AddInt64(&r.evictedPending, 1)
	}
}

// EvictedPending 回傳被驅逐的未完成 invocation 數（觀測用）。
func (r *Recorder) EvictedPending() int64 { return atomic.LoadInt64(&r.evictedPending) }

// ── event id ────────────────────────────────────────────────────────

var eventSeq int64

// newEventID 產生唯一 event id（事件順序以 JSONL 行序為權威，id 僅需唯一）。
func newEventID() string {
	n := atomic.AddInt64(&eventSeq, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), n)
}
