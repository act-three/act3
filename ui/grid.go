package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func grid(n int, attrs []attr.Node) html.Element {
	return Box(
		Class("u-grid"),
		attr.Stylef("--cols:%d", n),
		group(attrs...),
	)
}

func Grid2(attrs ...attr.Node) html.Element  { return grid(2, attrs) }
func Grid3(attrs ...attr.Node) html.Element  { return grid(3, attrs) }
func Grid4(attrs ...attr.Node) html.Element  { return grid(4, attrs) }
func Grid5(attrs ...attr.Node) html.Element  { return grid(5, attrs) }
func Grid6(attrs ...attr.Node) html.Element  { return grid(6, attrs) }
func Grid7(attrs ...attr.Node) html.Element  { return grid(7, attrs) }
func Grid8(attrs ...attr.Node) html.Element  { return grid(8, attrs) }
func Grid9(attrs ...attr.Node) html.Element  { return grid(9, attrs) }
func Grid10(attrs ...attr.Node) html.Element { return grid(10, attrs) }
func Grid11(attrs ...attr.Node) html.Element { return grid(11, attrs) }
func Grid12(attrs ...attr.Node) html.Element { return grid(12, attrs) }

var (
	ColSpan2  = Class("u-col-span-2")
	ColSpan3  = Class("u-col-span-3")
	ColSpan4  = Class("u-col-span-4")
	ColSpan5  = Class("u-col-span-5")
	ColSpan6  = Class("u-col-span-6")
	ColSpan7  = Class("u-col-span-7")
	ColSpan8  = Class("u-col-span-8")
	ColSpan9  = Class("u-col-span-9")
	ColSpan10 = Class("u-col-span-10")
	ColSpan11 = Class("u-col-span-11")
	ColSpan12 = Class("u-col-span-12")
)
