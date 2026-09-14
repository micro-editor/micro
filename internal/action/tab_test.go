package action

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// MainTab must return nil, not panic, before Tabs is initialized.
func TestMainTabHandlesUninitializedTabs(t *testing.T) {
	saved := Tabs
	defer func() { Tabs = saved }()

	Tabs = nil
	assert.Nil(t, MainTab(), "MainTab() with nil Tabs")

	Tabs = &TabList{}
	assert.Nil(t, MainTab(), "MainTab() with empty Tabs.List")
}
