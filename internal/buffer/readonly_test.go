package buffer

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func TestReadonlyMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Buffer)
	}{
		{"insert", func(b *Buffer) { b.Insert(b.Start(), "new") }},
		{"remove", func(b *Buffer) { b.Remove(b.Start(), b.End()) }},
		{"event insert", func(b *Buffer) { b.EventHandler.Insert(b.Start(), "new") }},
		{"insert bytes", func(b *Buffer) { b.InsertBytes(b.Start(), []byte("new")) }},
		{"event remove", func(b *Buffer) { b.EventHandler.Remove(b.Start(), b.End()) }},
		{"raw insert", func(b *Buffer) { b.SharedBuffer.insert(b.Start(), []byte("new"), userEdit) }},
		{"raw remove", func(b *Buffer) { b.SharedBuffer.remove(b.Start(), b.End(), userEdit) }},
		{"replace", func(b *Buffer) { b.Replace(b.Start(), Loc{4, 0}, "// comment") }},
		{"multiple replace", func(b *Buffer) { b.MultipleReplace([]Delta{{[]byte("new"), b.Start(), Loc{4, 0}}}) }},
		{"regex replace", func(b *Buffer) { b.ReplaceRegex(b.Start(), b.End(), regexp.MustCompile("word"), []byte("new"), false) }},
		{"diff", func(b *Buffer) { b.ApplyDiff("new\ntext") }},
		{"retab", func(b *Buffer) { b.Settings["tabstospaces"] = true; b.Retab() }},
		{"move lines up", func(b *Buffer) { b.MoveLinesUp(1, 2) }},
		{"move lines down", func(b *Buffer) { b.MoveLinesDown(0, 1) }},
		{"autocomplete", func(b *Buffer) {
			b.Autocomplete(func(*Buffer) ([]string, []string) { return []string{"ing", "s"}, []string{"wording", "words"} })
			b.CycleAutocomplete(true)
		}},
		{"write", func(b *Buffer) { b.Write([]byte("new")) }},
	}
	for _, typ := range []BufType{BTDefault, BTHelp, BTLog} {
		for _, tt := range tests {
			// Log writes are an explicitly supported internal update.
			if typ == BTLog && tt.name == "write" {
				continue
			}
			t.Run(fmt.Sprintf("%s/type%d", tt.name, typ.Kind), func(t *testing.T) {
				b := NewBufferFromString("word\n\ttext", "", typ)
				defer b.Close()
				b.Type.Readonly = true
				b.GetActiveCursor().GotoLoc(Loc{4, 0})
				b.ModifiedThisFrame = false
				cursor := *b.GetActiveCursor()
				tt.edit(b)
				if got := string(b.Bytes()); got != "word\n\ttext" {
					t.Errorf("text changed: %q", got)
				}
				if b.UndoStack.Len() != 0 || b.RedoStack.Len() != 0 {
					t.Error("history changed")
				}
				if !reflect.DeepEqual(cursor, *b.GetActiveCursor()) {
					t.Error("cursor changed")
				}
				if b.Modified() || b.ModifiedThisFrame {
					t.Error("buffer marked modified")
				}
			})
		}
	}
}

func TestReadonlySharedBuffer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.txt")
	b := NewBufferFromString("word", path, BTDefault)
	defer b.Close()
	other := NewBufferFromString("word", path, BTDefault)
	defer other.Close()
	if b.SharedBuffer != other.SharedBuffer {
		t.Fatal("buffers are not shared")
	}
	if err := other.SetOptionNative("readonly", true); err != nil {
		t.Fatal(err)
	}
	b.EventHandler.Replace(b.Start(), b.End(), "edit")
	if string(other.Bytes()) != "word" || b.UndoStack.Len() != 0 {
		t.Error("edit through another view bypassed readonly")
	}
	if err := other.SetOptionNative("readonly", false); err != nil {
		t.Fatal(err)
	}
	b.EventHandler.Replace(b.Start(), b.End(), "edit")
	if string(other.Bytes()) != "edit" {
		t.Error("shared buffer remained read-only")
	}
}

func TestReadonlyPluginEvents(t *testing.T) {
	for _, mode := range []string{"readonly", "become readonly", "internal update", "save cleanup"} {
		t.Run(mode, func(t *testing.T) {
			initial := "word"
			if mode == "save cleanup" {
				initial = "word "
			}
			b := NewBufferFromString(initial, "", BTDefault)
			defer b.Close()
			b.Insert(b.End(), "!")
			b.UndoOneEvent()
			redo := b.RedoStack.Peek()
			b.ModifiedThisFrame = false
			cursor := *b.GetActiveCursor()
			if mode == "internal update" {
				b.Type = BTLog
			} else {
				b.Type.Readonly = mode == "readonly"
			}

			const name = "readonly_test_plugin"
			oldPlugins := config.Plugins
			oldGlobal := ulua.L.GetGlobal(name)
			oldTarget := ulua.L.GetGlobal("readonly_test_target")
			t.Cleanup(func() {
				config.Plugins = oldPlugins
				ulua.L.SetGlobal(name, oldGlobal)
				ulua.L.SetGlobal("readonly_test_target", oldTarget)
			})
			ulua.L.SetGlobal("readonly_test_target", luar.New(ulua.L, b))
			if err := ulua.L.DoString(`
readonly_test_plugin = { calls = 0 }
function readonly_test_plugin.onBeforeTextEvent(buf, event)
    readonly_test_plugin.calls = readonly_test_plugin.calls + 1
    buf.Type.Readonly = true
    readonly_test_target:Insert(readonly_test_target:Start(), "plugin edit")
    return true
end
`); err != nil {
				t.Fatal(err)
			}
			config.Plugins = []*config.Plugin{{Name: name, Loaded: true}}
			want, calls := initial, lua.LNumber(0)
			if mode == "internal update" {
				b.Write([]byte(" output"))
				want, calls = "word output", 1
			} else {
				if mode == "save cleanup" {
					b.Settings["rmtrailingws"] = true
					b.Settings["eofnewline"] = true
					// A directory destination stops the save before filesystem I/O.
					if err := b.SaveAs(t.TempDir()); err == nil {
						t.Fatal("saving to a directory succeeded")
					}
				} else {
					b.Insert(b.Start(), "edit")
				}
				if mode == "become readonly" || mode == "save cleanup" {
					calls = 1
				}
				if b.UndoStack.Len() != 0 || b.RedoStack.Peek() != redo {
					t.Error("rejected event changed history")
				}
				if b.ModifiedThisFrame || !reflect.DeepEqual(cursor, *b.GetActiveCursor()) {
					t.Error("rejected event changed modification state or cursor")
				}
			}
			if got := string(b.Bytes()); got != want {
				t.Errorf("text: got %q, want %q", got, want)
			}
			if got := ulua.L.GetField(ulua.L.GetGlobal(name), "calls"); got != calls {
				t.Errorf("callback count: got %v, want %v", got, calls)
			}
			if !b.Type.Readonly {
				t.Error("callback did not leave buffer read-only")
			}
		})
	}
}

func TestMoveLinesUpPreservesEOFCursor(t *testing.T) {
	b := NewBufferFromString("one\ntwo", "", BTDefault)
	defer b.Close()
	c := b.GetActiveCursor()
	c.GotoLoc(Loc{3, 1})
	c.SetSelectionStart(Loc{0, 1})
	c.SetSelectionEnd(Loc{3, 1})
	b.MoveLinesUp(1, 2)
	if string(b.Bytes()) != "two\none\n" {
		t.Fatal("lines did not move")
	}
	if c.Loc != (Loc{3, 0}) || c.CurSelection != ([2]Loc{{0, 0}, {3, 0}}) {
		t.Errorf("cursor or selection moved away from text: %v, %v", c.Loc, c.CurSelection)
	}
}

func TestReadonlyHistory(t *testing.T) {
	for _, action := range []string{"undo", "redo", "undo one", "redo one", "insert", "replace"} {
		t.Run(action, func(t *testing.T) {
			b := NewBufferFromString("word", "", BTDefault)
			defer b.Close()
			b.Insert(b.End(), "!")
			b.Insert(b.End(), "?")
			b.UndoOneEvent()
			undo, redo := b.UndoStack.Peek(), b.RedoStack.Peek()
			undoCopy, redoCopy := *undo, *redo
			cursor := *b.GetActiveCursor()
			if err := b.SetOptionNative("readonly", true); err != nil {
				t.Fatal(err)
			}
			switch action {
			case "undo":
				if b.Undo() {
					t.Error("undo succeeded")
				}
			case "redo":
				if b.Redo() {
					t.Error("redo succeeded")
				}
			case "undo one":
				b.UndoOneEvent()
			case "redo one":
				b.RedoOneEvent()
			case "insert":
				b.EventHandler.Insert(b.Start(), "new")
			case "replace":
				b.Replace(b.Start(), b.End(), "new")
			}
			if string(b.Bytes()) != "word!" || b.UndoStack.Len() != 1 || b.RedoStack.Len() != 1 {
				t.Fatal("text or history changed")
			}
			if !reflect.DeepEqual(undoCopy, *undo) || !reflect.DeepEqual(redoCopy, *redo) || !reflect.DeepEqual(cursor, *b.GetActiveCursor()) {
				t.Error("event or cursor changed")
			}
			if err := b.SetOptionNative("readonly", false); err != nil {
				t.Fatal(err)
			}
			b.RedoOneEvent()
			if string(b.Bytes()) != "word!?" {
				t.Error("redo history lost")
			}
			b.UndoOneEvent()
			b.UndoOneEvent()
			if string(b.Bytes()) != "word" {
				t.Error("undo history lost")
			}
		})
	}
}

func TestReadonlyTextEvents(t *testing.T) {
	for _, action := range []string{"execute", "do", "do without undo", "undo event", "raw"} {
		t.Run(action, func(t *testing.T) {
			b := NewBufferFromString("word", "", BTHelp)
			defer b.Close()
			event := &TextEvent{C: *b.GetActiveCursor(), EventType: TextEventRemove, Deltas: []Delta{{nil, b.Start(), b.End()}}, Time: time.Now()}
			original := *event
			original.Deltas = append([]Delta(nil), event.Deltas...)
			switch action {
			case "execute":
				b.Execute(event)
			case "do":
				b.DoTextEvent(event, true)
			case "do without undo":
				b.DoTextEvent(event, false)
			case "undo event":
				b.UndoTextEvent(event)
			case "raw":
				ExecuteTextEvent(event, b.SharedBuffer)
			}
			if string(b.Bytes()) != "word" || !reflect.DeepEqual(original, *event) || b.UndoStack.Len() != 0 {
				t.Error("rejected event changed text, history or event data")
			}
		})
	}
}

func TestReadonlyInternalUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "readonly.txt")
	if err := os.WriteFile(path, []byte("new text"), 0600); err != nil {
		t.Fatal(err)
	}
	b := NewBufferFromString("old text", path, BTDefault)
	defer b.Close()
	b.Type.Readonly = true
	if err := b.ReOpen(); err != nil {
		t.Fatal(err)
	}
	if string(b.Bytes()) != "new text" || !b.Type.Readonly || b.Modified() {
		t.Error("reload failed to preserve read-only state")
	}
	b.Replace(b.Start(), b.End(), "edit")
	if string(b.Bytes()) != "new text" {
		t.Error("reload left buffer writable")
	}
	if n, err := b.Write([]byte("edit")); n != 0 || err == nil {
		t.Error("read-only writer did not report rejection")
	}

	log := NewBufferFromString("", "", BTLog)
	defer log.Close()
	if n, err := log.Write([]byte("log output")); n != 10 || err != nil {
		t.Fatalf("log write: %d, %v", n, err)
	}
	if string(log.Bytes()) != "log output" || !log.Type.Readonly {
		t.Error("log output was blocked or buffer became writable")
	}
	log.Replace(log.Start(), log.End(), "edit")
	if string(log.Bytes()) != "log output" {
		t.Error("log write left buffer writable")
	}
}

func TestRetabUndo(t *testing.T) {
	b := NewBufferFromString("\tword\n    text", "", BTDefault)
	defer b.Close()
	b.Settings["tabstospaces"] = true
	b.Settings["tabsize"] = float64(4)
	b.Retab()
	if string(b.Bytes()) != "    word\n    text" {
		t.Fatal("retab failed")
	}
	b.UndoOneEvent()
	if string(b.Bytes()) != "\tword\n    text" {
		t.Fatal("retab was not undoable")
	}
	b.RedoOneEvent()
	if string(b.Bytes()) != "    word\n    text" {
		t.Fatal("retab redo failed")
	}
}
