# Calming the palette

## The problem

Everything on screen is currently competing at full saturation: a blueberry-to-plum background, a gold
board frame, six fully saturated piece cards, a bright green PLAY button, and decorative emoji on the
title and the button. Nothing is louder than anything else, so nothing stands out — and for a child we
have just spent effort teaching to *look at the board*, the board is not winning that fight.

Worth noticing specifically: **the board frame is gold and the star is gold.** The single brightest
warm shape on the play screen is a ring of chrome around the thing he is supposed to be looking for.

## The principle

One rule decides every value below:

> **The board is the only bright thing on screen. The star is the only gold thing.**

Three tiers:

| Tier | What | Treatment |
|---|---|---|
| 1 — the game | the star, the piece, the move dots, the scarf, confetti | **unchanged.** Fully saturated, and now unopposed |
| 2 — the furniture | board squares, frame, piece cards, buttons | muted: same hues, much less saturation |
| 3 — the ground | background, tray, back button, dim text | near-neutral, recedes |

**Do not touch Tier 1.** The mint move dots, the amber pick-up wash, the star gold and the scarf green
are all load-bearing — they are the attention work from `HINT-PAUSE-PLAN.md`. Muting them would undo
it. If a value is not in a table below, leave it exactly as it is.

The good news: almost all of this is one file, `internal/render/palette.go`.

---

## Step 1 — `internal/render/palette.go`

Mechanical find-and-replace. Left column is what is in the file today.

### Background (Tier 3) — drop the purple entirely

```go
	// Background: a near-flat neutral ink. It used to be a saturated blueberry
	// -> plum gradient, which competed with the board for attention and tinted
	// every white pixel on screen. A dark *neutral* ground makes the board, the
	// star and the confetti pop harder than a dark colourful one does.
	ColorBGTop    = color.RGBA{38, 41, 52, 255}   // was {34, 38, 92, 255}
	ColorBGBottom = color.RGBA{28, 30, 39, 255}   // was {96, 48, 124, 255}
```

The gradient now darkens downward, so the eye settles up onto the board rather than sliding off toward
a bright bottom edge.

### Board frame (Tier 2) — this is the most important single change

```go
	ColorFrame      = color.RGBA{150, 138, 120, 255}  // was {255, 206, 110, 255}
	ColorFrameInner = color.RGBA{108, 98, 84, 255}    // was {204, 142, 52, 255}
```

A warm taupe "wooden" rim instead of gold. After this, gold on the play screen means *star* and nothing
else.

### Board squares (Tier 2) — keep the contrast, drop the colour cast

```go
	ColorBoardL = color.RGBA{244, 242, 238, 255}  // was {255, 255, 255, 255}
	ColorBoardD = color.RGBA{150, 152, 156, 255}  // was {140, 148, 176, 255}
```

Bone white instead of pure white (less glare on a phone held close), and a neutral grey instead of a
blue-grey. **Keep these two clearly apart** — the light/dark distinction is what the bishop's scarf is
teaching, so do not close the gap between them in the name of calm.

### Piece cards (Tier 2) — same six hues, much quieter

The colour coding earns its place: he can ask for "the green one". Keep six distinguishable hues, just
stop shouting them.

```go
var PieceCardColors = []color.RGBA{
	{198, 124, 124, 255}, // pawn   - dusty rose   (was {255, 122, 122})
	{200, 158, 102, 255}, // knight - sand         (was {255, 174,  66})
	{118, 168, 132, 255}, // bishop - sage         (was { 86, 205, 138})
	{116, 150, 190, 255}, // rook   - slate blue   (was { 78, 166, 255})
	{158, 134, 186, 255}, // queen  - heather      (was {190, 124, 255})
	{198, 180, 120, 255}, // king   - wheat        (was {255, 206,  88})
}

var PieceCardEdges = []color.RGBA{
	{140,  82,  82, 255}, // was {196,  74,  74}
	{142, 108,  64, 255}, // was {196, 118,  26}
	{ 78, 116,  88, 255}, // was { 42, 150,  92}
	{ 76, 102, 132, 255}, // was { 36, 110, 190}
	{108,  90, 128, 255}, // was {132,  70, 200}
	{138, 124,  80, 255}, // was {198, 148,  38}
}
```

### Buttons (Tiers 2 and 3)

```go
	ColorPlay     = color.RGBA{92, 158, 118, 255}  // was { 58, 214, 130}
	ColorPlayHi   = color.RGBA{124, 184, 146, 255} // was {120, 240, 178}
	ColorPlayEdge = color.RGBA{58, 112, 82, 255}   // was { 28, 150,  88}

	ColorBack     = color.RGBA{70, 76, 92, 255}    // was { 74,  92, 156}
	ColorBackEdge = color.RGBA{44, 48, 58, 255}    // was { 44,  56, 112}
```

### Finish (Tier 3)

```go
	ColorGloss   = rgba(255, 255, 255, 0.07)       // was 0.13 - less candy sheen on the cards
	ColorTextDim = color.RGBA{198, 200, 208, 255}  // was {214, 208, 240} - drops the purple cast
```

### Leave alone, explicitly

`ColorHint`, `ColorHintDot`, `ColorHintRing`, `ColorPickable`, `ColorPickableWash`, `ColorPicked`,
`ColorPickedWash`, `ColorPickedRing`, `ColorStarGlow`, `ColorScarf`, `ColorScarfEdge`, `ColorText`,
`ColorShadow`, `ColorTextShadow`, `ColorTray`, `ColorTraySlot`, `ColorPanel`, `PieceFills`.

---

## Step 2 — `internal/game/home.go`: delete the decoration

Four lines of pure ornament. Removing them is the biggest calm-per-keystroke win in the whole plan, and
it is not a colour change at all.

Delete the two stars flanking the title (around line 131):

```go
	starSize := r.title.H * 0.42
	render.DrawEmoji(dst, render.StarEmoji, tcx-r.title.W*0.40, tcy, starSize, 0, 1)
	render.DrawEmoji(dst, render.StarEmoji, tcx+r.title.W*0.40, tcy, starSize, 0, 1)
```

Delete the rocket and sparkle riding on the PLAY button (around line 143):

```go
	render.DrawEmoji(dst, "1f680", px+pw*0.155, py+ph*0.52, ph*0.46, -0.5, 1)
	render.DrawEmoji(dst, "1f31f", px+pw*0.845, py+ph*0.52, ph*0.46, 0, 1)
```

A rocket on the button means nothing here — it is just movement near a word he cannot read yet. And
the star, again, should mean one thing only: the thing to reach on the board.

Soften the glow behind the button while you are in there (same function):

```go
	render.DrawGlow(dst, r.play.X+r.play.W/2, r.play.Y+r.play.H/2, r.play.W*0.55, render.Alpha(render.ColorPlayHi, 0.15))
```

was `0.30`. Leave the button's breathing pulse alone — at `0.012` it is already almost subliminal, and
it is what says "press me".

`starSize` becomes unused once the title stars go; delete the variable or the compiler will complain.

---

## Deliberately not changed

- **The confetti** (`internal/anim/confetti.go:31`) stays fully saturated. It lasts about a second, it
  only fires when he has won, and it is the payoff the whole loop is built around — this is the one
  moment the screen *should* shout. If it turns out to be too much, cut the particle **count** in
  `collectStar` rather than the colours; fewer bright pieces reads as calmer than the same number of
  dull ones.
- **Every Tier 1 colour**, per the list above.
- **The sticker tray and its emoji.** They are small, static, and at the bottom edge.

---

## Verifying

```
go build ./... && go test ./...
make shots
```

Then open `shots/home` and `shots/rook` and check three things:

1. On the play screen, the **star is the brightest thing** and the only gold thing.
2. The **six cards are still tellable apart** at a glance. If two now look alike, push those two hues
   apart rather than turning the saturation back up on all six.
3. The **light and dark squares are still obviously different** — the bishop scarf lesson depends on it.

The whole change is two files, so `git diff internal/render/palette.go internal/game/home.go` is the
entire review, and reverting is `git checkout` on those two paths.

## If it is still too much

Next lever, in order:

1. Drop the card colours entirely — all six the same muted slate, told apart by the piece art and the
   word underneath. Colour coding is a nice-to-have; the artwork is the real label.
2. Flatten the background to a single colour (set `ColorBGBottom` equal to `ColorBGTop`) so there is no
   gradient at all.
3. Reduce the confetti count.

---

# Addendum: more reward stickers

This pulls in the opposite direction from Step 2 above, and that is deliberate — the distinction is
worth stating plainly:

- **Chrome decoration gets removed.** The rocket on the PLAY button and the stars around the title
  decorate something he is not looking at, and the star in particular steals meaning from the one on
  the board.
- **Reward stickers get added.** These are the payoff. Variety here is the whole point: the surprise of
  a sticker he has not seen before is what makes the tray worth filling.

Calming the screen and widening the prize pool are the same goal — put the interest where the reward
is, not in the furniture.

## How adding one works

Drop an SVG into `internal/render/assets/emoji/` named `<codepoint>.svg` and that is the entire change.
`init()` in `internal/render/emoji.go` walks the embedded directory, sorts the names, and everything not
listed in `uiEmoji` automatically joins the reward pool. **No Go code needs editing.**

Rules for the files:

- Source: **Twemoji** (https://github.com/jdecked/twemoji), `assets/svg/<codepoint>.svg`. Same set as
  the 93 already there, so the new ones will not look out of place. It is CC-BY 4.0 and already
  credited in `assets/emoji/ATTRIBUTION.md` — no attribution change needed.
- Name the file in **lowercase hex with no variation selector**: `2614.svg`, not `2614-fe0f.svg`. This
  matches the existing `2600`, `2764`, `26bd`.
- Do not add anything to `uiEmoji` — that map is the exclusion list for icons the interface itself
  uses, and it should stay at five entries.

## What counts as "easily recognisable"

The stickers render at roughly 28–30px in the tray on a phone. That is the constraint driving all of
this. A candidate earns its place only if:

1. It is **one object, centred, filling the frame** — no scenes, no pairs of things.
2. It has a **strong silhouette with few internal details**. Detail vanishes at tray size.
3. It is a thing he can **name in one word**, ideally a word he already says.
4. It is **not confusable with another sticker in the set** at that size. Check new ones against what
   is already in `assets/emoji/` before adding — the fruit especially is already crowded.

Rejects, for the same reasons: abstract symbols, anything with text or numerals, faces whose meaning
depends on reading the expression, tools and gadgets he has no word for, and any second member of a
look-alike pair.

## The gaps in the current 93

The set is already deep on **animals** (about 35) and **fruit and sweets** (about 25). It is thin on
exactly the categories toddlers name earliest. Suggested additions, all Twemoji, all passing the four
rules above:

**Vehicles** — currently only rocket, train, car, plane

```
1f68c  bus
1f692  fire engine
1f69c  tractor
1f6b2  bicycle
1f681  helicopter
26f5   sailboat
```

**Weather and nature** — currently only sun and rainbow

```
2601   cloud
2614   umbrella with rain
2744   snowflake
1f333  tree
1f33b  sunflower
```

**Toys and things around the house** — currently almost nothing

```
1f9f8  teddy bear
1fa81  kite
1f941  drum
1f514  bell
1f511  key
1f45f  shoe
```

**Animals worth filling in**

```
1f980  crab
1f42c  dolphin
1f988  shark
```

That is 20, taking the pool from 88 to 108. Add fewer if any of them look muddy at tray size — the
rules matter more than the count.

## Two things to know before adding a lot

- **Sticker indices shift.** `emojiNames` is sorted, so inserting files renumbers everything after
  them. This is safe *today* because the sticker list lives only in memory (`Game.stickers`) and is
  gone when the app closes — there is no save file anywhere in the project. If stickers are ever
  persisted between sessions, this stops being free, and the stored value has to become the codepoint
  string rather than the index.
- **The prize cycle gets longer.** `internal/reward` deals every sticker once before any repeats, so
  going from 88 to 108 means a favourite comes back around a fifth less often. That is usually good —
  more novelty — but if he has a sticker he lights up for, a bigger pool means seeing it less. Worth
  watching for once it is in.

## Verifying

```
go test ./internal/render/...
make shots
```

`TestEmojiAssetsRasterize`-style coverage will catch a malformed or empty SVG. Then look at
`shots/home` — the tray shows four stickers — and ideally view the new files at about 30px before
committing to them. Anything you cannot name instantly at that size does not belong in the pool.
