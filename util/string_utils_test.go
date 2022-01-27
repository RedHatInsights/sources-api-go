package util

import "testing"

func TestCapitalize(t *testing.T) {
	str := "thing"
	capped := Capitalize(str)

	if capped != "Thing" {
		t.Errorf("Capitalize not working correctly - got %v instead of %v", capped, "Thing")
	}
}
