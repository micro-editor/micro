# Options

Micro stores all of the user configuration in its configuration directory.

Micro uses `$MICRO_CONFIG_HOME` as the configuration directory. If this
environment variable is not set, it uses `$XDG_CONFIG_HOME/micro` instead. If
that environment variable is not set, it uses `~/.config/micro` as the
configuration directory. In the documentation, we use `~/.config/micro` to
refer to the configuration directory (even if it may in fact be somewhere else
if you have set either of the above environment variables).

Here are the available options:

* `autoindent`: when creating a new line, use the same indentation as the
   previous line.

    default value: `true`

* `autosave`: automatically save the buffer every n seconds, where n is the
   value of the autosave option. Also when quitting on a modified buffer, micro
   will automatically save and quit. Be warned, this option saves the buffer
   without prompting the user, so data may be overwritten. If this option is
   set to `0`, no autosaving is performed.

    default value: `0`

* `autosu`: When a file is saved that the user doesn't have permission to
   modify, micro will ask if the user would like to use super user
   privileges to save the file. If this option is enabled, micro will
   automatically attempt to use super user privileges to save without
   asking the user.

    default value: `false`

* `backup`: micro will automatically keep backups of all open buffers. Backups
   are stored in `~/.config/micro/backups` and are removed when the buffer is
   closed cleanly. In the case of a system crash or a micro crash, the contents
   of the buffer can be recovered automatically by opening the file that was
   being edited before the crash, or manually by searching for the backup in
   the backup directory. Backups are made in the background for newly modified
   buffers every 8 seconds, or when micro detects a crash.

    default value: `true`

* `backupdir`: the directory micro should place backups in. For the default
   value of `""` (empty string), the backup directory will be
   `ConfigDir/backups`, which is `~/.config/micro/backups` by default. The
   directory specified for backups will be created if it does not exist.

    default value: `""` (empty string)

* `basename`: in the infobar and tabbar, show only the basename of the file
   being edited rather than the full path.

    default value: `false`

* `clipboard`: specifies how micro should access the system clipboard.
   Possible values are:
    * `external`: accesses clipboard via an external tool, such as xclip/xsel
       or wl-clipboard on Linux, pbcopy/pbpaste on MacOS, and system calls on
       Windows. On Linux, if you do not have one of the tools installed, or if
       they are not working, micro will throw an error and use an internal
       clipboard.
    * `terminal`: accesses the clipboard via your terminal emulator. Note that
       there is limited support among terminal emulators for this feature
       (called OSC 52). Terminals that are known to work are Kitty (enable
       reading with `clipboard_control` setting), iTerm2 (only copying),
       st, rxvt-unicode and xterm if enabled (see `> help copypaste` for
       details). Note that Gnome-terminal does not support this feature. With
       this setting, copy-paste **will** work over ssh. See `> help copypaste`
       for details.
    * `internal`: micro will use an internal clipboard.

    default value: `external`

* `colorcolumn`: if this is not set to 0, it will display a column at the
   specified column. This is useful if you want column 80 to be highlighted
   special for example.

    default value: `0`

* `colorscheme`: use the given colorscheme. This setting is `global only`.
   The colorscheme can be either one of the colorschemes that micro comes with
   by default (such as `default`, `solarized` or `solarized-tc`) which are
   embedded in the micro binary, or a custom colorscheme stored in
   `~/.config/micro/colorschemes/$(option).micro` where `$(option)` is the
   option value. You can read more about micro's colorschemes and see the list
   of default colorschemes in `> help colors`.

    default value: `default`

* `cursorline`: highlight the line that the cursor is on in a different color
   (the color is defined by the colorscheme you are using).

    default value: `true`

* `detectlimit`: if this is not set to 0, it will limit the amount of first
   lines in a file that are matched to determine the filetype.
   A higher limit means better accuracy of guessing the filetype, but also
   taking more time.

   default value: `100`

* `diffgutter`: display diff indicators before lines.

    default value: `false`

* `divchars`: specifies the "divider" characters used for the dividing line
   between vertical/horizontal splits. The first character is for vertical
   dividers, and the second is for horizontal dividers. By default, for
   horizontal splits the statusline serves as a divider, but if the statusline
   is disabled the horizontal divider character will be used.

    default value: `|-`

* `divreverse`: colorschemes provide the color (foreground and background) for
   the characters displayed in split dividers. With this option enabled, the
   colors specified by the colorscheme will be reversed (foreground and
   background colors swapped).

    default value: `true`

* `encoding`: the encoding to open and save files with. Supported encodings
   are listed at https://www.w3.org/TR/encoding/.

    default value: `utf-8`

* `eofnewline`: micro will automatically add a newline to the end of the
   file if one does not exist.

    default value: `true`

* `fakecursor`: forces micro to render the cursor using terminal colors rather
   than the actual terminal cursor. This is useful when the terminal's cursor is
   slow or otherwise unavailable/undesirable to use.
   Note: This option defaults to `true` in case `micro` is used in the legacy
   Windows Console.

    default value: `false`

* `fastdirty`: this determines what kind of algorithm micro uses to determine
   if a buffer is modified or not. When `fastdirty` is on, micro just uses a
   boolean `modified` that is set to `true` as soon as the user makes an edit.
   This is fast, but can be inaccurate. If `fastdirty` is off, then micro will
   hash the current buffer against a hash of the original file (created when
   the buffer was loaded). This is more accurate but obviously more resource
   intensive. This option will be automatically enabled for the current buffer
   if the file size exceeds 50KB.

    default value: `false`

* `fileformat`: this determines what kind of line endings micro will use for
   the file. Unix line endings are just `\n` (linefeed) whereas dos line
   endings are `\r\n` (carriage return + linefeed). The two possible values for
   this option are `unix` and `dos`. The fileformat will be automatically
   detected (when you open an existing file) and displayed on the statusline,
   but this option is useful if you would like to change the line endings or if
   you are starting a new file. Changing this option while editing a file will
   change its line endings. Opening a file with this option set will only have
   an effect if the file is empty/newly created, because otherwise the fileformat
   will be automatically detected from the existing line endings.

    default value: `unix` on Unix systems, `dos` on Windows

* `filetype`: sets the filetype for the current buffer. Set this option to
   `off` to completely disable filetype detection.

    default value: `unknown`. This will be automatically overridden depending
    on the file you open.

* `helpsplit`: sets the split type to be used by the `help` command.
   Possible values:
    * `vsplit`: open help in a vertical split pane
    * `hsplit`: open help in a horizontal split pane

    default value: `hsplit`

* `hlsearch`: highlight all instances of the searched text after a successful
   search. This highlighting can be temporarily turned off via the
   `UnhighlightSearch` action (triggered by the Esc key by default) or toggled
   on/off via the `ToggleHighlightSearch` action. Note that these actions don't
   change the `hlsearch` setting. As long as `hlsearch` is set to true, the next
   search will have the highlighting turned on again.

    default value: `false`

* `hltaberrors`: highlight tabs when spaces are expected, and spaces when tabs
   are expected. More precisely: if `tabstospaces` option is on, highlight
   all tab characters; if `tabstospaces` is off, highlight space characters
   in the initial indent part of the line.

    default value: `false`

* `hltrailingws`: highlight trailing whitespaces at ends of lines. Note that
   it doesn't highlight newly added trailing whitespaces that naturally occur
   while typing text. It highlights only nasty forgotten trailing whitespaces.

    default value: `false`

* `ignorecase`: perform case-insensitive searches.

    default value: `true`

* `incsearch`: enable incremental search in "Find" prompt (matching as you type).

    default value: `true`

* `indentchar`: sets the character to be shown to display tab characters.
   This option is **deprecated**, use the `tab` key in `showchars` option instead.

    default value: ` ` (space)

* `infobar`: enables the line at the bottom of the editor where messages are
   printed. This option is `global only`.

    default value: `true`

* `keepautoindent`: when using autoindent, whitespace is added for you. This
   option determines if when you move to the next line without any insertions
   the whitespace that was added should be deleted to remove trailing
   whitespace. By default, the autoindent whitespace is deleted if the line
   was left empty.

    default value: `false`

* `keymenu`: display the nano-style key menu at the bottom of the screen. Note
   that ToggleKeyMenu is bound to `Alt-g` by default and this is displayed in
   the statusline. To disable the key binding, bind `Alt-g` to `None`.

    default value: `false`

* `lockbindings`: prevent plugins and lua scripts from binding any keys.
   Any custom actions must be binded manually either via commands like `bind`
   or by modifying the `bindings.json` file.

    default value: `false`

* `matchbrace`: show matching braces for '()', '{}', '[]' when the cursor
   is on a brace character or (if `matchbraceleft` is enabled) next to it.

    default value: `true`

* `matchbraceleft`: simulate I-beam cursor behavior (cursor located not on a
   character but "between" characters): when showing matching braces, if there
   is no brace character directly under the cursor, match the brace character
   to the left of the cursor instead. Also when jumping to the matching brace,
   move the cursor either to the matching brace character or to the character
   next to it, depending on whether the initial cursor position was on the
   brace character or next to it (i.e. "inside" or "outside" the braces).
   With `matchbraceleft` disabled, micro will only match the brace directly
   under the cursor and will only jump to precisely to the matching brace.

    default value: `true`

* `matchbracestyle`: whether to underline or highlight matching braces when
   `matchbrace` is enabled. The color of highlight is determined by the `match-brace`
   field in the current theme. Possible values:
    * `underline`: underline matching braces.
    * `highlight`: use `match-brace` style from the current theme.

    default value: `underline`

* `mkparents`: if a file is opened on a path that does not exist, the file
   cannot be saved because the parent directories don't exist. This option lets
   micro automatically create the parent directories in such a situation.

    default value: `false`

* `mouse`: mouse support. When mouse support is disabled,
   usually the terminal will be able to access mouse events which can be useful
   if you want to copy from the terminal instead of from micro (if over ssh for
   example, because the terminal has access to the local clipboard and micro
   does not).

    default value: `true`

* `multiopen`: specifies how to layout multiple files opened at startup.
   Most useful as a command-line option, like `-multiopen vsplit`. Possible
   values correspond to commands (see `> help commands`) that open files:
    * `tab`: open each file in a separate tab.
    * `vsplit`: open files side-by-side.
    * `hsplit`: open files stacked top to bottom.

    default value: `tab`

* `pageoverlap`: the number of lines from the current view to keep in view
   when paging up or down. If this is set to 2, for instance, and you page
   down, the last two lines of the previous page will be the first two lines
   of the next page.

    default value: `2`

* `parsecursor`: if enabled, this will cause micro to parse filenames such as
   `file.txt:10:5` as requesting to open `file.txt` with the cursor at line 10
   and column 5. The column number can also be dropped to open the file at a
   given line and column 0. Note that with this option enabled it is not possible
   to open a file such as `file.txt:10:5`, where `:10:5` is part of the filename.
   It is also possible to open a file with a certain cursor location by using the
   `+LINE:COL` flag syntax. See `micro -help` for the command line options.

    default value: `false`

* `paste`: treat characters sent from the terminal in a single chunk as a paste
   event rather than a series of manual key presses. If you are pasting using
   the terminal keybinding (not `Ctrl-v`, which is micro's default paste
   keybinding) then it is a good idea to enable this option during the paste
   and disable once the paste is over. See `> help copypaste` for details about
   copying and pasting in a terminal environment.

    default value: `false`

* `permbackup`: this option causes backups (see `backup` option) to be
   permanently saved. With permanent backups, micro will not remove backups when
   files are closed and will never apply them to existing files. Use this option
   if you are interested in manually managing your backup files.

    default value: `false`

* `pluginchannels`: list of URLs pointing to plugin channels for downloading and
   installing plugins. A plugin channel consists of a json file with links to
   plugin repos, which store information about plugin versions and download URLs.
   By default, this option points to the official plugin channel hosted on GitHub
   at https://github.com/micro-editor/plugin-channel.

    default value: `https://raw.githubusercontent.com/micro-editor/plugin-channel/master/channel.json`

* `pluginrepos`: a list of links to plugin repositories.

    default value: ``

* `readonly`: when enabled, disallows edits to the buffer. It is recommended
   to only ever set this option locally using `setlocal`.

    default value: `false`

* `relativeruler`: make line numbers display relatively. If set to true, all
   lines except for the line that the cursor is located will display the distance
   from the cursor's line.

    default value: `false`

* `reload`: controls the reload behavior of the current buffer in case the file
   has changed. The available options are `prompt`, `auto` & `disabled`.

   default value: `prompt`

* `rmtrailingws`: micro will automatically trim trailing whitespaces at ends of
   lines.
   Note: This setting overrides `keepautoindent` and isn't used at timed `autosave`
   or forced `autosave` in case the buffer didn't change. A manual save will
   involve the action regardless if the buffer has been changed or not.

    default value: `false`

* `ruler`: display line numbers.

    default value: `true`

* `savecursor`: remember where the cursor was last time the file was opened and
   put it there when you open the file again. Information is saved to
   `~/.config/micro/buffers/`

    default value: `false`

* `savehistory`: remember command history between closing and re-opening
   micro. Information is saved to `~/.config/micro/buffers/history`.

    default value: `true`

* `saveundo`: when this option is on, undo is saved even after you close a file
   so if you close and reopen a file, you can keep undoing. Information is
   saved to `~/.config/micro/buffers/`.

    default value: `false`

* `scrollbar`: display a scroll bar

    default value: `false`

* `scrollbarchar`: specifies the character used for displaying the scrollbar

    default value: `|`

* `scrollmargin`: margin at which the view starts scrolling when the cursor
   approaches the edge of the view.

    default value: `3`

* `scrollspeed`: amount of lines to scroll for one scroll event.

    default value: `2`

* `showchars`: sets what characters to be shown to display various invisible
   characters in the file. The characters shown will not be inserted into files.
   This option is specified in the form of `key1=value1,key2=value2,...`.

   Here are the list of keys:
   - `space`: space characters
   - `tab`: tab characters. If set, overrides the `indentchar` option.
   - `ispace`: space characters at indent position before the first visible
               character in a line. If this is not set, `space` will be shown
               instead.
   - `itab`: tab characters before the first visible character in a line.
             If this is not set, `tab` will be shown instead.

   Only `tab` and `itab` can display multiple characters (if possible),
   otherwise only the first character will be displayed.

   An example of this option value could be `tab=>,space=.,itab=|>,ispace=|`

   The color of the shown character is determined by the `indent-char`
   field in the current theme rather than the default text color.

    default value: ``

* `smartpaste`: add leading whitespace when pasting multiple lines.
   This will attempt to preserve the current indentation level when pasting an
   unindented block.

    default value: `true`

* `softwrap`: wrap lines that are too long to fit on the screen.

    default value: `false`

* `splitbottom`: when a horizontal split is created, create it below the
   current split.

    default value: `true`

* `splitright`: when a vertical split is created, create it to the right of the
   current split.

    default value: `true`

* `statusformatl`: format string definition for the left-justified part of the
   statusline. Special directives should be placed inside `$()`. Special
   directives include: `filename`, `modified`, `line`, `col`, `lines`,
   `percentage`, `opt`, `overwrite`, `bind`.
   The `opt` and `bind` directives take either an option or an action afterward
   and fill in the value of the option or the key bound to the action.

    default value: `$(filename) $(modified)$(overwrite)($(line),$(col)) $(status.paste)|
                    ft:$(opt:filetype) | $(opt:fileformat) | $(opt:encoding)`

* `statusformatr`: format string definition for the right-justified part of the
   statusline.

    default value: `$(bind:ToggleKeyMenu): bindings, $(bind:ToggleHelp): help`

* `statusline`: display the status line at the bottom of the screen.

    default value: `true`

* `sucmd`: specifies the super user command. On most systems this is "sudo" but
   on BSD it can be "doas." This option can be customized and is only used when
   saving with su.

    default value: `sudo`

* `syntax`: enables syntax highlighting.

    default value: `true`

* `tabhighlight`: inverts the tab characters' (filename, save indicator, etc)
   colors with respect to the tab bar.

    default value: `false`

* `tabmovement`: navigate spaces at the beginning of lines as if they are tabs
   (e.g. move over 4 spaces at once). This option only does anything if
   `tabstospaces` is on.

    default value: `false`

* `tabreverse`: reverses the tab bar colors when active.

    default value: `true`

* `tabsize`: the size in spaces that a tab character should be displayed with.

    default value: `4`

* `tabstospaces`: use spaces instead of tabs. Note: This option will be
   overridden by [the `ftoptions` plugin](https://github.com/micro-editor/micro/blob/master/runtime/plugins/ftoptions/ftoptions.lua)
   for certain filetypes. To disable this behavior, add `"ftoptions": false` to
   your config. See [issue #2213](https://github.com/micro-editor/micro/issues/2213)
   for more details.

    default value: `false`

* `truecolor`: controls whether micro will use true colors (24-bit colors) when
   using a colorscheme with true colors, such as `solarized-tc` or `atom-dark`.
   * `auto`: enable usage of true color if micro detects that it is supported by
      the terminal, otherwise disable it.
   * `on`: force usage of true color even if micro does not detect its support
      by the terminal (of course this is not guaranteed to work well unless the
      terminal actually supports true color).
   * `off`: disable true color usage.

   Note: The change will take effect after the next start of `micro`.

   default value: `auto`

* `useprimary` (only useful on unix): defines whether or not micro will use the
   primary clipboard to copy selections in the background. This does not affect
   the normal clipboard using `Ctrl-c` and `Ctrl-v`.

    default value: `true`

* `wordwrap`: wrap long lines by words, i.e. break at spaces. This option
   only does anything if `softwrap` is on.

    default value: `false`

* `xterm`: micro will assume that the terminal it is running in conforms to
  `xterm-256color` regardless of what the `$TERM` variable actually contains.
   Enabling this option may cause unwanted effects if your terminal in fact
   does not conform to the `xterm-256color` standard.

    default value: `false`

---

Plugin options: all plugins come with a special option to enable or disable
them. The option is a boolean with the same name as the plugin itself.

By default, the following plugins are provided, each with an option to enable
or disable them:

* `autoclose`: automatically closes brackets, quotes, etc...
* `comment`: provides automatic commenting for a number of languages
* `ftoptions`: alters some default options depending on the filetype
* `linter`: provides extensible linting for many languages
* `literate`: provides advanced syntax highlighting for the Literate
   programming tool.
* `status`: provides some extensions to the status line (integration with
   Git and more).
* `diff`: integrates the `diffgutter` option with Git. If you are in a Git
   directory, the diff gutter will show changes with respect to the most
   recent Git commit rather than the diff since opening the file.

Any option you set in the editor will be saved to the file
`~/.config/micro/settings.json` so, in effect, your configuration file will be
created for you. If you'd like to take your configuration with you to another
machine, simply copy the `settings.json` to the other machine.

## Settings.json file

The `settings.json` file should go in your configuration directory (by default
at `~/.config/micro`), and should contain only options which have been modified
from their default setting. Here is the full list of options in json format,
so that you can see what the formatting should look like.

```json
{
    "autoclose": true,
    "autoindent": true,
    "autosave": 0,
    "autosu": false,
    "backup": true,
    "backupdir": "",
    "basename": false,
    "clipboard": "external",
    "colorcolumn": 0,
    "colorscheme": "default",
    "comment": true,
    "cursorline": true,
    "detectlimit": 100,
    "diff": true,
    "diffgutter": false,
    "divchars": "|-",
    "divreverse": true,
    "encoding": "utf-8",
    "eofnewline": true,
    "fakecursor": false,
    "fastdirty": false,
    "fileformat": "unix",
    "filetype": "unknown",
    "ftoptions": true,
    "helpsplit": "hsplit",
    "hlsearch": false,
    "hltaberrors": false,
    "hltrailingws": false,
    "ignorecase": true,
    "incsearch": true,
    "indentchar": " ",
    "infobar": true,
    "initlua": true,
    "keepautoindent": false,
    "keymenu": false,
    "linter": true,
    "literate": true,
    "matchbrace": true,
    "matchbraceleft": true,
    "matchbracestyle": "underline",
    "mkparents": false,
    "mouse": true,
    "multiopen": "tab",
    "pageoverlap": 2,
    "parsecursor": false,
    "paste": false,
    "permbackup": false,
    "pluginchannels": [
        "https://raw.githubusercontent.com/micro-editor/plugin-channel/master/channel.json"
    ],
    "pluginrepos": [],
    "readonly": false,
    "relativeruler": false,
    "reload": "prompt",
    "rmtrailingws": false,
    "ruler": true,
    "savecursor": false,
    "savehistory": true,
    "saveundo": false,
    "scrollbar": false,
    "scrollbarchar": "|",
    "scrollmargin": 3,
    "scrollspeed": 2,
    "showchars": "",
    "smartpaste": true,
    "softwrap": false,
    "splitbottom": true,
    "splitright": true,
    "status": true,
    "statusformatl": "$(filename) $(modified)$(overwrite)($(line),$(col)) $(status.paste)| ft:$(opt:filetype) | $(opt:fileformat) | $(opt:encoding)",
    "statusformatr": "$(bind:ToggleKeyMenu): bindings, $(bind:ToggleHelp): help",
    "statusline": true,
    "sucmd": "sudo",
    "syntax": true,
    "tabhighlight": true,
    "tabmovement": false,
    "tabreverse": false,
    "tabalwaysshow": false,
    "tabsize": 4,
    "tabstospaces": false,
    "useprimary": true,
    "wordwrap": false,
    "xterm": false
}
```

## Global and local settings

You can set these settings either globally or locally. Locally means that the
setting won't be saved to `~/.config/micro/settings.json` and that it will only
be set in the current buffer. Setting an option globally is the default, and
will set the option in all buffers. Use the `setlocal` command to set an option
locally rather than globally.

The `colorscheme` option is global only, and the `filetype` option is local
only. To set an option locally, use `setlocal` instead of `set`.

In the `settings.json` file you can also put set options locally by specifying
either a glob or a filetype. Here is an example which has `tabstospaces` on for
all files except Go files, and `tabsize` 4 for all files except Ruby files:

```json
{
    "ft:go": {
        "tabstospaces": false
    },
    "ft:ruby": {
        "tabsize": 2
    },
    "tabstospaces": true,
    "tabsize": 4
}
```

Or similarly you can match with globs:

```json
{
    "*.go": {
        "tabstospaces": false
    },
    "*.rb": {
        "tabsize": 2
    },
    "tabstospaces": true,
    "tabsize": 4
}
```
