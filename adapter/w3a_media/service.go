// w3a_media/service.go — §9A W3A Media Provenance 主服務。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 整合所有 W3A Media 子模組的統一入口，供 app.go Wails        │
// │ binding 呼叫。                                              │
// │                                                             │
// │ 組合元件：                                                  │
// │  - KeyManager      → Ed25519 金鑰對管理（app_fingerprint）  │
// │  - TrustList       → 開發者信任清單（trust_list）           │
// │  - OperationRecorder → 操作記錄器（app_fingerprint）        │
// │                                                             │
// │ 主要 API（對應 Wails binding）：                            │
// │  1. VerifyMediaFile   → 完整驗證流程                        │
// │  2. GetMediaW3AInfo   → 讀取/計算 W3A 資訊                  │
// │  3. DetectPollution   → 模型污染偵測                        │
// │  4. ExportWithSidecar → 匯出媒體 + sidecar                  │
// │  5. ImportAndVerify   → 匯入驗證 + UX 提示                  │
// │  6. GetTransferGuidance → 傳輸引導                          │
// │  7. ListTrusted / AddTrusted → 信任清單管理                 │
// └─────────────────────────────────────────────────────────────┘
package w3a_media

import (
	"fmt"
	"time"
)

// ──────────────────────────────────────────────
// 主服務
// ──────────────────────────────────────────────

// Service W3A Media Provenance 主服務。
type Service struct {
	keyManager *KeyManager
	trustList  *TrustList
	recorder   *OperationRecorder
}

// NewService 建立 W3A Media 服務。
func NewService(hookRoot string) *Service {
	return &Service{
		keyManager: NewKeyManager(hookRoot),
		trustList:  NewTrustList(hookRoot),
		recorder:   NewOperationRecorder(),
	}
}

// ──────────────────────────────────────────────
// API 1: 完整驗證
// ──────────────────────────────────────────────

// VerifyMediaFile 對媒體檔案執行完整 W3A 驗證流程。
func (s *Service) VerifyMediaFile(filePath string) (*W3AMediaInfo, error) {
	// 嘗試讀取 sidecar
	sidecar, _ := ReadSidecar(filePath)

	// 執行驗證
	info, err := VerifyMedia(filePath, sidecar, s.trustList)
	if err != nil {
		return nil, err
	}

	// 計算訓練資格
	MarkTrainingEligibility(info)

	return info, nil
}

// ──────────────────────────────────────────────
// API 2: 取得/建立 W3A 資訊
// ──────────────────────────────────────────────

// GetMediaW3AInfo 取得媒體的 W3A 資訊。
// 若 sidecar 存在則讀取，否則計算新的指紋。
func (s *Service) GetMediaW3AInfo(filePath string) (*W3AMediaInfo, error) {
	// 先嘗試讀取 sidecar
	sidecar, _ := ReadSidecar(filePath)
	if sidecar != nil {
		return sidecar, nil
	}

	// 計算新的指紋
	scope := detectMediaScope(filePath)
	byteHash, err := ComputeByteHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("compute byte hash: %w", err)
	}

	pHash, _ := ComputePerceptualHash(filePath, scope)

	info := &W3AMediaInfo{
		Version:    "1.0",
		MediaScope: scope,
		FilePath:   filePath,
		Status:     StatusUnverified,
		Fingerprint: DualLayerFingerprint{
			OverallByteHash:       byteHash,
			OverallPerceptualHash: pHash,
		},
		CreatedAt: time.Now(),
	}

	return info, nil
}

// ──────────────────────────────────────────────
// API 3: 模型污染偵測
// ──────────────────────────────────────────────

// DetectMediaPollution 對媒體執行模型污染偵測。
func (s *Service) DetectMediaPollution(filePath string) (*PollutionReport, error) {
	scope := detectMediaScope(filePath)
	return DetectPollution(filePath, scope)
}

// ──────────────────────────────────────────────
// API 4: 匯出（帶 sidecar）
// ──────────────────────────────────────────────

// ExportMedia 匯出媒體檔案，同時複製 sidecar。
func (s *Service) ExportMedia(srcPath, destPath string) error {
	return ExportWithSidecar(srcPath, destPath)
}

// ──────────────────────────────────────────────
// API 5: 匯入驗證 + UX 提示
// ──────────────────────────────────────────────

// ImportAndVerify 匯入媒體並驗證，回傳含 UX 提示的結果。
func (s *Service) ImportAndVerify(filePath string) (*ImportResult, error) {
	hasSidecar := HasSidecar(filePath)

	info, err := s.VerifyMediaFile(filePath)
	if err != nil {
		return nil, err
	}

	// 組裝 W3A 功能說明（給前端選單用）
	capabilities := []string{
		"查看媒體驗證狀態（7 種等級）",
		"檢查媒體是否為原始檔案",
		"偵測模型污染風險",
		"查看操作指紋與開發者簽章",
		"判定是否適合作為訓練資料",
	}

	recommendation := "此媒體無 W3A 驗證資訊。"
	if hasSidecar {
		recommendation = fmt.Sprintf("偵測到 W3A sidecar，驗證結果：%s", info.Status.Label())
	}

	return &ImportResult{
		Info:           *info,
		HasSidecar:     hasSidecar,
		SidecarPath:    SidecarPath(filePath),
		Capabilities:   capabilities,
		Recommendation: recommendation,
	}, nil
}

// ──────────────────────────────────────────────
// API 6: 傳輸引導
// ──────────────────────────────────────────────

// GetGuidance 取得原檔傳輸引導建議。
func (s *Service) GetGuidance() TransferGuidance {
	return GetTransferGuidance()
}

// ──────────────────────────────────────────────
// API 7: 信任清單管理
// ──────────────────────────────────────────────

// ListTrustedDevelopers 列出所有信任的開發者。
func (s *Service) ListTrustedDevelopers() []TrustedDeveloper {
	return s.trustList.List()
}

// AddTrustedDeveloper 新增信任的開發者。
func (s *Service) AddTrustedDeveloper(appID, pubKey, displayName string) error {
	return s.trustList.Add(appID, pubKey, displayName)
}

// RemoveTrustedDeveloper 移除信任的開發者。
func (s *Service) RemoveTrustedDeveloper(appID string) error {
	return s.trustList.Remove(appID)
}

// ──────────────────────────────────────────────
// 內部：為匯出的媒體建立 sidecar
// ──────────────────────────────────────────────

// CreateSidecar 為媒體檔案建立新的 .w3a.json sidecar。
// 計算指紋並用 Console 的金鑰簽署。
func (s *Service) CreateSidecar(filePath string) (*W3AMediaInfo, error) {
	scope := detectMediaScope(filePath)

	byteHash, err := ComputeByteHash(filePath)
	if err != nil {
		return nil, err
	}

	pHash, _ := ComputePerceptualHash(filePath, scope)

	// 收集已記錄的操作
	ops := s.recorder.GetAll()

	// 建立 info
	info := &W3AMediaInfo{
		Version:    "1.0",
		MediaScope: scope,
		FilePath:   filePath,
		Status:     StatusExactOriginal,
		Fingerprint: DualLayerFingerprint{
			OverallByteHash:       byteHash,
			OverallPerceptualHash: pHash,
		},
		Operations: ops,
		CreatedAt:  time.Now(),
	}

	// 如有操作指紋，簽署最後一個
	if len(ops) > 0 {
		sig, err := s.keyManager.SignOperation(ops[len(ops)-1])
		if err == nil {
			info.DeveloperSignature = sig
			info.Status = StatusW3AAppProcessed
		}
	}

	// 計算訓練資格
	MarkTrainingEligibility(info)

	// 寫入 sidecar
	if err := WriteSidecar(info, filePath); err != nil {
		return nil, err
	}

	return info, nil
}
