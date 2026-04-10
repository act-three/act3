package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

// imgLargestAttrName marks an <img> as "largest variant only" so
// liveImgUpdate knows to set only src (not srcset) on it. Used
// by image editing dialogs and the tiny edit buttons that open
// them, which should always show the highest-fidelity rendition.
const imgLargestAttrName = "data-img-largest"

// imgAttrs renders the standard set of <img> attributes from an
// Image plus its live-update address: src on the smallest
// variant, a full srcset, and data-live + data-addrN attributes
// for live-update targeting. Layout dimensions come from CSS
// (the u-poster aspect-ratio rules), not HTML width/height.
func imgAttrs(im model.Image, addr []string) attr.Node {
	return attr.Group(
		attr.Attr("data-live"),
		LiveAddr(addr),
		attr.Src(im.SmallestURL()),
		attr.Srcset(im.Srcset()),
	)
}

// imgLargestAttrs renders <img> attributes pointing at the
// largest variant only, with no srcset. Used by image editing
// dialogs and edit buttons. The live-update addr is attached so
// they update in place when the underlying image changes; the
// data-img-largest marker routes the update to the "largest
// only" branch of liveImgUpdate.
func imgLargestAttrs(im model.Image, addr []string) attr.Node {
	return attr.Group(
		attr.Attr("data-live"),
		LiveAddr(addr),
		attr.Attr(imgLargestAttrName),
		attr.Src(im.LargestURL()),
	)
}

// liveImgUpdate emits two turbo SetTargets streams that together
// refresh every <img> previously rendered with imgAttrs or
// imgLargestAttrs for the given addr. Normal imgs get both src
// and srcset reset; "largest only" imgs (editing dialogs, edit
// buttons) get only src reset. Splitting the streams avoids
// applying a srcset to elements that shouldn't have one.
func liveImgUpdate(im model.Image, addr []string) html.Node {
	base := LiveSelector(addr)
	return html.Group(
		turbo.SetTargets(
			base+`:not([`+imgLargestAttrName+`])`,
			html.Div(
				attr.Src(im.SmallestURL()),
				attr.Srcset(im.Srcset()),
			)(),
		),
		turbo.SetTargets(
			base+`[`+imgLargestAttrName+`]`,
			html.Div(
				attr.Src(im.LargestURL()),
			)(),
		),
	)
}
