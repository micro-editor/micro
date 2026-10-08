package action

import (
	"strings"
	"time"

	luar "layeh.com/gopher-luar"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/micro-editor/tcell/v2"
	lua "github.com/yuin/gopher-lua"
)

type BufAction any

// BufKeyAction represents an action bound to a key.
type BufKeyAction func(*BufPane) bool

// BufMouseAction is an action that must be bound to a mouse event.
type BufMouseAction func(*BufPane, *tcell.EventMouse) bool

// BufBindings stores the bindings for the buffer pane type.
var BufBindings *KeyTree

// BufKeyActionGeneral makes a general pane action from a BufKeyAction.
func BufKeyActionGeneral(a BufKeyAction) PaneKeyAction {
	return func(p Pane) bool {
		return a(p.(*BufPane))
	}
}

// BufMouseActionGeneral makes a general pane mouse action from a BufKeyAction.
func BufMouseActionGeneral(a BufMouseAction) PaneMouseAction {
	return func(p Pane, me *tcell.EventMouse) bool {
		return a(p.(*BufPane), me)
	}
}

func init() {
	BufBindings = NewKeyTree()
}

// LuaAction makes an action from a lua function. It returns either a BufKeyAction
// or a BufMouseAction depending on the event type.
func LuaAction(fn string, k Event) BufAction {
	luaFn := strings.Split(fn, ".")
	if len(luaFn) <= 1 {
		return nil
	}
	plName, plFn := luaFn[0], luaFn[1]
	pl := config.FindPlugin(plName)
	if pl == nil {
		return nil
	}

	var action BufAction
	switch k.(type) {
	case KeyEvent, KeySequenceEvent, RawEvent:
		action = BufKeyAction(func(h *BufPane) bool {
			val, err := pl.Call(plFn, luar.New(ulua.L, h))
			if err != nil {
				screen.TermMessage(err)
			}
			if v, ok := val.(lua.LBool); !ok {
				return false
			} else {
				return bool(v)
			}
		})
	case MouseEvent:
		action = BufMouseAction(func(h *BufPane, te *tcell.EventMouse) bool {
			val, err := pl.Call(plFn, luar.New(ulua.L, h), luar.New(ulua.L, te))
			if err != nil {
				screen.TermMessage(err)
			}
			if v, ok := val.(lua.LBool); !ok {
				return false
			} else {
				return bool(v)
			}
		})
	}
	return action
}

// BufMapEvent maps an event to an action
func BufMapEvent(k Event, action string) {
	config.Bindings["buffer"][k.Name()] = action

	var actionfns []BufAction
	var names []string
	var types []byte
	for i := 0; ; i++ {
		if action == "" {
			break
		}

		idx := util.IndexAnyUnquoted(action, "&|,")
		a := action
		if idx >= 0 {
			a = action[:idx]
			types = append(types, action[idx])
			action = action[idx+1:]
		} else {
			types = append(types, ' ')
			action = ""
		}

		var afn BufAction
		if strings.HasPrefix(a, "command:") {
			a = strings.SplitN(a, ":", 2)[1]
			afn = CommandAction(a)
			names = append(names, "")
		} else if strings.HasPrefix(a, "command-edit:") {
			a = strings.SplitN(a, ":", 2)[1]
			afn = CommandEditAction(a)
			names = append(names, "")
		} else if strings.HasPrefix(a, "lua:") {
			a = strings.SplitN(a, ":", 2)[1]
			afn = LuaAction(a, k)
			if afn == nil {
				screen.TermMessage("Lua Error:", a, "does not exist")
				continue
			}
			split := strings.SplitN(a, ".", 2)
			if len(split) > 1 {
				a = strings.Title(split[0]) + strings.Title(split[1])
			} else {
				a = strings.Title(a)
			}

			names = append(names, a)
		} else if f, ok := BufKeyActions[a]; ok {
			afn = f
			names = append(names, a)
		} else if f, ok := BufMouseActions[a]; ok {
			afn = f
			names = append(names, a)
		} else {
			screen.TermMessage("Error in bindings: action", a, "does not exist")
			continue
		}
		actionfns = append(actionfns, afn)
	}
	bufAction := func(h *BufPane, te *tcell.EventMouse) bool {
		for i, a := range actionfns {
			var success bool
			if _, ok := MultiActions[names[i]]; ok {
				success = true
				for _, c := range h.Buf.GetCursors() {
					h.Buf.SetCurCursor(c.Num)
					h.Cursor = c
					success = success && h.execAction(a, names[i], te)
				}
			} else {
				h.Buf.SetCurCursor(0)
				h.Cursor = h.Buf.GetActiveCursor()
				success = h.execAction(a, names[i], te)
			}

			// if the action changed the current pane, update the reference
			h = MainTab().CurPane()
			if h == nil {
				// stop, in case the current pane is not a BufPane
				break
			}

			if (!success && types[i] == '&') || (success && types[i] == '|') {
				break
			}
		}
		return true
	}

	switch e := k.(type) {
	case KeyEvent, KeySequenceEvent, RawEvent:
		BufBindings.RegisterKeyBinding(e, BufKeyActionGeneral(func(h *BufPane) bool {
			return bufAction(h, nil)
		}))
	case MouseEvent:
		BufBindings.RegisterMouseBinding(e, BufMouseActionGeneral(bufAction))
	}
}

// BufUnmap unmaps a key or mouse event from any action
func BufUnmap(k Event) {
	// TODO
	// delete(BufKeyBindings, k)
	//
	// switch e := k.(type) {
	// case MouseEvent:
	// 	delete(BufMouseBindings, e)
	// }
}

// The BufPane connects the buffer and the window
// It provides a cursor (or multiple) and defines a set of actions
// that can be taken on the buffer
// The ActionHandler can access the window for necessary info about
// visual positions for mouse clicks and scrolling
type BufPane struct {
	display.BWindow

	// Buf is the buffer this BufPane views
	Buf *buffer.Buffer
	// Bindings stores the association of key events and actions
	bindings *KeyTree

	// Cursor is the currently active buffer cursor
	Cursor *buffer.Cursor

	// Since tcell doesn't differentiate between a mouse press event
	// and a mouse move event with button pressed (nor between a mouse
	// release event and a mouse move event with no buttons pressed),
	// we need to keep track of whether or not the mouse was previously
	// pressed, to determine mouse release and mouse drag events.
	// Moreover, since in case of a release event tcell doesn't tell us
	// which button was released, we need to keep track of which
	// (possibly multiple) buttons were pressed previously.
	mousePressed map[MouseEvent]bool

	// This stores when the last click was
	// This is useful for detecting double and triple clicks
	lastClickTime time.Time
	lastLoc       buffer.Loc

	// freshClip returns true if one or more lines have been cut to the clipboard
	// and have never been pasted yet.
	freshClip bool

	// Was the last mouse event actually a double click?
	// Useful for detecting triple clicks -- if a double click is detected
	// but the last mouse event was actually a double click, it's a triple click
	DoubleClick bool
	// Same here, just to keep track for mouse move events
	TripleClick bool

	// Should the current multiple cursor selection search based on word or
	// based on selection (false for selection, true for word)
	multiWord bool

	splitID uint64
	tab     *Tab

	// remember original location of a search in case the search is canceled
	searchOrig buffer.Loc

	// The pane may not yet be fully initialized after its creation
	// since we may not know the window geometry yet. In such case we finish
	// its initialization a bit later, after the initial resize.
	initialized bool
}

func newBufPane(buf *buffer.Buffer, win display.BWindow, tab *Tab) *BufPane {
	h := new(BufPane)
	h.Buf = buf
	h.BWindow = win
	h.tab = tab

	h.Cursor = h.Buf.GetActiveCursor()
	h.mousePressed = make(map[MouseEvent]bool)

	return h
}

// NewBufPane creates a new buffer pane with the given window.
func NewBufPane(buf *buffer.Buffer, win display.BWindow, tab *Tab) *BufPane {
	h := newBufPane(buf, win, tab)
	h.finishInitialize()
	return h
}

// NewBufPaneFromBuf constructs a new pane from the given buffer and automatically
// creates a buf window.
func NewBufPaneFromBuf(buf *buffer.Buffer, tab *Tab) *BufPane {
	w := display.NewBufWindow(0, 0, 0, 0, buf)
	h := newBufPane(buf, w, tab)
	// Postpone finishing initializing the pane until we know the actual geometry
	// of the buf window.
	return h
}

// TODO: make sure splitID and tab are set before finishInitialize is called
func (h *BufPane) finishInitialize() {
	h.initialRelocate()
	h.initialized = true

	err := config.RunPluginFn("onBufPaneOpen", luar.New(ulua.L, h))
	if err != nil {
		screen.TermMessage(err)
	}
}

// Resize resizes the pane
func (h *BufPane) Resize(width, height int) {
	h.BWindow.Resize(width, height)
	if !h.initialized {
		h.finishInitialize()
	}
}

// SetTab sets this pane's tab.
func (h *BufPane) SetTab(t *Tab) {
	h.tab = t
}

// Tab returns this pane's tab.
func (h *BufPane) Tab() *Tab {
	return h.tab
}

func (h *BufPane) ResizePane(size int) {
	n := h.tab.GetNode(h.splitID)
	n.ResizeSplit(size)
	h.tab.Resize()
}

// PluginCB calls all plugin callbacks with a certain name and displays an
// error if there is one and returns the aggregate boolean response.
// The bufpane is passed as the first argument to the callbacks,
// optional args are passed as the next arguments.
func (h *BufPane) PluginCB(cb string, args ...any) bool {
	largs := []lua.LValue{luar.New(ulua.L, h)}
	for _, a := range args {
		largs = append(largs, luar.New(ulua.L, a))
	}

	b, err := config.RunPluginFnBool(h.Buf.Settings, cb, largs...)
	if err != nil {
		screen.TermMessage(err)
	}
	return b
}

func (h *BufPane) resetMouse() {
	for me := range h.mousePressed {
		delete(h.mousePressed, me)
	}
}

// OpenBuffer opens the given buffer in this pane.
func (h *BufPane) OpenBuffer(b *buffer.Buffer) {
	h.Buf.Close()
	h.Buf = b
	h.BWindow.SetBuffer(b)
	h.Cursor = b.GetActiveCursor()
	h.Resize(h.GetView().Width, h.GetView().Height)
	h.initialRelocate()
	// Set mouseReleased to true because we assume the mouse is not being
	// pressed when the editor is opened
	h.resetMouse()
	h.lastClickTime = time.Time{}
}

// GotoLoc moves the cursor to a new location and adjusts the view accordingly.
// Use GotoLoc when the new location may be far away from the current location.
func (h *BufPane) GotoLoc(loc buffer.Loc) {
	sloc := h.SLocFromLoc(loc)
	d := h.Diff(h.SLocFromLoc(h.Cursor.Loc), sloc)

	h.Cursor.GotoLoc(loc)

	// If the new location is far away from the previous one,
	// ensure the cursor is at 25% of the window height
	height := h.BufView().Height
	if util.Abs(d) >= height {
		v := h.GetView()
		v.StartLine = h.Scroll(sloc, -height/4)
		h.ScrollAdjust()
		v.StartCol = 0
	}
	h.Relocate()
}

func (h *BufPane) initialRelocate() {
	sloc := h.SLocFromLoc(h.Cursor.Loc)
	height := h.BufView().Height

	// If the initial cursor location is far away from the beginning
	// of the buffer, ensure the cursor is at 25% of the window height
	v := h.GetView()
	if h.Diff(display.SLoc{0, 0}, sloc) < height {
		v.StartLine = display.SLoc{0, 0}
	} else {
		v.StartLine = h.Scroll(sloc, -height/4)
		h.ScrollAdjust()
	}
	v.StartCol = 0
	h.Relocate()
}

// ID returns this pane's split id.
func (h *BufPane) ID() uint64 {
	return h.splitID
}

// SetID sets the split ID of this pane.
func (h *BufPane) SetID(i uint64) {
	h.splitID = i
}

// Name returns the BufPane's name.
func (h *BufPane) Name() string {
	n := h.Buf.GetName()
	if h.Buf.Modified() {
		n += " +"
	}
	return n
}

// ReOpen reloads the file opened in the bufpane from disk
func (h *BufPane) ReOpen() {
	h.Buf.ReOpen()
	h.Relocate()
}

func (h *BufPane) getReloadSetting() string {
	reloadSetting := h.Buf.Settings["reload"]
	return reloadSetting.(string)
}

// HandleEvent executes the tcell event properly
func (h *BufPane) HandleEvent(event tcell.Event) {
	if h.Buf.ExternallyModified() && !h.Buf.ReloadDisabled {
		reload := h.getReloadSetting()

		if reload == "prompt" {
			InfoBar.YNPrompt("The file on disk has changed. Reload file? (y,n,esc)", func(yes, canceled bool) {
				if canceled {
					h.Buf.DisableReload()
				}
				if !yes || canceled {
					h.Buf.UpdateModTime()
				} else {
					h.ReOpen()
				}
			})
		} else if reload == "auto" {
			h.ReOpen()
		} else if reload == "disabled" {
			h.Buf.DisableReload()
		} else {
			InfoBar.Message("Invalid reload setting")
		}
	}

	switch e := event.(type) {
	case *tcell.EventRaw:
		re := RawEvent{
			esc: e.EscSeq(),
		}
		h.DoKeyEvent(re)
	case *tcell.EventPaste:
		h.paste(e.Text())
		h.Relocate()
	case *tcell.EventKey:
		ke := keyEvent(e)

		done := h.DoKeyEvent(ke)
		if !done && e.Key() == tcell.KeyRune {
			h.DoRuneInsert(e.Rune())
		}
	case *tcell.EventMouse:
		if e.Buttons() != tcell.ButtonNone {
			me := MouseEvent{
				btn:   e.Buttons(),
				mod:   metaToAlt(e.Modifiers()),
				state: MousePress,
			}
			isDrag := len(h.mousePressed) > 0

			if e.Buttons() & ^(tcell.WheelUp|tcell.WheelDown|tcell.WheelLeft|tcell.WheelRight) != tcell.ButtonNone {
				h.mousePressed[me] = true
			}

			if isDrag {
				me.state = MouseDrag
			}
			h.DoMouseEvent(me, e)
		} else {
			// Mouse event with no click - mouse was just released.
			// If there were multiple mouse buttons pressed, we don't know which one
			// was actually released, so we assume they all were released.
			pressed := len(h.mousePressed) > 0
			for me := range h.mousePressed {
				delete(h.mousePressed, me)

				me.state = MouseRelease
				h.DoMouseEvent(me, e)
			}
			if !pressed {
				// Propagate the mouse release in case the press wasn't for this BufPane
				Tabs.ResetMouse()
			}
		}
	}
	h.Buf.MergeCursors()

	if h.IsActive() {
		// Display any gutter messages for this line
		c := h.Buf.GetActiveCursor()
		none := true
		for _, m := range h.Buf.Messages {
			if c.Y == m.Start.Y || c.Y == m.End.Y {
				InfoBar.GutterMessage(m.Msg)
				none = false
				break
			}
		}
		if none && InfoBar.HasGutter {
			InfoBar.ClearGutter()
		}
	}

	cursors := h.Buf.GetCursors()
	for _, c := range cursors {
		if c.NewTrailingWsY != c.Y && (!c.HasSelection() ||
			(c.NewTrailingWsY != c.CurSelection[0].Y && c.NewTrailingWsY != c.CurSelection[1].Y)) {
			c.NewTrailingWsY = -1
		}
	}
}

// Bindings returns the current bindings tree for this buffer.
func (h *BufPane) Bindings() *KeyTree {
	if h.bindings != nil {
		return h.bindings
	}
	return BufBindings
}

// DoKeyEvent executes a key event by finding the action it is bound
// to and executing it (possibly multiple times for multiple cursors).
// Returns true if the action was executed OR if there are more keys
// remaining to process before executing an action (if this is a key
// sequence event). Returns false if no action found.
func (h *BufPane) DoKeyEvent(e Event) bool {
	binds := h.Bindings()
	action, more := binds.NextEvent(e, nil)
	if action != nil && !more {
		action(h)
		binds.ResetEvents()
		return true
	} else if action == nil && !more {
		binds.ResetEvents()
	}
	return more
}

func (h *BufPane) execAction(action BufAction, name string, te *tcell.EventMouse) bool {
	if name != "Autocomplete" && name != "CycleAutocompleteBack" {
		h.Buf.HasSuggestions = false
	}

	if !h.PluginCB("pre"+name, te) {
		return false
	}

	var success bool
	switch a := action.(type) {
	case BufKeyAction:
		success = a(h)
	case BufMouseAction:
		success = a(h, te)
	}
	success = success && h.PluginCB("on"+name, te)

	if _, ok := MultiActions[name]; ok {
		if recordingMacro {
			if name != "ToggleMacro" && name != "PlayMacro" {
				curmacro = append(curmacro, action)
			}
		}
	}

	return success
}

func (h *BufPane) completeAction(action string) {
	h.PluginCB("on" + action)
}

func (h *BufPane) HasKeyEvent(e Event) bool {
	// TODO
	return true
	// _, ok := BufKeyBindings[e]
	// return ok
}

// DoMouseEvent executes a mouse event by finding the action it is bound
// to and executing it
func (h *BufPane) DoMouseEvent(e MouseEvent, te *tcell.EventMouse) bool {
	binds := h.Bindings()
	action, _ := binds.NextEvent(e, te)
	if action != nil {
		action(h)
		binds.ResetEvents()
		return true
	}
	// TODO
	return false

	// if action, ok := BufMouseBindings[e]; ok {
	// 	if action(h, te) {
	// 		h.Relocate()
	// 	}
	// 	return true
	// } else if h.HasKeyEvent(e) {
	// 	return h.DoKeyEvent(e)
	// }
	// return false
}

// DoRuneInsert inserts a given rune into the current buffer
// (possibly multiple times for multiple cursors)
func (h *BufPane) DoRuneInsert(r rune) {
	cursors := h.Buf.GetCursors()
	for _, c := range cursors {
		// Insert a character
		h.Buf.SetCurCursor(c.Num)
		h.Cursor = c
		if !h.PluginCB("preRune", string(r)) {
			continue
		}
		if c.HasSelection() {
			c.DeleteSelection()
			c.ResetSelection()
		}

		if h.Buf.OverwriteMode {
			next := c.Loc
			next.X++
			h.Buf.Replace(c.Loc, next, string(r))
		} else {
			h.Buf.Insert(c.Loc, string(r))
		}
		if recordingMacro {
			curmacro = append(curmacro, r)
		}
		h.Relocate()
		h.PluginCB("onRune", string(r))
	}
}

// VSplitIndex opens the given buffer in a vertical split on the given side.
func (h *BufPane) VSplitIndex(buf *buffer.Buffer, right bool) *BufPane {
	e := NewBufPaneFromBuf(buf, h.tab)
	e.splitID = h.tab.GetNode(h.splitID).VSplit(right)
	currentPaneIdx := h.tab.GetPane(h.splitID)
	if right {
		currentPaneIdx++
	}
	h.tab.AddPane(e, currentPaneIdx)
	h.tab.Resize()
	h.tab.SetActive(currentPaneIdx)
	return e
}

// HSplitIndex opens the given buffer in a horizontal split on the given side.
func (h *BufPane) HSplitIndex(buf *buffer.Buffer, bottom bool) *BufPane {
	e := NewBufPaneFromBuf(buf, h.tab)
	e.splitID = h.tab.GetNode(h.splitID).HSplit(bottom)
	currentPaneIdx := h.tab.GetPane(h.splitID)
	if bottom {
		currentPaneIdx++
	}
	h.tab.AddPane(e, currentPaneIdx)
	h.tab.Resize()
	h.tab.SetActive(currentPaneIdx)
	return e
}

// VSplitBuf opens the given buffer in a new vertical split.
func (h *BufPane) VSplitBuf(buf *buffer.Buffer) *BufPane {
	return h.VSplitIndex(buf, h.Buf.Settings["splitright"].(bool))
}

// HSplitBuf opens the given buffer in a new horizontal split.
func (h *BufPane) HSplitBuf(buf *buffer.Buffer) *BufPane {
	return h.HSplitIndex(buf, h.Buf.Settings["splitbottom"].(bool))
}

// Close this pane.
func (h *BufPane) Close() {
	h.Buf.Close()
}

// SetActive marks this pane as active.
func (h *BufPane) SetActive(b bool) {
	if h.IsActive() == b {
		return
	}

	h.BWindow.SetActive(b)
	if b {
		// Display any gutter messages for this line
		c := h.Buf.GetActiveCursor()
		none := true
		for _, m := range h.Buf.Messages {
			if c.Y == m.Start.Y || c.Y == m.End.Y {
				InfoBar.GutterMessage(m.Msg)
				none = false
				break
			}
		}
		if none && InfoBar.HasGutter {
			InfoBar.ClearGutter()
		}

		err := config.RunPluginFn("onSetActive", luar.New(ulua.L, h))
		if err != nil {
			screen.TermMessage(err)
		}
	}
}

// BufKeyActions contains the list of all possible key actions the bufhandler could execute
var BufKeyActions = map[string]BufKeyAction{
	"CursorUp":                  (*BufPane).CursorUp,
	"CursorDown":                (*BufPane).CursorDown,
	"CursorPageUp":              (*BufPane).CursorPageUp,
	"CursorPageDown":            (*BufPane).CursorPageDown,
	"CursorLeft":                (*BufPane).CursorLeft,
	"CursorRight":               (*BufPane).CursorRight,
	"CursorStart":               (*BufPane).CursorStart,
	"CursorEnd":                 (*BufPane).CursorEnd,
	"CursorToViewTop":           (*BufPane).CursorToViewTop,
	"CursorToViewCenter":        (*BufPane).CursorToViewCenter,
	"CursorToViewBottom":        (*BufPane).CursorToViewBottom,
	"SelectToStart":             (*BufPane).SelectToStart,
	"SelectToEnd":               (*BufPane).SelectToEnd,
	"SelectUp":                  (*BufPane).SelectUp,
	"SelectDown":                (*BufPane).SelectDown,
	"SelectLeft":                (*BufPane).SelectLeft,
	"SelectRight":               (*BufPane).SelectRight,
	"WordRight":                 (*BufPane).WordRight,
	"WordLeft":                  (*BufPane).WordLeft,
	"SubWordRight":              (*BufPane).SubWordRight,
	"SubWordLeft":               (*BufPane).SubWordLeft,
	"SelectWordRight":           (*BufPane).SelectWordRight,
	"SelectWordLeft":            (*BufPane).SelectWordLeft,
	"SelectSubWordRight":        (*BufPane).SelectSubWordRight,
	"SelectSubWordLeft":         (*BufPane).SelectSubWordLeft,
	"DeleteWordRight":           (*BufPane).DeleteWordRight,
	"DeleteWordLeft":            (*BufPane).DeleteWordLeft,
	"DeleteSubWordRight":        (*BufPane).DeleteSubWordRight,
	"DeleteSubWordLeft":         (*BufPane).DeleteSubWordLeft,
	"SelectLine":                (*BufPane).SelectLine,
	"SelectToStartOfLine":       (*BufPane).SelectToStartOfLine,
	"SelectToStartOfText":       (*BufPane).SelectToStartOfText,
	"SelectToStartOfTextToggle": (*BufPane).SelectToStartOfTextToggle,
	"SelectToEndOfLine":         (*BufPane).SelectToEndOfLine,
	"ParagraphPrevious":         (*BufPane).ParagraphPrevious,
	"ParagraphNext":             (*BufPane).ParagraphNext,
	"SelectToParagraphPrevious": (*BufPane).SelectToParagraphPrevious,
	"SelectToParagraphNext":     (*BufPane).SelectToParagraphNext,
	"InsertNewline":             (*BufPane).InsertNewline,
	"Backspace":                 (*BufPane).Backspace,
	"Delete":                    (*BufPane).Delete,
	"InsertTab":                 (*BufPane).InsertTab,
	"Save":                      (*BufPane).Save,
	"SaveAll":                   (*BufPane).SaveAll,
	"SaveAs":                    (*BufPane).SaveAs,
	"Find":                      (*BufPane).Find,
	"FindLiteral":               (*BufPane).FindLiteral,
	"FindNext":                  (*BufPane).FindNext,
	"FindPrevious":              (*BufPane).FindPrevious,
	"DiffNext":                  (*BufPane).DiffNext,
	"DiffPrevious":              (*BufPane).DiffPrevious,
	"Center":                    (*BufPane).Center,
	"Undo":                      (*BufPane).Undo,
	"Redo":                      (*BufPane).Redo,
	"Copy":                      (*BufPane).Copy,
	"CopyLine":                  (*BufPane).CopyLine,
	"Cut":                       (*BufPane).Cut,
	"CutLine":                   (*BufPane).CutLine,
	"Duplicate":                 (*BufPane).Duplicate,
	"DuplicateLine":             (*BufPane).DuplicateLine,
	"DeleteLine":                (*BufPane).DeleteLine,
	"MoveLinesUp":               (*BufPane).MoveLinesUp,
	"MoveLinesDown":             (*BufPane).MoveLinesDown,
	"IndentSelection":           (*BufPane).IndentSelection,
	"OutdentSelection":          (*BufPane).OutdentSelection,
	"Autocomplete":              (*BufPane).Autocomplete,
	"CycleAutocompleteBack":     (*BufPane).CycleAutocompleteBack,
	"OutdentLine":               (*BufPane).OutdentLine,
	"IndentLine":                (*BufPane).IndentLine,
	"Paste":                     (*BufPane).Paste,
	"PastePrimary":              (*BufPane).PastePrimary,
	"SelectAll":                 (*BufPane).SelectAll,
	"OpenFile":                  (*BufPane).OpenFile,
	"Start":                     (*BufPane).Start,
	"End":                       (*BufPane).End,
	"PageUp":                    (*BufPane).PageUp,
	"PageDown":                  (*BufPane).PageDown,
	"SelectPageUp":              (*BufPane).SelectPageUp,
	"SelectPageDown":            (*BufPane).SelectPageDown,
	"HalfPageUp":                (*BufPane).HalfPageUp,
	"HalfPageDown":              (*BufPane).HalfPageDown,
	"StartOfText":               (*BufPane).StartOfText,
	"StartOfTextToggle":         (*BufPane).StartOfTextToggle,
	"StartOfLine":               (*BufPane).StartOfLine,
	"EndOfLine":                 (*BufPane).EndOfLine,
	"ToggleHelp":                (*BufPane).ToggleHelp,
	"ToggleKeyMenu":             (*BufPane).ToggleKeyMenu,
	"ToggleDiffGutter":          (*BufPane).ToggleDiffGutter,
	"ToggleRuler":               (*BufPane).ToggleRuler,
	"ToggleHighlightSearch":     (*BufPane).ToggleHighlightSearch,
	"UnhighlightSearch":         (*BufPane).UnhighlightSearch,
	"ResetSearch":               (*BufPane).ResetSearch,
	"ClearStatus":               (*BufPane).ClearStatus,
	"ShellMode":                 (*BufPane).ShellMode,
	"CommandMode":               (*BufPane).CommandMode,
	"ToggleOverwriteMode":       (*BufPane).ToggleOverwriteMode,
	"Escape":                    (*BufPane).Escape,
	"Quit":                      (*BufPane).Quit,
	"QuitAll":                   (*BufPane).QuitAll,
	"ForceQuit":                 (*BufPane).ForceQuit,
	"AddTab":                    (*BufPane).AddTab,
	"PreviousTab":               (*BufPane).PreviousTab,
	"NextTab":                   (*BufPane).NextTab,
	"FirstTab":                  (*BufPane).FirstTab,
	"LastTab":                   (*BufPane).LastTab,
	"NextSplit":                 (*BufPane).NextSplit,
	"PreviousSplit":             (*BufPane).PreviousSplit,
	"FirstSplit":                (*BufPane).FirstSplit,
	"LastSplit":                 (*BufPane).LastSplit,
	"Unsplit":                   (*BufPane).Unsplit,
	"VSplit":                    (*BufPane).VSplitAction,
	"HSplit":                    (*BufPane).HSplitAction,
	"ToggleMacro":               (*BufPane).ToggleMacro,
	"PlayMacro":                 (*BufPane).PlayMacro,
	"Suspend":                   (*BufPane).Suspend,
	"ScrollUp":                  (*BufPane).ScrollUpAction,
	"ScrollDown":                (*BufPane).ScrollDownAction,
	"SpawnMultiCursor":          (*BufPane).SpawnMultiCursor,
	"SpawnMultiCursorUp":        (*BufPane).SpawnMultiCursorUp,
	"SpawnMultiCursorDown":      (*BufPane).SpawnMultiCursorDown,
	"SpawnMultiCursorSelect":    (*BufPane).SpawnMultiCursorSelect,
	"RemoveMultiCursor":         (*BufPane).RemoveMultiCursor,
	"RemoveAllMultiCursors":     (*BufPane).RemoveAllMultiCursors,
	"SkipMultiCursor":           (*BufPane).SkipMultiCursor,
	"SkipMultiCursorBack":       (*BufPane).SkipMultiCursorBack,
	"JumpToMatchingBrace":       (*BufPane).JumpToMatchingBrace,
	"JumpToOpeningBrace":        (*BufPane).JumpToOpeningBrace,
	"JumpToClosingBrace":        (*BufPane).JumpToClosingBrace,
	"JumpLine":                  (*BufPane).JumpLine,
	"Deselect":                  (*BufPane).Deselect,
	"ClearInfo":                 (*BufPane).ClearInfo,
	"None":                      (*BufPane).None,

	// This was changed to InsertNewline but I don't want to break backwards compatibility
	"InsertEnter": (*BufPane).InsertNewline,
}

// BufMouseActions contains the list of all possible mouse actions the bufhandler could execute
var BufMouseActions = map[string]BufMouseAction{
	"MousePress":       (*BufPane).MousePress,
	"MouseDrag":        (*BufPane).MouseDrag,
	"MouseRelease":     (*BufPane).MouseRelease,
	"MouseMultiCursor": (*BufPane).MouseMultiCursor,
}

// MultiActions is a list of actions that should be executed multiple
// times if there are multiple cursors (one per cursor)
// Generally actions that modify global editor state like quitting or
// saving should not be included in this list
var MultiActions = map[string]bool{
	"CursorUp":                  true,
	"CursorDown":                true,
	"CursorPageUp":              true,
	"CursorPageDown":            true,
	"CursorLeft":                true,
	"CursorRight":               true,
	"CursorStart":               true,
	"CursorEnd":                 true,
	"SelectToStart":             true,
	"SelectToEnd":               true,
	"SelectUp":                  true,
	"SelectDown":                true,
	"SelectLeft":                true,
	"SelectRight":               true,
	"WordRight":                 true,
	"WordLeft":                  true,
	"SubWordRight":              true,
	"SubWordLeft":               true,
	"SelectWordRight":           true,
	"SelectWordLeft":            true,
	"SelectSubWordRight":        true,
	"SelectSubWordLeft":         true,
	"DeleteWordRight":           true,
	"DeleteWordLeft":            true,
	"DeleteSubWordRight":        true,
	"DeleteSubWordLeft":         true,
	"SelectLine":                true,
	"SelectToStartOfLine":       true,
	"SelectToStartOfText":       true,
	"SelectToStartOfTextToggle": true,
	"SelectToEndOfLine":         true,
	"ParagraphPrevious":         true,
	"ParagraphNext":             true,
	"InsertNewline":             true,
	"Backspace":                 true,
	"Delete":                    true,
	"InsertTab":                 true,
	"FindNext":                  true,
	"FindPrevious":              true,
	"CopyLine":                  true,
	"Copy":                      true,
	"Cut":                       true,
	"CutLine":                   true,
	"Duplicate":                 true,
	"DuplicateLine":             true,
	"DeleteLine":                true,
	"MoveLinesUp":               true,
	"MoveLinesDown":             true,
	"IndentSelection":           true,
	"OutdentSelection":          true,
	"OutdentLine":               true,
	"IndentLine":                true,
	"Paste":                     true,
	"PastePrimary":              true,
	"SelectPageUp":              true,
	"SelectPageDown":            true,
	"StartOfLine":               true,
	"StartOfText":               true,
	"StartOfTextToggle":         true,
	"EndOfLine":                 true,
	"JumpToMatchingBrace":       true,
	"JumpToOpeningBrace":        true,
	"JumpToClosingBrace":        true,
}
