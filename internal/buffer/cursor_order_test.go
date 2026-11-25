package buffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdjacentCursorReplacementOrder(t *testing.T) {
	b := NewBufferFromString("cc", "", BTDefault)
	defer b.Close()

	b.SetCursors(nil)
	c1 := NewCursor(b, Loc{0, 0})
	c1.SetSelectionStart(Loc{0, 0})
	c1.SetSelectionEnd(Loc{1, 0})
	b.AddCursor(c1)
	c2 := NewCursor(b, Loc{1, 0})
	c2.SetSelectionStart(Loc{1, 0})
	c2.SetSelectionEnd(Loc{2, 0})
	b.AddCursor(c2)

	for _, c := range []*Cursor{c1, c2} {
		b.SetCurCursor(c.Num)
		c.DeleteSelection()
		c.ResetSelection()
		b.Insert(c.Loc, "x")
	}

	assert.Equal(t, "xx", b.Line(0))
	assert.Equal(t, Loc{1, 0}, c1.Loc)
	assert.Equal(t, Loc{2, 0}, c2.Loc)
}
