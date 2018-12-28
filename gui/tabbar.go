// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gui

import (
	"fmt"
	"github.com/sansebasko/engine/math32"
	"github.com/sansebasko/engine/window"
)

// TabBar is a panel which can contain other panels arranged in horizontal Tabs.
// Only one panel is visible at a time.
// To show another panel the corresponding Tab must be selected.
type TabBar struct {
	Panel                                  // Embedded panel
	styles                   *TabBarStyles // Pointer to current styles
	tabs                     []*Tab        // Array of tabs
	separator                Panel         // Separator Panel
	listButton               *Label        // Icon for tab list button
	list                     *List         // List for not visible tabs
	selected                 int           // Index of the selected tab
	cursorOver               bool          // Cursor over TabBar panel flag
	labelAlign               Align         // Label align of all tabs (one of AlignCenter, AlignLeft, AlignRight)
	tabHeaderAlign           Align         // Tab header align (one of AlignTop, AlignBottom)
	consistentTabHeaderWidth bool          // Consistent tab header width (true) or only as width as needed (false)
}

// TabBarStyle describes the style of the TabBar
type TabBarStyle BasicStyle

// TabBarStyles describes all the TabBarStyles
type TabBarStyles struct {
	SepHeight          float32     // Separator width
	ListButtonIcon     string      // Icon for list button
	ListButtonPaddings RectBounds  // Paddings for list button
	Normal             TabBarStyle // Style for normal exhibition
	Over               TabBarStyle // Style when cursor is over the TabBar
	Focus              TabBarStyle // Style when the TabBar has key focus
	Disabled           TabBarStyle // Style when the TabBar is disabled
	Tab                TabStyles   // Style for Tabs
}

// TabStyle describes the style of the individual Tabs header
type TabStyle BasicStyle

// TabStyles describes all Tab styles
type TabStyles struct {
	IconPaddings     RectBounds   // Paddings for optional icon
	ImagePaddings    RectBounds   // Paddings for optional image
	IconClose        string       // Codepoint for close icon in Tab header
	Normal           TabStyle     // Style for normal exhibition
	Over             TabStyle     // Style when cursor is over the Tab
	Focus            TabStyle     // Style when the Tab has key focus
	Disabled         TabStyle     // Style when the Tab is disabled
	Selected         TabStyle     // Style when the Tab is selected
	SelectionAdvance AdvanceStyle // Style of advance when the tab is selected
}

type AdvanceStyle struct {
	Thickness float32
	Color     math32.Color4
}

// NewTabBar creates and returns a pointer to a new TabBar widget
// with the specified width and height
func NewTabBar(width, height float32) *TabBar {

	// Creates new TabBar
	tb := new(TabBar)
	tb.labelAlign = AlignLeft
	tb.tabHeaderAlign = AlignTop
	tb.consistentTabHeaderWidth = true
	tb.Initialize(width, height)
	tb.styles = &StyleDefault().TabBar
	tb.tabs = make([]*Tab, 0)
	tb.selected = -1

	// Creates separator panel (between the tab headers and content panel)
	tb.separator.Initialize(0, 0)
	tb.Add(&tb.separator)

	// Create list for contained tabs not visible
	tb.list = NewVList(0, 0)
	tb.list.Subscribe(OnMouseOut, func(evname string, ev interface{}) {
		tb.list.SetVisible(false)
	})
	tb.list.Subscribe(OnChange, tb.onListChange)
	tb.Add(tb.list)

	// Creates list icon button
	tb.listButton = NewIcon(tb.styles.ListButtonIcon)
	tb.listButton.SetPaddingsFrom(&tb.styles.ListButtonPaddings)
	tb.listButton.Subscribe(OnMouseDown, tb.onListButton)
	tb.Add(tb.listButton)

	// Subscribe to panel events
	tb.Subscribe(OnCursorEnter, tb.onCursor)
	tb.Subscribe(OnCursorLeave, tb.onCursor)
	tb.Subscribe(OnEnable, func(name string, ev interface{}) { tb.update() })
	tb.Subscribe(OnResize, func(name string, ev interface{}) { tb.recalc() })

	tb.recalc()
	tb.update()
	return tb
}

// Styles returns the styles of this TabBar
func (tb *TabBar) Styles() *TabBarStyles {
	return tb.styles
}

// LabelAlign returns the align of all its tab labels
func (tb *TabBar) LabelAlign() Align {
	return tb.labelAlign
}

// SetLabelAlign sets the align for all its tab labels if either AlignCenter,
// AlignLeft or AlignRight is specified, in which case true is returned.
// Otherwise nothing happens and false is returned.
func (tb *TabBar) SetLabelAlign(align Align) bool {
	if align == AlignCenter || align == AlignLeft || align == AlignRight {
		tb.labelAlign = align
		return true
	}
	return false
}

// TabHeaderAlign returns the align of the tab headers
func (tb *TabBar) TabHeaderAlign() Align {
	return tb.tabHeaderAlign
}

// SetTabHeaderAlign sets the align for all its tab headers if either AlignTop
// or AlignBottom is specified, in which case true is returned.
// Otherwise nothing happens and false is returned.
func (tb *TabBar) SetTabHeaderAlign(align Align) bool {
	if align == AlignTop || align == AlignBottom {
		tb.tabHeaderAlign = align
		return true
	}
	return false
}

// ConsistentTabHeaderWidth returns true if all its tab headers share a consistent width.
// Otherwise returns false, i.e. each tab header is only as wide as needed.
func (tb *TabBar) ConsistentTabHeaderWidth() bool {
	return tb.consistentTabHeaderWidth
}

// SetConsistentTabHeaderWidth affects the width of all its tab headers.
func (tb *TabBar) SetConsistentTabHeaderWidth(flag bool) {
	tb.consistentTabHeaderWidth = flag
}

// AddTab creates and adds a new Tab panel with the specified header text
// at the end of this TabBar list of tabs.
// Returns the pointer to thew new Tab.
func (tb *TabBar) AddTab(text string) *Tab {

	tab := tb.InsertTab(text, len(tb.tabs))
	tb.SetSelected(len(tb.tabs) - 1)
	return tab
}

// InsertTab creates and inserts a new Tab panel with the specified header text
// at the specified position in the TabBar from left to right.
// Returns the pointer to the new Tab or nil if the position is invalid.
func (tb *TabBar) InsertTab(text string, pos int) *Tab {

	// Checks position to insert into
	if pos < 0 || pos > len(tb.tabs) {
		return nil
	}

	// Inserts created Tab at the specified position
	tab := newTab(text, tb, &tb.styles.Tab)
	tb.tabs = append(tb.tabs, nil)
	copy(tb.tabs[pos+1:], tb.tabs[pos:])
	tb.tabs[pos] = tab
	tb.Add(&tab.header)

	tb.update()
	tb.recalc()
	return tab
}

// RemoveTab removes the tab at the specified position in the TabBar.
// Returns an error if the position is invalid.
func (tb *TabBar) RemoveTab(pos int) error {

	// Check position to remove from
	if pos < 0 || pos >= len(tb.tabs) {
		return fmt.Errorf("Invalid tab position:%d", pos)
	}

	// Remove tab from TabBar panel
	tab := tb.tabs[pos]
	tb.Remove(&tab.header)
	if tab.content != nil {
		tb.Remove(tab.content)
	}

	// Remove tab from tabbar array
	copy(tb.tabs[pos:], tb.tabs[pos+1:])
	tb.tabs[len(tb.tabs)-1] = nil
	tb.tabs = tb.tabs[:len(tb.tabs)-1]

	// If removed tab was selected, selects other tab.
	if tb.selected == pos {
		// Try to select tab at right
		if len(tb.tabs) > pos {
			tb.tabs[pos].setSelected(true)
			// Otherwise select tab at left
		} else if pos > 0 {
			tb.tabs[pos-1].setSelected(true)
		}
	}

	tb.update()
	tb.recalc()
	return nil
}

// MoveTab moves a Tab to another position in the Tabs list
func (tb *TabBar) MoveTab(src, dest int) error {

	// Check source position
	if src < 0 || src >= len(tb.tabs) {
		return fmt.Errorf("Invalid tab source position:%d", src)
	}
	// Check destination position
	if dest < 0 || dest >= len(tb.tabs) {
		return fmt.Errorf("Invalid tab destination position:%d", dest)
	}
	if src == dest {
		return nil
	}

	tabDest := tb.tabs[dest]
	tb.tabs[dest] = tb.tabs[src]
	tb.tabs[src] = tabDest
	tb.recalc()
	return nil
}

// TabCount returns the current number of Tabs in the TabBar
func (tb *TabBar) TabCount() int {

	return len(tb.tabs)
}

// TabAt returns the pointer of the Tab object at the specified position.
// Return nil if the position is invalid
func (tb *TabBar) TabAt(pos int) *Tab {

	if pos < 0 || pos >= len(tb.tabs) {
		return nil
	}
	return tb.tabs[pos]
}

// TabPosition returns the position of the Tab specified by its pointer
func (tb *TabBar) TabPosition(tab *Tab) int {

	for i := 0; i < len(tb.tabs); i++ {
		if tb.tabs[i] == tab {
			return i
		}
	}
	return -1
}

// SetSelected sets the selected tab of the TabBar to the tab with the specified position.
// Returns the pointer of the selected tab or nil if the position is invalid.
func (tb *TabBar) SetSelected(pos int) *Tab {

	if pos < 0 || pos >= len(tb.tabs) {
		return nil
	}
	for i := 0; i < len(tb.tabs); i++ {
		if i == pos {
			tb.tabs[i].setSelected(true)
		} else {
			tb.tabs[i].setSelected(false)
		}
	}
	tb.selected = pos
	return tb.tabs[pos]
}

// Selected returns the position of the selected Tab.
// Returns value < 0 if there is no selected Tab.
func (tb *TabBar) Selected() int {

	return tb.selected
}

// onCursor process subscribed cursor events
func (tb *TabBar) onCursor(evname string, ev interface{}) {

	switch evname {
	case OnCursorEnter:
		tb.cursorOver = true
		tb.update()
	case OnCursorLeave:
		tb.cursorOver = false
		tb.update()
	default:
		return
	}
	tb.root.StopPropagation(StopAll)
}

// onListButtonMouse process subscribed MouseButton events over the list button
func (tb *TabBar) onListButton(evname string, ev interface{}) {

	switch evname {
	case OnMouseDown:
		if !tb.list.Visible() {
			tb.list.SetVisible(true)
		}
	default:
		return
	}
	tb.root.StopPropagation(StopAll)
}

// onListChange process OnChange event from the tab list
func (tb *TabBar) onListChange(evname string, ev interface{}) {

	selected := tb.list.Selected()
	pos := selected[0].GetPanel().UserData().(int)
	log.Error("onListChange:%v", pos)
	tb.SetSelected(pos)
	tb.list.SetVisible(false)
}

// applyStyle applies the specified TabBar style
func (tb *TabBar) applyStyle(s *TabBarStyle) {

	tb.Panel.ApplyStyle(&s.PanelStyle)
	tb.separator.SetColor4(&s.BorderColor)
}

// recalc recalculates and updates the positions of all tabs
func (tb *TabBar) recalc() {

	// Determines how many tabs could be fully shown
	iconWidth := tb.listButton.Width()
	availWidth := tb.ContentWidth() - iconWidth
	var tabWidth float32
	var totalWidth float32
	var count int
	for i := 0; i < len(tb.tabs); i++ {
		tab := tb.tabs[i]
		minw := tab.minWidth()
		if minw > tabWidth {
			tabWidth = minw
		}
		totalWidth = float32(count+1) * tabWidth
		if totalWidth > availWidth {
			break
		}
		count++
	}

	// If there are more Tabs that can be shown, shows list button
	if count < len(tb.tabs) {
		// Sets the list button visible
		tb.listButton.SetVisible(true)
		height := tb.tabs[0].header.Height()
		iy := (height - tb.listButton.Height()) / 2
		tb.listButton.SetPosition(availWidth, iy)
		// Sets the tab list position and size
		listWidth := float32(200)
		lx := tb.ContentWidth() - listWidth
		ly := height + 1
		tb.list.SetPosition(lx, ly)
		tb.list.SetSize(listWidth, 200)
		tb.SetTopChild(tb.list)
	} else {
		tb.listButton.SetVisible(false)
		tb.list.SetVisible(false)
	}

	tb.list.Clear()
	var headerx float32
	// When there is available space limits the with of the tabs
	maxTabWidth := availWidth / float32(count)
	if tabWidth < maxTabWidth {
		tabWidth += (maxTabWidth - tabWidth) / 4
	}
	sepHeight := tb.styles.SepHeight

	for i := 0; i < len(tb.tabs); i++ {
		tab := tb.tabs[i]
		// Recalculate Tab header and sets its position
		tab.recalc(tabWidth)

		headery := float32(0)
		if tb.tabHeaderAlign == AlignBottom {
			headery = tb.ContentHeight() - tab.header.Height()
		}
		tab.header.SetPosition(headerx, headery)
		headerx += tab.header.Width()

		// Sets size and position of the Tab content panel
		if tab.content != nil {
			cpan := tab.content.GetPanel()
			cpan.SetWidth(tb.ContentWidth())
			if tb.tabHeaderAlign == AlignTop {
				cpan.SetPosition(0, tab.header.Height()+sepHeight)
				cpan.SetHeight(tb.ContentHeight() - cpan.Position().Y)
			} else {
				cpan.SetPosition(0, 0)
				cpan.SetHeight(headery)
			}
		}
		// If Tab can be shown set its header visible
		if i < count {
			tab.header.SetVisible(true)
			// Otherwise insert tab text in List
		} else {
			tab.header.SetVisible(false)
			item := NewImageLabel(tab.Label())
			item.SetUserData(i)
			tb.list.Add(item)
		}
	}

	// Sets the separator size, position and visibility
	if len(tb.tabs) > 0 {
		tb.separator.SetSize(tb.ContentWidth(), sepHeight)
		if tb.tabHeaderAlign == AlignTop {
			tb.separator.SetPositionY(tb.tabs[0].header.Height())
		} else {
			tb.separator.SetPositionY(tb.ContentHeight() - tb.tabs[0].header.Height() - sepHeight)
		}
		tb.separator.SetVisible(true)
	} else {
		tb.separator.SetVisible(false)
	}
}

// update updates the TabBar visual style
func (tb *TabBar) update() {

	if tb.tabHeaderAlign == AlignBottom {
		defer func() {
			tb.paddingSizes.Top, tb.paddingSizes.Bottom = tb.paddingSizes.Bottom, tb.paddingSizes.Top
		}()
	}

	if !tb.Enabled() {
		tb.applyStyle(&tb.styles.Disabled)
		return
	}
	if tb.cursorOver {
		tb.applyStyle(&tb.styles.Over)
		return
	}
	tb.applyStyle(&tb.styles.Normal)
}

//
// Tab describes an individual tab of the TabBar
//
type Tab struct {
	tb         *TabBar    // Pointer to parent *TabBar
	styles     *TabStyles // Pointer to Tab current styles
	header     Panel      // Tab header
	label      *Label     // Tab user label
	labelAlign Align      // Tab user label align
	iconClose  *Label     // Tab close icon
	icon       *Label     // Tab optional user icon
	image      *Image     // Tab optional user image
	cover      Panel      // Panel to cover the bottom/top edge of the tab
	advance    Panel      // Panel of the advance of the selected tab
	content    IPanel     // User content panel
	cursorOver bool
	selected   bool
	pinned     bool
}

// newTab creates and returns a pointer to a new Tab
func newTab(text string, tb *TabBar, styles *TabStyles) *Tab {

	tab := new(Tab)
	tab.tb = tb
	tab.styles = styles
	// Setup the header panel
	tab.header.Initialize(0, 0)
	tab.label = NewLabel(text)
	tab.labelAlign = tab.tb.labelAlign
	tab.iconClose = NewIcon(styles.IconClose)
	tab.header.Add(tab.label)
	tab.header.Add(tab.iconClose)
	// Creates the cover panel
	tab.cover.Initialize(0, 0)
	tab.cover.SetBounded(false)
	tab.cover.SetColor4(&tab.styles.Selected.BgColor)
	tab.header.Add(&tab.cover)
	// Creates the advance panel
	tab.advance.Initialize(0, 0)
	tab.advance.SetBounded(false)
	tab.advance.SetColor4(&tab.styles.SelectionAdvance.Color)
	tab.header.Add(&tab.advance)

	// Subscribe to header panel events
	tab.header.Subscribe(OnCursorEnter, tab.onCursor)
	tab.header.Subscribe(OnCursorLeave, tab.onCursor)
	tab.header.Subscribe(OnMouseDown, tab.onMouseHeader)
	tab.iconClose.Subscribe(OnMouseDown, tab.onMouseIcon)

	tab.update()
	return tab
}

// Label returns the text of the tab label
func (tab *Tab) Label() string {
	return tab.label.Text()
}

// SetLabel sets the text of the tab label
func (tab *Tab) SetLabel(text string) {
	tab.label.SetText(text)
	tab.tb.recalc()
}

// LabelAlign returns the align of the tab label
func (tab *Tab) LabelAlign() Align {
	return tab.labelAlign
}

// SetLabelAlign sets the align of the tab label if either AlignCenter,
// AlignLeft or AlignRight is specified, in which case true is returned.
// Otherwise nothing happens and false is returned.
func (tab *Tab) SetLabelAlign(align Align) bool {
	if align == AlignCenter || align == AlignLeft || align == AlignRight {
		tab.labelAlign = align
		return true
	}
	return false
}

// onCursor process subscribed cursor events over the tab header
func (tab *Tab) onCursor(evname string, ev interface{}) {

	switch evname {
	case OnCursorEnter:
		tab.cursorOver = true
		tab.update()
	case OnCursorLeave:
		tab.cursorOver = false
		tab.update()
	default:
		return
	}
	tab.header.root.StopPropagation(StopAll)
}

// onMouse process subscribed mouse events over the tab header
func (tab *Tab) onMouseHeader(evname string, ev interface{}) {

	switch evname {
	case OnMouseDown:
		mev := ev.(*window.MouseEvent)
		if mev.Button == window.MouseButtonLeft {
			tab.tb.SetSelected(tab.tb.TabPosition(tab))
		} else {
			tab.header.Dispatch(OnRightClick, ev)
		}
	default:
		return
	}
	tab.header.root.StopPropagation(StopAll)
}

// onMouseIcon process subscribed mouse events over the tab close icon
func (tab *Tab) onMouseIcon(evname string, ev interface{}) {

	switch evname {
	case OnMouseDown:
		_ = tab.tb.RemoveTab(tab.tb.TabPosition(tab))
	default:
		return
	}
	tab.header.root.StopPropagation(StopAll)
}

// SetText sets the text of the tab header
func (tab *Tab) SetText(text string) *Tab {

	tab.label.SetText(text)
	// Needs to recalculate all Tabs because this Tab width will change
	tab.tb.recalc()
	return tab
}

// SetIcon sets the optional icon of the Tab header
func (tab *Tab) SetIcon(icon string) *Tab {

	// Remove previous header image if any
	if tab.image != nil {
		tab.header.Remove(tab.image)
		tab.image.Dispose()
		tab.image = nil
	}
	// Creates or updates icon
	if tab.icon == nil {
		tab.icon = NewIcon(icon)
		tab.icon.SetPaddingsFrom(&tab.styles.IconPaddings)
		tab.header.Add(tab.icon)
	} else {
		tab.icon.SetText(icon)
	}
	// Needs to recalculate all Tabs because this Tab width will change
	tab.tb.recalc()
	return tab
}

// SetImage sets the optional image of the Tab header
func (tab *Tab) SetImage(imgfile string) error {

	// Remove previous icon if any
	if tab.icon != nil {
		tab.header.Remove(tab.icon)
		tab.icon.Dispose()
		tab.icon = nil
	}
	// Creates or updates image
	if tab.image == nil {
		// Creates image panel from file
		img, err := NewImage(imgfile)
		if err != nil {
			return err
		}
		tab.image = img
		tab.image.SetPaddingsFrom(&tab.styles.ImagePaddings)
		tab.header.Add(tab.image)
	} else {
		err := tab.image.SetImage(imgfile)
		if err != nil {
			return err
		}
	}
	// Scale image so its height is not greater than the Label height
	if tab.image.Height() > tab.label.Height() {
		tab.image.SetContentAspectHeight(tab.label.Height())
	}
	// Needs to recalculate all Tabs because this Tab width will change
	tab.tb.recalc()
	return nil
}

// SetPinned sets the tab pinned state.
// A pinned tab cannot be removed by the user because the close icon is not shown.
func (tab *Tab) SetPinned(pinned bool) {

	tab.pinned = pinned
	tab.iconClose.SetVisible(!pinned)
}

// Pinned returns this tab pinned state
func (tab *Tab) Pinned() bool {

	return tab.pinned
}

// Header returns a pointer to this Tab header panel.
// Can be used to set an event handler when the Tab header is right clicked.
// (to show a context Menu for example).
func (tab *Tab) Header() *Panel {

	return &tab.header
}

// SetContent sets or replaces this tab content panel.
func (tab *Tab) SetContent(ipan IPanel) {

	// Remove previous content if any
	if tab.content != nil {
		tab.tb.Remove(tab.content)
	}
	tab.content = ipan
	if ipan != nil {
		tab.tb.Add(tab.content)
	}
	tab.tb.recalc()
}

// Content returns a pointer to the specified Tab content panel
func (tab *Tab) Content() IPanel {

	return tab.content
}

// setSelected sets this Tab selected state
func (tab *Tab) setSelected(selected bool) {

	tab.selected = selected
	if tab.content != nil {
		tab.content.GetPanel().SetVisible(selected)
	}
	tab.cover.SetVisible(selected)
	tab.advance.SetVisible(selected)
	tab.update()
	tab.setCoverPanel()
	tab.setAdvancePanel()
}

// minWidth returns the minimum width of this Tab header to allow
// all of its elements to be shown in full.
func (tab *Tab) minWidth() float32 {

	var minWidth float32
	if tab.icon != nil {
		minWidth = tab.icon.Width()
	} else if tab.image != nil {
		minWidth = tab.image.Width()
	}
	minWidth += tab.label.Width()
	minWidth += tab.iconClose.Width()
	return minWidth + tab.header.MinWidth()
}

// applyStyle applies the specified Tab style to the Tab header
func (tab *Tab) applyStyle(s *TabStyle) {

	tab.header.ApplyStyle(&s.PanelStyle)
}

// update updates the Tab header visual style
func (tab *Tab) update() {

	if tab.tb.tabHeaderAlign == AlignBottom {
		defer func() {
			tab.header.borderSizes.Top, tab.header.borderSizes.Bottom = tab.header.borderSizes.Bottom, tab.header.borderSizes.Top
		}()
	}

	if !tab.header.Enabled() {
		tab.applyStyle(&tab.styles.Disabled)
		return
	}
	if tab.selected {
		tab.applyStyle(&tab.styles.Selected)
		return
	}
	if tab.cursorOver {
		tab.applyStyle(&tab.styles.Over)
		return
	}
	tab.applyStyle(&tab.styles.Normal)
}

// setCoverPanel sets the position and size of the Tab cover panel
// to cover the Tabs separator
func (tab *Tab) setCoverPanel() {

	if tab.selected {
		w := tab.header.ContentWidth() + tab.header.Paddings().Left + tab.header.Paddings().Right
		tab.cover.SetSize(w, tab.tb.styles.SepHeight)
		x := tab.styles.Selected.Margin.Left + tab.styles.Selected.Border.Left
		y := tab.header.Height()
		if tab.tb.tabHeaderAlign == AlignBottom {
			y = -tab.tb.styles.SepHeight
		}
		tab.cover.SetPosition(x, y)
	}
}

// setAdvancePanel sets the position and size of the Tab advance panel
// to advance the selected tab
func (tab *Tab) setAdvancePanel() {

	advance := tab.styles.SelectionAdvance.Thickness
	if tab.selected {
		w := tab.header.ContentWidth() + tab.header.Paddings().Left + tab.header.Paddings().Right
		tab.advance.SetSize(w, advance)
		x := tab.header.Margins().Left + tab.header.Borders().Left
		y := tab.header.Height() - advance - tab.header.Borders().Bottom
		if tab.tb.tabHeaderAlign == AlignTop {
			y = tab.header.Margins().Top + tab.header.Borders().Top
		}
		tab.advance.SetPosition(x, y)
	} else {
		if tab.tb.tabHeaderAlign == AlignBottom {
			tab.header.marginSizes.Bottom = advance
		} else {
			tab.header.marginSizes.Top = advance
		}
	}
}

// recalc recalculates the size of the Tab header and the size
// and positions of the Tab header internal panels
func (tab *Tab) recalc(width float32) {

	advance := tab.styles.SelectionAdvance.Thickness
	height := tab.label.Height()
	if tab.selected {
		height += advance
	}
	tab.header.SetContentHeight(height)

	labx := float32(0)
	lw := tab.label.ContentWidth()
	thw := lw + 24
	if tab.icon != nil {
		icy := (tab.header.ContentHeight() - tab.icon.Height()) / 2
		tab.icon.SetPosition(0, icy)
		labx = tab.icon.Width()
		thw += tab.icon.Width()
	} else if tab.image != nil {
		tab.image.SetPosition(0, 0)
		labx = tab.image.Width()
		thw += tab.image.Width()
	}

	if tab.tb.consistentTabHeaderWidth {
		thw = width
	}
	tab.header.SetContentWidth(thw)

	icw := float32(0)
	if tab.iconClose.Visible() {
		icw = tab.iconClose.Width()
	}
	if tab.labelAlign == AlignCenter {
		labx = (thw - icw - lw - labx) / 2
	} else if tab.labelAlign == AlignRight {
		labx = thw - lw
		if tab.iconClose.Visible() {
			labx -= icw
		}
	}
	laby := float32(0)
	if tab.selected && tab.tb.tabHeaderAlign == AlignTop {
		laby = advance
	}
	tab.label.SetPosition(labx, laby)

	// Sets the close icon position
	icx := thw - tab.iconClose.Width()
	icy := (tab.header.ContentHeight() - tab.iconClose.Height()) / 2
	if tab.selected {
		if tab.tb.tabHeaderAlign == AlignTop {
			icy += advance / 2
		} else {
			icy -= advance / 2
		}
	}
	tab.iconClose.SetPosition(icx, icy)

	// Sets the position of the cover panel to cover separator
	tab.setCoverPanel()
	tab.setAdvancePanel()
}
