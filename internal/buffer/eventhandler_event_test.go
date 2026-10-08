package buffer

import (
	"testing"
)

func makeCursorWithSelection(b *Buffer, locX, locY, selStartX, selStartY, selEndX, selEndY int) *Cursor {
	c := NewCursor(b, Loc{locX, locY})
	c.CurSelection = [2]Loc{{selStartX, selStartY}, {selEndX, selEndY}}
	c.OrigSelection = [2]Loc{{selStartX, selStartY}, {selEndX, selEndY}}
	return c
}

func TestInsertTextMoveCursorAfter(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{0, 0}, "Hi ")

	expected := Loc{8, 0}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "Hi hello world"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestInsertTextMoveCursorBefore(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{6, 0}, "Hi ")

	expected := Loc{5, 0}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}
}

func TestInsertMultiLineText(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{6, 0})
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{0, 0}, "line1\nline2\n")

	expected := Loc{6, 2}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "line1\nline2\nhello world"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestRemoveTextMoveCursorAfter(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{11, 0})
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{0, 0}, Loc{6, 0})

	expected := Loc{5, 0}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "world"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestRemoveTextMoveCursorBefore(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{6, 0}, Loc{11, 0})

	expected := Loc{5, 0}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "hello "
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestRemoveMultiLineText(t *testing.T) {
	buf := NewBufferFromString("line1\nline2\nline3", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 2})
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{0, 0}, Loc{0, 2})

	expected := Loc{5, 0}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "line3"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceSingleLineText(t *testing.T) {
	buf := NewBufferFromString("hello world test", "", BTDefault)
	cursor := NewCursor(buf, Loc{16, 0})
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{6, 0}, Loc{11, 0}, "everyone")

	expected := Loc{19, 0}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "hello everyone test"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceSingleLineWithMultiLineEH(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{11, 0})
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{6, 0}, Loc{11, 0}, "beautiful\nnew world")

	expected := Loc{9, 1}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "hello beautiful\nnew world"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceMultiLineWithSingleLineEH(t *testing.T) {
	buf := NewBufferFromString("hello\nbeautiful\nworld", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 2})
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{5, 0}, Loc{0, 2}, " ")

	expected := Loc{11, 0}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "hello world"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceMultiLineWithMultiLineEH(t *testing.T) {
	buf := NewBufferFromString("line1\nline2\nline3\nline4", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 3})
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{0, 1}, Loc{0, 3}, "new\nmiddle")

	expected := Loc{11, 2}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "line1\nnew\nmiddleline4"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceCursorBefore(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{11, 0})
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{0, 0}, Loc{5, 0}, "Hi")

	expected := Loc{8, 0}
	if cursor.Loc != expected {
		t.Errorf("Expected cursor at %v, got %v", expected, cursor.Loc)
	}

	expectedText := "Hi world"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestInsertWithSelection(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := makeCursorWithSelection(buf, 11, 0, 6, 0, 11, 0)
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{0, 0}, "Hi ")

	expectedLoc := Loc{14, 0}
	if cursor.Loc != expectedLoc {
		t.Errorf("Expected cursor.Loc at %v, got %v", expectedLoc, cursor.Loc)
	}

	expectedSelStart := Loc{9, 0}
	expectedSelEnd := Loc{14, 0}
	if cursor.CurSelection[0] != expectedSelStart || cursor.CurSelection[1] != expectedSelEnd {
		t.Errorf("Expected selection [%v,%v], got [%v,%v]",
			expectedSelStart, expectedSelEnd, cursor.CurSelection[0], cursor.CurSelection[1])
	}
}

func TestRemoveWithSelection(t *testing.T) {
	buf := NewBufferFromString("hello world test", "", BTDefault)
	cursor := makeCursorWithSelection(buf, 16, 0, 12, 0, 16, 0)
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{0, 0}, Loc{6, 0})

	expectedLoc := Loc{10, 0}
	if cursor.Loc != expectedLoc {
		t.Errorf("Expected cursor.Loc at %v, got %v", expectedLoc, cursor.Loc)
	}

	expectedSelStart := Loc{6, 0}
	expectedSelEnd := Loc{10, 0}
	if cursor.CurSelection[0] != expectedSelStart || cursor.CurSelection[1] != expectedSelEnd {
		t.Errorf("Expected selection [%v,%v], got [%v,%v]",
			expectedSelStart, expectedSelEnd, cursor.CurSelection[0], cursor.CurSelection[1])
	}
}

func TestReplaceWithSelection(t *testing.T) {
	buf := NewBufferFromString("hello world test", "", BTDefault)
	cursor := makeCursorWithSelection(buf, 16, 0, 12, 0, 16, 0)
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{0, 0}, Loc{5, 0}, "Hi")

	expectedLoc := Loc{13, 0}
	if cursor.Loc != expectedLoc {
		t.Errorf("Expected cursor.Loc at %v, got %v", expectedLoc, cursor.Loc)
	}

	expectedSelStart := Loc{9, 0}
	expectedSelEnd := Loc{13, 0}
	if cursor.CurSelection[0] != expectedSelStart || cursor.CurSelection[1] != expectedSelEnd {
		t.Errorf("Expected selection [%v,%v], got [%v,%v]",
			expectedSelStart, expectedSelEnd, cursor.CurSelection[0], cursor.CurSelection[1])
	}

	expectedText := "Hi world test"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceMultiLineCursorMovement(t *testing.T) {
	buf := NewBufferFromString("0123456789abcdefghijkLMNOPQrstuvwxyz", "", BTDefault)
	cursor := makeCursorWithSelection(buf, 27, 0, 21, 0, 27, 0)
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{20, 0}, Loc{23, 0}, "ABCDE")

	expectedLoc := Loc{29, 0}
	if cursor.Loc != expectedLoc {
		t.Errorf("Expected cursor.Loc at %v, got %v", expectedLoc, cursor.Loc)
	}

	expectedSelStart := Loc{25, 0}
	expectedSelEnd := Loc{29, 0}
	if cursor.CurSelection[0] != expectedSelStart || cursor.CurSelection[1] != expectedSelEnd {
		t.Errorf("Expected selection [%v,%v], got [%v,%v]",
			expectedSelStart, expectedSelEnd, cursor.CurSelection[0], cursor.CurSelection[1])
	}

	expectedText := "0123456789abcdefghijABCDENOPQrstuvwxyz"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceSelectedTextCursorPosition(t *testing.T) {
	buf := NewBufferFromString("hello world test", "", BTDefault)

	cursor := makeCursorWithSelection(buf, 11, 0, 6, 0, 11, 0)
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{6, 0}, Loc{11, 0}, "X")

	t.Logf("After replace: cursor.Loc=%v, selection=[%v,%v]",
		cursor.Loc, cursor.CurSelection[0], cursor.CurSelection[1])

	expectedLoc := Loc{7, 0}
	if cursor.Loc != expectedLoc {
		t.Errorf("Expected cursor at %v, got %v", expectedLoc, cursor.Loc)
	}

	expectedSelStart := Loc{7, 0}
	expectedSelEnd := Loc{7, 0}
	if cursor.CurSelection[0] != expectedSelStart || cursor.CurSelection[1] != expectedSelEnd {
		t.Errorf("Expected selection [%v,%v], got [%v,%v]",
			expectedSelStart, expectedSelEnd, cursor.CurSelection[0], cursor.CurSelection[1])
	}

	expectedText := "hello X test"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestInsertMultipleCursorsSameLine(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{5, 0})
	cursor2 := NewCursor(buf, Loc{11, 0})
	buf.SetCursors([]*Cursor{cursor1, cursor2})

	buf.Insert(Loc{0, 0}, "Hi ")

	expectedLoc1 := Loc{8, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{14, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedText := "Hi hello world"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestInsertMultipleCursorsBeforeAndAfter(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{3, 0})
	cursor2 := NewCursor(buf, Loc{9, 0})
	buf.SetCursors([]*Cursor{cursor1, cursor2})

	buf.Insert(Loc{6, 0}, "big ")

	expectedLoc1 := Loc{3, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{13, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedText := "hello big world"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestInsertMultipleCursorsDifferentLines(t *testing.T) {
	buf := NewBufferFromString("line1\nline2\nline3", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{5, 0})
	cursor2 := NewCursor(buf, Loc{5, 1})
	cursor3 := NewCursor(buf, Loc{5, 2})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3})

	buf.Insert(Loc{0, 0}, "START ")

	expectedLoc1 := Loc{11, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{5, 1}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedLoc3 := Loc{5, 2}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedText := "START line1\nline2\nline3"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestInsertMultipleCursorsMultilineText(t *testing.T) {
	buf := NewBufferFromString("hello\nworld", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{5, 0})
	cursor2 := NewCursor(buf, Loc{5, 1})
	buf.SetCursors([]*Cursor{cursor1, cursor2})

	buf.Insert(Loc{0, 0}, "A\nB\n")

	expectedLoc1 := Loc{5, 2}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{5, 3}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedText := "A\nB\nhello\nworld"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestInsertMultipleCursorsWithSelections(t *testing.T) {
	buf := NewBufferFromString("hello world test", "", BTDefault)
	cursor1 := makeCursorWithSelection(buf, 5, 0, 0, 0, 5, 0)
	cursor2 := makeCursorWithSelection(buf, 11, 0, 6, 0, 11, 0)
	buf.SetCursors([]*Cursor{cursor1, cursor2})

	buf.Insert(Loc{0, 0}, ">> ")

	expectedLoc1 := Loc{8, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}
	expectedSel1 := [2]Loc{{3, 0}, {8, 0}}
	if cursor1.CurSelection != expectedSel1 {
		t.Errorf("Expected cursor1 selection %v, got %v", expectedSel1, cursor1.CurSelection)
	}

	expectedLoc2 := Loc{14, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}
	expectedSel2 := [2]Loc{{9, 0}, {14, 0}}
	if cursor2.CurSelection != expectedSel2 {
		t.Errorf("Expected cursor2 selection %v, got %v", expectedSel2, cursor2.CurSelection)
	}

	expectedText := ">> hello world test"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestRemoveMultipleCursorsSameLine(t *testing.T) {
	buf := NewBufferFromString("hello world test", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{6, 0})
	cursor2 := NewCursor(buf, Loc{16, 0})
	buf.SetCursors([]*Cursor{cursor1, cursor2})

	buf.Remove(Loc{6, 0}, Loc{12, 0})

	expectedLoc1 := Loc{6, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{10, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedText := "hello test"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestRemoveMultipleCursorsBeforeAndAfter(t *testing.T) {
	buf := NewBufferFromString("hello world test", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{3, 0})
	cursor2 := NewCursor(buf, Loc{8, 0})
	cursor3 := NewCursor(buf, Loc{16, 0})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3})

	buf.Remove(Loc{0, 0}, Loc{6, 0})

	expectedLoc1 := Loc{0, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{2, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedLoc3 := Loc{10, 0}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedText := "world test"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestRemoveMultipleCursorsDifferentLines(t *testing.T) {
	buf := NewBufferFromString("line1\nline2\nline3\nline4", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{5, 0})
	cursor2 := NewCursor(buf, Loc{5, 1})
	cursor3 := NewCursor(buf, Loc{5, 2})
	cursor4 := NewCursor(buf, Loc{5, 3})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3, cursor4})

	buf.Remove(Loc{0, 1}, Loc{0, 3})

	expectedLoc1 := Loc{5, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{0, 1}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedLoc3 := Loc{0, 1}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedLoc4 := Loc{5, 1}
	if cursor4.Loc != expectedLoc4 {
		t.Errorf("Expected cursor4 at %v, got %v", expectedLoc4, cursor4.Loc)
	}

	expectedText := "line1\nline4"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestRemoveMultipleCursorsMultiline(t *testing.T) {
	buf := NewBufferFromString("hello\nworld\ntest\nmore", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{3, 0})
	cursor2 := NewCursor(buf, Loc{3, 2})
	cursor3 := NewCursor(buf, Loc{4, 3})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3})

	buf.Remove(Loc{2, 0}, Loc{2, 2})

	expectedLoc1 := Loc{2, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{3, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedLoc3 := Loc{4, 1}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedText := "hest\nmore"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestRemoveMultipleCursorsWithSelections(t *testing.T) {
	buf := NewBufferFromString("hello world test end", "", BTDefault)
	cursor1 := makeCursorWithSelection(buf, 5, 0, 0, 0, 5, 0)
	cursor2 := makeCursorWithSelection(buf, 16, 0, 12, 0, 16, 0)
	buf.SetCursors([]*Cursor{cursor1, cursor2})

	buf.Remove(Loc{0, 0}, Loc{6, 0})

	expectedLoc1 := Loc{0, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}
	expectedSel1 := [2]Loc{{0, 0}, {0, 0}}
	if cursor1.CurSelection != expectedSel1 {
		t.Errorf("Expected cursor1 selection %v, got %v", expectedSel1, cursor1.CurSelection)
	}

	expectedLoc2 := Loc{10, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}
	expectedSel2 := [2]Loc{{6, 0}, {10, 0}}
	if cursor2.CurSelection != expectedSel2 {
		t.Errorf("Expected cursor2 selection %v, got %v", expectedSel2, cursor2.CurSelection)
	}

	expectedText := "world test end"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceMultipleCursorsSameLine(t *testing.T) {
	buf := NewBufferFromString("hello world test", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{5, 0})
	cursor2 := NewCursor(buf, Loc{11, 0})
	cursor3 := NewCursor(buf, Loc{16, 0})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3})

	buf.Replace(Loc{6, 0}, Loc{11, 0}, "everyone")

	expectedLoc1 := Loc{5, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{14, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedLoc3 := Loc{19, 0}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedText := "hello everyone test"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceMultipleCursorsDifferentLines(t *testing.T) {
	buf := NewBufferFromString("line1\nline2\nline3", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{5, 0})
	cursor2 := NewCursor(buf, Loc{5, 1})
	cursor3 := NewCursor(buf, Loc{5, 2})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3})

	buf.Replace(Loc{0, 0}, Loc{5, 0}, "FIRST")

	expectedLoc1 := Loc{5, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{5, 1}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedLoc3 := Loc{5, 2}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedText := "FIRST\nline2\nline3"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceMultipleCursorsMultilineToSingleLine(t *testing.T) {
	buf := NewBufferFromString("line1\nline2\nline3\nline4", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{3, 0})
	cursor2 := NewCursor(buf, Loc{3, 1})
	cursor3 := NewCursor(buf, Loc{3, 2})
	cursor4 := NewCursor(buf, Loc{3, 3})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3, cursor4})

	buf.Replace(Loc{0, 0}, Loc{0, 2}, "REPLACED")

	expectedLoc1 := Loc{8, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{8, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedLoc3 := Loc{11, 0}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedLoc4 := Loc{3, 1}
	if cursor4.Loc != expectedLoc4 {
		t.Errorf("Expected cursor4 at %v, got %v", expectedLoc4, cursor4.Loc)
	}

	expectedText := "REPLACEDline3\nline4"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceMultipleCursorsSingleLineToMultiline(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{5, 0})
	cursor2 := NewCursor(buf, Loc{11, 0})
	buf.SetCursors([]*Cursor{cursor1, cursor2})

	buf.Replace(Loc{6, 0}, Loc{11, 0}, "new\ntext")

	expectedLoc1 := Loc{5, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{4, 1}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedText := "hello new\ntext"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceMultipleCursorsWithSelections(t *testing.T) {
	buf := NewBufferFromString("hello world test end", "", BTDefault)
	cursor1 := makeCursorWithSelection(buf, 5, 0, 0, 0, 5, 0)
	cursor2 := makeCursorWithSelection(buf, 16, 0, 12, 0, 16, 0)
	cursor3 := makeCursorWithSelection(buf, 20, 0, 17, 0, 20, 0)
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3})

	buf.Replace(Loc{6, 0}, Loc{12, 0}, "XX")

	expectedLoc1 := Loc{5, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{12, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}
	expectedSel2 := [2]Loc{{8, 0}, {12, 0}}
	if cursor2.CurSelection != expectedSel2 {
		t.Errorf("Expected cursor2 selection %v, got %v", expectedSel2, cursor2.CurSelection)
	}

	expectedLoc3 := Loc{16, 0}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}
	expectedSel3 := [2]Loc{{13, 0}, {16, 0}}
	if cursor3.CurSelection != expectedSel3 {
		t.Errorf("Expected cursor3 selection %v, got %v", expectedSel3, cursor3.CurSelection)
	}

	expectedText := "hello XXtest end"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestReplaceMultipleCursorsComplexMultiline(t *testing.T) {
	buf := NewBufferFromString("A\nB\nC\nD\nE", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{1, 0})
	cursor2 := NewCursor(buf, Loc{1, 2})
	cursor3 := NewCursor(buf, Loc{1, 4})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3})

	buf.Replace(Loc{0, 1}, Loc{0, 3}, "X\nY\nZ\n")

	expectedLoc1 := Loc{1, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{0, 4}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedLoc3 := Loc{1, 5}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedText := "A\nX\nY\nZ\nD\nE"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestMultipleCursorsInsertAtSameLocation(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{6, 0})
	cursor2 := NewCursor(buf, Loc{6, 0})
	cursor3 := NewCursor(buf, Loc{11, 0})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3})

	buf.Insert(Loc{6, 0}, "BIG ")

	expectedLoc := Loc{10, 0}
	if cursor1.Loc != expectedLoc {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc, cursor1.Loc)
	}
	if cursor2.Loc != expectedLoc {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc, cursor2.Loc)
	}

	expectedLoc3 := Loc{15, 0}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedText := "hello BIG world"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestMultipleCursorsRemoveAcrossCursors(t *testing.T) {
	buf := NewBufferFromString("0123456789", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{2, 0})
	cursor2 := NewCursor(buf, Loc{5, 0})
	cursor3 := NewCursor(buf, Loc{8, 0})
	buf.SetCursors([]*Cursor{cursor1, cursor2, cursor3})

	buf.Remove(Loc{3, 0}, Loc{7, 0})

	expectedLoc1 := Loc{2, 0}
	if cursor1.Loc != expectedLoc1 {
		t.Errorf("Expected cursor1 at %v, got %v", expectedLoc1, cursor1.Loc)
	}

	expectedLoc2 := Loc{3, 0}
	if cursor2.Loc != expectedLoc2 {
		t.Errorf("Expected cursor2 at %v, got %v", expectedLoc2, cursor2.Loc)
	}

	expectedLoc3 := Loc{4, 0}
	if cursor3.Loc != expectedLoc3 {
		t.Errorf("Expected cursor3 at %v, got %v", expectedLoc3, cursor3.Loc)
	}

	expectedText := "012789"
	if string(buf.Bytes()) != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, string(buf.Bytes()))
	}
}

func TestUpdateTrailingWsInsertAtEOL(t *testing.T) {
	buf := NewBufferFromString("hello", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{5, 0}, "  ")

	if cursor.NewTrailingWsY != 0 {
		t.Errorf("Expected NewTrailingWsY to be 0, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsInsertNonWsAtEOL(t *testing.T) {
	buf := NewBufferFromString("hello", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{5, 0}, "world")

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsInsertTextWithTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{5, 0}, " world  ")

	if cursor.NewTrailingWsY != 0 {
		t.Errorf("Expected NewTrailingWsY to be 0, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsInsertWsOnlyAfterWs(t *testing.T) {
	buf := NewBufferFromString("hello ", "", BTDefault)
	cursor := NewCursor(buf, Loc{6, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{6, 0}, "  ")

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (ws-only after ws), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsInsertMultiLineAtEOL(t *testing.T) {
	buf := NewBufferFromString("hello", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{5, 0}, "\nworld  ")

	if cursor.NewTrailingWsY != 1 {
		t.Errorf("Expected NewTrailingWsY to be 1, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsInsertMultiLineWithoutTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{5, 0}, "\nworld")

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsRemoveAtEOLLeavingTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello  world", "", BTDefault)
	cursor := NewCursor(buf, Loc{7, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{7, 0}, Loc{12, 0})

	if cursor.NewTrailingWsY != 0 {
		t.Errorf("Expected NewTrailingWsY to be 0, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsRemoveAtEOLNoTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{5, 0}, Loc{11, 0})

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsRemoveWsOnlyAtEOL(t *testing.T) {
	buf := NewBufferFromString("hello world  ", "", BTDefault)
	cursor := NewCursor(buf, Loc{11, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{11, 0}, Loc{13, 0})

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (removed ws-only), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsRemoveMultiLineAtEOL(t *testing.T) {
	buf := NewBufferFromString("hello  \nworld\ntest", "", BTDefault)
	cursor := NewCursor(buf, Loc{7, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{7, 0}, Loc{5, 1})

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (removed with newline immediately), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsRemoveMultiLineNonWsAtEOL(t *testing.T) {
	buf := NewBufferFromString("hello world\nmore\ntest", "", BTDefault)
	cursor := NewCursor(buf, Loc{11, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{7, 0}, Loc{4, 1})

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsRemoveMultiLineRevealTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello  text\nmore", "", BTDefault)
	cursor := NewCursor(buf, Loc{7, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{7, 0}, Loc{4, 1})

	if cursor.NewTrailingWsY != 0 {
		t.Errorf("Expected NewTrailingWsY to be 0 (revealed trailing ws by removing non-ws), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsRemoveMultiLineWsOnlyAtEOL(t *testing.T) {
	buf := NewBufferFromString("hello  \n  \nworld", "", BTDefault)
	cursor := NewCursor(buf, Loc{7, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{7, 0}, Loc{0, 2})

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (removed ws-only multiline), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsInsertNotAtEOL(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{5, 0}, "  ")

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (not at EOL), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsLineShiftInsert(t *testing.T) {
	buf := NewBufferFromString("line1\nline2  \nline3", "", BTDefault)
	cursor := NewCursor(buf, Loc{8, 1})
	cursor.Num = 0
	cursor.NewTrailingWsY = 1
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{0, 0}, "newline\n")

	if cursor.NewTrailingWsY != 2 {
		t.Errorf("Expected NewTrailingWsY to shift to 2, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsLineShiftRemove(t *testing.T) {
	buf := NewBufferFromString("line1\nline2\nline3  \nline4", "", BTDefault)
	cursor := NewCursor(buf, Loc{8, 2})
	cursor.Num = 0
	cursor.NewTrailingWsY = 2
	buf.SetCursors([]*Cursor{cursor})

	buf.Remove(Loc{0, 0}, Loc{0, 1})

	if cursor.NewTrailingWsY != 1 {
		t.Errorf("Expected NewTrailingWsY to shift to 1, got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsCursorNotAtEOL(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Insert(Loc{11, 0}, "  ")

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (cursor not at EOL), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsMultipleCursors(t *testing.T) {
	buf := NewBufferFromString("line1\nline2", "", BTDefault)
	cursor1 := NewCursor(buf, Loc{5, 0})
	cursor1.Num = 0
	cursor2 := NewCursor(buf, Loc{5, 1})
	cursor2.Num = 1
	buf.SetCursors([]*Cursor{cursor1, cursor2})

	buf.EventHandler.active = 0
	buf.Insert(Loc{5, 0}, "  ")

	if cursor1.NewTrailingWsY != 0 {
		t.Errorf("Expected cursor1 NewTrailingWsY to be 0, got %d", cursor1.NewTrailingWsY)
	}
	if cursor2.NewTrailingWsY != -1 {
		t.Errorf("Expected cursor2 NewTrailingWsY to be -1, got %d", cursor2.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsReplaceAddingTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{11, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{6, 0}, Loc{11, 0}, "test  ")

	if cursor.NewTrailingWsY != 0 {
		t.Errorf("Expected NewTrailingWsY to be 0 (replace added trailing ws), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsReplaceRemovingTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello world  ", "", BTDefault)
	cursor := NewCursor(buf, Loc{13, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{6, 0}, Loc{13, 0}, "test")

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (replace removed trailing ws), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsReplacePreservingTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello world  ", "", BTDefault)
	cursor := NewCursor(buf, Loc{13, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{6, 0}, Loc{11, 0}, "test")

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (cursor not at EOL after replace), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsReplaceMultiLineWithTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello world\nmore text", "", BTDefault)
	cursor := NewCursor(buf, Loc{11, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{6, 0}, Loc{4, 1}, "test  ")

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (not at EOL), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsReplaceToMultiLineWithTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{11, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{6, 0}, Loc{11, 0}, "new\ntext  ")

	if cursor.NewTrailingWsY != 1 {
		t.Errorf("Expected NewTrailingWsY to be 1 (multiline replace adds trailing ws), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsReplaceAtMiddle(t *testing.T) {
	buf := NewBufferFromString("hello world test", "", BTDefault)
	cursor := NewCursor(buf, Loc{16, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{6, 0}, Loc{11, 0}, "everyone")

	if cursor.NewTrailingWsY != -1 {
		t.Errorf("Expected NewTrailingWsY to be -1 (replace not at cursor EOL), got %d", cursor.NewTrailingWsY)
	}
}

func TestUpdateTrailingWsReplaceEmptyWithTrailingWs(t *testing.T) {
	buf := NewBufferFromString("hello", "", BTDefault)
	cursor := NewCursor(buf, Loc{5, 0})
	cursor.Num = 0
	buf.SetCursors([]*Cursor{cursor})

	buf.Replace(Loc{5, 0}, Loc{5, 0}, "  ")

	if cursor.NewTrailingWsY != 0 {
		t.Errorf("Expected NewTrailingWsY to be 0 (replace empty is insert with trailing ws), got %d", cursor.NewTrailingWsY)
	}
}

func TestMultipleReplace(t *testing.T) {
	buf := NewBufferFromString("hello world", "", BTDefault)
	cursor := NewCursor(buf, Loc{0, 0})
	buf.SetCursors([]*Cursor{cursor})

	deltas := []Delta{
		{[]byte("Earth"), Loc{6, 0}, Loc{11, 0}},
		{[]byte("Hi"), Loc{0, 0}, Loc{5, 0}},
	}

	buf.MultipleReplace(deltas)

	expected := "Hi Earth"
	if string(buf.Bytes()) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(buf.Bytes()))
	}

	buf.Undo()
	expectedUndo := "hello world"
	if string(buf.Bytes()) != expectedUndo {
		t.Errorf("Undo MultipleReplace failed, expected '%s', got '%s'", expectedUndo, string(buf.Bytes()))
	}
}
