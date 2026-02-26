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
			attr.Class(buttonBase),
			attr.FuncAttr("class", func(get func(any) any) string {
				variant, _ := get(buttonVariantKey).(buttonVariant)
				return buttonVariantTable[variant]
			}),
			attr.FuncAttr("class", func(get func(any) any) string {
				shape, _ := get(buttonShapeKey).(buttonShape)
				size, _ := get(buttonSizeKey).(buttonSize)
				if shape == buttonCircle {
					return buttonCircleSizeTable[size]
				}
				return buttonSizeTable[size]
			}),
			attr.FuncAttr("class", func(get func(any) any) string {
				shape, _ := get(buttonShapeKey).(buttonShape)
				return buttonShapeTable[shape]
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

const buttonBase = `
	relative
	inline-flex
	shrink-0
	items-center
	justify-center
	font-medium
	whitespace-nowrap
	transition-colors
	outline-none
	focus-visible:ring-[3px]
	focus-visible:ring-accent-8/40
	disabled:pointer-events-none
	disabled:opacity-50
	aria-disabled:pointer-events-none
	aria-disabled:opacity-50
	[&_svg]:pointer-events-none
	[&_svg]:shrink-0
	[&_svg:not([class*='size-'])]:size-4
	cursor-pointer
`

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

var buttonVariantTable = map[buttonVariant]string{
	buttonSolid:       "bg-accent-9 text-white hover:bg-accent-10",
	buttonSoft:        "bg-accent-3 text-accent-11 hover:bg-accent-4",
	buttonSurface:     "bg-accent-2 text-accent-11 border border-accent-7 hover:bg-accent-3",
	buttonOutline:     "border border-accent-8 text-accent-11 hover:bg-accent-2",
	buttonGhost:       "text-accent-11 hover:bg-accent-3",
	buttonDestructive: "bg-crimson-9 text-white hover:bg-crimson-10",
}

var buttonSizeTable = map[buttonSize]string{
	buttonSize1: "h-6 px-2 gap-1 text-xs",
	buttonSize2: "h-8 px-3 gap-1.5 text-sm",
	buttonSize3: "h-10 px-4 gap-2",
	buttonSize4: "h-12 px-6 gap-3",
}

var buttonCircleSizeTable = map[buttonSize]string{
	buttonSize1: "size-6 text-xs",
	buttonSize2: "size-8 text-sm",
	buttonSize3: "size-10",
	buttonSize4: "size-12",
}

var buttonShapeTable = map[buttonShape]string{
	buttonRadiusFull:   "rounded-full",
	buttonCircle:       "aspect-square rounded-full",
	buttonRadiusMedium: "rounded-md",
	buttonRadiusNone:   "rounded-none",
}
