// json_store.go 提供共用的 JSON 檔案持久化輔助結構 JSONStore。
//
// 取代 settings、preference、browser_pref、external_link、adapter_registry
// 五個套件中重複的 load/save 樣板程式碼。
//
// 設計原則：
//   - 泛型設計：T 為儲存的資料型別，由呼叫端決定
//   - 並發安全：所有公開方法持有 mu 鎖
//   - 檔案權限：目錄 0o700、檔案 0o600（只有擁有者可讀寫）
//   - 不負責業務邏輯：只處理序列化/反序列化 + 磁碟 I/O
//   - 不持有業務鎖：呼叫端若有自己的 sync.Mutex，可直接使用 LoadRaw/SaveRaw
//     繞過內建鎖，但必須確保自己的鎖已持有
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// JSONStore 是泛型 JSON 檔案持久化輔助結構。
// T 為儲存的資料型別（例如 []Adapter、BrowserPreference 等）。
//
// 典型用法：
//
//	store := storage.NewJSONStore[[]Adapter](path)
//	data, err := store.Load()
//	err = store.Save(data)
type JSONStore[T any] struct {
	mu   sync.Mutex
	path string
}

// NewJSONStore 建立一個 JSONStore，指定 JSON 檔案的完整路徑。
// 檔案目錄會在第一次 Save 時惰性建立。
func NewJSONStore[T any](path string) *JSONStore[T] {
	return &JSONStore[T]{path: path}
}

// Path 回傳此 store 管理的檔案路徑。
func (s *JSONStore[T]) Path() string {
	return s.path
}

// Load 從磁碟讀取並反序列化 JSON 檔案。
// 若檔案不存在，回傳 T 的零值與 nil error（首次使用情境）。
// 並發安全：持有 mu 鎖。
func (s *JSONStore[T]) Load() (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadRaw()
}

// Save 將資料序列化為 JSON 並寫入磁碟。
// 使用 MarshalIndent 產生易讀格式（2 空格縮排）。
// 並發安全：持有 mu 鎖。
func (s *JSONStore[T]) Save(data T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveRaw(data)
}

// LoadRaw 不持鎖版本的 Load。
// 呼叫端必須自行確保並發安全（例如持有自己的 sync.Mutex）。
// 適用於呼叫端在同一個鎖區間內需要先 load 再做業務邏輯再 save 的情境。
func (s *JSONStore[T]) LoadRaw() (T, error) {
	return s.loadRaw()
}

// SaveRaw 不持鎖版本的 Save。
// 呼叫端必須自行確保並發安全。
func (s *JSONStore[T]) SaveRaw(data T) error {
	return s.saveRaw(data)
}

// --- internal ---

func (s *JSONStore[T]) loadRaw() (T, error) {
	var zero T
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return zero, nil // 首次使用，回傳零值
	}
	if err != nil {
		return zero, fmt.Errorf("json_store: read %s: %w", filepath.Base(s.path), err)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return zero, fmt.Errorf("json_store: unmarshal %s: %w", filepath.Base(s.path), err)
	}
	return result, nil
}

func (s *JSONStore[T]) saveRaw(data T) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("json_store: mkdir %s: %w", filepath.Base(s.path), err)
	}
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("json_store: marshal %s: %w", filepath.Base(s.path), err)
	}
	return os.WriteFile(s.path, payload, 0o600)
}
