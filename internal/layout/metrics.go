package layout

import (
	"math"

	"github.com/testsabirweb/chess-app/internal/chess"
)

const minTapDP = 48

type Metrics struct {
	W, H, Scale float64
	Safe, Header, Board, Footer Rect
	Cell, MinTap, TitleSize, BodySize float64
	Cols, Rows int
	Portrait bool
}

func dp(scale, v float64) float64 {
	return v * scale
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func Compute(w, h, scale float64, in Insets, cols, rows int) Metrics {
	if scale <= 0 {
		scale = 1
	}
	m := Metrics{
		W: w, H: h, Scale: scale,
		Cols: cols, Rows: rows,
		MinTap:   dp(scale, minTapDP),
		Portrait: h >= w,
	}

	top := math.Max(in.Top, dp(scale, 20))
	bottom := math.Max(in.Bottom, dp(scale, 20))
	left := math.Max(in.Left, dp(scale, 14))
	right := math.Max(in.Right, dp(scale, 14))

	m.Safe = Rect{
		X: left, Y: top,
		W: w - left - right,
		H: h - top - bottom,
	}

	gap := dp(scale, 8)

	if m.Portrait {
		computePortrait(&m, gap, scale)
	} else {
		computeLandscape(&m, gap)
	}

	m.Cell = m.Board.W / float64(cols)
	m.TitleSize = clamp(m.Cell*0.45, dp(scale, 20), dp(scale, 56))
	m.BodySize = clamp(m.Cell*0.28, dp(scale, 14), dp(scale, 32))
	return m
}

func computePortrait(m *Metrics, gap, scale float64) {
	hdrMin := dp(scale, 56)
	ftrMin := dp(scale, 72)

	side := math.Min(m.Safe.W, m.Safe.H-hdrMin-ftrMin-2*gap)
	side = math.Min(side, m.Safe.W)
	side = math.Min(side, m.Safe.H-2*gap)
	if side < 0 {
		side = 0
	}

	remain := m.Safe.H - side - 2*gap
	if remain < 0 {
		side = m.Safe.H - 2*gap
		if side > m.Safe.W {
			side = m.Safe.W
		}
		remain = 0
	}

	hdr := remain * 0.4
	ftr := remain * 0.6
	hdr = math.Min(hdr, dp(scale, 180))
	ftr = math.Min(ftr, dp(scale, 240))

	stackH := hdr + gap + side + gap + ftr
	startY := m.Safe.Y + (m.Safe.H-stackH)/2

	m.Header = Rect{X: m.Safe.X, Y: startY, W: m.Safe.W, H: hdr}
	m.Board = Rect{
		X: m.Safe.X + (m.Safe.W-side)/2,
		Y: startY + hdr + gap,
		W: side, H: side,
	}
	m.Footer = Rect{
		X: m.Safe.X,
		Y: m.Board.Y + m.Board.H + gap,
		W: m.Safe.W, H: ftr,
	}
}

func computeLandscape(m *Metrics, gap float64) {
	minSide := m.MinTap * float64(m.Cols)
	maxSide := math.Min(m.Safe.H-2*gap, m.Safe.W-2*gap)

	side := math.Min(m.Safe.H-2*gap, m.Safe.W*0.62)
	panelW := m.Safe.W - side - 2*gap
	if panelW < m.MinTap*3 {
		panelW = m.MinTap * 3
		side = math.Min(m.Safe.H-2*gap, m.Safe.W-panelW-2*gap)
	}
	if side < minSide && minSide <= maxSide {
		side = math.Min(minSide, m.Safe.H-2*gap)
		panelW = m.Safe.W - side - 2*gap
		if panelW < m.MinTap {
			side = math.Max(0, math.Min(m.Safe.H-2*gap, m.Safe.W-m.MinTap-2*gap))
			panelW = m.Safe.W - side - 2*gap
		}
	}
	if side < 0 {
		side = 0
	}
	if panelW < 0 {
		panelW = 0
	}

	m.Board = Rect{
		X: m.Safe.X + gap,
		Y: m.Safe.Y + (m.Safe.H-side)/2,
		W: side, H: side,
	}
	panelX := m.Board.X + side + gap
	m.Header = Rect{X: panelX, Y: m.Safe.Y, W: panelW, H: m.Safe.H * 0.35}
	m.Footer = Rect{X: panelX, Y: m.Safe.Y + m.Safe.H*0.45, W: panelW, H: m.Safe.H * 0.55}
}

func (m Metrics) CellRect(f, r int) Rect {
	cell := m.Cell
	x := m.Board.X + float64(f)*cell
	y := m.Board.Y + float64(m.Rows-1-r)*cell
	return Rect{X: x, Y: y, W: cell, H: cell}
}

func (m Metrics) HitCell(x, y float64) (file, rank int, ok bool) {
	if !m.Board.Contains(x, y) {
		return 0, 0, false
	}
	f := int((x - m.Board.X) / m.Cell)
	r := m.Rows - 1 - int((y-m.Board.Y)/m.Cell)
	if f < 0 || f >= m.Cols || r < 0 || r >= m.Rows {
		return 0, 0, false
	}
	return f, r, true
}

func (m Metrics) HitStar(x, y float64, star chess.Square, solutions []chess.Square) bool {
	cr := m.CellRect(int(star.File), int(star.Rank))
	cx, cy := cr.Center()
	dx, dy := x-cx, y-cy
	radius := m.Cell * 0.55
	if dx*dx+dy*dy > radius*radius {
		return false
	}
	if f, r, ok := m.HitCell(x, y); ok {
		sq := chess.Sq(f, r)
		if sq != star {
			for _, sol := range solutions {
				if sol == sq {
					return false
				}
			}
		}
	}
	return true
}

func (m Metrics) TapOK() bool {
	return m.Cell >= m.MinTap
}
