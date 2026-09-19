package buffer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestFileComplete(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "foo.txt"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := os.Mkdir(filepath.Join(dir, "bar"), 0700); err != nil {
		t.Fatal(err)
	}

	complete := func(input string) []string {
		b := NewBufferFromString(input, "", BTDefault)
		b.GetActiveCursor().X = util.CharacterCountInString(input)
		_, suggestions := FileComplete(b)
		return suggestions
	}

	// Both separators are valid on Windows, so completion has to work with
	// either one of them and has to stay with the one already in use.
	for _, sep := range []string{"/", string(os.PathSeparator)} {
		prefix := "open " + filepath.ToSlash(dir) + sep

		assert.Equal(t, []string{"foo.txt"}, complete(prefix+"fo"))
		assert.Equal(t, []string{"bar" + sep}, complete(prefix+"ba"))
	}
}
