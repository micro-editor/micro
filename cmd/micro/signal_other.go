//go:build plan9 || nacl || windows

package main

import "os"

// notifyReload is a no-op: SIGUSR1 is not available on this platform.
func notifyReload(c chan os.Signal) {}
