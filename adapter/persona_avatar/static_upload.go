// persona_avatar/static_upload.go — 靜態圖片上傳（§10.6–§10.8）。
// 後端驗證 mime、大小、尺寸，裁切為 256x256，原圖保留。
package persona_avatar

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	// 註冊 webp 解碼器（若 Go 標準庫支援）
	_ "image/gif" // 用於偵測並拒絕
)

// ──────────────────────────────────────────────
// 允許的 MIME 類型
// ──────────────────────────────────────────────

var allowedMimeTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

var rejectedMimeTypes = map[string]bool{
	"image/svg+xml": true, // §10.7: 拒絕 SVG
	"image/gif":     true, // §10.7: 拒絕 GIF
}

// 最大檔案大小 5MB
const maxFileSize = 5 * 1024 * 1024

// ──────────────────────────────────────────────
// 上傳靜態圖片
// ──────────────────────────────────────────────

// SaveStaticAvatar 儲存靜態頭像圖片。
// 驗證：mime type、檔案大小、解碼尺寸、persona_id、路徑安全。
// 輸出：裁切後的 256x256 PNG 存至 avatar_static.png，原圖存至 avatar_static_original.png。
func (s *Service) SaveStaticAvatar(personaID, mimeType string, imageData []byte, crop *CropRect, outputSize int) error {
	if personaID == lockedAvatarPersonaID {
		return fmt.Errorf("persona avatar is locked: %s", personaID)
	}

	// 驗證 persona_id
	if personaID == "" || strings.Contains(personaID, "..") || strings.Contains(personaID, "/") {
		return fmt.Errorf("無效的 persona ID")
	}

	// 驗證 MIME 類型
	if rejectedMimeTypes[mimeType] {
		return fmt.Errorf("不允許的圖片格式: %s（僅接受 PNG、JPEG、WebP）", mimeType)
	}
	if !allowedMimeTypes[mimeType] {
		return fmt.Errorf("不支援的圖片格式: %s", mimeType)
	}

	// 驗證檔案大小
	if len(imageData) > maxFileSize {
		return fmt.Errorf("圖片大小超過上限（最大 5MB，實際 %d bytes）", len(imageData))
	}
	if len(imageData) == 0 {
		return fmt.Errorf("圖片資料為空")
	}

	// 解碼圖片
	reader := bytes.NewReader(imageData)
	img, format, err := image.Decode(reader)
	if err != nil {
		return fmt.Errorf("無法解碼圖片: %w", err)
	}

	// 拒絕 GIF 格式（即使 mime 宣稱其他格式）
	if format == "gif" {
		return fmt.Errorf("不允許 GIF 格式")
	}

	// 驗證解碼後尺寸
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < 64 || height < 64 {
		return fmt.Errorf("圖片太小（最小 64x64，實際 %dx%d）", width, height)
	}
	if width > 4096 || height > 4096 {
		return fmt.Errorf("圖片太大（最大 4096x4096，實際 %dx%d）", width, height)
	}

	// 設定輸出大小（預設 256）
	if outputSize <= 0 {
		outputSize = 256
	}

	// 建立目標目錄
	dir := s.avatarDir(personaID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("建立目錄失敗: %w", err)
	}

	// 儲存原圖
	originalPath := filepath.Join(dir, "avatar_static_original.png")
	if err := savePNG(originalPath, img); err != nil {
		return fmt.Errorf("儲存原圖失敗: %w", err)
	}

	// 裁切（若有指定）
	cropped := img
	if crop != nil && crop.Width > 0 && crop.Height > 0 {
		cropped = cropImage(img, crop)
	}

	// 縮放到目標大小
	resized := resizeImage(cropped, outputSize, outputSize)

	// 儲存裁切後的圖片
	staticPath := filepath.Join(dir, "avatar_static.png")
	if err := savePNG(staticPath, resized); err != nil {
		return fmt.Errorf("儲存裁切圖片失敗: %w", err)
	}

	// 更新 avatar_config.json
	relStaticPath := filepath.Join("data", "personas", personaID, "avatar", "avatar_static.png")
	relOrigPath := filepath.Join("data", "personas", personaID, "avatar", "avatar_static_original.png")

	config := s.GetCurrentAvatar(personaID)
	config.AvatarProvider = ProviderStaticImage
	config.PersonaID = personaID
	config.StaticAvatarPath = relStaticPath
	config.OriginalImagePath = relOrigPath
	config.Crop = crop
	config.OutputSize = outputSize
	config.UpdatedAt = time.Now().Format(time.RFC3339)

	return s.saveConfig(personaID, config)
}

// ──────────────────────────────────────────────
// 圖片處理輔助函式
// ──────────────────────────────────────────────

// savePNG 將圖片存為 PNG 檔案。
func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// cropImage 依照指定區域裁切圖片。
func cropImage(img image.Image, crop *CropRect) image.Image {
	// 建立裁切後的 RGBA 圖片
	rect := image.Rect(0, 0, crop.Width, crop.Height)
	cropped := image.NewRGBA(rect)

	bounds := img.Bounds()
	for y := 0; y < crop.Height; y++ {
		for x := 0; x < crop.Width; x++ {
			srcX := crop.X + x
			srcY := crop.Y + y
			if srcX >= bounds.Min.X && srcX < bounds.Max.X &&
				srcY >= bounds.Min.Y && srcY < bounds.Max.Y {
				cropped.Set(x, y, img.At(srcX, srcY))
			}
		}
	}
	return cropped
}

// resizeImage 將圖片縮放到目標大小（最近鄰插值，適合 pixel art 風格）。
func resizeImage(img image.Image, targetW, targetH int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	resized := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			srcX := bounds.Min.X + x*srcW/targetW
			srcY := bounds.Min.Y + y*srcH/targetH
			resized.Set(x, y, img.At(srcX, srcY))
		}
	}
	return resized
}

// 確保 jpeg 套件被引用（用於解碼 JPEG 上傳）
var _ = jpeg.Decode
