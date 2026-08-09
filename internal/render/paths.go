package render

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

func StarPath() *vector.Path {
	var p vector.Path
	outer, inner := 0.5, 0.22
	for i := 0; i < 10; i++ {
		angle := float64(i)*math.Pi/5 - math.Pi/2
		r := outer
		if i%2 == 1 {
			r = inner
		}
		x := 0.5 + float32(r*math.Cos(angle))
		y := 0.5 + float32(r*math.Sin(angle))
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return &p
}
