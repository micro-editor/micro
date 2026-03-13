package buffer

import (
	"crypto/md5"
	"reflect"

	"github.com/micro-editor/micro/v2/internal/config"
	ulua "github.com/micro-editor/micro/v2/internal/lua"
	"github.com/micro-editor/micro/v2/internal/screen"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
	luar "layeh.com/gopher-luar"
)

// settingAsString safely converts a settings value to string.
// If the value is not a string (e.g. a map from a glob/ft config entry),
// it falls back to the provided default. This prevents TypeAssertionError
// panics when config values are unexpectedly non-string (see issue #4042).
func settingAsString(v any, defaultVal string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return defaultVal
}

func (b *Buffer) ReloadSettings(reloadFiletype bool) {
	settings := config.ParsedSettings()
	config.UpdatePathGlobLocals(settings, b.AbsPath)

	oldFiletype := settingAsString(b.Settings["filetype"], "unknown")

	_, local := b.LocalSettings["filetype"]
	_, volatile := config.VolatileSettings["filetype"]
	if reloadFiletype && !local && !volatile {
		// need to update filetype before updating other settings based on it
		b.Settings["filetype"] = "unknown"
		if v, ok := settings["filetype"]; ok {
			// Only accept string values for filetype; non-string values
			// (e.g. a map from a glob config block) are silently ignored.
			if ft, ok := v.(string); ok {
				b.Settings["filetype"] = ft
			}
		}
	}

	// update syntax rules, which will also update filetype if needed
	b.UpdateRules()

	curFiletype := settingAsString(b.Settings["filetype"], "unknown")
	if oldFiletype != curFiletype {
		b.doCallbacks("filetype", oldFiletype, curFiletype)
	}

	config.UpdateFileTypeLocals(settings, curFiletype)

	for k, v := range config.DefaultCommonSettings() {
		if k == "filetype" {
			// prevent recursion
			continue
		}
		if _, ok := config.VolatileSettings[k]; ok {
			// reload should not override volatile settings
			continue
		}
		if _, ok := b.LocalSettings[k]; ok {
			// reload should not override local settings
			continue
		}
		if _, ok := settings[k]; ok {
			b.DoSetOptionNative(k, settings[k])
		} else {
			b.DoSetOptionNative(k, v)
		}
	}
}

func (b *Buffer) DoSetOptionNative(option string, nativeValue any) {
	oldValue := b.Settings[option]
	if reflect.DeepEqual(oldValue, nativeValue) {
		return
	}

	b.Settings[option] = nativeValue

	if option == "fastdirty" {
		if !nativeValue.(bool) {
			if b.Size() > LargeFileThreshold {
				b.Settings["fastdirty"] = true
			} else {
				if !b.isModified {
					b.calcHash(&b.origHash)
				} else {
					// prevent using an old stale origHash value
					b.origHash = [md5.Size]byte{}
				}
			}
		}
	} else if option == "statusline" {
		screen.Redraw()
	} else if option == "filetype" {
		b.ReloadSettings(false)
	} else if option == "fileformat" {
		switch b.Settings["fileformat"].(string) {
		case "unix":
			b.Endings = FFUnix
		case "dos":
			b.Endings = FFDos
		}
		b.setModified()
	} else if option == "syntax" {
		if !nativeValue.(bool) {
			b.ClearMatches()
		} else {
			b.UpdateRules()
		}
	} else if option == "encoding" {
		enc, err := htmlindex.Get(b.Settings["encoding"].(string))
		if err != nil {
			enc = unicode.UTF8
			b.Settings["encoding"] = "utf-8"
		}
		b.encoding = enc
		b.setModified()
	} else if option == "readonly" && b.Type.Kind == BTDefault.Kind {
		b.Type.Readonly = nativeValue.(bool)
	} else if option == "hlsearch" {
		for _, buf := range OpenBuffers {
			if b.SharedBuffer == buf.SharedBuffer {
				buf.HighlightSearch = nativeValue.(bool)
			}
		}
	} else {
		for _, pl := range config.Plugins {
			if option == pl.Name {
				if nativeValue.(bool) {
					if !pl.Loaded {
						pl.Load()
					}
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

	b.doCallbacks(option, oldValue, nativeValue)
}

func (b *Buffer) SetOptionNative(option string, nativeValue any) error {
	if err := config.OptionIsValid(option, nativeValue); err != nil {
		return err
	}

	b.DoSetOptionNative(option, nativeValue)
	b.LocalSettings[option] = true

	return nil
}

// SetOption sets a given option to a value just for this buffer
func (b *Buffer) SetOption(option, value string) error {
	if _, ok := b.Settings[option]; !ok {
		return config.ErrInvalidOption
	}

	nativeValue, err := config.GetNativeValue(option, value)
	if err != nil {
		return err
	}

	return b.SetOptionNative(option, nativeValue)
}

func (b *Buffer) doCallbacks(option string, oldValue any, newValue any) {
	if b.OptionCallback != nil {
		b.OptionCallback(option, newValue)
	}

	if err := config.RunPluginFn("onBufferOptionChanged",
		luar.New(ulua.L, b), luar.New(ulua.L, option),
		luar.New(ulua.L, oldValue), luar.New(ulua.L, newValue)); err != nil {
		screen.TermMessage(err)
	}
}
