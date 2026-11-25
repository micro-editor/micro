package buffer

import (
	"math/rand"
	"strings"
	"testing"
)

func TestApplyDiffUnchangedLeadingMarks(t *testing.T) {
	for _, text := range []string{"\u0301a", "\u0301", "first\n\u0301ab", "first\r\n\u0301ab"} {
		b := NewBufferFromString(text, "", BTDefault)
		b.GetActiveCursor().GotoLoc(Loc{1, 0})
		cursor := b.GetActiveCursor().Loc
		b.ApplyDiff(text)
		if got := string(b.Bytes()); got != text {
			t.Errorf("unchanged reload: got %q, want %q", got, text)
		}
		if b.UndoStack.Len() != 0 || b.GetActiveCursor().Loc != cursor {
			t.Error("unchanged reload altered undo history or cursor")
		}
		b.Close()
	}
}

func TestApplyDiffManyDistinctCharacters(t *testing.T) {
	// More than 0xd800 distinct characters exercises rune IDs across the
	// surrogate range, which cannot be represented in UTF-8 strings.
	var text strings.Builder
	for r := rune(0x20000); r < 0x20000+56000; r++ {
		text.WriteRune(r)
		text.WriteString("\u0301")
	}
	before := text.String()
	after := before + "!"
	b := NewBufferFromString(before, "", BTDefault)
	defer b.Close()
	b.ApplyDiff(after)
	if got := string(b.Bytes()); got != after {
		t.Fatal("reload corrupted a buffer with many distinct characters")
	}
	b.UndoOneEvent()
	if got := string(b.Bytes()); got != before {
		t.Fatal("undo corrupted a buffer with many distinct characters")
	}
}

func TestApplyDiffRandomEdits(t *testing.T) {
	random := rand.New(rand.NewSource(3303))
	characters := []string{"a", "b", " ", "e\u0301", "a\u0302\u0327", "\u4e16", "\U0001f4da", "\n"}
	randomText := func() string {
		var text strings.Builder
		for n := random.Intn(60); n > 0; n-- {
			text.WriteString(characters[random.Intn(len(characters))])
		}
		return text.String()
	}
	for i := 0; i < 200; i++ {
		before, after := randomText(), randomText()
		endings := FFUnix
		if i%2 == 0 {
			before = strings.ReplaceAll(before, "\n", "\r\n")
			after = strings.ReplaceAll(after, "\n", "\r\n")
			endings = FFDos
		}
		b := NewBufferFromString(before, "", BTDefault)
		b.Endings = FileFormat(endings)
		b.ApplyDiff(after)
		if got := string(b.Bytes()); got != after {
			t.Fatalf("case %d: %q -> %q produced %q", i, before, after, got)
		}
		for b.UndoStack.Len() > 0 {
			b.UndoOneEvent()
		}
		if got := string(b.Bytes()); got != before {
			t.Fatalf("case %d: undo got %q, want %q", i, got, before)
		}
		for b.RedoStack.Len() > 0 {
			b.RedoOneEvent()
		}
		if got := string(b.Bytes()); got != after {
			t.Fatalf("case %d: redo got %q, want %q", i, got, after)
		}
		b.Close()
	}
}
