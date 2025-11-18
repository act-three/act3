package model

import (
	"testing"
)

func TestTaskFailDelay(t *testing.T) {
	for _, tt := range []struct {
		n int64
		d string
	}{
		{0, "100ms"},
		{1, "200ms"},
		{2, "400ms"},
		{5, "3.2s"},
		{10, "1m42.4s"},
		{15, "54m36.8s"},
		{16, "1h49m13.6s"},
		{17, "2h0m0s"},
		{18, "2h0m0s"},
	} {
		d := taskFailDelay(tt.n)
		got := d.String()
		if got != tt.d {
			t.Errorf("taskFailDelay(%d) = %v, want %v", tt.n, got, tt.d)
		}
	}
}
