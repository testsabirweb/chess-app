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
