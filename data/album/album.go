// Package album 管理「紀念照」相冊：每張產生的圖以圖檔落地，
// 並在 album_index.jsonl 留一筆含「產生背景」的中繼資料，供回放與展示。
//
// 設計原則（對齊 repo 慣例）：
//   - 不新增外部依賴：stdlib + 既有 data/storage 的 AtomicWriteFile。
//   - jsonl append 落盤（對齊 memory_ops.jsonl / casebook.jsonl 慣例）。
//   - 圖檔與索引都放在專案層 album/ 下；檔案 0600、目錄 0700（本機隱私資料）。
//
// 「記住照片的產生背景」= Photo 內保留 Scene（場景描述）、ContextDigest
// （產生當下的對話摘要）、MemoryTag（記憶錨點，供 展開ㄌtagㄌ待命 撈回原文）、
// 以及實際送進產圖器的 Prompt / Seed / Model，讓每張紀念照都能被追溯。
package album

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ui_console/data/storage"
)

const (
	albumDirName    = "album"
	imagesDirName   = "images"
	indexFileName   = "album_index.jsonl"
	maxCaptionRunes = 280
)

// Photo 是一張紀念照的完整中繼資料。
type Photo struct {
	ID            string `json:"id"`             // 唯一 ID（時間序 + 短亂數）
	CreatedAt     string `json:"created_at"`     // RFC3339
	PersonaID     string `json:"persona_id"`     // 產生當下啟用的人格
	PersonaName   string `json:"persona_name"`   // 顯示用名稱
	Scene         string `json:"scene"`          // 場景描述（LLM 提議的 target）
	ContextDigest string `json:"context_digest"` // 產生背景：對話摘要快照
	MemoryTag     string `json:"memory_tag"`     // 記憶錨點標籤（例 S-12345），可空
	Prompt        string `json:"prompt"`         // 實際正向提示詞
	Negative      string `json:"negative"`       // 負向提示詞
	Seed          int64  `json:"seed"`           // 種子（可重現）
	Model         string `json:"model"`          // checkpoint 名稱
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	ImageRel      string `json:"image_rel"` // 相對 album/ 的圖檔路徑，例 images/<id>.png
	Caption       string `json:"caption"`   // 使用者自訂說明，可空
}

// Store 以專案層 album/ 目錄為根，持有索引與圖檔。
type Store struct {
	root string // album/ 絕對路徑
}

// NewStore 由「專案根」建立相冊 Store；album/ 與 album/images/ 在首次寫入時惰性建立。
func NewStore(projectRoot string) *Store {
	return &Store{root: filepath.Join(projectRoot, albumDirName)}
}

// NewStoreForProject 是常見呼叫法：用 storage.ProjectRoot 推出相冊根。
func NewStoreForProject(baseDir, projectID string) *Store {
	return NewStore(storage.ProjectRoot(baseDir, projectID))
}

func (s *Store) imagesDir() string { return filepath.Join(s.root, imagesDirName) }
func (s *Store) indexPath() string { return filepath.Join(s.root, indexFileName) }

// ImagePath 回傳某張紀念照圖檔的絕對路徑。
func (s *Store) ImagePath(p Photo) string {
	return filepath.Join(s.root, filepath.FromSlash(p.ImageRel))
}

func (s *Store) ensureDirs() error {
	if err := os.MkdirAll(s.imagesDir(), 0o700); err != nil {
		return fmt.Errorf("建立相冊目錄失敗: %w", err)
	}
	return nil
}

// NewPhotoID 產生時間序可排序的相冊 ID。
func NewPhotoID(now time.Time) string {
	return fmt.Sprintf("A-%s", now.UTC().Format("20060102T150405.000"))
}

// Add 落地一張紀念照：先寫圖檔（AtomicWrite），再 append 索引（fail-closed）。
// 呼叫端只需填好 Photo 的描述欄位與 PNG bytes；ID/CreatedAt/ImageRel 若空會自動補。
func (s *Store) Add(p Photo, png []byte) (Photo, error) {
	if len(png) == 0 {
		return Photo{}, fmt.Errorf("空白圖檔，拒絕落地")
	}
	if err := s.ensureDirs(); err != nil {
		return Photo{}, err
	}
	now := time.Now()
	if strings.TrimSpace(p.ID) == "" {
		p.ID = NewPhotoID(now)
	}
	if strings.TrimSpace(p.CreatedAt) == "" {
		p.CreatedAt = now.UTC().Format(time.RFC3339)
	}
	if strings.TrimSpace(p.ImageRel) == "" {
		p.ImageRel = filepath.ToSlash(filepath.Join(imagesDirName, p.ID+".png"))
	}
	p.Caption = clampRunes(p.Caption, maxCaptionRunes)

	// Step 1：寫圖檔（0600、atomic）。
	if err := storage.AtomicWriteFile(s.ImagePath(p), png, 0o600); err != nil {
		return Photo{}, fmt.Errorf("寫入紀念照圖檔失敗: %w", err)
	}

	// Step 2：append 索引；失敗則回收圖檔，維持索引與圖檔一致。
	if err := s.appendIndex(p); err != nil {
		_ = os.Remove(s.ImagePath(p))
		return Photo{}, fmt.Errorf("寫入相冊索引失敗: %w", err)
	}
	return p, nil
}

func (s *Store) appendIndex(p Photo) error {
	line, err := json.Marshal(p)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.indexPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// List 回傳所有紀念照，最新在前。索引不存在時回空切片。
func (s *Store) List() ([]Photo, error) {
	photos, err := s.readAll()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(photos, func(i, j int) bool {
		return photos[i].CreatedAt > photos[j].CreatedAt
	})
	return photos, nil
}

// Get 取單張。找不到回 (zero, false, nil)。
func (s *Store) Get(id string) (Photo, bool, error) {
	photos, err := s.readAll()
	if err != nil {
		return Photo{}, false, err
	}
	for _, p := range photos {
		if p.ID == id {
			return p, true, nil
		}
	}
	return Photo{}, false, nil
}

// SetCaption 更新某張的使用者說明，整檔重寫索引。
func (s *Store) SetCaption(id, caption string) (Photo, error) {
	photos, err := s.readAll()
	if err != nil {
		return Photo{}, err
	}
	var updated Photo
	found := false
	for i := range photos {
		if photos[i].ID == id {
			photos[i].Caption = clampRunes(caption, maxCaptionRunes)
			updated = photos[i]
			found = true
			break
		}
	}
	if !found {
		return Photo{}, fmt.Errorf("找不到紀念照 %s", id)
	}
	if err := s.rewriteIndex(photos); err != nil {
		return Photo{}, err
	}
	return updated, nil
}

// Delete 移除一張紀念照（圖檔 + 索引）。找不到視為已刪除（idempotent）。
func (s *Store) Delete(id string) error {
	photos, err := s.readAll()
	if err != nil {
		return err
	}
	kept := make([]Photo, 0, len(photos))
	var removed *Photo
	for i := range photos {
		if photos[i].ID == id {
			cp := photos[i]
			removed = &cp
			continue
		}
		kept = append(kept, photos[i])
	}
	if removed == nil {
		return nil
	}
	if err := s.rewriteIndex(kept); err != nil {
		return err
	}
	_ = os.Remove(s.ImagePath(*removed))
	return nil
}

// Count 回傳相冊張數（供「聊一段時間後是否再提議」之類的節流判斷）。
func (s *Store) Count() (int, error) {
	photos, err := s.readAll()
	if err != nil {
		return 0, err
	}
	return len(photos), nil
}

func (s *Store) readAll() ([]Photo, error) {
	f, err := os.Open(s.indexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return []Photo{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var photos []Photo
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var p Photo
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			// 容錯：壞行略過，不擋整本相冊。
			continue
		}
		photos = append(photos, p)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return photos, nil
}

func (s *Store) rewriteIndex(photos []Photo) error {
	var b strings.Builder
	for _, p := range photos {
		line, err := json.Marshal(p)
		if err != nil {
			return err
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return storage.AtomicWriteFile(s.indexPath(), []byte(b.String()), 0o600)
}

func clampRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
