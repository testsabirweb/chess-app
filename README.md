# Toddler Chess

A toddler-friendly "find the star" chess game for Android.

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
make bind          # rebuild native Android library (slow, first time ~minutes)
make apk           # build debug APK
make verify-16k    # Play Store page-size check
```
