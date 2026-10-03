package challenge

import (
	"math/rand/v2"

	"github.com/testsabirweb/chess-app/internal/chess"
)

// Which is one "who can reach the star?" round: several white pieces on the
// board and a target, of which exactly one piece can get there in a single move.
type Which struct {
	Board  *chess.Board
	Pieces []chess.Square // where each candidate stands
	Answer chess.Square   // the candidate that reaches Target
	Target chess.Square
}

const (
	whichSize = 5
	// whichAttempts bounds the random search for a valid layout at one piece
	// count before it settles for fewer pieces.
	whichAttempts = 400
	// whichStrict is how many of those attempts insist on dodging `avoid` and
	// the squares in memory; the
	// rest accept it, so a small pool can never get stuck.
	whichStrict = 300
)

// NewWhich deals a round with n distinct piece types from pool. The answer is
// not of type avoid when that can be helped, so the same piece is not the
// right pick round after round. It settles for fewer pieces rather than loop
// forever, and always returns a valid round.
func NewWhich(rng *rand.Rand, pool []chess.PieceType, n int, avoid chess.PieceType, mem *Memory) Which {
	if n > len(pool) {
		n = len(pool)
	}
	for ; n >= 2; n-- {
		for try := 0; try < whichAttempts; try++ {
			if w, ok := tryWhich(rng, pool, n, avoid, mem, try < whichStrict); ok {
				mem.Remember(append(append([]chess.Square{}, w.Pieces...), w.Target)...)
				return w
			}
		}
	}
	w := fallbackWhich()
	mem.Remember(append(append([]chess.Square{}, w.Pieces...), w.Target)...)
	return w
}

func tryWhich(rng *rand.Rand, pool []chess.PieceType, n int, avoid chess.PieceType, mem *Memory, strict bool) (Which, bool) {
	b := chess.NewBoard(whichSize, whichSize)
	types := make([]chess.PieceType, 0, n)
	for _, i := range rng.Perm(len(pool))[:n] {
		types = append(types, pool[i])
	}

	squares := make([]chess.Square, 0, n)
	for _, t := range types {
		placed := false
		for k := 0; k < 60 && !placed; k++ {
			sq := chess.Sq(rng.IntN(whichSize), rng.IntN(whichSize))
			// A pawn off its home rank or on the far one breaks the double-push
			// rule or has nowhere to go; see Generator.startOK.
			if t == chess.Pawn && (sq.Rank < 1 || sq.Rank > 3) {
				continue
			}
			if !b.At(sq).IsEmpty() {
				continue
			}
			b.Set(sq, chess.Piece{Type: t, Color: chess.White})
			squares = append(squares, sq)
			placed = true
		}
		if !placed {
			return Which{}, false
		}
		if strict && mem.Has(squares[len(squares)-1]) {
			return Which{}, false
		}
	}

	// Who can reach each empty square. All the candidates are on the board, so
	// they block one another exactly as real pieces would.
	reachedBy := map[chess.Square][]int{}
	for i, sq := range squares {
		for _, to := range b.MoveTargets(sq) {
			reachedBy[to] = append(reachedBy[to], i)
		}
	}
	var targets []chess.Square
	var owners []int
	for to, by := range reachedBy {
		if len(by) != 1 || !b.At(to).IsEmpty() {
			continue
		}
		if strict && (b.At(squares[by[0]]).Type == avoid || mem.Has(to)) {
			continue
		}
		targets = append(targets, to)
		owners = append(owners, by[0])
	}
	if len(targets) == 0 {
		return Which{}, false
	}
	// Map order is random; sort so a seed always gives the same round.
	sortSquaresWith(targets, owners)
	k := rng.IntN(len(targets))
	return Which{Board: b, Pieces: squares, Answer: squares[owners[k]], Target: targets[k]}, true
}

// sortSquaresWith orders squares by (Rank, File), carrying owners along.
func sortSquaresWith(sq []chess.Square, owners []int) {
	for i := 1; i < len(sq); i++ {
		for j := i; j > 0 && squareLess(sq[j], sq[j-1]); j-- {
			sq[j], sq[j-1] = sq[j-1], sq[j]
			owners[j], owners[j-1] = owners[j-1], owners[j]
		}
	}
}

func squareLess(a, b chess.Square) bool {
	if a.Rank != b.Rank {
		return a.Rank < b.Rank
	}
	return a.File < b.File
}

// fallbackWhich is the round used if the random search somehow finds nothing:
// a rook and a king, with the star straight up the rook's file.
func fallbackWhich() Which {
	b := chess.NewBoard(whichSize, whichSize)
	rook, king := chess.Sq(0, 0), chess.Sq(4, 4)
	b.Set(rook, chess.Piece{Type: chess.Rook, Color: chess.White})
	b.Set(king, chess.Piece{Type: chess.King, Color: chess.White})
	return Which{Board: b, Pieces: []chess.Square{rook, king}, Answer: rook, Target: chess.Sq(0, 3)}
}
