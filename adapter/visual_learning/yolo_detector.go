// visual_learning/yolo_detector.go — YOLO 偵測器（重構版）。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.3 + §14.6
//
// 重構說明（v4.2）：
//   原版為純 stub（永遠回傳 Degraded=true）。
//   重構後串接 InferenceEngine（§14.6.7）+ DecodeYOLOOutput（純 Go 後處理）。
//   InferenceEngine 由各平台的 build tag 決定實際實作：
//     macOS   → coreml_bridge_darwin.go  （CoreML）
//     Windows → directml_bridge_windows.go（DirectML / ONNX Runtime）
//     其他    → stub_bridge_fallback.go  （ErrInferenceUnavailable）
//
// 安全規則（§14.3）：
//   - YOLO 只提議區域（RegionProposal），不做語義判斷
//   - 不直接標記「提交按鈕」/「刪除按鈕」/「付款按鈕」
//   - 語義判斷需 DOM + OCR + LLM + 風險規則共同決定
//
// 降級規則（§14.5 + §14.6.4）：
//   - 引擎不可用時 → Degraded=true，fallback 到 OpenCV pipeline
//   - 降級模式不增加信心分、不跳過 dry-run、不繞過 review
package visual_learning

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

// YOLODetector 是 YOLO 區域偵測器的高層封裝。
// 它持有 InferenceEngine（平台原生推論）和 OpenCVPipeline（降級 fallback）。
//
// 使用方式：
//
//	detector := NewYOLODetector("assets/models/yolo_nano", pipeline)
//	result := detector.Detect(imageData, width, height)
//	// result.Proposals 包含 BBox + raw_score
//	// result.Degraded 表示是否使用了 OpenCV fallback
type YOLODetector struct {
	mu sync.Mutex

	// modelPath 是模型檔基底路徑（不含副檔名）。
	// macOS 會嘗試 modelPath + ".mlmodelc"
	// Windows 會嘗試 modelPath + ".onnx"
	modelPath string

	// engine 是平台原生推論引擎（由 build tag 決定實作）。
	engine InferenceEngine

	// engineReady 表示 engine 已載入模型且可用。
	engineReady bool

	// fallbackPipeline 是 OpenCV 純 Go 管線，作為降級 fallback。
	fallbackPipeline *OpenCVPipeline

	// config 是 YOLO 後處理參數（換版本時只改這裡）。
	config YOLOConfig

	// degradedReason 記錄引擎不可用的原因（供 UI 顯示）。
	degradedReason string
}

// DetectorResult 是 YOLO 偵測的輸出。
// 無論使用 YOLO 引擎或 OpenCV fallback，都回傳相同格式。
type DetectorResult struct {
	// Proposals 是偵測到的候選區域列表。
	// 不含語義標籤 — 語義分類在下游處理。
	Proposals []RegionProposal `json:"proposals"`

	// Degraded 為 true 表示使用了 OpenCV fallback（非 YOLO 推論）。
	Degraded bool `json:"degraded"`

	// Reason 在 Degraded=true 時說明降級原因。
	Reason string `json:"reason,omitempty"`

	// Backend 標示實際使用的偵測後端。
	Backend string `json:"backend"`
}

// RegionProposal 是單一候選區域。
// 只包含 BBox 和原始分數，不含語義標籤（§14.3）。
type RegionProposal struct {
	BBox       BBox    `json:"bbox"`
	RawScore   float64 `json:"raw_score"`   // 模型 confidence，不是 final_confidence
	ProposalID string  `json:"proposal_id"` // 唯一識別碼
}

// NewYOLODetector 建立 YOLO 偵測器。
//
// modelBasePath: 模型檔基底路徑（不含副檔名），
//   例如 "assets/models/yolo_nano"
//   macOS 會嘗試 + ".mlmodelc"，Windows 會嘗試 + ".onnx"
//
// fallback: OpenCV 純 Go 管線，作為降級 fallback（不可為 nil）
func NewYOLODetector(modelBasePath string, fallback *OpenCVPipeline) *YOLODetector {
	d := &YOLODetector{
		modelPath:        modelBasePath,
		fallbackPipeline: fallback,
		config:           DefaultYOLOv5NanoConfig,
	}

	// 嘗試初始化平台原生推論引擎
	d.engine = NewInferenceEngine()
	d.tryLoadModel()

	return d
}

// tryLoadModel 嘗試載入模型。引擎不可用或模型缺失時記錄原因，不 panic。
func (d *YOLODetector) tryLoadModel() {
	modelFile := d.findModelFile()
	if modelFile == "" {
		d.degradedReason = fmt.Sprintf("model file not found at base path %q (.mlmodelc / .onnx)", d.modelPath)
		return
	}

	// LoadModel 內部會做 hash 驗證（§14.6.6）
	err := d.engine.LoadModel(modelFile)
	if err != nil {
		if errors.Is(err, ErrInferenceUnavailable) {
			d.degradedReason = "hardware inference unavailable on this platform"
		} else if errors.Is(err, ErrModelIntegrityMismatch) {
			d.degradedReason = fmt.Sprintf("model integrity check failed: %v", err)
		} else {
			d.degradedReason = fmt.Sprintf("model load failed: %v", err)
		}
		return
	}

	d.engineReady = true
}

// findModelFile 根據平台尋找模型檔。
// 嘗試 .mlmodelc（CoreML 目錄）和 .onnx（ONNX 單檔）。
func (d *YOLODetector) findModelFile() string {
	mlmodelc := d.modelPath + ".mlmodelc"
	if info, err := os.Stat(mlmodelc); err == nil && info.IsDir() {
		return mlmodelc
	}

	onnx := d.modelPath + ".onnx"
	if info, err := os.Stat(onnx); err == nil && !info.IsDir() {
		return onnx
	}

	return ""
}

// IsAvailable 回傳 YOLO 推論引擎是否就緒。
func (d *YOLODetector) IsAvailable() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.engineReady
}

// Status 回傳推論引擎的詳細狀態（供 UI 顯示）。
func (d *YOLODetector) Status() InferenceStatus {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.engineReady {
		return InferenceStatus{
			Available: true,
			Backend:   InferenceBackendName(),
			Degraded:  false,
		}
	}
	return InferenceStatus{
		Available: false,
		Backend:   InferenceBackendName(),
		Degraded:  true,
		Reason:    d.degradedReason,
	}
}

// Detect 執行區域偵測。
//
// YOLO 引擎可用時：
//   影像 → Infer() → RawTensor → DecodeYOLOOutput() → []RegionProposal
//
// YOLO 引擎不可用時（降級模式）：
//   影像 → OpenCVPipeline.Propose() → 轉換為 DetectorResult
//
// 降級模式不增加信心分、不跳過 dry-run、不繞過 review（§14.5）。
func (d *YOLODetector) Detect(imageData []byte, width, height int) DetectorResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 引擎可用：走 YOLO 推論路徑
	if d.engineReady {
		raw, err := d.engine.Infer(imageData, width, height)
		if err != nil {
			// 推論失敗 → fallback 到 OpenCV
			return d.detectWithOpenCV(imageData, width, height,
				fmt.Sprintf("YOLO inference failed: %v; falling back to OpenCV", err))
		}

		// 純 Go 後處理
		proposals, decodeErr := DecodeYOLOOutput(raw, width, height, d.config)
		if decodeErr != nil {
			return d.detectWithOpenCV(imageData, width, height,
				fmt.Sprintf("YOLO decode failed: %v; falling back to OpenCV", decodeErr))
		}

		return DetectorResult{
			Proposals: proposals,
			Degraded:  false,
			Backend:   InferenceBackendName(),
		}
	}

	// 引擎不可用 → fallback 到 OpenCV
	return d.detectWithOpenCV(imageData, width, height, d.degradedReason)
}

// detectWithOpenCV 使用 OpenCV pipeline 作為 fallback。
func (d *YOLODetector) detectWithOpenCV(imageData []byte, width, height int, reason string) DetectorResult {
	if d.fallbackPipeline == nil {
		return DetectorResult{
			Proposals: nil,
			Degraded:  true,
			Reason:    "no fallback pipeline available",
			Backend:   "none",
		}
	}

	pipeResult := d.fallbackPipeline.Propose(imageData, width, height)

	// UIFingerprint → RegionProposal 轉換
	proposals := make([]RegionProposal, 0, len(pipeResult.Candidates))
	for i, fp := range pipeResult.Candidates {
		proposals = append(proposals, RegionProposal{
			BBox:       fp.BBoxRelative,
			RawScore:   fp.Confidence,
			ProposalID: fmt.Sprintf("opencv-%d", i),
		})
	}

	return DetectorResult{
		Proposals: proposals,
		Degraded:  true,
		Reason:    reason,
		Backend:   "opencv",
	}
}

// Close 釋放推論引擎的所有 native 資源。
// 應在 App 關閉時呼叫（由 session_close.go 的 shutdown hook 觸發）。
func (d *YOLODetector) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.engine != nil {
		err := d.engine.Close()
		d.engineReady = false
		return err
	}
	return nil
}
