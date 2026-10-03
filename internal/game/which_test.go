package game

import (
	"testing"

	"github.com/testsabirweb/chess-app/internal/input"
	"github.com/testsabirweb/chess-app/internal/layout"
)

func tapCell(g *Game, m layout.Metrics, f, r int) {
	x, y := m.CellRect(f, r).Center()
	g.pointer.JustPressed = []input.Event{{X: x, Y: y, Pressed: true}}
}

// Picking the right piece lifts it and waits. It must not move until the
// child taps the star.
func TestWhichPieceWaitsForTheStar(t *testing.T) {
	g := testGame()
	s := NewWhichScene(g)
	m := layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	ctx := &g.ctx
	ctx.M = m
	ctx.DT = 1.0 / 60

	step := func(n int) {
		for i := 0; i < n; i++ {
			if err := s.Update(ctx); err != nil {
				t.Fatal(err)
			}
			g.pointer.JustPressed = nil
		}
	}

	ans := s.cur.Answer
	tapCell(g, m, int(ans.File), int(ans.Rank))
	step(1)
	if !s.holding {
		t.Fatal("tapping the right piece did not pick it up")
	}
	step(300) // plenty of time to move by itself, if it were going to
	if s.state != whichIdle || !s.holding || len(g.Stickers()) != 0 {
		t.Fatalf("piece moved on its own: state=%v holding=%v stickers=%d", s.state, s.holding, len(g.Stickers()))
	}

	tapCell(g, m, int(s.cur.Target.File), int(s.cur.Target.Rank))
	step(600)
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("got %d stickers after tapping the star, want 1", got)
	}
}

// Tapping the held piece again puts it back down.
func TestWhichPutDown(t *testing.T) {
	g := testGame()
	s := NewWhichScene(g)
	m := layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	ctx := &g.ctx
	ctx.M = m
	ctx.DT = 1.0 / 60
	ans := s.cur.Answer
	for i := 0; i < 2; i++ {
		tapCell(g, m, int(ans.File), int(ans.Rank))
		if err := s.Update(ctx); err != nil {
			t.Fatal(err)
		}
		g.pointer.JustPressed = nil
	}
	if s.holding {
		t.Fatal("second tap on the piece should put it down")
	}
}
