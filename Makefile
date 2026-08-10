.PHONY: run test shots wasm serve tools bind apk install logs verify-16k clean

run:
	go run .

test:
	go test -race ./...

# Scripted screenshots at the Motorola Edge 50 Neo's real metrics
# (393x873 logical at dp scale 2.75 = 1080x2400 physical).
shots:
	go run ./cmd/shot -out shots/home -scene home -stickers 4 -w 393 -h 873 -scale 2.75
	go run ./cmd/shot -out shots/rook -scene play -piece rook -stickers 4 -w 393 -h 873 -scale 2.75

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

install: apk
	adb install -r android/app/build/outputs/apk/debug/app-debug.apk
	adb shell am start -n com.testsabirweb.chessapp/.MainActivity

logs:
	adb logcat -s Go:V GoLog:V ChessApp:V AndroidRuntime:E

verify-16k:
	@tmp=$$(mktemp -d); \
	unzip -q android/app/build/outputs/apk/debug/app-debug.apk -d $$tmp; \
	for so in $$tmp/lib/*/*.so; do \
	  echo "== $$so"; \
	  llvm-objdump -p "$$so" | grep LOAD; \
	done

clean:
	rm -rf shots
	rm -f web/game.wasm web/wasm_exec.js
	rm -rf android/.gradle android/build android/app/build
