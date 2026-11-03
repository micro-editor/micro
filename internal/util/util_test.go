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

func TestDecodeCharacterJoinControl(t *testing.T) {
	// ZWNJ and ZWJ are zero-width and must be kept together with
	// the preceding character instead of becoming separate characters
	s := "sl‌‌‍am"

	r, combc, size := DecodeCharacterInString(s[1:])
	assert.Equal(t, 'l', r)
	assert.Equal(t, []rune{'‌', '‌', '‍'}, combc)
	assert.Equal(t, 10, size)

	r, combc, size = DecodeCharacter([]byte(s[1:]))
	assert.Equal(t, 'l', r)
	assert.Equal(t, []rune{'‌', '‌', '‍'}, combc)
	assert.Equal(t, 10, size)

	assert.Equal(t, 4, CharacterCountInString(s))
	assert.Equal(t, 4, CharacterCount([]byte(s)))
}
