package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/micro-editor/json5"
	"github.com/micro-editor/micro/v2/internal/util"
	"github.com/zyedidia/glob"
	"golang.org/x/text/encoding/htmlindex"
)

// NOTE: '--' interprets rest of arguments as arguments for generate_settings.go
//go:generate $GOROOT/bin/go run ../../tools/generate_settings.go -- $GOFILE ../../runtime/help/options.md

type optionValidator func(string, any) error

// a list of settings that need option validators
var optionValidators = map[string]optionValidator{
	"autosave":        validateNonNegativeValue,
	"clipboard":       validateChoice,
	"colorcolumn":     validateNonNegativeValue,
	"colorscheme":     validateColorscheme,
	"detectlimit":     validateNonNegativeValue,
	"encoding":        validateEncoding,
	"fileformat":      validateChoice,
	"helpsplit":       validateChoice,
	"matchbracestyle": validateChoice,
	"multiopen":       validateChoice,
	"pageoverlap":     validateNonNegativeValue,
	"reload":          validateChoice,
	"scrollmargin":    validateNonNegativeValue,
	"scrollspeed":     validateNonNegativeValue,
	"tabsize":         validatePositiveValue,
	"truecolor":       validateChoice,
}

// a list of settings with pre-defined choices
var OptionChoices = map[string][]string{
	"clipboard": {
		// micro will use an internal clipboard.
		"internal",
		// accesses clipboard via an external tool, such as xclip/xsel
		// or wl-clipboard on Linux, pbcopy/pbpaste on MacOS, and system calls on
		// Windows. On Linux, if you do not have one of the tools installed, or if
		// they are not working, micro will throw an error and use an internal
		// clipboard.
		"external",
		// accesses the clipboard via your terminal emulator. Note that
		// there is limited support among terminal emulators for this feature
		// (called OSC 52). Terminals that are known to work are Kitty (enable
		// reading with `clipboard_control` setting), iTerm2 (only copying),
		// st, rxvt-unicode and xterm if enabled (see `> help copypaste` for
		// details). Note that Gnome-terminal does not support this feature. With
		// this setting, copy-paste **will** work over ssh. See `> help copypaste`
		// for details.
		"terminal",
	},
	"fileformat": {
		// default on Unix systems
		"unix",
		// default on Windows
		"dos",
	},
	"helpsplit": {
		// open help in a horizontal split pane
		"hsplit",
		// open help in a vertical split pane
		"vsplit",
	},
	"matchbracestyle": {
		// underline matching braces.
		"underline",
		// use `match-brace` style from the current theme.
		"highlight",
	},
	"multiopen":       {
		// open each file in a separate tab.
		"tab",
		// open files stacked top to bottom.
		"hsplit",
		// open files side-by-side.
		"vsplit",
	},
	"reload":          {"prompt", "auto", "disabled"},
	"truecolor": {
		// enable usage of true color if micro detects that it is supported by
		// the terminal, otherwise disable it.
		"auto",
		// force usage of true color even if micro does not detect its support
		// by the terminal (of course this is not guaranteed to work well unless the
		// terminal actually supports true color).
		"on",
		// disable true color usage.
		"off",
	},
}

// a list of settings that can be globally and locally modified and their
// default values
var defaultCommonSettings = map[string]any{
	// when creating a new line, use the same indentation as the previous line.
	"autoindent":      true,
	// When a file is saved that the user doesn't have permission to
	// modify, micro will ask if the user would like to use super user
	// privileges to save the file. If this option is enabled, micro will
	// automatically attempt to use super user privileges to save without
	// asking the user.
	"autosu":          false,
	// micro will automatically keep backups of all open buffers. Backups
	// are stored in `~/.config/micro/backups` and are removed when the buffer is
	// closed cleanly. In the case of a system crash or a micro crash, the contents
	// of the buffer can be recovered automatically by opening the file that was
	// being edited before the crash, or manually by searching for the backup in
	// the backup directory. Backups are made in the background for newly modified
	// buffers every 8 seconds, or when micro detects a crash.
	"backup":          true,
	// the directory micro should place backups in. For the default
	// value of `""` (empty string), the backup directory will be
	// `ConfigDir/backups`, which is `~/.config/micro/backups` by default. The
	// directory specified for backups will be created if it does not exist.
	"backupdir":       "",
	// in the infobar and tabbar, show only the basename of the file
	// being edited rather than the full path.
	"basename":        false,
	// if this is not set to 0, it will display a column at the
	// specified column. This is useful if you want column 80 to be highlighted
	// special for example.
	"colorcolumn":     float64(0),
	// highlight the line that the cursor is on in a different color
	// (the color is defined by the colorscheme you are using).
	"cursorline":      true,
	// if this is not set to 0, it will limit the amount of first
	// lines in a file that are matched to determine the filetype.
	// A higher limit means better accuracy of guessing the filetype, but also
	// taking more time.
	"detectlimit":     float64(100),
	// display diff indicators before lines.
	"diffgutter":      false,
	// the encoding to open and save files with. Supported encodings
	// are listed at https://www.w3.org/TR/encoding/.
	"encoding":        "utf-8",
	// micro will automatically add a newline to the end of the
	// file if one does not exist.
	"eofnewline":      true,
	// this determines what kind of algorithm micro uses to determine
	// if a buffer is modified or not. When `fastdirty` is on, micro just uses a
	// boolean `modified` that is set to `true` as soon as the user makes an edit.
	// This is fast, but can be inaccurate. If `fastdirty` is off, then micro will
	// hash the current buffer against a hash of the original file (created when
	// the buffer was loaded). This is more accurate but obviously more resource
	// intensive. This option will be automatically enabled for the current buffer
	// if the file size exceeds 50KB.
	"fastdirty":       false,
	// this determines what kind of line endings micro will use for
	// the file. Unix line endings are just `\n` (linefeed) whereas dos line
	// endings are `\r\n` (carriage return + linefeed). The two possible values for
	// this option are `unix` and `dos`. The fileformat will be automatically
	// detected (when you open an existing file) and displayed on the statusline,
	// but this option is useful if you would like to change the line endings or if
	// you are starting a new file. Changing this option while editing a file will
	// change its line endings. Opening a file with this option set will only have
	// an effect if the file is empty/newly created, because otherwise the fileformat
	// will be automatically detected from the existing line endings.
	"fileformat":      defaultFileFormat(),
	// sets the filetype for the current buffer. Set this option to
	// `off` to completely disable filetype detection.
	// The default value will be automatically overridden depending on the file you open.
	"filetype":        "unknown",
	// highlight all instances of the searched text after a successful
	// search. This highlighting can be temporarily turned off via the
	// `UnhighlightSearch` action (triggered by the Esc key by default) or toggled
	// on/off via the `ToggleHighlightSearch` action. Note that these actions don't
	// change the `hlsearch` setting. As long as `hlsearch` is set to true, the next
	// search will have the highlighting turned on again.
	"hlsearch":        false,
	// highlight tabs when spaces are expected, and spaces when tabs
	// are expected. More precisely: if `tabstospaces` option is on, highlight
	// all tab characters; if `tabstospaces` is off, highlight space characters
	// in the initial indent part of the line.
	"hltaberrors":     false,
	// highlight trailing whitespaces at ends of lines. Note that
	// it doesn't highlight newly added trailing whitespaces that naturally occur
	// while typing text. It highlights only nasty forgotten trailing whitespaces.
	"hltrailingws":    false,
	// perform case-insensitive searches.
	"ignorecase":      true,
	// enable incremental search in "Find" prompt (matching as you type).
	"incsearch":       true,
	// sets the character to be shown to display tab characters.
	// This option is **deprecated**, use the `tab` key in `showchars` option instead.
	"indentchar":      " ", // Deprecated
	// when using autoindent, whitespace is added for you. This
	// option determines if when you move to the next line without any insertions
	// the whitespace that was added should be deleted to remove trailing
	// whitespace. By default, the autoindent whitespace is deleted if the line
	// was left empty.
	"keepautoindent":  false,
	// show matching braces for '()', '{}', '[]' when the cursor
	// is on a brace character or (if `matchbraceleft` is enabled) next to it.
	"matchbrace":      true,
	// simulate I-beam cursor behavior (cursor located not on a
	// character but "between" characters): when showing matching braces, if there
	// is no brace character directly under the cursor, match the brace character
	// to the left of the cursor instead. Also when jumping to the matching brace,
	// move the cursor either to the matching brace character or to the character
	// next to it, depending on whether the initial cursor position was on the
	// brace character or next to it (i.e. "inside" or "outside" the braces).
	// With `matchbraceleft` disabled, micro will only match the brace directly
	// under the cursor and will only jump to precisely to the matching brace.
	"matchbraceleft":  true,
	// whether to underline or highlight matching braces when
	// `matchbrace` is enabled. The color of highlight is determined by the `match-brace`
	// field in the current theme.
	"matchbracestyle": "underline",
	// if a file is opened on a path that does not exist, the file
	// cannot be saved because the parent directories don't exist. This option lets
	// micro automatically create the parent directories in such a situation.
	"mkparents":       false,
	// the number of lines from the current view to keep in view
	// when paging up or down. If this is set to 2, for instance, and you page
	// down, the last two lines of the previous page will be the first two lines
	// of the next page.
	"pageoverlap":     float64(2),
	// this option causes backups (see `backup` option) to be
	// permanently saved. With permanent backups, micro will not remove backups when
	// files are closed and will never apply them to existing files. Use this option
	// if you are interested in manually managing your backup files.
	"permbackup":      false,
	// when enabled, disallows edits to the buffer. It is recommended
	// to only ever set this option locally using `setlocal`.
	"readonly":        false,
	// make line numbers display relatively. If set to true, all
	// lines except for the line that the cursor is located will display the distance
	// from the cursor's line.
	"relativeruler":   false,
	// controls the reload behavior of the current buffer in case the file
	// has changed.
	"reload":          "prompt",
	// micro will automatically trim trailing whitespaces at ends of
	// lines.
	// Note: This setting overrides `keepautoindent` and isn't used at timed `autosave`
	// or forced `autosave` in case the buffer didn't change. A manual save will
	// involve the action regardless if the buffer has been changed or not.
	"rmtrailingws":    false,
	// display line numbers.
	"ruler":           true,
	// remember where the cursor was last time the file was opened and
	// put it there when you open the file again. Information is saved to
	// `~/.config/micro/buffers/`
	"savecursor":      false,
	// when this option is on, undo is saved even after you close a file
	// so if you close and reopen a file, you can keep undoing. Information is
	// saved to `~/.config/micro/buffers/`.
	"saveundo":        false,
	// display a scroll bar
	"scrollbar":       false,
	// margin at which the view starts scrolling when the cursor
	// approaches the edge of the view.
	"scrollmargin":    float64(3),
	// amount of lines to scroll for one scroll event.
	"scrollspeed":     float64(2),
	// sets what characters to be shown to display various invisible
	// characters in the file. The characters shown will not be inserted into files.
	// This option is specified in the form of `key1=value1,key2=value2,...`.
	//
	// Here are the list of keys:
	// - `space`: space characters
	// - `tab`: tab characters. If set, overrides the `indentchar` option.
	// - `ispace`: space characters at indent position before the first visible
	// 			   character in a line. If this is not set, `space` will be shown
	// 			   instead.
	// - `itab`: tab characters before the first visible character in a line.
	//			 If this is not set, `tab` will be shown instead.
	//
	// Only `tab` and `itab` can display multiple characters (if possible),
	// otherwise only the first character will be displayed.
	//
	// An example of this option value could be `tab=>,space=.,itab=|>,ispace=|`
	//
	// The color of the shown character is determined by the `indent-char`
	// field in the current theme rather than the default text color.
	"showchars":       "",
	// add leading whitespace when pasting multiple lines.
	// This will attempt to preserve the current indentation level when pasting an
	// unindented block.
	"smartpaste":      true,
	// wrap lines that are too long to fit on the screen.
	"softwrap":        false,
	// when a horizontal split is created, create it below the
	// current split.
	"splitbottom":     true,
	// when a vertical split is created, create it to the right of the
	// current split.
	"splitright":      true,
	// format string definition for the left-justified part of the
	// statusline. Special directives should be placed inside `$()`. Special
	// directives include: `filename`, `modified`, `line`, `col`, `lines`,
	// `percentage`, `opt`, `overwrite`, `bind`.
	// The `opt` and `bind` directives take either an option or an action afterward
	// and fill in the value of the option or the key bound to the action.
	"statusformatl":   "$(filename) $(modified)$(overwrite)($(line),$(col)) $(status.paste)| ft:$(opt:filetype) | $(opt:fileformat) | $(opt:encoding)",
	// format string definition for the right-justified part of the
	// statusline.
	"statusformatr":   "$(bind:ToggleKeyMenu): bindings, $(bind:ToggleHelp): help",
	// display the status line at the bottom of the screen.
	"statusline":      true,
	// enables syntax highlighting.
	"syntax":          true,
	// navigate spaces at the beginning of lines as if they are tabs
	// (e.g. move over 4 spaces at once). This option only does anything if
	// `tabstospaces` is on.
	"tabmovement":     false,
	// the size in spaces that a tab character should be displayed with.
	"tabsize":         float64(4),
	// use spaces instead of tabs. Note: This option will be
	// overridden by [the `ftoptions` plugin](https://github.com/micro-editor/micro/blob/master/runtime/plugins/ftoptions/ftoptions.lua)
	// for certain filetypes. To disable this behavior, add `"ftoptions": false` to
	// your config. See [issue #2213](https://github.com/micro-editor/micro/issues/2213)
	// for more details.
	"tabstospaces":    false,
	// controls whether micro will use true colors (24-bit colors) when
	// using a colorscheme with true colors, such as `solarized-tc` or `atom-dark`.
	// Note: The change will take effect after the next start of `micro`.
	"truecolor":       "auto",
	// (only useful on unix) defines whether or not micro will use the
	// primary clipboard to copy selections in the background. This does not affect
	// the normal clipboard using `Ctrl-c` and `Ctrl-v`.
	"useprimary":      true,
	// wrap long lines by words, i.e. break at spaces. This option
	// only does anything if `softwrap` is on.
	"wordwrap":        false,
}

// a list of settings that should only be globally modified and their
// default values
var DefaultGlobalOnlySettings = map[string]any{
	// automatically save the buffer every n seconds, where n is the
	// value of the autosave option. Also when quitting on a modified buffer, micro
	// will automatically save and quit. Be warned, this option saves the buffer
	// without prompting the user, so data may be overwritten. If this option is
	// set to `0`, no autosaving is performed.
	"autosave":       float64(0),
	// specifies how micro should access the system clipboard.
	"clipboard":      "external",
	// use the given colorscheme. This setting is `global only`.
	// The colorscheme can be either one of the colorschemes that micro comes with
	// by default (such as `default`, `solarized` or `solarized-tc`) which are
	// embedded in the micro binary, or a custom colorscheme stored in
	// `~/.config/micro/colorschemes/$(option).micro` where `$(option)` is the
	// option value. You can read more about micro's colorschemes and see the list
	// of default colorschemes in `> help colors`.
	"colorscheme":    "default",
	// specifies the "divider" characters used for the dividing line
	// between vertical/horizontal splits. The first character is for vertical
	// dividers, and the second is for horizontal dividers. By default, for
	// horizontal splits the statusline serves as a divider, but if the statusline
	// is disabled the horizontal divider character will be used.
	"divchars":       "|-",
	// colorschemes provide the color (foreground and background) for
	// the characters displayed in split dividers. With this option enabled, the
	// colors specified by the colorscheme will be reversed (foreground and
	// background colors swapped).
	"divreverse":     true,
	// forces micro to render the cursor using terminal colors rather
	// than the actual terminal cursor. This is useful when the terminal's cursor is
	// slow or otherwise unavailable/undesirable to use.
	// Note: This option defaults to `true` in case `micro` is used in the legacy
	// Windows Console.
	"fakecursor":     defaultFakeCursor(),
	// sets the split type to be used by the `help` command.
	"helpsplit":      "hsplit",
	// enables the line at the bottom of the editor where messages are
	// printed. This option is `global only`.
	"infobar":        true,
	// display the nano-style key menu at the bottom of the screen. Note
	// that ToggleKeyMenu is bound to `Alt-g` by default and this is displayed in
	// the statusline. To disable the key binding, bind `Alt-g` to `None`.
	"keymenu":        false,
	// prevent plugins and lua scripts from binding any keys.
	// Any custom actions must be binded manually either via commands like `bind`
	// or by modifying the `bindings.json` file.
	"lockbindings":   false,
	// mouse support. When mouse support is disabled,
	// usually the terminal will be able to access mouse events which can be useful
	// if you want to copy from the terminal instead of from micro (if over ssh for
	// example, because the terminal has access to the local clipboard and micro
	// does not).
	"mouse":          true,
	// specifies how to layout multiple files opened at startup.
	// Most useful as a command-line option, like `-multiopen vsplit`. Possible
	// values correspond to commands (see `> help commands`) that open files:
	"multiopen":      "tab",
	// if enabled, this will cause micro to parse filenames such as
	// `file.txt:10:5` as requesting to open `file.txt` with the cursor at line 10
	// and column 5. The column number can also be dropped to open the file at a
	// given line and column 0. Note that with this option enabled it is not possible
	// to open a file such as `file.txt:10:5`, where `:10:5` is part of the filename.
	// It is also possible to open a file with a certain cursor location by using the
	// `+LINE:COL` flag syntax. See `micro -help` for the command line options.
	"parsecursor":    false,
	// treat characters sent from the terminal in a single chunk as a paste
	// event rather than a series of manual key presses. If you are pasting using
	// the terminal keybinding (not `Ctrl-v`, which is micro's default paste
	// keybinding) then it is a good idea to enable this option during the paste
	// and disable once the paste is over. See `> help copypaste` for details about
	// copying and pasting in a terminal environment.
	"paste":          false,
	// list of URLs pointing to plugin channels for downloading and
	// installing plugins. A plugin channel consists of a json file with links to
	// plugin repos, which store information about plugin versions and download URLs.
	// By default, this option points to the official plugin channel hosted on GitHub
	// at https://github.com/micro-editor/plugin-channel.
	"pluginchannels": []string{"https://raw.githubusercontent.com/micro-editor/plugin-channel/master/channel.json"},
	// a list of links to plugin repositories.
	"pluginrepos":    []string{},
	// remember command history between closing and re-opening
	// micro. Information is saved to `~/.config/micro/buffers/history`.
	"savehistory":    true,
	// specifies the character used for displaying the scrollbar
	"scrollbarchar":  "|",
	// specifies the super user command. On most systems this is "sudo" but
	// on BSD it can be "doas." This option can be customized and is only used when
	// saving with su.
	"sucmd":          "sudo",
	// always shows the tab bar, even when only one tab is open.
	"tabalways":      false,
	// inverts the tab characters' (filename, save indicator, etc)
	// colors with respect to the tab bar.
	"tabhighlight":   false,
	// reverses the tab bar colors when active.
	"tabreverse":     true,
	// micro will assume that the terminal it is running in conforms to
	// `xterm-256color` regardless of what the `$TERM` variable actually contains.
	// Enabling this option may cause unwanted effects if your terminal in fact
	// does not conform to the `xterm-256color` standard.
	"xterm":          false,
}

// a list of settings that should never be globally modified
var LocalSettings = []string{
	"filetype",
	"readonly",
}

var (
	ErrInvalidOption    = errors.New("Invalid option")
	ErrInvalidValue     = errors.New("Invalid value")
	ErrOptNotToggleable = errors.New("Option not toggleable")

	// The options that the user can set
	GlobalSettings map[string]any

	// This is the raw parsed json
	parsedSettings     map[string]any
	settingsParseError bool

	// ModifiedSettings is a map of settings which should be written to disk
	// because they have been modified by the user in this session
	ModifiedSettings map[string]bool

	// VolatileSettings is a map of settings which should not be written to disk
	// because they have been temporarily set for this session only
	VolatileSettings map[string]bool
)

func writeFile(name string, txt []byte) error {
	return util.SafeWrite(name, txt, false)
}

func init() {
	ModifiedSettings = make(map[string]bool)
	VolatileSettings = make(map[string]bool)
}

func validateParsedSettings() error {
	var err error
	defaults := DefaultAllSettings()
	for k, v := range parsedSettings {
		if strings.HasPrefix(reflect.TypeOf(v).String(), "map") {
			if strings.HasPrefix(k, "ft:") {
				for k1, v1 := range v.(map[string]any) {
					if _, ok := defaults[k1]; ok {
						if e := verifySetting(k1, v1, defaults[k1]); e != nil {
							err = e
							parsedSettings[k].(map[string]any)[k1] = defaults[k1]
							continue
						}
					}
				}
			} else {
				tk := strings.TrimPrefix(k, "glob:")
				if _, e := glob.Compile(tk); e != nil {
					err = errors.New("Error with glob setting " + tk + ": " + e.Error())
					delete(parsedSettings, k)
					continue
				}
				if !strings.HasPrefix(k, "glob:") {
					// Support non-prefixed glob settings but internally convert
					// them to prefixed ones for simplicity.
					delete(parsedSettings, k)
					k = "glob:" + k
					parsedSettings[k] = v
				}
				for k1, v1 := range v.(map[string]any) {
					if _, ok := defaults[k1]; ok {
						if e := verifySetting(k1, v1, defaults[k1]); e != nil {
							err = e
							parsedSettings[k].(map[string]any)[k1] = defaults[k1]
							continue
						}
					}
				}
			}
			continue
		}

		if k == "autosave" {
			// if autosave is a boolean convert it to float
			s, ok := v.(bool)
			if ok {
				if s {
					parsedSettings["autosave"] = 8.0
				} else {
					parsedSettings["autosave"] = 0.0
				}
			}
			continue
		}

		if _, ok := defaults[k]; ok {
			if e := verifySetting(k, v, defaults[k]); e != nil {
				err = e
				parsedSettings[k] = defaults[k]
				continue
			}
		}
	}
	return err
}

func ReadSettings() error {
	parsedSettings = make(map[string]any)
	filename := filepath.Join(ConfigDir, "settings.json")
	if _, e := os.Stat(filename); e == nil {
		input, err := os.ReadFile(filename)
		if err != nil {
			settingsParseError = true
			return errors.New("Error reading settings.json file: " + err.Error())
		}
		if !strings.HasPrefix(string(input), "null") {
			// Unmarshal the input into the parsed map
			err = json5.Unmarshal(input, &parsedSettings)
			if err != nil {
				settingsParseError = true
				return errors.New("Error reading settings.json: " + err.Error())
			}
			err = validateParsedSettings()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ParsedSettings() map[string]any {
	s := make(map[string]any)
	for k, v := range parsedSettings {
		if strings.HasPrefix(reflect.TypeOf(v).String(), "map") {
			continue
		}
		s[k] = v
	}
	return s
}

func verifySetting(option string, value any, def any) error {
	var interfaceArr []any
	valType := reflect.TypeOf(value)
	defType := reflect.TypeOf(def)
	assignable := false

	switch option {
	case "pluginrepos", "pluginchannels":
		assignable = valType.AssignableTo(reflect.TypeOf(interfaceArr))
	default:
		assignable = defType.AssignableTo(valType)
	}
	if !assignable {
		return fmt.Errorf("Error: setting '%s' has incorrect type (%s), using default value: %v (%s)", option, valType, def, defType)
	}

	if option == "colorscheme" {
		// Plugins are not initialized yet, so do not verify if the colorscheme
		// exists yet, since the colorscheme may be added by a plugin later.
		return nil
	}

	if err := OptionIsValid(option, value); err != nil {
		return err
	}

	return nil
}

// InitGlobalSettings initializes the options map and sets all options to their default values
// Must be called after ReadSettings
func InitGlobalSettings() error {
	var err error
	GlobalSettings = DefaultAllSettings()

	for k, v := range parsedSettings {
		if !strings.HasPrefix(reflect.TypeOf(v).String(), "map") {
			GlobalSettings[k] = v
		}
	}
	return err
}

// UpdatePathGlobLocals scans the already parsed settings and sets the options locally
// based on whether the path matches a glob
// Must be called after ReadSettings
func UpdatePathGlobLocals(settings map[string]any, path string) {
	for k, v := range parsedSettings {
		if strings.HasPrefix(reflect.TypeOf(v).String(), "map") && strings.HasPrefix(k, "glob:") {
			tk := strings.TrimPrefix(k, "glob:")
			g, _ := glob.Compile(tk)
			if g.MatchString(path) {
				for k1, v1 := range v.(map[string]any) {
					settings[k1] = v1
				}
			}
		}
	}
}

// UpdateFileTypeLocals scans the already parsed settings and sets the options locally
// based on whether the filetype matches to "ft:"
// Must be called after ReadSettings
func UpdateFileTypeLocals(settings map[string]any, filetype string) {
	for k, v := range parsedSettings {
		if strings.HasPrefix(reflect.TypeOf(v).String(), "map") && strings.HasPrefix(k, "ft:") {
			if filetype == k[3:] {
				for k1, v1 := range v.(map[string]any) {
					if k1 != "filetype" {
						settings[k1] = v1
					}
				}
			}
		}
	}
}

// WriteSettings writes the settings to the specified filename as JSON
func WriteSettings(filename string) error {
	if settingsParseError {
		// Don't write settings if there was a parse error
		// because this will delete the settings.json if it
		// is invalid. Instead we should allow the user to fix
		// it manually.
		return nil
	}

	var err error
	if _, e := os.Stat(ConfigDir); e == nil {
		defaults := DefaultAllSettings()

		// remove any options froms parsedSettings that have since been marked as default
		for k, v := range parsedSettings {
			if !strings.HasPrefix(reflect.TypeOf(v).String(), "map") {
				cur, okcur := GlobalSettings[k]
				_, vol := VolatileSettings[k]
				if def, ok := defaults[k]; ok && okcur && !vol && reflect.DeepEqual(cur, def) {
					delete(parsedSettings, k)
				}
			}
		}

		// add any options to parsedSettings that have since been marked as non-default
		for k, v := range GlobalSettings {
			if def, ok := defaults[k]; !ok || !reflect.DeepEqual(v, def) {
				if _, wr := ModifiedSettings[k]; wr {
					parsedSettings[k] = v
				}
			}
		}

		txt, _ := json.MarshalIndent(parsedSettings, "", "    ")
		txt = append(txt, '\n')
		err = writeFile(filename, txt)
	}
	return err
}

// OverwriteSettings writes the current settings to settings.json and
// resets any user configuration of local settings present in settings.json
func OverwriteSettings(filename string) error {
	settings := make(map[string]any)

	var err error
	if _, e := os.Stat(ConfigDir); e == nil {
		defaults := DefaultAllSettings()
		for k, v := range GlobalSettings {
			if def, ok := defaults[k]; !ok || !reflect.DeepEqual(v, def) {
				if _, wr := ModifiedSettings[k]; wr {
					settings[k] = v
				}
			}
		}

		txt, _ := json.MarshalIndent(parsedSettings, "", "    ")
		txt = append(txt, '\n')
		err = writeFile(filename, txt)
	}
	return err
}

// RegisterCommonOptionPlug creates a new option (called pl.name). This is meant to be called by plugins to add options.
func RegisterCommonOptionPlug(pl string, name string, defaultvalue any) error {
	return RegisterCommonOption(pl+"."+name, defaultvalue)
}

// RegisterGlobalOptionPlug creates a new global-only option (named pl.name)
func RegisterGlobalOptionPlug(pl string, name string, defaultvalue any) error {
	return RegisterGlobalOption(pl+"."+name, defaultvalue)
}

// RegisterCommonOption creates a new option
func RegisterCommonOption(name string, defaultvalue any) error {
	if _, ok := GlobalSettings[name]; !ok {
		GlobalSettings[name] = defaultvalue
	}
	defaultCommonSettings[name] = defaultvalue
	return nil
}

// RegisterGlobalOption creates a new global-only option
func RegisterGlobalOption(name string, defaultvalue any) error {
	if _, ok := GlobalSettings[name]; !ok {
		GlobalSettings[name] = defaultvalue
	}
	DefaultGlobalOnlySettings[name] = defaultvalue
	return nil
}

// GetGlobalOption returns the global value of the given option
func GetGlobalOption(name string) any {
	return GlobalSettings[name]
}

func defaultFileFormat() string {
	if runtime.GOOS == "windows" {
		return "dos"
	}
	return "unix"
}

func defaultFakeCursor() bool {
	_, wt := os.LookupEnv("WT_SESSION")
	if runtime.GOOS == "windows" && !wt {
		// enabled for windows consoles where the cursor is slow
		return true
	}
	return false
}

func GetInfoBarOffset() int {
	offset := 0
	if GetGlobalOption("infobar").(bool) {
		offset++
	}
	if GetGlobalOption("keymenu").(bool) {
		offset += 2
	}
	return offset
}

// DefaultCommonSettings returns a map of all common buffer settings
// and their default values
func DefaultCommonSettings() map[string]any {
	commonsettings := make(map[string]any)
	for k, v := range defaultCommonSettings {
		commonsettings[k] = v
	}
	return commonsettings
}

// DefaultAllSettings returns a map of all common buffer & global-only settings
// and their default values
func DefaultAllSettings() map[string]any {
	allsettings := make(map[string]any)
	for k, v := range defaultCommonSettings {
		allsettings[k] = v
	}
	for k, v := range DefaultGlobalOnlySettings {
		allsettings[k] = v
	}
	return allsettings
}

// GetNativeValue parses and validates a value for a given option
func GetNativeValue(option, value string) (any, error) {
	curVal := GetGlobalOption(option)
	if curVal == nil {
		return nil, ErrInvalidOption
	}

	switch kind := reflect.TypeOf(curVal).Kind(); kind {
	case reflect.Bool:
		b, err := util.ParseBool(value)
		if err != nil {
			return nil, ErrInvalidValue
		}
		return b, nil
	case reflect.String:
		return value, nil
	case reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, ErrInvalidValue
		}
		return f, nil
	default:
		return nil, ErrInvalidValue
	}
}

// OptionIsValid checks if a value is valid for a certain option
func OptionIsValid(option string, value any) error {
	if validator, ok := optionValidators[option]; ok {
		return validator(option, value)
	}

	return nil
}

// Option validators

func validatePositiveValue(option string, value any) error {
	nativeValue, ok := value.(float64)

	if !ok {
		return errors.New("Expected numeric type for " + option)
	}

	if nativeValue < 1 {
		return errors.New(option + " must be greater than 0")
	}

	return nil
}

func validateNonNegativeValue(option string, value any) error {
	nativeValue, ok := value.(float64)

	if !ok {
		return errors.New("Expected numeric type for " + option)
	}

	if nativeValue < 0 {
		return errors.New(option + " must be non-negative")
	}

	return nil
}

func validateChoice(option string, value any) error {
	if choices, ok := OptionChoices[option]; ok {
		val, ok := value.(string)
		if !ok {
			return errors.New("Expected string type for " + option)
		}

		for _, v := range choices {
			if val == v {
				return nil
			}
		}

		choicesStr := strings.Join(choices, ", ")
		return errors.New(option + " must be one of: " + choicesStr)
	}

	return errors.New("Option has no pre-defined choices")
}

func validateColorscheme(option string, value any) error {
	colorscheme, ok := value.(string)

	if !ok {
		return errors.New("Expected string type for colorscheme")
	}

	if !ColorschemeExists(colorscheme) {
		return errors.New(colorscheme + " is not a valid colorscheme")
	}

	return nil
}

func validateEncoding(option string, value any) error {
	_, err := htmlindex.Get(value.(string))
	return err
}
