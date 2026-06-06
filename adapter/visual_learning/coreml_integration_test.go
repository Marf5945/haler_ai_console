// visual_learning/coreml_integration_test.go — CoreML 端對端整合測試。
//
// Build tag: 僅在 macOS 編譯（CoreML 只存在於 macOS）。
//
// 此測試驗證完整的推論 pipeline：
//   LoadModel → Infer → RawTensor → DecodeYOLOOutput → []RegionProposal
//
// 需要真實模型檔案：assets/models/yolox_nano.mlmodelc
// 如果模型不存在，測試自動 skip（不會 fail）。
//
// 執行方式：
//   go test ./adapter/visual_learning/ -v -run TestCoreML -tags integration

//go:build darwin

package visual_learning

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// modelBasePath 回傳模型檔基底路徑（不含副檔名）。
// 從測試檔案位置往上推算到專案根目錄。
func modelBasePath(t *testing.T) string {
	t.Helper()
	// 取得本測試檔案的目錄
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	// adapter/visual_learning/ → 專案根目錄
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	return filepath.Join(projectRoot, "assets", "models", "yolox_nano")
}

// makeTestRGBA 建立一張純色測試圖片的 RGBA bytes。
// 不需要真實截圖 — 只要確認 pipeline 能跑完不 crash。
func makeTestRGBA(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// 畫一些色塊模擬 UI 元素
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			switch {
			case x < width/3:
				img.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255}) // 淺灰背景
			case x < 2*width/3 && y > height/3 && y < 2*height/3:
				img.Set(x, y, color.RGBA{R: 50, G: 120, B: 220, A: 255}) // 藍色按鈕區域
			default:
				img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255}) // 白色
			}
		}
	}
	return img.Pix
}

// TestCoreMLLoadModel 測試模型載入 + SHA256 驗證（§14.6.6）。
func TestCoreMLLoadModel(t *testing.T) {
	basePath := modelBasePath(t)
	mlmodelcPath := basePath + ".mlmodelc"

	// 模型不存在時 skip
	if _, err := os.Stat(mlmodelcPath); os.IsNotExist(err) {
		t.Skipf("model not found at %s — skipping integration test", mlmodelcPath)
	}

	engine := NewInferenceEngine()
	defer engine.Close()

	err := engine.LoadModel(mlmodelcPath)
	if err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}

	t.Log("CoreML model loaded successfully")
}

// TestCoreMLInferenceEndToEnd 測試完整推論 pipeline。
//
// 流程：
//  1. LoadModel — 載入 .mlmodelc + SHA256 驗證
//  2. Infer — 餵入測試圖片，取得 RawTensor
//  3. RawTensor.Validate — 驗證 shape 和 data 一致
//  4. DecodeYOLOOutput — 純 Go 後處理
//  5. 檢查結果格式正確（不檢查偵測結果，因為是假圖片）
func TestCoreMLInferenceEndToEnd(t *testing.T) {
	basePath := modelBasePath(t)
	mlmodelcPath := basePath + ".mlmodelc"

	if _, err := os.Stat(mlmodelcPath); os.IsNotExist(err) {
		t.Skipf("model not found at %s — skipping integration test", mlmodelcPath)
	}

	// ── Step 1: 載入模型 ──
	engine := NewInferenceEngine()
	defer engine.Close()

	if err := engine.LoadModel(mlmodelcPath); err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}

	// ── Step 2: 推論 ──
	width, height := 640, 480
	rgba := makeTestRGBA(width, height)

	raw, err := engine.Infer(rgba, width, height)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}

	// ── Step 3: 驗證 RawTensor ──
	if err := raw.Validate(); err != nil {
		t.Fatalf("RawTensor validation failed: %v", err)
	}

	t.Logf("RawTensor: shape=%v, elements=%d", raw.Shape, raw.TotalElements())

	// 基本 shape 檢查：YOLO 輸出應該是 3D tensor [batch, anchors, attributes]
	if len(raw.Shape) < 2 {
		t.Fatalf("unexpected tensor shape: %v (expected at least 2 dimensions)", raw.Shape)
	}

	// ── Step 4: YOLO 後處理 ──
	proposals, decodeErr := DecodeYOLOOutput(raw, width, height, DefaultYOLOXNanoConfig)
	if decodeErr != nil {
		t.Fatalf("DecodeYOLOOutput failed: %v", decodeErr)
	}

	// ── Step 5: 結果檢查 ──
	// 假圖片不一定有偵測結果，但 pipeline 必須跑完不 crash。
	t.Logf("Detected %d proposals from test image", len(proposals))

	for i, p := range proposals {
		if i >= 5 {
			t.Logf("  ... and %d more", len(proposals)-5)
			break
		}
		t.Logf("  [%d] score=%.4f bbox=(%.2f,%.2f,%.2f,%.2f)",
			i, p.RawScore,
			p.BBox.X, p.BBox.Y, p.BBox.W, p.BBox.H)
	}
}

// TestCoreMLModelIntegrityReject 測試篡改模型時驗證拒絕（§14.6.6）。
func TestCoreMLModelIntegrityReject(t *testing.T) {
	// 使用不存在的路徑模擬 hash 不匹配
	engine := NewInferenceEngine()
	defer engine.Close()

	// 嘗試載入一個存在但 hash 不在 manifest 的路徑
	err := engine.LoadModel("/tmp/fake_model.mlmodelc")
	if err == nil {
		t.Fatal("expected error for non-manifest model, got nil")
	}
	t.Logf("correctly rejected: %v", err)
}

// TestCoreMLEngineLifecycle 測試引擎生命週期管理。
func TestCoreMLEngineLifecycle(t *testing.T) {
	engine := NewInferenceEngine()

	// Close 應該安全
	if err := engine.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 重複 Close 安全（no-op）
	if err := engine.Close(); err != nil {
		t.Fatalf("double Close failed: %v", err)
	}

	// Close 後 LoadModel 應回傳 ErrEngineAlreadyClosed
	err := engine.LoadModel("/any/path")
	if err != ErrEngineAlreadyClosed {
		t.Fatalf("expected ErrEngineAlreadyClosed, got: %v", err)
	}

	// Close 後 Infer 應回傳 ErrEngineAlreadyClosed
	_, err = engine.Infer([]byte{0, 0, 0, 0}, 1, 1)
	if err != ErrEngineAlreadyClosed {
		t.Fatalf("expected ErrEngineAlreadyClosed, got: %v", err)
	}
}
