package hi_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"
	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/hi"
	"ily.dev/act3/hi/internal/uitest"
)

func TestPresentationClosedContent(t *testing.T) {
	for _, tc := range []struct {
		name string
		view func(hi.PresentationState, hi.View) hi.View
	}{
		{"popover", hi.Text("Trigger").Popover},
		{"dialog", hi.Text("Trigger").Dialog},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view := func(open bool) string {
				return render(t, tc.view(hi.Present(open, 7), hi.Text("Retained content")))
			}
			closed, open := view(false), view(true)
			zero := render(t, tc.view(hi.PresentationState{}, hi.Text("Retained content")))
			if !strings.Contains(zero, `data-hi-open="false"`) || !strings.Contains(zero, "Retained content") {
				t.Fatal("zero presentation state must retain closed content")
			}
			if !strings.Contains(closed, "Retained content") {
				t.Fatal("closed presentation discarded its content")
			}
			if strings.ReplaceAll(closed, `data-hi-open="false"`, `data-hi-open="true"`) != open {
				t.Fatalf("changing open state changed more than the state attribute:\nclosed: %s\nopen: %s", closed, open)
			}
		})
	}
}

func TestPresentationContentTitle(t *testing.T) {
	for _, kind := range []struct {
		name string
		view func(hi.PresentationState, hi.View) hi.View
	}{
		{"popover", hi.Text("Trigger").Popover},
		{"dialog", hi.Text("Trigger").Dialog},
	} {
		for _, tc := range []struct {
			name    string
			content hi.View
			title   string
		}{
			{"untitled", hi.Text("Content"), ""},
			{"titled", hi.Text("Content").Title("Settings"), "Settings"},
			{"wrapped", hi.VStack(hi.Text("Content").Title("Settings")).Padding(), "Settings"},
			{"nested", hi.Text("Trigger").Dialog(hi.Present(false, 7), hi.Text("Nested").Title("Inner")), ""},
		} {
			t.Run(kind.name+"/"+tc.name, func(t *testing.T) {
				for _, open := range []bool{false, true} {
					v := kind.view(hi.Present(open, 7), tc.content)
					if title, _ := hi.Render(v); title != "" {
						t.Errorf("presentation title escaped to page: %q", title)
					}
					if title, _ := hi.Render(v.Title("Page")); title != "Page" {
						t.Errorf("enclosing title = %q, want Page", title)
					}
					html := render(t, v)
					host := regexp.MustCompile(`<[^>]*data-hi-presentation="` + kind.name + `"[^>]*>`).FindString(html)
					if tc.title == "" {
						if strings.Contains(host, "aria-label=") {
							t.Errorf("untitled presentation has a name: %s", host)
						}
					} else if !strings.Contains(host, `aria-label="`+tc.title+`"`) {
						t.Errorf("content title missing from presentation: %s", host)
					}
				}
			})
		}
	}
}

func TestPresentationEnclosingOpenState(t *testing.T) {
	kinds := []struct {
		name string
		view func(hi.PresentationState, hi.View) hi.View
	}{
		{"popover", hi.Text("Trigger").Popover},
		{"dialog", hi.Text("Trigger").Dialog},
	}
	states := regexp.MustCompile(`data-hi-open="(true|false)"`)
	for _, outer := range kinds {
		for _, inner := range kinds {
			for bits := range 8 {
				a, b, c := bits&1 != 0, bits&2 != 0, bits&4 != 0
				t.Run(fmt.Sprintf("%s/%s/%d", outer.name, inner.name, bits), func(t *testing.T) {
					v := hi.VStack(
						outer.view(hi.Present(a, 7), hi.HStack(
							inner.view(hi.Present(b, 7), hi.Text("Nested trigger").Popover(hi.Present(c, 7), hi.Text("Nested"))),
						).Padding()),
						hi.Text("Sibling trigger").Popover(hi.Present(true, 7), hi.Text("Sibling")),
					)
					got := states.FindAllStringSubmatch(render(t, v), -1)
					want := []bool{a, a && b, a && b && c, true}
					if len(got) != len(want) {
						t.Fatalf("rendered %d presentations, want %d", len(got), len(want))
					}
					for i, open := range want {
						if got[i][1] != fmt.Sprint(open) {
							t.Errorf("presentation %d open = %s, want %t", i, got[i][1], open)
						}
					}
				})
			}
		}
	}
}

func TestPresentationAttachmentPresets(t *testing.T) {
	for _, kind := range []string{"popover", "menu"} {
		t.Run(kind, func(t *testing.T) {
			base := func(id string) hi.View {
				trigger := hi.HStack(hi.Text(id)).Padding(hi.Edges(3)).
					Frame(hi.Width(80), hi.Height(40)).Attr(attr.ID(id)).Background(hi.Red)
				content := hi.Text("Content").Frame(hi.Width(40), hi.Height(20)).Background(hi.Blue).Class("content")
				if kind == "menu" {
					return trigger.Menu(hi.Present(true, 7), content)
				}
				return trigger.Popover(hi.Present(true, 7), content)
			}
			for _, layout := range []string{"root", "siblings"} {
				t.Run(layout, func(t *testing.T) {
					v, ids := base("first"), []string{"first"}
					if layout == "siblings" {
						v, ids = hi.HStack(v, base("second")).Gap(100), []string{"first", "second"}
					}
					stage(t, v, func(s *uitest.Session) {
						s.Eval(`document.querySelectorAll('[popover]').forEach(e => e.showPopover())`, nil)
						for i, id := range ids {
							trigger, popover, content := s.Rect("#"+id, 0), s.Rect("[popover]", i), s.Rect(".content", i)
							within(t, id+" receiver width", trigger.W, 80, .1)
							within(t, id+" receiver height", trigger.H, 40, .1)
							if layout == "root" {
								within(t, "root receiver x", trigger.X, 260, .1)
								within(t, "root receiver y", trigger.Y, 180, .1)
							}
							x := trigger.X
							if kind == "popover" {
								x += (trigger.W - popover.W) / 2
							}
							within(t, id+" attachment x", popover.X, x, .1)
							within(t, id+" attachment y", popover.Y, trigger.Bottom(), .1)
							within(t, id+" exterior padding", content.Y, popover.Y+4, .1)
							within(t, id+" content width", content.W, 40, .1)
							within(t, id+" content height", content.H, 20, .1)
						}
					})
				})
			}
		})
	}
}

func TestPresentationContentModifiers(t *testing.T) {
	for _, kind := range []struct {
		name string
		view func(hi.PresentationState, hi.View) hi.View
	}{
		{"popover", hi.Text("Trigger").Popover},
		{"dialog", hi.Text("Trigger").Dialog},
	} {
		t.Run(kind.name, func(t *testing.T) {
			content := hi.Text("Content").
				Background(hi.Blue).
				Font(hi.SizeEm(20, 1.5), hi.Bold).
				Attr(attr.ID("presented"), domi.Name("data-application", "value")).
				BorderStroke(2, hi.Red)
			v := kind.view(hi.Present(true, 7), content)
			stage(t, v, func(s *uitest.Session) {
				s.Eval(`document.querySelectorAll('[data-hi-presentation]').forEach(e => {
					if (e.localName === 'dialog') e.showModal(); else e.showPopover();
				})`, nil)
				var failed []string
				s.Eval(`(() => {
					const host = document.querySelector('[data-hi-presentation]');
					const surface = document.querySelector('#presented'), style = getComputedStyle(surface);
					const textStyle = getComputedStyle(host.querySelector('hi-text'));
					return [
						['attributes', host.contains(surface) && surface.dataset.application === 'value'],
						['unpainted boundary', getComputedStyle(host).backgroundColor === 'rgba(0, 0, 0, 0)'],
						['background', style.backgroundColor !== 'rgba(0, 0, 0, 0)'],
						['typography', style.fontSize === '20px' && style.fontWeight === '700'],
						['inherited typography', textStyle.fontSize === '20px' && textStyle.fontWeight === '700'],
						['stroke', getComputedStyle(surface, '::after').boxShadow.includes('2px')],
						['positioning', getComputedStyle(host).position === 'fixed'],
					].filter(([, passed]) => !passed).map(([name]) => name);
				})()`, &failed)
				if len(failed) != 0 {
					t.Errorf("presentation modifiers failed: %v", failed)
				}
			})
		})
	}
}

func TestPresentationEnclosingViews(t *testing.T) {
	for _, kind := range []struct {
		name string
		view func(hi.PresentationState, hi.View) hi.View
	}{
		{"popover", hi.Text("Trigger").Popover},
		{"dialog", hi.Text("Trigger").Dialog},
	} {
		t.Run(kind.name, func(t *testing.T) {
			v := hi.HStack(kind.view(hi.Present(true, 7), hi.Text("Content").
				Background(hi.Blue).Attr(attr.ID("presented")))).
				Padding(hi.Edges(10)).Attr(attr.ID("padding")).
				Frame(hi.Width(240), hi.Height(120)).Attr(attr.ID("frame")).Background(hi.Red)
			stage(t, v, func(s *uitest.Session) {
				var correct bool
				s.Eval(`(() => {
					const host = document.querySelector('[data-hi-presentation]');
					const surface = document.querySelector('#presented');
					return host.contains(surface) &&
						host.closest('#padding').localName === 'hi-padding' &&
						host.closest('#frame').localName === 'hi-frame' &&
						getComputedStyle(surface).backgroundColor !==
						getComputedStyle(document.querySelector('#frame')).backgroundColor;
				})()`, &correct)
				if !correct {
					t.Error("enclosing stack, padding, frame, or paint entered the presentation")
				}
			})
		})
	}
}

func TestPresentationStackSpacing(t *testing.T) {
	for _, kind := range []string{"popover", "menu", "dialog"} {
		for _, axis := range []struct {
			name  string
			stack func(...hi.View) hi.StackView
		}{{"horizontal", hi.HStack}, {"vertical", hi.VStack}} {
			t.Run(kind+"/"+axis.name, func(t *testing.T) {
				first := hi.Text("First").Frame(hi.Width(40), hi.Height(40)).Attr(attr.ID("first"))
				second := hi.Text("Second").Frame(hi.Width(40), hi.Height(40)).Attr(attr.ID("second"))
				p := hi.Present(true, 7)
				var children hi.View
				switch kind {
				case "popover":
					children = hi.Group(first.Popover(p, hi.Text("Content")), second.Popover(p, hi.Text("More content")))
				case "menu":
					children = hi.Group(first.Menu(p, hi.Text("Content")), second.Menu(p, hi.Text("More content")))
				case "dialog":
					children = hi.Group(first.Dialog(p, hi.Text("Content")), second.Dialog(p, hi.Text("More content")))
				}
				stage(t, axis.stack(children).FixedSize().Attr(attr.ID("stack")), func(s *uitest.Session) {
					for _, open := range []bool{false, true} {
						if open {
							s.Eval(`document.querySelectorAll('[data-hi-presentation]').forEach(e => {
								if (e.localName === 'dialog') e.showModal(); else e.showPopover();
							})`, nil)
							var shown int
							s.Eval(`document.querySelectorAll(':modal, :popover-open').length`, &shown)
							if shown != 2 {
								t.Errorf("reused presentation state opened %d views, want 2", shown)
							}
						}
						first, second, stack := s.Rect("#first", 0), s.Rect("#second", 0), s.Rect("#stack", 0)
						gap, extent := second.X-first.Right(), stack.W
						if axis.name == "vertical" {
							gap, extent = second.Y-first.Bottom(), stack.H
						}
						within(t, fmt.Sprintf("gap (open=%t)", open), gap, 8, .1)
						within(t, fmt.Sprintf("extent (open=%t)", open), extent, 88, .1)
					}
				})
			})
		}
	}
}

func TestDialogCentered(t *testing.T) {
	for _, tc := range []struct {
		name string
		wrap func(hi.View) hi.View
	}{
		{"root receiver", func(v hi.View) hi.View { return v }},
		{"overlay", func(v hi.View) hi.View {
			return hi.Text("Trigger").OverlayAt(hi.BottomLeading, hi.TopLeading, hi.HStack(v))
		}},
		{"baseline overlay", func(v hi.View) hi.View {
			return hi.Text("Trigger").OverlayAt(hi.FirstBaseline, hi.FirstBaseline, v)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.wrap(hi.Text("Trigger").Dialog(hi.Present(true, 7), hi.Text("Dialog").Frame(hi.Width(120), hi.Height(80)).Class("content")))
			if strings.Contains(render(t, v), "position-anchor:") {
				t.Error("dialog used an overlay anchor")
			}
			stage(t, v, func(s *uitest.Session) {
				s.Eval(`document.querySelector('dialog').showModal()`, nil)
				r := s.Rect(".content", 0)
				within(t, "dialog x", r.X, 240, .1)
				within(t, "dialog y", r.Y, 160, .1)
				within(t, "dialog width", r.W, 120, .1)
				within(t, "dialog height", r.H, 80, .1)
			})
		})
	}
}

func TestDialogAvailableSpace(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content hi.View
	}{
		{"short text", hi.Text("Short")},
		{"wrapping text", hi.Text(strings.Repeat("Some wrapping text. ", 40))},
		{"rigid", hi.Text("Rigid").Frame(hi.Width(100), hi.Height(1060))},
		{"wide rigid", hi.Text("Wide").Frame(hi.Width(1000), hi.Height(80))},
		{"horizontal fill", hi.HStack(hi.Text("Leading"), hi.Spacer())},
		{"vertical fill", hi.VStack(hi.Text("Top"), hi.Spacer())},
		{"both fills", hi.Blue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := tc.content.Class("content")
			var root uitest.Rect
			stage(t, content.Padding(hi.Edges(12i)).Padding(hi.Edges(12)), func(s *uitest.Session) { root = s.Rect(".content", 0) })
			stage(t, hi.Text("Trigger").Dialog(hi.Present(true, 7), content), func(s *uitest.Session) {
				s.Eval(`document.querySelector('dialog').showModal()`, nil)
				r := s.Rect(".content", 0)
				within(t, "content x", r.X, root.X, .1)
				within(t, "content y", r.Y, root.Y, .1)
				within(t, "content width", r.W, root.W, .1)
				within(t, "content height", r.H, root.H, .1)
				r = s.Rect("dialog", 0)
				within(t, "boundary x", r.X, 12, .1)
				within(t, "boundary y", r.Y, 12, .1)
				within(t, "boundary width", r.W, 576, .1)
				within(t, "boundary height", r.H, 376, .1)
			})
		})
	}
	stage(t, hi.Text("Trigger").Dialog(hi.Present(true, 7), hi.ScrollView(hi.Vertical,
		hi.Text("Tall").Frame(hi.Height(1060)))),
		func(s *uitest.Session) {
			s.Eval(`document.querySelector('dialog').showModal()`, nil)
			r := s.Rect("hi-scroll", 0)
			within(t, "scroll width", r.W, 552, .1)
			within(t, "scroll height", r.H, 352, .1)
		})
}

func TestPresentationFillPreference(t *testing.T) {
	scroll := hi.ScrollView(hi.Vertical, hi.Text("Content"))
	for _, tc := range []struct {
		name    string
		content hi.View
		order   string
	}{
		{"natural", hi.Text("Content"), ""},
		{"vertical", scroll.Frame(hi.Width(200)), "most-height"},
		{"horizontal", scroll.Frame(hi.Height(200)), "most-width"},
		{"both", scroll, "most-height"},
		{"fixed", scroll.Frame(hi.Width(200), hi.Height(200)), ""},
		{"rigid", scroll.FixedSize(), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := hi.Text("Trigger").Menu(hi.Present(true, 7), tc.content)
			html := render(t, v)
			for _, order := range []string{"most-height", "most-width"} {
				if strings.Contains(html, "position-try-order:"+order) != (tc.order == order) {
					t.Errorf("content fill contract did not select %q:\n%s", tc.order, html)
				}
			}
		})
	}
}

type presentationMsg int

const (
	presentationToggle presentationMsg = iota + 1
	presentationDismiss
)

type presentationApp struct {
	kind       string
	open       bool
	nested     bool
	overlay    bool
	dismissals int
}

func (a *presentationApp) Update(_ context.Context, msg presentationMsg) domi.Cmd[presentationMsg] {
	switch msg {
	case presentationToggle:
		a.open = !a.open
	case presentationDismiss:
		a.dismissals++
		// Refusing the first request tests that browser gestures cannot
		// override the application's authoritative flag.
		a.open = a.dismissals == 1
	}
	return nil
}

func (a *presentationApp) View(_ context.Context, render hi.PageRenderer) hi.Page {
	content := hi.HTML(domi.Tag("input", attr.ID("retained-input"))()).
		Frame(hi.Width(180), hi.Height(50)).Title("Presentation test")
	if a.nested {
		content = hi.Text("Nested trigger").Popover(hi.Present(true, presentationToggle), content)
	}
	trigger := hi.Button(presentationToggle, hi.Text("Open")).Attr(attr.ID("trigger"))
	p := hi.Present(a.open, presentationDismiss)
	var presented hi.View
	if a.kind == "dialog" {
		if a.overlay {
			presented = hi.Text("Base").Overlay(hi.Center, trigger.Dialog(p, content))
		} else {
			presented = trigger.Dialog(p, content)
		}
	} else {
		presented = trigger.Popover(p, content)
	}
	return render(hi.VStack(presented,
		hi.Text(fmt.Sprint(a.dismissals)).Attr(attr.ID("dismissals")),
	))
}

func (*presentationApp) Subscriptions(context.Context) domi.Sub[presentationMsg] { return nil }

func (*presentationApp) Preview(context.Context, *url.URL, hi.PreviewRenderer) hi.Preview {
	return hi.Preview{}
}

func TestPresentationApplicationState(t *testing.T) {
	for _, kind := range []string{"popover", "dialog", "overlay dialog"} {
		t.Run(kind, func(t *testing.T) {
			overlay := kind == "overlay dialog"
			if overlay {
				kind = "dialog"
			}
			h := hi.Handler(
				func(context.Context, *url.URL) (*presentationApp, domi.Cmd[presentationMsg]) {
					return &presentationApp{kind: kind, overlay: overlay}, nil
				},
				func(*url.URL) presentationMsg { return 0 },
				func(*url.URL) presentationMsg { return 0 },
			)
			server := httptest.NewServer(h)
			defer server.Close()
			uitest.RunURL(t, 800, 600, server.URL, func(s *uitest.Session) {
				selector := `[data-hi-presentation="` + kind + `"]`
				opened := `document.querySelector('` + selector + `').matches('` + map[string]string{
					"popover": ":popover-open",
					"dialog":  ":modal",
				}[kind] + `')`
				s.Run(chromedp.WaitReady("#trigger", chromedp.ByQuery))
				var label string
				s.Eval(`document.querySelector('`+selector+`').getAttribute('aria-label')`, &label)
				if label != "Presentation test" {
					t.Errorf("native presentation accessible name = %q", label)
				}
				var initialOpen bool
				s.Eval(opened, &initialOpen)
				if initialOpen {
					t.Fatal("presentation opened despite false application state")
				}
				s.Run(chromedp.Click("#trigger", chromedp.ByQuery), chromedp.Poll(opened, nil))
				s.Eval(`globalThis.retainedInput = document.querySelector('#retained-input'); retainedInput.value = 'draft';`, nil)
				if kind == "dialog" {
					s.Eval(`globalThis.dialogDismissRequests = 0;
						document.querySelector('dialog > hi-dismiss').addEventListener('click', () => dialogDismissRequests++);`, nil)
					// Cover both the blank host and ::backdrop outside its inset.
					s.Run(chromedp.MouseClickXY(20, 20), chromedp.MouseClickXY(5, 5))
					var requests int
					s.Eval(`dialogDismissRequests`, &requests)
					if requests != 0 {
						t.Errorf("backdrop clicks requested %d dismissals", requests)
					}
				}
				s.Run(chromedp.KeyEvent("\x1b"), chromedp.Poll(`document.querySelector('#dismissals').textContent === '1'`, nil))
				var stillOpen bool
				s.Eval(opened, &stillOpen)
				if !stillOpen {
					t.Fatal("browser closed presentation after application refused dismissal")
				}
				s.Run(chromedp.KeyEvent("\x1b"), chromedp.Poll(`document.querySelector('#dismissals').textContent === '2' && !(`+opened+`)`, nil))
				s.Run(chromedp.Click("#trigger", chromedp.ByQuery), chromedp.Poll(opened, nil))
				var retained bool
				s.Eval(`document.querySelector('#retained-input') === retainedInput && retainedInput.value === 'draft'`, &retained)
				if !retained {
					t.Fatal("closing and reopening replaced the presentation content or lost input state")
				}
				if kind == "popover" {
					// Click outside without sending a second application message.
					s.Run(chromedp.MouseClickXY(5, 5),
						chromedp.Poll(`document.querySelector('#dismissals').textContent === '3' && !(`+opened+`)`, nil))
				}
			})
		})
	}
}

func TestPresentationNestedApplicationState(t *testing.T) {
	h := hi.Handler(
		func(context.Context, *url.URL) (*presentationApp, domi.Cmd[presentationMsg]) {
			return &presentationApp{kind: "dialog", nested: true}, nil
		},
		func(*url.URL) presentationMsg { return 0 },
		func(*url.URL) presentationMsg { return 0 },
	)
	server := httptest.NewServer(h)
	defer server.Close()
	uitest.RunURL(t, 800, 600, server.URL, func(s *uitest.Session) {
		const shown = `document.querySelector('dialog').matches(':modal') &&
			document.querySelector('[popover]').matches(':popover-open')`
		const hidden = `Array.from(document.querySelectorAll('[data-hi-presentation]')).every(e =>
			e.dataset.hiOpen === 'false' && !e.matches(':modal, :popover-open'))`
		s.Run(chromedp.WaitReady("#trigger", chromedp.ByQuery), chromedp.Poll(hidden, nil),
			chromedp.Click("#trigger", chromedp.ByQuery), chromedp.Poll(shown, nil))
		s.Eval(`globalThis.retainedInput = document.querySelector('#retained-input');
			retainedInput.value = 'draft';`, nil)
		// The inner popover's dismissal changes only the outer dialog's
		// flag. Its own application flag stays true across both renders.
		s.Run(chromedp.KeyEvent("\x1b"), chromedp.Poll(hidden, nil),
			chromedp.Click("#trigger", chromedp.ByQuery), chromedp.Poll(shown, nil))
		var retained bool
		s.Eval(`document.querySelector('#retained-input') === retainedInput &&
			retainedInput.value === 'draft'`, &retained)
		if !retained {
			t.Fatal("hiding the enclosing presentation lost the nested content's state")
		}
		s.Eval(`globalThis.outerDismissRequests = 0;
			document.querySelector('dialog > hi-dismiss').addEventListener('click', () => outerDismissRequests++);`, nil)
		s.Run(chromedp.MouseClickXY(5, 5), chromedp.Poll(hidden, nil))
		var requests int
		s.Eval(`outerDismissRequests`, &requests)
		if requests != 0 {
			t.Error("backdrop click dismissed the dialog beneath the topmost popover")
		}
	})
}
