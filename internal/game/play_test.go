package game

import (
	"math/rand/v2"
	"testing"

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
	p.target = chess.Sq(0, 3)
	p.board.Set(p.at, p.cur.Piece)
	p.solutions = p.board.MoveTargets(p.at)
	p.steps = 1
	p.optimal = 1
	p.laidOut = false
	p.pieceSelected = true
	p.syncPiecePos(m)
	p.startMove(p.target, m)

	for i := 0; i < 600; i++ {
		if err := p.Update(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("collected %d stickers from one star, want 1", got)
	}
}
