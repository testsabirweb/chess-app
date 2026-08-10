// Package reward decides which sticker a captured star hands out. It is a
// pure package - no ebiten import, so it is unit-testable without a GPU or
// audio device, unlike internal/game which owns a process-wide audio.Context.
package reward

import "math/rand/v2"

// Picker deals from a shuffled "bag" of prize indices without replacement:
// every prize is handed out exactly once before any of them repeat. This is
// what stops a run of unlucky coin flips from awarding the same sticker
// several times in a row.
type Picker struct {
	pool []int
	rng  *rand.Rand
	bag  []int
	last int
}

// NewPicker builds a Picker over pool (typically every non-interface emoji
// index). rng should NOT be shared with anything else whose draw count varies
// with gameplay - if it is, the "random" prize ends up correlated with how
// many random numbers gameplay happened to consume first, which reads as a
// fixed pattern rather than chance.
func NewPicker(pool []int, rng *rand.Rand) *Picker {
	p := make([]int, len(pool))
	copy(p, pool)
	return &Picker{pool: p, rng: rng, last: -1}
}

// Next deals the next prize.
func (p *Picker) Next() int {
	if len(p.bag) == 0 {
		p.refill()
	}
	if len(p.bag) == 0 {
		return 0
	}
	n := p.bag[len(p.bag)-1]
	p.bag = p.bag[:len(p.bag)-1]
	p.last = n
	return n
}

// refill reshuffles a fresh bag, swapping the previous prize out of the deal
// position if it landed there - otherwise a reshuffle boundary could produce
// the same prize twice in a row, which would look like Next() failing to
// honour "no repeats" even though each bag on its own is repeat-free.
func (p *Picker) refill() {
	bag := make([]int, len(p.pool))
	copy(bag, p.pool)
	p.rng.Shuffle(len(bag), func(i, j int) { bag[i], bag[j] = bag[j], bag[i] })
	if p.last >= 0 && len(bag) > 1 && bag[len(bag)-1] == p.last {
		bag[0], bag[len(bag)-1] = bag[len(bag)-1], bag[0]
	}
	p.bag = bag
}

// IntN exposes the picker's RNG for other cosmetic, reward-flavoured choices
// (e.g. which celebration message to show) that should share its real
// randomness rather than a deterministic gameplay stream.
func (p *Picker) IntN(n int) int {
	if n <= 0 {
		return 0
	}
	return p.rng.IntN(n)
}
