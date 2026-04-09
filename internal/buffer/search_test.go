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
