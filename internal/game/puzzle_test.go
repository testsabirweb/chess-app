package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/puzzles"
)

func newTestPuzzleScene(g *Game, p puzzles.Puzzle) *PuzzleScene {
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	s := NewPuzzleScene(g)
	s.loadPuzzle(p)
	return s
}

// press taps a cell and lets one frame run, so the scene has seen the tap.
func press(t *testing.T, s *PuzzleScene, g *Game, sq chess.Square) {
	t.Helper()
	tapCell(g, g.ctx.M, int(sq.File), int(sq.Rank))
	if err := s.Update(&g.ctx); err != nil {
		t.Fatal(err)
	}
	g.pointer.JustPressed = nil
}

// settle runs until the scene is waiting for a tap again and no reward is on.
func settlePuzzle(t *testing.T, s *PuzzleScene, g *Game) {
	t.Helper()
	for i := 0; i < 1500; i++ {
		if s.state == puzzleIdle && !s.kit.busy() {
			return
		}
		if err := s.Update(&g.ctx); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("puzzle scene never settled (state=%v busy=%v)", s.state, s.kit.busy())
}

func samePosition(a, b *chess.Board) bool {
	for r := 0; r < a.Height(); r++ {
		for f := 0; f < a.Width(); f++ {
			if a.At(chess.Sq(f, r)) != b.At(chess.Sq(f, r)) {
				return false
			}
		}
	}
	return true
}

func playMove(t *testing.T, s *PuzzleScene, g *Game, m chess.Move) {
	t.Helper()
	press(t, s, g, m.From)
	press(t, s, g, m.To)
	settlePuzzle(t, s, g)
}

// Each tier's stored solution must win, and mate in two needs the child's
// second move after black's reply.
func TestPuzzleTiersCanBeSolved(t *testing.T) {
	byTier := puzzles.ByTier()
	for _, tier := range puzzles.Tiers {
		p := byTier[tier][0]
		t.Run(string(tier), func(t *testing.T) {
			g := testGame()
			s := newTestPuzzleScene(g, p)
			playMove(t, s, g, p.Solution)
			if tier == puzzles.Mate2 {
				if s.stage != 2 {
					t.Fatalf("stage = %d after the first move, want 2", s.stage)
				}
				mates := puzzles.MateMoves(s.board, chess.White)
				if len(mates) == 0 {
					t.Fatal("no mate after black's reply")
				}
				playMove(t, s, g, mates[0])
			}
			if got := len(g.Stickers()); got != 1 {
				t.Fatalf("got %d stickers, want exactly 1", got)
			}
		})
	}
}

// A move that is not even a check is taken back, and costs nothing but a miss.
func TestPuzzleWrongMoveIsTakenBack(t *testing.T) {
	p := puzzles.ByTier()[puzzles.Check][0]
	g := testGame()
	s := newTestPuzzleScene(g, p)

	var quiet *chess.Move
	for _, m := range s.board.LegalMovesFor(chess.White) {
		if !s.board.After(m).InCheck(chess.Black) {
			mv := m
			quiet = &mv
			break
		}
	}
	if quiet == nil {
		t.Fatal("no quiet move in this position to try")
	}
	playMove(t, s, g, *quiet)
	if s.wrongs != 1 {
		t.Fatalf("wrongs = %d, want 1", s.wrongs)
	}
	if !samePosition(s.board, p.Board) {
		t.Fatal("the wrong move was not taken back")
	}
	if len(g.Stickers()) != 0 {
		t.Fatal("a wrong move won a sticker")
	}
	if s.kingPulseT <= 0 {
		t.Fatal("the king was not pointed at after a move that was not a check")
	}
}

// A check that is not mate plays out: the king escapes to a square that glows,
// then the board goes back to how it was.
func TestPuzzleTrapCheckShowsTheEscape(t *testing.T) {
	p := puzzles.ByTier()[puzzles.Mate1Trap][0]
	g := testGame()
	s := newTestPuzzleScene(g, p)

	var trap *chess.Move
	for _, m := range puzzles.CheckMoves(s.board, chess.White) {
		if m.From != p.Solution.From || m.To != p.Solution.To {
			mv := m
			trap = &mv
			break
		}
	}
	if trap == nil {
		t.Fatal("no trap check in the first trap puzzle")
	}
	playMove(t, s, g, *trap)
	if s.wrongs != 1 {
		t.Fatalf("wrongs = %d, want 1", s.wrongs)
	}
	if !samePosition(s.board, p.Board) {
		t.Fatal("the trap line was not taken back")
	}
	if len(g.Stickers()) != 0 {
		t.Fatal("the trap check won a sticker")
	}
}

// A first move in a mate in two that lets black slip out is shown and taken
// back too.
func TestPuzzleMateTwoWrongFirstMove(t *testing.T) {
	p := puzzles.ByTier()[puzzles.Mate2][0]
	g := testGame()
	s := newTestPuzzleScene(g, p)

	var quiet *chess.Move
	for _, m := range s.board.LegalMovesFor(chess.White) {
		if !s.board.After(m).InCheck(chess.Black) {
			mv := m
			quiet = &mv
			break
		}
	}
	if quiet == nil {
		t.Skip("no quiet first move in this position")
	}
	playMove(t, s, g, *quiet)
	if s.wrongs != 1 || s.stage != 1 {
		t.Fatalf("wrongs=%d stage=%d, want 1 and 1", s.wrongs, s.stage)
	}
	if !samePosition(s.board, p.Board) {
		t.Fatal("the first move was not taken back")
	}
}

// Three clean puzzles move him up a tier; three rough ones move him back down.
func TestPuzzleTierProgression(t *testing.T) {
	g := testGame()
	s := NewPuzzleScene(g)
	s.tier = puzzles.Check
	for i := 0; i < puzzleTierUp; i++ {
		s.noteResult(0)
	}
	if s.tier != puzzles.Mate1 {
		t.Fatalf("after %d clean puzzles tier = %s, want m1", puzzleTierUp, s.tier)
	}
	// One or two misses is a normal round and must not count as rough.
	s.noteResult(1)
	s.noteResult(2)
	if s.rough != 0 {
		t.Fatalf("a round with one or two misses counted as rough")
	}
	for i := 0; i < puzzleTierDown; i++ {
		s.noteResult(3)
	}
	if s.tier != puzzles.Check {
		t.Fatalf("after %d rough puzzles tier = %s, want check", puzzleTierDown, s.tier)
	}
	// It never leaves the ends of the list.
	s.tier = puzzles.Check
	for i := 0; i < 10; i++ {
		s.noteResult(3)
	}
	if s.tier != puzzles.Check {
		t.Fatalf("tier went below the first tier: %s", s.tier)
	}
	s.tier = puzzles.Mate2
	for i := 0; i < 10; i++ {
		s.noteResult(0)
	}
	if s.tier != puzzles.Mate2 {
		t.Fatalf("tier went past the last tier: %s", s.tier)
	}
}

// The help appears after two misses and names a move that really solves it.
func TestPuzzleHelpAfterMisses(t *testing.T) {
	p := puzzles.ByTier()[puzzles.Mate1][0]
	g := testGame()
	s := newTestPuzzleScene(g, p)
	if _, _, ok := s.answer(); !ok {
		t.Fatal("no answer for a mate in one")
	}
	dst := ebiten.NewImage(1080, 2400)
	s.wrongs = 2
	s.drawHelp(dst, g.ctx.M)
	s.wrongs = 3
	s.drawHelp(dst, g.ctx.M)
}

// A bag never deals the same puzzle twice before the tier is used up.
func TestPuzzleBagHasNoRepeats(t *testing.T) {
	g := testGame()
	s := NewPuzzleScene(g)
	s.tier = puzzles.Check
	s.bags = map[puzzles.Tier][]puzzles.Puzzle{}
	seen := map[string]bool{}
	for i := 0; i < len(puzzles.ByTier()[puzzles.Check]); i++ {
		p := s.deal()
		key := puzzles.PositionKey(p.Board)
		if seen[key] {
			t.Fatalf("puzzle %d repeated before the bag was empty", i)
		}
		seen[key] = true
	}
}

// forcedCheck reports whether white's move m gives check and every black reply
// allows a mate. escaping is true when at least one reply slips out.
func forcedCheck(b *chess.Board, m chess.Move) (forced, escaping bool) {
	after := b.After(m)
	if !after.InCheck(chess.Black) || after.Checkmate(chess.Black) {
		return false, false
	}
	for _, r := range after.LegalMovesFor(chess.Black) {
		if len(puzzles.MateMoves(after.After(r), chess.White)) == 0 {
			return false, true
		}
	}
	return true, false
}

func findMateTwo(t *testing.T, want func(*chess.Board, chess.Move) bool) (puzzles.Puzzle, chess.Move, bool) {
	t.Helper()
	for _, p := range puzzles.ByTier()[puzzles.Mate2] {
		for _, m := range p.Board.LegalMovesFor(chess.White) {
			if m.From == p.Solution.From && m.To == p.Solution.To {
				continue
			}
			if want(p.Board, m) {
				return p, m, true
			}
		}
	}
	return puzzles.Puzzle{}, chess.Move{}, false
}

// Any first move that checks and leaves black no escape is right, not just the
// one the bank happens to record. The alternative path must reach stage two.
func TestPuzzleMateTwoAcceptsAnotherForcedFirstMove(t *testing.T) {
	p, m, ok := findMateTwo(t, func(b *chess.Board, m chess.Move) bool {
		forced, _ := forcedCheck(b, m)
		return forced
	})
	if !ok {
		t.Skip("no mate-in-two puzzle in the bank has a second forced first move")
	}
	g := testGame()
	s := newTestPuzzleScene(g, p)
	playMove(t, s, g, m)
	if s.stage != 2 {
		t.Fatalf("stage = %d after a forced first move, want 2", s.stage)
	}
	mates := puzzles.MateMoves(s.board, chess.White)
	if len(mates) == 0 {
		t.Fatal("no mate after black's reply")
	}
	playMove(t, s, g, mates[0])
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("got %d stickers, want 1", got)
	}
}

// A check that lets the king step out must not reach stage two.
func TestPuzzleMateTwoRejectsAnEscapingCheck(t *testing.T) {
	p, m, ok := findMateTwo(t, func(b *chess.Board, m chess.Move) bool {
		_, escaping := forcedCheck(b, m)
		return escaping
	})
	if !ok {
		t.Skip("no mate-in-two puzzle in the bank has an escaping check")
	}
	g := testGame()
	s := newTestPuzzleScene(g, p)
	playMove(t, s, g, m)
	if s.stage != 1 || s.wrongs != 1 {
		t.Fatalf("stage=%d wrongs=%d, want 1 and 1", s.stage, s.wrongs)
	}
	if !samePosition(s.board, p.Board) {
		t.Fatal("the escaping check was not taken back")
	}
}
