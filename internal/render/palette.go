package render

import "image/color"

var (
	ColorBG       = color.RGBA{24, 26, 38, 255}
	ColorBoardL   = color.RGBA{255, 255, 255, 255}
	ColorBoardD   = color.RGBA{181, 181, 181, 255}
	ColorMoveHint = color.RGBA{255, 120, 120, 110}
	ColorOutline  = color.RGBA{35, 35, 50, 255}
	ColorWhite    = color.RGBA{252, 252, 255, 255}
	ColorAccent   = color.RGBA{255, 210, 60, 255}
	ColorStar     = color.RGBA{255, 220, 40, 255}
	ColorButton   = color.RGBA{88, 130, 220, 255}
	ColorButtonHi = color.RGBA{120, 160, 245, 255}
	ColorText     = color.RGBA{255, 255, 255, 255}
	ColorShadow   = color.RGBA{0, 0, 0, 80}

	PieceFills = map[string]color.RGBA{
		"white": {250, 250, 255, 255},
		"black": {70, 75, 95, 255},
	}
)
