package display

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
)

func TestChunkGeometry(t *testing.T) {
	// tabsize 4, space indent:
	//	def f():
	//	    if x:      <- start, indent 4
	//	        a()
	//	    return     <- end, indent 4
	cg := chunkGuide{buffer.Chunk{
		Start: 1, End: 3, StartIndent: 4, EndIndent: 4, GuideCol: 0,
	}}

	for _, c := range []struct {
		y, vcol int
		want    rune
	}{
		{1, 0, chunkCornerTop}, {1, 3, chunkHorizontal},
		{2, 0, chunkVertical},
		{3, 0, chunkCornerBot}, {3, 3, chunkArrow},
		{1, 4, 0}, {3, 4, 0}, {2, 1, 0}, {0, 0, 0}, // text cells and outside rows stay bare
	} {
		if got := cg.runeAt(c.y, c.vcol); got != c.want {
			t.Errorf("runeAt(%d,%d) = %q, want %q", c.y, c.vcol, got, c.want)
		}
	}

	if n := cg.cells(); n != 9 {
		t.Errorf("cells() = %d, want 9", n)
	}
	// sweep order: top corner fills leftwards, bars down, bottom corner
	// rightwards to the arrow
	for i, c := range []struct{ y, vcol int }{
		{1, 3}, {1, 2}, {1, 1}, {1, 0}, {2, 0}, {3, 0}, {3, 1}, {3, 2}, {3, 3},
	} {
		if got := cg.cellIndex(c.y, c.vcol); got != i {
			t.Errorf("cellIndex(%d,%d) = %d, want %d", c.y, c.vcol, got, i)
		}
	}
}
