package flurry

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestNewID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		for _, tt := range []struct {
			at   time.Time
			want []string
		}{
			{
				at: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				want: []string{
					"0enxh5g00",
					"0enxh5g01",
					"0enxh5g02",
				},
			},
			{
				at: time.Date(2026, 11, 12, 2, 3, 4, 567000000, time.UTC),
				want: []string{
					"0vbjn1qvg",
					"0vbjn1qvh",
					"0vbjn1qvj",
				},
			},
			{
				at: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
				want: []string{
					"0xbv2b000",
					"0xbv2b001",
					"0xbv2b002",
					"0xbv2b003",
					"0xbv2b004",
					"0xbv2b005",
					"0xbv2b006",
					"0xbv2b007",
					"0xbv2b008",
					"0xbv2b009",
					"0xbv2b00a",
					"0xbv2b00b",
					"0xbv2b00c",
					"0xbv2b00d",
					"0xbv2b00e",
					"0xbv2b00f",
					"0xbv2b00g",
					"0xbv2b00h",
				},
			},
		} {
			time.Sleep(time.Until(tt.at))
			t.Logf("%b", time.Now().UnixMilli())
			for i, want := range tt.want {
				got := NewID()
				if got != want {
					t.Errorf("at %v, NewID() #%d = %v, want %v", tt.at, i, got, want)
				}
			}
		}
	})
}
