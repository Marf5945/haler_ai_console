package skill_flow

import "time"

// Package skill_flow — 通用多輪互動收集引擎（Step 1：FlowDef 先由 Go 程式碼定義）。
//
// 設計目標：skill 宣告流程（欄位、提問文案、控制詞），app 提供狀態機執行；
// 引擎不認識任何特定 skill，所有 skill 專屬內容（文案、比對、輸出）由
// FlowDef 與 Resolver/呼叫端注入。詳見 REFACTOR_SKILL_FLOW.md。

// Candidate 是 lookup 比對的一個候選。
type Candidate struct {
	Value string // 帶入值（如料號）
	Label string // 顯示文字（如 品名（料號））
	Name  string // 純品名（pick 階段比對用，可空）
}

// ResolveStatus 是 Resolver 的查詢結果狀態。
type ResolveStatus string

const (
	ResolveOK       ResolveStatus = "ok"        // 至少一筆；1 筆自動帶入，多筆反問
	ResolveNone     ResolveStatus = "none"      // 資料有讀到但查無相符
	ResolveNoSource ResolveStatus = "no_source" // 找不到資料來源（如未載入 DB 檔）
)

// Resolver 是 lookup 欄位的外掛點：由呼叫端注入（可含模型精選）。
type Resolver interface {
	Resolve(query string) ([]Candidate, ResolveStatus)
}

// FieldDef 是依序收集的純量欄位。
type FieldDef struct {
	Name      string
	Prompt    string              // 首問
	RePrompt  string              // 空值/無效時再問
	Normalize func(string) string // 可為 nil
	// Ack 收到值後的引導語（含下一步提示）。回空字串則用下一欄 Prompt 或 List.Prompt。
	Ack func(value string) string
}

// ListDef 是逐項收集（lookup＋數量）的迴圈定義。所有文案由 skill 宣告。
type ListDef struct {
	MinItems       int
	MinItemsPrompt string // 未達 MinItems 就喊結束時
	Prompt         string // 進入清單收集的首問（無前欄 Ack 時使用）
	NextPrompt     string // more 控制詞後的提問
	DoneWords      []string
	MoreWords      []string

	QtyPrompt   string // 缺數量時補問
	QtyRePrompt string // 數量解析失敗時

	NoSourceText string // Resolver 回 no_source
	MatchedText  string // 單筆自動帶入，含 %s（label）
	NoMatchText  string // 查無、照原樣帶入，含 %s（原輸入）
	PickedText   string // pick 選定後，含 %s（label）

	PickPrompt        string // 多筆候選反問，含 %s（查詢字串）
	PickPromptNoQuery string // 重問時（無查詢字串）
	PickRePrompt      string // 沒看懂選哪個

	ReviewHeader string // 含 %d（項數）
	ReviewTail   string // 「要不要補？…」

	FixNoHitText      string // 修正定位失敗
	FixPickPrompt     string // 修正多筆符合，反問哪一項
	FixPickRePrompt   string // 修正反問沒看懂，含 %s（可選編號）
	FixQtyPrompt      string // 修正缺新數量，含 %d %s（清單編號、值）
	FixDoneText       string // 修正完成，含 %d %s %s %s（編號、值、舊、新）
	FixOutOfRangeText string // 指定「第N項」超界，含 %d %d（總數、N）
}

// FlowDef 是一個 skill 的完整流程定義。
type FlowDef struct {
	SkillName   string
	TTL         time.Duration
	CancelWords []string
	CancelText  string

	Fields []FieldDef // 依序收集的純量欄位
	List   *ListDef   // 之後的逐項收集；nil 表示收完欄位即完成
}

// Outcome 是引擎處理一回合的結果，由呼叫端轉成 UI 回應。
type Outcome struct {
	Ask       string // 非空 → 反問使用者（NeedsUser）
	Text      string // 非反問的告知文字（目前用於取消）
	Complete  bool   // 收集完成 → 呼叫端執行 on_complete 並清狀態
	Cancelled bool   // 使用者取消 → 呼叫端清狀態
}
