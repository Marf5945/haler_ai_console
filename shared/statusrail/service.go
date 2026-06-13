package statusrail

import (
	"math/rand"
	"path/filepath"
	"sync"
	"time"
)

type Layer string

const (
	LayerNotice Layer = "L2"
	LayerUser   Layer = "L1"
	LayerIdle   Layer = "L3"
)

type View struct {
	Text        string `json:"text"`
	Layer       Layer  `json:"layer"`
	Degraded    bool   `json:"degraded"`
	LockedCount int    `json:"lockedCount"`
}

type Service struct {
	mu           sync.Mutex
	root         string
	history      *HistoryStore
	snapshots    *SnapshotBuffer
	notices      NoticeQueue
	greetings    []string
	idlePhrases  []string
	currentText  string
	currentLayer Layer
	lastActivity time.Time
	rand         *rand.Rand
}

func NewService(root string, greetings []string) *Service {
	if root == "" {
		root = "."
	}
	if !filepath.IsAbs(root) {
		root, _ = filepath.Abs(root)
	}
	if len(greetings) == 0 {
		greetings = []string{"Hi 主人，今天天氣不錯！"}
	}
	history := NewHistoryStore(root)
	idleFallback := []string{
		"你好，主人。",
		"本大叔在，慢慢來。",
		"先喘口氣也可以。",
		"需要時叫我一聲。",
	}
	idlePhrases := history.LoadPhrases(idleFallback)
	now := time.Now()
	return &Service{
		root:         root,
		history:      history,
		snapshots:    NewSnapshotBuffer(2),
		greetings:    append([]string(nil), greetings...),
		idlePhrases:  idlePhrases,
		currentText:  greetings[0],
		currentLayer: LayerUser,
		lastActivity: now,
		rand:         rand.New(rand.NewSource(now.UnixNano())),
	}
}

func (s *Service) View() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.viewLocked(time.Now())
}

func (s *Service) NextGreeting(current string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastActivity = time.Now()
	if len(s.greetings) == 0 {
		return current
	}
	next := current
	for len(s.greetings) > 1 && next == current {
		next = s.greetings[s.rand.Intn(len(s.greetings))]
	}
	s.currentText = next
	s.currentLayer = LayerUser
	return next
}

func (s *Service) CompanionReply(input string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastActivity = time.Now()
	if IsBlocked(input) {
		s.currentText = RejectTemplate
		s.currentLayer = LayerUser
		return s.currentText
	}
	if reply, err := localCompanionReply(input, s.snapshots.Items()); err == nil {
		s.currentText = reply
	} else {
		s.currentText = s.fallbackPhraseLocked()
	}
	s.currentLayer = LayerUser
	return s.currentText
}

func (s *Service) AddSnapshot(role, text string) {
	s.snapshots.Add(role, text)
}

// ── FIX: CLI→Greeting 同步 ──
// SetText 直接更新顯示文字與活動時間，讓 polling 回傳最新值。
// 用於 CLI 回應同步到人格對話 greeting。
func (s *Service) SetText(text string) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentText = text
	s.currentLayer = LayerUser
	s.lastActivity = time.Now()
	return s.viewLocked(s.lastActivity)
}

func (s *Service) RecordMainInteraction() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastActivity = time.Now()
	return s.viewLocked(s.lastActivity)
}

func (s *Service) AddNotice(source string, template NoticeTemplate, subject string, priority NoticePriority) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.notices.Push(source, template, subject, priority, now)
	return s.viewLocked(now)
}

func (s *Service) AcknowledgeNotices() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.notices.Acknowledge()
	s.lastActivity = now
	return s.viewLocked(now)
}

func (s *Service) viewLocked(now time.Time) View {
	if notice := s.notices.Current(now); notice != nil {
		if now.Before(notice.DisplayUntil) || now.Equal(notice.DisplayUntil) {
			return View{Text: notice.Text, Layer: LayerNotice, Degraded: true, LockedCount: s.notices.LockedCount()}
		}
		s.notices.ReleaseDisplay(now)
	}
	if now.Sub(s.lastActivity) >= 20*time.Minute {
		s.currentText = s.idlePhraseLocked()
		s.currentLayer = LayerIdle
		s.lastActivity = now
	}
	return View{Text: s.currentText, Layer: s.currentLayer, Degraded: true, LockedCount: s.notices.LockedCount()}
}

func (s *Service) idlePhraseLocked() string {
	if len(s.idlePhrases) == 0 {
		return "你好，主人。"
	}
	return s.idlePhrases[s.rand.Intn(len(s.idlePhrases))]
}

func (s *Service) fallbackPhraseLocked() string {
	if len(s.greetings) == 0 {
		return "本大叔在。"
	}
	return s.greetings[s.rand.Intn(len(s.greetings))]
}
