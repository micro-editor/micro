package buffer

import (
	"github.com/micro-editor/micro/v2/internal/util"
)

// Loc stores a location
type Loc struct {
	X, Y int
}

// LessThan returns true if l is smaller than b
func (l Loc) LessThan(b Loc) bool {
	if l.Y < b.Y {
		return true
	}
	return l.Y == b.Y && l.X < b.X
}

// GreaterThan returns true if l is bigger than b
func (l Loc) GreaterThan(b Loc) bool {
	if l.Y > b.Y {
		return true
	}
	return l.Y == b.Y && l.X > b.X
}

// GreaterEqual returns true if l is greater than or equal to b
func (l Loc) GreaterEqual(b Loc) bool {
	if l.Y > b.Y {
		return true
	}
	if l.Y == b.Y && l.X > b.X {
		return true
	}
	return l == b
}

// LessEqual returns true if l is less than or equal to b
func (l Loc) LessEqual(b Loc) bool {
	if l.Y < b.Y {
		return true
	}
	if l.Y == b.Y && l.X < b.X {
		return true
	}
	return l == b
}

// Clamp clamps a loc between start and end
func (l Loc) Clamp(start, end Loc) Loc {
	if l.GreaterEqual(end) {
		return end
	} else if l.LessThan(start) {
		return start
	}
	return l
}

// The following functions require a buffer to know where newlines are

// Diff returns the distance between two locations
func DiffLA(a, b Loc, buf *LineArray) int {
	if a.Y == b.Y {
		if a.X > b.X {
			return a.X - b.X
		}
		return b.X - a.X
	}

	// Make sure a is guaranteed to be less than b
	if b.LessThan(a) {
		a, b = b, a
	}

	loc := 0
	for i := a.Y + 1; i < b.Y; i++ {
		// + 1 for the newline
		loc += util.CharacterCount(buf.LineBytes(i)) + 1
	}
	loc += util.CharacterCount(buf.LineBytes(a.Y)) - a.X + b.X + 1
	return loc
}

// This moves the location one character to the right
func (l Loc) right(buf *LineArray) Loc {
	if l == buf.End() {
		return Loc{l.X + 1, l.Y}
	}
	var res Loc
	if l.X < util.CharacterCount(buf.LineBytes(l.Y)) {
		res = Loc{l.X + 1, l.Y}
	} else {
		res = Loc{0, l.Y + 1}
	}
	return res
}

// This moves the given location one character to the left
func (l Loc) left(buf *LineArray) Loc {
	if l == buf.Start() {
		return Loc{l.X - 1, l.Y}
	}
	var res Loc
	if l.X > 0 {
		res = Loc{l.X - 1, l.Y}
	} else {
		res = Loc{util.CharacterCount(buf.LineBytes(l.Y - 1)), l.Y - 1}
	}
	return res
}

// MoveLA moves the cursor n characters to the left or right
// It moves the cursor left if n is negative
func (l Loc) MoveLA(n int, buf *LineArray) Loc {
	for n > 0 {
		l = l.right(buf)
		n--
	}
	for n < 0 {
		l = l.left(buf)
		n++
	}
	return l
}

// Diff returns the difference between two locs
func (l Loc) Diff(b Loc, buf *Buffer) int {
	return DiffLA(l, b, buf.LineArray)
}

// Move moves a loc n characters
func (l Loc) Move(n int, buf *Buffer) Loc {
	return l.MoveLA(n, buf.LineArray)
}

// ByteOffset is just like ToCharPos except it counts bytes instead of runes
func ByteOffset(pos Loc, buf *Buffer) int {
	x, y := pos.X, pos.Y
	loc := 0
	for i := 0; i < y; i++ {
		// + 1 for the newline
		loc += len(buf.Line(i)) + 1
	}
	loc += len(buf.Line(y)[:x])
	return loc
}

// clamps a loc within a buffer
func clamp(pos Loc, la *LineArray) Loc {
	return pos.Clamp(la.Start(), la.End())
}
