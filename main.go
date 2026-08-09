package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/game"
)

func main() {
	ebiten.SetWindowTitle("Toddler Chess")
	ebiten.SetWindowSize(432, 960)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(game.New()); err != nil {
		panic(err)
	}
}
