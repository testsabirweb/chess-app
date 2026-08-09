package sfx

import (
	"math"
)

const SampleRate = 48000

type Wave uint8

const (
	WaveSine Wave = iota
	WaveTriangle
)

type Voice struct {
	Freq       float64
	Duration   float64
	Amplitude  float64
	Wave       Wave
	FreqEnd    float64 // linear glide; 0 = constant
	Attack     float64
	Release    float64
	StartDelay float64
}

func Render(voices []Voice, sampleRate int) []byte {
	if sampleRate <= 0 {
		sampleRate = SampleRate
	}
	maxEnd := 0.0
	for _, v := range voices {
		end := v.StartDelay + v.Duration
		if end > maxEnd {
			maxEnd = end
		}
	}
	n := int(math.Round(maxEnd * float64(sampleRate)))
	if n <= 0 {
		return nil
	}
	mono := make([]float32, n)
	for _, v := range voices {
		renderVoice(mono, v, sampleRate)
	}
	out := make([]byte, n*2*4)
	for i, s := range mono {
		if s > 1 {
			s = 1
		}
		if s < -1 {
			s = -1
		}
		bits := math.Float32bits(s)
		off := i * 8
		out[off] = byte(bits)
		out[off+1] = byte(bits >> 8)
		out[off+2] = byte(bits >> 16)
		out[off+3] = byte(bits >> 24)
		// stereo duplicate
		out[off+4] = out[off]
		out[off+5] = out[off+1]
		out[off+6] = out[off+2]
		out[off+7] = out[off+3]
	}
	return out
}

func renderVoice(buf []float32, v Voice, rate int) {
	attack := v.Attack
	if attack < 0.005 {
		attack = 0.005
	}
	release := v.Release
	if release < 0.03 {
		release = 0.03
	}
	start := int(v.StartDelay * float64(rate))
	samples := int(v.Duration * float64(rate))
	amp := v.Amplitude
	if amp <= 0 {
		amp = 0.3
	}
	phase := 0.0
	for i := 0; i < samples; i++ {
		idx := start + i
		if idx >= len(buf) {
			break
		}
		t := float64(i) / float64(rate)
		freq := v.Freq
		if v.FreqEnd > 0 {
			freq = v.Freq + (v.FreqEnd-v.Freq)*(t/v.Duration)
		}
		phase += freq / float64(rate)
		var sample float64
		switch v.Wave {
		case WaveTriangle:
			sample = triangle(phase)
		default:
			sample = math.Sin(2 * math.Pi * phase)
		}
		env := envelope(t, v.Duration, attack, release)
		buf[idx] += float32(sample * amp * env)
	}
}

func envelope(t, dur, attack, release float64) float64 {
	if t < attack {
		return t / attack
	}
	if t > dur-release {
		return math.Max(0, (dur-t)/release)
	}
	return 1
}

func triangle(phase float64) float64 {
	x := phase - math.Floor(phase)
	if x < 0.5 {
		return 4*x - 1
	}
	return 3 - 4*x
}
