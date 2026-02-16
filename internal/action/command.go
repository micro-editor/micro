package action

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	shellquote "github.com/kballard/go-shellquote"
	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/clipboard"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/micro/v2/internal/shell"
	"github.com/micro-editor/micro/v2/internal/util"
)

// A Command contains information about how to execute a command
// It has the action for that command as well as a completer function
type Command struct {
	action    func(*BufPane, []string)
	completer buffer.Completer
}

var commands map[string]Command

func InitCommands() {
	commands = map[string]Command{
		"set":         {(*BufPane).SetCmd, OptionValueComplete},
		"setlocal":    {(*BufPane).SetLocalCmd, OptionValueComplete},
		"toggle":      {(*BufPane).ToggleCmd, OptionValueComplete},
		"togglelocal": {(*BufPane).ToggleLocalCmd, OptionValueComplete},
		"reset":       {(*BufPane).ResetCmd, OptionValueComplete},
		"show":        {(*BufPane).ShowCmd, OptionComplete},
		"showkey":     {(*BufPane).ShowKeyCmd, nil},
		"run":         {(*BufPane).RunCmd, nil},
		"bind":        {(*BufPane).BindCmd, nil},
		"unbind":      {(*BufPane).UnbindCmd, nil},
		"quit":        {(*BufPane).QuitCmd, nil},
		"goto":        {(*BufPane).GotoCmd, nil},
		"jump":        {(*BufPane).JumpCmd, nil},
		"save":        {(*BufPane).SaveCmd, nil},
		"replace":     {(*BufPane).ReplaceCmd, nil},
		"replaceall":  {(*BufPane).ReplaceAllCmd, nil},
		"vsplit":      {(*BufPane).VSplitCmd, buffer.FileComplete},
		"hsplit":      {(*BufPane).HSplitCmd, buffer.FileComplete},
		"tab":         {(*BufPane).NewTabCmd, buffer.FileComplete},
		"help":        {(*BufPane).HelpCmd, HelpComplete},
		"eval":        {(*BufPane).EvalCmd, nil},
		"log":         {(*BufPane).ToggleLogCmd, nil},
		"plugin":      {(*BufPane).PluginCmd, PluginComplete},
		"reload":      {(*BufPane).ReloadCmd, nil},
		"reopen":      {(*BufPane).ReopenCmd, nil},
		"cd":          {(*BufPane).CdCmd, buffer.FileComplete},
		"pwd":         {(*BufPane).PwdCmd, nil},
		"open":        {(*BufPane).OpenCmd, buffer.FileComplete},
		"tabmove":     {(*BufPane).TabMoveCmd, nil},
		"tabswitch":   {(*BufPane).TabSwitchCmd, nil},
		"term":        {(*BufPane).TermCmd, nil},
		"memusage":    {(*BufPane).MemUsageCmd, nil},
		"retab":       {(*BufPane).RetabCmd, nil},
		"raw":         {(*BufPane).RawCmd, nil},
		"textfilter":  {(*BufPane).TextFilterCmd, nil},
	}
}

// MakeCommand is a function to easily create new commands
// This can be called by plugins in Lua so that plugins can define their own commands
func MakeCommand(name string, action func(bp *BufPane, args []string), completer buffer.Completer) {
	if action != nil {
		commands[name] = Command{action, completer}
	}
}

// CommandEditAction returns a bindable function that opens a prompt with
// the given string and executes the command when the user presses
// enter
func CommandEditAction(prompt string) BufKeyAction {
	return func(h *BufPane) bool {
		InfoBar.Prompt("> ", prompt, "Command", nil, func(resp string, canceled bool) {
			if !canceled {
				MainTab().CurPane().HandleCommand(resp)
			}
		})
		return false
	}
}

// CommandAction returns a bindable function which executes the
// given command
func CommandAction(cmd string) BufKeyAction {
	return func(h *BufPane) bool {
		MainTab().CurPane().HandleCommand(cmd)
		return false
	}
}

var PluginCmds = []string{"install", "remove", "update", "available", "list", "search"}

// PluginCmd installs, removes, updates, lists, or searches for given plugins
func (h *BufPane) PluginCmd(args []string) {
	if len(args) < 1 {
		InfoBar.Error("Not enough arguments")
		return
	}

	if h.Buf.Type != buffer.BTLog {
		h.OpenLogBuf()
	}

	config.PluginCommand(buffer.LogBuf, args[0], args[1:])
}

// RetabCmd changes all spaces to tabs or all tabs to spaces
// depending on the user's settings
func (h *BufPane) RetabCmd(args []string) {
	h.Buf.Retab()
}

// RawCmd opens a new raw view which displays the escape sequences micro
// is receiving in real-time
func (h *BufPane) RawCmd(args []string) {
	width, height := screen.Screen.Size()
	iOffset := config.GetInfoBarOffset()
	tp := NewTabFromPane(0, 0, width, height-iOffset, NewRawPane(nil))
	Tabs.AddTab(tp)
	Tabs.SetActive(len(Tabs.List) - 1)
}

// TextFilterCmd filters the selection through the command.
// Selection goes to the command input.
// On successful run command output replaces the current selection.
func (h *BufPane) TextFilterCmd(args []string) {
	if len(args) == 0 {
		InfoBar.Error("usage: textfilter arguments")
		return
	}

	for _, c := range h.Buf.GetCursors() {
		sel := c.GetSelection()
		fromSelection := len(sel) > 0
		if !fromSelection {
			c.SelectWord()
			sel = c.GetSelection()
		}
		var bout, berr bytes.Buffer
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = strings.NewReader(string(sel))
		cmd.Stderr = &berr
		cmd.Stdout = &bout
		err := cmd.Run()
		if err != nil {
			InfoBar.Error(err.Error() + " " + berr.String())
			return
		}
		c.DeleteSelection()
		insertStart := c.Loc
		insertedText := bout.String()
		h.Buf.Insert(c.Loc, insertedText)

		if fromSelection {
			// Select the pasted output if the input was selected
			charCount := util.CharacterCountInString(insertedText)
			insertEnd := insertStart.Move(charCount, h.Buf)
			c.SetSelectionStart(insertStart)
			c.SetSelectionEnd(insertEnd)
			c.Loc = insertEnd
		}
	}
}

// TabMoveCmd moves the current tab to a given index (starts at 1). The
// displaced tabs are moved up.
func (h *BufPane) TabMoveCmd(args []string) {
	if len(args) <= 0 {
		InfoBar.Error("Not enough arguments: provide an index, starting at 1")
		return
	}

	if len(args[0]) <= 0 {
		InfoBar.Error("Invalid argument: empty string")
		return
	}

	num, err := strconv.Atoi(args[0])
	if err != nil {
		InfoBar.Error("Invalid argument: ", err)
		return
	}

	// Preserve sign for relative move, if one exists
	var shiftDirection byte
	if strings.Contains("-+", string([]byte{args[0][0]})) {
		shiftDirection = args[0][0]
	}

	// Relative positions -> absolute positions
	idxFrom := Tabs.Active()
	idxTo := 0
	offset := util.Abs(num)
	if shiftDirection == '-' {
		idxTo = idxFrom - offset
	} else if shiftDirection == '+' {
		idxTo = idxFrom + offset
	} else {
		idxTo = offset - 1
	}

	// Restrain position to within the valid range
	idxTo = util.Clamp(idxTo, 0, len(Tabs.List)-1)

	activeTab := Tabs.List[idxFrom]
	Tabs.RemoveTab(activeTab.Panes[0].ID())
	Tabs.List = append(Tabs.List, nil)
	copy(Tabs.List[idxTo+1:], Tabs.List[idxTo:])
	Tabs.List[idxTo] = activeTab
	Tabs.Resize()
	Tabs.UpdateNames()
	Tabs.SetActive(idxTo)
	// InfoBar.Message(fmt.Sprintf("Moved tab from slot %d to %d", idxFrom+1, idxTo+1))
}

// TabSwitchCmd switches to a given tab either by name or by number
func (h *BufPane) TabSwitchCmd(args []string) {
	if len(args) > 0 {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			// Check for tab with this name

			found := false
			for i, t := range Tabs.List {
				if t.Panes[t.active].Name() == args[0] {
					Tabs.SetActive(i)
					found = true
				}
			}
			if !found {
				InfoBar.Error("Could not find tab: ", err)
			}
		} else {
			num--
			if num >= 0 && num < len(Tabs.List) {
				Tabs.SetActive(num)
			} else {
				InfoBar.Error("Invalid tab index")
			}
		}
	}
}

// CdCmd changes the current working directory
func (h *BufPane) CdCmd(args []string) {
	if len(args) > 0 {
		path, err := util.ReplaceHome(args[0])
		if err != nil {
			InfoBar.Error(err)
			return
		}
		err = os.Chdir(path)
		if err != nil {
			InfoBar.Error(err)
			return
		}
		wd, _ := os.Getwd()
		for _, b := range buffer.OpenBuffers {
			if len(b.Path) > 0 {
				b.Path, _ = util.MakeRelative(b.AbsPath, wd)
				if p, _ := filepath.Abs(b.Path); !strings.Contains(p, wd) {
					b.Path = b.AbsPath
				}
			}
		}
	}
}

// MemUsageCmd prints micro's memory usage
// Alloc shows how many bytes are currently in use
// Sys shows how many bytes have been requested from the operating system
// NumGC shows how many times the GC has been run
// Note that Go commonly reserves more memory from the OS than is currently in-use/required
// Additionally, even if Go returns memory to the OS, the OS does not always claim it because
// there may be plenty of memory to spare
func (h *BufPane) MemUsageCmd(args []string) {
	InfoBar.Message(util.GetMemStats())
}

// PwdCmd prints the current working directory
func (h *BufPane) PwdCmd(args []string) {
	wd, err := os.Getwd()
	if err != nil {
		InfoBar.Message(err.Error())
	} else {
		InfoBar.Message(wd)
	}
}

// OpenCmd opens a new buffer with a given filename
func (h *BufPane) OpenCmd(args []string) {
	if len(args) > 0 {
		open := func() {
			b, err := buffer.NewBufferFromFile(args[0], buffer.BTDefault)
			if err != nil {
				InfoBar.Error(err)
				return
			}
			h.OpenBuffer(b)
		}
		if h.Buf.Modified() && !h.Buf.Shared() {
			h.closePrompt("Save", open)
		} else {
			open()
		}
	} else {
		InfoBar.Error("No filename")
	}
}

// ToggleLogCmd toggles the log view
func (h *BufPane) ToggleLogCmd(args []string) {
	if h.Buf.Type != buffer.BTLog {
		h.OpenLogBuf()
	} else {
		h.Quit()
	}
}

// ReloadCmd reloads all files (syntax files, colorschemes, plugins...)
func (h *BufPane) ReloadCmd(args []string) {
	reloadRuntime(true)
}

// ReloadConfig reloads only the configuration
func ReloadConfig() {
	reloadRuntime(false)
}

func reloadRuntime(reloadPlugins bool) {
	if reloadPlugins {
		err := config.RunPluginFn("deinit")
		if err != nil {
			screen.TermMessage(err)
		}
	}

	config.InitRuntimeFiles(true)

	if reloadPlugins {
		config.InitPlugins()
	}

	err := config.ReadSettings()
	if err != nil {
		screen.TermMessage(err)
	} else {
		parsedSettings := config.ParsedSettings()
		defaultSettings := config.DefaultAllSettings()
		for k := range defaultSettings {
			if _, ok := config.VolatileSettings[k]; ok {
				// reload should not override volatile settings
				continue
			}

			if _, ok := parsedSettings[k]; ok {
				err = doSetGlobalOptionNative(k, parsedSettings[k])
			} else {
				err = doSetGlobalOptionNative(k, defaultSettings[k])
			}
			if err != nil {
				screen.TermMessage(err)
			}
		}
	}

	if reloadPlugins {
		err = config.LoadAllPlugins()
		if err != nil {
			screen.TermMessage(err)
		}
	}

	InitBindings()
	InitCommands()

	if reloadPlugins {
		err = config.RunPluginFn("preinit")
		if err != nil {
			screen.TermMessage(err)
		}
		err = config.RunPluginFn("init")
		if err != nil {
			screen.TermMessage(err)
		}
		err = config.RunPluginFn("postinit")
		if err != nil {
			screen.TermMessage(err)
		}
	}

	err = config.InitColorscheme()
	if err != nil {
		screen.TermMessage(err)
	}
	for _, b := range buffer.OpenBuffers {
		b.ReloadSettings(true)
	}
}

// ReopenCmd reopens the buffer (reload from disk)
func (h *BufPane) ReopenCmd(args []string) {
	if h.Buf.Modified() {
		InfoBar.YNPrompt("Save file before reopen?", func(yes, canceled bool) {
			if !canceled && yes {
				h.Save()
				h.ReOpen()
			} else if !canceled {
				h.ReOpen()
			}
		})
	} else {
		h.ReOpen()
	}
}

func (h *BufPane) openHelp(page string, hsplit bool, forceSplit bool) error {
	if data, err := config.FindRuntimeFile(config.RTHelp, page).Data(); err != nil {
		return errors.New(fmt.Sprintf("Unable to load help text for %s: %v", page, err))
	} else {
		helpBuffer := buffer.NewBufferFromString(string(data), page+".md", buffer.BTHelp)
		helpBuffer.SetName("Help " + page)
		helpBuffer.SetOptionNative("hltaberrors", false)
		helpBuffer.SetOptionNative("hltrailingws", false)

		if h.Buf.Type == buffer.BTHelp && !forceSplit {
			h.OpenBuffer(helpBuffer)
		} else if hsplit {
			h.HSplitBuf(helpBuffer)
		} else {
			h.VSplitBuf(helpBuffer)
		}
	}
	return nil
}

// HelpCmd tries to open the given help page according to the split type
// configured with the "helpsplit" option. It can be overridden by the optional
// arguments "-vpslit" or "-hsplit". In case more than one help page is given
// as argument then it opens all of them with the defined split type.
func (h *BufPane) HelpCmd(args []string) {
	hsplit := config.GlobalSettings["helpsplit"] == "hsplit"
	if len(args) < 1 {
		// Open the default help if the user just typed "> help"
		h.openHelp("help", hsplit, false)
	} else {
		var topics []string
		forceSplit := false
		const errSplit = "hsplit and vsplit are not allowed at the same time"
		for _, arg := range args {
			switch arg {
			case "-vsplit":
				if forceSplit {
					InfoBar.Error(errSplit)
					return
				}
				hsplit = false
				forceSplit = true
			case "-hsplit":
				if forceSplit {
					InfoBar.Error(errSplit)
					return
				}
				hsplit = true
				forceSplit = true
			default:
				topics = append(topics, arg)
			}
		}

		if len(topics) < 1 {
			// Do the same as without arg
			h.openHelp("help", hsplit, forceSplit)
			return
		}
		if len(topics) > 1 {
			forceSplit = true
		}

		for _, topic := range topics {
			if config.FindRuntimeFile(config.RTHelp, topic) != nil {
				err := h.openHelp(topic, hsplit, forceSplit)
				if err != nil {
					InfoBar.Error(err)
				}
			} else {
				InfoBar.Error("Sorry, no help for ", topic)
			}
		}
	}
}

// VSplitCmd opens one or more vertical splits with the files given as arguments
// If no file is given, it opens an empty buffer in a new split
func (h *BufPane) VSplitCmd(args []string) {
	if len(args) == 0 {
		// Open an empty vertical split
		h.VSplitAction()
		return
	}

	for _, a := range args {
		buf, err := buffer.NewBufferFromFile(a, buffer.BTDefault)
		if err != nil {
			InfoBar.Error(err)
			return
		}

		h.VSplitBuf(buf)
	}
}

// HSplitCmd opens one or more horizontal splits with the files given as arguments
// If no file is given, it opens an empty buffer in a new split
func (h *BufPane) HSplitCmd(args []string) {
	if len(args) == 0 {
		// Open an empty horizontal split
		h.HSplitAction()
		return
	}

	for _, a := range args {
		buf, err := buffer.NewBufferFromFile(a, buffer.BTDefault)
		if err != nil {
			InfoBar.Error(err)
			return
		}

		h.HSplitBuf(buf)
	}
}

// EvalCmd evaluates a lua expression
func (h *BufPane) EvalCmd(args []string) {
	InfoBar.Error("Eval unsupported")
}

// NewTabCmd opens one or more tabs with the files given as arguments
// If no file is given, it opens an empty buffer in a new tab
func (h *BufPane) NewTabCmd(args []string) {
	width, height := screen.Screen.Size()
	iOffset := config.GetInfoBarOffset()
	if len(args) > 0 {
		for _, a := range args {
			b, err := buffer.NewBufferFromFile(a, buffer.BTDefault)
			if err != nil {
				InfoBar.Error(err)
				return
			}
			tp := NewTabFromBuffer(0, 0, width, height-1-iOffset, b)
			Tabs.AddTab(tp)
			Tabs.SetActive(len(Tabs.List) - 1)
		}
	} else {
		b := buffer.NewBufferFromString("", "", buffer.BTDefault)
		tp := NewTabFromBuffer(0, 0, width, height-iOffset, b)
		Tabs.AddTab(tp)
		Tabs.SetActive(len(Tabs.List) - 1)
	}
}

func doSetGlobalOptionNative(option string, nativeValue any) error {
	if reflect.DeepEqual(config.GlobalSettings[option], nativeValue) {
		return nil
	}

	config.GlobalSettings[option] = nativeValue
	config.ModifiedSettings[option] = true
	delete(config.VolatileSettings, option)

	if option == "colorscheme" || option == "background" {
		// LoadSyntaxFiles()
		config.InitColorscheme()
		for _, b := range buffer.OpenBuffers {
			b.UpdateRules()
		}
	} else if option == "infobar" || option == "keymenu" {
		Tabs.Resize()
	} else if option == "mouse" {
		if !nativeValue.(bool) {
			screen.Screen.DisableMouse()
		} else {
			screen.Screen.EnableMouse()
		}
	} else if option == "autosave" {
		if nativeValue.(float64) > 0 {
			config.SetAutoTime(nativeValue.(float64))
		} else {
			config.SetAutoTime(0)
		}
	} else if option == "paste" {
		screen.Screen.SetPaste(nativeValue.(bool))
	} else if option == "clipboard" {
		m := clipboard.SetMethod(nativeValue.(string))
		err := clipboard.Initialize(m)
		if err != nil {
			return err
		}
	} else {
		for _, pl := range config.Plugins {
			if option == pl.Name {
				if nativeValue.(bool) && !pl.Loaded {
					pl.Load()
					_, err := pl.Call("init")
					if err != nil && err != config.ErrNoSuchFunction {
						screen.TermMessage(err)
					}
				} else if !nativeValue.(bool) && pl.Loaded {
					_, err := pl.Call("deinit")
					if err != nil && err != config.ErrNoSuchFunction {
						screen.TermMessage(err)
					}
				}
			}
		}
	}

	return nil
}

func SetGlobalOptionNative(option string, nativeValue any, writeToFile bool) error {
	if err := config.OptionIsValid(option, nativeValue); err != nil {
		return err
	}

	// check for local option first...
	for _, s := range config.LocalSettings {
		if s == option {
			return MainTab().CurPane().Buf.SetOptionNative(option, nativeValue)
		}
	}

	// ...if it's not local continue with the globals...
	if err := doSetGlobalOptionNative(option, nativeValue); err != nil {
		return err
	}

	// ...at last check the buffer locals
	for _, b := range buffer.OpenBuffers {
		b.DoSetOptionNative(option, nativeValue)
		delete(b.LocalSettings, option)
	}

	if !writeToFile {
		return nil
	}

	err := config.WriteSettings(filepath.Join(config.ConfigDir, "settings.json"))
	if err != nil {
		if errors.Is(err, util.ErrOverwrite) {
			screen.TermMessage(err)
			err = errors.Unwrap(err)
		}
		return err
	}

	return nil
}

func SetGlobalOption(option, value string, writeToFile bool) error {
	if _, ok := config.GlobalSettings[option]; !ok {
		return config.ErrInvalidOption
	}

	nativeValue, err := config.GetNativeValue(option, value)
	if err != nil {
		return err
	}

	return SetGlobalOptionNative(option, nativeValue, writeToFile)
}

func SetGlobalOptionNativePlug(option string, nativeValue any) error {
	return SetGlobalOptionNative(option, nativeValue, false)
}

func SetGlobalOptionPlug(option, value string) error {
	return SetGlobalOption(option, value, false)
}

// ResetCmd resets a setting to its default value
func (h *BufPane) ResetCmd(args []string) {
	if len(args) < 1 {
		InfoBar.Error("Not enough arguments")
		return
	}

	option := args[0]
	defaults := config.DefaultAllSettings()

	if _, ok := defaults[option]; ok {
		SetGlobalOptionNative(option, defaults[option], true)
		return
	}
	InfoBar.Error(config.ErrInvalidOption)
}

// SetCmd sets an option
func (h *BufPane) SetCmd(args []string) {
	if len(args) < 2 {
		InfoBar.Error("Not enough arguments")
		return
	}

	option := args[0]
	value := args[1]

	err := SetGlobalOption(option, value, true)
	if err == config.ErrInvalidOption {
		err := h.Buf.SetOption(option, value)
		if err != nil {
			InfoBar.Error(err)
		}
	} else if err != nil {
		InfoBar.Error(err)
	}
}

// SetLocalCmd sets an option local to the buffer
func (h *BufPane) SetLocalCmd(args []string) {
	if len(args) < 2 {
		InfoBar.Error("Not enough arguments")
		return
	}

	option := args[0]
	value := args[1]

	err := h.Buf.SetOption(option, value)
	if err != nil {
		InfoBar.Error(err)
	}
}

func (h *BufPane) toggleOption(option string, local bool) error {
	var curVal, newVal any

	if local {
		curVal = h.Buf.Settings[option]
	} else {
		curVal = config.GetGlobalOption(option)
	}
	if curVal == nil {
		return config.ErrInvalidOption
	}

	if choices, ok := config.OptionChoices[option]; ok && len(choices) == 2 {
		if curVal == choices[0] {
			newVal = choices[1]
		} else {
			newVal = choices[0]
		}
	} else if curValBool, ok := curVal.(bool); ok {
		newVal = !curValBool
	} else {
		return config.ErrOptNotToggleable
	}

	if local {
		if err := h.Buf.SetOptionNative(option, newVal); err != nil {
			return err
		}
	} else {
		if err := SetGlobalOptionNative(option, newVal, true); err != nil {
			return err
		}
	}

	return nil
}

// ToggleCmd toggles a toggleable option
func (h *BufPane) ToggleCmd(args []string) {
	if len(args) < 1 {
		InfoBar.Error("Not enough arguments: provide a toggleable option")
		return
	}
	if err := h.toggleOption(args[0], false); err != nil {
		InfoBar.Error(err)
	}
}

// ToggleCmd toggles a toggleable option local to the buffer
func (h *BufPane) ToggleLocalCmd(args []string) {
	if len(args) < 1 {
		InfoBar.Error("Not enough arguments: provide a toggleable option")
		return
	}
	if err := h.toggleOption(args[0], true); err != nil {
		InfoBar.Error(err)
	}
}

// ShowCmd shows the value of the given option
func (h *BufPane) ShowCmd(args []string) {
	if len(args) < 1 {
		InfoBar.Error("Please provide an option to show")
		return
	}

	var option any
	if opt, ok := h.Buf.Settings[args[0]]; ok {
		option = opt
	} else if opt, ok := config.GlobalSettings[args[0]]; ok {
		option = opt
	}

	if option == nil {
		InfoBar.Error(args[0], " is not a valid option")
		return
	}

	InfoBar.Message(option)
}

func parseKeyArg(arg string) string {
	// If this is a raw escape sequence, convert it to its raw byte form
	return strings.ReplaceAll(arg, "\\x1b", "\x1b")
}

// ShowKeyCmd displays the action that a key is bound to
func (h *BufPane) ShowKeyCmd(args []string) {
	if len(args) < 1 {
		InfoBar.Error("Please provide a key to show")
		return
	}

	event, err := findEvent(parseKeyArg(args[0]))
	if err != nil {
		InfoBar.Error(err)
		return
	}
	if action, ok := config.Bindings["buffer"][event.Name()]; ok {
		InfoBar.Message(action)
	} else {
		InfoBar.Message(args[0], " has no binding")
	}
}

// BindCmd creates a new keybinding
func (h *BufPane) BindCmd(args []string) {
	if len(args) < 2 {
		InfoBar.Error("Not enough arguments")
		return
	}

	_, err := TryBindKey(parseKeyArg(args[0]), args[1], true, true)
	if err != nil {
		if errors.Is(err, util.ErrOverwrite) {
			screen.TermMessage(err)
		} else {
			InfoBar.Error(err)
		}
	}
}

// UnbindCmd binds a key to its default action
func (h *BufPane) UnbindCmd(args []string) {
	if len(args) < 1 {
		InfoBar.Error("Not enough arguments")
		return
	}

	err := UnbindKey(parseKeyArg(args[0]))
	if err != nil {
		if errors.Is(err, util.ErrOverwrite) {
			screen.TermMessage(err)
		} else {
			InfoBar.Error(err)
		}
	}
}

// RunCmd runs a shell command in the background
func (h *BufPane) RunCmd(args []string) {
	runf, err := shell.RunBackgroundShell(shellquote.Join(args...))
	if err != nil {
		InfoBar.Error(err)
	} else {
		go func() {
			InfoBar.Message(runf())
			screen.Redraw()
		}()
	}
}

// QuitCmd closes the main view
func (h *BufPane) QuitCmd(args []string) {
	h.Quit()
}

// GotoCmd is a command that will send the cursor to a certain
// position in the buffer
// For example: `goto line`, or `goto line:col`
func (h *BufPane) GotoCmd(args []string) {
	line, col, err := h.parseLineCol(args)
	if err != nil {
		InfoBar.Error(err)
		return
	}

	if line < 0 {
		line = h.Buf.LinesNum() + 1 + line
	}
	line = util.Clamp(line-1, 0, h.Buf.LinesNum()-1)
	col = util.Clamp(col-1, 0, util.CharacterCount(h.Buf.LineBytes(line)))

	h.RemoveAllMultiCursors()
	h.Cursor.Deselect(true)
	h.GotoLoc(buffer.Loc{col, line})
}

// JumpCmd is a command that will send the cursor to a certain relative
// position in the buffer
// For example: `jump line`, `jump -line`, or `jump -line:col`
func (h *BufPane) JumpCmd(args []string) {
	line, col, err := h.parseLineCol(args)
	if err != nil {
		InfoBar.Error(err)
		return
	}

	line = h.Buf.GetActiveCursor().Y + 1 + line
	line = util.Clamp(line-1, 0, h.Buf.LinesNum()-1)
	col = util.Clamp(col-1, 0, util.CharacterCount(h.Buf.LineBytes(line)))

	h.RemoveAllMultiCursors()
	h.Cursor.Deselect(true)
	h.GotoLoc(buffer.Loc{col, line})
}

// parseLineCol is a helper to parse the input of GotoCmd and JumpCmd
func (h *BufPane) parseLineCol(args []string) (line int, col int, err error) {
	if len(args) <= 0 {
		return 0, 0, errors.New("Not enough arguments")
	}

	line, col = 0, 0
	if strings.Contains(args[0], ":") {
		parts := strings.SplitN(args[0], ":", 2)
		line, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		col, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	} else {
		line, err = strconv.Atoi(args[0])
		if err != nil {
			return 0, 0, err
		}
	}

	return line, col, nil
}

// SaveCmd saves the buffer optionally with an argument file name
func (h *BufPane) SaveCmd(args []string) {
	if len(args) == 0 {
		h.Save()
	} else {
		h.saveBufToFile(args[0], "SaveAs", nil)
	}
}

// ReplaceCmd runs search and replace
func (h *BufPane) ReplaceCmd(args []string) {
	if len(args) < 2 || len(args) > 4 {
		// We need to find both a search and replace expression
		InfoBar.Error("Invalid replace statement: " + strings.Join(args, " "))
		return
	}

	all := false
	noRegex := false

	foundSearch := false
	foundReplace := false
	var search string
	var replaceStr string
	for _, arg := range args {
		switch arg {
		case "-a":
			all = true
		case "-l":
			noRegex = true
		default:
			if !foundSearch {
				foundSearch = true
				search = arg
			} else if !foundReplace {
				foundReplace = true
				replaceStr = arg
			} else {
				InfoBar.Error("Invalid flag: " + arg)
				return
			}
		}
	}

	if noRegex {
		search = regexp.QuoteMeta(search)
	}

	replace := []byte(replaceStr)

	var regex *regexp.Regexp
	var err error
	if h.Buf.Settings["ignorecase"].(bool) {
		regex, err = regexp.Compile("(?im)" + search)
	} else {
		regex, err = regexp.Compile("(?m)" + search)
	}
	if err != nil {
		// There was an error with the user's regex
		InfoBar.Error(err)
		return
	}

	nreplaced := 0
	start := h.Buf.Start()
	end := h.Buf.End()
	searchLoc := h.Cursor.Loc
	selection := h.Cursor.HasSelection()
	if selection {
		start = h.Cursor.CurSelection[0]
		end = h.Cursor.CurSelection[1]
		searchLoc = start // otherwise me might start at the end
	}
	if all {
		nreplaced, _ = h.Buf.ReplaceRegex(start, end, regex, replace, !noRegex)
	} else {
		inRange := func(l buffer.Loc) bool {
			return l.GreaterEqual(start) && l.LessEqual(end)
		}

		lastMatchEnd := buffer.Loc{-1, -1}
		var doReplacement func()
		doReplacement = func() {
			locs, found, err := h.Buf.FindNext(search, start, end, searchLoc, true, true)
			if err != nil {
				InfoBar.Error(err)
				return
			}
			if !found || !inRange(locs[0]) || !inRange(locs[1]) {
				h.Cursor.ResetSelection()
				h.Buf.RelocateCursors()

				return
			}

			if lastMatchEnd == locs[1] {
				// skip empty match right after previous match
				if searchLoc == end {
					searchLoc = start
					lastMatchEnd = buffer.Loc{-1, -1}
				} else {
					searchLoc = searchLoc.Move(1, h.Buf)
				}
				doReplacement()
				return
			}

			h.Cursor.SetSelectionStart(locs[0])
			h.Cursor.SetSelectionEnd(locs[1])
			h.GotoLoc(locs[0])
			h.Buf.LastSearch = search
			h.Buf.LastSearchRegex = true
			h.Buf.HighlightSearch = h.Buf.Settings["hlsearch"].(bool)

			InfoBar.YNPrompt("Perform replacement (y,n,esc)", func(yes, canceled bool) {
				if !canceled && yes {
					_, nrunes := h.Buf.ReplaceRegex(locs[0], locs[1], regex, replace, !noRegex)

					searchLoc = locs[0]
					searchLoc.X += nrunes + locs[0].Diff(locs[1], h.Buf)
					if end.Y == locs[1].Y {
						end = end.Move(nrunes, h.Buf)
					}
					h.Cursor.Loc = searchLoc
					nreplaced++
				} else if !canceled && !yes {
					searchLoc = locs[1]
				} else if canceled {
					h.Cursor.ResetSelection()
					h.Buf.RelocateCursors()
					return
				}
				lastMatchEnd = searchLoc
				doReplacement()
			})
		}
		doReplacement()
	}

	h.Buf.RelocateCursors()
	h.Relocate()

	var s string
	if nreplaced > 1 {
		s = fmt.Sprintf("Replaced %d occurrences of %s", nreplaced, search)
	} else if nreplaced == 1 {
		s = fmt.Sprintf("Replaced 1 occurrence of %s", search)
	} else {
		s = fmt.Sprintf("Nothing matched %s", search)
	}

	if selection {
		s += " in selection"
	}

	InfoBar.Message(s)
}

// ReplaceAllCmd replaces search term all at once
func (h *BufPane) ReplaceAllCmd(args []string) {
	// aliased to Replace command
	h.ReplaceCmd(append(args, "-a"))
}

func (h *BufPane) openTerm(args []string, newtab bool) {
	t := new(shell.Terminal)
	err := t.Start(args, false, true, nil, nil)
	if err != nil {
		InfoBar.Error(err)
		return
	}

	pane := 0
	id := h.ID()
	if newtab {
		h.AddTab()
		id = MainTab().Panes[pane].ID()
	} else {
		for i, p := range MainTab().Panes {
			if p.IsActive() {
				pane = i
				id = p.ID()
				p.Close()
				break
			}
		}
	}

	v := h.GetView()
	tp, err := NewTermPane(v.X, v.Y, v.Width, v.Height, t, id, MainTab())
	if err != nil {
		InfoBar.Error(err)
		return
	}
	MainTab().Panes[pane] = tp
	MainTab().SetActive(pane)
}

// TermCmd opens a terminal in the current view
func (h *BufPane) TermCmd(args []string) {
	if !TermEmuSupported {
		InfoBar.Error("Terminal emulator not supported on this system")
		return
	}

	if len(args) == 0 {
		sh := os.Getenv("SHELL")
		if sh == "" {
			InfoBar.Error("Shell environment not found")
			return
		}
		args = []string{sh}
	}

	// If there is only one open file we make a new tab instead of overwriting it
	newtab := len(MainTab().Panes) == 1 && len(Tabs.List) == 1
	if newtab {
		h.openTerm(args, true)
		return
	}

	if h.Buf.Modified() && !h.Buf.Shared() {
		h.closePrompt("Save", func() {
			h.openTerm(args, false)
		})
	} else {
		h.openTerm(args, false)
	}
}

// HandleCommand handles input from the user
func (h *BufPane) HandleCommand(input string) {
	args, err := shellquote.Split(input)
	if err != nil {
		InfoBar.Error("Error parsing args ", err)
		return
	}

	if len(args) == 0 {
		return
	}

	inputCmd := args[0]

	if _, ok := commands[inputCmd]; !ok {
		InfoBar.Error("Unknown command ", inputCmd)
	} else {
		WriteLog("> " + input + "\n")
		commands[inputCmd].action(h, args[1:])
		WriteLog("\n")
	}
}
