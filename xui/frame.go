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
	h, v  float64
	axes  AxisSet
	align Alignment
}

func (w wrapFrame) modify(n node) node {
	return func(env environment) box { return w.render(env, n) }
}

func (w wrapFrame) render(env environment, n node) box {
	inner := env
	// A definite axis is available space for the view inside,
	// so it is no longer unbounded.
	inner.unbounded &^= w.axes
	p := wrapSubview(inner, n)
	p.fills &^= w.axes
	p.rigid |= w.axes
	env.tag = cmp.Or(env.tag, "ui-frame")
	w.align.setItemsOn(&env.style)
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	if w.axes.hasAll(Horizontal) {
		env.style.Set("width", cssPx(w.h))
	}
	if w.axes.hasAll(Vertical) {
		env.style.Set("height", cssPx(w.v))
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
// If omitted, the frame adopts the height of the view inside.
func Height(h float64) FrameOption {
	return frameOption(func(w *wrapFrame) {
		w.v = h
		w.axes |= Vertical
	})
}

// Width sets the frame's width.
//
// If omitted, the frame adopts the width of the view inside.
func Width(w float64) FrameOption {
	return frameOption(func(f *wrapFrame) {
		f.h = w
		f.axes |= Horizontal
	})
}
