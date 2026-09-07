package ui

import (
	"cmp"
	_ "embed"
	"fmt"

	"ily.dev/domi"
)

// Icon displays the named icon.
// See [IconSource].
//
// The returned view is always square.
// When an icon is used with [FirstBaseline] alignment,
// it is centered between the text baseline and cap height.
//
//	HStack(
//		Icon("film"),
//		Text("A long paragraph that might wrap."),
//	).
//		Alignment(FirstBaseline)
func Icon(name string) View {
	return base{nodeIcon(name)}
}

// IconSource sets the source of icons displayed by [Icon].
//
// Given a name, f should return an SVG element
// designed to accommodate the CSS stroke-width property.
//
// If f returns nil, the Icon view displays a placeholder icon.
//
// The default icon source always returns nil.
func IconSource(f func(name string) domi.Node) Option {
	return optionIconSource{f: f}
}

// optionIconSource is the Option returned by IconSource.
// The embedded Option is never set; it only marks the type an Option.
type optionIconSource struct {
	domi.Option
	f func(string) domi.Node
}

func nodeIcon(name string) node {
	const scale, stroke = 1.6, 1.5
	return func(env environment) box {
		svg := env.iconSource(name)
		if svg == nil {
			svg = placeholderIcon
		}
		env.tag = cmp.Or(env.tag, "ui-icon")
		// The svg is inline content, so the box has a line box, and
		// the line's baseline is the box's baseline. With no leading
		// the line is exactly as tall as the svg, whose vertical-align
		// (in ui.css) then puts the baseline where the caller expects.
		env.style.Set("display", "block")
		env.style.Set("--ui-icon-scale", fmt.Sprintf("%.4gcap", scale))
		env.style.Set("--ui-icon-stroke-width", fmt.Sprintf("%.4g", stroke))
		env.style.Set("width", "var(--ui-icon-scale)")
		env.style.Set("height", "var(--ui-icon-scale)")
		env.lineHeight = append(env.lineHeight, term[string]{value: "0"})
		return build(env, plan{
			rigid:   Horizontal | Vertical,
			content: svg,
		})
	}
}

// placeholderSVG is the square-dashed icon from Lucide (ISC license),
// unmodified.
//
//go:embed placeholder.svg
var placeholderSVG string

var placeholderIcon = func() domi.Node {
	n, err := domi.UnsafeParseRaw(placeholderSVG)
	if err != nil {
		panic(err)
	}
	return n
}()
