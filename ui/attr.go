package ui

import "ily.dev/act3/html/attr"

var (
	Class = attr.Class
	Href  = attr.Href
	group = attr.Group
)

func Attr(name string) attr.AttrName { return attr.Attr(name) }

func Disabled(disabled bool) attr.Node {
	if disabled {
		return attr.Disabled
	} else {
		return group()
	}
}

var (
	Gap0 = Class("gap-0")
	Gap1 = Class("gap-1")
	Gap2 = Class("gap-2")
	Gap3 = Class("gap-3")
	Gap4 = Class("gap-4")
	Gap5 = Class("gap-5")
	Gap6 = Class("gap-6")
	Gap7 = Class("gap-7")
	Gap8 = Class("gap-8")
)

