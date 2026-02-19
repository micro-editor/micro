package highlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// resetGroups clears the global group registry so tests don't interfere with each other.
func resetGroups() {
	Groups = make(map[string]Group)
	numGroups = 0
}

// makeHighlighter parses a YAML syntax definition and returns a Highlighter.
func makeHighlighter(t *testing.T, yaml string) *Highlighter {
	t.Helper()
	data := []byte(yaml)

	header, err := MakeHeaderYaml(data)
	if err != nil {
		t.Fatalf("MakeHeaderYaml: %v", err)
	}

	f, err := ParseFile(data)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	def, err := ParseDef(f, header)
	if err != nil {
		t.Fatalf("ParseDef: %v", err)
	}

	return NewHighlighter(def)
}

// groupAt returns the Group name at a given character position in a LineMatch.
// It walks backwards from pos to find the most recent color change.
func groupAt(lm LineMatch, pos int) string {
	best := -1
	for k := range lm {
		if k <= pos && k > best {
			best = k
		}
	}
	if best < 0 {
		return ""
	}
	return lm[best].String()
}

func TestLookahead(t *testing.T) {
	resetGroups()
	assert := assert.New(t)

	h := makeHighlighter(t, `
filetype: test-lookahead
detect:
    filename: "\\.test$"
rules:
    - identifier.function: "\\w+(?=\\()"
`)

	matches := h.HighlightString("foo(bar)")
	assert.Len(matches, 1)

	lm := matches[0]

	// "foo" (positions 0-2) should be highlighted as identifier.function
	assert.Equal("identifier.function", groupAt(lm, 0))
	assert.Equal("identifier.function", groupAt(lm, 1))
	assert.Equal("identifier.function", groupAt(lm, 2))

	// "(" at position 3 should NOT be identifier.function (lookahead doesn't consume)
	assert.NotEqual("identifier.function", groupAt(lm, 3))

	// "bar" should not match (not followed by "(")
	assert.NotEqual("identifier.function", groupAt(lm, 4))
}

func TestNegativeLookahead(t *testing.T) {
	resetGroups()
	assert := assert.New(t)

	h := makeHighlighter(t, `
filetype: test-neg-lookahead
detect:
    filename: "\\.test$"
rules:
    - identifier: "foo(?!bar)"
`)

	matches := h.HighlightString("foobar foobaz")
	assert.Len(matches, 1)

	lm := matches[0]

	// "foobar": "foo" is followed by "bar", so negative lookahead fails — no match
	assert.NotEqual("identifier", groupAt(lm, 0))

	// "foobaz": "foo" at position 7 is NOT followed by "bar", so it matches
	assert.Equal("identifier", groupAt(lm, 7))
	assert.Equal("identifier", groupAt(lm, 8))
	assert.Equal("identifier", groupAt(lm, 9))

	// "baz" at position 10 should not be highlighted
	assert.NotEqual("identifier", groupAt(lm, 10))
}

func TestLookbehind(t *testing.T) {
	resetGroups()
	assert := assert.New(t)

	h := makeHighlighter(t, `
filetype: test-lookbehind
detect:
    filename: "\\.test$"
rules:
    - identifier.field: "(?<=\\.)\\w+"
`)

	matches := h.HighlightString("obj.field")
	assert.Len(matches, 1)

	lm := matches[0]

	// "obj" (positions 0-2) should NOT be highlighted
	assert.NotEqual("identifier.field", groupAt(lm, 0))

	// "." at position 3 should NOT be highlighted (lookbehind doesn't consume)
	assert.NotEqual("identifier.field", groupAt(lm, 3))

	// "field" (positions 4-8) should be highlighted
	assert.Equal("identifier.field", groupAt(lm, 4))
	assert.Equal("identifier.field", groupAt(lm, 5))
	assert.Equal("identifier.field", groupAt(lm, 8))
}

func TestNegativeLookbehind(t *testing.T) {
	resetGroups()
	assert := assert.New(t)

	h := makeHighlighter(t, `
filetype: test-neg-lookbehind
detect:
    filename: "\\.test$"
rules:
    - identifier: "(?<!\\.)\\b\\w+\\b"
`)

	matches := h.HighlightString("obj.field")
	assert.Len(matches, 1)

	lm := matches[0]

	// "obj" (positions 0-2) should be highlighted — not preceded by "."
	assert.Equal("identifier", groupAt(lm, 0))
	assert.Equal("identifier", groupAt(lm, 2))

	// "field" (positions 4-8) should NOT be highlighted — preceded by "."
	assert.NotEqual("identifier", groupAt(lm, 4))
}

func TestLookaheadInRegion(t *testing.T) {
	resetGroups()
	assert := assert.New(t)

	h := makeHighlighter(t, `
filetype: test-region-lookahead
detect:
    filename: "\\.test$"
rules:
    - constant.string:
        start: "\""
        end: "\""
        rules:
            - special: "\\w+(?=!)"
`)

	matches := h.HighlightString(`"hello!"`)
	assert.Len(matches, 1)

	lm := matches[0]

	// Position 0 is the opening `"` — should be constant.string (the region delimiter)
	assert.Equal("constant.string", groupAt(lm, 0))

	// "hello" at positions 1-5 inside the string should get "special" via lookahead
	assert.Equal("special", groupAt(lm, 1))
	assert.Equal("special", groupAt(lm, 5))

	// "!" at position 6 should revert to constant.string (not consumed by lookahead)
	assert.Equal("constant.string", groupAt(lm, 6))
}

func TestBasicHighlighting(t *testing.T) {
	resetGroups()
	assert := assert.New(t)

	h := makeHighlighter(t, `
filetype: test-basic
detect:
    filename: "\\.test$"
rules:
    - keyword: "\\b(if|else|while)\\b"
    - constant.number: "\\b\\d+\\b"
`)

	matches := h.HighlightString("if x 42 else")
	assert.Len(matches, 1)

	lm := matches[0]

	// "if" at positions 0-1
	assert.Equal("keyword", groupAt(lm, 0))
	assert.Equal("keyword", groupAt(lm, 1))

	// "x" at position 3 — no highlight
	assert.Equal("", groupAt(lm, 3))

	// "42" at positions 5-6
	assert.Equal("constant.number", groupAt(lm, 5))
	assert.Equal("constant.number", groupAt(lm, 6))

	// "else" at positions 8-11
	assert.Equal("keyword", groupAt(lm, 8))
	assert.Equal("keyword", groupAt(lm, 11))
}
