package statusrail

import "time"

type NoticePriority string

const (
	PriorityNormal      NoticePriority = "normal"
	PriorityCritical    NoticePriority = "critical"
	PriorityDestructive NoticePriority = "destructive"
)

type NoticeTemplate string

const (
	NoticeNeedsConfirm   NoticeTemplate = "needs_confirm"
	NoticeCompleted      NoticeTemplate = "completed"
	NoticeError          NoticeTemplate = "error"
	NoticeReviewPending  NoticeTemplate = "review_pending"
	NoticeAssetValidated NoticeTemplate = "asset_validated"
)

type Notice struct {
	ID           int64          `json:"id"`
	Source       string         `json:"source"`
	Template     NoticeTemplate `json:"template"`
	Subject      string         `json:"subject"`
	Priority     NoticePriority `json:"priority"`
	Text         string         `json:"text"`
	CreatedAt    time.Time      `json:"createdAt"`
	DisplayUntil time.Time      `json:"displayUntil"`
	Locked       bool           `json:"locked"`
}

type NoticeQueue struct {
	nextID      int64
	items       []Notice
	active      *Notice
	lockedCount int
}

func (q *NoticeQueue) Push(source string, template NoticeTemplate, subject string, priority NoticePriority, now time.Time) Notice {
	q.nextID++
	notice := Notice{
		ID:        q.nextID,
		Source:    source,
		Template:  template,
		Subject:   subject,
		Priority:  priority,
		Text:      noticeText(template, subject),
		CreatedAt: now,
		Locked:    true,
	}
	if priority == PriorityCritical || priority == PriorityDestructive {
		q.items = append([]Notice{notice}, q.items...)
	} else {
		q.items = append(q.items, notice)
	}
	q.lockedCount++
	return notice
}

func (q *NoticeQueue) Current(now time.Time) *Notice {
	if q.active != nil {
		return q.active
	}
	if len(q.items) == 0 {
		return nil
	}
	next := q.items[0]
	q.items = q.items[1:]
	next.DisplayUntil = now.Add(10 * time.Second)
	q.active = &next
	return q.active
}

func (q *NoticeQueue) ReleaseDisplay(now time.Time) {
	if q.active != nil && !q.active.DisplayUntil.IsZero() && now.After(q.active.DisplayUntil) {
		q.active = nil
	}
}

func (q *NoticeQueue) Acknowledge() {
	q.active = nil
	q.items = nil
	q.lockedCount = 0
}

func (q *NoticeQueue) LockedCount() int {
	return q.lockedCount
}

func noticeText(template NoticeTemplate, subject string) string {
	if subject == "" {
		subject = "任務"
	}
	switch template {
	case NoticeNeedsConfirm:
		return subject + " 需要您確認，請至 DAG 列查看"
	case NoticeCompleted:
		return subject + " 已完成，請至 DAG 列查看結果"
	case NoticeError:
		return subject + " 發生錯誤，請至 DAG 列處理"
	case NoticeReviewPending:
		return "Review 待處理：" + subject
	case NoticeAssetValidated:
		return subject + " 驗證完成"
	default:
		return RejectTemplate
	}
}
