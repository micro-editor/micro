VERSION = "1.0.0" -- all built-in plugins fixed to 1.0.0

local config = import("micro/config")

function startswith(str, start)
   return string.sub(str,1,string.len(start))==start
end

function endswith(str, endStr)
   return endStr=='' or string.sub(str,-string.len(endStr))==endStr
end

function split(string, sep)
    local sep, fields = sep or ":", {}
    local pattern = string.format("([^%s]+)", sep)
    string:gsub(pattern, function(c) fields[#fields+1] = c end)
    return fields
end

function onBufferOpen(buf)
    if not endswith(buf.Path, ".lit") then
        return
    end

    local codetype = "unknown"
    for i=1,buf:LinesNum() do
        local line = buf:Line(i-1)
        if startswith(line, "@code_type") then
            codetype = split(line, " ")[2]
            break
        end
    end

    local syntaxFile = ""
    syntaxFile = syntaxFile .. "filetype: literate-" .. codetype .. "\n"
    syntaxFile = syntaxFile .. "detect:\n"
    syntaxFile = syntaxFile .. "    filename: \"\\\\.lit$\"\n"
    syntaxFile = syntaxFile .. "rules:\n"
    syntaxFile = syntaxFile .. "    - include: \"markdown\"\n"
    syntaxFile = syntaxFile .. "    - special: \"^(@s|@title|@code_type|@comment_type|@include|@change|@change_end)\"\n"
    syntaxFile = syntaxFile .. "    - special: \"(@add_css|@overwrite_css|@colorscheme|@compiler|@error_format|@book)\"\n"
    syntaxFile = syntaxFile .. "    - default:\n"
    syntaxFile = syntaxFile .. "        start: \"---.*$\"\n"
    syntaxFile = syntaxFile .. "        end: \"---$\"\n"
    syntaxFile = syntaxFile .. "        limit-group: \"identifier\"\n"
    syntaxFile = syntaxFile .. "        rules:\n"
    syntaxFile = syntaxFile .. "            - special:\n"
    syntaxFile = syntaxFile .. "                start: \"@\\\\{\"\n"
    syntaxFile = syntaxFile .. "                end: \"\\\\}\"\n"
    syntaxFile = syntaxFile .. "            - include: " .. codetype .. "\n"

    config.AddRuntimeFileFromMemory(config.RTSyntax, "literate.yaml", syntaxFile)
    config.Reload()
    buf:UpdateRules()
end
