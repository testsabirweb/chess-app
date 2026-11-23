package chess

// The original games have no kings, so they only ever needed pseudo-legal
// moves: where a piece can go if the board is otherwise still. The puzzle game
// needs the real rules around the king, and this file adds exactly those.

// KingSquare returns where the king of colour c stands, or ok=false when the
// board has no such king (which is every board outside the puzzle game).
func (b *Board) KingSquare(c Color) (sq Square, ok bool) {
	for r := 0; r < b.height; r++ {
		for f := 0; f < b.width; f++ {
			s := Sq(f, r)
			if p := b.At(s); p.Type == King && p.Color == c {
				return s, true
			}
		}
	}
	return Square{}, false
}

// Attacked reports whether any piece of colour `by` attacks sq. A piece counts
// even when it could not legally move there (for example a pinned piece): that
// is the rule that keeps a king from stepping onto a defended square, and it is
// what Attacks already describes.
func (b *Board) Attacked(sq Square, by Color) bool {
	if !b.Contains(sq) {
		return false
	}
	for r := 0; r < b.height; r++ {
		for f := 0; f < b.width; f++ {
			from := Sq(f, r)
			p := b.At(from)
			if p.IsEmpty() || p.Color != by {
				continue
			}
			for _, a := range b.Attacks(from) {
				if a == sq {
					return true
				}
			}
		}
	}
	return false
}

// InCheck reports whether the king of colour c is attacked. A board with no
// king of that colour is never in check, so the games without kings are
// unaffected.
func (b *Board) InCheck(c Color) bool {
	sq, ok := b.KingSquare(c)
	if !ok {
		return false
	}
	return b.Attacked(sq, c.Opponent())
}

// After returns a copy of the board with m played. Callers that need the
// original untouched (searches, tests) use this rather than Set.
func (b *Board) After(m Move) *Board {
	nb := b.Clone()
	nb.Set(m.From, Piece{})
	nb.Set(m.To, b.At(m.From))
	return nb
}

// LegalMoves lists the moves of the piece on from that can actually be played:
// the pseudo-legal moves, minus those that would leave its own king in check
// (and, for the king itself, those onto a square an enemy piece attacks).
func (b *Board) LegalMoves(from Square) []Move {
	p := b.At(from)
	if p.IsEmpty() || !b.Contains(from) {
		return nil
	}
	var out []Move
	for _, m := range b.Moves(nil, from) {
		if !b.After(m).InCheck(p.Color) {
			out = append(out, m)
		}
	}
	return out
}

// LegalMovesFor lists every legal move of colour c on the board.
func (b *Board) LegalMovesFor(c Color) []Move {
	var out []Move
	for r := 0; r < b.height; r++ {
		for f := 0; f < b.width; f++ {
			s := Sq(f, r)
			if p := b.At(s); p.IsEmpty() || p.Color != c {
				continue
			}
			out = append(out, b.LegalMoves(s)...)
		}
	}
	return out
}

// Checkmate reports whether colour c is in check with no legal move left.
func (b *Board) Checkmate(c Color) bool {
	return b.InCheck(c) && len(b.LegalMovesFor(c)) == 0
}

// Stalemate reports whether it is colour c's turn, it is not in check, and it
// has no legal move.
func (b *Board) Stalemate(c Color) bool {
	return !b.InCheck(c) && len(b.LegalMovesFor(c)) == 0
}
