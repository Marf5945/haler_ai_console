package main

// app_blackboard.go — 黑板（多人會議）Wails binding。
//
// 依 BLACKBOARD_SPEC.md：核心引擎在 shared/blackboard，載體透過 Store 介面
// 抽換；本 binding 以 FileStore 為 MVP 載體（把交換檔放進任何同步資料夾
// 即為雲端黑板）。比照 popout hub 模式使用套件級單例，避免修改 app.go。
//
// 職責：
//   - 會議 session（開啟/關閉/同步/append，envelope 防偽）
//   - coordinator 迴圈（lease 搶租/續租、主檔寫出、自動歸檔，spec §9/§1/§13）
//   - 信任層設定（trusted_keys + HMAC 簽章，spec §6.2）
//   - 真刪除審批（delete_text_request → 主持人批准 → redaction，spec §8）
//   - request 通知去重的 cursor 持久化（spec §11）

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ui_console/shared/blackboard"
	"ui_console/shared/eventbus"
)

// blackboardHub holds the single active blackboard session.
type blackboardHub struct {
	mu         sync.Mutex
	store      *blackboard.FileStore
	syncer     *blackboard.Synchronizer
	watcher    *blackboard.PollingWatcher
	cancel     context.CancelFunc
	docPath    string
	actor      blackboard.Actor
	lastState  *blackboard.MeetingState
	missingIDs []string        // latest integrity result surfaced by BlackboardStatus
	notified   map[string]bool // request event ids already emitted (spec §7)

	// cursor persistence (spec §11)
	cursor     blackboard.Cursor
	cursorPath string

	// trust layer (spec §6.2)
	signKeyID  string
	signSecret []byte

	// coordinator loop (spec §9)
	coordCancel context.CancelFunc
	instance    blackboard.Actor // app-type actor for coordinator_lease events
	masterPath  string
}

var bbHub = &blackboardHub{notified: map[string]bool{}}

// BlackboardStatusDTO is the summary surfaced to the frontend.
type BlackboardStatusDTO struct {
	Open               bool     `json:"open"`
	DocPath            string   `json:"doc_path"`
	ActorType          string   `json:"actor_type"`
	ActorID            string   `json:"actor_id"`
	ProjectedUntil     string   `json:"projected_until"`
	EventCount         int      `json:"event_count"`
	AgendaCount        int      `json:"agenda_count"`
	DecisionCount      int      `json:"decision_count"`
	ActionItemCount    int      `json:"action_item_count"`
	MaterialCount      int      `json:"material_count"`
	VoteCount          int      `json:"vote_count"`
	DeadLetterCount    int      `json:"dead_letter_count"`
	MissingIDs         []string `json:"missing_ids"`
	TrustEnabled       bool     `json:"trust_enabled"`
	CoordinatorEnabled bool     `json:"coordinator_enabled"`
	MasterPath         string   `json:"master_path,omitempty"`
	LeaseHolder        string   `json:"lease_holder,omitempty"`
	LeaseValid         bool     `json:"lease_valid"`
	IAmCoordinator     bool     `json:"i_am_coordinator"`
}

func (h *blackboardHub) statusLocked() BlackboardStatusDTO {
	dto := BlackboardStatusDTO{
		Open: h.store != nil, DocPath: h.docPath,
		ActorType: h.actor.Type, ActorID: h.actor.ID,
		TrustEnabled:       h.syncer != nil && h.syncer.Trusted != nil,
		CoordinatorEnabled: h.coordCancel != nil,
		MasterPath:         h.masterPath,
	}
	if h.lastState != nil {
		st := h.lastState
		dto.ProjectedUntil = st.ProjectedUntil
		dto.EventCount = st.EventCount
		dto.AgendaCount = len(st.Agenda)
		dto.DecisionCount = len(st.Decisions)
		dto.ActionItemCount = len(st.ActionItems)
		dto.MaterialCount = len(st.SharedMaterials)
		dto.VoteCount = len(st.Votes)
		dto.DeadLetterCount = st.DeadLetterCount
		// Copy the slice so a caller cannot mutate the hub's last sync result.
		// Use [] instead of null on the Wails JSON boundary for a stable API.
		dto.MissingIDs = append([]string{}, h.missingIDs...)
		if st.Lease != nil {
			dto.LeaseHolder = st.Lease.Holder
			dto.LeaseValid = blackboard.LeaseValidAt(st.Lease, time.Now())
			dto.IAmCoordinator = dto.LeaseValid && st.Lease.Holder == h.instance.ID
		}
	}
	return dto
}

// blackboardMirrorDir keeps each exchange doc's mirror/seq/cursor separate.
func blackboardMirrorDir(docPath string) string {
	sum := sha256.Sum256([]byte(docPath))
	return filepath.Join(appDataRoot(), "blackboard", hex.EncodeToString(sum[:8]))
}

// ---------------------------------------------------------------------------
// Session lifecycle
// ---------------------------------------------------------------------------

// BlackboardOpen opens (or creates) an exchange document and starts the
// polling watcher. actorType 只接受 human/agent/coordinator/app。
func (a *App) BlackboardOpen(docPath, actorType, actorID string, pollMs int) (BlackboardStatusDTO, error) {
	docPath = strings.TrimSpace(expandUserPath(docPath))
	if docPath == "" {
		return BlackboardStatusDTO{}, fmt.Errorf("docPath is required")
	}
	switch actorType {
	case blackboard.ActorHuman, blackboard.ActorAgent, blackboard.ActorCoordinator, blackboard.ActorApp:
	default:
		return BlackboardStatusDTO{}, fmt.Errorf("invalid actor type %q", actorType)
	}
	if strings.TrimSpace(actorID) == "" {
		return BlackboardStatusDTO{}, fmt.Errorf("actorID is required")
	}

	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.store != nil {
		return bbHub.statusLocked(), fmt.Errorf("blackboard already open at %s; close it first", bbHub.docPath)
	}

	mirrorDir := blackboardMirrorDir(docPath)
	store := blackboard.NewFileStore(docPath)
	syncer := &blackboard.Synchronizer{
		Store:  store,
		Mirror: blackboard.NewMirror(mirrorDir),
	}
	bbHub.store = store
	bbHub.syncer = syncer
	bbHub.docPath = docPath
	bbHub.actor = blackboard.Actor{Type: actorType, ID: actorID}

	// Restore the request-dedup cursor (spec §11): a restart must not
	// re-trigger requests that were already surfaced.
	bbHub.cursorPath = filepath.Join(mirrorDir, "cursor.json")
	cursor, err := blackboard.LoadCursor(bbHub.cursorPath)
	if err != nil {
		cursor = blackboard.Cursor{}
	}
	cursor.DocID = docPath
	bbHub.cursor = cursor
	bbHub.notified = map[string]bool{}
	for _, id := range cursor.ProcessedIDs {
		bbHub.notified[id] = true
	}

	// Initial sync before the watcher takes over.
	if _, err := a.blackboardSyncLocked(); err != nil {
		bbHub.resetLocked()
		return BlackboardStatusDTO{}, fmt.Errorf("initial sync: %w", err)
	}

	interval := time.Duration(pollMs) * time.Millisecond
	if pollMs <= 0 {
		interval = 2 * time.Second
	}
	watcher := blackboard.NewPollingWatcher(store, interval)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := watcher.Start(ctx)
	if err != nil {
		cancel()
		bbHub.resetLocked()
		return BlackboardStatusDTO{}, fmt.Errorf("start watcher: %w", err)
	}
	bbHub.watcher = watcher
	bbHub.cancel = cancel

	go func() {
		for range ch {
			bbHub.mu.Lock()
			if bbHub.syncer == nil {
				bbHub.mu.Unlock()
				return
			}
			_, syncErr := a.blackboardSyncLocked()
			bbHub.mu.Unlock()
			_ = syncErr // sync errors are transient; next signal retries
		}
	}()

	return bbHub.statusLocked(), nil
}

func (h *blackboardHub) resetLocked() {
	if h.coordCancel != nil {
		h.coordCancel()
		h.coordCancel = nil
	}
	if h.cancel != nil {
		h.cancel()
	}
	if h.watcher != nil {
		h.watcher.Stop()
	}
	if h.cursorPath != "" {
		_ = blackboard.SaveCursor(h.cursorPath, h.cursor)
	}
	h.store, h.syncer, h.watcher, h.cancel = nil, nil, nil, nil
	h.docPath = ""
	h.actor = blackboard.Actor{}
	h.lastState = nil
	h.missingIDs = nil
	h.cursor = blackboard.Cursor{}
	h.cursorPath = ""
	h.signKeyID, h.signSecret = "", nil
	h.instance = blackboard.Actor{}
	h.masterPath = ""
}

// blackboardSyncLocked runs one sync round and emits frontend events.
// Caller must hold bbHub.mu.
func (a *App) blackboardSyncLocked() (*blackboard.SyncResult, error) {
	res, err := bbHub.syncer.Sync(context.Background())
	if err != nil {
		return nil, err
	}
	bbHub.lastState = &res.State
	bbHub.missingIDs = append(bbHub.missingIDs[:0], res.MissingIDs...)
	cursorDirty := false
	if bbHub.cursor.LastProjectionID != res.State.ProjectedUntil {
		bbHub.cursor.LastProjectionID = res.State.ProjectedUntil
		cursorDirty = true
	}

	if a.eventBus != nil {
		a.eventBus.Emit(eventbus.EventBlackboardStateUpdated, bbHub.statusLocked())
		if len(res.MissingIDs) > 0 {
			a.eventBus.Emit(eventbus.EventBlackboardIntegrityWarning, map[string]interface{}{
				"missing_ids": res.MissingIDs,
			})
		}
		if n := len(res.Parse.DeadLetters); n > 0 {
			a.eventBus.Emit(eventbus.EventBlackboardDeadLetter, map[string]interface{}{
				"count": n,
			})
		}
	}
	// 鐵律（spec §7）：只有明確點名本機 actor 的 request 才可能觸發行動。
	// 已通知的 request 記入 cursor，重啟不重複觸發（spec §11）。
	// Only an agent is an executable request target. Human/coordinator/app
	// sessions may read requests as data but must never receive action events.
	isLocalAgent := bbHub.actor.Type == blackboard.ActorAgent
	me := bbHub.actor.String()
	for _, req := range res.State.Requests {
		if !isLocalAgent || req.To != me || bbHub.notified[req.ID] {
			continue
		}
		bbHub.notified[req.ID] = true
		bbHub.cursor.MarkProcessed(req.ID)
		cursorDirty = true
		if a.eventBus != nil {
			a.eventBus.Emit(eventbus.EventBlackboardRequest, req)
		}
	}
	if cursorDirty && bbHub.cursorPath != "" {
		_ = blackboard.SaveCursor(bbHub.cursorPath, bbHub.cursor)
	}
	return res, nil
}

// BlackboardClose stops the watcher and coordinator and releases the session.
func (a *App) BlackboardClose() BlackboardStatusDTO {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	bbHub.resetLocked()
	return bbHub.statusLocked()
}

// BlackboardStatus returns the current summary without forcing a sync.
func (a *App) BlackboardStatus() BlackboardStatusDTO {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	return bbHub.statusLocked()
}

// BlackboardSync forces a synchronization round and returns the summary.
func (a *App) BlackboardSync() (BlackboardStatusDTO, error) {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.syncer == nil {
		return bbHub.statusLocked(), fmt.Errorf("blackboard not open")
	}
	if _, err := a.blackboardSyncLocked(); err != nil {
		return bbHub.statusLocked(), err
	}
	return bbHub.statusLocked(), nil
}

// BlackboardState returns the full projected Meeting State.
func (a *App) BlackboardState() (blackboard.MeetingState, error) {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.lastState == nil {
		return blackboard.MeetingState{}, fmt.Errorf("blackboard not open")
	}
	return *bbHub.lastState, nil
}

// BlackboardRenderState returns the Meeting State Markdown block
// (deterministic given the same log; timestamp injected here at the edge).
func (a *App) BlackboardRenderState() (string, error) {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.lastState == nil {
		return "", fmt.Errorf("blackboard not open")
	}
	return blackboard.RenderMeetingState(
		*bbHub.lastState, bbHub.actor.String(), time.Now().UTC().Format(time.RFC3339)), nil
}

// ---------------------------------------------------------------------------
// Appending events
// ---------------------------------------------------------------------------

// BlackboardAppendMessage appends a kind:"message" event as the session actor.
func (a *App) BlackboardAppendMessage(body string) (string, error) {
	return a.blackboardAppend(blackboard.KindMessage, func(e *blackboard.Event) {
		e.Body = body
	})
}

// BlackboardAppend appends an arbitrary event. kind 必須是 spec 已定義的
// kind；payloadJSON 是該 kind 的欄位（envelope 欄位由本機補上並覆蓋，
// 呼叫端無法偽造 id/actor/created_at）。
func (a *App) BlackboardAppend(kind string, payloadJSON string) (string, error) {
	if !blackboard.KnownKind(kind) {
		return "", fmt.Errorf("unknown kind %q", kind)
	}
	var payload blackboard.Event
	if strings.TrimSpace(payloadJSON) != "" {
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return "", fmt.Errorf("payload: %w", err)
		}
	}
	return a.blackboardAppend(kind, func(e *blackboard.Event) {
		envelope := *e // keep locally-stamped envelope fields
		*e = payload
		e.V = envelope.V
		e.ID = envelope.ID
		e.Kind = envelope.Kind
		e.Actor = envelope.Actor
		e.CreatedAt = envelope.CreatedAt
		e.Sig = nil // signatures are attached below by the trust layer
	})
}

// blackboardAppend stamps the envelope, signs when the trust layer is
// configured, appends with readback verification, then syncs.
func (a *App) blackboardAppend(kind string, mut func(*blackboard.Event)) (string, error) {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.store == nil {
		return "", fmt.Errorf("blackboard not open")
	}
	ev, err := bbHub.stampLocked(kind, bbHub.actor)
	if err != nil {
		return "", err
	}
	if mut != nil {
		mut(&ev)
	}
	return a.appendStampedLocked(ev)
}

// stampLocked builds a fresh envelope for the given actor.
func (h *blackboardHub) stampLocked(kind string, actor blackboard.Actor) (blackboard.Event, error) {
	id, err := blackboard.NewEventID(time.Now())
	if err != nil {
		return blackboard.Event{}, err
	}
	return blackboard.Event{
		V: blackboard.SchemaVersion, ID: id, Kind: kind,
		Actor:     actor,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// appendStampedLocked signs (if configured), appends, and re-syncs.
func (a *App) appendStampedLocked(ev blackboard.Event) (string, error) {
	if len(bbHub.signSecret) > 0 {
		if err := blackboard.SignEvent(&ev, bbHub.signKeyID, bbHub.signSecret); err != nil {
			return "", fmt.Errorf("sign: %w", err)
		}
	}
	if err := bbHub.store.AppendEvent(context.Background(), ev); err != nil {
		return "", err
	}
	if _, err := a.blackboardSyncLocked(); err != nil {
		return ev.ID, fmt.Errorf("appended %s but sync failed: %w", ev.ID, err)
	}
	return ev.ID, nil
}

// ---------------------------------------------------------------------------
// Trust layer (spec §6.2)
// ---------------------------------------------------------------------------

// BlackboardConfigureTrust loads the host-private trusted_keys.json and the
// local signing key. Empty trustedKeysPath disables the trust layer (all
// events trusted — single-machine mode). 金鑰交換走帶外；信任根不放共享文件。
func (a *App) BlackboardConfigureTrust(trustedKeysPath, signingKeyID, signingSecretHex string) (BlackboardStatusDTO, error) {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.syncer == nil {
		return bbHub.statusLocked(), fmt.Errorf("blackboard not open")
	}
	if strings.TrimSpace(trustedKeysPath) == "" {
		bbHub.syncer.Trusted = nil
		bbHub.signKeyID, bbHub.signSecret = "", nil
	} else {
		ts, err := blackboard.LoadTrustStore(expandUserPath(trustedKeysPath))
		if err != nil {
			return bbHub.statusLocked(), err
		}
		bbHub.syncer.Trusted = ts.TrustFunc()
		if strings.TrimSpace(signingSecretHex) != "" {
			secret, err := hex.DecodeString(strings.TrimSpace(signingSecretHex))
			if err != nil || len(secret) == 0 {
				return bbHub.statusLocked(), fmt.Errorf("invalid signing secret hex")
			}
			bbHub.signKeyID = signingKeyID
			bbHub.signSecret = secret
		} else {
			bbHub.signKeyID, bbHub.signSecret = "", nil
		}
	}
	if _, err := a.blackboardSyncLocked(); err != nil {
		return bbHub.statusLocked(), err
	}
	return bbHub.statusLocked(), nil
}

// ---------------------------------------------------------------------------
// Coordinator loop (spec §9 + §1 host master + §13 auto-archive)
// ---------------------------------------------------------------------------

// BlackboardEnableCoordinator starts the coordinator loop: acquire/renew the
// lease and, while holding it, rewrite the host master document and archive
// when thresholds are exceeded.
func (a *App) BlackboardEnableCoordinator(masterPath string, leaseSeconds int) (BlackboardStatusDTO, error) {
	masterPath = strings.TrimSpace(expandUserPath(masterPath))
	if masterPath == "" {
		return a.BlackboardStatus(), fmt.Errorf("masterPath is required")
	}
	if leaseSeconds < 10 {
		leaseSeconds = 60
	}
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.syncer == nil {
		return bbHub.statusLocked(), fmt.Errorf("blackboard not open")
	}
	if bbHub.coordCancel != nil {
		return bbHub.statusLocked(), fmt.Errorf("coordinator already enabled")
	}
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "host"
	}
	bbHub.instance = blackboard.Actor{
		Type: blackboard.ActorApp,
		ID:   fmt.Sprintf("app_%s_%s", bbHub.actor.ID, hostname),
	}
	bbHub.masterPath = masterPath

	ctx, cancel := context.WithCancel(context.Background())
	bbHub.coordCancel = cancel
	leaseDur := time.Duration(leaseSeconds) * time.Second
	tick := leaseDur / 3
	if tick < time.Second {
		tick = time.Second
	}
	go a.coordinatorLoop(ctx, leaseDur, tick)

	// Run one tick immediately so the UI sees the takeover without waiting.
	a.coordinatorTickLocked(ctx, leaseDur)
	return bbHub.statusLocked(), nil
}

// BlackboardDisableCoordinator stops the loop (the lease simply expires).
func (a *App) BlackboardDisableCoordinator() BlackboardStatusDTO {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.coordCancel != nil {
		bbHub.coordCancel()
		bbHub.coordCancel = nil
	}
	bbHub.masterPath = ""
	bbHub.instance = blackboard.Actor{}
	return bbHub.statusLocked()
}

func (a *App) coordinatorLoop(ctx context.Context, leaseDur, tick time.Duration) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bbHub.mu.Lock()
			if bbHub.syncer == nil || bbHub.coordCancel == nil {
				bbHub.mu.Unlock()
				return
			}
			a.coordinatorTickLocked(ctx, leaseDur)
			bbHub.mu.Unlock()
		}
	}
}

// coordinatorTickLocked runs one lease/projection round. Caller holds bbHub.mu.
func (a *App) coordinatorTickLocked(ctx context.Context, leaseDur time.Duration) {
	res, err := a.blackboardSyncLocked()
	if err != nil {
		return // transient; next tick retries
	}
	now := time.Now()
	holder := bbHub.instance.ID

	decision, epoch := blackboard.DecideLease(res.State.Lease, holder, now)
	if decision == blackboard.LeaseAcquire || decision == blackboard.LeaseRenew {
		ev, err := bbHub.stampLocked(blackboard.KindCoordinatorLease, bbHub.instance)
		if err != nil {
			return
		}
		ev.Holder = holder
		ev.Epoch = epoch
		ev.LeaseUntil = now.Add(leaseDur).UTC().Format(time.RFC3339)
		if _, err := a.appendStampedLocked(ev); err != nil {
			return // lost the race or I/O trouble; next tick re-evaluates
		}
		if fresh, err := a.blackboardSyncLocked(); err == nil {
			res = fresh
		}
	}

	l := res.State.Lease
	if l == nil || l.Holder != holder || !blackboard.LeaseValidAt(l, now) {
		return // someone else coordinates; we just observe
	}

	// Holding a valid lease: rewrite the host master (spec §1) …
	_ = blackboard.WriteHostMaster(bbHub.masterPath, res.State, holder, now.UTC().Format(time.RFC3339))

	// … and archive when thresholds trip (spec §13). Marker authority is the
	// human/coordinator identity of this session, per the permission table.
	markerActor := blackboard.Actor{Type: blackboard.ActorCoordinator, ID: bbHub.actor.ID}
	archiveDir := filepath.Join(filepath.Dir(bbHub.masterPath), "archive")
	if r, err := bbHub.syncer.ArchiveIfNeeded(ctx, blackboard.DefaultArchiveConfig(), markerActor, archiveDir, now); err == nil && r != nil {
		_, _ = a.blackboardSyncLocked()
	}
}

// BlackboardArchiveNow triggers archival manually with default thresholds.
// Only human/coordinator sessions may leave the archive_marker.
func (a *App) BlackboardArchiveNow() (*blackboard.ArchiveResult, error) {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.syncer == nil {
		return nil, fmt.Errorf("blackboard not open")
	}
	if bbHub.actor.Type != blackboard.ActorHuman && bbHub.actor.Type != blackboard.ActorCoordinator {
		return nil, fmt.Errorf("archive requires a human or coordinator session")
	}
	markerActor := blackboard.Actor{Type: blackboard.ActorCoordinator, ID: bbHub.actor.ID}
	archiveDir := filepath.Join(filepath.Dir(bbHub.docPath), "archive")
	if bbHub.masterPath != "" {
		archiveDir = filepath.Join(filepath.Dir(bbHub.masterPath), "archive")
	}
	res, err := bbHub.syncer.ArchiveIfNeeded(context.Background(),
		blackboard.DefaultArchiveConfig(), markerActor, archiveDir, time.Now())
	if err != nil {
		return nil, err
	}
	if res != nil {
		_, _ = a.blackboardSyncLocked()
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Real deletion (spec §8)
// ---------------------------------------------------------------------------

// BlackboardRequestDeletion appends a delete_text_request. Any participant
// (including agents) may request; only a host can approve.
func (a *App) BlackboardRequestDeletion(targetID, reason string) (string, error) {
	if err := blackboard.ValidateEventID(targetID); err != nil {
		return "", fmt.Errorf("targetID: %w", err)
	}
	return a.blackboardAppend(blackboard.KindDeleteTextRequest, func(e *blackboard.Event) {
		e.TargetID = targetID
		e.Reason = reason
	})
}

// BlackboardApproveDeletion executes a pending delete_text_request: the
// session actor must be human or coordinator. The engine mirrors a backup,
// removes the block, tombstones the id, and appends redaction_applied
// (never containing the sensitive original).
func (a *App) BlackboardApproveDeletion(requestEventID string) (string, error) {
	bbHub.mu.Lock()
	defer bbHub.mu.Unlock()
	if bbHub.syncer == nil {
		return "", fmt.Errorf("blackboard not open")
	}
	if bbHub.actor.Type != blackboard.ActorHuman && bbHub.actor.Type != blackboard.ActorCoordinator {
		return "", fmt.Errorf("deletion approval requires a human or coordinator session")
	}
	ctx := context.Background()
	content, _, err := bbHub.store.ReadLog(ctx)
	if err != nil {
		return "", err
	}
	var target, reason string
	for _, pe := range blackboard.ParseLog(content).Events {
		if pe.Event.ID == requestEventID && pe.Event.Kind == blackboard.KindDeleteTextRequest {
			target = pe.Event.TargetID
			reason = pe.Event.Reason
			break
		}
	}
	if target == "" {
		return "", fmt.Errorf("delete_text_request %s not found", requestEventID)
	}
	redactionID, err := bbHub.syncer.Redact(ctx, target, reason, bbHub.actor, time.Now())
	if err != nil {
		return "", err
	}
	if _, err := a.blackboardSyncLocked(); err != nil {
		return redactionID, fmt.Errorf("redacted but sync failed: %w", err)
	}
	return redactionID, nil
}
