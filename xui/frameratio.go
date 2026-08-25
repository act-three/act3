package ui

import (
	"cmp"
	"fmt"
)

// A FrameRatioOption configures a ratio frame.
//
// Type [Alignment] satisfies FrameRatioOption.
// To display a view in a 3:2 frame as wide as the view,
// aligned to the bottom of the frame:
//
//	view.FrameRatio(3, 2, Horizontal, Bottom)
type FrameRatioOption interface{ applyFrameRatio(*wrapFrameRatio) }

// wrapFrameRatio is an aspect-ratio frame: a single-cell grid whose
// width:height is fixed.
//
// On the anchor axis, it adopts its subview's sizing behavior,
// like an Auto frame axis. (We also call this layout-preserving.)
// It adopts the subview's size and passes its fill request
// and rigidity through.
// The other axis is derived from the anchor by the ratio,
// so it is determined here, like a definite frame axis.
// No fill request leaves it, it is rigid,
// and the subview gets it as definite available space.
type wrapFrameRatio struct {
	w, h   int
	anchor AxisSet
	align  Alignment
	node   node
}

func (w wrapFrameRatio) modify(n node) node { w.node = n; return w }

func (w wrapFrameRatio) render(env environment) box {
	derived := w.anchor.complement()
	inner := env
	inner.unbounded &^= derived
	node := w.node
	kind := containerGrid
	placeItems := w.align.placeItems()
	// With both frame dimensions automatic, intrinsic sizing resolves the
	// inline size from content and aspect-ratio derives the block size. It
	// does not derive inline size from an auto, content-sized block size.
	// So we use a vertical writing mode to make physical height the inline
	// axis. The subview is rotated back, and the ratio remains physical.
	if w.anchor == Vertical {
		// The block axis of the rotated frame runs left to right.
		// In a right-to-left document, it runs right to left,
		// so that Leading stays the leading edge.
		env.style.Set("writing-mode", "vertical-lr")
		env.style.SetPseudo(":dir(rtl)", "writing-mode", "vertical-rl")
		node = modStyle("writing-mode", "horizontal-tb").modify(node)
		kind = containerGridRotated
		placeItems = w.align.placeItemsRotated()
	}
	p := wrapSubviewIn(inner, kind, node)
	p.fills &^= derived
	p.rigid |= derived
	env.tag = cmp.Or(env.tag, "ui-aspect")
	env.style.Set("display", "grid")
	env.style.Set("grid-template-columns", "100%")
	env.style.Set("grid-template-rows", "100%")
	env.style.Set("place-items", placeItems)
	env.style.Set("aspect-ratio", fmt.Sprintf("%d / %d", w.w, w.h))
	// The derived axis has a content-based automatic minimum in
	// CSS, which would break the ratio when the subview is larger
	// than the derived size. Zeroing it keeps the ratio and lets
	// the subview overflow instead.
	if w.anchor == Vertical {
		env.style.Set("min-width", "0")
	} else {
		env.style.Set("min-height", "0")
	}
	return build(env, p)
}
