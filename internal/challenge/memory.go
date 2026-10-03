package challenge

import "github.com/testsabirweb/chess-app/internal/chess"

// memoryRounds is how many recent rounds a Memory keeps. Two, so a square used
// a moment ago does not come back as a start, a star or a pawn straight away -
// but the board is only 25 squares, so remembering much more would leave the
// generators nothing to choose from.
const memoryRounds = 2

// Memory remembers the squares the last few rounds used - where the piece
// started, where the stars or pawns were, where a guard stood - so the next
// round can look different. It is a preference, not a rule: every generator
// drops it rather than fail, and a nil Memory remembers nothing.
type Memory struct {
	rounds [memoryRounds][]chess.Square
	next   int
}

// Remember records one finished round's squares, forgetting the oldest round.
func (m *Memory) Remember(squares ...chess.Square) {
	if m == nil {
		return
	}
	m.rounds[m.next] = append(m.rounds[m.next][:0], squares...)
	m.next = (m.next + 1) % memoryRounds
}

// Has reports whether a recent round used the square.
func (m *Memory) Has(sq chess.Square) bool {
	if m == nil {
		return false
	}
	for _, r := range m.rounds {
		for _, s := range r {
			if s == sq {
				return true
			}
		}
	}
	return false
}

// HasAny reports whether a recent round used any of the squares.
func (m *Memory) HasAny(squares []chess.Square) bool {
	for _, s := range squares {
		if m.Has(s) {
			return true
		}
	}
	return false
}
