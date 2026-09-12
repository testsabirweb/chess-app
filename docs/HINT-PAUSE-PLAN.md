# Slowing the hint speed-run

## Context

The play scene shows every legal destination as mint dots the moment the piece is picked up, and
keeps them on screen until the piece moves. A toddler playing it found the obvious exploit: tap
anywhere to light up the dots, then tap whichever dot sits closest to the star, and repeat. It is
visual pathfinding over the hint overlay — the piece's identity and movement rule never enter into it.

Worth being clear about what is and is not broken: he *cannot* tap "the dot nearest the star" without
the dots, and the dot set **is** the piece's move rule rendered. Every move he makes is a choice from a
legally generated set, so he is reading the piece's reach on every turn. What is missing is
**prediction** — generating that set in his head before seeing it — and **naming**. Both are later
skills, and latching onto the most reliable visual cue is exactly what a 2–4 year old should be doing.
So: do not discourage the behaviour, change which cue is the most reliable one.

There is also a structural reason he has no incentive to slow down. `relocateStar` guarantees the
puzzle stays solvable and there is no move limit, so **greedy tapping is strictly optimal**. Nothing in
the loop costs him anything for going fast. That is the thing to fix.

## The three changes

1. **Pause before the hints appear.** Picking the piece up no longer shows the dots; they fade in after
   ~2.5 seconds. A child who already knows where the piece can go never waits, because tapping a legal
   square moves the piece whether the dots are showing or not. The delay only costs something when you
   were going to read the overlay instead of the board. The dots stop being a map to scan and become an
   answer that arrives after you have had a go at working it out.

2. **Put the piece down after every hop.** Today `land()` re-selects it, so the overlay stays lit from
   the first tap of a puzzle to the last — one continuous trail to the star. Dropping it means every hop
   starts from a clean board and earns its own look before the dots come back.

3. **A "perfect" celebration for the shortest route.** Solving in the fewest possible moves gets double
   confetti, a chime, a gold glow on the sticker and the word `Perfect!` where the piece name sits.
   Wandering still earns the same sticker, so there is nothing to lose by exploring — only something
   extra to win by looking first. This is what stops greedy from being the *best* strategy without ever
   making it a losing one.

Nothing here adds a fail state, a timer, or a harsher response to a wrong tap.

---

## Edits

### `internal/render/board.go`

`DrawMoveHints` gains a `fade float64` parameter and multiplies every colour through `Alpha`:

```go
// DrawMoveHints marks every square the piece can move to right now. The marks
// are still, not pulsing - they are information, not decoration.
//
// A square holding a piece gets a ring rather than a dot, so the piece stays
// readable; the star's square gets neither, because the star is already the
// clearest possible marker.
//
// fade scales the whole overlay, so the hints can arrive gently once the
// child has had a moment to look at the board rather than snapping on under
// their finger.
func DrawMoveHints(dst *ebiten.Image, m layout.Metrics, hints []Hint, fade float64) {
	if fade <= 0 {
		return
	}
	if fade > 1 {
		fade = 1
	}
	for _, h := range hints {
		cr := m.CellRect(int(h.Square.File), int(h.Square.Rank))
		cx, cy := cr.Center()
		DrawFilledRect(dst, cr.X, cr.Y, cr.W, cr.H, Alpha(ColorHint, fade))
		switch {
		case h.Target:
			// nothing: the star speaks for itself
		case h.Capture:
			FillRingSoft(dst, cx, cy, cr.W*0.42, Alpha(ColorHintDot, fade))
		default:
			r := cr.W * 0.16
			FillCircleSoft(dst, cx, cy, r*1.5, Alpha(ColorHintRing, 0.30*fade))
			FillCircleSoft(dst, cx, cy, r, Alpha(ColorHintDot, fade))
		}
	}
}
```

The palette is premultiplied (`rgba()` scales RGB by A), so `Alpha` composes correctly — which is why
the nested `Alpha(ColorHintRing, 0.30)` becomes a single `0.30*fade`.

### `internal/game/game.go`

Add a field to `Game`:

```go
	// hintDelay is how long a picked-up piece waits before its legal moves are
	// shown. Held here rather than as a constant so the screenshot tool can
	// switch the pause off; see defaultHintDelay for why it exists.
	hintDelay float64
```

and initialise it in `New()` alongside the other defaults:

```go
		hintDelay: defaultHintDelay,
```

(`New()`'s struct literal will need re-aligning; run `gofmt`.)

### `internal/game/play.go`

**New constants**, next to `milestoneEvery`:

```go
// defaultHintDelay is how long the piece sits picked up before the legal-move
// dots fade in, and hintFadeIn is how long they take to arrive once it is over.
//
// The pause is the whole point. A child who already knows where the piece can
// go never waits for it - tapping a legal square moves the piece whether the
// dots are showing or not - so the delay only costs something when you were
// going to read the overlay instead of the board. That is precisely the habit
// it exists to interrupt.
const (
	defaultHintDelay = 2.5
	hintFadeIn       = 0.45
)
```

**New `PlayScene` fields**, after `laidOut`:

```go
	// hintT counts the pause down after the piece is picked up; the dots fade
	// in over its last hintFadeIn seconds.
	hintT float64

	// optimal is the fewest moves this puzzle can be solved in from where the
	// piece started, kept in step with the star when it relocates. Matching it
	// earns the bigger celebration.
	optimal int
	perfect bool
```

**`newChallenge()`** — `Challenge.Moves` already carries the shortest solution, so no new search:

```go
	p.steps = 0
	p.optimal = p.cur.Moves
	p.perfect = false
	p.pieceSelected = false
	p.hintT = 0
	p.laidOut = false
```

**`Update()`** — tick the pause down alongside the other timers:

```go
	if p.hintT > 0 {
		p.hintT -= ctx.DT
	}
```

**`handleTap()`** — route the pick-up through a helper:

```go
	if !p.pieceSelected {
		// Any tap on the board picks the piece up. A toddler's instinct is to
		// tap the star, and answering that with a buzz teaches nothing; showing
		// them what the piece can do does - after a beat to look first.
		p.pickUp(ctx)
		return
	}
```

**Two new helpers**, above `oops`:

```go
// pickUp holds the piece and starts the pause before its moves are shown.
func (p *PlayScene) pickUp(ctx *Context) {
	ctx.SFX.Play(sfx.SndButton)
	p.pieceSelected = true
	p.hintT = p.game.hintDelay
}

// hintFade is how strongly the move dots are showing: nothing during the
// pause, ramping to full over its last hintFadeIn seconds.
func (p *PlayScene) hintFade() float64 {
	if !p.pieceSelected || p.state != stateIdle {
		return 0
	}
	if p.hintT <= 0 {
		return 1
	}
	if p.hintT >= hintFadeIn {
		return 0
	}
	return 1 - p.hintT/hintFadeIn
}
```

**`land()`** — replace the "keep the piece held" block:

```go
	// Put the piece down again. Holding it across the whole journey left the
	// dots up from the first tap to the last, which turns the overlay into a
	// trail to follow to the star; dropping it means every hop starts from a
	// clean board and earns its own look before the dots come back.
	p.pieceSelected = false
	p.hintT = 0
	p.state = stateIdle
```

**`relocateStar()`** — keep `optimal` honest when the star hops, instead of discarding the `Step`:

```go
	pick := pool[ctx.Rand.IntN(len(pool))]
	p.target = pick.Square
	// The moves already spent still count, so a wandering journey can no longer
	// come out "perfect" - but it is never scored as a failure either.
	p.optimal = p.steps + pick.Moves
	ctx.SFX.Play(sfx.SndHop)
```

**`collectStar()`** — decide the celebration before the burst:

```go
	// Solving it in the fewest moves gets a louder party: twice the confetti, a
	// chime, a gold glow on the sticker and a word the grown-up can read out.
	// Wandering still earns the same sticker, so there is nothing to lose by
	// exploring - only something extra to win by looking first.
	p.perfect = p.optimal > 0 && p.steps == p.optimal
	burst := 30
	if p.perfect {
		burst = 60
		ctx.SFX.Play(sfx.SndMilestone)
	}

	cr := m.CellRect(int(p.target.File), int(p.target.Rank))
	cx, cy := cr.Center()
	p.confetti.Burst(ctx.Rand, cx, cy, burst, m.Cell)
```

**`Draw()`** — pass the fade:

```go
			render.DrawMoveHints(dst, m, p.hints(), p.hintFade())
```

and put a gold glow behind a perfect sticker as it flies:

```go
	if p.rewardActive && p.reward.size > 0 {
		if p.perfect {
			render.DrawGlow(dst, p.reward.x, p.reward.y, p.reward.size, render.Alpha(render.ColorStarGlow, 0.45))
		}
		render.DrawEmoji(dst, render.EmojiName(p.reward.emoji), p.reward.x, p.reward.y, p.reward.size, 0, 1)
	}
```

**`drawHeader()`** — borrow the piece-name slot for the cheer, so nothing moves:

```go
	// Just the piece's name. The board says everything else, and anything more
	// up here is one more thing pulling the eye away from the puzzle. The one
	// exception is the shortest-route cheer, which borrows the same slot while
	// the confetti is falling so nothing moves.
	h := m.Header
	name, clr := render.PieceName(p.pieceType), render.ColorTextDim
	if p.perfect && (p.state == stateCelebrating || p.state == stateMilestone) {
		name, clr = "Perfect!", render.ColorStarGlow
	}
	size := render.FitTextSize(name, m.BodySize*1.25, h.W*0.6)
	render.DrawTextShadowed(dst, name, h.X+h.W/2, h.Y+h.H*0.72, size, clr)
```

### `internal/game/debug.go` and `cmd/shot/main.go` — required, not optional

Dropping the piece between hops breaks the screenshot script: `tapTowardStar` taps a destination while
nothing is selected, which now only picks the piece up, and the walk stalls.

In `debug.go`, add `Selected bool` to `PlayInfo` and populate it from `ps.pieceSelected`, and add:

```go
// SetHintDelay overrides the pause before a picked-up piece shows its moves.
// The screenshot tool sets it to zero: the shots exist to show what the hinted
// board looks like, not to sit through the wait first.
func (g *Game) SetHintDelay(d float64) { g.hintDelay = d }
```

In `cmd/shot/main.go`, `tapTowardStar` queues the pick-up tap first when it needs one — both taps land
in the same `Update` and are handled in order, so one call still completes a hop:

```go
	if !info.Selected {
		g.TapSquare(int(info.Piece.File), int(info.Piece.Rank))
	}
	g.TapSquare(int(pick.File), int(pick.Rank))
```

Add a `-hintdelay` flag (default `0`) and apply it to the play scene, shifting the post-tap frames so
the script can optionally watch the dots actually arrive:

```go
	hintDelay := flag.Float64("hintdelay", 0, "seconds a picked-up piece waits before its moves show (0 = no wait)")
```

```go
		g = game.NewInPlay(pieceByName(*piece))
		// The pause before the hints is off by default: these shots are for
		// looking at the hinted board, not for sitting through the wait. Pass
		// -hintdelay to watch the dots actually arrive.
		g.SetHintDelay(*hintDelay)
		g.SeedStickers(*seed)
		for i := 0; i < *skip; i++ {
			g.NextChallenge()
		}
		// wait is how long the script must hold after picking the piece up
		// before the dots are on screen. Only the hint shot needs it; the moves
		// themselves go through whether the dots are showing or not.
		wait := int(*hintDelay * 60)
		steps = []step{
			{frame: 30, shot: "01-idle"},
			{frame: 34, do: tapPiece},
			{frame: 46, shot: "02a-looking"},
			{frame: 55 + wait, shot: "02-hints"},
			{frame: 60 + wait, do: tapTowardStar},
			{frame: 70 + wait, shot: "03-moving"},
			{frame: 95 + wait, do: tapTowardStar},
			{frame: 108 + wait, shot: "04-second-hop"},
			{frame: 130 + wait, do: tapTowardStar},
			{frame: 152 + wait, shot: "05-reward-pop"},
			{frame: 175 + wait, shot: "06-reward-fly"},
			{frame: 200 + wait, shot: "07-milestone"},
			{frame: 235 + wait, shot: "08-next"},
		}
```

---

## Verifying

```
gofmt -w internal/render/board.go internal/game/game.go \
      internal/game/play.go internal/game/debug.go cmd/shot/main.go
go build ./...
go vet ./...
go test ./...     # all packages pass unchanged
make shots        # 02a-looking has no dots, 02-hints does
make run
```

To watch the pause itself:

```
go run ./cmd/shot -out /tmp/pause -scene play -piece knight -hintdelay 2.5
```

---

## Tuning, once he has played it

`defaultHintDelay` is the one number to touch. 2.5s is a starting guess:

- taps and then waits blankly → drop to 1.5s
- still speed-running → push to 4s

The natural next step, once you have watched him, is to **ramp** it with his sticker count — hints
instant for the first few puzzles, lengthening as he gets fluent. That is the Dragonbox pattern:
remove the scaffolding rather than add difficulty.

## Deliberately not changed

`relocateStar` stays exactly as it is. It is what keeps the game gentle and unlosable, and it should
not be traded away. It does mean greedy tapping still always eventually works — change 3 is what makes
it no longer the *fastest* route, which is the whole point.

## Not code: the part that will do the most work

The pause is only half of it. Fill the gap with a ritual:

- **"Point first, then tap."** He points at where he thinks the piece can go, *then* the dots appear.
  Predict-then-reveal turns the 2.5s wait from dead time into a guessing game with a payoff — the same
  loop as peekaboo.
- **"Where can Rook go?"**, in those exact words, every single time. Ritual phrasing beats variety here.
- **Take turns.** Watching an adult pause before moving models the pause better than any UI can.
- **Narrate the piece, never the goal.** "The horse *jumps*!" rather than "get to the star" — you are
  competing with the star for his attention, so do not reinforce it.

Two further ideas worth building later, both aimed at piece identity rather than the hint loop: a ~1.5s
intro animation before each puzzle showing the piece alone with ghost arrows sweeping its move pattern,
and saying the piece's name out loud on puzzle start. At this age a parent's recorded voice outperforms
almost anything on screen.

---

# Addendum: a green scarf on the dark-square bishop

## Why this one is easy

At home there are two wooden bishops and one wears a green scarf, so he can tell the dark-square one
from the light-square one by looking. Copying that into the game costs almost nothing, because of the
property the scarf is teaching in the first place: **a bishop can never change square colour.** The
scarf goes on when the puzzle is dealt and stays on for the whole journey — including mid-hop, while
the piece is animating and `p.at` is still the square it left. Nothing can make it flicker.

Only the dark-square bishop gets one, matching the physical set: one scarfed, one plain.

## Where the neck is

Measured, not guessed. Rasterising the cburnett bishop at 45px (its own viewBox units) and taking the
silhouette width per row:

```
y=18..22  width=19   <- the mitre, widest part of the head
y=26      width=13   <- narrowest point of the whole body: the neck
y=27      width=15
y=30      width=17
y=37      width=34   <- the foot
```

The waist at `y=26` is where the artwork already strokes a collar line (`M17.5 26h10` in the SVG). The
scarf sits just under it, slightly overhanging, so it reads as wrapped round the neck rather than
inlaid into it. The piece is symmetric about `x=22.5`, i.e. dead centre.

## Edits

### `internal/render/palette.go`

```go
	// The dark-square bishop's scarf. A deeper grass green than the mint hint
	// dots on purpose: the scarf sits on the board and must never read as "you
	// can move here". If his real scarf is a different green, use that instead -
	// the physical association is what does the teaching, not the hue.
	ColorScarf     = color.RGBA{46, 160, 67, 255}
	ColorScarfEdge = color.RGBA{20, 62, 30, 255}
```

### `internal/render/board.go`

Next to `DrawBoard`, so the parity rule lives beside the code that paints it:

```go
// DarkSquare reports whether a square is painted in the board's dark colour.
// It is the same parity DrawBoard uses - keep the two in step.
func DarkSquare(sq chess.Square) bool {
	return (int(sq.File)+int(sq.Rank))%2 == 1
}
```

### `internal/render/pieces.go`

```go
// scarfBand places the scarf inside the cburnett bishop's 45-unit viewBox.
// The numbers come from the rasterised silhouette: the body is at its
// narrowest at y=26 (13 units across, against 19 for the mitre above and 34
// for the foot below), which is the collar the artwork already draws a line
// on.
const (
	scarfCentreY = 27.0 / 45
	scarfWidth   = 16.0 / 45
	scarfHeight  = 3.6 / 45
	scarfEdge    = 1.0 / 45
)

// DrawBishopScarf ties a scarf round the bishop's neck, marking it as the one
// that runs on the dark squares - the same trick as the green scarf on the
// wooden set.
//
// r and lift must be the values passed to DrawPiece for the same piece. The
// piece SVG is square, so DrawPiece lands it in a box of side r.W centred
// vertically in r; the scarf is positioned off that box, not off r itself.
func DrawBishopScarf(dst *ebiten.Image, r layout.Rect, lift float64) {
	side := r.W
	top := r.Y + (r.H-side)/2 - lift
	cx := r.X + side/2
	cy := top + side*scarfCentreY

	w, h := side*scarfWidth, side*scarfHeight
	e := side * scarfEdge

	// A dark edge first, so the green keeps its shape against both the white
	// piece and the dark square it is standing on.
	FillRoundRect(dst, cx-w/2-e, cy-h/2-e, w+2*e, h+2*e, (h+2*e)/2, ColorScarfEdge)
	FillRoundRect(dst, cx-w/2, cy-h/2, w, h, h/2, ColorScarf)

	// A short tail hanging off one side. It is what makes the band read as a
	// scarf rather than a stripe.
	tw, th := w*0.26, h*1.6
	tx, ty := cx+w*0.20, cy+h*0.30
	FillRoundRect(dst, tx-e, ty-e, tw+2*e, th+2*e, (tw+2*e)/2, ColorScarfEdge)
	FillRoundRect(dst, tx, ty, tw, th, tw/2, ColorScarf)
}
```

### `internal/game/play.go`

In `drawPiece`, straight after the existing `render.DrawPiece(...)` call — and after the wobble offset
has already been applied to `cr`, so the scarf shakes with the piece:

```go
	render.DrawPiece(dst, p.cur.Piece, cr, lift, true)
	// The dark-square bishop wears a scarf, the same way the wooden set at home
	// has one with a scarf and one without. A bishop can never change square
	// colour, so it goes on when the puzzle is dealt and stays on for the whole
	// journey - including mid-hop, when p.at is still the square it left.
	if p.cur.Piece.Type == chess.Bishop && render.DarkSquare(p.at) {
		render.DrawBishopScarf(dst, cr, lift)
	}
```

Nothing is needed for the decoy pieces drawn in the loop above: decoys are only ever black pawns.

## Verifying

```
go test ./...
go run ./cmd/shot -out /tmp/scarf -scene play -piece bishop
go run ./cmd/shot -out /tmp/scarf2 -scene play -piece bishop -skip 1
```

The bishop's starting square is random, so run it with a couple of different `-skip` values until you
have seen both a scarfed and a bare bishop. Check the band sits on the neck, under the mitre and above
the flare of the foot, and that it does not read as a hint dot.

A one-line test is worth having so the parity can never drift from `DrawBoard`:

```go
func TestDarkSquareMatchesBoardParity(t *testing.T) {
	if !DarkSquare(chess.Sq(1, 0)) || DarkSquare(chess.Sq(0, 0)) {
		t.Fatal("DarkSquare disagrees with DrawBoard's (f+row)%2 checker")
	}
}
```

## Two things to decide

- **The green.** `{46, 160, 67}` is a grass green chosen to stay clear of the mint hint dots
  (`ColorHintDot` is `{24, 176, 122}`). If the real scarf is a different green, use the real one — the
  whole value here is the association with the object in his hand.
- **The home screen and header bishops** have no square, so they get no scarf. That is probably right,
  but if he asks where the scarf went, that is a good sign and worth a rethink.
