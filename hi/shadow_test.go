package hi_test

import (
	"bytes"
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
	red := hi.BorderShadow(1, 2, 5, 3, hi.OKLCHA(.6, .2, 20, .6))
	blue := hi.BorderShadow(-2, 1, 7, 4, hi.OKLCHA(.5, .2, 260, .7))
	for _, shape := range []hi.Shape{hi.Rectangle, hi.RoundedRectangle, hi.Capsule, hi.Ellipse} {
		for _, filled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/filled=%v", shape, filled), func(t *testing.T) {
				base := hi.Transparent.Frame(hi.Width(80), hi.Height(40))
				fill, edge := hi.Transparent, hi.Transparent
				if filled {
					fill, edge = hi.OKLCHA(.8, .08, 70, .5), hi.OKLCHA(.4, .1, 140, .7)
				}
				fused := base.Modify(red).Background(fill).BorderStroke(3, edge).Modify(blue).BorderShape(shape)
				reference := base.
					Padding(hi.Edges(0)).Modify(red).BorderShape(shape).
					Padding(hi.Edges(0)).Background(fill).BorderShape(shape).
					Padding(hi.Edges(0)).BorderStroke(3, edge).BorderShape(shape).
					Padding(hi.Edges(0)).Modify(blue).BorderShape(shape)
				stage(t, hi.HStack(fused.Class("fused"), reference.Class("reference")).Gap(60).Padding(hi.Edges(20)).Background(hi.White), func(s *uitest.Session) {
					var extra int
					s.Eval(`document.querySelector('.fused').querySelectorAll('*').length`, &extra)
					if extra != 1 {
						t.Errorf("fused frame has %d descendants, want only its color subview", extra)
					}
					for _, scale := range []int{1, 2} {
						s.Run(chromedp.EmulateViewport(600, 400, chromedp.EmulateScale(float64(scale))))
						compareShadowPixels(t, s, scale, ".fused", ".reference")
					}
				})
			})
		}
	}
}

func compareShadowPixels(t *testing.T, s *uitest.Session, scale int, a, b string) {
	t.Helper()
	img := shadowScreenshot(t, s)
	ra, rb := s.Rect(a, 0), s.Rect(b, 0)
	if ra.W != rb.W || ra.H != rb.H {
		t.Fatalf("reference geometry differs: %+v, %+v", ra, rb)
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

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func shadowScreenshot(t *testing.T, s *uitest.Session) image.Image {
	t.Helper()
	// Let the compositor finish a density change before capturing pixels.
	s.Run(page.BringToFront(), chromedp.Evaluate(`new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))`, nil,
		func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }))
	var data []byte
	s.Run(chromedp.CaptureScreenshot(&data))
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestBorderShadowClipAndOpacity(t *testing.T) {
	shadow := hi.BorderShadow(0, 0, 8, 0, hi.Black)
	base := hi.Transparent.Frame(hi.Width(80), hi.Height(40))
	for _, tt := range []struct {
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, hi.HStack(tt.view.Class("subject")).Padding(hi.Edges(20)).Background(hi.White), func(s *uitest.Session) {
				r := s.Rect(".subject", 0)
				img := shadowScreenshot(t, s)
				inside, _, _, _ := img.At(int(r.X+r.W/2), int(r.Y+r.H/2)).RGBA()
				outside, _, _, _ := img.At(int(r.X-3), int(r.Y+r.H/2)).RGBA()
				if inside != 65535 || absDiff(outside, tt.outside) > 257 {
					t.Errorf("interior/exterior = %d/%d, want 65535/%d", inside, outside, tt.outside)
				}
				if tt.name == "rounded transparent" {
					corner, _, _, _ := img.At(int(r.X+3), int(r.Y+3)).RGBA()
					if corner != 0 {
						t.Errorf("rounded exterior within bounding rectangle = %d, want black", corner)
					}
				}
			})
		})
	}
}

func TestBorderShadowStableTransparent(t *testing.T) {
	for _, c := range []hi.Color{hi.Transparent, hi.Black} {
		html := render(t, hi.Text("x").Opacity(.5).BorderShadow(0, 0, 5, 0, c))
		if strings.Count(html, "<ui-box ") != 1 {
			t.Errorf("%v changed the opacity boundary: %s", c, html)
		}
	}
}
