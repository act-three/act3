package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"ily.dev/act3/xbufio"
)

var overridePreset = os.Getenv("A3FFMPEGVIDEOPRESET")

// newCmd creates an *exec.Cmd for the named tool (ffmpeg or ffprobe).
// Tests override this to run tools inside Docker.
var newCmd = exec.CommandContext

// FrameRate is a frame rate represented as the exact fraction Num/Den.
// A zero or negative Den is treated as "unknown" by comparison methods.
type FrameRate struct {
	Num int // e.g. 24000
	Den int // e.g. 1001
}

// Le reports whether f ≤ n.
func (f FrameRate) Le(n int) bool {
	if f.Den <= 0 {
		return true // unknown treated as ≤ anything
	}
	return f.Num <= n*f.Den
}

// Gt reports whether f > n.
func (f FrameRate) Gt(n int) bool { return !f.Le(n) }

// Positive reports whether f represents a known, positive frame rate.
func (f FrameRate) Positive() bool { return f.Num > 0 && f.Den > 0 }

// String returns the fraction as "Num/Den".
func (f FrameRate) String() string { return fmt.Sprintf("%d/%d", f.Num, f.Den) }

// ProbeResult holds information about a media file's streams.
type ProbeResult struct {
	Duration time.Duration
	Video    *VideoStream // first video stream, or nil
	Audio    *AudioStream // first audio stream, or nil
}

// VideoStream describes a video stream.
type VideoStream struct {
	CodecName string    // e.g. "h264", "hevc", "vp9"
	BitRate   int64     // bits per second (0 if unknown)
	Width     int       //
	Height    int       //
	FrameRate FrameRate // exact fraction from r_frame_rate
}

// AudioStream describes an audio stream.
type AudioStream struct {
	CodecName     string // e.g. "aac", "ac3", "dts"
	BitRate       int64  // bits per second (0 if unknown)
	Channels      int    // number of channels (e.g. 2, 6)
	ChannelLayout string // e.g. "stereo", "5.1(side)", "5.1" (empty if unknown)
}

// EncodeParams describes how to produce one rendition.
type EncodeParams struct {
	File          *os.File // output file; media data is written here
	Remux         bool     // true: copy video stream; false: reencode
	Codec         string   // ffmpeg encoder name, e.g. "libx264" or "libx265" (ignored if Remux)
	Bitrate       int64    // target video bitrate in kbit/s (ignored if Remux)
	MaxHeight     int      // max output height; 0 = source (ignored if Remux)
	MaxFPS        int      // max output fps; 0 = source (ignored if Remux)
	Tag           string   // video tag, e.g. "hvc1" for HEVC in fMP4
	CopyAudio     bool     // true: copy audio stream; false: reencode to AAC
	SurroundAudio bool     // true: encode audio as 5.1(back); false: stereo downmix
}

// Probe runs ffprobe and returns stream information for the media in r.
func Probe(ctx context.Context, r *os.File) (*ProbeResult, error) {
	var stdout, stderr bytes.Buffer
	c := newCmd(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"/dev/stdin",
	)
	c.Stdin = r
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		return nil, errors.Join(err, errors.New(stderr.String()))
	}

	var raw struct {
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			CodecType     string `json:"codec_type"`
			CodecName     string `json:"codec_name"`
			Width         int    `json:"width"`
			Height        int    `json:"height"`
			BitRate       string `json:"bit_rate"`
			RFrameRate    string `json:"r_frame_rate"`
			Channels      int    `json:"channels"`
			ChannelLayout string `json:"channel_layout"`
		} `json:"streams"`
	}
	err = json.Unmarshal(stdout.Bytes(), &raw)
	if err != nil {
		return nil, err
	}

	durSec, _ := strconv.ParseFloat(raw.Format.Duration, 64)
	result := &ProbeResult{
		Duration: time.Duration(durSec * float64(time.Second)),
	}

	for _, s := range raw.Streams {
		switch s.CodecType {
		case "video":
			if result.Video == nil {
				br, _ := strconv.ParseInt(s.BitRate, 10, 64)
				result.Video = &VideoStream{
					CodecName: s.CodecName,
					BitRate:   br,
					Width:     s.Width,
					Height:    s.Height,
					FrameRate: parseFrameRate(s.RFrameRate),
				}
			}
		case "audio":
			if result.Audio == nil {
				br, _ := strconv.ParseInt(s.BitRate, 10, 64)
				result.Audio = &AudioStream{
					CodecName:     s.CodecName,
					BitRate:       br,
					Channels:      s.Channels,
					ChannelLayout: s.ChannelLayout,
				}
			}
		}
	}

	// Estimate video bitrate from format-level bitrate when
	// per-stream data is unavailable (common with MKV).
	if result.Video != nil && result.Video.BitRate == 0 {
		fmtBr, _ := strconv.ParseInt(raw.Format.BitRate, 10, 64)
		audioBr := int64(0)
		if result.Audio != nil {
			audioBr = result.Audio.BitRate
		}
		if fmtBr > audioBr {
			result.Video.BitRate = fmtBr - audioBr
		}
	}

	return result, nil
}

// ProbeDuration is a convenience wrapper that returns only the duration.
func ProbeDuration(ctx context.Context, r *os.File) (time.Duration, error) {
	p, err := Probe(ctx, r)
	if err != nil {
		return 0, err
	}
	return p.Duration, nil
}

// MediaName returns the media file name for rendition i.
// The caller uses this to fix up playlist references after hashing.
func MediaName(i int) string {
	return fmt.Sprintf("media%d.mp4", i)
}

func playlistName(i int) string {
	return fmt.Sprintf("stream%d.m3u8", i)
}

func passlogPath(tmpDir string, i int) string {
	return filepath.Join(tmpDir, fmt.Sprintf("passlog%d", i))
}

// Pass1Combined runs a single ffmpeg command that performs first-pass
// analysis for every reencode rendition, reading the source once.
// Stats files are written to the paths in passlogs (keyed by
// rendition index, matching idxs). The caller must ensure the
// parent directories exist. Returns the preset that must be passed
// to [Pass2Single].
func Pass1Combined(ctx context.Context, src *os.File, dsts []EncodeParams,
	idxs []int, passlogs map[int]string, duration time.Duration,
	onProgress func(float64),
) (string, error) {
	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	if err := pass1Combined(ctx, src, dsts, idxs, passlogs, report); err != nil {
		return "", err
	}

	preset := pass1DefaultPreset
	if overridePreset != "" {
		preset = overridePreset
	}
	return preset, nil
}

// Pass2Single runs second-pass encoding for a single rendition,
// producing one HLS fMP4 output. The passlog path must point to
// stats written by a prior [Pass1Combined] call for the same
// rendition parameters.
func Pass2Single(ctx context.Context, src *os.File, dst EncodeParams,
	passlog string, preset string, duration time.Duration, onProgress func(float64),
) (playlist string, err error) {
	tmpDir, err := os.MkdirTemp("", "ffmpeg-pass2-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	mediaPath := filepath.Join(tmpDir, MediaName(0))
	plsPath := filepath.Join(tmpDir, playlistName(0))

	// Build single-rendition pass-2 args.
	filterStr, labels := buildFilterComplex([]EncodeParams{dst}, []int{0})
	args := progressArgs()
	args = append(args, "-i", src.Name())
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	if overridePreset != "" {
		preset = overridePreset
	}

	args = append(args, "-map", labels[0])
	args = append(args, fpsPassthrough()...)
	args = append(args, "-map", "0:a:0?")
	args = append(args, "-c:v", dst.Codec, "-preset", preset)
	args = append(args, "-b:v", fmt.Sprintf("%dk", dst.Bitrate))
	if dst.Tag != "" {
		args = append(args, "-tag:v", dst.Tag)
	}
	args = append(args, twoPassArgs(dst.Codec, 2, passlog)...)
	if dst.CopyAudio {
		args = append(args, "-c:a", "copy")
	} else if dst.SurroundAudio {
		args = append(args, "-c:a", "aac", "-ac", "6", "-channel_layout", "5.1")
	} else {
		args = append(args, "-c:a", "aac", "-ac", "2")
	}
	args = append(args, "-sn")
	args = append(args, hlsOutputArgs(mediaPath)...)
	args = append(args, plsPath)

	if err := runWithProgress(ctx, args, report); err != nil {
		return "", err
	}

	// Copy media to caller's file and read playlist.
	mediaData, err := os.ReadFile(mediaPath)
	if err != nil {
		return "", fmt.Errorf("read media: %w", err)
	}
	if _, err := dst.File.Write(mediaData); err != nil {
		return "", fmt.Errorf("write media: %w", err)
	}
	b, err := os.ReadFile(plsPath)
	if err != nil {
		return "", fmt.Errorf("read playlist: %w", err)
	}

	// Correct EXTINF durations. ffmpeg's HLS muxer computes them
	// from raw encoder packet timestamps which can be offset from
	// the actual fMP4 segment PTS spans by the B-frame encoder
	// delay (~117ms with medium preset on VFR telecine input).
	return fixEXTINF(string(b), mediaData), nil
}

// RemuxSingle produces one HLS rendition by copying the video stream.
func RemuxSingle(ctx context.Context, src *os.File, dst EncodeParams,
	duration time.Duration, onProgress func(float64),
) (playlist string, err error) {
	tmpDir, err := os.MkdirTemp("", "ffmpeg-remux-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	if err := doRemux(ctx, src, tmpDir, dst, 0, report); err != nil {
		return "", err
	}

	// Copy media to caller's file and read playlist.
	mediaPath := filepath.Join(tmpDir, MediaName(0))
	plsPath := filepath.Join(tmpDir, playlistName(0))
	if err := copyFileData(dst.File, mediaPath); err != nil {
		return "", fmt.Errorf("copy media: %w", err)
	}
	b, err := os.ReadFile(plsPath)
	if err != nil {
		return "", fmt.Errorf("read playlist: %w", err)
	}
	return string(b), nil
}

// -----------------------------------------------------------------------
// Internal: the three encoding phases
// -----------------------------------------------------------------------

// doRemux produces one HLS rendition by copying the video stream.
func doRemux(ctx context.Context, src *os.File, tmpDir string,
	p EncodeParams, idx int, onProgress func(time.Duration),
) error {
	mediaPath := filepath.Join(tmpDir, MediaName(idx))
	plsPath := filepath.Join(tmpDir, playlistName(idx))

	args := progressArgs()
	args = append(args, "-i", src.Name())
	args = append(args,
		"-map", "0:v:0",
		"-map", "0:a:0?", // optional audio
	)
	args = append(args, "-c:v", "copy")
	if p.Tag != "" {
		args = append(args, "-tag:v", p.Tag)
	}
	if p.CopyAudio {
		args = append(args, "-c:a", "copy")
	} else if p.SurroundAudio {
		args = append(args, "-c:a", "aac", "-ac", "6", "-channel_layout", "5.1")
	} else {
		args = append(args, "-c:a", "aac", "-ac", "2")
	}
	args = append(args, "-sn")
	args = append(args, hlsOutputArgs(mediaPath)...)
	args = append(args, plsPath)
	return runWithProgress(ctx, args, onProgress)
}

// pass1Combined runs a single ffmpeg command that performs first-pass
// analysis for every reencode rendition, reading the source once.
// Different resolution/fps targets are handled via filter_complex+split.
func pass1Combined(ctx context.Context, src *os.File,
	dsts []EncodeParams, idxs []int, passlogs map[int]string,
	onProgress func(time.Duration),
) error {
	filterStr, labels := buildFilterComplex(dsts, idxs)

	args := progressArgs()
	args = append(args, "-i", src.Name())
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	preset := pass1DefaultPreset
	if overridePreset != "" {
		preset = overridePreset
	}

	for _, i := range idxs {
		p := dsts[i]
		args = append(args, "-map", labels[i])
		args = append(args, fpsPassthrough()...)
		args = append(args, "-c:v", p.Codec, "-preset", preset)
		args = append(args, "-b:v", fmt.Sprintf("%dk", p.Bitrate))
		args = append(args, twoPassArgs(p.Codec, 1, passlogs[i])...)
		args = append(args, "-an", "-sn")
		args = append(args, "-f", "null", "/dev/null")
	}

	return runWithProgress(ctx, args, onProgress)
}

// pass2Combined runs a single ffmpeg command that performs second-pass
// encoding for every reencode rendition, producing HLS fMP4 output
// for each. The source is read once and split via filter_complex when
// different resolutions or frame rates are needed.
func pass2Combined(ctx context.Context, src *os.File,
	dsts []EncodeParams, idxs []int, passlogs map[int]string,
	tmpDir string, onProgress func(time.Duration),
) error {
	filterStr, labels := buildFilterComplex(dsts, idxs)

	args := progressArgs()
	args = append(args, "-i", src.Name())
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	preset := pass2DefaultPreset
	if overridePreset != "" {
		preset = overridePreset
	}

	for _, i := range idxs {
		p := dsts[i]
		mediaPath := filepath.Join(tmpDir, MediaName(i))
		plsPath := filepath.Join(tmpDir, playlistName(i))

		args = append(args, "-map", labels[i])
		args = append(args, fpsPassthrough()...)
		args = append(args, "-map", "0:a:0?") // optional audio
		args = append(args, "-c:v", p.Codec, "-preset", preset)
		args = append(args, "-b:v", fmt.Sprintf("%dk", p.Bitrate))
		if p.Tag != "" {
			args = append(args, "-tag:v", p.Tag)
		}
		args = append(args, twoPassArgs(p.Codec, 2, passlogs[i])...)
		if p.CopyAudio {
			args = append(args, "-c:a", "copy")
		} else if p.SurroundAudio {
			args = append(args, "-c:a", "aac", "-ac", "6", "-channel_layout", "5.1")
		} else {
			args = append(args, "-c:a", "aac", "-ac", "2")
		}
		args = append(args, "-sn")
		args = append(args, hlsOutputArgs(mediaPath)...)
		args = append(args, plsPath)
	}

	return runWithProgress(ctx, args, onProgress)
}

// -----------------------------------------------------------------------
// Internal: filter_complex construction
// -----------------------------------------------------------------------

// buildFilterComplex produces a filter_complex string that splits the
// input video into one branch per rendition index, applying per-branch
// scale and fps filters as needed. It returns the filter string (empty
// if no filtering is required) and a map from rendition index to the
// label or stream specifier to use in -map.
func buildFilterComplex(dsts []EncodeParams, idxs []int) (string, map[int]string) {
	labels := make(map[int]string, len(idxs))

	// Check whether any rendition needs video filtering.
	anyFilter := false
	for _, i := range idxs {
		if dsts[i].MaxHeight > 0 || dsts[i].MaxFPS > 0 {
			anyFilter = true
			break
		}
	}

	if !anyFilter {
		// Every rendition uses source resolution and frame rate.
		// No filter_complex needed; each output maps 0:v:0 directly.
		for _, i := range idxs {
			labels[i] = "0:v:0"
		}
		return "", labels
	}

	// At least one rendition needs filtering, so we route all
	// reencode renditions through a split. Branches that need no
	// filtering are identity pass-throughs (virtually free).
	n := len(idxs)
	var parts []string

	// [0:v]split=N[s0][s1]...
	var split strings.Builder
	split.WriteString("[0:v]split=" + strconv.Itoa(n))
	for j := range n {
		fmt.Fprintf(&split, "[s%d]", j)
	}
	parts = append(parts, split.String())

	// Per-branch filters.
	for j, i := range idxs {
		in := fmt.Sprintf("[s%d]", j)
		out := fmt.Sprintf("[out%d]", j)

		var filters []string
		if dsts[i].MaxHeight > 0 {
			filters = append(filters,
				fmt.Sprintf("scale=-2:'min(%d,ih)'", dsts[i].MaxHeight))
		}
		if dsts[i].MaxFPS > 0 {
			filters = append(filters,
				fmt.Sprintf("fps=%d", dsts[i].MaxFPS))
		}

		if len(filters) > 0 {
			parts = append(parts, in+strings.Join(filters, ",")+out)
			labels[i] = out
		} else {
			// No filter needed; use the split output directly.
			labels[i] = in
		}
	}

	return strings.Join(parts, ";"), labels
}

// -----------------------------------------------------------------------
// Internal: ffmpeg argument helpers
// -----------------------------------------------------------------------

func progressArgs() []string {
	// -hwaccel none: force software decoding so ffmpeg doesn't fail
	// on codecs without hardware decoder support (e.g. AV1).
	return []string{"-y", "-hwaccel", "none", "-progress", "pipe:3", "-nostats"}
}

func hlsOutputArgs(mediaPath string) []string {
	return []string{
		"-f", "hls",
		"-hls_segment_type", "fmp4",
		"-hls_flags", "single_file",
		"-hls_playlist_type", "vod",
		"-hls_time", "6",
		"-hls_list_size", "0",
		"-hls_segment_filename", mediaPath,
	}
}

// fpsPassthrough returns args that prevent ffmpeg from duplicating
// or dropping video frames to match the container's advertised
// frame rate. Without this, MPEG-2 soft-telecine sources (where
// the MKV DefaultDuration says 59.94fps but the actual coded
// frames are ~24fps) cause pass 1 (-f null, no duplication) and
// pass 2 (-f hls, duplicates to 59.94fps) to produce different
// frame counts, which makes x265 fail with "Incomplete CU-tree
// stats file".
func fpsPassthrough() []string {
	return []string{"-fps_mode:v", "passthrough"}
}

// twoPassArgs returns encoder-specific flags for 2-pass encoding.
// Both libx264 and libx265 use their native parameter interface
// (-x264-params / -x265-params) to specify the stats file directly.
// This avoids ffmpeg's -passlogfile suffix-appending logic, which
// bases the suffix on a global output stream index that differs
// between pass 1 (video only) and pass 2 (video + audio).
func twoPassArgs(codec string, pass int, passlog string) []string {
	switch codec {
	case "libx265":
		return []string{
			"-x265-params",
			fmt.Sprintf("pass=%d:stats=%s:open-gop=0",
				pass, passlog),
		}
	case "libx264":
		return []string{
			"-x264-params",
			fmt.Sprintf("pass=%d:stats=%s", pass, passlog),
		}
	default:
		return []string{
			"-pass", strconv.Itoa(pass),
			"-passlogfile", passlog,
		}
	}
}

// -----------------------------------------------------------------------
// Internal: running ffmpeg with progress
// -----------------------------------------------------------------------

// runWithProgress starts ffmpeg with the given args, sets up a
// progress-reporting pipe on fd 3, and blocks until ffmpeg exits.
func runWithProgress(ctx context.Context, args []string, onProgress func(time.Duration)) error {
	stderr := &xbufio.BoundedWriter{Max: 100_000}
	c := newCmd(ctx, "ffmpeg", args...)
	c.Stderr = stderr

	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	c.ExtraFiles = []*os.File{pw}

	slog.DebugContext(ctx, "ffmpeg-command", "args", args)
	err = c.Start()
	pw.Close() // close parent's copy; child has its own fd
	if err != nil {
		pr.Close()
		return errors.Join(err, errors.New(stderr.String()))
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		readProgress(pr, onProgress)
		pr.Close()
	})

	err = c.Wait()
	wg.Wait()
	if err != nil {
		return errors.Join(err, errors.New(stderr.String()))
	}
	return nil
}

// readProgress reads ffmpeg -progress output from r and calls update
// with the current position in the output timeline.
func readProgress(r io.Reader, update func(time.Duration)) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		after, ok := strings.CutPrefix(line, "out_time_us=")
		if !ok {
			continue
		}
		us, err := strconv.ParseUint(after, 10, 64)
		if err != nil {
			continue
		}
		if update != nil {
			update(time.Microsecond * time.Duration(us))
		}
	}
}

// -----------------------------------------------------------------------
// Internal: misc helpers
// -----------------------------------------------------------------------

func parseFrameRate(s string) FrameRate {
	num, den, ok := strings.Cut(s, "/")
	if !ok {
		n, _ := strconv.Atoi(s)
		if n > 0 {
			return FrameRate{n, 1}
		}
		return FrameRate{}
	}
	n, _ := strconv.Atoi(num)
	d, _ := strconv.Atoi(den)
	return FrameRate{n, d}
}

// copyFileData copies the contents of the file at srcPath into dst.
func copyFileData(dst *os.File, srcPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(dst, f)
	return err
}

// totalWork returns the total encoding work for the given renditions,
// expressed in the same units as the source duration. This accounts
// for the three encoding phases (pass-1 + remux + pass-2) and can be
// used as the denominator for progress tracking.
func totalWork(dsts []EncodeParams, duration time.Duration) time.Duration {
	var total time.Duration
	nRemux := 0
	hasEncode := false
	for _, dst := range dsts {
		if dst.Remux {
			nRemux++
		} else {
			hasEncode = true
		}
	}
	total += time.Duration(nRemux) * duration // remux phases
	if hasEncode {
		total += 2 * duration // pass 1 + pass 2
	}
	return total
}

const (
	// Both passes must use the same preset so that x265 makes
	// identical frame-type decisions (B vs P). A preset mismatch
	// causes "Incomplete CU-tree stats file" / "slice=P but
	// 2pass stats say B" errors because the stats written in
	// pass 1 no longer match the frame structure in pass 2.
	pass1DefaultPreset = "medium"
	pass2DefaultPreset = "medium"
)
