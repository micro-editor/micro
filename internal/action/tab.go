package action

import (
	luar "layeh.com/gopher-luar"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/views"
	"github.com/micro-editor/tcell/v2"
)

// The TabList is a list of tabs and a window to display the tab bar
// at the top of the screen
type TabList struct {
	*display.TabWindow
	List []*Tab
}

// NewTabList creates a TabList from a list of buffers by creating a Tab
// for each buffer
func NewTabList(bufs []*buffer.Buffer) *TabList {
	w, h := screen.Screen.Size()
	iOffset := config.GetInfoBarOffset()
	tl := new(TabList)
	tl.List = make([]*Tab, len(bufs))
	if len(bufs) > 1 {
		for i, b := range bufs {
			tl.List[i] = NewTabFromBuffer(0, 1, w, h-1-iOffset, b)
		}
	} else {
		tl.List[0] = NewTabFromBuffer(0, 0, w, h-iOffset, bufs[0])
	}
	tl.TabWindow = display.NewTabWindow(w, 0)
	tl.Names = make([]string, len(bufs))

	return tl
}

// UpdateNames makes sure that the list of names the tab window has access to is
// correct
func (t *TabList) UpdateNames() {
	t.Names = t.Names[:0]
	for _, p := range t.List {
		t.Names = append(t.Names, p.Panes[p.active].Name())
	}
}

// AddTab adds a new tab to this TabList
func (t *TabList) AddTab(p *Tab) {
	t.List = append(t.List, p)
	t.Resize()
	t.UpdateNames()
}

// RemoveTab removes a tab with the given id from the TabList
func (t *TabList) RemoveTab(id uint64) {
	for i, p := range t.List {
		if len(p.Panes) == 0 {
			continue
		}
		if p.Panes[0].ID() == id {
			copy(t.List[i:], t.List[i+1:])
			t.List[len(t.List)-1] = nil
			t.List = t.List[:len(t.List)-1]
			if t.Active() >= len(t.List) {
				t.SetActive(len(t.List) - 1)
			}
			t.Resize()
			t.UpdateNames()
			return
		}
	}
}

// Resize resizes all elements within the tab list
// One thing to note is that when there is only 1 tab
// the tab bar should not be drawn so resizing must take
// that into account
func (t *TabList) Resize() {
	w, h := screen.Screen.Size()
	iOffset := config.GetInfoBarOffset()
	InfoBar.Resize(w, h-1)
	globalTabAlwaysShow := config.GetGlobalOption("tabalwaysshow").(bool)
	if len(t.List) > 1 || globalTabAlwaysShow {
		for _, p := range t.List {
			p.Y = 1
			p.Node.Resize(w, h-1-iOffset)
			p.Resize()
		}
	} else if len(t.List) == 1 {
		t.List[0].Y = 0
		t.List[0].Node.Resize(w, h-iOffset)
		t.List[0].Resize()
	}
	t.TabWindow.Resize(w, h)
}

// HandleEvent checks for a resize event or a mouse event on the tab bar
// otherwise it will forward the event to the currently active tab
func (t *TabList) HandleEvent(event tcell.Event) {
	switch e := event.(type) {
	case *tcell.EventResize:
		t.Resize()
	case *tcell.EventMouse:
		mx, my := e.Position()
		switch e.Buttons() {
		case tcell.Button1:
			if my == t.Y && len(t.List) > 1 {
				if mx == 0 {
					t.Scroll(-4)
				} else if mx == t.Width-1 {
					t.Scroll(4)
				} else {
					ind := t.LocFromVisual(buffer.Loc{mx, my})
					if ind != -1 {
						t.SetActive(ind)
					}
				}
				return
			}
		case tcell.ButtonNone:
			if t.List[t.Active()].release {
				// Mouse release received, while already released
				t.ResetMouse()
				return
			}
		case tcell.WheelUp:
			if my == t.Y && len(t.List) > 1 {
				t.Scroll(4)
				return
			}
		case tcell.WheelDown:
			if my == t.Y && len(t.List) > 1 {
				t.Scroll(-4)
				return
			}
		}
	}
	t.List[t.Active()].HandleEvent(event)
}

// Display updates the names and then displays the tab bar
func (t *TabList) Display() {
	t.UpdateNames()
	globalTabAlwaysShow := config.GetGlobalOption("tabalwaysshow").(bool)
	if len(t.List) > 1 || globalTabAlwaysShow {
		t.TabWindow.Display()
	}
}

func (t *TabList) SetActive(a int) {
	t.TabWindow.SetActive(a)

	for i, p := range t.List {
		if i == a {
			if !p.isActive {
				p.isActive = true

				err := config.RunPluginFn("onSetActive", luar.New(ulua.L, p.CurPane()))
				if err != nil {
					screen.TermMessage(err)
				}
			}
		} else {
			p.isActive = false
		}
	}
}

// ResetMouse resets the mouse release state after the screen was stopped
// or the pane changed.
// This prevents situations in which mouse releases are received at the wrong place
// and the mouse state is still pressed.
func (t *TabList) ResetMouse() {
	for _, tab := range t.List {
		if !tab.release && tab.resizing != nil {
			tab.resizing = nil
		}

		tab.release = true

		for _, p := range tab.Panes {
			if bp, ok := p.(*BufPane); ok {
				bp.resetMouse()
			}
		}
	}
}

// CloseTerms notifies term panes that a terminal job has finished.
func (t *TabList) CloseTerms() {
	for _, tab := range t.List {
		for _, p := range tab.Panes {
			if tp, ok := p.(*TermPane); ok {
				tp.HandleTermClose()
			}
		}
	}
}

// Tabs is the global tab list
var Tabs *TabList

func InitTabs(bufs []*buffer.Buffer) {
	multiopen := config.GetGlobalOption("multiopen").(string)
	if multiopen == "tab" {
		Tabs = NewTabList(bufs)
	} else {
		Tabs = NewTabList(bufs[:1])
		for _, b := range bufs[1:] {
			if multiopen == "vsplit" {
				MainTab().CurPane().VSplitBuf(b)
			} else { // default hsplit
				MainTab().CurPane().HSplitBuf(b)
			}
		}
	}

	screen.RestartCallback = Tabs.ResetMouse
}

func MainTab() *Tab {
	return Tabs.List[Tabs.Active()]
}

// A Tab represents a single tab
// It consists of a list of edit panes (the open buffers),
// a split tree (stored as just the root node), and a uiwindow
// to display the UI elements like the borders between splits
type Tab struct {
	*views.Node
	*display.UIWindow

	isActive bool

	Panes  []Pane
	active int

	resizing *views.Node // node currently being resized
	// captures whether the mouse is released
	release bool
}

// NewTabFromBuffer creates a new tab from the given buffer
func NewTabFromBuffer(x, y, width, height int, b *buffer.Buffer) *Tab {
	t := new(Tab)
	t.Node = views.NewRoot(x, y, width, height)
	t.UIWindow = display.NewUIWindow(t.Node)
	t.release = true

	e := NewBufPaneFromBuf(b, t)
	e.SetID(t.ID())

	t.Panes = append(t.Panes, e)
	return t
}

func NewTabFromPane(x, y, width, height int, pane Pane) *Tab {
	t := new(Tab)
	t.Node = views.NewRoot(x, y, width, height)
	t.UIWindow = display.NewUIWindow(t.Node)
	t.release = true
	pane.SetTab(t)
	pane.SetID(t.ID())

	t.Panes = append(t.Panes, pane)
	return t
}

// HandleEvent takes a tcell event and usually dispatches it to the current
// active pane. However if the event is a resize or a mouse event where the user
// is interacting with the UI (resizing splits) then the event is consumed here
// If the event is a mouse press event in a pane, that pane will become active
// and get the event
func (t *Tab) HandleEvent(event tcell.Event) {
	switch e := event.(type) {
	case *tcell.EventMouse:
		mx, my := e.Position()
		btn := e.Buttons()
		switch {
		case btn & ^(tcell.WheelUp|tcell.WheelDown|tcell.WheelLeft|tcell.WheelRight) != tcell.ButtonNone:
			// button press or drag
			wasReleased := t.release
			t.release = false

			if btn == tcell.Button1 {
				if t.resizing != nil {
					var size int
					if t.resizing.Kind == views.STVert {
						size = mx - t.resizing.X
					} else {
						size = my - t.resizing.Y + 1
					}
					t.resizing.ResizeSplit(size)
					t.Resize()
					return
				}
				if wasReleased {
					t.resizing = t.GetMouseSplitNode(buffer.Loc{mx, my})
					if t.resizing != nil {
						return
					}
				}
			}

			if wasReleased {
				for i, p := range t.Panes {
					v := p.GetView()
					inpane := mx >= v.X && mx < v.X+v.Width && my >= v.Y && my < v.Y+v.Height
					if inpane {
						t.SetActive(i)
						break
					}
				}
			}
		case btn == tcell.ButtonNone:
			// button release
			t.release = true
			if t.resizing != nil {
				t.resizing = nil
				return
			}
		default:
			// wheel move
			for _, p := range t.Panes {
				v := p.GetView()
				inpane := mx >= v.X && mx < v.X+v.Width && my >= v.Y && my < v.Y+v.Height
				if inpane {
					p.HandleEvent(event)
					return
				}
			}
		}

	}
	t.Panes[t.active].HandleEvent(event)
}

// SetActive changes the currently active pane to the specified index
func (t *Tab) SetActive(i int) {
	t.active = i
	for j, p := range t.Panes {
		if j == i {
			p.SetActive(true)
		} else {
			p.SetActive(false)
		}
	}
}

// AddPane adds a pane at a given index
func (t *Tab) AddPane(pane Pane, i int) {
	if len(t.Panes) == i {
		t.Panes = append(t.Panes, pane)
		return
	}
	t.Panes = append(t.Panes[:i+1], t.Panes[i:]...)
	t.Panes[i] = pane
}

// GetPane returns the pane with the given split index
func (t *Tab) GetPane(splitid uint64) int {
	for i, p := range t.Panes {
		if p.ID() == splitid {
			return i
		}
	}
	return 0
}

// Remove pane removes the pane with the given index
func (t *Tab) RemovePane(i int) {
	copy(t.Panes[i:], t.Panes[i+1:])
	t.Panes[len(t.Panes)-1] = nil
	t.Panes = t.Panes[:len(t.Panes)-1]
}

// Resize resizes all panes according to their corresponding split nodes
func (t *Tab) Resize() {
	for _, p := range t.Panes {
		n := t.GetNode(p.ID())
		pv := p.GetView()
		offset := 0
		if n.X != 0 {
			offset = 1
		}
		pv.X, pv.Y = n.X+offset, n.Y
		p.SetView(pv)
		p.Resize(n.W-offset, n.H)
	}
}

// CurPane returns the currently active pane
func (t *Tab) CurPane() *BufPane {
	p, ok := t.Panes[t.active].(*BufPane)
	if !ok {
		return nil
	}
	return p
}
