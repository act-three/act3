// Package fixture builds the demo page shared by the preview command
// and the xui test suite.
// The page exercises every component and layout mechanism once,
// so it doubles as the golden-test corpus
// and as the scene for browser geometry tests.
package fixture

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	. "ily.dev/act3/xui"
	"ily.dev/domi"
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
				FramedAs(ScaledToFill).
				Frame(Width(48), Height(48)).
				BorderShape(Ellipse),
			VStack(
				Text(user.Name).
					Font(Headline),
				Text(user.Email).
					Foreground(Muted),
			).
				Gap(4).
				Alignment(Leading),
			Spacer(),
			Button(Text("Edit"), Msg{Edit: true}).
				Role(RolePrimary),
		).
			Gap(12),
	).
		Padding(Edges(16)).
		LayerOver(
			TopTrailing,
			Badge("Pro").
				Padding(Edges(8)),
		)
}

func moviePage(movies []Movie) View {
	return VStack(
		HStack(
			Text("Movies").
				Font(Title),
			Spacer(),
			Button(Text("New"), Msg{New: true}).
				Role(RolePrimary),
		),

		VStack(For(
			movies,
			func(m Movie) string { return strconv.FormatUint(m.ID, 10) },
			movieRow,
		)),
	).
		Gap(16).
		Padding(Edges(24))
}

func movieRow(movie Movie) View {
	return HStack(
		Image(movie.PosterURL).
			Alt(movie.Title).
			FramedAs(ScaledToFill).
			Frame(Width(56), Height(84)).
			BorderShape(RoundedRectangle),
		VStack(
			Text(movie.Title).
				Font(Headline),
			Text(movie.Summary).
				Foreground(Muted).
				Font(Caption),
		).
			Gap(4).
			Alignment(Leading),
		Spacer(),
		Button(Text("Watched"), Msg{Watched: movie.ID}),
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
			Foreground(Danger),
	).
		Gap(12).
		Padding(Edges(12))
}

func dividerColumn() View {
	return VStack(
		Text("Profile").
			Font(Headline),
		Divider(),
		Text("Account").
			Foreground(Muted),
		Divider(),
		Text("Billing").
			Foreground(Muted),
	).
		Alignment(Leading).
		Padding(Edges(12)).
		Frame(Width(220))
}

func zstackDemo() View {
	return ZStack(
		CSSColor("#312e81").
			Frame(Width(120), Height(120)).
			BorderShape(RoundedRectangle),
		Text("ZStack with Long Text").
			TextForeground(CSSColor("#fff")).
			Font(Headline),
	).LayerOver(
		BottomTrailing,
		Text("layered").
			TextForeground(CSSColor("#c7d2fe")).
			Font(Caption),
	).
		Padding(Edges(8)).
		Background(CSSColor("#888"))
}

func scrollDemo() View {
	var rows []View
	for i := 1; i <= 20; i++ {
		rows = append(
			rows,
			Text("Episode "+strconv.Itoa(i)).
				Padding(Edges(8)),
		)
	}
	return ScrollView(
		Vertical,
		VStack(rows...).
			Alignment(Leading),
	).
		Frame(Height(160), Width(220)).
		BorderShape(RoundedRectangle).
		Class("demo-bordered")
}

func richText() View {
	return Text("Status: ").
		Bold().
		Concat(Text("Draft ").Italic()).
		Concat(Text("v2").Monospace()).
		TextForeground(Muted)
}

func textLayout() View {
	var t = Text("To be, or not to be, that is the question.")
	return VStack(
		Group(
			t,
			t.FixedSize(),
		).
			Frame(Width(100), Height(100)).
			Class("demo-bordered"),
	)
}

func section(title string, body View) View {
	return VStack(
		Text(title).
			TextForeground(Muted).
			TextFont(Caption),
		body,
	).
		Alignment(Leading)
}

// Page is the demo page: one section per component or layout mechanism.
func Page() View {
	user := User{Name: "Ada Lovelace", Email: "ada@example.com", PhotoURL: placeholderImage(96, 96, "#818cf8")}
	movies := []Movie{
		{ID: 1, Title: "Metropolis", Summary: "A city divided between thinkers and workers.", PosterURL: placeholderImage(120, 180, "#f59e0b")},
		{ID: 2, Title: "Solaris", Summary: "An ocean world that answers back.", PosterURL: placeholderImage(120, 180, "#0ea5e9")},
		{ID: 3, Title: "Stalker", Summary: "A guide leads two men into the Zone.", PosterURL: placeholderImage(120, 180, "#65a30d")},
	}
	return VStack(
		Text("ui component library").
			Font(LargeTitle),
		section("Account card (Card + HStack + Spacer + Overlay badge)", accountCard(user)),
		section("Movie page (Frame fill + keyed rows + For-style list)", moviePage(movies)),
		section("Dividers in an HStack (minor-axis, vertical)", dividerRow().Class("demo-bordered")),
		section("Dividers in a VStack (minor-axis, horizontal)", dividerColumn().Class("demo-bordered")),
		section("ZStack (layered, all subviews size the stack)", zstackDemo()),
		section("ScrollView (contained in a Frame)", scrollDemo()),
		section("Rich text (per-run bold/italic/mono, whole-text color)", richText()),
		section("Text Layout", textLayout()),
	).
		Gap(32).
		Alignment(Leading).
		Padding(Edges(32))
}

// Document renders Page into a standalone document with the xui stylesheet
// inlined, suitable for a browser or the golden test.
func Document() (string, error) {
	var sb strings.Builder
	sb.WriteString("<!doctype html><html lang=en><head><meta charset=utf-8>")
	sb.WriteString("<meta name=viewport content=\"width=device-width, initial-scale=1\"><style>\n")
	sb.WriteString(CSS)
	sb.WriteString("\nbody{margin:0;background:#f5f6f8}")
	sb.WriteString("\n.demo-bordered{border:1px solid var(--ui-color-border);border-radius:var(--ui-radius);background:#fff}")
	sb.WriteString("\n</style></head><body>")
	if err := domi.RenderTo(&sb, new(Renderer).Render(ScrollView(Vertical, Page()))); err != nil {
		return "", err
	}
	sb.WriteString("</body></html>")
	return sb.String(), nil
}
