package main

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/action"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

// callMicroFn calls micro.<name>() and returns its single result.
func callMicroFn(t *testing.T, pkg *lua.LTable, name string) lua.LValue {
	t.Helper()

	err := ulua.L.CallByParam(lua.P{
		Fn:      ulua.L.GetField(pkg, name),
		NRet:    1,
		Protect: true,
	})
	if err != nil {
		t.Fatalf("micro.%s(): %v", name, err)
	}

	got := ulua.L.Get(-1)
	ulua.L.Pop(1)
	return got
}

// Plugin hooks can run before InitTabs, so these wrappers must return nil
// instead of dereferencing the nil tab that MainTab() returns.
func TestLuaCurrentPaneAndTab(t *testing.T) {
	savedTabs := action.Tabs
	t.Cleanup(func() { action.Tabs = savedTabs })

	for _, tc := range []struct {
		name    string
		tabs    *action.TabList
		wantNil bool
	}{
		// TestMain has already run startup, so savedTabs is a live tab list.
		{"initialized", savedTabs, false},
		{"uninitialized", nil, true},
		{"empty", &action.TabList{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			action.Tabs = tc.tabs
			pkg := luaImportMicro()
			assert.Equal(t, tc.wantNil, callMicroFn(t, pkg, "CurPane") == lua.LNil)
			assert.Equal(t, tc.wantNil, callMicroFn(t, pkg, "CurTab") == lua.LNil)
		})
	}
}
