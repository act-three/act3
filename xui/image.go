package ui

import (
	"ily.dev/domi/attr"
)

// An ImageView displays an image.
type ImageView interface {
	View

	// Alt sets the alt text for the receiver.
	Alt(string) ImageView

	// FramedAs sets the framing mode for the receiver.
	//
	// The default framing mode is Native.
	FramedAs(FramingMode) ImageView
}

// Image displays the image at url.
func Image(url string) ImageView { return imageView{base{imageNode{src: url}}} }

type imageView struct{ base }

func (v imageView) Alt(s string) ImageView {
	n := v.base[0].(imageNode)
	n.alt = s
	v.base = base{n}
	return v
}

func (v imageView) FramedAs(f FramingMode) ImageView {
	n := v.base[0].(imageNode)
	n.fit = f
	v.base = base{n}
	return v
}

type imageNode struct {
	src, alt string
	fit      FramingMode
}

func (n imageNode) render(env environment) box {
	// An img is a replaced element and cannot host the stroke
	// carrier, so pending strokes box out around the image.
	if len(env.stroke) > 0 {
		return wrapMod(env, n)
	}

	env.tag = "img"
	env.add(attr.Src(n.src))
	if n.alt != "" { // alt="" would mark the image as decorative.
		env.add(attr.Alt(n.alt))
	}
	var p plan
	if n.fit == Native {
		// At native size the img's intrinsic geometry is the whole
		// contract, so the img is its own box, with no wrapper to
		// mediate between the box and the available space.
		p = plan{rigid: Horizontal | Vertical}
	} else {
		// A scaling mode is a statement about filling available
		// space: the img is fully flexible, and object-fit fits the
		// picture to the space it fills. On an axis with unbounded
		// available space, there is nothing to fill, so the img's
		// intrinsic geometry applies. When both axes are unbounded,
		// this is its natural size. Otherwise, the definite axis is
		// scaled by the img's ratio to apply to the unbounded axis.
		env.style.Set("min-width", "0")
		env.style.Set("min-height", "0")
		env.style.Set("object-fit", n.fit.objectFit())
		p = plan{fills: Horizontal | Vertical}
	}
	return build(env, p)
}
