package ffmpeg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestKeyframeArgsTimeForm checks the comma-separated time form
// keyframeArgs emits: each cut becomes a timestamp half a frame
// before the target frame's PTS, so ffmpeg's "first frame after
// time" rule lands the keyframe on the target frame exactly.
func TestKeyframeArgsTimeForm(t *testing.T) {
	tests := []struct {
		name      string
		cuts      []int64
		rate      FrameRate
		wantForce string
	}{
		{
			name: "empty",
			cuts: nil,
			rate: FrameRate{Num: 24, Den: 1},
		},
		{
			name: "24fps single cut",
			cuts: []int64{96},
			rate: FrameRate{Num: 24, Den: 1},
			// Shortest decimal that round-trips to the float64
			// closest to the exact rational (96-0.5)/24 = 191/48.
			wantForce: "3.9791666666666665",
		},
		{
			name: "23.976 (24000/1001) two cuts",
			cuts: []int64{96, 192},
			rate: FrameRate{Num: 24000, Den: 1001},
			// (k-0.5)*1001/24000, round-trip shortest.
			wantForce: "3.9831458333333334,7.987145833333333",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forceArgs, extras := keyframeArgs(tt.cuts, tt.rate)
			if len(tt.cuts) == 0 {
				if forceArgs != nil || extras != nil {
					t.Errorf("empty cuts: got %v, %v; want nil, nil", forceArgs, extras)
				}
				return
			}
			if len(forceArgs) != 2 || forceArgs[0] != "-force_key_frames" {
				t.Fatalf("forceArgs = %v, want [-force_key_frames, ...]", forceArgs)
			}
			if forceArgs[1] != tt.wantForce {
				t.Errorf("time list = %q, want %q", forceArgs[1], tt.wantForce)
			}
			wantExtras := []string{"keyint=99999", "scenecut=0"}
			if len(extras) != len(wantExtras) {
				t.Fatalf("extras = %v, want %v", extras, wantExtras)
			}
			for i, e := range extras {
				if e != wantExtras[i] {
					t.Errorf("extras[%d] = %q, want %q", i, e, wantExtras[i])
				}
			}
		})
	}
}

// TestKeyframeArgsScales locks in the property that broke before
// the fix: an arbitrarily long cut list produces a well-formed
// argument. The earlier expr:eq(n,N1)+eq(n,N2)+... form tripped
// libavutil/eval.c's alternation-term limit at 101 OR terms; the
// comma-separated time form has no such limit.
func TestKeyframeArgsScales(t *testing.T) {
	cuts := make([]int64, 1000)
	for i := range cuts {
		cuts[i] = int64(i + 1)
	}
	rate := FrameRate{Num: 24, Den: 1}
	forceArgs, _ := keyframeArgs(cuts, rate)
	if len(forceArgs) != 2 {
		t.Fatalf("forceArgs = %v, want 2 elements", forceArgs)
	}
	got := forceArgs[1]
	if strings.HasPrefix(got, "expr:") {
		t.Errorf("argument starts with %q; want comma-time form, "+
			"not the bounded-length expr form", "expr:")
	}
	if n := strings.Count(got, ",") + 1; n != len(cuts) {
		t.Errorf("got %d comma-separated values, want %d", n, len(cuts))
	}
}

func TestKeyframeArgsPanicsOnZeroRate(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("keyframeArgs with zero rate did not panic")
		}
	}()
	keyframeArgs([]int64{96}, FrameRate{})
}

// TestPass1AcceptsManyCuts is the integration regression guard for
// the production playback bug seen on Andor S1E1 and any other
// source long enough to produce >100 segment cut points.
//
// Before the fix, keyframeArgs emitted a single
//
//	-force_key_frames "expr:eq(n,N1)+eq(n,N2)+...+eq(n,Nk)"
//
// argument whose length scaled with len(cuts). libavutil/eval.c's
// expression parser rejected expressions with more than ~100 OR
// terms at output-init time, so any episode longer than ~6.7
// minutes at our 4s target was unencodable — and the production
// Andor playlist that started this hunt was a historical artifact
// from an earlier code path that hadn't yet hit the cap.
//
// After the fix, keyframeArgs emits the comma-separated time form,
// which has no parser limit. The test feeds Pass1Combined a
// 101-element cut list — one term past the old cliff — and
// expects success.
func TestPass1AcceptsManyCuts(t *testing.T) {
	dir := setupHost(t)
	requireFFmpegEncoder(t, "libsvtav1")
	setPreset(t, "ultrafast")
	ctx := t.Context()

	srcPath := filepath.Join(dir, "source.mkv")

	// Tiny AV1 10-bit + EAC3 5.1 source, same profile as production.
	t.Log("generating synthetic source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=2:size=160x90:rate=24",
		"-f", "lavfi", "-i",
		"sine=frequency=440:duration=2:sample_rate=48000",
		"-c:v", "libsvtav1", "-preset", "12", "-crf", "55",
		"-pix_fmt", "yuv420p10le",
		"-c:a", "eac3", "-ac", "6",
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

	// 101 cuts is one past libavutil/eval.c's old alternation-term
	// limit. The values themselves don't matter: cuts past the end
	// of this tiny source simply never fire. ffmpeg parses the
	// -force_key_frames argument before reading any frame, so this
	// is enough to exercise the format path.
	const numCuts = 101
	cuts := make([]int64, numCuts)
	for i := range cuts {
		cuts[i] = int64(i + 1)
	}

	params := EncodeParams{
		Path:                filepath.Join(dir, MediaName(0)),
		Codec:               "libx265",
		Bitrate:             500,
		Tag:                 "hvc1",
		StatsID:             "r0",
		SegmentBoundaries:   cuts,
		SegmentBoundaryRate: FrameRate{Num: 24, Den: 1},
	}

	t.Logf("running pass 1 with %d-element cut list...", len(cuts))
	if err := Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, dir, probe.Duration, nil); err != nil {
		t.Fatalf("Pass1Combined with %d cuts: %v", len(cuts), err)
	}
}
