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

// NewInMode builds a game that starts straight in the given mode. pt is the
// piece for the modes that need one.
func NewInMode(mode Mode, pt chess.PieceType) *Game {
	g := New()
	g.mode = mode
	g.scene = newModeScene(g, mode, pt)
	return g
}

// ModeByName finds a mode from the name the screenshot tool is given.
func ModeByName(name string) (Mode, bool) {
	switch name {
	case "star":
		return ModeStar, true
	case "which":
		return ModeWhich, true
	case "treasure":
		return ModeTreasure, true
	case "safe":
		return ModeSafe, true
	case "pawnwars":
		return ModePawnWars, true
	}
	return ModeStar, false
}

// WhichInfo is a snapshot of a "Which piece?" round for the screenshot tool.
type WhichInfo struct {
	Pieces []chess.Square
	Answer chess.Square
	Target chess.Square
}

// WhichInfo reports the current round, or ok=false on other scenes.
func (g *Game) WhichInfo() (WhichInfo, bool) {
	ws, ok := g.scene.(*WhichScene)
	if !ok {
		return WhichInfo{}, false
	}
	return WhichInfo{Pieces: ws.cur.Pieces, Answer: ws.cur.Answer, Target: ws.cur.Target}, true
}

// SeedStickers pre-fills the reward tray so the screenshot tool can reach the
// milestone celebration without playing five rounds.
func (g *Game) SeedStickers(n int) {
	for i := 0; i < n; i++ {
		g.stickers = append(g.stickers, i*7)
	}
}

// SetScaleOverride forces a dp scale, so the screenshot tool can render at a
// specific phone's metrics rather than the developer's monitor.
func (g *Game) SetScaleOverride(s float64) { g.scaleOverride = s }

// PlayInfo is a snapshot of the play scene for the screenshot tool.
type PlayInfo struct {
	Piece    chess.Square
	Target   chess.Square
	Hints    []chess.Square
	Board    *chess.Board
	Selected bool
	// Hot is the squares a guard attacks in the Stay-safe game, else nil.
	Hot []chess.Square
}

// PlayInfo reports the current play state, or ok=false on other scenes.
func (g *Game) PlayInfo() (PlayInfo, bool) {
	ps, ok := g.scene.(*PlayScene)
	if !ok {
		return PlayInfo{}, false
	}
	// Target is the first star still to collect; once none are left, the square
	// the piece is standing on.
	target := ps.at
	if len(ps.targets) > 0 {
		target = ps.targets[0]
	}
	return PlayInfo{
		Piece: ps.at, Target: target, Hints: ps.solutions, Board: ps.board,
		Selected: ps.pieceSelected, Hot: ps.hot,
	}, true
}

// SetHintDelay overrides the pause before a picked-up piece shows its moves.
// The screenshot tool sets it to zero: the shots exist to show what the hinted
// board looks like, not to sit through the wait first.
func (g *Game) SetHintDelay(d float64) { g.hintDelay = d }

// NextChallenge skips to the next generated puzzle, so the screenshot tool can
// walk through several without playing them.
func (g *Game) NextChallenge() {
	if ps, ok := g.scene.(*PlayScene); ok {
		ps.newChallenge()
	}
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
