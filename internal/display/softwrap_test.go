package display

import (
	"strings"
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

func init() {
	ulua.L = lua.NewState()
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
}

// newWrapTestWindow builds a BufWindow whose bufWidth exactly equals paneWidth
// (ruler, diffgutter, scrollbar all off, so gutterOffset is 0), with softwrap
// on and the given wordwrap/tabsize/softwrapcolumn settings. Using the default
// `statusline: true` setting means updateDisplayInfo never touches
// screen.Screen, so this works without any terminal, real or simulated.
func newWrapTestWindow(text string, paneWidth int, wordwrap bool, tabsize int, softwrapcolumn int) *BufWindow {
	buf := buffer.NewBufferFromString(text, "", buffer.BTDefault)
	buf.Settings["softwrap"] = true
	buf.Settings["wordwrap"] = wordwrap
	buf.Settings["tabsize"] = float64(tabsize)
	buf.Settings["softwrapcolumn"] = float64(softwrapcolumn)
	buf.Settings["ruler"] = false
	buf.Settings["diffgutter"] = false
	buf.Settings["scrollbar"] = false

	w := NewBufWindow(0, 0, paneWidth, 10, buf)
	w.updateDisplayInfo()
	return w
}

// assertRoundTrip checks that mapping every buffer column on line 0 to a
// visual location and back returns the original column — the invariant that
// cursor up/down, mouse clicks, and selection all rely on. This is the
// property that broke for the wrapindent PR (#3107) when it injected phantom
// leading columns on continuation rows; softwrapcolumn does not, so it must
// hold here too.
func assertRoundTrip(t *testing.T, w *BufWindow, lineLen int) {
	t.Helper()
	for x := 0; x <= lineLen; x++ {
		loc := buffer.Loc{X: x, Y: 0}
		vloc := w.VLocFromLoc(loc)
		got := w.LocFromVLoc(vloc)
		assert.Equal(t, x, got.X, "round-trip mismatch at column %d (vloc=%+v)", x, vloc)
	}
}

func TestSoftWrapColumnRoundTrip(t *testing.T) {
	line := strings.Repeat("A", 60)

	t.Run("off (0) wraps at the full pane width", func(t *testing.T) {
		w := newWrapTestWindow(line, 40, false, 4, 0)
		assertRoundTrip(t, w, len(line))
		assert.Equal(t, 0, w.VLocFromLoc(buffer.Loc{X: 39, Y: 0}).Row)
		assert.Equal(t, 1, w.VLocFromLoc(buffer.Loc{X: 40, Y: 0}).Row)
	})

	t.Run("narrower than the pane wraps at the configured column", func(t *testing.T) {
		w := newWrapTestWindow(line, 58, false, 4, 30)
		assertRoundTrip(t, w, len(line))
		assert.Equal(t, 0, w.VLocFromLoc(buffer.Loc{X: 29, Y: 0}).Row)
		assert.Equal(t, 1, w.VLocFromLoc(buffer.Loc{X: 30, Y: 0}).Row)
	})

	t.Run("wider than the pane clamps to the pane width", func(t *testing.T) {
		w := newWrapTestWindow(line, 18, false, 4, 30)
		assertRoundTrip(t, w, len(line))
		assert.Equal(t, 0, w.VLocFromLoc(buffer.Loc{X: 17, Y: 0}).Row)
		assert.Equal(t, 1, w.VLocFromLoc(buffer.Loc{X: 18, Y: 0}).Row)
	})

	t.Run("wordwrap on with a word longer than the column", func(t *testing.T) {
		longWord := "short " + strings.Repeat("B", 50)
		w := newWrapTestWindow(longWord, 58, true, 4, 10)
		assertRoundTrip(t, w, len(longWord))
	})

	t.Run("tabs straddling the wrap column", func(t *testing.T) {
		tabby := "a\tb\tc\td\te\tf\tg\th"
		w := newWrapTestWindow(tabby, 58, false, 4, 10)
		assertRoundTrip(t, w, util.CharacterCount([]byte(tabby)))
	})
}
