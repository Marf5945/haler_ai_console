// visual_learning/yolo_postprocess.go — 純 Go YOLOX 後處理（共用層）。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.6.7
//
// 此檔案是「換 YOLO 版本時唯一需要修改的檔案」。
// 平台橋接層（CoreML / DirectML）完全不含 YOLO 業務邏輯，
// 只回傳 RawTensor（浮點數陣列 + shape），所有解碼在這裡完成。
//
// 模型：YOLOX（Apache-2.0，anchor-free）。
//
//	來源：https://github.com/Megvii-BaseDetection/YOLOX
//	選用 YOLOX 是為了避免 Ultralytics YOLOv5/v8 的 AGPL-3.0 授權，
//	讓本專案維持 Apache-2.0 相容。
//
// 管線流程：
//
//	RawTensor → grid 解碼（anchor-free）→ objectness×class_score 過濾
//	→ 座標轉換（模型空間 → 原圖相對座標）→ NMS → []RegionProposal
//
// 安全規則（§14.3）：
//   - 此層只產出 RegionProposal（BBox + raw_score + proposal_id）
//   - 不做語義判斷（不標記「這是刪除按鈕」）
//   - 語義判斷需 DOM + OCR + LLM + 風險規則共同決定
package visual_learning

import (
	"fmt"
	"math"
	"sort"
)

// ──────────────────────────────────────────────
// YOLOX 模型參數（換版本時修改這裡）
// ──────────────────────────────────────────────

// YOLOConfig 定義 YOLOX 模型的解碼參數。
// YOLOX 是 anchor-free：每個 grid cell 只產生一個預測，無需 anchor 表。
// 換 YOLOX 尺寸（nano → tiny → s）時，通常只需確認 InputSize / Strides。
type YOLOConfig struct {
	// InputSize 是模型輸入的正方形邊長（像素），例如 416。
	InputSize int

	// NumClasses 是模型的分類數量，例如 COCO 為 80。
	NumClasses int

	// ConfThreshold 是 objectness × class_score 的最低門檻。
	// 低於此值的候選框直接丟棄。
	ConfThreshold float32

	// NMSThreshold 是 NMS 的 IoU 門檻。
	// 重疊度超過此值的框，只保留最高分。
	NMSThreshold float32

	// Strides 是每個 detection level 的步幅。
	// YOLOX 預設為 [8, 16, 32]，對應三個 feature map。
	// anchor-free：每個 stride level 的每個 grid cell 只有 1 個預測。
	Strides []int
}

// DefaultYOLOXNanoConfig 是 YOLOX-Nano 的預設解碼參數。
// 模型輸入 416×416，3 個 detection level（stride 8/16/32），COCO 80 類。
//
// 對應的權重檔需自行用 YOLOX 官方 repo（Apache-2.0）訓練 / 匯出為
// .onnx（Windows）或 .mlmodelc（macOS），放到 assets/models/yolox_nano。
var DefaultYOLOXNanoConfig = YOLOConfig{
	InputSize:     416,
	NumClasses:    80,
	ConfThreshold: 0.25,
	NMSThreshold:  0.45,
	Strides:       []int{8, 16, 32},
}

// DefaultYOLOXButtonSConfig is the app-local button detector configuration.
//
// The training exp at external/YOLOX/exps/example/custom/yolox_button_s.py uses
// YOLOX-S geometry, 640x640 test size, and one class: "button". Keep this config
// in the Go postprocess layer so the runtime can consume an exported ONNX/CoreML
// model without importing the YOLOX Python package into ui-console.
var DefaultYOLOXButtonSConfig = YOLOConfig{
	InputSize:     640,
	NumClasses:    1,
	ConfThreshold: 0.20,
	NMSThreshold:  0.45,
	Strides:       []int{8, 16, 32},
}

// ──────────────────────────────────────────────
// 內部型別
// ──────────────────────────────────────────────

// rawDetection 是解碼後、NMS 前的單一偵測結果（內部使用）。
type rawDetection struct {
	cx, cy, w, h float32 // 中心座標 + 寬高（模型輸入空間，像素）
	confidence   float32 // objectness × best_class_score
	classID      int     // 最高分的 class index
	proposalID   string  // 唯一識別碼
}

// ──────────────────────────────────────────────
// 主函式：DecodeYOLOOutput
// ──────────────────────────────────────────────

// DecodeYOLOOutput 將 RawTensor 解碼為 []RegionProposal。
//
// 這是 YOLOX 後處理的唯一入口。平台橋接層只需回傳 RawTensor，
// 所有 YOLOX 特定的解碼邏輯都在這裡。
//
// 解碼假設（YOLOX anchor-free，未在模型內部 decode）：
//   - entry[0..3] = tx, ty, tw, th（grid 相對偏移 + log-scale 尺寸）
//   - entry[4]    = objectness logit
//   - entry[5..]  = 各 class 的 logit
//
// grid 解碼：
//
//	cx = (tx + gx) * stride
//	cy = (ty + gy) * stride
//	w  = exp(tw) * stride
//	h  = exp(th) * stride
//
// 若你的匯出流程已在模型內部完成 decode（boxes 直接是像素座標、
// obj/cls 已過 sigmoid），請對應調整此函式。
//
// 參數：
//   - raw: 推論引擎回傳的原始浮點數陣列 + shape
//   - origWidth, origHeight: 原始影像的實際像素尺寸（保留供未來座標還原）
//   - config: YOLOX 模型參數（換版本時只改 config）
//
// 回傳：
//   - []RegionProposal: 已過濾 + NMS 後的候選區域
//   - error: tensor 格式不符時回傳錯誤
func DecodeYOLOOutput(raw RawTensor, origWidth, origHeight int, config YOLOConfig) ([]RegionProposal, error) {
	_ = origWidth  // 目前以模型輸入空間正規化到 0~1，原圖尺寸保留供未來使用
	_ = origHeight // 同上

	// 驗證 tensor 格式
	if err := raw.Validate(); err != nil {
		return nil, fmt.Errorf("yolox postprocess: %w", err)
	}

	// 每個候選框的 entry 長度 = 4 (bbox) + 1 (objectness) + NumClasses
	entryLen := 5 + config.NumClasses

	// 預期的 grid 預測總數（anchor-free：每 cell 1 個，所有 level 加總）
	expectedCells := 0
	for _, stride := range config.Strides {
		gridSize := config.InputSize / stride
		expectedCells += gridSize * gridSize
	}

	// 驗證 shape：預期 [1, expectedCells, entryLen] 或 [expectedCells, entryLen]
	totalExpected := expectedCells * entryLen
	if len(raw.Data) < totalExpected {
		return nil, fmt.Errorf("yolox postprocess: tensor data too short — got %d floats, expected at least %d (%d cells × %d entries)",
			len(raw.Data), totalExpected, expectedCells, entryLen)
	}

	// Step 1: 解碼所有 grid cell → rawDetection
	var candidates []rawDetection
	offset := 0
	proposalIdx := 0

	for _, stride := range config.Strides {
		gridSize := config.InputSize / stride

		// YOLOX flatten 順序：row-major（gy 外層、gx 內層），每 cell 一個預測。
		for gy := 0; gy < gridSize; gy++ {
			for gx := 0; gx < gridSize; gx++ {
				if offset+entryLen > len(raw.Data) {
					break
				}

				entry := raw.Data[offset : offset+entryLen]
				offset += entryLen

				// anchor-free grid 解碼
				cx := (entry[0] + float32(gx)) * float32(stride)
				cy := (entry[1] + float32(gy)) * float32(stride)
				w := float32(math.Exp(float64(entry[2]))) * float32(stride)
				h := float32(math.Exp(float64(entry[3]))) * float32(stride)

				objectness := sigmoid(entry[4])

				// Step 2: 找最高 class score
				bestClassScore := float32(0)
				bestClassID := 0
				for c := 0; c < config.NumClasses; c++ {
					score := sigmoid(entry[5+c])
					if score > bestClassScore {
						bestClassScore = score
						bestClassID = c
					}
				}

				// 最終信心 = objectness × best_class_score
				confidence := objectness * bestClassScore

				// Step 3: 門檻過濾
				if confidence < config.ConfThreshold {
					continue
				}

				candidates = append(candidates, rawDetection{
					cx:         cx,
					cy:         cy,
					w:          w,
					h:          h,
					confidence: confidence,
					classID:    bestClassID,
					proposalID: fmt.Sprintf("yolo-%d", proposalIdx),
				})
				proposalIdx++
			}
		}
	}

	// Step 4: NMS（非極大值抑制）
	kept := nms(candidates, config.NMSThreshold)

	// Step 5: 轉換為 []RegionProposal（對齊現有型別）
	// 座標從模型輸入空間（例如 416×416 像素）轉為原圖相對座標（0.0~1.0）
	inputSize := float32(config.InputSize)
	proposals := make([]RegionProposal, 0, len(kept))
	for _, det := range kept {
		// 中心座標 → 左上角座標，並正規化到 0.0~1.0
		x := (det.cx - det.w/2) / inputSize
		y := (det.cy - det.h/2) / inputSize
		w := det.w / inputSize
		h := det.h / inputSize

		// Clamp 到 [0, 1] 範圍
		x = clampf(x, 0, 1)
		y = clampf(y, 0, 1)
		w = clampf(w, 0, 1-x)
		h = clampf(h, 0, 1-y)

		proposals = append(proposals, RegionProposal{
			BBox: BBox{
				X: float64(x),
				Y: float64(y),
				W: float64(w),
				H: float64(h),
			},
			RawScore:   float64(det.confidence),
			ProposalID: det.proposalID,
		})
	}

	return proposals, nil
}

// ──────────────────────────────────────────────
// NMS（非極大值抑制）
// ──────────────────────────────────────────────

// nms 執行 Non-Maximum Suppression。
// 按 confidence 降序排列，高分候選保留，
// 與已保留候選 IoU ≥ threshold 的低分候選被抑制。
func nms(detections []rawDetection, iouThreshold float32) []rawDetection {
	if len(detections) == 0 {
		return nil
	}

	// 按 confidence 降序排列
	sort.Slice(detections, func(i, j int) bool {
		return detections[i].confidence > detections[j].confidence
	})

	kept := make([]rawDetection, 0, len(detections))
	suppressed := make([]bool, len(detections))

	for i := 0; i < len(detections); i++ {
		if suppressed[i] {
			continue
		}
		kept = append(kept, detections[i])

		// 抑制與 detections[i] 高度重疊的後續候選
		for j := i + 1; j < len(detections); j++ {
			if suppressed[j] {
				continue
			}
			if computeIoU(detections[i], detections[j]) >= iouThreshold {
				suppressed[j] = true
			}
		}
	}

	return kept
}

// computeIoU 計算兩個偵測框的 Intersection over Union。
// 輸入為中心座標格式 (cx, cy, w, h)。
func computeIoU(a, b rawDetection) float32 {
	// 轉為角座標
	ax1 := a.cx - a.w/2
	ay1 := a.cy - a.h/2
	ax2 := a.cx + a.w/2
	ay2 := a.cy + a.h/2

	bx1 := b.cx - b.w/2
	by1 := b.cy - b.h/2
	bx2 := b.cx + b.w/2
	by2 := b.cy + b.h/2

	// 交集區域
	ix1 := maxf(ax1, bx1)
	iy1 := maxf(ay1, by1)
	ix2 := minf(ax2, bx2)
	iy2 := minf(ay2, by2)

	iw := maxf(0, ix2-ix1)
	ih := maxf(0, iy2-iy1)
	intersection := iw * ih

	// 聯集區域
	aArea := a.w * a.h
	bArea := b.w * b.h
	union := aArea + bArea - intersection

	if union <= 0 {
		return 0
	}
	return intersection / union
}

// ──────────────────────────────────────────────
// 數學輔助函式
// ──────────────────────────────────────────────

// sigmoid 計算 sigmoid 函式：1 / (1 + exp(-x))。
func sigmoid(x float32) float32 {
	return float32(1.0 / (1.0 + math.Exp(float64(-x))))
}

// clampf 將值限制在 [lo, hi] 範圍內。
func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// minf 回傳兩個 float32 的較小值。
func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// maxf 回傳兩個 float32 的較大值。
func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
