package game

import (
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/input"
)

// This file exists for the desktop screenshot tool in cmd/shot. It is a few
// accessors and a synthetic tap; nothing here changes how the game plays.

// NewInPlay builds a game that starts straight in the play scene.
func NewInPlay(pt chess.PieceType) *Game {
	g := New()
	g.scene = NewPlayScene(g, pt)
	return g
}

// PlayInfo is a snapshot of the play scene for the screenshot tool.
type PlayInfo struct {
	Piece  chess.Square
	Target chess.Square
	Hints  []chess.Square
	Board  *chess.Board
}

// PlayInfo reports the current play state, or ok=false on other scenes.
func (g *Game) PlayInfo() (PlayInfo, bool) {
	ps, ok := g.scene.(*PlayScene)
	if !ok {
		return PlayInfo{}, false
	}
	return PlayInfo{Piece: ps.at, Target: ps.target, Hints: ps.solutions, Board: ps.board}, true
}

// TapSquare queues a synthetic press at the centre of a board cell.
func (g *Game) TapSquare(f, r int) {
	cr := g.ctx.M.CellRect(f, r)
	x, y := cr.Center()
	g.TapPoint(x, y)
}

// TapPoint queues a synthetic press at a screen position.
func (g *Game) TapPoint(x, y float64) {
	g.injected = append(g.injected, input.Event{X: x, Y: y, Pressed: true})
}
