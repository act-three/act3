package table

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Root(attrs ...attr.Node) html.Element {
	return html.Table(
		attr.Class("caption-bottom text-sm"),
		attr.Group(attrs...),
	)
}

func Header(attrs ...attr.Node) html.Element {
	return html.Thead(
		attr.Class("[&_tr]:border-b"),
		attr.Group(attrs...),
	)
}

func Body(attrs ...attr.Node) html.Element {
	return html.Tbody(
		attr.Class("[&_tr:last-child]:border-0"),
		attr.Group(attrs...),
	)
}

func Row(attrs ...attr.Node) html.Element {
	return html.Tr(
		attr.Class(`
			hover:[&,&>svelte-css-wrapper]:[&>th,td]:bg-gray-2/50
			data-[state=selected]:bg-gray-2
			border-b
			transition-colors
		`),
		attr.Group(attrs...),
	)
}

func Head(attrs ...attr.Node) html.Element {
	return html.Th(
		attr.Class(`
			text-gray-12
			h-10
			bg-clip-padding
			px-2
			text-start
			align-middle
			font-medium
			whitespace-nowrap
			[&:has([role=checkbox])]:pe-0
			w-[100px]
		`),
		attr.Group(attrs...),
	)
}

func Cell(attrs ...attr.Node) html.Element {
	return html.Td(
		attr.Class(`
			bg-clip-padding
			p-2
			align-middle
			whitespace-nowrap
			[&:has([role=checkbox])]:pe-0
		`),
		attr.Group(attrs...),
	)
}
