package marketplace

import (
	"errors"
	"net/http"
	"testing"
)

// httpClientErrorRequest is a mock of the http.Client object which always returns an error when the ".Do" function
// gets called.
type httpClientErrorRequest struct{}

// httpClientInvalidStatusCodeResponse is a mock of the http.Client which always returns a response with a 400 status
// code.
type httpClientInvalidStatusCodeResponse struct{}

// Do returns an error simulating a non-reachability issue to the provided host.
func (h httpClientErrorRequest) Do(req *http.Request) (*http.Response, error) {
	return nil, errors.New("simulating not being able to reach the marketplace")

}

// Do returns an empty response with an 400 code.
func (h httpClientInvalidStatusCodeResponse) Do(req *http.Request) (*http.Response, error) {
	response := http.Response{StatusCode: 400}

	return &response, nil
}

// TestNotReachingMarketplace tests that an error is returned when an error occurs within the HTTP Client.
func TestNotReachingMarketplace(t *testing.T) {
	_, err := GetToken(&httpClientErrorRequest{}, "a", "a")

	if err == nil {
		t.Errorf("want error, got none")
	}

	want := "could not perform the request to the marketplace: simulating not being able to reach the marketplace"
	if err.Error() != want {
		t.Errorf("want %s, got %s", want, err)
	}
}

// TestInvalidStatusCodeReturnsError checks that an error is returned when a non 200 status code is returned on the
// response from the marketplace.
func TestInvalidStatusCodeReturnsError(t *testing.T) {
	_, err := GetToken(&httpClientInvalidStatusCodeResponse{}, "a", "a")

	if err == nil {
		t.Errorf("want error, got none")
	}

	want := "unexpected status code received from the marketplace: 400"
	if err.Error() != want {
		t.Errorf("want %s, got %s", want, err)
	}
}
