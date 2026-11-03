package config

import (
	"strings"
	"testing"

	"github.com/micro-editor/micro/v2/pkg/highlight"
	"github.com/stretchr/testify/assert"
)

// highlightJS returns the name of the highlight group of every character of
// every line of the given JavaScript source
func highlightJS(t *testing.T, src string) [][]string {
	data, err := Asset("syntax/javascript.yaml")
	assert.NoError(t, err)
	f, err := highlight.ParseFile(data)
	assert.NoError(t, err)
	def, err := highlight.ParseDef(f, nil)
	assert.NoError(t, err)

	h := highlight.NewHighlighter(def)
	matches := h.HighlightString(src)

	var groups [][]string
	for i, line := range strings.Split(src, "\n") {
		var cur highlight.Group
		lineGroups := make([]string, len(line))
		for j := range line {
			if g, ok := matches[i][j]; ok {
				cur = g
			}
			lineGroups[j] = cur.String()
		}
		groups = append(groups, lineGroups)
	}
	return groups
}

func TestJavaScriptRegexLiteral(t *testing.T) {
	src := "if (`Let's find out`.match(/['\"]/)) {\n" +
		"  console.log(\"Looks like it does\");\n" +
		"}\n" +
		"x = a / b / c;\n" +
		"y = s.replace(/\\/\\*/g, '') // comment\n" +
		"foo(a, // comment\n" +
		"    b);"
	groups := highlightJS(t, src)

	// the regex literal is highlighted, quotes inside it do not start a string
	line := strings.Split(src, "\n")[0]
	re := strings.Index(line, "/['")
	assert.Equal(t, "constant.regex", groups[0][re])
	assert.Equal(t, "constant.regex", groups[0][re+4])
	assert.Equal(t, "constant.regex", groups[0][re+5])
	assert.NotEqual(t, "constant.regex", groups[0][re+6])

	// the following line is not part of a string
	assert.Equal(t, "", groups[1][2])
	assert.Equal(t, "constant.string", groups[1][14])

	// division is not a regex
	for _, g := range groups[3] {
		assert.NotEqual(t, "constant.regex", g)
	}

	// escaped slashes don't end the regex, comments still work
	line = strings.Split(src, "\n")[4]
	assert.Equal(t, "constant.regex", groups[4][strings.Index(line, "/g")])
	assert.Equal(t, "constant.string", groups[4][strings.Index(line, "''")])
	assert.Equal(t, "comment", groups[4][strings.Index(line, "// ")])
	assert.Equal(t, "comment", groups[5][strings.Index(src, "// comment\n    b")-strings.Index(src, "foo(")])
	assert.Equal(t, "", groups[6][4])
}
