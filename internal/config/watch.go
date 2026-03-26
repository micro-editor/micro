package config

import (
	"os"
	"time"
)

// WatchedFile describes a file to watch and the callback to invoke on modification.
type WatchedFile struct {
	Path     func() string // resolves to the current absolute path each tick
	OnChange func()        // called when modified; runs on a background goroutine
}

// StartConfigWatcher checks each WatchedFile once per second and calls its
// OnChange when the file is modified or the resolved path changes.
// OnChange runs on a background goroutine — post to a channel rather than
// mutating state directly.
// The returned function stops the watcher.
func StartConfigWatcher(files []WatchedFile) func() {
	stop := make(chan struct{})

	go func() {
		type fileState struct {
			path    string
			lastMod time.Time
		}
		states := make([]fileState, len(files))
		for i, f := range files {
			p := f.Path()
			states[i].path = p
			if fi, err := os.Stat(p); err == nil {
				states[i].lastMod = fi.ModTime()
			}
		}

		for {
			time.Sleep(1 * time.Second)
			select {
			case <-stop:
				return
			default:
			}
			for i, f := range files {
				newPath := f.Path()
				fi, err := os.Stat(newPath)
				if err != nil {
					continue
				}
				if newPath != states[i].path || fi.ModTime().After(states[i].lastMod) {
					states[i].path = newPath
					states[i].lastMod = fi.ModTime()
					f.OnChange()
				}
			}
		}
	}()

	return func() { close(stop) }
}
