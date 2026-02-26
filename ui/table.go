package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func TableRoot(attrs ...attr.Node) html.Element {
	return html.Table(
		attr.Class("a$table"),
		attr.Group(attrs...),
	)
}

func TableHeader(attrs ...attr.Node) html.Element {
	return html.Thead(
		attr.Class("a$table-header"),
		attr.Group(attrs...),
	)
}

func TableBody(attrs ...attr.Node) html.Element {
	return html.Tbody(
		attr.Class("a$table-body"),
		attr.Group(attrs...),
	)
}

func TableRow(attrs ...attr.Node) html.Element {
	return html.Tr(
		attr.Class("a$table-row"),
		attr.Group(attrs...),
	)
}

func TableHead(attrs ...attr.Node) html.Element {
	return html.Th(
		attr.Class("a$table-head"),
		attr.Group(attrs...),
	)
}

func TableCell(attrs ...attr.Node) html.Element {
	return html.Td(
		attr.Class("a$table-cell"),
		attr.Group(attrs...),
	)
}
