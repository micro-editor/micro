package buffer

// Chunk detection for the hlchunk option, in the spirit of hlchunk.nvim:
// locate the block around the active cursor so the display layer can draw
// a guide along its edge. Two detectors, selected by the hlchunkmode
// setting:
//
//   - indent: a chunk is delimited by the nearest lines above and below
//     the cursor with smaller visual indent (the opening statement and
//     the closing token, or the next sibling statement); a cursor on a
//     block-opening header anchors the chunk that header opens instead.
//     Works for any language but trusts the indentation.
//   - bracket: a chunk is the innermost (), [] or {} pair spanning more
//     than one line around the cursor. Exact block extents for brace
//     languages (multi-line conditions, mixed indent), but counts
//     brackets inside strings and comments, like matchbrace does.
//
// Both fill the same draw-ready Chunk, so the display layer never knows
// which mode fired.

import "github.com/micro-editor/micro/v2/internal/util"

// detection runs on every redraw (every 16ms while animating), so
// boundary scans bail beyond this distance instead of walking a huge
// file end to end
const chunkScanLimit = 5000

// Chunk describes the block around the cursor and where its guide sits
type Chunk struct {
	Start, End             int // boundary line numbers (corner rows)
	StartIndent, EndIndent int // visual indent width of the boundary lines
	GuideCol               int // guide column, in visual columns
}

// IndentChunk locates the indent chunk around line cury. It reports
// false when the cursor is at top level or a boundary is missing, or
// when a boundary lies further than the scan limit.
func (b *Buffer) IndentChunk(cury int) (Chunk, bool) {
	tabsize := util.IntOpt(b.Settings["tabsize"])
	return findIndentChunk(b.LineBytes, b.LinesNum(), cury, tabsize)
}

// BraceChunk locates the innermost bracket pair around cur that spans
// more than one line. It reports false when no pair encloses the cursor
// within the scan limit.
func (b *Buffer) BraceChunk(cur Loc) (Chunk, bool) {
	var cg Chunk
	tabsize := util.IntOpt(b.Settings["tabsize"])
	ymin, _ := chunkScanBounds(cur.Y, b.LinesNum())

	cg.Start = -1
	// a line that leaves a bracket open anchors the chunk that bracket
	// opens, mirroring the indent mode's header rule
	line := []rune(string(b.LineBytes(cur.Y)))
	if x := lastOpenBrace(line); x >= 0 {
		pair, _ := braceDir(line[x])
		if cl, ok := b.findMatchingBrace(pair, Loc{x, cur.Y}, pair[0]); ok && cl.Y > cur.Y {
			cg.Start, cg.End = cur.Y, cl.Y
		}
	}
	// enclosing pairs living on a single line are not chunks: consume
	// them and keep scanning outward
	pos := cur
	for cg.Start < 0 {
		op, pair, ok := enclosingBrace(b.LineBytes, ymin, pos)
		if !ok {
			return cg, false
		}
		if cl, ok := b.findMatchingBrace(pair, op, pair[0]); ok && cl.Y > op.Y {
			cg.Start, cg.End = op.Y, cl.Y
			break
		}
		pos = op
	}

	cg.StartIndent, _ = visualIndent(b.LineBytes(cg.Start), tabsize)
	cg.EndIndent, _ = visualIndent(b.LineBytes(cg.End), tabsize)
	cg.finalize(b.LineBytes, cur.Y, tabsize)
	return cg, true
}

// visualIndent returns the display width of the line's leading whitespace
// and whether the line contains nothing else.
func visualIndent(line []byte, tabsize int) (int, bool) {
	w := 0
	for _, c := range line {
		switch c {
		case ' ':
			w++
		case '\t':
			w += tabsize - (w % tabsize)
		default:
			return w, false
		}
	}
	return w, true
}

// chunkScanBounds clamps the boundary scan window around line y
func chunkScanBounds(y, nlines int) (int, int) {
	ymin := y - chunkScanLimit
	if ymin < 0 {
		ymin = 0
	}
	ymax := y + chunkScanLimit
	if ymax > nlines-1 {
		ymax = nlines - 1
	}
	return ymin, ymax
}

// finalize turns raw boundaries into a drawable guide: place the guide
// column one indent level left of the boundary indent. Column-zero
// boundary lines have no whitespace to hold their corner; the display
// layer skips such corner rows and the bars just span the body.
func (cg *Chunk) finalize(getLine func(int) []byte, cury, tabsize int) {
	cg.GuideCol = cg.StartIndent
	if cg.EndIndent < cg.GuideCol {
		cg.GuideCol = cg.EndIndent
	}
	cg.GuideCol -= tabsize
	if cg.GuideCol < 0 {
		cg.GuideCol = 0
	}

	// a column-zero opener likewise has no top corner: keep bars off
	// blank lines at the chunk's head (never past the cursor's line)
	if cg.StartIndent == 0 {
		for cg.Start+1 < cury {
			if _, b := visualIndent(getLine(cg.Start+1), tabsize); !b {
				break
			}
			cg.Start++
		}
	}
}

// findIndentChunk locates the indent chunk around line cury: the lines
// between the nearest lines above and below it with smaller visual
// indent.
func findIndentChunk(getLine func(int) []byte, nlines, cury, tabsize int) (Chunk, bool) {
	var cg Chunk
	ymin, ymax := chunkScanBounds(cury, nlines)

	curIndent, blank := visualIndent(getLine(cury), tabsize)
	if blank {
		return cg, false
	}
	// a line opening a deeper block anchors the chunk it opens, not
	// the block enclosing it: the header is the top corner row and the
	// chunk runs to the first line back at the header's indent or less
	header := false
	for y := cury + 1; y <= ymax; y++ {
		if w, b := visualIndent(getLine(y), tabsize); !b {
			header = w > curIndent
			break
		}
	}
	if curIndent == 0 && !header {
		return cg, false
	}

	cg.Start = -1
	cg.End = -1
	if header {
		cg.Start, cg.StartIndent = cury, curIndent
		for y := cury + 1; y <= ymax; y++ {
			if w, b := visualIndent(getLine(y), tabsize); !b && w <= curIndent {
				cg.End, cg.EndIndent = y, w
				break
			}
		}
	} else {
		for y := cury - 1; y >= ymin; y-- {
			if w, b := visualIndent(getLine(y), tabsize); !b && w < curIndent {
				cg.Start, cg.StartIndent = y, w
				break
			}
		}
		for y := cury + 1; y <= ymax; y++ {
			if w, b := visualIndent(getLine(y), tabsize); !b && w < curIndent {
				cg.End, cg.EndIndent = y, w
				break
			}
		}
	}
	if cg.Start < 0 || cg.End < 0 {
		return cg, false
	}

	// the lower boundary is the chunk's own last line only when it is
	// a closing token; a sibling statement (python's except:, the next
	// def) is not part of the chunk, so anchor the corner on the
	// chunk's last code line instead of dangling the bars over it.
	// col-0 closers retarget too: no whitespace there to hold a corner
	// (unlike bracket mode, where the closer stays the boundary)
	if cg.EndIndent == 0 || !closesChunk(getLine(cg.End)) {
		for y := cg.End - 1; y > cg.Start; y-- {
			if w, b := visualIndent(getLine(y), tabsize); !b {
				cg.End, cg.EndIndent = y, w
				break
			}
		}
	}

	cg.finalize(getLine, cury, tabsize)
	return cg, true
}

// closesChunk reports whether a boundary line is a block's own closing
// token (first code rune closes a bracket) rather than a sibling
// statement.
func closesChunk(line []byte) bool {
	for _, r := range string(line) {
		if r == ' ' || r == '\t' {
			continue
		}
		_, dir := braceDir(r)
		return dir < 0
	}
	return false
}

// braceDir reports which pair a bracket rune belongs to; dir is +1 for
// an opener, -1 for a closer, 0 for any other rune.
func braceDir(r rune) (pair [2]rune, dir int) {
	for _, bp := range BracePairs {
		if r == bp[0] {
			return bp, 1
		}
		if r == bp[1] {
			return bp, -1
		}
	}
	return pair, 0
}

// lastOpenBrace returns the position of the innermost bracket the line
// opens without closing, or -1. Pairing is type-blind: valid code nests
// brackets properly, and code that does not (brackets in strings)
// miscounts the same way matchbrace does.
func lastOpenBrace(line []rune) int {
	var open []int
	for x, r := range line {
		if _, dir := braceDir(r); dir > 0 {
			open = append(open, x)
		} else if dir < 0 && len(open) > 0 {
			open = open[:len(open)-1]
		}
	}
	if len(open) == 0 {
		return -1
	}
	return open[len(open)-1]
}

// enclosingBrace scans backwards from cur (exclusive) for the nearest
// bracket left open at the cursor, scanning no further than line ymin.
// Type-blind pairing, as in lastOpenBrace.
func enclosingBrace(getLine func(int) []byte, ymin int, cur Loc) (Loc, [2]rune, bool) {
	depth := 0
	for y := cur.Y; y >= ymin; y-- {
		l := []rune(string(getLine(y)))
		x0 := len(l) - 1
		if y == cur.Y && cur.X-1 < x0 {
			x0 = cur.X - 1
		}
		for x := x0; x >= 0; x-- {
			pair, dir := braceDir(l[x])
			if dir < 0 {
				depth++
			} else if dir > 0 {
				if depth == 0 {
					return Loc{x, y}, pair, true
				}
				depth--
			}
		}
	}
	return cur, [2]rune{}, false
}
