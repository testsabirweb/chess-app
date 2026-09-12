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

// The scarf marks the dark-square bishop, so its parity must stay in step with
// the checkerboard DrawBoard paints.
func TestDarkSquareMatchesBoardParity(t *testing.T) {
	for _, tc := range []struct {
		f, r int
		dark bool
	}{{0, 0, false}, {1, 0, true}, {0, 1, true}, {1, 1, false}, {4, 4, false}, {3, 4, true}} {
		if got := render.DarkSquare(chess.Sq(tc.f, tc.r)); got != tc.dark {
			t.Errorf("DarkSquare(%d,%d) = %v, want %v", tc.f, tc.r, got, tc.dark)
		}
	}
}
