package challenge

import (
	"math/rand/v2"
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

var whichPool = []chess.PieceType{chess.Pawn, chess.Bishop, chess.Rook, chess.Queen, chess.King}

func checkWhich(t *testing.T, w Which, n int, pool []chess.PieceType) {
	t.Helper()
	if len(w.Pieces) < 2 || len(w.Pieces) > n {
		t.Fatalf("got %d pieces, want 2..%d", len(w.Pieces), n)
	}
	if !w.Board.At(w.Target).IsEmpty() {
		t.Fatalf("target %v is under a piece", w.Target)
	}
	seen := map[chess.PieceType]bool{}
	reachers, answerReaches := 0, false
	for _, sq := range w.Pieces {
		p := w.Board.At(sq)
		if p.IsEmpty() || p.Color != chess.White {
			t.Fatalf("no white piece on %v", sq)
		}
		if seen[p.Type] {
			t.Fatalf("type %v appears twice", p.Type)
		}
		seen[p.Type] = true
		inPool := false
		for _, pt := range pool {
			inPool = inPool || pt == p.Type
		}
		if !inPool {
			t.Fatalf("%v is not in the pool", p.Type)
		}
		if p.Type == chess.Pawn && (sq.Rank < 1 || sq.Rank > 3) {
			t.Fatalf("pawn on rank %d", sq.Rank)
		}
		if w.Board.CanMove(sq, w.Target) {
			reachers++
			answerReaches = answerReaches || sq == w.Answer
		}
	}
	if reachers != 1 || !answerReaches {
		t.Fatalf("%d pieces reach the target, answer reaches=%v", reachers, answerReaches)
	}
}

func TestWhichHasExactlyOneAnswer(t *testing.T) {
	for seed := uint64(0); seed < 300; seed++ {
		rng := rand.New(rand.NewPCG(seed, 7))
		for n := 2; n <= 3; n++ {
			checkWhich(t, NewWhich(rng, whichPool, n, chess.NoPiece, nil), n, whichPool)
		}
	}
}

func TestWhichAvoidsTheLastAnswerType(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	for i := 0; i < 200; i++ {
		w := NewWhich(rng, whichPool, 3, chess.Rook, nil)
		if w.Board.At(w.Answer).Type == chess.Rook {
			t.Fatalf("round %d answered with the avoided type", i)
		}
	}
}

func TestWhichWorksWithTheKnight(t *testing.T) {
	pool := []chess.PieceType{chess.Knight, chess.Rook, chess.King}
	rng := rand.New(rand.NewPCG(9, 9))
	for i := 0; i < 100; i++ {
		checkWhich(t, NewWhich(rng, pool, 3, chess.NoPiece, nil), 3, pool)
	}
}

func TestWhichTinyPoolStillReturnsARound(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 1))
	w := NewWhich(rng, []chess.PieceType{chess.Rook}, 3, chess.NoPiece, nil)
	if w.Board == nil || len(w.Pieces) == 0 {
		t.Fatal("no round returned")
	}
}

func TestWhichDeterministicForSeed(t *testing.T) {
	a := NewWhich(rand.New(rand.NewPCG(5, 6)), whichPool, 3, chess.NoPiece, nil)
	b := NewWhich(rand.New(rand.NewPCG(5, 6)), whichPool, 3, chess.NoPiece, nil)
	if a.Target != b.Target || a.Answer != b.Answer || len(a.Pieces) != len(b.Pieces) {
		t.Fatalf("same seed gave different rounds: %+v vs %+v", a, b)
	}
	for i := range a.Pieces {
		if a.Pieces[i] != b.Pieces[i] {
			t.Fatalf("same seed gave different pieces")
		}
	}
}

func TestWhichAvoidsRecentSquares(t *testing.T) {
	rng := rand.New(rand.NewPCG(2, 2))
	var mem Memory
	reused, rounds := 0, 200
	var last []chess.Square
	for i := 0; i < rounds; i++ {
		w := NewWhich(rng, whichPool, 3, chess.NoPiece, &mem)
		now := append(append([]chess.Square{}, w.Pieces...), w.Target)
		for _, s := range now {
			for _, o := range last {
				if s == o {
					reused++
				}
			}
		}
		last = now
	}
	// Some overlap is unavoidable on a 25-square board; it should be rare.
	if reused > rounds/2 {
		t.Fatalf("%d squares reused from the previous round over %d rounds", reused, rounds)
	}
}
