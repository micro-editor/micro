package highlight

import (
	"strings"
	"testing"
)

func testHighlighter(t *testing.T, syntax string) *Highlighter {
	t.Helper()
	f, err := ParseFile([]byte(syntax))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	def, err := ParseDef(f, &Header{})
	if err != nil {
		t.Fatalf("ParseDef: %v", err)
	}
	return NewHighlighter(def)
}

// The rules inside a region must be applied no matter where on the line the
// region begins. The end match is located in the region's own slice of the
// line, so its index must not be compared against the region's absolute start
// offset: when the two happened to be equal, every rule inside the region was
// skipped and the contents were left with the region's own group.
const regionSyntax = `filetype: test

detect:
    filename: "\\.test$"

rules:
    - comment:
        start: "//"
        end: "$"
        rules:
            - todo: "TODO"
`

func TestRegionRulesAppliedAtAnyStartOffset(t *testing.T) {
	for prefix := 0; prefix < 24; prefix++ {
		line := strings.Repeat("x", prefix) + "// TODO"

		h := testHighlighter(t, regionSyntax)

		todo, ok := Groups["todo"]
		if !ok {
			t.Fatal("todo group not defined")
		}

		matches := h.HighlightString(line)

		found := false
		for _, g := range matches[0] {
			if g == todo {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("prefix %d: %q: todo inside the comment was not highlighted", prefix, line)
		}
	}
}

// The rules inside a region must be applied even if it contains a nested region.
const regionSyntax2 = `filetype: test

detect:
    filename: "\\.test$"

rules:
    - comment:
        start: "//"
        end: "$"
        rules:
            - todo: "TODO"
            - comment:
                start: "<"
                end: ">"
                rules:
                    - error: "FIXME"
`

func TestRegionRulesApplyWithNestedRegion(t *testing.T) {
	line := "xxx // TODO < FIXME > "

	h := testHighlighter(t, regionSyntax2)

	todo, ok := Groups["todo"]
	if !ok {
		t.Fatal("todo group not defined")
	}

	matches := h.HighlightString(line)

	found := false
	for _, g := range matches[0] {
		if g == todo {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("%q: todo inside the comment was not highlighted", line)
	}
}
