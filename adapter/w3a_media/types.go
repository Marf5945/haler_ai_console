// w3a_media/types.go — §9A W3A Media Provenance 核心型別。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 本檔案是 W3A Media 模組的「資料字典」，定義所有跨檔案       │
// │ 共用的型別與常數。                                          │
// │                                                             │
// │ 架構層次：                                                  │
// │  1. VerificationStatus  — 7 種驗證狀態（§9A.10）            │
// │  2. MediaScope          — 適用媒體類型                      │
// │  3. DualLayerFingerprint — 整體 + 段落雙層指紋（§9A.3）     │
// │  4. AppOperationFingerprint — W3A-aware app 操作記錄（§9A.4）│
// │  5. DeveloperSignature  — 開發者簽章（§9A.5）               │
// │  6. PollutionReport     — 模型污染偵測報告（§9A.9）         │
// │  7. W3AMediaInfo        — 頂層資訊結構（含驗證 + 指紋 + 政策）│
// │  8. TrainingEligibility — 訓練資料政策判定（§9A.14）        │
// │                                                             │
// │ 設計約束：                                                  │
// │  - 所有計算 Go-native only，零外部依賴（防供應鏈攻擊）      │
// │  - 感知指紋匹配不得恢復延伸權（§9A.11）                     │
// │  - platform_processed_copy 不得視為 training-safe（§9A.8）  │
// └─────────────────────────────────────────────────────────────┘
package w3a_media

import "time"

// ──────────────────────────────────────────────
// 驗證狀態（§9A.10）
// ──────────────────────────────────────────────

// VerificationStatus 媒體的 W3A 驗證結果，共 7 種。
type VerificationStatus string

const (
	StatusExactOriginal      VerificationStatus = "exact_original"
	StatusW3AAppProcessed    VerificationStatus = "w3a_app_processed"
	StatusPlatformProcessed  VerificationStatus = "platform_processed_copy"
	StatusUnauthorizedCopy   VerificationStatus = "unauthorized_copy"
	StatusContentModified    VerificationStatus = "content_modified"
	StatusModelPollutionRisk VerificationStatus = "model_pollution_risk"
	StatusUnverified         VerificationStatus = "unverified"
)

// Label 回傳驗證狀態的中文名稱。
func (v VerificationStatus) Label() string {
	switch v {
	case StatusExactOriginal:
		return "原始檔案"
	case StatusW3AAppProcessed:
		return "W3A 應用處理"
	case StatusPlatformProcessed:
		return "平台處理版本"
	case StatusUnauthorizedCopy:
		return "未授權複製"
	case StatusContentModified:
		return "內容已修改"
	case StatusModelPollutionRisk:
		return "模型污染風險"
	case StatusUnverified:
		return "未驗證"
	default:
		return string(v)
	}
}

// IsExtensionValid 判斷該狀態是否保有延伸權（§9A.10 表格）。
func (v VerificationStatus) IsExtensionValid() bool {
	switch v {
	case StatusExactOriginal, StatusW3AAppProcessed:
		return true
	default:
		return false
	}
}

// IsTrainingSafe 判斷該狀態是否可能為訓練安全原檔（§9A.14）。
// 注意：exact_original 與 w3a_app_processed 僅為「可能」安全，
// 仍需搭配其他政策判定。
func (v VerificationStatus) IsTrainingSafe() bool {
	switch v {
	case StatusExactOriginal:
		return true
	case StatusW3AAppProcessed:
		return true // 須搭配 app trust 判定
	default:
		return false
	}
}

// ──────────────────────────────────────────────
// 媒體範圍（§9A.2）
// ──────────────────────────────────────────────

// MediaScope 適用的媒體類型。
type MediaScope string

const (
	ScopeImage MediaScope = "image"
	ScopeAudio MediaScope = "audio"
	ScopeVideo MediaScope = "future_video" // 未來擴充
)

// ──────────────────────────────────────────────
// 雙層指紋（§9A.3 + §9A.11）
// ──────────────────────────────────────────────

// SegmentRegion 描述一個段落的位置資訊。
type SegmentRegion struct {
	Index       int    `json:"index"`
	Description string `json:"description"` // 例如 "tile:0,0~2,2" 或 "00:00.000~00:05.000"
}

// SegmentFingerprint 單一段落的雙重指紋。
type SegmentFingerprint struct {
	Region         SegmentRegion `json:"region"`
	ByteHash       string        `json:"byte_hash"`       // SHA-256 of segment data
	PerceptualHash string        `json:"perceptual_hash"` // Go-native pHash / spectral hash
}

// DualLayerFingerprint 整體 + 段落陣列的雙層結構。
type DualLayerFingerprint struct {
	OverallByteHash       string               `json:"overall_byte_hash"`
	OverallPerceptualHash string               `json:"overall_perceptual_hash"`
	Segments              []SegmentFingerprint `json:"segments,omitempty"`
}

// ──────────────────────────────────────────────
// App 操作指紋（§9A.4）
// ──────────────────────────────────────────────

// AppOperationFingerprint W3A-aware app 的操作記錄。
// 最低粒度：操作類型 / 時間範圍 / 影響區域 / 操作摘要數量。
type AppOperationFingerprint struct {
	Operation     string `json:"op"`                        // 例如 "brush_draw", "manual_audio_cut"
	TimeRange     string `json:"time_range"`                // ISO 8601 interval
	AffectedRange string `json:"affected_region,omitempty"` // 圖片: "tile:12,18~16,24" / 音訊: "00:12.000~00:18.500"
	SummaryCount  int    `json:"summary_count"`             // stroke_count / cut_count 等
}

// ──────────────────────────────────────────────
// 開發者簽章（§9A.5）
// ──────────────────────────────────────────────

// DeveloperSignature 由 W3A-aware app 開發者金鑰簽署操作指紋。
type DeveloperSignature struct {
	AppID     string `json:"app_id"`
	PublicKey string `json:"public_key"` // Ed25519 公鑰（hex 編碼）
	Signature string `json:"signature"`  // Ed25519 簽章（hex 編碼）
	SignedAt  string `json:"signed_at"`  // ISO 8601
}

// TrustedDeveloper 本機信任清單中的一筆開發者記錄。
type TrustedDeveloper struct {
	AppID       string `json:"app_id"`
	PublicKey   string `json:"public_key"`
	DisplayName string `json:"display_name"`
	AddedAt     string `json:"added_at"`
}

// RegistryStatus 線上登錄查詢結果（目前為 stub）。
type RegistryStatus string

const (
	RegistryVerified    RegistryStatus = "verified"
	RegistryUnknown     RegistryStatus = "unknown"
	RegistryUnavailable RegistryStatus = "registry_unavailable"
)

// ──────────────────────────────────────────────
// 模型污染偵測報告（§9A.9）
// ──────────────────────────────────────────────

// PollutionReport 模型污染偵測結果。
type PollutionReport struct {
	HighFreqScore   float64 `json:"high_freq_score"`   // 0.0–1.0，高頻能量異常
	HistogramScore  float64 `json:"histogram_score"`   // 0.0–1.0，直方圖異常
	LSBScore        float64 `json:"lsb_score"`         // 0.0–1.0，LSB 分佈異常
	WeightedTotal   float64 `json:"weighted_total"`    // 加權總分
	IsPollutionRisk bool    `json:"is_pollution_risk"` // WeightedTotal > 0.7
	Details         string  `json:"details,omitempty"` // 人可讀說明
}

// PollutionThreshold 模型污染判定閾值。
const PollutionThreshold = 0.7

// ──────────────────────────────────────────────
// 訓練資料政策（§9A.14）
// ──────────────────────────────────────────────

// TrainingEligibility 訓練資料政策判定結果。
type TrainingEligibility struct {
	TrainingSafe   bool   `json:"training_safe"`
	Reason         string `json:"reason"`
	FilterRequired bool   `json:"filter_required"` // 是否需要 §11 LLM Context 過濾
}

// ──────────────────────────────────────────────
// 頂層資訊結構
// ──────────────────────────────────────────────

// W3AMediaInfo 整合所有 W3A 驗證資訊的頂層結構。
// 對應 sidecar .w3a.json 的完整內容。
type W3AMediaInfo struct {
	Version            string                    `json:"version"`
	MediaScope         MediaScope                `json:"media_scope"`
	FilePath           string                    `json:"file_path,omitempty"`
	Status             VerificationStatus        `json:"status"`
	Fingerprint        DualLayerFingerprint      `json:"fingerprint"`
	Operations         []AppOperationFingerprint `json:"operations,omitempty"`
	DeveloperSignature *DeveloperSignature       `json:"developer_signature,omitempty"`
	Pollution          *PollutionReport          `json:"pollution,omitempty"`
	Training           *TrainingEligibility      `json:"training,omitempty"`
	CreatedAt          time.Time                 `json:"created_at"`
	VerifiedAt         time.Time                 `json:"verified_at,omitempty"`
}

// ──────────────────────────────────────────────
// 匯入結果
// ──────────────────────────────────────────────

// ImportResult 媒體匯入驗證結果（含 UX 提示）。
type ImportResult struct {
	Info           W3AMediaInfo `json:"info"`
	HasSidecar     bool         `json:"has_sidecar"`
	SidecarPath    string       `json:"sidecar_path,omitempty"`
	Capabilities   []string     `json:"capabilities"`   // W3A 功能清單（給前端選單用）
	Recommendation string       `json:"recommendation"` // 建議操作
}

// ──────────────────────────────────────────────
// 傳輸引導（§9A.12）
// ──────────────────────────────────────────────

// TransferGuidance 原檔傳輸引導建議。
type TransferGuidance struct {
	Recommended    []string `json:"recommended"`
	NotRecommended []string `json:"not_recommended"`
	UIMessage      string   `json:"ui_message"`
}
