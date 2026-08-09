# Toddler Chess — Go + Ebitengine, Android-first

## Context

**Why:** Recreate and improve `github.com/testsabirweb/kids_chess` (a vanilla-JS web toy) as a native
Android game for a toddler. The repo `github.com/testsabirweb/chess-app` is **empty** — 12-byte README,
one commit. This is greenfield.

**What the original does** (verified by reading the repo): single HTML page, 5×5 board, piece parked at
the centre, Unicode glyph pieces, "Find the ⭐" tap game, emoji sticker rewards, 5-sticker milestone
modal, WebAudio-synthesised sounds, DOM confetti. **Zero binary assets.** That last property is the best
idea in it and we keep it.

**What we're building:** a mobile-first portrait Ebitengine game. Home screen with a huge Play button
plus a six-piece picker; a responsive 5×5 board that eats most of the screen width; tap the star, watch
the piece hop there, get confetti and a sticker; repeat forever. No losing, no timers, no pressure.

**Decisions already made with the user:**
- **Zero binary assets.** Pieces, stickers, confetti and sounds are all generated in code.
- **Chunky toy piece art** — fat rounded vector shapes, thick dark outlines, glossy highlight. Not
  Staunton silhouettes.
- **All six pieces playable in milestone 1**, reached via a piece picker on the home screen.
- **Development moves to the MacBook.** This VM is headless (no `DISPLAY`, no GL/X11/ALSA dev headers,
  no `javac`) so the game can never be seen running here.

---

## 0. Handoff to the MacBook — do this first

This plan file lives on the Linux VM and will not travel. Before switching machines:

1. Write this document to `docs/PLAN.md` in the repo.
2. `git add docs/PLAN.md && git commit && git push origin main`.
3. On the MacBook: `git clone git@github.com:testsabirweb/chess-app.git && cd chess-app`, start Claude
   Code, and point it at `docs/PLAN.md`.

Everything below is written for macOS. The Go code is 100% portable; only SDK paths differ.

---

## 1. Verified facts this plan depends on

Checked against Ebitengine v2.9.9 source, the pinned `ebitengine/gomobile` fork, and Google's SDK
manifest. Do not re-derive these.

| Fact | Consequence |
|---|---|
| ebiten **v2.9.9** is latest stable; `go.mod` says `go 1.24.0` | Go 1.25.x is fine |
| `vector.DrawFilledCircle`/`DrawFilledRect` **deprecated in 2.9** | use `FillCircle` / `FillRect` / `FillPath` / `StrokePath` |
| `audio.NewPlayerFromBytes` **deprecated since v2.2** | use `NewPlayerF32FromBytes` — **float32 LE, stereo, no error return** |
| `vector.FillPath` has **no colour arg** | tint via `DrawPathOptions.ColorScale.ScaleWithColor` |
| `EbitenView.onLayout` calls `Ebitenmobileview.layout(px / deviceScale())`, and `deviceScale()` **is** `Monitor().DeviceScaleFactor()` | multiplying back in `LayoutF` reconstructs exact physical pixels → offscreen→view scale is 1.0 and touch coords need no conversion |
| `DeviceScaleFactor()` panics "no current JVM" if called from `init()` on Android (ebiten #597) | only call it inside `LayoutF` |
| `ebiten.TouchPosition(id)` returns **(0,0)** for an already-released touch | on release use `inpututil.TouchPositionInPreviousTick(id)` |
| `gofont` has **zero** U+2654–265F coverage | Unicode chess glyphs are tofu on *every* platform — vector pieces are mandatory, not a preference |
| `gobold.TTF` is a `[]byte` slice | `text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))` works directly |
| `ebitenmobile` builds its own gomobile+gobind into a temp dir per invocation | never `go install gomobile`; pin `ebitenmobile@v2.9.9` |
| `ebitenmobile` flags are exactly: `-o -gcflags -ldflags -target -bundleid -iosversion -tags -androidapi -a -i -n -v -x -trimpath -work -javapkg -prefix -classpath -bootclasspath` | there are **no** `ebitenmobile:*` source directives |
| `-javapkg` is **mandatory** for android; `EbitenView` lands in `<javapkg>.<goPkgName>` | Go dir must be one lowercase-ASCII package (`mobile`) |
| AAR manifest hardcodes `minSdkVersion = -androidapi` | Gradle `minSdk` must **equal or exceed** it or manifest merge fails |
| gomobile picks the **highest purely-numeric** `platforms/android-N` | must install `platforms;android-36`; `android-36.1` is silently skipped |
| `EbitenSurfaceView` calls `setEGLContextClientVersion(3)` | GLES 3.0 required, **no Vulkan**; declare `glEsVersion 0x00030000` |
| oto builds Oboe with `-DOBOE_ENABLE_AAUDIO=0` | OpenSL ES only, ~30–80 ms latency (fine here), **no permissions needed** |
| gomobile passes **no** page-size linker flags | pass `CGO_LDFLAGS=-Wl,-z,max-page-size=16384` yourself for Play's 16 KB rule |
| `ebitenmobile` only implements `bind` | `gomobile build` is **not** an option for Ebitengine — confirmed, not a style preference |
| Go 1.25 ships `wasm_exec.js` at `$(go env GOROOT)/lib/wasm/wasm_exec.js` | moved from `misc/wasm` |

---

## 2. Architecture

Module `github.com/testsabirweb/chess-app`. The first five `internal/` packages import **no Ebitengine**
and carry every test.

```
main.go                     desktop dev entry (go run .)
mobile/mobile.go            untagged: Dummy()  — package `mobile`
mobile/setgame_mobile.go    //go:build android || ios : init(){ mobile.SetGame(game.New()) }
web/                        index.html + wasm_exec.js (WASM test harness)
android/                    Gradle project (§7)
internal/
  chess/      PURE  square.go piece.go board.go moves.go              + tests
  challenge/  PURE  challenge.go generator.go                         + tests
  layout/     PURE  rect.go metrics.go insets.go                      + tests
  anim/       PURE  ease.go tween.go confetti.go                      + tests
  sfx/        synth.go (PURE, tested) + bank.go (only file importing ebiten)
  input/      pointer.go        unified mouse+touch
  render/     palette.go font.go paths.go sprites.go board.go widgets.go
  game/       game.go scene.go home.go play.go
```

**The build-tag split matters.** `ebiten/v2/mobile` has an `impl_notmobile.go` whose `setGame` *panics*.
Keeping the `init()` behind `//go:build android || ios` means `go build/vet/test ./...` on macOS never
trips it; keeping `Dummy()` untagged means the package is never empty on desktop. `gobind` loads with
`GOOS=android` so it sees both files.

**CI guardrail** — this is what keeps the decoupling honest rather than aspirational:

```sh
! go list -deps ./internal/chess ./internal/challenge ./internal/layout ./internal/anim \
  | grep -q hajimehoshi
```

---

## 3. Pure packages

### `internal/chess`

Rank 0 = bottom row. Board dimensions are parameters so 8×8 later is free.

```go
type Square struct{ File File; Rank Rank }   // comparable value type
func Sq(f, r int) Square
type PieceType uint8 // NoPiece, Pawn, Knight, Bishop, Rook, Queen, King
type Piece struct{ Type PieceType; Color Color }

func NewBoard(width, height int) *Board
func (b *Board) Contains(s Square) bool
func (b *Board) At(s Square) Piece      // zero Piece if empty/off-board
func (b *Board) Set(s Square, p Piece)  // panics off-board
func (b *Board) Clone() *Board
func (b *Board) Occupied() []Square     // sorted, deterministic

type Move struct{ From, To Square; Capture bool }

// Deterministic order: sorted by (Rank, File) so table tests compare literally.
func (b *Board) Moves(dst []Move, from Square) []Move
func (b *Board) MoveTargets(from Square) []Square
func (b *Board) CanMove(from, to Square) bool
```

Implementation is three direction tables (`orthoDirs`, `diagDirs`, `knightDirs`) plus a slide flag —
~60 lines. Sliders stop before friendly pieces and include enemy squares with `Capture: true`. Pawn is
the only special case: forward push, double push from the home rank, diagonal capture **only** onto an
occupied enemy square.

**Deliberately pseudo-legal — no check detection.** There is no king-safety concept in a toddler game.
If it's ever needed, add `LegalMoves` that filters `Moves`; nothing above changes.

The two decisions that make the whole suite literal-comparison instead of set-comparison: `Occupied()`
and `Moves()` both return sorted output.

### `internal/challenge`

```go
type Challenge struct {
    Board     *chess.Board
    Piece     chess.Piece
    From      chess.Square
    Target    chess.Square    // the star; always ∈ Solutions
    Solutions []chess.Square  // every legal destination — powers the "right piece move,
                              // wrong square" encouragement path, free because we have it
}

type Spec struct {
    BoardWidth, BoardHeight int
    Pieces                  []chess.PieceType
    Color                   chess.Color
    MinDistance, MaxDistance int  // Chebyshev; 0 = unconstrained
    Decoys                  bool  // see pawn note below
}

const HistorySize = 8
func NewGenerator(spec Spec, rng *rand.Rand) *Generator  // rand.New(rand.NewPCG(1,2)) in tests
func (g *Generator) Next() Challenge                      // always solvable, always terminates
```

**Anti-repetition via a relaxation ladder, not rejection sampling.** Rejection sampling with a retry cap
either fails or biases. Instead `Next` enumerates the full candidate set (25 squares × ≤8 targets on a
5×5 — microseconds) then drops constraints weakest-first until non-empty:

1. Pick a `PieceType`, excluding the previous one when the pool has >1.
2. Enumerate all `(from, to)` pairs for that piece.
3. Filter by Min/MaxDistance.
4. Drop `(piece, from, to)` triples in the `HistorySize` ring buffer.
5. Drop pairs where `from == lastFrom` or `to == lastTarget`.
6. If empty, undo filters in reverse (5, then 4, then 3) and retry. Step 2 is non-empty for every real
   piece, so the ladder always bottoms out on a valid set.
7. Pick uniformly; push the triple onto the ring.

No spin loops, no failure mode, no `error` in the signature, and each rung is independently assertable.

**Pawn needs `Decoys`.** A lone pawn on an empty 5×5 has only 1–2 targets, so the star is always directly
ahead — degenerate and boring. When `Decoys` is on and the piece is a pawn, sometimes place an enemy
piece on a forward diagonal so a capture target exists. This is the one generator special case; keep it
contained and tested.

### `internal/layout`

Design units are **dp**; `dp(v) = v * scale`. Minimum tap target 48dp (Material).

```go
type Rect struct{ X, Y, W, H float64 }
type Insets struct{ Top, Bottom, Left, Right float64 } // logical px
type Metrics struct {
    W, H, Scale float64
    Safe, Header, Board, Footer Rect   // Board always square
    Cell, MinTap, TitleSize, BodySize float64
    Cols, Rows int
    Portrait bool
}
func Compute(w, h, scale float64, in Insets, cols, rows int) Metrics
```

**The board is the anchor and claims width first; chrome is the remainder.** That ordering is what makes
it total — no iteration, no case where the stack overflows:

1. `Safe` = platform insets **floored** by our own margins (`max(in.Top, dp(20))`, sides `dp(14)`). The
   floors mean the layout is correct even if insets are never delivered, and on desktop/WASM.
2. `side = min(Safe.W, Safe.H - hdrMin - ftrMin - 2*gap)` with `hdrMin=dp(56)`, `ftrMin=dp(72)`.
3. Tap floor: if `side < MinTap*cols`, grow it to `min(MinTap*cols, min(Safe.W, Safe.H))`.
4. `remain = Safe.H - side - 2*gap`; if negative, give the space back to the board. **This step
   guarantees fit.**
5. Split `remain` 40/60 header/footer, capped at `dp(180)`/`dp(240)` so chrome isn't absurd on tall screens.
6. Centre the whole stack vertically in `Safe`.
7. Type scale keyed off **`Cell`**, not the screen — `TitleSize = clamp(Cell*0.45, dp(20), dp(56))` —
   so text and board always look proportional to each other.

| Device | logical px | scale | Cell | vs 48dp floor |
|---|---|---|---|---|
| Edge 50 Neo | 1080×2400 | 2.75 | ~210 | 132 ✓ (1.6×) |
| 1440p flagship | 1440×3120 | 3.5 | ~280 | 168 ✓ |
| budget phone | 720×1600 | 2.0 | ~138 | 96 ✓ |
| dev window | 432×960 | 1.0 | ~80 | 48 ✓ |
| landscape | 800×400 | 1.0 | 48 | 48 ✓ (step 3 fires) |

```go
func (m Metrics) CellRect(f, r int) Rect   // Y flipped: rank 0 is the bottom
func (m Metrics) HitCell(x, y float64) (file, rank int, ok bool)
func (m Metrics) HitStar(x, y float64, star chess.Square) bool
func (m Metrics) TapOK() bool              // dev-build assertion
```

`HitStar` is **deliberately forgiving**: accept a tap within a generous radius of the star's centre even
if it lands on a neighbouring cell, provided that neighbour isn't itself a solution. Toddlers miss by
10–15 mm routinely — this small thing is the difference between fun and frustrating.

### `internal/anim`

**Delta-time, not frame counting**, but `anim` never reads the clock — `Game.Update` computes
`ctx.DT = 1/ebiten.TPS()` (falling back to 1/60 when TPS ≤ 0 under `SyncWithFPS`) and passes it down.
Durations read as seconds; tests just loop `Update(1.0/60)`.

**Do not interpolate in `Draw`.** `Update` is fixed-rate, `Draw` is vsync-rate; keeping `Draw` read-only
keeps the render layer trivially correct.

```go
type Ease func(float64) float64  // Linear, EaseOutCubic, EaseInOutCubic, EaseOutBack, EaseOutElastic
type Tween struct{ Duration, Delay float64; Ease Ease }
type Pulse struct{ Period float64 }  // free-running oscillator, no state to reset
type Particle struct{ X, Y, VX, VY, Rot, VRot, Life, MaxLife, Size float64; Color color.RGBA; Shape uint8 }
type Confetti struct{ ... }
func (c *Confetti) Burst(rng *rand.Rand, x, y float64, n int, unit float64)  // unit = m.Cell
func (c *Confetti) ForEach(f func(Particle))                                 // keeps it ebiten-free
```

Piece slide ≈0.45 s on `EaseInOutCubic`, plus an arc hop (`- sin(p*π) * Cell*0.22`) and a brief
squash-and-stretch on touchdown. Three lines that carry most of the delight.

Confetti: `VY += gravity*dt`, drag `v *= pow(0.98, dt*60)`, fade `alpha = min(1, Life/0.4)`, dead
particles removed by swap-with-last so there's no allocation churn.

**Scale bursts by `m.Cell`** so the effect looks identical on a 720p budget phone and a 1440p flagship.

### `internal/sfx`

`Render(voices []Voice, sampleRate int) []byte` is pure and tested: float32 LE, **stereo** (write each
sample twice), 48000 Hz, headerless.

**Enforce a ≥5 ms attack and ≥30 ms release inside `Render`.** A sine starting or ending on a non-zero
sample clicks audibly on phone speakers, and at this age a click reads as "I did something wrong."

Sounds: `SndButton` (soft blip), `SndLand` (220 Hz with a fast pitch drop), `SndCheer` (C5-E5-G5-C6
triangle arpeggio + shimmer), `SndOops` (**gentle** descending G4→E4, low amplitude — never a buzzer),
`SndNear` (warm chime for "right piece move, wrong square").

`bank.go` holds one reusable player per sound (`Rewind()` then `Play()` on retrigger — players aren't
GC'd until they finish). `audio.NewContext` goes in `game.New()`, **never `init()`** — it's a per-process
singleton.

---

## 4. Rendering

### Pieces — chunky toy style

Build each shape **once in a 0..1 unit box** at package init as a `*vector.Path`, then transform per frame:

```go
var add vector.AddPathOptions
add.GeoM.Scale(r.W, r.H); add.GeoM.Translate(r.X, r.Y)
p.AddPath(unitPaths[t], &add)

var dop vector.DrawPathOptions
dop.AntiAlias = true
dop.ColorScale.ScaleWithColor(fill)          // ALWAYS set explicitly — FillPath has no colour arg
vector.FillPath(dst, &p, &vector.FillOptions{FillRule: vector.FillRuleNonZero}, &dop)
// then StrokePath with a dark outline, Width = r.W*0.045, LineJoinRound, LineCapRound
```

Shape recipes: rook = trapezoid base + body + 3 merlons (~12 `LineTo`); pawn = circle head + collar +
flared base; bishop = pear via two `CubicTo` + ball + mitre slit; queen = body + 5-point crown with ball
tips; king = body + fat cross; knight = blobby head via 4 `CubicTo` (the only fiddly one — keep it
abstract). Every piece gets a thick dark outline plus a light highlight blob for the toy look.

Define shapes as `[]pathCmd` **data tables** so `paths.go` is unit-testable without a GPU: assert each
path is closed and its bounds fall inside the unit box.

Available `Path` methods: `MoveTo LineTo QuadTo CubicTo Arc ArcTo Close Reset Bounds AddPath AddStroke`.
`AppendVerticesAndIndicesFor*` are deprecated.

### ⚠ The one real perf trap

`vector.FillPath` in 2.9 renders via a custom stencil-buffer shader. Excellent for two pieces on screen;
**wrong for 120 confetti particles** — that's 120 stencil passes per frame on a mid-range Adreno.

Fix, and it does **not** violate the zero-assets rule (that means no files in the repo, not no textures
in VRAM): `render/sprites.go` rasterises procedural shapes into `*ebiten.Image` once and caches them,
keyed by shape + pixel size, regenerated only when `Metrics.Cell` changes. Confetti becomes cheap
`DrawImage` calls with a rotating `GeoM`. Keep pieces as live `FillPath` initially for crispness; move
them into the cache if profiling on the Edge 50 Neo says so.

### Text

```go
boldSrc, _ := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
// cache faces by rounded pixel size — constructing per frame allocates and redoes shaping setup
op := &text.DrawOptions{}
op.GeoM.Translate(cx, cy)
op.ColorScale.ScaleWithColor(clr)
op.PrimaryAlign, op.SecondaryAlign = text.AlignCenter, text.AlignCenter  // translate point IS the centre
text.Draw(dst, s, face, op)
```

Fit-to-button via `text.Measure`, stepping the cached size down 2–3 times. Don't set `Script`/`Language`
— the Latin default is right and `Script` is deprecated in 2.9.

### Stickers

No colour-emoji font exists here, so rewards are vector shapes: star, heart, flower, sun, smiley, moon,
etc., in bright palette colours. Drawn from the same path-table mechanism as pieces.

---

## 5. Game layer

### Input — `internal/input`

```go
func (t *Tracker) Update()  // once per tick, BEFORE scenes; inpututil is Update-only
```

Merges `inpututil.IsMouseButtonJustPressed/Released` + `CursorPosition` with
`AppendJustPressedTouchIDs`/`AppendJustReleasedTouchIDs` + `TouchPosition`. Reuse slice buffers to avoid
per-frame allocation.

**Gotcha:** on release use `inpututil.TouchPositionInPreviousTick(id)` — `TouchPosition` silently returns
(0,0) for a released touch, which reads as "child tapped the top-left corner" and is maddening to debug.
`TouchPosition` already returns *logical* coordinates, so no conversion is needed.

### Scenes

One interface, one current pointer, one fade curtain. **No scene stack, no manager type.**

```go
type Scene interface {
    Update(ctx *Context) error
    Draw(dst *ebiten.Image, ctx *Context)
}
type Context struct {
    M layout.Metrics; Pointer *input.Tracker; SFX *sfx.Bank; Rand *rand.Rand; DT float64
    next Scene
}
func (c *Context) Switch(s Scene)
```

`Draw` paints the scene then a full-screen rect at the fade tween's alpha. Home↔Play costs ~6 lines.

**HomeScene:** big playful title, one **huge** Play button (starts Rook), and a 3×2 grid of six large
piece buttons below it — each showing the chunky piece art plus its name. Sticker count shown small in
a corner. Nothing else.

**PlayScene:** header shows the current piece large plus its name; board in the middle; footer holds the
earned-sticker row and a big Home button. Piece switching happens from Home only — that keeps the play
screen uncluttered.

### The challenge loop

Every transition is driven by a tween completing, so there is exactly one clock:

```go
const (
    stateIdle        playState = iota // star pulsing, waiting for a tap
    stateMoving                       // piece tweening From → Target
    stateCelebrating                  // confetti alive, cheer playing
    stateAdvancing                    // old challenge out, new in
)
```

`star` (a free-running `Pulse`) and `wobble` (per-square wrong-tap tweens) deliberately live **outside**
the state machine. A pulsing star and a wobbling square are decorations; forcing them into states is what
turns a 4-state FSM into a 12-state one.

**Wrong taps never change state**, and there are two tiers — this is where the design earns its keep:

- Tapped square **is** in `cur.Solutions` (a real rook move, wrong destination) → `SndNear`, gentle
  wobble, and briefly draw the piece's move rays. A genuine teaching moment, free because `Solutions`
  is already in the struct.
- Tapped square is unreachable → `SndOops` (quiet, low), small wobble, nothing else.

**Use press, not release, for taps** — more immediate, and toddlers drag their fingers before lifting.

### `LayoutF`

```go
func (g *Game) Layout(w, h int) (int, int) { return w, h }  // never called; LayoutFer wins

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
    s := ebiten.Monitor().DeviceScaleFactor()   // NEVER from init() — panics pre-JVM on Android
    if s <= 0 { s = 1 }
    g.scale = s
    return outsideWidth * s, outsideHeight * s
}
```

Do **not** implement `FinalScreenDrawer` — at 1:1 the default blit is already correct and implementing it
only adds a chance to get it wrong.

---

## 6. WASM — the fast feedback loop

Worth building early: it typechecks every line of rendering code with zero system deps, and it's the
quickest way to get the game onto the actual Motorola without the NDK.

```
GOOS=js GOARCH=wasm go build -o web/game.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
cd web && python3 -m http.server 8080
```

Then open `http://<mac-lan-ip>:8080` in Chrome on the Edge 50 Neo. Real touch, real screen, real portrait
aspect. `web/index.html` should set `viewport-fit=cover`, `user-scalable=no`, and a black background.

Note: insets aren't delivered in the browser, so the `dp(20)`/`dp(14)` floors in `Compute` are what keep
it looking right there.

---

## 7. Android

### macOS prerequisites

```bash
xcode-select --install
brew install go openjdk@17
brew install --cask android-commandlinetools

export JAVA_HOME="$(brew --prefix openjdk@17)/libexec/openjdk.jdk/Contents/Home"
export ANDROID_HOME="$(brew --prefix)/share/android-commandlinetools"
export ANDROID_SDK_ROOT="$ANDROID_HOME"
export ANDROID_NDK_HOME="$ANDROID_HOME/ndk/28.2.13676358"
export PATH="$ANDROID_HOME/platform-tools:$JAVA_HOME/bin:$(go env GOPATH)/bin:$PATH"

yes | sdkmanager --licenses
sdkmanager "platform-tools" "platforms;android-36" "build-tools;36.0.0" "ndk;28.2.13676358"
go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@v2.9.9
```

Why exactly these: `platforms;android-36` must be **numeric** (gomobile `strconv.Atoi`s the suffix, so
`android-36.1` is skipped and you get a confusing "failed to find android SDK platform"). NDK
**r28.2.13676358** is the known-good pick — it's AGP 9.3's default, it's the release where 16 KB
alignment became default, and its `meta/platforms.json` min API is 21 so `-androidapi 24` passes
`checkNDKRoot`. Avoid r29/r30: gomobile depends on the per-API clang wrapper scripts
(`aarch64-linux-android24-clang`) and their survival there is unverified.

**JDK 17 specifically** — not 8 (can't read android-36's class files), not 21/25 (gomobile hardcodes
`javac -source 1.8 -target 1.8`, and `-source 8` is being phased out in newer JDKs). If you must use a
newer JDK, smoke-test `javac -source 8 -target 8 -version` first.

**Apple Silicon check:** NDK r28 ships `toolchains/llvm/prebuilt/darwin-x86_64` (universal binaries; there
is no `darwin-arm64` dir). gomobile builds that path from `runtime.GOARCH`, so arm64 must map to
`x86_64`. Ebitengine's fork handles this — **verify it on the first `make bind`**, it's the single most
likely macOS-specific failure.

### Bind

```bash
CGO_LDFLAGS="-Wl,-z,max-page-size=16384 -Wl,-z,common-page-size=16384" \
ebitenmobile bind -target android/arm64 -androidapi 24 \
  -javapkg com.testsabirweb.chessapp -o android/app/libs/chessapp.aar -v ./mobile
```

Single ABI during development: the ~146 Oboe C++ translation units dominate bind time, and omitting the
arch builds all four. Release uses `-target android/arm64,android/arm`.

Each bind creates a temp module, runs `go get`/`go mod tidy` (**network required**) and wipes
`$(go env GOPATH)/pkg/gomobile`. First run takes several minutes.

### Gradle project

`android/` with AGP **8.13.2** + Gradle **8.14.5** (not AGP 9.x — the `implementation(files("*.aar"))`
pattern is verified on 8.x and unverified on 9). Bootstrap the wrapper by curling `gradlew` and
`gradle-wrapper.jar` from `raw.githubusercontent.com/gradle/gradle/v8.14.5` — no Gradle install needed.

Key settings:
- `namespace`/`applicationId` = `com.testsabirweb.chessapp`, `compileSdk`/`targetSdk` 36,
  **`minSdk = 24` (must equal `-androidapi`)**, Java 17 compat.
- `packaging { jniLibs { useLegacyPackaging = false } }` — uncompressed + page-aligned, **required for
  16 KB**. Don't "optimise" it back.
- `dependencies { implementation(files("libs/chessapp.aar")); implementation("androidx.appcompat:appcompat:1.7.1"); implementation("androidx.core:core:1.16.0") }`

Manifest: `<uses-feature android:glEsVersion="0x00030000" android:required="true"/>`, **no permissions
at all**, `launchMode="singleTop"` (per the 2025-10-04 doc update — *not* `singleInstance`),
`screenOrientation="portrait"`, and a deliberately broad `configChanges` list
(`orientation|screenSize|screenLayout|smallestScreenSize|keyboardHidden|uiMode|density|fontScale|locale|layoutDirection`).
That last one is a real gotcha-avoider: every Activity recreation rebuilds the `GLSurfaceView`, and
`EbitenSurfaceView.onContextLost()` calls `Runtime.getRuntime().exit(0)` — a hard kill.

`MainActivity` (Java): `WindowCompat.setDecorFitsSystemWindows(window, false)` → `setContentView` →
`Seq.setContext(getApplicationContext())` → immersive-sticky system bars → an
`OnApplyWindowInsetsListener` on the **FrameLayout container** (not the EbitenView) that applies
`systemBars() | displayCutout()` as padding. Padding the container is exactly how you tell Go the real
playable size, because `EbitenView.onLayout` forwards its own `(right-left, bottom-top)`. Wire
`suspendGame()`/`resumeGame()` in `onPause`/`onResume`.

`EbitenViewWithErrorHandling extends EbitenView` — the import path is the load-bearing bit:
`com.testsabirweb.chessapp.mobile.EbitenView` = `<javapkg>` + `.` + Go package name.

Edge-to-edge is **mandatory** on Android 15 for targetSdk 35+ and not opt-out-able on Android 16 for
targetSdk 36; `windowOptOutEdgeToEdgeEnforcement` is deprecated and ignored. Handle insets, don't fight them.

Optional refinement once the basics work: export `SetInsets(top, bottom, left, right float64)` from the
`mobile` package and call it from the listener instead of padding, so the game can paint edge-to-edge
behind the bars. gomobile passes `float64`→`double` fine, but ebitenmobile's passthrough of arbitrary
exported funcs is **unverified** — keep the padding approach as the fallback.

### Makefile targets

`run` (`go run .`), `test` (`go test -race ./...`), `wasm`, `serve`, `tools`, `bind`, `apk`
(`cd android && ./gradlew assembleDebug`), `install` (`adb install -r` + `am start`), `logs`
(`adb logcat -s Go:V GoLog:V ChessApp:V AndroidRuntime:E`), `verify-16k`, `clean`.

`verify-16k` unzips the APK and runs `llvm-objdump -p` on each `.so` — every `LOAD` segment must say
`align 2**14`. `2**12` means the page-size flags didn't take. **Do not take 16 KB on faith**: gomobile
passes no page-size flags, and whether bare r28 clang defaults to 16 KB is unverified. The Edge 50 Neo
runs 4 KB pages so a broken build still works on-device — only the Play upload catches it.

`.gitignore`: `android/app/libs/*.aar`, `*-sources.jar`, `android/.gradle/`, `android/build/`,
`android/app/build/`, `android/local.properties`, `web/game.wasm`, `web/wasm_exec.js`.

---

## 8. Tests

**`internal/chess`** — table-driven: per-piece targets from centre and corner of an empty 5×5 against
literal sets (rook c3→8, bishop c3→8, queen c3→16, knight a1→2, king c3→8 / a1→3, white pawn c2→{c3,c4});
blocked sliders and pawn capture rules; **deterministic order** (call twice, assert byte-identical —
guards against map iteration silently breaking every other test); bounds and panic-on-off-board;
deep clone; exhaustive "every target is on-board, ≠ from, no duplicates" over all pieces × all squares on
both 5×5 and 8×8; 180°-rotation symmetry for knight/king.

**`internal/challenge`** — 10k draws at a fixed PCG seed asserting `Target ∈ Solutions`,
`Solutions == MoveTargets(From)`, `From != Target`, exactly one piece on the board; anti-repetition
windows; determinism for a seed (this is what makes a bug report reproducible); distance specs;
**impossible spec still returns and does not hang** (run under a watchdog goroutine so a regression fails
rather than wedging CI); **coverage** — 5k rook draws must hit all 25 squares with min bucket >1% of max,
which catches "the star is always in the same corner", exactly the bug a child notices before you do;
pawn `Decoys` produces capture targets; `BenchmarkNext` sub-microsecond.

**`internal/layout`** — table over the five device rows in §3 plus a degenerate 200×200: board square
within 1e-9, board ⊆ safe, header/board/footer ordered and non-overlapping, stack ≤ `Safe.H`,
`Cell*Cols == Board.W`, and `Cell >= MinTap` for all real-phone cases. Plus `HitCell(centre of
CellRect(f,r)) == (f,r)` round-trip for every cell, header/footer taps returning `ok=false`, the star
magnet accepting a near-miss but **not** stealing a tap on a different solution square, and insets respected.

**`internal/anim`** — tween progress/delay/monotonicity; **every ease lands exactly on 1 at t=1**
(`EaseOutBack` overshoots mid-flight but must land, or pieces visibly miss their square); confetti
lifetime with the backing slice emptied (no leak); determinism for a seed.

**`internal/sfx`** — `len(b) == round(dur*rate) * 2ch * 4B`; no sample outside [-1,1]; **|first| and
|last| samples < 0.01**, proving the anti-click envelope applied; arpeggio note count.

No tests for `render`/`game` — they need a GPU. The mitigation is architectural: keep them thin and push
every decision worth testing into the pure packages. The one exception is the `render` **path tables**
(shapes are data): assert each piece path is closed with bounds inside the unit box. That catches the
most likely piece-drawing bug with no display.

CI: `go test -race ./internal/...` runs fully headless — the pure packages import no Ebitengine and the
`sfx` tests exercise only `Render`, never `audio.NewContext`, so no audio device is opened.

---

## 9. Build order

1. `go mod init`, `internal/chess` + tests. **Green before anything renders.**
2. `internal/challenge` + tests, incl. the coverage and no-hang tests.
3. `internal/layout` + `internal/anim` + `internal/sfx` synth + tests.
4. `internal/render` paths/palette/font; `internal/input`; `internal/game` with `LayoutF`.
5. `main.go` + `mobile/`. `go build ./... && go vet ./...` clean.
6. **WASM build → open on the phone.** First real look at it. Iterate on layout/colour/feel here — this
   loop is seconds long and needs no Android toolchain.
7. `go run .` on the Mac for desktop iteration alongside.
8. Android toolchain install → `make bind` (watch for the Apple-Silicon NDK path issue) → Gradle project
   → `make apk` → `make install`. First failure is almost always the `EbitenView` import path; check
   `chessapp-sources.jar` for the real generated path.
9. `make verify-16k` before ever touching the Play Console.

---

## 10. Verification

| Check | Command | Confirms |
|---|---|---|
| Pure logic | `go test -race ./internal/...` | movement, generation, layout maths, easing, audio envelopes |
| Decoupling | the `go list -deps \| grep -q hajimehoshi` guardrail | chess/challenge/layout/anim stay renderer-free |
| Full typecheck, no system deps | `GOOS=js GOARCH=wasm go build ./...` | every line of rendering code compiles |
| Real phone, no NDK | `make wasm serve` → Chrome on the Edge 50 Neo | portrait layout, tap targets, feel, perf |
| Desktop | `go run .` on the Mac | window resize behaviour, mouse input |
| Device | `make install && make logs` | the actual deliverable |
| Play readiness | `make verify-16k` | `align 2**14` on every LOAD segment |

**Acceptance for milestone 1:** on the Edge 50 Neo in portrait — home screen with a huge Play button and
six tappable piece buttons; board fills most of the width with cells ≥48dp; tapping the star animates the
piece and fires confetti + a cheer; a new challenge appears immediately; wrong taps produce only a gentle
wobble and a soft sound, never a failure state; nothing is hidden behind the status bar, nav bar, or
camera cutout; and it holds 60 fps with confetti on screen.
