package hi

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"ily.dev/act3/hi/internal/uitest"
	"ily.dev/domi"
)

func TestNotesViewConsumesOutbox(t *testing.T) {
	outbox := regexp.MustCompile(`(?s)<hi-note-outbox\b[^>]*>.*?</hi-note-outbox>`)
	a := &navigationApp{}
	a.view = Text("page")
	appUpdates := 0
	a.update = func(context.Context, int) domi.Cmd[int] { appUpdates++; return nil }
	a.preview = func(_ context.Context, _ *url.URL, render PreviewRenderer) Preview {
		return render("/next", Text("preview"))
	}
	in := &instance[int, *navigationApp]{app: a, config: configure(nil)}
	for _, text := range []string{"first <note>", "second", "second"} {
		in.Update(t.Context(), msgNotify{text: text})
	}
	in.Update(t.Context(), wrapMsg(7))
	if len(in.notes) != 3 || appUpdates != 1 {
		t.Fatalf("pending=%d, app updates=%d", len(in.notes), appUpdates)
	}
	_, _, preview := in.Preview(t.Context(), &url.URL{Path: "/next"})
	if strings.Contains(outbox.FindString(renderNode(t, preview)), "domi-key") || len(in.notes) != 3 {
		t.Fatal("preview delivered or consumed pending notes")
	}
	// An app can invoke its renderer more than once while constructing a page.
	a.onView = func(_ context.Context, render PageRenderer) Page {
		render(Text("discarded"))
		return render(Text("page"))
	}
	_, page := in.View(t.Context())
	body := outbox.FindString(renderNode(t, page))
	if strings.Count(body, "domi-key") != 3 || !strings.Contains(body, "first &lt;note&gt;") {
		t.Fatalf("missing outbox entries: %s", body)
	}
	if len(in.notes) != 0 {
		t.Fatal("View did not consume notes")
	}
	_, page = in.View(t.Context())
	if strings.Contains(outbox.FindString(renderNode(t, page)), "domi-key") {
		t.Fatal("a second View replayed notes")
	}
	in.Update(t.Context(), msgNotify{text: "later"})
	if in.notes[0].id != "4" {
		t.Fatal("delivery IDs were reused")
	}
	other := &instance[int, *navigationApp]{app: a, config: configure(nil)}
	_, page = other.View(t.Context())
	if strings.Contains(outbox.FindString(renderNode(t, page)), "domi-key") {
		t.Fatal("notes leaked between instances")
	}
}

type noteBrowserApp struct {
	stubApp
	count int
}

func (a *noteBrowserApp) Update(_ context.Context, m int) domi.Cmd[int] {
	a.count++
	if m == 1 {
		return domi.Batch[int](Notify[int]("Saved"), Notify[int]("Saved"))
	}
	return nil
}

func (*noteBrowserApp) Subscriptions(context.Context) domi.Sub[int] { return nil }

func (a *noteBrowserApp) View(_ context.Context, render PageRenderer) Page {
	return render(VStack(
		Button(1, Text("Save")).Class("save"),
		Button(2, Text("Update")).Class("update"),
		Text(fmt.Sprintf("%d", a.count)).Class("count"),
	))
}

func TestNotesBrowser(t *testing.T) {
	h := Handler(func(context.Context, *url.URL) (*noteBrowserApp, domi.Cmd[int]) {
		return &noteBrowserApp{}, Notify[int]("Initial")
	}, func(*url.URL) int { return 0 }, func(*url.URL) int { return 0 })
	server := httptest.NewServer(h)
	defer server.Close()
	uitest.RunURL(t, 800, 600, server.URL, func(s *uitest.Session) {
		s.Run(notePoll(`document.querySelector('hi-note-port')?.textContent === 'Initial'`),
			chromedp.Click(".save", chromedp.ByQuery),
			notePoll(`document.querySelector('hi-note-port').textContent === 'InitialSavedSaved'`),
		)
		// Private notification messages must never call the app's Update.
		var count string
		s.Eval(`document.querySelector('.count').textContent`, &count)
		if count != "1" {
			t.Fatalf("app update count=%s, want 1", count)
		}
		s.Run(chromedp.Click(".update", chromedp.ByQuery),
			notePoll(`document.querySelector('hi-note-outbox').children.length === 0`))
		var hidden bool
		s.Eval(`getComputedStyle(document.querySelector('hi-note-outbox')).display === 'none'`, &hidden)
		if !hidden {
			t.Fatal("outbox participates in the visible layout")
		}
		var text string
		s.Eval(`document.querySelector('hi-note-port').textContent`, &text)
		if text != "InitialSavedSaved" {
			t.Fatalf("cleanup changed displayed notes: %q", text)
		}
		// A fresh browser page owns a separate notification history.
		uitest.RunURL(t, 800, 600, server.URL, func(other *uitest.Session) {
			other.Run(
				notePoll(`document.querySelector('hi-note-port')?.textContent === 'Initial'`))
		})
	})
}

func TestNotesSnapshotRestoration(t *testing.T) {
	page := `<div id="notes"><hi-note-port></hi-note-port><hi-note-outbox hidden>
		<div domi-key="1">server</div>
		</hi-note-outbox></div><script type="module">` + string(rawClientJS) + `
		globalThis.Hi = {run};</script>`
	uitest.Run(t, 800, 600, page, func(s *uitest.Session) {
		s.Run(notePoll(`globalThis.Hi !== undefined`))
		var initialized bool
		s.Eval(`customElements.get('hi-note-port') !== undefined ||
			customElements.get('hi-note-outbox') !== undefined`, &initialized)
		if initialized {
			t.Fatal("importing the module initialized notifications before run")
		}
		s.Eval(`Hi.run(); Hi.run();`, nil)
		s.Run(notePoll(`document.querySelector('hi-note-port').textContent === 'server'`))
		s.Eval(`globalThis.snapshot = document.querySelector('#notes').cloneNode(true);
			const entry = document.createElement('div');
			entry.setAttribute('domi-key', '2');
			entry.textContent = 'later';
			document.querySelector('hi-note-outbox').appendChild(entry);`, nil)
		s.Run(notePoll(`document.querySelector('hi-note-port').textContent === 'serverlater'`))
		s.Eval(`document.querySelector('#notes').replaceWith(snapshot.cloneNode(true));`, nil)
		s.Run(notePoll(`document.querySelector('hi-note-port').textContent === 'serverlater'`))
		var original string
		s.Eval(`document.querySelector('hi-note-outbox').textContent.trim()`, &original)
		if original != "server" {
			t.Fatalf("client modified the outbox: %q", original)
		}
	})
}

// Background tabs may suspend animation frames, including a poll's timeout.
func notePoll(expression string) chromedp.PollAction {
	return chromedp.Poll(expression, nil, chromedp.WithPollingInterval(10*time.Millisecond), chromedp.WithPollingTimeout(5*time.Second))
}
