package buffer

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

type operation struct {
	start Loc
	end   Loc
	text  []string
}

func init() {
	ulua.L = lua.NewState()
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
	config.GlobalSettings["backup"] = false
	config.GlobalSettings["fastdirty"] = true
}

func check(t *testing.T, before []string, operations []operation, after []string) {
	assert := assert.New(t)

	b := NewBufferFromString(strings.Join(before, "\n"), "", BTDefault)

	assert.NotEqual("", b.GetName())
	assert.Equal(false, b.ExternallyModified())
	assert.Equal(false, b.Modified())
	assert.Equal(1, b.NumCursors())

	checkText := func(lines []string) {
		assert.Equal([]byte(strings.Join(lines, "\n")), b.Bytes())
		assert.Equal(len(lines), b.LinesNum())
		for i, s := range lines {
			assert.Equal(s, b.Line(i))
			assert.Equal([]byte(s), b.LineBytes(i))
		}
	}

	checkText(before)

	var cursors []*Cursor

	for _, op := range operations {
		cursor := NewCursor(b, op.start)
		cursor.SetSelectionStart(op.start)
		cursor.SetSelectionEnd(op.end)
		b.AddCursor(cursor)
		cursors = append(cursors, cursor)
	}

	assert.Equal(1+len(operations), b.NumCursors())

	for i, op := range operations {
		cursor := cursors[i]
		b.SetCurCursor(cursor.Num)
		cursor.DeleteSelection()
		b.Insert(cursor.Loc, strings.Join(op.text, "\n"))
	}

	checkText(after)

	// must have exactly two events per operation (delete and insert)
	for range operations {
		b.UndoOneEvent()
		b.UndoOneEvent()
	}

	checkText(before)

	for i, op := range operations {
		cursor := cursors[i]
		if op.start == op.end {
			assert.Equal(op.start, cursor.Loc)
		} else {
			assert.Equal(op.start, cursor.CurSelection[0])
			assert.Equal(op.end, cursor.CurSelection[1])
		}
	}

	for range operations {
		b.RedoOneEvent()
		b.RedoOneEvent()
	}

	checkText(after)

	b.Close()
}

const maxLineLength = 200

var alphabet = []rune(" abcdeäم📚")

func randomString(length int) string {
	runes := make([]rune, length)
	for i := range runes {
		runes[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(runes)
}

func randomText(nLines int) string {
	lines := make([]string, nLines)
	for i := range lines {
		lines[i] = randomString(rand.Intn(maxLineLength + 1))
	}
	return strings.Join(lines, "\n")
}

func benchCreateAndClose(testingB *testing.B, nLines int) {
	rand.Seed(int64(nLines))

	text := randomText(nLines)

	testingB.ResetTimer()

	for i := 0; i < testingB.N; i++ {
		b := NewBufferFromString(text, "", BTDefault)
		b.Close()
	}
}

func benchRead(testingB *testing.B, nLines int) {
	rand.Seed(int64(nLines))

	b := NewBufferFromString(randomText(nLines), "", BTDefault)

	testingB.ResetTimer()

	for i := 0; i < testingB.N; i++ {
		b.Bytes()
		for j := 0; j < b.LinesNum(); j++ {
			b.Line(j)
			b.LineBytes(j)
		}
	}

	testingB.StopTimer()

	b.Close()
}

func benchEdit(testingB *testing.B, nLines, nCursors int) {
	rand.Seed(int64(nLines + nCursors))

	b := NewBufferFromString(randomText(nLines), "", BTDefault)

	regionSize := nLines / nCursors

	operations := make([]operation, nCursors)
	for i := range operations {
		startLine := (i * regionSize) + rand.Intn(regionSize-5)
		startColumn := rand.Intn(util.CharacterCountInString(b.Line(startLine)) + 1)
		endLine := startLine + 1 + rand.Intn(5)
		endColumn := rand.Intn(util.CharacterCountInString(b.Line(endLine)) + 1)

		operations[i] = operation{
			start: Loc{startColumn, startLine},
			end:   Loc{endColumn, endLine},
			text:  []string{randomText(2 + rand.Intn(4))},
		}
	}

	testingB.ResetTimer()

	for i := 0; i < testingB.N; i++ {
		b.SetCursors([]*Cursor{})

		var cursors []*Cursor

		for _, op := range operations {
			cursor := NewCursor(b, op.start)
			cursor.SetSelectionStart(op.start)
			cursor.SetSelectionEnd(op.end)
			b.AddCursor(cursor)
			cursors = append(cursors, cursor)
		}

		for j, op := range operations {
			cursor := cursors[j]
			b.SetCurCursor(cursor.Num)
			cursor.DeleteSelection()
			b.Insert(cursor.Loc, op.text[0])
		}

		for b.UndoStack.Peek() != nil {
			b.UndoOneEvent()
		}
	}

	testingB.StopTimer()

	b.Close()
}

// Plugins can log from preinit and from the log buffer's own onBufferOpen,
// both of which run before LogBuf is assigned.
func TestWriteLogBeforeLogBufExists(t *testing.T) {
	saved := LogBuf
	t.Cleanup(func() { LogBuf = saved })

	LogBuf = nil
	WriteLog("dropped")

	LogBuf = NewBufferFromString("", "", BTLog)
	t.Cleanup(LogBuf.Close)
	WriteLog("kept")

	assert.Equal(t, []byte("kept"), LogBuf.Bytes())
}

func BenchmarkCreateAndClose10Lines(b *testing.B) {
	benchCreateAndClose(b, 10)
}

func BenchmarkCreateAndClose100Lines(b *testing.B) {
	benchCreateAndClose(b, 100)
}

func BenchmarkCreateAndClose1000Lines(b *testing.B) {
	benchCreateAndClose(b, 1000)
}

func BenchmarkCreateAndClose10000Lines(b *testing.B) {
	benchCreateAndClose(b, 10000)
}

func BenchmarkCreateAndClose100000Lines(b *testing.B) {
	benchCreateAndClose(b, 100000)
}

func BenchmarkCreateAndClose1000000Lines(b *testing.B) {
	benchCreateAndClose(b, 1000000)
}

func BenchmarkRead10Lines(b *testing.B) {
	benchRead(b, 10)
}

func BenchmarkRead100Lines(b *testing.B) {
	benchRead(b, 100)
}

func BenchmarkRead1000Lines(b *testing.B) {
	benchRead(b, 1000)
}

func BenchmarkRead10000Lines(b *testing.B) {
	benchRead(b, 10000)
}

func BenchmarkRead100000Lines(b *testing.B) {
	benchRead(b, 100000)
}

func BenchmarkRead1000000Lines(b *testing.B) {
	benchRead(b, 1000000)
}

func BenchmarkEdit10Lines1Cursor(b *testing.B) {
	benchEdit(b, 10, 1)
}

func BenchmarkEdit100Lines1Cursor(b *testing.B) {
	benchEdit(b, 100, 1)
}

func BenchmarkEdit100Lines10Cursors(b *testing.B) {
	benchEdit(b, 100, 10)
}

func BenchmarkEdit1000Lines1Cursor(b *testing.B) {
	benchEdit(b, 1000, 1)
}

func BenchmarkEdit1000Lines10Cursors(b *testing.B) {
	benchEdit(b, 1000, 10)
}

func BenchmarkEdit1000Lines100Cursors(b *testing.B) {
	benchEdit(b, 1000, 100)
}

func BenchmarkEdit10000Lines1Cursor(b *testing.B) {
	benchEdit(b, 10000, 1)
}

func BenchmarkEdit10000Lines10Cursors(b *testing.B) {
	benchEdit(b, 10000, 10)
}

func BenchmarkEdit10000Lines100Cursors(b *testing.B) {
	benchEdit(b, 10000, 100)
}

func BenchmarkEdit10000Lines1000Cursors(b *testing.B) {
	benchEdit(b, 10000, 1000)
}

func BenchmarkEdit100000Lines1Cursor(b *testing.B) {
	benchEdit(b, 100000, 1)
}

func BenchmarkEdit100000Lines10Cursors(b *testing.B) {
	benchEdit(b, 100000, 10)
}

func BenchmarkEdit100000Lines100Cursors(b *testing.B) {
	benchEdit(b, 100000, 100)
}

func BenchmarkEdit100000Lines1000Cursors(b *testing.B) {
	benchEdit(b, 100000, 1000)
}

func BenchmarkEdit1000000Lines1Cursor(b *testing.B) {
	benchEdit(b, 1000000, 1)
}

func BenchmarkEdit1000000Lines10Cursors(b *testing.B) {
	benchEdit(b, 1000000, 10)
}

func BenchmarkEdit1000000Lines100Cursors(b *testing.B) {
	benchEdit(b, 1000000, 100)
}

func BenchmarkEdit1000000Lines1000Cursors(b *testing.B) {
	benchEdit(b, 1000000, 1000)
}
