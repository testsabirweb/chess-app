package render

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/testsabirweb/chess-app/internal/chess"
)

var unitPaths map[chess.PieceType]*vector.Path

func init() {
	unitPaths = map[chess.PieceType]*vector.Path{
		chess.Pawn:   pawnShape(),
		chess.Rook:   rookShape(),
		chess.Knight: knightShape(),
		chess.Bishop: bishopShape(),
		chess.Queen:  queenShape(),
		chess.King:   kingShape(),
	}
}

func UnitPath(t chess.PieceType) *vector.Path {
	return unitPaths[t]
}

func addCircle(p *vector.Path, cx, cy, r float32) {
	const segments = 24
	for i := 0; i <= segments; i++ {
		angle := float32(i) / segments * 2 * math.Pi
		x := cx + r*float32(math.Cos(float64(angle)))
		y := cy + r*float32(math.Sin(float64(angle)))
		if i == 0 {
			p.MoveTo(x, y)
		} else {
			p.LineTo(x, y)
		}
	}
	p.Close()
}

func pawnShape() *vector.Path {
	var p vector.Path
	addCircle(&p, 0.5, 0.28, 0.12)
	p.MoveTo(0.35, 0.42)
	p.LineTo(0.65, 0.42)
	p.LineTo(0.78, 0.92)
	p.LineTo(0.22, 0.92)
	p.Close()
	return &p
}

func rookShape() *vector.Path {
	var p vector.Path
	p.MoveTo(0.22, 0.88)
	p.LineTo(0.78, 0.88)
	p.LineTo(0.72, 0.55)
	p.LineTo(0.28, 0.55)
	p.Close()
	p.MoveTo(0.25, 0.55)
	p.LineTo(0.75, 0.55)
	p.LineTo(0.68, 0.25)
	p.LineTo(0.32, 0.25)
	p.Close()
	for _, x := range []float32{0.32, 0.5, 0.68} {
		p.MoveTo(x, 0.25)
		p.LineTo(x, 0.1)
		p.LineTo(x+0.08, 0.1)
		p.LineTo(x+0.08, 0.25)
		p.Close()
	}
	return &p
}

func knightShape() *vector.Path {
	var p vector.Path
	p.MoveTo(0.3, 0.9)
	p.CubicTo(0.2, 0.7, 0.15, 0.5, 0.35, 0.35)
	p.CubicTo(0.55, 0.2, 0.75, 0.15, 0.8, 0.35)
	p.CubicTo(0.85, 0.5, 0.7, 0.55, 0.55, 0.5)
	p.CubicTo(0.45, 0.65, 0.4, 0.8, 0.3, 0.9)
	p.Close()
	return &p
}

func bishopShape() *vector.Path {
	var p vector.Path
	addCircle(&p, 0.5, 0.08, 0.06)
	p.MoveTo(0.38, 0.35)
	p.CubicTo(0.3, 0.55, 0.3, 0.75, 0.5, 0.92)
	p.CubicTo(0.7, 0.75, 0.7, 0.55, 0.62, 0.35)
	p.Close()
	p.MoveTo(0.48, 0.2)
	p.LineTo(0.52, 0.45)
	return &p
}

func queenShape() *vector.Path {
	var p vector.Path
	p.MoveTo(0.25, 0.9)
	p.LineTo(0.75, 0.9)
	p.LineTo(0.65, 0.5)
	p.LineTo(0.35, 0.5)
	p.Close()
	pts := [][2]float32{{0.2, 0.5}, {0.35, 0.2}, {0.5, 0.45}, {0.65, 0.2}, {0.8, 0.5}}
	for _, pt := range pts {
		addCircle(&p, pt[0], pt[1], 0.07)
	}
	return &p
}

func kingShape() *vector.Path {
	var p vector.Path
	p.MoveTo(0.28, 0.9)
	p.LineTo(0.72, 0.9)
	p.LineTo(0.62, 0.45)
	p.LineTo(0.38, 0.45)
	p.Close()
	p.MoveTo(0.5, 0.12)
	p.LineTo(0.5, 0.42)
	p.MoveTo(0.38, 0.22)
	p.LineTo(0.62, 0.22)
	return &p
}

func PathBoundsOK(t chess.PieceType) bool {
	return unitPaths[t] != nil
}

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
