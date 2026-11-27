package action

import (
	"fmt"
	"strings"
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/info"
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

func newMoveLinesPane(t *testing.T, nlines, viewHeight int) *BufPane {
	t.Helper()

	lines := make([]string, nlines)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i)
	}
	buf := buffer.NewBufferFromString(strings.Join(lines, "\n"), "", buffer.BTDefault)
	win := display.NewBufWindow(0, 0, 20, viewHeight+1, buf)
	win.Resize(20, viewHeight+1)
	return newBufPane(buf, win, nil)
}

func selectTo(pane *BufPane, anchor, active buffer.Loc) {
	pane.Cursor.Loc = anchor
	pane.Cursor.OrigSelection[0] = anchor
	pane.Cursor.Loc = active
	pane.Cursor.SelectTo(active)
}

func assertCursorAtActiveEndpoint(t *testing.T, pane *BufPane, endpoint int) {
	t.Helper()

	if pane.Cursor.Loc != pane.Cursor.CurSelection[endpoint] {
		t.Fatalf("cursor = %v, active selection endpoint = %v", pane.Cursor.Loc, pane.Cursor.CurSelection[endpoint])
	}
}

func assertCursorVisible(t *testing.T, pane *BufPane) {
	t.Helper()

	cursor := pane.SLocFromLoc(pane.Cursor.Loc)
	viewStart := pane.GetView().StartLine
	viewEnd := pane.Scroll(viewStart, pane.BufView().Height-1)
	if cursor.LessThan(viewStart) || cursor.GreaterThan(viewEnd) {
		t.Fatalf("cursor %v is outside view %v through %v", cursor, viewStart, viewEnd)
	}
}

func assertLines(t *testing.T, pane *BufPane, want []int) {
	t.Helper()

	for y, line := range want {
		got := string(pane.Buf.LineBytes(y))
		want := fmt.Sprintf("line %d", line)
		if got != want {
			t.Fatalf("line %d = %q, want %q", y, got, want)
		}
	}
}

func TestMoveLinesUpFollowsSelection(t *testing.T) {
	tests := []struct {
		name           string
		anchor         buffer.Loc
		active         buffer.Loc
		activeEndpoint int
		moves          int
		wantSelection  [2]buffer.Loc
	}{
		{
			name:           "top-down selection ending at column zero",
			anchor:         buffer.Loc{X: 0, Y: 6},
			active:         buffer.Loc{X: 0, Y: 9},
			activeEndpoint: 1,
			moves:          4,
			wantSelection:  [2]buffer.Loc{{X: 0, Y: 2}, {X: 0, Y: 5}},
		},
		{
			name:           "bottom-up selection ending within a line",
			anchor:         buffer.Loc{X: 2, Y: 8},
			active:         buffer.Loc{X: 0, Y: 6},
			activeEndpoint: 0,
			moves:          4,
			wantSelection:  [2]buffer.Loc{{X: 0, Y: 2}, {X: 2, Y: 4}},
		},
		{
			name:           "top-down selection ending within a line",
			anchor:         buffer.Loc{X: 0, Y: 6},
			active:         buffer.Loc{X: 2, Y: 8},
			activeEndpoint: 1,
			moves:          4,
			wantSelection:  [2]buffer.Loc{{X: 0, Y: 2}, {X: 2, Y: 4}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newMoveLinesPane(t, 12, 4)
			selectTo(pane, tt.anchor, tt.active)
			pane.GetView().StartLine = display.SLoc{Line: 6}

			for i := 0; i < tt.moves; i++ {
				if !pane.MoveLinesUp() {
					t.Fatalf("move %d failed", i+1)
				}
				assertCursorAtActiveEndpoint(t, pane, tt.activeEndpoint)
				assertCursorVisible(t, pane)
			}

			if pane.Cursor.CurSelection != tt.wantSelection {
				t.Fatalf("selection = %v, want %v", pane.Cursor.CurSelection, tt.wantSelection)
			}
			assertLines(t, pane, []int{0, 1, 6, 7, 8, 2, 3, 4, 5, 9, 10, 11})
		})
	}
}

func TestMoveLinesDownFollowsSelection(t *testing.T) {
	pane := newMoveLinesPane(t, 12, 4)
	selectTo(pane, buffer.Loc{X: 0, Y: 2}, buffer.Loc{X: 0, Y: 5})
	pane.GetView().StartLine = display.SLoc{Line: 2}

	for i := 0; i < 4; i++ {
		if !pane.MoveLinesDown() {
			t.Fatalf("move %d failed", i+1)
		}
		assertCursorAtActiveEndpoint(t, pane, 1)
		assertCursorVisible(t, pane)
	}

	wantSelection := [2]buffer.Loc{{X: 0, Y: 6}, {X: 0, Y: 9}}
	if pane.Cursor.CurSelection != wantSelection {
		t.Fatalf("selection = %v, want %v", pane.Cursor.CurSelection, wantSelection)
	}
	assertLines(t, pane, []int{0, 1, 5, 6, 7, 8, 2, 3, 4, 9, 10, 11})
}

func TestMoveLinesPreservesCursorInsideSelection(t *testing.T) {
	tests := []struct {
		name       string
		move       func(*BufPane) bool
		wantCursor buffer.Loc
	}{
		{
			name:       "up",
			move:       (*BufPane).MoveLinesUp,
			wantCursor: buffer.Loc{X: 6, Y: 3},
		},
		{
			name:       "down",
			move:       (*BufPane).MoveLinesDown,
			wantCursor: buffer.Loc{X: 6, Y: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newMoveLinesPane(t, 8, 4)
			pane.Cursor.Loc = buffer.Loc{X: 6, Y: 4}
			pane.Cursor.SelectLine()

			if !tt.move(pane) {
				t.Fatal("move failed")
			}
			if pane.Cursor.Loc != tt.wantCursor {
				t.Fatalf("cursor = %v, want %v", pane.Cursor.Loc, tt.wantCursor)
			}
		})
	}
}

func TestMoveLinesBoundaries(t *testing.T) {
	oldInfoBar := InfoBar
	InfoBar = &InfoPane{InfoBuf: &info.InfoBuf{}}
	t.Cleanup(func() {
		InfoBar = oldInfoBar
	})

	pane := newMoveLinesPane(t, 4, 3)
	if pane.MoveLinesUp() {
		t.Fatal("MoveLinesUp succeeded on the first line")
	}
	if InfoBar.Msg != "Cannot move further up" {
		t.Fatalf("message = %q, want %q", InfoBar.Msg, "Cannot move further up")
	}

	pane.Cursor.Loc = pane.Buf.End()
	if pane.MoveLinesDown() {
		t.Fatal("MoveLinesDown succeeded on the last line")
	}
	if InfoBar.Msg != "Cannot move further down" {
		t.Fatalf("message = %q, want %q", InfoBar.Msg, "Cannot move further down")
	}
}

func TestMoveLinesUpTallSelectionFollowsActiveEndpoint(t *testing.T) {
	pane := newMoveLinesPane(t, 10, 4)
	selectTo(pane, buffer.Loc{X: 0, Y: 2}, buffer.Loc{X: 0, Y: 7})
	pane.GetView().StartLine = display.SLoc{Line: 5}

	if !pane.MoveLinesUp() {
		t.Fatal("MoveLinesUp failed")
	}

	assertCursorAtActiveEndpoint(t, pane, 1)
	assertCursorVisible(t, pane)
}
