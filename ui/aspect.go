package ui

import "fmt"

// Aspect is a width:height image aspect ratio.
type Aspect struct {
	W, H int
}

// String returns the ratio as "W / H", the CSS aspect-ratio value form.
func (a Aspect) String() string {
	return fmt.Sprintf("%d / %d", a.W, a.H)
}

var (
	AspectPoster    = Aspect{2, 3}
	AspectThumbnail = Aspect{16, 9}
	AspectBanner    = Aspect{1000, 185}
)
