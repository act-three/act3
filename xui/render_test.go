package ui_test

import (
	"strconv"
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/domi"
)

// The examples from the DOM-45 design document, transcribed against the
// prototype's API. They double as the spec for what the slice must render.

type Msg struct {
	EditProfile bool
	NewMovie    bool
	Watched     uint64
}

type User struct {
	Name     string
	Email    string
	PhotoURL string
}

type Movie struct {
	ID        uint64
	Title     string
	Summary   string
	PosterURL string
}

func accountCard(user User) ui.View {
	return ui.Card(
		ui.HStack(
			ui.Image(user.PhotoURL).
				Alt(user.Name).
				FramedAs(ui.ScaledToFill).
				Frame(ui.Width(48), ui.Height(48)).
				BorderShape(ui.Ellipse),
			ui.VStack(
				ui.Text(user.Name).Font(ui.Headline),
				ui.Text(user.Email).Foreground(ui.Muted),
			).Gap(4).Alignment(ui.Leading),
			ui.Spacer(),
			ui.Button(ui.Text("Edit"), Msg{EditProfile: true}).Role(ui.RolePrimary),
		).Gap(12).Alignment(ui.Center),
	).
		Padding(ui.Edges(16)).
		LayerUnder(ui.Center, ui.Color("#fff")).
		LayerOver(ui.TopTrailing, ui.Badge("Pro"))
}

func moviePage(movies []Movie) ui.View {
	return ui.VStack(
		ui.HStack(
			ui.Text("Movies").Font(ui.Title),
			ui.Spacer(),
			ui.Button(ui.Text("New"), Msg{NewMovie: true}).Role(ui.RolePrimary),
		).Alignment(ui.Center),
		ui.For(movies, movieKey, movieRow),
	).
		Gap(16).
		Padding(ui.Edges(32))
}

func movieKey(m Movie) string { return strconv.FormatUint(m.ID, 10) }

func movieRow(movie Movie) ui.View {
	return ui.HStack(
		ui.Image(movie.PosterURL).
			Alt(movie.Title).
			FramedAs(ui.ScaledToFill).
			Frame(ui.Width(64), ui.Height(96)),
		ui.VStack(
			ui.Text(movie.Title).Font(ui.Headline),
			ui.Text(movie.Summary).Foreground(ui.Muted),
		).Gap(4),
		ui.Spacer(),
		ui.Button(ui.Text("Watched"), Msg{Watched: movie.ID}),
	).Gap(12).Padding(ui.Edges(12))
}

func render(t *testing.T, v ui.View) string {
	t.Helper()
	var sb strings.Builder
	if err := domi.RenderTo(&sb, ui.Render(v)); err != nil {
		t.Fatalf("render: %v", err)
	}
	return sb.String()
}

func TestAccountCard(t *testing.T) {
	html := render(t, accountCard(User{
		Name:     "Ada Lovelace",
		Email:    "ada@example.com",
		PhotoURL: "/ada.jpg",
	}))

	wants := []string{
		`<ui-root>`, // root
		`class="ui-hstack ui-card ui-cell-fill-x"`, // Card: a VStack with the surface class
		`class="ui-hstack ui-grow"`,                // the Spacer's fill stretches the row across the card
		`ui-border-ellipse`,                        // BorderShape applied to the image frame
		`ui-frame`,                                 // Size(48) introduces a frame wrapper
		`width:48px`,                               // ...with the resolved size
		`class="ui-spacer ui-spacer-h ui-grow"`,
		`class="ui-hstack ui-button ui-role-primary"`,
		`class="ui-layers ui-cell-fill-x"`, // Underlay + Overlay decoration layers
		`class="ui-underlay"`,
		`class="ui-overlay"`,
		`ui-border-capsule`, // the Badge's pill
		`Pro`,
		`Ada Lovelace`,
	}
	for _, w := range wants {
		if !strings.Contains(html, w) {
			t.Errorf("account card HTML missing %q\n\n%s", w, html)
		}
	}
}

func TestMoviePageFillPropagation(t *testing.T) {
	html := render(t, moviePage([]Movie{
		{ID: 1, Title: "Metropolis", Summary: "A city divided.", PosterURL: "/m.jpg"},
		{ID: 2, Title: "Solaris", Summary: "An ocean that thinks.", PosterURL: "/s.jpg"},
	}))

	wants := []string{
		// The header HStack contains a Spacer, so it fills horizontally —
		// the minor axis of the enclosing VStack, lowered as a self-stretch.
		`class="ui-hstack ui-stretch"`,
		// The outer VStack inherits that horizontal fill; at the root, a
		// grid, it lowers to a cell stretch.
		`class="ui-vstack ui-cell-fill-x"`,
		// Both movie rows rendered via For, each with its own Spacer.
		`Metropolis`,
		`Solaris`,
	}
	for _, w := range wants {
		if !strings.Contains(html, w) {
			t.Errorf("movie page HTML missing %q\n\n%s", w, html)
		}
	}

	// The For helper splices rows directly into the VStack, so there are two
	// movie-row spacers plus the header spacer: three in total.
	if got := strings.Count(html, "ui-spacer-h"); got != 3 {
		t.Errorf("ui-spacer count = %d, want 3\n\n%s", got, html)
	}
}

// TestTagNamesElement checks that Tag names the element of a view that is
// otherwise an anonymous div — directly on a stack, and on a frame, where
// the named element still carries its fill (it grows along the enclosing
// row's main axis) and the enclosing stack keeps propagating the request
// toward a definite ancestor.
func TestTagNamesElement(t *testing.T) {
	if html := render(t, ui.VStack(ui.Text("a")).Tag("ul")); !strings.Contains(html, `<ul class="ui-vstack"`) {
		t.Errorf("Tag should rename the stack's own element:\n%s", html)
	}

	html := render(t, ui.HStack(
		ui.HStack(ui.Text("a"), ui.Spacer()).Frame().Tag("nav"),
		ui.Text("b"),
	))
	for _, w := range []string{
		`<nav class="ui-frame ui-grow"`,         // the tagged frame carries the fill
		`class="ui-hstack ui-cell-fill-x"`,      // ...and the root stack keeps it
		`class="ui-spacer ui-spacer-h ui-grow"`, // the inner row distributes slack
	} {
		if !strings.Contains(html, w) {
			t.Errorf("tagged-frame fill chain missing %q:\n%s", w, html)
		}
	}
}

// TestTagPanicsOnNamedElement pins Tag's contract: naming an element that
// already has a name panics, whether the name is the view's own or came from
// an earlier Tag.
func TestTagPanicsOnNamedElement(t *testing.T) {
	for _, tt := range []struct {
		name string
		v    ui.View
	}{
		{"intrinsic tag", ui.Button(ui.Text("x"), Msg{}).Tag("figure")},
		{"double Tag", ui.VStack().Tag("ul").Tag("ol")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("rendering did not panic")
				}
			}()
			ui.Render(tt.v)
		})
	}
}

// TestDomiModifierPanics pins the [ui.HTML] contract: the wrapped node is
// opaque, so a modifier has no element to act on and applying one panics
// instead of silently dropping the effect.
func TestDomiModifierPanics(t *testing.T) {
	for _, tt := range []struct {
		name string
		v    ui.View
	}{
		{"style", ui.HTML(domi.Text("raw")).Class("x")},
		{"structural", ui.HTML(domi.Text("raw")).Padding(ui.Edges(4))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("modifying a Domi view did not panic")
				}
			}()
			render(t, tt.v)
		})
	}
}

// TestImmutableModifiers verifies the load-bearing value-semantics invariant:
// applying a modifier to a shared view must not affect the original.
func TestImmutableModifiers(t *testing.T) {
	header := ui.Text("Title").Font(ui.Title)

	padded := header.Padding(ui.Edges(16))
	plain := header.Padding(ui.Edges(0))

	ph := render(t, padded)
	if !strings.Contains(ph, "padding:16px") {
		t.Errorf("padded view lost its padding:\n%s", ph)
	}
	if pl := render(t, plain); strings.Contains(pl, "16px") {
		t.Errorf("sibling view leaked padding from the other branch:\n%s", pl)
	}
}

// TestColorAsView pins Color's View implementation: a Color renders as a
// solid fill of itself requesting fill on both axes, the zero Color renders
// black, and generic modifiers reach the fill's box.
func TestColorAsView(t *testing.T) {
	html := render(t, ui.Muted)
	if !strings.Contains(html, `class="ui-color-paint" style="background-color:var(--ui-color-muted)"`) {
		t.Errorf("color view should paint itself as its element's content:\n%s", html)
	}
	if !strings.Contains(html, `class="ui-color ui-cell-fill-x ui-cell-fill-y"`) {
		t.Errorf("color view should request fill on both axes:\n%s", html)
	}
	if zero := render(t, ui.Color("")); !strings.Contains(zero, "background-color:#000") {
		t.Errorf("zero color view should render black:\n%s", zero)
	}
	if mod := render(t, ui.Color("#eee").Opacity(0.5)); !strings.Contains(mod, "opacity:0.5") {
		t.Errorf("modifier on a color view should reach its box:\n%s", mod)
	}

	// Background merges onto the color's own element and paints under the
	// inner color content, visible where c is translucent — no extra layer,
	// and the Modify spelling is the same merge.
	bg := render(t, ui.Color("#0008").Background("#fff"))
	for _, w := range []string{"background-color:#0008", `style="background-color:#fff"`} {
		if !strings.Contains(bg, w) {
			t.Errorf("Background under a color missing %q:\n%s", w, bg)
		}
	}
	if strings.Contains(bg, "ui-underlay") {
		t.Errorf("Background on a color should merge, not add a layer:\n%s", bg)
	}
	if mod := render(t, ui.Color("#0008").Modify(ui.Background("#fff"))); mod != bg {
		t.Errorf("Modify(Background) diverged from the Background method:\n%s", mod)
	}
	// Underlay layers content behind the color.
	under := render(t, ui.Color("#0008").LayerUnder(ui.Center, ui.Text("behind")))
	for _, w := range []string{`class="ui-underlay"`, "behind"} {
		if !strings.Contains(under, w) {
			t.Errorf("Underlay behind a color missing %q:\n%s", w, under)
		}
	}
	// Modifiers with no possible effect on a color are no-ops.
	if noop := render(t, ui.Muted.Foreground("#fff").Font(ui.Title)); noop != html {
		t.Errorf("no-effect modifiers on a color should be no-ops:\n%s", noop)
	}
}

// TestPaddingComposes checks that separate Padding modifiers compose by
// nesting instead of clobbering each other: each emits its own declarations,
// padding only the edges it names.
func TestPaddingComposes(t *testing.T) {
	html := render(t, ui.Text("hi").Padding(ui.EdgeTop(8)).Padding(ui.EdgesPillarbox(12)))
	for _, w := range []string{
		"padding-block:8px 0px",
		"padding-inline:0px",
		"padding-block:0px",
		"padding-inline:12px",
	} {
		if !strings.Contains(html, w) {
			t.Errorf("composed padding missing %q:\n%s", w, html)
		}
	}
}

// TestIdealSize pins the unbounded-space context: FixedSize marks its
// subtree's available space as unbounded; the space-adaptive views take
// a deliberate default on each unbounded axis (Color 10px, ScrollView
// 100px, Divider 10px along its length); and unbounded clears per axis
// wherever a box makes real space available again — a frame's definite
// axes, a decoration layer, a
// scroll viewport.
func TestIdealSize(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       ui.View
		wants   []string
		rejects []string
	}{
		{
			"direct FixedSize",
			ui.Muted.FixedSize(),
			[]string{"width:10px", "height:10px"},
			nil,
		},
		{
			"FixedSize on an ancestor",
			ui.VStack(ui.Muted).FixedSize(),
			[]string{"width:10px", "height:10px"},
			nil,
		},
		{
			"definite frame axis clears its axis only",
			ui.VStack(ui.Muted).Frame(ui.Width(200)).FixedSize(),
			[]string{"height:10px"},
			[]string{"width:10px"},
		},
		{
			"both definite axes clear both",
			ui.VStack(ui.Muted).Frame(ui.Width(200), ui.Height(100)).FixedSize(),
			nil,
			[]string{"10px"},
		},
		{
			"scroll viewport takes 100px; content unbounded on the scroll axis",
			ui.ScrollView(ui.Vertical, ui.Muted).FixedSize(),
			[]string{"width:100px", "height:100px", "height:10px"},
			[]string{"width:10px"},
		},
		{
			"scroll axis is unbounded without FixedSize",
			ui.ScrollView(ui.Vertical, ui.Muted),
			[]string{"height:10px", "ui-cell-fill-x"},
			[]string{"width:10px"},
		},
		{
			"both-axes scroll makes both content axes unbounded",
			ui.ScrollView(ui.Horizontal|ui.Vertical, ui.Muted),
			[]string{"width:10px", "height:10px"},
			nil,
		},
		{
			"no-axis scroll makes neither content axis unbounded",
			ui.ScrollView(ui.AxisSet(0), ui.Muted).FixedSize(),
			[]string{"width:100px", "height:100px", "ui-scroll-none"},
			[]string{"10px"},
		},
		{
			"bounds frame takes its ideal and makes it the subview's space",
			ui.VStack(ui.Muted).FrameBounds(ui.IdealWidth(200), ui.IdealHeight(80)).FixedSize(),
			[]string{"width:200px", "height:80px"},
			[]string{"10px"},
		},
		{
			"bounds frame ideal is inert in bounded space",
			ui.Text("x").FrameBounds(ui.IdealWidth(200)),
			nil,
			[]string{"width"},
		},
		{
			// Bounds clamp sizes, not queries: adding a maximum must
			// never make the subview bigger. The color takes the same
			// 10px defaults it would take without the frame, and the
			// max clamps only the frame's answer.
			"a definite max passes the query through",
			ui.VStack(ui.Muted).FrameBounds(ui.MaxWidth(300)).FixedSize(),
			[]string{"width:10px", "height:10px", "max-width:300px"},
			[]string{"grid-template"},
		},
		{
			// A scaling mode meets an imposed box; with no box to
			// meet, the img's intrinsic geometry answers instead.
			"scaled image drops its fills on unbounded axes",
			ui.Image("/x.png").FramedAs(ui.ScaledToFill).FixedSize(),
			[]string{"object-fit:cover"},
			[]string{"ui-cell-fill"},
		},
		{
			// The viewport itself stays greedy on both axes; only
			// the image's own fill is dropped on the scroll axis.
			"scaled image keeps its fill on the bounded cross axis",
			ui.ScrollView(ui.Vertical, ui.Image("/x.png").FramedAs(ui.ScaledToFill)),
			[]string{`class="ui-image ui-cell-fill-x"`},
			nil,
		},
		{
			"divider takes 10px along its length",
			ui.VStack(ui.Divider()).FixedSize(),
			[]string{"ui-divider-h", "width:10px"},
			[]string{"height:10px"},
		},
		{
			"vertical divider takes 10px along its length",
			ui.HStack(ui.Divider()).FixedSize(),
			[]string{"ui-divider-v", "height:10px"},
			[]string{"width:10px"},
		},
		{
			"decoration layer clears both axes",
			ui.Text("x").LayerOver(ui.Center, ui.Muted).FixedSize(),
			nil,
			[]string{"10px"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			for _, w := range tt.wants {
				if !strings.Contains(html, w) {
					t.Errorf("missing %q\n\n%s", w, html)
				}
			}
			for _, r := range tt.rejects {
				if strings.Contains(html, r) {
					t.Errorf("should not emit %q\n\n%s", r, html)
				}
			}
		})
	}
}

// TestPaddingAddsValues checks that one Padding call with several EdgeSpace
// arguments sums them per edge into a single wrapper.
func TestPaddingAddsValues(t *testing.T) {
	html := render(t, ui.Text("hi").Padding(ui.EdgeTop(8), ui.Edges(4)))
	for _, w := range []string{
		"padding-block:12px 4px",
		"padding-inline:4px",
	} {
		if !strings.Contains(html, w) {
			t.Errorf("summed padding missing %q:\n%s", w, html)
		}
	}
	if got := strings.Count(html, "ui-padding"); got != 1 {
		t.Errorf("ui-padding wrapper count = %d, want 1:\n%s", got, html)
	}
}

// TestTextRunsPreserveType exercises the type-erasure rule: rich
// text composes while still a TextView, and per-run vs whole-text styling land
// where intended.
func TestTextRunsPreserveType(t *testing.T) {
	v := ui.Text("Status: ").
		Bold().
		Concat(ui.Text("Draft").Italic()).
		TextForeground(ui.Muted)

	html := render(t, v)
	for _, w := range []string{"font-weight:600", "font-style:italic", "Status: ", "Draft"} {
		if !strings.Contains(html, w) {
			t.Errorf("rich text missing %q\n\n%s", w, html)
		}
	}
	// TextColor applies to the whole text: once, on the enclosing element,
	// where every run inherits it.
	if got := strings.Count(html, "var(--ui-color-muted)"); got != 1 {
		t.Errorf("muted color count = %d, want 1 (whole-text color)\n\n%s", got, html)
	}
}

// TestTextWholeTextRule pins the whole-text rule: a text modifier applied
// after Concat styles all runs, while a run styled before Concat keeps its
// own styling.
func TestTextWholeTextRule(t *testing.T) {
	html := render(t, ui.Text("a").Concat(ui.Text("b").Italic()).Bold())
	if !strings.Contains(html, `<div class="ui-text"><span style="font-weight:600">a<span`) {
		t.Errorf("whole-text Bold should land on a span enclosing every run:\n%s", html)
	}
	if !strings.Contains(html, `style="font-style:italic"`) {
		t.Errorf("pre-Concat Italic should stay on its own run:\n%s", html)
	}
}

// Guard against an accidental change to the keyed-row idiom shown in the doc.
func TestForKeyLikeIdiom(t *testing.T) {
	items := []Movie{{ID: 7}, {ID: 42}}
	v := ui.VStack(ui.For(items, movieKey, func(m Movie) ui.View {
		return ui.Text("#" + strconv.FormatUint(m.ID, 10))
	}))
	html := render(t, v)
	for _, w := range []string{"#7", "#42"} {
		if !strings.Contains(html, w) {
			t.Errorf("For output missing %q\n\n%s", w, html)
		}
	}
}

// TestAlignProjectsOntoCrossAxis checks that a stack alignment projects the
// component for the axis the stack actually crosses: inline for a VStack,
// block for an HStack.
func TestAlignProjectsOntoCrossAxis(t *testing.T) {
	h := render(t, ui.HStack(ui.Text("a")).Alignment(ui.TopTrailing))
	if !strings.Contains(h, "align-items:start") {
		t.Errorf("HStack TopTrailing should align to the top:\n%s", h)
	}
	v := render(t, ui.VStack(ui.Text("a")).Alignment(ui.TopTrailing))
	if !strings.Contains(v, "align-items:end") {
		t.Errorf("VStack TopTrailing should align to the trailing edge:\n%s", v)
	}

	f := render(t, ui.Text("a").Frame(ui.Width(100), ui.BottomTrailing))
	if !strings.Contains(f, "place-items:end end") {
		t.Errorf("frame Align should place the content in both axes:\n%s", f)
	}
}

// TestDividerAxisAware checks that a divider orients against its stack's axis:
// vertical inside an HStack, horizontal inside a VStack, stretching along the
// minor axis either way.
func TestDividerAxisAware(t *testing.T) {
	h := render(t, ui.HStack(ui.Text("a"), ui.Divider(), ui.Text("b")))
	if !strings.Contains(h, "ui-divider ui-divider-v ui-stretch") {
		t.Errorf("divider in HStack should be vertical and stretch:\n%s", h)
	}

	v := render(t, ui.VStack(ui.Text("a"), ui.Divider(), ui.Text("b")))
	if !strings.Contains(v, "ui-divider ui-divider-h ui-stretch") {
		t.Errorf("divider in VStack should be horizontal and stretch:\n%s", v)
	}
}

// TestForKeysItems checks that each For item carries its key directly on its
// own element, with no wrapper in between.
func TestForKeysItems(t *testing.T) {
	items := []Movie{{ID: 7, Title: "Seven"}, {ID: 42, Title: "Forty-Two"}}
	v := ui.VStack(ui.For(items,
		func(m Movie) string { return strconv.FormatUint(m.ID, 10) },
		func(m Movie) ui.View { return ui.Text(m.Title) },
	))
	html := render(t, v)
	for _, w := range []string{"ui-vstack", `domi-key="7"`, `domi-key="42"`, "Seven", "Forty-Two"} {
		if !strings.Contains(html, w) {
			t.Errorf("keyed For missing %q\n\n%s", w, html)
		}
	}
}

// TestForMultiElementItemPanics pins the single-element contract: a For item
// must render to exactly one element for the key to live on, and rendering
// an item that spreads across several panics.
func TestForMultiElementItemPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a multi-element For item did not panic")
		}
	}()
	items := []Movie{{ID: 7, Title: "Seven"}}
	render(t, ui.VStack(ui.For(items, movieKey, func(m Movie) ui.View {
		return ui.Group(ui.Text(m.Title), ui.Text(m.Title))
	})))
}

// TestForNilKeyUnkeyed pins the unkeyed mode: a nil key splices the items
// with no key stamps and no per-item element requirement, so an item may
// render to several elements.
func TestForNilKeyUnkeyed(t *testing.T) {
	items := []Movie{{ID: 7, Title: "Seven"}}
	html := render(t, ui.VStack(ui.For(items, nil, func(m Movie) ui.View {
		return ui.Group(ui.Text(m.Title), ui.Text(m.Title))
	})))
	if strings.Contains(html, "key=") {
		t.Errorf("nil-key For should render unkeyed:\n%s", html)
	}
	if got := strings.Count(html, "Seven"); got != 2 {
		t.Errorf("nil-key For item should keep both elements, found %d:\n%s", got, html)
	}
}

// TestImageNative checks the native framing mode's lowering — the img is
// the view's own box, no wrapper, rigid on both axes — and that it is
// the default mode.
func TestImageNative(t *testing.T) {
	html := render(t, ui.HStack(ui.Image("/x.png").Alt("pic")))
	if strings.Contains(html, "ui-image") {
		t.Errorf("native image should have no wrapper:\n%s", html)
	}
	if want := `<img alt="pic" class="ui-rigid" src="/x.png">`; !strings.Contains(html, want) {
		t.Errorf("native image missing %q:\n%s", want, html)
	}
}

// TestScrollView checks the requested axis selects the right overflow variant.
func TestScrollView(t *testing.T) {
	cases := map[ui.AxisSet]string{
		ui.Vertical:                 "ui-scroll ui-scroll-y",
		ui.Horizontal:               "ui-scroll ui-scroll-x",
		ui.Horizontal | ui.Vertical: "ui-scroll ui-scroll-xy",
		ui.AxisSet(0):               "ui-scroll ui-scroll-none",
	}
	for axis, want := range cases {
		html := render(t, ui.ScrollView(axis, ui.Text("content")))
		if !strings.Contains(html, want) {
			t.Errorf("ScrollView(%v) missing %q\n\n%s", axis, want, html)
		}
	}
}

// TestFrameBounds checks the bounds frame's lowering: bounds and alignment on
// its own box, fill requests relayed above a minimum but absorbed into
// the frame's own size at a definite maximum, and Auto bounds emitting
// nothing.
func TestFrameBounds(t *testing.T) {
	bounded := render(t, ui.Text("x").FrameBounds(ui.MinWidth(96), ui.MaxWidth(320), ui.MinHeight(24), ui.MaxHeight(48), ui.Leading))
	for _, w := range []string{
		"min-width:96px", "max-width:320px", "min-height:24px", "max-height:48px", "place-items:center start",
		// An explicit minimum zeroes the axis's intrinsic track so the
		// min-* declaration is the floor.
		"grid-template-columns:minmax(0,100%)", "grid-template-rows:minmax(0,100%)",
	} {
		if !strings.Contains(bounded, w) {
			t.Errorf("bounded frame missing %q\n\n%s", w, bounded)
		}
	}

	relay := render(t, ui.HStack(ui.Spacer()).FrameBounds(ui.MinWidth(96)))
	if !strings.Contains(relay, "ui-cell-fill-x") {
		t.Errorf("a min-bounded frame should relay its subview's fill:\n%s", relay)
	}

	// A fill request meeting a definite max is absorbed: the frame
	// claims the max through its own track and relays nothing. The
	// subview still fills the frame's cell; the frame itself must not
	// carry the fill upward.
	capped := render(t, ui.HStack(ui.Spacer()).FrameBounds(ui.MaxWidth(320)))
	for _, want := range []string{"grid-template-columns:minmax(0,320px)", "max-width:320px"} {
		if !strings.Contains(capped, want) {
			t.Errorf("a max-bounded frame should claim its max, missing %q:\n%s", want, capped)
		}
	}
	if strings.Contains(capped, "ui-frame ui-cell-fill-x") {
		t.Errorf("a max-bounded frame should absorb its subview's fill:\n%s", capped)
	}

	auto := render(t, ui.Text("x").FrameBounds(ui.MinWidth(ui.Auto{}), ui.MaxWidth(ui.Auto{})))
	for _, r := range []string{"min-width", "max-width", "grid-template"} {
		if strings.Contains(auto, r) {
			t.Errorf("Auto bound should emit nothing, got %q:\n%s", r, auto)
		}
	}

	// Conflicting bounds apply in order: the later bound adjusts the
	// earlier one to itself.
	minThenMax := render(t, ui.Text("x").FrameBounds(ui.MinWidth(100), ui.MaxWidth(50)))
	for _, w := range []string{"min-width:50px", "max-width:50px"} {
		if !strings.Contains(minThenMax, w) {
			t.Errorf("later max should adjust the earlier min, missing %q:\n%s", w, minThenMax)
		}
	}
	maxThenMin := render(t, ui.Text("x").FrameBounds(ui.MaxWidth(50), ui.MinWidth(100)))
	for _, w := range []string{"min-width:100px", "max-width:100px"} {
		if !strings.Contains(maxThenMin, w) {
			t.Errorf("later min should adjust the earlier max, missing %q:\n%s", w, maxThenMin)
		}
	}

	// The ideal and the bounds also apply in order, each adjusting an
	// earlier conflicting slot to itself.
	for _, tt := range []struct {
		name  string
		v     ui.View
		wants []string
	}{
		{
			"later ideal raises an earlier max",
			ui.Text("x").FrameBounds(ui.MaxWidth(50), ui.IdealWidth(100)).FixedSize(),
			[]string{"width:100px", "max-width:100px"},
		},
		{
			"later max lowers an earlier ideal",
			ui.Text("x").FrameBounds(ui.IdealWidth(100), ui.MaxWidth(50)).FixedSize(),
			[]string{"width:50px", "max-width:50px"},
		},
		{
			"later ideal lowers an earlier min",
			ui.Text("x").FrameBounds(ui.MinWidth(200), ui.IdealWidth(100)).FixedSize(),
			[]string{"width:100px", "min-width:100px"},
		},
		{
			"later min raises an earlier ideal",
			ui.Text("x").FrameBounds(ui.IdealWidth(100), ui.MinWidth(200)).FixedSize(),
			[]string{"width:200px", "min-width:200px"},
		},
	} {
		html := render(t, tt.v)
		for _, w := range tt.wants {
			if !strings.Contains(html, w) {
				t.Errorf("%s: missing %q:\n%s", tt.name, w, html)
			}
		}
	}

	fill := render(t, ui.HStack(ui.Spacer()))
	if !strings.Contains(fill, "ui-cell-fill-x") {
		t.Errorf("a fill request should lower to a fill class:\n%s", fill)
	}
	if strings.Contains(fill, "max-width") {
		t.Errorf("a fill request should not emit a size bound:\n%s", fill)
	}
}

// TestFrameRigid pins definite-axis rigidity: a frame whose size along the
// enclosing flex major axis is determined — directly or through an
// anchored ratio — opts out of flex shrink, while auto axes and grid
// containers need no opt-out.
func TestFrameRigid(t *testing.T) {
	for _, tt := range []struct {
		name  string
		v     ui.View
		rigid bool
	}{
		{"definite width in a row", ui.HStack(ui.Text("x").Frame(ui.Width(200))), true},
		{"definite height in a column", ui.VStack(ui.Text("x").Frame(ui.Height(50))), true},
		{"auto width in a row", ui.HStack(ui.Text("x").Frame(ui.Height(50))), false},
		{"definite width in a grid cell", ui.ZStack(ui.Text("x").Frame(ui.Width(200))), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			if got := strings.Contains(html, "ui-rigid"); got != tt.rigid {
				t.Errorf("ui-rigid = %v, want %v:\n%s", got, tt.rigid, html)
			}
		})
	}
}

// TestFrameOptionOverride pins the frame's option resolution: options apply
// in order, a later option for the same setting replaces an earlier one, and
// Auto restores the default.
func TestFrameOptionOverride(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       ui.View
		wants   []string
		rejects []string
	}{
		{
			"later width overrides an earlier one",
			ui.Text("x").Frame(ui.Width(50), ui.Width(60)),
			[]string{"width:60px"},
			[]string{"width:50px"},
		},
		{
			"axes are independent",
			ui.Text("x").Frame(ui.Width(50), ui.Height(60)),
			[]string{"width:50px", "height:60px"},
			nil,
		},
		{
			"Auto is the default and emits nothing",
			ui.Text("x").Frame(ui.Width(50), ui.Width(ui.Auto{})),
			nil,
			[]string{"width", "auto"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			for _, w := range tt.wants {
				if !strings.Contains(html, w) {
					t.Errorf("frame missing %q\n\n%s", w, html)
				}
			}
			for _, r := range tt.rejects {
				if strings.Contains(html, r) {
					t.Errorf("frame should not emit %q\n\n%s", r, html)
				}
			}
		})
	}
}

// TestGapDoesNotLeak pins the tokens-vs-parameters rule: gap is a per-box
// parameter lowered to the real CSS property, so a gap set on one stack must
// appear on that stack only, never on a descendant that didn't ask for one.
// (The custom-property lowering this replaced inherited into nested stacks.)
func TestGapDoesNotLeak(t *testing.T) {
	html := render(t, ui.VStack(
		ui.VStack(ui.Text("a"), ui.Text("b")),
	).Gap(16))
	if got := strings.Count(html, "gap:"); got != 1 {
		t.Errorf("gap declaration count = %d, want 1 (outer stack only)\n\n%s", got, html)
	}
}

// TestEmptyControlFlow checks that Empty and a false branch contribute no
// markup, even with modifiers applied.
func TestEmptyControlFlow(t *testing.T) {
	html := render(t, ui.VStack(
		ui.Text("shown"),
		ui.Empty(),
		ui.Empty().Class("hidden-class").Padding(ui.Edges(8)),
		ui.If(false, ui.Text("hidden-if")),
		ui.When(false, func() ui.View { return ui.Text("hidden-when") }),
	))
	if strings.Contains(html, "hidden") {
		t.Errorf("false control-flow branches should render nothing:\n%s", html)
	}
	if !strings.Contains(html, "shown") {
		t.Errorf("control-flow test lost its visible content:\n%s", html)
	}
}

// TestModifierOrder checks that decoration order is preserved: padding-then-
// underlay wraps the padded box, the reverse pads the decorated box.
func TestModifierOrder(t *testing.T) {
	paddedThenBg := render(t, ui.Text("x").Padding(ui.Edges(8)).LayerUnder(ui.Center, ui.Color("#eee")))
	bgThenPadded := render(t, ui.Text("x").LayerUnder(ui.Center, ui.Color("#eee")).Padding(ui.Edges(8)))
	if paddedThenBg == bgThenPadded {
		t.Errorf("modifier order should change the lowering, but both rendered identically:\n%s", paddedThenBg)
	}
}

// TestRenderRootWrapsGroup pins the root's multiview rule: the viewport
// frames a single view, so a view of more than one node is wrapped in a
// VStack rather than distributed over.
func TestRenderRootWrapsGroup(t *testing.T) {
	group := render(t, ui.Group(ui.Text("a"), ui.Text("b")))
	if !strings.Contains(group, "ui-vstack") {
		t.Errorf("a root Group should be wrapped in a VStack:\n%s", group)
	}
	single := render(t, ui.Text("a"))
	if strings.Contains(single, "ui-vstack") {
		t.Errorf("a single root view should not be wrapped:\n%s", single)
	}
}
