package highlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const stringSyntax = `filetype: test

detect:
    filename: "\\.test$"

rules:
    - constant.string:
        start: "\""
        end: "\""
        skip: "\\\\."
        rules:
            - constant.specialChar: "\\\\."
`

func TestHighlightRegionPatterns(t *testing.T) {
	data := []byte(stringSyntax)
	header, err := MakeHeaderYaml(data)
	assert.NoError(t, err)
	file, err := ParseFile(data)
	assert.NoError(t, err)
	def, err := ParseDef(file, header)
	assert.NoError(t, err)

	// The escape starts the string body on every line. On the first line
	// the body starts at column 8 and is 8 characters long.
	lines := []struct {
		text string
		col  int
	}{
		{`xyzw = "\t\t\t\t";`, 8},
		{`xyz = "\t\t\t\t";`, 7},
		{`ab = "\t\t\t\t";`, 6},
	}

	input := ""
	for i, l := range lines {
		if i > 0 {
			input += "\n"
		}
		input += l.text
	}

	matches := NewHighlighter(def).HighlightString(input)
	for i, l := range lines {
		assert.Equal(t, Groups["constant.specialChar"], matches[i][l.col], l.text)
	}
}
