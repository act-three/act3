package ffmpeg

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestSegmentAlignment guards against a Chrome playback bug where
// ABR switches between HLS variants stalled around the first
// segment boundary because act3's variants disagreed on where
// segments ended. Re-encoded variants used libx264/libx265's
// default ~250-frame GOP (~10.4s on a 23.976fps source); remux
// variants inherited the source's keyframes. The HLS muxer cuts at
// keyframes, so the two variants ended up with different cut
// points. Chrome's MSE refused to bridge the resulting gap; Safari
// tolerated it.
//
// The fix probes the source's keyframes, picks canonical cut
// points (greedy: first keyframe at least minFrames after the
// previous cut) as source frame indices, and threads them through
// EncodeParams.SegmentBoundaries. Re-encodes are forced via
// -force_key_frames "expr:eq(n,N)" to keyframe at exactly those
// source frames; stream-copy already cuts at source keyframes by
// construction, so every variant agrees frame-for-frame.
//
// The test synthesises a 25s h264 source at 24000/1001fps with a
// 6s keyframe cadence and runs Pass1Combined + Pass2Single (one
// re-encode) and RemuxSingle (stream-copy) against it. Both
// variants must end every interior segment on exactly the same
// source frame as our chosen cut points — frame-exact, not
// approximate. We measure by reading the encoded media's keyframe
// frame indices directly (rather than EXTINF, which can disagree
// with the underlying fragment data when ffmpeg's HLS muxer writes
// irregular per-sample durations into the trun).
func TestSegmentAlignment(t *testing.T) {
	dir := setupAgent(t)
	preset := "ultrafast"
	ctx := t.Context()

	srcPath := filepath.Join(dir, "source.mkv")

	// Synthetic h264 source with a deterministic 6s keyframe
	// cadence. The four flags act in concert: -g 144 caps GOP
	// length (≈6s at 23.976fps), -keyint_min 144 prevents earlier
	// keyframes, -sc_threshold 0 disables scene-cut keyframes, and
	// -force_key_frames at every 6s mark nails down the boundary
	// exactly. testsrc2 is otherwise visually homogeneous enough
	// that without those flags libx264 picks its own cadence.
	t.Log("generating synthetic source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=25:size=320x180:rate=24000/1001",
		"-f", "lavfi", "-i",
		"sine=frequency=440:duration=25:sample_rate=48000",
		"-c:v", "libx264", "-preset", "ultrafast",
		"-g", "144", "-keyint_min", "144", "-sc_threshold", "0",
		"-force_key_frames", "expr:gte(t,n_forced*6)",
		"-c:a", "aac", "-ac", "2",
		srcPath,
	)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	t.Log("probing source...")
	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if probe.Video == nil {
		t.Fatal("probe: no video stream")
	}
	t.Logf("source: %dx%d %s @ %s fps, codec=%s, dur=%s",
		probe.Video.Width, probe.Video.Height, probe.FormatName,
		probe.Video.FrameRate, probe.Video.CodecName, probe.Duration)

	cuts := SegmentBoundaries(probe.Video.Keyframes, probe.Video.FrameRate, MinSegmentDuration)
	t.Logf("source keyframes: %v", probe.Video.Keyframes)
	t.Logf("segment boundaries (frame indices): %v", cuts)

	// Re-encode rendition: h264 at 540p (downsized from 320x180 is
	// a no-op, but matches the production ladder shape). With
	// SegmentBoundaries set, the encoder is forced to keyframe at
	// exactly the same source frames the remux variant cuts at.
	reencParams := EncodeParams{
		Path:                filepath.Join(dir, "reenc.mp4"),
		Codec:               "libx264",
		Bitrate:             500,
		MaxHeight:           540,
		StatsID:             "r0",
		SegmentBoundaries:   cuts,
		SegmentBoundaryRate: probe.Video.FrameRate,
	}

	t.Log("running re-encode pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{reencParams}, testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}
	t.Log("running re-encode pass 2...")
	if _, err := Pass2Single(ctx, srcFile, probe.FormatName,
		reencParams, testBatch, preset, probe.Duration, nil); err != nil {
		t.Fatalf("pass 2: %v", err)
	}

	// Remux rendition: stream-copy of the synthesised h264 stream.
	// Inherits the source's 6s keyframe cadence.
	remuxParams := EncodeParams{
		Path:  filepath.Join(dir, "remux.mp4"),
		Remux: true,
	}

	t.Log("running remux...")
	if _, err := RemuxSingle(ctx, srcFile, probe.FormatName,
		remuxParams, probe.Duration, nil); err != nil {
		t.Fatalf("remux: %v", err)
	}

	// Read each variant's actual keyframe frame indices from the
	// encoded media. These are what HLS players see when they
	// switch streams: frames N, M, ... will land on the same
	// timeline position iff the underlying media has keyframes at
	// the same display-order indices.
	reencFile, err := os.Open(reencParams.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer reencFile.Close()
	remuxFile, err := os.Open(remuxParams.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer remuxFile.Close()
	reencScan, err := scanPackets(ctx, reencFile, "mp4")
	if err != nil {
		t.Fatalf("probe re-encode keyframes: %v", err)
	}
	remuxScan, err := scanPackets(ctx, remuxFile, "mp4")
	if err != nil {
		t.Fatalf("probe remux keyframes: %v", err)
	}
	reencKf := reencScan.keyframes
	remuxKf := remuxScan.keyframes
	t.Logf("re-encode keyframes: %v", reencKf)
	t.Logf("remux keyframes:     %v", remuxKf)

	// Both variants must have exactly the keyframes our cuts asked
	// for, plus an implicit one at frame 0 (every encoded stream
	// starts with a keyframe).
	want := append([]int64{0}, cuts...)
	if !slices.Equal(reencKf, want) {
		t.Errorf("re-encode keyframes %v, want %v", reencKf, want)
	}
	if !slices.Equal(remuxKf, want) {
		t.Errorf("remux keyframes %v, want %v", remuxKf, want)
	}
}
