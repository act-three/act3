// Package fenc implements the encoder-agent protocol:
// the contract between act3 and the fenc box,
// the sibling VM that executes ffmpeg and ffprobe on act3's behalf.
//
// The agent is deliberately minimal.
// It holds no queue, makes no plans, and applies no policy;
// act3's task queue is the only scheduler,
// and all encode intelligence stays in act3.
// The agent's whole job is:
// run a tool with the argv it is given,
// report progress,
// and keep two-pass encoder stats between passes.
//
// # Protocol
//
// The agent serves HTTP:
//
//	GET /               handshake: protocol version and core count
//	POST /job           run one tool invocation
//	DELETE /stats/{id}  release a pass-1 stats directory
//
// A job request carries a slot-tokenized argv and an I/O manifest
// (see [JobRequest]).
// Bulk data never rides the HTTP connection:
// inputs are staged into a job directory in the spool —
// a filesystem shared between act3 and the agent —
// before the job is posted,
// and outputs (including captured tool stdout)
// appear under the job directory for the caller to collect.
//
// The response to POST /job streams newline-delimited JSON
// [Event] values:
// progress positions while the tool runs,
// then a final event carrying the exit status and bounded stderr.
// The request's lifetime is the job's lifetime:
// dropping the connection cancels the job and kills the tool,
// and the agent removes the canceled job's directory —
// the caller cannot know when the killed tool stops writing,
// so it cannot safely remove the directory itself.
//
// The agent trusts its caller.
// Reaching it at all is access control's job
// (the tailscale ACL admits only act3);
// the agent merely refuses to run anything but ffmpeg and ffprobe
// and rejects path escapes in the manifest and in slot-bearing
// arguments.
package fenc

// Protocol is the protocol version implemented by this package,
// reported in the [Hello] handshake.
const Protocol = 1

// Slot tokens.
// Argv in a [JobRequest] refers to the files a tool writes
// through these tokens rather than concrete paths;
// the agent resolves them against its own filesystem layout.
const (
	SlotOut   = "$OUT"   // $OUT/<name>: output file <name> in the job's out directory
	SlotStats = "$STATS" // $STATS/<name>: file <name> in the job's declared stats directory
)

// Hello is the response to GET /:
// the agent's protocol version and how many CPU cores it runs on.
// act3 sizes its cpu task queue from Cores.
type Hello struct {
	Protocol int `json:"protocol"`
	Cores    int `json:"cores"`
}

// A JobRequest describes one tool invocation, POSTed to /job.
type JobRequest struct {
	// Tool is the program to run: "ffmpeg" or "ffprobe".
	// The agent runs nothing else.
	Tool string `json:"tool"`

	// Args is the tool's argument list,
	// with slot tokens in place of paths.
	Args []string `json:"args"`

	// Job names the job's directory inside the spool.
	// The caller creates the directory and stages the input file
	// into it before posting the job;
	// outputs appear under <job>/out/.
	Job string `json:"job"`

	// Input is the staged input file's name inside the job
	// directory, presented on the tool's standard input
	// (argv reads it as fd: or pipe:0).
	// Empty when the job has no media input.
	Input string `json:"input,omitzero"`

	// Stdout names an output file (under <job>/out/) that
	// captures the tool's standard output.
	// Empty discards it.
	Stdout string `json:"stdout,omitzero"`

	// Stats names the pass-1 stats directory the job reads or
	// writes, if any.
	// SlotStats resolves inside it,
	// so a job can only touch the stats it declares.
	// The agent guarantees the directory exists before the tool
	// starts, and DELETE /stats/<Stats> releases it.
	// One stats directory typically serves a whole encode batch:
	// pass 1 writes every rendition's log there
	// and each pass-2 job reads its own back.
	Stats string `json:"stats,omitzero"`

	// Progress requests progress events.
	// Args must include "-progress pipe:3";
	// the agent supplies the pipe and parses the tool's
	// out_time_us reports into [Event] values.
	Progress bool `json:"progress,omitzero"`
}

// An Event is one line in the response stream of POST /job.
// Exactly one field is set per event.
type Event struct {
	// OutTimeUS is the tool's position in the output timeline,
	// in microseconds, as reported by ffmpeg -progress.
	// A zero position is never sent.
	OutTimeUS int64 `json:"out_time_us,omitzero"`

	// Done carries the job's final status.
	// It is the stream's last event.
	Done *Result `json:"done,omitzero"`
}

// A Result is the final status of a job.
type Result struct {
	// Exit is the tool's exit code.
	// 0 means success;
	// -1 means the tool did not run to completion
	// (killed by cancellation, or see Error).
	Exit int `json:"exit"`

	// Error describes an agent-side failure,
	// as opposed to the tool exiting nonzero.
	Error string `json:"error,omitzero"`

	// Stderr is the tool's captured standard error,
	// bounded to the first and last 100 KB.
	Stderr string `json:"stderr,omitzero"`
}
