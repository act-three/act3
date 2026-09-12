// Package fixture builds the demo page for the hi test suite.
// The page exercises every component and layout mechanism once,
// so it doubles as the golden-test corpus
// and as the scene for browser geometry tests.
package fixture

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"

	"ily.dev/domi"
	"ily.dev/domi/attr"

	. "ily.dev/act3/hi"
)

type Msg struct {
	Edit    bool
	New     bool
	Watched uint64
	Select  uint64
}

type User struct {
	Name, Email, PhotoURL string
}

type Movie struct {
	ID                        uint64
	Title, Summary, PosterURL string
}

// placeholderImage is a data: URI for a solid-color SVG of the given size, so
// the demo page renders hermetically: no DNS, no third-party hosts, and the
// page load event never waits on the network.
func placeholderImage(w, h int, color string) string {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><rect width="100%%" height="100%%" fill="%s"/></svg>`, w, h, color)
	return "data:image/svg+xml," + url.PathEscape(svg)
}

func accountCard(user User) View {
	return Card(
		HStack(
			Image(user.PhotoURL).
				Alt(user.Name).
				ScaledToFill().
				BorderShape(Ellipse).
				Frame(Width(48), Height(48)),
			VStack(
				Text(user.Name).
					Font(SemiBold, SizeEm(18i, 1.4)),
				Text(user.Email).
					Foreground(Secondary),
			).
				Gap(4).
				Alignment(Leading),
			Spacer(),
			Button(Msg{Edit: true}, Text("Edit")),
		).
			Gap(12).
			Padding(Edges(16)),
	).
		OverlayAt(TopTrailing, Center, Badge("1"))
}

func moviePage(movies []Movie) View {
	return VStack(
		HStack(
			Text("Movies").
				Font(Bold, SizeEm(24i, 1.2)),
			Spacer(),
			Button("/movies/trash", Text("Trash")),
			Button(Msg{New: true}, Text("New")).
				ButtonStyle(Prominent),
		),

		VStack(ForEach(
			movies,
			func(m Movie) string { return strconv.FormatUint(m.ID, 10) },
			movieRow,
		)),
	).
		Gap(16)
}

func movieRow(movie Movie) View {
	return HStack(
		Image(movie.PosterURL).
			Alt(movie.Title).
			ScaledToFill().
			BorderShape(RoundedRectangle).
			Frame(Width(56), Height(84)),
		VStack(
			Text(movie.Title).
				Font(SemiBold, SizeEm(18i, 1.4)),
			Text(movie.Summary).
				Foreground(Secondary).
				Font(Normal, SizeEm(12i, 1.3)),
		).
			Gap(4).
			Alignment(Leading),
		Spacer(),
		Button(Msg{Watched: movie.ID}, Text("Watched")),
	).
		Gap(12).
		Padding(Edges(12))
}

func dividerRow() View {
	return HStack(
		Text("Edit"),
		Divider(),
		Text("Duplicate"),
		Divider(),
		Text("Delete").
			Foreground(Red),
	).
		Gap(12).
		Padding(Edges(12))
}

func dividerColumn() View {
	return VStack(
		Text("Profile").
			Font(SemiBold, SizeEm(18i, 1.4)),
		Divider(),
		Text("Account").
			Foreground(Secondary),
		Divider(),
		Text("Billing").
			Foreground(Secondary),
	).
		Alignment(Leading).
		Padding(Edges(12)).
		Frame(Width(220))
}

func zstackDemo() View {
	return ZStack(
		OKLCH(0.359, 0.135, 279).
			Frame(Width(120), Height(120)).
			BorderShape(RoundedRectangle),
		Text("ZStack with Long Text").
			TextForeground(White).
			Font(SemiBold, SizeEm(18i, 1.4)),
	).Overlay(
		BottomTrailing,
		Text("layered").
			TextForeground(OKLCH(0.87, 0.062, 274)).
			Font(Normal, SizeEm(12i, 1.3)),
	).
		Padding(Edges(8)).
		Background(OKLCH(0.627, 0, 0))
}

func count(n int) []int {
	a := make([]int, n)
	for i := range n {
		a[i] = i + 1
	}
	return a
}

func scrollDemo() View {
	return Card(ScrollView(Vertical,
		VStack(ForEach(count(3), nil, func(s int) View {
			return VStack(
				Text("Season "+strconv.Itoa(s)).
					Font(SemiBold, SizeEm(18i, 1.4)).
					Padding(Edges(8)).
					Background(OKLCH(0.93, 0.033, 273)).
					Sticky(),
				ForEach(count(6), nil, func(e int) View {
					return Text(fmt.Sprintf("Episode %d.%d", s, e)).
						Padding(Edges(8))
				}),
			).
				Gap(0).
				Alignment(Leading)
		})).
			Alignment(Leading),
	)).
		Frame(Height(160), Width(220))
}

func richText() View {
	return Text("Status: ").
		TextFont(Bold).
		Concat(Text("Draft ").TextFont(Italic)).
		Concat(Text("v2").TextFont(Family("var(--hi-font-mono)"))).
		TextForeground(Secondary)
}

func linkText() View {
	return VStack(
		Text("Read the ").
			Concat(Link("/docs", Text("documentation"))).
			Concat(Text(", or ")).
			Concat(Link(Msg{New: true}, Text("start a new movie"))).
			Concat(Text(" right here.")),
		Text("You can't ").
			Concat(Link("/admin", Text("administer"))).
			Concat(Text(" anything, though.")).
			Disabled(true),
	).
		Alignment(Leading)
}

func lineLimitDemo() View {
	soliloquy := Text("To be, or not to be, that is the question: " +
		"whether 'tis nobler in the mind to suffer " +
		"the slings and arrows of outrageous fortune, " +
		"or to take arms against a sea of troubles " +
		"and by opposing end them.")
	return VStack(
		soliloquy.LineLimit(2),
		soliloquy.LineLimit(1),
	).
		Gap(8).
		Frame(Width(260))
}

func textLayout() View {
	var t = Text("To be, or not to be, that is the question.")
	return VStack(
		Card(t.Frame(Width(100), Height(100))),
		Card(t.FixedSize().Frame(Width(100), Height(100))),
	)
}

func fontSizeDemo() View {
	sample := Text("Handgloves 0123456789")
	return VStack(
		section("Cap height: 16 scaled pixels; line height: 32 scaled pixels",
			sample.
				Font(SizeCap(16i, 2)).
				Background(Blue),
		),
		section("Em square: 16 scaled pixels; line height: 32 scaled pixels",
			sample.
				Font(SizeEm(16i, 2)).
				Background(Blue),
		),
	).
		Alignment(Leading)
}

func textTrimDemo() View {
	t := Text("Hxflg").
		Font(SemiBold, SizeEm(18i, 1.4)).
		Background(OKLCH(0.9, 0.05, 70))
	return HStack(
		t,
		t.TextTrim(TextTop|TextBottom),
		t.TextTrim(TextCap|TextLastBaseline),
		t.TextTrim(TextEx|TextLastBaseline),
	).
		Alignment(FirstBaseline)
}

func iconDemo() View {
	return VStack(
		ForEach([][]FontOption{
			{Normal, SizeEm(12i, 1.3)},
			{Normal, SizeEm(16i, 1.4)},
			{Bold, SizeEm(24i, 1.2)},
		}, nil, func(opts []FontOption) View {
			return HStack(
				Icon("film"),
				Text("A long paragraph that wraps onto a second line to show the icon aligned with the first."),
			).
				Alignment(FirstBaseline).
				Font(opts...)
		}),
		HStack(
			ForEach([]complex128{20, 30, 40}, nil, func(n complex128) View {
				return HStack(
					Icon("film"),
					Yellow.
						Frame(Width(n), Height(n)),
				)
			}),
		),
	).
		Alignment(Leading).
		Frame(Width(360))
}

func stateDemo() View {
	return Text("Hover, focus, or press me").
		TextForeground(White).
		Padding(EdgesLetterbox(8), EdgesPillarbox(12)).
		WhilePressed(Background(OKLCH(0.491, 0.241, 293))).
		WhileHovered(Background(OKLCH(0.457, 0.215, 277))).
		Background(OKLCH(0.359, 0.135, 279)).
		WhileFocused(BorderStroke(2, OKLCH(0.769, 0.165, 70))).
		BorderShape(RoundedRectangle).
		Attr(attr.TabIndex("0"))
}

func gridDemo() View {
	var cells []View
	for i, c := range []Color{
		OKLCH(0.769, 0.165, 70), OKLCH(0.685, 0.148, 237), OKLCH(0.648, 0.175, 132),
		OKLCH(0.586, 0.222, 18), OKLCH(0.606, 0.219, 293), OKLCH(0.704, 0.123, 183),
		OKLCH(0.705, 0.187, 48), OKLCH(0.656, 0.212, 354), OKLCH(0.623, 0.188, 260),
		OKLCH(0.768, 0.204, 131), OKLCH(0.627, 0.233, 304), OKLCH(0.715, 0.126, 215),
		OKLCH(0.637, 0.208, 25),
	} {
		cells = append(
			cells,
			ZStack(
				c.
					Frame(Height(48)),
				Text(strconv.Itoa(i+1)).
					TextForeground(White),
			).
				BorderClipped().
				BorderShape(RoundedRectangle),
		)
	}
	return VStack(
		Grid(Columns(4), cells...),
		Grid(ColumnMinWidth(120), cells...).
			Gap(4),
	).
		Gap(16)
}

func posterWall() View {
	var posters []View
	for _, c := range []string{"#f59e0b", "#0ea5e9", "#65a30d", "#e11d48", "#8b5cf6", "#14b8a6", "#f97316", "#ec4899"} {
		posters = append(
			posters,
			Image(placeholderImage(120, 180, c)).
				ScaledToFill().
				FrameRatio(2, 3, Horizontal).
				BorderClipped().
				BorderShape(RoundedRectangle),
		)
	}
	return Grid(Columns(6), posters...)
}

func section(title string, body View) View {
	return VStack(
		Text(title).
			TextForeground(Secondary).
			TextFont(Normal, SizeEm(12i, 1.3)),
		body,
	).
		Alignment(Leading)
}

// page has one section per component or layout mechanism.
func page() View {
	user := User{Name: "Ada Lovelace", Email: "ada@example.com", PhotoURL: placeholderImage(96, 96, "#818cf8")}
	movies := []Movie{
		{ID: 1, Title: "Metropolis", Summary: "A city divided between thinkers and workers.", PosterURL: placeholderImage(120, 180, "#f59e0b")},
		{ID: 2, Title: "Solaris", Summary: "An ocean world that answers back.", PosterURL: placeholderImage(120, 180, "#0ea5e9")},
		{ID: 3, Title: "Stalker", Summary: "A guide leads two men into the Zone.", PosterURL: placeholderImage(120, 180, "#65a30d")},
	}
	return VStack(
		Text("hi component library").
			Title("hi component library").
			Font(Bold, SizeEm(32i, 1.15)),
		section("Account card (Card + HStack + Spacer + OverlayAt badge)", accountCard(user)),
		section("Movie page (Frame fill + keyed rows + ForEach-style list)", moviePage(movies)),
		buttonSection("Buttons", buttonGallery()),
		section("Dividers in an HStack (minor-axis, vertical)", Card(dividerRow())),
		section("Dividers in a VStack (minor-axis, horizontal)", Card(dividerColumn())),
		section("ZStack (layered, all subviews size the stack)", zstackDemo()),
		section("ScrollView (contained in a Frame, Sticky season headings)", scrollDemo()),
		section("Rich text (per-run bold/italic/mono, whole-text color)", richText()),
		section("Links in text (navigate, send, then a disabled line)", linkText()),
		section("LineLimit (2 lines, then 1)", lineLimitDemo()),
		section("Text Layout", textLayout()),
		section("Font size (cap height and em square)", fontSizeDemo()),
		section("TextTrim (none, top/bottom, cap/baseline, ex/baseline)", textTrimDemo()),
		section("Icon (FirstBaseline with wrapping text, at 12/16/24 scaled pixels)", iconDemo()),
		section("State modifiers (Hovered / Focused / Pressed)", stateDemo()),
		section("Grid (Columns(4), then CellMinWidth(120))", gridDemo()),
		section("FrameRatio (2:3 posters anchored on width, in Columns(6))", posterWall()),
	).
		Gap(32).
		Alignment(Leading).
		Padding(Edges(32))
}

// Document renders the page into a standalone document with the hi stylesheet
// inlined, suitable for a browser or the golden test.
func Document(css string) (string, error) {
	title, page := Render(
		ScrollView(Vertical, page()),
		Theme(OKLCH(0.9561, 0.0074, 80.7), OKLCH(0.782, 0.1404, 70.8), 25),
	)
	var sb strings.Builder
	sb.WriteString("<!doctype html><html lang=en><head><meta charset=utf-8>")
	sb.WriteString("<meta name=viewport content=\"width=device-width, initial-scale=1\">")
	sb.WriteString("<title>")
	sb.WriteString(html.EscapeString(title))
	sb.WriteString("</title><style>\n")
	sb.WriteString(css)
	sb.WriteString("\n</style></head><body>")
	if err := domi.RenderTo(&sb, page); err != nil {
		return "", err
	}
	sb.WriteString("</body></html>")
	return sb.String(), nil
}
