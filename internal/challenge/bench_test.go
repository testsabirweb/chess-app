package challenge

import (
	"math/rand/v2"
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

// These benchmarks guard the cost of dealing a round, which runs on every new
// puzzle, behind a short fade, on whatever phone the game is installed on.
// Run: go test ./internal/challenge -run xxx -bench . -benchmem

func benchPieces() []chess.PieceType {
	return []chess.PieceType{chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King}
}

func BenchmarkStarNext(b *testing.B) {
	spec := Spec{BoardWidth: 5, BoardHeight: 5, Pieces: []chess.PieceType{chess.Queen}, Color: chess.White, MinMoves: 1, MaxMoves: 3}
	g := NewGenerator(spec, rand.New(rand.NewPCG(1, 2)))
	var mem Memory
	g.SetMemory(&mem)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g.Next()
	}
}

func BenchmarkTreasure(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	var mem Memory
	pts := benchPieces()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewTreasure(rng, pts[i%len(pts)], chess.White, 3, &mem)
	}
}

func BenchmarkCatch(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	var mem Memory
	pts := benchPieces()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewCatch(rng, pts[i%len(pts)], chess.White, 3, &mem)
	}
}

func BenchmarkSafe(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	var mem Memory
	pts := benchPieces()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewSafe(rng, pts[i%len(pts)], chess.White, safeGuards, &mem, i%2 == 1)
	}
}

func BenchmarkWhich(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	var mem Memory
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewWhich(rng, whichPool, 3, chess.NoPiece, &mem)
	}
}

// The worst case the generators can hit: every preference unmeetable, so every
// attempt is spent before falling back. Pawns are the likeliest to get here.
func BenchmarkWorstCaseSafePawn(b *testing.B) {
	rng := rand.New(rand.NewPCG(3, 4))
	var mem Memory
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		NewSafe(rng, chess.Pawn, chess.White, safeGuards, &mem, true)
	}
}
