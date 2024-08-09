package pubsub_test

import (
	"cmp"

	"github.com/google/go-cmp/cmp/cmpopts"
)

// less is a generic function that returns true if x is less than y.
func less[T cmp.Ordered](x, y T) bool {
	return x < y
}

// sortStringSlices is an cmp package option to sort slices of strings in ascending order.
var sortStringSlices = cmpopts.SortSlices(less[string])
