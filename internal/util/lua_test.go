package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLuaRuneAt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		idx      int
		expected string
	}{
		{"first", "hello", 0, "h"},
		{"middle", "hello", 1, "e"},
		{"last", "hello", 4, "o"},
		{"past end", "hello", 5, ""},
		{"well past end", "hello", 100, ""},
		{"empty string", "", 0, ""},
		{"multibyte", "h" + precomposedE + "llo", 1, precomposedE},
		{"after multibyte", "h" + precomposedE + "llo", 2, "l"},
		// only the base rune is returned; combining marks are dropped
		{"combining mark", "h" + combiningE + "llo", 1, "e"},
		{"after combining mark", "h" + combiningE + "llo", 2, "l"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, LuaRuneAt(tt.input, tt.idx))
		})
	}
}

func TestLuaGetLeadingWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"spaces", "    hello", "    "},
		{"tabs", "\t\thello", "\t\t"},
		{"mixed", "  \t hello", "  \t "},
		{"none", "hello", ""},
		{"empty", "", ""},
		{"only whitespace", "   ", "   "},
		{"newline is not leading whitespace", "\nhello", ""},
		{"trailing whitespace ignored", "  hello  ", "  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, LuaGetLeadingWhitespace(tt.input))
		})
	}
}

func TestLuaIsWordChar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"letter", "hello", true},
		{"digit", "1abc", true},
		{"underscore", "_abc", true},
		{"multibyte letter", precomposedE, true},
		{"space", " abc", false},
		{"tab", "\tabc", false},
		{"punctuation", ".abc", false},
		{"dash", "-abc", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, LuaIsWordChar(tt.input))
		})
	}
}
