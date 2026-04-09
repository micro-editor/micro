package buffer

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/micro-editor/micro/v2/internal/util"
)

// We want "^" and "$" to match only the beginning/end of a line, not the
// beginning/end of the search region if it is in the middle of a line.
// In that case we use padded regexps to require a rune before or after
// the match. (This also affects other empty-string patters like "\\b".)
// The following two flags indicate the padding used.
const (
	padStart = 1 << iota
	padEnd
)

func normalizeSearchBounds(b *Buffer, start, end Loc) (Loc, Loc) {
	lastcn := util.CharacterCount(b.LineBytes(b.LinesNum() - 1))
	if start.Y > b.LinesNum()-1 {
		start.X = lastcn - 1
	}
	if end.Y > b.LinesNum()-1 {
		end.X = lastcn
	}
	start.Y = util.Clamp(start.Y, 0, b.LinesNum()-1)
	end.Y = util.Clamp(end.Y, 0, b.LinesNum()-1)

	if start.GreaterThan(end) {
		start, end = end, start
	}
	return start, end
}

func regexPatternContainsEscapedNewline(s string) bool {
	slashes := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			slashes++
			continue
		}
		if s[i] == 'n' && slashes%2 == 1 {
			return true
		}
		slashes = 0
	}
	return false
}

func shouldUseMultilineSearch(s string, useRegex bool) bool {
	if useRegex {
		return regexPatternContainsEscapedNewline(s)
	}
	return strings.Contains(s, "\n")
}

// CompileSearchRegex compiles a search pattern according to regex mode and
// buffer settings. It returns the normalized pattern and compiled regexp.
func (b *Buffer) CompileSearchRegex(s string, useRegex bool, multiline bool) (string, *regexp.Regexp, error) {
	pattern := s
	if !useRegex {
		pattern = regexp.QuoteMeta(pattern)
	}

	flags := ""
	if b.Settings["ignorecase"].(bool) {
		flags += "i"
	}
	if multiline {
		flags += "m"
	}

	if flags != "" {
		r, err := regexp.Compile("(?"+flags+")" + pattern)
		return pattern, r, err
	}

	r, err := regexp.Compile(pattern)
	return pattern, r, err
}

func (b *Buffer) findDownMultiline(r *regexp.Regexp, start, end Loc) ([2]Loc, bool) {
	start, end = normalizeSearchBounds(b, start, end)

	searchEnd := end
	endLineCharCount := util.CharacterCount(b.LineBytes(end.Y))
	if end.X < endLineCharCount {
		searchEnd = end.Move(1, b)
	}

	joined := b.Substr(start, searchEnd)
	match := r.FindIndex(joined)
	if match == nil {
		return [2]Loc{}, false
	}

	matchStart := start.Move(util.RunePos(joined, match[0]), b)
	matchEnd := start.Move(util.RunePos(joined, match[1]), b)
	return [2]Loc{matchStart, matchEnd}, true
}

func (b *Buffer) findUpMultiline(r *regexp.Regexp, start, end Loc) ([2]Loc, bool) {
	start, end = normalizeSearchBounds(b, start, end)

	searchEnd := end
	endLineCharCount := util.CharacterCount(b.LineBytes(end.Y))
	if end.X < endLineCharCount {
		searchEnd = end.Move(1, b)
	}

	joined := b.Substr(start, searchEnd)
	allMatches := r.FindAllIndex(joined, -1)
	if len(allMatches) == 0 {
		return [2]Loc{}, false
	}
	match := allMatches[len(allMatches)-1]

	matchStart := start.Move(util.RunePos(joined, match[0]), b)
	matchEnd := start.Move(util.RunePos(joined, match[1]), b)
	return [2]Loc{matchStart, matchEnd}, true
}

func findLineParams(b *Buffer, start, end Loc, i int, r *regexp.Regexp) ([]byte, int, int, *regexp.Regexp) {
	l := b.LineBytes(i)
	charpos := 0
	padMode := 0

	if i == end.Y {
		nchars := util.CharacterCount(l)
		end.X = util.Clamp(end.X, 0, nchars)
		if end.X < nchars {
			l = util.SliceStart(l, end.X+1)
			padMode |= padEnd
		}
	}

	if i == start.Y {
		nchars := util.CharacterCount(l)
		start.X = util.Clamp(start.X, 0, nchars)
		if start.X > 0 {
			charpos = start.X - 1
			l = util.SliceEnd(l, charpos)
			padMode |= padStart
		}
	}

	if padMode != 0 {
		re, err := regexp.Compile(r.String() + `\E`)
		if err == nil {
			// r contains \Q without closing \E
			r = re
		}

		if padMode == padStart {
			r = regexp.MustCompile(".(?:" + r.String() + ")")
		} else if padMode == padEnd {
			r = regexp.MustCompile("(?:" + r.String() + ").")
		} else {
			// padMode == padStart|padEnd
			r = regexp.MustCompile(".(?:" + r.String() + ").")
		}
	}

	return l, charpos, padMode, r
}

func (b *Buffer) findDown(r *regexp.Regexp, start, end Loc) ([2]Loc, bool) {
	start, end = normalizeSearchBounds(b, start, end)

	for i := start.Y; i <= end.Y; i++ {
		l, charpos, padMode, rPadded := findLineParams(b, start, end, i, r)

		match := rPadded.FindIndex(l)

		if match != nil {
			if padMode&padStart != 0 {
				_, size := utf8.DecodeRune(l[match[0]:])
				match[0] += size
			}
			if padMode&padEnd != 0 {
				_, size := utf8.DecodeLastRune(l[:match[1]])
				match[1] -= size
			}
			start := Loc{charpos + util.RunePos(l, match[0]), i}
			end := Loc{charpos + util.RunePos(l, match[1]), i}
			return [2]Loc{start, end}, true
		}
	}
	return [2]Loc{}, false
}

func (b *Buffer) findUp(r *regexp.Regexp, start, end Loc) ([2]Loc, bool) {
	start, end = normalizeSearchBounds(b, start, end)

	for i := end.Y; i >= start.Y; i-- {
		charCount := util.CharacterCount(b.LineBytes(i))
		from := Loc{0, i}.Clamp(start, end)
		to := Loc{charCount, i}.Clamp(start, end)

		allMatches := b.findAll(r, from, to)
		if allMatches != nil {
			match := allMatches[len(allMatches)-1]
			return [2]Loc{match[0], match[1]}, true
		}
	}
	return [2]Loc{}, false
}

func (b *Buffer) findAll(r *regexp.Regexp, start, end Loc) [][2]Loc {
	var matches [][2]Loc
	loc := start
	for {
		match, found := b.findDown(r, loc, end)
		if !found {
			break
		}
		matches = append(matches, match)
		if match[0] != match[1] {
			loc = match[1]
		} else if match[1] != end {
			loc = match[1].Move(1, b)
		} else {
			break
		}
	}
	return matches
}

// FindNext finds the next occurrence of a given string in the buffer
// It returns the start and end location of the match (if found) and
// a boolean indicating if it was found
// May also return an error if the search regex is invalid
func (b *Buffer) FindNext(s string, start, end, from Loc, down bool, useRegex bool) ([2]Loc, bool, error) {
	if s == "" {
		return [2]Loc{}, false, nil
	}

	multiline := shouldUseMultilineSearch(s, useRegex)
	_, r, err := b.CompileSearchRegex(s, useRegex, false)

	if err != nil {
		return [2]Loc{}, false, err
	}

	var found bool
	var l [2]Loc
	if down {
		if multiline {
			l, found = b.findDownMultiline(r, from, end)
		} else {
			l, found = b.findDown(r, from, end)
		}
		if !found {
			if multiline {
				l, found = b.findDownMultiline(r, start, end)
			} else {
				l, found = b.findDown(r, start, end)
			}
		}
	} else {
		if multiline {
			l, found = b.findUpMultiline(r, from, start)
		} else {
			l, found = b.findUp(r, from, start)
		}
		if !found {
			if multiline {
				l, found = b.findUpMultiline(r, end, start)
			} else {
				l, found = b.findUp(r, end, start)
			}
		}
	}
	return l, found, nil
}

// ReplaceRegex replaces all occurrences of 'search' with 'replace' in the given area
// and returns the number of replacements made and the number of characters
// added or removed on the last line of the range
func (b *Buffer) ReplaceRegex(start, end Loc, search *regexp.Regexp, replace []byte, captureGroups bool) (int, int) {
	if start.GreaterThan(end) {
		start, end = end, start
	}

	charsEnd := util.CharacterCount(b.LineBytes(end.Y))
	found := 0
	var deltas []Delta

	for i := start.Y; i <= end.Y; i++ {
		l := b.LineBytes(i)
		charCount := util.CharacterCount(l)
		if (i == start.Y && start.X > 0) || (i == end.Y && end.X < charCount) {
			// This replacement code works in general, but it creates a separate
			// modification for each match. We only use it for the first and last
			// lines, which may use padded regexps

			from := Loc{0, i}.Clamp(start, end)
			to := Loc{charCount, i}.Clamp(start, end)
			matches := b.findAll(search, from, to)
			found += len(matches)

			for j := len(matches) - 1; j >= 0; j-- {
				// if we counted upwards, the different deltas would interfere
				match := matches[j]
				var newText []byte
				if captureGroups {
					newText = search.ReplaceAll(b.Substr(match[0], match[1]), replace)
				} else {
					newText = replace
				}
				deltas = append(deltas, Delta{newText, match[0], match[1]})
			}
		} else {
			newLine := search.ReplaceAllFunc(l, func(in []byte) []byte {
				found++
				var result []byte
				if captureGroups {
					match := search.FindSubmatchIndex(in)
					result = search.Expand(result, replace, in, match)
				} else {
					result = replace
				}
				return result
			})
			deltas = append(deltas, Delta{newLine, Loc{0, i}, Loc{charCount, i}})
		}
	}

	b.MultipleReplace(deltas)

	return found, util.CharacterCount(b.LineBytes(end.Y)) - charsEnd
}
