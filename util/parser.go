package util

import (
	"fmt"
	"strconv"
	"time"
)

var RecordDateTimeFormat = "2006-01-02 15:04:05 MST"

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

func StringValueOrNil(inputValue interface{}) *string {
	switch value := inputValue.(type) {
	case string:
		if value == "" {
			return nil
		}

		return &value
	case *string:
		return value
	default:
		return nil
	}
}

func FormatTimeToString(inputTime time.Time, format string) string {
	if inputTime.IsZero() {
		return ""
	}

	return inputTime.Format(format)
}

func DateTimeToRecordFormat(inputTime time.Time) *string {
	return StringValueOrNil(FormatTimeToString(inputTime, RecordDateTimeFormat))
}
