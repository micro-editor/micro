package clipboard

import (
	"bytes"
)

// For storing multi cursor clipboard contents
type multiClipboard map[Register][]string

var multi multiClipboard

func (c multiClipboard) getAllTextConcated(r Register) string {
	multitext := c.getAllText(r)
	if multitext == nil {
		return ""
	}

	buf := &bytes.Buffer{}
	for i, s := range multitext {
		buf.WriteString(s)
		if i != len(multitext)-1 {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func (c multiClipboard) getAllText(r Register) []string {
	return c[r]
}

func (c multiClipboard) getText(r Register, num int) string {
	content := c[r]
	if content == nil || len(content) <= num {
		return ""
	}

	return content[num]
}

// isValid checks if the text stored in this multi-clipboard is the same as the
// text stored in the system clipboard (provided as an argument), and therefore
// if it is safe to use the multi-clipboard for pasting instead of the system
// clipboard.
func (c multiClipboard) isValid(r Register, clipboard *string) bool {
	content := c[r]
	if content == nil {
		return false
	}

	return c.getAllTextConcated(r) == *clipboard
}

func (c multiClipboard) writeText(text string, r Register, num int, ncursors int) {
	content := c[r]
	if content == nil || len(content) != ncursors {
		content = make([]string, ncursors, ncursors)
		c[r] = content
	}

	if num >= ncursors {
		return
	}

	content[num] = text
}

func init() {
	multi = make(multiClipboard)
}
