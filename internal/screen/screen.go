package screen

import (
	"log"
	"os"
	"sync"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/micro-editor/micro/v2/internal/config"
)

// Screen is the tcell screen we use to draw to the terminal
// Synchronization is used because we poll the screen on a separate
// thread and sometimes the screen is shut down by the main thread
// (for example on TermMessage) so we don't want to poll a nil/shutdown
// screen. TODO: maybe we should worry about polling and drawing at the
// same time too.
var Screen tcell.Screen

// Events is the channel of tcell events. It is an alias for the tcell
// Screen.EventQ() channel, populated during Init().
var Events chan tcell.Event

// RestartCallback is called when the screen is restarted after it was
// temporarily shut down
var RestartCallback func()

// The lock is necessary since the screen is polled on a separate thread
var lock sync.Mutex

// drawChan is a channel that will cause the screen to redraw when
// written to even if no event user event has occurred
var drawChan chan bool

// Lock locks the screen lock
func Lock() {
	lock.Lock()
}

// Unlock unlocks the screen lock
func Unlock() {
	lock.Unlock()
}

// Redraw schedules a redraw with the draw channel
func Redraw() {
	select {
	case drawChan <- true:
	default:
		// channel is full
	}
}

// DrawChan returns the draw channel
func DrawChan() chan bool {
	return drawChan
}

type screenCell struct {
	x, y  int
	r     rune
	combc []rune
	style tcell.Style
}

var lastCursor screenCell

// splitGrapheme splits a tcell v3 grapheme-cluster string into a
// (primary rune, combining runes) pair for the Screen.SetContent API,
// which still takes a rune + []rune.
func splitGrapheme(s string) (rune, []rune) {
	rs := []rune(s)
	switch len(rs) {
	case 0:
		return ' ', nil
	case 1:
		return rs[0], nil
	default:
		return rs[0], rs[1:]
	}
}

// ShowFakeCursor displays a cursor at the given position by modifying the
// style of the given column instead of actually using the terminal cursor
// This can be useful in certain terminals such as the windows console where
// modifying the cursor location is slow and frequent modifications cause flashing
// This keeps track of the most recent fake cursor location and resets it when
// a new fake cursor location is specified
func ShowFakeCursor(x, y int) {
	s, style, _ := Screen.Get(x, y)
	r, combc := splitGrapheme(s)
	Screen.SetContent(lastCursor.x, lastCursor.y, lastCursor.r, lastCursor.combc, lastCursor.style)
	Screen.SetContent(x, y, r, combc, config.DefStyle.Reverse(true))

	lastCursor.x, lastCursor.y = x, y
	lastCursor.r = r
	lastCursor.combc = combc
	lastCursor.style = style
}

func UseFake() bool {
	return config.GetGlobalOption("fakecursor").(bool)
}

// ShowFakeCursorMulti is the same as ShowFakeCursor except it does not
// reset previous locations of the cursor
// Fake cursors are also necessary to display multiple cursors
func ShowFakeCursorMulti(x, y int) {
	s, _, _ := Screen.Get(x, y)
	r, _ := splitGrapheme(s)
	Screen.SetContent(x, y, r, nil, config.DefStyle.Reverse(true))
}

// ShowCursor puts the cursor at the given location using a fake cursor
// if enabled or using the terminal cursor otherwise
// By default only the windows console will use a fake cursor
func ShowCursor(x, y int) {
	if UseFake() {
		ShowFakeCursor(x, y)
	} else {
		Screen.ShowCursor(x, y)
	}
}

// SetContent sets a cell at a point on the screen and makes sure that it is
// synced with the last cursor location
func SetContent(x, y int, mainc rune, combc []rune, style tcell.Style) {
	Screen.SetContent(x, y, mainc, combc, style)
	if UseFake() && lastCursor.x == x && lastCursor.y == y {
		lastCursor.r = mainc
		lastCursor.style = style
		lastCursor.combc = combc
	}
}

// TempFini shuts the screen down temporarily
func TempFini() bool {
	screenWasNil := Screen == nil

	if !screenWasNil {
		Screen.Fini()
		Lock()
		Screen = nil
	}
	return screenWasNil
}

// TempStart restarts the screen after it was temporarily disabled
func TempStart(screenWasNil bool) {
	if !screenWasNil {
		Init()
		Unlock()

		if RestartCallback != nil {
			RestartCallback()
		}
	}
}

// Init creates and initializes the tcell screen
func Init() error {
	drawChan = make(chan bool, 8)

	// Should we enable true color?
	truecolor := config.GetGlobalOption("truecolor").(string)
	if truecolor == "on" || (truecolor == "auto" && os.Getenv("MICRO_TRUECOLOR") == "1") {
		os.Setenv("TCELL_TRUECOLOR", "enable")
	} else if truecolor == "off" {
		os.Setenv("TCELL_TRUECOLOR", "disable")
	} else {
		// For "auto", tcell already autodetects truecolor by default
	}

	var oldTerm string
	modifiedTerm := false
	setXterm := func() {
		oldTerm = os.Getenv("TERM")
		os.Setenv("TERM", "xterm-256color")
		modifiedTerm = true
	}

	if config.GetGlobalOption("xterm").(bool) {
		setXterm()
	}

	// Initilize tcell
	var err error
	Screen, err = tcell.NewScreen()
	if err != nil {
		log.Println("Warning: during screen initialization:", err)
		log.Println("Falling back to TERM=xterm-256color")
		setXterm()
		Screen, err = tcell.NewScreen()
		if err != nil {
			return err
		}
	}
	if err = Screen.Init(); err != nil {
		return err
	}

	if config.GetGlobalOption("paste").(bool) {
		Screen.EnablePaste()
	} else {
		Screen.DisablePaste()
	}

	// restore TERM
	if modifiedTerm {
		os.Setenv("TERM", oldTerm)
	}

	if config.GetGlobalOption("mouse").(bool) {
		Screen.EnableMouse()
	}

	Events = Screen.EventQ()

	return nil
}

// InitMockScreen initializes a terminfo-backed tcell.Screen wired to a
// vt.MockTerm for use in tests. The returned MockTerm is the test-side
// handle for injecting keys/mouse/raw-bytes; the tcell.Screen is
// installed as screen.Screen and its EventQ() is aliased to
// screen.Events, so the rest of micro drives it unchanged.
//
// Uses OptTerm("xterm-256color") to pin terminfo lookup rather than
// relying on $TERM, which may be unset or "dumb" in CI.
//
// This replaces the v2-era InitSimScreen helper: tcell v3 removed
// SimulationScreen, and vt.MockTerm is the blessed upstream mock.
// Note that the vt package docstring flags its API as still under
// development, so test-only use is deliberate.
func InitMockScreen() (vt.MockTerm, error) {
	drawChan = make(chan bool, 8)

	mt := vt.NewMockTerm()
	s, err := tcell.NewTerminfoScreenFromTty(mt, tcell.OptTerm("xterm-256color"))
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	Screen = s
	Events = Screen.EventQ()

	if config.GetGlobalOption("mouse").(bool) {
		Screen.EnableMouse()
	}

	return mt, nil
}
