package challenge

import (
	"sort"

	"github.com/testsabirweb/chess-app/internal/chess"
)

// Step is one entry of a reachability map: a square the piece can get to, and
// the smallest number of legal moves it takes to get there.
type Step struct {
	Square chess.Square
	Moves  int
}

// Reach breadth-first-searches every square the piece standing on `from` can
// travel to within maxMoves legal moves. The piece is walked across a cloned
// board, so sliders are correctly re-evaluated from each intermediate square.
//
// The result is sorted by (Rank, File) and never contains `from`, so callers
// get deterministic output regardless of map iteration order.
func Reach(b *chess.Board, from chess.Square, maxMoves int) []Step {
	if b == nil || maxMoves <= 0 || !b.Contains(from) {
		return nil
	}
	piece := b.At(from)
	if piece.IsEmpty() {
		return nil
	}

	dist := map[chess.Square]int{from: 0}
	frontier := []chess.Square{from}
	out := make([]Step, 0, b.Width()*b.Height())

	for depth := 1; depth <= maxMoves && len(frontier) > 0; depth++ {
		var next []chess.Square
		for _, sq := range frontier {
			nb := b.Clone()
			if sq != from {
				nb.Set(from, chess.Piece{})
				nb.Set(sq, piece)
			}
			for _, to := range nb.MoveTargets(sq) {
				if _, seen := dist[to]; seen {
					continue
				}
				dist[to] = depth
				out = append(out, Step{Square: to, Moves: depth})
				next = append(next, to)
			}
		}
		frontier = next
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Square.Rank != out[j].Square.Rank {
			return out[i].Square.Rank < out[j].Square.Rank
		}
		return out[i].Square.File < out[j].Square.File
	})
	return out
}

// MovesTo reports the shortest number of moves from `from` to `target`.
// It returns 0 when they are the same square and -1 when the target cannot be
// reached within maxMoves.
func MovesTo(b *chess.Board, from, target chess.Square, maxMoves int) int {
	if from == target {
		return 0
	}
	for _, s := range Reach(b, from, maxMoves) {
		if s.Square == target {
			return s.Moves
		}
	}
	return -1
}

// CanReach is the boolean form of MovesTo.
func CanReach(b *chess.Board, from, target chess.Square, maxMoves int) bool {
	return MovesTo(b, from, target, maxMoves) > 0
}
