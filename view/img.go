package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/model"
	"ily.dev/act3/model/kind"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
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

// AppImageDialog renders the image-edit dialog:
// an upload control over the item's current image,
// sized to the image kind's aspect ratio.
func AppImageDialog(k kind.ImageOwner, id string, im model.Image) domi.Node {
	w, h := im.Kind.Aspect()
	a := Aspect{W: w, H: h}
	return imageDialog(&msg.DialogClose{}, a)(
		buttonUpload()(
			Hidden("kind", k.String()),
			Hidden("id", id),
			PosterImg(a, PosterFill, imgAttrs(im)),
		),
	)
}
