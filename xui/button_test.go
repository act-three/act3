package ui_test

import (
	"fmt"
	"reflect"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
)

func TestButtonSizeTypography(t *testing.T) {
	for _, tt := range []struct {
		size       ui.ControlSize
		font, line string
		height     float64
	}{
		{ui.Mini, "12px", "16px", 23},
		{ui.Small, "12px", "16px", 27},
		{ui.Regular, "13px", "18px", 31},
		{ui.Large, "13px", "18px", 43},
	} {
		t.Run(fmt.Sprint(tt.size), func(t *testing.T) {
			v := ui.VStack(
				ui.Button(Msg{}, ui.Text("Save")),
				ui.Button("/movies", ui.Text("Movies")),
				ui.Button(Msg{}, ui.Text("Title").Font(ui.Title)),
				ui.Button(Msg{}, ui.Text("Code").Monospace()),
				ui.Button(Msg{}, ui.VStack(ui.Text("First"), ui.Text("Second")).Gap(0)),
			).ControlSize(tt.size).Font(ui.LargeTitle)
			stage(t, v, func(s *uitest.Session) {
				var styles [][4]string
				s.Eval(`Array.from(document.querySelectorAll("button, a"), e => {
					const s = getComputedStyle(e.querySelector("ui-text"));
					return [s.fontSize, s.fontWeight, s.lineHeight, s.fontFamily];
				})`, &styles)
				if len(styles) != 5 {
					t.Fatalf("got %d controls, want 5", len(styles))
				}
				for _, i := range []int{0, 1, 3, 4} {
					if got := styles[i][:3]; !reflect.DeepEqual(got, []string{tt.font, "500", tt.line}) {
						t.Errorf("label %d typography = %v", i, got)
					}
				}
				if got := styles[2][:3]; !reflect.DeepEqual(got, []string{"24px", "700", "28.8px"}) {
					t.Errorf("explicit title typography = %v", got)
				}
				var family string
				s.Eval(`getComputedStyle(document.querySelector("ui-root")).fontFamily`, &family)
				if styles[0][3] != family || styles[3][3] == family {
					t.Errorf("font family inheritance: root %q, plain %q, monospace %q", family, styles[0][3], styles[3][3])
				}
				within(t, "message height", s.Rect("button", 0).H, tt.height, 0.1)
				within(t, "URL height", s.Rect("a", 0).H, tt.height, 0.1)
				if s.Rect("button", 1).H <= tt.height || s.Rect("button", 3).H <= tt.height {
					t.Error("large-font and multiline labels should grow their buttons")
				}
				var tree []string
				s.Eval(`Array.from(document.querySelector("button").querySelectorAll("*"), e => e.localName)`, &tree)
				if !reflect.DeepEqual(tree, []string{"ui-text"}) {
					t.Errorf("simple label tree = %v, want direct ui-text", tree)
				}
			})
		})
	}
}

func TestButtonLineLimit(t *testing.T) {
	const long = "Save these changes to the current collection"
	label := func() ui.View { return ui.Text(long).Frame(ui.Width(100)) }
	v := ui.VStack(
		ui.Button(Msg{}, label()).LineLimit(0).Class("default"),
		ui.Button("/movies", label().LineLimit(0)).LineLimit(1).Class("wrapped"),
		ui.Button(Msg{}, label().LineLimit(2)).Class("clamped"),
		ui.Text(long).Frame(ui.Width(100)).Class("sibling"),
	)
	stage(t, v, func(s *uitest.Session) {
		var styles []string
		s.Eval(`Array.from(document.querySelectorAll("ui-text"), e => getComputedStyle(e).webkitLineClamp)`, &styles)
		if !reflect.DeepEqual(styles, []string{"1", "none", "2", "none"}) {
			t.Errorf("line limits = %v", styles)
		}
		within(t, "default label stays one line", s.Rect(".default ui-text", 0).H, 18, 0.1)
		if s.Rect(".wrapped ui-text", 0).H <= 36 {
			t.Error("explicitly wrapping label should occupy more than two lines")
		}
		within(t, "wrapping composes with line limit", s.Rect(".clamped ui-text", 0).H, 36, 0.1)
	})
}

func TestButtonLineLimitUnderPressure(t *testing.T) {
	const long = "Save all changes to this collection"
	v := ui.HStack(
		ui.Button(Msg{}, ui.Text(long)),
		ui.Button("/movies", ui.Text(long)),
	).Gap(8).Frame(ui.Width(180))
	stage(t, v, func(s *uitest.Session) {
		row := s.Rect("ui-hstack", 0)
		for _, selector := range []string{"button", "a"} {
			button := s.Rect(selector, 0)
			label := s.Rect(selector+" > ui-text", 0)
			within(t, selector+" label height", label.H, 18, 0.1)
			if button.X < row.X-0.1 || button.Right() > row.Right()+0.1 {
				t.Errorf("%s overflows the row: button %+v, row %+v", selector, button, row)
			}
			var clipped bool
			s.Eval(`(() => {
				const e = document.querySelector("`+selector+` > ui-text");
				return e.scrollHeight > e.clientHeight && getComputedStyle(e).webkitLineClamp === "1";
			})()`, &clipped)
			if !clipped {
				t.Errorf("%s label should clamp overflowing text", selector)
			}
		}
	})
}
