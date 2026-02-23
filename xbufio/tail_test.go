package xbufio

import (
	"fmt"
	"strings"
	"testing"
)

func TestBoundedWriter_FitsInHead(t *testing.T) {
	w := &BoundedWriter{Max: 100}
	w.Write([]byte("hello\nworld\n"))
	want := "hello\nworld\n"
	if got := w.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBoundedWriter_MultipleWritesFitInHead(t *testing.T) {
	w := &BoundedWriter{Max: 50}
	w.Write([]byte("aaa"))
	w.Write([]byte("bbb"))
	w.Write([]byte("ccc"))
	want := "aaabbbccc"
	if got := w.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBoundedWriter_ExactlyFillsHead(t *testing.T) {
	w := &BoundedWriter{Max: 5}
	w.Write([]byte("ab"))
	w.Write([]byte("cde"))
	want := "abcde"
	if got := w.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBoundedWriter_HeadAndTailNoOverflow(t *testing.T) {
	// Total fits in head+tail (2*Max), so skip=0.
	w := &BoundedWriter{Max: 10}
	w.Write([]byte("0123456789")) // fills head
	w.Write([]byte("abcde"))      // fits in tail
	want := "0123456789abcde"
	if got := w.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBoundedWriter_TailDisplacesOldBytes(t *testing.T) {
	w := &BoundedWriter{Max: 10}
	w.Write([]byte("0123456789")) // fills head
	w.Write([]byte("aaaaaaaaaa")) // fills tail
	w.Write([]byte("bb"))         // displaces first 2 bytes of tail

	got := w.String()
	// Tail should be "aaaaaaaabb".
	if !strings.HasSuffix(got, "aaaaaaaabb") {
		t.Fatalf("tail: got %q", got)
	}
	if !strings.Contains(got, "2 bytes omitted") {
		t.Fatalf("skip marker: got %q", got)
	}
}

func TestBoundedWriter_IncrementalTailGrowth(t *testing.T) {
	w := &BoundedWriter{Max: 10}
	w.Write([]byte("0123456789")) // head
	w.Write([]byte("ab"))         // tail: "ab"
	w.Write([]byte("cd"))         // tail: "abcd"
	w.Write([]byte("efghij"))     // tail: "abcdefghij" (full, no skip yet)
	if w.skipped != 0 {
		t.Fatalf("expected no skip yet, got %d", w.skipped)
	}
	w.Write([]byte("KL")) // displaces "ab", tail: "cdefghijKL"
	if !strings.HasSuffix(w.String(), "cdefghijKL") {
		t.Fatalf("got %q", w.String())
	}
	if !strings.Contains(w.String(), "2 bytes omitted") {
		t.Fatalf("skip marker: got %q", w.String())
	}
}

func TestBoundedWriter_SingleWriteExceedsBoth(t *testing.T) {
	w := &BoundedWriter{Max: 10}
	// Single write of 50 bytes: head=first 10, tail=last 10, skip=30.
	w.Write([]byte("0123456789" + "AAAAAAAAAA" + "BBBBBBBBBB" + "CCCCCCCCCC" + "DDDDDDDDDD"))
	got := w.String()
	if !strings.HasPrefix(got, "0123456789") {
		t.Fatalf("head: got %q", got)
	}
	if !strings.HasSuffix(got, "DDDDDDDDDD") {
		t.Fatalf("tail: got %q", got)
	}
	if !strings.Contains(got, "30 bytes omitted") {
		t.Fatalf("skip marker: got %q", got)
	}
}

func TestBoundedWriter_LargeWriteReplacesTail(t *testing.T) {
	w := &BoundedWriter{Max: 10}
	w.Write([]byte("0123456789")) // head
	w.Write([]byte("old tail!!")) // tail full
	// A write larger than Max replaces the entire tail.
	w.Write([]byte(strings.Repeat("N", 20)))
	got := w.String()
	if !strings.HasSuffix(got, strings.Repeat("N", 10)) {
		t.Fatalf("tail: got %q", got)
	}
	// skip = old tail (10) + overflow (20-10) = 20
	if !strings.Contains(got, "20 bytes omitted") {
		t.Fatalf("skip marker: got %q", got)
	}
}

func TestBoundedWriter_LineAlignedOutput(t *testing.T) {
	w := &BoundedWriter{Max: 20}
	// Head: "first line\nsecond li" (20 bytes)
	w.Write([]byte("first line\nsecond li"))
	// Tail gets: "ne\nmiddle stuff\nlast line\n" (25 bytes, keeps last 20)
	w.Write([]byte("ne\nmiddle stuff\nlast line\n"))

	got := w.String()
	// Head trims to last newline: "first line\n"
	if !strings.HasPrefix(got, "first line\n") {
		t.Fatalf("head not line-aligned: got %q", got)
	}
	// Tail trims from first newline: should not start with partial line.
	if !strings.HasSuffix(got, "last line\n") {
		t.Fatalf("tail not line-aligned: got %q", got)
	}
	// "second li" (9) + partial tail before first newline get counted as omitted.
	if !strings.Contains(got, "omitted") {
		t.Fatalf("expected omitted marker: got %q", got)
	}
}

func TestBoundedWriter_NoNewlinesInHeadOrTail(t *testing.T) {
	// When there are no newlines, trimming is a no-op: full head and tail are kept.
	w := &BoundedWriter{Max: 5}
	w.Write([]byte("abcde")) // head
	w.Write([]byte("fghij")) // tail
	w.Write([]byte("klmno")) // displaces tail
	got := w.String()
	if !strings.HasPrefix(got, "abcde") {
		t.Fatalf("head: got %q", got)
	}
	if !strings.HasSuffix(got, "klmno") {
		t.Fatalf("tail: got %q", got)
	}
}

func TestBoundedWriter_SkipCountAccumulates(t *testing.T) {
	w := &BoundedWriter{Max: 5}
	w.Write([]byte("HHHHH")) // head
	w.Write([]byte("aaaaa")) // tail full
	w.Write([]byte("bb"))    // discard 2 → skip=2
	w.Write([]byte("cc"))    // discard 2 → skip=4
	w.Write([]byte("dd"))    // discard 2 → skip=6

	// tail should be "bccdd"
	got := w.String()
	if !strings.HasSuffix(got, "bccdd") {
		t.Fatalf("tail: got %q", got)
	}
	if !strings.Contains(got, "6 bytes omitted") {
		t.Fatalf("skip count: got %q", got)
	}
}

func TestBoundedWriter_Empty(t *testing.T) {
	w := &BoundedWriter{Max: 10}
	if got := w.String(); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestBoundedWriter_WriteReturnsFullLength(t *testing.T) {
	w := &BoundedWriter{Max: 5}
	data := []byte("this is a long string that exceeds the buffer")
	n, err := w.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) {
		t.Fatalf("Write returned %d, want %d", n, len(data))
	}
}

func TestBoundedWriter_SkipMarkerFormat(t *testing.T) {
	w := &BoundedWriter{Max: 10}
	w.Write([]byte("head line\n"))
	w.Write([]byte(strings.Repeat("x", 20)))
	w.Write([]byte("tail line\n"))

	got := w.String()
	// The marker should be on its own line, surrounded by blank lines.
	if !strings.Contains(got, "\n\n... ") {
		t.Fatalf("marker not preceded by blank line: got %q", got)
	}
	if !strings.Contains(got, " bytes omitted ...\n\n") {
		t.Fatalf("marker not followed by blank line: got %q", got)
	}
}

func TestBoundedWriter_RealisticStderr(t *testing.T) {
	w := &BoundedWriter{Max: 100}
	// Simulate ffmpeg: a few useful initial lines, lots of repeated junk, final summary.
	fmt.Fprintln(w, "ffmpeg version 6.1")
	fmt.Fprintln(w, "Input #0, matroska")
	for i := range 500 {
		fmt.Fprintf(w, "frame=%d repeated error message that is not useful\n", i)
	}
	fmt.Fprintln(w, "Conversion failed!")
	fmt.Fprintln(w, "Error: something broke")

	got := w.String()
	if !strings.HasPrefix(got, "ffmpeg version 6.1\n") {
		t.Fatalf("lost initial output: got %q...", got[:80])
	}
	if !strings.HasSuffix(got, "Error: something broke\n") {
		t.Fatalf("lost final output: ...got %q", got[len(got)-80:])
	}
	if !strings.Contains(got, "bytes omitted") {
		t.Fatal("expected truncation marker")
	}
	// Total output should be well bounded.
	if len(got) > 300 {
		t.Fatalf("output too large: %d bytes", len(got))
	}
}
