package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Button(attrs ...attr.Node) html.Element {
	a := attr.Group(attrs...)
	tag := "button"
	if a.Has("href") {
		tag = "a"
	}
	return func(nodes ...html.Node) html.Node {
		return html.Tag(tag)(
			attr.Class("a$button"),
			attr.FuncAttr("class", func(get func(any) any) string {
				v, _ := get(buttonVariantKey).(buttonVariant)
				return buttonVariantClasses[v]
			}),
			attr.FuncAttr("class", func(get func(any) any) string {
				s, _ := get(buttonSizeKey).(buttonSize)
				r, _ := get(buttonShapeKey).(buttonShape)
				if r == buttonCircle {
					return buttonCircleSizeClasses[s]
				}
				return buttonSizeClasses[s]
			}),
			attr.FuncAttr("class", func(get func(any) any) string {
				r, _ := get(buttonShapeKey).(buttonShape)
				return buttonShapeClasses[r]
			}),
			a,
		)(nodes...).With(TextSelectNone)
	}
}

// Variants
var (
	ButtonSolid       = html.WithValue(buttonVariantKey, buttonSolid)       // default
	ButtonSoft        = html.WithValue(buttonVariantKey, buttonSoft)
	ButtonSurface     = html.WithValue(buttonVariantKey, buttonSurface)
	ButtonOutline     = html.WithValue(buttonVariantKey, buttonOutline)
	ButtonGhost       = html.WithValue(buttonVariantKey, buttonGhost)
	ButtonDestructive = html.WithValue(buttonVariantKey, buttonDestructive)
)

// Sizes
var (
	ButtonSize1 = html.WithValue(buttonSizeKey, buttonSize1)
	ButtonSize2 = html.WithValue(buttonSizeKey, buttonSize2) // default
	ButtonSize3 = html.WithValue(buttonSizeKey, buttonSize3)
	ButtonSize4 = html.WithValue(buttonSizeKey, buttonSize4)
)

// Shapes / radius
var (
	ButtonRadiusFull   = html.WithValue(buttonShapeKey, buttonRadiusFull) // default
	ButtonCircle       = html.WithValue(buttonShapeKey, buttonCircle)
	ButtonRadiusMedium = html.WithValue(buttonShapeKey, buttonRadiusMedium)
	ButtonRadiusNone   = html.WithValue(buttonShapeKey, buttonRadiusNone)
)

type (
	buttonVariant int
	buttonSize    int
	buttonShape   int
)

const (
	buttonSolid buttonVariant = iota // default
	buttonSoft
	buttonSurface
	buttonOutline
	buttonGhost
	buttonDestructive
)

const (
	buttonSize2 buttonSize = iota // default
	buttonSize1
	buttonSize3
	buttonSize4
)

const (
	buttonRadiusFull buttonShape = iota // default
	buttonCircle
	buttonRadiusMedium
	buttonRadiusNone
)

var buttonVariantClasses = map[buttonVariant]string{
	buttonSolid:       "a$button+solid",
	buttonSoft:        "a$button+soft",
	buttonSurface:     "a$button+surface",
	buttonOutline:     "a$button+outline",
	buttonGhost:       "a$button+ghost",
	buttonDestructive: "a$button+destructive",
}

var buttonSizeClasses = map[buttonSize]string{
	buttonSize1: "a$button+size-1",
	buttonSize2: "a$button+size-2",
	buttonSize3: "a$button+size-3",
	buttonSize4: "a$button+size-4",
}

var buttonCircleSizeClasses = map[buttonSize]string{
	buttonSize1: "a$button+circle-1",
	buttonSize2: "a$button+circle-2",
	buttonSize3: "a$button+circle-3",
	buttonSize4: "a$button+circle-4",
}

var buttonShapeClasses = map[buttonShape]string{
	buttonRadiusFull:   "a$button+radius-full",
	buttonCircle:       "a$button+radius-circle",
	buttonRadiusMedium: "a$button+radius-medium",
	buttonRadiusNone:   "a$button+radius-none",
}
