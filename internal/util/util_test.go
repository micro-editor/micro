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

func TestSetAmbiguousWidth(t *testing.T) {
	defer SetAmbiguousWidth("auto")

	// U+03B1 is ambiguous width, U+3042 is wide, U+0061 is narrow
	bytes := []byte("\u03b1\u3042a")

	SetAmbiguousWidth("single")
	assert.Equal(t, 1, StringWidth(bytes, 1, 4))
	assert.Equal(t, 3, StringWidth(bytes, 2, 4))
	assert.Equal(t, 4, StringWidth(bytes, 3, 4))

	SetAmbiguousWidth("double")
	assert.Equal(t, 2, StringWidth(bytes, 1, 4))
	assert.Equal(t, 4, StringWidth(bytes, 2, 4))
	assert.Equal(t, 5, StringWidth(bytes, 3, 4))

	// "auto" restores the width implied by the environment
	SetAmbiguousWidth("auto")
	expected := 1
	if envAmbiguousWide {
		expected = 2
	}
	assert.Equal(t, expected, StringWidth(bytes, 1, 4))
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
