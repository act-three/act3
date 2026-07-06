package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ily.dev/act3/video/fenc"
)

// testBatch is the stats batch ID tests hand to Pass1Combined and
// the pass-2 functions. Each test gets its own agent stats root
// from setupAgent, so a fixed name never collides.
const testBatch = "b1"

// setupAgent configures the package to run ffmpeg/ffprobe through
// an in-process fenc agent backed by the host tools, and returns a
// working directory for test artifacts. The agent's spool and
// stats roots live under it, at "spool" and "stats"; a batch's
// pass-1 stats land in stats/<batch>/<StatsID>.
// Skips if no host ffmpeg is on PATH.
func setupAgent(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("ffmpeg test, skipped in -short mode")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("host ffmpeg not available")
	}
	dir := t.TempDir()
	spool := filepath.Join(dir, "spool")
	stats := filepath.Join(dir, "stats")
	for _, d := range []string{spool, stats} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	origAgent, origSpool := agent, agentSpool
	SetAgent(fenc.NewInProcessClient(&fenc.Server{Spool: spool, Stats: stats}), spool)
	t.Cleanup(func() { agent, agentSpool = origAgent, origSpool })
	return dir
}

// scanPackets stages f into a fresh job and scans its video
// packets, for tests that inspect the media they encoded.
func scanPackets(ctx context.Context, f *os.File, format string) (videoPacketScan, error) {
	j, err := newJob(f)
	if err != nil {
		return videoPacketScan{}, err
	}
	defer j.close()
	return scanVideoPackets(ctx, j, format)
}

// generate runs host ffmpeg directly to create a synthetic
// source file for testing.
func generate(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate source: %v\n%s", err, out)
	}
}

func requireFFmpegEncoder(t *testing.T, encoder string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "ffmpeg", "-hide_banner", "-encoders")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list ffmpeg encoders: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), " "+encoder+" ") {
		t.Skipf("ffmpeg encoder %q not available; install a host ffmpeg with that encoder", encoder)
	}
}

func TestEncodeAV1ToHEVC(t *testing.T) {
	if _, err := exec.LookPath("mediastreamvalidator"); err != nil {
		t.Skip("mediastreamvalidator not in PATH")
	}

	dir := setupAgent(t)
	requireFFmpegEncoder(t, "libsvtav1")
	preset := "ultrafast"
	ctx := t.Context()
	srcPath := filepath.Join(dir, "source.mkv")

	// Generate a tiny AV1 10-bit + EAC3 5.1 source, matching
	// the codec profile of our real content.
	t.Log("generating source...")
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

	t.Log("probing source...")
	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}

	mediaPath := filepath.Join(dir, MediaName(0))
	params := EncodeParams{
		Path:    mediaPath,
		Codec:   "libx265",
		Bitrate: 6000,
		Tag:     "hvc1",
		StatsID: "r0",
	}

	// Pass 1: analysis.
	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	// Pass 2: encode to HLS fMP4.
	t.Log("running pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}

	// Write playlist and validate with mediastreamvalidator.
	plsPath := filepath.Join(dir, "stream0.m3u8")
	if err := os.WriteFile(plsPath, []byte(playlist), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(
		"mediastreamvalidator", plsPath,
	).CombinedOutput()
	_ = err

	output := string(out)
	t.Logf("mediastreamvalidator output:\n%s", output)

	if strings.Contains(output, "Error injecting segment data") {
		t.Error("mediastreamvalidator reported: " +
			"Error injecting segment data")
	}
	if strings.Contains(output, "Processed 0 out of") {
		t.Error("mediastreamvalidator processed 0 segments")
	}
}

// TestEncodeMPEG2ToHEVC verifies that a synthetic MPEG2+MP2 source
// in MPEG-TS (typical of broadcast captures and DVD rips) encodes
// correctly to HEVC HLS fMP4.
func TestEncodeMPEG2ToHEVC(t *testing.T) {
	dir := setupAgent(t)
	preset := "ultrafast"
	ctx := t.Context()
	srcPath := filepath.Join(dir, "source.ts")

	// Generate a synthetic MPEG2 + MP2 source in MPEG-TS:
	// 720x480 at 29.97fps (NTSC DVD resolution).
	t.Log("generating MPEG2 source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=2:size=720x480:rate=30000/1001",
		"-f", "lavfi", "-i",
		"sine=frequency=440:duration=2:sample_rate=48000",
		"-c:v", "mpeg2video", "-b:v", "5000k",
		"-c:a", "mp2", "-b:a", "192k",
		"-f", "mpegts",
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

	mediaPath := filepath.Join(dir, MediaName(0))
	params := EncodeParams{
		Path:    mediaPath,
		Codec:   "libx265",
		Bitrate: 1500,
		Tag:     "hvc1",
		StatsID: "r0",
	}

	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	t.Log("running pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}

	// Verify media file is non-empty.
	info, err := os.Stat(mediaPath)
	if err != nil {
		t.Fatalf("media not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("media is empty")
	}

	// Validate with mediastreamvalidator if available.
	if _, err := exec.LookPath("mediastreamvalidator"); err == nil {
		plsPath := filepath.Join(dir, "stream0.m3u8")
		if err := os.WriteFile(plsPath, []byte(playlist), 0o644); err != nil {
			t.Fatal(err)
		}

		out, err := exec.Command(
			"mediastreamvalidator", plsPath,
		).CombinedOutput()
		_ = err

		output := string(out)
		t.Logf("mediastreamvalidator output:\n%s", output)

		if strings.Contains(output, "Error injecting segment data") {
			t.Error("mediastreamvalidator reported: " +
				"Error injecting segment data")
		}
		if strings.Contains(output, "Processed 0 out of") {
			t.Error("mediastreamvalidator processed 0 segments")
		}
	}
}

// TestMPEG2TelecineTwoPass verifies that 2-pass encoding works for
// MPEG-2 soft-telecine sources (DVD rips with repeat_first_field).
//
// Reproduces a production failure: the source MKV reports 59.94fps
// (from the MPEG-2 sequence header / MKV DefaultDuration) but
// contains only ~24fps of actual coded frames. Pass 1 outputs to
// -f null (no frame duplication), so x265 CU-tree stats cover
// ~24fps worth of frames. Pass 2 outputs to HLS fMP4 where the
// muxer forces the container frame rate (59.94fps), duplicating
// frames ~2.5×. x265 then runs out of CU-tree stats and fails
// with "Incomplete CU-tree stats file".
func TestMPEG2TelecineTwoPass(t *testing.T) {
	dir := setupAgent(t)
	preset := "ultrafast"
	ctx := t.Context()

	// Synthesise a source that mimics a DVD-rip MKV with soft
	// telecine: MPEG-2 at ~24fps actual coded frames, but the
	// MKV DefaultDuration claims 59.94fps.
	//
	// 1. Generate 24fps MPEG-2 + AC-3 in MPEG-TS.
	// 2. Patch the MPEG-2 sequence-header frame_rate to
	//    60000/1001 (≈59.94fps) via the mpeg2_metadata BSF.
	//    This only changes the bitstream, not the PTS values.
	// 3. Remux to MKV with -default_mode passthrough so the
	//    muxer copies the (now-patched) codec frame rate into
	//    the track's DefaultDuration. The result is an MKV
	//    where r_frame_rate=59.94 but actual timestamps are
	//    at ~24fps — exactly the production scenario.
	tsPath := filepath.Join(dir, "step1.ts")
	patchedPath := filepath.Join(dir, "step2.ts")
	srcPath := filepath.Join(dir, "source.mkv")

	t.Log("generating 24fps MPEG-2 source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=2:size=720x480:rate=24000/1001",
		"-f", "lavfi", "-i",
		"sine=frequency=440:duration=2:sample_rate=48000",
		"-c:v", "mpeg2video", "-b:v", "5000k",
		"-c:a", "ac3", "-ac", "2",
		"-f", "mpegts", tsPath,
	)

	t.Log("patching MPEG-2 frame_rate to 59.94fps...")
	generate(t,
		"-y", "-i", tsPath,
		"-c", "copy",
		"-bsf:v", "mpeg2_metadata=frame_rate=60000/1001",
		"-f", "mpegts", patchedPath,
	)

	t.Log("remuxing to MKV with passthrough DefaultDuration...")
	generate(t,
		"-y", "-i", patchedPath,
		"-c", "copy",
		"-default_mode", "passthrough",
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
	t.Logf("probe: %dx%d, %s fps, %d kbps",
		probe.Video.Width, probe.Video.Height,
		probe.Video.FrameRate, probe.Video.BitRate/1000)

	// Sanity-check: r_frame_rate should be ~59.94fps.
	if probe.Video.FrameRate.Le(30) {
		t.Fatalf("expected r_frame_rate > 30, got %s",
			probe.Video.FrameRate)
	}

	params := EncodeParams{
		Path:    filepath.Join(dir, MediaName(0)),
		Codec:   "libx265",
		Bitrate: 5000,
		Tag:     "hvc1",
		StatsID: "r0",
	}

	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	t.Log("running pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}
}

// TestX264MbtreePass2 verifies that x264 2-pass encoding works when
// pass 1 writes stats (including .mbtree) directly to a persistent
// directory and pass 2 reads from the same path.
func TestX264MbtreePass2(t *testing.T) {
	dir := setupAgent(t)
	preset := "medium"
	ctx := t.Context()

	srcPath := filepath.Join(dir, "source.mkv")

	// Generate a tiny H.264 + AAC source.
	t.Log("generating source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=2:size=160x90:rate=24",
		"-f", "lavfi", "-i",
		"sine=frequency=440:duration=2:sample_rate=48000",
		"-c:v", "libx264", "-preset", "ultrafast",
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

	mediaPath := filepath.Join(dir, MediaName(0))
	params := EncodeParams{
		Path:    mediaPath,
		Codec:   "libx264",
		Bitrate: 500,
		StatsID: "r0",
	}
	passlog := filepath.Join(dir, "stats", testBatch, params.StatsID)

	// Pass 1: x264 analysis writes stats + .mbtree directly.
	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	// Verify stats files were written. White-box check against the
	// layout ffmpeg owns inside statsDir.
	if _, err := os.Stat(passlog); err != nil {
		t.Fatalf("stats file not written: %v", err)
	}
	if _, err := os.Stat(passlog + ".mbtree"); err != nil {
		t.Fatal("expected .mbtree file from x264 pass 1")
	}

	// Pass 2: encode reading stats from the same dir.
	t.Log("running pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}

	t.Log("pass 2 succeeded — x264 mbtree stats OK")
}

func TestHDRFormat(t *testing.T) {
	tests := []struct {
		dvProfile int
		transfer  string
		want      string
	}{
		{0, "", ""},
		{0, "bt709", ""},
		{0, "smpte2084", "PQ"},
		{0, "arib-std-b67", "HLG"},
		{5, "", "PQ"},          // DV P5: converted to HDR10; source is untagged
		{8, "smpte2084", "PQ"}, // DV P8: HDR10-compatible base layer
	}
	for _, tt := range tests {
		if got := HDRFormat(tt.dvProfile, tt.transfer); got != tt.want {
			t.Errorf("HDRFormat(%d, %q) = %q, want %q",
				tt.dvProfile, tt.transfer, got, tt.want)
		}
	}
}

// TestHDR10EncodePreservesColor runs a native HDR10 source through the
// real two-pass encode path — including a scale branch, so the
// filtergraph is exercised — and verifies the output keeps its 10-bit
// BT.2020 PQ color tags without any explicit color arguments. This is
// the contract PlanVideoRenditions relies on when it labels every
// rendition of an HDR source with the source's VIDEO-RANGE.
//
// The source is tagged via setparams: frame-level color properties are
// what the encoder writes into the VUI; ffmpeg 8's -color* output
// flags no longer reach it.
func TestHDR10EncodePreservesColor(t *testing.T) {
	dir := setupAgent(t)
	preset := "ultrafast"
	ctx := t.Context()
	srcPath := filepath.Join(dir, "source.mkv")

	t.Log("generating HDR10 source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i", "testsrc2=duration=2:size=320x180:rate=24",
		"-vf", "setparams=colorspace=bt2020nc:color_primaries=bt2020:color_trc=smpte2084:range=tv,format=yuv420p10le",
		"-c:v", "libx265", "-preset", "ultrafast",
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
	if probe.Video.ColorTransfer != "smpte2084" {
		t.Fatalf("probe ColorTransfer = %q, want smpte2084", probe.Video.ColorTransfer)
	}

	mediaPath := filepath.Join(dir, MediaName(0))
	params := EncodeParams{
		Path:      mediaPath,
		Codec:     "libx265",
		Bitrate:   500,
		MaxHeight: 90, // force the scale filtergraph
		Tag:       "hvc1",
		StatsID:   "r0",
	}

	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}
	t.Log("running pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}

	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries",
		"stream=profile,pix_fmt,color_transfer,color_primaries,color_space",
		"-of", "default=noprint_wrappers=1", mediaPath)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("probe output: %v\n%s", err, buf.String())
	}
	out := buf.String()
	t.Logf("encoded stream:\n%s", out)
	for _, want := range []string{"Main 10", "yuv420p10le", "smpte2084", "bt2020"} {
		if !strings.Contains(out, want) {
			t.Errorf("HDR10 output missing %q", want)
		}
	}
}

func TestBuildFilterComplexDolbyVision(t *testing.T) {
	// No filtering and no Dolby Vision: direct map, no filter_complex.
	plain, labels := buildFilterComplex([]EncodeParams{{Codec: "libx265"}})
	if plain != "" {
		t.Errorf("plain reencode: want empty filter, got %q", plain)
	}
	if labels[0] != "0:v:0" {
		t.Errorf("plain reencode: want label 0:v:0, got %q", labels[0])
	}

	// Dolby Vision forces a graph even with no scaling, and the
	// libplacebo head must download a single-plane packed format
	// (rgba64le) rather than multiplanar YUV — software Vulkan returns
	// garbage chroma for the latter.
	dv, labels := buildFilterComplex([]EncodeParams{{Codec: "libx265", DolbyVision: true}})
	if !strings.Contains(dv, "apply_dolbyvision=true") {
		t.Errorf("DV: filter missing libplacebo DV pass: %q", dv)
	}
	if !strings.Contains(dv, "format=rgba64le") {
		t.Errorf("DV: filter missing rgba64le download workaround: %q", dv)
	}
	if strings.Contains(dv, "[dv]split=") == false {
		t.Errorf("DV: expected the DV output to feed the split: %q", dv)
	}
	if !strings.HasPrefix(labels[0], "[") {
		t.Errorf("DV: expected a filtergraph label, got %q", labels[0])
	}

	// Remux renditions never carry the DV flag into the graph.
	remux, labels := buildFilterComplex([]EncodeParams{{Remux: true}})
	if remux != "" || len(labels) != 0 {
		t.Errorf("remux: want no filter and no labels, got %q / %v", remux, labels)
	}
}

// TestDolbyVisionEncodeHDR10 runs a Dolby Vision Profile 5 source
// through the real two-pass encode path and verifies the output is
// 10-bit HDR10 (Main 10, BT.2020 PQ).
//
// The source is fully synthetic: a testsrc2 pattern encoded to 10-bit
// HEVC, a Profile 5 RPU generated and injected by dovi_tool, and the
// stream muxed by mkvmerge, which writes the DOVI configuration
// record the probe detects. dovi_tool and mkvmerge live in the
// act3-ffmpeg image alongside ffmpeg; the test skips when they aren't
// available (host-ffmpeg fallback, or a stale image).
func TestDolbyVisionEncodeHDR10(t *testing.T) {
	dir := setupAgent(t)
	preset := "ultrafast"
	ctx := t.Context()

	if err := exec.CommandContext(ctx, "dovi_tool", "--version").Run(); err != nil {
		t.Skip("dovi_tool not available (rebuild the act3-ffmpeg image)")
	}
	runTool := func(name string, args ...string) {
		t.Helper()
		if out, err := exec.CommandContext(ctx, name, args...).CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", name, err, out)
		}
	}

	// Synthesize a Profile 5 source. The RPU length must match the
	// encoded frame count (duration × rate).
	const frames = 48
	basePath := filepath.Join(dir, "base.hevc")
	runTool("ffmpeg",
		"-y", "-f", "lavfi", "-i", "testsrc2=duration=2:size=160x90:rate=24",
		"-pix_fmt", "yuv420p10le",
		"-c:v", "libx265", "-preset", "ultrafast",
		"-f", "hevc", basePath,
	)
	genPath := filepath.Join(dir, "gen.json")
	genConfig := fmt.Sprintf(`{
		"cm_version": "V29",
		"length": %d,
		"level6": {
			"max_display_mastering_luminance": 1000,
			"min_display_mastering_luminance": 1,
			"max_content_light_level": 1000,
			"max_frame_average_light_level": 400
		}
	}`, frames)
	if err := os.WriteFile(genPath, []byte(genConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	rpuPath := filepath.Join(dir, "rpu.bin")
	runTool("dovi_tool", "generate", "-j", genPath, "--profile", "5", "-o", rpuPath)
	dvPath := filepath.Join(dir, "dv.hevc")
	runTool("dovi_tool", "inject-rpu", "-i", basePath, "--rpu-in", rpuPath, "-o", dvPath)
	srcPath := filepath.Join(dir, "source.mkv")
	runTool("mkvmerge", "-q", "-o", srcPath, dvPath)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if probe.Video == nil || probe.Video.DolbyVisionProfile != 5 {
		t.Fatalf("expected Dolby Vision profile 5 source, got %+v", probe.Video)
	}

	mediaPath := filepath.Join(dir, MediaName(0))
	params := EncodeParams{
		Path:        mediaPath,
		Codec:       "libx265",
		Bitrate:     1000,
		Tag:         "hvc1",
		StatsID:     "r0",
		DolbyVision: true,
	}

	t.Log("pass 1 (libplacebo DV → HDR10 on software Vulkan)...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}
	t.Log("pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}

	// The encoded media must be 10-bit HDR10 (Main 10, BT.2020 PQ).
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries",
		"stream=profile,pix_fmt,color_transfer,color_primaries,color_space",
		"-of", "default=noprint_wrappers=1", mediaPath)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("probe output: %v\n%s", err, buf.String())
	}
	out := buf.String()
	t.Logf("encoded stream:\n%s", out)
	for _, want := range []string{"Main 10", "yuv420p10le", "smpte2084", "bt2020"} {
		if !strings.Contains(out, want) {
			t.Errorf("HDR10 output missing %q", want)
		}
	}
}
