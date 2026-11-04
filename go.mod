module github.com/micro-editor/micro/v2

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/gdamore/tcell/v3 v3.4.0
	github.com/go-errors/errors v1.0.1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mattn/go-isatty v0.0.20
	github.com/mattn/go-runewidth v0.0.16
	github.com/micro-editor/json5 v1.0.1-micro
	github.com/micro-editor/terminal v0.0.0-20250324214352-e587e959c6b5
	github.com/mitchellh/go-homedir v1.1.0
	github.com/sergi/go-diff v1.1.0
	github.com/stretchr/testify v1.4.0
	github.com/yuin/gopher-lua v1.1.1
	github.com/zyedidia/clipper v0.1.1
	github.com/zyedidia/glob v0.0.0-20170209203856-dd4023a66dc3
	golang.org/x/text v0.38.0
	gopkg.in/yaml.v2 v2.2.8
	layeh.com/gopher-luar v1.0.11
)

require (
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/creack/pty v1.1.18 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/term v0.44.0 // indirect
)

replace github.com/kballard/go-shellquote => github.com/micro-editor/go-shellquote v0.0.0-20250101105543-feb6c39314f5

replace layeh.com/gopher-luar v1.0.11 => github.com/layeh/gopher-luar v1.0.11

replace github.com/gdamore/tcell/v3 => github.com/micro-editor/tcell/v3 v3.0.0-20260629163637-dedaf40206a5

go 1.25.0
