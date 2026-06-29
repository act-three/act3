package a

import "io"

type myError struct{}

func (*myError) Error() string { return "" }

func f(err error, other error) {
	_ = err == io.EOF // want `comparing errors with ==; use errors.Is instead`
	_ = err != io.EOF // want `comparing errors with !=; use errors.Is instead`
	_ = err == other  // want `comparing errors with ==; use errors.Is instead`

	// Comparing against nil is the idiomatic presence check, not a
	// sentinel comparison, so it is allowed.
	_ = err == nil
	_ = err != nil

	// Concrete error types are not the error interface; the analyzer
	// only flags the interface type.
	var e *myError
	_ = e == nil

	// Non-error comparisons are untouched.
	_ = 1 == 2
}
