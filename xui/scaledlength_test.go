package ui_test

import (
	"fmt"
	"strings"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
)

func TestLengthCSS(t *testing.T) {
	for _, tt := range []struct {
		length complex128
		want   string
	}{
		{0, "0px"},
		{8, "8px"},
		{8i, "0.5rem"},
		{3 + 8i, "calc(3px + 0.5rem)"},
		{20 - 8i, "calc(20px - 0.5rem)"},
		{-3 + 8i, "calc(-3px + 0.5rem)"},
		{(3 + 8i) * 2 / 4, "calc(1.5px + 0.25rem)"},
	} {
		t.Run(fmt.Sprint(tt.length), func(t *testing.T) {
			html := render(t, ui.Text("x").Padding(ui.Edges(tt.length)))
			if !strings.Contains(html, "padding-block-start:"+tt.want+";") {
				t.Errorf("padding missing %q:\n%s", tt.want, html)
			}
		})
	}
}

// Change the root font after rendering to catch premature resolution,
// and override the local font to distinguish rem from em.
func TestLengthRootFont(t *testing.T) {
	v := ui.VStack(
		ui.Text("fixed").Padding(ui.Edges(8)).Class("fixed"),
		ui.Text("scaled").Padding(ui.Edges(8i)).Class("scaled"),
		ui.Text("mixed").Padding(ui.Edges(3+8i)).Class("mixed"),
		ui.Text("subtracted").Padding(ui.Edges(20-8i)).Class("subtracted"),
		ui.Text("summed").Padding(ui.Edges(3), ui.Edges(8i)).Class("summed"),
		ui.Text("edges").Padding(
			ui.EdgeTop(8i), ui.EdgeBottom(3+8i),
			ui.EdgeLeading(20-8i), ui.EdgeTrailing(8),
		).Class("edges"),
		ui.Text("pairs").Padding(ui.EdgesPillarbox(8i), ui.EdgesLetterbox(3+8i)).Class("pairs"),
		ui.HStack(ui.Text("a"), ui.Text("b")).Gap(3+8i).Class("stack"),
		ui.Grid(ui.ColumnMinWidth(16+64i), ui.Text("cell")).Gap(8i).
			Class("grid").Frame(ui.Width(320)),
		ui.Text("frame").Frame(ui.Width(3+80i), ui.Height(2+16i)).Class("frame"),
		ui.Text("stroke").BorderStroke(12-8i, ui.Red).Class("stroke"),
		ui.Text("sticky").Sticky(ui.EdgeTop(-10+8i)).Class("sticky"),
	)
	stage(t, v, func(s *uitest.Session) {
		s.Eval(`document.querySelector(".mixed").style.fontSize = "64px"`, nil)
		for _, root := range []float64{16, 24, 8, 32} {
			s.Eval(fmt.Sprintf(`document.documentElement.style.fontSize = "%gpx"`, root), nil)
			scale := root / 16
			for _, tt := range []struct {
				class, property string
				want            float64
			}{
				{"fixed", "paddingTop", 8},
				{"scaled", "paddingTop", 8 * scale},
				{"mixed", "paddingTop", 3 + 8*scale},
				{"subtracted", "paddingTop", 20 - 8*scale},
				{"summed", "paddingTop", 3 + 8*scale},
				{"edges", "paddingTop", 8 * scale},
				{"edges", "paddingBottom", 3 + 8*scale},
				{"edges", "paddingLeft", 20 - 8*scale},
				{"edges", "paddingRight", 8},
				{"pairs", "paddingTop", 3 + 8*scale},
				{"pairs", "paddingLeft", 8 * scale},
				{"stack", "columnGap", 3 + 8*scale},
				{"grid", "rowGap", 8 * scale},
				{"frame", "width", 3 + 80*scale},
				{"frame", "height", 2 + 16*scale},
				{"sticky", "top", -10 + 8*scale},
			} {
				var got *float64
				s.Eval(fmt.Sprintf(`parseFloat(getComputedStyle(document.querySelector(".%s")).%s)`,
					tt.class, tt.property), &got)
				if got == nil {
					t.Fatalf("root %g: %s.%s did not resolve to a number", root, tt.class, tt.property)
				}
				within(t, fmt.Sprintf("root %g: %s.%s", root, tt.class, tt.property), *got, tt.want, 0.01)
			}
			var columns string
			s.Eval(`getComputedStyle(document.querySelector(".grid")).gridTemplateColumns`, &columns)
			wantColumns := int((320 + 8*scale) / (16 + 72*scale))
			if got := len(strings.Fields(columns)); got != wantColumns {
				t.Errorf("root %g: grid columns = %q, want %d columns", root, columns, wantColumns)
			}
			var shadow string
			s.Eval(`getComputedStyle(document.querySelector(".stroke"), "::after").boxShadow`, &shadow)
			wantShadow := fmt.Sprintf("0px 0px 0px %gpx", 12-8*scale)
			if !strings.Contains(shadow, wantShadow) {
				t.Errorf("root %g: stroke = %q, want %q", root, shadow, wantShadow)
			}
		}
	})
}

func TestLengthFrameBounds(t *testing.T) {
	v := ui.VStack(
		ui.Text("ideal last").FrameBounds(
			ui.MinWidth(80), ui.IdealWidth(64i),
			ui.MinHeight(80), ui.IdealHeight(64i),
		).FixedSize().Class("ideal-last"),
		ui.Text("min last").FrameBounds(
			ui.IdealWidth(64i), ui.MinWidth(80),
			ui.IdealHeight(64i), ui.MinHeight(80),
		).FixedSize().Class("min-last"),
		ui.Text("repeated min").FrameBounds(
			ui.IdealWidth(64i), ui.MinWidth(80), ui.MinWidth(16+48i),
		).FixedSize().Class("repeated-min"),
		ui.Text("repeated ideal").FrameBounds(
			ui.MinWidth(64i), ui.IdealWidth(80), ui.IdealWidth(16+48i),
		).FixedSize().Class("repeated-ideal"),
		ui.Text("omitted minimum").FrameBounds(
			ui.IdealWidth(64i),
		).FixedSize().Class("omitted-min"),
		ui.Text("zero min").FrameBounds(
			ui.MinWidth(0), ui.IdealWidth(-4+8i),
		).FixedSize().Class("zero-min"),
		ui.Text("zero ideal").FrameBounds(
			ui.IdealWidth(0), ui.MinWidth(-4+8i),
		).FixedSize().Class("zero-ideal"),
	)
	stage(t, v, func(s *uitest.Session) {
		for _, root := range []float64{16, 24, 8} {
			s.Eval(fmt.Sprintf(`document.documentElement.style.fontSize = "%gpx"`, root), nil)
			scale := root / 16
			for _, tt := range []struct {
				class, property string
				want            float64
			}{
				{"ideal-last", "width", 64 * scale},
				{"ideal-last", "minWidth", min(80, 64*scale)},
				{"ideal-last", "height", 64 * scale},
				{"ideal-last", "minHeight", min(80, 64*scale)},
				{"min-last", "width", max(80, 64*scale)},
				{"min-last", "minWidth", 80},
				{"min-last", "height", max(80, 64*scale)},
				{"min-last", "minHeight", 80},
				{"repeated-min", "width", max(64*scale, 80, 16+48*scale)},
				{"repeated-min", "minWidth", 16 + 48*scale},
				{"repeated-ideal", "width", 16 + 48*scale},
				{"repeated-ideal", "minWidth", min(64*scale, 80, 16+48*scale)},
				{"omitted-min", "width", 64 * scale},
				{"zero-min", "width", max(0, -4+8*scale)},
				{"zero-min", "minWidth", 0},
				{"zero-ideal", "width", max(0, -4+8*scale)},
			} {
				var got *float64
				s.Eval(fmt.Sprintf(`parseFloat(getComputedStyle(document.querySelector(".%s")).%s)`,
					tt.class, tt.property), &got)
				if got == nil {
					t.Fatalf("root %g: %s.%s did not resolve to a number", root, tt.class, tt.property)
				}
				within(t, fmt.Sprintf("root %g: %s.%s", root, tt.class, tt.property), *got, tt.want, 0.01)
			}
		}
	})
}

func TestLengthColumnMinWidth(t *testing.T) {
	for _, tt := range []struct {
		width complex128
		want  string
	}{
		{16, "16px"},
		{64i, "4rem"},
		{16 + 64i, "calc(16px + 4rem)"},
		{0, "0px"},
		{-16, "-16px"},
		{-64i, "-4rem"},
		{-16 - 64i, "calc(-16px - 4rem)"},
		{16 - 64i, "calc(16px - 4rem)"},
		{-16 + 64i, "calc(-16px + 4rem)"},
	} {
		t.Run(fmt.Sprint(tt.width), func(t *testing.T) {
			html := render(t, ui.Grid(ui.ColumnMinWidth(tt.width), ui.Text("x")))
			want := "repeat(auto-fill, minmax(" + tt.want + ", 1fr))"
			if !strings.Contains(html, want) {
				t.Errorf("grid missing %q:\n%s", want, html)
			}
		})
	}
}
