VERSION = "1.0.0" -- all built-in plugins fixed to 1.0.0

function onBufferOpen(b)
    local ft = b:FileType()

    if ft == "go" or
    ft == "makefile" then
        b:SetOption("tabstospaces", "off")
    elseif ft == "fish" or
           ft == "python" or
           ft == "python2" or
           ft == "python3" or
           ft == "yaml" or
           ft == "nim" then
        b:SetOption("tabstospaces", "on")
    end
end
