package action

import (
	"testing"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/stretchr/testify/assert"
)

// Plugins can set options from preinit or onBufferOpen, before Tabs exists.
func TestSetGlobalOptionBeforeTabs(t *testing.T) {
	config.InitRuntimeFiles(false)
	if err := config.InitGlobalSettings(); err != nil {
		t.Fatal(err)
	}

	saved := Tabs
	defer func() { Tabs = saved }()
	Tabs = nil

	// Local options are set on the current pane's buffer, and there is none.
	assert.Equal(t, ErrNoPane, SetGlobalOptionPlug("filetype", "go"))

	// Layout options resize the tabs, which do not exist yet.
	assert.NoError(t, SetGlobalOptionPlug("tabalways", "true"))
	assert.Equal(t, true, config.GlobalSettings["tabalways"])
}
