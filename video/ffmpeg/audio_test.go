package ffmpeg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEncodeAudioValidation verifies argument validation rejects bad
// inputs without invoking ffmpeg. Runs without Docker.
func TestEncodeAudioValidation(t *testing.T) {
	cases := []struct {
		name string
		dst  AudioEncodeParams
	}{
		{
			name: "negative source stream index",
			dst:  AudioEncodeParams{SourceStreamIndex: -1, Channels: 2, Bitrate: 128},
		},
		{
			name: "channels=0",
			dst:  AudioEncodeParams{SourceStreamIndex: 0, Channels: 0, Bitrate: 128},
		},
		{
			name: "negative channels",
			dst:  AudioEncodeParams{SourceStreamIndex: 0, Channels: -1, Bitrate: 128},
		},
		{
			name: "zero bitrate, no stream copy",
			dst:  AudioEncodeParams{SourceStreamIndex: 0, Channels: 2, Bitrate: 0},
		},
		{
			name: "negative bitrate",
			dst:  AudioEncodeParams{SourceStreamIndex: 0, Channels: 2, Bitrate: -1},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("EncodeAudio(%+v) did not panic", c.dst)
				}
			}()
			EncodeAudio(t.Context(), nil, "matroska,webm", c.dst, 0, nil)
		})
	}
}

// TestStandardLayout pins the channel counts that map to a PCE-free
// AAC channel configuration. Runs without Docker.
func TestStandardLayout(t *testing.T) {
	cases := map[int]string{
		1: "mono", 2: "stereo", 3: "3.0", 4: "4.0",
		5: "5.0", 6: "5.1", 8: "7.1",
		0: "", 7: "", 9: "", 12: "",
	}
	for channels, want := range cases {
		if got := standardLayout[channels]; got != want {
			t.Errorf("standardLayout[%d] = %q, want %q", channels, got, want)
		}
	}
}

// audioCase describes one synthetic-source ↔ encode scenario.
type audioCase struct {
	name string
	// generate args (after "-y", before the output path).
	srcArgs   []string
	srcExt    string
	dst       AudioEncodeParams
	wantCodec string
	wantCh    int
	// wantLayout is checked when non-empty.
	wantLayout string
}

func runAudioCase(t *testing.T, dir string, c audioCase) {
	t.Helper()
	ctx := t.Context()
	srcPath := filepath.Join(dir, c.name+"."+c.srcExt)
	args := append([]string{"-y"}, c.srcArgs...)
	args = append(args, srcPath)
	t.Logf("generating source: %v", args)
	generate(t, args...)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe source: %v", err)
	}
	if len(probe.Audio) == 0 {
		t.Fatal("source has no audio stream")
	}
	t.Logf("source audio: codec=%s channels=%d layout=%s",
		probe.Audio[0].CodecName, probe.Audio[0].Channels, probe.Audio[0].ChannelLayout)

	mediaPath := filepath.Join(dir, c.name+"-out.mp4")
	c.dst.Path = mediaPath

	playlist, err := EncodeAudio(ctx, srcFile, probe.FormatName, c.dst,
		probe.Duration, nil)
	if err != nil {
		t.Fatalf("EncodeAudio: %v", err)
	}

	if playlist == "" {
		t.Fatal("empty playlist")
	}
	if !strings.Contains(playlist, "#EXTM3U") {
		t.Errorf("playlist missing #EXTM3U:\n%s", playlist)
	}
	if !strings.Contains(playlist, "#EXT-X-MAP:") {
		t.Errorf("playlist missing fMP4 init segment (#EXT-X-MAP):\n%s", playlist)
	}
	if !strings.Contains(playlist, MediaName(0)) {
		t.Errorf("playlist missing media reference %q:\n%s", MediaName(0), playlist)
	}
	if !strings.Contains(playlist, "#EXT-X-ENDLIST") {
		t.Errorf("playlist missing #EXT-X-ENDLIST (VOD):\n%s", playlist)
	}

	info, err := os.Stat(mediaPath)
	if err != nil {
		t.Fatalf("media not written: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("media file is empty")
	}

	// Probe the output to confirm codec / channels / layout.
	outFp, err := os.Open(mediaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer outFp.Close()
	outProbe, err := Probe(ctx, outFp)
	if err != nil {
		t.Fatalf("probe output: %v", err)
	}
	if len(outProbe.Audio) == 0 {
		t.Fatal("output has no audio stream")
	}
	a := outProbe.Audio[0]
	t.Logf("output audio: codec=%s channels=%d layout=%s",
		a.CodecName, a.Channels, a.ChannelLayout)
	if a.CodecName != c.wantCodec {
		t.Errorf("output codec = %q, want %q", a.CodecName, c.wantCodec)
	}
	if a.Channels != c.wantCh {
		t.Errorf("output channels = %d, want %d", a.Channels, c.wantCh)
	}
	if c.wantLayout != "" && a.ChannelLayout != c.wantLayout {
		t.Errorf("output channel layout = %q, want %q", a.ChannelLayout, c.wantLayout)
	}
}

// TestEncodeAudio runs EncodeAudio against several synthetic source
// configurations, covering the three encoder paths the spec calls out:
// stream-copy AAC stereo, re-encode of FLAC stereo, re-encode of AAC
// 5.1, and re-encode of AC3 5.1. Requires Docker (act3-ffmpeg image).
func TestEncodeAudio(t *testing.T) {
	dir := setupDocker(t)

	commonVideo := []string{
		"-f", "lavfi", "-i",
		"testsrc2=duration=2:size=160x90:rate=24",
	}

	cases := []audioCase{
		{
			name: "aac-stereo-copy",
			srcArgs: append(append([]string{}, commonVideo...),
				"-f", "lavfi", "-i",
				"sine=frequency=440:duration=2:sample_rate=48000",
				"-c:v", "libx264", "-preset", "ultrafast",
				"-c:a", "aac", "-ac", "2", "-b:a", "128k",
				"-t", "2",
			),
			srcExt:    "mkv",
			dst:       AudioEncodeParams{SourceStreamIndex: 0, Channels: 2, Bitrate: 128, StreamCopy: true},
			wantCodec: "aac",
			wantCh:    2,
		},
		{
			name: "aac-5_1-reencode",
			srcArgs: append(append([]string{}, commonVideo...),
				"-f", "lavfi", "-i",
				"aevalsrc=exprs=sin(440*2*PI*t):duration=2:sample_rate=48000:channel_layout=5.1",
				"-c:v", "libx264", "-preset", "ultrafast",
				"-c:a", "aac", "-b:a", "384k",
				"-t", "2",
			),
			srcExt:     "mkv",
			dst:        AudioEncodeParams{SourceStreamIndex: 0, Channels: 6, Bitrate: 384},
			wantCodec:  "aac",
			wantCh:     6,
			wantLayout: "5.1",
		},
		{
			// 8-channel source: exercises a channel count that the
			// old validation rejected, and confirms it lands on the
			// standard 7.1 (config 7) AAC layout.
			name: "aac-7_1-reencode",
			srcArgs: append(append([]string{}, commonVideo...),
				"-f", "lavfi", "-i",
				"aevalsrc=exprs=sin(440*2*PI*t):duration=2:sample_rate=48000:channel_layout=7.1",
				"-c:v", "libx264", "-preset", "ultrafast",
				"-c:a", "aac", "-b:a", "512k",
				"-t", "2",
			),
			srcExt:     "mkv",
			dst:        AudioEncodeParams{SourceStreamIndex: 0, Channels: 8, Bitrate: 512, SourceLayout: "7.1"},
			wantCodec:  "aac",
			wantCh:     8,
			wantLayout: "7.1",
		},
		{
			name: "flac-stereo-reencode",
			srcArgs: append(append([]string{}, commonVideo...),
				"-f", "lavfi", "-i",
				"sine=frequency=440:duration=2:sample_rate=48000",
				"-c:v", "libx264", "-preset", "ultrafast",
				"-c:a", "flac",
				"-t", "2",
			),
			srcExt:    "mkv",
			dst:       AudioEncodeParams{SourceStreamIndex: 0, Channels: 2, Bitrate: 128},
			wantCodec: "aac",
			wantCh:    2,
		},
		{
			name: "ac3-5_1-reencode",
			srcArgs: append(append([]string{}, commonVideo...),
				"-f", "lavfi", "-i",
				"aevalsrc=exprs=sin(440*2*PI*t):duration=2:sample_rate=48000:channel_layout=5.1",
				"-c:v", "libx264", "-preset", "ultrafast",
				"-c:a", "ac3", "-b:a", "384k",
				"-t", "2",
			),
			srcExt:     "mkv",
			dst:        AudioEncodeParams{SourceStreamIndex: 0, Channels: 6, Bitrate: 384},
			wantCodec:  "aac",
			wantCh:     6,
			wantLayout: "5.1",
		},
		{
			name: "mono-reencode",
			srcArgs: append(append([]string{}, commonVideo...),
				"-f", "lavfi", "-i",
				"sine=frequency=440:duration=2:sample_rate=48000",
				"-c:v", "libx264", "-preset", "ultrafast",
				"-c:a", "aac", "-b:a", "64k",
				"-t", "2",
			),
			srcExt:    "mkv",
			dst:       AudioEncodeParams{SourceStreamIndex: 0, Channels: 1, Bitrate: 64},
			wantCodec: "aac",
			wantCh:    1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runAudioCase(t, dir, c)
		})
	}
}

// TestEncodeAudioMissingStream verifies EncodeAudio surfaces an error
// when the requested SourceStreamIndex doesn't exist in the source.
// Requires Docker.
func TestEncodeAudioMissingStream(t *testing.T) {
	dir := setupDocker(t)
	ctx := t.Context()
	srcPath := filepath.Join(dir, "src-one-audio.mkv")

	t.Log("generating single-audio source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=2:size=160x90:rate=24",
		"-f", "lavfi", "-i",
		"sine=frequency=440:duration=2:sample_rate=48000",
		"-c:v", "libx264", "-preset", "ultrafast",
		"-c:a", "aac", "-ac", "2", "-b:a", "128k",
		"-t", "2",
		srcPath,
	)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe source: %v", err)
	}

	mediaPath := filepath.Join(dir, "missing-out.mp4")
	dst := AudioEncodeParams{
		Path:              mediaPath,
		SourceStreamIndex: 5, // nonexistent
		Channels:          2,
		Bitrate:           128,
	}
	_, err = EncodeAudio(ctx, srcFile, probe.FormatName, dst, probe.Duration, nil)
	if err == nil {
		t.Fatal("EncodeAudio with nonexistent stream returned nil error")
	}
	t.Logf("got expected error: %v", err)

	// Output should be empty (ffmpeg never wrote anything to it).
	if info, statErr := os.Stat(mediaPath); statErr == nil && info.Size() != 0 {
		t.Errorf("media file unexpectedly non-empty: size=%d", info.Size())
	}
}

// TestEncodeAudio51SideRemap specifically exercises the
// 5.1(side) → 5.1(back) channel-layout remap that lets CoreMedia /
// HLS clients accept the output. Requires Docker.
func TestEncodeAudio51SideRemap(t *testing.T) {
	dir := setupDocker(t)
	ctx := t.Context()
	srcPath := filepath.Join(dir, "src-5_1-side.mkv")

	t.Log("generating AC3 5.1(side) source...")
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=2:size=160x90:rate=24",
		"-f", "lavfi", "-i",
		"aevalsrc=exprs=sin(440*2*PI*t):duration=2:sample_rate=48000:channel_layout=5.1(side)",
		"-c:v", "libx264", "-preset", "ultrafast",
		"-c:a", "ac3", "-b:a", "384k",
		"-t", "2",
		srcPath,
	)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe source: %v", err)
	}
	t.Logf("source audio: codec=%s channels=%d layout=%s",
		probe.Audio[0].CodecName, probe.Audio[0].Channels,
		probe.Audio[0].ChannelLayout)
	if probe.Audio[0].ChannelLayout != "5.1(side)" {
		t.Fatalf("source layout %q is not 5.1(side); ffmpeg version doesn't preserve it, test goal evaporated",
			probe.Audio[0].ChannelLayout)
	}

	mediaPath := filepath.Join(dir, "remap-out.mp4")
	dst := AudioEncodeParams{
		Path:              mediaPath,
		SourceStreamIndex: 0,
		Channels:          6,
		Bitrate:           384,
	}
	if _, err := EncodeAudio(ctx, srcFile, probe.FormatName, dst, probe.Duration, nil); err != nil {
		t.Fatalf("EncodeAudio: %v", err)
	}

	outFp, err := os.Open(mediaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer outFp.Close()
	outProbe, err := Probe(ctx, outFp)
	if err != nil {
		t.Fatalf("probe output: %v", err)
	}
	if len(outProbe.Audio) == 0 {
		t.Fatal("output has no audio stream")
	}
	got := outProbe.Audio[0].ChannelLayout
	if got != "5.1" {
		t.Errorf("output ChannelLayout = %q, want %q (5.1(back))", got, "5.1")
	}
}
