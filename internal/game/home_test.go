package game

import (
	"testing"

	"github.com/testsabirweb/chess-app/internal/layout"
)

type homeDevice struct {
	name        string
	w, h, scale float64
}

var homeDevices = []homeDevice{
	{"edge50", 1080, 2400, 2.75},
	{"flagship", 1440, 3120, 3.5},
	{"budget", 720, 1600, 2.0},
	{"dev", 432, 960, 1.0},
	{"landscape", 800, 400, 1.0},
	{"small-old", 480, 800, 1.5},
	{"tall-21x9", 1080, 2640, 3.0},
	{"tablet-port", 1600, 2560, 2.0},
	{"tablet-land", 2560, 1600, 2.0},
	{"fold-inner", 1812, 2176, 2.4},
	{"split-screen", 1080, 900, 2.75},
	{"tiny", 400, 400, 1.0},
}

func homeBandHeight(r homeRects) float64 {
	return r.tray.Y + r.tray.H - r.title.Y
}

func TestHomeLayoutFitsSafe(t *testing.T) {
	for _, d := range homeDevices {
		t.Run(d.name, func(t *testing.T) {
			m := layout.Compute(d.w, d.h, d.scale, layout.Insets{}, 5, 5)
			r := homeLayout(m)
			bottom := r.tray.Y + r.tray.H
			if bottom > m.Safe.Y+m.Safe.H+1e-6 {
				t.Fatalf("home bands overflow safe: bottom=%f safeBottom=%f bands=%f",
					bottom, m.Safe.Y+m.Safe.H, homeBandHeight(r))
			}
			if homeBandHeight(r) > m.Safe.H+1e-6 {
				t.Fatalf("home band sum exceeds safe height: %f > %f", homeBandHeight(r), m.Safe.H)
			}
		})
	}
}

func rectsTouch(a, b layout.Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// With every mode on offer the row of mode buttons must still fit the safe
// area and keep clear of every other band, on every device.
func TestHomeLayoutWithAllModes(t *testing.T) {
	saved := activeModes
	t.Cleanup(func() { activeModes = saved })
	activeModes = []Mode{ModeStar, ModeWhich, ModeTreasure, ModeSafe, ModePawnWars}

	for _, d := range homeDevices {
		t.Run(d.name, func(t *testing.T) {
			m := layout.Compute(d.w, d.h, d.scale, layout.Insets{}, 5, 5)
			r := homeLayout(m)
			if bottom := r.tray.Y + r.tray.H; bottom > m.Safe.Y+m.Safe.H+1e-6 {
				t.Fatalf("bands overflow safe: bottom=%f safeBottom=%f", bottom, m.Safe.Y+m.Safe.H)
			}
			others := []layout.Rect{r.play, r.label, r.tray}
			others = append(others, r.cards[:]...)
			for i := range rowModes() {
				tile := r.modes[i]
				if tile.W <= 0 || tile.H <= 0 {
					t.Fatalf("mode %d has no tile", i)
				}
				if tile.X < m.Safe.X-1e-6 || tile.Y < m.Safe.Y-1e-6 ||
					tile.X+tile.W > m.Safe.X+m.Safe.W+1e-6 || tile.Y+tile.H > m.Safe.Y+m.Safe.H+1e-6 {
					t.Fatalf("mode %d tile %+v outside safe %+v", i, tile, m.Safe)
				}
				for j, o := range others {
					if rectsTouch(tile, o) {
						t.Fatalf("mode %d tile %+v overlaps band %d %+v", i, tile, j, o)
					}
				}
				for j := i + 1; j < len(rowModes()); j++ {
					if rectsTouch(tile, r.modes[j]) {
						t.Fatalf("mode tiles %d and %d overlap", i, j)
					}
				}
			}
		})
	}
}

// Until there is a game besides the star game, the home screen has no mode row.
func TestHomeHasNoModeRowForOnlyTheStarGame(t *testing.T) {
	saved := activeModes
	t.Cleanup(func() { activeModes = saved })
	activeModes = []Mode{ModeStar}
	m := layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	if r := homeLayout(m); r.modes[0].W != 0 || r.modeRow.H != 0 {
		t.Fatalf("unexpected mode row %+v", r.modeRow)
	}
}
