package application

import (
	"github.com/sansebasko/engine/window"
	"time"
)

// Millis
const DefaultDoubleClickDuration = 300

// MouseState keeps track of the state of pressed mouse buttons.
type MouseState struct {
	win    window.IWindow
	lastButton window.MouseButton
	DoubleClickDuration time.Duration
	states map[window.MouseButton]*mouseButtonState
}

type mouseButtonState struct {
	clickCount int
	lastClick time.Time
}

func (s *mouseButtonState) doubleClicked() bool {
	return s.clickCount == 2 || s.clickCount == -2
}

// NewMouseState returns a new MouseState object.
func NewMouseState(win window.IWindow) *MouseState {

	ms := new(MouseState)
	ms.win = win
	ms.DoubleClickDuration = DefaultDoubleClickDuration * time.Millisecond
	ms.states = map[window.MouseButton]*mouseButtonState{
		window.MouseButtonLeft:   {clickCount: 0, lastClick: time.Now()},
		window.MouseButtonRight:  {clickCount: 0, lastClick: time.Now()},
		window.MouseButtonMiddle: {clickCount: 0, lastClick: time.Now()},
	}

	// Subscribe to window mouse events
	ms.win.SubscribeID(window.OnMouseUp, &ms, ms.onMouseUp)
	ms.win.SubscribeID(window.OnMouseDown, &ms, ms.onMouseDown)

	return ms
}

// Dispose unsubscribes from the window events.
func (ms *MouseState) Dispose() {

	ms.win.UnsubscribeID(window.OnMouseUp, &ms)
	ms.win.UnsubscribeID(window.OnMouseDown, &ms)
}

// Pressed returns whether the specified mouse button is currently pressed.
func (ms *MouseState) Pressed(b window.MouseButton) bool {

	return ms.states[b].clickCount > 0
}

// Pressed returns whether the left mouse button is currently pressed.
func (ms *MouseState) LeftPressed() bool {

	return ms.states[window.MouseButtonLeft].clickCount > 0
}

// Pressed returns whether the right mouse button is currently pressed.
func (ms *MouseState) RightPressed() bool {

	return ms.states[window.MouseButtonRight].clickCount > 0
}

// Pressed returns whether the middle mouse button is currently pressed.
func (ms *MouseState) MiddlePressed() bool {

	return ms.states[window.MouseButtonMiddle].clickCount > 0
}

// Pressed returns whether the user left double clicked.
func (ms *MouseState) LeftDoubleClicked() bool {

	return ms.lastButton == window.MouseButtonLeft && ms.states[window.MouseButtonLeft].doubleClicked()
}

// Pressed returns whether the user right double clicked.
func (ms *MouseState) RightDoubleClicked() bool {

	return ms.lastButton == window.MouseButtonRight && ms.states[window.MouseButtonRight].doubleClicked()
}

// Pressed returns whether the user middle double clicked.
func (ms *MouseState) MiddleDoubleClicked() bool {

	return ms.lastButton == window.MouseButtonMiddle && ms.states[window.MouseButtonMiddle].doubleClicked()
}

// onMouse receives mouse events and updates the internal map of states.
func (ms *MouseState) onMouseUp(evname string, ev interface{}) {

	mev := ev.(*window.MouseEvent)
	if ms.states[mev.Button].clickCount > 0 {
		ms.states[mev.Button].clickCount *= -1
	}
}

// onMouse receives mouse events and updates the internal map of states.
func (ms *MouseState) onMouseDown(evname string, ev interface{}) {

	mev := ev.(*window.MouseEvent)
	ms.lastButton = mev.Button

	now := time.Now()

	if ms.states[mev.Button].clickCount == 0 {
		ms.states[mev.Button].clickCount = 1
		ms.states[mev.Button].lastClick = now
		return
	}

	if ms.states[mev.Button].clickCount == -1 {
		if ms.states[mev.Button].lastClick.Add(ms.DoubleClickDuration).Before(now) {
			ms.states[mev.Button].clickCount = 1
			ms.states[mev.Button].lastClick = now
			return
		}

		ms.states[mev.Button].clickCount = 2
		return
	}

	ms.states[mev.Button].clickCount = 1
	ms.states[mev.Button].lastClick = now
}
