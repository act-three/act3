package ffmpeg

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ily.dev/act3/video/fenc"
)

// The encoder agent every tool invocation runs on, and this
// process's path to the job spool it shares with that agent.
var (
	agent      *fenc.Client
	agentSpool string
)

// SetAgent directs all ffmpeg/ffprobe execution to the encoder
// agent reached by c.
// spool is this process's path to the job spool shared with that
// agent.
// SetAgent must be called before any media operation in this
// package.
func SetAgent(c *fenc.Client, spool string) {
	agent = c
	agentSpool = spool
}

// ReleaseStats releases an encode batch's pass-1 stats on the
// agent.
// Call it once every rendition of the batch has been encoded,
// or when the batch is abandoned.
func ReleaseStats(ctx context.Context, batch string) error {
	return agent.DeleteStats(ctx, batch)
}

// A job is one staged agent invocation:
// a directory in the spool holding the input clone
// and receiving outputs.
type job struct {
	name  string // directory name within the spool
	dir   string // this process's path to it
	input string // staged input name within it, or ""
}

// newJob creates a job directory in the spool and stages input
// into it. input may be nil for jobs with no media input.
func newJob(input *os.File) (*job, error) {
	if agent == nil {
		panic("ffmpeg: SetAgent not called")
	}
	var b [8]byte
	rand.Read(b[:])
	j := &job{name: "job-" + hex.EncodeToString(b[:])}
	j.dir = filepath.Join(agentSpool, j.name)
	if err := os.Mkdir(j.dir, 0o755); err != nil {
		return nil, err
	}
	if input != nil {
		j.input = "input"
		if err := stageFile(input, filepath.Join(j.dir, j.input)); err != nil {
			j.close()
			return nil, err
		}
	}
	return j, nil
}

// close removes the job directory and everything staged into or
// produced under it.
func (j *job) close() { os.RemoveAll(j.dir) }

// out returns this process's path to the named job output.
func (j *job) out(name string) string {
	return filepath.Join(j.dir, "out", name)
}

// run executes one tool invocation within the job, converting a
// tool failure into an error carrying the tool's stderr.
func (j *job) run(ctx context.Context, req fenc.JobRequest, onProgress func(time.Duration)) error {
	req.Job = j.name
	req.Input = j.input
	res, err := agent.Job(ctx, req, onProgress)
	if err != nil {
		return err
	}
	if res.Error != "" {
		return errors.Join(fmt.Errorf("%s: %s", req.Tool, res.Error), errors.New(res.Stderr))
	}
	if res.Exit != 0 {
		return errors.Join(fmt.Errorf("%s: exit status %d", req.Tool, res.Exit), errors.New(res.Stderr))
	}
	return nil
}
