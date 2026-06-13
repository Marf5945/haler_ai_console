// lifecycle.go — Skill 生命週期狀態與「應然」步驟鏈定義（TASK 31 / Phase 1.1）。
// 重點：expected_chain 的 ㄅㄔㄇㄖ step code 與 ㄌ 只以「結構化欄位」存在，
// 永不被當成不可信自由文字送回 LLM（不變式 6）。
package skill_step

// Skill lifecycle 狀態（取代單一 Enabled bool 的不足）。
const (
	LifecycleDraftCandidate = "draft_candidate" // 系統草稿，未經使用者確認，不顯示為工具
	LifecyclePending        = "pending_skill"   // 已保存，可見/可查/可候選，但不可自動執行
	LifecycleEnabled        = "enabled_skill"   // 使用者確認權限後，可正常路由與執行
	LifecycleDisabled       = "disabled_skill"  // 使用者停用，不自動候選
)

// Lifecycle 控制 skill 的可見性與可執行性。
type Lifecycle struct {
	Status           string `json:"status"`                       // 見上方常數
	VisibleInToolbar bool   `json:"visible_in_toolbar"`           // 候選區/工具列是否顯示
	RouteAsCandidate bool   `json:"route_as_candidate"`           // 可否被路由為候選
	AutoExecute      bool   `json:"auto_execute"`                 // 可否免確認自動執行
	CreatedFromTrace string `json:"created_from_trace,omitempty"` // 由哪個 DAG run 長出
	UserConfirmed    bool   `json:"user_confirmed"`               // 使用者是否已確認
}

// ExpectedStep 是一步的「應然」定義；LLM 不吐這些欄位，由作者/草稿產生。
type ExpectedStep struct {
	Action      string `json:"action"`
	Target      string `json:"target"`
	Next        string `json:"next,omitempty"` // 省略預設「待命」
	Code        string `json:"code"`           // ㄅ|ㄔ|ㄇ|ㄖ，獨立欄位（禁止 concat）
	Requirement string `json:"requirement"`    // OP|RE
}

// ExpectedChain 是 skill 的應然步驟序列；drift 拿它當比對基準。
type ExpectedChain struct {
	Schema   string         `json:"schema"`    // 固定 "skill_chain.v1"
	MaxSteps int            `json:"max_steps"` // 預設 16，超過僅低風險提示
	Steps    []ExpectedStep `json:"steps"`
}

// 合法的 step code 與 requirement 集合（草稿驗證用，Phase 1.5）。
var validStepCodes = map[string]bool{"ㄅ": true, "ㄔ": true, "ㄇ": true, "ㄖ": true}
var validRequirements = map[string]bool{"OP": true, "RE": true}

// IsValidStepCode 回報 code 是否為允許的 ㄅㄔㄇㄖ 之一。
func IsValidStepCode(code string) bool { return validStepCodes[code] }

// IsValidRequirement 回報 requirement 是否為 OP/RE。
func IsValidRequirement(req string) bool { return validRequirements[req] }

// riskLow 回報 manifest 的首個 risk tag 是否為 low（缺省視為 low）。
func riskLow(m *SkillManifest) bool {
	if len(m.Tags.RiskTag) == 0 {
		return true
	}
	return m.Tags.RiskTag[0] == "low"
}

// DefaultLifecycle 給缺 lifecycle 的 manifest（含所有 builtin）一個安全預設。
// auto_execute 只在「低風險 + 不寫檔 + 不執行」時預設開啟；
// 其餘（如 workspace_write / medium）預設 false，改走 riskgrant 第一次執行確認。
func DefaultLifecycle(m *SkillManifest) Lifecycle {
	auto := riskLow(m) &&
		m.Permissions.Filesystem != "workspace_write" &&
		m.Permissions.Execution == "none"
	return Lifecycle{
		Status:           LifecycleEnabled, // 缺 lifecycle 的視為已啟用，否則 builtin 會從工具列消失
		VisibleInToolbar: true,
		RouteAsCandidate: true,
		AutoExecute:      auto,
		UserConfirmed:    true, // builtin 視為內建已信任
	}
}

// EnsureLifecycle 在 manifest 缺 lifecycle 時補上安全預設，回傳是否有補。
func EnsureLifecycle(m *SkillManifest) bool {
	if m.Lifecycle != nil {
		return false
	}
	def := DefaultLifecycle(m)
	m.Lifecycle = &def
	return true
}
