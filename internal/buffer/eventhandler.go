package buffer

import (
	"bytes"
	"time"

	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
	dmp "github.com/sergi/go-diff/diffmatchpatch"
	luar "layeh.com/gopher-luar"
)

const (
	// Opposite and undoing events must have opposite values

	// TextEventInsert represents an insertion event
	TextEventInsert = 1
	// TextEventRemove represents a deletion event
	TextEventRemove = -1
	// TextEventReplace represents a replace event
	TextEventReplace = 0

	undoThreshold = 1000 // If two events are less than n milliseconds apart, undo both of them
)

// TextEvent holds data for a manipulation on some text that can be undone
type TextEvent struct {
	C Cursor

	EventType int
	Deltas    []Delta
	Time      time.Time
}

// A Delta is a change to the buffer
type Delta struct {
	Text  []byte
	Start Loc
	End   Loc
}

// Internal updates carry permission on the call, never on the shared buffer.
// A plugin called during an internal update still uses the guarded public API.
type textEventPermission bool

const (
	userEdit     textEventPermission = false
	internalEdit textEventPermission = true
)

// DoTextEvent runs a text event
func (eh *EventHandler) DoTextEvent(t *TextEvent, useUndo bool) {
	eh.doTextEvent(t, useUndo, userEdit)
}

func (eh *EventHandler) doTextEvent(t *TextEvent, useUndo bool, permission textEventPermission) {
	if !eh.buf.canEdit(permission) {
		return
	}
	oldl := eh.buf.LinesNum()

	if useUndo {
		if !eh.execute(t, permission) {
			return
		}
	} else {
		executeTextEvent(t, eh.buf, permission)
	}

	if len(t.Deltas) != 1 {
		return
	}

	text := t.Deltas[0].Text
	start := t.Deltas[0].Start
	lastnl := -1
	var endX int
	var textX int
	if t.EventType == TextEventInsert {
		linecount := eh.buf.LinesNum() - oldl
		textcount := util.CharacterCount(text)
		lastnl = bytes.LastIndex(text, []byte{'\n'})
		if lastnl >= 0 {
			endX = util.CharacterCount(text[lastnl+1:])
			textX = endX
		} else {
			endX = start.X + textcount
			textX = textcount
		}
		t.Deltas[0].End = clamp(Loc{endX, start.Y + linecount}, eh.buf.LineArray)
	}
	end := t.Deltas[0].End

	for _, c := range eh.cursors {
		move := func(loc Loc) Loc {
			if t.EventType == TextEventInsert {
				if start.Y != loc.Y && loc.GreaterThan(start) {
					loc.Y += end.Y - start.Y
				} else if loc.Y == start.Y && loc.GreaterEqual(start) {
					loc.Y += end.Y - start.Y
					if lastnl >= 0 {
						loc.X += textX - start.X
					} else {
						loc.X += textX
					}
				}
				return loc
			} else {
				if loc.Y != end.Y && loc.GreaterThan(end) {
					loc.Y -= end.Y - start.Y
				} else if loc.Y == end.Y && loc.GreaterEqual(end) {
					loc = loc.MoveLA(-DiffLA(start, end, eh.buf.LineArray), eh.buf.LineArray)
				}
				return loc
			}
		}
		c.Loc = move(c.Loc)
		c.CurSelection[0] = move(c.CurSelection[0])
		c.CurSelection[1] = move(c.CurSelection[1])
		c.OrigSelection[0] = move(c.OrigSelection[0])
		c.OrigSelection[1] = move(c.OrigSelection[1])
		c.Relocate()
		c.StoreVisualX()
	}

	if useUndo {
		eh.updateTrailingWs(t)
	}
}

// ExecuteTextEvent runs a text event
func ExecuteTextEvent(t *TextEvent, buf *SharedBuffer) {
	if !buf.canEdit(userEdit) {
		return
	}
	executeTextEvent(t, buf, userEdit)
}

// executeTextEvent applies an event whose permission has already been checked.
func executeTextEvent(t *TextEvent, buf *SharedBuffer, permission textEventPermission) {
	if t.EventType == TextEventInsert {
		for _, d := range t.Deltas {
			buf.insert(d.Start, d.Text, permission)
		}
	} else if t.EventType == TextEventRemove {
		for i, d := range t.Deltas {
			t.Deltas[i].Text = buf.remove(d.Start, d.End, permission)
		}
	} else if t.EventType == TextEventReplace {
		for i, d := range t.Deltas {
			t.Deltas[i].Text = buf.remove(d.Start, d.End, permission)
			buf.insert(d.Start, d.Text, permission)
			t.Deltas[i].Start = d.Start
			t.Deltas[i].End = Loc{d.Start.X + util.CharacterCount(d.Text), d.Start.Y}
		}
		for i, j := 0, len(t.Deltas)-1; i < j; i, j = i+1, j-1 {
			t.Deltas[i], t.Deltas[j] = t.Deltas[j], t.Deltas[i]
		}
	}
}

// UndoTextEvent undoes a text event
func (eh *EventHandler) UndoTextEvent(t *TextEvent) {
	if !eh.buf.canEdit(userEdit) {
		return
	}
	t.EventType = -t.EventType
	eh.DoTextEvent(t, false)
}

// EventHandler executes text manipulations and allows undoing and redoing
type EventHandler struct {
	buf       *SharedBuffer
	cursors   []*Cursor
	active    int
	UndoStack *TEStack
	RedoStack *TEStack
}

// NewEventHandler returns a new EventHandler
func NewEventHandler(buf *SharedBuffer, cursors []*Cursor) *EventHandler {
	eh := new(EventHandler)
	eh.UndoStack = new(TEStack)
	eh.RedoStack = new(TEStack)
	eh.buf = buf
	eh.cursors = cursors
	return eh
}

// ApplyDiff takes a string and runs the necessary insertion and deletion events to make
// the buffer equal to that string
// This means that we can transform the buffer into any string and still preserve undo/redo
// through insert and delete events
func (eh *EventHandler) ApplyDiff(new string) {
	eh.applyDiff(new, userEdit)
}

func (eh *EventHandler) applyDiff(new string, permission textEventPermission) {
	if !eh.buf.canEdit(permission) {
		return
	}
	differ := dmp.New()
	diff := differ.DiffMain(string(eh.buf.Bytes()), new, false)
	loc := eh.buf.Start()
	for _, d := range diff {
		if d.Type == dmp.DiffDelete {
			eh.remove(loc, loc.MoveLA(util.CharacterCountInString(d.Text), eh.buf.LineArray), permission)
		} else {
			if d.Type == dmp.DiffInsert {
				eh.insertBytes(loc, []byte(d.Text), permission)
			}
			loc = loc.MoveLA(util.CharacterCountInString(d.Text), eh.buf.LineArray)
		}
	}
}

// Insert creates an insert text event and executes it
func (eh *EventHandler) Insert(start Loc, textStr string) {
	text := []byte(textStr)
	eh.InsertBytes(start, text)
}

// InsertBytes creates an insert text event and executes it
func (eh *EventHandler) InsertBytes(start Loc, text []byte) {
	eh.insertBytes(start, text, userEdit)
}

func (eh *EventHandler) insertBytes(start Loc, text []byte, permission textEventPermission) {
	if len(text) == 0 {
		return
	}
	start = clamp(start, eh.buf.LineArray)
	e := &TextEvent{
		C:         *eh.cursors[eh.active],
		EventType: TextEventInsert,
		Deltas:    []Delta{{text, start, Loc{0, 0}}},
		Time:      time.Now(),
	}
	eh.doTextEvent(e, true, permission)
}

// Remove creates a remove text event and executes it
func (eh *EventHandler) Remove(start, end Loc) {
	eh.remove(start, end, userEdit)
}

func (eh *EventHandler) remove(start, end Loc, permission textEventPermission) {
	if start == end {
		return
	}
	start = clamp(start, eh.buf.LineArray)
	end = clamp(end, eh.buf.LineArray)
	e := &TextEvent{
		C:         *eh.cursors[eh.active],
		EventType: TextEventRemove,
		Deltas:    []Delta{{[]byte{}, start, end}},
		Time:      time.Now(),
	}
	eh.doTextEvent(e, true, permission)
}

// MultipleReplace creates a replace text event with multiple deltas and executes it
func (eh *EventHandler) MultipleReplace(deltas []Delta) {
	e := &TextEvent{
		C:         *eh.cursors[eh.active],
		EventType: TextEventReplace,
		Deltas:    deltas,
		Time:      time.Now(),
	}
	eh.Execute(e)
}

// Replace deletes from start to end and replaces it with the given string
func (eh *EventHandler) Replace(start, end Loc, replace string) {
	eh.Remove(start, end)
	eh.Insert(start, replace)
}

// Execute a textevent and add it to the undo stack
func (eh *EventHandler) Execute(t *TextEvent) {
	eh.execute(t, userEdit)
}

func (eh *EventHandler) execute(t *TextEvent, permission textEventPermission) bool {
	if !eh.buf.canEdit(permission) {
		return false
	}

	b, err := config.RunPluginFnBool(nil, "onBeforeTextEvent", luar.New(ulua.L, eh.buf), luar.New(ulua.L, t))
	if err != nil {
		screen.TermMessage(err)
	}

	// A plugin may have made the buffer read-only while handling the event.
	if !b || !eh.buf.canEdit(permission) {
		return false
	}

	if eh.RedoStack.Len() > 0 {
		eh.RedoStack = new(TEStack)
	}
	eh.UndoStack.Push(t)
	executeTextEvent(t, eh.buf, permission)
	return true
}

// Undo the first event in the undo stack. Returns false if read-only or empty.
func (eh *EventHandler) Undo() bool {
	if !eh.buf.canEdit(userEdit) {
		return false
	}
	t := eh.UndoStack.Peek()
	if t == nil {
		return false
	}

	startTime := t.Time.UnixNano() / int64(time.Millisecond)
	endTime := startTime - (startTime % undoThreshold)

	for {
		t = eh.UndoStack.Peek()
		if t == nil {
			break
		}

		if t.Time.UnixNano()/int64(time.Millisecond) < endTime {
			break
		}

		eh.UndoOneEvent()
	}
	return true
}

// UndoOneEvent undoes one event
func (eh *EventHandler) UndoOneEvent() {
	if !eh.buf.canEdit(userEdit) {
		return
	}
	// This event should be undone
	// Pop it off the stack
	t := eh.UndoStack.Pop()
	if t == nil {
		return
	}
	// Undo it
	// Modifies the text event
	eh.UndoTextEvent(t)

	// Set the cursor in the right place
	if t.C.Num >= 0 && t.C.Num < len(eh.cursors) {
		eh.cursors[t.C.Num].Goto(t.C)
		eh.cursors[t.C.Num].NewTrailingWsY = t.C.NewTrailingWsY
	}

	// Push it to the redo stack
	eh.RedoStack.Push(t)
}

// Redo the first event in the redo stack. Returns false if read-only or empty.
func (eh *EventHandler) Redo() bool {
	if !eh.buf.canEdit(userEdit) {
		return false
	}
	t := eh.RedoStack.Peek()
	if t == nil {
		return false
	}

	startTime := t.Time.UnixNano() / int64(time.Millisecond)
	endTime := startTime - (startTime % undoThreshold) + undoThreshold

	for {
		t = eh.RedoStack.Peek()
		if t == nil {
			break
		}

		if t.Time.UnixNano()/int64(time.Millisecond) > endTime {
			break
		}

		eh.RedoOneEvent()
	}
	return true
}

// RedoOneEvent redoes one event
func (eh *EventHandler) RedoOneEvent() {
	if !eh.buf.canEdit(userEdit) {
		return
	}
	t := eh.RedoStack.Pop()
	if t == nil {
		return
	}

	if t.C.Num >= 0 && t.C.Num < len(eh.cursors) {
		eh.cursors[t.C.Num].Goto(t.C)
		eh.cursors[t.C.Num].NewTrailingWsY = t.C.NewTrailingWsY
	}

	// Modifies the text event
	eh.UndoTextEvent(t)

	eh.UndoStack.Push(t)
}

// updateTrailingWs updates the cursor's trailing whitespace status after a text event
func (eh *EventHandler) updateTrailingWs(t *TextEvent) {
	if len(t.Deltas) != 1 {
		return
	}
	text := t.Deltas[0].Text
	start := t.Deltas[0].Start
	end := t.Deltas[0].End

	c := eh.cursors[eh.active]
	isEol := func(loc Loc) bool {
		return loc.X == util.CharacterCount(eh.buf.LineBytes(loc.Y))
	}
	if t.EventType == TextEventInsert && c.Loc == end && isEol(end) {
		var addedTrailingWs bool
		addedAfterWs := false
		addedWsOnly := false
		if start.Y == end.Y {
			addedTrailingWs = util.HasTrailingWhitespace(text)
			addedWsOnly = util.IsBytesWhitespace(text)
			addedAfterWs = start.X > 0 && util.IsWhitespace(c.buf.RuneAt(Loc{start.X - 1, start.Y}))
		} else {
			lastnl := bytes.LastIndex(text, []byte{'\n'})
			addedTrailingWs = util.HasTrailingWhitespace(text[lastnl+1:])
		}

		if addedTrailingWs && !(addedAfterWs && addedWsOnly) {
			c.NewTrailingWsY = c.Y
		} else if !addedTrailingWs {
			c.NewTrailingWsY = -1
		}
	} else if t.EventType == TextEventRemove && c.Loc == start && isEol(start) {
		removedAfterWs := util.HasTrailingWhitespace(eh.buf.LineBytes(start.Y))
		var removedWsOnly bool
		if start.Y == end.Y {
			removedWsOnly = util.IsBytesWhitespace(text)
		} else {
			firstnl := bytes.Index(text, []byte{'\n'})
			removedWsOnly = util.IsBytesWhitespace(text[:firstnl])
		}

		if removedAfterWs && !removedWsOnly {
			c.NewTrailingWsY = c.Y
		} else if !removedAfterWs {
			c.NewTrailingWsY = -1
		}
	} else if c.NewTrailingWsY != -1 && start.Y != end.Y && c.Loc.GreaterThan(start) &&
		((t.EventType == TextEventInsert && c.Y == c.NewTrailingWsY+(end.Y-start.Y)) ||
			(t.EventType == TextEventRemove && c.Y == c.NewTrailingWsY-(end.Y-start.Y))) {
		// The cursor still has its new trailingws
		// but its line number was shifted by insert or remove of lines above
		c.NewTrailingWsY = c.Y
	}
}
