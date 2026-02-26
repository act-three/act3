package ffmpeg

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const dockerImage = "act3-ffmpeg"

// dockerFFmpeg runs ffmpeg inside the act3-ffmpeg Docker container
// with dir mounted as /work.
func dockerFFmpeg(dir string, args ...string) *exec.Cmd {
	a := []string{
		"run", "--rm",
		"-v", dir + ":/work",
		"-w", "/work",
		dockerImage,
		"/out/ffmpeg",
	}
	a = append(a, args...)
	return exec.Command("docker", a...)
}

func TestEncodeAV1ToHEVC(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not in PATH")
	}
	if _, err := exec.LookPath("mediastreamvalidator"); err != nil {
		t.Skip("mediastreamvalidator not in PATH")
	}

	// Verify the Docker image exists.
	if err := exec.Command("docker", "image", "inspect",
		dockerImage).Run(); err != nil {
		t.Skipf("%s image not built", dockerImage)
	}

	dir := t.TempDir()

	// Generate a tiny AV1 10-bit + EAC3 5.1 source, matching
	// the codec profile of our real content.
	t.Log("generating source...")
	cmd := dockerFFmpeg(dir,
		"-y",
		"-f", "lavfi", "-i", "testsrc2=duration=2:size=160x90:rate=24",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2:sample_rate=48000",
		"-c:v", "libsvtav1", "-preset", "12", "-crf", "55",
		"-pix_fmt", "yuv420p10le",
		"-c:a", "eac3", "-ac", "6",
		"source.mkv",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate source: %v\n%s", err, out)
	}

	// Pass 1: analysis.
	t.Log("running pass 1...")
	cmd = dockerFFmpeg(dir,
		"-y", "-hwaccel", "none",
		"-i", "source.mkv",
		"-c:v", "libx265", "-preset", "ultrafast",
		"-b:v", "6000k",
		"-x265-params", "pass=1:stats=passlog0",
		"-an", "-sn",
		"-f", "null", "/dev/null",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pass 1: %v\n%s", err, out)
	}

	// Pass 2: encode to HLS fMP4, using the same args as
	// hlsOutputArgs.
	t.Log("running pass 2...")
	cmd = dockerFFmpeg(dir,
		"-y", "-hwaccel", "none",
		"-i", "source.mkv",
		"-c:v", "libx265", "-preset", "ultrafast",
		"-tag:v", "hvc1",
		"-b:v", "6000k",
		"-x265-params", "pass=2:stats=passlog0",
		"-c:a", "aac", "-ac", "2",
		"-sn",
		"-f", "hls",
		"-hls_segment_type", "fmp4",
		"-hls_flags", "single_file",
		"-hls_playlist_type", "vod",
		"-hls_time", "6",
		"-hls_list_size", "0",
		"-hls_segment_filename", "media0.mp4",
		"stream0.m3u8",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pass 2: %v\n%s", err, out)
	}

	// Validate.
	plsPath := filepath.Join(dir, "stream0.m3u8")
	out, err := exec.Command(
		"mediastreamvalidator", plsPath,
	).CombinedOutput()
	_ = err

	output := string(out)
	t.Logf("mediastreamvalidator output:\n%s", output)

	if strings.Contains(output, "Error injecting segment data") {
		t.Error("mediastreamvalidator reported: Error injecting segment data")
	}
	if strings.Contains(output, "Processed 0 out of") {
		t.Error("mediastreamvalidator processed 0 segments")
	}
}
