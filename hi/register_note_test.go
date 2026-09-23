package hi

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	cdppage "github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"ily.dev/act3/hi/internal/uitest"
	"ily.dev/domi"
)

func TestRegisterNoteEmptyName(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("RegisterNote accepted an empty name")
		}
	}()
	RegisterNote[int]("", Note{Message: "Saved"})
}

func TestNoteTemplatesLifecycle(t *testing.T) {
	t.Parallel()
	runNoteView(t, 800, 600, Note{Duration: time.Second}, func(s *uitest.Session) {
		s.Eval(`globalThis.unknownError = '';
			try { fixture.notify('missing'); } catch (e) { unknownError = e.message; }
			fixture.add('original');
			document.querySelector('hi-note-outbox').lastElementChild.dataset.noteTemplate = 'saved';`, nil)
		noteCheck(t, s, `unknownError.includes('unknown note template "missing"') && fixture.retained.length === 0`)
		// A registration has no running timer. Each invocation starts its own.
		s.Eval(`fixture.advance(10000);
			globalThis.snapshot = document.querySelector('#notes').cloneNode(true);
			fixture.notify('saved'); fixture.advance(500); fixture.notify('saved');
			fixture.hidden(true);`, nil)
		noteCheck(t, s, `fixture.retained.join() === 'original,original' && fixture.remaining.join() === '500,1000'`)
		s.Eval(`fixture.hidden(false); fixture.add('replacement');
			document.querySelector('hi-note-outbox').lastElementChild.dataset.noteTemplate = 'saved';`, nil)
		noteCheck(t, s, `fixture.retained.join() === 'original,original'`)
		// Replaying the old registration cannot undo the replacement.
		s.Eval(`document.querySelector('#notes').replaceWith(snapshot.cloneNode(true));`, nil)
		s.Eval(`fixture.notify('saved');`, nil)
		noteCheck(t, s, `fixture.retained.join() === 'original,original,replacement'`)
		s.Eval(`fixture.advance(500);`, nil)
		noteCheck(t, s, `fixture.retained.join() === 'original,replacement'`)
		s.Eval(`fixture.advance(500);`, nil)
		noteCheck(t, s, `fixture.retained.length === 0`)
		// Names are map keys, including names that would be special in an
		// object or a CSS selector. Delivery IDs share the live-note stream.
		s.Eval(`fixture.add('literal');
			document.querySelector('hi-note-outbox').lastElementChild.dataset.noteTemplate = '__proto__["x"]';`, nil)
		s.Eval(`fixture.notify('__proto__["x"]'); fixture.add('server');`, nil)
		s.Run(notePoll(`fixture.retained.join() === 'literal,server'`))
	})
}

func TestNoteTemplatesBrowser(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		action     View
		navigation bool
	}{
		{"message", Button(42, Text("Undo")), false},
		{"URL", Link("/action", Text("Open")), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := Handler(func(context.Context, *url.URL) (*navigationApp, domi.Cmd[int]) {
				a := &navigationApp{}
				count := 0
				a.update = func(_ context.Context, m int) domi.Cmd[int] {
					switch m {
					case 42:
						count++
					case 1:
						return domi.PushURL[int]("/away")
					case 2:
						return domi.PushURL[int]("/action")
					case 3:
						return RegisterNote[int]("saved", Note{Message: "Replacement", Action: Button(42, Text("Again"))})
					}
					return nil
				}
				a.onView = func(_ context.Context, render PageRenderer) Page {
					return render(VStack(Button(0, Text("Update")).Class("update"),
						Button(3, Text("Replace")).Class("replace"),
						Link("/away", Text("Away")).Class("away"), Text(fmt.Sprint(count)).Class("count")))
				}
				return a, RegisterNote[int]("saved", Note{
					Message: "Saved", Description: "Supporting text", Icon: "check", Action: tc.action,
				})
			}, func(u *url.URL) int {
				if u.Path == "/action" {
					return 2
				}
				return 1
			}, func(*url.URL) int { return 0 }, domi.InternalURLPrefix("/-/notes-test"))
			server := httptest.NewServer(h)
			defer server.Close()
			uitest.RunURL(t, 800, 600, "about:blank", func(s *uitest.Session) {
				s.Run(noteReduceMotion(), chromedp.ActionFunc(func(ctx context.Context) error {
					_, err := cdppage.AddScriptToEvaluateOnNewDocument(`Object.defineProperty(document, 'hidden', {value: true});`).Do(ctx)
					return err
				}), chromedp.Navigate(server.URL), chromedp.WaitReady("hi-note-outbox [data-note-template]", chromedp.ByQuery))
				s.Eval(fmt.Sprintf(`import(%q).then(m => globalThis.Hi = m);`, clientJSPath("/-/notes-test")), nil)
				s.Run(notePoll(`globalThis.Hi !== undefined`))
				noteCheck(t, s, noteText+` === '' && document.querySelector('.count').textContent === '0'`)
				// Capture an old outbox with the registration, then remove it
				// through a real update and navigate before using the template.
				s.Eval(`globalThis.oldOutbox = document.querySelector('hi-note-outbox').cloneNode(true);`, nil)
				s.Run(chromedp.Click(".update", chromedp.ByQuery),
					notePoll(`document.querySelector('hi-note-outbox').children.length === 0`),
					chromedp.Click(".away", chromedp.ByQuery), notePoll(`location.pathname === '/away'`),
					chromedp.NavigateBack(), notePoll(`location.pathname === '/'`))
				s.Eval(`Hi.notify('saved'); Hi.notify('saved');`, nil)
				noteCheck(t, s, noteText+` === 'SavedSaved' &&
					document.querySelector('hi-note-display').textContent.includes('Supporting text') &&
					document.querySelector('hi-note-display hi-icon svg') !== null &&
					getComputedStyle(document.querySelector('hi-note-display hi-note')).opacity === '1'`)
				// Replacement preserves displayed copies and their action scopes.
				s.Run(chromedp.Click(".replace", chromedp.ByQuery),
					notePoll(`document.querySelector('hi-note-outbox').textContent.includes('Replacement')`))
				s.Eval(`Hi.notify('saved');`, nil)
				noteCheck(t, s, noteText+` === 'SavedSavedReplacement'`)
				s.Eval(`document.querySelector('hi-note-display hi-note :is(button,a)').click();`, nil)
				if tc.navigation {
					s.Run(notePoll(`location.pathname === '/action'`), chromedp.NavigateBack(), notePoll(`location.pathname === '/'`))
				} else {
					s.Run(notePoll(`document.querySelector('.count').textContent === '1'`))
				}
				// A snapshot's registration is still a consumed delivery.
				s.Eval(`document.querySelector('hi-note-outbox').replaceWith(oldOutbox.cloneNode(true));`, nil)
				s.Eval(`Hi.notify('saved');`, nil)
				noteCheck(t, s, `document.querySelector('hi-note-display hi-note:last-child hi-text').textContent === 'Replacement'`)
				// A full page load gets a fresh registry and the original template.
				s.Run(chromedp.Navigate(server.URL), chromedp.WaitReady("hi-note-outbox [data-note-template]", chromedp.ByQuery))
				s.Eval(fmt.Sprintf(`import(%q).then(m => { globalThis.Hi = m; m.notify('saved'); });`, clientJSPath("/-/notes-test")), nil)
				s.Run(notePoll(noteText + ` === 'Saved'`))
			})
		})
	}
}
