package blackboard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// SeqEvent couples a parsed event with its permanent canonical sequence
// number (spec §3). Seq is assigned once by the synchronizer and never
// changes across archival or document reshuffling.
type SeqEvent struct {
	ParsedEvent
	Seq int64
}

// ProjectionStatus values for per-event adjudication.
const (
	StatusAccepted      = "accepted"
	StatusRejected      = "rejected"
	StatusRetracted     = "retracted"
	StatusIgnoredKind   = "ignored_unknown_kind"
	StatusSkippedSchema = "skipped_version"
	StatusLateVote      = "rejected_late_vote"
	StatusUntrusted     = "untrusted"
)

// Adjudication records how the projector ruled on one event. Violating
// events are never deleted — only marked (spec §6.1).
type Adjudication struct {
	EventID string `json:"event_id"`
	Seq     int64  `json:"seq"`
	Kind    string `json:"kind"`
	Status  string `json:"projection_status"`
	Reason  string `json:"reason,omitempty"`
}

// TrustFunc reports whether an event's claimed identity is trusted.
// nil means "trust everything" (single-machine / MVP file transport).
type TrustFunc func(ev Event) bool

// --- Meeting State view types (all slices are in canonical order) ---

type AgendaItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type DecisionItem struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Status string `json:"status,omitempty"` // e.g. done / cancelled via status_update
}

type ActionItem struct {
	ID       string `json:"id"`
	Assignee string `json:"assignee"`
	Task     string `json:"task"`
	Status   string `json:"status,omitempty"`
}

type Material struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary,omitempty"`
	Ref     *Ref   `json:"ref,omitempty"`
}

type ScopedOut struct {
	TargetID string `json:"target_id"`
	Reason   string `json:"reason,omitempty"`
}

type VoterChoice struct {
	Voter  string `json:"voter"`
	Choice string `json:"choice"`
}

type OptionCount struct {
	Option string `json:"option"`
	Count  int    `json:"count"`
}

type VoteView struct {
	ID             string        `json:"id"`
	Topic          string        `json:"topic"`
	Options        []string      `json:"options"`
	Closed         bool          `json:"closed"`
	Tally          []OptionCount `json:"tally"`
	Choices        []VoterChoice `json:"choices"`
	ComputedResult string        `json:"computed_result"`
	ResultMismatch bool          `json:"result_mismatch,omitempty"`
}

type RequestView struct {
	ID       string `json:"id"`
	To       string `json:"to"`
	Intent   string `json:"intent"`
	TargetID string `json:"target_id,omitempty"`
}

type LeaseView struct {
	EventID    string `json:"event_id"`
	Holder     string `json:"holder"`
	LeaseUntil string `json:"lease_until"`
	Epoch      int    `json:"epoch"`
}

// MeetingState is the pure-function projection of the event sequence.
type MeetingState struct {
	ProjectedUntil  string         `json:"projected_until"`
	ProjectedSeq    int64          `json:"projected_seq"`
	EventCount      int            `json:"event_count"`
	LogHash         string         `json:"log_hash"`
	DeadLetterCount int            `json:"dead_letter_count"`
	Agenda          []AgendaItem   `json:"agenda"`
	Decisions       []DecisionItem `json:"decisions"`
	ActionItems     []ActionItem   `json:"action_items"`
	SharedMaterials []Material     `json:"shared_materials"`
	Offers          []Material     `json:"artifact_offers"`
	OutOfScope      []ScopedOut    `json:"out_of_scope"`
	Votes           []VoteView     `json:"votes"`
	Requests        []RequestView  `json:"requests"`
	Lease           *LeaseView     `json:"lease,omitempty"`
	LeaseChain      []LeaseView    `json:"lease_chain"`
	Adjudications   []Adjudication `json:"adjudications"`
}

// node is the projector's working record for one event.
type node struct {
	se           SeqEvent
	status       string
	reason       string
	untrusted    bool
	retracted    bool
	excluded     bool
	excludeInfo  *ScopedOut
	effectiveRaw json.RawMessage
	effective    Event
}

// voteState tracks one open/closed vote during replay.
type voteState struct {
	open    *node
	closed  bool
	choices map[string]string // voter -> last accepted choice
	order   []string          // voter first-seen order for determinism
	result  *node             // last accepted vote_result
}

// Project replays events (must be pre-sorted by Seq ascending) and returns
// the Meeting State. It is a pure function: no clock, no I/O, no map
// iteration order leaks into the output. deadLetterCount is carried into
// the state for UI display only.
func Project(events []SeqEvent, trusted TrustFunc, deadLetterCount int) MeetingState {
	nodes := make([]*node, 0, len(events))
	byID := make(map[string]*node, len(events))
	logHash := ""

	// --- Pass A: static admission only. Effects are deliberately deferred so
	// a later retraction can invalidate an earlier modifier before replay. ---
	for _, se := range events {
		n := &node{se: se, status: StatusAccepted, effectiveRaw: se.RawJSON, effective: se.Event}
		nodes = append(nodes, n)

		// Chain hash over every admitted-for-parsing event.
		sum := sha256.Sum256([]byte(logHash + se.Hash))
		logHash = hex.EncodeToString(sum[:])

		if se.Skipped {
			n.status = StatusSkippedSchema
			n.reason = "unknown schema version"
			continue
		}
		if !KnownKind(se.Event.Kind) {
			n.status = StatusIgnoredKind
			n.reason = "unknown kind: preserved but not projected"
			continue
		}
		if trusted != nil && !trusted(se.Event) {
			// Untrusted events are readable data only (spec §6.2): a
			// message survives flagged; anything with authority is rejected.
			if se.Event.Kind == KindMessage {
				n.untrusted = true
			} else {
				n.status = StatusRejected
				n.reason = "untrusted source for kind " + se.Event.Kind
				continue
			}
		}
		if ok, why := checkPermission(se.Event, byID); !ok {
			n.status = StatusRejected
			n.reason = why
			continue
		}
		byID[se.Event.ID] = n
	}

	// Retractions are evaluated backwards. If a retraction is itself retracted,
	// it never invalidates its target; this also handles longer retraction chains
	// without recursion or a separate state-machine layer.
	for i := len(nodes) - 1; i >= 0; i-- {
		n := nodes[i]
		if n.status != StatusAccepted || n.retracted {
			continue
		}
		if n.se.Event.Kind == KindRetraction {
			if target := byID[n.se.Event.TargetID]; target != nil {
				target.retracted = true
			}
		}
	}

	// An amendment may itself be amended. Resolve only replay modifiers here;
	// ordinary target events still receive their patches in canonical order in
	// Pass B, preserving ordering against status_update and vote events.
	amendments := make(map[string][]*node)
	for _, n := range nodes {
		if n.status == StatusAccepted && !n.retracted &&
			(n.se.Event.Kind == KindAmendment || n.se.Event.Kind == KindArtifactUpdate) {
			amendments[n.se.Event.TargetID] = append(amendments[n.se.Event.TargetID], n)
		}
	}
	resolved := make(map[string]bool)
	resolving := make(map[string]bool)
	var resolveModifier func(*node)
	resolveModifier = func(target *node) {
		if resolved[target.se.Event.ID] || resolving[target.se.Event.ID] {
			return
		}
		resolving[target.se.Event.ID] = true
		for _, amendment := range amendments[target.se.Event.ID] {
			resolveModifier(amendment)
			if amendment.status != StatusAccepted || amendment.retracted || len(amendment.effective.Patch) == 0 {
				continue
			}
			if err := applyPatchToNode(target, amendment.effective.Patch); err != nil {
				amendment.status = StatusRejected
				amendment.reason = "merge patch failed: " + err.Error()
			}
		}
		delete(resolving, target.se.Event.ID)
		resolved[target.se.Event.ID] = true
	}
	for _, n := range nodes {
		if isReplayModifier(n.se.Event.Kind) {
			resolveModifier(n)
		}
	}

	// --- Pass B: replay active effects in canonical order. ---
	votes := make(map[string]*voteState)
	for _, n := range nodes {
		if n.status != StatusAccepted || n.retracted {
			continue
		}
		ev := n.effective
		switch ev.Kind {
		case KindRetraction:
			// Already adjudicated backwards above.
		case KindAmendment, KindArtifactUpdate:
			target := byID[ev.TargetID]
			if target == nil || isReplayModifier(target.se.Event.Kind) {
				// Modifier targets were pre-resolved so their corrected payload is
				// used when their earlier canonical position is replayed.
				continue
			}
			if len(ev.Patch) > 0 {
				if err := applyPatchToNode(target, ev.Patch); err != nil {
					n.status = StatusRejected
					n.reason = "merge patch failed: " + err.Error()
				}
			}
		case KindStatusUpdate:
			target := byID[ev.TargetID]
			if target == nil {
				n.status, n.reason = StatusRejected, "status_update target not found"
				continue
			}
			privileged := ev.Actor.Type == ActorHuman || ev.Actor.Type == ActorCoordinator
			if !privileged && ev.Actor.String() != target.effective.Assignee {
				n.status, n.reason = StatusRejected, "status_update requires assignee, human, or coordinator"
				continue
			}
			// Keep effectiveRaw and effective in lockstep. A later amendment must
			// layer on this status instead of reconstructing the pre-update event.
			statusPatch, _ := json.Marshal(map[string]string{"status": ev.Status})
			if err := applyPatchToNode(target, statusPatch); err != nil {
				n.status, n.reason = StatusRejected, "status_update failed: "+err.Error()
			}
		case KindScopeExclusion:
			if target := byID[ev.TargetID]; target != nil {
				target.excluded = true
				target.excludeInfo = &ScopedOut{TargetID: target.se.Event.ID, Reason: ev.Reason}
			}
		case KindVoteOpen:
			votes[ev.ID] = &voteState{open: n, choices: map[string]string{}}
		case KindVoteCast:
			vs := votes[ev.TargetID]
			if vs == nil {
				n.status, n.reason = StatusRejected, "vote_cast targets unknown vote"
				continue
			}
			if vs.closed {
				n.status, n.reason = StatusLateVote, "cast after vote_close"
				continue
			}
			voter := ev.Actor.String()
			if !contains(vs.open.effective.EligibleVoters, voter) {
				n.status, n.reason = StatusRejected, "voter "+voter+" not in eligible_voters"
				continue
			}
			if !contains(vs.open.effective.Options, ev.Choice) {
				n.status, n.reason = StatusRejected, "choice not in vote options"
				continue
			}
			if _, seen := vs.choices[voter]; !seen {
				vs.order = append(vs.order, voter)
			}
			vs.choices[voter] = ev.Choice
		case KindVoteClose:
			vs := votes[ev.TargetID]
			if vs == nil {
				n.status, n.reason = StatusRejected, "vote_close targets unknown vote"
				continue
			}
			vs.closed = true
		case KindVoteResult:
			vs := votes[ev.TargetID]
			if vs == nil {
				n.status, n.reason = StatusRejected, "vote_result targets unknown vote"
				continue
			}
			vs.result = n
		}
	}

	// --- Pass C: materialize state in canonical order. ---
	st := MeetingState{DeadLetterCount: deadLetterCount}
	var leaseBest *node
	for _, n := range nodes {
		ev := n.effective
		st.EventCount++
		adj := Adjudication{EventID: n.se.Event.ID, Seq: n.se.Seq, Kind: n.se.Event.Kind, Status: n.status, Reason: n.reason}
		if n.retracted && n.status == StatusAccepted {
			adj.Status = StatusRetracted
		}
		if n.untrusted && n.status == StatusAccepted {
			adj.Status = StatusUntrusted
		}
		st.Adjudications = append(st.Adjudications, adj)

		if n.status != StatusAccepted || n.retracted {
			continue
		}
		if n.excluded {
			if n.excludeInfo != nil {
				st.OutOfScope = append(st.OutOfScope, *n.excludeInfo)
			}
			continue
		}

		switch ev.Kind {
		case KindProposal:
			title := ev.Title
			if title == "" {
				title = ev.Body
			}
			st.Agenda = append(st.Agenda, AgendaItem{ID: ev.ID, Title: title})
		case KindDecision:
			text := ev.Decision
			if text == "" {
				text = ev.Body
			}
			st.Decisions = append(st.Decisions, DecisionItem{ID: ev.ID, Text: text, Status: ev.Status})
		case KindActionItem:
			st.ActionItems = append(st.ActionItems, ActionItem{ID: ev.ID, Assignee: ev.Assignee, Task: ev.Task, Status: ev.Status})
		case KindArtifactRef, KindSourceRef:
			st.SharedMaterials = append(st.SharedMaterials, Material{ID: ev.ID, Title: ev.Title, Summary: ev.Summary, Ref: ev.Ref})
		case KindArtifactOffer:
			st.Offers = append(st.Offers, Material{ID: ev.ID, Title: ev.Title, Summary: ev.Summary, Ref: ev.Ref})
		case KindRequest:
			st.Requests = append(st.Requests, RequestView{ID: ev.ID, To: ev.To, Intent: ev.Intent, TargetID: ev.TargetID})
		case KindCoordinatorLease:
			st.LeaseChain = append(st.LeaseChain, LeaseView{
				EventID: ev.ID, Holder: ev.Holder, LeaseUntil: ev.LeaseUntil, Epoch: ev.Epoch,
			})
			// Highest epoch wins; ties resolved by later canonical position
			// (spec §9). Runtime expiry is the app's job, not the projector's.
			if leaseBest == nil || ev.Epoch > leaseBest.effective.Epoch ||
				(ev.Epoch == leaseBest.effective.Epoch && n.se.Seq > leaseBest.se.Seq) {
				leaseBest = n
			}
		}

		st.ProjectedUntil = n.se.Event.ID
		st.ProjectedSeq = n.se.Seq
	}
	// ProjectedUntil should reflect the last event examined, accepted or not.
	if len(nodes) > 0 {
		last := nodes[len(nodes)-1]
		st.ProjectedUntil = last.se.Event.ID
		st.ProjectedSeq = last.se.Seq
	}
	st.LogHash = logHash

	if leaseBest != nil {
		lv := LeaseView{
			EventID: leaseBest.se.Event.ID, Holder: leaseBest.effective.Holder,
			LeaseUntil: leaseBest.effective.LeaseUntil, Epoch: leaseBest.effective.Epoch,
		}
		st.Lease = &lv
	}

	// Votes, in vote_open canonical order.
	for _, n := range nodes {
		if n.se.Event.Kind != KindVoteOpen || n.status != StatusAccepted || n.retracted {
			continue
		}
		vs := votes[n.se.Event.ID]
		if vs == nil {
			continue
		}
		view := VoteView{
			ID: n.se.Event.ID, Topic: n.effective.Topic,
			Options: n.effective.Options, Closed: vs.closed,
		}
		counts := map[string]int{}
		for _, voter := range vs.order {
			choice := vs.choices[voter]
			view.Choices = append(view.Choices, VoterChoice{Voter: voter, Choice: choice})
			counts[choice]++
		}
		var parts []string
		for _, opt := range n.effective.Options {
			view.Tally = append(view.Tally, OptionCount{Option: opt, Count: counts[opt]})
			parts = append(parts, fmt.Sprintf("%s=%d", opt, counts[opt]))
		}
		view.ComputedResult = strings.Join(parts, ",")
		if vs.result != nil && vs.result.effective.Result != "" &&
			vs.result.effective.Result != view.ComputedResult {
			view.ResultMismatch = true // recomputation is authoritative (spec §10)
		}
		st.Votes = append(st.Votes, view)
	}

	return st
}

// applyPatchToNode is the single amendment mutation point. Validation after
// merge keeps malformed effective events out of all later permission checks.
func applyPatchToNode(target *node, patch json.RawMessage) error {
	patched, err := ApplyMergePatch(target.effectiveRaw, patch)
	if err != nil {
		return err
	}
	var effective Event
	if err := json.Unmarshal(patched, &effective); err != nil {
		return fmt.Errorf("patched event is invalid: %w", err)
	}
	if err := effective.Validate(); err != nil {
		return fmt.Errorf("patched event fails schema: %w", err)
	}
	effective.Raw = patched
	target.effectiveRaw = patched
	target.effective = effective
	return nil
}

func isReplayModifier(kind string) bool {
	switch kind {
	case KindRetraction, KindAmendment, KindArtifactUpdate, KindStatusUpdate,
		KindScopeExclusion, KindVoteCast, KindVoteClose, KindVoteResult:
		return true
	default:
		return false
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// checkPermission enforces the permission table (spec §6.1). byID contains
// only previously admitted events, which suffices because the log is
// append-only: targets always precede their modifiers.
func checkPermission(ev Event, byID map[string]*node) (bool, string) {
	at := ev.Actor.Type
	isPriv := at == ActorHuman || at == ActorCoordinator

	switch ev.Kind {
	case KindMessage, KindProposal, KindVoteProposal, KindArtifactRef,
		KindSourceRef, KindArtifactOffer, KindRequest, KindIntegrityWarning,
		KindDeleteTextRequest:
		return true, ""

	case KindDecision, KindActionItem:
		if isPriv {
			return true, ""
		}
		return false, "kind " + ev.Kind + " requires human or coordinator"

	case KindVoteOpen, KindVoteClose, KindVoteResult:
		if isPriv {
			return true, ""
		}
		return false, "kind " + ev.Kind + " requires human or coordinator (agents use vote_proposal)"

	case KindArchiveMarker, KindRedactionApplied:
		if isPriv {
			return true, ""
		}
		return false, "kind " + ev.Kind + " requires human or coordinator"

	case KindVoteCast:
		target := byID[ev.TargetID]
		if target == nil || target.se.Event.Kind != KindVoteOpen {
			return false, "vote_cast targets unknown vote"
		}
		// Eligibility is checked during replay against the vote's effective
		// fields, after any earlier amendments have been applied.
		return true, ""

	case KindStatusUpdate:
		t := byID[ev.TargetID]
		if t == nil {
			return false, "status_update target not found"
		}
		if k := t.se.Event.Kind; k != KindActionItem && k != KindDecision {
			return false, "status_update only applies to action_item or decision"
		}
		// Assignee is amendment-sensitive, so dynamic permission is checked in
		// Pass B at the modifier's canonical position.
		return true, ""

	case KindRetraction, KindAmendment, KindArtifactUpdate:
		t := byID[ev.TargetID]
		if t == nil {
			return false, ev.Kind + " target not found"
		}
		if isPriv || ev.Actor == t.se.Event.Actor {
			return true, ""
		}
		return false, ev.Kind + " requires original author, human, or coordinator"

	case KindCoordinatorLease:
		if at == ActorApp {
			return true, ""
		}
		return false, "coordinator_lease may only be emitted by an app instance"

	case KindScopeExclusion:
		if byID[ev.TargetID] == nil {
			return false, "scope_exclusion target not found"
		}
		if isPriv {
			return true, ""
		}
		return false, "scope_exclusion requires human or coordinator"
	}
	return true, ""
}
