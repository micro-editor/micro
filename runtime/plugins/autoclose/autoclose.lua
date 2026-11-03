VERSION = "1.0.0"

local uutil = import("micro/util")
local buffer = import("micro/buffer")
local autoclosePairs = {"\"\"", "''", "``", "()", "{}", "[]"}
local autoNewlinePairs = {"()", "{}", "[]"}

function charAt(str, i)
    -- lua indexing is one off from go
    return uutil.RuneAt(str, i-1)
end

-- returns true if str ends inside an unterminated string quoted with q
function inString(str, q)
    local inside = false
    local i = 1
    while i <= #str do
        local c = str:sub(i, i)
        if c == "\\" then
            i = i + 1
        elseif c == q then
            inside = not inside
        end
        i = i + 1
    end
    return inside
end

function onRune(bp, r)
    for i = 1, #autoclosePairs do
        if r == charAt(autoclosePairs[i], 2) then
            local curLine = bp.Buf:Line(bp.Cursor.Y)

            if charAt(curLine, bp.Cursor.X+1) == charAt(autoclosePairs[i], 2) then
                bp:Backspace()
                bp:CursorRight()
                break
            end

            local isQuote = charAt(autoclosePairs[i], 1) == charAt(autoclosePairs[i], 2)

            if isQuote then
                -- don't autoclose a quote that terminates a string
                local before = uutil.String(bp.Buf:Substr(buffer.Loc(0, bp.Cursor.Y), buffer.Loc(bp.Cursor.X-1, bp.Cursor.Y)))
                if inString(before, r) then
                    break
                end
            end

            if bp.Cursor.X > 1 then
                local prev = charAt(curLine, bp.Cursor.X-1)
                -- a word character before a double quote or backtick is
                -- usually a string prefix (e.g. f"" in Python), but before
                -- a single quote it is usually an apostrophe
                if (uutil.IsWordChar(prev) and (not isQuote or r == "'")) or prev == charAt(autoclosePairs[i], 1) then
                    break
                end
            end
        end
        if r == charAt(autoclosePairs[i], 1) then
            local curLine = bp.Buf:Line(bp.Cursor.Y)

            if bp.Cursor.X == uutil.CharacterCountInString(curLine) or not uutil.IsWordChar(charAt(curLine, bp.Cursor.X+1)) then
                -- the '-' here is to derefence the pointer to bp.Cursor.Loc which is automatically made
                -- when converting go structs to lua
                -- It needs to be dereferenced because the function expects a non pointer struct
                bp.Buf:Insert(-bp.Cursor.Loc, charAt(autoclosePairs[i], 2))
                bp:CursorLeft()
                break
            end
        end
    end
    return true
end

function preInsertNewline(bp)
    local curLine = bp.Buf:Line(bp.Cursor.Y)
    local curRune = charAt(curLine, bp.Cursor.X)
    local nextRune = charAt(curLine, bp.Cursor.X+1)
    local ws = uutil.GetLeadingWhitespace(curLine)

    for i = 1, #autoNewlinePairs do
        if curRune == charAt(autoNewlinePairs[i], 1) then
            if nextRune == charAt(autoNewlinePairs[i], 2) then
                bp.Buf:Insert(-bp.Cursor.Loc, "\n" .. ws)
                bp:StartOfLine()
                bp:CursorLeft()
                bp:InsertNewline()
                bp:InsertTab()
                return false
            end
        end
    end

    return true
end

function preBackspace(bp)
    for i = 1, #autoclosePairs do
        local curLine = bp.Buf:Line(bp.Cursor.Y)
        if charAt(curLine, bp.Cursor.X+1) == charAt(autoclosePairs[i], 2) and charAt(curLine, bp.Cursor.X) == charAt(autoclosePairs[i], 1) then
            bp:Delete()
        end
    end

    return true
end
