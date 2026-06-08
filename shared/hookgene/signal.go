package hookgene

import "time"

// HookCode 是「行動基因」字母（§3.1.5.18.2）。
// 注意：純內部評分用，永不進入 LLM 回覆，與 ㄌ 分隔符 / §31 印章分層。
type HookCode rune

const (
	HookList    HookCode = 'ㄅ' // 列表：內部處理（整理/分類/比對/展開/過濾），不輸入不輸出
	HookInput   HookCode = 'ㄖ' // 輸入：資料進入系統
	HookOutput  HookCode = 'ㄔ' // 輸出：資料離開系統 / 對外輸出
	HookStandby HookCode = 'ㄇ' // 待命：等待；亦作 16 格補位
)

// SignalType 是執行層發出的 5 種 runtime 訊號（§3.1.5.18.2）。
// 整合時請在 skill execution / replay executor / Go program runner 三處打點發出。
type SignalType string

const (
	SignalDataEntered   SignalType = "data_entered_system"           // → ㄖ
	SignalDataProcessed SignalType = "data_transferred_or_processed" // → ㄅ
	SignalDataLeft      SignalType = "data_left_boundary"            // → ㄔ 或 ㄅ（看是否跨邊界）
	SignalPaused        SignalType = "skill_paused_or_waiting"       // → ㄇ
	SignalCompleted     SignalType = "skill_completed"               // 結束本次 gene 並計算
)

// Signal 是一筆 runtime 訊號。Recorder 只看行為事件，不讀資料內容（§3.1.5.18.3）。
type Signal struct {
	SkillID      string
	InvocationID string
	Type         SignalType
	// CrossedBoundary 由執行層在 data_left 時判定：資料是否離開本 skill 控制邊界。
	// 進入 使用者/外部工具/檔案/網路/replay executor/UI = true → ㄔ；仍在 skill 內部整理 = false → ㄅ。
	CrossedBoundary bool
	At              time.Time
}

// HookFor 把訊號對映成 hook code。ok=false 代表此訊號不直接產生 hook（如 completed）。
func HookFor(s Signal) (HookCode, bool) {
	switch s.Type {
	case SignalDataEntered:
		return HookInput, true
	case SignalDataProcessed:
		return HookList, true
	case SignalDataLeft:
		// ㄔ 邊界硬規則：只有真正離開 skill 控制邊界才算輸出，否則視為內部處理 ㄅ。
		if s.CrossedBoundary {
			return HookOutput, true
		}
		return HookList, true
	case SignalPaused:
		return HookStandby, true
	default:
		return 0, false
	}
}
