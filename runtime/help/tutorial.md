# Tutorial

This is a brief intro to `micro` workflow with simple examples how to configure
settings, keys, and use `init.lua`.

Hopefully you'll find this useful.

## Command mode

Press `Ctrl-e` to open micro's command prompt. Typing `help tutorial` will open
this documentation.

For the rest of the docs `> help tutoral` indicates pressing `Ctrl-e`.

## Default keyboard shortcuts

Enter `> help defaultkeys` for a list of the default keybindings.

## Simple workflow: Move content from one file to another

For this example, we edit micro's own codebase, and move contents between files.

Press `Ctrl-o` and enter filename to open the file in the current view
(use `Tab` key to help with autocompletion)
```
> open internal/config/plugin_manager.go
```
Then from the command prompt (`Ctrl-e`) open a second file in a vertical split
```
> vsplit internal/config/plugin.go
```
Use `Ctrl-w` ("jump to next split" shortcut) to switch to the first file and
cut the `PluginInfo` structure into clipboard using `Ctrl-x`.

Press `Ctrl-w` again to switch back to second file and paste the clipboard
content using `Ctrl-v`.

Now press `Ctrl-s` (or `F2`) to save current file and `Ctrl-q` to close it.

To preview the changes, run `git diff` by pressing `Ctrl-b` ("shell mode")
and entering the command. You will see changes only to the second file
`plugin.go`, because the first file is not saved yet.

Hit `Ctrl-q` again and micro will prompt if you want to save the first file
before closing. Press `y` and you're done.

Congratulations with completing your first mouseless tutorial with micro.

## Settings

In micro, your settings are stored in `~/.config/micro/settings.json`, which
is created on the first run. It can be edited directly, or you can invoke
`> set option value` command from `micro`. With the command changes are
applied  immediately and then saved into `settings.json`.

Options can also be set without saving them, using `setlocal` command. Local
means the setting is not global. For example, if you have two splits open,
and you type `> setlocal tabsize 2`, the tabsize will only be 2 in the
current buffer, and micro will not update `settings.json` file.

You can also set options for specific file types in `settings.json`. If you
want the `tabsize` to be 2 only in Ruby files, and 4 otherwise:

```json
{
    "*.rb": {
        "tabsize": 2
    },
    "tabsize": 4
}
```

Micro will set the `tabsize` to 2 only in files which match the glob `*.rb`.

See `> help options` to read about all the available options.

### Setting keybindings

Keybindings work in much the same way as options. You configure them using the
`~/.config/micro/bindings.json` file.

For example if you would like to bind `Ctrl-r` to redo you could put the
following in `bindings.json`:

```json
{
    "Ctrl-r": "Redo"
}
```

Very simple.

You can also bind keys while in micro by using the `> bind key action` command,
but the bindings you make with the command won't be saved to the
`bindings.json` file.

For more information about keybindings, like which keys can be bound, and what
actions are available, see the `keybindings` help topic (`> help keybindings`).

### Configuration with Lua

If you need more power than the json files provide, you can use the `init.lua`
file. Create it in `~/.config/micro`. This file is a lua file that is run when
micro starts and is essentially a one-file plugin. The plugin name is
`initlua`.

This example will show you how to use the `init.lua` file by creating a binding
to `Ctrl-r` which will execute the bash command `go run` on the current file,
given that the current file is a Go file.

You can do that by putting the following in `init.lua`:

```lua
local config = import("micro/config")
local shell = import("micro/shell")

function init()
    -- true means overwrite any existing binding to Ctrl-r
    -- this will modify the bindings.json file
    config.TryBindKey("Ctrl-r", "lua:initlua.gorun", true)
end

function gorun(bp)
    local buf = bp.Buf
    if buf:FileType() == "go" then
        -- the true means run in the foreground
        -- the false means send output to stdout (instead of returning it)
        shell.RunInteractiveShell("go run " .. buf.Path, true, false)
    end
end
```

Alternatively, you could get rid of the `TryBindKey` line, and put this line in
the `bindings.json` file:

```json
{
    "Ctrl-r": "lua:initlua.gorun"
}
```

For more information about plugins and the lua system that micro uses, see the
`plugins` help topic (`> help plugins`).
