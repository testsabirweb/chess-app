//go:build android

package mobile

import "github.com/testsabirweb/chess-app/internal/game"

func SetUpdateReady(ready bool) { game.SetUpdateReady(ready) }

func UpdateReady() bool { return game.UpdateReady() }

func RequestInstallUpdate() { game.RequestInstallUpdate() }

func ConsumeInstallRequest() bool { return game.ConsumeInstallRequest() }
