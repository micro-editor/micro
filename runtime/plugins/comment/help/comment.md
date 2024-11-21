# Comment Plugin

The comment plugin provides auto commenting/uncommenting.
The default binding to comment/uncomment a line is `Alt-/`
and `CtrlUnderscore`, which is equivalent in most terminals
to `Ctrl-/`. You can easily modify that in your `bindings.json`
file:

```json
{
    "Alt-g": "lua:comment.comment"
}
```

You can also execute a command which will do the same thing as
the binding:

```
> comment
```

If you have a selection of text, or multiple cursors, the plugin
will comment all the selected lines.

The comment type will be auto detected based on the filetype,
but it is only available for certain filetypes:

* apacheconf: `# %s`
* bat: `:: %s`
* c: `// %s`
* c++: `// %s`
* cmake: `# %s`
* conf: `# %s`
* crystal: `# %s`
* css: `/* %s */`
* d: `// %s`
* dart: `// %s`
* dockerfile: `# %s`
* elm: `-- %s`
* fish: `# %s`
* gdscript: `# %s`
* glsl: `// %s`
* go: `// %s`
* haskell: `-- %s`
* html: `<!-- %s -->`
* ini: `; %s`
* java: `// %s`
* javascript: `// %s`
* jinja2: `{# %s #}`
* json: `// %s`
* julia: `# %s`
* kotlin: `// %s`
* lua: `-- %s`
* markdown: `<!-- %s -->`
* nginx: `# %s`
* nim: `# %s`
* objc: `// %s`
* pascal: `{ %s }`
* perl: `# %s`
* php: `// %s`
* pony: `// %s`
* powershell: `# %s`
* proto: `// %s`
* python: `# %s`
* python3: `# %s`
* ruby: `# %s`
* rust: `// %s`
* scala: `// %s`
* shell: `# %s`
* sql: `-- %s`
* swift: `// %s`
* tex: `% %s`
* toml: `# %s`
* twig: `{# %s #}`
* v: `// %s`
* xml: `<!-- %s -->`
* yaml: `# %s`
* zig: `// %s`
* zscript: `// %s`
* zsh: `# %s`

If your filetype is not available here, you can simply modify
the `comment.type` option:

```
set comment.type "/* %s */"
```

Or in your `settings.json`:

```json
{
    "*.c": {
        "comment.type": "/* %s */"
    }
}
```

`commenttype` (without the dot) is the legacy option that is
superseded by `comment.type`. `commenttype` is still supported
but deprecated.
**Use `comment.type` instead.**
