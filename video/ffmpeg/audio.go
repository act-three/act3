package ffmpeg

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"ily.dev/act3/video/fenc"
)

// AudioEncodeParams describes one audio output rendition.
type AudioEncodeParams struct {
	Path              string // output file path; written by ffmpeg
	SourceStreamIndex int    // 0-based among audio streams; matches probe.Audio[i].Index
	Channels          int    // target channels (source count, or 2 for a downmix)
	SourceLayout      string // source ChannelLayout, used only to log layout remaps
	Bitrate           int64  // kbit/s; ignored when StreamCopy
	StreamCopy        bool   // true: copy source audio stream as-is
}

// EncodeAudio extracts one source audio stream into an HLS audio
// rendition: a single-file fMP4 plus media playlist. The playlist
// is returned as a string; the fMP4 is written to dst.Path.
//
// Stream-copy (StreamCopy=true) skips re-encoding entirely; the
// caller decides this based on source codec and channel match.
//
// Surround renditions are forced to the standard AAC channel layout
// for their channel count (see standardLayout). A non-standard source
// layout such as 5.1(side) otherwise makes ffmpeg emit a program
// config element (PCE), which CoreMedia's HLS parser rejects; forcing
// the standard layout remaps the channels into a PCE-free
// configuration. The remap is logged when it changes the layout.
func EncodeAudio(ctx context.Context, src *os.File, format string,
	dst AudioEncodeParams, duration time.Duration,
	onProgress func(float64),
) (playlist string, err error) {
	if dst.SourceStreamIndex < 0 {
		panic(fmt.Sprintf("ffmpeg.EncodeAudio: SourceStreamIndex %d < 0", dst.SourceStreamIndex))
	}
	if dst.Channels < 1 {
		panic(fmt.Sprintf("ffmpeg.EncodeAudio: non-positive Channels %d", dst.Channels))
	}
	if !dst.StreamCopy && dst.Bitrate <= 0 {
		panic(fmt.Sprintf("ffmpeg.EncodeAudio: non-positive Bitrate %d", dst.Bitrate))
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

	args := inputArgs(format)
	args = append(args, "-i", "fd:")
	args = append(args, "-map", fmt.Sprintf("0:a:%d", dst.SourceStreamIndex))
	args = append(args, "-vn", "-sn")
	if dst.StreamCopy {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac",
			"-b:a", fmt.Sprintf("%dk", dst.Bitrate),
			"-ac", strconv.Itoa(dst.Channels),
		)
		if dst.Channels > 2 {
			if layout := standardLayout[dst.Channels]; layout != "" {
				args = append(args, "-channel_layout", layout)
				if dst.SourceLayout != "" && dst.SourceLayout != layout {
					slog.InfoContext(ctx, "audio-channel-layout-remap",
						"channels", dst.Channels, "from", dst.SourceLayout, "to", layout)
				}
			} else {
				slog.WarnContext(ctx, "audio-channel-layout-no-standard-config",
					"channels", dst.Channels, "sourceLayout", dst.SourceLayout)
			}
		}
	}
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

// standardLayout maps a channel count to the ffmpeg
// channel-layout name that makes the AAC encoder emit a standard
// channel configuration (a non-zero channel_configuration, no PCE).
// AAC defines configurations for 1, 2, 3, 4, 5, 6, and 8 channels;
// 7-channel audio (6.1) and counts above 8 have no PCE-free
// configuration and are absent here.
var standardLayout = map[int]string{
	1: "mono",
	2: "stereo",
	3: "3.0",
	4: "4.0",
	5: "5.0",
	6: "5.1",
	8: "7.1",
}
