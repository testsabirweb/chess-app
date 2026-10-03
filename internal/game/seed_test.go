package game

import (
	"testing"
	"time"
)

// Every launch must deal different puzzles, so the seed has to come from the
// clock - and the puzzle and reward streams must not share one.
func TestEntropySeedChangesWithTheClockAndSalt(t *testing.T) {
	a1, a2 := entropySeed(puzzleSalt)
	time.Sleep(2 * time.Millisecond)
	b1, b2 := entropySeed(puzzleSalt)
	if a1 == b1 && a2 == b2 {
		t.Fatal("the seed did not change between launches")
	}
	c1, c2 := entropySeed(rewardSalt)
	if c1 == b1 && c2 == b2 {
		t.Fatal("the puzzle and reward streams share a seed")
	}
}
