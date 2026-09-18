# Tutorial

This is a brief intro to micro's configuration system that will give some
simple examples showing how to configure settings, rebind keys, and use
`init.lua` to configure micro to your liking.

Hopefully you'll find this useful.

See `> help defaultkeys` for a list an explanation of the default keybindings.

### Settings

In micro, your settings are stored in `~/.config/micro/settings.json`, a file
that is created the first time you run micro. It is a json file which holds all
the settings and their values. To change an option, you can either change the
value in the `settings.json` file, or you can type it in directly while using
micro.

Press Ctrl-e to go to command mode, and type `set option value` (in the
future, I will use `> set option value` to indicate pressing Ctrl-e). The change
will take effect immediately and will also be saved to the `settings.json` file
so that the setting will stick even after you close micro.

You can also set options locally which means that the setting will only have
the value you give it in the buffer you set it in. For example, if you have two
splits open, and you type `> setlocal tabsize 2`, the tabsize will only be 2 in
the current buffer. Also micro will not save this local change to the
`settings.json` file. However, you can still set options locally in the
`settings.json` file. For example, if you want the `tabsize` to be 2 only in
Ruby files, and 4 otherwise, you could put the following in `settings.json`:

```jsonc
{
    "*.rb": {
        "tabsize": 2
    },
    "tabsize": 4
}
```

Micro will set the `tabsize` to 2 only in files which match the glob `*.rb`.

If you would like to know more about all the available options, see the
`options` topic (`> help options`).

### Keybindings

Keybindings work in much the same way as options. You configure them using the
`~/.config/micro/bindings.json` file.

For example if you would like to bind `Ctrl-r` to redo you could put the
following in `bindings.json`:

```jsonc
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

```jsonc
{
    "Ctrl-r": "lua:initlua.gorun"
}
```

For more information about plugins and the lua system that micro uses, see the
`plugins` help topic (`> help plugins`).
