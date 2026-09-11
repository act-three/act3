package hi_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"ily.dev/domi"

	"ily.dev/act3/hi"
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

func accountCard(user User) hi.View {
	return hi.Card(
		hi.HStack(
			hi.Image(user.PhotoURL).
				Alt(user.Name).
				ScaledToFill().
				Frame(hi.Width(48), hi.Height(48)).
				BorderShape(hi.Ellipse),
			hi.VStack(
				hi.Text(user.Name),
				hi.Text(user.Email).Foreground(hi.Secondary),
			).Gap(4).Alignment(hi.Leading),
			hi.Spacer(),
			hi.Button(Msg{EditProfile: true}, hi.Text("Edit")).ButtonStyle(hi.Prominent),
		).Gap(12).Alignment(hi.Center),
	).
		Padding(hi.Edges(16)).
		Underlay(hi.Center, hi.White).
		Overlay(hi.TopTrailing, hi.Badge("Pro"))
}

func moviePage(movies []Movie) hi.View {
	return hi.VStack(
		hi.HStack(
			hi.Text("Movies"),
			hi.Spacer(),
			hi.Button(Msg{NewMovie: true}, hi.Text("New")).ButtonStyle(hi.Prominent),
		).Alignment(hi.Center),
		hi.For(movies, movieKey, movieRow),
	).
		Gap(16).
		Padding(hi.Edges(32))
}

func movieKey(m Movie) string { return strconv.FormatUint(m.ID, 10) }

func movieRow(movie Movie) hi.View {
	return hi.HStack(
		hi.Image(movie.PosterURL).
			Alt(movie.Title).
			ScaledToFill().
			Frame(hi.Width(64), hi.Height(96)),
		hi.VStack(
			hi.Text(movie.Title),
			hi.Text(movie.Summary).Foreground(hi.Secondary),
		).Gap(4),
		hi.Spacer(),
		hi.Button(Msg{Watched: movie.ID}, hi.Text("Watched")),
	).Gap(12).Padding(hi.Edges(12))
}

// Theme colors as rendered under the default theme.
const (
	redCSS   = "oklch(0.654 0.232 28.7)"
	blueCSS  = "oklch(0.603 0.218 257.4)"
	whiteCSS = "oklch(1 0 0)"
	mutedCSS = "oklch(0.3388 0.00395 100)"
)

// pageRoot reports whether html is a page whose ui-root element
// carries the given attributes after its generated class,
// with the style element as its first child.
func pageRoot(html, attrs string) bool {
	return regexp.MustCompile(`^<ui-root class="ui-\w+"` + regexp.QuoteMeta(attrs) + `><style>`).MatchString(html)
}

func render(t *testing.T, v hi.View, o ...hi.Option) string {
	t.Helper()
	var sb strings.Builder
	_, page := hi.Render(v, o...)
	if err := domi.RenderTo(&sb, page); err != nil {
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
		`<ui-root `,         // root
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
		`align-items:start`, // the Overlay's alignment
		`justify-items:end`, // the Overlay's alignment
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
	if html := render(t, hi.VStack(hi.Text("a")).Tag("ul")); !strings.Contains(html, `<ul class="`) {
		t.Errorf("Tag should rename the stack's own element:\n%s", html)
	}

	html := render(t, hi.HStack(
		hi.HStack(hi.Text("a"), hi.Spacer()).Frame().Tag("nav"),
		hi.Text("b"),
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
		v            hi.View
		want, absent []string
	}{
		{
			"send",
			hi.Button(Msg{}, hi.Text("x")),
			[]string{`<button `, ` type="button"`},
			[]string{` disabled`},
		},
		{
			"send disabled",
			hi.Button(Msg{}, hi.Text("x")).Disabled(true),
			[]string{`<button `, ` disabled`},
			nil,
		},
		{
			"navigate",
			hi.Button("/movies", hi.Text("x")),
			[]string{`<a `, ` href="/movies"`},
			[]string{`aria-disabled`},
		},
		{
			"navigate disabled",
			hi.Button("/movies", hi.Text("x")).Disabled(true),
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

// TestButtonLabelArity pins the button label's arity rule:
// empty and multi-node labels are arranged in an HStack,
// while a single node remains direct.
func TestButtonLabelArity(t *testing.T) {
	for _, tt := range []struct {
		name  string
		label hi.View
		stack bool
	}{
		{"empty", hi.Empty(), true},
		{"single", hi.Text("a"), false},
		{"multiple", hi.Group(hi.Text("a"), hi.Text("b")), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, hi.Button(Msg{}, tt.label))
			if got := strings.Contains(html, "<ui-hstack "); got != tt.stack {
				t.Errorf("HStack present = %v, want %v:\n%s", got, tt.stack, html)
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
		v    hi.View
		want string
	}{
		{"intrinsic tag", hi.Button(Msg{}, hi.Text("x")).Tag("figure"), "<button"},
		{"double Tag", hi.VStack().Tag("ul").Tag("ol"), "<ul"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			if !strings.Contains(html, tt.want) {
				t.Errorf("missing %q:\n%s", tt.want, html)
			}
		})
	}
}

// TestHTMLHost pins the [hi.HTML] host contract: the node renders inside
// a host element the view manages, and element modifiers land on that
// host rather than on the node.
func TestHTMLHost(t *testing.T) {
	html := render(t, hi.HTML(domi.Text("raw")))
	if want := regexp.MustCompile(`<ui-html class="ui-\w+">raw</ui-html>`); !want.MatchString(html) {
		t.Errorf("missing %q:\n%s", want, html)
	}

	mod := render(t, hi.HTML(domi.Text("raw")).BorderShape(hi.Ellipse).Class("x").Tag("section"))
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
		v       hi.View
		wants   []string
		rejects []string
	}{
		{
			"both axes at the root grid",
			hi.HTML(domi.Text("raw")),
			[]string{"align-self:stretch", "justify-self:stretch"},
			nil,
		},
		{
			"a row grows it, stretches it, and inherits the fill",
			hi.HStack(hi.HTML(domi.Text("raw"))),
			[]string{"flex-grow:1", "align-self:stretch", "justify-self:stretch"},
			nil,
		},
		{
			"FixedSize clears the fill axes",
			hi.HTML(domi.Text("raw")).FixedSize(),
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
	html := render(t, hi.HTML(domi.Text("raw")).Padding(hi.Edges(4)).Background(hi.Red))
	if got := classRule(t, html, `<ui-padding class="(ui-\w+)"`); got != "align-items:center;align-self:stretch;background-color:"+redCSS+";display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-items:center;justify-self:stretch;padding-block-end:4px;padding-block-start:4px;padding-inline-end:4px;padding-inline-start:4px" {
		t.Errorf("padding wrapper should carry the paint, got %q:\n%s", got, html)
	}
	if got := classRule(t, html, `<ui-html class="(ui-\w+)"`); got != "align-items:center;align-self:stretch;display:grid;grid-template-columns:100%;grid-template-rows:100%;justify-items:center;justify-self:stretch" {
		t.Errorf("host should stay untouched inside, got %q:\n%s", got, html)
	}
}

// TestImmutableModifiers verifies the load-bearing value-semantics invariant:
// applying a modifier to a shared view must not affect the original.
func TestImmutableModifiers(t *testing.T) {
	header := hi.Text("Title")

	padded := header.Padding(hi.Edges(16))
	plain := header.Padding(hi.Edges(0))

	ph := render(t, padded)
	if !strings.Contains(ph, "padding-block-start:16px") {
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
	html := render(t, hi.Secondary)
	if got := classRule(t, html, `<ui-color class="(ui-\w+)"`); got != "align-self:stretch;background-color:"+mutedCSS+";justify-self:stretch" {
		t.Errorf("color view should paint its own box and fill both axes, got %q:\n%s", got, html)
	}
	if mod := render(t, hi.OKLCH(0.9, 0, 0).Opacity(0.5)); !strings.Contains(mod, "opacity:0.5") {
		t.Errorf("modifier on a color view should reach its box:\n%s", mod)
	}

	// Background layers behind the color on the color's own element,
	// visible where c is translucent — ordinary painting order, not a
	// decoration layer, and the Modify spelling is the same lowering.
	bg := render(t, hi.OKLCHA(0, 0, 0, 0.5).Background(hi.White))
	if got := classRule(t, bg, `<ui-color class="(ui-\w+)"`); got != "align-self:stretch;background-color:"+whiteCSS+";background-image:linear-gradient(oklch(0 0 0 / 0.5),oklch(0 0 0 / 0.5));justify-self:stretch" {
		t.Errorf("Background should layer under the color, got %q:\n%s", got, bg)
	}
	if strings.Contains(bg, "ui-underlay") {
		t.Errorf("Background on a color should merge, not add a layer:\n%s", bg)
	}
	if mod := render(t, hi.OKLCHA(0, 0, 0, 0.5).Modify(hi.Background(hi.White))); mod != bg {
		t.Errorf("Modify(Background) diverged from the Background method:\n%s", mod)
	}
	// Underlay layers content behind the color.
	under := render(t, hi.OKLCHA(0, 0, 0, 0.5).Underlay(hi.Center, hi.Text("behind")))
	for _, w := range []string{`<ui-underlay `, "behind"} {
		if !strings.Contains(under, w) {
			t.Errorf("Underlay behind a color missing %q:\n%s", w, under)
		}
	}
	// Modifiers with no possible effect on a color are no-ops.
	if noop := render(t, hi.Secondary.Foreground(hi.White)); noop != html {
		t.Errorf("no-effect modifiers on a color should be no-ops:\n%s", noop)
	}
}

// TestPaddingComposes checks that separate Padding modifiers compose by
// nesting instead of clobbering each other: each emits its own declarations,
// padding only the edges it names.
func TestPaddingComposes(t *testing.T) {
	html := render(t, hi.Text("hi").Padding(hi.EdgeTop(8)).Padding(hi.EdgesPillarbox(12)))
	for _, w := range []string{
		"padding-block-start:8px",
		"padding-inline-start:0",
		"padding-block-start:0",
		"padding-inline-start:12px",
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
		v       hi.View
		wants   []string
		rejects []string
	}{
		{
			// The color's own box is the FixedSize boundary: its
			// fills are stripped, so it takes its ideal as its size.
			"direct FixedSize",
			hi.Secondary.FixedSize(),
			[]string{"width:10px", "height:10px"},
			[]string{"min-width", "min-height"},
		},
		{
			// The stack is the boundary; the color inside keeps its
			// fills and contributes its ideal as a minimum.
			"FixedSize on an ancestor",
			hi.VStack(hi.Secondary).FixedSize(),
			[]string{"min-width:10px", "min-height:10px"},
			nil,
		},
		{
			"definite frame axis clears its axis only",
			hi.VStack(hi.Secondary).Frame(hi.Width(200)).FixedSize(),
			[]string{"min-height:10px"},
			[]string{"min-width"},
		},
		{
			"both definite axes clear both",
			hi.VStack(hi.Secondary).Frame(hi.Width(200), hi.Height(100)).FixedSize(),
			nil,
			[]string{"10px"},
		},
		{
			"scroll viewport takes 100px; content unbounded on the scroll axis",
			hi.ScrollView(hi.Vertical, hi.Secondary).FixedSize().Padding(hi.Edges(0)),
			[]string{"width:100px", "height:100px", "height:10px"},
			[]string{"width:10px"},
		},
		{
			// The color's fill is stripped on the scroll axis only,
			// so its ideal is its size there, and it keeps filling
			// the bounded cross axis.
			"scroll axis is unbounded without FixedSize",
			hi.ScrollView(hi.Vertical, hi.Secondary).Padding(hi.Edges(0)),
			[]string{"height:10px;justify-self:stretch"},
			[]string{"width:10px"},
		},
		{
			"both-axes scroll makes both content axes unbounded",
			hi.ScrollView(hi.Horizontal|hi.Vertical, hi.Secondary).Padding(hi.Edges(0)),
			[]string{"width:10px", "height:10px"},
			nil,
		},
		{
			"no-axis scroll makes neither content axis unbounded",
			hi.ScrollView(hi.AxisSet(0), hi.Secondary).FixedSize().Padding(hi.Edges(0)),
			[]string{"width:100px", "height:100px", "overflow-x:clip;overflow-y:clip"},
			[]string{"10px"},
		},
		{
			"bounds frame takes its ideal and makes it the subview's space",
			hi.VStack(hi.Secondary).FrameBounds(hi.IdealWidth(200), hi.IdealHeight(80)).FixedSize(),
			[]string{"width:200px", "height:80px"},
			[]string{"10px"},
		},
		{
			"bounds frame ideal is inert in bounded space",
			hi.Text("x").FrameBounds(hi.IdealWidth(200)),
			nil,
			[]string{"width"},
		},
		{
			// A scaling mode meets an imposed box; with no box to
			// meet, the img's intrinsic geometry answers instead.
			"scaled image drops its fills on unbounded axes",
			hi.Image("/x.png").ScaledToFill().FixedSize(),
			[]string{"object-fit:cover"},
			[]string{"justify-self", "align-self"},
		},
		{
			// The viewport itself stays greedy on both axes; only
			// the image's own fill is dropped on the scroll axis.
			"scaled image keeps its fill on the bounded cross axis",
			hi.ScrollView(hi.Vertical, hi.Image("/x.png").ScaledToFill()).Padding(hi.Edges(0)),
			[]string{"justify-self:stretch;min-height:0;min-width:0;object-fit:cover"},
			[]string{"align-self:stretch;justify-self:stretch;min-height"},
		},
		{
			"divider contributes 10px along its length",
			hi.VStack(hi.Divider()).FixedSize(),
			[]string{"height:1px", "min-width:10px", "align-self:stretch"},
			[]string{"min-height"},
		},
		{
			"vertical divider contributes 10px along its length",
			hi.HStack(hi.Divider()).FixedSize(),
			[]string{"width:1px", "min-height:10px", "align-self:stretch"},
			[]string{"min-width"},
		},
		{
			"decoration layer clears both axes",
			hi.Text("x").Overlay(hi.Center, hi.Secondary).FixedSize(),
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
	html := render(t, hi.Text("hi").Padding(hi.EdgeTop(8), hi.Edges(4)))
	for _, w := range []string{
		"padding-block-start:12px",
		"padding-block-end:4px",
		"padding-inline-start:4px",
		"padding-inline-end:4px",
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
	v := hi.Text("Status: ").
		TextFont(hi.Bold).
		Concat(hi.Text("Draft").TextFont(hi.Italic)).
		TextForeground(hi.Secondary)

	html := render(t, v)
	for _, w := range []string{"font-weight:700", "font-style:italic", "Status: ", "Draft"} {
		if !strings.Contains(html, w) {
			t.Errorf("rich text missing %q\n\n%s", w, html)
		}
	}
	// TextColor applies to the whole text: once, on the enclosing element,
	// where every run inherits it.
	if got := strings.Count(html, mutedCSS); got != 1 {
		t.Errorf("muted color count = %d, want 1 (whole-text color)\n\n%s", got, html)
	}
}

func TestFontOptionsCopied(t *testing.T) {
	for name, makeView := range map[string]func(...hi.FontOption) hi.View{
		"view": func(opts ...hi.FontOption) hi.View { return hi.Text("x").Font(opts...) },
		"text": func(opts ...hi.FontOption) hi.View { return hi.Text("x").TextFont(opts...) },
	} {
		t.Run(name, func(t *testing.T) {
			opts := []hi.FontOption{hi.Italic, hi.OpenTypeFeature("ss01", 1)}
			v := makeView(opts...)
			want := render(t, v)
			opts[0] = hi.Roman
			opts[1] = hi.OpenTypeFeature("ss01", 0)
			if got := render(t, v); got != want {
				t.Error("changing the options slice changed an existing view")
			}
		})
	}
}

func TestFontOptions(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts []hi.FontOption
		want string
	}{
		{"reset italic", []hi.FontOption{hi.Italic, hi.Roman, hi.Family("serif")},
			"display:block;font-family:serif;font-style:normal;overflow-wrap:break-word"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for name, v := range map[string]hi.View{
				"view": hi.Text("x").Font(tt.opts...),
				"text": hi.Text("x").TextFont(tt.opts...),
			} {
				t.Run(name, func(t *testing.T) {
					html := render(t, v)
					if got := classRule(t, html, `<ui-text class="(ui-\w+)"`); got != tt.want {
						t.Errorf("font rule = %q, want %q", got, tt.want)
					}
				})
			}
		})
	}

	if got, want := render(t, hi.Text("x").Font()), render(t, hi.Text("x")); got != want {
		t.Errorf("empty Font changed rendering:\n%s", got)
	}
}

// TestTextWholeTextRule pins the whole-text rule: a text modifier applied
// after Concat styles all runs, landing on the text's own element, while
// a run styled before Concat keeps its own styling.
func TestTextWholeTextRule(t *testing.T) {
	html := render(t, hi.Text("a").Concat(hi.Text("b").TextFont(hi.Italic)).TextFont(hi.Bold))
	if got := classRule(t, html, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "font-weight:700") {
		t.Errorf("whole-text TextFont should land on the text element, got %q:\n%s", got, html)
	}
	if got := classRule(t, html, `>a<span class="(ui-\w+)"`); got != "font-style:italic" {
		t.Errorf("pre-Concat TextFont should stay on its own run, got %q:\n%s", got, html)
	}
}

// TestLink pins the lowering of each kind of link action: a URL
// navigates from an anchor, a message sends from a button element,
// and either one is an inline run that carries the pending style.
// A link used as a box stays the anchor element, and text modifiers
// applied outside the link lower to the class on that element.
func TestLink(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       hi.View
		pattern string
		want    []string
	}{
		{
			"navigate",
			hi.Text("see ").Concat(hi.Link("/docs", hi.Text("docs"))),
			`<a class="(ui-\w+)" href="/docs">docs</a>`,
			nil,
		},
		{
			"send",
			hi.Text("or ").Concat(hi.Link(Msg{}, hi.Text("retry"))),
			`<button class="(ui-\w+)" domi-msg-click="[^"]*" type="button">retry</button>`,
			nil,
		},
		{
			"outer style",
			hi.Link("/docs", hi.Text("docs")).TextFont(hi.Bold),
			`</style><a class="(ui-\w+)" href="/docs">docs</a>`,
			[]string{"font-weight:700"},
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
// means of activation, while an enabled one retains it.
func TestLinkDisabled(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       hi.View
		pattern string
		want    []string
		absent  []string
	}{
		{
			"navigate",
			hi.Text("a").Concat(hi.Link("/docs", hi.Text("x"))).Disabled(true),
			`<a aria-disabled="true" class="(ui-\w+)" role="link">x</a>`,
			nil,
			[]string{` href=`},
		},
		{
			"send",
			hi.Text("a").Concat(hi.Link(Msg{}, hi.Text("x"))).Disabled(true),
			`<button class="(ui-\w+)" disabled domi-msg-click="[^"]*" type="button">x</button>`,
			nil,
			nil,
		},
		{
			"enabled",
			hi.Text("a").Concat(hi.Link("/docs", hi.Text("x"))).Disabled(false),
			`<a class="(ui-\w+)" href="/docs">x</a>`,
			nil,
			[]string{"aria-disabled"},
		},
		{
			"block",
			hi.Link("/docs", hi.Text("x")).Disabled(true),
			`<a aria-disabled="true" class="(ui-\w+)" role="link">x</a>`,
			[]string{"display:block"},
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

// TestLinkPolicy pins LinkPolicy: every navigating link and button
// in the subtree gains the domi annotation for a non-default policy,
// the innermost policy wins, and a sending or disabled link is
// unaffected.
func TestLinkPolicy(t *testing.T) {
	for name, tc := range map[string]struct {
		v    hi.View
		want string // the domi-handle attribute value, or "" for none
	}{
		"none":     {hi.Link("/docs", hi.Text("x")).LinkPolicy(hi.HandleNone), "no"},
		"all":      {hi.Link("/docs", hi.Text("x")).LinkPolicy(hi.HandleAll), "yes"},
		"default":  {hi.Link("/docs", hi.Text("x")).LinkPolicy(hi.HandleSameOrigin), ""},
		"run":      {hi.Text("a").Concat(hi.Link("/docs", hi.Text("x"))).LinkPolicy(hi.HandleNone), "no"},
		"subtree":  {hi.VStack(hi.Link("/docs", hi.Text("x"))).LinkPolicy(hi.HandleNone), "no"},
		"button":   {hi.Button("/docs", hi.Text("x")).LinkPolicy(hi.HandleNone), "no"},
		"inner":    {hi.VStack(hi.Link("/docs", hi.Text("x")).LinkPolicy(hi.HandleAll)).LinkPolicy(hi.HandleNone), "yes"},
		"send":     {hi.Link(Msg{}, hi.Text("x")).LinkPolicy(hi.HandleNone), ""},
		"disabled": {hi.Link("/docs", hi.Text("x")).LinkPolicy(hi.HandleNone).Disabled(true), ""},
		"without":  {hi.Link("/docs", hi.Text("x")), ""},
	} {
		t.Run(name, func(t *testing.T) {
			html := render(t, tc.v)
			var got string
			if m := regexp.MustCompile(`domi-handle="([^"]*)"`).FindStringSubmatch(html); m != nil {
				got = m[1]
			}
			if got != tc.want {
				t.Errorf("domi-handle = %q, want %q:\n%s", got, tc.want, html)
			}
		})
	}
}

// TestDisabledSubtree pins Disabled's reach and stickiness: it
// disables every control below the modifier, and a control inside a
// disabled view cannot be re-enabled.
func TestDisabledSubtree(t *testing.T) {
	for name, v := range map[string]hi.View{
		"button":  hi.VStack(hi.Button(Msg{}, hi.Text("x"))).Disabled(true),
		"sticky":  hi.VStack(hi.Button(Msg{}, hi.Text("x")).Disabled(false)).Disabled(true),
		"navlink": hi.VStack(hi.Text("a").Concat(hi.Link("/x", hi.Text("x")))).Disabled(true),
	} {
		t.Run(name, func(t *testing.T) {
			html := render(t, v)
			if !strings.Contains(html, " disabled") && !strings.Contains(html, `aria-disabled="true"`) {
				t.Errorf("control should be disabled:\n%s", html)
			}
			if strings.Contains(html, " href=") {
				t.Errorf("disabled link should drop its URL:\n%s", html)
			}
		})
	}
}

// TestTextStyleBeatsStatePaint pins that text styling is
// unconditional: a state-scoped paint modifier applies while its
// states are active, but never overrides a property the text
// styling sets, which is written closest to the view.
func TestTextStyleBeatsStatePaint(t *testing.T) {
	// TextForeground holds in every state, even against a hovered color.
	html := render(t, hi.Text("x").TextForeground(hi.Red).WhileHovered(hi.Foreground(hi.Blue)))
	if strings.Contains(html, "hover") || !strings.Contains(html, "color:"+redCSS) {
		t.Errorf("hovered color should lose to TextForeground entirely:\n%s", html)
	}
}

// TestLineLimit pins the lowering and reach of LineLimit: it clamps
// the text it is applied to, reaches every text below the modifier,
// and the limit nearest a text wins.
func TestLineLimit(t *testing.T) {
	direct := render(t, hi.Text("x").LineLimit(2))
	if got := classRule(t, direct, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "-webkit-line-clamp:2") {
		t.Errorf("rule = %q, want -webkit-line-clamp:2:\n%s", got, direct)
	}

	subtree := render(t, hi.VStack(hi.Text("a"), hi.Text("b").LineLimit(1)).LineLimit(3))
	for want, n := range map[string]int{"-webkit-line-clamp:3": 1, "white-space:nowrap": 1} {
		if got := strings.Count(subtree, want); got != n {
			t.Errorf("%s count = %d, want %d:\n%s", want, got, n, subtree)
		}
	}

	// The limit survives boundaries that establish a new layout
	// context for their contents.
	for name, v := range map[string]hi.View{
		"scroll":  hi.ScrollView(hi.Vertical, hi.Text("x")).LineLimit(2),
		"overlay": hi.VStack().Overlay(hi.Center, hi.Text("x")).LineLimit(2),
	} {
		if got := render(t, v); !strings.Contains(got, "-webkit-line-clamp:2") {
			t.Errorf("%s content should clamp:\n%s", name, got)
		}
	}
}

func TestLineLimitClearsInheritedLimit(t *testing.T) {
	for _, n := range []int{0, -1} {
		html := render(t, hi.VStack(hi.Text("x").LineLimit(n)).LineLimit(2))
		if strings.Contains(html, "-webkit-line-clamp") {
			t.Errorf("limit %d should clear inherited clamping:\n%s", n, html)
		}
	}
}

// Guard against an accidental change to the keyed-row idiom shown in the doc.
func TestForKeyLikeIdiom(t *testing.T) {
	items := []Movie{{ID: 7}, {ID: 42}}
	v := hi.VStack(hi.For(items, movieKey, func(m Movie) hi.View {
		return hi.Text("#" + strconv.FormatUint(m.ID, 10))
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
	h := render(t, hi.HStack(hi.Text("a")).Alignment(hi.TopTrailing))
	if !strings.Contains(h, "align-items:start") {
		t.Errorf("HStack TopTrailing should align to the top:\n%s", h)
	}
	v := render(t, hi.VStack(hi.Text("a")).Alignment(hi.TopTrailing))
	if !strings.Contains(v, "align-items:end") {
		t.Errorf("VStack TopTrailing should align to the trailing edge:\n%s", v)
	}

	f := render(t, hi.Text("a").Frame(hi.Width(100), hi.BottomTrailing))
	if !strings.Contains(f, "align-items:end") || !strings.Contains(f, "justify-items:end") {
		t.Errorf("frame Align should place the content in both axes:\n%s", f)
	}

	// An alignment whose relevant axis projects to center emits the
	// center it projects to.
	c := render(t, hi.VStack(hi.Text("a")).Alignment(hi.Top))
	if !strings.Contains(c, "align-items:center") {
		t.Errorf("VStack Top projects to center on the cross axis:\n%s", c)
	}
}

// TestDividerAxisAware checks that a divider orients against its stack's axis:
// vertical inside an HStack, horizontal inside a VStack, stretching along the
// minor axis either way.
func TestDividerAxisAware(t *testing.T) {
	h := render(t, hi.HStack(hi.Text("a"), hi.Divider(), hi.Text("b")))
	if got := classRule(t, h, `<ui-divider class="(ui-\w+)"`); !strings.Contains(got, "align-self:stretch") || !strings.Contains(got, "width:") {
		t.Errorf("divider in HStack should be vertical and stretch, got %q:\n%s", got, h)
	}

	v := render(t, hi.VStack(hi.Text("a"), hi.Divider(), hi.Text("b")))
	if got := classRule(t, v, `<ui-divider class="(ui-\w+)"`); !strings.Contains(got, "align-self:stretch") || !strings.Contains(got, "height:") {
		t.Errorf("divider in VStack should be horizontal and stretch, got %q:\n%s", got, v)
	}
}

// TestForKeysItems checks that each For item carries its key directly on its
// own element, with no wrapper in between.
func TestForKeysItems(t *testing.T) {
	items := []Movie{{ID: 7, Title: "Seven"}, {ID: 42, Title: "Forty-Two"}}
	v := hi.VStack(hi.For(items,
		func(m Movie) string { return strconv.FormatUint(m.ID, 10) },
		func(m Movie) hi.View { return hi.Text(m.Title) },
	))
	html := render(t, v)
	for _, w := range []string{"ui-vstack", `domi-key="7"`, `domi-key="42"`, "Seven", "Forty-Two"} {
		if !strings.Contains(html, w) {
			t.Errorf("keyed For missing %q\n\n%s", w, html)
		}
	}
}

// TestForNilKeyUnkeyed pins the unkeyed mode: a nil key splices the items
// with no key stamps and no per-item element requirement, so an item may
// render to several elements.
func TestForNilKeyUnkeyed(t *testing.T) {
	items := []Movie{{ID: 7, Title: "Seven"}}
	html := render(t, hi.VStack(hi.For(items, nil, func(m Movie) hi.View {
		return hi.Group(hi.Text(m.Title), hi.Text(m.Title))
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
	html := render(t, hi.HStack(hi.Image("/x.png").Alt("pic")))
	if strings.Contains(html, "object-fit") {
		t.Errorf("native image should have no framing mode:\n%s", html)
	}
	if want := `<img alt="pic" class="ui-\w+" src="/x.png">`; !regexp.MustCompile(want).MatchString(html) {
		t.Errorf("native image missing %q:\n%s", want, html)
	}
}

// TestImageStroked pins that the wrapper a stroked image boxes out into
// keeps the image's own alt text and framing mode.
func TestImageStroked(t *testing.T) {
	html := render(t, hi.Image("/x.png").Alt("pic").ScaledToFill().BorderStroke(1, hi.Accent))
	if want := `<img alt="pic" class="ui-\w+" src="/x.png">`; !regexp.MustCompile(want).MatchString(html) {
		t.Errorf("stroked image missing %q:\n%s", want, html)
	}
	if !strings.Contains(html, "object-fit:cover") {
		t.Errorf("stroked image missing object-fit:cover:\n%s", html)
	}
}

// TestScrollView checks the requested axis selects the right overflow variant.
func TestScrollView(t *testing.T) {
	cases := map[hi.AxisSet]string{
		hi.Vertical:                 "overflow-x:hidden;overflow-y:auto",
		hi.Horizontal:               "overflow-x:auto;overflow-y:hidden",
		hi.Horizontal | hi.Vertical: "overflow-x:auto;overflow-y:auto",
		hi.AxisSet(0):               "overflow-x:clip;overflow-y:clip",
	}
	for axis, want := range cases {
		// Padding makes the ScrollView a subview, so this test exercises
		// the ordinary element viewport rather than page scrolling.
		html := render(t, hi.ScrollView(axis, hi.Text("content")).Padding(hi.Edges(0)))
		if !strings.Contains(html, want) {
			t.Errorf("ScrollView(%v) missing %q\n\n%s", axis, want, html)
		}
		if !strings.Contains(html, "isolation:isolate") {
			t.Errorf("ScrollView(%v) missing viewport isolation\n\n%s", axis, html)
		}
	}
}

// TestScrollViewContentArity pins the scroll viewport's arity rule:
// non-unary content is arranged in a VStack, while a single node
// remains directly inside the viewport.
func TestScrollViewContentArity(t *testing.T) {
	group := render(t, hi.ScrollView(hi.Vertical,
		hi.Group(hi.Text("a"), hi.Text("b"))).Padding(hi.Edges(0)))
	if !strings.Contains(group, "<ui-scroll ") || !strings.Contains(group, "<ui-vstack ") {
		t.Errorf("a ScrollView Group should be wrapped in a VStack:\n%s", group)
	}

	single := render(t, hi.ScrollView(hi.Vertical, hi.Text("a")).Padding(hi.Edges(0)))
	if strings.Contains(single, "<ui-vstack ") {
		t.Errorf("a single ScrollView node should not be wrapped:\n%s", single)
	}

	empty := render(t, hi.ScrollView(hi.Vertical, hi.Empty()).Padding(hi.Edges(0)))
	if !strings.Contains(empty, "<ui-vstack ") {
		t.Errorf("an empty ScrollView should be wrapped in a VStack:\n%s", empty)
	}
}

// TestScrollViewPageLowering pins the narrow root specialization: a bare root
// ScrollView delegates scrolling to the document and splices out ui-scroll,
// while an enclosing wrapper or rendering modifier keeps the ordinary element
// viewport. Ancillary metadata remains transparent to the specialization.
func TestScrollViewPageLowering(t *testing.T) {
	cases := []struct {
		axis hi.AxisSet
		want string
	}{
		{hi.Horizontal, `scroll="x"`},
		{hi.Vertical, `scroll="y"`},
		{hi.Horizontal | hi.Vertical, `scroll="x y"`},
	}
	for _, tt := range cases {
		html := render(t, hi.ScrollView(tt.axis, hi.Text("content")))
		if !pageRoot(html, " "+tt.want) {
			t.Errorf("ScrollView(%v) root missing %q:\n%s", tt.axis, tt.want, html)
		}
		if strings.Contains(html, "<ui-scroll ") {
			t.Errorf("root ScrollView(%v) retained its element viewport:\n%s", tt.axis, html)
		}
	}

	none := render(t, hi.ScrollView(hi.AxisSet(0), hi.Text("content")))
	if !pageRoot(none, "") || strings.Contains(none, "<ui-scroll ") {
		t.Errorf("a root ScrollView with no axes should lower as an ordinary page root:\n%s", none)
	}

	wrapped := render(t, hi.ScrollView(hi.Vertical, hi.Text("content")).Padding(hi.Edges(0)))
	if !pageRoot(wrapped, "") || !strings.Contains(wrapped, "<ui-scroll ") {
		t.Errorf("padding should prevent page lowering:\n%s", wrapped)
	}

	modified := []struct {
		name string
		view hi.View
	}{
		{"background", hi.ScrollView(hi.Vertical, hi.Text("content")).Background(hi.Red)},
		{"class", hi.ScrollView(hi.Vertical, hi.Text("content")).Class("sentinel")},
		{"attribute", hi.ScrollView(hi.Vertical, hi.Text("content")).Attr(domi.Name("data-sentinel", "true"))},
		{"fixed size", hi.ScrollView(hi.Vertical, hi.Text("content")).FixedSize()},
		{"clipping", hi.ScrollView(hi.Vertical, hi.Text("content")).BorderClipped()},
	}
	for _, tt := range modified {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.view)
			if !pageRoot(html, "") || !strings.Contains(html, "<ui-scroll ") {
				t.Errorf("rendering modifier should prevent page lowering:\n%s", html)
			}
		})
	}

	background := render(t, modified[0].view)
	if got := classRule(t, background, `<ui-scroll class="(ui-\w+)"`); !strings.Contains(got, "background-color:"+redCSS) {
		t.Errorf("background should remain on the element viewport, got %q", got)
	}

	titled := hi.ScrollView(hi.Vertical, hi.Text("content")).Title("title")
	if title, _ := hi.Render(titled); title != "title" {
		t.Errorf("root ScrollView title = %q, want title", title)
	}
	titledHTML := render(t, titled)
	if !pageRoot(titledHTML, ` scroll="y"`) || strings.Contains(titledHTML, "<ui-scroll ") {
		t.Errorf("Title should preserve page lowering:\n%s", titledHTML)
	}
}

func TestSticky(t *testing.T) {
	html := render(t, hi.Text("h").Sticky())
	rule := classRule(t, html, `<ui-sticky class="([^" ]+)`)
	for _, w := range []string{
		"position:sticky", "z-index:1",
		"inset-block-start:0", "inset-block-end:0", "inset-inline-start:0", "inset-inline-end:0",
	} {
		if !strings.Contains(rule, w) {
			t.Errorf("sticky box missing %q, got %q", w, rule)
		}
	}
}

func TestStickyInsetsAdd(t *testing.T) {
	html := render(t, hi.Text("h").Sticky(hi.EdgeTop(60), hi.Edges(4)))
	for _, w := range []string{"inset-block-start:64px", "inset-block-end:4px", "inset-inline-start:4px", "inset-inline-end:4px"} {
		if !strings.Contains(html, w) {
			t.Errorf("summed insets missing %q:\n%s", w, html)
		}
	}
}

// TestStickyModifierOrder pins the enclosure contract: a modifier
// applied inside Sticky is held in view along with the receiver, while
// one applied outside encloses the sticky box and confines it.
func TestStickyModifierOrder(t *testing.T) {
	inside := render(t, hi.Text("h").Padding(hi.Edges(8)).Sticky())
	if s, p := strings.Index(inside, "<ui-sticky"), strings.Index(inside, "<ui-padding"); s < 0 || p < 0 || s > p {
		t.Errorf("padding inside Sticky: want ui-sticky enclosing ui-padding:\n%s", inside)
	}
	outside := render(t, hi.Text("h").Sticky().Padding(hi.Edges(8)))
	if s, p := strings.Index(outside, "<ui-sticky"), strings.Index(outside, "<ui-padding"); s < 0 || p < 0 || p > s {
		t.Errorf("padding outside Sticky: want ui-padding enclosing ui-sticky:\n%s", outside)
	}
}

// TestFrameBounds checks the bounds frame's lowering: bounds and alignment on
// its own box, fill requests relayed above a minimum, and omitted bounds
// emitting no sizing declarations.
func TestFrameBounds(t *testing.T) {
	bounded := render(t, hi.Text("x").FrameBounds(hi.MinWidth(96), hi.MinHeight(24), hi.Leading))
	for _, w := range []string{
		"min-width:96px", "min-height:24px", "align-items:center", "justify-items:start",
		// An explicit minimum zeroes the axis's intrinsic track so the
		// min-* declaration is the floor.
		"grid-template-columns:minmax(0, 100%)", "grid-template-rows:minmax(0, 100%)",
	} {
		if !strings.Contains(bounded, w) {
			t.Errorf("bounded frame missing %q\n\n%s", w, bounded)
		}
	}

	relay := render(t, hi.HStack(hi.Spacer()).FrameBounds(hi.MinWidth(96)))
	if !strings.Contains(relay, "justify-self:stretch") {
		t.Errorf("a min-bounded frame should relay its subview's fill:\n%s", relay)
	}

	unbounded := render(t, hi.Text("x").FrameBounds())
	for _, r := range []string{"width", "height", "minmax"} {
		if strings.Contains(unbounded, r) {
			t.Errorf("omitted bounds should emit nothing, got %q:\n%s", r, unbounded)
		}
	}

	// The ideal and the bounds apply in order, each adjusting an
	// earlier conflicting slot to itself.
	for _, tt := range []struct {
		name  string
		v     hi.View
		wants []string
	}{
		{
			"later ideal lowers an earlier min",
			hi.Text("x").FrameBounds(hi.MinWidth(200), hi.IdealWidth(100)).FixedSize(),
			[]string{"width:100px", "min-width:100px"},
		},
		{
			"later min raises an earlier ideal",
			hi.Text("x").FrameBounds(hi.IdealWidth(100), hi.MinWidth(200)).FixedSize(),
			[]string{"width:200px", "min-width:200px"},
		},
		{
			"zero minima remain explicit",
			hi.Text("x").FrameBounds(hi.MinWidth(0), hi.MinHeight(0)),
			[]string{"min-width:0", "min-height:0", "grid-template-columns:minmax(0, 100%)", "grid-template-rows:minmax(0, 100%)"},
		},
		{
			"zero ideals lower earlier minima",
			hi.Text("x").FrameBounds(hi.MinWidth(100), hi.MinHeight(100), hi.IdealWidth(0), hi.IdealHeight(0)).FixedSize(),
			[]string{"width:0", "height:0", "min-width:0", "min-height:0"},
		},
	} {
		html := render(t, tt.v)
		for _, w := range tt.wants {
			if !strings.Contains(html, w) {
				t.Errorf("%s: missing %q:\n%s", tt.name, w, html)
			}
		}
	}

	fill := render(t, hi.HStack(hi.Spacer()))
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
		v     hi.View
		rigid bool
	}{
		{"definite width in a row", hi.HStack(hi.Text("x").Frame(hi.Width(200))), true},
		{"definite height in a column", hi.VStack(hi.Text("x").Frame(hi.Height(50))), true},
		{"auto width in a row", hi.HStack(hi.Text("x").Frame(hi.Height(50))), false},
		{"definite width in a grid cell", hi.ZStack(hi.Text("x").Frame(hi.Width(200))), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			if got := strings.Contains(html, "flex-shrink:0"); got != tt.rigid {
				t.Errorf("flex-shrink:0 = %v, want %v:\n%s", got, tt.rigid, html)
			}
		})
	}
}

// TestSubviewRigidIntersection pins rigidity's composition rule:
// a container is rigid on an axis exactly when every subview is rigid
// there, with an empty collection vacuously rigid.
func TestSubviewRigidIntersection(t *testing.T) {
	rigid := func() hi.View { return hi.Text("x").Frame(hi.Width(20)) }
	for _, tt := range []struct {
		name     string
		subviews []hi.View
		rigid    bool
	}{
		{"empty", nil, true},
		{"all rigid", []hi.View{rigid(), rigid()}, true},
		{"one flexible", []hi.View{rigid(), hi.Text("x")}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			inner := hi.HStack(tt.subviews...).Class("inner")
			html := render(t, hi.HStack(inner))
			rule := classRule(t, html, `<ui-hstack class="inner (ui-\w+)"`)
			if got := strings.Contains(rule, "flex-shrink:0"); got != tt.rigid {
				t.Errorf("inner rigidity = %v, want %v; rule %q:\n%s", got, tt.rigid, rule, html)
			}
		})
	}
}

// TestFrameOptionOverride pins the frame's option resolution: options apply
// in order, and a later option for the same setting replaces an earlier one.
func TestFrameOptionOverride(t *testing.T) {
	for _, tt := range []struct {
		name    string
		v       hi.View
		wants   []string
		rejects []string
	}{
		{
			"later width overrides an earlier one",
			hi.Text("x").Frame(hi.Width(50), hi.Width(60)),
			[]string{"width:60px"},
			[]string{"width:50px"},
		},
		{
			"axes are independent",
			hi.Text("x").Frame(hi.Width(50), hi.Height(60)),
			[]string{"width:50px", "height:60px"},
			nil,
		},
		{
			"omitted dimensions emit nothing",
			hi.Text("x").Frame(),
			nil,
			[]string{"width", "height", "auto"},
		},
		{
			"zero dimensions remain explicit",
			hi.Text("x").Frame(hi.Width(50), hi.Width(0), hi.Height(0)),
			[]string{"width:0", "height:0"},
			[]string{"width:50px"},
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
	html := render(t, hi.VStack(
		hi.VStack(hi.Text("a"), hi.Text("b")),
	).Gap(16))
	if got := strings.Count(html, "row-gap:16px"); got != 1 {
		t.Errorf("row-gap:16px declaration count = %d, want 1 (outer stack only)\n\n%s", got, html)
	}
	if got := strings.Count(html, "row-gap:8px"); got != 1 {
		t.Errorf("row-gap:8px declaration count = %d, want 1 (the inner stack's own default)\n\n%s", got, html)
	}
}

// TestRepeatedGapAlignment pins that Gap and Alignment compose like
// every other modifier: the innermost call wins, not the last one.
func TestRepeatedGapAlignment(t *testing.T) {
	tests := []struct {
		name    string
		view    hi.View
		tag     string
		want    string
		rejects string
	}{
		{
			name:    "gap",
			view:    hi.VStack(hi.Text("a")).Gap(4).Gap(12),
			tag:     "ui-vstack",
			want:    "gap:4px",
			rejects: "gap:12px",
		},
		{
			name:    "alignment",
			view:    hi.VStack(hi.Text("a")).Alignment(hi.Leading).Alignment(hi.Trailing),
			tag:     "ui-vstack",
			want:    "align-items:start",
			rejects: "align-items:end",
		},
		{
			name:    "zstack alignment",
			view:    hi.ZStack(hi.Text("a")).Alignment(hi.TopLeading).Alignment(hi.BottomTrailing),
			tag:     "ui-zstack",
			want:    "justify-items:start",
			rejects: "justify-items:end",
		},
		{
			name:    "grid gap",
			view:    hi.Grid(hi.Columns(2), hi.Text("a")).Gap(4).Gap(12),
			tag:     "ui-grid",
			want:    "gap:4px",
			rejects: "gap:12px",
		},
		{
			name:    "grid alignment",
			view:    hi.Grid(hi.Columns(2), hi.Text("a")).Alignment(hi.TopLeading).Alignment(hi.BottomTrailing),
			tag:     "ui-grid",
			want:    "justify-items:start",
			rejects: "justify-items:end",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.view)
			rule := classRule(t, html, `<`+tt.tag+` class="(ui-\w+)"`)
			if !strings.Contains(rule, tt.want) {
				t.Errorf("rule = %q, want %q", rule, tt.want)
			}
			if strings.Contains(rule, tt.rejects) {
				t.Errorf("rule = %q, must not contain %q", rule, tt.rejects)
			}
		})
	}
}

// TestEmptyControlFlow checks that Empty and a false branch contribute no
// markup, even with modifiers applied.
func TestEmptyControlFlow(t *testing.T) {
	html := render(t, hi.VStack(
		hi.Text("shown"),
		hi.Empty(),
		hi.Empty().Class("hidden-class").Padding(hi.Edges(8)),
		hi.If(false, hi.Text("hidden-if")),
		hi.When(false, func() hi.View { return hi.Text("hidden-when") }),
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
	paddedThenBg := render(t, hi.Text("x").Padding(hi.Edges(8)).Underlay(hi.Center, hi.OKLCH(0.9, 0, 0)))
	bgThenPadded := render(t, hi.Text("x").Underlay(hi.Center, hi.OKLCH(0.9, 0, 0)).Padding(hi.Edges(8)))
	if paddedThenBg == bgThenPadded {
		t.Errorf("modifier order should change the lowering, but both rendered identically:\n%s", paddedThenBg)
	}
}

// TestStateModifiers pins the state lowering: each state set present
// declares its differing properties under the matching pseudo-classes,
// with hover variants gated to devices that can hover.
func TestStateModifiers(t *testing.T) {
	hovered := render(t, hi.Text("x").WhileHovered(hi.Foreground(hi.Blue)))
	if got := classRule(t, hovered, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "@media (hover: hover){&:hover{color:"+blueCSS+"}}") {
		t.Errorf("Hovered rule = %q, want a hover-gated color variant:\n%s", got, hovered)
	}
	focused := render(t, hi.Text("x").WhileFocused(hi.Foreground(hi.Blue)))
	if got := classRule(t, focused, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "&:focus-visible{color:"+blueCSS+"}") {
		t.Errorf("Focused rule = %q, want a focus variant:\n%s", got, focused)
	}
	// A combination applies only while every given state is active,
	// regardless of the order or repetition of the states.
	both := render(t, hi.Text("x").Modify(hi.Background(hi.OKLCH(0.9, 0, 0)), hi.Pressed, hi.Hovered, hi.Pressed))
	if got := classRule(t, both, `<ui-text class="(ui-\w+)"`); !strings.Contains(got, "@media (hover: hover){&:hover:active{background-color:oklch(0.9 0 0)}}") {
		t.Errorf("combined rule = %q, want a hover+active variant:\n%s", got, both)
	}
	blue := hi.Foreground(hi.Blue)
	for _, tc := range []struct {
		v    hi.View
		want string
	}{
		{hi.Text("x").WhileDisabled(blue), `&:is(:disabled, [aria-disabled="true"]){color:` + blueCSS + `}`},
		{hi.Text("x").WhileChecked(blue), "&:checked{color:" + blueCSS + "}"},
		{hi.Text("x").WhileInvalid(blue), "&:user-invalid{color:" + blueCSS + "}"},
		{hi.Text("x").WhilePlaceholder(blue), "&:placeholder-shown{color:" + blueCSS + "}"},
		{hi.Text("x").Modify(blue, hi.Disabled, hi.Checked), `&:is(:disabled, [aria-disabled="true"]):checked{color:` + blueCSS + `}`},
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
	html := render(t, hi.Text("x").WhileHovered(hi.Foreground(hi.Blue)).Foreground(hi.Red))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	for _, w := range []string{"color:" + redCSS, "@media (hover: hover){&:hover{color:" + blueCSS + "}}"} {
		if !strings.Contains(got, w) {
			t.Errorf("rule = %q, missing %q:\n%s", got, w, html)
		}
	}
}

// TestStateBackgroundStacking pins the DOM-48 layering design: each
// state's rule declares its complete background list, with a
// state-scoped layer slotted at its chain position.
func TestStateBackgroundStacking(t *testing.T) {
	html := render(t, hi.Text("x").
		Background(hi.OKLCH(0.1, 0, 0)).WhileHovered(

		hi.Background(hi.OKLCH(0.2, 0, 0))).
		Background(hi.OKLCH(0.3, 0, 0)))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	for _, w := range []string{
		"background-color:oklch(0.3 0 0);background-image:linear-gradient(oklch(0.1 0 0),oklch(0.1 0 0))",
		"@media (hover: hover){&:hover{background-image:linear-gradient(oklch(0.1 0 0),oklch(0.1 0 0)),linear-gradient(oklch(0.2 0 0),oklch(0.2 0 0))}}",
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
	html := render(t, hi.Text("x").WhileFocused(hi.BorderStroke(2, hi.Blue)))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	for _, w := range []string{
		`&::after{border-radius:inherit;content:"";inset:0;pointer-events:none;position:absolute}`,
		"&:focus-visible::after{box-shadow:inset 0 0 0 2px " + blueCSS + "}",
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
	html := render(t, hi.Text("x").
		Background(hi.OKLCH(0.1, 0, 0)).WhileHovered(

		hi.Background(hi.OKLCH(0.2, 0, 0))).WhilePressed(

		hi.Background(hi.OKLCH(0.4, 0, 0))).
		WhileFocused(

			hi.Background(hi.OKLCH(0.5, 0, 0))))
	got := classRule(t, html, `<ui-text class="(ui-\w+)"`)
	for _, w := range []string{
		"background-color:oklch(0.1 0 0)",
		"&:active{background-color:oklch(0.4 0 0);background-image:linear-gradient(oklch(0.1 0 0),oklch(0.1 0 0))}",
		"&:hover{background-color:oklch(0.2 0 0);background-image:linear-gradient(oklch(0.1 0 0),oklch(0.1 0 0))}",
		"&:hover:active{background-color:oklch(0.4 0 0);background-image:linear-gradient(oklch(0.1 0 0),oklch(0.1 0 0)),linear-gradient(oklch(0.2 0 0),oklch(0.2 0 0))}",
		"&:hover:focus-visible:active{background-color:oklch(0.5 0 0);background-image:linear-gradient(oklch(0.1 0 0),oklch(0.1 0 0)),linear-gradient(oklch(0.2 0 0),oklch(0.2 0 0)),linear-gradient(oklch(0.4 0 0),oklch(0.4 0 0))}",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("rule = %q, missing %q:\n%s", got, w, html)
		}
	}
}

// TestStateNoChangeEmitsNothing pins the diffing: a state variant
// equal to the base paint declares nothing.
func TestStateNoChangeEmitsNothing(t *testing.T) {
	plain := render(t, hi.Text("x").Foreground(hi.Red))
	same := render(t, hi.Text("x").WhileHovered(hi.Foreground(hi.Red)).Foreground(hi.Red))
	if plain != same {
		t.Errorf("no-op state variant changed the rendering:\nplain:\n%s\nwith state:\n%s", plain, same)
	}
}

// TestStateUnionRestoresBase pins the subset override: when one state
// sets a property and another restores it to the base value, the
// union variant must redeclare the base value, or the single-state
// variant would still win while both states are active.
func TestStateUnionRestoresBase(t *testing.T) {
	got := render(t, hi.Text("x").WhilePressed(
		hi.Foreground(hi.Red)).
		WhileHovered(

			hi.Foreground(hi.Blue)).
		Foreground(hi.Red))
	rule := classRule(t, got, `<ui-text class="(ui-\w+)"`)
	if !strings.Contains(rule, "&:hover:active{color:"+redCSS+"}") {
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
	over := render(t, hi.Text("x").OverlayAt(hi.TopTrailing, hi.Center, hi.Text("o").Class("probe")))
	if got := classRule(t, over, `<ui-overlay class="(ui-\w+)"`); !strings.Contains(got, "align-items:start") || !strings.Contains(got, "justify-items:end") {
		t.Errorf("overlay placement should follow at, got %q:\n%s", got, over)
	}
	if got := classRule(t, over, `<ui-text class="probe (ui-\w+)"`); !strings.Contains(got, "translate:50% -50%") {
		t.Errorf("overlay view should shift its anchor onto at, got %q:\n%s", got, over)
	}
	// elm-ui's below: the underlay hangs off the base's bottom edge.
	under := render(t, hi.Text("x").UnderlayAt(hi.Bottom, hi.Top, hi.Text("u").Class("probe")))
	if got := classRule(t, under, `<ui-text class="probe (ui-\w+)"`); !strings.Contains(got, "translate:0% 100%") {
		t.Errorf("underlay view should shift its anchor onto at, got %q:\n%s", got, under)
	}
	// Coincident points shift nothing, matching Overlay's lowering.
	same := render(t, hi.Text("x").OverlayAt(hi.TopTrailing, hi.TopTrailing, hi.Text("o").Class("probe")))
	if strings.Contains(same, "translate") {
		t.Errorf("coincident points should not shift the overlay view:\n%s", same)
	}
}

// TestOverlayPageLowering pins the narrow root specialization: the base keeps
// the page-root lowering path, while the overlay becomes a fixed sibling. A
// ScrollView base can therefore retain document scrolling, and chained
// overlays become ordered fixed siblings. A modifier belonging to the layer
// composite, a wrapper, or an Underlay retains the ordinary layer box.
func TestOverlayPageLowering(t *testing.T) {
	plain := render(t, hi.Text("base").Overlay(hi.Center, hi.Text("overlay")))
	if strings.Contains(plain, "<ui-layer ") {
		t.Errorf("root Overlay retained its composite wrapper:\n%s", plain)
	}
	if strings.Count(plain, "<ui-overlay ") != 1 {
		t.Errorf("root Overlay should emit one overlay sibling:\n%s", plain)
	}
	if got := classRule(t, plain, `<ui-overlay class="(ui-\w+)"`); !strings.Contains(got, "position:fixed") || !strings.Contains(got, "pointer-events:none") || !strings.Contains(got, "z-index:2") {
		t.Errorf("root overlay rule = %q, want a fixed hit-transparent front layer", got)
	}
	if got := classRule(t, plain, `<ui-text class="(ui-\w+)">base`); !strings.Contains(got, "isolation:isolate") {
		t.Errorf("root overlay base rule = %q, want stacking isolation", got)
	}
	baseModified := render(t, hi.Text("base").Background(hi.Red).
		Overlay(hi.Center, hi.Text("overlay")))
	if strings.Contains(baseModified, "<ui-layer ") {
		t.Errorf("a modifier owned by the base should preserve root Overlay lowering:\n%s", baseModified)
	}
	if got := classRule(t, baseModified, `<ui-text class="(ui-\w+)">base`); !strings.Contains(got, "background-color:"+redCSS) {
		t.Errorf("base background rule = %q, want the modifier on the base", got)
	}

	scrolling := render(t, hi.ScrollView(hi.Vertical,
		hi.Text("content")).Overlay(hi.Top, hi.Text("toolbar")))
	if !pageRoot(scrolling, ` scroll="y"`) || strings.Contains(scrolling, "<ui-scroll ") || strings.Contains(scrolling, "<ui-layer ") {
		t.Errorf("Overlay should preserve its ScrollView base's document lowering:\n%s", scrolling)
	}
	if got := classRule(t, scrolling, `<ui-text class="(ui-\w+)">content`); !strings.Contains(got, "isolation:isolate") {
		t.Errorf("document ScrollView base rule = %q, want the root-carried isolation", got)
	}

	chained := render(t, hi.Text("base").
		Overlay(hi.Center, hi.Text("first")).
		Overlay(hi.Center, hi.Text("second")))
	if strings.Count(chained, "<ui-overlay ") != 2 || strings.Contains(chained, "<ui-layer ") {
		t.Errorf("chained root Overlays should emit two fixed siblings:\n%s", chained)
	}
	if first, second := strings.Index(chained, ">first<"), strings.Index(chained, ">second<"); first < 0 || second < first {
		t.Errorf("chained root Overlays should retain application order:\n%s", chained)
	}
	fallbacks := []struct {
		name string
		view hi.View
	}{
		{"composite background", hi.Text("base").Overlay(hi.Center, hi.Text("overlay")).Background(hi.Red)},
		{"wrapper", hi.Text("base").Overlay(hi.Center, hi.Text("overlay")).Padding(hi.Edges(0))},
		{"underlay", hi.Text("base").Underlay(hi.Center, hi.Text("underlay"))},
	}
	for _, tt := range fallbacks {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.view)
			if !strings.Contains(html, "<ui-layer ") {
				t.Errorf("%s should retain the ordinary layer composite:\n%s", tt.name, html)
			}
		})
	}
	scrollFallback := render(t, hi.ScrollView(hi.Vertical, hi.Text("base")).
		Overlay(hi.Center, hi.Text("overlay")).
		Background(hi.Red))
	if !strings.Contains(scrollFallback, "<ui-layer ") || !strings.Contains(scrollFallback, "<ui-scroll ") {
		t.Errorf("a modified Overlay composite should also keep its ScrollView base in element mode:\n%s", scrollFallback)
	}
}

// TestLayerArity pins the layer arity rule: multi-node overlay and
// underlay views are composed into a ZStack, while empty and unary
// views remain direct.
func TestLayerArity(t *testing.T) {
	for _, tt := range []struct {
		name  string
		layer func(hi.View) hi.View
	}{
		{"overlay", func(v hi.View) hi.View { return hi.Text("base").Overlay(hi.Center, v) }},
		{"underlay", func(v hi.View) hi.View { return hi.Text("base").Underlay(hi.Center, v) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, arity := range []struct {
				name  string
				view  hi.View
				stack bool
			}{
				{"empty", hi.Empty(), true},
				{"single", hi.Text("a"), false},
				{"multiple", hi.Group(hi.Text("a"), hi.Text("b")), true},
			} {
				t.Run(arity.name, func(t *testing.T) {
					html := render(t, tt.layer(arity.view))
					if got := strings.Contains(html, "<ui-zstack "); got != arity.stack {
						t.Errorf("ZStack present = %v, want %v:\n%s", got, arity.stack, html)
					}
				})
			}
		})
	}
}

// TestOverlayAtMovesGroupAsOne pins that two-point placement shifts
// the aggregate ZStack rather than each member of a multi-node layer.
func TestOverlayAtMovesGroupAsOne(t *testing.T) {
	html := render(t, hi.Text("base").OverlayAt(
		hi.TopTrailing,
		hi.Center,
		hi.Group(hi.Text("a"), hi.Text("b")),
	))
	if got := classRule(t, html, `<ui-zstack class="(ui-\w+)"`); !strings.Contains(got, "translate:50% -50%") {
		t.Errorf("overlay ZStack should shift its anchor onto at, got %q:\n%s", got, html)
	}
	for _, text := range []string{"a", "b"} {
		pattern := `<ui-text class="(ui-\w+)">` + text
		if got := classRule(t, html, pattern); strings.Contains(got, "translate:") {
			t.Errorf("overlay member %q should not shift independently, got %q:\n%s", text, got, html)
		}
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
	render(t, hi.Text("x").OverlayAt(hi.FirstBaseline, hi.Center, hi.Text("o")))
}

// TestRenderRootArity pins the root's arity rule: the viewport frames
// a single node, so non-unary views are arranged in a VStack.
func TestRenderRootArity(t *testing.T) {
	group := render(t, hi.Group(hi.Text("a"), hi.Text("b")))
	if !strings.Contains(group, "ui-vstack") {
		t.Errorf("a root Group should be wrapped in a VStack:\n%s", group)
	}
	single := render(t, hi.Text("a"))
	if strings.Contains(single, "ui-vstack") {
		t.Errorf("a single root view should not be wrapped:\n%s", single)
	}
	empty := render(t, hi.Empty())
	if !strings.Contains(empty, "ui-vstack") {
		t.Errorf("an empty root view should be wrapped in a VStack:\n%s", empty)
	}
}

// TestGrid pins the grid lowering: the layout becomes the column
// template, the grid carries the library's default gap and alignment,
// a filling subview stretches across its cell, and its fill request
// propagates to the grid.
func TestGrid(t *testing.T) {
	for _, tt := range []struct {
		name   string
		layout hi.GridLayout
		want   string
	}{
		{"Columns", hi.Columns(3), "grid-template-columns:repeat(3, minmax(0, 1fr))"},
		{"CellMinWidth", hi.ColumnMinWidth(120), "grid-template-columns:repeat(auto-fill, minmax(120px, 1fr))"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, hi.VStack(hi.Grid(tt.layout, hi.Red, hi.Text("x"))))
			grid := classRule(t, html, `<ui-grid class="(ui-\w+)"`)
			for _, w := range []string{"display:grid", tt.want, "row-gap:8px", "column-gap:8px", "align-items:center", "justify-items:center", "align-self:stretch"} {
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
	custom := classRule(t, render(t, hi.Grid(hi.Columns(2), hi.Text("x")).Gap(0).Alignment(hi.TopLeading)), `<ui-grid class="(ui-\w+)"`)
	for _, w := range []string{"row-gap:0", "column-gap:0", "align-items:start", "justify-items:start"} {
		if !strings.Contains(custom, w) {
			t.Errorf("grid rule missing %q: %q", w, custom)
		}
	}
}

// TestGridFill pins who asks for width: a Columns grid of non-filling
// subviews hugs them like a stack, while a CellMinWidth grid fills
// its available width regardless, since its column count depends on it.
func TestGridFill(t *testing.T) {
	hug := classRule(t, render(t, hi.VStack(hi.Grid(hi.Columns(2), hi.Text("x")))), `<ui-grid class="(ui-\w+)"`)
	if strings.Contains(hug, "align-self:stretch") {
		t.Errorf("Columns grid of non-filling subviews should hug them, got %q", hug)
	}
	fill := classRule(t, render(t, hi.VStack(hi.Grid(hi.ColumnMinWidth(100), hi.Text("x")))), `<ui-grid class="(ui-\w+)"`)
	if !strings.Contains(fill, "align-self:stretch") {
		t.Errorf("CellMinWidth grid should fill its width, got %q", fill)
	}
}

// TestFrameRatio pins the lowering: the ratio is declared, the
// anchor axis keeps the subview's fill, and the derived axis issues
// none, so no stretch can override the ratio.
func TestFrameRatio(t *testing.T) {
	wide := render(t, hi.VStack(hi.Red.FrameRatio(2, 3, hi.Horizontal)))
	rule := classRule(t, wide, `<ui-aspect class="(ui-\w+)"`)
	for _, w := range []string{"aspect-ratio:2 / 3", "align-self:stretch", "min-height:0"} {
		if !strings.Contains(rule, w) {
			t.Errorf("width-anchored rule missing %q: %q", w, rule)
		}
	}
	if strings.Contains(rule, "flex-grow") {
		t.Errorf("derived height should not fill the column, got %q", rule)
	}
	tall := render(t, hi.HStack(hi.Red.FrameRatio(2, 3, hi.Vertical)))
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
	half := render(t, hi.HStack(hi.Text("x").FrameRatio(2, 3, hi.Vertical)))
	if sub := classRule(t, half, `<ui-text class="(ui-\w+)"`); strings.Contains(sub, "stretch") {
		t.Errorf("non-filling subview should not stretch in the rotated frame, got %q", sub)
	}
}

// TestFrameRatioAlignment pins placement in the ratio frame: the
// alignment lowers directly for a horizontal anchor and on swapped
// axes for the rotated vertical anchor, which also keeps Leading on
// the leading edge in a right-to-left document.
func TestFrameRatioAlignment(t *testing.T) {
	wide := classRule(t, render(t, hi.Text("x").FrameRatio(2, 3, hi.Horizontal, hi.BottomLeading)), `<ui-aspect class="(ui-\w+)"`)
	if !strings.Contains(wide, "align-items:end") || !strings.Contains(wide, "justify-items:start") {
		t.Errorf("width-anchored rule should place bottom leading, got %q", wide)
	}
	tall := classRule(t, render(t, hi.Text("x").FrameRatio(2, 3, hi.Vertical, hi.BottomLeading)), `<ui-aspect class="(ui-\w+)"`)
	for _, w := range []string{"align-items:start", "justify-items:end", "&:dir(rtl){writing-mode:vertical-rl}"} {
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
		v    hi.View
	}{
		{"Frame", hi.Text("x").Frame(hi.Width(100), hi.FirstBaselineTrailing)},
		{"FrameBounds", hi.Text("x").FrameBounds(hi.MinWidth(100), hi.FirstBaselineTrailing)},
		{"FrameRatio", hi.Text("x").FrameRatio(1, 1, hi.Horizontal, hi.FirstBaselineTrailing)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, tt.v)
			if !strings.Contains(html, "align-items:start") || !strings.Contains(html, "justify-items:end") || strings.Contains(html, "baseline") {
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
		{"zero ratio", func() { hi.Text("x").FrameRatio(0, 1, hi.Horizontal) }},
		{"both axes", func() { hi.Text("x").FrameRatio(1, 1, hi.Horizontal|hi.Vertical) }},
		{"no axis", func() { hi.Text("x").FrameRatio(1, 1, 0) }},
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
		{"Columns(0)", func() { hi.Columns(0) }},
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

// TestRenderTitle verifies how a page title flows out of a view tree:
// the innermost title wins, then the first of several siblings,
// a title applied to a group lands on each member,
// and wrappers and layers pass a title through.
func TestRenderTitle(t *testing.T) {
	for _, tt := range []struct {
		name string
		view hi.View
		want string
	}{
		{"none", hi.Text("a"), ""},
		{"leaf", hi.Text("a").Title("x"), "x"},
		{"inner wins", hi.VStack(hi.Text("a").Title("x")).Title("y"), "x"},
		{"outer fills empty", hi.VStack(hi.Text("a")).Title("y"), "y"},
		{"first sibling wins", hi.VStack(hi.Text("a").Title("x"), hi.Text("b").Title("z")), "x"},
		{"empty sibling skipped", hi.VStack(hi.Text("a"), hi.Text("b").Title("x")), "x"},
		{"group", hi.Group(hi.Text("a").Title("x"), hi.Text("b")), "x"},
		{"group outer yields to first member", hi.Group(hi.Text("a").Title("x"), hi.Text("b")).Title("d"), "x"},
		{"group outer lands on first member", hi.Group(hi.Text("a"), hi.Text("b").Title("x")).Title("d"), "d"},
		{"group outer fills empty", hi.Group(hi.Text("a"), hi.Text("b")).Title("d"), "d"},
		{"for", hi.For([]string{"a", "b"}, nil, func(s string) hi.View { return hi.Text(s).Title(s) }).Title("d"), "a"},
		{"keyed for", hi.For([]string{"a", "b"}, func(s string) string { return s }, func(s string) hi.View { return hi.Text(s).Title(s) }).Title("d"), "a"},
		{"through padding", hi.Text("a").Title("x").Padding(hi.Edges(8)), "x"},
		{"through frame", hi.Text("a").Title("x").Frame(hi.Width(40)), "x"},
		{"through paint", hi.Text("a").Title("x").Background(hi.Accent).Opacity(0.5), "x"},
		{"through scroll", hi.ScrollView(hi.Vertical, hi.Text("a").Title("x")), "x"},
		{"from overlay", hi.Text("a").Overlay(hi.Center, hi.Text("b").Title("x")), "x"},
		{"base wins over overlay", hi.Text("a").Title("x").Overlay(hi.Center, hi.Text("b").Title("z")), "x"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := hi.Render(tt.view)
			if got != tt.want {
				t.Errorf("title = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderStyleElement verifies that the style element is always present,
// as the first child of ui-root, even for an empty view.
func TestRenderStyleElement(t *testing.T) {
	var sb strings.Builder
	_, page := hi.Render(hi.Empty())
	if err := domi.RenderTo(&sb, page); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := sb.String()
	m := regexp.MustCompile(`^<ui-root class="(ui-\w+)"><style>([^<]*)</style>`).FindStringSubmatch(html)
	if m == nil || !strings.Contains(m[2], "."+m[1]+"{") {
		t.Errorf("style element containing the root rule not first in ui-root:\n%s", html)
	}
}

// TestTextTrim pins the lowering of TextTrim: each edge trims to the
// smallest box named for it, an edge with no box named keeps its
// half-leading, and the empty set trims nothing.
func TestTextTrim(t *testing.T) {
	for _, tt := range []struct {
		edges hi.TextEdgeSet
		want  string
	}{
		{0, ""},
		{hi.TextTop, "text-box-edge:text text;text-box-trim:trim-start"},
		{hi.TextCap, "text-box-edge:cap text;text-box-trim:trim-start"},
		{hi.TextEx, "text-box-edge:ex text;text-box-trim:trim-start"},
		{hi.TextLastBaseline, "text-box-edge:text alphabetic;text-box-trim:trim-end"},
		{hi.TextBottom, "text-box-edge:text text;text-box-trim:trim-end"},
		{hi.TextCap | hi.TextLastBaseline, "text-box-edge:cap alphabetic;text-box-trim:trim-both"},
		{hi.TextTop | hi.TextCap | hi.TextEx, "text-box-edge:ex text;text-box-trim:trim-start"},
		{hi.TextLastBaseline | hi.TextBottom, "text-box-edge:text alphabetic;text-box-trim:trim-end"},
	} {
		html := render(t, hi.Text("x").TextTrim(tt.edges))
		rule := classRule(t, html, `<ui-text class="(ui-\w+)"`)
		got := strings.Join(regexp.MustCompile(`text-box-[a-z]+:[^;]*`).FindAllString(rule, -1), ";")
		if got != tt.want {
			t.Errorf("edges %b: trim = %q, want %q", tt.edges, got, tt.want)
		}
	}

	subtree := render(t, hi.VStack(hi.Text("a"), hi.Text("b").TextTrim(hi.TextEx)).TextTrim(hi.TextCap))
	for want, n := range map[string]int{"text-box-edge:cap text": 1, "text-box-edge:ex text": 1} {
		if got := strings.Count(subtree, want); got != n {
			t.Errorf("%s count = %d, want %d:\n%s", want, got, n, subtree)
		}
	}
}
