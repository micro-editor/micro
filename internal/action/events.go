package action

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v3"
)

type Event interface {
	Name() string
}

// RawEvent is simply an escape code
// We allow users to directly bind escape codes
// to get around some of a limitations of terminals
type RawEvent struct {
	esc string
}

func (r RawEvent) Name() string {
	return r.esc
}

// KeyEvent is a key event containing a key code,
// some possible modifiers (alt, ctrl, etc...) and
// a grapheme-cluster string if it was simply a character press
// Note: to be compatible with tcell events, for ctrl keys str
// holds the rune-derived string.
type KeyEvent struct {
	code tcell.Key
	mod  tcell.ModMask
	str  string
	any  bool
}

func metaToAlt(mod tcell.ModMask) tcell.ModMask {
	if mod&tcell.ModMeta != 0 {
		mod &= ^tcell.ModMeta
		mod |= tcell.ModAlt
	}
	return mod
}

func keyEvent(e *tcell.EventKey) KeyEvent {
	ke := KeyEvent{
		code: e.Key(),
		mod:  metaToAlt(e.Modifiers()),
	}
	if e.Key() == tcell.KeyRune {
		ke.str = e.Str()
	}
	return ke
}

func (k KeyEvent) Name() string {
	if k.any {
		return "<any>"
	}
	s := ""
	m := []string{}
	if k.mod&tcell.ModShift != 0 {
		m = append(m, "Shift")
	}
	if k.mod&tcell.ModAlt != 0 {
		m = append(m, "Alt")
	}
	if k.mod&tcell.ModMeta != 0 {
		m = append(m, "Meta")
	}
	if k.mod&tcell.ModCtrl != 0 {
		m = append(m, "Ctrl")
	}

	ok := false
	if s, ok = tcell.KeyNames[k.code]; !ok {
		if k.code == tcell.KeyRune {
			s = k.str
		} else {
			s = fmt.Sprintf("Key[%d]", k.code)
		}
	}
	if len(m) != 0 {
		if k.mod&tcell.ModCtrl != 0 && strings.HasPrefix(s, "Ctrl-") {
			s = s[5:]
			if len(s) == 1 {
				s = strings.ToLower(s)
			}
		}
		return fmt.Sprintf("%s-%s", strings.Join(m, "-"), s)
	}
	return s
}

// A KeySequence defines a list of consecutive
// events. All events in the sequence must be KeyEvents
// or MouseEvents.
type KeySequenceEvent struct {
	keys []Event
}

func (k KeySequenceEvent) Name() string {
	buf := bytes.Buffer{}
	for _, e := range k.keys {
		buf.WriteByte('<')
		buf.WriteString(e.Name())
		buf.WriteByte('>')
	}
	return buf.String()
}

type MouseState int

const (
	MousePress = iota
	MouseDrag
	MouseRelease
)

// MouseEvent is a mouse event with a mouse button and
// any possible key modifiers
type MouseEvent struct {
	btn   tcell.ButtonMask
	mod   tcell.ModMask
	state MouseState
}

func (m MouseEvent) Name() string {
	mod := ""
	if m.mod&tcell.ModShift != 0 {
		mod = "Shift-"
	}
	if m.mod&tcell.ModAlt != 0 {
		mod = "Alt-"
	}
	if m.mod&tcell.ModMeta != 0 {
		mod = "Meta-"
	}
	if m.mod&tcell.ModCtrl != 0 {
		mod = "Ctrl-"
	}

	state := ""
	switch m.state {
	case MouseDrag:
		state = "Drag"
	case MouseRelease:
		state = "Release"
	}

	for k, v := range mouseEvents {
		if v == m.btn {
			return fmt.Sprintf("%s%s%s", mod, k, state)
		}
	}
	return ""
}

// eventEscSeq returns the raw escape sequence associated with a tcell
// event, if any. Only EventKey carries source bytes through the
// tcell v3 EscSeq overlay.
func eventEscSeq(event tcell.Event) string {
	if e, ok := event.(*tcell.EventKey); ok {
		return e.EscSeq()
	}
	return ""
}

// ConstructEvent takes a tcell event and returns a micro
// event. Note that tcell events can't express certain
// micro events such as key sequences. This function is
// mostly used for debugging/raw panes or constructing
// intermediate micro events while parsing a sequence.
func ConstructEvent(event tcell.Event) (Event, error) {
	switch e := event.(type) {
	case *tcell.EventKey:
		return keyEvent(e), nil
	case *tcell.EventMouse:
		return MouseEvent{
			btn: e.Buttons(),
			mod: metaToAlt(e.Modifiers()),
		}, nil
	}
	return nil, errors.New("No micro event equivalent")
}

// A Handler will take a tcell event and execute it
// appropriately
type Handler interface {
	HandleEvent(tcell.Event)
	HandleCommand(string)
}
