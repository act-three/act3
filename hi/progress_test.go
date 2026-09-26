package hi_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/emulation"

	"ily.dev/act3/hi"
	"ily.dev/act3/hi/internal/uitest"
)

func TestProgressValues(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		v    []float64
		want string
	}{
		{"zero", []float64{0}, "0"},
		{"fraction", []float64{0.25}, "0.25"},
		{"total", []float64{3, 4}, "0.75"},
		{"extra", []float64{3, 4, 100}, "0.75"},
		{"complete", []float64{1}, "1"},
		{"negative", []float64{-1}, "0"},
		{"overflow", []float64{5, 4}, "1"},
		{"zero total", []float64{1, 0}, "0"},
		{"negative total", []float64{-1, -2}, "0"},
		{"nan", []float64{math.NaN()}, "0"},
		{"nan total", []float64{1, math.NaN()}, "0"},
		{"infinity", []float64{math.Inf(1)}, "1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			html := render(t, hi.Progress(tt.v...))
			for _, want := range []string{
				`<progress `, `max="1"`, `value="` + tt.want + `"`,
			} {
				if !strings.Contains(html, want) {
					t.Errorf("progress missing %s: %s", want, html)
				}
			}
		})
	}
}

func TestProgressGeometry(t *testing.T) {
	t.Parallel()
	for _, family := range []string{"sans-serif", "serif"} {
		for _, capHeight := range []float64{12, 24} {
			for _, trim := range []hi.TextEdgeSet{0, hi.TextCap | hi.TextLastBaseline} {
				t.Run(fmt.Sprintf("%s/cap=%g/trim=%d", family, capHeight, trim), func(t *testing.T) {
					var rows []hi.View
					for _, size := range []hi.ControlSize{hi.Mini, hi.Small, hi.Regular, hi.Large} {
						rows = append(rows, hi.HStack(
							hi.Progress(),
							hi.Text("H").TextTrim(hi.TextCap|hi.TextLastBaseline).Class("cap"),
							hi.Text("Loading"),
						).Alignment(hi.FirstBaseline).Gap(32).ControlSize(size))
					}
					v := hi.VStack(rows...).Gap(40).TextTrim(trim).
						Font(hi.Family(family), hi.SizeCap(complex(capHeight, 0), 2))
					stage(t, v, func(s *uitest.Session) {
						for i, diameter := range []float64{14, 14, 20, 35} {
							box := s.Rect("hi-progress", i)
							svg := s.Rect("hi-progress > svg", i)
							cap := s.Rect(".cap", i)
							within(t, "box width", box.W, diameter, .1)
							within(t, "box height", box.H, diameter, .1)
							within(t, "artwork width", svg.W, diameter, .1)
							within(t, "artwork height", svg.H, diameter, .1)
							within(t, "horizontal center", svg.X+svg.W/2, box.X+box.W/2, .1)
							within(t, "vertical center", svg.Y+svg.H/2, box.Y+box.H/2, .1)
							within(t, "cap band center", svg.Y+svg.H/2, cap.Y+cap.H/2, .1)
							within(t, "baseline", box.Y+(diameter+capHeight)/2, cap.Bottom(), .1)
						}
					})
				})
			}
		}
	}
}

func TestProgressAnimationAndTheme(t *testing.T) {
	t.Parallel()
	v := hi.HStack(
		hi.Progress().ThemeBackground(hi.White),
		hi.Progress().ThemeBackground(hi.Black),
		hi.Icon("film"),
	).Gap(32)
	html := render(t, v)
	if strings.Contains(html, "@keyframes") {
		t.Fatal("progress animation CSS must come from the shared stylesheet")
	}
	stage(t, v, func(s *uitest.Session) {
		s.Run(emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{
			{Name: "prefers-reduced-motion", Value: "reduce"},
		}))
		var colors []string
		s.Eval(`Array.from(document.querySelectorAll('hi-progress > svg'), svg => getComputedStyle(svg).color)`, &colors)
		if fmt.Sprint(colors) != "[rgb(60, 60, 67) rgb(235, 235, 245)]" {
			t.Errorf("progress colors = %v", colors)
		}
		var count int
		s.Eval(`document.querySelector('hi-icon').getAnimations({subtree:true}).length`, &count)
		if count != 0 {
			t.Fatal("progress animation leaked onto an icon")
		}
		var result struct {
			Count         int
			MaxAlphaError int
		}
		s.Eval(`(() => {
			const paths = [...document.querySelectorAll('hi-progress:first-child > svg > path')];
			const animations = paths.flatMap(p => p.getAnimations());
			const alpha = [130,119,108,97,86,75,64,53,42,42,42,42,42,42,42,42];
			let error = 0;
			animations.forEach(a => a.pause());
			for (let frame = 0; frame < 16; frame++) {
				animations.forEach(a => a.currentTime = frame * 50 + 25);
				paths.forEach((p, i) => {
					error = Math.max(error, Math.abs(Math.round(Number(getComputedStyle(p).opacity)*255) - alpha[(frame - 2*i + 16)%16]));
				});
			}
			return {Count: animations.length, MaxAlphaError: error};
		})()`, &result)
		if result.Count != 8 || result.MaxAlphaError != 0 {
			t.Errorf("progress animation with reduced motion = %+v, want 8 animations matching all 8-bit alpha levels", result)
		}
	})
}
