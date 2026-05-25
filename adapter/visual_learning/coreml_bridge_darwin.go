// visual_learning/coreml_bridge_darwin.go — macOS CoreML 推論引擎（暫時 stub）。
//
// Build tag: 僅在 macOS 編譯。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.6.1 + §14.6.2
//
// 目前狀態：暫時 stub，所有方法回傳 ErrInferenceUnavailable。
// 正式 CoreML 橋接實作時，此檔案將被替換為：
//   - coreml_bridge_darwin.go（Go 端，呼叫 cgo）
//   - coreml_bridge_darwin.m（Objective-C 端，CoreML API）
//
// §14.6.1：CoreML bridge 的 Objective-C 程式碼必須在獨立 .m 檔案中，
//           不得 inline 進 .go 檔案。
// §14.6.2：CoreML 推論必須在 runtime.LockOSThread() 的專屬 worker 上執行。
// §14.6.3：unsafe 操作只允許在 bridge 檔案內部。
//
// 降級規則（§14.5 + §14.6.4）：
//   - 不增加信心分
//   - 不跳過 dry-run
//   - 不繞過 review
//   - UI 顯示「硬體推論不可用」

//go:build darwin

package visual_learning

// coremlStubEngine 是 macOS CoreML 推論引擎的暫時 stub。
// 正式實作前，所有推論相關方法回傳 ErrInferenceUnavailable，
// 讓系統降級到 OpenCV-only pipeline。
type coremlStubEngine struct {
	closed bool
}

// NewInferenceEngine 在 macOS 上回傳 CoreML 推論引擎。
// 目前為暫時 stub，正式實作時將初始化 CoreML session。
//
// 呼叫端應檢查 LoadModel() 的 error 來決定是否降級到 OpenCV-only。
func NewInferenceEngine() InferenceEngine {
	return &coremlStubEngine{}
}

// LoadModel 載入 CoreML 模型（.mlmodelc 格式）。
// 目前為 stub，永遠回傳 ErrInferenceUnavailable。
//
// 正式實作時流程：
//  1. ModelVerifier.Verify() 驗證 SHA256（§14.6.6）
//  2. runtime.LockOSThread() 鎖定 OS thread（§14.6.2）
//  3. 透過 cgo 呼叫 Objective-C 初始化 MLModel
func (e *coremlStubEngine) LoadModel(path string) error {
	return ErrInferenceUnavailable
}

// Infer 執行一次 CoreML 推論。
// 目前為 stub，永遠回傳 ErrInferenceUnavailable。
//
// 正式實作時流程：
//  1. RGBA → CVPixelBuffer 轉換（在 .m 檔案中，§14.6.1）
//  2. MLModel prediction（在 locked OS thread 上，§14.6.2）
//  3. 輸出 MLMultiArray → Go []float32 複製（§14.6.3：unsafe 限制在 bridge 內）
//  4. 回傳 RawTensor，不做任何 YOLO 後處理
func (e *coremlStubEngine) Infer(rgba []byte, width, height int) (RawTensor, error) {
	return RawTensor{}, ErrInferenceUnavailable
}

// Close 釋放 CoreML 資源。
// 目前為 stub，僅標記已關閉。
//
// 正式實作時：釋放 MLModel session，CVPixelBuffer 由 @autoreleasepool 管理。
func (e *coremlStubEngine) Close() error {
	e.closed = true
	return nil
}

// InferenceBackendName 回傳此平台的推論後端名稱。
func InferenceBackendName() string {
	return "coreml"
}
