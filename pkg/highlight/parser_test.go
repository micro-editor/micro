package highlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func parseTestDef(t *testing.T, src string) (*Def, *File) {
	t.Helper()
	f, err := ParseFile([]byte(src))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	d, err := ParseDef(f, nil)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return d, f
}

func TestResolveIncludeInvalidFile(t *testing.T) {
	def, _ := parseTestDef(t, `
filetype: outer
rules:
    - include: "broken"
    - default:
        start: "<"
        end: ">"
        rules:
            - include: "broken"
`)

	broken, err := ParseFile([]byte(`
filetype: broken
rules:
    - identifier: "("
`))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.Error(t, ResolveInclude(def, broken))
	assert.Empty(t, def.rules.patterns)
	assert.Empty(t, def.rules.regions[0].rules.patterns)
}

func TestResolveIncludeSelfInclude(t *testing.T) {
	src := `
filetype: self
rules:
    - identifier: "x"
    - default:
        start: "\\("
        end: "\\)"
        rules:
            - include: "self"
`
	def, f := parseTestDef(t, src)

	assert.NoError(t, ResolveInclude(def, f))

	// The region gets the included rules once; the region copied in by
	// that include does not include the same file again.
	inner := def.rules.regions[0]
	if assert.Len(t, inner.rules.patterns, 1) && assert.Len(t, inner.rules.regions, 1) {
		assert.Empty(t, inner.rules.regions[0].rules.patterns)
		assert.Empty(t, inner.rules.regions[0].rules.regions)
	}
}
