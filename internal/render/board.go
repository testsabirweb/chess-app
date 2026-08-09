package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/testsabirweb/chess-app/internal/anim"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
)

type Sprites struct {
	confetti map[int]*ebiten.Image
	lastCell float64
}

func NewSprites() *Sprites { return &Sprites{confetti: map[int]*ebiten.Image{}} }

func (s *Sprites) ConfettiParticle(cell float64, shape uint8, clr color.RGBA) *ebiten.Image {
	px := int(cell*0.15 + 0.5)
	if px < 4 {
		px = 4
	}
	key := px*10 + int(shape)
	if img, ok := s.confetti[key]; ok && s.lastCell == cell {
		return img
	}
	img := ebiten.NewImage(px, px)
	half := float32(px) / 2
	switch shape {
	case anim.ShapeCircle:
		vectorFillCircle(img, half, half, half-0.5, clr)
	default:
		vectorFillRect(img, 0, 0, float32(px), float32(px), clr)
	}
	s.confetti[key] = img
	s.lastCell = cell
	return img
}

func vectorFillCircle(dst *ebiten.Image, cx, cy, r float32, clr color.RGBA) {
	// use render helper
	DrawFilledCircle(dst, float64(cx), float64(cy), float64(r), clr)
}

func vectorFillRect(dst *ebiten.Image, x, y, w, h float32, clr color.RGBA) {
	DrawFilledRect(dst, float64(x), float64(y), float64(w), float64(h), clr)
}

func DrawFilledCircle(dst *ebiten.Image, cx, cy, r float64, clr color.Color) {
	vector.FillCircle(dst, float32(cx), float32(cy), float32(r), clr, true)
}

func DrawBoard(dst *ebiten.Image, m layout.Metrics) {
	for r := 0; r < m.Rows; r++ {
		for f := 0; f < m.Cols; f++ {
			cr := m.CellRect(f, r)
			clr := ColorBoardL
			if (f+r)%2 == 1 {
				clr = ColorBoardD
			}
			DrawFilledRect(dst, cr.X, cr.Y, cr.W, cr.H, clr)
		}
	}
}


func DrawStar(dst *ebiten.Image, sq chess.Square, m layout.Metrics, pulse float64) {
	cr := m.CellRect(int(sq.File), int(sq.Rank))
	pad := cr.W * (0.18 - 0.04*(pulse-0.85))
	DrawPath(dst, StarPath(), cr.X+pad, cr.Y+pad, cr.W-2*pad, cr.H-2*pad, ColorStar, ColorOutline, cr.W*0.04)
}

func DrawConfetti(dst *ebiten.Image, c *anim.Confetti, sprites *Sprites, cell float64) {
	c.ForEach(func(p anim.Particle) {
		img := sprites.ConfettiParticle(cell, p.Shape, p.Color)
		alpha := anim.ParticleAlpha(p)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(img.Bounds().Dx())/2, -float64(img.Bounds().Dy())/2)
		op.GeoM.Rotate(p.Rot)
		op.GeoM.Translate(p.X, p.Y)
		op.ColorScale.ScaleAlpha(float32(alpha))
		dst.DrawImage(img, op)
	})
}

func PieceName(t chess.PieceType) string {
	switch t {
	case chess.Pawn:
		return "Pawn"
	case chess.Knight:
		return "Knight"
	case chess.Bishop:
		return "Bishop"
	case chess.Rook:
		return "Rook"
	case chess.Queen:
		return "Queen"
	case chess.King:
		return "King"
	default:
		return ""
	}
}

func PieceFill(c chess.Color) color.RGBA {
	if c == chess.White {
		return PieceFills["white"]
	}
	return PieceFills["black"]
}

func WobbleOffset(phase float64, amp float64) (float64, float64) {
	return math.Sin(phase*math.Pi*4) * amp, 0
}
