//go:build android || ios

package mobile

import (
	"github.com/hajimehoshi/ebiten/v2/mobile"
	"github.com/testsabirweb/chess-app/internal/game"
)

func init() {
	mobile.SetGame(game.New())
}
