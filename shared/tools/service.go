// Package tools implements the v3.3.2 #44 Tool Visibility / Unavailable State
// 合約，包含斷線排序機制與 reauth 攔截。
//
// 核心規則：
//   - 已安裝工具只要出現在工具欄，就代表 main agent 可候選使用。
//   - 禁止「加入使用工具」按鈕或二次加入流程。
//   - 工具斷線時保持可見，icon 加黑線打叉覆蓋，排到後方。
//   - 系統產生 120 分鐘暫存排序紀錄；恢復時回到原位，過期則永久化。
//   - 執行前攔截，提示 reauth / reconnect。
package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// Tool 結構
// ──────────────────────────────────────────────

// Tool 代表工具欄中的一個工具。
type Tool struct {
	ID         string   `json:"id"`
	Icon       string   `json:"icon"`
	Title      string   `json:"title"`
	Detail     string   `json:"detail"`
	Kind       string   `json:"kind"` // panel / connector / external / mcp / skill
	Target     string   `json:"target"`
	Enabled    bool     `json:"enabled"`
	ActionTags []string `json:"action_tags,omitempty"`

	// ── #44 Unavailable 狀態 ──
	Available         bool   `json:"available"`    // false = 斷線，icon 加黑線打叉
	NeedsReauth       bool   `json:"needs_reauth"` // true = 執行前攔截
	UnavailableReason string `json:"unavailable_reason,omitempty"`
}

// ActionResult 是工具啟動的結果。
type ActionResult struct {
	ToolID  string `json:"toolId"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Kind    string `json:"kind"`
	Target  string `json:"target"`
}

// ──────────────────────────────────────────────
// 斷線排序暫存紀錄（120 分鐘有效）
// ──────────────────────────────────────────────

// disconnectRecord 記錄工具斷線前的原始排序位置。
type disconnectRecord struct {
	ToolID       string    `json:"tool_id"`
	OriginalRank int       `json:"original_rank"` // 斷線前在 tools slice 中的 index
	DisconnectAt time.Time `json:"disconnect_at"`
	ExpiresAt    time.Time `json:"expires_at"` // disconnect_at + 120 分鐘
}

const disconnectTTL = 120 * time.Minute

// ──────────────────────────────────────────────
// Service 主體
// ──────────────────────────────────────────────

// Service 管理工具欄的工具清單、可見性與斷線排序。
type Service struct {
	mu              sync.Mutex
	tools           []Tool
	disconnectCache map[string]*disconnectRecord // toolID → 暫存紀錄
	disconnectFile  string                       // 暫存紀錄檔路徑
}

// NewService 建立工具服務，載入預設工具。
func NewService() *Service {
	return &Service{
		tools: []Tool{
			{ID: "tool-entrance", Icon: "⌕", Title: "使用工具", Detail: "開啟工具入口", Kind: "panel", Target: "tools", Enabled: true, Available: true},
			{ID: "doc-entrance", Icon: "▤", Title: "引用文件", Detail: "文件與素材入口", Kind: "panel", Target: "documents", Enabled: true, Available: true},
			{ID: "external-link", Icon: "↗", Title: "外部連結", Detail: "保留給安全開啟器", Kind: "external", Target: "", Enabled: false, Available: true},
			{ID: "gmail", Icon: "✉", Title: "Gmail", Detail: "等待 connector 接入", Kind: "connector", Target: "gmail", Enabled: false, Available: true},
		},
		disconnectCache: make(map[string]*disconnectRecord),
	}
}

// NewServiceWithDataRoot 建立含資料路徑的工具服務（支援暫存檔持久化）。
func NewServiceWithDataRoot(dataRoot string) *Service {
	svc := NewService()
	svc.disconnectFile = filepath.Join(dataRoot, "data", "tools", "disconnect_order_cache.json")
	_ = svc.loadDisconnectCache()
	return svc
}

// ──────────────────────────────────────────────
// 工具清單操作
// ──────────────────────────────────────────────

// List 回傳排序後的工具清單。
// 排序規則：可用工具在前、斷線工具在後（保持各自相對順序）。
func (s *Service) List() []Tool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireDisconnectRecords()
	return s.sortedToolsLocked()
}

// AddTool 新增一個工具到工具欄（安裝完成後呼叫）。
func (s *Service) AddTool(tool Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 去重
	for _, t := range s.tools {
		if t.ID == tool.ID {
			return
		}
	}
	tool.Available = true
	s.tools = append(s.tools, tool)
}

// AddActionTag records a reviewed action tag on the selected tool.
func (s *Service) AddActionTag(toolID, tag string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return false
	}
	for i := range s.tools {
		if s.tools[i].ID != toolID {
			continue
		}
		for _, existing := range s.tools[i].ActionTags {
			if existing == tag {
				return true
			}
		}
		s.tools[i].ActionTags = append(s.tools[i].ActionTags, tag)
		return true
	}
	return false
}

// ActionTags returns all unique tool-provided tags for prompt synthesis.
func (s *Service) ActionTags() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := make(map[string]bool)
	var tags []string
	for _, tool := range s.tools {
		for _, tag := range tool.ActionTags {
			tag = strings.TrimSpace(tag)
			if tag == "" || seen[tag] {
				continue
			}
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	return tags
}

// ──────────────────────────────────────────────
// #44 斷線 / 恢復 / reauth 攔截
// ──────────────────────────────────────────────

// MarkUnavailable 標記工具為斷線狀態。
// 記錄原始排序位置到暫存紀錄（120 分鐘有效）。
func (s *Service) MarkUnavailable(toolID, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tools {
		if t.ID == toolID {
			s.tools[i].Available = false
			s.tools[i].NeedsReauth = true
			s.tools[i].UnavailableReason = reason

			// 只在第一次斷線時記錄原始位置
			if _, exists := s.disconnectCache[toolID]; !exists {
				now := time.Now()
				s.disconnectCache[toolID] = &disconnectRecord{
					ToolID:       toolID,
					OriginalRank: i,
					DisconnectAt: now,
					ExpiresAt:    now.Add(disconnectTTL),
				}
			}
			break
		}
	}
	_ = s.saveDisconnectCache()
}

// MarkAvailable 標記工具恢復連線。
// 若在 120 分鐘內恢復，工具回到原始排序位置。
func (s *Service) MarkAvailable(toolID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tools {
		if t.ID == toolID {
			s.tools[i].Available = true
			s.tools[i].NeedsReauth = false
			s.tools[i].UnavailableReason = ""
			break
		}
	}

	// 檢查暫存紀錄：120 分鐘內恢復則回到原位
	if record, exists := s.disconnectCache[toolID]; exists {
		if time.Now().Before(record.ExpiresAt) {
			s.restoreOriginalRank(toolID, record.OriginalRank)
		}
		delete(s.disconnectCache, toolID)
		_ = s.saveDisconnectCache()
	}
}

// Activate 啟動工具。
// #44 reauth 攔截：若工具需要 reauth，攔截並回傳提示。
func (s *Service) Activate(id string) ActionResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, tool := range s.tools {
		if tool.ID != id {
			continue
		}
		// ── reauth 攔截：斷線工具不可執行 ──
		if tool.NeedsReauth || !tool.Available {
			reason := tool.UnavailableReason
			if reason == "" {
				reason = "外部連線已中斷"
			}
			return ActionResult{
				ToolID:  id,
				OK:      false,
				Message: tool.Title + " 需要重新連線：" + reason,
				Kind:    tool.Kind,
				Target:  tool.Target,
			}
		}
		if !tool.Enabled {
			return ActionResult{
				ToolID:  id,
				OK:      false,
				Message: tool.Title + " 尚未接入正式 binding",
				Kind:    tool.Kind,
				Target:  tool.Target,
			}
		}
		return ActionResult{
			ToolID:  id,
			OK:      true,
			Message: tool.Title + " 已由 Wails binding 接手",
			Kind:    tool.Kind,
			Target:  tool.Target,
		}
	}
	return ActionResult{ToolID: id, OK: false, Message: "找不到工具"}
}

// ──────────────────────────────────────────────
// 排序邏輯
// ──────────────────────────────────────────────

// sortedToolsLocked 回傳排序後的工具清單副本。
// 可用工具在前、斷線工具在後，各自保持原始相對順序。
func (s *Service) sortedToolsLocked() []Tool {
	result := make([]Tool, len(s.tools))
	copy(result, s.tools)

	sort.SliceStable(result, func(i, j int) bool {
		// 可用工具排在前面
		if result[i].Available != result[j].Available {
			return result[i].Available
		}
		return false // 保持相對順序
	})
	return result
}

// restoreOriginalRank 將工具移回原始位置（120 分鐘內恢復時）。
func (s *Service) restoreOriginalRank(toolID string, originalRank int) {
	// 找到工具目前位置
	currentIdx := -1
	for i, t := range s.tools {
		if t.ID == toolID {
			currentIdx = i
			break
		}
	}
	if currentIdx < 0 || currentIdx == originalRank {
		return
	}

	// 移除工具
	tool := s.tools[currentIdx]
	s.tools = append(s.tools[:currentIdx], s.tools[currentIdx+1:]...)

	// 插回原始位置
	if originalRank > len(s.tools) {
		originalRank = len(s.tools)
	}
	s.tools = append(s.tools[:originalRank], append([]Tool{tool}, s.tools[originalRank:]...)...)
}

// expireDisconnectRecords 清除已過期的暫存紀錄。
func (s *Service) expireDisconnectRecords() {
	now := time.Now()
	changed := false
	for id, record := range s.disconnectCache {
		if now.After(record.ExpiresAt) {
			delete(s.disconnectCache, id)
			changed = true
		}
	}
	if changed {
		_ = s.saveDisconnectCache()
	}
}

// ──────────────────────────────────────────────
// 暫存紀錄檔持久化
// ──────────────────────────────────────────────
// 注意：此紀錄屬 UI 暫態，不寫入 append-only audit log。

func (s *Service) loadDisconnectCache() error {
	if s.disconnectFile == "" {
		return nil
	}
	data, err := os.ReadFile(s.disconnectFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var records []*disconnectRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}
	now := time.Now()
	for _, r := range records {
		if now.Before(r.ExpiresAt) {
			s.disconnectCache[r.ToolID] = r
		}
	}
	return nil
}

func (s *Service) saveDisconnectCache() error {
	if s.disconnectFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.disconnectFile), 0o700); err != nil {
		return err
	}
	var records []*disconnectRecord
	for _, r := range s.disconnectCache {
		records = append(records, r)
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.disconnectFile, data, 0o600)
}
