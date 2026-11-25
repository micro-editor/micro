package util

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeZip(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, data := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "test.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUnzipNormal(t *testing.T) {
	content := []byte("hello world")
	z := writeZip(t, map[string][]byte{"a.txt": content})
	dest := t.TempDir()
	if err := Unzip(z, dest); err != nil {
		t.Fatalf("normal unzip failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch: got %q", got)
	}
}

func TestUnzipBomb(t *testing.T) {
	// highly compressible large payload: ~300 MiB of zeros
	big := make([]byte, 300<<20)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.CreateHeader(&zip.FileHeader{Name: "bomb.bin", Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	// write through a flate-level compressor implicitly; raw write of zeros compresses well
	if _, err := f.Write(big); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("zip size: %d bytes for %d MiB payload", buf.Len(), len(big)>>20)
	if buf.Len() > 5<<20 {
		t.Skip("zip not compressing enough for a fast test")
	}
	path := filepath.Join(t.TempDir(), "bomb.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	err = Unzip(path, dest)
	if err == nil {
		t.Fatal("expected zip bomb to be rejected, got nil error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("rejected as expected: %v", err)
}
