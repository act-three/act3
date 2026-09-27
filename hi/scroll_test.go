package hi_test

import (
	"fmt"
	"strings"
	"testing"

	"ily.dev/act3/hi"
	"ily.dev/act3/hi/internal/uitest"
)

func TestGeometryScrollCrossSizing(t *testing.T) {
	t.Parallel()
	vertical := func(v hi.View) hi.View { return hi.ScrollView(hi.Vertical, v).Class("probe") }
	horizontal := func(v hi.View) hi.View { return hi.ScrollView(hi.Horizontal, v).Class("probe") }
	fixed := hi.Blue.Frame(hi.Width(140), hi.Height(90))
	for _, tt := range []struct {
		name string
		view hi.View
		w, h float64
	}{
		{"vertical intrinsic", vertical(fixed), 140, 400},
		{"horizontal intrinsic", horizontal(fixed), 600, 90},
		{"vertical fill", vertical(hi.Blue.Frame(hi.Height(90))), 600, 400},
		{"horizontal fill", horizontal(hi.Blue.Frame(hi.Width(140))), 600, 400},
		{"vertical rigid", hi.HStack(vertical(fixed), hi.Red.Frame(hi.Width(550))).Gap(0), 140, 400},
		{"horizontal rigid", hi.VStack(horizontal(fixed), hi.Red.Frame(hi.Height(350))).Gap(0), 600, 90},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, tt.view, func(s *uitest.Session) {
				r := s.Rect(".probe", 0)
				within(t, "width", r.W, tt.w, 1)
				within(t, "height", r.H, tt.h, 1)
			})
		})
	}
}

func TestGeometryScrollInheritsUnboundedCrossAxis(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		axis  hi.AxisSet
		ideal hi.FrameBoundsOption
		w, h  float64
	}{
		{"vertical", hi.Vertical, hi.IdealWidth(73), 73, 100},
		{"horizontal", hi.Horizontal, hi.IdealHeight(73), 100, 73},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v := hi.ScrollView(tt.axis, hi.Blue.FrameBounds(tt.ideal)).FixedSize()
			stage(t, v, func(s *uitest.Session) {
				r := s.Rect("hi-scroll", 0)
				within(t, "width", r.W, tt.w, 1)
				within(t, "height", r.H, tt.h, 1)
			})
		})
	}
}

// Compare minimum sizes with an ordinary frame under flex pressure,
// without assuming a particular font's metrics. Explicit zero minimums
// must still permit shrinking below the text's intrinsic minimum.
func TestGeometryScrollPreservesCrossMinimum(t *testing.T) {
	t.Parallel()
	for _, axis := range []hi.AxisSet{hi.Vertical, hi.Horizontal} {
		for _, zeroMin := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/zero=%v", axis, zeroMin), func(t *testing.T) {
				content := hi.View(hi.Text(strings.Repeat("longestword short ", 10)))
				if zeroMin {
					content = content.FrameBounds(hi.MinWidth(0), hi.MinHeight(0))
				}
				var sizes []float64
				for _, v := range []hi.View{content.Frame(), hi.ScrollView(axis, content)} {
					v = v.Class("measure")
					if axis == hi.Vertical {
						v = hi.HStack(v, hi.Blue.Frame(hi.Width(20))).Gap(0).Frame(hi.Width(40))
					} else {
						v = hi.VStack(v, hi.Blue.Frame(hi.Height(20))).Gap(0).Frame(hi.Height(40))
					}
					stage(t, hi.ScrollView(axis, v), func(s *uitest.Session) {
						r := s.Rect(".measure", 0)
						if axis == hi.Vertical {
							sizes = append(sizes, r.W)
						} else {
							sizes = append(sizes, r.H)
						}
					})
				}
				within(t, "cross minimum matches frame", sizes[1], sizes[0], 1)
			})
		}
	}
}
