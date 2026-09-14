package render

import "image/color"

// rgba builds an alpha-premultiplied colour, which is what image/color.RGBA and
// therefore Ebitengine expect. Writing {255, 255, 255, 40} directly is the
// classic mistake: it is not a valid premultiplied colour and renders opaque.
func rgba(r, g, b uint8, a float64) color.RGBA {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	return color.RGBA{
		R: uint8(float64(r) * a),
		G: uint8(float64(g) * a),
		B: uint8(float64(b) * a),
		A: uint8(255 * a),
	}
}

// The whole look is defined here. Everything else in render/ reads these.
var (
	// Background: a warm, near-flat charcoal. Dark enough to stay out of the
	// way, warm enough to read as a chosen colour rather than an absence.
	ColorBGTop    = color.RGBA{48, 45, 41, 255}
	ColorBGBottom = color.RGBA{34, 32, 29, 255}

	// Board: the classic cream-and-green. High value contrast keeps the light
	// and dark squares obviously different, which the bishop's scarf depends on,
	// and the warm cream flatters the gold star far better than a cool white.
	ColorBoardL     = color.RGBA{238, 238, 210, 255}
	ColorBoardD     = color.RGBA{118, 150, 86, 255}
	ColorFrame      = color.RGBA{140, 108, 74, 255}
	ColorFrameInner = color.RGBA{96, 72, 48, 255}

	// Move hints. Blue, not mint: the board is green now, and a green dot on a
	// green square is no signal at all.
	ColorHint     = rgba(70, 150, 255, 0.32)
	ColorHintDot  = rgba(28, 110, 220, 0.95)
	ColorHintRing = rgba(255, 255, 255, 0.80)

	// Knight L-trail: light blue so it reads as "the path you took", distinct
	// from the yellow picked-piece glow and the darker hint dots.
	ColorTrail     = rgba(140, 205, 255, 0.95)
	ColorTrailGlow = rgba(100, 180, 255, 0.65)

	// The piece that is ready to be picked up. Unpicked is a quiet breathe;
	// picked is a bright wash plus a white ring so it reads even before the
	// move dots arrive.
	ColorPickable     = rgba(255, 206, 92, 0.40)
	ColorPickableWash = rgba(255, 202, 80, 0.28)
	ColorPicked       = rgba(255, 236, 150, 0.95)
	ColorPickedWash   = rgba(255, 220, 100, 0.72)
	ColorPickedRing   = rgba(255, 255, 255, 0.92)

	ColorStarGlow = color.RGBA{255, 226, 120, 255}

	// The dark-square bishop's scarf. A deeper grass green than the board on
	// purpose: the scarf sits on the piece body and must never read as "you can
	// move here".
	ColorScarf     = color.RGBA{46, 160, 67, 255}
	ColorScarfEdge = color.RGBA{20, 62, 30, 255}

	ColorText       = color.RGBA{255, 255, 255, 255}
	ColorTextDim    = color.RGBA{206, 200, 190, 255}
	ColorTextShadow = rgba(20, 16, 10, 0.45)
	ColorShadow     = rgba(18, 14, 8, 0.40)
	ColorGloss      = rgba(255, 255, 255, 0.10)

	// PLAY in the reference's green — the one loud thing on the home screen.
	ColorPlay     = color.RGBA{129, 182, 76, 255}
	ColorPlayHi   = color.RGBA{160, 205, 110, 255}
	ColorPlayEdge = color.RGBA{84, 124, 48, 255}
	// The back button stays deliberately quiet, warm rather than blue-grey.
	ColorBack     = color.RGBA{70, 64, 58, 255}
	ColorBackEdge = color.RGBA{46, 42, 38, 255}
	ColorTray     = rgba(255, 255, 255, 0.10)
	ColorTraySlot = rgba(255, 255, 255, 0.10)
	ColorPanel    = rgba(255, 255, 255, 0.12)

	PieceFills = map[string]color.RGBA{
		"white": {250, 250, 255, 255},
		"black": {70, 75, 95, 255},
	}
)

// PieceCardColors are the six card colours on the home screen, indexed the same
// way as game.allPieces (pawn, knight, bishop, rook, queen, king).
var PieceCardColors = []color.RGBA{
	{230, 106, 106, 255}, // pawn   - coral
	{235, 150, 58, 255},  // knight - orange
	{92, 180, 120, 255},  // bishop - green
	{84, 150, 225, 255},  // rook   - blue
	{168, 116, 224, 255}, // queen  - purple
	{235, 186, 76, 255},  // king   - yellow
}

// PieceCardEdges are the darker rims that give the cards their chunky, tappable
// look.
var PieceCardEdges = []color.RGBA{
	{158, 62, 62, 255},
	{162, 98, 28, 255},
	{52, 128, 80, 255},
	{44, 100, 168, 255},
	{114, 68, 164, 255},
	{166, 126, 34, 255},
}

// Alpha scales a premultiplied colour's opacity. Every channel is scaled,
// because in premultiplied form the RGB channels already carry the alpha.
func Alpha(c color.RGBA, a float64) color.RGBA {
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	return color.RGBA{
		R: uint8(float64(c.R) * a),
		G: uint8(float64(c.G) * a),
		B: uint8(float64(c.B) * a),
		A: uint8(float64(c.A) * a),
	}
}
