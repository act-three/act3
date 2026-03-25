package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	TableGhost   = attr.Attr("data-table")("ghost")
	TableSurface = attr.Attr("data-table")("surface")
)

var (
	TableSize1 = attr.Attr("data-table-size")("1")
	TableSize2 = attr.Attr("data-table-size")("2")
	TableSize3 = attr.Attr("data-table-size")("3")
)

func TableRoot(attrs ...attr.Node) html.Element {
	return html.Table(
		attr.Class("u-table"),
		attr.Group(attrs...),
	)
}

func TableHeader(attrs ...attr.Node) html.Element {
	return html.Thead(
		attr.Class("u-table-header"),
		attr.Group(attrs...),
	)
}

func TableBody(attrs ...attr.Node) html.Element {
	return html.Tbody(
		attr.Class("u-table-body"),
		attr.Group(attrs...),
	)
}

func TableRow(attrs ...attr.Node) html.Element {
	return html.Tr(
		attr.Class("u-table-row"),
		attr.Group(attrs...),
	)
}

func TableHead(attrs ...attr.Node) html.Element {
	return html.Th(
		attr.Class("u-table-head"),
		attr.Group(attrs...),
	)
}

func TableCell(attrs ...attr.Node) html.Element {
	return html.Td(
		attr.Class("u-table-cell"),
		attr.Group(attrs...),
	)
}
