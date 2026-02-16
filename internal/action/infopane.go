package action

import (
	"bytes"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/info"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/micro-editor/tcell/v2"
)

type InfoKeyAction func(*InfoPane)

var InfoBindings *KeyTree
var InfoBufBindings *KeyTree

func init() {
	InfoBindings = NewKeyTree()
	InfoBufBindings = NewKeyTree()
}

func InfoMapEvent(k Event, action string) {
	config.Bindings["command"][k.Name()] = action

	switch e := k.(type) {
	case KeyEvent, KeySequenceEvent, RawEvent:
		infoMapKey(e, action)
	case MouseEvent:
		infoMapMouse(e, action)
	}
}

func infoMapKey(k Event, action string) {
	if f, ok := InfoKeyActions[action]; ok {
		InfoBindings.RegisterKeyBinding(k, InfoKeyActionGeneral(f))
	} else if f, ok := BufKeyActions[action]; ok {
		InfoBufBindings.RegisterKeyBinding(k, BufKeyActionGeneral(f))
	}
}

func infoMapMouse(k MouseEvent, action string) {
	// TODO: map mouse
	if f, ok := BufMouseActions[action]; ok {
		InfoBufBindings.RegisterMouseBinding(k, BufMouseActionGeneral(f))
	} else {
		infoMapKey(k, action)
	}
}

func InfoKeyActionGeneral(a InfoKeyAction) PaneKeyAction {
	return func(p Pane) bool {
		a(p.(*InfoPane))
		return true
	}
}

type InfoPane struct {
	*BufPane
	*info.InfoBuf
}

func NewInfoPane(ib *info.InfoBuf, w display.BWindow, tab *Tab) *InfoPane {
	ip := new(InfoPane)
	ip.InfoBuf = ib
	ip.BufPane = NewBufPane(ib.Buffer, w, tab)
	ip.BufPane.bindings = InfoBufBindings

	return ip
}

func NewInfoBar() *InfoPane {
	ib := info.NewBuffer()
	w := display.NewInfoWindow(ib)
	return NewInfoPane(ib, w, nil)
}

func (h *InfoPane) Close() {
	h.InfoBuf.Close()
	h.BufPane.Close()
}

func (h *InfoPane) HandleEvent(event tcell.Event) {
	switch e := event.(type) {
	case *tcell.EventResize:
		// TODO
	case *tcell.EventKey:
		ke := keyEvent(e)

		done := h.DoKeyEvent(ke)
		hasYN := h.HasYN
		if e.Key() == tcell.KeyRune && hasYN {
			y := e.Rune() == 'y' || e.Rune() == 'Y'
			n := e.Rune() == 'n' || e.Rune() == 'N'
			if y || n {
				h.YNResp = y
				h.DonePrompt(false)

				InfoBindings.ResetEvents()
				InfoBufBindings.ResetEvents()
			}
		}
		if e.Key() == tcell.KeyRune && !done && !hasYN {
			h.DoRuneInsert(e.Rune())
			done = true
		}
		if done && h.HasPrompt && !hasYN {
			resp := string(h.LineBytes(0))
			hist := h.History[h.PromptType]
			if resp != hist[h.HistoryNum] {
				h.HistoryNum = len(hist) - 1
				hist[h.HistoryNum] = resp
				h.HistorySearch = false
			}
			if h.EventCallback != nil {
				h.EventCallback(resp)
			}
		}
	default:
		h.BufPane.HandleEvent(event)
	}
}

// DoKeyEvent executes a key event for the command bar, doing any overridden actions.
// Returns true if the action was executed OR if there are more keys remaining
// to process before executing an action (if this is a key sequence event).
// Returns false if no action found.
func (h *InfoPane) DoKeyEvent(e KeyEvent) bool {
	action, more := InfoBindings.NextEvent(e, nil)
	if action != nil && !more {
		action(h)
		InfoBindings.ResetEvents()

		return true
	} else if action == nil && !more {
		InfoBindings.ResetEvents()
		// return false //TODO:?
	}

	if !more {
		// If no infopane action found, try to find a bufpane action.
		//
		// TODO: this is buggy. For example, if the command bar has the following
		// two bindings:
		//
		//   "<Ctrl-x><Ctrl-p>": "HistoryUp",
		//   "<Ctrl-x><Ctrl-v>": "Paste",
		//
		// the 2nd binding (with a bufpane action) doesn't work, since <Ctrl-x>
		// has been already consumed by the 1st binding (with an infopane action).
		//
		// We should either iterate both InfoBindings and InfoBufBindings keytrees
		// together, or just use the same keytree for both infopane and bufpane
		// bindings.
		action, more = InfoBufBindings.NextEvent(e, nil)
		if action != nil && !more {
			action(h.BufPane)
			InfoBufBindings.ResetEvents()
			return true
		} else if action == nil && !more {
			InfoBufBindings.ResetEvents()
		}
	}

	return more
}

// HistoryUp cycles history up
func (h *InfoPane) HistoryUp() {
	h.UpHistory(h.History[h.PromptType])
}

// HistoryDown cycles history down
func (h *InfoPane) HistoryDown() {
	h.DownHistory(h.History[h.PromptType])
}

// HistorySearchUp fetches the previous history item beginning with the text
// in the infobuffer before cursor
func (h *InfoPane) HistorySearchUp() {
	h.SearchUpHistory(h.History[h.PromptType])
}

// HistorySearchDown fetches the next history item beginning with the text
// in the infobuffer before cursor
func (h *InfoPane) HistorySearchDown() {
	h.SearchDownHistory(h.History[h.PromptType])
}

// Autocomplete begins autocompletion
func (h *InfoPane) CommandComplete() {
	b := h.Buf
	c := b.GetActiveCursor()

	cc := buffer.AutocompleteCursorCheck(c)
	rc := buffer.AutocompleteRuneCheck(c)

	// Cycling commands
	if !b.HasSuggestions && !cc && !rc {
		return
	}

	if b.HasSuggestions {
		if !cc {
			return
		}

		prevSuggestion := b.CycleAutocomplete(true)
		b.PerformSingleAutocomplete(prevSuggestion, c)
		return
	}

	// Otherwise start autocomplete
	l := b.LineBytes(0)
	l = util.SliceStart(l, c.X)
	args := bytes.Split(l, []byte{' '})
	cmd := string(args[0])

	var completer buffer.Completer = nil

	if h.PromptType == "Command" {
		if len(args) == 1 {
			completer = CommandComplete
		} else if action, ok := commands[cmd]; ok {
			completer = action.completer
		}
	} else {
		// by default use filename autocompletion
		completer = buffer.FileComplete
	}
	if completer == nil {
		return
	}
	if !b.StartAutocomplete(completer) {
		return
	}
	prevSuggestion := b.CycleAutocomplete(true)
	b.PerformSingleAutocomplete(prevSuggestion, c)
}

// ExecuteCommand completes the prompt
func (h *InfoPane) ExecuteCommand() {
	if !h.HasYN {
		h.DonePrompt(false)
	}
}

// AbortCommand cancels the prompt
func (h *InfoPane) AbortCommand() {
	h.DonePrompt(true)
}

// InfoKeyActions contains the list of all possible key actions the infopane could execute
var InfoKeyActions = map[string]InfoKeyAction{
	"HistoryUp":         (*InfoPane).HistoryUp,
	"HistoryDown":       (*InfoPane).HistoryDown,
	"HistorySearchUp":   (*InfoPane).HistorySearchUp,
	"HistorySearchDown": (*InfoPane).HistorySearchDown,
	"CommandComplete":   (*InfoPane).CommandComplete,
	"ExecuteCommand":    (*InfoPane).ExecuteCommand,
	"AbortCommand":      (*InfoPane).AbortCommand,
}
