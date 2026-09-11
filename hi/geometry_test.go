package hi_test

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"ily.dev/domi"

	"ily.dev/act3/hi"
	"ily.dev/act3/hi/internal/fixture"
	"ily.dev/act3/hi/internal/uitest"
)

func TestMain(m *testing.M) { os.Exit(uitest.Main(m)) }

// stage renders v as the page root of a 600x400 viewport — the definite
// frame every fill chain terminates at — and hands the loaded page to fn.
func stage(t *testing.T, v hi.View, fn func(*uitest.Session)) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("<!doctype html><meta charset=utf-8><style>")
	sb.WriteString(staticCSS)
	sb.WriteString(`</style><body>`)
	_, page := hi.Render(v)
	if err := domi.RenderTo(&sb, page); err != nil {
		t.Fatalf("render: %v", err)
	}
	uitest.Run(t, 600, 400, sb.String(), fn)
}

// within asserts got is within tol of want.
func within(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	if got < want-tol || got > want+tol {
		t.Errorf("%s = %g, want %g ± %g", what, got, want, tol)
	}
}

func TestGeometrySpacerAbsorbsSlack(t *testing.T) {
	stage(t, hi.HStack(hi.Text("a"), hi.Spacer(), hi.Text("b")), func(s *uitest.Session) {
		within(t, "row width", s.Rect("hi-hstack", 0).W, 600, 1)
		if w := s.Rect("hi-spacer", 0).W; w < 400 {
			t.Errorf("spacer width = %g, want most of the row's slack", w)
		}
		within(t, "trailing text right edge", s.Rect("hi-text", 1).Right(), 600, 1)
	})
	stage(t, hi.VStack(hi.Text("a"), hi.Spacer(), hi.Text("b")), func(s *uitest.Session) {
		within(t, "column height", s.Rect("hi-vstack", 0).H, 400, 1)
		if h := s.Rect("hi-spacer", 0).H; h < 300 {
			t.Errorf("spacer height = %g, want most of the column's slack", h)
		}
		within(t, "trailing text bottom edge", s.Rect("hi-text", 1).Bottom(), 400, 1)
	})
}

// TestGeometryRootIsViewport pins the root contract: the rendered root
// covers the viewport exactly, and the viewport acts as a definite
// frame — the root view's fills on both axes terminate at it.
func TestGeometryRootIsViewport(t *testing.T) {
	stage(t, hi.Secondary, func(s *uitest.Session) {
		root := s.Rect("hi-root", 0)
		within(t, "root x", root.X, 0, 0.5)
		within(t, "root y", root.Y, 0, 0.5)
		within(t, "root width", root.W, 600, 1)
		within(t, "root height", root.H, 400, 1)
		fill := s.Rect("hi-color", 0)
		within(t, "color fills the viewport", fill.W, 600, 1)
		within(t, "color fills the viewport", fill.H, 400, 1)
	})
}

// TestGeometryRootScrollUsesDocument pins the page ScrollView lowering:
// hi-root hugs its content on each scrolling axis, stretches across each
// non-scrolling axis, and delegates overflow to the document viewport.
func TestGeometryRootScrollUsesDocument(t *testing.T) {
	tests := []struct {
		name      string
		axis      hi.AxisSet
		content   hi.View
		wantRootW float64
		wantRootH float64
		wantX     string
		wantY     string
	}{
		{
			name:      "vertical",
			axis:      hi.Vertical,
			content:   hi.OKLCH(0.5, 0, 0).Frame(hi.Width(40), hi.Height(1000)),
			wantRootW: 600,
			wantRootH: 1000,
			wantX:     "hidden",
			wantY:     "auto",
		},
		{
			name:      "horizontal",
			axis:      hi.Horizontal,
			content:   hi.OKLCH(0.5, 0, 0).Frame(hi.Width(1000), hi.Height(40)),
			wantRootW: 1000,
			wantRootH: 400,
			wantX:     "auto",
			wantY:     "hidden",
		},
		{
			name:      "both",
			axis:      hi.Horizontal | hi.Vertical,
			content:   hi.OKLCH(0.5, 0, 0).Frame(hi.Width(1000), hi.Height(1000)),
			wantRootW: 1000,
			wantRootH: 1000,
			wantX:     "auto",
			wantY:     "auto",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, hi.ScrollView(tt.axis, tt.content), func(s *uitest.Session) {
				root := s.Rect("hi-root", 0)
				within(t, "root width", root.W, tt.wantRootW, 1)
				within(t, "root height", root.H, tt.wantRootH, 1)

				var overflow [2]string
				s.Eval(`(() => { const s = getComputedStyle(document.documentElement); return [s.overflowX, s.overflowY] })()`, &overflow)
				if overflow != [2]string{tt.wantX, tt.wantY} {
					t.Errorf("html overflow = %q, want [%q %q]", overflow, tt.wantX, tt.wantY)
				}

				var hasScroll bool
				s.Eval(`document.querySelector("hi-scroll") !== null`, &hasScroll)
				if hasScroll {
					t.Error("root ScrollView retained an element viewport")
				}

				s.Eval(`window.scrollTo(300, 300)`, nil)
				var offset [2]float64
				s.Eval(`[window.scrollX, window.scrollY]`, &offset)
				if tt.axis&hi.Horizontal != 0 && offset[0] < 299 {
					t.Errorf("window.scrollX = %g, want document horizontal scrolling", offset[0])
				}
				if tt.axis&hi.Horizontal == 0 && offset[0] != 0 {
					t.Errorf("window.scrollX = %g on a non-scrolling axis, want 0", offset[0])
				}
				if tt.axis&hi.Vertical != 0 && offset[1] < 299 {
					t.Errorf("window.scrollY = %g, want document vertical scrolling", offset[1])
				}
				if tt.axis&hi.Vertical == 0 && offset[1] != 0 {
					t.Errorf("window.scrollY = %g on a non-scrolling axis, want 0", offset[1])
				}
			})
		})
	}
}

func TestGeometryRootScrollShortContentHugsItsScrollAxis(t *testing.T) {
	stage(t, hi.ScrollView(hi.Vertical, hi.Text("short")), func(s *uitest.Session) {
		root := s.Rect("hi-root", 0)
		within(t, "root width", root.W, 600, 1)
		if root.H >= 100 {
			t.Errorf("root height = %g, want short content height", root.H)
		}
		var heights [2]float64
		s.Eval(`[document.scrollingElement.clientHeight, document.scrollingElement.scrollHeight]`, &heights)
		if heights[1] != heights[0] {
			t.Errorf("short page heights = %v, want no document overflow", heights)
		}
	})
}

func TestGeometryRootScrollStickyUsesDocumentViewport(t *testing.T) {
	v := hi.ScrollView(hi.Vertical, hi.VStack(
		hi.Text("heading").Sticky().Class("heading"),
		hi.OKLCH(0.5, 0, 0).Frame(hi.Height(1000)),
	).Gap(0).Alignment(hi.Leading))
	stage(t, v, func(s *uitest.Session) {
		s.Eval(`window.scrollTo(0, 300)`, nil)
		within(t, "sticky heading top", s.Rect(".heading", 0).Y, 0, 1)
	})
}

// TestGeometrySpacerMinimumLength pins the spacer's floor: under
// pressure a spacer holds a small minimum along its expansion axis
// instead of letting its neighbors fuse, and with unbounded available
// space the same minimum is its answer.
func TestGeometrySpacerMinimumLength(t *testing.T) {
	rigid := func(w float64) hi.View { return hi.OKLCH(0.5, 0, 0).Frame(hi.Width(complex(w, 0)), hi.Height(40)) }
	stage(t, hi.HStack(rigid(300), hi.Spacer(), rigid(300)).Gap(0), func(s *uitest.Session) {
		within(t, "squeezed spacer floors at the minimum", s.Rect("hi-spacer", 0).W, 8, 1)
	})
	stage(t, hi.HStack(hi.Text("a"), hi.Spacer(), hi.Text("b")).Gap(0).FixedSize(), func(s *uitest.Session) {
		within(t, "fixed spacer takes the minimum", s.Rect("hi-spacer", 0).W, 8, 1)
	})
}

func TestGeometryDividerSpansMinorAxis(t *testing.T) {
	v := hi.HStack(hi.Text("a"), hi.Divider(), hi.Text("tall").Padding(hi.Edges(32)))
	stage(t, v, func(s *uitest.Session) {
		row, div := s.Rect("hi-hstack", 0), s.Rect("hi-divider", 0)
		within(t, "divider height", div.H, row.H, 1)
	})
}

// TestGeometryDividerSpansMinorAxisUnbounded pins minor-axis fills in
// unbounded available space: the row's height is resolved from its
// tallest sibling, and each divider expands to that extent — through
// a padding wrapper too — instead of taking its 10px ideal.
func TestGeometryDividerSpansMinorAxisUnbounded(t *testing.T) {
	row := hi.HStack(
		hi.Text("a"),
		hi.Divider(),
		hi.Divider().Padding(hi.Edges(1)),
		hi.Text("tall").Padding(hi.Edges(32)),
	)
	check := func(name string) func(*uitest.Session) {
		return func(s *uitest.Session) {
			h := s.Rect("hi-hstack", 0).H
			if h > 200 {
				t.Errorf("%s: row height = %g, want content height, not the viewport's", name, h)
			}
			within(t, name+": bare divider height", s.Rect("hi-divider", 0).H, h, 1)
			within(t, name+": padded divider height", s.Rect("hi-divider", 1).H, h-2, 1)
		}
	}
	stage(t, hi.ScrollView(hi.Vertical, row), check("scroll"))
	stage(t, row.FixedSize(), check("fixedsize"))
}

// TestGeometryFixedSizeColor pins the fill boundary: a FixedSize color
// keeps its 10px ideal even in a container with slack to offer.
func TestGeometryFixedSizeColor(t *testing.T) {
	stage(t, hi.OKLCH(0.5, 0, 0).FixedSize(), func(s *uitest.Session) {
		c := s.Rect("hi-color", 0)
		within(t, "color width", c.W, 10, 0.5)
		within(t, "color height", c.H, 10, 0.5)
	})
}

func TestGeometryFrameSubviewKeepsIntrinsicSize(t *testing.T) {
	stage(t, hi.Text("hi").Frame(hi.Width(120), hi.Height(120)), func(s *uitest.Session) {
		frame, text := s.Rect("hi-frame", 0), s.Rect("hi-text", 0)
		within(t, "frame width", frame.W, 120, 1)
		within(t, "frame height", frame.H, 120, 1)
		if text.W > 100 {
			t.Errorf("text width = %g, want intrinsic size, not stretched to the frame", text.W)
		}
		within(t, "text centered", text.X+text.W/2, frame.X+frame.W/2, 1)
	})
}

// TestGeometryDefiniteFrameDoesNotGrow pins fill termination for the
// axis-relative wants: a Spacer's major-axis fill is cancelled by a definite
// frame on the axis it resolves to, so the frame holds its width instead of
// growing into the row's slack.
func TestGeometryDefiniteFrameDoesNotGrow(t *testing.T) {
	v := hi.HStack(hi.Spacer().Frame(hi.Width(100)), hi.Text("b"))
	stage(t, v, func(s *uitest.Session) {
		within(t, "framed spacer width", s.Rect("hi-frame", 0).W, 100, 1)
		if w := s.Rect("hi-hstack", 0).W; w > 300 {
			t.Errorf("row width = %g, want shrink-wrapped: the definite frame settled the only fill", w)
		}
	})
}

// TestGeometryDefiniteFillThroughContainers pins fill termination across
// boxless containers: a want killed by its requester's definite frame stays
// dead through Group and a tagged frame, while a sibling's live want merged with it
// stays live.
func TestGeometryDefiniteFillThroughContainers(t *testing.T) {
	framed := func() hi.View { return hi.Spacer().Frame(hi.Width(100)) }
	cases := []struct {
		name     string
		v        hi.View
		wantGrow bool
	}{
		{"group", hi.HStack(hi.Group(framed()), hi.Text("b")), false},
		{"tagged frame", hi.HStack(framed().Frame().Tag("nav"), hi.Text("b")), false},
		{"group with live sibling", hi.HStack(hi.Group(framed(), hi.Spacer()), hi.Text("b")), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stage(t, c.v, func(s *uitest.Session) {
				w := s.Rect("hi-hstack", 0).W
				if c.wantGrow {
					within(t, "row width", w, 600, 1)
				} else if w > 300 {
					t.Errorf("row width = %g, want shrink-wrapped: the definite frame settled the only fill", w)
				}
			})
		})
	}
}

// TestGeometryFrameCentersOversizedSubview pins default placement when the
// subview outgrows the frame: the frame's track is pinned to its own box,
// so item alignment governs overflow and a centered subview overflows
// symmetrically instead of hanging off the trailing side.
func TestGeometryFrameCentersOversizedSubview(t *testing.T) {
	v := hi.Text(strings.Repeat("overflow ", 8)).
		FixedSize().
		Frame(hi.Width(100), hi.Height(100))
	stage(t, v, func(s *uitest.Session) {
		frame, text := s.Rect("hi-frame", 0), s.Rect("hi-text", 0)
		if text.W <= frame.W {
			t.Fatalf("text width = %g, want wider than the %g frame", text.W, frame.W)
		}
		within(t, "horizontal center offset", (text.X+text.W/2)-(frame.X+frame.W/2), 0, 1)
		within(t, "vertical center offset", (text.Y+text.H/2)-(frame.Y+frame.H/2), 0, 1)
	})
}

// TestGeometryOverlayHitTest pins overlay hit-testing: overlay content
// receives clicks, while clicks in the empty parts of the layer pass
// through to the base.
func TestGeometryOverlayHitTest(t *testing.T) {
	v := hi.Text("base").
		Frame(hi.Width(300), hi.Height(100)).
		Overlay(hi.TopTrailing, hi.Badge("hit").Class("probe"))
	stage(t, v, func(s *uitest.Session) {
		var badgeHit bool
		s.Eval(`(() => {
			const badge = document.querySelector(".probe");
			const r = badge.getBoundingClientRect();
			const e = document.elementFromPoint(r.x + r.width/2, r.y + r.height/2);
			return e === badge || badge.contains(e);
		})()`, &badgeHit)
		if !badgeHit {
			t.Error("overlay content is not hit-testable")
		}

		var passThrough bool
		s.Eval(`(() => {
			const overlay = document.querySelector("hi-overlay");
			const r = overlay.getBoundingClientRect();
			// The overlay's bottom-left corner is empty: the badge sits
			// top-trailing.
			const e = document.elementFromPoint(r.x + 4, r.bottom - 4);
			return !overlay.contains(e);
		})()`, &passThrough)
		if !passThrough {
			t.Error("empty overlay area blocks clicks to the base")
		}
	})
}

// TestGeometryRootOverlayUsesViewport pins the fixed root geometry across
// document scrolling. Both overlay grids stay on the browser viewport and the
// later overlay paints above the earlier one where their contents overlap.
func TestGeometryRootOverlayUsesViewport(t *testing.T) {
	v := hi.ScrollView(hi.Vertical,
		hi.OKLCH(0.5, 0, 0).Frame(hi.Width(600), hi.Height(1000))).
		Overlay(hi.Center, hi.Badge("first").Class("first")).
		Overlay(hi.Center, hi.Badge("second").Class("second"))
	stage(t, v, func(s *uitest.Session) {
		for i := range 2 {
			r := s.Rect("hi-overlay", i)
			within(t, "overlay left", r.X, 0, 0.5)
			within(t, "overlay top", r.Y, 0, 0.5)
			within(t, "overlay width", r.W, 600, 1)
			within(t, "overlay height", r.H, 400, 1)
		}
		s.Eval(`window.scrollTo(0, 300)`, nil)
		within(t, "scrolled first overlay top", s.Rect("hi-overlay", 0).Y, 0, 0.5)
		within(t, "scrolled second overlay top", s.Rect("hi-overlay", 1).Y, 0, 0.5)

		var secondOnTop bool
		s.Eval(`(() => {
			const r = document.querySelector(".second").getBoundingClientRect();
			return document.elementFromPoint(r.x + r.width/2, r.y + r.height/2).closest(".second") !== null;
		})()`, &secondOnTop)
		if !secondOnTop {
			t.Error("later root overlay does not paint above the earlier one")
		}
	})
}

func TestGeometryFillTerminatesAtDefiniteAncestor(t *testing.T) {
	v := hi.VStack(hi.HStack(hi.Text("a"), hi.Spacer())).Frame(hi.Width(300))
	stage(t, v, func(s *uitest.Session) {
		within(t, "row width", s.Rect("hi-hstack", 0).W, 300, 1)
	})
}

func TestGeometryTagFrameCarriesFill(t *testing.T) {
	v := hi.HStack(
		hi.HStack(hi.Text("a"), hi.Spacer()).Frame().Tag("nav"),
		hi.Text("b"),
	)
	stage(t, v, func(s *uitest.Session) {
		if w := s.Rect("nav", 0).W; w < 400 {
			t.Errorf("nav width = %g, want the row's slack to flow through the tagged frame", w)
		}
		within(t, "trailing text right edge", s.Rect("hi-text", 1).Right(), 600, 1)
		if w := s.Rect("hi-spacer", 0).W; w < 400 {
			t.Errorf("spacer width = %g, want the slack the tagged frame carried in", w)
		}
	})
}

func TestGeometryScrollViewportTakesItsFrame(t *testing.T) {
	var rows []hi.View
	for i := range 20 {
		rows = append(rows, hi.Text("Episode "+strconv.Itoa(i)))
	}
	v := hi.ScrollView(hi.Vertical, hi.VStack(rows...)).Frame(hi.Width(220), hi.Height(160))
	stage(t, v, func(s *uitest.Session) {
		scroll := s.Rect("hi-scroll", 0)
		within(t, "viewport width", scroll.W, 220, 1)
		within(t, "viewport height", scroll.H, 160, 1)
		if h := s.Rect("hi-scroll > hi-vstack", 0).H; h <= 160 {
			t.Errorf("content height = %g, want overflow to scroll against", h)
		}
	})
}

// TestGeometryOverlayAtAnchors pins the two-point geometry. At the root, the
// fixed overlay uses the viewport's at point; inside a wrapper, the ordinary
// overlay uses the base composite's at point.
func TestGeometryOverlayAtAnchors(t *testing.T) {
	v := hi.Text("base").
		Frame(hi.Width(300), hi.Height(100)).
		OverlayAt(hi.TopTrailing, hi.Center, hi.Badge("3").Class("probe"))
	for _, tt := range []struct {
		name  string
		view  hi.View
		layer string
	}{
		{"root", v, "hi-overlay"},
		{"wrapped", v.Padding(hi.Edges(0)), "hi-layer"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, tt.view, func(s *uitest.Session) {
				layer := s.Rect(tt.layer, 0)
				badge := s.Rect(".probe", 0)
				within(t, "badge center x", badge.X+badge.W/2, layer.Right(), 1)
				within(t, "badge center y", badge.Y+badge.H/2, layer.Y, 1)
			})
		})
	}
}

// TestGeometryLayerCoincidesUnderStretch pins the layer composite's
// coincidence: when a granted fill stretches a layered box past its
// content size, the base subview and the layer both track the
// container's box.
func TestGeometryLayerCoincidesUnderStretch(t *testing.T) {
	layered := hi.VStack(hi.Text("base"), hi.Secondary).Underlay(hi.Center, hi.Blue)
	v := hi.HStack(hi.Text("tall").Padding(hi.Edges(140)), layered)
	stage(t, v, func(s *uitest.Session) {
		box := s.Rect("hi-layer", 0)
		within(t, "container height", box.H, s.Rect("hi-hstack", 0).H, 1)
		within(t, "subview height", s.Rect("hi-layer > hi-vstack", 0).H, box.H, 1)
		within(t, "underlay height", s.Rect("hi-underlay", 0).H, box.H, 1)
		within(t, "underlay width", s.Rect("hi-underlay", 0).W, box.W, 1)
	})
}

// TestGeometryScrollContributesItsIdeal pins the viewport's sizing:
// its contents never contribute to an enclosing container's intrinsic
// sizing, so a content-sized row resolves from its siblings and the
// scroll axis survives. In unbounded space the viewport contributes
// its 100px ideal as a floor, with its fill stretching it past the
// ideal when a sibling resolves taller; in bounded space the ideal is
// inert, even below 100px.
func TestGeometryScrollContributesItsIdeal(t *testing.T) {
	var rows []hi.View
	for i := range 20 {
		rows = append(rows, hi.Text("Episode "+strconv.Itoa(i)))
	}
	page := func(sibling hi.View) hi.View {
		return hi.ScrollView(hi.Vertical, hi.HStack(
			sibling,
			hi.ScrollView(hi.Vertical, hi.VStack(rows...)),
		))
	}
	check := func(name string, wantH func(s *uitest.Session) float64) func(*uitest.Session) {
		return func(s *uitest.Session) {
			scroll, want := s.Rect("hi-scroll", 0), wantH(s)
			within(t, name+": viewport height", scroll.H, want, 1)
			within(t, name+": row height", s.Rect("hi-hstack", 0).H, want, 1)
			var contentH float64
			s.Eval(`document.querySelectorAll("hi-scroll")[0].scrollHeight`, &contentH)
			if contentH <= scroll.H {
				t.Errorf("%s: content height = %g, want overflow to scroll against", name, contentH)
			}
		}
	}
	sibling := func(s *uitest.Session) float64 { return s.Rect("hi-hstack > hi-padding", 0).H }
	// A short sibling: the viewport's own 100px floor wins.
	stage(t, page(hi.Text("tall").Padding(hi.Edges(32))),
		check("short sibling", func(*uitest.Session) float64 { return 100 }))
	// A tall sibling: the viewport stretches past its ideal.
	stage(t, page(hi.Text("tall").Padding(hi.Edges(140))), check("tall sibling", sibling))
	// An 80px cell: the floor does not force overflow past given space.
	stage(t, hi.ScrollView(hi.Vertical, hi.VStack(rows...)).Frame(hi.Height(80)),
		func(s *uitest.Session) {
			scroll := s.Rect("hi-scroll", 0)
			within(t, "bounded: viewport height", scroll.H, 80, 1)
			var contentH float64
			s.Eval(`document.querySelectorAll("hi-scroll")[0].scrollHeight`, &contentH)
			if contentH <= scroll.H {
				t.Errorf("bounded: content height = %g, want overflow to scroll against", contentH)
			}
		})
}

// TestGeometryDefiniteFrameIsStrict pins the strictness of a definite
// frame: the frame is exactly its declared size even when its content
// wants more — the declared size caps the frame's automatic minimum.
func TestGeometryDefiniteFrameIsStrict(t *testing.T) {
	long := strings.Repeat("overflow ", 40)
	stage(t, hi.Text(long).Frame(hi.Width(200), hi.Height(100)), func(s *uitest.Session) {
		frame := s.Rect("hi-frame", 0)
		within(t, "frame width", frame.W, 200, 1)
		within(t, "frame height", frame.H, 100, 1)
	})
}

// natImage is a data: URI for a solid SVG with a natural size of w×h.
func natImage(w, h int) string {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><rect width="100%%" height="100%%" fill="#8cf"/></svg>`, w, h)
	return "data:image/svg+xml," + url.PathEscape(svg)
}

// TestGeometryImageScalingModesFill pins the scaling modes' implied
// fills: the img — its own box — meets the imposed size in a grid cell
// and under flex pressure alike.
func TestGeometryImageScalingModesFill(t *testing.T) {
	v := hi.Image(natImage(400, 400)).
		ScaledToFill().
		Frame(hi.Width(120), hi.Height(80))
	stage(t, v, func(s *uitest.Session) {
		r := s.Rect("img", 0)
		within(t, "img width", r.W, 120, 1)
		within(t, "img height", r.H, 80, 1)
	})

	row := hi.HStack(
		hi.Image(natImage(400, 400)).ScaledToFill(),
		hi.OKLCH(0.3, 0, 0).Frame(hi.Width(500), hi.Height(40)),
	).Gap(0).Frame(hi.Width(600), hi.Height(100))
	stage(t, row, func(s *uitest.Session) {
		r := s.Rect("img", 0)
		within(t, "squeezed img width", r.W, 100, 1)
		within(t, "stretched img height", r.H, 100, 1)
	})
}

// TestGeometryNativeImageHolds pins the native framing mode: the img is
// exactly its natural size wherever it lands — here under flex pressure —
// and the enclosing machinery composes: the stack encloses the img and
// the floored text, and frame alignment places the text at the frame's
// trailing edge.
func TestGeometryNativeImageHolds(t *testing.T) {
	v := hi.HStack(
		hi.Image(natImage(200, 200)),
		hi.Text("hi"),
	).Frame(hi.Width(100), hi.Height(100), hi.Trailing)
	stage(t, v, func(s *uitest.Session) {
		img, row, frame, text := s.Rect("img", 0), s.Rect("hi-hstack", 0), s.Rect("hi-frame", 0), s.Rect("hi-text", 0)
		within(t, "img natural width", img.W, 200, 1)
		within(t, "img natural height", img.H, 200, 1)
		within(t, "row encloses the img and text", row.W, 208+text.W, 1)
		within(t, "trailing text right edge", text.Right(), frame.Right(), 1)
	})
}

// TestGeometryScaledImageIdeal pins the scaling modes on unbounded
// axes: with no definite space to fill, the img's intrinsic
// geometry answers — natural size under FixedSize, and in a
// vertical scroll the definite cross axis is filled while the
// height scales through the picture's ratio, giving the viewport
// real overflow to scroll against.
func TestGeometryScaledImageIdeal(t *testing.T) {
	stage(t, hi.Image(natImage(200, 150)).ScaledToFill().FixedSize(), func(s *uitest.Session) {
		img := s.Rect("img", 0)
		within(t, "img natural width", img.W, 200, 1)
		within(t, "img natural height", img.H, 150, 1)
	})

	v := hi.ScrollView(hi.Vertical, hi.Image(natImage(200, 150)).ScaledToFit()).
		Frame(hi.Width(400), hi.Height(100))
	stage(t, v, func(s *uitest.Session) {
		img := s.Rect("img", 0)
		within(t, "img fills the viewport width", img.W, 400, 1)
		within(t, "img height scales through the ratio", img.H, 300, 1)
	})
}

// TestGeometryRigidFrameHolds pins definite-axis rigidity: definite frames
// along a row's major axis keep their sizes under pressure — the row
// overflows — instead of being compressed by flex shrink.
func TestGeometryRigidFrameHolds(t *testing.T) {
	v := hi.HStack(
		hi.OKLCH(0.3, 0, 0).Frame(hi.Width(400), hi.Height(40)).Class("a"),
		hi.OKLCH(0.5, 0, 0).Frame(hi.Width(300), hi.Height(40)).Class("b"),
	)
	stage(t, v, func(s *uitest.Session) {
		within(t, "first frame width", s.Rect(".a", 0).W, 400, 1)
		within(t, "second frame width", s.Rect(".b", 0).W, 300, 1)
	})
}

// TestGeometryStackEnclosesItems pins stack enclosure: when its items'
// minimums exceed the available space, a stack grows to enclose them —
// overflowing its own container — instead of pinching them off at the
// available space.
func TestGeometryStackEnclosesItems(t *testing.T) {
	v := hi.HStack(
		hi.OKLCH(0.3, 0, 0).Frame(hi.Width(200), hi.Height(40)),
		hi.Text("hi"),
	).Frame(hi.Width(100), hi.Height(100), hi.Leading)
	stage(t, v, func(s *uitest.Session) {
		frame, row, text := s.Rect("hi-frame", 0), s.Rect("hi-hstack", 0), s.Rect("hi-text", 0)
		within(t, "frame width", frame.W, 100, 1)
		if text.W < 5 {
			t.Errorf("text width = %g, want its min-content floor, not squeezed away", text.W)
		}
		if row.W <= 208 {
			t.Errorf("row width = %g, want the rigid frame, the gap, and the text enclosed", row.W)
		}
		within(t, "row encloses its last item", row.Right(), text.Right(), 1)
	})
}

// TestGeometryTextFloorsAtMinContent pins the text minimum: squeezed
// below its content, text wraps down to its longest token and no
// further.
func TestGeometryTextFloorsAtMinContent(t *testing.T) {
	v := hi.VStack(
		hi.HStack(
			hi.OKLCH(0.3, 0, 0).Frame(hi.Width(550), hi.Height(20)),
			hi.Text("wrappable words").Class("probe"),
		).Class("row"),
		hi.Text("wrappable").FixedSize().Class("ref"),
	)
	stage(t, v, func(s *uitest.Session) {
		probe, ref, row := s.Rect(".probe", 0), s.Rect(".ref", 0), s.Rect(".row", 0)
		within(t, "squeezed text floors at its longest token", probe.W, ref.W, 1)
		within(t, "row encloses the floored text", row.W, 558+probe.W, 1)
		if row.W <= 600 {
			t.Errorf("row width = %g, want overflow past the stage: the floors do not fit", row.W)
		}
	})
}

// TestGeometryLineLimit pins the clamp: limited text is exactly as
// tall as its line count allows, and text within the limit keeps its
// natural height.
func TestGeometryLineLimit(t *testing.T) {
	long := "to be or not to be that is the question whether tis nobler in the mind"
	v := hi.VStack(
		hi.Text(long).LineLimit(2).Class("clamped"),
		hi.Text(long).Class("free"),
		hi.Text("short").LineLimit(2).Class("short"),
		hi.Text("ref").Class("ref"),
	).Frame(hi.Width(160))
	stage(t, v, func(s *uitest.Session) {
		line := s.Rect(".ref", 0).H
		within(t, "clamped text is two lines tall", s.Rect(".clamped", 0).H, 2*line, 1)
		if free := s.Rect(".free", 0).H; free < 3*line {
			t.Errorf("free text height = %g, want at least 3 lines (%g): the fixture does not overflow", free, 3*line)
		}
		within(t, "text within the limit keeps its height", s.Rect(".short", 0).H, line, 1)
	})
}

// A one-line limit must not restore the leading removed by TextTrim,
// whether the text fits or needs truncation.
func TestGeometryLineLimitTextTrim(t *testing.T) {
	v := hi.VStack(
		hi.Text("Save").Class("reference"),
		hi.Text("Save").TextTrim(0).Class("untrimmed"),
		hi.Text("Save").LineLimit(1).Class("short"),
		hi.Text("Save all changes to this collection and update every episode in the library").
			LineLimit(1).Class("long"),
	).TextTrim(hi.TextCap | hi.TextLastBaseline).Frame(hi.Width(120))
	stage(t, v, func(s *uitest.Session) {
		reference := s.Rect(".reference", 0)
		if reference.H <= 0 || reference.H >= s.Rect(".untrimmed", 0).H {
			t.Fatal("reference text must have a positive height with leading trimmed")
		}
		for _, selector := range []string{".short", ".long"} {
			within(t, selector+" trimmed height", s.Rect(selector, 0).H, reference.H, 0.1)
		}
		within(t, "short text hugs its content", s.Rect(".short", 0).W, reference.W, 0.1)
		within(t, "long text fits the frame", s.Rect(".long", 0).W, 120, 0.1)
	})
}

// A nowrap line must yield to the width offered through intermediate
// containers, including the space left by a rigid sibling and a gap.
func TestGeometryLineLimitWidth(t *testing.T) {
	const long = "Save all changes to this collection and update every episode in the library"
	for _, tt := range []struct {
		name  string
		wrap  func(hi.View) hi.View
		inset float64
	}{
		{"text", func(v hi.View) hi.View { return v }, 0},
		{"column", func(v hi.View) hi.View { return hi.VStack(v) }, 0},
		{"row", func(v hi.View) hi.View { return hi.HStack(v) }, 0},
		{"layers", func(v hi.View) hi.View { return hi.ZStack(v) }, 0},
		{"padding", func(v hi.View) hi.View { return v.Padding(hi.Edges(6)) }, 12},
		{"nested", func(v hi.View) hi.View { return hi.VStack(hi.HStack(hi.ZStack(v.Padding(hi.Edges(6))))) }, 12},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v := hi.HStack(
				hi.Red.Frame(hi.Width(80), hi.Height(20)).Class("rigid"),
				tt.wrap(hi.Text(long).LineLimit(1).Class("text")).Class("child"),
			).Gap(8).Class("row").Frame(hi.Width(260))
			stage(t, v, func(s *uitest.Session) {
				within(t, "row width", s.Rect(".row", 0).W, 260, 0.1)
				within(t, "rigid width", s.Rect(".rigid", 0).W, 80, 0.1)
				within(t, "child width", s.Rect(".child", 0).W, 172, 0.1)
				within(t, "text width", s.Rect(".text", 0).W, 172-tt.inset, 0.1)
				within(t, "gap", s.Rect(".child", 0).X-s.Rect(".rigid", 0).Right(), 8, 0.1)
			})
		})
	}

	stage(t, hi.VStack(
		hi.Text(long).LineLimit(2).Class("two"),
		hi.Text(long).LineLimit(1).Class("one"),
	).Frame(hi.Width(260)), func(s *uitest.Session) {
		within(t, "column width", s.Rect("hi-vstack", 0).W, 260, 0.1)
		within(t, "two-line width", s.Rect(".two", 0).W, 260, 0.1)
		within(t, "one-line width", s.Rect(".one", 0).W, 260, 0.1)
		within(t, "two-line height", s.Rect(".two", 0).H, 2*s.Rect(".one", 0).H, 0.1)
	})

	stage(t, hi.Grid(hi.Columns(2),
		hi.VStack(hi.Text(long).LineLimit(1)),
		hi.Text(long).LineLimit(1),
	).Frame(hi.Width(260)), func(s *uitest.Session) {
		for i := range 2 {
			within(t, "text fits its grid cell", s.Rect("hi-text", i).W, 126, 0.1)
		}
	})
}

// Text can shrink to zero, but a stack must still enclose its rigid
// children, gaps, and explicit minima when they exceed the offer.
func TestGeometryLineLimitMinimum(t *testing.T) {
	for _, tt := range []struct {
		name  string
		wrap  func(hi.View) hi.View
		width float64
	}{
		{"zero", func(v hi.View) hi.View { return hi.VStack(v) }, 0},
		{"explicit", func(v hi.View) hi.View { return hi.VStack(v.FrameBounds(hi.MinWidth(150))) }, 150},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v := hi.HStack(
				hi.Red.Frame(hi.Width(80), hi.Height(20)).Class("rigid"),
				tt.wrap(hi.Text("Save all changes to this collection").LineLimit(1).Class("text")).Class("child"),
			).Gap(8).Class("row").Frame(hi.Width(50))
			stage(t, v, func(s *uitest.Session) {
				within(t, "rigid child holds its width", s.Rect(".rigid", 0).W, 80, 0.1)
				within(t, "child reaches its minimum", s.Rect(".child", 0).W, tt.width, 0.1)
				within(t, "stack encloses its children", s.Rect(".row", 0).W, 88+tt.width, 0.1)
				within(t, "last child meets the stack edge", s.Rect(".child", 0).Right(), s.Rect(".row", 0).Right(), 0.1)
			})
		})
	}
}

func TestGeometryLineLimitIdeal(t *testing.T) {
	const long = "Save all changes to this collection and update every episode in the library"
	v := hi.VStack(
		hi.Text(long).FixedSize().Class("reference"),
		hi.Text(long).LineLimit(1).FixedSize().Class("direct"),
		hi.VStack(hi.Text(long).LineLimit(1).Class("nested")).FixedSize(),
		hi.VStack(hi.Text(long).LineLimit(1).Class("bounded")).Frame(hi.Width(120)).FixedSize(),
	).Frame(hi.Width(120))
	stage(t, v, func(s *uitest.Session) {
		reference := s.Rect(".reference", 0)
		if reference.W <= 120 {
			t.Fatal("reference text must exceed the offered width")
		}
		for _, selector := range []string{".direct", ".nested"} {
			within(t, selector+" ideal width", s.Rect(selector, 0).W, reference.W, 0.1)
			within(t, selector+" ideal height", s.Rect(selector, 0).H, reference.H, 0.1)
		}
		within(t, "definite frame restores bounded space", s.Rect(".bounded", 0).W, 120, 0.1)
	})
}

// TestGeometrySoftFrameTracksSpace pins the soft frame's contract in the
// three regimes it can land in: on a flex major axis it yields to the
// available space and floors at its minimum; in a grid cell and on a flex
// cross axis it shrinks to fit the cell's available space.
func TestGeometrySoftFrameTracksSpace(t *testing.T) {
	rigid := func(w float64) hi.View { return hi.OKLCH(0.5, 0, 0).Frame(hi.Width(complex(w, 0)), hi.Height(40)) }
	soft := hi.OKLCH(0.3, 0, 0).
		Frame(hi.Width(500), hi.Height(40)).FrameBounds(

		hi.MinWidth(120)).
		Class("soft")

	stage(t, hi.HStack(soft, rigid(400)).Gap(0), func(s *uitest.Session) {
		within(t, "soft width yields to space", s.Rect(".soft", 0).W, 200, 1)
	})
	stage(t, hi.HStack(soft, rigid(550)).Gap(0), func(s *uitest.Session) {
		within(t, "soft width floors at min", s.Rect(".soft", 0).W, 120, 1)
	})

	long := hi.Text(strings.Repeat("soft ", 40))
	grid := long.FrameBounds(
		hi.MinWidth(120)).
		Class("soft").
		Frame(hi.Width(200), hi.Height(200))
	stage(t, grid, func(s *uitest.Session) {
		within(t, "soft width in a grid cell", s.Rect(".soft", 0).W, 200, 1)
	})

	cross := hi.VStack(
		long.FrameBounds(
			hi.MinWidth(120)).
			Class("soft"),
	).
		Frame(hi.Width(200))
	stage(t, cross, func(s *uitest.Session) {
		within(t, "soft width on a flex cross axis", s.Rect(".soft", 0).W, 200, 1)
	})
}

// TestGeometrySoftFrameIdeal pins the ideal slots: a fixed soft frame
// takes its ideal and makes it the available space of the view inside;
// with bounded available space the same ideal is inert and the
// frame tracks space as usual.
func TestGeometrySoftFrameIdeal(t *testing.T) {
	v := hi.VStack(hi.Secondary).
		FrameBounds(hi.IdealWidth(500), hi.IdealHeight(80)).
		Class("soft")

	stage(t, v.FixedSize(), func(s *uitest.Session) {
		soft := s.Rect(".soft", 0)
		within(t, "fixed soft width takes the ideal", soft.W, 500, 1)
		within(t, "fixed soft height takes the ideal", soft.H, 80, 1)
		fill := s.Rect("hi-color", 0)
		within(t, "color fills the ideal-sized frame", fill.W, 500, 1)
		within(t, "color fills the ideal-sized frame", fill.H, 80, 1)
	})

	stage(t, v.Frame(hi.Width(600), hi.Height(300)), func(s *uitest.Session) {
		soft := s.Rect(".soft", 0)
		within(t, "bounded soft width ignores the ideal", soft.W, 600, 1)
		within(t, "bounded soft height ignores the ideal", soft.H, 300, 1)
	})
}

// TestGeometryGridColumns pins cell geometry: Columns divides the
// available width, less the gaps, equally; subviews wrap into rows in
// order; and a filling subview takes its whole cell.
func TestGeometryGridColumns(t *testing.T) {
	var cells []hi.View
	for range 5 {
		cells = append(cells, hi.Red.Frame(hi.Height(20)))
	}
	stage(t, hi.VStack(hi.Grid(hi.Columns(3), cells...)), func(s *uitest.Session) {
		grid := s.Rect("hi-grid", 0)
		within(t, "grid width", grid.W, 600, 1)
		cell := (600 - 2*8) / 3.0
		for i := range 5 {
			r := s.Rect("hi-color", i)
			within(t, fmt.Sprintf("cell %d width", i), r.W, cell, 1)
			within(t, fmt.Sprintf("cell %d left", i), r.X, float64(i%3)*(cell+8), 1)
			within(t, fmt.Sprintf("cell %d top", i), r.Y-grid.Y, float64(i/3)*(20+8), 1)
		}
	})
}

// TestGeometryGridColumnsStayEqual pins the track contract under
// pressure: a subview wider than its cell overflows it rather than
// widening its column, so the other columns keep their share.
func TestGeometryGridColumnsStayEqual(t *testing.T) {
	wide := hi.Red.Frame(hi.Width(500), hi.Height(20))
	fill := hi.Blue.Frame(hi.Height(20))
	stage(t, hi.VStack(hi.Grid(hi.Columns(3), wide, fill, fill)), func(s *uitest.Session) {
		cell := (600 - 2*8) / 3.0
		within(t, "wide subview width", s.Rect("hi-color", 0).W, 500, 1)
		within(t, "second cell left", s.Rect("hi-color", 1).X, cell+8, 1)
		within(t, "second cell width", s.Rect("hi-color", 1).W, cell, 1)
		within(t, "third cell left", s.Rect("hi-color", 2).X, 2*(cell+8), 1)
	})
}

// TestGeometryGridHugs pins the hugging size: a Columns grid of
// non-filling subviews is as wide as its widest subview times its
// column count, plus gaps, and sits at the stack's alignment.
func TestGeometryGridHugs(t *testing.T) {
	narrow := hi.Red.Frame(hi.Width(30), hi.Height(20))
	wide := hi.Blue.Frame(hi.Width(50), hi.Height(20))
	stage(t, hi.VStack(hi.Grid(hi.Columns(3), narrow, wide, narrow, narrow)), func(s *uitest.Session) {
		grid := s.Rect("hi-grid", 0)
		within(t, "grid width", grid.W, 3*50+2*8, 1)
		within(t, "grid centered", grid.X+grid.W/2, 300, 1)
		within(t, "second column left", s.Rect("hi-color", 1).X-grid.X, 50+8, 1)
	})
}

// TestGeometryGridCellMinWidth pins the adaptive layout: as many
// columns as fit at the minimum width, each widened to share the rest.
func TestGeometryGridCellMinWidth(t *testing.T) {
	var cells []hi.View
	for range 4 {
		cells = append(cells, hi.Red.Frame(hi.Height(20)))
	}
	// Three 150px columns and two gaps fit in 600px; four do not.
	stage(t, hi.VStack(hi.Grid(hi.ColumnMinWidth(150), cells...)), func(s *uitest.Session) {
		cell := (600 - 2*8) / 3.0
		within(t, "cell width", s.Rect("hi-color", 0).W, cell, 1)
		within(t, "fourth cell top", s.Rect("hi-color", 3).Y-s.Rect("hi-grid", 0).Y, 20+8, 1)
	})
}

// TestGeometryFrameRatioAnchors pins the ratio frame's contract in each
// kind of parent: on the anchor axis the frame is sized as its subview
// would be, and the other axis follows by the ratio.
func TestGeometryFrameRatioAnchors(t *testing.T) {
	color := func() hi.View { return hi.Red }
	for _, tt := range []struct {
		name string
		v    hi.View
		w, h float64
	}{
		// A fill anchored in a grid cell, a row's main axis,
		// and a column's cross axis.
		{"grid cell", hi.Grid(hi.Columns(3), color().FrameRatio(2, 3, hi.Horizontal)), (600 - 16) / 3.0, (600 - 16) / 3.0 * 1.5},
		{"row main axis", hi.HStack(color().FrameRatio(3, 1, hi.Horizontal)), 600, 200},
		{"column cross axis", hi.VStack(color().FrameRatio(3, 1, hi.Horizontal)), 600, 200},
		{"column main axis", hi.VStack(color().FrameRatio(1, 2, hi.Vertical)), 200, 400},
		{"row cross axis", hi.HStack(color().FrameRatio(1, 2, hi.Vertical)), 200, 400},
		// A definite frame anchors the other axis.
		{"definite width", color().Frame(hi.Width(90)).FrameRatio(3, 2, hi.Horizontal), 90, 60},
		{"definite height", color().Frame(hi.Height(100)).FrameRatio(2, 1, hi.Vertical), 200, 100},
		// An ideal size anchors under unbounded available space.
		{"ideal width", hi.ScrollView(hi.Horizontal, color().FrameBounds(hi.IdealWidth(120)).FrameRatio(3, 1, hi.Horizontal)), 120, 40},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, tt.v, func(s *uitest.Session) {
				r := s.Rect("hi-aspect", 0)
				within(t, "width", r.W, tt.w, 1)
				within(t, "height", r.H, tt.h, 1)
				c := s.Rect("hi-color", 0)
				within(t, "subview width", c.W, tt.w, 1)
				within(t, "subview height", c.H, tt.h, 1)
			})
		})
	}
	// A subview smaller than the frame sits at the frame's alignment,
	// in the rotated frame too.
	// A 20px square anchors each axis in turn; the derived axis is
	// larger (40) or smaller (10) than the square.
	small := func() hi.View { return hi.Blue.Frame(hi.Width(20), hi.Height(20)) }
	for _, tt := range []struct {
		name string
		v    hi.View
		w, h float64
	}{
		{"horizontal anchor", hi.VStack(small().FrameRatio(1, 2, hi.Horizontal, hi.TopLeading)), 20, 40},
		{"vertical anchor", hi.VStack(small().FrameRatio(1, 2, hi.Vertical, hi.TopLeading)), 10, 20},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stage(t, tt.v, func(s *uitest.Session) {
				frame, sub := s.Rect("hi-aspect", 0), s.Rect("hi-color", 0)
				within(t, "frame width", frame.W, tt.w, 1)
				within(t, "frame height", frame.H, tt.h, 1)
				within(t, "subview left", sub.X, frame.X, 1)
				within(t, "subview top", sub.Y, frame.Y, 1)
			})
		})
	}
	// An intrinsic size anchors a non-filling view.
	stage(t, hi.HStack(hi.Text("hello").FrameRatio(1, 1, hi.Horizontal), hi.Text("b")), func(s *uitest.Session) {
		r := s.Rect("hi-aspect", 0)
		within(t, "square", r.H, r.W, 1)
		within(t, "width is the text's", r.W, s.Rect("hi-text", 0).W, 1)
	})
}

// TestGeometryStickyPins pins the sticky contract in a scrolling
// viewport: a section heading holds at the viewport's top edge while
// its section scrolls by, paints over the rows passing under it, and
// is pushed out by its section's end as the next section arrives —
// whereupon the next section's heading takes its place.
func TestGeometryStickyPins(t *testing.T) {
	section := func(i int) hi.View {
		rows := []hi.View{
			hi.Text(fmt.Sprintf("Heading %d", i)).
				Padding(hi.Edges(8)).
				Background(hi.Accent).
				Sticky().
				Class("heading"),
		}
		for r := range 5 {
			rows = append(rows, hi.Text(fmt.Sprintf("Row %d.%d", i, r)).
				Padding(hi.Edges(12)).
				BorderStroke(1, hi.Secondary))
		}
		return hi.VStack(rows...).Gap(0).Alignment(hi.Leading)
	}
	v := hi.ScrollView(hi.Vertical,
		hi.VStack(section(0), section(1)).Gap(0).Alignment(hi.Leading),
	).
		Frame(hi.Width(300), hi.Height(200))
	stage(t, v, func(s *uitest.Session) {
		viewport := s.Rect("hi-scroll", 0)
		sec := s.Rect("hi-vstack", 1)
		heading := s.Rect(".heading", 0)
		scroll := func(y float64) {
			s.Eval(fmt.Sprintf(`document.querySelector("hi-scroll").scrollTop = %g`, y), nil)
		}

		// Half a heading into the first section:
		// the heading holds at the viewport's top edge.
		scroll(heading.H / 2)
		h := s.Rect(".heading", 0)
		within(t, "pinned heading top", h.Y, viewport.Y, 1)

		// The pinned heading paints over the row scrolled under it.
		var hit string
		s.Eval(fmt.Sprintf(`document.elementFromPoint(%g, %g).textContent`, h.X+h.W/2, h.Y+h.H/2), &hit)
		if hit != "Heading 0" {
			t.Errorf("element painted over pinned heading center = %q, want %q", hit, "Heading 0")
		}

		// As the next section arrives, the first heading is pushed
		// out along with its section's end.
		scroll(sec.H - heading.H/2)
		h = s.Rect(".heading", 0)
		within(t, "pushed-out heading bottom", h.Bottom(), viewport.Y+heading.H/2, 1)

		// The next section's heading holds in its place.
		scroll(sec.H + heading.H)
		within(t, "next pinned heading top", s.Rect(".heading", 1).Y, viewport.Y, 1)
	})
}

// TestGeometryGalleryFits loads the full fixture gallery and checks nothing
// forces its scroll viewport to scroll horizontally.
func TestGeometryGalleryFits(t *testing.T) {
	html, err := fixture.Document(staticCSS)
	if err != nil {
		t.Fatalf("fixture.Document: %v", err)
	}
	uitest.Run(t, 1000, 800, html, func(s *uitest.Session) {
		var over bool
		s.Eval(`(() => {
			const e = document.querySelector("hi-scroll");
			return e.scrollWidth > e.clientWidth;
		})()`, &over)
		if over {
			t.Error("gallery overflows horizontally")
		}
	})
}
