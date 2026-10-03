# Learning modes: what comes after "where does this piece go?"

## Status

Phases 0, 2 and 3 are implemented (shared round kit, mode row, "Which piece?"). Phases 4–7 are
still to do.

**Phase 1 (voice clips) was built and then removed** at the parent's request; the app is back to
sound effects only. Ignore every mention of voice clips, `Say`, `ClipID`, `oops` rate limiting and
`make voice` in the phases below.

Other differences from the text below:
- The star game has no button in the mode row: the piece cards already play it. `rowModes()` in
  `internal/game/mode.go` lists the modes that do get one, and the row only appears when there is
  at least one. A mode is added to `activeModes` in the same change that makes it playable.
- Buttons for games that need a piece toggle (tap again to go back to the star game); games that
  deal their own pieces start straight away.
- In "Which piece?" the child moves the piece: tapping the right piece picks it up and shows its
  moves, and tapping the star makes it hop.

## Context

The star game has done its job: our son can now say where every piece goes except the knight, which
is deliberately being left for later (most curricula put knight games at age 5+). The next skills, in
the order the Steps Method (the most widely used children's chess curriculum) teaches Step 1, are
**attacking and capturing**, then **defending / noticing danger**, then check. Coaches also agree on the
design rule this app already follows: **one mini-game trains one skill**.

This plan adds voice clips and five new modes, one phase at a time. Each phase is a separate commit and
leaves the app shippable.

| Phase | What | Skill it trains |
|---|---|---|
| 0 | Extract the shared round chrome out of `PlayScene` | (refactor, no behaviour change) |
| 1 | Voice clips in a parent's voice | Naming pieces; spoken feedback for a pre-reader |
| 2 | Mode row on the home screen | (plumbing for 3–7) |
| 3 | **Which piece?** — several pieces, one can reach the star | Recognition in reverse; checks he really knows the moves |
| 4 | **Collect the stars** — 2–3 stars in one puzzle | Planning a route |
| 5 | **Catch the pawns** — capture 1–3 black pawns | Capturing; lines opening as pieces disappear |
| 6 | **Stay safe** — an enemy piece guards some squares | Seeing the *other side's* moves — the step from moving pieces to playing chess |
| 7 | **Pawn Wars** — two players on one phone | His first real game against a person |

Research this is based on: Steps Method Step 1 (stappenmethode.nl), ChessWorld's mini-game list (Pawn
Wars, Rook Road, Treasure Knight, Safe Square Hunt), Magnus' Kingdom of Chess (enemy attacks drawn as
arrows; 4-year-olds pick up "safe square" that way), Story Time Chess (ages 3+), and Callaghan 2021,
*Mobile app features that scaffold pre-school learning: verbal feedback and leveling designs* (spoken
feedback and gradual difficulty beat text and random order).

**Explicitly not in scope:** keeping stickers across launches (the parent does not want it), knight-
specific games, an adventure map / unlockable pieces, story characters per piece.

---

## Ground rules for whoever implements this

- **One phase per commit**, in order. Phases 3–7 depend on 0 and 2; phase 1 is independent. Commit
  messages follow the repo's style: one plain sentence, e.g. `Add the "Which piece?" mode.`
- After every phase: `make test` passes (it runs `go test -race ./...`), `go vet ./...` is clean, and
  `make shots` still renders. Look at the PNGs in `shots/` — this is the only UI check we have without a
  phone. Add a shot for each new mode (see phase 2).
- **Match the surrounding code.** Comments in this repo explain *why*, in full sentences, often from the
  child's point of view (see `play.go`). No comment that only restates the code. Constants with a
  sentence explaining the number. Rendering allocates nothing per frame (see `PlayScene.hints`).
- **No fail states** in the single-player modes (3–6). A wrong tap gives the existing soft wobble
  (`PlayScene.oops`) and nothing harsher. Every puzzle is solvable; if a wandering piece makes a target
  unreachable, the target hops (as `relocateStar` does today). Pawn Wars (7) is the one mode where
  someone loses, because it is played with a grown-up.
- **The knight is out of the new piece pools.** Use one shared list:
  ```go
  // beginnerPieces are the pieces the new modes deal on their own. The knight
  // is left out on purpose: its L is the one move he has not got yet, and a
  // mode that keeps springing it on him would turn a game he can win into one
  // he guesses at. Picking the knight card explicitly still works.
  var beginnerPieces = []chess.PieceType{chess.Pawn, chess.Bishop, chess.Rook, chess.Queen, chess.King}
  ```
  Generators must still *work* for the knight (the knight card stays on the home screen), so test them
  with it.
- **Do not change how the existing star game plays.** Phase 0 is a pure refactor; phase 4 generalises
  one target to several but the star mode keeps exactly one.
- Keep `README.md` current: add each mode to "How it plays" and any new tuning constants to the
  "Knobs worth knowing" table.

---

## Phase 0 — Extract the shared round chrome

**Why:** `internal/game/play.go` (832 lines) owns things every board mode needs: the back button (tap
to go home, hold 2s to install an update), the header text, the sticker tray in the footer, the sticker
that pops out and flies into the tray, confetti, the every-fifth-sticker milestone overlay, and the fade
between puzzles. "Which piece?" and Pawn Wars are different scenes and would otherwise copy ~250 lines.

**Do:** create `internal/game/kit.go` with a `roundKit` struct that `PlayScene` embeds or holds. Move
into it, with no behaviour change:

- fields: `game`, `confetti`, `sprites`, `reward`/`rewardActive`, `perfect`, `advanceT`/`advanceSwp`,
  the four `milestone*` fields, `backHoldT`/`backInstallF`;
- the `stateCelebrating`, `stateMilestone` and `stateAdvancing` states. `PlayScene` keeps only
  `stateIdle` and `stateMoving`, and only while the kit says a round is in play;
- `updateReward`, `finishReward`, `drawFooter`, `drawMilestone`, the advance fade in `Draw`, the back
  button handling at the top of `PlayScene.Update`, and the back button/name part of `drawHeader`;
- the footer geometry helpers (`traySlots`, `trayRect`, `trayEmojiPos`, `backButtonRect`) and
  `milestoneMessages` can stay where they are or move to `kit.go` — whichever reads better.

Suggested shape (adjust freely, but keep it this small):

```go
func (k *roundKit) update(ctx *Context, m layout.Metrics) (leave, deal bool)
// leave: the back button was tapped; the caller switches to the home scene.
// deal: the advance fade just reached its midpoint; the caller deals the next
// puzzle now, while the screen is covered.

func (k *roundKit) busy() bool                  // celebrating, milestone or advancing: ignore board taps
func (k *roundKit) win(ctx *Context, m layout.Metrics, cx, cy float64, perfect bool)
func (k *roundKit) restart()                    // advance to a new puzzle without a reward (the stuck-pawn case in relocateStar)
func (k *roundKit) drawHeader(dst *ebiten.Image, m layout.Metrics, name string)
func (k *roundKit) drawOverlays(dst *ebiten.Image, ctx *Context, m layout.Metrics) // confetti, flying sticker, footer, milestone, fade
```

`win` does what `collectStar` does after the sounds: the confetti burst (60 when perfect, else 30),
the gold glow, `"Perfect!"` in the header, and the sticker fly. Sounds stay with the caller, because each
mode chooses its own.

Taps during the milestone overlay dismiss it (today: `case stateMilestone: p.milestoneT = 0`). Keep that
in the kit.

**Check:** `make test`; `make shots` before and after the refactor, then compare the PNGs side by side.
Puzzles are random so the boards will differ, but the chrome, reward pop, fly and milestone frames must
look the same.

---

## Phase 1 — Voice clips (text-to-speech)

**Why:** he cannot read. The app currently speaks only in synthesised beeps (`internal/sfx`), and the
2021 study above found spoken feedback is what helps preschoolers most. The clips are generated with the
Mac's built-in text-to-speech (`say`), so there is nothing personal in them and they are committed to
the repo. A better (AI or recorded) voice can replace them later by overwriting the files.

### Clips

All optional at runtime. **A missing file is silent and is not an error**, and the synthesised sounds
keep playing either way. The text each one says lives in `internal/sfx/voice/clips.txt`, one
`name|text` per line, so the whole set can be regenerated.

| File | Says | Played when |
|---|---|---|
| `pawn` `knight` `bishop` `rook` `queen` `king` | "Pawn." … | A piece card is tapped on the home screen; in "Which piece?" when a *wrong* piece is tapped |
| `yay` | "Yay! You did it!" | A puzzle is solved (not perfect) |
| `perfect` | "Perfect!" | A puzzle is solved in the fewest moves (replaces `yay` that time) |
| `oops` | "Oops. Try again!" | A wrong tap — **at most once every 6 seconds**, or it nags |
| `which` | "Who can reach the star?" | Each "Which piece?" round starts |
| `careful` | "Careful! Not there." | A guarded square is tapped in "Stay safe" |
| `white-turn` `black-turn` | "White's turn." / "Black's turn." | Pawn Wars, the turn changes |
| `white-wins` `black-wins` | "White wins!" / "Black wins!" | Pawn Wars, someone wins |
| `great-job` | "Great job!" | The every-fifth-sticker milestone |

Add the clips for modes that don't exist yet now anyway; the trigger lands with the mode.

### Code

- New `internal/sfx/clips.go`. **Name collision:** `sfx.Voice` already means a synth voice
  (`synth.go`), so call these clips: `type ClipID uint8`, constants `ClipPawn … ClipGreatJob`, a
  `clipFiles = map[ClipID]string`, and `func (b *Bank) Say(id ClipID)`.
- Embed: `//go:embed voice/*.wav` on an `embed.FS`, directory `internal/sfx/voice/`. Keep `clips.txt` in
  that directory so `go:embed` always has something to match.
- WAV, not Ogg: ebiten decodes it with `audio/wav` (`wav.DecodeF32`), and macOS can produce it with no
  extra tools. Decode once in `NewBank`, read it fully, and build the player with
  `ctx.NewPlayerF32FromBytes`, as `Bank.load` does for the synth players. A clip that fails to decode is
  skipped, not a panic.
- **One clip at a time:** `Say` pauses whatever clip is still playing before starting the new one, so
  two sentences never talk over each other. Synth effects are separate and keep layering as today.
- Rate limiting for `oops` lives in the game, not the bank: a `lastOops float64` against `ctx.T`.
- Test (`internal/sfx/clips_test.go`): every `ClipID` has a file name; names are unique; every named file
  is embedded and decodes. Follow `synth_test.go`: do not open a real audio device in tests.

### Generating the clips

```make
# Regenerate the spoken clips from internal/sfx/voice/clips.txt with the Mac's
# built-in text-to-speech. Swap any file for a better recording by hand.
voice:
	@cd internal/sfx/voice && while IFS='|' read -r name text; do \
	  [ -n "$$name" ] || continue; \
	  say -v Samantha -r 150 -o "$$name.aiff" "$$text" && \
	  afconvert -f WAVE -d LEI16@24000 -c 1 "$$name.aiff" "$$name.wav" && rm "$$name.aiff"; \
	  echo "$$name.wav"; done < clips.txt
```

Add `voice` to `.PHONY`, run it once and commit the `.wav` files (about 1 MB in all). No CI change is
needed.

---

## Phase 2 — Mode row on the home screen

**Why:** modes are a grown-up's choice, made deliberately; the game should not level up on its own.

- `internal/game/mode.go`:
  ```go
  type Mode uint8

  const (
  	ModeStar Mode = iota // the original game
  	ModeWhich
  	ModeTreasure
  	ModeCatch
  	ModeSafe
  	ModePawnWars
  )
  ```
  with a `modeInfo` table: name shown on screen (`"Find the star"`, `"Which piece?"`,
  `"Collect the stars"`, `"Catch the pawns"`, `"Stay safe"`, `"Pawn Wars"`), icon emoji, and
  `needsPiece bool` (false for Which and Pawn Wars).
- `Game` gets `mode Mode` (default `ModeStar`). It lives for the session only, like the stickers.
- **Icons** are Twemoji, like every other emoji here: `2b50` ⭐ (exists), `2753` ❓, `1f48e` 💎,
  `1f36a` 🍪 (exists), `1f6e1` 🛡️, `1f91d` 🤝. Download the missing SVGs from
  `https://cdn.jsdelivr.net/gh/jdecked/twemoji@latest/assets/svg/<code>.svg` into
  `internal/render/assets/emoji/`, and **add all six to `uiEmoji` in `internal/render/emoji.go`** so a
  mode icon is never handed out as a sticker (the comment there explains why). Check
  `assets/emoji/ATTRIBUTION.md` still covers them.
- **Layout** (`internal/game/home.go`): add `modes [6]layout.Rect` to `homeRects`.
  - Portrait: a new band between `play` and `label`, height `min(max(s.H*0.07, m.MinTap), homeDp(m, 64))`,
    included in `scaleHomeBands` (which needs a `modeH` parameter) so it shrinks with everything else.
  - Landscape: the same band inside the right-hand panel, between `play` and `label`.
  - Six equal round tiles in one row; each at least `m.MinTap` wide where the screen allows it.
  - Extend `TestHomeLayoutFitsSafe` in `home_test.go`: mode tiles inside the safe area, not overlapping
    each other or any other band, on every device in `homeDevices`.
- **Behaviour:**
  - Tapping a piece mode (Star, Treasure, Catch, Safe) *selects* it: button sound, a ring around the
    selected tile (`render.FillRingSoft`), and the `"Pick a piece"` label becomes `"<mode name> — pick a piece"`
    (use `FitTextSize`). The piece cards then start that mode with that piece.
  - Tapping Which or Pawn Wars *starts* it straight away, through the same press-and-hold animation the
    cards use (`HomeScene.arm`).
  - **PLAY** starts the selected mode with the rook, as it does today for the star game.
  - Tapping a piece card says the piece's name (phase 1).
- `NewPlayScene(g, pt)` becomes `NewPlayScene(g, pt, mode)`; the back button returns home with the mode
  still selected.
- `cmd/shot`: add `-mode star|which|treasure|catch|safe|pawnwars` and `debug.go` helpers to start a
  scene in that mode (`NewInMode`). `-scene play` keeps meaning star mode. Add one `make shots` line per
  mode as it lands (portrait Edge 50 Neo metrics, like the existing `rook` line), plus one landscape
  home shot showing the mode row.

---

## Phase 3 — Which piece?

**The game:** 2–3 white pieces of different types stand on the board with a star. Exactly one of them
can reach the star in one move. He taps that piece; it hops to the star; sticker.

**Wrong pick:** that piece's move dots appear **at once** (no hint pause — this is the teaching moment),
the voice says its name, the piece wobbles, and after ~1.5s the dots fade and the piece is drawn
faded (about 40% alpha) and can't be tapped again. With three pieces he can't miss more than twice, so
he always gets there. **Perfect** = right first time.

**Generation** — new `internal/challenge/which.go`:

```go
// Which is one "who can reach the star?" round.
type Which struct {
	Board  *chess.Board   // every candidate piece on it
	Pieces []chess.Square // where the candidates stand
	Answer chess.Square   // the one candidate that reaches Target in one move
	Target chess.Square
}

func NewWhich(rng *rand.Rand, pool []chess.PieceType, n int) Which
```

- Pick `n` distinct types from `pool` (`beginnerPieces`), place them on distinct random squares. White
  pawns only on ranks 1–3 (a pawn on rank 0 breaks the double-push rule — see `Generator.startOK`; a pawn
  on rank 4 can't move).
- Target: an empty square that is in `MoveTargets` of **exactly one** candidate, computed on the board
  with all candidates on it (they block each other, which is real chess).
- Retry until valid; bound the retries and fall back to `n-1` pieces rather than loop forever.
- Avoid the same answer *type* three rounds running (keep the last two answer types; reject a third).
- `n = 2` for the first 3 rounds of the scene, `3` after.
- Tests (`which_test.go`): for many seeds and every `n`, exactly one candidate reaches the target; target
  is empty; types are distinct; no knight when the pool has none; deterministic for a seed.

**Scene** — `internal/game/which.go`, `WhichScene` using the phase-0 kit. Reuse the hop animation from
`PlayScene` (`startMove`/the moving case of `Update`; factor a small helper if needed rather than
copying it). The star is drawn with `render.DrawStar`. A faded piece needs alpha: add a variant of
`render.DrawPiece` that takes an alpha (via `ColorScale`), without changing the existing signature's
callers. Voice: `which` at each round start.

---

## Phase 4 — Collect the stars

**The game:** the star game with 2 (first 3 rounds) then 3 stars, collected in any order. Each star
pops with a small burst (15 confetti, `SndPop`) and disappears; the last one gives the sticker.
**Perfect** = the shortest possible tour.

**Generalise `PlayScene` from one target to several** — this is the engine for phases 4–6 too:

- `target chess.Square` → `targets []chess.Square`. Star mode always has exactly one, so it plays
  exactly as before.
- `render.Hint.Target` is true for any square in `targets`. The star magnet in `handleTap` loops over
  the targets that are one move away.
- `land`: if `p.at` is a target, remove it; if none are left, `collectStar` as today; else the small
  pop and play on.
- `relocateStar` → relocate whichever remaining target has become unreachable (to a reachable empty
  square that isn't already a target), and recompute `optimal = steps + tour length`.

**Challenge package:**

```go
// Puzzle is one round of a single-piece mode.
type Puzzle struct {
	Board   *chess.Board
	From    chess.Square
	Piece   chess.Piece
	Targets []chess.Square // stars to land on, or black pieces to capture
	Optimal int            // fewest moves to clear every target, in any order
}

// Tour reports the fewest moves for the piece on `from` to visit every target,
// in any order, or -1 if that can't be done within maxMoves in total. Landing
// on a target that holds a piece captures it, so the board is replayed leg by
// leg: lines open up as pieces disappear.
func Tour(b *chess.Board, from chess.Square, targets []chess.Square, maxMoves int) int
```

`Tour` tries every order (at most 3! = 6), using `MovesTo` per leg on a cloned board with the piece
moved (and the captured piece removed). Give the existing `Generator` a way to produce a `Puzzle`
(Targets = `[]chess.Square{Target}`) so `PlayScene` has one input type; behind a small interface,
`type puzzleSource interface{ Next() challenge.Puzzle }`, with one source per mode.

`NewTreasure(spec, rng)`: random start (respect `startOK`), 2–3 distinct empty target squares each
reachable within `maxJourney`, and a total `Tour` of at most `treasureMaxTour = 5` moves (a constant with
a comment: longer tours are a slog at this age). Use the same history idea as `Generator` so the same
layout doesn't repeat back to back.

Tests: tours are solvable and `Optimal` equals a brute-force check on small boards; no target under a
piece; deterministic for a seed; works for every piece type including the knight.

---

## Phase 5 — Catch the pawns

**The game:** like phase 4, but the targets are **black pawns** standing on the board — real pieces,
not emoji, so it carries over to a real board. Landing on one captures it (`SndPop`, the existing
capture sound). 2 pawns for the first 3 rounds, then 3. The black pawns never move.

Why it's worth having on top of phase 4: captured pieces open lines (a rook can now slide past where a
pawn stood), sliders are blocked by the pawns they haven't caught yet, and his own pawn can *only*
capture diagonally — the rule most children get wrong.

**Generation** — `NewCatch(spec, rng)` in the challenge package: place the player's piece, place
black pawns on random empty squares, accept if `Tour` (which already handles captures) is solvable
within `catchMaxTour = 5`. For a white pawn player this will naturally put black pawns diagonally ahead.
Black pawns are not placed on rank 0 or 4 (they'd look odd to a parent who knows chess).

**Play:** `PlayScene` with `targets` = the black pawns' squares; they are drawn as pieces (the existing
loop over `p.board.Occupied()`), not stars. If one becomes unreachable it hops away like a star does
(`SndHop`), to an empty square the piece can still reach.

Tests as for phase 4, plus: every target square holds a black pawn; capturing all of them in the
`Optimal` order is legal move by move.

---

## Phase 6 — Stay safe

**The game:** the star game with one black **guard** piece that doesn't move. Squares it attacks are
tinted soft red. He must reach the star without stopping on a red square. Moving through the guard's
line is not the issue — only where the piece *stops*.

**Tapping a red square:** the move is refused — the guard does a small lunge toward that square, all
red squares flash brighter for ~1.2s, `SndOops`, and the `careful` clip. Nothing else. The guard can't
be captured in this mode (tapping it is an ordinary wrong tap).

**Fading the help** (the Magnus' Kingdom idea, matched to our hint pause): for the first
`safeTeachRounds = 5` rounds of the scene the red tint is always visible. After that it shows only when
the move dots do — after the hint pause — so he starts spotting danger before it is drawn for him.
Legal-move dots still appear on red squares; the red tint under them is what he must notice. Hiding
them would teach nothing.

**Chess package** — `func (b *Board) Attacks(from Square) []Square`: the squares the piece on `from`
attacks. Same as `MoveTargets` except for pawns, which attack their two forward diagonals whether or
not anything stands there (and never the square in front). Tests next to the existing ones in
`chess_test.go`.

**Which squares are red:** the guard's `Attacks` on the board **with the player's piece removed**.
That is the correct chess idea (a piece moving along a line doesn't block that line for itself) and it
makes the red set fixed for the whole puzzle, so what is shown is exactly what is enforced. Compute it
once per puzzle.

**Challenge package:**

- `ReachAvoiding(b, from, maxMoves, avoid func(chess.Square) bool) []Step` — `Reach`, but never stops
  on an avoided square. Make `Reach` call it with `nil`.
- `NewSafe(spec, rng)`: guard type from `beginnerPieces` minus the player's own type when that's
  confusing (a rook guarding a rook puzzle is fine; just avoid an identical-looking pair by preferring a
  different type). Accept a layout when: start and star aren't red; the star is empty and isn't the
  guard; the star is reachable avoiding red squares and the guard square in 2–3 moves; and **danger
  matters** — at least one of the piece's first moves lands on red. Half the time also require a
  detour: the safe route is longer than the unrestricted one. Bounded retries with fallbacks, as
  `Generator.Next` does.
- The `Puzzle` struct gains `Guard chess.Square` and `Hot []chess.Square`.

**Play:** `PlayScene` refuses hot destinations in `handleTap`; `CanReach` in `land` and the relocation
in `relocateStar` use `ReachAvoiding`. Drawing: a new `render.DrawDangerTint(dst, m, squares, strength)`
drawn under the move hints, colour `ColorDanger` in `palette.go` — a warm, soft red that fits the
cream-and-green theme, not an alarm red.

Tests: generated puzzles are solvable avoiding red; start and star never red; danger always matters;
pawn guards make the diagonals red, not the square in front.

---

## Phase 7 — Pawn Wars

**The game** (every curriculum we found starts real games here): he plays white at the bottom, a
grown-up plays black from across the table, one phone. Three pawns each — white on files 1–3 of rank
1, black on files 1–3 of rank 3 (`pawnWarsFiles = []int{1, 2, 3}`, a constant with a comment: five a
side jams a 5×5 board solid). Normal pawn moves and captures; the double push works from rank 1/3 when
the way is clear. No en passant, no promotion piece choice.

**Turns:** white first. Tap your own pawn to pick it up (dots after `game.hintDelay`, same as play
mode), tap a dot to move. Tapping the other side's pawn when it's not their turn is an ordinary wobble.
Show whose turn it is with a soft glow along that player's edge of the board and the header text
`"White's turn"` / `"Black's turn"`, plus the turn clips.

**Ending:**

- A pawn reaches the far rank, or the other side has no pawns left → that side wins.
- The side to move has no legal move → **everyone wins** (confetti, sticker). Don't explain stalemate to
  a 4-year-old.
- **White wins:** the full kit celebration and a sticker; `white-wins` clip.
- **Black wins:** confetti over black's half, header `"Black wins!"`, `black-wins` clip, **no
  sticker**, then `"Tap to play again"` and a fresh board. This is the one place in the app with a
  loser; that is intentional, because the opponent is a person who can choose to let him win.

**Code:** `internal/game/pawnwars.go`, `PawnWarsScene`, using the kit. All move rules already exist in
`chess.Board.Moves` (pawns move up for white and down for black). Put the win/draw rules in a pure
function so they're testable without the scene:

```go
// pawnWarsResult reports who has won after `mover` has just moved, or ok=false
// while the game is still going.
func pawnWarsResult(b *chess.Board, mover chess.Color) (winner chess.Color, draw, ok bool)
```

Tests: reaching the far rank wins for either side; losing the last pawn loses; a blocked position is a
draw; the start position is not over.

Out of scope for now: rotating black's pieces to face the person across the table.

---

## Verification (every phase)

1. `make test` — all packages green, with `-race`.
2. `go vet ./...`.
3. `make shots`, then look at every PNG in `shots/`, including the new per-mode shots and the landscape
   and small-phone ones. Nothing overlaps, nothing falls outside the safe area, text fits.
4. `make run` on the Mac and play the mode by hand: solve one puzzle perfectly, one by wandering, tap
   wrong squares on purpose, and leave with the back button.
5. Phase 1 only: run once with **no** `.ogg` files (silent, no errors) and once with a couple of test
   clips (they play, never overlap each other, `oops` doesn't repeat within 6s).
6. When a phase is done, `make apk && make install` and hand the phone to the actual user.
