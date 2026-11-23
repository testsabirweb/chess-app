package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/anim"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/puzzles"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/sfx"
)

// The puzzle game asks the question every game so far has avoided: can you
// checkmate? It starts with "find a check", then mate in one, then mate in one
// with a trap check, then mate in two. A wrong move is always taken back, with
// the reason shown, and never scored as a loss.

const (
	// puzzleShowDur is how long a failed line stays on the board before it is
	// taken back, long enough to see what happened.
	puzzleShowDur = 1.4
	// puzzleEscapeGlow is the green square shown for the king's way out.
	puzzleFlashDur = 1.0
	// puzzleTierUp is how many clean solves in a row move him up a tier, and
	// puzzleTierDown how many rough puzzles in a row move him back down.
	puzzleTierUp   = 3
	puzzleTierDown = 3
)

type puzzleState int

const (
	puzzleIdle puzzleState = iota
	puzzleAnimating
	puzzleShowing
)

type PuzzleScene struct {
	game *Game
	kit  roundKit

	cur   puzzles.Puzzle
	board *chess.Board
	// base is the position a failed attempt goes back to.
	base  *chess.Board
	tier  puzzles.Tier
	stage int // 1, or 2 once black has replied in a mate in two

	selected  chess.Square
	hasSel    bool
	hintT     float64
	solutions []chess.Move

	state puzzleState
	hop   hopper
	hopTo chess.Square
	hopPc chess.Piece
	after func(*Context, layout.Metrics)

	showT    float64
	escape   chess.Square
	escapeOK bool

	wrongs int
	// clean and rough count whole rounds, for moving between tiers.
	clean, rough int

	// bags hold the puzzles of each tier still to deal, so none repeats until
	// its tier is used up.
	bags map[puzzles.Tier][]puzzles.Puzzle

	ringPulse anim.Pulse
	ringScale float64

	hintBuf []render.Hint

	wobbleSq  chess.Square
	wobbleT   float64
	wobbleAmp float64

	checkT   float64
	checkSq  chess.Square
	mateT    float64
	mateFrom chess.Square
	mateSq   chess.Square
	fadeT    float64
}

func NewPuzzleScene(g *Game) *PuzzleScene {
	s := &PuzzleScene{
		game:      g,
		kit:       newRoundKit(g),
		tier:      puzzles.Check,
		ringPulse: anim.Pulse{Period: 1.6},
		bags:      map[puzzles.Tier][]puzzles.Puzzle{},
	}
	s.newPuzzle()
	return s
}

// loadPuzzle sets up one puzzle. It is what newPuzzle does once it has picked
// one, split out so tests can use a known position.
func (s *PuzzleScene) loadPuzzle(p puzzles.Puzzle) {
	s.cur = p
	s.tier = p.Tier
	s.board = p.Board.Clone()
	s.base = p.Board.Clone()
	s.stage = 1
	s.wrongs = 0
	s.hasSel = false
	s.selected = chess.Square{}
	s.hintT = 0
	s.state = puzzleIdle
	s.showT = 0
	s.checkT = 0
	s.mateT = 0
	s.fadeT = 0
	s.hopTo = chess.Square{}
	s.after = nil
}

func (s *PuzzleScene) newPuzzle() {
	s.loadPuzzle(s.deal())
}

// deal takes the next puzzle of the current tier from its bag, refilling the
// bag (shuffled) when it is empty.
func (s *PuzzleScene) deal() puzzles.Puzzle {
	byTier := puzzles.ByTier()
	if len(byTier[s.tier]) == 0 {
		s.tier = puzzles.Check
	}
	bag := s.bags[s.tier]
	if len(bag) == 0 {
		bag = append([]puzzles.Puzzle{}, byTier[s.tier]...)
		s.game.ctx.Rand.Shuffle(len(bag), func(i, j int) { bag[i], bag[j] = bag[j], bag[i] })
	}
	p := bag[len(bag)-1]
	s.bags[s.tier] = bag[:len(bag)-1]
	return p
}

func (s *PuzzleScene) Update(ctx *Context) error {
	m := ctx.M
	s.ringScale = s.ringPulse.Update(ctx.DT)
	if s.wobbleT > 0 {
		s.wobbleT -= ctx.DT
	}
	if s.hintT > 0 {
		s.hintT -= ctx.DT
	}
	if s.checkT > 0 {
		s.checkT -= ctx.DT
	}
	if s.mateT > 0 {
		s.mateT -= ctx.DT
	}
	if s.fadeT > 0 {
		s.fadeT -= ctx.DT
	}

	leave, deal := s.kit.update(ctx, m)
	if leave {
		ctx.Switch(NewHomeScene(s.game))
		return nil
	}
	if deal {
		s.newPuzzle()
	}
	if s.kit.busy() {
		return nil
	}

	switch s.state {
	case puzzleIdle:
		for _, ev := range ctx.Pointer.Pressed() {
			s.handleTap(ctx, ev.X, ev.Y, m)
		}
	case puzzleAnimating:
		if s.hop.update(ctx.DT, m.Cell) {
			s.state = puzzleIdle
			f := s.after
			s.after = nil
			if f != nil {
				f(ctx, m)
			}
		}
	case puzzleShowing:
		s.showT -= ctx.DT
		if s.showT <= 0 {
			s.revert()
		}
	}
	return nil
}

func (s *PuzzleScene) handleTap(ctx *Context, x, y float64, m layout.Metrics) {
	f, r, ok := m.HitCell(x, y)
	if !ok {
		return
	}
	sq := chess.Sq(f, r)

	if s.hasSel {
		switch {
		case sq == s.selected:
			ctx.SFX.Play(sfx.SndButton)
			s.hasSel = false
		case s.board.At(sq).Color == chess.White && !s.board.At(sq).IsEmpty():
			s.pickUp(ctx, sq)
		case s.canMoveTo(sq):
			s.play(ctx, m, sq)
		default:
			s.oops(ctx, sq, m)
		}
		return
	}
	p := s.board.At(sq)
	if !p.IsEmpty() && p.Color == chess.White {
		s.pickUp(ctx, sq)
		return
	}
	if !p.IsEmpty() {
		// Black's pieces cannot be picked up, but a tap on one is a guess, not
		// a mistake worth a buzz.
		s.oops(ctx, sq, m)
	}
}

func (s *PuzzleScene) pickUp(ctx *Context, sq chess.Square) {
	ctx.SFX.Play(sfx.SndButton)
	s.hasSel = true
	s.selected = sq
	s.hintT = s.game.hintDelay
	s.solutions = s.board.LegalMoves(sq)
}

func (s *PuzzleScene) canMoveTo(sq chess.Square) bool {
	for _, mv := range s.solutions {
		if mv.To == sq {
			return true
		}
	}
	return false
}

func (s *PuzzleScene) oops(ctx *Context, sq chess.Square, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndOops)
	s.wobbleSq = sq
	s.wobbleT = 0.25
	s.wobbleAmp = m.Cell * 0.025
}

// play starts the player's move and evaluates it when the piece lands.
func (s *PuzzleScene) play(ctx *Context, m layout.Metrics, to chess.Square) {
	from := s.selected
	pc := s.board.At(from)
	s.base = s.board.Clone()
	s.hasSel = false
	s.hintT = 0
	s.captureSound(ctx, to)
	s.startHop(ctx, m, from, to, pc, func(ctx *Context, m layout.Metrics) {
		s.judge(ctx, m)
	})
}

func (s *PuzzleScene) captureSound(ctx *Context, to chess.Square) {
	if !s.board.At(to).IsEmpty() {
		ctx.SFX.Play(sfx.SndPop)
	} else {
		ctx.SFX.Play(sfx.SndStep)
	}
}

// startHop moves the piece on the board at once and animates a copy of it from
// the old square to the new one.
func (s *PuzzleScene) startHop(ctx *Context, m layout.Metrics, from, to chess.Square, pc chess.Piece, after func(*Context, layout.Metrics)) {
	s.board.Set(from, chess.Piece{})
	s.board.Set(to, pc)
	s.hop.start(from, to, m, pc.Type == chess.Knight)
	s.hopTo = to
	s.hopPc = pc
	s.after = after
	s.state = puzzleAnimating
}

// judge looks at the position after the player's move and decides what to say.
func (s *PuzzleScene) judge(ctx *Context, m layout.Metrics) {
	black := chess.Black
	if s.board.Checkmate(black) {
		s.win(ctx, m)
		return
	}

	switch s.tier {
	case puzzles.Check:
		if s.board.InCheck(black) {
			s.sayCheck()
			s.win(ctx, m)
			return
		}
		s.fail(ctx, m, chess.Square{}, false)
	case puzzles.Mate1, puzzles.Mate1Trap:
		if s.board.InCheck(black) {
			s.sayCheck()
			s.showEscape(ctx, m)
			return
		}
		s.fail(ctx, m, chess.Square{}, false)
	case puzzles.Mate2:
		if s.stage == 2 {
			s.fail(ctx, m, chess.Square{}, false)
			return
		}
		if !s.board.InCheck(black) {
			s.fail(ctx, m, chess.Square{}, false)
			return
		}
		replies := s.board.LegalMovesFor(black)
		good, bad := -1, -1
		for i, r := range replies {
			if len(puzzles.MateMoves(s.board.After(r), chess.White)) > 0 {
				if good < 0 {
					good = i
				}
			} else if bad < 0 {
				bad = i
			}
		}
		if bad >= 0 {
			// One reply slips out: play it and show where the king went.
			r := replies[bad]
			s.startHop(ctx, m, r.From, r.To, s.board.At(r.From), func(ctx *Context, m layout.Metrics) {
				s.showEscapeAt(ctx, m, r.To, true)
			})
			return
		}
		if good < 0 {
			s.fail(ctx, m, chess.Square{}, false)
			return
		}
		// The first move is right: black replies, then it is his turn to mate.
		r := replies[good]
		if n := len(replies); n > 1 {
			r = replies[ctx.Rand.IntN(n)]
		}
		s.startHop(ctx, m, r.From, r.To, s.board.At(r.From), func(ctx *Context, m layout.Metrics) {
			s.stage = 2
			s.base = s.board.Clone()
			s.sayCheck()
		})
	}
}

func (s *PuzzleScene) sayCheck() {
	sq, ok := s.board.KingSquare(chess.Black)
	if ok {
		s.checkSq = sq
		s.checkT = puzzleFlashDur
	}
}

// showEscape plays what happens after a check that is not mate: the king steps
// away, and the square it reaches glows green.
func (s *PuzzleScene) showEscape(ctx *Context, m layout.Metrics) {
	replies := s.board.LegalMovesFor(chess.Black)
	if len(replies) == 0 {
		s.fail(ctx, m, chess.Square{}, false)
		return
	}
	r := replies[ctx.Rand.IntN(len(replies))]
	s.startHop(ctx, m, r.From, r.To, s.board.At(r.From), func(ctx *Context, m layout.Metrics) {
		s.showEscapeAt(ctx, m, r.To, true)
	})
}

func (s *PuzzleScene) showEscapeAt(ctx *Context, m layout.Metrics, sq chess.Square, green bool) {
	s.escape = sq
	s.escapeOK = green
	s.showT = puzzleShowDur
	s.wrongs++
	s.state = puzzleShowing
}

// fail takes a move back after a short pause, with a wobble on the square.
func (s *PuzzleScene) fail(ctx *Context, m layout.Metrics, sq chess.Square, green bool) {
	s.wrongs++
	s.escape = sq
	s.escapeOK = green
	s.showT = puzzleShowDur / 2
	s.state = puzzleShowing
	if !sqIsZero(sq) {
		s.oops(ctx, sq, m)
	}
}

func (s *PuzzleScene) revert() {
	s.board = s.base.Clone()
	s.hasSel = false
	s.selected = chess.Square{}
	s.hintT = 0
	s.state = puzzleIdle
	s.fadeT = 0.22
}

func (s *PuzzleScene) win(ctx *Context, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndLand)
	ctx.SFX.Play(sfx.SndCheer)
	ctx.SFX.Play(sfx.SndPop)
	perfect := s.wrongs == 0
	if perfect {
		ctx.SFX.Play(sfx.SndMilestone)
	}
	if sq, ok := s.board.KingSquare(chess.Black); ok {
		s.mateSq = sq
	}
	s.mateFrom = s.hopTo
	s.mateT = 1.2

	s.noteResult(s.wrongs)
	cx, cy := m.CellRect(int(s.mateSq.File), int(s.mateSq.Rank)).Center()
	s.kit.win(ctx, m, cx, cy, perfect)
	s.hasSel = false
}

// noteResult moves the tier after a run of clean or rough puzzles.
func (s *PuzzleScene) noteResult(wrongs int) {
	switch {
	case wrongs == 0:
		s.clean++
		s.rough = 0
	case wrongs >= 2:
		s.rough++
		s.clean = 0
	default:
		s.clean, s.rough = 0, 0
	}
	if s.clean >= puzzleTierUp {
		s.tier = nextTier(s.tier)
		s.clean, s.rough = 0, 0
	}
	if s.rough >= puzzleTierDown {
		s.tier = prevTier(s.tier)
		s.clean, s.rough = 0, 0
	}
}

func nextTier(t puzzles.Tier) puzzles.Tier {
	for i, tier := range puzzles.Tiers {
		if tier == t && i+1 < len(puzzles.Tiers) {
			return puzzles.Tiers[i+1]
		}
	}
	return t
}

func prevTier(t puzzles.Tier) puzzles.Tier {
	for i, tier := range puzzles.Tiers {
		if tier == t && i > 0 {
			return puzzles.Tiers[i-1]
		}
	}
	return t
}

func tierTitle(t puzzles.Tier) string {
	switch t {
	case puzzles.Check:
		return "Find a check"
	case puzzles.Mate1, puzzles.Mate1Trap:
		return "Mate in 1"
	case puzzles.Mate2:
		return "Mate in 2"
	}
	return "Puzzles"
}

// --- drawing -----------------------------------------------------------------

func (s *PuzzleScene) Draw(dst *ebiten.Image, ctx *Context) {
	m := ctx.M
	render.DrawBackground(dst, m)

	s.kit.drawHeader(dst, m, tierTitle(s.tier))
	render.DrawBoard(dst, m)

	// The king to attack gets a soft halo, so "look at the king" is the first
	// thing the eye does.
	if ksq, ok := s.board.KingSquare(chess.Black); ok {
		cr := m.CellRect(int(ksq.File), int(ksq.Rank))
		cx, cy := cr.Center()
		render.DrawGlow(dst, cx, cy, m.Cell*0.55*s.ringScale, render.Alpha(render.ColorPickable, 0.45))
	}

	if s.state == puzzleShowing && s.escapeOK {
		render.DrawSquareTint(dst, m, s.escape, 0, 0, render.Alpha(render.ColorEscape, 0.65))
	}

	if s.state == puzzleIdle && !s.kit.busy() {
		if s.hasSel {
			render.DrawMoveHints(dst, m, s.hints(), s.hintFade())
			render.DrawPickableRing(dst, m, s.selected, s.ringScale, true, ctx.T)
		} else {
			for _, sq := range s.board.Occupied() {
				if s.board.At(sq).Color == chess.White {
					render.DrawPickableRing(dst, m, sq, s.ringScale, false, ctx.T)
				}
			}
		}
	}

	if s.wobbleT > 0 {
		render.DrawSquareTint(dst, m, s.wobbleSq, 0, 0, render.Alpha(render.ColorShadow, 0.5))
	}

	for _, sq := range s.board.Occupied() {
		if s.state == puzzleAnimating && sq == s.hopTo {
			continue // drawn in the air below
		}
		cr := m.CellRect(int(sq.File), int(sq.Rank))
		render.DrawPiece(dst, s.board.At(sq), cr, 0, true)
	}
	if s.state == puzzleAnimating {
		cr := layout.Rect{X: s.hop.x - m.Cell/2, Y: s.hop.y - m.Cell/2, W: m.Cell, H: m.Cell}
		render.DrawPiece(dst, s.hopPc, cr, m.Cell*0.05, true)
	}

	s.drawHelp(dst, m)

	if s.checkT > 0 {
		cr := m.CellRect(int(s.checkSq.File), int(s.checkSq.Rank))
		cx, cy := cr.Center()
		render.DrawTextShadowed(dst, "Check!", cx, cy-m.Cell*0.62, m.BodySize*0.95, render.ColorStarGlow)
	}
	if s.mateT > 0 {
		for _, sq := range []chess.Square{s.mateSq, s.mateFrom} {
			if sqIsZero(sq) {
				continue
			}
			cr := m.CellRect(int(sq.File), int(sq.Rank))
			cx, cy := cr.Center()
			render.DrawGlow(dst, cx, cy, m.Cell*0.75, render.Alpha(render.ColorStarGlow, 0.5))
		}
		render.DrawTextShadowed(dst, "Checkmate!", m.W/2, m.Header.Y+m.Header.H*0.30, render.FitTextSize("Checkmate!", m.BodySize*1.1, m.Header.W*0.8), render.ColorStarGlow)
	}

	if s.fadeT > 0 {
		render.DrawFilledRect(dst, 0, 0, m.W, m.H, render.Alpha(render.ColorBGTop, 0.6))
	}

	s.kit.drawOverlays(dst, ctx, m)
}

// drawHelp shows the answer piece after two misses and the square after three,
// so he always finishes on his own.
func (s *PuzzleScene) drawHelp(dst *ebiten.Image, m layout.Metrics) {
	if s.wrongs < 2 || s.kit.busy() || s.state != puzzleIdle {
		return
	}
	from, to, ok := s.answer()
	if !ok {
		return
	}
	cr := m.CellRect(int(from.File), int(from.Rank))
	cx, cy := cr.Center()
	render.DrawGlow(dst, cx, cy, m.Cell*0.55, render.Alpha(render.ColorStarGlow, 0.4))
	if s.wrongs >= 3 {
		tcr := m.CellRect(int(to.File), int(to.Rank))
		tx, ty := tcr.Center()
		render.DrawGlow(dst, tx, ty, m.Cell*0.5, render.Alpha(render.ColorHintDot, 0.5))
	}
}

// answer is a move that solves the current stage: a mate in a mating tier, a
// check in the check tier, and a forced first move in a mate in two.
func (s *PuzzleScene) answer() (from, to chess.Square, ok bool) {
	white, black := chess.White, chess.Black
	for _, m := range puzzles.MateMoves(s.board, white) {
		return m.From, m.To, true
	}
	if s.tier == puzzles.Check {
		for _, m := range puzzles.CheckMoves(s.board, white) {
			return m.From, m.To, true
		}
	}
	if s.tier == puzzles.Mate2 && s.stage == 1 {
		for _, m := range puzzles.CheckMoves(s.board, white) {
			after := s.board.After(m)
			forced := true
			for _, r := range after.LegalMovesFor(black) {
				if len(puzzles.MateMoves(after.After(r), white)) == 0 {
					forced = false
				}
			}
			if forced {
				return m.From, m.To, true
			}
		}
	}
	return chess.Square{}, chess.Square{}, false
}

func (s *PuzzleScene) hints() []render.Hint {
	s.hintBuf = s.hintBuf[:0]
	for _, mv := range s.solutions {
		s.hintBuf = append(s.hintBuf, render.Hint{
			Square:  mv.To,
			Capture: !s.board.At(mv.To).IsEmpty(),
		})
	}
	return s.hintBuf
}

// hintFade is how strongly the move dots are showing: nothing during the pause,
// ramping up over its last hintFadeIn seconds, exactly as in the star game.
func (s *PuzzleScene) hintFade() float64 {
	if !s.hasSel || s.state != puzzleIdle {
		return 0
	}
	if s.hintT <= 0 {
		return 1
	}
	if s.hintT >= hintFadeIn {
		return 0
	}
	return 1 - s.hintT/hintFadeIn
}

func sqIsZero(sq chess.Square) bool { return sq == chess.Square{} }

// --- hopper ------------------------------------------------------------------

// hopper animates one piece from square to square, with the same hop and the
// same L-shaped route for a knight as the star game. It is a smaller copy of
// the play scene's animation, kept separate so changing one cannot break the
// other.
type hopper struct {
	tween anim.Tween
	pts   [3][2]float64
	n     int
	leg   int
	hold  float64
	x, y  float64
	arc   float64
}

func (h *hopper) start(from, to chess.Square, m layout.Metrics, knight bool) {
	fc := m.CellRect(int(from.File), int(from.Rank))
	tc := m.CellRect(int(to.File), int(to.Rank))
	fx, fy := fc.Center()
	tx, ty := tc.Center()

	h.n, h.leg, h.hold = 2, 0, 0
	h.pts[0][0], h.pts[0][1] = fx, fy
	h.pts[1][0], h.pts[1][1] = tx, ty
	h.arc = 0.28
	h.tween.Duration = 0.42
	h.tween.Ease = anim.EaseInOutCubic
	if knight {
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
		cc := m.CellRect(int(corner.File), int(corner.Rank))
		cx, cy := cc.Center()
		h.pts[1][0], h.pts[1][1] = cx, cy
		h.pts[2][0], h.pts[2][1] = tx, ty
		h.n = 3
		h.tween.Duration = 0.30
		h.arc = 0.16
	}
	h.x, h.y = fx, fy
	h.tween.Start()
}

// update advances the hop; done is true on the frame the piece lands.
func (h *hopper) update(dt, cell float64) bool {
	if h.hold > 0 {
		h.hold -= dt
		end := h.leg + 1
		h.x, h.y = h.pts[end][0], h.pts[end][1]
		if h.hold <= 0 {
			h.leg++
			h.tween.Start()
		}
		return false
	}
	from, to := h.pts[h.leg], h.pts[h.leg+1]
	t := h.tween.Update(dt)
	arc := -math.Sin(t*math.Pi) * cell * h.arc
	h.x = from[0] + (to[0]-from[0])*t
	h.y = from[1] + (to[1]-from[1])*t + arc
	if !h.tween.Done() {
		return false
	}
	if h.leg+1 < h.n-1 {
		h.x, h.y = to[0], to[1]
		h.hold = 0.12
		return false
	}
	h.x, h.y = to[0], to[1]
	return true
}
