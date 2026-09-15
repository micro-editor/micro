package screen

import (
	"errors"
	"log"
	"os"
	"sync"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/micro-editor/tcell/v2"
)

// Screen is the tcell screen we use to draw to the terminal
// Synchronization is used because we poll the screen on a separate
// thread and sometimes the screen is shut down by the main thread
// (for example on TermMessage) so we don't want to poll a nil/shutdown
// screen. TODO: maybe we should worry about polling and drawing at the
// same time too.
var Screen tcell.Screen

// Events is the channel of tcell events
var Events chan (tcell.Event)

// RestartCallback is called when the screen is restarted after it was
// temporarily shut down
var RestartCallback func()

// The lock is necessary since the screen is polled on a separate thread
var lock sync.Mutex

// drawChan is a channel that will cause the screen to redraw when
// written to even if no event user event has occurred
var drawChan chan bool

// rawSeq is the list of raw escape sequences that are bound to some actions
// via keybindings and thus should be parsed by tcell. We need to register
// them in tcell every time we reinitialize the screen, so we need to remember
// them in a list
var rawSeq = make([]string, 0)

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

// ShowFakeCursor displays a cursor at the given position by modifying the
// style of the given column instead of actually using the terminal cursor
// This can be useful in certain terminals such as the windows console where
// modifying the cursor location is slow and frequent modifications cause flashing
// This keeps track of the most recent fake cursor location and resets it when
// a new fake cursor location is specified
func ShowFakeCursor(x, y int) {
	r, combc, style, _ := Screen.GetContent(x, y)
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
	r, _, _, _ := Screen.GetContent(x, y)
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
	if !Screen.CanDisplay(mainc, true) {
		mainc = '�'
	}

	Screen.SetContent(x, y, mainc, combc, style)
	if UseFake() && lastCursor.x == x && lastCursor.y == y {
		lastCursor.r = mainc
		lastCursor.style = style
		lastCursor.combc = combc
	}
}

// RegisterRawSeq registers a raw escape sequence that should be parsed by tcell
func RegisterRawSeq(r string) {
	for _, seq := range rawSeq {
		if seq == r {
			return
		}
	}
	rawSeq = append(rawSeq, r)

	if Screen != nil {
		Screen.RegisterRawSeq(r)
	}
}

// UnregisterRawSeq unregisters a raw escape sequence that should be parsed by tcell
func UnregisterRawSeq(r string) {
	for i, seq := range rawSeq {
		if seq == r {
			rawSeq[i] = rawSeq[len(rawSeq)-1]
			rawSeq = rawSeq[:len(rawSeq)-1]
		}
	}

	if Screen != nil {
		Screen.UnregisterRawSeq(r)
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

	util.SetAmbiguousWidth(config.GetGlobalOption("ambiguouswidth").(string))

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

	Screen.SetPaste(config.GetGlobalOption("paste").(bool))

	// restore TERM
	if modifiedTerm {
		os.Setenv("TERM", oldTerm)
	}

	if config.GetGlobalOption("mouse").(bool) {
		Screen.EnableMouse()
	}

	for _, r := range rawSeq {
		Screen.RegisterRawSeq(r)
	}

	return nil
}

// InitSimScreen initializes a simulation screen for testing purposes
func InitSimScreen() (tcell.SimulationScreen, error) {
	drawChan = make(chan bool, 8)

	// Initilize tcell
	var err error
	s := tcell.NewSimulationScreen("")
	if s == nil {
		return nil, errors.New("Failed to get a simulation screen")
	}
	if err = s.Init(); err != nil {
		return nil, err
	}

	s.SetSize(80, 24)
	Screen = s

	if config.GetGlobalOption("mouse").(bool) {
		Screen.EnableMouse()
	}

	return s, nil
}
