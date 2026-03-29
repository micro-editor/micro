VERSION = "1.0.0"

local util = import("micro/util")
local config = import("micro/config")
local buffer = import("micro/buffer")
local micro = import("micro")

local ft = {}

ft["apacheconf"] = "# %s"
ft["batch"] = ":: %s"
ft["c"] = "// %s"
ft["c++"] = "// %s"
ft["cmake"] = "# %s"
ft["conf"] = "# %s"
ft["crystal"] = "# %s"
ft["css"] = "/* %s */"
ft["d"] = "// %s"
ft["dart"] = "// %s"
ft["dockerfile"] = "# %s"
ft["elm"] = "-- %s"
ft["fish"] = "# %s"
ft["gdscript"] = "# %s"
ft["glsl"] = "// %s"
ft["go"] = "// %s"
ft["haskell"] = "-- %s"
ft["html"] = "<!-- %s -->"
ft["ini"] = "; %s"
ft["java"] = "// %s"
ft["javascript"] = "// %s"
ft["jinja2"] = "{# %s #}"
ft["json"] = "// %s"
ft["julia"] = "# %s"
ft["kotlin"] = "// %s"
ft["lua"] = "-- %s"
ft["markdown"] = "<!-- %s -->"
ft["nginx"] = "# %s"
ft["nim"] = "# %s"
ft["objc"] = "// %s"
ft["ocaml"] = "(* %s *)"
ft["pascal"] = "{ %s }"
ft["perl"] = "# %s"
ft["php"] = "// %s"
ft["pony"] = "// %s"
ft["powershell"] = "# %s"
ft["proto"] = "// %s"
ft["python"] = "# %s"
ft["python3"] = "# %s"
ft["ruby"] = "# %s"
ft["rust"] = "// %s"
ft["scala"] = "// %s"
ft["shell"] = "# %s"
ft["sql"] = "-- %s"
ft["swift"] = "// %s"
ft["tex"] = "% %s"
ft["toml"] = "# %s"
ft["twig"] = "{# %s #}"
ft["typescript"] = "// %s"
ft["v"] = "// %s"
ft["xml"] = "<!-- %s -->"
ft["yaml"] = "# %s"
ft["zig"] = "// %s"
ft["zscript"] = "// %s"
ft["zsh"] = "# %s"

function updateCommentType(buf)
    -- NOTE: Using DoSetOptionNative to avoid LocalSettings[option] = true
    -- so that "comment.type" can be reset by a "filetype" change to default.
    if (buf.Settings["comment.type"] == "") then
        -- NOTE: This won't get triggered if a filetype is change via `setlocal filetype`
        -- since it is not registered with `RegisterGlobalOption()``
        if buf.Settings["commenttype"] ~= nil then
            buf:DoSetOptionNative("comment.type", buf.Settings["commenttype"])
        else
            if (ft[buf.Settings["filetype"]] ~= nil) then
                buf:DoSetOptionNative("comment.type", ft[buf.Settings["filetype"]])
            else
                buf:DoSetOptionNative("comment.type", "# %s")
            end
        end
    end
end

function isCommented(bp, lineN, commentRegex)
    local line = bp.Buf:Line(lineN)
    local regex = commentRegex:gsub("%s+", "%s*")
    return string.match(line, regex)
end

function commentLine(bp, lineN, indentLen)
    local line = bp.Buf:Line(lineN)
    local commentType = bp.Buf.Settings["comment.type"]
    local indent = string.sub(line, 1, indentLen)
    local trimmedLine = string.sub(line, indentLen + 1)
    trimmedLine = trimmedLine:gsub("%%", "%%%%")
    local commentedLine = commentType:gsub("%%s", trimmedLine)
    bp.Buf:Replace(buffer.Loc(0, lineN), buffer.Loc(#line, lineN), indent .. commentedLine)
end

function uncommentLine(bp, lineN, commentRegex)
    local line = bp.Buf:Line(lineN)
    if not string.match(line, commentRegex) then
        commentRegex = commentRegex:gsub("%s+", "%s*")
    end
    if string.match(line, commentRegex) then
        uncommentedLine = string.match(line, commentRegex)
        bp.Buf:Replace(buffer.Loc(0, lineN), buffer.Loc(#line, lineN), util.GetLeadingWhitespace(line) .. uncommentedLine)
    end
end

function toggleCommentSelection(bp, lines, commentRegex)
    local allComments = true
    for line,_ in pairs(lines) do
        if not isCommented(bp, line, commentRegex) then
            allComments = false
            break
        end
    end

    -- NOTE: we assume that the indentation is either tabs only or spaces only
    local indentMin = -1
    if not allComments then
        for line,_ in pairs(lines) do
            local indentLen = #util.GetLeadingWhitespace(bp.Buf:Line(line))
            if indentMin == -1 or indentLen < indentMin then
                indentMin = indentLen
            end
        end
    end

    for line,_ in pairs(lines) do
        if allComments then
            uncommentLine(bp, line, commentRegex)
        else
            commentLine(bp, line, indentMin)
        end
    end
    return not allComments
end

function comment(bp, args)
    updateCommentType(bp.Buf)

    local commentType = bp.Buf.Settings["comment.type"]
    local commentRegex = "^%s*" .. commentType:gsub("%%","%%%%"):gsub("%$","%$"):gsub("%)","%)"):gsub("%(","%("):gsub("%?","%?"):gsub("%*", "%*"):gsub("%-", "%-"):gsub("%.", "%."):gsub("%+", "%+"):gsub("%]", "%]"):gsub("%[", "%["):gsub("%%%%s", "(.*)")

    local lines = {}
    local curData = {}
    -- gather cursor data and lines to (un)comment
    for i = 1,#bp.Buf:getCursors() do
        local cursor = bp.Buf:getCursor(i-1)
        local hasSelection = cursor:HasSelection()
        local excludedEnd = nil
        if hasSelection then
            local startSel = 1
            local endSel = 2
            if cursor.CurSelection[startSel]:GreaterThan(-cursor.CurSelection[endSel]) then
                startSel = 2
                endSel = 1
            end
            local fromLineNo = cursor.CurSelection[startSel].Y
            local toLineNo = cursor.CurSelection[endSel].Y

            -- don't indent the line after when selection ends in a newline
            if cursor.CurSelection[endSel].X == 0 then
                excludedEnd = endSel
                toLineNo = toLineNo - 1
            end

            for lineN = fromLineNo,toLineNo do
                lines[lineN] = true
            end
        else
            lines[cursor.Y] = true
        end
        table.insert(curData, {
            sel = -cursor.CurSelection,
            curpos = -cursor.Loc,
            cursor = cursor,
            hasSelection = hasSelection,
            excludedEnd = excludedEnd,
        })
    end
    -- (un)comment selected lines
    local commented = toggleCommentSelection(bp, lines, commentRegex)
    -- restore cursors
    local displacement = (string.find(commentType, "%%s") - 1) * (commented and 1 or -1)
    for i=1,#curData do
        local cursor = curData[i].cursor
        if curData[i].hasSelection then
            for j=1,2 do
                cursor.CurSelection[j].Y = curData[i].sel[j].Y
                cursor.CurSelection[j].X = curData[i].sel[j].X + (j == curData[i].excludedEnd and 0 or displacement)
            end
        else
            cursor.Y = curData[i].curpos.Y
            cursor.X = curData[i].curpos.X + displacement
        end
        cursor:Relocate()
        cursor:StoreVisualX()
    end
end

function string.starts(String,Start)
    return string.sub(String,1,string.len(Start))==Start
end

function preinit()
    config.RegisterCommonOption("comment", "type", "")
end

function init()
    config.MakeCommand("comment", comment, config.NoComplete)
    config.TryBindKey("Alt-/", "lua:comment.comment", false)
    config.TryBindKey("CtrlUnderscore", "lua:comment.comment", false)
    config.AddRuntimeFile("comment", config.RTHelp, "help/comment.md")
end
