# Platform plan: screen sizes, auto-update, knight animation

Three independent pieces of work, all code.

**Recommended order**, because one of these unblocks another:

1. **Release signing + versionCode** (§2.1). Nothing else about updates works until this is done, and
   it changes how the APK gets onto his device, so do it before shipping anything else.
2. **Screen sizes** (§1). Self-contained and fully unit-testable.
3. **Knight animation** (§3). Self-contained, and the most fun.
4. **The updater itself** (§2.2 onward), once signing is sorted.

---

# 1. Support every screen size

## Where it already works

`layout.Compute` in `internal/layout/metrics.go` is already resolution-independent: it takes `w, h,
scale, insets` and derives the board, header, footer, cell size and both font sizes from them. Nothing
is hardcoded to a device. `Game.LayoutF` recomputes metrics every frame, and the manifest already
declares `configChanges` for `screenSize|screenLayout|smallestScreenSize|density`, so live resizing
(foldables, split screen, desktop windows) already flows through correctly.

So this is not a rewrite. There are four specific gaps.

## Gap 1 — landscape is unimplemented, and on Android 16 that is no longer optional

`Metrics.Portrait` is computed in `Compute` and **never read anywhere in the codebase** (only
`layout_test.go` mentions it, to *exempt* landscape from the minimum-tap-size assertion). There is no
landscape branch: `Compute` always stacks header / board / footer vertically, so in landscape the board
shrinks to the available height and the header and footer squeeze to nothing.

The manifest pins `android:screenOrientation="portrait"`, which has hidden this so far.

**The catch:** `targetSdk` is 36. On large screens, recent Android versions ignore an activity's
orientation and resizability restrictions — a portrait lock is not honoured on a tablet-class display.
There has been a temporary manifest opt-out, but it is being removed. **Verify the current behaviour in
the Android developer docs for API 36 before deciding** — I am not certain of the exact flag name or
which API level it stops working at, and it matters. Either way, the safe assumption is: on a tablet
the game will be asked to render landscape whether or not the manifest says portrait.

Recommendation: **keep the portrait lock for phones** (rotation is a distraction for a toddler and
nothing about the game benefits from it) **and implement a real landscape layout** so tablets and
foldables are not broken.

The landscape layout that fits this game: board on one side, header and tray stacked in a column
beside it.

```go
// In Compute, after m.Safe is established, branch on m.Portrait.
//
// Landscape: the board takes the full safe height and sits against the leading
// edge; the header and footer become a single column beside it. Stacking them
// above and below a landscape board leaves a board barely wider than a thumb.
if !m.Portrait {
	side := math.Min(m.Safe.H-2*gap, m.Safe.W*0.62)
	panelW := m.Safe.W - side - 2*gap
	if panelW < m.MinTap*3 {
		// Very wide-but-short window: give the panel its minimum and let the
		// board take what is left.
		panelW = m.MinTap * 3
		side = math.Min(m.Safe.H-2*gap, m.Safe.W-panelW-2*gap)
	}
	m.Board = Rect{
		X: m.Safe.X + gap,
		Y: m.Safe.Y + (m.Safe.H-side)/2,
		W: side, H: side,
	}
	panelX := m.Board.X + side + gap
	m.Header = Rect{X: panelX, Y: m.Safe.Y, W: panelW, H: m.Safe.H * 0.35}
	m.Footer = Rect{X: panelX, Y: m.Safe.Y + m.Safe.H*0.45, W: panelW, H: m.Safe.H * 0.55}
	// ... then the shared Cell / TitleSize / BodySize tail below.
}
```

Pull the `m.Cell`, `m.TitleSize`, `m.BodySize` assignments out into the shared tail so both branches use
them. `backButtonRect` and `trayRect` in `play.go` read from `m.Safe` and `m.Footer`, so they follow
automatically — but check the tray: at `traySlots = 8` across a narrow side panel the slots will be
tiny, so in landscape either reduce the slot count or stack the tray vertically.

## Gap 2 — `homeLayout` can overflow the safe area

`homeLayout` in `internal/game/home.go` sizes everything as a percentage of `Safe.H` with no dp caps:

```go
	titleH := s.H * 0.12
	playH := math.Max(s.H*0.12, m.MinTap*1.25)
	...
	cardsH := s.H - titleH - playH - labelH - trayH - 4*gap
	if cardsH < m.MinTap*2 {
		cardsH = m.MinTap * 2
	}
```

When `Safe.H` is short (landscape, a small window, split screen), the `math.Max` floors and that final
clamp push the total above `Safe.H` and the grid runs off the bottom — the clamp fixes the card height
but never re-checks the sum.

Fix: give each band a dp cap as well as a percentage, compute the total, and if it still exceeds
`Safe.H`, scale every band down by the same factor rather than clamping one of them. Then in landscape
lay the six cards out as `6×1` or `2×3` beside the Play button instead of `3×2` below it.

Add an assertion to the tests: the sum of the home bands must never exceed `Safe.H`.

## Gap 3 — nothing enforces the minimum tap target

`Metrics.TapOK()` exists and is called **only** from `layout_test.go`. It is not consulted at runtime,
and `Compute` has a fallback that can make the board *larger* than the safe area:

```go
	if side < m.MinTap*float64(cols) {
		side = math.Min(m.MinTap*float64(cols), math.Min(m.Safe.W, m.Safe.H))
	}
```

On a tiny window this produces a board that overflows. Either make the fallback respect the safe area
in both axes, or accept cells below `MinTap` on genuinely tiny screens and drop the fallback — an
undersized board that fits is better than a correctly-sized one that is clipped. Prefer the latter:
delete the fallback, and let the test assert `TapOK()` only for real device sizes.

## Gap 4 — the test matrix and the screenshot tool cover one phone

`internal/layout/layout_test.go` has five entries and `Makefile`'s `shots` target hardcodes the Edge 50
Neo. Extend the `devices` table to cover the real range:

```go
var devices = []device{
	{"edge50", 1080, 2400, 2.75},        // existing
	{"flagship", 1440, 3120, 3.5},       // existing
	{"budget", 720, 1600, 2.0},          // existing
	{"dev", 432, 960, 1.0},              // existing
	{"small-old", 480, 800, 1.5},        // Android 7-era phone
	{"tall-21x9", 1080, 2640, 3.0},      // very tall
	{"tablet-port", 1600, 2560, 2.0},    // 10" portrait
	{"tablet-land", 2560, 1600, 2.0},    // 10" landscape
	{"fold-inner", 1812, 2176, 2.4},     // near-square foldable
	{"split-screen", 1080, 900, 2.75},   // short window
	{"tiny", 400, 400, 1.0},             // degenerate
}
```

Once the landscape branch exists, **delete the `d.name != "landscape"` exemption** in
`TestComputeDevices` — that exemption is the marker for this whole gap. Add invariants that hold in
both orientations: board square, board inside `Safe`, header/footer not overlapping the board, and the
whole stack within `Safe`.

Then make the screenshot target sweep a few sizes so regressions are visible:

```make
shots:
	go run ./cmd/shot -out shots/home       -scene home -stickers 4 -w 393 -h 873 -scale 2.75
	go run ./cmd/shot -out shots/rook       -scene play -piece rook -stickers 4 -w 393 -h 873 -scale 2.75
	go run ./cmd/shot -out shots/tablet     -scene play -piece rook -stickers 4 -w 800 -h 1280 -scale 2.0
	go run ./cmd/shot -out shots/landscape  -scene play -piece rook -stickers 4 -w 1280 -h 800 -scale 2.0
	go run ./cmd/shot -out shots/small      -scene play -piece rook -stickers 4 -w 320 -h 533 -scale 1.5
```

## One hardware limit worth checking

The manifest requires OpenGL ES 3.0:

```xml
<uses-feature android:glEsVersion="0x00030000" android:required="true" />
```

`minSdk` is 24 (Android 7), but ES 3.0 is the stricter constraint. Anything from roughly 2013 onward
has it; a genuinely ancient tablet may not, and the Play/installer will refuse the APK rather than fail
at runtime. Check any old device against this before planning to use one — `adb shell dumpsys SurfaceFlinger |
grep GLES` reports it.

---

# 2. Auto-update from GitHub releases

## 2.1 The blocker: the APK is debug-signed with a throwaway key

This has to be fixed first, and it explains a symptom you may already have hit.

`.github/workflows/android-release.yml` runs `make apk`, which is `./gradlew assembleDebug`. A debug
build is signed with `~/.android/debug.keystore`, and the Android Gradle Plugin **generates that file
if it is absent**. The workflow caches `~/.gradle` (via `setup-java`'s `cache: gradle`) but not
`~/.android` — so every release build on a fresh runner invents a **new random signing key**.

Consequences:

- Two releases are signed by different keys, so Android refuses to install one over the other
  (`INSTALL_FAILED_UPDATE_INCOMPATIBLE`). Every update today requires uninstall-then-install.
- No auto-update mechanism of any kind can work until this changes. Not the in-app updater, not
  Obtainium, not the Play Store.

**Fix:** generate one release keystore, keep it forever, and put it in GitHub Secrets.

```bash
keytool -genkeypair -v -keystore chessapp-release.jks \
  -alias chessapp -keyalg RSA -keysize 4096 -validity 10000
base64 -i chessapp-release.jks | pbcopy   # paste into the secret
```

Store `KEYSTORE_BASE64`, `KEYSTORE_PASSWORD`, `KEY_ALIAS`, `KEY_PASSWORD` as repository secrets. **Back
the `.jks` file up somewhere you will still have in five years** — losing it means every future release
needs a fresh install, and it is not recoverable. Never commit it.

In `android/app/build.gradle.kts`:

```kotlin
    signingConfigs {
        create("release") {
            val ks = System.getenv("KEYSTORE_FILE")
            if (ks != null) {
                storeFile = file(ks)
                storePassword = System.getenv("KEYSTORE_PASSWORD")
                keyAlias = System.getenv("KEY_ALIAS")
                keyPassword = System.getenv("KEY_PASSWORD")
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            // Fall back to unsigned locally; CI supplies the env vars.
            if (System.getenv("KEYSTORE_FILE") != null) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
    }
```

Add an `apk-release` target to the `Makefile` (`./gradlew assembleRelease`) and switch the workflow's
"Build debug APK" step to it, decoding the secret to a file first.

**One-time cost on his device:** the currently installed app is debug-signed, so the first
release-signed build must be installed over an uninstall. After that, updates are in place forever.

## 2.2 versionCode must increase with each release

`build.gradle.kts` hardcodes `versionCode = 1` / `versionName = "1.0"`, so Android sees every release as
the same version and has no basis for "newer". Derive both from the git tag.

```kotlin
    defaultConfig {
        // Supplied by CI from the release tag; defaults keep local builds working.
        versionCode = (providers.gradleProperty("versionCode").orNull ?: "1").toInt()
        versionName = providers.gradleProperty("versionName").orNull ?: "dev"
    }
```

In the workflow, before the build:

```yaml
      - name: Derive version from tag
        id: ver
        run: |
          tag="${{ github.event.release.tag_name }}"      # e.g. v1.5.0
          v="${tag#v}"
          IFS=. read -r MA MI PA <<< "$v"
          echo "code=$(( MA * 10000 + MI * 100 + PA ))" >> "$GITHUB_OUTPUT"
          echo "name=$v" >> "$GITHUB_OUTPUT"
```

and pass `-PversionCode=... -PversionName=...` through `make apk-release`. Your current tags
(`v1.4.0` → `10400`) fit this scheme, and it stays monotonic as long as minor and patch stay under 100.

## 2.3 Option A — Obtainium, and write no code at all

Worth naming because it is free and immediate: **Obtainium** is an Android app that watches a GitHub
releases page and installs updates from it. Point it at `testsabirweb/chess-app`, and once §2.1 and
§2.2 are done, updates arrive with one tap and no code in this repo.

The trade-off against option B is only that updating is a separate app you open rather than something
the game handles itself.

## 2.4 Option B — an in-app updater

Put this entirely on the Java side. The install step is Android-only, and the check should never block
the game loop.

**Manifest additions:**

```xml
    <uses-permission android:name="android.permission.INTERNET" />
    <uses-permission android:name="android.permission.REQUEST_INSTALL_PACKAGES" />
```

plus a `FileProvider` inside `<application>`, with a `res/xml/file_paths.xml` exposing the download
directory. Downloading to `getExternalFilesDir(null)` needs no storage permission.

**New `Updater.java`:**

1. On a background thread at startup, GET
   `https://api.github.com/repos/testsabirweb/chess-app/releases/latest`.
   Unauthenticated, so no token; the rate limit is 60/hour per IP, which one launch-time check will
   never approach.
2. Parse `tag_name`, strip the `v`, convert to a versionCode with the *same* arithmetic as §2.2, and
   compare against `getPackageManager().getPackageInfo(getPackageName(), 0).getLongVersionCode()`.
   Comparing derived integers rather than strings avoids any "is 1.10 newer than 1.9" problem.
3. If newer, find the asset whose `name` ends in `.apk` (they are named
   `toddler-chess-<tag>.apk`) and download `browser_download_url` — note it redirects, so follow
   redirects.
4. Hand the file to the system installer:

```java
Uri uri = FileProvider.getUriForFile(ctx, ctx.getPackageName() + ".fileprovider", apk);
Intent i = new Intent(Intent.ACTION_VIEW)
        .setDataAndType(uri, "application/vnd.android.package-archive")
        .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_ACTIVITY_NEW_TASK);
ctx.startActivity(i);
```

Android shows its own confirmation dialog, and **there is no way to skip it.** Silent installation is
permitted only to an app that is the device owner, which means a full kiosk provisioning flow on a
factory-reset device — out of scope here. So the update will always end in one "Install" tap by an
adult, whichever option you pick.

**The UX problem, and it is the real design question here:** an "Update" button on screen is a button a
toddler will press. Do not add one. Instead follow the precedent already set by `backButtonRect`, which
is deliberately placed somewhere "that has to be reached for":

- Check and download in the background, silently. Store the downloaded path.
- Surface it as a small dot on the back chevron in the top-left corner — no text, no colour that draws
  the eye.
- Fire the install intent only on a **long press of two seconds or more** on that corner. A toddler's
  taps are short; a deliberate parent hold is not.

Nothing about an update should ever interrupt a puzzle.

---

# 3. Knight animation: teaching the L

## What happens now

Every piece animates identically. `startMove` records one start and one end point, and `Update` walks a
single tween with one parabolic hop:

```go
	prog := p.moveTween.Update(ctx.DT)
	arc := -math.Sin(prog*math.Pi) * m.Cell * 0.28
	p.pieceX = lerp(p.fromX, p.toX, prog)
	p.pieceY = lerp(p.fromY, p.toY, prog) + arc
```

For a knight that draws a straight diagonal-ish line between two squares, which is exactly the shape
the piece does *not* move in. The animation currently teaches the wrong thing.

## The design

Animate the knight as **two legs with a beat between them**, and leave the L on the board:

1. **Long leg first** — the two-square component, straight along a rank or file, with a low hop.
2. **A hold of about 80ms at the corner**, with `SndStep`. This is the beat that makes it "two… then
   one" rather than one smear.
3. **Short leg** — the one-square perpendicular component, low hop, then `SndLand`.
4. **A breadcrumb trail** drawn under the piece along origin → corner → destination: a chunky rounded
   polyline with a dot at the corner, fading out over ~0.6s after landing.

Long leg first, always — it matches how the move is said out loud, and it makes the turn the memorable
part.

**Timing:** leg A 0.30s, hold 0.08s, leg B 0.24s — about 0.62s total, against 0.42s today. Deliberately
slower; this is a teaching beat, and it is still well under a second.

**The trade-off, stated so you can overrule it:** a knight's other distinctive property is that it
*jumps over* things, which one tall arc conveys better than two low ones. Two legs teach the L, one arc
teaches the jump, and you cannot have both in the same movement. The L is the more useful lesson here —
on a 5×5 board with decoys only ever generated for pawns, the knight almost never has anything to jump
over, so the jump property has nothing to demonstrate. If you disagree, keep one tall arc and rely on
the trail alone to show the shape.

## Implementation sketch

**`internal/game/play.go`** — generalise the single move into waypoints:

```go
// moveWaypoints are the corners the piece travels through, start included. Most
// pieces have two entries (straight from A to B); the knight has three, so the
// animation goes two squares and then one across, the way the move is taught.
type movePath struct {
	pts  [3]struct{ x, y float64 }
	n    int     // 2 or 3
	leg  int     // which leg is running
	hold float64 // seconds left of the pause at the corner
}
```

In `startMove`, build the corner for a knight from the move's components: if `|df| == 2` the corner is
`(to.File, from.Rank)`, otherwise `(from.File, to.Rank)`. That is the long leg first in both cases.
Everything else keeps `n = 2` and behaves exactly as it does today.

In `Update`'s `stateMoving` branch, run the current leg's tween, arc it with the existing
`-math.Sin(prog*math.Pi) * m.Cell * arcScale` (use a smaller `arcScale`, around `0.16`, for two-leg
moves), and on leg completion either start the hold, start the next leg, or call `land`. **`land` must
run only after the final leg** — it is what applies the move to the board.

Keep `p.moveTo` as-is so `land` needs no changes.

**`internal/render/board.go`** — a trail:

```go
// DrawMoveTrail draws the path a piece took, so the shape of the move stays on
// screen for a moment after the piece has landed. fade scales it out.
func DrawMoveTrail(dst *ebiten.Image, m layout.Metrics, pts [][2]float64, fade float64)
```

A rounded thick line per segment plus a filled dot at each interior corner. Colour: the amber
pick-up family (`ColorPicked`) rather than the blue hint dots — this is "where the piece went", not
"where it can go", and the two must not be confused.

**Draw order** matters: trail under the piece, above the square washes.

## Optional, and a good idea if the L lands well

A **knight intro**: the first time a knight puzzle is dealt in a session, animate the L once on the
otherwise-empty board before play starts. It is the piece-intro idea from the earlier plan, scoped to
the one piece whose move is genuinely hard to guess. Gate it behind a `seenKnightIntro bool` on `Game`
so it happens once per session, not once per puzzle.

## Verifying

The screenshot script's frame numbers assume a 0.42s move, so knight shots will land mid-animation
once the move takes 0.62s. Add knight-specific frames, or a `-piece knight` run with later frames:

```
go run ./cmd/shot -out /tmp/knight -scene play -piece knight -w 393 -h 873 -scale 2.75
```

Check: the long leg runs first, the corner is visibly a corner rather than a rounded curve, the trail
reads as an L after landing, and the two hops feel like two rather than a stutter.
