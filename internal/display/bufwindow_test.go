package display

import (
	"strings"
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/tcell/v2"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

func newSoftwrapTestWindow(t *testing.T, text string, width, height int) *BufWindow {
	t.Helper()

	// NewBufferFromString needs a Lua state to expose the buffer to plugins.
	if ulua.L == nil {
		ulua.L = lua.NewState()
	}

	if err := config.InitGlobalSettings(); err != nil {
		t.Fatalf("InitGlobalSettings: %v", err)
	}
	// Use the real terminal cursor so SimulationScreen.GetCursor() reports it;
	// fakecursor defaults to true on some Windows consoles.
	config.GlobalSettings["fakecursor"] = false
	if _, err := screen.InitSimScreen(); err != nil {
		t.Fatalf("InitSimScreen: %v", err)
	}

	buf := buffer.NewBufferFromString(text, "", buffer.BTDefault)
	buf.Settings["softwrap"] = true
	// Turn off the ruler so gutterOffset is 0 and content width == width.
	buf.Settings["ruler"] = false

	w := NewBufWindow(0, 0, width, height, buf)
	w.Resize(width, height)
	return w
}

// A line that exactly fills the window must not push the next line down by a
// blank wrapped row.
func TestDisplayBufferNoExtraRowAtExactWidth(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		expectedRows int
	}{
		{"one narrower than width: no wrap", 9, 1},
		{"exactly window width: must not wrap to an extra row", 10, 1},
		{"one wider than width: wraps with 1 overflow char", 11, 2},
		{"exactly two window widths: wraps once, no extra row", 20, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := newSoftwrapTestWindow(t, strings.Repeat("x", tc.width)+"\nSECOND", 10, 5)
			w.displayBuffer()

			r, _, _, _ := screen.Screen.GetContent(0, tc.expectedRows)
			assert.Equal(t, 'S', r, "second buffer line should render right after the first line's %d row(s), with no extra blank rows between them", tc.expectedRows)
		})
	}
}

// A cursor at the end of a line that exactly fills the window must still be
// drawn on screen, not past the right edge.
func TestDisplayBufferCursorVisibleAtExactWidthEOL(t *testing.T) {
	tests := []struct {
		name  string
		width int
	}{
		{"one narrower than width", 9},
		{"exactly window width", 10},
		{"one wider than width", 11},
		{"exactly two window widths", 20},
	}

	const winWidth = 10

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := newSoftwrapTestWindow(t, strings.Repeat("x", tc.width)+"\nSECOND", winWidth, 5)
			w.active = true

			c := w.Buf.GetActiveCursor()
			c.GotoLoc(buffer.Loc{X: tc.width, Y: 0})

			w.displayBuffer()

			sim := screen.Screen.(tcell.SimulationScreen)
			cx, cy, vis := sim.GetCursor()

			assert.True(t, vis, "cursor should be visible")
			assert.GreaterOrEqual(t, cx, 0)
			assert.Less(t, cx, winWidth, "cursor x must be within the window, not off the right edge")
			assert.GreaterOrEqual(t, cy, 0)
			assert.Less(t, cy, 5, "cursor y must be within the window")
		})
	}
}
