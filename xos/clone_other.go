//go:build !linux && !darwin

package xos

import (
	"errors"
	"os"
)

func clone(dst string, src *os.File) error {
	return errors.ErrUnsupported
}

func cloneInto(dst, src *os.File) error {
	return errNoCloneInto
}
