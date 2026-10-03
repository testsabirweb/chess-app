package game

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/anim"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/sfx"
)

// roundKit is everything a board mode shares and none of them should write
// twice: the back button, the sticker tray, the sticker that pops out of a win
// and flies into it, the party every fifth sticker, and the fade between
// puzzles. A mode owns its board and its rules; it hands the kit a win and
// the kit takes it from there.
type roundKit struct {
	game *Game

	phase kitPhase

	confetti anim.Confetti
	sprites  *render.Sprites

	reward       rewardFly
	rewardActive bool
	// perfect is set by win and cleared when the next puzzle is dealt; it adds
	// the gold glow and the "Perfect!" header while the party is on.
	perfect bool

	advanceT   float64
	advanceSwp bool

	milestoneT      float64
	milestoneCount  int
	milestoneEmojis []int
	milestoneMsg    string

	backHoldT    float64
	backInstallF bool
}

// kitPhase says where a round is: being played, or somewhere in the wind-down
// after a win. The mode ignores board taps unless the phase is kitPlay.
type kitPhase int

const (
	kitPlay kitPhase = iota
	kitCelebrating
	kitMilestone
	kitAdvancing
)

// milestoneEvery is how many stickers earn the big celebration.
const milestoneEvery = 5

// advanceHalf is how long the screen takes to fade to cream, and again to fade
// back, while the next puzzle is dealt behind it.
const advanceHalf = 0.22

// milestoneMessages are shown at random alongside the sticker count. Picked
// from Game.RewardIntN, the same real-randomness source as the stickers
// themselves, not the deterministic puzzle stream.
var milestoneMessages = []string{
	"Great job!",
	"Keep it up!",
	"You're a star!",
	"Amazing work!",
	"Wow, look at you go!",
	"Fantastic!",
	"You did it!",
	"Super job!",
	"Way to go!",
}

// rewardFly is the sticker popping out of the star and flying into the tray.
type rewardFly struct {
	emoji              int
	fromX, fromY       float64
	toX, toY           float64
	x, y, size         float64
	pop, fly           anim.Tween
	phase              int // 0 = popping, 1 = flying, 2 = landed
	popSize, traySize  float64
	popDoneX, popDoneY float64
}

func newRoundKit(g *Game) roundKit {
	return roundKit{game: g, sprites: render.NewSprites()}
}

// busy reports whether a win is still being celebrated, so the mode should
// ignore the board.
func (k *roundKit) busy() bool { return k.phase != kitPlay }

func (k *roundKit) celebrating() bool { return k.phase == kitCelebrating }

// restart fades to the next puzzle without a reward: the way out when a piece
// has been left with nowhere to go.
func (k *roundKit) restart() {
	k.phase = kitAdvancing
	k.advanceT = 0
	k.advanceSwp = false
}

// update runs the kit's animations and the back button.
//
// leave means the back button was tapped and the caller should switch to the
// home scene. deal means the screen is fully covered by the fade, so the
// caller should deal the next puzzle now.
func (k *roundKit) update(ctx *Context, m layout.Metrics) (leave, deal bool) {
	k.confetti.Update(ctx.DT, m.Cell*8)
	k.updateReward(ctx, m)

	back := backButtonRect(m)
	for _, ev := range ctx.Pointer.JustReleased {
		if back.Contains(ev.X, ev.Y) && k.backHoldT > 0 && k.backHoldT < 2.0 {
			ctx.SFX.Play(sfx.SndButton)
			return true, false
		}
	}
	if held, ok := ctx.Pointer.Held(); ok && back.Contains(held.X, held.Y) {
		k.backHoldT += ctx.DT
		if k.backHoldT >= 2.0 && UpdateReady() && !k.backInstallF {
			RequestInstallUpdate()
			k.backInstallF = true
		}
	} else {
		k.backHoldT = 0
		k.backInstallF = false
	}

	if k.phase == kitMilestone {
		// A tap anywhere skips the rest of the party.
		for range ctx.Pointer.Pressed() {
			k.milestoneT = 0
		}
	}

	switch k.phase {
	case kitCelebrating:
		if k.reward.phase == 2 {
			k.finishReward(ctx)
		}
	case kitMilestone:
		k.milestoneT -= ctx.DT
		if k.milestoneT <= 0 {
			k.restart()
		}
	case kitAdvancing:
		k.advanceT += ctx.DT
		if !k.advanceSwp && k.advanceT >= advanceHalf {
			k.advanceSwp = true
			k.perfect = false
			deal = true
		}
		if k.advanceT >= 2*advanceHalf {
			k.phase = kitPlay
		}
	}
	return false, deal
}

// win starts the celebration for a solved puzzle: confetti and a sticker
// popping out at (cx, cy). Perfect solutions get twice the confetti and a gold
// glow. The caller plays the sounds, because each mode chooses its own.
func (k *roundKit) win(ctx *Context, m layout.Metrics, cx, cy float64, perfect bool) {
	k.perfect = perfect
	burst := 30
	if perfect {
		burst = 60
	}
	k.confetti.Burst(ctx.Rand, cx, cy, burst, m.Cell)

	slot := trayEmojiPos(m, len(k.game.Stickers()))
	k.reward = rewardFly{
		emoji:    k.game.NextRewardEmoji(),
		fromX:    cx,
		fromY:    cy,
		toX:      slot.X,
		toY:      slot.Y,
		popSize:  m.Cell * 0.78,
		traySize: slot.W,
		pop:      anim.Tween{Duration: 0.42, Ease: anim.EaseOutBack},
		fly:      anim.Tween{Duration: 0.55, Ease: anim.EaseInOutCubic},
	}
	k.reward.pop.Start()
	k.reward.x, k.reward.y = cx, cy
	k.reward.size = 0
	k.rewardActive = true
	k.phase = kitCelebrating
}

func (k *roundKit) updateReward(ctx *Context, m layout.Metrics) {
	if !k.rewardActive {
		return
	}
	switch k.reward.phase {
	case 0:
		t := k.reward.pop.Update(ctx.DT)
		k.reward.size = k.reward.popSize * t
		k.reward.y = k.reward.fromY - m.Cell*0.25*t
		if k.reward.pop.Done() {
			k.reward.phase = 1
			k.reward.popDoneX, k.reward.popDoneY = k.reward.x, k.reward.y
			k.reward.fly.Start()
		}
	case 1:
		t := k.reward.fly.Update(ctx.DT)
		k.reward.x = lerp(k.reward.popDoneX, k.reward.toX, t)
		// A shallow arc so it looks tossed into the tray, not dragged.
		k.reward.y = lerp(k.reward.popDoneY, k.reward.toY, t) - math.Sin(t*math.Pi)*m.Cell*0.55
		k.reward.size = lerp(k.reward.popSize, k.reward.traySize, t)
		if k.reward.fly.Done() {
			k.reward.phase = 2
		}
	}
}

func (k *roundKit) finishReward(ctx *Context) {
	k.rewardActive = false
	total := k.game.AddSticker(k.reward.emoji)
	if total%milestoneEvery == 0 {
		ctx.SFX.Play(sfx.SndMilestone)
		k.phase = kitMilestone
		k.milestoneT = 3.4
		k.milestoneCount = total
		k.milestoneMsg = milestoneMessages[k.game.RewardIntN(len(milestoneMessages))]
		all := k.game.Stickers()
		k.milestoneEmojis = all[len(all)-milestoneEvery:]
		return
	}
	k.restart()
}

// --- drawing -----------------------------------------------------------------

// drawHeader draws the back button and the mode's title. The one exception to
// the title is the shortest-route cheer, which borrows the same slot while the
// confetti is falling so nothing moves.
func (k *roundKit) drawHeader(dst *ebiten.Image, m layout.Metrics, name string) {
	b := backButtonRect(m)
	render.DrawChunkyButton(dst, b.X, b.Y, b.W, b.H, render.ColorBack, render.ColorBackEdge, false)
	render.DrawChevronLeft(dst, b.X, b.Y, b.W, b.H, b.H*0.12, render.ColorText)
	if UpdateReady() {
		dotR := b.H * 0.06
		render.FillCircleSoft(dst, b.X+b.W-dotR*2.5, b.Y+dotR*2.5, dotR, render.ColorTextDim)
	}

	h := m.Header
	clr := render.ColorTextDim
	if k.perfect && (k.phase == kitCelebrating || k.phase == kitMilestone) {
		name, clr = "Perfect!", render.ColorStarGlow
	}
	size := render.FitTextSize(name, m.BodySize*1.25, h.W*0.6)
	render.DrawTextShadowed(dst, name, h.X+h.W/2, h.Y+h.H*0.72, size, clr)
}

// drawOverlays draws everything that sits above the board: confetti, the
// flying sticker, the tray, the milestone party and the fade between puzzles.
func (k *roundKit) drawOverlays(dst *ebiten.Image, ctx *Context, m layout.Metrics) {
	render.DrawConfetti(dst, &k.confetti, k.sprites, m.Cell)

	if k.rewardActive && k.reward.size > 0 {
		if k.perfect {
			render.DrawGlow(dst, k.reward.x, k.reward.y, k.reward.size, render.Alpha(render.ColorStarGlow, 0.45))
		}
		render.DrawEmoji(dst, render.EmojiName(k.reward.emoji), k.reward.x, k.reward.y, k.reward.size, 0, 1)
	}

	k.drawFooter(dst, m)

	if k.phase == kitMilestone {
		k.drawMilestone(dst, ctx, m)
	}
	if k.phase == kitAdvancing {
		a := 1 - math.Abs(k.advanceT/advanceHalf-1)
		render.DrawFilledRect(dst, 0, 0, m.W, m.H, render.Alpha(render.ColorBGTop, clamp01(a)*0.85))
	}
}

func (k *roundKit) drawFooter(dst *ebiten.Image, m layout.Metrics) {
	tr := trayRect(m)
	render.FillRoundRect(dst, tr.X, tr.Y, tr.W, tr.H, tr.H*0.35, render.ColorTray)

	// Empty slots are drawn too, so the tray reads as a row waiting to be
	// filled rather than one lonely sticker in a wide bar.
	slots := traySlots(m)
	for i := 0; i < slots; i++ {
		slot := trayEmojiPos(m, i)
		render.FillCircleSoft(dst, slot.X, slot.Y, slot.W*0.30, render.ColorTraySlot)
	}

	stickers := k.game.Stickers()
	show := stickers
	if len(show) > slots {
		show = show[len(show)-slots:]
	}
	base := len(stickers) - len(show)
	for i, e := range show {
		slot := trayEmojiPos(m, base+i)
		render.DrawEmoji(dst, render.EmojiName(e), slot.X, slot.Y, slot.W, 0, 1)
	}
}

func (k *roundKit) drawMilestone(dst *ebiten.Image, ctx *Context, m layout.Metrics) {
	render.DrawFilledRect(dst, 0, 0, m.W, m.H, render.Alpha(render.ColorBGTop, 0.88))
	cx := m.W / 2
	cy := m.H * 0.44

	pw := m.Safe.W
	ph := m.Cell * 4.1
	render.FillRoundRect(dst, cx-pw/2, cy-ph/2, pw, ph, m.Cell*0.4, render.ColorPanel)
	render.DrawGlow(dst, cx, cy, m.W*0.5, render.Alpha(render.ColorStarGlow, 0.22))

	title := fmt.Sprintf("%d Stickers!", k.milestoneCount)
	render.DrawTextShadowed(dst, title, cx, cy-ph*0.36, render.FitTextSize(title, m.TitleSize*1.0, pw*0.8), render.ColorText)

	render.DrawTextShadowed(dst, k.milestoneMsg, cx, cy-ph*0.14, render.FitTextSize(k.milestoneMsg, m.BodySize*1.15, pw*0.85), render.ColorStarGlow)

	n := len(k.milestoneEmojis)
	if n > 0 {
		step := math.Min(m.Cell*0.95, (pw*0.88)/float64(n))
		startX := cx - step*float64(n-1)/2
		for i, e := range k.milestoneEmojis {
			bob := math.Sin(ctx.T*4+float64(i)*0.8) * step * 0.07
			render.DrawEmoji(dst, render.EmojiName(e), startX+step*float64(i), cy+ph*0.12+bob, step*0.86, 0, 1)
		}
	}
	msg := "Tap to keep playing"
	render.DrawTextShadowed(dst, msg, cx, cy+ph*0.40, render.FitTextSize(msg, m.BodySize*0.9, pw*0.8), render.ColorTextDim)
}

// --- footer geometry ---------------------------------------------------------

const traySlotsPortrait = 8
const traySlotsLandscape = 4

func traySlots(m layout.Metrics) int {
	if m.Portrait {
		return traySlotsPortrait
	}
	return traySlotsLandscape
}

func trayRect(m layout.Metrics) layout.Rect {
	h := math.Min(m.Footer.H*0.40, m.MinTap*0.95)
	return layout.Rect{X: m.Footer.X, Y: m.Footer.Y + m.Footer.H*0.12, W: m.Footer.W, H: h}
}

// trayEmojiPos returns the centre and diameter of tray slot i (X, Y are the
// centre; W is the size).
func trayEmojiPos(m layout.Metrics, index int) layout.Rect {
	tr := trayRect(m)
	slots := traySlots(m)
	slot := index % slots
	step := (tr.W - tr.H*0.4) / float64(slots)
	size := math.Min(step*0.86, tr.H*0.70)
	x := tr.X + tr.H*0.2 + step*float64(slot) + step/2
	return layout.Rect{X: x, Y: tr.Y + tr.H/2, W: size, H: size}
}

// backButtonRect puts the only way out in the top-left corner, deliberately far
// from where a small hand rests. A big button along the bottom edge gets pressed
// by accident over and over; this one has to be reached for.
//
// It stays a full 48dp tap target - the point is to move it out of the way, not
// to make it fiddly for the grown-up.
func backButtonRect(m layout.Metrics) layout.Rect {
	size := math.Max(m.MinTap, m.Safe.W*0.13)
	return layout.Rect{X: m.Safe.X, Y: m.Safe.Y, W: size, H: size}
}
