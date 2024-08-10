package pubsub_test

import (
	"cmp"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
)

// less is a generic function that returns true if x is less than y.
func less[T cmp.Ordered](x, y T) bool {
	return x < y
}

// sortStringSlices is an cmp package option to sort slices of strings in ascending order.
var sortStringSlices = cmpopts.SortSlices(less[string])

// RetryUntil keeps calling the function f until it returns true or the deadline d has been reached.
func RetryUntil(d time.Duration, f func() bool) bool {
	deadline := time.Now().Add(d)

	for time.Now().Before(deadline) {
		if f() {
			return true
		}
	}

	return false
}
