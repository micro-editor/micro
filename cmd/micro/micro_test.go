package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/go-errors/errors"
	"github.com/micro-editor/micro/v2/internal/action"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/micro-editor/tcell/v2"
	"github.com/stretchr/testify/assert"
)

var tempDir string
var sim tcell.SimulationScreen

func init() {
	screen.Events = make(chan tcell.Event, 8)
}

func startup(args []string) (tcell.SimulationScreen, error) {
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

	s, err := screen.InitSimScreen()
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
			// Print the stack trace too
			log.Fatalf(errors.Wrap(err, 2).ErrorStack())
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

	s.InjectResize()
	handleEvent()

	return s, nil
}

func cleanup() {
	os.RemoveAll(tempDir)
}

func handleEvent() {
	screen.Lock()
	e := screen.Screen.PollEvent()
	screen.Unlock()
	if e != nil {
		screen.Events <- e
	}

	for len(screen.DrawChan()) > 0 || len(screen.Events) > 0 {
		DoEvent()
	}
}

func injectKey(key tcell.Key, r rune, mod tcell.ModMask) {
	sim.InjectKey(key, r, mod)
	handleEvent()
}

func injectMouse(x, y int, buttons tcell.ButtonMask, mod tcell.ModMask) {
	sim.InjectMouse(x, y, buttons, mod)
	handleEvent()
}

func injectString(str string) {
	// the tcell simulation screen event channel can only handle
	// 10 events at once, so we need to divide up the key events
	// into chunks of 10 and handle the 10 events before sending
	// another chunk of events
	iters := len(str) / 10
	extra := len(str) % 10

	for i := 0; i < iters; i++ {
		s := i * 10
		e := i*10 + 10
		sim.InjectKeyBytes([]byte(str[s:e]))
		for i := 0; i < 10; i++ {
			handleEvent()
		}
	}

	sim.InjectKeyBytes([]byte(str[len(str)-extra:]))
	for i := 0; i < extra; i++ {
		handleEvent()
	}
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
	sim, err = startup([]string{})
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

	// test both kinds of backspace
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

func TestSoftwrapScroll(t *testing.T) {
	// lines of 1, 3, 6 and 8 rows, so rows and lines don't match up
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&sb, "%d %s\n", i, strings.Repeat("word ", i%4*40))
	}
	text := sb.String()
	openFile(createTestFile(t, text))
	h := action.MainTab().CurPane()
	h.Buf.SetOptionNative("softwrap", true)
	height := h.BufView().Height
	end := h.SLocFromLoc(h.Buf.End())
	lastPage := h.Scroll(end, -height+1)

	// check against Diff() for every StartLine near the end
	v := h.GetView()
	for s := h.Scroll(end, -3*height); s != end; s = h.Scroll(s, 1) {
		v.StartLine = s
		assert.Equal(t, h.Diff(s, end) < height, h.ScrollReachedEnd(), "ScrollReachedEnd at %v", s)
		want := s
		if h.Diff(s, end) < height-1 {
			want = lastPage
		}
		h.ScrollAdjust()
		assert.Equal(t, want, h.GetView().StartLine, "ScrollAdjust at %v", s)
	}

	// PageDown stops with the last line at the bottom of the view
	h.GotoLoc(buffer.Loc{X: 0, Y: 0})
	for i := 0; i < 50; i++ {
		h.PageDown()
	}
	assert.Equal(t, lastPage, h.GetView().StartLine)

	// GotoLoc recenters only when the cursor moves by at least a view height
	from := h.Scroll(end, -2*height)
	for _, d := range []int{height - 1, height, 1 - height, -height} {
		h.GotoLoc(h.LocFromVLoc(display.VLoc{SLoc: from}))
		sloc := h.Scroll(from, d)
		h.GotoLoc(h.LocFromVLoc(display.VLoc{SLoc: sloc}))
		recentered := h.GetView().StartLine == h.Scroll(sloc, -height/4)
		assert.Equal(t, util.Abs(d) >= height, recentered, "GotoLoc by %d rows", d)
	}

	// opening a buffer recenters only when the cursor is not on the first page
	for _, d := range []int{height - 1, height} {
		sloc := h.Scroll(display.SLoc{}, d)
		b := buffer.NewBufferFromString(text, "", buffer.BTDefault)
		b.SetOptionNative("softwrap", true)
		b.GetActiveCursor().GotoLoc(h.LocFromVLoc(display.VLoc{SLoc: sloc}))
		h.OpenBuffer(b)
		recentered := h.GetView().StartLine == h.Scroll(sloc, -height/4)
		assert.Equal(t, d >= height, recentered, "open at row %d", d)
	}
}

func TestMultiCursor(t *testing.T) {
	// TODO
}

func TestSettingsPersistence(t *testing.T) {
	// TODO
}

// more tests (rendering, tabs, plugins)?
