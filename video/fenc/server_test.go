package fenc

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"encoding/json/v2"
)

func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	s := &Server{Spool: t.TempDir(), Stats: t.TempDir()}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return s, ts
}

// needFFmpeg skips the test unless a host ffmpeg is available.
func needFFmpeg(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("ffmpeg test, skipped in -short mode")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("host ffmpeg not available")
	}
}

// stageJob creates a job directory in the spool and returns its path.
func stageJob(t *testing.T, s *Server, name string) string {
	t.Helper()
	dir := filepath.Join(s.Spool, name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// generate runs host ffmpeg to create a synthetic source file.
func generate(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("generate source: %v\n%s", err, out)
	}
}

// postJob posts req and returns the streamed events.
func postJob(t *testing.T, ts *httptest.Server, req JobRequest) []Event {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(ts.URL+"/job", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /job: %s\n%s", resp.Status, msg)
	}
	var events []Event
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			t.Fatalf("bad event %q: %v", scanner.Text(), err)
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) == 0 || events[len(events)-1].Done == nil {
		t.Fatalf("stream did not end with a done event: %+v", events)
	}
	return events
}

func done(t *testing.T, events []Event) *Result {
	t.Helper()
	return events[len(events)-1].Done
}

func TestHello(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var h Hello
	if err := json.UnmarshalRead(resp.Body, &h); err != nil {
		t.Fatal(err)
	}
	if h.Protocol != Protocol {
		t.Errorf("protocol = %d, want %d", h.Protocol, Protocol)
	}
	if h.Cores < 1 {
		t.Errorf("cores = %d, want >= 1", h.Cores)
	}

	// Anything else at the root is not found.
	resp2, err := http.Get(ts.URL + "/nonsense")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("GET /nonsense = %s, want 404", resp2.Status)
	}
}

func TestJobValidation(t *testing.T) {
	s, ts := newTestServer(t)
	stageJob(t, s, "staged")

	tests := []struct {
		name string
		req  JobRequest
	}{
		{"unknown tool", JobRequest{Tool: "sh", Args: []string{"-c", "true"}, Job: "staged"}},
		{"empty job", JobRequest{Tool: "ffprobe", Job: ""}},
		{"dotdot job", JobRequest{Tool: "ffprobe", Job: ".."}},
		{"separator in job", JobRequest{Tool: "ffprobe", Job: "a/b"}},
		{"separator in input", JobRequest{Tool: "ffprobe", Job: "staged", Input: "a/b"}},
		{"separator in stdout", JobRequest{Tool: "ffprobe", Job: "staged", Stdout: "a/b"}},
		{"separator in stats", JobRequest{Tool: "ffprobe", Job: "staged", Stats: "a/b"}},
		{"slot stats without stats", JobRequest{Tool: "ffmpeg", Job: "staged", Args: []string{"-x265-params", "pass=1:stats=$STATS/r0"}}},
		{"escape in slot arg", JobRequest{Tool: "ffprobe", Job: "staged", Args: []string{"$OUT/../../x"}}},
		{"unstaged job dir", JobRequest{Tool: "ffprobe", Job: "nonexistent"}},
		{"unstaged input", JobRequest{Tool: "ffprobe", Job: "staged", Input: "missing.mkv"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.Post(ts.URL+"/job", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %s, want 400", resp.Status)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	req := JobRequest{
		Args: []string{
			"-i", "fd:",
			"-x265-params", "pass=1:stats=$STATS/rf01:open-gop=0",
			"-hls_segment_filename", "$OUT/media0.mp4",
			"$OUT/stream0.m3u8",
			"plain",
		},
	}
	got := req.resolve("/spool/job1/out", "/stats/batch1")
	want := []string{
		"-i", "fd:",
		"-x265-params", "pass=1:stats=/stats/batch1/rf01:open-gop=0",
		"-hls_segment_filename", "/spool/job1/out/media0.mp4",
		"/spool/job1/out/stream0.m3u8",
		"plain",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDeleteStats(t *testing.T) {
	s, ts := newTestServer(t)
	dir := filepath.Join(s.Stats, "rf01")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "log"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	del := func() int {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/stats/rf01", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}

	if code := del(); code != http.StatusNoContent {
		t.Errorf("first delete = %d, want 204", code)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("stats dir still exists after delete")
	}
	// Idempotent: deleting released stats succeeds.
	if code := del(); code != http.StatusNoContent {
		t.Errorf("second delete = %d, want 204", code)
	}
}

func TestSweepOrphans(t *testing.T) {
	s := &Server{Spool: t.TempDir(), Stats: t.TempDir()}
	old := time.Now().Add(-48 * time.Hour)
	for _, dir := range []string{
		filepath.Join(s.Spool, "job-old"),
		filepath.Join(s.Stats, "stats-old"),
	} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(dir, old, old); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{
		filepath.Join(s.Spool, "job-new"),
		filepath.Join(s.Stats, "stats-new"),
	} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	s.SweepOrphans(24 * time.Hour)

	if _, err := os.Stat(filepath.Join(s.Spool, "job-old")); err == nil {
		t.Error("job-old survived the sweep")
	}
	if _, err := os.Stat(filepath.Join(s.Stats, "stats-old")); err == nil {
		t.Error("stats-old survived the sweep")
	}
	if _, err := os.Stat(filepath.Join(s.Spool, "job-new")); err != nil {
		t.Errorf("job-new was swept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Stats, "stats-new")); err != nil {
		t.Errorf("stats-new was swept: %v", err)
	}
}

func TestJobProbe(t *testing.T) {
	needFFmpeg(t)
	s, ts := newTestServer(t)
	jobDir := stageJob(t, s, "job1")
	generate(t,
		"-y", "-f", "lavfi", "-i", "testsrc2=duration=1:size=160x90:rate=24",
		"-c:v", "mpeg4", "-f", "matroska",
		filepath.Join(jobDir, "source.mkv"),
	)

	events := postJob(t, ts, JobRequest{
		Tool:   "ffprobe",
		Args:   []string{"-v", "error", "-print_format", "json", "-show_format", "fd:"},
		Job:    "job1",
		Input:  "source.mkv",
		Stdout: "probe.json",
		Stats:  "st1",
	})
	if res := done(t, events); res.Exit != 0 {
		t.Fatalf("exit = %d, error %q, stderr:\n%s", res.Exit, res.Error, res.Stderr)
	}

	b, err := os.ReadFile(filepath.Join(jobDir, "out", "probe.json"))
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if !strings.Contains(string(b), "matroska") {
		t.Errorf("probe output missing container format:\n%s", b)
	}

	// The declared stats dir exists after the run.
	if fi, err := os.Stat(filepath.Join(s.Stats, "st1")); err != nil || !fi.IsDir() {
		t.Errorf("stats dir not created: %v", err)
	}
}

func TestJobPipe0(t *testing.T) {
	needFFmpeg(t)
	s, ts := newTestServer(t)
	jobDir := stageJob(t, s, "job1")
	generate(t,
		"-y", "-f", "lavfi", "-i", "testsrc2=duration=1:size=160x90:rate=24",
		"-c:v", "mpeg4", "-f", "matroska",
		filepath.Join(jobDir, "source.mkv"),
	)

	events := postJob(t, ts, JobRequest{
		Tool: "ffprobe",
		Args: []string{
			"-v", "error",
			"-protocol_whitelist", "pipe",
			"-format_whitelist", "matroska",
			"-print_format", "json", "-show_format",
			"pipe:0",
		},
		Job:    "job1",
		Input:  "source.mkv",
		Stdout: "probe.json",
	})
	if res := done(t, events); res.Exit != 0 {
		t.Fatalf("exit = %d, error %q, stderr:\n%s", res.Exit, res.Error, res.Stderr)
	}
	b, err := os.ReadFile(filepath.Join(jobDir, "out", "probe.json"))
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if !strings.Contains(string(b), "matroska") {
		t.Errorf("probe output missing container format:\n%s", b)
	}
}

func TestJobToolFailure(t *testing.T) {
	needFFmpeg(t)
	s, ts := newTestServer(t)
	jobDir := stageJob(t, s, "job1")
	if err := os.WriteFile(filepath.Join(jobDir, "bad.bin"),
		[]byte("this is not a media file"), 0o644); err != nil {
		t.Fatal(err)
	}

	events := postJob(t, ts, JobRequest{
		Tool:  "ffprobe",
		Args:  []string{"-v", "error", "-show_format", "fd:"},
		Job:   "job1",
		Input: "bad.bin",
	})
	res := done(t, events)
	if res.Exit == 0 {
		t.Error("probing garbage succeeded")
	}
	if res.Stderr == "" {
		t.Error("no stderr captured from failing tool")
	}
}

func TestJobProgressAndOutput(t *testing.T) {
	needFFmpeg(t)
	s, ts := newTestServer(t)
	jobDir := stageJob(t, s, "job1")
	generate(t,
		"-y", "-f", "lavfi", "-i", "testsrc2=duration=2:size=160x90:rate=24",
		"-c:v", "mpeg4", "-f", "matroska",
		filepath.Join(jobDir, "source.mkv"),
	)

	events := postJob(t, ts, JobRequest{
		Tool: "ffmpeg",
		Args: []string{
			"-y", "-nostdin", "-hide_banner",
			"-progress", "pipe:3", "-nostats",
			"-i", "fd:",
			"-c:v", "copy",
			"$OUT/out.mkv",
		},
		Job:      "job1",
		Input:    "source.mkv",
		Progress: true,
	})
	if res := done(t, events); res.Exit != 0 {
		t.Fatalf("exit = %d, error %q, stderr:\n%s", res.Exit, res.Error, res.Stderr)
	}

	var sawProgress bool
	for _, ev := range events {
		if ev.OutTimeUS > 0 {
			sawProgress = true
		}
	}
	if !sawProgress {
		t.Error("no progress events received")
	}

	fi, err := os.Stat(filepath.Join(jobDir, "out", "out.mkv"))
	if err != nil {
		t.Fatalf("output not written: %v", err)
	}
	if fi.Size() == 0 {
		t.Error("output is empty")
	}
}

func TestJobCancel(t *testing.T) {
	needFFmpeg(t)
	s, ts := newTestServer(t)
	jobDir := stageJob(t, s, "job1")

	// -re paces the lavfi source at realtime, so this job would
	// run for a minute if cancellation didn't kill it.
	req := JobRequest{
		Tool: "ffmpeg",
		Args: []string{
			"-y", "-nostdin", "-hide_banner",
			"-progress", "pipe:3", "-nostats",
			"-re", "-f", "lavfi", "-i", "testsrc2=duration=60:size=160x90:rate=24",
			"-c:v", "mpeg4",
			"$OUT/out.mkv",
		},
		Job:      "job1",
		Progress: true,
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		ts.URL+"/job", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	resp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Wait for the first progress event so the tool is known to be
	// running, then hang up.
	scanner := bufio.NewScanner(resp.Body)
	if !scanner.Scan() {
		t.Fatalf("no first event: %v", scanner.Err())
	}
	cancel()

	// The deferred ts.Close blocks until the handler returns, so a
	// leaked ffmpeg would hang the test; the elapsed check is just
	// a friendlier failure.
	for scanner.Scan() {
	}
	if elapsed := time.Since(start); elapsed > 30*time.Second {
		t.Errorf("cancellation took %v", elapsed)
	}

	// The agent removes a canceled job's directory before its
	// handler returns, and Close blocks until the handler returns.
	ts.Close()
	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Errorf("canceled job directory was not removed: %v", err)
	}
}
