package marketplace

import "net/http"

// HttpClient abstracts away the client to be used in the GetToken function, and allows mocking it easily for the
// tests.
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type TokenCacher interface {
	// FetchToken fetches a marketplace token from the cache.
	FetchToken() (*BearerToken, error)
	// CacheToken sets a marketplace token on the cache.
	CacheToken(token *BearerToken) error
}

type TokenProvider interface {
	// RequestToken returns a bearer token using the given API Key.
	RequestToken() (*BearerToken, error)
}
