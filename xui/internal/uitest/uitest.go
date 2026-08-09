// Package uitest drives a headless Chrome-compatible browser to measure
// rendered xui layouts.
// Markup assertions cannot catch CSS regressions;
// the unit under test here is computed geometry.
package uitest

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// The test binary shares one browser process across all Run calls:
// each call opens a tab in it rather than paying — and risking —
// a browser cold start per test.
var browser struct {
	once sync.Once
	ctx  context.Context
	err  error
	stop context.CancelFunc
}

func startBrowser() {
	// The browser renders only the harness's own local fixture files, so
	// the Chrome sandbox buys nothing here — and it cannot start at all
	// on runners that restrict unprivileged user namespaces (GitHub's
	// Ubuntu 24.04 images).
	// The browser's own output is captured, with verbose logging turned
	// on, so a failed or hung start reports what the browser was doing —
	// with timestamps — not just that it timed out.
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
			chromedp.Flag("v", "1"),
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
	path := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatalf("write page: %v", err)
	}

	browser.once.Do(startBrowser)
	if browser.err != nil {
		if strings.Contains(browser.err.Error(), "executable file not found") {
			t.Skipf("no browser available: %v", browser.err)
		}
		t.Fatalf("start browser: %v", browser.err)
	}

	ctx, cancel := chromedp.NewContext(browser.ctx)
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTimeout()

	u := url.URL{Scheme: "file", Path: path}
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(int64(w), int64(h)),
		chromedp.Navigate(u.String()),
	)
	if err != nil {
		t.Fatalf("load page: %v", err)
	}
	fn(&Session{t: t, ctx: ctx})
}

// Session is a loaded page ready to be measured.
type Session struct {
	t   *testing.T
	ctx context.Context
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
