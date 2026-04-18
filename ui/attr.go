package ui

import "ily.dev/act3/html/attr"

var (
	Class = attr.Class
	Style = attr.Style
	Href  = attr.Href
	group = attr.Group
)

// Stylef returns a style attribute with a formatted value.
func Stylef(format string, a ...any) attr.Node {
	return attr.Stylef(format, a...)
}

func Attr(name string) attr.AttrName { return attr.Attr(name) }

func Disabled(disabled bool) attr.Node {
	if disabled {
		return attr.Disabled
	} else {
		return group()
	}
}

func Inert(inert bool) attr.Node {
	if inert {
		return attr.Inert
	} else {
		return group()
	}
}

var (
	Gap0 = Class("u-gap-0")
	Gap1 = Class("u-gap-1")
	Gap2 = Class("u-gap-2")
	Gap3 = Class("u-gap-3")
	Gap4 = Class("u-gap-4")
	Gap5 = Class("u-gap-5")
	Gap6 = Class("u-gap-6")
	Gap7 = Class("u-gap-7")
	Gap8 = Class("u-gap-8")
)
