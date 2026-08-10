package challenge_test

import (
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/testsabirweb/chess-app/internal/challenge"
	"github.com/testsabirweb/chess-app/internal/chess"
)

func boardWith(p chess.Piece, at chess.Square) *chess.Board {
	b := chess.NewBoard(5, 5)
	b.Set(at, p)
	return b
}

func stepMap(steps []challenge.Step) map[chess.Square]int {
	out := map[chess.Square]int{}
	for _, s := range steps {
		out[s.Square] = s.Moves
	}
	return out
}

func TestReachRookCoversBoardInTwoMoves(t *testing.T) {
	b := boardWith(chess.Piece{Type: chess.Rook, Color: chess.White}, chess.Sq(0, 0))
	got := stepMap(challenge.Reach(b, chess.Sq(0, 0), 3))
	if len(got) != 24 {
		t.Fatalf("rook should reach the other 24 squares, got %d", len(got))
	}
	for sq, moves := range got {
		want := 2
		if sq.File == 0 || sq.Rank == 0 {
			want = 1
		}
		if moves != want {
			t.Errorf("rook to %+v: want %d moves, got %d", sq, want, moves)
		}
	}
}

func TestReachBishopStaysOnItsColour(t *testing.T) {
	from := chess.Sq(0, 0)
	b := boardWith(chess.Piece{Type: chess.Bishop, Color: chess.White}, from)
	steps := challenge.Reach(b, from, 4)
	if len(steps) == 0 {
		t.Fatal("bishop reached nothing")
	}
	for _, s := range steps {
		if (int(s.Square.File)+int(s.Square.Rank))%2 != 0 {
			t.Fatalf("bishop reached opposite-colour square %+v", s.Square)
		}
	}
	// Every same-colour square on a 5x5 is reachable in at most two moves.
	if len(steps) != 12 {
		t.Fatalf("want 12 same-colour squares, got %d", len(steps))
	}
}

func TestReachPawnWalksUpItsFile(t *testing.T) {
	from := chess.Sq(2, 0)
	b := boardWith(chess.Piece{Type: chess.Pawn, Color: chess.White}, from)
	got := stepMap(challenge.Reach(b, from, 3))
	want := map[chess.Square]int{
		chess.Sq(2, 1): 1,
		chess.Sq(2, 2): 2, // single push, then a double push off the home rank
		chess.Sq(2, 3): 2,
		chess.Sq(2, 4): 3,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pawn reach mismatch:\n got %v\nwant %v", got, want)
	}
}

func TestReachPawnCapturesChangeFile(t *testing.T) {
	from := chess.Sq(2, 0)
	b := boardWith(chess.Piece{Type: chess.Pawn, Color: chess.White}, from)
	b.Set(chess.Sq(3, 1), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	got := stepMap(challenge.Reach(b, from, 2))
	if got[chess.Sq(3, 1)] != 1 {
		t.Fatalf("pawn should capture onto (3,1) in one move: %v", got)
	}
	if _, ok := got[chess.Sq(3, 2)]; !ok {
		t.Fatalf("pawn should continue up the new file after capturing: %v", got)
	}
}

func TestReachIsDeterministic(t *testing.T) {
	from := chess.Sq(2, 2)
	b := boardWith(chess.Piece{Type: chess.Knight, Color: chess.White}, from)
	a := challenge.Reach(b, from, 3)
	c := challenge.Reach(b, from, 3)
	if !reflect.DeepEqual(a, c) {
		t.Fatal("Reach is not deterministic")
	}
}

func TestMovesToUnreachable(t *testing.T) {
	from := chess.Sq(0, 0)
	b := boardWith(chess.Piece{Type: chess.Bishop, Color: chess.White}, from)
	if got := challenge.MovesTo(b, from, chess.Sq(1, 0), 5); got != -1 {
		t.Fatalf("dark square must be unreachable for this bishop, got %d", got)
	}
	if got := challenge.MovesTo(b, from, from, 5); got != 0 {
		t.Fatalf("MovesTo(self) = %d, want 0", got)
	}
}

func multiMoveSpec() challenge.Spec {
	return challenge.Spec{
		BoardWidth: 5, BoardHeight: 5,
		Pieces: []chess.PieceType{
			chess.Pawn, chess.Knight, chess.Bishop,
			chess.Rook, chess.Queen, chess.King,
		},
		Color:    chess.White,
		MinMoves: 1,
		MaxMoves: 3,
		Decoys:   true,
	}
}

func TestMultiMoveTargetsAreAlwaysReachable(t *testing.T) {
	g := challenge.NewGenerator(multiMoveSpec(), rand.New(rand.NewPCG(11, 13)))
	far := 0
	const draws = 1500
	for i := 0; i < draws; i++ {
		c := g.Next()
		if c.From == c.Target {
			t.Fatalf("draw %d: from == target", i)
		}
		if c.Moves < 1 || c.Moves > 3 {
			t.Fatalf("draw %d: Moves=%d out of range", i, c.Moves)
		}
		if got := challenge.MovesTo(c.Board, c.From, c.Target, 3); got != c.Moves {
			t.Fatalf("draw %d: reported %d moves, board says %d", i, c.Moves, got)
		}
		if c.Moves > 1 {
			far++
		}
	}
	if far < draws/2 {
		t.Fatalf("expected most challenges to need more than one move, got %d/%d", far, draws)
	}
}

func TestMultiMoveIsDeterministicForSeed(t *testing.T) {
	a := challenge.NewGenerator(multiMoveSpec(), rand.New(rand.NewPCG(3, 4)))
	b := challenge.NewGenerator(multiMoveSpec(), rand.New(rand.NewPCG(3, 4)))
	for i := 0; i < 100; i++ {
		ca, cb := a.Next(), b.Next()
		if ca.From != cb.From || ca.Target != cb.Target || ca.Moves != cb.Moves {
			t.Fatalf("draw %d diverged", i)
		}
	}
}

func TestMultiMoveCoversWholeBoard(t *testing.T) {
	spec := multiMoveSpec()
	spec.Pieces = []chess.PieceType{chess.Knight}
	g := challenge.NewGenerator(spec, rand.New(rand.NewPCG(21, 22)))
	seen := map[chess.Square]int{}
	for i := 0; i < 3000; i++ {
		seen[g.Next().Target]++
	}
	if len(seen) != 25 {
		t.Fatalf("knight targets covered %d/25 squares", len(seen))
	}
}

func TestTargetIsNeverUnderAPiece(t *testing.T) {
	spec := multiMoveSpec()
	spec.Pieces = []chess.PieceType{chess.Pawn}
	g := challenge.NewGenerator(spec, rand.New(rand.NewPCG(31, 41)))
	for i := 0; i < 1200; i++ {
		c := g.Next()
		if !c.Board.At(c.Target).IsEmpty() {
			t.Fatalf("draw %d: star placed under a %v", i, c.Board.At(c.Target).Type)
		}
	}
}

// A pawn gets one double push, on its very first move. Starting a pawn behind
// its home rank used to hand it a second one: step onto the home rank, then
// jump two.
func TestPawnNeverGetsASecondDoublePush(t *testing.T) {
	spec := multiMoveSpec()
	spec.Pieces = []chess.PieceType{chess.Pawn}
	g := challenge.NewGenerator(spec, rand.New(rand.NewPCG(77, 79)))

	doubleFrom := func(b *chess.Board, at chess.Square) bool {
		for _, m := range b.Moves(nil, at) {
			if m.To.Rank-at.Rank == 2 || at.Rank-m.To.Rank == 2 {
				return true
			}
		}
		return false
	}

	for i := 0; i < 1500; i++ {
		c := g.Next()
		if c.From.Rank < 1 {
			t.Fatalf("draw %d: pawn started behind its home rank at %+v", i, c.From)
		}
		// Walk to every square the pawn can get to and check that none of them
		// still offers a two-square push.
		for _, step := range challenge.Reach(c.Board, c.From, 3) {
			b := c.Board.Clone()
			b.Set(c.From, chess.Piece{})
			b.Set(step.Square, c.Piece)
			if doubleFrom(b, step.Square) {
				t.Fatalf("draw %d: pawn from %+v could double push again at %+v",
					i, c.From, step.Square)
			}
		}
	}
}
