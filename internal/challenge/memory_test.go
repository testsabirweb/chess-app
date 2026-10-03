package challenge

import (
	"math/rand/v2"
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

func TestMemoryForgetsTheOldestRound(t *testing.T) {
	var m Memory
	m.Remember(chess.Sq(0, 0))
	m.Remember(chess.Sq(1, 1))
	if !m.Has(chess.Sq(0, 0)) || !m.Has(chess.Sq(1, 1)) {
		t.Fatal("both recent rounds should be remembered")
	}
	m.Remember(chess.Sq(2, 2))
	if m.Has(chess.Sq(0, 0)) {
		t.Fatal("the oldest round should have been forgotten")
	}
	if !m.HasAny([]chess.Square{chess.Sq(4, 4), chess.Sq(2, 2)}) {
		t.Fatal("HasAny missed a remembered square")
	}
}

func TestNilMemoryIsHarmless(t *testing.T) {
	var m *Memory
	m.Remember(chess.Sq(0, 0))
	if m.Has(chess.Sq(0, 0)) || m.HasAny([]chess.Square{chess.Sq(0, 0)}) {
		t.Fatal("a nil Memory remembers nothing")
	}
}

// The original star game shares the memory: its start and its star avoid what
// recent rounds used.
func TestStarGeneratorAvoidsRecentSquares(t *testing.T) {
	spec := Spec{BoardWidth: 5, BoardHeight: 5, Pieces: []chess.PieceType{chess.Rook}, Color: chess.White, MinMoves: 1, MaxMoves: 3}
	var mem Memory
	g := NewGenerator(spec, rand.New(rand.NewPCG(1, 2)))
	g.SetMemory(&mem)
	reused, rounds := 0, 300
	var last []chess.Square
	for i := 0; i < rounds; i++ {
		c := g.Next()
		now := []chess.Square{c.From, c.Target}
		for _, s := range now {
			for _, o := range last {
				if s == o {
					reused++
				}
			}
		}
		last = now
	}
	if reused > rounds/20 {
		t.Fatalf("%d squares reused from the previous round over %d rounds", reused, rounds)
	}
}
