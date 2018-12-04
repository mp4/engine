// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gui

import (
	"github.com/sansebasko/engine/math32"
	"github.com/sansebasko/engine/window"
	"math"
)

// Splitter is a GUI element that splits two panels and can be adjusted
type Splitter struct {
	Panel                     // Embedded panel
	P0        Panel           // Left/Top panel
	P1        Panel           // Right/Bottom panel
	splitType SplitType       // relative (0-1), absolute (in pixels) or reverse absolute (in pixels)
	styles    *SplitterStyles // pointer to current styles
	spacer    Panel           // spacer panel
	horiz     bool            // horizontal or vertical splitter
	pos       float32         // relative position (0 to 1) of the center of the spacer panel (split type == Relative) or absolut position in pixels from left (split type == Absolute) or from right plus spacer width (split type == ReverseAbsolute)
	min       int             // minimal number of pixels of the top/left (split type == Relative/Absolute) or bottom/right (split type == ReverseAbsolute)
	max       int             // maximal number of pixels of the top/left (split type == Relative/Absolute) or bottom/right (split type == ReverseAbsolute)
	posLast   float32         // last position in pixels of the mouse cursor when dragging
	pressed   bool            // mouse button is pressed and dragging
	mouseOver bool            // mouse is over the spacer panel
}

// SplitterStyle contains the styling of a Splitter
type SplitterStyle struct {
	SpacerBorderColor math32.Color4
	SpacerColor       math32.Color4
	SpacerSize        float32
}

// SplitterStyles contains a SplitterStyle for each valid GUI state
type SplitterStyles struct {
	Normal SplitterStyle
	Over   SplitterStyle
	Drag   SplitterStyle
}

type SplitType int

const (
	Relative SplitType = iota
	Absolute
	ReverseAbsolute
)

// NewHSplitter creates and returns a pointer to a new horizontal splitter
// widget with the specified initial dimensions
func NewHSplitter(width, height float32) *Splitter {

	return newSplitter(true, width, height)
}

// NewVSplitter creates and returns a pointer to a new vertical splitter
// widget with the specified initial dimensions
func NewVSplitter(width, height float32) *Splitter {

	return newSplitter(false, width, height)
}

// newSpliter creates and returns a pointer of a new splitter with
// the specified orientation and initial dimensions.
func newSplitter(horiz bool, width, height float32) *Splitter {

	s := new(Splitter)
	s.splitType = Relative
	s.pos = 0.5
	s.min = 0
	s.max = math.MaxInt32
	s.horiz = horiz
	s.styles = &StyleDefault().Splitter
	s.Panel.Initialize(width, height)

	// Initialize left/top panel
	s.P0.Initialize(0, 0)
	s.Panel.Add(&s.P0)

	// Initialize right/bottom panel
	s.P1.Initialize(0, 0)
	s.Panel.Add(&s.P1)

	// Initialize spacer panel
	s.spacer.Initialize(0, 0)
	s.Panel.Add(&s.spacer)

	if horiz {
		s.spacer.SetBorders(0, 1, 0, 1)
	} else {
		s.spacer.SetBorders(1, 0, 1, 0)
	}

	s.Subscribe(OnResize, s.onResize)
	s.spacer.Subscribe(OnMouseDown, s.onMouse)
	s.spacer.Subscribe(OnMouseUp, s.onMouse)
	s.spacer.Subscribe(OnCursor, s.onCursor)
	s.spacer.Subscribe(OnCursorEnter, s.onCursor)
	s.spacer.Subscribe(OnCursorLeave, s.onCursor)
	s.update()
	s.recalc()
	return s
}

// SetSplitType sets the type of the split, which
// has an impact of how the split position is interpreted
func (s *Splitter) SetSplitType(splitType SplitType) {

	s.splitType = splitType
	s.recalc()
}

// SplitType returns the split type
func (s *Splitter) SplitType() SplitType {

	return s.splitType
}

// SetSplitMin sets the minimal number of pixels of the
// top/left panel if split type is Relative or Absolute.
// Otherwise it sets the minimal number of pixels of the
// bottom/right panel
func (s *Splitter) SetSplitMin(min int) {

	if min < 0 {
		s.min = 0
	} else if min > s.max {
		s.min = s.max
		s.max = min
	} else {
		s.min = min
	}
	s.SetSplit(s.pos)
}

// SplitMin returns the minimal number of pixels of the
// top/left panel if split type is Relative or Absolute.
// Otherwise it returns the minimal number of pixels of the
// bottom/right panel
func (s *Splitter) SplitMin() int {

	return s.min
}

// SetSplitMax sets the maximal number of pixels of the
// top/left panel if split type is Relative or Absolute.
// Otherwise it sets the maximal number of pixels of the
// bottom/right panel
func (s *Splitter) SetSplitMax(max int) {

	if max < 0 {
		s.max = 0
	} else if max < s.min {
		s.max = s.min
		s.min = max
	} else {
		s.max = max
	}
	s.SetSplit(s.pos)
}

// SplitMax returns the maximal number of pixels of the
// top/left panel if split type is Relative or Absolute.
// Otherwise it returns the maximal number of pixels of the
// bottom/right panel
func (s *Splitter) SplitMax() int {

	return s.max
}

// SetSplit sets the position of the splitter bar.
// It accepts a value from 0.0 to 1.0 if split type is relative,
// otherwise the given value is interpreted as pixel count
func (s *Splitter) SetSplit(pos float32) {

	s.setSplit(pos)
	s.recalc()
}

// Split returns the current position of the splitter bar.
// It returns a value from 0.0 to 1.0 if split type is relative,
// otherwise the width of the split
func (s *Splitter) Split() float32 {

	return s.pos
}

// onResize receives subscribed resize events for the whole splitter panel
func (s *Splitter) onResize(evname string, ev interface{}) {

	s.recalc()
}

// onMouse receives subscribed mouse events over the spacer panel
func (s *Splitter) onMouse(evname string, ev interface{}) {

	mev := ev.(*window.MouseEvent)
	switch evname {
	case OnMouseDown:
		s.pressed = true
		if mev.Button == window.MouseButtonLeft {
			if s.horiz {
				s.posLast = mev.Xpos
			} else {
				s.posLast = mev.Ypos
			}
			s.root.SetMouseFocus(&s.spacer)
		}
	case OnMouseUp:
		if mev.Button == window.MouseButtonLeft {
			s.root.SetCursorNormal()
			s.root.SetMouseFocus(nil)
		} else if mev.Button == window.MouseButtonRight && s.pressed {
			s.SetSplit(float32(s.min))
		}
		s.pressed = false
	default:
	}
	s.root.StopPropagation(Stop3D)
}

// onCursor receives subscribed cursor events over the spacer panel
func (s *Splitter) onCursor(evname string, ev interface{}) {

	if evname == OnCursorEnter {
		if s.horiz {
			s.root.SetCursorHResize()
		} else {
			s.root.SetCursorVResize()
		}
		s.mouseOver = true
		s.update()
	} else if evname == OnCursorLeave {
		s.root.SetCursorNormal()
		s.mouseOver = false
		s.update()
	} else if evname == OnCursor {
		if !s.pressed {
			return
		}
		cev := ev.(*window.CursorEvent)
		var delta float32
		pos := s.pos
		if s.horiz {
			delta = cev.Xpos - s.posLast
			s.posLast = cev.Xpos
			if s.splitType == Relative {
				pos += delta / s.ContentWidth()
			} else if s.splitType == Absolute {
				pos += delta
			} else {
				pos -= delta
			}
		} else {
			delta = cev.Ypos - s.posLast
			s.posLast = cev.Ypos
			if s.splitType == Relative {
				pos += delta / s.ContentHeight()
			} else if s.splitType == Absolute {
				pos += delta
			} else {
				pos -= delta
			}
		}
		s.setSplit(pos)
		s.recalc()
	}
	s.root.StopPropagation(Stop3D)
}

// setSplit sets the validated and clamped split position from the received value.
func (s *Splitter) setSplit(pos float32) {

	min := float32(s.min)
	max := float32(s.max)
	if s.splitType == Relative {
		if pos < 0 {
			pos = 0
		}
		if pos > 1 {
			pos = 1
		}
		if s.horiz {
			width := s.ContentWidth()
			if width == 0 {
				s.pos = pos
				return
			}
			p := width*pos - s.spacer.Width()/2
			if p < min {
				s.pos = (min + s.spacer.Width()/2) / width
			} else if p > max {
				s.pos = (max + s.spacer.Width()/2) / width
			} else {
				s.pos = pos
			}
		} else {
			height := s.ContentHeight()
			if height == 0 {
				s.pos = pos
				return
			}
			p := height*pos - s.spacer.Height()/2
			if p < min {
				s.pos = (min + s.spacer.Height()/2) / height
			} else if p > max {
				s.pos = (max + s.spacer.Height()/2) / height
			} else {
				s.pos = pos
			}
		}
	} else {
		if pos < min {
			s.pos = min
		} else if pos > max {
			s.pos = max
		} else {
			s.pos = pos
		}
	}
}

// update updates the splitter visual state
func (s *Splitter) update() {

	if s.pressed {
		s.applyStyle(&s.styles.Drag)
		return
	}
	if s.mouseOver {
		s.applyStyle(&s.styles.Over)
		return
	}
	s.applyStyle(&s.styles.Normal)
}

// applyStyle applies the specified splitter style
func (s *Splitter) applyStyle(ss *SplitterStyle) {

	s.spacer.SetBordersColor4(&ss.SpacerBorderColor)
	s.spacer.SetColor4(&ss.SpacerColor)
	if s.horiz {
		s.spacer.SetWidth(ss.SpacerSize)
	} else {
		s.spacer.SetHeight(ss.SpacerSize)
	}
}

// recalc recalculates the position and sizes of the internal panels
func (s *Splitter) recalc() {

	width := s.ContentWidth()
	height := s.ContentHeight()

	if s.horiz {
		// Calculate x position for spacer panel
		var spx float32
		if s.splitType == Relative {
			spx = width*s.pos - s.spacer.Width()/2
		} else if s.splitType == Absolute {
			spx = s.pos
		} else {
			spx = width - s.pos - s.spacer.Width()
		}

		if spx < 0 {
			spx = 0
		} else if spx > width-s.spacer.Width() {
			spx = width - s.spacer.Width()
		}
		// Left panel
		s.P0.SetPosition(0, 0)
		s.P0.SetSize(spx, height)
		// Spacer panel
		s.spacer.SetPosition(spx, 0)
		s.spacer.SetHeight(height)
		// Right panel
		s.P1.SetPosition(spx+s.spacer.Width(), 0)
		s.P1.SetSize(width-spx-s.spacer.Width(), height)
	} else {
		// Calculate y position for spacer panel
		var spy float32
		if s.splitType == Relative {
			spy = height*s.pos - s.spacer.Height()/2
		} else if s.splitType == Absolute {
			spy = s.pos
		} else {
			spy = height - s.pos - s.spacer.Height()
		}
		if spy < 0 {
			spy = 0
		} else if spy > height-s.spacer.Height() {
			spy = height - s.spacer.Height()
		}
		// Top panel
		s.P0.SetPosition(0, 0)
		s.P0.SetSize(width, spy)
		// Spacer panel
		s.spacer.SetPosition(0, spy)
		s.spacer.SetWidth(width)
		// Bottom panel
		s.P1.SetPosition(0, spy+s.spacer.Height())
		s.P1.SetSize(width, height-spy-s.spacer.Height())
	}
}
