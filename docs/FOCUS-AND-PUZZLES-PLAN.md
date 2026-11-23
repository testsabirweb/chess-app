# Focus, and checkmate puzzles

## Context

He has outgrown the current games. He knows how every piece moves (the knight too, give or take),
can collect several targets, take pawns, and dodge a guard. Two things followed:

1. **He is losing interest and focus.** That is the expected shape, not a failure of the app or of
   him. Studies of children's learning games find the same curve: novelty drives play at first and
   fades within weeks, and rewards like stickers stretch the *early* sessions but do not change total
   play over time. Stickers handed out for every win can even dilute a child's own sense of
   accomplishment. What holds attention for longer is **challenge at the right level** (a common
   coaching rule of thumb is roughly four out of five solved) and **short sessions that end while he
   still wants more** (about 5–10 minutes of sustained attention is typical around age five).
2. **He is ready for the next concept: check and checkmate.** Coaches teach check first, then
   mate in one, often as a short daily set of 5–10 puzzles, and move to mate in two once mate in one
   comes easily. Patterns with the enemy king on an edge, finished by a queen or a rook, are the
   usual starting point.

So the plan has two halves. **Part A** is about focus: some of it is changes to the app, and some
is what we do away from it. **Part B** is the new puzzle mode.

Nothing in this plan undoes an earlier decision without saying so. Where an idea runs into one (for
example, "no saving between launches"), it is listed under **Open questions** rather than assumed.

## Status

**Part B is implemented**: the chess engine has check, legal moves, checkmate and stalemate; the
puzzle bank is built offline by `make puzzles` (`cmd/genpuzzles`) and embedded from
`internal/puzzles/bank.txt`; the puzzle game is on the home screen (crown button) with all four
tiers, including mate in two. The puzzle tier moves up after three clean rounds and down after three
rough ones, in the session only (see open question 1).

**Part A:** A4 (Pawn Wars) is implemented too: the handshake button opens it. Still open are
away-from-the-app advice (A0, no code), levels for the older games (A1), the today trail (A2) and the
reward changes (A3).

The board stays 5x5. It is the board he knows, the squares stay large for small fingers, and every
bank puzzle fits on it; `layout.Compute` already takes the board size, so an 8x8 mode is possible
later without rework, but there is no need for one now.

---

## Part A — Helping him focus

### A0. Away from the app (no code; for the grown-ups)

These matter more than any feature, and cost nothing:

- **Short and finite.** About 5–10 minutes, then stop *while he is still enjoying it*. Ending on a
  win makes the next session easy to start; ending on a meltdown does the opposite.
- **Play next to him, not just hand it over.** Sit with him, react to his moves, ask "where can the
  rook go?" before the dots appear. At this age playing together does more for attention than any
  reward screen.
- **Let him teach you.** "Show me how the bishop moves." Explaining is harder than doing, and he
  gets to be the expert.
- **Take it off the screen.** Set the same position up on the wooden set at home (the bishop with
  the scarf is already the bridge). A puzzle solved with real pieces sticks.
- **Praise how, not who.** "You looked at all the squares before you moved" rather than "you're so
  smart". That is the habit the hint pause and the "Perfect!" reward are trying to build.
- **Let him choose the game**, and **rest it** for a few days when interest dips. Novelty comes
  back after a break; pushing through a slump rarely helps.
- **Don't use it as a reward or a punishment.** That turns it into currency rather than a game.

### A1. Right-sized challenge (adaptive difficulty)

The biggest in-app cause of boredom is that the games no longer stretch him. Each game gets a small
**level** that goes up after a run of clean solves and down after a struggle. It never goes up so
far that he stops winning.

| Game | What a level changes |
|---|---|
| Find the star | Star 1 → 2 → 3 moves away (`MinMoves`); hint pause 2.5s → 4s → dots only on request |
| Which piece? | 2 → 3 → 4 candidates |
| Collect | 2 → 3 targets; longer tours allowed (`treasureMaxTour`, `catchMaxTour`) |
| Stay safe | Warning shown always → only with the dots → not at all; detour every round |
| Puzzles (Part B) | Check → mate in 1 (easy) → mate in 1 (with decoy checks) → mate in 2 |

Rules (one shared helper, e.g. `internal/game/level.go`):

- Up one level after **3 perfect solves in a row**. Down one after **2 rounds in a row with three or
  more wrong taps**, or a stuck round.
- Each game has a small, fixed number of levels (3–4). The level is shown to the grown-up only, as
  small dots under the header, not to him as a score.
- Uses the "Perfect!" signal that already exists (`roundKit.win(..., perfect)`) and the wrong-tap
  path that already exists (`oops`, `refuse`, `WhichScene.wrong`).

### A2. A short "today" trail

A finite session with a visible end is the in-app version of "short and finite":

- A new home button (a path or flag icon) starts **today's trail**: about 6 rounds mixing the games,
  for example star → which → collect → puzzle → safe → puzzle. The mix means no single game wears
  thin, and it keeps the repetition each skill needs.
- A row of 6 dots across the top fills in as he goes. At the end comes one big celebration (a
  trophy and the "every five stickers" party, made bigger), then a gentle **"All done for today!"**
  screen that sends him home. It does not lock him out: the grown-up can still start anything from
  the home screen. It simply gives a natural place to stop.
- Built on the existing scenes. `roundKit` gets a hook that runs after a round's reward lands, so a
  trail can move to its next step instead of dealing another puzzle of the same game.

### A3. Rewards tied to mastery, not to every win

The research is fairly clear that a sticker for every win loses its pull and can crowd out his own
pride in solving. Suggested changes, smallest first:

- **Keep** the sticker for each solve (he likes them), but make the trail's end and a level-up the
  big moments: a level-up gets its own short "You're ready for harder ones!" celebration.
- **Spoken or shown "why"**: after a "Perfect!" in a puzzle, replay the winning move once, slowly,
  with the mating lines drawn. Rewarding success with *information* holds attention better than
  more confetti.
- A **sticker album** with themed pages to complete (animals, food, ...) would turn collecting into a
  goal. It needs stickers kept between launches, which we decided against, so it is an open question
  rather than part of the plan.

### A4. Pawn Wars (already planned)

Phase 7 of `LEARNING-MODES-PLAN.md`: two players on one phone. Playing *with* a grown-up is the
strongest focus tool we have, and it is the first real game against a person. Bring it forward,
ahead of A2/A3 if needed.

---

## Part B — Checkmate puzzles

### B0. Decisions

- **Board: stay on 5×5.** It is the board he knows, the squares stay big, and the layout code is
  already built for it. Real mates fit: king and queen, or king and rook, against a king on the edge.
  An 8×8 board can come later, since `layout.Compute` already takes `cols, rows` (today it is called
  with `5, 5` in `game.go`).
- **He always plays white, from the bottom, and always moves first.**
- **Puzzles are generated offline, not on the phone.** A command, `cmd/genpuzzles`, searches positions
  on the Mac and writes a small checked-in file (`internal/puzzles/bank.txt`, a few kilobytes, about
  300 puzzles), which the app embeds. This keeps quality reviewable, costs nothing at run time, and
  matters for old phones: mate-in-2 search is too slow and too heavy to run there on every round.
- **Still no losing.** A wrong move is undone with a short explanation, never scored as a defeat.

### B1. Chess rules the app does not have yet

`internal/chess` generates *pseudo-legal* moves only: nothing knows about check. Puzzles need:

- `(*Board).KingSquare(c Color) (Square, bool)`
- `(*Board).Attacked(sq Square, by Color) bool`, built on the existing `Attacks`
- `(*Board).InCheck(c Color) bool`
- `(*Board).LegalMoves(from Square) []Move`: pseudo-legal moves that do not leave your own king in
  check (the king cannot step into check or onto a defended piece)
- `(*Board).Checkmate(c Color) bool` and `Stalemate(c Color) bool`

Existing games keep using `Moves`/`MoveTargets` unchanged, since they have no kings, so nothing they do
changes. Tests go next to `chess_test.go`/`attacks_test.go`: known mates (back-rank rook, queen
supported by king), known non-mates (the king can take an undefended queen), stalemate, and
"the king may not step into check".

### B2. The puzzle bank

**Format**, one puzzle per line, readable by eye:

```
# tier | board, rank 4 down to rank 0, "." empty, KQRBNP white, kqrbnp black | solution
m1 | ..k.. / ..... / ..K.. / ..... / Q.... | a0-a4
```

(Exact notation is up to the implementer: file letters a–e, ranks 0–4, as in the app.)

**Tiers**, which become the puzzle levels in A1:

1. **Check!** White to move. Find *any* move that attacks the black king. It teaches what check
   looks like. Positions have several pieces and only one or two checking moves.
2. **Mate in 1, easy.** Exactly one mating move, and it is the only check on the board.
3. **Mate in 1, with decoys.** Exactly one mating move, plus at least one check that is *not* mate,
   which is the classic beginner trap.
4. **Mate in 2.** White moves, the app plays black's reply, and white mates. No mate in 1 exists.
   Every black reply must allow a mate (that is what "forced" means), and black must have at least one
   legal reply (not stalemate).

**Generator rules** (`cmd/genpuzzles`):

- Material sets, from simple to richer: K+Q v K, K+R v K, K+R+R v K, K+Q v K+pawns, K+R v K+pawns
  (black pawns in front of their king make back-rank mates), K+B+R v K. No knights in tiers 1–2.
- Reject unless: kings not adjacent; black not in check with white to move; position is not already
  mate or stalemate; the black king is preferably on an edge for tiers 1–3.
- Deduplicate mirror images (left–right flip) and identical positions.
- Order the bank so each tier gets gradually richer, then shuffle within small bands at run time.
- Verify every line on load in a test: tier rules hold, and the stated solution actually mates.

**Repeats:** the app deals from a shuffled bag per tier and never repeats a puzzle until the bag is
empty, the same idea as `reward.Picker` for stickers.

### B3. Playing a puzzle

- The board shows white and black pieces. The black king gets a soft glow, so "attack the king" is
  the obvious thing to look for. Tapping a white piece picks it up, with the same hint pause and dots
  as the star game. Black pieces cannot be picked up.
- **Right move:** the piece lands, and the lines it attacks through the king's square flash. The
  black king topples over, then the usual confetti and sticker follow. Right first time is "Perfect!".
- **A check that is not mate** (tier 3 trap): the move plays, the black king steps to its escape
  square with a little hop, the escape square glows green ("the king got away!"), then everything
  slides back. He *sees* why it was not mate, which is the whole lesson.
- **Not even a check:** the piece slides back with the existing wobble, and the black king pulses.
- **Help after misses:** after 2 wrong tries, the piece that mates gets a gentle glow. After 3, its
  destination square glows too. He always finishes; it is just not "Perfect!".
- **Mate in 2:** after his first (correct) move there is a short pause, then the app plays black's
  reply with the normal hop animation (a random reply among black's legal ones, so the same puzzle can
  play out differently). Then it is his turn to mate. If his first move does not lead to a forced mate,
  it is undone like a tier 3 trap: he is shown the reply that escapes, then the position resets.
- **Checking the king** is said with a short "Check!" text pop over the king (no voice clips; that
  feature was removed at the parent's request).

**Scene:** a new `PuzzleScene` (`internal/game/puzzle.go`) using `roundKit` for the header, tray,
rewards and back button, and reusing the piece hop/arc animation from `PlayScene` (factor it out
rather than copy it, as `WhichScene` should have done). Home: a new mode button (crown or target
icon), which deals its own pieces (`needsPiece: false`).

### B4. Rules of the existing games are untouched

Puzzles are the only place kings and check exist. The other games keep their no-kings boards and
pseudo-legal moves, so none of their tests or behaviour change.

---

## Order of work

| Step | What | Why this order |
|---|---|---|
| 1 | B1 check/mate rules + tests | Everything in Part B rests on it |
| 2 | B2 generator + bank (tiers 1–3) + bank tests | Content before UI; reviewable as text |
| 3 | B3 puzzle scene, tiers 1–3 | The new challenge he is ready for |
| 4 | A1 levels, starting with puzzles and the star game | Keeps every game at the right difficulty |
| 5 | A4 Pawn Wars | Playing together |
| 6 | B2/B3 tier 4 (mate in 2) | Once mate in 1 comes easily |
| 7 | A2 today's trail | Ties the games into short, finite sessions |
| 8 | A3 reward tweaks | Last; cheap once the rest exists |

Each step is its own commit and leaves the app shippable. After each: `make test`, `go vet ./...`,
`make shots` plus a new shot script per new scene, then install on the phone and watch him play it.

## Constraints to keep

- **Old phones:** no search at run time beyond what the current generators already do; the puzzle
  bank is precomputed and embedded (a few KB). Any new cache must be bounded, like the emoji cache
  (`emojiCacheBudget`).
- **No losing** outside Pawn Wars; wrong moves are undone and explained, never scored.
- **No voice clips** (removed at the parent's request).
- **No progress saved between launches** unless the open question below changes that.

## Open questions for the parent

1. **Save progress between launches?** Without it, levels (A1) and puzzle tiers reset to easy every
   time the app opens, which works against the "right-sized challenge" idea, and a sticker album (A3)
   is impossible. A middle ground is to save only the *levels*, not the stickers.
2. **How old is he now, and does he know the knight yet?** That decides whether knights appear in
   puzzle tiers 3–4.
3. **Today's trail length:** 6 rounds is a guess at 5–10 minutes. Adjust after watching him.
4. **Pawn Wars before puzzles?** If the main worry is focus rather than new content, playing together
   may be the better first step.
