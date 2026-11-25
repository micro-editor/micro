package clipboard

import (
	"errors"
	"github.com/zyedidia/clipper"
)

type Method int

const (
	// External relies on external tools for accessing the clipboard
	// These include xclip, xsel, wl-clipboard for linux, pbcopy/pbpaste on Mac,
	// and Syscalls on Windows.
	External Method = iota
	// Terminal uses the terminal to manage the clipboard via OSC 52. Many
	// terminals do not support OSC 52, in which case this method won't work.
	Terminal
	// Internal just manages the clipboard with an internal buffer and doesn't
	// attempt to interface with the system clipboard
	Internal
)

// CurrentMethod is the method used to store clipboard information
var CurrentMethod Method = Internal

// A Register is a buffer used to store text. The system clipboard has the 'clipboard'
// and 'primary' (linux-only) registers, but other registers may be used internal to micro.
type Register int

const (
	// ClipboardReg is the main system clipboard
	ClipboardReg Register = -1
	// PrimaryReg is the system primary clipboard (linux only)
	PrimaryReg = -2
)

var clipboard clipper.Clipboard

// Initialize attempts to initialize the clipboard using the given method
func Initialize(m Method) error {
	var err error
	switch m {
	case External:
		clips := make([]clipper.Clipboard, 0, len(clipper.Clipboards)+1)
		clips = append(clips, &clipper.Custom{
			Name: "micro-clip",
		})
		clips = append(clips, clipper.Clipboards...)
		clipboard, err = clipper.GetClipboard(clips...)
	}
	if err != nil {
		CurrentMethod = Internal
	}
	return err
}

// SetMethod changes the clipboard access method
func SetMethod(m string) Method {
	switch m {
	case "internal":
		CurrentMethod = Internal
	case "external":
		CurrentMethod = External
	case "terminal":
		CurrentMethod = Terminal
	}
	return CurrentMethod
}

// Read reads from a clipboard register
func Read(r Register) (string, error) {
	return read(r, CurrentMethod)
}

// Write writes text to a clipboard register
func Write(text string, r Register) error {
	return write(text, r, CurrentMethod)
}

// ReadMulti reads text array from a clipboard register, which can be a multi cursor clipboard
func ReadMulti(r Register) ([]string, error) {
	clip, err := Read(r)
	multivalid := false
	if err == nil {
		multivalid = ValidMulti(r, &clip)
	} else {
		multivalid = ValidMulti(r, nil)
	}

	if !multivalid {
		returnarray := make([]string, 1, 1)
		if err == nil {
			returnarray[0] = clip
		} else {
			returnarray[0] = ""
		}
		return returnarray, err
	}

	return multi.getAllText(r), nil
}

// WriteMulti writes text to a clipboard register for a certain multi-cursor
func WriteMulti(text string, r Register, num int, ncursors int) error {
	return writeMulti(text, r, num, ncursors, CurrentMethod)
}

// ValidMulti checks if the internal multi-clipboard is valid and up-to-date
// with the system clipboard
func ValidMulti(r Register, clip *string) bool {
	return multi.isValid(r, clip)
}

func writeMulti(text string, r Register, num int, ncursors int, m Method) error {
	// Write to multi cursor clipboard
	multi.writeText(text, r, num, ncursors)

	// Write to normal cliipboard
	multitext := multi.getAllTextConcated(r)
	if multitext == "" {
		return write("", r, m)
	}
	return write(multitext, r, m)
}

func read(r Register, m Method) (string, error) {
	switch m {
	case External:
		switch r {
		case ClipboardReg:
			b, e := clipboard.ReadAll(clipper.RegClipboard)
			return string(b), e
		case PrimaryReg:
			b, e := clipboard.ReadAll(clipper.RegPrimary)
			return string(b), e
		default:
			return internal.read(r), nil
		}
	case Internal:
		return internal.read(r), nil
	case Terminal:
		switch r {
		case ClipboardReg:
			// terminal paste works by sending an esc sequence to the
			// terminal to trigger a paste event
			return terminal.read("clipboard")
		case PrimaryReg:
			return terminal.read("primary")
		default:
			return internal.read(r), nil
		}
	}
	return "", errors.New("Invalid clipboard method")
}

func write(text string, r Register, m Method) error {
	switch m {
	case External:
		switch r {
		case ClipboardReg:
			return clipboard.WriteAll(clipper.RegClipboard, []byte(text))
		case PrimaryReg:
			return clipboard.WriteAll(clipper.RegPrimary, []byte(text))
		default:
			internal.write(text, r)
		}
	case Internal:
		internal.write(text, r)
	case Terminal:
		switch r {
		case ClipboardReg:
			return terminal.write(text, "c")
		case PrimaryReg:
			return terminal.write(text, "p")
		default:
			internal.write(text, r)
		}
	}
	return nil
}
