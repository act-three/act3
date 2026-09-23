package hi_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"ily.dev/act3/hi"
	"ily.dev/act3/hi/internal/uitest"
)

func TestBorderShadowStates(t *testing.T) {
	t.Parallel()
	stage(t, hi.Transparent.Frame(hi.Width(80), hi.Height(40)).
		WhileHovered(hi.BorderShadow(0, 0, 3, 0, hi.Red)).
		WhilePressed(hi.BorderShadow(0, 0, 6, 0, hi.Blue)).
		BorderShape(hi.Capsule).Class("subject"), func(s *uitest.Session) {
		var nodes []*cdp.Node
		s.Run(css.Enable(), chromedp.Nodes(".subject", &nodes, chromedp.ByQuery))
		// Forcing :hover does not override the hover-capability media guard.
		// Check the browser's actual capability and explicitly test touch mode.
		for _, touch := range []bool{false, true} {
			var opts []chromedp.EmulateViewportOption
			if touch {
				opts = append(opts, chromedp.EmulateTouch)
			}
			s.Run(chromedp.EmulateViewport(600, 400, opts...))
			var canHover bool
			s.Eval(`matchMedia('(hover: hover)').matches`, &canHover)
			if touch && canHover {
				t.Fatal("touch emulation did not disable hover capability")
			}
			for _, tt := range []struct {
				states            []string
				want, wantNoHover string
			}{
				{nil, "none", "none"},
				{[]string{"hover"}, redCSS + " 0px 0px 0px 3px", "none"},
				{[]string{"active"}, blueCSS + " 0px 0px 0px 6px", blueCSS + " 0px 0px 0px 6px"},
				{[]string{"hover", "active"}, redCSS + " 0px 0px 0px 3px, " + blueCSS + " 0px 0px 0px 6px", blueCSS + " 0px 0px 0px 6px"},
				{nil, "none", "none"},
			} {
				s.Run(css.ForcePseudoState(nodes[0].NodeID, tt.states))
				var got string
				s.Eval(`getComputedStyle(document.querySelector('.subject')).boxShadow`, &got)
				want := tt.want
				if !canHover {
					want = tt.wantNoHover
				}
				if got != want {
					t.Errorf("canHover=%v, %v shadow = %s, want %s", canHover, tt.states, got, want)
				}
			}
		}
	})
}

// Zero-padding wrappers give each paint modifier a separate, equal-sized
// box. Compare that literal wrapper model to the fused paint at both
// densities, including paint outside rounded corners of a transparent view.
func TestBorderShadowWrapperEquivalence(t *testing.T) {
	t.Parallel()
	red := hi.BorderShadow(1, 2, 5, 3, hi.OKLCHA(.6, .2, 20, .6))
	blue := hi.BorderShadow(-2, 1, 7, 4, hi.OKLCHA(.5, .2, 260, .7))
	var views []hi.View
	var names []string
	for _, tt := range []struct {
		name  string
		shape hi.Shape
	}{
		{"rectangle", hi.Rectangle},
		{"rounded", hi.RoundedRectangle(8i)},
		{"capsule", hi.Capsule},
		{"ellipse", hi.Ellipse},
	} {
		for _, filled := range []bool{false, true} {
			base := hi.Transparent.Frame(hi.Width(80), hi.Height(40))
			fill, edge := hi.Transparent, hi.Transparent
			if filled {
				fill, edge = hi.OKLCHA(.8, .08, 70, .5), hi.OKLCHA(.4, .1, 140, .7)
			}
			fused := base.Modify(red).Background(fill).BorderStroke(3, edge).Modify(blue).BorderShape(tt.shape)
			reference := base.
				Padding(hi.Edges(0)).Modify(red).BorderShape(tt.shape).
				Padding(hi.Edges(0)).Background(fill).BorderShape(tt.shape).
				Padding(hi.Edges(0)).BorderStroke(3, edge).BorderShape(tt.shape).
				Padding(hi.Edges(0)).Modify(blue).BorderShape(tt.shape)
			views = append(views, hi.HStack(fused.Class("fused"), reference.Class("reference")).Gap(60))
			names = append(names, fmt.Sprintf("%s/filled=%v", tt.name, filled))
		}
	}
	paint(t, views, []int{1, 2}, func(s *uitest.Session, img image.Image, scale int) {
		for i, name := range names {
			t.Run(fmt.Sprintf("%s/%dx", name, scale), func(t *testing.T) {
				prefix := fmt.Sprintf(".paint-case-%d ", i)
				var extra int
				s.Eval(fmt.Sprintf(`document.querySelector(%q).querySelectorAll('*').length`, prefix+".fused"), &extra)
				if extra != 1 {
					t.Errorf("fused frame has %d descendants, want only its color subview", extra)
				}
				compareShadowPixels(t, img, scale, s.Rect(prefix+".fused", 0), s.Rect(prefix+".reference", 0))
			})
		}
	})
}

// Pack independent paint fixtures into rows so each density needs one
// screenshot. Each row leaves enough white space for paint outside its box.
func paint(t *testing.T, views []hi.View, scales []int, fn func(*uitest.Session, image.Image, int)) {
	t.Helper()
	rows := make([]hi.View, len(views))
	for i, v := range views {
		rows[i] = v.Padding(hi.Edges(20)).Background(hi.White).
			Frame(hi.Height(100)).Class(fmt.Sprintf("paint-case-%d", i))
	}
	stage(t, hi.VStack(rows...).Gap(0).Background(hi.White), func(s *uitest.Session) {
		for _, scale := range scales {
			s.Run(chromedp.EmulateViewport(300, int64(len(rows)*100), chromedp.EmulateScale(float64(scale))))
			fn(s, shadowScreenshot(t, s), scale)
		}
	})
}

func compareShadowPixels(t *testing.T, img image.Image, scale int, ra, rb uitest.Rect) {
	t.Helper()
	if ra.W != rb.W || ra.H != rb.H {
		t.Fatalf("reference geometry differs: %+v, %+v", ra, rb)
	}
	for _, r := range []uitest.Rect{ra, rb} {
		region := image.Rect(int(r.X)*scale-14*scale, int(r.Y)*scale-14*scale,
			int(r.X)*scale+int(r.W+14)*scale, int(r.Y)*scale+int(r.H+14)*scale)
		if !region.In(img.Bounds()) {
			t.Fatalf("paint region %v outside screenshot %v", region, img.Bounds())
		}
	}
	var different, total int
	for y := -14 * scale; y < int(ra.H+14)*scale; y++ {
		for x := -14 * scale; x < int(ra.W+14)*scale; x++ {
			r1, g1, b1, _ := img.At(int(ra.X)*scale+x, int(ra.Y)*scale+y).RGBA()
			r2, g2, b2, _ := img.At(int(rb.X)*scale+x, int(rb.Y)*scale+y).RGBA()
			// Separate rounded layers can round an antialiased edge
			// differently. Large differences indicate an ordering/mask bug.
			if max(absDiff(r1, r2), absDiff(g1, g2), absDiff(b1, b2)) > 5*257 {
				different++
			}
			total++
		}
	}
	// A narrow fringe at the silhouette may differ because separate
	// translucent fills/strokes antialias independently at rounded edges.
	if different > total/50 {
		t.Errorf("%dx: %d/%d pixels differ by more than 5 levels", scale, different, total)
	}
}

func paintRed(t *testing.T, img image.Image, x, y int) uint32 {
	t.Helper()
	if !(image.Point{X: x, Y: y}).In(img.Bounds()) {
		t.Fatalf("sample (%d, %d) outside screenshot %v", x, y, img.Bounds())
	}
	r, _, _, _ := img.At(x, y).RGBA()
	return r
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func shadowScreenshot(t *testing.T, s *uitest.Session) image.Image {
	t.Helper()
	// Let the compositor finish a density change before capturing pixels.
	s.Run(chromedp.Evaluate(`new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))`, nil,
		func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }))
	var data []byte
	s.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		data, err = page.CaptureScreenshot().WithOptimizeForSpeed(true).Do(ctx)
		return err
	}))
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestBorderShadowClipAndOpacity(t *testing.T) {
	t.Parallel()
	shadow := hi.BorderShadow(0, 0, 8, 0, hi.Black)
	base := hi.Transparent.Frame(hi.Width(80), hi.Height(40))
	cases := []struct {
		name    string
		view    hi.View
		outside uint32
	}{
		{"transparent", base.Modify(shadow), 0},
		{"rounded transparent", base.Modify(shadow).BorderShape(hi.Capsule), 0},
		{"clip outside", base.Modify(shadow).BorderClipped(), 65535},
		{"clip inside", base.BorderClipped().Modify(shadow), 0},
		{"opacity outside", base.Modify(shadow).Opacity(.5), 127 * 257},
		{"opacity inside", base.Opacity(.5).Modify(shadow), 0},
		{"ancestor clip", base.Modify(shadow).Padding(hi.Edges(4)).BorderClipped(), 65535},
	}
	var views []hi.View
	for _, tt := range cases {
		views = append(views, hi.HStack(tt.view.Class("subject")))
	}
	paint(t, views, []int{1}, func(s *uitest.Session, img image.Image, scale int) {
		for i, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				r := s.Rect(fmt.Sprintf(".paint-case-%d .subject", i), 0)
				inside := paintRed(t, img, int(r.X+r.W/2), int(r.Y+r.H/2))
				outside := paintRed(t, img, int(r.X-3), int(r.Y+r.H/2))
				if inside != 65535 || absDiff(outside, tt.outside) > 257 {
					t.Errorf("interior/exterior = %d/%d, want 65535/%d", inside, outside, tt.outside)
				}
				if tt.name == "rounded transparent" {
					corner := paintRed(t, img, int(r.X+3), int(r.Y+3))
					if corner != 0 {
						t.Errorf("rounded exterior within bounding rectangle = %d, want black", corner)
					}
				}
			})
		}
	})
}

func TestBorderShadowStableTransparent(t *testing.T) {
	t.Parallel()
	for _, c := range []hi.Color{hi.Transparent, hi.Black} {
		html := render(t, hi.Text("x").Opacity(.5).BorderShadow(0, 0, 5, 0, c))
		if strings.Count(html, "<hi-box ") != 1 {
			t.Errorf("%v changed the opacity boundary: %s", c, html)
		}
	}
}
