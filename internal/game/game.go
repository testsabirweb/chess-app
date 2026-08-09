package game

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/input"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/sfx"
)

type Scene interface {
	Update(ctx *Context) error
	Draw(dst *ebiten.Image, ctx *Context)
}

type Context struct {
	M       layout.Metrics
	Pointer *input.Tracker
	SFX     *sfx.Bank
	Rand    *rand.Rand
	DT      float64
	next    Scene
}

func (c *Context) Switch(s Scene) { c.next = s }

type Game struct {
	scene   Scene
	ctx     Context
	pointer input.Tracker
	sfx     *sfx.Bank
	scale   float64
	insets  layout.Insets
}

func New() *Game {
	g := &Game{
		sfx: sfx.NewBank(),
		ctx: Context{Rand: rand.New(rand.NewPCG(1, 2))},
	}
	g.ctx.SFX = g.sfx
	g.ctx.Pointer = &g.pointer
	g.scene = NewHomeScene(g)
	return g
}

func (g *Game) Update() error {
	dt := 1.0 / 60.0
	if tps := ebiten.TPS(); tps > 0 {
		dt = 1.0 / float64(tps)
	}
	g.ctx.DT = dt
	g.pointer.Update()

	w, h := ebiten.WindowSize()
	if w == 0 || h == 0 {
		w, h = ebiten.ScreenSizeInFullscreen()
	}
	g.ctx.M = layout.Compute(float64(w), float64(h), g.scale, g.insets, 5, 5)

	if err := g.scene.Update(&g.ctx); err != nil {
		return err
	}
	if g.ctx.next != nil {
		g.scene = g.ctx.next
		g.ctx.next = nil
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.scene.Draw(screen, &g.ctx)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	if s <= 0 {
		s = 1
	}
	g.scale = s
	return outsideWidth * s, outsideHeight * s
}

// SetInsets allows Android to pass safe-area padding.
func (g *Game) SetInsets(top, bottom, left, right float64) {
	g.insets = layout.Insets{Top: top, Bottom: bottom, Left: left, Right: right}
}

func init() {
	_ = render.ColorBG
}
