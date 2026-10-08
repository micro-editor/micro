package main

import (
	"fmt"
	"testing"

	"github.com/micro-editor/micro/v2/internal/action"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/tcell/v2"
)

func TestNestedSplitMouseResize(t *testing.T) {
	newBuffer := func(text string) *buffer.Buffer {
		b := buffer.NewBufferFromString(text, "", buffer.BTDefault)
		b.Settings["backup"] = false
		t.Cleanup(b.Close)
		return b
	}
	tab := action.NewTabFromBuffer(0, 0, 80, 24, newBuffer("first"))
	first := tab.CurPane()
	second := first.VSplitIndex(newBuffer("second"), true)
	third := second.HSplitIndex(newBuffer("third"), true)
	fourth := third.VSplitIndex(newBuffer("fourth"), true)
	activeID := fourth.ID()
	mouse := func(x, y int, button tcell.ButtonMask) {
		tab.HandleEvent(tcell.NewEventMouse(x, y, button, tcell.ModNone, ""))
	}

	// Repeatedly drag the initial divider toward the right screen edge.
	mouse(40, 0, tcell.Button1)
	for x := 41; x < 80; x++ {
		mouse(x, 0, tcell.Button1)
		for _, p := range tab.Panes {
			n := tab.GetNode(p.ID())
			if n == nil || n.W < 1 || n.H < 1 {
				t.Fatalf("mouse drag to %d lost pane %d", x, p.ID())
			}
		}
	}
	mouse(79, 0, tcell.ButtonNone)
	if tab.CurPane().ID() != activeID || len(tab.Panes) != 4 {
		t.Fatal("resize changed focus or pane membership")
	}

	// The nested divider still renders and can be selected with the mouse.
	n := tab.GetNode(third.ID())
	divider := buffer.Loc{X: n.X + n.W, Y: n.Y}
	if tab.GetMouseSplitNode(divider) != n {
		t.Fatal("nested divider is no longer reachable")
	}
	sim.Clear()
	tab.UIWindow.Display()
	if char, _, _, _ := sim.GetContent(divider.X, divider.Y); char != '|' {
		t.Fatalf("nested divider is not visible: %q", char)
	}

	// Drag the parent back: all four panes regain their original width.
	outer := tab.GetNode(first.ID())
	mouse(outer.X+outer.W, 0, tcell.Button1)
	mouse(40, 0, tcell.Button1)
	mouse(40, 0, tcell.ButtonNone)
	if tab.GetNode(third.ID()).W != 20 || tab.GetNode(fourth.ID()).W != 20 {
		t.Fatal("nested panes did not recover their proportions")
	}
	if third.Buf.Line(0) != "third" || fourth.Buf.Line(0) != "fourth" {
		t.Fatal("resize changed buffer contents")
	}
}

func TestNestedSplitResizeKeepsTextVisible(t *testing.T) {
	for _, softwrap := range []bool{false, true} {
		t.Run(fmt.Sprintf("softwrap=%v", softwrap), func(t *testing.T) {
			newBuffer := func(text string) *buffer.Buffer {
				b := buffer.NewBufferFromString(text, "", buffer.BTDefault)
				b.Settings["backup"] = false
				b.Settings["softwrap"] = softwrap
				// Match text entered by the user: the cursor is at the end.
				b.GetActiveCursor().GotoLoc(b.End())
				t.Cleanup(b.Close)
				return b
			}
			tab := action.NewTabFromBuffer(0, 0, 80, 24, newBuffer("first"))
			first := tab.CurPane()
			second := first.VSplitIndex(newBuffer("second"), true)
			third := second.HSplitIndex(newBuffer("third"), true)
			fourth := third.VSplitIndex(newBuffer("fourth"), true)
			panes := []*action.BufPane{first, second, third, fourth}
			texts := []string{"first", "second", "third", "fourth"}
			draw := func() {
				sim.Clear()
				for _, p := range panes {
					p.Display()
				}
				tab.UIWindow.Display()
			}
			checkText := func() {
				t.Helper()
				draw()
				for i, p := range panes {
					if string(p.Buf.Bytes()) != texts[i] {
						t.Fatalf("pane %d buffer contents changed: %q", i, p.Buf.Bytes())
					}
					v := p.BufView()
					var visible []rune
					for x := 0; x < len(texts[i]); x++ {
						char, _, _, _ := sim.GetContent(v.X+x, v.Y)
						visible = append(visible, char)
					}
					if string(visible) != texts[i] {
						t.Errorf("pane %d text is not visible: got %q, want %q; view=%+v cursor=%v", i, string(visible), texts[i], v, p.Cursor.Loc)
					}
				}
			}
			checkText()
			mouse := func(x int, button tcell.ButtonMask) {
				tab.HandleEvent(tcell.NewEventMouse(x, 0, button, tcell.ModNone, ""))
				draw()
			}
			// Shrink the right panes and then the first pane, restoring the
			// original divider position after each drag.
			for _, target := range []int{78, 40, 1, 40} {
				outer := tab.GetNode(first.ID())
				position := outer.X + outer.W
				mouse(position, tcell.Button1)
				step := 1
				if target < position {
					step = -1
				}
				for position != target {
					position += step
					mouse(position, tcell.Button1)
				}
				mouse(position, tcell.ButtonNone)
				if target == 40 {
					checkText()
				}
			}
		})
	}
}
