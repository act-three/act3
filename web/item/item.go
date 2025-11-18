package item

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	Selected attr.Node = attr.Attr("data-selected")
)

func Item(attrs ...attr.Node) html.Element {
	a := attr.Group(attrs...)
	tag := "div"
	if a.Has("href") {
		tag = "a"
	}
	return html.Tag(tag)(
		attr.Class(`
			[a]:hover:bg-accent-9/50
			focus-visible:border-gray-6
			focus-visible:ring-gray-6/50
			flex
			flex-wrap
			items-center
			rounded-sm
			border
			border-transparent
			text-sm
			outline-none
			focus-visible:ring-[3px]
			bg-transparent
			gap-2
			p-2
			data-selected:bg-accent-9
			data-selected:text-accent-12
		`),
		a,
		attr.Attr("data-slot")("item"),
		attr.Attr("data-variant")("default"),
		attr.Attr("data-size")("default"),
	)
}

func Media(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class(`
			flex
			shrink-0
			items-center
			justify-center
			gap-2
			group-has-[[data-slot=item-description]]/item:translate-y-0.5
			group-has-[[data-slot=item-description]]/item:self-start
			[&_svg]:pointer-events-none
			w-10
			h-15
			overflow-hidden
			[&_img]:size-full
			[&_img]:object-cover
		`),
		attr.Group(attrs...),
		attr.Attr("data-slot")("item-media"),
		attr.Attr("data-variant")("poster"),
	)
}

func Content(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class(`
			flex
			flex-1
			flex-col
			gap-1
			[&+[data-slot=item-content]]:flex-none
		`),
		attr.Group(attrs...),
		attr.Attr("data-slot")("item-content"),
	)
}

func Title(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class(`
			flex
			w-fit
			items-center
			gap-2
			text-sm
			font-medium
			leading-snug
		`),
		attr.Group(attrs...),
		attr.Attr("data-slot")("item-title"),
	)
}

func Description(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class(`
			text-gray-11
			line-clamp-2
			text-balance
			text-sm
			font-normal
			leading-normal
			[&>a:hover]:text-gray-12
			[&>a]:underline
			[&>a]:underline-offset-4
		`),
		attr.Group(attrs...),
		attr.Attr("data-slot")("item-description"),
	)
}
