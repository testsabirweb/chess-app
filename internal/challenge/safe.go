package challenge

import (
	"math/rand/v2"

	"github.com/testsabirweb/chess-app/internal/chess"
)

const (
	safeLeg   = 3
	safeTries = 500
)

// safeMinMoves is the fewest moves the star may be from the piece. Two, so there
// is a route to plan - except for the pawn, which can only climb and often has
// no two-move route that dodges anything.
func safeMinMoves(pt chess.PieceType) int {
	if pt == chess.Pawn {
		return 1
	}
	return 2
}

// HotSet turns a puzzle's hot squares plus its guard into an avoid function:
// the piece may neither stop where the guard attacks nor take the guard.
func HotSet(hot []chess.Square, guard chess.Square) func(chess.Square) bool {
	set := make(map[chess.Square]bool, len(hot)+1)
	for _, s := range hot {
		set[s] = true
	}
	set[guard] = true
	return func(s chess.Square) bool { return set[s] }
}

// guardAttacks lists where a guard on `at` attacks, with the player's piece
// taken off the board. A piece moving along a line does not block that line
// for itself, and it keeps the picture fixed for the whole puzzle: what is
// shown is exactly what is enforced, wherever the piece wanders.
func guardAttacks(b *chess.Board, player, at chess.Square) []chess.Square {
	nb := b.Clone()
	nb.Set(player, chess.Piece{})
	return nb.Attacks(at)
}

// NewSafe deals a round of the Stay-safe game: one star, and a black guard that
// attacks some squares. The piece must reach the star without stopping on an
// attacked square. guards are the types the guard may be; mem is what recent
// rounds used, which the start, the star and the guard avoid. detour asks for a round where the safe way round is
// longer than the direct one - it is a preference, dropped if it cannot be met.
// It always returns a valid puzzle.
func NewSafe(rng *rand.Rand, pt chess.PieceType, color chess.Color, guards []chess.PieceType, mem *Memory, detour bool) Puzzle {
	for try := 0; try < safeTries; try++ {
		// The last third of the attempts drop the preferences.
		strict := try < safeTries*2/3
		if p, ok := trySafe(rng, pt, color, guards, mem, detour && strict, strict); ok {
			mem.Remember(p.From, p.Targets[0], p.Guard)
			return p
		}
	}
	p := fallbackSafe(pt, color)
	mem.Remember(p.From, p.Targets[0], p.Guard)
	return p
}

func trySafe(rng *rand.Rand, pt chess.PieceType, color chess.Color, guards []chess.PieceType, mem *Memory, detour, strict bool) (Puzzle, bool) {
	const size = 5
	from := chess.Sq(rng.IntN(size), rng.IntN(size))
	if strict && mem.Has(from) {
		return Puzzle{}, false
	}
	if pt == chess.Pawn && (from.Rank < 1 || from.Rank > size-2) {
		return Puzzle{}, false
	}
	guardAt := chess.Sq(rng.IntN(size), rng.IntN(size))
	if guardAt == from || (strict && mem.Has(guardAt)) {
		return Puzzle{}, false
	}
	// A pawn guard stands on a rank where it still has somewhere to look.
	gt := guards[rng.IntN(len(guards))]
	if gt == chess.Pawn && (guardAt.Rank < 1 || guardAt.Rank > size-2) {
		return Puzzle{}, false
	}

	b := chess.NewBoard(size, size)
	piece := chess.Piece{Type: pt, Color: color}
	b.Set(from, piece)
	b.Set(guardAt, chess.Piece{Type: gt, Color: color.Opponent()})
	hot := guardAttacks(b, from, guardAt)
	avoidFn := HotSet(hot, guardAt)
	if avoidFn(from) {
		return Puzzle{}, false
	}

	// Danger has to matter: at least one of the piece's first moves lands on a
	// hot square, or the guard is decoration.
	dangerous := false
	for _, to := range b.MoveTargets(from) {
		if avoidFn(to) {
			dangerous = true
			break
		}
	}
	if !dangerous {
		return Puzzle{}, false
	}

	var stars []Step
	for _, s := range ReachAvoiding(b, from, safeLeg, avoidFn) {
		if s.Moves < safeMinMoves(pt) || !b.At(s.Square).IsEmpty() {
			continue
		}
		if strict && mem.Has(s.Square) {
			continue
		}
		if detour {
			// The safe way round must be longer than going straight, ignoring
			// the guard's squares.
			if direct := MovesTo(b, from, s.Square, safeLeg); direct <= 0 || s.Moves <= direct {
				continue
			}
		}
		stars = append(stars, s)
	}
	if len(stars) == 0 {
		return Puzzle{}, false
	}
	star := stars[rng.IntN(len(stars))]
	return Puzzle{
		Board: b, From: from, Piece: piece,
		Targets: []chess.Square{star.Square}, Optimal: star.Moves,
		Guard: guardAt, Hot: hot,
	}, true
}

// fallbackSafe is the puzzle used if the random search somehow finds nothing:
// the same generator with a fixed seed and a rook for a guard, so it is valid by
// construction and the same every time.
func fallbackSafe(pt chess.PieceType, color chess.Color) Puzzle {
	rng := rand.New(rand.NewPCG(1, 1))
	for try := 0; try < 20000; try++ {
		if p, ok := trySafe(rng, pt, color, []chess.PieceType{chess.Rook}, nil, false, false); ok {
			return p
		}
	}
	// Not reachable for any real piece; an unguarded board is still a valid state.
	b := chess.NewBoard(5, 5)
	from := chess.Sq(2, 1)
	piece := chess.Piece{Type: pt, Color: color}
	b.Set(from, piece)
	return Puzzle{Board: b, From: from, Piece: piece, Targets: []chess.Square{Reach(b, from, 1)[0].Square}, Optimal: 1, Guard: chess.Sq(0, 0)}
}
