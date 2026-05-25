// visual_learning/opencv_pipeline.go — 精簡視覺管線（純 Go，零 CGo 依賴）。
// 管線：RGBA → 灰階 → 自適應閾值 → Sobel 邊緣 → 區域提取 → 合併 → hash → UIFingerprint
//
// 安全規則：
//   - 管線僅提議候選區域，不做語意判斷
//   - 不會判定「這是提交按鈕」或「這是付款按鈕」
//   - 語意判斷需要 DOM + OCR + LLM + 風險規則共同決定
//   - Degraded 模式不增加信心分，不跳過 dry-run
//
// 便攜性：純 Go 標準庫實作，無 CGo/OpenCV 依賴，二進位增量 ~0MB
package visual_learning

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"
)

// ──────────────────────────────────────────────
// 管線結構
// ──────────────────────────────────────────────

// OpenCVPipeline 提供 UI 區域提案的影像管線。
// 使用純 Go 實作核心演算法，無需 OpenCV shared lib。
type OpenCVPipeline struct {
	available bool
	config    PipelineConfig
}

// PipelineConfig 管線參數。
type PipelineConfig struct {
	CannyLow       float64 // Sobel 邊緣低閾值（預設 50）
	CannyHigh      float64 // Sobel 邊緣高閾值（預設 150）
	MinRegionArea  int     // 最小區域面積（像素²，預設 400）
	MaxRegionArea  int     // 最大區域面積（佔總面積百分比 × 100，預設 40）
	MergeThreshold int     // 合併距離閾值（像素，預設 8）
}

// DefaultPipelineConfig 預設管線參數。
var DefaultPipelineConfig = PipelineConfig{
	CannyLow:       50,
	CannyHigh:      150,
	MinRegionArea:  400,
	MaxRegionArea:  40,
	MergeThreshold: 8,
}

// PipelineResult 管線輸出。
type PipelineResult struct {
	Candidates []UIFingerprint `json:"candidates"`
	Degraded   bool            `json:"degraded"`
	Reason     string          `json:"reason,omitempty"`
}

// ──────────────────────────────────────────────
// 初始化
// ──────────────────────────────────────────────

// NewOpenCVPipeline 建立純 Go 影像管線（永遠可用，無外部依賴）。
func NewOpenCVPipeline() *OpenCVPipeline {
	return &OpenCVPipeline{
		available: true,
		config:    DefaultPipelineConfig,
	}
}

// IsAvailable 回傳管線是否可用（純 Go 實作永遠為 true）。
func (p *OpenCVPipeline) IsAvailable() bool {
	return p.available
}

// ──────────────────────────────────────────────
// 主管線
// ──────────────────────────────────────────────

// Propose 執行完整管線，回傳候選區域 fingerprint。
// imageData: RGBA 原始位元組（4 bytes per pixel）
// width, height: 影像尺寸
func (p *OpenCVPipeline) Propose(imageData []byte, width, height int) PipelineResult {
	if !p.available {
		return PipelineResult{Degraded: true, Reason: "pipeline disabled"}
	}
	expectedLen := width * height * 4
	if len(imageData) < expectedLen {
		return PipelineResult{Degraded: true, Reason: fmt.Sprintf("image data too short: expected %d, got %d", expectedLen, len(imageData))}
	}

	// Step 1: RGBA → 灰階
	gray := rgbaToGray(imageData, width, height)

	// Step 2: Sobel 邊緣偵測（簡化版 Canny）
	edges := sobelEdge(gray, width, height, p.config.CannyLow, p.config.CannyHigh)

	// Step 3: 連通區域提取
	regions := extractRegions(edges, width, height)

	// Step 4: 過濾過小/過大區域
	totalArea := width * height
	maxArea := totalArea * p.config.MaxRegionArea / 100
	var filtered []regionRect
	for _, r := range regions {
		area := r.w * r.h
		if area >= p.config.MinRegionArea && area <= maxArea {
			filtered = append(filtered, r)
		}
	}

	// Step 5: 合併鄰近區域
	merged := mergeRegions(filtered, p.config.MergeThreshold)

	// Step 6: 轉換為 UIFingerprint（含 hash）
	var candidates []UIFingerprint
	for i, r := range merged {
		fp := regionToFingerprint(r, gray, width, height, i)
		candidates = append(candidates, fp)
	}

	return PipelineResult{Candidates: candidates, Degraded: false}
}

// ──────────────────────────────────────────────
// 影像處理核心（純 Go）
// ──────────────────────────────────────────────

// rgbaToGray 將 RGBA 轉為灰階（ITU-R BT.601 權重）。
func rgbaToGray(data []byte, w, h int) []byte {
	gray := make([]byte, w*h)
	for i := 0; i < w*h; i++ {
		offset := i * 4
		r := float64(data[offset])
		g := float64(data[offset+1])
		b := float64(data[offset+2])
		gray[i] = byte(0.299*r + 0.587*g + 0.114*b)
	}
	return gray
}

// sobelEdge 執行 Sobel 邊緣偵測 + 雙閾值過濾。
func sobelEdge(gray []byte, w, h int, low, high float64) []byte {
	edges := make([]byte, w*h)
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			// Sobel 3×3 核心
			gx := -float64(gray[(y-1)*w+(x-1)]) + float64(gray[(y-1)*w+(x+1)]) +
				-2*float64(gray[y*w+(x-1)]) + 2*float64(gray[y*w+(x+1)]) +
				-float64(gray[(y+1)*w+(x-1)]) + float64(gray[(y+1)*w+(x+1)])
			gy := -float64(gray[(y-1)*w+(x-1)]) - 2*float64(gray[(y-1)*w+x]) - float64(gray[(y-1)*w+(x+1)]) +
				float64(gray[(y+1)*w+(x-1)]) + 2*float64(gray[(y+1)*w+x]) + float64(gray[(y+1)*w+(x+1)])
			mag := math.Sqrt(gx*gx + gy*gy)
			if mag >= high {
				edges[y*w+x] = 255 // 強邊緣
			} else if mag >= low {
				edges[y*w+x] = 128 // 弱邊緣
			}
		}
	}

	// 弱邊緣連接：僅保留與強邊緣相鄰的弱邊緣
	result := make([]byte, w*h)
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			if edges[y*w+x] == 255 {
				result[y*w+x] = 255
			} else if edges[y*w+x] == 128 {
				// 檢查 8 鄰域是否有強邊緣
				hasStrong := false
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if edges[(y+dy)*w+(x+dx)] == 255 {
							hasStrong = true
						}
					}
				}
				if hasStrong {
					result[y*w+x] = 255
				}
			}
		}
	}
	return result
}

// regionRect 矩形區域（像素座標）。
type regionRect struct {
	x, y, w, h int
}

// extractRegions 從邊緣圖提取連通區域的外接矩形。
// 使用簡化的 flood-fill 方式。
func extractRegions(edges []byte, w, h int) []regionRect {
	visited := make([]bool, w*h)
	var regions []regionRect

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if edges[y*w+x] == 255 && !visited[y*w+x] {
				// Flood-fill 找連通區域
				minX, minY, maxX, maxY := x, y, x, y
				stack := []int{y*w + x}
				visited[y*w+x] = true

				for len(stack) > 0 {
					idx := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					cy, cx := idx/w, idx%w

					if cx < minX {
						minX = cx
					}
					if cx > maxX {
						maxX = cx
					}
					if cy < minY {
						minY = cy
					}
					if cy > maxY {
						maxY = cy
					}

					// 4-鄰域擴展
					neighbors := [][2]int{{cx - 1, cy}, {cx + 1, cy}, {cx, cy - 1}, {cx, cy + 1}}
					for _, n := range neighbors {
						nx, ny := n[0], n[1]
						if nx >= 0 && nx < w && ny >= 0 && ny < h {
							ni := ny*w + nx
							if edges[ni] == 255 && !visited[ni] {
								visited[ni] = true
								stack = append(stack, ni)
							}
						}
					}
				}

				rw := maxX - minX + 1
				rh := maxY - minY + 1
				if rw > 2 && rh > 2 {
					regions = append(regions, regionRect{x: minX, y: minY, w: rw, h: rh})
				}
			}
		}
	}
	return regions
}

// mergeRegions 合併距離小於 threshold 的鄰近區域。
func mergeRegions(regions []regionRect, threshold int) []regionRect {
	if len(regions) == 0 {
		return nil
	}
	merged := make([]bool, len(regions))
	var result []regionRect

	for i := 0; i < len(regions); i++ {
		if merged[i] {
			continue
		}
		r := regions[i]
		// 嘗試與後續區域合併
		for j := i + 1; j < len(regions); j++ {
			if merged[j] {
				continue
			}
			if regionsClose(r, regions[j], threshold) {
				r = mergeTwo(r, regions[j])
				merged[j] = true
			}
		}
		result = append(result, r)
	}
	return result
}

// regionsClose 檢查兩區域是否距離小於 threshold。
func regionsClose(a, b regionRect, threshold int) bool {
	dx := maxInt(a.x, b.x) - minInt(a.x+a.w, b.x+b.w)
	dy := maxInt(a.y, b.y) - minInt(a.y+a.h, b.y+b.h)
	if dx < 0 {
		dx = 0
	}
	if dy < 0 {
		dy = 0
	}
	return dx <= threshold && dy <= threshold
}

// mergeTwo 合併兩個矩形為外接矩形。
func mergeTwo(a, b regionRect) regionRect {
	x := minInt(a.x, b.x)
	y := minInt(a.y, b.y)
	x2 := maxInt(a.x+a.w, b.x+b.w)
	y2 := maxInt(a.y+a.h, b.y+b.h)
	return regionRect{x: x, y: y, w: x2 - x, h: y2 - y}
}

// ──────────────────────────────────────────────
// 區域 → UIFingerprint 轉換
// ──────────────────────────────────────────────

// regionToFingerprint 將區域轉為 UIFingerprint（含 hash）。
func regionToFingerprint(r regionRect, gray []byte, imgW, imgH, idx int) UIFingerprint {
	// 計算區域的灰階 hash
	regionHash := computeRegionHash(gray, imgW, r)

	// 正規化座標（0.0–1.0）
	bbox := BBox{
		X: float64(r.x) / float64(imgW),
		Y: float64(r.y) / float64(imgH),
		W: float64(r.w) / float64(imgW),
		H: float64(r.h) / float64(imgH),
	}

	return UIFingerprint{
		RegionID:              fmt.Sprintf("region-%d-%d", time.Now().UnixNano(), idx),
		BBoxRelative:          bbox,
		Source:                "pure_go_pipeline",
		ShapeHash:             regionHash,
		EdgeHash64:            regionHash[:16],
		Confidence:            0.5, // 基礎信心（需後續 LLM 提升）
		ReadablePatchExported: false,
		CreatedAt:             time.Now(),
	}
}

// computeRegionHash 計算區域灰階值的 SHA-256 hash。
func computeRegionHash(gray []byte, imgW int, r regionRect) string {
	h := sha256.New()
	for y := r.y; y < r.y+r.h && y < len(gray)/imgW; y++ {
		start := y*imgW + r.x
		end := start + r.w
		if end > len(gray) {
			end = len(gray)
		}
		if start < len(gray) {
			h.Write(gray[start:end])
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ──────────────────────────────────────────────
// 輔助函式
// ──────────────────────────────────────────────

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
