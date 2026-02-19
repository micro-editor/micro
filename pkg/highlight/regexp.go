package highlight

import (
	"log"
	"time"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
)

// compileRegex compiles a pattern using regexp2 with a 1-second match timeout
// for backtracking safety.
func compileRegex(pattern string) (*regexp2.Regexp, error) {
	re, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, err
	}
	re.MatchTimeout = 1 * time.Second
	return re, nil
}

// matchString returns whether re matches s. On timeout, it logs the error
// and returns false.
func matchString(re *regexp2.Regexp, s string) bool {
	m, err := re.MatchString(s)
	if err != nil {
		log.Printf("highlight: regex timeout matching pattern %q: %v", re.String(), err)
		return false
	}
	return m
}

// matchBytes returns whether re matches b. On timeout, it logs the error
// and returns false.
func matchBytes(re *regexp2.Regexp, b []byte) bool {
	return matchString(re, string(b))
}

// charPosFromRunePos converts a rune index (as returned by regexp2) to a
// character index (which skips combining marks via isMark). For ASCII text
// (the vast majority of source code), rune pos == char pos.
func charPosFromRunePos(runeIdx int, str []byte) int {
	charPos := 0
	runeCount := 0
	for i := 0; i < len(str); {
		if runeCount >= runeIdx {
			return charPos
		}
		r, size := utf8.DecodeRune(str[i:])
		i += size
		runeCount++
		if !isMark(r) {
			charPos++
		}
	}
	return charPos
}
