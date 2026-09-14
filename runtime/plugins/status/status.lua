VERSION = "1.0.0" -- all built-in plugins fixed to 1.0.0

local micro = import("micro")
local buffer = import("micro/buffer")
local config = import("micro/config")
local shell = import("micro/shell")
local filepath = import("filepath")
local humanize = import("humanize")
local strings = import("strings")

function init()
    micro.SetStatusInfoFn("status.branch")
    micro.SetStatusInfoFn("status.hash")
    micro.SetStatusInfoFn("status.paste")
    micro.SetStatusInfoFn("status.vcol")
    micro.SetStatusInfoFn("status.lines")
    micro.SetStatusInfoFn("status.bytes")
    micro.SetStatusInfoFn("status.size")
    config.AddRuntimeFile("status", config.RTHelp, "help/status.md")
end

function lines(b)
    return tostring(b:LinesNum())
end

function vcol(b)
    return tostring(b:GetActiveCursor():GetVisualX(false))
end

function bytes(b)
    return tostring(b:Size())
end

function size(b)
    return humanize.Bytes(b:Size())
end

local function parseRevision(b, opt)
    if b.Type.Kind ~= buffer.BTInfo then
        local dir = filepath.Dir(b.Path)
        local str, err = shell.ExecCommand("git", "-C", dir, "rev-parse", opt, "HEAD")
        if err == nil then
            return strings.TrimSpace(str)
        end
    end
    return ""
end

function branch(b)
    return parseRevision(b, "--abbrev-ref")
end

function hash(b)
    return parseRevision(b, "--short")
end

function paste(b)
    if config.GetGlobalOption("paste") then
        return "PASTE "
    end
    return ""
end
