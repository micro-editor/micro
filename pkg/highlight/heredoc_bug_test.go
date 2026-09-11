package highlight

import (
	"os"
	"testing"
)

// TestHeredocPrematureEnd verifies that a single word like "DoesNotWork"
// inside a bash heredoc body does not prematurely end the heredoc region.
// See https://github.com/micro-editor/micro/issues/4114
func TestHeredocPrematureEnd(t *testing.T) {
	data, err := os.ReadFile("../../runtime/syntax/sh.yaml")
	if err != nil {
		t.Fatalf("Failed to read syntax file: %v", err)
	}
	file, err := ParseFile(data)
	if err != nil {
		t.Fatalf("Failed to parse syntax file: %v", err)
	}
	def, err := ParseDef(file, nil)
	if err != nil {
		t.Fatalf("Failed to parse syntax def: %v", err)
	}
	h := NewHighlighter(def)

	// heredoc with a single word line inside that should NOT end it
	input := "cat <<HELLO\n# comment\nDoesNotWork\n\nHELLO\n"
	matches := h.HighlightString(input)

	if len(matches) < 5 {
		t.Fatalf("expected at least 5 lines, got %d", len(matches))
	}

	constStringGroup := Groups["constant.string"]

	// Line 2 (index 2, "DoesNotWork") must be entirely inside the heredoc.
	// If the heredoc ended prematurely, group would change to 0.
	for pos, group := range matches[2] {
		if group == 0 && pos > 0 {
			t.Fatalf("heredoc ended prematurely on line 'DoesNotWork' at position %d", pos)
		}
		if group != constStringGroup && group != 0 {
			t.Fatalf("unexpected group %d on line 'DoesNotWork' at position %d", group, pos)
		}
	}

	// Line 4 (index 4, "HELLO") should end the heredoc.
	// The HELLO line starts in the heredoc and transitions to group 0 after the word.
	foundEnd := false
	for group := range matches[4] {
		if group == 0 {
			foundEnd = true
		}
	}
	if !foundEnd {
		t.Fatal("heredoc did not end on the proper delimiter line 'HELLO'")
	}
}
