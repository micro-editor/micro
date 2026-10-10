package display

import (
	"strings"
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/stretchr/testify/assert"
)

// A line that exactly fills the window's content width must not count as
// needing an extra, empty row.
func TestGetRowCountExactWidth(t *testing.T) {
	tests := []struct {
		name             string
		width            int
		expectedRows     int
		expectedLastRowX int
	}{
		{"one narrower than width: no wrap", 79, 1, 79},
		{"exactly window width: must not wrap to an extra row", 80, 1, 80},
		{"one wider than width: wraps with 1 overflow char", 81, 2, 1},
		{"two wider than width: wraps with 2 overflow chars", 82, 2, 2},
		{"exactly two window widths: wraps once, no extra row", 160, 2, 80},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := newSoftwrapTestWindow(t, strings.Repeat("x", tc.width), 80, 24)

			assert.Equal(t, tc.expectedRows, w.getRowCount(0))

			vloc := w.getVLocFromLoc(buffer.Loc{X: tc.width, Y: 0})
			assert.Equal(t, tc.expectedRows-1, vloc.Row)
			assert.Equal(t, tc.expectedLastRowX, vloc.VisualX)
		})
	}
}
