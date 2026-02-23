// Package xbufio provides extended buffered I/O utilities.
package xbufio

import "bytes"

// BoundedWriter is an io.Writer that keeps the first and last Max
// bytes of output, discarding the middle. This is useful for
// capturing error output from long-running processes where the
// beginning (initial errors) and end (final state) are most useful.
type BoundedWriter struct {
	Max     int // capacity for each of head and tail
	head    []byte
	tail    []byte // ring buffer, allocated once at Max bytes
	pos     int    // next write position in tail
	full    bool   // true once tail has wrapped at least once
	skipped int64  // bytes discarded from the middle
}

func (w *BoundedWriter) Write(p []byte) (int, error) {
	n := len(p)
	// Fill head first.
	if len(w.head) < w.Max {
		room := w.Max - len(w.head)
		if room >= len(p) {
			w.head = append(w.head, p...)
			return n, nil
		}
		w.head = append(w.head, p[:room]...)
		p = p[room:]
	}
	// Allocate ring buffer on first use.
	if w.tail == nil {
		w.tail = make([]byte, w.Max)
	}
	// Write into the ring buffer.
	w.writeTail(p)
	return n, nil
}

// writeTail writes p into the circular tail buffer, updating skip count.
func (w *BoundedWriter) writeTail(p []byte) {
	if len(p) >= w.Max {
		// More data than tail capacity: keep only the last Max bytes.
		w.skipped += int64(len(p) - w.Max)
		if w.full {
			w.skipped += int64(w.Max)
		} else {
			w.skipped += int64(w.pos)
		}
		copy(w.tail, p[len(p)-w.Max:])
		w.pos = 0
		w.full = true
		return
	}
	// Count bytes we're overwriting.
	if w.full {
		w.skipped += int64(len(p))
	} else if w.pos+len(p) > w.Max {
		// First wrap: only the bytes that land on already-written
		// positions count as skipped.
		w.skipped += int64(w.pos + len(p) - w.Max)
	}
	// Copy in up to two segments.
	first := copy(w.tail[w.pos:], p)
	if first < len(p) {
		copy(w.tail, p[first:])
		w.full = true
	}
	w.pos = (w.pos + len(p)) % w.Max
	if !w.full && w.pos == 0 {
		w.full = true
	}
}

// tailBytes returns the tail contents in order.
func (w *BoundedWriter) tailBytes() []byte {
	if w.tail == nil {
		return nil
	}
	if !w.full {
		return w.tail[:w.pos]
	}
	out := make([]byte, w.Max)
	n := copy(out, w.tail[w.pos:])
	copy(out[n:], w.tail[:w.pos])
	return out
}

// Bytes returns the captured output. If output was truncated, a
// marker line showing the number of omitted bytes separates head
// and tail.
func (w *BoundedWriter) Bytes() []byte {
	tail := w.tailBytes()
	if w.skipped == 0 {
		return append(w.head, tail...)
	}
	// Trim head to last newline so we don't cut mid-line.
	head := trimToLastNewline(w.head)
	// Trim tail from first newline.
	rawTailLen := len(tail)
	tail = trimFromFirstNewline(tail)
	out := make([]byte, 0, len(head)+len(tail)+64)
	out = append(out, head...)
	out = appendSkipMarker(out, w.skipped+int64(len(w.head)-len(head))+int64(rawTailLen-len(tail)))
	out = append(out, tail...)
	return out
}

func (w *BoundedWriter) String() string {
	return string(w.Bytes())
}

func trimToLastNewline(b []byte) []byte {
	i := bytes.LastIndexByte(b, '\n')
	if i < 0 {
		return b
	}
	return b[:i+1]
}

func trimFromFirstNewline(b []byte) []byte {
	i := bytes.IndexByte(b, '\n')
	if i < 0 {
		return b
	}
	return b[i+1:]
}

func appendSkipMarker(dst []byte, n int64) []byte {
	// Ensure marker starts on its own line.
	if len(dst) > 0 && dst[len(dst)-1] != '\n' {
		dst = append(dst, '\n')
	}
	dst = append(dst, "\n... "...)
	dst = appendInt64(dst, n)
	dst = append(dst, " bytes omitted ...\n\n"...)
	return dst
}

func appendInt64(dst []byte, v int64) []byte {
	var buf [20]byte
	i := len(buf)
	if v == 0 {
		return append(dst, '0')
	}
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return append(dst, buf[i:]...)
}
