package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
)

// Frame positions v inside an invisible frame
// with the given dimensions and alignment.
//
// Note that type [Alignment] satisfies FrameOption.
func (v base) Frame(o ...FrameOption) View {
	var w wrapFrame
	for _, o := range o {
		o.applyFrame(&w)
	}
	return v.wrap(w)
}

// wrapFrame is a sizing frame. It is a single-cell grid that places a
// view within it.
//
// A fixed size axis makes its size become the inside view's available
// space. An auto axis makes the frame's available space available to
// the view inside.
//
// For the frame's parent, a fixed size axis never issues a fill
// request. An auto axis propagates the inside view's fill request, if
// any.
type wrapFrame struct {
	h, v  size
	align Alignment
}

// definite is the set of axes the frame makes definite.
func (w wrapFrame) definite() (a AxisSet) {
	if w.h.definite {
		a |= Horizontal
	}
	if w.v.definite {
		a |= Vertical
	}
	return a
}

func (w wrapFrame) wrapElement(_ renderContext, content domi.Node, f AxisSet) box {
	var align domi.Attr
	if w.align != Center {
		align = attr.Class(w.align.placeClass())
	}
	b := box{
		fills:   f &^ w.definite(),
		rigid:   w.definite(),
		attrs:   domi.Group(attr.Class("ui-frame"), align),
		content: content,
	}
	if w.h.definite {
		b.setStyle("width", w.h.css())
	}
	if w.v.definite {
		b.setStyle("height", w.v.css())
	}
	return b
}

// context clears unbounded on each axis the frame makes definite.
// A definite axis is available space for the view inside.
func (w wrapFrame) context(rc renderContext) renderContext {
	rc.unbounded &^= w.definite()
	return rc
}

// A FrameOption configures the size and alignment of a frame.
//
// Type [Alignment] satisfies FrameOption.
// To specify a 100px-square bottom-center-aligned frame:
//
//	view.Frame(Width(100), Height(100), Bottom)
type FrameOption interface{ applyFrame(*wrapFrame) }

type frameOption func(*wrapFrame)

func (o frameOption) applyFrame(w *wrapFrame) { o(w) }

// Height sets the frame's height.
//
// If h is Auto, the frame adopts the height of the view inside.
// The default height is Auto.
func Height[L Length | Auto](h L) FrameOption {
	s := newSize(h)
	return frameOption(func(w *wrapFrame) { w.v = s })
}

// Width sets the frame's width.
//
// If w is Auto, the frame adopts the width of the view inside.
// The default width is Auto.
func Width[L Length | Auto](w L) FrameOption {
	s := newSize(w)
	return frameOption(func(f *wrapFrame) { f.h = s })
}
