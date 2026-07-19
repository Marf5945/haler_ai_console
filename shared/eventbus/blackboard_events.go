// blackboard_events.go — 黑板（多人會議）相關事件常數。
// 獨立檔案避免修改 eventbus.go 主檔，降低合併衝突風險。
package eventbus

const (
	// EventBlackboardStateUpdated 在同步輪完成、Meeting State 重新投影後發送。
	// payload: blackboard 狀態摘要（projected_until、各區塊數量、落後則數）。
	EventBlackboardStateUpdated = "blackboard:state_updated"

	// EventBlackboardIntegrityWarning 在偵測到共享文件缺少鏡像中已知事件時發送。
	// payload: {"missing_ids": [...]}。不自動還原，只提醒（spec §12）。
	EventBlackboardIntegrityWarning = "blackboard:integrity_warning"

	// EventBlackboardDeadLetter 在同步輪發現無法解析事件時發送。
	// payload: {"count": N}。壞資料只寫本機 jsonl，不回寫共享 log（spec §5）。
	EventBlackboardDeadLetter = "blackboard:dead_letter"

	// EventBlackboardRequest 在投影中出現點名本機 agent 的 request 事件時發送。
	// 這是 agent 唯一可被觸發行動的入口（spec §7 鐵律）。
	EventBlackboardRequest = "blackboard:request"
)
