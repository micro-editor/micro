package action

import (
	"errors"
	"runtime"

	"github.com/micro-editor/micro/v2/internal/clipboard"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/shell"
	"github.com/micro-editor/tcell/v2"
	"github.com/micro-editor/terminal"
)

type TermKeyAction func(*TermPane)

var TermBindings *KeyTree

func init() {
	TermBindings = NewKeyTree()
}

func TermKeyActionGeneral(a TermKeyAction) PaneKeyAction {
	return func(p Pane) bool {
		a(p.(*TermPane))
		return true
	}
}

func TermMapEvent(k Event, action string) {
	config.Bindings["terminal"][k.Name()] = action

	switch e := k.(type) {
	case KeyEvent, KeySequenceEvent, RawEvent:
		termMapKey(e, action)
	case MouseEvent:
		termMapMouse(e, action)
	}
}

func termMapKey(k Event, action string) {
	if f, ok := TermKeyActions[action]; ok {
		TermBindings.RegisterKeyBinding(k, TermKeyActionGeneral(f))
	}
}

func termMapMouse(k MouseEvent, action string) {
	// TODO: map mouse
	termMapKey(k, action)
}

type TermPane struct {
	*shell.Terminal
	display.Window

	mouseReleased bool
	id            uint64
	tab           *Tab
}

func NewTermPane(x, y, w, h int, t *shell.Terminal, id uint64, tab *Tab) (*TermPane, error) {
	if !TermEmuSupported {
		return nil, errors.New("Terminal emulator is not supported on this system")
	}

	th := new(TermPane)
	th.Terminal = t
	th.id = id
	th.mouseReleased = true
	th.Window = display.NewTermWindow(x, y, w, h, t)
	th.tab = tab
	return th, nil
}

func (t *TermPane) ID() uint64 {
	return t.id
}

func (t *TermPane) SetID(i uint64) {
	t.id = i
}

func (t *TermPane) Name() string {
	return t.Terminal.Name()
}

func (t *TermPane) SetTab(tab *Tab) {
	t.tab = tab
}

func (t *TermPane) Tab() *Tab {
	return t.tab
}

func (t *TermPane) Close() {}

// Quit closes this termpane
func (t *TermPane) Quit() {
	t.Close()
	if len(MainTab().Panes) > 1 {
		t.Unsplit()
	} else if len(Tabs.List) > 1 {
		Tabs.RemoveTab(t.id)
	} else {
		screen.Screen.Fini()
		InfoBar.Close()
		runtime.Goexit()
	}
}

// Unsplit removes this split
func (t *TermPane) Unsplit() {
	n := MainTab().GetNode(t.id)
	n.Unsplit()

	MainTab().RemovePane(MainTab().GetPane(t.id))
	MainTab().Resize()
	MainTab().SetActive(len(MainTab().Panes) - 1)
}

// HandleEvent handles a tcell event by forwarding it to the terminal emulator
// If the event is a mouse event and the program running in the emulator
// does not have mouse support, the emulator will support selections and
// copy-paste
func (t *TermPane) HandleEvent(event tcell.Event) {
	if e, ok := event.(*tcell.EventKey); ok {
		ke := keyEvent(e)
		action, more := TermBindings.NextEvent(ke, nil)

		if !more {
			if action != nil {
				action(t)
				TermBindings.ResetEvents()
				return
			}
			TermBindings.ResetEvents()
		}

		if more {
			return
		}

		if t.Status == shell.TTDone {
			switch e.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlQ, tcell.KeyEnter:
				t.Close()
				t.Quit()
			default:
			}
		}
		if e.Key() == tcell.KeyCtrlC && t.HasSelection() {
			clipboard.Write(t.GetSelection(t.GetView().Width), clipboard.ClipboardReg)
			InfoBar.Message("Copied selection to clipboard")
		} else if t.Status != shell.TTDone {
			t.WriteString(event.EscSeq())
		}
	} else if _, ok := event.(*tcell.EventPaste); ok {
		if t.Status != shell.TTDone {
			t.WriteString(event.EscSeq())
		}
	} else if e, ok := event.(*tcell.EventMouse); !ok || t.State.Mode(terminal.ModeMouseMask) {
		// t.WriteString(event.EscSeq())
	} else {
		x, y := e.Position()
		v := t.GetView()
		x -= v.X
		y -= v.Y

		if e.Buttons() == tcell.Button1 {
			if !t.mouseReleased {
				// drag
				t.Selection[1].X = x
				t.Selection[1].Y = y
			} else {
				t.Selection[0].X = x
				t.Selection[0].Y = y
				t.Selection[1].X = x
				t.Selection[1].Y = y
			}

			t.mouseReleased = false
		} else if e.Buttons() == tcell.ButtonNone {
			if !t.mouseReleased {
				t.Selection[1].X = x
				t.Selection[1].Y = y
			}
			t.mouseReleased = true
		}
	}
}

// HandleTermClose is called when a terminal has finished its job
// and should be closed. If that terminal is this termpane's terminal,
// HandleTermClose will close the terminal and the termpane itself.
func (t *TermPane) HandleTermClose() {
	if t.Status == shell.TTClose {
		t.Quit()
	}
}

// Exit closes the termpane
func (t *TermPane) Exit() {
	t.Terminal.Close()
	t.Quit()
}

// CommandMode opens the termpane's command mode
func (t *TermPane) CommandMode() {
	InfoBar.Prompt("> ", "", "TerminalCommand", nil, func(resp string, canceled bool) {
		if !canceled {
			t.HandleCommand(resp)
		}
	})
}

// NextSplit changes the view to the next split
func (t *TermPane) NextSplit() {
	a := Tabs.List[Tabs.Active()]
	a.SetActive((a.active + 1) % len(a.Panes))
}

// PreviousSplit changes the view to the previous split
func (t *TermPane) PreviousSplit() {
	a := Tabs.List[Tabs.Active()]
	panesLen := len(a.Panes)
	a.SetActive((a.active + panesLen - 1) % panesLen)
}

// PreviousTab switches to the previous tab in the tab list
func (t *TermPane) PreviousTab() {
	tabsLen := len(Tabs.List)
	Tabs.SetActive((Tabs.Active() + tabsLen - 1) % tabsLen)
}

// NextTab switches to the next tab in the tab list
func (t *TermPane) NextTab() {
	Tabs.SetActive((Tabs.Active() + 1) % len(Tabs.List))
}

// HandleCommand handles a command for the term pane
func (t *TermPane) HandleCommand(input string) {
	InfoBar.Error("Commands are unsupported in term for now")
}

// TermKeyActions contains the list of all possible key actions the termpane could execute
var TermKeyActions = map[string]TermKeyAction{
	"Exit":          (*TermPane).Exit,
	"CommandMode":   (*TermPane).CommandMode,
	"NextSplit":     (*TermPane).NextSplit,
	"PreviousSplit": (*TermPane).PreviousSplit,
	"NextTab":       (*TermPane).NextTab,
	"PreviousTab":   (*TermPane).PreviousTab,
}
