// Package puzzles holds the hand-checked checkmate puzzles the app deals.
//
// The puzzles are not generated on the phone. A small bank is built offline by
// cmd/genpuzzles, every line is verified by the rules below, and the result is
// embedded in the app. Running the search on an old phone, on every round,
// would be slow and would use memory the game does not need to use.
package puzzles

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	"github.com/testsabirweb/chess-app/internal/chess"
)

// Tier is the kind of question a puzzle asks. They get harder in this order.
type Tier string

const (
	// Check asks for any move that attacks the black king. It is the first
	// step: the point is to learn what check looks like.
	Check Tier = "check"
	// Mate1 has exactly one mating move, and it is the only check available.
	Mate1 Tier = "m1"
	// Mate1Trap has exactly one mating move plus at least one check that is not
	// mate, the classic beginner trap.
	Mate1Trap Tier = "m1trap"
	// Mate2 needs two white moves: black must reply in between, and every reply
	// leaves a mate.
	Mate2 Tier = "m2"
)

// Tiers lists them from the first to the hardest.
var Tiers = []Tier{Check, Mate1, Mate1Trap, Mate2}

// Puzzle is one position, white to move, with the intended first move.
type Puzzle struct {
	Tier     Tier
	Board    *chess.Board
	Solution chess.Move
}

//go:embed bank.txt
var bankFS embed.FS

var (
	bankOnce sync.Once
	bank     []Puzzle
	bankErr  error
)

// Bank returns every puzzle, in tier order. It is parsed once.
func Bank() []Puzzle {
	bankOnce.Do(func() {
		data, err := bankFS.ReadFile("bank.txt")
		if err != nil {
			bankErr = err
			return
		}
		bank, bankErr = Load(string(data))
	})
	return bank
}

// ByTier groups the bank by tier, keeping the bank's order within each tier.
func ByTier() map[Tier][]Puzzle {
	out := map[Tier][]Puzzle{}
	for _, p := range Bank() {
		out[p.Tier] = append(out[p.Tier], p)
	}
	return out
}

// Load parses a bank: comment lines start with #, and every other non-empty
// line is "tier | rows | solution", where rows are ranks 4 down to 0, five
// characters each, "." empty, uppercase white, lowercase black, and the
// solution is like "b4-b1".
func Load(text string) ([]Puzzle, error) {
	var out []Puzzle
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := Decode(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty bank")
	}
	return out, nil
}

// Decode parses one bank line.
func Decode(line string) (Puzzle, error) {
	parts := strings.Split(line, "|")
	if len(parts) != 3 {
		return Puzzle{}, fmt.Errorf("want tier | rows | solution, got %q", line)
	}
	tier := Tier(strings.TrimSpace(parts[0]))
	found := false
	for _, t := range Tiers {
		found = found || t == tier
	}
	if !found {
		return Puzzle{}, fmt.Errorf("unknown tier %q", tier)
	}
	rows := strings.Split(strings.TrimSpace(parts[1]), "/")
	if len(rows) != 5 {
		return Puzzle{}, fmt.Errorf("want 5 rows, got %d", len(rows))
	}
	b := chess.NewBoard(5, 5)
	for ri, row := range rows {
		row = strings.TrimSpace(row)
		if len(row) != 5 {
			return Puzzle{}, fmt.Errorf("row %q is not 5 squares", row)
		}
		rank := 4 - ri
		for f, ch := range row {
			p, err := pieceFor(ch)
			if err != nil {
				return Puzzle{}, err
			}
			if !p.IsEmpty() {
				b.Set(chess.Sq(f, rank), p)
			}
		}
	}
	m, err := decodeMove(strings.TrimSpace(parts[2]))
	if err != nil {
		return Puzzle{}, err
	}
	return Puzzle{Tier: tier, Board: b, Solution: m}, nil
}

func pieceFor(ch rune) (chess.Piece, error) {
	if ch == '.' {
		return chess.Piece{}, nil
	}
	white := ch >= 'A' && ch <= 'Z'
	up := ch
	if !white {
		up = ch - 'a' + 'A'
	}
	var t chess.PieceType
	switch up {
	case 'K':
		t = chess.King
	case 'Q':
		t = chess.Queen
	case 'R':
		t = chess.Rook
	case 'B':
		t = chess.Bishop
	case 'N':
		t = chess.Knight
	case 'P':
		t = chess.Pawn
	default:
		return chess.Piece{}, fmt.Errorf("unknown piece %q", ch)
	}
	c := chess.Black
	if white {
		c = chess.White
	}
	return chess.Piece{Type: t, Color: c}, nil
}

func decodeMove(s string) (chess.Move, error) {
	bits := strings.Split(s, "-")
	if len(bits) != 2 || len(bits[0]) != 2 || len(bits[1]) != 2 {
		return chess.Move{}, fmt.Errorf("bad move %q", s)
	}
	from, err := decodeSquare(bits[0])
	if err != nil {
		return chess.Move{}, err
	}
	to, err := decodeSquare(bits[1])
	if err != nil {
		return chess.Move{}, err
	}
	return chess.Move{From: from, To: to}, nil
}

func decodeSquare(s string) (chess.Square, error) {
	f := int(s[0] - 'a')
	r := int(s[1] - '0')
	if f < 0 || f > 4 || r < 0 || r > 4 {
		return chess.Square{}, fmt.Errorf("square %q off the board", s)
	}
	return chess.Sq(f, r), nil
}

func squareName(s chess.Square) string {
	return string(rune('a'+int(s.File))) + string(rune('0'+int(s.Rank)))
}

// Encode writes one bank line.
func Encode(p Puzzle) string {
	var rows []string
	for rank := 4; rank >= 0; rank-- {
		var row strings.Builder
		for f := 0; f < 5; f++ {
			row.WriteRune(runeFor(p.Board.At(chess.Sq(f, rank))))
		}
		rows = append(rows, row.String())
	}
	return fmt.Sprintf("%s | %s | %s-%s", p.Tier, strings.Join(rows, " / "),
		squareName(p.Solution.From), squareName(p.Solution.To))
}

func runeFor(p chess.Piece) rune {
	if p.IsEmpty() {
		return '.'
	}
	var ch rune
	switch p.Type {
	case chess.King:
		ch = 'K'
	case chess.Queen:
		ch = 'Q'
	case chess.Rook:
		ch = 'R'
	case chess.Bishop:
		ch = 'B'
	case chess.Knight:
		ch = 'N'
	case chess.Pawn:
		ch = 'P'
	default:
		return '?'
	}
	if p.Color == chess.Black {
		ch = ch - 'A' + 'a'
	}
	return ch
}

// PositionKey is the position with files mirrored where that makes it smaller,
// so a puzzle and its mirror image count as one.
func PositionKey(b *chess.Board) string {
	forward := make([]rune, 0, 25)
	mirror := make([]rune, 0, 25)
	for rank := 4; rank >= 0; rank-- {
		for f := 0; f < 5; f++ {
			forward = append(forward, runeFor(b.At(chess.Sq(f, rank))))
		}
		for f := 4; f >= 0; f-- {
			mirror = append(mirror, runeFor(b.At(chess.Sq(f, rank))))
		}
	}
	f, m := string(forward), string(mirror)
	if m < f {
		return m
	}
	return f
}

// --- the rules a puzzle must keep -------------------------------------------

// MateMoves lists colour c's moves that checkmate the opponent.
func MateMoves(b *chess.Board, c chess.Color) []chess.Move {
	var out []chess.Move
	for _, m := range b.LegalMovesFor(c) {
		if b.After(m).Checkmate(c.Opponent()) {
			out = append(out, m)
		}
	}
	return out
}

// CheckMoves lists colour c's moves that give check, mating or not.
func CheckMoves(b *chess.Board, c chess.Color) []chess.Move {
	var out []chess.Move
	for _, m := range b.LegalMovesFor(c) {
		if b.After(m).InCheck(c.Opponent()) {
			out = append(out, m)
		}
	}
	return out
}

// Valid checks that the position is legal to show white to move, and that it is
// the tier it claims to be. A puzzle that fails is never shipped: TestBankValid
// runs this over every line.
func (p Puzzle) Valid() error {
	b := p.Board
	wc, bc := chess.White, chess.Black
	if _, ok := b.KingSquare(wc); !ok {
		return fmt.Errorf("no white king")
	}
	if _, ok := b.KingSquare(bc); !ok {
		return fmt.Errorf("no black king")
	}
	if b.InCheck(bc) {
		return fmt.Errorf("black is already in check with white to move")
	}
	if b.InCheck(wc) {
		return fmt.Errorf("white is in check with white to move")
	}
	var legal bool
	for _, m := range b.LegalMovesFor(wc) {
		if m.From == p.Solution.From && m.To == p.Solution.To {
			legal = true
		}
	}
	if !legal {
		return fmt.Errorf("solution %s is not a legal move", Encode(p))
	}

	mates := MateMoves(b, wc)
	checks := CheckMoves(b, wc)

	switch p.Tier {
	case Check:
		if !b.After(p.Solution).InCheck(bc) {
			return fmt.Errorf("solution does not give check")
		}
	case Mate1:
		if len(mates) != 1 {
			return fmt.Errorf("want exactly 1 mate, found %d", len(mates))
		}
		if !sameMove(mates[0], p.Solution) {
			return fmt.Errorf("solution is not the mating move")
		}
		if len(checks) != 1 {
			return fmt.Errorf("want exactly 1 check, found %d", len(checks))
		}
	case Mate1Trap:
		if len(mates) != 1 || !sameMove(mates[0], p.Solution) {
			return fmt.Errorf("want exactly 1 mate, and the solution is it")
		}
		falseCheck := false
		for _, m := range checks {
			if !sameMove(m, p.Solution) {
				falseCheck = true
			}
		}
		if !falseCheck {
			return fmt.Errorf("want at least one check that is not mate")
		}
	case Mate2:
		if len(mates) != 0 {
			return fmt.Errorf("mate in two must not have a mate in one")
		}
		after := b.After(p.Solution)
		if !after.InCheck(bc) {
			return fmt.Errorf("solution does not give check")
		}
		if after.Checkmate(bc) {
			return fmt.Errorf("solution mates in one")
		}
		replies := after.LegalMovesFor(bc)
		if len(replies) == 0 {
			return fmt.Errorf("black has no reply")
		}
		for _, r := range replies {
			next := after.After(r)
			if len(MateMoves(next, wc)) == 0 {
				return fmt.Errorf("black's reply %s escapes the mate", squareName(r.From)+"-"+squareName(r.To))
			}
		}
	default:
		return fmt.Errorf("unknown tier %q", p.Tier)
	}
	return nil
}

func sameMove(a, b chess.Move) bool { return a.From == b.From && a.To == b.To }
