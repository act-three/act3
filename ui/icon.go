package ui

import (
	"cmp"
	"fmt"

	"ily.dev/domi"

	"ily.dev/act3/ui/icon"
)

func HiIcon(name string) domi.Node {
	return Icon(cmp.Or(hiIconNames[name], name))
}

func Icon(name string) domi.Node {
	n, err := domi.UnsafeParseRaw(icon.SVG(name))
	if err != nil {
		panic(fmt.Errorf("ui: parse icon %q: %w", name, err))
	}
	return n
}

var hiIconNames = map[string]string{
	"x": "line/x-close",
}
