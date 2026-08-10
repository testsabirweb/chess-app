package sfx

import (
	"github.com/hajimehoshi/ebiten/v2/audio"
)

type ID uint8

const (
	SndButton ID = iota
	SndLand
	SndCheer
	SndOops
	SndNear
	SndStep
	SndPop
	SndHop
	SndMilestone
)

type Bank struct {
	ctx     *audio.Context
	players map[ID]*audio.Player
}

func NewBank() *Bank {
	ctx := audio.NewContext(SampleRate)
	b := &Bank{ctx: ctx, players: make(map[ID]*audio.Player)}
	b.load(SndButton, []Voice{{Freq: 660, Duration: 0.08, Amplitude: 0.12}})
	b.load(SndLand, []Voice{{Freq: 220, Duration: 0.15, Amplitude: 0.2, FreqEnd: 160}})
	b.load(SndCheer, cheerVoices())
	b.load(SndOops, []Voice{{Freq: 392, Duration: 0.2, Amplitude: 0.06, FreqEnd: 330, Wave: WaveTriangle}})
	b.load(SndNear, []Voice{{Freq: 523.25, Duration: 0.18, Amplitude: 0.1, Wave: WaveTriangle}})
	// A soft tick for an ordinary step along the way to the star.
	b.load(SndStep, []Voice{{Freq: 494, Duration: 0.09, Amplitude: 0.09, FreqEnd: 587, Wave: WaveTriangle}})
	// The sticker popping out of the star.
	b.load(SndPop, []Voice{{Freq: 784, Duration: 0.14, Amplitude: 0.13, FreqEnd: 1318, Wave: WaveTriangle}})
	// The star hopping to a new square when the piece can no longer reach it.
	b.load(SndHop, []Voice{
		{Freq: 659.25, Duration: 0.1, Amplitude: 0.09, Wave: WaveTriangle},
		{Freq: 880, Duration: 0.12, Amplitude: 0.09, Wave: WaveTriangle, StartDelay: 0.09},
	})
	b.load(SndMilestone, milestoneVoices())
	return b
}

func cheerVoices() []Voice {
	notes := []float64{523.25, 659.25, 783.99, 1046.5}
	out := make([]Voice, len(notes))
	for i, f := range notes {
		out[i] = Voice{
			Freq: f, Duration: 0.12, Amplitude: 0.12,
			Wave: WaveTriangle, StartDelay: float64(i) * 0.08,
		}
	}
	return out
}

func milestoneVoices() []Voice {
	notes := []float64{523.25, 659.25, 783.99, 1046.5, 1318.5, 1567.98}
	out := make([]Voice, 0, len(notes)+2)
	for i, f := range notes {
		out = append(out, Voice{
			Freq: f, Duration: 0.16, Amplitude: 0.11,
			Wave: WaveTriangle, StartDelay: float64(i) * 0.1,
		})
	}
	// A held chord to finish on.
	out = append(out,
		Voice{Freq: 523.25, Duration: 0.6, Amplitude: 0.08, Wave: WaveTriangle, StartDelay: 0.62},
		Voice{Freq: 783.99, Duration: 0.6, Amplitude: 0.07, Wave: WaveTriangle, StartDelay: 0.62},
	)
	return out
}

func (b *Bank) load(id ID, voices []Voice) {
	data := Render(voices, SampleRate)
	p := b.ctx.NewPlayerF32FromBytes(data)
	b.players[id] = p
}

func (b *Bank) Play(id ID) {
	p, ok := b.players[id]
	if !ok || p == nil {
		return
	}
	_ = p.Rewind()
	p.Play()
}
