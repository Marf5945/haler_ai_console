// w3a_media/segment.go — §9A.3 Overall + Segment 雙層結構（自適應分割）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 媒體指紋不只記錄整體，還要記錄段落級指紋。                  │
// │ 段落分割讓我們能偵測「哪個區域被修改了」。                  │
// │                                                             │
// │ 圖片分割（依解析度自適應）：                                │
// │  ≤1024px  → 4×4 = 16 tiles                                 │
// │  ≤4096px  → 8×8 = 64 tiles                                 │
// │  >4096px  → 16×16 = 256 tiles                              │
// │                                                             │
// │ 音訊分割（依取樣率自適應）：                                │
// │  ≤44100Hz → 每 5 秒一段                                    │
// │  >44100Hz → 每 3 秒一段                                    │
// │                                                             │
// │ 分割邏輯不讀取像素/音訊資料，僅需 metadata（寬高/取樣率）  │
// └─────────────────────────────────────────────────────────────┘
package w3a_media

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

// ──────────────────────────────────────────────
// 圖片自適應分割
// ──────────────────────────────────────────────

// SplitImageSegments 依解析度自適應決定 tile grid 大小。
// 回傳每個 tile 的區域描述。
func SplitImageSegments(width, height int) []SegmentRegion {
	gridSize := imageGridSize(width, height)
	tileW := width / gridSize
	tileH := height / gridSize

	regions := make([]SegmentRegion, 0, gridSize*gridSize)
	idx := 0
	for row := 0; row < gridSize; row++ {
		for col := 0; col < gridSize; col++ {
			x0 := col * tileW
			y0 := row * tileH
			x1 := x0 + tileW
			y1 := y0 + tileH
			// 最後一行/列吃掉餘數
			if col == gridSize-1 {
				x1 = width
			}
			if row == gridSize-1 {
				y1 = height
			}
			regions = append(regions, SegmentRegion{
				Index:       idx,
				Description: fmt.Sprintf("tile:%d,%d~%d,%d", x0, y0, x1, y1),
			})
			idx++
		}
	}
	return regions
}

// imageGridSize 依最大邊長決定 grid 大小。
func imageGridSize(width, height int) int {
	maxDim := width
	if height > maxDim {
		maxDim = height
	}
	switch {
	case maxDim <= 1024:
		return 4
	case maxDim <= 4096:
		return 8
	default:
		return 16
	}
}

// ──────────────────────────────────────────────
// 音訊自適應分割
// ──────────────────────────────────────────────

// SplitAudioSegments 依取樣率與時長自適應決定分段。
func SplitAudioSegments(sampleRate int, durationSec float64) []SegmentRegion {
	segDuration := audioSegmentDuration(sampleRate)
	numSegments := int(math.Ceil(durationSec / segDuration))
	if numSegments < 1 {
		numSegments = 1
	}

	regions := make([]SegmentRegion, numSegments)
	for i := 0; i < numSegments; i++ {
		start := float64(i) * segDuration
		end := start + segDuration
		if end > durationSec {
			end = durationSec
		}
		regions[i] = SegmentRegion{
			Index:       i,
			Description: fmt.Sprintf("%s~%s", formatAudioTime(start), formatAudioTime(end)),
		}
	}
	return regions
}

// audioSegmentDuration 依取樣率決定每段長度（秒）。
func audioSegmentDuration(sampleRate int) float64 {
	if sampleRate > 44100 {
		return 3.0
	}
	return 5.0
}

// formatAudioTime 格式化音訊時間為 MM:SS.mmm。
func formatAudioTime(sec float64) string {
	minutes := int(sec) / 60
	seconds := sec - float64(minutes*60)
	return fmt.Sprintf("%02d:%06.3f", minutes, seconds)
}

// ──────────────────────────────────────────────
// 段落指紋計算
// ──────────────────────────────────────────────

// ComputeImageSegmentFingerprints 計算圖片各 tile 的指紋。
// 每個 tile 獨立計算 byte hash + perceptual hash。
func ComputeImageSegmentFingerprints(filePath string, regions []SegmentRegion) ([]SegmentFingerprint, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	results := make([]SegmentFingerprint, len(regions))
	for i, region := range regions {
		// 解析 tile 座標
		var x0, y0, x1, y1 int
		fmt.Sscanf(region.Description, "tile:%d,%d~%d,%d", &x0, &y0, &x1, &y1)

		// 提取 tile 像素 → 計算指紋
		tileData := extractTileBytes(img, x0, y0, x1, y1)
		byteHash := ComputeByteHashFromBytes(tileData)

		// tile 的 perceptual hash：縮小到 8×8 灰階 → 簡易 hash
		tileGray := extractTileGray(img, x0, y0, x1, y1, 8)
		pHash := grayToSimpleHash(tileGray)

		results[i] = SegmentFingerprint{
			Region:         region,
			ByteHash:       byteHash,
			PerceptualHash: fmt.Sprintf("phash:%016x", pHash),
		}
	}
	return results, nil
}

// extractTileBytes 提取 tile 區域的像素原始資料。
func extractTileBytes(img image.Image, x0, y0, x1, y1 int) []byte {
	var data []byte
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			data = append(data, byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8))
		}
	}
	return data
}

// extractTileGray 提取 tile 區域並縮小到指定灰階尺寸。
func extractTileGray(img image.Image, x0, y0, x1, y1, size int) []float64 {
	tileW := x1 - x0
	tileH := y1 - y0
	result := make([]float64, size*size)

	for i := 0; i < size; i++ {
		srcY := y0 + i*tileH/size
		for j := 0; j < size; j++ {
			srcX := x0 + j*tileW/size
			r, g, b, _ := img.At(srcX, srcY).RGBA()
			gray := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
			result[i*size+j] = gray / 65535.0
		}
	}
	return result
}

// grayToSimpleHash 從 8×8 灰階值產生 64-bit hash（中位數比較）。
func grayToSimpleHash(gray []float64) uint64 {
	if len(gray) < 64 {
		return 0
	}
	median := computeMedian(gray)
	var hash uint64
	for i := 0; i < 64; i++ {
		if gray[i] > median {
			hash |= 1 << uint(63-i)
		}
	}
	return hash
}

// GetImageDimensions 讀取圖片寬高（不載入全部像素）。
func GetImageDimensions(filePath string) (int, int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	config, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return config.Width, config.Height, nil
}
