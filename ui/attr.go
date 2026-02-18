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

type attrEmbed[T any] struct {
	attr.Node
	v T
}

func option[T any](o T) attr.Node { return attrEmbed[T]{group(), o} }

func getOption[T any](attrs []attr.Node, def T) T {
	v := def
	for _, a := range attrs {
		a, ok := a.(attrEmbed[T])
		if !ok {
			continue
		}
		v = a.v
	}
	return v
}

type sizeOption int

var (
	Size1 = option(size1)
	Size2 = option(size2)
	Size3 = option(size3)
)

const (
	size1 sizeOption = iota
	size2
	size3
)

type sideOption int

var (
	SideAll    = option(sideAll)
	SideX      = option(sideX)
	SideY      = option(sideY)
	SideTop    = option(sideTop)
	SideBottom = option(sideBottom)
	SideLeft   = option(sideLeft)
	siedRigth  = option(sideRight)
)

const (
	sideAll sideOption = iota
	sideX
	sideY
	sideTop
	sideBottom
	sideLeft
	sideRight
)

//type asOption struct{ tag string }
//
//func As(tag string) attr.Node {
//	return attrEmbed{group(), asOption{tag}}
//}
//
//type alignmentOption struct{ a alignment }
//
//func TopLeading() attr.Node {
//	return attrEmbed{group(), alignmentOption{topLeading}}
//}
