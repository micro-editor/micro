//go:build linux || darwin || dragonfly || solaris || openbsd || netbsd || freebsd

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// notifyReload relays SIGUSR1 to c, so that external tools (e.g. theme
// switchers) can ask a running micro to reload its configuration.
func notifyReload(c chan os.Signal) {
	signal.Notify(c, syscall.SIGUSR1)
}
