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

### Knobs worth knowing

| What | Where |
|---|---|
| Dark square colour | `render.ColorBoardD` in `internal/render/palette.go` |
| How far the star can be planted | `maxJourney` in `internal/game/play.go` |
| Stickers per celebration | `milestoneEvery` in `internal/game/play.go` |
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
make apk           # build debug APK
make verify-16k    # Play Store page-size check
```

`make shots` drives the game headlessly-ish through a scripted tap sequence and
writes PNGs, which is the quickest way to check a UI change without a phone.

## CI/CD — build APK on GitHub Release (free)

The workflow in `.github/workflows/android-release.yml` runs **only when you publish a GitHub Release**, not on every push. It runs tests, builds the APK, checks 16 KB alignment, and attaches the APK to the release.

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
