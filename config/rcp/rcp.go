package rcp

import "errors"

// Errors returned by the package.
var (
	ErrCannotFetchConfig     = errors.New("cannot fetch relay config")
	ErrHTTPRequestFailed     = errors.New("http request failed")
	ErrMalformedProviderURL  = errors.New("malformed relay config provider url")
	ErrMalformedResponseBody = errors.New("malformed response body")
)
