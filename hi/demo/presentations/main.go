// Command presentations serves a playground for Hi popovers and dialogs.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"

	. "ily.dev/act3/hi"
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:0", "HTTP listen address")
	dark := flag.Bool("dark", false, "use a dark theme")
	flag.Parse()
	background := OKLCH(0.96, 0.008, 90)
	surface := OKLCH(1, 0, 0)
	if *dark {
		background = OKLCH(0.2, 0.005, 280)
		surface = OKLCH(0.27, 0.005, 280)
	}
	handler := Handler(
		func(context.Context, *url.URL) (*app, domi.Cmd[message]) {
			return &app{surface: surface, lastAction: "No action yet"}, nil
		},
		func(*url.URL) message { return message{} },
		func(*url.URL) message { return message{} },
		AppTitle("Hi Presentations Demo"),
		Theme(background, OKLCH(0.6, 0.2, 280), 30),
	)
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
	fmt.Println("http://" + listener.Addr().String())
	if err := http.Serve(listener, handler); err != nil {
		slog.Error("serve demo", "err", err)
		os.Exit(1)
	}
}

type message struct {
	Kind  string
	Value string
}

type app struct {
	surface        Color
	menuOpen       bool
	longOpen       bool
	dialogOpen     bool
	fillDialogOpen bool
	helpOpen       bool
	keepDialog     bool
	dismissals     int
	lastAction     string
	draft          string
}

func (a *app) Update(_ context.Context, m message) domi.Cmd[message] {
	switch m.Kind {
	case "open-menu":
		a.menuOpen = true
	case "dismiss-menu":
		a.menuOpen = false
	case "open-long":
		a.longOpen = true
	case "dismiss-long":
		a.longOpen = false
	case "choose":
		a.lastAction = m.Value
		a.menuOpen, a.longOpen = false, false
	case "open-dialog":
		a.menuOpen = false
		a.dialogOpen = true
	case "dismiss-dialog":
		a.dismissals++
		if !a.keepDialog {
			a.dialogOpen = false
		}
	case "close-dialog":
		a.dialogOpen = false
	case "open-fill-dialog":
		a.fillDialogOpen = true
	case "close-fill-dialog":
		a.fillDialogOpen = false
	case "keep-dialog":
		a.keepDialog = !a.keepDialog
	case "open-help":
		a.helpOpen = true
	case "dismiss-help":
		a.helpOpen = false
	case "edit-draft":
		a.draft = m.Value
	}
	return nil
}

func (*app) Subscriptions(context.Context) domi.Sub[message] { return nil }

func (a *app) Preview(_ context.Context, u *url.URL, render PreviewRenderer) Preview {
	return render(u.Path, a.page())
}

func (a *app) View(_ context.Context, render PageRenderer) Page {
	return render(a.page())
}

func (a *app) page() View {
	content := VStack(
		VStack(
			Text("Hi Popovers & Dialogs").
				Font(SizeEmAbs(30, 38), Bold).
				Tag("h1"),
			Text("Open a surface, try its actions, and explore how it closes.").
				Foreground(Secondary),
		).
			Alignment(Leading),
		a.anchoredMenuSection(),
		a.centeredDialogSection(),
		a.scrollableMenuSection(),
		Text(a.lastAction).
			Font(Medium).
			Attr(attr.ID("last-action")),
		a.fillingDialogSection(),
		Text("Popovers close on Escape or an outside click. Dialogs close on Escape unless the application declines; clicking the backdrop leaves them open.").
			Foreground(Secondary),
		Text("Use Tab to explore focus. Open Help inside the dialog, then press Escape to dismiss only the help popover. Choose “Close Both” inside Help, then reopen the dialog: both return.").
			Foreground(Secondary),
		Spacer().
			Frame(Height(240)),
		Text("Scroll back up to try the episode menu near the top or bottom of the window.").
			Foreground(Secondary),
	).
		Alignment(Leading).
		Gap(24).
		Padding(Edges(24)).
		Attr(attr.Style("max-width:680px")).
		Tag("main")
	return ScrollView(Vertical, content)
}

func (a *app) section(title, description string, content View) View {
	return VStack(
		HStack(
			Text(title).
				Font(SizeEmAbs(20, 28), Bold).
				Tag("h2"),
			Spacer(),
		),
		Text(description).
			Foreground(Secondary),
		content,
	).
		Alignment(Leading).
		Gap(12).
		Padding(Edges(20))
}

func (a *app) anchoredMenuSection() View {
	return a.section("Anchored Menu", "This panel uses a contrasting theme. The menu inherits it and can extend beyond the clipped panel.",
		Button(message{Kind: "open-menu"}, Text("Movie Actions")).
			Selected(a.menuOpen).
			Attr(attr.ID("menu-trigger")).
			Menu(Present(a.menuOpen, message{Kind: "dismiss-menu"}),
				VStack(
					Button(message{Kind: "choose", Value: "Added to your watchlist"}, Text("Add to Watchlist")),
					Button(message{Kind: "choose", Value: "Marked as watched"}, Text("Mark as Watched")),
					Button(message{Kind: "open-dialog"}, Text("Edit a Note…")),
				).
					Alignment(Leading).
					Gap(10).
					Padding(Edges(8)).
					Title("Movie Actions").
					Attr(attr.ID("actions-menu")),
			),
	).
		ThemeBackground(ModeColor(OKLCH(0.22, 0.005, 280), OKLCH(0.94, 0.008, 90))).
		BorderShape(RoundedRectangle(12)).
		BorderClipped()
}

func (a *app) scrollableMenuSection() View {
	return a.section("Scrollable Menu", "The menu opens on the side with more height. Scroll the page while it is open: it follows the button and keeps its size. Reopen it to fit the new space.",
		Button(message{Kind: "open-long"}, Text("Choose an Episode")).
			Selected(a.longOpen).
			Attr(attr.ID("long-trigger")).
			Menu(Present(a.longOpen, message{Kind: "dismiss-long"}),
				ScrollView(Vertical,
					VStack(
						ForEach(count(40), nil, func(i int) View {
							label := fmt.Sprintf("Episode %02d", i)
							return Button(message{Kind: "choose", Value: "Selected " + label}, Text(label))
						}),
					).
						Alignment(Leading).
						Padding(Edges(8)),
				).
					Frame(Width(200)).
					Title("Choose an Episode").
					Attr(attr.ID("episode-menu")),
			),
	).
		ThemeBackground(a.surface).
		BorderShape(RoundedRectangle(12))
}

func count(n int) []int {
	a := make([]int, n)
	for i := range n {
		a[i] = i + 1
	}
	return a
}

func (a *app) centeredDialogSection() View {
	return a.section("Centered Dialog", "This panel also uses a contrasting theme, but the dialog uses the page's root theme. Your draft survives closing and reopening it.",
		Button(message{Kind: "open-dialog"}, Text("Open Dialog")).
			ButtonStyle(Prominent).
			Attr(attr.ID("dialog-trigger")).
			Dialog(Present(a.dialogOpen, message{Kind: "dismiss-dialog"}),
				VStack(
					HStack(
						Text("Movie Note").
							Font(SizeEmAbs(24, 32), Bold).
							Tag("h2"),
						Spacer(),
						Button(message{Kind: "close-dialog"}, Icon("x")).
							ButtonStyle(Subtle).
							ControlSize(Small),
					),
					Text("Your draft stays here when this dialog closes.").
						Foreground(Secondary),
					HTML(domi.Tag("label", attr.For("draft"))(domi.Text("Note"))).
						FixedSize(),
					HTML(domi.Tag("textarea",
						attr.ID("draft"),
						attr.Rows("4"),
						attr.Autofocus(true),
						attr.Placeholder("Write a note…"),
						event.Input(func(value string) message { return message{Kind: "edit-draft", Value: value} }),
						attr.Style("width:100%;box-sizing:border-box;font:inherit;padding:10px;border:1px solid currentColor;border-radius:6px;background:transparent;color:inherit;resize:vertical"),
					)(
						domi.Text(a.draft),
					)).
						FixedSize(),
					Button(message{Kind: "keep-dialog"}, Text("Keep Open on Escape")).
						Selected(a.keepDialog),
					Text(fmt.Sprintf("Dismissal requests: %d", a.dismissals)).
						Foreground(Secondary),
					HStack(
						Button(message{Kind: "open-help"}, Text("Help")).
							Selected(a.helpOpen).
							Attr(attr.ID("help-trigger")).
							Popover(Present(a.helpOpen, message{Kind: "dismiss-help"}),
								VStack(
									Text("A Popover inside a Dialog").
										Font(Bold),
									Text("Escape dismisses this popover first. The dialog remains open."),
									Button(message{Kind: "dismiss-help"}, Text("Got It")),
									Button(message{Kind: "close-dialog"}, Text("Close Both")),
								).
									Alignment(Leading).
									Gap(12).
									Padding(Edges(12)).
									Frame(Width(240)).
									Title("Help").
									Attr(attr.ID("help-popover")),
							),
						Spacer(),
						Button(message{Kind: "close-dialog"}, Text("Save Draft")).
							ButtonStyle(Prominent),
					),
				).
					Gap(14).
					Alignment(Leading).
					Frame(Width(340)).
					Title("Movie Note").
					Attr(attr.ID("note-dialog")),
			),
	).
		ThemeBackground(ModeColor(OKLCH(0.22, 0.005, 280), OKLCH(0.94, 0.008, 90))).
		BorderShape(RoundedRectangle(12))
}

func (a *app) fillingDialogSection() View {
	return a.section("Filling Dialog", "This dialog fills both axes, leaving 12px around the viewport edges. Resize the window to see it follow the available space.",
		Button(message{Kind: "open-fill-dialog"}, Text("Open Filling Dialog")).
			Dialog(Present(a.fillDialogOpen, message{Kind: "close-fill-dialog"}),
				VStack(
					HStack(
						Text("Room to Fill").
							Font(SizeEmAbs(24, 32), Bold).
							Tag("h2"),
						Spacer(),
					),
					Text("This dialog expands horizontally and vertically. The dimmed backdrop remains visible around all four edges.").
						Foreground(Secondary),
					Spacer(),
					HStack(
						Spacer(),
						Button(message{Kind: "close-fill-dialog"}, Text("Dismiss")).
							ButtonStyle(Prominent),
					),
				).
					Alignment(Leading).
					Title("Room to Fill").
					Attr(attr.ID("filling-dialog")),
			),
	).
		ThemeBackground(a.surface).
		BorderShape(RoundedRectangle(12))
}
