package render

import (
	"bytes"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
)

var (
	boldSrc     *text.GoTextFaceSource
	faceCache   = map[int]*text.GoTextFace{}
	faceCacheMu sync.Mutex
)

func init() {
	var err error
	boldSrc, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		panic(err)
	}
}

func Face(size float64) *text.GoTextFace {
	px := int(size + 0.5)
	if px < 8 {
		px = 8
	}
	faceCacheMu.Lock()
	defer faceCacheMu.Unlock()
	if f, ok := faceCache[px]; ok {
		return f
	}
	f := &text.GoTextFace{Source: boldSrc, Size: float64(px)}
	faceCache[px] = f
	return f
}

// MeasureText returns the drawn size of s at the given face size.
func MeasureText(s string, size float64) (float64, float64) {
	return text.Measure(s, Face(size), 0)
}

// FitTextSize shrinks size until s fits within maxW.
func FitTextSize(s string, size, maxW float64) float64 {
	for i := 0; i < 8 && size > 8; i++ {
		w, _ := MeasureText(s, size)
		if w <= maxW {
			break
		}
		size *= 0.88
	}
	return size
}

// DrawTextCentered draws s centred on (cx, cy).
func DrawTextCentered(dst *ebiten.Image, s string, cx, cy float64, size float64, clr color.Color) {
	face := Face(size)
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleWithColor(clr)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	text.Draw(dst, s, face, op)
}

// DrawTextShadowed is the standard label look: a soft dark offset copy behind
// the text so it stays readable over any background. The offset is small and
// capped, otherwise large headings read as a doubled image rather than a shadow.
func DrawTextShadowed(dst *ebiten.Image, s string, cx, cy, size float64, clr color.Color) {
	off := math.Min(size*0.05, 3)
	DrawTextCentered(dst, s, cx+off*0.6, cy+off, size, ColorTextShadow)
	DrawTextCentered(dst, s, cx, cy, size, clr)
}
