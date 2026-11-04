package config

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/stretchr/testify/assert"
)

func TestSimpleStringToStyle(t *testing.T) {
	s := StringToStyle("lightblue,magenta")

	fg := s.GetForeground()
	bg := s.GetBackground()

	assert.Equal(t, tcell.ColorBlue, fg)
	assert.Equal(t, tcell.ColorPurple, bg)
}

func TestAttributeStringToStyle(t *testing.T) {
	s := StringToStyle("bold cyan,brightcyan")

	fg := s.GetForeground()
	bg := s.GetBackground()
	attr := s.GetAttributes()

	assert.Equal(t, tcell.ColorTeal, fg)
	assert.Equal(t, tcell.ColorAqua, bg)
	assert.Equal(t, tcell.AttrBold, attr)
}

func TestMultiAttributesStringToStyle(t *testing.T) {
	s := StringToStyle("bold blink dim italic reverse strikethrough underline cyan,brightcyan")

	fg := s.GetForeground()
	bg := s.GetBackground()
	attr := s.GetAttributes()
	ul := s.GetUnderlineStyle()

	assert.Equal(t, tcell.ColorTeal, fg)
	assert.Equal(t, tcell.ColorAqua, bg)
	assert.Equal(t, tcell.AttrBold|
		tcell.AttrBlink|
		tcell.AttrDim|
		tcell.AttrItalic|
		tcell.AttrReverse|
		tcell.AttrStrikeThrough,
		attr)
	assert.Equal(t, tcell.UnderlineStyleSolid, ul)
}

func TestColor256StringToStyle(t *testing.T) {
	s := StringToStyle("128,60")

	fg := s.GetForeground()
	bg := s.GetBackground()

	assert.Equal(t, tcell.Color128, fg)
	assert.Equal(t, tcell.Color60, bg)
}

func TestColorHexStringToStyle(t *testing.T) {
	s := StringToStyle("#deadbe,#ef1234")

	fg := s.GetForeground()
	bg := s.GetBackground()

	assert.Equal(t, tcell.NewRGBColor(222, 173, 190), fg)
	assert.Equal(t, tcell.NewRGBColor(239, 18, 52), bg)
}

func TestColorschemeParser(t *testing.T) {
	testColorscheme := `color-link default "#F8F8F2,#282828"
color-link comment "#75715E,#282828"
# comment
color-link identifier "#66D9EF,#282828" #comment
color-link constant "#AE81FF,#282828"
color-link constant.string "#E6DB74,#282828"
color-link constant.string.char "#BDE6AD,#282828"`

	c, err := ParseColorscheme("testColorscheme", testColorscheme, nil)
	assert.Nil(t, err)

	fg := c["comment"].GetForeground()
	bg := c["comment"].GetBackground()

	assert.Equal(t, tcell.NewRGBColor(117, 113, 94), fg)
	assert.Equal(t, tcell.NewRGBColor(40, 40, 40), bg)
}
