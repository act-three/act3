package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProbeDuration runs ffprobe on r and returns the format duration.
func ProbeDuration(ctx context.Context, r io.Reader) (time.Duration, error) {
	var stdout, stderr bytes.Buffer
	c := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"pipe:0",
	)
	c.Stdin = r
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		return 0, errors.Join(err, errors.New(stderr.String()))
	}
	var result struct {
		Format struct {
			Duration time.Duration `json:"duration,string,format:sec"`
		} `json:"format"`
	}
	err = json.Unmarshal(stdout.Bytes(), &result)
	if err != nil {
		return 0, err
	}
	return result.Format.Duration, nil
}

// Remux remuxes src into an MP4 written to dst, copying video
// (with hvc1 tag) and transcoding audio to AAC. Subtitles are
// removed.
//
// Remux calls onProgress roughly once per second
// with the current transcode position in the video timeline.
//
// On failure the returned error includes ffmpeg's stderr output.
func Remux(ctx context.Context, src io.Reader, dst *os.File, onProgress func(time.Duration)) error {
	stderr := &bytes.Buffer{}
	c := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-progress", "pipe:3",
		"-nostats",
		//"-analyzeduration", "5M",

		"-i", "fd:",

		"-map", "0", // select all streams

		"-c:v", "copy",
		"-tag:v", "hvc1",
		"-movflags", "faststart",

		"-c:a", "aac",

		//"-map", "-0:s", // remove subtitles
		"-sn", // remove subtitles

		"-f", "mp4",

		// We can't use fd: here for some reason.
		// FFmpeg hangs at the beginning of the second pass.
		// (Reproduced locally on my macos laptop.)
		// I suspect a bug in ffmpeg.
		dst.Name(),
	)
	c.Stdin = src
	c.Stdout = dst
	c.Stderr = stderr

	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	c.ExtraFiles = []*os.File{pw}

	err = c.Start()
	pw.Close() // close parent's copy; child has its own fd
	defer pr.Close()
	if err != nil {
		pr.Close()
		return err
	}

	var wg sync.WaitGroup
	wg.Go(func() { readProgress(pr, onProgress) })
	defer wg.Wait()

	err = c.Wait()
	if err != nil {
		return errors.Join(err, errors.New(stderr.String()))
	}
	return nil
}

// readProgress reads ffmpeg -progress output from r and calls
// update with the current percentage (0–100) each time a new
// out_time_us value is parsed.
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
		update(time.Microsecond * time.Duration(us))
	}
}
