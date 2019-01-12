// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gui

// DockLayout is the layout for docking panels to the internal edges of their parent.
type DockLayout struct {
	incumbents []Edge
}

type Edge int

// DockLayoutParams specifies the edge to dock to.
type DockLayoutParams struct {
	Edge Edge
}

// The different types of docking.
const (
	DockTop Edge = iota + 1
	DockRight
	DockBottom
	DockLeft
	DockCenter
)

// NewDockLayout returns a pointer to a new DockLayout.
func NewDockLayout(incumbents ...Edge) *DockLayout {

	return &DockLayout{incumbents: incumbents}
}

// Recalc (which satisfies the ILayout interface) recalculates the positions and sizes of the children panels.
func (dl *DockLayout) Recalc(ipan IPanel) {

	pan := ipan.GetPanel()

	y := float32(0)
	x := float32(0)
	w := pan.Width()
	h := pan.Height()

	if !contains(dl.incumbents, DockTop) {
		dl.incumbents = append(dl.incumbents, DockTop)
	}
	if !contains(dl.incumbents, DockBottom) {
		dl.incumbents = append(dl.incumbents, DockBottom)
	}
	if !contains(dl.incumbents, DockLeft) {
		dl.incumbents = append(dl.incumbents, DockLeft)
	}
	if !contains(dl.incumbents, DockRight) {
		dl.incumbents = append(dl.incumbents, DockRight)
	}

	for _, incumbent := range dl.incumbents {
		// DockCenter can never be an incumbent and will be ignored if specified as incumbent
		if incumbent == DockCenter {
			continue
		}
		for _, iobj := range pan.Children() {
			child := iobj.(IPanel).GetPanel()
			if child.layoutParams == nil {
				continue
			}
			params := child.layoutParams.(*DockLayoutParams)
			if params.Edge == incumbent {
				if incumbent == DockTop {
					child.SetPosition(x, y)
					y += child.Height()
					h -= child.Height()
					child.SetWidth(w)
				}
				if incumbent == DockBottom {
					h -= child.Height()
					child.SetPosition(x, y+h)
					child.SetWidth(w)
				}
				if incumbent == DockLeft {
					child.SetPosition(x, y)
					x += child.Width()
					w -= child.Width()
					child.SetHeight(h)
				}
				if incumbent == DockRight {
					w -= child.Width()
					child.SetPosition(x+w, y)
					child.SetHeight(h)
				}
			}
		}
	}

	// Center (only the first found)
	for _, iobj := range pan.Children() {
		child := iobj.(IPanel).GetPanel()
		if child.layoutParams == nil {
			continue
		}
		params := child.layoutParams.(*DockLayoutParams)
		if params.Edge == DockCenter {
			child.SetPosition(x, y)
			child.SetSize(w, h)
			break
		}
	}
}

func contains(edges []Edge, edge Edge) bool {
	for _, e := range edges {
		if e == edge {
			return true
		}
	}
	return false
}
