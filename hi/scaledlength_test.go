package hi_test

import (
	"fmt"
	"strings"
	"testing"

	"ily.dev/act3/hi"
	"ily.dev/act3/hi/internal/uitest"
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
			html := render(t, hi.Text("x").Padding(hi.Edges(tt.length)))
			if !strings.Contains(html, "padding-block-start:"+tt.want+";") {
				t.Errorf("padding missing %q:\n%s", tt.want, html)
			}
		})
	}
}

// Change the root font after rendering to catch premature resolution,
// and override the local font to distinguish rem from em.
func TestLengthRootFont(t *testing.T) {
	v := hi.VStack(
		hi.Text("fixed").Padding(hi.Edges(8)).Class("fixed"),
		hi.Text("scaled").Padding(hi.Edges(8i)).Class("scaled"),
		hi.Text("mixed").Padding(hi.Edges(3+8i)).Class("mixed"),
		hi.Text("subtracted").Padding(hi.Edges(20-8i)).Class("subtracted"),
		hi.Text("summed").Padding(hi.Edges(3), hi.Edges(8i)).Class("summed"),
		hi.Text("edges").Padding(
			hi.EdgeTop(8i), hi.EdgeBottom(3+8i),
			hi.EdgeLeading(20-8i), hi.EdgeTrailing(8),
		).Class("edges"),
		hi.Text("pairs").Padding(hi.EdgesPillarbox(8i), hi.EdgesLetterbox(3+8i)).Class("pairs"),
		hi.HStack(hi.Text("a"), hi.Text("b")).Gap(3+8i).Class("stack"),
		hi.Grid(hi.ColumnMinWidth(16+64i), hi.Text("cell")).Gap(8i).
			Class("grid").Frame(hi.Width(320)),
		hi.Text("frame").Frame(hi.Width(3+80i), hi.Height(2+16i)).Class("frame"),
		hi.Text("stroke").BorderStroke(12-8i, hi.Red).Class("stroke"),
		hi.Text("sticky").Sticky(hi.EdgeTop(-10+8i)).Class("sticky"),
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
	v := hi.VStack(
		hi.Text("ideal last").FrameBounds(
			hi.MinWidth(80), hi.IdealWidth(64i),
			hi.MinHeight(80), hi.IdealHeight(64i),
		).FixedSize().Class("ideal-last"),
		hi.Text("min last").FrameBounds(
			hi.IdealWidth(64i), hi.MinWidth(80),
			hi.IdealHeight(64i), hi.MinHeight(80),
		).FixedSize().Class("min-last"),
		hi.Text("repeated min").FrameBounds(
			hi.IdealWidth(64i), hi.MinWidth(80), hi.MinWidth(16+48i),
		).FixedSize().Class("repeated-min"),
		hi.Text("repeated ideal").FrameBounds(
			hi.MinWidth(64i), hi.IdealWidth(80), hi.IdealWidth(16+48i),
		).FixedSize().Class("repeated-ideal"),
		hi.Text("omitted minimum").FrameBounds(
			hi.IdealWidth(64i),
		).FixedSize().Class("omitted-min"),
		hi.Text("zero min").FrameBounds(
			hi.MinWidth(0), hi.IdealWidth(-4+8i),
		).FixedSize().Class("zero-min"),
		hi.Text("zero ideal").FrameBounds(
			hi.IdealWidth(0), hi.MinWidth(-4+8i),
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
			html := render(t, hi.Grid(hi.ColumnMinWidth(tt.width), hi.Text("x")))
			want := "repeat(auto-fill, minmax(" + tt.want + ", 1fr))"
			if !strings.Contains(html, want) {
				t.Errorf("grid missing %q:\n%s", want, html)
			}
		})
	}
}
