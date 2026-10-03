package render

import "testing"

// However many different stickers are drawn at however many sizes, the cache
// stays within its budget (plus the one image just added).
func TestEmojiCacheStaysWithinBudget(t *testing.T) {
	maxOne := 320 * 320 * 4
	for _, idx := range RewardEmojiIndices() {
		name := EmojiName(idx)
		for _, px := range rasterSizes {
			if EmojiImage(name, px) == nil {
				t.Fatalf("%s at %d did not rasterise", name, px)
			}
			emojiMu.Lock()
			got := emojiBytes
			emojiMu.Unlock()
			if got > emojiCacheBudget+maxOne {
				t.Fatalf("cache holds %d bytes, budget %d", got, emojiCacheBudget)
			}
		}
	}
}

// The bookkeeping must match what is actually in the map.
func TestEmojiCacheByteCountIsExact(t *testing.T) {
	for _, idx := range RewardEmojiIndices()[:40] {
		for _, px := range []int{48, 128, 192} {
			EmojiImage(EmojiName(idx), px)
		}
	}
	emojiMu.Lock()
	defer emojiMu.Unlock()
	sum := 0
	for _, e := range emojiCache {
		sum += e.bytes
	}
	if sum != emojiBytes {
		t.Fatalf("tracked %d bytes, cache really holds %d", emojiBytes, sum)
	}
}

// What was just drawn survives eviction; the oldest goes first.
func TestEmojiCacheEvictsLeastRecentlyUsed(t *testing.T) {
	emojiMu.Lock()
	emojiCache = map[emojiKey]*emojiEntry{}
	emojiBytes = 0
	emojiMu.Unlock()

	first := EmojiName(RewardEmojiIndices()[0])
	EmojiImage(first, 320)
	for _, idx := range RewardEmojiIndices()[1:40] {
		EmojiImage(EmojiName(idx), 320) // 400 KB each: far over the budget
	}
	emojiMu.Lock()
	_, stillThere := emojiCache[emojiKey{name: first, px: 320}]
	emojiMu.Unlock()
	if stillThere {
		t.Fatal("the oldest image should have been evicted")
	}
	last := EmojiName(RewardEmojiIndices()[39])
	emojiMu.Lock()
	_, ok := emojiCache[emojiKey{name: last, px: 320}]
	emojiMu.Unlock()
	if !ok {
		t.Fatal("the newest image should still be cached")
	}
}
