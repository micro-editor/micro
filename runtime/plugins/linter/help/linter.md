# Linter

The linter plugin runs a compiler or linter on your source code
and parses the resulting output so that the messages and line numbers
can be viewed from within micro. By default, the plugin supports the
following filetypes and linters:

* **c**: gcc
* **c++**: g++
* **d**: dmd
* **d**: ldc2
* **d**: gdc
* **go**: go build
* **go**: go vet
* **haskell**: hlint
* **java**: javac
* **javascript**: jshint
* **javascript**: eslint
* **literate**: lit
* **lua**: luacheck
* **nim**: nim
* **nix**: nix-linter
* **objective-c**: clang
* **python**: flake8
* **python**: mypy
* **python**: pyflakes
* **python**: pylint
* **python**: ruff
* **rust**: cargo clippy
* **shell**: shfmt
* **shell**: shellcheck
* **swift**: swiftc (MacOS and Linux only)
* **yaml**: yamllint

You can easily enable linting in your `settings.json` file:

```json
{
    "linter": true
}
```

If the linter plugin is enabled and the file corresponds to one of
these filetypes, each time the buffer is saved, or when the `> lint`
command is executed, micro will run the corresponding utility in the
background and display the messages when it completes.

The linter plugin also allows users to extend the supported filetypes.
From inside another micro plugin, the function `linter.makeLinter` can
be called to register a new filetype. Here is the spec for the `makeLinter`
function:

`linter.makeLinter(name, filetype, cmd, args, errorformat, os, whitelist, domatch, loffset, coffset, callback)`

* **name**: name of the linter
* **filetype**: filetype to check for to use linter
* **cmd**: main linter process that is executed
* **args**: arguments to pass to the linter process
    * use %f to refer to the current file name
    * use %d to refer to the current directory name
* **errorformat**: how to parse the linter/compiler process output
    %f: file, %l: line number, %m: error/warning message
* **os**: list of OSs this linter is supported or unsupported on
    optional param, default: {}
* **whitelist**: should the OS list be a blacklist (do not run the linter for these OSs)
           or a whitelist (only run the linter for these OSs)
    optional param, default: false (should blacklist)
* **domatch**: should the filetype be interpreted as a lua pattern to match with
         the actual filetype, or should the linter only activate on an exact match
    optional param, default: false (require exact match)
* **loffset**: line offset will be added to the line number returned by the linter
         useful if the linter returns 0-indexed lines
    optional param, default: 0
* **coffset**: column offset will be added to the col number returned by the linter
         useful if the linter returns 0-indexed columns
    optional param, default: 0
* **callback**: function to call before executing the linter, if it returns
          false the lint is canceled. The callback is passed the buf.
    optional param, default: nil

Below is an example for including a linter for any filetype using
the `misspell` linter which checks for misspelled words in a file.

```lua
function init()
    -- uses the default linter plugin
    -- matches any filetype
    linter.makeLinter("misspell", "", "misspell", {"%f"}, "%f:%l:%c: %m", {}, false, true)
end
```

Here is an example for a more typical use-case, where the linter will only match one filetype (C in this case):

```lua
function init()
    linter.makeLinter("gcc", "c", "gcc", {"-fsyntax-only", "-Wall", "-Wextra", "%f"}, "%f:%l:%c:.+: %m")
end
```
