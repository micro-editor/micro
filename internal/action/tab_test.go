package action

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/buffer"
	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	lua "github.com/yuin/gopher-lua"
)

func init() {
	ulua.L = lua.NewState()
	config.InitRuntimeFiles(false)
	config.InitGlobalSettings()
	config.GlobalSettings["backup"] = false
	config.GlobalSettings["fastdirty"] = true
}

func newTestTab() *Tab {
	return NewTabFromBuffer(0, 0, 80, 24, buffer.NewBufferFromString("", "", buffer.BTDefault))
}

func TestTabIDWithSplits(t *testing.T) {
	tab1 := newTestTab()
	tab2 := newTestTab()

	id := tab1.ID()
	if id == 0 {
		t.Fatal("tab id is 0")
	}
	if tab2.ID() == id {
		t.Fatalf("tabs share id %d", id)
	}

	tab1.VSplit(true)
	if got := tab1.ID(); got != id {
		t.Errorf("tab id changed from %d to %d after split", id, got)
	}
}
