// Package uitest drives a headless Chrome-compatible browser to measure
// rendered hi layouts.
// Markup assertions cannot catch CSS regressions;
// the unit under test here is computed geometry.
package uitest

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

// Fixtures share a browser and reuse tabs, but navigate to a fresh document
// for every Run. Live pages get dedicated tabs because their tests can
// install hooks that persist across navigations.
var browser struct {
	once sync.Once
	ctx  context.Context
	err  error
	stop context.CancelFunc
	mu   sync.Mutex
	idle []tab
}

type tab struct {
	ctx   context.Context
	stop  context.CancelFunc
	w, h  int
	dirty bool
}

func startBrowser() {
	// The browser renders only the harness's own local fixtures, so
	// the Chrome sandbox buys nothing here — and it cannot start at all
	// on runners that restrict unprivileged user namespaces (GitHub's
	// Ubuntu 24.04 images).
	// Capture browser diagnostics so a failed or hung start reports
	// more than a timeout.
	// The startup budget is paid once per test binary and is a cap, not
	// a wait, so it can afford to ride out the CPU variance of a noisy
	// CI VM, where a healthy start plausibly reaches the 20-second
	// default.
	var output lockedBuffer
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.NoSandbox,
			chromedp.WSURLReadTimeout(60*time.Second),
			chromedp.Flag("enable-logging", "stderr"),
			chromedp.CombinedOutput(&output),
		)...)
	ctx, cancel := chromedp.NewContext(allocCtx)
	start := time.Now()
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		cancelAlloc()
		err = fmt.Errorf("%w (after %v, loadavg %s)", err, time.Since(start).Round(time.Millisecond), loadavg())
		if s := output.String(); s != "" {
			err = fmt.Errorf("%w\nbrowser output:\n%s", err, s)
		}
		browser.err = err
		return
	}
	browser.ctx = ctx
	browser.stop = func() {
		cancel()
		cancelAlloc()
	}
}

// loadavg reports the system load averages, so that a browser start
// timeout on a saturated CI runner is distinguishable from a hang.
func loadavg() string {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "unavailable"
	}
	return strings.TrimSpace(string(b))
}

// lockedBuffer is a strings.Builder safe for use as chromedp's
// combined-output writer, which is written from the browser's copy
// goroutine.
type lockedBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// Main runs the tests and then shuts down the shared browser.
// A test package using this harness must call it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(uitest.Main(m)) }
func Main(m *testing.M) int {
	code := m.Run()
	if browser.stop != nil {
		browser.stop()
	}
	return code
}

// Run renders html in a browser tab with a w×h viewport and hands the
// loaded page to fn.
// It skips the test when no Chrome-compatible browser is available.
func Run(t *testing.T, w, h int, html string, fn func(*Session)) {
	t.Helper()
	runURL(t, w, h, "about:blank", true, func(s *Session) {
		s.Run(chromedp.ActionFunc(func(ctx context.Context) error {
			frame, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frame.Frame.ID, html).Do(ctx)
		}), chromedp.Poll(`document.readyState === 'complete'`, nil, chromedp.WithPollingInterval(time.Millisecond)))
		s.dirty = false
		fn(s)
	})
}

// RunURL opens a live page in a browser tab with a w×h viewport.
func RunURL(t *testing.T, w, h int, pageURL string, fn func(*Session)) {
	t.Helper()
	runURL(t, w, h, pageURL, false, fn)
}

func runURL(t *testing.T, w, h int, pageURL string, reuse bool, fn func(*Session)) {
	t.Helper()
	browser.once.Do(startBrowser)
	if browser.err != nil {
		if strings.Contains(browser.err.Error(), "executable file not found") {
			t.Skipf("no browser available: %v", browser.err)
		}
		t.Fatalf("start browser: %v", browser.err)
	}

	var page tab
	if reuse {
		browser.mu.Lock()
		if n := len(browser.idle); n > 0 {
			page = browser.idle[n-1]
			browser.idle = browser.idle[:n-1]
		}
		browser.mu.Unlock()
	}
	if page.ctx == nil {
		var opts []chromedp.ContextOption
		// Linux Chrome throttles background tabs' animation frames even with
		// focus emulation. Separate windows keep every fixture rendering;
		// macOS does not need them and creates tabs substantially faster.
		if runtime.GOOS != "darwin" {
			createCtx, cancelCreate := context.WithTimeout(browser.ctx, 30*time.Second)
			id, err := target.CreateTarget("about:blank").WithNewWindow(true).
				Do(cdp.WithExecutor(createCtx, chromedp.FromContext(browser.ctx).Browser))
			cancelCreate()
			if err != nil {
				t.Fatalf("create window: %v", err)
			}
			opts = append(opts, chromedp.WithTargetID(id))
		}
		page.ctx, page.stop = chromedp.NewContext(browser.ctx, opts...)
		page.dirty = true
		startup := time.AfterFunc(30*time.Second, page.stop)
		// The protocol reader must outlive each individual test's timeout.
		// Focus emulation keeps parallel pages active without stealing focus.
		err := chromedp.Run(page.ctx, emulation.SetFocusEmulationEnabled(true))
		startup.Stop()
		if err != nil {
			page.stop()
			t.Fatalf("create tab: %v", err)
		}
	}
	ctx, cancelTimeout := context.WithTimeout(page.ctx, 30*time.Second)
	defer func() {
		if reuse && ctx.Err() == nil && !t.Failed() {
			browser.mu.Lock()
			browser.idle = append(browser.idle, page)
			browser.mu.Unlock()
		} else {
			page.stop()
		}
		cancelTimeout()
	}()

	var reset chromedp.Tasks
	// Eval only affects the document, which navigation replaces. Run can
	// also change persistent emulation and input state.
	if page.dirty {
		reset = append(reset, emulation.SetEmulatedMedia(), input.DispatchMouseEvent(input.MouseMoved, -1, -1))
	}
	if page.dirty || page.w != w || page.h != h {
		reset = append(reset, chromedp.EmulateViewport(int64(w), int64(h)))
	}
	page.w, page.h = w, h
	err := chromedp.Run(ctx, append(reset, chromedp.Navigate(pageURL))...)
	if err != nil {
		t.Fatalf("load page: %v", err)
	}
	s := &Session{t: t, ctx: ctx}
	defer func() { page.dirty = s.dirty }()
	fn(s)
}

// Session is a loaded page ready to be measured.
type Session struct {
	t     *testing.T
	ctx   context.Context
	dirty bool
}

// Rect is an element's border box in page coordinates.
type Rect struct {
	X, Y, W, H float64
}

// Right is the x coordinate of the box's right edge.
func (r Rect) Right() float64 { return r.X + r.W }

// Bottom is the y coordinate of the box's bottom edge.
func (r Rect) Bottom() float64 { return r.Y + r.H }

// Rect measures the nth match of a CSS selector, failing the test when the
// element does not exist.
func (s *Session) Rect(sel string, n int) Rect {
	s.t.Helper()
	js := fmt.Sprintf(`(() => {
		const e = document.querySelectorAll(%s)[%d];
		if (!e) return null;
		const r = e.getBoundingClientRect();
		return {X: r.x, Y: r.y, W: r.width, H: r.height};
	})()`, jsString(sel), n)
	var r *Rect
	s.Eval(js, &r)
	if r == nil {
		s.t.Fatalf("no element %d matching %q", n, sel)
	}
	return *r
}

// Eval runs a JavaScript expression and decodes its result into out.
func (s *Session) Eval(js string, out any) {
	s.t.Helper()
	if err := chromedp.Run(s.ctx, chromedp.Evaluate(js, out)); err != nil {
		s.t.Fatalf("evaluate %s: %v", js, err)
	}
}

func jsString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Run executes browser actions in this session.
// Actions that install persistent browser hooks belong in a RunURL session.
func (s *Session) Run(actions ...chromedp.Action) {
	s.t.Helper()
	s.dirty = true
	if err := chromedp.Run(s.ctx, actions...); err != nil {
		s.t.Fatal(err)
	}
}
