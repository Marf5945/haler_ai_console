package skill_flow

import (
	"sync"
	"time"
)

// Phase 是流程目前等待的輸入種類。
type Phase string

const (
	PhaseField   Phase = "field"    // 等第 FieldIdx 個純量欄位
	PhaseCollect Phase = "collect"  // 等清單項目描述或結束詞
	PhasePick    Phase = "pick"     // 等使用者從 lookup 候選中選一個
	PhaseQty     Phase = "qty"      // 等數量（新項目補數量，或修正的新數量）
	PhaseFixPick Phase = "fix_pick" // 修正多筆符合時等使用者選哪一項
)

// Item 是清單收集到的一筆項目。
type Item struct {
	Value string // 帶入值（如料號；查無時為原輸入）
	Label string // 顯示文字
	Qty   string
	Note  string
}

// State 是一個 session 進行中的流程狀態。
type State struct {
	Phase    Phase
	FieldIdx int
	Values   map[string]string
	Items    []Item

	// pick / qty 階段暫存：
	Candidates   []Candidate
	PendingValue string
	PendingLabel string
	PendingQty   string

	// 修正（fix）暫存：
	FixIndexes []int  // 多筆符合時等使用者選（1-based 清單編號）
	FixIndex   int    // 已鎖定要改的清單編號（>0 時 qty 階段改為更新）
	FixQty     string // 修正語句裡已先講的新數量

	ExpiresAt time.Time
}

// Store 是 sessionID → State 的 TTL 存放區。
type Store struct {
	mu  sync.Mutex
	m   map[string]*State
	ttl time.Duration
}

func NewStore(ttl time.Duration) *Store {
	return &Store{m: map[string]*State{}, ttl: ttl}
}

// Get 取出未過期的狀態；過期即刪除並回 false。
func (s *Store) Get(id string) (*State, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[id]
	if !ok {
		return nil, false
	}
	if time.Now().After(st.ExpiresAt) {
		delete(s.m, id)
		return nil, false
	}
	return st, true
}

// Put 存入並刷新 TTL。
func (s *Store) Put(id string, st *State) {
	st.ExpiresAt = time.Now().Add(s.ttl)
	s.mu.Lock()
	s.m[id] = st
	s.mu.Unlock()
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	delete(s.m, id)
	s.mu.Unlock()
}
