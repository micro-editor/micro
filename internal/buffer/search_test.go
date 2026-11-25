package buffer

import "testing"

func TestFindNextRegexAcrossLinesDown(t *testing.T) {
	b := NewBufferFromString("abc\ndef\nghi", "", BTDefault)
	defer b.Close()

	m, found, err := b.FindNext(`c\nd`, b.Start(), b.End(), b.Start(), true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("expected match but none found")
	}
	if m[0] != (Loc{2, 0}) || m[1] != (Loc{1, 1}) {
		t.Fatalf("unexpected match locations: got %v", m)
	}
}

func TestFindNextRegexAcrossLinesUpReturnsLastMatch(t *testing.T) {
	b := NewBufferFromString("aa\nbb\nxx\naa\nbb", "", BTDefault)
	defer b.Close()

	m, found, err := b.FindNext(`aa\nbb`, b.Start(), b.End(), b.End(), false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("expected match but none found")
	}
	if m[0] != (Loc{0, 3}) || m[1] != (Loc{2, 4}) {
		t.Fatalf("unexpected match locations: got %v", m)
	}
}

func TestFindNextRegexEscapedBackslashNStaysSingleLine(t *testing.T) {
	b := NewBufferFromString("a\\nb\nx\ny", "", BTDefault)
	defer b.Close()

	m, found, err := b.FindNext(`a\\nb`, b.Start(), b.End(), b.Start(), true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatalf("expected match but none found")
	}
	if m[0] != (Loc{0, 0}) || m[1] != (Loc{4, 0}) {
		t.Fatalf("unexpected match locations: got %v", m)
	}
}

func TestReplaceRegexAcrossLines(t *testing.T) {
	b := NewBufferFromString("foo\nbar\nfoo\nbar", "", BTDefault)
	defer b.Close()

	_, regex, err := b.CompileSearchRegex(`foo\nbar`, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n, _ := b.ReplaceRegex(b.Start(), b.End(), regex, []byte("X"), true)
	if n != 2 {
		t.Fatalf("unexpected replacement count: got %d", n)
	}
	if got := string(b.Bytes()); got != "X\nX" {
		t.Fatalf("unexpected buffer text: got %q", got)
	}
}

func TestReplaceRegexAcrossLinesLiteralReplace(t *testing.T) {
	b := NewBufferFromString("a\nb", "", BTDefault)
	defer b.Close()

	_, regex, err := b.CompileSearchRegex(`a\nb`, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	n, _ := b.ReplaceRegex(b.Start(), b.End(), regex, []byte("$1"), false)
	if n != 1 {
		t.Fatalf("unexpected replacement count: got %d", n)
	}
	if got := string(b.Bytes()); got != "$1" {
		t.Fatalf("unexpected buffer text: got %q", got)
	}
}
