package fenc

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// clientModes runs f once per transport: over loopback TCP, over
// httptest's in-memory network, and over the in-process pipes of
// NewInProcessClient. All three must be behaviorally identical.
func clientModes(t *testing.T, f func(t *testing.T, s *Server, c *Client)) {
	t.Run("tcp", func(t *testing.T) {
		s := &Server{Spool: t.TempDir(), Stats: t.TempDir()}
		ts := httptest.NewTestServer(t, s.Handler())
		ts.Start()
		f(t, s, &Client{BaseURL: ts.URL})
	})
	t.Run("memnet", func(t *testing.T) {
		s := &Server{Spool: t.TempDir(), Stats: t.TempDir()}
		ts := httptest.NewTestServer(t, s.Handler())
		hc := ts.Client() // also sets ts.URL
		f(t, s, &Client{BaseURL: ts.URL, HTTPClient: hc})
	})
	t.Run("inprocess", func(t *testing.T) {
		s := &Server{Spool: t.TempDir(), Stats: t.TempDir()}
		f(t, s, NewInProcessClient(s))
	})
}

func TestClientHello(t *testing.T) {
	clientModes(t, func(t *testing.T, s *Server, c *Client) {
		h, err := c.Hello(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if h.Protocol != Protocol {
			t.Errorf("protocol = %d, want %d", h.Protocol, Protocol)
		}
		if h.Cores < 1 {
			t.Errorf("cores = %d, want >= 1", h.Cores)
		}
	})
}

func TestClientJobRejected(t *testing.T) {
	clientModes(t, func(t *testing.T, s *Server, c *Client) {
		_, err := c.Job(t.Context(), JobRequest{
			Tool: "sh", Args: []string{"-c", "true"}, Job: "nope",
		}, nil)
		if err == nil || !strings.Contains(err.Error(), "unknown tool") {
			t.Errorf("err = %v, want agent rejection naming the tool", err)
		}
	})
}

func TestClientDeleteStats(t *testing.T) {
	clientModes(t, func(t *testing.T, s *Server, c *Client) {
		dir := filepath.Join(s.Stats, "rf01")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := c.DeleteStats(t.Context(), "rf01"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Error("stats dir still exists after delete")
		}
		// Idempotent.
		if err := c.DeleteStats(t.Context(), "rf01"); err != nil {
			t.Errorf("second delete: %v", err)
		}
	})
}

func TestClientJob(t *testing.T) {
	needFFmpeg(t)
	clientModes(t, func(t *testing.T, s *Server, c *Client) {
		jobDir := stageJob(t, s, "job1")
		generate(t,
			"-y", "-f", "lavfi", "-i", "testsrc2=duration=2:size=160x90:rate=24",
			"-c:v", "mpeg4", "-f", "matroska",
			filepath.Join(jobDir, "source.mkv"),
		)

		var last time.Duration
		res, err := c.Job(t.Context(), JobRequest{
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
		}, func(d time.Duration) { last = d })
		if err != nil {
			t.Fatal(err)
		}
		if res.Exit != 0 {
			t.Fatalf("exit = %d, error %q, stderr:\n%s", res.Exit, res.Error, res.Stderr)
		}
		if last <= 0 {
			t.Error("no progress reported")
		}
		if fi, err := os.Stat(filepath.Join(jobDir, "out", "out.mkv")); err != nil || fi.Size() == 0 {
			t.Errorf("output missing or empty: %v", err)
		}
	})
}

func TestClientJobToolFailure(t *testing.T) {
	needFFmpeg(t)
	clientModes(t, func(t *testing.T, s *Server, c *Client) {
		jobDir := stageJob(t, s, "job1")
		if err := os.WriteFile(filepath.Join(jobDir, "bad.bin"),
			[]byte("not media"), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := c.Job(t.Context(), JobRequest{
			Tool:  "ffprobe",
			Args:  []string{"-v", "error", "-show_format", "fd:"},
			Job:   "job1",
			Input: "bad.bin",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Exit == 0 {
			t.Error("probing garbage succeeded")
		}
		if res.Stderr == "" {
			t.Error("no stderr in result")
		}
	})
}

func TestClientJobCancel(t *testing.T) {
	needFFmpeg(t)
	clientModes(t, func(t *testing.T, s *Server, c *Client) {
		stageJob(t, s, "job1")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		// Cancel on the first progress event, once the tool is
		// demonstrably running. The timer is a backstop in case
		// no progress ever arrives.
		backstop := time.AfterFunc(5*time.Second, cancel)
		defer backstop.Stop()
		start := time.Now()
		firstProgress := func(time.Duration) { cancel() }
		_, err := c.Job(ctx, JobRequest{
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
		}, firstProgress)
		if err == nil {
			t.Fatal("cancelled job returned no error")
		}
		if elapsed := time.Since(start); elapsed > 30*time.Second {
			t.Errorf("cancellation took %v", elapsed)
		}
	})
}
