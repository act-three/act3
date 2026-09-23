package hi_test

import (
	"context"
	"testing"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"ily.dev/act3/hi/internal/uitest"
)

func TestBrowserFixtureIsolation(t *testing.T) {
	// Stay serial so both fixtures deterministically acquire the same tab.
	var first target.ID
	var hover bool
	uitest.Run(t, 600, 400, `<body><script>
		const fixtureLexical = 1;
		globalThis.fixtureGlobal = 1;
		customElements.define('fixture-element', class extends HTMLElement {});
		</script>`, func(s *uitest.Session) {
		s.Eval(`matchMedia('(hover: hover)').matches`, &hover)
		s.Run(chromedp.ActionFunc(func(ctx context.Context) error {
			first = chromedp.FromContext(ctx).Target.TargetID
			return nil
		}), chromedp.EmulateViewport(300, 200, chromedp.EmulateScale(2), chromedp.EmulateTouch),
			emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{{Name: "prefers-reduced-motion", Value: "reduce"}}),
			input.DispatchMouseEvent(input.MouseMoved, 20, 20))
	})
	uitest.Run(t, 600, 400, `<body style="margin:0"><button style="width:100px;height:100px">test</button>`, func(s *uitest.Session) {
		s.Run(chromedp.ActionFunc(func(ctx context.Context) error {
			if got := chromedp.FromContext(ctx).Target.TargetID; got != first {
				t.Fatal("isolation check did not reuse the tab")
			}
			return nil
		}))
		var clean map[string]bool
		s.Eval(`({
			lexical: typeof fixtureLexical === 'undefined',
			global: typeof fixtureGlobal === 'undefined',
			element: customElements.get('fixture-element') === undefined,
			width: innerWidth === 600, height: innerHeight === 400, scale: devicePixelRatio === 1,
			touch: navigator.maxTouchPoints === 0,
			motion: !matchMedia('(prefers-reduced-motion: reduce)').matches,
			pointer: !document.querySelector('button').matches(':hover'),
		})`, &clean)
		var restoredHover bool
		s.Eval(`matchMedia('(hover: hover)').matches`, &restoredHover)
		clean["hover"] = restoredHover == hover
		for name, ok := range clean {
			if !ok {
				t.Errorf("fixture inherited %s state", name)
			}
		}
	})
}
