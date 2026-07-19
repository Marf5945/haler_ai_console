package blackboard

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RenderMeetingState renders the projected state as the Meeting State
// Markdown block written into the host master document.
//
// Determinism contract (spec §16): the output depends only on the state.
// Wall-clock values (projectedAt) and identity (projectedBy) are injected
// by the caller so tests can pin them.
func RenderMeetingState(st MeetingState, projectedBy, projectedAt string) string {
	var b strings.Builder
	b.WriteString("## Meeting State\n\n")

	header := struct {
		ProjectedUntil  string `json:"projected_until"`
		ProjectedAt     string `json:"projected_at,omitempty"`
		ProjectedBy     string `json:"projected_by,omitempty"`
		LeaseID         string `json:"lease_id,omitempty"`
		EventCount      int    `json:"event_count"`
		LastEventID     string `json:"last_event_id"`
		LogHash         string `json:"log_hash"`
		DeadLetterCount int    `json:"dead_letter_count"`
	}{
		ProjectedUntil: st.ProjectedUntil, ProjectedAt: projectedAt,
		ProjectedBy: projectedBy, EventCount: st.EventCount,
		LastEventID: st.ProjectedUntil, LogHash: st.LogHash,
		DeadLetterCount: st.DeadLetterCount,
	}
	if st.Lease != nil {
		header.LeaseID = st.Lease.EventID
	}
	hb, _ := json.MarshalIndent(header, "", "  ")
	b.WriteString("```json\n")
	b.Write(hb)
	b.WriteString("\n```\n\n")

	b.WriteString("### Agenda\n\n")
	if len(st.Agenda) == 0 {
		b.WriteString("(none)\n\n")
	} else {
		for _, a := range st.Agenda {
			fmt.Fprintf(&b, "- %s → source: %s\n", a.Title, a.ID)
		}
		b.WriteString("\n")
	}

	b.WriteString("### Decisions\n\n")
	if len(st.Decisions) == 0 {
		b.WriteString("(none)\n\n")
	} else {
		for _, d := range st.Decisions {
			if d.Status != "" {
				fmt.Fprintf(&b, "- [%s] %s → source: %s\n", d.Status, d.Text, d.ID)
			} else {
				fmt.Fprintf(&b, "- %s → source: %s\n", d.Text, d.ID)
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("### Action Items\n\n")
	if len(st.ActionItems) == 0 {
		b.WriteString("(none)\n\n")
	} else {
		for _, ai := range st.ActionItems {
			status := ai.Status
			if status == "" {
				status = "open"
			}
			fmt.Fprintf(&b, "- [%s] %s: %s → source: %s\n", status, ai.Assignee, ai.Task, ai.ID)
		}
		b.WriteString("\n")
	}

	b.WriteString("### Shared Materials\n\n")
	if len(st.SharedMaterials) == 0 {
		b.WriteString("(none)\n\n")
	} else {
		for _, m := range st.SharedMaterials {
			line := m.Title
			if m.Ref != nil && m.Ref.URL != "" {
				line = fmt.Sprintf("[%s](%s)", m.Title, m.Ref.URL)
			}
			fmt.Fprintf(&b, "- %s → source: %s\n", line, m.ID)
		}
		b.WriteString("\n")
	}

	if len(st.Votes) > 0 {
		b.WriteString("### Votes\n\n")
		for _, v := range st.Votes {
			status := "open"
			if v.Closed {
				status = "closed"
			}
			fmt.Fprintf(&b, "- [%s] %s → %s (source: %s)", status, v.Topic, v.ComputedResult, v.ID)
			if v.ResultMismatch {
				b.WriteString(" ⚠ result_mismatch: projector recomputation is authoritative")
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(st.OutOfScope) > 0 {
		b.WriteString("### Out of Scope\n\n")
		for _, o := range st.OutOfScope {
			if o.Reason != "" {
				fmt.Fprintf(&b, "- %s (%s)\n", o.TargetID, o.Reason)
			} else {
				fmt.Fprintf(&b, "- %s\n", o.TargetID)
			}
		}
		b.WriteString("\n")
	}

	if st.Lease != nil {
		fmt.Fprintf(&b, "### Coordinator\n\n- holder: %s, epoch %d, lease_until %s (source: %s)\n\n",
			st.Lease.Holder, st.Lease.Epoch, st.Lease.LeaseUntil, st.Lease.EventID)
	}
	if len(st.LeaseChain) > 0 {
		b.WriteString("### Coordinator Lease Chain\n\n")
		for _, lease := range st.LeaseChain {
			fmt.Fprintf(&b, "- %s: holder %s, epoch %d, lease_until %s\n",
				lease.EventID, lease.Holder, lease.Epoch, lease.LeaseUntil)
		}
		b.WriteString("\n")
	}

	// Rejections summary (violations are marked, never deleted).
	var rejected []Adjudication
	for _, a := range st.Adjudications {
		if a.Status == StatusRejected || a.Status == StatusLateVote {
			rejected = append(rejected, a)
		}
	}
	if len(rejected) > 0 {
		b.WriteString("### Rejected Events\n\n")
		for _, r := range rejected {
			fmt.Fprintf(&b, "- %s (%s): %s\n", r.EventID, r.Status, r.Reason)
		}
		b.WriteString("\n")
	}

	return b.String()
}
