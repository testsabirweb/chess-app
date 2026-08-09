package chess

import "sort"

type Board struct {
	width, height int
	cells         []Piece
}

func NewBoard(width, height int) *Board {
	if width <= 0 || height <= 0 {
		panic("chess: invalid board dimensions")
	}
	return &Board{
		width:  width,
		height: height,
		cells:  make([]Piece, width*height),
	}
}

func (b *Board) Width() int  { return b.width }
func (b *Board) Height() int { return b.height }

func (b *Board) index(s Square) int {
	return int(s.Rank)*b.width + int(s.File)
}

func (b *Board) Contains(s Square) bool {
	return int(s.File) >= 0 && int(s.File) < b.width &&
		int(s.Rank) >= 0 && int(s.Rank) < b.height
}

func (b *Board) At(s Square) Piece {
	if !b.Contains(s) {
		return Piece{}
	}
	return b.cells[b.index(s)]
}

func (b *Board) Set(s Square, p Piece) {
	if !b.Contains(s) {
		panic("chess: set off-board")
	}
	b.cells[b.index(s)] = p
}

func (b *Board) Clone() *Board {
	clone := NewBoard(b.width, b.height)
	copy(clone.cells, b.cells)
	return clone
}

func (b *Board) Occupied() []Square {
	out := make([]Square, 0, len(b.cells))
	for r := 0; r < b.height; r++ {
		for f := 0; f < b.width; f++ {
			s := Sq(f, r)
			if !b.At(s).IsEmpty() {
				out = append(out, s)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		return out[i].File < out[j].File
	})
	return out
}
