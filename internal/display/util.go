package display

import (
	"sort"
	"strings"

	"github.com/zyedidia/micro/v2/internal/config"
)

// try to make a keybinding more "friendly"
// specifically:
// all letters are capitalized
// Ctrl- becomes ^
// Alt- becomes !
func friendlyBinding(k string) string {
	k = strings.ToUpper(k)
	k = strings.Replace(k, "CTRL-", "^", -1)
	k = strings.Replace(k, "ALT-", "!", -1)
	return k
}

// find a buffer binding by key
// has stable results
// "friendli-fies" the result if set to do so
// returns "??" if none found
func findBinding(action string, friendly bool) string {
	cat := "buffer"
	// sort the keys
	// if you don't sort the keys, the order of iterating a map is not guaranteed
	// this means that with >1 binding for a given command
	// which one you get is random
	keys := make([]string, len(config.Bindings[cat]))
	i := 0
	for k, _ := range config.Bindings[cat] {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	for _, k := range keys {
		if config.Bindings[cat][k] == action {
			if friendly {
				return friendlyBinding(k)
			} else {
				return k
			}
		}
	}
	return "??"

}
