package sfx

import "testing"

func TestEveryClipHasAUniqueFile(t *testing.T) {
	seen := map[string]ClipID{}
	for id := ClipID(0); id < clipCount; id++ {
		name, ok := clipFiles[id]
		if !ok || name == "" {
			t.Fatalf("clip %d has no file name", id)
		}
		if other, dup := seen[name]; dup {
			t.Fatalf("clips %d and %d share the file %q", other, id, name)
		}
		seen[name] = id
	}
}

// The tests never open an audio device, so this checks the embedded files
// decode rather than that they play.
func TestEveryClipDecodes(t *testing.T) {
	for id := ClipID(0); id < clipCount; id++ {
		pcm, err := clipPCM(id, SampleRate)
		if err != nil {
			t.Fatalf("clip %q: %v", clipFiles[id], err)
		}
		if len(pcm) == 0 {
			t.Fatalf("clip %q decoded to nothing", clipFiles[id])
		}
	}
}

func TestSayOnABankWithoutClipsIsSilent(t *testing.T) {
	b := &Bank{}
	b.Say(ClipPawn) // must not panic
}
