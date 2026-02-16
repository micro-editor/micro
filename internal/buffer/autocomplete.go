package buffer

import (
	"bytes"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/micro-editor/micro/v2/internal/util"
)

// A Completer is a function that takes a buffer and returns info
// describing what autocompletions should be inserted at the current
// cursor location
// It returns a list of string suggestions which will be inserted at
// the current cursor location if selected as well as a list of
// suggestion names which can be displayed in an autocomplete box or
// other UI element
type Completer func(*Buffer) ([]string, []string)

func (b *Buffer) GetSuggestions() {

}

func AutocompleteRuneCheck(cursor *Cursor) bool {
	r := cursor.RuneUnder(cursor.X)
	prev := cursor.RuneUnder(cursor.X - 1)
	if !util.IsAutocomplete(prev) || util.IsWordChar(r) {
		// don't autocomplete if cursor is within a word
		return false
	}
	return true
}

func AutocompleteCursorCheck(cursor *Cursor) bool {
	if cursor.HasSelection() {
		return false
	}
	if cursor.X == 0 {
		return false
	}
	return true
}

func (b *Buffer) StartAutocomplete(c Completer) bool {
	b.Completions, b.Suggestions = c(b)
	if len(b.Completions) != len(b.Suggestions) || len(b.Completions) == 0 {
		return false
	}
	b.CurSuggestion = -1
	return true
}

// CycleAutocomplete moves to the next suggestion and return the previous suggestion
func (b *Buffer) CycleAutocomplete(forward bool) int {
	prevSuggestion := b.CurSuggestion

	if forward {
		b.CurSuggestion++
	} else {
		b.CurSuggestion--
	}
	if b.CurSuggestion >= len(b.Suggestions) {
		b.CurSuggestion = 0
	} else if b.CurSuggestion < 0 {
		b.CurSuggestion = len(b.Suggestions) - 1
	}

	if len(b.Suggestions) > 1 {
		b.HasSuggestions = true
	}

	return prevSuggestion
}

func (b *Buffer) PerformSingleAutocomplete(prevSuggestion int, cursor *Cursor) {
	curLoc := cursor.Loc
	curStart := curLoc
	curEnd := curLoc

	if prevSuggestion < len(b.Suggestions) && prevSuggestion >= 0 {
		curStart = curEnd.Move(-util.CharacterCountInString(b.Completions[prevSuggestion]), b)
	}

	hasSuggestions := b.HasSuggestions
	b.Replace(curStart, curEnd, b.Completions[b.CurSuggestion])
	b.HasSuggestions = hasSuggestions
}

// GetWord gets the most recent word separated by any separator
// (whitespace, punctuation, any non alphanumeric character)
func (b *Buffer) GetWord() ([]byte, int) {
	c := b.GetActiveCursor()
	l := b.LineBytes(c.Y)
	l = util.SliceStart(l, c.X)

	if c.X == 0 || util.IsWhitespace(b.RuneAt(c.Loc.Move(-1, b))) {
		return []byte{}, -1
	}

	if util.IsNonWordChar(b.RuneAt(c.Loc.Move(-1, b))) {
		return []byte{}, c.X
	}

	args := bytes.FieldsFunc(l, util.IsNonWordChar)
	input := args[len(args)-1]
	return input, c.X - util.CharacterCount(input)
}

// GetArg gets the most recent word (separated by ' ' only)
func (b *Buffer) GetArg() (string, int) {
	c := b.GetActiveCursor()
	l := b.LineBytes(c.Y)
	l = util.SliceStart(l, c.X)

	args := bytes.Split(l, []byte{' '})
	input := string(args[len(args)-1])
	argstart := 0
	for i, a := range args {
		if i == len(args)-1 {
			break
		}
		argstart += util.CharacterCount(a) + 1
	}

	return input, argstart
}

// FileComplete autocompletes filenames
func FileComplete(b *Buffer) ([]string, []string) {
	c := b.GetActiveCursor()
	input, argstart := b.GetArg()

	sep := string(os.PathSeparator)
	dirs := strings.Split(input, sep)

	var files []fs.DirEntry
	var err error
	if len(dirs) > 1 {
		directories := strings.Join(dirs[:len(dirs)-1], sep) + sep

		directories, _ = util.ReplaceHome(directories)
		files, err = os.ReadDir(directories)
	} else {
		files, err = os.ReadDir(".")
	}

	if err != nil {
		return nil, nil
	}

	var suggestions []string
	for _, f := range files {
		name := f.Name()
		if f.IsDir() {
			name += sep
		}
		if strings.HasPrefix(name, dirs[len(dirs)-1]) {
			suggestions = append(suggestions, name)
		}
	}

	sort.Strings(suggestions)
	completions := make([]string, len(suggestions))
	for i := range suggestions {
		var complete string
		if len(dirs) > 1 {
			complete = strings.Join(dirs[:len(dirs)-1], sep) + sep + suggestions[i]
		} else {
			complete = suggestions[i]
		}
		completions[i] = util.SliceEndStr(complete, c.X-argstart)
	}

	return completions, suggestions
}

// BufferComplete autocompletes based on previous words in the buffer
func BufferComplete(b *Buffer) ([]string, []string) {
	c := b.GetActiveCursor()
	input, argstart := b.GetWord()

	if argstart == -1 {
		return []string{}, []string{}
	}

	inputLen := util.CharacterCount(input)

	suggestionsSet := make(map[string]struct{})

	var suggestions []string
	for i := c.Y; i >= 0; i-- {
		l := b.LineBytes(i)
		words := bytes.FieldsFunc(l, util.IsNonWordChar)
		for _, w := range words {
			if bytes.HasPrefix(w, input) && util.CharacterCount(w) > inputLen {
				strw := string(w)
				if _, ok := suggestionsSet[strw]; !ok {
					suggestionsSet[strw] = struct{}{}
					suggestions = append(suggestions, strw)
				}
			}
		}
	}
	for i := c.Y + 1; i < b.LinesNum(); i++ {
		l := b.LineBytes(i)
		words := bytes.FieldsFunc(l, util.IsNonWordChar)
		for _, w := range words {
			if bytes.HasPrefix(w, input) && util.CharacterCount(w) > inputLen {
				strw := string(w)
				if _, ok := suggestionsSet[strw]; !ok {
					suggestionsSet[strw] = struct{}{}
					suggestions = append(suggestions, strw)
				}
			}
		}
	}
	if len(suggestions) > 1 {
		suggestions = append(suggestions, string(input))
	}

	completions := make([]string, len(suggestions))
	for i := range suggestions {
		completions[i] = util.SliceEndStr(suggestions[i], c.X-argstart)
	}

	return completions, suggestions
}
