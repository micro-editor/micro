package main

import (
	"log"
	"time"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"

	"github.com/micro-editor/micro/v2/internal/action"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/display"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/shell"
	"github.com/micro-editor/micro/v2/internal/util"
)

func init() {
	ulua.L = lua.NewState()
	ulua.L.SetGlobal("import", luar.New(ulua.L, LuaImport))
}

// LuaImport is meant to be called from lua by a plugin and will import the given micro package
func LuaImport(pkg string) *lua.LTable {
	switch pkg {
	case "micro":
		return luaImportMicro()
	case "micro/shell":
		return luaImportMicroShell()
	case "micro/buffer":
		return luaImportMicroBuffer()
	case "micro/config":
		return luaImportMicroConfig()
	case "micro/util":
		return luaImportMicroUtil()
	default:
		return ulua.Import(pkg)
	}
}

func luaImportMicro() *lua.LTable {
	pkg := ulua.L.NewTable()

	ulua.L.SetField(pkg, "TermMessage", luar.New(ulua.L, screen.TermMessage))
	ulua.L.SetField(pkg, "TermError", luar.New(ulua.L, screen.TermError))
	ulua.L.SetField(pkg, "InfoBar", luar.New(ulua.L, action.GetInfoBar))
	ulua.L.SetField(pkg, "Log", luar.New(ulua.L, log.Println))
	ulua.L.SetField(pkg, "SetStatusInfoFn", luar.New(ulua.L, display.SetStatusInfoFnLua))
	ulua.L.SetField(pkg, "CurPane", luar.New(ulua.L, func() *action.BufPane {
		return action.MainTab().CurPane()
	}))
	ulua.L.SetField(pkg, "CurTab", luar.New(ulua.L, action.MainTab))
	ulua.L.SetField(pkg, "Tabs", luar.New(ulua.L, func() *action.TabList {
		return action.Tabs
	}))
	ulua.L.SetField(pkg, "After", luar.New(ulua.L, func(t time.Duration, f func()) {
		time.AfterFunc(t, func() {
			timerChan <- f
		})
	}))

	return pkg
}

func luaImportMicroConfig() *lua.LTable {
	pkg := ulua.L.NewTable()

	ulua.L.SetField(pkg, "MakeCommand", luar.New(ulua.L, action.MakeCommand))
	ulua.L.SetField(pkg, "FileComplete", luar.New(ulua.L, buffer.FileComplete))
	ulua.L.SetField(pkg, "HelpComplete", luar.New(ulua.L, action.HelpComplete))
	ulua.L.SetField(pkg, "OptionComplete", luar.New(ulua.L, action.OptionComplete))
	ulua.L.SetField(pkg, "OptionValueComplete", luar.New(ulua.L, action.OptionValueComplete))
	ulua.L.SetField(pkg, "NoComplete", luar.New(ulua.L, nil))
	ulua.L.SetField(pkg, "TryBindKey", luar.New(ulua.L, action.TryBindKeyPlug))
	ulua.L.SetField(pkg, "Reload", luar.New(ulua.L, action.ReloadConfig))
	ulua.L.SetField(pkg, "AddRuntimeFileFromMemory", luar.New(ulua.L, config.PluginAddRuntimeFileFromMemory))
	ulua.L.SetField(pkg, "AddRuntimeFilesFromDirectory", luar.New(ulua.L, config.PluginAddRuntimeFilesFromDirectory))
	ulua.L.SetField(pkg, "AddRuntimeFile", luar.New(ulua.L, config.PluginAddRuntimeFile))
	ulua.L.SetField(pkg, "ListRuntimeFiles", luar.New(ulua.L, config.PluginListRuntimeFiles))
	ulua.L.SetField(pkg, "ReadRuntimeFile", luar.New(ulua.L, config.PluginReadRuntimeFile))
	ulua.L.SetField(pkg, "NewRTFiletype", luar.New(ulua.L, config.NewRTFiletype))
	ulua.L.SetField(pkg, "RTColorscheme", luar.New(ulua.L, config.RTColorscheme))
	ulua.L.SetField(pkg, "RTSyntax", luar.New(ulua.L, config.RTSyntax))
	ulua.L.SetField(pkg, "RTHelp", luar.New(ulua.L, config.RTHelp))
	ulua.L.SetField(pkg, "RTPlugin", luar.New(ulua.L, config.RTPlugin))
	ulua.L.SetField(pkg, "RegisterCommonOption", luar.New(ulua.L, config.RegisterCommonOptionPlug))
	ulua.L.SetField(pkg, "RegisterGlobalOption", luar.New(ulua.L, config.RegisterGlobalOptionPlug))
	ulua.L.SetField(pkg, "GetGlobalOption", luar.New(ulua.L, config.GetGlobalOption))
	ulua.L.SetField(pkg, "SetGlobalOption", luar.New(ulua.L, action.SetGlobalOptionPlug))
	ulua.L.SetField(pkg, "SetGlobalOptionNative", luar.New(ulua.L, action.SetGlobalOptionNativePlug))
	ulua.L.SetField(pkg, "ConfigDir", luar.New(ulua.L, config.ConfigDir))

	return pkg
}

func luaImportMicroShell() *lua.LTable {
	pkg := ulua.L.NewTable()

	ulua.L.SetField(pkg, "ExecCommand", luar.New(ulua.L, shell.ExecCommand))
	ulua.L.SetField(pkg, "RunCommand", luar.New(ulua.L, shell.RunCommand))
	ulua.L.SetField(pkg, "RunBackgroundShell", luar.New(ulua.L, shell.RunBackgroundShell))
	ulua.L.SetField(pkg, "ExecBackgroundCommand", luar.New(ulua.L, shell.ExecBackgroundCommand))
	ulua.L.SetField(pkg, "RunBackgroundCommand", luar.New(ulua.L, shell.RunBackgroundCommand))
	ulua.L.SetField(pkg, "RunInteractiveShell", luar.New(ulua.L, shell.RunInteractiveShell))
	ulua.L.SetField(pkg, "JobStart", luar.New(ulua.L, shell.JobStart))
	ulua.L.SetField(pkg, "JobSpawn", luar.New(ulua.L, shell.JobSpawn))
	ulua.L.SetField(pkg, "JobStop", luar.New(ulua.L, shell.JobStop))
	ulua.L.SetField(pkg, "JobSend", luar.New(ulua.L, shell.JobSend))
	ulua.L.SetField(pkg, "RunTermEmulator", luar.New(ulua.L, action.RunTermEmulator))
	ulua.L.SetField(pkg, "TermEmuSupported", luar.New(ulua.L, action.TermEmuSupported))

	return pkg
}

func luaImportMicroBuffer() *lua.LTable {
	pkg := ulua.L.NewTable()

	ulua.L.SetField(pkg, "NewMessage", luar.New(ulua.L, buffer.NewMessage))
	ulua.L.SetField(pkg, "NewMessageAtLine", luar.New(ulua.L, buffer.NewMessageAtLine))
	ulua.L.SetField(pkg, "MTInfo", luar.New(ulua.L, buffer.MTInfo))
	ulua.L.SetField(pkg, "MTWarning", luar.New(ulua.L, buffer.MTWarning))
	ulua.L.SetField(pkg, "MTError", luar.New(ulua.L, buffer.MTError))
	ulua.L.SetField(pkg, "Loc", luar.New(ulua.L, func(x, y int) buffer.Loc {
		return buffer.Loc{x, y}
	}))
	ulua.L.SetField(pkg, "SLoc", luar.New(ulua.L, func(line, row int) display.SLoc {
		return display.SLoc{line, row}
	}))
	ulua.L.SetField(pkg, "BTDefault", luar.New(ulua.L, buffer.BTDefault.Kind))
	ulua.L.SetField(pkg, "BTHelp", luar.New(ulua.L, buffer.BTHelp.Kind))
	ulua.L.SetField(pkg, "BTLog", luar.New(ulua.L, buffer.BTLog.Kind))
	ulua.L.SetField(pkg, "BTScratch", luar.New(ulua.L, buffer.BTScratch.Kind))
	ulua.L.SetField(pkg, "BTRaw", luar.New(ulua.L, buffer.BTRaw.Kind))
	ulua.L.SetField(pkg, "BTInfo", luar.New(ulua.L, buffer.BTInfo.Kind))
	ulua.L.SetField(pkg, "NewBuffer", luar.New(ulua.L, func(text, path string) *buffer.Buffer {
		return buffer.NewBufferFromString(text, path, buffer.BTDefault)
	}))
	ulua.L.SetField(pkg, "NewBufferFromFile", luar.New(ulua.L, func(path string) (*buffer.Buffer, error) {
		return buffer.NewBufferFromFile(path, buffer.BTDefault)
	}))
	ulua.L.SetField(pkg, "ByteOffset", luar.New(ulua.L, buffer.ByteOffset))
	ulua.L.SetField(pkg, "Log", luar.New(ulua.L, buffer.WriteLog))
	ulua.L.SetField(pkg, "LogBuf", luar.New(ulua.L, buffer.GetLogBuf))

	return pkg
}

func luaImportMicroUtil() *lua.LTable {
	pkg := ulua.L.NewTable()

	ulua.L.SetField(pkg, "RuneAt", luar.New(ulua.L, util.LuaRuneAt))
	ulua.L.SetField(pkg, "GetLeadingWhitespace", luar.New(ulua.L, util.LuaGetLeadingWhitespace))
	ulua.L.SetField(pkg, "IsWordChar", luar.New(ulua.L, util.LuaIsWordChar))
	ulua.L.SetField(pkg, "String", luar.New(ulua.L, util.String))
	ulua.L.SetField(pkg, "Unzip", luar.New(ulua.L, util.Unzip))
	ulua.L.SetField(pkg, "Version", luar.New(ulua.L, util.Version))
	ulua.L.SetField(pkg, "SemVersion", luar.New(ulua.L, util.SemVersion))
	ulua.L.SetField(pkg, "HttpRequest", luar.New(ulua.L, util.HttpRequest))
	ulua.L.SetField(pkg, "CharacterCountInString", luar.New(ulua.L, util.CharacterCountInString))
	ulua.L.SetField(pkg, "RuneStr", luar.New(ulua.L, func(r rune) string {
		return string(r)
	}))

	return pkg
}
