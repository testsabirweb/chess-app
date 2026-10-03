package challenge

import (
	"math/rand/v2"
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

var safeGuards = []chess.PieceType{chess.Pawn, chess.Bishop, chess.Rook, chess.Queen, chess.King}

func checkSafe(t *testing.T, p Puzzle, pt chess.PieceType) {
	t.Helper()
	if len(p.Targets) != 1 {
		t.Fatalf("%v: %d stars, want 1", pt, len(p.Targets))
	}
	star := p.Targets[0]
	avoid := HotSet(p.Hot, p.Guard)

	if p.Board.At(p.From) != p.Piece {
		t.Fatalf("%v: piece missing from its start", pt)
	}
	if g := p.Board.At(p.Guard); g.IsEmpty() || g.Color == p.Piece.Color {
		t.Fatalf("%v: no enemy guard on %v", pt, p.Guard)
	}
	if avoid(p.From) {
		t.Fatalf("%v: starts on a hot square or the guard's own", pt)
	}
	if avoid(star) || !p.Board.At(star).IsEmpty() {
		t.Fatalf("%v: star %v is hot, taken or under a piece", pt, star)
	}

	// Hot is exactly what the guard attacks with the player's piece lifted.
	nb := p.Board.Clone()
	nb.Set(p.From, chess.Piece{})
	want := map[chess.Square]bool{}
	for _, s := range nb.Attacks(p.Guard) {
		want[s] = true
	}
	if len(want) != len(p.Hot) {
		t.Fatalf("%v: Hot has %d squares, guard attacks %d", pt, len(p.Hot), len(want))
	}
	for _, s := range p.Hot {
		if !want[s] {
			t.Fatalf("%v: %v is hot but not attacked", pt, s)
		}
	}

	d := MovesToAvoiding(p.Board, p.From, star, safeLeg, avoid)
	if d != p.Optimal || d < safeMinMoves(pt) || d > safeLeg {
		t.Fatalf("%v: Optimal=%d, safe route=%d", pt, p.Optimal, d)
	}

	// Danger has to matter.
	hotMove := false
	for _, to := range p.Board.MoveTargets(p.From) {
		hotMove = hotMove || avoid(to)
	}
	if !hotMove {
		t.Fatalf("%v: none of the first moves is dangerous", pt)
	}
}

func TestSafeIsAlwaysSolvable(t *testing.T) {
	for seed := uint64(0); seed < 40; seed++ {
		rng := rand.New(rand.NewPCG(seed, 17))
		for _, pt := range allTypes {
			for _, detour := range []bool{false, true} {
				checkSafe(t, NewSafe(rng, pt, chess.White, safeGuards, nil, detour), pt)
			}
		}
	}
}

func TestSafeFallbackIsValidForEveryPiece(t *testing.T) {
	for _, pt := range allTypes {
		checkSafe(t, fallbackSafe(pt, chess.White), pt)
	}
}

// Asking for a detour should usually get one: the safe way round is longer than
// going straight.
func TestSafeDetourIsUsuallyHonoured(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 3))
	got := 0
	const n = 100
	for i := 0; i < n; i++ {
		p := NewSafe(rng, chess.Queen, chess.White, safeGuards, nil, true)
		if p.Optimal > MovesTo(p.Board, p.From, p.Targets[0], safeLeg) {
			got++
		}
	}
	if got < n*6/10 {
		t.Fatalf("only %d/%d rounds had a detour", got, n)
	}
}

func TestSafeAvoidsRecentSquares(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 1))
	var mem Memory
	reused, rounds := 0, 200
	var last []chess.Square
	for i := 0; i < rounds; i++ {
		p := NewSafe(rng, chess.Rook, chess.White, safeGuards, &mem, false)
		now := []chess.Square{p.From, p.Targets[0], p.Guard}
		for _, s := range now {
			for _, o := range last {
				if s == o {
					reused++
				}
			}
		}
		last = now
	}
	if reused > rounds/10 {
		t.Fatalf("%d squares reused from the previous round over %d rounds", reused, rounds)
	}
}

func TestSafeDeterministicForSeed(t *testing.T) {
	a := NewSafe(rand.New(rand.NewPCG(6, 6)), chess.Bishop, chess.White, safeGuards, nil, true)
	b := NewSafe(rand.New(rand.NewPCG(6, 6)), chess.Bishop, chess.White, safeGuards, nil, true)
	if a.From != b.From || a.Guard != b.Guard || a.Targets[0] != b.Targets[0] || len(a.Hot) != len(b.Hot) {
		t.Fatal("same seed, different puzzle")
	}
}

// A pawn guard attacks diagonally down the board, never straight ahead.
func TestPawnGuardHotSquares(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), chess.Piece{Type: chess.Rook, Color: chess.White})
	b.Set(chess.Sq(2, 3), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	hot := guardAttacks(b, chess.Sq(0, 0), chess.Sq(2, 3))
	want := map[chess.Square]bool{chess.Sq(1, 2): true, chess.Sq(3, 2): true}
	if len(hot) != 2 || !want[hot[0]] || !want[hot[1]] {
		t.Fatalf("hot = %v", hot)
	}
}

// The piece may travel through a hot square, only not stop on it.
func TestReachAvoidingMayPassThrough(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), chess.Piece{Type: chess.Rook, Color: chess.White})
	avoid := func(s chess.Square) bool { return s == chess.Sq(0, 2) }
	if got := MovesToAvoiding(b, chess.Sq(0, 0), chess.Sq(0, 4), 3, avoid); got != 1 {
		t.Fatalf("rook should slide past the avoided square in 1 move, got %d", got)
	}
	if got := MovesToAvoiding(b, chess.Sq(0, 0), chess.Sq(0, 2), 3, avoid); got != -1 {
		t.Fatalf("stopping on an avoided square must not be allowed, got %d", got)
	}
}
