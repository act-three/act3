package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	TableSize1 = Attr("data-table-size")("1")
	TableSize2 = Attr("data-table-size")("2")
	TableSize3 = Attr("data-table-size")("3")
)

func TableRoot(attrs ...attr.Node) html.Element {
	return html.Table(
		Class("u-table"),
		group(attrs...),
	)
}

func TableHeader(attrs ...attr.Node) html.Element {
	return html.Thead(
		Class("u-table-header"),
		group(attrs...),
	)
}

func TableBody(attrs ...attr.Node) html.Element {
	return html.Tbody(
		Class("u-table-body"),
		group(attrs...),
	)
}

func TableRow(attrs ...attr.Node) html.Element {
	return html.Tr(
		Class("u-table-row"),
		group(attrs...),
	)
}

func TableHead(attrs ...attr.Node) html.Element {
	return html.Th(
		Class("u-table-head"),
		group(attrs...),
	)
}

func TableCell(attrs ...attr.Node) html.Element {
	return html.Td(
		Class("u-table-cell"),
		group(attrs...),
	)
}
