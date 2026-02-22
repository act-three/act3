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
			attr.Class(base),
			attr.FuncAttr("class", func(get func(any) any) string {
				variant, _ := get(buttonVariantKey).(buttonVariant)
				return buttonVariantTable[variant]
			}),
			attr.FuncAttr("class", func(get func(any) any) string {
				variant, _ := get(buttonVariantKey).(buttonVariant)
				size, _ := get(buttonSizeKey).(buttonSize)
				return buttonSizeTable[variant][size]
			}),
			attr.FuncAttr("class", func(get func(any) any) string {
				shape, _ := get(buttonShapeKey).(buttonShape)
				return buttonShapeTable[shape]
			}),
			a,
		)(nodes...).With(TextSelectNone)
	}
}

var (
	ButtonBorderless  = html.WithValue(buttonVariantKey, buttonBorderless) // default
	ButtonBordered    = html.WithValue(buttonVariantKey, buttonBordered)
	ButtonProminent   = html.WithValue(buttonVariantKey, buttonProminent)
	ButtonDestructive = html.WithValue(buttonVariantKey, buttonDestructive)
)

var (
	ButtonSM = html.WithValue(buttonSizeKey, buttonSM)
	ButtonMD = html.WithValue(buttonSizeKey, buttonMD) // default
	ButtonLG = html.WithValue(buttonSizeKey, buttonLG)
)

var (
	ButtonCapsule     = html.WithValue(buttonShapeKey, buttonCapsule) // default
	ButtonCircle      = html.WithValue(buttonShapeKey, buttonCircle)
	ButtonRoundedRect = html.WithValue(buttonShapeKey, buttonRoundedRect)
)

const base = `
	focus-visible:border-gray-6
	focus-visible:ring-gray-7/50
	aria-invalid:ring-crimson-8/20
	aria-invalid:border-crimson-8
	inline-flex
	shrink-0
	items-center
	justify-center
	gap-2
	font-medium
	whitespace-nowrap
	transition-all
	outline-none
	focus-visible:ring-[3px]
	disabled:pointer-events-none
	disabled:opacity-50
	aria-disabled:pointer-events-none
	aria-disabled:opacity-50
	[&_svg]:pointer-events-none
	[&_svg]:shrink-0
	[&_svg:not([class*='size-'])]:size-4
	active:opacity-60
	cursor-pointer
`

type (
	buttonVariant int
	buttonSize    int
	buttonShape   int
)

const (
	buttonBorderless buttonVariant = iota // default
	buttonBordered
	buttonProminent
	buttonDestructive
)

const (
	buttonMD buttonSize = iota // default
	buttonSM
	buttonLG
)

const (
	buttonCapsule buttonShape = iota // default
	buttonCircle
	buttonRoundedRect
)

var buttonVariantTable = map[buttonVariant]string{
	buttonBorderless: `
		text-accent-11
	`,
	buttonBordered: `
		bg-gray-3
		text-accent-11
	`,
	buttonProminent: `
		bg-accent-9
		text-white-12
	`,
	buttonDestructive: `
		bg-crimson-9
		text-white-12
	`,
}

var (
	buttonSizeTable = map[buttonVariant]map[buttonSize]string{
		buttonBorderless:  buttonSizeBorderless,
		buttonBordered:    buttonSizeBordered,
		buttonProminent:   buttonSizeBordered,
		buttonDestructive: buttonSizeBordered,
	}

	buttonSizeBordered = map[buttonSize]string{
		buttonSM: "h-8 gap-1.5 px-3 has-[>svg]:px-2.5",
		buttonMD: "h-9 px-4 py-2 has-[>svg]:px-3",
		buttonLG: "h-10 px-6 has-[>svg]:px-4",
	}

	buttonSizeBorderless = map[buttonSize]string{
		buttonSM: "gap-1.5 has-[>svg]:px-2.5",
		buttonMD: "has-[>svg]:px-3",
		buttonLG: "has-[>svg]:px-4",
	}
)

var buttonShapeTable = map[buttonShape]string{
	buttonCapsule: `
		aspect-auto
		rounded-full
	`,
	buttonCircle: `
		aspect-square
		rounded-full
	`,
	buttonRoundedRect: `
		aspect-auto
		rounded-md
	`,
}
