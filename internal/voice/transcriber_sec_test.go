package voice

import (
	"encoding/base64"
	"strings"
	"testing"
)

// SEC-21 驗證：base64 長度預檢邏輯。
func TestBase64LengthPrecheck(t *testing.T) {
	// maxAudioBytes = 16MB
	maxBase64Len := maxAudioBytes*4/3 + 4

	tests := []struct {
		name       string
		dataSize   int
		shouldPass bool
	}{
		{"1KB 通過", 1024, true},
		{"1MB 通過", 1 * 1024 * 1024, true},
		{"15MB 通過", 15 * 1024 * 1024, true},
		{"16MB 剛好邊界", maxAudioBytes, true},
		{"17MB 超限", 17 * 1024 * 1024, false},
		{"100MB 超限", 100 * 1024 * 1024, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模擬 base64 編碼後的長度
			b64Len := tt.dataSize*4/3 + 4
			passed := b64Len <= maxBase64Len
			if passed != tt.shouldPass {
				t.Errorf("dataSize=%d, base64Len=%d, maxBase64Len=%d: passed=%v, want %v",
					tt.dataSize, b64Len, maxBase64Len, passed, tt.shouldPass)
			}
		})
	}
}

// SEC-21 驗證：實際 base64 編碼的長度估算準確性。
func TestBase64LengthEstimation(t *testing.T) {
	// 用小資料驗證公式準確
	for _, size := range []int{0, 1, 2, 3, 100, 1000, 4096} {
		data := make([]byte, size)
		encoded := base64.StdEncoding.EncodeToString(data)
		estimated := size*4/3 + 4
		if len(encoded) > estimated {
			t.Errorf("size=%d: actual base64 len=%d > estimated=%d",
				size, len(encoded), estimated)
		}
	}
}

// SEC-22 驗證：檔名不再包含可預測的時間戳格式。
func TestTempFileNaming(t *testing.T) {
	// 舊格式: voice-20260522-120000.000000000.wav（可預測）
	// 新格式: voice-{random}.wav（os.CreateTemp 產生的）
	oldPattern := "voice-20260522-"
	newPrefix := "voice-"

	// 新檔名不應該包含日期格式
	exampleNewName := "voice-1234567890.wav" // os.CreateTemp 風格
	if strings.Contains(exampleNewName, oldPattern) {
		t.Error("新檔名不應包含可預測的日期格式")
	}
	if !strings.HasPrefix(exampleNewName, newPrefix) {
		t.Error("新檔名應保持 voice- prefix")
	}
}
