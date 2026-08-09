package render_test

import (
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/render"
)

func TestPieceAssetsRasterize(t *testing.T) {
	pieces := []chess.PieceType{chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King}
	for _, pt := range pieces {
		for _, c := range []chess.Color{chess.White, chess.Black} {
			img := render.RasterPieceForTest(chess.Piece{Type: pt, Color: c}, 64)
			if img.Bounds().Dx() != 64 || img.Bounds().Dy() != 64 {
				t.Fatalf("bad raster size for %v %v", pt, c)
			}
		}
	}
}
