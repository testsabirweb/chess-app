package render

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"strings"
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

// OpponentFill recolours the black pieces. The cburnett black set is drawn with
// plain black fills - and the pawn and queen have no fill attribute at all, so
// they fall back to black - which on a small screen reads as a featureless
// blob rather than a chess piece. A saturated colour keeps the silhouette and
// makes "the other side" obvious. Set it to "#000" for the classic look.
const OpponentFill = "#6B3FA0"

// recolour rewrites an SVG's black fills. Paths that set their own fill (the
// light detail lines, and fill="none" outlines) are left alone; everything that
// inherits its fill picks up the wrapper group's colour.
func recolour(data []byte, hex string) []byte {
	s := string(data)
	s = strings.ReplaceAll(s, `fill="#000"`, `fill="`+hex+`"`)
	tagEnd := strings.Index(s, ">")
	svgEnd := strings.LastIndex(s, "</svg>")
	if tagEnd < 0 || svgEnd < 0 || svgEnd < tagEnd {
		return data
	}
	return []byte(s[:tagEnd+1] + `<g fill="` + hex + `">` + s[tagEnd+1:svgEnd] + `</g>` + s[svgEnd:])
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
	if p.Color == chess.Black && OpponentFill != "#000" {
		data = recolour(data, OpponentFill)
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

// RasterPieceForTest exposes rasterization for tests.
func RasterPieceForTest(p chess.Piece, px int) *ebiten.Image {
	return pieceImage(p, px)
}
