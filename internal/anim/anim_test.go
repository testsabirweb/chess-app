package anim_test

import (
	"math"
	"testing"

	"github.com/testsabirweb/chess-app/internal/anim"
)

func TestTweenProgress(t *testing.T) {
	tw := anim.Tween{Duration: 1, Ease: anim.EaseOutCubic}
	var last float64
	for i := 0; i < 60; i++ {
		p := tw.Update(1.0 / 60)
		if p < last {
			t.Fatal("tween not monotonic")
		}
		last = p
	}
	if !tw.Done() {
		t.Fatal("tween should be done")
	}
	if math.Abs(last-1) > 1e-9 {
		t.Fatalf("tween ended at %f, want 1", last)
	}
}

func TestTweenDelay(t *testing.T) {
	tw := anim.Tween{Duration: 0.5, Delay: 0.5}
	if tw.Update(0.1) != 0 {
		t.Fatal("should respect delay")
	}
}

func TestEaseLandAtOne(t *testing.T) {
	eases := []anim.Ease{
		anim.Linear,
		anim.EaseOutCubic,
		anim.EaseInOutCubic,
		anim.EaseOutBack,
		anim.EaseOutElastic,
	}
	for _, e := range eases {
		if math.Abs(e(1)-1) > 1e-6 {
			t.Fatalf("ease did not land at 1")
		}
	}
}

func TestConfettiEmpties(t *testing.T) {
	var c anim.Confetti
	c.Burst(nil, 100, 100, 20, 50)
	for i := 0; i < 300; i++ {
		c.Update(1.0/60, 400)
	}
	if c.Alive() {
		t.Fatal("confetti should finish")
	}
}
