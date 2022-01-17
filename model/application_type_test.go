package model

import (
	"os"
	"testing"
)

func TestGoodUrl(t *testing.T) {
	expected := "http://a.good/uri"
	os.Setenv("TEST_NAME_AVAILABILITY_CHECK_URL", expected)

	a := ApplicationType{Name: "/this/is/my/test-name"}
	uri := a.AvailabilityCheckURL()

	if uri.String() != expected {
		t.Errorf("got the wrong availability check url, got %v expected %v", uri.String(), expected)
	}
}

func TestNotExistingUrl(t *testing.T) {
	a := ApplicationType{Name: "/this/one/does/not/exist"}
	uri := a.AvailabilityCheckURL()

	if uri != nil {
		t.Errorf("uri pulled from ENV even though it does not exist")
	}
}
