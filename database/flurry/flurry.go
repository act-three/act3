/*
Flurry IDs. Like snowflake IDs, but using fewer bits.
It doesn't support as many IDs per second (16,000/s sustained).
It only runs on a single machine (no machine ID field).

	timestamp bits = 41
	machine id bits = 0
	sequence bits = 4

The timestamps embedded in returned IDs aren't guaranteed to be
accurate: if NewID is called more than 16x/ms, the returned values
race ahead of the actual time, using future timestamps to allocate
more IDs rather than failing.
*/
package flurry

import (
	"encoding/binary"
	"strings"
	"sync"
	"time"

	"ily.dev/act3/encoding/base32c"
)

const (
	tsbit = 41
	sqbit = 4
	size  = tsbit + sqbit
	len   = size / 5
)

var epoch = must(time.Parse("2006", "2025")).UnixMilli()

var (
	mu   sync.Mutex
	last uint64
)

func NewID() string {
	t := time.Now().UnixMilli() - epoch
	v := alloc(t)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v<<(64-size))
	return strings.ToLower(base32c.EncodeToString(b)[:len])
}

func alloc(t int64) uint64 {
	mu.Lock()
	defer mu.Unlock()
	cur := uint64(t << sqbit)
	if cur > last {
		last = cur
		return cur
	}
	last++
	return last
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
