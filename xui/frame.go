package ui

import "cmp"

// wrapFrame is a sizing frame. It is a single-cell grid that places a
// view within it.
//
// A fixed size axis makes its size become the inside view's available
// space. An auto axis makes the frame's available space available to
// the view inside.
//
// For the frame's parent, a fixed size axis never issues a fill
// request. An auto axis propagates the inside view's fill request, if
// any, and likewise its rigidity.
type wrapFrame struct {
	h, v  size
	align Alignment
}

func (w wrapFrame) modify(n node) node {
	return func(env environment) box { return w.render(env, n) }
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

func (w wrapFrame) render(env environment, n node) box {
	inner := env
	// A definite axis is available space for the view inside,
	// so it is no longer unbounded.
	inner.unbounded &^= w.definite()
	p := wrapSubview(inner, n)
	p.fills &^= w.definite()
	p.rigid |= w.definite()
	env.tag = cmp.Or(env.tag, "ui-frame")
	w.align.setItemsOn(&env.style)
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	if w.h.definite {
		env.style.Set("width", w.h.css())
	}
	if w.v.definite {
		env.style.Set("height", w.v.css())
	}
	return build(env, p)
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
func Height[Size int | float64 | Auto](h Size) FrameOption {
	s := newSize(h)
	return frameOption(func(w *wrapFrame) { w.v = s })
}

// Width sets the frame's width.
//
// If w is Auto, the frame adopts the width of the view inside.
// The default width is Auto.
func Width[Size int | float64 | Auto](w Size) FrameOption {
	s := newSize(w)
	return frameOption(func(f *wrapFrame) { f.h = s })
}
