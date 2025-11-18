package tlog

import (
	"context"
	"log/slog"
	"time"
)

func Elapsed(ctx context.Context, label string, args ...any) func() {
	slog.InfoContext(ctx, label+"-begin", args...)
	t := time.Now()
	return func() {
		slog.InfoContext(ctx, label+"-end", "elapsed", time.Since(t))
	}
}
