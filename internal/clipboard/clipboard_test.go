package clipboard

import (
	"testing"
	"time"
)

func TestClipboardGatingIdenticalWrites(t *testing.T) {
	ResetCache()
	CurrentMethod = Internal

	err := Write("sample content", ClipboardReg)
	if err != nil {
		t.Fatalf("expected write to succeed, got %v", err)
	}

	last, ok := LastWritten(ClipboardReg)
	if !ok || last != "sample content" {
		t.Fatalf("expected last written to be 'sample content', got '%s'", last)
	}

	// Repeated identical write should be gated cleanly
	err = Write("sample content", ClipboardReg)
	if err != nil {
		t.Fatalf("gated write should succeed without error, got %v", err)
	}

	// Writing new content updates cache
	err = Write("updated content", ClipboardReg)
	if err != nil {
		t.Fatalf("write of updated content failed: %v", err)
	}

	last, ok = LastWritten(ClipboardReg)
	if !ok || last != "updated content" {
		t.Fatalf("expected last written to be 'updated content', got '%s'", last)
	}
}

func TestClipboardPrimaryRegisterGating(t *testing.T) {
	ResetCache()
	CurrentMethod = Internal

	err := Write("primary text", PrimaryReg)
	if err != nil {
		t.Fatalf("primary write failed: %v", err)
	}

	val, ok := LastWritten(PrimaryReg)
	if !ok || val != "primary text" {
		t.Fatalf("expected primary register to record 'primary text', got '%s'", val)
	}

	// Gated identical write
	err = Write("primary text", PrimaryReg)
	if err != nil {
		t.Fatalf("gated primary write failed: %v", err)
	}
}

func TestClipboardResetCache(t *testing.T) {
	ResetCache()
	CurrentMethod = Internal

	_ = Write("cache test", ClipboardReg)
	_, ok := LastWritten(ClipboardReg)
	if !ok {
		t.Fatalf("expected cached entry before reset")
	}

	ResetCache()
	_, ok = LastWritten(ClipboardReg)
	if ok {
		t.Fatalf("expected cache to be empty after ResetCache()")
	}
}

func TestClipboardWriteDebounce(t *testing.T) {
	ResetCache()
	CurrentMethod = External
	SetWriteDebounce(50 * time.Millisecond)
	defer SetWriteDebounce(0)

	// First write
	_ = write("debounce initial", ClipboardReg, External)
	val, ok := LastWritten(ClipboardReg)
	if !ok || val != "debounce initial" {
		t.Fatalf("initial write should be recorded")
	}

	// Rapid subsequent write within debounce interval should be gated
	_ = write("debounce rapid", ClipboardReg, External)
	val, _ = LastWritten(ClipboardReg)
	if val != "debounce initial" {
		t.Fatalf("rapid write should have been debounced")
	}

	time.Sleep(60 * time.Millisecond)
	_ = write("debounce after window", ClipboardReg, External)
	val, _ = LastWritten(ClipboardReg)
	if val != "debounce after window" {
		t.Fatalf("write after debounce window should succeed, got %s", val)
	}
}
