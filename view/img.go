package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

// imgAttrs renders the standard set of <img> attributes from an
// Image plus its live-update address: src on the smallest
// variant, a full srcset, and data-live + data-addrN attributes
// for live-update targeting. Layout dimensions come from CSS
// (the u-poster aspect-ratio rules), not HTML width/height.
func imgAttrs(im model.Image, addr []string) attr.Node {
	return group(
		Attr("data-live"),
		LiveAddr(addr),
		attr.Src(im.SmallestURL()),
		attr.Srcset(im.Srcset()),
	)
}

// liveImgUpdate emits a turbo SetTargets stream that refreshes
// every <img> previously rendered with imgAttrs for the given
// addr.
func liveImgUpdate(im model.Image, addr []string) html.Node {
	return turbo.SetTargets(
		LiveSelector(addr),
		html.Div(
			attr.Src(im.SmallestURL()),
			attr.Srcset(im.Srcset()),
		)(),
	)
}
