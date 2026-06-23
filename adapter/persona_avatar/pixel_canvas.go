package persona_avatar

import (
	"image"
	"image/color"
)

const (
	pixelAvatarLogicalSize = 64
	defaultPixelAvatarSize = 128
)

// pixelCanvas lets the avatar drawings keep their original 64x64 logical
// coordinates while writing directly to the requested PNG size.
type pixelCanvas struct {
	img   *image.RGBA
	size  int
	scale int
}

// canvas64 is kept as a package-local compatibility alias for the existing
// drawing functions. It now renders directly to 128/256 instead of storing a
// separate 64x64 source image.
type canvas64 = pixelCanvas

func normalizePixelAvatarSize(size int) int {
	switch size {
	case 128, 256:
		return size
	default:
		return defaultPixelAvatarSize
	}
}

func newPixelCanvas(size int) *pixelCanvas {
	size = normalizePixelAvatarSize(size)
	return &pixelCanvas{
		img:   image.NewRGBA(image.Rect(0, 0, size, size)),
		size:  size,
		scale: size / pixelAvatarLogicalSize,
	}
}

func (c *pixelCanvas) set(x, y int, col color.RGBA) {
	if c == nil || x < 0 || y < 0 || x >= pixelAvatarLogicalSize || y >= pixelAvatarLogicalSize {
		return
	}
	px := x * c.scale
	py := y * c.scale
	for yy := py; yy < py+c.scale; yy++ {
		for xx := px; xx < px+c.scale; xx++ {
			c.img.SetRGBA(xx, yy, col)
		}
	}
}

func (c *pixelCanvas) get(x, y int) color.RGBA {
	if c == nil || x < 0 || y < 0 || x >= pixelAvatarLogicalSize || y >= pixelAvatarLogicalSize {
		return color.RGBA{}
	}
	return c.img.RGBAAt(x*c.scale, y*c.scale)
}

func (c *pixelCanvas) fillRect(x0, y0, x1, y1 int, col color.RGBA) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			c.set(x, y, col)
		}
	}
}

func (c *pixelCanvas) hline(y, x0, x1 int, col color.RGBA) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	for x := x0; x <= x1; x++ {
		c.set(x, y, col)
	}
}

func colEq(a, b color.RGBA) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A
}
