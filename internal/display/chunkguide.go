package display

// The hlchunk option draws a guide around the chunk containing the
// active cursor, in the spirit of hlchunk.nvim. Detection lives in
// internal/buffer (chunk.go); this file owns guide geometry and
// animation. The guide is drawn one indent level left of the boundary
// indent:
//
//	if x {     <- start row:   ╭──
//	    a()    <- middle row:  │
//	}          <- end row:     ╰─>
//
// Guide runes only ever replace whitespace cells, so text is never covered.
//
// A fresh chunk is animated: its cells appear one by one over
// chunkAnimDuration, sweeping from the opening line's text around to the
// closing line's text, like the original plugin.

import (
	"time"

	"github.com/micro-editor/micro/v2/internal/buffer"
)

const (
	chunkCornerTop  = '╭'
	chunkCornerBot  = '╰'
	chunkVertical   = '│'
	chunkHorizontal = '─'
	chunkArrow      = '>'

	chunkAnimDuration = 200 * time.Millisecond
	chunkAnimFrame    = 16 * time.Millisecond

	// a new chunk stays hidden until the cursor settles on it, so
	// holding up/down does not fire a sweep at every step
	chunkSettleDelay = 200 * time.Millisecond
)

// chunkGuide adds guide geometry to a detected chunk
type chunkGuide struct {
	buffer.Chunk
}

// chunkAnim animates a guide by revealing its cells over chunkAnimDuration
type chunkAnim struct {
	shown chunkGuide
	start time.Time
}

// visible returns how many guide cells are revealed right now, and whether
// more frames are needed. A changed chunk restarts the clock, set in the
// future so the sweep only begins once the cursor has settled.
func (a *chunkAnim) visible(cg chunkGuide) (int, bool) {
	if cg != a.shown {
		a.shown = cg
		a.start = time.Now().Add(chunkSettleDelay)
	}
	elapsed := time.Since(a.start)
	if elapsed < 0 {
		return 0, true
	}
	total := cg.cells()
	if elapsed >= chunkAnimDuration {
		return total, false
	}
	return int(elapsed * time.Duration(total) / chunkAnimDuration), true
}

// cells returns the number of cells the guide occupies (animation steps)
func (cg *chunkGuide) cells() int {
	n := cg.End - cg.Start - 1
	if cg.StartIndent > cg.GuideCol {
		n += cg.StartIndent - cg.GuideCol
	}
	if cg.EndIndent > cg.GuideCol {
		n += cg.EndIndent - cg.GuideCol
	}
	return n
}

// cellIndex returns the draw-order position of a cell the guide covers:
// the top corner row fills leftwards from the opening line's text, the
// bars run downwards, the bottom corner rightwards to the arrow.
func (cg *chunkGuide) cellIndex(y, vcol int) int {
	topLen := 0
	if cg.StartIndent > cg.GuideCol {
		topLen = cg.StartIndent - cg.GuideCol
	}
	switch {
	case y == cg.Start:
		return cg.StartIndent - 1 - vcol
	case y < cg.End:
		return topLen + y - cg.Start - 1
	default:
		return topLen + cg.End - cg.Start - 1 + vcol - cg.GuideCol
	}
}

// runeAt returns the guide rune for visual column vcol of line y, or 0 if
// the guide does not cover that cell. Corner rows are skipped when their
// boundary line has no leading whitespace to draw into.
func (cg *chunkGuide) runeAt(y, vcol int) rune {
	switch {
	case y > cg.Start && y < cg.End:
		if vcol == cg.GuideCol {
			return chunkVertical
		}
	case y == cg.Start && cg.StartIndent > cg.GuideCol:
		switch {
		case vcol == cg.GuideCol:
			return chunkCornerTop
		case vcol > cg.GuideCol && vcol < cg.StartIndent:
			return chunkHorizontal
		}
	case y == cg.End && cg.EndIndent > cg.GuideCol:
		switch {
		case vcol == cg.GuideCol:
			return chunkCornerBot
		case vcol == cg.EndIndent-1:
			return chunkArrow
		case vcol > cg.GuideCol && vcol < cg.EndIndent:
			return chunkHorizontal
		}
	}
	return 0
}
