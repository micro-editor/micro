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

// InfoBufBindings aliases the command-bar tree for embedded buffer events.
var InfoBufBindings *KeyTree

func init() {
	InfoBindings = NewKeyTree()
	InfoBufBindings = InfoBindings
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
	if util.IndexAnyUnquoted(action, "&|,") >= 0 {
		if f := infoKeyChain(action); f != nil && infoKeyboardEvent(k) {
			InfoBindings.RegisterKeyBinding(k, f)
		}
		return
	}
	if f, ok := InfoKeyActions[action]; ok {
		InfoBindings.RegisterKeyBinding(k, InfoKeyActionGeneral(f))
	} else if f, ok := BufKeyActions[action]; ok {
		InfoBindings.RegisterKeyBinding(k, BufKeyActionGeneral(f))
	}
}

// infoKeyboardEvent identifies bindings dispatched through InfoPane.DoKeyEvent.
// Raw and mouse events remain single-action bindings in the same tree.
func infoKeyboardEvent(k Event) bool {
	switch e := k.(type) {
	case KeyEvent:
		return true
	case KeySequenceEvent:
		for _, key := range e.keys {
			if _, ok := key.(KeyEvent); !ok {
				return false
			}
		}
		return true
	}
	return false
}

// infoKeyChain keeps command-bar actions on the command buffer, even when a
// buffer action would normally update its target to the current document pane.
func infoKeyChain(action string) PaneKeyAction {
	type step struct {
		run    func(*InfoPane) bool
		sep    byte
		closes bool
	}
	var steps []step
	for action != "" {
		name, sep := action, byte(' ')
		if idx := util.IndexAnyUnquoted(action, "&|,"); idx >= 0 {
			name, sep, action = action[:idx], action[idx], action[idx+1:]
		} else {
			action = ""
		}
		s := step{sep: sep, closes: name == "ExecuteCommand" || name == "AbortCommand"}
		if f, ok := InfoKeyActions[name]; ok {
			s.run = func(h *InfoPane) bool {
				f(h)
				return true
			}
		} else if f, ok := BufKeyActions[name]; ok {
			s.run = func(h *InfoPane) bool {
				return h.BufPane.execAction(f, name, nil)
			}
		} else {
			// Do not execute a partial chain containing an unsupported action.
			return nil
		}
		steps = append(steps, s)
	}
	return func(p Pane) bool {
		h := commandInfoPane(p)
		if h == nil {
			return false
		}
		for _, s := range steps {
			if !h.HasPrompt || h.HasYN {
				break
			}
			success := s.run(h)
			if s.closes || (!success && s.sep == '&') || (success && s.sep == '|') {
				break
			}
		}
		return true
	}
}

func infoMapMouse(k MouseEvent, action string) {
	// TODO: map mouse
	if f, ok := BufMouseActions[action]; ok {
		InfoBindings.RegisterMouseBinding(k, BufMouseActionGeneral(f))
	} else {
		infoMapKey(k, action)
	}
}

func InfoKeyActionGeneral(a InfoKeyAction) PaneKeyAction {
	return func(p Pane) bool {
		h := commandInfoPane(p)
		if h == nil {
			return false
		}
		a(h)
		return true
	}
}

// commandInfoPane adapts actions from the embedded command buffer without
// redirecting them to a document pane. Non-command targets are rejected.
func commandInfoPane(p Pane) *InfoPane {
	switch h := p.(type) {
	case *InfoPane:
		return h
	case *BufPane:
		return h.infoPane
	}
	return nil
}

type InfoPane struct {
	*BufPane
	*info.InfoBuf
}

func NewInfoPane(ib *info.InfoBuf, w display.BWindow, tab *Tab) *InfoPane {
	ip := new(InfoPane)
	ip.InfoBuf = ib
	ip.BufPane = NewBufPane(ib.Buffer, w, tab)
	ip.BufPane.bindings = InfoBindings
	ip.BufPane.infoPane = ip

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
		action(h.BufPane)
		InfoBindings.ResetEvents()
		return true
	} else if action == nil && !more {
		InfoBindings.ResetEvents()
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
	if b.HasSuggestions {
		b.CycleAutocomplete(true)
		return
	}

	c := b.GetActiveCursor()
	l := b.LineBytes(0)
	l = util.SliceStart(l, c.X)

	args := bytes.Split(l, []byte{' '})
	cmd := string(args[0])

	if h.PromptType == "Command" {
		if len(args) == 1 {
			b.Autocomplete(CommandComplete)
		} else if action, ok := commands[cmd]; ok {
			if action.completer != nil {
				b.Autocomplete(action.completer)
			}
		}
	} else {
		// by default use filename autocompletion
		b.Autocomplete(buffer.FileComplete)
	}
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
