package util

import (
	"fmt"
	"strconv"
)

// InterfaceToInt64 takes an interface and returns an int64 for any float64, int64 or string received -whether they
// are pointers or not-.
func InterfaceToInt64(i interface{}) (int64, error) {
	switch value := i.(type) {
	case float64:
		return int64(value), nil
	case *float64:
		if value == nil {
			return 0, fmt.Errorf("cannot parse a nil pointer to a float64")
		}
		return int64(*value), nil
	case int64:
		return value, nil
	case *int64:
		if value == nil {
			return 0, fmt.Errorf("cannot parse a nil pointer to an int64")
		}

		return *value, nil
	case string:
		parsedValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, err
		}

		return parsedValue, nil
	case *string:
		if value == nil {
			return 0, fmt.Errorf("cannot parse a nil pointer to a string")
		}

		parsedValue, err := strconv.ParseInt(*value, 10, 64)
		if err != nil {
			return 0, err
		}

		return parsedValue, nil
	default:
		return 0, fmt.Errorf("invalid format provided")
	}
}
