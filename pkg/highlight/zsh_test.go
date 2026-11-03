package highlight

import (
	"os"
	"testing"
)

func TestZshEscapedQuote(t *testing.T) {
	data, err := os.ReadFile("../../runtime/syntax/zsh.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f, err := ParseFile(data)
	if err != nil {
		t.Fatal(err)
	}
	header, err := MakeHeaderYaml(data)
	if err != nil {
		t.Fatal(err)
	}
	def, err := ParseDef(f, header)
	if err != nil {
		t.Fatal(err)
	}

	h := NewHighlighter(def)
	matches := h.HighlightString("printf 'it'\\''s' \\\necho done\nprintf 'a\\' b")
	str := Groups["constant.string"]
	groupAt := func(m LineMatch, pos int) Group {
		g, last := Group(0), -1
		for i, mg := range m {
			if i <= pos && i > last {
				g, last = mg, i
			}
		}
		return g
	}

	// 'it'\''s' is two strings joined by an escaped quote
	if groupAt(matches[0], 17) == str {
		t.Errorf("string continues past the end of line 0: %v", matches[0])
	}
	// a trailing backslash must not leak into the next line
	if groupAt(matches[1], 0) == str {
		t.Errorf("line 1 is highlighted as part of a region: %v", matches[1])
	}
	// backslash has no special meaning inside single quotes
	if groupAt(matches[2], 11) == str {
		t.Errorf("string 'a\\' does not end at the second quote: %v", matches[2])
	}
}
