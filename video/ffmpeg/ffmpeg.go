package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

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
	CodecName string // e.g. "aac", "ac3", "dts"
	BitRate   int64  // bits per second (0 if unknown)
}

// EncodeParams describes how to produce one rendition.
type EncodeParams struct {
	File      *os.File // output file; media data is written here
	Remux     bool     // true: copy video stream; false: reencode
	Codec     string   // ffmpeg encoder name, e.g. "libx264" or "libx265" (ignored if Remux)
	Bitrate   int64    // target video bitrate in kbit/s (ignored if Remux)
	MaxHeight int      // max output height; 0 = source (ignored if Remux)
	MaxFPS    int      // max output fps; 0 = source (ignored if Remux)
	Tag       string   // video tag, e.g. "hvc1" for HEVC in fMP4
	CopyAudio bool     // true: copy audio stream; false: reencode to AAC
}

// Probe runs ffprobe and returns stream information for the media in r.
func Probe(ctx context.Context, r io.Reader) (*ProbeResult, error) {
	var stdout, stderr bytes.Buffer
	c := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"pipe:0",
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
			CodecType  string `json:"codec_type"`
			CodecName  string `json:"codec_name"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			BitRate    string `json:"bit_rate"`
			RFrameRate string `json:"r_frame_rate"`
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
					CodecName: s.CodecName,
					BitRate:   br,
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
func ProbeDuration(ctx context.Context, r io.Reader) (time.Duration, error) {
	p, err := Probe(ctx, r)
	if err != nil {
		return 0, err
	}
	return p.Duration, nil
}

// TotalWork returns the total encoding work for the given renditions,
// expressed in the same units as the source duration. This accounts
// for the three encoding phases (pass-1 + remux + pass-2) and can be
// used as the denominator for progress tracking.
func TotalWork(dsts []EncodeParams, duration time.Duration) time.Duration {
	var total time.Duration
	hasRemux, hasEncode := false, false
	for _, dst := range dsts {
		if dst.Remux {
			hasRemux = true
		} else {
			hasEncode = true
		}
	}
	if hasRemux {
		total += duration // remux phase
	}
	if hasEncode {
		total += 2 * duration // pass 1 + pass 2
	}
	return total
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

// Encode produces one HLS fMP4 (single-file, byte-range) rendition per
// entry in dsts. For each rendition the fMP4 media data is written to
// [EncodeParams.File] and the HLS media playlist content is returned.
//
// Internally, encoding runs in up to three phases to minimise source reads:
//
//  1. Combined 2-pass analysis for all reencode renditions (single ffmpeg command).
//  2. Remux for the copy rendition (single ffmpeg command).
//  3. Combined 2-pass encode for all reencode renditions (single ffmpeg command).
//
// Each media playlist references its media file by a temporary basename
// (see [MediaName]). The caller must replace this with the content-addressed
// storage path.
//
// onProgress is called with cumulative time-based progress; the total can
// be computed with [TotalWork].
func Encode(ctx context.Context,
	src *os.File,
	dsts []EncodeParams,
	duration time.Duration,
	onProgress func(time.Duration),
) (playlists []string, err error) {
	if len(dsts) == 0 {
		return nil, fmt.Errorf("no renditions specified")
	}

	tmpDir, err := os.MkdirTemp("", "ffmpeg-encode-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Classify renditions into remux and reencode.
	remuxIdx := -1
	var encodeIdxs []int
	for i, dst := range dsts {
		if dst.Remux {
			if remuxIdx >= 0 {
				return nil, fmt.Errorf("multiple remux renditions not supported")
			}
			remuxIdx = i
		} else {
			encodeIdxs = append(encodeIdxs, i)
		}
	}

	playlists = make([]string, len(dsts))
	var cumulative time.Duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(cumulative + d)
		}
	}

	// Phase 1: combined first-pass analysis for all reencode renditions.
	if len(encodeIdxs) > 0 {
		if err = pass1Combined(ctx, src, tmpDir, dsts, encodeIdxs, report); err != nil {
			return nil, fmt.Errorf("pass 1: %w", err)
		}
		cumulative += duration
		if _, err = src.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}

	// Phase 2: remux the copy rendition.
	if remuxIdx >= 0 {
		if err = doRemux(ctx, src, tmpDir, dsts[remuxIdx], remuxIdx, report); err != nil {
			return nil, fmt.Errorf("remux: %w", err)
		}
		cumulative += duration
		if _, err = src.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}

	// Phase 3: combined second-pass encode for all reencode renditions.
	if len(encodeIdxs) > 0 {
		if err = pass2Combined(ctx, src, tmpDir, dsts, encodeIdxs, report); err != nil {
			return nil, fmt.Errorf("pass 2: %w", err)
		}
		// (no need to seek; we're done reading src)
	}

	// Copy media files to the caller's output files and read playlists.
	for i := range dsts {
		mediaPath := filepath.Join(tmpDir, MediaName(i))
		if err = copyFileData(dsts[i].File, mediaPath); err != nil {
			return nil, fmt.Errorf("rendition %d: copy media: %w", i, err)
		}

		plsPath := filepath.Join(tmpDir, playlistName(i))
		b, err := os.ReadFile(plsPath)
		if err != nil {
			return nil, fmt.Errorf("rendition %d: read playlist: %w", i, err)
		}
		playlists[i] = string(b)
	}

	return playlists, nil
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
	} else {
		args = append(args, "-c:a", "aac")
	}
	args = append(args, "-sn")
	args = append(args, hlsOutputArgs(mediaPath)...)
	args = append(args, plsPath)
	return runWithProgress(ctx, args, onProgress)
}

// pass1Combined runs a single ffmpeg command that performs first-pass
// analysis for every reencode rendition, reading the source once.
// Different resolution/fps targets are handled via filter_complex+split.
func pass1Combined(ctx context.Context, src *os.File, tmpDir string,
	dsts []EncodeParams, idxs []int, onProgress func(time.Duration),
) error {
	filterStr, labels := buildFilterComplex(dsts, idxs)

	args := progressArgs()
	args = append(args, "-i", src.Name())
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	for _, i := range idxs {
		p := dsts[i]
		args = append(args, "-map", labels[i])
		args = append(args, "-c:v", p.Codec, "-preset", "medium")
		args = append(args, "-b:v", fmt.Sprintf("%dk", p.Bitrate))
		args = append(args, twoPassArgs(p.Codec, 1, passlogPath(tmpDir, i))...)
		args = append(args, "-an", "-sn")
		args = append(args, "-f", "null", "/dev/null")
	}

	return runWithProgress(ctx, args, onProgress)
}

// pass2Combined runs a single ffmpeg command that performs second-pass
// encoding for every reencode rendition, producing HLS fMP4 output
// for each. The source is read once and split via filter_complex when
// different resolutions or frame rates are needed.
func pass2Combined(ctx context.Context, src *os.File, tmpDir string,
	dsts []EncodeParams, idxs []int, onProgress func(time.Duration),
) error {
	filterStr, labels := buildFilterComplex(dsts, idxs)

	args := progressArgs()
	args = append(args, "-i", src.Name())
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	for _, i := range idxs {
		p := dsts[i]
		mediaPath := filepath.Join(tmpDir, MediaName(i))
		plsPath := filepath.Join(tmpDir, playlistName(i))

		args = append(args, "-map", labels[i])
		args = append(args, "-map", "0:a:0?") // optional audio
		args = append(args, "-c:v", p.Codec, "-preset", "medium")
		args = append(args, "-b:v", fmt.Sprintf("%dk", p.Bitrate))
		if p.Tag != "" {
			args = append(args, "-tag:v", p.Tag)
		}
		args = append(args, twoPassArgs(p.Codec, 2, passlogPath(tmpDir, i))...)
		if p.CopyAudio {
			args = append(args, "-c:a", "copy")
		} else {
			args = append(args, "-c:a", "aac")
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
	split := "[0:v]split=" + strconv.Itoa(n)
	for j := range n {
		split += fmt.Sprintf("[s%d]", j)
	}
	parts = append(parts, split)

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
	return []string{"-y", "-progress", "pipe:3", "-nostats"}
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

// twoPassArgs returns encoder-specific flags for 2-pass encoding.
// libx265 uses -x265-params; libx264 uses standard -pass flags.
func twoPassArgs(codec string, pass int, passlog string) []string {
	if codec == "libx265" {
		return []string{
			"-x265-params",
			fmt.Sprintf("pass=%d:stats=%s", pass, passlog),
		}
	}
	return []string{
		"-pass", strconv.Itoa(pass),
		"-passlogfile", passlog,
	}
}

// -----------------------------------------------------------------------
// Internal: running ffmpeg with progress
// -----------------------------------------------------------------------

// runWithProgress starts ffmpeg with the given args, sets up a
// progress-reporting pipe on fd 3, and blocks until ffmpeg exits.
func runWithProgress(ctx context.Context, args []string, onProgress func(time.Duration)) error {
	stderr := &bytes.Buffer{}
	c := exec.CommandContext(ctx, "ffmpeg", args...)
	c.Stderr = stderr

	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	c.ExtraFiles = []*os.File{pw}

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
