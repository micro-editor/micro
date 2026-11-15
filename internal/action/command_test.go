package action

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/micro-editor/micro/v2/internal/config"
)

func TestSetGlobalOptionPlugWritesSettingsWhenEnabled(t *testing.T) {
	oldConfigDir := config.ConfigDir
	oldGlobalSettings := config.GlobalSettings
	oldModifiedSettings := config.ModifiedSettings
	oldVolatileSettings := config.VolatileSettings
	defer func() {
		config.ConfigDir = oldConfigDir
		config.GlobalSettings = oldGlobalSettings
		config.ModifiedSettings = oldModifiedSettings
		config.VolatileSettings = oldVolatileSettings
	}()

	config.ConfigDir = t.TempDir()
	config.ModifiedSettings = make(map[string]bool)
	config.VolatileSettings = make(map[string]bool)
	if err := config.ReadSettings(); err != nil {
		t.Fatal(err)
	}
	if err := config.InitGlobalSettings(); err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(config.ConfigDir, "settings.json")
	if err := SetGlobalOptionPlug("showchars", "session-only"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Fatalf("plugin option change created settings.json: %v", err)
	}

	if err := SetGlobalOptionPlug("writesettings", "plugins"); err != nil {
		t.Fatal(err)
	}
	if err := SetGlobalOptionPlug("showchars", "persisted"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if got, want := settings["showchars"], "persisted"; got != want {
		t.Fatalf("showchars = %v, want %q", got, want)
	}
	if got, want := settings["writesettings"], "plugins"; got != want {
		t.Fatalf("writesettings = %v, want %q", got, want)
	}
}
