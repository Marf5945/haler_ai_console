// ambiguity.go 實作模糊狀態判定器 AmbiguityDetector。
// 它是路由管線的最後一關：接收 Scorer 計算完的候選清單，
// 依照五條優先規則，輸出最終的 ResolveStatus。
//
// 設計哲學：
//   - 安全優先：高/緊急風險一律進入人工審查，無論分數多高
//   - 確定性優先：單一低風險且分數達標時才自動注入，不猜測
//   - 透明性優先：多個候選時不靜默選一個，而是讓 CLI 或使用者知情後選擇
//
// 五條規則的優先順序是固定的，改動時必須同步更新 APPENDIX 文件。
package skill_step

// AmbiguityDetector 依候選清單與風險等級，判定此次路由的最終狀態。
// 目前為無狀態設計，未來若需要可配置的風險策略（如組織層級的 policy override），
// 可在此加入設定欄位，不影響現有的 Decide 介面。
type AmbiguityDetector struct{}

// NewAmbiguityDetector 建立 AmbiguityDetector，目前不需要任何參數。
func NewAmbiguityDetector() *AmbiguityDetector { return &AmbiguityDetector{} }

// highRiskValues 列舉所有需要強制進入人工審查的風險等級。
// 使用 map[string]bool 而非 switch，方便未來新增等級（如 "extreme"）時不需修改 Decide 邏輯。
var highRiskValues = map[string]bool{
	"high":     true, // 高風險：需要使用者明確確認
	"critical": true, // 緊急風險：嚴格要求人工審查，不得自動注入
}

// Decide 依照五條優先規則，從 candidates 中判定最終 ResolveStatus。
//
// 參數：
//   - candidates：所有由 Scorer 計算出分數 > 0 的候選清單（可能為空）
//   - minScore：固定自動候選信心門檻，目前產品規格鎖定為 0.8
//
// 五條規則（依優先順序）：
//  1. 任何超過可行門檻（threshold）且風險為 high/critical 的候選 → needs_user_review
//  2. 沒有任何可行候選（score < threshold）→ rejected
//  3. 恰好一個低風險候選，且分數 ≥ minScore → auto_selected
//  4. 多個低風險候選（都超過 threshold）→ needs_cli_candidate
//  5. 其他情況（單個候選但分數不足 minScore）→ needs_user_review
func (d *AmbiguityDetector) Decide(candidates []Candidate, minScore float64) ResolveStatus {
	// 過濾出分數足夠的可行候選
	var viable []Candidate
	for _, c := range candidates {
		if c.Score >= minScore {
			viable = append(viable, c)
		}
	}

	// 規則一：任何可行候選帶有高/緊急風險 → 強制人工審查
	// 這條規則最優先，確保高風險 skill 不會被任何情況自動注入
	for _, c := range viable {
		if highRiskValues[c.Risk] {
			return StatusNeedsReview
		}
	}

	// 規則二：沒有任何可行候選 → 路由失敗，拒絕
	if len(viable) == 0 {
		return StatusRejected
	}

	// 規則三：恰好一個低風險候選，且分數達到自動選取門檻 → 自動注入
	// 同時滿足：唯一性（排除模糊）+ 低風險 + 分數達標，才能自動注入
	if len(viable) == 1 && viable[0].Score >= minScore {
		return StatusAutoSelected
	}

	// 規則四：多個低風險候選 → 交由 CLI 做二次選擇
	// 不靜默選第一個，保持選擇透明性
	if len(viable) > 1 {
		return StatusNeedsCLI
	}

	// 規則五：其他情況需要使用者在 Review Card 確認。
	return StatusNeedsReview
}
