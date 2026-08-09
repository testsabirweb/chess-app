package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/sfx"
)

type HomeScene struct {
	game *Game
}

func NewHomeScene(g *Game) *HomeScene { return &HomeScene{game: g} }

func (h *HomeScene) Update(ctx *Context) error {
	m := ctx.M
	playRect := layoutButton(m.Safe, 0.15, 0.12)
	pieceRects := piecePickerRects(m)

	for _, ev := range ctx.Pointer.Pressed() {
		if playRect.Contains(ev.X, ev.Y) {
			ctx.SFX.Play(sfx.SndButton)
			ctx.Switch(NewPlayScene(h.game, chess.Rook))
			return nil
		}
		for i, pr := range pieceRects {
			if pr.Contains(ev.X, ev.Y) {
				ctx.SFX.Play(sfx.SndButton)
				ctx.Switch(NewPlayScene(h.game, allPieces[i]))
				return nil
			}
		}
	}
	return nil
}

func (h *HomeScene) Draw(dst *ebiten.Image, ctx *Context) {
	m := ctx.M
	render.DrawFilledRect(dst, 0, 0, m.W, m.H, render.ColorBG)
	render.DrawTextCentered(dst, "Find the Star!", m.Safe.X+m.Safe.W/2, m.Header.Y+m.Header.H*0.35, m.TitleSize, render.ColorText)

	playRect := layoutButton(m.Safe, 0.15, 0.12)
	render.DrawRoundedButton(dst, playRect.X, playRect.Y, playRect.W, playRect.H, render.ColorButton)
	render.DrawTextCentered(dst, "Play!", playRect.X+playRect.W/2, playRect.Y+playRect.H/2, m.TitleSize*0.9, render.ColorText)

	pieceRects := piecePickerRects(m)
	for i, pr := range pieceRects {
		render.DrawRoundedButton(dst, pr.X, pr.Y, pr.W, pr.H, render.ColorButtonHi)
		pad := pr.W * 0.15
		render.DrawPiece(dst, allPieces[i], layout.Rect{X: pr.X + pad, Y: pr.Y + pad, W: pr.W - 2*pad, H: pr.H - 2*pad}, render.PieceFill(chess.White))
		render.DrawTextCentered(dst, render.PieceName(allPieces[i]), pr.X+pr.W/2, pr.Y+pr.H*0.88, m.BodySize*0.7, render.ColorText)
	}
}

var allPieces = []chess.PieceType{chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King}

func layoutButton(safe layout.Rect, topFrac, hFrac float64) layout.Rect {
	h := safe.H * hFrac
	w := safe.W * 0.85
	return layout.Rect{X: safe.X + (safe.W-w)/2, Y: safe.Y + safe.H*topFrac, W: w, H: h}
}

func piecePickerRects(m layout.Metrics) []layout.Rect {
	gap := m.Cell * 0.08
	btn := (m.Safe.W - 2*gap) / 3
	startY := m.Safe.Y + m.Safe.H*0.32
	out := make([]layout.Rect, 6)
	for i := 0; i < 6; i++ {
		col := i % 3
		row := i / 3
		out[i] = layout.Rect{
			X: m.Safe.X + float64(col)*(btn+gap),
			Y: startY + float64(row)*(btn*1.1+gap),
			W: btn, H: btn * 1.1,
		}
	}
	return out
}
