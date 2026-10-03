package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/anim"
	"github.com/testsabirweb/chess-app/internal/challenge"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/sfx"
)

// Which piece? is the star game turned around. Instead of one piece and a
// question about where it goes, there are a few pieces and a star, and the
// question is who can get there. It shows whether he knows the moves or is just
// following the dots.

const (
	// whichShowDur is how long a wrong pick's moves stay on the board before it
	// fades back. Long enough to see the shape, short enough not to be a lecture.
	whichShowDur = 1.5
	// whichShowFade is the last stretch of that time, over which the dots go.
	whichShowFade = 0.3
	// whichDimAlpha is how faint a ruled-out piece is drawn.
	whichDimAlpha = 0.4
	// whichEasyRounds is how many rounds have two pieces before it goes to
	// three.
	whichEasyRounds = 3
)

type whichState int

const (
	whichIdle whichState = iota
	whichHopping
)

type WhichScene struct {
	game *Game
	kit  roundKit

	cur    challenge.Which
	dim    []bool // ruled out this round
	answer int    // index into cur.Pieces
	tries  int    // wrong picks so far this round
	round  int

	// lastAnswers are the types that answered the last two rounds, so a third
	// in a row can be avoided.
	lastAnswers [2]chess.PieceType

	state whichState

	// The piece currently showing what it can do, after a wrong pick.
	showing int // index into cur.Pieces, -1 for none
	showT   float64
	hintBuf []render.Hint

	// holding is true once the right piece has been picked up. It waits there
	// until the child taps the star: the piece never moves on its own.
	holding bool

	hopTween   anim.Tween
	hopFrom    layout.Rect // the answer's square when it set off
	hopX, hopY float64
	won        bool // the answer piece is on the star

	starPulse anim.Pulse
	starScale float64
	starPopT  float64

	wobbleSq  chess.Square
	wobbleT   float64
	wobbleAmp float64
}

func NewWhichScene(g *Game) *WhichScene {
	s := &WhichScene{
		game:      g,
		kit:       newRoundKit(g),
		starPulse: anim.Pulse{Period: 1.6},
		hopTween:  anim.Tween{Duration: 0.42, Ease: anim.EaseInOutCubic},
	}
	s.deal()
	return s
}

// deal sets up the next round: a new set of pieces and a star only one can reach.
func (s *WhichScene) deal() {
	n := 3
	if s.round < whichEasyRounds {
		n = 2
	}
	avoid := chess.NoPiece
	if s.lastAnswers[0] != chess.NoPiece && s.lastAnswers[0] == s.lastAnswers[1] {
		avoid = s.lastAnswers[0]
	}
	s.cur = challenge.NewWhich(s.game.ctx.Rand, beginnerPieces, n, avoid, &s.game.recent)
	s.answer = 0
	for i, sq := range s.cur.Pieces {
		if sq == s.cur.Answer {
			s.answer = i
		}
	}
	s.lastAnswers[0], s.lastAnswers[1] = s.lastAnswers[1], s.cur.Board.At(s.cur.Answer).Type
	s.dim = make([]bool, len(s.cur.Pieces))
	s.tries = 0
	s.round++
	s.state = whichIdle
	s.showing = -1
	s.showT = 0
	s.holding = false
	s.won = false
}

func (s *WhichScene) Update(ctx *Context) error {
	m := ctx.M

	s.starScale = s.starPulse.Update(ctx.DT)
	if s.starPopT > 0 {
		s.starPopT -= ctx.DT
	}
	if s.wobbleT > 0 {
		s.wobbleT -= ctx.DT
	}
	if s.showing >= 0 {
		s.showT -= ctx.DT
		if s.showT <= 0 {
			s.dim[s.showing] = true
			s.showing = -1
		}
	}

	leave, deal := s.kit.update(ctx, m)
	if leave {
		ctx.Switch(NewHomeScene(s.game))
		return nil
	}
	if deal {
		s.deal()
	}

	if s.kit.busy() {
		return nil
	}
	switch s.state {
	case whichIdle:
		for _, ev := range ctx.Pointer.Pressed() {
			s.handleTap(ctx, ev.X, ev.Y, m)
		}
	case whichHopping:
		t := s.hopTween.Update(ctx.DT)
		to := m.CellRect(int(s.cur.Target.File), int(s.cur.Target.Rank))
		tx, ty := to.Center()
		fx, fy := s.hopFrom.Center()
		s.hopX = lerp(fx, tx, t)
		s.hopY = lerp(fy, ty, t) - math.Sin(t*math.Pi)*m.Cell*0.28
		if s.hopTween.Done() {
			s.arrive(ctx, m, tx, ty)
		}
	}
	return nil
}

func (s *WhichScene) handleTap(ctx *Context, x, y float64, m layout.Metrics) {
	f, r, ok := m.HitCell(x, y)
	if !ok {
		return
	}
	sq := chess.Sq(f, r)

	if s.holding {
		switch {
		case sq == s.cur.Target:
			s.startHop(ctx, m)
			return
		case sq == s.cur.Pieces[s.answer]:
			// Tapping the piece again puts it back down.
			ctx.SFX.Play(sfx.SndButton)
			s.holding = false
			return
		}
		// Anything else: put the piece down and treat the tap as a fresh pick,
		// below, so changing his mind is one tap.
		s.holding = false
	}

	for i, ps := range s.cur.Pieces {
		if ps != sq || s.dim[i] {
			continue
		}
		if i == s.answer {
			s.pickUp(ctx)
		} else {
			s.wrong(ctx, i, m)
		}
		return
	}

	if sq == s.cur.Target {
		// A toddler's instinct is to tap the star. Asking the question again
		// teaches more than a buzz would.
		ctx.SFX.Play(sfx.SndButton)
		return
	}
	s.oops(ctx, sq, m)
}

// wrong shows what the piece he picked can do - at once, because this is the
// moment he is looking - and says its name. Nothing is lost: it simply fades
// out of the running so the next try is easier.
func (s *WhichScene) wrong(ctx *Context, i int, m layout.Metrics) {
	if i == s.showing {
		return // already showing it; tapping again must not count as a second miss
	}
	if s.showing >= 0 {
		s.dim[s.showing] = true
	}
	s.tries++
	s.showing = i
	s.showT = whichShowDur
	ctx.SFX.Play(sfx.SndOops)

	s.wobbleSq = s.cur.Pieces[i]
	s.wobbleT = 0.25
	s.wobbleAmp = m.Cell * 0.025

	s.buildHints(i)
}

// buildHints fills hintBuf with where candidate i can move.
func (s *WhichScene) buildHints(i int) {
	s.hintBuf = s.hintBuf[:0]
	for _, to := range s.cur.Board.MoveTargets(s.cur.Pieces[i]) {
		s.hintBuf = append(s.hintBuf, render.Hint{
			Square:  to,
			Capture: !s.cur.Board.At(to).IsEmpty(),
			Target:  to == s.cur.Target,
		})
	}
}

// pickUp lifts the right piece and shows its moves, star included, then waits.
// The moves show at once: he has already answered the question, and the next
// step - tapping the star himself - is the part that is his to do.
func (s *WhichScene) pickUp(ctx *Context) {
	ctx.SFX.Play(sfx.SndButton)
	if s.showing >= 0 {
		s.dim[s.showing] = true
		s.showing = -1
	}
	s.holding = true
	s.buildHints(s.answer)
}

func (s *WhichScene) oops(ctx *Context, sq chess.Square, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndOops)
	s.wobbleSq = sq
	s.wobbleT = 0.25
	s.wobbleAmp = m.Cell * 0.025
}

func (s *WhichScene) startHop(ctx *Context, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndButton)
	s.holding = false
	from := s.cur.Pieces[s.answer]
	s.hopFrom = m.CellRect(int(from.File), int(from.Rank))
	s.hopX, s.hopY = s.hopFrom.Center()
	s.hopTween.Start()
	s.state = whichHopping
}

func (s *WhichScene) arrive(ctx *Context, m layout.Metrics, cx, cy float64) {
	s.won = true
	ctx.SFX.Play(sfx.SndLand)
	ctx.SFX.Play(sfx.SndCheer)
	ctx.SFX.Play(sfx.SndPop)

	// Right first time is the shortest route here.
	perfect := s.tries == 0
	if perfect {
		ctx.SFX.Play(sfx.SndMilestone)
	}
	s.kit.win(ctx, m, cx, cy, perfect)
	s.starPopT = 0.3
	s.state = whichIdle
}

// hintFade is how strongly a wrong pick's moves are showing: full, then
// gone over the last whichShowFade seconds.
func (s *WhichScene) hintFade() float64 {
	if s.showing < 0 {
		return 0
	}
	if s.showT >= whichShowFade {
		return 1
	}
	return s.showT / whichShowFade
}

func (s *WhichScene) Draw(dst *ebiten.Image, ctx *Context) {
	m := ctx.M
	render.DrawBackground(dst, m)

	s.kit.drawHeader(dst, m, ModeWhich.info().name)
	render.DrawBoard(dst, m)

	idle := s.state == whichIdle && !s.kit.busy()
	if s.holding && idle {
		render.DrawMoveHints(dst, m, s.hintBuf, 1)
		render.DrawPickableRing(dst, m, s.cur.Pieces[s.answer], s.starScale, true, ctx.T)
	} else if s.showing >= 0 {
		render.DrawMoveHints(dst, m, s.hintBuf, s.hintFade())
	}
	if idle && !s.holding {
		// Every piece still in the running breathes, saying "pick me".
		for i, sq := range s.cur.Pieces {
			if !s.dim[i] {
				render.DrawPickableRing(dst, m, sq, s.starScale, false, ctx.T)
			}
		}
	}
	if s.wobbleT > 0 {
		render.DrawSquareTint(dst, m, s.wobbleSq, 0, 0, render.Alpha(render.ColorShadow, 0.5))
	}

	if !s.kit.celebrating() || s.starPopT > 0 {
		scale := s.starScale
		if s.starPopT > 0 {
			scale = s.starScale * (1 + (1-s.starPopT/0.3)*1.4)
		}
		render.DrawStar(dst, s.cur.Target, m, scale, ctx.T)
	}

	for i, sq := range s.cur.Pieces {
		if i == s.answer && (s.state == whichHopping || s.won) {
			continue // drawn last, on top
		}
		s.drawPiece(dst, m, i, sq)
	}
	if s.state == whichHopping || s.won {
		s.drawHopper(dst, m)
	}

	s.kit.drawOverlays(dst, ctx, m)
}

// drawPiece draws candidate i standing on its square. A ruled-out piece is
// faint and a wrong pick wobbles.
func (s *WhichScene) drawPiece(dst *ebiten.Image, m layout.Metrics, i int, sq chess.Square) {
	cr := m.CellRect(int(sq.File), int(sq.Rank))
	if s.wobbleT > 0 && sq == s.wobbleSq {
		wx, _ := render.WobbleOffset(1-s.wobbleT/0.25, s.wobbleAmp)
		cr.X += wx
	}
	p := s.cur.Board.At(sq)
	alpha := 1.0
	if s.dim[i] {
		alpha = whichDimAlpha
	}
	lift := 0.0
	if s.holding && i == s.answer {
		lift = m.Cell * 0.12
	}
	render.DrawPieceAlpha(dst, p, cr, lift, true, alpha)
	// The dark-square bishop keeps its scarf, as in the star game.
	if p.Type == chess.Bishop && render.DarkSquare(sq) && !s.dim[i] {
		render.DrawBishopScarf(dst, cr, lift)
	}
}

// drawHopper draws the right piece on its way to the star, and sitting on it
// afterwards.
func (s *WhichScene) drawHopper(dst *ebiten.Image, m layout.Metrics) {
	x, y := s.hopX, s.hopY
	if s.won {
		to := m.CellRect(int(s.cur.Target.File), int(s.cur.Target.Rank))
		x, y = to.Center()
	}
	cr := layout.Rect{X: x - m.Cell/2, Y: y - m.Cell/2, W: m.Cell, H: m.Cell}
	lift := 0.0
	if s.state == whichHopping {
		lift = m.Cell * 0.05
	}
	p := s.cur.Board.At(s.cur.Answer)
	render.DrawPiece(dst, p, cr, lift, true)
	if p.Type == chess.Bishop && render.DarkSquare(s.cur.Target) {
		render.DrawBishopScarf(dst, cr, lift)
	}
}
