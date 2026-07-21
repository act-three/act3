package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
)

var (
	Class = attr.Class
	Style = attr.Style
	Href  = attr.Href
	group = domi.Group
)

// Stylef returns a style attribute with a formatted value.
func Stylef(format string, a ...any) domi.Attr {
	return attr.Stylef(format, a...)
}

func Attr(name string) func(...string) domi.Attr {
	return func(value ...string) domi.Attr { return domi.Name(name, value...) }
}

// BoolAttr returns a name-only attribute when b is true and nothing otherwise.
func BoolAttr(name string, b bool) domi.Attr {
	if b {
		return domi.Name(name)
	}
	return nil
}

func Disabled(disabled bool) domi.Attr { return attr.Disabled(disabled) }

func Inert(inert bool) domi.Attr { return attr.Inert(inert) }

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
