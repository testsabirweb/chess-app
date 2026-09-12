package render

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
)

//go:embed assets/pieces/cburnett/*.svg
var pieceFS embed.FS

type pieceKey struct {
	typ chess.PieceType
	col chess.Color
	px  int
}

var (
	pieceCache = map[pieceKey]*ebiten.Image{}
	pieceMu    sync.Mutex
)

func pieceAssetName(p chess.Piece) string {
	color := 'w'
	if p.Color == chess.Black {
		color = 'b'
	}
	var kind byte
	switch p.Type {
	case chess.Pawn:
		kind = 'P'
	case chess.Knight:
		kind = 'N'
	case chess.Bishop:
		kind = 'B'
	case chess.Rook:
		kind = 'R'
	case chess.Queen:
		kind = 'Q'
	case chess.King:
		kind = 'K'
	default:
		kind = 'P'
	}
	return fmt.Sprintf("assets/pieces/cburnett/%c%c.svg", color, kind)
}

func pieceImage(p chess.Piece, px int) *ebiten.Image {
	px = quantize(px)
	key := pieceKey{typ: p.Type, col: p.Color, px: px}
	pieceMu.Lock()
	defer pieceMu.Unlock()
	if img, ok := pieceCache[key]; ok {
		return img
	}
	data, err := pieceFS.ReadFile(pieceAssetName(p))
	if err != nil {
		panic(err)
	}
	img, err := rasterSVG(data, px)
	if err != nil {
		panic(err)
	}
	pieceCache[key] = img
	return img
}

// rasterSVG renders an SVG into a square image of the given pixel size.
func rasterSVG(data []byte, size int) (*ebiten.Image, error) {
	rgba, err := rasterSVGImage(data, size)
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(rgba), nil
}

// rasterSVGImage is the CPU half of rasterSVG, kept separate so tests can
// inspect the pixels without a GPU.
func rasterSVGImage(data []byte, size int) (*image.RGBA, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1)
	return rgba, nil
}

// PieceSilhouetteForTest rasterises a piece's SVG on the CPU and returns which
// pixels it covers. Used by the shape-parity test.
func PieceSilhouetteForTest(p chess.Piece, size int) ([]bool, error) {
	data, err := pieceFS.ReadFile(pieceAssetName(p))
	if err != nil {
		return nil, err
	}
	img, err := rasterSVGImage(data, size)
	if err != nil {
		return nil, err
	}
	mask := make([]bool, size*size)
	for i := range mask {
		mask[i] = img.Pix[i*4+3] > 128
	}
	return mask, nil
}

// DrawPiece paints a piece inside the rect. On the board it gets a soft contact
// shadow so it looks like it is standing on the square rather than printed on
// it; on flat surfaces (the home cards, the header) pass shadow=false.
func DrawPiece(dst *ebiten.Image, p chess.Piece, r layout.Rect, lift float64, shadow bool) {
	if p.IsEmpty() {
		return
	}
	img := pieceImage(p, int(r.W+0.5))
	b := img.Bounds()
	scale := r.W / float64(b.Dx())

	if shadow {
		cx := r.X + r.W/2
		baseY := r.Y + r.H*0.92
		shadowScale := 1.0 - clamp01(lift/(r.H*0.5))*0.35
		DrawSoftShadow(dst, cx, baseY, r.W*0.34*shadowScale, r.H*0.12*shadowScale, ColorShadow)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(r.X, r.Y+(r.H-float64(b.Dy())*scale)/2-lift)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(img, op)
}

// scarfBand places the scarf inside the cburnett bishop's 45-unit viewBox. The
// numbers come from the rasterised silhouette: the body is at its narrowest at
// y=26 (13 units across, against 19 for the mitre above and 34 for the foot
// below), which is the collar the artwork already draws a line on.
const (
	scarfCentreY = 27.0 / 45
	scarfWidth   = 16.0 / 45
	scarfHeight  = 3.6 / 45
	scarfEdge    = 1.0 / 45
)

// DrawBishopScarf ties a scarf round the bishop's neck, marking it as the one
// that runs on the dark squares - the same trick as the green scarf on the
// wooden set at home.
//
// r and lift must be the values passed to DrawPiece for the same piece. The
// piece SVG is square, so DrawPiece lands it in a box of side r.W centred
// vertically in r; the scarf is positioned off that box, not off r itself.
func DrawBishopScarf(dst *ebiten.Image, r layout.Rect, lift float64) {
	side := r.W
	top := r.Y + (r.H-side)/2 - lift
	cx := r.X + side/2
	cy := top + side*scarfCentreY

	w, h := side*scarfWidth, side*scarfHeight
	e := side * scarfEdge

	// A dark edge first, so the green keeps its shape against both the white
	// piece and the dark square it is standing on.
	FillRoundRect(dst, cx-w/2-e, cy-h/2-e, w+2*e, h+2*e, (h+2*e)/2, ColorScarfEdge)
	FillRoundRect(dst, cx-w/2, cy-h/2, w, h, h/2, ColorScarf)

	// A short tail hanging off one side. It is what makes the band read as a
	// scarf rather than a stripe.
	tw, th := w*0.26, h*1.6
	tx, ty := cx+w*0.20, cy+h*0.30
	FillRoundRect(dst, tx-e, ty-e, tw+2*e, th+2*e, (tw+2*e)/2, ColorScarfEdge)
	FillRoundRect(dst, tx, ty, tw, th, tw/2, ColorScarf)
}

// RasterPieceForTest exposes rasterization for tests.
func RasterPieceForTest(p chess.Piece, px int) *ebiten.Image {
	return pieceImage(p, px)
}
