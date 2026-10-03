# Toddler Chess

A toddler-friendly "find the star" chess game for Android, built for a
Motorola Edge 50 Neo (1080×2400, portrait).

## How it plays

Pick a piece on the home screen. A star appears somewhere the piece can
**reach** — one, two or three legal moves away, not just one step. Tap the
piece, tap one of the mint dots, and the piece hops there; repeat until it
lands on the star. Most stars can be reached by more than one route.

Landing on the star pops a random emoji sticker that flies into the tray at the
bottom. Every fifth sticker gets a small celebration.

There is no way to lose. A wrong tap gives a soft wobble and nothing else, and
if a wandering piece can no longer reach the star, the star quietly hops to a
square it can reach.

### Other games

The piece cards (and **PLAY**) always start the star game, so it has no button
of its own. Every other game gets a round button under **PLAY**: games that deal
their own pieces start when tapped; games that need a piece are picked first (the
button gets a gold frame) and started by a piece card. Tap the button again to
go back to the star game.

- **Collect the stars** (the gem button) — two stars at first, three after a few
  rounds, collected in any order. Each pop is a small celebration; the last one
  wins the sticker. Picking the quickest route through all of them earns the
  "Perfect!" party. If the piece wanders so a star can't be reached any more, that
  star hops somewhere it can. Pick it, then choose a piece card to start.
- **Which piece?** — two or three pieces and a star. Exactly one piece can reach
  the star in a single move. Tap it to pick it up, then tap the star to move it
  there yourself. A wrong pick shows what
  that piece *can* do, then fades out of the running, so the
  next guess is easier. Getting it right first time earns the "Perfect!" party.
  The knight is never dealt in this game.

### Knobs worth knowing

| What | Where |
|---|---|
| Dark square colour | `render.ColorBoardD` in `internal/render/palette.go` |
| How far the star can be planted | `maxJourney` in `internal/game/play.go` |
| Stickers per celebration | `milestoneEvery` in `internal/game/kit.go` |
| Pieces dealt by the "own pieces" games | `beginnerPieces` in `internal/game/mode.go` |
| How long a wrong "Which piece?" pick shows its moves | `whichShowDur` in `internal/game/which.go` |
| The sticker set | drop more Twemoji SVGs into `internal/render/assets/emoji/` |

## Play on your Mac (desktop)

```bash
make run
```

## Play in the phone browser (no install)

On your Mac:

```bash
make wasm serve
```

On the phone (same Wi‑Fi), open Chrome and go to `http://<your-mac-ip>:8080`.

## Install the Android app

**You never run commands on the phone.** Everything below is on the Mac.

### Option A — USB cable (easiest after setup)

1. On the phone: **Settings → About phone → tap Build number 7 times** to enable Developer options.
2. **Settings → Developer options → USB debugging** → turn on.
3. Plug the phone into the Mac with a USB cable. Tap **Allow** on the phone if asked.
4. On the Mac:

```bash
export JAVA_HOME="$(brew --prefix openjdk@17)/libexec/openjdk.jdk/Contents/Home"
export ANDROID_HOME="$(brew --prefix)/share/android-commandlinetools"
export PATH="$ANDROID_HOME/platform-tools:$JAVA_HOME/bin:$(go env GOPATH)/bin:$PATH"

make install
```

That installs and opens the app. Rebuild anytime with `make apk` then `make install`.

### Option B — No cable (AirDrop / Files)

1. On the Mac, build the APK once: `make apk`
2. AirDrop or copy this file to the phone:
   `android/app/build/outputs/apk/debug/app-debug.apk`
3. On the phone, open the APK and allow install from that app (e.g. Files or AirDrop) when prompted.

## Develop

```bash
make test          # unit tests
make shots         # render PNG screenshots at Edge 50 Neo metrics into shots/
make bind          # rebuild native Android library (slow, first time ~minutes)
make apk           # build debug APK (local dev)
make apk-release   # build release APK (unsigned locally unless KEYSTORE_* env set)
make install-release  # USB install release APK (uninstall debug build first)
make verify-16k    # Play Store page-size check (release APK if present, else debug)
```

`make shots` drives the game headlessly-ish through a scripted tap sequence and
writes PNGs, which is the quickest way to check a UI change without a phone.

## CI/CD — build APK on GitHub Release (free)

The workflow in `.github/workflows/android-release.yml` runs **only when you publish a GitHub Release**, not on every push. It runs tests, builds a **release-signed** APK, checks 16 KB alignment, and attaches the APK to the release.

### One-time: release signing secrets

CI builds with a persistent release keystore (not the debug key). Add these repository secrets under **Settings → Secrets and variables → Actions**:

| Secret | Value |
|--------|-------|
| `KEYSTORE_BASE64` | Base64 of your `.jks` file |
| `KEYSTORE_PASSWORD` | Keystore password |
| `KEY_ALIAS` | Key alias (e.g. `chessapp`) |
| `KEY_PASSWORD` | Key password |

Generate a keystore once, back it up somewhere safe, and never commit it:

```bash
keytool -genkeypair -v -keystore chessapp-release.jks \
  -alias chessapp -keyalg RSA -keysize 4096 -validity 10000
base64 -i chessapp-release.jks | pbcopy   # paste into KEYSTORE_BASE64
```

Losing the keystore means every future user must uninstall before installing again — it cannot be recovered.

**First install after switching from debug:** uninstall the old app on the device, then install the release APK. After that, updates install in place.

### How to get an APK from CI

1. On GitHub: **Releases → Create a new release**
2. Choose a tag (e.g. `v0.1.0`) and publish
3. Wait ~15–25 minutes for the workflow to finish
4. Download **`toddler-chess-v0.1.0.apk`** from the release assets
5. AirDrop or copy to your phone and install (no terminal on the phone)

### Cost

| Repo type | Cost |
|-----------|------|
| **Public** | Free — unlimited Actions minutes |
| **Private** | Free tier — 2,000 Actions minutes/month (~80+ release builds) |

To test the workflow without creating a release: **Actions → Android Release → Run workflow**.
