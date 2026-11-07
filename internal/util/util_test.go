package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBytesWhitespaceUnicode(t *testing.T) {
	for _, input := range []string{"", " \t\n\r", "\u00a0", "\u2003", "\u3000", " \u00a0\t\u2003\u3000"} {
		assert.True(t, IsBytesWhitespace([]byte(input)), "expected whitespace: %q", input)
	}
	for _, input := range []string{"text", " \u00a0x\u3000", "\u200b", "\xff", "\x85", "\xa0"} {
		assert.False(t, IsBytesWhitespace([]byte(input)), "expected non-whitespace: %q", input)
	}
}

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
