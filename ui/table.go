package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

type tableVariant int

const (
	tableGhost   tableVariant = iota // default
	tableSurface
)

type tableSize int

const (
	tableSize2 tableSize = iota // default
	tableSize1
	tableSize3
)

var (
	TableGhost   = html.WithValue(tableVariantKey, tableGhost)
	TableSurface = html.WithValue(tableVariantKey, tableSurface)
)

var (
	TableSize1 = html.WithValue(tableSizeKey, tableSize1)
	TableSize2 = html.WithValue(tableSizeKey, tableSize2)
	TableSize3 = html.WithValue(tableSizeKey, tableSize3)
)

var tableVariantClasses = map[tableVariant]string{
	tableGhost:   "u-table+ghost",
	tableSurface: "u-table+surface",
}

var tableSizeClasses = map[tableSize]string{
	tableSize1: "u-table+size-1",
	tableSize2: "u-table+size-2",
	tableSize3: "u-table+size-3",
}

func TableRoot(attrs ...attr.Node) html.Element {
	return html.Table(
		attr.Class("u-table"),
		attr.FuncAttr("class", func(get func(any) any) string {
			v, _ := get(tableVariantKey).(tableVariant)
			return tableVariantClasses[v]
		}),
		attr.FuncAttr("class", func(get func(any) any) string {
			s, _ := get(tableSizeKey).(tableSize)
			return tableSizeClasses[s]
		}),
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
