# Theme v2 — warm, alive, still focused

**This supersedes the palette half of `CALMER-PALETTE-PLAN.md`.** The emoji addendum in that file still
stands and is already applied; the colour values in it are replaced by everything below.

---

## What went wrong

The last pass asked for "calmer" and got "dead". Three separate causes, worth separating because only
one of them was a taste error:

**1. Desaturated colour at mid lightness is the universal signal for "disabled".** The six piece cards
kept their original brightness and lost their chroma, which is exactly the recipe every UI toolkit uses
to grey a button out. They do not read as calm, they read as switched off.

**2. A neutral grey ground reads as absence, not restraint.** Dark is right. Dark *with no hue at all*
is what makes a screen feel unfinished — there is nothing to say the colour was chosen rather than
skipped.

**3. There is an actual rendering bug making the background look dirty.** See the fix below. On the old
saturated purple it passed as a design flourish; on flat grey it just looks like a smudge, and it is a
big part of why the current screen feels grubby rather than quiet.

## The corrected principle

The last plan said the wrong thing. The right version:

> Focus comes from **how few things compete** and from **value contrast** — not from draining colour.
> A handful of genuinely saturated things on a deep, warm, low-value ground reads as both alive and
> calm. Mid-tone mud everywhere reads as neither.

And a distinction the last plan missed entirely:

> **The home screen and the play screen have different jobs.** Home is a choosing screen — it should be
> colourful and inviting, and there is no puzzle to distract from. Play is a focus screen. Applying one
> rule to both is what flattened the home screen.

## The direction: chess.com's warm green

Taking the reference you suggested. What makes it work is not the specific green, it is that the ground
is a **warm charcoal** rather than a cool grey, and the board is **cream and green** rather than white
and grey. Warm dark is cosy; cool dark is clinical. For a toddler that difference is most of the
"likeable" and costs nothing in distraction, because the ground stays dark and quiet either way.

One consequence to plan for: **once the board is green, green can no longer mean "you can move here".**
The move dots move to blue. That is the only Tier 1 change in this plan, and it is forced.

---

## Step 1 — `internal/render/shapes.go`: fix the gradient bug first

Do this before touching any colour, or you will be judging the new palette through the same smudge.

`gradientImage` builds a **1 pixel wide** by 256 tall image. `DrawVerticalGradient` then scales it to
the full screen width (about 1080px) with `ebiten.FilterLinear`. Bilinear filtering on a source one
texel wide has nothing valid to interpolate against horizontally, so it samples off the edge of the
texture and falls off — which is the broad vertical light band down the right of every screenshot. It
is not a gradient anyone designed.

In `DrawVerticalGradient`, change the filter:

```go
	op.Filter = ebiten.FilterNearest  // was ebiten.FilterLinear
```

and in `gradientImage`, raise the row count so the vertical blend stays smooth without interpolation:

```go
	const n = 512  // was 256
```

`op.GeoM.Scale(w, h/256)` in `DrawVerticalGradient` must change to `h/512` to match, or the gradient
will only cover half the screen. Check for any other use of the 256 constant before you finish.

Nearest is correct here: the image varies only vertically, so there is nothing to gain from filtering
horizontally and everything to lose.

---

## Step 2 — `internal/render/palette.go`

### Ground — warm charcoal (Tier 3)

```go
	// Background: a warm, near-flat charcoal. Dark enough to stay out of the
	// way, warm enough to read as a chosen colour rather than an absence. Cool
	// neutral grey was the mistake here - it made the whole screen feel unfinished.
	ColorBGTop    = color.RGBA{48, 45, 41, 255}  // was {38, 41, 52, 255}
	ColorBGBottom = color.RGBA{34, 32, 29, 255}  // was {28, 30, 39, 255}
```

### Board — cream and green (Tier 2)

```go
	// Board: the classic cream-and-green. High value contrast keeps the light
	// and dark squares obviously different, which the bishop's scarf depends on,
	// and the warm cream flatters the gold star far better than a cool white.
	ColorBoardL = color.RGBA{238, 238, 210, 255}  // was {244, 242, 238, 255}
	ColorBoardD = color.RGBA{118, 150, 86, 255}   // was {150, 152, 156, 255}

	// Frame: warm wood. Not gold - gold belongs to the star alone - but not the
	// drab taupe it was either.
	ColorFrame      = color.RGBA{140, 108, 74, 255}  // was {150, 138, 120, 255}
	ColorFrameInner = color.RGBA{96, 72, 48, 255}    // was {108, 98, 84, 255}
```

### Move hints — green board forces these to blue (Tier 1)

```go
	// Move hints. Blue, not the old mint: the board is green now, and a green
	// dot on a green square is no signal at all. Blue is the furthest usable hue
	// from both the board and the gold star, and reads hard against cream and
	// green alike.
	ColorHint     = rgba(70, 150, 255, 0.32)   // was rgba(60, 220, 160, 0.30)
	ColorHintDot  = rgba(28, 110, 220, 0.95)   // was rgba(24, 176, 122, 0.95)
	ColorHintRing = rgba(255, 255, 255, 0.80)  // was 0.75
```

This is the one change here that touches the attention work. **Keep the dots at full strength** — the
hue changes, the salience must not. If anything they should read slightly harder than before.

### Piece cards — put the colour back (Tier 2)

Saturated again, but pitched a little deeper than the originals so they sit on warm charcoal instead of
glaring off it. On the home screen there is no puzzle to compete with, so this is free.

```go
var PieceCardColors = []color.RGBA{
	{230, 106, 106, 255}, // pawn   - coral   (was {198, 124, 124})
	{235, 150,  58, 255}, // knight - orange  (was {200, 158, 102})
	{ 92, 180, 120, 255}, // bishop - green   (was {118, 168, 132})
	{ 84, 150, 225, 255}, // rook   - blue    (was {116, 150, 190})
	{168, 116, 224, 255}, // queen  - purple  (was {158, 134, 186})
	{235, 186,  76, 255}, // king   - yellow  (was {198, 180, 120})
}

var PieceCardEdges = []color.RGBA{
	{158,  62,  62, 255}, // was {140,  82,  82}
	{162,  98,  28, 255}, // was {142, 108,  64}
	{ 52, 128,  80, 255}, // was { 78, 116,  88}
	{ 44, 100, 168, 255}, // was { 76, 102, 132}
	{114,  68, 164, 255}, // was {108,  90, 128}
	{166, 126,  34, 255}, // was {138, 124,  80}
}
```

### Buttons

```go
	// PLAY in the reference's green. It is the one loud thing on the home
	// screen, which is the one screen where loud is the right answer.
	ColorPlay     = color.RGBA{129, 182, 76, 255}  // was { 92, 158, 118}
	ColorPlayHi   = color.RGBA{160, 205, 110, 255} // was {124, 184, 146}
	ColorPlayEdge = color.RGBA{84, 124, 48, 255}   // was { 58, 112,  82}

	// The back button stays deliberately quiet, now warm rather than blue-grey
	// so it belongs to the same room as everything else.
	ColorBack     = color.RGBA{70, 64, 58, 255}  // was {70, 76, 92, 255}
	ColorBackEdge = color.RGBA{46, 42, 38, 255}  // was {44, 48, 58, 255}
```

### Finish

```go
	ColorTextDim    = color.RGBA{206, 200, 190, 255}  // was {198, 200, 208} - warm off-white
	ColorTextShadow = rgba(20, 16, 10, 0.45)          // was rgba(12, 10, 34, 0.45)
	ColorShadow     = rgba(18, 14, 8, 0.40)           // was rgba(10,  8, 28, 0.40)
	ColorGloss      = rgba(255, 255, 255, 0.10)       // was 0.07 - a little life back on the cards
```

### Leave alone

`ColorStarGlow`, `ColorPickable`, `ColorPickableWash`, `ColorPicked`, `ColorPickedWash`,
`ColorPickedRing`, `ColorText`, `ColorTray`, `ColorTraySlot`, `ColorPanel`, `PieceFills`, and the
confetti palette in `internal/anim/confetti.go`.

The amber pick-up wash is worth a second look once this is on screen: amber on green is a strong,
pleasing combination, but amber and the gold star are now the only two warm things on the board. If the
picked-up square starts competing with the star, the fix is to desaturate the wash toward white rather
than to touch the star.

### The scarf — check, do not pre-emptively change

`ColorScarf` at `{46, 160, 67}` is now a green sitting on a green board. It should still read, because
it is drawn on the white body of the piece and has its own dark outline, and it is lighter and more
saturated than `ColorBoardD`. **Look at it before deciding.** If it muddies, deepen it to
`{32, 118, 52}` rather than changing its hue — the whole point is that it matches the scarf on the
wooden bishop in his hand.

---

## Not changing, on purpose

- **No decorative emoji come back.** The rocket, the sparkle and the title stars stay deleted.
- **No animation changes.** The Play button's breathing pulse stays at `0.012`; the background stays a
  two-stop near-flat gradient with nothing moving in it.
- **The hint pause stays at 2.5s.** None of this touches the attention work beyond the forced hue move.
- **The confetti stays fully saturated.** It is the payoff, it lasts a second, and it fires only on a win.

## Verifying

```
go build ./... && go test ./...
make shots
```

Then check, in this order:

1. **The background is clean.** No vertical band, no smudge down the right side. If it is still there,
   Step 1 is not finished — nothing else is worth judging until it is.
2. **The star is still the most salient thing on the board**, and the only gold thing.
3. **The blue move dots read instantly** on both cream and green squares. This is the change most at
   risk; if they are at all soft, deepen `ColorHintDot` rather than enlarging the dots.
4. **Light and dark squares are obviously different** — the scarf lesson depends on it.
5. **The six cards look tappable, not disabled.** If any one still reads as greyed out, it is too close
   in lightness to its neighbours, not too low in saturation.
6. **The scarf is still clearly green** against the green board.

`git diff internal/render/palette.go internal/render/shapes.go` is the whole review.

## If it is still not right

The likely remaining complaints and the lever for each:

- **Still too dull** → the cards and PLAY are already saturated, so the ground is the culprit: lighten
  `ColorBGTop`/`ColorBGBottom` by 10–15 per channel before touching anything else.
- **Too busy again** → do not desaturate. Reduce the *count*: the tray, the header text and the back
  button can all lose contrast without losing function.
- **The green board is wrong for him** → the same structure works with the blue-grey board you had
  (`ColorBoardD = {140, 148, 176}`, `ColorBoardL` white). Keep the warm charcoal ground and revert the
  move dots to mint, since green is free again in that version.
