package buffer

import (
	"bufio"
	"bytes"
	"io"
	"sync"

	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/micro-editor/micro/v2/pkg/highlight"
)

// A searchState contains the search match info for a single line
type searchState struct {
	search     string
	useRegex   bool
	ignorecase bool
	match      [][2]int
	done       bool
}

type Character struct {
	combc []rune
}

// A Line contains the slice of runes as well as a highlight state, match
// and a flag for whether the highlighting needs to be updated
type Line struct {
	runes []Character

	state highlight.State
	match highlight.LineMatch
	lock  sync.Mutex

	// The search states for the line, used for highlighting of search matches,
	// separately from the syntax highlighting.
	// A map is used because the line array may be shared between multiple buffers
	// (multiple instances of the same file opened in different edit panes)
	// which have distinct searches, so in the general case there are multiple
	// searches per a line, one search per a Buffer containing this line.
	search map[*Buffer]*searchState
}

// data returns the line as byte slice
func (l Line) data() []byte {
	var runes []rune
	for _, r := range l.runes {
		runes = append(runes, r.combc[0:]...)
	}
	return []byte(string(runes))
}

// String returns the line as string
func (l Line) String() string {
	var runes []rune
	for _, r := range l.runes {
		runes = append(runes, r.combc[0:]...)
	}
	return string(runes)
}

const (
	// Line ending file formats
	FFAuto = 0 // Autodetect format
	FFUnix = 1 // LF line endings (unix style '\n')
	FFDos  = 2 // CRLF line endings (dos style '\r\n')
)

type FileFormat byte

// A LineArray simply stores and array of lines and makes it easy to insert
// and delete in it
type LineArray struct {
	lines    []Line
	Endings  FileFormat
	initsize uint64
	lock     sync.Mutex
}

// Append efficiently appends lines together
// It allocates an additional 10000 lines if the original estimate
// is incorrect
func Append(slice []Line, data ...Line) []Line {
	l := len(slice)
	if l+len(data) > cap(slice) { // reallocate
		newSlice := make([]Line, (l+len(data))+10000)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0 : l+len(data)]
	for i, c := range data {
		slice[l+i] = c
	}
	return slice
}

// NewLineArray returns a new line array from an array of runes
func NewLineArray(size uint64, endings FileFormat, reader io.Reader) *LineArray {
	la := new(LineArray)

	la.lines = make([]Line, 0, 1000)
	la.initsize = size

	br := bufio.NewReader(reader)
	var loaded int

	la.Endings = endings

	n := 0
	for {
		data, err := br.ReadBytes('\n')
		// Detect the line ending by checking to see if there is a '\r' char
		// before the '\n'
		// Even if the file format is set to DOS, the '\r' is removed so
		// that all lines end with '\n'
		dlen := len(data)
		if dlen > 1 && data[dlen-2] == '\r' {
			data = append(data[:dlen-2], '\n')
			if la.Endings == FFAuto {
				la.Endings = FFDos
			}
			dlen = len(data)
		} else if dlen > 0 {
			if la.Endings == FFAuto {
				la.Endings = FFUnix
			}
		}

		// If we are loading a large file (greater than 1000) we use the file
		// size and the length of the first 1000 lines to try to estimate
		// how many lines will need to be allocated for the rest of the file
		// We add an extra 10000 to the original estimate to be safe and give
		// plenty of room for expansion
		if n >= 1000 && loaded >= 0 {
			totalLinesNum := int(float64(size) * (float64(n) / float64(loaded)))
			newSlice := make([]Line, len(la.lines), totalLinesNum+10000)
			copy(newSlice, la.lines)
			la.lines = newSlice
			loaded = -1
		}

		// Counter for the number of bytes in the first 1000 lines
		if loaded >= 0 {
			loaded += dlen
		}

		var runes []Character
		if err != nil {
			if err == io.EOF {
				for len(data) > 0 {
					combc, s := util.DecodeCombinedCharacter(data)
					runes = append(runes, Character{combc})
					data = data[s:]
				}
				la.lines = Append(la.lines, Line{
					runes: runes,
					state: nil,
					match: nil,
				})
			}
			// Last line was read
			break
		} else {
			data = data[:dlen-1]
			for len(data) > 0 {
				combc, s := util.DecodeCombinedCharacter(data)
				runes = append(runes, Character{combc})
				data = data[s:]
			}
			la.lines = Append(la.lines, Line{
				runes: runes,
				state: nil,
				match: nil,
			})
		}
		n++
	}

	return la
}

// Bytes returns the string that should be written to disk when
// the line array is saved
func (la *LineArray) Bytes() []byte {
	b := new(bytes.Buffer)
	// initsize should provide a good estimate
	b.Grow(int(la.initsize + 4096))
	for i, l := range la.lines {
		b.Write(l.data())
		if i != len(la.lines)-1 {
			if la.Endings == FFDos {
				b.WriteByte('\r')
			}
			b.WriteByte('\n')
		}
	}
	return b.Bytes()
}

// newlineBelow adds a newline below the given line number
func (la *LineArray) newlineBelow(y int) {
	la.lines = append(la.lines, Line{
		runes: []Character{},
		state: nil,
		match: nil,
	})
	copy(la.lines[y+2:], la.lines[y+1:])
	la.lines[y+1] = Line{
		runes: []Character{},
		state: la.lines[y].state,
		match: nil,
	}
}

// Inserts a byte array at a given location
func (la *LineArray) insert(pos Loc, value []byte) {
	la.lock.Lock()
	defer la.lock.Unlock()

	var runes []Character
	for len(value) > 0 {
		combc, s := util.DecodeCombinedCharacter(value)
		runes = append(runes, Character{combc})
		value = value[s:]
	}
	x, y := util.Min(pos.X, len(la.lines[pos.Y].runes)), pos.Y
	start := -1

outer:
	for i, r := range runes {
		for j := 0; j < len(r.combc); j++ {
			if r.combc[j] == '\n' || (r.combc[j] == '\r' && i < len(runes)-1 && r.combc[j+1] == '\n') {
				la.split(Loc{x, y})
				if i > 0 && start < len(runes) && start < i {
					if start < 0 {
						start = 0
					}
					la.insertRunes(Loc{x, y}, runes[start:i])
				}

				x = 0
				y++

				if r.combc[j] == '\r' {
					i++
				}
				if i+1 <= len(runes) {
					start = i + 1
				}

				continue outer
			}
		}
	}
	if start < 0 {
		la.insertRunes(Loc{x, y}, runes)
	} else if start < len(runes) {
		la.insertRunes(Loc{x, y}, runes[start:])
	}
}

// Inserts a rune array at a given location
func (la *LineArray) insertRunes(pos Loc, runes []Character) {
	la.lines[pos.Y].runes = append(la.lines[pos.Y].runes, runes...)
	copy(la.lines[pos.Y].runes[pos.X+len(runes):], la.lines[pos.Y].runes[pos.X:])
	copy(la.lines[pos.Y].runes[pos.X:], runes)
}

// joinLines joins the two lines a and b
func (la *LineArray) joinLines(a, b int) {
	la.insertRunes(Loc{len(la.lines[a].runes), a}, la.lines[b].runes)
	la.deleteLine(b)
}

// split splits a line at a given position
func (la *LineArray) split(pos Loc) {
	la.newlineBelow(pos.Y)
	la.insertRunes(Loc{0, pos.Y + 1}, la.lines[pos.Y].runes[pos.X:])
	la.lines[pos.Y+1].state = la.lines[pos.Y].state
	la.lines[pos.Y].state = nil
	la.lines[pos.Y].match = nil
	la.lines[pos.Y+1].match = nil
	la.deleteToEnd(Loc{pos.X, pos.Y})
}

// removes from start to end
func (la *LineArray) remove(start, end Loc) []byte {
	la.lock.Lock()
	defer la.lock.Unlock()

	sub := la.Substr(start, end)
	startX := util.Min(start.X, len(la.lines[start.Y].runes))
	endX := util.Min(end.X, len(la.lines[end.Y].runes))
	if start.Y == end.Y {
		la.lines[start.Y].runes = append(la.lines[start.Y].runes[:startX], la.lines[start.Y].runes[endX:]...)
	} else {
		la.deleteLines(start.Y+1, end.Y-1)
		la.deleteToEnd(Loc{startX, start.Y})
		la.deleteFromStart(Loc{endX - 1, start.Y + 1})
		la.joinLines(start.Y, start.Y+1)
	}
	return sub
}

// deleteToEnd deletes from the end of a line to the position
func (la *LineArray) deleteToEnd(pos Loc) {
	la.lines[pos.Y].runes = la.lines[pos.Y].runes[:pos.X]
}

// deleteFromStart deletes from the start of a line to the position
func (la *LineArray) deleteFromStart(pos Loc) {
	la.lines[pos.Y].runes = la.lines[pos.Y].runes[pos.X+1:]
}

// deleteLine deletes the line number
func (la *LineArray) deleteLine(y int) {
	la.lines = la.lines[:y+copy(la.lines[y:], la.lines[y+1:])]
}

func (la *LineArray) deleteLines(y1, y2 int) {
	la.lines = la.lines[:y1+copy(la.lines[y1:], la.lines[y2+1:])]
}

// Substr returns the string representation between two locations
func (la *LineArray) Substr(start, end Loc) []byte {
	startX := util.Min(start.X, len(la.lines[start.Y].runes))
	endX := util.Min(end.X, len(la.lines[end.Y].runes))
	var runes []rune
	if start.Y == end.Y && startX <= endX {
		for _, r := range la.lines[start.Y].runes[startX:endX] {
			runes = append(runes, r.combc[0:]...)
		}
		return []byte(string(runes))
	}

	var str []byte
	for _, r := range la.lines[start.Y].runes[startX:] {
		runes = append(runes, r.combc[0:]...)
	}
	str = append(str, []byte(string(runes))...)
	str = append(str, '\n')
	for i := start.Y + 1; i <= end.Y-1; i++ {
		runes = runes[:0]
		for _, r := range la.lines[i].runes {
			runes = append(runes, r.combc[0:]...)
		}
		str = append(str, []byte(string(runes))...)
		str = append(str, '\n')
	}
	runes = runes[:0]
	for _, r := range la.lines[end.Y].runes[:endX] {
		runes = append(runes, r.combc[0:]...)
	}
	str = append(str, []byte(string(runes))...)
	return str
}

// LinesNum returns the number of lines in the buffer
func (la *LineArray) LinesNum() int {
	return len(la.lines)
}

// Start returns the start of the buffer
func (la *LineArray) Start() Loc {
	return Loc{0, 0}
}

// End returns the location of the last character in the buffer
func (la *LineArray) End() Loc {
	numlines := len(la.lines)
	return Loc{len(la.lines[numlines-1].runes), numlines - 1}
}

// LineCharacters returns line n as an array of characters
func (la *LineArray) LineCharacters(n int) []Character {
	if n >= len(la.lines) || n < 0 {
		return []Character{}
	}

	return la.lines[n].runes
}

// LineBytes returns line n as an array of bytes
func (la *LineArray) LineBytes(n int) []byte {
	if n >= len(la.lines) || n < 0 {
		return []byte{}
	}
	return la.lines[n].data()
}

// LineString returns line n as an string
func (la *LineArray) LineString(n int) string {
	if n >= len(la.lines) || n < 0 {
		return string("")
	}

	var runes []rune
	for _, r := range la.lines[n].runes {
		runes = append(runes, r.combc[0:]...)
	}

	return string(runes)
}

// State gets the highlight state for the given line number
func (la *LineArray) State(lineN int) highlight.State {
	la.lines[lineN].lock.Lock()
	defer la.lines[lineN].lock.Unlock()
	return la.lines[lineN].state
}

// SetState sets the highlight state at the given line number
func (la *LineArray) SetState(lineN int, s highlight.State) {
	la.lines[lineN].lock.Lock()
	defer la.lines[lineN].lock.Unlock()
	la.lines[lineN].state = s
}

// SetMatch sets the match at the given line number
func (la *LineArray) SetMatch(lineN int, m highlight.LineMatch) {
	la.lines[lineN].lock.Lock()
	defer la.lines[lineN].lock.Unlock()
	la.lines[lineN].match = m
}

// Match retrieves the match for the given line number
func (la *LineArray) Match(lineN int) highlight.LineMatch {
	la.lines[lineN].lock.Lock()
	defer la.lines[lineN].lock.Unlock()
	return la.lines[lineN].match
}

// Locks the whole LineArray
func (la *LineArray) Lock() {
	la.lock.Lock()
}

// Unlocks the whole LineArray
func (la *LineArray) Unlock() {
	la.lock.Unlock()
}

// SearchMatch returns true if the location `pos` is within a match
// of the last search for the buffer `b`.
// It is used for efficient highlighting of search matches (separately
// from the syntax highlighting).
// SearchMatch searches for the matches if it is called first time
// for the given line or if the line was modified. Otherwise the
// previously found matches are used.
//
// The buffer `b` needs to be passed because the line array may be shared
// between multiple buffers (multiple instances of the same file opened
// in different edit panes) which have distinct searches, so SearchMatch
// needs to know which search to match against.
func (la *LineArray) SearchMatch(b *Buffer, pos Loc) bool {
	if b.LastSearch == "" {
		return false
	}

	lineN := pos.Y
	if la.lines[lineN].search == nil {
		la.lines[lineN].search = make(map[*Buffer]*searchState)
	}
	s, ok := la.lines[lineN].search[b]
	if !ok {
		// Note: here is a small harmless leak: when the buffer `b` is closed,
		// `s` is not deleted from the map. It means that the buffer
		// will not be garbage-collected until the line array is garbage-collected,
		// i.e. until all the buffers sharing this file are closed.
		s = new(searchState)
		la.lines[lineN].search[b] = s
	}
	if !ok || s.search != b.LastSearch || s.useRegex != b.LastSearchRegex ||
		s.ignorecase != b.Settings["ignorecase"].(bool) {
		s.search = b.LastSearch
		s.useRegex = b.LastSearchRegex
		s.ignorecase = b.Settings["ignorecase"].(bool)
		s.done = false
	}

	if !s.done {
		s.match = nil
		start := Loc{0, lineN}
		end := Loc{len(la.lines[lineN].runes), lineN}
		for start.X < end.X {
			m, found, _ := b.FindNext(b.LastSearch, start, end, start, true, b.LastSearchRegex)
			if !found {
				break
			}
			s.match = append(s.match, [2]int{m[0].X, m[1].X})

			start.X = m[1].X
			if m[1].X == m[0].X {
				start.X = m[1].X + 1
			}
		}

		s.done = true
	}

	for _, m := range s.match {
		if pos.X >= m[0] && pos.X < m[1] {
			return true
		}
	}
	return false
}

// invalidateSearchMatches marks search matches for the given line as outdated.
// It is called when the line is modified.
func (la *LineArray) invalidateSearchMatches(lineN int) {
	if la.lines[lineN].search != nil {
		for _, s := range la.lines[lineN].search {
			s.done = false
		}
	}
}
