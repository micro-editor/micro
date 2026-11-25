package display

import (
	runewidth "github.com/mattn/go-runewidth"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/util"
	"strings"
	"unicode/utf8"
)

type TabWindow struct {
	Names   []string
	active  int
	Y       int
	Width   int
	hscroll int
}

func NewTabWindow(w int, y int) *TabWindow {
	tw := new(TabWindow)
	tw.Width = w
	tw.Y = y
	return tw
}

func (w *TabWindow) Resize(width, height int) {
	w.Width = width
}

func (w *TabWindow) LocFromVisual(vloc buffer.Loc) int {
	x := -w.hscroll
	tabactiverunes, tabinactiverunes, tabdivrunes := GetTabRunes()
	for i, n := range w.Names {
		if i == w.active {
			x += len(tabactiverunes) / 2
		} else {
			x += len(tabinactiverunes) / 2
		}

		s := util.CharacterCountInString(n)
		if vloc.Y == w.Y && vloc.X < x+s {
			return i
		}
		x += s

		if i == w.active {
			x += len(tabactiverunes) - len(tabactiverunes)/2
		} else {
			x += len(tabinactiverunes) - len(tabinactiverunes)/2
		}
		x += len(tabdivrunes)
		if x >= w.Width {
			break
		}
	}
	return -1
}

func (w *TabWindow) Scroll(amt int) {
	w.hscroll += amt
	s := w.TotalSize()
	w.hscroll = util.Clamp(w.hscroll, 0, s-w.Width)

	if s-w.Width <= 0 {
		w.hscroll = 0
	}
}

func (w *TabWindow) TotalSize() int {
	sum := 2
	for _, n := range w.Names {
		sum += runewidth.StringWidth(n) + 4
	}
	return sum - 4
}

func (w *TabWindow) Active() int {
	return w.active
}

func (w *TabWindow) SetActive(a int) {
	w.active = a
	x := 2
	s := w.TotalSize()

	for i, n := range w.Names {
		c := util.CharacterCountInString(n)
		if i == a {
			if x+c >= w.hscroll+w.Width {
				w.hscroll = util.Clamp(x+c+1-w.Width, 0, s-w.Width)
			} else if x < w.hscroll {
				w.hscroll = util.Clamp(x-4, 0, s-w.Width)
			}
			break
		}
		x += c + 4
	}

	if s-w.Width <= 0 {
		w.hscroll = 0
	}
}

func GetTabRunes() ([]rune, []rune, []rune) {
	var tabactivechars string
	var tabinactivechars string
	var tabdivchars string
	for _, entry := range strings.Split(config.GetGlobalOption("tabbarchars").(string), ",") {
		split := strings.SplitN(entry, "=", 2)
		if len(split) < 2 {
			continue
		}
		key, val := split[0], split[1]
		switch key {
		case "active":
			tabactivechars = val
		case "inactive":
			tabinactivechars = val
		case "div":
			tabdivchars = val
		}
	}

	if utf8.RuneCountInString(tabactivechars) < 2 {
		tabactivechars = ""
	}
	if utf8.RuneCountInString(tabinactivechars) < 2 {
		tabinactivechars = ""
	}

	tabactiverunes := []rune(tabactivechars)
	tabinactiverunes := []rune(tabinactivechars)
	tabdivrunes := []rune(tabdivchars)
	return tabactiverunes, tabinactiverunes, tabdivrunes
}

func (w *TabWindow) Display() {
	x := -w.hscroll
	done := false

	globalTabReverse := config.GetGlobalOption("tabreverse").(bool)
	globalTabHighlight := config.GetGlobalOption("tabhighlight").(bool)
	tabBarStyle := config.DefStyle

	if style, ok := config.Colorscheme["tabbar"]; ok {
		tabBarStyle = style
	}
	if globalTabReverse {
		tabBarStyle = config.ReverseColor(tabBarStyle)
	}
	tabBarActiveStyle := tabBarStyle
	if globalTabHighlight {
		tabBarActiveStyle = config.ReverseColor(tabBarStyle)
	}
	if style, ok := config.Colorscheme["tabbar.active"]; ok {
		tabBarActiveStyle = style
	}
	tabBarInactiveStyle := tabBarStyle
	if style, ok := config.Colorscheme["tabbar.inactive"]; ok {
		tabBarInactiveStyle = style
	}
	tabBarDivStyle := tabBarStyle
	if style, ok := config.Colorscheme["tabbar.div"]; ok {
		tabBarDivStyle = style
	}

	draw := func(r rune, n int, active bool, tab bool, div bool) {
		style := tabBarStyle
		if tab {
			if active {
				style = tabBarActiveStyle
			} else {
				style = tabBarInactiveStyle
			}
		} else if div {
			style = tabBarDivStyle
		}

		for i := 0; i < n; i++ {
			rw := runewidth.RuneWidth(r)
			for j := 0; j < rw; j++ {
				c := r
				if j > 0 {
					c = ' '
				}
				if x == w.Width-1 && !done {
					screen.SetContent(w.Width-1, w.Y, '>', nil, style)
					x++
					break
				} else if x == 0 && w.hscroll > 0 {
					screen.SetContent(0, w.Y, '<', nil, style)
				} else if x >= 0 && x < w.Width {
					screen.SetContent(x, w.Y, c, nil, style)
				}
				x++
			}
		}
	}

	tabactiverunes, tabinactiverunes, tabdivrunes := GetTabRunes()
	leftactiverunes := tabactiverunes[0 : len(tabactiverunes)/2]
	rightactiverunes := tabactiverunes[len(tabactiverunes)/2:]

	leftinactiverunes := tabinactiverunes[0 : len(tabinactiverunes)/2]
	rightinactiverunes := tabinactiverunes[len(tabinactiverunes)/2:]

	for i, n := range w.Names {
		if i == w.active {
			for j := 0; j < len(leftactiverunes); j++ {
				draw(leftactiverunes[j], 1, true, true, false)
			}
		} else {
			for j := 0; j < len(leftinactiverunes); j++ {
				draw(leftinactiverunes[j], 1, false, true, false)
			}
		}

		for _, c := range n {
			draw(c, 1, i == w.active, true, false)
		}

		if i == len(w.Names)-1 {
			done = true
		}

		if i == w.active {
			for j := 0; j < len(rightactiverunes); j++ {
				draw(rightactiverunes[j], 1, true, true, false)
			}
		} else {
			for j := 0; j < len(rightinactiverunes); j++ {
				draw(rightinactiverunes[j], 1, false, true, false)
			}
		}

		for j := 0; j < len(tabdivrunes); j++ {
			draw(tabdivrunes[j], 1, false, false, true)
		}

		if x >= w.Width {
			break
		}
	}

	if x < w.Width {
		draw(' ', w.Width-x, false, false, false)
	}
}
