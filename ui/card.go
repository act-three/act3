package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Card(attrs ...attr.Node) html.Element {
	sizeClass := "p-3 rounded-lg [--inset-padding:-12px] [--inset-rounded:8px]"
	switch getOption[sizeOption](attrs, 1) {
	case 2:
		sizeClass = "p-4 rounded-lg [--inset-padding:-16px] [--inset-rounded:8px]"
	case 3:
		sizeClass = "p-6 rounded-xl [--inset-padding:-24px] [--inset-rounded:12px]"
	}

	return html.Div(
		Class(sizeClass),
		Class(`
			bg-gray-1/70
			inset-ring
			inset-ring-gray-6
		`),
		group(attrs...),
	)
}
