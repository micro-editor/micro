package buffer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyDiff(t *testing.T) {
	tests := []struct {
		name   string
		before string
		after  string
	}{
		{"unchanged", "one\ntwo\n", "one\ntwo\n"},
		{"insert", "one\ntwo\n", "one\nnew two\n"},
		{"delete", "one\nold two\n", "one\ntwo\n"},
		{"replace", "569\n489\n45\n", "569\n123\n456\n"},
		{"empty", "", ""},
		{"insert into empty", "", "one\ntwo\n"},
		{"delete all", "one\ntwo\n", ""},
		{"add final newline", "one\ntwo", "one\ntwo\n"},
		{"remove final newline", "one\ntwo\n", "one\ntwo"},
		{"blank lines", "one\n\n\ntwo\n", "\none\n\ntwo\n\n"},
		{"remove mark", "Te\u0302st\n", "Test\n"},
		{"add mark", "Test\n", "Te\u0302st\n"},
		{"replace mark", "Te\u0302st\n", "Te\u0301st\n"},
		{"replace base", "Te\u0302st\n", "Ta\u0302st\n"},
		{"multiple marks", "a\u0301\u0327bc\nkeep\n", "a\u0327b\u0302c!\nkeep\n"},
		{"marks around newline", "a\u0301\nb\u0302\n", "a\u0301b\u0302\n"},
		{"split marked line", "a\u0301b\u0302\n", "a\u0301\nb\u0302\n"},
		{"utf8", "\u4e16\u754c \U0001f4da caf\u00e9\n\u0645\n", "\u4e16\u754c! \U0001f4d6 cafe\u0301\n\u0645\u0646\n"},
		{"literal carriage return", "one\rtwo\nend\r", "one\rnew two\nend\r!"},
	}
	for _, endings := range []string{"\n", "\r\n"} {
		format := FFUnix
		name := "LF"
		if endings == "\r\n" {
			format = FFDos
			name = "CRLF"
		}
		for _, tt := range tests {
			t.Run(name+"/"+tt.name, func(t *testing.T) {
				before := strings.ReplaceAll(tt.before, "\n", endings)
				after := strings.ReplaceAll(tt.after, "\n", endings)
				b := NewBufferFromString(before, "", BTDefault)
				defer b.Close()
				b.Endings = FileFormat(format)
				b.ApplyDiff(after)
				if got := string(b.Bytes()); got != after {
					t.Fatalf("ApplyDiff: got %q, want %q", got, after)
				}
				events := b.UndoStack.Len()
				if (events == 0) != (before == after) {
					t.Fatalf("got %d undo events for %q -> %q", events, before, after)
				}
				for i := 0; i < events; i++ {
					b.UndoOneEvent()
				}
				if got := string(b.Bytes()); got != before {
					t.Fatalf("undo: got %q, want %q", got, before)
				}
				for i := 0; i < events; i++ {
					b.RedoOneEvent()
				}
				if got := string(b.Bytes()); got != after {
					t.Fatalf("redo: got %q, want %q", got, after)
				}
			})
		}
	}
}

func TestApplyDiffNormalizesLineEndings(t *testing.T) {
	for _, before := range []string{"one\ntwo\n", "one\r\ntwo\r\n"} {
		for _, after := range []string{"one\ntwo\n", "one\r\ntwo\r\n", "one\r\ntwo\n"} {
			b := NewBufferFromString(before, "", BTDefault)
			b.ApplyDiff(after)
			if got := string(b.Bytes()); got != before {
				t.Errorf("line ending change: got %q, want %q", got, before)
			}
			if b.UndoStack.Len() != 0 {
				t.Error("line ending normalization created text events")
			}
			b.Close()
		}
	}
}

func TestApplyDiffPreservesCursors(t *testing.T) {
	for _, endings := range []string{"\n", "\r\n"} {
		before := "The bottle says DRINK ME" + endings + "Te\u0302st" + endings + "keep"
		after := "The cake says EAT ME" + endings + "Test!" + endings + "keep"
		b := NewBufferFromString(before, "", BTDefault)
		b.GetActiveCursor().GotoLoc(Loc{22, 0})
		b.AddCursor(NewCursor(b, Loc{4, 1}))
		b.AddCursor(NewCursor(b, Loc{2, 2}))
		b.ApplyDiff(after)
		for i, want := range []Loc{{18, 0}, {5, 1}, {2, 2}} {
			if got := b.GetCursor(i).Loc; got != want {
				t.Errorf("cursor %d with %q endings: got %v, want %v", i, endings, got, want)
			}
		}
		for b.UndoStack.Len() > 0 {
			b.UndoOneEvent()
		}
		if got := b.GetActiveCursor().Loc; got != (Loc{22, 0}) {
			t.Errorf("undo cursor: got %v, want {22 0}", got)
		}
		b.Close()
	}
}

func TestApplyDiffPreservesUndoHistory(t *testing.T) {
	b := NewBufferFromString("one\nkeep\n", "", BTDefault)
	defer b.Close()
	b.Insert(Loc{3, 0}, "!")
	prior := b.UndoStack.Peek()
	b.ApplyDiff("two\nkeep\n")
	for b.UndoStack.Len() > 1 {
		b.UndoOneEvent()
	}
	if b.UndoStack.Peek() != prior || string(b.Bytes()) != "one!\nkeep\n" {
		t.Fatalf("reload did not preserve the prior edit: %q", b.Bytes())
	}
	b.UndoOneEvent()
	if got := string(b.Bytes()); got != "one\nkeep\n" {
		t.Fatalf("undo prior edit: got %q", got)
	}
	for b.RedoStack.Len() > 0 {
		b.RedoOneEvent()
	}
	if got := string(b.Bytes()); got != "two\nkeep\n" {
		t.Fatalf("redo history: got %q", got)
	}
}

func TestReOpenDiff(t *testing.T) {
	for _, endings := range []string{"\n", "\r\n"} {
		t.Run(strings.ReplaceAll(endings, "\n", "LF"), func(t *testing.T) {
			before := strings.ReplaceAll("head\nTe\u0302st\nkeep\nremove\ntail\n", "\n", endings)
			after := strings.ReplaceAll("head\nTest\nkeep\ntail\nadded\n", "\n", endings)
			path := filepath.Join(t.TempDir(), "reload.txt")
			if err := os.WriteFile(path, []byte(before), 0600); err != nil {
				t.Fatal(err)
			}
			b, err := NewBufferFromFile(path, BTDefault)
			if err != nil {
				t.Fatal(err)
			}
			defer b.Close()
			b.Settings["savecursor"] = false
			b.Settings["saveundo"] = false
			b.Settings["diffgutter"] = true
			b.SetDiffBase([]byte(before))
			b.GetActiveCursor().GotoLoc(Loc{2, 4})
			if err := os.WriteFile(path, []byte(after), 0600); err != nil {
				t.Fatal(err)
			}
			if err := b.ReOpen(); err != nil {
				t.Fatal(err)
			}
			if got := string(b.Bytes()); got != after {
				t.Fatalf("ReOpen: got %q, want %q", got, after)
			}
			if b.Modified() {
				t.Error("reopened buffer is marked modified")
			}
			if got := b.GetActiveCursor().Loc; got != (Loc{2, 3}) {
				t.Errorf("reopened cursor: got %v, want {2 3}", got)
			}
			checkDiff := func(want []DiffStatus) {
				t.Helper()
				b.UpdateDiff()
				for i, status := range want {
					if got := b.DiffStatus(i); got != status {
						t.Errorf("diffgutter line %d: got %d, want %d", i, got, status)
					}
				}
			}
			statuses := []DiffStatus{DSUnchanged, DSModified, DSUnchanged, DSDeletedAbove, DSAdded, DSUnchanged}
			checkDiff(statuses)
			for b.Undo() {
			}
			if got := string(b.Bytes()); got != before {
				t.Fatalf("undo reopen: got %q, want %q", got, before)
			}
			checkDiff(make([]DiffStatus, 6))
			for b.Redo() {
			}
			if got := string(b.Bytes()); got != after {
				t.Fatalf("redo reopen: got %q, want %q", got, after)
			}
			checkDiff(statuses)
		})
	}
}
