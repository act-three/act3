package primaryredirect

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		target  string
		wantLoc string // empty means pass through
	}{
		{"primary host passes through", "https://ily.dev", "https://ily.dev/series?page=2", ""},
		{"host match ignores case", "https://ily.dev", "https://ILY.DEV/", ""},
		{"host match ignores request scheme", "https://ily.dev", "http://ily.dev/", ""},
		{"other host redirects", "https://ily.dev", "https://other.example/", "https://ily.dev/"},
		{"redirect keeps path and query", "https://ily.dev", "https://other.example/series?page=2", "https://ily.dev/series?page=2"},
		{"port is part of the host", "http://localhost:4445", "http://localhost:4446/", "http://localhost:4445/"},
		{"empty origin disables redirects", "", "https://other.example/", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := Handler(tt.origin, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("GET", tt.target, nil))
			if tt.wantLoc == "" {
				if !called {
					t.Fatal("wrapped handler was not called")
				}
				return
			}
			if called {
				t.Error("wrapped handler was called, want redirect")
			}
			if rec.Code != http.StatusTemporaryRedirect {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusTemporaryRedirect)
			}
			if got := rec.Header().Get("Location"); got != tt.wantLoc {
				t.Errorf("Location = %q, want %q", got, tt.wantLoc)
			}
		})
	}
}

func TestHandlerBadOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
	}{
		{"unparseable url", "https://ily.dev/\n"},
		{"bad scheme", "ftp://ily.dev"},
		{"host and port mistaken for scheme", "ily.dev:443"},
		{"empty host", "https://"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Handler(%q) did not panic", tt.origin)
				}
			}()
			Handler(tt.origin, http.NotFoundHandler())
		})
	}
}
