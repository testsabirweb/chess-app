package challenge

import (
	"math/rand/v2"

	"github.com/testsabirweb/chess-app/internal/chess"
)

// Puzzle is one round of a single-piece game: one piece, and one or more
// targets to land on. For the star games the targets are empty squares; for
// capture games they hold pieces that landing on removes.
type Puzzle struct {
	Board   *chess.Board
	From    chess.Square
	Piece   chess.Piece
	Targets []chess.Square
	// Optimal is the fewest moves that clear every target, in any order.
	Optimal int
	// Guard and Hot are only set by the Stay-safe game: the enemy piece that
	// does not move, and the squares it attacks, where the piece must not stop.
	Guard chess.Square
	Hot   []chess.Square
}

// Tour reports the fewest moves for the piece on `from` to land on every
// target, visiting them in whichever order is quickest. Each leg may take up
// to legMax moves. ok is false when some order cannot be completed at all.
//
// The board is replayed leg by leg: the piece leaves its square, and a target
// that holds a piece is captured as the piece lands, so lines open up and
// close exactly as they would in play.
func Tour(b *chess.Board, from chess.Square, targets []chess.Square, legMax int) (moves int, ok bool) {
	return TourAvoiding(b, from, targets, legMax, nil)
}

// TourAvoiding is Tour for a piece that never stops on an avoided square.
func TourAvoiding(b *chess.Board, from chess.Square, targets []chess.Square, legMax int, avoid func(chess.Square) bool) (moves int, ok bool) {
	if b == nil || len(targets) == 0 {
		return 0, true
	}
	piece := b.At(from)
	if piece.IsEmpty() {
		return 0, false
	}
	best := -1
	order := make([]chess.Square, len(targets))
	var try func(board *chess.Board, at chess.Square, left []chess.Square, spent int)
	try = func(board *chess.Board, at chess.Square, left []chess.Square, spent int) {
		if best >= 0 && spent >= best {
			return
		}
		if len(left) == 0 {
			best = spent
			return
		}
		for i, t := range left {
			d := MovesToAvoiding(board, at, t, legMax, avoid)
			if d <= 0 {
				continue
			}
			nb := board.Clone()
			nb.Set(at, chess.Piece{})
			nb.Set(t, piece)
			rest := make([]chess.Square, 0, len(left)-1)
			rest = append(rest, left[:i]...)
			rest = append(rest, left[i+1:]...)
			try(nb, t, rest, spent+d)
		}
	}
	copy(order, targets)
	try(b, from, order, 0)
	if best < 0 {
		return 0, false
	}
	return best, true
}

const (
	// treasureMaxTour caps how many moves the shortest way through all the stars
	// may take. Longer tours are a slog at this age.
	treasureMaxTour = 5
	// treasureLeg is the longest single leg a star may be from the piece.
	treasureLeg = 3
	// treasureAttempts bounds the random search at one star count before it
	// settles for fewer stars.
	treasureAttempts = 300
)

// NewTreasure deals a round for the piece pt with k stars to collect. It
// settles for fewer stars rather than loop forever (a pawn can only climb, so
// it often cannot reach three), and always returns a valid puzzle. mem is what
// recent rounds used: the start and the stars avoid those squares for most of
// the attempts, never all of them.
func NewTreasure(rng *rand.Rand, pt chess.PieceType, color chess.Color, k int, mem *Memory) Puzzle {
	for ; k >= 1; k-- {
		for try := 0; try < treasureAttempts; try++ {
			if p, ok := tryTreasure(rng, pt, color, k, mem, try < treasureAttempts*2/3); ok {
				mem.Remember(append([]chess.Square{p.From}, p.Targets...)...)
				return p
			}
		}
	}
	p := fallbackTreasure(pt, color)
	mem.Remember(append([]chess.Square{p.From}, p.Targets...)...)
	return p
}

func tryTreasure(rng *rand.Rand, pt chess.PieceType, color chess.Color, k int, mem *Memory, strict bool) (Puzzle, bool) {
	const size = 5
	from := chess.Sq(rng.IntN(size), rng.IntN(size))
	if strict && mem.Has(from) {
		return Puzzle{}, false
	}
	// A pawn behind its home rank could double-push twice; see Generator.startOK.
	if pt == chess.Pawn && (from.Rank < 1 || from.Rank > size-2) {
		return Puzzle{}, false
	}
	b := chess.NewBoard(size, size)
	piece := chess.Piece{Type: pt, Color: color}
	b.Set(from, piece)

	var pool []chess.Square
	for _, s := range Reach(b, from, treasureLeg) {
		pool = append(pool, s.Square)
	}
	if len(pool) < k {
		return Puzzle{}, false
	}
	picked := make([]chess.Square, 0, k)
	for _, i := range rng.Perm(len(pool))[:k] {
		picked = append(picked, pool[i])
	}
	if strict && mem.HasAny(picked) {
		return Puzzle{}, false
	}
	total, ok := Tour(b, from, picked, treasureLeg)
	// A tour with fewer moves than stars would mean one landing collected two,
	// which cannot happen; this is just a guard.
	if !ok || total < k || total > treasureMaxTour {
		return Puzzle{}, false
	}
	return Puzzle{Board: b, From: from, Piece: piece, Targets: picked, Optimal: total}, true
}

// fallbackTreasure is a puzzle that is always valid for any piece: one star,
// one move from the piece's start.
func fallbackTreasure(pt chess.PieceType, color chess.Color) Puzzle {
	b := chess.NewBoard(5, 5)
	from := chess.Sq(2, 1)
	piece := chess.Piece{Type: pt, Color: color}
	b.Set(from, piece)
	steps := Reach(b, from, 1)
	t := steps[0].Square
	return Puzzle{Board: b, From: from, Piece: piece, Targets: []chess.Square{t}, Optimal: 1}
}
