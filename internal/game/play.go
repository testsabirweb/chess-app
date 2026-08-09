package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/anim"
	"github.com/testsabirweb/chess-app/internal/challenge"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/sfx"
)

type playState int

const (
	stateIdle playState = iota
	stateMoving
	stateCelebrating
	stateAdvancing
)

type PlayScene struct {
	game      *Game
	gen       *challenge.Generator
	cur       challenge.Challenge
	state     playState
	pieceType chess.PieceType

	moveTween anim.Tween
	fadeTween anim.Tween
	starPulse anim.Pulse
	confetti  anim.Confetti
	sprites   *render.Sprites

	pieceX, pieceY float64
	fromX, fromY     float64
	toX, toY         float64
	starScale        float64

	pieceSelected bool
	wobbleSq      chess.Square
	wobbleT       float64
	wobbleAmp     float64
	stickerRow    []chess.PieceType
}

func NewPlayScene(g *Game, pt chess.PieceType) *PlayScene {
	spec := challenge.Spec{
		BoardWidth: 5, BoardHeight: 5,
		Pieces: []chess.PieceType{pt},
		Color:  chess.White,
		Decoys: pt == chess.Pawn,
	}
	ps := &PlayScene{
		game:      g,
		gen:       challenge.NewGenerator(spec, g.ctx.Rand),
		pieceType: pt,
		starPulse: anim.Pulse{Period: 1.2},
		sprites:   render.NewSprites(),
		moveTween: anim.Tween{Duration: 0.45, Ease: anim.EaseInOutCubic},
		fadeTween: anim.Tween{Duration: 0.25, Ease: anim.EaseOutCubic},
	}
	ps.newChallenge()
	return ps
}

func (p *PlayScene) newChallenge() {
	p.cur = p.gen.Next()
	p.pieceX, p.pieceY = 0, 0
	p.pieceSelected = false
}

func (p *PlayScene) syncPiecePos(m layout.Metrics) {
	cr := m.CellRect(int(p.cur.From.File), int(p.cur.From.Rank))
	p.pieceX, p.pieceY = cr.Center()
}

func (p *PlayScene) Update(ctx *Context) error {
	m := ctx.M
	if p.pieceX == 0 && p.pieceY == 0 {
		p.syncPiecePos(m)
	}

	p.starScale = p.starPulse.Update(ctx.DT)
	p.confetti.Update(ctx.DT, m.Cell*8)

	homeRect := homeButtonRect(m)
	for _, ev := range ctx.Pointer.Pressed() {
		if homeRect.Contains(ev.X, ev.Y) {
			ctx.SFX.Play(sfx.SndButton)
			ctx.Switch(NewHomeScene(p.game))
			return nil
		}
		if p.state == stateIdle {
			p.handleTap(ctx, ev.X, ev.Y, m)
		}
	}

	switch p.state {
	case stateMoving:
		prog := p.moveTween.Update(ctx.DT)
		arc := -math.Sin(prog*math.Pi) * m.Cell * 0.22
		p.pieceX = lerp(p.fromX, p.toX, prog)
		p.pieceY = lerp(p.fromY, p.toY, prog) + arc
		if p.moveTween.Done() {
			ctx.SFX.Play(sfx.SndLand)
			p.state = stateCelebrating
			cr := m.CellRect(int(p.cur.Target.File), int(p.cur.Target.Rank))
			cx, cy := cr.Center()
			p.confetti.Burst(ctx.Rand, cx, cy, 40, m.Cell)
			ctx.SFX.Play(sfx.SndCheer)
			p.stickerRow = append(p.stickerRow, p.pieceType)
		}
	case stateCelebrating:
		if !p.confetti.Alive() {
			p.state = stateAdvancing
			p.fadeTween.Start()
		}
	case stateAdvancing:
		if p.fadeTween.Update(ctx.DT) >= 1 {
			p.newChallenge()
			p.syncPiecePos(m)
			p.state = stateIdle
			p.moveTween.Reset()
		}
	}

	if p.wobbleT > 0 {
		p.wobbleT -= ctx.DT
	}
	return nil
}

func (p *PlayScene) handleTap(ctx *Context, x, y float64, m layout.Metrics) {
	f, r, ok := m.HitCell(x, y)
	if !ok {
		return
	}
	sq := chess.Sq(f, r)

	if !p.pieceSelected {
		if sq == p.cur.From {
			ctx.SFX.Play(sfx.SndButton)
			p.pieceSelected = true
		} else {
			ctx.SFX.Play(sfx.SndOops)
			p.wobbleSq = sq
			p.wobbleT = 0.25
			p.wobbleAmp = m.Cell * 0.02
		}
		return
	}

	if m.HitStar(x, y, p.cur.Target, p.cur.Solutions) {
		cr := m.CellRect(int(p.cur.Target.File), int(p.cur.Target.Rank))
		p.fromX, p.fromY = p.pieceX, p.pieceY
		p.toX, p.toY = cr.Center()
		p.moveTween.Start()
		p.state = stateMoving
		p.pieceSelected = false
		return
	}

	for _, sol := range p.cur.Solutions {
		if sol == sq {
			ctx.SFX.Play(sfx.SndNear)
			p.wobbleSq = sq
			p.wobbleT = 0.35
			p.wobbleAmp = m.Cell * 0.04
			return
		}
	}

	ctx.SFX.Play(sfx.SndOops)
	p.wobbleSq = sq
	p.wobbleT = 0.25
	p.wobbleAmp = m.Cell * 0.02
}

func (p *PlayScene) Draw(dst *ebiten.Image, ctx *Context) {
	m := ctx.M
	render.DrawFilledRect(dst, 0, 0, m.W, m.H, render.ColorBG)

	render.DrawTextCentered(dst, render.PieceName(p.pieceType), m.Header.X+m.Header.W/2, m.Header.Y+m.Header.H*0.35, m.TitleSize, render.ColorText)
	pr := layout.Rect{X: m.Header.X + m.Header.W*0.35, Y: m.Header.Y + m.Header.H*0.45, W: m.Header.W * 0.3, H: m.Header.H * 0.5}
	render.DrawPiece(dst, chess.Piece{Type: p.pieceType, Color: chess.White}, pr)

	render.DrawBoard(dst, m)
	if p.pieceSelected && p.state == stateIdle {
		render.DrawMoveHints(dst, m, p.cur.Solutions)
		pcr := m.CellRect(int(p.cur.From.File), int(p.cur.From.Rank))
		render.DrawFilledRect(dst, pcr.X, pcr.Y, pcr.W, pcr.H, colorAlpha(render.ColorAccent, 0.2))
	}

	if p.state == stateIdle || p.state == stateMoving {
		render.DrawStar(dst, p.cur.Target, m, p.starScale)
	}

	wx, wy := 0.0, 0.0
	if p.wobbleT > 0 {
		wx, wy = render.WobbleOffset(1-p.wobbleT/0.35, p.wobbleAmp)
	}

	var cr layout.Rect
	if p.state == stateMoving || p.state == stateCelebrating || p.state == stateAdvancing {
		cr = layout.Rect{X: p.pieceX - m.Cell/2, Y: p.pieceY - m.Cell/2, W: m.Cell, H: m.Cell}
	} else {
		cr = m.CellRect(int(p.cur.From.File), int(p.cur.From.Rank))
	}
	cr.X += wx
	cr.Y += wy
	render.DrawPiece(dst, p.cur.Piece, cr)

	if p.wobbleT > 0 {
		wcr := m.CellRect(int(p.wobbleSq.File), int(p.wobbleSq.Rank))
		render.DrawFilledRect(dst, wcr.X+wx, wcr.Y+wy, wcr.W, wcr.H, colorAlpha(render.ColorShadow, 0.4))
	}

	render.DrawConfetti(dst, &p.confetti, p.sprites, m.Cell)

	stickerW := m.Cell * 0.35
	for i := range p.stickerRow {
		if i >= 5 {
			break
		}
		sx := m.Footer.X + float64(i)*(stickerW+4) + 8
		sy := m.Footer.Y + m.Footer.H*0.15
		render.DrawPath(dst, render.StarPath(), sx, sy, stickerW, stickerW, render.ColorAccent, render.ColorOutline, 2)
	}

	home := homeButtonRect(m)
	render.DrawRoundedButton(dst, home.X, home.Y, home.W, home.H, render.ColorButton)
	render.DrawTextCentered(dst, "Home", home.X+home.W/2, home.Y+home.H/2, m.BodySize, render.ColorText)

	if p.state == stateAdvancing {
		a := p.fadeTween.Progress()
		render.DrawFilledRect(dst, 0, 0, m.W, m.H, colorAlpha(render.ColorBG, a*0.7))
	}
}

func homeButtonRect(m layout.Metrics) layout.Rect {
	w := m.Safe.W * 0.5
	h := m.Footer.H * 0.45
	return layout.Rect{X: m.Safe.X + (m.Safe.W-w)/2, Y: m.Footer.Y + m.Footer.H*0.5, W: w, H: h}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func colorAlpha(c color.RGBA, a float64) color.RGBA {
	return color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * a)}
}
