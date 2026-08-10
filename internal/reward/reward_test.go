package reward_test

import (
	"math/rand/v2"
	"testing"

	"github.com/testsabirweb/chess-app/internal/reward"
)

func pool(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

// A toddler noticed that the sticker awarded after capturing a star followed
// a pattern tied to which piece was in play. The cause: the reward pick and
// the puzzle generator shared one RNG stream, so the "random" emoji was
// really a side effect of how many random draws that piece's moves happened
// to cost - same piece, same move, same-looking sticker every time. Picker
// exists to be driven by its own RNG stream instead; these tests guard its
// actual contract (no repeats within a cycle, none across a reshuffle either).
func TestNextCyclesWithoutRepeats(t *testing.T) {
	p := reward.NewPicker(pool(12), rand.New(rand.NewPCG(1, 2)))
	seen := make(map[int]int)
	const cycles = 6
	for c := 0; c < cycles; c++ {
		round := make(map[int]bool)
		for i := 0; i < 12; i++ {
			n := p.Next()
			if round[n] {
				t.Fatalf("cycle %d: %d repeated before the bag was exhausted", c, n)
			}
			round[n] = true
			seen[n]++
		}
	}
	if len(seen) != 12 {
		t.Fatalf("only %d/12 distinct prizes were ever dealt", len(seen))
	}
}

func TestNextNeverRepeatsAcrossReshuffle(t *testing.T) {
	p := reward.NewPicker(pool(8), rand.New(rand.NewPCG(3, 4)))
	prev := -1
	for i := 0; i < 8*50; i++ {
		n := p.Next()
		if n == prev {
			t.Fatalf("draw %d: %d repeated immediately across a reshuffle", i, n)
		}
		prev = n
	}
}

func TestNextIsDeterministicForSeed(t *testing.T) {
	a := reward.NewPicker(pool(10), rand.New(rand.NewPCG(9, 9)))
	b := reward.NewPicker(pool(10), rand.New(rand.NewPCG(9, 9)))
	for i := 0; i < 100; i++ {
		if a.Next() != b.Next() {
			t.Fatalf("draw %d diverged for the same seed", i)
		}
	}
}

func TestNextOnSinglePrizePool(t *testing.T) {
	p := reward.NewPicker(pool(1), rand.New(rand.NewPCG(1, 1)))
	for i := 0; i < 5; i++ {
		if got := p.Next(); got != 0 {
			t.Fatalf("draw %d: want 0, got %d", i, got)
		}
	}
}

func TestNextOnEmptyPool(t *testing.T) {
	p := reward.NewPicker(nil, rand.New(rand.NewPCG(1, 1)))
	if got := p.Next(); got != 0 {
		t.Fatalf("empty pool: want 0, got %d", got)
	}
}

func TestIntN(t *testing.T) {
	p := reward.NewPicker(pool(4), rand.New(rand.NewPCG(5, 5)))
	for i := 0; i < 200; i++ {
		if n := p.IntN(7); n < 0 || n >= 7 {
			t.Fatalf("IntN(7) = %d, out of range", n)
		}
	}
	if got := p.IntN(0); got != 0 {
		t.Fatalf("IntN(0) = %d, want 0", got)
	}
}
