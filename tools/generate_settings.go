package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
)

type optionScope uint
const (
	scopeUnset      optionScope = iota // uninitialized value
	scopeCommon                        // Can use either set or setlocal
	scopeOnlyGlobal                    // Only can use set (editor scope)
	scopeOnlyLocal                     // Only can use setlocal (buffer scope)
)

var scopeString = [4]string{
	scopeUnset:      "ScopeUnset",
	scopeCommon:     "ScopeCommon",
	scopeOnlyGlobal: "ScopeOnlyGlobal",
	scopeOnlyLocal:  "ScopeOnlyLocal",
}

type choice struct {
	name    string
	comment string
}

type option struct {
	name    string
	comment string

	typ          string // (string, bool, float, int, array ...)
	defaultValue any    // (string, bool, float, int, array ...)

	scope   optionScope
	choices []choice
	filled  bool // found the option in `defaultCommonSettings` or `DefaultGlobalOnlySettings`.
 				 // Extra measure to be sure.
				 // set to true @ collectOptionsFromMapString().
				 // NOTE: `option.filled` should be done in the last top level node parsed
}

func (o option) writeMarkdown(sb *strings.Builder) {
	if sb == nil { panic("`sb` MUST not be nil") }

	sb.WriteString(fmt.Sprintf("* `%s`: %s\n", o.name, o.comment))

	if len(o.choices) > 0 {
		sb.WriteString("   Possible values are:\n\n")
		for _, choice := range o.choices {
			if choice.comment == "" {
				sb.WriteString(fmt.Sprintf("   * `%s`\n", choice.name))
			} else {
				sb.WriteString(fmt.Sprintf("   * `%s`: %s", choice.name, choice.comment))
			}
		}

		sb.WriteString("\n")
	}

	if o.typ == "bool" || o.typ == "float" || o.typ == "string" || o.typ == "[]string" {
		sb.WriteString(fmt.Sprintf("    default value: `%s`\n\n", o.defaultValue))
	} else {
		panic("This type was not checked if converts to string without issue")
	}
}

// function for debugging purposes
func (o option) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("name: %s {\n", o.name))
	sb.WriteString(fmt.Sprintf("\ttype: \033[31m%s\033[0m,\n", o.typ))
	sb.WriteString(fmt.Sprintf("\tdefault: %v,\n", o.defaultValue))
	sb.WriteString(fmt.Sprintf("\tcomment: \033[33m%s\033[0m,\n", strings.TrimRight(o.comment, "\n")))
	sb.WriteString(fmt.Sprintf("\tscope: \033[34m%s\033[0m,\n", scopeString[o.scope]))

	if len(o.choices) > 0 {
		sb.WriteString(fmt.Sprintf("\tchoices: "))
		for _, choice := range o.choices {
			sb.WriteString(fmt.Sprintf("\"%s\", ", choice.name))
		}
		sb.WriteString(fmt.Sprintf("\n"))
	}

	sb.WriteString(fmt.Sprintf("\tfilled: %v,\n", o.filled))
	sb.WriteString("}")
	return sb.String()
}

func commentGroupToString(cgs []*ast.CommentGroup) string {
	var sb strings.Builder
	for _, cg := range cgs {
		for i, comment := range cg.List {
			linecomment := strings.TrimPrefix(comment.Text, "//")
			linecomment = strings.TrimLeft(linecomment, " ")
			if i > 0 { sb.WriteString("   ") } // indentation used in markdown
			sb.WriteString(linecomment)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// Returns a reference to the option and if it was created or not
func findOrCreateOption(optname string, opts *[]option) (*option, bool) {
	if opts == nil { panic("`opts` MUST NOT be nil") }

	for i := 0; i < len(*opts); i++ {
		if optname == (*opts)[i].name { return &(*opts)[i], false }
	}

	*opts = append(*opts, option{name: optname, scope: scopeUnset, filled: false})
	return &(*opts)[len(*opts)-1], true
}

func collectOptionsFromMapString(mapkv *ast.CompositeLit, fset *token.FileSet, comments ast.CommentMap, opts []option, scope optionScope) []option {
	if scope != scopeCommon && scope != scopeOnlyGlobal {
		panic("This functions should only be used for: `defaultCommonSettings` or `DefaultGlobalOnlySettings`")
	}

	// ast.Print(fset, mapkv.Elts)
	for _, elem := range mapkv.Elts {
		kv, ok := elem.(*ast.KeyValueExpr)
		if !ok { panic("Should be an `ast.KeyValueExpr`") }
		// ast.Print(fset, kv)

		key, ok := kv.Key.(*ast.BasicLit)
		if !ok { panic("Should be an `ast.BasicLit`") }

		optionComment := comments[kv]
		// NOTE: enforce all options in `defaultCommonSettings` and
		// `DefaultGlobalOnlySettings` have a comment.
		if optionComment == nil {
			fmt.Fprintf(os.Stderr, "\033[31mERROR: All options MUST have a description\n\033[0m")
			fmt.Fprintf(os.Stderr, "\tMissing description for %s @ %v\n",
				key.Value, fset.Position(key.ValuePos))
			os.Exit(1)
		}

		optname := strings.Trim(key.Value, "\"")
		opt, created := findOrCreateOption(optname, &opts)

		// NOTE: Option cannot exist from other scope or be already filled
		// we set both in this function.
		if !created && (opt.scope != scopeUnset || opt.filled) {
			fmt.Fprintf(os.Stderr, "\033[31mERROR: option '%s' @ %v already present from a wrong scope (%s) or filled\n\033[0m", opt.name, fset.Position(key.ValuePos), scopeString[opt.scope])
			os.Exit(1)
		}

		opt.comment = commentGroupToString(optionComment)

		switch val := kv.Value.(type) {
		case *ast.Ident:
			name := val.Name
			if name == "true" || name == "false" {
				opt.typ = "bool"
				opt.defaultValue = name
			} else {
				ast.Print(fset, val)
				panic("Unexpected value")
			}

		case *ast.BasicLit:
			if val.Kind == token.STRING {
				opt.typ = "string"
				opt.defaultValue = val.Value
			} else {
				ast.Print(fset, val)
				panic("Unexpected value")
			}

		case *ast.CallExpr:
			// ast.Print(fset, val)
			function, ok := val.Fun.(*ast.Ident)
			if !ok { panic("Expected `*ast.Ident`") }

			if function.Name == "float64" {
				if 1 != len(val.Args) { panic("Expects 1") }
				uniqArg, ok := val.Args[0].(*ast.BasicLit)
				if !ok { panic("Expected `*ast.BasicLit`") }
				if uniqArg.Kind != token.INT { panic("Expected `token.INT`") }
				opt.typ = "float"
				opt.defaultValue = uniqArg.Value

			} else if function.Name == "defaultFileFormat" { // NOTE: Special runtime case
				opt.typ = "string"
				opt.defaultValue = "\"unix\""

			} else if function.Name == "defaultFakeCursor" { // NOTE: Special runtime case
				opt.typ = "bool"
				opt.defaultValue = "false"

			} else {
				ast.Print(fset, val)
				panic("Unhandled CallExpr")
			}

		case *ast.CompositeLit:
			array, ok := val.Type.(*ast.ArrayType)
			if !ok { panic("Expected only array types") }
			arrayType, ok := array.Elt.(*ast.Ident)
			if !ok { panic("Expected `*ast.Ident` as Name for `*ast.ArrayType`") }
			opt.typ = fmt.Sprintf("[]%s", arrayType.Name)

			var strValue []string
			for _, elem := range val.Elts {
				switch e := elem.(type) {
				case *ast.BasicLit:
					strValue = append(strValue, e.Value)
				default:
					ast.Print(fset, elem)
					panic("Unhandled element type in *ast.ArrayType")
				}
			}
			opt.defaultValue = strValue

		default:
			ast.Print(fset, val)
			panic("Unhandled type for map value")
		}

		opt.scope = scope
		opt.filled = true
	}

	return opts
}

func collectOptionChoicesFromMapString(mapkv *ast.CompositeLit, fset *token.FileSet, comments ast.CommentMap, opts []option) []option {
	// ast.Print(fset, mapkv.Elts)
	for _, elem := range mapkv.Elts {
		kv, ok := elem.(*ast.KeyValueExpr)
		if !ok { panic("Should be an `ast.KeyValueExpr`") }
		// ast.Print(fset, kv)

		key, ok := kv.Key.(*ast.BasicLit)
		if !ok { panic("Should be an `ast.BasicLit`") }

		optname := strings.Trim(key.Value, "\"")
		opt, _ := findOrCreateOption(optname, &opts)

		// Get the choices
		switch val := kv.Value.(type) {
		case *ast.CompositeLit: // OptionChoices is a map with value type []string
			for _, elem := range val.Elts {
				switch e := elem.(type) {
				case *ast.BasicLit:
					choiceName := strings.Trim(e.Value, "\"")
					choiceComment := comments[e]
					if choiceComment == nil {
						fmt.Fprintf(os.Stderr, "\033[33mWARN: no description for choice %s @ %v\n\033[0m",
							choiceName, fset.Position(e.ValuePos))
					}
					opt.choices = append(opt.choices, choice{name: choiceName, comment: commentGroupToString(choiceComment)})

				default:
					ast.Print(fset, elem)
					panic("Unreachable: expected only `string`")
				}
			}

		default:
			ast.Print(fset, val)
			panic("Unreachable: `OptionChoices` is a map with value type []string")
		}

	}

	return opts
}

func collectOptions(fset *token.FileSet, astFile *ast.File, comments ast.CommentMap) []option {
	var options = make([]option, 0, 128)

	for _, decl := range astFile.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			continue
		case *ast.GenDecl:
			// ast.Print(fset, d)

			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					if 1 < len(s.Names) { panic("only one name was expected") }
					name := s.Names[0].Name // NOTE: only care for first one

					// All the variables with information are all maps (for now).
					unwrapCompositeLitFromValueSpec := func(value *ast.ValueSpec) *ast.CompositeLit {
						if 1 != len(value.Values) { panic("Should be 1 map") }
						maplit, ok := value.Values[0].(*ast.CompositeLit)
						if !ok { panic("Should be an `ast.CompositeLit`") }
						return maplit
					}

					if name == "OptionChoices" {
						fmt.Fprintf(os.Stderr, "Processing: %s\n", name)
						mapstring := unwrapCompositeLitFromValueSpec(s)
						options = collectOptionChoicesFromMapString(mapstring, fset, comments, options)

					} else if name == "defaultCommonSettings" {
						fmt.Fprintf(os.Stderr, "Processing: %s\n", name)
						mapstring := unwrapCompositeLitFromValueSpec(s)
						options = collectOptionsFromMapString(mapstring, fset, comments, options, scopeCommon)
					} else if name == "DefaultGlobalOnlySettings" {
						fmt.Fprintf(os.Stderr, "Processing: %s\n", name)
						mapstring := unwrapCompositeLitFromValueSpec(s)
						options = collectOptionsFromMapString(mapstring, fset, comments, options, scopeOnlyGlobal)

					} else if name == "LocalSettings" {
						fmt.Fprintf(os.Stderr, "Processing: %s\n", name)
						// NOTE: we patch the scope of the defaultCommonSettings
						// of the options in the []string LocalSettings.
						array := unwrapCompositeLitFromValueSpec(s)
						for _, elem := range array.Elts {
							_ = elem
							value, ok := elem.(*ast.BasicLit)
							if !ok { panic("Should be an `ast.BasicLit`") }

							optname := strings.Trim(value.Value, "\"")
							opt, created := findOrCreateOption(optname, &options)
							if created {
								panic("Should be already present in `defaultCommonSettings`")
							}
							if opt.scope != scopeCommon {
								panic("Should use `scopeCommon` because should be from `defaultCommonSettings`")
							}
							opt.scope = scopeOnlyLocal
						}
					}
				default:
					continue
				}
			}

		default:
			ast.Print(fset, d)
			panic("Unhandled top level type in `internal/config/settings.go`")
		}
	}

	return options
}

func generateAndCheckPluginSection() string {
	var sb strings.Builder

	var pluginsInfo = []struct {
		name string
		info string
	}{
		{"autoclose", "automatically closes brackets, quotes, etc..."},
		{"comment", "provides automatic commenting for a number of languages"},
		{"ftoptions", "alters some default options depending on the filetype"},
		{"linter", "provides extensible linting for many languages"},
		{"literate", "provides advanced syntax highlighting for the Literate programming tool"},
		{"status", "provides some extensions to the status line (integration with Git and more)."},
		{"diff", `integrates the 'diffgutter' option with Git. If you are in a Git
   directory, the diff gutter will show changes with respect to the most
   recent Git commit rather than the diff since opening the file.`},
	}

	// Check all plugins are documented in `pluginsInfo`
	// NOTE: this path is relative to this code location
	const runtimePluginsDir = "../../runtime/plugins"
	entries, err := os.ReadDir(runtimePluginsDir)
	if err != nil { panic(err) }

	for i := 0; i < len(entries); i++ {
		if !entries[i].IsDir() { continue }
		var j int
		for j = 0; j < len(pluginsInfo); j++ {
			if entries[i].Name() == pluginsInfo[j].name { break }
		}
		if j == len(pluginsInfo) {
			panic(fmt.Sprintf("%s is a built-in plugin not documented in `generate_settings.go`", entries[i].Name()))
		}
	}

	sb.WriteString(`---

Plugin options: all plugins come with a special option to enable or disable
them. The option is a boolean with the same name as the plugin itself.

By default, the following plugins are provided, each with an option to enable
or disable them:

`)

	for i := 0; i < len(pluginsInfo); i++ {
		sb.WriteString(fmt.Sprintf("* `%s`: %s\n", pluginsInfo[i].name, pluginsInfo[i].info))
	}

	sb.WriteString(`
Any option you set in the editor will be saved to the file
'~/.config/micro/settings.json' so, in effect, your configuration file will be
created for you. If you'd like to take your configuration with you to another
machine, simply copy the 'settings.json' to the other machine.

`)

	return sb.String()
}

func generateMarkdownFile(options []option, path string) {
	// NOTE: you can not use backticks inside multiline strings(``).
	// Trade-off for readability in code(?)
	const optionsSection = `# Options

Micro stores all of the user configuration in its configuration directory.

Micro uses '$MICRO_CONFIG_HOME' as the configuration directory. If this
environment variable is not set, it uses '$XDG_CONFIG_HOME/micro' instead. If
that environment variable is not set, it uses '~/.config/micro' as the
configuration directory. In the documentation, we use '~/.config/micro' to
refer to the configuration directory (even if it may in fact be somewhere else
if you have set either of the above environment variables).

Here are the available options:

`

	const settingsFileSection = `## Settings.json file

The 'settings.json' file should go in your configuration directory (by default
at '~/.config/micro'), and should contain only options which have been modified
from their default setting. Here is the full list of options in json format,
so that you can see what the formatting should look like.

`

	var globalAndLocalSection string = `
## Global and local settings

You can set these settings either globally or locally. Locally means that the
setting won't be saved to '~/.config/micro/settings.json' and that it will only
be set in the current buffer. Setting an option globally is the default, and
will set the option in all buffers. Use the 'setlocal' command to set an option
locally rather than globally.

` +

"The `colorscheme` option is global only, and the `filetype` option is local\n" +
"only. To set an option locally, use `setlocal` instead of `set`.\n\n" +

"In the `settings.json` file you can also put set options locally by specifying\n" +
"either a glob or a filetype. Here is an example which has `tabstospaces` on for\n" +
"all files except Go files, and `tabsize` 4 for all files except Ruby files:\n\n" +
"```json\n" +
`{
    "ft:go": {
        "tabstospaces": false
    },
    "ft:ruby": {
        "tabsize": 2
    },
    "tabstospaces": true,
    "tabsize": 4
}
` +
"```\n\n" +

"Or similarly you can match with globs:\n\n" +

"```json\n" +
`{
    "glob:*.go": {
        "tabstospaces": false
    },
    "glob:*.rb": {
        "tabsize": 2
    },
    "tabstospaces": true,
    "tabsize": 4
}
` +
"```\n\n" +

"You can also omit the `glob:` prefix before globs:\n\n" +

"```json\n" +
`{
    "*.go": {
        "tabstospaces": false
    },
    "*.rb": {
        "tabsize": 2
    },
    "tabstospaces": true,
    "tabsize": 4
}
` +
"```\n\n" +

"But it is generally more recommended to use the `glob:` prefix, as it avoids\n" +
"potential conflicts with option names.\n"

	var sb strings.Builder

	sb.WriteString(optionsSection)
	for _, opt := range options {
		if opt.filled == false {
			panic(fmt.Sprintf("Option '%s' was not filled!", opt.name))
		}
		opt.writeMarkdown(&sb)
	}

	sb.WriteString(generateAndCheckPluginSection())

	sb.WriteString(settingsFileSection)
	// write the default values in json format
	sb.WriteString("```json\n{\n")
	for i := 0; i < len(options); i++ {
		opt := options[i]
		sb.WriteString(fmt.Sprintf("    \"%s\": ", opt.name))
		if opt.typ == "bool" || opt.typ == "float" || opt.typ == "string" {
			sb.WriteString(fmt.Sprintf("%s", opt.defaultValue))
		} else if opt.typ == "[]string" {
			var arr []string = opt.defaultValue.([]string)
			if len(arr) == 0 {
				sb.WriteString("[]")
			} else {
				sb.WriteString("[\n")
				for i := 0; i < len(arr); i++ {
					if i == len(arr)-1 {
						sb.WriteString(fmt.Sprintf("        %s\n", arr[i]))
					} else {
						sb.WriteString(fmt.Sprintf("        %s,\n", arr[i]))
					}
				}
				sb.WriteString("    ]")
			}

		} else { panic(fmt.Sprintf("unhandled '%s' for %s\n", opt.typ, opt.name)) }

		if i == len(options) - 1 { sb.WriteString("\n") } else { sb.WriteString(",\n") }
	}
	sb.WriteString("}\n```\n\n")

	sb.WriteString(globalAndLocalSection)

	err := os.WriteFile(path, []byte(sb.String()), 0644)
	if err != nil { panic(err) }
}

func main() {
	settingsGoPath, markdownDestPath := "", ""
	{ // Arguments validation
		var err string = ""
		if len(os.Args) != 4 {
			err = "unexpected amount of arguments"
		}
		// NOTE: '--' is needed in order to avoid picking the go file as part of the
		// 'go run' command.
		if err == "" && os.Args[1] != "--" {
			err = "unexpected first argument should be '--'"
		}
		if err == "" && !strings.HasSuffix(os.Args[2], ".go") {
			err = "second argument must be a *.go file"
		}
		if err == "" && strings.Contains(os.Args[2], "/") {
			err = "second argument .go file must be a file in our relative path, no '/' allowed"
		}
		if err == "" && !strings.HasSuffix(os.Args[3], ".md") {
			err = "third argument must be a markdown filepath"
		}

		if err != "" {
			fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\n\033[0m", err)
			fmt.Fprintf(os.Stderr, "Args: %d %v\n", len(os.Args), os.Args)
			fmt.Fprintf(os.Stderr, "Usage: go run generate_settings.go -- <go-file-with-editor-settings.go> <path-to-generated-markdown>\n")
			os.Exit(1)
		}

		settingsGoPath = os.Args[2]
		markdownDestPath = os.Args[3]
	}

	var fileset token.FileSet
	var astFile *ast.File
	var err error
	var mode parser.Mode = parser.ParseComments | parser.SkipObjectResolution
	astFile, err = parser.ParseFile(&fileset, settingsGoPath, nil, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mERROR: parsing %s: %v\n\033[0m", settingsGoPath, err)
		os.Exit(1)
	}

	allcomments := ast.NewCommentMap(&fileset, astFile, astFile.Comments)
	options := collectOptions(&fileset, astFile, allcomments)

	sort.Slice(options, func(i, j int) bool {
		return options[i].name < options[j].name
	})

	if false { // for debugging
		for _, option := range options { fmt.Fprintf(os.Stderr, "%s\n", option) }
	}

	generateMarkdownFile(options, markdownDestPath)
}
