// Command notes serves a playground for Hi notification interactions.
package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"ily.dev/domi"
	"ily.dev/domi/attr"

	. "ily.dev/act3/hi"
)

//go:embed client.js
var clientJS string

func main() {
	dark := flag.Bool("dark", false, "use a dark theme")
	contrast := flag.Float64("contrast", 30, "theme contrast (15–100)")
	flag.Parse()
	bg := OKLCH(0.955, 0.0083, 91.48)
	if *dark {
		bg = OKLCH(0.2, 0.005, 280)
	}
	cssDigest, cssHandler := Stylesheet()
	cssPath := "/hi." + cssDigest + ".css"
	handler := Handler(
		func(context.Context, *url.URL) (*app, domi.Cmd[message]) {
			return &app{}, RegisterNote[message]("client-note", Note{
				Message:     "A note from JavaScript",
				Description: "This template keeps its Go action.",
				Icon:        "zap",
				Action:      Button(message{Kind: "undo"}, Text("Undo")),
				Duration:    10 * time.Second,
			})
		},
		func(u *url.URL) message { return message{Kind: "request", Path: u.Path} },
		func(*url.URL) message { return message{} },
		AppTitle("Hi notes demo"),
		Theme(bg, OKLCH(0.6, 0.2, 280), *contrast),
		domi.InternalURLPrefix("/-/domi"),
		domi.Document(func(title string, body domi.Node) domi.Node {
			return domi.Tag("html")(
				domi.Tag("head")(
					domi.Tag("meta", attr.Charset("utf-8"))(),
					domi.Tag("meta", attr.Name("viewport"), attr.Content("width=device-width,initial-scale=1"))(),
					domi.Tag("title")(domi.Text(title)),
					domi.Tag("link", attr.Rel("stylesheet"), attr.Href(cssPath))(),
					ClientModule("/-/domi"),
					domi.Tag("script", attr.Type("module"))(domi.Text(clientJS)),
				),
				body,
			)
		}),
	)
	mux := http.NewServeMux()
	mux.Handle("GET "+cssPath, cssHandler)
	mux.Handle("/", handler)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
	fmt.Println("http://" + listener.Addr().String())
	if err := http.Serve(listener, mux); err != nil {
		slog.Error("serve demo", "err", err)
		os.Exit(1)
	}
}

type message struct {
	Kind      string
	Path      string
	Remaining int
	Variant   string
	Delay     time.Duration
}

type app struct {
	sent    int
	pending int
	undone  int
}

func (a *app) Update(ctx context.Context, m message) domi.Cmd[message] {
	switch m.Kind {
	case "request":
		return domi.PushURL[message](m.Path)
	case "undo":
		a.undone++
	case "queue":
		a.pending += m.Remaining
		m.Kind = "send"
		return after(ctx, m.Delay, m)
	case "send":
		a.pending--
		a.sent++
		text := fmt.Sprintf("Note %d · Saved.", a.sent)
		if m.Variant == "long" {
			text = fmt.Sprintf("Note %d · %sEnd of the long note.",
				a.sent, strings.Repeat("Your library is ready. New episodes have been added and are ready to watch. ", 7))
		} else if m.Variant == "mixed" && m.Remaining == 1 {
			text = fmt.Sprintf("Note %d · Your library is ready. This longer notification wraps across several lines so you can try expanding the stack, moving between notes of different heights, and swiping a taller note away. The other notes should move smoothly into place.", a.sent)
		} else if m.Variant == "mixed" && m.Remaining == 2 {
			text = fmt.Sprintf("Note %d · Three new episodes have been added to your library and are ready to watch.", a.sent)
		}
		n := Note{Message: text}
		switch m.Variant {
		case "description":
			n.Description = "New episodes have been added to your library."
		case "icon":
			n.Icon = "circle-check"
		case "rich":
			n.Icon = "circle-check"
			n.Description = "This note has an explicit icon."
		case "undo":
			n.Message = "Item deleted"
			n.Icon = "trash-2"
			n.Description = "You can restore it to your library."
			n.Action = Button(message{Kind: "undo"}, Text("Undo"))
			n.Duration = 10 * time.Second
		case "link":
			n.Icon = "circle-check"
			n.Description = "Follow the link to another page."
			n.Action = Link("/away", Text("Open"))
		case "short":
			n.Duration = time.Second
		case "long":
			n.Icon = "info"
			n.Description = strings.Repeat("Additional details about your library update. ", 12)
			n.Action = Link("/away", Text("Open"))
		}
		if m.Variant == "mixed" && m.Remaining == 1 {
			n.Description = "New episodes have been added and are ready to watch."
		}
		cmd := Notify[message](n)
		if m.Remaining > 1 {
			m.Remaining--
			return domi.Batch[message](cmd, after(ctx, 150*time.Millisecond, m))
		}
		return cmd
	}
	return nil
}

func after(ctx context.Context, delay time.Duration, m message) domi.Cmd[message] {
	return domi.Func(func() message {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return message{}
		case <-timer.C:
			return m
		}
	})
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
			Text("Hi notes").Font(SizeEmAbs(30, 38), Bold).Tag("h1"),
			Text("Appearance and interaction playground").Foreground(Secondary),
		).Alignment(Leading).Gap(8),
		Text("Send a few notes, then hover or focus the stack at the bottom of the screen. The latest three appear; dismiss one to reveal an older note that is still active."),
		VStack(
			Button(message{Kind: "queue", Remaining: 1}, Text("Add a note")),
			Button(message{Kind: "queue", Remaining: 1, Variant: "description"}, Text("With description")),
			Button(message{Kind: "queue", Remaining: 1, Variant: "icon"}, Text("With icon")),
			Button(message{Kind: "queue", Remaining: 1, Variant: "rich"}, Text("With icon and description")),
			Button(message{Kind: "queue", Remaining: 1, Variant: "undo"}, Text("Undo action · 10 seconds")),
			Button(message{Kind: "queue", Remaining: 1, Variant: "link"}, Text("Link action · 4 seconds")),
			Button(message{Kind: "queue", Remaining: 1, Variant: "short"}, Text("One-second note")),
			Button(message{Kind: "queue", Remaining: 6}, Text("Burst of six")),
			Button(message{Kind: "queue", Remaining: 3, Variant: "mixed"}, Text("Three different heights")),
			Button(message{Kind: "queue", Remaining: 1, Variant: "long"}, Text("Add a long note")),
			Button(message{Kind: "queue", Remaining: 6, Delay: 3 * time.Second}, Text("Burst in 3 seconds")),
			Button(message{}, Text("Client-side note · 10 seconds")).
				Attr(attr.ID("client-note")),
		).Alignment(Leading).Gap(10),
		Text(fmt.Sprintf("%d notes sent · %d awaiting delivery · %d undone", a.sent, a.pending, a.undone)).
			Foreground(Secondary),
		VStack(
			Text("Try these interactions").Font(Bold).Tag("h2"),
			Text("Hover or Tab into the stack to expand it and pause its timers. Notes last four seconds unless a duration is specified. Move through the gaps; the stack should stay open."),
			Text("Activate an action to perform it and dismiss the note, or use the dismiss button to close the note without acting. The message itself does not activate the action. Try an action after navigating away and back."),
			Text("Press Escape to return focus to the page. The stack stays open while hovered or expanded by touch."),
			Text("On touch screens, tap the stack to expand and tap outside to collapse. Swipe a note downward, or use its dismiss button."),
			Text("Add a long note to check the two-line message and three-line description limits. Its action row pushes the content past the height cap, where it is clipped."),
			Text("Add notes, choose “Burst in 3 seconds,” then engage with the stack before the burst arrives. New arrivals appear immediately."),
			Text("Choose “Client-side note” to reuse a registered template without sending a server message. Try it again after navigating away and back, and use Undo to check its Go action."),
			Text("Switch tabs to pause timers, or enable reduced motion in your system settings to try the quieter transitions."),
		).Alignment(Leading).Gap(12),
		VStack(
			Group(
				Path("/", Text("Playground").Title("Playground")),
				Path("/away", Text("Another page").Title("Another page")),
			).Font(Bold).Tag("h2"),
			Text("Send a note, then follow this link or use browser Back and Forward. Notes should keep their remaining time and dismissed notes should stay gone."),
			Path("/", Link("/away", Text("Go to another page"))),
			Path("/away", Link("/", Text("Back to playground"))),
		).Alignment(Leading).Gap(12),
	).Alignment(Leading).Gap(24).
		Padding(Edges(24), EdgeBottom(200)).
		Attr(attr.Style("width:100%;max-width:640px;box-sizing:border-box")).
		Tag("main")
	return ScrollView(Vertical, content)
}
