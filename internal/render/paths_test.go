package render_test

import (
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/render"
)

func TestPiecePathsClosed(t *testing.T) {
	pieces := []chess.PieceType{chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King}
	for _, pt := range pieces {
		if !render.PathBoundsOK(pt) {
			t.Fatalf("path missing for %v", pt)
		}
	}
}
