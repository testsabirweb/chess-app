package challenge

import (
	"math"
	"math/rand/v2"

	"github.com/testsabirweb/chess-app/internal/chess"
)

const HistorySize = 8

type Challenge struct {
	Board     *chess.Board
	Piece     chess.Piece
	From      chess.Square
	Target    chess.Square
	Solutions []chess.Square
}

type Spec struct {
	BoardWidth, BoardHeight int
	Pieces                  []chess.PieceType
	Color                   chess.Color
	MinDistance, MaxDistance int // Chebyshev; 0 = unconstrained
	Decoys                  bool
}

type triple struct {
	piece chess.PieceType
	from  chess.Square
	to    chess.Square
	decoy bool
}

type Generator struct {
	spec    Spec
	rng     *rand.Rand
	history [HistorySize]triple
	head    int
	count   int
	last    triple
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
	return &Generator{spec: spec, rng: rng}
}

func (g *Generator) Next() Challenge {
	pieceType := g.pickPieceType()
	candidates := g.enumerate(pieceType)
	candidates = g.filterDistance(candidates)
	candidates = g.filterHistory(candidates)
	candidates = g.filterLastFromTarget(candidates)

	if len(candidates) == 0 {
		candidates = g.filterLastFromTarget(g.filterHistory(g.filterDistance(g.enumerate(pieceType))))
	}
	if len(candidates) == 0 {
		candidates = g.filterHistory(g.filterDistance(g.enumerate(pieceType)))
	}
	if len(candidates) == 0 {
		candidates = g.filterDistance(g.enumerate(pieceType))
	}
	if len(candidates) == 0 {
		candidates = g.enumerate(pieceType)
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
	}
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

func (g *Generator) enumerate(pieceType chess.PieceType) []triple {
	w, h := g.spec.BoardWidth, g.spec.BoardHeight
	out := make([]triple, 0, w*h*8)
	for r := 0; r < h; r++ {
		for f := 0; f < w; f++ {
			from := chess.Sq(f, r)
			withDecoy := g.spec.Decoys && pieceType == chess.Pawn && g.rng.IntN(3) == 0
			board := g.boardFor(from, pieceType, withDecoy)
			for _, to := range board.MoveTargets(from) {
				out = append(out, triple{piece: pieceType, from: from, to: to, decoy: withDecoy})
			}
		}
	}
	return out
}

func (g *Generator) boardFor(from chess.Square, pieceType chess.PieceType, withDecoy bool) *chess.Board {
	b := chess.NewBoard(g.spec.BoardWidth, g.spec.BoardHeight)
	b.Set(from, chess.Piece{Type: pieceType, Color: g.spec.Color})

	if withDecoy && pieceType == chess.Pawn {
		fwd := 1
		if g.spec.Color == chess.Black {
			fwd = -1
		}
		for _, df := range []int{1, -1} {
			cap := chess.Sq(int(from.File)+df, int(from.Rank)+fwd)
			if b.Contains(cap) && b.At(cap).IsEmpty() {
				b.Set(cap, chess.Piece{Type: chess.Pawn, Color: g.spec.Color.Opponent()})
				break
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
