package util

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/blang/semver"
	runewidth "github.com/mattn/go-runewidth"
)

var (
	// These variables should be set by the linker when compiling

	// Version is the version number or commit hash
	Version = "0.0.0-unknown"
	// SemVersion is the Semantic version
	SemVersion semver.Version
	// CommitHash is the commit this version was built on
	CommitHash = "Unknown"
	// CompileDate is the date this binary was compiled on
	CompileDate = "Unknown"
	// Debug logging
	Debug = "OFF"

	// Stdout is a buffer that is written to stdout when micro closes
	Stdout *bytes.Buffer
	// Sigterm is a channel where micro exits when written
	Sigterm chan os.Signal

	// To be used for fails on (over-)write with safe writes
	ErrOverwrite = OverwriteError{}
)

// To be used for file writes before umask is applied
const FileMode os.FileMode = 0666

const BackupSuffix = ".micro-backup"

const OverwriteFailMsg = `An error occurred while writing to the file:

%s

The file may be corrupted now. The good news is that it has been
successfully backed up. Next time you open this file with Micro,
Micro will ask if you want to recover it from the backup.

The backup path is:

%s`

// OverwriteError is a custom error to add additional information
type OverwriteError struct {
	What       error
	BackupName string
}

func (e OverwriteError) Error() string {
	return fmt.Sprintf(OverwriteFailMsg, e.What, e.BackupName)
}

func (e OverwriteError) Is(target error) bool {
	return target == ErrOverwrite
}

func (e OverwriteError) Unwrap() error {
	return e.What
}

func init() {
	var err error
	SemVersion, err = semver.Make(Version)
	if err != nil {
		fmt.Println("Invalid version: ", Version, err)
	}

	Stdout = new(bytes.Buffer)
}

// SliceEnd returns a byte slice where the index is a rune index
// Slices off the start of the slice
func SliceEnd(slc []byte, index int) []byte {
	len := len(slc)
	i := 0
	totalSize := 0
	for totalSize < len {
		if i >= index {
			return slc[totalSize:]
		}

		_, _, size := DecodeCharacter(slc[totalSize:])
		totalSize += size
		i++
	}

	return slc[totalSize:]
}

// SliceEndStr is the same as SliceEnd but for strings
func SliceEndStr(str string, index int) string {
	len := len(str)
	i := 0
	totalSize := 0
	for totalSize < len {
		if i >= index {
			return str[totalSize:]
		}

		_, _, size := DecodeCharacterInString(str[totalSize:])
		totalSize += size
		i++
	}

	return str[totalSize:]
}

// SliceStart returns a byte slice where the index is a rune index
// Slices off the end of the slice
func SliceStart(slc []byte, index int) []byte {
	len := len(slc)
	i := 0
	totalSize := 0
	for totalSize < len {
		if i >= index {
			return slc[:totalSize]
		}

		_, _, size := DecodeCharacter(slc[totalSize:])
		totalSize += size
		i++
	}

	return slc[:totalSize]
}

// SliceStartStr is the same as SliceStart but for strings
func SliceStartStr(str string, index int) string {
	len := len(str)
	i := 0
	totalSize := 0
	for totalSize < len {
		if i >= index {
			return str[:totalSize]
		}

		_, _, size := DecodeCharacterInString(str[totalSize:])
		totalSize += size
		i++
	}

	return str[:totalSize]
}

// SliceVisualEnd will take a byte slice and slice off the start
// up to a given visual index. If the index is in the middle of a
// rune the number of visual columns into the rune will be returned
// It will also return the char pos of the first character of the slice
func SliceVisualEnd(b []byte, n, tabsize int) ([]byte, int, int) {
	width := 0
	i := 0
	for len(b) > 0 {
		r, _, size := DecodeCharacter(b)

		w := 0
		switch r {
		case '\t':
			ts := tabsize - (width % tabsize)
			w = ts
		default:
			w = runewidth.RuneWidth(r)
		}
		if width+w > n {
			return b, n - width, i
		}
		width += w
		b = b[size:]
		i++
	}
	return b, n - width, i
}

// Abs is a simple absolute value function for ints
func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// StringWidth returns the visual width of a byte array indexed from 0 to n (rune index)
// with a given tabsize
func StringWidth(b []byte, n, tabsize int) int {
	if n <= 0 {
		return 0
	}
	i := 0
	width := 0
	for len(b) > 0 {
		r, _, size := DecodeCharacter(b)
		b = b[size:]

		switch r {
		case '\t':
			ts := tabsize - (width % tabsize)
			width += ts
		default:
			width += runewidth.RuneWidth(r)
		}

		i++

		if i == n {
			return width
		}
	}
	return width
}

// Min takes the min of two ints
func Min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// Max takes the max of two ints
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// FSize gets the size of a file
func FSize(f *os.File) int64 {
	fi, _ := f.Stat()
	return fi.Size()
}

// IsWordChar returns whether or not a rune is a 'word character'
// Word characters are defined as numbers, letters or sub-word delimiters
func IsWordChar(r rune) bool {
	return IsAlphanumeric(r) || IsSubwordDelimiter(r)
}

// IsNonWordChar returns whether or not a rune is not a 'word character'
// Non word characters are defined as all characters not being numbers, letters or sub-word delimiters
// See IsWordChar()
func IsNonWordChar(r rune) bool {
	return !IsWordChar(r)
}

// IsSubwordDelimiter returns whether or not a rune is a 'sub-word delimiter character'
// i.e. is considered a part of the word and is used as a delimiter between sub-words of the word.
// For now the only sub-word delimiter character is '_'.
func IsSubwordDelimiter(r rune) bool {
	return r == '_'
}

// IsAlphanumeric returns whether or not a rune is an 'alphanumeric character'
// Alphanumeric characters are defined as numbers or letters
func IsAlphanumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// IsUpperAlphanumeric returns whether or not a rune is an 'upper alphanumeric character'
// Upper alphanumeric characters are defined as numbers or upper-case letters
func IsUpperAlphanumeric(r rune) bool {
	return IsUpperLetter(r) || unicode.IsNumber(r)
}

// IsLowerAlphanumeric returns whether or not a rune is a 'lower alphanumeric character'
// Lower alphanumeric characters are defined as numbers or lower-case letters
func IsLowerAlphanumeric(r rune) bool {
	return IsLowerLetter(r) || unicode.IsNumber(r)
}

// IsUpperLetter returns whether or not a rune is an 'upper letter character'
// Upper letter characters are defined as upper-case letters
func IsUpperLetter(r rune) bool {
	// unicode.IsUpper() returns true for letters only
	return unicode.IsUpper(r)
}

// IsLowerLetter returns whether or not a rune is a 'lower letter character'
// Lower letter characters are defined as lower-case letters
func IsLowerLetter(r rune) bool {
	// unicode.IsLower() returns true for letters only
	return unicode.IsLower(r)
}

// Spaces returns a string with n spaces
func Spaces(n int) string {
	return strings.Repeat(" ", n)
}

// IsSpaces checks if a given string is only spaces
func IsSpaces(str []byte) bool {
	for _, c := range str {
		if c != ' ' {
			return false
		}
	}

	return true
}

// IsSpacesOrTabs checks if a given string contains only spaces and tabs
func IsSpacesOrTabs(str []byte) bool {
	for _, c := range str {
		if c != ' ' && c != '\t' {
			return false
		}
	}

	return true
}

// IsWhitespace returns true if the given rune is a space, tab, or newline
func IsWhitespace(c rune) bool {
	return unicode.IsSpace(c)
}

// IsBytesWhitespace returns true if the given bytes are all whitespace
func IsBytesWhitespace(b []byte) bool {
	for _, c := range b {
		if !IsWhitespace(rune(c)) {
			return false
		}
	}
	return true
}

// RunePos returns the rune index of a given byte index
// Make sure the byte index is not between code points
func RunePos(b []byte, i int) int {
	return CharacterCount(b[:i])
}

// IndexAnyUnquoted returns the first position in s of a character from chars.
// Escaped (with backslash) and quoted (with single or double quotes) characters
// are ignored. Returns -1 if not successful
func IndexAnyUnquoted(s, chars string) int {
	var e bool
	var q rune
	for i, r := range s {
		if e {
			e = false
		} else if (q == 0 || q == '"') && r == '\\' {
			e = true
		} else if r == q {
			q = 0
		} else if q == 0 && (r == '\'' || r == '"') {
			q = r
		} else if q == 0 && strings.IndexRune(chars, r) >= 0 {
			return i
		}
	}
	return -1
}

// MakeRelative will attempt to make a relative path between path and base
func MakeRelative(path, base string) (string, error) {
	if len(path) > 0 {
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return path, err
		}
		return rel, nil
	}
	return path, nil
}

// ReplaceHome takes a path as input and replaces ~ at the start of the path with the user's
// home directory. Does nothing if the path does not start with '~'.
func ReplaceHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	var userData *user.User
	var err error

	homeString := strings.Split(path, "/")[0]
	if homeString == "~" {
		userData, err = user.Current()
		if err != nil {
			return "", errors.New("Could not find user: " + err.Error())
		}
	} else {
		userData, err = user.Lookup(homeString[1:])
		if err != nil {
			return "", errors.New("Could not find user: " + err.Error())
		}
	}

	home := userData.HomeDir

	return strings.Replace(path, homeString, home, 1), nil
}

// GetPathAndCursorPosition returns a filename without everything following a `:`
// This is used for opening files like util.go:10:5 to specify a line and column
// Special cases like Windows Absolute path (C:\myfile.txt:10:5) are handled correctly.
func GetPathAndCursorPosition(path string) (string, []string) {
	re := regexp.MustCompile(`([\s\S]+?)(?::(\d+))(?::(\d+))?$`)
	match := re.FindStringSubmatch(path)
	// no lines/columns were specified in the path, return just the path with no cursor location
	if len(match) == 0 {
		return path, nil
	} else if match[len(match)-1] != "" {
		// if the last capture group match isn't empty then both line and column were provided
		return match[1], match[2:]
	}
	// if it was empty, then only a line was provided, so default to column 0
	return match[1], []string{match[2], "0"}
}

// GetModTime returns the last modification time for a given file
func GetModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Now(), err
	}
	return info.ModTime(), nil
}

func HashStringMd5(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

// EscapePathUrl encodes the path in URL query form
func EscapePathUrl(path string) string {
	return url.QueryEscape(filepath.ToSlash(path))
}

// EscapePathLegacy replaces every path separator in a given path with a %
func EscapePathLegacy(path string) string {
	path = filepath.ToSlash(path)
	if runtime.GOOS == "windows" {
		// ':' is not valid in a path name on Windows but is ok on Unix
		path = strings.ReplaceAll(path, ":", "%")
	}
	return strings.ReplaceAll(path, "/", "%")
}

// DetermineEscapePath escapes a path, determining whether it should be escaped
// using URL encoding (preferred, since it encodes unambiguously) or
// legacy encoding with '%' (for backward compatibility, if the legacy-escaped
// path exists in the given directory).
// In case the length of the escaped path (plus the backup extension) exceeds
// the filename length limit, a hash of the path is returned instead. In such
// case the second return value is the name of a file the original path should
// be saved to (since the original path cannot be derived from its hash).
// Otherwise the second return value is an empty string.
func DetermineEscapePath(dir string, path string) (string, string) {
	url := filepath.Join(dir, EscapePathUrl(path))
	if _, err := os.Stat(url); err == nil {
		return url, ""
	}

	legacy := filepath.Join(dir, EscapePathLegacy(path))
	if _, err := os.Stat(legacy); err == nil {
		return legacy, ""
	}

	if len(url)+len(BackupSuffix) > 255 {
		hash := HashStringMd5(path)
		return filepath.Join(dir, hash), filepath.Join(dir, hash+".path")
	}

	return url, ""
}

// GetLeadingWhitespace returns the leading whitespace of the given byte array
func GetLeadingWhitespace(b []byte) []byte {
	ws := []byte{}
	for len(b) > 0 {
		r, _, size := DecodeCharacter(b)
		if r == ' ' || r == '\t' {
			ws = append(ws, byte(r))
		} else {
			break
		}

		b = b[size:]
	}
	return ws
}

// GetTrailingWhitespace returns the trailing whitespace of the given byte array
func GetTrailingWhitespace(b []byte) []byte {
	ws := []byte{}
	for len(b) > 0 {
		r, size := utf8.DecodeLastRune(b)
		if IsWhitespace(r) {
			ws = append([]byte(string(r)), ws...)
		} else {
			break
		}

		b = b[:len(b)-size]
	}
	return ws
}

// HasTrailingWhitespace returns true if the given byte array ends with a whitespace
func HasTrailingWhitespace(b []byte) bool {
	r, _ := utf8.DecodeLastRune(b)
	return IsWhitespace(r)
}

// GetTextLengthAfterLastLinebreak returns the length of the remaining
// characters after the last line break in the given byte array and whether it
// contains line breaks.
// If no line breaks were found, it returns the length of all characters in the byte array.
func GetTextLengthAfterLastLinebreak(text []byte) (int, bool) {
	lastnl := bytes.LastIndex(text, []byte{'\n'})
	if lastnl >= 0 {
		return CharacterCount(text[lastnl+1:]), true
	}
	return CharacterCount(text), false
}

// IntOpt turns a float64 setting to an int
func IntOpt(opt any) int {
	return int(opt.(float64))
}

// GetCharPosInLine gets the char position of a visual x y
// coordinate (this is necessary because tabs are 1 char but
// 4 visual spaces)
func GetCharPosInLine(b []byte, visualPos int, tabsize int) int {
	// Scan rune by rune until we exceed the visual width that we are
	// looking for. Then we can return the character position we have found
	i := 0     // char pos
	width := 0 // string visual width
	for len(b) > 0 {
		r, _, size := DecodeCharacter(b)
		b = b[size:]

		switch r {
		case '\t':
			ts := tabsize - (width % tabsize)
			width += ts
		default:
			width += runewidth.RuneWidth(r)
		}

		if width >= visualPos {
			if width == visualPos {
				i++
			}
			break
		}
		i++
	}

	return i
}

// ParseBool is almost exactly like strconv.ParseBool, except it also accepts 'on' and 'off'
// as 'true' and 'false' respectively
func ParseBool(str string) (bool, error) {
	if str == "on" {
		return true, nil
	}
	if str == "off" {
		return false, nil
	}
	return strconv.ParseBool(str)
}

// Clamp clamps a value between min and max
func Clamp(val, min, max int) int {
	if val < min {
		val = min
	} else if val > max {
		val = max
	}
	return val
}

// IsAutocomplete returns whether a character should begin an autocompletion.
func IsAutocomplete(c rune) bool {
	return c == '.' || IsWordChar(c)
}

// String converts a byte array to a string (for lua plugins)
func String(s []byte) string {
	return string(s)
}

// Unzip unzips a file to given folder
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

// HttpRequest returns a new http.Client for making custom requests (for lua plugins)
func HttpRequest(method string, url string, headers []string) (resp *http.Response, err error) {
	client := http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(headers); i += 2 {
		req.Header.Add(headers[i], headers[i+1])
	}
	return client.Do(req)
}

// SafeWrite writes bytes to a file in a "safe" way, preventing loss of the
// contents of the file if it fails to write the new contents.
// This means that the file is not overwritten directly but by writing to a
// temporary file first.
//
// If rename is true, write is performed atomically, by renaming the temporary
// file to the target file after the data is successfully written to the
// temporary file. This guarantees that the file will not remain in a corrupted
// state, but it also has limitations, e.g. the file should not be a symlink
// (otherwise SafeWrite silently replaces this symlink with a regular file),
// the file creation date in Linux is not preserved (since the file inode
// changes) etc. Use SafeWrite with rename=true for files that are only created
// and used by micro for its own needs and are not supposed to be used directly
// by the user.
//
// If rename is false, write is performed by overwriting the target file after
// the data is successfully written to the temporary file.
// This means that the target file may remain corrupted if overwrite fails,
// but in such case the temporary file is preserved as a backup so the file
// can be recovered later. So it is less convenient than atomic write but more
// universal. Use SafeWrite with rename=false for files that may be managed
// directly by the user, like settings.json and bindings.json.
func SafeWrite(path string, bytes []byte, rename bool) error {
	var err error
	if _, err = os.Stat(path); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		// Force rename for new files!
		rename = true
	}

	var file *os.File
	if !rename {
		file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE, FileMode)
		if err != nil {
			return err
		}
		defer file.Close()
	}

	tmp := path + BackupSuffix
	err = os.WriteFile(tmp, bytes, FileMode)
	if err != nil {
		os.Remove(tmp)
		return err
	}

	if rename {
		err = os.Rename(tmp, path)
	} else {
		err = file.Truncate(0)
		if err == nil {
			_, err = file.Write(bytes)
		}
		if err == nil {
			err = file.Sync()
		}
	}
	if err != nil {
		if rename {
			os.Remove(tmp)
		} else {
			err = OverwriteError{err, tmp}
		}
		return err
	}

	if !rename {
		os.Remove(tmp)
	}
	return nil
}
