package ffmpeg

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEncoderHonorsManyForcedKeyframes verifies that when keyframeArgs
// is handed a long cut list, ffmpeg/x265 actually emits a keyframe at
// EACH requested source frame — not just the first ~16.
//
// Production reports degenerate 16-segment playlists on long episodes
// even with cuts at the standard 96-frame (4s) interval. The HLS
// muxer's -hls_time can mask missing keyframes (a segment grows to
// the next available cut), so reading playlist segment count is not
// enough. Read the encoded media's keyframe indices directly.
func TestEncoderHonorsManyForcedKeyframes(t *testing.T) {
	dir := setupDocker(t)
	setPreset(t, "ultrafast")
	ctx := t.Context()

	srcPath := filepath.Join(dir, "source.mkv")

	// 5-minute 24fps source = 7200 frames. Cuts every 96 frames
	// (4 seconds) matches what UniformSegmentBoundaries returns in
	// production. 7200/96 ≈ 75 cuts. Spaced well above x265's
	// auto-min-keyint (=fps=24), so every cut should be honored.
	const sourceSeconds = 300
	t.Log("generating 5min source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=300:size=320x180:rate=24",
		"-c:v", "libx264", "-preset", "ultrafast",
		"-pix_fmt", "yuv420p",
		srcPath,
	)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}

	rate := probe.Video.CodedFrameRate()
	if !rate.Positive() {
		rate = probe.Video.FrameRate
	}

	var cuts []int64
	const step = int64(96)
	for f := step; f < sourceSeconds*24; f += step {
		cuts = append(cuts, f)
	}
	t.Logf("requesting %d cuts at %s, last cut frame %d",
		len(cuts), rate, cuts[len(cuts)-1])

	params := EncodeParams{
		Path:                filepath.Join(dir, "out.mp4"),
		Codec:               "libx265",
		Bitrate:             500,
		Tag:                 "hvc1",
		StatsID:             "r0",
		SegmentBoundaries:   cuts,
		SegmentBoundaryRate: rate,
	}

	t.Log("running pass 1...")
	if err := Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, dir, probe.Duration, nil); err != nil {
		t.Fatalf("pass 1: %v", err)
	}
	t.Log("running pass 2...")
	if _, err := Pass2Single(ctx, srcFile, probe.FormatName,
		params, dir, probe.Duration, nil); err != nil {
		t.Fatalf("pass 2: %v", err)
	}

	outFile, err := os.Open(params.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer outFile.Close()
	scan, err := scanVideoPackets(ctx, outFile, "mp4")
	if err != nil {
		t.Fatalf("scan output: %v", err)
	}
	t.Logf("encoded output: %d packets, %d keyframes",
		scan.packetCount, len(scan.keyframes))
	t.Logf("first 5 keyframes: %v", scan.keyframes[:min(5, len(scan.keyframes))])
	if len(scan.keyframes) > 5 {
		t.Logf("last 5 keyframes:  %v",
			scan.keyframes[len(scan.keyframes)-5:])
	}

	// Encoder always emits a keyframe at frame 0 implicitly. The
	// remaining keyframes should match cuts one-for-one.
	want := len(cuts) + 1
	if len(scan.keyframes) < want {
		t.Errorf("encoded file has %d keyframes, want %d (initial + %d cuts)",
			len(scan.keyframes), want, len(cuts))
	}
}
