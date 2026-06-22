package timing

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerExplicitHeaderThenBody(t *testing.T) {
	h := Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		Add(req.Context(), "handler", 0)
		w.WriteHeader(http.StatusCreated)
		if _, err := io.WriteString(w, "created"); err != nil {
			t.Fatal(err)
		}
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	res := rec.Result()
	defer res.Body.Close()

	if got := res.StatusCode; got != http.StatusCreated {
		t.Fatalf("status = %d, want %d", got, http.StatusCreated)
	}
	if got := res.Header.Values("Server-Timing"); len(got) != 1 || got[0] != "handler" {
		t.Fatalf("Server-Timing = %q, want [handler]", got)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(body); got != "created" {
		t.Fatalf("body = %q, want %q", got, "created")
	}
}

func TestResponseIgnoresSecondWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	resp := &response{
		w: rec,
		t: &table{},
	}

	resp.WriteHeader(http.StatusAccepted)
	resp.WriteHeader(http.StatusInternalServerError)

	if got := rec.Code; got != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", got, http.StatusAccepted)
	}
}

func TestResponseControllerFlushMarksHeaderWritten(t *testing.T) {
	rec := &countingResponseWriter{header: http.Header{}}
	resp := &response{
		w: rec,
		t: &table{},
	}

	if err := http.NewResponseController(resp).Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(resp, "body"); err != nil {
		t.Fatal(err)
	}

	if got := rec.writeHeaderCount; got != 1 {
		t.Fatalf("WriteHeader calls = %d, want 1", got)
	}
	if got := rec.code; got != http.StatusOK {
		t.Fatalf("status = %d, want %d", got, http.StatusOK)
	}
	if got := rec.body; got != "body" {
		t.Fatalf("body = %q, want %q", got, "body")
	}
}

type countingResponseWriter struct {
	header           http.Header
	code             int
	body             string
	writeHeaderCount int
}

func (w *countingResponseWriter) Header() http.Header {
	return w.header
}

func (w *countingResponseWriter) Write(p []byte) (int, error) {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	w.body += string(p)
	return len(p), nil
}

func (w *countingResponseWriter) WriteHeader(code int) {
	w.writeHeaderCount++
	if w.code == 0 {
		w.code = code
	}
}

func (w *countingResponseWriter) FlushError() error {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return nil
}
