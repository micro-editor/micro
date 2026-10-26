package util

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringWidth(t *testing.T) {
	bytes := []byte("\tPot să \tmănânc sticlă și ea nu mă rănește.")

	n := StringWidth(bytes, 23, 4)
	assert.Equal(t, 26, n)
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

func writeZip(t *testing.T, add func(w *zip.Writer)) string {
	path := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(path)
	assert.NoError(t, err)
	w := zip.NewWriter(f)
	add(w)
	assert.NoError(t, w.Close())
	assert.NoError(t, f.Close())
	return path
}

func TestUnzip(t *testing.T) {
	src := writeZip(t, func(w *zip.Writer) {
		fw, _ := w.Create("dir/hello.txt")
		fw.Write([]byte("hello"))
	})
	dest := t.TempDir()

	assert.NoError(t, Unzip(src, dest))
	data, err := os.ReadFile(filepath.Join(dest, "dir", "hello.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestUnzipTooLarge(t *testing.T) {
	// A header that claims more than the limit, backed by only a few real bytes
	src := writeZip(t, func(w *zip.Writer) {
		fw, _ := w.CreateRaw(&zip.FileHeader{
			Name:               "bomb.txt",
			Method:             zip.Store,
			CompressedSize64:   5,
			UncompressedSize64: maxUnzipSize + 1,
		})
		fw.Write([]byte("hello"))
	})
	dest := t.TempDir()

	err := Unzip(src, dest)
	assert.EqualError(t, err, "file too large: "+filepath.Join(dest, "bomb.txt"))
	_, err = os.Stat(filepath.Join(dest, "bomb.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestUnzipUnderstatedSize(t *testing.T) {
	// A header that declares 10 bytes in front of 300 real ones. Unzip trusts
	// the declared size because archive/zip refuses to read past it.
	payload := bytes.Repeat([]byte("micro "), 50)
	src := writeZip(t, func(w *zip.Writer) {
		fw, _ := w.CreateRaw(&zip.FileHeader{
			Name:               "bomb.txt",
			Method:             zip.Store,
			CompressedSize64:   uint64(len(payload)),
			UncompressedSize64: 10,
		})
		fw.Write(payload)
	})

	assert.Equal(t, zip.ErrFormat, Unzip(src, t.TempDir()))
}
