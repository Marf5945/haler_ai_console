// source_trust/ranking.go — USER_GENERATED 排序補償（§9.7）。
// 品質信號可提升排序分數，但 label 必須維持 USER_GENERATED。
package source_trust

// ──────────────────────────────────────────────
// 品質信號結構
// ──────────────────────────────────────────────

// QualitySignal 描述 UGC 來源的品質指標。
type QualitySignal struct {
	UpvoteCount        int  `json:"upvote_count"`
	AcceptedAnswer     bool `json:"accepted_answer"`
	AnswerAgeDays      int  `json:"answer_age_days"`
	AuthorReputation   int  `json:"author_reputation"`
	CitationCount      int  `json:"citation_count"`
	MaintainerAnswer   bool `json:"maintainer_answer"`
	OfficialStaffReply bool `json:"official_staff_reply"`
	SurvivalTimeDays   int  `json:"survival_time_days"`
	CrossRefCount      int  `json:"cross_reference_count"`
}

// ──────────────────────────────────────────────
// 排序補償計算
// ──────────────────────────────────────────────

// AdjustRanking 根據品質信號調整排序分數。
// 硬規則（§9.7）：
//   - 分數可提升，不可使 label 脫離 USER_GENERATED
//   - USER_GENERATED 不得變為 VERIFIED_AUTHORITY
//   - USER_GENERATED 不得產生 AUTH_OK:true
func AdjustRanking(baseScore int, signals QualitySignal) int {
	score := baseScore

	// 投票數加分（每 10 票 +1，上限 +10）
	bonus := signals.UpvoteCount / 10
	if bonus > 10 {
		bonus = 10
	}
	score += bonus

	// 被接受的回答 +5
	if signals.AcceptedAnswer {
		score += 5
	}

	// 維護者/官方回覆 +3
	if signals.MaintainerAnswer {
		score += 3
	}
	if signals.OfficialStaffReply {
		score += 3
	}

	// 引用數加分（每次 +1，上限 +5）
	citBonus := signals.CitationCount
	if citBonus > 5 {
		citBonus = 5
	}
	score += citBonus

	// 交叉引用加分（每次 +1，上限 +3）
	crossBonus := signals.CrossRefCount
	if crossBonus > 3 {
		crossBonus = 3
	}
	score += crossBonus

	// 存活時間加分（超過 365 天 +2，超過 180 天 +1）
	if signals.SurvivalTimeDays > 365 {
		score += 2
	} else if signals.SurvivalTimeDays > 180 {
		score += 1
	}

	// 作者聲譽加分（每 1000 +1，上限 +5）
	repBonus := signals.AuthorReputation / 1000
	if repBonus > 5 {
		repBonus = 5
	}
	score += repBonus

	// 分數上限 100，下限 0
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}
