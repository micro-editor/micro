package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A character with an accent may be a single precomposed code point, or a
// base letter followed by one or more combining marks. micro treats a base
// letter plus its combining marks as a single character. The two forms are
// visually identical, so they are named here rather than inlined.
const (
	precomposedE     = "é"          // é
	combiningE       = "é"         // e + combining acute
	combiningDoubleE = "é̂"        // e + combining acute + circumflex
	trebleClef       = "\U0001d11e" // 4-byte rune
)

func TestDecodeCharacter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		r     rune
		combc []rune
		size  int
	}{
		{"ascii", "abc", 'a', nil, 1},
		{"two byte rune", "ñb", 'ñ', nil, 2},
		{"three byte rune", "€b", '€', nil, 3},
		{"four byte rune", trebleClef + "b", '\U0001d11e', nil, 4},
		{"precomposed accent", precomposedE + "b", 'é', nil, 2},
		{"combining accent", combiningE, 'e', []rune{'́'}, 3},
		{"multiple combining", combiningDoubleE, 'e', []rune{'́', '̂'}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, combc, size := DecodeCharacter([]byte(tt.input))
			assert.Equal(t, tt.r, r)
			assert.Equal(t, tt.combc, combc)
			assert.Equal(t, tt.size, size)

			r, combc, size = DecodeCharacterInString(tt.input)
			assert.Equal(t, tt.r, r)
			assert.Equal(t, tt.combc, combc)
			assert.Equal(t, tt.size, size)
		})
	}
}

func TestCharacterCount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 0},
		{"ascii", "hello", 5},
		{"precomposed multibyte", "h" + precomposedE + "llo", 5},
		{"combining marks are not counted", "h" + combiningE + "llo", 5},
		{"only combining", combiningE, 1},
		{"cjk", "日本語", 3},
		{"astral plane", trebleClef + trebleClef, 2},
		{"tabs and spaces", "\t a", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, CharacterCount([]byte(tt.input)))
			assert.Equal(t, tt.expected, CharacterCountInString(tt.input))
		})
	}
}

func TestRunePos(t *testing.T) {
	b := []byte("h" + precomposedE + "llo")

	assert.Equal(t, 0, RunePos(b, 0))
	assert.Equal(t, 1, RunePos(b, 1))
	// the precomposed 'é' occupies two bytes, so byte index 3 is rune index 2
	assert.Equal(t, 2, RunePos(b, 3))
	assert.Equal(t, 5, RunePos(b, len(b)))
}
