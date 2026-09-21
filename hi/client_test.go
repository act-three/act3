package hi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	cdppage "github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"ily.dev/act3/hi/internal/uitest"
	"ily.dev/domi"
)

func TestClientModuleBrowser(t *testing.T) {
	for _, order := range []string{"Hi only", "Domi before Hi", "Domi after Hi"} {
		t.Run(order, func(t *testing.T) {
			const prefix = "/-/client-test"
			h := Handler(
				func(context.Context, *url.URL) (*stubApp, domi.Cmd[struct{}]) {
					return &stubApp{view: Text("page")}, nil
				},
				func(*url.URL) struct{} { return struct{}{} },
				func(*url.URL) struct{} { return struct{}{} },
				domi.InternalURLPrefix(prefix),
				domi.Document(func(title string, body domi.Node) domi.Node {
					var before, after domi.Node
					if order == "Domi before Hi" {
						before = domi.ClientModule(prefix)
					} else if order == "Domi after Hi" {
						after = domi.ClientModule(prefix)
					}
					return domi.Tag("html")(
						domi.Tag("head")(before, ClientModule(prefix), after), body,
					)
				}),
			)
			domiRequested := make(chan struct{}, 1)
			var domiRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case prefix + "/" + domiClientModueName:
					domiRequests.Add(1)
					select {
					case domiRequested <- struct{}{}:
					default:
					}
				case clientJSPath(prefix):
					// Domi must be discovered without first downloading Hi.
					select {
					case <-domiRequested:
					case <-r.Context().Done():
						return
					case <-time.After(5 * time.Second):
						t.Error("Domi was not requested before Hi's response")
						http.Error(w, "module fetch waterfall", http.StatusGatewayTimeout)
						return
					}
				}
				h.ServeHTTP(w, r)
			}))
			defer server.Close()
			uitest.RunURL(t, 800, 600, "about:blank", func(s *uitest.Session) {
				s.Run(chromedp.ActionFunc(func(ctx context.Context) error {
					_, err := cdppage.AddScriptToEvaluateOnNewDocument(`
						globalThis.moduleErrors = [];
						addEventListener('error', e => moduleErrors.push(e.message));
						addEventListener('unhandledrejection', e => moduleErrors.push(String(e.reason)));
						globalThis.eventSources = 0;
						globalThis.EventSource = class extends EventSource {
							constructor(...args) { super(...args); eventSources++; }
						};
					`).Do(ctx)
					return err
				}), chromedp.Navigate(server.URL), chromedp.WaitReady("hi-note-display", chromedp.ByQuery))
				s.Run(chromedp.Poll(`!!customElements.get('hi-note-display') && eventSources === 1`, nil))
				var errors []string
				s.Eval(`moduleErrors`, &errors)
				if len(errors) != 0 {
					t.Errorf("module errors: %v", errors)
				}
				if got := domiRequests.Load(); got != 1 {
					t.Errorf("Domi module requests = %d, want 1", got)
				}
			})
		})
	}
}
