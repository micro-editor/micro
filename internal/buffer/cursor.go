package buffer

import (
	"github.com/micro-editor/micro/v2/internal/clipboard"
	"github.com/micro-editor/micro/v2/internal/util"
)

// InBounds returns whether the given location is a valid character position in the given buffer
func InBounds(pos Loc, buf *Buffer) bool {
	if pos.Y < 0 || pos.Y >= len(buf.lines) || pos.X < 0 || pos.X > util.CharacterCount(buf.LineBytes(pos.Y)) {
		return false
	}

	return true
}

// The Cursor struct stores the location of the cursor in the buffer
// as well as the selection
type Cursor struct {
	buf *Buffer
	Loc

	// Last visual x position of the cursor. Used in cursor up/down movements
	// for remembering the original x position when moving to a line that is
	// shorter than current x position.
	LastVisualX int
	// Similar to LastVisualX but takes softwrapping into account, i.e. last
	// visual x position in a visual (wrapped) line on the screen, which may be
	// different from the line in the buffer.
	LastWrappedVisualX int

	// The current selection as a range of character numbers (inclusive)
	CurSelection [2]Loc
	// The original selection as a range of character numbers
	// This is used for line and word selection where it is necessary
	// to know what the original selection was
	OrigSelection [2]Loc

	// The line number where a new trailing whitespace has been added
	// or -1 if there is no new trailing whitespace at this cursor.
	// This is used for checking if a trailing whitespace should be highlighted
	NewTrailingWsY int

	// Which cursor index is this (for multiple cursors)
	Num int
}

func NewCursor(b *Buffer, l Loc) *Cursor {
	c := &Cursor{
		buf: b,
		Loc: l,

		NewTrailingWsY: -1,
	}
	c.StoreVisualX()
	return c
}

func (c *Cursor) SetBuf(b *Buffer) {
	c.buf = b
}

func (c *Cursor) Buf() *Buffer {
	return c.buf
}

// Goto puts the cursor at the given cursor's location and gives
// the current cursor its selection too
func (c *Cursor) Goto(b Cursor) {
	c.X, c.Y = b.X, b.Y
	c.OrigSelection, c.CurSelection = b.OrigSelection, b.CurSelection
	c.StoreVisualX()
}

// GotoLoc puts the cursor at the given cursor's location and gives
// the current cursor its selection too
func (c *Cursor) GotoLoc(l Loc) {
	c.X, c.Y = l.X, l.Y
	c.StoreVisualX()
}

// GetVisualX returns the x value of the cursor in visual spaces
func (c *Cursor) GetVisualX(wrap bool) int {
	if wrap && c.buf.GetVisualX != nil {
		return c.buf.GetVisualX(c.Loc)
	}

	if c.X <= 0 {
		c.X = 0
		return 0
	}

	bytes := c.buf.LineBytes(c.Y)
	tabsize := int(c.buf.Settings["tabsize"].(float64))

	return util.StringWidth(bytes, c.X, tabsize)
}

// GetCharPosInLine gets the char position of a visual x y
// coordinate (this is necessary because tabs are 1 char but
// 4 visual spaces)
func (c *Cursor) GetCharPosInLine(b []byte, visualPos int) int {
	tabsize := int(c.buf.Settings["tabsize"].(float64))
	return util.GetCharPosInLine(b, visualPos, tabsize)
}

// Start moves the cursor to the start of the line it is on
func (c *Cursor) Start() {
	c.X = 0
	c.StoreVisualX()
}

// StartOfText moves the cursor to the first non-whitespace rune of
// the line it is on
func (c *Cursor) StartOfText() {
	c.Start()
	for util.IsWhitespace(c.RuneUnder(c.X)) {
		if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
			break
		}
		c.Right()
	}
}

// IsStartOfText returns whether the cursor is at the first
// non-whitespace rune of the line it is on
func (c *Cursor) IsStartOfText() bool {
	x := 0
	for util.IsWhitespace(c.RuneUnder(x)) {
		if x == util.CharacterCount(c.buf.LineBytes(c.Y)) {
			break
		}
		x++
	}
	return c.X == x
}

// End moves the cursor to the end of the line it is on
func (c *Cursor) End() {
	c.X = util.CharacterCount(c.buf.LineBytes(c.Y))
	c.StoreVisualX()
}

// CopySelection copies the user's selection to either "primary"
// or "clipboard"
func (c *Cursor) CopySelection(target clipboard.Register) {
	if c.HasSelection() {
		if target != clipboard.PrimaryReg || c.buf.Settings["useprimary"].(bool) {
			clipboard.WriteMulti(string(c.GetSelection()), target, c.Num, c.buf.NumCursors())
		}
	}
}

// ResetSelection resets the user's selection
func (c *Cursor) ResetSelection() {
	c.CurSelection[0] = c.buf.Start()
	c.CurSelection[1] = c.buf.Start()
}

// SetSelectionStart sets the start of the selection
func (c *Cursor) SetSelectionStart(pos Loc) {
	c.CurSelection[0] = pos
}

// SetSelectionEnd sets the end of the selection
func (c *Cursor) SetSelectionEnd(pos Loc) {
	c.CurSelection[1] = pos
}

// HasSelection returns whether or not the user has selected anything
func (c *Cursor) HasSelection() bool {
	return c.CurSelection[0] != c.CurSelection[1]
}

// DeleteSelection deletes the currently selected text
func (c *Cursor) DeleteSelection() {
	if c.CurSelection[0].GreaterThan(c.CurSelection[1]) {
		c.buf.Remove(c.CurSelection[1], c.CurSelection[0])
		c.Loc = c.CurSelection[1]
	} else if !c.HasSelection() {
		return
	} else {
		c.buf.Remove(c.CurSelection[0], c.CurSelection[1])
		c.Loc = c.CurSelection[0]
	}
}

// Deselect closes the cursor's current selection
// Start indicates whether the cursor should be placed
// at the start or end of the selection
func (c *Cursor) Deselect(start bool) {
	if c.HasSelection() {
		if start {
			c.Loc = c.CurSelection[0]
		} else {
			c.Loc = c.CurSelection[1]
		}
		c.ResetSelection()
		c.StoreVisualX()
	}
}

// GetSelection returns the cursor's selection
func (c *Cursor) GetSelection() []byte {
	if InBounds(c.CurSelection[0], c.buf) && InBounds(c.CurSelection[1], c.buf) {
		if c.CurSelection[0].GreaterThan(c.CurSelection[1]) {
			return c.buf.Substr(c.CurSelection[1], c.CurSelection[0])
		}
		return c.buf.Substr(c.CurSelection[0], c.CurSelection[1])
	}
	return []byte{}
}

// SelectLine selects the current line
func (c *Cursor) SelectLine() {
	c.Start()
	c.SetSelectionStart(c.Loc)
	c.End()
	if len(c.buf.lines)-1 > c.Y {
		c.SetSelectionEnd(c.Loc.Move(1, c.buf))
	} else {
		c.SetSelectionEnd(c.Loc)
	}

	c.OrigSelection = c.CurSelection
}

// AddLineToSelection adds the current line to the selection
func (c *Cursor) AddLineToSelection() {
	if c.Loc.LessThan(c.OrigSelection[0]) {
		c.Start()
		c.SetSelectionStart(c.Loc)
		c.SetSelectionEnd(c.OrigSelection[1])
	}
	if c.Loc.GreaterThan(c.OrigSelection[1]) {
		c.End()
		c.SetSelectionEnd(c.Loc.Move(1, c.buf))
		c.SetSelectionStart(c.OrigSelection[0])
	}

	if c.Loc.LessThan(c.OrigSelection[1]) && c.Loc.GreaterThan(c.OrigSelection[0]) {
		c.CurSelection = c.OrigSelection
	}
}

// UpN moves the cursor up N lines (if possible)
func (c *Cursor) UpN(amount int) {
	proposedY := c.Y - amount
	if proposedY < 0 {
		proposedY = 0
	} else if proposedY >= len(c.buf.lines) {
		proposedY = len(c.buf.lines) - 1
	}

	bytes := c.buf.LineBytes(proposedY)
	c.X = c.GetCharPosInLine(bytes, c.LastVisualX)

	if c.X > util.CharacterCount(bytes) || (amount < 0 && proposedY == c.Y) {
		c.X = util.CharacterCount(bytes)
		c.StoreVisualX()
	}

	if c.X < 0 || (amount > 0 && proposedY == c.Y) {
		c.X = 0
		c.StoreVisualX()
	}

	c.Y = proposedY
}

// DownN moves the cursor down N lines (if possible)
func (c *Cursor) DownN(amount int) {
	c.UpN(-amount)
}

// Up moves the cursor up one line (if possible)
func (c *Cursor) Up() {
	c.UpN(1)
}

// Down moves the cursor down one line (if possible)
func (c *Cursor) Down() {
	c.DownN(1)
}

// Left moves the cursor left one cell (if possible) or to
// the previous line if it is at the beginning
func (c *Cursor) Left() {
	if c.Loc == c.buf.Start() {
		return
	}
	if c.X > 0 {
		c.X--
	} else {
		c.Up()
		c.End()
	}
	c.StoreVisualX()
}

// Right moves the cursor right one cell (if possible) or
// to the next line if it is at the end
func (c *Cursor) Right() {
	if c.Loc == c.buf.End() {
		return
	}
	if c.X < util.CharacterCount(c.buf.LineBytes(c.Y)) {
		c.X++
	} else {
		c.Down()
		c.Start()
	}
	c.StoreVisualX()
}

// Relocate makes sure that the cursor is inside the bounds
// of the buffer If it isn't, it moves it to be within the
// buffer's lines
func (c *Cursor) Relocate() {
	if c.Y < 0 {
		c.Y = 0
	} else if c.Y >= len(c.buf.lines) {
		c.Y = len(c.buf.lines) - 1
	}

	if c.X < 0 {
		c.X = 0
	} else if c.X > util.CharacterCount(c.buf.LineBytes(c.Y)) {
		c.X = util.CharacterCount(c.buf.LineBytes(c.Y))
	}
}

// SelectWord selects the word the cursor is currently on
func (c *Cursor) SelectWord() {
	if len(c.buf.LineBytes(c.Y)) == 0 {
		return
	}

	if !util.IsWordChar(c.RuneUnder(c.X)) {
		c.SetSelectionStart(c.Loc)
		c.SetSelectionEnd(c.Loc.Move(1, c.buf))
		c.OrigSelection = c.CurSelection
		return
	}

	forward, backward := c.X, c.X

	for backward > 0 && util.IsWordChar(c.RuneUnder(backward-1)) {
		backward--
	}

	c.SetSelectionStart(Loc{backward, c.Y})
	c.OrigSelection[0] = c.CurSelection[0]

	lineLen := util.CharacterCount(c.buf.LineBytes(c.Y)) - 1
	for forward < lineLen && util.IsWordChar(c.RuneUnder(forward+1)) {
		forward++
	}

	c.SetSelectionEnd(Loc{forward, c.Y}.Move(1, c.buf))
	c.OrigSelection[1] = c.CurSelection[1]
	c.Loc = c.CurSelection[1]
}

// AddWordToSelection adds the word the cursor is currently on
// to the selection
func (c *Cursor) AddWordToSelection() {
	if c.Loc.GreaterThan(c.OrigSelection[0]) && c.Loc.LessThan(c.OrigSelection[1]) {
		c.CurSelection = c.OrigSelection
		return
	}

	if c.Loc.LessThan(c.OrigSelection[0]) {
		backward := c.X

		for backward > 0 && util.IsWordChar(c.RuneUnder(backward-1)) {
			backward--
		}

		c.SetSelectionStart(Loc{backward, c.Y})
		c.SetSelectionEnd(c.OrigSelection[1])
	}

	if c.Loc.GreaterThan(c.OrigSelection[1]) {
		forward := c.X

		lineLen := util.CharacterCount(c.buf.LineBytes(c.Y)) - 1
		for forward < lineLen && util.IsWordChar(c.RuneUnder(forward+1)) {
			forward++
		}

		c.SetSelectionEnd(Loc{forward, c.Y}.Move(1, c.buf))
		c.SetSelectionStart(c.OrigSelection[0])
	}

	c.Loc = c.CurSelection[1]
}

// SelectTo selects from the current cursor location to the given
// location
func (c *Cursor) SelectTo(loc Loc) {
	if loc.GreaterThan(c.OrigSelection[0]) {
		c.SetSelectionStart(c.OrigSelection[0])
		c.SetSelectionEnd(loc)
	} else {
		c.SetSelectionStart(loc)
		c.SetSelectionEnd(c.OrigSelection[0])
	}
}

// WordRight moves the cursor one word to the right
func (c *Cursor) WordRight() {
	if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
		c.Right()
		return
	}
	for util.IsWhitespace(c.RuneUnder(c.X)) {
		if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
			return
		}
		c.Right()
	}
	if util.IsNonWordChar(c.RuneUnder(c.X)) && !util.IsWhitespace(c.RuneUnder(c.X)) &&
		util.IsNonWordChar(c.RuneUnder(c.X+1)) {
		for util.IsNonWordChar(c.RuneUnder(c.X)) && !util.IsWhitespace(c.RuneUnder(c.X)) {
			if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
				return
			}
			c.Right()
		}
		return
	}
	c.Right()
	for util.IsWordChar(c.RuneUnder(c.X)) {
		if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
			return
		}
		c.Right()
	}
}

// WordLeft moves the cursor one word to the left
func (c *Cursor) WordLeft() {
	if c.X == 0 {
		c.Left()
		return
	}
	c.Left()
	for util.IsWhitespace(c.RuneUnder(c.X)) {
		if c.X == 0 {
			return
		}
		c.Left()
	}
	if util.IsNonWordChar(c.RuneUnder(c.X)) && !util.IsWhitespace(c.RuneUnder(c.X)) &&
		util.IsNonWordChar(c.RuneUnder(c.X-1)) {
		for util.IsNonWordChar(c.RuneUnder(c.X)) && !util.IsWhitespace(c.RuneUnder(c.X)) {
			if c.X == 0 {
				return
			}
			c.Left()
		}
		c.Right()
		return
	}
	c.Left()
	for util.IsWordChar(c.RuneUnder(c.X)) {
		if c.X == 0 {
			return
		}
		c.Left()
	}
	c.Right()
}

// SubWordRight moves the cursor one sub-word to the right
func (c *Cursor) SubWordRight() {
	if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
		c.Right()
		return
	}
	if util.IsWhitespace(c.RuneUnder(c.X)) {
		for util.IsWhitespace(c.RuneUnder(c.X)) {
			if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
				return
			}
			c.Right()
		}
		return
	}
	if util.IsNonWordChar(c.RuneUnder(c.X)) && !util.IsWhitespace(c.RuneUnder(c.X)) {
		for util.IsNonWordChar(c.RuneUnder(c.X)) && !util.IsWhitespace(c.RuneUnder(c.X)) {
			if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
				return
			}
			c.Right()
		}
		return
	}
	if util.IsSubwordDelimiter(c.RuneUnder(c.X)) {
		for util.IsSubwordDelimiter(c.RuneUnder(c.X)) {
			if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
				return
			}
			c.Right()
		}
		if util.IsWhitespace(c.RuneUnder(c.X)) {
			return
		}
	}
	if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
		return
	}
	if util.IsUpperLetter(c.RuneUnder(c.X)) &&
		util.IsUpperLetter(c.RuneUnder(c.X+1)) {
		for util.IsUpperAlphanumeric(c.RuneUnder(c.X)) {
			if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
				return
			}
			c.Right()
		}
		if util.IsLowerAlphanumeric(c.RuneUnder(c.X)) {
			c.Left()
		}
	} else {
		c.Right()
		for util.IsLowerAlphanumeric(c.RuneUnder(c.X)) {
			if c.X == util.CharacterCount(c.buf.LineBytes(c.Y)) {
				return
			}
			c.Right()
		}
	}
}

// SubWordLeft moves the cursor one sub-word to the left
func (c *Cursor) SubWordLeft() {
	if c.X == 0 {
		c.Left()
		return
	}
	c.Left()
	if util.IsWhitespace(c.RuneUnder(c.X)) {
		for util.IsWhitespace(c.RuneUnder(c.X)) {
			if c.X == 0 {
				return
			}
			c.Left()
		}
		c.Right()
		return
	}
	if util.IsNonWordChar(c.RuneUnder(c.X)) && !util.IsWhitespace(c.RuneUnder(c.X)) {
		for util.IsNonWordChar(c.RuneUnder(c.X)) && !util.IsWhitespace(c.RuneUnder(c.X)) {
			if c.X == 0 {
				return
			}
			c.Left()
		}
		c.Right()
		return
	}
	if util.IsSubwordDelimiter(c.RuneUnder(c.X)) {
		for util.IsSubwordDelimiter(c.RuneUnder(c.X)) {
			if c.X == 0 {
				return
			}
			c.Left()
		}
		if util.IsWhitespace(c.RuneUnder(c.X)) {
			c.Right()
			return
		}
	}
	if c.X == 0 {
		return
	}
	if util.IsUpperLetter(c.RuneUnder(c.X)) &&
		util.IsUpperLetter(c.RuneUnder(c.X-1)) {
		for util.IsUpperAlphanumeric(c.RuneUnder(c.X)) {
			if c.X == 0 {
				return
			}
			c.Left()
		}
		if !util.IsUpperAlphanumeric(c.RuneUnder(c.X)) {
			c.Right()
		}
	} else {
		for util.IsLowerAlphanumeric(c.RuneUnder(c.X)) {
			if c.X == 0 {
				return
			}
			c.Left()
		}
		if !util.IsAlphanumeric(c.RuneUnder(c.X)) {
			c.Right()
		}
	}
}

// RuneUnder returns the rune under the given x position
func (c *Cursor) RuneUnder(x int) rune {
	line := c.buf.LineCharacters(c.Y)
	if len(line) == 0 || x >= len(line) {
		return '\n'
	} else if x < 0 {
		x = 0
	}
	return line[x].combc[0]
}

func (c *Cursor) StoreVisualX() {
	c.LastVisualX = c.GetVisualX(false)
	c.LastWrappedVisualX = c.GetVisualX(true)
}
