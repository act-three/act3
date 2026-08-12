package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"
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
	return v.wrap(w)
}

// A FrameBoundsOption configures the bounds and alignment of a frame.
// The available space for the view inside the frame
// is clamped to the configured bounds.
//
// Later options override earlier ones.
// For instance:
//
//	view.FrameBounds(MinHeight(100), MaxHeight(50)) //  50 px min AND max height
//	view.FrameBounds(MaxHeight(50), MinHeight(100)) // 100 px min AND max height
//
// Type [Alignment] satisfies FrameBoundsOption.
// To specify a bottom-center-aligned frame at most 100px wide:
//
//	view.FrameBounds(MaxWidth(100), Bottom)
type FrameBoundsOption interface{ applyFrameBounds(*wrapFrameBounds) }

type frameBoundsOption func(*wrapFrameBounds)

func (o frameBoundsOption) applyFrameBounds(w *wrapFrameBounds) { o(w) }

// MaxHeight sets the frame's maximum height.
//
// If h is less than the frame's minimum height,
// MaxHeight also sets the minimum to h.
//
// If h is less than the frame's ideal height,
// MaxHeight also sets the ideal to h.
//
// If h is Auto, the height is unbounded.
// The default is unbounded height.
func MaxHeight[L Length | Auto](h L) FrameBoundsOption {
	s := newSize(h)
	return frameBoundsOption(func(w *wrapFrameBounds) { w.v.setMax(s) })
}

// MaxWidth sets the frame's maximum width.
//
// If w is less than the frame's minimum width,
// MaxWidth also sets the minimum to w.
//
// If w is less than the frame's ideal width,
// MaxWidth also sets the ideal to w.
//
// If w is Auto, the width is unbounded.
// The default is unbounded width.
func MaxWidth[L Length | Auto](w L) FrameBoundsOption {
	s := newSize(w)
	return frameBoundsOption(func(w *wrapFrameBounds) { w.h.setMax(s) })
}

// MinHeight sets the frame's minimum height.
//
// If h is greater than the frame's maximum height,
// MinHeight also sets the maximum to h.
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
// If w is greater than the frame's maximum width,
// MinWidth also sets the maximum to w.
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
// If h is greater than the frame's maximum height,
// IdealHeight also sets the maximum to h.
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
// If w is greater than the frame's maximum width,
// IdealWidth also sets the maximum to w.
//
// If w is Auto, the frame adopts the ideal width of the view inside.
// The default ideal width is Auto.
func IdealWidth[L Length | Auto](w L) FrameBoundsOption {
	s := newSize(w)
	return frameBoundsOption(func(w *wrapFrameBounds) { w.h.setIdeal(s) })
}

// axisBounds is one axis of a bounds frame.
// it specifies the lower and upper bounds
// as well as the ideal size the axis takes
// when its available space is unbounded.
// invariant: min ≤ ideal ≤ max (for any concrete values among those fields).
type axisBounds struct{ min, ideal, max size }

func (x *axisBounds) setMin(s size) {
	x.min = s
	if s.definite && x.ideal.definite && x.ideal.px < s.px {
		x.ideal = s
	}
	if s.definite && x.max.definite && x.max.px < s.px {
		x.max = s
	}
}

func (x *axisBounds) setMax(s size) {
	x.max = s
	if s.definite && x.ideal.definite && x.ideal.px > s.px {
		x.ideal = s
	}
	if s.definite && x.min.definite && x.min.px > s.px {
		x.min = s
	}
}

func (x *axisBounds) setIdeal(s size) {
	x.ideal = s
	if s.definite && x.min.definite && x.min.px > s.px {
		x.min = s
	}
	if s.definite && x.max.definite && x.max.px < s.px {
		x.max = s
	}
}

// wrapFrameBounds is a bounded frame:
// a single-cell CSS grid that adopts the size of the subview,
// clamped within its per-axis bounds.
// it places the subview in the resulting box.
// Bounds are not definite sizes:
// the frame stays transparent to fill requests
// and to unbounded available space,
// and its box tracks available space between the bounds wherever it lands.
// One exception is an axis with unbounded available space and a set
// ideal size: the ideal, being definite, settles the axis — fills stop,
// and the subview gets real available space.
// The other is a fill request on an axis with a definite maximum,
// which the frame absorbs rather than relays (see cappedFills).
type wrapFrameBounds struct {
	h, v  axisBounds
	align Alignment
}

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
	if w.h.ideal.definite || w.h.min.definite || w.h.max.definite {
		a |= Horizontal
	}
	if w.v.ideal.definite || w.v.min.definite || w.v.max.definite {
		a |= Vertical
	}
	return a
}

// cappedFills is the set of axes on which a fill request from the subview
// meets a definite maximum.
// The frame is greedy only up to the maximum,
// so it absorbs the fill on those axes:
// it claims the maximum as its own size,
// yielding under pressure like any available-space tracker,
// and relays nothing upward —
// ancestors would grow to provide space the frame cannot use.
func (w wrapFrameBounds) cappedFills(f AxisSet) (a AxisSet) {
	if f.hasAll(Horizontal) && w.h.max.definite {
		a |= Horizontal
	}
	if f.hasAll(Vertical) && w.v.max.definite {
		a |= Vertical
	}
	return a
}

func (w wrapFrameBounds) wrapElement(env environment, content domi.Node, f, r AxisSet) box {
	ideal := w.idealAxes(env)
	capped := w.cappedFills(f) &^ ideal
	var align domi.Attr
	if w.align != Center {
		align = attr.Class(w.align.placeClass())
	}
	b := box{
		fills: f &^ (ideal | capped),
		// An axis taking its ideal is rigid on its own. An axis with
		// no bounds takes the subview's sizing and its rigidity with
		// it. A bounded axis tracks space between its bounds instead,
		// regardless of the subview's rigidity.
		rigid:   ideal | (r &^ w.boundedAxes()),
		attrs:   domi.Group(attr.Class("ui-frame"), align),
		content: content,
	}
	w.setStyles(&b, ideal, capped)
	return b
}

// context clears unbounded on each axis that takes its ideal:
// the ideal is a definite size — real available space for the subview.
// Bounds never clear it: they clamp sizes, not queries.
// A maximum clamps the space the subview lays out against
// and the answer the frame reports (max-*),
// but it cannot turn the absence of available space into space —
// adding a maximum must never make the subview bigger.
func (w wrapFrameBounds) context(env environment) environment {
	env.unbounded &^= w.idealAxes(env)
	return env
}

// setStyles adds the frame's size and track declarations to b.
func (w wrapFrameBounds) setStyles(b *box, ideal, capped AxisSet) {
	// An absorbed fill claims the maximum through the frame's own
	// track: minmax(0, max) makes the frame's max-content size the
	// maximum — hugging ancestors size themselves around the full
	// claim — and its min-content size zero, so the claim yields
	// under pressure, flooring at any explicit min-*. A size property
	// could not encode this: it would set both intrinsic sizes at
	// once, either poisoning the container's floor or (flex-basis)
	// vanishing from its max-content entirely.
	if ideal.hasAll(Horizontal) {
		b.setStyle("width", w.h.ideal.css())
	} else if capped.hasAll(Horizontal) {
		b.setStyle("grid-template-columns", "minmax(0,"+w.h.max.css()+")")
	}
	if ideal.hasAll(Vertical) {
		b.setStyle("height", w.v.ideal.css())
	} else if capped.hasAll(Vertical) {
		b.setStyle("grid-template-rows", "minmax(0,"+w.v.max.css()+")")
	}
	// An explicit minimum replaces the axis's content-derived floor:
	// without intervention the frame's min-content size is its subview's,
	// and CSS min-* can only raise a floor, never lower one. Zeroing
	// the track's intrinsic contribution makes min-* the floor. A
	// capped axis's track is already zeroed.
	if w.h.min.definite {
		b.setStyle("min-width", w.h.min.css())
		if !capped.hasAll(Horizontal) {
			b.add(attr.Class("ui-min-track-x"))
		}
	}
	if w.h.max.definite {
		b.setStyle("max-width", w.h.max.css())
	}
	if w.v.min.definite {
		b.setStyle("min-height", w.v.min.css())
		if !capped.hasAll(Vertical) {
			b.add(attr.Class("ui-min-track-y"))
		}
	}
	if w.v.max.definite {
		b.setStyle("max-height", w.v.max.css())
	}
}
