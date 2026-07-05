package fenc

import (
	"context"
	"net"
	"net/http"
	"sync"
)

// NewInProcessClient returns a Client served by srv within this
// process: the local-execution mode of the protocol.
// Requests never touch the network — they travel over synchronous
// in-memory pipes — but pass through the real HTTP client and
// server, so streaming, flushing, and cancellation-by-disconnect
// behave exactly as they do against a remote agent.
//
// The client (and the goroutines serving it) lives for the rest
// of the process.
func NewInProcessClient(srv *Server) *Client {
	ln := newMemListener()
	go func() {
		hs := &http.Server{Handler: srv.Handler()}
		hs.Serve(ln)
	}()
	return &Client{
		// The host is never dialed; memListener supplies the
		// connections. The reserved .invalid TLD keeps any
		// misrouted request from resolving anywhere real.
		BaseURL: "http://fenc.invalid",
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return ln.dial(ctx)
				},
			},
		},
	}
}

// memListener is a net.Listener whose connections are in-memory
// pipes created by dial.
type memListener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

func newMemListener() *memListener {
	return &memListener{
		conns: make(chan net.Conn),
		done:  make(chan struct{}),
	}
}

// dial creates a connected in-memory pipe, handing one end to
// Accept and returning the other.
//
// The pipe is deliberately unbuffered: writes block until the
// peer reads, so a slow consumer applies backpressure to the
// event stream the way a real socket's send window would.
func (l *memListener) dial(ctx context.Context) (net.Conn, error) {
	c1, c2 := net.Pipe()
	select {
	case l.conns <- memConn{c2}:
		return memConn{c1}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.done:
		return nil, opError("dial", net.ErrClosed)
	}
}

func (l *memListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, opError("accept", net.ErrClosed)
	}
}

// opError dresses err the way a real network listener would, for
// callers that inspect more than errors.Is.
func opError(op string, err error) error {
	return &net.OpError{Op: op, Net: memAddr{}.Network(), Addr: memAddr{}, Err: err}
}

func (l *memListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *memListener) Addr() net.Addr { return memAddr{} }

// memConn is a net.Pipe end that reports the in-process address
// instead of net.Pipe's "pipe", so connection metadata that
// surfaces elsewhere (e.g. http.Request.RemoteAddr in logs) names
// what the connection really is.
type memConn struct{ net.Conn }

func (memConn) LocalAddr() net.Addr  { return memAddr{} }
func (memConn) RemoteAddr() net.Addr { return memAddr{} }

type memAddr struct{}

func (memAddr) Network() string { return "mem" }
func (memAddr) String() string  { return "fenc-inprocess" }
