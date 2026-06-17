// Hashbench compares throughput of crypto/sha256 and blake3 over a
// large in-memory workload. Intended to be run on both the dev VM
// and act3 production hardware to decide which algorithm to use for
// content-hashing ingested video files.
//
// Usage: go run ./cmd/hashbench
package main

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"math/rand/v2"
	"runtime"
	"time"

	"lukechampine.com/blake3"
)

const (
	bufSize    = 1 << 20  // 1 MiB per Write
	totalBytes = 10 << 30 // 10 GiB total per algorithm
	iters      = totalBytes / bufSize
	warmupIter = 64
)

type algo struct {
	name string
	mk   func() hash.Hash
}

func main() {
	buf := make([]byte, bufSize)
	rng := rand.New(rand.NewPCG(0xdeadbeef, 0xcafebabe))
	for i := range buf {
		buf[i] = byte(rng.Uint32())
	}

	algos := []algo{
		{"sha256", func() hash.Hash { return sha256.New() }},
		{"blake3-256", func() hash.Hash { return blake3.New(32, nil) }},
	}

	fmt.Printf("go=%s GOARCH=%s GOOS=%s CPUs=%d\n",
		runtime.Version(), runtime.GOARCH, runtime.GOOS, runtime.NumCPU())
	fmt.Printf("buf=%d KiB  total=%d GiB  iters=%d\n\n",
		bufSize>>10, totalBytes>>30, iters)

	// Run each algo twice, alternating, to smooth thermal/cache bias.
	for pass := 1; pass <= 2; pass++ {
		fmt.Printf("pass %d\n", pass)
		for _, a := range algos {
			bench(a, buf)
		}
		fmt.Println()
	}
}

func bench(a algo, buf []byte) {
	h := a.mk()
	for range warmupIter {
		h.Write(buf)
	}
	h.Reset()

	start := time.Now()
	for range iters {
		h.Write(buf)
	}
	sum := h.Sum(nil)
	dur := time.Since(start)

	mibps := float64(totalBytes) / dur.Seconds() / (1 << 20)
	fmt.Printf("  %-11s  %7.1f MiB/s   (%6.2fs, digest=%x…)\n",
		a.name, mibps, dur.Seconds(), sum[:4])
}
