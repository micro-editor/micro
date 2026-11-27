package display

import (
	"strings"
	"testing"
	"time"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/clipboard"
	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/tcell/v2"
	lua "github.com/yuin/gopher-lua"
)

func init() {
	ulua.L = lua.NewState()
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
	config.GlobalSettings["backup"] = false
	config.GlobalSettings["fastdirty"] = true
	config.GlobalSettings["statusline"] = true
	config.GlobalSettings["scrollbar"] = false
	config.GlobalSettings["infobar"] = false
}

func setupTestScreen() {
	buffer.OpenBuffers = nil
	sim := tcell.NewSimulationScreen("UTF-8")
	_ = sim.Init()
	sim.SetSize(80, 24)
	screen.Screen = sim
}

func TestBufWindowRenderStateInitial(t *testing.T) {
	setupTestScreen()
	defer screen.Screen.Fini()

	bufText := "line 1\nline 2\nline 3"
	buf := buffer.NewBufferFromString(bufText, "test.txt", buffer.BTDefault)
	win := NewBufWindow(0, 0, 80, 24, buf)

	rs := win.GetRenderState()

	if rs.X != 0 || rs.Y != 0 || rs.Width != 80 || rs.Height != 24 {
		t.Fatalf("unexpected geometry: %+v", rs)
	}
	if rs.BufLinesNum != 3 {
		t.Fatalf("expected 3 buffer lines, got %d", rs.BufLinesNum)
	}
	if rs.BufModified {
		t.Fatalf("expected buffer to be unmodified initially")
	}
	if !rs.Active {
		t.Fatalf("expected window to be active initially")
	}
	if rs.CursorLoc != (buffer.Loc{X: 0, Y: 0}) {
		t.Fatalf("unexpected initial cursor loc: %+v", rs.CursorLoc)
	}
	if rs.HasSelection {
		t.Fatalf("expected no selection initially")
	}
	if !win.ShouldRedraw() {
		t.Fatalf("expected window to need initial redraw")
	}
}

func TestBufWindowRenderStateEqualityAndDifferences(t *testing.T) {
	setupTestScreen()
	defer screen.Screen.Fini()

	buf := buffer.NewBufferFromString("hello\nworld", "test.txt", buffer.BTDefault)
	win := NewBufWindow(0, 0, 80, 24, buf)

	stateA := win.GetRenderState()
	stateB := win.GetRenderState()
	if stateA != stateB {
		t.Fatalf("identical states should compare equal")
	}

	// Test cursor location sensitivity
	stateCursor := stateA
	stateCursor.CursorLoc = buffer.Loc{X: 3, Y: 1}
	if stateA == stateCursor {
		t.Fatalf("render state should detect cursor movement")
	}

	// Test selection sensitivity
	stateSel := stateA
	stateSel.HasSelection = true
	stateSel.SelStart = buffer.Loc{X: 0, Y: 0}
	stateSel.SelEnd = buffer.Loc{X: 5, Y: 0}
	if stateA == stateSel {
		t.Fatalf("render state should detect selection change")
	}

	// Test viewport scroll sensitivity
	stateScroll := stateA
	stateScroll.StartLine = SLoc{Line: 1, Row: 0}
	if stateA == stateScroll {
		t.Fatalf("render state should detect scroll change")
	}

	// Test buffer modified sensitivity
	stateMod := stateA
	stateMod.BufModified = true
	if stateA == stateMod {
		t.Fatalf("render state should detect buffer modification change")
	}

	// Test decoration active sensitivity
	stateActive := stateA
	stateActive.Active = false
	if stateA == stateActive {
		t.Fatalf("render state should detect active flag change")
	}

	// Test statusline text sensitivity
	stateStatus := stateA
	stateStatus.StatusLineText = "custom-status"
	if stateA == stateStatus {
		t.Fatalf("render state should detect statusline text change")
	}

	// Test scrollbar state sensitivity
	stateSB := stateA
	stateSB.ScrollBarSize = 5
	stateSB.ScrollBarStart = 2
	if stateA == stateSB {
		t.Fatalf("render state should detect scrollbar change")
	}
}

func TestBufWindowRenderStateDisplayGating(t *testing.T) {
	setupTestScreen()
	defer screen.Screen.Fini()

	buf := buffer.NewBufferFromString("alpha\nbeta\ngamma", "test.txt", buffer.BTDefault)
	win := NewBufWindow(0, 0, 80, 24, buf)

	if !win.ShouldRedraw() {
		t.Fatalf("expected initial ShouldRedraw to be true")
	}

	win.Display()

	if win.ShouldRedraw() {
		t.Fatalf("expected ShouldRedraw to be false after Display() with no state changes")
	}

	// Second Display() call should be gated and retain identical render state
	prevRenderState := win.LastRenderState()
	win.Display()
	if win.LastRenderState() != prevRenderState {
		t.Fatalf("render state changed unexpectedly on redundant display")
	}

	// Invalidate forces redraw
	win.Invalidate()
	if !win.ShouldRedraw() {
		t.Fatalf("expected ShouldRedraw to be true after Invalidate()")
	}

	win.Display()
	if win.ShouldRedraw() {
		t.Fatalf("expected ShouldRedraw to be false after re-rendering")
	}
}

func TestBufWindowRenderStateBufferEdits(t *testing.T) {
	setupTestScreen()
	defer screen.Screen.Fini()

	buf := buffer.NewBufferFromString("first line\nsecond line", "test.txt", buffer.BTDefault)
	win := NewBufWindow(0, 0, 80, 24, buf)

	win.Display()
	if win.ShouldRedraw() {
		t.Fatalf("window should not need redraw immediately after display")
	}

	// Edit buffer
	buf.Insert(buffer.Loc{X: 0, Y: 0}, "inserted ")
	if !buf.Modified() {
		t.Fatalf("buffer should be marked modified after insert")
	}

	if !win.ShouldRedraw() {
		t.Fatalf("window should need redraw after buffer edit")
	}

	win.Display()
	if win.ShouldRedraw() {
		t.Fatalf("window should not need redraw after displaying edited buffer")
	}
	if !win.LastRenderState().BufModified {
		t.Fatalf("render state should record BufModified=true")
	}
}

func TestBufWindowRenderStateClipboardWithoutBufferEdits(t *testing.T) {
	setupTestScreen()
	defer screen.Screen.Fini()

	buf := buffer.NewBufferFromString("clipboard test text", "test.txt", buffer.BTDefault)
	win := NewBufWindow(0, 0, 80, 24, buf)

	win.Display()
	if win.ShouldRedraw() {
		t.Fatalf("window should not need redraw immediately after display")
	}

	// Perform external clipboard operations without editing the buffer
	clipboard.ResetCache()
	err := clipboard.Write("clipboard text without edits", clipboard.ClipboardReg)
	if err != nil {
		t.Fatalf("clipboard write error: %v", err)
	}

	// Verify buffer is untouched
	if buf.Modified() {
		t.Fatalf("buffer should not be modified by external clipboard sync")
	}

	// Verify buffer window rendering state gates redundant redrawing
	if win.ShouldRedraw() {
		t.Fatalf("redundant redrawing should be gated when clipboard changes occur without buffer edits")
	}

	// Additional clipboard operation
	err = clipboard.Write("another external clipboard payload", clipboard.PrimaryReg)
	if err != nil {
		t.Fatalf("primary clipboard write error: %v", err)
	}

	if buf.Modified() {
		t.Fatalf("buffer should remain unmodified")
	}
	if win.ShouldRedraw() {
		t.Fatalf("buffer window redraw should remain gated")
	}
}

func TestBufWindowRenderStateWindowDecorations(t *testing.T) {
	setupTestScreen()
	defer screen.Screen.Fini()

	buf := buffer.NewBufferFromString(strings.Repeat("line\n", 50), "test.txt", buffer.BTDefault)
	win := NewBufWindow(0, 0, 80, 24, buf)

	win.Display()
	if win.ShouldRedraw() {
		t.Fatalf("window should not need redraw immediately after display")
	}

	// Active window decoration change
	win.SetActive(false)
	if !win.ShouldRedraw() {
		t.Fatalf("window should need redraw after active state changed")
	}
	win.Display()
	if win.ShouldRedraw() {
		t.Fatalf("window should not need redraw after display")
	}

	// Resize window decoration change
	win.Resize(100, 30)
	if !win.ShouldRedraw() {
		t.Fatalf("window should need redraw after resize")
	}
	win.Display()
	if win.ShouldRedraw() {
		t.Fatalf("window should not need redraw after display")
	}
}

func TestBufWindowRenderStateDebounce(t *testing.T) {
	setupTestScreen()
	defer screen.Screen.Fini()

	buf := buffer.NewBufferFromString("debounce line 1\ndebounce line 2", "test.txt", buffer.BTDefault)
	win := NewBufWindow(0, 0, 80, 24, buf)

	win.SetRedrawDebounce(40 * time.Millisecond)
	win.Display()

	// Immediately calling Display() when buffer is not modified and within debounce duration
	win.Display()
	if win.ShouldRedraw() {
		t.Fatalf("redundant redraw should remain gated during debounce window")
	}

	time.Sleep(50 * time.Millisecond)
	if win.ShouldRedraw() {
		t.Fatalf("unmodified buffer without state changes should still not require redraw after sleep")
	}
}
