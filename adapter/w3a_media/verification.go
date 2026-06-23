// w3a_media/verification.go — §9A.10 主驗證流程（10 步判定 → 7 種狀態）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 媒體驗證的核心決策引擎，整合所有子模組產生最終狀態。        │
// │                                                             │
// │ 判定流程（10 步）：                                         │
// │  1. 讀取 sidecar .w3a.json                                  │
// │  2. 計算 byte hash                                          │
// │  3. byte hash 匹配 → exact_original                         │
// │  4. 計算 perceptual hash                                    │
// │  5. 檢查 app operation fingerprint + developer signature    │
// │  6. 有效簽章 → w3a_app_processed                            │
// │  7. 感知匹配但無簽章 → platform_processed_copy              │
// │  8. 執行 pollution detection → model_pollution_risk         │
// │  9. 有 sidecar 但指紋不匹配 → content_modified              │
// │  10. 無任何 W3A 證據 → unverified                           │
// │                                                             │
// │ 硬規則強制執行：                                            │
// │  - platform_processed_copy 不保留延伸權（§9A.8）            │
// │  - 感知匹配不恢復延伸權（§9A.11）                           │
// │  - model_pollution_risk 不進訓練語料（§9A.9）               │
// └─────────────────────────────────────────────────────────────┘
package w3a_media

import (
	"fmt"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// 感知匹配閾值
// ──────────────────────────────────────────────

const (
	// PerceptualMatchThreshold 感知指紋相似度閾值。
	// 超過此值視為「來源相似」（但不恢復延伸權）。
	PerceptualMatchThreshold = 0.85
)

// ──────────────────────────────────────────────
// 主驗證函式
// ──────────────────────────────────────────────

// VerifyMedia 執行完整 W3A 媒體驗證流程。
// filePath: 目標媒體檔案路徑
// sidecar: 已載入的 sidecar 資訊（可為 nil）
// trustList: 本機信任清單（用於驗證開發者簽章）
func VerifyMedia(filePath string, sidecar *W3AMediaInfo, trustList *TrustList) (*W3AMediaInfo, error) {
	scope := detectMediaScope(filePath)
	now := time.Now()

	info := &W3AMediaInfo{
		Version:    "1.0",
		MediaScope: scope,
		FilePath:   filePath,
		Status:     StatusUnverified,
		VerifiedAt: now,
	}

	// Step 1–2: 計算目標檔案的 byte hash
	byteHash, err := ComputeByteHash(filePath)
	if err != nil {
		return info, fmt.Errorf("compute byte hash: %w", err)
	}

	// Step 3: 與 sidecar 比對 byte hash
	if sidecar != nil && sidecar.Fingerprint.OverallByteHash == byteHash {
		// byte hash 完全匹配 → exact_original
		info.Status = StatusExactOriginal
		info.Fingerprint = sidecar.Fingerprint
		info.Operations = sidecar.Operations
		info.DeveloperSignature = sidecar.DeveloperSignature
		info.CreatedAt = sidecar.CreatedAt
		return info, nil
	}

	// Step 4: 計算 perceptual hash
	pHash, pErr := ComputePerceptualHash(filePath, scope)

	// 設定目前檔案的指紋
	info.Fingerprint.OverallByteHash = byteHash
	if pErr == nil {
		info.Fingerprint.OverallPerceptualHash = pHash
	}

	// Step 5–6: 檢查 sidecar 中的 app operation fingerprint + developer signature
	if sidecar != nil && sidecar.DeveloperSignature != nil {
		sigValid := false
		for _, op := range sidecar.Operations {
			valid, verr := VerifySignature(op, *sidecar.DeveloperSignature)
			if verr == nil && valid {
				// 進一步檢查是否在信任清單中
				if trustList != nil && trustList.IsTrusted(
					sidecar.DeveloperSignature.AppID,
					sidecar.DeveloperSignature.PublicKey,
				) {
					sigValid = true
					break
				}
			}
		}

		if sigValid {
			info.Status = StatusW3AAppProcessed
			info.Operations = sidecar.Operations
			info.DeveloperSignature = sidecar.DeveloperSignature
			// 繼續執行 pollution check
		}
	}

	// Step 7: 感知匹配但無有效簽章 → platform_processed_copy
	// 注意：感知匹配不恢復延伸權（§9A.11 硬規則）
	if info.Status == StatusUnverified && sidecar != nil && pErr == nil {
		similarity := ComparePerceptual(pHash, sidecar.Fingerprint.OverallPerceptualHash)
		if similarity >= PerceptualMatchThreshold {
			info.Status = StatusPlatformProcessed
			// 明確不設定延伸權
		}
	}

	// Step 8: 執行 pollution detection
	pollReport, pollErr := DetectPollution(filePath, scope)
	if pollErr == nil {
		info.Pollution = pollReport
		if pollReport.IsPollutionRisk {
			// model_pollution_risk 覆蓋其他狀態（§9A.9）
			info.Status = StatusModelPollutionRisk
			return info, nil
		}
	}

	// Step 9: 有 sidecar 但指紋不匹配 → content_modified
	if info.Status == StatusUnverified && sidecar != nil {
		info.Status = StatusContentModified
	}

	// Step 10: 無任何 W3A 證據 → unverified（已是預設值）

	return info, nil
}

// ──────────────────────────────────────────────
// 媒體類型偵測
// ──────────────────────────────────────────────

// detectMediaScope 根據副檔名推斷媒體類型。
func detectMediaScope(filePath string) MediaScope {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".png"),
		strings.HasSuffix(lower, ".jpg"),
		strings.HasSuffix(lower, ".jpeg"),
		strings.HasSuffix(lower, ".gif"),
		strings.HasSuffix(lower, ".bmp"),
		strings.HasSuffix(lower, ".webp"),
		strings.HasSuffix(lower, ".tiff"),
		strings.HasSuffix(lower, ".svg"):
		return ScopeImage
	case strings.HasSuffix(lower, ".wav"),
		strings.HasSuffix(lower, ".mp3"),
		strings.HasSuffix(lower, ".flac"),
		strings.HasSuffix(lower, ".aac"),
		strings.HasSuffix(lower, ".ogg"),
		strings.HasSuffix(lower, ".m4a"):
		return ScopeAudio
	case strings.HasSuffix(lower, ".mp4"),
		strings.HasSuffix(lower, ".avi"),
		strings.HasSuffix(lower, ".mkv"),
		strings.HasSuffix(lower, ".mov"),
		strings.HasSuffix(lower, ".webm"):
		return ScopeVideo
	default:
		return ScopeImage // 預設
	}
}
