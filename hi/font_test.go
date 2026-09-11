package hi_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"ily.dev/act3/hi"
	"ily.dev/act3/hi/internal/uitest"
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
				v := hi.VStack(
					hi.Text("H").TextTrim(hi.TextCap|hi.TextLastBaseline).Class("cap"),
					hi.Text("H").Class("line"),
				).Font(hi.SizeCap(tc.size, 2), hi.Family(family))
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
	v := hi.VStack(
		hi.Text("H").Font(hi.SizeEm(20, 1.5)).Class("em"),
		hi.Text("H").TextFont(hi.SizeEm(20, 1.5)).Class("text-em"),
		hi.Text("H").Font(hi.Weight(500)).Class("inherited"),
		hi.Text("H").Font(hi.SizeCap(12, 2), hi.SizeEm(20, 1.5)).Class("later-em"),
		hi.Text("H").Font(hi.SizeEm(20, 1.5), hi.SizeCap(12, 2)).Class("later-cap"),
		hi.Text("H").Font(hi.SizeCapAbs(12, 36), hi.SizeEm(20, 1.5)).Class("later-relative"),
		hi.Text("H").Font(hi.SizeEm(20, 1.5), hi.SizeCapAbs(12, 36)).Class("later-absolute"),
		hi.Text("prefix ").Concat(hi.Text("H").TextFont(hi.SizeEm(20, 1.5))).Class("rich"),
	).Font(hi.SizeCap(24, 2))
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
	for name, option := range map[string]func(complex128, complex128) hi.FontOption{
		"em": hi.SizeEmAbs, "cap": hi.SizeCapAbs,
	} {
		for _, tc := range []struct {
			name   string
			height complex128
		}{
			{"pixels", 36}, {"scaled", 36i}, {"mixed", 20 + 8i}, {"zero", 0},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				v := hi.VStack(
					hi.Text("H").Font(option(24i, tc.height)).Class("large"),
					hi.Text("H").TextFont(option(12i, tc.height)).Class("small"),
					hi.Text("H").Class("inherited"),
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
	opts := []hi.FontOption{
		hi.OldstyleNums, hi.ProportionalNums, hi.TabularNums, hi.SlashedZero,
		hi.NoCommonLigatures, hi.CommonLigatures, hi.DiscretionaryLigatures,
		hi.OpenTypeFeature("ss01", 1), hi.OpenTypeFeature("cv02", 3),
		hi.OpenTypeFeature("ss01", 0),
	}
	v := hi.VStack(
		hi.Text("0123").Font(opts...).Class("view"),
		hi.Text("0123").TextFont(opts...).Class("text"),
		hi.Text("prefix ").Concat(hi.Text("0123").TextFont(opts...)).Class("rich"),
		hi.Text("0123").Class("inherited"),
	).Font(hi.SmallCaps, hi.LiningNums, hi.TabularNums, hi.OpenTypeFeature("ss02", 1))
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
	inner := []hi.FontOption{hi.TabularNums, hi.CommonLigatures, hi.OpenTypeFeature("ss01", 0)}
	outer := []hi.FontOption{
		hi.SlashedZero, hi.ProportionalNums, hi.DiscretionaryLigatures, hi.NoCommonLigatures,
		hi.OpenTypeFeature("ss01", 1), hi.OpenTypeFeature("ss02", 2),
	}
	for _, tc := range []struct {
		name        string
		view        hi.View
		selector    string
		separateBox bool
	}{
		{"one call", hi.Text("x").Font(slices.Concat(outer, inner)...), "hi-text", false},
		{"view calls", hi.Text("x").Font(inner...).Font(outer...), "hi-text", false},
		{"text calls", hi.Text("x").TextFont(inner...).TextFont(outer...), "hi-text", false},
		{"mixed calls", hi.Text("x").TextFont(inner...).Font(outer...), "hi-text", false},
		{"container", hi.VStack(hi.Text("x").Font(inner...)).Font(outer...), "hi-text", true},
		{"frame", hi.Text("x").Font(inner...).Frame(hi.Width(100)).Font(outer...), "hi-text", true},
		{"rich text", hi.Text("prefix ").Concat(hi.Text("x").TextFont(inner...)).TextFont(outer...), "hi-text span", true},
		{"inline link", hi.Text("prefix ").Concat(hi.Link("/", hi.Text("x").TextFont(inner...))).TextFont(outer...), "a span", true},
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
	stage(t, hi.VStack(
		hi.Text("ffi").Font(hi.CommonLigatures, hi.DiscretionaryLigatures, hi.NoLigatures).Class("off"),
		hi.Text("ffi").Font(hi.NoLigatures, hi.CommonLigatures).Class("common"),
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
			hi.OpenTypeFeature(tc.tag, tc.value)
		})
	}
	stage(t, hi.Text("x").Font(hi.OpenTypeFeature("a\"\\b", 3)), func(s *uitest.Session) {
		var got string
		s.Eval(`getComputedStyle(document.querySelector("hi-text")).fontFeatureSettings`, &got)
		if got != `"a\"\\b" 3` {
			t.Errorf("escaped feature = %q, want a valid quoted CSS tag", got)
		}
	})
}
