// conversation/intent.go — 意圖分類與動作路由。
// 解析 LLM 回覆，判斷是一般對話還是動作標籤，並路由到對應的 sub 或 tool。
package conversation

import "strings"

// ──────────────────────────────────────────────
// 意圖分類資料結構
// ──────────────────────────────────────────────

// IntentResult 儲存 LLM 回覆的意圖解析結果。
type IntentResult struct {
	IsAction         bool   // true 表示識別到動作標籤
	ActionTag        string // 匹配到的動作標籤（IsAction=true 時有值）
	ConversationReply string // 一般對話回覆（IsAction=false 時有值）
}

// ──────────────────────────────────────────────
// 意圖分類函式
// ──────────────────────────────────────────────

// ClassifyIntent 掃描 LLM 回覆，判斷是否命中可用的動作標籤。
// 比對順序：精確全字比對 → 包含比對（模糊）。
// 未命中任何標籤時，IsAction=false，ConversationReply 為完整回覆。
func ClassifyIntent(llmResponse string, availableTags []string) IntentResult {
	trimmed := strings.TrimSpace(llmResponse)

	// 第一輪：精確比對（全字相同，不分大小寫）
	lower := strings.ToLower(trimmed)
	for _, tag := range availableTags {
		if strings.ToLower(tag) == lower {
			return IntentResult{
				IsAction:  true,
				ActionTag: tag,
			}
		}
	}

	// 第二輪：包含比對（回覆中含有標籤字串）
	for _, tag := range availableTags {
		if strings.Contains(lower, strings.ToLower(tag)) {
			return IntentResult{
				IsAction:  true,
				ActionTag: tag,
			}
		}
	}

	// 無匹配，視為一般對話回覆
	return IntentResult{
		IsAction:         false,
		ConversationReply: llmResponse,
	}
}

// ──────────────────────────────────────────────
// 標籤收集
// ──────────────────────────────────────────────

// SubRegistryEntry 代表子系統登錄表中的一個條目。
type SubRegistryEntry struct {
	ID         string   // 子系統唯一識別碼
	Name       string   // 子系統名稱
	Triggers   []string // 觸發關鍵字
	ActionTags []string // 動作語意標籤
	ToolsUsed  []string // 使用的工具 ID 清單
}

// ToolEntry 代表工具登錄表中的一個條目。
type ToolEntry struct {
	ID   string   // 工具唯一識別碼
	Name string   // 工具名稱
	Tags []string // 工具語意標籤
}

// CollectActionTags 從子系統登錄表與工具列表收集所有唯一的動作標籤。
// 去除重複，保持首次出現順序。
func CollectActionTags(subRegistry []SubRegistryEntry, toolDB []ToolEntry) []string {
	seen := make(map[string]struct{})
	var result []string

	// 收集子系統的 ActionTags 與 Triggers
	for _, sub := range subRegistry {
		for _, tag := range sub.ActionTags {
			if _, ok := seen[tag]; !ok {
				seen[tag] = struct{}{}
				result = append(result, tag)
			}
		}
		for _, trigger := range sub.Triggers {
			if _, ok := seen[trigger]; !ok {
				seen[trigger] = struct{}{}
				result = append(result, trigger)
			}
		}
	}

	// 收集工具的 Tags
	for _, tool := range toolDB {
		for _, tag := range tool.Tags {
			if _, ok := seen[tag]; !ok {
				seen[tag] = struct{}{}
				result = append(result, tag)
			}
		}
	}

	return result
}

// ──────────────────────────────────────────────
// 動作路由
// ──────────────────────────────────────────────

// RouteResult 儲存路由比對結果。
type RouteResult struct {
	MatchType  string // "sub" / "tool" / "none"
	TargetID   string // 命中的子系統或工具 ID
	TargetName string // 命中的名稱
}

// RouteAction 依標籤字串比對子系統（優先）再比對工具，回傳第一個命中結果。
// 比對方式：精確全字比對 → 包含比對。
func RouteAction(tag string, subs []SubRegistryEntry, tools []ToolEntry) RouteResult {
	lowerTag := strings.ToLower(tag)

	// 優先：比對子系統的 Triggers 與 ActionTags
	for _, sub := range subs {
		for _, t := range append(sub.Triggers, sub.ActionTags...) {
			if strings.ToLower(t) == lowerTag {
				return RouteResult{MatchType: "sub", TargetID: sub.ID, TargetName: sub.Name}
			}
		}
	}
	// 子系統模糊比對
	for _, sub := range subs {
		for _, t := range append(sub.Triggers, sub.ActionTags...) {
			if strings.Contains(lowerTag, strings.ToLower(t)) || strings.Contains(strings.ToLower(t), lowerTag) {
				return RouteResult{MatchType: "sub", TargetID: sub.ID, TargetName: sub.Name}
			}
		}
	}

	// 次選：比對工具 Tags
	for _, tool := range tools {
		for _, t := range tool.Tags {
			if strings.ToLower(t) == lowerTag {
				return RouteResult{MatchType: "tool", TargetID: tool.ID, TargetName: tool.Name}
			}
		}
	}
	// 工具模糊比對
	for _, tool := range tools {
		for _, t := range tool.Tags {
			if strings.Contains(lowerTag, strings.ToLower(t)) || strings.Contains(strings.ToLower(t), lowerTag) {
				return RouteResult{MatchType: "tool", TargetID: tool.ID, TargetName: tool.Name}
			}
		}
	}

	// 無匹配
	return RouteResult{MatchType: "none"}
}
