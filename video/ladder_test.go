package video

import (
	"testing"

	"ily.dev/act3/priority"
	"ily.dev/act3/video/ffmpeg"
)

func TestPlanVideoRenditions_noVideo(t *testing.T) {
	probe := &ffmpeg.ProbeResult{}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	if rs != nil {
		t.Fatalf("expected nil, got %d renditions", len(rs))
	}
}

func TestPlanVideoRenditions_zeroBitrate(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{CodecName: "h264", BitRate: 0},
	}
	_, err := PlanVideoRenditions(probe)
	if err == nil {
		t.Fatal("expected error for zero bitrate")
	}
}

func TestPlanVideoRenditions_h264Remux(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{
			CodecName: "h264",
			BitRate:   30_000_000, // 30 Mbps
			Width:     1920,
			Height:    1080,
			FrameRate: ffmpeg.FrameRate{Num: 24000, Den: 1001},
		},
		Audio: []ffmpeg.AudioStream{{
			CodecName: "aac",
			Channels:  2,
		}},
	}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) == 0 {
		t.Fatal("expected renditions")
	}
	best := rs[0]
	if !best.Remux {
		t.Error("expected remux for h264 source below ceiling")
	}
	if best.Codec != "h264" {
		t.Errorf("expected h264 codec, got %s", best.Codec)
	}
	if best.TargetBitrate != 30_000 {
		t.Errorf("expected 30000 kbit/s, got %d", best.TargetBitrate)
	}
	if !best.CopyAudio {
		t.Error("expected CopyAudio for AAC stereo")
	}
	if best.Priority != priority.Encode1st {
		t.Errorf("expected priority %d, got %d", priority.Encode1st, best.Priority)
	}
	// All ladder entries below 30000 should be included.
	// Ladder: 20000, 5000, 3000, 1500, 420 — all below 30000.
	if len(rs) != 6 {
		t.Errorf("expected 6 renditions (best + 5 ladder), got %d", len(rs))
	}
}

func TestPlanVideoRenditions_hevcRemux(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{
			CodecName: "hevc",
			BitRate:   50_000_000,
			Width:     3840,
			Height:    2160,
			FrameRate: ffmpeg.FrameRate{Num: 30, Den: 1},
		},
		Audio: []ffmpeg.AudioStream{{
			CodecName: "flac",
			Channels:  2,
		}},
	}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	best := rs[0]
	if !best.Remux {
		t.Error("expected remux for hevc below ceiling")
	}
	if best.Codec != "hevc" {
		t.Errorf("expected hevc, got %s", best.Codec)
	}
	if best.CopyAudio {
		t.Error("expected CopyAudio=false for non-AAC audio")
	}
}

func TestPlanVideoRenditions_nonNativeCodec(t *testing.T) {
	// VP9 source below reencode threshold → reencode at 110%.
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{
			CodecName: "vp9",
			BitRate:   10_000_000, // 10 Mbps
			Width:     1920,
			Height:    1080,
			FrameRate: ffmpeg.FrameRate{Num: 30, Den: 1},
		},
	}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	best := rs[0]
	if best.Remux {
		t.Error("expected reencode for vp9")
	}
	if best.Codec != "hevc" {
		t.Errorf("expected hevc, got %s", best.Codec)
	}
	want := int64(10_000 * 11 / 10) // 11000
	if best.TargetBitrate != want {
		t.Errorf("expected %d kbit/s, got %d", want, best.TargetBitrate)
	}
}

func TestPlanVideoRenditions_highBitrateReencode(t *testing.T) {
	// Source above reencode threshold → cap at topTierCeiling.
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{
			CodecName: "vp9",
			BitRate:   600_000_000, // 600 Mbps
			Width:     3840,
			Height:    2160,
			FrameRate: ffmpeg.FrameRate{Num: 60, Den: 1},
		},
	}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	best := rs[0]
	if best.TargetBitrate != topTierCeiling {
		t.Errorf("expected %d, got %d", topTierCeiling, best.TargetBitrate)
	}
}

func TestPlanVideoRenditions_noSurroundDuplicate(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{
			CodecName: "h264",
			BitRate:   30_000_000,
			Width:     1920,
			Height:    1080,
			FrameRate: ffmpeg.FrameRate{Num: 24, Den: 1},
		},
		Audio: []ffmpeg.AudioStream{{
			CodecName: "ac3",
			Channels:  6,
		}},
	}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	best := rs[0]
	for i, r := range rs {
		if r.SurroundAudio {
			t.Errorf("rs[%d]: SurroundAudio should always be false", i)
		}
		if i > 0 && r.TargetBitrate == best.TargetBitrate {
			t.Errorf("rs[%d]: duplicate bitrate %d matches best", i, r.TargetBitrate)
		}
	}
}

func TestPlanVideoRenditions_lowFPSReduction(t *testing.T) {
	// ≤25fps content gets 20% bitrate reduction on ladder entries.
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{
			CodecName: "h264",
			BitRate:   25_000_000,
			Width:     1920,
			Height:    1080,
			FrameRate: ffmpeg.FrameRate{Num: 24, Den: 1}, // 24fps ≤ 25
		},
		Audio: []ffmpeg.AudioStream{{CodecName: "aac", Channels: 2}},
	}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	// First ladder entry is 20000 → reduced to 16000.
	// best is 25000 kbit/s, so 16000 < 25000 should be included.
	found := false
	for _, r := range rs[1:] {
		if r.TargetBitrate == 16_000 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ladder entry with 20% reduction (16000 kbit/s)")
		for i, r := range rs {
			t.Logf("  rs[%d]: bitrate=%d", i, r.TargetBitrate)
		}
	}
}

func TestPlanVideoRenditions_ladderSkipsAboveBest(t *testing.T) {
	// Source at 4 Mbps: ladder entries at 20000 and 5000 should be skipped.
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{
			CodecName: "h264",
			BitRate:   4_000_000,
			Width:     1280,
			Height:    720,
			FrameRate: ffmpeg.FrameRate{Num: 30, Den: 1},
		},
	}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rs[1:] {
		if r.TargetBitrate >= 4_000 {
			t.Errorf("ladder entry %d kbit/s should have been skipped (best is 4000)", r.TargetBitrate)
		}
	}
}

func TestPlanVideoRenditions_maxHeightResolved(t *testing.T) {
	// 4K source: ladder entries with MaxHeight caps should have them set.
	// 720p source: ladder entries with MaxHeight=1080 should resolve to 0.
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{
			CodecName: "h264",
			BitRate:   50_000_000,
			Width:     3840,
			Height:    2160,
			FrameRate: ffmpeg.FrameRate{Num: 60, Den: 1},
		},
	}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	// Find the 5000 kbit/s entry — ladder defines MaxHeight=1080 for it.
	for _, r := range rs {
		if r.TargetBitrate == 5_000 {
			if r.MaxHeight != 1080 {
				t.Errorf("5000 kbit/s: expected MaxHeight=1080, got %d", r.MaxHeight)
			}
			return
		}
	}
	t.Error("did not find 5000 kbit/s ladder entry")
}

func TestPlanVideoRenditions_maxHeightNotSetWhenSourceSmaller(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Video: &ffmpeg.VideoStream{
			CodecName: "h264",
			BitRate:   50_000_000,
			Width:     1280,
			Height:    720,
			FrameRate: ffmpeg.FrameRate{Num: 30, Den: 1},
		},
	}
	rs, err := PlanVideoRenditions(probe)
	if err != nil {
		t.Fatal(err)
	}
	// All ladder entries that have MaxHeight >= 720 should resolve to 0.
	for _, r := range rs[1:] {
		if r.MaxHeight == 1080 {
			t.Error("MaxHeight should not be 1080 for 720p source")
		}
	}
}

func TestFFmpegCodec(t *testing.T) {
	tests := []struct {
		codec string
		want  string
	}{
		{"h264", "libx264"},
		{"hevc", "libx265"},
		{"unknown", "libx265"},
	}
	for _, tt := range tests {
		r := &Rendition{Codec: tt.codec}
		if got := r.FFmpegCodec(); got != tt.want {
			t.Errorf("FFmpegCodec(%s) = %s, want %s", tt.codec, got, tt.want)
		}
	}
}

func TestVideoTag(t *testing.T) {
	if got := (&Rendition{Codec: "hevc"}).VideoTag(); got != "hvc1" {
		t.Errorf("VideoTag(hevc) = %q, want hvc1", got)
	}
	if got := (&Rendition{Codec: "h264"}).VideoTag(); got != "" {
		t.Errorf("VideoTag(h264) = %q, want empty", got)
	}
}

func TestHLSCodecs(t *testing.T) {
	got := (&Rendition{Codec: "h264"}).HLSCodecs()
	if got != "avc1.640028,mp4a.40.2" {
		t.Errorf("HLSCodecs(h264) = %q", got)
	}
	got = (&Rendition{Codec: "hevc"}).HLSCodecs()
	if got != "hvc1.1.6.L150.90,mp4a.40.2" {
		t.Errorf("HLSCodecs(hevc) = %q", got)
	}
}
