// visual_learning/detector.go — Platform-native Inference Engine 統一介面。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.6
//
// 設計原則（§14.6.7 Architecture Summary）：
//   - InferenceEngine 介面只做三件事：載入模型、推論、釋放資源。
//   - 平台橋接層（CoreML / DirectML）是「矩陣計算機」——
//     只負責餵圖片、吐浮點數陣列，不含任何 YOLO 業務邏輯。
//   - 所有 YOLO 後處理（BBox 解碼、NMS、score filter）在
//     yolo_postprocess.go 的純 Go 共用層處理。
//   - 換 YOLO 版本時只改 yolo_postprocess.go，平台橋接層不動。
//
// 硬規範摘要：
//   §14.6.1 — macOS CoreML bridge 固定使用 .m 檔，不 inline 進 .go
//   §14.6.2 — CoreML 推論固定 runtime.LockOSThread() worker
//   §14.6.3 — unsafe 操作只允許在 bridge 檔案內部
//   §14.6.4 — Windows DLL 載入集中初始化，缺失回傳 typed error + 降級
//   §14.6.5 — 測試 fixture 用 //go:embed testdata/
//   §14.6.6 — LoadModel() 前 SHA256 驗證，hash 來自受信任 manifest
package visual_learning

import (
	"errors"
	"fmt"
)

// ──────────────────────────────────────────────
// Typed Errors（§14.6.4 + §14.6.6）
// ──────────────────────────────────────────────

// ErrInferenceUnavailable 表示當前平台不支援硬體推論，
// 或 DLL/framework 載入失敗。收到此 error 時應降級為 OpenCV-only。
// 降級規則（§14.5）：不增加信心分、不跳過 dry-run、不繞過 review。
var ErrInferenceUnavailable = errors.New("inference engine: hardware inference unavailable on this platform")

// ErrModelIntegrityMismatch 表示模型檔 SHA256 與受信任 manifest 不一致。
// 收到此 error 時必須拒絕載入，不得 fallback 到未驗證模型（§14.6.6）。
var ErrModelIntegrityMismatch = errors.New("inference engine: model file SHA256 does not match trusted manifest")

// ErrModelNotLoaded 表示呼叫 Infer() 時模型尚未載入。
var ErrModelNotLoaded = errors.New("inference engine: model not loaded; call LoadModel() first")

// ErrEngineAlreadyClosed 表示引擎已釋放，不可再使用。
var ErrEngineAlreadyClosed = errors.New("inference engine: engine already closed")

// ──────────────────────────────────────────────
// RawTensor — 推論原始輸出（§14.6.3）
// ──────────────────────────────────────────────

// RawTensor 是平台橋接層回傳的原始推論結果。
// 外部 Go 程式（yolo_postprocess.go）只會拿到這個結構，
// 不會接觸任何 native pointer 或 unsafe 操作。
//
// 典型 YOLO nano 輸出 shape: [1, 25200, 85]
//   - 25200 = anchor 數量（不同 grid 加總）
//   - 85 = 4 (x,y,w,h) + 1 (objectness) + 80 (class scores)
type RawTensor struct {
	// Data 是推論輸出的浮點數陣列（已從 native memory 複製到 Go heap）。
	// 平台橋接層負責複製，外部不持有 native pointer。
	Data []float32 `json:"data"`

	// Shape 描述 tensor 的維度，例如 [1, 25200, 85]。
	// 後處理層根據 shape 解讀 Data 的排列方式。
	Shape []int `json:"shape"`
}

// TotalElements 回傳 tensor 的總元素數（所有維度相乘）。
func (t RawTensor) TotalElements() int {
	if len(t.Shape) == 0 {
		return 0
	}
	n := 1
	for _, dim := range t.Shape {
		n *= dim
	}
	return n
}

// Validate 驗證 Data 長度是否與 Shape 一致。
func (t RawTensor) Validate() error {
	expected := t.TotalElements()
	if len(t.Data) != expected {
		return fmt.Errorf("raw tensor: data length %d does not match shape %v (expected %d)", len(t.Data), t.Shape, expected)
	}
	return nil
}

// ──────────────────────────────────────────────
// InferenceEngine — 統一推論介面（§14.6.7）
// ──────────────────────────────────────────────

// InferenceEngine 是跨平台推論引擎的統一介面。
// 每個平台（macOS CoreML / Windows DirectML / fallback stub）
// 各自實作此介面。
//
// 使用方式：
//
//	engine := NewInferenceEngine()
//	defer engine.Close()  // ← 保證釋放所有 native 資源（§14.6.2）
//	if err := engine.LoadModel("path/to/model"); err != nil { ... }
//	raw, err := engine.Infer(rgbaBytes, width, height)
//	proposals := DecodeYOLOOutput(raw, ...)  // ← 純 Go 後處理
type InferenceEngine interface {
	// LoadModel 載入推論模型。
	//
	// 實作必須在載入前驗證模型檔 SHA256（§14.6.6），
	// hash 不符時回傳 ErrModelIntegrityMismatch。
	//
	// macOS: 載入 .mlmodelc（CoreML 格式）
	// Windows: 載入 .onnx（ONNX 格式）
	// Fallback: 回傳 ErrInferenceUnavailable
	LoadModel(path string) error

	// Infer 執行一次推論。
	//
	// 輸入：RGBA 原始位元組（4 bytes per pixel）+ 影像寬高。
	// 輸出：RawTensor — 純浮點數陣列 + shape metadata。
	//
	// 橋接層不做任何 YOLO 後處理（不解碼 BBox、不做 NMS、不過濾 score）。
	// 所有後處理在 yolo_postprocess.go 的 DecodeYOLOOutput() 完成。
	//
	// 模型未載入時回傳 ErrModelNotLoaded。
	// 引擎已關閉時回傳 ErrEngineAlreadyClosed。
	Infer(rgba []byte, width, height int) (RawTensor, error)

	// Close 釋放所有 native 資源。
	//
	// macOS: 釋放 MLModel session（CVPixelBuffer 由 @autoreleasepool 管理）
	// Windows: OrtReleaseSession() + OrtReleaseEnv()
	// Fallback: no-op
	//
	// 呼叫後引擎不可再使用，後續 Infer() 回傳 ErrEngineAlreadyClosed。
	// Go 端應使用 defer engine.Close() 確保釋放。
	Close() error
}

// ──────────────────────────────────────────────
// InferenceStatus — 狀態回報
// ──────────────────────────────────────────────

// InferenceStatus 回報推論引擎的當前狀態，供 UI 顯示。
type InferenceStatus struct {
	// Available 為 true 表示引擎已載入模型，可以推論。
	Available bool `json:"available"`

	// Backend 標示使用的推論後端，例如 "coreml", "directml", "stub"。
	Backend string `json:"backend"`

	// Degraded 為 true 表示引擎處於降級模式（例如模型缺失、DLL 不可用）。
	Degraded bool `json:"degraded"`

	// Reason 在 Degraded=true 時說明降級原因。
	Reason string `json:"reason,omitempty"`
}
