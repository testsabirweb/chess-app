package puzzles

import (
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

// Judging a move in a mate in three runs on the phone as the piece lands, so
// it has to stay quick.
func BenchmarkMateThreeJudge(b *testing.B) {
	p := ByTier()[Mate3][0]
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ContinuesMate(p.Board, chess.White, p.Solution, 2)
	}
}

func BenchmarkMateThreeHelp(b *testing.B) {
	p := ByTier()[Mate3][0]
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		MateMove(p.Board, chess.White, 3)
	}
}
