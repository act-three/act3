package ffmpeg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const dockerImage = "act3-ffmpeg"

// setupDocker configures the package to run ffmpeg/ffprobe inside
// the act3-ffmpeg Docker container and returns a working directory
// that is bind-mounted into the container at the same host path.
//
// If docker is unavailable but a host ffmpeg is on PATH, falls back
// to running ffmpeg/ffprobe directly via the default newCmd. Skips
// if neither is available.
func setupDocker(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Temp dirs created by production code (e.g. Pass2Single) must
	// land under dir so they are accessible inside the container
	// (and stay together for the host fallback too).
	t.Setenv("TMPDIR", dir)

	if _, err := exec.LookPath("docker"); err == nil {
		if err := exec.Command("docker", "image", "inspect",
			dockerImage).Run(); err == nil {
			origCmd := newCmd
			newCmd = dockerCmd(dir)
			t.Cleanup(func() { newCmd = origCmd })
			return dir
		}
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("neither docker+act3-ffmpeg nor host ffmpeg available")
	}
	return dir
}

// setupHost is like setupDocker but never routes through the
// container, even when docker and the act3-ffmpeg image are
// available. Use it for tests that hit a code path Docker's stdin
// pipe breaks — e.g. mpegts probing, where the demuxer needs a
// seekable input to compute duration but `docker run -i` always
// pipes stdin through the daemon.
func setupHost(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("host ffmpeg not available")
	}
	return dir
}

// dockerCmd returns a newCmd replacement that runs tools inside
// Docker, with dir bind-mounted at the same host path.
func dockerCmd(dir string) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Strip -progress pipe:3 and -nostats: the progress
		// pipe fd is not forwarded into the Docker container.
		var filtered []string
		for i := 0; i < len(args); i++ {
			if args[i] == "-progress" && i+1 < len(args) {
				i++ // skip value
				continue
			}
			if args[i] == "-nostats" {
				continue
			}
			filtered = append(filtered, args[i])
		}

		a := []string{
			"run", "--rm", "-i",
			"-v", dir + ":" + dir,
			dockerImage,
			"/out/" + name,
		}
		a = append(a, filtered...)
		return exec.CommandContext(ctx, "docker", a...)
	}
}

// setPreset overrides the ffmpeg video preset for the duration
// of the test.
func setPreset(t *testing.T, preset string) {
	t.Helper()
	orig := overridePreset
	overridePreset = preset
	t.Cleanup(func() { overridePreset = orig })
}

// generate runs ffmpeg (through newCmd) to create a synthetic
// source file for testing.
func generate(t *testing.T, args ...string) {
	t.Helper()
	cmd := newCmd(t.Context(), "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate source: %v\n%s", err, out)
	}
}

func TestTwoPassArgsRejectsInjection(t *testing.T) {
	for _, passlog := range []string{
		"/foo/bar:crf=0",
		"/foo\\bar",
		"/foo\nbar",
	} {
		t.Run(passlog, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("twoPassArgs(%q) did not panic", passlog)
				}
			}()
			twoPassArgs("libx265", 1, passlog)
		})
	}
}

func TestEncodeAV1ToHEVC(t *testing.T) {
	if _, err := exec.LookPath("mediastreamvalidator"); err != nil {
		t.Skip("mediastreamvalidator not in PATH")
	}

	dir := setupDocker(t)
	setPreset(t, "ultrafast")
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
	outFile, err := os.Create(mediaPath)
	if err != nil {
		t.Fatal(err)
	}

	params := EncodeParams{
		File:    outFile,
		Codec:   "libx265",
		Bitrate: 6000,
		Tag:     "hvc1",
		StatsID: "r0",
	}

	// Pass 1: analysis.
	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, dir, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	// Pass 2: encode to HLS fMP4.
	t.Log("running pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		dir, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}
	outFile.Close()

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
// correctly to HEVC HLS fMP4. Always runs against host ffmpeg
// because Probe needs a seekable stdin to compute mpegts duration,
// and `docker run -i` strips that.
func TestEncodeMPEG2ToHEVC(t *testing.T) {
	dir := setupHost(t)
	setPreset(t, "ultrafast")
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
	outFile, err := os.Create(mediaPath)
	if err != nil {
		t.Fatal(err)
	}

	params := EncodeParams{
		File:    outFile,
		Codec:   "libx265",
		Bitrate: 1500,
		Tag:     "hvc1",
		StatsID: "r0",
	}

	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, dir, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	t.Log("running pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		dir, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}
	outFile.Close()

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
	dir := setupDocker(t)
	setPreset(t, "ultrafast")
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

	mediaFile, err := os.Create(filepath.Join(dir, MediaName(0)))
	if err != nil {
		t.Fatal(err)
	}
	defer mediaFile.Close()

	params := EncodeParams{
		File:    mediaFile,
		Codec:   "libx265",
		Bitrate: 5000,
		Tag:     "hvc1",
		StatsID: "r0",
	}

	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, dir, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	t.Log("running pass 2...")
	playlist, err := Pass2Single(ctx, srcFile, probe.FormatName, params,
		dir, probe.Duration, nil)
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
	dir := setupDocker(t)
	setPreset(t, "medium")
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
	outFile, err := os.Create(mediaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer outFile.Close()

	params := EncodeParams{
		File:    outFile,
		Codec:   "libx264",
		Bitrate: 500,
		StatsID: "r0",
	}
	passlog := filepath.Join(dir, params.StatsID)

	// Pass 1: x264 analysis writes stats + .mbtree directly.
	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{params}, dir, probe.Duration, nil)
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
		dir, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 2: %v", err)
	}
	if playlist == "" {
		t.Fatal("empty playlist")
	}

	t.Log("pass 2 succeeded — x264 mbtree stats OK")
}
