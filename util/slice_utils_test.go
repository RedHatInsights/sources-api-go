package util

import (
	"testing"
)

var slice = []string{"f", "a", "c", "d", "b", "e"}
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

// TestSliceContainsString tests that the SliceContainsString function returns the expected result when
// searching for a target string in a slice.
func TestSliceContainsString(t *testing.T) {
	for _, tt := range valuesToTest {
		isPresent := SliceContainsString(slice, tt.valueToSearchFor)
		if tt.expectedResult != isPresent {
			t.Errorf(
				"got %t, want %t. Slice: %#v, target %#v",
				isPresent,
				tt.expectedResult,
				slice,
				tt.valueToSearchFor,
			)
		}
	}
}
