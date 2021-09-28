package util

import "sort"

// IsStringPresentInSlice returns true if the specified target is present in the given slice. Requires the slice to
// be ordered in ascending order, due to the underlying requirement with SearchStrings.
func IsStringPresentInSlice(target string, slice []string) bool {
	index := sort.SearchStrings(slice, target)

	return index < len(slice) && slice[index] == target
}

// SortAndIsStringPresentInSlice sorts the slice before trying to find the target in the given slice.
func SortAndIsStringPresentInSlice(target string, slice []string) bool {
	sort.Strings(slice)
	return IsStringPresentInSlice(target, slice)
}
