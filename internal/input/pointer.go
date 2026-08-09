package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Event struct {
	X, Y    float64
	Pressed bool
}

type Tracker struct {
	JustPressed  []Event
	JustReleased []Event
	pressBuf     []Event
	releaseBuf   []Event
	touchBuf     []ebiten.TouchID
}

func (t *Tracker) Update() {
	t.JustPressed = t.JustPressed[:0]
	t.JustReleased = t.JustReleased[:0]

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		t.JustPressed = append(t.JustPressed, Event{X: float64(x), Y: float64(y), Pressed: true})
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		t.JustReleased = append(t.JustReleased, Event{X: float64(x), Y: float64(y)})
	}

	t.touchBuf = inpututil.AppendJustPressedTouchIDs(t.touchBuf[:0])
	for _, id := range t.touchBuf {
		x, y := ebiten.TouchPosition(id)
		t.JustPressed = append(t.JustPressed, Event{X: float64(x), Y: float64(y), Pressed: true})
	}
	t.touchBuf = inpututil.AppendJustReleasedTouchIDs(t.touchBuf[:0])
	for _, id := range t.touchBuf {
		x, y := inpututil.TouchPositionInPreviousTick(id)
		t.JustReleased = append(t.JustReleased, Event{X: float64(x), Y: float64(y)})
	}
}

func (t *Tracker) Pressed() []Event { return t.JustPressed }
