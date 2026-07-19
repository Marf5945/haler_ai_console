package blackboard

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// DeadLetterReason classifies why an entry was quarantined (spec §5).
type DeadLetterReason string

const (
	ReasonParseError      DeadLetterReason = "json_parse_error"
	ReasonSchemaError     DeadLetterReason = "schema_validation_error"
	ReasonConflict        DeadLetterReason = "conflict_duplicate_id_hash_mismatch"
	ReasonUpgradeRequired DeadLetterReason = "unknown_schema_version"
	ReasonHeadingMismatch DeadLetterReason = "heading_id_mismatch"
	ReasonMissingFence    DeadLetterReason = "missing_json_fence"
)

// DeadLetter records an entry that could not be admitted. Dead letters are
// written to a local jsonl only — never back into the shared log.
type DeadLetter struct {
	Line    int              `json:"line"`
	EventID string           `json:"event_id,omitempty"`
	Reason  DeadLetterReason `json:"reason"`
	Detail  string           `json:"detail,omitempty"`
	RawText string           `json:"raw_text,omitempty"`
}

// ParsedEvent is an admitted event plus its parse metadata.
type ParsedEvent struct {
	Event   Event
	DocPos  int    // 0-based order of appearance in this parse round
	Line    int    // 1-based line number of the heading
	Hash    string // sha256 hex of the compacted raw JSON
	RawJSON []byte // exact fenced JSON payload
	Skipped bool   // true when unknown v (kept for visibility, not projected)
}

// ParseResult is the outcome of one parse round over the Event Log region.
type ParseResult struct {
	Events      []ParsedEvent
	DeadLetters []DeadLetter
}

var headingRe = regexp.MustCompile(`^###\s+BBM\s+(\S+)(?:\s+kind:(\S+))?\s*$`)

// HashJSON returns the canonical hash of an event payload: sha256 over the
// JSON with insignificant whitespace removed.
func HashJSON(raw []byte) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		// Fall back to hashing the raw bytes; parse errors are handled
		// separately by the caller.
		sum := sha256.Sum256(raw)
		return hex.EncodeToString(sum[:])
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

// ParseLog scans the Event Log region of a shared document and extracts
// events per spec §2. Content after the Sentinel is ignored. Any malformed
// entry becomes a DeadLetter; parsing never aborts (spec §5).
func ParseLog(content string) ParseResult {
	var res ParseResult
	seen := map[string]string{} // id -> hash of first occurrence

	// Only the region before the sentinel is the live log.
	if idx := strings.Index(content, Sentinel); idx >= 0 {
		content = content[:idx]
	}
	lines := strings.Split(content, "\n")

	i := 0
	for i < len(lines) {
		m := headingRe.FindStringSubmatch(strings.TrimRight(lines[i], "\r"))
		if m == nil {
			i++
			continue
		}
		headingLine := i + 1
		headingID := m[1]

		// Find the fenced JSON block that follows (allowing blank lines).
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[j]), "```") {
			res.DeadLetters = append(res.DeadLetters, DeadLetter{
				Line: headingLine, EventID: headingID,
				Reason: ReasonMissingFence,
				Detail: "heading not followed by a fenced JSON block",
			})
			i++
			continue
		}
		fenceStart := j
		j++
		var body []string
		closed := false
		for j < len(lines) {
			if strings.TrimSpace(lines[j]) == "```" {
				closed = true
				break
			}
			body = append(body, lines[j])
			j++
		}
		if !closed {
			res.DeadLetters = append(res.DeadLetters, DeadLetter{
				Line: fenceStart + 1, EventID: headingID,
				Reason: ReasonMissingFence,
				Detail: "unterminated JSON fence",
			})
			break // rest of document is inside the broken fence
		}
		i = j + 1 // continue after the closing fence

		raw := []byte(strings.Join(body, "\n"))
		admitParsed(&res, seen, headingLine, headingID, raw)
	}
	return res
}

// admitParsed validates one raw payload and appends either a ParsedEvent or
// a DeadLetter.
func admitParsed(res *ParseResult, seen map[string]string, line int, headingID string, raw []byte) {
	truncated := truncate(string(raw), 500)

	var ev Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		res.DeadLetters = append(res.DeadLetters, DeadLetter{
			Line: line, EventID: headingID, Reason: ReasonParseError,
			Detail: err.Error(), RawText: truncated,
		})
		return
	}
	ev.Raw = append([]byte(nil), raw...)

	if ev.ID != headingID {
		res.DeadLetters = append(res.DeadLetters, DeadLetter{
			Line: line, EventID: headingID, Reason: ReasonHeadingMismatch,
			Detail:  fmt.Sprintf("heading id %q != body id %q", headingID, ev.ID),
			RawText: truncated,
		})
		return
	}
	if err := ev.Validate(); err != nil {
		res.DeadLetters = append(res.DeadLetters, DeadLetter{
			Line: line, EventID: headingID, Reason: ReasonSchemaError,
			Detail: err.Error(), RawText: truncated,
		})
		return
	}

	hash := HashJSON(raw)
	if prev, dup := seen[ev.ID]; dup {
		if prev == hash {
			return // identical duplicate: idempotent, silently ignore
		}
		res.DeadLetters = append(res.DeadLetters, DeadLetter{
			Line: line, EventID: ev.ID, Reason: ReasonConflict,
			Detail:  "same id appears twice with different content",
			RawText: truncated,
		})
		return
	}
	seen[ev.ID] = hash

	pe := ParsedEvent{
		Event: ev, DocPos: len(res.Events), Line: line,
		Hash: hash, RawJSON: ev.Raw,
	}
	if ev.V > SchemaVersion {
		pe.Skipped = true
		res.DeadLetters = append(res.DeadLetters, DeadLetter{
			Line: line, EventID: ev.ID, Reason: ReasonUpgradeRequired,
			Detail: fmt.Sprintf("event v=%d > supported v=%d; upgrade needed", ev.V, SchemaVersion),
		})
	}
	res.Events = append(res.Events, pe)
}

// FormatEvent renders an event as the Markdown heading + fenced JSON block
// to be appended before the sentinel (spec §2). The JSON is written with
// two-space indentation for human readability.
func FormatEvent(ev Event) (string, error) {
	if err := ev.Validate(); err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "### BBM %s kind:%s\n", ev.ID, ev.Kind)
	b.WriteString("```json\n")
	b.Write(raw)
	b.WriteString("\n```\n\n")
	return b.String(), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
