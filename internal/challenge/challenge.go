package challenge

import (
	"math"
	"math/rand/v2"

	"github.com/testsabirweb/chess-app/internal/chess"
)

const HistorySize = 8

// maxDecoyVariants is how many distinct decoy layouts the generator can build
// for a given starting square. Variant 0 always means "no decoys".
const maxDecoyVariants = 4

type Challenge struct {
	Board     *chess.Board
	Piece     chess.Piece
	From      chess.Square
	Target    chess.Square
	Solutions []chess.Square
	// Moves is the shortest number of legal moves from From to Target.
	Moves int
}

type Spec struct {
	BoardWidth, BoardHeight int
	Pieces                  []chess.PieceType
	Color                   chess.Color
	// MinMoves/MaxMoves bound how many moves the star is away. Both default to
	// 1, which reproduces the original "star is one move away" behaviour.
	MinMoves, MaxMoves       int
	MinDistance, MaxDistance int // Chebyshev; 0 = unconstrained
	Decoys                   bool
}

type triple struct {
	piece chess.PieceType
	from  chess.Square
	to    chess.Square
	moves int
	decoy uint8
}

type Generator struct {
	spec      Spec
	rng       *rand.Rand
	history   [HistorySize]triple
	head      int
	count     int
	last      triple
	prevPiece chess.PieceType
}

func NewGenerator(spec Spec, rng *rand.Rand) *Generator {
	if rng == nil {
		rng = rand.New(rand.NewPCG(1, 2))
	}
	if len(spec.Pieces) == 0 {
		spec.Pieces = []chess.PieceType{chess.Rook}
	}
	if spec.BoardWidth <= 0 {
		spec.BoardWidth = 5
	}
	if spec.BoardHeight <= 0 {
		spec.BoardHeight = 5
	}
	if spec.MaxMoves <= 0 {
		spec.MaxMoves = 1
	}
	if spec.MinMoves <= 0 {
		spec.MinMoves = 1
	}
	if spec.MinMoves > spec.MaxMoves {
		spec.MinMoves = spec.MaxMoves
	}
	return &Generator{spec: spec, rng: rng}
}

func (g *Generator) Next() Challenge {
	pieceType := g.pickPieceType()
	all := g.enumerate(pieceType)

	candidates := g.filterFar(g.filterLastFromTarget(g.filterHistory(g.filterDistance(clone(all)))))
	if len(candidates) == 0 {
		candidates = g.filterLastFromTarget(g.filterHistory(g.filterDistance(clone(all))))
	}
	if len(candidates) == 0 {
		candidates = g.filterHistory(g.filterDistance(clone(all)))
	}
	if len(candidates) == 0 {
		candidates = g.filterDistance(clone(all))
	}
	if len(candidates) == 0 {
		candidates = all
	}

	pick := candidates[g.rng.IntN(len(candidates))]
	g.pushHistory(pick)
	g.prevPiece = pick.piece

	board := g.boardFor(pick.from, pick.piece, pick.decoy)
	piece := chess.Piece{Type: pick.piece, Color: g.spec.Color}
	solutions := board.MoveTargets(pick.from)
	return Challenge{
		Board:     board,
		Piece:     piece,
		From:      pick.from,
		Target:    pick.to,
		Solutions: solutions,
		Moves:     pick.moves,
	}
}

func clone(in []triple) []triple {
	out := make([]triple, len(in))
	copy(out, in)
	return out
}

func (g *Generator) pickPieceType() chess.PieceType {
	pool := g.spec.Pieces
	if len(pool) == 1 {
		return pool[0]
	}
	filtered := pool
	if g.prevPiece != chess.NoPiece && len(pool) > 1 {
		tmp := make([]chess.PieceType, 0, len(pool)-1)
		for _, p := range pool {
			if p != g.prevPiece {
				tmp = append(tmp, p)
			}
		}
		if len(tmp) > 0 {
			filtered = tmp
		}
	}
	return filtered[g.rng.IntN(len(filtered))]
}

// enumerate walks every start square, builds the board for it and breadth-first
// searches everywhere the piece can travel within MaxMoves.
func (g *Generator) enumerate(pieceType chess.PieceType) []triple {
	w, h := g.spec.BoardWidth, g.spec.BoardHeight
	out := make([]triple, 0, w*h*8)
	for r := 0; r < h; r++ {
		for f := 0; f < w; f++ {
			from := chess.Sq(f, r)
			if !g.startOK(pieceType, from) {
				continue
			}
			variant := uint8(0)
			if g.spec.Decoys && pieceType == chess.Pawn {
				variant = uint8(g.rng.IntN(maxDecoyVariants))
			}
			board := g.boardFor(from, pieceType, variant)
			for _, step := range Reach(board, from, g.spec.MaxMoves) {
				if step.Moves < g.spec.MinMoves {
					continue
				}
				// Never plant the star under another piece: the piece would
				// simply cover it up and the child would have nothing to aim at.
				if !board.At(step.Square).IsEmpty() {
					continue
				}
				out = append(out, triple{
					piece: pieceType,
					from:  from,
					to:    step.Square,
					moves: step.Moves,
					decoy: variant,
				})
			}
		}
	}
	return out
}

// startOK rejects starting squares a piece could never legally occupy.
//
// It matters for exactly one piece: the double push is allowed from the home
// rank, which in real chess is also where pawns begin, so "on the home rank"
// and "has not moved yet" mean the same thing. Start a pawn *behind* its home
// rank, though, and it can step forward onto the home rank and then double
// push - a second first move. Keeping pawns on or ahead of their home rank
// makes the positional rule exact again.
func (g *Generator) startOK(pieceType chess.PieceType, from chess.Square) bool {
	if pieceType != chess.Pawn {
		return true
	}
	if g.spec.Color == chess.Black {
		return int(from.Rank) <= g.spec.BoardHeight-2
	}
	return int(from.Rank) >= 1
}

// decoySquares lists, deterministically, where the enemy pieces go for a pawn
// starting on `from` under the given variant. Decoys always sit on a
// neighbouring file so they can never block the pawn's own file.
func (g *Generator) decoySquares(from chess.Square, variant uint8) []chess.Square {
	if variant == 0 {
		return nil
	}
	fwd := 1
	if g.spec.Color == chess.Black {
		fwd = -1
	}
	var cands []chess.Square
	for k := 1; k < g.spec.BoardHeight; k++ {
		r := int(from.Rank) + k*fwd
		if r < 0 || r >= g.spec.BoardHeight {
			break
		}
		for _, df := range []int{-1, 1} {
			f := int(from.File) + df
			if f < 0 || f >= g.spec.BoardWidth {
				continue
			}
			cands = append(cands, chess.Sq(f, r))
		}
	}
	if len(cands) == 0 {
		return nil
	}
	n := int(variant)
	if n > len(cands) {
		n = len(cands)
	}
	out := make([]chess.Square, 0, n)
	seen := map[chess.Square]bool{}
	for i := 0; i < n; i++ {
		sq := cands[(i*3+int(variant)*5)%len(cands)]
		if seen[sq] {
			continue
		}
		seen[sq] = true
		out = append(out, sq)
	}
	return out
}

func (g *Generator) boardFor(from chess.Square, pieceType chess.PieceType, variant uint8) *chess.Board {
	b := chess.NewBoard(g.spec.BoardWidth, g.spec.BoardHeight)
	b.Set(from, chess.Piece{Type: pieceType, Color: g.spec.Color})

	if variant > 0 && g.spec.Decoys && pieceType == chess.Pawn {
		for _, sq := range g.decoySquares(from, variant) {
			if b.Contains(sq) && b.At(sq).IsEmpty() {
				b.Set(sq, chess.Piece{Type: chess.Pawn, Color: g.spec.Color.Opponent()})
			}
		}
	}
	return b
}

func chebyshev(a, b chess.Square) int {
	df := int(a.File) - int(b.File)
	if df < 0 {
		df = -df
	}
	dr := int(a.Rank) - int(b.Rank)
	if dr < 0 {
		dr = -dr
	}
	if df > dr {
		return df
	}
	return dr
}

func (g *Generator) filterDistance(c []triple) []triple {
	if g.spec.MinDistance <= 0 && g.spec.MaxDistance <= 0 {
		return c
	}
	out := c[:0]
	for _, t := range c {
		d := chebyshev(t.from, t.to)
		if g.spec.MinDistance > 0 && d < g.spec.MinDistance {
			continue
		}
		if g.spec.MaxDistance > 0 && d > g.spec.MaxDistance {
			continue
		}
		out = append(out, t)
	}
	return out
}

// filterFar biases most challenges towards journeys of two or more moves, so
// the star is usually somewhere the child has to plan a little route to reach.
// It is a preference, not a rule: it is the first rung dropped when empty.
func (g *Generator) filterFar(c []triple) []triple {
	if g.spec.MaxMoves < 2 || g.rng.IntN(10) >= 7 {
		return c
	}
	out := c[:0]
	for _, t := range c {
		if t.moves >= 2 {
			out = append(out, t)
		}
	}
	return out
}

func (g *Generator) filterHistory(c []triple) []triple {
	if g.count == 0 {
		return c
	}
	out := c[:0]
	for _, t := range c {
		if g.inHistory(t) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (g *Generator) filterLastFromTarget(c []triple) []triple {
	if g.count == 0 {
		return c
	}
	out := c[:0]
	for _, t := range c {
		if t.from == g.last.from || t.to == g.last.to {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (g *Generator) inHistory(t triple) bool {
	for i := 0; i < g.count && i < HistorySize; i++ {
		idx := (g.head - 1 - i + HistorySize) % HistorySize
		h := g.history[idx]
		if h.piece == t.piece && h.from == t.from && h.to == t.to {
			return true
		}
	}
	return false
}

func (g *Generator) pushHistory(t triple) {
	g.history[g.head] = t
	g.head = (g.head + 1) % HistorySize
	if g.count < HistorySize {
		g.count++
	}
	g.last = t
}

func Distance(a, b chess.Square) int {
	return chebyshev(a, b)
}

func MaxChebyshev(w, h int) int {
	return int(math.Max(float64(w-1), float64(h-1)))
}
