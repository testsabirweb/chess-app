package render_test

import (
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/render"
)

// The cburnett black pieces are svgo-optimised: several are written with
// compact elliptical-arc commands where the white ones use cubic curves. Our
// SVG rasteriser gets one of those arcs wrong, which nobody could see while
// the piece was black-on-black - the black pawn came out with a lopsided head.
//
// This compares the two colours as silhouettes. They are the same drawing, so
// they must cover the same pixels.
func TestBlackAndWhitePiecesShareASilhouette(t *testing.T) {
	const size = 128
	// The queen is the one deliberate difference in the set: white has open
	// crown tips, black has filled ones. Everything else is the same drawing.
	minOverlap := map[chess.PieceType]float64{
		chess.Pawn: 0.97, chess.Knight: 0.97, chess.Bishop: 0.97,
		chess.Rook: 0.97, chess.King: 0.97, chess.Queen: 0.80,
	}
	for pt := range minOverlap {
		white, err := render.PieceSilhouetteForTest(chess.Piece{Type: pt, Color: chess.White}, size)
		if err != nil {
			t.Fatalf("%v white: %v", pt, err)
		}
		black, err := render.PieceSilhouetteForTest(chess.Piece{Type: pt, Color: chess.Black}, size)
		if err != nil {
			t.Fatalf("%v black: %v", pt, err)
		}

		var union, both int
		for i := range white {
			if white[i] || black[i] {
				union++
			}
			if white[i] && black[i] {
				both++
			}
		}
		if union == 0 {
			t.Fatalf("%v: both silhouettes are empty", pt)
		}
		overlap := float64(both) / float64(union)
		t.Logf("piece %v: silhouettes overlap %.1f%%", pt, overlap*100)
		if overlap < minOverlap[pt] {
			t.Errorf("%v: black and white silhouettes only overlap %.1f%%, want >= %.0f%%; the two colours have drifted apart",
				pt, overlap*100, minOverlap[pt]*100)
		}
	}
}
