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

func homeDp(m layout.Metrics, v float64) float64 {
	return v * m.Scale
}

func scaleHomeBands(titleH, playH, labelH, trayH, gap *float64, cardsH *float64, safeH float64) {
	gaps := 4 * *gap
	total := *titleH + *playH + *labelH + *trayH + gaps
	if *cardsH > 0 {
		total += *cardsH
	}
	if total <= safeH+1e-9 {
		return
	}
	// Scale every band and the gap by the same factor so nothing runs off-screen.
	scale := safeH / total
	*titleH *= scale
	*playH *= scale
	*labelH *= scale
	*trayH *= scale
	*gap *= scale
	if *cardsH > 0 {
		*cardsH *= scale
	}
}

func homeLayout(m layout.Metrics) homeRects {
	if !m.Portrait {
		return homeLayoutLandscape(m)
	}
	return homeLayoutPortrait(m)
}

func homeLayoutPortrait(m layout.Metrics) homeRects {
	s := m.Safe
	gap := s.H * 0.018

	titleH := math.Min(s.H*0.12, homeDp(m, 140))
	playH := math.Min(math.Max(s.H*0.12, m.MinTap*1.25), homeDp(m, 120))
	labelH := math.Min(s.H*0.055, homeDp(m, 50))
	trayH := math.Min(math.Max(s.H*0.11, m.MinTap*0.9), homeDp(m, 130))

	gaps := 4 * gap
	cardsH := s.H - titleH - playH - labelH - trayH - gaps
	scaleHomeBands(&titleH, &playH, &labelH, &trayH, &gap, &cardsH, s.H)

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
	rowGap := cardsH * 0.10
	ch := math.Min((cardsH-rowGap)/2, cw*1.32)
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

func homeLayoutLandscape(m layout.Metrics) homeRects {
	s := m.Safe
	gap := homeDp(m, 8)

	r := homeRects{}
	r.title = m.Board

	panelX := m.Board.X + m.Board.W + gap
	panelW := s.X + s.W - panelX
	panelY := s.Y
	panelH := s.H

	vGap := panelH * 0.02
	playH := math.Min(math.Max(panelH*0.18, m.MinTap*1.25), homeDp(m, 100))
	labelH := math.Min(panelH*0.08, homeDp(m, 40))
	trayH := math.Min(math.Max(panelH*0.15, m.MinTap*0.9), homeDp(m, 100))

	content := playH + labelH + trayH + 2*vGap
	cardsH := panelH - content
	if content > panelH {
		scale := panelH / content
		playH *= scale
		labelH *= scale
		trayH *= scale
		vGap *= scale
		cardsH = panelH - playH - labelH - trayH - 2*vGap
	}

	y := panelY
	pw := panelW * 0.82
	r.play = layout.Rect{X: panelX + (panelW-pw)/2, Y: y, W: pw, H: playH}
	y += playH + vGap

	r.label = layout.Rect{X: panelX, Y: y, W: panelW, H: labelH}
	y += labelH

	colGap := panelW * 0.04
	rowGap := cardsH * 0.08

	// Prefer 2×3 when it fits; fall back to a single row of six.
	cw3 := (panelW - 2*colGap) / 3
	ch3 := math.Min(cw3*1.32, (cardsH-rowGap)/2)
	gridH23 := ch3*2 + rowGap
	useGrid := gridH23 <= cardsH+1e-9 && cw3 >= m.MinTap*0.55

	if useGrid {
		cw, ch := cw3, ch3
		gridH := ch*2 + rowGap
		gy := y + (cardsH-gridH)/2
		for i := 0; i < 6; i++ {
			col, row := i%3, i/3
			r.cards[i] = layout.Rect{
				X: panelX + float64(col)*(cw+colGap),
				Y: gy + float64(row)*(ch+rowGap),
				W: cw, H: ch,
			}
		}
	} else {
		cw := (panelW - 5*colGap) / 6
		ch := math.Min(cw*1.32, cardsH)
		gy := y + (cardsH-ch)/2
		for i := 0; i < 6; i++ {
			r.cards[i] = layout.Rect{
				X: panelX + float64(i)*(cw+colGap),
				Y: gy,
				W: cw, H: ch,
			}
		}
	}

	r.tray = layout.Rect{X: panelX, Y: panelY + panelH - trayH, W: panelW, H: trayH}
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

	// Title only — stars belong on the board, not in the chrome.
	tcx, tcy := r.title.Center()
	titleSize := render.FitTextSize("Chess Stars", m.TitleSize*1.35, r.title.W*0.66)
	render.DrawTextShadowed(dst, "Chess Stars", tcx, tcy, titleSize, render.ColorText)

	// The Play button breathes just enough to read as "press me", no more.
	pulse := 1 + math.Sin(ctx.T*2.2)*0.012
	pw, ph := r.play.W*pulse, r.play.H*pulse
	px := r.play.X - (pw-r.play.W)/2
	py := r.play.Y - (ph-r.play.H)/2
	render.DrawGlow(dst, r.play.X+r.play.W/2, r.play.Y+r.play.H/2, r.play.W*0.55, render.Alpha(render.ColorPlayHi, 0.15))
	render.DrawChunkyButton(dst, px, py, pw, ph, render.ColorPlay, render.ColorPlayEdge, h.pressed == 0)
	playText := render.FitTextSize("PLAY", ph*0.44, pw*0.44)
	render.DrawTextShadowed(dst, "PLAY", px+pw/2, py+ph*0.52, playText, render.ColorText)

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
