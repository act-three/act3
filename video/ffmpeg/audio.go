package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// AudioEncodeParams describes one audio output rendition.
type AudioEncodeParams struct {
	File              *os.File // output file: media data is written here
	SourceStreamIndex int      // 0-based among audio streams; matches probe.Audio[i].Index
	Channels          int      // target channels (1, 2, or 6)
	Bitrate           int64    // kbit/s; ignored when StreamCopy
	StreamCopy        bool     // true: copy source audio stream as-is
}

// EncodeAudio extracts one source audio stream into an HLS audio
// rendition: a single-file fMP4 plus media playlist. The playlist
// is returned as a string; the fMP4 bytes are written to dst.File.
//
// Stream-copy (StreamCopy=true) skips re-encoding entirely; the
// caller decides this based on source codec and channel match.
//
// For Channels==6, ffmpeg's -channel_layout 5.1 forces the 5.1(back)
// layout, remapping 5.1(side) sources so CoreMedia's HLS parser
// accepts the output (it rejects PCE-coded 5.1(side) layouts).
func EncodeAudio(ctx context.Context, src *os.File, format string,
	dst AudioEncodeParams, duration time.Duration,
	onProgress func(float64),
) (playlist string, err error) {
	if dst.SourceStreamIndex < 0 {
		return "", fmt.Errorf("ffmpeg.EncodeAudio: SourceStreamIndex %d < 0", dst.SourceStreamIndex)
	}
	switch dst.Channels {
	case 1, 2, 6:
	default:
		return "", fmt.Errorf("ffmpeg.EncodeAudio: invalid Channels %d (want 1, 2, or 6)", dst.Channels)
	}
	if !dst.StreamCopy && dst.Bitrate <= 0 {
		return "", fmt.Errorf("ffmpeg.EncodeAudio: non-positive Bitrate %d", dst.Bitrate)
	}

	tmpDir, err := os.MkdirTemp("", "ffmpeg-audio-*")
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

	args := inputArgs(format)
	args = append(args, "-i", src.Name())
	args = append(args, "-map", fmt.Sprintf("0:a:%d", dst.SourceStreamIndex))
	args = append(args, "-vn", "-sn")
	if dst.StreamCopy {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac",
			"-b:a", fmt.Sprintf("%dk", dst.Bitrate),
			"-ac", strconv.Itoa(dst.Channels),
		)
		if dst.Channels == 6 {
			args = append(args, "-channel_layout", "5.1")
		}
	}
	args = append(args, hlsOutputArgs(mediaPath)...)
	args = append(args, plsPath)

	if err := runWithProgress(ctx, args, report); err != nil {
		return "", err
	}

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
	return string(b), nil
}
