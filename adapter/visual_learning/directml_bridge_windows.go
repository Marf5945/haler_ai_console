// visual_learning/directml_bridge_windows.go — Windows DirectML 推論引擎（暫時 stub）。
//
// Build tag: 僅在 Windows 編譯。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.6.4
//
// 目前狀態：暫時 stub，所有方法回傳 ErrInferenceUnavailable。
// 正式 DirectML 橋接實作時，此檔案將包含：
//   - DLL 載入（onnxruntime.dll）集中初始化
//   - ONNX Runtime session 管理
//   - DirectML execution provider 設定
//
// §14.6.4：Windows DLL 載入必須集中初始化，
//           缺失時回傳 typed error + 降級到 OpenCV-only。
// §14.6.3：unsafe 操作只允許在 bridge 檔案內部。
//
// 降級規則（§14.5 + §14.6.4）：
//   - 不增加信心分
//   - 不跳過 dry-run
//   - 不繞過 review
//   - UI 顯示「硬體推論不可用」

//go:build windows

package visual_learning

// directmlStubEngine 是 Windows DirectML 推論引擎的暫時 stub。
// 正式實作前，所有推論相關方法回傳 ErrInferenceUnavailable，
// 讓系統降級到 OpenCV-only pipeline。
type directmlStubEngine struct {
	closed bool
}

// NewInferenceEngine 在 Windows 上回傳 DirectML 推論引擎。
// 目前為暫時 stub，正式實作時將載入 onnxruntime.dll 並初始化 session。
//
// 呼叫端應檢查 LoadModel() 的 error 來決定是否降級到 OpenCV-only。
func NewInferenceEngine() InferenceEngine {
	return &directmlStubEngine{}
}

// LoadModel 載入 ONNX 模型（.onnx 格式）。
// 目前為 stub，永遠回傳 ErrInferenceUnavailable。
//
// 正式實作時流程：
//  1. ModelVerifier.Verify() 驗證 SHA256（§14.6.6）
//  2. syscall.LoadDLL("onnxruntime.dll") 集中初始化（§14.6.4）
//  3. OrtCreateSession() 建立推論 session
//  4. 設定 DirectML execution provider
func (e *directmlStubEngine) LoadModel(path string) error {
	return ErrInferenceUnavailable
}

// Infer 執行一次 DirectML 推論。
// 目前為 stub，永遠回傳 ErrInferenceUnavailable。
//
// 正式實作時流程：
//  1. RGBA → OrtValue tensor 轉換
//  2. OrtRun() 執行推論（DirectML GPU 加速）
//  3. 輸出 OrtValue → Go []float32 複製（§14.6.3：unsafe 限制在 bridge 內）
//  4. 回傳 RawTensor，不做任何 YOLO 後處理
func (e *directmlStubEngine) Infer(rgba []byte, width, height int) (RawTensor, error) {
	return RawTensor{}, ErrInferenceUnavailable
}

// Close 釋放 DirectML / ONNX Runtime 資源。
// 目前為 stub，僅標記已關閉。
//
// 正式實作時：OrtReleaseSession() + OrtReleaseEnv()。
func (e *directmlStubEngine) Close() error {
	e.closed = true
	return nil
}

// InferenceBackendName 回傳此平台的推論後端名稱。
func InferenceBackendName() string {
	return "directml"
}
