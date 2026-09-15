package display

import (
	"github.com/gdamore/tcell/v3"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/shell"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/micro-editor/terminal"
)

type TermWindow struct {
	*View
	*shell.Terminal

	active bool
}

func NewTermWindow(x, y, w, h int, term *shell.Terminal) *TermWindow {
	tw := new(TermWindow)
	tw.View = new(View)
	tw.Terminal = term
	tw.X, tw.Y = x, y
	tw.Resize(w, h)
	return tw
}

// Resize informs the terminal of a resize event
func (w *TermWindow) Resize(width, height int) {
	if config.GetGlobalOption("statusline").(bool) {
		height--
	}
	w.Term.Resize(width, height)
	w.Width, w.Height = width, height
}

func (w *TermWindow) SetActive(b bool) {
	w.active = b
}

func (w *TermWindow) IsActive() bool {
	return w.active
}

func (w *TermWindow) LocFromVisual(vloc buffer.Loc) buffer.Loc {
	return vloc
}

func (w *TermWindow) Clear() {
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			screen.SetContent(w.X+x, w.Y+y, ' ', nil, config.DefStyle)
		}
	}
}

func (w *TermWindow) Relocate() bool { return true }
func (w *TermWindow) GetView() *View {
	return w.View
}
func (w *TermWindow) SetView(v *View) {
	w.View = v
}

// Display displays this terminal in a view
func (w *TermWindow) Display() {
	w.State.Lock()
	defer w.State.Unlock()

	var l buffer.Loc
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			l.X, l.Y = x, y
			c, f, b := w.State.Cell(x, y)

			fg, bg := int(f), int(b)
			if f == terminal.DefaultFG {
				fg = int(tcell.ColorDefault)
			}
			if b == terminal.DefaultBG {
				bg = int(tcell.ColorDefault)
			}
			st := tcell.StyleDefault.Foreground(config.GetColor256(fg)).Background(config.GetColor256(bg))

			if l.LessThan(w.Selection[1]) && l.GreaterEqual(w.Selection[0]) || l.LessThan(w.Selection[0]) && l.GreaterEqual(w.Selection[1]) {
				st = st.Reverse(true)
			}

			screen.SetContent(w.X+x, w.Y+y, c, nil, st)
		}
	}
	if config.GetGlobalOption("statusline").(bool) {
		statusLineStyle := config.DefStyle.Reverse(true)
		if style, ok := config.Colorscheme["statusline"]; ok {
			statusLineStyle = style
		}

		text := []byte(w.Name())
		textLen := util.CharacterCount(text)
		for x := 0; x < w.Width; x++ {
			if x < textLen {
				r, combc, size := util.DecodeCharacter(text)
				text = text[size:]
				screen.SetContent(w.X+x, w.Y+w.Height, r, combc, statusLineStyle)
			} else {
				screen.SetContent(w.X+x, w.Y+w.Height, ' ', nil, statusLineStyle)
			}
		}
	}
	if w.State.CursorVisible() && w.active {
		curx, cury := w.State.Cursor()
		if curx < w.Width && cury < w.Height {
			screen.ShowCursor(curx+w.X, cury+w.Y)
		}
	}
}
