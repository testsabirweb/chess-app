.PHONY: run test shots wasm serve tools bind apk apk-release install install-release logs verify-16k clean

run:
	go run .

test:
	go test -race ./...

# Scripted screenshots at the Motorola Edge 50 Neo's real metrics
# (393x873 logical at dp scale 2.75 = 1080x2400 physical).
shots:
	go run ./cmd/shot -out shots/home       -scene home -stickers 4 -w 393 -h 873 -scale 2.75
	go run ./cmd/shot -out shots/rook       -scene play -piece rook -stickers 4 -w 393 -h 873 -scale 2.75
	go run ./cmd/shot -out shots/tablet     -scene play -piece rook -stickers 4 -w 800 -h 1280 -scale 2.0
	go run ./cmd/shot -out shots/landscape  -scene play -piece rook -stickers 4 -w 1280 -h 800 -scale 2.0
	go run ./cmd/shot -out shots/small      -scene play -piece rook -stickers 4 -w 320 -h 533 -scale 1.5

wasm:
	GOOS=js GOARCH=wasm go build -o web/game.wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/

serve: wasm
	cd web && python3 -m http.server 8080

tools:
	go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@v2.9.9

bind:
	CGO_LDFLAGS="-Wl,-z,max-page-size=16384 -Wl,-z,common-page-size=16384" \
	ebitenmobile bind -target android/arm64 -androidapi 24 \
	  -javapkg com.testsabirweb.chessapp -o android/app/libs/chessapp.aar -v ./mobile

apk:
	cd android && ./gradlew assembleDebug

apk-release:
	cd android && ./gradlew assembleRelease \
	  $(if $(versionCode),-PversionCode=$(versionCode)) \
	  $(if $(versionName),-PversionName=$(versionName))

install: apk
	adb install -r android/app/build/outputs/apk/debug/app-debug.apk
	adb shell am start -n com.testsabirweb.chessapp/.MainActivity

# Release-signed APK; first install over a debug build requires uninstall first.
install-release: apk-release
	adb install -r android/app/build/outputs/apk/release/app-release.apk
	adb shell am start -n com.testsabirweb.chessapp/.MainActivity

logs:
	adb logcat -s Go:V GoLog:V ChessApp:V AndroidRuntime:E

verify-16k:
	@apk="$(APK)"; \
	if [ -z "$$apk" ]; then \
	  if [ -f android/app/build/outputs/apk/release/app-release.apk ]; then \
	    apk=android/app/build/outputs/apk/release/app-release.apk; \
	  else \
	    apk=android/app/build/outputs/apk/debug/app-debug.apk; \
	  fi; \
	fi; \
	echo "Checking $$apk"; \
	tmp=$$(mktemp -d); \
	unzip -q "$$apk" -d $$tmp; \
	for so in $$tmp/lib/*/*.so; do \
	  echo "== $$so"; \
	  llvm-objdump -p "$$so" | grep LOAD; \
	done

clean:
	rm -rf shots
	rm -f web/game.wasm web/wasm_exec.js
	rm -rf android/.gradle android/build android/app/build
