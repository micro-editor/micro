package main

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/go-errors/errors"
	"github.com/micro-editor/micro/v2/internal/action"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/stretchr/testify/assert"
)

var tempDir string
var mt vt.MockTerm

// settleEvents waits briefly for the tcell reader goroutine to parse
// whatever bytes MockTerm just delivered into EventKey/EventMouse
// events on screen.Events, then returns. Unlike v2's SimulationScreen
// (which exposed a synchronous PollEvent), v3 parses asynchronously,
// so we have to give the reader a window to catch up before draining.
func settleEvents() {
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(screen.Events) > 0 || len(screen.DrawChan()) > 0 {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// drainEvents runs DoEvent once per pending queue entry. Each DoEvent
// call consumes one event from screen.Events (or one Redraw token).
// After draining, a final short settle pass catches any tail events
// the processed action may have posted (e.g. a command prompt).
func drainEvents() {
	for {
		settleEvents()
		if len(screen.Events) == 0 && len(screen.DrawChan()) == 0 {
			return
		}
		for len(screen.Events) > 0 || len(screen.DrawChan()) > 0 {
			DoEvent()
		}
	}
}

func startup(args []string) (vt.MockTerm, error) {
	var err error

	tempDir, err = os.MkdirTemp("", "micro_test")
	if err != nil {
		return nil, err
	}
	err = config.InitConfigDir(tempDir)
	if err != nil {
		return nil, err
	}

	config.InitRuntimeFiles(true)
	config.InitPlugins()

	err = config.ReadSettings()
	if err != nil {
		return nil, err
	}
	err = config.InitGlobalSettings()
	if err != nil {
		return nil, err
	}

	mtLocal, err := screen.InitMockScreen()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := recover(); err != nil {
			screen.Screen.Fini()
			fmt.Println("Micro encountered an error:", err)
			// immediately backup all buffers with unsaved changes
			for _, b := range buffer.OpenBuffers {
				if b.Modified() {
					b.Backup()
				}
			}
			log.Fatalf("%s", errors.Wrap(err, 2).ErrorStack())
		}
	}()

	err = config.LoadAllPlugins()
	if err != nil {
		screen.TermMessage(err)
	}

	action.InitBindings()
	action.InitCommands()

	err = config.InitColorscheme()
	if err != nil {
		return nil, err
	}

	b := LoadInput(args)

	if len(b) == 0 {
		return nil, errors.New("No buffers opened")
	}

	action.InitTabs(b)
	action.InitGlobals()

	err = config.RunPluginFn("init")
	if err != nil {
		return nil, err
	}

	w, h := screen.Screen.Size()
	mtLocal.SetSize(vt.Coord{X: vt.Col(w), Y: vt.Row(h)})
	drainEvents()

	return mtLocal, nil
}

func cleanup() {
	os.RemoveAll(tempDir)
}

// keyBytes translates a (tcell.Key, rune, mod) injection into the raw
// byte sequence tcell's input parser expects. For plain runes the
// payload is the UTF-8 encoding; for control bytes (Ctrl-A .. Ctrl-Z,
// Enter, Tab, Backspace, Esc) it's the single-byte code; for the
// arrow keys it's the legacy CSI escape. This covers every key
// currently exercised by the test suite. Unknown combinations panic
// so a missing mapping fails loudly rather than silently no-opping.
func keyBytes(key tcell.Key, r rune, _ tcell.ModMask) []byte {
	switch key {
	case tcell.KeyRune:
		return []byte(string(r))
	case tcell.KeyEnter:
		return []byte{'\r'}
	case tcell.KeyEscape:
		return []byte{0x1b}
	case tcell.KeyTab:
		return []byte{'\t'}
	case tcell.KeyBackspace:
		return []byte{0x08}
	case tcell.KeyBackspace2:
		return []byte{0x7f}
	case tcell.KeyUp:
		return []byte{0x1b, '[', 'A'}
	case tcell.KeyDown:
		return []byte{0x1b, '[', 'B'}
	case tcell.KeyRight:
		return []byte{0x1b, '[', 'C'}
	case tcell.KeyLeft:
		return []byte{0x1b, '[', 'D'}
	}
	if key >= tcell.KeyCtrlA && key <= tcell.KeyCtrlZ {
		// tcell v3 reshuffled the Key constants: KeyCtrlA is 65, not 1.
		// Convert back to the ASCII control byte (0x01..0x1A).
		return []byte{byte(key-tcell.KeyCtrlA) + 1}
	}
	if r != 0 {
		return []byte(string(r))
	}
	panic(fmt.Sprintf("keyBytes: unsupported key 0x%x", key))
}

func injectKey(key tcell.Key, r rune, mod tcell.ModMask) {
	mt.SendRaw(keyBytes(key, r, mod))
	drainEvents()
}

func injectString(str string) {
	mt.SendRaw([]byte(str))
	drainEvents()
}

// tcellButtonToVt maps a tcell ButtonMask to the single vt.Button it
// corresponds to. micro_test only ever passes Button1 or ButtonNone;
// for ButtonNone (release) the caller must remember the button that
// was last pressed — vt.MouseEvent requires a non-nil Button even
// for release events. We assume Button1 for all releases, which is
// correct for every call site below.
func tcellButtonToVt(b tcell.ButtonMask) vt.Button {
	switch {
	case b&tcell.Button1 != 0:
		return vt.Button1
	case b&tcell.Button2 != 0:
		return vt.Button2
	case b&tcell.Button3 != 0:
		return vt.Button3
	}
	return vt.Button1
}

func injectMouse(x, y int, buttons tcell.ButtonMask, _ tcell.ModMask) {
	down := buttons != tcell.ButtonNone
	mt.MouseEvent(vt.MouseEvent{
		Position: vt.Coord{X: vt.Col(x), Y: vt.Row(y)},
		Button:   tcellButtonToVt(buttons),
		Down:     down,
	})
	drainEvents()
}

func openFile(file string) {
	injectKey(tcell.KeyCtrlE, rune(tcell.KeyCtrlE), tcell.ModCtrl)
	injectString(fmt.Sprintf("open %s", file))
	injectKey(tcell.KeyEnter, rune(tcell.KeyEnter), tcell.ModNone)
}

func findBuffer(file string) *buffer.Buffer {
	var buf *buffer.Buffer
	file = util.ResolvePath(file)
	for _, b := range buffer.OpenBuffers {
		if b.AbsPath == file {
			buf = b
		}
	}
	return buf
}

func createTestFile(t *testing.T, content string) string {
	f, err := os.CreateTemp(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}

	return f.Name()
}

func TestMain(m *testing.M) {
	var err error
	mt, err = startup([]string{})
	if err != nil {
		log.Fatalln(err)
	}

	retval := m.Run()
	cleanup()

	os.Exit(retval)
}

func TestSimpleEdit(t *testing.T) {
	file := createTestFile(t, "base content")

	openFile(file)

	if findBuffer(file) == nil {
		t.Fatalf("Could not find buffer %s", file)
	}

	injectKey(tcell.KeyEnter, rune(tcell.KeyEnter), tcell.ModNone)
	injectKey(tcell.KeyUp, 0, tcell.ModNone)
	injectString("first line")

	for i := 0; i < len("ne"); i++ {
		injectKey(tcell.KeyBackspace, rune(tcell.KeyBackspace), tcell.ModNone)
	}
	for i := 0; i < len(" li"); i++ {
		injectKey(tcell.KeyBackspace2, rune(tcell.KeyBackspace2), tcell.ModNone)
	}
	injectString("foobar")

	injectKey(tcell.KeyCtrlS, rune(tcell.KeyCtrlS), tcell.ModCtrl)

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "firstfoobar\nbase content\n", string(data))
}

func TestMouse(t *testing.T) {
	file := createTestFile(t, "base content")

	openFile(file)

	if findBuffer(file) == nil {
		t.Fatalf("Could not find buffer %s", file)
	}

	// buffer:
	// base content
	// the selections need to happen at different locations to avoid a double click
	injectMouse(3, 0, tcell.Button1, tcell.ModNone)
	injectKey(tcell.KeyLeft, 0, tcell.ModNone)
	injectMouse(0, 0, tcell.ButtonNone, tcell.ModNone)
	injectString("secondline")
	// buffer:
	// secondlinebase content
	injectKey(tcell.KeyEnter, rune(tcell.KeyEnter), tcell.ModNone)
	// buffer:
	// secondline
	// base content
	injectMouse(2, 0, tcell.Button1, tcell.ModNone)
	injectMouse(0, 0, tcell.ButtonNone, tcell.ModNone)
	injectKey(tcell.KeyEnter, rune(tcell.KeyEnter), tcell.ModNone)
	// buffer:
	//
	// secondline
	// base content
	injectKey(tcell.KeyUp, 0, tcell.ModNone)
	injectString("firstline")
	// buffer:
	// firstline
	// secondline
	// base content
	injectKey(tcell.KeyCtrlS, rune(tcell.KeyCtrlS), tcell.ModCtrl)

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "firstline\nsecondline\nbase content\n", string(data))
}

var srTestStart = `foo
foo
foofoofoo
Ernleȝe foo æðelen
`
var srTest2 = `test_string
test_string
test_stringtest_stringtest_string
Ernleȝe test_string æðelen
`
var srTest3 = `test_foo
test_string
test_footest_stringtest_foo
Ernleȝe test_string æðelen
`

func TestSearchAndReplace(t *testing.T) {
	file := createTestFile(t, srTestStart)

	openFile(file)

	if findBuffer(file) == nil {
		t.Fatalf("Could not find buffer %s", file)
	}

	injectKey(tcell.KeyCtrlE, rune(tcell.KeyCtrlE), tcell.ModCtrl)
	injectString(fmt.Sprintf("replaceall %s %s", "foo", "test_string"))
	injectKey(tcell.KeyEnter, rune(tcell.KeyEnter), tcell.ModNone)

	injectKey(tcell.KeyCtrlS, rune(tcell.KeyCtrlS), tcell.ModCtrl)

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, srTest2, string(data))

	injectKey(tcell.KeyCtrlE, rune(tcell.KeyCtrlE), tcell.ModCtrl)
	injectString(fmt.Sprintf("replace %s %s", "string", "foo"))
	injectKey(tcell.KeyEnter, rune(tcell.KeyEnter), tcell.ModNone)
	injectString("ynyny")
	injectKey(tcell.KeyEscape, 0, tcell.ModNone)

	injectKey(tcell.KeyCtrlS, rune(tcell.KeyCtrlS), tcell.ModCtrl)

	data, err = os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, srTest3, string(data))
}

func TestMultiCursor(t *testing.T) {
	// TODO
}

func TestSettingsPersistence(t *testing.T) {
	// TODO
}
