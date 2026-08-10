package render

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// --- cached soft sprites -----------------------------------------------------
//
// Circles, glows and shadows are the most-drawn shapes in the game, and
// vector.FillPath costs a stencil pass each. These are generated on the CPU
// once, then drawn as ordinary tinted images, which is far cheaper on mobile
// GPUs and looks softer besides.

var (
	spriteOnce   sync.Once
	discSprite   *ebiten.Image
	glowSprite   *ebiten.Image
	shadowSprite *ebiten.Image
)

const softSpriteSize = 128

func initSoftSprites() {
	spriteOnce.Do(func() {
		discSprite = radialSprite(func(d float64) float64 {
			// hard disc with a 2px antialiased rim
			edge := 1.0 - d
			return clamp01(edge * softSpriteSize / 2)
		})
		glowSprite = radialSprite(func(d float64) float64 {
			g := 1 - d
			return clamp01(g * g * g)
		})
		shadowSprite = radialSprite(func(d float64) float64 {
			g := 1 - d
			return clamp01(g*g*1.15) * 0.9
		})
	})
}

func radialSprite(f func(dist float64) float64) *ebiten.Image {
	const n = softSpriteSize
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	c := float64(n-1) / 2
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := (float64(x)-c)/c, (float64(y)-c)/c
			d := math.Hypot(dx, dy)
			a := 0.0
			if d <= 1 {
				a = f(d)
			}
			v := uint8(a * 255)
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v, v, v, v
		}
	}
	return ebiten.NewImageFromImage(img)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func drawSoft(dst *ebiten.Image, sprite *ebiten.Image, cx, cy, r float64, clr color.RGBA) {
	if r <= 0 {
		return
	}
	s := (r * 2) / softSpriteSize
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-softSpriteSize/2, -softSpriteSize/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(sprite, op)
}

// FillCircleSoft draws a filled, antialiased circle from the cached sprite.
func FillCircleSoft(dst *ebiten.Image, cx, cy, r float64, clr color.RGBA) {
	initSoftSprites()
	drawSoft(dst, discSprite, cx, cy, r, clr)
}

// DrawGlow paints a soft radial halo, used behind the star and the play button.
func DrawGlow(dst *ebiten.Image, cx, cy, r float64, clr color.RGBA) {
	initSoftSprites()
	drawSoft(dst, glowSprite, cx, cy, r, clr)
}

// DrawSoftShadow paints a blurred ellipse under pieces and cards.
func DrawSoftShadow(dst *ebiten.Image, cx, cy, rx, ry float64, clr color.RGBA) {
	initSoftSprites()
	if rx <= 0 || ry <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-softSpriteSize/2, -softSpriteSize/2)
	op.GeoM.Scale(rx*2/softSpriteSize, ry*2/softSpriteSize)
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(shadowSprite, op)
}

// --- gradient ---------------------------------------------------------------

type gradKey struct{ top, bottom color.RGBA }

var (
	gradCache = map[gradKey]*ebiten.Image{}
	gradMu    sync.Mutex
)

func gradientImage(top, bottom color.RGBA) *ebiten.Image {
	key := gradKey{top, bottom}
	gradMu.Lock()
	defer gradMu.Unlock()
	if img, ok := gradCache[key]; ok {
		return img
	}
	const n = 256
	src := image.NewRGBA(image.Rect(0, 0, 1, n))
	for y := 0; y < n; y++ {
		t := float64(y) / float64(n-1)
		src.Set(0, y, color.RGBA{
			uint8(float64(top.R) + (float64(bottom.R)-float64(top.R))*t),
			uint8(float64(top.G) + (float64(bottom.G)-float64(top.G))*t),
			uint8(float64(top.B) + (float64(bottom.B)-float64(top.B))*t),
			255,
		})
	}
	img := ebiten.NewImageFromImage(src)
	gradCache[key] = img
	return img
}

// DrawVerticalGradient fills the rect with a smooth top-to-bottom blend.
func DrawVerticalGradient(dst *ebiten.Image, x, y, w, h float64, top, bottom color.RGBA) {
	img := gradientImage(top, bottom)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h/256)
	op.GeoM.Translate(x, y)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(img, op)
}

// --- rounded rectangles ------------------------------------------------------

func roundRectPath(p *vector.Path, x, y, w, h, r float64) {
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	fx, fy, fw, fh, fr := float32(x), float32(y), float32(w), float32(h), float32(r)
	p.MoveTo(fx+fr, fy)
	p.LineTo(fx+fw-fr, fy)
	p.Arc(fx+fw-fr, fy+fr, fr, -math.Pi/2, 0, vector.Clockwise)
	p.LineTo(fx+fw, fy+fh-fr)
	p.Arc(fx+fw-fr, fy+fh-fr, fr, 0, math.Pi/2, vector.Clockwise)
	p.LineTo(fx+fr, fy+fh)
	p.Arc(fx+fr, fy+fh-fr, fr, math.Pi/2, math.Pi, vector.Clockwise)
	p.LineTo(fx, fy+fr)
	p.Arc(fx+fr, fy+fr, fr, math.Pi, 3*math.Pi/2, vector.Clockwise)
	p.Close()
}

// FillRoundRect draws a filled rounded rectangle.
func FillRoundRect(dst *ebiten.Image, x, y, w, h, r float64, clr color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	var p vector.Path
	roundRectPath(&p, x, y, w, h, r)
	var dop vector.DrawPathOptions
	dop.AntiAlias = true
	dop.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &p, &vector.FillOptions{FillRule: vector.FillRuleNonZero}, &dop)
}

// DrawChunkyButton is the one button look used everywhere: a soft drop shadow,
// a darker rim underneath for depth, the face on top, and a glossy highlight.
func DrawChunkyButton(dst *ebiten.Image, x, y, w, h float64, face, edge color.RGBA, pressed bool) {
	r := math.Min(h*0.42, w*0.42)
	lift := h * 0.09
	if pressed {
		y += lift * 0.6
		lift *= 0.4
	}
	DrawSoftShadow(dst, x+w/2, y+h+lift*0.6, w*0.52, h*0.34, ColorShadow)
	FillRoundRect(dst, x, y+lift, w, h, r, edge)
	FillRoundRect(dst, x, y, w, h, r, face)
	// glossy top highlight
	FillRoundRect(dst, x+w*0.05, y+h*0.07, w*0.90, h*0.24, r*0.9, ColorGloss)
}

// DrawFilledRect is the plain axis-aligned rect used for board cells.
func DrawFilledRect(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), clr, false)
}
