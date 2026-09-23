package hi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/input"
	cdppage "github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"ily.dev/act3/hi/internal/uitest"
	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"
)

func TestNotesViewConsumesOutbox(t *testing.T) {
	t.Parallel()
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
		in.Update(t.Context(), msgNotify{note: Note{Message: text}})
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
	in.Update(t.Context(), msgNotify{note: Note{Message: "later"}})
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
		return domi.Batch[int](Notify[int](Note{Message: "Saved"}), Notify[int](Note{Message: "Saved"}))
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
	t.Parallel()
	h := Handler(func(context.Context, *url.URL) (*noteBrowserApp, domi.Cmd[int]) {
		return &noteBrowserApp{}, Notify[int](Note{Message: "Initial"})
	}, func(*url.URL) int { return 0 }, func(*url.URL) int { return 0 },
		StyleNonce(func(context.Context) string { return "notes-test" }))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "style-src 'self' 'nonce-notes-test'")
		h.ServeHTTP(w, r)
	}))
	defer server.Close()
	open := func(fn func(*uitest.Session)) {
		uitest.RunURL(t, 800, 600, "about:blank", func(s *uitest.Session) {
			// Delivery is under test here; expiry has a separate fake-clock test.
			s.Run(noteReduceMotion(), chromedp.ActionFunc(func(ctx context.Context) error {
				_, err := cdppage.AddScriptToEvaluateOnNewDocument(`Object.defineProperty(document, 'hidden', {value: true});`).Do(ctx)
				return err
			}), chromedp.Navigate(server.URL),
				chromedp.WaitReady("hi-note-display", chromedp.ByQuery))
			fn(s)
		})
	}
	open(func(s *uitest.Session) {
		s.Run(notePoll(noteText + ` === 'Initial'`))
		s.Run(
			chromedp.Click(".save", chromedp.ByQuery),
			notePoll(noteText+` === 'InitialSavedSaved'`),
		)
		// Private notification messages must never call the app's Update.
		var count string
		s.Eval(`document.querySelector('.count').textContent`, &count)
		if count != "1" {
			t.Fatalf("app update count=%s, want 1", count)
		}
		s.Run(chromedp.Click(".update", chromedp.ByQuery),
			notePoll(`document.querySelector('hi-note-outbox').children.length === 0`))
		var text string
		s.Eval(noteText, &text)
		if text != "InitialSavedSaved" {
			t.Fatalf("cleanup changed displayed notes: %q", text)
		}
		// A fresh browser page owns a separate notification history.
		open(func(other *uitest.Session) {
			other.Run(
				notePoll(noteText + ` === 'Initial'`))
		})
	})
}

func TestNotesSnapshotRestoration(t *testing.T) {
	t.Parallel()
	page := notesPage(t, []note{{id: "1", Note: Note{Message: "server"}}}, Note{}) + `<script type="module">` + string(rawClientJS) + `
		globalThis.Hi = {run};</script>`
	uitest.Run(t, 800, 600, page, func(s *uitest.Session) {
		s.Run(noteReduceMotion(), notePoll(`globalThis.Hi !== undefined`))
		var initialized bool
		s.Eval(`customElements.get('hi-note-display') !== undefined ||
			customElements.get('hi-note-outbox') !== undefined`, &initialized)
		if initialized {
			t.Fatal("importing the module initialized notifications before run")
		}
		s.Eval(`Hi.run({clone: e => e.cloneNode(true)}); Hi.run({clone: e => e.cloneNode(true)});`, nil)
		s.Run(notePoll(noteText + ` === 'server'`))
		s.Eval(`globalThis.snapshot = document.querySelector('#notes').cloneNode(true);
			const entry = document.querySelector('#note-template').content.firstElementChild.cloneNode(true);
			entry.setAttribute('domi-key', '2');
			entry.querySelector('hi-text').textContent = 'later';
			document.querySelector('hi-note-outbox').appendChild(entry);`, nil)
		s.Run(notePoll(noteText + ` === 'serverlater'`))
		s.Eval(`document.querySelector('#notes').replaceWith(snapshot.cloneNode(true));`, nil)
		s.Run(notePoll(noteText + ` === 'serverlater'`))
		var original string
		s.Eval(`document.querySelector('hi-note-outbox hi-text').textContent`, &original)
		if original != "server" {
			t.Fatalf("client modified the outbox: %q", original)
		}
	})
}

const noteText = `[...document.querySelectorAll('hi-note-display hi-note')].map(n => n.querySelector('hi-text').textContent).join('')`

func notesPage(t *testing.T, notes []note, n Note) string {
	t.Helper()
	template := ZStack(note{Note: n}.view()).Alignment(Bottom).Tag("template").
		Attr(attr.ID("note-template"), attr.Style("display:none"))
	in := instance[struct{}, App[struct{}]]{config: configure(nil)}
	page := in.render(VStack(Text("Page control").Tag("button").Attr(attr.ID("page")), template), nil, notes)
	return `<style>` + string(staticCSS) + `</style><div id="notes">` + renderNode(t, page.page) + `</div>`
}

// Background tabs may suspend animation frames, including a poll's timeout.
func notePoll(expression string) chromedp.PollAction {
	return chromedp.Poll(expression, nil, chromedp.WithPollingInterval(10*time.Millisecond), chromedp.WithPollingTimeout(5*time.Second))
}

// The fixture exercises the real module and layout. Only its clock is
// controlled, so timer overlap and snapshots can be checked without sleeps.
// Reduced motion is the default; animation tests opt in with noteAnimate.
func runNotes(t *testing.T, w, h int, fn func(*uitest.Session)) {
	t.Helper()
	runNoteView(t, w, h, Note{}, fn)
}

func runNoteView(t *testing.T, w, h int, n Note, fn func(*uitest.Session)) {
	t.Helper()
	page := notesPage(t, nil, n) + `<script type="module">
		let now = 0, timerID = 0;
		const timers = new Map();
		const performance = {now: () => now};
		const setTimeout = (fn, delay = 0) => {
			const id = ++timerID;
			timers.set(id, {fn, at: now + delay});
			return id;
		};
		const clearTimeout = id => timers.delete(id);
		` + string(rawClientJS) + `
		let nextID = 0;
		globalThis.fixture = {
			add(...texts) {
				for (const text of texts) {
					const e = document.querySelector('#note-template').content.firstElementChild.cloneNode(true);
					e.setAttribute('domi-key', String(++nextID));
					e.querySelector('hi-text').textContent = text;
					document.querySelector('hi-note-outbox').append(e);
				}
			},
			advance(ms) {
				const end = now + ms;
				for (;;) {
					const next = [...timers].sort((a,b) => a[1].at - b[1].at)[0];
					if (!next || next[1].at > end) break;
					now = next[1].at;
					timers.delete(next[0]);
					next[1].fn();
				}
				now = end;
			},
			get retained() { return notes.map(n => n.node.querySelector('hi-text').textContent); },
			get visible() { return notes.filter(n => n.node.dataset.visible === 'true').map(n => n.node.querySelector('hi-text').textContent); },
			get remaining() { return notes.map(n => n.remaining); },
			get dragging() { return !!gesture; },
			hidden(value) {
				Object.defineProperty(document, 'hidden', {configurable: true, value});
				document.dispatchEvent(new Event('visibilitychange'));
			},
		};
		run({clone: e => e.cloneNode(true)});
		</script>`
	uitest.Run(t, w, h, page, func(s *uitest.Session) {
		s.Run(noteReduceMotion(), notePoll(`globalThis.fixture !== undefined`))
		fn(s)
	})
}

func noteCheck(t *testing.T, s *uitest.Session, expression string) {
	t.Helper()
	var ok bool
	s.Eval(expression, &ok)
	if !ok {
		t.Fatalf("false: %s", expression)
	}
}

func noteAdd(count int, texts string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Evaluate("fixture.add("+texts+");", nil),
		notePoll(fmt.Sprintf("fixture.retained.length === %d", count)),
	}
}

func noteReduceMotion() chromedp.Action {
	return emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{{Name: "prefers-reduced-motion", Value: "reduce"}})
}

// Finish setup animations and deliver their events before recording the
// transitions under test.
func noteFinishAnimations() chromedp.Action {
	return chromedp.Tasks{
		chromedp.Evaluate(`document.getAnimations().forEach(a => a.finish())`, nil),
		noteWaitAnimations(),
	}
}

func noteWaitAnimations() chromedp.Action {
	return chromedp.Tasks{
		notePoll(`document.getAnimations().length === 0`),
		chromedp.Evaluate(`new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))`, nil,
			func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }),
	}
}

func noteAnimate() chromedp.Action {
	return emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{{Name: "prefers-reduced-motion", Value: "no-preference"}})
}

func TestNotesArrivalsDuringEngagement(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, engage, release string
	}{
		{"hover", `document.querySelector('hi-note-display').dispatchEvent(new PointerEvent('pointerenter', {pointerType: 'mouse'}));`,
			`document.querySelector('hi-note-display').dispatchEvent(new PointerEvent('pointerleave', {pointerType: 'mouse'}));`},
		{"focus", `document.querySelector('hi-note-display hi-note button').focus();`, `document.querySelector('#page').focus();`},
		{"touch", `document.querySelector('hi-note-display hi-note hi-text').dispatchEvent(new PointerEvent('pointerup', {pointerType: 'touch', bubbles: true}));`,
			`document.querySelector('#page').dispatchEvent(new PointerEvent('pointerdown', {pointerType: 'touch', bubbles: true}));`},
	} {
		t.Run(test.name, func(t *testing.T) {
			runNotes(t, 800, 600, func(s *uitest.Session) {
				s.Run(noteAdd(1, `'existing'`))
				s.Eval(`fixture.advance(1000); `+test.engage+` fixture.add('incoming');`, nil)
				s.Run(notePoll(`fixture.retained.join() === 'existing,incoming'`))
				noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'true' &&
					fixture.remaining.join() === '3000,4000'`)
				s.Eval(`const outbox = document.querySelector('hi-note-outbox');
					outbox.append(outbox.lastElementChild.cloneNode(true));`, nil)
				s.Eval(`fixture.advance(10000);`, nil)
				noteCheck(t, s, `fixture.retained.join() === 'existing,incoming' &&
					fixture.remaining.join() === '3000,4000' &&
					document.querySelectorAll('hi-note-display hi-note').length === 2`)
				s.Eval(test.release, nil)
				s.Run(notePoll(`document.querySelector('hi-note-display').dataset.expanded === 'false'`))
			})
		})
	}
}

func TestNotesLatestThree(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Eval(`fixture.add('1', '2', '3', '4', '5');`, nil)
		s.Run(notePoll(`fixture.visible.join() === '3,4,5'`))
		noteCheck(t, s, `fixture.retained.join() === '1,2,3,4,5' &&
			document.querySelector('hi-note-outbox').children.length === 5 &&
			[...document.querySelectorAll('hi-note-display hi-note[data-visible=false]')].every(n => getComputedStyle(n).pointerEvents === 'none')`)
		s.Eval(`fixture.advance(1200);
			document.querySelector('#page').focus();
			document.querySelectorAll('hi-note-display hi-note[data-visible=true] button')[2].focus();
			fixture.add('6');`, nil)
		s.Run(notePoll(`fixture.visible.join() === '4,5,6'`))
		noteCheck(t, s, `fixture.remaining.join() === '2800,2800,2800,2800,2800,4000' && document.activeElement.closest('hi-note').textContent.includes('5')`)
		s.Eval(`fixture.advance(10000); fixture.add('7', '8', '9', '10');`, nil)
		s.Run(notePoll(`fixture.visible.join() === '8,9,10'`))
		noteCheck(t, s, `fixture.retained.length === 10 && document.activeElement.closest('hi-note').textContent.includes('8') &&
			document.querySelector('hi-note-display').dataset.expanded === 'true'`)
		s.Eval(`document.querySelector('hi-note-display hi-note[data-visible=true] button').focus();
			fixture.advance(10000); document.activeElement.click();`, nil)
		noteCheck(t, s, `fixture.visible.join() === '7,9,10' && fixture.retained.length === 9 &&
			document.activeElement.closest('hi-note').textContent.includes('9')`)
		s.Eval(`document.querySelector('hi-note-display hi-note:last-child button').click();`, nil)
		noteCheck(t, s, `fixture.visible.join() === '6,7,9' && fixture.retained.length === 8 &&
			[...document.querySelectorAll('hi-note-display hi-note[data-visible=true] button')].every(b => b.tabIndex === 0)`)
		s.Eval(`document.querySelector('#page').focus();`, nil)
		s.Run(notePoll(`document.querySelector('hi-note-display').dataset.expanded === 'false'`))
		s.Eval(`fixture.advance(2800);`, nil)
		noteCheck(t, s, `fixture.retained.join() === '6,7,9' && fixture.visible.join() === '6,7,9'`)
	})
}

func TestNotesRetainedVisibility(t *testing.T) {
	t.Parallel()
	action := HStack(Button(42, Text("Undo")), Link("#other", Text("Other")))
	runNoteView(t, 800, 600, Note{Action: action}, func(s *uitest.Session) {
		s.Run(noteAdd(1, `'0'`))
		s.Eval(`globalThis.initialRemaining = fixture.remaining[0];
			fixture.advance(1000);
			globalThis.originalFocus = document.querySelector('hi-note-display hi-note button');
			originalFocus.focus();
			fixture.add(...Array.from({length: 4}, (_, i) => String(i + 1)));`, nil)
		s.Run(notePoll(`fixture.retained.length === 5 && fixture.visible.join() === '2,3,4'`))
		noteCheck(t, s, `document.activeElement === document.querySelector('hi-note-display hi-note[data-visible=true]') &&
			originalFocus.tabIndex === 0 && !originalFocus.hasAttribute('tabindex') &&
			[...document.querySelectorAll('hi-note-display hi-note[data-visible=false]')].every(n =>
				getComputedStyle(n).opacity === '0' && getComputedStyle(n).pointerEvents === 'none' &&
				getComputedStyle(n.querySelector('button')).pointerEvents === 'none' &&
				n.querySelector('button').tabIndex === 0 && n.inert)`)
		s.Eval(`originalFocus.focus();`, nil)
		noteCheck(t, s, `document.activeElement !== originalFocus`)
		s.Run(chromedp.KeyEvent("\t"))
		noteCheck(t, s, `document.activeElement.matches('hi-note[data-visible=true] button')`)
		s.Run(chromedp.KeyEvent("\t"))
		noteCheck(t, s, `document.activeElement.matches('hi-note[data-visible=true] a[href="#other"]') &&
			[...document.querySelectorAll('hi-note-display hi-note :is(button,a)')].every(e =>
				e.tabIndex === 0 && !e.hasAttribute('tabindex'))`)
		noteCheck(t, s, `(() => {
			const display = document.querySelector('hi-note-display');
			const visible = [...display.querySelectorAll('[data-visible=true]')];
			const expected = visible.reduce((sum, n) => sum + parseFloat(getComputedStyle(n).height), 28);
			const hidden = display.querySelectorAll('[data-visible=false]');
			const rect = hidden[hidden.length - 1].getBoundingClientRect();
			return Math.abs(parseFloat(display.style.height) - expected) < 1 &&
				rect.y + rect.height / 2 < display.getBoundingClientRect().top &&
				!display.contains(document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2));
		})()`)
		s.Eval(`fixture.advance(10000); document.querySelector('#page').focus();`, nil)
		s.Run(notePoll(`document.querySelector('hi-note-display').dataset.expanded === 'false'`))
		noteCheck(t, s, `(() => {
			const display = document.querySelector('hi-note-display');
			return Math.abs(parseFloat(display.style.height) - display.lastElementChild.offsetHeight - 28) < 1 &&
				fixture.remaining[0] === initialRemaining - 1000 &&
				fixture.remaining.slice(1).every(remaining => remaining === initialRemaining);
		})()`)
	})
}

func TestNotesFocusBetweenButtons(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAdd(2, `'one', 'two'`))
		s.Eval(`document.querySelector('hi-note-display hi-note button').focus();
			document.querySelector('hi-note-display').addEventListener('focusout', () => {
				globalThis.focusoutExpanded = document.querySelector('hi-note-display').dataset.expanded;
				fixture.advance(4000);
			}, {once: true});
			document.querySelector('hi-note-display hi-note:last-child button').focus();`, nil)
		noteCheck(t, s, `focusoutExpanded === 'true' && fixture.retained.length === 2 &&
			fixture.remaining.every(remaining => remaining === 4000)`)
	})
}

func TestNotesCollapsedContent(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAdd(3, `'a taller rear note '.repeat(20), 'middle', 'front'`))
		noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'false' &&
			[...document.querySelectorAll('hi-note[data-covered=true]')].every(n =>
				getComputedStyle(n).opacity === '1' &&
				getComputedStyle(n.firstElementChild).opacity === '0')`)
	})
}

func TestNotesMessageFocus(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAdd(1, `'click me'`))
		s.Eval(`document.querySelector('#notes').tabIndex = -1;
			document.querySelector('#page').focus();`, nil)
		r := s.Rect("hi-note-display hi-note hi-text", 0)
		x, y := r.X+r.W/2, r.Y+r.H/2
		s.Run(input.DispatchMouseEvent(input.MouseMoved, x, y),
			input.DispatchMouseEvent(input.MousePressed, x, y).WithButton(input.Left).WithClickCount(1))
		noteCheck(t, s, `fixture.dragging`)
		s.Run(input.DispatchMouseEvent(input.MouseReleased, x, y).WithButton(input.Left).WithClickCount(1),
			input.DispatchMouseEvent(input.MouseMoved, 2, 2))
		noteCheck(t, s, `document.activeElement === document.querySelector('hi-note-display hi-note') &&
			document.activeElement.tabIndex === 0 && !document.activeElement.matches(':focus-visible') &&
			document.querySelector('hi-note-display').dataset.expanded === 'false'`)
		s.Eval(`fixture.advance(1000);`, nil)
		s.Run(chromedp.KeyEvent("a"))
		noteCheck(t, s, `document.activeElement.matches('hi-note:focus-visible') &&
			document.querySelector('hi-note-display').dataset.expanded === 'true'`)
		s.Eval(`fixture.advance(10000);`, nil)
		noteCheck(t, s, `fixture.retained.length === 1 && fixture.remaining[0] === 3000`)
		s.Run(chromedp.KeyEvent("\x1b"))
		noteCheck(t, s, `document.activeElement.id === 'page'`)
		s.Run(chromedp.KeyEvent("\t"))
		noteCheck(t, s, `document.activeElement.matches('hi-note:focus-visible')`)
		s.Run(chromedp.KeyEvent("\r"))
		noteCheck(t, s, `fixture.retained.length === 1`)
		s.Run(chromedp.KeyEvent("\t"))
		noteCheck(t, s, `document.activeElement.matches('hi-note button:focus-visible') &&
			document.querySelector('hi-note-display').dataset.expanded === 'true'`)
		s.Eval(`fixture.advance(10000);`, nil)
		noteCheck(t, s, `fixture.retained.length === 1 && fixture.remaining[0] === 3000`)
		s.Run(chromedp.KeyEvent("\x1b"),
			notePoll(`document.querySelector('hi-note-display').dataset.expanded === 'false'`))
		s.Eval(`fixture.advance(4000);`, nil)
		noteCheck(t, s, `fixture.retained.length === 0`)
	})
}

func TestNotesMouseFocusExpires(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAdd(1, `'click me'`))
		s.Eval(`document.querySelector('#page').focus();`, nil)
		s.Run(chromedp.Click("hi-note-display hi-note hi-text", chromedp.ByQuery),
			input.DispatchMouseEvent(input.MouseMoved, 2, 2))
		noteCheck(t, s, `document.activeElement.matches('hi-note:not(:focus-visible)') &&
			document.querySelector('hi-note-display').dataset.expanded === 'false'`)
		s.Eval(`fixture.advance(4000);`, nil)
		noteCheck(t, s, `fixture.retained.length === 0 && document.activeElement.id === 'page'`)
	})
}

func TestNotesPauseReasons(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAdd(1, `'one'`))
		s.Eval(`fixture.advance(1000);
			document.querySelector('hi-note-display').dispatchEvent(new PointerEvent('pointerenter', {pointerType:'mouse'}));
			document.querySelector('hi-note-display hi-note button').focus();
			fixture.hidden(true);
			fixture.advance(10000);
			document.querySelector('hi-note-display').dispatchEvent(new PointerEvent('pointerleave', {pointerType:'mouse'}));
			fixture.hidden(false);
			fixture.advance(10000);`, nil)
		noteCheck(t, s, `fixture.retained.length === 1 && fixture.remaining[0] === 3000`)
		s.Eval(`document.querySelector('#page').focus();`, nil)
		s.Run(notePoll(`document.querySelector('hi-note-display').dataset.expanded === 'false'`))
		s.Eval(`fixture.advance(2999);`, nil)
		noteCheck(t, s, `fixture.retained.length === 1`)
		s.Eval(`fixture.advance(1); fixture.hidden(true); fixture.add('hidden');`, nil)
		s.Run(notePoll(`fixture.retained.join() === 'hidden'`))
		s.Eval(`fixture.advance(10000); fixture.hidden(false); fixture.advance(3999);`, nil)
		noteCheck(t, s, `fixture.retained.join() === 'hidden'`)
		s.Eval(`fixture.advance(1);`, nil)
		noteCheck(t, s, `fixture.retained.length === 0`)
	})
}

func TestNotesSnapshotLifecycle(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAdd(5, `'1', '2', '3', '4', '5'`))
		s.Eval(`globalThis.snapshot = document.querySelector('#notes').cloneNode(true);
			fixture.advance(2000);
			document.querySelector('hi-note-display hi-note:last-child button').click();
			const restored = snapshot.cloneNode(true);
			globalThis.restoredDisplay = restored.querySelector('hi-note-display');
			document.querySelector('#notes').replaceWith(restored);`, nil)
		noteCheck(t, s, `fixture.retained.join() === '1,2,3,4' && fixture.visible.join() === '2,3,4' &&
			document.querySelectorAll('hi-note-display hi-note').length === 4 &&
			document.querySelector('hi-note-display') === restoredDisplay`)
		s.Eval(`fixture.advance(1999);`, nil)
		noteCheck(t, s, `fixture.retained.join() === '1,2,3,4'`)
		s.Eval(`fixture.advance(1); document.querySelector('#notes').replaceWith(snapshot.cloneNode(true));`, nil)
		noteCheck(t, s, `fixture.retained.length === 0 && document.querySelectorAll('hi-note-display hi-note').length === 0`)
	})
}

func TestNotesSnapshotHover(t *testing.T) {
	t.Parallel()
	for _, leave := range []bool{false, true} {
		t.Run(fmt.Sprintf("left_window=%t", leave), func(t *testing.T) {
			runNotes(t, 800, 600, func(s *uitest.Session) {
				s.Run(noteAdd(1, `'one'`))
				r := s.Rect("hi-note-display hi-note", 0)
				s.Run(input.DispatchMouseEvent(input.MouseMoved, r.X+20, r.Y+20))
				if leave {
					// Leaving the window delivers pointerleave without another
					// pointermove giving the document new coordinates.
					s.Run(input.DispatchMouseEvent(input.MouseMoved, -1, -1))
				}
				s.Eval(`document.querySelector('#notes').replaceWith(document.querySelector('#notes').cloneNode(true));`, nil)
				noteCheck(t, s, fmt.Sprintf(`document.querySelector('hi-note-display').dataset.expanded === '%t'`, !leave))
				s.Eval(`fixture.advance(4000);`, nil)
				noteCheck(t, s, fmt.Sprintf(`(fixture.retained.length === 0) === %t`, leave))
			})
		})
	}
}

func TestNotesLayoutAndFocus(t *testing.T) {
	t.Parallel()
	runNotes(t, 400, 360, func(s *uitest.Session) {
		s.Eval(`fixture.add('short', 'A longer message with enough words to wrap over several lines.', 'very long '.repeat(100));`, nil)
		s.Run(notePoll(`fixture.retained.length === 3 && document.querySelector('hi-note-display').offsetHeight > 0`))
		s.Eval(`document.querySelector('#page').focus(); document.querySelector('hi-note-display hi-note button').focus();`, nil)
		s.Run(notePoll(`document.querySelector('hi-note-display').dataset.expanded === 'true'`))
		noteCheck(t, s, `(() => {
			const cards = [...document.querySelectorAll('hi-note-display hi-note')].map(n => n.getBoundingClientRect());
			return cards.every(r => r.x >= 16 && r.right <= 384 && r.top >= 16 && r.bottom <= 344) &&
				cards[0].bottom < cards[1].top && cards[1].bottom < cards[2].top;
		})()`)
		s.Eval(`document.querySelector('hi-note-display hi-note button').click();
			document.activeElement.querySelector('button').click();
			document.activeElement.querySelector('button').click();`, nil)
		noteCheck(t, s, `document.activeElement.id === 'page' && fixture.retained.length === 0`)
	})
}

func TestNotesCloseFocus(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"mouse", "touch", "keyboard"} {
		t.Run(mode, func(t *testing.T) {
			runNotes(t, 800, 600, func(s *uitest.Session) {
				s.Run(noteAdd(2, `'older', 'newer'`))
				s.Eval(`fixture.advance(1000); document.querySelector('#page').focus();`, nil)
				if mode == "mouse" {
					r := s.Rect("hi-note-display hi-note button", 1)
					x, y := r.X+r.W/2, r.Y+r.H/2
					s.Run(input.DispatchMouseEvent(input.MouseMoved, x, y),
						input.DispatchMouseEvent(input.MousePressed, x, y).WithButton(input.Left).WithClickCount(1),
						input.DispatchMouseEvent(input.MouseReleased, x, y).WithButton(input.Left).WithClickCount(1))
				} else {
					s.Eval(`document.querySelectorAll('hi-note-display hi-note button')[1].focus();`, nil)
					if mode == "touch" {
						s.Eval(`document.querySelector('hi-note-display hi-note hi-text').dispatchEvent(new PointerEvent('pointerup', {pointerType: 'touch', bubbles: true}));
							document.activeElement.dispatchEvent(new MouseEvent('click', {detail: 1, bubbles: true}));`, nil)
					} else {
						s.Eval(`document.querySelector('hi-note-display hi-note button').disabled = true;`, nil)
						s.Run(chromedp.KeyEvent("\r"))
					}
				}
				s.Run(notePoll(`fixture.retained.join() === 'older'`))
				s.Eval(`fixture.advance(10000);`, nil)
				noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'true' &&
					fixture.retained.join() === 'older' && fixture.remaining.join() === '3000'`)
				if mode == "keyboard" {
					noteCheck(t, s, `document.activeElement === document.querySelector('hi-note-display hi-note')`)
					s.Run(chromedp.KeyEvent("\x1b"))
				} else {
					noteCheck(t, s, `document.activeElement.id === 'page'`)
					s.Eval(`document.querySelector('hi-note-display hi-note button').focus();`, nil)
					s.Run(chromedp.KeyEvent("\x1b"))
					noteCheck(t, s, `document.activeElement.id === 'page' &&
						document.querySelector('hi-note-display').dataset.expanded === 'true'`)
					if mode == "mouse" {
						s.Run(input.DispatchMouseEvent(input.MouseMoved, 2, 2))
					} else {
						s.Eval(`document.querySelector('#page').dispatchEvent(new PointerEvent('pointerdown', {pointerType: 'touch', bubbles: true}));`, nil)
					}
				}
				s.Run(notePoll(`document.querySelector('hi-note-display').dataset.expanded === 'false'`))
			})
		})
	}
}

func TestNotesEscapeFocus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, target, fallback string
	}{
		{name: "focus only", target: "hi-note-display hi-note button"},
		{name: "content focus", target: "hi-note-display hi-note hi-text"},
		{name: "removed previous focus", target: "hi-note-display hi-note button", fallback: "remove()"},
		{name: "disabled previous focus", target: "hi-note-display hi-note button", fallback: "disabled = true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runNotes(t, 800, 600, func(s *uitest.Session) {
				s.Run(noteAdd(1, `'existing'`))
				s.Eval(`fixture.advance(1000); document.querySelector('#page').focus();
					const target = document.querySelector('`+tc.target+`');
					target.tabIndex = -1; target.focus();`, nil)
				noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'true' &&
					fixture.remaining.join() === '3000'`)
				if tc.fallback != "" {
					s.Eval(`document.querySelector('#page').`+tc.fallback+`;`, nil)
				}
				s.Eval(`fixture.advance(10000);`, nil)
				s.Run(chromedp.KeyEvent("\x1b"))
				s.Run(notePoll(`!document.querySelector('hi-note-display').contains(document.activeElement)`))
				if tc.fallback == "" {
					noteCheck(t, s, `document.activeElement.id === 'page'`)
				} else {
					noteCheck(t, s, `document.activeElement === document.body && !document.body.hasAttribute('tabindex')`)
				}
				s.Run(notePoll(`document.querySelector('hi-note-display').dataset.expanded === 'false'`))
			})
		})
	}
}

func TestNotesPointerDismissal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		drag     float64
		duration time.Duration
		cancel   string
		dismiss  bool
	}{
		{"distance", 90, time.Second, "", true},
		{"downward flick", 24, 100 * time.Millisecond, "", true},
		{"upward flick", -90, 100 * time.Millisecond, "", false},
		{"short slow drag", 24, 300 * time.Millisecond, "", false},
		{"cancel", 90, 100 * time.Millisecond, "cancel", false},
		{"lost capture", 90, 100 * time.Millisecond, "lost", false},
		{"snapshot", 90, 100 * time.Millisecond, "snapshot", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runNotes(t, 800, 600, func(s *uitest.Session) {
				s.Run(noteAdd(1, `'drag me'`))
				r := s.Rect("hi-note-display hi-note", 0)
				x, y := r.X+20, r.Y+20
				// Gesture speed depends on event timestamps, not runner load.
				start := time.Now()
				at := func(event *input.DispatchMouseEventParams, elapsed time.Duration) chromedp.Action {
					return chromedp.ActionFunc(func(ctx context.Context) error {
						// cdproto's TimeSinceEpoch marshaler truncates fractions.
						return cdp.Execute(ctx, input.CommandDispatchMouseEvent, struct {
							*input.DispatchMouseEventParams
							Timestamp float64 `json:"timestamp"`
						}{event, float64(start.Add(elapsed).UnixMicro()) / 1e6}, nil)
					})
				}
				s.Run(input.DispatchMouseEvent(input.MouseMoved, x, y),
					at(input.DispatchMouseEvent(input.MousePressed, x, y).WithButton(input.Left).WithClickCount(1), 0))
				noteCheck(t, s, `fixture.dragging`)
				s.Eval(`fixture.add('arrived');`, nil)
				s.Run(notePoll(`fixture.retained.join() === 'drag me,arrived'`))
				noteCheck(t, s, `fixture.dragging`)
				s.Run(at(input.DispatchMouseEvent(input.MouseMoved, x, y+tc.drag).WithButton(input.Left).WithButtons(1), 10*time.Millisecond))
				if tc.name == "distance" {
					noteCheck(t, s, `document.querySelector('hi-note-display hi-note').style.getPropertyValue('--swipe') === '90px'`)
					s.Run(at(input.DispatchMouseEvent(input.MouseMoved, x, y+100).WithButton(input.Left).WithButtons(1), 20*time.Millisecond))
					noteCheck(t, s, `document.querySelector('hi-note-display hi-note').style.getPropertyValue('--swipe') === '100px'`)
					s.Run(at(input.DispatchMouseEvent(input.MouseMoved, x, y+tc.drag).WithButton(input.Left).WithButtons(1), 30*time.Millisecond))
				}
				switch tc.cancel {
				case "cancel":
					s.Eval(`document.querySelector('hi-note-display hi-note').dispatchEvent(new PointerEvent('pointercancel', {pointerId:1, bubbles:true}));`, nil)
				case "lost":
					s.Eval(`document.querySelector('hi-note-display hi-note').releasePointerCapture(1);`, nil)
				case "snapshot":
					s.Eval(`document.querySelector('#notes').replaceWith(document.querySelector('#notes').cloneNode(true));`, nil)
				}
				s.Run(at(input.DispatchMouseEvent(input.MouseReleased, x, y+tc.drag).WithButton(input.Left).WithClickCount(1), tc.duration),
					input.DispatchMouseEvent(input.MouseMoved, 2, 2))
				noteCheck(t, s, `!fixture.dragging`)
				if tc.dismiss {
					noteCheck(t, s, `!fixture.retained.includes('drag me')`)
				} else {
					noteCheck(t, s, `fixture.retained.includes('drag me')`)
				}
			})
		})
	}
}

func TestNotesCoveredPress(t *testing.T) {
	t.Parallel()
	for _, pointerType := range []string{"touch", "pen"} {
		t.Run(pointerType, func(t *testing.T) {
			runNotes(t, 800, 600, func(s *uitest.Session) {
				s.Run(noteAdd(2, `'rear', 'front'`))
				press := func(x, y float64) {
					if pointerType == "touch" {
						s.Run(input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{{X: x, Y: y}}))
					} else {
						s.Run(input.DispatchMouseEvent(input.MousePressed, x, y).WithPointerType("pen").WithButton(input.Left).WithClickCount(1))
					}
				}
				release := func(x, y float64) {
					if pointerType == "touch" {
						s.Run(input.DispatchTouchEvent(input.TouchMove, []*input.TouchPoint{{X: x, Y: y}}), input.DispatchTouchEvent(input.TouchEnd, nil))
					} else {
						s.Run(input.DispatchMouseEvent(input.MouseMoved, x, y).WithPointerType("pen").WithButton(input.Left).WithButtons(1),
							input.DispatchMouseEvent(input.MouseReleased, x, y).WithPointerType("pen").WithButton(input.Left).WithClickCount(1))
					}
				}
				r := s.Rect("hi-note-display hi-note", 0)
				press(r.X+30, r.Y+3)
				noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'true' && !fixture.dragging`)
				release(r.X+30, r.Y+63)
				noteCheck(t, s, `fixture.retained.join() === 'rear,front'`)
				// Once revealed, the rear note supports ordinary swiping.
				r = s.Rect("hi-note-display hi-note", 0)
				press(r.X+30, r.Y+20)
				noteCheck(t, s, `fixture.dragging`)
				release(r.X+30, r.Y+80)
				noteCheck(t, s, `fixture.retained.join() === 'front'`)
			})
		})
	}
}

func TestNotesTouchExpansion(t *testing.T) {
	t.Parallel()
	runNotes(t, 400, 500, func(s *uitest.Session) {
		s.Run(emulation.SetTouchEmulationEnabled(true))
		s.Run(noteAdd(3, `'one', 'two', 'three'`))
		r := s.Rect("hi-note-display hi-note", 2)
		s.Run(input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{{X: r.X + 20, Y: r.Y + 20}}),
			input.DispatchTouchEvent(input.TouchEnd, nil))
		noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'true' && fixture.retained.length === 3`)
		s.Eval(`fixture.add('four', 'five', 'six', 'seven');`, nil)
		s.Run(notePoll(`fixture.visible.join() === 'five,six,seven'`))
		noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'true'`)
		s.Eval(`document.querySelector('#page').addEventListener('click', () => document.body.dataset.clicked = 'yes');`, nil)
		r = s.Rect("#page", 0)
		s.Run(input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{{X: r.X + 10, Y: r.Y + 10}}),
			input.DispatchTouchEvent(input.TouchEnd, nil))
		s.Run(notePoll(`fixture.visible.join() === 'five,six,seven' && document.body.dataset.clicked === 'yes'`))
		// New arrivals preserve touch expansion until the display is emptied.
		r = s.Rect("hi-note-display hi-note[data-state=active][data-visible=true]", 2)
		s.Run(input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{{X: r.X + 20, Y: r.Y + 20}}),
			input.DispatchTouchEvent(input.TouchEnd, nil))
		s.Eval(`fixture.add('last');`, nil)
		s.Run(notePoll(`fixture.visible.join() === 'six,seven,last'`))
		noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'true'`)
		s.Eval(`for (const button of document.querySelectorAll('hi-note-display hi-note[data-state=active] button')) button.click();`, nil)
		noteCheck(t, s, `fixture.retained.length === 0 && document.querySelector('hi-note-display').dataset.expanded === 'false'`)
		s.Eval(`fixture.advance(6000);`, nil)
		noteCheck(t, s, `document.querySelectorAll('hi-note-display hi-note').length === 0`)
	})
}

func TestNotesHoverContinuityAndResize(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAdd(2, `'first', 'second'`))
		r := s.Rect("hi-note-display hi-note", 1)
		s.Run(input.DispatchMouseEvent(input.MouseMoved, r.X+20, r.Y+20))
		older, newer := s.Rect("hi-note-display hi-note", 0), s.Rect("hi-note-display hi-note", 1)
		s.Run(input.DispatchMouseEvent(input.MouseMoved, r.X+20, (older.Bottom()+newer.Y)/2))
		s.Eval(`fixture.advance(10000); fixture.add('third');`, nil)
		s.Run(notePoll(`fixture.retained.join() === 'first,second,third'`))
		noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'true'`)
		s.Run(input.DispatchMouseEvent(input.MouseMoved, 2, 2))
		s.Eval(`document.querySelector('#page').focus();`, nil)
		s.Run(chromedp.KeyEvent("\t"), chromedp.EmulateViewport(600, 220))
		noteCheck(t, s, `document.activeElement.matches('hi-note-display hi-note') &&
			document.querySelector('hi-note-display').dataset.expanded === 'true' &&
			[...document.querySelectorAll('hi-note-display hi-note[data-state=active]')].every(n => {
				const r = n.getBoundingClientRect(); return r.width === 360 && r.x >= 16 && r.right <= 584 && r.top >= 16;
			})`)
	})
}

func TestNotesFixedHeightCap(t *testing.T) {
	t.Parallel()
	n := Note{
		Description: strings.Repeat("Details about the update. ", 20),
		Action:      Button(noAction{}, Text("Undo")),
	}
	runNoteView(t, 400, 600, n, func(s *uitest.Session) {
		s.Run(noteAdd(1, `'long text '.repeat(200)`))
		s.Eval(`globalThis.first = document.querySelector('hi-note-display hi-note');
			first.querySelector('[data-dismiss]').focus();`, nil)
		noteCheck(t, s, `first.offsetHeight === 112 && getComputedStyle(first.firstElementChild).maxHeight === '112px'`)
		s.Run(noteAdd(3, `'short', 'another long text '.repeat(200)`))
		s.Eval(`globalThis.cappedHeights = [...document.querySelectorAll('hi-note-display hi-note')].map(n => n.offsetHeight);`, nil)
		noteCheck(t, s, `cappedHeights[0] === 112 && cappedHeights[1] <= 112 && cappedHeights[2] === 112`)
	})
}

func TestNotesTouchDismissal(t *testing.T) {
	t.Parallel()
	runNotes(t, 400, 360, func(s *uitest.Session) {
		s.Run(emulation.SetTouchEmulationEnabled(true))
		s.Run(noteAdd(1, `'long text '.repeat(200)`))
		r := s.Rect("hi-note-display hi-note", 0)
		x, y := r.X+30, r.Y+20
		s.Run(input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{{X: x, Y: y}}))
		noteCheck(t, s, `fixture.dragging`)
		s.Run(input.DispatchTouchEvent(input.TouchMove, []*input.TouchPoint{{X: x, Y: y - 20}}),
			input.DispatchTouchEvent(input.TouchEnd, nil))
		noteCheck(t, s, `fixture.retained.length === 1 && !fixture.dragging`)
		s.Run(input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{{X: x, Y: y}}),
			input.DispatchTouchEvent(input.TouchMove, []*input.TouchPoint{{X: x, Y: y + 60}}),
			input.DispatchTouchEvent(input.TouchEnd, nil))
		noteCheck(t, s, `fixture.retained.length === 0 && !fixture.dragging`)
	})
}

func TestNotesCollapseKeepsStableGeometry(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAnimate())
		s.Run(noteAdd(3, `'short', 'medium text '.repeat(8), 'tall text '.repeat(25)`), noteFinishAnimations())
		front := s.Rect("hi-note-display hi-note", 2)
		s.Run(input.DispatchMouseEvent(input.MouseMoved, front.X+30, front.Y+20),
			noteFinishAnimations())
		display := s.Rect("hi-note-display", 0)
		s.Eval(`globalThis.collapseFrames = [];
			globalThis.collapseDone = false;
			let settled = 0;
			function sample() {
				collapseFrames.push({
					expanded: document.querySelector('hi-note-display').dataset.expanded,
					tops: [...document.querySelectorAll('hi-note-display hi-note')].map(n => n.getBoundingClientRect().top),
				});
				const collapsed = document.querySelector('hi-note-display').dataset.expanded === 'false';
				settled = collapsed && document.getAnimations().length === 0 ? settled + 1 : 0;
				if (settled >= 4 && collapseFrames.filter(f => f.expanded === 'false').length > 10) collapseDone = true;
				else requestAnimationFrame(sample);
			}
			requestAnimationFrame(sample);`, nil)
		s.Run(input.DispatchMouseEvent(input.MouseMoved, front.X+30, display.Y-2),
			notePoll(`collapseDone`))
		noteCheck(t, s, `(() => {
			const first = collapseFrames.findIndex(f => f.expanded === 'false');
			const frames = collapseFrames.slice(first);
			return first >= 0 && frames.length > 10 && frames.every(f => f.expanded === 'false') &&
				frames.every((f, i) => i === 0 || f.tops.every((top, j) => top >= frames[i-1].tops[j] - 1));
		})()`)
		// Keyboard focus holds expansion independently of the pointer.
		s.Eval(`document.querySelector('hi-note-display hi-note button').focus();`, nil)
		s.Run(input.DispatchMouseEvent(input.MouseMoved, 2, 2))
		noteCheck(t, s, `document.querySelector('hi-note-display').dataset.expanded === 'true'`)
	})
}

func TestNotesEngagementKeepsTransitions(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAnimate())
		s.Run(noteAdd(2, `'short', 'tall text '.repeat(25)`), noteFinishAnimations())
		s.Eval(`globalThis.motionRuns = [];
			globalThis.motionCancels = [];
			const back = document.querySelector('hi-note-display hi-note');
			back.addEventListener('transitionrun', e => {
				if (e.target === back) motionRuns.push(e.propertyName);
			});
			back.addEventListener('transitioncancel', e => {
				if (e.target === back) motionCancels.push(e.propertyName);
			});
			back.querySelector('button').focus();`, nil)
		s.Run(notePoll(`motionRuns.includes('height') && motionRuns.includes('transform') &&
			back.getAnimations().some(a => a.playState === 'running' && a.currentTime > 0)`))
		// Unrelated ancestor changes must not disturb an in-flight transition.
		s.Eval(`document.body.style.setProperty('--unrelated', '1');`, nil)
		front := s.Rect("hi-note-display hi-note", 1)
		s.Run(input.DispatchMouseEvent(input.MouseMoved, front.X+30, front.Y+20),
			noteWaitAnimations())
		s.Eval(`document.querySelector('#page').focus();`, nil)
		s.Run(notePoll(`document.querySelector('hi-note-display').dataset.expanded === 'true'`),
			input.DispatchMouseEvent(input.MouseMoved, 2, 2), noteWaitAnimations())
		noteCheck(t, s, `motionCancels.length === 0 &&
			motionRuns.filter(p => p === 'height').length === 2 &&
			motionRuns.filter(p => p === 'transform').length === 2`)
	})
}

func TestNotesDetachedAdmission(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Run(noteAdd(1, `'short'`))
		s.Eval(`globalThis.back = document.querySelector('hi-note-display hi-note');
			globalThis.shortHeight = back.offsetHeight;
			fixture.add('tall text '.repeat(25));`, nil)
		s.Run(notePoll(`fixture.retained.length === 2 && back.offsetHeight > shortHeight`))
		s.Eval(`back.querySelector('button').focus();`, nil)
		noteCheck(t, s, `back.offsetHeight === shortHeight`)
		// Detached admissions have no geometry until the display reconnects.
		s.Eval(`globalThis.noteDisplay = document.querySelector('hi-note-display');
			globalThis.displayParent = noteDisplay.parentNode;
			noteDisplay.remove(); fixture.add('detached '.repeat(30));`, nil)
		s.Run(notePoll(`fixture.retained.length === 3`))
		s.Eval(`displayParent.append(noteDisplay); back.querySelector('button').focus();`, nil)
		noteCheck(t, s, `back.offsetHeight === shortHeight &&
			document.querySelector('hi-note-display hi-note:last-child').offsetHeight > shortHeight`)
	})
}

func TestNotesExitRemoval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		kind    string
		reduced bool
	}{
		{kind: "normal"},
		{kind: "covered"},
		{kind: "swipe"},
		{kind: "normal", reduced: true},
	} {
		kind, reduced := tc.kind, tc.reduced
		t.Run(fmt.Sprintf("%s/reduced=%t", kind, reduced), func(t *testing.T) {
			t.Parallel()
			runNotes(t, 800, 600, func(s *uitest.Session) {
				s.Run(noteAdd(3, `'rear', 'tall text '.repeat(25), 'front'`))
				selector := "hi-note-display hi-note:last-child"
				if kind == "covered" {
					selector = "hi-note-display hi-note:first-child"
				}
				s.Eval(fmt.Sprintf(`globalThis.exiting = document.querySelector(%q);
						globalThis.capture = () => {
							const style = getComputedStyle(exiting);
							globalThis.before = {height: style.height, transform: style.transform,
								top: exiting.getBoundingClientRect().top};
							};`, selector), nil)
				var x, y float64
				if kind == "swipe" {
					r := s.Rect(selector, 0)
					x, y = r.X+20, r.Y+20
					s.Run(input.DispatchMouseEvent(input.MouseMoved, x, y),
						input.DispatchMouseEvent(input.MousePressed, x, y).WithButton(input.Left).WithClickCount(1),
						input.DispatchMouseEvent(input.MouseMoved, x, y+90).WithButton(input.Left).WithButtons(1))
				}
				s.Eval(`capture();`, nil)
				// Only the exit needs animation; capture flushes setup layout.
				if !reduced {
					s.Run(noteAnimate())
				}
				if kind == "swipe" {
					s.Run(input.DispatchMouseEvent(input.MouseReleased, x, y+90).WithButton(input.Left).WithClickCount(1))
				} else {
					s.Eval(`exiting.querySelector('button').click();`, nil)
				}
				noteCheck(t, s, fmt.Sprintf(`exiting.dataset.exit === %q && exiting.dataset.state === 'exiting' &&
						fixture.retained.length === 2 && exiting.style.height === before.height &&
						exiting.style.getPropertyValue('--exit-transform') === before.transform`, kind))
				if reduced {
					noteCheck(t, s, `exiting.getAnimations().length === 0`)
				} else {
					s.Eval(`for (const a of exiting.getAnimations()) { a.pause(); a.currentTime = 0; }
							globalThis.exitStyle = getComputedStyle(exiting);`, nil)
					noteCheck(t, s, `Math.abs(exiting.getBoundingClientRect().top - before.top) < 0.1`)
					switch kind {
					case "normal":
						noteCheck(t, s, `exitStyle.transitionProperty === 'transform, opacity, height' &&
								exitStyle.transitionDuration === '0.4s, 0.4s, 0.4s' &&
								exitStyle.transitionTimingFunction === 'ease, ease, ease'`)
					case "covered":
						noteCheck(t, s, `exitStyle.transitionProperty === 'transform, opacity' &&
								exitStyle.transitionDuration === '0.5s, 0.2s' && exitStyle.transitionTimingFunction === 'ease, ease'`)
					case "swipe":
						noteCheck(t, s, `exitStyle.animationName === 'hi-note-swipe-out' &&
								exitStyle.animationDuration === '0.2s' && exitStyle.animationTimingFunction === 'ease-out'`)
					}
				}
				// Unmounting uses its own clock, even when CSS takes longer
				// or reduced motion removes the animation entirely.
				s.Eval(`fixture.advance(199);`, nil)
				noteCheck(t, s, `exiting.isConnected`)
				s.Eval(`fixture.advance(1);`, nil)
				noteCheck(t, s, `!exiting.isConnected && fixture.retained.length === 2`)
			})
		})
	}
}

func TestNoteSchema(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		n    Note
		want []string
	}{
		{
			name: "message",
			n:    Note{Message: "Saved <item>"},
			want: []string{"Saved &lt;item&gt;", `aria-label="Dismiss"`, `data-duration="0"`},
		},
		{
			name: "description and icon",
			n:    Note{Message: "Saved", Description: "Library <updated>", Icon: "check"},
			want: []string{"Library &lt;updated&gt;", "<hi-icon", "<svg"},
		},
		{
			name: "button",
			n:    Note{Message: "Deleted", Action: Button(42, Text("Undo")), Duration: 10 * time.Second},
			want: []string{"Undo", `data-duration="10000"`, `aria-label="Dismiss"`},
		},
		{
			name: "link",
			n:    Note{Message: "Saved", Action: Link("/next", Text("Open")), Duration: 1500 * time.Microsecond},
			want: []string{`href="/next"`, `data-duration="1"`, `aria-label="Dismiss"`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, page := Render(note{Note: tc.n}.view())
			html := renderNode(t, page)
			for _, want := range tc.want {
				if !strings.Contains(html, want) {
					t.Errorf("missing %q in %s", want, html)
				}
			}
		})
	}
}

func TestNotesDuration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		duration time.Duration
		millis   float64
	}{
		{0, 4000},
		{-time.Second, 4000},
		{1500 * time.Microsecond, 1},
		{10 * time.Second, 10000},
		{30 * 24 * time.Hour, 2592000000},
	} {
		t.Run(tc.duration.String(), func(t *testing.T) {
			runNoteView(t, 800, 600, Note{Duration: tc.duration}, func(s *uitest.Session) {
				s.Run(noteAdd(1, `'timed'`))
				s.Eval(fmt.Sprintf(`fixture.advance(%g);`, tc.millis-0.5), nil)
				noteCheck(t, s, `fixture.retained.length === 1`)
				s.Eval(`fixture.advance(0.5);`, nil)
				noteCheck(t, s, `fixture.retained.length === 0`)
			})
		})
	}
}

func TestNotesActionActivation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		action  View
		dismiss bool
	}{
		{"default dismiss", nil, true},
		{"message button", Button(42, Text("Undo")), true},
		{"message link", Link(42, Text("Undo")), true},
		{"URL button", Button("#next", Text("Open")), true},
		{"URL link", Link("#next", Text("Open")), true},
		{"custom hyperlink", Text("Open").Tag("a").Attr(attr.Href("#next")), true},
		{"custom click handler", Text("Undo").Attr(event.Click(42)), true},
		{"click handler ancestor", HStack(Text("Undo").Class("target")).Attr(event.Click(42)), true},
		{"plain button", Text("Button").Tag("button"), false},
		{"anchor without href", Text("Anchor").Tag("a"), false},
		{"plain text", Text("Text"), false},
		{"input handler", Text("").Tag("textarea").Attr(event.Input(func(string) int { return 42 })), false},
		{"change handler", Text("").Tag("textarea").Attr(event.Change(func(string) int { return 42 })), false},
		{"submit handler", HStack(Text("Submit").Tag("button").Class("target")).Tag("form").Attr(event.Submit(42)), false},
		{"disabled button", Button(42, Text("Undo")).Disabled(true), false},
		{"disabled link", Link("#next", Text("Open")).Disabled(true), false},
		{"aria disabled handler", Text("Undo").Attr(event.Click(42), domi.Name("aria-disabled", "true")), false},
	}
	var actions []View
	for i, tc := range cases {
		if tc.action != nil {
			actions = append(actions, tc.action.Class(fmt.Sprintf("test-action-%d", i)))
		}
	}
	runNoteView(t, 800, 600, Note{Action: VStack(actions...)}, func(s *uitest.Session) {
		s.Eval(`const display = document.querySelector('hi-note-display');
			display.setAttribute('domi-msg-click', 'outside-note');
			display.addEventListener('click', e => e.preventDefault());`, nil)
		for i, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s.Run(noteAdd(1, `'message'`))
				defer func() {
					s.Eval(`fixture.advance(5000);`, nil)
					noteCheck(t, s, `fixture.retained.length === 0 && display.children.length === 0`)
				}()
				selector := fmt.Sprintf(".test-action-%d", i)
				if tc.action == nil {
					selector = "[data-dismiss]"
				}
				s.Eval(fmt.Sprintf(`(() => {
					const action = display.querySelector(%q);
					const target = action.querySelector('.target') || action;
					target.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
				})()`, selector), nil)
				noteCheck(t, s, fmt.Sprintf(`(fixture.retained.length === 0) === %t`, tc.dismiss))
			})
		}
	})
}

func TestNotesRetainedActions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		action     View
		navigation bool
	}{
		{"message button", Button(42, Text("Undo")), false},
		{"message link", Link(42, Text("Undo")), false},
		{"URL link", Link("/action", Text("Open")), true},
		{"URL button", Button("/action", Text("Open")), true},
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
					}
					return nil
				}
				a.onView = func(_ context.Context, render PageRenderer) Page {
					return render(VStack(Button(0, Text("Update")).Class("update"),
						Link("/away", Text("Away")).Class("away"), Text(fmt.Sprint(count)).Class("count")))
				}
				return a, Notify[int](Note{Message: "Rich note", Description: "Supporting text", Icon: "check", Action: tc.action, Duration: 10 * time.Second})
			}, func(u *url.URL) int {
				if u.Path == "/action" {
					return 2
				}
				return 1
			}, func(*url.URL) int { return 0 })
			server := httptest.NewServer(h)
			defer server.Close()
			uitest.RunURL(t, 800, 600, "about:blank", func(s *uitest.Session) {
				s.Run(noteReduceMotion(), chromedp.ActionFunc(func(ctx context.Context) error {
					_, err := cdppage.AddScriptToEvaluateOnNewDocument(`Object.defineProperty(document, 'hidden', {value: true});`).Do(ctx)
					return err
				}), chromedp.Navigate(server.URL), chromedp.WaitReady("hi-note-display", chromedp.ByQuery), notePoll(noteText+` === 'Rich note'`))
				s.Run(chromedp.Click(".update", chromedp.ByQuery), notePoll(`document.querySelector('hi-note-outbox').children.length === 0`),
					chromedp.Click(".away", chromedp.ByQuery), notePoll(`location.pathname === '/away'`), chromedp.NavigateBack(), notePoll(`location.pathname === '/'`))
				noteCheck(t, s, noteText+` === 'Rich note'`)
				s.Eval(`document.querySelector('.update').focus();
					document.querySelector('hi-note-display hi-note :is(button,a)').focus();`, nil)
				s.Run(chromedp.KeyEvent("\r"), notePoll(`document.querySelectorAll('hi-note-display hi-note[data-state=active]').length === 0`))
				if tc.navigation {
					s.Run(notePoll(`location.pathname === '/action'`), chromedp.NavigateBack(), notePoll(`location.pathname === '/'`))
				} else {
					s.Run(notePoll(`document.querySelector('.count').textContent === '1'`))
					noteCheck(t, s, `document.activeElement.classList.contains('update')`)
				}
				noteCheck(t, s, `document.querySelectorAll('hi-note-display hi-note[data-state=active]').length === 0`)
			})
		})
	}
}

func TestNotesRemovedBeforeAdmission(t *testing.T) {
	t.Parallel()
	runNotes(t, 800, 600, func(s *uitest.Session) {
		s.Eval(`fixture.add('removed');
			globalThis.entry = document.querySelector('hi-note-outbox').lastElementChild;
			entry.remove();`, nil)
		noteCheck(t, s, `fixture.retained.length === 0`)
		// No delivery was recorded for the removed entry.
		s.Eval(`document.querySelector('hi-note-outbox').append(entry);`, nil)
		s.Run(notePoll(`fixture.retained.join() === 'removed'`))
	})
}

func TestNotesActionPointerIsolation(t *testing.T) {
	t.Parallel()
	for _, action := range []View{Button(42, Text("Undo")), Link("#action", Text("Open"))} {
		runNoteView(t, 800, 600, Note{Action: action}, func(s *uitest.Session) {
			s.Run(noteAdd(1, `'swipe me'`))
			s.Eval(`globalThis.activations = 0;
				globalThis.action = document.querySelector('hi-note-display hi-note :is(button,a)');
				action.addEventListener('click', () => activations++);
				document.querySelector('hi-note-display hi-text').click();
				action.dispatchEvent(new PointerEvent('pointerdown', {button: 0, isPrimary: true, bubbles: true}));`, nil)
			noteCheck(t, s, `fixture.retained.length === 1 && activations === 0 && !fixture.dragging`)
			r := s.Rect("hi-note-display hi-note", 0)
			x, y := r.X+20, r.Y+10
			s.Run(input.DispatchMouseEvent(input.MouseMoved, x, y),
				input.DispatchMouseEvent(input.MousePressed, x, y).WithButton(input.Left).WithClickCount(1),
				input.DispatchMouseEvent(input.MouseMoved, x, y-20).WithButton(input.Left).WithButtons(1),
				input.DispatchMouseEvent(input.MouseReleased, x, y-20).WithButton(input.Left).WithClickCount(1))
			s.Eval(`action.dispatchEvent(new MouseEvent('click', {bubbles:true, cancelable:true, detail:1}));`, nil)
			noteCheck(t, s, `fixture.retained.length === 1 && activations === 0 && !fixture.dragging`)
			s.Run(input.DispatchMouseEvent(input.MousePressed, x, y).WithButton(input.Left).WithClickCount(1),
				input.DispatchMouseEvent(input.MouseMoved, x, y+60).WithButton(input.Left).WithButtons(1),
				input.DispatchMouseEvent(input.MouseReleased, x, y+60).WithButton(input.Left).WithClickCount(1))
			noteCheck(t, s, `fixture.retained.length === 0 && activations === 0 && !fixture.dragging`)
		})
	}
}
