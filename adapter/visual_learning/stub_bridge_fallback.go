// visual_learning/stub_bridge_fallback.go — 非支援平台的 InferenceEngine 降級實作。
//
// Build tag: 只在非 macOS、非 Windows 的平台編譯（例如 Linux）。
// 所有方法回傳 ErrInferenceUnavailable，符合 §14.6.4 降級規範。
//
// 降級規則（§14.5 + §14.6.4）：
//   - 不增加信心分
//   - 不跳過 dry-run
//   - 不繞過 review
//   - UI 顯示「硬體推論不可用」
//
// 此檔案不含任何 unsafe 操作（§14.6.3）。

//go:build !darwin && !windows

package visual_learning

// stubEngine 是非支援平台的 InferenceEngine stub。
// 所有推論相關方法回傳 ErrInferenceUnavailable。
type stubEngine struct {
	closed bool
}

// NewInferenceEngine 在非支援平台回傳 stub 引擎。
// 呼叫端應檢查 LoadModel() 的 error 來決定是否降級到 OpenCV-only。
func NewInferenceEngine() InferenceEngine {
	return &stubEngine{}
}

// LoadModel 在 stub 引擎上永遠回傳 ErrInferenceUnavailable。
// 呼叫端收到此 error 後應降級為 OpenCV pipeline。
func (s *stubEngine) LoadModel(path string) error {
	return ErrInferenceUnavailable
}

// Infer 在 stub 引擎上永遠回傳 ErrInferenceUnavailable。
func (s *stubEngine) Infer(rgba []byte, width, height int) (RawTensor, error) {
	return RawTensor{}, ErrInferenceUnavailable
}

// Close 在 stub 引擎上為 no-op（沒有需要釋放的 native 資源）。
func (s *stubEngine) Close() error {
	s.closed = true
	return nil
}

// InferenceBackendName 回傳 stub 引擎的後端名稱。
func InferenceBackendName() string {
	return "stub"
}
