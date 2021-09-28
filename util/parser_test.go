package util

import (
	"regexp"
	"testing"
)

var pointerErrorRegex = regexp.MustCompile(`^cannot parse a nil pointer to an? (float64|int64|string)$`)

// TestValidConversionValues tests a set of values that are expected to be valid.
func TestValidConversionValues(t *testing.T) {
	floatPointerValue := 102.4512932
	int64PointerValue := int64(980)
	stringPointervalue := "712"

	validTestValues := []struct {
		expectedValue interface{}
		testedValue   interface{}
	}{
		{int64(45), 45.23},
		{int64(102), &floatPointerValue},
		{int64(451), int64(451)},
		{int64(980), &int64PointerValue},
		{int64(12), "12"},
		{int64(712), &stringPointervalue},
	}

	var i interface{}
	for _, tt := range validTestValues {
		i = tt.testedValue
		value, err := InterfaceToInt64(i)

		if err != nil {
			t.Errorf("Not expecting an error, got \"%s\"", err)
		}

		if value != tt.expectedValue {
			t.Errorf("got %v, want %v", tt.testedValue, tt.expectedValue)
		}
	}
}

// TestNilPointers tests that when passed nil pointers to the InterfaceToInt64 function, this returns an error.
func TestNilPointers(t *testing.T) {
	var nilFloatPointer *float64 = nil
	var nilInt64Pointer *int64 = nil
	var nilStringPointer *string = nil

	nilPointers := []struct {
		pointer interface{}
	}{
		{nilFloatPointer},
		{nilInt64Pointer},
		{nilStringPointer},
	}

	var i interface{}
	for _, tt := range nilPointers {
		i = tt.pointer
		value, err := InterfaceToInt64(i)

		if err == nil {
			t.Errorf("Expecting an error, got none")
		}

		if !pointerErrorRegex.MatchString(err.Error()) {
			t.Errorf("got \"%s\", want \"%s\"", err.Error(), pointerErrorRegex.String())
		}

		if value != 0 {
			t.Errorf("got %d, want 0", value)
		}
	}
}

// TestUnparseableString tests that when provided an unparseable string to an int64, the InterfaceToInt64 function
// returns an error.
func TestUnparseableString(t *testing.T) {
	notNumberString := "Hello, Go!"

	notValues := []struct {
		value interface{}
	}{
		{notNumberString},
		{&notNumberString},
	}

	var i interface{}
	for _, tt := range notValues {
		i = tt.value
		value, err := InterfaceToInt64(i)
		if err == nil {
			t.Errorf("Error expected, got none")
		}

		if value != 0 {
			t.Errorf("got %d, want 0", value)
		}
	}
}

// TestPassingInvalidFormats tests that the InterfaceToInt64 function returns an error when unexpected formats are
// passed to the function
func TestPassingInvalidFormats(t *testing.T) {
	var booleanPointer *bool = nil

	invalidFormats := []struct {
		value interface{}
	}{
		{&booleanPointer},
		{true},
		{false},
		{complex(12, 5)},
		{[]int64{25, 50, 75}},
		{uint(5)},
		{[]string{"a, b, c"}},
	}

	var i interface{}
	for _, tt := range invalidFormats {
		i = tt.value

		value, err := InterfaceToInt64(i)
		if err == nil {
			t.Errorf("Error expected, got none")
		}

		if err.Error() != "invalid format provided" {
			t.Errorf("got \"%s\", want \"%s\"", err.Error(), "invalid format provided")
		}

		if value != 0 {
			t.Errorf("got %d, want 0", value)
		}
	}
}
