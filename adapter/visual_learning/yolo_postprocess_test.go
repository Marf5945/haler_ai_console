// visual_learning/yolo_postprocess_test.go — YOLOX 後處理單元測試。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.6.5
//
// 測試資料使用 //go:embed 從 testdata/ 載入固定 tensor fixture。
// 換 YOLO 版本或 tensor shape 時，只需替換 fixture 檔案。
package visual_learning

import (
	_ "embed"
	"encoding/json"
	"math"
	"testing"
)

// ──────────────────────────────────────────────
// Test Fixture 載入（§14.6.5：用 //go:embed，不硬編碼 float array）
// ──────────────────────────────────────────────

//go:embed testdata/yolox_nano_output.json
var testFixtureJSON []byte

// testFixture 是從 testdata/yolox_nano_output.json 解析的測試資料。
// YOLOX 為 anchor-free，fixture 不含 anchor 表。
type testFixture struct {
	Description string `json:"description"`
	Tensor      struct {
		Data  []float32 `json:"data"`
		Shape []int     `json:"shape"`
	} `json:"tensor"`
	Config struct {
		InputSize     int     `json:"input_size"`
		NumClasses    int     `json:"num_classes"`
		ConfThreshold float32 `json:"conf_threshold"`
		NMSThreshold  float32 `json:"nms_threshold"`
		Strides       []int   `json:"strides"`
	} `json:"config"`
	Expected struct {
		TotalBeforeNMS       int    `json:"total_before_nms"`
		TotalAfterNMS        int    `json:"total_after_nms"`
		KeptProposalID       string `json:"kept_proposal_id"`
		SuppressedProposalID string `json:"suppressed_proposal_id"`
	} `json:"expected"`
}

// loadFixture 解析 test fixture JSON。
func loadFixture(t *testing.T) testFixture {
	t.Helper()
	var f testFixture
	if err := json.Unmarshal(testFixtureJSON, &f); err != nil {
		t.Fatalf("failed to parse test fixture: %v", err)
	}
	return f
}

// fixtureToConfig 將 fixture 的 config 轉為 YOLOConfig（anchor-free）。
func fixtureToConfig(f testFixture) YOLOConfig {
	return YOLOConfig{
		InputSize:     f.Config.InputSize,
		NumClasses:    f.Config.NumClasses,
		ConfThreshold: f.Config.ConfThreshold,
		NMSThreshold:  f.Config.NMSThreshold,
		Strides:       f.Config.Strides,
	}
}

// ──────────────────────────────────────────────
// TestDecodeYOLOOutput — 驗證完整解碼管線
// ──────────────────────────────────────────────

func TestDecodeYOLOOutput(t *testing.T) {
	f := loadFixture(t)
	config := fixtureToConfig(f)

	raw := RawTensor{
		Data:  f.Tensor.Data,
		Shape: f.Tensor.Shape,
	}

	proposals, err := DecodeYOLOOutput(raw, 1920, 1080, config)
	if err != nil {
		t.Fatalf("DecodeYOLOOutput returned error: %v", err)
	}

	// 預期：NMS 後剩下 fixture 指定數量的 proposal
	if len(proposals) != f.Expected.TotalAfterNMS {
		t.Errorf("expected %d proposals after NMS, got %d", f.Expected.TotalAfterNMS, len(proposals))
	}

	// 驗證保留的第一個（最高分）是預期的高分候選
	if len(proposals) > 0 && proposals[0].ProposalID != f.Expected.KeptProposalID {
		t.Errorf("expected kept proposal ID %q, got %q", f.Expected.KeptProposalID, proposals[0].ProposalID)
	}

	// 驗證被抑制的候選不應出現在結果中
	for _, p := range proposals {
		if p.ProposalID == f.Expected.SuppressedProposalID {
			t.Errorf("proposal %q should have been suppressed by NMS, but survived", p.ProposalID)
		}
	}

	// 驗證座標在 [0, 1] 範圍內
	for _, p := range proposals {
		if p.BBox.X < 0 || p.BBox.X > 1 || p.BBox.Y < 0 || p.BBox.Y > 1 {
			t.Errorf("proposal %s has out-of-range coordinates: %+v", p.ProposalID, p.BBox)
		}
		if p.BBox.W < 0 || p.BBox.W > 1 || p.BBox.H < 0 || p.BBox.H > 1 {
			t.Errorf("proposal %s has out-of-range dimensions: %+v", p.ProposalID, p.BBox)
		}
	}
}

// ──────────────────────────────────────────────
// TestNMS — 驗證非極大值抑制
// ──────────────────────────────────────────────

func TestNMS(t *testing.T) {
	// 建立 3 個候選：2 個高度重疊（IoU > 0.45），1 個不重疊
	detections := []rawDetection{
		{cx: 100, cy: 100, w: 50, h: 50, confidence: 0.9, proposalID: "high"},
		{cx: 105, cy: 105, w: 50, h: 50, confidence: 0.7, proposalID: "overlap-suppressed"},
		{cx: 300, cy: 300, w: 50, h: 50, confidence: 0.8, proposalID: "separate-kept"},
	}

	kept := nms(detections, 0.45)

	// 預期保留 2 個：high + separate-kept
	if len(kept) != 2 {
		t.Fatalf("expected 2 after NMS, got %d", len(kept))
	}

	// 驗證 "overlap-suppressed" 被抑制
	for _, d := range kept {
		if d.proposalID == "overlap-suppressed" {
			t.Errorf("expected 'overlap-suppressed' to be suppressed by NMS, but it survived")
		}
	}
}

// ──────────────────────────────────────────────
// TestScoreFilter — 驗證低分候選被過濾
// ──────────────────────────────────────────────

func TestScoreFilter(t *testing.T) {
	// 建立只有 1 個低分 grid cell 的 tensor（conf < 0.25 threshold）
	entryLen := 85
	data := make([]float32, entryLen)
	data[0] = 0.5            // tx（grid 偏移）
	data[1] = 0.5            // ty
	data[2] = 0.0            // tw（log-scale）→ exp(0)=1
	data[3] = 0.0            // th
	data[4] = invSigmoid(0.1) // objectness = 0.1
	data[5] = invSigmoid(0.1) // class 0 = 0.1 → conf = 0.01

	// anchor-free config：input 8、stride 8 → 1×1 grid，1 個 cell。
	config := YOLOConfig{
		InputSize:     8,
		NumClasses:    80,
		ConfThreshold: 0.25,
		NMSThreshold:  0.45,
		Strides:       []int{8},
	}

	raw := RawTensor{Data: data, Shape: []int{1, 1, entryLen}}

	proposals, err := DecodeYOLOOutput(raw, 1920, 1080, config)
	if err != nil {
		t.Fatalf("DecodeYOLOOutput error: %v", err)
	}

	// 預期：所有候選都被 threshold 過濾，結果為空
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals (all below threshold), got %d", len(proposals))
	}
}

// ──────────────────────────────────────────────
// TestRawTensorValidate — 驗證 tensor 格式檢查
// ──────────────────────────────────────────────

func TestRawTensorValidate(t *testing.T) {
	// 正確的 tensor
	good := RawTensor{Data: make([]float32, 255), Shape: []int{1, 3, 85}}
	if err := good.Validate(); err != nil {
		t.Errorf("valid tensor should not error: %v", err)
	}

	// 資料長度不符
	bad := RawTensor{Data: make([]float32, 100), Shape: []int{1, 3, 85}}
	if err := bad.Validate(); err == nil {
		t.Error("tensor with wrong data length should return error")
	}

	// 空 shape
	empty := RawTensor{Data: nil, Shape: nil}
	if n := empty.TotalElements(); n != 0 {
		t.Errorf("empty shape should have 0 elements, got %d", n)
	}
}

// ──────────────────────────────────────────────
// TestComputeIoU — 驗證 IoU 計算
// ──────────────────────────────────────────────

func TestComputeIoU(t *testing.T) {
	// 完全重疊
	a := rawDetection{cx: 50, cy: 50, w: 20, h: 20}
	b := rawDetection{cx: 50, cy: 50, w: 20, h: 20}
	iou := computeIoU(a, b)
	if math.Abs(float64(iou)-1.0) > 0.01 {
		t.Errorf("identical boxes should have IoU=1.0, got %f", iou)
	}

	// 完全不重疊
	c := rawDetection{cx: 50, cy: 50, w: 20, h: 20}
	d := rawDetection{cx: 200, cy: 200, w: 20, h: 20}
	iou2 := computeIoU(c, d)
	if iou2 != 0 {
		t.Errorf("non-overlapping boxes should have IoU=0, got %f", iou2)
	}
}

// ──────────────────────────────────────────────
// TestSigmoid — 驗證 sigmoid 數學正確性
// ──────────────────────────────────────────────

func TestSigmoid(t *testing.T) {
	// sigmoid(0) = 0.5
	if math.Abs(float64(sigmoid(0))-0.5) > 0.001 {
		t.Errorf("sigmoid(0) should be 0.5, got %f", sigmoid(0))
	}

	// sigmoid(large) ≈ 1.0
	if sigmoid(10) < 0.999 {
		t.Errorf("sigmoid(10) should be close to 1.0, got %f", sigmoid(10))
	}

	// sigmoid(very negative) ≈ 0.0
	if sigmoid(-10) > 0.001 {
		t.Errorf("sigmoid(-10) should be close to 0.0, got %f", sigmoid(-10))
	}
}

// ──────────────────────────────────────────────
// 測試輔助函式
// ──────────────────────────────────────────────

// invSigmoid 計算 sigmoid 的反函式（用於建構測試資料）。
func invSigmoid(y float32) float32 {
	return float32(math.Log(float64(y) / (1.0 - float64(y))))
}
