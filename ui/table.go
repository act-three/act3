package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

var (
	TableSize1 = Attr("data-table-size")("1")
	TableSize2 = Attr("data-table-size")("2")
	TableSize3 = Attr("data-table-size")("3")
)

func TableRoot(attrs ...domi.Attr) domi.Element {
	return html.Table(
		Class("u-table"),
		group(attrs...),
	)
}

func TableHeader(attrs ...domi.Attr) domi.Element {
	return html.THead(
		Class("u-table-header"),
		group(attrs...),
	)
}

func TableBody(attrs ...domi.Attr) domi.Element {
	return html.TBody(
		Class("u-table-body"),
		group(attrs...),
	)
}

func TableRow(attrs ...domi.Attr) domi.Element {
	return html.TR(
		Class("u-table-row"),
		group(attrs...),
	)
}

func TableHead(attrs ...domi.Attr) domi.Element {
	return html.TH(
		Class("u-table-head"),
		group(attrs...),
	)
}

func TableCell(attrs ...domi.Attr) domi.Element {
	return html.TD(
		Class("u-table-cell"),
		group(attrs...),
	)
}
