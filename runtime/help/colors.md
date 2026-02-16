# Colors

This help page aims to cover two aspects of micro's syntax highlighting engine:

* How to create colorschemes and use them.
* How to create syntax files to add to the list of languages micro can
  highlight.

## Colorschemes

To change your colorscheme, press `Ctrl-e` in micro to bring up the command
prompt, and type:

```
set colorscheme twilight
```

(or whichever colorscheme you choose).

Micro comes with a number of colorschemes by default. The colorschemes that you
can display will depend on what kind of color support your terminal has.

Omit color-link default "[fg color],[bg color]" will make the background color match the terminal's, and transparency if set.

Modern terminals tend to have a palette of 16 user-configurable colors (these
colors can often be configured in the terminal preferences), and additional
color support comes in three flavors.

* 16-color: A colorscheme that uses the 16 default colors will always work but
  will only look good if the 16 default colors have been configured to the
  user's liking. Using a colorscheme that only uses the 16 colors from the
  terminal palette will also preserve the terminal's theme from other
  applications since the terminal will often use those same colors for other
  applications. Default colorschemes of this type include `simple` and
  `solarized`.

* 256-color: Almost all terminals support displaying an additional 240 colors
  on top of the 16 user-configurable colors (creating 256 colors total).
  Colorschemes which use 256-color are portable because they will look the
  same regardless of the configured 16-color palette. However, the color
  range is fairly limited due to the small number of colors available.
  Default 256-color colorschemes include `monokai`, `twilight`, `zenburn`,
  `darcula` and more.

* true-color: Some terminals support displaying "true color" with 16 million
  colors using standard RGB values. This mode will be able to support
  displaying any colorscheme, but it should be noted that the user-configured
  16-color palette is ignored when using true-color mode (this means the
  colors while using the terminal emulator will be slightly off). Not all
  terminals support true color but at this point most do (see below).
  True-color colorschemes in micro typically end with `-tc`, such as
  `solarized-tc`, `atom-dark`, `material-tc`, etc... If true color is not
  enabled but a true color colorscheme is used, micro will do its best to
  approximate the colors to the available 256 colors.

Here is the list of colorschemes:

### 256 color

These should work and look nice in most terminals. I recommend these
themes the most.

* `monokai` (also the `default` colorscheme)
* `zenburn`
* `gruvbox`
* `darcula`
* `twilight`
* `railscast`
* `bubblegum` (light theme)

### 16 color

These may vary widely based on the 16 colors selected for your terminal.

* `simple`
* `solarized` (must have the solarized color palette in your terminal to use
   this colorscheme properly)
* `cmc-16`
* `cmc-paper`
* `geany`

### True color

Micro enables true color support by default as long as it detects that the
terminal supports it (which is usually indicated by the environment variable
`COLORTERM` being set to `truecolor`, `24bit` or `24-bit`). You can also force
enabling it unconditionally by setting the option `truecolor` to `on` (or
alternatively by setting the environment variable `MICRO_TRUECOLOR` to 1, which
is supported for backward compatibility).

* `solarized-tc`: this is the solarized colorscheme for true color.
* `atom-dark`: this colorscheme is based off of Atom's "dark" colorscheme.
* `cmc-tc`: A true colour variant of the cmc theme.  It requires true color to
   look its best. Use cmc-16 if your terminal doesn't support true color.
* `gruvbox-tc`: The true color version of the gruvbox colorscheme
* `material-tc`: Colorscheme based off of Google's Material Design palette

## Creating a Colorscheme

Micro's colorschemes are also extremely simple to create. The default ones can
be found
[here](https://github.com/micro-editor/micro/tree/master/runtime/colorschemes).

Custom colorschemes should be placed in the `~/.config/micro/colorschemes`
directory.

A number of custom directives are placed in a `.micro` file. Colorschemes are
typically only 18-30 lines in total.

To create the colorscheme you need to link highlight groups with
actual colors. This is done using the `color-link` command.

For example, to highlight all comments in green, you would use the command:

```
color-link comment "green"
```

Background colors can also be specified with a comma:

```
color-link comment "green,blue"
```

This will give the comments a blue background.

If you would like no foreground you can just use a comma with nothing in front:

```
color-link comment ",blue"
```

You can also put bold, italic, or underline in front of the color:

```
color-link comment "bold red"
```

---

There are three different ways to specify the color.

Color terminals usually have 16 colors that are preset by the user. This means
that you cannot depend on those colors always being the same. You can use those
colors with the names `black, red, green, yellow, blue, magenta, cyan, white`
and the bright variants of each one (brightblack, brightred...).

Then you can use the terminals 256 colors by using their numbers 1-256 (numbers
1-16 will refer to the named colors).

If the user's terminal supports true color, then you can also specify colors
exactly using their hex codes. If the terminal is not true color but micro is
told to use a true color colorscheme it will attempt to map the colors to the
available 256 colors.

Generally colorschemes which require true color terminals to look good are
marked with a `-tc` suffix and colorschemes which supply a white background are
marked with a `-paper` suffix.

---

Here is a list of the colorscheme groups that you can use:

* default (color of the background and foreground for unhighlighted text)
* comment
* identifier
* constant
* statement
* symbol
* preproc
* type
* special
* underlined
* error
* todo
* selection (Color of the text selection)
* statusline (Color of the statusline)
* statusline.inactive (Color of the statusline of inactive split panes)
* statusline.suggestions (Color of the autocomplete suggestions menu)
* tabbar (Color of the tabbar that lists open files)
* tabbar.active (Color of the active tab in the tabbar)
* indent-char (Color of the character which indicates tabs if the option is
  enabled)
* line-number
* gutter-info
* gutter-error
* gutter-warning
* diff-added
* diff-modified
* diff-deleted
* cursor-line
* current-line-number
* color-column
* ignore
* scrollbar
* divider (Color of the divider between vertical splits)
* message (Color of messages in the bottom line of the screen)
* error-message (Color of error messages in the bottom line of the screen)
* match-brace (Color of matching brackets when `matchbracestyle` is set to `highlight`)
* hlsearch (Color of highlighted search results when `hlsearch` is enabled)
* hlsearch.selection (Color of the selection within the highlighted search results)
* tab-error (Color of tab vs space errors when `hltaberrors` is enabled)
* trailingws (Color of trailing whitespaces when `hltrailingws` is enabled)

Colorschemes must be placed in the `~/.config/micro/colorschemes` directory to
be used.

---

In addition to the main colorscheme groups, there are subgroups that you can
specify by adding `.subgroup` to the group. If you're creating your own custom
syntax files, you can make use of your own subgroups.

If micro can't match the subgroup, it'll default to the root group, so  it's
safe and recommended to use subgroups in your custom syntax files.

For example if `constant.string` is found in your colorscheme, micro will us
that for highlighting strings. If it's not found, it will use constant instead.
Micro tries to match the largest set of groups it can find in the colorscheme
definitions, so if, for example `constant.bool.true` is found then micro will
use that. If `constant.bool.true` is not found but `constant.bool` is found
micro will use `constant.bool`. If not, it uses `constant`.

Here's a list of subgroups used in micro's built-in syntax files.

* comment.bright (Some filetypes have distinctions between types of comments)
* constant.bool
* constant.bool.true
* constant.bool.false
* constant.number
* constant.specialChar
* constant.string
* constant.string.url
* identifier.class (Also used for functions)
* identifier.macro
* identifier.var
* preproc.shebang (The #! at the beginning of a file that tells the os what
  script interpreter to use)
* symbol.brackets (`{}()[]` and sometimes `<>`)
* symbol.operator (Color operator symbols differently)
* symbol.tag (For html tags, among other things)
* type.keyword (If you want a special highlight for keywords like `private`)

In the future, plugins may also be able to use color groups for styling.

---

Last but not least it's even possible to use `include` followed by the
colorscheme name as string to include a different colorscheme within a new one.
Additionally the groups can then be extended or overwritten. The `default.micro`
theme can be seen as an example, which links to the chosen default colorscheme.

## Syntax files

The syntax files are written in yaml-format and specify how to highlight
languages.

Micro's builtin syntax highlighting tries very hard to be sane, sensible and
provide ample coverage of the meaningful elements of a language. Micro has
syntax files built in for over 100 languages now! However, there may be
situations where you find Micro's highlighting to be insufficient or not to
your liking. The good news is that you can create your own syntax files, and
place them in  `~/.config/micro/syntax` and Micro will use those instead.

### Filetype definition

You must start the syntax file by declaring the filetype:

```
filetype: go
```

### Detect definition

Then you must provide information about how to detect the filetype:

```
detect:
    filename: "\\.go$"
```

Micro will match this regex against a given filename to detect the filetype.

In addition to the `filename` regex (or even instead of it) you can provide
a `header` regex that will check the first line of the file. For example:

```
detect:
    filename: "\\.ya?ml$"
    header: "%YAML"
```

This is useful in cases when the given file name is not sufficient to determine
the filetype, e.g. with the above example, if a YAML file has no `.yaml`
extension but may contain a `%YAML` directive in its first line.

`filename` takes precedence over `header`, i.e. if there is a syntax file that
matches the file with a filetype by the `filename` and another syntax file that
matches the same file with another filetype by the `header`, the first filetype
will be used.

Finally, in addition to `filename` and/or `header` (but not instead of them)
you may also provide an optional `signature` regex which is useful for resolving
ambiguities when there are multiple syntax files matching the same file with
different filetypes. If a `signature` regex is given, micro will match a certain
amount of first lines in the file (this amount is determined by the `detectlimit`
option) against this regex, and if any of the lines match, this syntax file's
filetype will be preferred over other matching filetypes.

For example, to distinguish C++ header files from C and Objective-C header files
that have the same `.h` extension:

```
detect:
    filename: "\\.c(c|pp|xx)$|\\.h(h|pp|xx)?$"
    signature: "namespace|template|public|protected|private"
```

### Syntax rules

Next you must provide the syntax highlighting rules. There are two types of
rules: patterns and regions. A pattern is matched on a single line and usually
a single word as well. A region highlights between two patterns over multiple
lines and may have rules of its own inside the region.

Here are some example patterns in Go:

```
rules:
    - special: "\\b(break|case|continue|default|go|goto|range|return)\\b"
    - statement: "\\b(else|for|if|switch)\\b"
    - preproc: "\\b(package|import|const|var|type|struct|func|go|defer|iota)\\b"
```

The order of patterns does matter as patterns lower in the file will overwrite
the ones defined above them.

And here are some example regions for Go:

```
- constant.string:
    start: "\""
    end: "\""
    rules:
        - constant.specialChar: "%."
        - constant.specialChar: "\\\\[abfnrtv'\\\"\\\\]"
        - constant.specialChar: "\\\\([0-7]{3}|x[A-Fa-f0-9]{2}|u[A-Fa-f0-9]{4}|U[A-Fa-f0-9]{8})"

- comment:
    start: "//"
    end: "$"
    rules:
        - todo: "(TODO|XXX|FIXME):?"

- comment:
    start: "/\\*"
    end: "\\*/"
    rules:
        - todo: "(TODO|XXX|FIXME):?"
```

Notice how the regions may contain rules inside of them. Any inner rules that
are matched are then skipped when searching for the end of the region. For
example, when highlighting `"foo \" bar"`, since `\"` is matched by an inner
rule in the region, it is skipped. Likewise for `"foo \\" bar`, since `\\` is
matched by an inner rule, it is skipped, and then the `"` is found and the
string ends at the correct place.

You may also explicitly mark skip regexes if you don't want them to be
highlighted. For example:

```
- constant.string:
    start: "\""
    end: "\""
    skip: "\\."
```

#### Includes

You may also include rules from other syntax files as embedded languages. For
example, the following is possible for html:

```
- default:
    start: "<script.*?>"
    end: "</script.*?>"
    rules:
        - include: "javascript"

- default:
    start: "<style.*?>"
    end: "</style.*?>"
    rules:
        - include: "css"
```

Note that nested include (i.e. including syntax files that include other syntax
files) is not supported yet.

### Default syntax highlighting

If micro cannot detect the filetype of the file, it falls back to using the
default syntax highlighting for it, which highlights just the bare minimum:
email addresses, URLs etc.

Just like in other cases, you can override the default highlighting by adding
your own custom `default.yaml` file to `~/.config/micro/syntax`.

For example, if you work with various config files that use the `#` sign to mark
the beginning of a comment, you can use the following custom `default.yaml` to
highlight those comments by default:

```
filetype: unknown

detect:
    filename: ""

rules:
    - comment: "(^|\\s)#.*$"
```
