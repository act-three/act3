package view

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{500 * time.Millisecond, "0s"},
		{45 * time.Second, "45s"},
		{59*time.Second + 999*time.Millisecond, "59s"},
		{1 * time.Minute, "1m 0s"},
		{3*time.Minute + 24*time.Second, "3m 24s"},
		{59*time.Minute + 59*time.Second, "59m 59s"},
		{1 * time.Hour, "1h 0m"},
		{1*time.Hour + 12*time.Minute, "1h 12m"},
		{2*time.Hour + 30*time.Minute + 45*time.Second, "2h 30m"},
	}
	for _, tt := range tests {
		if got := formatDuration(tt.d); got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
