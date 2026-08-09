package layout

type Rect struct {
	X, Y, W, H float64
}

func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

func (r Rect) Center() (float64, float64) {
	return r.X + r.W/2, r.Y + r.H/2
}
