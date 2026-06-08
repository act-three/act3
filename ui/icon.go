package ui

import (
	"fmt"

	"ily.dev/domi"

	"ily.dev/act3/ui/icon"
)

func Icon(name string) domi.Node {
	n, err := domi.UnsafeParseRaw(icon.SVG(name))
	if err != nil {
		panic(fmt.Errorf("ui: parse icon %q: %w", name, err))
	}
	return n
}
