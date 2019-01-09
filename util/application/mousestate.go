package application

import "github.com/sansebasko/engine/window"

// MouseState keeps track of the state of pressed mouse buttons.
type MouseState struct {
	win    window.IWindow
	states map[window.MouseButton]bool
}

// NewMouseState returns a new MouseState object.
func NewMouseState(win window.IWindow) *MouseState {

	ms := new(MouseState)
	ms.win = win
	ms.states = map[window.MouseButton]bool{
		window.MouseButtonLeft:   false,
		window.MouseButtonRight:  false,
		window.MouseButtonMiddle: false,
	}

	// Subscribe to window mouse events
	ms.win.SubscribeID(window.OnMouseUp, &ms, ms.onMouse)
	ms.win.SubscribeID(window.OnMouseDown, &ms, ms.onMouse)

	return ms
}

// Dispose unsubscribes from the window events.
func (ms *MouseState) Dispose() {

	ms.win.UnsubscribeID(window.OnMouseUp, &ms)
	ms.win.UnsubscribeID(window.OnMouseDown, &ms)
}

// Pressed returns whether the specified mouse button is currently pressed.
func (ms *MouseState) Pressed(b window.MouseButton) bool {

	return ms.states[b]
}

// onMouse receives mouse events and updates the internal map of states.
func (ms *MouseState) onMouse(evname string, ev interface{}) {

	mev := ev.(*window.MouseEvent)
	switch evname {
	case window.OnMouseUp:
		ms.states[mev.Button] = false
	case window.OnMouseDown:
		ms.states[mev.Button] = true
	}
}
