package challenge

import (
	"math/rand/v2"

	"github.com/testsabirweb/chess-app/internal/chess"
)

const (
	// catchMaxTour caps how many moves the shortest way to take every pawn may
	// take; longer is a slog at this age.
	catchMaxTour = 5
	catchLeg     = 3
	catchTries   = 300
)

// NewCatch deals a round for the piece pt with k black pawns to capture. The
// pawns are real pieces on the board and never move. Like NewTreasure it
// settles for fewer pawns rather than loop forever and always returns a valid
// puzzle; avoid is the previous round's start square.
func NewCatch(rng *rand.Rand, pt chess.PieceType, color chess.Color, k int, avoid chess.Square) Puzzle {
	for ; k >= 1; k-- {
		for try := 0; try < catchTries; try++ {
			if p, ok := tryCatch(rng, pt, color, k, avoid, try < catchTries*2/3); ok {
				return p
			}
		}
	}
	return fallbackCatch(pt, color)
}

// pawnHome reports whether a black pawn may stand on this rank. Pawns on the
// first or last rank would look odd to anyone who knows chess.
func pawnHome(rank int) bool { return rank >= 1 && rank <= 3 }

func tryCatch(rng *rand.Rand, pt chess.PieceType, color chess.Color, k int, avoid chess.Square, strict bool) (Puzzle, bool) {
	const size = 5
	from := chess.Sq(rng.IntN(size), rng.IntN(size))
	if strict && from == avoid {
		return Puzzle{}, false
	}
	if pt == chess.Pawn && (from.Rank < 1 || from.Rank > size-2) {
		return Puzzle{}, false
	}
	b := chess.NewBoard(size, size)
	piece := chess.Piece{Type: pt, Color: color}
	b.Set(from, piece)

	var spots []chess.Square
	for r := 1; r <= 3; r++ {
		for f := 0; f < size; f++ {
			if sq := chess.Sq(f, r); sq != from {
				spots = append(spots, sq)
			}
		}
	}
	if len(spots) < k {
		return Puzzle{}, false
	}
	pawns := make([]chess.Square, 0, k)
	enemy := chess.Piece{Type: chess.Pawn, Color: color.Opponent()}
	for _, i := range rng.Perm(len(spots))[:k] {
		pawns = append(pawns, spots[i])
		b.Set(spots[i], enemy)
	}
	total, ok := Tour(b, from, pawns, catchLeg)
	if !ok || total < k || total > catchMaxTour {
		return Puzzle{}, false
	}
	return Puzzle{Board: b, From: from, Piece: piece, Targets: pawns, Optimal: total}, true
}

// fallbackCatch is a valid puzzle for any piece: one pawn on the first square
// the piece can capture, trying each spot in turn.
func fallbackCatch(pt chess.PieceType, color chess.Color) Puzzle {
	const size = 5
	from := chess.Sq(2, 1)
	piece := chess.Piece{Type: pt, Color: color}
	enemy := chess.Piece{Type: chess.Pawn, Color: color.Opponent()}
	for r := 1; r <= 3; r++ {
		for f := 0; f < size; f++ {
			sq := chess.Sq(f, r)
			if sq == from {
				continue
			}
			b := chess.NewBoard(size, size)
			b.Set(from, piece)
			b.Set(sq, enemy)
			if total, ok := Tour(b, from, []chess.Square{sq}, catchLeg); ok {
				return Puzzle{Board: b, From: from, Piece: piece, Targets: []chess.Square{sq}, Optimal: total}
			}
		}
	}
	// Unreachable for any real piece; an empty board is still a valid state.
	b := chess.NewBoard(size, size)
	b.Set(from, piece)
	return Puzzle{Board: b, From: from, Piece: piece}
}
