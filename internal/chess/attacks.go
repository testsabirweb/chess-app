package chess

import "sort"

var allDirs = append(append([]dir{}, orthoDirs...), diagDirs...)

// Attacks lists the squares the piece on `from` attacks: the squares it would
// be able to capture on if an enemy piece stood there. That is not the same as
// where it can move, in two ways:
//
//   - a pawn attacks its two forward diagonals whether or not anything stands
//     on them, and never the square straight ahead, which it can move to but
//     not capture on;
//   - a slider's line runs up to and including the first piece it meets, of
//     either colour, because that piece is defended.
//
// The result is sorted by (Rank, File). An empty square attacks nothing.
func (b *Board) Attacks(from Square) []Square {
	if !b.Contains(from) {
		return nil
	}
	p := b.At(from)
	var out []Square
	switch p.Type {
	case Pawn:
		fwd := 1
		if p.Color == Black {
			fwd = -1
		}
		for _, df := range []int{-1, 1} {
			if to := Sq(int(from.File)+df, int(from.Rank)+fwd); b.Contains(to) {
				out = append(out, to)
			}
		}
	case Knight:
		out = b.attackSteps(out, from, knightDirs)
	case King:
		out = b.attackSteps(out, from, allDirs)
	case Bishop:
		out = b.attackLines(out, from, diagDirs)
	case Rook:
		out = b.attackLines(out, from, orthoDirs)
	case Queen:
		out = b.attackLines(out, from, allDirs)
	default:
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		return out[i].File < out[j].File
	})
	return out
}

func (b *Board) attackSteps(dst []Square, from Square, dirs []dir) []Square {
	for _, d := range dirs {
		if to := Sq(int(from.File)+d.df, int(from.Rank)+d.dr); b.Contains(to) {
			dst = append(dst, to)
		}
	}
	return dst
}

func (b *Board) attackLines(dst []Square, from Square, dirs []dir) []Square {
	for _, d := range dirs {
		f, r := int(from.File)+d.df, int(from.Rank)+d.dr
		for {
			to := Sq(f, r)
			if !b.Contains(to) {
				break
			}
			dst = append(dst, to)
			if !b.At(to).IsEmpty() {
				break
			}
			f += d.df
			r += d.dr
		}
	}
	return dst
}
