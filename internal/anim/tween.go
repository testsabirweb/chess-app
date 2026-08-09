package anim

import "math"

type Tween struct {
	Duration, Delay float64
	Ease            Ease
	elapsed         float64
	done            bool
}

func (tw *Tween) Reset() {
	tw.elapsed = 0
	tw.done = false
}

func (tw *Tween) Start() {
	tw.Reset()
}

func (tw *Tween) Update(dt float64) float64 {
	if tw.done {
		return 1
	}
	tw.elapsed += dt
	if tw.elapsed < tw.Delay {
		return 0
	}
	t := (tw.elapsed - tw.Delay) / tw.Duration
	if t >= 1 {
		tw.done = true
		return 1
	}
	if tw.Ease == nil {
		return t
	}
	return tw.Ease(t)
}

func (tw *Tween) Done() bool { return tw.done }

func (tw *Tween) Progress() float64 {
	if tw.Duration <= 0 {
		return 1
	}
	t := (tw.elapsed - tw.Delay) / tw.Duration
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

type Pulse struct {
	Period  float64
	elapsed float64
}

func (p *Pulse) Update(dt float64) float64 {
	if p.Period <= 0 {
		return 1
	}
	p.elapsed += dt
	phase := (p.elapsed / p.Period) * 2 * math.Pi
	return 0.85 + 0.15*(0.5+0.5*math.Sin(phase))
}
