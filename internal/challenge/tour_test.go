package challenge

import (
	"math/rand/v2"
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

func boardWith(pt chess.PieceType, at chess.Square) *chess.Board {
	b := chess.NewBoard(5, 5)
	b.Set(at, chess.Piece{Type: pt, Color: chess.White})
	return b
}

func TestTourPicksTheBestOrder(t *testing.T) {
	b := boardWith(chess.Rook, chess.Sq(0, 0))
	// (4,0) is on the rook's rank, (4,4) is two moves away. Going to (4,0)
	// first costs 1+1; going to (4,4) first costs 2+1.
	got, ok := Tour(b, chess.Sq(0, 0), []chess.Square{chess.Sq(4, 4), chess.Sq(4, 0)}, 3)
	if !ok || got != 2 {
		t.Fatalf("Tour = %d, %v, want 2, true", got, ok)
	}
}

func TestTourNoTargetsIsZero(t *testing.T) {
	b := boardWith(chess.Rook, chess.Sq(0, 0))
	if got, ok := Tour(b, chess.Sq(0, 0), nil, 3); !ok || got != 0 {
		t.Fatalf("Tour = %d, %v", got, ok)
	}
}

func TestTourUnreachable(t *testing.T) {
	// A pawn cannot go back down the board.
	b := boardWith(chess.Pawn, chess.Sq(2, 3))
	if _, ok := Tour(b, chess.Sq(2, 3), []chess.Square{chess.Sq(2, 1)}, 3); ok {
		t.Fatal("pawn should not reach a square behind it")
	}
}

// Capturing opens a line: the rook is blocked until the first pawn is taken.
func TestTourReplaysCaptures(t *testing.T) {
	b := boardWith(chess.Rook, chess.Sq(0, 0))
	b.Set(chess.Sq(0, 2), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	b.Set(chess.Sq(0, 4), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	got, ok := Tour(b, chess.Sq(0, 0), []chess.Square{chess.Sq(0, 2), chess.Sq(0, 4)}, 3)
	if !ok || got != 2 {
		t.Fatalf("Tour = %d, %v, want 2, true (take one, then the next)", got, ok)
	}
}

func TestTourDoesNotMutateTheBoard(t *testing.T) {
	b := boardWith(chess.Queen, chess.Sq(1, 1))
	before := b.Clone()
	Tour(b, chess.Sq(1, 1), []chess.Square{chess.Sq(4, 4), chess.Sq(0, 3)}, 3)
	for f := 0; f < 5; f++ {
		for r := 0; r < 5; r++ {
			if b.At(chess.Sq(f, r)) != before.At(chess.Sq(f, r)) {
				t.Fatalf("board changed at %d,%d", f, r)
			}
		}
	}
}

var allTypes = []chess.PieceType{chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King}

func checkTreasure(t *testing.T, p Puzzle, k int, pt chess.PieceType) {
	t.Helper()
	if len(p.Targets) < 1 || len(p.Targets) > k {
		t.Fatalf("%v: %d targets, want 1..%d", pt, len(p.Targets), k)
	}
	if p.Board.At(p.From) != p.Piece {
		t.Fatalf("piece not on its start square")
	}
	seen := map[chess.Square]bool{}
	for _, s := range p.Targets {
		if s == p.From || seen[s] || !p.Board.At(s).IsEmpty() {
			t.Fatalf("%v: bad target %v", pt, s)
		}
		seen[s] = true
	}
	total, ok := Tour(p.Board, p.From, p.Targets, treasureLeg)
	if !ok || total != p.Optimal {
		t.Fatalf("%v: Optimal=%d, Tour=%d,%v", pt, p.Optimal, total, ok)
	}
	if p.Optimal > treasureMaxTour {
		t.Fatalf("%v: tour of %d moves is too long", pt, p.Optimal)
	}
}

func TestTreasureIsAlwaysSolvable(t *testing.T) {
	for seed := uint64(0); seed < 60; seed++ {
		rng := rand.New(rand.NewPCG(seed, 11))
		for _, pt := range allTypes {
			for k := 2; k <= 3; k++ {
				checkTreasure(t, NewTreasure(rng, pt, chess.White, k, chess.Sq(0, 0)), k, pt)
			}
		}
	}
}

// Every piece but the pawn (which can only climb) should get the full number
// of stars most of the time.
func TestTreasureUsuallyHasAllTheStars(t *testing.T) {
	rng := rand.New(rand.NewPCG(2, 3))
	for _, pt := range []chess.PieceType{chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King} {
		full := 0
		for i := 0; i < 50; i++ {
			if len(NewTreasure(rng, pt, chess.White, 3, chess.Sq(0, 0)).Targets) == 3 {
				full++
			}
		}
		if full < 40 {
			t.Fatalf("%v: only %d/50 rounds had 3 stars", pt, full)
		}
	}
}

func TestTreasureAvoidsTheLastStart(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 8))
	same := 0
	for i := 0; i < 200; i++ {
		if NewTreasure(rng, chess.Queen, chess.White, 2, chess.Sq(2, 2)).From == chess.Sq(2, 2) {
			same++
		}
	}
	if same > 0 {
		t.Fatalf("started on the avoided square %d/200 times", same)
	}
}

func TestTreasureDeterministicForSeed(t *testing.T) {
	a := NewTreasure(rand.New(rand.NewPCG(5, 5)), chess.Bishop, chess.White, 3, chess.Sq(0, 0))
	b := NewTreasure(rand.New(rand.NewPCG(5, 5)), chess.Bishop, chess.White, 3, chess.Sq(0, 0))
	if a.From != b.From || a.Optimal != b.Optimal || len(a.Targets) != len(b.Targets) {
		t.Fatal("same seed, different puzzle")
	}
	for i := range a.Targets {
		if a.Targets[i] != b.Targets[i] {
			t.Fatal("same seed, different targets")
		}
	}
}

func checkCatch(t *testing.T, p Puzzle, k int, pt chess.PieceType) {
	t.Helper()
	if len(p.Targets) < 1 || len(p.Targets) > k {
		t.Fatalf("%v: %d pawns, want 1..%d", pt, len(p.Targets), k)
	}
	if p.Board.At(p.From) != p.Piece {
		t.Fatalf("piece not on its start square")
	}
	seen := map[chess.Square]bool{}
	for _, s := range p.Targets {
		got := p.Board.At(s)
		if s == p.From || seen[s] || got.Type != chess.Pawn || got.Color == p.Piece.Color {
			t.Fatalf("%v: target %v does not hold an enemy pawn", pt, s)
		}
		if !pawnHome(int(s.Rank)) {
			t.Fatalf("%v: pawn on rank %d", pt, s.Rank)
		}
		seen[s] = true
	}
	// Nothing else on the board but the piece and its targets.
	if got := len(p.Board.Occupied()); got != len(p.Targets)+1 {
		t.Fatalf("%v: %d pieces on the board, want %d", pt, got, len(p.Targets)+1)
	}
	total, ok := Tour(p.Board, p.From, p.Targets, catchLeg)
	if !ok || total != p.Optimal || total > catchMaxTour {
		t.Fatalf("%v: Optimal=%d, Tour=%d,%v", pt, p.Optimal, total, ok)
	}
}

func TestCatchIsAlwaysSolvable(t *testing.T) {
	for seed := uint64(0); seed < 60; seed++ {
		rng := rand.New(rand.NewPCG(seed, 13))
		for _, pt := range allTypes {
			for k := 2; k <= 3; k++ {
				checkCatch(t, NewCatch(rng, pt, chess.White, k, chess.Sq(0, 0)), k, pt)
			}
		}
	}
}

func TestCatchUsuallyHasAllThePawns(t *testing.T) {
	rng := rand.New(rand.NewPCG(4, 5))
	for _, pt := range []chess.PieceType{chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King} {
		full := 0
		for i := 0; i < 50; i++ {
			if len(NewCatch(rng, pt, chess.White, 3, chess.Sq(0, 0)).Targets) == 3 {
				full++
			}
		}
		if full < 25 {
			t.Fatalf("%v: only %d/50 rounds had 3 pawns", pt, full)
		}
	}
}

func TestCatchDeterministicForSeed(t *testing.T) {
	a := NewCatch(rand.New(rand.NewPCG(8, 8)), chess.Rook, chess.White, 3, chess.Sq(0, 0))
	b := NewCatch(rand.New(rand.NewPCG(8, 8)), chess.Rook, chess.White, 3, chess.Sq(0, 0))
	if a.From != b.From || a.Optimal != b.Optimal || len(a.Targets) != len(b.Targets) {
		t.Fatal("same seed, different puzzle")
	}
}

func TestCatchFallbackIsSolvableForEveryPiece(t *testing.T) {
	for _, pt := range allTypes {
		checkCatch(t, fallbackCatch(pt, chess.White), 1, pt)
	}
}
