package challenge_test

import (
	"math/rand/v2"
	"reflect"
	"testing"
	"time"

	"github.com/testsabirweb/chess-app/internal/challenge"
	"github.com/testsabirweb/chess-app/internal/chess"
)

func defaultSpec() challenge.Spec {
	return challenge.Spec{
		BoardWidth:  5,
		BoardHeight: 5,
		Pieces: []chess.PieceType{
			chess.Pawn, chess.Knight, chess.Bishop,
			chess.Rook, chess.Queen, chess.King,
		},
		Color:    chess.White,
		Decoys:   true,
	}
}

func TestNextAlwaysSolvable(t *testing.T) {
	g := challenge.NewGenerator(defaultSpec(), rand.New(rand.NewPCG(42, 7)))
	for i := 0; i < 10000; i++ {
		c := g.Next()
		playerPieces := 0
		for _, sq := range c.Board.Occupied() {
			if c.Board.At(sq).Color == c.Piece.Color {
				playerPieces++
			}
		}
		if playerPieces != 1 {
			t.Fatalf("draw %d: expected one player piece, got %d", i, playerPieces)
		}
		if c.From == c.Target {
			t.Fatalf("draw %d: from == target", i)
		}
		if !containsSquare(c.Solutions, c.Target) {
			t.Fatalf("draw %d: target not in solutions", i)
		}
		if !reflect.DeepEqual(c.Solutions, c.Board.MoveTargets(c.From)) {
			t.Fatalf("draw %d: solutions mismatch", i)
		}
	}
}

func TestDeterministicForSeed(t *testing.T) {
	spec := defaultSpec()
	a := challenge.NewGenerator(spec, rand.New(rand.NewPCG(99, 1)))
	b := challenge.NewGenerator(spec, rand.New(rand.NewPCG(99, 1)))
	for i := 0; i < 50; i++ {
		ca, cb := a.Next(), b.Next()
		if ca.From != cb.From || ca.Target != cb.Target || ca.Piece.Type != cb.Piece.Type {
			t.Fatalf("draw %d diverged: %+v vs %+v", i, ca, cb)
		}
	}
}

func TestDistanceSpec(t *testing.T) {
	spec := challenge.Spec{
		BoardWidth: 5, BoardHeight: 5,
		Pieces:      []chess.PieceType{chess.Rook},
		Color:       chess.White,
		MinDistance: 2,
		MaxDistance: 3,
	}
	g := challenge.NewGenerator(spec, rand.New(rand.NewPCG(5, 5)))
	for i := 0; i < 500; i++ {
		c := g.Next()
		d := challenge.Distance(c.From, c.Target)
		if d < 2 || d > 3 {
			t.Fatalf("distance %d out of range for %+v", d, c)
		}
	}
}

func TestImpossibleSpecDoesNotHang(t *testing.T) {
	spec := challenge.Spec{
		BoardWidth:  5,
		BoardHeight: 5,
		Pieces:      []chess.PieceType{chess.Rook},
		Color:       chess.White,
		MinDistance: 99,
		MaxDistance: 100,
	}
	done := make(chan struct{})
	go func() {
		g := challenge.NewGenerator(spec, rand.New(rand.NewPCG(1, 1)))
		_ = g.Next()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("generator hung on impossible spec")
	}
}

func TestRookCoverage(t *testing.T) {
	spec := challenge.Spec{
		BoardWidth: 5, BoardHeight: 5,
		Pieces: []chess.PieceType{chess.Rook},
		Color:  chess.White,
	}
	g := challenge.NewGenerator(spec, rand.New(rand.NewPCG(123, 456)))
	counts := map[chess.Square]int{}
	for i := 0; i < 5000; i++ {
		c := g.Next()
		counts[c.Target]++
	}
	min, max := int(^uint(0) >> 1), 0
	for _, n := range counts {
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	if len(counts) < 20 {
		t.Fatalf("rook coverage too narrow: %d squares", len(counts))
	}
	if float64(min) < 0.01*float64(max) {
		t.Fatalf("coverage skewed: min=%d max=%d", min, max)
	}
}

func TestPawnDecoysCapture(t *testing.T) {
	spec := challenge.Spec{
		BoardWidth: 5, BoardHeight: 5,
		Pieces: []chess.PieceType{chess.Pawn},
		Color:  chess.White,
		Decoys: true,
	}
	g := challenge.NewGenerator(spec, rand.New(rand.NewPCG(7, 8)))
	captures := 0
	for i := 0; i < 2000; i++ {
		c := g.Next()
		for _, m := range c.Board.Moves(nil, c.From) {
			if m.Capture {
				captures++
				break
			}
		}
	}
	if captures == 0 {
		t.Fatal("pawn decoys never produced capture targets")
	}
}

func BenchmarkNext(b *testing.B) {
	g := challenge.NewGenerator(defaultSpec(), rand.New(rand.NewPCG(1, 2)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Next()
	}
}

func containsSquare(ss []chess.Square, s chess.Square) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
