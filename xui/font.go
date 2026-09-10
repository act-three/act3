package ui

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"ily.dev/act3/xui/internal/sheet"
)

// A FontOption configures typography.
// See [View.Font] and [TextView.TextFont].
type FontOption interface{ apply(environment) environment }

type fontOption func(environment) environment

func (o fontOption) apply(env environment) environment { return o(env) }

// These options select weights on the OpenType weight scale.
var (
	Thin       FontOption = Weight(100)
	ExtraLight FontOption = Weight(200)
	Light      FontOption = Weight(300)
	Normal     FontOption = Weight(400)
	Medium     FontOption = Weight(500)
	SemiBold   FontOption = Weight(600)
	Bold       FontOption = Weight(700)
	ExtraBold  FontOption = Weight(800)
	Heavy      FontOption = Weight(900)
)

// Weight sets the text weight on a scale of 1 to 1000.
//
// Values outside this range are clamped.
func Weight(w int) FontOption {
	w = min(max(w, 1), 1000)
	return fontOption(func(env environment) environment {
		env.fontWeight = strconv.Itoa(w)
		return env
	})
}

// These options configure italic text.
var (
	Italic FontOption = italic(true)
	Roman  FontOption = italic(false)
)

func italic(b bool) fontOption {
	return func(env environment) environment {
		env.fontStyle = map[bool]string{true: "italic", false: "normal"}[b]
		return env
	}
}

// SizeCap sets the height of capital letters.
// It also sets the line height to a multiple of cap.
//
// SizeCap is equivalent to SizeCapAbs(cap, lineHeight*cap).
func SizeCap(cap complex128, lineHeight float64) FontOption {
	return SizeCapAbs(cap, complex(lineHeight, 0)*cap)
}

// SizeCapAbs sets the height of capital letters.
// It also sets the line height.
func SizeCapAbs(cap, lineHeight complex128) FontOption {
	return fontSizeOption(cap, lineHeight, sizeCap)
}

// SizeEm sets the height of the text's em square.
// It also sets the line height to a multiple of em.
//
// SizeEm is equivalent to SizeEmAbs(em, lineHeight*em).
func SizeEm(em complex128, lineHeight float64) FontOption {
	return SizeEmAbs(em, complex(lineHeight, 0)*em)
}

// SizeEmAbs sets the height of the text's em square.
// It also sets the line height.
func SizeEmAbs(em, lineHeight complex128) FontOption {
	return fontSizeOption(em, lineHeight, sizeEm)
}

func fontSizeOption(size, height complex128, basis fontSizeBasis) FontOption {
	checkLength(size)
	checkLength(height)
	return fontOption(func(env environment) environment {
		env.fontSize = fontSize{basis, size}
		env.lineHeight = &height
		return env
	})
}

// Family selects the type family as a CSS font-family list.
func Family(list string) FontOption {
	return fontOption(func(env environment) environment {
		env.fontFamily = list
		return env
	})
}

// OpenTypeFeature sets an OpenType feature tag to the given value.
//
// A value of 0 disables the feature.
// A value of 1 enables it.
// Larger values select alternatives for features that support them.
//
// The tag must be four printable ASCII characters.
//
// If an OpenType feature has a corresponding named variant
// (such as [SmallCaps] or [TabularNums]),
// and a view is configured with both,
// the OpenType feature overrides the named variant.
func OpenTypeFeature(tag string, value int) FontOption {
	if len(tag) != 4 {
		panic("ui: OpenType feature tag must contain four printable ASCII characters")
	}
	for i := range len(tag) {
		if tag[i] < 0x20 || tag[i] > 0x7e {
			panic("ui: OpenType feature tag must contain four printable ASCII characters")
		}
	}
	if value < 0 {
		panic("ui: OpenType feature value must be nonnegative")
	}
	return fontOption(func(env environment) environment {
		features := make(openTypeFeatures, len(env.fontFeatures)+1)
		maps.Copy(features, env.fontFeatures)
		features[tag] = value
		env.fontFeatures = features
		return env
	})
}

// Feature maps are immutable once shared by environments.
type openTypeFeatures map[string]int

func (f openTypeFeatures) css() string {
	var settings []string
	for _, tag := range slices.Sorted(maps.Keys(f)) {
		settings = append(settings, strconv.Quote(tag)+" "+strconv.Itoa(f[tag]))
	}
	return strings.Join(settings, ", ")
}

func font(opts []FontOption) func(environment) environment {
	opts = slices.Clone(opts)
	return func(env environment) environment {
		for _, o := range opts {
			env = o.apply(env)
		}
		return env
	}
}

type fontSizeBasis int

const (
	sizeCap fontSizeBasis = iota + 1
	sizeEm
)

type fontSize struct {
	basis  fontSizeBasis
	length complex128
}

func addFontStylesTo(s *sheet.StyleSet, env environment) {
	adjust, ok := map[fontSizeBasis]string{
		sizeCap: "cap-height 1",
		sizeEm:  "none",
	}[env.fontSize.basis]
	if ok {
		s.Set("font-size", cssLength(env.fontSize.length))
		s.Set("font-size-adjust", adjust)
	}
	if env.lineHeight != nil {
		s.Set("line-height", cssLength(*env.lineHeight))
	}
	for _, d := range []decl{
		{"font-family", env.fontFamily},
		{"font-style", env.fontStyle},
		{"font-weight", env.fontWeight},
		{"font-variant-caps", env.fontVariants.caps.css()},
		{"font-variant-numeric", variantList(env.fontVariants.numeric[:])},
		{"font-variant-ligatures", variantList(env.fontVariants.ligatures[:])},
		{"font-variant-position", env.fontVariants.position.css()},
		{"font-feature-settings", env.fontFeatures.css()},
	} {
		if d.value != "" {
			s.Set(d.property, d.value)
		}
	}
}
