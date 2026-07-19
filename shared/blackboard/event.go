// Package blackboard implements the multi-party meeting blackboard engine
// defined in BLACKBOARD_SPEC.md.
//
// Design invariants:
//   - The shared Event Log is append-only; corrections are new events.
//   - bbm_<ULID> is event identity; document position is only the admission
//     order; canonical_seq (assigned once) is the permanent order.
//   - Meeting State is a pure-function projection over the event sequence.
//   - The carrier (local folder, synced folder, Google Docs, ...) is
//     abstracted behind the Store interface; the engine never assumes one.
package blackboard

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SchemaVersion is the highest event schema version this engine understands.
const SchemaVersion = 1

// IDPrefix prefixes every event ID.
const IDPrefix = "bbm_"

// Sentinel marks the end of the Event Log region in the shared document.
const Sentinel = "<!-- BBM_EVENT_LOG_END -->"

// Event kinds (spec §2.2).
const (
	KindMessage           = "message"
	KindProposal          = "proposal"
	KindDecision          = "decision"
	KindActionItem        = "action_item"
	KindStatusUpdate      = "status_update"
	KindRetraction        = "retraction"
	KindAmendment         = "amendment"
	KindScopeExclusion    = "scope_exclusion"
	KindArtifactRef       = "artifact_ref"
	KindSourceRef         = "source_ref"
	KindArtifactUpdate    = "artifact_update"
	KindArtifactOffer     = "artifact_offer"
	KindRequest           = "request"
	KindVoteProposal      = "vote_proposal"
	KindVoteOpen          = "vote_open"
	KindVoteCast          = "vote_cast"
	KindVoteClose         = "vote_close"
	KindVoteResult        = "vote_result"
	KindCoordinatorLease  = "coordinator_lease"
	KindArchiveMarker     = "archive_marker"
	KindDeleteTextRequest = "delete_text_request"
	KindRedactionApplied  = "redaction_applied"
	KindIntegrityWarning  = "integrity_warning"
)

// Actor types.
const (
	ActorHuman       = "human"
	ActorAgent       = "agent"
	ActorCoordinator = "coordinator"
	ActorApp         = "app"
)

// knownKinds is the set of kinds this engine projects. Unknown kinds are
// preserved but excluded from state projection (spec §5).
var knownKinds = map[string]bool{
	KindMessage: true, KindProposal: true, KindDecision: true,
	KindActionItem: true, KindStatusUpdate: true, KindRetraction: true,
	KindAmendment: true, KindScopeExclusion: true, KindArtifactRef: true,
	KindSourceRef: true, KindArtifactUpdate: true, KindArtifactOffer: true,
	KindRequest: true, KindVoteProposal: true, KindVoteOpen: true,
	KindVoteCast: true, KindVoteClose: true, KindVoteResult: true,
	KindCoordinatorLease: true, KindArchiveMarker: true,
	KindDeleteTextRequest: true, KindRedactionApplied: true,
	KindIntegrityWarning: true,
}

// KnownKind reports whether the engine projects the given kind.
func KnownKind(kind string) bool { return knownKinds[kind] }

// Actor identifies who claims to have emitted an event.
type Actor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// String returns "type:id", the canonical voter/permission form.
func (a Actor) String() string { return a.Type + ":" + a.ID }

// Sig is an optional HMAC signature envelope (spec §6.2).
type Sig struct {
	Alg   string `json:"alg"`
	KeyID string `json:"key_id"`
	Value string `json:"value"`
}

// EventRange marks the inclusive id range covered by an archive_marker.
type EventRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Ref points at an external artifact or document.
type Ref struct {
	Type  string `json:"type,omitempty"`
	URL   string `json:"url,omitempty"`
	DocID string `json:"doc_id,omitempty"`
	Path  string `json:"path,omitempty"`
}

// Event is the universal event envelope. Kind-specific fields are all
// optional; Raw preserves the exact JSON as written for hashing and
// amendment application.
type Event struct {
	V         int    `json:"v"`
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Actor     Actor  `json:"actor"`
	CreatedAt string `json:"created_at"`
	Sig       *Sig   `json:"sig,omitempty"`

	// Common payloads.
	Body     string `json:"body,omitempty"`
	Title    string `json:"title,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Reason   string `json:"reason,omitempty"`
	TargetID string `json:"target_id,omitempty"`

	// amendment
	Patch json.RawMessage `json:"patch,omitempty"`

	// action_item / status_update
	Assignee string `json:"assignee,omitempty"`
	Task     string `json:"task,omitempty"`
	Status   string `json:"status,omitempty"`

	// decision
	Decision string `json:"decision,omitempty"`

	// artifact_ref / source_ref
	Ref       *Ref     `json:"ref,omitempty"`
	RelatedTo []string `json:"related_to,omitempty"`
	SourceIDs []string `json:"source_ids,omitempty"`

	// request
	To     string `json:"to,omitempty"`
	Intent string `json:"intent,omitempty"`

	// vote_*
	Topic          string   `json:"topic,omitempty"`
	Options        []string `json:"options,omitempty"`
	EligibleVoters []string `json:"eligible_voters,omitempty"`
	ClosesAt       string   `json:"closes_at,omitempty"`
	Choice         string   `json:"choice,omitempty"`
	Result         string   `json:"result,omitempty"`

	// coordinator_lease
	Holder     string `json:"holder,omitempty"`
	LeaseUntil string `json:"lease_until,omitempty"`
	Epoch      int    `json:"epoch,omitempty"`

	// integrity_warning
	MissingIDs []string `json:"missing_ids,omitempty"`

	// archive_marker
	Range      *EventRange `json:"range,omitempty"`
	ArchiveRef *Ref        `json:"archive_ref,omitempty"`

	// Raw is the exact fenced JSON as parsed from the document.
	// Not serialized; used for hashing and as the amendment base.
	Raw json.RawMessage `json:"-"`
}

// Validate checks the required envelope fields (spec §2.1).
func (e *Event) Validate() error {
	if e.V <= 0 {
		return fmt.Errorf("missing or invalid v")
	}
	if err := ValidateEventID(e.ID); err != nil {
		return err
	}
	if e.Kind == "" {
		return fmt.Errorf("missing kind")
	}
	if e.Actor.Type == "" || e.Actor.ID == "" {
		return fmt.Errorf("missing actor")
	}
	if e.CreatedAt == "" {
		return fmt.Errorf("missing created_at")
	}
	if _, err := time.Parse(time.RFC3339, e.CreatedAt); err != nil {
		return fmt.Errorf("created_at is not RFC3339: %v", err)
	}
	return nil
}

// ValidateEventID rejects look-alike IDs that only carry the bbm_ prefix.
// The first ULID character is limited to 0..7 because a ULID encodes exactly
// 128 bits in 26 Crockford Base32 characters (the leading two bits are zero).
func ValidateEventID(id string) error {
	if !strings.HasPrefix(id, IDPrefix) {
		return fmt.Errorf("id must have prefix %q", IDPrefix)
	}
	ulid := strings.TrimPrefix(id, IDPrefix)
	if len(ulid) != 26 {
		return fmt.Errorf("id must be %s plus a 26-character ULID", IDPrefix)
	}
	if ulid[0] < '0' || ulid[0] > '7' {
		return fmt.Errorf("ULID first character must be 0..7")
	}
	for _, ch := range ulid {
		if !strings.ContainsRune(crockford, ch) {
			return fmt.Errorf("ULID contains non-Crockford character %q", ch)
		}
	}
	return nil
}

// crockford is the Crockford base32 alphabet used by ULID.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewULID returns a 26-character ULID for the given time using
// crypto/rand entropy. Implemented inline to avoid a new dependency.
func NewULID(t time.Time) (string, error) {
	var b [16]byte
	ms := uint64(t.UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	if _, err := rand.Read(b[6:]); err != nil {
		return "", fmt.Errorf("ulid entropy: %w", err)
	}
	// Encode 128 bits as 26 base32 chars (2 zero bits of left padding).
	var out [26]byte
	// Build a big-endian bit reader over b.
	bitPos := -2 // pretend two leading zero bits
	for i := 0; i < 26; i++ {
		var v byte
		for j := 0; j < 5; j++ {
			v <<= 1
			p := bitPos + i*5 + j
			if p >= 0 {
				byteIdx := p / 8
				bitIdx := 7 - (p % 8)
				v |= (b[byteIdx] >> bitIdx) & 1
			}
		}
		out[i] = crockford[v]
	}
	return string(out[:]), nil
}

// NewEventID returns a fresh "bbm_<ULID>" identifier.
func NewEventID(t time.Time) (string, error) {
	u, err := NewULID(t)
	if err != nil {
		return "", err
	}
	return IDPrefix + u, nil
}
