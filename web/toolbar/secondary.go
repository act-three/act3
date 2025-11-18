package toolbar

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
)

func Secondary(attrs ...attr.Node) html.Element {
	return html.Header(
		attr.Class(`
			flex h-12 shrink-0 items-center content-center
			justify-between gap-4 border-b px-4
		`),
		attr.Group(attrs...),
	).
		With(ButtonSM).
		With(ButtonBordered).(html.Element)
}
