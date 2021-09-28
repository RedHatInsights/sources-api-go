package util

import (
	"testing"
)

var sortedSlice = []string{"a", "b", "c", "d", "e", "f"}
var unsortedSlice = []string{"f", "e", "d", "c", "b", "a"}
var valuesToTest = []struct {
	expectedResult   bool
	valueToSearchFor string
}{
	{true, "f"},
	{true, "c"},
	{true, "a"},
	{true, "b"},
	{true, "e"},
	{false, "hello"},
	{false, "z"},
	{false, "it is present, I swear"},
	{false, "I'll give up then :("},
}

// TestIsStringPresentInSlice tests that the IsStringPresentInSlice function returns the expected result when
// searching for a target string in a sorted slice.
func TestIsStringPresentInSlice(t *testing.T) {
	for _, tt := range valuesToTest {
		isPresent := IsStringPresentInSlice(tt.valueToSearchFor, sortedSlice)
		if tt.expectedResult != isPresent {
			t.Errorf(
				"got %t, want %t. Slice: %#v, target %#v",
				isPresent,
				tt.expectedResult,
				sortedSlice,
				tt.valueToSearchFor,
			)
		}
	}
}

// TestSortAndIsStringPresentInSlice tests that the SortAndIsStringPresentInSlice function returns the expected result
// when searching for a target string in an unsorted slice. The sorting algorithm only has to do the sorting work once,
//as the slice will be modified by it.
func TestSortAndIsStringPresentInSlice(t *testing.T) {
	for _, tt := range valuesToTest {
		isPresent := SortAndIsStringPresentInSlice(tt.valueToSearchFor, unsortedSlice)
		if tt.expectedResult != isPresent {
			t.Errorf(
				"got %t, want %t. Slice: %#v, target %#v",
				isPresent,
				tt.expectedResult,
				sortedSlice,
				tt.valueToSearchFor,
			)
		}
	}
}
