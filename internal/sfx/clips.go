package sfx

import (
	"bytes"
	"embed"
	"io"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// ClipID names a spoken clip. They are called clips, not voices, because Voice
// already means one tone of a synthesised effect.
type ClipID uint8

const (
	ClipPawn ClipID = iota
	ClipKnight
	ClipBishop
	ClipRook
	ClipQueen
	ClipKing
	ClipYay
	ClipPerfect
	ClipOops
	ClipWhich
	ClipCareful
	ClipWhiteTurn
	ClipBlackTurn
	ClipWhiteWins
	ClipBlackWins
	ClipGreatJob
	clipCount
)

// clipFiles maps each clip to its file in voice/. What each one says is in
// voice/clips.txt, which `make voice` turns into the .wav files.
var clipFiles = map[ClipID]string{
	ClipPawn:      "pawn",
	ClipKnight:    "knight",
	ClipBishop:    "bishop",
	ClipRook:      "rook",
	ClipQueen:     "queen",
	ClipKing:      "king",
	ClipYay:       "yay",
	ClipPerfect:   "perfect",
	ClipOops:      "oops",
	ClipWhich:     "which",
	ClipCareful:   "careful",
	ClipWhiteTurn: "white-turn",
	ClipBlackTurn: "black-turn",
	ClipWhiteWins: "white-wins",
	ClipBlackWins: "black-wins",
	ClipGreatJob:  "great-job",
}

//go:embed voice
var clipFS embed.FS

// clipPCM reads a clip and resamples it to the audio context's rate as 16-bit
// stereo. A clip that is missing or will not decode comes back as an error and
// the bank simply leaves that sound silent.
func clipPCM(id ClipID, sampleRate int) ([]byte, error) {
	raw, err := clipFS.ReadFile("voice/" + clipFiles[id] + ".wav")
	if err != nil {
		return nil, err
	}
	s, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(s)
}

// loadClips decodes every clip once, up front, so saying one is just a rewind.
func (b *Bank) loadClips() {
	b.clips = make(map[ClipID]*audio.Player, clipCount)
	for id := ClipID(0); id < clipCount; id++ {
		pcm, err := clipPCM(id, SampleRate)
		if err != nil {
			continue
		}
		b.clips[id] = b.ctx.NewPlayerFromBytes(pcm)
	}
}

// Say plays a spoken clip. Only one is ever heard at a time: a new one cuts
// the last short, so two sentences never talk over each other. Effects from
// Play are separate and keep layering as before.
func (b *Bank) Say(id ClipID) {
	p := b.clips[id]
	if p == nil {
		return
	}
	if b.speaking != nil && b.speaking.IsPlaying() {
		b.speaking.Pause()
	}
	_ = p.Rewind()
	p.Play()
	b.speaking = p
}
