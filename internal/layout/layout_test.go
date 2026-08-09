package layout_test

import (
	"math"
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
)

type device struct {
	name string
	w, h, scale float64
}

var devices = []device{
	{"edge50", 1080, 2400, 2.75},
	{"flagship", 1440, 3120, 3.5},
	{"budget", 720, 1600, 2.0},
	{"dev", 432, 960, 1.0},
	{"landscape", 800, 400, 1.0},
}

func TestComputeDevices(t *testing.T) {
	for _, d := range devices {
		t.Run(d.name, func(t *testing.T) {
			m := layout.Compute(d.w, d.h, d.scale, layout.Insets{}, 5, 5)
			if math.Abs(m.Board.W-m.Board.H) > 1e-9 {
				t.Fatalf("board not square: %v", m.Board)
			}
			if m.Board.X < m.Safe.X || m.Board.Y < m.Safe.Y {
				t.Fatal("board outside safe area")
			}
			if m.Header.Y+m.Header.H > m.Board.Y+1e-9 {
				t.Fatal("header overlaps board")
			}
			if m.Board.Y+m.Board.H > m.Footer.Y+1e-9 {
				t.Fatal("board overlaps footer")
			}
			stack := m.Footer.Y + m.Footer.H - m.Header.Y
			if stack > m.Safe.H+1e-9 {
				t.Fatalf("stack taller than safe: %f > %f", stack, m.Safe.H)
			}
			if math.Abs(m.Cell*5-m.Board.W) > 1e-9 {
				t.Fatalf("cell*cols != board width")
			}
			if d.name != "landscape" && !m.TapOK() {
				t.Fatalf("tap target too small: cell=%f minTap=%f", m.Cell, m.MinTap)
			}
		})
	}
}

func TestHitCellRoundTrip(t *testing.T) {
	m := layout.Compute(432, 960, 1, layout.Insets{}, 5, 5)
	for r := 0; r < 5; r++ {
		for f := 0; f < 5; f++ {
			cr := m.CellRect(f, r)
			cx, cy := cr.Center()
			ff, rr, ok := m.HitCell(cx, cy)
			if !ok || ff != f || rr != r {
				t.Fatalf("hit (%d,%d) got (%d,%d,%v)", f, r, ff, rr, ok)
			}
		}
	}
}

func TestHeaderFooterNotBoard(t *testing.T) {
	m := layout.Compute(432, 960, 1, layout.Insets{}, 5, 5)
	cx, cy := m.Header.Center()
	if _, _, ok := m.HitCell(cx, cy); ok {
		t.Fatal("header tap should miss board")
	}
	cx, cy = m.Footer.Center()
	if _, _, ok := m.HitCell(cx, cy); ok {
		t.Fatal("footer tap should miss board")
	}
}

func TestHitStarMagnet(t *testing.T) {
	m := layout.Compute(432, 960, 1, layout.Insets{}, 5, 5)
	star := chess.Sq(2, 2)
	solutions := []chess.Square{star, chess.Sq(0, 2)}
	cr := m.CellRect(2, 2)
	cx, cy := cr.Center()
	if !m.HitStar(cx+m.Cell*0.4, cy, star, solutions) {
		t.Fatal("near miss should count")
	}
	other := m.CellRect(0, 2)
	ox, oy := other.Center()
	if m.HitStar(ox, oy, star, solutions) {
		t.Fatal("should not steal tap on different solution square")
	}
}
