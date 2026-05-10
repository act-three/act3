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
		fps       FrameRate
		target    time.Duration
		want      []int64
	}{
		{
			name:      "empty",
			keyframes: nil,
			fps:       FrameRate{Num: 24, Den: 1},
			target:    4 * time.Second,
			want:      nil,
		},
		{
			name:      "single keyframe",
			keyframes: []int64{0},
			fps:       FrameRate{Num: 24, Den: 1},
			target:    4 * time.Second,
			want:      nil,
		},
		{
			name:      "all spaced ≥ minFrames",
			keyframes: []int64{0, 144, 288, 432, 576},
			fps:       FrameRate{Num: 24, Den: 1},
			target:    4 * time.Second,
			want:      []int64{144, 288, 432, 576},
		},
		{
			name:      "skip closely-spaced keyframes",
			keyframes: []int64{0, 50, 100, 200, 250, 350},
			fps:       FrameRate{Num: 25, Den: 1}, // 4s × 25 = 100 frames/seg
			target:    4 * time.Second,
			want:      []int64{100, 200, 350},
		},
		{
			// When a previous cut overshoots its target, the next
			// target is still on the absolute schedule — so a
			// segment can come out shorter than the nominal
			// frame-count target as long as its end keyframe sits
			// at or past N × target on the time axis. Mirrors
			// ffmpeg's stream-copy schedule for irregular source
			// GOPs (scene-change keyframes interspersed with the
			// regular cadence).
			name:      "absolute schedule with overshoot recovery",
			keyframes: []int64{0, 98, 194, 295, 391, 488, 587, 672, 695},
			fps:       FrameRate{Num: 24000, Den: 1001}, // 4s ≈ 95.9 frames
			target:    4 * time.Second,
			want:      []int64{98, 194, 295, 391, 488, 587, 672},
		},
		{
			// Quantization regression: with minFrames = ceil(4 ×
			// 24000/1001) = 96, an integer-frame schedule would put
			// segment 14's target at 14×96 = 1344, missing the
			// keyframe at 1343 which sits at PTS 1343×1001/24000 =
			// 56.014s ≥ the actual 56s target. The exact-rational
			// schedule picks 1343 (matching ffmpeg) instead of
			// jumping to 1369. Cut points up to segment 13 omitted
			// for brevity; only the 14th boundary is asserted.
			name: "exact arithmetic catches keyframes integer math drifts past",
			keyframes: []int64{
				0, 98, 194, 295, 391, 488, 587, 672, 787, 876, 965,
				1063, 1152, 1250, 1343, 1369, 1393,
			},
			fps:    FrameRate{Num: 24000, Den: 1001},
			target: 4 * time.Second,
			want: []int64{
				98, 194, 295, 391, 488, 587, 672, 787, 876, 965,
				1063, 1152, 1250, 1343,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SegmentBoundaries(tt.keyframes, tt.fps, tt.target)
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

func TestParseVideoPackets(t *testing.T) {
	// CSV format is "pts,duration,flags" per packet, in DTS order
	// — same as `ffprobe -show_packets -show_entries
	// packet=pts,duration,flags -of csv=p=0`.
	tests := []struct {
		name              string
		out               string
		wantKeyframes     []int64
		wantPacketCount   int64
		wantDurationTicks int64
	}{
		{
			name:              "empty",
			out:               "",
			wantKeyframes:     nil,
			wantPacketCount:   0,
			wantDurationTicks: 0,
		},
		{
			name: "closed GOP, no B-frames (DTS == display)",
			// 5 frames at 24000/1001 with 1/24000 timebase.
			out: "0,1001,K__\n" +
				"1001,1001,___\n" +
				"2002,1001,___\n" +
				"3003,1001,___\n" +
				"4004,1001,___\n",
			wantKeyframes:     []int64{0},
			wantPacketCount:   5,
			wantDurationTicks: 5005,
		},
		{
			name: "open GOP, 2-frame leading B-pyramid (Zeta-like)",
			// First GOP starts at IDR at display 0; second IDR is at
			// DTS index 4 but displays at 6 because two leading
			// B-frames at DTS 5,6 display at positions 4,5 (ahead of
			// the IDR in display order).
			//
			// DTS:    0  1  2  3  4  5  6  7
			// Display:0  ?  ?  ?  6  4  5  7
			// IDR2 at DTS=4 has the highest PTS of the first 5
			// packets, so post-sort it lands at display index 6.
			out: "0,1001,K__\n" +
				"1001,1001,___\n" +
				"2002,1001,___\n" +
				"3003,1001,___\n" +
				"6006,1001,K__\n" +
				"4004,1001,___\n" +
				"5005,1001,___\n" +
				"7007,1001,___\n",
			wantKeyframes:     []int64{0, 6},
			wantPacketCount:   8,
			wantDurationTicks: 8008,
		},
		{
			name: "missing PTS falls back to DTS-order keyframes",
			// Without complete PTS data we can't sort to display
			// order, so the keyframe list is whatever DTS gave us.
			out: "0,1001,K__\n" +
				",1001,___\n" +
				"2002,1001,K__\n",
			wantKeyframes:     []int64{0, 2},
			wantPacketCount:   3,
			wantDurationTicks: 3003, // (2002+1001) - 0
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKf, gotN, gotDur := parseVideoPackets(tt.out)
			if !slices.Equal(gotKf, tt.wantKeyframes) {
				t.Errorf("keyframes: got %v, want %v", gotKf, tt.wantKeyframes)
			}
			if gotN != tt.wantPacketCount {
				t.Errorf("packetCount: got %d, want %d", gotN, tt.wantPacketCount)
			}
			if gotDur != tt.wantDurationTicks {
				t.Errorf("durationTicks: got %d, want %d", gotDur, tt.wantDurationTicks)
			}
		})
	}
}

func TestVideoStreamCodedFrameRate(t *testing.T) {
	tests := []struct {
		name string
		vs   *VideoStream
		want FrameRate
	}{
		{
			name: "nil",
			vs:   nil,
			want: FrameRate{},
		},
		{
			name: "zero",
			vs:   &VideoStream{},
			want: FrameRate{},
		},
		{
			name: "CFR 24000/1001 in 1/1000 timebase",
			vs: &VideoStream{
				PacketCount:   600,
				DurationTicks: 25025,
				TimebaseNum:   1, TimebaseDen: 1000,
			},
			want: FrameRate{Num: 24000, Den: 1001},
		},
		{
			name: "CFR 24000/1001 in 1/24000 timebase",
			vs: &VideoStream{
				PacketCount:   600,
				DurationTicks: 600600,
				TimebaseNum:   1, TimebaseDen: 24000,
			},
			want: FrameRate{Num: 24000, Den: 1001},
		},
		{
			name: "soft-telecine (24fps coded under MKV 1/1000)",
			vs: &VideoStream{
				PacketCount:   1560, // 24000/1001 × 65.065s
				DurationTicks: 65065,
				TimebaseNum:   1, TimebaseDen: 1000,
			},
			want: FrameRate{Num: 24000, Den: 1001},
		},
		{
			name: "30/1 in 1/30 timebase",
			vs: &VideoStream{
				PacketCount:   300,
				DurationTicks: 300,
				TimebaseNum:   1, TimebaseDen: 30,
			},
			want: FrameRate{Num: 30, Den: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.vs.CodedFrameRate()
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
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
