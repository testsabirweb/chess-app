package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/anim"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
)

// --- background --------------------------------------------------------------

// DrawBackground paints the still gradient the whole game sits on. It is
// deliberately motionless: the only things that move on screen are the star and
// the piece, so that is where a small child looks.
func DrawBackground(dst *ebiten.Image, m layout.Metrics) {
	DrawVerticalGradient(dst, 0, 0, m.W, m.H, ColorBGTop, ColorBGBottom)
}

// --- board -------------------------------------------------------------------

// DrawBoard paints the frame and the checkerboard.
func DrawBoard(dst *ebiten.Image, m layout.Metrics) {
	pad := m.Cell * 0.14
	r := m.Cell * 0.28
	fx, fy := m.Board.X-pad, m.Board.Y-pad
	fw, fh := m.Board.W+2*pad, m.Board.H+2*pad

	DrawSoftShadow(dst, m.Board.X+m.Board.W/2, m.Board.Y+m.Board.H+pad, m.Board.W*0.52, m.Cell*0.42, ColorShadow)
	FillRoundRect(dst, fx, fy+pad*0.35, fw, fh, r, ColorFrameInner)
	FillRoundRect(dst, fx, fy, fw, fh, r, ColorFrame)

	for row := 0; row < m.Rows; row++ {
		for f := 0; f < m.Cols; f++ {
			cr := m.CellRect(f, row)
			clr := ColorBoardL
			if (f+row)%2 == 1 {
				clr = ColorBoardD
			}
			DrawFilledRect(dst, cr.X, cr.Y, cr.W, cr.H, clr)
		}
	}
}

// DrawSquareTint washes a single square in a colour (selection, wobble, hints).
func DrawSquareTint(dst *ebiten.Image, m layout.Metrics, sq chess.Square, dx, dy float64, clr color.RGBA) {
	cr := m.CellRect(int(sq.File), int(sq.Rank))
	DrawFilledRect(dst, cr.X+dx, cr.Y+dy, cr.W, cr.H, clr)
}

// DrawMoveHints marks every square the piece can move to right now. The dots
// are still, not pulsing - they are information, not decoration.
func DrawMoveHints(dst *ebiten.Image, m layout.Metrics, squares []chess.Square) {
	for _, sq := range squares {
		cr := m.CellRect(int(sq.File), int(sq.Rank))
		cx, cy := cr.Center()
		DrawFilledRect(dst, cr.X, cr.Y, cr.W, cr.H, ColorHint)
		r := cr.W * 0.16
		FillCircleSoft(dst, cx, cy, r*1.5, Alpha(ColorHintRing, 0.30))
		FillCircleSoft(dst, cx, cy, r, ColorHintDot)
	}
}

// DrawPickableRing is the gentle "tap me" halo under the piece.
func DrawPickableRing(dst *ebiten.Image, m layout.Metrics, sq chess.Square, pulse float64, picked bool) {
	cr := m.CellRect(int(sq.File), int(sq.Rank))
	cx, cy := cr.Center()
	clr := ColorPickable
	if picked {
		clr = ColorPicked
	}
	DrawGlow(dst, cx, cy, cr.W*0.62*pulse, clr)
}

// DrawStar paints the target: a warm halo, then the star sticker itself,
// breathing gently.
func DrawStar(dst *ebiten.Image, sq chess.Square, m layout.Metrics, pulse, t float64) {
	cr := m.CellRect(int(sq.File), int(sq.Rank))
	cx, cy := cr.Center()
	DrawStarAt(dst, cx, cy, cr.W, pulse, t)
}

// DrawStarAt is DrawStar at an arbitrary point, for the pop animation.
func DrawStarAt(dst *ebiten.Image, cx, cy, cell, pulse, t float64) {
	DrawGlow(dst, cx, cy, cell*1.05*pulse, Alpha(ColorStarGlow, 0.45))
	DrawGlow(dst, cx, cy, cell*0.62*pulse, Alpha(ColorStarGlow, 0.55))
	DrawEmoji(dst, StarEmoji, cx, cy, cell*0.82*pulse, 0, 1)
}

// --- confetti ----------------------------------------------------------------

// Sprites caches the tiny white shapes confetti is drawn from. They are tinted
// at draw time, so one image serves every colour.
type Sprites struct {
	shapes map[int]*ebiten.Image
}

func NewSprites() *Sprites { return &Sprites{shapes: map[int]*ebiten.Image{}} }

func (s *Sprites) shape(kind uint8) *ebiten.Image {
	key := int(kind)
	if img, ok := s.shapes[key]; ok {
		return img
	}
	const px = 32
	img := ebiten.NewImage(px, px)
	switch kind {
	case anim.ShapeCircle:
		FillCircleSoft(img, px/2, px/2, px/2, color.RGBA{255, 255, 255, 255})
	case anim.ShapeTri:
		DrawFilledRect(img, 0, px*0.25, px, px*0.5, color.RGBA{255, 255, 255, 255})
	default:
		DrawFilledRect(img, 0, 0, px, px, color.RGBA{255, 255, 255, 255})
	}
	s.shapes[key] = img
	return img
}

func DrawConfetti(dst *ebiten.Image, c *anim.Confetti, sprites *Sprites, cell float64) {
	c.ForEach(func(p anim.Particle) {
		img := sprites.shape(p.Shape)
		b := img.Bounds()
		size := p.Size
		if size <= 0 {
			size = cell * 0.12
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
		op.GeoM.Scale(size/float64(b.Dx()), size/float64(b.Dy()))
		op.GeoM.Rotate(p.Rot)
		op.GeoM.Translate(p.X, p.Y)
		op.ColorScale.ScaleWithColor(p.Color)
		op.ColorScale.ScaleAlpha(float32(anim.ParticleAlpha(p)))
		op.Filter = ebiten.FilterLinear
		dst.DrawImage(img, op)
	})
}

// --- misc --------------------------------------------------------------------

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
