package buffer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func fileComplete(input string) ([]string, []string) {
	b := NewBufferFromString(input, "", BTInfo)
	c := b.GetActiveCursor()
	c.GotoLoc(b.End())
	return FileComplete(b)
}

func TestFileCompleteEscapesSpecialChars(t *testing.T) {
	dir := t.TempDir()
	sep := string(os.PathSeparator)
	assert.NoError(t, os.Mkdir(filepath.Join(dir, "my folder"), 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "my folder", "it's a file.txt"), nil, 0644))

	// the suggestion is the plain name, the completion is escaped
	completions, suggestions := fileComplete("open " + dir + sep + "my")
	assert.Equal(t, []string{"my folder" + sep}, suggestions)
	assert.Equal(t, []string{`\ folder` + sep}, completions)

	// an escaped space does not start a new argument
	completions, suggestions = fileComplete("open " + dir + sep + `my\ folder` + sep + "it")
	assert.Equal(t, []string{"it's a file.txt"}, suggestions)
	assert.Equal(t, []string{`\'s\ a\ file.txt`}, completions)

	completions, _ = fileComplete("open " + dir + sep + `my\ folder` + sep + `it\'s\ a`)
	assert.Equal(t, []string{`\ file.txt`}, completions)
}
