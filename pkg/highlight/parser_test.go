package highlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestResolveIncludesSkipsInvalidFile(t *testing.T) {
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

	assert.NotPanics(t, func() {
		ResolveIncludes(def, []*File{broken})
	})
	assert.Empty(t, def.rules.patterns)
	assert.Empty(t, def.rules.regions[0].rules.patterns)
}

func TestResolveIncludesSelfInclude(t *testing.T) {
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

	ResolveIncludes(def, []*File{f})

	// The region gets the included rules once; the region copied in by
	// that include does not include the same file again.
	inner := def.rules.regions[0]
	require.Len(t, inner.rules.patterns, 1)
	require.Len(t, inner.rules.regions, 1)
	assert.Empty(t, inner.rules.regions[0].rules.patterns)
	assert.Empty(t, inner.rules.regions[0].rules.regions)
}
