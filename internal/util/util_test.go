package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringWidth(t *testing.T) {
	bytes := []byte("\tPot să \tmănânc sticlă și ea nu mă rănește.")

	n := StringWidth(bytes, 23, 4)
	assert.Equal(t, 26, n)
}

func TestSliceVisualEnd(t *testing.T) {
	s := []byte("\thello")
	slc, n, _ := SliceVisualEnd(s, 2, 4)
	assert.Equal(t, []byte("\thello"), slc)
	assert.Equal(t, 2, n)

	slc, n, _ = SliceVisualEnd(s, 1, 4)
	assert.Equal(t, []byte("\thello"), slc)
	assert.Equal(t, 1, n)

	slc, n, _ = SliceVisualEnd(s, 4, 4)
	assert.Equal(t, []byte("hello"), slc)
	assert.Equal(t, 0, n)

	slc, n, _ = SliceVisualEnd(s, 5, 4)
	assert.Equal(t, []byte("ello"), slc)
	assert.Equal(t, 0, n)
}

func TestAbs(t *testing.T) {
	assert.Equal(t, 0, Abs(0))
	assert.Equal(t, 5, Abs(5))
	assert.Equal(t, 5, Abs(-5))
	assert.Equal(t, 1, Abs(-1))
}

func TestMinMax(t *testing.T) {
	assert.Equal(t, 1, Min(1, 2))
	assert.Equal(t, 1, Min(2, 1))
	assert.Equal(t, 1, Min(1, 1))
	assert.Equal(t, -2, Min(-1, -2))

	assert.Equal(t, 2, Max(1, 2))
	assert.Equal(t, 2, Max(2, 1))
	assert.Equal(t, 1, Max(1, 1))
	assert.Equal(t, -1, Max(-1, -2))
}

func TestClamp(t *testing.T) {
	assert.Equal(t, 5, Clamp(5, 0, 10))
	assert.Equal(t, 0, Clamp(-5, 0, 10))
	assert.Equal(t, 10, Clamp(15, 0, 10))
	assert.Equal(t, 0, Clamp(0, 0, 10))
	assert.Equal(t, 10, Clamp(10, 0, 10))
	assert.Equal(t, -5, Clamp(-10, -5, 5))
}

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		word bool
	}{
		{"lower letter", 'a', true},
		{"upper letter", 'Z', true},
		{"digit", '7', true},
		{"underscore", '_', true},
		{"multibyte letter", 'é', true},
		{"cjk", '日', true},
		{"space", ' ', false},
		{"tab", '\t', false},
		{"dash", '-', false},
		{"dot", '.', false},
		{"paren", '(', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.word, IsWordChar(tt.r))
			// IsNonWordChar is defined as the negation of IsWordChar
			assert.Equal(t, !tt.word, IsNonWordChar(tt.r))
		})
	}
}

func TestRuneClasses(t *testing.T) {
	assert.True(t, IsSubwordDelimiter('_'))
	assert.False(t, IsSubwordDelimiter('-'))
	assert.False(t, IsSubwordDelimiter('a'))

	assert.True(t, IsAlphanumeric('a'))
	assert.True(t, IsAlphanumeric('Z'))
	assert.True(t, IsAlphanumeric('0'))
	assert.False(t, IsAlphanumeric('_'))
	assert.False(t, IsAlphanumeric(' '))

	assert.True(t, IsUpperLetter('A'))
	assert.False(t, IsUpperLetter('a'))
	// unicode.IsUpper reports true for letters only, not digits
	assert.False(t, IsUpperLetter('1'))

	assert.True(t, IsLowerLetter('a'))
	assert.False(t, IsLowerLetter('A'))
	assert.False(t, IsLowerLetter('1'))

	assert.True(t, IsUpperAlphanumeric('A'))
	assert.True(t, IsUpperAlphanumeric('1'))
	assert.False(t, IsUpperAlphanumeric('a'))

	assert.True(t, IsLowerAlphanumeric('a'))
	assert.True(t, IsLowerAlphanumeric('1'))
	assert.False(t, IsLowerAlphanumeric('A'))

	assert.True(t, IsAutocomplete('.'))
	assert.True(t, IsAutocomplete('a'))
	assert.True(t, IsAutocomplete('_'))
	assert.False(t, IsAutocomplete(' '))
	assert.False(t, IsAutocomplete('-'))
}

func TestIsWhitespace(t *testing.T) {
	assert.True(t, IsWhitespace(' '))
	assert.True(t, IsWhitespace('\t'))
	assert.True(t, IsWhitespace('\n'))
	assert.True(t, IsWhitespace('\r'))
	assert.False(t, IsWhitespace('a'))
	assert.False(t, IsWhitespace('_'))
}

func TestSpaces(t *testing.T) {
	assert.Equal(t, "", Spaces(0))
	assert.Equal(t, " ", Spaces(1))
	assert.Equal(t, "    ", Spaces(4))
}

func TestIsSpaces(t *testing.T) {
	assert.True(t, IsSpaces([]byte("   ")))
	assert.True(t, IsSpaces([]byte("")))
	assert.False(t, IsSpaces([]byte("  \t")))
	assert.False(t, IsSpaces([]byte(" a ")))
}

func TestIsSpacesOrTabs(t *testing.T) {
	assert.True(t, IsSpacesOrTabs([]byte("  \t ")))
	assert.True(t, IsSpacesOrTabs([]byte("")))
	assert.True(t, IsSpacesOrTabs([]byte("\t\t")))
	assert.False(t, IsSpacesOrTabs([]byte(" \n")))
	assert.False(t, IsSpacesOrTabs([]byte("a")))
}

func TestIsBytesWhitespace(t *testing.T) {
	assert.True(t, IsBytesWhitespace([]byte(" \t\n\r")))
	assert.True(t, IsBytesWhitespace([]byte("")))
	assert.False(t, IsBytesWhitespace([]byte(" a ")))
}

func TestGetLeadingWhitespace(t *testing.T) {
	assert.Equal(t, []byte("    "), GetLeadingWhitespace([]byte("    hello")))
	assert.Equal(t, []byte("\t\t"), GetLeadingWhitespace([]byte("\t\thello")))
	assert.Equal(t, []byte("  \t "), GetLeadingWhitespace([]byte("  \t hello")))
	assert.Equal(t, []byte{}, GetLeadingWhitespace([]byte("hello")))
	assert.Equal(t, []byte{}, GetLeadingWhitespace([]byte("")))
	// only spaces and tabs count as leading whitespace here
	assert.Equal(t, []byte{}, GetLeadingWhitespace([]byte("\nhello")))
}

func TestGetTrailingWhitespace(t *testing.T) {
	assert.Equal(t, []byte("  \t"), GetTrailingWhitespace([]byte("hello  \t")))
	assert.Equal(t, []byte("\n"), GetTrailingWhitespace([]byte("hello\n")))
	assert.Equal(t, []byte{}, GetTrailingWhitespace([]byte("hello")))
	assert.Equal(t, []byte{}, GetTrailingWhitespace([]byte("")))
	assert.Equal(t, []byte("   "), GetTrailingWhitespace([]byte("   ")))
}

func TestHasTrailingWhitespace(t *testing.T) {
	assert.True(t, HasTrailingWhitespace([]byte("hello ")))
	assert.True(t, HasTrailingWhitespace([]byte("hello\t")))
	assert.True(t, HasTrailingWhitespace([]byte("hello\n")))
	assert.False(t, HasTrailingWhitespace([]byte("hello")))
	assert.False(t, HasTrailingWhitespace([]byte("")))
}

func TestSliceEnd(t *testing.T) {
	assert.Equal(t, []byte("hello"), SliceEnd([]byte("hello"), 0))
	assert.Equal(t, []byte("llo"), SliceEnd([]byte("hello"), 2))
	assert.Equal(t, []byte(""), SliceEnd([]byte("hello"), 5))
	assert.Equal(t, []byte(""), SliceEnd([]byte("hello"), 10))
	// the index is a rune index, not a byte index
	assert.Equal(t, []byte("llo"), SliceEnd([]byte("h"+precomposedE+"llo"), 2))
}

func TestSliceEndStr(t *testing.T) {
	assert.Equal(t, "hello", SliceEndStr("hello", 0))
	assert.Equal(t, "llo", SliceEndStr("hello", 2))
	assert.Equal(t, "", SliceEndStr("hello", 5))
	assert.Equal(t, "llo", SliceEndStr("h"+precomposedE+"llo", 2))
}

func TestSliceStart(t *testing.T) {
	assert.Equal(t, []byte(""), SliceStart([]byte("hello"), 0))
	assert.Equal(t, []byte("he"), SliceStart([]byte("hello"), 2))
	assert.Equal(t, []byte("hello"), SliceStart([]byte("hello"), 5))
	assert.Equal(t, []byte("hello"), SliceStart([]byte("hello"), 10))
	assert.Equal(t, []byte("h"+precomposedE), SliceStart([]byte("h"+precomposedE+"llo"), 2))
}

func TestSliceStartStr(t *testing.T) {
	assert.Equal(t, "", SliceStartStr("hello", 0))
	assert.Equal(t, "he", SliceStartStr("hello", 2))
	assert.Equal(t, "hello", SliceStartStr("hello", 5))
	assert.Equal(t, "h"+precomposedE, SliceStartStr("h"+precomposedE+"llo", 2))
}

func TestIndexAnyUnquoted(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		chars    string
		expected int
	}{
		{"simple", "a|b", "|", 1},
		{"at start", "|ab", "|", 0},
		{"not present", "abc", "|", -1},
		{"empty string", "", "|", -1},
		{"multiple chars", "ab&c", "|&", 2},
		{"single quoted is skipped", "'a|b'|c", "|", 5},
		{"double quoted is skipped", `"a|b"|c`, "|", 5},
		{"escaped is skipped", `\|a|b`, "|", 3},
		{"unterminated quote", "'a|b", "|", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IndexAnyUnquoted(tt.s, tt.chars))
		})
	}
}

func TestParseBool(t *testing.T) {
	truthy := []string{"on", "1", "t", "T", "true", "TRUE", "True"}
	for _, s := range truthy {
		v, err := ParseBool(s)
		assert.NoError(t, err)
		assert.True(t, v, "expected %q to parse as true", s)
	}

	falsy := []string{"off", "0", "f", "F", "false", "FALSE", "False"}
	for _, s := range falsy {
		v, err := ParseBool(s)
		assert.NoError(t, err)
		assert.False(t, v, "expected %q to parse as false", s)
	}

	invalid := []string{"", "yes", "no", "ON", "OFF", "2"}
	for _, s := range invalid {
		_, err := ParseBool(s)
		assert.Error(t, err, "expected %q to be rejected", s)
	}
}

func TestGetPathAndCursorPosition(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		path   string
		cursor []string
	}{
		{"no position", "util.go", "util.go", nil},
		{"line only", "util.go:10", "util.go", []string{"10", "0"}},
		{"line and column", "util.go:10:5", "util.go", []string{"10", "5"}},
		{"nested path", "internal/util/util.go:10:5", "internal/util/util.go", []string{"10", "5"}},
		{"windows absolute path", `C:\myfile.txt:10:5`, `C:\myfile.txt`, []string{"10", "5"}},
		{"windows path line only", `C:\myfile.txt:10`, `C:\myfile.txt`, []string{"10", "0"}},
		{"trailing colon is not a position", "util.go:", "util.go:", nil},
		{"non numeric suffix", "util.go:abc", "util.go:abc", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cursor := GetPathAndCursorPosition(tt.input)
			assert.Equal(t, tt.path, path)
			assert.Equal(t, tt.cursor, cursor)
		})
	}
}

func TestHashStringMd5(t *testing.T) {
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", HashStringMd5(""))
	assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", HashStringMd5("hello"))
	// the digest is always rendered as 32 lowercase hex characters
	assert.Len(t, HashStringMd5("/some/path/to/a/file.txt"), 32)
}

func TestGetCharPosInLine(t *testing.T) {
	// ascii: one character per visual column
	assert.Equal(t, 0, GetCharPosInLine([]byte("hello"), 0, 4))
	assert.Equal(t, 3, GetCharPosInLine([]byte("hello"), 3, 4))

	// a tab occupies a whole tabstop
	assert.Equal(t, 1, GetCharPosInLine([]byte("\thello"), 4, 4))
	// a visual position inside the tab resolves to the tab itself
	assert.Equal(t, 0, GetCharPosInLine([]byte("\thello"), 2, 4))
	assert.Equal(t, 2, GetCharPosInLine([]byte("\thello"), 5, 4))
}

func TestIntOpt(t *testing.T) {
	assert.Equal(t, 5, IntOpt(float64(5)))
	assert.Equal(t, 0, IntOpt(float64(0)))
	assert.Equal(t, -3, IntOpt(float64(-3)))
	// the conversion truncates toward zero
	assert.Equal(t, 3, IntOpt(3.9))
}

func TestString(t *testing.T) {
	assert.Equal(t, "hello", String([]byte("hello")))
	assert.Equal(t, "", String([]byte{}))
	assert.Equal(t, precomposedE, String([]byte(precomposedE)))
}
