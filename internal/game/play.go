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

type playState int

// A round's win and the wind-down after it belong to roundKit; the scene only
// tells idle from mid-hop.
const (
	stateIdle playState = iota
	stateMoving
)

// maxJourney bounds how far away the star may be planted, in moves.
const maxJourney = 3

// defaultHintDelay is how long the piece sits picked up before the legal-move
// dots fade in, and hintFadeIn is how long they take to arrive once it is over.
//
// The pause is the whole point. A child who already knows where the piece can
// go never waits for it - tapping a legal square moves the piece whether the
// dots are showing or not - so the delay only costs something when you were
// going to read the overlay instead of the board. That is precisely the habit
// it exists to interrupt.
const (
	defaultHintDelay = 2.5
	hintFadeIn       = 0.45
)

// movePath holds the waypoints a piece travels through during a move. Most
// pieces go straight from A to B (n=2); the knight takes an L via a corner (n=3).
type movePath struct {
	pts  [3]struct{ x, y float64 }
	n    int     // 2 or 3
	leg  int     // which leg is running
	hold float64 // seconds left of the pause at the corner
}

type PlayScene struct {
	game *Game
	// next deals the following puzzle. The star game deals one target and the
	// treasure game several, through the same scene.
	next      func() challenge.Puzzle
	cur       challenge.Puzzle
	pieceType chess.PieceType

	// Live state of the current puzzle. The piece really moves across `board`,
	// so sliders, captures and blocked paths all stay correct move after move.
	board *chess.Board
	at    chess.Square
	// targets are the stars still to collect, in the order they were dealt.
	// When capture is set (worked out for each round) they are black pawns
	// standing on the board instead of stars, and landing on one takes it.
	targets []chess.Square
	capture bool

	// The Stay-safe game: a guard that never moves, and the squares it attacks,
	// where the piece may not stop. avoid is nil in every other game.
	safe      bool
	guard     chess.Square
	hot       []chess.Square
	avoid     func(chess.Square) bool
	safeRound int
	flashT    float64 // the red squares flash after a refused move
	lungeT    float64 // the guard lunges at the square that was tapped
	lungeTo   chess.Square
	solutions []chess.Square
	steps     int

	kit roundKit

	state         playState
	pieceSelected bool
	laidOut       bool

	// hintT counts the pause down after the piece is picked up; the dots fade
	// in over its last hintFadeIn seconds.
	hintT float64

	// optimal is the fewest moves this puzzle can be solved in from where the
	// piece started, kept in step with the star when it relocates. Matching it
	// earns the bigger celebration.
	optimal int

	moveTween anim.Tween
	movePath  movePath
	arcScale  float64

	trailPts  [3][2]float64
	trailN    int
	trailFade float64
	// trailHold keeps the trail at full strength before it begins fading out.
	trailHold float64

	starPulse anim.Pulse

	pieceX, pieceY float64
	fromX, fromY   float64
	toX, toY       float64
	moveTo         chess.Square
	starScale      float64
	// starPopT times the last star's flash as it is collected, at popSq.
	starPopT float64
	popSq    chess.Square

	wobbleSq  chess.Square
	wobbleT   float64
	wobbleAmp float64

	hintBuf []render.Hint
}

// NewPlayScene is the star game: one star to find, with the piece the child
// picked.
func NewPlayScene(g *Game, pt chess.PieceType) *PlayScene {
	spec := challenge.Spec{
		BoardWidth: 5, BoardHeight: 5,
		Pieces:   []chess.PieceType{pt},
		Color:    chess.White,
		MinMoves: 1,
		MaxMoves: maxJourney,
		Decoys:   pt == chess.Pawn,
	}
	gen := challenge.NewGenerator(spec, g.ctx.Rand)
	gen.SetMemory(&g.recent)
	return newPlayScene(g, pt, func() challenge.Puzzle {
		c := gen.Next()
		return challenge.Puzzle{
			Board: c.Board, From: c.From, Piece: c.Piece,
			Targets: []chess.Square{c.Target}, Optimal: c.Moves,
		}
	})
}

// multiEasyRounds is how many rounds of the games with several targets have
// two before they go to three.
const multiEasyRounds = 3

// multiSource deals rounds from a generator that takes the target count and the
// memory of recent squares, so a round does not look like the last ones.
func multiSource(g *Game, deal func(round, k int, mem *challenge.Memory) challenge.Puzzle) func() challenge.Puzzle {
	round := 0
	return func() challenge.Puzzle {
		k := 3
		if round < multiEasyRounds {
			k = 2
		}
		p := deal(round, k, &g.recent)
		round++
		return p
	}
}

// NewTreasureScene is the collect game: several targets on the board, collected
// in any order. Rounds alternate between stars (landing on an empty square) and
// black pawns (capturing them), so the same game also teaches taking pieces:
// the pawns block a rook's line until they are taken, and a pawn can only take
// diagonally.
func NewTreasureScene(g *Game, pt chess.PieceType) *PlayScene {
	return newPlayScene(g, pt, multiSource(g, func(round, k int, mem *challenge.Memory) challenge.Puzzle {
		if round%2 == 1 {
			return challenge.NewCatch(g.ctx.Rand, pt, chess.White, k, mem)
		}
		return challenge.NewTreasure(g.ctx.Rand, pt, chess.White, k, mem)
	}))
}

// safeTeachRounds is how many rounds of Stay safe keep the red squares on show
// the whole time. After that they only appear with the move dots, once the
// pause is over, so he starts spotting the danger himself before it is drawn.
const safeTeachRounds = 5

// Timings for a refused move.
const (
	safeFlashDur = 1.2
	safeLungeDur = 0.35
)

// NewSafeScene is the stay-safe game: find the star without stopping on a
// square a guard attacks. The guard never moves and is never captured.
func NewSafeScene(g *Game, pt chess.PieceType) *PlayScene {
	round := 0
	return newPlayScene(g, pt, func() challenge.Puzzle {
		// Every other round asks for a detour: the safe way round is longer.
		detour := round%2 == 1
		round++
		return challenge.NewSafe(g.ctx.Rand, pt, chess.White, beginnerPieces, &g.recent, detour)
	}, func(p *PlayScene) { p.safe = true })
}

// A playOption sets up a scene before its first puzzle is dealt.
type playOption func(*PlayScene)

func newPlayScene(g *Game, pt chess.PieceType, next func() challenge.Puzzle, opts ...playOption) *PlayScene {
	ps := &PlayScene{
		game:      g,
		kit:       newRoundKit(g),
		next:      next,
		pieceType: pt,
		starPulse: anim.Pulse{Period: 1.6},
		moveTween: anim.Tween{Duration: 0.42, Ease: anim.EaseInOutCubic},
	}
	for _, o := range opts {
		o(ps)
	}
	ps.newChallenge()
	return ps
}

func (p *PlayScene) newChallenge() {
	p.cur = p.next()
	if p.safe {
		p.guard, p.hot = p.cur.Guard, p.cur.Hot
		p.avoid = challenge.HotSet(p.hot, p.guard)
		p.safeRound++
	}
	p.board = p.cur.Board.Clone()
	p.at = p.cur.From
	p.targets = append(p.targets[:0], p.cur.Targets...)
	// A round's targets are pawns to take if there are pieces standing on them,
	// and stars to land on if the squares are empty.
	p.capture = len(p.targets) > 0 && !p.board.At(p.targets[0]).IsEmpty()
	p.solutions = p.board.MoveTargets(p.at)
	p.steps = 0
	p.optimal = p.cur.Optimal
	p.pieceSelected = false
	p.hintT = 0
	p.laidOut = false
}

func (p *PlayScene) syncPiecePos(m layout.Metrics) {
	cr := m.CellRect(int(p.at.File), int(p.at.Rank))
	p.pieceX, p.pieceY = cr.Center()
}

func (p *PlayScene) Update(ctx *Context) error {
	m := ctx.M
	if !p.laidOut {
		p.syncPiecePos(m)
		p.laidOut = true
	}

	p.starScale = p.starPulse.Update(ctx.DT)
	if p.starPopT > 0 {
		p.starPopT -= ctx.DT
	}
	if p.wobbleT > 0 {
		p.wobbleT -= ctx.DT
	}
	if p.flashT > 0 {
		p.flashT -= ctx.DT
	}
	if p.lungeT > 0 {
		p.lungeT -= ctx.DT
	}
	if p.hintT > 0 {
		p.hintT -= ctx.DT
	}
	if p.trailHold > 0 {
		p.trailHold -= ctx.DT
	} else if p.trailFade > 0 {
		p.trailFade -= ctx.DT / trailFadeDur
		if p.trailFade < 0 {
			p.trailFade = 0
		}
	}

	leave, deal := p.kit.update(ctx, m)
	if leave {
		ctx.Switch(NewHomeScene(p.game))
		return nil
	}
	if deal {
		p.newChallenge()
		p.syncPiecePos(m)
		p.laidOut = true
		p.moveTween.Reset()
	}
	if p.idle() {
		for _, ev := range ctx.Pointer.Pressed() {
			p.handleTap(ctx, ev.X, ev.Y, m)
		}
	}

	switch p.state {
	case stateMoving:
		if p.movePath.hold > 0 {
			p.movePath.hold -= ctx.DT
			end := p.movePath.leg + 1
			p.pieceX = p.movePath.pts[end].x
			p.pieceY = p.movePath.pts[end].y
			if p.movePath.hold <= 0 {
				p.movePath.leg++
				if p.movePath.n == 3 {
					p.moveTween.Duration = 0.24
				}
				p.moveTween.Start()
			}
		} else {
			leg := p.movePath.leg
			from := p.movePath.pts[leg]
			to := p.movePath.pts[leg+1]
			prog := p.moveTween.Update(ctx.DT)
			arc := -math.Sin(prog*math.Pi) * m.Cell * p.arcScale
			p.pieceX = lerp(from.x, to.x, prog)
			p.pieceY = lerp(from.y, to.y, prog) + arc
			if p.moveTween.Done() {
				if leg+1 < p.movePath.n-1 {
					p.pieceX = to.x
					p.pieceY = to.y
					p.movePath.hold = 0.12
					if p.pieceType == chess.Knight {
						p.setTrailFromPath(trailMidLeg)
					}
					ctx.SFX.Play(sfx.SndStep)
				} else {
					p.land(ctx, m)
				}
			}
		}
	}
	return nil
}

// Knight trail timing: show the planned L as soon as the move starts, brighten
// at the corner, then hold at full strength so a toddler can trace the shape.
const (
	trailFadeDur = 2.5
	trailHoldDur = 1.4
	trailPreview = 0.50 // ghost L while the first leg is running
	trailMidLeg  = 0.85 // first leg done, corner locked in
)

func (p *PlayScene) handleTap(ctx *Context, x, y float64, m layout.Metrics) {
	f, r, ok := m.HitCell(x, y)
	if !ok {
		return
	}
	sq := chess.Sq(f, r)

	if !p.pieceSelected {
		// Any tap on the board picks the piece up. A toddler's instinct is to
		// tap the star, and answering that with a buzz teaches nothing; showing
		// them what the piece can do does - after a beat to look first.
		p.pickUp(ctx)
		return
	}

	if sq == p.at {
		// Tapping the piece again puts it back down.
		ctx.SFX.Play(sfx.SndButton)
		p.pieceSelected = false
		p.hintT = 0
		return
	}

	// A generous magnet around a star, but only when that star is one move
	// away - otherwise the child would jump a step they have not earned.
	for _, t := range p.targets {
		if containsSquare(p.solutions, t) && m.HitStar(x, y, t, p.solutions) {
			p.startMove(t, m)
			return
		}
	}
	if containsSquare(p.solutions, sq) {
		if p.avoid != nil && p.avoid(sq) {
			p.refuse(ctx, sq, m)
			return
		}
		p.startMove(sq, m)
		return
	}
	p.oops(ctx, sq, m)
}

// refuse turns down a move onto a square the guard attacks: the guard lunges at
// it, the red squares flash, and the piece stays put. Nothing is lost - it is
// the same soft wobble as any other wrong tap, with the reason made visible.
func (p *PlayScene) refuse(ctx *Context, sq chess.Square, m layout.Metrics) {
	p.oops(ctx, sq, m)
	p.flashT = safeFlashDur
	p.lungeT = safeLungeDur
	p.lungeTo = sq
}

// pickUp holds the piece and starts the pause before its moves are shown.
func (p *PlayScene) pickUp(ctx *Context) {
	ctx.SFX.Play(sfx.SndButton)
	p.pieceSelected = true
	p.hintT = p.game.hintDelay
}

// hintFade is how strongly the move dots are showing: nothing during the
// pause, ramping to full over its last hintFadeIn seconds.
func (p *PlayScene) hintFade() float64 {
	if !p.pieceSelected || !p.idle() {
		return 0
	}
	if p.hintT <= 0 {
		return 1
	}
	if p.hintT >= hintFadeIn {
		return 0
	}
	return 1 - p.hintT/hintFadeIn
}

func (p *PlayScene) oops(ctx *Context, sq chess.Square, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndOops)
	p.wobbleSq = sq
	p.wobbleT = 0.25
	p.wobbleAmp = m.Cell * 0.025
}

func (p *PlayScene) startMove(to chess.Square, m layout.Metrics) {
	from := p.at
	toCR := m.CellRect(int(to.File), int(to.Rank))

	p.movePath.n = 2
	p.movePath.leg = 0
	p.movePath.hold = 0
	p.movePath.pts[0].x, p.movePath.pts[0].y = p.pieceX, p.pieceY
	p.movePath.pts[1].x, p.movePath.pts[1].y = toCR.Center()

	p.arcScale = 0.28
	p.moveTween.Duration = 0.42

	if p.pieceType == chess.Knight {
		df := int(to.File) - int(from.File)
		if df < 0 {
			df = -df
		}
		var corner chess.Square
		if df == 2 {
			corner = chess.Sq(int(to.File), int(from.Rank))
		} else {
			corner = chess.Sq(int(from.File), int(to.Rank))
		}
		cornerCR := m.CellRect(int(corner.File), int(corner.Rank))
		p.movePath.pts[1].x, p.movePath.pts[1].y = cornerCR.Center()
		p.movePath.pts[2].x, p.movePath.pts[2].y = toCR.Center()
		p.movePath.n = 3
		p.moveTween.Duration = 0.30
		p.arcScale = 0.16
		p.setTrailFromPath(trailPreview)
	}

	p.fromX, p.fromY = p.movePath.pts[0].x, p.movePath.pts[0].y
	p.toX, p.toY = p.movePath.pts[1].x, p.movePath.pts[1].y
	p.moveTo = to
	p.moveTween.Start()
	p.state = stateMoving
}

// land applies the finished move to the board and decides what happens next.
func (p *PlayScene) setTrailFromPath(fade float64) {
	p.trailN = p.movePath.n
	for i := 0; i < p.trailN; i++ {
		p.trailPts[i][0] = p.movePath.pts[i].x
		p.trailPts[i][1] = p.movePath.pts[i].y
	}
	p.trailFade = fade
	p.trailHold = 0
}

func (p *PlayScene) land(ctx *Context, m layout.Metrics) {
	if p.pieceType == chess.Knight {
		p.setTrailFromPath(1)
		p.trailHold = trailHoldDur
	}

	captured := !p.board.At(p.moveTo).IsEmpty()
	p.board.Set(p.at, chess.Piece{})
	p.board.Set(p.moveTo, p.cur.Piece)
	p.at = p.moveTo
	p.solutions = p.board.MoveTargets(p.at)
	p.steps++
	p.syncPiecePos(m)

	if i := indexOfSquare(p.targets, p.at); i >= 0 {
		p.targets = append(p.targets[:i], p.targets[i+1:]...)
		if len(p.targets) == 0 {
			// The hop is over. Leaving the state at stateMoving would have the
			// next frame find the finished tween and land all over again,
			// collecting the same star every frame; the kit's busy flag keeps
			// the board quiet now.
			p.state = stateIdle
			p.collectStar(ctx, m)
			return
		}
		// One star of several: a small pop, and play on.
		ctx.SFX.Play(sfx.SndPop)
		cr := m.CellRect(int(p.at.File), int(p.at.Rank))
		cx, cy := cr.Center()
		p.kit.confetti.Burst(ctx.Rand, cx, cy, 14, m.Cell)
	} else if captured {
		ctx.SFX.Play(sfx.SndPop)
	} else {
		ctx.SFX.Play(sfx.SndStep)
	}
	// Put the piece down again. Holding it across the whole journey left the
	// dots up from the first tap to the last, which turns the overlay into a
	// trail to follow to the star; dropping it means every hop starts from a
	// clean board and earns its own look before the dots come back.
	p.pieceSelected = false
	p.hintT = 0
	p.state = stateIdle

	p.keepStarsReachable(ctx)
}

// keepStarsReachable moves any star the piece can no longer get to somewhere it
// can. It is the safety net for a wandering toddler: the game never becomes
// unwinnable and never scolds, the star just twinkles somewhere new.
func (p *PlayScene) keepStarsReachable(ctx *Context) {
	moved := false
	for i, t := range p.targets {
		if challenge.CanReachAvoiding(p.board, p.at, t, maxJourney+2, p.avoid) {
			continue
		}
		if !p.relocateStar(ctx, i) {
			// The piece is completely stuck (a pawn on the far rank); start over.
			p.kit.restart()
			return
		}
		moved = true
	}
	if !moved {
		return
	}
	ctx.SFX.Play(sfx.SndHop)
	// The moves already spent still count, so a wandering journey can no longer
	// come out "perfect" - but it is never scored as a failure either. If no
	// shortest route can be worked out, "perfect" is simply off the table.
	p.optimal = 0
	if left, ok := challenge.TourAvoiding(p.board, p.at, p.targets, maxJourney, p.avoid); ok {
		p.optimal = p.steps + left
	}
}

// relocateStar moves target i to a square the piece can reach, preferring one a
// couple of moves away and never one that is already taken. It reports false
// when the piece has nowhere to go.
func (p *PlayScene) relocateStar(ctx *Context, i int) bool {
	var open, far []challenge.Step
	if p.capture {
		open = p.takeableSquares(i)
	} else {
		for _, s := range challenge.ReachAvoiding(p.board, p.at, maxJourney, p.avoid) {
			if !p.board.At(s.Square).IsEmpty() || indexOfSquare(p.targets, s.Square) >= 0 {
				continue
			}
			open = append(open, s)
		}
	}
	for _, s := range open {
		if s.Moves >= 2 {
			far = append(far, s)
		}
	}
	pool := open
	if len(far) > 0 {
		pool = far
	}
	if len(pool) == 0 {
		return false
	}
	to := pool[ctx.Rand.IntN(len(pool))].Square
	if p.capture {
		// The target is a real piece: lift the pawn and set it down again.
		enemy := p.board.At(p.targets[i])
		p.board.Set(p.targets[i], chess.Piece{})
		p.board.Set(to, enemy)
	}
	p.targets[i] = to
	return true
}

// takeableSquares lists the empty squares where pawn i could be put down and
// still be captured by the piece. It cannot just use Reach: a white pawn only
// captures onto a square that is occupied, so the squares it can take on are
// not squares it can "reach" while they are empty. Each square is tried with
// the pawn standing on it. Squares on a pawn's own ranks come first.
func (p *PlayScene) takeableSquares(i int) []challenge.Step {
	enemy := p.board.At(p.targets[i])
	var home, edge []challenge.Step
	for r := 0; r < p.board.Height(); r++ {
		for f := 0; f < p.board.Width(); f++ {
			sq := chess.Sq(f, r)
			if sq == p.at || !p.board.At(sq).IsEmpty() {
				continue
			}
			nb := p.board.Clone()
			nb.Set(p.targets[i], chess.Piece{})
			nb.Set(sq, enemy)
			d := challenge.MovesTo(nb, p.at, sq, maxJourney)
			if d <= 0 {
				continue
			}
			step := challenge.Step{Square: sq, Moves: d}
			if r >= 1 && r <= p.board.Height()-2 {
				home = append(home, step)
			} else {
				edge = append(edge, step)
			}
		}
	}
	// Pawns stay on ranks a pawn could really be on, if there is any such square.
	if len(home) > 0 {
		return home
	}
	return edge
}

func (p *PlayScene) collectStar(ctx *Context, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndLand)
	ctx.SFX.Play(sfx.SndCheer)
	ctx.SFX.Play(sfx.SndPop)

	// Solving it in the fewest moves gets a louder party: twice the confetti, a
	// chime, a gold glow on the sticker and a word the grown-up can read out.
	// Wandering still earns the same sticker, so there is nothing to lose by
	// exploring - only something extra to win by looking first.
	perfect := p.optimal > 0 && p.steps == p.optimal
	if perfect {
		ctx.SFX.Play(sfx.SndMilestone)
	}

	cr := m.CellRect(int(p.at.File), int(p.at.Rank))
	cx, cy := cr.Center()
	p.kit.win(ctx, m, cx, cy, perfect)
	p.popSq = p.at
	p.starPopT = 0.3
	p.pieceSelected = false
}

// --- drawing -----------------------------------------------------------------

func (p *PlayScene) Draw(dst *ebiten.Image, ctx *Context) {
	m := ctx.M
	render.DrawBackground(dst, m)

	p.kit.drawHeader(dst, m, render.PieceName(p.pieceType))
	render.DrawBoard(dst, m)

	if p.safe {
		render.DrawDangerTint(dst, m, p.hot, p.dangerStrength())
	}

	// Hints and the pick-me halo.
	if p.idle() {
		if p.pieceSelected {
			render.DrawMoveHints(dst, m, p.hints(), p.hintFade())
			render.DrawPickableRing(dst, m, p.at, p.starScale, true, ctx.T)
		} else {
			render.DrawPickableRing(dst, m, p.at, p.starScale, false, ctx.T)
		}
	}

	if p.wobbleT > 0 {
		render.DrawSquareTint(dst, m, p.wobbleSq, 0, 0, render.Alpha(render.ColorShadow, 0.5))
	}

	p.drawStars(dst, ctx, m)

	// Every other piece on the board (pawn decoys).
	for _, sq := range p.board.Occupied() {
		if sq == p.at {
			continue
		}
		cr := m.CellRect(int(sq.File), int(sq.Rank))
		if p.safe && sq == p.guard && p.lungeT > 0 {
			cr = p.lunged(cr, m)
		}
		render.DrawPiece(dst, p.board.At(sq), cr, 0, true)
	}

	if p.trailFade > 0 {
		render.DrawMoveTrail(dst, m, p.trailPts[:p.trailN], p.trailFade)
	}

	p.drawPiece(dst, m)
	p.kit.drawOverlays(dst, ctx, m)
}

// dangerStrength is how strongly the red squares are showing. For the first few
// rounds they are always there; after that they come with the move dots, and
// they flash up whenever a move onto one has just been refused.
func (p *PlayScene) dangerStrength() float64 {
	s := 0.0
	if p.safeRound <= safeTeachRounds {
		s = 1
	} else {
		s = p.hintFade()
	}
	if p.flashT > 0 {
		pulse := 0.75 + 0.25*math.Sin(p.flashT*18)
		if pulse > s {
			s = pulse
		}
	}
	return s
}

// lunged shifts the guard's rectangle towards the square it is lunging at.
func (p *PlayScene) lunged(cr layout.Rect, m layout.Metrics) layout.Rect {
	to := m.CellRect(int(p.lungeTo.File), int(p.lungeTo.Rank))
	dx, dy := to.X-cr.X, to.Y-cr.Y
	if d := math.Hypot(dx, dy); d > 0 {
		k := math.Sin(math.Pi*(1-p.lungeT/safeLungeDur)) * m.Cell * 0.22 / d
		cr.X += dx * k
		cr.Y += dy * k
	}
	return cr
}

// idle is true while the board is waiting for a tap: no hop in flight and no
// celebration or fade running.
func (p *PlayScene) idle() bool {
	return p.state == stateIdle && !p.kit.busy()
}

// hints describes each legal destination for the renderer, reusing one buffer
// so drawing allocates nothing.
func (p *PlayScene) hints() []render.Hint {
	p.hintBuf = p.hintBuf[:0]
	for _, sq := range p.solutions {
		if p.safe && sq == p.guard {
			continue // the guard cannot be taken, so it is not offered
		}
		p.hintBuf = append(p.hintBuf, render.Hint{
			Square:  sq,
			Capture: !p.board.At(sq).IsEmpty(),
			Target:  !p.capture && indexOfSquare(p.targets, sq) >= 0,
		})
	}
	return p.hintBuf
}

// drawStars draws every star still on the board, and the last one flashing
// bigger and fading as it is collected.
func (p *PlayScene) drawStars(dst *ebiten.Image, ctx *Context, m layout.Metrics) {
	if p.capture {
		return // the pawns on the board are the targets; no stars
	}
	for _, t := range p.targets {
		render.DrawStar(dst, t, m, p.starScale, ctx.T)
	}
	if p.starPopT > 0 {
		t := 1 - p.starPopT/0.3
		render.DrawStar(dst, p.popSq, m, p.starScale*(1+t*1.4), ctx.T)
	}
}

func (p *PlayScene) drawPiece(dst *ebiten.Image, m layout.Metrics) {
	wx := 0.0
	if p.wobbleT > 0 {
		wx, _ = render.WobbleOffset(1-p.wobbleT/0.25, p.wobbleAmp)
	}
	cr := layout.Rect{X: p.pieceX - m.Cell/2, Y: p.pieceY - m.Cell/2, W: m.Cell, H: m.Cell}
	cr.X += wx
	lift := 0.0
	if p.pieceSelected && p.idle() {
		lift = m.Cell * 0.12
	}
	if p.state == stateMoving {
		lift = m.Cell * 0.05
	}
	render.DrawPiece(dst, p.cur.Piece, cr, lift, true)
	// The dark-square bishop wears a scarf, the same way the wooden set at home
	// has one with a scarf and one without. A bishop can never change square
	// colour, so it goes on when the puzzle is dealt and stays on for the whole
	// journey - including mid-hop, when p.at is still the square it left.
	if p.cur.Piece.Type == chess.Bishop && render.DarkSquare(p.at) {
		render.DrawBishopScarf(dst, cr, lift)
	}
}

// --- small helpers -----------------------------------------------------------

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func containsSquare(list []chess.Square, sq chess.Square) bool {
	for _, s := range list {
		if s == sq {
			return true
		}
	}
	return false
}

// indexOfSquare finds sq in list, or -1.
func indexOfSquare(list []chess.Square, sq chess.Square) int {
	for i, s := range list {
		if s == sq {
			return i
		}
	}
	return -1
}
