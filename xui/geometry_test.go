package ui_test

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/fixture"
	"ily.dev/act3/xui/internal/uitest"
	"ily.dev/domi"
)

func TestMain(m *testing.M) { os.Exit(uitest.Main(m)) }

// stage renders v as the page root of a 600x400 viewport — the definite
// frame every fill chain terminates at — and hands the loaded page to fn.
func stage(t *testing.T, v ui.View, fn func(*uitest.Session)) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("<!doctype html><meta charset=utf-8><style>")
	sb.WriteString(ui.CSS)
	sb.WriteString(`</style><body>`)
	if err := domi.RenderTo(&sb, new(ui.Renderer).Render(v)); err != nil {
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
	stage(t, ui.HStack(ui.Text("a"), ui.Spacer(), ui.Text("b")), func(s *uitest.Session) {
		within(t, "row width", s.Rect(".ui-hstack", 0).W, 600, 1)
		if w := s.Rect(".ui-spacer", 0).W; w < 400 {
			t.Errorf("spacer width = %g, want most of the row's slack", w)
		}
		within(t, "trailing text right edge", s.Rect(".ui-text", 1).Right(), 600, 1)
	})
	stage(t, ui.VStack(ui.Text("a"), ui.Spacer(), ui.Text("b")), func(s *uitest.Session) {
		within(t, "column height", s.Rect(".ui-vstack", 0).H, 400, 1)
		if h := s.Rect(".ui-spacer", 0).H; h < 300 {
			t.Errorf("spacer height = %g, want most of the column's slack", h)
		}
		within(t, "trailing text bottom edge", s.Rect(".ui-text", 1).Bottom(), 400, 1)
	})
}

// TestGeometryRootIsViewport pins the root contract: the rendered root
// covers the viewport exactly, and the viewport acts as a definite
// frame — the root view's fills on both axes terminate at it.
func TestGeometryRootIsViewport(t *testing.T) {
	stage(t, ui.Muted, func(s *uitest.Session) {
		root := s.Rect("ui-root", 0)
		within(t, "root x", root.X, 0, 0.5)
		within(t, "root y", root.Y, 0, 0.5)
		within(t, "root width", root.W, 600, 1)
		within(t, "root height", root.H, 400, 1)
		fill := s.Rect(".ui-color", 0)
		within(t, "color fills the viewport", fill.W, 600, 1)
		within(t, "color fills the viewport", fill.H, 400, 1)
	})
}

// TestGeometrySpacerMinimumLength pins the spacer's floor: under
// pressure a spacer holds a small minimum along its expansion axis
// instead of letting its neighbors fuse, and with unbounded available
// space the same minimum is its answer.
func TestGeometrySpacerMinimumLength(t *testing.T) {
	rigid := func(w int) ui.View { return ui.Color("#567").Frame(ui.Width(w), ui.Height(40)) }
	stage(t, ui.HStack(rigid(300), ui.Spacer(), rigid(300)).Gap(0), func(s *uitest.Session) {
		within(t, "squeezed spacer floors at the minimum", s.Rect(".ui-spacer", 0).W, 8, 1)
	})
	stage(t, ui.HStack(ui.Text("a"), ui.Spacer(), ui.Text("b")).Gap(0).FixedSize(), func(s *uitest.Session) {
		within(t, "fixed spacer takes the minimum", s.Rect(".ui-spacer", 0).W, 8, 1)
	})
}

func TestGeometryDividerSpansMinorAxis(t *testing.T) {
	v := ui.HStack(ui.Text("a"), ui.Divider(), ui.Text("tall").Padding(ui.Edges(32)))
	stage(t, v, func(s *uitest.Session) {
		row, div := s.Rect(".ui-hstack", 0), s.Rect(".ui-divider", 0)
		within(t, "divider height", div.H, row.H, 1)
		within(t, "divider width", div.W, 1, 0.5)
	})
}

func TestGeometryFrameSubviewKeepsIntrinsicSize(t *testing.T) {
	stage(t, ui.Text("hi").Frame(ui.Width(120), ui.Height(120)), func(s *uitest.Session) {
		frame, text := s.Rect(".ui-frame", 0), s.Rect(".ui-text", 0)
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
	v := ui.HStack(ui.Spacer().Frame(ui.Width(100)), ui.Text("b"))
	stage(t, v, func(s *uitest.Session) {
		within(t, "framed spacer width", s.Rect(".ui-frame", 0).W, 100, 1)
		if w := s.Rect(".ui-hstack", 0).W; w > 300 {
			t.Errorf("row width = %g, want shrink-wrapped: the definite frame settled the only fill", w)
		}
	})
}

// TestGeometryDefiniteFillThroughContainers pins fill termination across
// boxless containers: a want killed by its requester's definite frame stays
// dead through Group and a tagged frame, while a sibling's live want merged with it
// stays live.
func TestGeometryDefiniteFillThroughContainers(t *testing.T) {
	framed := func() ui.View { return ui.Spacer().Frame(ui.Width(100)) }
	cases := []struct {
		name     string
		v        ui.View
		wantGrow bool
	}{
		{"group", ui.HStack(ui.Group(framed()), ui.Text("b")), false},
		{"tagged frame", ui.HStack(framed().Frame().Tag("nav"), ui.Text("b")), false},
		{"group with live sibling", ui.HStack(ui.Group(framed(), ui.Spacer()), ui.Text("b")), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stage(t, c.v, func(s *uitest.Session) {
				w := s.Rect(".ui-hstack", 0).W
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
	v := ui.Text(strings.Repeat("overflow ", 8)).
		FixedSize().
		Frame(ui.Width(100), ui.Height(100))
	stage(t, v, func(s *uitest.Session) {
		frame, text := s.Rect(".ui-frame", 0), s.Rect(".ui-text", 0)
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
	v := ui.Text("base").
		Frame(ui.Width(300), ui.Height(100)).
		LayerOver(ui.TopTrailing, ui.Badge("hit").Class("probe"))
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
			const overlay = document.querySelector(".ui-overlay");
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

func TestGeometryFillTerminatesAtDefiniteAncestor(t *testing.T) {
	v := ui.VStack(ui.HStack(ui.Text("a"), ui.Spacer())).Frame(ui.Width(300))
	stage(t, v, func(s *uitest.Session) {
		within(t, "row width", s.Rect(".ui-hstack", 0).W, 300, 1)
	})
}

func TestGeometryTagFrameCarriesFill(t *testing.T) {
	v := ui.HStack(
		ui.HStack(ui.Text("a"), ui.Spacer()).Frame().Tag("nav"),
		ui.Text("b"),
	)
	stage(t, v, func(s *uitest.Session) {
		if w := s.Rect("nav", 0).W; w < 400 {
			t.Errorf("nav width = %g, want the row's slack to flow through the tagged frame", w)
		}
		within(t, "trailing text right edge", s.Rect(".ui-text", 1).Right(), 600, 1)
		if w := s.Rect(".ui-spacer", 0).W; w < 400 {
			t.Errorf("spacer width = %g, want the slack the tagged frame carried in", w)
		}
	})
}

func TestGeometryScrollViewportTakesItsFrame(t *testing.T) {
	var rows []ui.View
	for i := range 20 {
		rows = append(rows, ui.Text("Episode "+strconv.Itoa(i)))
	}
	v := ui.ScrollView(ui.Vertical, ui.VStack(rows...)).Frame(ui.Width(220), ui.Height(160))
	stage(t, v, func(s *uitest.Session) {
		scroll := s.Rect(".ui-scroll", 0)
		within(t, "viewport width", scroll.W, 220, 1)
		within(t, "viewport height", scroll.H, 160, 1)
		if h := s.Rect(".ui-scroll > .ui-vstack", 0).H; h <= 160 {
			t.Errorf("content height = %g, want overflow to scroll against", h)
		}
	})
}

// TestGeometryDefiniteFrameIsStrict pins the strictness of a definite
// frame: the frame is exactly its declared size even when its content
// wants more — the declared size caps the frame's automatic minimum.
func TestGeometryDefiniteFrameIsStrict(t *testing.T) {
	long := strings.Repeat("overflow ", 40)
	stage(t, ui.Text(long).Frame(ui.Width(200), ui.Height(100)), func(s *uitest.Session) {
		frame := s.Rect(".ui-frame", 0)
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
	v := ui.Image(natImage(400, 400)).
		FramedAs(ui.ScaledToFill).
		Frame(ui.Width(120), ui.Height(80))
	stage(t, v, func(s *uitest.Session) {
		r := s.Rect(".ui-image", 0)
		within(t, "img width", r.W, 120, 1)
		within(t, "img height", r.H, 80, 1)
	})

	row := ui.HStack(
		ui.Image(natImage(400, 400)).FramedAs(ui.ScaledToFill),
		ui.Color("#345").Frame(ui.Width(500), ui.Height(40)),
	).Gap(0).Frame(ui.Width(600), ui.Height(100))
	stage(t, row, func(s *uitest.Session) {
		r := s.Rect(".ui-image", 0)
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
	v := ui.HStack(
		ui.Image(natImage(200, 200)).FramedAs(ui.Native),
		ui.Text("hi"),
	).Frame(ui.Width(100), ui.Height(100), ui.Trailing)
	stage(t, v, func(s *uitest.Session) {
		img, row, frame, text := s.Rect("img", 0), s.Rect(".ui-hstack", 0), s.Rect(".ui-frame", 0), s.Rect(".ui-text", 0)
		within(t, "img natural width", img.W, 200, 1)
		within(t, "img natural height", img.H, 200, 1)
		within(t, "row encloses the img and text", row.W, 208+text.W, 1)
		within(t, "trailing text right edge", text.Right(), frame.Right(), 1)
	})
}

// TestGeometryScaledImageIdeal pins the scaling modes on unbounded
// axes: with no box to meet, the img's intrinsic geometry answers —
// natural size under FixedSize, and in a vertical scroll the bounded
// cross axis is met while the height scales through the picture's
// ratio, giving the viewport real overflow to scroll against.
func TestGeometryScaledImageIdeal(t *testing.T) {
	stage(t, ui.Image(natImage(200, 150)).FramedAs(ui.ScaledToFill).FixedSize(), func(s *uitest.Session) {
		img := s.Rect(".ui-image", 0)
		within(t, "img natural width", img.W, 200, 1)
		within(t, "img natural height", img.H, 150, 1)
	})

	v := ui.ScrollView(ui.Vertical, ui.Image(natImage(200, 150)).FramedAs(ui.ScaledToFit)).
		Frame(ui.Width(400), ui.Height(100))
	stage(t, v, func(s *uitest.Session) {
		img := s.Rect(".ui-image", 0)
		within(t, "img fills the viewport width", img.W, 400, 1)
		within(t, "img height scales through the ratio", img.H, 300, 1)
	})
}

// TestGeometryRigidFrameHolds pins definite-axis rigidity: definite frames
// along a row's major axis keep their sizes under pressure — the row
// overflows — instead of being compressed by flex shrink.
func TestGeometryRigidFrameHolds(t *testing.T) {
	v := ui.HStack(
		ui.Color("#345").Frame(ui.Width(400), ui.Height(40)).Class("a"),
		ui.Color("#567").Frame(ui.Width(300), ui.Height(40)).Class("b"),
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
	v := ui.HStack(
		ui.Color("#345").Frame(ui.Width(200), ui.Height(40)),
		ui.Text("hi"),
	).Frame(ui.Width(100), ui.Height(100), ui.Leading)
	stage(t, v, func(s *uitest.Session) {
		frame, row, text := s.Rect(".ui-frame", 0), s.Rect(".ui-hstack", 0), s.Rect(".ui-text", 0)
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
	v := ui.VStack(
		ui.HStack(
			ui.Color("#345").Frame(ui.Width(550), ui.Height(20)),
			ui.Text("wrappable words").Class("probe"),
		).Class("row"),
		ui.Text("wrappable").FixedSize().Class("ref"),
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

// TestGeometryStackHugsCappedFill pins fill absorption at a definite
// maximum: a stack over a capped greedy subview is exactly its bounding
// rectangle — no dead space where a relayed fill would have grown the
// stack — while the capped frame still yields to flex pressure and to
// a cell smaller than its maximum.
func TestGeometryStackHugsCappedFill(t *testing.T) {
	capped := ui.Color("#345").FrameBounds(ui.MaxWidth(300)).Class("capped")

	stage(t, ui.HStack(capped).Class("outer"), func(s *uitest.Session) {
		within(t, "capped frame width", s.Rect(".capped", 0).W, 300, 1)
		within(t, "stack hugs the capped fill", s.Rect(".outer", 0).W, 300, 1)
	})

	squeezed := ui.HStack(
		capped,
		ui.Color("#567").Frame(ui.Width(450), ui.Height(40)),
	).Gap(0).Class("outer")
	stage(t, squeezed, func(s *uitest.Session) {
		within(t, "squeezed capped width", s.Rect(".capped", 0).W, 150, 1)
		within(t, "row is the bounding rectangle", s.Rect(".outer", 0).W, 600, 1)
	})

	stage(t, capped.Frame(ui.Width(200), ui.Height(100)), func(s *uitest.Session) {
		within(t, "small cell clamps the claim", s.Rect(".capped", 0).W, 200, 1)
	})

	// The cross axis absorbs the same way: the claim caps the row's
	// height contribution at the maximum, not at the fill.
	cross := ui.HStack(
		ui.Color("#345").FrameBounds(ui.MaxHeight(50)).Class("vcapped"),
		ui.Color("#567").Frame(ui.Width(40), ui.Height(80)),
	).Class("vrow")
	stage(t, cross, func(s *uitest.Session) {
		within(t, "cross-capped height", s.Rect(".vcapped", 0).H, 50, 1)
		within(t, "row height hugs the tallest item", s.Rect(".vrow", 0).H, 80, 1)
	})
}

// TestGeometrySoftFrameTracksSpace pins the soft frame's contract in the
// three regimes it can land in: on a flex major axis it yields to the
// available space and floors at its minimum; in a grid cell and on a flex
// cross axis it shrinks to fit the cell's available space.
func TestGeometrySoftFrameTracksSpace(t *testing.T) {
	rigid := func(w int) ui.View { return ui.Color("#567").Frame(ui.Width(w), ui.Height(40)) }
	soft := ui.Color("#345").
		Frame(ui.Width(500), ui.Height(40)).FrameBounds(

		ui.MinWidth(120)).
		Class("soft")

	stage(t, ui.HStack(soft, rigid(400)).Gap(0), func(s *uitest.Session) {
		within(t, "soft width yields to space", s.Rect(".soft", 0).W, 200, 1)
	})
	stage(t, ui.HStack(soft, rigid(550)).Gap(0), func(s *uitest.Session) {
		within(t, "soft width floors at min", s.Rect(".soft", 0).W, 120, 1)
	})

	long := ui.Text(strings.Repeat("soft ", 40))
	grid := long.FrameBounds(
		ui.MinWidth(120)).
		Class("soft").
		Frame(ui.Width(200), ui.Height(200))
	stage(t, grid, func(s *uitest.Session) {
		within(t, "soft width in a grid cell", s.Rect(".soft", 0).W, 200, 1)
	})

	cross := ui.VStack(
		long.FrameBounds(
			ui.MinWidth(120)).
			Class("soft"),
	).
		Frame(ui.Width(200))
	stage(t, cross, func(s *uitest.Session) {
		within(t, "soft width on a flex cross axis", s.Rect(".soft", 0).W, 200, 1)
	})
}

// TestGeometrySoftFrameIdeal pins the ideal slots: a fixed soft frame
// takes its ideal, clamped by its own bounds, and makes the result the
// available space of the view inside; with bounded available space the
// same ideal is inert and the
// frame tracks space as usual.
func TestGeometrySoftFrameIdeal(t *testing.T) {
	v := ui.VStack(ui.Muted).
		FrameBounds(ui.IdealWidth(500), ui.MaxWidth(300), ui.IdealHeight(80)).
		Class("soft")

	stage(t, v.FixedSize(), func(s *uitest.Session) {
		soft := s.Rect(".soft", 0)
		within(t, "fixed soft width takes the clamped ideal", soft.W, 300, 1)
		within(t, "fixed soft height takes the ideal", soft.H, 80, 1)
		fill := s.Rect(".ui-color", 0)
		within(t, "color fills the ideal-sized frame", fill.W, 300, 1)
		within(t, "color fills the ideal-sized frame", fill.H, 80, 1)
	})

	stage(t, v.Frame(ui.Width(600), ui.Height(300)), func(s *uitest.Session) {
		soft := s.Rect(".soft", 0)
		within(t, "bounded soft width ignores the ideal", soft.W, 300, 1)
		within(t, "bounded soft height ignores the ideal", soft.H, 300, 1)
	})

	// Bounds clamp sizes, not queries: adding a maximum must never make
	// the subview bigger. With no available space to bound, the color
	// takes the same 10px defaults it would take without the frame.
	stage(t, ui.VStack(ui.Muted).FrameBounds(ui.MaxWidth(300)).FixedSize(), func(s *uitest.Session) {
		fill := s.Rect(".ui-color", 0)
		within(t, "color keeps its default under a max", fill.W, 10, 1)
		within(t, "color keeps its default under a max", fill.H, 10, 1)
	})
}

// TestGeometryGalleryFits loads the full fixture gallery and checks nothing
// forces its scroll viewport to scroll horizontally.
func TestGeometryGalleryFits(t *testing.T) {
	html, err := fixture.Document()
	if err != nil {
		t.Fatalf("fixture.Document: %v", err)
	}
	uitest.Run(t, 1000, 800, html, func(s *uitest.Session) {
		var over bool
		s.Eval(`(() => {
			const e = document.querySelector(".ui-scroll");
			return e.scrollWidth > e.clientWidth;
		})()`, &over)
		if over {
			t.Error("gallery overflows horizontally")
		}
	})
}
