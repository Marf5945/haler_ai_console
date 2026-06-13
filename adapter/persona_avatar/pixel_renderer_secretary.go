// persona_avatar/pixel_renderer_secretary.go — Built-in Pixel Avatar 渲染器（女秘書版 64x64）。
// 用 Go image 標準庫以程序化方式繪製 pixel art PNG。
// 9 種狀態 × 2 種尺寸（256 / 128），自製素材避免授權風險。
// v1.0 — 可愛女秘書：深棕鮑伯頭、藍框眼鏡、金色髮夾、腮紅、紅蝴蝶結。
package persona_avatar

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// ──────────────────────────────────────────────
// 像素顏色定義 — 女秘書配色（64x64 版）
// ──────────────────────────────────────────────

var (
	secBG       = color.RGBA{0, 0, 0, 0}
	secOutline  = color.RGBA{40, 30, 50, 255}
	secSkin     = color.RGBA{240, 210, 185, 255}
	secSkinD    = color.RGBA{220, 190, 165, 255}
	secEyeWhite = color.RGBA{240, 238, 235, 255}
	secPupil    = color.RGBA{100, 65, 45, 255}  // 深棕瞳色
	secPupilH   = color.RGBA{160, 120, 85, 255} // 瞳孔高光
	secMouth    = color.RGBA{200, 110, 110, 255}
	secHair     = color.RGBA{45, 25, 15, 255}    // 巧克力棕髮
	secHairH    = color.RGBA{75, 50, 35, 255}    // 髮絲高光
	secGlassF   = color.RGBA{80, 130, 180, 255}  // 眼鏡框（深藍）
	secGlassL   = color.RGBA{200, 220, 240, 80}  // 鏡片（半透明）
	secBlush    = color.RGBA{245, 175, 160, 255} // 腮紅
	secBrow     = color.RGBA{55, 35, 25, 255}
	secTeeth    = color.RGBA{245, 242, 238, 255}
	secLip      = color.RGBA{210, 100, 100, 255}
	secLipD     = color.RGBA{180, 75, 80, 255}
	secNose     = color.RGBA{220, 185, 155, 255}
	secNoseD    = color.RGBA{200, 165, 135, 255}
	secCollar   = color.RGBA{240, 240, 245, 255} // 白色領口
	secCollarD  = color.RGBA{215, 218, 225, 255}
	secRibbon   = color.RGBA{220, 60, 80, 255} // 紅色蝴蝶結
	secRibbonD  = color.RGBA{185, 45, 65, 255}
	secRibbonH  = color.RGBA{255, 100, 120, 255}
	secEyelash  = color.RGBA{35, 25, 20, 255}
	secHairpin  = color.RGBA{255, 200, 60, 255} // 金色髮夾

	secStateColors = map[AvatarStateTrigger]color.RGBA{
		StateIdle:       {240, 210, 185, 255},
		StateThinking:   {140, 170, 230, 255},
		StateWorking:    {100, 200, 140, 255},
		StateHappy:      {255, 200, 80, 255},
		StateWarning:    {255, 175, 60, 255},
		StateBlocked:    {230, 80, 80, 255},
		StateSleepy:     {190, 185, 220, 255},
		StateSad:        {120, 160, 210, 255},
		StateSpeechless: {185, 185, 185, 255},
	}
)

// ── 繪圖工具（複用 canvas64 / byteWriter / colEq） ──
// canvas64, byteWriter, colEq 已定義於 pixel_renderer_uncle.go

// ── 女秘書基底臉 ──

func drawSecretaryBase(c *canvas64) {
	// ── 頭髮（dome — 鮑伯頭） ──
	c.fillRect(18, 0, 45, 2, secHair)
	c.fillRect(14, 3, 49, 5, secHair)
	c.fillRect(10, 6, 53, 8, secHair)
	c.fillRect(7, 9, 56, 12, secHair)
	c.fillRect(5, 13, 58, 16, secHair)
	// 兩側鮑伯頭垂髮
	c.fillRect(3, 14, 7, 40, secHair)
	c.fillRect(56, 14, 60, 40, secHair)
	c.fillRect(4, 41, 8, 46, secHair)
	c.fillRect(55, 41, 59, 46, secHair)
	// 髮絲高光
	for _, yy := range []int{4, 7, 10, 14} {
		for _, xx := range []int{20, 28, 36, 44} {
			c.set(xx, yy, secHairH)
			c.set(xx+1, yy, secHairH)
		}
	}
	// 側髮高光
	for yy := 18; yy < 40; yy += 5 {
		c.set(5, yy, secHairH)
		c.set(57, yy, secHairH)
	}

	// ── 金色髮夾（左側） ──
	c.fillRect(8, 18, 11, 19, secHairpin)
	c.fillRect(9, 17, 10, 20, secHairpin)
	c.set(9, 16, secHairpin)
	c.set(10, 21, secHairpin)

	// ── 臉部輪廓（圓潤型）──
	for yy := 14; yy < 52; yy++ {
		var w int
		switch {
		case yy < 18:
			w = 8 - (yy - 14)
		case yy < 42:
			w = 6
		case yy < 48:
			w = 6 + (yy-42)*2
		default:
			w = 14 + (yy-48)*3
		}
		lx := w
		rx := 63 - w
		if lx < 8 {
			lx = 8
		}
		if rx > 55 {
			rx = 55
		}
		c.set(lx, yy, secOutline)
		c.set(rx, yy, secOutline)
		for xx := lx + 1; xx < rx; xx++ {
			c.set(xx, yy, secSkin)
		}
	}

	// 膚色暗面（兩側）
	for yy := 18; yy < 44; yy++ {
		for xx := 9; xx < 12; xx++ {
			if colEq(c.get(xx, yy), secSkin) {
				c.set(xx, yy, secSkinD)
			}
		}
		for xx := 52; xx < 55; xx++ {
			if colEq(c.get(xx, yy), secSkin) {
				c.set(xx, yy, secSkinD)
			}
		}
	}

	// ── 瀏海 ──
	c.fillRect(8, 14, 55, 16, secHair)
	for xx := 9; xx < 55; xx++ {
		if xx%3 != 0 {
			c.set(xx, 17, secHair)
		}
		if xx%5 < 2 {
			c.set(xx, 18, secHair)
		}
	}

	// ── 眉毛（細、弧形）──
	c.hline(19, 14, 24, secBrow)
	c.hline(18, 16, 22, secBrow)
	c.hline(19, 39, 49, secBrow)
	c.hline(18, 41, 47, secBrow)

	// ── 圓框眼鏡 ──
	// 輔助：Bresenham 圓
	drawCircle := func(cx, cy, r int, col color.RGBA) {
		x, y, d := 0, r, 3-2*r
		for x <= y {
			for _, p := range [][2]int{
				{cx + x, cy + y}, {cx - x, cy + y}, {cx + x, cy - y}, {cx - x, cy - y},
				{cx + y, cy + x}, {cx - y, cy + x}, {cx + y, cy - x}, {cx - y, cy - x},
			} {
				c.set(p[0], p[1], col)
			}
			if d < 0 {
				d += 4*x + 6
			} else {
				d += 4*(x-y) + 10
				y--
			}
			x++
		}
	}
	fillCircle := func(cx, cy, r int, col color.RGBA) {
		for yy := cy - r; yy <= cy+r; yy++ {
			for xx := cx - r; xx <= cx+r; xx++ {
				if (xx-cx)*(xx-cx)+(yy-cy)*(yy-cy) <= r*r {
					c.set(xx, yy, col)
				}
			}
		}
	}

	// 左鏡片 — 圓心 (19, 26), 半徑 6
	fillCircle(19, 26, 5, secGlassL)
	drawCircle(19, 26, 6, secGlassF)
	drawCircle(19, 26, 7, secGlassF)
	// 右鏡片 — 圓心 (44, 26), 半徑 6
	fillCircle(44, 26, 5, secGlassL)
	drawCircle(44, 26, 6, secGlassF)
	drawCircle(44, 26, 7, secGlassF)
	// 鼻樑（微弧）
	c.hline(25, 25, 38, secGlassF)
	c.hline(24, 27, 36, secGlassF)
	// 鏡腳
	c.hline(25, 8, 13, secGlassF)
	c.hline(25, 50, 55, secGlassF)

	// ── 眼睛（圓框鏡片內，大而可愛）──
	// 左眼
	c.fillRect(15, 23, 23, 29, secEyeWhite)
	c.fillRect(17, 24, 22, 28, secPupil)
	c.fillRect(18, 25, 21, 27, secOutline)
	c.set(19, 24, secPupilH)
	c.set(20, 24, secPupilH)
	c.set(19, 25, secPupilH)
	// 睫毛（從圓框上方翹出）
	c.set(13, 20, secEyelash)
	c.set(14, 19, secEyelash)
	c.set(15, 19, secEyelash)
	c.set(23, 19, secEyelash)
	c.set(24, 19, secEyelash)
	c.set(25, 20, secEyelash)

	// 右眼
	c.fillRect(40, 23, 48, 29, secEyeWhite)
	c.fillRect(41, 24, 46, 28, secPupil)
	c.fillRect(42, 25, 45, 27, secOutline)
	c.set(43, 24, secPupilH)
	c.set(44, 24, secPupilH)
	c.set(43, 25, secPupilH)
	// 睫毛
	c.set(38, 20, secEyelash)
	c.set(39, 19, secEyelash)
	c.set(40, 19, secEyelash)
	c.set(48, 19, secEyelash)
	c.set(49, 19, secEyelash)
	c.set(50, 20, secEyelash)

	// ── 腮紅 ──
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if dx*dx+dy*dy <= 5 {
				c.set(14+dx, 35+dy, secBlush)
				c.set(49+dx, 35+dy, secBlush)
			}
		}
	}

	// ── 鼻子（小巧）──
	c.fillRect(30, 32, 33, 34, secNose)
	c.set(30, 35, secNoseD)
	c.set(33, 35, secNoseD)

	// ── 嘴巴（微笑）──
	c.hline(39, 26, 37, secLip)
	c.hline(40, 27, 36, secLipD)
	c.set(25, 38, secLip)
	c.set(38, 38, secLip)
	c.hline(39, 28, 35, secTeeth)

	// ── 下巴 + 頸部 ──
	for yy := 50; yy < 54; yy++ {
		margin := 14 + (yy-50)*3
		for xx := margin; xx < 64-margin; xx++ {
			if colEq(c.get(xx, yy), secBG) {
				c.set(xx, yy, secSkin)
			}
		}
		if margin-1 >= 0 {
			c.set(margin-1, yy, secOutline)
		}
		if 64-margin < 64 {
			c.set(64-margin, yy, secOutline)
		}
	}

	// ── 白色領口 ──
	c.fillRect(20, 54, 43, 58, secCollar)
	c.fillRect(22, 59, 41, 62, secCollarD)
	// V 領
	for i := 0; i < 4; i++ {
		c.set(31-i, 54+i, secSkin)
		c.set(32+i, 54+i, secSkin)
	}

	// ── 紅色蝴蝶結 ──
	c.fillRect(27, 55, 30, 57, secRibbon)
	c.fillRect(33, 55, 36, 57, secRibbon)
	c.fillRect(30, 55, 33, 58, secRibbonD)
	c.set(31, 56, secRibbon)
	c.set(32, 56, secRibbon)
	c.set(28, 55, secRibbonH)
	c.set(35, 55, secRibbonH)
}

// ── 狀態表情覆蓋 ──

func applySecretaryState(c *canvas64, state AvatarStateTrigger) {
	accent := secStateColors[state]
	if accent.A == 0 {
		accent = secStateColors[StateIdle]
	}

	// 清除嘴巴區域
	for yy := 37; yy < 45; yy++ {
		for xx := 22; xx < 42; xx++ {
			cur := c.get(xx, yy)
			if !colEq(cur, secOutline) && !colEq(cur, secSkin) && !colEq(cur, secSkinD) &&
				!colEq(cur, secNose) && !colEq(cur, secNoseD) && !colEq(cur, secBlush) {
				c.set(xx, yy, secSkin)
			}
		}
	}

	switch state {
	case StateIdle:
		// 微笑
		c.hline(39, 26, 37, secLip)
		c.hline(40, 27, 36, secLipD)
		c.set(25, 38, secLip)
		c.set(38, 38, secLip)
		c.hline(39, 28, 35, secTeeth)

	case StateThinking:
		// 嘟嘴偏右 + 思考泡泡
		c.fillRect(33, 39, 40, 41, secLip)
		c.fillRect(34, 40, 39, 40, secLipD)
		// 思考泡泡
		c.fillRect(10, 44, 12, 46, accent)
		c.fillRect(14, 42, 16, 44, accent)
		c.fillRect(18, 40, 20, 42, accent)

	case StateWorking:
		// 咬唇認真
		c.hline(39, 26, 37, secLip)
		c.hline(40, 28, 35, secTeeth)
		c.hline(41, 27, 36, secLipD)
		// 進度指示
		c.fillRect(28, 43, 35, 44, accent)

	case StateHappy:
		// 大笑露齒 + 瞇眼
		c.hline(38, 24, 39, secLip)
		c.fillRect(24, 39, 39, 40, secLip)
		c.fillRect(26, 40, 37, 41, secTeeth)
		c.fillRect(25, 42, 38, 43, secLipD)
		// 瞇眼效果（覆蓋下半眼睛）
		for yy := 27; yy <= 29; yy++ {
			for xx := 14; xx <= 24; xx++ {
				if !colEq(c.get(xx, yy), secGlassF) {
					c.set(xx, yy, secSkin)
				}
			}
			for xx := 39; xx <= 49; xx++ {
				if !colEq(c.get(xx, yy), secGlassF) {
					c.set(xx, yy, secSkin)
				}
			}
		}

	case StateWarning:
		// 微張嘴驚訝
		c.fillRect(28, 38, 35, 42, secLip)
		c.fillRect(29, 39, 34, 41, secTeeth)
		// 驚嘆號
		c.fillRect(8, 44, 10, 47, accent)
		c.fillRect(53, 44, 55, 47, accent)

	case StateBlocked:
		// 生氣嘟嘴
		c.hline(39, 24, 39, secLipD)
		c.fillRect(26, 40, 37, 41, secLip)
		c.fillRect(27, 42, 36, 42, secLipD)
		// 怒火
		c.fillRect(4, 18, 6, 20, accent)
		c.fillRect(57, 18, 59, 20, accent)

	case StateSleepy:
		// 半閉眼 + ZZZ
		for yy := 26; yy <= 29; yy++ {
			for xx := 14; xx <= 24; xx++ {
				if !colEq(c.get(xx, yy), secGlassF) {
					c.set(xx, yy, secSkin)
				}
			}
			for xx := 39; xx <= 49; xx++ {
				if !colEq(c.get(xx, yy), secGlassF) {
					c.set(xx, yy, secSkin)
				}
			}
		}
		// 小口打哈欠
		c.fillRect(29, 39, 34, 42, secLip)
		c.fillRect(30, 40, 33, 41, secMouth)
		// ZZZ
		c.fillRect(50, 10, 54, 12, accent)
		c.fillRect(52, 7, 56, 9, accent)
		c.fillRect(54, 4, 58, 6, accent)

	case StateSad:
		// 扁嘴
		c.hline(41, 26, 37, secLip)
		c.hline(40, 24, 25, secLip)
		c.hline(40, 38, 39, secLip)
		c.hline(42, 27, 36, secSkin)
		// 淚珠
		c.fillRect(24, 30, 25, 34, accent)
		c.fillRect(50, 30, 51, 34, accent)

	case StateSpeechless:
		// 一字嘴 + 汗滴
		c.hline(40, 26, 37, accent)
		c.hline(41, 27, 36, accent)
		c.fillRect(8, 18, 9, 22, accent)
		c.fillRect(54, 18, 55, 22, accent)
	}
}

// ──────────────────────────────────────────────
// 公開 API
// ──────────────────────────────────────────────

// RenderSecretaryPixelAvatar 渲染指定狀態的女秘書 pixel art 頭像。
// 預設產出 128x128；仍接受 256x256 供高解析預覽使用。
func RenderSecretaryPixelAvatar(state AvatarStateTrigger, size int) ([]byte, error) {
	c := newPixelCanvas(size)
	drawSecretaryBase(c)
	applySecretaryState(c, state)

	var buf []byte
	w := &byteWriter{buf: &buf}
	if err := png.Encode(w, c.img); err != nil {
		return nil, fmt.Errorf("PNG 編碼失敗: %w", err)
	}
	return buf, nil
}

// GenerateAllSecretaryPixelAvatars 產生所有狀態的 128x128 PNG。
func GenerateAllSecretaryPixelAvatars(outputDir string) error {
	sizes := []int{defaultPixelAvatarSize}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("建立輸出目錄失敗: %w", err)
	}
	for _, state := range AllStateTriggers {
		for _, size := range sizes {
			data, err := RenderSecretaryPixelAvatar(state, size)
			if err != nil {
				return fmt.Errorf("渲染 %s_%d 失敗: %w", state, size, err)
			}
			filename := fmt.Sprintf("secretary_%s_%d.png", state, size)
			path := filepath.Join(outputDir, filename)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				return fmt.Errorf("寫入 %s 失敗: %w", filename, err)
			}
		}
	}
	return nil
}

// GetSecretaryPixelAvatarPath 取得指定狀態和尺寸的女秘書 pixel avatar 檔案路徑。
func GetSecretaryPixelAvatarPath(outputDir string, state AvatarStateTrigger, size int) string {
	return filepath.Join(outputDir, fmt.Sprintf("secretary_%s_%d.png", state, size))
}
