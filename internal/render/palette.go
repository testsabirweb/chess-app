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
	// Background: deep blueberry fading into plum. A dark, saturated ground is
	// what makes the white board, the gold star and the confetti pop.
	ColorBGTop    = color.RGBA{34, 38, 92, 255}
	ColorBGBottom = color.RGBA{96, 48, 124, 255}

	// Board. The light square stays white; the dark square is a noticeably
	// deeper slate grey with a hint of blue so the two read clearly apart.
	// Tweak ColorBoardD alone to make the dark squares lighter or darker.
	ColorBoardL     = color.RGBA{255, 255, 255, 255}
	ColorBoardD     = color.RGBA{140, 148, 176, 255}
	ColorFrame      = color.RGBA{255, 206, 110, 255}
	ColorFrameInner = color.RGBA{204, 142, 52, 255}

	// Move hints: a soft mint dot the child learns to chase.
	ColorHint     = rgba(60, 220, 160, 0.30)
	ColorHintDot  = rgba(24, 176, 122, 0.95)
	ColorHintRing = rgba(255, 255, 255, 0.75)

	// The piece that is ready to be picked up.
	ColorPickable     = rgba(255, 206, 92, 0.55)
	ColorPickableWash = rgba(255, 202, 80, 0.42)
	ColorPicked       = rgba(255, 236, 150, 0.80)
	ColorPickedWash   = rgba(255, 216, 100, 0.58)

	ColorStarGlow = color.RGBA{255, 226, 120, 255}

	ColorText       = color.RGBA{255, 255, 255, 255}
	ColorTextDim    = color.RGBA{214, 208, 240, 255}
	ColorTextShadow = rgba(12, 10, 34, 0.45)
	ColorShadow     = rgba(10, 8, 28, 0.40)
	ColorGloss      = rgba(255, 255, 255, 0.13)

	// Buttons.
	ColorPlay     = color.RGBA{58, 214, 130, 255}
	ColorPlayHi   = color.RGBA{120, 240, 178, 255}
	ColorPlayEdge = color.RGBA{28, 150, 88, 255}
	// The back button is deliberately quieter than the play buttons. It is
	// chrome for the grown-up, not something to invite a small hand over.
	ColorBack     = color.RGBA{74, 92, 156, 255}
	ColorBackEdge = color.RGBA{44, 56, 112, 255}
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
	{255, 122, 122, 255}, // pawn   - coral
	{255, 174, 66, 255},  // knight - orange
	{86, 205, 138, 255},  // bishop - green
	{78, 166, 255, 255},  // rook   - blue
	{190, 124, 255, 255}, // queen  - purple
	{255, 206, 88, 255},  // king   - yellow
}

// PieceCardEdges are the darker rims that give the cards their chunky, tappable
// look.
var PieceCardEdges = []color.RGBA{
	{196, 74, 74, 255},
	{196, 118, 26, 255},
	{42, 150, 92, 255},
	{36, 110, 190, 255},
	{132, 70, 200, 255},
	{198, 148, 38, 255},
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
