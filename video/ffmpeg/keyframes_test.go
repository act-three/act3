package ffmpeg

import (
	"slices"
	"testing"
	"time"
)

func TestSegmentBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		keyframes []int64
		minFrames int64
		want      []int64
	}{
		{
			name:      "empty",
			keyframes: nil,
			minFrames: 96,
			want:      nil,
		},
		{
			name:      "single keyframe",
			keyframes: []int64{0},
			minFrames: 96,
			want:      nil,
		},
		{
			name:      "all spaced ≥ minFrames",
			keyframes: []int64{0, 144, 288, 432, 576},
			minFrames: 96,
			want:      []int64{144, 288, 432, 576},
		},
		{
			name:      "skip closely-spaced keyframes",
			keyframes: []int64{0, 50, 100, 200, 250, 350},
			minFrames: 100,
			want:      []int64{100, 200, 350},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SegmentBoundaries(tt.keyframes, tt.minFrames)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinFramesPerSegment(t *testing.T) {
	tests := []struct {
		name string
		fps  FrameRate
		dur  time.Duration
		want int64
	}{
		{"24000/1001 4s", FrameRate{24000, 1001}, 4 * time.Second, 96},
		{"30/1 4s", FrameRate{30, 1}, 4 * time.Second, 120},
		{"60/1 4s", FrameRate{60, 1}, 4 * time.Second, 240},
		{"degenerate fps", FrameRate{0, 1}, 4 * time.Second, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MinFramesPerSegment(tt.fps, tt.dur)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestUniformSegmentBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		fps       FrameRate
		duration  time.Duration
		minFrames int64
		want      []int64
	}{
		{
			name:      "16s @ 24000/1001 with minFrames=96",
			fps:       FrameRate{24000, 1001},
			duration:  16 * time.Second,
			minFrames: 96,
			want:      []int64{96, 192, 288},
		},
		{
			name:      "20s @ 30 with minFrames=120",
			fps:       FrameRate{30, 1},
			duration:  20 * time.Second,
			minFrames: 120,
			want:      []int64{120, 240, 360, 480},
		},
		{
			name:      "shorter than one segment",
			fps:       FrameRate{30, 1},
			duration:  3 * time.Second,
			minFrames: 120,
			want:      nil,
		},
		{
			name:      "exactly one segment",
			fps:       FrameRate{30, 1},
			duration:  4 * time.Second,
			minFrames: 120,
			want:      nil,
		},
		{
			name:      "degenerate fps",
			fps:       FrameRate{0, 1},
			duration:  16 * time.Second,
			minFrames: 96,
			want:      nil,
		},
		{
			name:      "zero minFrames",
			fps:       FrameRate{30, 1},
			duration:  16 * time.Second,
			minFrames: 0,
			want:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniformSegmentBoundaries(tt.fps, tt.duration, tt.minFrames)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxKeyframeGap(t *testing.T) {
	tests := []struct {
		name      string
		keyframes []int64
		want      int64
	}{
		{"empty", nil, 0},
		{"single", []int64{0}, 0},
		{"uniform", []int64{0, 100, 200, 300}, 100},
		{"irregular, max in middle", []int64{0, 50, 300, 400}, 250},
		{"irregular, max at end", []int64{0, 100, 200, 600}, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxKeyframeGap(tt.keyframes)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
