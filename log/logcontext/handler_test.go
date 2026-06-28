package logcontext

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func newLogger(buf *bytes.Buffer, extractors ...AttrExtractor) *slog.Logger {
	h := slog.NewTextHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})
	return slog.New(Handler(h, extractors...))
}

type sidKey struct{}

func sessionID(ctx context.Context) []slog.Attr {
	if id, _ := ctx.Value(sidKey{}).(string); id != "" {
		return []slog.Attr{slog.String("session", id)}
	}
	return nil
}

func TestExtractor(t *testing.T) {
	var buf bytes.Buffer
	log := newLogger(&buf, sessionID)

	ctx := context.WithValue(context.Background(), sidKey{}, "abc123")
	log.InfoContext(ctx, "hello")
	if got := buf.String(); !strings.Contains(got, "session=abc123") {
		t.Errorf("missing extracted attr: %q", got)
	}

	buf.Reset()
	log.InfoContext(context.Background(), "hello")
	if got := buf.String(); strings.Contains(got, "session=") {
		t.Errorf("extractor should add nothing when absent: %q", got)
	}
}

func TestExtractorSurvivesWith(t *testing.T) {
	var buf bytes.Buffer
	log := newLogger(&buf, sessionID).With("a", 1).WithGroup("g")

	ctx := context.WithValue(context.Background(), sidKey{}, "abc123")
	log.InfoContext(ctx, "hello")
	if got := buf.String(); !strings.Contains(got, "session=abc123") {
		t.Errorf("extractor dropped by With/WithGroup: %q", got)
	}
}
