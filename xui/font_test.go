package ui_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	ui "ily.dev/act3/xui"
	"ily.dev/act3/xui/internal/uitest"
)

func TestFontCapSize(t *testing.T) {
	for _, family := range []string{"serif", "sans-serif", "monospace"} {
		for _, tc := range []struct {
			name string
			size complex128
		}{
			{"pixels", 20}, {"scaled", 20i}, {"mixed", 10 + 8i},
		} {
			t.Run(family+"/"+tc.name, func(t *testing.T) {
				v := ui.VStack(
					ui.Text("H").TextTrim(ui.TextCap|ui.TextLastBaseline).Class("cap"),
					ui.Text("H").Class("line"),
				).Font(ui.SizeCap(tc.size, 2), ui.Family(family))
				stage(t, v, func(s *uitest.Session) {
					for _, root := range []float64{16, 24} {
						s.Eval(fmt.Sprintf(`document.documentElement.style.fontSize = "%gpx"`, root), nil)
						want := real(tc.size) + imag(tc.size)*root/16
						within(t, "cap height", s.Rect(".cap", 0).H, want, 0.1)
						within(t, "line height", s.Rect(".line", 0).H, want*2, 0.1)
					}
				})
			})
		}
	}
}

func TestFontSizeBasisOverrides(t *testing.T) {
	v := ui.VStack(
		ui.Text("H").Font(ui.SizeEm(20, 1.5)).Class("em"),
		ui.Text("H").TextFont(ui.SizeEm(20, 1.5)).Class("text-em"),
		ui.Text("H").Font(ui.Weight(500)).Class("inherited"),
		ui.Text("H").Font(ui.SizeCap(12, 2), ui.SizeEm(20, 1.5)).Class("later-em"),
		ui.Text("H").Font(ui.SizeEm(20, 1.5), ui.SizeCap(12, 2)).Class("later-cap"),
		ui.Text("H").Font(ui.SizeCapAbs(12, 36), ui.SizeEm(20, 1.5)).Class("later-relative"),
		ui.Text("H").Font(ui.SizeEm(20, 1.5), ui.SizeCapAbs(12, 36)).Class("later-absolute"),
		ui.Text("prefix ").Concat(ui.Text("H").TextFont(ui.SizeEm(20, 1.5))).Class("rich"),
	).Font(ui.SizeCap(24, 2))
	stage(t, v, func(s *uitest.Session) {
		for _, tc := range []struct {
			selector, size, adjust, height string
		}{
			{".em", "20px", "none", "30px"},
			{".text-em", "20px", "none", "30px"},
			{".inherited", "24px", "cap-height 1", "48px"},
			{".later-em", "20px", "none", "30px"},
			{".later-cap", "12px", "cap-height 1", "24px"},
			{".later-relative", "20px", "none", "30px"},
			{".later-absolute", "12px", "cap-height 1", "36px"},
			{".rich span", "20px", "none", "30px"},
		} {
			var got []string
			s.Eval(`(() => {
				const s = getComputedStyle(document.querySelector("`+tc.selector+`"));
				return [s.fontSize, s.fontSizeAdjust, s.lineHeight];
			})()`, &got)
			if want := []string{tc.size, tc.adjust, tc.height}; !reflect.DeepEqual(got, want) {
				t.Errorf("%s size settings = %q, want %q", tc.selector, got, want)
			}
		}
	})
}

func TestFontAbsoluteLineHeight(t *testing.T) {
	for name, option := range map[string]func(complex128, complex128) ui.FontOption{
		"em": ui.SizeEmAbs, "cap": ui.SizeCapAbs,
	} {
		for _, tc := range []struct {
			name   string
			height complex128
		}{
			{"pixels", 36}, {"scaled", 36i}, {"mixed", 20 + 8i}, {"zero", 0},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				v := ui.VStack(
					ui.Text("H").Font(option(24i, tc.height)).Class("large"),
					ui.Text("H").TextFont(option(12i, tc.height)).Class("small"),
					ui.Text("H").Class("inherited"),
				).Font(option(16i, tc.height))
				stage(t, v, func(s *uitest.Session) {
					for _, root := range []float64{16, 24} {
						s.Eval(fmt.Sprintf(`document.documentElement.style.fontSize = "%gpx"`, root), nil)
						want := real(tc.height) + imag(tc.height)*root/16
						for _, selector := range []string{".large", ".small", ".inherited"} {
							within(t, selector+" line height", s.Rect(selector, 0).H, want, 0.1)
						}
					}
				})
			})
		}
	}
}

func TestFontFeaturesCompose(t *testing.T) {
	opts := []ui.FontOption{
		ui.OldstyleNums, ui.ProportionalNums, ui.TabularNums, ui.SlashedZero,
		ui.NoCommonLigatures, ui.CommonLigatures, ui.DiscretionaryLigatures,
		ui.OpenTypeFeature("ss01", 1), ui.OpenTypeFeature("cv02", 3),
		ui.OpenTypeFeature("ss01", 0),
	}
	v := ui.VStack(
		ui.Text("0123").Font(opts...).Class("view"),
		ui.Text("0123").TextFont(opts...).Class("text"),
		ui.Text("prefix ").Concat(ui.Text("0123").TextFont(opts...)).Class("rich"),
		ui.Text("0123").Class("inherited"),
	).Font(ui.SmallCaps, ui.LiningNums, ui.TabularNums, ui.OpenTypeFeature("ss02", 1))
	stage(t, v, func(s *uitest.Session) {
		want := []string{
			"small-caps", "oldstyle-nums tabular-nums slashed-zero",
			"common-ligatures discretionary-ligatures", `"cv02" 3, "ss01" 0`,
		}
		for _, selector := range []string{".view", ".text", ".rich span", ".inherited"} {
			var got []string
			s.Eval(`(() => {
				const s = getComputedStyle(document.querySelector("`+selector+`"));
				return [s.fontVariantCaps, s.fontVariantNumeric,
					s.fontVariantLigatures, s.fontFeatureSettings];
			})()`, &got)
			if selector == ".inherited" {
				want = []string{"small-caps", "lining-nums tabular-nums", "normal", `"ss02"`}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s font settings = %q, want %q", selector, got, want)
			}
		}
	})
}

func TestFontFeaturesAcrossModifiers(t *testing.T) {
	inner := []ui.FontOption{ui.TabularNums, ui.CommonLigatures, ui.OpenTypeFeature("ss01", 0)}
	outer := []ui.FontOption{
		ui.SlashedZero, ui.ProportionalNums, ui.DiscretionaryLigatures, ui.NoCommonLigatures,
		ui.OpenTypeFeature("ss01", 1), ui.OpenTypeFeature("ss02", 2),
	}
	for _, tc := range []struct {
		name        string
		view        ui.View
		selector    string
		separateBox bool
	}{
		{"one call", ui.Text("x").Font(slices.Concat(outer, inner)...), "ui-text", false},
		{"view calls", ui.Text("x").Font(inner...).Font(outer...), "ui-text", false},
		{"text calls", ui.Text("x").TextFont(inner...).TextFont(outer...), "ui-text", false},
		{"mixed calls", ui.Text("x").TextFont(inner...).Font(outer...), "ui-text", false},
		{"container", ui.VStack(ui.Text("x").Font(inner...)).Font(outer...), "ui-text", true},
		{"frame", ui.Text("x").Font(inner...).Frame(ui.Width(100)).Font(outer...), "ui-text", true},
		{"rich text", ui.Text("prefix ").Concat(ui.Text("x").TextFont(inner...)).TextFont(outer...), "ui-text span", true},
		{"inline link", ui.Text("prefix ").Concat(ui.Link("/", ui.Text("x").TextFont(inner...))).TextFont(outer...), "a span", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stage(t, tc.view, func(s *uitest.Session) {
				var got []string
				s.Eval(`(() => {
					const s = getComputedStyle(document.querySelector("`+tc.selector+`"));
					return [s.fontVariantNumeric, s.fontVariantLigatures, s.fontFeatureSettings];
				})()`, &got)
				want := []string{"tabular-nums slashed-zero", "common-ligatures discretionary-ligatures", `"ss01" 0, "ss02" 2`}
				if tc.separateBox {
					want = []string{"tabular-nums", "common-ligatures", `"ss01" 0`}
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("font settings = %q, want %q", got, want)
				}
			})
		})
	}
}

func TestFontLigaturesReset(t *testing.T) {
	stage(t, ui.VStack(
		ui.Text("ffi").Font(ui.CommonLigatures, ui.DiscretionaryLigatures, ui.NoLigatures).Class("off"),
		ui.Text("ffi").Font(ui.NoLigatures, ui.CommonLigatures).Class("common"),
	), func(s *uitest.Session) {
		for _, tc := range []struct {
			selector, want string
		}{
			{".off", "none"},
			{".common", "common-ligatures no-discretionary-ligatures no-historical-ligatures no-contextual"},
		} {
			var got string
			s.Eval(`getComputedStyle(document.querySelector("`+tc.selector+`")).fontVariantLigatures`, &got)
			if got != tc.want {
				t.Errorf("%s ligatures = %q, want %q", tc.selector, got, tc.want)
			}
		}
	})
}

func TestOpenTypeFeatureValidation(t *testing.T) {
	for _, tc := range []struct {
		tag   string
		value int
	}{
		{"abc", 1}, {"abcde", 1}, {"a\nbc", 1}, {"abc\x7f", 1}, {"éab", 1}, {"ss01", -1},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("invalid OpenType feature did not panic")
				}
			}()
			ui.OpenTypeFeature(tc.tag, tc.value)
		})
	}
	stage(t, ui.Text("x").Font(ui.OpenTypeFeature("a\"\\b", 3)), func(s *uitest.Session) {
		var got string
		s.Eval(`getComputedStyle(document.querySelector("ui-text")).fontFeatureSettings`, &got)
		if got != `"a\"\\b" 3` {
			t.Errorf("escaped feature = %q, want a valid quoted CSS tag", got)
		}
	})
}
