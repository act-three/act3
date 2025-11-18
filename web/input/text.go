package input

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Text(attrs ...attr.Node) html.Element {
	return html.Input(
		attr.Class(`
			border-input
			bg-gray-1
			selection:bg-accent-9
			selection:text-white
			ring-offset-gray-2
			placeholder:text-gray-11
			shadow-xs
			flex
			h-9
			w-full
			min-w-0
			rounded-md
			border
			px-3
			py-1
			text-gray-12
			outline-none
			transition-[color,box-shadow]
			disabled:cursor-not-allowed
			disabled:opacity-50
			md:text-sm
			focus-visible:border-accent-8
			focus-visible:ring-accent-8/50
			focus-visible:ring-[3px]
			aria-invalid:ring-crimson-8/20
			aria-invalid:border-crimson-8
		`),
		attr.Group(attrs...),
		attr.Attr("data-slot")("input"),
	)
}

func Submit(attrs ...attr.Node) html.Element {
	return html.Input(
		attr.Group(attrs...),
		attr.Type("submit"),
	)
}
