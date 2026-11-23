// Command genpuzzles searches small endgame positions for check, mate-in-one
// and mate-in-two puzzles and writes the bank the app embeds.
//
//	go run ./cmd/genpuzzles -out internal/puzzles/bank.txt
//
// It runs on a desktop, never on the phone: the search is cheap here and would
// cost time and memory on an old Android device. The output is reviewed text,
// verified again by the app's tests before it ships.
package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"sort"
	"strings"

	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/puzzles"
)

// material is a white army (a king is always added) and how many black pawns
// stand in its way. Black has only a king, so every puzzle is a forced mate.
type material struct {
	pieces []chess.PieceType
	pawns  int
}

var materials = []material{
	{pieces: []chess.PieceType{chess.Queen}},
	{pieces: []chess.PieceType{chess.Rook}},
	{pieces: []chess.PieceType{chess.Rook, chess.Rook}},
	{pieces: []chess.PieceType{chess.Bishop, chess.Rook}},
	{pieces: []chess.PieceType{chess.Queen}, pawns: 1},
	{pieces: []chess.PieceType{chess.Rook}, pawns: 1},
	{pieces: []chess.PieceType{chess.Queen}, pawns: 2},
	{pieces: []chess.PieceType{chess.Rook}, pawns: 2},
}

// targets is how many puzzles of each tier to aim for. These counts keep the
// bank a few kilobytes.
var targets = map[puzzles.Tier]int{
	puzzles.Check:     60,
	puzzles.Mate1:     100,
	puzzles.Mate1Trap: 80,
	puzzles.Mate2:     60,
}

// generateOrder runs the hardest, rarest kinds first, so a position that could
// be more than one tier is claimed by the one that is harder to find.
var generateOrder = []puzzles.Tier{
	puzzles.Mate2, puzzles.Mate1Trap, puzzles.Mate1, puzzles.Check,
}

const maxAttempts = 400000

func main() {
	out := flag.String("out", "internal/puzzles/bank.txt", "where to write the bank")
	seed := flag.Uint64("seed", 20261003, "random seed")
	flag.Parse()

	rng := rand.New(rand.NewPCG(*seed, *seed^0x9E3779B97F4A7C15))
	used := map[string]puzzles.Tier{} // canonical position -> tier that claimed it
	found := map[puzzles.Tier][]puzzles.Puzzle{}

	for _, tier := range generateOrder {
		want := targets[tier]
		for try := 0; try < maxAttempts && len(found[tier]) < want; try++ {
			b := randomPosition(rng)
			if b == nil {
				continue
			}
			m, ok := candidate(b, tier, rng)
			if !ok {
				continue
			}
			p := puzzles.Puzzle{Tier: tier, Board: b, Solution: m}
			if err := p.Valid(); err != nil {
				continue
			}
			key := puzzles.PositionKey(b)
			if _, dup := used[key]; dup {
				continue
			}
			used[key] = tier
			found[tier] = append(found[tier], p)
		}
		fmt.Fprintf(os.Stderr, "%s: %d/%d after search\n", tier, len(found[tier]), want)
		if len(found[tier]) == 0 {
			fail("no %s puzzles generated", tier)
		}
	}

	var b strings.Builder
	b.WriteString("# Checkmate puzzle bank. Built by cmd/genpuzzles; every line is verified by\n")
	b.WriteString("# internal/puzzles tests. Format: tier | 5 rows, rank 4 down to rank 0 | solution\n")
	b.WriteString("# (upper case white, lower case black, '.' empty).\n")
	for _, tier := range puzzles.Tiers {
		list := found[tier]
		sort.Slice(list, func(i, j int) bool {
			return puzzles.PositionKey(list[i].Board) < puzzles.PositionKey(list[j].Board)
		})
		fmt.Fprintf(&b, "\n# %s (%d)\n", tier, len(list))
		for _, p := range list {
			b.WriteString(puzzles.Encode(p))
			b.WriteByte('\n')
		}
	}
	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		fail("%v", err)
	}
	total := 0
	for _, n := range found {
		total += len(n)
	}
	fmt.Fprintf(os.Stderr, "wrote %d puzzles to %s\n", total, *out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// randomPosition places a black king on an edge (mates are easier to see), a
// white king and one material set, plus black pawns on ranks where they cannot
// promote in the coming reply. It returns nil when the placement breaks a rule.
func randomPosition(rng *rand.Rand) *chess.Board {
	const size = 5
	mat := materials[rng.IntN(len(materials))]
	b := chess.NewBoard(size, size)

	blackKing := edgeSquare(rng, size)
	b.Set(blackKing, chess.Piece{Type: chess.King, Color: chess.Black})

	whiteKing := freeSquare(rng, size, b)
	b.Set(whiteKing, chess.Piece{Type: chess.King, Color: chess.White})

	for _, t := range mat.pieces {
		s := freeSquare(rng, size, b)
		if isZero(s) {
			return nil
		}
		b.Set(s, chess.Piece{Type: t, Color: chess.White})
	}
	for i := 0; i < mat.pawns; i++ {
		// Ranks 2 and 3 only: a black pawn on rank 1 could promote during the
		// reply, and the game has no promotion yet.
		var spots []chess.Square
		for r := 2; r <= 3; r++ {
			for f := 0; f < size; f++ {
				if sq := chess.Sq(f, r); b.At(sq).IsEmpty() {
					spots = append(spots, sq)
				}
			}
		}
		if len(spots) == 0 {
			return nil
		}
		s := spots[rng.IntN(len(spots))]
		b.Set(s, chess.Piece{Type: chess.Pawn, Color: chess.Black})
	}

	if b.InCheck(chess.Black) || b.InCheck(chess.White) {
		return nil
	}
	if b.Checkmate(chess.Black) || b.Stalemate(chess.Black) {
		return nil
	}
	if b.Checkmate(chess.White) || b.Stalemate(chess.White) {
		return nil
	}
	return b
}

func edgeSquare(rng *rand.Rand, n int) chess.Square {
	if rng.IntN(2) == 0 {
		f := 0
		if rng.IntN(2) == 1 {
			f = n - 1
		}
		return chess.Sq(f, rng.IntN(n))
	}
	r := 0
	if rng.IntN(2) == 1 {
		r = n - 1
	}
	return chess.Sq(rng.IntN(n), r)
}

func freeSquare(rng *rand.Rand, n int, b *chess.Board) chess.Square {
	for i := 0; i < 40; i++ {
		s := chess.Sq(rng.IntN(n), rng.IntN(n))
		if b.At(s).IsEmpty() {
			return s
		}
	}
	return chess.Square{File: -1, Rank: -1}
}

func isZero(s chess.Square) bool { return s.File < 0 || s.Rank < 0 }

// candidate finds the move the app will show as the intended answer, and
// reports whether the position really is the tier it is being tried for.
func candidate(b *chess.Board, tier puzzles.Tier, rng *rand.Rand) (chess.Move, bool) {
	white, black := chess.White, chess.Black
	mates := puzzles.MateMoves(b, white)
	checks := puzzles.CheckMoves(b, white)
	switch tier {
	case puzzles.Check:
		// A position with no mate: the lesson is the check itself.
		if len(mates) != 0 || len(checks) == 0 {
			return chess.Move{}, false
		}
		return checks[rng.IntN(len(checks))], true
	case puzzles.Mate1:
		if len(mates) == 1 && len(checks) == 1 {
			return mates[0], true
		}
	case puzzles.Mate1Trap:
		if len(mates) == 1 && len(checks) >= 2 {
			return mates[0], true
		}
	case puzzles.Mate2:
		if len(mates) != 0 {
			return chess.Move{}, false
		}
		rng.Shuffle(len(checks), func(i, j int) { checks[i], checks[j] = checks[j], checks[i] })
		for _, m := range checks {
			after := b.After(m)
			if after.Checkmate(black) || after.Stalemate(black) {
				continue
			}
			replies := after.LegalMovesFor(black)
			if len(replies) == 0 {
				continue
			}
			forced := true
			for _, r := range replies {
				if len(puzzles.MateMoves(after.After(r), white)) == 0 {
					forced = false
					break
				}
			}
			if forced {
				return m, true
			}
		}
	}
	return chess.Move{}, false
}
