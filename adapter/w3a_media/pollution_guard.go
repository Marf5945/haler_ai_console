// w3a_media/pollution_guard.go — §9A.9 模型污染偵測（基本統計檢測）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 模型污染攻擊：看起來正常的媒體，內含隱藏信號，              │
// │ 可能干擾 LLM / VLM / OCR / 多模態訓練資料。                │
// │                                                             │
// │ 三項 Go-native 統計檢測：                                   │
// │                                                             │
// │  1. 高頻能量異常（HighFreqDetector）                        │
// │     - 圖片：計算相鄰像素差值的能量比例                      │
// │     - 異常高 → 可能有對抗性噪音                             │
// │                                                             │
// │  2. 直方圖異常（HistogramDetector）                         │
// │     - 統計值域分佈，偵測不自然峰值或平坦區                  │
// │     - 異常 → 可能有隱藏觸發模式                             │
// │                                                             │
// │  3. LSB 分佈異常（LSBDetector）                             │
// │     - 最低有效位元的 Monobit test                           │
// │     - 非隨機分佈 → 可能有隱寫術 payload                     │
// │                                                             │
// │ 加權公式：0.4 × 高頻 + 0.3 × 直方圖 + 0.3 × LSB           │
// │ 總分 > 0.7 → model_pollution_risk                           │
// │                                                             │
// │ 零外部依賴：僅使用 Go 標準庫 image + math                   │
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
// 主偵測入口
// ──────────────────────────────────────────────

// DetectPollution 執行模型污染偵測，回傳三項分數與加權結果。
func DetectPollution(filePath string, scope MediaScope) (*PollutionReport, error) {
	switch scope {
	case ScopeImage:
		return detectImagePollution(filePath)
	case ScopeAudio:
		return detectAudioPollution(filePath)
	default:
		return &PollutionReport{Details: "unsupported scope"}, nil
	}
}

// ──────────────────────────────────────────────
// 圖片污染偵測
// ──────────────────────────────────────────────

func detectImagePollution(filePath string) (*PollutionReport, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width < 2 || height < 2 {
		return &PollutionReport{Details: "image too small for analysis"}, nil
	}

	// 提取灰階像素值（0–255）
	pixels := make([][]uint8, height)
	for y := 0; y < height; y++ {
		pixels[y] = make([]uint8, width)
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			gray := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256.0
			pixels[y][x] = uint8(gray)
		}
	}

	// 檢測 1：高頻能量異常
	highFreq := detectHighFreqEnergy(pixels, width, height)

	// 檢測 2：直方圖異常
	histogram := detectHistogramAnomaly(pixels, width, height)

	// 檢測 3：LSB 分佈異常
	lsb := detectLSBAnomaly(pixels, width, height)

	// 加權總分
	weighted := 0.4*highFreq + 0.3*histogram + 0.3*lsb

	details := fmt.Sprintf("高頻能量=%.3f 直方圖=%.3f LSB=%.3f", highFreq, histogram, lsb)
	if weighted > PollutionThreshold {
		details += " → 判定為模型污染風險"
	}

	return &PollutionReport{
		HighFreqScore:   highFreq,
		HistogramScore:  histogram,
		LSBScore:        lsb,
		WeightedTotal:   weighted,
		IsPollutionRisk: weighted > PollutionThreshold,
		Details:         details,
	}, nil
}

// ──────────────────────────────────────────────
// 檢測 1：高頻能量異常
// ──────────────────────────────────────────────

// detectHighFreqEnergy 計算相鄰像素差值的能量占比。
// 自然圖片的高頻能量通常在 0.05–0.25 之間。
// 對抗性噪音會顯著提高此值。
func detectHighFreqEnergy(pixels [][]uint8, w, h int) float64 {
	var totalEnergy, highFreqEnergy float64

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			val := float64(pixels[y][x])
			totalEnergy += val * val

			// 水平差值
			if x+1 < w {
				diff := val - float64(pixels[y][x+1])
				highFreqEnergy += diff * diff
			}
			// 垂直差值
			if y+1 < h {
				diff := val - float64(pixels[y+1][x])
				highFreqEnergy += diff * diff
			}
		}
	}

	if totalEnergy == 0 {
		return 0
	}

	ratio := highFreqEnergy / totalEnergy
	// 正規化到 0–1（自然圖片約 0.1，對抗性噪音可達 0.5+）
	score := math.Min(ratio/0.5, 1.0)
	return score
}

// ──────────────────────────────────────────────
// 檢測 2：直方圖異常
// ──────────────────────────────────────────────

// detectHistogramAnomaly 偵測像素值分佈的不自然峰值或平坦區。
// 自然圖片通常有平滑的鐘形分佈。
// 被注入隱藏觸發模式的圖片可能出現異常尖峰。
func detectHistogramAnomaly(pixels [][]uint8, w, h int) float64 {
	// 計算 256-bin 直方圖
	var hist [256]int
	total := w * h
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			hist[pixels[y][x]]++
		}
	}

	// 計算直方圖的平均值與標準差
	mean := float64(total) / 256.0
	var variance float64
	for i := 0; i < 256; i++ {
		diff := float64(hist[i]) - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / 256.0)

	// 計算超過 3σ 的 bin 數量（異常峰值指標）
	threshold := mean + 3*stddev
	anomalyCount := 0
	for i := 0; i < 256; i++ {
		if float64(hist[i]) > threshold {
			anomalyCount++
		}
	}

	// 同時檢測過多零值 bin（不自然平坦區）
	zeroBins := 0
	for i := 0; i < 256; i++ {
		if hist[i] == 0 {
			zeroBins++
		}
	}

	// 正規化：異常峰值 + 零值比例
	peakScore := math.Min(float64(anomalyCount)/10.0, 1.0)
	flatScore := math.Min(float64(zeroBins)/128.0, 1.0)

	return math.Max(peakScore, flatScore)
}

// ──────────────────────────────────────────────
// 檢測 3：LSB 分佈異常（Monobit test）
// ──────────────────────────────────────────────

// detectLSBAnomaly 檢測最低有效位元是否隨機分佈。
// 自然圖片的 LSB 應接近 50/50 分佈。
// 隱寫術 payload 會使 LSB 呈現非隨機模式。
func detectLSBAnomaly(pixels [][]uint8, w, h int) float64 {
	total := w * h
	if total == 0 {
		return 0
	}

	// Monobit test: 計算 LSB 為 1 的比例
	onesCount := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if pixels[y][x]&1 == 1 {
				onesCount++
			}
		}
	}

	ratio := float64(onesCount) / float64(total)
	// 偏離 0.5 的程度
	deviation := math.Abs(ratio - 0.5)

	// 也檢查相鄰 LSB 的相關性（隱寫術常產生局部 pattern）
	correlatedPairs := 0
	totalPairs := 0
	for y := 0; y < h; y++ {
		for x := 0; x+1 < w; x++ {
			if (pixels[y][x] & 1) == (pixels[y][x+1] & 1) {
				correlatedPairs++
			}
			totalPairs++
		}
	}

	correlation := 0.0
	if totalPairs > 0 {
		correlation = math.Abs(float64(correlatedPairs)/float64(totalPairs) - 0.5)
	}

	// 結合偏離度與相關性
	score := math.Min((deviation+correlation)*4.0, 1.0)
	return score
}

// ──────────────────────────────────────────────
// 音訊污染偵測（基本版）
// ──────────────────────────────────────────────

func detectAudioPollution(filePath string) (*PollutionReport, error) {
	samples, _, err := readWAVSamples(filePath)
	if err != nil {
		return nil, fmt.Errorf("read wav: %w", err)
	}

	if len(samples) < 100 {
		return &PollutionReport{Details: "audio too short for analysis"}, nil
	}

	// 檢測 1：高頻能量（相鄰樣本差值）
	highFreq := detectAudioHighFreq(samples)

	// 檢測 2：振幅直方圖異常
	histogram := detectAudioHistogramAnomaly(samples)

	// 檢測 3：LSB 分佈（將樣本量化到 16-bit 後檢測）
	lsb := detectAudioLSB(samples)

	weighted := 0.4*highFreq + 0.3*histogram + 0.3*lsb

	details := fmt.Sprintf("高頻能量=%.3f 直方圖=%.3f LSB=%.3f", highFreq, histogram, lsb)
	if weighted > PollutionThreshold {
		details += " → 判定為模型污染風險"
	}

	return &PollutionReport{
		HighFreqScore:   highFreq,
		HistogramScore:  histogram,
		LSBScore:        lsb,
		WeightedTotal:   weighted,
		IsPollutionRisk: weighted > PollutionThreshold,
		Details:         details,
	}, nil
}

func detectAudioHighFreq(samples []int16) float64 {
	var totalE, diffE float64
	for i, s := range samples {
		totalE += sampleEnergy(s)
		if i+1 < len(samples) {
			d := float64(int32(s)-int32(samples[i+1])) / 32768.0
			diffE += d * d
		}
	}
	if totalE == 0 {
		return 0
	}
	return math.Min((diffE/totalE)/0.5, 1.0)
}

func detectAudioHistogramAnomaly(samples []int16) float64 {
	// 量化到 100 個 bin
	var hist [100]int
	for _, s := range samples {
		bin := int((int(s) + 32768) * 100 / 65536)
		if bin < 0 {
			bin = 0
		}
		if bin > 99 {
			bin = 99
		}
		hist[bin]++
	}

	mean := float64(len(samples)) / 100.0
	var variance float64
	for i := 0; i < 100; i++ {
		d := float64(hist[i]) - mean
		variance += d * d
	}
	stddev := math.Sqrt(variance / 100.0)

	anomaly := 0
	threshold := mean + 3*stddev
	for i := 0; i < 100; i++ {
		if float64(hist[i]) > threshold {
			anomaly++
		}
	}
	return math.Min(float64(anomaly)/5.0, 1.0)
}

func detectAudioLSB(samples []int16) float64 {
	ones := 0
	for _, s := range samples {
		if s&1 == 1 {
			ones++
		}
	}
	ratio := float64(ones) / float64(len(samples))
	deviation := math.Abs(ratio - 0.5)
	return math.Min(deviation*4.0, 1.0)
}
