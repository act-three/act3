package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/model"
)

// imgAttrs renders the standard set of <img> attributes from an
// Image: src on the smallest variant and a full srcset. Layout
// dimensions come from CSS (the u-poster aspect-ratio rules), not
// HTML width/height.
func imgAttrs(im model.Image) domi.Attr {
	return group(
		attr.Src(im.SmallestURL()),
		attr.SrcSet(im.Srcset()),
	)
}
