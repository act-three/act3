package xos

import (
	"fmt"
	"log/slog"
	"sync/atomic"
)

// cloneDegraded records the first clone failure since startup.
var cloneDegraded atomic.Pointer[error]

// CloneDegradation returns the first clone degradation since startup
// — a clone failure [Clone] or [CloneInto] rescued with a byte copy —
// or nil if cloning is working as expected.
// A non-nil value means bulk file traffic has degraded to full copies
// and the storage configuration needs attention.
func CloneDegradation() error {
	if p := cloneDegraded.Load(); p != nil {
		return *p
	}
	return nil
}

// degradeClone makes a rescued clone failure loud:
// it logs the failure and remembers the first one for [CloneDegradation].
func degradeClone(src, dst string, err error) {
	err = fmt.Errorf("clone %s to %s degraded to copy: %w", src, dst, err)
	cloneDegraded.CompareAndSwap(nil, &err)
	slog.Error("clone-degraded", "src", src, "dst", dst, "err", err)
}
