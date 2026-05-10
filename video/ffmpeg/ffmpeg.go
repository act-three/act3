package ffmpeg

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"ily.dev/act3/xbufio"
	"kr.dev/errorfmt"
)

var overridePreset string

// OverridePreset sets a global preset override for all ffmpeg
// encoding operations (e.g. "ultrafast" for development).
func OverridePreset(preset string) {
	overridePreset = preset
}

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
	FormatName string // container format from ffprobe, e.g. "matroska,webm"
	Duration   time.Duration
	Video      *VideoStream     // first video stream, or nil
	Audio      []AudioStream    // all audio streams
	Subtitles  []SubtitleStream // text subtitle streams (bitmap formats are filtered out)
}

// ContentType returns the MIME type for the probed container format.
func (p *ProbeResult) ContentType() string {
	// ffprobe format_name is comma-separated; match on the first component.
	name, _, _ := strings.Cut(p.FormatName, ",")
	switch name {
	case "matroska":
		return "video/x-matroska"
	case "mov":
		return "video/mp4"
	case "avi":
		return "video/x-msvideo"
	case "mpegts":
		return "video/mp2t"
	case "flv":
		return "video/x-flv"
	case "ogg":
		return "video/ogg"
	case "webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

// FirstAudio returns the first audio stream, or nil if none.
func (p *ProbeResult) FirstAudio() *AudioStream {
	if len(p.Audio) == 0 {
		return nil
	}
	return &p.Audio[0]
}

// VideoStream describes a video stream.
type VideoStream struct {
	CodecName string    // e.g. "h264", "hevc", "vp9"
	BitRate   int64     // bits per second (0 if unknown)
	Width     int       //
	Height    int       //
	FrameRate FrameRate // exact fraction from r_frame_rate (display rate)

	// Stream timebase (seconds per tick) and exact stream duration in
	// those ticks. Together with PacketCount these give an exact
	// rational coded picture rate via CodedFrameRate, which is the
	// rate the encoder receives frames at under -fps_mode passthrough
	// — distinct from the display rate FrameRate carries on
	// soft-telecine and VFR sources.
	TimebaseNum   int   // e.g. 1 for "1/24000"
	TimebaseDen   int   // e.g. 24000
	DurationTicks int64 // exact stream duration, in TimebaseNum/TimebaseDen ticks

	// Display-order frame indices of every video keyframe and the
	// total video packet count, populated from a packet-level scan.
	// For closed-GOP encodings (which all standard HLS inputs use) a
	// keyframe's stream-order packet index equals its display-order
	// frame index, so these indices line up with the encoder's `n`
	// variable in `-force_key_frames "expr:..."`.
	//
	// HLS segment boundaries can only fall on keyframes (each fMP4
	// segment must be independently decodable), so Keyframes
	// determines achievable cut points for any rendition produced
	// from this source — both stream-copy and re-encode.
	Keyframes   []int64
	PacketCount int64
}

// CodedFrameRate returns the source's coded picture rate as the exact
// rational PacketCount / (DurationTicks × Timebase). For CFR sources
// this equals FrameRate; for soft-telecine and VFR it's the rate at
// which the encoder receives frames under -fps_mode passthrough.
// Returns the zero FrameRate if any input is missing.
func (v *VideoStream) CodedFrameRate() FrameRate {
	if v == nil || v.PacketCount <= 0 || v.DurationTicks <= 0 ||
		v.TimebaseNum <= 0 || v.TimebaseDen <= 0 {
		return FrameRate{}
	}
	num := v.PacketCount * int64(v.TimebaseDen)
	den := v.DurationTicks * int64(v.TimebaseNum)
	g := gcdInt64(num, den)
	return FrameRate{Num: int(num / g), Den: int(den / g)}
}

func gcdInt64(a, b int64) int64 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// AudioStream describes an audio stream.
type AudioStream struct {
	Index         int    // audio stream index (0-based among audio streams)
	CodecName     string // e.g. "aac", "ac3", "dts"
	Profile       string // codec profile, e.g. "LC", "HE-AAC" (empty if unknown)
	BitRate       int64  // bits per second (0 if unknown)
	SampleRate    int    // Hz (0 if unknown)
	Channels      int    // number of channels (e.g. 2, 6)
	ChannelLayout string // e.g. "stereo", "5.1(side)", "5.1" (empty if unknown)
	Language      string // e.g. "eng", "jpn" (empty if unknown)
	Title         string // e.g. "English", "Commentary" (empty if unknown)
}

// SubtitleStream describes a text-based subtitle stream.
// Bitmap formats (PGS, VOBSUB, DVB) are filtered out before
// they reach this type — see Probe.
type SubtitleStream struct {
	Index     int    // 0-based index among all source subtitle streams (including bitmap ones we don't surface), suitable for `-map 0:s:N`
	CodecName string // e.g. "ass", "subrip", "webvtt", "mov_text"
	Language  string // e.g. "eng", "jpn" (empty if unknown)
	Title     string // e.g. "English (Forced)" (empty if unknown)
	Forced    bool   // ffprobe disposition.forced
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
	StatsID   string   // stable filename for pass-1 stats inside statsDir; required when !Remux, ignored otherwise

	// SegmentBoundaries is the list of HLS segment cut points shared
	// by every rendition of one source video, expressed as
	// display-order source frame indices. When set on a re-encode
	// rendition, the encoder is forced to emit keyframes at exactly
	// these frame indices (and is told not to insert other
	// keyframes), so the HLS muxer cuts segments at the same source
	// frames the stream-copy remux variant naturally cuts at.
	// Without this, libx264/libx265 use their own GOP cadence and
	// the resulting variants disagree on segment boundaries —
	// Chrome's MSE refuses to bridge the resulting buffer gap on ABR
	// switch. Ignored for stream-copy renditions, which inherit
	// source keyframes regardless.
	//
	// Frame indices (rather than times) are used so the forced
	// keyframe lands on an exact frame boundary and not approximate
	// to within a frame's duration, satisfying the HLS authoring
	// spec's "aligned to within 1 video frame" requirement.
	SegmentBoundaries []int64
}

// allowedDemuxers is the comma-separated list of libavformat demuxer
// short names we accept. It expands to the MKV/WebM, MP4/M4V, and AVI
// container families — all self-contained formats that don't open
// nested URLs. Used as both the -format_whitelist for stage-1 probing
// and the acceptance gate for the -f value forced into subsequent
// ffprobe/ffmpeg invocations, so demuxers like concat, HLS, and
// QuickTime reference movies never auto-detect on attacker bytes.
const allowedDemuxers = "matroska,webm,avi,mov,mp4,m4a,3gp,3g2,mj2,mpegts"

// subtitleTextCodecs lists the text-based subtitle codecs we surface
// from Probe. Bitmap formats (PGS, VOBSUB, DVB) are intentionally
// absent: rendering them would require OCR, which we don't do.
var subtitleTextCodecs = map[string]bool{
	"ass":      true,
	"ssa":      true,
	"subrip":   true,
	"webvtt":   true,
	"mov_text": true,
}

// Probe runs ffprobe and returns stream information for the media in r.
//
// It probes in two stages. Stage 1 reads from pipe:0 with
// -protocol_whitelist pipe and -format_whitelist allowedDemuxers, so
// ffprobe can't open any filesystem paths during demuxer auto-detection
// and can't accept a container not on our list. Stage 2 re-probes with
// seekable /dev/stdin so duration scanning works, but forces the
// demuxer via -f using the format learned in stage 1 — so demuxer
// auto-detection never runs on attacker-controlled bytes.
func Probe(ctx context.Context, r *os.File) (*ProbeResult, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	format, err := probeFormat(ctx, r)
	if err != nil {
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	c := newCmd(ctx, "ffprobe",
		"-v", "quiet",
		"-protocol_whitelist", "file",
		"-f", format,
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"/dev/stdin",
	)
	c.Stdin = r
	c.Stdout = &stdout
	c.Stderr = &stderr
	err = c.Run()
	if err != nil {
		return nil, errors.Join(err, errors.New(stderr.String()))
	}

	var raw struct {
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
			BitRate    string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			CodecType     string `json:"codec_type"`
			CodecName     string `json:"codec_name"`
			Profile       string `json:"profile"`
			Width         int    `json:"width"`
			Height        int    `json:"height"`
			BitRate       string `json:"bit_rate"`
			SampleRate    string `json:"sample_rate"`
			RFrameRate    string `json:"r_frame_rate"`
			TimeBase      string `json:"time_base"`
			DurationTs    int64  `json:"duration_ts"`
			Channels      int    `json:"channels"`
			ChannelLayout string `json:"channel_layout"`
			Tags          struct {
				Language string `json:"language"`
				Title    string `json:"title"`
			} `json:"tags"`
			Disposition struct {
				Forced int `json:"forced"`
			} `json:"disposition"`
		} `json:"streams"`
	}
	err = json.Unmarshal(stdout.Bytes(), &raw)
	if err != nil {
		return nil, err
	}

	durSec, err := strconv.ParseFloat(raw.Format.Duration, 64)
	if err != nil {
		return nil, fmt.Errorf("probe: parse duration %q: %w", raw.Format.Duration, err)
	}
	if durSec <= 0 {
		return nil, fmt.Errorf("probe: non-positive duration %v", durSec)
	}
	result := &ProbeResult{
		FormatName: raw.Format.FormatName,
		Duration:   time.Duration(durSec * float64(time.Second)),
	}

	audioIdx := 0
	subtitleIdx := 0
	for _, s := range raw.Streams {
		switch s.CodecType {
		case "video":
			if result.Video == nil {
				br, _ := strconv.ParseInt(s.BitRate, 10, 64)
				tb := parseFrameRate(s.TimeBase)
				result.Video = &VideoStream{
					CodecName:     s.CodecName,
					BitRate:       br,
					Width:         s.Width,
					Height:        s.Height,
					FrameRate:     parseFrameRate(s.RFrameRate),
					TimebaseNum:   tb.Num,
					TimebaseDen:   tb.Den,
					DurationTicks: s.DurationTs,
				}
			}
		case "audio":
			br, _ := strconv.ParseInt(s.BitRate, 10, 64)
			sr, _ := strconv.Atoi(s.SampleRate)
			result.Audio = append(result.Audio, AudioStream{
				Index:         audioIdx,
				CodecName:     s.CodecName,
				Profile:       s.Profile,
				BitRate:       br,
				SampleRate:    sr,
				Channels:      s.Channels,
				ChannelLayout: s.ChannelLayout,
				Language:      s.Tags.Language,
				Title:         s.Tags.Title,
			})
			audioIdx++
		case "subtitle":
			// Increment unconditionally: ffmpeg's -map 0:s:N
			// counts every subtitle stream in the source
			// container regardless of codec, so the index we
			// hand back must reflect the source-side position
			// even when we filter the entry out.
			idx := subtitleIdx
			subtitleIdx++
			if !subtitleTextCodecs[s.CodecName] {
				continue
			}
			result.Subtitles = append(result.Subtitles, SubtitleStream{
				Index:     idx,
				CodecName: s.CodecName,
				Language:  s.Tags.Language,
				Title:     s.Tags.Title,
				Forced:    s.Disposition.Forced != 0,
			})
		}
	}

	// Estimate video bitrate from format-level bitrate when
	// per-stream data is unavailable (common with MKV).
	if result.Video != nil && result.Video.BitRate == 0 {
		fmtBr, _ := strconv.ParseInt(raw.Format.BitRate, 10, 64)
		audioBr := int64(0)
		if a := result.FirstAudio(); a != nil {
			audioBr = a.BitRate
		}
		if fmtBr > audioBr {
			result.Video.BitRate = fmtBr - audioBr
		}
	}

	if result.Video != nil {
		keyframes, packetCount, durationTicks, err := scanVideoPackets(ctx, r, format)
		if err != nil {
			return nil, err
		}
		result.Video.Keyframes = keyframes
		result.Video.PacketCount = packetCount
		// Stream-level duration_ts isn't always populated (MKV
		// doesn't carry per-track durations natively), so prefer
		// the value derived from actual packet timing.
		if durationTicks > 0 {
			result.Video.DurationTicks = durationTicks
		}
	}

	return result, nil
}

// scanVideoPackets enumerates video packets via ffprobe to derive
// keyframe positions (in display order, suitable for
// `-force_key_frames "expr:eq(n,...)"` since ffmpeg hands the
// encoder frames in display order under -fps_mode passthrough),
// the total packet count, and the stream's exact duration in
// timebase ticks (max(pts+duration) − min(pts), the wall-clock
// interval the encoder receives frames over).
//
// ffprobe walks packets in DTS order, which is the same as display
// order for closed-GOP sources but differs for open-GOP h264 (BD,
// some DVD remuxes): each new GOP's leading B-pictures decode
// after the IDR but display before it, so the IDR's display
// position sits a frame or two past its DTS index. Keyframe
// positions are translated to display order by sorting packets on
// PTS and emitting the post-sort indices.
func scanVideoPackets(ctx context.Context, r *os.File, format string) ([]int64, int64, int64, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, 0, 0, err
	}
	var stdout, stderr bytes.Buffer
	c := newCmd(ctx, "ffprobe",
		"-v", "error",
		"-protocol_whitelist", "file",
		"-f", format,
		"-select_streams", "v:0",
		"-show_packets",
		"-show_entries", "packet=pts,duration,flags",
		"-of", "csv=p=0",
		"/dev/stdin",
	)
	c.Stdin = r
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return nil, 0, 0, errors.Join(err, errors.New(stderr.String()))
	}
	keyframes, packetCount, durationTicks := parseVideoPackets(stdout.String())
	return keyframes, packetCount, durationTicks, nil
}

// pktInfo carries the per-packet fields we extract during
// scanVideoPackets parsing.
type pktInfo struct {
	pts   int64
	dur   int64
	isKey bool
	valid bool // pts and dur both parsed successfully
}

// parseVideoPackets parses ffprobe `-show_packets -show_entries
// packet=pts,duration,flags -of csv=p=0` output and returns the
// display-order keyframe positions, total packet count, and the
// stream's exact duration in timebase ticks (max(pts+dur) −
// min(pts)). Split out from scanVideoPackets for testability.
func parseVideoPackets(out string) (keyframes []int64, packetCount int64, durationTicks int64) {
	var pkts []pktInfo
	var minPTS, maxEnd int64
	var haveSpan bool
	allValid := true
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		// CSV fields, in -show_entries order: pts, duration, flags.
		fields := strings.Split(line, ",")
		var p pktInfo
		p.isKey = len(fields) >= 3 && strings.HasPrefix(fields[2], "K")
		if len(fields) >= 2 {
			pts, ptsErr := strconv.ParseInt(fields[0], 10, 64)
			dur, durErr := strconv.ParseInt(fields[1], 10, 64)
			if ptsErr == nil && durErr == nil {
				p.pts, p.dur, p.valid = pts, dur, true
				if !haveSpan || pts < minPTS {
					minPTS = pts
				}
				if end := pts + dur; !haveSpan || end > maxEnd {
					maxEnd = end
				}
				haveSpan = true
			}
		}
		if !p.valid {
			allValid = false
		}
		pkts = append(pkts, p)
	}

	// Sort to display order when every packet has a valid PTS;
	// fall back to DTS order otherwise so a stream with patchy
	// timestamps still produces a usable keyframe list.
	if allValid {
		slices.SortStableFunc(pkts, func(a, b pktInfo) int {
			return cmp.Compare(a.pts, b.pts)
		})
	}
	for i, p := range pkts {
		if p.isKey {
			keyframes = append(keyframes, int64(i))
		}
	}

	if haveSpan {
		durationTicks = maxEnd - minPTS
	}
	return keyframes, int64(len(pkts)), durationTicks
}

// probeFormat runs a constrained ffprobe to detect the container
// format. Input is piped as pipe:0 (non-seekable) under a
// -protocol_whitelist of pipe only, so ffprobe has no protocol-level
// way to touch the filesystem, and restricted via -format_whitelist to
// allowedDemuxers so auto-detection itself will fail for containers
// we don't accept.
//
// When the detected format is the libavformat mov/mp4 demuxer
// (shared between MP4 and QuickTime), the ftyp box's major_brand is
// additionally checked against an mp4-family allowlist so QuickTime
// reference movies — which carry a "qt  " major brand and can embed
// external-file references — are rejected at the probe stage.
//
// Returns the ffprobe format_name string, suitable for passing back
// as -f on subsequent invocations.
func probeFormat(ctx context.Context, r *os.File) (string, error) {
	var stdout, stderr bytes.Buffer
	c := newCmd(ctx, "ffprobe",
		"-v", "error",
		"-protocol_whitelist", "pipe",
		"-format_whitelist", allowedDemuxers,
		"-print_format", "json",
		"-show_format",
		"pipe:0",
	)
	c.Stdin = r
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return "", errors.Join(err, errors.New(stderr.String()))
	}
	var raw struct {
		Format struct {
			FormatName string `json:"format_name"`
			Tags       struct {
				MajorBrand       string `json:"major_brand"`
				CompatibleBrands string `json:"compatible_brands"`
			} `json:"tags"`
		} `json:"format"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return "", err
	}
	if raw.Format.FormatName == "" {
		return "", errors.New("probe: unrecognized container format")
	}
	if isISOBMFF(raw.Format.FormatName) {
		if err := checkMP4Brand(raw.Format.Tags.MajorBrand, raw.Format.Tags.CompatibleBrands); err != nil {
			return "", err
		}
	}
	return raw.Format.FormatName, nil
}

// allowedMP4Brands lists the ISO BMFF major brands we treat as
// mp4-family. Files whose ftyp major_brand falls outside this set —
// most importantly QuickTime's "qt  " — are rejected before ffmpeg
// ever sees them, closing the rmra/external-dref attack surface that
// the shared mov/mp4 demuxer would otherwise expose.
var allowedMP4Brands = map[string]bool{
	"isom": true, // ISO base media (generic MP4)
	"iso2": true,
	"iso4": true,
	"iso5": true,
	"iso6": true,
	"mp41": true, // MP4 v1
	"mp42": true, // MP4 v2 (what ffmpeg emits)
	"avc1": true, // H.264-branded MP4
	"M4V ": true, // iTunes video (trailing space is literal)
	"dash": true, // DASH-compatible MP4
}

// isISOBMFF reports whether formatName is the libavformat demuxer
// shared between MP4/M4V/3GP and QuickTime. The two variants cannot
// be told apart at the demuxer level but differ in the ftyp box's
// major_brand, which the caller gates on via checkMP4Brand.
func isISOBMFF(formatName string) bool {
	for _, tok := range strings.Split(formatName, ",") {
		if tok == "mov" || tok == "mp4" {
			return true
		}
	}
	return false
}

// checkMP4Brand validates an ISO BMFF file's ftyp brands. It returns
// an error if major_brand isn't on the mp4-family allowlist, or if
// compatible_brands advertises QuickTime — which catches files that
// try to sneak through by mislabeling their major brand.
func checkMP4Brand(majorBrand, compatibleBrands string) error {
	if !allowedMP4Brands[majorBrand] {
		return fmt.Errorf("probe: major_brand %q not on mp4 allowlist", majorBrand)
	}
	// compatible_brands is a concatenation of 4-byte brand codes
	// with no separator; the QuickTime brand is exactly "qt  ".
	for i := 0; i+4 <= len(compatibleBrands); i += 4 {
		if compatibleBrands[i:i+4] == "qt  " {
			return errors.New("probe: compatible_brands advertises QuickTime")
		}
	}
	return nil
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
// analysis for every reencode rendition in dsts (those with Remux
// false), reading the source once. For each such rendition, stats
// are stored inside statsDir under its StatsID, and the preset
// chosen for the batch is persisted alongside them so that
// [Pass2Single] and [Pass2ToMP4] can recover everything they need
// from statsDir and the rendition's EncodeParams alone.
//
// The caller must ensure statsDir exists, and must treat its contents
// as opaque: files inside belong to the ffmpeg package.
func Pass1Combined(ctx context.Context, src *os.File, format string,
	dsts []EncodeParams, statsDir string,
	duration time.Duration, onProgress func(float64),
) error {
	if i := slices.IndexFunc(dsts, func(p EncodeParams) bool {
		return !p.Remux && p.StatsID == ""
	}); i >= 0 {
		return fmt.Errorf("ffmpeg.Pass1Combined: dsts[%d] missing StatsID", i)
	}
	if !slices.ContainsFunc(dsts, func(p EncodeParams) bool { return !p.Remux }) {
		return nil
	}

	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	if err := pass1Combined(ctx, src, format, statsDir, dsts, report); err != nil {
		return err
	}

	preset := presetDefault
	if overridePreset != "" {
		preset = overridePreset
	}
	return os.WriteFile(filepath.Join(statsDir, pass1PresetFile), []byte(preset), 0o666)
}

// pass1PresetFile is the filename inside a pass-1 stats directory
// where [Pass1Combined] persists the preset for [Pass2Single] /
// [Pass2ToMP4] to read back.
const pass1PresetFile = "preset"

// Pass2Single runs second-pass encoding for a single rendition,
// producing one HLS fMP4 output. dst.StatsID and statsDir must refer
// to a rendition that was included in a prior [Pass1Combined] call.
// On success, Pass2Single removes its own per-rendition stats from
// statsDir; the caller is responsible for removing statsDir itself
// once every rendition for the batch has been processed.
func Pass2Single(ctx context.Context, src *os.File, format string, dst EncodeParams,
	statsDir string, duration time.Duration, onProgress func(float64),
) (playlist string, err error) {
	if dst.StatsID == "" {
		return "", fmt.Errorf("ffmpeg.Pass2Single: dst.StatsID is empty")
	}
	passlog := filepath.Join(statsDir, dst.StatsID)
	presetBytes, err := os.ReadFile(filepath.Join(statsDir, pass1PresetFile))
	if err != nil {
		return "", fmt.Errorf("read pass1 preset: %w", err)
	}
	preset := string(presetBytes)

	defer func() {
		if err == nil {
			removePass1StatsFor(passlog)
		}
	}()

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
	filterStr, labels := buildFilterComplex([]EncodeParams{dst})
	args := inputArgs(format)
	args = append(args, "-i", src.Name())
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	if overridePreset != "" {
		preset = overridePreset
	}

	args = append(args, "-map", labels[0])
	args = append(args, fpsPassthrough()...)
	args = append(args, "-c:v", dst.Codec, "-preset", preset)
	args = append(args, "-b:v", fmt.Sprintf("%dk", dst.Bitrate))
	if dst.Tag != "" {
		args = append(args, "-tag:v", dst.Tag)
	}
	forceArgs, codecExtras := keyframeArgs(dst.SegmentBoundaries)
	args = append(args, forceArgs...)
	args = append(args, twoPassArgs(dst.Codec, 2, passlog, codecExtras...)...)
	args = append(args, "-an", "-sn")
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
func RemuxSingle(ctx context.Context, src *os.File, format string, dst EncodeParams,
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

	if err := doRemux(ctx, src, format, tmpDir, dst, 0, report); err != nil {
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

// RemuxToMP4 produces a single downloadable MP4 by copying the video stream.
// Every source audio track is muxed in (stream-copied when AAC, otherwise
// re-encoded to AAC), with per-track language and title metadata preserved
// and the first track marked as the MP4 default audio track.
func RemuxToMP4(ctx context.Context, src *os.File, format string, dst EncodeParams,
	duration time.Duration, onProgress func(float64),
) error {
	probe, err := Probe(ctx, src)
	if err != nil {
		return fmt.Errorf("probe source: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "ffmpeg-remux-mp4-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	outPath := filepath.Join(tmpDir, "output.mp4")
	args := inputArgs(format)
	args = append(args, "-i", src.Name(),
		"-map", "0:v:0",
		"-c:v", "copy",
	)
	if dst.Tag != "" {
		args = append(args, "-tag:v", dst.Tag)
	}
	args = append(args, audioMuxArgsForDownload(probe.Audio)...)
	args = append(args, "-sn", "-movflags", "+faststart", outPath)

	if err := runWithProgress(ctx, args, report); err != nil {
		return err
	}
	return copyFileData(dst.File, outPath)
}

// Pass2ToMP4 runs second-pass encoding for a single rendition,
// producing a downloadable MP4 with faststart. dst.StatsID and
// statsDir must refer to a rendition that was included in a prior
// [Pass1Combined] call. On success, Pass2ToMP4 removes its own
// per-rendition stats from statsDir. Every source audio track is
// muxed in (stream-copied when AAC, otherwise re-encoded to AAC),
// with per-track language and title metadata preserved and the
// first track marked as the MP4 default audio track.
func Pass2ToMP4(ctx context.Context, src *os.File, format string, dst EncodeParams,
	statsDir string, duration time.Duration, onProgress func(float64),
) (err error) {
	if dst.StatsID == "" {
		return fmt.Errorf("ffmpeg.Pass2ToMP4: dst.StatsID is empty")
	}
	probe, err := Probe(ctx, src)
	if err != nil {
		return fmt.Errorf("probe source: %w", err)
	}
	passlog := filepath.Join(statsDir, dst.StatsID)
	presetBytes, err := os.ReadFile(filepath.Join(statsDir, pass1PresetFile))
	if err != nil {
		return fmt.Errorf("read pass1 preset: %w", err)
	}
	preset := string(presetBytes)

	defer func() {
		if err == nil {
			removePass1StatsFor(passlog)
		}
	}()

	tmpDir, err := os.MkdirTemp("", "ffmpeg-pass2-mp4-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	outPath := filepath.Join(tmpDir, "output.mp4")
	filterStr, labels := buildFilterComplex([]EncodeParams{dst})
	args := inputArgs(format)
	args = append(args, "-i", src.Name())
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	if overridePreset != "" {
		preset = overridePreset
	}

	args = append(args, "-map", labels[0])
	args = append(args, fpsPassthrough()...)
	args = append(args, "-c:v", dst.Codec, "-preset", preset)
	args = append(args, "-b:v", fmt.Sprintf("%dk", dst.Bitrate))
	if dst.Tag != "" {
		args = append(args, "-tag:v", dst.Tag)
	}
	args = append(args, twoPassArgs(dst.Codec, 2, passlog)...)
	args = append(args, audioMuxArgsForDownload(probe.Audio)...)
	args = append(args, "-sn", "-movflags", "+faststart", outPath)

	if err := runWithProgress(ctx, args, report); err != nil {
		return err
	}
	return copyFileData(dst.File, outPath)
}

// audioMuxArgsForDownload returns the ffmpeg args needed to mux every
// source audio stream into a download MP4. Each stream is stream-copied
// when the source is AAC, otherwise re-encoded to AAC at a
// channel-count-scaled bitrate. Per-track language and title metadata
// from the source is propagated, and the first track gets the MP4
// default-audio disposition.
//
// Stream-copy is more permissive here than on the streaming path: HE-AAC
// and unusual sample rates carry through unchanged. The streaming-side
// guards (Profile == "LC", SampleRate ∈ {44100, 48000}) exist for HLS
// manifest CODECS-attribute correctness and CoreMedia compatibility,
// neither of which apply to downloadable MP4. The output's esds box
// describes whatever the source actually was, so VLC and QuickTime
// pick the right decoder.
func audioMuxArgsForDownload(audio []AudioStream) []string {
	// Be explicit about audio-less output rather than relying on
	// "no -map for audio" → "no audio" by omission.
	if len(audio) == 0 {
		return []string{"-an"}
	}
	var args []string
	for i, a := range audio {
		args = append(args, "-map", fmt.Sprintf("0:a:%d", a.Index))
		if a.CodecName == "aac" {
			args = append(args, fmt.Sprintf("-c:a:%d", i), "copy")
		} else {
			args = append(args,
				fmt.Sprintf("-c:a:%d", i), "aac",
				fmt.Sprintf("-b:a:%d", i), fmt.Sprintf("%dk", downloadAudioBitrate(a.Channels)),
			)
		}
		// "und" (undetermined) propagates as a literal label in some
		// players; treat it the same as missing.
		if a.Language != "" && a.Language != "und" {
			args = append(args,
				fmt.Sprintf("-metadata:s:a:%d", i), "language="+a.Language)
		}
		if a.Title != "" {
			// MP4's per-track name lives in the hdlr box, set via
			// handler_name; the title= key is silently dropped by
			// the mov muxer.
			args = append(args,
				fmt.Sprintf("-metadata:s:a:%d", i), "handler_name="+a.Title)
		}
		if i == 0 {
			args = append(args, fmt.Sprintf("-disposition:a:%d", i), "default")
		} else {
			args = append(args, fmt.Sprintf("-disposition:a:%d", i), "0")
		}
	}
	return args
}

// downloadAudioBitrate scales bitrate with channel count for re-encoded
// download audio: 64 kbit/s per channel. Falls back to 128 when the
// source channel count is missing or zero.
func downloadAudioBitrate(channels int) int {
	if channels <= 0 {
		return 128
	}
	return 64 * channels
}

// -----------------------------------------------------------------------
// Subtitle extraction
// -----------------------------------------------------------------------

// subtitleOriginalFormat maps a subtitle codec name (as reported by
// ffprobe) to the ffmpeg output format that wraps the codec as a
// standalone file. mov_text is intentionally absent: it is the
// 3GPP timed-text codec carried inside MP4, and has no standalone
// on-disk format that round-trips losslessly. Callers wanting
// mov_text on disk should use ExtractSubtitleWebVTT instead.
var subtitleOriginalFormat = map[string]string{
	"ass":    "ass",
	"ssa":    "ass", // ffmpeg writes both as ASS
	"subrip": "srt",
	"webvtt": "webvtt",
}

// ExtractSubtitleOriginal extracts subtitle stream streamIndex from
// src in its original codec, writing it to dst. format is the ffprobe
// format_name from a prior Probe; it is forced via -f to avoid
// demuxer auto-detection on attacker-controlled bytes. codec is the
// subtitle codec name from the same Probe; it selects the output
// container so we don't have to re-probe to know what we're writing.
func ExtractSubtitleOriginal(ctx context.Context, src *os.File, format string, streamIndex int, codec string, dst *os.File) (err error) {
	defer errorfmt.Handlef("extract subtitle original: %w", &err)

	outFormat, ok := subtitleOriginalFormat[codec]
	if !ok {
		return fmt.Errorf("no standalone output format for codec %q", codec)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	var stderr bytes.Buffer
	c := newCmd(ctx, "ffmpeg",
		"-v", "error",
		"-nostdin", "-hide_banner",
		"-protocol_whitelist", "file",
		"-f", format,
		"-i", "/dev/stdin",
		"-map", fmt.Sprintf("0:s:%d", streamIndex),
		"-c:s", "copy",
		"-f", outFormat,
		"pipe:1",
	)
	c.Stdin = src
	c.Stdout = dst
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return errors.Join(err, errors.New(stderr.String()))
	}
	return nil
}

// ExtractSubtitleWebVTT extracts subtitle stream streamIndex from
// src and converts to WebVTT, writing it to dst. format is the
// ffprobe format_name from a prior Probe.
func ExtractSubtitleWebVTT(ctx context.Context, src *os.File, format string, streamIndex int, dst *os.File) (err error) {
	defer errorfmt.Handlef("extract subtitle webvtt: %w", &err)

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	var stderr bytes.Buffer
	c := newCmd(ctx, "ffmpeg",
		"-v", "error",
		"-nostdin", "-hide_banner",
		"-protocol_whitelist", "file",
		"-f", format,
		"-i", "/dev/stdin",
		"-map", fmt.Sprintf("0:s:%d", streamIndex),
		"-c:s", "webvtt",
		"-f", "webvtt",
		"pipe:1",
	)
	c.Stdin = src
	c.Stdout = dst
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return errors.Join(err, errors.New(stderr.String()))
	}
	return nil
}

// -----------------------------------------------------------------------
// Internal: the three encoding phases
// -----------------------------------------------------------------------

// doRemux produces one HLS rendition by copying the video stream.
func doRemux(ctx context.Context, src *os.File, format, tmpDir string,
	p EncodeParams, idx int, onProgress func(time.Duration),
) error {
	mediaPath := filepath.Join(tmpDir, MediaName(idx))
	plsPath := filepath.Join(tmpDir, playlistName(idx))

	args := inputArgs(format)
	args = append(args, "-i", src.Name())
	args = append(args, "-map", "0:v:0")
	args = append(args, "-c:v", "copy")
	if p.Tag != "" {
		args = append(args, "-tag:v", p.Tag)
	}
	args = append(args, "-an", "-sn")
	args = append(args, hlsOutputArgs(mediaPath)...)
	args = append(args, plsPath)
	return runWithProgress(ctx, args, onProgress)
}

// pass1Combined runs a single ffmpeg command that performs first-pass
// analysis for every reencode rendition in dsts (those with Remux
// false), reading the source once. Different resolution/fps targets
// are handled via filter_complex+split.
func pass1Combined(ctx context.Context, src *os.File, format, statsDir string,
	dsts []EncodeParams, onProgress func(time.Duration),
) error {
	filterStr, labels := buildFilterComplex(dsts)

	args := inputArgs(format)
	args = append(args, "-i", src.Name())
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	preset := presetDefault
	if overridePreset != "" {
		preset = overridePreset
	}

	for i, p := range dsts {
		if p.Remux {
			continue
		}
		args = append(args, "-map", labels[i])
		args = append(args, fpsPassthrough()...)
		args = append(args, "-c:v", p.Codec, "-preset", preset)
		args = append(args, "-b:v", fmt.Sprintf("%dk", p.Bitrate))
		forceArgs, codecExtras := keyframeArgs(p.SegmentBoundaries)
		args = append(args, forceArgs...)
		args = append(args, twoPassArgs(p.Codec, 1, filepath.Join(statsDir, p.StatsID), codecExtras...)...)
		args = append(args, "-an", "-sn")
		args = append(args, "-f", "null", "/dev/null")
	}

	return runWithProgress(ctx, args, onProgress)
}

// -----------------------------------------------------------------------
// Internal: filter_complex construction
// -----------------------------------------------------------------------

// buildFilterComplex produces a filter_complex string that splits the
// input video into one branch per reencode rendition (those with Remux
// false), applying per-branch scale and fps filters as needed. It
// returns the filter string (empty if no filtering is required) and a
// map from rendition index in dsts to the label or stream specifier
// to use in -map. Remux entries are absent from the returned map.
func buildFilterComplex(dsts []EncodeParams) (string, map[int]string) {
	labels := make(map[int]string)

	// Check whether any reencode rendition needs video filtering.
	anyFilter := false
	for _, p := range dsts {
		if p.Remux {
			continue
		}
		if p.MaxHeight > 0 || p.MaxFPS > 0 {
			anyFilter = true
			break
		}
	}

	if !anyFilter {
		// Every rendition uses source resolution and frame rate.
		// No filter_complex needed; each output maps 0:v:0 directly.
		for i, p := range dsts {
			if p.Remux {
				continue
			}
			labels[i] = "0:v:0"
		}
		return "", labels
	}

	// At least one rendition needs filtering, so we route all
	// reencode renditions through a split. Branches that need no
	// filtering are identity pass-throughs (virtually free).
	var idxs []int
	for i, p := range dsts {
		if p.Remux {
			continue
		}
		idxs = append(idxs, i)
	}
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

// inputArgs returns the flags ffmpeg needs before -i <input>. It
// bundles progress-reporting wiring, the shared security flags, and
// the forced input demuxer.
//
// -protocol_whitelist file,pipe: restrict every protocol open (main
// input and any nested opens a demuxer might attempt) to real files
// and our progress pipe. -f format: skip demuxer auto-detection so
// attacker bytes can't coerce ffmpeg into running concat, HLS, or
// similar multi-URL demuxers. -hwaccel none: force software decoding
// so ffmpeg doesn't fail on codecs without a hardware decoder (e.g.
// AV1). -nostdin / -hide_banner: standard unattended-run hygiene.
func inputArgs(format string) []string {
	return []string{
		"-y", "-nostdin", "-hide_banner",
		"-hwaccel", "none",
		"-protocol_whitelist", "file,pipe",
		"-progress", "pipe:3", "-nostats",
		"-f", format,
	}
}

// MinSegmentDuration is the floor on HLS segment length. Both
// SegmentBoundaries (which greedily picks source keyframes at
// least this far apart) and the HLS muxer's -hls_time trigger key
// off this value, so they agree on where to cut.
//
// Apple's HLS authoring spec recommends ≤6s; we want to stay under
// that ceiling whenever the source allows. A 4s floor gives us
// room to land on closely-spaced source keyframes — content
// typically uses 2–5s GOPs — instead of skipping every other
// keyframe and overshooting to 8–10s. Sources whose GOPs exceed
// the upper bound (handled separately, see ACT-180) fall back to
// re-encoding the top tier so we can pick boundaries freely.
const MinSegmentDuration = 4 * time.Second

func hlsOutputArgs(mediaPath string) []string {
	return []string{
		"-f", "hls",
		"-hls_segment_type", "fmp4",
		"-hls_flags", "single_file",
		"-hls_playlist_type", "vod",
		"-hls_time", strconv.Itoa(int(MinSegmentDuration.Seconds())),
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

// removePass1StatsFor removes the pass-1 stats files for a single
// rendition: the base stats file plus the .mbtree / .cutree sidecars
// that x264 / x265 deposit next to it. Best effort: missing files are
// not errors, and failures are silently ignored since the caller's
// eventual statsDir RemoveAll will sweep anything left behind.
func removePass1StatsFor(passlog string) {
	os.Remove(passlog)
	os.Remove(passlog + ".mbtree")
	os.Remove(passlog + ".cutree")
}

// twoPassArgs returns encoder-specific flags for 2-pass encoding.
// Both libx264 and libx265 use their native parameter interface
// (-x264-params / -x265-params) to specify the stats file directly.
// This avoids ffmpeg's -passlogfile suffix-appending logic, which
// bases the suffix on a global output stream index that differs
// between pass 1 (video only) and pass 2 (video + audio).
//
// Any extras are appended to the colon-separated params string for
// libx264/libx265; for other codecs they are ignored. Each extra is
// expected to be a single "key=value" fragment.
//
// Panics if passlog contains ':', '\\', or '\n': those characters
// would be reinterpreted as additional options inside the
// colon-separated -x264-params / -x265-params lists. The passlog
// path is built from operator config ($A3STORAGE), so this
// represents misconfiguration, not user input.
func twoPassArgs(codec string, pass int, passlog string, extras ...string) []string {
	if strings.ContainsAny(passlog, ":\\\n") {
		panic(fmt.Sprintf("ffmpeg: passlog path %q contains forbidden character (:, \\, or \\n)", passlog))
	}
	var key, params string
	switch codec {
	case "libx265":
		key = "-x265-params"
		params = fmt.Sprintf("pass=%d:stats=%s:open-gop=0", pass, passlog)
	case "libx264":
		key = "-x264-params"
		params = fmt.Sprintf("pass=%d:stats=%s", pass, passlog)
	default:
		return []string{
			"-pass", strconv.Itoa(pass),
			"-passlogfile", passlog,
		}
	}
	for _, e := range extras {
		if e != "" {
			params += ":" + e
		}
	}
	return []string{key, params}
}

// keyframeArgs returns the encoder-level flags that pin segment
// boundaries to the supplied source frame indices. Empty if cuts
// is empty.
//
// -force_key_frames "expr:eq(n,N1)+eq(n,N2)+..." pins each keyframe
// to a specific encoder-input frame index — exact, no time-rounding
// slop. (The Boolean ORs in the expression evaluate to 0/1 ints; a
// non-zero sum forces a keyframe.) The codec-params extras
// (returned for splicing into twoPassArgs) raise the maximum GOP
// size far above any plausible cut spacing and disable scene-cut
// detection, so the only keyframes the encoder produces are the
// forced ones — which is what makes ffmpeg's HLS muxer cut segments
// at exactly the requested frames.
//
// Assumes the rendition is encoded at the source's frame rate (no
// fps filter): then encoder frame `n` equals source display frame
// `n`. fps-changed re-encodes would need to convert source frame
// indices to encoder frame indices first.
func keyframeArgs(cuts []int64) (forceArgs []string, extras []string) {
	if len(cuts) == 0 {
		return nil, nil
	}
	var sb strings.Builder
	sb.WriteString("expr:")
	for i, c := range cuts {
		if i > 0 {
			sb.WriteByte('+')
		}
		fmt.Fprintf(&sb, "eq(n,%d)", c)
	}
	return []string{"-force_key_frames", sb.String()},
		[]string{"keyint=99999", "scenecut=0"}
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
		readProgress(ctx, pr, onProgress)
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
func readProgress(ctx context.Context, r io.Reader, update func(time.Duration)) {
	scanner := bufio.NewScanner(r)
	// ffmpeg -progress lines are short key=value pairs (well under
	// 100 bytes). 4 KB is ample; anything bigger is a bug worth surfacing.
	scanner.Buffer(make([]byte, 1024), 4*1024)
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
	if err := scanner.Err(); err != nil {
		slog.WarnContext(ctx, "ffmpeg-progress-read", "err", err)
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

// presetDefault is the preset Pass1Combined records when no
// override is set. Pass2Single / Pass2ToMP4 read it back so both
// passes agree: x265 makes identical B/P frame decisions only when
// the preset matches, otherwise it errors with "Incomplete CU-tree
// stats file" or "slice=P but 2pass stats say B".
const presetDefault = "medium"
