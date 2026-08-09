package render

import (
	"bytes"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	boldSrc     *text.GoTextFaceSource
	faceCache   = map[int]*text.GoTextFace{}
	faceCacheMu sync.Mutex
)

func init() {
	var err error
	boldSrc, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
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

func DrawTextCentered(dst *ebiten.Image, s string, cx, cy float64, size float64, clr color.Color) {
	face := Face(size)
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleWithColor(clr)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	text.Draw(dst, s, face, op)
}

func DrawFilledRect(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), clr, false)
}

func DrawRoundedButton(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), clr, false)
}

func DrawPath(dst *ebiten.Image, path *vector.Path, x, y, w, h float64, fill, stroke color.Color, strokeW float64) {
	var p vector.Path
	var add vector.AddPathOptions
	add.GeoM.Scale(w, h)
	add.GeoM.Translate(x, y)
	p.AddPath(path, &add)

	var dop vector.DrawPathOptions
	dop.AntiAlias = true
	dop.ColorScale.ScaleWithColor(fill)
	vector.FillPath(dst, &p, &vector.FillOptions{FillRule: vector.FillRuleNonZero}, &dop)

	var sop vector.StrokeOptions
	sop.Width = float32(strokeW)
	sop.LineJoin = vector.LineJoinRound
	sop.LineCap = vector.LineCapRound
	var strokeOp vector.DrawPathOptions
	strokeOp.AntiAlias = true
	strokeOp.ColorScale.ScaleWithColor(stroke)
	vector.StrokePath(dst, &p, &sop, &strokeOp)
}
