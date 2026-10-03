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
	pendMode Mode
	pendPlay bool
}

func NewHomeScene(g *Game) *HomeScene { return &HomeScene{game: g, pressed: -1} }

type homeRects struct {
	title, play, label, tray layout.Rect
	cards                    [6]layout.Rect
	// modeRow holds the mode buttons, one per rowModes entry in modes. It is
	// empty (zero height) when there is only one mode to choose.
	modeRow layout.Rect
	modes   [len(modeTable)]layout.Rect
}

// layoutModes spreads one square button per active mode across the row.
func (r *homeRects) layoutModes(row layout.Rect) {
	n := len(rowModes())
	if n < 1 || row.H <= 0 {
		return
	}
	r.modeRow = row
	g := row.W * 0.03
	tile := math.Min(row.H, (row.W*0.94-float64(n-1)*g)/float64(n))
	x := row.X + (row.W-(float64(n)*tile+float64(n-1)*g))/2
	y := row.Y + (row.H-tile)/2
	for i := 0; i < n; i++ {
		r.modes[i] = layout.Rect{X: x + float64(i)*(tile+g), Y: y, W: tile, H: tile}
	}
}

func homeDp(m layout.Metrics, v float64) float64 {
	return v * m.Scale
}

func scaleHomeBands(titleH, playH, modeH, labelH, trayH, gap *float64, cardsH *float64, safeH float64) {
	gaps := 4 * *gap
	if *modeH > 0 {
		gaps += *gap
	}
	total := *titleH + *playH + *modeH + *labelH + *trayH + gaps
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
	*modeH *= scale
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
	modeH := 0.0
	gaps := 4 * gap
	if len(rowModes()) > 0 {
		modeH = math.Min(math.Max(s.H*0.07, m.MinTap), homeDp(m, 64))
		gaps += gap
	}

	cardsH := s.H - titleH - playH - modeH - labelH - trayH - gaps
	scaleHomeBands(&titleH, &playH, &modeH, &labelH, &trayH, &gap, &cardsH, s.H)

	var r homeRects
	y := s.Y
	r.title = layout.Rect{X: s.X, Y: y, W: s.W, H: titleH}
	y += titleH + gap

	pw := s.W * 0.82
	r.play = layout.Rect{X: s.X + (s.W-pw)/2, Y: y, W: pw, H: playH}
	y += playH + gap

	if modeH > 0 {
		r.layoutModes(layout.Rect{X: s.X, Y: y, W: s.W, H: modeH})
		y += modeH + gap
	}

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
	modeH, vGaps := 0.0, 2.0
	if len(rowModes()) > 0 {
		modeH = math.Min(math.Max(panelH*0.12, m.MinTap), homeDp(m, 56))
		vGaps++
	}

	content := playH + modeH + labelH + trayH + vGaps*vGap
	cardsH := panelH - content
	if content > panelH {
		scale := panelH / content
		playH *= scale
		modeH *= scale
		labelH *= scale
		trayH *= scale
		vGap *= scale
		cardsH = panelH - playH - modeH - labelH - trayH - vGaps*vGap
	}

	y := panelY
	pw := panelW * 0.82
	r.play = layout.Rect{X: panelX + (panelW-pw)/2, Y: y, W: pw, H: playH}
	y += playH + vGap

	if modeH > 0 {
		r.layoutModes(layout.Rect{X: panelX, Y: y, W: panelW, H: modeH})
		y += modeH + vGap
	}

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
				ctx.Switch(newModeScene(h.game, h.pendMode, h.pendPick))
			}
		}
		return nil
	}

	for _, ev := range ctx.Pointer.Pressed() {
		if r.play.Contains(ev.X, ev.Y) {
			h.arm(ctx, 0, h.game.mode, chess.Rook)
			return nil
		}
		for i, mode := range rowModes() {
			if !r.modes[i].Contains(ev.X, ev.Y) {
				continue
			}
			if mode.info().needsPiece {
				// A game that needs a piece waits for the piece card, and the
				// gold frame shows it took. Tapping it again drops back to the
				// star game the cards play by default.
				ctx.SFX.Play(sfx.SndButton)
				if h.game.mode == mode {
					h.game.mode = ModeStar
				} else {
					h.game.mode = mode
				}
			} else {
				h.arm(ctx, -1, mode, chess.Rook)
			}
			return nil
		}
		for i, cr := range r.cards {
			if cr.Contains(ev.X, ev.Y) {
				h.arm(ctx, i+1, h.game.mode, allPieces[i])
				return nil
			}
		}
	}
	return nil
}

// arm presses a button and, once it has visibly squashed, starts the game.
// slot is the button's place in the squash animation (-1 for none).
func (h *HomeScene) arm(ctx *Context, slot int, mode Mode, pt chess.PieceType) {
	ctx.SFX.Play(sfx.SndButton)
	h.pressed = slot
	h.pressT = pressHold
	h.pendPick = pt
	h.pendMode = mode
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

	h.drawModes(dst, ctx, r)

	label := "Pick a piece"
	if h.game.mode != ModeStar {
		label = h.game.mode.info().name + " - pick a piece"
	}
	lcx, lcy := r.label.Center()
	render.DrawTextShadowed(dst, label, lcx, lcy, render.FitTextSize(label, m.BodySize*1.05, r.label.W*0.95), render.ColorTextDim)

	for i, cr := range r.cards {
		h.drawCard(dst, ctx, i, cr)
	}

	h.drawTray(dst, ctx, r.tray)
}

// drawModes draws one round button per game. The one picked for the piece cards
// sits in a gold frame.
func (h *HomeScene) drawModes(dst *ebiten.Image, ctx *Context, r homeRects) {
	for i, mode := range rowModes() {
		tr := r.modes[i]
		if tr.W <= 0 {
			continue
		}
		if mode == h.game.mode {
			pad := tr.W * 0.08
			render.FillRoundRect(dst, tr.X-pad, tr.Y-pad, tr.W+2*pad, tr.H+2*pad, tr.W*0.42+pad, render.ColorStarGlow)
		}
		render.DrawChunkyButton(dst, tr.X, tr.Y, tr.W, tr.H, render.ColorModeTile, render.ColorModeTileEdge, false)
		cx, cy := tr.Center()
		render.DrawEmoji(dst, mode.info().icon, cx, cy-tr.H*0.02, tr.W*0.58, 0, 1)
	}
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
