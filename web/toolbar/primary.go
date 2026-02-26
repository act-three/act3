package toolbar

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	. "ily.dev/act3/ui"
)

func Primary(attrs ...attr.Node) html.Element {
	return html.Header(
		attr.Class(`
			flex h-16 shrink-0 items-center content-center
			justify-between gap-4 border-b border-gray-6 px-4
		`),
		attr.Group(attrs...),
	).
		With(ButtonSurface).(html.Element)
}
