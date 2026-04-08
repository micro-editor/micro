package action

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	shellquote "github.com/kballard/go-shellquote"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/clipboard"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/shell"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/micro-editor/tcell/v2"
)

// ScrollUp is not an action
func (h *BufPane) ScrollUp(n int) {
	v := h.GetView()
	v.StartLine = h.Scroll(v.StartLine, -n)
	h.SetView(v)
}

// ScrollDown is not an action
func (h *BufPane) ScrollDown(n int) {
	v := h.GetView()
	v.StartLine = h.Scroll(v.StartLine, n)
	h.SetView(v)
}

// ScrollAdjust can be used to shift the view so that the last line is at the
// bottom if the user has scrolled past the last line.
func (h *BufPane) ScrollAdjust() {
	v := h.GetView()
	end := h.SLocFromLoc(h.Buf.End())
	if h.Diff(v.StartLine, end) < h.BufView().Height-1 {
		v.StartLine = h.Scroll(end, -h.BufView().Height+1)
	}
	h.SetView(v)
}

// ScrollReachedEnd returns true if the view is at the end of the buffer,
// i.e. the last line of the buffer is in the view.
func (h *BufPane) ScrollReachedEnd() bool {
	v := h.GetView()
	end := h.SLocFromLoc(h.Buf.End())
	return h.Diff(v.StartLine, end) < h.BufView().Height
}

// MousePress is the event that should happen when a normal click happens
// This is almost always bound to left click
func (h *BufPane) MousePress(e *tcell.EventMouse) bool {
	b := h.Buf
	mx, my := e.Position()
	// ignore click on the status line
	if my >= h.BufView().Y+h.BufView().Height {
		return false
	}
	mouseLoc := h.LocFromVisual(buffer.Loc{mx, my})
	h.Cursor.Loc = mouseLoc

	if b.NumCursors() > 1 {
		b.ClearCursors()
		h.Relocate()
		h.Cursor = h.Buf.GetActiveCursor()
		h.Cursor.Loc = mouseLoc
	}
	if time.Since(h.lastClickTime)/time.Millisecond < config.DoubleClickThreshold && (mouseLoc.X == h.lastLoc.X && mouseLoc.Y == h.lastLoc.Y) {
		if h.DoubleClick {
			// Triple click
			h.lastClickTime = time.Now()

			h.TripleClick = true
			h.DoubleClick = false

			h.Cursor.SelectLine()
			h.Cursor.CopySelection(clipboard.PrimaryReg)
		} else {
			// Double click
			h.lastClickTime = time.Now()

			h.DoubleClick = true
			h.TripleClick = false

			h.Cursor.SelectWord()
			h.Cursor.CopySelection(clipboard.PrimaryReg)
		}
	} else {
		h.DoubleClick = false
		h.TripleClick = false
		h.lastClickTime = time.Now()

		h.Cursor.OrigSelection[0] = h.Cursor.Loc
		h.Cursor.CurSelection[0] = h.Cursor.Loc
		h.Cursor.CurSelection[1] = h.Cursor.Loc
	}

	h.Cursor.StoreVisualX()
	h.lastLoc = mouseLoc
	h.Relocate()
	return true
}

func (h *BufPane) MouseDrag(e *tcell.EventMouse) bool {
	mx, my := e.Position()
	// ignore drag on the status line
	if my >= h.BufView().Y+h.BufView().Height {
		return false
	}
	h.Cursor.Loc = h.LocFromVisual(buffer.Loc{mx, my})

	if h.TripleClick {
		h.Cursor.AddLineToSelection()
	} else if h.DoubleClick {
		h.Cursor.AddWordToSelection()
	} else {
		h.Cursor.SelectTo(h.Cursor.Loc)
	}

	h.Cursor.StoreVisualX()
	h.Relocate()
	return true
}

func (h *BufPane) MouseRelease(e *tcell.EventMouse) bool {
	// We could finish the selection based on the release location as in the
	// commented out code below, to allow text selections even in a terminal
	// that doesn't support mouse motion events. But when the mouse click is
	// within the scroll margin, that would cause a scroll and selection
	// even for a simple mouse click, which is not good.
	// if !h.DoubleClick && !h.TripleClick {
	// 	mx, my := e.Position()
	// 	h.Cursor.Loc = h.LocFromVisual(buffer.Loc{mx, my})
	// 	h.Cursor.SetSelectionEnd(h.Cursor.Loc)
	// }

	if h.Cursor.HasSelection() {
		h.Cursor.CopySelection(clipboard.PrimaryReg)
	}
	return true
}

// ScrollUpAction scrolls the view up
func (h *BufPane) ScrollUpAction() bool {
	h.ScrollUp(util.IntOpt(h.Buf.Settings["scrollspeed"]))
	return true
}

// ScrollDownAction scrolls the view down
func (h *BufPane) ScrollDownAction() bool {
	h.ScrollDown(util.IntOpt(h.Buf.Settings["scrollspeed"]))
	return true
}

// Center centers the view on the cursor
func (h *BufPane) Center() bool {
	v := h.GetView()
	v.StartLine = h.Scroll(h.SLocFromLoc(h.Cursor.Loc), -h.BufView().Height/2)
	h.SetView(v)
	h.ScrollAdjust()
	return true
}

// CursorToViewTop moves the cursor to the top of the view,
// offset by scrollmargin unless at the beginning or end of the file
func (h *BufPane) CursorToViewTop() bool {
	v := h.GetView()
	h.Buf.ClearCursors()
	scrollmargin := int(h.Buf.Settings["scrollmargin"].(float64))
	bStart := display.SLoc{0, 0}
	if v.StartLine == bStart {
		scrollmargin = 0
	}
	h.Cursor.GotoLoc(h.LocFromVLoc(display.VLoc{
		SLoc:    h.Scroll(v.StartLine, scrollmargin),
		VisualX: 0,
	}))
	return true
}

// CursorToViewCenter moves the cursor to the center of the view
func (h *BufPane) CursorToViewCenter() bool {
	v := h.GetView()
	h.Buf.ClearCursors()
	h.Cursor.GotoLoc(h.LocFromVLoc(display.VLoc{
		SLoc:    h.Scroll(v.StartLine, h.BufView().Height/2),
		VisualX: 0,
	}))
	return true
}

// CursorToViewBottom moves the cursor to the bottom of the view,
// offset by scrollmargin unless at the beginning or end of the file
func (h *BufPane) CursorToViewBottom() bool {
	v := h.GetView()
	h.Buf.ClearCursors()
	scrollmargin := int(h.Buf.Settings["scrollmargin"].(float64))
	bEnd := h.SLocFromLoc(h.Buf.End())
	lastLine := h.Scroll(v.StartLine, h.BufView().Height-1)
	if lastLine == bEnd {
		scrollmargin = 0
	}
	h.Cursor.GotoLoc(h.LocFromVLoc(display.VLoc{
		SLoc:    h.Scroll(lastLine, -scrollmargin),
		VisualX: 0,
	}))
	return true
}

// MoveCursorUp is not an action
func (h *BufPane) MoveCursorUp(n int) {
	if !h.Buf.Settings["softwrap"].(bool) {
		h.Cursor.UpN(n)
	} else {
		vloc := h.VLocFromLoc(h.Cursor.Loc)
		sloc := h.Scroll(vloc.SLoc, -n)
		if sloc == vloc.SLoc {
			// we are at the beginning of buffer
			h.Cursor.Loc = h.Buf.Start()
			h.Cursor.StoreVisualX()
		} else {
			vloc.SLoc = sloc
			vloc.VisualX = h.Cursor.LastWrappedVisualX
			h.Cursor.Loc = h.LocFromVLoc(vloc)
		}
	}
}

// MoveCursorDown is not an action
func (h *BufPane) MoveCursorDown(n int) {
	if !h.Buf.Settings["softwrap"].(bool) {
		h.Cursor.DownN(n)
	} else {
		vloc := h.VLocFromLoc(h.Cursor.Loc)
		sloc := h.Scroll(vloc.SLoc, n)
		if sloc == vloc.SLoc {
			// we are at the end of buffer
			h.Cursor.Loc = h.Buf.End()
			h.Cursor.StoreVisualX()
		} else {
			vloc.SLoc = sloc
			vloc.VisualX = h.Cursor.LastWrappedVisualX
			h.Cursor.Loc = h.LocFromVLoc(vloc)
		}
	}
}

// CursorUp moves the cursor up
func (h *BufPane) CursorUp() bool {
	h.Cursor.Deselect(true)
	h.MoveCursorUp(1)
	h.Relocate()
	return true
}

// CursorDown moves the cursor down
func (h *BufPane) CursorDown() bool {
	selectionEndNewline := h.Cursor.HasSelection() && h.Cursor.CurSelection[1].X == 0
	h.Cursor.Deselect(false)
	if selectionEndNewline {
		h.Cursor.Start()
	} else {
		h.MoveCursorDown(1)
	}
	h.Relocate()
	return true
}

// CursorLeft moves the cursor left
func (h *BufPane) CursorLeft() bool {
	if h.Cursor.HasSelection() {
		h.Cursor.Deselect(true)
	} else {
		tabstospaces := h.Buf.Settings["tabstospaces"].(bool)
		tabmovement := h.Buf.Settings["tabmovement"].(bool)
		if tabstospaces && tabmovement {
			tabsize := int(h.Buf.Settings["tabsize"].(float64))
			line := h.Buf.LineBytes(h.Cursor.Y)
			if h.Cursor.X-tabsize >= 0 && util.IsSpaces(line[h.Cursor.X-tabsize:h.Cursor.X]) && util.IsBytesWhitespace(line[0:h.Cursor.X-tabsize]) {
				for i := 0; i < tabsize; i++ {
					h.Cursor.Left()
				}
			} else {
				h.Cursor.Left()
			}
		} else {
			h.Cursor.Left()
		}
	}
	h.Relocate()
	return true
}

// CursorRight moves the cursor right
func (h *BufPane) CursorRight() bool {
	if h.Cursor.HasSelection() {
		h.Cursor.Deselect(false)
	} else {
		tabstospaces := h.Buf.Settings["tabstospaces"].(bool)
		tabmovement := h.Buf.Settings["tabmovement"].(bool)
		if tabstospaces && tabmovement {
			tabsize := int(h.Buf.Settings["tabsize"].(float64))
			line := h.Buf.LineBytes(h.Cursor.Y)
			if h.Cursor.X+tabsize < util.CharacterCount(line) && util.IsSpaces(line[h.Cursor.X:h.Cursor.X+tabsize]) && util.IsBytesWhitespace(line[0:h.Cursor.X]) {
				for i := 0; i < tabsize; i++ {
					h.Cursor.Right()
				}
			} else {
				h.Cursor.Right()
			}
		} else {
			h.Cursor.Right()
		}
	}

	h.Relocate()
	return true
}

// WordRight moves the cursor one word to the right
func (h *BufPane) WordRight() bool {
	h.Cursor.Deselect(false)
	h.Cursor.WordRight()
	h.Relocate()
	return true
}

// WordLeft moves the cursor one word to the left
func (h *BufPane) WordLeft() bool {
	h.Cursor.Deselect(true)
	h.Cursor.WordLeft()
	h.Relocate()
	return true
}

// SubWordRight moves the cursor one sub-word to the right
func (h *BufPane) SubWordRight() bool {
	h.Cursor.Deselect(false)
	h.Cursor.SubWordRight()
	h.Relocate()
	return true
}

// SubWordLeft moves the cursor one sub-word to the left
func (h *BufPane) SubWordLeft() bool {
	h.Cursor.Deselect(true)
	h.Cursor.SubWordLeft()
	h.Relocate()
	return true
}

// SelectUp selects up one line
func (h *BufPane) SelectUp() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.MoveCursorUp(1)
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectDown selects down one line
func (h *BufPane) SelectDown() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.MoveCursorDown(1)
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectLeft selects the character to the left of the cursor
func (h *BufPane) SelectLeft() bool {
	loc := h.Cursor.Loc
	count := h.Buf.End()
	if loc.GreaterThan(count) {
		loc = count
	}
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = loc
	}
	h.Cursor.Left()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectRight selects the character to the right of the cursor
func (h *BufPane) SelectRight() bool {
	loc := h.Cursor.Loc
	count := h.Buf.End()
	if loc.GreaterThan(count) {
		loc = count
	}
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = loc
	}
	h.Cursor.Right()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectWordRight selects the word to the right of the cursor
func (h *BufPane) SelectWordRight() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.Cursor.WordRight()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectWordLeft selects the word to the left of the cursor
func (h *BufPane) SelectWordLeft() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.Cursor.WordLeft()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectSubWordRight selects the sub-word to the right of the cursor
func (h *BufPane) SelectSubWordRight() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.Cursor.SubWordRight()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectSubWordLeft selects the sub-word to the left of the cursor
func (h *BufPane) SelectSubWordLeft() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.Cursor.SubWordLeft()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// StartOfText moves the cursor to the start of the text of the line
func (h *BufPane) StartOfText() bool {
	h.Cursor.Deselect(true)
	h.Cursor.StartOfText()
	h.Relocate()
	return true
}

// StartOfTextToggle toggles the cursor between the start of the text of the line
// and the start of the line
func (h *BufPane) StartOfTextToggle() bool {
	h.Cursor.Deselect(true)
	if h.Cursor.IsStartOfText() {
		h.Cursor.Start()
	} else {
		h.Cursor.StartOfText()
	}
	h.Relocate()
	return true
}

// StartOfLine moves the cursor to the start of the line
func (h *BufPane) StartOfLine() bool {
	h.Cursor.Deselect(true)
	h.Cursor.Start()
	h.Relocate()
	return true
}

// EndOfLine moves the cursor to the end of the line
func (h *BufPane) EndOfLine() bool {
	h.Cursor.Deselect(true)
	h.Cursor.End()
	h.Relocate()
	return true
}

// SelectLine selects the entire current line
func (h *BufPane) SelectLine() bool {
	h.Cursor.SelectLine()
	h.Relocate()
	return true
}

// SelectToStartOfText selects to the start of the text on the current line
func (h *BufPane) SelectToStartOfText() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.Cursor.StartOfText()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectToStartOfTextToggle toggles the selection between the start of the text
// on the current line and the start of the line
func (h *BufPane) SelectToStartOfTextToggle() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	if h.Cursor.IsStartOfText() {
		h.Cursor.Start()
	} else {
		h.Cursor.StartOfText()
	}
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectToStartOfLine selects to the start of the current line
func (h *BufPane) SelectToStartOfLine() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.Cursor.Start()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectToEndOfLine selects to the end of the current line
func (h *BufPane) SelectToEndOfLine() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.Cursor.End()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

func (h *BufPane) paragraphPrevious() {
	var line int
	// Skip to the first non-empty line
	for line = h.Cursor.Y; line > 0; line-- {
		if len(h.Buf.LineBytes(line)) != 0 {
			break
		}
	}
	// Find the first empty line
	for ; line > 0; line-- {
		if len(h.Buf.LineBytes(line)) == 0 {
			h.Cursor.X = 0
			h.Cursor.Y = line
			break
		}
	}
	// If no empty line was found, move the cursor to the start of the buffer
	if line == 0 {
		h.Cursor.Loc = h.Buf.Start()
	}
}

func (h *BufPane) paragraphNext() {
	var line int
	// Skip to the first non-empty line
	for line = h.Cursor.Y; line < h.Buf.LinesNum(); line++ {
		if len(h.Buf.LineBytes(line)) != 0 {
			break
		}
	}
	// Find the first empty line
	for ; line < h.Buf.LinesNum(); line++ {
		if len(h.Buf.LineBytes(line)) == 0 {
			h.Cursor.X = 0
			h.Cursor.Y = line
			break
		}
	}
	// If no empty line was found, move the cursor to the end of the buffer
	if line == h.Buf.LinesNum() {
		h.Cursor.Loc = h.Buf.End()
	}
}

// ParagraphPrevious moves the cursor to the first empty line that comes before
// the paragraph closest to the cursor, or beginning of the buffer if there
// isn't a paragraph
func (h *BufPane) ParagraphPrevious() bool {
	h.Cursor.Deselect(true)
	h.paragraphPrevious()
	h.Relocate()
	return true
}

// ParagraphNext moves the cursor to the first empty line that comes after the
// paragraph closest to the cursor, or end of the buffer if there isn't a
// paragraph
func (h *BufPane) ParagraphNext() bool {
	h.Cursor.Deselect(true)
	h.paragraphNext()
	h.Relocate()
	return true
}

// SelectToParagraphPrevious selects to the first empty line that comes before
// the paragraph closest to the cursor, or beginning of the buffer if there
// isn't a paragraph
func (h *BufPane) SelectToParagraphPrevious() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.paragraphPrevious()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// SelectToParagraphNext selects to the first empty line that comes after the
// paragraph closest to the cursor, or end of the buffer if there isn't a
// paragraph
func (h *BufPane) SelectToParagraphNext() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.paragraphNext()
	h.Cursor.SelectTo(h.Cursor.Loc)
	h.Relocate()
	return true
}

// Retab changes all tabs to spaces or all spaces to tabs depending
// on the user's settings
func (h *BufPane) Retab() bool {
	h.Buf.Retab()
	h.Relocate()
	return true
}

// CursorStart moves the cursor to the start of the buffer
func (h *BufPane) CursorStart() bool {
	h.Cursor.Deselect(true)
	h.Cursor.X = 0
	h.Cursor.Y = 0
	h.Cursor.StoreVisualX()
	h.Relocate()
	return true
}

// CursorEnd moves the cursor to the end of the buffer
func (h *BufPane) CursorEnd() bool {
	h.Cursor.Deselect(true)
	h.Cursor.Loc = h.Buf.End()
	h.Cursor.StoreVisualX()
	h.Relocate()
	return true
}

// SelectToStart selects the text from the cursor to the start of the buffer
func (h *BufPane) SelectToStart() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.CursorStart()
	h.Cursor.SelectTo(h.Buf.Start())
	h.Relocate()
	return true
}

// SelectToEnd selects the text from the cursor to the end of the buffer
func (h *BufPane) SelectToEnd() bool {
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.CursorEnd()
	h.Cursor.SelectTo(h.Buf.End())
	h.Relocate()
	return true
}

// InsertNewline inserts a newline plus possible some whitespace if autoindent is on
func (h *BufPane) InsertNewline() bool {
	// Insert a newline
	if h.Cursor.HasSelection() {
		h.Cursor.DeleteSelection()
		h.Cursor.ResetSelection()
	}

	ws := util.GetLeadingWhitespace(h.Buf.LineBytes(h.Cursor.Y))
	cx := h.Cursor.X
	h.Buf.Insert(h.Cursor.Loc, "\n")
	// h.Cursor.Right()

	if h.Buf.Settings["autoindent"].(bool) {
		if cx < len(ws) {
			ws = ws[0:cx]
		}
		h.Buf.Insert(h.Cursor.Loc, string(ws))
		// for i := 0; i < len(ws); i++ {
		// 	h.Cursor.Right()
		// }

		// Remove the whitespaces if keepautoindent setting is off
		if util.IsSpacesOrTabs(h.Buf.LineBytes(h.Cursor.Y-1)) && !h.Buf.Settings["keepautoindent"].(bool) {
			line := h.Buf.LineBytes(h.Cursor.Y - 1)
			h.Buf.Remove(buffer.Loc{X: 0, Y: h.Cursor.Y - 1}, buffer.Loc{X: util.CharacterCount(line), Y: h.Cursor.Y - 1})
		}
	}
	h.Cursor.StoreVisualX()
	h.Relocate()
	return true
}

// Backspace deletes the previous character
func (h *BufPane) Backspace() bool {
	if h.Cursor.HasSelection() {
		h.Cursor.DeleteSelection()
		h.Cursor.ResetSelection()
	} else if h.Cursor.Loc.GreaterThan(h.Buf.Start()) {
		// We have to do something a bit hacky here because we want to
		// delete the line by first moving left and then deleting backwards
		// but the undo redo would place the cursor in the wrong place
		// So instead we move left, save the position, move back, delete
		// and restore the position

		// If the user is using spaces instead of tabs and they are deleting
		// whitespace at the start of the line, we should delete as if it's a
		// tab (tabSize number of spaces)
		lineStart := util.SliceStart(h.Buf.LineBytes(h.Cursor.Y), h.Cursor.X)
		tabSize := int(h.Buf.Settings["tabsize"].(float64))
		if h.Buf.Settings["tabstospaces"].(bool) && util.IsSpaces(lineStart) && len(lineStart) != 0 && util.CharacterCount(lineStart)%tabSize == 0 {
			loc := h.Cursor.Loc
			h.Buf.Remove(loc.Move(-tabSize, h.Buf), loc)
		} else {
			loc := h.Cursor.Loc
			h.Buf.Remove(loc.Move(-1, h.Buf), loc)
		}
	}
	h.Cursor.StoreVisualX()
	h.Relocate()
	return true
}

// DeleteWordRight deletes the word to the right of the cursor
func (h *BufPane) DeleteWordRight() bool {
	h.SelectWordRight()
	if h.Cursor.HasSelection() {
		h.Cursor.DeleteSelection()
		h.Cursor.ResetSelection()
	}
	h.Relocate()
	return true
}

// DeleteWordLeft deletes the word to the left of the cursor
func (h *BufPane) DeleteWordLeft() bool {
	h.SelectWordLeft()
	if h.Cursor.HasSelection() {
		h.Cursor.DeleteSelection()
		h.Cursor.ResetSelection()
	}
	h.Relocate()
	return true
}

// DeleteSubWordRight deletes the sub-word to the right of the cursor
func (h *BufPane) DeleteSubWordRight() bool {
	h.SelectSubWordRight()
	if h.Cursor.HasSelection() {
		h.Cursor.DeleteSelection()
		h.Cursor.ResetSelection()
	}
	h.Relocate()
	return true
}

// DeleteSubWordLeft deletes the sub-word to the left of the cursor
func (h *BufPane) DeleteSubWordLeft() bool {
	h.SelectSubWordLeft()
	if h.Cursor.HasSelection() {
		h.Cursor.DeleteSelection()
		h.Cursor.ResetSelection()
	}
	h.Relocate()
	return true
}

// Delete deletes the next character
func (h *BufPane) Delete() bool {
	if h.Cursor.HasSelection() {
		h.Cursor.DeleteSelection()
		h.Cursor.ResetSelection()
	} else {
		loc := h.Cursor.Loc
		if loc.LessThan(h.Buf.End()) {
			h.Buf.Remove(loc, loc.Move(1, h.Buf))
		}
	}
	h.Relocate()
	return true
}

// IndentSelection indents the current selection
func (h *BufPane) IndentSelection() bool {
	if h.Cursor.HasSelection() {
		start := h.Cursor.CurSelection[0]
		end := h.Cursor.CurSelection[1]
		if end.Y < start.Y {
			start, end = end, start
			h.Cursor.SetSelectionStart(start)
			h.Cursor.SetSelectionEnd(end)
		}

		startY := start.Y
		endY := end.Move(-1, h.Buf).Y
		endX := end.Move(-1, h.Buf).X
		tabsize := int(h.Buf.Settings["tabsize"].(float64))
		indentsize := len(h.Buf.IndentString(tabsize))
		for y := startY; y <= endY; y++ {
			if len(h.Buf.LineBytes(y)) > 0 {
				h.Buf.Insert(buffer.Loc{X: 0, Y: y}, h.Buf.IndentString(tabsize))
				if y == startY && start.X > 0 {
					h.Cursor.SetSelectionStart(start.Move(indentsize, h.Buf))
				}
				if y == endY {
					h.Cursor.SetSelectionEnd(buffer.Loc{X: endX + indentsize + 1, Y: endY})
				}
			}
		}
		h.Buf.RelocateCursors()

		h.Relocate()
		return true
	}
	return false
}

// IndentLine moves the current line forward one indentation
func (h *BufPane) IndentLine() bool {
	if h.Cursor.HasSelection() {
		return false
	}

	tabsize := int(h.Buf.Settings["tabsize"].(float64))
	indentstr := h.Buf.IndentString(tabsize)
	h.Buf.Insert(buffer.Loc{X: 0, Y: h.Cursor.Y}, indentstr)
	h.Buf.RelocateCursors()
	h.Relocate()
	return true
}

// OutdentLine moves the current line back one indentation
func (h *BufPane) OutdentLine() bool {
	if h.Cursor.HasSelection() {
		return false
	}

	for x := 0; x < len(h.Buf.IndentString(util.IntOpt(h.Buf.Settings["tabsize"]))); x++ {
		if len(util.GetLeadingWhitespace(h.Buf.LineBytes(h.Cursor.Y))) == 0 {
			break
		}
		h.Buf.Remove(buffer.Loc{X: 0, Y: h.Cursor.Y}, buffer.Loc{X: 1, Y: h.Cursor.Y})
	}
	h.Buf.RelocateCursors()
	h.Relocate()
	return true
}

// OutdentSelection takes the current selection and moves it back one indent level
func (h *BufPane) OutdentSelection() bool {
	if h.Cursor.HasSelection() {
		start := h.Cursor.CurSelection[0]
		end := h.Cursor.CurSelection[1]
		if end.Y < start.Y {
			start, end = end, start
			h.Cursor.SetSelectionStart(start)
			h.Cursor.SetSelectionEnd(end)
		}

		startY := start.Y
		endY := end.Move(-1, h.Buf).Y
		for y := startY; y <= endY; y++ {
			for x := 0; x < len(h.Buf.IndentString(util.IntOpt(h.Buf.Settings["tabsize"]))); x++ {
				if len(util.GetLeadingWhitespace(h.Buf.LineBytes(y))) == 0 {
					break
				}
				h.Buf.Remove(buffer.Loc{X: 0, Y: y}, buffer.Loc{X: 1, Y: y})
			}
		}
		h.Buf.RelocateCursors()

		h.Relocate()
		return true
	}
	return false
}

// Autocomplete cycles the suggestions and performs autocompletion if there are suggestions
func (h *BufPane) Autocomplete() bool {
	b := h.Buf

	if h.Cursor.HasSelection() {
		return false
	}

	if b.HasSuggestions {
		b.CycleAutocomplete(true)
		return true
	}

	if h.Cursor.X == 0 {
		return false
	}
	r := h.Cursor.RuneUnder(h.Cursor.X)
	prev := h.Cursor.RuneUnder(h.Cursor.X - 1)
	if !util.IsAutocomplete(prev) || util.IsWordChar(r) {
		// don't autocomplete if cursor is within a word
		return false
	}

	return b.Autocomplete(buffer.BufferComplete)
}

// CycleAutocompleteBack cycles back in the autocomplete suggestion list
func (h *BufPane) CycleAutocompleteBack() bool {
	if h.Cursor.HasSelection() {
		return false
	}

	if h.Buf.HasSuggestions {
		h.Buf.CycleAutocomplete(false)
		return true
	}
	return false
}

// InsertTab inserts a tab or spaces
func (h *BufPane) InsertTab() bool {
	b := h.Buf
	indent := b.IndentString(util.IntOpt(b.Settings["tabsize"]))
	tabBytes := len(indent)
	bytesUntilIndent := tabBytes - (h.Cursor.GetVisualX(false) % tabBytes)
	b.Insert(h.Cursor.Loc, indent[:bytesUntilIndent])
	h.Relocate()
	return true
}

// SaveAll saves all open buffers
func (h *BufPane) SaveAll() bool {
	for _, b := range buffer.OpenBuffers {
		b.Save()
	}
	return true
}

// SaveCB performs a save and does a callback at the very end (after all prompts have been resolved)
func (h *BufPane) SaveCB(action string, callback func()) bool {
	// If this is an empty buffer, ask for a filename
	if h.Buf.Path == "" {
		h.SaveAsCB(action, callback)
	} else {
		noPrompt := h.saveBufToFile(h.Buf.Path, action, callback)
		if noPrompt {
			return true
		}
	}
	return false
}

// Save the buffer to disk
func (h *BufPane) Save() bool {
	return h.SaveCB("Save", nil)
}

// SaveAsCB performs a save as and does a callback at the very end (after all prompts have been resolved)
// The callback is only called if the save was successful
func (h *BufPane) SaveAsCB(action string, callback func()) bool {
	InfoBar.Prompt("Filename: ", "", "Save", nil, func(resp string, canceled bool) {
		if !canceled {
			// the filename might or might not be quoted, so unquote first then join the strings.
			args, err := shellquote.Split(resp)
			if err != nil {
				InfoBar.Error("Error parsing arguments: ", err)
				return
			}
			if len(args) == 0 {
				InfoBar.Error("No filename given")
				return
			}
			filename := strings.Join(args, " ")
			fileinfo, err := os.Stat(filename)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
					noPrompt := h.saveBufToFile(filename, action, callback)
					if noPrompt {
						h.completeAction(action)
						return
					}
				} else {
					InfoBar.Error(err)
					return
				}
			} else {
				InfoBar.YNPrompt(
					fmt.Sprintf("The file %s already exists in the directory, would you like to overwrite? Y/n", fileinfo.Name()),
					func(yes, canceled bool) {
						if yes && !canceled {
							noPrompt := h.saveBufToFile(filename, action, callback)
							if noPrompt {
								h.completeAction(action)
							}
						}
					},
				)
			}
		}
	})
	return false
}

// SaveAs saves the buffer to disk with the given name
func (h *BufPane) SaveAs() bool {
	return h.SaveAsCB("SaveAs", nil)
}

// This function saves the buffer to `filename` and changes the buffer's path and name
// to `filename` if the save is successful
// The callback is only called if the save was successful
func (h *BufPane) saveBufToFile(filename string, action string, callback func()) bool {
	err := h.Buf.SaveAs(filename)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			if runtime.GOOS == "windows" {
				InfoBar.Error("Permission denied. Save with sudo not supported on Windows")
				return true
			}

			saveWithSudo := func() {
				err = h.Buf.SaveAsWithSudo(filename)
				if err != nil {
					InfoBar.Error(err)
				} else {
					InfoBar.Message("Saved " + filename)
					if callback != nil {
						callback()
					}
				}
			}
			if h.Buf.Settings["autosu"].(bool) {
				saveWithSudo()
			} else {
				InfoBar.YNPrompt(
					fmt.Sprintf("Permission denied. Do you want to save this file using %s? (y,n)", config.GlobalSettings["sucmd"].(string)),
					func(yes, canceled bool) {
						if yes && !canceled {
							saveWithSudo()
							h.completeAction(action)
						}
					},
				)
				return false
			}
		} else {
			InfoBar.Error(err)
		}
	} else {
		InfoBar.Message("Saved " + filename)
		if callback != nil {
			callback()
		}
	}
	return true
}

// Find opens a prompt and searches forward for the input
func (h *BufPane) Find() bool {
	return h.find(true)
}

// FindLiteral is the same as Find() but does not support regular expressions
func (h *BufPane) FindLiteral() bool {
	return h.find(false)
}

// Search searches for a given string/regex in the buffer and selects the next
// match if a match is found
// This function behaves the same way as Find and FindLiteral actions:
// it affects the buffer's LastSearch and LastSearchRegex (saved searches)
// for use with FindNext and FindPrevious, and turns HighlightSearch on or off
// according to hlsearch setting
func (h *BufPane) Search(str string, useRegex bool, searchDown bool) error {
	match, found, err := h.Buf.FindNext(str, h.Buf.Start(), h.Buf.End(), h.Cursor.Loc, searchDown, useRegex)
	if err != nil {
		return err
	}
	if found {
		h.Cursor.SetSelectionStart(match[0])
		h.Cursor.SetSelectionEnd(match[1])
		h.Cursor.OrigSelection[0] = h.Cursor.CurSelection[0]
		h.Cursor.OrigSelection[1] = h.Cursor.CurSelection[1]
		h.GotoLoc(h.Cursor.CurSelection[1])
		h.Buf.LastSearch = str
		h.Buf.LastSearchRegex = useRegex
		h.Buf.HighlightSearch = h.Buf.Settings["hlsearch"].(bool)
	} else {
		h.Cursor.ResetSelection()
	}
	return nil
}

func (h *BufPane) find(useRegex bool) bool {
	h.searchOrig = h.Cursor.Loc
	prompt := "Find: "
	if useRegex {
		prompt = "Find (regex): "
	}
	var eventCallback func(resp string)
	if h.Buf.Settings["incsearch"].(bool) {
		eventCallback = func(resp string) {
			match, found, _ := h.Buf.FindNext(resp, h.Buf.Start(), h.Buf.End(), h.searchOrig, true, useRegex)
			if found {
				h.Cursor.SetSelectionStart(match[0])
				h.Cursor.SetSelectionEnd(match[1])
				h.Cursor.OrigSelection[0] = h.Cursor.CurSelection[0]
				h.Cursor.OrigSelection[1] = h.Cursor.CurSelection[1]
				h.GotoLoc(match[1])
			} else {
				h.GotoLoc(h.searchOrig)
				h.Cursor.ResetSelection()
			}
		}
	}
	findCallback := func(resp string, canceled bool) {
		// Finished callback
		if !canceled {
			match, found, err := h.Buf.FindNext(resp, h.Buf.Start(), h.Buf.End(), h.searchOrig, true, useRegex)
			if err != nil {
				InfoBar.Error(err)
			} else if found {
				h.Cursor.SetSelectionStart(match[0])
				h.Cursor.SetSelectionEnd(match[1])
				h.Cursor.OrigSelection[0] = h.Cursor.CurSelection[0]
				h.Cursor.OrigSelection[1] = h.Cursor.CurSelection[1]
				h.GotoLoc(h.Cursor.CurSelection[1])
				h.Buf.LastSearch = resp
				h.Buf.LastSearchRegex = useRegex
				h.Buf.HighlightSearch = h.Buf.Settings["hlsearch"].(bool)
			} else {
				h.Cursor.ResetSelection()
				InfoBar.Message("No matches found")
			}
		} else {
			h.Cursor.ResetSelection()
		}
	}
	pattern := string(h.Cursor.GetSelection())
	if useRegex && pattern != "" {
		pattern = regexp.QuoteMeta(pattern)
	}
	if eventCallback != nil && pattern != "" {
		eventCallback(pattern)
	}
	InfoBar.Prompt(prompt, pattern, "Find", eventCallback, findCallback)
	if pattern != "" {
		InfoBar.SelectAll()
	}
	return true
}

// ToggleHighlightSearch toggles highlighting all instances of the last used search term
func (h *BufPane) ToggleHighlightSearch() bool {
	h.Buf.HighlightSearch = !h.Buf.HighlightSearch
	return true
}

// UnhighlightSearch unhighlights all instances of the last used search term
func (h *BufPane) UnhighlightSearch() bool {
	if !h.Buf.HighlightSearch {
		return false
	}
	h.Buf.HighlightSearch = false
	return true
}

// ResetSearch resets the last used search term
func (h *BufPane) ResetSearch() bool {
	if h.Buf.LastSearch != "" {
		h.Buf.LastSearch = ""
		return true
	}
	return false
}

// FindNext searches forwards for the last used search term
func (h *BufPane) FindNext() bool {
	if h.Buf.LastSearch == "" {
		return false
	}
	// If the cursor is at the start of a selection and we search we want
	// to search from the end of the selection in the case that
	// the selection is a search result in which case we wouldn't move at
	// at all which would be bad
	searchLoc := h.Cursor.Loc
	if h.Cursor.HasSelection() {
		searchLoc = h.Cursor.CurSelection[1]
	}
	match, found, err := h.Buf.FindNext(h.Buf.LastSearch, h.Buf.Start(), h.Buf.End(), searchLoc, true, h.Buf.LastSearchRegex)
	if err != nil {
		InfoBar.Error(err)
	} else if found && searchLoc == match[0] && match[0] == match[1] {
		// skip empty match at present cursor location
		if searchLoc == h.Buf.End() {
			searchLoc = h.Buf.Start()
		} else {
			searchLoc = searchLoc.Move(1, h.Buf)
		}
		match, found, _ = h.Buf.FindNext(h.Buf.LastSearch, h.Buf.Start(), h.Buf.End(), searchLoc, true, h.Buf.LastSearchRegex)
	}
	if found {
		h.Cursor.SetSelectionStart(match[0])
		h.Cursor.SetSelectionEnd(match[1])
		h.Cursor.OrigSelection[0] = h.Cursor.CurSelection[0]
		h.Cursor.OrigSelection[1] = h.Cursor.CurSelection[1]
		h.GotoLoc(h.Cursor.CurSelection[1])
	} else {
		h.Cursor.ResetSelection()
	}
	return true
}

// FindPrevious searches backwards for the last used search term
func (h *BufPane) FindPrevious() bool {
	if h.Buf.LastSearch == "" {
		return false
	}
	// If the cursor is at the end of a selection and we search we want
	// to search from the beginning of the selection in the case that
	// the selection is a search result in which case we wouldn't move at
	// at all which would be bad
	searchLoc := h.Cursor.Loc
	if h.Cursor.HasSelection() {
		searchLoc = h.Cursor.CurSelection[0]
	}
	match, found, err := h.Buf.FindNext(h.Buf.LastSearch, h.Buf.Start(), h.Buf.End(), searchLoc, false, h.Buf.LastSearchRegex)
	if err != nil {
		InfoBar.Error(err)
	} else if found && searchLoc == match[0] && match[0] == match[1] {
		// skip empty match at present cursor location
		if searchLoc == h.Buf.Start() {
			searchLoc = h.Buf.End()
		} else {
			searchLoc = searchLoc.Move(-1, h.Buf)
		}
		match, found, _ = h.Buf.FindNext(h.Buf.LastSearch, h.Buf.Start(), h.Buf.End(), searchLoc, false, h.Buf.LastSearchRegex)
	}
	if found {
		h.Cursor.SetSelectionStart(match[0])
		h.Cursor.SetSelectionEnd(match[1])
		h.Cursor.OrigSelection[0] = h.Cursor.CurSelection[0]
		h.Cursor.OrigSelection[1] = h.Cursor.CurSelection[1]
		h.GotoLoc(h.Cursor.CurSelection[1])
	} else {
		h.Cursor.ResetSelection()
	}
	return true
}

// DiffNext searches forward until the beginning of the next block of diffs
func (h *BufPane) DiffNext() bool {
	cur := h.Cursor.Loc.Y
	dl, err := h.Buf.FindNextDiffLine(cur, true)
	if err != nil {
		return false
	}
	h.GotoLoc(buffer.Loc{0, dl})
	return true
}

// DiffPrevious searches forward until the end of the previous block of diffs
func (h *BufPane) DiffPrevious() bool {
	cur := h.Cursor.Loc.Y
	dl, err := h.Buf.FindNextDiffLine(cur, false)
	if err != nil {
		return false
	}
	h.GotoLoc(buffer.Loc{0, dl})
	return true
}

// Undo undoes the last action
func (h *BufPane) Undo() bool {
	if !h.Buf.Undo() {
		return false
	}
	InfoBar.Message("Undid action")
	h.Relocate()
	return true
}

// Redo redoes the last action
func (h *BufPane) Redo() bool {
	if !h.Buf.Redo() {
		return false
	}
	InfoBar.Message("Redid action")
	h.Relocate()
	return true
}

func (h *BufPane) selectLines() int {
	if h.Cursor.HasSelection() {
		start := h.Cursor.CurSelection[0]
		end := h.Cursor.CurSelection[1]
		if start.GreaterThan(end) {
			start, end = end, start
		}
		if end.X == 0 {
			end = end.Move(-1, h.Buf)
		}

		h.Cursor.Deselect(true)
		h.Cursor.SetSelectionStart(buffer.Loc{0, start.Y})
		h.Cursor.SetSelectionEnd(buffer.Loc{0, end.Y + 1})
	} else {
		h.Cursor.SelectLine()
	}

	nlines := h.Cursor.CurSelection[1].Y - h.Cursor.CurSelection[0].Y
	if nlines == 0 && h.Cursor.HasSelection() {
		// selected last line and it is not empty
		nlines++
	}
	return nlines
}

// Copy the selection to the system clipboard
func (h *BufPane) Copy() bool {
	if !h.Cursor.HasSelection() {
		return false
	}
	h.Cursor.CopySelection(clipboard.ClipboardReg)
	h.freshClip = false
	InfoBar.Message("Copied selection")
	h.Relocate()
	return true
}

// CopyLine copies the current line to the clipboard. If there is a selection,
// CopyLine copies all the lines that are (fully or partially) in the selection.
func (h *BufPane) CopyLine() bool {
	origLoc := h.Cursor.Loc
	origLastVisualX := h.Cursor.LastVisualX
	origLastWrappedVisualX := h.Cursor.LastWrappedVisualX
	origSelection := h.Cursor.CurSelection

	nlines := h.selectLines()
	if nlines == 0 {
		return false
	}
	h.Cursor.CopySelection(clipboard.ClipboardReg)
	h.freshClip = false
	if nlines > 1 {
		InfoBar.Message(fmt.Sprintf("Copied %d lines", nlines))
	} else {
		InfoBar.Message("Copied line")
	}

	h.Cursor.Loc = origLoc
	h.Cursor.LastVisualX = origLastVisualX
	h.Cursor.LastWrappedVisualX = origLastWrappedVisualX
	h.Cursor.CurSelection = origSelection
	h.Relocate()
	return true
}

// Cut the selection to the system clipboard
func (h *BufPane) Cut() bool {
	if !h.Cursor.HasSelection() {
		return false
	}
	h.Cursor.CopySelection(clipboard.ClipboardReg)
	h.Cursor.DeleteSelection()
	h.Cursor.ResetSelection()
	h.freshClip = false
	InfoBar.Message("Cut selection")

	h.Relocate()
	return true
}

// CutLine cuts the current line to the clipboard. If there is a selection,
// CutLine cuts all the lines that are (fully or partially) in the selection.
func (h *BufPane) CutLine() bool {
	nlines := h.selectLines()
	if nlines == 0 {
		return false
	}
	totalLines := nlines
	if h.freshClip {
		if clip, err := clipboard.Read(clipboard.ClipboardReg); err != nil {
			InfoBar.Error(err)
			return false
		} else {
			clipboard.WriteMulti(clip+string(h.Cursor.GetSelection()), clipboard.ClipboardReg, h.Cursor.Num, h.Buf.NumCursors())
			totalLines = strings.Count(clip, "\n") + nlines
		}
	} else {
		h.Cursor.CopySelection(clipboard.ClipboardReg)
	}
	h.freshClip = true
	h.Cursor.DeleteSelection()
	h.Cursor.ResetSelection()
	h.Cursor.StoreVisualX()
	if totalLines > 1 {
		InfoBar.Message(fmt.Sprintf("Cut %d lines", totalLines))
	} else {
		InfoBar.Message("Cut line")
	}
	h.Relocate()
	return true
}

// Duplicate the selection
func (h *BufPane) Duplicate() bool {
	if !h.Cursor.HasSelection() {
		return false
	}
	h.Buf.Insert(h.Cursor.CurSelection[1], string(h.Cursor.GetSelection()))
	InfoBar.Message("Duplicated selection")
	h.Relocate()
	return true
}

// DuplicateLine duplicates the current line. If there is a selection, DuplicateLine
// duplicates all the lines that are (fully or partially) in the selection.
func (h *BufPane) DuplicateLine() bool {
	if h.Cursor.HasSelection() {
		origLoc := h.Cursor.Loc
		origLastVisualX := h.Cursor.LastVisualX
		origLastWrappedVisualX := h.Cursor.LastWrappedVisualX
		origSelection := h.Cursor.CurSelection

		start := h.Cursor.CurSelection[0]
		end := h.Cursor.CurSelection[1]
		if start.GreaterThan(end) {
			start, end = end, start
		}
		if end.X == 0 {
			end = end.Move(-1, h.Buf)
		}

		h.Cursor.Deselect(true)
		h.Cursor.Loc = end
		h.Cursor.End()
		for y := start.Y; y <= end.Y; y++ {
			h.Buf.Insert(h.Cursor.Loc, "\n"+string(h.Buf.LineBytes(y)))
		}

		h.Cursor.Loc = origLoc
		h.Cursor.LastVisualX = origLastVisualX
		h.Cursor.LastWrappedVisualX = origLastWrappedVisualX
		h.Cursor.CurSelection = origSelection

		if start.Y < end.Y {
			InfoBar.Message(fmt.Sprintf("Duplicated %d lines", end.Y-start.Y+1))
		} else {
			InfoBar.Message("Duplicated line")
		}
	} else {
		h.Cursor.End()
		h.Buf.Insert(h.Cursor.Loc, "\n"+string(h.Buf.LineBytes(h.Cursor.Y)))
		InfoBar.Message("Duplicated line")
	}
	h.Relocate()
	return true
}

// DeleteLine deletes the current line. If there is a selection, DeleteLine
// deletes all the lines that are (fully or partially) in the selection.
func (h *BufPane) DeleteLine() bool {
	nlines := h.selectLines()
	if nlines == 0 {
		return false
	}
	h.Cursor.DeleteSelection()
	h.Cursor.ResetSelection()
	h.Cursor.StoreVisualX()
	if nlines > 1 {
		InfoBar.Message(fmt.Sprintf("Deleted %d lines", nlines))
	} else {
		InfoBar.Message("Deleted line")
	}
	h.Relocate()
	return true
}

// MoveLinesUp moves up the current line or selected lines if any
func (h *BufPane) MoveLinesUp() bool {
	if h.Cursor.HasSelection() {
		if h.Cursor.CurSelection[0].Y == 0 {
			InfoBar.Message("Cannot move further up")
			return false
		}
		start := h.Cursor.CurSelection[0].Y
		end := h.Cursor.CurSelection[1].Y
		sel := 1
		if start > end {
			end, start = start, end
			sel = 0
		}

		compensate := false
		if h.Cursor.CurSelection[sel].X != 0 {
			end++
		} else {
			compensate = true
		}

		h.Buf.MoveLinesUp(
			start,
			end,
		)
		if compensate {
			h.Cursor.CurSelection[sel].Y -= 1
		}
	} else {
		if h.Cursor.Loc.Y == 0 {
			InfoBar.Message("Cannot move further up")
			return false
		}
		h.Buf.MoveLinesUp(
			h.Cursor.Loc.Y,
			h.Cursor.Loc.Y+1,
		)
	}

	h.Relocate()
	return true
}

// MoveLinesDown moves down the current line or selected lines if any
func (h *BufPane) MoveLinesDown() bool {
	if h.Cursor.HasSelection() {
		if h.Cursor.CurSelection[1].Y >= h.Buf.LinesNum() {
			InfoBar.Message("Cannot move further down")
			return false
		}
		start := h.Cursor.CurSelection[0].Y
		end := h.Cursor.CurSelection[1].Y
		sel := 1
		if start > end {
			end, start = start, end
			sel = 0
		}

		if h.Cursor.CurSelection[sel].X != 0 {
			end++
		}

		h.Buf.MoveLinesDown(
			start,
			end,
		)
	} else {
		if h.Cursor.Loc.Y >= h.Buf.LinesNum()-1 {
			InfoBar.Message("Cannot move further down")
			return false
		}
		h.Buf.MoveLinesDown(
			h.Cursor.Loc.Y,
			h.Cursor.Loc.Y+1,
		)
	}

	h.Relocate()
	return true
}

// Paste whatever is in the system clipboard into the buffer
// Delete and paste if the user has a selection
func (h *BufPane) Paste() bool {
	clip, err := clipboard.ReadMulti(clipboard.ClipboardReg, h.Cursor.Num, h.Buf.NumCursors())
	if err != nil {
		InfoBar.Error(err)
	} else {
		h.paste(clip)
	}
	h.Relocate()
	return true
}

// PastePrimary pastes from the primary clipboard (only use on linux)
func (h *BufPane) PastePrimary() bool {
	clip, err := clipboard.ReadMulti(clipboard.PrimaryReg, h.Cursor.Num, h.Buf.NumCursors())
	if err != nil {
		InfoBar.Error(err)
	} else {
		h.paste(clip)
	}
	h.Relocate()
	return true
}

func (h *BufPane) paste(clip string) {
	if h.Buf.Settings["smartpaste"].(bool) {
		if h.Cursor.X > 0 {
			leadingPasteWS := string(util.GetLeadingWhitespace([]byte(clip)))
			if leadingPasteWS != " " && strings.Contains(clip, "\n"+leadingPasteWS) {
				leadingWS := string(util.GetLeadingWhitespace(h.Buf.LineBytes(h.Cursor.Y)))
				clip = strings.TrimPrefix(clip, leadingPasteWS)
				clip = strings.ReplaceAll(clip, "\n"+leadingPasteWS, "\n"+leadingWS)
			}
		}
	}

	if h.Cursor.HasSelection() {
		h.Buf.Replace(h.Cursor.CurSelection[0], h.Cursor.CurSelection[1], clip)
	} else {
		h.Buf.Insert(h.Cursor.Loc, clip)
	}

	// h.Cursor.Loc = h.Cursor.Loc.Move(Count(clip), h.Buf)
	h.freshClip = false
	InfoBar.Message("Pasted clipboard")
}

// JumpToMatchingBrace moves the cursor to the matching brace if it is
// currently on a brace
func (h *BufPane) JumpToMatchingBrace() bool {
	matchingBrace, left, found := h.Buf.FindMatchingBrace(h.Cursor.Loc)
	if found {
		if h.Buf.Settings["matchbraceleft"].(bool) {
			if left {
				h.Cursor.GotoLoc(matchingBrace)
			} else {
				h.Cursor.GotoLoc(matchingBrace.Move(1, h.Buf))
			}
		} else {
			h.Cursor.GotoLoc(matchingBrace)
		}
		h.Relocate()
		return true
	}
	return false
}

// SelectAll selects the entire buffer
func (h *BufPane) SelectAll() bool {
	h.Cursor.SetSelectionStart(h.Buf.Start())
	h.Cursor.SetSelectionEnd(h.Buf.End())
	// Put the cursor at the beginning
	h.Cursor.X = 0
	h.Cursor.Y = 0
	h.Relocate()
	return true
}

// OpenFile opens a new file in the buffer
func (h *BufPane) OpenFile() bool {
	InfoBar.Prompt("> ", "open ", "Open", nil, func(resp string, canceled bool) {
		if !canceled {
			h.HandleCommand(resp)
		}
	})
	return true
}

// JumpLine asks the user to enter a line number to jump to
func (h *BufPane) JumpLine() bool {
	InfoBar.Prompt("> ", "goto ", "Command", nil, func(resp string, canceled bool) {
		if !canceled {
			h.HandleCommand(resp)
		}
	})
	return true
}

// Start moves the viewport to the start of the buffer
func (h *BufPane) Start() bool {
	v := h.GetView()
	v.StartLine = display.SLoc{0, 0}
	h.SetView(v)
	return true
}

// End moves the viewport to the end of the buffer
func (h *BufPane) End() bool {
	v := h.GetView()
	v.StartLine = h.Scroll(h.SLocFromLoc(h.Buf.End()), -h.BufView().Height+1)
	h.SetView(v)
	return true
}

// PageUp scrolls the view up a page
func (h *BufPane) PageUp() bool {
	pageOverlap := int(h.Buf.Settings["pageoverlap"].(float64))
	h.ScrollUp(h.BufView().Height - pageOverlap)
	return true
}

// PageDown scrolls the view down a page
func (h *BufPane) PageDown() bool {
	pageOverlap := int(h.Buf.Settings["pageoverlap"].(float64))
	h.ScrollDown(h.BufView().Height - pageOverlap)
	h.ScrollAdjust()
	return true
}

// SelectPageUp selects up one page
func (h *BufPane) SelectPageUp() bool {
	pageOverlap := int(h.Buf.Settings["pageoverlap"].(float64))
	scrollAmount := h.BufView().Height - pageOverlap
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.MoveCursorUp(scrollAmount)
	h.Cursor.SelectTo(h.Cursor.Loc)
	if h.Cursor.Num == 0 {
		h.ScrollUp(scrollAmount)
	}
	h.Relocate()
	return true
}

// SelectPageDown selects down one page
func (h *BufPane) SelectPageDown() bool {
	pageOverlap := int(h.Buf.Settings["pageoverlap"].(float64))
	scrollAmount := h.BufView().Height - pageOverlap
	if !h.Cursor.HasSelection() {
		h.Cursor.OrigSelection[0] = h.Cursor.Loc
	}
	h.MoveCursorDown(scrollAmount)
	h.Cursor.SelectTo(h.Cursor.Loc)
	if h.Cursor.Num == 0 && !h.ScrollReachedEnd() {
		h.ScrollDown(scrollAmount)
		h.ScrollAdjust()
	}
	h.Relocate()
	return true
}

// CursorPageUp places the cursor a page up,
// moving the view to keep cursor at the same relative position in the view
func (h *BufPane) CursorPageUp() bool {
	h.Cursor.Deselect(true)
	pageOverlap := int(h.Buf.Settings["pageoverlap"].(float64))
	scrollAmount := h.BufView().Height - pageOverlap
	h.MoveCursorUp(scrollAmount)
	if h.Cursor.Num == 0 {
		h.ScrollUp(scrollAmount)
	}
	h.Relocate()
	return true
}

// CursorPageDown places the cursor a page down,
// moving the view to keep cursor at the same relative position in the view
func (h *BufPane) CursorPageDown() bool {
	selectionEndNewline := h.Cursor.HasSelection() && h.Cursor.CurSelection[1].X == 0
	h.Cursor.Deselect(false)
	pageOverlap := int(h.Buf.Settings["pageoverlap"].(float64))
	scrollAmount := h.BufView().Height - pageOverlap
	if selectionEndNewline {
		scrollAmount--
	}
	h.MoveCursorDown(scrollAmount)
	if h.Cursor.Num == 0 && !h.ScrollReachedEnd() {
		h.ScrollDown(scrollAmount)
		h.ScrollAdjust()
	}
	if selectionEndNewline {
		h.Cursor.Start()
	}
	h.Relocate()
	return true
}

// HalfPageUp scrolls the view up half a page
func (h *BufPane) HalfPageUp() bool {
	h.ScrollUp(h.BufView().Height / 2)
	return true
}

// HalfPageDown scrolls the view down half a page
func (h *BufPane) HalfPageDown() bool {
	h.ScrollDown(h.BufView().Height / 2)
	h.ScrollAdjust()
	return true
}

// ToggleDiffGutter turns the diff gutter off and on
func (h *BufPane) ToggleDiffGutter() bool {
	diffgutter := !h.Buf.Settings["diffgutter"].(bool)
	h.Buf.SetOptionNative("diffgutter", diffgutter)
	if diffgutter {
		h.Buf.UpdateDiff()
		InfoBar.Message("Enabled diff gutter")
	} else {
		InfoBar.Message("Disabled diff gutter")
	}
	return true
}

// ToggleRuler turns line numbers off and on
func (h *BufPane) ToggleRuler() bool {
	ruler := !h.Buf.Settings["ruler"].(bool)
	h.Buf.SetOptionNative("ruler", ruler)
	if ruler {
		InfoBar.Message("Enabled ruler")
	} else {
		InfoBar.Message("Disabled ruler")
	}
	return true
}

// ClearStatus clears the infobar. It is an alias for ClearInfo.
func (h *BufPane) ClearStatus() bool {
	return h.ClearInfo()
}

// ToggleHelp toggles the help screen
func (h *BufPane) ToggleHelp() bool {
	if h.Buf.Type == buffer.BTHelp {
		h.Quit()
	} else {
		hsplit := config.GlobalSettings["helpsplit"] == "hsplit"
		h.openHelp("help", hsplit, false)
	}
	return true
}

// ToggleKeyMenu toggles the keymenu option and resizes all tabs
func (h *BufPane) ToggleKeyMenu() bool {
	config.GlobalSettings["keymenu"] = !config.GetGlobalOption("keymenu").(bool)
	Tabs.Resize()
	return true
}

// ShellMode opens a terminal to run a shell command
func (h *BufPane) ShellMode() bool {
	InfoBar.Prompt("$ ", "", "Shell", nil, func(resp string, canceled bool) {
		if !canceled {
			// The true here is for openTerm to make the command interactive
			shell.RunInteractiveShell(resp, true, false)
		}
	})

	return true
}

// CommandMode lets the user enter a command
func (h *BufPane) CommandMode() bool {
	InfoBar.Prompt("> ", "", "Command", nil, func(resp string, canceled bool) {
		if !canceled {
			h.HandleCommand(resp)
		}
	})
	return true
}

// ToggleOverwriteMode lets the user toggle the text overwrite mode
func (h *BufPane) ToggleOverwriteMode() bool {
	h.Buf.OverwriteMode = !h.Buf.OverwriteMode
	return true
}

// Escape leaves current mode
func (h *BufPane) Escape() bool {
	return true
}

// Deselect deselects on the current cursor
func (h *BufPane) Deselect() bool {
	if !h.Cursor.HasSelection() {
		return false
	}
	h.Cursor.Deselect(true)
	return true
}

// ClearInfo clears the infobar
func (h *BufPane) ClearInfo() bool {
	if InfoBar.Msg == "" {
		return false
	}
	InfoBar.Message("")
	return true
}

// ForceQuit closes the tab or view even if there are unsaved changes
// (no prompt)
func (h *BufPane) ForceQuit() bool {
	h.Buf.Close()
	if len(h.tab.Panes) > 1 {
		h.Unsplit()
	} else if len(Tabs.List) > 1 {
		Tabs.RemoveTab(h.splitID)
	} else {
		screen.Screen.Fini()
		InfoBar.Close()
		runtime.Goexit()
	}
	return true
}

// closePrompt displays a prompt to save the buffer before closing it to proceed
// with a different action or command
func (h *BufPane) closePrompt(action string, callback func()) {
	InfoBar.YNPrompt("Save changes to "+h.Buf.GetName()+" before closing? (y,n,esc)", func(yes, canceled bool) {
		if !canceled && !yes {
			callback()
		} else if !canceled && yes {
			h.SaveCB(action, callback)
		}
	})
}

// Quit this will close the current tab or view that is open
func (h *BufPane) Quit() bool {
	if h.Buf.Modified() && !h.Buf.Shared() {
		if config.GlobalSettings["autosave"].(float64) > 0 && h.Buf.Path != "" {
			// autosave on means we automatically save when quitting
			h.SaveCB("Quit", func() {
				h.ForceQuit()
			})
		} else {
			h.closePrompt("Quit", func() {
				h.ForceQuit()
			})
		}
	} else {
		h.ForceQuit()
	}
	return true
}

// QuitAll quits the whole editor; all splits and tabs
func (h *BufPane) QuitAll() bool {
	anyModified := false
	for _, b := range buffer.OpenBuffers {
		if b.Modified() {
			anyModified = true
			break
		}
	}

	quit := func() {
		buffer.CloseOpenBuffers()
		screen.Screen.Fini()
		InfoBar.Close()
		runtime.Goexit()
	}

	if anyModified {
		InfoBar.YNPrompt("Quit micro? (all open buffers will be closed without saving)", func(yes, canceled bool) {
			if !canceled && yes {
				quit()
			}
		})
	} else {
		quit()
	}

	return true
}

// AddTab adds a new tab with an empty buffer
func (h *BufPane) AddTab() bool {
	width, height := screen.Screen.Size()
	iOffset := config.GetInfoBarOffset()
	b := buffer.NewBufferFromString("", "", buffer.BTDefault)
	tp := NewTabFromBuffer(0, 0, width, height-iOffset, b)
	Tabs.AddTab(tp)
	Tabs.SetActive(len(Tabs.List) - 1)

	return true
}

// PreviousTab switches to the previous tab in the tab list
func (h *BufPane) PreviousTab() bool {
	if Tabs.Active() == 0 {
		return false
	}
	Tabs.SetActive(Tabs.Active() - 1)
	return true
}

// NextTab switches to the next tab in the tab list
func (h *BufPane) NextTab() bool {
	if Tabs.Active() == len(Tabs.List)-1 {
		return false
	}
	Tabs.SetActive(Tabs.Active() + 1)
	return true
}

// FirstTab switches to the first tab in the tab list
func (h *BufPane) FirstTab() bool {
	if Tabs.Active() == 0 {
		return false
	}
	Tabs.SetActive(0)
	return true
}

// LastTab switches to the last tab in the tab list
func (h *BufPane) LastTab() bool {
	lastTabIndex := len(Tabs.List) - 1
	if Tabs.Active() == lastTabIndex {
		return false
	}
	Tabs.SetActive(lastTabIndex)
	return true
}

// VSplitAction opens an empty vertical split
func (h *BufPane) VSplitAction() bool {
	h.VSplitBuf(buffer.NewBufferFromString("", "", buffer.BTDefault))

	return true
}

// HSplitAction opens an empty horizontal split
func (h *BufPane) HSplitAction() bool {
	h.HSplitBuf(buffer.NewBufferFromString("", "", buffer.BTDefault))

	return true
}

// Unsplit closes all splits in the current tab except the active one
func (h *BufPane) Unsplit() bool {
	tab := h.tab
	n := tab.GetNode(h.splitID)
	ok := n.Unsplit()
	if ok {
		tab.RemovePane(tab.GetPane(h.splitID))
		tab.Resize()
		tab.SetActive(len(tab.Panes) - 1)

		return true
	}
	return false
}

// NextSplit changes the view to the next split
func (h *BufPane) NextSplit() bool {
	if h.tab.active == len(h.tab.Panes)-1 {
		return false
	}
	h.tab.SetActive(h.tab.active + 1)
	return true
}

// PreviousSplit changes the view to the previous split
func (h *BufPane) PreviousSplit() bool {
	if h.tab.active == 0 {
		return false
	}
	h.tab.SetActive(h.tab.active - 1)
	return true
}

// FirstSplit changes the view to the first split
func (h *BufPane) FirstSplit() bool {
	if h.tab.active == 0 {
		return false
	}
	h.tab.SetActive(0)
	return true
}

// LastSplit changes the view to the last split
func (h *BufPane) LastSplit() bool {
	lastPaneIdx := len(h.tab.Panes) - 1
	if h.tab.active == lastPaneIdx {
		return false
	}
	h.tab.SetActive(lastPaneIdx)
	return true
}

var curmacro []any
var recordingMacro bool

// ToggleMacro toggles recording of a macro
func (h *BufPane) ToggleMacro() bool {
	recordingMacro = !recordingMacro
	if recordingMacro {
		curmacro = []any{}
		InfoBar.Message("Recording")
	} else {
		InfoBar.Message("Stopped recording")
	}
	h.Relocate()
	return true
}

// PlayMacro plays back the most recently recorded macro
func (h *BufPane) PlayMacro() bool {
	if recordingMacro {
		return false
	}
	for _, action := range curmacro {
		switch t := action.(type) {
		case rune:
			h.DoRuneInsert(t)
		case BufKeyAction:
			t(h)
		}
	}
	h.Relocate()
	return true
}

// SpawnMultiCursor creates a new multiple cursor at the next occurrence of the current selection or current word
func (h *BufPane) SpawnMultiCursor() bool {
	spawner := h.Buf.GetCursor(h.Buf.NumCursors() - 1)
	if !spawner.HasSelection() {
		spawner.SelectWord()
		h.multiWord = true
		h.Relocate()
		return true
	}

	sel := spawner.GetSelection()
	searchStart := spawner.CurSelection[1]

	search := string(sel)
	search = regexp.QuoteMeta(search)
	if h.multiWord {
		search = "\\b" + search + "\\b"
	}
	match, found, err := h.Buf.FindNext(search, h.Buf.Start(), h.Buf.End(), searchStart, true, true)
	if err != nil {
		InfoBar.Error(err)
	}
	if found {
		c := buffer.NewCursor(h.Buf, buffer.Loc{})
		c.SetSelectionStart(match[0])
		c.SetSelectionEnd(match[1])
		c.OrigSelection[0] = c.CurSelection[0]
		c.OrigSelection[1] = c.CurSelection[1]
		c.Loc = c.CurSelection[1]

		h.Buf.AddCursor(c)
		h.Buf.SetCurCursor(h.Buf.NumCursors() - 1)
		h.Buf.MergeCursors()
	} else {
		InfoBar.Message("No matches found")
	}

	h.Relocate()
	return true
}

// SpawnCursorAtLoc spawns a new cursor at a location and merges the cursors
func (h *BufPane) SpawnCursorAtLoc(loc buffer.Loc) *buffer.Cursor {
	c := buffer.NewCursor(h.Buf, loc)
	h.Buf.AddCursor(c)
	h.Buf.MergeCursors()
	return c
}

// SpawnMultiCursorUpN is not an action
func (h *BufPane) SpawnMultiCursorUpN(n int) bool {
	lastC := h.Buf.GetCursor(h.Buf.NumCursors() - 1)
	if n > 0 && lastC.Y == 0 {
		return false
	}
	if n < 0 && lastC.Y+1 == h.Buf.LinesNum() {
		return false
	}

	h.Buf.DeselectCursors()

	c := buffer.NewCursor(h.Buf, buffer.Loc{lastC.X, lastC.Y - n})
	c.LastVisualX = lastC.LastVisualX
	c.LastWrappedVisualX = lastC.LastWrappedVisualX
	c.X = c.GetCharPosInLine(h.Buf.LineBytes(c.Y), c.LastVisualX)
	c.Relocate()

	h.Buf.AddCursor(c)
	h.Buf.SetCurCursor(h.Buf.NumCursors() - 1)
	h.Buf.MergeCursors()

	h.Relocate()
	return true
}

// SpawnMultiCursorUp creates additional cursor, at the same X (if possible), one Y less.
func (h *BufPane) SpawnMultiCursorUp() bool {
	return h.SpawnMultiCursorUpN(1)
}

// SpawnMultiCursorDown creates additional cursor, at the same X (if possible), one Y more.
func (h *BufPane) SpawnMultiCursorDown() bool {
	return h.SpawnMultiCursorUpN(-1)
}

// SpawnMultiCursorSelect adds a cursor at the beginning of each line of a selection
func (h *BufPane) SpawnMultiCursorSelect() bool {
	// Avoid cases where multiple cursors already exist, that would create problems
	if h.Buf.NumCursors() > 1 {
		return false
	}

	var startLine int
	var endLine int

	a, b := h.Cursor.CurSelection[0].Y, h.Cursor.CurSelection[1].Y
	if a > b {
		startLine, endLine = b, a
	} else {
		startLine, endLine = a, b
	}

	if h.Cursor.HasSelection() {
		h.Cursor.ResetSelection()
		h.Cursor.GotoLoc(buffer.Loc{0, startLine})

		for i := startLine; i <= endLine; i++ {
			c := buffer.NewCursor(h.Buf, buffer.Loc{0, i})
			c.StoreVisualX()
			h.Buf.AddCursor(c)
		}
		h.Buf.MergeCursors()
	} else {
		return false
	}
	InfoBar.Message("Added cursors from selection")
	return true
}

// MouseMultiCursor is a mouse action which puts a new cursor at the mouse position,
// or removes a cursor if it is already there
func (h *BufPane) MouseMultiCursor(e *tcell.EventMouse) bool {
	b := h.Buf
	mx, my := e.Position()
	// ignore click on the status line
	if my >= h.BufView().Y+h.BufView().Height {
		return false
	}
	mouseLoc := h.LocFromVisual(buffer.Loc{X: mx, Y: my})

	if h.Buf.NumCursors() > 1 {
		cursors := h.Buf.GetCursors()
		for _, c := range cursors {
			if c.Loc == mouseLoc {
				h.Buf.RemoveCursor(c.Num)
				return true
			}
		}
	}

	c := buffer.NewCursor(b, mouseLoc)
	b.AddCursor(c)
	b.MergeCursors()

	return true
}

func (h *BufPane) skipMultiCursor(forward bool) bool {
	lastC := h.Buf.GetCursor(h.Buf.NumCursors() - 1)
	if !lastC.HasSelection() {
		return false
	}
	sel := lastC.GetSelection()
	searchStart := lastC.CurSelection[1]
	if !forward {
		searchStart = lastC.CurSelection[0]
	}

	search := string(sel)
	search = regexp.QuoteMeta(search)
	if h.multiWord {
		search = "\\b" + search + "\\b"
	}

	match, found, err := h.Buf.FindNext(search, h.Buf.Start(), h.Buf.End(), searchStart, forward, true)
	if err != nil {
		InfoBar.Error(err)
	}
	if found {
		lastC.SetSelectionStart(match[0])
		lastC.SetSelectionEnd(match[1])
		lastC.OrigSelection[0] = lastC.CurSelection[0]
		lastC.OrigSelection[1] = lastC.CurSelection[1]
		lastC.Loc = lastC.CurSelection[1]

		h.Buf.MergeCursors()
		h.Buf.SetCurCursor(h.Buf.NumCursors() - 1)
	} else {
		InfoBar.Message("No matches found")
	}
	h.Relocate()
	return true
}

// SkipMultiCursor moves the current multiple cursor to the next available position
func (h *BufPane) SkipMultiCursor() bool {
	return h.skipMultiCursor(true)
}

// SkipMultiCursorBack moves the current multiple cursor to the previous available position
func (h *BufPane) SkipMultiCursorBack() bool {
	return h.skipMultiCursor(false)
}

// RemoveMultiCursor removes the latest multiple cursor
func (h *BufPane) RemoveMultiCursor() bool {
	if h.Buf.NumCursors() > 1 {
		h.Buf.RemoveCursor(h.Buf.NumCursors() - 1)
		h.Buf.SetCurCursor(h.Buf.NumCursors() - 1)
		h.Buf.UpdateCursors()
	} else if h.multiWord {
		h.multiWord = false
		h.Cursor.Deselect(true)
	} else {
		return false
	}
	h.Relocate()
	return true
}

// RemoveAllMultiCursors removes all cursors except the base cursor
func (h *BufPane) RemoveAllMultiCursors() bool {
	if h.Buf.NumCursors() > 1 || h.multiWord {
		h.Buf.ClearCursors()
		h.multiWord = false
	} else {
		return false
	}
	h.Relocate()
	return true
}

// None is an action that does nothing
func (h *BufPane) None() bool {
	return true
}
