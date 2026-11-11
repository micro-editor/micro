package main

import (
	"reflect"
	"testing"

	"github.com/micro-editor/micro/v2/internal/action"
	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/tcell/v2"
)

func TestCommandBarActionChains(t *testing.T) {
	oldInfo, oldBuf := action.InfoBindings, action.InfoBufBindings
	oldConfig := config.Bindings["command"]
	defer func() {
		action.InfoBindings, action.InfoBufBindings = oldInfo, oldBuf
		config.Bindings["command"] = oldConfig
	}()

	var calls []string
	for name, success := range map[string]bool{"TestChainTrue": true, "TestChainFalse": false, "TestChainNext": true} {
		name, success := name, success
		old, existed := action.BufKeyActions[name]
		action.BufKeyActions[name] = func(p *action.BufPane) bool {
			if p != action.InfoBar.BufPane {
				t.Error("command action escaped the command buffer")
			}
			calls = append(calls, name)
			return success
		}
		defer func() {
			if existed {
				action.BufKeyActions[name] = old
			} else {
				delete(action.BufKeyActions, name)
			}
		}()
	}
	const infoAction = "TestChainInfo"
	old, existed := action.InfoKeyActions[infoAction]
	action.InfoKeyActions[infoAction] = func(p *action.InfoPane) {
		if p != action.InfoBar {
			t.Error("wrong command pane")
		}
		calls = append(calls, infoAction)
	}
	defer func() {
		if existed {
			action.InfoKeyActions[infoAction] = old
		} else {
			delete(action.InfoKeyActions, infoAction)
		}
	}()

	for _, tc := range []struct {
		name, binding  string
		want           []string
		closed, reopen bool
	}{
		{"comma", "TestChainFalse,TestChainNext", []string{"TestChainFalse", "TestChainNext"}, false, false},
		{"and failure", "TestChainFalse&TestChainNext", []string{"TestChainFalse"}, false, false},
		{"and success", "TestChainTrue&TestChainNext", []string{"TestChainTrue", "TestChainNext"}, false, false},
		{"or success", "TestChainTrue|TestChainNext", []string{"TestChainTrue"}, false, false},
		{"or failure", "TestChainFalse|TestChainNext", []string{"TestChainFalse", "TestChainNext"}, false, false},
		{"mixed actions", "TestChainInfo,TestChainNext", []string{"TestChainInfo", "TestChainNext"}, false, false},
		{"abort", "AbortCommand,TestChainNext", nil, true, false},
		{"execute", "ExecuteCommand,TestChainNext", nil, true, false},
		{"abort opens another prompt", "AbortCommand,TestChainNext", nil, false, true},
		{"unknown action", "TestChainTrue,NoSuchChainAction", nil, false, false},
		{"single buffer action", "TestChainNext", []string{"TestChainNext"}, false, false},
		{"single info action", "TestChainInfo", []string{"TestChainInfo"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			action.InfoBindings = action.NewKeyTree()
			action.InfoBufBindings = action.InfoBindings
			config.Bindings["command"] = make(map[string]string)
			calls = nil
			action.InfoBar.Prompt("test: ", "one two", "chain-test", nil, func(string, bool) {
				if tc.reopen {
					action.InfoBar.Prompt("replacement: ", "untouched", "chain-replacement", nil, func(string, bool) {})
				}
			})
			key, err := action.ConstructEvent(tcell.NewEventKey(tcell.KeyF12, 0, tcell.ModNone, ""))
			if err != nil {
				t.Fatal("F12 event unavailable")
			}
			// Rebinding must replace a chain even when the new binding is a
			// single buffer action.
			if tc.name != "unknown action" {
				action.InfoMapEvent(key, "TestChainInfo,TestChainNext")
			}
			action.InfoMapEvent(key, tc.binding)
			injectKey(tcell.KeyF12, 0, tcell.ModNone)
			if !reflect.DeepEqual(calls, tc.want) {
				t.Errorf("calls = %v, want %v", calls, tc.want)
			}
			if action.InfoBar.HasPrompt == tc.closed {
				t.Errorf("HasPrompt = %v, closed = %v", action.InfoBar.HasPrompt, tc.closed)
			}
			if tc.reopen && string(action.InfoBar.LineBytes(0)) != "untouched" {
				t.Error("replacement prompt was edited by the old chain")
			}
			if action.InfoBar.HasPrompt {
				action.InfoBar.PromptCallback = func(string, bool) {}
				action.InfoBar.DonePrompt(true)
			}
		})
	}
}

func TestCommandBarChainedEditing(t *testing.T) {
	oldInfo, oldBuf := action.InfoBindings, action.InfoBufBindings
	oldConfig := config.Bindings["command"]
	defer func() {
		action.InfoBindings, action.InfoBufBindings = oldInfo, oldBuf
		config.Bindings["command"] = oldConfig
	}()
	action.InfoBindings = action.NewKeyTree()
	action.InfoBufBindings = action.InfoBindings
	config.Bindings["command"] = make(map[string]string)
	key, _ := action.ConstructEvent(tcell.NewEventKey(tcell.KeyF12, 0, tcell.ModNone, ""))
	action.InfoMapEvent(key, "StartOfText,DeleteWordRight")
	document := action.MainTab().CurPane().Buf
	before := string(document.Bytes())
	action.InfoBar.Prompt("test: ", "one two", "chain-edit-test", nil, func(string, bool) {})
	defer action.InfoBar.DonePrompt(true)
	injectKey(tcell.KeyF12, 0, tcell.ModNone)
	if got := string(action.InfoBar.LineBytes(0)); got != " two" {
		t.Errorf("command buffer = %q, want %q", got, " two")
	}
	if string(document.Bytes()) != before {
		t.Error("command-bar chain changed the document")
	}
}

func TestBufferChainedEditingControl(t *testing.T) {
	oldBindings, oldConfig := action.BufBindings, config.Bindings["buffer"]
	defer func() { action.BufBindings, config.Bindings["buffer"] = oldBindings, oldConfig }()
	file := createTestFile(t, "one two")
	openFile(file)
	action.BufBindings = action.NewKeyTree()
	config.Bindings["buffer"] = make(map[string]string)
	action.BindKey("F12", "StartOfText,DeleteWordRight", action.BufMapEvent)
	injectKey(tcell.KeyF12, 0, tcell.ModNone)
	if got := string(action.MainTab().CurPane().Buf.LineBytes(0)); got != " two" {
		t.Errorf("document = %q, want %q", got, " two")
	}
	if !action.MainTab().CurPane().Save() {
		t.Fatal("could not save control fixture")
	}
}

func TestCommandBarExistingEventDispatch(t *testing.T) {
	oldInfo, oldBuf, oldBar := action.InfoBindings, action.InfoBufBindings, action.InfoBar
	oldConfig := config.Bindings["command"]
	action.InfoBindings = action.NewKeyTree()
	action.InfoBufBindings = action.InfoBindings
	config.Bindings["command"] = make(map[string]string)
	action.InfoBar = action.NewInfoBar()
	defer func() {
		action.InfoBar.Close()
		action.InfoBindings, action.InfoBufBindings, action.InfoBar = oldInfo, oldBuf, oldBar
		config.Bindings["command"] = oldConfig
	}()

	raw := tcell.NewEventRaw("\x1b[999~")
	mouse := tcell.NewEventMouse(1, 1, tcell.WheelRight, tcell.ModNone, "")
	key := tcell.NewEventKey(tcell.KeyF11, 0, tcell.ModNone, "")
	for _, tc := range []struct {
		name, sequence, binding string
		events                  []tcell.Event
		wantX                   int
	}{
		{"raw single edit", "", "StartOfText", []tcell.Event{raw}, 0},
		{"mouse single edit", "", "StartOfText", []tcell.Event{mouse}, 0},
		{"mixed sequence single edit", "<F11><MouseWheelRight>", "StartOfText", []tcell.Event{key, mouse}, 0},
		{"unsupported raw chain", "", "StartOfText,DeleteWordRight", []tcell.Event{tcell.NewEventRaw("\x1b[1000~")}, 7},
		{"unsupported mouse chain", "", "StartOfText,DeleteWordRight", []tcell.Event{tcell.NewEventMouse(1, 1, tcell.WheelLeft, tcell.ModNone, "")}, 7},
		{"unsupported mixed sequence chain", "<F10><MouseWheelLeft>", "StartOfText,DeleteWordRight", []tcell.Event{tcell.NewEventKey(tcell.KeyF10, 0, tcell.ModNone, ""), tcell.NewEventMouse(1, 1, tcell.WheelLeft, tcell.ModNone, "")}, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.sequence != "" {
				action.BindKey(tc.sequence, tc.binding, action.InfoMapEvent)
			} else {
				event, err := action.ConstructEvent(tc.events[0])
				if err != nil {
					t.Fatal(err)
				}
				action.InfoMapEvent(event, tc.binding)
			}
			action.InfoBar.Prompt("test: ", "one two", "event-dispatch-test", nil, func(string, bool) {})
			defer action.InfoBar.DonePrompt(true)
			if action.InfoBar.Cursor.X != 7 {
				t.Fatalf("initial cursor = %d, want 7", action.InfoBar.Cursor.X)
			}
			for _, event := range tc.events {
				action.InfoBar.HandleEvent(event)
			}
			if got := action.InfoBar.Cursor.X; got != tc.wantX {
				t.Errorf("cursor = %d, want %d", got, tc.wantX)
			}
		})
	}
}

func TestCommandBarSharedEventPrefixes(t *testing.T) {
	oldInfo, oldBuf, oldBar := action.InfoBindings, action.InfoBufBindings, action.InfoBar
	oldConfig := config.Bindings["command"]
	defer func() {
		action.InfoBindings, action.InfoBufBindings, action.InfoBar = oldInfo, oldBuf, oldBar
		config.Bindings["command"] = oldConfig
	}()
	key := func(k tcell.Key) tcell.Event { return tcell.NewEventKey(k, 0, tcell.ModNone, "") }
	mouse := func(b tcell.ButtonMask) tcell.Event { return tcell.NewEventMouse(1, 1, b, tcell.ModNone, "") }
	for _, tc := range []struct {
		name     string
		bindings map[string]string
		events   []tcell.Event
		wantText string
		wantX    int
	}{
		{"keyboard and mixed sequence", map[string]string{"<F11><F12>": "StartOfText", "<F11><MouseWheelRight>": "StartOfText"}, []tcell.Event{key(tcell.KeyF11), mouse(tcell.WheelRight)}, "one two", 0},
		{"mouse prefix and standalone key", map[string]string{"F11": "EndOfLine", "<MouseWheelRight><F11>": "StartOfText"}, []tcell.Event{mouse(tcell.WheelRight), key(tcell.KeyF11)}, "one two", 0},
		{"chain and mixed sequence", map[string]string{"<F11><F12>": "EndOfLine,StartOfText", "<F11><MouseWheelRight>": "StartOfText"}, []tcell.Event{key(tcell.KeyF11), mouse(tcell.WheelRight)}, "one two", 0},
		{"mouse prefix and standalone chain", map[string]string{"F11": "EndOfLine,DeleteWordLeft", "<MouseWheelRight><F11>": "StartOfText"}, []tcell.Event{mouse(tcell.WheelRight), key(tcell.KeyF11)}, "one two", 0},
		{"chain after mouse action", map[string]string{"MouseWheelRight": "EndOfLine", "F12": "StartOfText,DeleteWordRight"}, []tcell.Event{mouse(tcell.WheelRight), key(tcell.KeyF12)}, " two", 0},
		{"unmatched mouse clears keyboard prefix", map[string]string{"<F11><F12>": "EndOfLine", "F12": "StartOfText,DeleteWordRight"}, []tcell.Event{key(tcell.KeyF11), mouse(tcell.WheelLeft), key(tcell.KeyF12)}, " two", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			action.InfoBindings = action.NewKeyTree()
			action.InfoBufBindings = action.InfoBindings
			config.Bindings["command"] = make(map[string]string)
			action.InfoBar = action.NewInfoBar()
			defer action.InfoBar.Close()
			for key, binding := range tc.bindings {
				action.BindKey(key, binding, action.InfoMapEvent)
			}
			action.InfoBar.Prompt("test: ", "one two", "shared-prefix-test", nil, func(string, bool) {})
			defer action.InfoBar.DonePrompt(true)
			for _, event := range tc.events {
				action.InfoBar.HandleEvent(event)
			}
			if got := string(action.InfoBar.LineBytes(0)); got != tc.wantText {
				t.Errorf("text = %q, want %q", got, tc.wantText)
			}
			if got := action.InfoBar.Cursor.X; got != tc.wantX {
				t.Errorf("cursor = %d, want %d", got, tc.wantX)
			}
		})
	}
}

func TestCommandBarAdaptersRejectDocumentTarget(t *testing.T) {
	oldInfo, oldBuf := action.InfoBindings, action.InfoBufBindings
	oldConfig := config.Bindings["command"]
	defer func() {
		action.InfoBindings, action.InfoBufBindings = oldInfo, oldBuf
		config.Bindings["command"] = oldConfig
	}()
	action.InfoBindings = action.NewKeyTree()
	action.InfoBufBindings = action.InfoBindings
	config.Bindings["command"] = make(map[string]string)
	key, err := action.ConstructEvent(tcell.NewEventKey(tcell.KeyF12, 0, tcell.ModNone, ""))
	if err != nil {
		t.Fatal(err)
	}
	document := action.MainTab().CurPane()
	before := string(document.Buf.Bytes())
	for _, binding := range []string{"HistoryUp", "StartOfText,DeleteWordRight"} {
		action.InfoMapEvent(key, binding)
		fn, more := action.InfoBindings.NextEvent(key, nil)
		if fn == nil || more {
			t.Fatal("missing standalone binding")
		}
		if fn(document) {
			t.Error("command adapter accepted document pane")
		}
		action.InfoBindings.ResetEvents()
	}
	if string(document.Buf.Bytes()) != before {
		t.Error("rejected command adapter changed document")
	}
}
