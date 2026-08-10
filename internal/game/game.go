package game

import (
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/input"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/reward"
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

	// reward picks stickers and celebration-message indices from its own RNG
	// stream, seeded separately from ctx.Rand below. They used to share one
	// stream, so the emoji you got was a side effect of how many random draws
	// that piece's move happened to cost, not a fresh coin flip - same piece,
	// same move, same-looking pattern every time. See internal/reward.
	reward *reward.Picker

	// injected holds synthetic taps from the screenshot tool (see debug.go).
	injected []input.Event
	// scaleOverride lets the screenshot tool reproduce a phone's dp scale on a
	// desktop monitor. Zero means "use the real device scale factor".
	scaleOverride float64
}

func New() *Game {
	g := &Game{
		sfx:     sfx.NewBank(),
		ctx:     Context{Rand: rand.New(rand.NewPCG(entropySeed(puzzleSalt)))},
		reward:  reward.NewPicker(render.RewardEmojiIndices(), rand.New(rand.NewPCG(entropySeed(rewardSalt)))),
		scale:   1,
		screenW: 432,
		screenH: 960,
	}
	g.ctx.SFX = g.sfx
	g.ctx.Pointer = &g.pointer
	g.scene = NewHomeScene(g)
	return g
}

// puzzleSalt and rewardSalt keep the puzzle generator's stream and the reward
// stream from correlating even if entropySeed is called for both within the
// same clock tick (see entropySeed).
const (
	puzzleSalt = 0xA24BAED4963EE407
	rewardSalt = 0x9E3779B97F4A7C15
)

// entropySeed mixes the wall clock and a caller-specific salt into two PCG
// seed halves via a splitmix64 step, so two calls in the same nanosecond still
// produce different, well-mixed seeds. It only needs to vary from one app
// launch to the next and between the puzzle/reward streams, not withstand an
// adversary - this drives which square a piece starts on and which sticker it
// wins, nothing security-sensitive.
func entropySeed(salt uint64) (uint64, uint64) {
	z := uint64(time.Now().UnixNano()) + salt + 0x9E3779B97F4A7C15
	a := z
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	z = z ^ (z >> 31)
	return a, z
}

// AddSticker records a reward and reports the new total.
func (g *Game) AddSticker(emoji int) int {
	g.stickers = append(g.stickers, emoji)
	return len(g.stickers)
}

func (g *Game) Stickers() []int { return g.stickers }

// NextRewardEmoji deals the next sticker: every reward emoji is handed out
// exactly once before any of them repeat.
func (g *Game) NextRewardEmoji() int { return g.reward.Next() }

// RewardIntN exposes the reward RNG for other cosmetic reward-flavour picks
// (e.g. which celebration message to show) that should share its real
// randomness rather than the deterministic puzzle stream.
func (g *Game) RewardIntN(n int) int { return g.reward.IntN(n) }

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
