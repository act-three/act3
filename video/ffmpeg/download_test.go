package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// downloadAudio is the per-audio-stream subset of an ffprobe report
// used to assert per-track codec, language, title, and disposition
// after [RemuxToMP4] / [Pass2ToMP4] muxes a multi-track source.
type downloadAudio struct {
	CodecName   string
	Channels    int
	Language    string
	Title       string
	DefaultFlag bool
}

// probeDownload runs ffprobe on a finished MP4 and pulls back the
// fields we want to assert on per audio stream. We don't reuse [Probe]
// because it is hardened for untrusted input (whitelisted demuxers,
// no-protocols) and doesn't surface disposition.default or the per-track
// title tag in MP4 form.
func probeDownload(t *testing.T, ctx context.Context, path string) []downloadAudio {
	t.Helper()
	var stdout, stderr bytes.Buffer
	c := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "a",
		path,
	)
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		t.Fatalf("ffprobe %s: %v\n%s", path, errors.Join(err, errors.New(stderr.String())), stderr.String())
	}
	var raw struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			Channels  int    `json:"channels"`
			Tags      struct {
				Language    string `json:"language"`
				HandlerName string `json:"handler_name"`
			} `json:"tags"`
			Disposition struct {
				Default int `json:"default"`
			} `json:"disposition"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("parse ffprobe json: %v\n%s", err, stdout.String())
	}
	var out []downloadAudio
	for _, s := range raw.Streams {
		out = append(out, downloadAudio{
			CodecName:   s.CodecName,
			Channels:    s.Channels,
			Language:    s.Tags.Language,
			Title:       s.Tags.HandlerName,
			DefaultFlag: s.Disposition.Default != 0,
		})
	}
	return out
}

// hasFaststart returns true if the MP4 at path is faststart-formatted:
// the moov atom appears before the mdat atom in the file.
func hasFaststart(t *testing.T, path string) bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mp4: %v", err)
	}
	moov := bytes.Index(data, []byte("moov"))
	mdat := bytes.Index(data, []byte("mdat"))
	if moov < 0 {
		t.Fatalf("no moov atom found in %s", path)
	}
	if mdat < 0 {
		t.Fatalf("no mdat atom found in %s", path)
	}
	return moov < mdat
}

// generateTwoAudioMKV writes a two-second MKV at srcPath with two audio
// tracks: track 0 AAC stereo (English) and track 1 AC3 5.1 (commentary).
// Both carry language=eng plus distinct title tags.
func generateTwoAudioMKV(t *testing.T, srcPath string) {
	t.Helper()
	generate(t,
		"-y",
		"-f", "lavfi", "-i",
		"testsrc2=duration=2:size=160x90:rate=24",
		"-f", "lavfi", "-i",
		"sine=frequency=440:duration=2:sample_rate=48000",
		"-f", "lavfi", "-i",
		"aevalsrc=exprs=sin(880*2*PI*t):duration=2:sample_rate=48000:channel_layout=5.1",
		"-map", "0:v", "-map", "1:a", "-map", "2:a",
		"-c:v", "libx264", "-preset", "ultrafast",
		"-c:a:0", "aac", "-ac:a:0", "2", "-b:a:0", "128k",
		"-c:a:1", "ac3", "-b:a:1", "384k",
		"-metadata:s:a:0", "language=eng",
		"-metadata:s:a:0", "handler_name=English",
		"-metadata:s:a:1", "language=eng",
		"-metadata:s:a:1", "title=Director's Commentary",
		"-t", "2",
		srcPath,
	)
}

// assertTwoAudioOutput asserts the muxed download MP4 carries both
// source audio tracks with the expected per-track codec, channel count,
// language, title, and default-track flag.
func assertTwoAudioOutput(t *testing.T, ctx context.Context, mp4Path string) {
	t.Helper()
	got := probeDownload(t, ctx, mp4Path)
	if len(got) != 2 {
		t.Fatalf("output audio streams = %d, want 2: %+v", len(got), got)
	}
	t.Logf("output audio: %+v", got)

	// Stream 0: AAC stereo English, default flag set, stream-copied
	// (still aac in the output).
	if got[0].CodecName != "aac" {
		t.Errorf("stream 0 codec = %q, want %q", got[0].CodecName, "aac")
	}
	if got[0].Channels != 2 {
		t.Errorf("stream 0 channels = %d, want 2", got[0].Channels)
	}
	if got[0].Language != "eng" {
		t.Errorf("stream 0 language = %q, want %q", got[0].Language, "eng")
	}
	if got[0].Title != "English" {
		t.Errorf("stream 0 title = %q, want %q", got[0].Title, "English")
	}
	if !got[0].DefaultFlag {
		t.Error("stream 0 default flag not set")
	}

	// Stream 1: re-encoded AC3 5.1 -> AAC 5.1, commentary metadata,
	// default flag cleared.
	if got[1].CodecName != "aac" {
		t.Errorf("stream 1 codec = %q, want %q (re-encoded from ac3)", got[1].CodecName, "aac")
	}
	if got[1].Channels != 6 {
		t.Errorf("stream 1 channels = %d, want 6", got[1].Channels)
	}
	if got[1].Language != "eng" {
		t.Errorf("stream 1 language = %q, want %q", got[1].Language, "eng")
	}
	if got[1].Title != "Director's Commentary" {
		t.Errorf("stream 1 title = %q, want %q", got[1].Title, "Director's Commentary")
	}
	if got[1].DefaultFlag {
		t.Error("stream 1 default flag should be cleared")
	}
}

// TestRemuxToMP4_TwoAudioTracks verifies that the remux download path
// preserves every source audio track, propagates per-track language and
// title metadata, marks only the first track as default, and produces a
// faststart-formatted MP4. Requires host ffmpeg.
func TestRemuxToMP4_TwoAudioTracks(t *testing.T) {
	dir := setupAgent(t)
	ctx := t.Context()

	srcPath := filepath.Join(dir, "two-audio.mkv")
	t.Log("generating two-audio source...")
	generateTwoAudioMKV(t, srcPath)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if len(probe.Audio) != 2 {
		t.Fatalf("source audio streams = %d, want 2", len(probe.Audio))
	}

	mp4Path := filepath.Join(dir, "out.mp4")
	dst := EncodeParams{Path: mp4Path, Remux: true}

	t.Log("running RemuxToMP4...")
	if err := RemuxToMP4(ctx, srcFile, probe.FormatName, dst, probe.Duration, nil); err != nil {
		t.Fatalf("RemuxToMP4: %v", err)
	}

	assertTwoAudioOutput(t, ctx, mp4Path)
	if !hasFaststart(t, mp4Path) {
		t.Error("output MP4 is not faststart (moov atom not before mdat)")
	}
}

// TestPass2ToMP4_TwoAudioTracks verifies that the 2-pass download path
// preserves every source audio track, propagates per-track language and
// title metadata, marks only the first track as default, and produces a
// faststart-formatted MP4. Requires host ffmpeg.
func TestPass2ToMP4_TwoAudioTracks(t *testing.T) {
	dir := setupAgent(t)
	preset := "ultrafast"
	ctx := t.Context()

	srcPath := filepath.Join(dir, "two-audio.mkv")
	t.Log("generating two-audio source...")
	generateTwoAudioMKV(t, srcPath)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()

	probe, err := Probe(ctx, srcFile)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if len(probe.Audio) != 2 {
		t.Fatalf("source audio streams = %d, want 2", len(probe.Audio))
	}

	mp4Path := filepath.Join(dir, "out.mp4")
	dst := EncodeParams{
		Path:    mp4Path,
		Codec:   "libx264",
		Bitrate: 500,
		StatsID: "r0",
	}

	t.Log("running pass 1...")
	err = Pass1Combined(ctx, srcFile, probe.FormatName,
		[]EncodeParams{dst}, testBatch, preset, probe.Duration, nil)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	t.Log("running Pass2ToMP4...")
	if err := Pass2ToMP4(ctx, srcFile, probe.FormatName, dst, testBatch, preset, probe.Duration, nil); err != nil {
		t.Fatalf("Pass2ToMP4: %v", err)
	}

	assertTwoAudioOutput(t, ctx, mp4Path)
	if !hasFaststart(t, mp4Path) {
		t.Error("output MP4 is not faststart (moov atom not before mdat)")
	}
}

// TestPass2ToMP4_RejectsEmptyStatsID verifies the precondition guard
// at the top of Pass2ToMP4. Runs without ffmpeg (panics before any
// ffmpeg invocation).
func TestPass2ToMP4_RejectsEmptyStatsID(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for empty StatsID, got none")
		}
	}()
	Pass2ToMP4(t.Context(), nil, "", EncodeParams{StatsID: ""}, "", "", 0, nil)
}

// TestAudioMuxArgsForDownload exercises the per-stream argument
// builder directly. Runs without ffmpeg.
func TestAudioMuxArgsForDownload(t *testing.T) {
	cases := []struct {
		name  string
		audio []AudioStream
		want  []string
	}{
		{
			name:  "empty audio yields explicit -an",
			audio: nil,
			want:  []string{"-an"},
		},
		{
			name: "language=und is dropped",
			audio: []AudioStream{
				{Index: 0, CodecName: "aac", Channels: 2, Language: "und"},
			},
			want: []string{
				"-map", "0:a:0",
				"-c:a:0", "copy",
				"-disposition:a:0", "default",
			},
		},
		{
			name: "single aac stereo with metadata",
			audio: []AudioStream{
				{Index: 0, CodecName: "aac", Channels: 2, Language: "eng", Title: "English"},
			},
			want: []string{
				"-map", "0:a:0",
				"-c:a:0", "copy",
				"-metadata:s:a:0", "language=eng",
				"-metadata:s:a:0", "handler_name=English",
				"-disposition:a:0", "default",
			},
		},
		{
			name: "two streams: aac stereo + ac3 5.1",
			audio: []AudioStream{
				{Index: 0, CodecName: "aac", Channels: 2, Language: "eng", Title: "English"},
				{Index: 1, CodecName: "ac3", Channels: 6, Language: "eng", Title: "Commentary"},
			},
			want: []string{
				"-map", "0:a:0",
				"-c:a:0", "copy",
				"-metadata:s:a:0", "language=eng",
				"-metadata:s:a:0", "handler_name=English",
				"-disposition:a:0", "default",
				"-map", "0:a:1",
				"-c:a:1", "aac",
				"-b:a:1", "384k",
				"-metadata:s:a:1", "language=eng",
				"-metadata:s:a:1", "handler_name=Commentary",
				"-disposition:a:1", "0",
			},
		},
		{
			name: "no-metadata stream is still mapped and re-encoded",
			audio: []AudioStream{
				{Index: 0, CodecName: "flac", Channels: 2},
			},
			want: []string{
				"-map", "0:a:0",
				"-c:a:0", "aac",
				"-b:a:0", "128k",
				"-disposition:a:0", "default",
			},
		},
		{
			name: "non-contiguous source indices are honoured",
			audio: []AudioStream{
				{Index: 2, CodecName: "aac", Channels: 2},
			},
			want: []string{
				"-map", "0:a:2",
				"-c:a:0", "copy",
				"-disposition:a:0", "default",
			},
		},
		{
			name: "zero channels falls back to 128k",
			audio: []AudioStream{
				{Index: 0, CodecName: "ac3", Channels: 0},
			},
			want: []string{
				"-map", "0:a:0",
				"-c:a:0", "aac",
				"-b:a:0", "128k",
				"-disposition:a:0", "default",
			},
		},
		{
			name: "slot tokens are dropped from tags",
			audio: []AudioStream{
				{Index: 0, CodecName: "aac", Channels: 2, Language: "e$OUTng", Title: "Dub $STATS/x"},
			},
			want: []string{
				"-map", "0:a:0",
				"-c:a:0", "copy",
				"-metadata:s:a:0", "language=eng",
				"-metadata:s:a:0", "handler_name=Dub /x",
				"-disposition:a:0", "default",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := audioMuxArgsForDownload(c.audio)
			if !equalStrings(got, c.want) {
				t.Errorf("audioMuxArgsForDownload mismatch\n got: %v\nwant: %v", got, c.want)
			}
		})
	}
}

func TestTagValue(t *testing.T) {
	cases := []struct{ in, want string }{
		{"English", "English"},
		{"$10 Movie Commentary", "$10 Movie Commentary"},
		{"$OUT", ""},
		{"a $STATS/rf01 b", "a /rf01 b"},
		// Removing one token must not splice the fragments
		// around it into another.
		{"$OU$STATST", ""},
		{"$$OUTSTATS", ""},
	}
	for _, c := range cases {
		if got := tagValue(c.in); got != c.want {
			t.Errorf("tagValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
