// Package timing records timing data for HTTP requests
// and sends it to clients in the Server-Timing respnse header.
package timing

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type key string

var timingKey key

// Add adds d to the named metric.
// Multiple calls to Add for the same metric are cumulative;
// that is, Add(ctx, "x", 2) followed by Add(ctx, "x", 3)
// is equivalent to Add(ctx, "x", 5).
//
// If there is no metric table in ctx, Add has no effect.
// If the metrics have already been written to the response header,
// Add has no effect.
func Add(ctx context.Context, metric string, d time.Duration) {
	t, ok := ctx.Value(timingKey).(*table)
	if !ok {
		return
	}
	t.add(metric, d)
}

// Set is equivalent to Add with a duration of 0.
//
// If a metric already exists (with or without duration),
// Set has no effect.
// If there is no metric table in ctx, Set has no effect.
// If the metrics have already been written to the response header,
// Set has no effect.
func Set(ctx context.Context, metric string) {
	Add(ctx, metric, 0)
}

// Measure calls f, then adds the elapsed time to the given metric in ctx.
func Measure(ctx context.Context, metric string, f func()) {
	t0 := time.Now()
	f()
	Add(ctx, metric, time.Since(t0))
}

// Handler returns a new http.Handler that records timing data for each request.
// Inside h, code can use Add and Set functions to measure time taken.Handler
// The returned handler will add the accumulated timing data in the response's
// Server-Timing header field.
func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		t := &table{ctx: ctx, tab: map[string]time.Duration{}}
		ctx = context.WithValue(ctx, timingKey, t)
		req = req.WithContext(ctx)
		h.ServeHTTP(&response{w: w, t: t}, req)
	})
}

type response struct {
	w     http.ResponseWriter
	t     *table
	wrote bool
}

func (r *response) Header() http.Header {
	return r.w.Header()
}

func (r *response) Write(p []byte) (int, error) {
	if !r.wrote {
		r.WriteHeader(http.StatusOK)
	}
	return r.w.Write(p)
}

func (r *response) WriteHeader(code int) {
	if r.wrote {
		return
	}
	r.wrote = true
	r.t.addTo(r.w.Header())
	r.w.WriteHeader(code)
}

func (r *response) Flush() {
	_ = r.FlushError()
}

func (r *response) FlushError() error {
	if !r.wrote {
		r.WriteHeader(http.StatusOK)
	}
	return http.NewResponseController(r.w).Flush()
}

func (r *response) Unwrap() http.ResponseWriter {
	return r.w
}

type table struct {
	ctx context.Context

	mu  sync.Mutex
	tab map[string]time.Duration
}

func (t *table) add(metric string, dur time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tab[metric] += dur
}

// write writes accumulated metrics to the response header.
func (t *table) addTo(h http.Header) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for metric, dur := range t.tab {
		var s string
		if dur == 0 {
			s = metric
			slog.DebugContext(t.ctx, metric)
		} else {
			d := float64(dur.Microseconds()) / 1000
			s = fmt.Sprintf("%s;dur=%.2f", metric, d)
			slog.DebugContext(t.ctx, metric, "dur", d, "unit", "ms")
		}
		h.Add("Server-Timing", s)
	}
}
