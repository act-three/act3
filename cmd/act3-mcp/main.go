// Command act3-mcp is an MCP server that manages
// the act3 development server lifecycle.
// It exposes tools for starting, stopping, reloading,
// and checking the status of the server.
//
// Claude Code spawns this process and communicates with it
// over stdio using the MCP protocol.
//
// The working directory is used for build and server commands.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var project, _ = os.Getwd()

func main() {
	s := &devServer{
		listenAddr: ":4444",
	}

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "act3-dev-server",
			Version: "0.1.0",
		},
		&mcp.ServerOptions{
			Instructions: "Manage the act3 development server. Use server_start before testing UI changes, server_reload after code changes, and server_stop when done.",
		},
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "server_start",
		Description: "Build and start the act3 dev server. Blocks until the server is listening and ready to accept requests. Returns the address the server is listening on.",
	}, s.start)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "server_reload",
		Description: "Rebuild and restart the act3 dev server. Kills the running server, rebuilds, starts again, and blocks until ready. Use this after making code changes.",
	}, s.reload)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "server_stop",
		Description: "Stop the act3 dev server. Kills the running process and confirms the port is free.",
	}, s.stop)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "server_status",
		Description: "Check the current state of the dev server: whether it's running, on what address, its PID, and how long it's been up.",
	}, s.status)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// devServer manages the lifecycle of a single act3 dev server process.
// All exported methods are safe for concurrent use.
type devServer struct {
	mu         sync.Mutex
	listenAddr string
	cmd        *exec.Cmd
	pgid       int
	startedAt  time.Time
}

type startInput struct {
	Listen  string `json:"listen,omitempty" jsonschema:"address to listen on, e.g. :4444"`
	Verbose bool   `json:"verbose,omitempty"`
}

type statusOutput struct {
	Running bool   `json:"running"`
	PID     int    `json:"pid,omitempty"`
	Listen  string `json:"listen,omitempty"`
	Uptime  string `json:"uptime,omitempty"`
}

func (s *devServer) start(_ context.Context, _ *mcp.CallToolRequest, in startInput) (*mcp.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning() {
		return textResultf("server already running (pid %d) on %s", s.cmd.Process.Pid, s.listenAddr), nil, nil
	}

	if in.Listen != "" {
		s.listenAddr = in.Listen
	}

	if err := s.doStart(in.Verbose); err != nil {
		return nil, nil, err
	}

	return textResultf("server started (pid %d) on %s", s.cmd.Process.Pid, s.listenAddr), nil, nil
}

func (s *devServer) reload(_ context.Context, _ *mcp.CallToolRequest, in startInput) (*mcp.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning() {
		if err := s.doStop(); err != nil {
			slog.Warn("stop during reload", "err", err)
		}
	}

	if in.Listen != "" {
		s.listenAddr = in.Listen
	}

	if err := s.doStart(in.Verbose); err != nil {
		return nil, nil, err
	}

	return textResultf("server reloaded (pid %d) on %s", s.cmd.Process.Pid, s.listenAddr), nil, nil
}

func (s *devServer) stop(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning() {
		return textResultf("server is not running"), nil, nil
	}
	if err := s.doStop(); err != nil {
		return nil, nil, err
	}
	return textResultf("server stopped"), nil, nil
}

func (s *devServer) status(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, statusOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := statusOutput{
		Running: s.isRunning(),
	}
	if out.Running {
		out.PID = s.cmd.Process.Pid
		out.Listen = s.listenAddr
		out.Uptime = time.Since(s.startedAt).Round(time.Second).String()
	}
	return nil, out, nil
}

// doStart builds and starts the server process.
//
// 1. Use `go run . [-v] [-listen addr]` in the project root.
// 3. Use Setpgid so the whole process group can be killed reliably.
// 4. Wait for the server to be ready (poll the listen address).
// 5. Set s.cmd, s.pgid, and s.startedAt on success.
func (s *devServer) doStart(verbose bool) error {
	tag := rand.Text()
	vflag := "-v=false"
	if verbose {
		vflag = "-v"
	}
	cmd := exec.Command("go", "run",
		"-ldflags", "-X ily.dev/act3/web.tag="+tag,
		".",
		vflag,
		"-listen", s.listenAddr,
	)
	cmd.Dir = project
	cmd.Stdout = nil
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start failed: %w", err)
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return fmt.Errorf("getpgid: %w", err)
	}

	buferr := &bytes.Buffer{}
	err = scanForListen(io.TeeReader(stderr, buferr))
	if err != nil {
		_ = killGroup(pgid, cmd)
		return fmt.Errorf("server did not become ready: %w\nstderr:\n%s", err, buferr.Bytes())
	}

	err = waitReady(s.listenAddr, tag, 250*time.Millisecond)
	if err != nil {
		_ = killGroup(pgid, cmd)
		return fmt.Errorf("server did not become ready: %w\nstderr:\n%s", err, buferr.Bytes())
	}

	s.cmd = cmd
	s.pgid = pgid
	s.startedAt = time.Now()
	go cmd.Wait() // Avoid zombies. (doStop also calls Wait.)
	return nil
}

// doStop kills the running server process and waits for it to exit.
//
// 1. Send SIGTERM to the process group.
// 2. Wait briefly for graceful shutdown.
// 3. Send SIGKILL if it doesn't exit.
// 4. Clear s.cmd etc.
func (s *devServer) doStop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	if err := killGroup(s.pgid, s.cmd); err != nil {
		return fmt.Errorf("stop failed: %w", err)
	}
	s.cmd = nil
	s.pgid = 0
	s.startedAt = time.Time{}
	return nil
}

// isRunning reports whether the server process is still alive.
func (s *devServer) isRunning() bool {
	if s.cmd == nil || s.cmd.Process == nil {
		return false
	}
	// Check if process is still alive.
	err := s.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// killGroup sends SIGTERM to the process group, waits briefly,
// then sends SIGKILL if the process hasn't exited.
func killGroup(pgid int, cmd *exec.Cmd) error {
	// SIGTERM the whole group.
	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(250 * time.Millisecond):
	}

	// SIGKILL.
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	select {
	case <-done:
		return nil
	case <-time.After(250 * time.Millisecond):
		return fmt.Errorf("process %d did not exit after SIGKILL", cmd.Process.Pid)
	}
}

func scanForListen(r io.Reader) error {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if strings.Contains(sc.Text(), " msg=listen ") {
			return nil
		}
	}
	return fmt.Errorf("msg=listen not found")
}

// waitReady polls the given address until a TCP connection succeeds
// or the timeout is reached.
func waitReady(addr, tag string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + host + "/-/status")
		if err == nil {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if !bytes.Contains(body, []byte(tag)) {
				return fmt.Errorf("tag not found in server response:\n%s", body)
			}
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}

func textResultf(format string, args ...any) *mcp.CallToolResult {
	msg := fmt.Sprintf(format, args...)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
