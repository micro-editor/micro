VERSION = "1.0.0"

local os = import("os")
local filepath = import("path/filepath")
local shell = import("micro/shell")

function onBufferOpen(buf)
	if buf.Settings["diffgutter"] and (not buf.Type.Scratch) and (buf.Path ~= "") then
		-- check that file exists
		local _, err = os.Stat(buf.AbsPath)
		if err == nil then
			local dirName, fileName = filepath.Split(buf.AbsPath)
			
			local autocrlf, err = shell.ExecCommand("git", "-C", dirName, "config", "get", "core.autocrlf")
			
			local diffBase
			if not err and string.match(autocrlf, "true") then
				diffBase, err = shell.ExecCommand("git", "-C", dirName, "cat-file", "--filters", "HEAD:./" .. fileName)
			else
				diffBase, err = shell.ExecCommand("git", "-C", dirName, "show", "HEAD:./" .. fileName)
			end
			
			if err ~= nil then
				diffBase = buf:Bytes()
			end
			buf:SetDiffBase(diffBase)
		end
	end
end
