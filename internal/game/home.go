package game

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/sfx"
)

var allPieces = []chess.PieceType{chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King}

// pressHold is how long a button stays visibly squashed before the tap is
// acted on. Toddlers need to see that the tap landed.
const pressHold = 0.14

type HomeScene struct {
	game *Game

	pressed  int // -1 none, 0 play, 1..6 piece card
	pressT   float64
	pendPick chess.PieceType
	pendPlay bool
}

func NewHomeScene(g *Game) *HomeScene { return &HomeScene{game: g, pressed: -1} }

type homeRects struct {
	title, play, label, tray layout.Rect
	cards                    [6]layout.Rect
}

func homeLayout(m layout.Metrics) homeRects {
	s := m.Safe
	gap := s.H * 0.018
	titleH := s.H * 0.12
	playH := math.Max(s.H*0.12, m.MinTap*1.25)
	labelH := s.H * 0.055
	trayH := math.Max(s.H*0.11, m.MinTap*0.9)
	cardsH := s.H - titleH - playH - labelH - trayH - 4*gap
	if cardsH < m.MinTap*2 {
		cardsH = m.MinTap * 2
	}

	var r homeRects
	y := s.Y
	r.title = layout.Rect{X: s.X, Y: y, W: s.W, H: titleH}
	y += titleH + gap

	pw := s.W * 0.82
	r.play = layout.Rect{X: s.X + (s.W-pw)/2, Y: y, W: pw, H: playH}
	y += playH + gap

	r.label = layout.Rect{X: s.X, Y: y, W: s.W, H: labelH}
	y += labelH

	colGap := s.W * 0.045
	cw := (s.W - 2*colGap) / 3
	// Keep the cards close to square; very tall cards look empty and make the
	// piece art tiny relative to the button. Any height left over widens the
	// gap between the rows rather than leaving a hole under the grid.
	ch := math.Min((cardsH-cardsH*0.08)/2, cw*1.25)
	rowGap := math.Min(cardsH-2*ch, cardsH*0.22)
	gridH := ch*2 + rowGap
	gy := y + (cardsH-gridH)/2
	for i := 0; i < 6; i++ {
		col, row := i%3, i/3
		r.cards[i] = layout.Rect{
			X: s.X + float64(col)*(cw+colGap),
			Y: gy + float64(row)*(ch+rowGap),
			W: cw, H: ch,
		}
	}
	y += cardsH + gap

	r.tray = layout.Rect{X: s.X, Y: y, W: s.W, H: trayH}
	return r
}

func (h *HomeScene) Update(ctx *Context) error {
	r := homeLayout(ctx.M)

	if h.pressT > 0 {
		h.pressT -= ctx.DT
		if h.pressT <= 0 {
			h.pressed = -1
			if h.pendPlay {
				h.pendPlay = false
				ctx.Switch(NewPlayScene(h.game, h.pendPick))
			}
		}
		return nil
	}

	for _, ev := range ctx.Pointer.Pressed() {
		if r.play.Contains(ev.X, ev.Y) {
			h.arm(ctx, 0, chess.Rook)
			return nil
		}
		for i, cr := range r.cards {
			if cr.Contains(ev.X, ev.Y) {
				h.arm(ctx, i+1, allPieces[i])
				return nil
			}
		}
	}
	return nil
}

func (h *HomeScene) arm(ctx *Context, slot int, pt chess.PieceType) {
	ctx.SFX.Play(sfx.SndButton)
	h.pressed = slot
	h.pressT = pressHold
	h.pendPick = pt
	h.pendPlay = true
}

func (h *HomeScene) Draw(dst *ebiten.Image, ctx *Context) {
	m := ctx.M
	r := homeLayout(m)
	render.DrawBackground(dst, m)

	// Title with a star on each side.
	tcx, tcy := r.title.Center()
	titleSize := render.FitTextSize("Chess Stars", m.TitleSize*1.35, r.title.W*0.66)
	render.DrawTextShadowed(dst, "Chess Stars", tcx, tcy, titleSize, render.ColorText)
	starSize := r.title.H * 0.42
	render.DrawEmoji(dst, render.StarEmoji, tcx-r.title.W*0.40, tcy, starSize, 0, 1)
	render.DrawEmoji(dst, render.StarEmoji, tcx+r.title.W*0.40, tcy, starSize, 0, 1)

	// The Play button breathes just enough to read as "press me", no more.
	pulse := 1 + math.Sin(ctx.T*2.2)*0.012
	pw, ph := r.play.W*pulse, r.play.H*pulse
	px := r.play.X - (pw-r.play.W)/2
	py := r.play.Y - (ph-r.play.H)/2
	render.DrawGlow(dst, r.play.X+r.play.W/2, r.play.Y+r.play.H/2, r.play.W*0.55, render.Alpha(render.ColorPlayHi, 0.30))
	render.DrawChunkyButton(dst, px, py, pw, ph, render.ColorPlay, render.ColorPlayEdge, h.pressed == 0)
	playText := render.FitTextSize("PLAY", ph*0.44, pw*0.44)
	render.DrawTextShadowed(dst, "PLAY", px+pw/2, py+ph*0.52, playText, render.ColorText)
	render.DrawEmoji(dst, "1f680", px+pw*0.155, py+ph*0.52, ph*0.46, -0.5, 1)
	render.DrawEmoji(dst, "1f31f", px+pw*0.845, py+ph*0.52, ph*0.46, 0, 1)

	lcx, lcy := r.label.Center()
	render.DrawTextShadowed(dst, "Pick a piece", lcx, lcy, m.BodySize*1.05, render.ColorTextDim)

	for i, cr := range r.cards {
		h.drawCard(dst, ctx, i, cr)
	}

	h.drawTray(dst, ctx, r.tray)
}

func (h *HomeScene) drawCard(dst *ebiten.Image, ctx *Context, i int, cr layout.Rect) {
	m := ctx.M
	pressed := h.pressed == i+1
	face := render.PieceCardColors[i%len(render.PieceCardColors)]
	edge := render.PieceCardEdges[i%len(render.PieceCardEdges)]
	render.DrawChunkyButton(dst, cr.X, cr.Y, cr.W, cr.H, face, edge, pressed)

	off := 0.0
	if pressed {
		off = cr.H * 0.05
	}
	pad := cr.W * 0.13
	pw := cr.W - 2*pad
	pr := layout.Rect{X: cr.X + pad, Y: cr.Y + cr.H*0.08 + off, W: pw, H: cr.H * 0.60}
	render.DrawPiece(dst, chess.Piece{Type: allPieces[i], Color: chess.White}, pr, 0, false)

	name := render.PieceName(allPieces[i])
	size := render.FitTextSize(name, m.BodySize*0.9, cr.W*0.86)
	render.DrawTextShadowed(dst, name, cr.X+cr.W/2, cr.Y+cr.H*0.82+off, size, render.ColorText)
}

func (h *HomeScene) drawTray(dst *ebiten.Image, ctx *Context, tr layout.Rect) {
	m := ctx.M
	render.FillRoundRect(dst, tr.X, tr.Y, tr.W, tr.H, tr.H*0.35, render.ColorTray)

	stickers := h.game.Stickers()
	if len(stickers) == 0 {
		cx, cy := tr.Center()
		render.DrawTextShadowed(dst, "Find stars to win stickers!", cx, cy, render.FitTextSize("Find stars to win stickers!", m.BodySize*0.8, tr.W*0.9), render.ColorTextDim)
		return
	}

	countText := fmt.Sprintf("%d", len(stickers))
	size := tr.H * 0.62
	countW := tr.H * 0.9
	render.DrawEmoji(dst, "1f3c6", tr.X+tr.H*0.42, tr.Y+tr.H/2, size, 0, 1)
	render.DrawTextShadowed(dst, countText, tr.X+tr.H*0.42+countW*0.55, tr.Y+tr.H/2, m.BodySize, render.ColorText)

	// Most recent stickers, newest on the right.
	startX := tr.X + tr.H*1.15
	avail := tr.X + tr.W - startX - tr.H*0.2
	step := math.Min(size*0.95, avail/8)
	show := stickers
	if len(show) > 8 {
		show = show[len(show)-8:]
	}
	for i, e := range show {
		x := startX + step*float64(i) + step/2
		render.DrawEmoji(dst, render.EmojiName(e), x, tr.Y+tr.H/2, step*0.92, 0, 1)
	}
}
