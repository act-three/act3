package panicstack

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Handler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(req.Context(), "panic", "error", r)
				debug.PrintStack()
			}
		}()
		h.ServeHTTP(w, req)
	}
}
