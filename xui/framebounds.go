package ui

import (
	"ily.dev/domi/attr"

	"ily.dev/act3/xui/internal/sheet"
)

// FrameBounds positions v inside an invisible frame
// with the given bounds and alignment.
//
// Note that type [Alignment] satisfies FrameBoundsOption.
func (v base) FrameBounds(o ...FrameBoundsOption) View {
	var w wrapFrameBounds
	for _, o := range o {
		o.applyFrameBounds(&w)
	}
	return v.Modify(w)
}

// A FrameBoundsOption configures the bounds and alignment of a frame.
// The available space for the view inside the frame
// is clamped to the configured bounds.
//
// Later options override earlier ones.
// For instance:
//
//	view.FrameBounds(MinWidth(100), IdealWidth(50)) //  50 px min AND ideal width
//	view.FrameBounds(IdealWidth(50), MinWidth(100)) // 100 px min AND ideal width
//
// Type [Alignment] satisfies FrameBoundsOption.
// To specify a bottom-center-aligned frame at least 100px wide:
//
//	view.FrameBounds(MinWidth(100), Bottom)
type FrameBoundsOption interface{ applyFrameBounds(*wrapFrameBounds) }

type frameBoundsOption func(*wrapFrameBounds)

func (o frameBoundsOption) applyFrameBounds(w *wrapFrameBounds) { o(w) }

// MinHeight sets the frame's minimum height.
//
// If h is greater than the frame's ideal height,
// MinHeight also sets the ideal to h.
//
// Auto adopts the minimum height of the view inside.
// The default minimum height is Auto.
func MinHeight[L Length | Auto](h L) FrameBoundsOption {
	s := newSize(h)
	return frameBoundsOption(func(w *wrapFrameBounds) { w.v.setMin(s) })
}

// MinWidth sets the frame's minimum width.
//
// If w is greater than the frame's ideal width,
// MinWidth also sets the ideal to w.
//
// Auto adopts the minimum width of the view inside.
// The default minimum width is Auto.
func MinWidth[L Length | Auto](w L) FrameBoundsOption {
	s := newSize(w)
	return frameBoundsOption(func(w *wrapFrameBounds) { w.h.setMin(s) })
}

// IdealHeight sets the frame's ideal height.
//
// The ideal height is used only when the available height is unbounded,
// such as directly inside a vertical ScrollView.
// In that case, the frame's height will be set to its ideal height.
//
// If h is less than the frame's minimum height,
// IdealHeight also sets the minimum to h.
//
// If h is Auto, the frame adopts the ideal height of the view inside.
// The default ideal height is Auto.
func IdealHeight[L Length | Auto](h L) FrameBoundsOption {
	s := newSize(h)
	return frameBoundsOption(func(w *wrapFrameBounds) { w.v.setIdeal(s) })
}

// IdealWidth sets the frame's ideal width.
//
// The ideal width is used only when the available width is unbounded,
// such as directly inside a horizontal ScrollView.
// In that case, the frame's width will be set to its ideal width.
//
// If w is less than the frame's minimum width,
// IdealWidth also sets the minimum to w.
//
// If w is Auto, the frame adopts the ideal width of the view inside.
// The default ideal width is Auto.
func IdealWidth[L Length | Auto](w L) FrameBoundsOption {
	s := newSize(w)
	return frameBoundsOption(func(w *wrapFrameBounds) { w.h.setIdeal(s) })
}

// axisBounds is one axis of a bounds frame.
// it specifies the lower bound
// as well as the ideal size the axis takes
// when its available space is unbounded.
// invariant: min ≤ ideal (when both are concrete values).
type axisBounds struct{ min, ideal size }

func (x *axisBounds) setMin(s size) {
	x.min = s
	if s.definite && x.ideal.definite && x.ideal.px < s.px {
		x.ideal = s
	}
}

func (x *axisBounds) setIdeal(s size) {
	x.ideal = s
	if s.definite && x.min.definite && x.min.px > s.px {
		x.min = s
	}
}

// wrapFrameBounds is a bounded frame:
// a single-cell CSS grid that adopts the size of the subview,
// clamped within its per-axis bounds.
// it places the subview in the resulting box.
// Bounds are not definite sizes:
// the frame stays transparent to fill requests
// and to unbounded available space,
// and its box tracks available space above the bounds wherever it lands.
// The exception is an axis with unbounded available space and a set
// ideal size: the ideal, being definite, settles the axis — fills stop,
// and the subview gets real available space.
type wrapFrameBounds struct {
	h, v  axisBounds
	align Alignment
	node  node
}

func (w wrapFrameBounds) modify(n node) node { w.node = n; return w }

// idealAxes is the set of axes on which the frame uses its ideal size in env.
func (w wrapFrameBounds) idealAxes(env environment) (a AxisSet) {
	if env.unbounded.hasAll(Horizontal) && w.h.ideal.definite {
		a |= Horizontal
	}
	if env.unbounded.hasAll(Vertical) && w.v.ideal.definite {
		a |= Vertical
	}
	return a
}

// boundedAxes is the set of axes with any bound set.
// A bounded axis's sizing is governed by the frame,
// so the subview's rigidity does not pass through it.
func (w wrapFrameBounds) boundedAxes() (a AxisSet) {
	if w.h.ideal.definite || w.h.min.definite {
		a |= Horizontal
	}
	if w.v.ideal.definite || w.v.min.definite {
		a |= Vertical
	}
	return a
}

func (w wrapFrameBounds) render(env environment) box {
	ideal := w.idealAxes(env)
	inner := env
	// An axis that takes its ideal has a definite size —
	// real available space for the subview, no longer unbounded.
	// Bounds never clear unboundedness: they clamp sizes, not queries.
	inner.unbounded &^= ideal
	p := wrapSubview(inner, w.node)
	// An axis with no bounds takes the subview's sizing and its
	// rigidity with it. A bounded axis tracks space above its
	// bounds instead, regardless of the subview's rigidity.
	p.rigid &^= w.boundedAxes()
	env.add(attr.Class("ui-frame"))
	env.style.Set("place-items", w.align.placeItems())
	w.setStyles(&env.style, ideal)
	return build(env, p)
}

// setStyles adds the frame's size and track declarations to ss.
func (w wrapFrameBounds) setStyles(ss *sheet.StyleSet, ideal AxisSet) {
	if ideal.hasAll(Horizontal) {
		ss.Set("width", w.h.ideal.css())
	}
	if ideal.hasAll(Vertical) {
		ss.Set("height", w.v.ideal.css())
	}
	// A floored axis's track gives up its intrinsic contribution.
	// Without intervention, the frame's min-content size is its subview's,
	// and CSS min-* can only raise a floor, not lower it. Zeroing
	// the track's intrinsic contribution makes min-* the floor.
	cols, rows := "100%", "100%"
	if w.h.min.definite {
		ss.Set("min-width", w.h.min.css())
		cols = "minmax(0, 100%)"
	}
	if w.v.min.definite {
		ss.Set("min-height", w.v.min.css())
		rows = "minmax(0, 100%)"
	}
	ss.Set("grid-template-columns", cols)
	ss.Set("grid-template-rows", rows)
}
