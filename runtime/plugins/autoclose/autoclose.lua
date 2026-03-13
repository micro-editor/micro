VERSION = "1.0.0"

local uutil = import("micro/util")
local utf8 = import("utf8")
local autoclosePairs = {"\"\"", "''", "``", "()", "{}", "[]"}
local autoNewlinePairs = {"()", "{}", "[]"}

function charAt(str, i)
    -- lua indexing is one off from go
    return uutil.RuneAt(str, i-1)
end

function preInsertNewlineAct(bp)
    local curLine = bp.Buf:Line(bp.Cursor.Y)
    local curRune = charAt(curLine, bp.Cursor.X)
    local nextRune = charAt(curLine, bp.Cursor.X+1)
    local ws = uutil.GetLeadingWhitespace(curLine)

    for j = 1, #autoNewlinePairs do
        if curRune == charAt(autoNewlinePairs[j], 1) then
            if nextRune == charAt(autoNewlinePairs[j], 2) then
                bp.Buf:Insert(-bp.Cursor.Loc, "\n" .. ws)
                bp:StartOfLine()
                bp:CursorLeft()
                bp:InsertNewline()
                bp:InsertTab()
                return
            end
        end
    end
    
    bp:InsertNewline()
    return
end

function preBackspaceAct(bp)
    for i = 1, #autoclosePairs do
        local curLine = bp.Buf:Line(bp.Cursor.Y)
        if charAt(curLine, bp.Cursor.X+1) == charAt(autoclosePairs[i], 2) and charAt(curLine, bp.Cursor.X) == charAt(autoclosePairs[i], 1) then
            bp:Delete()
            return
        end
    end
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

            if bp.Cursor.X > 1 and (uutil.IsWordChar(charAt(curLine, bp.Cursor.X-1)) or charAt(curLine, bp.Cursor.X-1) == charAt(autoclosePairs[i], 1)) then
                break
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
end

function preInsertNewline(bp)
    local activeCursorNum = bp.Buf:GetActiveCursor().Num
    local inserted = false
    for i = 1,#bp.Buf:getCursors() do
        bp.Cursor = bp.Buf:GetCursor(i-1)
        preInsertNewlineAct(bp)
    end
    bp.Cursor = bp.Buf:GetCursor(activeCursorNum)

    return false
end

function preBackspace(bp)
    local activeCursorNum = bp.Buf:GetActiveCursor().Num
    local inserted = false
    for i = 1,#bp.Buf:getCursors() do
        bp.Cursor = bp.Buf:GetCursor(i-1)
        preBackspaceAct(bp)
        bp:Backspace()
    end
    bp.Cursor = bp.Buf:GetCursor(activeCursorNum)

    return false
end
