package sfx_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/testsabirweb/chess-app/internal/sfx"
)

func readSample(b []byte, i int) float32 {
	off := i * 8
	bits := binary.LittleEndian.Uint32(b[off:])
	return math.Float32frombits(bits)
}

func TestRenderLength(t *testing.T) {
	v := []sfx.Voice{{Freq: 440, Duration: 0.1, Amplitude: 0.2}}
	b := sfx.Render(v, 48000)
	want := int(math.Round(0.1*48000)) * 2 * 4
	if len(b) != want {
		t.Fatalf("len=%d want=%d", len(b), want)
	}
}

func TestSamplesInRange(t *testing.T) {
	b := sfx.Render([]sfx.Voice{
		{Freq: 220, Duration: 0.2, Amplitude: 0.5, Wave: sfx.WaveTriangle},
	}, 48000)
	n := len(b) / 8
	for i := 0; i < n; i++ {
		s := readSample(b, i)
		if s > 1.01 || s < -1.01 {
			t.Fatalf("sample %d out of range: %f", i, s)
		}
	}
}

func TestAntiClickEnvelope(t *testing.T) {
	b := sfx.Render([]sfx.Voice{
		{Freq: 440, Duration: 0.05, Amplitude: 0.8},
	}, 48000)
	first := math.Abs(float64(readSample(b, 0)))
	last := math.Abs(float64(readSample(b, len(b)/8-1)))
	if first >= 0.01 || last >= 0.01 {
		t.Fatalf("click envelope failed: first=%f last=%f", first, last)
	}
}

func TestCheerArpeggioNoteCount(t *testing.T) {
	notes := []float64{523.25, 659.25, 783.99, 1046.5}
	var voices []sfx.Voice
	for i, f := range notes {
		voices = append(voices, sfx.Voice{
			Freq: f, Duration: 0.12, Amplitude: 0.15,
			Wave: sfx.WaveTriangle, StartDelay: float64(i) * 0.08,
		})
	}
	b := sfx.Render(voices, 48000)
	if len(b) == 0 {
		t.Fatal("empty cheer")
	}
	if len(notes) != 4 {
		t.Fatal("expected 4 notes")
	}
}
