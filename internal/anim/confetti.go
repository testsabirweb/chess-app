package anim

import (
	"image/color"
	"math"
	"math/rand/v2"
)

const (
	ShapeRect uint8 = iota
	ShapeCircle
	ShapeTri
)

type Particle struct {
	X, Y, VX, VY, Rot, VRot, Life, MaxLife, Size float64
	Color                                          color.RGBA
	Shape                                          uint8
}

type Confetti struct {
	particles []Particle
}

func (c *Confetti) Alive() bool { return len(c.particles) > 0 }

func (c *Confetti) Burst(rng *rand.Rand, x, y float64, n int, unit float64) {
	if rng == nil {
		rng = rand.New(rand.NewPCG(1, 2))
	}
	palette := []color.RGBA{
		{255, 99, 132, 255},
		{255, 205, 86, 255},
		{75, 192, 192, 255},
		{54, 162, 235, 255},
		{153, 102, 255, 255},
		{255, 159, 64, 255},
	}
	for i := 0; i < n; i++ {
		angle := rng.Float64() * 2 * math.Pi
		speed := unit * (2 + rng.Float64()*4)
		life := 0.8 + rng.Float64()*0.6
		c.particles = append(c.particles, Particle{
			X: x, Y: y,
			VX: math.Cos(angle) * speed,
			VY: math.Sin(angle)*speed - unit*2,
			Rot: rng.Float64() * math.Pi,
			VRot: (rng.Float64() - 0.5) * 8,
			Life: life, MaxLife: life,
			Size: unit * (0.08 + rng.Float64()*0.12),
			Color: palette[rng.IntN(len(palette))],
			Shape: uint8(rng.IntN(3)),
		})
	}
}

func (c *Confetti) Update(dt float64, gravity float64) {
	alive := c.particles[:0]
	for _, p := range c.particles {
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.VY += gravity * dt
		drag := math.Pow(0.98, dt*60)
		p.VX *= drag
		p.VY *= drag
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Rot += p.VRot * dt
		alive = append(alive, p)
	}
	c.particles = alive
}

func (c *Confetti) ForEach(f func(Particle)) {
	for _, p := range c.particles {
		f(p)
	}
}

func ParticleAlpha(p Particle) float64 {
	return math.Min(1, p.Life/0.4)
}
