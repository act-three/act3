package hi

import (
	"cmp"

	"ily.dev/act3/hi/internal/canon"
)

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
// If omitted, the frame adopts the minimum height of the view inside.
func MinHeight(h complex128) FrameBoundsOption {
	checkLength(h)
	return frameBoundsOption(func(f *wrapFrameBounds) { f.v.setMin(h) })
}

// MinWidth sets the frame's minimum width.
//
// If w is greater than the frame's ideal width,
// MinWidth also sets the ideal to w.
//
// If omitted, the frame adopts the minimum width of the view inside.
func MinWidth(w complex128) FrameBoundsOption {
	checkLength(w)
	return frameBoundsOption(func(f *wrapFrameBounds) { f.h.setMin(w) })
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
// If omitted, the frame adopts the ideal height of the view inside.
func IdealHeight(h complex128) FrameBoundsOption {
	checkLength(h)
	return frameBoundsOption(func(f *wrapFrameBounds) { f.v.setIdeal(h) })
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
// If omitted, the frame adopts the ideal width of the view inside.
func IdealWidth(w complex128) FrameBoundsOption {
	checkLength(w)
	return frameBoundsOption(func(f *wrapFrameBounds) { f.h.setIdeal(w) })
}

// axisBounds is one axis of a bounds frame.
// it specifies the lower bound
// as well as the ideal size the axis takes
// when its available space is unbounded.
// invariant: min ≤ ideal (when both are concrete values).
type axisBounds struct {
	min, ideal       boundLength
	minSet, idealSet bool
}

func (x *axisBounds) setMin(v complex128) {
	s := boundLength{length: v}
	x.min, x.minSet = s, true
	if x.idealSet {
		x.ideal = x.ideal.max(s)
	}
}

func (x *axisBounds) setIdeal(v complex128) {
	s := boundLength{length: v}
	x.ideal, x.idealSet = s, true
	if x.minSet {
		x.min = x.min.min(s)
	}
}

// Bounds whose order depends on the root font size must be compared
// in CSS. Keep their min/max expressions alongside literal lengths.
type boundLength struct {
	length complex128
	expr   string
}

func (a boundLength) css() string {
	if a.expr != "" {
		return a.expr
	}
	return cssLength(a.length)
}

// atMost reports whether a <= b at every positive root font size.
func (a boundLength) atMost(b boundLength) bool {
	return a.expr == "" && b.expr == "" &&
		real(a.length) <= real(b.length) && imag(a.length) <= imag(b.length)
}

func (a boundLength) min(b boundLength) boundLength {
	if a.atMost(b) {
		return a
	}
	if b.atMost(a) {
		return b
	}
	return boundLength{expr: "min(" + a.css() + ", " + b.css() + ")"}
}

func (a boundLength) max(b boundLength) boundLength {
	if a.atMost(b) {
		return b
	}
	if b.atMost(a) {
		return a
	}
	return boundLength{expr: "max(" + a.css() + ", " + b.css() + ")"}
}

// wrapFrameBounds is a bounded frame:
// a single-cell CSS grid that adopts the size of the subview,
// clamped within its per-axis bounds.
// it places the subview in the resulting box.
// Bounds are not definite sizes:
// the frame propagates fill requests outward
// and unbounded available space inward,
// and its box tracks available space above the bounds wherever it lands.
// The exception is an axis with unbounded available space and a set
// ideal size: the ideal, being definite, settles the axis — fills stop,
// and the subview gets real available space.
type wrapFrameBounds struct {
	h, v  axisBounds
	align Alignment
}

func (w wrapFrameBounds) modify(n node) node {
	return func(env environment) box { return w.render(env, n) }
}

// idealAxes is the set of axes on which the frame uses its ideal size in env.
func (w wrapFrameBounds) idealAxes(env environment) (a AxisSet) {
	if env.unbounded.hasAll(Horizontal) && w.h.idealSet {
		a |= Horizontal
	}
	if env.unbounded.hasAll(Vertical) && w.v.idealSet {
		a |= Vertical
	}
	return a
}

// boundedAxes is the set of axes with any bound set.
// A bounded axis's sizing is governed by the frame,
// so the subview's rigidity does not pass through it.
func (w wrapFrameBounds) boundedAxes() (a AxisSet) {
	if w.h.idealSet || w.h.minSet {
		a |= Horizontal
	}
	if w.v.idealSet || w.v.minSet {
		a |= Vertical
	}
	return a
}

func (w wrapFrameBounds) render(env environment, n node) box {
	ideal := w.idealAxes(env)
	inner := env
	// An axis that takes its ideal has a definite size —
	// real available space for the subview, no longer unbounded.
	// Bounds never clear unboundedness: they clamp sizes, not queries.
	inner.unbounded &^= ideal
	p := wrapSubview(inner, n)
	// An axis with no bounds takes the subview's sizing and its
	// rigidity with it. A bounded axis tracks space above its
	// bounds instead, regardless of the subview's rigidity.
	p.rigid &^= w.boundedAxes()
	env.tag = cmp.Or(env.tag, "ui-frame")
	w.align.setItemsOn(&env.style)
	env.style.Set("display", "grid")
	w.setStyles(&env.style, ideal)
	return build(env, p)
}

// setStyles adds the frame's size and track declarations to ss.
func (w wrapFrameBounds) setStyles(ss *canon.StyleSet, ideal AxisSet) {
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
	if w.h.minSet {
		ss.Set("min-width", w.h.min.css())
		cols = "minmax(0, 100%)"
	}
	if w.v.minSet {
		ss.Set("min-height", w.v.min.css())
		rows = "minmax(0, 100%)"
	}
	ss.Set("grid-template-columns", cols)
	ss.Set("grid-template-rows", rows)
}
