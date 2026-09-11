package hi

import (
	"archive/zip"
	"cmp"
	_ "embed"
	"fmt"
	"io/fs"
	"strings"

	"ily.dev/domi"
	"ily.dev/domi/attr"
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
	return view(nodeIcon(name))
}

// IconSource sets the source of icons displayed by [Icon].
//
// Given a name, f should return an SVG element
// designed to accommodate the CSS stroke-width property.
//
// If f returns nil, the Icon view displays a placeholder icon.
//
// The default source provides icons from the [Lucide] icon set.
//
// [Lucide]: https://lucide.dev/
func IconSource(f func(name string) domi.Node) Option {
	return optionIconSource{f: f}
}

// optionIconSource is the Option returned by IconSource.
// The embedded Option is never set; it only marks the type an Option.
type optionIconSource struct {
	domi.Option
	f func(string) domi.Node
}

//go:embed lucide/icons.zip
var lucideZIP string

var lucideIcons = func() *zip.Reader {
	r, err := zip.NewReader(strings.NewReader(lucideZIP), int64(len(lucideZIP)))
	if err != nil {
		panic(err)
	}
	return r
}()

func defaultIconSource(name string) domi.Node {
	b, err := fs.ReadFile(lucideIcons, name+".svg")
	if err != nil {
		return nil
	}
	n, err := domi.UnsafeParseRaw(string(b))
	if err != nil {
		return nil
	}
	return n
}

func nodeIcon(name string) node {
	const scale, stroke = 1.6, 1.5
	return func(env environment) box {
		svg := env.iconSource(name)
		if svg == nil {
			svg = placeholderIcon
		}
		env.tag = cmp.Or(env.tag, "hi-icon")
		// The svg is inline content, so the box has a line box, and
		// the line's baseline is the box's baseline. The svg's margins
		// and vertical-align (in hi.css) keep it centered on the cap
		// band with or without trimming.
		env.style.Set("display", "block")
		env.style.Set("--hi-icon-scale", fmt.Sprintf("%.4gcap", scale))
		env.style.Set("--hi-icon-stroke-width", fmt.Sprintf("%.4g", stroke))
		size := "var(--hi-icon-scale)"
		if env.textTrim == TextCap|TextLastBaseline {
			size = "1cap"
			env.attrs = domi.Group(env.attrs, attr.Class("hi-icon-trim"))
		}
		env.style.Set("width", size)
		env.style.Set("height", size)
		env.lineHeight = new(complex128)
		return build(env, plan{
			rigid:   Horizontal | Vertical,
			content: svg,
		})
	}
}

var placeholderIcon = func() domi.Node {
	n := defaultIconSource("square-dashed")
	if n == nil {
		panic("missing or invalid bundled square-dashed icon")
	}
	return n
}()
