// persona_avatar/pixel_renderer_uncle.go — Built-in Pixel Avatar 渲染器（大叔版 64x64）。
// 用 Go image 標準庫以程序化方式繪製 pixel art PNG。
// 9 種狀態 × 2 種尺寸（256 / 128），自製素材避免授權風險。
// v3.0 — 帥氣大叔版：修長臉型、銳利杏仁眼、有型短髮、短鬍渣、雙耳金耳環。
package persona_avatar

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// ──────────────────────────────────────────────
// 像素顏色定義 — 帥氣大叔配色（64x64 版）
// ──────────────────────────────────────────────

var (
	uclBG       = color.RGBA{0, 0, 0, 0}
	uclOutline  = color.RGBA{28, 30, 42, 255}
	uclSkin     = color.RGBA{215, 180, 148, 255}
	uclSkinD    = color.RGBA{192, 158, 125, 255}
	uclSkinDD   = color.RGBA{175, 140, 108, 255} // 深陰影（顎線）
	uclEyeWhite = color.RGBA{238, 235, 230, 255}
	uclPupil    = color.RGBA{145, 175, 205, 255} // 冷藍灰瞳
	uclPupilD   = color.RGBA{100, 135, 170, 255} // 瞳暗環
	uclPupilH   = color.RGBA{195, 218, 240, 255} // 瞳高光
	uclMouth    = color.RGBA{55, 38, 48, 255}
	uclHair     = color.RGBA{28, 26, 30, 255}
	uclHairH    = color.RGBA{55, 52, 60, 255}
	uclHairHH   = color.RGBA{78, 74, 85, 255} // 亮高光
	uclEarring  = color.RGBA{218, 185, 50, 255}
	uclEarringH = color.RGBA{245, 215, 85, 255}
	uclStubble  = color.RGBA{188, 155, 128, 255} // 短鬍渣
	uclStubbleD = color.RGBA{170, 138, 112, 255} // 鬍渣深色
	uclNose     = color.RGBA{195, 155, 122, 255}
	uclNoseD    = color.RGBA{175, 135, 102, 255}
	uclNoseH    = color.RGBA{228, 198, 172, 255} // 鼻樑高光
	uclBrow     = color.RGBA{32, 30, 36, 255}
	uclTeeth    = color.RGBA{240, 238, 230, 255}
	uclLip      = color.RGBA{180, 125, 105, 255}
	uclLipD     = color.RGBA{155, 100, 85, 255}
	uclNeckSh   = color.RGBA{185, 150, 118, 255}
	uclCheekH   = color.RGBA{225, 192, 162, 255} // 頰高光

	uclStateColors = map[AvatarStateTrigger]color.RGBA{
		StateIdle:       {215, 180, 148, 255},
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

// ── 大叔基底臉（v3 帥氣版）──

func drawUncleBase(c *canvas64) {
	// ── 有型短髮（頂部蓬鬆、兩側收短）──
	c.fillRect(19, 0, 44, 1, uclHair)
	c.fillRect(15, 2, 48, 3, uclHair)
	c.fillRect(11, 4, 52, 5, uclHair)
	c.fillRect(9, 6, 54, 8, uclHair)
	c.fillRect(7, 9, 56, 11, uclHair)
	c.fillRect(6, 12, 57, 14, uclHair)
	// 髮絲高光（掃向一側的造型感）
	for _, yy := range []int{3, 5, 7, 9, 11, 13} {
		off := yy % 3
		c.set(19+off, yy, uclHairH)
		c.set(20+off, yy, uclHairHH)
		c.set(27+yy%2, yy, uclHairH)
		c.set(28+yy%2, yy, uclHairHH)
		c.set(35+off, yy, uclHairH)
		c.set(42+yy%2, yy, uclHairH)
		c.set(43+yy%2, yy, uclHairHH)
	}
	c.hline(1, 22, 28, uclHairH)
	c.hline(0, 24, 32, uclHairHH)

	// ── 修長臉部輪廓 ──
	for yy := 13; yy < 55; yy++ {
		var w int
		switch {
		case yy < 16:
			w = 9 - (yy - 13)
		case yy < 20:
			w = 7
		case yy < 37:
			w = 7
		case yy < 41:
			w = 7 + (yy - 37)
		case yy < 47:
			w = 11 + (yy-41)*2
		case yy < 51:
			w = 23 + (yy-47)*3
		default:
			w = 35 + (yy-51)*4
		}
		if w < 6 {
			w = 6
		}
		lx := w
		rx := 63 - w
		if lx < 5 {
			lx = 5
		}
		if rx > 58 {
			rx = 58
		}
		if lx >= rx {
			continue
		}
		c.set(lx, yy, uclOutline)
		c.set(rx, yy, uclOutline)
		for xx := lx + 1; xx < rx; xx++ {
			c.set(xx, yy, uclSkin)
		}
	}

	// 側面陰影
	for yy := 16; yy < 50; yy++ {
		for xx := 7; xx < 12; xx++ {
			if colEq(c.get(xx, yy), uclSkin) {
				c.set(xx, yy, uclSkinD)
			}
		}
		for xx := 52; xx < 57; xx++ {
			if colEq(c.get(xx, yy), uclSkin) {
				c.set(xx, yy, uclSkinD)
			}
		}
	}
	// 顎線深陰影
	for yy := 41; yy < 52; yy++ {
		for xx := 10; xx < 16; xx++ {
			p := c.get(xx, yy)
			if colEq(p, uclSkin) || colEq(p, uclSkinD) {
				c.set(xx, yy, uclSkinDD)
			}
		}
		for xx := 48; xx < 54; xx++ {
			p := c.get(xx, yy)
			if colEq(p, uclSkin) || colEq(p, uclSkinD) {
				c.set(xx, yy, uclSkinDD)
			}
		}
	}
	// 頰骨高光
	for yy := 28; yy < 34; yy++ {
		for _, xx := range []int{15, 16, 17} {
			if colEq(c.get(xx, yy), uclSkin) {
				c.set(xx, yy, uclCheekH)
			}
		}
		for _, xx := range []int{46, 47, 48} {
			if colEq(c.get(xx, yy), uclSkin) {
				c.set(xx, yy, uclCheekH)
			}
		}
	}

	// ── 瀏海（鋸齒質感）──
	c.fillRect(6, 13, 57, 15, uclHair)
	for xx := 7; xx < 57; xx++ {
		if xx%3 != 1 {
			c.set(xx, 16, uclHair)
		}
		if xx%4 == 0 {
			c.set(xx, 17, uclHair)
		}
	}
	// 鬢角
	for yy := 15; yy < 26; yy++ {
		c.set(6, yy, uclHair)
		c.set(7, yy, uclHair)
		c.set(56, yy, uclHair)
		c.set(57, yy, uclHair)
	}
	for yy := 15; yy < 20; yy++ {
		c.set(8, yy, uclHair)
		c.set(55, yy, uclHair)
	}

	// ── 銳角粗眉（內粗外尖）──
	c.fillRect(13, 19, 17, 21, uclBrow)
	c.fillRect(18, 19, 23, 20, uclBrow)
	c.fillRect(24, 19, 27, 19, uclBrow)
	c.set(13, 22, uclBrow)
	c.set(14, 22, uclBrow)
	c.fillRect(46, 19, 50, 21, uclBrow)
	c.fillRect(40, 19, 45, 20, uclBrow)
	c.fillRect(36, 19, 39, 19, uclBrow)
	c.set(49, 22, uclBrow)
	c.set(50, 22, uclBrow)

	// ── 銳利杏仁眼 ──
	drawUncleSharpEyes(c)

	// ── 耳環 ──
	c.fillRect(2, 26, 4, 30, uclEarring)
	c.set(3, 26, uclEarringH)
	c.set(3, 27, uclEarringH)
	c.fillRect(59, 26, 61, 30, uclEarring)
	c.set(60, 26, uclEarringH)
	c.set(60, 27, uclEarringH)

	// ── 鼻子 ──
	c.fillRect(29, 30, 34, 34, uclNose)
	c.set(31, 29, uclNoseH)
	c.set(32, 29, uclNoseH)
	c.set(31, 30, uclNoseH)
	c.set(32, 30, uclNoseH)
	c.fillRect(29, 35, 34, 36, uclNoseD)
	c.set(29, 35, uclOutline)
	c.set(30, 35, uclOutline)
	c.set(33, 35, uclOutline)
	c.set(34, 35, uclOutline)

	// ── 短鬍渣 ──
	drawUncleStubble(c)

	// ── 嘴巴（自信壞笑）──
	c.hline(39, 23, 40, uclLip)
	c.hline(40, 25, 38, uclLipD)
	c.set(40, 38, uclMouth)
	c.set(41, 38, uclMouth)
	c.set(42, 37, uclMouth)
	c.set(22, 39, uclMouth)
	c.set(23, 40, uclMouth)

	// ── 下巴 + 頸部 ──
	for yy := 52; yy < 56; yy++ {
		margin := 18 + (yy-52)*4
		for xx := margin; xx < 64-margin; xx++ {
			if colEq(c.get(xx, yy), uclBG) {
				c.set(xx, yy, uclSkin)
			}
		}
		if margin-1 >= 0 {
			c.set(margin-1, yy, uclOutline)
		}
		if 64-margin < 64 {
			c.set(64-margin, yy, uclOutline)
		}
	}
	c.fillRect(25, 56, 38, 59, uclSkin)
	c.fillRect(27, 57, 36, 59, uclNeckSh)
	c.hline(60, 25, 38, uclOutline)
}

func drawUncleSharpEyes(c *canvas64) {
	// 左眼
	c.hline(22, 12, 27, uclOutline)         // 上眼瞼
	c.fillRect(13, 23, 26, 26, uclEyeWhite) // 眼白（窄：4px 高）
	c.hline(27, 15, 25, uclOutline)         // 下眼瞼
	c.set(12, 23, uclOutline)
	c.set(12, 24, uclOutline) // 內眼角
	c.set(27, 24, uclOutline)
	c.set(27, 25, uclOutline) // 外眼角
	c.set(28, 23, uclOutline) // 外眼尾上勾
	// 虹膜
	c.fillRect(17, 23, 23, 26, uclPupil)
	c.set(16, 24, uclPupilD)
	c.set(16, 25, uclPupilD)
	c.set(24, 24, uclPupilD)
	c.set(24, 25, uclPupilD)
	// 瞳孔
	c.fillRect(19, 24, 21, 25, uclOutline)
	// 高光
	c.set(17, 23, uclPupilH)
	c.set(18, 23, uclPupilH)
	// 眼下陰影（增加深邃感）
	c.set(14, 28, uclSkinD)
	c.set(15, 28, uclSkinD)
	c.set(16, 28, uclSkinD)

	// 右眼（鏡射）
	c.hline(22, 36, 51, uclOutline)
	c.fillRect(37, 23, 50, 26, uclEyeWhite)
	c.hline(27, 38, 48, uclOutline)
	c.set(51, 23, uclOutline)
	c.set(51, 24, uclOutline)
	c.set(36, 24, uclOutline)
	c.set(36, 25, uclOutline)
	c.set(35, 23, uclOutline)
	c.fillRect(40, 23, 46, 26, uclPupil)
	c.set(39, 24, uclPupilD)
	c.set(39, 25, uclPupilD)
	c.set(47, 24, uclPupilD)
	c.set(47, 25, uclPupilD)
	c.fillRect(42, 24, 44, 25, uclOutline)
	c.set(41, 23, uclPupilH)
	c.set(42, 23, uclPupilH)
	c.set(47, 28, uclSkinD)
	c.set(48, 28, uclSkinD)
	c.set(49, 28, uclSkinD)
}

func clearUncleEyes(c *canvas64) {
	// 清除眼睛區域，用於半閉眼等狀態
	for yy := 22; yy <= 28; yy++ {
		for xx := 12; xx <= 28; xx++ {
			if !colEq(c.get(xx, yy), uclOutline) || yy >= 22 {
				c.set(xx, yy, uclSkin)
			}
		}
		for xx := 35; xx <= 51; xx++ {
			if !colEq(c.get(xx, yy), uclOutline) || yy >= 22 {
				c.set(xx, yy, uclSkin)
			}
		}
	}
	// 補回側面陰影
	for yy := 22; yy <= 28; yy++ {
		for xx := 7; xx < 12; xx++ {
			if colEq(c.get(xx, yy), uclSkin) {
				c.set(xx, yy, uclSkinD)
			}
		}
		for xx := 52; xx < 57; xx++ {
			if colEq(c.get(xx, yy), uclSkin) {
				c.set(xx, yy, uclSkinD)
			}
		}
	}
}

func drawUncleStubble(c *canvas64) {
	// 全臉鬍渣
	for yy := 40; yy < 52; yy++ {
		for xx := 14; xx < 50; xx++ {
			p := c.get(xx, yy)
			if colEq(p, uclSkin) || colEq(p, uclSkinD) {
				v := (xx*7 + yy*11) % 13
				if v < 3 {
					c.set(xx, yy, uclStubble)
				} else if v < 5 {
					c.set(xx, yy, uclStubbleD)
				}
			}
		}
	}
	// 下巴中央更密
	for yy := 44; yy < 51; yy++ {
		for xx := 20; xx < 44; xx++ {
			if colEq(c.get(xx, yy), uclSkin) {
				if (xx+yy)%3 != 0 {
					c.set(xx, yy, uclStubble)
					if (xx*3+yy)%7 < 2 {
						c.set(xx, yy, uclStubbleD)
					}
				}
			}
		}
	}
	// 上唇鬍渣
	for yy := 37; yy < 40; yy++ {
		for xx := 20; xx < 44; xx++ {
			if colEq(c.get(xx, yy), uclSkin) {
				if (xx+yy*3)%7 < 2 {
					c.set(xx, yy, uclStubble)
				}
			}
		}
	}
}

// ── 狀態表情覆蓋 ──

func applyUncleState(c *canvas64, state AvatarStateTrigger) {
	accent := uclStateColors[state]
	if accent.A == 0 {
		accent = uclStateColors[StateIdle]
	}

	// 清除嘴巴區域，再補回鬍渣
	for yy := 37; yy < 44; yy++ {
		for xx := 18; xx < 46; xx++ {
			p := c.get(xx, yy)
			if !colEq(p, uclOutline) && !colEq(p, uclNose) && !colEq(p, uclNoseD) && !colEq(p, uclNoseH) {
				c.set(xx, yy, uclSkin)
			}
		}
	}
	// 補回上唇鬍渣
	for yy := 37; yy < 40; yy++ {
		for xx := 20; xx < 44; xx++ {
			if colEq(c.get(xx, yy), uclSkin) {
				if (xx+yy*3)%7 < 2 {
					c.set(xx, yy, uclStubble)
				}
			}
		}
	}

	switch state {
	case StateIdle:
		// 自信壞笑
		c.hline(39, 23, 40, uclLip)
		c.hline(40, 25, 38, uclLipD)
		c.set(40, 38, uclMouth)
		c.set(41, 38, uclMouth)
		c.set(42, 37, uclMouth)
		c.set(22, 39, uclMouth)
		c.set(23, 40, uclMouth)

	case StateThinking:
		// 嘴巴偏一邊 + 思考泡泡
		c.hline(40, 30, 41, uclLip)
		c.hline(41, 31, 40, uclLipD)
		c.set(42, 39, uclMouth)
		c.fillRect(10, 44, 12, 46, accent)
		c.fillRect(14, 42, 16, 44, accent)
		c.fillRect(18, 40, 20, 42, accent)

	case StateWorking:
		// 嘴巴微張露齒
		c.hline(39, 24, 39, uclLip)
		c.fillRect(26, 40, 37, 40, uclTeeth)
		c.hline(41, 25, 38, uclLipD)
		c.fillRect(28, 42, 35, 43, accent)

	case StateHappy:
		// 開心大笑
		c.hline(38, 22, 41, uclLip)
		c.fillRect(22, 39, 41, 39, uclMouth)
		c.fillRect(24, 40, 39, 41, uclTeeth)
		c.fillRect(23, 42, 40, 43, uclMouth)
		c.hline(44, 25, 38, uclLipD)

	case StateWarning:
		// 咬牙 + 警告符號
		c.hline(38, 22, 41, uclLip)
		c.fillRect(24, 39, 27, 40, uclTeeth)
		c.fillRect(28, 39, 35, 40, uclMouth)
		c.fillRect(36, 39, 39, 40, uclTeeth)
		c.hline(41, 23, 40, uclLipD)
		c.fillRect(10, 45, 14, 47, accent)
		c.fillRect(49, 45, 53, 47, accent)

	case StateBlocked:
		// 張嘴驚訝 + 怒氣符號
		c.hline(37, 22, 41, uclMouth)
		c.fillRect(22, 38, 41, 38, uclMouth)
		c.fillRect(24, 39, 27, 40, uclTeeth)
		c.fillRect(36, 39, 39, 40, uclTeeth)
		c.fillRect(28, 39, 35, 42, uclMouth)
		c.hline(43, 23, 40, uclLipD)
		c.fillRect(3, 18, 5, 20, accent)
		c.fillRect(58, 18, 60, 20, accent)

	case StateSleepy:
		// 半閉眼 + ZZZ
		clearUncleEyes(c)
		// 重畫眉毛
		c.fillRect(13, 19, 17, 21, uclBrow)
		c.fillRect(18, 19, 23, 20, uclBrow)
		c.fillRect(24, 19, 27, 19, uclBrow)
		c.fillRect(46, 19, 50, 21, uclBrow)
		c.fillRect(40, 19, 45, 20, uclBrow)
		c.fillRect(36, 19, 39, 19, uclBrow)
		// 半閉眼線
		c.hline(24, 13, 26, uclOutline)
		c.hline(25, 14, 25, uclOutline)
		c.hline(24, 37, 50, uclOutline)
		c.hline(25, 38, 49, uclOutline)
		// 嘴巴微張
		c.fillRect(28, 39, 35, 42, uclMouth)
		c.fillRect(29, 40, 34, 41, uclMouth)
		// ZZZ
		c.fillRect(50, 10, 54, 12, accent)
		c.fillRect(52, 7, 56, 9, accent)
		c.fillRect(54, 4, 58, 6, accent)

	case StateSad:
		// 嘴角下垂 + 淚滴
		c.hline(40, 26, 37, uclLip)
		c.set(24, 39, uclMouth)
		c.set(25, 39, uclMouth)
		c.set(38, 39, uclMouth)
		c.set(39, 39, uclMouth)
		c.hline(41, 27, 36, uclLipD)
		// 淚滴
		c.fillRect(24, 28, 25, 32, accent)
		c.fillRect(50, 28, 51, 32, accent)

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

// RenderUnclePixelAvatar 渲染指定狀態的大叔 pixel art 頭像。
// 預設產出 128x128；仍接受 256x256 供高解析預覽使用。
func RenderUnclePixelAvatar(state AvatarStateTrigger, size int) ([]byte, error) {
	c := newPixelCanvas(size)
	drawUncleBase(c)
	applyUncleState(c, state)

	var buf []byte
	w := &byteWriter{buf: &buf}
	if err := png.Encode(w, c.img); err != nil {
		return nil, fmt.Errorf("PNG 編碼失敗: %w", err)
	}
	return buf, nil
}

// GenerateAllUnclePixelAvatars 產生所有狀態的 128x128 PNG。
func GenerateAllUnclePixelAvatars(outputDir string) error {
	sizes := []int{defaultPixelAvatarSize}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("建立輸出目錄失敗: %w", err)
	}
	for _, state := range AllStateTriggers {
		for _, size := range sizes {
			data, err := RenderUnclePixelAvatar(state, size)
			if err != nil {
				return fmt.Errorf("渲染 %s_%d 失敗: %w", state, size, err)
			}
			filename := fmt.Sprintf("uncle_%s_%d.png", state, size)
			path := filepath.Join(outputDir, filename)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				return fmt.Errorf("寫入 %s 失敗: %w", filename, err)
			}
		}
	}
	return nil
}

// GetUnclePixelAvatarPath 取得指定狀態和尺寸的大叔 pixel avatar 檔案路徑。
func GetUnclePixelAvatarPath(outputDir string, state AvatarStateTrigger, size int) string {
	return filepath.Join(outputDir, fmt.Sprintf("uncle_%s_%d.png", state, size))
}
