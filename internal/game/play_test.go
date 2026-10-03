package game

import (
	"math/rand/v2"
	"testing"

	"github.com/testsabirweb/chess-app/internal/challenge"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/reward"
)

func testGame() *Game {
	g := &Game{
		reward: reward.NewPicker(render.RewardEmojiIndices(), rand.New(rand.NewPCG(1, 2))),
		ctx:    Context{Rand: rand.New(rand.NewPCG(3, 4))},
	}
	g.ctx.Pointer = &g.pointer
	return g
}

// Landing on the star must collect it exactly once. A regression left the scene
// in stateMoving after the win, so every following frame landed again and the
// reward popped over and over.
func TestStarIsCollectedOnce(t *testing.T) {
	g := testGame()
	p := NewPlayScene(g, chess.Rook)
	m := layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	ctx := &g.ctx
	ctx.M = m
	ctx.DT = 1.0 / 60

	// Put the rook one step from the star and hop it there.
	p.board.Set(p.at, chess.Piece{})
	p.at = chess.Sq(0, 0)
	p.targets = []chess.Square{chess.Sq(0, 3)}
	p.board.Set(p.at, p.cur.Piece)
	p.solutions = p.board.MoveTargets(p.at)
	p.steps = 1
	p.optimal = 1
	p.laidOut = false
	p.pieceSelected = true
	p.syncPiecePos(m)
	p.startMove(chess.Sq(0, 3), m)

	for i := 0; i < 600; i++ {
		if err := p.Update(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("collected %d stickers from one star, want 1", got)
	}
}

// fixedPuzzle is a rook with two stars, each one move apart in turn:
// (0,0) -> (0,3) -> (4,3).
func fixedPuzzle() challenge.Puzzle {
	b := chess.NewBoard(5, 5)
	piece := chess.Piece{Type: chess.Rook, Color: chess.White}
	b.Set(chess.Sq(0, 0), piece)
	return challenge.Puzzle{
		Board: b, From: chess.Sq(0, 0), Piece: piece,
		Targets: []chess.Square{chess.Sq(0, 3), chess.Sq(4, 3)}, Optimal: 2,
	}
}

func hop(t *testing.T, g *Game, p *PlayScene, to chess.Square, frames int) {
	t.Helper()
	p.pieceSelected = true
	p.startMove(to, g.ctx.M)
	for i := 0; i < frames; i++ {
		if err := p.Update(&g.ctx); err != nil {
			t.Fatal(err)
		}
	}
}

// Several stars: each one is collected as the piece lands on it, and only the
// last one wins the round.
func TestTreasureNeedsEveryStar(t *testing.T) {
	g := testGame()
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	served := false
	p := newPlayScene(g, chess.Rook, func() challenge.Puzzle {
		if served {
			return challenge.NewTreasure(g.ctx.Rand, chess.Rook, chess.White, 2, chess.Sq(9, 9))
		}
		served = true
		return fixedPuzzle()
	})

	hop(t, g, p, chess.Sq(0, 3), 60)
	if len(p.targets) != 1 || len(g.Stickers()) != 0 || p.kit.busy() {
		t.Fatalf("after one star: %d left, %d stickers, busy=%v", len(p.targets), len(g.Stickers()), p.kit.busy())
	}

	hop(t, g, p, chess.Sq(4, 3), 600)
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("got %d stickers after the last star, want exactly 1", got)
	}
}

func newFixedTreasure(g *Game) *PlayScene {
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	return newPlayScene(g, chess.Rook, func() challenge.Puzzle { return fixedPuzzle() })
}

// Clearing every star in the fewest moves earns the bigger celebration.
func TestTreasureShortestRouteIsPerfect(t *testing.T) {
	g := testGame()
	p := newFixedTreasure(g)
	hop(t, g, p, chess.Sq(0, 3), 40)
	hop(t, g, p, chess.Sq(4, 3), 30) // lands at ~25 frames, before the next puzzle is dealt
	if !p.kit.celebrating() || !p.kit.perfect {
		t.Fatalf("celebrating=%v perfect=%v, want a perfect win in 2 moves", p.kit.celebrating(), p.kit.perfect)
	}
}

// Wandering still wins, it just is not "perfect".
func TestTreasureLongRouteIsNotPerfect(t *testing.T) {
	g := testGame()
	p := newFixedTreasure(g)
	hop(t, g, p, chess.Sq(4, 0), 40)
	hop(t, g, p, chess.Sq(4, 3), 40)
	if len(p.targets) != 0 && len(p.targets) != 1 {
		t.Fatalf("%d stars left", len(p.targets))
	}
	hop(t, g, p, chess.Sq(0, 3), 30)
	if !p.kit.celebrating() || p.kit.perfect {
		t.Fatalf("celebrating=%v perfect=%v, want a plain win after 3 moves", p.kit.celebrating(), p.kit.perfect)
	}
}

// A star the piece can no longer reach hops somewhere it can, and the round
// stays winnable.
func TestStarsStayReachable(t *testing.T) {
	g := testGame()
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	b := chess.NewBoard(5, 5)
	piece := chess.Piece{Type: chess.Pawn, Color: chess.White}
	b.Set(chess.Sq(2, 1), piece)
	p := newPlayScene(g, chess.Pawn, func() challenge.Puzzle {
		return challenge.Puzzle{
			Board: b.Clone(), From: chess.Sq(2, 1), Piece: piece,
			Targets: []chess.Square{chess.Sq(2, 3), chess.Sq(2, 4)}, Optimal: 3,
		}
	})
	// The pawn steps up and past nothing, but a star behind it would be lost; put
	// one there directly and let the safety net notice.
	p.targets = []chess.Square{chess.Sq(2, 0), chess.Sq(2, 4)}
	p.at = chess.Sq(2, 2)
	p.board.Set(chess.Sq(2, 1), chess.Piece{})
	p.board.Set(p.at, piece)
	p.keepStarsReachable(&g.ctx)
	for i, tgt := range p.targets {
		if !challenge.CanReach(p.board, p.at, tgt, maxJourney+2) {
			t.Fatalf("star %d at %v is still out of reach", i, tgt)
		}
	}
}
