package fenc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"encoding/json/v2"

	"ily.dev/act3/xbufio"
)

// A Server is the agent's state: the two directories it touches.
// Spool holds per-job directories, shared with act3;
// Stats holds pass-1 encoder stats, private to the agent.
type Server struct {
	Spool string // job directories (the shared spool)
	Stats string // pass-1 stats directories (agent-private)
	Cores int    // reported in the handshake; 0 means runtime.NumCPU()
}

// Handler returns the agent's HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleHello)
	mux.HandleFunc("POST /job", s.handleJob)
	mux.HandleFunc("DELETE /stats/{id}", s.handleDeleteStats)
	return mux
}

func (s *Server) handleHello(w http.ResponseWriter, r *http.Request) {
	cores := s.Cores
	if cores == 0 {
		cores = runtime.NumCPU()
	}
	w.Header().Set("Content-Type", "application/json")
	json.MarshalWrite(w, Hello{Protocol: Protocol, Cores: cores})
}

func (s *Server) handleDeleteStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validName(id) {
		http.Error(w, "invalid stats id", http.StatusBadRequest)
		return
	}
	if err := os.RemoveAll(filepath.Join(s.Stats, id)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.InfoContext(r.Context(), "stats-released", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	var req JobRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := req.validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jobDir := filepath.Join(s.Spool, req.Job)
	if fi, err := os.Stat(jobDir); err != nil || !fi.IsDir() {
		http.Error(w, "job directory not staged: "+req.Job, http.StatusBadRequest)
		return
	}
	if req.Input != "" {
		if _, err := os.Stat(filepath.Join(jobDir, req.Input)); err != nil {
			http.Error(w, "input not staged: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	outDir := filepath.Join(jobDir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var statsDir string
	if req.Stats != "" {
		statsDir = filepath.Join(s.Stats, req.Stats)
		if err := os.MkdirAll(statsDir, 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// The request context is the job's lifetime: when the caller
	// drops the connection, the context ends and the tool is
	// killed. That is the protocol's only cancellation mechanism.
	ctx := r.Context()
	args := req.resolve(jobDir, outDir, statsDir)
	c := exec.CommandContext(ctx, req.Tool, args...)
	stderr := &xbufio.BoundedWriter{Max: 100_000}
	c.Stderr = stderr

	if req.Stdin {
		in, err := os.Open(filepath.Join(jobDir, req.Input))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer in.Close()
		c.Stdin = in
	}
	if req.Stdout != "" {
		out, err := os.Create(filepath.Join(outDir, req.Stdout))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer out.Close()
		c.Stdout = out
	}

	var pr, pw *os.File
	if req.Progress {
		var err error
		pr, pw, err = os.Pipe()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		c.ExtraFiles = []*os.File{pw}
	}

	slog.InfoContext(ctx, "job-start", "tool", req.Tool, "job", req.Job, "args", args)
	err := c.Start()
	if pw != nil {
		pw.Close() // close parent's copy; child has its own fd
	}
	if err != nil {
		if pr != nil {
			pr.Close()
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	ew := &eventWriter{w: w, rc: http.NewResponseController(w)}

	var wg sync.WaitGroup
	if pr != nil {
		wg.Go(func() {
			defer pr.Close()
			readProgress(pr, func(us int64) {
				ew.write(Event{OutTimeUS: us})
			})
		})
	}

	err = c.Wait()
	wg.Wait()

	res := &Result{Stderr: stderr.String()}
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		res.Exit = ee.ExitCode()
	} else if err != nil {
		res.Exit = -1
		res.Error = err.Error()
	}
	ew.write(Event{Done: res})
	slog.InfoContext(ctx, "job-done", "tool", req.Tool, "job", req.Job,
		"exit", res.Exit, "err", res.Error)
}

// validate checks a decoded JobRequest before any filesystem or
// process work happens.
func (req *JobRequest) validate() error {
	switch req.Tool {
	case "ffmpeg", "ffprobe":
	default:
		return fmt.Errorf("unknown tool %q", req.Tool)
	}
	if !validName(req.Job) {
		return fmt.Errorf("invalid job name %q", req.Job)
	}
	if req.Input != "" && !validName(req.Input) {
		return fmt.Errorf("invalid input name %q", req.Input)
	}
	if req.Stdout != "" && !validName(req.Stdout) {
		return fmt.Errorf("invalid stdout name %q", req.Stdout)
	}
	if req.Stats != "" && !validName(req.Stats) {
		return fmt.Errorf("invalid stats id %q", req.Stats)
	}
	if req.Input == "" && (req.Stdin || slicesContainsSubstring(req.Args, SlotInput)) {
		return errors.New("job references an input but names none")
	}
	if req.Stats == "" && slicesContainsSubstring(req.Args, SlotStats) {
		return errors.New("job references stats but declares none")
	}
	for _, a := range req.Args {
		if hasSlot(a) && strings.Contains(a, "..") {
			return fmt.Errorf("slot-bearing arg %q contains %q", a, "..")
		}
	}
	return nil
}

// validName reports whether name is usable as a single path
// element: nonempty, no separators, and not "." or "..".
func validName(name string) bool {
	return name != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, `/\`)
}

func hasSlot(arg string) bool {
	return strings.Contains(arg, SlotInput) ||
		strings.Contains(arg, SlotOut) ||
		strings.Contains(arg, SlotStats)
}

func slicesContainsSubstring(args []string, sub string) bool {
	for _, a := range args {
		if strings.Contains(a, sub) {
			return true
		}
	}
	return false
}

// resolve returns req.Args with each slot token replaced by its
// path on the agent's filesystem.
// statsDir is the job's declared stats directory,
// not the stats root: SlotStats reaches only the stats the job
// declared.
func (req *JobRequest) resolve(jobDir, outDir, statsDir string) []string {
	r := strings.NewReplacer(
		SlotInput, filepath.Join(jobDir, req.Input),
		SlotOut, outDir,
		SlotStats, statsDir,
	)
	args := make([]string, len(req.Args))
	for i, a := range req.Args {
		args[i] = r.Replace(a)
	}
	return args
}

// eventWriter serializes concurrent event writes to one response
// stream, flushing after each so progress reaches the caller
// promptly. Write errors are ignored: they mean the caller is
// gone, and the request context takes care of the job.
type eventWriter struct {
	mu sync.Mutex
	w  io.Writer
	rc *http.ResponseController
}

func (ew *eventWriter) write(ev Event) {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	if err := json.MarshalWrite(ew.w, ev); err != nil {
		return
	}
	io.WriteString(ew.w, "\n")
	ew.rc.Flush()
}

// readProgress parses ffmpeg -progress output from r, reporting
// each nonzero out_time_us position.
func readProgress(r io.Reader, report func(int64)) {
	scanner := bufio.NewScanner(r)
	// ffmpeg -progress lines are short key=value pairs (well under
	// 100 bytes). 4 KB is ample; anything bigger is a bug worth surfacing.
	scanner.Buffer(make([]byte, 1024), 4*1024)
	for scanner.Scan() {
		after, ok := strings.CutPrefix(scanner.Text(), "out_time_us=")
		if !ok {
			continue
		}
		us, err := strconv.ParseInt(after, 10, 64)
		if err != nil || us == 0 {
			continue
		}
		report(us)
	}
	if err := scanner.Err(); err != nil {
		slog.Warn("progress-read", "err", err)
	}
}
