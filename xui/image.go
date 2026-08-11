package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// An ImageView displays an image.
type ImageView struct{ base }

// Image displays the image at url.
func Image(url string) ImageView { return ImageView{base{imageNode{src: url}}} }

// Alt sets the alt text for v.
func (v ImageView) Alt(s string) ImageView {
	n := v.base[0].(imageNode)
	n.alt = s
	v.base = base{n}
	return v
}

// FramedAs sets the framing mode for v.
//
// The default framing mode is Native.
func (v ImageView) FramedAs(f FramingMode) ImageView {
	n := v.base[0].(imageNode)
	n.fit = f
	v.base = base{n}
	return v
}

type imageNode struct {
	src, alt string
	fit      FramingMode
}

func (n imageNode) render(rc renderContext) box {
	var alt domi.Attr
	if n.alt != "" { // alt="" would mark the image as decorative.
		alt = attr.Alt(n.alt)
	}
	if n.fit == Native {
		// At native size the img's intrinsic geometry is the whole
		// contract, so the img is its own box, with no wrapper to
		// mediate between the box and the available space.
		return box{
			tag:   "img",
			rigid: Horizontal | Vertical,
			attrs: domi.Group(attr.Src(n.src), alt, rc.shapeClass()),
		}
	}
	// A scaling mode is a statement about meeting an imposed box: the
	// img is fully flexible, and object-fit fits the picture to
	// whatever box it lands in. On an unbounded axis there is no box
	// to meet, so the fill drops away and the img's intrinsic
	// geometry answers — its natural size, or the other axis scaled
	// through the picture's ratio.
	return box{
		tag:   "img",
		fills: (Horizontal | Vertical) &^ rc.unbounded,
		attrs: domi.Group(attr.Src(n.src), alt, attr.Class("ui-image", n.fit.class()), rc.shapeClass()),
	}
}
