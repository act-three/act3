package hi

import (
	"cmp"
	_ "embed"
	"fmt"
	"math"

	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/hi/internal/sheet"
)

// Progress indicates progress on a task.
//
//   - If no values are given,
//     the view shows indeterminate progress.
//
//   - If one value is given,
//     it is a fraction between 0 and 1,
//     and the progress view shows determinate progress.
//
//   - If a second value is given, it is the total.
func Progress(v ...float64) View {
	if len(v) == 0 {
		return view(nodeProgress)
	}
	p := v[0]
	if len(v) > 1 {
		p /= v[1]
		if v[1] <= 0 {
			p = 0
		}
	}
	if math.IsNaN(p) {
		p = 0
	}
	return view(nodeProgressBar(min(max(p, 0), 1)))
}

func nodeProgressBar(fraction float64) node {
	return func(env environment) box {
		height := "6px"
		switch env.controlSize {
		case Mini, Small:
			height = "4px"
		case Large:
			height = "8px"
		}
		env.tag = cmp.Or(env.tag, "progress")
		env.attrs = domi.Group(env.attrs,
			domi.Name("max", "1"),
			domi.Name("value", fmt.Sprint(fraction)),
		)
		env.style.Set("width", "auto")
		env.style.Set("height", height)
		var ss sheet.StyleSet
		ss.Set("appearance", "none")
		ss.Set("border-radius", "999px")
		track := Primary.color().colorCoords(env.theme)
		track.a = 0.1
		ss.Set("background", track.css())
		ss.Set("overflow", "hidden")
		ss.SetPseudo("::-webkit-progress-bar", "background", "transparent")
		ss.SetPseudo("::-webkit-progress-value", "border-radius", "999px")
		ss.SetPseudo("::-webkit-progress-value", "background", env.theme.accent.css())
		env.add(attr.Class(env.sheet.ClassFor(ss)))
		return build(env, plan{
			fills: Horizontal,
			rigid: Vertical,
			ideal: rect{width: 100},
		})
	}
}

func nodeProgress(env environment) box {
	i, size := 1, "20px"
	switch env.controlSize {
	case Mini, Small:
		i, size = 0, "14px"
	case Large:
		i, size = 2, "35px"
	}
	env.tag = cmp.Or(env.tag, "hi-progress")
	env.style.Set("display", "block")
	env.style.Set("width", size)
	env.style.Set("height", size)
	env.style.Set("--hi-progress-size", size)
	// Inline SVG supplies the baseline. Its vertical-align in hi.css
	// centers the artwork on the cap band without changing its box.
	env.lineHeight = new(complex128)
	return build(env, plan{
		rigid:   Horizontal | Vertical,
		content: progressImages[i],
	})
}

//go:embed progress/small.svg
var progressSmall string

//go:embed progress/regular.svg
var progressRegular string

//go:embed progress/large.svg
var progressLarge string

var progressImages = func() [3]domi.Node {
	var images [3]domi.Node
	for i, source := range [...]string{progressSmall, progressRegular, progressLarge} {
		var err error
		images[i], err = domi.UnsafeParseRaw(source)
		if err != nil {
			panic(err)
		}
	}
	return images
}()
