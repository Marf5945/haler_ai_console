// w3a_media/fingerprint.go — §9A.11 雙層指紋計算（Byte Hash + Perceptual Hash）。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 媒體驗證的核心：兩層指紋各司其職                            │
// │                                                             │
// │  Byte Hash（SHA-256）：                                     │
// │    - 全檔案雜湊，任何 1 byte 差異即不同                     │
// │    - 匹配 → 可證明 exact_original                           │
// │                                                             │
// │  Perceptual Hash（Go-native pHash / spectral）：            │
// │    - 容忍輕微壓縮 / 轉碼的視覺/聽覺相似度                  │
// │    - 匹配 → 僅證明來源相似，不恢復延伸權（§9A.11 硬規則）  │
// │                                                             │
// │ 圖片 pHash 算法：                                           │
// │  1. 縮小到 32×32 灰階                                      │
// │  2. 取 8×8 DCT 低頻區                                      │
// │  3. 中位數比較 → 64-bit 二進位指紋                          │
// │                                                             │
// │ 音訊 spectral hash 算法：                                   │
// │  1. 讀取 WAV PCM samples                                    │
// │  2. 分窗 → 計算各窗能量分佈                                 │
// │  3. 能量比較 → 二進位指紋                                   │
// │                                                             │
// │ 零外部依賴：僅使用 Go 標準庫（crypto/sha256, image, math）  │
// └─────────────────────────────────────────────────────────────┘
package w3a_media

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
)

// ──────────────────────────────────────────────
// Byte Hash（SHA-256 全檔雜湊）
// ──────────────────────────────────────────────

// ComputeByteHash 計算檔案的 SHA-256 雜湊。
func ComputeByteHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// ComputeByteHashFromBytes 從 byte slice 計算 SHA-256。
func ComputeByteHashFromBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", h[:])
}

// ──────────────────────────────────────────────
// Perceptual Hash — 圖片（pHash 算法）
// ──────────────────────────────────────────────

// ComputeImagePerceptualHash 計算圖片的感知雜湊。
// 算法：縮小 → 灰階 → DCT → 中位數比較 → 64-bit 指紋。
func ComputeImagePerceptualHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	// Step 1: 縮小到 32×32 並轉灰階
	gray32 := resizeToGray(img, 32, 32)

	// Step 2: 計算 32×32 DCT，取 8×8 低頻區
	dct := computeDCT2D(gray32, 32)
	lowFreq := extractLowFreq(dct, 8)

	// Step 3: 中位數比較 → 64-bit 指紋
	hash := dctToHash(lowFreq)
	return fmt.Sprintf("phash:%016x", hash), nil
}

// resizeToGray 將圖片縮小到指定尺寸並轉為灰階值陣列。
// 使用最近鄰插值（Go-native，無外部依賴）。
func resizeToGray(img image.Image, w, h int) [][]float64 {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	result := make([][]float64, h)
	for y := 0; y < h; y++ {
		result[y] = make([]float64, w)
		srcY := bounds.Min.Y + y*srcH/h
		for x := 0; x < w; x++ {
			srcX := bounds.Min.X + x*srcW/w
			r, g, b, _ := img.At(srcX, srcY).RGBA()
			// ITU-R BT.601 灰階轉換
			gray := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
			result[y][x] = gray / 65535.0
		}
	}
	return result
}

// computeDCT2D 計算 2D 離散餘弦變換。
func computeDCT2D(pixels [][]float64, size int) [][]float64 {
	// 先對每一行做 1D DCT
	temp := make([][]float64, size)
	for i := 0; i < size; i++ {
		temp[i] = computeDCT1D(pixels[i])
	}

	// 再對每一列做 1D DCT
	result := make([][]float64, size)
	for i := 0; i < size; i++ {
		result[i] = make([]float64, size)
	}

	for x := 0; x < size; x++ {
		col := make([]float64, size)
		for y := 0; y < size; y++ {
			col[y] = temp[y][x]
		}
		dctCol := computeDCT1D(col)
		for y := 0; y < size; y++ {
			result[y][x] = dctCol[y]
		}
	}
	return result
}

// computeDCT1D 計算 1D DCT-II。
func computeDCT1D(input []float64) []float64 {
	n := len(input)
	output := make([]float64, n)
	for k := 0; k < n; k++ {
		sum := 0.0
		for i := 0; i < n; i++ {
			sum += input[i] * math.Cos(math.Pi*float64(2*i+1)*float64(k)/(2.0*float64(n)))
		}
		output[k] = sum
	}
	return output
}

// extractLowFreq 從 DCT 矩陣提取左上角低頻區域。
func extractLowFreq(dct [][]float64, size int) []float64 {
	result := make([]float64, 0, size*size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			result = append(result, dct[y][x])
		}
	}
	return result
}

// dctToHash 將低頻 DCT 係數轉為 64-bit 雜湊。
func dctToHash(values []float64) uint64 {
	// 排除 DC 分量（index 0），計算中位數
	sorted := make([]float64, len(values)-1)
	copy(sorted, values[1:])
	median := computeMedian(sorted)

	var hash uint64
	for i := 0; i < 64 && i < len(values)-1; i++ {
		if values[i+1] > median {
			hash |= 1 << uint(63-i)
		}
	}
	return hash
}

// computeMedian 計算中位數（不修改原陣列）。
func computeMedian(data []float64) float64 {
	n := len(data)
	if n == 0 {
		return 0
	}
	// 簡易選擇排序找中位數（64 元素足夠）
	tmp := make([]float64, n)
	copy(tmp, data)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if tmp[j] < tmp[i] {
				tmp[i], tmp[j] = tmp[j], tmp[i]
			}
		}
	}
	if n%2 == 0 {
		return (tmp[n/2-1] + tmp[n/2]) / 2
	}
	return tmp[n/2]
}

// ──────────────────────────────────────────────
// Perceptual Hash — 音訊（頻譜能量指紋）
// ──────────────────────────────────────────────

// ComputeAudioPerceptualHash 計算音訊的感知雜湊。
// 算法：讀取 WAV PCM → 分窗 → 計算能量分佈 → 二進位指紋。
func ComputeAudioPerceptualHash(filePath string) (string, error) {
	samples, _, err := readWAVSamples(filePath)
	if err != nil {
		return "", fmt.Errorf("read wav: %w", err)
	}

	if len(samples) == 0 {
		return "ahash:0000000000000000", nil
	}

	// 分成 64 個窗口，每窗計算能量
	windowSize := len(samples) / 64
	if windowSize < 1 {
		windowSize = 1
	}

	energies := make([]float64, 64)
	for i := 0; i < 64; i++ {
		start := i * windowSize
		end := start + windowSize
		if end > len(samples) {
			end = len(samples)
		}
		sum := 0.0
		for j := start; j < end; j++ {
			sum += samples[j] * samples[j]
		}
		energies[i] = sum / float64(end-start)
	}

	// 中位數比較 → 64-bit 指紋
	median := computeMedian(energies)
	var hash uint64
	for i := 0; i < 64; i++ {
		if energies[i] > median {
			hash |= 1 << uint(63-i)
		}
	}
	return fmt.Sprintf("ahash:%016x", hash), nil
}

// readWAVSamples 讀取 WAV 檔案的 PCM 樣本（Go-native，支援 16-bit PCM）。
func readWAVSamples(filePath string) ([]float64, int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	// 讀取 RIFF header
	header := make([]byte, 44)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, 0, fmt.Errorf("read wav header: %w", err)
	}

	// 驗證 RIFF/WAVE 標識
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a valid WAV file")
	}

	// 解析格式：channels, sample rate, bits per sample
	channels := int(binary.LittleEndian.Uint16(header[22:24]))
	sampleRate := int(binary.LittleEndian.Uint32(header[24:28]))
	bitsPerSample := int(binary.LittleEndian.Uint16(header[34:36]))

	if bitsPerSample != 16 {
		return nil, sampleRate, fmt.Errorf("only 16-bit PCM supported, got %d-bit", bitsPerSample)
	}

	// 讀取 data chunk
	dataSize := int(binary.LittleEndian.Uint32(header[40:44]))
	data := make([]byte, dataSize)
	n, _ := io.ReadFull(f, data)
	data = data[:n]

	// 轉換為 float64 樣本（取第一聲道）
	bytesPerSample := bitsPerSample / 8
	frameSize := channels * bytesPerSample
	numFrames := len(data) / frameSize

	samples := make([]float64, numFrames)
	for i := 0; i < numFrames; i++ {
		offset := i * frameSize
		if offset+1 >= len(data) {
			break
		}
		raw := int16(binary.LittleEndian.Uint16(data[offset : offset+2]))
		samples[i] = float64(raw) / 32768.0
	}

	return samples, sampleRate, nil
}

// ──────────────────────────────────────────────
// 感知指紋比較
// ──────────────────────────────────────────────

// ComparePerceptual 比較兩個感知雜湊的相似度（0.0–1.0，1.0 = 完全相同）。
// 使用 Hamming distance 正規化。
func ComparePerceptual(a, b string) float64 {
	// 提取 hex 部分（跳過 "phash:" 或 "ahash:" 前綴）
	hexA := extractHex(a)
	hexB := extractHex(b)

	if len(hexA) != len(hexB) || len(hexA) == 0 {
		return 0.0
	}

	// 解析為 uint64
	var valA, valB uint64
	fmt.Sscanf(hexA, "%x", &valA)
	fmt.Sscanf(hexB, "%x", &valB)

	// Hamming distance
	xor := valA ^ valB
	distance := 0
	for xor != 0 {
		distance++
		xor &= xor - 1
	}

	return 1.0 - float64(distance)/64.0
}

// extractHex 從 "phash:xxxx" 或 "ahash:xxxx" 格式提取 hex 部分。
func extractHex(hash string) string {
	for i := 0; i < len(hash); i++ {
		if hash[i] == ':' {
			return hash[i+1:]
		}
	}
	return hash
}

// ──────────────────────────────────────────────
// 通用入口
// ──────────────────────────────────────────────

// ComputePerceptualHash 依媒體類型自動選擇 pHash 或 spectral hash。
func ComputePerceptualHash(filePath string, scope MediaScope) (string, error) {
	switch scope {
	case ScopeImage:
		return ComputeImagePerceptualHash(filePath)
	case ScopeAudio:
		return ComputeAudioPerceptualHash(filePath)
	default:
		return "", fmt.Errorf("unsupported media scope: %s", scope)
	}
}
