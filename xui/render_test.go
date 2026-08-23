package ui_test

import (
	"regexp"
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
			ui.Button(Msg{EditProfile: true}, ui.Text("Edit")).Role(ui.RolePrimary),
		).Gap(12).Alignment(ui.Center),
	).
		Padding(ui.Edges(16)).
		Underlay(ui.Center, ui.CSSColor("#fff")).
		Overlay(ui.TopTrailing, ui.Badge("Pro"))
}

func moviePage(movies []Movie) ui.View {
	return ui.VStack(
		ui.HStack(
			ui.Text("Movies").Font(ui.Title),
			ui.Spacer(),
			ui.Button(Msg{NewMovie: true}, ui.Text("New")).Role(ui.RolePrimary),
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
		ui.Button(Msg{Watched: movie.ID}, ui.Text("Watched")),
	).Gap(12).Padding(ui.Edges(12))
}

func render(t *testing.T, v ui.View) string {
	t.Helper()
	var sb strings.Builder
	if err := domi.RenderTo(&sb, new(ui.Renderer).Render(v)); err != nil {
		t.Fatalf("render: %v", err)
	}
	return sb.String()
}

// classRule finds an element matching pattern.
// It returns the declarations for the generated class captured by the pattern.
func classRule(t *testing.T, html, pattern string) string {
	t.Helper()
	m := regexp.MustCompile(pattern).FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no element matching %q in:\n%s", pattern, html)
	}
	// The body can nest two block levels deep: a media block holding
	// pseudo-class blocks.
	r := regexp.MustCompile(regexp.QuoteMeta("."+m[1]) + `\{((?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*)\}`).FindStringSubmatch(html)
	if r == nil {
		t.Fatalf("no rule for class %s in:\n%s", m[1], html)
	}
	return r[1]
}

func TestAccountCard(t *testing.T) {
	html := render(t, accountCard(User{
		Name:     "Ada Lovelace",
		Email:    "ada@example.com",
		PhotoURL: "/ada.jpg",
	}))

	wants := []string{
		`<ui-root>`,         // root
		`<ui-card `,         // Card: an HStack named by its tag
		`flex-grow:1`,       // the Spacer's fill stretches the row across the card
		`border-radius:50%`, // BorderShape applied to the image frame
		`<ui-frame`,         // Size(48) introduces a frame wrapper
		`width:48px`,        // ...with the resolved size
		`<ui-spacer `,
		`<button `,
		`<ui-layer `, // Underlay + Overlay decoration layers
		`<ui-underlay `,
		`<ui-overlay `,
		`place-items:start end`, // the Overlay's alignment
		`border-radius:9999px`,  // the Badge's pill
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
		`align-self:stretch`,
		// The outer VStack inherits that horizontal fill; at the root, a
		// grid, it lowers to a cell stretch.
		`justify-self:stretch`,
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
	if got := strings.Count(html, "<ui-spacer "); got != 3 {
		t.Errorf("ui-spacer count = %d, want 3\n\n%s", got, html)
	}
}

// TestTagNamesElement checks that Tag names the element, replacing the
// box type's own tag — directly on a stack, and on a frame, where the
// named element still carries its fill (it grows along the enclosing
// row's main axis) and the enclosing stack keeps propagating the request
// toward a definite ancestor.
func TestTagNamesElement(t *testing.T) {
	if html := render(t, ui.VStack(ui.Text("a")).Tag("ul")); !strings.Contains(html, `<ul class="`) {
		t.Errorf("Tag should rename the stack's own element:\n%s", html)
	}

	html := render(t, ui.HStack(
		ui.HStack(ui.Text("a"), ui.Spacer()).Frame().Tag("nav"),
		ui.Text("b"),
	))
	for _, tt := range []struct{ pattern, want string }{
		{`<nav class="(ui-\w+)"`, "flex-grow:1"},                // the tagged frame carries the fill
		{`<ui-hstack class="(ui-\w+)"`, "justify-self:stretch"}, // ...and the root stack keeps it
		{`<ui-spacer class="(ui-\w+)"`, "flex-grow:1"},          // the inner row distributes slack
	} {
		if got := classRule(t, html, tt.pattern); !strings.Contains(got, tt.want) {
			t.Errorf("tagged-frame fill chain: %s rule = %q, want %q:\n%s", tt.pattern, got, tt.want, html)
		}
	}
}

// TestButtonAction pins the lowering of each kind of button action:
// a message sends from a button element, a URL navigates from an
// anchor, and disabling either one removes its means of activation.
func TestButtonAction(t *testing.T) {
	for _, tt := range []struct {
		name         string
		v            ui.View
		want, absent []string
	}{
		{
			"send",
			ui.Button(Msg{}, ui.Text("x")),
			[]string{`<button `, ` type="button"`},
			[]string{` disabled`},
		},
		{
			"send disabled",
			ui.Button(Msg{}, ui.Text("x")).Disabled(true),
			[]string{`<button `, ` disabled`},
			nil,
		},
		{
			"navigate",
			ui.Button("/movies", ui.Text("x")),
			[]string{`<a `, ` href="/movies"`},
			[]string{`aria-disabled`},
		},
		{
			"navigate disabled",
			ui.Button("/movies", ui.Text("x")).Disabled(true),
			[]string{`<a `, ` aria-disabled="true"`},
			[]string{` href=`},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			for _, s := range tt.want {
				if !strings.Contains(html, s) {
					t.Errorf("missing %q:\n%s", s, html)
				}
			}
			for _, s := range tt.absent {
				if strings.Contains(html, s) {
					t.Errorf("unexpected %q:\n%s", s, html)
				}
			}
		})
	}
}

// TestTagInnermostWins pins Tag's contract: the innermost tag names the
// element, so a view's intrinsic tag beats a Tag modifier, and the Tag
// nearest the view beats a repetition.
func TestTagInnermostWins(t *testing.T) {
	for _, tt := range []struct {
		name string
		v    ui.View
		want string
	}{
		{"intrinsic tag", ui.Button(Msg{}, ui.Text("x")).Tag("figure"), "<button"},
		{"double Tag", ui.VStack().Tag("ul").Tag("ol"), "<ul"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			if !strings.Contains(html, tt.want) {
				t.Errorf("missing %q:\n%s", tt.want, html)
			}
		})
	}
}

// TestHTMLHost pins the [ui.HTML] host contract: the node renders inside
// a host element the view manages, and element modifiers land on that
// host rather than on the node.
func TestHTMLHost(t *testing.T) {
	html := render(t, ui.HTML(domi.Text("raw")))
	if want := regexp.MustCompile(`<ui-html class="ui-\w+">raw</ui-html>`); !want.MatchString(html) {
		t.Errorf("missing %q:\n%s", want, html)
	}

	mod := render(t, ui.HTML(domi.Text("raw")).BorderShape(ui.Ellipse).Class("x").Tag("section"))
	for _, w := range []string{"<section", `class="x `, "border-radius:50%", ">raw<"} {
		if !strings.Contains(mod, w) {
			t.Errorf("host modifiers missing %q:\n%s", w, mod)
		}
	}
}

// TestHTMLFill pins the view's fill personality: like a Color, it
// requests fill on both axes — lowered per the enclosing container and
// propagated through an enclosing stack — and FixedSize opts into
// content sizing.
func TestHTMLFill(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       ui.View
		wants   []string
		rejects []string
	}{
		{
			"both axes at the root grid",
			ui.HTML(domi.Text("raw")),
			[]string{"align-self:stretch", "justify-self:stretch"},
			nil,
		},
		{
			"a row grows it, stretches it, and inherits the fill",
			ui.HStack(ui.HTML(domi.Text("raw"))),
			[]string{"flex-grow:1", "align-self:stretch", "justify-self:stretch"},
			nil,
		},
		{
			"FixedSize clears the fill axes",
			ui.HTML(domi.Text("raw")).FixedSize(),
			[]string{`<ui-html class="ui-fixed-size `},
			[]string{"align-self:stretch", "justify-self:stretch"},
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

// TestHTMLWrappers pins that wrapping modifiers enclose the host: the
// wrapper element appears outside it, carrying the paint, and the node
// stays untouched inside.
func TestHTMLWrappers(t *testing.T) {
	html := render(t, ui.HTML(domi.Text("raw")).Padding(ui.Edges(4)).Background(ui.CSSColor("red")))
	if got := classRule(t, html, `<ui-padding class="(ui-\w+)"`); got != "align-self:stretch;background-color:red;display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-self:stretch;padding:4px;place-items:center" {
		t.Errorf("padding wrapper should carry the paint, got %q:\n%s", got, html)
	}
	if got := classRule(t, html, `<ui-html class="(ui-\w+)"`); got != "align-self:stretch;display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-self:stretch;place-items:center" {
		t.Errorf("host should stay untouched inside, got %q:\n%s", got, html)
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
// solid fill of itself requesting fill on both axes, and generic
// modifiers reach the fill's box.
func TestColorAsView(t *testing.T) {
	html := render(t, ui.Muted)
	if got := classRule(t, html, `<ui-color class="(ui-\w+)"`); got != "align-self:stretch;background-color:var(--ui-color-muted);justify-self:stretch" {
		t.Errorf("color view should paint its own box and fill both axes, got %q:\n%s", got, html)
	}
	if mod := render(t, ui.CSSColor("#eee").Opacity(0.5)); !strings.Contains(mod, "opacity:0.5") {
		t.Errorf("modifier on a color view should reach its box:\n%s", mod)
	}

	// Background layers behind the color on the color's own element,
	// visible where c is translucent — ordinary painting order, not a
	// decoration layer, and the Modify spelling is the same lowering.
	bg := render(t, ui.CSSColor("#0008").Background(ui.CSSColor("#fff")))
	if got := classRule(t, bg, `<ui-color class="(ui-\w+)"`); got != "align-self:stretch;background-color:#fff;background-image:linear-gradient(#0008,#0008);justify-self:stretch" {
		t.Errorf("Background should layer under the color, got %q:\n%s", got, bg)
	}
	if strings.Contains(bg, "ui-underlay") {
		t.Errorf("Background on a color should merge, not add a layer:\n%s", bg)
	}
	if mod := render(t, ui.CSSColor("#0008").Modify(ui.Background(ui.CSSColor("#fff")))); mod != bg {
		t.Errorf("Modify(Background) diverged from the Background method:\n%s", mod)
	}
	// Underlay layers content behind the color.
	under := render(t, ui.CSSColor("#0008").Underlay(ui.Center, ui.Text("behind")))
	for _, w := range []string{`<ui-underlay `, "behind"} {
		if !strings.Contains(under, w) {
			t.Errorf("Underlay behind a color missing %q:\n%s", w, under)
		}
	}
	// Modifiers with no possible effect on a color are no-ops.
	if noop := render(t, ui.Muted.Foreground(ui.CSSColor("#fff")).Font(ui.Title)); noop != html {
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

// TestIdealSize pins the unbounded-space contract: FixedSize marks its
// subtree's available space as unbounded; the space-adaptive views
// answer each unbounded axis with a deliberate ideal (Color 10px,
// ScrollView 100px, Divider 10px along its length), contributed as a
// minimum where the box also fills the axis and taken as its size
// where it does not; fills survive unbounded space but are stripped at
// fill boundaries — scroll content along a scroll axis, a FixedSize
// subtree's outermost box; and unbounded clears per axis wherever a
// box makes real space available again — a frame's definite axes, a
// decoration layer, a scroll viewport.
func TestIdealSize(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       ui.View
		wants   []string
		rejects []string
	}{
		{
			// The color's own box is the FixedSize boundary: its
			// fills are stripped, so it takes its ideal as its size.
			"direct FixedSize",
			ui.Muted.FixedSize(),
			[]string{"width:10px", "height:10px"},
			[]string{"min-width", "min-height"},
		},
		{
			// The stack is the boundary; the color inside keeps its
			// fills and contributes its ideal as a minimum.
			"FixedSize on an ancestor",
			ui.VStack(ui.Muted).FixedSize(),
			[]string{"min-width:10px", "min-height:10px"},
			nil,
		},
		{
			"definite frame axis clears its axis only",
			ui.VStack(ui.Muted).Frame(ui.Width(200)).FixedSize(),
			[]string{"min-height:10px"},
			[]string{"min-width"},
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
			// The color's fill is stripped on the scroll axis only,
			// so its ideal is its size there, and it keeps filling
			// the bounded cross axis.
			"scroll axis is unbounded without FixedSize",
			ui.ScrollView(ui.Vertical, ui.Muted),
			[]string{"height:10px;justify-self:stretch"},
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
			[]string{"width:100px", "height:100px", "overflow-x:clip;overflow-y:clip"},
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
			// A scaling mode meets an imposed box; with no box to
			// meet, the img's intrinsic geometry answers instead.
			"scaled image drops its fills on unbounded axes",
			ui.Image("/x.png").FramedAs(ui.ScaledToFill).FixedSize(),
			[]string{"object-fit:cover"},
			[]string{"justify-self", "align-self"},
		},
		{
			// The viewport itself stays greedy on both axes; only
			// the image's own fill is dropped on the scroll axis.
			"scaled image keeps its fill on the bounded cross axis",
			ui.ScrollView(ui.Vertical, ui.Image("/x.png").FramedAs(ui.ScaledToFill)),
			[]string{"justify-self:stretch;min-height:0;min-width:0;object-fit:cover"},
			[]string{"align-self:stretch;justify-self:stretch;min-height"},
		},
		{
			"divider contributes 10px along its length",
			ui.VStack(ui.Divider()).FixedSize(),
			[]string{"height:1px", "min-width:10px", "align-self:stretch"},
			[]string{"min-height"},
		},
		{
			"vertical divider contributes 10px along its length",
			ui.HStack(ui.Divider()).FixedSize(),
			[]string{"width:1px", "min-height:10px", "align-self:stretch"},
			[]string{"min-width"},
		},
		{
			"decoration layer clears both axes",
			ui.Text("x").Overlay(ui.Center, ui.Muted).FixedSize(),
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
	if got := strings.Count(html, "<ui-padding "); got != 1 {
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

// TestFontSpecifiesWholeType pins that a font is a complete type
// setting: inside a Title subtree, each slot's size, weight, and
// line height are its own, while a box with no font set inherits
// and emits nothing.
func TestFontSpecifiesWholeType(t *testing.T) {
	for _, tt := range []struct {
		f     ui.FontSize
		wants []string
	}{
		{ui.Body, []string{"font-size:1rem", "font-weight:400", "line-height:1.4"}},
		{ui.Caption, []string{"font-size:0.75rem", "font-weight:400", "line-height:1.3"}},
		{ui.Headline, []string{"font-size:1.125rem", "font-weight:600", "line-height:1.4"}},
		{ui.LargeTitle, []string{"font-size:2rem", "font-weight:700", "line-height:1.15"}},
	} {
		html := render(t, ui.VStack(ui.Text("x").Font(tt.f)).Font(ui.Title))
		for _, w := range tt.wants {
			if !strings.Contains(html, w) {
				t.Errorf("Font(%q) should emit %q:\n%s", tt.f, w, html)
			}
		}
	}
	if html := render(t, ui.Text("x")); strings.Contains(html, "font-size") {
		t.Errorf("an unset font should emit nothing:\n%s", html)
	}
}

// TestTextWholeTextRule pins the whole-text rule: a text modifier applied
// after Concat styles all runs, while a run styled before Concat keeps its
// own styling.
func TestTextWholeTextRule(t *testing.T) {
	html := render(t, ui.Text("a").Concat(ui.Text("b").Italic()).Bold())
	if got := classRule(t, html, `<ui-text[^>]*><span class="(ui-\w+)"`); got != "font-weight:600" {
		t.Errorf("whole-text Bold should land on a span enclosing every run, got %q:\n%s", got, html)
	}
	if got := classRule(t, html, `>a<span class="(ui-\w+)"`); got != "font-style:italic" {
		t.Errorf("pre-Concat Italic should stay on its own run, got %q:\n%s", got, html)
	}
}

// TestLink pins the lowering of each kind of link action: a URL
// navigates from an anchor, a message sends from a button element,
// and either one is an inline run that carries the pending style.
func TestLink(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       ui.View
		pattern string
		want    []string
	}{
		{
			"navigate",
			ui.Text("see ").Concat(ui.Link("/docs", ui.Text("docs"))),
			`<a class="(ui-\w+)" href="/docs">docs</a>`,
			[]string{"cursor:pointer", "color:var(--ui-color-accent)"},
		},
		{
			"send",
			ui.Text("or ").Concat(ui.Link(Msg{}, ui.Text("retry"))),
			`<button class="(ui-\w+)" domi-msg-click="[^"]*" type="button">retry</button>`,
			[]string{"cursor:pointer", "color:var(--ui-color-accent)"},
		},
		{
			"outer style",
			ui.Link("/docs", ui.Text("docs")).Bold(),
			`<a class="(ui-\w+)" href="/docs">docs</a>`,
			[]string{"color:var(--ui-color-accent)", "font-weight:600"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			got := classRule(t, html, tt.pattern)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("rule = %q, missing %q:\n%s", got, w, html)
				}
			}
		})
	}
}

// TestLinkDisabled pins that a disabled link of either kind loses its
// means of activation and is faded, while an enabled one is not.
func TestLinkDisabled(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       ui.View
		pattern string
		want    []string
		absent  []string
	}{
		{
			"navigate",
			ui.Text("a").Concat(ui.Link("/docs", ui.Text("x")).Disabled(true)),
			`<a aria-disabled="true" class="(ui-\w+)" role="link">x</a>`,
			[]string{"cursor:default", "opacity:0.5"},
			[]string{` href=`},
		},
		{
			"send",
			ui.Text("a").Concat(ui.Link(Msg{}, ui.Text("x")).Disabled(true)),
			`<button class="(ui-\w+)" disabled domi-msg-click="[^"]*" type="button">x</button>`,
			[]string{"cursor:default", "opacity:0.5"},
			nil,
		},
		{
			"enabled",
			ui.Text("a").Concat(ui.Link("/docs", ui.Text("x")).Disabled(false)),
			`<a class="(ui-\w+)" href="/docs">x</a>`,
			[]string{"cursor:pointer"},
			[]string{"opacity", "aria-disabled"},
		},
		{
			"block",
			ui.Link("/docs", ui.Text("x")).Disabled(true),
			`<a aria-disabled="true" class="(ui-\w+)" role="link"><span`,
			[]string{"cursor:default", "opacity:0.5", "display:block"},
			[]string{` href=`, "ui-text"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			got := classRule(t, html, tt.pattern)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("rule = %q, missing %q:\n%s", got, w, html)
				}
			}
			for _, s := range tt.absent {
				if strings.Contains(html, s) {
					t.Errorf("unexpected %q:\n%s", s, html)
				}
			}
		})
	}
}

// TestLinkRequirePageLoad pins RequirePageLoad: a navigating link
// gains the domi bypass annotation, and a sending or disabled one
// is unaffected.
func TestLinkRequirePageLoad(t *testing.T) {
	for name, tc := range map[string]struct {
		v    ui.View
		want bool
	}{
		"navigate": {ui.Link("/docs", ui.Text("x")).RequirePageLoad(), true},
		"run":      {ui.Text("a").Concat(ui.Link("/docs", ui.Text("x")).RequirePageLoad()), true},
		"send":     {ui.Link(Msg{}, ui.Text("x")).RequirePageLoad(), false},
		"disabled": {ui.Link("/docs", ui.Text("x")).RequirePageLoad().Disabled(true), false},
		"without":  {ui.Link("/docs", ui.Text("x")), false},
	} {
		t.Run(name, func(t *testing.T) {
			html := render(t, tc.v)
			if got := strings.Contains(html, "domi-bypass"); got != tc.want {
				t.Errorf("bypass annotation = %v, want %v:\n%s", got, tc.want, html)
			}
		})
	}
}

// TestLinkColor pins the link color as a color set at the link:
// a color set inside the label wins over it,
// and it wins over a color set outside the link.
func TestLinkColor(t *testing.T) {
	inner := render(t, ui.Text("a").Concat(ui.Link("/", ui.Text("x").TextForeground(ui.Muted))))
	if got := classRule(t, inner, `<a class="(ui-\w+)" href="/"`); !strings.Contains(got, "color:var(--ui-color-accent)") {
		t.Errorf("link rule = %q, want the accent color:\n%s", got, inner)
	}
	if got := classRule(t, inner, `<span class="(ui-\w+)"`); got != "color:var(--ui-color-muted)" {
		t.Errorf("label rule = %q, want its own color inside the link:\n%s", got, inner)
	}

	outer := render(t, ui.Text("a").Concat(ui.Link("/", ui.Text("x")).TextForeground(ui.Muted)))
	if got := classRule(t, outer, `<a class="(ui-\w+)" href="/"`); !strings.Contains(got, "color:var(--ui-color-accent)") || strings.Contains(outer, "muted") {
		t.Errorf("link rule = %q, want the accent color to replace the outer color:\n%s", got, outer)
	}

	// As a box, the link's label carries the color.
	block := render(t, ui.Link("/", ui.Text("x")))
	if got := classRule(t, block, `<a class="(ui-\w+)" href="/"><span`); strings.Contains(got, "color") {
		t.Errorf("block link rule = %q, want no color on the box:\n%s", got, block)
	}
	if got := classRule(t, block, `<span class="(ui-\w+)">x`); got != "color:var(--ui-color-accent)" {
		t.Errorf("block label rule = %q, want the accent color:\n%s", got, block)
	}
}

// TestLineLimit pins the lowering and reach of LineLimit: it clamps
// the text it is applied to, reaches every text below the modifier,
// and the limit nearest a text wins.
func TestLineLimit(t *testing.T) {
	direct := render(t, ui.Text("x").LineLimit(2))
	if got := classRule(t, direct, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "-webkit-line-clamp:2") {
		t.Errorf("rule = %q, want -webkit-line-clamp:2:\n%s", got, direct)
	}

	subtree := render(t, ui.VStack(ui.Text("a"), ui.Text("b").LineLimit(1)).LineLimit(3))
	for want, n := range map[string]int{"-webkit-line-clamp:3": 1, "-webkit-line-clamp:1": 1} {
		if got := strings.Count(subtree, want); got != n {
			t.Errorf("%s count = %d, want %d:\n%s", want, got, n, subtree)
		}
	}

	// The limit survives boundaries that establish a new layout
	// context for their contents.
	for name, v := range map[string]ui.View{
		"scroll":  ui.ScrollView(ui.Vertical, ui.Text("x")).LineLimit(2),
		"overlay": ui.VStack().Overlay(ui.Center, ui.Text("x")).LineLimit(2),
	} {
		if got := render(t, v); !strings.Contains(got, "-webkit-line-clamp:2") {
			t.Errorf("%s content should clamp:\n%s", name, got)
		}
	}
}

func TestLineLimitClamps(t *testing.T) {
	html := render(t, ui.Text("x").LineLimit(0))
	if got := classRule(t, html, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "-webkit-line-clamp:1") {
		t.Errorf("rule = %q, want a limit below 1 raised to 1:\n%s", got, html)
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

	// An alignment whose relevant axis projects to center emits the
	// center it projects to.
	c := render(t, ui.VStack(ui.Text("a")).Alignment(ui.Top))
	if !strings.Contains(c, "align-items:center") {
		t.Errorf("VStack Top projects to center on the cross axis:\n%s", c)
	}
}

// TestDividerAxisAware checks that a divider orients against its stack's axis:
// vertical inside an HStack, horizontal inside a VStack, stretching along the
// minor axis either way.
func TestDividerAxisAware(t *testing.T) {
	h := render(t, ui.HStack(ui.Text("a"), ui.Divider(), ui.Text("b")))
	if got := classRule(t, h, `<ui-divider class="(ui-\w+)"`); !strings.Contains(got, "align-self:stretch") || !strings.Contains(got, "width:1px") {
		t.Errorf("divider in HStack should be vertical and stretch, got %q:\n%s", got, h)
	}

	v := render(t, ui.VStack(ui.Text("a"), ui.Divider(), ui.Text("b")))
	if got := classRule(t, v, `<ui-divider class="(ui-\w+)"`); !strings.Contains(got, "align-self:stretch") || !strings.Contains(got, "height:1px") {
		t.Errorf("divider in VStack should be horizontal and stretch, got %q:\n%s", got, v)
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
	if strings.Contains(html, "object-fit") {
		t.Errorf("native image should have no framing mode:\n%s", html)
	}
	if want := `<img alt="pic" class="ui-\w+" src="/x.png">`; !regexp.MustCompile(want).MatchString(html) {
		t.Errorf("native image missing %q:\n%s", want, html)
	}
}

// TestScrollView checks the requested axis selects the right overflow variant.
func TestScrollView(t *testing.T) {
	cases := map[ui.AxisSet]string{
		ui.Vertical:                 "overflow-x:hidden;overflow-y:auto",
		ui.Horizontal:               "overflow-x:auto;overflow-y:hidden",
		ui.Horizontal | ui.Vertical: "overflow-x:auto;overflow-y:auto",
		ui.AxisSet(0):               "overflow-x:clip;overflow-y:clip",
	}
	for axis, want := range cases {
		html := render(t, ui.ScrollView(axis, ui.Text("content")))
		if !strings.Contains(html, want) {
			t.Errorf("ScrollView(%v) missing %q\n\n%s", axis, want, html)
		}
	}
}

// TestFrameBounds checks the bounds frame's lowering: bounds and alignment on
// its own box, fill requests relayed above a minimum, and Auto bounds
// emitting no sizing declarations.
func TestFrameBounds(t *testing.T) {
	bounded := render(t, ui.Text("x").FrameBounds(ui.MinWidth(96), ui.MinHeight(24), ui.Leading))
	for _, w := range []string{
		"min-width:96px", "min-height:24px", "place-items:center start",
		// An explicit minimum zeroes the axis's intrinsic track so the
		// min-* declaration is the floor.
		"grid-template-columns:minmax(0, 100%)", "grid-template-rows:minmax(0, 100%)",
	} {
		if !strings.Contains(bounded, w) {
			t.Errorf("bounded frame missing %q\n\n%s", w, bounded)
		}
	}

	relay := render(t, ui.HStack(ui.Spacer()).FrameBounds(ui.MinWidth(96)))
	if !strings.Contains(relay, "justify-self:stretch") {
		t.Errorf("a min-bounded frame should relay its subview's fill:\n%s", relay)
	}

	auto := render(t, ui.Text("x").FrameBounds(ui.MinWidth(ui.Auto{}), ui.IdealWidth(ui.Auto{})))
	for _, r := range []string{"min-width", "width", "minmax"} {
		if strings.Contains(auto, r) {
			t.Errorf("Auto bound should emit nothing, got %q:\n%s", r, auto)
		}
	}

	// The ideal and the bounds apply in order, each adjusting an
	// earlier conflicting slot to itself.
	for _, tt := range []struct {
		name  string
		v     ui.View
		wants []string
	}{
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
	if !strings.Contains(fill, "justify-self:stretch") {
		t.Errorf("a fill request should lower to a fill declaration:\n%s", fill)
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
			if got := strings.Contains(html, "flex-shrink:0"); got != tt.rigid {
				t.Errorf("flex-shrink:0 = %v, want %v:\n%s", got, tt.rigid, html)
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
	if got := strings.Count(html, "gap:16px"); got != 1 {
		t.Errorf("gap:16px declaration count = %d, want 1 (outer stack only)\n\n%s", got, html)
	}
	if got := strings.Count(html, "gap:8px"); got != 1 {
		t.Errorf("gap:8px declaration count = %d, want 1 (the inner stack's own default)\n\n%s", got, html)
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
	paddedThenBg := render(t, ui.Text("x").Padding(ui.Edges(8)).Underlay(ui.Center, ui.CSSColor("#eee")))
	bgThenPadded := render(t, ui.Text("x").Underlay(ui.Center, ui.CSSColor("#eee")).Padding(ui.Edges(8)))
	if paddedThenBg == bgThenPadded {
		t.Errorf("modifier order should change the lowering, but both rendered identically:\n%s", paddedThenBg)
	}
}

// TestStateModifiers pins the state lowering: each state set present
// declares its differing properties under the matching pseudo-classes,
// with hover variants gated to devices that can hover.
func TestStateModifiers(t *testing.T) {
	hovered := render(t, ui.Text("x").WhileHovered(ui.Foreground(ui.CSSColor("#00f"))))
	if got := classRule(t, hovered, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "@media (hover: hover){&:hover{color:#00f}}") {
		t.Errorf("Hovered rule = %q, want a hover-gated color variant:\n%s", got, hovered)
	}
	focused := render(t, ui.Text("x").WhileFocused(ui.Foreground(ui.CSSColor("#00f"))))
	if got := classRule(t, focused, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "&:focus-visible{color:#00f}") {
		t.Errorf("Focused rule = %q, want a focus variant:\n%s", got, focused)
	}
	pressed := render(t, ui.Text("x").WhilePressed(ui.Font(ui.Title)))
	if got := classRule(t, pressed, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "&:active{font-size:1.5rem;font-weight:700;line-height:1.2}") {
		t.Errorf("Pressed rule = %q, want an active font variant:\n%s", got, pressed)
	}
	// A combination applies only while every given state is active,
	// regardless of the order or repetition of the states.
	both := render(t, ui.Text("x").Modify(ui.Background(ui.CSSColor("#eee")), ui.Pressed, ui.Hovered, ui.Pressed))
	if got := classRule(t, both, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "@media (hover: hover){&:hover:active{background-color:#eee}}") {
		t.Errorf("combined rule = %q, want a hover+active variant:\n%s", got, both)
	}
	blue := ui.Foreground(ui.CSSColor("#00f"))
	for _, tc := range []struct {
		v    ui.View
		want string
	}{
		{ui.Text("x").WhileDisabled(blue), `&:is(:disabled, [aria-disabled="true"]){color:#00f}`},
		{ui.Text("x").WhileChecked(blue), "&:checked{color:#00f}"},
		{ui.Text("x").WhileInvalid(blue), "&:user-invalid{color:#00f}"},
		{ui.Text("x").WhilePlaceholder(blue), "&:placeholder-shown{color:#00f}"},
		{ui.Text("x").Modify(blue, ui.Disabled, ui.Checked), `&:is(:disabled, [aria-disabled="true"]):checked{color:#00f}`},
	} {
		got := render(t, tc.v)
		if rule := classRule(t, got, `<ui-text class="(ui-\w+)"`); !strings.Contains(rule, tc.want) {
			t.Errorf("rule = %q, want %q", rule, tc.want)
		}
	}
}

// TestStateModifierOverride pins per-state independence: a base
// modifier after a state-scoped one styles the other states only.
func TestStateModifierOverride(t *testing.T) {
	html := render(t, ui.Text("x").WhileHovered(ui.Foreground(ui.CSSColor("#00f"))).Foreground(ui.CSSColor("#f00")))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	for _, w := range []string{"color:#f00", "@media (hover: hover){&:hover{color:#00f}}"} {
		if !strings.Contains(got, w) {
			t.Errorf("rule = %q, missing %q:\n%s", got, w, html)
		}
	}
}

// TestStateBackgroundStacking pins the DOM-48 layering design: each
// state's rule declares its complete background list, with a
// state-scoped layer slotted at its chain position.
func TestStateBackgroundStacking(t *testing.T) {
	html := render(t, ui.Text("x").
		Background(ui.CSSColor("#a")).WhileHovered(

		ui.Background(ui.CSSColor("#b"))).
		Background(ui.CSSColor("#c")))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	for _, w := range []string{
		"background-color:#c;background-image:linear-gradient(#a,#a)",
		"@media (hover: hover){&:hover{background-image:linear-gradient(#a,#a),linear-gradient(#b,#b)}}",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("rule = %q, missing %q:\n%s", got, w, html)
		}
	}
}

// TestStateStroke pins the stroke carrier sharing: a state-scoped
// stroke draws on the same ::after carrier the base states declare.
// The exact carrier block also pins that the base state draws no
// stroke of its own.
func TestStateStroke(t *testing.T) {
	html := render(t, ui.Text("x").WhileFocused(ui.BorderStroke(2, ui.CSSColor("#00f"))))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	for _, w := range []string{
		`&::after{border-radius:inherit;content:"";inset:0;pointer-events:none;position:absolute}`,
		"&:focus-visible::after{box-shadow:inset 0 0 0 2px #00f}",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("rule = %q, missing %q:\n%s", got, w, html)
		}
	}
}

// TestStateUnionComposes pins the union closure: when several states
// style the same property, every union of the states declares their
// combined effect, outweighing the narrower variants while it holds.
func TestStateUnionComposes(t *testing.T) {
	html := render(t, ui.Text("x").
		Background(ui.CSSColor("#a")).WhileHovered(

		ui.Background(ui.CSSColor("#b"))).WhilePressed(

		ui.Background(ui.CSSColor("#d"))).
		WhileFocused(

			ui.Background(ui.CSSColor("#e"))))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	for _, w := range []string{
		"background-color:#a",
		"&:active{background-color:#d;background-image:linear-gradient(#a,#a)}",
		"&:hover{background-color:#b;background-image:linear-gradient(#a,#a)}",
		"&:hover:active{background-color:#d;background-image:linear-gradient(#a,#a),linear-gradient(#b,#b)}",
		"&:hover:focus-visible:active{background-color:#e;background-image:linear-gradient(#a,#a),linear-gradient(#b,#b),linear-gradient(#d,#d)}",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("rule = %q, missing %q:\n%s", got, w, html)
		}
	}
}

// TestStateNoChangeEmitsNothing pins the diffing: a state variant
// equal to the base paint declares nothing.
func TestStateNoChangeEmitsNothing(t *testing.T) {
	plain := render(t, ui.Text("x").Foreground(ui.CSSColor("#f00")))
	same := render(t, ui.Text("x").WhileHovered(ui.Foreground(ui.CSSColor("#f00"))).Foreground(ui.CSSColor("#f00")))
	if plain != same {
		t.Errorf("no-op state variant changed the rendering:\nplain:\n%s\nwith state:\n%s", plain, same)
	}
}

// TestStateUnionRestoresBase pins the subset override: when one state
// sets a property and another restores it to the base value, the
// union variant must redeclare the base value, or the single-state
// variant would still win while both states are active.
func TestStateUnionRestoresBase(t *testing.T) {
	got := render(t, ui.Text("x").WhilePressed(
		ui.Foreground(ui.CSSColor("#f00"))).
		WhileHovered(

			ui.Foreground(ui.CSSColor("#00f"))).
		Foreground(ui.CSSColor("#f00")))
	rule := classRule(t, got, `<ui-text class="(ui-\w+)"`)
	if !strings.Contains(rule, "&:hover:active{color:#f00}") {
		t.Errorf("union variant should restore the base color, got %q", rule)
	}
	if strings.Contains(rule, "&:active{") {
		t.Errorf("pressed variant equal to the base should declare nothing, got %q", rule)
	}
}

// TestOverlayAt pins the two-point lowering: the layer keeps its
// single-point placement at the base's at point, and the layered view
// is shifted by the two points' difference, in percentages of its own
// box, so its anchor point lands on at.
func TestOverlayAt(t *testing.T) {
	over := render(t, ui.Text("x").OverlayAt(ui.TopTrailing, ui.Center, ui.Text("o").Class("probe")))
	if got := classRule(t, over, `<ui-overlay class="(ui-\w+)"`); !strings.Contains(got, "place-items:start end") {
		t.Errorf("overlay placement should follow at, got %q:\n%s", got, over)
	}
	if got := classRule(t, over, `<ui-text class="probe (ui-\w+)"`); !strings.Contains(got, "translate:50% -50%") {
		t.Errorf("overlay view should shift its anchor onto at, got %q:\n%s", got, over)
	}
	// elm-ui's below: the underlay hangs off the base's bottom edge.
	under := render(t, ui.Text("x").UnderlayAt(ui.Bottom, ui.Top, ui.Text("u").Class("probe")))
	if got := classRule(t, under, `<ui-text class="probe (ui-\w+)"`); !strings.Contains(got, "translate:0% 100%") {
		t.Errorf("underlay view should shift its anchor onto at, got %q:\n%s", got, under)
	}
	// Coincident points shift nothing, matching Overlay's lowering.
	same := render(t, ui.Text("x").OverlayAt(ui.TopTrailing, ui.TopTrailing, ui.Text("o").Class("probe")))
	if strings.Contains(same, "translate") {
		t.Errorf("coincident points should not shift the overlay view:\n%s", same)
	}
}

// TestOverlayAtBaselinePanics pins the fixed-point contract: when the
// two points differ, FirstBaseline does not name a point to shift from,
// and rendering panics.
func TestOverlayAtBaselinePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a two-point overlay at FirstBaseline did not panic")
		}
	}()
	render(t, ui.Text("x").OverlayAt(ui.FirstBaseline, ui.Center, ui.Text("o")))
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

// TestGrid pins the grid lowering: the layout becomes the column
// template, the grid carries the library's default gap and alignment,
// a filling subview stretches across its cell, and its fill request
// propagates to the grid.
func TestGrid(t *testing.T) {
	for _, tt := range []struct {
		name   string
		layout ui.GridLayout
		want   string
	}{
		{"Columns", ui.Columns(3), "grid-template-columns:repeat(3, minmax(0, 1fr))"},
		{"CellMinWidth", ui.ColumnMinWidth(120), "grid-template-columns:repeat(auto-fill, minmax(120px, 1fr))"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, ui.VStack(ui.Grid(tt.layout, ui.CSSColor("#f00"), ui.Text("x"))))
			grid := classRule(t, html, `<ui-grid class="(ui-\w+)"`)
			for _, w := range []string{"display:grid", tt.want, "gap:8px", "place-items:center", "align-self:stretch"} {
				if !strings.Contains(grid, w) {
					t.Errorf("grid rule missing %q: %q", w, grid)
				}
			}
			cell := classRule(t, html, `<ui-color class="(ui-\w+)"`)
			if !strings.Contains(cell, "justify-self:stretch") {
				t.Errorf("filling subview should stretch across its cell, got %q", cell)
			}
		})
	}
	custom := classRule(t, render(t, ui.Grid(ui.Columns(2), ui.Text("x")).Gap(0).Alignment(ui.TopLeading)), `<ui-grid class="(ui-\w+)"`)
	for _, w := range []string{"gap:0px", "place-items:start start"} {
		if !strings.Contains(custom, w) {
			t.Errorf("grid rule missing %q: %q", w, custom)
		}
	}
}

// TestGridFill pins who asks for width: a Columns grid of non-filling
// subviews hugs them like a stack, while a CellMinWidth grid fills
// its available width regardless, since its column count depends on it.
func TestGridFill(t *testing.T) {
	hug := classRule(t, render(t, ui.VStack(ui.Grid(ui.Columns(2), ui.Text("x")))), `<ui-grid class="(ui-\w+)"`)
	if strings.Contains(hug, "align-self:stretch") {
		t.Errorf("Columns grid of non-filling subviews should hug them, got %q", hug)
	}
	fill := classRule(t, render(t, ui.VStack(ui.Grid(ui.ColumnMinWidth(100), ui.Text("x")))), `<ui-grid class="(ui-\w+)"`)
	if !strings.Contains(fill, "align-self:stretch") {
		t.Errorf("CellMinWidth grid should fill its width, got %q", fill)
	}
}

// TestFrameRatio pins the lowering: the ratio is declared, the
// anchor axis keeps the subview's fill, and the derived axis issues
// none, so no stretch can override the ratio.
func TestFrameRatio(t *testing.T) {
	wide := render(t, ui.VStack(ui.CSSColor("#f00").FrameRatio(2, 3, ui.Horizontal)))
	rule := classRule(t, wide, `<ui-aspect class="(ui-\w+)"`)
	for _, w := range []string{"aspect-ratio:2 / 3", "align-self:stretch", "min-height:0"} {
		if !strings.Contains(rule, w) {
			t.Errorf("width-anchored rule missing %q: %q", w, rule)
		}
	}
	if strings.Contains(rule, "flex-grow") {
		t.Errorf("derived height should not fill the column, got %q", rule)
	}
	tall := render(t, ui.HStack(ui.CSSColor("#f00").FrameRatio(2, 3, ui.Vertical)))
	rule = classRule(t, tall, `<ui-aspect class="(ui-\w+)"`)
	for _, w := range []string{"aspect-ratio:2 / 3", "align-self:stretch", "min-width:0", "flex-shrink:0", "writing-mode:vertical-lr"} {
		if !strings.Contains(rule, w) {
			t.Errorf("height-anchored rule missing %q: %q", w, rule)
		}
	}
	if strings.Contains(rule, "flex-grow") {
		t.Errorf("derived width should not fill the row, got %q", rule)
	}
	// The rotated frame's subview is rotated back, and its fills
	// are lowered in the frame's rotated axes.
	sub := classRule(t, tall, `<ui-color class="(ui-\w+)"`)
	for _, w := range []string{"writing-mode:horizontal-tb", "justify-self:stretch", "align-self:stretch"} {
		if !strings.Contains(sub, w) {
			t.Errorf("rotated frame's subview rule missing %q: %q", w, sub)
		}
	}
	half := render(t, ui.HStack(ui.Text("x").FrameRatio(2, 3, ui.Vertical)))
	if sub := classRule(t, half, `<ui-text class="(ui-\w+)"`); strings.Contains(sub, "stretch") {
		t.Errorf("non-filling subview should not stretch in the rotated frame, got %q", sub)
	}
}

// TestFrameRatioAlignment pins placement in the ratio frame: the
// alignment lowers directly for a horizontal anchor and on swapped
// axes for the rotated vertical anchor, which also keeps Leading on
// the leading edge in a right-to-left document.
func TestFrameRatioAlignment(t *testing.T) {
	wide := classRule(t, render(t, ui.Text("x").FrameRatio(2, 3, ui.Horizontal, ui.BottomLeading)), `<ui-aspect class="(ui-\w+)"`)
	if !strings.Contains(wide, "place-items:end start") {
		t.Errorf("width-anchored rule should place bottom leading, got %q", wide)
	}
	tall := classRule(t, render(t, ui.Text("x").FrameRatio(2, 3, ui.Vertical, ui.BottomLeading)), `<ui-aspect class="(ui-\w+)"`)
	for _, w := range []string{"place-items:start end", "&:dir(rtl){writing-mode:vertical-rl}"} {
		if !strings.Contains(tall, w) {
			t.Errorf("height-anchored rule missing %q: %q", w, tall)
		}
	}
}

// TestFrameBaselineIsTop pins the frames' definition of a baseline
// alignment: a frame has no baseline, so FirstBaseline is Top.
func TestFrameBaselineIsTop(t *testing.T) {
	for _, tt := range []struct {
		name string
		v    ui.View
	}{
		{"Frame", ui.Text("x").Frame(ui.Width(100), ui.FirstBaselineTrailing)},
		{"FrameBounds", ui.Text("x").FrameBounds(ui.MinWidth(100), ui.FirstBaselineTrailing)},
		{"FrameRatio", ui.Text("x").FrameRatio(1, 1, ui.Horizontal, ui.FirstBaselineTrailing)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			if !strings.Contains(html, "place-items:start end") || strings.Contains(html, "baseline") {
				t.Errorf("FirstBaseline in a frame should be Top:\n%s", html)
			}
		})
	}
}

// TestFrameRatioPanics pins the constructor's contract.
func TestFrameRatioPanics(t *testing.T) {
	for _, tt := range []struct {
		name string
		f    func()
	}{
		{"zero ratio", func() { ui.Text("x").FrameRatio(0, 1, ui.Horizontal) }},
		{"both axes", func() { ui.Text("x").FrameRatio(1, 1, ui.Horizontal|ui.Vertical) }},
		{"no axis", func() { ui.Text("x").FrameRatio(1, 1, 0) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			tt.f()
		})
	}
}

// TestGridLayoutPanics pins the layout constructors' contracts.
func TestGridLayoutPanics(t *testing.T) {
	for _, tt := range []struct {
		name string
		f    func()
	}{
		{"Columns(0)", func() { ui.Columns(0) }},
		{"CellMinWidth(0)", func() { ui.ColumnMinWidth(0) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			tt.f()
		})
	}
}
