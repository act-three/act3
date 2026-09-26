package hi

import (
	"cmp"
	_ "embed"

	"ily.dev/domi"
)

// Progress indicates progress on a task of indeterminate length.
func Progress() View { return view(nodeProgress) }

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
