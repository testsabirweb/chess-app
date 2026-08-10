package game

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/anim"
	"github.com/testsabirweb/chess-app/internal/challenge"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/sfx"
)

type playState int

const (
	stateIdle playState = iota
	stateMoving
	stateCelebrating
	stateMilestone
	stateAdvancing
)

// maxJourney bounds how far away the star may be planted, in moves.
const maxJourney = 3

// milestoneEvery is how many stickers earn the big celebration.
const milestoneEvery = 5

type PlayScene struct {
	game      *Game
	gen       *challenge.Generator
	cur       challenge.Challenge
	pieceType chess.PieceType

	// Live state of the current puzzle. The piece really moves across `board`,
	// so sliders, captures and blocked paths all stay correct move after move.
	board     *chess.Board
	at        chess.Square
	target    chess.Square
	solutions []chess.Square
	steps     int

	state         playState
	pieceSelected bool
	laidOut       bool

	moveTween anim.Tween
	starPulse anim.Pulse
	confetti  anim.Confetti
	sprites   *render.Sprites

	pieceX, pieceY float64
	fromX, fromY   float64
	toX, toY       float64
	moveTo         chess.Square
	starScale      float64
	starPopT       float64

	wobbleSq  chess.Square
	wobbleT   float64
	wobbleAmp float64

	hintBuf      []render.Hint
	reward       rewardFly
	rewardActive bool

	advanceT   float64
	advanceSwp bool

	milestoneT      float64
	milestoneCount  int
	milestoneEmojis []int
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

func NewPlayScene(g *Game, pt chess.PieceType) *PlayScene {
	spec := challenge.Spec{
		BoardWidth: 5, BoardHeight: 5,
		Pieces:   []chess.PieceType{pt},
		Color:    chess.White,
		MinMoves: 1,
		MaxMoves: maxJourney,
		Decoys:   pt == chess.Pawn,
	}
	ps := &PlayScene{
		game:      g,
		gen:       challenge.NewGenerator(spec, g.ctx.Rand),
		pieceType: pt,
		starPulse: anim.Pulse{Period: 1.6},
		sprites:   render.NewSprites(),
		moveTween: anim.Tween{Duration: 0.42, Ease: anim.EaseInOutCubic},
	}
	ps.newChallenge()
	return ps
}

func (p *PlayScene) newChallenge() {
	p.cur = p.gen.Next()
	p.board = p.cur.Board.Clone()
	p.at = p.cur.From
	p.target = p.cur.Target
	p.solutions = p.board.MoveTargets(p.at)
	p.steps = 0
	p.pieceSelected = false
	p.laidOut = false
}

func (p *PlayScene) syncPiecePos(m layout.Metrics) {
	cr := m.CellRect(int(p.at.File), int(p.at.Rank))
	p.pieceX, p.pieceY = cr.Center()
}

func (p *PlayScene) Update(ctx *Context) error {
	m := ctx.M
	if !p.laidOut {
		p.syncPiecePos(m)
		p.laidOut = true
	}

	p.starScale = p.starPulse.Update(ctx.DT)
	p.confetti.Update(ctx.DT, m.Cell*8)
	if p.starPopT > 0 {
		p.starPopT -= ctx.DT
	}
	if p.wobbleT > 0 {
		p.wobbleT -= ctx.DT
	}
	p.updateReward(ctx, m)

	home := homeButtonRect(m)
	for _, ev := range ctx.Pointer.Pressed() {
		if home.Contains(ev.X, ev.Y) {
			ctx.SFX.Play(sfx.SndButton)
			ctx.Switch(NewHomeScene(p.game))
			return nil
		}
		switch p.state {
		case stateIdle:
			p.handleTap(ctx, ev.X, ev.Y, m)
		case stateMilestone:
			p.milestoneT = 0
		}
	}

	switch p.state {
	case stateMoving:
		prog := p.moveTween.Update(ctx.DT)
		arc := -math.Sin(prog*math.Pi) * m.Cell * 0.28
		p.pieceX = lerp(p.fromX, p.toX, prog)
		p.pieceY = lerp(p.fromY, p.toY, prog) + arc
		if p.moveTween.Done() {
			p.land(ctx, m)
		}
	case stateCelebrating:
		if p.reward.phase == 2 {
			p.finishReward(ctx)
		}
	case stateMilestone:
		p.milestoneT -= ctx.DT
		if p.milestoneT <= 0 {
			p.state = stateAdvancing
			p.advanceT = 0
			p.advanceSwp = false
		}
	case stateAdvancing:
		p.advanceT += ctx.DT
		if !p.advanceSwp && p.advanceT >= advanceHalf {
			p.advanceSwp = true
			p.newChallenge()
			p.syncPiecePos(m)
			p.laidOut = true
		}
		if p.advanceT >= 2*advanceHalf {
			p.state = stateIdle
			p.moveTween.Reset()
		}
	}
	return nil
}

const advanceHalf = 0.22

func (p *PlayScene) handleTap(ctx *Context, x, y float64, m layout.Metrics) {
	f, r, ok := m.HitCell(x, y)
	if !ok {
		return
	}
	sq := chess.Sq(f, r)

	if !p.pieceSelected {
		// Any tap on the board picks the piece up. A toddler's instinct is to
		// tap the star, and answering that with a buzz teaches nothing; showing
		// them what the piece can do does.
		ctx.SFX.Play(sfx.SndButton)
		p.pieceSelected = true
		return
	}

	if sq == p.at {
		// Tapping the piece again puts it back down.
		ctx.SFX.Play(sfx.SndButton)
		p.pieceSelected = false
		return
	}

	// A generous magnet around the star, but only when the star is one move
	// away - otherwise the child would jump a step they have not earned.
	if containsSquare(p.solutions, p.target) && m.HitStar(x, y, p.target, p.solutions) {
		p.startMove(p.target, m)
		return
	}
	if containsSquare(p.solutions, sq) {
		p.startMove(sq, m)
		return
	}
	p.oops(ctx, sq, m)
}

func (p *PlayScene) oops(ctx *Context, sq chess.Square, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndOops)
	p.wobbleSq = sq
	p.wobbleT = 0.25
	p.wobbleAmp = m.Cell * 0.025
}

func (p *PlayScene) startMove(to chess.Square, m layout.Metrics) {
	cr := m.CellRect(int(to.File), int(to.Rank))
	p.fromX, p.fromY = p.pieceX, p.pieceY
	p.toX, p.toY = cr.Center()
	p.moveTo = to
	p.moveTween.Start()
	p.state = stateMoving
}

// land applies the finished move to the board and decides what happens next.
func (p *PlayScene) land(ctx *Context, m layout.Metrics) {
	captured := !p.board.At(p.moveTo).IsEmpty()
	p.board.Set(p.at, chess.Piece{})
	p.board.Set(p.moveTo, p.cur.Piece)
	p.at = p.moveTo
	p.solutions = p.board.MoveTargets(p.at)
	p.steps++
	p.syncPiecePos(m)

	if p.at == p.target {
		p.collectStar(ctx, m)
		return
	}

	if captured {
		ctx.SFX.Play(sfx.SndPop)
	} else {
		ctx.SFX.Play(sfx.SndStep)
	}
	// Keep the piece held so the next hop is a single tap.
	p.pieceSelected = true
	p.state = stateIdle

	if !challenge.CanReach(p.board, p.at, p.target, maxJourney+2) {
		p.relocateStar(ctx)
	}
}

// relocateStar hops the star somewhere the piece can still get to. It is the
// safety net for a wandering toddler: the game never becomes unwinnable and
// never scolds, the star just twinkles somewhere new.
func (p *PlayScene) relocateStar(ctx *Context) {
	steps := challenge.Reach(p.board, p.at, maxJourney)
	if len(steps) == 0 {
		// The piece is completely stuck (a pawn on the far rank); start over.
		p.state = stateAdvancing
		p.advanceT = 0
		p.advanceSwp = false
		return
	}
	far := steps[:0:0]
	for _, s := range steps {
		if s.Moves >= 2 {
			far = append(far, s)
		}
	}
	pool := steps
	if len(far) > 0 {
		pool = far
	}
	p.target = pool[ctx.Rand.IntN(len(pool))].Square
	ctx.SFX.Play(sfx.SndHop)
}

func (p *PlayScene) collectStar(ctx *Context, m layout.Metrics) {
	ctx.SFX.Play(sfx.SndLand)
	ctx.SFX.Play(sfx.SndCheer)
	ctx.SFX.Play(sfx.SndPop)

	cr := m.CellRect(int(p.target.File), int(p.target.Rank))
	cx, cy := cr.Center()
	p.confetti.Burst(ctx.Rand, cx, cy, 30, m.Cell)
	p.starPopT = 0.3
	p.pieceSelected = false
	p.state = stateCelebrating

	slot := trayEmojiPos(m, len(p.game.Stickers()))
	p.reward = rewardFly{
		emoji:    render.RandomEmoji(ctx.Rand),
		fromX:    cx,
		fromY:    cy,
		toX:      slot.X,
		toY:      slot.Y,
		popSize:  m.Cell * 0.78,
		traySize: slot.W,
		pop:      anim.Tween{Duration: 0.42, Ease: anim.EaseOutBack},
		fly:      anim.Tween{Duration: 0.55, Ease: anim.EaseInOutCubic},
	}
	p.reward.pop.Start()
	p.reward.x, p.reward.y = cx, cy
	p.reward.size = 0
	p.rewardActive = true
}

func (p *PlayScene) updateReward(ctx *Context, m layout.Metrics) {
	if !p.rewardActive {
		return
	}
	switch p.reward.phase {
	case 0:
		t := p.reward.pop.Update(ctx.DT)
		p.reward.size = p.reward.popSize * t
		p.reward.y = p.reward.fromY - m.Cell*0.25*t
		if p.reward.pop.Done() {
			p.reward.phase = 1
			p.reward.popDoneX, p.reward.popDoneY = p.reward.x, p.reward.y
			p.reward.fly.Start()
		}
	case 1:
		t := p.reward.fly.Update(ctx.DT)
		p.reward.x = lerp(p.reward.popDoneX, p.reward.toX, t)
		// A shallow arc so it looks tossed into the tray, not dragged.
		p.reward.y = lerp(p.reward.popDoneY, p.reward.toY, t) - math.Sin(t*math.Pi)*m.Cell*0.55
		p.reward.size = lerp(p.reward.popSize, p.reward.traySize, t)
		if p.reward.fly.Done() {
			p.reward.phase = 2
		}
	}
}

func (p *PlayScene) finishReward(ctx *Context) {
	p.rewardActive = false
	total := p.game.AddSticker(p.reward.emoji)
	if total%milestoneEvery == 0 {
		ctx.SFX.Play(sfx.SndMilestone)
		p.state = stateMilestone
		p.milestoneT = 3.4
		p.milestoneCount = total
		all := p.game.Stickers()
		p.milestoneEmojis = all[len(all)-milestoneEvery:]
		return
	}
	p.state = stateAdvancing
	p.advanceT = 0
	p.advanceSwp = false
}

// --- drawing -----------------------------------------------------------------

func (p *PlayScene) Draw(dst *ebiten.Image, ctx *Context) {
	m := ctx.M
	render.DrawBackground(dst, m)

	p.drawHeader(dst, ctx, m)
	render.DrawBoard(dst, m)

	// Hints and the pick-me halo.
	if p.state == stateIdle {
		if p.pieceSelected {
			render.DrawMoveHints(dst, m, p.hints())
			render.DrawPickableRing(dst, m, p.at, p.starScale, true)
		} else {
			render.DrawPickableRing(dst, m, p.at, p.starScale, false)
		}
	}

	if p.wobbleT > 0 {
		render.DrawSquareTint(dst, m, p.wobbleSq, 0, 0, render.Alpha(render.ColorShadow, 0.5))
	}

	if p.state != stateCelebrating || p.starPopT > 0 {
		p.drawStar(dst, ctx, m)
	}

	// Every other piece on the board (pawn decoys).
	for _, sq := range p.board.Occupied() {
		if sq == p.at {
			continue
		}
		cr := m.CellRect(int(sq.File), int(sq.Rank))
		render.DrawPiece(dst, p.board.At(sq), cr, 0, true)
	}

	p.drawPiece(dst, m)
	render.DrawConfetti(dst, &p.confetti, p.sprites, m.Cell)

	if p.rewardActive && p.reward.size > 0 {
		render.DrawEmoji(dst, render.EmojiName(p.reward.emoji), p.reward.x, p.reward.y, p.reward.size, 0, 1)
	}

	p.drawFooter(dst, ctx, m)

	if p.state == stateMilestone {
		p.drawMilestone(dst, ctx, m)
	}
	if p.state == stateAdvancing {
		a := 1 - math.Abs(p.advanceT/advanceHalf-1)
		render.DrawFilledRect(dst, 0, 0, m.W, m.H, render.Alpha(render.ColorBGTop, clamp01(a)*0.85))
	}
}

// hints describes each legal destination for the renderer, reusing one buffer
// so drawing allocates nothing.
func (p *PlayScene) hints() []render.Hint {
	p.hintBuf = p.hintBuf[:0]
	for _, sq := range p.solutions {
		p.hintBuf = append(p.hintBuf, render.Hint{
			Square:  sq,
			Capture: !p.board.At(sq).IsEmpty(),
			Target:  sq == p.target,
		})
	}
	return p.hintBuf
}

func (p *PlayScene) drawStar(dst *ebiten.Image, ctx *Context, m layout.Metrics) {
	scale := p.starScale
	if p.starPopT > 0 {
		// Collected: the star flashes bigger and fades out.
		t := 1 - p.starPopT/0.3
		scale = p.starScale * (1 + t*1.4)
	}
	render.DrawStar(dst, p.target, m, scale, ctx.T)
}

func (p *PlayScene) drawPiece(dst *ebiten.Image, m layout.Metrics) {
	wx := 0.0
	if p.wobbleT > 0 {
		wx, _ = render.WobbleOffset(1-p.wobbleT/0.25, p.wobbleAmp)
	}
	cr := layout.Rect{X: p.pieceX - m.Cell/2, Y: p.pieceY - m.Cell/2, W: m.Cell, H: m.Cell}
	cr.X += wx
	lift := 0.0
	if p.pieceSelected && p.state == stateIdle {
		lift = m.Cell * 0.08
	}
	if p.state == stateMoving {
		lift = m.Cell * 0.05
	}
	render.DrawPiece(dst, p.cur.Piece, cr, lift, true)
}

func (p *PlayScene) drawHeader(dst *ebiten.Image, ctx *Context, m layout.Metrics) {
	h := m.Header
	cx := h.X + h.W/2
	rowY := h.Y + h.H*0.42
	icon := math.Min(h.H*0.6, m.Cell*0.9)

	// piece  · · ·  star   - a wordless "get this there" instruction.
	render.DrawPiece(dst, chess.Piece{Type: p.pieceType, Color: chess.White},
		layout.Rect{X: cx - h.W*0.30 - icon/2, Y: rowY - icon/2, W: icon, H: icon}, 0, false)
	for i := 0; i < 3; i++ {
		x := cx + h.W*0.15*(float64(i)-1)
		render.FillCircleSoft(dst, x, rowY, icon*0.09, render.Alpha(render.ColorText, 0.5))
	}
	render.DrawEmoji(dst, render.StarEmoji, cx+h.W*0.30, rowY, icon*0.85, math.Sin(ctx.T*1.6)*0.12, 1)

	name := render.PieceName(p.pieceType)
	size := render.FitTextSize(name, m.BodySize*1.1, h.W*0.5)
	render.DrawTextShadowed(dst, name, cx, h.Y+h.H*0.87, size, render.ColorTextDim)
}

func (p *PlayScene) drawFooter(dst *ebiten.Image, ctx *Context, m layout.Metrics) {
	tr := trayRect(m)
	render.FillRoundRect(dst, tr.X, tr.Y, tr.W, tr.H, tr.H*0.35, render.ColorTray)

	// Empty slots are drawn too, so the tray reads as a row waiting to be
	// filled rather than one lonely sticker in a wide bar.
	for i := 0; i < traySlots; i++ {
		slot := trayEmojiPos(m, i)
		render.FillCircleSoft(dst, slot.X, slot.Y, slot.W*0.30, render.ColorTraySlot)
	}

	stickers := p.game.Stickers()
	show := stickers
	if len(show) > traySlots {
		show = show[len(show)-traySlots:]
	}
	base := len(stickers) - len(show)
	for i, e := range show {
		slot := trayEmojiPos(m, base+i)
		render.DrawEmoji(dst, render.EmojiName(e), slot.X, slot.Y, slot.W, 0, 1)
	}

	home := homeButtonRect(m)
	render.DrawChunkyButton(dst, home.X, home.Y, home.W, home.H, render.ColorHome, render.ColorHomeEdge, false)
	render.DrawEmoji(dst, "1f3e0", home.X+home.W*0.26, home.Y+home.H/2, home.H*0.6, 0, 1)
	render.DrawTextShadowed(dst, "Home", home.X+home.W*0.60, home.Y+home.H/2, render.FitTextSize("Home", m.BodySize, home.W*0.45), render.ColorText)
}

func (p *PlayScene) drawMilestone(dst *ebiten.Image, ctx *Context, m layout.Metrics) {
	render.DrawFilledRect(dst, 0, 0, m.W, m.H, render.Alpha(render.ColorBGTop, 0.88))
	cx := m.W / 2
	cy := m.H * 0.44

	pw := m.Safe.W
	ph := m.Cell * 3.6
	render.FillRoundRect(dst, cx-pw/2, cy-ph/2, pw, ph, m.Cell*0.4, render.ColorPanel)
	render.DrawGlow(dst, cx, cy, m.W*0.5, render.Alpha(render.ColorStarGlow, 0.22))

	title := fmt.Sprintf("%d Stickers!", p.milestoneCount)
	render.DrawTextShadowed(dst, title, cx, cy-ph*0.32, render.FitTextSize(title, m.TitleSize*1.1, pw*0.8), render.ColorText)

	n := len(p.milestoneEmojis)
	if n > 0 {
		step := math.Min(m.Cell*0.95, (pw*0.88)/float64(n))
		startX := cx - step*float64(n-1)/2
		for i, e := range p.milestoneEmojis {
			bob := math.Sin(ctx.T*4+float64(i)*0.8) * step * 0.07
			render.DrawEmoji(dst, render.EmojiName(e), startX+step*float64(i), cy+ph*0.04+bob, step*0.86, 0, 1)
		}
	}
	msg := "Tap to keep playing"
	render.DrawTextShadowed(dst, msg, cx, cy+ph*0.36, render.FitTextSize(msg, m.BodySize*0.9, pw*0.8), render.ColorTextDim)
}

// --- footer geometry ---------------------------------------------------------

const traySlots = 8

func trayRect(m layout.Metrics) layout.Rect {
	h := math.Min(m.Footer.H*0.40, m.MinTap*0.95)
	return layout.Rect{X: m.Footer.X, Y: m.Footer.Y + m.Footer.H*0.12, W: m.Footer.W, H: h}
}

// trayEmojiPos returns the centre and diameter of tray slot i (X, Y are the
// centre; W is the size).
func trayEmojiPos(m layout.Metrics, index int) layout.Rect {
	tr := trayRect(m)
	slot := index % traySlots
	step := (tr.W - tr.H*0.4) / traySlots
	size := math.Min(step*0.86, tr.H*0.70)
	x := tr.X + tr.H*0.2 + step*float64(slot) + step/2
	return layout.Rect{X: x, Y: tr.Y + tr.H/2, W: size, H: size}
}

func homeButtonRect(m layout.Metrics) layout.Rect {
	tr := trayRect(m)
	top := tr.Y + tr.H + m.Footer.H*0.10
	avail := m.Footer.Y + m.Footer.H - top
	h := math.Max(math.Min(avail*0.78, m.MinTap*1.6), m.MinTap)
	w := math.Min(m.Safe.W*0.60, m.MinTap*4.5)
	y := top + math.Max(0, (avail-h)/2)
	return layout.Rect{X: m.Safe.X + (m.Safe.W-w)/2, Y: y, W: w, H: h}
}

// --- small helpers -----------------------------------------------------------

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func containsSquare(list []chess.Square, sq chess.Square) bool {
	for _, s := range list {
		if s == sq {
			return true
		}
	}
	return false
}
