// size_guard.go — 大檔案軟防護。
// 不拒絕任何大小，但依檔案大小分級處理，避免 UI 凍結或 OOM。
// ≤512KB: 直接讀入記憶體  >512KB: 串流讀取  >5MB: 建議呼叫端用 goroutine
package builtin

import (
	"fmt"
	"io"
	"os"
)

const (
	// ThresholdInline 以下的檔案可直接載入記憶體（512KB）。
	ThresholdInline int64 = 512 * 1024
	// ThresholdAsync 以上的檔案建議用背景 goroutine 處理（5MB）。
	ThresholdAsync int64 = 5 * 1024 * 1024
)

// SizeClass 描述檔案大小等級，呼叫端據此選擇處理策略。
type SizeClass int

const (
	SizeInline SizeClass = iota // ≤512KB：安全直接載入
	SizeStream                  // 512KB–5MB：串流處理
	SizeAsync                   // >5MB：建議背景 goroutine
)

// ClassifySize 判定檔案大小等級。
func ClassifySize(size int64) SizeClass {
	if size <= ThresholdInline {
		return SizeInline
	}
	if size <= ThresholdAsync {
		return SizeStream
	}
	return SizeAsync
}

// ReadWithGuard 依檔案大小安全讀取。
// 所有等級都能讀取成功，差異在於大檔案會用 LimitReader 防止失控。
// maxBytes=0 表示不限制（但仍會分級）。
func ReadWithGuard(path string, maxBytes int64) ([]byte, SizeClass, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, SizeInline, fmt.Errorf("size_guard: stat %s: %w", path, err)
	}
	class := ClassifySize(info.Size())

	f, err := os.Open(path)
	if err != nil {
		return nil, class, fmt.Errorf("size_guard: open %s: %w", path, err)
	}
	defer f.Close()

	var reader io.Reader = f
	if maxBytes > 0 {
		reader = io.LimitReader(f, maxBytes)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, class, fmt.Errorf("size_guard: read %s: %w", path, err)
	}
	return data, class, nil
}
