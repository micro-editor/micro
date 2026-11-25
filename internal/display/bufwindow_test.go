package display

import (
	"testing"

	runewidth "github.com/mattn/go-runewidth"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	lua "github.com/yuin/gopher-lua"
)

func init() {
	ulua.L = lua.NewState()
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
	config.GlobalSettings["backup"] = false
	config.GlobalSettings["fastdirty"] = true
}

// newRelocateWindow builds a BufWindow showing a buffer with softwrap off.
// ruler, diffgutter and scrollbar are off, so bufWidth equals the window
// width. statusline stays on because updateDisplayInfo() touches the global
// screen only when it is off.
func newRelocateWindow(text string, bufWidth, height, startCol int) *BufWindow {
	b := buffer.NewBufferFromString(text, "", buffer.BTDefault)
	b.Settings["softwrap"] = false
	b.Settings["wordwrap"] = false
	b.Settings["ruler"] = false
	b.Settings["diffgutter"] = false
	b.Settings["scrollbar"] = false
	b.Settings["statusline"] = true
	w := NewBufWindow(0, 0, bufWidth, height, b)
	w.updateDisplayInfo()
	w.StartLine = SLoc{0, 0}
	w.StartCol = startCol
	return w
}

// runeWidthAt returns the display width of the rune at loc; a rune that
// takes up no space on its own (a tab, or the virtual newline at the end of
// a line) counts as one column.
func runeWidthAt(b *buffer.Buffer, loc buffer.Loc) int {
	rw := runewidth.RuneWidth(b.RuneAt(loc))
	if rw == 0 {
		rw = 1
	}
	return rw
}

// selectionEndpoints returns the visual X of the leftmost and rightmost
// endpoint of the selection, and the buffer location of the rightmost
// endpoint. Note that for a selection spanning several lines these are the
// visual X values of the endpoints on their own lines.
func selectionEndpoints(w *BufWindow, c *buffer.Cursor) (cxStart, cxEnd int, endLoc buffer.Loc) {
	x1 := w.VLocFromLoc(c.CurSelection[0]).VisualX
	x2 := w.VLocFromLoc(c.CurSelection[1]).VisualX
	if x1 <= x2 {
		return x1, x2, c.CurSelection[1]
	}
	return x2, x1, c.CurSelection[0]
}

// runeVisible reports whether the rune at loc is inside the window. A rune
// that is wider than the window (a double-width rune in a 1 column window)
// cannot be displayed in full, so for those the requirement collapses to
// showing the rune at the left edge.
func runeVisible(w *BufWindow, loc buffer.Loc) bool {
	x := w.VLocFromLoc(loc).VisualX
	rw := runeWidthAt(w.Buf, loc)
	if w.bufWidth >= rw {
		return w.StartCol <= x && x+rw <= w.StartCol+w.bufWidth
	}
	return w.StartCol == x
}

// cursorFullyVisible reports whether the rune at the cursor is inside the
// window, i.e. not cut off at the left edge and, for a double-width rune, not
// split at the right edge.
func cursorFullyVisible(w *BufWindow, c *buffer.Cursor) bool {
	return runeVisible(w, c.Loc)
}

// selectionFitsInWindow reports whether the whole selection can be displayed
// at once in the window.
func selectionFitsInWindow(w *BufWindow, c *buffer.Cursor) bool {
	if !c.HasSelection() {
		return true
	}
	cxStart, cxEnd, endLoc := selectionEndpoints(w, c)
	return cxEnd+runeWidthAt(w.Buf, endLoc)-cxStart <= w.bufWidth
}

// selectionFullyVisible reports whether the entire selection is inside the
// window.
func selectionFullyVisible(w *BufWindow, c *buffer.Cursor) bool {
	if !c.HasSelection() {
		return true
	}
	cxStart, cxEnd, endLoc := selectionEndpoints(w, c)
	return w.StartCol <= cxStart && cxEnd+runeWidthAt(w.Buf, endLoc) <= w.StartCol+w.bufWidth
}

// cursorAtEndpoint reports whether the cursor sits at one of the visual
// endpoints of the selection, which is the case for every selection a core
// action creates (the cursor is at the anchor or at the moving end).
func cursorAtEndpoint(w *BufWindow, c *buffer.Cursor) bool {
	if !c.HasSelection() {
		return true
	}
	cxStart, cxEnd, _ := selectionEndpoints(w, c)
	cursorX := w.VLocFromLoc(c.Loc).VisualX
	return cursorX == cxStart || cursorX == cxEnd
}

// horizontalCase describes a horizontal relocation scenario. Locations are
// character offsets in the buffer.
type horizontalCase struct {
	name     string
	text     string
	bufWidth int
	startCol int
	cursor   buffer.Loc
	// selStart, selEnd are CurSelection[0] and CurSelection[1]. If they are
	// equal there is no selection; if selStart > selEnd the selection is
	// reversed (the user selected backwards).
	selStart  buffer.Loc
	selEnd    buffer.Loc
	wantStart int
	wantMoved bool
}

// runHorizontalCase applies one scenario and checks the result as well as the
// invariants of Relocate() for horizontal scrolling.
func runHorizontalCase(t *testing.T, tc horizontalCase) {
	t.Helper()
	w := newRelocateWindow(tc.text, tc.bufWidth, 5, tc.startCol)
	c := w.Buf.GetActiveCursor()
	c.GotoLoc(tc.cursor)
	c.SetSelectionStart(tc.selStart)
	c.SetSelectionEnd(tc.selEnd)

	moved := w.Relocate()

	if w.StartCol != tc.wantStart {
		t.Errorf("StartCol = %d, want %d", w.StartCol, tc.wantStart)
	}
	if moved != tc.wantMoved {
		t.Errorf("Relocate() returned %v, want %v", moved, tc.wantMoved)
	}
	if !cursorFullyVisible(w, c) {
		t.Errorf("cursor is not fully visible after Relocate()")
	}
	// A selection that fits in the window must be fully visible, as long as
	// the cursor is at a selection endpoint, which is the case for every
	// selection a core action creates. If the cursor is outside the selection
	// (only reachable through the plugin API), Relocate() keeps the cursor
	// visible, which takes precedence.
	if selectionFitsInWindow(w, c) && cursorAtEndpoint(w, c) && !selectionFullyVisible(w, c) {
		t.Errorf("a selection that fits in the window is not fully visible")
	}
}

// TestRelocateHorizontal covers horizontal relocation (scrolling) when softwrap
// is off, i.e. the change of https://github.com/micro-editor/micro/pull/3826.
// Expected behavior:
//   - the cursor is always fully visible,
//   - a selection that fits in the window is fully visible,
//   - a selection wider than the window keeps the cursor's end of the
//     selection visible,
//   - a double-width rune at the end of the selection is not split across
//     the right edge of the window.
func TestRelocateHorizontal(t *testing.T) {
	// "0123456789abcdefghijklmnopqrstuvwxyz"
	// index 0-9 are digits, 10='a' .. 35='z'
	ascii := "0123456789abcdefghijklmnopqrstuvwxyz"

	cases := []horizontalCase{
		{
			name:      "cursor right of the view scrolls it into view",
			text:      ascii,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 15},
			selStart:  buffer.Loc{X: 15},
			selEnd:    buffer.Loc{X: 15},
			wantStart: 6,
			wantMoved: true,
		},
		{
			name:      "cursor left of the view scrolls back",
			text:      ascii,
			bufWidth:  10,
			startCol:  10,
			cursor:    buffer.Loc{X: 2},
			selStart:  buffer.Loc{X: 2},
			selEnd:    buffer.Loc{X: 2},
			wantStart: 2,
			wantMoved: true,
		},
		{
			name:      "a selection that is fully visible does not scroll",
			text:      ascii,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 8},
			selStart:  buffer.Loc{X: 5},
			selEnd:    buffer.Loc{X: 8},
			wantStart: 0,
			wantMoved: false,
		},
		{
			name:      "a selection partially visible on the right scrolls so that its end fits",
			text:      ascii,
			bufWidth:  10,
			startCol:  12,
			cursor:    buffer.Loc{X: 25},
			selStart:  buffer.Loc{X: 20},
			selEnd:    buffer.Loc{X: 25},
			wantStart: 16,
			wantMoved: true,
		},
		{
			name:      "a selection that extends right past the window scrolls fully into view when the cursor is at its start",
			text:      ascii,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 20},
			selStart:  buffer.Loc{X: 20},
			selEnd:    buffer.Loc{X: 25},
			wantStart: 16,
			wantMoved: true,
		},
		{
			name:      "a selection that extends left past the window scrolls fully into view when the cursor is at its end",
			text:      ascii,
			bufWidth:  10,
			startCol:  5,
			cursor:    buffer.Loc{X: 8},
			selStart:  buffer.Loc{X: 0},
			selEnd:    buffer.Loc{X: 8},
			wantStart: 0,
			wantMoved: true,
		},
		{
			name:      "a selection wider than the window keeps its start visible when the cursor is at the start",
			text:      ascii,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 20},
			selStart:  buffer.Loc{X: 20},
			selEnd:    buffer.Loc{X: 30},
			wantStart: 11,
			wantMoved: true,
		},
		{
			name:      "a selection wider than the window keeps its end visible when the cursor is at the end",
			text:      ascii,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 30},
			selStart:  buffer.Loc{X: 20},
			selEnd:    buffer.Loc{X: 30},
			wantStart: 21,
			wantMoved: true,
		},
		{
			name:      "a reversed wide selection keeps its end visible when the cursor is at CurSelection[0]",
			text:      ascii,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 30},
			selStart:  buffer.Loc{X: 30},
			selEnd:    buffer.Loc{X: 20},
			wantStart: 21,
			wantMoved: true,
		},
		{
			name:      "a wide selection scrolls toward the cursor when the cursor is inside it",
			text:      ascii,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 25},
			selStart:  buffer.Loc{X: 20},
			selEnd:    buffer.Loc{X: 30},
			wantStart: 16,
			wantMoved: true,
		},
		{
			name:      "a double-width rune at the end of the selection is not split when the cursor is at the end",
			text:      "一二三四五六七八九十甲乙",
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 6},
			selStart:  buffer.Loc{X: 0},
			selEnd:    buffer.Loc{X: 6},
			wantStart: 4,
			wantMoved: true,
		},
		{
			name:      "a selection wider than the window keeps the start visible when the cursor is at the start",
			text:      "一二三四五六七八九十甲乙",
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 0},
			selStart:  buffer.Loc{X: 0},
			selEnd:    buffer.Loc{X: 6},
			wantStart: 0,
			wantMoved: false,
		},
		{
			name:      "tabs count as tabsize columns",
			text:      "\t\twidth",
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 7},
			selStart:  buffer.Loc{X: 2},
			selEnd:    buffer.Loc{X: 7},
			wantStart: 4,
			wantMoved: true,
		},
		{
			name:      "empty line scrolls back to the beginning",
			text:      "",
			bufWidth:  10,
			startCol:  5,
			cursor:    buffer.Loc{X: 0},
			selStart:  buffer.Loc{X: 0},
			selEnd:    buffer.Loc{X: 0},
			wantStart: 0,
			wantMoved: true,
		},
		{
			name:      "cursor outside the selection on the same line keeps the cursor visible",
			text:      ascii,
			bufWidth:  10,
			startCol:  12,
			cursor:    buffer.Loc{X: 0},
			selStart:  buffer.Loc{X: 5},
			selEnd:    buffer.Loc{X: 8},
			wantStart: 0,
			wantMoved: true,
		},
		{
			name:      "a one column window shows a double-width rune at the left edge",
			text:      "一二三四五六七八九十",
			bufWidth:  1,
			startCol:  5,
			cursor:    buffer.Loc{X: 0},
			selStart:  buffer.Loc{X: 0},
			selEnd:    buffer.Loc{X: 0},
			wantStart: 0,
			wantMoved: true,
		},
		{
			name:      "a one column window keeps the cursor visible next to a selection that does not fit",
			text:      "abcdefghij",
			bufWidth:  1,
			startCol:  2,
			cursor:    buffer.Loc{X: 2},
			selStart:  buffer.Loc{X: 0},
			selEnd:    buffer.Loc{X: 1},
			wantStart: 2,
			wantMoved: false,
		},
		{
			name:      "the cursor range collapses when the cursor rune is wider than the window",
			text:      "一二三四五六七八九十",
			bufWidth:  1,
			startCol:  0,
			cursor:    buffer.Loc{X: 1},
			selStart:  buffer.Loc{X: 0},
			selEnd:    buffer.Loc{X: 1},
			wantStart: 2,
			wantMoved: true,
		},
		{
			name:      "a wide selection does not scroll when the cursor's end is already visible",
			text:      ascii,
			bufWidth:  2,
			startCol:  5,
			cursor:    buffer.Loc{X: 5},
			selStart:  buffer.Loc{X: 0},
			selEnd:    buffer.Loc{X: 5},
			wantStart: 5,
			wantMoved: false,
		},
		{
			name:      "a wide selection does not scroll when the cursor's start is already visible",
			text:      ascii,
			bufWidth:  2,
			startCol:  0,
			cursor:    buffer.Loc{X: 1},
			selStart:  buffer.Loc{X: 1},
			selEnd:    buffer.Loc{X: 3},
			wantStart: 0,
			wantMoved: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runHorizontalCase(t, tc)
		})
	}
}

// TestRelocateHorizontalMultiLine checks that horizontal relocation keeps the
// cursor visible when the selection spans several lines, so its two endpoints
// have visual X coordinates on different lines.
func TestRelocateHorizontalMultiLine(t *testing.T) {
	threeLines := "0123456789\nabcdefghij\nklmnopqrstuv"
	twoLines := "0123456789abcdefghij\nklmnop"

	cases := []horizontalCase{
		{
			name:      "forward selection over several lines scrolls so the cursor's end fits",
			text:      threeLines,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 12, Y: 2},
			selStart:  buffer.Loc{X: 2, Y: 0},
			selEnd:    buffer.Loc{X: 12, Y: 2},
			wantStart: 3,
			wantMoved: true,
		},
		{
			name:      "multi-line selection keeps the cursor visible when the cursor is at the start",
			text:      threeLines,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 2, Y: 0},
			selStart:  buffer.Loc{X: 2, Y: 0},
			selEnd:    buffer.Loc{X: 12, Y: 2},
			wantStart: 0,
			wantMoved: false,
		},
		{
			name:      "reversed multi-line selection keeps the cursor's end visible",
			text:      threeLines,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 12, Y: 2},
			selStart:  buffer.Loc{X: 12, Y: 2},
			selEnd:    buffer.Loc{X: 2, Y: 0},
			wantStart: 3,
			wantMoved: true,
		},
		{
			name:      "cursor on a short line below a long selected line stays visible",
			text:      twoLines,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 5, Y: 1},
			selStart:  buffer.Loc{X: 18, Y: 0},
			selEnd:    buffer.Loc{X: 5, Y: 1},
			wantStart: 0,
			wantMoved: false,
		},
		{
			name:      "cursor on the long selected line scrolls to its own end",
			text:      twoLines,
			bufWidth:  10,
			startCol:  0,
			cursor:    buffer.Loc{X: 18, Y: 0},
			selStart:  buffer.Loc{X: 5, Y: 1},
			selEnd:    buffer.Loc{X: 18, Y: 0},
			wantStart: 9,
			wantMoved: true,
		},
		{
			name:      "a multi-line wide selection does not scroll when the cursor's end is visible",
			text:      twoLines,
			bufWidth:  2,
			startCol:  10,
			cursor:    buffer.Loc{X: 10, Y: 0},
			selStart:  buffer.Loc{X: 0, Y: 0},
			selEnd:    buffer.Loc{X: 10, Y: 0},
			wantStart: 10,
			wantMoved: false,
		},
		{
			name:      "a multi-line wide selection does not scroll when the cursor on a short line is visible",
			text:      twoLines,
			bufWidth:  2,
			startCol:  5,
			cursor:    buffer.Loc{X: 5, Y: 1},
			selStart:  buffer.Loc{X: 0, Y: 0},
			selEnd:    buffer.Loc{X: 5, Y: 1},
			wantStart: 5,
			wantMoved: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newRelocateWindow(tc.text, tc.bufWidth, 5, tc.startCol)
			c := w.Buf.GetActiveCursor()
			c.GotoLoc(tc.cursor)
			c.SetSelectionStart(tc.selStart)
			c.SetSelectionEnd(tc.selEnd)

			moved := w.Relocate()

			if w.StartCol != tc.wantStart {
				t.Errorf("StartCol = %d, want %d", w.StartCol, tc.wantStart)
			}
			if moved != tc.wantMoved {
				t.Errorf("Relocate() returned %v, want %v", moved, tc.wantMoved)
			}

			// The other endpoint is on a different line, so only the cursor
			// is required to be visible; horizontal scroll is shared by all
			// lines and cannot show both endpoints in general.
			if !cursorFullyVisible(w, c) {
				t.Errorf("cursor is not fully visible after Relocate()")
			}
		})
	}
}
