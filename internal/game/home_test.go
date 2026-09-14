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
