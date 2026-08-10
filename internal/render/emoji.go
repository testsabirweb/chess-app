package render

import (
	"embed"
	"math/rand/v2"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/emoji/*.svg
var emojiFS embed.FS

// emojiNames holds the Unicode codepoint file names (e.g. "1f436") in a stable
// sorted order, so an emoji index is safe to store in game state.
var emojiNames []string

// StarEmoji is the plain gold star used as the tap target.
const StarEmoji = "2b50"

func init() {
	entries, err := emojiFS.ReadDir("assets/emoji")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		emojiNames = append(emojiNames, strings.TrimSuffix(e.Name(), ".svg"))
	}
	sort.Strings(emojiNames)
}

// EmojiName returns the sticker name for an index (wrapping, so callers never
// have to bounds-check).
func EmojiName(i int) string {
	if len(emojiNames) == 0 {
		return ""
	}
	return emojiNames[((i%len(emojiNames))+len(emojiNames))%len(emojiNames)]
}

// RandomEmoji picks a reward sticker, never the plain star (that one is the
// target, so it would not feel like a prize).
func RandomEmoji(rng *rand.Rand) int {
	if len(emojiNames) == 0 {
		return 0
	}
	for i := 0; i < 8; i++ {
		n := rng.IntN(len(emojiNames))
		if emojiNames[n] != StarEmoji {
			return n
		}
	}
	return 0
}

type emojiKey struct {
	name string
	px   int
}

var (
	emojiCache = map[emojiKey]*ebiten.Image{}
	emojiMu    sync.Mutex
)

// rasterSizes is the ladder every SVG raster snaps to. A sticker that shrinks
// as it flies into the tray would otherwise rasterise itself again at every
// size step, which is a visible hitch on a phone. Snapping means at most two or
// three rasters per emoji for the whole animation, and a small cache.
var rasterSizes = []int{24, 32, 48, 64, 96, 128, 192, 256, 320}

// quantize rounds a pixel size up to the next entry on the ladder.
func quantize(px int) int {
	for _, s := range rasterSizes {
		if px <= s {
			return s
		}
	}
	return rasterSizes[len(rasterSizes)-1]
}

// EmojiImage rasterises an emoji SVG at (roughly) the requested pixel size.
func EmojiImage(name string, px int) *ebiten.Image {
	px = quantize(px)
	key := emojiKey{name: name, px: px}
	emojiMu.Lock()
	defer emojiMu.Unlock()
	if img, ok := emojiCache[key]; ok {
		return img
	}
	data, err := emojiFS.ReadFile(path.Join("assets/emoji", name+".svg"))
	if err != nil {
		return nil
	}
	img, err := rasterSVG(data, px)
	if err != nil {
		return nil
	}
	emojiCache[key] = img
	return img
}

// DrawEmoji paints an emoji centred on (cx, cy) with the given diameter,
// rotation (radians) and opacity.
func DrawEmoji(dst *ebiten.Image, name string, cx, cy, size float64, rot, alpha float64) {
	if size <= 0 || alpha <= 0 {
		return
	}
	img := EmojiImage(name, int(size+0.5))
	if img == nil {
		return
	}
	b := img.Bounds()
	s := size / float64(b.Dx())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(s, s)
	if rot != 0 {
		op.GeoM.Rotate(rot)
	}
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleAlpha(float32(alpha))
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(img, op)
}
