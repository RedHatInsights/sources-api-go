package util

import (
	"sort"

	"github.com/google/go-cmp/cmp"
)

// SliceContainsString returns true if the specified target is present in the given slice.
func SliceContainsString(slice []string, target string) bool {
	for _, element := range slice {
		if element == target {
			return true
		}
	}
	return false
}

// ElementsInSlicesEqual sorts and compare slices of int64 and returns that slices are equal
func ElementsInSlicesEqual(sliceA []int64, sliceB []int64) bool {
	sort.Slice(sliceA, func(i, j int) bool { return sliceA[i] < sliceA[j] })
	sort.Slice(sliceB, func(i, j int) bool { return sliceB[i] < sliceB[j] })

	return cmp.Equal(sliceA, sliceB)
}
