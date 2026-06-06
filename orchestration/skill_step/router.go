// router.go 實作核心路由邏輯：
// 接收解析完成的 ActionTarget，遍歷所有已歸檔的 skill manifest，
// 經過評分（Scorer）與模糊判定（AmbiguityDetector）後，輸出 ResolveResult。
//
// 路由生命週期：
//  1. app.go 呼叫 Router.Resolve(at, sessionID)
//  2. Router 從 ArchiveService.ListArchived() 取得所有 manifest
//  3. Scorer 對每個 manifest 計算分數，分數 > 0 的進入候選清單
//  4. AmbiguityDetector 依候選清單與風險判定最終狀態
//  5. 結果回傳給 app.go，由 Wails binding 序列化給前端
package skill_step

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ResolveStatus 表示一次 Resolve 呼叫的最終判定結果。
// 前端 UI 依這個值決定顯示哪種卡片（自動注入 / CLI 候選 / 人工審查 / 拒絕）。
type ResolveStatus string

const (
	// StatusAutoSelected：唯一低風險候選且分數達門檻，直接自動注入，不需使用者確認。
	StatusAutoSelected ResolveStatus = "auto_selected"
	// StatusNeedsCLI：多個低風險候選，讓 CLI 自行選擇，不直接注入。
	StatusNeedsCLI ResolveStatus = "needs_cli_candidate"
	// StatusNeedsReview：高/緊急風險，或分數不足，須由使用者在 Review Card 確認。
	StatusNeedsReview ResolveStatus = "needs_user_review"
	// StatusRejected：沒有任何分數足夠的候選，本次請求無法路由到任何 skill。
	StatusRejected ResolveStatus = "rejected"
)

// Candidate 代表一個通過最低分數門檻的 skill 候選。
// Scorer 計算 Score 後，AmbiguityDetector 依 Candidate 清單做最終判定。
//
// 欄位說明：
//   - SkillID：對應到 skill_manifest.json 中的 skill_id
//   - Score：Scorer 計算出的相關性分數，範圍 [0.0, 1.0]
//   - Risk：從 manifest.Tags.RiskTag[0] 取出，預設 "low"
//   - Reason：人類可讀的評分理由，供 UI 顯示及 audit log 記錄使用
type Candidate struct {
	SkillID string  // skill 的唯一識別碼
	Score   float64 // 相關性分數，由 Scorer 計算
	Risk    string  // 風險等級：low / medium / high / critical
	Reason  string  // 評分理由，供 UI 與 audit 使用
}

// ResolveResult 是 Router.Resolve 的完整回傳值。
// app.go 的 Wails binding 會將這個結構直接序列化成 JSON 傳給前端。
//
// 欄位說明：
//   - ResolveID：每次 Resolve 產生的唯一識別碼（UnixNano），供追蹤與日誌使用
//   - Status：最終判定狀態，決定前端要顯示哪種 UI 卡片
//   - SelectedSkillID：僅在 StatusAutoSelected 時填入；其他狀態為空字串
//   - Candidates：所有通過門檻的候選，供前端或 CLI 選擇
//   - ReviewRequired：快捷布林值，等同 Status == StatusNeedsReview
type ResolveResult struct {
	ResolveID       string        // 本次解析的唯一 ID
	SessionID       string        // Resolve 所屬 session，供 BuildSkillContext/audit 對齊
	Status          ResolveStatus // 最終路由狀態
	SelectedSkillID string        // 自動選中的 skill ID（僅 auto_selected 時有值）
	Candidates      []Candidate   // 所有符合門檻的候選清單
	ReviewRequired  bool          // 是否需要使用者人工確認
}

// Router 是 skill 路由的主服務結構。
// 它依賴 ArchiveService 取得已歸檔的 skill 清單，
// 並透過 Scorer 和 AmbiguityDetector 完成路由決策。
//
// 重要：Router 應與 ArchiveService 共用同一個實例（見 app.go 的初始化），
// 確保 ListArchived 讀取的是同一份歸檔資料庫，而非另起一個空的 ArchiveService。
type Router struct {
	archiveService    *ArchiveService           // 共用的歸檔服務，不可為 nil
	scorer            *Scorer                   // 相關性評分器
	ambiguity         *AmbiguityDetector        // 模糊狀態判定器
	builtinManifests  map[string]*SkillManifest // 內建能力，啟動時註冊，優先於外部 skill
}

// NewRouter 建立一個與 archiveService 綁定的 Router。
// scorer 與 ambiguity 都使用預設實作，不需要外部配置。
//
// 呼叫端注意：
//   - archiveService 必須與 app.go 中 skillArchive 欄位共用同一個指標，
//     否則路由器讀到的 manifest 清單會是空的（不同實例的 previews cache 互相獨立）。
func NewRouter(archiveService *ArchiveService) *Router {
	return &Router{
		archiveService:   archiveService,
		scorer:           NewScorer(),
		ambiguity:        NewAmbiguityDetector(),
		builtinManifests: make(map[string]*SkillManifest),
	}
}

// RegisterBuiltin 註冊內建能力 manifest，啟動時呼叫。
// 內建 skill 不寫磁碟，優先於外部同 ID skill。
func (r *Router) RegisterBuiltin(m *SkillManifest) {
	m.Source = SkillSource{SourceType: SourceBuiltin}
	r.builtinManifests[m.SkillID] = m
}

// ActionTags returns all router-visible action tags for prompt synthesis.
func (r *Router) ActionTags() ([]string, error) {
	manifests, err := r.archiveService.ListArchived()
	if err != nil {
		return nil, fmt.Errorf("skill_step: action tags list archived: %w", err)
	}
	for _, m := range r.builtinManifests {
		manifests = append(manifests, *m)
	}
	return collectManifestActionTags(manifests), nil
}

// BuiltinActionTags exposes Controller-owned builtin tags even if archive fails.
func (r *Router) BuiltinActionTags() []string {
	manifests := make([]SkillManifest, 0, len(r.builtinManifests))
	for _, m := range r.builtinManifests {
		manifests = append(manifests, *m)
	}
	return collectManifestActionTags(manifests)
}

func collectManifestActionTags(manifests []SkillManifest) []string {
	seen := make(map[string]bool)
	var tags []string
	for _, manifest := range manifests {
		for _, tag := range manifest.Tags.ActionTag {
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

// Resolve 嘗試將 at（ActionTarget）對應到已歸檔的 skill。
//
// 流程：
//  1. 從 archiveService 載入所有已歸檔的 skill manifest
//  2. 用 Scorer 對每個 manifest 評分，score > 0 的加入候選清單
//  3. 計算全域最低自動選取門檻（取所有 manifest 中最保守的值）
//  4. 交由 AmbiguityDetector 判定最終 ResolveStatus
//  5. 若狀態為 auto_selected 且只有一個候選，填入 SelectedSkillID
func (r *Router) Resolve(at ActionTarget, sessionID string) (*ResolveResult, error) {
	// 取得所有已歸檔的 skill manifest
	manifests, err := r.archiveService.ListArchived()
	if err != nil {
		return nil, fmt.Errorf("skill_step: resolve list archived: %w", err)
	}

	// 合併 builtin + archived，builtin 優先（同 ID 覆蓋外部）
	merged := make(map[string]SkillManifest, len(manifests)+len(r.builtinManifests))
	for _, m := range manifests {
		merged[m.SkillID] = m
	}
	for _, m := range r.builtinManifests {
		merged[m.SkillID] = *m // builtin 覆蓋同 ID 外部 skill
	}

	// 對每個 manifest 評分，收集分數 > 0 的候選
	var candidates []Candidate
	for _, m := range merged {
		score := r.scorer.Score(at, m)
		if score <= 0 {
			continue // 分數為零：此 skill 與本次動作無關，略過
		}

		// 從 manifest 取出風險標籤；若無標籤則預設 "low"（最保守安全預設）
		risk := "low"
		if len(m.Tags.RiskTag) > 0 {
			risk = m.Tags.RiskTag[0]
		}

		candidates = append(candidates, Candidate{
			SkillID: m.SkillID,
			Score:   score,
			Risk:    risk,
			Reason:  fmt.Sprintf("score=%.2f via action/target match", score),
		})

	}

	// Product decision: routing confidence is fixed at 0.8. New tools start at
	// 0.8 and later evidence can add or subtract score, so fresh tools are still
	// eligible as candidates without per-skill dynamic thresholds.
	const minAutoScore = 0.8

	// 交由 AmbiguityDetector 依優先規則判定最終狀態
	status := r.ambiguity.Decide(candidates, minAutoScore)

	result := &ResolveResult{
		ResolveID:      fmt.Sprintf("resolve-%d", time.Now().UnixNano()),
		SessionID:      sessionID,
		Status:         status,
		Candidates:     candidates,
		ReviewRequired: status == StatusNeedsReview, // 快捷欄位，與 Status 保持同步
	}

	// 僅在自動選取且恰好只有一個候選時，填入 SelectedSkillID
	// 多個候選即使都是 auto_selected 分數也不應自動選——交由 CLI 或使用者決定
	if status == StatusAutoSelected && len(candidates) == 1 {
		result.SelectedSkillID = candidates[0].SkillID
	}

	return result, nil
}
