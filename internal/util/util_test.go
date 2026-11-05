package util

import (
	"os"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringWidth(t *testing.T) {
	bytes := []byte("\tPot să \tmănânc sticlă și ea nu mă rănește.")

	n := StringWidth(bytes, 23, 4)
	assert.Equal(t, 26, n)
}

func TestReplaceHome(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	home := usr.HomeDir

	path, err := ReplaceHome("~")
	assert.NoError(t, err)
	assert.Equal(t, home, path)

	// Both separators are valid on Windows, so `~` has to be recognized
	// regardless of which one follows it.
	for _, sep := range []string{"/", string(os.PathSeparator)} {
		path, err = ReplaceHome("~" + sep + "foo.txt")
		assert.NoError(t, err)
		assert.Equal(t, home+sep+"foo.txt", path)
	}

	path, err = ReplaceHome("foo~bar")
	assert.NoError(t, err)
	assert.Equal(t, "foo~bar", path)
}

func TestSliceVisualEnd(t *testing.T) {
	s := []byte("\thello")
	slc, n, _ := SliceVisualEnd(s, 2, 4)
	assert.Equal(t, []byte("\thello"), slc)
	assert.Equal(t, 2, n)

	slc, n, _ = SliceVisualEnd(s, 1, 4)
	assert.Equal(t, []byte("\thello"), slc)
	assert.Equal(t, 1, n)

	slc, n, _ = SliceVisualEnd(s, 4, 4)
	assert.Equal(t, []byte("hello"), slc)
	assert.Equal(t, 0, n)

	slc, n, _ = SliceVisualEnd(s, 5, 4)
	assert.Equal(t, []byte("ello"), slc)
	assert.Equal(t, 0, n)
}
