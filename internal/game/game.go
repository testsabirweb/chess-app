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
	// T is a free-running seconds counter for decorative animation.
	T    float64
	next Scene
}

func (c *Context) Switch(s Scene) { c.next = s }

type Game struct {
	scene   Scene
	ctx     Context
	pointer input.Tracker
	sfx     *sfx.Bank
	scale   float64
	insets  layout.Insets
	screenW float64
	screenH float64

	// stickers are every emoji reward collected this session; the home screen
	// and the play footer both read it.
	stickers []int

	// injected holds synthetic taps from the screenshot tool (see debug.go).
	injected []input.Event
	// scaleOverride lets the screenshot tool reproduce a phone's dp scale on a
	// desktop monitor. Zero means "use the real device scale factor".
	scaleOverride float64
}

func New() *Game {
	g := &Game{
		sfx:     sfx.NewBank(),
		ctx:     Context{Rand: rand.New(rand.NewPCG(uint64(seedA), uint64(seedB)))},
		scale:   1,
		screenW: 432,
		screenH: 960,
	}
	g.ctx.SFX = g.sfx
	g.ctx.Pointer = &g.pointer
	g.scene = NewHomeScene(g)
	return g
}

// Fixed seed: the sequence is varied enough for a toddler and a reproducible
// run is worth far more when debugging than a random one.
const (
	seedA = 0x9E3779B97F4A7C15
	seedB = 0xBF58476D1CE4E5B9
)

// AddSticker records a reward and reports the new total.
func (g *Game) AddSticker(emoji int) int {
	g.stickers = append(g.stickers, emoji)
	return len(g.stickers)
}

func (g *Game) Stickers() []int { return g.stickers }

func (g *Game) Update() error {
	dt := 1.0 / 60.0
	if tps := ebiten.TPS(); tps > 0 {
		dt = 1.0 / float64(tps)
	}
	g.ctx.DT = dt
	g.ctx.T += dt
	g.pointer.Update()
	if len(g.injected) > 0 {
		g.pointer.JustPressed = append(g.pointer.JustPressed, g.injected...)
		g.injected = g.injected[:0]
	}

	g.ctx.M = layout.Compute(g.screenW, g.screenH, g.scale, g.insets, 5, 5)

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
	if g.scaleOverride > 0 {
		s = g.scaleOverride
	}
	g.scale = s
	// The offscreen is in physical pixels, so the layout must be computed from
	// the same numbers Draw receives - not from WindowSize, which is logical.
	g.screenW = outsideWidth * s
	g.screenH = outsideHeight * s
	return g.screenW, g.screenH
}

// SetInsets allows Android to pass safe-area padding.
func (g *Game) SetInsets(top, bottom, left, right float64) {
	g.insets = layout.Insets{Top: top, Bottom: bottom, Left: left, Right: right}
}

func init() {
	_ = render.ColorBGTop
}
