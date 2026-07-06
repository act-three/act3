package ffmpeg

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ily.dev/act3/video/fenc"
	"kr.dev/errorfmt"
)

// Per-encoder thread budgets. Encoder thread pools are capped so
// that concurrent encodes pack predictably onto large machines,
// rather than every x265/x264 instance sizing its pools for the
// whole machine and thrashing. EncoderThreads is the general
// budget: every pass-1 encoder, and pass 2 of the smaller tiers,
// which stop scaling well past 8 threads anyway. TopEncoderThreads
// goes to pass 2 of high-bitrate renditions, which dominate
// time-to-watchable and still scale usefully to 16.
const (
	EncoderThreads    = 8
	TopEncoderThreads = 16
)

// ThreadsFor returns the thread budget granted to dst's pass-2
// encoder: TopEncoderThreads for high-bitrate renditions (the top
// tier and best-rendition re-encodes), EncoderThreads otherwise.
// Bitrate stands in for output resolution, which EncodeParams
// cannot express (MaxHeight 0 means "source").
//
// Remux renditions run no encoder; ThreadsFor reports
// EncoderThreads for them, a deliberately conservative figure that
// doubles as a concurrency throttle when used as a scheduling
// weight, since concurrent stream copies contend on IO instead.
func ThreadsFor(dst EncodeParams) int {
	if !dst.Remux && dst.Bitrate >= 10_000 {
		return TopEncoderThreads
	}
	return EncoderThreads
}

// threadParams returns a codec-params fragment capping the encoder
// at threads. For x265, pools sizes the worker pool, and
// frame-threads is pinned to threads/4 (min 2) because its auto
// value is derived from the machine's core count, which oversizes
// frame parallelism once the pool is capped. For x264, threads=
// replaces its own machine-derived default (1.5x cores, capped at
// 16).
func threadParams(codec string, threads int) string {
	switch codec {
	case "libx265":
		return fmt.Sprintf("pools=%d:frame-threads=%d", threads, max(threads/4, 2))
	case "libx264":
		return fmt.Sprintf("threads=%d", threads)
	}
	return ""
}

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

	// HasExplicitDTS is true when the source's first packet carries
	// an explicit DTS value, false when ffprobe reports it as N/A.
	// MKV doesn't store DTS, so h264-in-MKV with B-frame reordering
	// has DTS unset on the leading packets — when the mp4 muxer is
	// asked to stream-copy such a source it synthesises DTS by
	// shifting the timeline forward and writing an empty edit-list
	// entry to compensate, which means the resulting fmp4 init can't
	// share an HLS timeline with re-encodes (which start at 0). The
	// planner uses this to refuse remux for those sources; see
	// PlanVideoRenditions.
	HasExplicitDTS bool

	// DolbyVisionProfile is the Dolby Vision profile carried by the
	// stream's DOVI configuration record, or 0 when the stream is not
	// Dolby Vision. Profile 5 stores its base layer in a reshaped IPT
	// color space that is unwatchable without applying the RPU
	// metadata, so the planner re-encodes such sources through a
	// Dolby-Vision-aware filter rather than passing them through; see
	// PlanVideoRenditions.
	DolbyVisionProfile int

	// ColorTransfer is the stream's transfer characteristics as
	// reported by ffprobe (e.g. "smpte2084", "arib-std-b67", "bt709"),
	// or empty when the source carries no color tags.
	// HDRFormat derives the output dynamic range from it.
	ColorTransfer string
}

// DolbyVisionNeedsConversion reports whether a stream carrying the
// given Dolby Vision profile must be converted to HDR10 before
// encoding. Profile 5 stores its base layer in a reshaped IPT color
// space that is unwatchable without applying the RPU; the
// HDR10-compatible profiles (8.x) carry a standard base layer and are
// left alone. A profile of 0 means the stream is not Dolby Vision.
func DolbyVisionNeedsConversion(profile int) bool {
	return profile == 5
}

// HDRFormat returns the dynamic range of the output the pipeline
// produces from a source with the given Dolby Vision profile and
// transfer characteristics, as an HLS VIDEO-RANGE label:
// "PQ" for a PQ (SMPTE 2084) source
// or a Dolby Vision source that is converted to HDR10,
// "HLG" for an HLG (ARIB STD-B67) source,
// and "" for SDR.
// Re-encoding preserves the source's dynamic range
// (color properties carry through the filtergraph into the output),
// so the label applies to every rendition, remuxed or re-encoded.
func HDRFormat(dolbyVisionProfile int, colorTransfer string) string {
	if DolbyVisionNeedsConversion(dolbyVisionProfile) {
		return "PQ" // dolbyVisionFilter emits BT.2020 PQ
	}
	switch colorTransfer {
	case "smpte2084":
		return "PQ"
	case "arib-std-b67":
		return "HLG"
	}
	return ""
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
	Path      string // output file path; the encoded media is collected here
	Remux     bool   // true: copy video stream; false: reencode
	Codec     string // ffmpeg encoder name, e.g. "libx264" or "libx265" (ignored if Remux)
	Bitrate   int64  // target video bitrate in kbit/s (ignored if Remux)
	MaxHeight int    // max output height; 0 = source (ignored if Remux)
	MaxFPS    int    // max output fps; 0 = source (ignored if Remux)
	Tag       string // video tag, e.g. "hvc1" for HEVC in fMP4
	StatsID   string // stable pass-1 stats name within the encode batch; required when !Remux, ignored otherwise

	// DolbyVision, when set on a re-encode rendition, applies the
	// source's Dolby Vision RPU and converts the reshaped base layer
	// to HDR10 (BT.2020 PQ) before scaling and encoding. This is
	// required for Profile 5 sources, whose base layer is otherwise
	// unwatchable (green/magenta cast). It requires an ffmpeg built
	// with libplacebo+libdovi and a Vulkan device; see
	// dolbyVisionFilter. Ignored for stream-copy renditions.
	DolbyVision bool

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
	// Frame indices (rather than times) are used so the cut points
	// can be expressed exactly in the source's timeline; keyframeArgs
	// converts each index N to a timestamp half a frame before
	// frame N's PTS, which makes ffmpeg's "first frame after time"
	// rule land the keyframe on frame N exactly — satisfying the
	// HLS authoring spec's "aligned to within 1 video frame"
	// requirement. The conversion needs SegmentBoundaryRate.
	SegmentBoundaries []int64

	// SegmentBoundaryRate is the frame rate against which
	// SegmentBoundaries indices are interpreted, i.e. the source's
	// coded picture rate (since -fps_mode passthrough hands the
	// encoder one frame per source coded picture). Required when
	// SegmentBoundaries is non-empty; ignored otherwise.
	SegmentBoundaryRate FrameRate
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
// It probes in two stages.
// Stage 1 reads the staged input from pipe:0
// with -protocol_whitelist pipe and -format_whitelist allowedDemuxers,
// so ffprobe can't open any filesystem paths
// during demuxer auto-detection
// and can't accept a container not on our list.
// Stage 2 re-probes through the fd protocol —
// the staged input bound to fd 0, seekable, so duration scanning works —
// and forces the demuxer via -f using the format learned in stage 1,
// so demuxer auto-detection never runs on attacker-controlled bytes.
// With -protocol_whitelist fd, no filesystem protocol is available at all:
// a file reference inside attacker bytes, relative or absolute,
// cannot be opened.
func Probe(ctx context.Context, r *os.File) (*ProbeResult, error) {
	j, err := newJob(r)
	if err != nil {
		return nil, err
	}
	defer j.close()
	format, err := probeFormat(ctx, j)
	if err != nil {
		return nil, err
	}
	err = j.run(ctx, fenc.JobRequest{
		Tool: "ffprobe",
		Args: []string{
			"-v", "quiet",
			"-protocol_whitelist", "fd",
			"-f", format,
			"-print_format", "json",
			"-show_format",
			"-show_streams",
			"fd:",
		},
		Stdout: "probe.json",
	}, nil)
	if err != nil {
		return nil, err
	}
	stdout, err := os.ReadFile(j.out("probe.json"))
	if err != nil {
		return nil, err
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
			ColorTransfer string `json:"color_transfer"`
			Tags          struct {
				Language string `json:"language"`
				Title    string `json:"title"`
			} `json:"tags"`
			Disposition struct {
				Forced int `json:"forced"`
			} `json:"disposition"`
			SideDataList []struct {
				SideDataType string `json:"side_data_type"`
				DVProfile    int    `json:"dv_profile"`
			} `json:"side_data_list"`
		} `json:"streams"`
	}
	err = json.Unmarshal(stdout, &raw)
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
				dvProfile := 0
				for _, sd := range s.SideDataList {
					if sd.SideDataType == "DOVI configuration record" {
						dvProfile = sd.DVProfile
						break
					}
				}
				result.Video = &VideoStream{
					CodecName:          s.CodecName,
					BitRate:            br,
					Width:              s.Width,
					Height:             s.Height,
					FrameRate:          parseFrameRate(s.RFrameRate),
					TimebaseNum:        tb.Num,
					TimebaseDen:        tb.Den,
					DurationTicks:      s.DurationTs,
					DolbyVisionProfile: dvProfile,
					ColorTransfer:      s.ColorTransfer,
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
		scan, err := scanVideoPackets(ctx, j, format)
		if err != nil {
			return nil, err
		}
		result.Video.Keyframes = scan.keyframes
		result.Video.PacketCount = scan.packetCount
		result.Video.HasExplicitDTS = scan.hasExplicitDTS
		// Stream-level duration_ts isn't always populated (MKV
		// doesn't carry per-track durations natively), so prefer
		// the value derived from actual packet timing.
		if scan.durationTicks > 0 {
			result.Video.DurationTicks = scan.durationTicks
		}
	}

	return result, nil
}

// videoPacketScan bundles the derived facts we extract from a
// packet-level ffprobe walk.
type videoPacketScan struct {
	keyframes      []int64 // display-order indices of K_-flagged packets
	packetCount    int64
	durationTicks  int64 // max(pts+dur) − min(pts) in stream-timebase ticks
	hasExplicitDTS bool  // first ffprobe-reported packet carries a DTS value
}

// scanVideoPackets enumerates video packets via ffprobe to derive
// keyframe positions (in display order, suitable for
// `-force_key_frames "expr:eq(n,...)"` since ffmpeg hands the
// encoder frames in display order under -fps_mode passthrough),
// the total packet count, the stream's exact duration in timebase
// ticks (max(pts+duration) − min(pts), the wall-clock interval the
// encoder receives frames over), and whether the first packet
// carries an explicit DTS (vs. N/A, which means the muxer would
// have to synthesize one — see VideoStream.HasExplicitDTS).
//
// ffprobe walks packets in DTS order, which is the same as display
// order for closed-GOP sources but differs for open-GOP h264 (BD,
// some DVD remuxes): each new GOP's leading B-pictures decode
// after the IDR but display before it, so the IDR's display
// position sits a frame or two past its DTS index. Keyframe
// positions are translated to display order by sorting packets on
// PTS and emitting the post-sort indices.
func scanVideoPackets(ctx context.Context, j *job, format string) (videoPacketScan, error) {
	err := j.run(ctx, fenc.JobRequest{
		Tool: "ffprobe",
		Args: []string{
			"-v", "error",
			"-protocol_whitelist", "fd",
			"-f", format,
			"-select_streams", "v:0",
			"-show_packets",
			"-show_entries", "packet=pts,dts,duration,flags",
			"-of", "csv=p=0",
			"fd:",
		},
		Stdout: "packets.csv",
	}, nil)
	if err != nil {
		return videoPacketScan{}, err
	}
	stdout, err := os.ReadFile(j.out("packets.csv"))
	if err != nil {
		return videoPacketScan{}, err
	}
	return parseVideoPackets(string(stdout)), nil
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
// packet=pts,dts,duration,flags -of csv=p=0` output. Split out from
// scanVideoPackets for testability.
func parseVideoPackets(out string) videoPacketScan {
	var pkts []pktInfo
	var scan videoPacketScan
	var minPTS, maxEnd int64
	var haveSpan bool
	allValid := true
	first := true
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		// CSV fields, in -show_entries order: pts, dts, duration, flags.
		fields := strings.Split(line, ",")
		if first {
			// ffprobe emits "N/A" (not an empty field) when DTS is
			// unset, so a parseable integer is the positive signal.
			if len(fields) >= 2 {
				if _, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					scan.hasExplicitDTS = true
				}
			}
			first = false
		}
		var p pktInfo
		p.isKey = len(fields) >= 4 && strings.HasPrefix(fields[3], "K")
		if len(fields) >= 3 {
			pts, ptsErr := strconv.ParseInt(fields[0], 10, 64)
			dur, durErr := strconv.ParseInt(fields[2], 10, 64)
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
			scan.keyframes = append(scan.keyframes, int64(i))
		}
	}

	scan.packetCount = int64(len(pkts))
	if haveSpan {
		scan.durationTicks = maxEnd - minPTS
	}
	return scan
}

// probeFormat runs a constrained ffprobe to detect the container
// format. The staged input is presented as pipe:0 (non-seekable)
// under a -protocol_whitelist of pipe only, so ffprobe has no
// protocol-level way to touch the filesystem, and restricted via
// -format_whitelist to allowedDemuxers so auto-detection itself will
// fail for containers we don't accept.
//
// When the detected format is the libavformat mov/mp4 demuxer
// (shared between MP4 and QuickTime), the ftyp box's major_brand is
// additionally checked against an mp4-family allowlist so QuickTime
// reference movies — which carry a "qt  " major brand and can embed
// external-file references — are rejected at the probe stage.
//
// Returns the ffprobe format_name string, suitable for passing back
// as -f on subsequent invocations.
func probeFormat(ctx context.Context, j *job) (string, error) {
	err := j.run(ctx, fenc.JobRequest{
		Tool: "ffprobe",
		Args: []string{
			"-v", "error",
			"-protocol_whitelist", "pipe",
			"-format_whitelist", allowedDemuxers,
			"-print_format", "json",
			"-show_format",
			"pipe:0",
		},
		Stdout: "format.json",
	}, nil)
	if err != nil {
		return "", err
	}
	stdout, err := os.ReadFile(j.out("format.json"))
	if err != nil {
		return "", err
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
	if err := json.Unmarshal(stdout, &raw); err != nil {
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
	for tok := range strings.SplitSeq(formatName, ",") {
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

// Pass1Combined runs a single ffmpeg command
// that performs first-pass analysis
// for every reencode rendition in dsts (those with Remux false),
// reading the source once.
// For each such rendition,
// stats are kept by the encoder agent
// under the given batch ID and the rendition's StatsID.
//
// preset selects the encoder preset for every reencode rendition.
// The caller passes the same preset to [Pass2Single] and [Pass2ToMP4],
// which may run in a later process:
// both passes must agree on it —
// x265 makes identical B/P frame decisions only when the presets match,
// and rejects the two-pass stats otherwise.
//
// The batch's stats belong to the agent;
// release them with [ReleaseStats]
// once every rendition has been encoded.
func Pass1Combined(ctx context.Context, src *os.File, format string,
	dsts []EncodeParams, batch, preset string,
	duration time.Duration, onProgress func(float64),
) error {
	if i := slices.IndexFunc(dsts, func(p EncodeParams) bool {
		return !p.Remux && p.StatsID == ""
	}); i >= 0 {
		panic(fmt.Sprintf("ffmpeg.Pass1Combined: dsts[%d] missing StatsID", i))
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

	return pass1Combined(ctx, src, format, batch, preset, dsts, report)
}

// Pass2Single runs second-pass encoding for a single rendition,
// producing one HLS fMP4 output.
// batch and dst.StatsID must refer to a rendition
// that was included in a prior [Pass1Combined] call,
// and preset must be the preset that call analyzed with.
func Pass2Single(ctx context.Context, src *os.File, format string, dst EncodeParams,
	batch, preset string, duration time.Duration, onProgress func(float64),
) (playlist string, err error) {
	if dst.StatsID == "" {
		panic("ffmpeg.Pass2Single: dst.StatsID is empty")
	}

	j, err := newJob(src)
	if err != nil {
		return "", err
	}
	defer j.close()

	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	// Build single-rendition pass-2 args.
	filterStr, labels := buildFilterComplex([]EncodeParams{dst})
	args := hwDeviceArgs(dst.DolbyVision)
	args = append(args, inputArgs(format)...)
	args = append(args, "-i", "fd:")
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	args = append(args, "-map", labels[0])
	args = append(args, fpsPassthrough()...)
	args = append(args, "-c:v", dst.Codec, "-preset", preset)
	args = append(args, "-b:v", fmt.Sprintf("%dk", dst.Bitrate))
	args = append(args, hdr10OutputArgs(dst.DolbyVision)...)
	if dst.Tag != "" {
		args = append(args, "-tag:v", dst.Tag)
	}
	forceArgs, codecExtras := keyframeArgs(dst.SegmentBoundaries, dst.SegmentBoundaryRate)
	args = append(args, forceArgs...)
	codecExtras = append(codecExtras, threadParams(dst.Codec, ThreadsFor(dst)))
	args = append(args, twoPassArgs(dst.Codec, 2, fenc.SlotStats+"/"+dst.StatsID, codecExtras...)...)
	args = append(args, "-an", "-sn")
	args = append(args, hlsOutputArgs(fenc.SlotOut+"/"+MediaName(0))...)
	args = append(args, fenc.SlotOut+"/"+playlistName(0))

	err = j.run(ctx, fenc.JobRequest{
		Tool:     "ffmpeg",
		Args:     args,
		Stats:    batch,
		Progress: true,
	}, report)
	if err != nil {
		return "", err
	}

	if err := collectInto(dst.Path, j.out(MediaName(0))); err != nil {
		return "", fmt.Errorf("collect media: %w", err)
	}
	b, err := os.ReadFile(j.out(playlistName(0)))
	if err != nil {
		return "", fmt.Errorf("read playlist: %w", err)
	}
	pls := string(b)

	// Correct EXTINF durations. ffmpeg's HLS muxer computes them
	// from raw encoder packet timestamps which can be offset from
	// the actual fMP4 segment PTS spans by the B-frame encoder
	// delay (~117ms with medium preset on VFR telecine input).
	// Map the media read-only rather than reading it into memory:
	// fixEXTINF touches only each segment's leading boxes (sidx and
	// moof), so the kernel pages in a few KB per segment instead of
	// the whole multi-GB rendition. The job's spool copy has the
	// same bytes as the collected media; map that one.
	media, err := os.Open(j.out(MediaName(0)))
	if err != nil {
		return "", fmt.Errorf("open media: %w", err)
	}
	defer media.Close()
	fi, err := media.Stat()
	if err != nil {
		return "", fmt.Errorf("stat media: %w", err)
	}
	if fi.Size() == 0 {
		return pls, nil // nothing to correct; mmap rejects length 0
	}
	mediaData, err := syscall.Mmap(int(media.Fd()), 0, int(fi.Size()),
		syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return "", fmt.Errorf("mmap media: %w", err)
	}
	defer syscall.Munmap(mediaData)
	return fixEXTINF(pls, mediaData), nil
}

// RemuxSingle produces one HLS rendition by copying the video stream.
func RemuxSingle(ctx context.Context, src *os.File, format string, dst EncodeParams,
	duration time.Duration, onProgress func(float64),
) (playlist string, err error) {
	j, err := newJob(src)
	if err != nil {
		return "", err
	}
	defer j.close()

	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	args := inputArgs(format)
	args = append(args, "-i", "fd:")
	args = append(args, "-map", "0:v:0")
	args = append(args, "-c:v", "copy")
	if dst.Tag != "" {
		args = append(args, "-tag:v", dst.Tag)
	}
	args = append(args, "-an", "-sn")
	args = append(args, hlsOutputArgs(fenc.SlotOut+"/"+MediaName(0))...)
	args = append(args, fenc.SlotOut+"/"+playlistName(0))

	err = j.run(ctx, fenc.JobRequest{
		Tool:     "ffmpeg",
		Args:     args,
		Progress: true,
	}, report)
	if err != nil {
		return "", err
	}

	if err := collectInto(dst.Path, j.out(MediaName(0))); err != nil {
		return "", fmt.Errorf("collect media: %w", err)
	}
	b, err := os.ReadFile(j.out(playlistName(0)))
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

	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	j, err := newJob(src)
	if err != nil {
		return err
	}
	defer j.close()

	args := inputArgs(format)
	args = append(args, "-i", "fd:",
		"-map", "0:v:0",
		"-c:v", "copy",
	)
	if dst.Tag != "" {
		args = append(args, "-tag:v", dst.Tag)
	}
	args = append(args, audioMuxArgsForDownload(probe.Audio)...)
	args = append(args, "-sn", "-movflags", "+faststart", fenc.SlotOut+"/"+MediaName(0))

	err = j.run(ctx, fenc.JobRequest{
		Tool:     "ffmpeg",
		Args:     args,
		Progress: true,
	}, report)
	if err != nil {
		return err
	}
	return collectInto(dst.Path, j.out(MediaName(0)))
}

// Pass2ToMP4 runs second-pass encoding for a single rendition,
// producing a downloadable MP4 with faststart.
// batch and dst.StatsID must refer to a rendition
// that was included in a prior [Pass1Combined] call,
// and preset must be the preset that call analyzed with.
// Every source audio track is muxed in
// (stream-copied when AAC, otherwise re-encoded to AAC),
// with per-track language and title metadata preserved
// and the first track marked as the MP4 default audio track.
func Pass2ToMP4(ctx context.Context, src *os.File, format string, dst EncodeParams,
	batch, preset string, duration time.Duration, onProgress func(float64),
) (err error) {
	if dst.StatsID == "" {
		panic("ffmpeg.Pass2ToMP4: dst.StatsID is empty")
	}
	probe, err := Probe(ctx, src)
	if err != nil {
		return fmt.Errorf("probe source: %w", err)
	}

	j, err := newJob(src)
	if err != nil {
		return err
	}
	defer j.close()

	total := duration
	report := func(d time.Duration) {
		if onProgress != nil {
			onProgress(float64(d) / float64(total))
		}
	}

	filterStr, labels := buildFilterComplex([]EncodeParams{dst})
	args := hwDeviceArgs(dst.DolbyVision)
	args = append(args, inputArgs(format)...)
	args = append(args, "-i", "fd:")
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	args = append(args, "-map", labels[0])
	args = append(args, fpsPassthrough()...)
	args = append(args, "-c:v", dst.Codec, "-preset", preset)
	args = append(args, "-b:v", fmt.Sprintf("%dk", dst.Bitrate))
	args = append(args, hdr10OutputArgs(dst.DolbyVision)...)
	if dst.Tag != "" {
		args = append(args, "-tag:v", dst.Tag)
	}
	args = append(args, twoPassArgs(dst.Codec, 2, fenc.SlotStats+"/"+dst.StatsID, threadParams(dst.Codec, ThreadsFor(dst)))...)
	args = append(args, audioMuxArgsForDownload(probe.Audio)...)
	args = append(args, "-sn", "-movflags", "+faststart", fenc.SlotOut+"/"+MediaName(0))

	err = j.run(ctx, fenc.JobRequest{
		Tool:     "ffmpeg",
		Args:     args,
		Stats:    batch,
		Progress: true,
	}, report)
	if err != nil {
		return err
	}
	return collectInto(dst.Path, j.out(MediaName(0)))
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
				fmt.Sprintf("-metadata:s:a:%d", i), "language="+tagValue(a.Language))
		}
		if a.Title != "" {
			// MP4's per-track name lives in the hdlr box, set via
			// handler_name; the title= key is silently dropped by
			// the mov muxer.
			args = append(args,
				fmt.Sprintf("-metadata:s:a:%d", i), "handler_name="+tagValue(a.Title))
		}
		if i == 0 {
			args = append(args, fmt.Sprintf("-disposition:a:%d", i), "default")
		} else {
			args = append(args, fmt.Sprintf("-disposition:a:%d", i), "0")
		}
	}
	return args
}

// tagValue makes a source-controlled tag value safe to place in
// agent argv.
// The agent substitutes slot tokens anywhere in any argument,
// so a tag carrying one would be rewritten to an agent-side path —
// or rejected outright, permanently failing the encode.
// The tokens never appear in honest metadata, so they are dropped.
// Removal repeats until no token remains:
// overlapping fragments must not reassemble into a new token.
func tagValue(s string) string {
	for {
		t := strings.ReplaceAll(s, fenc.SlotOut, "")
		t = strings.ReplaceAll(t, fenc.SlotStats, "")
		if t == s {
			return t
		}
		s = t
	}
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
	return extractSubtitle(ctx, src, format, streamIndex, "copy", outFormat, dst)
}

// ExtractSubtitleWebVTT extracts subtitle stream streamIndex from
// src and converts to WebVTT, writing it to dst. format is the
// ffprobe format_name from a prior Probe.
func ExtractSubtitleWebVTT(ctx context.Context, src *os.File, format string, streamIndex int, dst *os.File) (err error) {
	defer errorfmt.Handlef("extract subtitle webvtt: %w", &err)

	return extractSubtitle(ctx, src, format, streamIndex, "webvtt", "webvtt", dst)
}

// extractSubtitle runs one subtitle-extraction job: stream
// streamIndex of src, transcoded by subtitle codec csCodec into
// container outFormat, written to dst.
func extractSubtitle(ctx context.Context, src *os.File, format string,
	streamIndex int, csCodec, outFormat string, dst *os.File,
) error {
	j, err := newJob(src)
	if err != nil {
		return err
	}
	defer j.close()

	err = j.run(ctx, fenc.JobRequest{
		Tool: "ffmpeg",
		Args: []string{
			"-v", "error",
			"-nostdin", "-hide_banner",
			"-protocol_whitelist", "fd,pipe",
			"-f", format,
			"-i", "fd:",
			"-map", fmt.Sprintf("0:s:%d", streamIndex),
			"-c:s", csCodec,
			"-f", outFormat,
			"pipe:1",
		},
		Stdout: "sub",
	}, nil)
	if err != nil {
		return err
	}
	return collectFile(dst, j.out("sub"))
}

// -----------------------------------------------------------------------
// Internal: combined first-pass analysis
// -----------------------------------------------------------------------

// pass1Combined runs a single ffmpeg command that performs first-pass
// analysis for every reencode rendition in dsts (those with Remux
// false), reading the source once. Different resolution/fps targets
// are handled via filter_complex+split.
func pass1Combined(ctx context.Context, src *os.File, format, batch, preset string,
	dsts []EncodeParams, onProgress func(time.Duration),
) error {
	j, err := newJob(src)
	if err != nil {
		return err
	}
	defer j.close()

	filterStr, labels := buildFilterComplex(dsts)

	args := hwDeviceArgs(anyDolbyVision(dsts))
	args = append(args, inputArgs(format)...)
	args = append(args, "-i", "fd:")
	if filterStr != "" {
		args = append(args, "-filter_complex", filterStr)
	}

	for i, p := range dsts {
		if p.Remux {
			continue
		}
		args = append(args, "-map", labels[i])
		args = append(args, fpsPassthrough()...)
		args = append(args, "-c:v", p.Codec, "-preset", preset)
		args = append(args, "-b:v", fmt.Sprintf("%dk", p.Bitrate))
		forceArgs, codecExtras := keyframeArgs(p.SegmentBoundaries, p.SegmentBoundaryRate)
		args = append(args, forceArgs...)
		codecExtras = append(codecExtras, threadParams(p.Codec, EncoderThreads))
		args = append(args, twoPassArgs(p.Codec, 1, fenc.SlotStats+"/"+p.StatsID, codecExtras...)...)
		args = append(args, "-an", "-sn")
		args = append(args, "-f", "null", "/dev/null")
	}

	return j.run(ctx, fenc.JobRequest{
		Tool:     "ffmpeg",
		Args:     args,
		Stats:    batch,
		Progress: true,
	}, onProgress)
}

// -----------------------------------------------------------------------
// Internal: filter_complex construction
// -----------------------------------------------------------------------

// dolbyVisionFilter applies the source's Dolby Vision RPU and converts
// the reshaped base layer to HDR10 (BT.2020 PQ, 10-bit) via libplacebo.
// It uploads to and downloads from the Vulkan device that the caller
// must have created (see hwDeviceArgs). Without this conversion a
// Profile 5 base layer is misread as ordinary YCbCr, producing a
// green/magenta cast.
//
// libplacebo emits rgba64le (16-bit packed RGB, enough for 10-bit
// HDR), which is then converted to yuv420p10le in software for the
// encoder. The packed intermediate is deliberate: the appliance has no
// GPU, so libplacebo runs on a software Vulkan rasterizer, and reading
// a multiplanar YUV image back from such a device returns garbage
// chroma (an all-green frame). Single-plane packed formats download
// correctly, so the YUV conversion is done on the CPU after hwdownload.
const dolbyVisionFilter = "hwupload," +
	"libplacebo=apply_dolbyvision=true:colorspace=bt2020nc:color_primaries=bt2020:color_trc=smpte2084:format=rgba64le," +
	"hwdownload,format=rgba64le,format=yuv420p10le"

// buildFilterComplex produces a filter_complex string that splits the
// input video into one branch per reencode rendition (those with Remux
// false), applying per-branch scale and fps filters as needed. It
// returns the filter string (empty if no filtering is required) and a
// map from rendition index in dsts to the label or stream specifier
// to use in -map. Remux entries are absent from the returned map.
//
// Dolby Vision renditions force a filter graph even when no scaling is
// needed: a single dolbyVisionFilter pass converts the source to HDR10
// up front, and its output feeds the split (a libplacebo output pad can
// only be consumed once, so every branch draws from the split).
func buildFilterComplex(dsts []EncodeParams) (string, map[int]string) {
	labels := make(map[int]string)

	// Check whether any reencode rendition needs video filtering, and
	// whether the source is Dolby Vision (a whole-stream property, so
	// any reencode rendition carrying the flag implies it for all).
	anyFilter := false
	dolbyVision := false
	for _, p := range dsts {
		if p.Remux {
			continue
		}
		if p.MaxHeight > 0 || p.MaxFPS > 0 {
			anyFilter = true
		}
		if p.DolbyVision {
			dolbyVision = true
		}
	}

	if !anyFilter && !dolbyVision {
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

	// At least one rendition needs filtering (or Dolby Vision
	// conversion), so we route all reencode renditions through a
	// split. Branches that need no filtering are identity
	// pass-throughs (virtually free).
	var idxs []int
	for i, p := range dsts {
		if p.Remux {
			continue
		}
		idxs = append(idxs, i)
	}
	n := len(idxs)
	var parts []string

	// Source feeding the split. For Dolby Vision, a single libplacebo
	// pass converts the whole stream to HDR10 before the split.
	splitSrc := "[0:v]"
	if dolbyVision {
		parts = append(parts, "[0:v]"+dolbyVisionFilter+"[dv]")
		splitSrc = "[dv]"
	}

	// <src>split=N[s0][s1]...
	var split strings.Builder
	split.WriteString(splitSrc + "split=" + strconv.Itoa(n))
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
// -protocol_whitelist fd,pipe: the input arrives on fd 0 (the
// staged spool file, bound by the agent), so no filesystem
// protocol is whitelisted and a file reference inside attacker
// bytes — relative or absolute — cannot be opened during the
// encode. The whitelist is a per-input option; the muxer's own
// output opens (media, playlist) run under the output context and
// are unaffected. pipe stays listed for the fd-3 progress pipe —
// fd: itself can only address the default descriptors (0 to read,
// 1 to write; anything else needs the per-URL-unsettable -fd
// option), which is why progress can't ride the fd protocol.
// -f format: skip demuxer auto-detection so attacker bytes can't
// coerce ffmpeg into running concat, HLS, or similar multi-URL
// demuxers. -hwaccel none: force software decoding so ffmpeg
// doesn't fail on codecs without a hardware decoder (e.g. AV1).
// -nostdin / -hide_banner: standard unattended-run hygiene.
func inputArgs(format string) []string {
	return []string{
		"-y", "-nostdin", "-hide_banner",
		"-hwaccel", "none",
		"-protocol_whitelist", "fd,pipe",
		"-progress", "pipe:3", "-nostats",
		"-f", format,
	}
}

// anyDolbyVision reports whether any reencode rendition in dsts needs
// the Dolby Vision conversion.
func anyDolbyVision(dsts []EncodeParams) bool {
	for _, p := range dsts {
		if !p.Remux && p.DolbyVision {
			return true
		}
	}
	return false
}

// hwDeviceArgs returns the global flags that create the Vulkan device
// dolbyVisionFilter runs on, or nil when no rendition needs it. The
// device is created with no explicit physical-device index so
// libavutil picks the first available Vulkan device — on the appliance
// that is the bundled software rasterizer. These are global options
// and so are placed ahead of the per-input flags.
func hwDeviceArgs(dolbyVision bool) []string {
	if !dolbyVision {
		return nil
	}
	return []string{"-init_hw_device", "vulkan=vk", "-filter_hw_device", "vk"}
}

// hdr10OutputArgs tags a reencode output as HDR10 (BT.2020 PQ) so the
// encoder writes matching VUI signaling and the mp4 muxer writes a colr
// box. Returned only for Dolby Vision renditions, whose frames
// dolbyVisionFilter has converted to HDR10; nil otherwise.
func hdr10OutputArgs(dolbyVision bool) []string {
	if !dolbyVision {
		return nil
	}
	return []string{
		"-colorspace", "bt2020nc",
		"-color_primaries", "bt2020",
		"-color_trc", "smpte2084",
		"-color_range", "tv",
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
// passlog is a $STATS slot token,
// which the agent resolves to a path on its own filesystem.
// That path lands inside the colon-separated
// -x264-params / -x265-params value,
// where ':', '\' and '\n' would be reinterpreted as
// option separators,
// so the agent's stats root must avoid those characters.
// Nothing on this side of the protocol can check that.
func twoPassArgs(codec string, pass int, passlog string, extras ...string) []string {
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
// -force_key_frames T1,T2,... asks ffmpeg to mark a keyframe at the
// first encoded frame whose PTS reaches Ti. To land that keyframe
// on source frame Ni exactly, Ti is set half a frame before frame
// Ni's PTS — i.e. (Ni - 0.5) * Den/Num seconds — giving a half-frame
// margin on either side of any float-rounding ffmpeg does
// internally. The codec-params extras (returned for splicing into
// twoPassArgs) raise the maximum GOP size far above any plausible
// cut spacing and disable scene-cut detection, so the only
// keyframes the encoder produces are the forced ones — which is
// what makes ffmpeg's HLS muxer cut segments at exactly the
// requested frames.
//
// The earlier expr:eq(n,N1)+eq(n,N2)+... form was exact in frame
// index but tripped libavutil/eval.c's alternation-term limit
// (>~100 OR terms reject the expression outright with "Invalid
// force_key_frames expression"), which made any episode longer
// than ~6.7 minutes at our 4s target unencodable. The time form
// has no such limit.
//
// Assumes the rendition is encoded at the source's frame rate (no
// fps filter): then encoder PTS equals source display PTS.
// fps-changed re-encodes would need to convert source frame
// indices to encoder timestamps separately.
func keyframeArgs(cuts []int64, rate FrameRate) (forceArgs []string, extras []string) {
	if len(cuts) == 0 {
		return nil, nil
	}
	if !rate.Positive() {
		panic("ffmpeg: keyframeArgs requires a positive frame rate")
	}
	// 2*Num as the divisor lets us encode the half-frame offset as
	// integer arithmetic before the single float division.
	denom := 2 * float64(rate.Num)
	var sb strings.Builder
	for i, c := range cuts {
		if i > 0 {
			sb.WriteByte(',')
		}
		// (c - 0.5) * Den/Num seconds. The rational is in general
		// irrational in binary (e.g. 1001/24000) and ffmpeg parses
		// -force_key_frames values via av_strtod into a double
		// anyway, so exact transport isn't possible. FormatFloat's
		// shortest-round-trip form gives the closest decimal that
		// re-parses to the same float64 we computed; any residual
		// imprecision is sub-nanosecond — orders of magnitude
		// smaller than the half-frame margin, which exists to
		// absorb exactly this kind of slop.
		t := float64((2*c-1)*int64(rate.Den)) / denom
		sb.WriteString(strconv.FormatFloat(t, 'f', -1, 64))
	}
	return []string{"-force_key_frames", sb.String()},
		[]string{"keyint=99999", "scenecut=0"}
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
