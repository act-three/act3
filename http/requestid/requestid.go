package requestid

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"

	"ily.dev/act3/log/logcontext"
)

type key int

const serverKey key = iota

// FromContext returns the request ID stored in ctx,
// or the empty string if there isn't one.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(serverKey).(string)
	return id
}

func Handler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		serverID := rand.Text()[:8]
		ctx = context.WithValue(ctx, serverKey, serverID)
		ctx = logcontext.With(ctx, slog.Group("request", "id", serverID))
		req = req.WithContext(ctx)
		w.Header().Add("Request-ID", serverID)
		h.ServeHTTP(w, req)
	}
}
