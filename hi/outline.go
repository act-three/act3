package hi

// BorderOutline draws a line
// of the given width and color
// around the outside of a view's border.
// The gap specifies the distance
// between the border and the outline.
//
// The outline takes no layout space.
// If any of the view's content
// overlaps with the outline,
// the outline is drawn in front.
//
// A view can have at most one outline.
// If BorderOutline is applied
// to a view that already has an outline,
// it has no effect.
//
// The gap and width are clamped to the nonnegative range.
func BorderOutline(gap, width complex128, c Color) Modifier {
	checkLength(gap)
	checkLength(width)
	return modEnvState(func(env environment, state State) environment {
		o := outline{gap, width, c.color()}
		env.outline = append(env.outline, term[outline]{state, o})
		env.hasPaint = true
		return env
	})
}

type outline struct {
	gap   complex128
	width complex128
	color color // nil means no applicable outline
}

func outlineLength(v complex128) string {
	s := cssLength(v)
	if real(v) < 0 || imag(v) < 0 {
		return "max(0px," + s + ")"
	}
	return s
}
