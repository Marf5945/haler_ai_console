// persona_avatar/pixel_renderer.go — Built-in Pixel Avatar 渲染器（§10.9）。
// 用 Go image 標準庫以程序化方式繪製 pixel art PNG。
// 9 種狀態 × 2 種尺寸（256 / 128），自製素材避免授權風險。
// v3.0 — 狼犬獸人版 64x64：銀白+黑藍毛色、尖耳帶耳環、壞笑。
package persona_avatar

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// ──────────────────────────────────────────────
// 像素顏色定義 — 狼犬獸人配色（64x64 版）
// ──────────────────────────────────────────────

var (
	wlfBG       = color.RGBA{0, 0, 0, 0}
	wlfOutline  = color.RGBA{18, 20, 32, 255}
	wlfFur      = color.RGBA{184, 196, 210, 255}
	wlfFurD     = color.RGBA{128, 145, 170, 255}
	wlfFurH     = color.RGBA{220, 228, 238, 255}
	wlfEye      = color.RGBA{246, 181, 38, 255}
	wlfEyeD     = color.RGBA{188, 92, 24, 255}
	wlfMouth    = color.RGBA{50, 36, 46, 255}
	wlfEarIn    = color.RGBA{25, 35, 64, 255}
	wlfEarH     = color.RGBA{62, 82, 126, 255}
	wlfEarring  = color.RGBA{170, 185, 210, 255}
	wlfEarringH = color.RGBA{200, 215, 235, 255}
	wlfMuzzle   = color.RGBA{230, 235, 242, 255}
	wlfMuzzleD  = color.RGBA{210, 215, 225, 255}
	wlfNose     = color.RGBA{24, 22, 36, 255}
	wlfFang     = color.RGBA{240, 238, 230, 255}
	wlfBrow     = color.RGBA{20, 22, 34, 255}

	wlfStateColors = map[AvatarStateTrigger]color.RGBA{
		StateIdle:       {200, 210, 220, 255},
		StateThinking:   {100, 150, 255, 255},
		StateWorking:    {50, 200, 100, 255},
		StateHappy:      {255, 200, 50, 255},
		StateWarning:    {255, 165, 0, 255},
		StateBlocked:    {220, 50, 50, 255},
		StateSleepy:     {180, 180, 220, 255},
		StateSad:        {90, 150, 210, 255},
		StateSpeechless: {185, 185, 185, 255},
	}
)

// ── 狼犬基底臉 64x64 ──

func drawWolfBase(c *canvas64) {
	// ── 尖耳（左）──
	for yy := 0; yy < 16; yy++ {
		w := yy
		x0 := 8 - w
		if x0 < 4 {
			x0 = 4
		}
		x1 := 8 + w
		if x1 > 16 {
			x1 = 16
		}
		for xx := x0; xx <= x1; xx++ {
			c.set(xx, yy, wlfOutline)
		}
		for xx := x0 + 1; xx < x1; xx++ {
			if xx < (x0+x1)/2 {
				c.set(xx, yy, wlfEarIn)
			} else {
				c.set(xx, yy, wlfFur)
			}
		}
		if yy > 3 && yy < 12 {
			c.set(x0+2, yy, wlfEarH)
		}
	}

	// ── 尖耳（右）──
	for yy := 0; yy < 16; yy++ {
		w := yy
		x0 := 55 - w
		if x0 < 47 {
			x0 = 47
		}
		x1 := 55 + w
		if x1 > 59 {
			x1 = 59
		}
		for xx := x0; xx <= x1; xx++ {
			c.set(xx, yy, wlfOutline)
		}
		for xx := x0 + 1; xx < x1; xx++ {
			if xx > (x0+x1)/2 {
				c.set(xx, yy, wlfEarIn)
			} else {
				c.set(xx, yy, wlfFur)
			}
		}
		if yy > 3 && yy < 12 {
			c.set(x1-2, yy, wlfEarH)
		}
	}
	// Dark ear tips sharpen the silhouette.
	c.fillRect(6, 0, 9, 2, wlfOutline)
	c.fillRect(54, 0, 57, 2, wlfOutline)
	c.fillRect(4, 12, 7, 15, wlfOutline)
	c.fillRect(56, 12, 59, 15, wlfOutline)

	// 左耳環
	c.fillRect(1, 18, 3, 22, wlfEarring)
	c.set(2, 18, wlfEarringH)

	// ── 頭部 ──
	for yy := 12; yy < 50; yy++ {
		var margin int
		switch {
		case yy < 16:
			margin = 18 - (yy-12)*2
		case yy < 40:
			margin = 6
		case yy < 48:
			margin = 6 + (yy-40)*2
		default:
			margin = 22 + (yy-48)*4
		}
		lx := margin
		if lx < 4 {
			lx = 4
		}
		rx := 63 - margin
		if rx > 59 {
			rx = 59
		}
		c.set(lx, yy, wlfOutline)
		c.set(rx, yy, wlfOutline)
		for xx := lx + 1; xx < rx; xx++ {
			c.set(xx, yy, wlfFur)
		}
	}

	// 毛色暗面：側臉加深，讓狼味更兇。
	for yy := 16; yy < 48; yy++ {
		for xx := 5; xx < 14; xx++ {
			if colEq(c.get(xx, yy), wlfFur) {
				c.set(xx, yy, wlfFurD)
			}
		}
		for xx := 50; xx < 59; xx++ {
			if colEq(c.get(xx, yy), wlfFur) {
				c.set(xx, yy, wlfFurD)
			}
		}
	}
	// 毛色高光
	for yy := 14; yy < 20; yy++ {
		for _, xx := range []int{20, 21, 42, 43} {
			if colEq(c.get(xx, yy), wlfFur) {
				c.set(xx, yy, wlfFurH)
			}
		}
	}

	// ── 眼睛（窄長琥珀金）──
	drawWolfVillainEyes(c)

	// ── 口吻部 + 鼻 ──
	c.fillRect(20, 31, 43, 45, wlfMuzzle)
	for yy := 40; yy < 46; yy++ {
		for xx := 20; xx < 44; xx++ {
			if colEq(c.get(xx, yy), wlfMuzzle) {
				c.set(xx, yy, wlfMuzzleD)
			}
		}
	}
	c.fillRect(27, 32, 36, 36, wlfNose)
	c.set(30, 33, wlfFurD)
	c.set(31, 33, wlfFurD)

	// ── 嘴（預設反派壞笑）──
	drawWolfSmirk(c, 38, true, true)

	// ── 下巴 + 頸部 ──
	for yy := 48; yy < 54; yy++ {
		margin := 14 + (yy-48)*3
		for xx := margin; xx < 64-margin; xx++ {
			if colEq(c.get(xx, yy), wlfBG) {
				c.set(xx, yy, wlfFur)
			}
		}
		if margin-1 >= 0 {
			c.set(margin-1, yy, wlfOutline)
		}
		if 64-margin < 64 {
			c.set(64-margin, yy, wlfOutline)
		}
	}
	c.fillRect(24, 54, 39, 56, wlfFur)
	c.hline(57, 24, 39, wlfOutline)
}

func drawWolfVillainEyes(c *canvas64) {
	clearWolfEyes(c)
	// Lowered brows and slanted eyelids make the gaze narrow and predatory.
	c.hline(20, 12, 22, wlfBrow)
	c.hline(21, 13, 25, wlfBrow)
	c.hline(22, 16, 26, wlfBrow)
	c.hline(20, 41, 51, wlfBrow)
	c.hline(21, 38, 50, wlfBrow)
	c.hline(22, 37, 47, wlfBrow)

	c.fillRect(14, 24, 25, 27, wlfEye)
	c.fillRect(38, 24, 49, 27, wlfEye)
	c.hline(23, 13, 26, wlfOutline)
	c.hline(28, 13, 26, wlfOutline)
	c.hline(23, 37, 50, wlfOutline)
	c.hline(28, 37, 50, wlfOutline)
	c.set(13, 24, wlfOutline)
	c.set(13, 27, wlfOutline)
	c.set(26, 24, wlfOutline)
	c.set(26, 27, wlfOutline)
	c.set(37, 24, wlfOutline)
	c.set(37, 27, wlfOutline)
	c.set(50, 24, wlfOutline)
	c.set(50, 27, wlfOutline)

	c.fillRect(18, 24, 20, 27, wlfOutline)
	c.fillRect(43, 24, 45, 27, wlfOutline)
	c.set(16, 24, wlfFurH)
	c.set(41, 24, wlfFurH)
	c.set(24, 27, wlfEyeD)
	c.set(49, 27, wlfEyeD)
}

func clearWolfEyes(c *canvas64) {
	c.fillRect(11, 20, 27, 30, wlfFur)
	c.fillRect(36, 20, 52, 30, wlfFur)
	for yy := 20; yy <= 30; yy++ {
		for xx := 11; xx <= 14; xx++ {
			if colEq(c.get(xx, yy), wlfFur) {
				c.set(xx, yy, wlfFurD)
			}
		}
		for xx := 49; xx <= 52; xx++ {
			if colEq(c.get(xx, yy), wlfFur) {
				c.set(xx, yy, wlfFurD)
			}
		}
	}
}

func drawWolfSmirk(c *canvas64, y int, leftFang, rightFang bool) {
	c.hline(y, 25, 36, wlfMouth)
	c.hline(y-1, 36, 40, wlfMouth)
	c.set(24, y+1, wlfMouth)
	c.set(40, y-2, wlfMouth)
	c.fillRect(27, y+1, 36, y+2, wlfMouth)
	if leftFang {
		c.fillRect(24, y, 25, y+2, wlfFang)
	}
	if rightFang {
		c.fillRect(37, y-1, 39, y+1, wlfFang)
	}
}

// ── 狼犬狀態表情覆蓋 ──

func applyWolfState(c *canvas64, state AvatarStateTrigger) {
	accent := wlfStateColors[state]
	if accent.A == 0 {
		accent = wlfStateColors[StateIdle]
	}

	// 清除嘴巴區域
	c.fillRect(24, 37, 40, 44, wlfMuzzleD)

	switch state {
	case StateIdle:
		drawWolfSmirk(c, 38, true, true)

	case StateThinking:
		c.hline(39, 27, 39, wlfMouth)
		c.hline(38, 35, 40, wlfMouth)
		c.fillRect(8, 44, 11, 46, accent)
		c.fillRect(13, 42, 16, 44, accent)
		c.fillRect(18, 40, 21, 42, accent)
		c.set(24, 38, wlfFang)
		c.set(40, 37, wlfFang)

	case StateWorking:
		c.hline(38, 26, 37, wlfMouth)
		c.fillRect(28, 39, 35, 40, accent)
		c.hline(41, 26, 37, wlfMouth)
		c.hline(18, 15, 24, wlfBrow)
		c.hline(18, 39, 48, wlfBrow)

	case StateHappy:
		c.hline(37, 24, 39, wlfMouth)
		c.fillRect(24, 38, 25, 41, wlfFang)
		c.fillRect(38, 37, 40, 40, wlfFang)
		c.fillRect(26, 38, 37, 42, wlfMouth)
		c.hline(43, 26, 37, wlfMuzzle)

	case StateWarning:
		drawWolfSmirk(c, 37, true, true)
		c.fillRect(28, 38, 35, 41, wlfMouth)
		c.fillRect(10, 44, 14, 46, accent)
		c.fillRect(49, 44, 53, 46, accent)
		c.hline(19, 13, 25, wlfBrow)
		c.hline(19, 38, 50, wlfBrow)

	case StateBlocked:
		c.hline(36, 22, 41, wlfMouth)
		c.hline(37, 22, 41, wlfMouth)
		c.fillRect(24, 37, 27, 42, wlfFang)
		c.fillRect(36, 37, 40, 42, wlfFang)
		c.fillRect(28, 38, 35, 43, wlfMouth)
		c.fillRect(3, 18, 5, 20, accent)
		c.fillRect(58, 18, 60, 20, accent)
		c.hline(19, 12, 26, accent)
		c.hline(19, 37, 51, accent)

	case StateSleepy:
		// 眼半閉，但保留冷淡壓迫感。
		c.hline(26, 14, 25, wlfOutline)
		c.hline(26, 38, 49, wlfOutline)
		c.fillRect(28, 39, 35, 42, wlfMouth)
		c.set(25, 38, wlfFang)
		c.set(38, 38, wlfFang)
		c.fillRect(50, 8, 54, 10, accent)
		c.fillRect(52, 5, 56, 7, accent)
		c.fillRect(54, 2, 58, 4, accent)

	case StateSad:
		c.hline(41, 26, 37, wlfMouth)
		c.hline(40, 24, 25, wlfMouth)
		c.hline(40, 38, 39, wlfMouth)
		c.fillRect(23, 31, 24, 35, accent)
		c.fillRect(49, 31, 50, 35, accent)
		c.hline(22, 13, 25, wlfBrow)
		c.hline(22, 38, 50, wlfBrow)

	case StateSpeechless:
		c.hline(40, 27, 36, accent)
		c.hline(41, 28, 35, accent)
		c.fillRect(8, 16, 9, 20, accent)
		c.fillRect(54, 16, 55, 20, accent)
		c.hline(23, 15, 24, wlfBrow)
		c.hline(23, 39, 48, wlfBrow)
	}
}

// ──────────────────────────────────────────────
// 公開 API — 狼犬版
// ──────────────────────────────────────────────

// RenderPixelAvatar 渲染指定狀態的狼犬 pixel art 頭像。
// 預設產出 128x128；仍接受 256x256 供高解析預覽使用。
func RenderPixelAvatar(state AvatarStateTrigger, size int) ([]byte, error) {
	c := newPixelCanvas(size)
	drawWolfBase(c)
	applyWolfState(c, state)

	var buf []byte
	w := &byteWriter{buf: &buf}
	if err := png.Encode(w, c.img); err != nil {
		return nil, fmt.Errorf("PNG 編碼失敗: %w", err)
	}
	return buf, nil
}

// byteWriter 將 png.Encode 的輸出寫入 byte slice。
type byteWriter struct {
	buf *[]byte
}

func (w *byteWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// GenerateAllPixelAvatars 產生所有狀態的 128x128 PNG。
func GenerateAllPixelAvatars(outputDir string) error {
	sizes := []int{defaultPixelAvatarSize}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("建立輸出目錄失敗: %w", err)
	}
	for _, state := range AllStateTriggers {
		for _, size := range sizes {
			data, err := RenderPixelAvatar(state, size)
			if err != nil {
				return fmt.Errorf("渲染 %s_%d 失敗: %w", state, size, err)
			}
			filename := fmt.Sprintf("pixel_%s_%d.png", state, size)
			path := filepath.Join(outputDir, filename)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				return fmt.Errorf("寫入 %s 失敗: %w", filename, err)
			}
		}
	}
	return nil
}

// GetPixelAvatarPath 取得指定狀態和尺寸的 pixel avatar 檔案路徑。
func GetPixelAvatarPath(outputDir string, state AvatarStateTrigger, size int) string {
	return filepath.Join(outputDir, fmt.Sprintf("pixel_%s_%d.png", state, size))
}
