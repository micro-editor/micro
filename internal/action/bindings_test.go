package action

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/micro-editor/micro/v2/internal/config"
)

func TestDefaultMultiCursorMouseBindings(t *testing.T) {
	bindings := DefaultBindings("buffer")
	for _, key := range []string{"Ctrl-MouseLeft", "Alt-MouseLeft"} {
		if got := bindings[key]; got != "MouseMultiCursor" {
			t.Errorf("%s binding = %q, want MouseMultiCursor", key, got)
		}
	}
}

func TestMultiCursorMouseBindingOverride(t *testing.T) {
	oldConfigDir, oldBinder := config.ConfigDir, Binder
	config.ConfigDir = t.TempDir()
	t.Cleanup(func() {
		config.ConfigDir, Binder = oldConfigDir, oldBinder
	})

	bindings := map[string]string{}
	Binder = map[string]func(Event, string){
		"buffer": func(event Event, action string) { bindings[event.Name()] = action },
	}
	for _, contents := range []string{
		`{"Alt-MouseLeft": "MousePress"}`,
		`{"buffer": {"Alt-MouseLeft": "MousePress"}}`,
	} {
		if err := os.WriteFile(filepath.Join(config.ConfigDir, "bindings.json"), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		InitBindings()
		if got := bindings["Alt-MouseLeft"]; got != "MousePress" {
			t.Errorf("Alt-MouseLeft override = %q, want MousePress", got)
		}
		if got := bindings["Ctrl-MouseLeft"]; got != "MouseMultiCursor" {
			t.Errorf("Ctrl-MouseLeft binding = %q, want MouseMultiCursor", got)
		}
	}
}
