// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gui

import (
	"github.com/sansebasko/engine/gui/assets/icon"
	"github.com/sansebasko/engine/window"
)

const (
	checkON  = string(icon.CheckBox)
	checkOFF = string(icon.CheckBoxOutlineBlank)
	radioON  = string(icon.RadioButtonChecked)
	radioOFF = string(icon.RadioButtonUnchecked)
)

// CheckRadio is a GUI element that can be either a checkbox or a radio button
type CheckRadio struct {
	Panel             // Embedded panel
	Label      *Label // Text label
	icon       *Label
	styles     *CheckRadioStyles
	check      bool
	cursorOver bool
	state      bool
	codeON     string
	codeOFF    string
	groups     []*RadioGroup // Slice of pointers to all radio groups the radio button is a member of
}

// RadioGroup holds only radio buttons and ensures
// that at most one of them is selected at a time
type RadioGroup struct {
	name               string        // the name of this radio group
	members            []*CheckRadio // Slice of pointers to the toggle button members
	DeselectingAllowed bool          // Whether deselecting is allowed for its members
}

// CheckRadioStyle contains the styling of a CheckRadio
type CheckRadioStyle BasicStyle

// CheckRadioStyles contains an CheckRadioStyle for each valid GUI state
type CheckRadioStyles struct {
	Normal   CheckRadioStyle
	Over     CheckRadioStyle
	Focus    CheckRadioStyle
	Disabled CheckRadioStyle
}

// NewCheckBox creates and returns a pointer to a new CheckBox widget
// with the specified text
func NewCheckBox(text string) *CheckRadio {

	return newCheckRadio(true, text)
}

// NewRadioButton creates and returns a pointer to a new RadioButton widget
// with the specified text
func NewRadioButton(text string) *CheckRadio {

	return newCheckRadio(false, text)
}

// newCheckRadio creates and returns a pointer to a new CheckRadio widget
// with the specified type and text
func newCheckRadio(check bool, text string) *CheckRadio {

	cb := new(CheckRadio)
	cb.styles = &StyleDefault().CheckRadio

	// Adapts to specified type: CheckBox or RadioButton
	cb.check = check
	cb.state = false
	if cb.check {
		cb.codeON = checkON
		cb.codeOFF = checkOFF
	} else {
		cb.codeON = radioON
		cb.codeOFF = radioOFF
	}

	// Initialize panel
	cb.Panel.Initialize(0, 0)

	// Subscribe to events
	cb.Panel.Subscribe(OnKeyDown, cb.onKey)
	cb.Panel.Subscribe(OnCursorEnter, cb.onCursor)
	cb.Panel.Subscribe(OnCursorLeave, cb.onCursor)
	cb.Panel.Subscribe(OnMouseDown, cb.onMouse)
	cb.Panel.Subscribe(OnEnable, func(evname string, ev interface{}) { cb.update() })

	// Creates label
	cb.Label = NewLabel(text)
	if len(text) > 0 {
		cb.Label.Subscribe(OnResize, func(evname string, ev interface{}) { cb.recalc() })
		cb.Panel.Add(cb.Label)
	}

	// Creates icon label
	cb.icon = NewIcon(" ")
	cb.Panel.Add(cb.icon)

	cb.recalc()
	cb.update()
	return cb
}

// NewRadioGroup creates and returns a pointer to a new radio group.
func NewRadioGroup(allowDeselecting bool) *RadioGroup {
	rg := new(RadioGroup)
	rg.DeselectingAllowed = allowDeselecting
	return rg
}

// Add adds the given radio button to the members of this radio group if it is
// not already contained and is indeed a radio button, in which case true is returned.
// Otherwise nothing happens and false is returned.
func (rg *RadioGroup) Add(radioButton *CheckRadio) bool {
	if radioButton.check || rg.Contains(radioButton) {
		return false
	}
	rg.members = append(rg.members, radioButton)
	radioButton.groups = append(radioButton.groups, rg)
	return true
}

// Remove removes the given button from the members of this toggle group if it is
// contained, in which case true is returned.
// Otherwise nothing happens and false is returned.
func (rg *RadioGroup) Remove(radioButton *CheckRadio) bool {
	if radioButton.check {
		return false
	}
	for i, b := range rg.members {
		if b == radioButton {
			rg.members = append(rg.members[:i], rg.members[i+1:]...)
			for i, g := range radioButton.groups {
				if g == rg {
					radioButton.groups = append(radioButton.groups[:i], radioButton.groups[i+1:]...)
					return true
				}
			}
		}
	}
	return false
}

// Contains returns true if the given radio button is a member of this radio group.
// Otherwise false is returned.
func (rg *RadioGroup) Contains(radioButton *CheckRadio) bool {
	if radioButton.check {
		return false
	}
	for _, b := range rg.members {
		if b == radioButton {
			return true
		}
	}
	return false
}

// deselectOthers deselects all radio buttons contained in this radio group except the
// given one.
func (rg *RadioGroup) deselectOthers(radioButton *CheckRadio) {
	if radioButton.check {
		return
	}
	for _, cb := range rg.members {
		if cb != radioButton {
			if cb.state {
				cb.state = false
				cb.update()
				cb.root.Dispatch(OnRadioGroup, cb)
			}
		}
	}
}

// deselectingAllowed returns true if none of all radio groups this radio button is a member of
// disallows deselecting. Otherwise false is returned.
func (b *CheckRadio) deselectingAllowed() bool {
	if b.check {
		return true
	}
	for _, g := range b.groups {
		if !g.DeselectingAllowed {
			return false
		}
	}
	return true
}

// GetGroup returns the radio groups of this button if any
func (cb *CheckRadio) GetGroups(name string) []*RadioGroup {

	var gg []*RadioGroup
	for _, g := range cb.groups {
		if g.name == name {
			gg = append(gg, g)
		}
	}
	return gg
}

// Value returns the current state of the checkbox
func (cb *CheckRadio) Value() bool {

	return cb.state
}

// SetValue sets the current state of the checkbox
func (cb *CheckRadio) SetValue(state bool) *CheckRadio {

	if state == cb.state {
		return cb
	}
	cb.state = state
	cb.update()
	cb.Dispatch(OnChange, nil)
	return cb
}

// SetStyles set the button styles overriding the default style
func (cb *CheckRadio) SetStyles(bs *CheckRadioStyles) {

	cb.styles = bs
	cb.update()
}

// toggleState toggles the current state of the checkbox/radiobutton
func (cb *CheckRadio) toggleState() {

	if cb.check {
		cb.state = !cb.state
	} else {
		if cb.state && !cb.deselectingAllowed() {
			return
		}
		if !cb.state {
			for _, g := range cb.groups {
				g.deselectOthers(cb)
			}
		}
		cb.state = !cb.state
		if len(cb.groups) > 0 {
			cb.root.Dispatch(OnRadioGroup, cb)
		}
	}
	cb.update()
	cb.Dispatch(OnChange, nil)
}

// onMouse process OnMouseDown events
func (cb *CheckRadio) onMouse(evname string, ev interface{}) {

	cb.root.SetKeyFocus(cb)
	cb.root.StopPropagation(Stop3D)
	cb.toggleState()
	// Dispatch OnClick for left mouse button down
	if evname == OnMouseDown {
		mev := ev.(*window.MouseEvent)
		if mev.Button == window.MouseButtonLeft {
			cb.Dispatch(OnClick, nil)
		}
	}
	cb.root.StopPropagation(StopAll)
}

// onCursor process OnCursor* events
func (cb *CheckRadio) onCursor(evname string, ev interface{}) {

	if evname == OnCursorEnter {
		cb.cursorOver = true
	} else {
		cb.cursorOver = false
	}
	cb.update()
	cb.root.StopPropagation(StopAll)
}

// onKey receives subscribed key events
func (cb *CheckRadio) onKey(evname string, ev interface{}) {

	kev := ev.(*window.KeyEvent)
	if evname == OnKeyDown && kev.Keycode == window.KeyEnter {
		cb.toggleState()
		cb.update()
		cb.Dispatch(OnClick, nil)
		cb.root.StopPropagation(Stop3D)
		return
	}
	return
}

// update updates the visual appearance of the checkbox
func (cb *CheckRadio) update() {

	if cb.state {
		cb.icon.SetText(cb.codeON)
	} else {
		cb.icon.SetText(cb.codeOFF)
	}

	if !cb.Enabled() {
		cb.applyStyle(&cb.styles.Disabled)
		return
	}
	if cb.cursorOver {
		cb.applyStyle(&cb.styles.Over)
		return
	}
	cb.applyStyle(&cb.styles.Normal)
}

// setStyle sets the specified checkradio style
func (cb *CheckRadio) applyStyle(s *CheckRadioStyle) {

	cb.Panel.ApplyStyle(&s.PanelStyle)
	cb.icon.SetColor4(&s.FgColor)
	cb.Label.SetColor4(&s.FgColor)
}

// recalc recalculates dimensions and position from inside out
func (cb *CheckRadio) recalc() {

	// Sets icon position
	cb.icon.SetFontSize(cb.Label.FontSize() * 1.3)
	cb.icon.SetPosition(0, 0)

	// Label position
	width := cb.icon.Width()
	if len(cb.Label.text) > 0 {
		spacing := float32(4)
		cb.Label.SetPosition(cb.icon.Width()+spacing, 0)
		width += spacing + cb.Label.Width()
	}

	// Content width
	cb.SetContentSize(width, cb.Label.Height())
}
