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
	if px < 8 {
		px = 8
	}
	key := pieceKey{typ: p.Type, col: p.Color, px: px}
	pieceMu.Lock()
	defer pieceMu.Unlock()
	if img, ok := pieceCache[key]; ok {
		return img
	}
	name := pieceAssetName(p)
	data, err := pieceFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	img, err := rasterPieceSVG(data, px)
	if err != nil {
		panic(err)
	}
	pieceCache[key] = img
	return img
}

func rasterPieceSVG(data []byte, size int) (*ebiten.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1)
	return ebiten.NewImageFromImage(rgba), nil
}

func DrawPiece(dst *ebiten.Image, p chess.Piece, r layout.Rect) {
	px := int(r.W + 0.5)
	img := pieceImage(p, px)
	op := &ebiten.DrawImageOptions{}
	b := img.Bounds()
	scale := r.W / float64(b.Dx())
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(r.X, r.Y+(r.H-float64(b.Dy())*scale)/2)
	dst.DrawImage(img, op)
}

// RasterPieceForTest exposes rasterization for tests.
func RasterPieceForTest(p chess.Piece, px int) *ebiten.Image {
	return pieceImage(p, px)
}
