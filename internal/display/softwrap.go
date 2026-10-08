package display

import (
	"bytes"

	runewidth "github.com/mattn/go-runewidth"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/util"
)

// SLoc represents a vertical scrolling location, i.e. a location of a visual line
// in the buffer. When softwrap is enabled, a buffer line may be displayed as
// multiple visual lines (rows). So SLoc stores a number of a line in the buffer
// and a number of a row within this line.
type SLoc struct {
	Line, Row int
}

// LessThan returns true if s is less b
func (s SLoc) LessThan(b SLoc) bool {
	if s.Line < b.Line {
		return true
	}
	return s.Line == b.Line && s.Row < b.Row
}

// GreaterThan returns true if s is bigger than b
func (s SLoc) GreaterThan(b SLoc) bool {
	if s.Line > b.Line {
		return true
	}
	return s.Line == b.Line && s.Row > b.Row
}

// LessEqual returns true if s is less than or equal to b
func (s SLoc) LessEqual(b SLoc) bool {
	if s.Line < b.Line {
		return true
	}
	if s.Line == b.Line && s.Row < b.Row {
		return true
	}
	return s == b
}

// GreaterEqual returns true if s is bigger than or equal to b
func (s SLoc) GreaterEqual(b SLoc) bool {
	if s.Line > b.Line {
		return true
	}
	if s.Line == b.Line && s.Row > b.Row {
		return true
	}
	return s == b
}

// VLoc represents a location in the buffer as a visual location in the
// linewrapped buffer.
type VLoc struct {
	SLoc
	VisualX int
}

type SoftWrap interface {
	Scroll(s SLoc, n int) SLoc
	Diff(s1, s2 SLoc) int
	SLocFromLoc(loc buffer.Loc) SLoc
	VLocFromLoc(loc buffer.Loc) VLoc
	LocFromVLoc(vloc VLoc) buffer.Loc
}

func (w *BufWindow) getVLocFromLoc(loc buffer.Loc) VLoc {
	vloc := VLoc{SLoc: SLoc{loc.Y, 0}, VisualX: 0}

	if loc.X <= 0 {
		return vloc
	}

	if w.bufWidth <= 0 {
		return vloc
	}

	wordwrap := w.Buf.Settings["wordwrap"].(bool)
	tabsize := util.IntOpt(w.Buf.Settings["tabsize"])
	leadingvisualspaces := w.getLeadingVisualSpaces(vloc)

	line := w.Buf.LineBytes(loc.Y)
	x := 0
	totalwidth := 0

	wordwidth := 0
	wordoffset := 0

	for len(line) > 0 {
		r, _, size := util.DecodeCharacter(line)
		line = line[size:]

		width := 0
		switch r {
		case '\t':
			ts := tabsize - (totalwidth % tabsize)
			width = util.Min(ts, w.bufWidth-vloc.VisualX)
			totalwidth += ts
		default:
			width = runewidth.RuneWidth(r)
			totalwidth += width
		}

		wordwidth += width

		// Collect a complete word to know its width.
		// If wordwrap is off, every single character is a complete "word".
		if wordwrap {
			if !util.IsWhitespace(r) && len(line) > 0 && wordwidth < w.bufWidth {
				if x < loc.X {
					wordoffset += width
					x++
				}
				continue
			}
		}

		// If a word (or just a wide rune) does not fit in the window
		if vloc.VisualX+wordwidth > w.bufWidth && vloc.VisualX > 0 {
			vloc.Row++
			vloc.VisualX = 0
			vloc.VisualX += leadingvisualspaces
		}

		if x == loc.X {
			vloc.VisualX += wordoffset
			return vloc
		}
		x++

		vloc.VisualX += wordwidth

		wordwidth = 0
		wordoffset = 0

		if vloc.VisualX >= w.bufWidth {
			vloc.Row++
			vloc.VisualX = 0
			vloc.VisualX += leadingvisualspaces
		}
	}
	return vloc
}

func (w *BufWindow) getLocFromVLoc(svloc VLoc) buffer.Loc {
	loc := buffer.Loc{X: 0, Y: svloc.Line}

	if w.bufWidth <= 0 {
		return loc
	}

	wordwrap := w.Buf.Settings["wordwrap"].(bool)
	tabsize := util.IntOpt(w.Buf.Settings["tabsize"])
	leadingvisualspaces := w.getLeadingVisualSpaces(svloc)

	line := w.Buf.LineBytes(svloc.Line)
	vloc := VLoc{SLoc: SLoc{svloc.Line, 0}, VisualX: 0}

	totalwidth := 0

	var widths []int
	if wordwrap {
		widths = make([]int, 0, w.bufWidth)
	} else {
		widths = make([]int, 0, 1)
	}
	wordwidth := 0

	for len(line) > 0 {
		r, _, size := util.DecodeCharacter(line)
		line = line[size:]

		width := 0
		switch r {
		case '\t':
			ts := tabsize - (totalwidth % tabsize)
			width = util.Min(ts, w.bufWidth-vloc.VisualX)
			totalwidth += ts
		default:
			width = runewidth.RuneWidth(r)
			totalwidth += width
		}

		widths = append(widths, width)
		wordwidth += width

		// Collect a complete word to know its width.
		// If wordwrap is off, every single character is a complete "word".
		if wordwrap {
			if !util.IsWhitespace(r) && len(line) > 0 && wordwidth < w.bufWidth {
				continue
			}
		}

		// If a word (or just a wide rune) does not fit in the window
		if vloc.VisualX+wordwidth > w.bufWidth && vloc.VisualX > 0 {
			if vloc.Row == svloc.Row {
				if wordwrap {
					// it's a word, not a wide rune
					loc.X--
				}
				return loc
			}
			vloc.Row++
			vloc.VisualX = 0
			vloc.VisualX += leadingvisualspaces
		}

		for i := range widths {
			vloc.VisualX += widths[i]
			if vloc.Row == svloc.Row && vloc.VisualX > svloc.VisualX {
				return loc
			}
			loc.X++
		}

		widths = widths[:0]
		wordwidth = 0

		if vloc.VisualX >= w.bufWidth {
			vloc.Row++
			vloc.VisualX = 0
			vloc.VisualX += leadingvisualspaces
		}
	}
	return loc
}

// For wrapindent, count leading space to set correct VisualX on up, down action
func (w *BufWindow) getLeadingVisualSpaces(vloc VLoc) int {
	wrapindent := util.IntOpt(w.Buf.Settings["wrapindent"])
	if wrapindent < 0 {
		return 0
	}
	line := w.Buf.LineBytes(vloc.Line)
	leadingwhitespace := util.GetLeadingWhitespace(line)
	leadingtabscount := bytes.Count(leadingwhitespace, []byte{'\t'})
	return wrapindent + leadingtabscount*int(w.Buf.Settings["tabsize"].(float64)) + (len(leadingwhitespace) - leadingtabscount)
}

func (w *BufWindow) getRowCount(line int) int {
	eol := buffer.Loc{X: util.CharacterCount(w.Buf.LineBytes(line)), Y: line}
	return w.getVLocFromLoc(eol).Row + 1
}

func (w *BufWindow) scrollUp(s SLoc, n int) SLoc {
	for n > 0 {
		if n <= s.Row {
			s.Row -= n
			n = 0
		} else if s.Line > 0 {
			s.Line--
			n -= s.Row + 1
			s.Row = w.getRowCount(s.Line) - 1
		} else {
			s.Row = 0
			break
		}
	}
	return s
}

func (w *BufWindow) scrollDown(s SLoc, n int) SLoc {
	for n > 0 {
		rc := w.getRowCount(s.Line)
		if n < rc-s.Row {
			s.Row += n
			n = 0
		} else if s.Line < w.Buf.LinesNum()-1 {
			s.Line++
			n -= rc - s.Row
			s.Row = 0
		} else {
			s.Row = rc - 1
			break
		}
	}
	return s
}

func (w *BufWindow) scroll(s SLoc, n int) SLoc {
	if n < 0 {
		return w.scrollUp(s, -n)
	}
	return w.scrollDown(s, n)
}

func (w *BufWindow) diff(s1, s2 SLoc) int {
	n := 0
	for s1.LessThan(s2) {
		if s1.Line < s2.Line {
			n += w.getRowCount(s1.Line) - s1.Row
			s1.Line++
			s1.Row = 0
		} else {
			n += s2.Row - s1.Row
			s1.Row = s2.Row
		}
	}
	return n
}

// Scroll returns the location which is n visual lines below the location s
// i.e. the result of scrolling n lines down. n can be negative,
// which means scrolling up. The returned location is guaranteed to be
// within the buffer boundaries.
func (w *BufWindow) Scroll(s SLoc, n int) SLoc {
	if !w.Buf.Settings["softwrap"].(bool) {
		s.Line = util.Clamp(s.Line+n, 0, w.Buf.LinesNum()-1)
		return s
	}
	return w.scroll(s, n)
}

// Diff returns the difference (the vertical distance) between two SLocs.
func (w *BufWindow) Diff(s1, s2 SLoc) int {
	if !w.Buf.Settings["softwrap"].(bool) {
		return s2.Line - s1.Line
	}
	if s1.GreaterThan(s2) {
		return -w.diff(s2, s1)
	}
	return w.diff(s1, s2)
}

// SLocFromLoc takes a position in the buffer and returns the location
// of the visual line containing this position.
func (w *BufWindow) SLocFromLoc(loc buffer.Loc) SLoc {
	if !w.Buf.Settings["softwrap"].(bool) {
		return SLoc{loc.Y, 0}
	}
	return w.getVLocFromLoc(loc).SLoc
}

// VLocFromLoc takes a position in the buffer and returns the corresponding
// visual location in the linewrapped buffer.
func (w *BufWindow) VLocFromLoc(loc buffer.Loc) VLoc {
	if !w.Buf.Settings["softwrap"].(bool) {
		tabsize := util.IntOpt(w.Buf.Settings["tabsize"])

		visualx := util.StringWidth(w.Buf.LineBytes(loc.Y), loc.X, tabsize)
		return VLoc{SLoc{loc.Y, 0}, visualx}
	}
	return w.getVLocFromLoc(loc)
}

// LocFromVLoc takes a visual location in the linewrapped buffer and returns
// the position in the buffer corresponding to this visual location.
func (w *BufWindow) LocFromVLoc(vloc VLoc) buffer.Loc {
	if !w.Buf.Settings["softwrap"].(bool) {
		tabsize := util.IntOpt(w.Buf.Settings["tabsize"])

		x := util.GetCharPosInLine(w.Buf.LineBytes(vloc.Line), vloc.VisualX, tabsize)
		return buffer.Loc{x, vloc.Line}
	}
	return w.getLocFromVLoc(vloc)
}
