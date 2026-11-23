package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/anim"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/sfx"
)

// Pawn Wars is the first game played against a person: three pawns each, one
// phone between two players. The child plays white at the bottom; the grown-up
// plays black from the other side. It is the one place in the app where someone
// can lose, and that is on purpose: the opponent is a person who can choose to
// let him win, and losing to a grown-up is a different thing from the game
// telling you that you lost.

// pawnWarsFiles are the three files the pawns stand on: the middle three,
// leaving one empty file at either edge. Five pawns a side on a 5x5 board would
// be a wall with no room to play.
var pawnWarsFiles = []int{1, 2, 3}

// pawnWarsResult decides a game that has just had a move played on it.
//
//   - A pawn on the far rank wins for its colour: in this game reaching the end
//     is the win, so there is no promotion piece to choose.
//   - A side with no pawns left has lost.
//   - If the side whose turn it is can not move, nobody wins. This is the
//     stalemate rule, kept as a friendly draw: explaining stalemate to a small
//     child is not worth it, and losing to a rule they cannot see would be
//     worse.
func pawnWarsResult(b *chess.Board, mover chess.Color) (winner chess.Color, draw, over bool) {
	white, black := 0, 0
	for _, sq := range b.Occupied() {
		p := b.At(sq)
		if p.Type != chess.Pawn {
			continue
		}
		switch p.Color {
		case chess.White:
			white++
			if int(sq.Rank) == b.Height()-1 {
				return chess.White, false, true
			}
		case chess.Black:
			black++
			if int(sq.Rank) == 0 {
				return chess.Black, false, true
			}
		}
	}
	switch {
	case white == 0:
		return chess.Black, false, true
	case black == 0:
		return chess.White, false, true
	}
	if len(b.LegalMovesFor(mover.Opponent())) == 0 {
		return noWinner, true, true
	}
	return noWinner, false, false
}

// noWinner is what pawnWarsResult reports when the game is drawn. It is not a
// real colour: chess.Color only has White and Black, and a third value keeps
// "nobody" out of the way of both.
const noWinner chess.Color = 255

// pionWarsSetup places the starting position. It is a separate function so the
// tests can build the same board without a scene.
func pawnWarsSetup() *chess.Board {
	b := chess.NewBoard(5, 5)
	white := chess.Piece{Type: chess.Pawn, Color: chess.White}
	black := chess.Piece{Type: chess.Pawn, Color: chess.Black}
	for _, f := range pawnWarsFiles {
		b.Set(chess.Sq(f, 1), white)
		b.Set(chess.Sq(f, 3), black)
	}
	return b
}

const (
	// pawnWarsBlackWinShow is how long "Black wins!" stays up before the board
	// goes back to the start.
	pawnWarsBlackWinShow = 2.8
	// pawnWarsResetFade covers the board while it is set up again.
	pawnWarsResetFade = 0.25
)

type pawnWarsState int

const (
	pwIdle pawnWarsState = iota
	pwAnimating
	pwBlackWon
)

type PawnWarsScene struct {
	game *Game
	kit  roundKit

	board *chess.Board
	turn  chess.Color
	// banner overrides the turn line after a win or a draw, until the board is
	// set up again.
	banner string

	selected  chess.Square
	hasSel    bool
	hintT     float64
	solutions []chess.Move
	hintBuf   []render.Hint

	state     pawnWarsState
	hop       hopper
	hopTo     chess.Square
	hopPc     chess.Piece
	after     func()
	blackWinT float64
	fadeT     float64

	ringPulse anim.Pulse
	ringScale float64

	wobbleSq  chess.Square
	wobbleT   float64
	wobbleAmp float64
}

func NewPawnWarsScene(g *Game) *PawnWarsScene {
	s := &PawnWarsScene{
		game:      g,
		kit:       newRoundKit(g),
		ringPulse: anim.Pulse{Period: 1.6},
	}
	s.setUp()
	return s
}

// setUp starts a fresh game.
func (s *PawnWarsScene) setUp() {
	s.board = pawnWarsSetup()
	s.turn = chess.White
	s.banner = ""
	s.selected = chess.Square{}
	s.hasSel = false
	s.hintT = 0
	s.state = pwIdle
	s.blackWinT = 0
	s.hopTo = chess.Square{}
	s.after = nil
	s.fadeT = pawnWarsResetFade
}

func (s *PawnWarsScene) Update(ctx *Context) error {
	m := ctx.M
	s.ringScale = s.ringPulse.Update(ctx.DT)
	if s.wobbleT > 0 {
		s.wobbleT -= ctx.DT
	}
	if s.hintT > 0 {
		s.hintT -= ctx.DT
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
		s.setUp()
	}
	if s.kit.busy() {
		return nil
	}

	switch s.state {
	case pwBlackWon:
		s.blackWinT -= ctx.DT
		for range ctx.Pointer.Pressed() {
			s.blackWinT = 0
		}
		if s.blackWinT <= 0 {
			s.setUp()
		}
	case pwAnimating:
		if s.hop.update(ctx.DT, m.Cell) {
			s.state = pwIdle
			f := s.after
			s.after = nil
			if f != nil {
				f()
			}
		}
	case pwIdle:
		for _, ev := range ctx.Pointer.Pressed() {
			s.handleTap(ctx, ev.X, ev.Y, m)
		}
	}
	return nil
}

func (s *PawnWarsScene) handleTap(ctx *Context, x, y float64, m layout.Metrics) {
	f, r, ok := m.HitCell(x, y)
	if !ok {
		return
	}
	sq := chess.Sq(f, r)
	p := s.board.At(sq)

	if s.hasSel {
		switch {
		case sq == s.selected:
			ctx.SFX.Play(sfx.SndButton)
			s.hasSel = false
		case !p.IsEmpty() && p.Color == s.turn:
			s.pickUp(ctx, sq)
		case s.canMoveTo(sq):
			s.play(ctx, m, sq)
		default:
			s.oops(ctx, sq, m)
		}
		return
	}

	if p.IsEmpty() {
		return
	}
	if p.Color != s.turn {
		// The other player's pawn: an ordinary wobble, not an accusation.
		s.oops(ctx, sq, m)
		return
	}
	s.pickUp(ctx, sq)
}

func (s *PawnWarsScene) pickUp(ctx *Context, sq chess.Square) {
	ctx.SFX.Play(sfx.SndButton)
	s.hasSel = true
	s.selected = sq
	s.hintT = s.game.hintDelay
	s.solutions = s.board.LegalMoves(sq)
}

func (s *PawnWarsScene) canMoveTo(sq chess.Square) bool {
	for _, mv := range s.solutions {
		if mv.To == sq {
			return true
		}
	}
	return false
}

func (s *PawnWarsScene) oops(ctx *Context, sq chess.Square, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndOops)
	s.wobbleSq = sq
	s.wobbleT = 0.25
	s.wobbleAmp = m.Cell * 0.025
}

func (s *PawnWarsScene) play(ctx *Context, m layout.Metrics, to chess.Square) {
	from := s.selected
	pc := s.board.At(from)
	if !s.board.At(to).IsEmpty() {
		ctx.SFX.Play(sfx.SndPop)
	} else {
		ctx.SFX.Play(sfx.SndStep)
	}
	s.hasSel = false
	s.hintT = 0
	s.selected = chess.Square{}
	s.startHop(ctx, m, from, to, pc, func() {
		s.settle(ctx, m)
	})
}

func (s *PawnWarsScene) startHop(ctx *Context, m layout.Metrics, from, to chess.Square, pc chess.Piece, after func()) {
	s.board.Set(from, chess.Piece{})
	s.board.Set(to, pc)
	s.hop.start(from, to, m, false)
	s.hopTo = to
	s.hopPc = pc
	s.after = after
	s.state = pwAnimating
}

// settle decides the game after a move, or hands the turn over.
func (s *PawnWarsScene) settle(ctx *Context, m layout.Metrics) {
	winner, draw, over := pawnWarsResult(s.board, s.turn)
	if over {
		switch {
		case draw:
			s.banner = "Everyone wins!"
			ctx.SFX.Play(sfx.SndCheer)
			cx, cy := m.Board.Center()
			s.kit.win(ctx, m, cx, cy, false)
		case winner == chess.Black:
			s.blackWin(ctx, m)
		default:
			s.banner = "White wins!"
			ctx.SFX.Play(sfx.SndCheer)
			s.whiteWin(ctx, m)
		}
		return
	}
	s.turn = s.turn.Opponent()
}

func (s *PawnWarsScene) whiteWin(ctx *Context, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndLand)
	ctx.SFX.Play(sfx.SndPop)
	cx, cy := m.Board.Center()
	s.kit.win(ctx, m, cx, cy, false)
}

// blackWin is the grown-up's win: confetti over black's side, no sticker, and
// the same gentle way back to a new game.
func (s *PawnWarsScene) blackWin(ctx *Context, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndLand)
	ctx.SFX.Play(sfx.SndHop)
	s.banner = "Black wins!"
	cx := m.Board.X + m.Board.W/2
	cy := m.Board.Y + m.Board.H*0.22
	s.kit.confetti.Burst(ctx.Rand, cx, cy, 36, m.Cell)
	s.blackWinT = pawnWarsBlackWinShow
	s.state = pwBlackWon
}

// --- drawing -----------------------------------------------------------------

func (s *PawnWarsScene) Draw(dst *ebiten.Image, ctx *Context) {
	m := ctx.M
	render.DrawBackground(dst, m)

	s.kit.drawHeader(dst, m, s.title())
	render.DrawBoard(dst, m)

	// Whose turn it is, shown at that player's edge of the board: white at the
	// bottom, black at the top. The header says it too, for the grown-up.
	if s.state != pwBlackWon {
		// White's edge is the bottom of the board, black's the top.
		y := m.Board.Y + m.Board.H + m.Cell*0.18
		clr := render.ColorStarGlow
		if s.turn == chess.Black {
			y = m.Board.Y - m.Cell*0.18
			clr = render.ColorHintDot
		}
		render.DrawGlow(dst, m.Board.X+m.Board.W/2, y, m.Board.W*0.45, render.Alpha(clr, 0.35))
	}

	if s.state == pwIdle && !s.kit.busy() {
		if s.hasSel {
			render.DrawMoveHints(dst, m, s.hints(), s.hintFade())
			render.DrawPickableRing(dst, m, s.selected, s.ringScale, true, ctx.T)
		} else {
			for _, sq := range s.board.Occupied() {
				if s.board.At(sq).Color == s.turn {
					render.DrawPickableRing(dst, m, sq, s.ringScale, false, ctx.T)
				}
			}
		}
	}

	if s.wobbleT > 0 {
		render.DrawSquareTint(dst, m, s.wobbleSq, 0, 0, render.Alpha(render.ColorShadow, 0.5))
	}

	for _, sq := range s.board.Occupied() {
		if s.state == pwAnimating && sq == s.hopTo {
			continue
		}
		cr := m.CellRect(int(sq.File), int(sq.Rank))
		render.DrawPiece(dst, s.board.At(sq), cr, 0, true)
	}
	if s.state == pwAnimating {
		cr := layout.Rect{X: s.hop.x - m.Cell/2, Y: s.hop.y - m.Cell/2, W: m.Cell, H: m.Cell}
		render.DrawPiece(dst, s.hopPc, cr, m.Cell*0.05, true)
	}

	if s.fadeT > 0 {
		render.DrawFilledRect(dst, 0, 0, m.W, m.H, render.Alpha(render.ColorBGTop, 0.45))
	}

	s.kit.drawOverlays(dst, ctx, m)
}

func (s *PawnWarsScene) title() string {
	if s.banner != "" {
		return s.banner
	}
	if s.turn == chess.White {
		return "White's turn"
	}
	return "Black's turn"
}

func (s *PawnWarsScene) hints() []render.Hint {
	s.hintBuf = s.hintBuf[:0]
	for _, mv := range s.solutions {
		s.hintBuf = append(s.hintBuf, render.Hint{
			Square:  mv.To,
			Capture: !s.board.At(mv.To).IsEmpty(),
		})
	}
	return s.hintBuf
}

func (s *PawnWarsScene) hintFade() float64 {
	if !s.hasSel || s.state != pwIdle {
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
