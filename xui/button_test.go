package ui_test

import (
	"bytes"
	"image/png"
	"reflect"
	"testing"

	"github.com/chromedp/chromedp"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
)

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
		s.Eval(`Array.from(document.querySelectorAll("ui-text"), e => {
			const s = getComputedStyle(e.firstElementChild || e);
			return s.whiteSpace + "/" + s.webkitLineClamp;
		})`, &styles)
		if !reflect.DeepEqual(styles, []string{"nowrap/none", "normal/none", "normal/2", "normal/none"}) {
			t.Errorf("line limits = %v", styles)
		}
		one := s.Rect(".default ui-text", 0).H
		two := s.Rect(".clamped ui-text", 0).H
		if two <= one {
			t.Error("two-line label should be taller than the single-line label")
		}
		if s.Rect(".wrapped ui-text", 0).H <= two {
			t.Error("explicitly wrapping label should occupy more than two lines")
		}
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
			if button.X < row.X-0.1 || button.Right() > row.Right()+0.1 {
				t.Errorf("%s overflows the row: button %+v, row %+v", selector, button, row)
			}
			var truncated bool
			s.Eval(`(() => {
				const e = document.querySelector("`+selector+` > ui-text > span");
				return e.scrollWidth > e.clientWidth && getComputedStyle(e).textOverflow === "ellipsis";
			})()`, &truncated)
			if !truncated {
				t.Errorf("%s label should ellipsize overflowing text", selector)
			}
			// Computed text-overflow does not prove the browser painted
			// an ellipsis. Switching to clip must change the pixels.
			var ellipsis, clipped []byte
			s.Run(chromedp.Screenshot(selector+" > ui-text", &ellipsis, chromedp.ByQuery))
			s.Eval(`document.querySelector("`+selector+` > ui-text > span").style.textOverflow = "clip"`, nil)
			s.Run(chromedp.Screenshot(selector+" > ui-text", &clipped, chromedp.ByQuery))
			ellipsisImage, err := png.Decode(bytes.NewReader(ellipsis))
			if err != nil {
				t.Fatal(err)
			}
			clippedImage, err := png.Decode(bytes.NewReader(clipped))
			if err != nil {
				t.Fatal(err)
			}
			if ellipsisImage.Bounds() != clippedImage.Bounds() {
				t.Error("switching to clip changed the label geometry")
			}
			if reflect.DeepEqual(ellipsisImage, clippedImage) {
				t.Errorf("%s ellipsis paints the same as clipping", selector)
			}
		}
	})
}
